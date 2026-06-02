package render

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	xdraw "golang.org/x/image/draw"
)

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

func TestRenderShapeAntialiasesFractionalRectangleFillEdges(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		Name:         "Fractional Rectangle",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch / 4,
		HasFill:      true,
		FillColor:    color.RGBA{G: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected shape render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(5, 1); got != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("expected fully covered rectangle interior, got %#v", got)
	}
	edge := img.RGBAAt(5, 2)
	if edge.R < 126 || edge.R > 128 || edge.G != 255 || edge.B < 126 || edge.B > 128 {
		t.Fatalf("expected half-covered fractional bottom edge to blend against background, got %#v", edge)
	}
	if got := img.RGBAAt(5, 3); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("expected pixels beyond fractional rectangle to remain untouched, got %#v", got)
	}
}

func TestFillShapeRectWithFloatBoundsPreservesThinFractionalCoverage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	fillShapeRectWithFloatBounds(img, image.Rect(0, 0, 4, 4), floatRect{MinX: 0, MinY: 0, MaxX: 4, MaxY: 2.2}, color.RGBA{B: 255, A: 255})

	edge := img.RGBAAt(2, 2)
	if edge.R < 203 || edge.R > 205 || edge.G < 203 || edge.G > 205 || edge.B != 255 {
		t.Fatalf("expected exact 20%% bottom-edge coverage to be retained, got %#v", edge)
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

func TestRenderShapeHonorsFlatLineCapOnDashedLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "cxnSp",
		Name:         "Dashed Flat Connector",
		PrstGeom:     "line",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 2,
		ExtCX:        emuPerInch / 2,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    114300,
		LineDash:     "dash",
		LineCap:      "flat",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected dashed flat-cap line render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(20, 44).RGBA(); a != 0 {
		t.Fatalf("dashed flat cap should not extend before its endpoint, got alpha=%04x", a)
	}
	if got := img.RGBAAt(24, 48); got.R == 0 || got.A == 0 {
		t.Fatalf("expected dashed flat-cap line to paint at its authored endpoint, got %#v", got)
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
	if _, _, _, a := img.At(0, 1).RGBA(); a != 0xffff {
		t.Fatalf("default square-cap dashed outline should antialias through the stroke width, got alpha=%04x", a)
	}
}

func TestRenderShapeUsesStrokeWidthForSystemDotRectOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "System Dot Rectangle",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    19050,
		LineDash:     "sysDot",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected system-dot outline render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	for _, x := range []int{0, 1, 4, 5} {
		if _, _, _, a := img.At(x, 0).RGBA(); a == 0 {
			t.Fatalf("expected system-dot dash at x=%d", x)
		}
	}
	for _, x := range []int{3, 7} {
		if _, _, _, a := img.At(x, 0).RGBA(); a != 0 {
			t.Fatalf("expected system-dot gap at x=%d, got alpha=%04x", x, a)
		}
	}
}

func TestRenderShapeHonorsExplicitFlatCapForSystemDotRectOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Flat Cap System Dot Rectangle",
		PrstGeom:     "rect",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    19050,
		LineDash:     "sysDot",
		LineCap:      "flat",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected flat-cap system-dot outline render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(2, 0).RGBA(); a == 0 {
		t.Fatal("expected explicit flat cap to draw through the dash segment endpoint")
	}
	if _, _, _, a := img.At(3, 0).RGBA(); a != 0 {
		t.Fatalf("expected flat-cap system-dot gap after endpoint, got alpha=%04x", a)
	}
}

func TestRenderShapeHonorsCenteredRectLineAlignment(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Centered Stroke Rectangle",
		PrstGeom:     "rect",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    19050,
		LineAlign:    "ctr",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected centered outline render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(25, 48); got.A != 0 {
		t.Fatalf("centered 2px stroke should not paint two pixels inside the left edge, got %#v", got)
	}
	if got := img.RGBAAt(24, 48); got.R != 255 || got.A != 255 {
		t.Fatalf("centered 2px stroke should paint the boundary pixel, got %#v", got)
	}
	if got := img.RGBAAt(23, 48); got.R != 255 || got.A != 255 {
		t.Fatalf("centered 2px stroke should paint one pixel outside the left edge, got %#v", got)
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

func TestRenderShapePaintsOvalLineMarkerType(t *testing.T) {
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
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected oval line marker render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if countOpaquePixelsWithRed(img) == 0 {
		t.Fatal("expected oval line marker to paint red pixels")
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

func TestRenderShapeReportsUnsupportedVisibleShape3DProperties(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:            "sp",
		Name:            "Beveled Shape",
		PrstGeom:        "rect",
		HasTransform:    true,
		ExtCX:           emuPerInch / 2,
		ExtCY:           emuPerInch / 2,
		HasFill:         true,
		FillColor:       color.RGBA{R: 255, A: 255},
		HasShape3D:      true,
		Shape3DFeatures: []string{"3-D top bevel", "3-D scene camera orthographicFront"},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "3-D top bevel") || !strings.Contains(unsupported[0].Message, "3-D scene camera orthographicFront") || !element.Rendered {
		t.Fatalf("expected unsupported 3-D shape report with flat render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestRenderShapePaintsSoftEdgeEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "sp",
		Name:           "Soft Shape",
		PrstGeom:       "rect",
		HasTransform:   true,
		ExtCX:          emuPerInch / 2,
		ExtCY:          emuPerInch / 2,
		HasFill:        true,
		FillColor:      color.RGBA{R: 255, A: 255},
		HasSoftEdge:    true,
		SoftEdgeRadius: 203200,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported soft edge render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	_, _, _, edgeAlpha := img.At(0, 0).RGBA()
	if edgeAlpha == 0 || edgeAlpha == 0xffff {
		t.Fatalf("expected soft edge to blur shape edge alpha, got alpha=%04x", edgeAlpha)
	}
	r, g, b, a := img.At(24, 24).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected soft edge center to remain red, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
}

func TestRenderShapeReportsSoftEdgeOnlyWhenShapeLayerCannotRender(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "sp",
		Name:           "Unsupported Soft Shape",
		PrstGeom:       "unsupportedShape",
		HasTransform:   true,
		ExtCX:          emuPerInch / 2,
		ExtCY:          emuPerInch / 2,
		HasFill:        true,
		FillColor:      color.RGBA{R: 255, A: 255},
		HasSoftEdge:    true,
		SoftEdgeRadius: 203200,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "soft edge effect") || element.Rendered {
		t.Fatalf("expected unsupported soft edge report only when layer cannot render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
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

func TestRenderShapePaintsCustomGeometryFillWithFractionalBounds(t *testing.T) {
	size := slideSize{CX: 100, CY: 100}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		Kind:         "sp",
		Name:         "Fractional Freeform 1",
		HasTransform: true,
		ExtCX:        100,
		ExtCY:        25,
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
	if got := img.RGBAAt(5, 1); got != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("expected fully covered custom geometry interior, got %#v", got)
	}
	edge := img.RGBAAt(5, 2)
	if edge.R < 126 || edge.R > 128 || edge.G < 126 || edge.G > 128 || edge.B != 255 {
		t.Fatalf("expected fractional custom geometry edge coverage, got %#v", edge)
	}
	if got := img.RGBAAt(5, 3); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("expected pixels beyond fractional custom geometry to remain untouched, got %#v", got)
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

func TestCollectSlideElementsParsesBlipRotWithShape(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
      <p:pic>
        <p:nvPicPr><p:cNvPr id="5" name="Fixed Image Rotation"/></p:nvPicPr>
        <p:blipFill rotWithShape="0"><a:blip r:embed="rId1"/></p:blipFill>
        <p:spPr><a:xfrm rot="5400000"><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm></p:spPr>
      </p:pic>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one element, got %+v", elements)
	}
	if !elements[0].HasBlipRotWithShape || elements[0].BlipRotWithShape {
		t.Fatalf("expected DrawingML blipFill rotWithShape=0 to be parsed, got %+v", elements[0])
	}
	if pictureRotatesWithShape(elements[0]) {
		t.Fatalf("expected picture raster to ignore shape rotation when rotWithShape=0, got %+v", elements[0])
	}
	if !pictureRotatesWithShape(slideElement{}) {
		t.Fatal("missing rotWithShape should default to rotating the blip with its shape")
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
	          <a:ln w="9525" cap="rnd" algn="ctr"><a:solidFill><a:srgbClr val="0000FF"/></a:solidFill><a:prstDash val="dash"/></a:ln>
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
	if elements[0].LineAlign != "ctr" {
		t.Fatalf("expected line alignment to be parsed, got %+v", elements[0])
	}
}

func TestLineDashPatternPixelsUsesDrawingMLPresetRuns(t *testing.T) {
	tests := []struct {
		name  string
		dash  string
		width int
		want  []int
	}{
		{name: "dot", dash: "dot", width: 2, want: []int{2, 6}},
		{name: "dash", dash: "dash", width: 2, want: []int{8, 6}},
		{name: "large dash", dash: "lgDash", width: 2, want: []int{16, 6}},
		{name: "dash dot", dash: "dashDot", width: 2, want: []int{8, 6, 2, 6}},
		{name: "large dash dot", dash: "lgDashDot", width: 2, want: []int{16, 6, 2, 6}},
		{name: "large dash dot dot", dash: "lgDashDotDot", width: 2, want: []int{16, 6, 2, 6, 2, 6}},
		{name: "system dot", dash: "sysDot", width: 2, want: []int{2, 2}},
		{name: "system dash", dash: "sysDash", width: 2, want: []int{6, 2}},
		{name: "system dash dot", dash: "sysDashDot", width: 2, want: []int{6, 2, 2, 2}},
		{name: "system dash dot dot", dash: "sysDashDotDot", width: 2, want: []int{6, 2, 2, 2, 2, 2}},
		{name: "minimum unit", dash: "sysDot", width: 0, want: []int{1, 1}},
		{name: "unknown fallback", dash: "unknown", width: 2, want: []int{8, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lineDashPatternPixels(tt.dash, tt.width)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("lineDashPatternPixels(%q, %d) = %v, want %v", tt.dash, tt.width, got, tt.want)
			}
		})
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

func TestCollectSlideElementsParsesVisibleShape3DProperties(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="12" name="Beveled Shape"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="rect"/>
          <a:scene3d>
            <a:camera prst="orthographicFront" fov="5400000" zoom="120000"><a:rot lat="1" lon="2" rev="3"/></a:camera>
            <a:lightRig rig="threePt" dir="t"><a:rot lat="4" lon="5" rev="6"/></a:lightRig>
            <a:backdrop><a:anchor x="0" y="0" z="0"/><a:norm dx="0" dy="0" dz="1"/><a:up dx="0" dy="1" dz="0"/></a:backdrop>
          </a:scene3d>
          <a:sp3d z="63500" extrusionH="165100" contourW="50800"><a:bevelT/></a:sp3d>
        </p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	for _, feature := range []string{
		"3-D top bevel",
		"3-D extrusion",
		"3-D contour",
		"3-D z offset",
		"3-D scene camera orthographicFront",
		"3-D scene camera field of view",
		"3-D scene camera zoom",
		"3-D scene camera rotation",
		"3-D scene light rig threePt/t",
		"3-D scene light rig rotation",
		"3-D scene backdrop",
	} {
		if !elements[0].HasShape3D || !slices.Contains(elements[0].Shape3DFeatures, feature) {
			t.Fatalf("expected visible 3-D feature %q to be parsed, got %+v", feature, elements[0])
		}
	}
}

func TestCollectSlideElementsParsesDefaultShape3DBevel(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="12" name="Default Bevel"/></p:nvSpPr>
	  <p:spPr><a:sp3d><a:bevelT/></a:sp3d></p:spPr>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.HasShape3D || !slices.Contains(got.Shape3DFeatures, "3-D top bevel") {
		t.Fatalf("schema-default 3-D bevel dimensions should be reported as visible content: %+v", got)
	}
}

func TestCollectSlideElementsParsesNonVisualTextBoxFlag(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="8" name="TextBox 7"/><p:cNvSpPr txBox="1"><a:spLocks noGrp="1" noRot="0" noTextEdit="1"/></p:cNvSpPr><p:nvPr/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/></p:spPr>
	  <p:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>Text</a:t></a:r></a:p></p:txBody>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.IsTextBox {
		t.Fatalf("expected cNvSpPr txBox metadata to be preserved: %+v", got)
	}
	if len(got.NonVisualLocks) != 2 || got.NonVisualLocks[0] != "spLocks.noGrp" || got.NonVisualLocks[1] != "spLocks.noTextEdit" {
		t.Fatalf("expected true non-visual shape lock flags to be preserved deterministically: %+v", got.NonVisualLocks)
	}
}

func TestCollectSlideElementsParsesPictureLockFlags(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:pic xmlns:p="p" xmlns:a="a" xmlns:r="r" xmlns:adec="http://schemas.microsoft.com/office/drawing/2017/decorative" xmlns:a16="http://schemas.microsoft.com/office/drawing/2014/main">
	  <p:nvPicPr><p:cNvPr id="3" name="Picture 2" descr="Diagram description" title="Lifecycle chart" hidden="0"><a:extLst><a:ext uri="{FF2B5EF4-FFF2-40B4-BE49-F238E27FC236}"><a16:creationId id="{D370CED6-CFF9-4448-B657-3980B0E8E342}"/></a:ext><a:ext uri="{C183D7F6-B498-43B3-948B-1728B52AA6E4}"><adec:decorative val="1"/></a:ext></a:extLst></p:cNvPr><p:cNvPicPr><a:picLocks noChangeAspect="1" noCrop="0"/></p:cNvPicPr><p:nvPr/></p:nvPicPr>
	  <p:blipFill><a:blip r:embed="rId4"/><a:stretch><a:fillRect/></a:stretch></p:blipFill>
	  <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"/></p:spPr>
	</p:pic>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.Description != "Diagram description" || got.Title != "Lifecycle chart" {
		t.Fatalf("expected cNvPr description/title metadata to be preserved: %+v", got)
	}
	if got.CreationID != "{D370CED6-CFF9-4448-B657-3980B0E8E342}" {
		t.Fatalf("expected cNvPr creationId metadata to be preserved: %+v", got)
	}
	if !got.HasHidden || got.Hidden || !got.HasDecorative || !got.Decorative {
		t.Fatalf("expected cNvPr hidden/decorative flags to be preserved: %+v", got)
	}
	if len(got.NonVisualProperties) != 2 || got.NonVisualProperties[0] != "decorative=true" || got.NonVisualProperties[1] != "hidden=false" {
		t.Fatalf("expected deterministic cNvPr boolean properties: %+v", got.NonVisualProperties)
	}
	if len(got.NonVisualLocks) != 1 || got.NonVisualLocks[0] != "picLocks.noChangeAspect" {
		t.Fatalf("expected true picture lock flag to be preserved: %+v", got.NonVisualLocks)
	}
}

func TestCollectSlideElementsIgnoresOneDimensionalShape3DBevel(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="12" name="Flat Bevel"/></p:nvSpPr>
	  <p:spPr><a:sp3d><a:bevelT w="63500" h="0"/></a:sp3d></p:spPr>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.HasShape3D {
		t.Fatalf("one-dimensional 3-D bevel should not be reported as visible content: %+v", got)
	}
}

func TestCollectSlideElementsParsesScene3DWithoutShape3D(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="12" name="Scene Only"/></p:nvSpPr>
	  <p:spPr><a:scene3d><a:camera prst="perspectiveFront"/><a:lightRig rig="soft" dir="br"/></a:scene3d></p:spPr>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if !got.HasShape3D || !slices.Contains(got.Shape3DFeatures, "3-D scene camera perspectiveFront") || !slices.Contains(got.Shape3DFeatures, "3-D scene light rig soft/br") {
		t.Fatalf("expected visible 3-D scene properties to be parsed, got %+v", got)
	}
}

func TestCollectSlideElementsIgnoresZeroSizedShape3DBevel(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="12" name="Flat 3D Defaults"/></p:nvSpPr>
	  <p:spPr><a:sp3d prstMaterial="plastic"><a:bevelT w="0" h="0"/></a:sp3d></p:spPr>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.HasShape3D {
		t.Fatalf("zero-sized 3-D defaults should not be reported as visible content: %+v", got)
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

func TestParseCustomGeometryPathPreservesCubicCommands(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:cubicBezTo><a:pt x="50" y="0"/><a:pt x="100" y="50"/><a:pt x="100" y="100"/></a:cubicBezTo><a:close/></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}
	_, commands, unsupported := parseCustomGeometryPathCommandsWithDiagnostics(root)
	if len(unsupported) != 0 {
		t.Fatalf("unexpected unsupported diagnostics: %+v", unsupported)
	}
	if len(commands) != 3 || commands[0].Kind != "moveTo" || commands[1].Kind != "cubicBezTo" || commands[2].Kind != "close" {
		t.Fatalf("expected DrawingML path commands to be preserved, got %+v", commands)
	}
	if len(commands[1].Points) != 3 || commands[1].Points[2] != (pathPoint{X: 1, Y: 1}) {
		t.Fatalf("unexpected cubic command points: %+v", commands[1])
	}
}

func TestParseCustomGeometryPathSupportsArcCommands(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:lnTo><a:pt x="100" y="0"/></a:lnTo><a:arcTo wR="10" hR="10" stAng="0" swAng="5400000"/><a:lnTo><a:pt x="0" y="100"/></a:lnTo><a:close/></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}

	points, unsupported := parseCustomGeometryPathWithDiagnostics(root)
	if len(points) < 3 {
		t.Fatalf("expected supported path segments to be retained, got %+v", points)
	}
	if len(unsupported) != 0 {
		t.Fatalf("expected arcTo to be supported, got %+v", unsupported)
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
	root, err := parseXMLNode([]byte(`<p:graphicFrame xmlns:p="p" xmlns:a="a" xmlns:a16="http://schemas.microsoft.com/office/drawing/2014/main">
		<p:nvGraphicFramePr><p:cNvPr id="4" name="Table 1"/></p:nvGraphicFramePr>
		<p:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm>
		<a:graphic><a:graphicData><a:tbl>
			<a:tblPr firstRow="1" firstCol="1"><a:tableStyleId>{STYLE-READ}</a:tableStyleId></a:tblPr>
			<a:tblGrid>
				<a:gridCol w="300000"><a:extLst><a:ext uri="{9D8B030D-6E8A-4147-A177-3AD203B41FA5}"><a16:colId val="111"/></a:ext></a:extLst></a:gridCol>
				<a:gridCol w="600000"><a:extLst><a:ext uri="{9D8B030D-6E8A-4147-A177-3AD203B41FA5}"><a16:colId val="222"/></a:ext></a:extLst></a:gridCol>
			</a:tblGrid>
			<a:tr h="200000"><a:extLst><a:ext uri="{0D108BD9-81ED-4DB2-BD59-A6C34878D82A}"><a16:rowId val="333"/></a:ext></a:extLst>
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
	if !slices.Equal(element.Table.ColumnIDs, []string{"111", "222"}) {
		t.Fatalf("unexpected table column ids: %+v", element.Table.ColumnIDs)
	}
	if !element.Table.FirstRow || !element.Table.FirstCol || element.Table.StyleID != "{STYLE-READ}" {
		t.Fatalf("unexpected table properties: %+v", element.Table)
	}
	if len(element.Table.Rows) != 1 || len(element.Table.Rows[0].Cells) != 2 {
		t.Fatalf("unexpected table rows: %+v", element.Table.Rows)
	}
	if element.Table.Rows[0].ID != "333" {
		t.Fatalf("unexpected table row id: %+v", element.Table.Rows[0])
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

func TestParseTableModelReadsTablePropertiesFill(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblPr><a:solidFill><a:srgbClr val="123456"/></a:solidFill></a:tblPr>
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p/></a:txBody><a:tcPr/></a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseTableModel(root, defaultThemeColors())
	if !got.HasBackground || got.NoBackground || got.Background.Color != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}) {
		t.Fatalf("expected direct table background fill, got %+v", got)
	}
}

func TestParseTableModelReadsTablePropertiesNoFill(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblPr><a:noFill/></a:tblPr>
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p/></a:txBody><a:tcPr/></a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseTableModel(root, defaultThemeColors())
	if !got.NoBackground || got.HasBackground {
		t.Fatalf("expected direct table background noFill, got %+v", got)
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

func TestParseTableCellAnchorCenterLowersToTextElement(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tc xmlns:a="a">
		<a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
		<a:tcPr anchor="ctr" anchorCtr="1" horzOverflow="clip" vertOverflow="overflow" vert="vert270"/>
	</a:tc>`))
	if err != nil {
		t.Fatal(err)
	}

	cell := parseTableCell(root, defaultThemeColors())
	if cell.TextAnchor != "ctr" || !cell.HasTextAnchorCenter || !cell.TextAnchorCenter {
		t.Fatalf("expected table-cell anchor and anchorCtr to parse, got %+v", cell)
	}
	if !cell.HasTextHorizontalOverflow || cell.TextHorizontalOverflow != "clip" || !cell.HasTextVerticalOverflow || cell.TextVerticalOverflow != "overflow" || !cell.HasTextVertical || cell.TextVertical != "vert270" {
		t.Fatalf("expected table-cell overflow and vertical text properties to parse, got %+v", cell)
	}
	element := tableCellTextElement(tableStyleRegion{}, cell, cell.HasTextColor, cell.TextColor)
	if element.TextAnchor != "ctr" || !element.HasTextAnchorCenter || !element.TextAnchorCenter {
		t.Fatalf("expected table-cell anchorCtr to lower to text element, got %+v", element)
	}
	if !element.HasTextHorizontalOverflow || element.TextHorizontalOverflow != "clip" || !element.HasTextVerticalOverflow || element.TextVerticalOverflow != "overflow" || !element.HasTextVertical || element.TextVertical != "vert270" {
		t.Fatalf("expected table-cell overflow and vertical text properties to lower, got %+v", element)
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

func TestRenderGraphicFramePaintsTableStyleBackground(t *testing.T) {
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
			StyleID: "{STYLE-BG}",
			Columns: []int64{1},
			Rows:    []tableRow{{Height: 1, Cells: []tableCell{{}}}},
		},
	}
	styles := tableStyleSet{Styles: map[string]tableStyle{
		normalizedTableStyleID("{STYLE-BG}"): {
			ID:            "{STYLE-BG}",
			HasBackground: true,
			Background:    backgroundPaint{Color: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}},
		},
	}}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 48); got != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}) {
		t.Fatalf("expected table background to paint behind no-fill cells, got %#v", got)
	}
}

func TestRenderGraphicFrameUsesDirectTablePropertiesBackgroundBeforeStyle(t *testing.T) {
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
			StyleID:       "{STYLE-BG}",
			HasBackground: true,
			Background:    backgroundPaint{Color: color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}},
			Columns:       []int64{1},
			Rows:          []tableRow{{Height: 1, Cells: []tableCell{{}}}},
		},
	}
	styles := tableStyleSet{Styles: map[string]tableStyle{
		normalizedTableStyleID("{STYLE-BG}"): {
			ID:            "{STYLE-BG}",
			HasBackground: true,
			Background:    backgroundPaint{Color: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}},
		},
	}}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 48); got != (color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}) {
		t.Fatalf("expected direct table background to override style background, got %#v", got)
	}
}

