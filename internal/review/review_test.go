package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
)

func TestReviewReadsCommandResultChanges(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	changesPath := filepath.Join(dir, "changes.json")
	if err := os.WriteFile(changesPath, []byte(`{
  "schema_version": "puppt.v1",
  "changes": [
    {
      "slide_number": 1,
      "object_id": "ppt/slides/slide1.xml#shape-2",
      "message": "Changed title."
    }
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Review(context.Background(), deckPath, changesPath)
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if result.Status != "ok" || result.Inspection == nil || result.Inspection.SlideCount != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Changes) != 1 || result.Changes[0].Message != "Changed title." {
		t.Fatalf("unexpected changes: %+v", result.Changes)
	}
}

func TestReviewReadsChangeArray(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	changesPath := filepath.Join(dir, "changes.json")
	if err := os.WriteFile(changesPath, []byte(`[
  {
    "slide_number": 1,
    "object_id": "ppt/slides/slide1.xml#shape-2",
    "message": "Changed title."
  }
]`), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Review(context.Background(), deckPath, changesPath)
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("unexpected changes: %+v", result.Changes)
	}
}
