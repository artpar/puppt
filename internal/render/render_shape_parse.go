package render

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

func parseShapeProperties(spPr *xmlNode, transform renderTransform, element *slideElement, theme themeColors) {
	if bwMode := attrValue(spPr.Attrs, "bwMode"); bwMode != "" {
		element.BWMode = bwMode
	}
	if xfrm := firstChild(spPr, "xfrm"); xfrm != nil {
		parseTransform(xfrm, transform, element)
	}
	if prstGeom := firstChild(spPr, "prstGeom"); prstGeom != nil {
		element.PrstGeom = attrValue(prstGeom.Attrs, "prst")
		element.PrstGeomAdjustments = parsePresetGeometryAdjustments(prstGeom)
	}
	if custGeom := firstChild(spPr, "custGeom"); custGeom != nil {
		element.CustomPath, element.CustomPathCommands, element.CustomPathUnsupported = parseCustomGeometryPathCommandsWithDiagnostics(custGeom)
	}
	if firstChild(spPr, "noFill") != nil {
		element.NoFill = true
	}
	if solidFill := firstChild(spPr, "solidFill"); solidFill != nil {
		if fill, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			element.HasFill = true
			element.FillColor = fill
		}
	}
	if gradFill := firstChild(spPr, "gradFill"); gradFill != nil {
		if gradient, ok := parseGradientFill(gradFill, theme); ok {
			element.HasFill = true
			element.FillColor = gradient.Stops[0].Color
			element.HasFillGradient = true
			element.FillGradient = gradient
		}
	}
	if ln := firstChild(spPr, "ln"); ln != nil {
		if attrValue(ln.Attrs, "w") != "" {
			element.HasLineWidth = true
			element.LineWidth = parseIntAttr(ln.Attrs, "w")
		}
		if cap := attrValue(ln.Attrs, "cap"); cap != "" {
			element.HasLineCap = true
			element.LineCap = cap
		}
		if align := attrValue(ln.Attrs, "algn"); align != "" {
			element.HasLineAlign = true
			element.LineAlign = align
		}
		if element.LineWidth == 0 {
			element.LineWidth = 9525
		}
		if firstChild(ln, "noFill") != nil {
			element.NoLine = true
		} else if solidFill := firstChild(ln, "solidFill"); solidFill != nil {
			if lineColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				element.HasLine = true
				element.LineColor = lineColor
			}
		}
		if dash := firstChild(ln, "prstDash"); dash != nil {
			element.HasLineDash = true
			if value := attrValue(dash.Attrs, "val"); value != "" && value != "solid" {
				element.LineDash = value
			}
		}
		element.HeadLineMarker, element.HeadLineMarkerWidth, element.HeadLineMarkerLength = lineEndMarkerProperties(ln, "headEnd")
		element.TailLineMarker, element.TailLineMarkerWidth, element.TailLineMarkerLength = lineEndMarkerProperties(ln, "tailEnd")
		element.HasLineMarker = element.HeadLineMarker != "" || element.TailLineMarker != ""
	}
	if effectList := firstChild(spPr, "effectLst"); effectList != nil {
		element.HasEffectProperties = true
		parseShapeEffects(effectList, element, theme)
	} else if firstChild(spPr, "effectDag") != nil {
		element.HasEffectProperties = true
	}
	if sp3d := firstChild(spPr, "sp3d"); sp3d != nil {
		parseShape3DProperties(sp3d, element)
	}
}

func parseShape3DProperties(sp3d *xmlNode, element *slideElement) {
	features := visibleShape3DFeatures(sp3d)
	if len(features) == 0 {
		return
	}
	element.HasShape3D = true
	element.Shape3DFeatures = appendDistinctStrings(element.Shape3DFeatures, features...)
	element.HasEffectProperties = true
}

func visibleShape3DFeatures(sp3d *xmlNode) []string {
	if sp3d == nil {
		return nil
	}
	var features []string
	if parseIntAttr(sp3d.Attrs, "extrusionH") > 0 {
		features = append(features, "3-D extrusion")
	}
	if parseIntAttr(sp3d.Attrs, "contourW") > 0 {
		features = append(features, "3-D contour")
	}
	if bevelHasVisibleSize(firstChild(sp3d, "bevelT")) {
		features = append(features, "3-D top bevel")
	}
	if bevelHasVisibleSize(firstChild(sp3d, "bevelB")) {
		features = append(features, "3-D bottom bevel")
	}
	return features
}