func TestRenderGraphicFrameTablePropertiesNoFillSuppressesStyleBackground(t *testing.T) {
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
			StyleID:      "{STYLE-BG}",
			NoBackground: true,
			Columns:      []int64{1},
			Rows:         []tableRow{{Height: 1, Cells: []tableCell{{}}}},
		},
	}
	styles := tableStyleSet{Styles: map[string]tableStyle{
		normalizedTableStyleID("{STYLE-BG}"): {
			ID:            "{STYLE-BG}",
			HasBackground: true,
			Background:    backgroundPaint{Color: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}},
		},
	}}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 48); got.A != 0 {
		t.Fatalf("expected direct table noFill to suppress style background, got %#v", got)
	}
}

func TestRenderGraphicFramePaintsTableStyleBackgroundGradient(t *testing.T) {
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
			StyleID: "{STYLE-BG}",
			Columns: []int64{1},
			Rows:    []tableRow{{Height: 1, Cells: []tableCell{{}}}},
		},
	}
	styles := tableStyleSet{Styles: map[string]tableStyle{
		normalizedTableStyleID("{STYLE-BG}"): {
			ID:            "{STYLE-BG}",
			HasBackground: true,
			Background: backgroundPaint{
				HasGradient: true,
				Gradient: gradientPaint{Stops: []gradientStop{
					{Position: 0, Color: color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 0xff}},
					{Position: 100000, Color: color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}},
				}},
			},
		},
	}}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	top := img.RGBAAt(48, 10)
	bottom := img.RGBAAt(48, 86)
	if top.R >= bottom.R || top.B >= bottom.B {
		t.Fatalf("expected table background gradient to span entire table, top=%#v bottom=%#v", top, bottom)
	}
}

func TestRenderGraphicFramePaintsTableStyleBackgroundShadow(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Table 1",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		HasTable:     true,
		Table: tableModel{
			StyleID: "{STYLE-BG}",
			Columns: []int64{1},
			Rows:    []tableRow{{Height: 1, Cells: []tableCell{{}}}},
		},
	}
	styles := tableStyleSet{Styles: map[string]tableStyle{
		normalizedTableStyleID("{STYLE-BG}"): {
			ID:                  "{STYLE-BG}",
			HasBackground:       true,
			Background:          backgroundPaint{Color: color.RGBA{R: 0xff, A: 0xff}},
			HasBackgroundEffect: true,
			BackgroundEffect: themeEffectStyle{
				HasShadow:       true,
				ShadowColor:     color.RGBA{A: 0xff},
				ShadowDistance:  emuPerInch / 10,
				ShadowDirection: 0,
			},
		},
	}}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, styles)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected table render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if _, _, _, a := img.At(78, 48).RGBA(); a == 0 {
		t.Fatal("expected table background shadow to render behind the table")
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

func TestTableRowOffsetsPreserveSpanningFirstRowHeightBeforeStretchingBody(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		Rows: []tableRow{
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{ColSpan: 2}}},
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{}, {}}},
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{}, {}}},
		},
	}

	got := tableRowOffsets(table, 0, 100, 0, emuPerInch, emuPerInch, 100)
	want := []int{0, 25, 63, 100}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected spanning first row to keep authored height and body rows to fill frame, got %v want %v", got, want)
	}
}

func TestTableRowOffsetsScaleOrdinaryFirstRowWithTableFrame(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		Rows: []tableRow{
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{}, {}}},
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{}, {}}},
			{HasHeight: true, Height: emuPerInch / 4, Cells: []tableCell{{}, {}}},
		},
	}

	got := tableRowOffsets(table, 0, 100, 0, emuPerInch, emuPerInch, 100)
	want := []int{0, 33, 67, 100}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected ordinary first row to scale with the table frame, got %v want %v", got, want)
	}
}

func TestAdjustTableRowOffsetsForMinimumHeightsGrowsTextRowsInsideFrame(t *testing.T) {
	got := adjustTableRowOffsetsForMinimumHeights([]int{10, 30, 50, 70}, []int{32, 8, 8})
	want := []int{10, 42, 56, 70}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected text minimum row height to grow and shrink flexible rows inside frame, got %v want %v", got, want)
	}
}

func TestAdjustTableRowOffsetsForMinimumHeightsKeepsRowsWhenMinimumsCannotFit(t *testing.T) {
	input := []int{10, 30, 50, 70}
	got := adjustTableRowOffsetsForMinimumHeights(input, []int{50, 30, 30})
	if !reflect.DeepEqual(got, input) {
		t.Fatalf("expected impossible text minimums to keep existing row offsets, got %v want %v", got, input)
	}
}

func TestTableRowOffsetsWithZeroAuthoredHeightsGrowMultiParagraphHeader(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		BandRow:  true,
		Columns:  []int64{1000000, 1000000},
		Rows: []tableRow{
			{
				HasHeight: true,
				Height:    0,
				Cells: []tableCell{
					{
						Text:         "Assay\n1",
						FontSize:     1600,
						TextAnchor:   "ctr",
						HasMargins:   true,
						MarginTop:    0,
						MarginBottom: 0,
						TextParagraphs: []textParagraph{
							{Text: "Assay", TextAlign: "ctr", Runs: []textRun{{Text: "Assay", FontSize: 1600}}},
							{Text: "1", TextAlign: "ctr", Runs: []textRun{{Text: "1", FontSize: 1600}}},
						},
					},
					{Text: "Population", FontSize: 1600, TextAnchor: "ctr"},
				},
			},
			{HasHeight: true, Height: 0, Cells: []tableCell{{Text: "1", FontSize: 1600, HasMargins: true}, {Text: "General", FontSize: 1600, HasMargins: true}}},
			{HasHeight: true, Height: 0, Cells: []tableCell{{Text: "2", FontSize: 1600, HasMargins: true}, {Text: "Alternate", FontSize: 1600, HasMargins: true}}},
		},
	}
	base := tableRowOffsets(table, 0, 60, 0, emuPerInch, emuPerInch, 60)
	minimums := tableTextMinimumRowHeights(table, tableStyleSet{}, []int{0, 30, 90}, image.Rect(0, 0, 90, 60), slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 90, 60), defaultOutputDPI)
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, []int{0, 30, 90}, base, image.Rect(0, 0, 90, 60), slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 90, 60), defaultOutputDPI)
	if got[1]-got[0] <= base[1]-base[0] {
		t.Fatalf("expected zero-height multi-paragraph header row to grow beyond equal distribution, minimums=%v base=%v got=%v", minimums, base, got)
	}
	if got[len(got)-1] != base[len(base)-1] {
		t.Fatalf("expected row growth to stay within table frame, base=%v got=%v", base, got)
	}
}

func TestTableTextMinimumRowHeightsMeasuresSpanningHeaderWidth(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		BandRow:  true,
		Columns:  []int64{1000000, 1000000, 1000000},
		Rows: []tableRow{
			{
				HasHeight: true,
				Height:    370840,
				Cells: []tableCell{
					{
						Text:       "Number of quality-assured products eligible for\nprocurement through WHO and Global Fund",
						FontSize:   1600,
						TextAnchor: "ctr",
						ColSpan:    3,
						TextParagraphs: []textParagraph{
							{Text: "Number of quality-assured products eligible for", TextAlign: "ctr", Runs: []textRun{{Text: "Number of quality-assured products eligible for", FontSize: 1600}}},
							{Text: "procurement through WHO and Global Fund", TextAlign: "ctr", Runs: []textRun{{Text: "procurement through WHO and Global Fund", FontSize: 1600}}},
						},
					},
					{HMerge: true},
					{HMerge: true},
				},
			},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{}, {Text: "WHO", FontSize: 1600}, {Text: "Global Fund", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Dual HIV/Syphilis RDTs", FontSize: 1600}, {Text: "3", FontSize: 1600}, {Text: "3", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "HIV RDTs", FontSize: 1600}, {Text: "19", FontSize: 1600}, {Text: "27", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "HIV EIAs", FontSize: 1600}, {Text: "4", FontSize: 1600}, {Text: "17", FontSize: 1600}}},
		},
	}
	narrow := tableTextMinimumRowHeights(table, tableStyleSet{}, []int{0, 80, 80, 80}, image.Rect(0, 0, 240, 100), slideSize{CX: 5 * 370840, CY: 5 * 370840}, image.Rect(0, 0, 240, 100), defaultOutputDPI)
	spanned := tableTextMinimumRowHeights(table, tableStyleSet{}, []int{0, 80, 160, 240}, image.Rect(0, 0, 240, 100), slideSize{CX: 5 * 370840, CY: 5 * 370840}, image.Rect(0, 0, 240, 100), defaultOutputDPI)
	if spanned[0] >= narrow[0] {
		t.Fatalf("expected spanning header minimum to use full spanned width, narrow=%v spanned=%v", narrow, spanned)
	}
	if spanned[0] <= 0 {
		t.Fatalf("expected measured spanning header minimum, got %v", spanned)
	}
}

