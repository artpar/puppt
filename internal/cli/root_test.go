package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
)

func TestHelpListsRequiredV1Commands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute(context.Background(), []string{"--help"}, &stdout, &stderr); err != nil {
		t.Fatalf("help failed: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"inspect", "plan", "edit", "create", "validate", "review", "render", "version"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote stderr: %s", stderr.String())
	}
}

func TestRenderJSON(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "slide-1.png")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"render", deckPath, "--slide", "1", "--out", outputPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("render failed: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("render wrote stderr: %s", stderr.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Output        string `json:"output"`
		Render        struct {
			SlideNumber int    `json:"slide_number"`
			SlidePart   string `json:"slide_part"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
		} `json:"render"`
		Unsupported []struct {
			Code string `json:"code"`
			Part string `json:"part"`
		} `json:"unsupported"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "render" || payload.Status != "partial" {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	if payload.Output != outputPath || payload.Render.SlideNumber != 1 || payload.Render.SlidePart != "ppt/slides/slide1.xml" {
		t.Fatalf("unexpected render payload: %+v", payload)
	}
	if payload.Render.Width != 960 || payload.Render.Height != 540 || len(payload.Unsupported) != 1 {
		t.Fatalf("unexpected render details: %+v", payload)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("render did not write output: %v", err)
	}
}

func TestRenderJSONHonorsDPIFlag(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "slide-1.png")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"render", deckPath, "--slide", "1", "--out", outputPath, "--dpi", "96", "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("render failed: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("render wrote stderr: %s", stderr.String())
	}

	var payload struct {
		Render struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"render"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.Render.Width != 1280 || payload.Render.Height != 720 {
		t.Fatalf("unexpected render dimensions: %+v", payload.Render)
	}
}

func TestVersionIncludesSchemaVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute(context.Background(), []string{"version"}, &stdout, &stderr); err != nil {
		t.Fatalf("version failed: %v", err)
	}

	output := stdout.String()
	for _, want := range []string{"puppt", "dev", "puppt.v1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("version output missing %q: %s", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("version wrote stderr: %s", stderr.String())
	}
}

