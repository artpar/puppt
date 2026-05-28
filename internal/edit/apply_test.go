package edit

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

func TestApplyReplacesTargetTextAndPreservesUnrelatedParts(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Original title", Image: "image bytes"},
			{PartName: "ppt/slides/slide2.xml", Text: "Untouched"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Updated title"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %s: %+v", result.Status, result.Errors)
	}
	if result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid output: %+v", result.Validation)
	}

	inspection := inspectOutput(t, outputPath)
	if got := inspection.Slides[0].VisibleText[0].Text; got != "Updated title" {
		t.Fatalf("unexpected updated text: %s", got)
	}
	if got := inspection.Slides[1].VisibleText[0].Text; got != "Untouched" {
		t.Fatalf("unrelated slide changed: %s", got)
	}

	before, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatal(err)
	}
	after, err := pptx.Open(context.Background(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before.Parts["ppt/media/image1.png"]) != string(after.Parts["ppt/media/image1.png"]) {
		t.Fatal("unrelated media part changed")
	}
}

func TestApplyDeckWideTextReplacementReportsExactMatches(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Repeat"},
			{PartName: "ppt/slides/slide2.xml", Text: "Repeat"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_text",
  "target": {
    "type": "visible_text",
    "scope": "deck",
    "text": "Repeat"
  },
  "replacement": "Updated"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if len(result.Changes) != 2 {
		t.Fatalf("expected two changes: %+v", result.Changes)
	}
	for _, change := range result.Changes {
		if !strings.Contains(change.Message, "1 text match") {
			t.Fatalf("change does not report exact match count: %+v", change)
		}
	}

	inspection := inspectOutput(t, outputPath)
	for _, slide := range inspection.Slides {
		if got := slide.VisibleText[0].Text; got != "Updated" {
			t.Fatalf("unexpected slide text: %s", got)
		}
	}
}

func TestApplyUpdatesNotesRoundTrip(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Slide", Notes: "Old notes"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "update_notes",
  "target": {
    "type": "notes",
    "slide_number": 1
  },
  "replacement": "New speaker notes"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %s", result.Status)
	}

	inspection := inspectOutput(t, outputPath)
	if got := inspection.Slides[0].Notes[0].Text; got != "New speaker notes" {
		t.Fatalf("unexpected notes: %s", got)
	}
}

func TestApplyUpdatesMetadataAndValidates(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Metadata: fixtures.Metadata{Title: "Old title", Creator: "Author", Subject: "Subject"},
		Slides:   []fixtures.Slide{{PartName: "ppt/slides/slide1.xml", Text: "Slide"}},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "update_metadata",
  "target": {
    "type": "metadata",
    "property": "title"
  },
  "replacement": "New deck title"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid edit: %+v", result)
	}

	inspection := inspectOutput(t, outputPath)
	if got := inspection.Metadata.Title; got != "New deck title" {
		t.Fatalf("unexpected title: %s", got)
	}
}

