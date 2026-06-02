package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strings"
	"testing"
)

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

func TestRenderShapeReportsUnsupportedOuterShadowTransforms(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:            "sp",
		Name:            "Scaled Shadow",
		PrstGeom:        "rect",
		HasTransform:    true,
		ExtCX:           emuPerInch / 2,
		ExtCY:           emuPerInch / 2,
		HasShadow:       true,
		ShadowColor:     color.RGBA{A: 128},
		ShadowDistance:  91440,
		ShadowDirection: 0,
		HasShadowScaleX: true,
		ShadowScaleX:    120000,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "outer shadow scale/skew transform") {
		t.Fatalf("expected unsupported shadow transform report, got unsupported=%+v", unsupported)
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