func bevelHasVisibleSize(bevel *xmlNode) bool {
	if bevel == nil {
		return false
	}
	return parseIntAttr(bevel.Attrs, "w") > 0 || parseIntAttr(bevel.Attrs, "h") > 0
}

func appendDistinctStrings(base []string, values ...string) []string {
	seen := map[string]bool{}
	for _, value := range base {
		seen[value] = true
	}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		base = append(base, value)
		seen[value] = true
	}
	return base
}

func parseShapeEffects(effectList *xmlNode, element *slideElement, theme themeColors) {
	if softEdge := firstChild(effectList, "softEdge"); softEdge != nil {
		if radius := parseIntAttr(softEdge.Attrs, "rad"); radius > 0 {
			element.HasSoftEdge = true
			element.SoftEdgeRadius = radius
		}
	}
	shadow := firstChild(effectList, "outerShdw")
	if shadow == nil {
		return
	}
	element.HasShadow = true
	element.ShadowBlur = parseIntAttr(shadow.Attrs, "blurRad")
	element.ShadowDistance = parseIntAttr(shadow.Attrs, "dist")
	element.ShadowDirection = parseIntAttr(shadow.Attrs, "dir")
	element.ShadowAlignment = attrValue(shadow.Attrs, "algn")
	if value := attrValue(shadow.Attrs, "rotWithShape"); value != "" {
		element.HasShadowRotateWithShape = true
		element.ShadowRotateWithShape = boolAttrOn(value)
	}
	if value := attrValue(shadow.Attrs, "sx"); value != "" {
		element.HasShadowScaleX = true
		element.ShadowScaleX = parsePercentAttr(shadow.Attrs, "sx")
	}
	if value := attrValue(shadow.Attrs, "sy"); value != "" {
		element.HasShadowScaleY = true
		element.ShadowScaleY = parsePercentAttr(shadow.Attrs, "sy")
	}
	if value := attrValue(shadow.Attrs, "kx"); value != "" {
		element.HasShadowSkewX = true
		element.ShadowSkewX = parseIntAttr(shadow.Attrs, "kx")
	}
	if value := attrValue(shadow.Attrs, "ky"); value != "" {
		element.HasShadowSkewY = true
		element.ShadowSkewY = parseIntAttr(shadow.Attrs, "ky")
	}
	if shadowColor, ok := colorFromColorNodeWithTheme(shadow, theme); ok {
		element.ShadowColor = shadowColor
	} else {
		element.ShadowColor = color.RGBA{A: 96}
	}
}

