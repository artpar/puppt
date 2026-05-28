package inspect

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
	"github.com/artpar/puppt/internal/report"
)

func TestInspectReturnsSlideOrderAndVisibleText(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(filePath, []string{"ppt/slides/slide1.xml", "ppt/slides/slide2.xml"}); err != nil {
		t.Fatal(err)
	}

	result, err := Inspect(context.Background(), filePath)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	if result.SchemaVersion != "puppt.v1" {
		t.Fatalf("unexpected schema version: %s", result.SchemaVersion)
	}
	if result.Command != "inspect" || result.Status != "ok" {
		t.Fatalf("unexpected command status: %s %s", result.Command, result.Status)
	}
	if result.Inspection == nil {
		t.Fatal("missing inspection")
	}
	if result.Inspection.SlideCount != 2 {
		t.Fatalf("unexpected slide count: %d", result.Inspection.SlideCount)
	}
	if got := result.Inspection.Slides[0].Part; got != "ppt/slides/slide1.xml" {
		t.Fatalf("unexpected first slide part: %s", got)
	}
	if got := result.Inspection.Slides[0].VisibleText[0].Text; got != "Slide 1" {
		t.Fatalf("unexpected first slide text: %s", got)
	}
	if got := result.Inspection.Slides[1].VisibleText[0].Text; got != "Slide 2" {
		t.Fatalf("unexpected second slide text: %s", got)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected partial inspection warning")
	}
}

func TestInspectGoldenJSON(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "minimal.pptx")
	if err := fixtures.WriteMinimalPPTX(filePath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	result, err := Inspect(context.Background(), filePath)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	result.Input = "testdata/minimal.pptx"

	var output bytes.Buffer
	if err := report.WriteJSON(&output, result); err != nil {
		t.Fatalf("write json: %v", err)
	}

	golden, err := os.ReadFile(filepath.Join("testdata", "minimal.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	if output.String() != string(golden) {
		t.Fatalf("golden mismatch\n--- got ---\n%s\n--- want ---\n%s", output.String(), string(golden))
	}
}
