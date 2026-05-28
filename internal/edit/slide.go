package edit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

const (
	presentationMLNamespace = "http://schemas.openxmlformats.org/presentationml/2006/main"
	relationshipsNamespace  = "http://schemas.openxmlformats.org/package/2006/relationships"
	officeRelationshipNS    = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	slideRelationshipType   = officeRelationshipNS + "/slide"
	slideContentType        = "application/vnd.openxmlformats-officedocument.presentationml.slide+xml"
	contentTypesPartName    = "[Content_Types].xml"
)

type slideRef struct {
	ID             int
	RelationshipID string
}

func applySlideOperation(pkg *pptx.Package, spec model.EditSpec, matches []model.TargetMatch) ([]model.ChangeItem, error) {
	if len(matches) != 1 {
		return nil, fmt.Errorf("slide operation requires one resolved slide target")
	}
	switch spec.Operation {
	case "slide_add":
		return applySlideAdd(pkg, spec, matches[0])
	case "slide_delete":
		return applySlideDelete(pkg, matches[0])
	case "slide_move":
		return applySlideMove(pkg, spec, matches[0])
	case "slide_duplicate":
		return applySlideDuplicate(pkg, spec, matches[0])
	default:
		return nil, fmt.Errorf("unsupported slide operation %q", spec.Operation)
	}
}

func applySlideAdd(pkg *pptx.Package, spec model.EditSpec, match model.TargetMatch) ([]model.ChangeItem, error) {
	state, err := loadSlideState(pkg)
	if err != nil {
		return nil, err
	}
	insertIndex := clampInsertIndex(spec.Target.SlideNumber, len(state.refs))
	newSlidePart := nextSlidePart(pkg)
	newRelationshipID := nextRelationshipID(state.relationships)
	newSlideID := nextSlideID(state.refs)

	pkg.Parts[newSlidePart] = []byte(simpleSlideXML(spec.Replacement))
	state.relationships = append(state.relationships, pptx.Relationship{
		ID:     newRelationshipID,
		Type:   slideRelationshipType,
		Target: presentationRelationshipTarget(pkg.PresentationPath, newSlidePart),
	})
	state.refs = insertSlideRef(state.refs, insertIndex, slideRef{ID: newSlideID, RelationshipID: newRelationshipID})
	pkg.SlideParts = insertString(pkg.SlideParts, insertIndex, newSlidePart)
	if err := writeSlideState(pkg, state); err != nil {
		return nil, err
	}
	if err := ensureSlideContentType(pkg, newSlidePart); err != nil {
		return nil, err
	}
	return []model.ChangeItem{{
		SlideNumber: insertIndex + 1,
		ObjectID:    newSlidePart,
		Message:     fmt.Sprintf("Added slide after position %d using new part %s.", match.SlideNumber, newSlidePart),
	}}, nil
}

func applySlideDelete(pkg *pptx.Package, match model.TargetMatch) ([]model.ChangeItem, error) {
	state, err := loadSlideState(pkg)
	if err != nil {
		return nil, err
	}
	index, err := slideIndexByRelationship(pkg.PresentationPath, state.refs, state.relationships, match.SlideID)
	if err != nil {
		return nil, err
	}
	relationshipID := state.refs[index].RelationshipID
	delete(pkg.Parts, match.SlideID)
	delete(pkg.Parts, pptx.RelationshipsPartFor(match.SlideID))
	state.refs = append(state.refs[:index], state.refs[index+1:]...)
	state.relationships = removeRelationship(state.relationships, relationshipID)
	pkg.SlideParts = removeString(pkg.SlideParts, match.SlideID)
	if err := removeSlideContentType(pkg, match.SlideID); err != nil {
		return nil, err
	}
	if err := writeSlideState(pkg, state); err != nil {
		return nil, err
	}
	return []model.ChangeItem{{
		SlideNumber: match.SlideNumber,
		ObjectID:    match.SlideID,
		Message:     fmt.Sprintf("Deleted slide position %d with part %s.", match.SlideNumber, match.SlideID),
	}}, nil
}

func applySlideMove(pkg *pptx.Package, spec model.EditSpec, match model.TargetMatch) ([]model.ChangeItem, error) {
	state, err := loadSlideState(pkg)
	if err != nil {
		return nil, err
	}
	from := match.SlideNumber - 1
	to := spec.DestinationSlideNumber - 1
	if from < 0 || from >= len(state.refs) {
		return nil, fmt.Errorf("source slide position out of range: %d", match.SlideNumber)
	}
	if to < 0 || to >= len(state.refs) {
		return nil, fmt.Errorf("destination slide position out of range: %d", spec.DestinationSlideNumber)
	}
	state.refs = moveSlideRef(state.refs, from, to)
	pkg.SlideParts = moveString(pkg.SlideParts, from, to)
	if err := writeSlideState(pkg, state); err != nil {
		return nil, err
	}
	return []model.ChangeItem{{
		SlideNumber: spec.DestinationSlideNumber,
		ObjectID:    match.SlideID,
		Message:     fmt.Sprintf("Moved slide %s from position %d to position %d.", match.SlideID, match.SlideNumber, spec.DestinationSlideNumber),
	}}, nil
}