func TestTableTextMinimumRowHeightsMeasuresAuthoredBlankParagraphs(t *testing.T) {
	table := tableModel{
		Columns: []int64{1000000},
		Rows: []tableRow{
			{
				HasHeight: true,
				Height:    0,
				Cells: []tableCell{{
					TextParagraphs: []textParagraph{
						{Text: "", FontSize: 1600},
						{Text: "", FontSize: 1600},
						{Text: "", FontSize: 1600},
					},
					HasMargins: true,
				}},
			},
			{
				HasHeight: true,
				Height:    0,
				Cells:     []tableCell{{Text: "Visible", FontSize: 1600, HasMargins: true}},
			},
		},
	}
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	canvas := image.Rect(0, 0, 100, 80)
	columns := []int{0, 100}

	minimums := tableTextMinimumRowHeights(table, tableStyleSet{}, columns, canvas, size, canvas, defaultOutputDPI)
	if minimums[0] <= minimums[1] {
		t.Fatalf("expected authored blank paragraph line boxes to size row, got %v", minimums)
	}

	rowOffsets := tableRowOffsets(table, 0, 80, 0, emuPerInch, emuPerInch, 80)
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, columns, rowOffsets, canvas, size, canvas, defaultOutputDPI)
	if got[1]-got[0] <= rowOffsets[1]-rowOffsets[0] {
		t.Fatalf("expected authored blank paragraph row to grow, base=%v got=%v minimums=%v", rowOffsets, got, minimums)
	}
}

func TestTableTextMinimumRowHeightsDistributesRowSpanText(t *testing.T) {
	spannedCell := tableCell{
		Text:      "Verification\n\n\n\nComplete",
		FontSize:  1600,
		TextAlign: "ctr",
		RowSpan:   3,
		TextParagraphs: []textParagraph{
			{Text: "Verification", TextAlign: "ctr", Runs: []textRun{{Text: "Verification", FontSize: 1600}}},
			{Text: "", TextAlign: "ctr"},
			{Text: "", TextAlign: "ctr"},
			{Text: "", TextAlign: "ctr"},
			{Text: "Complete", TextAlign: "ctr", Runs: []textRun{{Text: "Complete", FontSize: 1600}}},
		},
	}
	table := tableModel{
		Columns: []int64{1000000, 1000000},
		Rows: []tableRow{
			{HasHeight: true, Height: 370840, Cells: []tableCell{spannedCell, {Text: "A", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{VMerge: true}, {Text: "B", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{VMerge: true}, {Text: "C", FontSize: 1600}}},
		},
	}
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	canvas := image.Rect(0, 0, 120, 90)
	columnOffsets := []int{0, 60, 120}
	spannedMinimums := tableTextMinimumRowHeights(table, tableStyleSet{}, columnOffsets, canvas, size, canvas, defaultOutputDPI)
	unspanned := tableModel{
		Columns: table.Columns,
		Rows: []tableRow{
			{HasHeight: true, Height: 370840, Cells: []tableCell{spannedCell, {Text: "A", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{VMerge: true}, {Text: "B", FontSize: 1600}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{VMerge: true}, {Text: "C", FontSize: 1600}}},
		},
	}
	unspanned.Rows[0].Cells[0].RowSpan = 1
	unspannedMinimums := tableTextMinimumRowHeights(unspanned, tableStyleSet{}, columnOffsets, canvas, size, canvas, defaultOutputDPI)

	if spannedMinimums[0] <= 0 || spannedMinimums[0] >= unspannedMinimums[0] {
		t.Fatalf("expected row-spanned text minimum to be distributed, spanned=%v unspanned=%v", spannedMinimums, unspannedMinimums)
	}
	if spannedMinimums[1] != spannedMinimums[0] || spannedMinimums[2] != spannedMinimums[0] {
		t.Fatalf("expected row-spanned text minimum to apply across covered rows, got %v", spannedMinimums)
	}

	rowOffsets := []int{0, 30, 60, 90}
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, columnOffsets, rowOffsets, canvas, size, canvas, defaultOutputDPI)
	if !reflect.DeepEqual(got, rowOffsets) {
		t.Fatalf("row-spanned text that fits the spanned rectangle should not inflate only the origin row, base=%v got=%v minimums=%v", rowOffsets, got, spannedMinimums)
	}
}

func TestTableRowOffsetsWithTextMinimumsUsesMinimumProportionsWhenFrameIsOverCapacity(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		BandRow:  true,
		Columns:  []int64{1000000, 1000000, 1000000},
		Rows: []tableRow{
			{
				HasHeight: true,
				Height:    370840,
				Cells: []tableCell{
					{
						Text:      "Number of quality-assured products eligible for\nprocurement through WHO and Global Fund",
						FontSize:  1800,
						TextAlign: "ctr",
						ColSpan:   3,
						TextParagraphs: []textParagraph{
							{Text: "Number of quality-assured products eligible for", TextAlign: "ctr", FontSize: 1800, Runs: []textRun{{Text: "Number of quality-assured products eligible for", FontSize: 1800}}},
							{Text: "procurement through WHO and Global Fund", TextAlign: "ctr", FontSize: 1800, Runs: []textRun{{Text: "procurement through WHO and Global Fund", FontSize: 1800}}},
						},
					},
					{HMerge: true},
					{HMerge: true},
				},
			},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "WHO", FontSize: 1800, TextAlign: "ctr"}, {Text: "Global Fund", FontSize: 1800, TextAlign: "ctr"}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Dual HIV/Syphilis RDTs", FontSize: 1800, TextAlign: "ctr"}, {Text: "3", FontSize: 1800, TextAlign: "ctr"}, {Text: "3", FontSize: 1800, TextAlign: "ctr"}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "HIV RDTs", FontSize: 1800, TextAlign: "ctr"}, {Text: "19", FontSize: 1800, TextAlign: "ctr"}, {Text: "27", FontSize: 1800, TextAlign: "ctr"}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "HIV EIAs", FontSize: 1800, TextAlign: "ctr"}, {Text: "4", FontSize: 1800, TextAlign: "ctr"}, {Text: "17", FontSize: 1800, TextAlign: "ctr"}}},
		},
	}
	rowOffsets := []int{0, 29, 64, 98, 133, 167}
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, []int{0, 223, 365, 555}, rowOffsets, image.Rect(0, 0, 555, 167), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540), defaultOutputDPI)
	if got[1]-got[0] <= rowOffsets[1]-rowOffsets[0] {
		t.Fatalf("expected over-capacity text minimums to grow the multi-paragraph header row, base=%v got=%v", rowOffsets, got)
	}
	if got[len(got)-1] != rowOffsets[len(rowOffsets)-1] {
		t.Fatalf("expected over-capacity row reflow to stay inside frame, base=%v got=%v", rowOffsets, got)
	}
	if got[2]-got[1] >= rowOffsets[2]-rowOffsets[1] {
		t.Fatalf("expected body rows to compact under source text-minimum proportions, base=%v got=%v", rowOffsets, got)
	}
}

func TestTableRowOffsetsWithTextMinimumsReflowsNonSpanningFirstRowWhenFrameIsOverCapacity(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		BandRow:  true,
		Columns:  []int64{1000000, 1000000, 1000000},
		Rows: []tableRow{
			{HasHeight: true, Height: 696525, Cells: []tableCell{
				{Text: "Emission\nRate\nPM2.5\n(g/hr)", FontSize: 1600, TextParagraphs: []textParagraph{
					{Text: "Emission", FontSize: 1600, Runs: []textRun{{Text: "Emission", FontSize: 1600}}},
					{Text: "Rate", FontSize: 1600, Runs: []textRun{{Text: "Rate", FontSize: 1600}}},
					{Text: "PM2.5", FontSize: 1600, Runs: []textRun{{Text: "PM2.5", FontSize: 1600}}},
					{Text: "(g/hr)", FontSize: 1600, Runs: []textRun{{Text: "(g/hr)", FontSize: 1600}}},
				}},
				{Text: "Emission Factor PM2.5 (g/kg)", FontSize: 1600, TextParagraphs: []textParagraph{{Text: "Emission Factor PM2.5 (g/kg)", FontSize: 1600, Runs: []textRun{{Text: "Emission Factor PM2.5 (g/kg)", FontSize: 1600}}}}},
				{Text: "Firepower(W)", FontSize: 1600, TextParagraphs: []textParagraph{{Text: "Firepower(W)", FontSize: 1600, Runs: []textRun{{Text: "Firepower(W)", FontSize: 1600}}}}},
			}},
			{HasHeight: true, Height: 487575, Cells: []tableCell{{}, {}, {}}},
			{HasHeight: true, Height: 487575, Cells: []tableCell{{}, {}, {}}},
		},
	}
	rowOffsets := []int{0, 20, 35, 50}
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, []int{0, 58, 116, 174}, rowOffsets, image.Rect(0, 0, 174, 50), slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 174, 50), defaultOutputDPI)
	if got[1]-got[0] <= rowOffsets[1]-rowOffsets[0] {
		t.Fatalf("expected non-spanning source first row to grow when it is the only row over text minimum, base=%v got=%v", rowOffsets, got)
	}
	if got[len(got)-1] != rowOffsets[len(rowOffsets)-1] {
		t.Fatalf("expected non-spanning first-row reflow to stay inside frame, base=%v got=%v", rowOffsets, got)
	}
	if got[2]-got[1] >= rowOffsets[2]-rowOffsets[1] {
		t.Fatalf("expected body rows to compact under source text-minimum proportions, base=%v got=%v", rowOffsets, got)
	}
}

func TestTableRowOffsetsWithTextMinimumsDoesNotReproportionWhenSpanningHeaderAlreadyFits(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		Columns:  []int64{1000000, 1000000},
		Rows: []tableRow{
			{HasHeight: true, Height: 370840, Cells: []tableCell{
				{Text: "Header", FontSize: 1200, ColSpan: 2, TextParagraphs: []textParagraph{{Text: "Header", FontSize: 1200, Runs: []textRun{{Text: "Header", FontSize: 1200}}}}},
				{HMerge: true},
			}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Body row one wraps", FontSize: 1800, TextParagraphs: []textParagraph{{Text: "Body row one wraps", FontSize: 1800, Runs: []textRun{{Text: "Body row one wraps", FontSize: 1800}}}}}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Body row two wraps", FontSize: 1800, TextParagraphs: []textParagraph{{Text: "Body row two wraps", FontSize: 1800, Runs: []textRun{{Text: "Body row two wraps", FontSize: 1800}}}}}}},
		},
	}
	rowOffsets := []int{0, 30, 42, 54}
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, []int{0, 40, 80}, rowOffsets, image.Rect(0, 0, 80, 54), slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 80, 54), defaultOutputDPI)
	if !reflect.DeepEqual(got, rowOffsets) {
		t.Fatalf("spanning first row that already satisfies its text minimum should not trigger proportional fallback, base=%v got=%v", rowOffsets, got)
	}
}

func TestTableRowOffsetsWithTextMinimumsDoesNotReproportionWhenBodyRowsAlsoExceedCapacity(t *testing.T) {
	table := tableModel{
		FirstRow: true,
		Columns:  []int64{1000000, 1000000},
		Rows: []tableRow{
			{HasHeight: true, Height: 370840, Cells: []tableCell{
				{Text: "Long spanning header wraps", FontSize: 1800, ColSpan: 2, TextParagraphs: []textParagraph{{Text: "Long spanning header wraps", FontSize: 1800, Runs: []textRun{{Text: "Long spanning header wraps", FontSize: 1800}}}}},
				{HMerge: true},
			}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Long body row wraps too", FontSize: 1800, TextParagraphs: []textParagraph{{Text: "Long body row wraps too", FontSize: 1800, Runs: []textRun{{Text: "Long body row wraps too", FontSize: 1800}}}}}}},
			{HasHeight: true, Height: 370840, Cells: []tableCell{{Text: "Second body row wraps too", FontSize: 1800, TextParagraphs: []textParagraph{{Text: "Second body row wraps too", FontSize: 1800, Runs: []textRun{{Text: "Second body row wraps too", FontSize: 1800}}}}}}},
		},
	}
	rowOffsets := []int{0, 18, 36, 54}
	got := tableRowOffsetsWithTextMinimums(table, tableStyleSet{}, []int{0, 30, 60}, rowOffsets, image.Rect(0, 0, 60, 54), slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 60, 54), defaultOutputDPI)
	if !reflect.DeepEqual(got, rowOffsets) {
		t.Fatalf("multi-row over-capacity table should not use first-row header proportional fallback, base=%v got=%v", rowOffsets, got)
	}
}

