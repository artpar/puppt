package edit

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"

	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	"github.com/artpar/puppt/internal/validate"
)

// Apply plans, applies, writes, and validates a supported edit.
func Apply(ctx context.Context, deckPath string, specPath string, outputPath string) (*model.CommandResult, error) {
	spec, err := readSpec(specPath)
	if err != nil {
		return nil, err
	}

	planResult, err := planSpec(ctx, deckPath, spec)
	if err != nil {
		return nil, err
	}
	planResult.Command = "edit"
	if planResult.Status != "ok" {
		return planResult, nil
	}
	if message := validateMutationSupport(spec.Operation); message != "" {
		planResult.Status = statusUnsupported
		planResult.Summary.Human = "Edit operation is unsupported for mutation."
		planResult.Plan.Status = statusUnsupported
		planResult.Plan.Message = message
		planResult.Unsupported = []model.SkipItem{{
			Code:    "unsupported_mutation",
			Message: message,
		}}
		return planResult, nil
	}
	planResult.Output = &outputPath

	pkg, err := pptx.Open(ctx, deckPath)
	if err != nil {
		return nil, err
	}

	changes, err := applyMutation(pkg, spec, planResult.Plan.Matches)
	if err != nil {
		return nil, err
	}
	if err := pptx.Write(ctx, pkg, outputPath); err != nil {
		return nil, err
	}

	validationResult, err := validate.Validate(ctx, outputPath)
	if err != nil {
		return nil, err
	}
	validation := validationResult.Validation
	if validation == nil {
		validation = &model.Validation{Valid: true, Warnings: []model.Warning{}, Errors: []model.ErrorItem{}}
	}
	if err := verifyApplied(ctx, outputPath, spec, planResult.Plan.Matches); err != nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, model.ErrorItem{
			Code:    "edit_validation_failed",
			Message: err.Error(),
		})
	}

	status := "ok"
	summary := fmt.Sprintf("Applied %s with %d change(s).", spec.Operation, len(changes))
	if !validation.Valid {
		status = "invalid"
		summary = "Edit wrote output but validation failed."
	}

	return &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "edit",
		Status:        status,
		Input:         deckPath,
		Output:        &outputPath,
		Warnings:      append(planResult.Warnings, validation.Warnings...),
		Errors:        validation.Errors,
		Summary:       model.Summary{Human: summary},
		Plan:          planResult.Plan,
		Changes:       changes,
		Validation:    validation,
	}, nil
}

func planSpec(ctx context.Context, deckPath string, spec model.EditSpec) (*model.CommandResult, error) {
	if message := validateOperationTarget(spec); message != "" {
		plan := &model.EditPlan{
			Operation:              spec.Operation,
			Target:                 spec.Target,
			Matches:                []model.TargetMatch{},
			Status:                 statusUnsupported,
			Message:                message,
			Replacement:            spec.Replacement,
			ImagePath:              spec.ImagePath,
			InsertAfterSlide:       spec.InsertAfterSlide,
			DestinationSlideNumber: spec.DestinationSlideNumber,
		}
		return &model.CommandResult{
			SchemaVersion: model.SchemaVersion,
			Command:       "plan",
			Status:        statusUnsupported,
			Input:         deckPath,
			Output:        nil,
			Warnings:      []model.Warning{},
			Errors:        []model.ErrorItem{},
			Summary:       model.Summary{Human: "Edit operation is unsupported for the requested target."},
			Plan:          plan,
			Unsupported: []model.SkipItem{{
				Code:    "unsupported_operation_target",
				Message: message,
			}},
		}, nil
	}

	inspectionResult, err := inspect.Inspect(ctx, deckPath)
	if err != nil {
		return nil, err
	}
	return buildPlanResult(deckPath, spec, inspectionResult), nil
}

func validateMutationSupport(operation string) string {
	switch operation {
	case "replace_text", "update_notes", "update_metadata", "replace_image", "add_text_box", "add_shape", "slide_add", "slide_delete", "slide_move", "slide_duplicate":
		return ""
	default:
		return fmt.Sprintf("operation %q is planned but not implemented for mutation yet", operation)
	}
}

