package render

import (
	"image"
	"math"
	"testing"
)

func TestRenderElementTransformBoundsFractionalRotationAndFlip(t *testing.T) {
	element := slideElement{
		Kind:         "sp",
		HasTransform: true,
		OffX:         125,
		OffY:         125,
		ExtCX:        250,
		ExtCY:        250,
		HasRotation:  true,
		Rotation:     5400000,
		FlipH:        true,
		FlipV:        true,
	}
	got := renderElementTransformFor(element, slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 101, 53))

	if got.Target != image.Rect(13, 7, 38, 20) {
		t.Fatalf("unexpected integer target: %v", got.Target)
	}
	assertFloat(t, got.FractionalTarget.MinX, 12.625)
	assertFloat(t, got.FractionalTarget.MinY, 6.625)
	assertFloat(t, got.FractionalTarget.MaxX, 37.875)
	assertFloat(t, got.FractionalTarget.MaxY, 19.875)
	if got.PixelBounds != (ObjectPixelBounds{MinX: 13, MinY: 7, MaxX: 37, MaxY: 19}) {
		t.Fatalf("unexpected pixel bounds: %+v", got.PixelBounds)
	}
	if got.RotationDegrees != 90 || !got.FlipH || !got.FlipV {
		t.Fatalf("unexpected rotation/flip model: %+v", got)
	}
}

func TestRenderElementTransformBoundsAreClippedForObjectMasks(t *testing.T) {
	element := slideElement{
		Kind:         "pic",
		HasTransform: true,
		OffX:         -100,
		OffY:         900,
		ExtCX:        300,
		ExtCY:        300,
	}
	size := slideSize{CX: 1000, CY: 1000}
	canvas := image.Rect(0, 0, 100, 100)

	got := renderElementTransformFor(element, size, canvas)
	if got.Target != image.Rect(-10, 90, 20, 120) || got.ClippedTarget != image.Rect(0, 90, 20, 100) {
		t.Fatalf("unexpected raw/clipped target: raw=%v clipped=%v", got.Target, got.ClippedTarget)
	}
	if objectPixelBounds(element, size, canvas) != got.ClippedPixelBounds {
		t.Fatalf("object debug pixel bounds diverged from shared transform: debug=%+v shared=%+v", objectPixelBounds(element, size, canvas), got.ClippedPixelBounds)
	}
	if got.ClippedPixelBounds != (ObjectPixelBounds{MinX: 0, MinY: 90, MaxX: 19, MaxY: 99}) {
		t.Fatalf("unexpected clipped object mask bounds: %+v", got.ClippedPixelBounds)
	}
}

func TestRenderElementTransformZeroAndNegativeSizesDoNotPanic(t *testing.T) {
	size := slideSize{CX: 1000, CY: 1000}
	canvas := image.Rect(0, 0, 100, 100)
	for _, element := range []slideElement{
		{Kind: "sp", HasTransform: true, ExtCX: 0, ExtCY: 100},
		{Kind: "sp", HasTransform: true, ExtCX: 100, ExtCY: 0},
		{Kind: "sp", HasTransform: true, ExtCX: -100, ExtCY: 100},
		{Kind: "sp", HasTransform: true, ExtCX: 100, ExtCY: -100},
	} {
		got := renderElementTransformFor(element, size, canvas)
		if !got.Target.Empty() || got.PixelBounds != (ObjectPixelBounds{}) || got.ClippedPixelBounds != (ObjectPixelBounds{}) {
			t.Fatalf("expected empty bounds for non-positive extent, element=%+v got=%+v", element, got)
		}
	}
}

func TestLineEndpointsForElementUseSharedTransformAndFlipModel(t *testing.T) {
	element := slideElement{
		Kind:         "cxnSp",
		HasTransform: true,
		OffX:         100,
		OffY:         200,
		ExtCX:        300,
		ExtCY:        400,
		FlipH:        true,
	}
	startX, startY, endX, endY := lineEndpointsForElement(element, slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100))
	if startX != 40 || startY != 20 || endX != 10 || endY != 60 {
		t.Fatalf("unexpected line endpoints: start=(%d,%d) end=(%d,%d)", startX, startY, endX, endY)
	}
}

func TestCollectSlideElementsAppliesNestedGroupTransformStack(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:grpSp>
        <p:grpSpPr>
          <a:xfrm>
            <a:off x="1000" y="2000"/>
            <a:ext cx="4000" cy="6000"/>
            <a:chOff x="0" y="0"/>
            <a:chExt cx="2000" cy="3000"/>
          </a:xfrm>
        </p:grpSpPr>
        <p:grpSp>
          <p:grpSpPr>
            <a:xfrm>
              <a:off x="100" y="200"/>
              <a:ext cx="1000" cy="1500"/>
              <a:chOff x="0" y="0"/>
              <a:chExt cx="500" cy="750"/>
            </a:xfrm>
          </p:grpSpPr>
          <p:sp>
            <p:nvSpPr><p:cNvPr id="4" name="Nested Group Rect"/></p:nvSpPr>
            <p:spPr>
              <a:xfrm><a:off x="10" y="20"/><a:ext cx="30" cy="40"/></a:xfrm>
              <a:prstGeom prst="rect"/>
            </p:spPr>
          </p:sp>
        </p:grpSp>
      </p:grpSp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one flattened nested grouped element, got %+v", elements)
	}
	got := elements[0]
	if got.OffX != 1240 || got.OffY != 2480 || got.ExtCX != 120 || got.ExtCY != 160 {
		t.Fatalf("unexpected nested group transform: %+v", got)
	}
}

func TestRenderObjectDebugRecordUsesSharedTransformBounds(t *testing.T) {
	element := slideElement{
		Kind:         "sp",
		ID:           "7",
		Name:         "Clipped Rect",
		HasTransform: true,
		OffX:         -100,
		OffY:         100,
		ExtCX:        300,
		ExtCY:        200,
	}
	size := slideSize{CX: 1000, CY: 1000}
	canvas := image.Rect(0, 0, 100, 100)

	record := paintedObjectRecord("ppt/slides/slide1.xml", "ppt/slides/slide1.xml", element, 1, size, canvas, nil, nil, false, nil)
	model := renderElementTransformFor(element, size, canvas)
	if record.PixelBounds != model.ClippedPixelBounds {
		t.Fatalf("object debug record did not use shared clipped bounds: record=%+v model=%+v", record.PixelBounds, model.ClippedPixelBounds)
	}
	if record.FractionalBounds != model.FractionalTarget {
		t.Fatalf("object debug record did not use shared fractional bounds: record=%+v model=%+v", record.FractionalBounds, model.FractionalTarget)
	}
}

func assertFloat(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %f want %f", got, want)
	}
}
