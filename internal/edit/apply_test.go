package edit

import (
	"context"
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

func TestApplyRejectsUnsupportedMutationWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{{PartName: "ppt/slides/slide1.xml", Text: "Slide", Image: "old image"}},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_image",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#rId1"
  },
  "image_path": "replacement.png"
}`)

	result, err := Apply(context.Background(), deckPath, specPath, outputPath)
	if err != nil {
		t.Fatalf("apply failed: %v", err)
	}
	if result.Status != "unsupported" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if len(result.Unsupported) != 1 {
		t.Fatalf("expected unsupported item: %+v", result.Unsupported)
	}
	if result.Output != nil {
		t.Fatalf("unsupported mutation reported output: %s", *result.Output)
	}
	if _, err := pptx.Open(context.Background(), outputPath); err == nil {
		t.Fatal("unsupported mutation wrote output")
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
