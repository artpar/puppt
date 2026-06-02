package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
)

func drawPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA) {
	if len(points) < 3 || bounds.Empty() {
		return
	}
	polygon := polygonImagePoints(bounds, points)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := polygonCoverage(float64(x), float64(y), polygon)
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else if coverage > 0 {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func drawPolygonWithFloatBounds(img *image.RGBA, paintBounds image.Rectangle, bounds floatRect, points []pathPoint, c color.RGBA) {
	paintBounds = paintBounds.Intersect(img.Bounds())
	if len(points) < 3 || paintBounds.Empty() || c.A == 0 || bounds.MaxX <= bounds.MinX || bounds.MaxY <= bounds.MinY {
		return
	}
	polygon := polygonFloatPoints(bounds, points)
	for y := paintBounds.Min.Y; y < paintBounds.Max.Y; y++ {
		for x := paintBounds.Min.X; x < paintBounds.Max.X; x++ {
			coverage := polygonFloatCoverage(float64(x), float64(y), polygon)
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else if coverage > 0 {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func drawPolygonOutline(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA, width int) {
	if len(points) < 2 || bounds.Empty() {
		return
	}
	polygon := polygonImagePoints(bounds, points)
	for index := range polygon {
		next := (index + 1) % len(polygon)
		drawLine(img, polygon[index].X, polygon[index].Y, polygon[next].X, polygon[next].Y, c, width)
	}
}

func drawPathOutlineStyled(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA, width int, dash string, cap string, join string, compound string, closed bool) {
	if len(points) < 2 || bounds.Empty() {
		return
	}
	scaled := polygonImagePoints(bounds, points)
	limit := len(scaled) - 1
	if closed {
		limit = len(scaled)
	}
	for index := 0; index < limit; index++ {
		next := index + 1
		if next >= len(scaled) {
			next = 0
		}
		drawStyledCompoundLine(img, scaled[index].X, scaled[index].Y, scaled[next].X, scaled[next].Y, c, width, dash, cap, compound)
	}
	if join == "round" && (compound == "" || compound == "sng") {
		for index, point := range scaled {
			if !closed && (index == 0 || index == len(scaled)-1) {
				continue
			}
			drawRoundLineJoin(img, point.X, point.Y, c, width)
		}
	}
}

func polygonImagePoints(bounds image.Rectangle, points []pathPoint) []image.Point {
	polygon := make([]image.Point, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, image.Point{
			X: bounds.Min.X + int(math.Round(point.X*float64(bounds.Dx()))),
			Y: bounds.Min.Y + int(math.Round(point.Y*float64(bounds.Dy()))),
		})
	}
	return polygon
}

func polygonFloatPoints(bounds floatRect, points []pathPoint) []floatPoint {
	width := bounds.MaxX - bounds.MinX
	height := bounds.MaxY - bounds.MinY
	polygon := make([]floatPoint, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, floatPoint{
			X: bounds.MinX + point.X*width,
			Y: bounds.MinY + point.Y*height,
		})
	}
	return polygon
}

func pathPointBoundsRect(bounds image.Rectangle, points []pathPoint) image.Rectangle {
	if bounds.Empty() || len(points) == 0 {
		return image.Rectangle{}
	}
	minX := math.Inf(1)
	minY := math.Inf(1)
	maxX := math.Inf(-1)
	maxY := math.Inf(-1)
	for _, point := range points {
		x := float64(bounds.Min.X) + point.X*float64(bounds.Dx())
		y := float64(bounds.Min.Y) + point.Y*float64(bounds.Dy())
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if x > maxX {
			maxX = x
		}
		if y > maxY {
			maxY = y
		}
	}
	return image.Rect(
		int(math.Floor(minX)),
		int(math.Floor(minY)),
		int(math.Ceil(maxX)),
		int(math.Ceil(maxY)),
	)
}

var coverageSampleOffsets = []struct {
	x float64
	y float64
}{
	{x: 0.25, y: 0.25},
	{x: 0.75, y: 0.25},
	{x: 0.25, y: 0.75},
	{x: 0.75, y: 0.75},
}

func polygonCoverage(x float64, y float64, polygon []image.Point) int {
	coverage := 0
	for _, offset := range coverageSampleOffsets {
		if pointInPolygonFloat(x+offset.x, y+offset.y, polygon) {
			coverage++
		}
	}
	return coverage
}

func polygonFloatCoverage(x float64, y float64, polygon []floatPoint) int {
	coverage := 0
	for _, offset := range coverageSampleOffsets {
		if pointInPolygonFloatPoints(x+offset.x, y+offset.y, polygon) {
			coverage++
		}
	}
	return coverage
}

func coverageAlpha(alpha uint8, coverage int) uint8 {
	if coverage <= 0 || alpha == 0 {
		return 0
	}
	if coverage >= 4 {
		return alpha
	}
	return uint8((int(alpha)*coverage + 2) / 4)
}

func drawSoftPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA, blur int) {
	if len(points) < 3 || bounds.Empty() || c.A == 0 {
		return
	}
	polygon := make([]image.Point, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, image.Point{
			X: bounds.Min.X + int(math.Round(point.X*float64(bounds.Dx()))),
			Y: bounds.Min.Y + int(math.Round(point.Y*float64(bounds.Dy()))),
		})
	}
	drawBlurredShadowMask(img, bounds, c, blur, func(x int, y int) bool {
		return pointInPolygon(x, y, polygon)
	})
}

func drawBlendPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA) {
	if len(points) < 3 || bounds.Empty() || c.A == 0 {
		return
	}
	polygon := make([]image.Point, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, image.Point{
			X: bounds.Min.X + int(math.Round(point.X*float64(bounds.Dx()))),
			Y: bounds.Min.Y + int(math.Round(point.Y*float64(bounds.Dy()))),
		})
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if pointInPolygon(x, y, polygon) {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func drawEllipse(img *image.RGBA, bounds image.Rectangle, c color.RGBA) {
	if bounds.Empty() {
		return
	}
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	if radiusX <= 0 || radiusY <= 0 {
		return
	}
	centerX := float64(bounds.Min.X) + radiusX
	centerY := float64(bounds.Min.Y) + radiusY
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := (float64(x) + 0.5 - centerX) / radiusX
			dy := (float64(y) + 0.5 - centerY) / radiusY
			if dx*dx+dy*dy <= 1 {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func fillShapeRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	if rect.Empty() || c.A == 0 {
		return
	}
	if c.A == 255 {
		draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
		return
	}
	fillBlendRect(img, rect, c)
}

func fillShapeRectWithFloatBounds(img *image.RGBA, paintBounds image.Rectangle, rect floatRect, c color.RGBA) {
	paintBounds = paintBounds.Intersect(img.Bounds())
	if paintBounds.Empty() || c.A == 0 || rect.MaxX <= rect.MinX || rect.MaxY <= rect.MinY {
		return
	}
	for y := paintBounds.Min.Y; y < paintBounds.Max.Y; y++ {
		for x := paintBounds.Min.X; x < paintBounds.Max.X; x++ {
			coverage := floatRectPixelCoverage(float64(x), float64(y), rect)
			if coverage <= 0 {
				continue
			}
			if coverage >= 1 && c.A == 255 {
				img.SetRGBA(x, y, c)
				continue
			}
			layer := c
			layer.A = coverageFloatAlpha(c.A, coverage)
			blendPixel(img, x, y, layer)
		}
	}
}

func floatRectPixelCoverage(x float64, y float64, rect floatRect) float64 {
	overlapX := math.Min(x+1, rect.MaxX) - math.Max(x, rect.MinX)
	overlapY := math.Min(y+1, rect.MaxY) - math.Max(y, rect.MinY)
	if overlapX <= 0 || overlapY <= 0 {
		return 0
	}
	coverage := overlapX * overlapY
	if coverage > 1 {
		return 1
	}
	return coverage
}

func coverageFloatAlpha(alpha uint8, coverage float64) uint8 {
	if alpha == 0 || coverage <= 0 {
		return 0
	}
	if coverage >= 1 {
		return alpha
	}
	return uint8(math.Round(float64(alpha) * coverage))
}

func roundRectRadius(bounds image.Rectangle, adjustments map[string]int64) int {
	minDimension := minInt(bounds.Dx(), bounds.Dy())
	if minDimension <= 0 {
		return 0
	}
	adjustment := int64(16667)
	if value, ok := adjustments["adj"]; ok && value >= 0 {
		adjustment = value
	}
	radius := int(math.Round(float64(minDimension) * float64(adjustment) / 100000))
	maxRadius := minDimension / 2
	if radius > maxRadius {
		return maxRadius
	}
	if radius < 0 {
		return 0
	}
	return radius
}

func fillRoundRect(img *image.RGBA, bounds image.Rectangle, radius int, corners roundedCorners, c color.RGBA) {
	if bounds.Empty() || c.A == 0 {
		return
	}
	if radius <= 0 {
		fillShapeRect(img, bounds, c)
		return
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := roundRectCoverage(float64(x), float64(y), bounds, radius, corners)
			if coverage == 0 {
				continue
			}
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func roundRectCoverage(x float64, y float64, bounds image.Rectangle, radius int, corners roundedCorners) int {
	coverage := 0
	for _, offset := range coverageSampleOffsets {
		if pointInRoundRectAt(x+offset.x, y+offset.y, bounds, radius, corners) {
			coverage++
		}
	}
	return coverage
}

func pointInRoundRect(x int, y int, bounds image.Rectangle, radius int, corners roundedCorners) bool {
	return pointInRoundRectAt(float64(x)+0.5, float64(y)+0.5, bounds, radius, corners)
}

func pointInRoundRectAt(px float64, py float64, bounds image.Rectangle, radius int, corners roundedCorners) bool {
	r := float64(radius)
	if corners.TopLeft && px < float64(bounds.Min.X+radius) && py < float64(bounds.Min.Y+radius) {
		return pointInCircle(px, py, float64(bounds.Min.X)+r, float64(bounds.Min.Y)+r, r)
	}
	if corners.TopRight && px >= float64(bounds.Max.X-radius) && py < float64(bounds.Min.Y+radius) {
		return pointInCircle(px, py, float64(bounds.Max.X)-r, float64(bounds.Min.Y)+r, r)
	}
	if corners.BottomLeft && px < float64(bounds.Min.X+radius) && py >= float64(bounds.Max.Y-radius) {
		return pointInCircle(px, py, float64(bounds.Min.X)+r, float64(bounds.Max.Y)-r, r)
	}
	if corners.BottomRight && px >= float64(bounds.Max.X-radius) && py >= float64(bounds.Max.Y-radius) {
		return pointInCircle(px, py, float64(bounds.Max.X)-r, float64(bounds.Max.Y)-r, r)
	}
	return true
}

func pointInCircle(x float64, y float64, centerX float64, centerY float64, radius float64) bool {
	dx := x - centerX
	dy := y - centerY
	return dx*dx+dy*dy <= radius*radius
}

func drawSoftEllipse(img *image.RGBA, bounds image.Rectangle, c color.RGBA, blur int) {
	if bounds.Empty() || c.A == 0 {
		return
	}
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	if radiusX <= 0 || radiusY <= 0 {
		return
	}
	centerX := float64(bounds.Min.X) + radiusX
	centerY := float64(bounds.Min.Y) + radiusY
	drawBlurredShadowMask(img, bounds, c, blur, func(x int, y int) bool {
		dx := (float64(x) + 0.5 - centerX) / radiusX
		dy := (float64(y) + 0.5 - centerY) / radiusY
		return dx*dx+dy*dy <= 1
	})
}

func drawBlendEllipse(img *image.RGBA, bounds image.Rectangle, c color.RGBA) {
	if bounds.Empty() || c.A == 0 {
		return
	}
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	if radiusX <= 0 || radiusY <= 0 {
		return
	}
	centerX := float64(bounds.Min.X) + radiusX
	centerY := float64(bounds.Min.Y) + radiusY
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := (float64(x) + 0.5 - centerX) / radiusX
			dy := (float64(y) + 0.5 - centerY) / radiusY
			if dx*dx+dy*dy <= 1 {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func drawEllipseOutline(img *image.RGBA, bounds image.Rectangle, c color.RGBA, width int) {
	if bounds.Empty() {
		return
	}
	outerRadiusX := float64(bounds.Dx()) / 2
	outerRadiusY := float64(bounds.Dy()) / 2
	if outerRadiusX <= 0 || outerRadiusY <= 0 {
		return
	}
	innerRadiusX := outerRadiusX - float64(width)
	innerRadiusY := outerRadiusY - float64(width)
	centerX := float64(bounds.Min.X) + outerRadiusX
	centerY := float64(bounds.Min.Y) + outerRadiusY
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dx := float64(x) + 0.5 - centerX
			dy := float64(y) + 0.5 - centerY
			outer := (dx*dx)/(outerRadiusX*outerRadiusX) + (dy*dy)/(outerRadiusY*outerRadiusY)
			if outer > 1 {
				continue
			}
			if innerRadiusX <= 0 || innerRadiusY <= 0 {
				img.SetRGBA(x, y, c)
				continue
			}
			inner := (dx*dx)/(innerRadiusX*innerRadiusX) + (dy*dy)/(innerRadiusY*innerRadiusY)
			if inner >= 1 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

func drawRightBrace(img *image.RGBA, bounds image.Rectangle, element slideElement, c color.RGBA, width int) {
	if bounds.Empty() {
		return
	}
	drawOpenPathOutline(img, bounds, rightBracePresetPath(element), c, width)
}

func drawCurvedArrow(img *image.RGBA, bounds image.Rectangle, element slideElement, c color.RGBA) {
	if bounds.Empty() {
		return
	}
	paths := curvedArrowPresetFillPaths(element)
	if len(paths) == 0 {
		return
	}
	drawPolygon(img, bounds, paths[0], c)
	if len(paths) > 1 {
		drawPolygon(img, bounds, paths[1], c)
		drawPolygon(img, bounds, paths[1], darkenLessPathFill())
	}
}

func curvedArrowPresetFillPaths(element slideElement) [][]pathPoint {
	g := curvedArrowGuideValues(element)
	if g.W <= 0 || g.H <= 0 || g.WR <= 0 {
		return nil
	}
	var paths [][]pathPoint
	if element.PrstGeom == "curvedUpArrow" {
		main := []pathPoint{{X: g.X6, Y: 0}, {X: g.X8, Y: g.Y1}, {X: g.X7, Y: g.Y1}}
		main = appendCurvedArrowArcPoints(main, g.WR, g.H, g.StAng3, g.SwAng3, true)
		main = appendCurvedArrowArcPoints(main, g.WR, g.H, g.StAng2, g.SwAng2, true)
		main = append(main, pathPoint{X: g.X4, Y: g.Y1})

		shade := []pathPoint{{X: g.WR, Y: g.H}}
		shade = appendCurvedArrowArcPoints(shade, g.WR, g.H, 5400000, 5400000, false)
		shade = append(shade, pathPoint{X: g.Th, Y: 0})
		shade = appendCurvedArrowArcPoints(shade, g.WR, g.H, 10800000, -5400000, false)
		paths = [][]pathPoint{main, shade}
	} else {
		main := []pathPoint{{X: g.X6, Y: g.H}, {X: g.X4, Y: g.Y1}, {X: g.X5, Y: g.Y1}}
		main = appendCurvedArrowArcPoints(main, g.WR, g.H, g.StAng, g.MSwAng, false)
		main = append(main, pathPoint{X: g.X3, Y: 0})
		main = appendCurvedArrowArcPoints(main, g.WR, g.H, 16200000, g.SwAng, false)
		main = append(main, pathPoint{X: g.X8, Y: g.Y1})

		shade := []pathPoint{{X: g.IX, Y: g.IY}}
		shade = appendCurvedArrowArcPoints(shade, g.WR, g.H, g.StAng2, g.SwAng2, false)
		shade = append(shade, pathPoint{X: 0, Y: g.H})
		shade = appendCurvedArrowArcPoints(shade, g.WR, g.H, 10800000, g.SwAng3, false)
		paths = [][]pathPoint{main, shade}
	}
	for index := range paths {
		paths[index] = transformedPathPoints(normalizedGeometryPathPoints(paths[index], g.W, g.H), element)
	}
	return paths
}

func curvedArrowPresetOutlinePath(element slideElement) []pathPoint {
	paths := curvedArrowPresetFillPaths(element)
	if len(paths) == 0 || len(paths[0]) < 2 {
		return nil
	}
	points := append([]pathPoint{}, paths[0]...)
	points = append(points, points[0])
	return points
}

func curvedArrowGuideValues(element slideElement) curvedArrowGuides {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj1 := presetAdjustment(element, "adj1", 25000)
	adj2 := presetAdjustment(element, "adj2", 50000)
	adj3 := presetAdjustment(element, "adj3", 25000)
	maxAdj2 := 50000 * w / ss
	a1 := clampFloat(adj1, 0, 100000)
	a2 := clampFloat(adj2, 0, maxAdj2)
	th := ss * a1 / 100000
	aw := ss * a2 / 100000
	q1 := (th + aw) / 4
	wR := w/2 - q1
	q7 := wR * 2
	idy := 0.0
	if q7 != 0 {
		idy = math.Sqrt(math.Max(q7*q7-th*th, 0)) * h / q7
	}
	maxAdj3 := 100000 * idy / ss
	a3 := clampFloat(adj3, 0, maxAdj3)
	ah := ss * a3 / 100000
	dx := 0.0
	if h != 0 {
		dx = math.Sqrt(math.Max(h*h-ah*ah, 0)) * wR / h
	}
	x3 := wR - th
	x5 := wR - dx
	x7 := x3 - dx
	dh := (aw - th) / 2
	x4 := x5 - dh
	x8 := x7 + dh
	x6 := w - aw/2
	swAng := ooxmlAt2(ah, dx)
	dang2 := ooxmlAt2(idy, th/2)
	g := curvedArrowGuides{
		W:      w,
		H:      h,
		Th:     th,
		WR:     wR,
		X3:     x3,
		X4:     x4,
		X5:     x5,
		X6:     x6,
		X7:     x7,
		X8:     x8,
		IX:     (wR + x3) / 2,
		SwAng:  swAng,
		MSwAng: -swAng,
	}
	if element.PrstGeom == "curvedUpArrow" {
		g.Y1 = ah
		g.IY = idy
		g.SwAng2 = dang2 - swAng
		g.StAng3 = 5400000 - swAng
		g.SwAng3 = swAng - dang2
		g.StAng2 = 5400000 - dang2
	} else {
		g.Y1 = h - ah
		g.IY = h - idy
		g.StAng = 16200000 - swAng
		g.StAng2 = 16200000 - dang2
		g.SwAng2 = dang2 - 5400000
		g.SwAng3 = 5400000 - dang2
	}
	return g
}

func appendOoxmlArcPoints(points []pathPoint, radiusX float64, radiusY float64, startAngle float64, sweepAngle float64) []pathPoint {
	if len(points) == 0 || radiusX <= 0 || radiusY <= 0 || sweepAngle == 0 {
		return points
	}
	current := points[len(points)-1]
	ooStart := startAngle / 60000
	ooExtent := sweepAngle / 60000
	awtStart := convertOoxmlToAwtAngle(ooStart, radiusX, radiusY)
	awtSweep := convertOoxmlToAwtAngle(ooStart+ooExtent, radiusX, radiusY) - awtStart
	radStart := ooStart * math.Pi / 180
	invStart := math.Atan2(radiusX*math.Sin(radStart), radiusY*math.Cos(radStart))
	centerX := current.X - radiusX*math.Cos(invStart)
	centerY := current.Y - radiusY*math.Sin(invStart)
	segments := maxInt(4, int(math.Ceil(math.Abs(awtSweep)/6)))
	for step := 1; step <= segments; step++ {
		theta := (awtStart + awtSweep*float64(step)/float64(segments)) * math.Pi / 180
		points = append(points, pathPoint{
			X: centerX + radiusX*math.Cos(theta),
			Y: centerY - radiusY*math.Sin(theta),
		})
	}
	return points
}

func appendCurvedArrowArcPoints(points []pathPoint, radiusX float64, radiusY float64, startAngle float64, sweepAngle float64, upperMainPath bool) []pathPoint {
	if len(points) == 0 || radiusX <= 0 || radiusY <= 0 || sweepAngle == 0 {
		return points
	}
	current := points[len(points)-1]
	start := startAngle / 60000 * math.Pi / 180
	sweep := sweepAngle / 60000 * math.Pi / 180
	centerYSign := -1.0
	pointYSign := 1.0
	if upperMainPath {
		centerYSign = 1
		pointYSign = -1
	}
	centerX := current.X - radiusX*math.Cos(start)
	centerY := current.Y + centerYSign*radiusY*math.Sin(start)
	segments := maxInt(4, int(math.Ceil(math.Abs(sweepAngle/60000)/6)))
	for step := 1; step <= segments; step++ {
		theta := start + sweep*float64(step)/float64(segments)
		points = append(points, pathPoint{
			X: centerX + radiusX*math.Cos(theta),
			Y: centerY + pointYSign*radiusY*math.Sin(theta),
		})
	}
	return points
}

func convertOoxmlToAwtAngle(angle float64, width float64, height float64) float64 {
	aspect := height / width
	awtAngle := -angle
	angleRemainder := math.Mod(awtAngle, 360)
	angleBase := awtAngle - angleRemainder
	switch int(angleRemainder / 90) {
	case -3:
		angleBase -= 360
		angleRemainder += 360
	case -2, -1:
		angleBase -= 180
		angleRemainder += 180
	case 1, 2:
		angleBase += 180
		angleRemainder -= 180
	case 3:
		angleBase += 360
		angleRemainder -= 360
	}
	return math.Atan2(math.Tan(angleRemainder*math.Pi/180), aspect)*180/math.Pi + angleBase
}

func ooxmlAt2(x float64, y float64) float64 {
	return math.Atan2(x, y) * 180 / math.Pi * 60000
}

func normalizedGeometryPathPoints(points []pathPoint, width float64, height float64) []pathPoint {
	normalized := make([]pathPoint, 0, len(points))
	for _, point := range points {
		normalized = append(normalized, pathPoint{
			X: point.X / width,
			Y: point.Y / height,
		})
	}
	return normalized
}

func darkenLessPathFill() color.RGBA {
	return color.RGBA{A: 0x32}
}

func rightBracePresetPath(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj2 := presetAdjustment(element, "adj2", 50000)
	a2 := clampFloat(adj2, 0, 100000)
	q1 := 100000 - a2
	q2 := math.Min(q1, a2)
	maxAdj1 := q2 / 2 * h / ss
	adj1 := presetAdjustment(element, "adj1", 8333)
	a1 := clampFloat(adj1, 0, maxAdj1)
	y1 := ss * a1 / 100000
	y3 := h * a2 / 100000
	y2 := y3 - y1
	y4 := h - y1
	wd2 := w / 2
	hc := w / 2

	points := []pathPoint{{X: 0, Y: 0}}
	points = appendOoxmlArcPoints(points, wd2, y1, 16200000, 5400000)
	points = append(points, pathPoint{X: hc, Y: y2})
	points = appendOoxmlArcPoints(points, wd2, y1, 10800000, -5400000)
	points = appendOoxmlArcPoints(points, wd2, y1, 16200000, -5400000)
	points = append(points, pathPoint{X: hc, Y: y4})
	points = appendOoxmlArcPoints(points, wd2, y1, 0, 5400000)
	return transformedPathPoints(normalizedGeometryPathPoints(points, w, h), element)
}

func appendCubicPathPoints(points []pathPoint, start pathPoint, c1 pathPoint, c2 pathPoint, end pathPoint) []pathPoint {
	for step := 1; step <= customBezierSegments; step++ {
		t := float64(step) / customBezierSegments
		points = append(points, cubicBezierPoint(start, c1, c2, end, t))
	}
	return points
}

func drawOpenPathOutline(img *image.RGBA, bounds image.Rectangle, points []pathPoint, c color.RGBA, width int) {
	if len(points) < 2 || bounds.Empty() {
		return
	}
	previous := scaledPathPoint(bounds, points[0])
	for _, point := range points[1:] {
		current := scaledPathPoint(bounds, point)
		drawLine(img, previous.X, previous.Y, current.X, current.Y, c, width)
		previous = current
	}
}

func scaledPathPoint(bounds image.Rectangle, point pathPoint) image.Point {
	return image.Point{
		X: bounds.Min.X + int(math.Round(point.X*float64(bounds.Dx()))),
		Y: bounds.Min.Y + int(math.Round(point.Y*float64(bounds.Dy()))),
	}
}

func pointInPolygon(x int, y int, polygon []image.Point) bool {
	inside := false
	j := len(polygon) - 1
	for i := range polygon {
		yi := polygon[i].Y
		yj := polygon[j].Y
		if (yi > y) != (yj > y) {
			xIntersect := float64(polygon[j].X-polygon[i].X)*float64(y-yi)/float64(yj-yi) + float64(polygon[i].X)
			if float64(x) < xIntersect {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

func pointInPolygonFloat(x float64, y float64, polygon []image.Point) bool {
	inside := false
	j := len(polygon) - 1
	for i := range polygon {
		yi := float64(polygon[i].Y)
		yj := float64(polygon[j].Y)
		if (yi > y) != (yj > y) {
			xIntersect := float64(polygon[j].X-polygon[i].X)*(y-yi)/(yj-yi) + float64(polygon[i].X)
			if x < xIntersect {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

func pointInPolygonFloatPoints(x float64, y float64, polygon []floatPoint) bool {
	inside := false
	j := len(polygon) - 1
	for i := range polygon {
		yi := polygon[i].Y
		yj := polygon[j].Y
		if (yi > y) != (yj > y) {
			xIntersect := (polygon[j].X-polygon[i].X)*(y-yi)/(yj-yi) + polygon[i].X
			if x < xIntersect {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

func emuLineWidthToPixels(widthEMU int64, slideCX int64, outputWidth int) int {
	if widthEMU <= 0 {
		return 1
	}
	pixels := scaleEMU(widthEMU, slideCX, outputWidth)
	if pixels < 1 {
		return 1
	}
	return pixels
}

func drawRectOutline(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int) {
	if rect.Empty() {
		return
	}
	for i := 0; i < width; i++ {
		drawLine(img, rect.Min.X, rect.Min.Y+i, rect.Max.X-1, rect.Min.Y+i, c, 1)
		drawLine(img, rect.Min.X, rect.Max.Y-1-i, rect.Max.X-1, rect.Max.Y-1-i, c, 1)
		drawLine(img, rect.Min.X+i, rect.Min.Y, rect.Min.X+i, rect.Max.Y-1, c, 1)
		drawLine(img, rect.Max.X-1-i, rect.Min.Y, rect.Max.X-1-i, rect.Max.Y-1, c, 1)
	}
}

func drawStyledRectOutline(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int, dash string) {
	drawStyledRectOutlineAligned(img, rect, c, width, dash, "")
}

func drawStyledRectOutlineAligned(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int, dash string, align string) {
	drawStyledRectOutlineAlignedWithCap(img, rect, c, width, dash, align, "")
}

func drawStyledRectOutlineAlignedWithCap(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int, dash string, align string, cap string) {
	drawStyledRectOutlineCompound(img, rect, c, width, dash, align, cap, "", "")
}

func drawStyledRectOutlineCompound(img *image.RGBA, rect image.Rectangle, c color.RGBA, width int, dash string, align string, cap string, join string, compound string) {
	rect = alignedStrokeRect(rect, width, align)
	if dash == "" {
		if join != "round" && (compound == "" || compound == "sng") {
			drawRectOutline(img, rect, c, width)
			return
		}
		drawPathOutlineStyled(img, rect, []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, c, width, dash, cap, join, compound, true)
		return
	}
	if rect.Empty() {
		return
	}
	if join == "round" || (compound != "" && compound != "sng") {
		drawPathOutlineStyled(img, rect, []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, c, width, dash, cap, join, compound, true)
		return
	}
	for i := 0; i < width; i++ {
		drawStyledLineWithPatternWidth(img, rect.Min.X, rect.Min.Y+i, rect.Max.X-1, rect.Min.Y+i, c, 1, dash, cap, width)
		drawStyledLineWithPatternWidth(img, rect.Min.X, rect.Max.Y-1-i, rect.Max.X-1, rect.Max.Y-1-i, c, 1, dash, cap, width)
		drawStyledLineWithPatternWidth(img, rect.Min.X+i, rect.Min.Y, rect.Min.X+i, rect.Max.Y-1, c, 1, dash, cap, width)
		drawStyledLineWithPatternWidth(img, rect.Max.X-1-i, rect.Min.Y, rect.Max.X-1-i, rect.Max.Y-1, c, 1, dash, cap, width)
	}
}

func alignedStrokeRect(rect image.Rectangle, width int, align string) image.Rectangle {
	if rect.Empty() || width <= 1 {
		return rect
	}
	switch align {
	case "ctr":
		return rect.Inset(-(width / 2))
	case "out":
		return rect.Inset(-(width - 1))
	default:
		return rect
	}
}

func drawSoftRect(img *image.RGBA, rect image.Rectangle, c color.RGBA, blur int) {
	if rect.Empty() || c.A == 0 {
		return
	}
	drawBlurredShadowMask(img, rect, c, blur, func(x int, y int) bool {
		return image.Pt(x, y).In(rect)
	})
}

func drawBlurredShadowMask(img *image.RGBA, shapeBounds image.Rectangle, c color.RGBA, blur int, covers func(x int, y int) bool) {
	if shapeBounds.Empty() || c.A == 0 {
		return
	}
	if blur <= 0 {
		for y := shapeBounds.Min.Y; y < shapeBounds.Max.Y; y++ {
			for x := shapeBounds.Min.X; x < shapeBounds.Max.X; x++ {
				if image.Pt(x, y).In(img.Bounds()) && covers(x, y) {
					blendPixel(img, x, y, c)
				}
			}
		}
		return
	}
	maskBounds := shapeBounds.Inset(-blur).Intersect(img.Bounds())
	if maskBounds.Empty() {
		return
	}
	width := maskBounds.Dx()
	height := maskBounds.Dy()
	mask := make([]uint8, width*height)
	for y := maskBounds.Min.Y; y < maskBounds.Max.Y; y++ {
		for x := maskBounds.Min.X; x < maskBounds.Max.X; x++ {
			if covers(x, y) {
				mask[(y-maskBounds.Min.Y)*width+x-maskBounds.Min.X] = c.A
			}
		}
	}
	blurred := gaussianBlurAlpha(mask, width, height, blur)
	for y := maskBounds.Min.Y; y < maskBounds.Max.Y; y++ {
		for x := maskBounds.Min.X; x < maskBounds.Max.X; x++ {
			alpha := blurred[(y-maskBounds.Min.Y)*width+x-maskBounds.Min.X]
			if alpha == 0 {
				continue
			}
			blendPixel(img, x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: alpha})
		}
	}
}

func gaussianBlurAlpha(src []uint8, width int, height int, radius int) []uint8 {
	if radius <= 0 || width <= 0 || height <= 0 {
		dst := make([]uint8, len(src))
		copy(dst, src)
		return dst
	}
	kernel := gaussianKernel(radius)
	tmp := make([]float64, len(src))
	dstFloat := make([]float64, len(src))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0.0
			for offset := -radius; offset <= radius; offset++ {
				sampleX := x + offset
				if sampleX < 0 {
					sampleX = 0
				} else if sampleX >= width {
					sampleX = width - 1
				}
				sum += float64(src[y*width+sampleX]) * kernel[offset+radius]
			}
			tmp[y*width+x] = sum
		}
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0.0
			for offset := -radius; offset <= radius; offset++ {
				sampleY := y + offset
				if sampleY < 0 {
					sampleY = 0
				} else if sampleY >= height {
					sampleY = height - 1
				}
				sum += tmp[sampleY*width+x] * kernel[offset+radius]
			}
			dstFloat[y*width+x] = sum
		}
	}
	dst := make([]uint8, len(src))
	for index, value := range dstFloat {
		if value <= 0 {
			continue
		}
		if value >= 255 {
			dst[index] = 255
			continue
		}
		dst[index] = uint8(math.Round(value))
	}
	return dst
}

func gaussianBlurRGBA(src *image.RGBA, radius int) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	if radius <= 0 || width <= 0 || height <= 0 {
		draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
		return dst
	}
	alpha := make([]float64, width*height)
	redAlpha := make([]float64, width*height)
	greenAlpha := make([]float64, width*height)
	blueAlpha := make([]float64, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := src.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			index := y*width + x
			a := float64(pixel.A)
			alpha[index] = a
			redAlpha[index] = float64(pixel.R) * a
			greenAlpha[index] = float64(pixel.G) * a
			blueAlpha[index] = float64(pixel.B) * a
		}
	}
	alpha = gaussianBlurFloatPlane(alpha, width, height, radius)
	redAlpha = gaussianBlurFloatPlane(redAlpha, width, height, radius)
	greenAlpha = gaussianBlurFloatPlane(greenAlpha, width, height, radius)
	blueAlpha = gaussianBlurFloatPlane(blueAlpha, width, height, radius)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			a := alpha[index]
			if a <= 0 {
				continue
			}
			dst.SetRGBA(x, y, color.RGBA{
				R: clampColor(int64(math.Round(redAlpha[index] / a))),
				G: clampColor(int64(math.Round(greenAlpha[index] / a))),
				B: clampColor(int64(math.Round(blueAlpha[index] / a))),
				A: clampColor(int64(math.Round(a))),
			})
		}
	}
	return dst
}

func gaussianBlurFloatPlane(src []float64, width int, height int, radius int) []float64 {
	if radius <= 0 || width <= 0 || height <= 0 {
		dst := make([]float64, len(src))
		copy(dst, src)
		return dst
	}
	kernel := gaussianKernel(radius)
	tmp := make([]float64, len(src))
	dst := make([]float64, len(src))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0.0
			for offset := -radius; offset <= radius; offset++ {
				sampleX := x + offset
				if sampleX < 0 {
					sampleX = 0
				} else if sampleX >= width {
					sampleX = width - 1
				}
				sum += src[y*width+sampleX] * kernel[offset+radius]
			}
			tmp[y*width+x] = sum
		}
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0.0
			for offset := -radius; offset <= radius; offset++ {
				sampleY := y + offset
				if sampleY < 0 {
					sampleY = 0
				} else if sampleY >= height {
					sampleY = height - 1
				}
				sum += tmp[sampleY*width+x] * kernel[offset+radius]
			}
			dst[y*width+x] = sum
		}
	}
	return dst
}

func gaussianKernel(radius int) []float64 {
	if radius <= 0 {
		return []float64{1}
	}
	sigma := float64(radius) / 2
	if sigma < 0.5 {
		sigma = 0.5
	}
	kernel := make([]float64, radius*2+1)
	sum := 0.0
	denominator := 2 * sigma * sigma
	for offset := -radius; offset <= radius; offset++ {
		value := math.Exp(-float64(offset*offset) / denominator)
		kernel[offset+radius] = value
		sum += value
	}
	if sum == 0 {
		kernel[radius] = 1
		return kernel
	}
	for index := range kernel {
		kernel[index] /= sum
	}
	return kernel
}

func boxBlurAlpha(src []uint8, width int, height int, radius int) []uint8 {
	if radius <= 0 || width <= 0 || height <= 0 {
		dst := make([]uint8, len(src))
		copy(dst, src)
		return dst
	}
	tmp := make([]uint8, len(src))
	dst := make([]uint8, len(src))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0
			count := 0
			for xx := maxInt(0, x-radius); xx <= minInt(width-1, x+radius); xx++ {
				sum += int(src[y*width+xx])
				count++
			}
			tmp[y*width+x] = uint8(sum / count)
		}
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sum := 0
			count := 0
			for yy := maxInt(0, y-radius); yy <= minInt(height-1, y+radius); yy++ {
				sum += int(tmp[yy*width+x])
				count++
			}
			dst[y*width+x] = uint8(sum / count)
		}
	}
	return dst
}

func fillBlendRect(img *image.RGBA, rect image.Rectangle, c color.RGBA) {
	if rect.Empty() || c.A == 0 {
		return
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			blendPixel(img, x, y, c)
		}
	}
}

func blendPixel(img *image.RGBA, x int, y int, src color.RGBA) {
	if src.A == 0 {
		return
	}
	if src.A == 255 {
		img.SetRGBA(x, y, src)
		return
	}
	dst := img.RGBAAt(x, y)
	alpha := int(src.A)
	invAlpha := 255 - alpha
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((int(src.R)*alpha + int(dst.R)*invAlpha + 127) / 255),
		G: uint8((int(src.G)*alpha + int(dst.G)*invAlpha + 127) / 255),
		B: uint8((int(src.B)*alpha + int(dst.B)*invAlpha + 127) / 255),
		A: uint8(alpha + (int(dst.A)*invAlpha+127)/255),
	})
}

func applyDisplayP3OutputTransform(img *image.RGBA) {
	if img == nil {
		return
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := img.PixOffset(bounds.Min.X, y)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b := srgbToDisplayP3(img.Pix[offset], img.Pix[offset+1], img.Pix[offset+2])
			img.Pix[offset] = r
			img.Pix[offset+1] = g
			img.Pix[offset+2] = b
			offset += 4
		}
	}
}

func srgbToDisplayP3(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := srgbByteToLinear(r)
	linearG := srgbByteToLinear(g)
	linearB := srgbByteToLinear(b)

	// D65 sRGB linear RGB to D65 Display P3 linear RGB.
	p3R := 0.822461969*linearR + 0.177538031*linearG
	p3G := 0.033194199*linearR + 0.966805801*linearG
	p3B := 0.017082631*linearR + 0.072397440*linearG + 0.910519929*linearB
	return linearToSRGBByte(p3R), linearToSRGBByte(p3G), linearToSRGBByte(p3B)
}

func srgbByteToLinear(value uint8) float64 {
	encoded := float64(value) / 255
	if encoded <= 0.04045 {
		return encoded / 12.92
	}
	return math.Pow((encoded+0.055)/1.055, 2.4)
}

func linearToSRGBByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	var encoded float64
	if value <= 0.0031308 {
		encoded = 12.92 * value
	} else {
		encoded = 1.055*math.Pow(value, 1.0/2.4) - 0.055
	}
	return uint8(math.Round(encoded * 255))
}

func drawLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int) {
	if width < 1 {
		width = 1
	}
	dx := absInt(x1 - x0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	dy := -absInt(y1 - y0)
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		setThickPixel(img, x0, y0, c, width)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawStyledLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string) {
	drawStyledLineWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, cap, width)
}

func drawStyledCompoundLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string, compound string) {
	if compound == "" || compound == "sng" {
		drawStyledLine(img, x0, y0, x1, y1, c, width, dash, cap)
		return
	}
	for _, segment := range compoundLineSegments(width, compound) {
		ox0, oy0, ox1, oy1 := offsetLineEndpoints(x0, y0, x1, y1, segment.offset)
		drawStyledLineWithPatternWidth(img, ox0, oy0, ox1, oy1, c, segment.width, dash, cap, width)
	}
}

