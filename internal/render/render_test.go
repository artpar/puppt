package render

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
	"github.com/artpar/puppt/internal/model"
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

func TestRenderShapePaintsSolidRectangleFill(t *testing.T) {
	size := slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		ID:           "2",
		Name:         "Rectangle 1",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{G: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected shape render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(10, 10).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Fatalf("expected green rectangle pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsRoundSingleCornerFill(t *testing.T) {
	size := slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		ID:           "2",
		Name:         "Rounded Rectangle 1",
		PrstGeom:     "round1Rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{G: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected round single-corner render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(0, 0); got != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("expected square top-left corner to be filled, got %+v", got)
	}
	if got := img.RGBAAt(127, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("expected rounded top-right corner to stay outside fill, got %+v", got)
	}
	if got := roundRectCoverage(118, 3, image.Rect(0, 0, 128, 128), roundRectRadius(image.Rect(0, 0, 128, 128), nil), roundedCorners{TopRight: true}); got <= 0 || got >= 4 {
		t.Fatalf("expected round1Rect edge coverage to antialias the curved corner, got %d", got)
	}
}

func TestRenderShapeAlphaBlendsSolidRectangleFill(t *testing.T) {
	size := slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		Name:         "Rectangle 1",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{G: 128, B: 255, A: 128},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected alpha shape render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	got := img.RGBAAt(10, 10)
	if got.A != 255 || got.R >= 255 || got.G <= 128 || got.B != 255 {
		t.Fatalf("expected alpha rectangle fill to composite over white, got %#v", got)
	}
}

func TestBlendPixelRoundsSourceOverChannels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	blendPixel(img, 0, 0, color.RGBA{R: 1, A: 128})
	if got := img.RGBAAt(0, 0); got != (color.RGBA{R: 1, A: 128}) {
		t.Fatalf("expected rounded source-over blend for low-intensity translucent pixel, got %#v", got)
	}
}

func TestCoverageAlphaRoundsPartialCoverage(t *testing.T) {
	if got := coverageAlpha(1, 2); got != 1 {
		t.Fatalf("expected partial coverage alpha to round instead of truncate, got %d", got)
	}
	if got := coverageAlpha(255, 4); got != 255 {
		t.Fatalf("expected full coverage alpha to be preserved, got %d", got)
	}
}

func TestRenderShapeReportsTextAsPartialAfterFill(t *testing.T) {
	size := slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	img := image.NewRGBA(image.Rect(0, 0, 1280, 720))
	element := slideElement{
		Kind:         "sp",
		Name:         "Rectangle 1",
		Text:         "Label",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{B: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported text after fill render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestRenderShapePaintsConnectorLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Line 1",
		PrstGeom:     "straightConnector1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected line render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red line pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsRoundLineCap(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Line 1",
		PrstGeom:     "straightConnector1",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 2,
		ExtCX:        emuPerInch / 2,
		HasLine:      true,
		LineColor:    color.RGBA{G: 255, A: 255},
		LineWidth:    114300,
		LineCap:      "rnd",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected round-cap line render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(20, 44).RGBA()
	if r != 0 || g == 0 || b != 0 || a == 0 {
		t.Fatalf("expected green round-cap endpoint pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapeHonorsFlatLineCap(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Line 1",
		PrstGeom:     "straightConnector1",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 2,
		ExtCX:        emuPerInch / 2,
		HasLine:      true,
		LineColor:    color.RGBA{G: 255, A: 255},
		LineWidth:    114300,
		LineCap:      "flat",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected flat-cap line render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(20, 44).RGBA(); a != 0 {
		t.Fatalf("expected flat cap to leave endpoint-adjacent pixel transparent, got alpha=%04x", a)
	}
}

func TestRenderShapeCompositesTransparentConnectorLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		Name:         "Line 1",
		PrstGeom:     "straightConnector1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 128},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected transparent line render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	got := img.RGBAAt(48, 48)
	if got.A != 255 || got.R != 255 || got.G >= 255 || got.B >= 255 {
		t.Fatalf("expected transparent connector line to composite over white, got %#v", got)
	}
}

func TestRenderShapePaintsDashedRectOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Dashed Rectangle 1",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    9525,
		LineDash:     "dash",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected dashed outline render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(2, 0).RGBA()
	if r != 0 || g != 0 || b != 0xffff || a != 0xffff {
		t.Fatalf("expected blue dash pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	_, _, _, a = img.At(5, 0).RGBA()
	if a != 0 {
		t.Fatalf("expected dashed outline gap to stay transparent, got alpha=%04x", a)
	}
}

func TestRenderShapePaintsZeroHeightConnectorLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "cxnSp",
		Name:           "Straight Arrow Connector 1",
		PrstGeom:       "straightConnector1",
		HasTransform:   true,
		OffY:           emuPerInch / 2,
		ExtCX:          emuPerInch,
		ExtCY:          0,
		HasLine:        true,
		LineColor:      color.RGBA{R: 255, A: 255},
		LineWidth:      9525,
		HasLineMarker:  true,
		TailLineMarker: "triangle",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected zero-height connector render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red horizontal connector pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	r, g, b, a = img.At(89, 49).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red connector arrowhead pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsZeroWidthConnectorLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "cxnSp",
		Name:         "Straight Connector 1",
		PrstGeom:     "line",
		HasTransform: true,
		ExtCX:        0,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{G: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected zero-width connector render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(0, 48).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Fatalf("expected green vertical connector pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestLineEndpointsForElementHonorsFlip(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	element := slideElement{
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		FlipH:        true,
		FlipV:        true,
	}
	startX, startY, endX, endY := lineEndpointsForElement(element, size, image.Rect(0, 0, 96, 96))
	if startX != 96 || startY != 96 || endX != 0 || endY != 0 {
		t.Fatalf("unexpected flipped line endpoints: start=(%d,%d) end=(%d,%d)", startX, startY, endX, endY)
	}
}

func TestRenderShapeReportsUnsupportedLineMarkerType(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "cxnSp",
		Name:           "Oval Marker Connector 1",
		PrstGeom:       "straightConnector1",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		HasLine:        true,
		LineColor:      color.RGBA{R: 255, A: 255},
		LineWidth:      9525,
		HasLineMarker:  true,
		TailLineMarker: "oval",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "line markers") || !element.Rendered {
		t.Fatalf("expected partial unsupported line marker, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestDrawLineTriangleMarkerHonorsLargeSize(t *testing.T) {
	medium := image.NewRGBA(image.Rect(0, 0, 64, 64))
	large := image.NewRGBA(image.Rect(0, 0, 64, 64))
	red := color.RGBA{R: 255, A: 255}

	drawLineTriangleMarker(medium, 40, 20, 20, 0, red, 4, "", "")
	drawLineTriangleMarker(large, 40, 20, 20, 0, red, 4, "lg", "lg")

	if _, _, _, a := medium.At(21, 29).RGBA(); a != 0 {
		t.Fatalf("expected medium marker to leave large-only edge transparent, got alpha=%04x", a)
	}
	if _, _, _, a := large.At(21, 29).RGBA(); a == 0 {
		t.Fatal("expected large marker to paint large-only edge")
	}
}

func TestRenderShapePaintsTriangleFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Triangle 1",
		PrstGeom:     "triangle",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected triangle render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red triangle center pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapeRotatesTriangleFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Rotated Triangle 1",
		PrstGeom:     "triangle",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasRotation:  true,
		Rotation:     10800000,
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected rotated triangle render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(4, 92).RGBA(); a != 0 {
		t.Fatalf("expected rotated triangle to leave bottom corner transparent, got alpha=%04x", a)
	}
	r, g, b, a := img.At(48, 92).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red rotated triangle bottom tip, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestPresetPolygonPointsUsesTriangleAdjustment(t *testing.T) {
	points, ok := presetPolygonPointsForElement(slideElement{
		PrstGeom:            "triangle",
		PrstGeomAdjustments: map[string]int64{"adj": 100000},
	})
	if !ok || len(points) != 3 {
		t.Fatalf("expected adjusted triangle points, got %+v ok=%v", points, ok)
	}
	if points[0].X != 1 {
		t.Fatalf("expected adjusted triangle top point at right edge, got %+v", points)
	}
}

func TestRenderShapePaintsEllipseFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Oval 1",
		PrstGeom:     "ellipse",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{G: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected ellipse render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Fatalf("expected green ellipse center pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsEllipseOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Oval 1",
		PrstGeom:     "ellipse",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected ellipse outline render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 0).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red ellipse outline top pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsRightArrowFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Right Arrow 1",
		PrstGeom:     "rightArrow",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{B: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected right-arrow render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(80, 48).RGBA()
	if r != 0 || g != 0 || b != 0xffff || a != 0xffff {
		t.Fatalf("expected blue right-arrow head pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapePaintsPresetPolygonOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Chevron 1",
		PrstGeom:     "chevron",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{B: 255, A: 255},
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected chevron render with outline result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(12, 0).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red chevron outline pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestPresetPolygonPointsUseDrawingMLAspectRatioGuides(t *testing.T) {
	chevron, ok := presetPolygonPointsForElement(slideElement{
		PrstGeom: "chevron",
		ExtCX:    4 * emuPerInch,
		ExtCY:    emuPerInch,
	})
	if !ok {
		t.Fatal("expected chevron preset points")
	}
	if !nearlyEqual(chevron[1].X, 0.875) || !nearlyEqual(chevron[5].X, 0.125) {
		t.Fatalf("wide chevron should use ss-based DrawingML guides, got %+v", chevron)
	}

	rightArrow, ok := presetPolygonPointsForElement(slideElement{
		PrstGeom: "rightArrow",
		ExtCX:    4 * emuPerInch,
		ExtCY:    emuPerInch,
	})
	if !ok {
		t.Fatal("expected rightArrow preset points")
	}
	if !nearlyEqual(rightArrow[0].Y, 0.25) || !nearlyEqual(rightArrow[5].Y, 0.75) || !nearlyEqual(rightArrow[1].X, 0.875) {
		t.Fatalf("wide rightArrow should use DrawingML guide formulas, got %+v", rightArrow)
	}

	arrow, ok := presetPolygonPointsForElement(slideElement{
		PrstGeom: "notchedRightArrow",
		ExtCX:    4 * emuPerInch,
		ExtCY:    emuPerInch,
	})
	if !ok {
		t.Fatal("expected notchedRightArrow preset points")
	}
	if !nearlyEqual(arrow[0].Y, 0.25) || !nearlyEqual(arrow[6].Y, 0.75) || !nearlyEqual(arrow[1].X, 0.875) || !nearlyEqual(arrow[7].X, 0.0625) {
		t.Fatalf("wide notchedRightArrow should use DrawingML guide formulas, got %+v", arrow)
	}
}

func nearlyEqual(a float64, b float64) bool {
	return math.Abs(a-b) < 0.000001
}

func TestRenderShapePaintsOuterShadow(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:            "sp",
		Name:            "Shadowed Rectangle 1",
		PrstGeom:        "rect",
		HasTransform:    true,
		ExtCX:           emuPerInch / 2,
		ExtCY:           emuPerInch / 2,
		HasFill:         true,
		FillColor:       color.RGBA{R: 255, A: 255},
		HasShadow:       true,
		ShadowColor:     color.RGBA{A: 128},
		ShadowDistance:  91440,
		ShadowDirection: 0,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported outer shadow render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(55, 20).RGBA()
	if !(r < 0xffff && g < 0xffff && b < 0xffff && a == 0xffff) {
		t.Fatalf("expected gray blended shadow pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestDrawShapeShadowKeepsSourceGeometryWhenClipped(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		PrstGeom:    "triangle",
		ShadowColor: color.RGBA{A: 255},
	}

	if !drawShapeShadow(img, image.Rect(-10, 0, 10, 20), element, slideSize{CX: emuPerInch, CY: emuPerInch}) {
		t.Fatal("expected clipped shadow to intersect the canvas")
	}
	if got := img.RGBAAt(0, 5); got != (color.RGBA{A: 255}) {
		t.Fatalf("expected original off-canvas triangle geometry to paint visible center edge, got %+v", got)
	}
	if got := img.RGBAAt(5, 1); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("expected clipped bounds not to rescale triangle geometry, got %+v", got)
	}
}

func TestShadowBlurPixelsScalesDrawingMLBlurRadius(t *testing.T) {
	element := slideElement{ShadowBlur: emuPerInch}
	got := shadowBlurPixels(element, slideSize{CX: emuPerInch, CY: emuPerInch}, 96)
	if got != 96 {
		t.Fatalf("expected DrawingML shadow blur radius to scale to output pixels, got %d", got)
	}
}

func TestRenderShapeReportsUnsupportedShadowGeometry(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:            "sp",
		Name:            "Unsupported Shadow",
		PrstGeom:        "rightBrace",
		HasTransform:    true,
		ExtCX:           emuPerInch / 2,
		ExtCY:           emuPerInch / 2,
		HasShadow:       true,
		ShadowColor:     color.RGBA{A: 128},
		ShadowDistance:  91440,
		ShadowDirection: 0,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "outer shadow geometry") {
		t.Fatalf("expected unsupported shadow geometry report, got unsupported=%+v", unsupported)
	}
}

func TestRenderShapePaintsCurvedArrowPresetGeometry(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Curved Arrow 1",
		PrstGeom:     "curvedDownArrow",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected curved-arrow render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected curved-arrow preset geometry to paint pixels")
	}
	paths := curvedArrowPresetFillPaths(element)
	if len(paths) != 2 {
		t.Fatalf("expected curved-arrow preset to render the main and shaded fill paths, got %+v", paths)
	}
	if minPathY(paths[0]) > 0.01 || maxPathY(paths[0]) < 0.99 {
		t.Fatalf("expected curved-arrow main path to span the preset arc extents, got %+v", paths[0])
	}
}

func TestRenderShapePaintsCurvedArrowPresetOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Curved Arrow Outline 1",
		PrstGeom:     "curvedUpArrow",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected curved-arrow outline result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected curved-arrow preset outline to paint pixels")
	}
	points := curvedArrowPresetOutlinePath(element)
	if len(points) < 8 || minPathY(points) > 0.01 || maxPathY(points) < 0.85 {
		t.Fatalf("expected curved-arrow outline path to span the preset arc extents, got %+v", points)
	}
}

func TestCurvedArrowPresetGeometryStaysInVerticalShapeBounds(t *testing.T) {
	for _, preset := range []string{"curvedDownArrow", "curvedUpArrow"} {
		element := slideElement{
			PrstGeom: preset,
			ExtCX:    4 * emuPerInch,
			ExtCY:    emuPerInch,
		}

		for _, path := range curvedArrowPresetFillPaths(element) {
			if minPathY(path) < -0.2 || maxPathY(path) > 1.2 {
				t.Fatalf("%s fill path escaped far beyond vertical shape bounds: %+v", preset, path)
			}
		}
		outline := curvedArrowPresetOutlinePath(element)
		if minPathY(outline) < -0.2 || maxPathY(outline) > 1.2 {
			t.Fatalf("%s outline path escaped far beyond vertical shape bounds: %+v", preset, outline)
		}
	}
}

func TestDrawCurvedArrowAppliesDarkenLessPathFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		PrstGeom: "curvedDownArrow",
		ExtCX:    emuPerInch,
		ExtCY:    emuPerInch,
	}

	drawCurvedArrow(img, img.Bounds(), element, color.RGBA{R: 200, G: 120, B: 80, A: 255})

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			got := img.RGBAAt(x, y)
			if got.A == 255 && got.R < 200 && got.G < 120 && got.B < 80 {
				return
			}
		}
	}
	t.Fatal("expected darkenLess preset path fill to darken part of the curved arrow")
}

func TestRenderShapePaintsRightBracePresetGeometry(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Right Brace 1",
		PrstGeom:     "rightBrace",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{G: 255, A: 255},
		LineWidth:    9525,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected right-brace render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected right-brace preset geometry to paint pixels")
	}
	points := rightBracePresetPath(element)
	if len(points) < 12 || !nearlyEqual(minPathY(points), 0) || !nearlyEqual(maxPathY(points), 1) || maxPathX(points) < 0.99 {
		t.Fatalf("expected right-brace preset path to follow DrawingML guide extents, got %+v", points)
	}
}

func TestRenderShapeRotatesTextAtRightAngle(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 120, 120))
	element := slideElement{
		Kind:         "sp",
		Name:         "Rotated Label",
		PrstGeom:     "rect",
		HasTransform: true,
		OffX:         35 * emuPerInch / 120,
		OffY:         45 * emuPerInch / 120,
		ExtCX:        70 * emuPerInch / 120,
		ExtCY:        30 * emuPerInch / 120,
		HasRotation:  true,
		Rotation:     16200000,
		Text:         "Axis",
		TextAlign:    "ctr",
		TextAnchor:   "ctr",
		FontSize:     800,
		TextParagraphs: []textParagraph{{
			Text:     "Axis",
			FontSize: 800,
			Runs:     []textRun{{Text: "Axis", FontSize: 800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported right-angle rotated label, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	bounds := opaqueBounds(img)
	if bounds.Empty() || bounds.Dy() <= bounds.Dx() {
		t.Fatalf("expected rotated text bounds to be taller than wide, got %v", bounds)
	}
}

func TestRenderShapePaintsCustomGeometryFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Freeform 1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasFill:      true,
		FillColor:    color.RGBA{B: 255, A: 255},
		CustomPath: []pathPoint{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
			{X: 1, Y: 1},
			{X: 0, Y: 1},
		},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected custom geometry render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0 || g != 0 || b != 0xffff || a != 0xffff {
		t.Fatalf("expected blue custom geometry pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapeFlipsCustomGeometryFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Flipped Freeform 1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		FlipH:        true,
		HasFill:      true,
		FillColor:    color.RGBA{B: 255, A: 255},
		CustomPath: []pathPoint{
			{X: 0, Y: 0},
			{X: 0.25, Y: 0},
			{X: 0.25, Y: 1},
			{X: 0, Y: 1},
		},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected flipped custom geometry render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(8, 48).RGBA(); a != 0 {
		t.Fatalf("expected original left strip to be transparent after flip, got alpha=%04x", a)
	}
	if _, _, _, a := img.At(88, 48).RGBA(); a == 0 {
		t.Fatal("expected flipped custom geometry to paint right strip")
	}
}

func TestCollectSlideElementsIncludesConnectorShapes(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:cxnSp>
        <p:nvCxnSpPr><p:cNvPr id="7" name="Connector 6"/></p:nvCxnSpPr>
        <p:spPr>
          <a:xfrm flipH="1" flipV="1"><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="straightConnector1"><a:avLst/></a:prstGeom>
          <a:ln w="9525"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:tailEnd type="triangle" w="lg" len="lg"/></a:ln>
        </p:spPr>
      </p:cxnSp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one connector element, got %+v", elements)
	}
	got := elements[0]
	if got.Kind != "cxnSp" || got.Name != "Connector 6" || got.PrstGeom != "straightConnector1" || !got.HasLine {
		t.Fatalf("unexpected connector element: %+v", got)
	}
	if !got.HasLineMarker {
		t.Fatalf("expected connector line marker to be parsed: %+v", got)
	}
	if got.TailLineMarkerWidth != "lg" || got.TailLineMarkerLength != "lg" {
		t.Fatalf("expected connector line marker size to be parsed: %+v", got)
	}
	if !got.FlipH || !got.FlipV {
		t.Fatalf("expected connector flips to be parsed: %+v", got)
	}
}

func TestCollectSlideElementsParsesShapeBlackWhiteMode(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:pic>
        <p:nvPicPr><p:cNvPr id="5" name="Gray Picture"/></p:nvPicPr>
        <p:blipFill><a:blip r:embed="rId1" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"/></p:blipFill>
        <p:spPr bwMode="gray"><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm></p:spPr>
      </p:pic>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one element, got %+v", elements)
	}
	if elements[0].BWMode != "gray" {
		t.Fatalf("expected DrawingML bwMode to be parsed, got %+v", elements[0])
	}
}

func TestCollectSlideElementsParsesLineDash(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="9" name="Dashed Box"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
	          <a:ln w="9525" cap="rnd"><a:solidFill><a:srgbClr val="0000FF"/></a:solidFill><a:prstDash val="dash"/></a:ln>
        </p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	if elements[0].LineDash != "dash" {
		t.Fatalf("expected line dash to be parsed, got %+v", elements[0])
	}
	if elements[0].LineCap != "rnd" {
		t.Fatalf("expected line cap to be parsed, got %+v", elements[0])
	}
}

func TestCollectSlideElementsParsesSoftEdge(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="r">
  <p:cSld>
    <p:spTree>
      <p:pic>
        <p:nvPicPr><p:cNvPr id="11" name="Soft Picture"/></p:nvPicPr>
        <p:blipFill><a:blip r:embed="rId1"/></p:blipFill>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:effectLst><a:softEdge rad="203200"/></a:effectLst>
        </p:spPr>
      </p:pic>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one picture element, got %+v", elements)
	}
	if !elements[0].HasSoftEdge || elements[0].SoftEdgeRadius != 203200 {
		t.Fatalf("expected soft edge to be parsed, got %+v", elements[0])
	}
}

func TestCollectSlideElementsParsesPresetGeometryAdjustments(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="12" name="Adjusted Triangle"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="triangle"><a:avLst><a:gd name="adj" fmla="val 100000"/></a:avLst></a:prstGeom>
        </p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	if elements[0].PrstGeomAdjustments["adj"] != 100000 {
		t.Fatalf("expected preset adjustment to be parsed, got %+v", elements[0])
	}
}

func TestParseCustomGeometryPath(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="50"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:lnTo><a:pt x="100" y="0"/></a:lnTo><a:lnTo><a:pt x="100" y="50"/></a:lnTo></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}
	points := parseCustomGeometryPath(root)
	if len(points) != 3 || points[1].X != 1 || points[2].Y != 1 {
		t.Fatalf("unexpected custom geometry points: %+v", points)
	}
}

func TestParseCustomGeometryPathApproximatesCubicBezier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:cubicBezTo><a:pt x="50" y="0"/><a:pt x="100" y="50"/><a:pt x="100" y="100"/></a:cubicBezTo><a:lnTo><a:pt x="0" y="100"/></a:lnTo></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}
	points := parseCustomGeometryPath(root)
	if len(points) != customBezierSegments+2 {
		t.Fatalf("expected cubic path to be expanded into polyline points, got %+v", points)
	}
	lastCurve := points[customBezierSegments]
	if math.Abs(lastCurve.X-1) > 0.001 || math.Abs(lastCurve.Y-1) > 0.001 {
		t.Fatalf("unexpected cubic endpoint: %+v", lastCurve)
	}
}

func TestParseCustomGeometryPathReportsUnsupportedCommands(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:lnTo><a:pt x="100" y="0"/></a:lnTo><a:arcTo wR="10" hR="10" stAng="0" swAng="5400000"/><a:lnTo><a:pt x="0" y="100"/></a:lnTo><a:close/></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}

	points, unsupported := parseCustomGeometryPathWithDiagnostics(root)
	if len(points) < 3 {
		t.Fatalf("expected supported path segments to be retained, got %+v", points)
	}
	if !slices.Contains(unsupported, "custom geometry uses unsupported arcTo command") {
		t.Fatalf("expected unsupported arcTo diagnostic, got %+v", unsupported)
	}
}

func TestRenderGraphicFramePaintsTextAsPartial(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Table 1",
		Text:         "Cell",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		FontSize:     1200,
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !element.Rendered {
		t.Fatalf("unexpected graphic frame render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected graphic frame text to paint non-transparent pixels")
	}
}

