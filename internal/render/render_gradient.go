package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func drawGradientBackground(img *image.RGBA, gradient gradientPaint) {
	drawGradientRect(img, img.Bounds(), gradient, true)
}

func drawGradientRect(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint, replace bool) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	if len(gradient.Stops) == 1 {
		op := draw.Over
		if replace || gradient.Stops[0].Color.A == 255 {
			op = draw.Src
		}
		draw.Draw(img, bounds, &image.Uniform{C: gradient.Stops[0].Color}, image.Point{}, op)
		return
	}
	if replace || gradientStopsAreOpaque(gradient.Stops) {
		drawOpaqueGradientRect(img, bounds, gradient)
		return
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var position int64
			switch gradient.Path {
			case "circle":
				position = radialGradientPosition(bounds, x, y, gradient)
			case "rect":
				position = rectangularGradientPosition(bounds, x, y, gradient)
			default:
				position = linearGradientPosition(bounds, x, y, gradient)
			}
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			if replace || c.A == 255 {
				img.SetRGBA(x, y, c)
			} else {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func gradientStopsAreOpaque(stops []gradientStop) bool {
	for _, stop := range stops {
		if stop.Color.A != 255 {
			return false
		}
	}
	return true
}

func drawOpaqueGradientRect(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint) {
	if gradient.Path == "" && !gradient.HasAngle {
		drawVerticalOpaqueGradientRect(img, bounds, gradient)
		return
	}
	if gradient.Path == "circle" {
		drawRadialOpaqueGradientRect(img, bounds, gradient)
		return
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := img.PixOffset(bounds.Min.X, y)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var position int64
			switch gradient.Path {
			case "circle":
				position = radialGradientPosition(bounds, x, y, gradient)
			case "rect":
				position = rectangularGradientPosition(bounds, x, y, gradient)
			default:
				position = linearGradientPosition(bounds, x, y, gradient)
			}
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			img.Pix[offset] = c.R
			img.Pix[offset+1] = c.G
			img.Pix[offset+2] = c.B
			img.Pix[offset+3] = c.A
			offset += 4
		}
	}
}

func drawRadialOpaqueGradientRect(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint) {
	params := radialGradientParamsForBounds(bounds, gradient)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := img.PixOffset(bounds.Min.X, y)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			position := radialGradientPositionWithParams(float64(x)+0.5, float64(y)+0.5, params)
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			img.Pix[offset] = c.R
			img.Pix[offset+1] = c.G
			img.Pix[offset+2] = c.B
			img.Pix[offset+3] = c.A
			offset += 4
		}
	}
}

func drawVerticalOpaqueGradientRect(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint) {
	height := bounds.Dy()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		position := int64(0)
		if height > 1 {
			position = int64(math.Round((float64(y) + 0.5 - float64(bounds.Min.Y)) / float64(height) * 100000))
		}
		if position < 0 {
			position = 0
		} else if position > 100000 {
			position = 100000
		}
		c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
		offset := img.PixOffset(bounds.Min.X, y)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Pix[offset] = c.R
			img.Pix[offset+1] = c.G
			img.Pix[offset+2] = c.B
			img.Pix[offset+3] = c.A
			offset += 4
		}
	}
}

func drawGradientEllipse(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	if radiusX <= 0 || radiusY <= 0 {
		return
	}
	centerX := float64(bounds.Min.X) + radiusX
	centerY := float64(bounds.Min.Y) + radiusY
	drawGradientWithCoverage(img, bounds, bounds, gradient, func(x int, y int) int {
		coverage := 0
		for _, offset := range coverageSampleOffsets {
			dx := (float64(x) + offset.x - centerX) / radiusX
			dy := (float64(y) + offset.y - centerY) / radiusY
			if dx*dx+dy*dy <= 1 {
				coverage++
			}
		}
		return coverage
	})
}

func drawGradientRoundRect(img *image.RGBA, bounds image.Rectangle, radius int, corners roundedCorners, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	if radius <= 0 {
		drawGradientRect(img, bounds, gradient, false)
		return
	}
	drawGradientWithCoverage(img, bounds, bounds, gradient, func(x int, y int) int {
		return roundRectCoverage(float64(x), float64(y), bounds, radius, corners)
	})
}

func drawGradientPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(points) < 3 || len(gradient.Stops) == 0 {
		return
	}
	polygon := polygonImagePoints(bounds, points)
	gradientBounds := pathPointBoundsRect(bounds, points).Intersect(img.Bounds())
	if gradientBounds.Empty() {
		return
	}
	drawGradientWithCoverage(img, bounds, gradientBounds, gradient, func(x int, y int) int {
		return polygonCoverage(float64(x), float64(y), polygon)
	})
}

func drawGradientWithCoverage(img *image.RGBA, paintBounds image.Rectangle, gradientBounds image.Rectangle, gradient gradientPaint, coverageAt func(x int, y int) int) {
	paintBounds = paintBounds.Intersect(img.Bounds())
	if paintBounds.Empty() || gradientBounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	for y := paintBounds.Min.Y; y < paintBounds.Max.Y; y++ {
		for x := paintBounds.Min.X; x < paintBounds.Max.X; x++ {
			coverage := coverageAt(x, y)
			if coverage <= 0 {
				continue
			}
			var position int64
			switch gradient.Path {
			case "circle":
				position = radialGradientPosition(gradientBounds, x, y, gradient)
			case "rect":
				position = rectangularGradientPosition(gradientBounds, x, y, gradient)
			default:
				position = linearGradientPosition(gradientBounds, x, y, gradient)
			}
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			if coverage >= 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
				continue
			}
			c.A = coverageAlpha(c.A, coverage)
			blendPixel(img, x, y, c)
		}
	}
}

func linearGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	if bounds.Dx() <= 1 && bounds.Dy() <= 1 {
		return 0
	}
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	if !gradient.HasAngle {
		height := bounds.Dy()
		if height <= 1 {
			return 0
		}
		position := (sampleY - float64(bounds.Min.Y)) / float64(height)
		if position < 0 {
			position = 0
		} else if position > 1 {
			position = 1
		}
		return int64(math.Round(position * 100000))
	}
	radians := float64(gradient.Angle) / 60000 * math.Pi / 180
	dx := math.Cos(radians)
	dy := math.Sin(radians)
	if gradient.HasScaled && gradient.Scaled {
		dx *= float64(bounds.Dx())
		dy *= float64(bounds.Dy())
	}
	corners := []struct {
		X float64
		Y float64
	}{
		{X: float64(bounds.Min.X), Y: float64(bounds.Min.Y)},
		{X: float64(bounds.Max.X - 1), Y: float64(bounds.Min.Y)},
		{X: float64(bounds.Min.X), Y: float64(bounds.Max.Y - 1)},
		{X: float64(bounds.Max.X - 1), Y: float64(bounds.Max.Y - 1)},
	}
	minProjection := math.Inf(1)
	maxProjection := math.Inf(-1)
	for _, corner := range corners {
		projection := corner.X*dx + corner.Y*dy
		if projection < minProjection {
			minProjection = projection
		}
		if projection > maxProjection {
			maxProjection = projection
		}
	}
	span := maxProjection - minProjection
	if span <= 0 {
		return 0
	}
	projection := (sampleX*dx + sampleY*dy - minProjection) / span
	if projection < 0 {
		projection = 0
	} else if projection > 1 {
		projection = 1
	}
	return int64(math.Round(projection * 100000))
}

func floatRectFromImageRect(rect image.Rectangle) floatRect {
	return floatRect{
		MinX: float64(rect.Min.X),
		MinY: float64(rect.Min.Y),
		MaxX: float64(rect.Max.X),
		MaxY: float64(rect.Max.Y),
	}
}

func floatRectPixelBounds(rect floatRect) image.Rectangle {
	return image.Rect(
		int(math.Floor(rect.MinX)),
		int(math.Floor(rect.MinY)),
		int(math.Ceil(rect.MaxX)),
		int(math.Ceil(rect.MaxY)),
	)
}

