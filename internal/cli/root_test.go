package cli

import (
	"bytes"
	"context"
	"encoding/json"
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

	err := Execute(context.Background(), []string{"plan"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("plan unexpectedly succeeded")
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
