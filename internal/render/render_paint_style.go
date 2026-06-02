package render

import (
	"image"
	"image/color"
	"image/draw"
)

func fillPaintFromContainer(container *xmlNode, theme themeColors, groupFill *backgroundPaint) (backgroundPaint, bool) {
	for _, child := range container.Children {
		switch child.Name {
		case "solidFill", "gradFill", "pattFill", "grpFill":
			return fillPaintFromNode(child, theme, groupFill)
		}
	}
	return backgroundPaint{}, false
}

func fillPaintFromNode(node *xmlNode, theme themeColors, groupFill *backgroundPaint) (backgroundPaint, bool) {
	switch node.Name {
	case "solidFill":
		if c, ok := colorFromSolidFillWithTheme(node, theme); ok {
			return backgroundPaint{Color: c}, true
		}
	case "gradFill":
		if gradient, ok := parseGradientFill(node, theme); ok {
			return backgroundPaint{Color: gradient.Stops[0].Color, HasGradient: true, Gradient: gradient}, true
		}
	case "pattFill":
		pattern := parsePatternFill(node, theme)
		return backgroundPaint{Color: pattern.Foreground, HasPattern: true, Pattern: pattern}, true
	case "grpFill":
		if groupFill != nil {
			return *groupFill, true
		}
	}
	return backgroundPaint{}, false
}

func drawPatternRect(img *image.RGBA, rect image.Rectangle, pattern patternPaint) {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if patternForegroundAt(pattern.Preset, x-rect.Min.X, y-rect.Min.Y) {
				blendPixel(img, x, y, pattern.Foreground)
			} else {
				blendPixel(img, x, y, pattern.Background)
			}
		}
	}
}

func applyFillOverlay(img *image.RGBA, bounds image.Rectangle, paint backgroundPaint, blend string) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() {
		return
	}
	overlay := fillOverlayPaintImage(bounds, paint)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			base := img.RGBAAt(x, y)
			if base.A == 0 {
				continue
			}
			over := overlay.RGBAAt(x-bounds.Min.X, y-bounds.Min.Y)
			if over.A == 0 {
				continue
			}
			img.SetRGBA(x, y, fillOverlayBlendPixel(base, over, blend))
		}
	}
}

func fillOverlayPaintImage(bounds image.Rectangle, paint backgroundPaint) *image.RGBA {
	overlay := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	switch {
	case paint.HasGradient:
		drawGradientRect(overlay, overlay.Bounds(), paint.Gradient, true)
	case paint.HasPattern:
		drawPatternRect(overlay, overlay.Bounds(), paint.Pattern)
	default:
		draw.Draw(overlay, overlay.Bounds(), &image.Uniform{C: paint.Color}, image.Point{}, draw.Src)
	}
	return overlay
}

func fillOverlayBlendPixel(base color.RGBA, overlay color.RGBA, blend string) color.RGBA {
	alpha := int(overlay.A)
	invAlpha := 255 - alpha
	blended := color.RGBA{
		R: fillOverlayBlendChannel(base.R, overlay.R, blend),
		G: fillOverlayBlendChannel(base.G, overlay.G, blend),
		B: fillOverlayBlendChannel(base.B, overlay.B, blend),
		A: base.A,
	}
	if alpha >= 255 {
		return blended
	}
	return color.RGBA{
		R: uint8((int(blended.R)*alpha + int(base.R)*invAlpha + 127) / 255),
		G: uint8((int(blended.G)*alpha + int(base.G)*invAlpha + 127) / 255),
		B: uint8((int(blended.B)*alpha + int(base.B)*invAlpha + 127) / 255),
		A: base.A,
	}
}

func fillOverlayBlendChannel(base uint8, overlay uint8, blend string) uint8 {
	switch blend {
	case "mult":
		return uint8((int(base)*int(overlay) + 127) / 255)
	case "screen":
		return uint8(255 - ((255-int(base))*(255-int(overlay))+127)/255)
	case "darken":
		if overlay < base {
			return overlay
		}
		return base
	case "lighten":
		if overlay > base {
			return overlay
		}
		return base
	default:
		return overlay
	}
}