func TestTableTextParagraphsWithColorOverridesParagraphDefaultsButPreservesRuns(t *testing.T) {
	styleColor := color.RGBA{R: 0xee, G: 0xee, B: 0xee, A: 0xff}
	paragraphs := []textParagraph{{
		HasTextColor: true,
		TextColor:    color.RGBA{A: 0xff},
		Runs: []textRun{
			{Text: "Styled"},
			{Text: "Direct", HasTextColor: true, TextColor: color.RGBA{R: 0xff, A: 0xff}},
		},
	}}

	got := tableTextParagraphsWithColor(paragraphs, "", styleColor)
	if len(got) != 1 || !got[0].HasTextColor || got[0].TextColor != styleColor {
		t.Fatalf("expected table style color to override paragraph default, got %+v", got)
	}
	defaultSegment := runToSegment(got[0].Runs[0], got[0])
	if !defaultSegment.HasTextColor || defaultSegment.TextColor != styleColor {
		t.Fatalf("expected uncolored run to inherit table style color, got %+v", defaultSegment)
	}
	directSegment := runToSegment(got[0].Runs[1], got[0])
	if !directSegment.HasTextColor || directSegment.TextColor != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected explicit run color to win over table style color, got %+v", directSegment)
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

func TestParseTableStylesReadsTableBackgroundFillReference(t *testing.T) {
	fillStyles := parseThemeFillStyles([]byte(`<a:theme xmlns:a="a">
		<a:themeElements><a:fmtScheme name="Office"><a:fillStyleLst>
			<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
		</a:fillStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	effectStyles := parseThemeEffectStyles([]byte(`<a:theme xmlns:a="a">
		<a:themeElements><a:fmtScheme name="Office"><a:effectStyleLst>
			<a:effectStyle><a:effectLst><a:outerShdw blurRad="40000" dist="20000" dir="5400000"><a:schemeClr val="phClr"><a:alpha val="38000"/></a:schemeClr></a:outerShdw></a:effectLst></a:effectStyle>
		</a:effectStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
		<a:tblStyle styleId="{STYLE-BG}" styleName="Background Style">
			<a:tblBg>
				<a:fillRef idx="1"><a:schemeClr val="accent2"/></a:fillRef>
				<a:effectRef idx="1"><a:schemeClr val="accent2"/></a:effectRef>
			</a:tblBg>
			<a:wholeTbl><a:tcStyle><a:fill><a:noFill/></a:fill></a:tcStyle></a:wholeTbl>
		</a:tblStyle>
	</a:tblStyleLst>`), themeColors{"accent2": {R: 0x22, G: 0x44, B: 0x66, A: 0xff}}, themeFonts{}, fillStyles, themeLineStyles{}, effectStyles)

	style, ok := styles.Styles[normalizedTableStyleID("{STYLE-BG}")]
	if !ok || !style.HasBackground {
		t.Fatalf("expected table background style, got %+v", styles)
	}
	if style.Background.Color != (color.RGBA{R: 0x22, G: 0x44, B: 0x66, A: 0xff}) {
		t.Fatalf("unexpected table background fill: %+v", style.Background)
	}
	if !style.HasBackgroundEffect || !style.BackgroundEffect.HasShadow || style.BackgroundEffect.ShadowColor.A == 0 {
		t.Fatalf("unexpected table background effect: %+v", style.BackgroundEffect)
	}
}

func TestParseTableStylesResolvesCellStyleFillReference(t *testing.T) {
	fillStyles := parseThemeFillStyles([]byte(`<a:theme xmlns:a="a">
		<a:themeElements><a:fmtScheme name="Office"><a:fillStyleLst>
			<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
		</a:fillStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	styles := parseTableStyles([]byte(`<a:tblStyleLst xmlns:a="a">
		<a:tblStyle styleId="{STYLE-FILLREF}" styleName="Cell FillRef Style">
			<a:wholeTbl>
				<a:tcStyle><a:fillRef idx="1"><a:schemeClr val="accent2"/></a:fillRef></a:tcStyle>
			</a:wholeTbl>
		</a:tblStyle>
	</a:tblStyleLst>`), themeColors{"accent2": {R: 0x44, G: 0x55, B: 0x66, A: 0xff}}, themeFonts{}, fillStyles, themeLineStyles{}, themeEffectStyles{})

	style, ok := styles.Styles[normalizedTableStyleID("{STYLE-FILLREF}")]
	if !ok {
		t.Fatalf("expected parsed fillRef table style, got %+v", styles)
	}
	whole := style.Regions["wholeTbl"]
	if !whole.HasFill || whole.FillColor != (color.RGBA{R: 0x44, G: 0x55, B: 0x66, A: 0xff}) {
		t.Fatalf("expected table cell fillRef to resolve through theme fill style, got %+v", whole)
	}
}

func TestThemeFillStylesResolveBackgroundFillReference(t *testing.T) {
	fillStyles := parseThemeFillStyles([]byte(`<a:theme xmlns:a="a">
		<a:themeElements><a:fmtScheme name="Office">
			<a:fillStyleLst>
				<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
			</a:fillStyleLst>
			<a:bgFillStyleLst>
				<a:solidFill><a:srgbClr val="010203"/></a:solidFill>
				<a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
			</a:bgFillStyleLst>
		</a:fmtScheme></a:themeElements>
	</a:theme>`))

	paint, ok := fillStyles.Style(1002, themeWithPlaceholderColor(defaultThemeColors(), color.RGBA{R: 0x33, G: 0x44, B: 0x55, A: 0xff}))
	if !ok || paint.Color != (color.RGBA{R: 0x33, G: 0x44, B: 0x55, A: 0xff}) {
		t.Fatalf("expected background fillRef idx 1002 to resolve second bgFillStyleLst entry, got=%+v ok=%v", paint, ok)
	}
	if _, ok := fillStyles.Style(1000, defaultThemeColors()); ok {
		t.Fatalf("expected fillRef idx 1000 to mean no fill")
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
	</a:tblStyleLst>`), defaultThemeColors(), themeFonts{MinorLatin: "Calibri"}, themeFillStyles{}, themeLineStyles{}, themeEffectStyles{})

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
	</a:tblStyleLst>`), themeColors{"accent2": {R: 12, G: 34, B: 56, A: 255}}, themeFonts{}, themeFillStyles{}, lineStyles, themeEffectStyles{})

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

	table.FirstRow = false
	noHeaderBand1 := resolvedTableCellStyle(table, styles, 0, 1)
	if !noHeaderBand1.HasFill || noHeaderBand1.FillColor != (color.RGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 0xff}) {
		t.Fatalf("expected first non-header row to use band1 style, got %+v", noHeaderBand1)
	}
	noHeaderBand2 := resolvedTableCellStyle(table, styles, 1, 1)
	if !noHeaderBand2.HasFill || noHeaderBand2.FillColor != (color.RGBA{R: 0xdd, G: 0xee, B: 0xff, A: 0xff}) {
		t.Fatalf("expected second non-header row to use band2 style, got %+v", noHeaderBand2)
	}
}

func TestTableStyleRegionNamesStartBandsAfterHeaderRegions(t *testing.T) {
	table := tableModel{
		Columns:  []int64{1, 1, 1},
		FirstRow: true,
		FirstCol: true,
		BandRow:  true,
		BandCol:  true,
		Rows: []tableRow{
			{Cells: []tableCell{{}, {}, {}}},
			{Cells: []tableCell{{}, {}, {}}},
			{Cells: []tableCell{{}, {}, {}}},
		},
	}

	if got := tableStyleRegionNamesForCell(table, 1, 1); !slices.Contains(got, "band1H") || !slices.Contains(got, "band1V") {
		t.Fatalf("expected first non-header row/column to use band1 regions, got %v", got)
	}
	if got := tableStyleRegionNamesForCell(table, 2, 2); !slices.Contains(got, "band2H") || !slices.Contains(got, "band2V") {
		t.Fatalf("expected second non-header row/column to use band2 regions, got %v", got)
	}

	table.FirstRow = false
	table.FirstCol = false
	if got := tableStyleRegionNamesForCell(table, 0, 0); !slices.Contains(got, "band1H") || !slices.Contains(got, "band1V") {
		t.Fatalf("expected first row/column without headers to use band1 regions, got %v", got)
	}
	if got := tableStyleRegionNamesForCell(table, 1, 1); !slices.Contains(got, "band2H") || !slices.Contains(got, "band2V") {
		t.Fatalf("expected second row/column without headers to use band2 regions, got %v", got)
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

func TestRenderTableCellBorderPaintsDoubleCompoundLine(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	border := tableCellBorder{
		Specified: true,
		HasLine:   true,
		Color:     color.RGBA{R: 0xff, A: 0xff},
		Width:     57150,
		Compound:  "dbl",
	}

	drawTableCellBorder(img, size, image.Rect(0, 0, 96, 96), image.Rect(12, 40, 84, 56), border, tableEdgeTop)

	if got := img.RGBAAt(48, 37); got != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected first double-line stroke, got %#v", got)
	}
	if _, _, _, a := img.At(48, 40).RGBA(); a != 0 {
		t.Fatalf("expected double-line gap to remain transparent, got alpha=%04x", a)
	}
	if got := img.RGBAAt(48, 43); got != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected second double-line stroke, got %#v", got)
	}
}

func TestRenderTableCellBorderPaintsKnownLineEndMarkers(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 64, 32))
	border := tableCellBorder{
		Specified:        true,
		HasLine:          true,
		Color:            color.RGBA{R: 0xff, A: 0xff},
		Width:            38100,
		Cap:              "flat",
		HeadMarker:       "triangle",
		HeadMarkerWidth:  "sm",
		HeadMarkerLength: "sm",
		TailMarker:       "diamond",
		TailMarkerWidth:  "sm",
		TailMarkerLength: "sm",
	}

	drawTableCellBorder(img, size, image.Rect(0, 0, 64, 32), image.Rect(16, 16, 49, 24), border, tableEdgeTop)

	if got := img.RGBAAt(24, 16); got.R == 0 || got.A == 0 {
		t.Fatalf("expected head marker to paint near start of table border, got %#v", got)
	}
	if got := img.RGBAAt(40, 16); got.R == 0 || got.A == 0 {
		t.Fatalf("expected tail marker to paint near end of table border, got %#v", got)
	}
}

func TestRenderTableCellDiagonalBorderPaintsKnownLineEndMarkers(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	border := tableCellBorder{
		Specified:  true,
		HasLine:    true,
		Color:      color.RGBA{B: 0xff, A: 0xff},
		Width:      38100,
		HeadMarker: "triangle",
		TailMarker: "oval",
	}

	drawTableCellDiagonalBorder(img, size, image.Rect(12, 12, 52, 52), border, true)

	if got := img.RGBAAt(20, 20); got.B == 0 || got.A == 0 {
		t.Fatalf("expected diagonal head marker to paint near start, got %#v", got)
	}
	if got := img.RGBAAt(44, 44); got.B == 0 || got.A == 0 {
		t.Fatalf("expected diagonal tail marker to paint near end, got %#v", got)
	}
}

func TestRenderTableCellBorderHonorsFlatLineCap(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	border := tableCellBorder{
		Specified: true,
		HasLine:   true,
		Color:     color.RGBA{R: 0xff, A: 0xff},
		Width:     57150,
		Cap:       "flat",
	}

	drawTableCellBorder(img, size, image.Rect(0, 0, 32, 32), image.Rect(10, 12, 24, 20), border, tableEdgeTop)

	if _, _, _, a := img.At(10, 12).RGBA(); a == 0 {
		t.Fatal("expected flat-cap border to paint its endpoint")
	}
	if _, _, _, a := img.At(7, 12).RGBA(); a != 0 {
		t.Fatalf("flat-cap table border should not extend before its endpoint, got alpha=%04x", a)
	}
}

func TestTableCellFillDirectNoFillSuppressesStyleFill(t *testing.T) {
	style := tableStyleRegion{HasFill: true, FillColor: color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}}
	if _, ok := tableCellFill(style, tableCell{NoFill: true}); ok {
		t.Fatal("expected direct noFill to suppress table style fill")
	}
}

