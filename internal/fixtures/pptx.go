package fixtures

import (
	"archive/zip"
	"fmt"
	"os"
)

// PPTXOptions describes a deterministic fixture deck.
type PPTXOptions struct {
	Metadata Metadata
	Slides   []Slide
}

type Metadata struct {
	Title   string
	Creator string
	Subject string
}

type Slide struct {
	PartName string
	Text     string
	Notes    string
	Image    string
	Layout   string
}

// WriteMinimalPPTX writes a deterministic minimal modern .pptx package with
// the provided slide part names in presentation order.
func WriteMinimalPPTX(filePath string, slidePartNames []string) error {
	slides := make([]Slide, 0, len(slidePartNames))
	for index, partName := range slidePartNames {
		slides = append(slides, Slide{
			PartName: partName,
			Text:     fmt.Sprintf("Slide %d", index+1),
		})
	}
	return WritePPTX(filePath, PPTXOptions{Slides: slides})
}

// WritePPTX writes a deterministic fixture deck with optional metadata, notes,
// image relationships, and layout relationships.
func WritePPTX(filePath string, options PPTXOptions) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	for _, part := range fixtureParts(options) {
		writer, err := archive.Create(part.Name)
		if err != nil {
			archive.Close()
			return err
		}
		if _, err := writer.Write(part.Data); err != nil {
			archive.Close()
			return err
		}
	}
	return archive.Close()
}

type part struct {
	Name string
	Data []byte
}

func fixtureParts(options PPTXOptions) []part {
	parts := []part{
		{Name: "[Content_Types].xml", Data: []byte(contentTypes(options))},
		{Name: "_rels/.rels", Data: []byte(rootRelationships(options.Metadata))},
		{Name: "ppt/presentation.xml", Data: []byte(presentation(options.Slides))},
		{Name: "ppt/_rels/presentation.xml.rels", Data: []byte(presentationRelationships(options.Slides))},
	}

	if hasMetadata(options.Metadata) {
		parts = append(parts, part{Name: "docProps/core.xml", Data: []byte(coreProperties(options.Metadata))})
	}

	for index, slide := range options.Slides {
		parts = append(parts, part{Name: slide.PartName, Data: []byte(slideXML(slide.Text))})
		if slide.Notes != "" {
			parts = append(parts, part{Name: notesPart(index), Data: []byte(notesXML(slide.Notes))})
		}
		if slide.Image != "" {
			parts = append(parts, part{Name: imagePart(index), Data: []byte(slide.Image)})
		}
		if slide.Layout != "" {
			parts = append(parts, part{Name: layoutPart(index), Data: []byte(layoutXML(slide.Layout))})
		}
		if slide.Notes != "" || slide.Image != "" || slide.Layout != "" {
			parts = append(parts, part{Name: slideRelationshipsPart(slide.PartName), Data: []byte(slideRelationships(index, slide))})
		}
	}
	return parts
}

func contentTypes(options PPTXOptions) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
`
	if hasMetadata(options.Metadata) {
		output += `  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
`
	}
	for index, slide := range options.Slides {
		output += fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slide+xml\"/>\n", slide.PartName)
		if slide.Notes != "" {
			output += fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.notesSlide+xml\"/>\n", notesPart(index))
		}
		if slide.Layout != "" {
			output += fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml\"/>\n", layoutPart(index))
		}
	}
	return output + `</Types>
`
}

func rootRelationships(metadata Metadata) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
`
	if hasMetadata(metadata) {
		output += `  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
`
	}
	return output + `</Relationships>
`
}

func presentation(slides []Slide) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
`
	for i := range slides {
		output += fmt.Sprintf("    <p:sldId id=\"%d\" r:id=\"rId%d\"/>\n", 256+i, i+1)
	}
	return output + `  </p:sldIdLst>
</p:presentation>
`
}

func presentationRelationships(slides []Slide) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`
	for i, slide := range slides {
		output += fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide\" Target=\"slides/%s\"/>\n", i+1, baseSlideName(slide.PartName))
	}
	return output + `</Relationships>
`
}

func slideRelationships(index int, slide Slide) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`
	nextID := 1
	if slide.Notes != "" {
		output += fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesSlide\" Target=\"../notesSlides/notesSlide%d.xml\"/>\n", nextID, index+1)
		nextID++
	}
	if slide.Image != "" {
		output += fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/image\" Target=\"../media/image%d.png\"/>\n", nextID, index+1)
		nextID++
	}
	if slide.Layout != "" {
		output += fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout\" Target=\"../slideLayouts/slideLayout%d.xml\"/>\n", nextID, index+1)
	}
	return output + `</Relationships>
`
}

func coreProperties(metadata Metadata) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <dc:title>%s</dc:title>
  <dc:creator>%s</dc:creator>
  <dc:subject>%s</dc:subject>
</cp:coreProperties>
`, metadata.Title, metadata.Creator, metadata.Subject)
}

func hasMetadata(metadata Metadata) bool {
	return metadata.Title != "" || metadata.Creator != "" || metadata.Subject != ""
}

func baseSlideName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}

func slideRelationshipsPart(slidePart string) string {
	return "ppt/slides/_rels/" + baseSlideName(slidePart) + ".rels"
}

func notesPart(index int) string {
	return fmt.Sprintf("ppt/notesSlides/notesSlide%d.xml", index+1)
}

func imagePart(index int) string {
	return fmt.Sprintf("ppt/media/image%d.png", index+1)
}

func layoutPart(index int) string {
	return fmt.Sprintf("ppt/slideLayouts/slideLayout%d.xml", index+1)
}

func slideXML(text string) string {
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
`, text)
}

func notesXML(text string) string {
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
`, text)
}

func layoutXML(name string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" type="title" preserve="1">
  <p:cSld name="%s"/>
</p:sldLayout>
`, name)
}