func rectangularGradientFocusRect(bounds image.Rectangle, gradient gradientPaint) floatRect {
	if !gradient.HasFillRect {
		centerX := float64(bounds.Min.X) + float64(bounds.Dx())/2
		centerY := float64(bounds.Min.Y) + float64(bounds.Dy())/2
		return floatRect{MinX: centerX, MinY: centerY, MaxX: centerX, MaxY: centerY}
	}
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())
	left := float64(bounds.Min.X) + width*float64(gradient.FillRect.Left)/100000
	top := float64(bounds.Min.Y) + height*float64(gradient.FillRect.Top)/100000
	right := float64(bounds.Max.X) - width*float64(gradient.FillRect.Right)/100000
	bottom := float64(bounds.Max.Y) - height*float64(gradient.FillRect.Bottom)/100000
	if right < left {
		center := (left + right) / 2
		left = center
		right = center
	}
	if bottom < top {
		center := (top + bottom) / 2
		top = center
		bottom = center
	}
	return floatRect{MinX: left, MinY: top, MaxX: right, MaxY: bottom}
}

func rectangularGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0
	}
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	outer := floatRect{
		MinX: float64(bounds.Min.X),
		MinY: float64(bounds.Min.Y),
		MaxX: float64(bounds.Max.X),
		MaxY: float64(bounds.Max.Y),
	}
	inner := rectangularGradientFocusRect(bounds, gradient)
	if pointInRect(sampleX, sampleY, inner) {
		return 0
	}
	position := 0.0
	if sampleX < inner.MinX {
		position = math.Max(position, normalizedGradientDistance(inner.MinX-sampleX, inner.MinX-outer.MinX))
	} else if sampleX > inner.MaxX {
		position = math.Max(position, normalizedGradientDistance(sampleX-inner.MaxX, outer.MaxX-inner.MaxX))
	}
	if sampleY < inner.MinY {
		position = math.Max(position, normalizedGradientDistance(inner.MinY-sampleY, inner.MinY-outer.MinY))
	} else if sampleY > inner.MaxY {
		position = math.Max(position, normalizedGradientDistance(sampleY-inner.MaxY, outer.MaxY-inner.MaxY))
	}
	return int64(math.Round(clampGradientRatio(position) * 100000))
}

func normalizedGradientDistance(distance float64, span float64) float64 {
	if distance <= 0 {
		return 0
	}
	if span <= 0 {
		return 1
	}
	return distance / span
}

func pointInRect(x float64, y float64, rect floatRect) bool {
	return x >= rect.MinX && x <= rect.MaxX && y >= rect.MinY && y <= rect.MaxY
}

func gradientFocusRect(bounds image.Rectangle, gradient gradientPaint) floatRect {
	if gradient.Path != "circle" {
		return floatRect{}
	}
	outer := radialGradientOuterRect(bounds)
	if !gradient.HasFillRect {
		centerX := (outer.MinX + outer.MaxX) / 2
		centerY := (outer.MinY + outer.MaxY) / 2
		return floatRect{MinX: centerX, MinY: centerY, MaxX: centerX, MaxY: centerY}
	}
	width := outer.MaxX - outer.MinX
	height := outer.MaxY - outer.MinY
	left := outer.MinX + width*float64(gradient.FillRect.Left)/100000
	top := outer.MinY + height*float64(gradient.FillRect.Top)/100000
	right := outer.MaxX - width*float64(gradient.FillRect.Right)/100000
	bottom := outer.MaxY - height*float64(gradient.FillRect.Bottom)/100000
	if right < left {
		center := (left + right) / 2
		left = center
		right = center
	}
	if bottom < top {
		center := (top + bottom) / 2
		top = center
		bottom = center
	}
	return floatRect{MinX: left, MinY: top, MaxX: right, MaxY: bottom}
}