func TestParseGraphicFrameReadsTableGrid(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:graphicFrame xmlns:p="p" xmlns:a="a">
		<p:nvGraphicFramePr><p:cNvPr id="4" name="Table 1"/></p:nvGraphicFramePr>
		<p:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm>
		<a:graphic><a:graphicData><a:tbl>
			<a:tblPr firstRow="1" firstCol="1"><a:tableStyleId>{STYLE-READ}</a:tableStyleId></a:tblPr>
			<a:tblGrid><a:gridCol w="300000"/><a:gridCol w="600000"/></a:tblGrid>
			<a:tr h="200000">
				<a:tc gridSpan="2"><a:txBody><a:bodyPr/><a:p><a:pPr algn="ctr"/><a:r><a:rPr sz="1800"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:rPr><a:t>Header</a:t></a:r></a:p></a:txBody><a:tcPr marL="91440" marR="182880" marT="45720" marB="91440"><a:solidFill><a:srgbClr val="DDEEFF"/></a:solidFill><a:lnR w="12700"><a:noFill/><a:prstDash val="solid"/></a:lnR><a:lnB w="12700"><a:solidFill><a:srgbClr val="00FF00"/></a:solidFill><a:prstDash val="dash"/></a:lnB></a:tcPr></a:tc>
				<a:tc hMerge="1"><a:txBody><a:bodyPr/><a:p><a:r><a:t>Value</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>
			</a:tr>
		</a:tbl></a:graphicData></a:graphic>
	</p:graphicFrame>`))
	if err != nil {
		t.Fatal(err)
	}

	element := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !element.HasTable {
		t.Fatal("expected table to be detected")
	}
	if len(element.Table.Columns) != 2 || element.Table.Columns[0] != 300000 || element.Table.Columns[1] != 600000 {
		t.Fatalf("unexpected table columns: %+v", element.Table.Columns)
	}
	if !element.Table.FirstRow || !element.Table.FirstCol || element.Table.StyleID != "{STYLE-READ}" {
		t.Fatalf("unexpected table properties: %+v", element.Table)
	}
	if len(element.Table.Rows) != 1 || len(element.Table.Rows[0].Cells) != 2 {
		t.Fatalf("unexpected table rows: %+v", element.Table.Rows)
	}
	cell := element.Table.Rows[0].Cells[0]
	if cell.Text != "Header" || cell.FontSize != 1800 || cell.TextAlign != "ctr" {
		t.Fatalf("unexpected first cell text properties: %+v", cell)
	}
	if cell.ColSpan != 2 || element.Table.Rows[0].Cells[1].ColSpan != 1 || !element.Table.Rows[0].Cells[1].HMerge {
		t.Fatalf("unexpected horizontal merge properties: %+v", element.Table.Rows[0].Cells)
	}
	if len(cell.TextParagraphs) != 1 || len(cell.TextParagraphs[0].Runs) != 1 || !cell.TextParagraphs[0].Runs[0].HasTextColor {
		t.Fatalf("expected first cell run color: %+v", cell.TextParagraphs)
	}
	if !cell.HasFill || cell.FillColor != (color.RGBA{R: 0xdd, G: 0xee, B: 0xff, A: 0xff}) {
		t.Fatalf("unexpected first cell fill: %+v", cell)
	}
	if !cell.HasMargins || cell.MarginLeft != 91440 || cell.MarginRight != 182880 || cell.MarginTop != 45720 || cell.MarginBottom != 91440 {
		t.Fatalf("unexpected first cell margins: %+v", cell)
	}
	if !cell.BorderRight.Specified || !cell.BorderRight.NoLine {
		t.Fatalf("expected right cell border noFill: %+v", cell.BorderRight)
	}
	if !cell.BorderBottom.Specified || !cell.BorderBottom.HasLine || cell.BorderBottom.Color != (color.RGBA{G: 0xff, A: 0xff}) || cell.BorderBottom.Dash != "dash" {
		t.Fatalf("unexpected bottom cell border: %+v", cell.BorderBottom)
	}
}

func TestTableCellTextRectUsesCellMargins(t *testing.T) {
	got := tableCellTextRect(image.Rect(0, 0, 96, 96), tableCell{
		HasMargins:   true,
		MarginLeft:   emuPerInch / 10,
		MarginRight:  emuPerInch / 5,
		MarginTop:    emuPerInch / 20,
		MarginBottom: emuPerInch / 10,
	}, slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 96, 96))
	if got != (image.Rectangle{Min: image.Point{X: 10, Y: 5}, Max: image.Point{X: 77, Y: 86}}) {
		t.Fatalf("unexpected table text rectangle: %v", got)
	}
}

func TestTableCellTextRectUsesDefaultCellMargins(t *testing.T) {
	got := tableCellTextRect(image.Rect(0, 0, 96, 96), tableCell{}, slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 96, 96))
	if got != (image.Rectangle{Min: image.Point{X: 10, Y: 5}, Max: image.Point{X: 86, Y: 91}}) {
		t.Fatalf("unexpected default table text rectangle: %v", got)
	}
}

func TestParseTableCellMarginsKeepsDefaultsForOmittedSides(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tc xmlns:a="a">
		<a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
		<a:tcPr marL="0"/>
	</a:tc>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseTableCell(root, defaultThemeColors())
	if !got.HasMargins {
		t.Fatalf("expected explicit table-cell margins: %+v", got)
	}
	if got.MarginLeft != 0 || got.MarginRight != defaultTableCellHorizontalMarginEMU || got.MarginTop != defaultTableCellVerticalMarginEMU || got.MarginBottom != defaultTableCellVerticalMarginEMU {
		t.Fatalf("omitted table-cell margins should keep DrawingML defaults, got %+v", got)
	}
}

func TestTableCellRectUsesColumnSpan(t *testing.T) {
	got := tableCellRect([]int{0, 40, 96}, []int{0, 48, 96}, 0, 0, tableCell{ColSpan: 2, RowSpan: 1})
	if got != (image.Rectangle{Min: image.Point{X: 0, Y: 0}, Max: image.Point{X: 96, Y: 48}}) {
		t.Fatalf("unexpected colspan table cell rectangle: %v", got)
	}
}

func TestTableCellTextAnchorDoesNotInferRowSpanCentering(t *testing.T) {
	if got := tableCellTextAnchor(tableCell{RowSpan: 4}); got != "" {
		t.Fatalf("row-spanned table cell without explicit anchor should stay unanchored, got %q", got)
	}
	if got := tableCellTextAnchor(tableCell{RowSpan: 4, TextAnchor: "ctr"}); got != "ctr" {
		t.Fatalf("explicit row-spanned table cell anchor was not preserved, got %q", got)
	}
}

func TestRenderGraphicFramePaintsSupportedTableGrid(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Table 1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1, 1},
			Rows: []tableRow{
				{
					Height: 1,
					Cells: []tableCell{
						{Text: "A", FontSize: 1200, HasFill: true, FillColor: color.RGBA{R: 0x20, G: 0x80, B: 0xe0, A: 0xff}},
						{Text: "B", FontSize: 1200},
					},
				},
				{
					Height: 1,
					Cells: []tableCell{
						{Text: "C", FontSize: 1200},
						{Text: "D", FontSize: 1200},
					},
				},
			},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(30, 36).RGBA()
	if r != 0x2020 || g != 0x8080 || b != 0xe0e0 || a != 0xffff {
		t.Fatalf("expected blue first cell fill, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	_, _, _, borderAlpha := img.At(48, 8).RGBA()
	if borderAlpha != 0xffff {
		t.Fatal("expected table grid border to be painted")
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected table rendering to paint non-transparent pixels")
	}
}

func TestRenderGraphicFrameDrawsSharedTableBordersOnce(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Table 1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1, 1},
			Rows: []tableRow{{
				Height: 1,
				Cells:  []tableCell{{}, {}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(48, 40).RGBA(); a != 0xffff {
		t.Fatalf("expected shared table border to be painted on grid line, got alpha=%04x", a)
	}
	if _, _, _, a := img.At(47, 40).RGBA(); a != 0 {
		t.Fatalf("shared table border should not expand left of grid line, got alpha=%04x", a)
	}
	if _, _, _, a := img.At(49, 40).RGBA(); a != 0 {
		t.Fatalf("shared table border should not expand right of grid line, got alpha=%04x", a)
	}
}

func TestRenderGraphicFrameUsesExplicitTableCellBorders(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Table 1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1},
			Rows: []tableRow{
				{
					Height: 1,
					Cells: []tableCell{{
						RowSpan:      1,
						ColSpan:      1,
						BorderBottom: tableCellBorder{Specified: true, HasLine: true, Color: color.RGBA{A: 255}, Width: 9525},
						BorderRight:  tableCellBorder{Specified: true, NoLine: true, Width: 9525},
					}},
				},
			},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(48, 95).RGBA(); a != 0xffff {
		t.Fatalf("expected explicit bottom border to be painted, got alpha=%04x", a)
	}
	if _, _, _, a := img.At(95, 48).RGBA(); a != 0 {
		t.Fatalf("expected explicit noFill right border to stay transparent, got alpha=%04x", a)
	}
}

func TestTableGridOffsetsScaleUnderfilledExtentsToFrame(t *testing.T) {
	got := tableGridOffsets([]int64{emuPerInch / 4, emuPerInch / 4}, 10, 110, emuPerInch/10, emuPerInch, emuPerInch, 100)
	want := []int{10, 60, 110}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected underfilled table row heights to scale to frame, got %v want %v", got, want)
	}
}

func TestTableRowWeightsPreserveExplicitZeroHeightRows(t *testing.T) {
	table := tableModel{Rows: []tableRow{
		{HasHeight: true, Height: 300},
		{HasHeight: true, Height: 0},
		{Cells: []tableCell{{}}},
	}}

	got := tableRowWeights(table)
	want := []int64{300, 0, 1}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected explicit zero-height table row to stay zero, got %v want %v", got, want)
	}
}

func TestTableGridOffsetsPreserveAuthoredExtentsWhenTheyMatchFrame(t *testing.T) {
	got := tableGridOffsets([]int64{emuPerInch / 4, emuPerInch / 4}, 10, 110, emuPerInch/10, emuPerInch/2, emuPerInch, 100)
	want := []int{10, 35, 60}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected matching authored table row heights to be preserved, got %v want %v", got, want)
	}
}

func TestTableGridOffsetsPreserveAuthoredExtentsFromAbsoluteOrigin(t *testing.T) {
	got := tableGridOffsets([]int64{15, 15}, 2, 5, 15, 30, 100, 10)
	want := []int{2, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected matching authored grid lines to scale from absolute origin, got %v want %v", got, want)
	}
}

func TestTableGridOffsetsScaleOverflowingExtentsToFrame(t *testing.T) {
	got := tableGridOffsets([]int64{emuPerInch, emuPerInch}, 10, 110, emuPerInch/10, emuPerInch, emuPerInch, 100)
	want := []int{10, 60, 110}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected overflowing table row heights to scale to frame, got %v want %v", got, want)
	}
}

func TestTableGridOffsetsScalesFallbackWeightsToFrame(t *testing.T) {
	got := tableGridOffsets([]int64{1, 1}, 10, 110, emuPerInch/10, emuPerInch, emuPerInch, 100)
	want := []int{10, 60, 110}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected fallback table weights to scale to frame, got %v want %v", got, want)
	}
}

func TestParseTableStylesReadsConditionalRegions(t *testing.T) {
	styles := testTableStyleSet(t)
	style, ok := styles.Styles[normalizedTableStyleID("{STYLE-ONE}")]
	if !ok {
		t.Fatalf("expected parsed table style, got %+v", styles)
	}
	if styles.DefaultID != "{STYLE-ONE}" || style.Name != "Generic Style" {
		t.Fatalf("unexpected table style identity: %+v", style)
	}
	whole := style.Regions["wholeTbl"]
	if !whole.HasFill || whole.FillColor != (color.RGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}) {
		t.Fatalf("unexpected whole-table fill: %+v", whole)
	}
	if whole.FontFamily != "Calibri" {
		t.Fatalf("expected whole-table font family from table fontRef, got %+v", whole)
	}
	if !whole.Borders.InsideV.Specified || !whole.Borders.InsideV.HasLine || whole.Borders.InsideV.Color != (color.RGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}) {
		t.Fatalf("unexpected inside vertical border: %+v", whole.Borders.InsideV)
	}
	firstRow := style.Regions["firstRow"]
	if !firstRow.HasBold || !firstRow.Bold || !firstRow.HasTextColor || firstRow.TextColor != (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
		t.Fatalf("unexpected first-row text style: %+v", firstRow)
	}
	if !firstRow.Borders.Bottom.Specified || !firstRow.Borders.Bottom.HasLine || firstRow.Borders.Bottom.Color != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("unexpected first-row bottom border: %+v", firstRow.Borders.Bottom)
	}
}

func TestParseTableStylesReadsDirectTableTextFontAndItalic(t *testing.T) {
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
		<a:tblStyle styleId="{STYLE-DIRECT}" styleName="Direct Font Style">
			<a:wholeTbl>
				<a:tcTxStyle i="on">
					<a:font>
						<a:latin typeface="Aptos"/>
						<a:ea typeface="MS Mincho"/>
						<a:cs typeface="Arial"/>
					</a:font>
					<a:srgbClr val="123456"/>
				</a:tcTxStyle>
			</a:wholeTbl>
		</a:tblStyle>
	</a:tblStyleLst>`), defaultThemeColors(), themeFonts{MinorLatin: "Calibri"}, themeLineStyles{})

	style, ok := styles.Styles[normalizedTableStyleID("{STYLE-DIRECT}")]
	if !ok {
		t.Fatalf("expected parsed direct-font table style, got %+v", styles)
	}
	whole := style.Regions["wholeTbl"]
	if whole.FontFamily != "Aptos" || !whole.HasItalic || !whole.Italic || !tableCellTextItalic(whole) {
		t.Fatalf("unexpected direct table text style: %+v", whole)
	}
	if !whole.HasTextColor || whole.TextColor != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}) {
		t.Fatalf("unexpected direct table text color: %+v", whole)
	}
}

func TestParseTableStylesResolvesThemeLineReferences(t *testing.T) {
	lineStyles := parseThemeLineStyles([]byte(`<a:theme xmlns:a="a">
		<a:themeElements><a:fmtScheme name="Office"><a:lnStyleLst>
			<a:ln w="9525"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
			<a:ln w="25400"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="dash"/></a:ln>
		</a:lnStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
		<a:tblStyle styleId="{STYLE-LINE}" styleName="Theme Line Style">
			<a:wholeTbl>
				<a:tcStyle><a:tcBdr>
					<a:left><a:lnRef idx="2"><a:schemeClr val="accent2"/></a:lnRef></a:left>
					<a:right><a:lnRef idx="0"><a:schemeClr val="accent2"/></a:lnRef></a:right>
				</a:tcBdr></a:tcStyle>
			</a:wholeTbl>
		</a:tblStyle>
	</a:tblStyleLst>`), themeColors{"accent2": {R: 12, G: 34, B: 56, A: 255}}, themeFonts{}, lineStyles)

	style := styles.Styles[normalizedTableStyleID("{STYLE-LINE}")]
	left := style.Regions["wholeTbl"].Borders.Left
	if !left.Specified || !left.HasLine || left.Width != 25400 || left.Dash != "dash" || left.Color != (color.RGBA{R: 12, G: 34, B: 56, A: 255}) {
		t.Fatalf("unexpected resolved lnRef table border: %+v", left)
	}
	right := style.Regions["wholeTbl"].Borders.Right
	if !right.Specified || !right.NoLine {
		t.Fatalf("expected idx=0 line reference to suppress border, got %+v", right)
	}
}

func TestResolvedTableCellStyleAppliesGenericRegionPrecedence(t *testing.T) {
	styles := testTableStyleSet(t)
	table := tableModel{
		Columns:  []int64{1, 1},
		StyleID:  "{STYLE-ONE}",
		FirstRow: true,
		FirstCol: true,
		BandRow:  true,
		Rows: []tableRow{
			{Cells: []tableCell{{}, {}}},
			{Cells: []tableCell{{}, {}}},
			{Cells: []tableCell{{}, {}}},
		},
	}

	header := resolvedTableCellStyle(table, styles, 0, 1)
	if !header.HasFill || header.FillColor != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}) || !tableCellTextBold(header) {
		t.Fatalf("unexpected first-row style: %+v", header)
	}
	firstColumn := resolvedTableCellStyle(table, styles, 1, 0)
	if !firstColumn.HasFill || firstColumn.FillColor != (color.RGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff}) || !tableCellTextBold(firstColumn) {
		t.Fatalf("unexpected first-column style: %+v", firstColumn)
	}
	band1 := resolvedTableCellStyle(table, styles, 1, 1)
	if !band1.HasFill || band1.FillColor != (color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}) {
		t.Fatalf("unexpected band1 style: %+v", band1)
	}
	band2 := resolvedTableCellStyle(table, styles, 2, 1)
	if !band2.HasFill || band2.FillColor != (color.RGBA{R: 0xdd, G: 0xee, B: 0xff, A: 0xff}) {
		t.Fatalf("unexpected band2 style: %+v", band2)
	}
}

func TestTableEdgeBorderUsesInsideBordersForInternalEdges(t *testing.T) {
	right := tableCellBorder{Specified: true, HasLine: true, Color: color.RGBA{R: 200, A: 255}}
	bottom := tableCellBorder{Specified: true, HasLine: true, Color: color.RGBA{G: 200, A: 255}}
	insideV := tableCellBorder{Specified: true, NoLine: true}
	insideH := tableCellBorder{Specified: true, NoLine: true}
	borders := tableStyleBorders{Right: right, Bottom: bottom, InsideV: insideV, InsideH: insideH}

	if got := tableEdgeBorder(borders, tableEdgeRight, 0, 0, 2, 2); !got.NoLine {
		t.Fatalf("expected internal right edge to use insideV, got %+v", got)
	}
	if got := tableEdgeBorder(borders, tableEdgeBottom, 0, 0, 2, 2); !got.NoLine {
		t.Fatalf("expected internal bottom edge to use insideH, got %+v", got)
	}
	if got := tableEdgeBorder(borders, tableEdgeRight, 0, 1, 2, 2); got != right {
		t.Fatalf("expected outside right edge to use right border, got %+v", got)
	}
	if got := tableEdgeBorder(borders, tableEdgeBottom, 1, 0, 2, 2); got != bottom {
		t.Fatalf("expected outside bottom edge to use bottom border, got %+v", got)
	}
}

func TestRenderGraphicFramePaintsParsedTableStyle(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Styled Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns:  []int64{1, 1},
			StyleID:  "{STYLE-ONE}",
			FirstRow: true,
			FirstCol: true,
			BandRow:  true,
			Rows: []tableRow{
				{Height: 1, Cells: []tableCell{{RowSpan: 1}, {RowSpan: 1}}},
				{Height: 1, Cells: []tableCell{{RowSpan: 2}, {RowSpan: 1}}},
				{Height: 1, Cells: []tableCell{{VMerge: true, RowSpan: 1}, {RowSpan: 1}}},
			},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, testTableStyleSet(t))
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected styled table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(72, 12); got != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}) {
		t.Fatalf("expected parsed first-row fill, got %#v", got)
	}
	if got := img.RGBAAt(24, 50); got != (color.RGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff}) {
		t.Fatalf("expected parsed first-column fill, got %#v", got)
	}
	if got := img.RGBAAt(72, 50); got != (color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}) {
		t.Fatalf("expected parsed band fill, got %#v", got)
	}
}

func TestRenderGraphicFrameUsesParsedTableStyleBorders(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Grid Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1},
			StyleID: "{STYLE-ONE}",
			Rows: []tableRow{{
				Height: 1,
				Cells: []tableCell{{
					RowSpan:     1,
					NoFill:      true,
					BorderRight: tableCellBorder{Specified: true, NoLine: true, Width: 12700},
				}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, testTableStyleSet(t))
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected styled border table result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 0); got != (color.RGBA{R: 0xcc, G: 0xcc, B: 0xcc, A: 0xff}) {
		t.Fatalf("expected parsed top grid border, got %#v", got)
	}
	if _, _, _, a := img.At(95, 48).RGBA(); a != 0 {
		t.Fatalf("expected explicit noFill right border to suppress style default, got alpha=%04x", a)
	}
}

func TestTableCellFillDirectNoFillSuppressesStyleFill(t *testing.T) {
	style := tableStyleRegion{HasFill: true, FillColor: color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}}
	if _, ok := tableCellFill(style, tableCell{NoFill: true}); ok {
		t.Fatal("expected direct noFill to suppress table style fill")
	}
}

func TestParseTableModelRecordsUnsupportedVisibleFeatures(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
			<a:tr h="914400"><a:tc gridSpan="2"><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
				<a:tcPr>
					<a:gradFill><a:gsLst><a:gs pos="0"><a:srgbClr val="FFFFFF"/></a:gs><a:gs pos="100000"><a:srgbClr val="000000"/></a:gs></a:gsLst></a:gradFill>
					<a:effectLst><a:outerShdw blurRad="12700" dist="12700" dir="5400000"><a:srgbClr val="000000"/></a:outerShdw></a:effectLst>
					<a:lnB cmpd="dbl" cap="rnd"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:round/><a:tailEnd type="triangle"/></a:lnB>
			</a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	for _, want := range []string{
		"uses a non-solid cell fill that was not rendered",
		"uses effects that were not rendered",
		"uses border line caps that were not rendered",
		"uses compound border lines that were not rendered",
		"uses border line end decorations that were not rendered",
	} {
		if !slices.Contains(table.UnsupportedFeatures, want) {
			t.Fatalf("expected unsupported table feature %q in %+v", want, table.UnsupportedFeatures)
		}
	}
	if slices.Contains(table.UnsupportedFeatures, "uses merged cells rendered with simplified layout") {
		t.Fatalf("merged cells are rendered through table span geometry and should not be reported partial: %+v", table.UnsupportedFeatures)
	}
}

func TestParseTableModelTreatsMergedCellsAsSupported(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
	<a:tblGrid><a:gridCol w="457200"/><a:gridCol w="457200"/></a:tblGrid>
	<a:tr h="457200">
		<a:tc gridSpan="2"><a:txBody><a:bodyPr/><a:p><a:r><a:t>Header</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>
		<a:tc hMerge="1"><a:txBody><a:bodyPr/><a:p><a:r><a:t/></a:r></a:p></a:txBody><a:tcPr/></a:tc>
	</a:tr>
	<a:tr h="457200">
		<a:tc rowSpan="2"><a:txBody><a:bodyPr/><a:p><a:r><a:t>Side</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>
		<a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Value</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>
	</a:tr>
	<a:tr h="457200">
		<a:tc vMerge="1"><a:txBody><a:bodyPr/><a:p><a:r><a:t/></a:r></a:p></a:txBody><a:tcPr/></a:tc>
		<a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Value 2</a:t></a:r></a:p></a:txBody><a:tcPr/></a:tc>
	</a:tr>
</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if len(table.UnsupportedFeatures) != 0 {
		t.Fatalf("expected merged table cells to be supported by span geometry, got %+v", table.UnsupportedFeatures)
	}
	header := table.Rows[0].Cells[0]
	if header.ColSpan != 2 || !table.Rows[0].Cells[1].HMerge {
		t.Fatalf("expected horizontal merge metadata, got %+v", table.Rows[0].Cells)
	}
	side := table.Rows[1].Cells[0]
	if side.RowSpan != 2 || !table.Rows[2].Cells[0].VMerge {
		t.Fatalf("expected vertical merge metadata, got row1=%+v row2=%+v", table.Rows[1].Cells, table.Rows[2].Cells)
	}
}

func TestParseTableCellBorderPreservesLineJoin(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tcPr xmlns:a="a"><a:lnB w="12700"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:round/></a:lnB></a:tcPr>`))
	if err != nil {
		t.Fatal(err)
	}

	border := parseTableCellBorder(root, "lnB", defaultThemeColors())
	if !border.Specified || !border.HasLine || border.Join != "round" {
		t.Fatalf("expected parsed round line join, got %+v", border)
	}
}

func TestRenderGraphicFrameReportsSpecificUnsupportedTableFeatures(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Feature Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns:             []int64{1},
			UnsupportedFeatures: []string{"uses effects that were not rendered"},
			Rows: []tableRow{{
				Height: 1,
				Cells:  []tableCell{{RowSpan: 1, NoFill: true}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "uses effects") {
		t.Fatalf("expected specific table unsupported feature, got %+v", unsupported)
	}
}

func testTableStyleSet(t *testing.T) tableStyleSet {
	t.Helper()
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a" def="{STYLE-ONE}">
		<a:tblStyle styleId="{STYLE-ONE}" styleName="Generic Style">
			<a:wholeTbl>
				<a:tcTxStyle><a:fontRef idx="minor"/><a:srgbClr val="000000"/></a:tcTxStyle>
				<a:tcStyle>
					<a:tcBdr>
						<a:left><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:left>
						<a:right><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:right>
						<a:top><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:top>
						<a:bottom><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:bottom>
						<a:insideH><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:insideH>
						<a:insideV><a:ln w="12700"><a:solidFill><a:srgbClr val="CCCCCC"/></a:solidFill></a:ln></a:insideV>
					</a:tcBdr>
					<a:fill><a:solidFill><a:srgbClr val="EEEEEE"/></a:solidFill></a:fill>
				</a:tcStyle>
			</a:wholeTbl>
			<a:band1H><a:tcStyle><a:fill><a:solidFill><a:srgbClr val="AABBCC"/></a:solidFill></a:fill></a:tcStyle></a:band1H>
			<a:band2H><a:tcStyle><a:fill><a:solidFill><a:srgbClr val="DDEEFF"/></a:solidFill></a:fill></a:tcStyle></a:band2H>
			<a:firstCol>
				<a:tcTxStyle b="on"><a:srgbClr val="FFFFFF"/></a:tcTxStyle>
				<a:tcStyle><a:fill><a:solidFill><a:srgbClr val="445566"/></a:solidFill></a:fill></a:tcStyle>
			</a:firstCol>
			<a:firstRow>
				<a:tcTxStyle b="on"><a:srgbClr val="FFFFFF"/></a:tcTxStyle>
				<a:tcStyle>
					<a:tcBdr><a:bottom><a:ln w="12700"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:ln></a:bottom></a:tcBdr>
					<a:fill><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:fill>
				</a:tcStyle>
			</a:firstRow>
		</a:tblStyle>
	</a:tblStyleLst>`), defaultThemeColors(), themeFonts{MinorLatin: "Calibri"}, themeLineStyles{})
	if len(styles.Styles) == 0 {
		t.Fatal("expected test table style XML to parse")
	}
	return styles
}

func TestTableTextParagraphsWithBoldCopiesParagraphs(t *testing.T) {
	paragraphs := []textParagraph{{
		Text: "Header",
		Runs: []textRun{{Text: "Header"}},
	}}

	got := tableTextParagraphsWithBold(paragraphs, "")
	if len(got) != 1 || !got[0].Bold || len(got[0].Runs) != 1 || !got[0].Runs[0].Bold {
		t.Fatalf("expected bold paragraph copy, got %+v", got)
	}
	if paragraphs[0].Bold || paragraphs[0].Runs[0].Bold {
		t.Fatalf("source paragraphs were mutated: %+v", paragraphs)
	}
}

func TestRenderGraphicFramePaintsSupportedDiagramDrawing(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/diagrams/data1.xml":    []byte(`<dgm:dataModel xmlns:dgm="dgm" xmlns:a="a" xmlns:dsp="dsp"><dgm:extLst><a:ext><dsp:dataModelExt relId="rId2"/></a:ext></dgm:extLst></dgm:dataModel>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a"><dsp:spTree><dsp:sp><dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="notchedRightArrow"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></dsp:spPr><dsp:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr sz="1200"/><a:t>Diagram</a:t></a:r></a:p></dsp:txBody></dsp:sp></dsp:spTree></dsp:drawing>`),
	}}
	element := slideElement{
		Kind:          "graphicFrame",
		Name:          "Diagram 1",
		DiagramDataID: "rId1",
		HasTransform:  true,
		ExtCX:         emuPerInch,
		ExtCY:         emuPerInch,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: diagramDataRelType, Target: "../diagrams/data1.xml"},
		"rId2": {ID: "rId2", Type: diagramDrawingRelType, Target: "../diagrams/drawing1.xml"},
	}

	unsupported := renderGraphicFrame(pkg, "ppt/slides/slide1.xml", size, img, &element, relationships, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported diagram render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(60, 48).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red diagram shape pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderGraphicFrameUsesPackageThemeForDiagramDrawing(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/theme/theme1.xml":      []byte(`<a:theme xmlns:a="a"><a:themeElements><a:clrScheme name="Custom"><a:dk1><a:srgbClr val="000000"/></a:dk1><a:lt1><a:srgbClr val="FFFFFF"/></a:lt1><a:dk2><a:srgbClr val="000000"/></a:dk2><a:lt2><a:srgbClr val="FFFFFF"/></a:lt2><a:accent1><a:srgbClr val="000000"/></a:accent1><a:accent2><a:srgbClr val="000000"/></a:accent2><a:accent3><a:srgbClr val="000000"/></a:accent3><a:accent4><a:srgbClr val="000000"/></a:accent4><a:accent5><a:srgbClr val="112233"/></a:accent5><a:accent6><a:srgbClr val="FF0000"/></a:accent6></a:clrScheme></a:themeElements></a:theme>`),
		"ppt/diagrams/data1.xml":    []byte(`<dgm:dataModel xmlns:dgm="dgm" xmlns:a="a" xmlns:dsp="dsp"><dgm:extLst><a:ext><dsp:dataModelExt relId="rId2"/></a:ext></dgm:extLst></dgm:dataModel>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a"><dsp:spTree><dsp:sp><dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="chevron"><a:avLst><a:gd name="adj" fmla="val 25000"/></a:avLst></a:prstGeom><a:solidFill><a:schemeClr val="accent5"/></a:solidFill><a:ln w="9525"><a:solidFill><a:schemeClr val="accent6"/></a:solidFill></a:ln></dsp:spPr></dsp:sp></dsp:spTree></dsp:drawing>`),
	}}
	element := slideElement{
		Kind:          "graphicFrame",
		Name:          "Diagram 1",
		DiagramDataID: "rId1",
		HasTransform:  true,
		ExtCX:         emuPerInch,
		ExtCY:         emuPerInch,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: diagramDataRelType, Target: "../diagrams/data1.xml"},
		"rId2": {ID: "rId2", Type: diagramDrawingRelType, Target: "../diagrams/drawing1.xml"},
	}

	unsupported := renderGraphicFrame(pkg, "ppt/slides/slide1.xml", size, img, &element, relationships, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported diagram render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0x1111 || g != 0x2222 || b != 0x3333 || a != 0xffff {
		t.Fatalf("expected package-theme diagram fill, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	r, g, b, a = img.At(12, 0).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected package-theme diagram outline, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestDiagramDrawingElementsResolvePackageThemeFonts(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/theme/theme1.xml":      []byte(`<a:theme xmlns:a="a"><a:themeElements><a:fontScheme name="Custom"><a:majorFont><a:latin typeface="Trebuchet MS"/></a:majorFont><a:minorFont><a:latin typeface="Arial"/></a:minorFont></a:fontScheme></a:themeElements></a:theme>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a"><dsp:spTree><dsp:sp><dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"/></dsp:spPr><dsp:style><a:fontRef idx="minor"/></dsp:style><dsp:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>Diagram</a:t></a:r></a:p></dsp:txBody></dsp:sp></dsp:spTree></dsp:drawing>`),
	}}

	got := diagramDrawingElements(pkg, "ppt/diagrams/drawing1.xml")
	if len(got) != 1 {
		t.Fatalf("expected one diagram element, got %d", len(got))
	}
	if got[0].FontFamily != "Arial" {
		t.Fatalf("expected diagram fontRef minor to resolve through package theme fonts, got %+v", got[0])
	}
}

func TestRenderGraphicFrameReportsUnsupportedDiagramContent(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/diagrams/data1.xml":    []byte(`<dgm:dataModel xmlns:dgm="dgm" xmlns:a="a" xmlns:dsp="dsp"><dgm:extLst><a:ext><dsp:dataModelExt relId="rId2"/></a:ext></dgm:extLst></dgm:dataModel>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a" xmlns:r="r"><dsp:spTree><dsp:pic><dsp:nvPicPr><dsp:cNvPr id="1" name="Diagram Picture"/></dsp:nvPicPr><dsp:blipFill><a:blip r:embed="rIdImg"/></dsp:blipFill><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm></dsp:spPr></dsp:pic></dsp:spTree></dsp:drawing>`),
	}}
	element := slideElement{
		Kind:          "graphicFrame",
		Name:          "Diagram 1",
		DiagramDataID: "rId1",
		HasTransform:  true,
		ExtCX:         emuPerInch,
		ExtCY:         emuPerInch,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: diagramDataRelType, Target: "../diagrams/data1.xml"},
		"rId2": {ID: "rId2", Type: diagramDrawingRelType, Target: "../diagrams/drawing1.xml"},
	}

	unsupported := renderGraphicFrame(pkg, "ppt/slides/slide1.xml", size, img, &element, relationships, tableStyleSet{})
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "diagram contains picture content") {
		t.Fatalf("expected unsupported diagram picture content, got %+v", unsupported)
	}
	if element.Rendered {
		t.Fatal("diagram with only unsupported content should not be marked rendered")
	}
}

func TestParseSlideElementNodeReadsDiagramTextTransform(t *testing.T) {
	root, err := parseXMLNode([]byte(`<dsp:sp xmlns:dsp="dsp" xmlns:a="a">
	  <dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr>
	  <dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1000" cy="1000"/></a:xfrm></dsp:spPr>
	  <dsp:txXfrm><a:off x="100" y="200"/><a:ext cx="300" cy="400"/></dsp:txXfrm>
	</dsp:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 2, ScaleY: 3, OffsetX: 10, OffsetY: 20})
	if !got.HasTextTransform {
		t.Fatalf("expected diagram text transform to be parsed: %+v", got)
	}
	if got.TextOffX != 210 || got.TextOffY != 620 || got.TextExtCX != 600 || got.TextExtCY != 1200 {
		t.Fatalf("unexpected transformed diagram text box: %+v", got)
	}
}

func TestFitDiagramElementsToFramePreservesContainedCoordinates(t *testing.T) {
	elements := []slideElement{{
		HasTransform: true,
		OffX:         100,
		OffY:         200,
		ExtCX:        300,
		ExtCY:        400,
	}}
	frame := slideElement{
		HasTransform: true,
		OffX:         1000,
		OffY:         2000,
		ExtCX:        1000,
		ExtCY:        1000,
	}

	got := fitDiagramElementsToFrame(elements, frame)
	if len(got) != 1 {
		t.Fatalf("expected one fitted element, got %d", len(got))
	}
	if got[0].OffX != 1100 || got[0].OffY != 2200 || got[0].ExtCX != 300 || got[0].ExtCY != 400 {
		t.Fatalf("expected contained diagram coordinates to be translated without scaling, got %+v", got[0])
	}
}

