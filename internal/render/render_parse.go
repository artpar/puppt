package render

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image/color"
	"math"
	"sort"
	"strings"
)

func collectSlideElements(data []byte) []slideElement {
	return collectSlideElementsWithTheme(data, defaultThemeColors())
}

func collectSlideElementsWithTheme(data []byte, theme themeColors) []slideElement {
	return collectSlideElementsWithThemeAndEffects(data, theme, themeEffectStyles{})
}

func collectSlideElementsWithThemeAndEffects(data []byte, theme themeColors, effectStyles themeEffectStyles) []slideElement {
	return collectSlideElementsWithThemeEffectsAndFills(data, theme, effectStyles, themeFillStyles{}, themeLineStyles{})
}

func collectSlideElementsWithThemeEffectsAndFills(data []byte, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) []slideElement {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	return collectSlideElementsFromRootWithThemeEffectsAndFills(root, theme, effectStyles, fillStyles, lineStyles)
}

func collectSlideElementsFromRootWithThemeEffectsAndFills(root *xmlNode, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) []slideElement {
	if root == nil {
		return nil
	}
	return collectElementsFromNode(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, effectStyles, fillStyles, lineStyles)
}

func cloneSlideElements(elements []slideElement) []slideElement {
	if len(elements) == 0 {
		return nil
	}
	cloned := make([]slideElement, len(elements))
	copy(cloned, elements)
	for index := range cloned {
		cloned[index].TextParagraphs = cloneTextParagraphs(cloned[index].TextParagraphs)
		cloned[index].PlaceholderParagraphStyles = cloneParagraphStyleMap(cloned[index].PlaceholderParagraphStyles)
		cloned[index].EffectUnsupported = append([]string{}, cloned[index].EffectUnsupported...)
		cloned[index].Shape3DFeatures = append([]string{}, cloned[index].Shape3DFeatures...)
		cloned[index].NonVisualProperties = append([]string{}, cloned[index].NonVisualProperties...)
		cloned[index].NonVisualLocks = append([]string{}, cloned[index].NonVisualLocks...)
	}
	return cloned
}

func cloneParagraphStyleMap(styles map[int]paragraphStyle) map[int]paragraphStyle {
	if len(styles) == 0 {
		return nil
	}
	cloned := make(map[int]paragraphStyle, len(styles))
	for key, style := range styles {
		cloned[key] = style
	}
	return cloned
}

func parseXMLNode(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []*xmlNode
	var root *xmlNode
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch item := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Name: item.Name.Local, Attrs: item.Attr}
			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.Children = append(parent.Children, node)
			}
			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].Text += string(item)
			}
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if root == nil {
		return nil, errors.New("empty xml")
	}
	return root, nil
}

func collectElementsFromNode(node *xmlNode, transform renderTransform, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) []slideElement {
	var elements []slideElement
	for _, child := range node.Children {
		switch child.Name {
		case "sp", "cxnSp", "pic", "graphicFrame", "txSp":
			if child.Name == "graphicFrame" {
				if lockedCanvas := firstDescendant(child, "lockedCanvas"); lockedCanvas != nil {
					frameTransform := composeGraphicFrameContentTransform(transform, child)
					elements = append(elements, collectElementsFromNode(lockedCanvas, composeGroupTransform(frameTransform, lockedCanvas, theme), theme, effectStyles, fillStyles, lineStyles)...)
					continue
				}
			}
			element := parseSlideElementNodeWithThemeEffectsAndFills(child, transform, theme, effectStyles, fillStyles, lineStyles)
			if child.Name == "txSp" {
				element.Kind = "sp"
			}
			elements = append(elements, element)
		case "contentPart", "oleObj", "control", "audio", "video", "audioFile", "videoFile":
			element := parseSlideElementNodeWithThemeEffectsAndFills(child, transform, theme, effectStyles, fillStyles, lineStyles)
			elements = append(elements, element)
			elements = append(elements, collectElementsFromNode(child, transform, theme, effectStyles, fillStyles, lineStyles)...)
		case "grpSp":
			elements = append(elements, collectElementsFromNode(child, composeGroupTransform(transform, child, theme), theme, effectStyles, fillStyles, lineStyles)...)
		default:
			elements = append(elements, collectElementsFromNode(child, transform, theme, effectStyles, fillStyles, lineStyles)...)
		}
	}
	return elements
}

