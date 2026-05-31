package render

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
	"github.com/artpar/puppt/internal/pptx"
)

func TestRenderWritesPNGAndReportsUnsupportedTextShape(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "slide-1.png")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	result, err := Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: outputPath})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if result.Status != "partial" {
		t.Fatalf("expected partial render while text is unsupported, got %q", result.Status)
	}
	if result.Render == nil || result.Render.Width != 960 || result.Render.Height != 540 {
		t.Fatalf("unexpected render metadata: %+v", result.Render)
	}
	if len(result.Unsupported) != 1 {
		t.Fatalf("expected one unsupported text shape, got %+v", result.Unsupported)
	}
	img := decodePNG(t, outputPath)
	if got := img.Bounds().Dx(); got != 960 {
		t.Fatalf("unexpected rendered width: %d", got)
	}
	if got := img.Bounds().Dy(); got != 540 {
		t.Fatalf("unexpected rendered height: %d", got)
	}
	xDensity, yDensity, ok := pngPhysicalPixelsPerMeter(t, outputPath)
	if !ok {
		t.Fatal("rendered PNG did not include pHYs density metadata")
	}
	if xDensity != 2835 || yDensity != 2835 {
		t.Fatalf("unexpected default PNG density: got=%dx%d pixels/meter", xDensity, yDensity)
	}
	if got := pngChunkData(t, outputPath, "cICP"); !bytes.Equal(got, displayP3CICPChunkData) {
		t.Fatalf("rendered PNG did not declare Display P3 cICP metadata: %v", got)
	}
}

func TestRenderHonorsOutputDPI(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	outputPath := filepath.Join(dir, "slide-1.png")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	result, err := Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: outputPath, DPI: 96})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if result.Render == nil || result.Render.Width != 1280 || result.Render.Height != 720 {
		t.Fatalf("unexpected 96-DPI render metadata: %+v", result.Render)
	}
	img := decodePNG(t, outputPath)
	if img.Bounds().Dx() != 1280 || img.Bounds().Dy() != 720 {
		t.Fatalf("unexpected 96-DPI PNG bounds: %v", img.Bounds())
	}
	xDensity, yDensity, ok := pngPhysicalPixelsPerMeter(t, outputPath)
	if !ok {
		t.Fatal("96-DPI rendered PNG did not include pHYs density metadata")
	}
	if xDensity != 3780 || yDensity != 3780 {
		t.Fatalf("unexpected 96-DPI PNG density: got=%dx%d pixels/meter", xDensity, yDensity)
	}
}

func TestRenderRejectsOutOfRangeSlide(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	_, err := Render(context.Background(), deckPath, Options{SlideNumber: 2, OutputPath: filepath.Join(dir, "slide-2.png")})
	if err == nil {
		t.Fatal("out-of-range slide unexpectedly rendered")
	}
}

func TestRenderPaintsEmbeddedPNGPicture(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "picture.pptx")
	outputPath := filepath.Join(dir, "slide-1.png")
	if err := writePicturePPTX(deckPath); err != nil {
		t.Fatal(err)
	}

	result, err := Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: outputPath})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected supported picture render, got status=%s unsupported=%+v", result.Status, result.Unsupported)
	}
	img := decodePNG(t, outputPath)
	got := color.RGBAModel.Convert(img.At(10, 10)).(color.RGBA)
	want := color.RGBA{R: 234, G: 51, B: 35, A: 255}
	if got != want {
		t.Fatalf("expected Display P3 transformed red rendered picture pixel, got %#v want %#v", got, want)
	}
}

func TestRenderElementsPaintsShapeLevelBlipFill(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 2, 2))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, source); err != nil {
		t.Fatal(err)
	}
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rIdImage" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/></Relationships>`),
			"ppt/media/image1.png":             imageData.Bytes(),
		},
		ContentTypes: pptx.ContentTypes{Defaults: map[string]string{"png": "image/png"}},
	}
	slideXML := []byte(`<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
<p:cSld><p:spTree>
<p:sp>
  <p:nvSpPr><p:cNvPr id="2" name="Shape Image Fill"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
    <a:blipFill><a:blip r:embed="rIdImage"/><a:stretch><a:fillRect/></a:stretch></a:blipFill>
  </p:spPr>
</p:sp>
</p:spTree></p:cSld>
</p:sld>`)
	elements := collectSlideElements(slideXML)
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	unsupported := renderElements(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, elements, tableStyleSet{})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported shape blip fill render, got %+v", unsupported)
	}
	if got := img.RGBAAt(48, 48); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("expected shape-level blip fill to paint image pixels, got %#v", got)
	}
}

func TestDisplayP3OutputTransformMatchesColorSyncReferenceColors(t *testing.T) {
	cases := []struct {
		name string
		in   color.RGBA
		want color.RGBA
	}{
		{name: "office blue", in: color.RGBA{R: 0x00, G: 0x70, B: 0xC0, A: 0xFF}, want: color.RGBA{R: 0x2F, G: 0x6E, B: 0xBA, A: 0xFF}},
		{name: "dark green", in: color.RGBA{R: 0x12, G: 0x58, B: 0x3B, A: 0xFF}, want: color.RGBA{R: 0x29, G: 0x57, B: 0x3D, A: 0xFF}},
		{name: "office red", in: color.RGBA{R: 0xC0, G: 0x00, B: 0x00, A: 0xFF}, want: color.RGBA{R: 0xB0, G: 0x24, B: 0x18, A: 0xFF}},
		{name: "cyan", in: color.RGBA{R: 0x00, G: 0x9C, B: 0xDE, A: 0xFF}, want: color.RGBA{R: 0x45, G: 0x9A, B: 0xD8, A: 0xFF}},
		{name: "white point", in: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x80}, want: color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x80}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, 1, 1))
			img.SetRGBA(0, 0, tc.in)
			applyDisplayP3OutputTransform(img)
			if got := img.RGBAAt(0, 0); got != tc.want {
				t.Fatalf("Display P3 output transform mismatch: got %#v want %#v", got, tc.want)
			}
		})
	}
}