func TestApplyRepeatedEditsKeepXMLNamespaceDeclarationsValid(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	firstPath := filepath.Join(dir, "first.pptx")
	secondPath := filepath.Join(dir, "second.pptx")
	thirdPath := filepath.Join(dir, "third.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Metadata: fixtures.Metadata{
			Title:          "Old title",
			Creator:        "Author",
			Subject:        "Subject",
			Keywords:       "old, keywords",
			LastModifiedBy: "Author",
		},
		Slides: []fixtures.Slide{{
			PartName: "ppt/slides/slide1.xml",
			Text:     "First text",
			Notes:    "First notes",
		}},
	}); err != nil {
		t.Fatal(err)
	}

	textSpecPath := writeSpec(t, `{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Second text"
}`)
	if result, err := Apply(context.Background(), deckPath, textSpecPath, firstPath); err != nil || result.Status != "ok" {
		t.Fatalf("first apply failed: result=%+v err=%v", result, err)
	}

	notesSpecPath := writeSpec(t, `{
  "operation": "update_notes",
  "target": {
    "type": "notes",
    "slide_number": 1
  },
  "replacement": "Second notes"
}`)
	if result, err := Apply(context.Background(), firstPath, notesSpecPath, secondPath); err != nil || result.Status != "ok" {
		t.Fatalf("second apply failed: result=%+v err=%v", result, err)
	}

	metadataSpecPath := writeSpec(t, `{
  "operation": "update_metadata",
  "target": {
    "type": "metadata",
    "property": "keywords"
  },
  "replacement": "puppt, xml, valid"
}`)
	if result, err := Apply(context.Background(), secondPath, metadataSpecPath, thirdPath); err != nil || result.Status != "ok" {
		t.Fatalf("third apply failed: result=%+v err=%v", result, err)
	}

	for _, partName := range []string{
		"ppt/slides/slide1.xml",
		"ppt/notesSlides/notesSlide1.xml",
		"docProps/core.xml",
	} {
		data := readZipPart(t, thirdPath, partName)
		if strings.Contains(string(data), "_xmlns") {
			t.Fatalf("%s contains malformed namespace attribute: %s", partName, data)
		}
	}

	inspection := inspectOutput(t, thirdPath)
	if inspection.Metadata.Keywords != "puppt, xml, valid" {
		t.Fatalf("unexpected keywords: %q", inspection.Metadata.Keywords)
	}
}