func applySlideDuplicate(pkg *pptx.Package, spec model.EditSpec, match model.TargetMatch) ([]model.ChangeItem, error) {
	state, err := loadSlideState(pkg)
	if err != nil {
		return nil, err
	}
	sourceIndex, err := slideIndexByRelationship(pkg.PresentationPath, state.refs, state.relationships, match.SlideID)
	if err != nil {
		return nil, err
	}
	insertIndex := clampInsertIndex(spec.InsertAfterSlide, len(state.refs))
	newSlidePart := nextSlidePart(pkg)
	newRelationshipID := nextRelationshipID(state.relationships)
	newSlideID := nextSlideID(state.refs)

	pkg.Parts[newSlidePart] = append([]byte(nil), pkg.Parts[match.SlideID]...)
	sourceRels := pptx.RelationshipsPartFor(match.SlideID)
	if relsData, ok := pkg.Parts[sourceRels]; ok {
		pkg.Parts[pptx.RelationshipsPartFor(newSlidePart)] = append([]byte(nil), relsData...)
	}
	state.relationships = append(state.relationships, pptx.Relationship{
		ID:     newRelationshipID,
		Type:   slideRelationshipType,
		Target: presentationRelationshipTarget(pkg.PresentationPath, newSlidePart),
	})
	state.refs = insertSlideRef(state.refs, insertIndex, slideRef{ID: newSlideID, RelationshipID: newRelationshipID})
	pkg.SlideParts = insertString(pkg.SlideParts, insertIndex, newSlidePart)
	if err := writeSlideState(pkg, state); err != nil {
		return nil, err
	}
	if err := ensureSlideContentType(pkg, newSlidePart); err != nil {
		return nil, err
	}
	return []model.ChangeItem{{
		SlideNumber: insertIndex + 1,
		ObjectID:    newSlidePart,
		Message:     fmt.Sprintf("Duplicated slide %s from position %d to position %d as %s.", match.SlideID, sourceIndex+1, insertIndex+1, newSlidePart),
	}}, nil
}

type slideState struct {
	refs          []slideRef
	relationships []pptx.Relationship
}

func loadSlideState(pkg *pptx.Package) (slideState, error) {
	refs, err := parseSlideRefs(pkg.Parts[pkg.PresentationPath])
	if err != nil {
		return slideState{}, err
	}
	relationships, err := pkg.RelationshipsForPart(pkg.PresentationPath)
	if err != nil {
		return slideState{}, err
	}
	return slideState{refs: refs, relationships: relationships}, nil
}

func writeSlideState(pkg *pptx.Package, state slideState) error {
	presentation, err := writePresentationSlideRefs(pkg.Parts[pkg.PresentationPath], state.refs)
	if err != nil {
		return err
	}
	pkg.Parts[pkg.PresentationPath] = presentation
	pkg.Parts[pptx.RelationshipsPartFor(pkg.PresentationPath)] = writeRelationships(state.relationships)
	pkg.PresentationRelationships = state.relationships
	return nil
}

func parseSlideRefs(data []byte) ([]slideRef, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var refs []slideRef
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return refs, nil
			}
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sldId" {
			continue
		}
		ref := slideRef{}
		for _, attr := range start.Attr {
			switch {
			case attr.Name.Local == "id" && attr.Name.Space == "":
				id, err := strconv.Atoi(attr.Value)
				if err != nil {
					return nil, fmt.Errorf("invalid slide id %q: %w", attr.Value, err)
				}
				ref.ID = id
			case attr.Name.Local == "id":
				ref.RelationshipID = attr.Value
			}
		}
		if ref.ID == 0 || ref.RelationshipID == "" {
			return nil, fmt.Errorf("slide id entry is incomplete")
		}
		refs = append(refs, ref)
	}
}