func TestFitDiagramElementsToFrameScalesDownOverflowingCoordinates(t *testing.T) {
	elements := []slideElement{{
		HasTransform: true,
		OffX:         100,
		OffY:         200,
		ExtCX:        900,
		ExtCY:        1800,
	}}
	frame := slideElement{
		HasTransform: true,
		OffX:         10,
		OffY:         20,
		ExtCX:        500,
		ExtCY:        1000,
	}

	got := fitDiagramElementsToFrame(elements, frame)
	if len(got) != 1 {
		t.Fatalf("expected one fitted element, got %d", len(got))
	}
	if got[0].OffX != 60 || got[0].OffY != 120 || got[0].ExtCX != 450 || got[0].ExtCY != 900 {
		t.Fatalf("expected overflowing diagram coordinates to be scaled down into frame, got %+v", got[0])
	}
}

func TestFitDiagramElementsToFrameScalesDiagramTextTransform(t *testing.T) {
	elements := []slideElement{{
		HasTransform:     true,
		OffX:             0,
		OffY:             0,
		ExtCX:            1000,
		ExtCY:            1000,
		HasTextTransform: true,
		TextOffX:         1000,
		TextOffY:         500,
		TextExtCX:        1000,
		TextExtCY:        500,
	}}
	frame := slideElement{
		HasTransform: true,
		OffX:         100,
		OffY:         200,
		ExtCX:        1000,
		ExtCY:        1000,
	}

	got := fitDiagramElementsToFrame(elements, frame)
	if got[0].OffX != 100 || got[0].ExtCX != 500 {
		t.Fatalf("expected shape coordinates to scale against diagram text extents, got %+v", got[0])
	}
	if got[0].TextOffX != 600 || got[0].TextOffY != 700 || got[0].TextExtCX != 500 || got[0].TextExtCY != 500 {
		t.Fatalf("expected diagram text transform to scale with frame, got %+v", got[0])
	}
}

func TestTextBoundsUsesDiagramTextTransform(t *testing.T) {
	element := slideElement{
		HasTextTransform: true,
		TextOffX:         emuPerInch / 4,
		TextOffY:         emuPerInch / 2,
		TextExtCX:        emuPerInch / 2,
		TextExtCY:        emuPerInch / 4,
		HasInsets:        true,
	}

	got := textBounds(image.Rect(0, 0, 96, 96), element, slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 96, 96))
	want := image.Rect(24, 48, 72, 72)
	if got != want {
		t.Fatalf("expected diagram text transform to define text bounds: got=%v want=%v", got, want)
	}
}

func TestTextFromNodePreservesParagraphBreaks(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>First</a:t></a:r></a:p><a:p><a:r><a:t>Second</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(textFromNode(root)); got != "First\nSecond" {
		t.Fatalf("unexpected paragraph text: %q", got)
	}
}

func TestTextFromNodePreservesTabs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(textFromNode(root)); got != "Cost\tTotal" {
		t.Fatalf("unexpected tabbed text: %q", got)
	}
	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || len(paragraphs[0].Runs) != 1 || paragraphs[0].Runs[0].Text != "Cost\tTotal" {
		t.Fatalf("expected tab in text run, got %+v", paragraphs)
	}
}

func TestTextParagraphsFromNodeParsesTabStops(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:tabLst><a:tab pos="1074738" algn="l"/></a:tabLst></a:pPr><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || len(paragraphs[0].TabStops) != 1 || paragraphs[0].TabStops[0] != 1074738 {
		t.Fatalf("expected explicit paragraph tab stop, got %+v", paragraphs)
	}
	if stops := tabStopsAtDPI(paragraphs[0].TabStops, 96); len(stops) != 1 || stops[0] != 113 {
		t.Fatalf("expected tab stop to scale to 96 DPI pixels, got %+v", stops)
	}
}

func TestTextParagraphsFromNodeParsesDefaultTabSize(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:lstStyle><a:lvl1pPr defTabSz="457200"/></a:lstStyle>
	  <a:p><a:pPr lvl="0"/><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || !paragraphs[0].HasDefaultTab || paragraphs[0].DefaultTabSize != 457200 {
		t.Fatalf("expected default tab size inherited from list style, got %+v", paragraphs)
	}
	stops := paragraphTabStopsAtDPI(paragraphs[0], 72, 160)
	if len(stops) < 3 || stops[0] != 36 || stops[1] != 72 || stops[2] != 108 {
		t.Fatalf("expected repeating half-inch default tab stops, got %+v", stops)
	}
}

func TestTextParagraphsFromNodeParsesRightMargin(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:pPr marR="914400"/><a:r><a:t>Right margin</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || !paragraphs[0].HasMarginRight || paragraphs[0].MarginRight != 914400 {
		t.Fatalf("expected paragraph right margin, got %+v", paragraphs)
	}
	if got := paragraphRightOffsetAtDPI(paragraphs[0], 96); got != 96 {
		t.Fatalf("expected right margin to scale to 96 DPI pixels, got %d", got)
	}
}

func TestTextParagraphsFromNodeDetectsBulletsAndLevels(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr b="1"/><a:t>Primary energy resources</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"/><a:r><a:t>Fossil</a:t></a:r></a:p>
  <a:p><a:pPr lvl="2"><a:buSzPts val="1400"/><a:buChar char="-"/></a:pPr><a:r><a:t>Coal</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"><a:buFont typeface="Wingdings"/><a:buChar char="§"/></a:pPr><a:r><a:t>Wingdings square</a:t></a:r></a:p>
  <a:p><a:pPr><a:buNone/></a:pPr><a:r><a:t>No bullet</a:t></a:r></a:p>
  <a:p><a:pPr><a:buClrTx/><a:buFontTx/><a:buChar char="•"/></a:pPr><a:r><a:t>Follow text</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 6 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Text != "Primary energy resources" || !got[0].Bold || len(got[0].Runs) != 1 || !got[0].Runs[0].Bold {
		t.Fatalf("unexpected first paragraph: %+v", got[0])
	}
	if got[1].Text != "Fossil" || got[1].Bullet != "•" || got[1].Level != 1 {
		t.Fatalf("unexpected default bullet paragraph: %+v", got[1])
	}
	if got[2].Text != "Coal" || got[2].Bullet != "-" || got[2].Level != 2 {
		t.Fatalf("unexpected explicit bullet paragraph: %+v", got[2])
	}
	if got[2].BulletFontSize != 1400 {
		t.Fatalf("expected explicit bullet font size, got %+v", got[2])
	}
	expectedWingdingsSquare := "▪"
	if exactFontFamilyAvailable("Wingdings") {
		expectedWingdingsSquare = "\uf0a7"
	}
	if got[3].Text != "Wingdings square" || got[3].Bullet != expectedWingdingsSquare || got[3].Level != 1 {
		t.Fatalf("unexpected Wingdings bullet paragraph: %+v", got[3])
	}
	if got[3].BulletFontFamily != "Wingdings" {
		t.Fatalf("expected Wingdings bullet font family to be preserved, got %+v", got[3])
	}
	if got[4].Text != "No bullet" || !got[4].NoBullet {
		t.Fatalf("unexpected no-bullet paragraph: %+v", got[4])
	}
	if !got[5].BulletColorTx || !got[5].BulletFontTx {
		t.Fatalf("expected bullet color/font to follow text, got %+v", got[5])
	}
}

func TestTextParagraphsFromNodeMapsWingdingsNotSignBullet(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buFont typeface="Wingdings"/><a:buChar char="Ø"/></a:pPr><a:r><a:t>Mapped</a:t></a:r></a:p>
  <a:p><a:pPr><a:buFont typeface="Arial"/><a:buChar char="Ø"/></a:pPr><a:r><a:t>Literal</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	expectedWingdingsNotSign := "¬"
	if exactFontFamilyAvailable("Wingdings") {
		expectedWingdingsNotSign = "\uf0d8"
	}
	if got[0].Bullet != expectedWingdingsNotSign || got[0].BulletFontFamily != "Wingdings" {
		t.Fatalf("expected Wingdings Ø bullet to map to Unicode not sign, got %+v", got[0])
	}
	if got[1].Bullet != "Ø" || got[1].BulletFontFamily != "Arial" {
		t.Fatalf("non-Wingdings Ø bullet should stay literal, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeNumbersAutoBullets(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>First</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>Second</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod" startAt="4"/></a:pPr><a:r><a:t>Restarted</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"><a:buAutoNum type="arabicParenR" startAt="2"/></a:pPr><a:r><a:t>Nested</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>Continued</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 5 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bullet != "a." || got[1].Bullet != "b." || got[2].Bullet != "d." || got[3].Bullet != "2)" || got[4].Bullet != "e." {
		t.Fatalf("unexpected auto-number bullets: %+v", got)
	}
}

func TestTextParagraphsPreservesSingleLeadingRunSpace(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:t> Welcome</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 || got[0].Runs[0].Text != " Welcome" {
		t.Fatalf("expected single leading space to be preserved, got %+v", got)
	}
}

func TestTextParagraphsPreservesManualLeadingPadding(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:t>          Centered title</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 || got[0].Runs[0].Text != "          Centered title" {
		t.Fatalf("expected manual leading padding to be preserved, got %+v", got)
	}
}

func TestTextParagraphsFromNodeUsesNoBulletSizeAsFallbackFontSize(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buSzPts val="4400"/><a:buNone/><a:spcAft><a:spcPts val="1200"/></a:spcAft></a:pPr><a:r><a:rPr/><a:t>Title</a:t></a:r></a:p>
  <a:p><a:pPr><a:buSzPts val="2200"/><a:buNone/></a:pPr><a:r><a:rPr sz="1800"/><a:t>Explicit</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 4400 {
		t.Fatalf("expected no-bullet paragraph size fallback, got %+v", got[0])
	}
	if got[0].SpaceAfter != 12 {
		t.Fatalf("expected paragraph after-spacing, got %+v", got[0])
	}
	if got[1].FontSize != 1800 {
		t.Fatalf("explicit run size should win over no-bullet fallback, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeParsesPercentParagraphSpacing(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:spcBef><a:spcPct val="90000"/></a:spcBef><a:spcAft><a:spcPct val="110000"/></a:spcAft></a:pPr><a:r><a:rPr sz="1800"/><a:t>Percent spacing</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || !got[0].HasSpaceBefore || got[0].SpaceBeforePct != 90000 || got[0].SpaceAfterPct != 110000 {
		t.Fatalf("expected percent paragraph spacing, got %+v", got)
	}
	if got[0].SpaceBefore != 0 || got[0].SpaceAfter != 0 {
		t.Fatalf("percent spacing should not be stored as fixed pixels, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodePreservesEmptyParagraphs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="2400"/><a:t>Before</a:t></a:r></a:p>
  <a:p><a:endParaRPr sz="2400" b="1"/></a:p>
  <a:p><a:r><a:rPr sz="2400"/><a:t>After</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 3 {
		t.Fatalf("expected empty paragraph to be preserved, got %+v", got)
	}
	if got[1].Text != "" || got[1].FontSize != 2400 || !got[1].Bold {
		t.Fatalf("expected empty paragraph end properties, got %+v", got[1])
	}
	if !got[1].NoBullet {
		t.Fatalf("expected empty paragraph to reserve space without a bullet, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeDoesNotApplyEndParagraphPropertiesToExistingRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1700"/><a:t>Visible</a:t></a:r><a:endParaRPr sz="1700" b="1"/></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bold || len(got[0].Runs) != 1 || got[0].Runs[0].Bold {
		t.Fatalf("endParaRPr should not restyle existing text runs, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphPropertiesForUnstyledRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr/><a:t>Visible</a:t></a:r><a:endParaRPr sz="1700" b="1"/></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 1700 || !got[0].Bold {
		t.Fatalf("endParaRPr should seed paragraph defaults when runs are unstyled, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphPropertiesWhenRunsOnlySetColor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:rPr><a:t>Visible</a:t></a:r>
    <a:endParaRPr sz="1700" b="1"/>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 1700 || !got[0].Bold {
		t.Fatalf("color-only runs should still allow endParaRPr to seed missing defaults, got %+v", got[0])
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if !segment.Bold || !segment.HasTextColor || segment.TextColor.R != 0xff {
		t.Fatalf("endParaRPr defaults should not replace run color, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesMixedRunStyles(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"/><a:t>Energy services - </a:t></a:r><a:r><a:rPr sz="1800" b="1"/><a:t>Mobility</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected mixed run parse: %+v", got)
	}
	if got[0].Bold || got[0].Italic || got[0].FontSize != 1800 {
		t.Fatalf("paragraph-level metadata should preserve only uniform values: %+v", got[0])
	}
	if got[0].Runs[0].Text != "Energy services - " || got[0].Runs[0].Bold || got[0].Runs[1].Text != "Mobility" || !got[0].Runs[1].Bold {
		t.Fatalf("unexpected run styles: %+v", got[0].Runs)
	}
}

func TestExplicitRunBoldFalseOverridesInheritedBold(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:r><a:rPr b="0"/><a:t>Normal</a:t></a:r><a:r><a:rPr/><a:t>Inherited</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	got[0].Bold = true
	normal := runToSegment(got[0].Runs[0], got[0])
	inherited := runToSegment(got[0].Runs[1], got[0])
	if normal.Bold {
		t.Fatalf("explicit b=0 run should stay non-bold under inherited bold paragraph: %+v", normal)
	}
	if !inherited.Bold {
		t.Fatalf("run without explicit bold should inherit paragraph bold: %+v", inherited)
	}
}

func TestExplicitRunItalicFalseOverridesInheritedItalic(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:r><a:rPr i="0"/><a:t>Normal</a:t></a:r><a:r><a:rPr/><a:t>Inherited</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	got[0].Italic = true
	normal := runToSegment(got[0].Runs[0], got[0])
	inherited := runToSegment(got[0].Runs[1], got[0])
	if normal.Italic {
		t.Fatalf("explicit i=0 run should stay non-italic under inherited italic paragraph: %+v", normal)
	}
	if !inherited.Italic {
		t.Fatalf("run without explicit italic should inherit paragraph italic: %+v", inherited)
	}
}

func TestTextParagraphsFromNodeCapturesMixedRunItalics(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"/><a:t>Regular </a:t></a:r><a:r><a:rPr sz="1800" i="1"/><a:t>Italic</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected mixed run parse: %+v", got)
	}
	if got[0].Italic {
		t.Fatalf("mixed italic runs should not promote the whole paragraph: %+v", got[0])
	}
	if got[0].Runs[0].Italic || !got[0].Runs[1].Italic {
		t.Fatalf("unexpected run italic styles: %+v", got[0].Runs)
	}
}

func TestTextParagraphsFromNodeCapturesRunBaseline(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="3200" b="1"/><a:t>CO</a:t></a:r><a:r><a:rPr sz="3200" b="1" baseline="-25000"/><a:t>2</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[1].Baseline != -25000 {
		t.Fatalf("expected subscript baseline to be preserved, got %+v", got[0].Runs[1])
	}
	if shift := segmentBaselineShift(textLineSegment{FontSize: 3200, Baseline: -25000}, 3200); shift >= 0 {
		t.Fatalf("expected negative baseline to produce downward drawing offset, got %d", shift)
	}
	if shift := segmentBaselineShiftAtDPI(textLineSegment{FontSize: 3200, Baseline: -25000}, 3200, 96); shift != -11 {
		t.Fatalf("expected 96 DPI baseline shift to scale from point size, got %d", shift)
	}
	segment := runToSegment(got[0].Runs[1], got[0])
	if segment.FontSize >= got[0].Runs[1].FontSize || segment.BaselineFontSize != got[0].Runs[1].FontSize {
		t.Fatalf("expected baseline run to render smaller while preserving shift font size, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesRunHighlight(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:highlight><a:srgbClr val="FFFF00"/></a:highlight></a:rPr><a:t>Marked</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	run := got[0].Runs[0]
	if !run.HasHighlightColor || run.HighlightColor.R != 0xff || run.HighlightColor.G != 0xff || run.HighlightColor.B != 0x00 {
		t.Fatalf("expected highlight color to be preserved, got %+v", run)
	}
	segment := runToSegment(run, got[0])
	if !segment.HasHighlightColor || segment.HighlightColor != run.HighlightColor {
		t.Fatalf("expected highlight color on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesRunUnderline(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800" u="sng"/><a:t>Underlined</a:t></a:r><a:r><a:rPr sz="1800" u="none"/><a:t>Plain</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if !got[0].Runs[0].Underline || got[0].Runs[1].Underline {
		t.Fatalf("expected only single underline run, got %+v", got[0].Runs)
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if !segment.Underline {
		t.Fatalf("expected underline on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeKeepsParagraphAlignmentScoped(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr algn="ctr"/><a:r><a:t>Centered</a:t></a:r></a:p>
  <a:p><a:r><a:t>Default</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph parse: %+v", got)
	}
	if got[0].TextAlign != "ctr" || got[1].TextAlign != "" {
		t.Fatalf("paragraph alignment leaked across paragraphs: %+v", got)
	}
	element := parseSlideElementNode(&xmlNode{Name: "sp", Children: []*xmlNode{root}}, renderTransform{ScaleX: 1, ScaleY: 1})
	if element.TextAlign != "" {
		t.Fatalf("paragraph alignment should not promote to shape alignment, got %+v", element)
	}
}

func TestTextParagraphsFromNodeInheritsListStyleParagraphAlignment(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:bodyPr/>
  <a:lstStyle><a:lvl1pPr algn="ctr"/></a:lstStyle>
  <a:p><a:pPr/><a:r><a:t>Centered by style</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || got[0].TextAlign != "ctr" {
		t.Fatalf("expected list style paragraph alignment, got %+v", got)
	}
}

func TestTextParagraphsFromNodeCapturesRunFontFamily(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:latin typeface="Trebuchet MS"/></a:rPr><a:t>Label</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	run := got[0].Runs[0]
	if run.FontFamily != "Trebuchet MS" {
		t.Fatalf("expected run font family to be preserved, got %+v", run)
	}
	segment := runToSegment(run, got[0])
	if segment.FontFamily != "Trebuchet MS" {
		t.Fatalf("expected run font family on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeUsesExplicitAlternateTypefaceForNonLatinText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>标题</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "Arial" {
		t.Fatalf("expected run fallback typeface to be preserved, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeDoesNotUseAlternateTypefaceForLatinText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>Label</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "" {
		t.Fatalf("latin text without a latin typeface should not use alternate font slots, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeDoesNotUseAlternateTypefaceForLatinTextWithMathSymbol(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>value ≥99%</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "" {
		t.Fatalf("math symbols in Latin text should not switch the whole run to alternate font slots, got %+v", got[0].Runs[0])
	}
}

func TestDrawTextUnderlinePaintsBelowBaseline(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	face, err := openFontFace(1800, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()
	drawTextUnderline(img, face, 10, 20, 60, color.RGBA{R: 255, A: 255})
	if !hasColorPixel(img, color.RGBA{R: 255, A: 255}) {
		t.Fatal("expected underline to paint red pixels")
	}
}

func TestResolveSlidePlaceholdersAppliesBodyDefaultBullets(t *testing.T) {
	elements := []slideElement{{
		Text:            "Primary",
		TextParagraphs:  []textParagraph{{Text: "Primary"}, {Text: "No bullet", NoBullet: true}},
		IsPlaceholder:   true,
		PlaceholderIdx:  "1",
		HasTransform:    true,
		PlaceholderType: "",
	}}
	sources := map[string]slideElement{
		"idx:1": {
			IsPlaceholder:   true,
			PlaceholderType: "body",
			PlaceholderIdx:  "1",
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].PlaceholderType != "body" || got[0].TextParagraphs[0].Bullet != "•" {
		t.Fatalf("body placeholder default bullet was not inherited: %+v", got[0])
	}
	if got[0].TextParagraphs[1].Bullet != "" {
		t.Fatalf("explicit no-bullet paragraph gained a bullet: %+v", got[0].TextParagraphs[1])
	}
}

func TestApplyInheritedTextStylesAppliesBodyParagraphMargins(t *testing.T) {
	elements := []slideElement{{
		Text:            "Nested",
		TextParagraphs:  []textParagraph{{Text: "Nested", Level: 1}},
		IsPlaceholder:   true,
		PlaceholderType: "body",
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"body": {
			ParagraphStyles: map[int]paragraphStyle{
				1: {
					HasMarginLeft:  true,
					MarginLeft:     685800,
					HasIndent:      true,
					Indent:         -228600,
					FontSize:       1600,
					Bullet:         "•",
					HasBulletColor: true,
					BulletColor:    color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff},
					HasLineSpacing: true,
					LineSpacingPct: 90000,
					TextAlign:      "ctr",
				},
			},
		},
	})
	if got[0].TextParagraphs[0].Bullet != "•" || !got[0].TextParagraphs[0].HasMarginLeft || got[0].TextParagraphs[0].MarginLeft != 685800 {
		t.Fatalf("body paragraph style was not inherited: %+v", got[0].TextParagraphs[0])
	}
	if !got[0].TextParagraphs[0].HasBulletColor || got[0].TextParagraphs[0].BulletColor.G != 0xad {
		t.Fatalf("body bullet color was not inherited: %+v", got[0].TextParagraphs[0])
	}
	if got[0].TextParagraphs[0].LineSpacingPct != 90000 {
		t.Fatalf("body line spacing was not inherited: %+v", got[0].TextParagraphs[0])
	}
	if got[0].TextParagraphs[0].TextAlign != "ctr" {
		t.Fatalf("body paragraph alignment was not inherited: %+v", got[0].TextParagraphs[0])
	}
	if got[0].TextParagraphs[0].FontSize != 1600 {
		t.Fatalf("body paragraph font size was not inherited: %+v", got[0].TextParagraphs[0])
	}
}

func TestApplyInheritedTextStylesPreservesExplicitZeroParagraphSpacing(t *testing.T) {
	elements := []slideElement{{
		Text: "Title",
		TextParagraphs: []textParagraph{{
			Text:           "Title",
			HasSpaceBefore: true,
			HasSpaceAfter:  true,
			HasLineSpacing: true,
		}},
		IsPlaceholder:   true,
		PlaceholderType: "title",
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"title": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {
					HasSpaceBefore: true,
					SpaceBefore:    12,
					HasSpaceAfter:  true,
					SpaceAfter:     14,
					HasLineSpacing: true,
					LineSpacingPct: 90000,
				},
			},
		},
	})
	paragraph := got[0].TextParagraphs[0]
	if paragraph.SpaceBefore != 0 || paragraph.SpaceBeforePct != 0 || paragraph.SpaceAfter != 0 || paragraph.SpaceAfterPct != 0 || paragraph.LineSpacingPct != 0 {
		t.Fatalf("explicit zero paragraph spacing should win over inherited style: %+v", paragraph)
	}
}

func TestParseTextStylesNormalizesOfficeSymbolBullets(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buClr><a:srgbClr val="70AD47"/></a:buClr>
        <a:buFont typeface="Wingdings 3"/>
        <a:buSzPct val="80000"/>
        <a:buChar char="&#xF075;"/>
        <a:defRPr sz="1800" b="1"><a:solidFill><a:srgbClr val="112233"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
	</p:sldMaster>`), defaultThemeColors())
	style := styles["body"].ParagraphStyles[0]
	expectedBullet := "▶"
	if exactFontFamilyAvailable("Wingdings 3") {
		expectedBullet = "\uf075"
	}
	if style.Bullet != expectedBullet {
		t.Fatalf("unexpected symbol bullet, got %+v", style)
	}
	if style.BulletFontFamily != "Wingdings 3" {
		t.Fatalf("expected paragraph bullet font family, got %+v", style)
	}
	if style.FontSize != 1800 {
		t.Fatalf("expected paragraph defRPr font size, got %+v", style)
	}
	if style.FontFamily != "Arial" {
		t.Fatalf("expected paragraph defRPr font family, got %+v", style)
	}
	if style.BulletSizePct != 80000 {
		t.Fatalf("expected paragraph bullet size percent, got %+v", style)
	}
	if !style.Bold || !style.HasTextColor || style.TextColor != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}) {
		t.Fatalf("expected paragraph defRPr bold and text color, got %+v", style)
	}
	if !style.HasBulletColor || style.BulletColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff}) {
		t.Fatalf("expected parsed bullet color, got %+v", style)
	}
}

func TestParseTextStylesCapturesBulletFollowTextProperties(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buClrTx/>
        <a:buFontTx/>
        <a:buChar char="•"/>
        <a:defRPr sz="1800"><a:solidFill><a:srgbClr val="112233"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())
	style := styles["body"].ParagraphStyles[0]
	if !style.BulletColorTx || !style.BulletFontTx || style.HasBulletColor || style.BulletFontFamily != "" {
		t.Fatalf("expected bullet follow-text properties, got %+v", style)
	}
}

func TestApplyParagraphStyleKeepsLocalExplicitBulletProperties(t *testing.T) {
	paragraph := textParagraph{
		BulletFontFamily: "Arial",
		HasBulletColor:   true,
		BulletColor:      color.RGBA{R: 1, G: 2, B: 3, A: 255},
	}
	applyParagraphStyle(&paragraph, paragraphStyle{
		BulletFontTx:  true,
		BulletColorTx: true,
	})
	if paragraph.BulletFontTx || paragraph.BulletFontFamily != "Arial" {
		t.Fatalf("local explicit bullet font should win over inherited buFontTx: %+v", paragraph)
	}
	if paragraph.BulletColorTx || !paragraph.HasBulletColor || paragraph.BulletColor.R != 1 {
		t.Fatalf("local explicit bullet color should win over inherited buClrTx: %+v", paragraph)
	}
}

func TestTextParagraphsFromNodeUsesParagraphDefaultRunProperties(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800" b="1"><a:solidFill><a:srgbClr val="336699"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
    <a:r><a:rPr><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:rPr><a:t> Red</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "Arial" || got[0].FontSize != 1800 || !got[0].Bold || !got[0].HasTextColor || got[0].TextColor != (color.RGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Fatalf("paragraph default run properties were not applied: %+v", got[0])
	}
	defaultSegment := runToSegment(got[0].Runs[0], got[0])
	if defaultSegment.FontFamily != "Arial" || !defaultSegment.Bold || !defaultSegment.HasTextColor || defaultSegment.TextColor != got[0].TextColor {
		t.Fatalf("paragraph defaults were not carried to unstyled run: %+v", defaultSegment)
	}
	redSegment := runToSegment(got[0].Runs[1], got[0])
	if !redSegment.HasTextColor || redSegment.TextColor.R != 0xff || redSegment.TextColor.G != 0 || redSegment.TextColor.B != 0 {
		t.Fatalf("explicit run color should win over paragraph default: %+v", redSegment)
	}
}

func TestTextParagraphsFromNodeDoesNotUseDefaultAlternateTypefaceWithoutRunText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "" {
		t.Fatalf("paragraph default alternate typeface should not apply without script-specific text: %+v", got[0])
	}
}

func TestParagraphDefaultRunPropertiesPreserveThemeFontTokens(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800"><a:latin typeface="+mn-lt"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "+mn-lt" {
		t.Fatalf("theme font token should be preserved until theme resolution: %+v", got[0])
	}
}

func TestApplyThemeFontFamiliesResolvesParagraphDefaults(t *testing.T) {
	got := applyThemeFontFamilies([]slideElement{{
		Text: "Defaulted",
		TextParagraphs: []textParagraph{{
			FontFamily: "Arial",
			Runs: []textRun{
				{Text: "Defaulted"},
				{Text: " Explicit", FontFamily: "+mj-lt"},
			},
		}},
	}}, themeFonts{MajorLatin: "Trebuchet MS", MinorLatin: "Arial"})
	paragraph := got[0].TextParagraphs[0]
	if paragraph.FontFamily != "Arial" || paragraph.Runs[1].FontFamily != "Trebuchet MS" {
		t.Fatalf("theme font families were not resolved: %+v", paragraph)
	}
	if segment := runToSegment(paragraph.Runs[0], paragraph); segment.FontFamily != "Arial" {
		t.Fatalf("paragraph font family was not used for unstyled run: %+v", segment)
	}
}

func TestInheritedTextStylesResolveThemeFontFamiliesAfterApplication(t *testing.T) {
	elements := []slideElement{{
		Text:            "Title",
		TextParagraphs:  []textParagraph{{Text: "Title"}},
		IsPlaceholder:   true,
		PlaceholderType: "ctrTitle",
	}}
	elements = applyInheritedTextStyles(elements, map[string]textStyle{
		"ctrTitle": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {FontFamily: "+mj-lt"},
			},
		},
	})
	got := applyThemeFontFamilies(elements, themeFonts{MajorLatin: "Calibri Light", MinorLatin: "Calibri"})
	if got[0].TextParagraphs[0].FontFamily != "Calibri Light" {
		t.Fatalf("inherited theme font token was not resolved after style application: %+v", got[0].TextParagraphs[0])
	}
}

func TestApplyInheritedTextStylesPreservesExplicitParagraphFontSize(t *testing.T) {
	elements := []slideElement{{
		Text:            "Nested",
		TextParagraphs:  []textParagraph{{Text: "Nested", Level: 1, FontSize: 1400}},
		IsPlaceholder:   true,
		PlaceholderType: "body",
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"body": {
			ParagraphStyles: map[int]paragraphStyle{
				1: {FontSize: 1600},
			},
		},
	})
	if got[0].TextParagraphs[0].FontSize != 1400 {
		t.Fatalf("inherited paragraph style overrode explicit font size: %+v", got[0].TextParagraphs[0])
	}
}

func TestTextRenderLinesCarryParagraphLineSpacing(t *testing.T) {
	lines := textLayoutStyledParagraphLines(nil, nil, []textParagraph{{
		Text:           "First",
		LineSpacingPct: 90000,
	}}, "", 200, "none")
	if len(lines) != 1 || lines[0].LineSpacingPct != 90000 {
		t.Fatalf("expected line spacing on rendered line, got %+v", lines)
	}
}

func TestApplyLineSpacingScalesHeight(t *testing.T) {
	if got := applyLineSpacing(50, 90000); got != 45 {
		t.Fatalf("unexpected scaled line height: %d", got)
	}
	if got := applyLineSpacing(50, 0); got != 50 {
		t.Fatalf("unexpected default line height: %d", got)
	}
}

func TestApplyLineSpacingUsesDrawingMLFontSizeForPercentSpacing(t *testing.T) {
	if got := applyLineSpacingAtDPI(32, 150000, 1700, 72); got != 26 {
		t.Fatalf("expected 150%% line spacing from 17pt font size, got %d", got)
	}
	if got := applyLineSpacingAtDPI(32, 150000, 1700, 96); got != 35 {
		t.Fatalf("expected 96-DPI line spacing from 17pt font size, got %d", got)
	}
	if got := applyLineSpacingAtDPI(32, 100000, 1700, 72); got != 32 {
		t.Fatalf("100%% explicit line spacing should preserve drawable metric height, got %d", got)
	}
}

func TestParagraphSpacingPercentPixelsScalesFontSize(t *testing.T) {
	if got := paragraphSpacingPercentPixels(90000, 2000); got != 18 {
		t.Fatalf("expected 90%% paragraph spacing from 20pt text, got %d", got)
	}
	if got := paragraphSpacingPercentPixelsAtDPI(110000, 1800, 96); got != 26 {
		t.Fatalf("expected 110%% paragraph spacing from 18pt text at 96 DPI, got %d", got)
	}
	if got := paragraphSpacingPercentPixels(0, 2000); got != 0 {
		t.Fatalf("expected zero paragraph spacing, got %d", got)
	}
}

func TestLineFontSizeUsesLargestSegmentFontSize(t *testing.T) {
	got := lineFontSize(textRenderLine{FontSize: 1200, Segments: []textLineSegment{
		{Text: "small", FontSize: 1000},
		{Text: "large", FontSize: 2200},
		{Text: "fallback"},
	}}, 1800)
	if got != 2200 {
		t.Fatalf("expected largest explicit segment font size, got %d", got)
	}
}

func TestMeasureTextRenderLinesUsesFontLineMetricHeight(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()

	face, err := faces.Get(1800, false, false)
	if err != nil {
		t.Fatal(err)
	}
	metrics := face.Metrics()
	want := defaultLineMetricHeight(metrics)

	got, err := measureTextRenderLines(faces, []textRenderLine{{Text: "A", FontSize: 1800}}, 1800)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Height != want {
		t.Fatalf("expected font line metric height %d, got %+v", want, got)
	}
}

func TestTextBoundsAppliesBodyInsets(t *testing.T) {
	got := textBounds(
		image.Rect(0, 0, 200, 100),
		slideElement{HasInsets: true, InsetLeft: 914400, InsetTop: 914400, InsetRight: 457200, InsetBottom: 457200},
		slideSize{CX: 9144000, CY: 4572000},
		image.Rect(0, 0, 1000, 500),
	)
	want := image.Rect(100, 100, 150, 50)
	if got != want {
		t.Fatalf("unexpected text bounds: got=%v want=%v", got, want)
	}
}

func TestTextBoundsUsesDrawingMLDefaultInsets(t *testing.T) {
	got := textBounds(
		image.Rect(0, 0, 200, 100),
		slideElement{},
		slideSize{CX: 9144000, CY: 4572000},
		image.Rect(0, 0, 1000, 500),
	)
	want := image.Rect(10, 5, 190, 95)
	if got != want {
		t.Fatalf("unexpected default text bounds: got=%v want=%v", got, want)
	}
}

func TestParseBodyPropertiesKeepsDefaultInsetsForOmittedSides(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a" lIns="0"/>`))
	if err != nil {
		t.Fatal(err)
	}

	var got slideElement
	parseBodyProperties(root, &got)
	if !got.HasInsets {
		t.Fatalf("expected explicit body insets: %+v", got)
	}
	if got.InsetLeft != 0 || got.InsetRight != defaultTextInsetXEMU || got.InsetTop != defaultTextInsetYEMU || got.InsetBottom != defaultTextInsetYEMU {
		t.Fatalf("omitted body insets should keep DrawingML defaults, got %+v", got)
	}
}

func TestParseTextPropertiesCapturesItalicRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
		<p:txBody><a:p><a:r><a:rPr sz="1800" b="1" i="1"/><a:t>Styled</a:t></a:r></a:p></p:txBody>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	element := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if element.Italic {
		t.Fatalf("run-only italics should not promote the whole shape")
	}
	if len(element.TextParagraphs) != 1 || len(element.TextParagraphs[0].Runs) != 1 || !element.TextParagraphs[0].Runs[0].Italic {
		t.Fatalf("expected italic run to be preserved, got %+v", element.TextParagraphs)
	}
}

func TestTextLayoutParagraphLinesAddsBulletPrefix(t *testing.T) {
	face, err := openFontFace(1200, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	got := textLayoutParagraphLines(face, []textParagraph{
		{Text: "Primary energy resources"},
		{Text: "Fossil fuels", Bullet: "•", Level: 1},
	}, "", 300, "")
	want := []string{"Primary energy resources", "  • Fossil fuels"}
	if len(got) != len(want) {
		t.Fatalf("unexpected line count: got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected line %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestTextLayoutParagraphLinesPreservesExplicitBreaks(t *testing.T) {
	face, err := openFontFace(1200, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	got := textLayoutParagraphLines(face, []textParagraph{
		{Text: "Line one\nLine two", Bullet: "•"},
	}, "", 300, "none")
	want := []string{"• Line one", "  Line two"}
	if len(got) != len(want) {
		t.Fatalf("unexpected line count: got=%q want=%q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected line %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestWrapTextWithPrefixesPreservesAuthoredSpaces(t *testing.T) {
	face, err := openFontFace(1200, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	lines := wrapTextWithPrefixes(face, "  Alpha   beta gamma", measureString(face, "  Alpha   beta"), "", "")
	if len(lines) != 2 || lines[0] != "  Alpha   beta" || lines[1] != "gamma" {
		t.Fatalf("expected plain wrapping to preserve authored spaces without carrying separator spaces to the next line, got %+v", lines)
	}
}

func TestTextRenderLinesForElementKeepsMixedRunSegments(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:     "Energy services - Mobility",
			FontSize: 1800,
			Runs: []textRun{
				{Text: "Energy services - ", FontSize: 1800},
				{Text: "Mobility", FontSize: 1800, Bold: true},
			},
		}},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) != 2 {
		t.Fatalf("expected one segmented line, got %+v", lines)
	}
	if strings.TrimSpace(lines[0].Segments[1].Text) != "Mobility" || !lines[0].Segments[1].Bold {
		t.Fatalf("mixed-run bold segment was not preserved: %+v", lines[0].Segments)
	}
}

func TestTextRenderLinesForElementPreservesLeadingSpacesWhenLineFits(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:     "Padded heading",
			FontSize: 1800,
			Runs:     []textRun{{Text: "     Padded heading", FontSize: 1800}},
		}},
	}, 800)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) != 1 || lines[0].Segments[0].Text != "     Padded heading" {
		t.Fatalf("expected leading spaces to stay in the rendered segment, got %+v", lines)
	}
}

func TestTextRenderLinesForElementAppliesBulletFontSize(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:           "Bullet",
			Bullet:         "•",
			FontSize:       2000,
			BulletSizePct:  80000,
			HasBulletColor: true,
			BulletColor:    color.RGBA{R: 1, G: 2, B: 3, A: 255},
			Runs:           []textRun{{Text: "Bullet", FontSize: 2000}},
		}},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) < 2 {
		t.Fatalf("expected bullet prefix segments, got %+v", lines)
	}
	bullet := lines[0].Segments[0]
	if bullet.Text != "•" || bullet.FontSize != 1600 || !bullet.HasTextColor || bullet.TextColor.R != 1 {
		t.Fatalf("unexpected bullet segment: %+v", bullet)
	}
	if lines[0].Segments[1].FontSize != 2000 {
		t.Fatalf("bullet size should not alter spacer/text size: %+v", lines[0].Segments)
	}
}

func TestTextRenderLinesForElementAppliesBulletFontFamily(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:             "Bullet",
			Bullet:           "•",
			BulletFontFamily: "Arial",
			FontSize:         1800,
			Runs:             []textRun{{Text: "Bullet", FontSize: 1800}},
		}},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) < 2 || lines[0].Segments[0].FontFamily != "Arial" {
		t.Fatalf("expected bullet segment to use authored bullet font family, got %+v", lines)
	}

	wingdings := appendPrefixSegment("¬ ", textParagraph{Bullet: "¬", BulletFontFamily: "Wingdings", FontSize: 1800}, nil)
	if len(wingdings) < 1 || wingdings[0].FontFamily != "" {
		t.Fatalf("unexpected Wingdings bullet font family, got %+v", wingdings)
	}
}

func TestTextRenderLinesForElementUsesParagraphFontForBulletFontTx(t *testing.T) {
	segments := appendPrefixSegment("• ", textParagraph{
		Bullet:     "•",
		FontFamily: "Trebuchet MS",
		FontSize:   1800,
	}, nil)
	if len(segments) < 1 || segments[0].FontFamily != "Trebuchet MS" {
		t.Fatalf("bullet without explicit buFont should inherit paragraph font family, got %+v", segments)
	}

	wingdings := appendPrefixSegment("¬ ", textParagraph{
		Bullet:           "¬",
		BulletFontFamily: "Wingdings",
		FontFamily:       "Trebuchet MS",
		FontSize:         1800,
	}, nil)
	if len(wingdings) < 1 || wingdings[0].FontFamily != "Trebuchet MS" {
		t.Fatalf("unexpected mapped symbol bullet font, got %+v", wingdings)
	}
}

func TestTextRenderLinesForElementAppliesBulletColorTx(t *testing.T) {
	segments := appendPrefixSegment("• ", textParagraph{
		Bullet:        "•",
		BulletColorTx: true,
		FontSize:      1800,
		Runs: []textRun{{
			Text:         "Body",
			FontSize:     1800,
			HasTextColor: true,
			TextColor:    color.RGBA{R: 9, G: 8, B: 7, A: 255},
		}},
	}, []textLineSegment{{
		Text:         "Body",
		FontSize:     1800,
		HasTextColor: true,
		TextColor:    color.RGBA{R: 9, G: 8, B: 7, A: 255},
	}})
	if len(segments) < 1 || !segments[0].HasTextColor || segments[0].TextColor.R != 9 || segments[0].TextColor.G != 8 {
		t.Fatalf("expected bullet color to follow text, got %+v", segments)
	}
}

func TestTextRenderLinesForElementAppliesParagraphSpaceAfter(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{
			{Text: "First", FontSize: 1800, SpaceAfter: 7, Runs: []textRun{{Text: "First", FontSize: 1800}}},
			{Text: "Second", FontSize: 1800, SpaceBefore: 5, Runs: []textRun{{Text: "Second", FontSize: 1800}}},
		},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %+v", lines)
	}
	if lines[0].SpaceBefore != 0 || lines[1].SpaceBefore != 12 {
		t.Fatalf("unexpected paragraph spacing: %+v", lines)
	}
}

func TestTextRenderLinesForElementAppliesPercentParagraphSpacing(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{
			{Text: "First", FontSize: 1800, SpaceAfterPct: 90000, Runs: []textRun{{Text: "First", FontSize: 1800}}},
			{Text: "Second", FontSize: 1800, SpaceBeforePct: 110000, Runs: []textRun{{Text: "Second", FontSize: 1800}}},
		},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].SpaceBeforePct != 0 || lines[1].SpaceBeforePct != 200000 {
		t.Fatalf("unexpected percent paragraph spacing on render lines: %+v", lines)
	}
	measured := []measuredTextLine{
		{Height: 10},
		{SpaceBefore: paragraphSpacingPercentPixels(lines[1].SpaceBeforePct, 1800), Height: 10},
	}
	if got := measuredTextHeight(measured); got != 56 {
		t.Fatalf("expected measured height to include percent paragraph spacing, got %d", got)
	}
}

func TestTextRenderLinesForElementHonorsFirstLastParagraphSpacing(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize:                1800,
		IncludeFirstLastSpacing: true,
		TextParagraphs: []textParagraph{
			{Text: "First", FontSize: 1800, SpaceBefore: 4, SpaceAfter: 7, Runs: []textRun{{Text: "First", FontSize: 1800}}},
			{Text: "Second", FontSize: 1800, SpaceBefore: 5, SpaceAfter: 9, Runs: []textRun{{Text: "Second", FontSize: 1800}}},
		},
	}, 400)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected two lines, got %+v", lines)
	}
	if lines[0].SpaceBefore != 4 || lines[1].SpaceBefore != 12 || lines[1].SpaceAfter != 9 {
		t.Fatalf("unexpected first/last paragraph spacing: %+v", lines)
	}
	measured := []measuredTextLine{
		{SpaceBefore: lines[0].SpaceBefore, Height: 10},
		{SpaceBefore: lines[1].SpaceBefore, Height: 10, SpaceAfter: lines[1].SpaceAfter},
	}
	if got := measuredTextHeight(measured); got != 45 {
		t.Fatalf("expected measured height to include first and last spacing, got %d", got)
	}
}

func TestTextRenderLinesForElementPreservesTabbedParagraphs(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:     "Cost per piece \t= 273",
			FontSize: 1800,
			Runs: []textRun{{
				Text:     "Cost per piece \t= 273",
				FontSize: 1800,
			}},
		}},
	}, 80)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) != 1 || !strings.Contains(lines[0].Segments[0].Text, "\t") {
		t.Fatalf("expected tabbed paragraph to remain on one tabbed line, got %+v", lines)
	}
}