func drawPatternRoundRect(img *image.RGBA, bounds image.Rectangle, radius int, corners roundedCorners, pattern patternPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() {
		return
	}
	if radius <= 0 {
		drawPatternRect(img, bounds, pattern)
		return
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := roundRectCoverage(float64(x), float64(y), bounds, radius, corners)
			if coverage == 0 {
				continue
			}
			layer := patternColorAt(pattern, x-bounds.Min.X, y-bounds.Min.Y)
			if coverage == 4 && layer.A == 255 {
				img.SetRGBA(x, y, layer)
			} else {
				layer.A = coverageAlpha(layer.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func drawPatternEllipse(img *image.RGBA, bounds image.Rectangle, pattern patternPaint) {
	bounds = bounds.Intersect(img.Bounds())
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
				blendPixel(img, x, y, patternColorAt(pattern, x-bounds.Min.X, y-bounds.Min.Y))
			}
		}
	}
}

func drawPatternPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, pattern patternPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if len(points) < 3 || bounds.Empty() {
		return
	}
	polygon := polygonImagePoints(bounds, points)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := polygonCoverage(float64(x), float64(y), polygon)
			if coverage == 0 {
				continue
			}
			layer := patternColorAt(pattern, x-bounds.Min.X, y-bounds.Min.Y)
			if coverage == 4 && layer.A == 255 {
				img.SetRGBA(x, y, layer)
			} else {
				layer.A = coverageAlpha(layer.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func patternColorAt(pattern patternPaint, x int, y int) color.RGBA {
	if patternForegroundAt(pattern.Preset, x, y) {
		return pattern.Foreground
	}
	return pattern.Background
}

func patternForegroundAt(preset string, x int, y int) bool {
	switch preset {
	case "pct5":
		return (x+y*3)%20 == 0
	case "pct10":
		return (x+y*3)%10 == 0
	case "pct20":
		return (x+y)%5 == 0
	case "pct25":
		return (x+y)%4 == 0
	case "pct30":
		return (x+y*2)%10 < 3
	case "pct40":
		return (x+y)%5 < 2
	case "pct50":
		return (x+y)%2 == 0
	case "pct60":
		return (x+y)%5 < 3
	case "pct70":
		return (x+y*2)%10 < 7
	case "pct75":
		return (x+y)%4 != 0
	case "pct80":
		return (x+y)%5 != 0
	case "pct90":
		return (x+y*3)%10 != 0
	case "horz", "ltHorz":
		return y%4 == 0
	case "dkHorz":
		return y%4 <= 1
	case "vert", "ltVert":
		return x%4 == 0
	case "dkVert":
		return x%4 <= 1
	case "cross":
		return x%4 == 0 || y%4 == 0
	case "smGrid":
		return x%6 == 0 || y%6 == 0
	case "lgGrid":
		return x%10 == 0 || y%10 == 0
	case "dnDiag", "ltDnDiag":
		return (x-y)%6 == 0
	case "dkDnDiag":
		return modInt(x-y, 6) <= 1
	case "upDiag", "ltUpDiag":
		return (x+y)%6 == 0
	case "dkUpDiag":
		return (x+y)%6 <= 1
	case "diagCross":
		return (x+y)%6 == 0 || modInt(x-y, 6) == 0
	case "dotGrid":
		return x%6 == 0 && y%6 == 0
	case "dashHorz":
		return y%6 == 0 && x%8 < 4
	case "dashVert":
		return x%6 == 0 && y%8 < 4
	default:
		return (x+y)%2 == 0
	}
}

func modInt(value int, modulus int) int {
	result := value % modulus
	if result < 0 {
		result += modulus
	}
	return result
}

func parsePatternFill(node *xmlNode, theme themeColors) patternPaint {
	pattern := patternPaint{
		Preset:     attrValue(node.Attrs, "prst"),
		Foreground: color.RGBA{A: 255},
		Background: color.RGBA{R: 255, G: 255, B: 255, A: 255},
	}
	if pattern.Preset == "" {
		pattern.Preset = "pct5"
	}
	if fg := firstChild(node, "fgClr"); fg != nil {
		if c, ok := colorFromColorNodeWithTheme(fg, theme); ok {
			pattern.Foreground = c
		}
	}
	if bg := firstChild(node, "bgClr"); bg != nil {
		if c, ok := colorFromColorNodeWithTheme(bg, theme); ok {
			pattern.Background = c
		}
	}
	return pattern
}

func applyFillPaintToElement(element *slideElement, paint backgroundPaint) {
	element.HasFill = true
	element.FillColor = paint.Color
	if paint.HasGradient {
		element.HasFillGradient = true
		element.FillGradient = paint.Gradient
	}
	if paint.HasPattern {
		element.HasPatternFill = true
		element.PatternFill = paint.Pattern
		element.FillColor = paint.Pattern.Foreground
	}
	if len(paint.Unsupported) > 0 {
		element.PaintUnsupported = appendDistinctStrings(element.PaintUnsupported, paint.Unsupported...)
	}
}