type compoundLineSegment struct {
	offset int
	width  int
}

func compoundLineSegments(width int, compound string) []compoundLineSegment {
	if width < 1 {
		width = 1
	}
	switch compound {
	case "dbl":
		strokeWidth, firstOffset, secondOffset := doubleCompoundLineMetrics(width)
		return []compoundLineSegment{{offset: firstOffset, width: strokeWidth}, {offset: secondOffset, width: strokeWidth}}
	case "thickThin":
		thick := maxInt(1, width/2)
		thin := maxInt(1, width/4)
		return []compoundLineSegment{{offset: -maxInt(1, width/4), width: thick}, {offset: maxInt(1, width/3), width: thin}}
	case "thinThick":
		thick := maxInt(1, width/2)
		thin := maxInt(1, width/4)
		return []compoundLineSegment{{offset: -maxInt(1, width/3), width: thin}, {offset: maxInt(1, width/4), width: thick}}
	case "tri":
		stroke := maxInt(1, width/4)
		outer := maxInt(1, width/2)
		return []compoundLineSegment{{offset: -outer, width: stroke}, {offset: 0, width: stroke}, {offset: outer, width: stroke}}
	default:
		return []compoundLineSegment{{width: width}}
	}
}

func offsetLineEndpoints(x0 int, y0 int, x1 int, y1 int, offset int) (int, int, int, int) {
	if offset == 0 {
		return x0, y0, x1, y1
	}
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return x0, y0, x1, y1
	}
	perpX := -dy / length
	perpY := dx / length
	ox := int(math.Round(perpX * float64(offset)))
	oy := int(math.Round(perpY * float64(offset)))
	return x0 + ox, y0 + oy, x1 + ox, y1 + oy
}

