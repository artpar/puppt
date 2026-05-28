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
	for _, want := range []string{"inspect", "plan", "edit", "create", "validate", "review", "version"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help wrote stderr: %s", stderr.String())
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

func TestStubCommandFailsExplicitly(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Execute(context.Background(), []string{"edit"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("edit unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "not implemented yet") {
		t.Fatalf("unexpected error: %v", err)
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
