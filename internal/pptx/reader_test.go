package pptx

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
)

func TestOpenRejectsUnsupportedFileType(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.txt")
	if err := os.WriteFile(filePath, []byte("not a deck"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(context.Background(), filePath)
	assertPackageError(t, err, ErrorUnsupportedFileType)
}

func TestOpenRejectsInvalidZip(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := os.WriteFile(filePath, []byte("not a zip"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Open(context.Background(), filePath)
	assertPackageError(t, err, ErrorInvalidPackage)
}

func TestOpenRejectsMissingContentTypes(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	writeZip(t, filePath, map[string]string{
		rootRelationshipsPart: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>
`,
	})

	_, err := Open(context.Background(), filePath)
	assertPackageError(t, err, ErrorMissingPart)
}

func TestOpenReadsPackageAndSlideOrder(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	slides := []string{"ppt/slides/slide1.xml", "ppt/slides/slide2.xml"}
	if err := fixtures.WriteMinimalPPTX(filePath, slides); err != nil {
		t.Fatal(err)
	}

	pkg, err := Open(context.Background(), filePath)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	if pkg.PresentationPath != "ppt/presentation.xml" {
		t.Fatalf("unexpected presentation path: %s", pkg.PresentationPath)
	}
	if !reflect.DeepEqual(pkg.SlideParts, slides) {
		t.Fatalf("unexpected slide order: got %v want %v", pkg.SlideParts, slides)
	}
	for _, want := range []string{contentTypesPart, rootRelationshipsPart, "ppt/presentation.xml", "ppt/_rels/presentation.xml.rels", slides[0], slides[1]} {
		if _, ok := pkg.Parts[want]; !ok {
			t.Fatalf("missing part %s", want)
		}
	}
	if got := pkg.ContentTypes.Overrides["ppt/presentation.xml"]; got == "" {
		t.Fatal("missing presentation content type override")
	}
	if len(pkg.RootRelationships) != 1 {
		t.Fatalf("unexpected root relationship count: %d", len(pkg.RootRelationships))
	}
	if len(pkg.PresentationRelationships) != 2 {
		t.Fatalf("unexpected presentation relationship count: %d", len(pkg.PresentationRelationships))
	}
}

func assertPackageError(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	var packageErr *PackageError
	if !errors.As(err, &packageErr) {
		t.Fatalf("expected PackageError, got %T: %v", err, err)
	}
	if packageErr.Kind != want {
		t.Fatalf("unexpected error kind: got %s want %s", packageErr.Kind, want)
	}
}

func writeZip(t *testing.T, filePath string, parts map[string]string) {
	t.Helper()
	file, err := os.Create(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	archive := zip.NewWriter(file)
	for name, data := range parts {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write([]byte(data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
}
