package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/target"
)

const statusUnsupported = "unsupported"

// Plan reads an edit spec and returns the non-mutating edit plan.
func Plan(ctx context.Context, deckPath string, specPath string) (*model.CommandResult, error) {
	spec, err := readSpec(specPath)
	if err != nil {
		return nil, err
	}
	return planSpec(ctx, deckPath, spec)
}

func buildPlanResult(deckPath string, spec model.EditSpec, inspectionResult *model.CommandResult) *model.CommandResult {
	matches, status, message := target.Resolve(inspectionResult.Inspection, spec.Target)
	if status == target.StatusReady {
		if unsupportedMessage := validateOperationMatches(spec.Operation, matches); unsupportedMessage != "" {
			status = statusUnsupported
			message = unsupportedMessage
		}
	}
	plan := &model.EditPlan{
		Operation:              spec.Operation,
		Target:                 spec.Target,
		Matches:                matches,
		Status:                 status,
		Message:                message,
		Replacement:            spec.Replacement,
		ImagePath:              spec.ImagePath,
		InsertAfterSlide:       spec.InsertAfterSlide,
		DestinationSlideNumber: spec.DestinationSlideNumber,
	}

	resultStatus := "ok"
	if status != target.StatusReady {
		resultStatus = status
	}

	result := &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "plan",
		Status:        resultStatus,
		Input:         deckPath,
		Output:        nil,
		Warnings:      inspectionResult.Warnings,
		Errors:        []model.ErrorItem{},
		Summary: model.Summary{
			Human: humanSummary(plan),
		},
		Plan: plan,
	}

	switch status {
	case target.StatusNoMatch:
		result.Skipped = []model.SkipItem{{
			Code:    "no_match",
			Message: message,
		}}
	case target.StatusAmbiguous:
		result.Ambiguous = []model.SkipItem{{
			Code:    "ambiguous_target",
			Message: message,
		}}
	case statusUnsupported:
		result.Unsupported = []model.SkipItem{{
			Code:    "unsupported_operation_target",
			Message: message,
		}}
	}
	return result
}

func validateOperationTarget(spec model.EditSpec) string {
	allowedTargets := map[string]map[string]bool{
		"replace_text": {
			"title":        true,
			"visible_text": true,
			"object_id":    true,
		},
		"update_notes": {
			"notes": true,
		},
		"update_metadata": {
			"metadata": true,
		},
		"replace_image": {
			"object_id": true,
			"image":     true,
		},
		"update_description": {
			"slide_number": true,
		},
		"add_text_box": {
			"slide_number": true,
		},
		"add_shape": {
			"slide_number": true,
		},
		"slide_add": {
			"slide_number": true,
		},
		"slide_delete": {
			"slide_number": true,
		},
		"slide_move": {
			"slide_number": true,
		},
		"slide_duplicate": {
			"slide_number": true,
		},
	}

	targets, ok := allowedTargets[spec.Operation]
	if !ok {
		return fmt.Sprintf("unsupported operation %q", spec.Operation)
	}
	if !targets[spec.Target.Type] {
		return fmt.Sprintf("operation %q does not support target type %q", spec.Operation, spec.Target.Type)
	}
	if spec.Operation == "replace_text" && spec.Replacement == "" {
		return "replace_text requires replacement"
	}
	if spec.Operation == "update_notes" && spec.Replacement == "" {
		return "update_notes requires replacement"
	}
	if spec.Operation == "update_metadata" && spec.Target.Property == "" {
		return "update_metadata requires target property"
	}
	if spec.Operation == "update_metadata" && !isSupportedMetadataProperty(spec.Target.Property) {
		return fmt.Sprintf("unsupported metadata property %q", spec.Target.Property)
	}
	if spec.Operation == "replace_image" && spec.ImagePath == "" {
		return "replace_image requires image_path"
	}
	if spec.Operation == "update_description" && spec.Replacement == "" {
		return "update_description requires replacement"
	}
	if spec.Operation == "slide_move" && spec.DestinationSlideNumber == 0 {
		return "slide_move requires destination_slide_number"
	}
	if spec.Operation == "slide_duplicate" && spec.InsertAfterSlide == 0 {
		return "slide_duplicate requires insert_after_slide"
	}
	if spec.Operation == "slide_add" && spec.Replacement == "" {
		return "slide_add requires replacement"
	}
	if spec.Operation == "add_text_box" && spec.Replacement == "" {
		return "add_text_box requires replacement"
	}
	if spec.Operation == "add_shape" && spec.Replacement == "" {
		return "add_shape requires replacement"
	}
	return ""
}

func isSupportedMetadataProperty(property string) bool {
	switch property {
	case "title", "author", "subject", "keywords", "last_modified_by":
		return true
	default:
		return false
	}
}

func validateOperationMatches(operation string, matches []model.TargetMatch) string {
	for _, match := range matches {
		if !operationSupportsMatchKind(operation, match.Kind) {
			return fmt.Sprintf("operation %q does not support resolved target kind %q", operation, match.Kind)
		}
	}
	return ""
}

func operationSupportsMatchKind(operation string, kind string) bool {
	switch operation {
	case "replace_text":
		return kind == "visible_text" || kind == "title"
	case "update_notes":
		return kind == "notes"
	case "update_metadata":
		return kind == "metadata"
	case "replace_image":
		return kind == "image"
	case "add_text_box", "add_shape", "update_description":
		return kind == "slide"
	case "slide_add", "slide_delete", "slide_move", "slide_duplicate":
		return kind == "slide"
	default:
		return false
	}
}

func readSpec(specPath string) (model.EditSpec, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return model.EditSpec{}, err
	}

	var spec model.EditSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return model.EditSpec{}, err
	}
	if spec.Operation == "" {
		return model.EditSpec{}, fmt.Errorf("edit spec missing operation")
	}
	if spec.Target.Type == "" {
		return model.EditSpec{}, fmt.Errorf("edit spec missing target type")
	}
	return spec, nil
}

func humanSummary(plan *model.EditPlan) string {
	switch plan.Status {
	case target.StatusReady:
		return fmt.Sprintf("Planned %s for %d target(s).", plan.Operation, len(plan.Matches))
	case target.StatusAmbiguous:
		return "Edit target is ambiguous."
	case target.StatusNoMatch:
		return "No matching edit target found."
	case statusUnsupported:
		return "Edit operation is unsupported for the requested target."
	default:
		return plan.Message
	}
}