func TestTextRenderLinesForElementUsesParagraphDefaultTabSize(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 1800,
		TextParagraphs: []textParagraph{{
			Text:           "A\tB",
			FontSize:       1800,
			HasDefaultTab:  true,
			DefaultTabSize: emuPerInch / 2,
			Runs:           []textRun{{Text: "A\tB", FontSize: 1800}},
		}},
	}, 160)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].TabStops) < 2 || lines[0].TabStops[0] != 36 || lines[0].TabStops[1] != 72 {
		t.Fatalf("expected half-inch default tab stops on render line, got %+v", lines)
	}
}

func TestTextRenderLinesForElementUsesRightMarginForWrapping(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := textParagraph{
		Text:           "alpha beta",
		FontSize:       1800,
		HasMarginRight: true,
		MarginRight:    emuPerInch,
		Runs:           []textRun{{Text: "alpha beta", FontSize: 1800}},
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize:       1800,
		TextParagraphs: []textParagraph{paragraph},
	}, measureString(face, "alpha beta")+4)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].RightOffset != defaultOutputDPI {
		t.Fatalf("expected right margin to reduce wrapping width, got %+v", lines)
	}
}

func TestTextRenderLinesForElementPreservesAuthoredSpacesWhenStyledTextWraps(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := textParagraph{
		Text:     "  Alpha   beta gamma",
		FontSize: 1800,
		Runs:     []textRun{{Text: "  Alpha   beta gamma", FontSize: 1800}},
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize:       1800,
		TextParagraphs: []textParagraph{paragraph},
	}, measureString(face, "  Alpha   beta")+4)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected wrapped styled text, got %+v", lines)
	}
	if !strings.HasPrefix(lines[0].Text, "  Alpha") || strings.Contains(lines[1].Text, "   ") {
		t.Fatalf("expected wrapped styled text to preserve authored leading spaces without carrying separator spaces to the next line, got %+v", lines)
	}
}

func TestStyledWordTokensPreservesRunBoundaryWithoutInventedSpace(t *testing.T) {
	tokens := styledWordTokens([]textRun{{Text: "High"}, {Text: "light"}}, textParagraph{FontSize: 1800})
	if len(tokens) != 2 {
		t.Fatalf("unexpected token count: %+v", tokens)
	}
	if tokens[1].segmentWithPrefix(true).Text != "light" {
		t.Fatalf("expected run boundary without whitespace to stay unspaced, got %+v", tokens)
	}
}

func TestTextRenderLinesForElementMarksWrappedJustifiedLines(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1800, true)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := textParagraph{
		Text:      "alpha beta gamma",
		FontSize:  1800,
		TextAlign: "just",
		Runs:      []textRun{{Text: "alpha beta gamma", FontSize: 1800}},
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize:       1800,
		TextParagraphs: []textParagraph{paragraph},
	}, measureString(face, "alpha beta"))
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) < 2 || !lines[0].Justify || lines[len(lines)-1].Justify {
		t.Fatalf("expected wrapped non-final lines to be justified, got %+v", lines)
	}
}

func TestMeasureTextSegmentWithTabsAdvancesToDefaultStops(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	prefix := measureString(face, "Cost ")
	got := measureTextSegmentWithTabs(face, "Cost \t=", 0)
	if prefix >= defaultTextTabPixels {
		t.Fatalf("test prefix should fit before first tab stop, got %d", prefix)
	}
	if got != defaultTextTabPixels+measureString(face, "=") {
		t.Fatalf("expected tab to advance to first default stop, got %d", got)
	}
	got96 := measureTextSegmentWithTabsAtDPI(face, "Cost \t=", 0, 96)
	if got96 != 96+measureString(face, "=") {
		t.Fatalf("expected 96 DPI tab to advance to one-inch stop, got %d", got96)
	}
	gotExplicit := measureTextSegmentWithTabsAtDPI(face, "Cost \t=", 0, 96, []int{113})
	if gotExplicit != 113+measureString(face, "=") {
		t.Fatalf("expected explicit tab stop to advance to configured position, got %d", gotExplicit)
	}
}

func TestParagraphPixelOffsetsScaleWithDPI(t *testing.T) {
	paragraph := textParagraph{
		HasMarginLeft: true,
		MarginLeft:    emuPerInch,
		HasIndent:     true,
		Indent:        -emuPerInch / 2,
	}

	first, hanging, ok := paragraphPixelOffsetsAtDPI(paragraph, 96)
	if !ok || first != 48 || hanging != 96 {
		t.Fatalf("expected 96 DPI paragraph offsets, got first=%d hanging=%d ok=%v", first, hanging, ok)
	}
}

func TestMarkerSegmentWidthScalesWithDPI(t *testing.T) {
	segment := textLineSegment{Marker: "triangle", FontSize: 2400}
	if got, base := markerSegmentWidthAtDPI(segment, 0, 96), markerSegmentWidth(segment, 0); got <= base {
		t.Fatalf("expected marker width to scale at 96 DPI, got %d base %d", got, base)
	}
}

func TestOpenFontFaceSupportsFontCollections(t *testing.T) {
	fontPath := "/System/Library/Fonts/Helvetica.ttc"
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Skipf("font collection unavailable: %v", err)
	}
	parsed, err := parseFontData(data, false, false)
	if err != nil {
		t.Fatalf("expected font collection to parse: %v", err)
	}
	if parsed == nil {
		t.Fatal("expected parsed font from collection")
	}
}

func TestFallbackFontPointSizeUsesDrawingMLPointSize(t *testing.T) {
	if got := fallbackFontPointSize(2400, false, false); math.Abs(got-24) > 0.001 {
		t.Fatalf("unexpected fallback point size: %v", got)
	}
	if got := fallbackFontPointSize(2400, true, false); math.Abs(got-24) > 0.001 {
		t.Fatalf("unexpected bold fallback point size: %v", got)
	}
	if got := fallbackFontPointSize(4000, false, false); math.Abs(got-40) > 0.001 {
		t.Fatalf("unexpected large fallback point size: %v", got)
	}
	if got := fallbackFontPointSizeWithScale(1600, false, false, 1); math.Abs(got-16) > 0.001 {
		t.Fatalf("unexpected explicit point scale size: %v", got)
	}
	if got := fallbackFontPointSizeWithScaleAndFamily(4400, false, false, 0, "Calibri Light"); math.Abs(got-44) > 0.001 {
		t.Fatalf("font family should not hide point-size scaling, got %v", got)
	}
}

func TestFontResolutionDistinguishesExactAndSubstituteFonts(t *testing.T) {
	for _, candidate := range exactFontCandidatesForFamily("Calibri", false, false) {
		if strings.Contains(candidate, "Helvetica") || strings.Contains(candidate, "Arial") {
			t.Fatalf("exact Calibri candidates must not contain substitute font path: %s", candidate)
		}
	}
	trebuchetCandidates := exactFontCandidatesForFamily("Trebuchet MS", false, false)
	if len(trebuchetCandidates) == 0 || !strings.Contains(trebuchetCandidates[0], "Trebuchet MS.ttf") {
		t.Fatalf("expected exact Trebuchet MS candidates, got %+v", trebuchetCandidates)
	}
	source, ok := substituteFontSourceForFamily("Calibri", false, false)
	if !ok {
		t.Fatal("expected configured Calibri substitute font")
	}
	if !strings.Contains(source.Label, "Carlito") {
		t.Fatalf("expected Carlito substitute, got %q", source.Label)
	}
	if _, err := parseFontData(source.Data, false, false); err != nil {
		t.Fatalf("expected Carlito font to parse: %v", err)
	}
	lightBold, ok := substituteFontSourceForFamily("Calibri Light", true, false)
	if !ok {
		t.Fatal("expected configured Calibri Light substitute font")
	}
	if !strings.Contains(lightBold.Label, "Carlito") {
		t.Fatalf("expected Calibri Light bold substitute to preserve the light face metrics, got %q", lightBold.Label)
	}
}

func TestConfiguredFontMapProvidesExactFontSource(t *testing.T) {
	source, err := readBundledFont(carlitoAssetPath(false, false))
	if err != nil {
		t.Fatal(err)
	}
	fontPath := filepath.Join(t.TempDir(), "PinnedOfficeFont.ttf")
	if err := os.WriteFile(fontPath, source.Data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUPPT_FONT_MAP", "Missing Office Font="+fontPath)

	if !exactFontFamilyAvailable("Missing Office Font") {
		t.Fatal("configured font map should make an otherwise missing font available")
	}
	resolved, err := resolveFontSource("Missing Office Font", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Label != fontPath {
		t.Fatalf("expected configured font source, got %q", resolved.Label)
	}
	if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Missing Office Font"}); message != "" {
		t.Fatalf("configured exact font should not be reported as fallback: %q", message)
	}
}

func TestConfiguredFontMapSupportsStyleSpecificSources(t *testing.T) {
	regular, err := readBundledFont(carlitoAssetPath(false, false))
	if err != nil {
		t.Fatal(err)
	}
	bold, err := readBundledFont(carlitoAssetPath(true, false))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	regularPath := filepath.Join(dir, "Regular.ttf")
	boldPath := filepath.Join(dir, "Bold.ttf")
	if err := os.WriteFile(regularPath, regular.Data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(boldPath, bold.Data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUPPT_FONT_MAP", "Office Sans="+regularPath+";Office Sans:bold="+boldPath)

	got := configuredFontCandidatesForFamily("Office Sans", true, false)
	if len(got) != 2 || got[0] != boldPath || got[1] != regularPath {
		t.Fatalf("expected style-specific source before regular fallback, got %+v", got)
	}
}

func TestFontResolutionUnsupportedMessageReportsFallback(t *testing.T) {
	message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Missing Office Font"})
	if !strings.Contains(message, "generic fallback font") {
		t.Fatalf("expected generic fallback report, got %q", message)
	}
	if !exactFontFamilyAvailable("Calibri") {
		message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Calibri"})
		if message != "" {
			t.Fatalf("supported bundled Calibri substitute should not be reported as fallback: %q", message)
		}
	}
	if firstExistingPath(symbolFontSubstituteCandidates()) != "" {
		message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Symbol"})
		if message != "" {
			t.Fatalf("supported symbol substitute should not be reported as fallback: %q", message)
		}
	}
	if exactFontFamilyAvailable("Arial") {
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Arial"}); message != "" {
			t.Fatalf("exact font should not be reported as fallback: %q", message)
		}
	}
	if exactFontFamilyAvailable("Trebuchet MS") {
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Trebuchet MS"}); message != "" {
			t.Fatalf("exact Trebuchet MS font should not be reported as fallback: %q", message)
		}
	}
	if exactFontFamilyAvailable("Wingdings") {
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Wingdings"}); message != "" {
			t.Fatalf("exact Wingdings font should not be reported as fallback: %q", message)
		}
	}
	if exactFontFamilyAvailable("Times New Roman") {
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Times New Roman"}); message != "" {
			t.Fatalf("exact Times New Roman font should not be reported as fallback: %q", message)
		}
	}
	messages := fontResolutionUnsupportedMessages(slideElement{
		TextParagraphs: []textParagraph{{
			Runs: []textRun{{Text: "A", FontFamily: "Missing Office Font"}},
		}},
	})
	if len(messages) != 1 || !strings.Contains(messages[0], "generic fallback font") {
		t.Fatalf("expected run-level generic fallback report, got %+v", messages)
	}
	messages = fontResolutionUnsupportedMessages(slideElement{
		TextParagraphs: []textParagraph{{
			FontFamily: "Missing Paragraph Font",
			Runs:       []textRun{{Text: "A"}},
		}},
	})
	if len(messages) != 1 || !strings.Contains(messages[0], "Missing Paragraph Font") || !strings.Contains(messages[0], "generic fallback font") {
		t.Fatalf("expected paragraph-level generic fallback report, got %+v", messages)
	}
}

func TestSupportedFontSubstitutesAreResolvedButNotUnsupported(t *testing.T) {
	for _, family := range []string{"Calibri", "Calibri Light"} {
		source, ok := substituteFontSourceForFamily(family, false, false)
		if !ok {
			t.Fatalf("expected supported substitute for %s", family)
		}
		if !strings.Contains(source.Label, "Carlito") {
			t.Fatalf("expected Carlito source for %s, got %q", family, source.Label)
		}
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: family}); message != "" {
			t.Fatalf("supported substitute for %s should not be reported: %q", family, message)
		}
	}
	if firstExistingPath(symbolFontSubstituteCandidates()) != "" {
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Symbol"}); message != "" {
			t.Fatalf("supported symbol substitute should not be reported: %q", message)
		}
	}
}

func TestFontResolutionUnsupportedMessagesHonorStyleSpecificFontMap(t *testing.T) {
	bold, err := readBundledFont(carlitoAssetPath(true, false))
	if err != nil {
		t.Fatal(err)
	}
	boldPath := filepath.Join(t.TempDir(), "OfficeSans-Bold.ttf")
	if err := os.WriteFile(boldPath, bold.Data, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PUPPT_FONT_MAP", "Office Sans:bold="+boldPath)

	messages := fontResolutionUnsupportedMessages(slideElement{
		TextParagraphs: []textParagraph{{
			FontFamily: "Office Sans",
			Bold:       true,
			Runs:       []textRun{{Text: "A"}},
		}},
	})
	if len(messages) != 0 {
		t.Fatalf("style-specific configured font should not be reported as fallback: %+v", messages)
	}
}

func TestDrawShapeTextWrapsAndCenters(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 220, 120))
	element := slideElement{
		Text:      "short words wrap here",
		FontSize:  1200,
		TextAlign: "ctr",
	}
	if err := drawShapeText(img, image.Rect(0, 0, 70, 120), element); err != nil {
		t.Fatal(err)
	}
	if !hasOpaquePixel(img) {
		t.Fatal("expected wrapped centered text to paint pixels")
	}
}