func applyMutation(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	switch spec.Operation {
	case "replace_text":
		return applyTextReplacement(pkg, spec, matches)
	case "update_notes":
		return applyNotesUpdate(pkg, spec, matches)
	case "update_metadata":
		return applyMetadataUpdate(pkg, spec)
	case "replace_image":
		return applyImageReplacement(pkg, spec, matches)
	case "add_text_box", "add_shape":
		return applySimpleAddition(pkg, spec, matches)
	case "slide_add", "slide_delete", "slide_move", "slide_duplicate":
		return applySlideOperation(pkg, spec, matches)
	default:
		return nil, fmt.Errorf("unsupported mutation operation %q", spec.Operation)
	}
}

func applyTextReplacement(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	var changes []model.ChangeItem
	for _, match := range matches {
		partName := match.SlideID
		data, ok := pkg.Parts[partName]
		if !ok {
			return nil, fmt.Errorf("matched slide part missing: %s", partName)
		}
		shapeID := shapeIDFromObjectID(match.ObjectID)
		if shapeID == "" {
			return nil, fmt.Errorf("matched text target lacks shape object id: %s", match.ObjectID)
		}

		wholeObject := spec.Target.Type == "object_id" || spec.Target.Type == "title"
		updated, count, err := replaceShapeText(data, shapeID, spec.Target.Text, spec.Replacement, wholeObject)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("no text runs changed for object %s", match.ObjectID)
		}
		pkg.Parts[partName] = updated
		changes = append(changes, model.ChangeItem{
			SlideNumber: match.SlideNumber,
			ObjectID:    match.ObjectID,
			Message:     fmt.Sprintf("Replaced %d text match(es).", count),
		})
	}
	return changes, nil
}

func applyNotesUpdate(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	var changes []model.ChangeItem
	for _, match := range matches {
		notesPart, err := notesPartForSlide(pkg, match.SlideID)
		if err != nil {
			return nil, err
		}
		data, ok := pkg.Parts[notesPart]
		if !ok {
			return nil, fmt.Errorf("notes part missing: %s", notesPart)
		}
		updated, count, err := replaceFirstTextObject(data, spec.Replacement)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, fmt.Errorf("notes part has no editable text: %s", notesPart)
		}
		pkg.Parts[notesPart] = updated
		changes = append(changes, model.ChangeItem{
			SlideNumber: match.SlideNumber,
			Message:     "Updated speaker notes.",
		})
	}
	return changes, nil
}

func applyMetadataUpdate(pkg *pptx.Package, spec model.EditSpec) ([]model.ChangeItem, error) {
	corePart := ""
	for _, relationship := range pkg.RootRelationships {
		if relationship.Type == pptx.CorePropertiesRelType {
			corePart = pptx.ResolveTargetPart("", relationship.Target)
			break
		}
	}
	if corePart == "" {
		return nil, fmt.Errorf("core metadata part is missing")
	}
	data, ok := pkg.Parts[corePart]
	if !ok {
		return nil, fmt.Errorf("core metadata part is missing: %s", corePart)
	}
	updated, count, err := replaceCoreProperty(data, spec.Target.Property, spec.Replacement)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("metadata property %q not found", spec.Target.Property)
	}
	pkg.Parts[corePart] = updated
	return []model.ChangeItem{{
		Message: fmt.Sprintf("Updated metadata property %s.", spec.Target.Property),
	}}, nil
}

func notesPartForSlide(pkg *pptx.Package, slidePart string) (string, error) {
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return "", err
	}
	for _, relationship := range relationships {
		if relationship.Type == pptx.NotesSlideRelType {
			return pptx.ResolveTargetPart(slidePart, relationship.Target), nil
		}
	}
	return "", fmt.Errorf("slide has no notes relationship: %s", slidePart)
}

type shapeState struct {
	target bool
}

