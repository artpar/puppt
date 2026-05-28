package main

import (
	"context"
	"testing"

	"github.com/artpar/puppt/internal/cli"
)

func TestCommandPackageWiresCLI(t *testing.T) {
	if err := cli.Execute(context.Background(), []string{"version"}, testWriter{}, testWriter{}); err != nil {
		t.Fatalf("version failed: %v", err)
	}
}

type testWriter struct{}

func (testWriter) Write(data []byte) (int, error) {
	return len(data), nil
}