func TestUnknownCommandFails(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute(context.Background(), []string{"unknown"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("unknown command unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEditJSON(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "out.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "edit.json")
	if err := os.WriteFile(specPath, []byte(`{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Updated"
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"edit", deckPath, "--edit", specPath, "--out", outputPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("edit failed: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("edit wrote stderr: %s", stderr.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Output        string `json:"output"`
		Validation    struct {
			Valid bool `json:"valid"`
		} `json:"validation"`
		Changes []struct {
			ObjectID string `json:"object_id"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "edit" || payload.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	if payload.Output != outputPath || !payload.Validation.Valid || len(payload.Changes) != 1 {
		t.Fatalf("unexpected edit payload: %+v", payload)
	}
}

func TestCreateJSON(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "deck.json")
	outputPath := filepath.Join(dir, "created.pptx")
	if err := os.WriteFile(inputPath, []byte(`{
  "metadata": {
    "title": "CLI Deck"
  },
  "slides": [
    {
      "layout": "title",
      "title": "Created from CLI"
    }
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"create", "--input", inputPath, "--out", outputPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("create failed: %v\n%s", err, stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("create wrote stderr: %s", stderr.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Output        string `json:"output"`
		Validation    struct {
			Valid bool `json:"valid"`
		} `json:"validation"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "create" || payload.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	if payload.Output != outputPath || !payload.Validation.Valid {
		t.Fatalf("unexpected create payload: %+v", payload)
	}
}

func TestValidateJSON(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(filePath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"validate", filePath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("validate failed: %v\n%s", err, stdout.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Validation    struct {
			Valid bool `json:"valid"`
		} `json:"validation"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "validate" || payload.Status != "ok" || !payload.Validation.Valid {
		t.Fatalf("unexpected validate payload: %+v", payload)
	}
	if stderr.Len() != 0 {
		t.Fatalf("validate wrote stderr: %s", stderr.String())
	}
}

func TestReviewJSON(t *testing.T) {
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
      "message": "Reviewed change."
    }
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := Execute(context.Background(), []string{"review", deckPath, "--changes", changesPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("review failed: %v\n%s", err, stdout.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Changes       []struct {
			Message string `json:"message"`
		} `json:"changes"`
		Inspection struct {
			SlideCount int `json:"slide_count"`
		} `json:"inspection"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "review" || payload.Status != "ok" {
		t.Fatalf("unexpected review payload: %+v", payload)
	}
	if len(payload.Changes) != 1 || payload.Inspection.SlideCount != 1 {
		t.Fatalf("unexpected review details: %+v", payload)
	}
	if stderr.Len() != 0 {
		t.Fatalf("review wrote stderr: %s", stderr.String())
	}
}

func TestAcceptanceWorkflowEndToEnd(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "deck.json")
	createdPath := filepath.Join(dir, "created.pptx")
	editedPath := filepath.Join(dir, "edited.pptx")
	if err := os.WriteFile(inputPath, []byte(`{
  "metadata": {
    "title": "Acceptance Deck"
  },
  "slides": [
    {
      "layout": "title_body",
      "title": "Original",
      "body": "Body"
    }
  ]
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var createOut bytes.Buffer
	if err := Execute(context.Background(), []string{"create", "--input", inputPath, "--out", createdPath, "--json"}, &createOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("create failed: %v\n%s", err, createOut.String())
	}
	var inspectOut bytes.Buffer
	if err := Execute(context.Background(), []string{"inspect", createdPath, "--json"}, &inspectOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("inspect failed: %v\n%s", err, inspectOut.String())
	}
	editPath := filepath.Join(dir, "edit.json")
	if err := os.WriteFile(editPath, []byte(`{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Updated"
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var editOut bytes.Buffer
	if err := Execute(context.Background(), []string{"edit", createdPath, "--edit", editPath, "--out", editedPath, "--json"}, &editOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("edit failed: %v\n%s", err, editOut.String())
	}
	changesPath := filepath.Join(dir, "changes.json")
	if err := os.WriteFile(changesPath, editOut.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	var validateOut bytes.Buffer
	if err := Execute(context.Background(), []string{"validate", editedPath, "--json"}, &validateOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("validate failed: %v\n%s", err, validateOut.String())
	}
	var reviewOut bytes.Buffer
	if err := Execute(context.Background(), []string{"review", editedPath, "--changes", changesPath, "--json"}, &reviewOut, &bytes.Buffer{}); err != nil {
		t.Fatalf("review failed: %v\n%s", err, reviewOut.String())
	}

	var reviewPayload struct {
		Status  string `json:"status"`
		Changes []struct {
			ObjectID string `json:"object_id"`
		} `json:"changes"`
	}
	if err := json.Unmarshal(reviewOut.Bytes(), &reviewPayload); err != nil {
		t.Fatalf("invalid review json: %v\n%s", err, reviewOut.String())
	}
	if reviewPayload.Status != "ok" || len(reviewPayload.Changes) != 1 {
		t.Fatalf("unexpected acceptance review payload: %+v", reviewPayload)
	}
}

func TestInspectJSON(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(filePath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute(context.Background(), []string{"inspect", filePath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("inspect failed: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("inspect wrote stderr: %s", stderr.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Inspection    struct {
			SlideCount int `json:"slide_count"`
			Slides     []struct {
				VisibleText []struct {
					Text string `json:"text"`
				} `json:"visible_text"`
			} `json:"slides"`
		} `json:"inspection"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "inspect" || payload.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	if payload.Inspection.SlideCount != 1 {
		t.Fatalf("unexpected slide count: %d", payload.Inspection.SlideCount)
	}
	if got := payload.Inspection.Slides[0].VisibleText[0].Text; got != "Slide 1" {
		t.Fatalf("unexpected text: %s", got)
	}
}

func TestPlanJSON(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "edit.json")
	if err := os.WriteFile(specPath, []byte(`{
  "operation": "replace_text",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  },
  "replacement": "Updated"
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := Execute(context.Background(), []string{"plan", deckPath, "--edit", specPath, "--json"}, &stdout, &stderr); err != nil {
		t.Fatalf("plan failed: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("plan wrote stderr: %s", stderr.String())
	}

	var payload struct {
		SchemaVersion string `json:"schema_version"`
		Command       string `json:"command"`
		Status        string `json:"status"`
		Plan          struct {
			Operation string `json:"operation"`
			Status    string `json:"status"`
			Matches   []struct {
				ObjectID string `json:"object_id"`
			} `json:"matches"`
		} `json:"plan"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != "puppt.v1" || payload.Command != "plan" || payload.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", payload)
	}
	if payload.Plan.Operation != "replace_text" || payload.Plan.Status != "ready" {
		t.Fatalf("unexpected plan: %+v", payload.Plan)
	}
	if len(payload.Plan.Matches) != 1 || payload.Plan.Matches[0].ObjectID != "ppt/slides/slide1.xml#shape-2" {
		t.Fatalf("unexpected plan matches: %+v", payload.Plan.Matches)
	}
}

func TestPlanAmbiguousJSONWritesPayloadAndFails(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{
			{PartName: "ppt/slides/slide1.xml", Text: "Repeat"},
			{PartName: "ppt/slides/slide2.xml", Text: "Repeat"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "edit.json")
	if err := os.WriteFile(specPath, []byte(`{
  "operation": "replace_text",
  "target": {
    "type": "visible_text",
    "text": "Repeat"
  },
  "replacement": "Updated"
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute(context.Background(), []string{"plan", deckPath, "--edit", specPath, "--json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("ambiguous plan unexpectedly succeeded")
	}

	var payload struct {
		Status string `json:"status"`
		Plan   struct {
			Status  string        `json:"status"`
			Matches []interface{} `json:"matches"`
		} `json:"plan"`
		Ambiguous []interface{} `json:"ambiguous"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &payload); jsonErr != nil {
		t.Fatalf("invalid json: %v\n%s", jsonErr, stdout.String())
	}
	if payload.Status != "ambiguous" || payload.Plan.Status != "ambiguous" {
		t.Fatalf("unexpected ambiguous payload: %+v", payload)
	}
	if len(payload.Plan.Matches) != 2 || len(payload.Ambiguous) != 1 {
		t.Fatalf("unexpected ambiguous details: %+v", payload)
	}
}

func TestPlanUnsupportedJSONWritesPayloadAndFails(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	specPath := filepath.Join(dir, "edit.json")
	if err := os.WriteFile(specPath, []byte(`{
  "operation": "unknown_operation",
  "target": {
    "type": "object_id",
    "object_id": "ppt/slides/slide1.xml#shape-2"
  }
}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute(context.Background(), []string{"plan", deckPath, "--edit", specPath, "--json"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("unsupported plan unexpectedly succeeded")
	}

	var payload struct {
		Status      string        `json:"status"`
		Unsupported []interface{} `json:"unsupported"`
	}
	if jsonErr := json.Unmarshal(stdout.Bytes(), &payload); jsonErr != nil {
		t.Fatalf("invalid json: %v\n%s", jsonErr, stdout.String())
	}
	if payload.Status != "unsupported" || len(payload.Unsupported) != 1 {
		t.Fatalf("unexpected unsupported payload: %+v", payload)
	}
}