func parseSlideElementNode(node *xmlNode, transform renderTransform) slideElement {
	return parseSlideElementNodeWithTheme(node, transform, defaultThemeColors())
}

func parseSlideElementNodeWithTheme(node *xmlNode, transform renderTransform, theme themeColors) slideElement {
	return parseSlideElementNodeWithThemeAndEffects(node, transform, theme, themeEffectStyles{})
}

func parseSlideElementNodeWithThemeAndEffects(node *xmlNode, transform renderTransform, theme themeColors, effectStyles themeEffectStyles) slideElement {
	return parseSlideElementNodeWithThemeEffectsAndFills(node, transform, theme, effectStyles, themeFillStyles{}, themeLineStyles{})
}

func parseSlideElementNodeWithThemeEffectsAndFills(node *xmlNode, transform renderTransform, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) slideElement {
	element := slideElement{Kind: node.Name}
	if cNvPr := firstDescendant(node, "cNvPr"); cNvPr != nil {
		parseCommonNonVisualDrawingProperties(cNvPr, &element)
	}
	if node.Name == "sp" || node.Name == "txSp" {
		if nvSpPr := firstChild(node, "nvSpPr"); nvSpPr != nil {
			if cNvSpPr := firstChild(nvSpPr, "cNvSpPr"); cNvSpPr != nil {
				element.IsTextBox = boolAttrOn(attrValue(cNvSpPr.Attrs, "txBox"))
				element.NonVisualLocks = append(element.NonVisualLocks, nonVisualLockFlags(cNvSpPr)...)
			}
		}
	}
	if node.Name == "pic" {
		if nvPicPr := firstChild(node, "nvPicPr"); nvPicPr != nil {
			if cNvPicPr := firstChild(nvPicPr, "cNvPicPr"); cNvPicPr != nil {
				element.NonVisualLocks = append(element.NonVisualLocks, nonVisualLockFlags(cNvPicPr)...)
			}
		}
	}
	if node.Name == "graphicFrame" {
		if nvGraphicFramePr := firstChild(node, "nvGraphicFramePr"); nvGraphicFramePr != nil {
			if cNvGraphicFramePr := firstChild(nvGraphicFramePr, "cNvGraphicFramePr"); cNvGraphicFramePr != nil {
				element.NonVisualLocks = append(element.NonVisualLocks, nonVisualLockFlags(cNvGraphicFramePr)...)
			}
		}
	}
	if ph := firstDescendant(node, "ph"); ph != nil {
		element.IsPlaceholder = true
		element.PlaceholderType = attrValue(ph.Attrs, "type")
		element.PlaceholderIdx = attrValue(ph.Attrs, "idx")
	}
	if blip := firstDescendant(node, "blip"); blip != nil {
		element.EmbedID = attrValue(blip.Attrs, "embed")
		element.LinkID = attrValue(blip.Attrs, "link")
		element.BlipCompressionState = attrValue(blip.Attrs, "cstate")
		parseBlipEffectsWithTheme(blip, &element, theme)
	}
	if blipFill := firstDescendant(node, "blipFill"); blipFill != nil {
		parseBlipFillMode(blipFill, &element)
		if value := attrValue(blipFill.Attrs, "rotWithShape"); value != "" {
			element.HasBlipRotWithShape = true
			element.BlipRotWithShape = boolAttrOn(value)
		}
	}
	if svgBlip := firstDescendant(node, "svgBlip"); svgBlip != nil {
		element.SVGEmbedID = attrValue(svgBlip.Attrs, "embed")
	}
	if relIDs := firstDescendant(node, "relIds"); relIDs != nil {
		element.DiagramDataID = attrValue(relIDs.Attrs, "dm")
	}
	if node.Name == "graphicFrame" {
		if tableNode := firstDescendant(node, "tbl"); tableNode != nil {
			element.HasTable = true
			element.Table = parseTableModel(tableNode, theme)
		}
	}
	parseNonBasicPayloadProperties(node, &element)
	if srcRect := firstDescendant(node, "srcRect"); srcRect != nil {
		element.CropLeft = parseIntAttr(srcRect.Attrs, "l")
		element.CropTop = parseIntAttr(srcRect.Attrs, "t")
		element.CropRight = parseIntAttr(srcRect.Attrs, "r")
		element.CropBottom = parseIntAttr(srcRect.Attrs, "b")
		element.HasCrop = element.CropLeft != 0 || element.CropTop != 0 || element.CropRight != 0 || element.CropBottom != 0
	}
	if spPr := firstChild(node, "spPr"); spPr != nil {
		parseShapeProperties(spPr, transform, &element, theme)
	} else if xfrm := firstChild(node, "xfrm"); xfrm != nil {
		parseTransform(xfrm, transform, &element)
	}
	if txXfrm := firstChild(node, "txXfrm"); txXfrm != nil {
		parseTextTransform(txXfrm, transform, &element)
	}
	if style := firstChild(node, "style"); style != nil {
		parseStyleProperties(style, &element, theme, effectStyles, fillStyles, lineStyles)
	}
	parseTextProperties(node, &element, theme)
	element.Text = strings.TrimSpace(textFromNode(node))
	element.TextParagraphs = textParagraphsFromNodeWithTheme(node, theme)
	element.PlaceholderParagraphStyles = paragraphStylesFromListStyle(firstDescendant(node, "lstStyle"), theme)
	if textParagraphsHaveRunColor(element.TextParagraphs) {
		element.HasTextColor = false
		element.TextColor = color.RGBA{}
	}
	return element
}

