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

func TestInspectReturnsMetadataNotesImagesAndLayout(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "rich.pptx")
	if err := fixtures.WritePPTX(filePath, fixtures.PPTXOptions{
		Metadata: fixtures.Metadata{
			Title:   "Quarterly Review",
			Creator: "Puppt Test",
			Subject: "Inspection",
		},
		Slides: []fixtures.Slide{
			{
				PartName: "ppt/slides/slide1.xml",
				Text:     "Summary",
				Notes:    "Speaker note",
				Image:    "fake png bytes",
				Layout:   "Title Layout",
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := Inspect(context.Background(), filePath)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	inspection := result.Inspection
	if inspection.Metadata.Title != "Quarterly Review" {
		t.Fatalf("unexpected title metadata: %q", inspection.Metadata.Title)
	}
	if inspection.Metadata.Author != "Puppt Test" {
		t.Fatalf("unexpected author metadata: %q", inspection.Metadata.Author)
	}
	if inspection.Metadata.Subject != "Inspection" {
		t.Fatalf("unexpected subject metadata: %q", inspection.Metadata.Subject)
	}

	slide := inspection.Slides[0]
	if slide.Layout != "ppt/slideLayouts/slideLayout1.xml" {
		t.Fatalf("unexpected layout: %q", slide.Layout)
	}
	if len(slide.Notes) != 1 || slide.Notes[0].Text != "Speaker note" {
		t.Fatalf("unexpected notes: %+v", slide.Notes)
	}
	if len(slide.Images) != 1 {
		t.Fatalf("unexpected images: %+v", slide.Images)
	}
	if slide.Images[0].Target != "ppt/media/image1.png" {
		t.Fatalf("unexpected image target: %+v", slide.Images[0])
	}
	if slide.Images[0].ContentType != "image/png" {
		t.Fatalf("unexpected image content type: %+v", slide.Images[0])
	}
}

func TestInspectReportsRepeatedVisibleText(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "repeated.pptx")
	if err := fixtures.WritePPTX(filePath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Repeat me"},
			{PartName: "ppt/slides/slide2.xml", Text: "Repeat me"},
			{PartName: "ppt/slides/slide3.xml", Text: "Unique"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := Inspect(context.Background(), filePath)
	if err != nil {
		t.Fatalf("inspect failed: %v", err)
	}

	repeated := result.Inspection.RepeatedText
	if len(repeated) != 1 {
		t.Fatalf("unexpected repeated text: %+v", repeated)
	}
	if repeated[0].Text != "Repeat me" || repeated[0].Count != 2 {
		t.Fatalf("unexpected repeated text result: %+v", repeated[0])
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
