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
		element.CustomPaths, element.CustomPathFills, element.CustomPathStrokes, element.CustomPathCommands, element.CustomPathUnsupported = parseCustomGeometryPathsCommandsWithDiagnostics(custGeom)
		if len(element.CustomPaths) > 0 {
			element.CustomPath = element.CustomPaths[0]
		}
	}
	if firstChild(spPr, "noFill") != nil {
		element.NoFill = true
	}
	if paint, ok := fillPaintFromContainer(spPr, theme, transform.GroupFill); ok {
		applyFillPaintToElement(element, paint)
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
		if compound := attrValue(ln.Attrs, "cmpd"); compound != "" {
			element.HasLineCompound = true
			element.LineCompound = compound
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
		} else if dash := firstChild(ln, "custDash"); dash != nil {
			element.HasLineDash = true
			element.LineDash = customDashPatternValue(dash)
		}
		if join := lineJoinValue(ln); join != "" {
			element.HasLineJoin = true
			element.LineJoin = join
		}
		element.HeadLineMarker, element.HeadLineMarkerWidth, element.HeadLineMarkerLength = lineEndMarkerProperties(ln, "headEnd")
		element.TailLineMarker, element.TailLineMarkerWidth, element.TailLineMarkerLength = lineEndMarkerProperties(ln, "tailEnd")
		element.HasLineMarker = element.HeadLineMarker != "" || element.TailLineMarker != ""
	}
	if effectList := firstChild(spPr, "effectLst"); effectList != nil {
		element.HasEffectProperties = true
		parseShapeEffects(effectList, element, theme)
	} else if effectDag := firstChild(spPr, "effectDag"); effectDag != nil {
		element.HasEffectProperties = true
		parseShapeEffectDag(effectDag, element, theme)
	}
	if scene3d := firstChild(spPr, "scene3d"); scene3d != nil {
		parseScene3DProperties(scene3d, element)
	}
	if sp3d := firstChild(spPr, "sp3d"); sp3d != nil {
		parseShape3DProperties(sp3d, element)
	}
}

func parseShapeEffectDag(effectDag *xmlNode, element *slideElement, theme themeColors) {
	if effectDag == nil {
		return
	}
	flattened := &xmlNode{Name: "effectLst"}
	unsupported := flattenSupportedEffectDagNodes(effectDag, flattened)
	if len(flattened.Children) > 0 {
		parseShapeEffects(flattened, element, theme)
		element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset")
	}
	if unsupported || len(flattened.Children) == 0 {
		element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes")
	}
}

func flattenSupportedEffectDagNodes(node *xmlNode, flattened *xmlNode) bool {
	if node == nil {
		return false
	}
	unsupported := false
	for _, child := range node.Children {
		switch child.Name {
		case "cont":
			if attrValue(child.Attrs, "type") == "tree" {
				unsupported = true
			}
			if flattenSupportedEffectDagNodes(child, flattened) {
				unsupported = true
			}
		case "blend":
			unsupported = true
			if blendContainer := firstChild(child, "cont"); blendContainer != nil {
				if flattenSupportedEffectDagNodes(blendContainer, flattened) {
					unsupported = true
				}
			}
		case "alphaOutset", "blur", "fillOverlay", "glow", "innerShdw", "outerShdw", "prstShdw", "reflection", "relOff", "softEdge", "xfrm":
			flattened.Children = append(flattened.Children, child)
		case "":
			continue
		default:
			if visibleEffectDagNode(child) {
				unsupported = true
			}
		}
	}
	return unsupported
}

func visibleEffectDagNode(node *xmlNode) bool {
	if node == nil {
		return false
	}
	switch node.Name {
	case "effect":
		return attrValue(node.Attrs, "ref") != ""
	case "alphaBiLevel", "alphaCeiling", "alphaFloor", "alphaInv", "alphaMod", "alphaModFix", "alphaOutset", "alphaRepl", "biLevel", "blend", "clrChange", "clrRepl", "duotone", "fill", "grayscl", "hsl", "lum", "relOff", "tint", "xfrm":
		return true
	default:
		return len(node.Children) > 0
	}
}

func parseScene3DProperties(scene3d *xmlNode, element *slideElement) {
	features := visibleScene3DFeatures(scene3d)
	if len(features) == 0 {
		return
	}
	element.HasShape3D = true
	element.Shape3DFeatures = appendDistinctStrings(element.Shape3DFeatures, features...)
	element.HasEffectProperties = true
}