func drawStyledLineWithPatternWidth(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string, patternWidth int) {
	if cap == "" || cap == "sq" {
		if dash == "" {
			drawLine(img, x0, y0, x1, y1, c, width)
			return
		}
		drawDashedLineWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, "sq", patternWidth)
		return
	}
	if dash == "" {
		drawLineWithCap(img, x0, y0, x1, y1, c, width, cap)
		return
	}
	drawDashedLineWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, cap, patternWidth)
}

func drawDashedLineLegacy(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string) {
	drawDashedLineLegacyWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, width)
}

func drawDashedLineLegacyWithPatternWidth(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, patternWidth int) {
	if width < 1 {
		width = 1
	}
	length := math.Hypot(float64(x1-x0), float64(y1-y0))
	if length == 0 {
		setThickPixel(img, x0, y0, c, width)
		return
	}
	pattern := lineDashPatternPixels(dash, patternWidth)
	if len(pattern) == 0 {
		drawLine(img, x0, y0, x1, y1, c, width)
		return
	}
	patternIndex := 0
	patternRemaining := float64(pattern[0])
	drawSegment := true
	for distance := 0.0; distance <= length; distance++ {
		if drawSegment {
			t := distance / length
			x := int(math.Round(float64(x0) + float64(x1-x0)*t))
			y := int(math.Round(float64(y0) + float64(y1-y0)*t))
			setThickPixel(img, x, y, c, width)
		}
		patternRemaining--
		if patternRemaining <= 0 {
			patternIndex = (patternIndex + 1) % len(pattern)
			patternRemaining = float64(pattern[patternIndex])
			drawSegment = patternIndex%2 == 0
		}
	}
}

func drawDashedLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string) {
	drawDashedLineWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, cap, width)
}

func drawDashedLineWithPatternWidth(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string, patternWidth int) {
	if width < 1 {
		width = 1
	}
	length := math.Hypot(float64(x1-x0), float64(y1-y0))
	if length == 0 {
		drawLineWithCap(img, x0, y0, x1, y1, c, width, cap)
		return
	}
	pattern := lineDashPatternPixels(dash, patternWidth)
	if len(pattern) == 0 {
		drawLineWithCap(img, x0, y0, x1, y1, c, width, cap)
		return
	}
	patternIndex := 0
	position := 0.0
	drawSegment := true
	for position < length {
		next := math.Min(position+float64(pattern[patternIndex]), length)
		if drawSegment {
			drawLineDistanceSegment(img, x0, y0, x1, y1, length, position, next, c, width, cap)
		}
		position = next
		patternIndex = (patternIndex + 1) % len(pattern)
		drawSegment = patternIndex%2 == 0
	}
}

func drawLineDistanceSegment(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, length float64, start float64, end float64, c color.RGBA, width int, cap string) {
	if end <= start || length <= 0 {
		return
	}
	t0 := start / length
	t1 := end / length
	sx := int(math.Round(float64(x0) + float64(x1-x0)*t0))
	sy := int(math.Round(float64(y0) + float64(y1-y0)*t0))
	ex := int(math.Round(float64(x0) + float64(x1-x0)*t1))
	ey := int(math.Round(float64(y0) + float64(y1-y0)*t1))
	drawLineWithCap(img, sx, sy, ex, ey, c, width, cap)
}