func parsePresetGeometryAdjustments(prstGeom *xmlNode) map[string]int64 {
	avLst := firstChild(prstGeom, "avLst")
	if avLst == nil {
		return nil
	}
	adjustments := map[string]int64{}
	for _, gd := range childrenByName(avLst, "gd") {
		name := attrValue(gd.Attrs, "name")
		if name == "" {
			continue
		}
		fields := strings.Fields(attrValue(gd.Attrs, "fmla"))
		if len(fields) != 2 || fields[0] != "val" {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		adjustments[name] = value
	}
	if len(adjustments) == 0 {
		return nil
	}
	return adjustments
}

func parseStyleProperties(style *xmlNode, element *slideElement, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) {
	if fillRef := firstChild(style, "fillRef"); fillRef != nil && !element.HasFill && attrValue(fillRef.Attrs, "idx") != "0" {
		placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(fillRef, theme)
		if hasPlaceholderColor {
			if paint, ok := fillStyles.Style(parseIntAttr(fillRef.Attrs, "idx"), themeWithPlaceholderColor(theme, placeholderColor)); ok {
				applyStyleFillPaint(element, paint)
			}
		}
		if !element.HasFill && hasPlaceholderColor {
			element.HasFill = true
			element.FillColor = placeholderColor
		}
	}
	if lineRef := firstChild(style, "lnRef"); lineRef != nil && !element.HasLine && !element.NoLine {
		applyStyleLineReference(element, lineRef, theme, lineStyles)
	}
	if fontRef := firstChild(style, "fontRef"); fontRef != nil {
		if element.FontFamily == "" {
			element.FontFamily = fontRefTypeface(attrValue(fontRef.Attrs, "idx"))
		}
		if !element.HasTextColor {
			if textColor, ok := colorFromColorNodeWithTheme(fontRef, theme); ok {
				element.HasTextColor = true
				element.TextColor = textColor
			}
		}
	}
	if effectRef := firstChild(style, "effectRef"); effectRef != nil && !element.HasShadow && !element.HasEffectProperties {
		styleTheme := theme
		if placeholderColor, ok := colorFromColorNodeWithTheme(effectRef, theme); ok {
			styleTheme = themeWithPlaceholderColor(theme, placeholderColor)
		}
		if effects, ok := effectStyles.Style(parseIntAttr(effectRef.Attrs, "idx"), styleTheme); ok {
			applyThemeEffectStyle(element, effects)
		}
	}
}

func applyStyleLineReference(element *slideElement, lineRef *xmlNode, theme themeColors, lineStyles themeLineStyles) {
	if attrValue(lineRef.Attrs, "idx") == "0" {
		element.NoLine = true
		return
	}
	placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(lineRef, theme)
	styleTheme := theme
	if hasPlaceholderColor {
		styleTheme = themeWithPlaceholderColor(theme, placeholderColor)
	}
	if border, ok := lineStyles.Style(parseIntAttr(lineRef.Attrs, "idx"), styleTheme); ok {
		applyTableBorderToShapeLine(element, border)
		return
	}
	if hasPlaceholderColor {
		element.HasLine = true
		element.LineColor = placeholderColor
		if element.LineWidth == 0 {
			element.LineWidth = 9525
		}
	}
}

func applyTableBorderToShapeLine(element *slideElement, border tableCellBorder) {
	if border.NoLine {
		element.NoLine = true
		element.HasLine = false
		return
	}
	if !border.HasLine {
		return
	}
	element.HasLine = true
	element.LineColor = border.Color
	if !element.HasLineWidth && border.Width > 0 {
		element.LineWidth = border.Width
	}
	if !element.HasLineDash {
		element.LineDash = border.Dash
	}
	if !element.HasLineCap {
		element.LineCap = border.Cap
	}
	if !element.HasLineAlign {
		element.LineAlign = border.Align
	}
	if element.LineWidth == 0 {
		element.LineWidth = 9525
	}
}

func applyStyleFillPaint(element *slideElement, paint backgroundPaint) {
	element.HasFill = true
	element.FillColor = paint.Color
	if paint.HasGradient {
		element.HasFillGradient = true
		element.FillGradient = paint.Gradient
	}
}

func applyThemeEffectStyle(element *slideElement, effects themeEffectStyle) {
	if effects.HasShadow && !element.HasShadow {
		element.HasShadow = true
		element.ShadowColor = effects.ShadowColor
		element.ShadowBlur = effects.ShadowBlur
		element.ShadowDistance = effects.ShadowDistance
		element.ShadowDirection = effects.ShadowDirection
		element.ShadowAlignment = effects.ShadowAlignment
		element.HasShadowRotateWithShape = effects.HasShadowRotateWithShape
		element.ShadowRotateWithShape = effects.ShadowRotateWithShape
		element.HasShadowScaleX = effects.HasShadowScaleX
		element.ShadowScaleX = effects.ShadowScaleX
		element.HasShadowScaleY = effects.HasShadowScaleY
		element.ShadowScaleY = effects.ShadowScaleY
		element.HasShadowSkewX = effects.HasShadowSkewX
		element.ShadowSkewX = effects.ShadowSkewX
		element.HasShadowSkewY = effects.HasShadowSkewY
		element.ShadowSkewY = effects.ShadowSkewY
	}
	if effects.HasShape3D && !element.HasShape3D {
		element.HasShape3D = true
		element.Shape3DFeatures = append([]string{}, effects.Shape3DFeatures...)
		element.HasEffectProperties = true
	}
}

func lineEndMarkerProperties(ln *xmlNode, name string) (string, string, string) {
	end := firstChild(ln, name)
	if end == nil {
		return "", "", ""
	}
	typ := attrValue(end.Attrs, "type")
	if typ == "none" {
		return "", "", ""
	}
	return typ, attrValue(end.Attrs, "w"), attrValue(end.Attrs, "len")
}

func parseCustomGeometryPath(custGeom *xmlNode) []pathPoint {
	points, _ := parseCustomGeometryPathWithDiagnostics(custGeom)
	return points
}

func parseCustomGeometryPathWithDiagnostics(custGeom *xmlNode) ([]pathPoint, []string) {
	points, _, unsupported := parseCustomGeometryPathCommandsWithDiagnostics(custGeom)
	return points, unsupported
}

func parseCustomGeometryPathCommandsWithDiagnostics(custGeom *xmlNode) ([]pathPoint, []pathCommand, []string) {
	pathList := firstChild(custGeom, "pathLst")
	if pathList == nil {
		return nil, nil, []string{"custom geometry has no path list"}
	}
	pathNodes := childrenByName(pathList, "path")
	if len(pathNodes) == 0 {
		return nil, nil, []string{"custom geometry has no path"}
	}
	var unsupported []string
	if len(pathNodes) > 1 {
		unsupported = append(unsupported, "custom geometry uses multiple paths")
	}
	pathNode := pathNodes[0]
	width := parseIntAttr(pathNode.Attrs, "w")
	height := parseIntAttr(pathNode.Attrs, "h")
	if width <= 0 || height <= 0 {
		return nil, nil, append(unsupported, "custom geometry path has no coordinate bounds")
	}
	var points []pathPoint
	var commands []pathCommand
	var current pathPoint
	hasCurrent := false
	for _, command := range pathNode.Children {
		switch command.Name {
		case "moveTo":
			pt := firstChild(command, "pt")
			if pt == nil {
				continue
			}
			current = normalizedPathPoint(pt, width, height)
			hasCurrent = true
			points = append(points, current)
			commands = append(commands, pathCommand{Kind: "moveTo", Points: []pathPoint{current}})
		case "lnTo":
			pt := firstChild(command, "pt")
			if pt == nil || !hasCurrent {
				continue
			}
			current = normalizedPathPoint(pt, width, height)
			points = append(points, current)
			commands = append(commands, pathCommand{Kind: "lnTo", Points: []pathPoint{current}})
		case "cubicBezTo":
			curvePoints := childrenByName(command, "pt")
			if len(curvePoints) != 3 || !hasCurrent {
				continue
			}
			c1 := normalizedPathPoint(curvePoints[0], width, height)
			c2 := normalizedPathPoint(curvePoints[1], width, height)
			end := normalizedPathPoint(curvePoints[2], width, height)
			commands = append(commands, pathCommand{Kind: "cubicBezTo", Points: []pathPoint{c1, c2, end}})
			for step := 1; step <= customBezierSegments; step++ {
				t := float64(step) / customBezierSegments
				points = append(points, cubicBezierPoint(current, c1, c2, end, t))
			}
			current = end
		case "close":
			if len(points) > 0 {
				current = points[0]
				hasCurrent = true
			}
			commands = append(commands, pathCommand{Kind: "close"})
		default:
			unsupported = append(unsupported, fmt.Sprintf("custom geometry uses unsupported %s command", command.Name))
		}
	}
	if len(points) < 3 {
		return nil, nil, append(unsupported, "custom geometry path has fewer than three points")
	}
	return points, commands, sortedUniqueStrings(unsupported)
}

func sortedUniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	unique := map[string]bool{}
	for _, value := range values {
		if value != "" {
			unique[value] = true
		}
	}
	return sortedKeys(unique)
}

