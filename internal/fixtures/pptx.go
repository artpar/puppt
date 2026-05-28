package fixtures

import (
	"archive/zip"
	"fmt"
	"os"
)

// WriteMinimalPPTX writes a deterministic minimal modern .pptx package with
// the provided slide part names in presentation order.
func WriteMinimalPPTX(filePath string, slidePartNames []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	for _, part := range minimalParts(slidePartNames) {
		writer, err := archive.Create(part.Name)
		if err != nil {
			archive.Close()
			return err
		}
		if _, err := writer.Write([]byte(part.Data)); err != nil {
			archive.Close()
			return err
		}
	}
	return archive.Close()
}

type part struct {
	Name string
	Data string
}

func minimalParts(slidePartNames []string) []part {
	parts := []part{
		{Name: "[Content_Types].xml", Data: contentTypes(slidePartNames)},
		{Name: "_rels/.rels", Data: rootRelationships()},
		{Name: "ppt/presentation.xml", Data: presentation(slidePartNames)},
		{Name: "ppt/_rels/presentation.xml.rels", Data: presentationRelationships(slidePartNames)},
	}
	for i, name := range slidePartNames {
		parts = append(parts, part{Name: name, Data: slideXML(i + 1)})
	}
	return parts
}

func contentTypes(slidePartNames []string) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
`
	for _, name := range slidePartNames {
		output += fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slide+xml\"/>\n", name)
	}
	return output + `</Types>
`
}

func rootRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>
`
}

func presentation(slidePartNames []string) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>
`
	for i := range slidePartNames {
		output += fmt.Sprintf("    <p:sldId id=\"%d\" r:id=\"rId%d\"/>\n", 256+i, i+1)
	}
	return output + `  </p:sldIdLst>
</p:presentation>
`
}

func presentationRelationships(slidePartNames []string) string {
	output := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
`
	for i, name := range slidePartNames {
		output += fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide\" Target=\"slides/%s\"/>\n", i+1, baseSlideName(name))
	}
	return output + `</Relationships>
`
}

func baseSlideName(name string) string {
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' {
			return name[i+1:]
		}
	}
	return name
}

func slideXML(index int) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:txBody>
          <a:p xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
            <a:r><a:t>Slide %d</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>
`, index)
}
