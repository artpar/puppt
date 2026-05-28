package edit

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
)

func TestPlanReadyForObjectID(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "New title"
}`)

	result, err := Plan(context.Background(), deckPath, specPath)
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.Plan == nil || result.Plan.Status != "ready" {
		t.Fatalf("unexpected plan: %+v", result.Plan)
	}
	if len(result.Plan.Matches) != 1 {
		t.Fatalf("unexpected matches: %+v", result.Plan.Matches)
	}
}

func TestPlanReportsAmbiguousVisibleText(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
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
    "text": "Repeat"
  },
  "replacement": "Updated"
}`)

	result, err := Plan(context.Background(), deckPath, specPath)
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if result.Status != "ambiguous" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if len(result.Ambiguous) != 1 {
		t.Fatalf("expected ambiguous item: %+v", result.Ambiguous)
	}
}

func TestPlanReportsUnsupportedOperationTarget(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "update_metadata",
  "target": {
    "type": "visible_text",
    "text": "Slide 1"
  },
  "replacement": "Updated"
}`)

	result, err := Plan(context.Background(), deckPath, specPath)
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if result.Status != "unsupported" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if len(result.Unsupported) != 1 {
		t.Fatalf("expected unsupported item: %+v", result.Unsupported)
	}
}

func TestPlanReportsMissingRequiredReplacement(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := writeSpec(t, `{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  }
}`)

	result, err := Plan(context.Background(), deckPath, specPath)
	if err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if result.Status != "unsupported" {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.Plan.Message != "replace_text requires replacement" {
		t.Fatalf("unexpected message: %s", result.Plan.Message)
	}
}

func writeSpec(t *testing.T, data string) string {
	t.Helper()
	filePath := filepath.Join(t.TempDir(), "edit.json")
	if err := os.WriteFile(filePath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return filePath
}