func visibleScene3DFeatures(scene3d *xmlNode) []string {
	if scene3d == nil {
		return nil
	}
	var features []string
	if camera := firstChild(scene3d, "camera"); camera != nil {
		if prst := attrValue(camera.Attrs, "prst"); prst != "" {
			features = append(features, "3-D scene camera "+prst)
		} else {
			features = append(features, "3-D scene camera")
		}
		if attrValue(camera.Attrs, "fov") != "" {
			features = append(features, "3-D scene camera field of view")
		}
		if zoom := attrValue(camera.Attrs, "zoom"); zoom != "" && zoom != "100%" && zoom != "100000" {
			features = append(features, "3-D scene camera zoom")
		}
		if firstChild(camera, "rot") != nil {
			features = append(features, "3-D scene camera rotation")
		}
	}
	if lightRig := firstChild(scene3d, "lightRig"); lightRig != nil {
		rig := attrValue(lightRig.Attrs, "rig")
		dir := attrValue(lightRig.Attrs, "dir")
		switch {
		case rig != "" && dir != "":
			features = append(features, "3-D scene light rig "+rig+"/"+dir)
		case rig != "":
			features = append(features, "3-D scene light rig "+rig)
		default:
			features = append(features, "3-D scene light rig")
		}
		if firstChild(lightRig, "rot") != nil {
			features = append(features, "3-D scene light rig rotation")
		}
	}
	if firstChild(scene3d, "backdrop") != nil {
		features = append(features, "3-D scene backdrop")
	}
	return features
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
	if attrValue(sp3d.Attrs, "z") != "" && parseIntAttr(sp3d.Attrs, "z") != 0 {
		features = append(features, "3-D z offset")
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
	width := int64(76200)
	height := int64(76200)
	if value := attrValue(bevel.Attrs, "w"); value != "" {
		width = parseIntAttr(bevel.Attrs, "w")
	}
	if value := attrValue(bevel.Attrs, "h"); value != "" {
		height = parseIntAttr(bevel.Attrs, "h")
	}
	return width > 0 && height > 0
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
	for _, child := range effectList.Children {
		switch child.Name {
		case "alphaOutset", "blur", "fillOverlay", "glow", "innerShdw", "outerShdw", "prstShdw", "reflection", "relOff", "softEdge", "xfrm":
			continue
		}
	}
	if effectTransform := firstChild(effectList, "xfrm"); effectTransform != nil {
		sx := parsePercentAttrDefault(effectTransform.Attrs, "sx", 100000)
		sy := parsePercentAttrDefault(effectTransform.Attrs, "sy", 100000)
		kx := parseIntAttr(effectTransform.Attrs, "kx")
		ky := parseIntAttr(effectTransform.Attrs, "ky")
		tx := parseIntAttr(effectTransform.Attrs, "tx")
		ty := parseIntAttr(effectTransform.Attrs, "ty")
		if tx != 0 || ty != 0 {
			element.HasEffectTransform = true
			element.EffectTransformScaleX = sx
			element.EffectTransformScaleY = sy
			element.EffectTransformSkewX = kx
			element.EffectTransformSkewY = ky
			element.EffectTransformOffsetX = tx
			element.EffectTransformOffsetY = ty
		}
		if sx != 100000 || sy != 100000 || kx != 0 || ky != 0 {
			element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect xfrm scale/skew transform was not rendered")
		}
	}
	if relOff := firstChild(effectList, "relOff"); relOff != nil {
		tx := parsePercentAttr(relOff.Attrs, "tx")
		ty := parsePercentAttr(relOff.Attrs, "ty")
		if tx != 0 || ty != 0 {
			element.HasRelativeOffset = true
			element.RelativeOffsetX = tx
			element.RelativeOffsetY = ty
		}
	}
	if alphaOutset := firstChild(effectList, "alphaOutset"); alphaOutset != nil {
		if radius := parseIntAttr(alphaOutset.Attrs, "rad"); radius > 0 {
			element.HasAlphaOutset = true
			element.AlphaOutsetRadius = radius
		}
	}
	if blur := firstChild(effectList, "blur"); blur != nil {
		if radius := parseIntAttr(blur.Attrs, "rad"); radius > 0 {
			element.HasBlur = true
			element.BlurRadius = radius
			element.BlurGrow = true
			if grow := attrValue(blur.Attrs, "grow"); grow != "" {
				element.BlurGrow = boolAttrOn(grow)
			}
		}
	}
	if overlay := firstChild(effectList, "fillOverlay"); overlay != nil {
		if paint, ok := fillPaintFromContainer(overlay, theme, nil); ok {
			element.HasFillOverlay = true
			element.FillOverlay = paint
			element.FillOverlayBlend = attrValue(overlay.Attrs, "blend")
			if element.FillOverlayBlend == "" {
				element.FillOverlayBlend = "over"
			}
		} else if visibleShapeEffectNode(overlay) {
			element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect fillOverlay fill was not resolved")
		}
	}
	if innerShadow := firstChild(effectList, "innerShdw"); innerShadow != nil {
		if innerShadowColor, ok := colorFromColorNodeWithTheme(innerShadow, theme); ok {
			element.HasInnerShadow = true
			element.InnerShadowColor = innerShadowColor
			element.InnerShadowBlur = parseIntAttr(innerShadow.Attrs, "blurRad")
			element.InnerShadowDistance = parseIntAttr(innerShadow.Attrs, "dist")
			element.InnerShadowDirection = parseIntAttr(innerShadow.Attrs, "dir")
		} else if visibleShapeEffectNode(innerShadow) {
			element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect innerShdw color was not resolved")
		}
	}
	if reflection := firstChild(effectList, "reflection"); reflection != nil {
		element.HasReflection = true
		element.ReflectionBlur = parseIntAttr(reflection.Attrs, "blurRad")
		element.ReflectionStartAlpha = parsePercentAttrDefault(reflection.Attrs, "stA", 100000)
		element.ReflectionStartPosition = parsePercentAttrDefault(reflection.Attrs, "stPos", 0)
		element.ReflectionEndAlpha = parsePercentAttrDefault(reflection.Attrs, "endA", 0)
		element.ReflectionEndPosition = parsePercentAttrDefault(reflection.Attrs, "endPos", 100000)
		element.ReflectionDistance = parseIntAttr(reflection.Attrs, "dist")
		element.ReflectionDirection = parseIntAttr(reflection.Attrs, "dir")
		element.ReflectionFadeDirection = parseIntAttrDefault(reflection.Attrs, "fadeDir", 5400000)
		element.ReflectionScaleX = parsePercentAttrDefault(reflection.Attrs, "sx", 100000)
		element.ReflectionScaleY = parsePercentAttrDefault(reflection.Attrs, "sy", 100000)
		element.ReflectionSkewX = parseIntAttr(reflection.Attrs, "kx")
		element.ReflectionSkewY = parseIntAttr(reflection.Attrs, "ky")
		element.ReflectionAlignment = attrValue(reflection.Attrs, "algn")
		if element.ReflectionAlignment == "" {
			element.ReflectionAlignment = "b"
		}
		if value := attrValue(reflection.Attrs, "rotWithShape"); value != "" {
			element.HasReflectionRotate = true
			element.ReflectionRotateWithShape = boolAttrOn(value)
		} else {
			element.ReflectionRotateWithShape = true
		}
		if element.ReflectionScaleX != 100000 || element.ReflectionScaleY != 100000 || element.ReflectionSkewX != 0 || element.ReflectionSkewY != 0 || element.ReflectionAlignment != "b" || (element.HasReflectionRotate && !element.ReflectionRotateWithShape) {
			element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect reflection transform was rendered with simplified bottom mirror geometry")
		}
	}
	if softEdge := firstChild(effectList, "softEdge"); softEdge != nil {
		if radius := parseIntAttr(softEdge.Attrs, "rad"); radius > 0 {
			element.HasSoftEdge = true
			element.SoftEdgeRadius = radius
		}
	}
	if glow := firstChild(effectList, "glow"); glow != nil {
		if radius := parseIntAttr(glow.Attrs, "rad"); radius > 0 {
			if glowColor, ok := colorFromColorNodeWithTheme(glow, theme); ok {
				element.HasGlow = true
				element.GlowColor = glowColor
				element.GlowRadius = radius
			} else {
				element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect glow color was not resolved")
			}
		}
	}
	shadow := firstChild(effectList, "outerShdw")
	presetShadow := false
	if shadow == nil {
		shadow = firstChild(effectList, "prstShdw")
		presetShadow = shadow != nil
	}
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
	if presetShadow {
		element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, "effect prstShdw preset style was rendered as a simplified offset shadow")
	}
}