func TestRenderGraphicFramePaintsGradientTableCellFill(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Gradient Table",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{emuPerInch},
			Rows: []tableRow{{
				Height: emuPerInch,
				Cells: []tableCell{{
					RowSpan: 1,
					HasFill: true,
					FillPaint: backgroundPaint{
						Color:       color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
						HasGradient: true,
						Gradient: gradientPaint{Stops: []gradientStop{
							{Position: 0, Color: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}},
							{Position: 100000, Color: color.RGBA{A: 0xff}},
						}},
					},
				}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported gradient table cell render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	top := img.RGBAAt(48, 4)
	bottom := img.RGBAAt(48, 91)
	if !(top.R > bottom.R && top.G > bottom.G && top.B > bottom.B) {
		t.Fatalf("expected vertical gradient table cell fill, top=%#v bottom=%#v", top, bottom)
	}
}

func TestParseTableModelRecordsUnsupportedVisibleFeatures(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
			<a:tr h="914400"><a:tc gridSpan="2"><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
				<a:tcPr>
					<a:gradFill><a:gsLst><a:gs pos="0"><a:srgbClr val="FFFFFF"/></a:gs><a:gs pos="100000"><a:srgbClr val="000000"/></a:gs></a:gsLst></a:gradFill>
					<a:pattFill prst="pct50"><a:fgClr><a:srgbClr val="FF0000"/></a:fgClr><a:bgClr><a:srgbClr val="FFFFFF"/></a:bgClr></a:pattFill>
					<a:blipFill/>
					<a:effectLst><a:outerShdw blurRad="12700" dist="12700" dir="5400000"><a:srgbClr val="000000"/></a:outerShdw></a:effectLst>
					<a:cell3D prstMaterial="metal"><a:bevel w="12700" h="12700"/></a:cell3D>
					<a:lnB cmpd="thickThin" cap="unsupportedCap"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:bevel/><a:tailEnd type="triangle"/></a:lnB>
			</a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	for _, want := range []string{
		"uses image/group cell fills that were not rendered",
		"uses effects that were not rendered",
		"uses cell 3-D properties that were not rendered",
		"uses border line caps that were not rendered",
		"uses border line joins that were not rendered",
	} {
		if !slices.Contains(table.UnsupportedFeatures, want) {
			t.Fatalf("expected unsupported table feature %q in %+v", want, table.UnsupportedFeatures)
		}
	}
	for _, notWant := range []string{
		"uses compound border lines that were not rendered",
		"uses border line end decorations that were not rendered",
	} {
		if slices.Contains(table.UnsupportedFeatures, notWant) {
			t.Fatalf("supported table feature should not be reported unsupported %q in %+v", notWant, table.UnsupportedFeatures)
		}
	}
	if slices.Contains(table.UnsupportedFeatures, "uses merged cells rendered with simplified layout") {
		t.Fatalf("merged cells are rendered through table span geometry and should not be reported partial: %+v", table.UnsupportedFeatures)
	}
}

func TestRenderGraphicFramePaintsRoundTableBorderJoins(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	roundBorder := tableCellBorder{
		Specified: true,
		HasLine:   true,
		Color:     color.RGBA{R: 0xff, A: 0xff},
		Width:     emuPerInch / 12,
		Cap:       "flat",
		Compound:  "sng",
		Join:      "round",
	}
	element := slideElement{
		Kind:         "graphicFrame",
		Name:         "Round Join Table",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		HasTable:     true,
		Table: tableModel{
			Columns: []int64{1},
			Rows: []tableRow{{
				Height: 1,
				Cells: []tableCell{{
					RowSpan:      1,
					ColSpan:      1,
					BorderTop:    roundBorder,
					BorderLeft:   roundBorder,
					BorderRight:  tableCellBorder{Specified: true, NoLine: true},
					BorderBottom: tableCellBorder{Specified: true, NoLine: true},
				}},
			}},
		},
	}

	unsupported := renderGraphicFrame(&pptx.Package{}, "ppt/slides/slide1.xml", size, img, &element, nil, tableStyleSet{})
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported round-join table render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if alpha := img.RGBAAt(22, 22).A; alpha == 0 {
		t.Fatalf("expected round join to paint diagonal corner coverage, alpha=%d", alpha)
	}
}

func TestParseTableModelDoesNotReportInvisibleBorderLineJoins(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnB><a:noFill/><a:round/></a:lnB>
			</a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if slices.Contains(table.UnsupportedFeatures, "uses border line joins that were not rendered") {
		t.Fatalf("invisible no-fill border join should not be reported partial: %+v", table.UnsupportedFeatures)
	}
}

func TestParseTableModelDoesNotReportRenderedRoundBorderLineJoins(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnB><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:round/></a:lnB>
			</a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if slices.Contains(table.UnsupportedFeatures, "uses border line joins that were not rendered") {
		t.Fatalf("rendered round border join should not be reported partial: %+v", table.UnsupportedFeatures)
	}
}

func TestParseTableModelTreatsDoubleCompoundBorderAsSupported(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr><a:lnB cmpd="dbl"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:lnB></a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if len(table.UnsupportedFeatures) != 0 {
		t.Fatalf("double compound table border should be supported, got %+v", table.UnsupportedFeatures)
	}
	border := table.Rows[0].Cells[0].BorderBottom
	if !border.Specified || !border.HasLine || border.Compound != "dbl" {
		t.Fatalf("expected parsed double compound border, got %+v", border)
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

func TestParseTableModelDoesNotReportRenderedRoundBorderCaps(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:tbl xmlns:a="a">
		<a:tblGrid><a:gridCol w="914400"/></a:tblGrid>
		<a:tr h="914400"><a:tc><a:txBody><a:bodyPr/><a:p><a:r><a:t>Cell</a:t></a:r></a:p></a:txBody>
			<a:tcPr>
				<a:lnB cap="rnd"><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:lnB>
			</a:tcPr>
		</a:tc></a:tr>
	</a:tbl>`))
	if err != nil {
		t.Fatal(err)
	}

	table := parseTableModel(root, defaultThemeColors())
	if slices.Contains(table.UnsupportedFeatures, "uses border line caps that were not rendered") {
		t.Fatalf("round table border caps are rendered and should not be reported partial: %+v", table.UnsupportedFeatures)
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
	</a:tblStyleLst>`), defaultThemeColors(), themeFonts{MinorLatin: "Calibri"}, themeFillStyles{}, themeLineStyles{}, themeEffectStyles{})
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

func TestTableTextParagraphsWithItalicCopiesParagraphs(t *testing.T) {
	paragraphs := []textParagraph{{
		Text: "Header",
		Runs: []textRun{{Text: "Header"}},
	}}

	got := tableTextParagraphsWithItalic(paragraphs, "")
	if len(got) != 1 || !got[0].Italic || len(got[0].Runs) != 1 || !got[0].Runs[0].Italic {
		t.Fatalf("expected italic paragraph copy, got %+v", got)
	}
	if paragraphs[0].Italic || paragraphs[0].Runs[0].Italic {
		t.Fatalf("source paragraphs were mutated: %+v", paragraphs)
	}
}

func TestTableTextParagraphsWithFontFamilySuppliesParagraphDefault(t *testing.T) {
	paragraphs := []textParagraph{{
		Text: "Header",
		Runs: []textRun{
			{Text: "Default"},
			{Text: "Direct", FontFamily: "Aptos"},
		},
	}}

	got := tableTextParagraphsWithFontFamily(paragraphs, "", "Calibri")
	if len(got) != 1 || got[0].FontFamily != "Calibri" {
		t.Fatalf("expected table style font family on paragraph, got %+v", got)
	}
	defaultSegment := runToSegment(got[0].Runs[0], got[0])
	if defaultSegment.FontFamily != "Calibri" {
		t.Fatalf("expected run without direct font to inherit table style font, got %+v", defaultSegment)
	}
	directSegment := runToSegment(got[0].Runs[1], got[0])
	if directSegment.FontFamily != "Aptos" {
		t.Fatalf("expected direct run font to win over table style font, got %+v", directSegment)
	}
	if paragraphs[0].FontFamily != "" {
		t.Fatalf("source paragraphs were mutated: %+v", paragraphs)
	}
}

func TestTableCellTextElementAppliesStyleTextDefaultsToSegments(t *testing.T) {
	style := tableStyleRegion{
		FontFamily:   "Calibri",
		HasItalic:    true,
		Italic:       true,
		HasTextColor: true,
		TextColor:    color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
	}
	cell := tableCell{
		Text: "Header",
		TextParagraphs: []textParagraph{{
			Runs: []textRun{{Text: "Header"}},
		}},
	}

	element := tableCellTextElement(style, cell, false, color.RGBA{})
	if len(element.TextParagraphs) != 1 || len(element.TextParagraphs[0].Runs) != 1 {
		t.Fatalf("expected styled cell paragraph, got %+v", element.TextParagraphs)
	}
	segment := runToSegment(element.TextParagraphs[0].Runs[0], element.TextParagraphs[0])
	if segment.FontFamily != "Calibri" || !segment.Italic || !segment.HasTextColor || segment.TextColor != style.TextColor {
		t.Fatalf("expected table style text defaults in render segment, got %+v", segment)
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

func TestDiagramDrawingElementsResolveSlideThemeColorMapAndFonts(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`),
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"><p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent6"/></p:sldMaster>`),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme2.xml"/>
</Relationships>`),
		"ppt/theme/theme1.xml":      []byte(`<a:theme xmlns:a="a"><a:themeElements><a:clrScheme name="Fallback"><a:accent1><a:srgbClr val="FF0000"/></a:accent1></a:clrScheme><a:fontScheme name="Fallback"><a:minorFont><a:latin typeface="Fallback"/></a:minorFont></a:fontScheme></a:themeElements></a:theme>`),
		"ppt/theme/theme2.xml":      []byte(`<a:theme xmlns:a="a"><a:themeElements><a:clrScheme name="Slide"><a:accent1><a:srgbClr val="0000FF"/></a:accent1><a:accent6><a:srgbClr val="70AD47"/></a:accent6></a:clrScheme><a:fontScheme name="Slide"><a:majorFont><a:latin typeface="Trebuchet MS"/></a:majorFont><a:minorFont><a:latin typeface="Arial"/></a:minorFont></a:fontScheme></a:themeElements></a:theme>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a"><dsp:spTree><dsp:sp><dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"/><a:solidFill><a:schemeClr val="accent1"/></a:solidFill></dsp:spPr><dsp:style><a:fontRef idx="minor"/></dsp:style><dsp:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:t>Diagram</a:t></a:r></a:p></dsp:txBody></dsp:sp></dsp:spTree></dsp:drawing>`),
	}}

	got := diagramDrawingElements(pkg, "ppt/slides/slide1.xml", "ppt/diagrams/drawing1.xml")
	if len(got) != 1 {
		t.Fatalf("expected one diagram element, got %d", len(got))
	}
	if got[0].FillColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff}) {
		t.Fatalf("expected diagram scheme color to resolve through slide master color map, got %+v", got[0].FillColor)
	}
	if got[0].FontFamily != "Arial" {
		t.Fatalf("expected diagram fontRef minor to resolve through slide theme fonts, got %+v", got[0])
	}
}

func TestDiagramDrawingElementsResolveSlideThemeFillAndEffectStyles(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`),
		"ppt/slideLayouts/_rels/slideLayout1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`),
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"><p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent6"/></p:sldMaster>`),
		"ppt/slideMasters/_rels/slideMaster1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme2.xml"/>
</Relationships>`),
		"ppt/theme/theme1.xml": []byte(`<a:theme xmlns:a="a"><a:themeElements><a:clrScheme name="Fallback"><a:accent1><a:srgbClr val="FF0000"/></a:accent1></a:clrScheme></a:themeElements></a:theme>`),
		"ppt/theme/theme2.xml": []byte(`<a:theme xmlns:a="a"><a:themeElements>
  <a:clrScheme name="Slide"><a:accent1><a:srgbClr val="0000FF"/></a:accent1><a:accent6><a:srgbClr val="70AD47"/></a:accent6></a:clrScheme>
  <a:fmtScheme name="Slide">
    <a:fillStyleLst>
      <a:solidFill><a:schemeClr val="accent1"/></a:solidFill>
    </a:fillStyleLst>
    <a:lnStyleLst>
      <a:ln w="38100" cap="rnd"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="dash"/></a:ln>
    </a:lnStyleLst>
    <a:effectStyleLst>
      <a:effectStyle><a:effectLst><a:outerShdw blurRad="40000" dist="20000" dir="5400000"><a:schemeClr val="phClr"><a:alpha val="50000"/></a:schemeClr></a:outerShdw></a:effectLst></a:effectStyle>
    </a:effectStyleLst>
  </a:fmtScheme>
</a:themeElements></a:theme>`),
		"ppt/diagrams/drawing1.xml": []byte(`<dsp:drawing xmlns:dsp="dsp" xmlns:a="a"><dsp:spTree><dsp:sp><dsp:nvSpPr><dsp:cNvPr id="1" name="Diagram Shape"/></dsp:nvSpPr><dsp:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"/></dsp:spPr><dsp:style><a:lnRef idx="1"><a:schemeClr val="accent1"/></a:lnRef><a:fillRef idx="1"><a:schemeClr val="accent1"/></a:fillRef><a:effectRef idx="1"><a:schemeClr val="accent1"/></a:effectRef></dsp:style></dsp:sp></dsp:spTree></dsp:drawing>`),
	}}

	got := diagramDrawingElements(pkg, "ppt/slides/slide1.xml", "ppt/diagrams/drawing1.xml")
	if len(got) != 1 {
		t.Fatalf("expected one diagram element, got %d", len(got))
	}
	if got[0].FillColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff}) {
		t.Fatalf("expected diagram fillRef to resolve through slide theme fill style and color map, got %+v", got[0].FillColor)
	}
	if !got[0].HasLine || got[0].LineWidth != 38100 || got[0].LineDash != "dash" || got[0].LineCap != "rnd" {
		t.Fatalf("expected diagram lnRef to resolve through slide theme line style, got %+v", got[0])
	}
	if got[0].LineColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff}) {
		t.Fatalf("expected diagram lnRef phClr to use mapped slide color, got %+v", got[0].LineColor)
	}
	if !got[0].HasShadow {
		t.Fatalf("expected diagram effectRef to resolve through slide theme effect style, got %+v", got[0])
	}
	if got[0].ShadowColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0x7f}) {
		t.Fatalf("expected diagram effectRef phClr to use mapped slide color, got %+v", got[0].ShadowColor)
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

func TestApplyInheritedTableTextStylesUsesPresentationDefaultForImplicitCellText(t *testing.T) {
	elements := []slideElement{{
		Kind:     "graphicFrame",
		HasTable: true,
		Table: tableModel{Rows: []tableRow{{
			Cells: []tableCell{
				{
					Text: "Implicit",
					TextParagraphs: []textParagraph{{
						Text: "Implicit",
						Runs: []textRun{{Text: "Implicit"}},
					}},
					FontSize: 1200,
				},
				{
					Text:        "Explicit",
					FontSize:    1600,
					HasFontSize: true,
					TextParagraphs: []textParagraph{{
						Text:     "Explicit",
						FontSize: 1600,
						Runs:     []textRun{{Text: "Explicit", FontSize: 1600}},
					}},
				},
			},
		}}},
	}}

	got := applyInheritedTableTextStyles(elements, map[string]textStyle{
		"default": {
			FontSize: 1800,
			ParagraphStyles: map[int]paragraphStyle{
				0: {FontSize: 1800, FontFamily: "+mn-lt", TextAlign: "ctr"},
			},
		},
	})
	implicit := got[0].Table.Rows[0].Cells[0]
	if implicit.FontSize != 1800 || implicit.TextParagraphs[0].FontSize != 1800 || implicit.TextParagraphs[0].FontFamily != "+mn-lt" || implicit.TextAlign != "ctr" {
		t.Fatalf("implicit table cell did not inherit default text style: %+v", implicit)
	}
	explicit := got[0].Table.Rows[0].Cells[1]
	if explicit.FontSize != 1600 || explicit.TextParagraphs[0].FontSize != 1600 || explicit.TextParagraphs[0].Runs[0].FontSize != 1600 {
		t.Fatalf("explicit table cell font size was overwritten: %+v", explicit)
	}
}

func TestApplyThemeFontFamiliesResolvesTableCellParagraphFonts(t *testing.T) {
	elements := []slideElement{{
		Kind:     "graphicFrame",
		HasTable: true,
		Table: tableModel{Rows: []tableRow{{
			Cells: []tableCell{{
				TextParagraphs: []textParagraph{{
					FontFamily:       "+mn-lt",
					BulletFontFamily: "+mj-lt",
					Runs:             []textRun{{Text: "A", FontFamily: "+mn-lt"}},
				}},
			}},
		}}},
	}}

	got := applyThemeFontFamilies(elements, themeFonts{MajorLatin: "Calibri Light", MinorLatin: "Calibri"})
	paragraph := got[0].Table.Rows[0].Cells[0].TextParagraphs[0]
	if paragraph.FontFamily != "Calibri" || paragraph.BulletFontFamily != "Calibri Light" || paragraph.Runs[0].FontFamily != "Calibri" {
		t.Fatalf("table cell theme fonts were not resolved: %+v", paragraph)
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

func TestWrapTextWithPrefixesBreaksAfterAuthoredHyphen(t *testing.T) {
	face, err := openFontFace(1800, false, false, 0, "Arial")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	width := maxInt(measureString(face, "alpha treatment-"), measureString(face, "adjusted prevalence")) + 2
	lines := wrapTextWithPrefixes(face, "alpha treatment-adjusted prevalence", width, "", "")
	if len(lines) != 2 || lines[0] != "alpha treatment-" || lines[1] != "adjusted prevalence" {
		t.Fatalf("expected authored hyphen to provide a wrap point, got %+v", lines)
	}
}

func TestWrapTextWithPrefixesBreaksAfterAuthoredSlash(t *testing.T) {
	face, err := openFontFace(1600, false, false, 0, "Arial")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	width := maxInt(measureString(face, "Emission Rate PM2.5 (g/"), measureString(face, "hr)")) + 2
	lines := wrapTextWithPrefixes(face, "Emission Rate PM2.5 (g/hr)", width, "", "")
	if len(lines) != 2 || lines[0] != "Emission Rate PM2.5 (g/" || lines[1] != "hr)" {
		t.Fatalf("expected authored slash to provide a wrap point, got %+v", lines)
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

func TestTextRenderLinesForElementAppliesElementTextColorToStyledRuns(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(3600, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(3600, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize:     3600,
		HasTextColor: true,
		TextColor:    color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		TextParagraphs: []textParagraph{{
			FontSize: 3600,
			Runs: []textRun{
				{Text: "Defaulted", FontSize: 3600},
				{Text: " Direct", FontSize: 3600, HasTextColor: true, TextColor: color.RGBA{R: 0xff, A: 0xff}},
			},
		}},
	}, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) != 2 {
		t.Fatalf("expected two styled run segments, got %+v", lines)
	}
	defaultSegment := lines[0].Segments[0]
	if !defaultSegment.HasTextColor || defaultSegment.TextColor != (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
		t.Fatalf("expected uncolored run to inherit element text color, got %+v", defaultSegment)
	}
	directSegment := lines[0].Segments[1]
	if !directSegment.HasTextColor || directSegment.TextColor != (color.RGBA{R: 0xff, A: 0xff}) {
		t.Fatalf("expected explicit run color to win over element text color, got %+v", directSegment)
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

	wingdingsSquare := appendPrefixSegment("▪ ", textParagraph{Bullet: "▪", BulletFontFamily: "Wingdings", FontSize: 1800}, nil)
	if len(wingdingsSquare) < 1 || wingdingsSquare[0].FontFamily != "" {
		t.Fatalf("unexpected Wingdings square bullet font family, got %+v", wingdingsSquare)
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

func TestTextRenderLinesForElementUsesHangingBulletTabStop(t *testing.T) {
	faces := newFontFaceCache(false, "")
	defer faces.Close()
	face, err := faces.Get(2000, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(2000, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{
		FontSize: 2000,
		TextParagraphs: []textParagraph{{
			Text:          "Identify the products",
			FontSize:      2000,
			Bullet:        "1.",
			HasAutoNumber: true,
			HasMarginLeft: true,
			MarginLeft:    emuPerInch / 2,
			HasIndent:     true,
			Indent:        -emuPerInch / 2,
			Runs:          []textRun{{Text: "Identify the products", FontSize: 2000}},
		}},
	}, 300)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 1 || len(lines[0].Segments) < 3 {
		t.Fatalf("expected one segmented numbered line, got %+v", lines)
	}
	if lines[0].XOffset != 0 || !lines[0].HasXOffset {
		t.Fatalf("expected bullet to start at first-line offset, got %+v", lines[0])
	}
	if lines[0].Segments[0].Text != "1." || lines[0].Segments[1].Text != "\t" {
		t.Fatalf("expected numbered bullet followed by a tab spacer, got %+v", lines[0].Segments[:2])
	}
	if len(lines[0].TabStops) == 0 || lines[0].TabStops[0] != 36 {
		t.Fatalf("expected text margin tab stop at half inch, got %+v", lines[0].TabStops)
	}
	prefixWidth, err := measureStyledSegmentsAtDPI(faces, face, boldFace, lines[0].Segments[:2], defaultOutputDPI, lines[0].TabStops)
	if err != nil {
		t.Fatal(err)
	}
	if prefixWidth != 36 {
		t.Fatalf("expected bullet prefix to advance to hanging text margin, got %d", prefixWidth)
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

func TestStyledWordTokensExposeHyphenWrapPoint(t *testing.T) {
	tokens := styledWordTokens([]textRun{{Text: "treatment-adjusted prevalence"}}, textParagraph{FontSize: 1800})
	if len(tokens) != 3 {
		t.Fatalf("unexpected hyphen token count: %+v", tokens)
	}
	if tokens[0].segmentWithPrefix(false).Text != "treatment-" || tokens[1].segmentWithPrefix(true).Text != "adjusted" || tokens[2].segmentWithPrefix(true).Text != " prevalence" {
		t.Fatalf("expected authored hyphen to become a wrap point without adding spaces, got %+v", tokens)
	}
}

func TestStyledWordTokensExposeSlashWrapPoint(t *testing.T) {
	tokens := styledWordTokens([]textRun{{Text: "PM2.5 (g/hr)"}}, textParagraph{FontSize: 1600})
	if len(tokens) != 3 {
		t.Fatalf("unexpected slash token count: %+v", tokens)
	}
	if tokens[0].segmentWithPrefix(false).Text != "PM2.5" || tokens[1].segmentWithPrefix(true).Text != " (g/" || tokens[2].segmentWithPrefix(true).Text != "hr)" {
		t.Fatalf("expected authored slash to become a wrap point without adding spaces, got %+v", tokens)
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

func TestCalibriFontCandidatesIncludeMicrosoftOfficeCloudFontCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	localAppData := filepath.Join(t.TempDir(), "LocalAppData")
	t.Setenv("LOCALAPPDATA", localAppData)

	candidates := exactFontCandidatesForFamily("Calibri", false, false)
	expected := []string{
		filepath.Join(home, "Library", "Group Containers", "UBF8T346G9.Office", "FontCache", "*", "CloudFonts", "Calibri.ttf"),
		filepath.Join(home, "Library", "Group Containers", "UBF8T346G9.Office", "FontCache", "*", "CloudFonts", "Calibri", "Calibri.ttf"),
		filepath.Join(localAppData, "Microsoft", "FontCache", "*", "CloudFonts", "Calibri.ttf"),
		filepath.Join(localAppData, "Microsoft", "FontCache", "*", "CloudFonts", "Calibri", "Calibri.ttf"),
	}
	for _, candidate := range expected {
		if !slices.Contains(candidates, candidate) {
			t.Fatalf("expected Microsoft Office cloud font cache candidate %q in %+v", candidate, candidates)
		}
	}
}

func TestCalibriFontCandidatesIncludeOfficeFileNames(t *testing.T) {
	tests := []struct {
		family string
		bold   bool
		italic bool
		name   string
	}{
		{family: "Calibri", name: "Calibri.ttf"},
		{family: "Calibri", name: "calibri.ttf"},
		{family: "Calibri", bold: true, name: "Calibrib.ttf"},
		{family: "Calibri", bold: true, name: "calibrib.ttf"},
		{family: "Calibri", italic: true, name: "Calibrii.ttf"},
		{family: "Calibri", italic: true, name: "calibrii.ttf"},
		{family: "Calibri", bold: true, italic: true, name: "Calibriz.ttf"},
		{family: "Calibri", bold: true, italic: true, name: "calibriz.ttf"},
		{family: "Calibri Light", name: "calibril.ttf"},
		{family: "Calibri Light", italic: true, name: "calibrili.ttf"},
	}
	for _, tt := range tests {
		candidates := exactFontCandidatesForFamily(tt.family, tt.bold, tt.italic)
		if !slices.Contains(candidates, filepath.Join("/Library/Fonts", tt.name)) {
			t.Fatalf("expected %s candidates to include Office filename %q, got %+v", tt.family, tt.name, candidates)
		}
	}
}

func TestCalibriOfficeCacheGlobResolvesCapitalizedStyleFileNames(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fontRoot := filepath.Join(home, ".cache", "puppt", "fonts", "microsoft-word", "expanded", "Microsoft_Word.pkg", "Payload", "Microsoft Word.app", "Contents", "Resources", "DFonts")
	if err := os.MkdirAll(fontRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(fontRoot, "Calibrib.ttf")
	if err := os.WriteFile(want, []byte("font"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := firstExistingPath(exactFontCandidatesForFamily("Calibri", true, false))
	if got != want {
		t.Fatalf("expected capitalized Office Calibri bold filename %q to resolve through cache glob, got %q", want, got)
	}
}

func TestCalibriFontCandidatesIncludeMicrosoftAndLinuxFontRoots(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	candidates := exactFontCandidatesForFamily("Calibri", false, false)
	expected := []string{
		filepath.Join("/Library/Fonts/Microsoft", "calibri.ttf"),
		filepath.Join(home, "Library", "Fonts", "Microsoft", "calibri.ttf"),
		filepath.Join("/Applications", "Microsoft Word.app", "Contents", "Resources", "DFonts", "Calibri.ttf"),
		filepath.Join("/Applications", "Microsoft Excel.app", "Contents", "Resources", "DFonts", "Calibri.ttf"),
		filepath.Join(home, "Applications", "Microsoft Word.app", "Contents", "Resources", "DFonts", "Calibri.ttf"),
		filepath.Join(home, ".cache", "puppt", "fonts", "*", "expanded", "*", "Payload", "Microsoft Word.app", "Contents", "Resources", "DFonts", "Calibri.ttf"),
		filepath.Join(home, ".cache", "puppt", "fonts", "*", "expanded", "*", "Payload", "Microsoft Excel.app", "Contents", "Resources", "DFonts", "Calibri.ttf"),
		filepath.Join("/usr/local/share/fonts", "calibri.ttf"),
		filepath.Join("/usr/share/fonts", "calibri.ttf"),
		filepath.Join("/usr/share/fonts/truetype/msttcorefonts", "calibri.ttf"),
		filepath.Join(home, ".local", "share", "fonts", "calibri.ttf"),
	}
	for _, path := range expected {
		if !slices.Contains(candidates, path) {
			t.Fatalf("expected Calibri candidates to include %q, got %+v", path, candidates)
		}
	}
}

func TestFirstExistingPathExpandsSortedGlobCandidates(t *testing.T) {
	dir := t.TempDir()
	latePath := filepath.Join(dir, "B.ttf")
	earlyPath := filepath.Join(dir, "A.ttf")
	if err := os.WriteFile(latePath, []byte("late"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(earlyPath, []byte("early"), 0o600); err != nil {
		t.Fatal(err)
	}

	got := firstExistingPath([]string{filepath.Join(dir, "*.ttf")})
	if got != earlyPath {
		t.Fatalf("expected deterministic first glob match %q, got %q", earlyPath, got)
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
		if !strings.Contains(message, "metric-compatible substitute font") {
			t.Fatalf("expected supported Calibri substitute to be reported as partial, got %q", message)
		}
	}
	if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Symbol"}); message != "" {
		t.Fatalf("Segoe UI Symbol should use a supported Office font substitute when exact font is absent, got %q", message)
	}
	if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Historic"}); message != "" {
		t.Fatalf("Segoe UI Historic should use a supported Office font substitute when exact font is absent, got %q", message)
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
	if !exactFontFamilyAvailable("Calibri Light") {
		messages = fontResolutionUnsupportedMessages(slideElement{
			FontFamily: "Calibri Light",
			TextParagraphs: []textParagraph{{
				FontFamily: "Calibri Light",
				Runs:       []textRun{{Text: "A", FontFamily: "Calibri Light"}},
			}},
		})
		if len(messages) != 1 || !strings.Contains(messages[0], "metric-compatible substitute font") {
			t.Fatalf("expected duplicate inherited Calibri Light reports to collapse, got %+v", messages)
		}
	}
}

func TestCalibriFontSubstitutesAreResolvedAndReportedAsPartial(t *testing.T) {
	for _, family := range []string{"Calibri", "Calibri Light"} {
		source, ok := substituteFontSourceForFamily(family, false, false)
		if !ok {
			t.Fatalf("expected supported substitute for %s", family)
		}
		if !strings.Contains(source.Label, "Carlito") {
			t.Fatalf("expected Carlito source for %s, got %q", family, source.Label)
		}
		if !exactFontFamilyAvailable(family) {
			message := fontResolutionUnsupportedMessage(slideElement{FontFamily: family})
			if !strings.Contains(message, "metric-compatible substitute font") {
				t.Fatalf("supported substitute for %s should be reported as partial, got %q", family, message)
			}
		}
	}
}

func TestSymbolFontSubstitutesAreResolvedButNotUnsupported(t *testing.T) {
	if !exactFontFamilyAvailable("Segoe UI Symbol") {
		if source, ok := substituteFontSourceForFamily("Segoe UI Symbol", false, false); !ok || source.Label == "" {
			t.Fatalf("Segoe UI Symbol should use a supported sans-serif substitute, got %q ok=%v", source.Label, ok)
		}
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Symbol"}); message != "" {
			t.Fatalf("supported Segoe UI Symbol substitute should not be reported, got %q", message)
		}
	}
	if !exactFontFamilyAvailable("Segoe UI Historic") {
		if source, ok := substituteFontSourceForFamily("Segoe UI Historic", false, false); !ok || source.Label == "" {
			t.Fatalf("Segoe UI Historic should use a supported sans-serif substitute, got %q ok=%v", source.Label, ok)
		}
		if message := fontResolutionUnsupportedMessage(slideElement{FontFamily: "Segoe UI Historic"}); message != "" {
			t.Fatalf("supported Segoe UI Historic substitute should not be reported, got %q", message)
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

func TestElementShouldReportFontResolutionSkipsTinyImagePlaceholderMarkerText(t *testing.T) {
	element := slideElement{
		Name:    "Picture Placeholder 8",
		Text:    ".",
		EmbedID: "rId1",
		TextParagraphs: []textParagraph{{
			Text:       ".",
			FontFamily: "Missing Office Font",
			FontSize:   100,
			Runs: []textRun{{
				Text:       ".",
				FontFamily: "Missing Office Font",
				FontSize:   100,
			}},
		}},
	}
	if messages := fontResolutionUnsupportedMessages(element); len(messages) == 0 {
		t.Fatal("test fixture should use a font that would normally report fallback")
	}
	if elementShouldRenderText(element) {
		t.Fatalf("tiny image-placeholder marker text should not render: %+v", element)
	}
	if elementShouldReportFontResolution(element) {
		t.Fatalf("tiny image-placeholder marker text should not report font resolution: %+v", element)
	}
}

func TestElementShouldReportFontResolutionKeepsVisibleImageShapeText(t *testing.T) {
	element := slideElement{
		Name:    "Picture Placeholder 8",
		Text:    "Visible caption",
		EmbedID: "rId1",
		TextParagraphs: []textParagraph{{
			Text:       "Visible caption",
			FontFamily: "Missing Office Font",
			FontSize:   1200,
			Runs: []textRun{{
				Text:       "Visible caption",
				FontFamily: "Missing Office Font",
				FontSize:   1200,
			}},
		}},
	}
	if !elementShouldReportFontResolution(element) {
		t.Fatalf("visible image-shape text should still report font resolution: %+v", element)
	}
	if !elementShouldRenderText(element) {
		t.Fatalf("visible image-shape text should still render: %+v", element)
	}
}

func TestRenderShapeSkipsTinyImagePlaceholderMarkerText(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "pic",
		Name:         "Picture Placeholder 8",
		Text:         ".",
		EmbedID:      "rId1",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		TextParagraphs: []textParagraph{{
			Text:       ".",
			FontFamily: "Missing Office Font",
			FontSize:   100,
			Runs: []textRun{{
				Text:       ".",
				FontFamily: "Missing Office Font",
				FontSize:   100,
			}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element)
	if len(unsupported) != 0 {
		t.Fatalf("hidden image-placeholder marker text should not report unsupported content, got %+v", unsupported)
	}
	if hasOpaquePixel(img) {
		t.Fatal("hidden image-placeholder marker text should not paint pixels")
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

func TestAnchorCenteredTextBoundsCentersNarrowTextBox(t *testing.T) {
	got := anchorCenteredTextBounds(image.Rect(10, 20, 210, 80), 60)
	want := image.Rect(80, 20, 140, 80)
	if got != want {
		t.Fatalf("unexpected anchor-centered bounds: got=%v want=%v", got, want)
	}
	if got := anchorCenteredTextBounds(image.Rect(10, 20, 210, 80), 220); got != image.Rect(10, 20, 210, 80) {
		t.Fatalf("text wider than the body bounds should keep original bounds, got %v", got)
	}
}

func TestDrawShapeTextHonorsAnchorCenter(t *testing.T) {
	left := image.NewRGBA(image.Rect(0, 0, 220, 90))
	centered := image.NewRGBA(image.Rect(0, 0, 220, 90))
	element := slideElement{
		FontFamily: "Carlito",
		FontSize:   2400,
		TextParagraphs: []textParagraph{{
			Runs: []textRun{{Text: "Hi", FontSize: 2400}},
		}},
	}
	if err := drawShapeTextWithDPI(left, left.Bounds(), element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	element.TextAnchorCenter = true
	if err := drawShapeTextWithDPI(centered, centered.Bounds(), element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	leftBounds := opaqueBounds(left)
	centeredBounds := opaqueBounds(centered)
	if leftBounds.Empty() || centeredBounds.Empty() {
		t.Fatalf("expected text pixels, got left=%v centered=%v", leftBounds, centeredBounds)
	}
	if centeredBounds.Min.X <= leftBounds.Min.X+70 {
		t.Fatalf("expected anchorCtr text bounds to move toward the horizontal center, got left=%v centered=%v", leftBounds, centeredBounds)
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

func TestRasterizePathMaskAntialiasesCustomGeometryEdges(t *testing.T) {
	mask := rasterizePathMask(image.Rect(0, 0, 16, 16), []pathPoint{
		{X: 0, Y: 0},
		{X: 1, Y: 0},
		{X: 0, Y: 1},
	})
	hasOpaque := false
	hasPartial := false
	for y := mask.Bounds().Min.Y; y < mask.Bounds().Max.Y; y++ {
		for x := mask.Bounds().Min.X; x < mask.Bounds().Max.X; x++ {
			alpha := mask.AlphaAt(x, y).A
			if alpha == 255 {
				hasOpaque = true
			}
			if alpha > 0 && alpha < 255 {
				hasPartial = true
			}
		}
	}
	if !hasOpaque || !hasPartial {
		t.Fatalf("expected vector mask to include opaque and antialiased pixels, opaque=%v partial=%v", hasOpaque, hasPartial)
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

func TestSoftEdgeRadiusUsesDrawingMLRadius(t *testing.T) {
	element := slideElement{SoftEdgeRadius: emuPerInch / 10}
	got := softEdgeRadiusPixels(element, slideSize{CX: emuPerInch, CY: emuPerInch}, 100)
	if got != 10 {
		t.Fatalf("expected DrawingML soft-edge radius to scale directly to output pixels, got %d", got)
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

func TestPictureScalerUsesHighQualityReconstructionForJPEGSources(t *testing.T) {
	src := image.NewYCbCr(image.Rect(0, 0, 4, 4), image.YCbCrSubsampleRatio444)
	if got := pictureScaler(src, src.Bounds()); got != xdraw.CatmullRom {
		t.Fatalf("YCbCr JPEG sources should use high-quality reconstruction, got %T", got)
	}

	if got := pictureScaler(image.NewRGBA(src.Bounds()), src.Bounds()); got != xdraw.ApproxBiLinear {
		t.Fatalf("non-JPEG raster sources should keep the default scaler, got %T", got)
	}

	if got := pictureScaler(src, image.Rect(-1, 0, 4, 4)); got != xdraw.ApproxBiLinear {
		t.Fatalf("virtual crop padding should keep transparent-source behavior, got %T", got)
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

func TestPictureSourceForElementPreservesBlackWhiteModeForColorRendering(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 100, G: 150, B: 200, A: 220})

	got, bounds := pictureSourceForElement(src, slideElement{BWMode: "gray"})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed source bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel != (color.RGBA{R: 100, G: 150, B: 200, A: 220}) {
		t.Fatalf("normal color rendering should preserve authored image color despite bwMode, got %#v", pixel)
	}
}

func TestPictureSourceForElementPreservesConcreteBlackWhiteModesForColorRendering(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 100, G: 150, B: 200, A: 220})

	for _, mode := range []string{"black", "white", "hidden", "blackWhite"} {
		got, bounds := pictureSourceForElement(src, slideElement{BWMode: mode})
		if bounds != image.Rect(0, 0, 1, 1) {
			t.Fatalf("%s: unexpected transformed source bounds: %v", mode, bounds)
		}
		if pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA); pixel != (color.RGBA{R: 100, G: 150, B: 200, A: 220}) {
			t.Fatalf("%s: normal color rendering should preserve authored image color, got %#v", mode, pixel)
		}
	}
}

func TestRenderShapePreservesColorWithBlackWhiteModeForColorRendering(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Gray Shape",
		BWMode:       "gray",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		PrstGeom:     "rect",
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
		HasLine:      true,
		LineColor:    color.RGBA{B: 255, A: 255},
		LineWidth:    emuPerInch / 12,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected gray shape render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(48, 48); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("normal color rendering should preserve authored fill color despite bwMode, got %#v", got)
	}
	if got := img.RGBAAt(0, 48); got.B != 255 || got.A == 0 {
		t.Fatalf("normal color rendering should preserve authored outline color despite bwMode, got %#v", got)
	}
}

func TestRenderShapeDoesNotReportBlackWhiteModeForColorRendering(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Black White Shape",
		BWMode:       "blackWhite",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
		PrstGeom:     "rect",
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("normal color rendering should not report bwMode as unsupported, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
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

func TestParseBlipEffectsReadsDefaultAlphaModFixAmount(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:blip xmlns:a="a"><a:alphaModFix/></a:blip>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBlipEffects(root, &element)
	if !element.HasImageAlphaModFix || element.ImageAlphaModFixPct != 100000 {
		t.Fatalf("alphaModFix without amt should parse the DrawingML default opacity, got %+v", element)
	}
	if shouldApplyImageAlphaModFix(element) {
		t.Fatalf("default alphaModFix should not alter rendered opacity, got %+v", element)
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

func TestDecodeSVGImageResolvesClassAndInlineFillStyles(t *testing.T) {
	source, err := decodeImage("ppt/media/icon.svg", "image/svg+xml", []byte(`<svg viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
		<style>
			.IconFill { fill: #0070C0; fill-opacity: 0.5; }
			.HiddenFill { fill: none; }
		</style>
		<rect x="2" y="2" width="6" height="6" class="IconFill" fill="#ff0000"/>
		<rect x="10" y="2" width="6" height="6" class="HiddenFill" fill="#00ff00"/>
		<rect x="2" y="10" width="6" height="6" class="IconFill" style="fill:#ff0000;fill-opacity:1"/>
	</svg>`))
	if err != nil {
		t.Fatalf("decode svg failed: %v", err)
	}
	r, g, b, a := source.At(3, 3).RGBA()
	if r != 0 || g != 0x7070 || b != 0xc0c0 || a != 0x8080 {
		t.Fatalf("expected class fill to override presentation fill, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
	}
	_, _, _, a = source.At(11, 3).RGBA()
	if a != 0 {
		t.Fatalf("expected class fill:none to suppress presentation fill, got alpha=%04x", a)
	}
	r, g, b, a = source.At(3, 11).RGBA()
	if r != 0xffff || g != 0 || b != 0 || a != 0xffff {
		t.Fatalf("expected inline style fill to override class fill, got rgba=%04x,%04x,%04x,%04x", r, g, b, a)
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

func TestJPEGICCProfileReassemblesOutOfOrderAPP2Chunks(t *testing.T) {
	profile := []byte("split-profile-data")
	data := []byte{0xff, 0xd8}
	data = append(data, testJPEGAPP2ICCChunk(2, 3, profile[5:12])...)
	data = append(data, testJPEGAPP2ICCChunk(1, 3, profile[:5])...)
	data = append(data, testJPEGAPP2ICCChunk(3, 3, profile[12:])...)
	data = append(data, 0xff, 0xd9)

	got, ok := jpegICCProfile(data)
	if !ok || string(got) != string(profile) {
		t.Fatalf("expected reassembled JPEG ICC profile, got %q ok=%v", got, ok)
	}
}

func TestJPEGICCProfileRejectsMissingOrDuplicateChunks(t *testing.T) {
	missing := []byte{0xff, 0xd8}
	missing = append(missing, testJPEGAPP2ICCChunk(1, 2, []byte("first"))...)
	missing = append(missing, 0xff, 0xd9)
	if _, ok := jpegICCProfile(missing); ok {
		t.Fatal("expected missing JPEG ICC chunk to be rejected")
	}

	duplicate := []byte{0xff, 0xd8}
	duplicate = append(duplicate, testJPEGAPP2ICCChunk(1, 2, []byte("first"))...)
	duplicate = append(duplicate, testJPEGAPP2ICCChunk(1, 2, []byte("again"))...)
	duplicate = append(duplicate, 0xff, 0xd9)
	if _, ok := jpegICCProfile(duplicate); ok {
		t.Fatal("expected duplicate JPEG ICC chunk to be rejected")
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

func TestInheritedPlaceholderSourcesMergeLayoutWithoutTransform(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:cSld><p:spTree>
    <p:sp>
      <p:nvSpPr><p:cNvPr id="1" name="Master Body"/><p:nvPr><p:ph type="body" idx="1"/></p:nvPr></p:nvSpPr>
      <p:spPr><a:xfrm><a:off x="10" y="20"/><a:ext cx="300" cy="400"/></a:xfrm></p:spPr>
      <p:txBody><a:bodyPr anchor="t"/><a:lstStyle><a:lvl1pPr><a:defRPr sz="1800"/></a:lvl1pPr></a:lstStyle></p:txBody>
    </p:sp>
  </p:spTree></p:cSld>
</p:sldMaster>`),
		"ppt/slideLayouts/slideLayout1.xml": []byte(`<p:sldLayout xmlns:p="p" xmlns:a="a">
  <p:cSld><p:spTree>
    <p:sp>
      <p:nvSpPr><p:cNvPr id="2" name="Layout Body"/><p:nvPr><p:ph type="body" idx="1"/></p:nvPr></p:nvSpPr>
      <p:spPr/>
      <p:txBody><a:bodyPr anchor="b"/><a:lstStyle><a:lvl1pPr><a:defRPr sz="2400"/></a:lvl1pPr></a:lstStyle></p:txBody>
    </p:sp>
  </p:spTree></p:cSld>
</p:sldLayout>`),
	}}

	sources := inheritedPlaceholderSourcesWithThemeResolver(pkg, []string{
		"ppt/slideMasters/slideMaster1.xml",
		"ppt/slideLayouts/slideLayout1.xml",
		"ppt/slides/slide1.xml",
	}, "ppt/slides/slide1.xml", func(string) themeColors { return defaultThemeColors() })
	got, ok := sources["type:body"]
	if !ok {
		t.Fatalf("expected body placeholder source, got %+v", sources)
	}
	if !got.HasTransform || got.OffX != 10 || got.ExtCX != 300 {
		t.Fatalf("layout placeholder without transform should retain master geometry, got %+v", got)
	}
	if got.TextAnchor != "b" {
		t.Fatalf("layout placeholder body properties should override master text anchor, got %+v", got)
	}
	if got.PlaceholderParagraphStyles[0].FontSize != 2400 {
		t.Fatalf("layout placeholder paragraph style should override master style, got %+v", got.PlaceholderParagraphStyles)
	}
}

func TestResolveSlidePlaceholdersInheritsVisualProperties(t *testing.T) {
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
			HasFill:         true,
			FillColor:       color.RGBA{R: 10, G: 20, B: 30, A: 255},
			HasLine:         true,
			LineColor:       color.RGBA{R: 40, G: 50, B: 60, A: 255},
			HasLineWidth:    true,
			LineWidth:       19050,
			HasShadow:       true,
			ShadowColor:     color.RGBA{A: 80},
			ShadowBlur:      12700,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if !got[0].HasFill || got[0].FillColor.R != 10 {
		t.Fatalf("placeholder fill was not inherited: %+v", got[0])
	}
	if !got[0].HasLine || got[0].LineColor.R != 40 || got[0].LineWidth != 19050 {
		t.Fatalf("placeholder line was not inherited: %+v", got[0])
	}
	if !got[0].HasShadow || got[0].ShadowBlur != 12700 {
		t.Fatalf("placeholder effects were not inherited: %+v", got[0])
	}
}

func TestResolveSlidePlaceholdersKeepsLocalNoFillAndNoLine(t *testing.T) {
	elements := []slideElement{{
		Kind:            "sp",
		Name:            "Title 1",
		Text:            "Slide title",
		IsPlaceholder:   true,
		PlaceholderType: "title",
		NoFill:          true,
		NoLine:          true,
	}}
	sources := map[string]slideElement{
		"type:title": {
			IsPlaceholder:   true,
			PlaceholderType: "title",
			HasFill:         true,
			FillColor:       color.RGBA{R: 10, A: 255},
			HasLine:         true,
			LineColor:       color.RGBA{R: 40, A: 255},
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].HasFill || !got[0].NoFill {
		t.Fatalf("local noFill should block inherited fill: %+v", got[0])
	}
	if got[0].HasLine || !got[0].NoLine {
		t.Fatalf("local noLine should block inherited line: %+v", got[0])
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

func TestResolveSlidePlaceholdersInheritsUnspecifiedBodyTextProperties(t *testing.T) {
	elements := []slideElement{{
		Kind:                    "sp",
		Name:                    "Content Placeholder 1",
		Text:                    "Slide body",
		IsPlaceholder:           true,
		PlaceholderType:         "body",
		HasBodyProperties:       true,
		HasTextWrap:             true,
		TextWrap:                "square",
		HasTextVerticalOverflow: true,
		TextVerticalOverflow:    "clip",
	}}
	sources := map[string]slideElement{
		"type:body": {
			IsPlaceholder:             true,
			PlaceholderType:           "body",
			HasTextWrap:               true,
			TextWrap:                  "none",
			HasTextHorizontalOverflow: true,
			TextHorizontalOverflow:    "overflow",
			HasTextVerticalOverflow:   true,
			TextVerticalOverflow:      "overflow",
			HasTextVertical:           true,
			TextVertical:              "eaVert",
			HasTextBodyRotation:       true,
			TextBodyRotation:          5400000,
			HasTextColumns:            true,
			TextColumnCount:           2,
			HasTextRightToLeftColumns: true,
			TextRightToLeftColumns:    true,
			HasTextAnchorCenter:       true,
			TextAnchorCenter:          true,
			HasFirstLastSpacing:       true,
			IncludeFirstLastSpacing:   true,
		},
	}

	got := resolveSlidePlaceholders(elements, sources)
	if got[0].TextWrap != "square" {
		t.Fatalf("local body wrap should block inherited wrap: %+v", got[0])
	}
	if got[0].TextVerticalOverflow != "clip" || !got[0].HasTextHorizontalOverflow || got[0].TextHorizontalOverflow != "overflow" {
		t.Fatalf("local vertical overflow should block inherited value while missing horizontal overflow inherits: %+v", got[0])
	}
	if got[0].TextVertical != "eaVert" || !got[0].HasTextBodyRotation || got[0].TextBodyRotation != 5400000 || !got[0].HasTextColumns || got[0].TextColumnCount != 2 || !got[0].HasTextRightToLeftColumns || !got[0].TextRightToLeftColumns || !got[0].TextAnchorCenter {
		t.Fatalf("unspecified body text properties were not inherited: %+v", got[0])
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

func TestMergePlaceholderSourceKeepsMasterParagraphFontSizeWhenLayoutDefRPrOmitsSize(t *testing.T) {
	masterStyleNode, err := parseXMLNode([]byte(`<a:lvl1pPr xmlns:a="a">
  <a:lnSpc><a:spcPct val="90000"/></a:lnSpc>
  <a:buSzPts val="2800"/>
  <a:buFont typeface="Arial"/>
  <a:buChar char="•"/>
  <a:defRPr sz="2800">
    <a:latin typeface="Calibri"/>
  </a:defRPr>
</a:lvl1pPr>`))
	if err != nil {
		t.Fatal(err)
	}
	layoutStyleNode, err := parseXMLNode([]byte(`<a:lvl1pPr xmlns:a="a" marL="457200" indent="-342900">
  <a:lnSpc><a:spcPct val="90000"/></a:lnSpc>
  <a:buSzPts val="1800"/>
  <a:buChar char="•"/>
  <a:defRPr/>
</a:lvl1pPr>`))
	if err != nil {
		t.Fatal(err)
	}
	master := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "body",
		PlaceholderParagraphStyles: map[int]paragraphStyle{
			0: parseParagraphStyle(masterStyleNode, defaultThemeColors()),
		},
	}
	layout := slideElement{
		IsPlaceholder:   true,
		PlaceholderType: "body",
		PlaceholderParagraphStyles: map[int]paragraphStyle{
			0: parseParagraphStyle(layoutStyleNode, defaultThemeColors()),
		},
	}

	got := mergePlaceholderSource(master, layout)
	style := got.PlaceholderParagraphStyles[0]
	if style.FontSize != 2800 {
		t.Fatalf("layout defRPr without sz should not erase master placeholder font size: %+v", style)
	}
	if style.FontFamily != "Calibri" {
		t.Fatalf("layout defRPr without latin should not erase master placeholder font family: %+v", style)
	}
	if !style.HasMarginLeft || style.MarginLeft != 457200 || !style.HasIndent || style.Indent != -342900 {
		t.Fatalf("layout paragraph geometry should still override master geometry: %+v", style)
	}
	if style.BulletFontSize != 1800 {
		t.Fatalf("layout bullet size should override master bullet size independently of text font size: %+v", style)
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

func TestParseSlideElementMergesLocalLinePropertiesWithStyleLineRef(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="7" name="Dashed Rectangle"/></p:nvSpPr>
	  <p:spPr>
	    <a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="1"/></a:xfrm>
	    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
	    <a:ln w="22225" cap="flat" algn="ctr"><a:prstDash val="sysDot"/></a:ln>
	  </p:spPr>
	  <p:style><a:lnRef idx="1"><a:schemeClr val="accent1"/></a:lnRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	lineStyles := parseThemeLineStyles([]byte(`<a:theme xmlns:a="a"><a:themeElements><a:fmtScheme><a:lnStyleLst>
	  <a:ln w="38100" cap="rnd"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="dash"/></a:ln>
	</a:lnStyleLst></a:fmtScheme></a:themeElements></a:theme>`))
	theme := themeColors{"accent1": {R: 0, G: 0x70, B: 0xC0, A: 255}}

	got := parseSlideElementNodeWithThemeEffectsAndFills(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, themeEffectStyles{}, themeFillStyles{}, lineStyles)
	if !got.HasLine || got.LineColor != theme["accent1"] {
		t.Fatalf("expected style lineRef to provide line color, got %+v", got)
	}
	if got.LineWidth != 22225 || got.LineDash != "sysDot" || got.LineCap != "flat" || got.LineAlign != "ctr" {
		t.Fatalf("expected local line width/dash/cap/align to override style line, got %+v", got)
	}
}

func TestParseSlideElementLocalSolidDashSuppressesStyleDash(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="7" name="Solid Rectangle"/></p:nvSpPr>
	  <p:spPr>
	    <a:xfrm><a:off x="0" y="0"/><a:ext cx="1" cy="1"/></a:xfrm>
	    <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
	    <a:ln><a:prstDash val="solid"/></a:ln>
	  </p:spPr>
	  <p:style><a:lnRef idx="1"><a:schemeClr val="accent1"/></a:lnRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	lineStyles := parseThemeLineStyles([]byte(`<a:theme xmlns:a="a"><a:themeElements><a:fmtScheme><a:lnStyleLst>
	  <a:ln w="38100"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill><a:prstDash val="dash"/></a:ln>
	</a:lnStyleLst></a:fmtScheme></a:themeElements></a:theme>`))

	got := parseSlideElementNodeWithThemeEffectsAndFills(root, renderTransform{ScaleX: 1, ScaleY: 1}, themeColors{"accent1": {A: 255}}, themeEffectStyles{}, themeFillStyles{}, lineStyles)
	if !got.HasLine || got.LineDash != "" {
		t.Fatalf("explicit solid dash should suppress inherited style dash, got %+v", got)
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
	got := parseSlideElementNodeWithThemeEffectsAndFills(root, renderTransform{ScaleX: 1, ScaleY: 1}, themeColors{"accent1": {R: 200, G: 100, B: 50, A: 255}}, themeEffectStyles{}, fillStyles, themeLineStyles{})
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
      <a:outerShdw blurRad="50800" dist="38100" dir="2700000" algn="tl" rotWithShape="0" sx="120000" sy="80000" kx="60000" ky="-60000">
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
	if got.ShadowAlignment != "tl" || !got.HasShadowRotateWithShape || got.ShadowRotateWithShape {
		t.Fatalf("unexpected parsed shadow alignment/rotation: %+v", got)
	}
	if !got.HasShadowScaleX || got.ShadowScaleX != 120000 || !got.HasShadowScaleY || got.ShadowScaleY != 80000 || !got.HasShadowSkewX || got.ShadowSkewX != 60000 || !got.HasShadowSkewY || got.ShadowSkewY != -60000 {
		t.Fatalf("unexpected parsed shadow transform metadata: %+v", got)
	}
	if got.ShadowColor.A != 102 {
		t.Fatalf("expected alpha-modified shadow color, got %#v", got.ShadowColor)
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

func TestDrawShapeTextAllowsExplicitNoAutofitOverflow(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 260, 120))
	element := slideElement{
		Text:         "First\nSecond",
		FontSize:     2400,
		HasNoAutofit: true,
		TextParagraphs: []textParagraph{{
			Text:     "First",
			FontSize: 2400,
			Runs:     []textRun{{Text: "First", FontSize: 2400}},
		}, {
			Text:     "Second",
			FontSize: 2400,
			Runs:     []textRun{{Text: "Second", FontSize: 2400}},
		}},
	}

	bounds := image.Rect(10, 10, 250, 28)
	if err := drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}

	painted := opaqueBounds(img)
	if painted.Empty() || painted.Max.Y <= bounds.Max.Y {
		t.Fatalf("expected explicit no-autofit text to paint below text bounds %v, got %v", bounds, painted)
	}
}

func TestDrawShapeTextAllowsImplicitNoAutofitHorizontalOverflow(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 260, 80))
	element := slideElement{
		Text:     "This line should overflow",
		TextWrap: "none",
		FontSize: 2400,
		TextParagraphs: []textParagraph{{
			Text:     "This line should overflow",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This line should overflow", FontSize: 2400}},
		}},
	}

	bounds := image.Rect(10, 10, 70, 70)
	if err := drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}

	painted := opaqueBounds(img)
	if painted.Empty() || painted.Max.X <= bounds.Max.X {
		t.Fatalf("expected implicit no-autofit text to paint past right text bounds %v, got %v", bounds, painted)
	}
}

func TestDrawShapeTextClipsHorizontalOverflowWhenRequested(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 260, 80))
	element := slideElement{
		Text:                      "This line should overflow",
		TextWrap:                  "none",
		HasTextHorizontalOverflow: true,
		TextHorizontalOverflow:    "clip",
		FontSize:                  2400,
		TextParagraphs: []textParagraph{{
			Text:     "This line should overflow",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This line should overflow", FontSize: 2400}},
		}},
	}

	bounds := image.Rect(10, 10, 70, 70)
	if err := drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}

	painted := opaqueBounds(img)
	if painted.Empty() || painted.Max.X > bounds.Max.X {
		t.Fatalf("expected horizontal overflow clip to constrain text to bounds %v, got %v", bounds, painted)
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
		Text3DFeatures:      []string{"text 3-D top bevel"},
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
	for _, want := range []string{"vertical mode", "rotation", "columns", "text 3-D top bevel"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in unsupported messages, got %s", want, got)
		}
	}
	if strings.Contains(got, "anchor-center") {
		t.Fatalf("anchor-center is supported for horizontal text and should not be reported independently, got %s", got)
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

func TestRenderShapeDoesNotReportSupportedDerivedNormalAutofitScale(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 140, 70))
	element := slideElement{
		Kind:           "sp",
		Name:           "Derived Autofit Text",
		PrstGeom:       "rect",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		ExtCY:          emuPerInch / 2,
		Text:           "Wide Heading With Several Words",
		FontFamily:     "Carlito",
		FontSize:       2400,
		HasNormAutofit: true,
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 2400,
			Runs:     []textRun{{Text: "Wide Heading With Several Words", FontSize: 2400}},
		}},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	got := unsupportedMessages(unsupported)
	if strings.Contains(got, "normal autofit was rendered with simplified sizing") {
		t.Fatalf("derived normal-autofit sizing should be supported when a fitting scale exists, got %s", got)
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
		t.Fatalf("expected source-backed circle gradient to be fully supported: %+v", got.Gradient)
	}
}

func TestParseSlideBackgroundGradientSupportsRectangularPath(t *testing.T) {
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
	if got.Gradient.Path != "rect" || !got.Gradient.FullySupported {
		t.Fatalf("rectangular gradient path should be fully supported: %+v", got.Gradient)
	}
}

func TestGradientFillReportsUnsupportedTileRectAndFlip(t *testing.T) {
	for _, tc := range []struct {
		name string
		xml  string
	}{
		{
			name: "non-empty tileRect",
			xml: `<a:gradFill xmlns:a="a">
  <a:gsLst><a:gs pos="0"><a:srgbClr val="FFFFFF"/></a:gs><a:gs pos="100000"><a:srgbClr val="000000"/></a:gs></a:gsLst>
  <a:path path="circle"><a:fillToRect l="50000" t="50000" r="50000" b="50000"/></a:path>
  <a:tileRect l="10000"/>
</a:gradFill>`,
		},
		{
			name: "flip",
			xml: `<a:gradFill xmlns:a="a" flip="xy">
  <a:gsLst><a:gs pos="0"><a:srgbClr val="FFFFFF"/></a:gs><a:gs pos="100000"><a:srgbClr val="000000"/></a:gs></a:gsLst>
  <a:path path="circle"><a:fillToRect l="50000" t="50000" r="50000" b="50000"/></a:path>
</a:gradFill>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := parseXMLNode([]byte(tc.xml))
			if err != nil {
				t.Fatal(err)
			}
			got, ok := parseGradientFill(root, defaultThemeColors())
			if !ok {
				t.Fatal("gradient was not parsed")
			}
			if got.FullySupported {
				t.Fatalf("expected unsupported gradient metadata for %s: %+v", tc.name, got)
			}
		})
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

func TestParseStylePropertiesAppliesThemeShape3DEffectReference(t *testing.T) {
	effectStyles := parseThemeEffectStyles([]byte(`<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
	  <a:themeElements><a:fmtScheme name="Office"><a:effectStyleLst>
	    <a:effectStyle>
	      <a:scene3d><a:camera prst="orthographicFront"/><a:lightRig rig="threePt" dir="t"/></a:scene3d>
	      <a:sp3d><a:bevelT w="63500" h="25400"/></a:sp3d>
	    </a:effectStyle>
	  </a:effectStyleLst></a:fmtScheme></a:themeElements>
	</a:theme>`))
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:nvSpPr><p:cNvPr id="2" name="Styled Bevel"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
	  <p:spPr><a:prstGeom prst="rect"/></p:spPr>
	  <p:style><a:effectRef idx="1"><a:schemeClr val="accent1"/></a:effectRef></p:style>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	got := parseSlideElementNodeWithThemeAndEffects(root, renderTransform{ScaleX: 1, ScaleY: 1}, defaultThemeColors(), effectStyles)
	if !got.HasShape3D || !slices.Contains(got.Shape3DFeatures, "3-D top bevel") || !slices.Contains(got.Shape3DFeatures, "3-D scene camera orthographicFront") || !slices.Contains(got.Shape3DFeatures, "3-D scene light rig threePt/t") {
		t.Fatalf("expected effectRef to apply theme 3-D shape properties, got %+v", got)
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
			Path:        "circle",
			HasFillRect: true,
			FillRect:    relativeRect{Left: 50000, Top: 50000, Right: 50000, Bottom: 50000},
			Stops: []gradientStop{
				{Position: 0, Color: color.RGBA{R: 255, A: 255}},
				{Position: 100000, Color: color.RGBA{B: 255, A: 255}},
			},
			FullySupported: true,
		},
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported circle gradient fill, got %+v", unsupported)
	}
	if got := img.RGBAAt(5, 5); got.A == 0 {
		t.Fatalf("expected circle gradient to render visible fill")
	}
}

func TestRenderShapeSupportsEllipseGradientFill(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 12, 12))
	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, &slideElement{
		Kind:            "sp",
		Name:            "Gradient Ellipse",
		HasTransform:    true,
		ExtCX:           emuPerInch,
		ExtCY:           emuPerInch,
		PrstGeom:        "ellipse",
		HasFill:         true,
		HasFillGradient: true,
		FillGradient: gradientPaint{
			Path:           "circle",
			FullySupported: true,
			Stops: []gradientStop{
				{Position: 0, Color: color.RGBA{R: 255, A: 255}},
				{Position: 100000, Color: color.RGBA{B: 255, A: 255}},
			},
		},
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported ellipse gradient fill, got %+v", unsupported)
	}
	if got := img.RGBAAt(0, 0); got.A != 0 {
		t.Fatalf("expected ellipse gradient to stay clipped to ellipse, got corner=%#v", got)
	}
	if got := img.RGBAAt(6, 6); got.A == 0 {
		t.Fatalf("expected ellipse center to be painted")
	}
}

func TestDrawGradientPolygonUsesPathBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	drawGradientPolygon(img, img.Bounds(), []pathPoint{
		{X: 0.5, Y: 0},
		{X: 1, Y: 0},
		{X: 1, Y: 1},
		{X: 0.5, Y: 1},
	}, gradientPaint{
		HasAngle: true,
		Angle:    0,
		Stops: []gradientStop{
			{Position: 0, Color: color.RGBA{R: 255, A: 255}},
			{Position: 100000, Color: color.RGBA{B: 255, A: 255}},
		},
	})
	if got := img.RGBAAt(9, 5); got.A != 0 {
		t.Fatalf("expected gradient polygon to stay clipped to path, got left pixel=%#v", got)
	}
	if got := img.RGBAAt(10, 5); got.R <= got.B {
		t.Fatalf("expected left edge of path-bounded gradient to start near first stop, got %#v", got)
	}
	if got := img.RGBAAt(19, 5); got.B <= got.R {
		t.Fatalf("expected right edge of path-bounded gradient to end near last stop, got %#v", got)
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

func TestRectangularGradientPositionUsesFillToRectFocus(t *testing.T) {
	gradient := gradientPaint{
		Path:        "rect",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 25000, Top: 25000, Right: 25000, Bottom: 25000},
	}
	if got := rectangularGradientPosition(image.Rect(0, 0, 100, 100), 50, 50, gradient); got != 0 {
		t.Fatalf("expected sample inside rectangular focus to stay at first stop, got %d", got)
	}
	if got := rectangularGradientPosition(image.Rect(0, 0, 100, 100), 0, 50, gradient); got < 95000 {
		t.Fatalf("expected sample near anchor edge to reach the outer stop, got %d", got)
	}
	if got := rectangularGradientPosition(image.Rect(0, 0, 100, 100), 13, 50, gradient); got <= 0 || got >= 100000 {
		t.Fatalf("expected sample between focus and anchor rectangles to interpolate, got %d", got)
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

func TestRadialGradientCenterFastPathMatchesCenteredFillRect(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 50)
	gradient := gradientPaint{
		Path:        "circle",
		HasFillRect: true,
		FillRect:    relativeRect{Left: 50000, Top: 50000, Right: 50000, Bottom: 50000},
	}
	centerX, centerY, scale, ok := radialGradientCenterFastPath(bounds, gradient)
	if !ok {
		t.Fatal("expected centered fill rect to use radial fast path")
	}
	params := radialGradientParamsForBounds(bounds, gradient)
	for _, sample := range []floatPoint{{X: 50.5, Y: 25.5}, {X: 10.5, Y: 25.5}, {X: 99.5, Y: 49.5}} {
		got := int64(math.Round(math.Hypot(sample.X-centerX, sample.Y-centerY) * scale))
		if got < 0 {
			got = 0
		} else if got > 100000 {
			got = 100000
		}
		want := radialGradientPositionWithParams(sample.X, sample.Y, params)
		if got != want {
			t.Fatalf("fast path position mismatch for %+v: got %d want %d", sample, got, want)
		}
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

func TestPresentationUnsupportedItemsReportsEmbeddedFonts(t *testing.T) {
	got := presentationUnsupportedItems("ppt/presentation.xml", []byte(`<p:presentation xmlns:p="p">
  <p:embeddedFontLst>
    <p:embeddedFont>
      <p:font typeface="Example Font"/>
      <p:regular r:id="rIdFont" xmlns:r="r"/>
    </p:embeddedFont>
  </p:embeddedFontLst>
</p:presentation>`))

	if len(got) != 1 {
		t.Fatalf("expected embedded font unsupported item, got %+v", got)
	}
	if got[0].Code != partialUnsupportedCode || got[0].Part != "ppt/presentation.xml" || !strings.Contains(got[0].Message, "embedded font") {
		t.Fatalf("unexpected embedded font unsupported item: %+v", got[0])
	}
}

func TestPresentationUnsupportedItemsIgnoresPresentationWithoutEmbeddedFonts(t *testing.T) {
	got := presentationUnsupportedItems("ppt/presentation.xml", []byte(`<p:presentation xmlns:p="p"><p:sldIdLst/></p:presentation>`))
	if len(got) != 0 {
		t.Fatalf("presentation without embedded fonts should not report unsupported items, got %+v", got)
	}
}

func TestTimingUnsupportedItemsAcceptsStaticVisibilityEntranceBuilds(t *testing.T) {
	got := timingUnsupportedItems("ppt/slides/slide1.xml", []byte(`<p:sld xmlns:p="p">
  <p:cSld/>
  <p:timing>
    <p:tnLst>
      <p:par>
        <p:cTn id="1" presetID="1" presetClass="entr">
          <p:childTnLst>
            <p:set>
              <p:cBhvr>
                <p:tgtEl><p:spTgt spid="7"/></p:tgtEl>
                <p:attrNameLst><p:attrName>style.visibility</p:attrName></p:attrNameLst>
              </p:cBhvr>
              <p:to><p:strVal val="visible"/></p:to>
            </p:set>
            <p:animEffect transition="in" filter="fade">
              <p:cBhvr><p:tgtEl><p:spTgt spid="7"/></p:tgtEl></p:cBhvr>
            </p:animEffect>
          </p:childTnLst>
        </p:cTn>
      </p:par>
    </p:tnLst>
  </p:timing>
</p:sld>`), []slideElement{{ID: "7", Name: "Animated Shape"}})

	if len(got) != 0 {
		t.Fatalf("visibility entrance builds should be handled as static final-state renders, got %+v", got)
	}
}

func TestTimingUnsupportedItemsReportsUnsupportedAnimationBehavior(t *testing.T) {
	got := timingUnsupportedItems("ppt/slides/slide1.xml", []byte(`<p:sld xmlns:p="p">
  <p:cSld/>
  <p:timing>
    <p:tnLst>
      <p:par>
        <p:cTn id="1" presetID="3" presetClass="emph">
          <p:childTnLst>
            <p:animMotion>
              <p:cBhvr>
                <p:tgtEl><p:spTgt spid="7"/></p:tgtEl>
                <p:attrNameLst><p:attrName>ppt_x</p:attrName></p:attrNameLst>
              </p:cBhvr>
            </p:animMotion>
          </p:childTnLst>
        </p:cTn>
      </p:par>
    </p:tnLst>
  </p:timing>
</p:sld>`), []slideElement{{ID: "7", Name: "Animated Shape"}})

	if len(got) != 1 {
		t.Fatalf("expected unsupported timing item, got %+v", got)
	}
	if got[0].Code != partialUnsupportedCode || got[0].Part != "ppt/slides/slide1.xml" {
		t.Fatalf("unexpected timing unsupported item: %+v", got[0])
	}
}

func TestTimingUnsupportedItemsIgnoresEmptyTimingTree(t *testing.T) {
	got := timingUnsupportedItems("ppt/slides/slide1.xml", []byte(`<p:sld xmlns:p="p"><p:cSld/><p:timing><p:tnLst/></p:timing></p:sld>`), nil)
	if len(got) != 0 {
		t.Fatalf("empty timing tree should not be reported, got %+v", got)
	}
}

func TestFilterInheritedPlaceholdersKeepsEnabledSlideNumberPlaceholder(t *testing.T) {
	elements := []slideElement{{
		Kind:            "sp",
		Name:            "Slide Number Placeholder",
		Text:            "‹#›",
		IsPlaceholder:   true,
		PlaceholderType: "sldNum",
		TextParagraphs: []textParagraph{{
			Text: "‹#›",
			Runs: []textRun{{Text: "‹#›", FieldType: "slidenum"}},
		}},
	}}
	sources := map[string]slideElement{
		"type:sldNum": {
			IsPlaceholder:   true,
			PlaceholderType: "sldNum",
			HasTransform:    true,
			OffX:            10,
			OffY:            20,
			ExtCX:           30,
			ExtCY:           40,
		},
	}

	settings := defaultHeaderFooterSettings()
	settings.SlideNumber = true
	got := filterInheritedPlaceholdersForRender(elements, sources, settings, true)
	if len(got) != 1 || !got[0].HasTransform || got[0].OffX != 10 || got[0].Text != "‹#›" {
		t.Fatalf("expected inherited slide-number placeholder to be resolved and kept, got %+v", got)
	}
	resolved := resolveTextFields(got, 9)
	if resolved[0].Text != "9" || resolved[0].TextParagraphs[0].Runs[0].Text != "9" {
		t.Fatalf("expected kept slide-number placeholder to resolve its field, got %+v", resolved)
	}
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

func pngPhysicalPixelsPerMeter(t *testing.T, path string) (uint32, uint32, bool) {
	t.Helper()
	chunk, ok := pngOptionalChunkData(t, path, "pHYs")
	if !ok {
		return 0, 0, false
	}
	if len(chunk) != 9 || chunk[8] != 1 {
		t.Fatalf("invalid pHYs chunk in %s: %v", path, chunk)
	}
	return readUint32BE(chunk[0:4]), readUint32BE(chunk[4:8]), true
}

func pngChunkData(t *testing.T, path string, chunkType string) []byte {
	t.Helper()
	chunk, ok := pngOptionalChunkData(t, path, chunkType)
	if !ok {
		t.Fatalf("missing PNG chunk %s in %s", chunkType, path)
	}
	return chunk
}

func pngOptionalChunkData(t *testing.T, path string, wantedType string) ([]byte, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read png: %v", err)
	}
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatalf("not a png: %s", path)
	}
	offset := 8
	for offset+8 <= len(data) {
		length := int(readUint32BE(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length+4 > len(data) {
			t.Fatalf("invalid png chunk %q in %s", chunkType, path)
		}
		chunk := data[offset : offset+length]
		offset += length + 4
		if chunkType == wantedType {
			return append([]byte(nil), chunk...), true
		}
		if chunkType == "IEND" {
			break
		}
	}
	return nil, false
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

func testJPEGAPP2ICCChunk(sequenceNumber int, sequenceTotal int, payload []byte) []byte {
	chunkPayload := append([]byte("ICC_PROFILE\x00"), byte(sequenceNumber), byte(sequenceTotal))
	chunkPayload = append(chunkPayload, payload...)
	length := len(chunkPayload) + 2
	data := []byte{0xff, 0xe2, byte(length >> 8), byte(length)}
	data = append(data, chunkPayload...)
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
