package render

import "image"

type renderElementTransformModel struct {
	HasTransform       bool
	BoundsEMU          ObjectEMUPointBounds
	Target             image.Rectangle
	ClippedTarget      image.Rectangle
	PixelBounds        ObjectPixelBounds
	ClippedPixelBounds ObjectPixelBounds
	FractionalTarget   ObjectFloatBounds
	RotationDegrees    int
	FlipH              bool
	FlipV              bool
}

func renderElementTransformFor(element slideElement, size slideSize, canvas image.Rectangle) renderElementTransformModel {
	model := renderElementTransformModel{
		HasTransform:    element.HasTransform,
		BoundsEMU:       objectEMUPointBounds(element),
		RotationDegrees: normalizedRotationDegrees(element.Rotation),
		FlipH:           element.FlipH,
		FlipV:           element.FlipV,
	}
	if !element.HasTransform || size.CX <= 0 || size.CY <= 0 || canvas.Dx() <= 0 || canvas.Dy() <= 0 {
		return model
	}

	// DrawingML xfrm stores off/ext in EMUs. Group transforms are composed into
	// these values during parse; this helper is the single EMU-to-pixel boundary.
	model.Target = image.Rectangle{
		Min: image.Point{
			X: scaleEMU(element.OffX, size.CX, canvas.Dx()),
			Y: scaleEMU(element.OffY, size.CY, canvas.Dy()),
		},
		Max: image.Point{
			X: scaleEMU(element.OffX+element.ExtCX, size.CX, canvas.Dx()),
			Y: scaleEMU(element.OffY+element.ExtCY, size.CY, canvas.Dy()),
		},
	}
	model.FractionalTarget = ObjectFloatBounds{
		MinX: scaleEMUFloat(element.OffX, size.CX, canvas.Dx()),
		MinY: scaleEMUFloat(element.OffY, size.CY, canvas.Dy()),
		MaxX: scaleEMUFloat(element.OffX+element.ExtCX, size.CX, canvas.Dx()),
		MaxY: scaleEMUFloat(element.OffY+element.ExtCY, size.CY, canvas.Dy()),
	}
	model.ClippedTarget = model.Target.Intersect(canvas)
	model.PixelBounds = pixelBoundsFromRect(model.Target)
	model.ClippedPixelBounds = pixelBoundsFromRect(model.ClippedTarget)
	return model
}

func renderTextTransformTarget(element slideElement, size slideSize, canvas image.Rectangle) (image.Rectangle, bool) {
	if !element.HasTextTransform || element.TextExtCX <= 0 || element.TextExtCY <= 0 || size.CX <= 0 || size.CY <= 0 || canvas.Dx() <= 0 || canvas.Dy() <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		scaleEMU(element.TextOffX, size.CX, canvas.Dx()),
		scaleEMU(element.TextOffY, size.CY, canvas.Dy()),
		scaleEMU(element.TextOffX+element.TextExtCX, size.CX, canvas.Dx()),
		scaleEMU(element.TextOffY+element.TextExtCY, size.CY, canvas.Dy()),
	), true
}

func sceneElementPixelTarget(element slideElement, size slideSize, canvas image.Rectangle) image.Rectangle {
	return renderElementTransformFor(element, size, canvas).Target
}

func sceneElementClippedPixelTarget(element slideElement, size slideSize, canvas image.Rectangle) image.Rectangle {
	return renderElementTransformFor(element, size, canvas).ClippedTarget
}

func elementFractionalTarget(element slideElement, size slideSize, canvas image.Rectangle) ObjectFloatBounds {
	return renderElementTransformFor(element, size, canvas).FractionalTarget
}

func renderElementPixelBounds(element slideElement, size slideSize, canvas image.Rectangle) ObjectPixelBounds {
	return renderElementTransformFor(element, size, canvas).PixelBounds
}

func renderElementClippedPixelBounds(element slideElement, size slideSize, canvas image.Rectangle) ObjectPixelBounds {
	return renderElementTransformFor(element, size, canvas).ClippedPixelBounds
}

func lineEndpointsForElement(element slideElement, size slideSize, bounds image.Rectangle) (int, int, int, int) {
	target := sceneElementPixelTarget(element, size, bounds)
	startX, startY := target.Min.X, target.Min.Y
	endX, endY := target.Max.X, target.Max.Y
	if element.FlipH {
		startX, endX = endX, startX
	}
	if element.FlipV {
		startY, endY = endY, startY
	}
	return startX, startY, endX, endY
}