func visibleShapeEffectNode(node *xmlNode) bool {
	if node == nil {
		return false
	}
	switch node.Name {
	case "blur":
		return parseIntAttr(node.Attrs, "rad") > 0
	case "fillOverlay":
		return len(node.Children) > 0
	case "glow":
		return parseIntAttr(node.Attrs, "rad") > 0
	case "innerShdw":
		return parseIntAttr(node.Attrs, "blurRad") > 0 || parseIntAttr(node.Attrs, "dist") > 0 || len(node.Children) > 0
	case "prstShdw":
		return attrValue(node.Attrs, "prst") != "" || parseIntAttr(node.Attrs, "dist") > 0 || len(node.Children) > 0
	case "reflection":
		return parseIntAttr(node.Attrs, "blurRad") > 0 || parseIntAttr(node.Attrs, "dist") > 0 || parsePercentAttr(node.Attrs, "stA") > 0 || parsePercentAttr(node.Attrs, "endA") > 0
	default:
		return false
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
	if !element.HasLineJoin && border.Join != "" {
		element.HasLineJoin = true
		element.LineJoin = border.Join
	}
	if !element.HasLineCompound && border.Compound != "" {
		element.HasLineCompound = true
		element.LineCompound = border.Compound
	}
	if element.LineWidth == 0 {
		element.LineWidth = 9525
	}
}

func applyStyleFillPaint(element *slideElement, paint backgroundPaint) {
	applyFillPaintToElement(element, paint)
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
	if effects.HasGlow && !element.HasGlow {
		element.HasGlow = true
		element.GlowColor = effects.GlowColor
		element.GlowRadius = effects.GlowRadius
		element.HasEffectProperties = true
	}
	if effects.HasBlur && !element.HasBlur {
		element.HasBlur = true
		element.BlurRadius = effects.BlurRadius
		element.BlurGrow = effects.BlurGrow
		element.HasEffectProperties = true
	}
	if effects.HasAlphaOutset && !element.HasAlphaOutset {
		element.HasAlphaOutset = true
		element.AlphaOutsetRadius = effects.AlphaOutsetRadius
		element.HasEffectProperties = true
	}
	if effects.HasRelativeOffset && !element.HasRelativeOffset {
		element.HasRelativeOffset = true
		element.RelativeOffsetX = effects.RelativeOffsetX
		element.RelativeOffsetY = effects.RelativeOffsetY
		element.HasEffectProperties = true
	}
	if effects.HasEffectTransform && !element.HasEffectTransform {
		element.HasEffectTransform = true
		element.EffectTransformScaleX = effects.EffectTransformScaleX
		element.EffectTransformScaleY = effects.EffectTransformScaleY
		element.EffectTransformSkewX = effects.EffectTransformSkewX
		element.EffectTransformSkewY = effects.EffectTransformSkewY
		element.EffectTransformOffsetX = effects.EffectTransformOffsetX
		element.EffectTransformOffsetY = effects.EffectTransformOffsetY
		element.HasEffectProperties = true
	}
	if effects.HasFillOverlay && !element.HasFillOverlay {
		element.HasFillOverlay = true
		element.FillOverlay = effects.FillOverlay
		element.FillOverlayBlend = effects.FillOverlayBlend
		element.HasEffectProperties = true
	}
	if effects.HasInnerShadow && !element.HasInnerShadow {
		element.HasInnerShadow = true
		element.InnerShadowColor = effects.InnerShadowColor
		element.InnerShadowBlur = effects.InnerShadowBlur
		element.InnerShadowDistance = effects.InnerShadowDistance
		element.InnerShadowDirection = effects.InnerShadowDirection
		element.HasEffectProperties = true
	}
	if effects.HasReflection && !element.HasReflection {
		element.HasReflection = true
		element.ReflectionBlur = effects.ReflectionBlur
		element.ReflectionStartAlpha = effects.ReflectionStartAlpha
		element.ReflectionStartPosition = effects.ReflectionStartPosition
		element.ReflectionEndAlpha = effects.ReflectionEndAlpha
		element.ReflectionEndPosition = effects.ReflectionEndPosition
		element.ReflectionDistance = effects.ReflectionDistance
		element.ReflectionDirection = effects.ReflectionDirection
		element.ReflectionFadeDirection = effects.ReflectionFadeDirection
		element.ReflectionScaleX = effects.ReflectionScaleX
		element.ReflectionScaleY = effects.ReflectionScaleY
		element.ReflectionSkewX = effects.ReflectionSkewX
		element.ReflectionSkewY = effects.ReflectionSkewY
		element.ReflectionAlignment = effects.ReflectionAlignment
		element.HasReflectionRotate = effects.HasReflectionRotate
		element.ReflectionRotateWithShape = effects.ReflectionRotateWithShape
		element.HasEffectProperties = true
	}
	if len(effects.EffectUnsupported) > 0 {
		element.EffectUnsupported = appendDistinctStrings(element.EffectUnsupported, effects.EffectUnsupported...)
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
	paths, _, _, commands, unsupported := parseCustomGeometryPathsCommandsWithDiagnostics(custGeom)
	if len(paths) == 0 {
		return nil, commands, unsupported
	}
	return paths[0], commands, unsupported
}

func parseCustomGeometryPathsCommandsWithDiagnostics(custGeom *xmlNode) ([][]pathPoint, []bool, []bool, []pathCommand, []string) {
	pathList := firstChild(custGeom, "pathLst")
	if pathList == nil {
		return nil, nil, nil, nil, []string{"custom geometry has no path list"}
	}
	pathNodes := childrenByName(pathList, "path")
	if len(pathNodes) == 0 {
		return nil, nil, nil, nil, []string{"custom geometry has no path"}
	}
	var unsupported []string
	var paths [][]pathPoint
	var fills []bool
	var strokes []bool
	var commands []pathCommand
	for _, pathNode := range pathNodes {
		points, pathCommands, pathUnsupported := parseCustomGeometryPathNode(pathNode)
		unsupported = append(unsupported, pathUnsupported...)
		if len(points) < 3 {
			continue
		}
		paths = append(paths, points)
		fills = append(fills, customPathHasFill(pathNode))
		strokes = append(strokes, customPathHasStroke(pathNode))
		commands = append(commands, pathCommands...)
	}
	if len(paths) == 0 {
		return nil, nil, nil, commands, sortedUniqueStrings(unsupported)
	}
	return paths, fills, strokes, commands, sortedUniqueStrings(unsupported)
}

func parseCustomGeometryPathNode(pathNode *xmlNode) ([]pathPoint, []pathCommand, []string) {
	width := parseIntAttr(pathNode.Attrs, "w")
	height := parseIntAttr(pathNode.Attrs, "h")
	if width <= 0 || height <= 0 {
		return nil, nil, []string{"custom geometry path has no coordinate bounds"}
	}
	var unsupported []string
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
		case "quadBezTo":
			curvePoints := childrenByName(command, "pt")
			if len(curvePoints) != 2 || !hasCurrent {
				continue
			}
			c1 := normalizedPathPoint(curvePoints[0], width, height)
			end := normalizedPathPoint(curvePoints[1], width, height)
			commands = append(commands, pathCommand{Kind: "quadBezTo", Points: []pathPoint{c1, end}})
			for step := 1; step <= customBezierSegments; step++ {
				t := float64(step) / customBezierSegments
				points = append(points, quadraticBezierPoint(current, c1, end, t))
			}
			current = end
		case "arcTo":
			if !hasCurrent {
				continue
			}
			arcPoints := appendCustomGeometryArcPoints(current, command, width, height)
			if len(arcPoints) == 0 {
				unsupported = append(unsupported, "custom geometry arcTo command has invalid radius")
				continue
			}
			commands = append(commands, pathCommand{Kind: "arcTo", Points: append([]pathPoint{}, arcPoints...)})
			points = append(points, arcPoints...)
			current = arcPoints[len(arcPoints)-1]
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
		return nil, commands, append(unsupported, "custom geometry path has fewer than three points")
	}
	return points, commands, sortedUniqueStrings(unsupported)
}

func customPathHasFill(pathNode *xmlNode) bool {
	switch attrValue(pathNode.Attrs, "fill") {
	case "", "norm":
		return true
	case "none":
		return false
	default:
		return true
	}
}

func customPathHasStroke(pathNode *xmlNode) bool {
	value := attrValue(pathNode.Attrs, "stroke")
	return value == "" || boolAttrOn(value)
}

func customDashPatternValue(dash *xmlNode) string {
	var parts []string
	for _, stop := range childrenByName(dash, "ds") {
		d := parsePercentAttr(stop.Attrs, "d")
		sp := parsePercentAttr(stop.Attrs, "sp")
		if d <= 0 && sp <= 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%d/%d", d, sp))
	}
	if len(parts) == 0 {
		return ""
	}
	return "cust:" + strings.Join(parts, ",")
}

func lineJoinValue(ln *xmlNode) string {
	switch {
	case firstChild(ln, "round") != nil:
		return "round"
	case firstChild(ln, "bevel") != nil:
		return "bevel"
	case firstChild(ln, "miter") != nil:
		return "miter"
	default:
		return ""
	}
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

func quadraticBezierPoint(p0 pathPoint, p1 pathPoint, p2 pathPoint, t float64) pathPoint {
	mt := 1 - t
	return pathPoint{
		X: mt*mt*p0.X + 2*mt*t*p1.X + t*t*p2.X,
		Y: mt*mt*p0.Y + 2*mt*t*p1.Y + t*t*p2.Y,
	}
}

func appendCustomGeometryArcPoints(current pathPoint, command *xmlNode, width int64, height int64) []pathPoint {
	radiusX := float64(parseIntAttr(command.Attrs, "wR"))
	radiusY := float64(parseIntAttr(command.Attrs, "hR"))
	if radiusX <= 0 || radiusY <= 0 {
		return nil
	}
	start := parseIntAttr(command.Attrs, "stAng")
	sweep := parseIntAttr(command.Attrs, "swAng")
	absolute := []pathPoint{{X: current.X * float64(width), Y: current.Y * float64(height)}}
	absolute = appendOoxmlArcPoints(absolute, radiusX, radiusY, float64(start), float64(sweep))
	if len(absolute) <= 1 {
		return nil
	}
	normalized := make([]pathPoint, 0, len(absolute)-1)
	for _, point := range absolute[1:] {
		normalized = append(normalized, pathPoint{X: point.X / float64(width), Y: point.Y / float64(height)})
	}
	return normalized
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
	if value := attrValue(node.Attrs, "rtlCol"); value != "" {
		element.HasTextRightToLeftColumns = true
		element.TextRightToLeftColumns = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "anchorCtr"); value != "" {
		element.HasTextAnchorCenter = true
		element.TextAnchorCenter = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "spcFirstLastPara"); value != "" {
		element.HasFirstLastSpacing = true
		element.IncludeFirstLastSpacing = boolAttrOn(value)
	}
	parseTextBody3DProperties(node, element)
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

func parseTextBody3DProperties(node *xmlNode, element *slideElement) {
	if scene3d := firstChild(node, "scene3d"); scene3d != nil {
		element.Text3DFeatures = appendDistinctStrings(element.Text3DFeatures, prefix3DTextFeatures(visibleScene3DFeatures(scene3d))...)
	}
	if sp3d := firstChild(node, "sp3d"); sp3d != nil {
		element.Text3DFeatures = appendDistinctStrings(element.Text3DFeatures, prefix3DTextFeatures(visibleShape3DFeatures(sp3d))...)
	}
	if flatTx := firstChild(node, "flatTx"); flatTx != nil {
		if attrValue(flatTx.Attrs, "z") != "" && parseIntAttr(flatTx.Attrs, "z") != 0 {
			element.Text3DFeatures = appendDistinctStrings(element.Text3DFeatures, "text 3-D flat text z offset")
		} else {
			element.Text3DFeatures = appendDistinctStrings(element.Text3DFeatures, "text 3-D flat text")
		}
	}
}

func prefix3DTextFeatures(features []string) []string {
	if len(features) == 0 {
		return nil
	}
	prefixed := make([]string, 0, len(features))
	for _, feature := range features {
		if feature == "" {
			continue
		}
		prefixed = append(prefixed, "text "+feature)
	}
	return prefixed
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