func TestRenderPictureAppliesSourceCrop(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redGreenPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "pic",
		Name:         "Picture 1",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasCrop:      true,
		CropLeft:     50000,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected cropped picture to render fully, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(10, 10).RGBA()
	if r != 0 || g != 0xffff || b != 0 || a != 0xffff {
		t.Fatalf("expected green cropped pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderPicturePaintsOutline(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "pic",
		Name:         "Picture 1",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    9525,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected outlined picture to render fully, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0xffff || a != 0xffff {
		t.Fatalf("expected blue picture outline, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderPictureAppliesCustomGeometryMask(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "pic",
		Name:         "Masked Picture 1",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		CustomPath: []pathPoint{
			{X: 0, Y: 0},
			{X: 1, Y: 0},
			{X: 0, Y: 1},
		},
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported masked picture render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(10, 10).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red masked picture pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	_, _, _, a = img.At(90, 90).RGBA()
	if a != 0 {
		t.Fatalf("expected custom mask to leave outside pixel transparent, got alpha=%04x", a)
	}
}

func TestRenderPicturePaintsOuterShadow(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 120, 96))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:            "pic",
		Name:            "Shadowed Picture 1",
		EmbedID:         "rId1",
		HasTransform:    true,
		ExtCX:           emuPerInch / 2,
		ExtCY:           emuPerInch / 2,
		HasShadow:       true,
		ShadowColor:     color.RGBA{A: 128},
		ShadowDistance:  91440,
		ShadowDirection: 0,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported picture shadow render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(70, 20).RGBA()
	if !(r < 0xffff && g < 0xffff && b < 0xffff && a == 0xffff) {
		t.Fatalf("expected gray blended picture shadow pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderPicturePaintsSoftEdge(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "pic",
		Name:           "Soft Picture 1",
		EmbedID:        "rId1",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		ExtCY:          emuPerInch,
		HasSoftEdge:    true,
		SoftEdgeRadius: emuPerInch / 10,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported picture soft edge render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	_, _, _, edgeAlpha := img.At(0, 0).RGBA()
	if edgeAlpha == 0 || edgeAlpha == 0xffff {
		t.Fatalf("expected soft edge to blur image edge alpha, got alpha=%04x", edgeAlpha)
	}
	r, g, b, a := img.At(48, 48).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected soft edge center to remain red, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestSoftEdgeRadiusUsesGaussianKernelRadius(t *testing.T) {
	element := slideElement{SoftEdgeRadius: emuPerInch / 10}
	got := softEdgeRadiusPixels(element, slideSize{CX: emuPerInch, CY: emuPerInch}, 100)
	if got != 5 {
		t.Fatalf("expected DrawingML soft-edge width to map to half-width Gaussian radius, got %d", got)
	}
}

func TestRenderPictureReportsUnsupportedCustomGeometryCommand(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                  "pic",
		Name:                  "Masked Picture 1",
		EmbedID:               "rId1",
		HasTransform:          true,
		ExtCX:                 emuPerInch,
		ExtCY:                 emuPerInch,
		CustomPath:            []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 0, Y: 1}},
		CustomPathUnsupported: []string{"custom geometry uses unsupported arcTo command"},
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {
			ID:     "rId1",
			Type:   pptx.ImageRelType,
			Target: "../media/image1.png",
		},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "unsupported arcTo command") || !element.Rendered {
		t.Fatalf("expected unsupported custom geometry command report, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestRenderElementsPaintsShapeBlipFill(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/></Relationships>`),
			"ppt/media/image1.png":             redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	elements := []slideElement{{
		Kind:         "sp",
		Name:         "Image Fill Shape",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
	}}

	unsupported := renderElements(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, elements, tableStyleSet{})
	if len(unsupported) != 0 {
		t.Fatalf("expected shape blip fill to render, got unsupported=%+v", unsupported)
	}
	if !elements[0].Rendered {
		t.Fatal("shape blip fill render state was not preserved after shape rendering")
	}
	r, g, b, a := img.At(10, 10).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red shape image fill pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderElementsPaintsRotatedShapeBlipFill(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/></Relationships>`),
			"ppt/media/image1.png":             redPNG(),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	elements := []slideElement{{
		Kind:         "sp",
		Name:         "Rotated Image Fill Shape",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasRotation:  true,
		Rotation:     900000,
	}}

	unsupported := renderElements(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, elements, tableStyleSet{})
	if len(unsupported) != 0 {
		t.Fatalf("expected rotated shape image fill to render without unsupported diagnostics, got unsupported=%+v", unsupported)
	}
	if !elements[0].Rendered {
		t.Fatal("shape blip fill render state was not preserved after shape rendering")
	}
	if got := img.RGBAAt(48, 48); got.R == 0 || got.A == 0 {
		t.Fatalf("expected rotated shape image fill to paint center pixel, got %#v", got)
	}
}

func TestScaleImageCompositesTransparentPixels(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	src.SetRGBA(1, 0, color.RGBA{})
	src.SetRGBA(0, 1, color.RGBA{})
	src.SetRGBA(1, 1, color.RGBA{})

	scaleImage(dst, image.Rect(0, 0, 2, 2), src, src.Bounds())
	r, g, b, a := dst.At(1, 1).RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff || a != 0xffff {
		t.Fatalf("expected transparent source pixel to preserve destination, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestScaleImageUsesInterpolatedSampling(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 4, 1))
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	src.SetRGBA(0, 0, red)
	src.SetRGBA(1, 0, blue)

	scaleImage(dst, image.Rect(0, 0, 4, 1), src, src.Bounds())
	if got := dst.RGBAAt(1, 0); got == red || got == blue || got.A != 255 {
		t.Fatalf("expected interior pixel to be interpolated, got %#v", got)
	}
	if got := dst.RGBAAt(2, 0); got == red || got == blue || got.A != 255 {
		t.Fatalf("expected interior pixel to be interpolated, got %#v", got)
	}
}

func TestSourceCropRectPreservesNegativeCropPadding(t *testing.T) {
	bounds := image.Rect(0, 0, 20, 10)
	got := sourceCropRect(bounds, slideElement{HasCrop: true, CropLeft: -50000, CropRight: -25000})
	want := image.Rect(-10, 0, 25, 10)
	if got != want {
		t.Fatalf("negative DrawingML crop should expand source rectangle: got=%v want=%v", got, want)
	}
}

func TestPictureSourceForElementAppliesDrawingMLFlips(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	src.SetRGBA(0, 0, red)
	src.SetRGBA(1, 0, green)
	src.SetRGBA(2, 0, blue)
	src.SetRGBA(0, 1, white)

	got, bounds := pictureSourceForElement(src, slideElement{HasCrop: true, CropLeft: 33333, FlipH: true, FlipV: true})
	if bounds != image.Rect(0, 0, 2, 2) {
		t.Fatalf("unexpected flipped source bounds: %v", bounds)
	}
	if color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA) != color.RGBAModel.Convert(src.At(2, 1)).(color.RGBA) {
		t.Fatalf("expected top-left pixel to come from horizontally and vertically flipped crop")
	}
	if color.RGBAModel.Convert(got.At(1, 1)).(color.RGBA) != green {
		t.Fatalf("expected bottom-right pixel to come from cropped and flipped source")
	}
}

func TestPictureSourceForElementAppliesAlphaModFix(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 20, G: 40, B: 60, A: 200})

	got, bounds := pictureSourceForElement(src, slideElement{HasImageAlphaModFix: true, ImageAlphaModFixPct: 50000})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed source bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel != (color.RGBA{R: 20, G: 40, B: 60, A: 100}) {
		t.Fatalf("expected alphaModFix to scale opacity only, got %#v", pixel)
	}
}

func TestPictureSourceForElementAppliesGrayBlackWhiteMode(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 100, G: 150, B: 200, A: 220})

	got, bounds := pictureSourceForElement(src, slideElement{BWMode: "gray"})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed source bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel.R != pixel.G || pixel.G != pixel.B || pixel.A != 220 {
		t.Fatalf("expected bwMode gray to preserve alpha and render grayscale, got %#v", pixel)
	}
}

func TestParseBlipEffectsReadsAlphaModFixAmount(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:blip xmlns:a="a" r:embed="rId1" xmlns:r="r"><a:alphaModFix amt="40000"/></a:blip>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBlipEffects(root, &element)
	if !element.HasImageAlphaModFix || element.ImageAlphaModFixPct != 40000 {
		t.Fatalf("expected alphaModFix amount to be parsed, got %+v", element)
	}
}

func TestParseBlipEffectsIgnoresDefaultAlphaModFix(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:blip xmlns:a="a"><a:alphaModFix/></a:blip>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBlipEffects(root, &element)
	if element.HasImageAlphaModFix {
		t.Fatalf("alphaModFix without amt should keep default opacity, got %+v", element)
	}
}

func TestScaleImageAllowsVirtualTransparentSourcePadding(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 4, 1))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	src.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})

	scaleImage(dst, image.Rect(0, 0, 4, 1), src, image.Rect(-2, 0, 2, 1))
	if got := dst.RGBAAt(0, 0); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("transparent virtual padding should preserve destination, got %#v", got)
	}
	if got := dst.RGBAAt(3, 0); got.A == 0 || (got.R == 255 && got.G == 255 && got.B == 255) {
		t.Fatalf("expected in-bounds source content after negative crop padding, got %#v", got)
	}
}

func TestDecodeSVGImagePaintsBasicShapes(t *testing.T) {
	source, err := decodeImage("ppt/media/icon.svg", "image/svg+xml", []byte(`<svg viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
		<rect x="2" y="2" width="6" height="6" fill="#0070C0"/>
		<circle cx="14" cy="5" r="3" fill="black"/>
		<path d="M2 18 18 18 18 20 2 20Z" fill="#ff0000"/>
	</svg>`))
	if err != nil {
		t.Fatalf("decode svg failed: %v", err)
	}
	r, g, b, a := source.At(3, 3).RGBA()
	if r != 0 || g != 0x7070 || b != 0xc0c0 || a != 0xffff {
		t.Fatalf("expected blue rect pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	r, g, b, a = source.At(14, 5).RGBA()
	if r != 0 || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected black circle pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	r, g, b, a = source.At(10, 19).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected red path pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	_, _, _, a = source.At(0, 0).RGBA()
	if a != 0 {
		t.Fatalf("expected transparent untouched pixel, got alpha=%04x", a)
	}
}

func TestJPEGAdobeRGBProfileDetection(t *testing.T) {
	payload := append([]byte("ICC_PROFILE\x00\x01\x01"), []byte("Adobe RGB (1998)")...)
	length := len(payload) + 2
	data := []byte{0xff, 0xd8, 0xff, 0xe2, byte(length >> 8), byte(length)}
	data = append(data, payload...)
	data = append(data, 0xff, 0xd9)
	if !jpegHasAdobeRGBProfile(data) {
		t.Fatal("expected Adobe RGB ICC profile marker to be detected")
	}
}

func TestConvertAdobeRGBImageToSRGB(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 128, G: 64, B: 32, A: 200})

	got := convertAdobeRGBImageToSRGB(source).RGBAAt(0, 0)
	want := color.RGBA{R: 146, G: 62, B: 23, A: 200}
	if got != want {
		t.Fatalf("unexpected Adobe RGB to sRGB conversion: got %#v want %#v", got, want)
	}
}

func TestPNGICCProfileExtraction(t *testing.T) {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write([]byte("profile-data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	chunkPayload := append([]byte("ICC Profile\x00\x00"), compressed.Bytes()...)
	data := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, testPNGChunk("iCCP", chunkPayload)...)
	data = append(data, testPNGChunk("IEND", nil)...)

	got, ok := pngICCProfile(data)
	if !ok || string(got) != "profile-data" {
		t.Fatalf("expected decompressed iCCP profile, got %q ok=%v", got, ok)
	}
}

func TestParseICCRGBToSRGBProfileReadsMatrixAndCurveTags(t *testing.T) {
	profile, ok := parseICCRGBToSRGBProfile(testICCProfileData())
	if !ok {
		t.Fatal("expected RGB XYZ ICC profile to parse")
	}
	if math.Abs(profile.rXYZ[0]-0.4360747) > 0.00002 || math.Abs(profile.gXYZ[1]-0.7168786) > 0.00002 || math.Abs(profile.bXYZ[2]-0.7141733) > 0.00002 {
		t.Fatalf("unexpected profile matrix: %#v %#v %#v", profile.rXYZ, profile.gXYZ, profile.bXYZ)
	}
	if got := profile.rTRC.linearize(128); math.Abs(got-(128.0/255.0)) > 0.00001 {
		t.Fatalf("expected gamma 1 TRC, got %.6f", got)
	}
}

func TestConvertICCImageToSRGB(t *testing.T) {
	profile, ok := parseICCRGBToSRGBProfile(testICCProfileData())
	if !ok {
		t.Fatal("expected RGB XYZ ICC profile to parse")
	}
	source := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	source.SetNRGBA(0, 0, color.NRGBA{R: 128, G: 64, B: 32, A: 200})

	got := convertICCImageToSRGB(source, profile).RGBAAt(0, 0)
	want := color.RGBA{R: 188, G: 137, B: 99, A: 200}
	if got != want {
		t.Fatalf("unexpected ICC conversion: got %#v want %#v", got, want)
	}
}

func TestDrawPictureRasterAppliesQuarterTurnRotation(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 2, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	source.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	dst := image.NewRGBA(image.Rect(0, 0, 2, 2))

	drawPictureRaster(dst, dst.Bounds(), source, source.Bounds(), slideElement{HasRotation: true, Rotation: 5400000}, slideSize{CX: emuPerInch, CY: emuPerInch})

	if got := dst.RGBAAt(0, 0); got.R == 0 || got.G != 0 {
		t.Fatalf("expected rotated red source pixel at top, got %#v", got)
	}
	if got := dst.RGBAAt(0, 1); got.G == 0 || got.R != 0 {
		t.Fatalf("expected rotated green source pixel at bottom, got %#v", got)
	}
}

func TestDrawPictureRasterAppliesArbitraryRotation(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	dst := image.NewRGBA(image.Rect(0, 0, 64, 64))
	target := image.Rect(22, 22, 42, 42)

	drawPictureRaster(dst, target, source, source.Bounds(), slideElement{HasRotation: true, Rotation: 2700000}, slideSize{CX: emuPerInch, CY: emuPerInch})

	bounds := opaqueBounds(dst)
	if bounds.Empty() {
		t.Fatal("expected arbitrary picture rotation to paint pixels")
	}
	if bounds.Dx() <= target.Dx() || bounds.Dy() <= target.Dy() {
		t.Fatalf("expected arbitrary rotation to expand painted bounds beyond the unrotated target, got %v target=%v", bounds, target)
	}
}

func TestRenderPictureUsesRasterFallbackForSVGBlip(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.png": redPNG(),
			"ppt/media/image2.svg": []byte(`<svg viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10" fill="#0070C0"/></svg>`),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{
				"png": "image/png",
				"svg": "image/svg+xml",
			},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	element := slideElement{
		Kind:         "pic",
		Name:         "Graphic 1",
		EmbedID:      "rId1",
		SVGEmbedID:   "rId2",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: pptx.ImageRelType, Target: "../media/image1.png"},
		"rId2": {ID: "rId2", Type: pptx.ImageRelType, Target: "../media/image2.svg"},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected fallback picture to render fully, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(5, 5).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected fallback png red pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderPictureSupportsDirectSVGImage(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/media/image1.svg": []byte(`<svg viewBox="0 0 10 10" xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10" fill="#0070C0"/></svg>`),
		},
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"svg": "image/svg+xml"},
		},
	}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	element := slideElement{
		Kind:         "pic",
		Name:         "Graphic 1",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
	}
	relationships := map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: pptx.ImageRelType, Target: "../media/image1.svg"},
	}

	unsupported := renderPicture(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element, relationships)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected direct svg picture to render fully, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	r, g, b, a := img.At(5, 5).RGBA()
	if r != 0 || g != 0x7070 || b != 0xc0c0 || a != 0xffff {
		t.Fatalf("expected svg blue pixel, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestInheritedRenderPartsOrdersMasterLayoutThenSlide(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{{
			PartName: "ppt/slides/slide1.xml",
			Text:     "Slide",
			Layout:   "Layout",
			Master:   "Master",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatal(err)
	}

	got := inheritedRenderParts(pkg, "ppt/slides/slide1.xml")
	want := []string{"ppt/slideMasters/slideMaster1.xml", "ppt/slideLayouts/slideLayout1.xml", "ppt/slides/slide1.xml"}
	if len(got) != len(want) {
		t.Fatalf("unexpected inherited part count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected inherited part order: got=%v want=%v", got, want)
		}
	}
}

func TestVisibleRenderPartsHonorsShowMasterSpFalse(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "deck.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{{
			PartName: "ppt/slides/slide1.xml",
			Text:     "Slide",
			Layout:   "Layout",
			Master:   "Master",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatal(err)
	}
	layout := "ppt/slideLayouts/slideLayout1.xml"
	pkg.Parts[layout] = bytes.Replace(pkg.Parts[layout], []byte("<p:sldLayout "), []byte(`<p:sldLayout showMasterSp="0" `), 1)

	parts := inheritedRenderParts(pkg, "ppt/slides/slide1.xml")
	got := visibleRenderParts(pkg, "ppt/slides/slide1.xml", parts)
	want := []string{"ppt/slideLayouts/slideLayout1.xml", "ppt/slides/slide1.xml"}
	if len(got) != len(want) {
		t.Fatalf("unexpected visible part count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected visible part order: got=%v want=%v", got, want)
		}
	}
	if inherited := inheritedRenderParts(pkg, "ppt/slides/slide1.xml"); len(inherited) != 3 {
		t.Fatalf("master should remain available for inheritance, got %v", inherited)
	}
}

func TestFilterInheritedPlaceholdersKeepsDecorativeObjects(t *testing.T) {
	got := filterInheritedPlaceholders([]slideElement{
		{Kind: "sp", Name: "Title Placeholder", IsPlaceholder: true},
		{Kind: "sp", Name: "Master Accent"},
	})
	if len(got) != 1 || got[0].Name != "Master Accent" {
		t.Fatalf("unexpected inherited placeholder filtering: %+v", got)
	}
}

func TestResolveSlidePlaceholdersCopiesInheritedTransform(t *testing.T) {
	elements := []slideElement{{
		Kind:            "sp",
		Name:            "Title 1",
		Text:            "Slide title",
		IsPlaceholder:   true,
		PlaceholderType: "title",
	}}
	sources := map[string]slideElement{
		"type:title": {
			IsPlaceholder:   true,
			PlaceholderType: "title",
			HasTransform:    true,
			OffX:            100,
			OffY:            200,
			ExtCX:           300,
			ExtCY:           400,
			HasInsets:       true,
			InsetLeft:       10,
			TextAlign:       "ctr",
			TextAnchor:      "ctr",
			FontSize:        2400,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if !got[0].HasTransform || got[0].OffX != 100 || got[0].ExtCY != 400 {
		t.Fatalf("placeholder transform was not resolved: %+v", got[0])
	}
	if !got[0].HasInsets || got[0].InsetLeft != 10 || got[0].TextAlign != "ctr" || got[0].TextAnchor != "ctr" || got[0].FontSize != 2400 {
		t.Fatalf("placeholder text-box properties were not resolved: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersCopiesTextPropertiesWithLocalTransform(t *testing.T) {
	elements := []slideElement{{
		Kind:            "sp",
		Name:            "Title 1",
		Text:            "Slide title",
		IsPlaceholder:   true,
		PlaceholderType: "ctrTitle",
		HasTransform:    true,
		OffX:            10,
		OffY:            20,
		ExtCX:           30,
		ExtCY:           40,
		FontSize:        4400,
	}}
	sources := map[string]slideElement{
		"type:ctrTitle": {
			IsPlaceholder:   true,
			PlaceholderType: "ctrTitle",
			HasTransform:    true,
			OffX:            100,
			OffY:            200,
			ExtCX:           300,
			ExtCY:           400,
			TextAnchor:      "b",
			FontSize:        6000,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].OffX != 10 || got[0].OffY != 20 || got[0].ExtCX != 30 || got[0].ExtCY != 40 {
		t.Fatalf("local placeholder transform was unexpectedly overwritten: %+v", got[0])
	}
	if got[0].TextAnchor != "b" {
		t.Fatalf("missing text anchor was not inherited: %+v", got[0])
	}
	if got[0].FontSize != 4400 {
		t.Fatalf("explicit slide font size was unexpectedly overwritten: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersAppliesInheritedPlaceholderParagraphStyle(t *testing.T) {
	elements := []slideElement{{
		Kind:            "sp",
		Name:            "Title 1",
		Text:            "Slide title",
		IsPlaceholder:   true,
		PlaceholderType: "title",
		TextParagraphs:  []textParagraph{{Text: "Slide title"}},
	}}
	sources := map[string]slideElement{
		"type:title": {
			IsPlaceholder:   true,
			PlaceholderType: "title",
			FontSize:        6000,
			PlaceholderParagraphStyles: map[int]paragraphStyle{
				0: {
					FontSize:     6000,
					HasTextColor: true,
					TextColor:    color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff},
				},
			},
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	got = applyInheritedTextStyles(got, map[string]textStyle{
		"title": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {FontSize: 1400},
			},
		},
	})
	if got[0].TextParagraphs[0].FontSize != 6000 {
		t.Fatalf("layout placeholder paragraph style should win over master title style: %+v", got[0].TextParagraphs[0])
	}
	if !got[0].TextParagraphs[0].HasTextColor || got[0].TextParagraphs[0].TextColor.R != 0x88 {
		t.Fatalf("inherited placeholder paragraph color was not applied: %+v", got[0].TextParagraphs[0])
	}
}

func TestResolveSlidePlaceholdersKeepsLocalBodyProperties(t *testing.T) {
	elements := []slideElement{{
		Kind:              "sp",
		Name:              "Content Placeholder 1",
		Text:              "Slide body",
		IsPlaceholder:     true,
		PlaceholderType:   "body",
		HasBodyProperties: true,
		HasNormAutofit:    true,
	}}
	sources := map[string]slideElement{
		"type:body": {
			IsPlaceholder:           true,
			PlaceholderType:         "body",
			HasInsets:               true,
			InsetLeft:               10,
			TextAnchor:              "b",
			HasShapeAutofit:         true,
			HasNormAutofit:          true,
			FontScalePct:            85000,
			LineSpacingReductionPct: 20000,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].TextAnchor != "b" || !got[0].HasInsets || got[0].InsetLeft != 10 {
		t.Fatalf("missing local anchor should inherit placeholder anchor while preserving inherited insets: %+v", got[0])
	}
	if got[0].HasShapeAutofit || got[0].FontScalePct != 0 || got[0].LineSpacingReductionPct != 0 {
		t.Fatalf("local bodyPr should block inherited body properties: %+v", got[0])
	}
	if !got[0].HasNormAutofit {
		t.Fatalf("local normal autofit should be preserved: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersInheritsUnspecifiedFirstLastSpacing(t *testing.T) {
	elements := []slideElement{{
		Kind:              "sp",
		Name:              "Content Placeholder 1",
		Text:              "Slide body",
		IsPlaceholder:     true,
		PlaceholderType:   "body",
		HasBodyProperties: true,
	}}
	sources := map[string]slideElement{
		"type:body": {
			IsPlaceholder:           true,
			PlaceholderType:         "body",
			HasFirstLastSpacing:     true,
			IncludeFirstLastSpacing: true,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if !got[0].HasFirstLastSpacing || !got[0].IncludeFirstLastSpacing {
		t.Fatalf("unspecified first/last paragraph spacing was not inherited: %+v", got[0])
	}

	elements[0].HasFirstLastSpacing = true
	elements[0].IncludeFirstLastSpacing = false
	got = resolveSlidePlaceholders(elements, sources)
	if !got[0].HasFirstLastSpacing || got[0].IncludeFirstLastSpacing {
		t.Fatalf("explicit false first/last paragraph spacing should block inheritance: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersDefaultsLocalCenterTitleBodyAnchor(t *testing.T) {
	elements := []slideElement{{
		Kind:              "sp",
		Name:              "Title 1",
		Text:              "Slide title",
		IsPlaceholder:     true,
		PlaceholderType:   "ctrTitle",
		HasBodyProperties: true,
		HasNormAutofit:    true,
		FontScalePct:      90000,
	}}
	sources := map[string]slideElement{
		"type:ctrTitle": {
			IsPlaceholder:   true,
			PlaceholderType: "ctrTitle",
			TextAnchor:      "b",
			HasShapeAutofit: true,
			FontScalePct:    85000,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].TextAnchor != "b" {
		t.Fatalf("missing local center-title anchor should inherit placeholder anchor: %+v", got[0])
	}
	if got[0].HasShapeAutofit || got[0].FontScalePct != 90000 {
		t.Fatalf("local center title bodyPr should still block inherited autofit properties: %+v", got[0])
	}
	if !got[0].HasNormAutofit {
		t.Fatalf("local normal autofit should be preserved: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersInheritsCenterTitleAnchorWithoutLocalFontScale(t *testing.T) {
	elements := []slideElement{{
		Kind:              "sp",
		Name:              "Title 1",
		Text:              "Slide title",
		IsPlaceholder:     true,
		PlaceholderType:   "ctrTitle",
		HasBodyProperties: true,
		HasNormAutofit:    true,
	}}
	sources := map[string]slideElement{
		"type:ctrTitle": {
			IsPlaceholder:   true,
			PlaceholderType: "ctrTitle",
			TextAnchor:      "b",
			HasShapeAutofit: true,
			FontScalePct:    85000,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].TextAnchor != "b" {
		t.Fatalf("center title without local font scale should inherit placeholder anchor: %+v", got[0])
	}
	if got[0].HasShapeAutofit || got[0].FontScalePct != 0 {
		t.Fatalf("local center title bodyPr should still block inherited autofit properties: %+v", got[0])
	}
}

func TestMergePlaceholderSourceKeepsMasterTextWhenLayoutOmitsIt(t *testing.T) {
	master := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "title",
		HasTransform:    true,
		OffX:            100,
		ExtCY:           500,
		ExtCX:           300,
		FontSize:        4400,
		HasTextColor:    true,
		TextColor:       color.RGBA{R: 12, G: 34, B: 56, A: 255},
		TextAnchor:      "ctr",
	}
	layout := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "title",
		HasTransform:    true,
		OffX:            200,
		ExtCX:           400,
		ExtCY:           600,
	}

	got := mergePlaceholderSource(master, layout)
	if got.OffX != 200 || got.ExtCX != 400 {
		t.Fatalf("layout transform was not preserved: %+v", got)
	}
	if got.FontSize != 4400 || !got.HasTextColor || got.TextColor.R != 12 || got.TextAnchor != "ctr" {
		t.Fatalf("master text properties were not retained: %+v", got)
	}
}

func TestMergePlaceholderSourceMergesParagraphStyles(t *testing.T) {
	master := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "title",
		PlaceholderParagraphStyles: map[int]paragraphStyle{
			0: {
				FontSize:     4400,
				HasTextColor: true,
				TextColor:    color.RGBA{R: 12, G: 34, B: 56, A: 255},
			},
		},
	}
	layout := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "title",
		PlaceholderParagraphStyles: map[int]paragraphStyle{
			0: {FontSize: 6000},
		},
	}

	got := mergePlaceholderSource(master, layout)
	style := got.PlaceholderParagraphStyles[0]
	if style.FontSize != 6000 || !style.HasTextColor || style.TextColor.R != 12 {
		t.Fatalf("expected layout paragraph font size with retained master color, got %+v", style)
	}
}

func TestResolveSlidePlaceholdersMatchesInheritedIdxFallback(t *testing.T) {
	elements := []slideElement{{
		Kind:           "sp",
		Name:           "Content Placeholder 2",
		Text:           "Body",
		IsPlaceholder:  true,
		PlaceholderIdx: "1",
	}}
	sources := map[string]slideElement{
		"type:body": {
			IsPlaceholder:   true,
			PlaceholderType: "body",
			PlaceholderIdx:  "1",
			HasTransform:    true,
			OffX:            100,
			ExtCX:           300,
			ExtCY:           400,
		},
		"idx:1": {
			IsPlaceholder:   true,
			PlaceholderType: "body",
			PlaceholderIdx:  "1",
			HasTransform:    true,
			OffX:            100,
			ExtCX:           300,
			ExtCY:           400,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if !got[0].HasTransform || got[0].OffX != 100 || got[0].ExtCX != 300 {
		t.Fatalf("placeholder idx fallback was not resolved: %+v", got[0])
	}
}

func TestParseSlideElementPlaceholderKey(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p"><p:nvSpPr><p:cNvPr id="2" name="Title 1"/><p:nvPr><p:ph type="title" idx="1"/></p:nvPr></p:nvSpPr></p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.IsPlaceholder || got.PlaceholderType != "title" || got.PlaceholderIdx != "1" {
		t.Fatalf("unexpected placeholder parse: %+v", got)
	}
}

func TestParseSlideElementAppliesStyleFillAndLineFallbacks(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Right Arrow 1"/></p:nvSpPr>
	  <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="1"/></a:xfrm><a:prstGeom prst="rightArrow"><a:avLst/></a:prstGeom></p:spPr>
  <p:style><a:lnRef idx="1"><a:schemeClr val="dk1"/></a:lnRef><a:fillRef idx="1"><a:schemeClr val="accent1"/></a:fillRef><a:fontRef idx="minor"><a:schemeClr val="lt1"/></a:fontRef></p:style>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.HasFill || got.FillColor.A != 255 || !got.HasLine || got.LineColor.A != 255 || got.FontFamily != "+mn-lt" || !got.HasTextColor || got.TextColor != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("expected style fill, line, and font fallbacks, got %+v", got)
	}
}

func TestParseSlideElementResolvesThemeGradientFillRef(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Gradient Rect"/></p:nvSpPr>
	  <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="1"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr>
	  <p:style><a:fillRef idx="2"><a:schemeClr val="accent1"/></a:fillRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	fillStyles := parseThemeFillStyles([]byte(`<a:theme xmlns:a="a"><a:themeElements><a:fmtScheme name="Office"><a:fillStyleLst>
	  <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
	  <a:gradFill><a:gsLst><a:gs pos="0"><a:schemeClr val="phClr"/></a:gs><a:gs pos="100000"><a:srgbClr val="000000"/></a:gs></a:gsLst><a:lin ang="5400000" scaled="0"/></a:gradFill>
	</a:fillStyleLst></a:fmtScheme></a:themeElements></a:theme>`))
	got := parseSlideElementNodeWithThemeEffectsAndFills(root, renderTransform{ScaleX: 1, ScaleY: 1}, themeColors{"accent1": {R: 200, G: 100, B: 50, A: 255}}, themeEffectStyles{}, fillStyles)
	if !got.HasFill || !got.HasFillGradient || len(got.FillGradient.Stops) != 2 {
		t.Fatalf("expected style fillRef to resolve theme gradient, got %+v", got)
	}
	if got.FillGradient.Stops[0].Color != (color.RGBA{R: 200, G: 100, B: 50, A: 255}) {
		t.Fatalf("expected fillRef placeholder color to seed gradient, got %+v", got.FillGradient.Stops[0].Color)
	}
}

func TestParseSlideElementKeepsExplicitNoLineOverStyleFallback(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="No Outline Shape"/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/><a:ln><a:noFill/></a:ln></p:spPr>
	  <p:style><a:lnRef idx="1"><a:schemeClr val="accent1"/></a:lnRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNodeWithTheme(root, renderTransform{ScaleX: 1, ScaleY: 1}, themeColors{"accent1": {R: 10, G: 20, B: 30, A: 255}})
	if !got.NoLine || got.HasLine {
		t.Fatalf("explicit noFill line should suppress style lnRef fallback, got %+v", got)
	}
}

func TestParseSlideElementDoesNotInferAlignmentFromLeadingSpaces(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Manual Centered Title"/></p:nvSpPr>
	  <p:txBody><a:bodyPr/><a:p><a:r><a:t>          Centered title</a:t></a:r></a:p></p:txBody>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.TextAlign != "" || got.Text != "Centered title" || len(got.TextParagraphs) != 1 || len(got.TextParagraphs[0].Runs) != 1 || got.TextParagraphs[0].Runs[0].Text != "          Centered title" {
		t.Fatalf("expected leading spaces to remain text content only, got %+v", got)
	}
}

func TestParseSlideElementReadsLineEndMarkers(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:cxnSp xmlns:p="p" xmlns:a="a">
  <p:nvCxnSpPr><p:cNvPr id="2" name="Arrow Connector"/></p:nvCxnSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="0"/></a:xfrm>
    <a:prstGeom prst="straightConnector1"><a:avLst/></a:prstGeom>
    <a:ln><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:headEnd type="triangle" w="sm" len="sm"/><a:tailEnd type="triangle" w="lg" len="lg"/></a:ln>
  </p:spPr>
</p:cxnSp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.HasLineMarker || got.HeadLineMarker != "triangle" || got.TailLineMarker != "triangle" {
		t.Fatalf("expected parsed triangle line markers, got %+v", got)
	}
	if got.HeadLineMarkerWidth != "sm" || got.HeadLineMarkerLength != "sm" || got.TailLineMarkerWidth != "lg" || got.TailLineMarkerLength != "lg" {
		t.Fatalf("expected parsed line marker sizes, got %+v", got)
	}
}

func TestParseSlideElementReadsOuterShadowEffect(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Shadowed Rectangle"/></p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="1"/></a:xfrm>
    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
    <a:effectLst>
      <a:outerShdw blurRad="50800" dist="38100" dir="2700000" algn="tl" rotWithShape="0">
        <a:prstClr val="black"><a:alpha val="40000"/></a:prstClr>
      </a:outerShdw>
    </a:effectLst>
  </p:spPr>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.HasShadow || got.ShadowBlur != 50800 || got.ShadowDistance != 38100 || got.ShadowDirection != 2700000 {
		t.Fatalf("unexpected parsed shadow metadata: %+v", got)
	}
	if got.ShadowColor.A != 102 {
		t.Fatalf("expected alpha-modified shadow color, got %#v", got.ShadowColor)
	}
}

func TestParseTextStylesReadsMasterTitleDefaults(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
	  <p:txStyles>
    <p:titleStyle>
      <a:lvl1pPr algn="ctr">
        <a:defRPr sz="4400" b="1">
          <a:solidFill><a:srgbClr val="0070C0"/></a:solidFill>
        </a:defRPr>
      </a:lvl1pPr>
    </p:titleStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())
	got, ok := styles["ctrTitle"]
	if !ok {
		t.Fatalf("expected ctrTitle style, got %+v", styles)
	}
	if got.FontSize != 4400 || !got.Bold || got.TextAlign != "ctr" || !got.HasTextColor || got.TextColor.R != 0x00 || got.TextColor.G != 0x70 || got.TextColor.B != 0xc0 {
		t.Fatalf("unexpected parsed title style: %+v", got)
	}
}

func TestApplyInheritedTextStylesDoesNotOverrideExplicitRunSize(t *testing.T) {
	elements := []slideElement{{
		Text:            "Title",
		IsPlaceholder:   true,
		PlaceholderType: "ctrTitle",
		FontSize:        3200,
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"ctrTitle": {
			FontSize:     4400,
			HasTextColor: true,
			TextColor:    color.RGBA{B: 255, A: 255},
			TextAlign:    "ctr",
		},
	})
	if got[0].FontSize != 3200 {
		t.Fatalf("inherited style overrode explicit font size: %+v", got[0])
	}
	if !got[0].HasTextColor || got[0].TextColor.B != 255 || got[0].TextAlign != "ctr" {
		t.Fatalf("inherited missing properties were not applied: %+v", got[0])
	}
}

func TestApplyInheritedTextStylesAppliesTitleButSkipsBodyPlaceholders(t *testing.T) {
	elements := []slideElement{
		{
			Text:            "Title",
			TextParagraphs:  []textParagraph{{Text: "Title"}},
			IsPlaceholder:   true,
			PlaceholderType: "title",
		},
		{
			Text:            "Body",
			IsPlaceholder:   true,
			PlaceholderType: "body",
		},
	}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"title": {
			FontSize:     4400,
			Bold:         true,
			HasTextColor: true,
			TextColor:    color.RGBA{B: 255, A: 255},
			ParagraphStyles: map[int]paragraphStyle{
				0: {HasLineSpacing: true, LineSpacingPct: 90000},
			},
		},
		"body": {
			FontSize:     2800,
			HasTextColor: true,
			TextColor:    color.RGBA{R: 255, A: 255},
		},
	})
	if got[0].FontSize != 4400 || !got[0].HasTextColor || got[0].TextColor.B != 255 || !got[0].TextParagraphs[0].Bold || got[0].TextParagraphs[0].LineSpacingPct != 90000 {
		t.Fatalf("title placeholder was not styled by inherited title fallback: %+v", got)
	}
	if got[1].FontSize != 0 || got[1].HasTextColor {
		t.Fatalf("body placeholder was unexpectedly styled by title-only fallback: %+v", got)
	}
}

func TestParseBodyPropertiesReadsTextAnchor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a" anchor="ctr" wrap="square" vert="eaVert" rot="5400000" numCol="2" anchorCtr="1" spcFirstLastPara="1"><a:spAutoFit/><a:normAutofit fontScale="85000" lnSpcReduction="20000"/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if element.TextAnchor != "ctr" || element.TextWrap != "square" {
		t.Fatalf("unexpected body properties: %+v", element)
	}
	if element.TextVertical != "eaVert" || !element.HasTextBodyRotation || element.TextBodyRotation != 5400000 || element.TextColumnCount != 2 || !element.TextAnchorCenter {
		t.Fatalf("expected text layout body properties: %+v", element)
	}
	if !element.HasFirstLastSpacing || !element.IncludeFirstLastSpacing {
		t.Fatalf("expected first/last paragraph spacing flag: %+v", element)
	}
	if !element.HasNormAutofit {
		t.Fatalf("expected normal autofit to be detected: %+v", element)
	}
	if !element.HasShapeAutofit {
		t.Fatalf("expected shape autofit to be detected: %+v", element)
	}
	if !element.HasFontScalePct || element.FontScalePct != 85000 {
		t.Fatalf("unexpected autofit font scale: %+v", element)
	}
	if !element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 20000 {
		t.Fatalf("unexpected autofit line spacing reduction: %+v", element)
	}
}

func TestParseBodyPropertiesReadsExplicitFirstLastSpacingOff(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a" spcFirstLastPara="0"/>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasFirstLastSpacing || element.IncludeFirstLastSpacing {
		t.Fatalf("expected explicit false first/last paragraph spacing flag: %+v", element)
	}
}

func TestParseBodyPropertiesReadsNoAutofitChoice(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a"><a:noAutofit/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasBodyProperties || !element.HasNoAutofit {
		t.Fatalf("expected explicit DrawingML noAutofit state, got %+v", element)
	}
	if element.HasShapeAutofit || element.HasNormAutofit || element.HasFontScalePct || element.FontScalePct != 0 || element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 0 {
		t.Fatalf("noAutofit should not leave active autofit properties, got %+v", element)
	}
}

func TestParseBodyPropertiesNoAutofitSuppressesOtherAutofitChoices(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a"><a:spAutoFit/><a:normAutofit fontScale="85000" lnSpcReduction="20000"/><a:noAutofit/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasNoAutofit {
		t.Fatalf("expected explicit noAutofit state, got %+v", element)
	}
	if element.HasShapeAutofit || element.HasNormAutofit || element.HasFontScalePct || element.FontScalePct != 0 || element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 0 {
		t.Fatalf("noAutofit should win over other malformed autofit choices, got %+v", element)
	}
}

func TestRenderShapeDoesNotReportSimpleTextAsSimplifiedLayout(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Label",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		Text:         "Ready",
		FontSize:     1800,
		TextParagraphs: []textParagraph{{
			Text:     "Ready",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Ready", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected simple text to render without partial scaffold, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestRenderShapeReportsSpecificUnsupportedTextLayoutFeatures(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                "sp",
		Name:                "Vertical Text",
		PrstGeom:            "rect",
		HasTransform:        true,
		ExtCX:               emuPerInch,
		ExtCY:               emuPerInch,
		Text:                "Ready",
		FontSize:            1800,
		TextVertical:        "eaVert",
		HasTextBodyRotation: true,
		TextBodyRotation:    5400000,
		TextColumnCount:     2,
		TextAnchorCenter:    true,
		TextParagraphs: []textParagraph{{
			Text:     "Ready",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Ready", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if !element.Rendered {
		t.Fatalf("expected text to still render best-effort, got rendered=%v", element.Rendered)
	}
	got := unsupportedMessages(unsupported)
	for _, want := range []string{"vertical mode", "rotation", "columns", "anchor-center"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in unsupported messages, got %s", want, got)
		}
	}
	if strings.Contains(got, "simplified layout") {
		t.Fatalf("blanket simplified-layout scaffold should not be emitted: %s", got)
	}
}

func TestRenderShapeReportsSimplifiedAutofitTextSizing(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	text := strings.Repeat("W", 1000)
	element := slideElement{
		Kind:           "sp",
		Name:           "Autofit Text",
		PrstGeom:       "rect",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		ExtCY:          emuPerInch,
		Text:           text,
		TextWrap:       "none",
		FontSize:       1800,
		HasNormAutofit: true,
		TextParagraphs: []textParagraph{{
			Text:     text,
			FontSize: 1800,
			Runs:     []textRun{{Text: text, FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if !element.Rendered {
		t.Fatalf("expected autofit text to still render best-effort, got rendered=%v", element.Rendered)
	}
	got := unsupportedMessages(unsupported)
	if !strings.Contains(got, "normal autofit was rendered with simplified sizing") {
		t.Fatalf("expected normal autofit partial diagnostic, got %s", got)
	}
}

func TestRenderShapeDoesNotReportImplicitNormalAutofitWhenTextAlreadyFits(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "sp",
		Name:           "Fitting Autofit Text",
		PrstGeom:       "rect",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		ExtCY:          emuPerInch,
		Text:           "Ready",
		FontSize:       1800,
		HasNormAutofit: true,
		TextParagraphs: []textParagraph{{
			Text:     "Ready",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Ready", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	got := unsupportedMessages(unsupported)
	if strings.Contains(got, "normal autofit was rendered with simplified sizing") {
		t.Fatalf("fitting implicit normal-autofit text should not require simplified sizing, got %s", got)
	}
}

func TestRenderShapeDoesNotReportAuthoredNormalAutofitScaleAsSimplified(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                       "sp",
		Name:                       "Authored Autofit Text",
		PrstGeom:                   "rect",
		HasTransform:               true,
		ExtCX:                      emuPerInch,
		ExtCY:                      emuPerInch,
		Text:                       "Ready",
		FontSize:                   1800,
		HasNormAutofit:             true,
		HasFontScalePct:            true,
		FontScalePct:               85000,
		HasLineSpacingReductionPct: true,
		LineSpacingReductionPct:    20000,
		TextParagraphs: []textParagraph{{
			Text:     "Ready",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Ready", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	got := unsupportedMessages(unsupported)
	if strings.Contains(got, "normal autofit was rendered with simplified sizing") {
		t.Fatalf("authored normal-autofit scale should be rendered directly, got %s", got)
	}
}

func TestRenderShapeReportsUnsupportedRotatedShapeAutofit(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:            "sp",
		Name:            "Rotated Autofit Text",
		PrstGeom:        "rect",
		HasTransform:    true,
		ExtCX:           emuPerInch,
		ExtCY:           emuPerInch,
		Rotation:        5400000,
		Text:            "Ready",
		FontSize:        1800,
		HasShapeAutofit: true,
		TextParagraphs: []textParagraph{{
			Text:     "Ready",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Ready", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	got := unsupportedMessages(unsupported)
	if !strings.Contains(got, "shape autofit was rendered with simplified sizing") {
		t.Fatalf("expected rotated shape autofit partial diagnostic, got %s", got)
	}
}

func TestRenderShapeTreatsKnownSymbolFontBulletMappingsAsSupported(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Symbol Bullets",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		Text:         "Specificity",
		FontSize:     1800,
		TextParagraphs: []textParagraph{{
			Text:             "Specificity",
			Bullet:           "¬",
			BulletFontFamily: "Wingdings",
			FontSize:         1800,
			Runs:             []textRun{{Text: "Specificity", FontSize: 1800}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if !element.Rendered {
		t.Fatalf("expected symbol-bullet text to still render best-effort, got rendered=%v", element.Rendered)
	}
	got := unsupportedMessages(unsupported)
	if strings.Contains(got, "symbol font bullets were rendered with Unicode substitutes") {
		t.Fatalf("known symbol-font bullet mapping should not report unsupported content, got %s", got)
	}
}

func unsupportedMessages(items []model.SkipItem) string {
	messages := make([]string, 0, len(items))
	for _, item := range items {
		messages = append(messages, item.Message)
	}
	return strings.Join(messages, "; ")
}

func TestShapeAutofitTargetExpandsHeightForText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		Text:            "First\nSecond",
		FontSize:        4800,
		TextParagraphs: []textParagraph{{
			Text:     "First",
			FontSize: 4800,
			Runs:     []textRun{{Text: "First", FontSize: 4800}},
		}, {
			Text:     "Second",
			FontSize: 4800,
			Runs:     []textRun{{Text: "Second", FontSize: 4800}},
		}},
	}, image.Rect(10, 20, 210, 30), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dy() <= 10 {
		t.Fatalf("expected shape target to grow, got %+v", got)
	}
	if got.Min.Y != 20 || got.Min.X != 10 || got.Max.X != 210 {
		t.Fatalf("unexpected horizontal or top adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetShrinksHeightToText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		Text:            "Short",
		FontSize:        1800,
		TextParagraphs: []textParagraph{{
			Text:     "Short",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Short", FontSize: 1800}},
		}},
	}, image.Rect(10, 20, 210, 220), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dy() >= 200 || got.Dy() <= 0 {
		t.Fatalf("expected shape target to shrink to measured text, got %+v", got)
	}
	if got.Min.Y != 20 || got.Min.X != 10 || got.Max.X != 210 {
		t.Fatalf("unexpected horizontal or top adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetExpandsNoWrapWidthForText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		TextWrap:        "none",
		Text:            "This heading is intentionally wider than the original box",
		FontSize:        2400,
		TextParagraphs: []textParagraph{{
			Text:     "This heading is intentionally wider than the original box",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This heading is intentionally wider than the original box", FontSize: 2400}},
		}},
	}, image.Rect(10, 20, 90, 70), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dx() <= 80 {
		t.Fatalf("expected no-wrap shape target to grow horizontally, got %+v", got)
	}
	if got.Min.X != 10 || got.Min.Y != 20 {
		t.Fatalf("unexpected top-left adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetDoesNotExpandWrappedWidth(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		TextWrap:        "square",
		Text:            "This heading is intentionally wider than the original box",
		FontSize:        2400,
		TextParagraphs: []textParagraph{{
			Text:     "This heading is intentionally wider than the original box",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This heading is intentionally wider than the original box", FontSize: 2400}},
		}},
	}, image.Rect(10, 20, 90, 70), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Min.X != 10 || got.Max.X != 90 {
		t.Fatalf("expected wrapped shape target to preserve horizontal bounds, got %+v", got)
	}
}

func TestFitNormalAutofitElementScalesTextToBounds(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:  true,
		PlaceholderType: "title",
		FontScalePct:    90000,
		FontSize:        4000,
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 420, 65))
	if got.FontScalePct == 0 || got.FontScalePct >= 90000 {
		t.Fatalf("expected normal autofit to select a reduced font scale, got %+v", got)
	}
}

func TestFitNormalAutofitElementUsesAuthoredScaleAsProbeStart(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:  true,
		HasFontScalePct: true,
		FontScalePct:    90000,
		FontFamily:      "Carlito",
		FontSize:        4000,
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 420, 65))
	if got.FontScalePct == 0 || got.FontScalePct > 90000 {
		t.Fatalf("authored normal-autofit fontScale should cap the probe start, got %+v", got)
	}
}

func TestFitNormalAutofitElementCanScaleBelowFiftyPercent(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit: true,
		FontFamily:     "Carlito",
		FontSize:       4000,
		TextWrap:       "none",
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 180, 30))
	if got.FontScalePct >= 50000 || got.FontScalePct < minimumNormalAutofitFontScalePct {
		t.Fatalf("expected normal autofit to use the supported scale range below 50%%, got %+v", got)
	}
	if !textFitsAtScale(got, image.Rect(0, 0, 180, 30), got.FontScalePct, normalAutofitMaxSoftLines(got), defaultOutputDPI) {
		t.Fatalf("selected scale should fit in the target bounds, got %+v", got)
	}
}

func TestFitNormalAutofitElementSkipsLineSpacingReductionWhenTextFitsVertically(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:             true,
		HasLineSpacingReductionPct: true,
		LineSpacingReductionPct:    10000,
		FontFamily:                 "Carlito",
		FontSize:                   2400,
		TextWrap:                   "square",
		TextParagraphs: []textParagraph{{
			Text:           "Short body",
			FontSize:       2400,
			LineSpacingPct: 90000,
			Runs: []textRun{{
				Text:     "Short body",
				FontSize: 2400,
			}},
		}},
	}, image.Rect(0, 0, 500, 100))
	if got.HasLineSpacingReductionPct || got.LineSpacingReductionPct != 0 {
		t.Fatalf("line spacing reduction should be unused when text already fits vertically: %+v", got)
	}
}

func TestNormalAutofitMaxSoftLinesHonorsWrapNoneAndHardBreaks(t *testing.T) {
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "square",
		Text:     "Single line title",
		TextParagraphs: []textParagraph{{
			Text: "Single line title",
			Runs: []textRun{{Text: "Single line title"}},
		}},
	}); got != 0 {
		t.Fatalf("wrapping text without hard breaks should not cap soft lines, got %d", got)
	}
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "none",
		Text:     "Single line title",
		TextParagraphs: []textParagraph{{
			Text: "Single line title",
			Runs: []textRun{{Text: "Single line title"}},
		}},
	}); got != 1 {
		t.Fatalf(`expected wrap="none" text without hard breaks to require single-line fit, got %d`, got)
	}
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "square",
		Text:     "Line one\nLine two",
		TextParagraphs: []textParagraph{{
			Text: "Line one\nLine two",
			Runs: []textRun{{Text: "Line one\nLine two"}},
		}},
	}); got != 2 {
		t.Fatalf("hard breaks should cap normal-autofit soft lines to authored line count, got %d", got)
	}
}

func TestTextRenderLinesPreserveDrawingMLBreakRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:bodyPr><a:normAutofit/></a:bodyPr><a:lstStyle/><a:p><a:r><a:rPr sz="4400"/><a:t> Welcome to </a:t></a:r><a:br><a:rPr sz="4400"/></a:br><a:r><a:rPr sz="4400"/><a:t>GENERATE: The Game of Energy Choices</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := textParagraphsFromNode(root)
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(4400, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(4400, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{TextParagraphs: paragraphs, FontSize: 4400}, 900)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected DrawingML break to create two render lines, got %d: %+v", len(lines), lines)
	}
	if !strings.Contains(lines[1].Text, "GENERATE") {
		t.Fatalf("expected second line to preserve following run text, got %+v", lines[1])
	}
}

func TestFitNormalAutofitAllowsWrappingWithinHardBreakLines(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit: true,
		FontScalePct:   90000,
		FontFamily:     "Carlito",
		FontSize:       4000,
		Text:           "Residual Risk and Technology Review of\nSurface Coating NESHAP",
		TextParagraphs: []textParagraph{{
			Text:     "Residual Risk and Technology Review of\nSurface Coating NESHAP",
			FontSize: 4000,
			Runs: []textRun{
				{Text: "Residual Risk and Technology Review of ", FontSize: 4000},
				{Text: "\n", FontSize: 4000},
				{Text: "Surface Coating NESHAP", FontSize: 4000},
			},
		}},
	}, image.Rect(0, 0, 520, 150))
	if got.FontScalePct == 90000 {
		t.Fatalf("hard-break segments that soft-wrap should trigger normal autofit scaling, got %+v", got)
	}
}

func TestFitNormalAutofitDoesNotMutateParagraphFontSizes(t *testing.T) {
	element := slideElement{
		HasNormAutofit: true,
		FontScalePct:   90000,
		FontFamily:     "Carlito",
		FontSize:       4000,
		TextParagraphs: []textParagraph{{
			TextAlign: "ctr",
			FontSize:  4000,
			Runs: []textRun{
				{Text: "Residual Risk and Technology Review of ", FontSize: 4000},
				{Text: "\n", FontSize: 4000},
				{Text: "Surface Coating NESHAP ", FontSize: 4000},
				{Text: "\n"},
			},
		}},
	}

	got := fitNormalAutofitElement(element, image.Rect(0, 0, 700, 200))
	if element.TextParagraphs[0].FontSize != 4000 || element.TextParagraphs[0].Runs[0].FontSize != 4000 {
		t.Fatalf("normal-autofit probing mutated source text sizes: %+v", element.TextParagraphs[0])
	}

	scaled := scaledTextElement(got)
	if scaled.FontSize != 3600 || scaled.TextParagraphs[0].FontSize != 3600 || scaled.TextParagraphs[0].Runs[0].FontSize != 3600 {
		t.Fatalf("expected explicit 90%% normal-autofit to scale text once, got element=%+v paragraph=%+v", scaled, scaled.TextParagraphs[0])
	}
}

func TestScaleParagraphSpacingForDPIDoesNotMutateSourceParagraphs(t *testing.T) {
	element := slideElement{TextParagraphs: []textParagraph{{
		SpaceBefore: 9,
		SpaceAfter:  18,
		TabStops:    []int64{914400},
		Runs:        []textRun{{Text: "Title", FontSize: 2400}},
	}}}

	got := scaleParagraphSpacingForDPI(element, 96)
	if element.TextParagraphs[0].SpaceBefore != 9 || element.TextParagraphs[0].SpaceAfter != 18 || element.TextParagraphs[0].Runs[0].FontSize != 2400 {
		t.Fatalf("DPI spacing scaling mutated source paragraphs: %+v", element.TextParagraphs[0])
	}
	got.TextParagraphs[0].Runs[0].FontSize = 1200
	got.TextParagraphs[0].TabStops[0] = 1
	if element.TextParagraphs[0].Runs[0].FontSize != 2400 || element.TextParagraphs[0].TabStops[0] != 914400 {
		t.Fatalf("DPI spacing scaling reused nested paragraph slices: %+v", element.TextParagraphs[0])
	}
}

func TestMeasuredTextHeightIncludesInkExtentsWhenLineSpacingIsTight(t *testing.T) {
	got := measuredTextHeight([]measuredTextLine{
		{Ascent: 39, Descent: 10, Height: 36},
		{Ascent: 39, Descent: 10, Height: 36},
		{Ascent: 39, Descent: 10, Height: 36},
	})
	if got != 121 {
		t.Fatalf("expected ink extents to exceed tight line advances, got %d", got)
	}
}

func TestGoogleTitleNormalAutofitMetricsIncludeInkExtents(t *testing.T) {
	element := slideElement{
		HasNormAutofit:          true,
		FontFamily:              "Calibri",
		FontSize:                4000,
		TextAnchor:              "b",
		IncludeFirstLastSpacing: true,
		TextParagraphs: []textParagraph{{
			TextAlign:        "ctr",
			FontFamily:       "Calibri",
			FontSize:         4000,
			LineSpacingPct:   90000,
			HasLineSpacing:   true,
			HasSpaceBefore:   true,
			HasSpaceAfter:    true,
			BulletFontFamily: "Calibri",
			NoBullet:         true,
			Runs: []textRun{{
				Text:       "Emissions from Residential Wood Combustion: A Snapshot of Real-Time Monitoring in Rural Oregon",
				FontSize:   4000,
				FontFamily: "Calibri",
			}},
		}},
	}
	bounds := image.Rect(0, 91, 720, 276)
	element = fitNormalAutofitElement(element, bounds)
	scaled := scaledTextElement(element)
	faces := newFontFaceCache(false, scaled.FontFamily)
	defer faces.Close()
	face, err := faces.Get(scaled.FontSize, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(scaled.FontSize, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, scaled, bounds.Dx())
	if err != nil {
		t.Fatal(err)
	}
	measured, err := measureTextRenderLines(faces, lines, scaled.FontSize)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected residential title to wrap to three lines, got %+v", lines)
	}
	if got := measuredTextHeight(measured); got <= 108 {
		t.Fatalf("expected measured text block to include ink extents beyond line advances, got height=%d measured=%+v", got, measured)
	}
}

func TestDrawShapeTextDoesNotDropBottomAnchoredLineByBaseline(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 44))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		FontFamily: "Carlito",
		FontSize:   2400,
		TextAnchor: "b",
		TextParagraphs: []textParagraph{{
			Runs: []textRun{
				{Text: "First line", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "\n", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "Second line", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
			},
		}},
	}
	if err := drawShapeTextWithDPI(img, img.Bounds(), element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	if got := countNonWhitePixelsBelow(img, 26); got == 0 {
		t.Fatal("expected bottom-anchored second line to render when its line box intersects the bounds")
	}
}

func TestDrawShapeTextClipsGlyphsToTextBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 220, 70))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		FontFamily: "Carlito",
		FontSize:   3200,
		TextAnchor: "b",
		TextParagraphs: []textParagraph{{
			Runs: []textRun{
				{Text: "First", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "\n", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "Second", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
			},
		}},
	}
	bounds := image.Rect(0, 0, 220, 44)
	if err := drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	if got := countNonWhitePixelsBelow(img, 44); got != 0 {
		t.Fatalf("expected text drawing to be clipped at the text bounds, got %d painted pixel(s) below", got)
	}
	if got := countNonWhitePixelsBelow(img, 26); got == 0 {
		t.Fatal("expected the bottom line to remain visible inside the clipped text bounds")
	}
}

func countNonWhitePixelsBelow(img *image.RGBA, minY int) int {
	count := 0
	bounds := img.Bounds()
	for y := maxInt(bounds.Min.Y, minY); y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
				count++
			}
		}
	}
	return count
}

func TestShouldFitNormalAutofitUsesImplicitScaleWhenRequested(t *testing.T) {
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "title"}) {
		t.Fatal("regular title normal-autofit should derive a scale when requested")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "title", FontScalePct: 90000}) {
		t.Fatal("expected title normal-autofit with explicit fontScale to fit")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "ctrTitle"}) {
		t.Fatal("centered title normal-autofit should derive a scale when requested")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "ctrTitle", FontScalePct: 90000}) {
		t.Fatal("expected centered title normal-autofit with explicit fontScale to fit")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, LineSpacingReductionPct: 10000}) {
		t.Fatal("content normal-autofit should derive a scale when requested")
	}
	if shouldFitNormalAutofit(slideElement{}) {
		t.Fatal("normal-autofit should not run when the text body did not request it")
	}
}

func TestScaledTextElementAppliesNormalAutofitFontScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		FontScalePct:            85000,
		FontSize:                2000,
		LineSpacingReductionPct: 20000,
		TextParagraphs: []textParagraph{{
			FontSize:       1000,
			SpaceBefore:    10,
			SpaceAfter:     20,
			LineSpacingPct: 90000,
			Runs: []textRun{
				{Text: "A", FontSize: 1200},
			},
		}},
	})
	if got.FontSize != 1700 || got.TextParagraphs[0].FontSize != 850 || got.TextParagraphs[0].Runs[0].FontSize != 1020 {
		t.Fatalf("unexpected scaled text sizes: %+v", got)
	}
	if got.TextParagraphs[0].LineSpacingPct != 70000 {
		t.Fatalf("unexpected reduced line spacing: %+v", got)
	}
	if got.TextParagraphs[0].SpaceBefore != 9 || got.TextParagraphs[0].SpaceAfter != 17 {
		t.Fatalf("unexpected scaled paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementDefersNormalAutofitLineSpacingReductionWithoutFontScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		LineSpacingReductionPct: 10000,
		FontSize:                2000,
		TextParagraphs: []textParagraph{{
			FontSize:       2000,
			SpaceBefore:    8,
			SpaceAfter:     12,
			LineSpacingPct: 90000,
			Runs: []textRun{{
				Text:     "A",
				FontSize: 2000,
			}},
		}},
	})
	if got.FontSize != 2000 || got.TextParagraphs[0].FontSize != 2000 || got.TextParagraphs[0].Runs[0].FontSize != 2000 {
		t.Fatalf("line spacing reduction must not scale font sizes: %+v", got)
	}
	if got.TextParagraphs[0].LineSpacingPct != 90000 {
		t.Fatalf("line spacing reduction without font scaling should be deferred: %+v", got)
	}
	if got.TextParagraphs[0].SpaceBefore != 8 || got.TextParagraphs[0].SpaceAfter != 12 {
		t.Fatalf("line spacing reduction must not scale paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementScalesParagraphSpacingForDPI(t *testing.T) {
	got := scaledTextElement(slideElement{
		TextParagraphs: []textParagraph{{
			SpaceBefore: 9,
			SpaceAfter:  12,
		}},
	}, 96)
	if got.TextParagraphs[0].SpaceBefore != 12 || got.TextParagraphs[0].SpaceAfter != 16 {
		t.Fatalf("unexpected dpi-scaled paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementDoesNotApplyLineSpacingReductionAtFullScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		FontScalePct:            100000,
		LineSpacingReductionPct: 10000,
		TextParagraphs: []textParagraph{{
			Text:           "Body",
			FontSize:       2400,
			LineSpacingPct: 100000,
		}},
	})
	if got.TextParagraphs[0].LineSpacingPct != 100000 {
		t.Fatalf("line spacing reduction should wait until normal autofit scales text below 100%%: %+v", got)
	}
}

func TestScaledTextElementDoesNotInventPercentageLineSpacingForReduction(t *testing.T) {
	got := scaledTextElement(slideElement{
		LineSpacingReductionPct: 10000,
		TextParagraphs: []textParagraph{{
			Text:     "Body",
			FontSize: 2400,
		}},
	})
	if got.TextParagraphs[0].LineSpacingPct != 0 {
		t.Fatalf("line spacing reduction applies only to percentage line spacing: %+v", got)
	}
}

func TestFallbackFontPointSizeKeepsThirtyTwoPointText(t *testing.T) {
	got := fallbackFontPointSize(3200, false, false)
	want := 32.0
	if got != want {
		t.Fatalf("expected 32pt text to keep its DrawingML point size: got %v want %v", got, want)
	}
}

func TestParseTextPropertiesKeepsRunSizeOverEndParagraphDefault(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
  <p:txBody>
    <a:p>
      <a:r><a:rPr sz="1400"><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:rPr><a:t>1</a:t></a:r>
      <a:endParaRPr sz="2000"><a:solidFill><a:srgbClr val="445566"/></a:solidFill></a:endParaRPr>
    </a:p>
  </p:txBody>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.FontSize != 0 {
		t.Fatalf("endParaRPr font size should not promote to shape fallback, got %+v", got)
	}
	if got.HasTextColor {
		t.Fatalf("expected direct run text color to stay run-scoped, got %+v", got)
	}
	if len(got.TextParagraphs) != 1 || len(got.TextParagraphs[0].Runs) != 1 {
		t.Fatalf("expected one text run, got %+v", got.TextParagraphs)
	}
	run := got.TextParagraphs[0].Runs[0]
	if run.FontSize != 1400 {
		t.Fatalf("expected run font size to stay run-scoped, got %+v", run)
	}
	if !run.HasTextColor || run.TextColor.R != 0x11 || run.TextColor.G != 0x22 || run.TextColor.B != 0x33 {
		t.Fatalf("expected run text color to win over endParaRPr, got %+v", run)
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphDefaultForParagraphOnly(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr/><a:t>Default sized</a:t></a:r>
    <a:endParaRPr sz="2000" b="1"/>
  </a:p>
  <a:p>
    <a:r><a:rPr/><a:t>Plain</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph parse: %+v", got)
	}
	if got[0].FontSize != 2000 || !got[0].Bold {
		t.Fatalf("expected endParaRPr to become first paragraph default, got %+v", got[0])
	}
	if got[1].FontSize != 0 || got[1].Bold {
		t.Fatalf("endParaRPr leaked to sibling paragraph: %+v", got[1])
	}
}

func TestAnchoredTextStartYCentersLines(t *testing.T) {
	got := anchoredTextStartY(image.Rect(0, 10, 100, 110), 2, 20, 12, "ctr")
	want := 52
	if got != want {
		t.Fatalf("unexpected centered text y: got=%d want=%d", got, want)
	}
}

func TestCollectSlideElementsFlattensGroupedShapesWithTransform(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:grpSp>
        <p:nvGrpSpPr><p:cNvPr id="10" name="Group 9"/></p:nvGrpSpPr>
        <p:grpSpPr>
          <a:xfrm>
            <a:off x="1000" y="2000"/>
            <a:ext cx="2000" cy="2000"/>
            <a:chOff x="5000" y="6000"/>
            <a:chExt cx="2000" cy="2000"/>
          </a:xfrm>
        </p:grpSpPr>
        <p:sp>
          <p:nvSpPr><p:cNvPr id="11" name="Grouped Text"/></p:nvSpPr>
          <p:spPr>
            <a:xfrm><a:off x="5500" y="6500"/><a:ext cx="500" cy="400"/></a:xfrm>
            <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
            <a:solidFill><a:srgbClr val="FF0000"/></a:solidFill>
          </p:spPr>
          <p:txBody>
            <a:p><a:r><a:rPr sz="1200"/><a:t>Grouped</a:t></a:r></a:p>
          </p:txBody>
        </p:sp>
      </p:grpSp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one flattened grouped element, got %+v", elements)
	}
	got := elements[0]
	if got.Name != "Grouped Text" || got.Text != "Grouped" {
		t.Fatalf("unexpected grouped element identity: %+v", got)
	}
	if got.OffX != 1500 || got.OffY != 2500 || got.ExtCX != 500 || got.ExtCY != 400 {
		t.Fatalf("unexpected grouped transform: %+v", got)
	}
	if !got.HasFill || got.FillColor.R != 255 {
		t.Fatalf("expected grouped fill parse, got %+v", got)
	}
}

func TestSchemeColorMapsCommonThemeSlots(t *testing.T) {
	got, ok := schemeColor("accent1")
	if !ok {
		t.Fatal("accent1 was not mapped")
	}
	if got.R == 0 && got.G == 0 && got.B == 0 {
		t.Fatalf("accent1 should not map to black: %#v", got)
	}
	got, ok = schemeColor("tx1")
	if !ok || got.A != 255 || got.R != 0 || got.G != 0 || got.B != 0 {
		t.Fatalf("unexpected tx1 mapping: %#v ok=%v", got, ok)
	}
}

func TestParseThemeColorsMapsPackageScheme(t *testing.T) {
	theme := parseThemeColors([]byte(`<a:theme xmlns:a="a">
  <a:themeElements>
    <a:clrScheme name="Office">
      <a:dk1><a:sysClr val="windowText" lastClr="111111"/></a:dk1>
      <a:lt1><a:sysClr val="window" lastClr="EEEEEE"/></a:lt1>
      <a:dk2><a:srgbClr val="44546A"/></a:dk2>
      <a:lt2><a:srgbClr val="E7E6E6"/></a:lt2>
      <a:accent5><a:srgbClr val="4472C4"/></a:accent5>
      <a:accent6><a:srgbClr val="70AD47"/></a:accent6>
    </a:clrScheme>
  </a:themeElements>
</a:theme>`))
	got, ok := schemeColorWithTheme("accent6", theme)
	if !ok || got.R != 0x70 || got.G != 0xad || got.B != 0x47 {
		t.Fatalf("unexpected accent6 from theme: %#v ok=%v", got, ok)
	}
	got, ok = schemeColorWithTheme("bg1", theme)
	if !ok || got.R != 0xee || got.G != 0xee || got.B != 0xee {
		t.Fatalf("unexpected bg1 alias from theme: %#v ok=%v", got, ok)
	}
}

func TestApplyThemeColorMapUsesMasterSlotAliases(t *testing.T) {
	theme := themeColors{
		"lt2":     {R: 0xee, G: 0xee, B: 0xee, A: 0xff},
		"dk2":     {R: 0x44, G: 0x55, B: 0x66, A: 0xff},
		"accent1": {R: 0x11, G: 0x22, B: 0x33, A: 0xff},
		"accent6": {R: 0x70, G: 0xad, B: 0x47, A: 0xff},
	}
	mapped := applyThemeColorMap(theme, map[string]string{
		"bg2":     "dk2",
		"tx2":     "lt2",
		"accent1": "accent6",
	})
	if mapped["bg2"] != theme["dk2"] || mapped["tx2"] != theme["lt2"] || mapped["accent1"] != theme["accent6"] {
		t.Fatalf("theme color map was not applied: %#v", mapped)
	}
}

func TestThemeColorsForPartAppliesSlideMasterColorMap(t *testing.T) {
	pkg := &pptx.Package{
		Parts: map[string][]byte{
			"ppt/slides/slide1.xml": []byte(`<p:sld xmlns:p="p"/>`),
			"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`),
			"ppt/slideLayouts/slideLayout1.xml": []byte(`<p:sldLayout xmlns:p="p"><p:clrMapOvr><a:masterClrMapping xmlns:a="a"/></p:clrMapOvr></p:sldLayout>`),
			"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`),
			"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"><p:clrMap bg1="lt1" tx1="dk1" bg2="dk2" tx2="lt2" accent1="accent6"/></p:sldMaster>`),
			"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`),
			"ppt/theme/theme1.xml": []byte(`<a:theme xmlns:a="a"><a:themeElements><a:clrScheme name="Office">
  <a:dk1><a:srgbClr val="000000"/></a:dk1><a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>
  <a:dk2><a:srgbClr val="445566"/></a:dk2><a:lt2><a:srgbClr val="EEEEEE"/></a:lt2>
  <a:accent6><a:srgbClr val="70AD47"/></a:accent6>
</a:clrScheme></a:themeElements></a:theme>`),
		},
	}

	colors := themeColorsForPart(pkg, "ppt/slides/slide1.xml", defaultThemeColors())
	if got := colors["bg2"]; got.R != 0x44 || got.G != 0x55 || got.B != 0x66 {
		t.Fatalf("expected bg2 to map through master clrMap to dk2, got %#v", got)
	}
	if got := colors["tx2"]; got.R != 0xee || got.G != 0xee || got.B != 0xee {
		t.Fatalf("expected tx2 to map through master clrMap to lt2, got %#v", got)
	}
	if got := colors["accent1"]; got.R != 0x70 || got.G != 0xad || got.B != 0x47 {
		t.Fatalf("expected accent1 remap to accent6, got %#v", got)
	}
}

func TestParseThemeFontsMapsLatinScheme(t *testing.T) {
	fonts := parseThemeFonts([]byte(`<a:theme xmlns:a="a">
  <a:themeElements>
    <a:fontScheme name="Facet">
      <a:majorFont><a:latin typeface="Trebuchet MS"/><a:ea typeface="Yu Gothic"/><a:cs typeface="Times New Roman"/></a:majorFont>
      <a:minorFont><a:latin typeface="Arial"/><a:ea typeface="MS Gothic"/><a:cs typeface="Tahoma"/></a:minorFont>
    </a:fontScheme>
  </a:themeElements>
</a:theme>`))
	if fonts.MajorLatin != "Trebuchet MS" || fonts.MinorLatin != "Arial" {
		t.Fatalf("unexpected theme fonts: %+v", fonts)
	}
	if fonts.MajorEA != "Yu Gothic" || fonts.MajorCS != "Times New Roman" || fonts.MinorEA != "MS Gothic" || fonts.MinorCS != "Tahoma" {
		t.Fatalf("unexpected non-Latin theme fonts: %+v", fonts)
	}
}

func TestApplyThemeFontFamiliesUsesMajorForTitles(t *testing.T) {
	elements := []slideElement{
		{Text: "Title", IsPlaceholder: true, PlaceholderType: "title"},
		{Text: "Body", IsPlaceholder: true, PlaceholderType: "body"},
		{Text: "Fixed", FontFamily: "Existing"},
		{Text: "ElementToken", FontFamily: "+mn-lt"},
		{Text: "Runs", TextParagraphs: []textParagraph{{Runs: []textRun{
			{Text: "Major", FontFamily: "+mj-lt"},
			{Text: "Minor", FontFamily: "+mn-lt"},
			{Text: "MajorEA", FontFamily: "+mj-ea"},
			{Text: "MinorCS", FontFamily: "+mn-cs"},
		}}}},
		{Text: "Bullet", TextParagraphs: []textParagraph{{Bullet: "•", BulletFontFamily: "+mj-cs"}}},
	}
	got := applyThemeFontFamilies(elements, themeFonts{
		MajorLatin: "Trebuchet MS",
		MajorEA:    "Yu Gothic",
		MajorCS:    "Times New Roman",
		MinorLatin: "Arial",
		MinorEA:    "MS Gothic",
		MinorCS:    "Tahoma",
	})
	if got[0].FontFamily != "Trebuchet MS" || got[1].FontFamily != "Arial" || got[2].FontFamily != "Existing" || got[3].FontFamily != "Arial" {
		t.Fatalf("unexpected font family application: %+v", got)
	}
	if got[4].TextParagraphs[0].Runs[0].FontFamily != "Trebuchet MS" || got[4].TextParagraphs[0].Runs[1].FontFamily != "Arial" {
		t.Fatalf("unexpected run font family application: %+v", got[4].TextParagraphs[0].Runs)
	}
	if got[4].TextParagraphs[0].Runs[2].FontFamily != "Yu Gothic" || got[4].TextParagraphs[0].Runs[3].FontFamily != "Tahoma" {
		t.Fatalf("unexpected non-Latin run font family application: %+v", got[4].TextParagraphs[0].Runs)
	}
	if got[5].TextParagraphs[0].BulletFontFamily != "Times New Roman" {
		t.Fatalf("unexpected bullet font family application: %+v", got[5].TextParagraphs[0])
	}
}

func TestCollectSlideElementsUsesPackageThemeColors(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="Themed Rectangle"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:schemeClr val="accent6"/></a:solidFill>
        </p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElementsWithTheme(data, themeColors{
		"accent6": {R: 0x70, G: 0xad, B: 0x47, A: 0xff},
	})
	if len(elements) != 1 || !elements[0].HasFill {
		t.Fatalf("expected one themed fill element, got %+v", elements)
	}
	if got := elements[0].FillColor; got.R != 0x70 || got.G != 0xad || got.B != 0x47 {
		t.Fatalf("unexpected themed fill color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLModifiers(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent5"><a:lumMod val="20000"/><a:lumOff val="80000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 219 || got.G != 238 || got.B != 244 || got.A != 255 {
		t.Fatalf("unexpected modified color: %#v", got)
	}
}

func TestColorFromColorNodeRoundsLuminanceModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent6"><a:lumMod val="50000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent6": {R: 0x70, G: 0xad, B: 0x47, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 56, G: 87, B: 36, A: 255}) {
		t.Fatalf("unexpected rounded luminance color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesLuminanceInHSLSpace(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:lumMod val="50000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 32, G: 56, B: 100, A: 255}) {
		t.Fatalf("unexpected HSL luminance color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLTint(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:tint val="20000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 233, G: 235, B: 245, A: 255}) {
		t.Fatalf("unexpected tinted color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLShade(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:shade val="40000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 42, G: 73, B: 129, A: 255}) {
		t.Fatalf("unexpected shaded color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesAlphaModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="804020"><a:alpha val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0x80 || got.G != 0x40 || got.B != 0x20 || got.A != 127 {
		t.Fatalf("unexpected alpha-modified color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesAlphaOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="804020"><a:alphaOff val="-10000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0x80 || got.G != 0x40 || got.B != 0x20 || got.A != 230 {
		t.Fatalf("unexpected alpha-offset color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesSaturationModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="806060"><a:satMod val="200000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R <= 0x80 || got.G >= 0x60 || got.B >= 0x60 || got.A != 255 {
		t.Fatalf("expected saturation modifier to widen channel contrast, got %#v", got)
	}
}

func TestColorFromColorNodeAppliesSaturationOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="806060"><a:satOff val="-50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R >= 0x80 || got.G <= 0x60 || got.B <= 0x60 || got.A != 255 {
		t.Fatalf("expected saturation offset to narrow channel contrast, got %#v", got)
	}
}

func TestColorFromColorNodeAppliesHueOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="FF0000"><a:hueOff val="7200000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0 || got.G != 255 || got.B != 0 || got.A != 255 {
		t.Fatalf("unexpected hue-offset color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesModifiersInDocumentOrder(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="808080"><a:tint val="50000"/><a:shade val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	reversedRoot, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="808080"><a:shade val="50000"/><a:tint val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	reversed, ok := colorFromColorNode(reversedRoot)
	if !ok {
		t.Fatal("reversed color was not parsed")
	}
	if got == reversed {
		t.Fatalf("modifier order should affect color: ordered=%#v reversed=%#v", got, reversed)
	}
	if got != (color.RGBA{R: 150, G: 150, B: 150, A: 255}) {
		t.Fatalf("unexpected ordered color: %#v", got)
	}
}

func TestColorFromColorNodeParsesScRGB(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:scrgbClr r="50000" g="0" b="100000"><a:alpha val="50000"/></a:scrgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 188 || got.G != 0 || got.B != 255 || got.A != 127 {
		t.Fatalf("unexpected scrgb color: %#v", got)
	}
}

func TestColorFromColorNodeClampsScRGBBeforeSRGBTransfer(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:scrgbClr r="-50000" g="150000" b="25000"/></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 0, G: 255, B: 137, A: 255}) {
		t.Fatalf("unexpected clamped scrgb color: %#v", got)
	}
}

func TestColorFromColorNodeParsesSystemColorLastColor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:sysClr val="windowText" lastClr="123456"><a:alpha val="50000"/></a:sysClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 127}) {
		t.Fatalf("unexpected system color: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetRed(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:prstClr val="red"><a:alpha val="25000"/></a:prstClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 255, A: 63}) {
		t.Fatalf("unexpected preset red: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetBlack(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:effectLst xmlns:a="a"><a:outerShdw><a:prstClr val="black"><a:alpha val="40000"/></a:prstClr></a:outerShdw></a:effectLst>`))
	if err != nil {
		t.Fatal(err)
	}
	shadow := firstDescendant(root, "outerShdw")
	got, ok := colorFromColorNode(shadow)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{A: 102}) {
		t.Fatalf("unexpected preset black: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetWhite(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:prstClr val="white"/></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("unexpected preset white: %#v", got)
	}
}

func TestParseSlideBackgroundSolidSRGB(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:srgbClr val="1A2B3C"/></a:solidFill>
      </p:bgPr>
    </p:bg>
  </p:cSld>
</p:sld>`)

	got := parseSlideBackground(data)
	if got.R != 0x1a || got.G != 0x2b || got.B != 0x3c || got.A != 0xff {
		t.Fatalf("unexpected background color: %#v", got)
	}
}

func TestParseSlideBackgroundGradient(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:gradFill>
          <a:gsLst>
            <a:gs pos="100000"><a:srgbClr val="DFE0E2"/></a:gs>
            <a:gs pos="0"><a:schemeClr val="bg1"/></a:gs>
          </a:gsLst>
          <a:path path="circle"><a:fillToRect l="50000" t="-80000" r="50000" b="180000"/></a:path>
        </a:gradFill>
      </p:bgPr>
    </p:bg>
  </p:cSld>
</p:sld>`)

	got, ok := parseSlideBackgroundPaintWithTheme(data, themeColors{
		"bg1": {R: 255, G: 255, B: 255, A: 255},
	})
	if !ok || !got.HasGradient || got.Gradient.Path != "circle" || len(got.Gradient.Stops) != 2 {
		t.Fatalf("unexpected gradient background parse: got=%+v ok=%v", got, ok)
	}
	if got.Gradient.Stops[0].Position != 0 || got.Gradient.Stops[1].Position != 100000 {
		t.Fatalf("gradient stops were not sorted: %+v", got.Gradient.Stops)
	}
	if !got.Gradient.HasFillRect || got.Gradient.FillRect.Top != -80000 || got.Gradient.FillRect.Bottom != 180000 {
		t.Fatalf("gradient fillToRect was not parsed: %+v", got.Gradient)
	}
	if !got.Gradient.FullySupported {
		t.Fatalf("circle gradient with fillToRect should be fully supported: %+v", got.Gradient)
	}
}

func TestParseSlideBackgroundGradientMarksUnsupportedPathPartial(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	  <p:cSld>
	    <p:bg>
	      <p:bgPr>
	        <a:gradFill>
	          <a:gsLst>
	            <a:gs pos="0"><a:srgbClr val="FFFFFF"/></a:gs>
	            <a:gs pos="100000"><a:srgbClr val="000000"/></a:gs>
	          </a:gsLst>
	          <a:path path="rect"/>
	        </a:gradFill>
	      </p:bgPr>
	    </p:bg>
	  </p:cSld>
	</p:sld>`)

	got, ok := parseSlideBackgroundPaintWithTheme(data, defaultThemeColors())
	if !ok || !got.HasGradient {
		t.Fatalf("expected gradient background parse: got=%+v ok=%v", got, ok)
	}
	if got.Gradient.FullySupported {
		t.Fatalf("unsupported gradient path was marked fully supported: %+v", got.Gradient)
	}
}

func TestParseSlideBackgroundRefUsesThemeFillStyle(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
	<p:sldMaster xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgRef idx="1002"><a:schemeClr val="bg1"/></p:bgRef>
    </p:bg>
  </p:cSld>
</p:sldMaster>`)
	theme := []byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <a:themeElements>
    <a:fmtScheme name="Office">
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:gradFill>
          <a:gsLst>
            <a:gs pos="0"><a:schemeClr val="phClr"><a:tint val="90000"/></a:schemeClr></a:gs>
            <a:gs pos="100000"><a:srgbClr val="000000"/></a:gs>
          </a:gsLst>
          <a:lin ang="5400000" scaled="0"/>
        </a:gradFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`)
	baseTheme := themeColors{"bg1": {R: 100, G: 120, B: 140, A: 255}}
	got, ok := parseSlideBackgroundPaintWithThemeAndResolver(data, baseTheme, func(idx int64, placeholderColor color.RGBA) (backgroundPaint, bool) {
		return parseThemeBackgroundFill(theme, idx, placeholderColor, baseTheme)
	})
	if !ok || !got.HasGradient || len(got.Gradient.Stops) != 2 {
		t.Fatalf("expected bgRef theme gradient, got=%+v ok=%v", got, ok)
	}
	if got.Gradient.Stops[0].Color.R <= 100 || got.Gradient.Stops[0].Color.G <= 120 || got.Gradient.Stops[0].Color.B <= 140 {
		t.Fatalf("expected phClr tint to use bgRef placeholder color, got %+v", got.Gradient.Stops[0].Color)
	}
	if !got.Gradient.HasAngle || got.Gradient.Angle != 5400000 {
		t.Fatalf("expected linear gradient angle to be parsed, got %+v", got.Gradient)
	}
	if !got.Gradient.HasScaled || got.Gradient.Scaled {
		t.Fatalf("expected linear gradient scaled flag to be parsed as false, got %+v", got.Gradient)
	}
}

func TestParseStylePropertiesAppliesThemeEffectReference(t *testing.T) {
	theme := themeColors{
		"accent1": {R: 10, G: 20, B: 30, A: 255},
	}
	effectStyles := parseThemeEffectStyles([]byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	  <a:themeElements>
	    <a:fmtScheme name="Office">
	      <a:effectStyleLst>
	        <a:effectStyle>
	          <a:effectLst>
	            <a:outerShdw blurRad="40000" dist="20000" dir="5400000">
	              <a:schemeClr val="phClr"><a:alpha val="50000"/></a:schemeClr>
	            </a:outerShdw>
	          </a:effectLst>
	        </a:effectStyle>
	      </a:effectStyleLst>
	    </a:fmtScheme>
	  </a:themeElements>
	</a:theme>`))
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Styled Rectangle"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/></p:spPr>
	  <p:style><a:effectRef idx="1"><a:schemeClr val="accent1"/></a:effectRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNodeWithThemeAndEffects(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, effectStyles)
	if !got.HasShadow {
		t.Fatalf("expected effectRef to apply theme outer shadow: %+v", got)
	}
	if got.ShadowBlur != 40000 || got.ShadowDistance != 20000 || got.ShadowDirection != 5400000 {
		t.Fatalf("unexpected inherited shadow metrics: %+v", got)
	}
	if got.ShadowColor != (color.RGBA{R: 10, G: 20, B: 30, A: 127}) {
		t.Fatalf("expected effectRef color to bind phClr, got %#v", got.ShadowColor)
	}
}