func normalizedPathPoint(node *xmlNode, width int64, height int64) pathPoint {
	return pathPoint{
		X: float64(parseIntAttr(node.Attrs, "x")) / float64(width),
		Y: float64(parseIntAttr(node.Attrs, "y")) / float64(height),
	}
}

func cubicBezierPoint(p0 pathPoint, p1 pathPoint, p2 pathPoint, p3 pathPoint, t float64) pathPoint {
	mt := 1 - t
	return pathPoint{
		X: mt*mt*mt*p0.X + 3*mt*mt*t*p1.X + 3*mt*t*t*p2.X + t*t*t*p3.X,
		Y: mt*mt*mt*p0.Y + 3*mt*mt*t*p1.Y + 3*mt*t*t*p2.Y + t*t*t*p3.Y,
	}
}

func parseTransform(xfrm *xmlNode, transform renderTransform, element *slideElement) {
	element.HasTransform = true
	element.FlipH = attrValue(xfrm.Attrs, "flipH") == "1"
	element.FlipV = attrValue(xfrm.Attrs, "flipV") == "1"
	if rot := attrValue(xfrm.Attrs, "rot"); rot != "" && rot != "0" {
		element.HasRotation = true
		element.Rotation = int(parseIntAttr(xfrm.Attrs, "rot"))
	}
	off := firstChild(xfrm, "off")
	ext := firstChild(xfrm, "ext")
	if off != nil {
		element.OffX = transformCoord(parseIntAttr(off.Attrs, "x"), transform.ScaleX, transform.OffsetX)
		element.OffY = transformCoord(parseIntAttr(off.Attrs, "y"), transform.ScaleY, transform.OffsetY)
	}
	if ext != nil {
		element.ExtCX = transformLength(parseIntAttr(ext.Attrs, "cx"), transform.ScaleX)
		element.ExtCY = transformLength(parseIntAttr(ext.Attrs, "cy"), transform.ScaleY)
	}
}

