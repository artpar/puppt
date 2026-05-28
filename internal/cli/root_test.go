package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
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

	err := Execute(context.Background(), []string{"inspect"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("inspect unexpectedly succeeded")
	}
	if !strings.Contains(err.Error(), "not implemented yet") {
		t.Fatalf("unexpected error: %v", err)
	}
}