func parseCommonNonVisualDrawingProperties(cNvPr *xmlNode, element *slideElement) {
	element.ID = attrValue(cNvPr.Attrs, "id")
	element.Name = attrValue(cNvPr.Attrs, "name")
	element.Description = attrValue(cNvPr.Attrs, "descr")
	element.Title = attrValue(cNvPr.Attrs, "title")
	if creationID := firstDescendant(cNvPr, "creationId"); creationID != nil {
		element.CreationID = attrValue(creationID.Attrs, "id")
	}
	if value, ok := attrValueIfPresent(cNvPr.Attrs, "hidden"); ok {
		element.HasHidden = true
		element.Hidden = boolAttrOn(value)
		element.NonVisualProperties = append(element.NonVisualProperties, "hidden="+boolPropertyValue(element.Hidden))
	}
	if decorative := firstDescendant(cNvPr, "decorative"); decorative != nil {
		if value, ok := attrValueIfPresent(decorative.Attrs, "val"); ok {
			element.HasDecorative = true
			element.Decorative = boolAttrOn(value)
			element.NonVisualProperties = append(element.NonVisualProperties, "decorative="+boolPropertyValue(element.Decorative))
		}
	}
	sort.Strings(element.NonVisualProperties)
}

func nonVisualLockFlags(node *xmlNode) []string {
	var locks []string
	for _, child := range node.Children {
		switch child.Name {
		case "spLocks", "picLocks", "cxnSpLocks", "grpSpLocks", "graphicFrameLocks", "cpLocks":
			for _, attr := range child.Attrs {
				if boolAttrOn(attr.Value) {
					locks = append(locks, child.Name+"."+attr.Name.Local)
				}
			}
		}
	}
	sort.Strings(locks)
	return locks
}

func attrValueIfPresent(attrs []xml.Attr, local string) (string, bool) {
	for _, attr := range attrs {
		if attr.Name.Local == local {
			return attr.Value, true
		}
	}
	return "", false
}