func TestParseStylePropertiesLeavesExplicitShadowOverThemeEffectReference(t *testing.T) {
	theme := themeColors{
		"accent1": {R: 10, G: 20, B: 30, A: 255},
	}
	effectStyles := parseThemeEffectStyles([]byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	  <a:themeElements><a:fmtScheme name="Office"><a:effectStyleLst>
	    <a:effectStyle><a:effectLst><a:outerShdw blurRad="40000" dist="20000" dir="5400000"><a:schemeClr val="phClr"/></a:outerShdw></a:effectLst></a:effectStyle>
	  </a:effectStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Styled Rectangle"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/><a:effectLst><a:outerShdw blurRad="10000" dist="5000" dir="0"><a:srgbClr val="000000"><a:alpha val="25000"/></a:srgbClr></a:outerShdw></a:effectLst></p:spPr>
	  <p:style><a:effectRef idx="1"><a:schemeClr val="accent1"/></a:effectRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNodeWithThemeAndEffects(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, effectStyles)
	if got.ShadowBlur != 10000 || got.ShadowDistance != 5000 || got.ShadowDirection != 0 {
		t.Fatalf("expected explicit shape shadow to win over style effectRef, got %+v", got)
	}
	if got.ShadowColor != (color.RGBA{A: 63}) {
		t.Fatalf("unexpected explicit shadow color: %#v", got.ShadowColor)
	}
}

func TestParseStylePropertiesTreatsEmptyLocalEffectListAsOverride(t *testing.T) {
	theme := themeColors{
		"accent1": {R: 10, G: 20, B: 30, A: 255},
	}
	effectStyles := parseThemeEffectStyles([]byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	  <a:themeElements><a:fmtScheme name="Office"><a:effectStyleLst>
	    <a:effectStyle><a:effectLst><a:outerShdw blurRad="40000" dist="20000" dir="5400000"><a:schemeClr val="phClr"/></a:outerShdw></a:effectLst></a:effectStyle>
	  </a:effectStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Styled Rectangle"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/><a:effectLst/></p:spPr>
	  <p:style><a:effectRef idx="1"><a:schemeClr val="accent1"/></a:effectRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNodeWithThemeAndEffects(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, effectStyles)
	if !got.HasEffectProperties {
		t.Fatalf("expected local empty effectLst to be recorded: %+v", got)
	}
	if got.HasShadow {
		t.Fatalf("empty local effectLst should suppress theme effectRef shadow, got %+v", got)
	}
}

func TestThemePartForRenderPartFollowsSlideLayoutMasterThemeRelationships(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
		</Relationships>`),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
		</Relationships>`),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme2.xml"/>
		</Relationships>`),
	}}

	if got := themePartForRenderPart(pkg, "ppt/slides/slide1.xml"); got != "ppt/theme/theme2.xml" {
		t.Fatalf("expected slide theme relationship chain to resolve theme2, got %q", got)
	}
}

func TestThemeColorsAndFontsForPartUseResolvedThemeRelationship(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
		</Relationships>`),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
		</Relationships>`),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
		  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme2.xml"/>
		</Relationships>`),
		"ppt/theme/theme2.xml": []byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
		  <a:themeElements>
		    <a:clrScheme name="Test">
		      <a:dk1><a:srgbClr val="111111"/></a:dk1><a:lt1><a:srgbClr val="EEEEEE"/></a:lt1>
		      <a:dk2><a:srgbClr val="222222"/></a:dk2><a:lt2><a:srgbClr val="DDDDDD"/></a:lt2>
		      <a:accent1><a:srgbClr val="123456"/></a:accent1>
		    </a:clrScheme>
		    <a:fontScheme name="Test">
		      <a:majorFont><a:latin typeface="Trebuchet MS"/></a:majorFont>
		      <a:minorFont><a:latin typeface="Arial"/></a:minorFont>
		    </a:fontScheme>
		  </a:themeElements>
		</a:theme>`),
	}}

	colors := themeColorsForPart(pkg, "ppt/slides/slide1.xml", defaultThemeColors())
	if colors["accent1"] != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}) {
		t.Fatalf("expected colors from resolved theme part, got %+v", colors["accent1"])
	}
	fonts := themeFontsForPart(pkg, "ppt/slides/slide1.xml", themeFonts{})
	if fonts.MajorLatin != "Trebuchet MS" || fonts.MinorLatin != "Arial" {
		t.Fatalf("expected fonts from resolved theme part, got %+v", fonts)
	}
}

func TestDrawGradientBackgroundPaintsRadialStops(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 11, 11))
	drawGradientBackground(img, gradientPaint{
		Path: "circle",
		Stops: []gradientStop{
			{Position: 0, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
			{Position: 100000, Color: color.RGBA{A: 255}},
		},
	})
	centerR, _, _, _ := img.At(5, 5).RGBA()
	cornerR, _, _, _ := img.At(0, 0).RGBA()
	if centerR <= cornerR {
		t.Fatalf("expected radial gradient center to be lighter than corner: center=%04x corner=%04x", centerR, cornerR)
	}
}

func TestDrawGradientBackgroundSamplesRadialGradientAtPixelCenters(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 12, 8))
	edge := color.RGBA{R: 223, G: 224, B: 226, A: 255}
	drawGradientBackground(img, gradientPaint{
		Path: "circle",
		Stops: []gradientStop{
			{Position: 0, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
			{Position: 100000, Color: edge},
		},
	})
	got := img.RGBAAt(0, 0)
	if got.R <= edge.R || got.G <= edge.G || got.B <= edge.B {
		t.Fatalf("expected corner pixel center to sample just inside the final stop: got=%#v edge=%#v", got, edge)
	}
}

func TestDrawGradientBackgroundUsesFillToRectFocus(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	drawGradientBackground(img, gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 50000, Top: -80000, Right: 50000, Bottom: 180000},
		Stops: []gradientStop{
			{Position: 0, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
			{Position: 100000, Color: color.RGBA{A: 255}},
		},
	})
	topR, _, _, _ := img.At(10, 0).RGBA()
	bottomR, _, _, _ := img.At(10, 9).RGBA()
	if topR <= bottomR {
		t.Fatalf("expected fillToRect focus above the slide to keep top lighter than bottom: top=%04x bottom=%04x", topR, bottomR)
	}
}

func TestDrawGradientBackgroundUsesLinearAngle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	drawGradientBackground(img, gradientPaint{
		HasAngle: true,
		Angle:    16200000,
		Stops: []gradientStop{
			{Position: 0, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
			{Position: 100000, Color: color.RGBA{A: 255}},
		},
	})
	topR, _, _, _ := img.At(1, 0).RGBA()
	bottomR, _, _, _ := img.At(1, 2).RGBA()
	if bottomR <= topR {
		t.Fatalf("expected 270-degree linear gradient to reverse vertical direction: top=%04x bottom=%04x", topR, bottomR)
	}
}

func TestRenderShapePaintsRectGradientFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &slideElement{
		Kind:            "sp",
		Name:            "Gradient Rect",
		HasTransform:    true,
		ExtCX:           emuPerInch,
		ExtCY:           emuPerInch,
		PrstGeom:        "rect",
		HasFill:         true,
		FillColor:       color.RGBA{R: 255, A: 255},
		HasFillGradient: true,
		FillGradient: gradientPaint{
			FullySupported: true,
			Stops: []gradientStop{
				{Position: 0, Color: color.RGBA{R: 255, A: 255}},
				{Position: 100000, Color: color.RGBA{B: 255, A: 255}},
			},
		},
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported rect gradient fill, got %+v", unsupported)
	}
	top := img.RGBAAt(5, 0)
	bottom := img.RGBAAt(5, 9)
	if top.R <= bottom.R || bottom.B <= top.B {
		t.Fatalf("expected vertical gradient to transition from red to blue, top=%#v bottom=%#v", top, bottom)
	}
}

func TestRenderShapeSupportsCircleGradientFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &slideElement{
		Kind:            "sp",
		Name:            "Radial Gradient Rect",
		HasTransform:    true,
		ExtCX:           emuPerInch,
		ExtCY:           emuPerInch,
		PrstGeom:        "rect",
		HasFillGradient: true,
		FillGradient: gradientPaint{
			Path:           "circle",
			HasFillRect:    true,
			FillRect:       relativeRect{Left: 50000, Top: 50000, Right: 50000, Bottom: 50000},
			FullySupported: true,
			Stops: []gradientStop{
				{Position: 0, Color: color.RGBA{R: 255, A: 255}},
				{Position: 100000, Color: color.RGBA{B: 255, A: 255}},
			},
		},
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported circle gradient fill, got %+v", unsupported)
	}
}

func TestLinearGradientPositionHonorsScaledAngle(t *testing.T) {
	bounds := image.Rect(0, 0, 20, 10)
	unscaled := linearGradientPosition(bounds, 10, 0, gradientPaint{HasAngle: true, Angle: 2700000})
	scaled := linearGradientPosition(bounds, 10, 0, gradientPaint{HasAngle: true, Angle: 2700000, HasScaled: true, Scaled: true})
	if scaled <= unscaled {
		t.Fatalf("expected scaled 45-degree gradient to account for fill-region aspect ratio: scaled=%d unscaled=%d", scaled, unscaled)
	}
}

func TestLinearGradientPositionSamplesPixelCenters(t *testing.T) {
	bounds := image.Rect(0, 0, 10, 10)
	if got := linearGradientPosition(bounds, 0, 0, gradientPaint{}); got != 5000 {
		t.Fatalf("expected top pixel center to sample inside gradient bounds, got %d", got)
	}
	if got := linearGradientPosition(bounds, 0, 9, gradientPaint{}); got != 95000 {
		t.Fatalf("expected bottom pixel center to sample inside gradient bounds, got %d", got)
	}
}

func TestGradientFocusRectCollapsesInvertedFillToRect(t *testing.T) {
	got := gradientFocusRect(image.Rect(0, 0, 100, 50), gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 50000, Top: 50000, Right: 100000, Bottom: 100000},
	})
	if math.Abs(got.MinX-22.049150) > 0.000001 || math.Abs(got.MaxX-22.049150) > 0.000001 ||
		math.Abs(got.MinY+2.950850) > 0.000001 || math.Abs(got.MaxY+2.950850) > 0.000001 {
		t.Fatalf("unexpected collapsed focus rect: %+v", got)
	}
}

func TestRadialGradientFocusPointUsesOpenSpecificationsFormula(t *testing.T) {
	got := radialGradientFocusPoint(image.Rect(0, 0, 100, 100), gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 10000, Top: 30000, Right: 80000, Bottom: 20000},
	})
	if math.Abs(got.X+4.997194) > 0.000001 || math.Abs(got.Y-64.142136) > 0.000001 {
		t.Fatalf("unexpected radial gradient focus point: %+v", got)
	}
}

func TestRadialGradientFocusRectUsesCircumscribedCircleBounds(t *testing.T) {
	got := gradientFocusRect(image.Rect(0, 0, 100, 50), gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 50000, Top: 50000, Right: 50000, Bottom: 50000},
	})
	if math.Abs(got.MinX-50) > 0.000001 || math.Abs(got.MaxX-50) > 0.000001 ||
		math.Abs(got.MinY-25) > 0.000001 || math.Abs(got.MaxY-25) > 0.000001 {
		t.Fatalf("expected centered focus point in circumscribed circle bounds: %+v", got)
	}
	outer := radialGradientOuterRect(image.Rect(0, 0, 100, 50))
	if math.Abs((outer.MaxX-outer.MinX)-math.Hypot(100, 50)) > 0.000001 ||
		math.Abs((outer.MaxY-outer.MinY)-math.Hypot(100, 50)) > 0.000001 {
		t.Fatalf("expected radial outer rect to circumscribe anchor rectangle: %+v", outer)
	}
}

func TestRadialGradientPositionKeepsFocusEllipseAtFirstStop(t *testing.T) {
	gradient := gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 25000, Top: 25000, Right: 25000, Bottom: 25000},
	}
	if got := radialGradientPosition(image.Rect(0, 0, 100, 100), 50, 30, gradient); got != 0 {
		t.Fatalf("expected sample inside focus ellipse to stay at first stop, got %d", got)
	}
	if got := radialGradientPosition(image.Rect(0, 0, 100, 100), 50, 90, gradient); got <= 0 {
		t.Fatalf("expected sample outside focus ellipse to advance through radial range, got %d", got)
	}
}

func TestColorAtGradientPositionUsesLinearInterpolationForRadialPath(t *testing.T) {
	stops := []gradientStop{
		{Position: 38000, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{Position: 100000, Color: color.RGBA{R: 223, G: 224, B: 226, A: 255}},
	}
	linear := colorAtGradientPositionForPath(stops, 86437, "")
	radial := colorAtGradientPositionForPath(stops, 86437, "circle")
	if radial != linear {
		t.Fatalf("expected radial gradient to use stop interpolation directly: linear=%#v radial=%#v", linear, radial)
	}
}

func TestColorAtGradientPositionUsesOfficeGammaForFullRangeTwoStopGradients(t *testing.T) {
	stops := []gradientStop{
		{Position: 0, Color: color.RGBA{A: 255}},
		{Position: 100000, Color: color.RGBA{R: 255, G: 255, B: 255, A: 127}},
	}
	got := colorAtGradientPosition(stops, 50000)
	linear := colorAtGradientPosition([]gradientStop{
		{Position: 0, Color: color.RGBA{A: 255}},
		{Position: 99999, Color: color.RGBA{R: 255, G: 255, B: 255, A: 127}},
	}, 50000)
	if got.R <= linear.R || got.G <= linear.G || got.B <= linear.B {
		t.Fatalf("expected Office gamma interpolation to brighten RGB channels: got=%#v linear=%#v", got, linear)
	}
	if got.A != 191 {
		t.Fatalf("expected alpha to remain linearly interpolated, got=%#v", got)
	}
}

func TestColorAtGradientPositionUsesOfficeGammaForMirroredThreeStopGradients(t *testing.T) {
	stops := []gradientStop{
		{Position: 0, Color: color.RGBA{A: 255}},
		{Position: 40000, Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}},
		{Position: 100000, Color: color.RGBA{A: 255}},
	}
	left := colorAtGradientPosition(stops, 20000)
	right := colorAtGradientPosition(stops, 70000)
	if left != right {
		t.Fatalf("expected mirrored three-stop gradient to interpolate symmetrically: left=%#v right=%#v", left, right)
	}
}

func TestDrawSoftRectDistributesShadowAlpha(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	drawSoftRect(img, image.Rect(5, 5, 15, 15), color.RGBA{A: 90}, 4)

	r, _, _, _ := img.At(10, 10).RGBA()
	if r < 0xa000 {
		t.Fatalf("expected bounded shadow alpha to avoid over-darkening, red=%04x", r)
	}
}

func TestDrawSoftRectFadesBeyondShapeBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	drawSoftRect(img, image.Rect(8, 8, 16, 16), color.RGBA{A: 128}, 4)

	center := img.RGBAAt(12, 12)
	edge := img.RGBAAt(6, 12)
	outside := img.RGBAAt(2, 12)
	if center.R >= edge.R {
		t.Fatalf("expected center shadow to be darker than faded edge, center=%+v edge=%+v", center, edge)
	}
	if edge.R >= outside.R {
		t.Fatalf("expected faded edge to be darker than untouched outside, edge=%+v outside=%+v", edge, outside)
	}
}

func TestGaussianKernelNormalizesWeights(t *testing.T) {
	kernel := gaussianKernel(4)
	sum := 0.0
	for _, weight := range kernel {
		sum += weight
	}
	if math.Abs(sum-1) > 0.000001 {
		t.Fatalf("expected normalized gaussian kernel, got sum=%f weights=%+v", sum, kernel)
	}
	if kernel[4] <= kernel[0] {
		t.Fatalf("expected gaussian center weight to dominate edge, got %+v", kernel)
	}
}

func TestDrawPolygonAntialiasesEdges(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	drawPolygon(img, img.Bounds(), []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, color.RGBA{A: 255})

	r, _, _, _ := img.At(5, 5).RGBA()
	if r == 0 || r == 0xffff {
		t.Fatalf("expected partially covered edge pixel, red=%04x", r)
	}
}

func TestUnsupportedItemsSkipsEmptyPlaceholders(t *testing.T) {
	got := unsupportedItems("ppt/slides/slide1.xml", []slideElement{
		{Kind: "sp", Name: "Content Placeholder 2", IsPlaceholder: true},
		{Kind: "sp", Name: "Picture Placeholder 8"},
		{Kind: "sp", Name: "Rectangle 1"},
	})
	if len(got) != 1 || !strings.Contains(got[0].Message, "Rectangle 1") {
		t.Fatalf("unexpected unsupported items: %+v", got)
	}
}

func TestFormatUnsupportedSummarySortsDeterministically(t *testing.T) {
	got := formatUnsupportedSummary(map[string]int{
		"later alphabetically":   2,
		"earlier alphabetically": 2,
		"most common":            3,
	}, 2)
	want := "3x most common; 2x earlier alphabetically"
	if got != want {
		t.Fatalf("unexpected unsupported summary: got=%q want=%q", got, want)
	}
	if got := formatUnsupportedSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty unsupported summary, got %q", got)
	}
}

func TestFormatSlideDiffSummarySortsDeterministically(t *testing.T) {
	got := formatSlideDiffSummary([]realWorldSlideDiff{
		{Label: "deck slide 002", DifferentPixels: 20, Unsupported: 3, Status: "partial"},
		{Label: "deck slide 001", DifferentPixels: 20, Unsupported: 2, Status: "partial"},
		{Label: "deck slide 003", DifferentPixels: 30, Unsupported: 1, Status: "partial"},
	}, 2)
	want := "deck slide 003=30px/1 unsupported/partial; deck slide 001=20px/2 unsupported/partial"
	if got != want {
		t.Fatalf("unexpected slide diff summary: got=%q want=%q", got, want)
	}
	if got := formatSlideDiffSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty slide diff summary, got %q", got)
	}
}

func TestRealWorldGoldenComparison(t *testing.T) {
	if os.Getenv("PUPPT_RUN_REALWORLD_RENDER_TESTS") != "1" {
		t.Skip("set PUPPT_RUN_REALWORLD_RENDER_TESTS=1 to run the 61-slide renderer golden comparison")
	}

	referenceRoot := realWorldReferenceRoot()
	manifestPath := filepath.Join(referenceRoot, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest referenceManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	total := 0
	failures := 0
	totalDifferentPixels := int64(0)
	worstDifferentPixels := int64(0)
	worstLabel := ""
	unsupportedCounts := map[string]int{}
	var slideDiffs []realWorldSlideDiff
	for _, deck := range manifest.Decks {
		if deck.Input == "" {
			t.Fatalf("manifest deck missing input: %+v", deck)
		}
		for index, slide := range deck.Slides {
			total++
			outputPath := filepath.Join(t.TempDir(), filepath.Base(deck.Input)+"-slide.png")
			result, err := Render(context.Background(), filepath.Join("..", "..", deck.Input), Options{
				SlideNumber: index + 1,
				OutputPath:  outputPath,
				DPI:         referenceDPIForSlide(slide.Width),
			})
			if err != nil {
				t.Fatalf("render %s slide %d: %v", deck.Input, index+1, err)
			}
			referencePath := filepath.Join(referenceRoot, slide.File)
			diff, err := comparePNG(outputPath, referencePath)
			if err != nil {
				t.Fatalf("compare %s slide %d: %v", deck.Input, index+1, err)
			}
			if diff.Width != slide.Width || diff.Height != slide.Height {
				t.Fatalf("manifest dimensions disagree for %s: manifest=%dx%d reference=%dx%d", slide.File, slide.Width, slide.Height, diff.Width, diff.Height)
			}
			if diff.GotWidth != slide.Width || diff.GotHeight != slide.Height {
				t.Fatalf("render dimensions disagree for %s slide %d: got=%dx%d reference=%dx%d", deck.Input, index+1, diff.GotWidth, diff.GotHeight, slide.Width, slide.Height)
			}
			if diff.DifferentPixels != 0 {
				writeRealWorldDiffArtifacts(t, outputPath, referencePath, deck.Input, index+1, result, diff)
				failures++
				totalDifferentPixels += int64(diff.DifferentPixels)
				for _, item := range result.Unsupported {
					unsupportedCounts[item.Message]++
				}
				if int64(diff.DifferentPixels) > worstDifferentPixels {
					worstDifferentPixels = int64(diff.DifferentPixels)
					worstLabel = fmt.Sprintf("%s slide %03d", deck.Input, index+1)
				}
				slideDiffs = append(slideDiffs, realWorldSlideDiff{
					Label:           fmt.Sprintf("%s slide %03d", deck.Input, index+1),
					DifferentPixels: diff.DifferentPixels,
					Unsupported:     len(result.Unsupported),
					Status:          result.Status,
				})
				if failures <= 10 {
					t.Logf("%s slide %03d: %d differing pixel(s), unsupported=%d, status=%s", deck.Input, index+1, diff.DifferentPixels, len(result.Unsupported), result.Status)
				}
			}
		}
	}
	if total != 61 {
		t.Fatalf("expected 61 real-world reference slides, got %d", total)
	}
	if failures != 0 {
		t.Fatalf("%d of %d real-world slides differ from %s references; total differing pixels=%d; worst=%s with %d differing pixels; worst slides: %s; top unsupported rendering gaps: %s", failures, total, referenceRoot, totalDifferentPixels, worstLabel, worstDifferentPixels, formatSlideDiffSummary(slideDiffs, 8), formatUnsupportedSummary(unsupportedCounts, 8))
	}
}

type realWorldSlideDiff struct {
	Label           string
	DifferentPixels int
	Unsupported     int
	Status          string
}

func formatSlideDiffSummary(items []realWorldSlideDiff, limit int) string {
	if len(items) == 0 || limit <= 0 {
		return "none"
	}
	items = append([]realWorldSlideDiff(nil), items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].DifferentPixels != items[j].DifferentPixels {
			return items[i].DifferentPixels > items[j].DifferentPixels
		}
		return items[i].Label < items[j].Label
	})
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%s=%dpx/%d unsupported/%s", item.Label, item.DifferentPixels, item.Unsupported, item.Status))
	}
	return builder.String()
}

func formatUnsupportedSummary(counts map[string]int, limit int) string {
	if len(counts) == 0 || limit <= 0 {
		return "none"
	}
	items := make([]unsupportedSummaryItem, 0, len(counts))
	for message, count := range counts {
		if count <= 0 {
			continue
		}
		items = append(items, unsupportedSummaryItem{Message: message, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Message < items[j].Message
	})
	if len(items) == 0 {
		return "none"
	}
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%dx %s", item.Count, item.Message))
	}
	return builder.String()
}

type unsupportedSummaryItem struct {
	Message string
	Count   int
}

func realWorldReferenceRoot() string {
	if root := os.Getenv("PUPPT_REALWORLD_REFERENCE_ROOT"); root != "" {
		return root
	}
	return filepath.Join("..", "..", "testdata", "realworld-ppts", "reference-renders", "manual-using-apple-note")
}

func TestWriteRealWorldDiffArtifactsWritesMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PUPPT_REALWORLD_ARTIFACT_DIR", dir)
	gotPath := filepath.Join(dir, "got-source.png")
	referencePath := filepath.Join(dir, "reference-source.png")
	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	got.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	got.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}

	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       commandName,
		Status:        "partial",
		Unsupported: []model.SkipItem{{
			Code:    partialUnsupportedCode,
			Message: "shape text was rendered with simplified layout",
			Part:    "ppt/slides/slide1.xml",
		}},
	}
	diff := imageDiff{Width: 2, Height: 1, GotWidth: 2, GotHeight: 1, DifferentPixels: 1}
	writeRealWorldDiffArtifacts(t, gotPath, referencePath, "testdata/realworld-ppts/example.pptx", 1, result, diff)

	slideDir := filepath.Join(dir, "example", "slide-001")
	for _, name := range []string{"got.png", "reference.png", "diff.png", "result.json", "diff.json"} {
		if _, err := os.Stat(filepath.Join(slideDir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(slideDir, "diff.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"different_pixels": 1`) {
		t.Fatalf("diff artifact did not include pixel count: %s", data)
	}
	data, err = os.ReadFile(filepath.Join(slideDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "simplified layout") {
		t.Fatalf("result artifact did not include unsupported details: %s", data)
	}
}

func TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls(t *testing.T) {
	forbidden := []string{
		"Apple Notes",
		"manual-using-apple-note",
		"reference-renders",
		"LibreOffice",
		"soffice",
		"PowerPoint",
		"Keynote",
		"Google Slides",
		"chromedp",
		"playwright",
		"selenium",
		"exec.Command",
		"WHO-HIV-testing-algorithms-toolkit",
		"EPA-generate-2021-presentation",
		"EPA-metal-coil-NESHAP-2018",
		"EPA-residential-wood-MacCarty",
	}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Errorf("%s contains forbidden renderer implementation string %q", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan renderer implementation: %v", err)
	}
}

func writeRealWorldDiffArtifacts(t *testing.T, gotPath string, referencePath string, deckInput string, slideNumber int, result model.CommandResult, diff imageDiff) {
	t.Helper()
	artifactRoot := os.Getenv("PUPPT_REALWORLD_ARTIFACT_DIR")
	if artifactRoot == "" {
		return
	}
	label := strings.TrimSuffix(filepath.Base(deckInput), filepath.Ext(deckInput))
	slideDir := filepath.Join(artifactRoot, label, fmt.Sprintf("slide-%03d", slideNumber))
	if err := os.MkdirAll(slideDir, 0o755); err != nil {
		t.Fatalf("create artifact dir %s: %v", slideDir, err)
	}
	for _, item := range []struct {
		source string
		name   string
	}{
		{source: gotPath, name: "got.png"},
		{source: referencePath, name: "reference.png"},
	} {
		if err := copyFile(item.source, filepath.Join(slideDir, item.name)); err != nil {
			t.Fatalf("write %s artifact for %s slide %d: %v", item.name, deckInput, slideNumber, err)
		}
	}
	if err := writeDiffPNG(gotPath, referencePath, filepath.Join(slideDir, "diff.png")); err != nil {
		t.Fatalf("write diff artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "result.json"), result); err != nil {
		t.Fatalf("write result artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "diff.json"), realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write diff metadata artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
}

func copyFile(sourcePath string, targetPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

type realWorldDiffArtifact struct {
	DeckInput   string    `json:"deck_input"`
	SlideNumber int       `json:"slide_number"`
	Diff        imageDiff `json:"diff"`
}

func writeJSONFile(targetPath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(targetPath, data, 0o644)
}

func writeDiffPNG(gotPath string, referencePath string, targetPath string) error {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return err
	}
	reference, err := decodePNGFile(referencePath)
	if err != nil {
		return err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := max(gotBounds.Dx(), referenceBounds.Dx())
	height := max(gotBounds.Dy(), referenceBounds.Dy())
	if width <= 0 || height <= 0 {
		return writePNG(targetPath, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	}
	diff := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotInside := x < gotBounds.Dx() && y < gotBounds.Dy()
			referenceInside := x < referenceBounds.Dx() && y < referenceBounds.Dy()
			if !gotInside || !referenceInside {
				diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
				continue
			}
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr == rr && gg == rg && gb == rb && ga == ra {
				diff.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				continue
			}
			diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	return writePNG(targetPath, diff)
}

type referenceManifest struct {
	Decks []struct {
		Input  string `json:"input"`
		Slides []struct {
			File   string `json:"file"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"slides"`
	} `json:"decks"`
}

type imageDiff struct {
	Width           int `json:"width"`
	Height          int `json:"height"`
	GotWidth        int `json:"got_width"`
	GotHeight       int `json:"got_height"`
	DifferentPixels int `json:"different_pixels"`
}

func referenceDPIForSlide(width int) int {
	if width <= 0 {
		return defaultOutputDPI
	}
	return normalizeOutputDPI(int(math.Round(float64(width) * emuPerInch / defaultSlideCX)))
}

func comparePNG(gotPath string, wantPath string) (imageDiff, error) {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return imageDiff{}, err
	}
	want, err := decodePNGFile(wantPath)
	if err != nil {
		return imageDiff{}, err
	}
	gotBounds := got.Bounds()
	wantBounds := want.Bounds()
	diff := imageDiff{Width: wantBounds.Dx(), Height: wantBounds.Dy(), GotWidth: gotBounds.Dx(), GotHeight: gotBounds.Dy()}
	if gotBounds.Dx() != wantBounds.Dx() || gotBounds.Dy() != wantBounds.Dy() {
		diff.DifferentPixels = max(gotBounds.Dx(), wantBounds.Dx()) * max(gotBounds.Dy(), wantBounds.Dy())
		return diff, nil
	}
	for y := 0; y < wantBounds.Dy(); y++ {
		for x := 0; x < wantBounds.Dx(); x++ {
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := want.At(wantBounds.Min.X+x, wantBounds.Min.Y+y).RGBA()
			if gr != wr || gg != wg || gb != wb || ga != wa {
				diff.DifferentPixels++
			}
		}
	}
	return diff, nil
}

func decodePNG(t *testing.T, path string) image.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	return img
}

func hasOpaquePixel(img image.Image) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0 {
				return true
			}
		}
	}
	return false
}

func minPathY(points []pathPoint) float64 {
	minY := math.Inf(1)
	for _, point := range points {
		if point.Y < minY {
			minY = point.Y
		}
	}
	return minY
}

func maxPathY(points []pathPoint) float64 {
	maxY := math.Inf(-1)
	for _, point := range points {
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	return maxY
}

func maxPathX(points []pathPoint) float64 {
	maxX := math.Inf(-1)
	for _, point := range points {
		if point.X > maxX {
			maxX = point.X
		}
	}
	return maxX
}

func opaqueBounds(img image.Image) image.Rectangle {
	bounds := img.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX+1, maxY+1)
}

func hasColorPixel(img image.Image, want color.RGBA) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if uint8(r>>8) == want.R && uint8(g>>8) == want.G && uint8(b>>8) == want.B && uint8(a>>8) == want.A {
				return true
			}
		}
	}
	return false
}