func drawLineWithCap(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, cap string) {
	if c.A == 0 {
		return
	}
	if width < 1 {
		width = 1
	}
	mode := normalizeLineCap(cap)
	length := math.Hypot(float64(x1-x0), float64(y1-y0))
	if length == 0 {
		drawPointCap(img, x0, y0, c, width, mode)
		return
	}
	radius := float64(width) / 2
	padding := int(math.Ceil(radius)) + 2
	bounds := image.Rect(
		minInt(x0, x1)-padding,
		minInt(y0, y1)-padding,
		maxInt(x0, x1)+padding+1,
		maxInt(y0, y1)+padding+1,
	).Intersect(img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := lineStrokeCoverage(float64(x), float64(y), float64(x0), float64(y0), float64(x1), float64(y1), radius, mode)
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else if coverage > 0 {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func drawPointCap(img *image.RGBA, x int, y int, c color.RGBA, width int, cap string) {
	if cap == "rnd" {
		drawLineWithCap(img, x, y, x+1, y, c, width, cap)
		return
	}
	setThickPixel(img, x, y, c, width)
}

func normalizeLineCap(cap string) string {
	switch cap {
	case "flat", "rnd", "sq":
		return cap
	default:
		return "sq"
	}
}

func lineStrokeCoverage(x float64, y float64, x0 float64, y0 float64, x1 float64, y1 float64, radius float64, cap string) int {
	coverage := 0
	for _, offset := range []struct {
		x float64
		y float64
	}{
		{x: -0.25, y: -0.25},
		{x: 0.25, y: -0.25},
		{x: -0.25, y: 0.25},
		{x: 0.25, y: 0.25},
	} {
		if pointInLineStroke(x+offset.x, y+offset.y, x0, y0, x1, y1, radius, cap) {
			coverage++
		}
	}
	return coverage
}

func pointInLineStroke(x float64, y float64, x0 float64, y0 float64, x1 float64, y1 float64, radius float64, cap string) bool {
	dx := x1 - x0
	dy := y1 - y0
	length := math.Hypot(dx, dy)
	if length == 0 {
		return math.Hypot(x-x0, y-y0) <= radius
	}
	ux := dx / length
	uy := dy / length
	vx := x - x0
	vy := y - y0
	projection := vx*ux + vy*uy
	perpendicular := math.Abs(vx*(-uy) + vy*ux)
	switch cap {
	case "flat":
		return projection >= 0 && projection <= length && perpendicular <= radius
	case "rnd":
		if projection >= 0 && projection <= length && perpendicular <= radius {
			return true
		}
		return math.Hypot(x-x0, y-y0) <= radius || math.Hypot(x-x1, y-y1) <= radius
	default:
		return projection >= -radius && projection <= length+radius && perpendicular <= radius
	}
}

func lineDashPatternPixels(dash string, width int) []int {
	if strings.HasPrefix(dash, "cust:") {
		return customLineDashPatternPixels(strings.TrimPrefix(dash, "cust:"), width)
	}
	patterns := map[string]string{
		"dash":          "1111000",
		"dashDot":       "11110001000",
		"dot":           "1000",
		"lgDash":        "11111111000",
		"lgDashDot":     "111111110001000",
		"lgDashDotDot":  "1111111100010001000",
		"sysDash":       "1110",
		"sysDashDot":    "111010",
		"sysDashDotDot": "11101010",
		"sysDot":        "10",
	}
	bits, ok := patterns[dash]
	if !ok {
		bits = patterns["dash"]
	}
	return binaryDashPatternPixels(bits, width)
}

func customLineDashPatternPixels(encoded string, width int) []int {
	if encoded == "" {
		return nil
	}
	unit := maxInt(width, 1)
	var pattern []int
	for _, part := range strings.Split(encoded, ",") {
		pieces := strings.Split(part, "/")
		if len(pieces) != 2 {
			continue
		}
		d, errD := strconv.ParseInt(pieces[0], 10, 64)
		sp, errSp := strconv.ParseInt(pieces[1], 10, 64)
		if errD != nil || errSp != nil {
			continue
		}
		pattern = append(pattern, maxInt(1, int(math.Round(float64(unit)*float64(d)/100000))))
		pattern = append(pattern, maxInt(1, int(math.Round(float64(unit)*float64(sp)/100000))))
	}
	if len(pattern) == 0 {
		return nil
	}
	return pattern
}

func binaryDashPatternPixels(bits string, width int) []int {
	if bits == "" {
		return nil
	}
	unit := maxInt(width, 1)
	pattern := make([]int, 0, len(bits))
	run := 1
	for i := 1; i < len(bits); i++ {
		if bits[i] == bits[i-1] {
			run++
			continue
		}
		pattern = append(pattern, maxInt(unit*run, run))
		run = 1
	}
	pattern = append(pattern, maxInt(unit*run, run))
	return pattern
}

func drawLineTriangleMarker(img *image.RGBA, tipX int, tipY int, dirX int, dirY int, c color.RGBA, lineWidth int, markerWidth string, markerLength string) {
	drawLineEndMarker(img, "triangle", tipX, tipY, dirX, dirY, c, lineWidth, markerWidth, markerLength)
}

func drawLineEndMarker(img *image.RGBA, markerType string, tipX int, tipY int, dirX int, dirY int, c color.RGBA, lineWidth int, markerWidth string, markerLength string) bool {
	length := math.Hypot(float64(dirX), float64(dirY))
	if length == 0 {
		return true
	}
	unitX := float64(dirX) / length
	unitY := float64(dirY) / length
	drawLength := math.Max(float64(lineWidth)*lineEndLengthFactor(markerLength), 8)
	markerHalfWidth := math.Max(float64(lineWidth)*lineEndWidthFactor(markerWidth), 4)
	baseX := float64(tipX) - unitX*drawLength
	baseY := float64(tipY) - unitY*drawLength
	perpX := -unitY
	perpY := unitX
	switch markerType {
	case "triangle":
		drawFilledPolygon(img, []image.Point{
			{X: tipX, Y: tipY},
			{X: int(math.Round(baseX + perpX*markerHalfWidth)), Y: int(math.Round(baseY + perpY*markerHalfWidth))},
			{X: int(math.Round(baseX - perpX*markerHalfWidth)), Y: int(math.Round(baseY - perpY*markerHalfWidth))},
		}, c)
	case "stealth":
		innerX := float64(tipX) - unitX*drawLength*0.62
		innerY := float64(tipY) - unitY*drawLength*0.62
		drawFilledPolygon(img, []image.Point{
			{X: tipX, Y: tipY},
			{X: int(math.Round(baseX + perpX*markerHalfWidth)), Y: int(math.Round(baseY + perpY*markerHalfWidth))},
			{X: int(math.Round(innerX)), Y: int(math.Round(innerY))},
			{X: int(math.Round(baseX - perpX*markerHalfWidth)), Y: int(math.Round(baseY - perpY*markerHalfWidth))},
		}, c)
	case "diamond":
		midX := float64(tipX) - unitX*drawLength/2
		midY := float64(tipY) - unitY*drawLength/2
		backX := float64(tipX) - unitX*drawLength
		backY := float64(tipY) - unitY*drawLength
		drawFilledPolygon(img, []image.Point{
			{X: tipX, Y: tipY},
			{X: int(math.Round(midX + perpX*markerHalfWidth)), Y: int(math.Round(midY + perpY*markerHalfWidth))},
			{X: int(math.Round(backX)), Y: int(math.Round(backY))},
			{X: int(math.Round(midX - perpX*markerHalfWidth)), Y: int(math.Round(midY - perpY*markerHalfWidth))},
		}, c)
	case "oval":
		centerX := int(math.Round(float64(tipX) - unitX*drawLength/2))
		centerY := int(math.Round(float64(tipY) - unitY*drawLength/2))
		drawMarkerOval(img, centerX, centerY, int(math.Round(drawLength/2)), int(math.Round(markerHalfWidth)), c)
	case "arrow":
		leftX := int(math.Round(baseX + perpX*markerHalfWidth))
		leftY := int(math.Round(baseY + perpY*markerHalfWidth))
		rightX := int(math.Round(baseX - perpX*markerHalfWidth))
		rightY := int(math.Round(baseY - perpY*markerHalfWidth))
		drawStyledLine(img, tipX, tipY, leftX, leftY, c, lineWidth, "", "flat")
		drawStyledLine(img, tipX, tipY, rightX, rightY, c, lineWidth, "", "flat")
	default:
		return false
	}
	return true
}

func drawMarkerOval(img *image.RGBA, centerX int, centerY int, radiusX int, radiusY int, c color.RGBA) {
	if c.A == 0 || radiusX <= 0 || radiusY <= 0 {
		return
	}
	bounds := image.Rect(centerX-radiusX-1, centerY-radiusY-1, centerX+radiusX+2, centerY+radiusY+2).Intersect(img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			nx := float64(x-centerX) / float64(radiusX)
			ny := float64(y-centerY) / float64(radiusY)
			if nx*nx+ny*ny <= 1 {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func drawLegacyTriangleMarkerPolygon(img *image.RGBA, tipX int, tipY int, baseX float64, baseY float64, perpX float64, perpY float64, markerHalfWidth float64, c color.RGBA) {
	drawFilledPolygon(img, []image.Point{
		{X: tipX, Y: tipY},
		{X: int(math.Round(baseX + perpX*markerHalfWidth)), Y: int(math.Round(baseY + perpY*markerHalfWidth))},
		{X: int(math.Round(baseX - perpX*markerHalfWidth)), Y: int(math.Round(baseY - perpY*markerHalfWidth))},
	}, c)
}

func lineEndLengthFactor(value string) float64 {
	switch value {
	case "sm":
		return 3
	case "lg":
		return 5
	default:
		return 4
	}
}

func lineEndWidthFactor(value string) float64 {
	switch value {
	case "sm":
		return 1.1
	case "lg":
		return 2.4
	default:
		return 1.6
	}
}

func drawFilledPolygon(img *image.RGBA, polygon []image.Point, c color.RGBA) {
	if len(polygon) < 3 || c.A == 0 {
		return
	}
	minX, maxX := polygon[0].X, polygon[0].X
	minY, maxY := polygon[0].Y, polygon[0].Y
	for _, point := range polygon[1:] {
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	bounds := image.Rect(minX, minY, maxX+1, maxY+1).Intersect(img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := polygonCoverage(float64(x), float64(y), polygon)
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else if coverage > 0 {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func setThickPixel(img *image.RGBA, x int, y int, c color.RGBA, width int) {
	radius := width / 2
	for yy := y - radius; yy <= y+radius; yy++ {
		for xx := x - radius; xx <= x+radius; xx++ {
			if image.Pt(xx, yy).In(img.Bounds()) {
				blendPixel(img, xx, yy, c)
			}
		}
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