func writePresentationSlideRefs(data []byte, refs []slideRef) ([]byte, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sldIdLst" {
			if err := encoder.EncodeToken(token); err != nil {
				return nil, err
			}
			continue
		}
		if err := encoder.EncodeToken(start); err != nil {
			return nil, err
		}
		for _, ref := range refs {
			if err := encodeSlideRef(encoder, ref); err != nil {
				return nil, err
			}
		}
		if err := discardUntilEnd(decoder, start.Name); err != nil {
			return nil, err
		}
		if err := encoder.EncodeToken(xml.EndElement{Name: start.Name}); err != nil {
			return nil, err
		}
	}
	if err := encoder.Flush(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func encodeSlideRef(encoder *xml.Encoder, ref slideRef) error {
	start := xml.StartElement{
		Name: xml.Name{Space: presentationMLNamespace, Local: "sldId"},
		Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: strconv.Itoa(ref.ID)},
			{Name: xml.Name{Space: officeRelationshipNS, Local: "id"}, Value: ref.RelationshipID},
		},
	}
	if err := encoder.EncodeToken(start); err != nil {
		return err
	}
	return encoder.EncodeToken(xml.EndElement{Name: start.Name})
}

func discardUntilEnd(decoder *xml.Decoder, name xml.Name) error {
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch item := token.(type) {
		case xml.StartElement:
			if item.Name == name {
				depth++
			}
		case xml.EndElement:
			if item.Name == name {
				depth--
			}
		}
	}
	return nil
}

func writeRelationships(relationships []pptx.Relationship) []byte {
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	relationshipsStart := xml.StartElement{Name: xml.Name{Space: relationshipsNamespace, Local: "Relationships"}}
	_ = encoder.EncodeToken(relationshipsStart)
	for _, relationship := range relationships {
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "Id"}, Value: relationship.ID},
			{Name: xml.Name{Local: "Type"}, Value: relationship.Type},
			{Name: xml.Name{Local: "Target"}, Value: relationship.Target},
		}
		if relationship.TargetMode != "" {
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "TargetMode"}, Value: relationship.TargetMode})
		}
		start := xml.StartElement{Name: xml.Name{Space: relationshipsNamespace, Local: "Relationship"}, Attr: attrs}
		_ = encoder.EncodeToken(start)
		_ = encoder.EncodeToken(xml.EndElement{Name: start.Name})
	}
	_ = encoder.EncodeToken(xml.EndElement{Name: relationshipsStart.Name})
	_ = encoder.Flush()
	return output.Bytes()
}

func nextSlidePart(pkg *pptx.Package) string {
	used := map[int]bool{}
	re := regexp.MustCompile(`^ppt/slides/slide([0-9]+)\.xml$`)
	for partName := range pkg.Parts {
		matches := re.FindStringSubmatch(partName)
		if len(matches) != 2 {
			continue
		}
		number, err := strconv.Atoi(matches[1])
		if err == nil {
			used[number] = true
		}
	}
	for number := 1; ; number++ {
		if !used[number] {
			return fmt.Sprintf("ppt/slides/slide%d.xml", number)
		}
	}
}

func nextRelationshipID(relationships []pptx.Relationship) string {
	maxID := 0
	for _, relationship := range relationships {
		if !strings.HasPrefix(relationship.ID, "rId") {
			continue
		}
		number, err := strconv.Atoi(strings.TrimPrefix(relationship.ID, "rId"))
		if err == nil && number > maxID {
			maxID = number
		}
	}
	return fmt.Sprintf("rId%d", maxID+1)
}

func nextSlideID(refs []slideRef) int {
	maxID := 255
	for _, ref := range refs {
		if ref.ID > maxID {
			maxID = ref.ID
		}
	}
	return maxID + 1
}

func slideIndexByRelationship(presentationPart string, refs []slideRef, relationships []pptx.Relationship, slidePart string) (int, error) {
	byID := map[string]pptx.Relationship{}
	for _, relationship := range relationships {
		byID[relationship.ID] = relationship
	}
	for index, ref := range refs {
		relationship, ok := byID[ref.RelationshipID]
		if !ok || relationship.Type != slideRelationshipType {
			continue
		}
		if pptx.ResolveTargetPart(presentationPart, relationship.Target) == slidePart {
			return index, nil
		}
	}
	return -1, fmt.Errorf("slide relationship for %s not found", slidePart)
}

func presentationRelationshipTarget(presentationPart string, slidePart string) string {
	baseDir := path.Dir(presentationPart)
	if strings.HasPrefix(slidePart, baseDir+"/") {
		return strings.TrimPrefix(slidePart, baseDir+"/")
	}
	return path.Clean(slidePart)
}

