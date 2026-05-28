package fixtures

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
)

func TestWriteMinimalPPTXCreatesReadableDeck(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := WriteMinimalPPTX(filePath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}
	pkg, err := pptx.Open(context.Background(), filePath)
	if err != nil {
		t.Fatalf("open fixture failed: %v", err)
	}
	if len(pkg.SlideParts) != 1 || pkg.SlideParts[0] != "ppt/slides/slide1.xml" {
		t.Fatalf("unexpected slide parts: %+v", pkg.SlideParts)
	}
}