func radialGradientOuterRect(bounds image.Rectangle) floatRect {
	centerX := float64(bounds.Min.X) + float64(bounds.Dx())/2
	centerY := float64(bounds.Min.Y) + float64(bounds.Dy())/2
	diameter := math.Hypot(float64(bounds.Dx()), float64(bounds.Dy()))
	radius := diameter / 2
	return floatRect{
		MinX: centerX - radius,
		MinY: centerY - radius,
		MaxX: centerX + radius,
		MaxY: centerY + radius,
	}
}

func radialGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	return radialGradientPositionWithParams(sampleX, sampleY, radialGradientParamsForBounds(bounds, gradient))
}

type radialGradientParams struct {
	Origin floatPoint
	Inner  floatRect
	Outer  floatRect
}

func radialGradientParamsForBounds(bounds image.Rectangle, gradient gradientPaint) radialGradientParams {
	outer := radialGradientOuterRect(bounds)
	inner := gradientFocusRect(bounds, gradient)
	return radialGradientParams{
		Origin: radialGradientFocusPointFromRects(inner, outer),
		Inner:  inner,
		Outer:  outer,
	}
}

func radialGradientPositionWithParams(sampleX float64, sampleY float64, params radialGradientParams) int64 {
	dx := sampleX - params.Origin.X
	dy := sampleY - params.Origin.Y
	distance := math.Hypot(dx, dy)
	if distance <= 0 {
		return 0
	}

	if pointInEllipse(sampleX, sampleY, params.Inner) {
		return 0
	}
	innerDistance := rayEllipseExitDistance(params.Origin, dx/distance, dy/distance, params.Inner)
	outerDistance := rayEllipseExitDistance(params.Origin, dx/distance, dy/distance, params.Outer)
	if outerDistance <= innerDistance {
		return 100000
	}
	position := (distance - innerDistance) / (outerDistance - innerDistance)
	if position < 0 {
		position = 0
	} else if position > 1 {
		position = 1
	}
	return int64(math.Round(position * 100000))
}

func pointInEllipse(x float64, y float64, rect floatRect) bool {
	width := rect.MaxX - rect.MinX
	height := rect.MaxY - rect.MinY
	if width <= 0 || height <= 0 {
		return false
	}
	rx := width / 2
	ry := height / 2
	cx := (rect.MinX + rect.MaxX) / 2
	cy := (rect.MinY + rect.MaxY) / 2
	nx := (x - cx) / rx
	ny := (y - cy) / ry
	return nx*nx+ny*ny <= 1
}

func rayEllipseExitDistance(origin floatPoint, unitX float64, unitY float64, rect floatRect) float64 {
	width := rect.MaxX - rect.MinX
	height := rect.MaxY - rect.MinY
	if width <= 0 || height <= 0 {
		return 0
	}
	rx := width / 2
	ry := height / 2
	cx := (rect.MinX + rect.MaxX) / 2
	cy := (rect.MinY + rect.MaxY) / 2
	ox := origin.X - cx
	oy := origin.Y - cy
	a := unitX*unitX/(rx*rx) + unitY*unitY/(ry*ry)
	if a <= 0 {
		return 0
	}
	b := 2 * (ox*unitX/(rx*rx) + oy*unitY/(ry*ry))
	c := ox*ox/(rx*rx) + oy*oy/(ry*ry) - 1
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0
	}
	root := math.Sqrt(discriminant)
	first := (-b - root) / (2 * a)
	second := (-b + root) / (2 * a)
	switch {
	case first >= 0 && second >= 0:
		return math.Max(first, second)
	case second >= 0:
		return second
	case first >= 0:
		return first
	default:
		return 0
	}
}

func radialGradientFocusPoint(bounds image.Rectangle, gradient gradientPaint) floatPoint {
	inner := gradientFocusRect(bounds, gradient)
	outer := radialGradientOuterRect(bounds)
	return radialGradientFocusPointFromRects(inner, outer)
}

