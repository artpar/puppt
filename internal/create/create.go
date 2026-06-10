package create

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	"github.com/artpar/puppt/internal/validate"
)

const (
	slideRelType       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"
	slideLayoutRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout"
	notesRelType       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide"
	imageRelType       = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/image"
	masterRelType      = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster"
)

// DeckSpec is the structured JSON contract for deterministic deck creation.
type DeckSpec struct {
	Metadata model.Metadata `json:"metadata,omitempty"`
	Slides   []SlideSpec    `json:"slides"`
}

type SlideSpec struct {
	Layout    string   `json:"layout"`
	Title     string   `json:"title,omitempty"`
	Body      string   `json:"body,omitempty"`
	Bullets   []string `json:"bullets,omitempty"`
	Notes     string   `json:"notes,omitempty"`
	ImagePath string   `json:"image_path,omitempty"`
}

// Create reads a structured deck JSON file and writes a deterministic .pptx.
func Create(ctx context.Context, inputPath string, outputPath string) (*model.CommandResult, error) {
	spec, err := readDeckSpec(inputPath)
	if err != nil {
		return nil, err
	}
	if err := validateDeckSpec(spec); err != nil {
		return nil, err
	}

	pkg, changes, err := buildPackage(inputPath, spec)
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
	if err := verifyCreated(ctx, outputPath, spec); err != nil {
		validation.Valid = false
		validation.Errors = append(validation.Errors, model.ErrorItem{Code: "creation_validation_failed", Message: err.Error()})
	}

	status := "ok"
	summary := fmt.Sprintf("Created %d slide deck.", len(spec.Slides))
	if !validation.Valid {
		status = "invalid"
		summary = "Created deck but validation failed."
	}
	return &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "create",
		Status:        status,
		Input:         inputPath,
		Output:        &outputPath,
		Warnings:      validation.Warnings,
		Errors:        validation.Errors,
		Summary:       model.Summary{Human: summary},
		Changes:       changes,
		Validation:    validation,
	}, nil
}

func readDeckSpec(inputPath string) (DeckSpec, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return DeckSpec{}, err
	}
	var spec DeckSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return DeckSpec{}, err
	}
	return spec, nil
}

func validateDeckSpec(spec DeckSpec) error {
	if len(spec.Slides) == 0 {
		return fmt.Errorf("deck spec requires at least one slide")
	}
	for index, slide := range spec.Slides {
		layout := normalizedLayout(slide.Layout)
		switch layout {
		case "title", "section", "title_body":
		default:
			return fmt.Errorf("slide %d has unsupported layout %q", index+1, slide.Layout)
		}
		if slide.Title == "" && slide.Body == "" && len(slide.Bullets) == 0 && slide.ImagePath == "" {
			return fmt.Errorf("slide %d has no editable content", index+1)
		}
	}
	return nil
}

func buildPackage(inputPath string, spec DeckSpec) (*pptx.Package, []model.ChangeItem, error) {
	parts := map[string][]byte{}
	changes := make([]model.ChangeItem, 0, len(spec.Slides))
	layouts := layoutCatalog()

	parts["ppt/presentation.xml"] = []byte(presentationXML(len(spec.Slides)))
	parts["ppt/_rels/presentation.xml.rels"] = []byte(presentationRelationships(len(spec.Slides)))
	parts["ppt/slideMasters/slideMaster1.xml"] = []byte(masterXML())
	for _, layout := range layouts {
		parts[layout.Part] = []byte(layoutXML(layout.Name))
		parts[pptx.RelationshipsPartFor(layout.Part)] = []byte(layoutRelationships())
	}
	if hasMetadata(spec.Metadata) {
		parts["docProps/core.xml"] = []byte(corePropertiesXML(spec.Metadata))
	}

	for index, slide := range spec.Slides {
		slideNumber := index + 1
		slidePart := fmt.Sprintf("ppt/slides/slide%d.xml", slideNumber)
		layout := layouts[normalizedLayout(slide.Layout)]
		imagePart := ""
		imageExt := ""
		imageData := []byte(nil)
		if slide.ImagePath != "" {
			data, err := os.ReadFile(slide.ImagePath)
			if err != nil {
				return nil, nil, fmt.Errorf("read slide %d image: %w", slideNumber, err)
			}
			imageExt = imageExtension(slide.ImagePath)
			imagePart = fmt.Sprintf("ppt/media/image%d.%s", slideNumber, imageExt)
			imageData = data
			parts[imagePart] = imageData
		}
		imageRelationshipID := imageRelationshipID(slide.Notes != "", imagePart != "")
		parts[slidePart] = []byte(slideXML(slide, imageRelationshipID))
		parts[pptx.RelationshipsPartFor(slidePart)] = []byte(slideRelationships(slideNumber, layout.Part, slide.Notes != "", imagePart, imageExt))
		if slide.Notes != "" {
			parts[fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", slideNumber)] = []byte(notesXML(slide.Notes))
		}
		changes = append(changes, model.ChangeItem{
			SlideNumber: slideNumber,
			ObjectID:    slidePart,
			Message:     fmt.Sprintf("Created slide %d from %s.", slideNumber, normalizedLayout(slide.Layout)),
		})
	}

	parts["[Content_Types].xml"] = []byte(contentTypesXML(spec))
	parts["_rels/.rels"] = []byte(rootRelationshipsXML(hasMetadata(spec.Metadata)))

	return &pptx.Package{
		Path:             inputPath,
		Parts:            parts,
		PresentationPath: "ppt/presentation.xml",
	}, changes, nil
}

type layoutInfo struct {
	Name string
	Part string
}

func layoutCatalog() map[string]layoutInfo {
	return map[string]layoutInfo{
		"title":      {Name: "Title", Part: "ppt/slideLayouts/slideLayout1.xml"},
		"section":    {Name: "Section", Part: "ppt/slideLayouts/slideLayout2.xml"},
		"title_body": {Name: "Title and Body", Part: "ppt/slideLayouts/slideLayout3.xml"},
	}
}

func normalizedLayout(layout string) string {
	if layout == "" {
		return "title_body"
	}
	return layout
}

func imageExtension(imagePath string) string {
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(imagePath)), ".")
	switch extension {
	case "jpg", "jpeg", "gif":
		return extension
	default:
		return "png"
	}
}

