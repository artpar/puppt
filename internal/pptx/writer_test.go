package pptx

import (
	"archive/zip"
	"context"
	"io"
	"path/filepath"
	"testing"
)

func TestWritePreservesUnsupportedPayloadParts(t *testing.T) {
	output := filepath.Join(t.TempDir(), "unsupported-payloads.pptx")
	parts := map[string][]byte{
		"[Content_Types].xml": []byte(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/ppt/charts/chart1.xml" ContentType="application/vnd.openxmlformats-officedocument.drawingml.chart+xml"/>
  <Override PartName="/ppt/embeddings/oleObject1.bin" ContentType="application/vnd.openxmlformats-officedocument.oleObject"/>
</Types>`),
		"_rels/.rels":                   []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`),
		"ppt/charts/chart1.xml":         []byte(`<c:chartSpace xmlns:c="c"><c:chart/></c:chartSpace>`),
		"ppt/embeddings/oleObject1.bin": []byte("opaque ole bytes"),
		"ppt/activeX/activeX1.xml":      []byte(`<ax:ocx xmlns:ax="ax"/>`),
		"ppt/media/media1.mp4":          []byte("video bytes"),
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rIdChart" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/chart" Target="../charts/chart1.xml"/>
  <Relationship Id="rIdOle" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject" Target="../embeddings/oleObject1.bin"/>
</Relationships>`),
	}

	if err := Write(context.Background(), &Package{Parts: parts}, output); err != nil {
		t.Fatalf("write package: %v", err)
	}
	archive, err := zip.OpenReader(output)
	if err != nil {
		t.Fatalf("open written package: %v", err)
	}
	defer archive.Close()
	got := map[string][]byte{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("open part %s: %v", file.Name, err)
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			reader.Close()
			t.Fatalf("read part %s: %v", file.Name, err)
		}
		reader.Close()
		got[file.Name] = data
	}
	for name, want := range parts {
		if string(got[name]) != string(want) {
			t.Fatalf("part %s was not preserved: got %q want %q", name, got[name], want)
		}
	}
}