func radialGradientFocusPointFromRects(inner floatRect, outer floatRect) floatPoint {
	outerLeft := outer.MinX
	outerTop := outer.MinY
	outerWidth := outer.MaxX - outer.MinX
	outerHeight := outer.MaxY - outer.MinY
	innerWidth := inner.MaxX - inner.MinX
	innerHeight := inner.MaxY - inner.MinY
	point := floatPoint{X: inner.MinX, Y: inner.MinY}
	if innerWidth > 0 {
		widthDiff := outerWidth - innerWidth
		if math.Abs(widthDiff) > 2*math.SmallestNonzeroFloat64 {
			point.X += innerWidth * (inner.MinX - outerLeft) / widthDiff
		}
	}
	if innerHeight > 0 {
		heightDiff := outerHeight - innerHeight
		if math.Abs(heightDiff) > 2*math.SmallestNonzeroFloat64 {
			point.Y += innerHeight * (inner.MinY - outerTop) / heightDiff
		}
	}
	return point
}

func colorAtGradientPositionForPath(stops []gradientStop, position int64, path string) color.RGBA {
	return colorAtGradientPosition(stops, position)
}

func colorAtGradientPosition(stops []gradientStop, position int64) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if c, ok := colorAtOfficeGammaGradientPosition(stops, position); ok {
		return c
	}
	if position <= stops[0].Position {
		return stops[0].Color
	}
	for index := 1; index < len(stops); index++ {
		right := stops[index]
		if position > right.Position {
			continue
		}
		left := stops[index-1]
		span := right.Position - left.Position
		if span <= 0 {
			return right.Color
		}
		numerator := position - left.Position
		return color.RGBA{
			R: interpolateChannel(left.Color.R, right.Color.R, numerator, span),
			G: interpolateChannel(left.Color.G, right.Color.G, numerator, span),
			B: interpolateChannel(left.Color.B, right.Color.B, numerator, span),
			A: interpolateChannel(left.Color.A, right.Color.A, numerator, span),
		}
	}
	return stops[len(stops)-1].Color
}

func colorAtOfficeGammaGradientPosition(stops []gradientStop, position int64) (color.RGBA, bool) {
	if len(stops) == 2 && stops[0].Position == 0 && stops[1].Position == 100000 {
		t := clampGradientRatio(float64(position) / 100000)
		return interpolateOfficeGammaColor(stops[0].Color, stops[1].Color, t), true
	}
	if len(stops) == 3 && stops[0].Position == 0 && stops[2].Position == 100000 && stops[0].Color == stops[2].Color {
		mid := stops[1].Position
		if mid <= 0 || mid >= 100000 {
			return color.RGBA{}, false
		}
		if position <= mid {
			t := clampGradientRatio(float64(position) / float64(mid))
			return interpolateOfficeGammaColor(stops[0].Color, stops[1].Color, t), true
		}
		t := clampGradientRatio(float64(100000-position) / float64(100000-mid))
		return interpolateOfficeGammaColor(stops[2].Color, stops[1].Color, t), true
	}
	return color.RGBA{}, false
}

func clampGradientRatio(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func interpolateOfficeGammaColor(left color.RGBA, right color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: interpolateOfficeGammaChannel(left.R, right.R, t),
		G: interpolateOfficeGammaChannel(left.G, right.G, t),
		B: interpolateOfficeGammaChannel(left.B, right.B, t),
		A: interpolateLinearFloatChannel(left.A, right.A, t),
	}
}

func interpolateOfficeGammaChannel(left uint8, right uint8, t float64) uint8 {
	if left == right {
		return left
	}
	ratio := math.Pow(t, 1.875)
	if right > left {
		ratio = 1 - math.Pow(1-t, 1.875)
	}
	return interpolateFloatChannel(left, right, ratio)
}

func interpolateLinearFloatChannel(left uint8, right uint8, t float64) uint8 {
	return interpolateFloatChannel(left, right, t)
}

func interpolateFloatChannel(left uint8, right uint8, t float64) uint8 {
	value := float64(left) + (float64(right)-float64(left))*t
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}
	return uint8(math.Round(value))
}

func interpolateChannel(left uint8, right uint8, numerator int64, denominator int64) uint8 {
	if denominator <= 0 {
		return right
	}
	return uint8((int64(left)*(denominator-numerator) + int64(right)*numerator + denominator/2) / denominator)
}
