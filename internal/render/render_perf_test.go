package render

import (
	"context"
	"path/filepath"
	"testing"
)

func BenchmarkRenderEPASlide2DPI144(b *testing.B) {
	input := filepath.Join("..", "..", "testdata", "realworld-ppts", "EPA-generate-2021-presentation.pptx")
	output := filepath.Join(b.TempDir(), "slide2.png")
	options := Options{SlideNumber: 2, DPI: 144, OutputPath: output}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Render(context.Background(), input, options); err != nil {
			b.Fatalf("render failed: %v", err)
		}
	}
}