func parseTextTransform(xfrm *xmlNode, transform renderTransform, element *slideElement) {
	off := firstChild(xfrm, "off")
	ext := firstChild(xfrm, "ext")
	if off == nil || ext == nil {
		return
	}
	element.HasTextTransform = true
	element.TextOffX = transformCoord(parseIntAttr(off.Attrs, "x"), transform.ScaleX, transform.OffsetX)
	element.TextOffY = transformCoord(parseIntAttr(off.Attrs, "y"), transform.ScaleY, transform.OffsetY)
	element.TextExtCX = transformLength(parseIntAttr(ext.Attrs, "cx"), transform.ScaleX)
	element.TextExtCY = transformLength(parseIntAttr(ext.Attrs, "cy"), transform.ScaleY)
}

func parseTextProperties(node *xmlNode, element *slideElement, theme themeColors) {
	for _, child := range node.Children {
		if child.Name == "bodyPr" {
			parseBodyProperties(child, element)
		}
		if child.Name == "rPr" || child.Name == "defRPr" || child.Name == "endParaRPr" {
			if size := parseIntAttr(child.Attrs, "sz"); size > 0 && child.Name == "defRPr" && element.FontSize == 0 {
				element.FontSize = int(size)
			}
			if child.Name == "defRPr" && attrValue(child.Attrs, "i") == "1" {
				element.Italic = true
			}
			if child.Name == "defRPr" {
				if solidFill := firstChild(child, "solidFill"); solidFill != nil {
					if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok && !element.HasTextColor {
						element.HasTextColor = true
						element.TextColor = textColor
					}
				}
			}
		}
		parseTextProperties(child, element, theme)
	}
}

