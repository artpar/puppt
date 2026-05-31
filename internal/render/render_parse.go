package render

import (
	"bytes"
	"encoding/xml"
	"errors"
	"image/color"
	"math"
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
	return collectElementsFromNode(root, renderTransform{ScaleX: 1, ScaleY: 1}, theme, effectStyles, fillStyles, lineStyles)
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
		case "sp", "cxnSp", "pic", "graphicFrame":
			element := parseSlideElementNodeWithThemeEffectsAndFills(child, transform, theme, effectStyles, fillStyles, lineStyles)
			elements = append(elements, element)
		case "grpSp":
			elements = append(elements, collectElementsFromNode(child, composeGroupTransform(transform, child), theme, effectStyles, fillStyles, lineStyles)...)
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
		element.ID = attrValue(cNvPr.Attrs, "id")
		element.Name = attrValue(cNvPr.Attrs, "name")
	}
	if ph := firstDescendant(node, "ph"); ph != nil {
		element.IsPlaceholder = true
		element.PlaceholderType = attrValue(ph.Attrs, "type")
		element.PlaceholderIdx = attrValue(ph.Attrs, "idx")
	}
	if blip := firstDescendant(node, "blip"); blip != nil {
		element.EmbedID = attrValue(blip.Attrs, "embed")
		parseBlipEffects(blip, &element)
	}
	if blipFill := firstDescendant(node, "blipFill"); blipFill != nil {
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

func parseBlipEffects(blip *xmlNode, element *slideElement) {
	if alphaModFix := firstChild(blip, "alphaModFix"); alphaModFix != nil {
		element.HasImageAlphaModFix = true
		if attrValue(alphaModFix.Attrs, "amt") == "" {
			element.ImageAlphaModFixPct = 100000
		} else if amount := parseIntAttr(alphaModFix.Attrs, "amt"); amount > 0 {
			element.ImageAlphaModFixPct = amount
		}
	}
}

func composeGroupTransform(parent renderTransform, group *xmlNode) renderTransform {
	xfrm := firstDescendant(group, "xfrm")
	if xfrm == nil {
		return parent
	}
	off := firstChild(xfrm, "off")
	ext := firstChild(xfrm, "ext")
	chOff := firstChild(xfrm, "chOff")
	chExt := firstChild(xfrm, "chExt")
	if off == nil || ext == nil || chOff == nil || chExt == nil {
		return parent
	}
	childExtX := parseIntAttr(chExt.Attrs, "cx")
	childExtY := parseIntAttr(chExt.Attrs, "cy")
	if childExtX == 0 || childExtY == 0 {
		return parent
	}
	scaleX := float64(parseIntAttr(ext.Attrs, "cx")) / float64(childExtX)
	scaleY := float64(parseIntAttr(ext.Attrs, "cy")) / float64(childExtY)
	childOffX := parseIntAttr(chOff.Attrs, "x")
	childOffY := parseIntAttr(chOff.Attrs, "y")
	offX := parseIntAttr(off.Attrs, "x")
	offY := parseIntAttr(off.Attrs, "y")
	return renderTransform{
		ScaleX:  parent.ScaleX * scaleX,
		ScaleY:  parent.ScaleY * scaleY,
		OffsetX: parent.OffsetX + parent.ScaleX*(float64(offX)-float64(childOffX)*scaleX),
		OffsetY: parent.OffsetY + parent.ScaleY*(float64(offY)-float64(childOffY)*scaleY),
	}
}

func transformCoord(value int64, scale float64, offset float64) int64 {
	return int64(math.Round(float64(value)*scale + offset))
}

func transformLength(value int64, scale float64) int64 {
	return int64(math.Round(float64(value) * scale))
}