func presentationXML(slideCount int) string {
	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
`)
	for index := 0; index < slideCount; index++ {
		output.WriteString(fmt.Sprintf("    <p:sldId id=\"%d\" r:id=\"rId%d\"/>\n", 256+index, index+1))
	}
	output.WriteString(`  </p:sldIdLst>
</p:presentation>
`)
	return output.String()
}

func presentationRelationships(slideCount int) string {
	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`)
	for index := 0; index < slideCount; index++ {
		output.WriteString(fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"%s\" Target=\"slides/slide%d.xml\"/>\n", index+1, slideRelType, index+1))
	}
	output.WriteString(`</Relationships>
`)
	return output.String()
}

func slideXML(slide SlideSpec, imageRelationshipID string) string {
	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
`)
	shapeID := 2
	if slide.Title != "" {
		output.WriteString(shapeXML(shapeID, "Title 1", []string{slide.Title}))
		shapeID++
	}
	bodyRuns := append([]string{}, splitBody(slide.Body)...)
	bodyRuns = append(bodyRuns, slide.Bullets...)
	if len(bodyRuns) > 0 {
		output.WriteString(shapeXML(shapeID, "Body 1", bodyRuns))
		shapeID++
	}
	if imageRelationshipID != "" {
		output.WriteString(pictureXML(shapeID, "Picture 1", imageRelationshipID))
	}
	output.WriteString(`    </p:spTree>
  </p:cSld>
</p:sld>
`)
	return output.String()
}

func splitBody(body string) []string {
	if body == "" {
		return nil
	}
	return []string{body}
}

func shapeXML(id int, name string, runs []string) string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="%s"/>
        </p:nvSpPr>
        <p:txBody>
`, id, escapeText(name)))
	for _, run := range runs {
		output.WriteString(fmt.Sprintf(`          <a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:r><a:t>%s</a:t></a:r>
          </a:p>
`, escapeText(run)))
	}
	output.WriteString(`        </p:txBody>
      </p:sp>
`)
	return output.String()
}

func pictureXML(id int, name string, relationshipID string) string {
	return fmt.Sprintf(`      <p:pic>
        <p:nvPicPr>
          <p:cNvPr id="%d" name="%s"/>
        </p:nvPicPr>
        <p:blipFill>
          <a:blip xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" r:embed="%s"/>
          <a:stretch xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"><a:fillRect/></a:stretch>
        </p:blipFill>
        <p:spPr>
          <a:xfrm xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:off x="914400" y="2057400"/>
            <a:ext cx="10363200" cy="4114800"/>
          </a:xfrm>
          <a:prstGeom xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" prst="rect"><a:avLst/></a:prstGeom>
        </p:spPr>
      </p:pic>
`, id, escapeText(name), relationshipID)
}

func slideRelationships(slideNumber int, layoutPart string, hasNotes bool, imagePart string, imageExt string) string {
	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`)
	output.WriteString(fmt.Sprintf("  <Relationship Id=\"rId1\" Type=\"%s\" Target=\"../slideLayouts/%s\"/>\n", slideLayoutRelType, path.Base(layoutPart)))
	nextID := 2
	if hasNotes {
		output.WriteString(fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"%s\" Target=\"../notesSlides/notesSlide%d.xml\"/>\n", nextID, notesRelType, slideNumber))
		nextID++
	}
	if imagePart != "" {
		output.WriteString(fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"%s\" Target=\"../media/%s\"/>\n", nextID, imageRelType, path.Base(imagePart)))
	}
	output.WriteString(`</Relationships>
`)
	return output.String()
}

func imageRelationshipID(hasNotes bool, hasImage bool) string {
	if !hasImage {
		return ""
	}
	if hasNotes {
		return "rId3"
	}
	return "rId2"
}

func notesXML(notes string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:notes xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Notes Placeholder"/>
        </p:nvSpPr>
        <p:txBody>
          <a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:r><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:notes>
`, escapeText(notes))
}