func TestReplaceDescriptionAttributes(t *testing.T) {
	input := []byte(`<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:pic><p:nvPicPr><p:cNvPr id="4" descr="Old description"/></p:nvPicPr></p:pic>
      <p:sp><p:nvSpPr><p:cNvPr id="5"/></p:nvSpPr></p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	updated, count, err := replaceDescriptionAttributes(input, "Puppt description")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unexpected replacement count: %d", count)
	}
	got := string(updated)
	if !strings.Contains(got, `descr="Puppt description"`) {
		t.Fatalf("description was not replaced: %s", got)
	}
	if strings.Contains(got, "_xmlns") {
		t.Fatalf("malformed namespace attribute: %s", got)
	}
}

func TestApplyReplacesImageAndPreservesSlideText(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{{PartName: "ppt/slides/slide1.xml", Text: "Slide", Image: "old image"}},
	}); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(dir, "replacement.png")
	if err := os.WriteFile(imagePath, []byte("new image"), 0o600); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_image",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#rId1"
  },
  "image_path": "`+imagePath+`"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if len(result.Changes) != 1 || result.Changes[0].ObjectID != "ppt/slides/slide1.xml#rId1" {
		t.Fatalf("unexpected image replacement changes: %+v", result.Changes)
	}
	pkg, err := pptx.Open(context.Background(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(pkg.Parts["ppt/media/image1.png"]); got != "new image" {
		t.Fatalf("unexpected image bytes: %s", got)
	}
	inspection := inspectOutput(t, outputPath)
	if got := inspection.Slides[0].VisibleText[0].Text; got != "Slide" {
		t.Fatalf("slide text changed during image replacement: %s", got)
	}
}

func TestApplyAddsSlideAndValidatesRelationships(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "slide_add",
  "target": {
    "type": "slide_number",
    "slide_number": 1
  },
  "replacement": "Inserted slide"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid slide add: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].SlideNumber != 2 || !strings.Contains(result.Changes[0].ObjectID, "ppt/slides/") {
		t.Fatalf("change does not name touched slide position and id: %+v", result.Changes)
	}

	inspection := inspectOutput(t, outputPath)
	if inspection.SlideCount != 2 {
		t.Fatalf("unexpected slide count: %d", inspection.SlideCount)
	}
	if got := inspection.Slides[1].VisibleText[0].Text; got != "Inserted slide" {
		t.Fatalf("unexpected inserted text: %s", got)
	}
}

func TestApplyDeletesSlideAndValidatesRelationships(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Keep"},
			{PartName: "ppt/slides/slide2.xml", Text: "Delete", Image: "image bytes"},
			{PartName: "ppt/slides/slide3.xml", Text: "Keep too"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "slide_delete",
  "target": {
    "type": "slide_number",
    "slide_number": 2
  }
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid slide delete: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].SlideNumber != 2 || result.Changes[0].ObjectID != "ppt/slides/slide2.xml" {
		t.Fatalf("change does not name deleted slide: %+v", result.Changes)
	}

	inspection := inspectOutput(t, outputPath)
	if inspection.SlideCount != 2 {
		t.Fatalf("unexpected slide count: %d", inspection.SlideCount)
	}
	for _, slide := range inspection.Slides {
		if slide.ID == "ppt/slides/slide2.xml" || slide.Title == "Delete" {
			t.Fatalf("deleted slide still present: %+v", slide)
		}
	}
}

func TestApplyMovesSlideAndValidatesOrder(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "One"},
			{PartName: "ppt/slides/slide2.xml", Text: "Two"},
			{PartName: "ppt/slides/slide3.xml", Text: "Three"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "slide_move",
  "target": {
    "type": "slide_number",
    "slide_number": 1
  },
  "destination_slide_number": 3
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid slide move: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].SlideNumber != 3 || result.Changes[0].ObjectID != "ppt/slides/slide1.xml" {
		t.Fatalf("change does not name moved slide: %+v", result.Changes)
	}

	inspection := inspectOutput(t, outputPath)
	got := []string{
		inspection.Slides[0].VisibleText[0].Text,
		inspection.Slides[1].VisibleText[0].Text,
		inspection.Slides[2].VisibleText[0].Text,
	}
	want := []string{"Two", "Three", "One"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected order: got %v want %v", got, want)
		}
	}
}

func TestApplyDuplicatesSlideAndValidatesRelationships(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "One"},
			{PartName: "ppt/slides/slide2.xml", Text: "Two", Image: "image bytes"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "slide_duplicate",
  "target": {
    "type": "slide_number",
    "slide_number": 2
  },
  "insert_after_slide": 2
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid slide duplicate: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].SlideNumber != 3 || !strings.Contains(result.Changes[0].ObjectID, "ppt/slides/") {
		t.Fatalf("change does not name duplicated slide: %+v", result.Changes)
	}

	inspection := inspectOutput(t, outputPath)
	if inspection.SlideCount != 3 {
		t.Fatalf("unexpected slide count: %d", inspection.SlideCount)
	}
	if got := inspection.Slides[2].VisibleText[0].Text; got != "Two" {
		t.Fatalf("unexpected duplicated slide text: %s", got)
	}
	if len(inspection.Slides[2].Images) != 1 {
		t.Fatalf("duplicated slide lost image relationship: %+v", inspection.Slides[2])
	}
}

func TestApplyAddsEditableTextBox(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "add_text_box",
  "target": {
    "type": "slide_number",
    "slide_number": 1
  },
  "replacement": "New editable textbox"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid text box addition: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].ObjectID != "ppt/slides/slide1.xml#shape-3" {
		t.Fatalf("unexpected changes: %+v", result.Changes)
	}
	inspection := inspectOutput(t, outputPath)
	if len(inspection.Slides[0].VisibleText) != 2 {
		t.Fatalf("expected two editable text objects: %+v", inspection.Slides[0].VisibleText)
	}
	if got := inspection.Slides[0].VisibleText[1].Text; got != "New editable textbox" {
		t.Fatalf("unexpected text box text: %s", got)
	}
}

func TestApplyAddsEditableShape(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "add_shape",
  "target": {
    "type": "slide_number",
    "slide_number": 1
  },
  "replacement": "Shape label"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid shape addition: %+v", result)
	}
	inspection := inspectOutput(t, outputPath)
	if got := inspection.Slides[0].VisibleText[1].Text; got != "Shape label" {
		t.Fatalf("unexpected shape text: %s", got)
	}
}

func inspectOutput(t *testing.T, outputPath string) *model.Inspection {
	t.Helper()
	result, err := inspect.Inspect(context.Background(), outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Inspection == nil {
		t.Fatal("missing inspection")
	}
	return result.Inspection
}

func readZipPart(t *testing.T, filePath string, partName string) []byte {
	t.Helper()
	archive, err := zip.OpenReader(filePath)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()

	for _, file := range archive.File {
		if file.Name != partName {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	t.Fatalf("part not found: %s", partName)
	return nil
}