func replaceShapeText(data []byte, shapeID string, oldText string, replacement string, wholeObject bool) ([]byte, int, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	var shapes []shapeState
	count := 0
	wroteWholeObject := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			if item.Name.Local == "sp" {
				shapes = append(shapes, shapeState{})
			}
			if item.Name.Local == "cNvPr" && len(shapes) > 0 && attrValue(item.Attr, "id") == shapeID {
				shapes[len(shapes)-1].target = true
			}
			if item.Name.Local == "t" && currentShapeTarget(shapes) {
				var value string
				if err := decoder.DecodeElement(&value, &item); err != nil {
					return nil, 0, err
				}
				nextValue := value
				switch {
				case wholeObject && !wroteWholeObject:
					nextValue = replacement
					wroteWholeObject = true
					count++
				case wholeObject:
					nextValue = ""
				default:
					matches := strings.Count(value, oldText)
					if matches > 0 {
						nextValue = strings.ReplaceAll(value, oldText, replacement)
						count += matches
					}
				}
				if err := encodeTextElement(encoder, item, nextValue); err != nil {
					return nil, 0, err
				}
				continue
			}
			if err := encoder.EncodeToken(item); err != nil {
				return nil, 0, err
			}
		case xml.EndElement:
			if err := encoder.EncodeToken(item); err != nil {
				return nil, 0, err
			}
			if item.Name.Local == "sp" && len(shapes) > 0 {
				shapes = shapes[:len(shapes)-1]
			}
		default:
			if err := encoder.EncodeToken(token); err != nil {
				return nil, 0, err
			}
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, 0, err
	}
	return output.Bytes(), count, nil
}

func replaceFirstTextObject(data []byte, replacement string) ([]byte, int, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	count := 0
	replaced := false

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, err
		}
		start, ok := token.(xml.StartElement)
		if ok && start.Name.Local == "t" {
			var value string
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return nil, 0, err
			}
			nextValue := ""
			if !replaced {
				nextValue = replacement
				replaced = true
				count++
			}
			if err := encodeTextElement(encoder, start, nextValue); err != nil {
				return nil, 0, err
			}
			continue
		}
		if err := encoder.EncodeToken(token); err != nil {
			return nil, 0, err
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, 0, err
	}
	return output.Bytes(), count, nil
}

func replaceCoreProperty(data []byte, property string, replacement string) ([]byte, int, error) {
	elementName := corePropertyElement(property)
	if elementName == "" {
		return nil, 0, fmt.Errorf("unsupported metadata property %q", property)
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	count := 0

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, err
		}
		start, ok := token.(xml.StartElement)
		if ok && start.Name.Local == elementName {
			var oldValue string
			if err := decoder.DecodeElement(&oldValue, &start); err != nil {
				return nil, 0, err
			}
			if err := encodeTextElement(encoder, start, replacement); err != nil {
				return nil, 0, err
			}
			count++
			continue
		}
		if err := encoder.EncodeToken(token); err != nil {
			return nil, 0, err
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, 0, err
	}
	return output.Bytes(), count, nil
}

func verifyApplied(ctx context.Context, outputPath string, spec model.EditSpec, matches []model.TargetMatch) error {
	result, err := inspect.Inspect(ctx, outputPath)
	if err != nil {
		return err
	}
	inspection := result.Inspection
	if inspection == nil {
		return fmt.Errorf("inspection result missing")
	}

	switch spec.Operation {
	case "replace_text":
		for _, match := range matches {
			if !textObjectContains(inspection, match.ObjectID, spec.Replacement) {
				return fmt.Errorf("replacement text not found for %s", match.ObjectID)
			}
		}
	case "update_notes":
		for _, match := range matches {
			if !slideNotesContain(inspection, match.SlideNumber, spec.Replacement) {
				return fmt.Errorf("replacement notes not found on slide %d", match.SlideNumber)
			}
		}
	case "update_metadata":
		if metadataValue(inspection.Metadata, spec.Target.Property) != spec.Replacement {
			return fmt.Errorf("metadata property %q was not updated", spec.Target.Property)
		}
	case "replace_image":
		if err := verifyImageReplacement(ctx, outputPath, spec, matches); err != nil {
			return err
		}
	case "add_text_box", "add_shape":
		if !slideNotesOrTextContain(inspection, matches[0].SlideNumber, spec.Replacement) {
			return fmt.Errorf("added text not found on slide %d", matches[0].SlideNumber)
		}
	case "slide_add":
		if !slideTextAt(inspection, spec.Target.SlideNumber+1, spec.Replacement) {
			return fmt.Errorf("added slide text not found at position %d", spec.Target.SlideNumber+1)
		}
	case "slide_delete":
		if slideIDExists(inspection, matches[0].SlideID) {
			return fmt.Errorf("deleted slide still exists: %s", matches[0].SlideID)
		}
	case "slide_move":
		if !slideIDAt(inspection, spec.DestinationSlideNumber, matches[0].SlideID) {
			return fmt.Errorf("slide %s not found at position %d", matches[0].SlideID, spec.DestinationSlideNumber)
		}
	case "slide_duplicate":
		expectedPosition := spec.InsertAfterSlide + 1
		if !slideTextAt(inspection, expectedPosition, matches[0].Text) {
			return fmt.Errorf("duplicated slide text not found at position %d", expectedPosition)
		}
	}
	return nil
}