func parseBodyProperties(node *xmlNode, element *slideElement) {
	element.HasBodyProperties = true
	if wrap := attrValue(node.Attrs, "wrap"); wrap != "" {
		element.HasTextWrap = true
		element.TextWrap = wrap
	}
	if overflow := attrValue(node.Attrs, "horzOverflow"); overflow != "" {
		element.HasTextHorizontalOverflow = true
		element.TextHorizontalOverflow = overflow
	}
	if overflow := attrValue(node.Attrs, "vertOverflow"); overflow != "" {
		element.HasTextVerticalOverflow = true
		element.TextVerticalOverflow = overflow
	}
	if anchor := attrValue(node.Attrs, "anchor"); anchor != "" {
		element.TextAnchor = anchor
	}
	if vertical := attrValue(node.Attrs, "vert"); vertical != "" {
		element.HasTextVertical = true
		element.TextVertical = vertical
	}
	if rotation := attrValue(node.Attrs, "rot"); rotation != "" {
		element.TextBodyRotation = int(parseIntAttr(node.Attrs, "rot"))
		element.HasTextBodyRotation = true
	}
	if attrValue(node.Attrs, "numCol") != "" {
		element.HasTextColumns = true
		if columns := int(parseIntAttr(node.Attrs, "numCol")); columns > 0 {
			element.TextColumnCount = columns
		}
	}
	if value := attrValue(node.Attrs, "anchorCtr"); value != "" {
		element.HasTextAnchorCenter = true
		element.TextAnchorCenter = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "spcFirstLastPara"); value != "" {
		element.HasFirstLastSpacing = true
		element.IncludeFirstLastSpacing = boolAttrOn(value)
	}
	if firstChild(node, "noAutofit") != nil {
		element.HasNoAutofit = true
		element.HasShapeAutofit = false
		element.HasNormAutofit = false
		element.HasFontScalePct = false
		element.FontScalePct = 0
		element.HasLineSpacingReductionPct = false
		element.LineSpacingReductionPct = 0
	}
	if firstChild(node, "spAutoFit") != nil {
		element.HasShapeAutofit = true
	}
	if insets, ok := parseTextBodyInsets(node.Attrs); ok {
		element.HasInsets = true
		element.InsetLeft = insets.Left
		element.InsetTop = insets.Top
		element.InsetRight = insets.Right
		element.InsetBottom = insets.Bottom
	}
	if autofit := firstChild(node, "normAutofit"); autofit != nil {
		element.HasNormAutofit = true
		if attrValue(autofit.Attrs, "fontScale") != "" {
			element.HasFontScalePct = true
			fontScale := int(parsePercentAttr(autofit.Attrs, "fontScale"))
			if fontScale > 0 {
				element.FontScalePct = fontScale
			} else {
				element.FontScalePct = 100000
			}
		}
		if attrValue(autofit.Attrs, "lnSpcReduction") != "" {
			element.HasLineSpacingReductionPct = true
			if lineSpacingReduction := int(parsePercentAttr(autofit.Attrs, "lnSpcReduction")); lineSpacingReduction > 0 {
				element.LineSpacingReductionPct = lineSpacingReduction
			}
		}
	}
	if element.HasNoAutofit {
		element.HasShapeAutofit = false
		element.HasNormAutofit = false
		element.HasFontScalePct = false
		element.FontScalePct = 0
		element.HasLineSpacingReductionPct = false
		element.LineSpacingReductionPct = 0
	}
}

func parseTextBodyInsets(attrs []xml.Attr) (textInsets, bool) {
	insets := textInsets{
		Left:   defaultTextInsetXEMU,
		Top:    defaultTextInsetYEMU,
		Right:  defaultTextInsetXEMU,
		Bottom: defaultTextInsetYEMU,
	}
	return parseInsetsWithDefaults(attrs, insets, "lIns", "tIns", "rIns", "bIns")
}

func parseTableCellMargins(attrs []xml.Attr) (textInsets, bool) {
	insets := textInsets{
		Left:   defaultTableCellHorizontalMarginEMU,
		Top:    defaultTableCellVerticalMarginEMU,
		Right:  defaultTableCellHorizontalMarginEMU,
		Bottom: defaultTableCellVerticalMarginEMU,
	}
	return parseInsetsWithDefaults(attrs, insets, "marL", "marT", "marR", "marB")
}

func parseInsetsWithDefaults(attrs []xml.Attr, defaults textInsets, leftName string, topName string, rightName string, bottomName string) (textInsets, bool) {
	insets := defaults
	found := false
	for _, item := range []struct {
		name  string
		value *int64
	}{
		{name: leftName, value: &insets.Left},
		{name: topName, value: &insets.Top},
		{name: rightName, value: &insets.Right},
		{name: bottomName, value: &insets.Bottom},
	} {
		raw := attrValue(attrs, item.name)
		if raw == "" {
			continue
		}
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			*item.value = parsed
			found = true
		}
	}
	return insets, found
}