func boolPropertyValue(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func parseNonBasicPayloadProperties(node *xmlNode, element *slideElement) {
	switch node.Name {
	case "graphicFrame":
		parseGraphicFramePayloadProperties(node, element)
	case "contentPart":
		element.GraphicPayloadKind = "content part"
		element.PayloadRelationshipID = attrValue(node.Attrs, "id")
	case "oleObj":
		element.GraphicPayloadKind = "OLE object"
		element.PayloadRelationshipID = attrValue(node.Attrs, "id")
		element.OLEProgID = attrValue(node.Attrs, "progId")
	case "control":
		element.GraphicPayloadKind = "control"
		element.PayloadRelationshipID = attrValue(node.Attrs, "id")
	case "audio", "audioFile":
		element.GraphicPayloadKind = "audio"
		element.PayloadRelationshipID = firstNonEmptyAttr(node.Attrs, "embed", "link", "id")
	case "video", "videoFile":
		element.GraphicPayloadKind = "video"
		element.PayloadRelationshipID = firstNonEmptyAttr(node.Attrs, "embed", "link", "id")
	}
}

func parseGraphicFramePayloadProperties(node *xmlNode, element *slideElement) {
	graphicData := firstDescendant(node, "graphicData")
	if graphicData == nil {
		return
	}
	element.GraphicPayloadURI = attrValue(graphicData.Attrs, "uri")
	if element.HasTable || element.DiagramDataID != "" {
		return
	}
	if firstChild(graphicData, "lockedCanvas") != nil {
		return
	}
	if chart := firstDescendant(graphicData, "chart"); chart != nil {
		element.GraphicPayloadKind = "chart"
		element.PayloadRelationshipID = attrValue(chart.Attrs, "id")
		return
	}
	if len(graphicData.Children) > 0 || element.GraphicPayloadURI != "" {
		element.GraphicPayloadKind = "unknown graphic payload"
	}
}

func firstNonEmptyAttr(attrs []xml.Attr, names ...string) string {
	for _, name := range names {
		if value := attrValue(attrs, name); value != "" {
			return value
		}
	}
	return ""
}

func parseBlipEffects(blip *xmlNode, element *slideElement) {
	parseBlipEffectsWithTheme(blip, element, defaultThemeColors())
}

func parseBlipEffectsWithTheme(blip *xmlNode, element *slideElement, theme themeColors) {
	for _, child := range blip.Children {
		switch child.Name {
		case "alphaBiLevel":
			element.HasImageAlphaBiLevel = true
			element.ImageAlphaBiLevelThreshold = parsePercentAttr(child.Attrs, "thresh")
		case "alphaCeiling":
			element.HasImageAlphaCeiling = true
		case "alphaFloor":
			element.HasImageAlphaFloor = true
		case "alphaInv":
			element.HasImageAlphaInverse = true
		case "alphaRepl":
			element.HasImageAlphaReplace = true
			element.ImageAlphaReplacePct = parsePercentAttr(child.Attrs, "a")
		case "biLevel":
			element.HasImageBiLevel = true
			element.ImageBiLevelThreshold = parsePercentAttr(child.Attrs, "thresh")
		case "clrChange":
			fromNode := firstChild(child, "clrFrom")
			toNode := firstChild(child, "clrTo")
			from, fromOK := colorFromBlipEffectColorNode(fromNode, theme)
			to, toOK := colorFromBlipEffectColorNode(toNode, theme)
			if fromOK && toOK {
				element.HasImageColorChange = true
				element.ImageColorChangeFrom = from
				element.ImageColorChangeTo = to
				element.ImageColorChangeUseAlpha = true
				if value := attrValue(child.Attrs, "useA"); value != "" {
					element.ImageColorChangeUseAlpha = boolAttrOn(value)
				}
			} else {
				element.ImageUnsupported = append(element.ImageUnsupported, "blip effect clrChange color choice was not resolved")
			}
		case "clrRepl":
			if replacement, ok := colorFromColorNodeWithTheme(child, theme); ok {
				element.HasImageColorReplace = true
				element.ImageColorReplace = replacement
			} else {
				element.ImageUnsupported = append(element.ImageUnsupported, "blip effect clrRepl color choice was not resolved")
			}
		case "duotone":
			colors := blipEffectColorChoices(child, theme)
			if len(colors) == 2 {
				element.HasImageDuotone = true
				element.ImageDuotoneDark = colors[0]
				element.ImageDuotoneLight = colors[1]
			} else {
				element.ImageUnsupported = append(element.ImageUnsupported, "blip effect duotone color choices were not resolved")
			}
		case "grayscl":
			element.HasImageGrayscale = true
		case "lum":
			element.HasImageLuminance = true
			element.ImageLuminanceBright = parsePercentAttr(child.Attrs, "bright")
			element.ImageLuminanceContrast = parsePercentAttr(child.Attrs, "contrast")
		case "hsl":
			element.HasImageHSL = true
			element.ImageHSLHue = parseIntAttr(child.Attrs, "hue")
			element.ImageHSLSaturation = parsePercentAttr(child.Attrs, "sat")
			element.ImageHSLLuminance = parsePercentAttr(child.Attrs, "lum")
		case "tint":
			element.HasImageTint = true
			element.ImageTintHue = parseIntAttr(child.Attrs, "hue")
			element.ImageTintAmount = parsePercentAttr(child.Attrs, "amt")
		case "blur":
			if radius := parseIntAttr(child.Attrs, "rad"); radius > 0 {
				element.HasImageBlur = true
				element.ImageBlurRadius = radius
				element.ImageBlurGrow = true
				if grow := attrValue(child.Attrs, "grow"); grow != "" {
					element.ImageBlurGrow = boolAttrOn(grow)
				}
			}
		case "fillOverlay":
			if paint, ok := fillPaintFromContainer(child, theme, nil); ok {
				element.HasImageFillOverlay = true
				element.ImageFillOverlay = paint
				element.ImageFillOverlayBlend = attrValue(child.Attrs, "blend")
				if element.ImageFillOverlayBlend == "" {
					element.ImageFillOverlayBlend = "over"
				}
			} else {
				element.ImageUnsupported = append(element.ImageUnsupported, "blip effect fillOverlay fill was not resolved")
			}
		case "alphaModFix":
			parseAlphaModFixEffect(child, element)
		case "alphaMod":
			parseAlphaModulateEffect(child, element)
		}
	}
}

func parseAlphaModFixEffect(alphaModFix *xmlNode, element *slideElement) {
	element.HasImageAlphaModFix = true
	if attrValue(alphaModFix.Attrs, "amt") == "" {
		element.ImageAlphaModFixPct = 100000
	} else {
		element.ImageAlphaModFixPct = parseIntAttr(alphaModFix.Attrs, "amt")
	}
}

func parseAlphaModulateEffect(alphaMod *xmlNode, element *slideElement) {
	amount, ok := alphaModulateEffectAmount(alphaMod)
	if !ok {
		element.ImageUnsupported = append(element.ImageUnsupported, "blip effect alphaMod container was not rendered")
		return
	}
	element.HasImageAlphaModulate = true
	element.ImageAlphaModulatePct = amount
}

func alphaModulateEffectAmount(alphaMod *xmlNode) (int64, bool) {
	cont := firstChild(alphaMod, "cont")
	if cont == nil {
		return 0, false
	}
	return alphaModulateContainerAmount(cont)
}

func alphaModulateContainerAmount(cont *xmlNode) (int64, bool) {
	amount := int64(100000)
	for _, child := range cont.Children {
		var childAmount int64
		switch child.Name {
		case "alphaModFix":
			childAmount = parseAlphaModFixAmount(child)
		case "alphaMod":
			nextAmount, ok := alphaModulateEffectAmount(child)
			if !ok {
				return 0, false
			}
			childAmount = nextAmount
		case "cont":
			nextAmount, ok := alphaModulateContainerAmount(child)
			if !ok {
				return 0, false
			}
			childAmount = nextAmount
		default:
			return 0, false
		}
		amount = amount * childAmount / 100000
	}
	return amount, true
}

func parseAlphaModFixAmount(alphaModFix *xmlNode) int64 {
	if attrValue(alphaModFix.Attrs, "amt") == "" {
		return 100000
	}
	return parseIntAttr(alphaModFix.Attrs, "amt")
}

func parseBlipFillMode(blipFill *xmlNode, element *slideElement) {
	if tile := firstChild(blipFill, "tile"); tile != nil {
		element.BlipFillMode = "tile"
		element.BlipTileOffsetX = parseIntAttr(tile.Attrs, "tx")
		element.BlipTileOffsetY = parseIntAttr(tile.Attrs, "ty")
		element.BlipTileScaleX = parsePercentAttr(tile.Attrs, "sx")
		element.BlipTileScaleY = parsePercentAttr(tile.Attrs, "sy")
		element.BlipTileFlip = attrValue(tile.Attrs, "flip")
		element.BlipTileAlignment = attrValue(tile.Attrs, "algn")
		return
	}
	if firstChild(blipFill, "stretch") != nil {
		element.BlipFillMode = "stretch"
	}
}

func colorFromBlipEffectColorNode(node *xmlNode, theme themeColors) (color.RGBA, bool) {
	if node == nil {
		return color.RGBA{}, false
	}
	return colorFromColorNodeWithTheme(node, theme)
}

func blipEffectColorChoices(node *xmlNode, theme themeColors) []color.RGBA {
	var colors []color.RGBA
	for _, child := range node.Children {
		if c, ok := colorFromColorNodeWithTheme(&xmlNode{Children: []*xmlNode{child}}, theme); ok {
			colors = append(colors, c)
		}
	}
	return colors
}

func composeGroupTransform(parent renderTransform, group *xmlNode, theme themeColors) renderTransform {
	next := parent
	if grpSpPr := firstChild(group, "grpSpPr"); grpSpPr != nil {
		if paint, ok := fillPaintFromContainer(grpSpPr, theme, parent.GroupFill); ok {
			next.GroupFill = &paint
		}
	}
	xfrm := firstDescendant(group, "xfrm")
	if xfrm == nil {
		return next
	}
	off := firstChild(xfrm, "off")
	ext := firstChild(xfrm, "ext")
	chOff := firstChild(xfrm, "chOff")
	chExt := firstChild(xfrm, "chExt")
	if off == nil || ext == nil || chOff == nil || chExt == nil {
		return next
	}
	childExtX := parseIntAttr(chExt.Attrs, "cx")
	childExtY := parseIntAttr(chExt.Attrs, "cy")
	if childExtX == 0 || childExtY == 0 {
		return next
	}
	scaleX := float64(parseIntAttr(ext.Attrs, "cx")) / float64(childExtX)
	scaleY := float64(parseIntAttr(ext.Attrs, "cy")) / float64(childExtY)
	childOffX := parseIntAttr(chOff.Attrs, "x")
	childOffY := parseIntAttr(chOff.Attrs, "y")
	offX := parseIntAttr(off.Attrs, "x")
	offY := parseIntAttr(off.Attrs, "y")
	next.ScaleX = parent.ScaleX * scaleX
	next.ScaleY = parent.ScaleY * scaleY
	next.OffsetX = parent.OffsetX + parent.ScaleX*(float64(offX)-float64(childOffX)*scaleX)
	next.OffsetY = parent.OffsetY + parent.ScaleY*(float64(offY)-float64(childOffY)*scaleY)
	return next
}

func composeGraphicFrameContentTransform(parent renderTransform, frame *xmlNode) renderTransform {
	next := parent
	xfrm := firstChild(frame, "xfrm")
	if xfrm == nil {
		return next
	}
	off := firstChild(xfrm, "off")
	if off == nil {
		return next
	}
	next.OffsetX = parent.OffsetX + parent.ScaleX*float64(parseIntAttr(off.Attrs, "x"))
	next.OffsetY = parent.OffsetY + parent.ScaleY*float64(parseIntAttr(off.Attrs, "y"))
	return next
}

func transformCoord(value int64, scale float64, offset float64) int64 {
	return int64(math.Round(float64(value)*scale + offset))
}

func transformLength(value int64, scale float64) int64 {
	return int64(math.Round(float64(value) * scale))
}