func simpleSlideXML(text string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Title 1"/>
        </p:nvSpPr>
        <p:txBody>
          <a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:r><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>
`, escapeXMLText(text))
}

func escapeXMLText(text string) string {
	var output bytes.Buffer
	_ = xml.EscapeText(&output, []byte(text))
	return output.String()
}

func ensureSlideContentType(pkg *pptx.Package, slidePart string) error {
	contentTypes, err := readContentTypes(pkg.Parts[contentTypesPartName])
	if err != nil {
		return err
	}
	for _, override := range contentTypes.Overrides {
		if strings.TrimPrefix(override.PartName, "/") == slidePart {
			return nil
		}
	}
	contentTypes.Overrides = append(contentTypes.Overrides, contentTypeOverride{PartName: "/" + slidePart, ContentType: slideContentType})
	sort.Slice(contentTypes.Overrides, func(left int, right int) bool {
		return contentTypes.Overrides[left].PartName < contentTypes.Overrides[right].PartName
	})
	pkg.Parts[contentTypesPartName] = writeContentTypes(contentTypes)
	return nil
}

func removeSlideContentType(pkg *pptx.Package, slidePart string) error {
	contentTypes, err := readContentTypes(pkg.Parts[contentTypesPartName])
	if err != nil {
		return err
	}
	partName := "/" + slidePart
	filtered := contentTypes.Overrides[:0]
	for _, override := range contentTypes.Overrides {
		if override.PartName != partName {
			filtered = append(filtered, override)
		}
	}
	contentTypes.Overrides = filtered
	pkg.Parts[contentTypesPartName] = writeContentTypes(contentTypes)
	return nil
}

type contentTypesDoc struct {
	Defaults  []contentTypeDefault  `xml:"Default"`
	Overrides []contentTypeOverride `xml:"Override"`
}

type contentTypeDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}

type contentTypeOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

func readContentTypes(data []byte) (contentTypesDoc, error) {
	var result contentTypesDoc
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&result); err != nil {
		return contentTypesDoc{}, err
	}
	return result, nil
}

func writeContentTypes(contentTypes contentTypesDoc) []byte {
	var output bytes.Buffer
	encoder := xml.NewEncoder(&output)
	typesStart := xml.StartElement{Name: xml.Name{Space: "http://schemas.openxmlformats.org/package/2006/content-types", Local: "Types"}}
	_ = encoder.EncodeToken(typesStart)
	for _, item := range contentTypes.Defaults {
		start := xml.StartElement{Name: xml.Name{Local: "Default"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "Extension"}, Value: item.Extension},
			{Name: xml.Name{Local: "ContentType"}, Value: item.ContentType},
		}}
		_ = encoder.EncodeToken(start)
		_ = encoder.EncodeToken(xml.EndElement{Name: start.Name})
	}
	for _, item := range contentTypes.Overrides {
		start := xml.StartElement{Name: xml.Name{Local: "Override"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "PartName"}, Value: item.PartName},
			{Name: xml.Name{Local: "ContentType"}, Value: item.ContentType},
		}}
		_ = encoder.EncodeToken(start)
		_ = encoder.EncodeToken(xml.EndElement{Name: start.Name})
	}
	_ = encoder.EncodeToken(xml.EndElement{Name: typesStart.Name})
	_ = encoder.Flush()
	return output.Bytes()
}

func clampInsertIndex(afterSlide int, slideCount int) int {
	if afterSlide < 0 {
		return 0
	}
	if afterSlide > slideCount {
		return slideCount
	}
	return afterSlide
}

func insertSlideRef(refs []slideRef, index int, ref slideRef) []slideRef {
	refs = append(refs, slideRef{})
	copy(refs[index+1:], refs[index:])
	refs[index] = ref
	return refs
}

func moveSlideRef(refs []slideRef, from int, to int) []slideRef {
	item := refs[from]
	refs = append(refs[:from], refs[from+1:]...)
	refs = append(refs, slideRef{})
	copy(refs[to+1:], refs[to:])
	refs[to] = item
	return refs
}

func insertString(items []string, index int, value string) []string {
	items = append(items, "")
	copy(items[index+1:], items[index:])
	items[index] = value
	return items
}

func removeString(items []string, value string) []string {
	for index, item := range items {
		if item == value {
			return append(items[:index], items[index+1:]...)
		}
	}
	return items
}

func moveString(items []string, from int, to int) []string {
	item := items[from]
	items = append(items[:from], items[from+1:]...)
	items = append(items, "")
	copy(items[to+1:], items[to:])
	items[to] = item
	return items
}

func removeRelationship(relationships []pptx.Relationship, relationshipID string) []pptx.Relationship {
	filtered := relationships[:0]
	for _, relationship := range relationships {
		if relationship.ID != relationshipID {
			filtered = append(filtered, relationship)
		}
	}
	return filtered
}