func textObjectContains(inspection *model.Inspection, objectID string, text string) bool {
	for _, slide := range inspection.Slides {
		for _, block := range slide.VisibleText {
			if block.ObjectID == objectID && strings.Contains(block.Text, text) {
				return true
			}
		}
	}
	return false
}

func slideNotesContain(inspection *model.Inspection, slideNumber int, text string) bool {
	for _, slide := range inspection.Slides {
		if slide.Number != slideNumber {
			continue
		}
		for _, block := range slide.Notes {
			if strings.Contains(block.Text, text) {
				return true
			}
		}
	}
	return false
}

func slideNotesOrTextContain(inspection *model.Inspection, slideNumber int, text string) bool {
	for _, slide := range inspection.Slides {
		if slide.Number != slideNumber {
			continue
		}
		for _, block := range slide.VisibleText {
			if strings.Contains(block.Text, text) {
				return true
			}
		}
		for _, block := range slide.Notes {
			if strings.Contains(block.Text, text) {
				return true
			}
		}
	}
	return false
}

func metadataValue(metadata model.Metadata, property string) string {
	switch property {
	case "title":
		return metadata.Title
	case "author":
		return metadata.Author
	case "subject":
		return metadata.Subject
	default:
		return ""
	}
}

func slideIDExists(inspection *model.Inspection, slideID string) bool {
	for _, slide := range inspection.Slides {
		if slide.ID == slideID {
			return true
		}
	}
	return false
}

func slideIDAt(inspection *model.Inspection, slideNumber int, slideID string) bool {
	for _, slide := range inspection.Slides {
		if slide.Number == slideNumber {
			return slide.ID == slideID
		}
	}
	return false
}

func slideTextAt(inspection *model.Inspection, slideNumber int, text string) bool {
	for _, slide := range inspection.Slides {
		if slide.Number != slideNumber {
			continue
		}
		for _, block := range slide.VisibleText {
			if strings.Contains(block.Text, text) {
				return true
			}
		}
	}
	return false
}

func encodeTextElement(encoder *xml.Encoder, start xml.StartElement, value string) error {
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	if err := encoder.EncodeToken(xml.CharData([]byte(value))); err != nil {
		return err
	}
	return encoder.EncodeToken(xml.EndElement{Name: start.Name})
}

func currentShapeTarget(shapes []shapeState) bool {
	return len(shapes) > 0 && shapes[len(shapes)-1].target
}

func attrValue(attrs []xml.Attr, localName string) string {
	for _, attr := range attrs {
		if attr.Name.Local == localName {
			return attr.Value
		}
	}
	return ""
}

func shapeIDFromObjectID(objectID string) string {
	_, suffix, ok := strings.Cut(objectID, "#shape-")
	if !ok {
		return ""
	}
	return suffix
}

func corePropertyElement(property string) string {
	switch property {
	case "title":
		return "title"
	case "author":
		return "creator"
	case "subject":
		return "subject"
	default:
		return ""
	}
}
