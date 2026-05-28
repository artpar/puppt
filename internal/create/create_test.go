package create

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/validate"
)

func TestCreateBuildsInspectableValidDeck(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "image.png")
	if err := os.WriteFile(imagePath, []byte("image bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	inputPath := writeDeckSpec(t, dir, `{
  "metadata": {
    "title": "Created Deck",
    "author": "Puppt",
    "subject": "Creation"
  },
  "slides": [
    {
      "layout": "title",
      "title": "Opening"
    },
    {
      "layout": "section",
      "title": "Section One",
      "notes": "Talk track"
    },
    {
      "layout": "title_body",
      "title": "Details",
      "body": "Body text",
      "bullets": ["First bullet", "Second bullet"],
      "image_path": "`+imagePath+`"
    }
  ]
}`)
	outputPath := filepath.Join(dir, "created.pptx")

	result, err := Create(context.Background(), inputPath, outputPath)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("expected valid created deck: %+v", result)
	}
	if len(result.Changes) != 3 {
		t.Fatalf("unexpected changes: %+v", result.Changes)
	}

	inspectionResult, err := inspect.Inspect(context.Background(), outputPath)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	inspection := inspectionResult.Inspection
	if inspection.SlideCount != 3 {
		t.Fatalf("unexpected slide count: %d", inspection.SlideCount)
	}
	if inspection.Metadata.Title != "Created Deck" || inspection.Metadata.Author != "Puppt" || inspection.Metadata.Subject != "Creation" {
		t.Fatalf("unexpected metadata: %+v", inspection.Metadata)
	}
	if got := inspection.Slides[0].Title; got != "Opening" {
		t.Fatalf("unexpected title slide: %s", got)
	}
	if got := inspection.Slides[1].Notes[0].Text; got != "Talk track" {
		t.Fatalf("unexpected notes: %s", got)
	}
	body := inspection.Slides[2].VisibleText[1]
	for _, want := range []string{"Body text", "First bullet", "Second bullet"} {
		if !contains(body.Runs, want) {
			t.Fatalf("body runs missing %q: %+v", want, body.Runs)
		}
	}
	if len(inspection.Slides[2].Images) != 1 {
		t.Fatalf("expected image ref: %+v", inspection.Slides[2].Images)
	}

	validationResult, err := validate.Validate(context.Background(), outputPath)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if validationResult.Status != "ok" || validationResult.Validation == nil || !validationResult.Validation.Valid {
		t.Fatalf("unexpected validation result: %+v", validationResult)
	}
}

func TestCreateOutputIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	inputPath := writeDeckSpec(t, dir, `{
  "metadata": {
    "title": "Stable"
  },
  "slides": [
    {
      "layout": "title_body",
      "title": "Slide",
      "body": "Same content",
      "bullets": ["A", "B"]
    }
  ]
}`)
	firstPath := filepath.Join(dir, "first.pptx")
	secondPath := filepath.Join(dir, "second.pptx")

	if _, err := Create(context.Background(), inputPath, firstPath); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	if _, err := Create(context.Background(), inputPath, secondPath); err != nil {
		t.Fatalf("second create failed: %v", err)
	}
	first, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("created decks are not byte deterministic")
	}
}

func TestCreateRejectsUnsupportedLayout(t *testing.T) {
	dir := t.TempDir()
	inputPath := writeDeckSpec(t, dir, `{
  "slides": [
    {
      "layout": "freeform",
      "title": "No"
    }
  ]
}`)
	_, err := Create(context.Background(), inputPath, filepath.Join(dir, "out.pptx"))
	if err == nil {
		t.Fatal("unsupported layout unexpectedly succeeded")
	}
}

func writeDeckSpec(t *testing.T, dir string, data string) string {
	t.Helper()
	inputPath := filepath.Join(dir, "deck.json")
	if err := os.WriteFile(inputPath, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return inputPath
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