func decodePNGFile(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func writePicturePPTX(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	parts := map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`),
		"_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`),
		"ppt/presentation.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst>
  <p:sldSz cx="12192000" cy="6858000"/>
</p:presentation>`),
		"ppt/_rels/presentation.xml.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`),
		"ppt/slides/slide1.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
      <p:pic>
        <p:nvPicPr><p:cNvPr id="2" name="Picture 1"/><p:cNvPicPr/><p:nvPr/></p:nvPicPr>
        <p:blipFill><a:blip r:embed="rId1"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>
        <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></p:spPr>
      </p:pic>
    </p:spTree>
  </p:cSld>
</p:sld>`),
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/>
</Relationships>`),
		"ppt/media/image1.png": redPNG(),
	}
	for _, name := range []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"ppt/presentation.xml",
		"ppt/_rels/presentation.xml.rels",
		"ppt/slides/slide1.xml",
		"ppt/slides/_rels/slide1.xml.rels",
		"ppt/media/image1.png",
	} {
		writer, err := archive.Create(name)
		if err != nil {
			archive.Close()
			return err
		}
		if _, err := writer.Write(parts[name]); err != nil {
			archive.Close()
			return err
		}
	}
	return archive.Close()
}

func redPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		panic(err)
	}
	return data.Bytes()
}

func testPNGChunk(chunkType string, payload []byte) []byte {
	data := make([]byte, 0, 12+len(payload))
	data = appendTestUint32(data, uint32(len(payload)))
	data = append(data, []byte(chunkType)...)
	data = append(data, payload...)
	data = appendTestUint32(data, 0)
	return data
}

func testICCProfileData() []byte {
	tags := []struct {
		signature string
		payload   []byte
	}{
		{"rXYZ", testICCXYZTag(0.4360747, 0.2225045, 0.0139322)},
		{"gXYZ", testICCXYZTag(0.3850649, 0.7168786, 0.0971045)},
		{"bXYZ", testICCXYZTag(0.1430804, 0.0606169, 0.7141733)},
		{"rTRC", testICCGammaCurveTag(1)},
		{"gTRC", testICCGammaCurveTag(1)},
		{"bTRC", testICCGammaCurveTag(1)},
	}
	data := make([]byte, 128)
	copy(data[16:20], "RGB ")
	copy(data[20:24], "XYZ ")
	data = appendTestUint32(data, uint32(len(tags)))
	tableOffset := len(data)
	data = append(data, make([]byte, len(tags)*12)...)
	payloadOffset := len(data)
	for index, tag := range tags {
		entry := tableOffset + index*12
		copy(data[entry:entry+4], tag.signature)
		putTestUint32(data[entry+4:entry+8], uint32(payloadOffset))
		putTestUint32(data[entry+8:entry+12], uint32(len(tag.payload)))
		data = append(data, tag.payload...)
		payloadOffset += len(tag.payload)
	}
	return data
}

func testICCXYZTag(x float64, y float64, z float64) []byte {
	data := make([]byte, 20)
	copy(data[:4], "XYZ ")
	putTestFixed16(data[8:12], x)
	putTestFixed16(data[12:16], y)
	putTestFixed16(data[16:20], z)
	return data
}

func testICCGammaCurveTag(gamma float64) []byte {
	data := make([]byte, 14)
	copy(data[:4], "curv")
	putTestUint32(data[8:12], 1)
	putTestUint16(data[12:14], uint16(math.Round(gamma*256)))
	return data
}

func appendTestUint32(data []byte, value uint32) []byte {
	return append(data, byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
}

func putTestUint32(data []byte, value uint32) {
	data[0] = byte(value >> 24)
	data[1] = byte(value >> 16)
	data[2] = byte(value >> 8)
	data[3] = byte(value)
}

func putTestUint16(data []byte, value uint16) {
	data[0] = byte(value >> 8)
	data[1] = byte(value)
}

func putTestFixed16(data []byte, value float64) {
	putTestUint32(data, uint32(int32(math.Round(value*65536))))
}

func redGreenPNG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, image.Rect(0, 0, 2, 4), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(2, 0, 4, 4), &image.Uniform{C: color.RGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		panic(err)
	}
	return data.Bytes()
}