func layoutXML(name string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="title" preserve="1">
  <p:cSld name="%s"/>
</p:sldLayout>
`, escapeText(name))
}

func layoutRelationships() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="%s" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>
`, masterRelType)
}

func masterXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld name="Puppt Master"/>
</p:sldMaster>
`
}

func contentTypesXML(spec DeckSpec) string {
	extensions := map[string]string{
		"rels": "application/vnd.openxmlformats-package.relationships+xml",
		"xml":  "application/xml",
		"png":  "image/png",
	}
	for _, slide := range spec.Slides {
		if slide.ImagePath == "" {
			continue
		}
		switch imageExtension(slide.ImagePath) {
		case "jpg":
			extensions["jpg"] = "image/jpeg"
		case "jpeg":
			extensions["jpeg"] = "image/jpeg"
		case "gif":
			extensions["gif"] = "image/gif"
		}
	}

	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
`)
	for _, extension := range []string{"rels", "xml", "png", "jpg", "jpeg", "gif"} {
		contentType, ok := extensions[extension]
		if ok {
			output.WriteString(fmt.Sprintf("  <Default Extension=\"%s\" ContentType=\"%s\"/>\n", extension, contentType))
		}
	}
	output.WriteString(`  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
`)
	for index := 1; index <= 3; index++ {
		output.WriteString(fmt.Sprintf("  <Override PartName=\"/ppt/slideLayouts/slideLayout%d.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml\"/>\n", index))
	}
	if hasMetadata(spec.Metadata) {
		output.WriteString(`  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
`)
	}
	for index, slide := range spec.Slides {
		output.WriteString(fmt.Sprintf("  <Override PartName=\"/ppt/slides/slide%d.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slide+xml\"/>\n", index+1))
		if slide.Notes != "" {
			output.WriteString(fmt.Sprintf("  <Override PartName=\"/ppt/notesSlides/notesSlide%d.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml\"/>\n", index+1))
		}
	}
	output.WriteString(`</Types>
`)
	return output.String()
}

func rootRelationshipsXML(includeMetadata bool) string {
	var output strings.Builder
	output.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
`)
	if includeMetadata {
		output.WriteString(`  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
`)
	}
	output.WriteString(`</Relationships>
`)
	return output.String()
}

func corePropertiesXML(metadata model.Metadata) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <dc:title>%s</dc:title>
  <dc:creator>%s</dc:creator>
  <dc:subject>%s</dc:subject>
</cp:coreProperties>
`, escapeText(metadata.Title), escapeText(metadata.Author), escapeText(metadata.Subject))
}

func hasMetadata(metadata model.Metadata) bool {
	return metadata.Title != "" || metadata.Author != "" || metadata.Subject != ""
}

func verifyCreated(ctx context.Context, outputPath string, spec DeckSpec) error {
	result, err := inspect.Inspect(ctx, outputPath)
	if err != nil {
		return err
	}
	inspection := result.Inspection
	if inspection == nil {
		return fmt.Errorf("inspection result missing")
	}
	if inspection.SlideCount != len(spec.Slides) {
		return fmt.Errorf("created slide count mismatch: got %d want %d", inspection.SlideCount, len(spec.Slides))
	}
	if hasMetadata(spec.Metadata) {
		if inspection.Metadata.Title != spec.Metadata.Title || inspection.Metadata.Author != spec.Metadata.Author || inspection.Metadata.Subject != spec.Metadata.Subject {
			return fmt.Errorf("created metadata mismatch")
		}
	}
	for index, slide := range spec.Slides {
		created := inspection.Slides[index]
		for _, want := range expectedSlideTexts(slide) {
			if !slideContainsText(created, want) {
				return fmt.Errorf("slide %d missing text %q", index+1, want)
			}
		}
		if slide.Notes != "" && !slideNotesContain(created, slide.Notes) {
			return fmt.Errorf("slide %d missing notes", index+1)
		}
		if slide.ImagePath != "" && len(created.Images) == 0 {
			return fmt.Errorf("slide %d missing image reference", index+1)
		}
	}
	return nil
}

func expectedSlideTexts(slide SlideSpec) []string {
	var texts []string
	if slide.Title != "" {
		texts = append(texts, slide.Title)
	}
	if slide.Body != "" {
		texts = append(texts, slide.Body)
	}
	texts = append(texts, slide.Bullets...)
	return texts
}

func slideContainsText(slide model.Slide, text string) bool {
	for _, block := range slide.VisibleText {
		for _, run := range block.Runs {
			if run == text {
				return true
			}
		}
		if strings.Contains(block.Text, text) {
			return true
		}
	}
	return false
}

func slideNotesContain(slide model.Slide, text string) bool {
	for _, block := range slide.Notes {
		if strings.Contains(block.Text, text) {
			return true
		}
	}
	return false
}

func escapeText(text string) string {
	var output bytes.Buffer
	_ = xml.EscapeText(&output, []byte(text))
	return output.String()
}
