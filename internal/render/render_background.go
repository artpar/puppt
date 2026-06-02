package render

import (
	"bytes"
	"encoding/xml"
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/artpar/puppt/internal/pptx"
)

func inheritedRenderParts(pkg *pptx.Package, slidePart string) []string {
	var parts []string
	layoutPart := firstRelationshipTarget(pkg, slidePart, pptx.SlideLayoutRelType)
	masterPart := ""
	if layoutPart != "" {
		masterPart = firstRelationshipTarget(pkg, layoutPart, pptx.SlideMasterRelType)
	}
	for _, part := range []string{masterPart, layoutPart, slidePart} {
		if part == "" {
			continue
		}
		if _, ok := pkg.Parts[part]; ok {
			parts = append(parts, part)
		}
	}
	return parts
}

func visibleRenderParts(pkg *pptx.Package, slidePart string, parts []string) []string {
	if !layoutHidesMasterShapes(pkg, slidePart) {
		return parts
	}
	layoutPart := firstRelationshipTarget(pkg, slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return parts
	}
	masterPart := firstRelationshipTarget(pkg, layoutPart, pptx.SlideMasterRelType)
	if masterPart == "" {
		return parts
	}
	visible := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == masterPart {
			continue
		}
		visible = append(visible, part)
	}
	return visible
}

func layoutHidesMasterShapes(pkg *pptx.Package, slidePart string) bool {
	layoutPart := firstRelationshipTarget(pkg, slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return false
	}
	data, ok := pkg.Parts[layoutPart]
	if !ok {
		return false
	}
	root, err := parseXMLNode(data)
	if err != nil {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(attrValue(root.Attrs, "showMasterSp")))
	return value == "0" || value == "false"
}

func presentationShowsSpecialPlaceholdersOnTitleSlide(pkg *pptx.Package) bool {
	root, err := parseXMLNode(pkg.Parts[pkg.PresentationPath])
	if err != nil {
		return true
	}
	value := strings.TrimSpace(attrValue(root.Attrs, "showSpecialPlsOnTitleSld"))
	if value == "" {
		return true
	}
	return boolAttrOn(value)
}

func slideUsesTitleLayout(pkg *pptx.Package, slidePart string) bool {
	layoutPart := firstRelationshipTarget(pkg, slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return false
	}
	root, err := parseXMLNode(pkg.Parts[layoutPart])
	if err != nil {
		return false
	}
	return strings.TrimSpace(attrValue(root.Attrs, "type")) == "title"
}

func firstRelationshipTarget(pkg *pptx.Package, sourcePart string, relationshipType string) string {
	relationships, err := pkg.RelationshipsForPart(sourcePart)
	if err != nil {
		return ""
	}
	for _, relationship := range relationships {
		if relationship.Type != relationshipType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
			continue
		}
		return pptx.ResolveTargetPart(sourcePart, relationship.Target)
	}
	return ""
}

func themePartForRenderPart(pkg *pptx.Package, renderPart string) string {
	if strings.HasPrefix(renderPart, "ppt/slides/") {
		layoutPart := firstRelationshipTarget(pkg, renderPart, pptx.SlideLayoutRelType)
		if layoutPart == "" {
			return ""
		}
		renderPart = layoutPart
	}
	if strings.HasPrefix(renderPart, "ppt/slideLayouts/") {
		masterPart := firstRelationshipTarget(pkg, renderPart, pptx.SlideMasterRelType)
		if masterPart == "" {
			return ""
		}
		renderPart = masterPart
	}
	return firstRelationshipTarget(pkg, renderPart, themeRelType)
}

func colorMapForRenderPart(pkg *pptx.Package, renderPart string) map[string]string {
	if data, ok := pkg.Parts[renderPart]; ok {
		if mapping, ok := parseColorMapOverride(data); ok {
			return mapping
		}
	}
	if strings.HasPrefix(renderPart, "ppt/slideMasters/") {
		return parseMasterColorMap(pkg.Parts[renderPart])
	}
	if strings.HasPrefix(renderPart, "ppt/slides/") {
		layoutPart := firstRelationshipTarget(pkg, renderPart, pptx.SlideLayoutRelType)
		if layoutPart == "" {
			return nil
		}
		renderPart = layoutPart
	}
	if strings.HasPrefix(renderPart, "ppt/slideLayouts/") {
		if data, ok := pkg.Parts[renderPart]; ok {
			if mapping, ok := parseColorMapOverride(data); ok {
				return mapping
			}
		}
		masterPart := firstRelationshipTarget(pkg, renderPart, pptx.SlideMasterRelType)
		if masterPart == "" {
			return nil
		}
		return parseMasterColorMap(pkg.Parts[masterPart])
	}
	return nil
}

func inheritedBackground(pkg *pptx.Package, renderParts []string, theme themeColors) backgroundPaint {
	return inheritedBackgroundWithThemeResolver(pkg, renderParts, func(string) themeColors { return theme })
}

func inheritedBackgroundWithThemeResolver(pkg *pptx.Package, renderParts []string, themeForPart func(string) themeColors) backgroundPaint {
	background := backgroundPaint{Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}}
	for _, renderPart := range renderParts {
		partTheme := themeForPart(renderPart)
		resolveStyle := func(idx int64, placeholderColor color.RGBA) (backgroundPaint, bool) {
			return themeBackgroundFillForPart(pkg, renderPart, idx, placeholderColor, partTheme)
		}
		if paint, ok := parseSlideBackgroundPaintWithThemeAndResolver(pkg.Parts[renderPart], partTheme, resolveStyle); ok {
			paint.Part = renderPart
			background = paint
		}
	}
	return background
}

func parseSlideSize(data []byte) slideSize {
	size := slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return size
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "sldSz" {
			continue
		}
		for _, attr := range start.Attr {
			switch attr.Name.Local {
			case "cx":
				if value, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
					size.CX = value
				}
			case "cy":
				if value, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
					size.CY = value
				}
			}
		}
		return size
	}
}

func emuToPixels(value int64) int {
	return emuToPixelsAtDPI(value, defaultOutputDPI)
}

func emuToPixelsAtDPI(value int64, dpi int) int {
	return int(math.Round(float64(value) / emuPerInch * float64(normalizeOutputDPI(dpi))))
}

func normalizeOutputDPI(dpi int) int {
	if dpi <= 0 {
		return defaultOutputDPI
	}
	return dpi
}

func renderDPIForCanvas(size slideSize, canvas image.Rectangle) int {
	if size.CX <= 0 || canvas.Dx() <= 0 {
		return defaultOutputDPI
	}
	return normalizeOutputDPI(int(math.Round(float64(canvas.Dx()) * emuPerInch / float64(size.CX))))
}

func parseSlideBackground(data []byte) color.RGBA {
	if c, ok := parseSlideBackgroundColorWithTheme(data, defaultThemeColors()); ok {
		return c
	}
	return color.RGBA{R: 255, G: 255, B: 255, A: 255}
}

func parseSlideBackgroundColor(data []byte) (color.RGBA, bool) {
	return parseSlideBackgroundColorWithTheme(data, defaultThemeColors())
}

func parseSlideBackgroundColorWithTheme(data []byte, theme themeColors) (color.RGBA, bool) {
	paint, ok := parseSlideBackgroundPaintWithTheme(data, theme)
	if !ok || paint.HasGradient {
		return color.RGBA{}, false
	}
	return paint.Color, true
}

func parseSlideBackgroundPaintWithTheme(data []byte, theme themeColors) (backgroundPaint, bool) {
	return parseSlideBackgroundPaintWithThemeAndResolver(data, theme, nil)
}

func parseSlideBackgroundPaintWithThemeAndResolver(data []byte, theme themeColors, resolveStyle backgroundStyleResolver) (backgroundPaint, bool) {
	root, err := parseXMLNode(data)
	if err != nil {
		return backgroundPaint{}, false
	}
	return parseSlideBackgroundPaintFromRootWithThemeAndResolver(root, theme, resolveStyle)
}

func parseSlideBackgroundPaintFromRootWithThemeAndResolver(root *xmlNode, theme themeColors, resolveStyle backgroundStyleResolver) (backgroundPaint, bool) {
	background := firstDescendant(root, "bg")
	if background == nil {
		return backgroundPaint{}, false
	}
	if bgPr := firstChild(background, "bgPr"); bgPr != nil {
		if paint, ok := fillPaintFromContainer(bgPr, theme, nil); ok {
			return paint, true
		}
	}
	if resolveStyle != nil {
		if bgRef := firstChild(background, "bgRef"); bgRef != nil {
			if placeholderColor, ok := colorFromColorNodeWithTheme(bgRef, theme); ok {
				if paint, ok := resolveStyle(parseIntAttr(bgRef.Attrs, "idx"), placeholderColor); ok {
					return paint, true
				}
			}
		}
	}
	return backgroundPaint{}, false
}

func parseGradientFill(gradFill *xmlNode, theme themeColors) (gradientPaint, bool) {
	stopList := firstChild(gradFill, "gsLst")
	if stopList == nil {
		return gradientPaint{}, false
	}
	var stops []gradientStop
	for _, child := range stopList.Children {
		if child.Name != "gs" {
			continue
		}
		if c, ok := colorFromColorNodeWithTheme(child, theme); ok {
			stops = append(stops, gradientStop{
				Position: parseIntAttr(child.Attrs, "pos"),
				Color:    c,
			})
		}
	}
	if len(stops) < 2 {
		return gradientPaint{}, false
	}
	sort.Slice(stops, func(i, j int) bool {
		return stops[i].Position < stops[j].Position
	})
	gradient := gradientPaint{Stops: stops}
	if pathNode := firstChild(gradFill, "path"); pathNode != nil {
		gradient.Path = attrValue(pathNode.Attrs, "path")
		if fillToRect := firstChild(pathNode, "fillToRect"); fillToRect != nil {
			gradient.HasFillRect = true
			gradient.FillRect = relativeRect{
				Left:   parseIntAttr(fillToRect.Attrs, "l"),
				Top:    parseIntAttr(fillToRect.Attrs, "t"),
				Right:  parseIntAttr(fillToRect.Attrs, "r"),
				Bottom: parseIntAttr(fillToRect.Attrs, "b"),
			}
		}
	}
	if linNode := firstChild(gradFill, "lin"); linNode != nil {
		gradient.Angle = parseIntAttr(linNode.Attrs, "ang")
		gradient.HasAngle = attrValue(linNode.Attrs, "ang") != ""
		if value := attrValue(linNode.Attrs, "scaled"); value != "" {
			gradient.HasScaled = true
			gradient.Scaled = boolAttrOn(value)
		}
	}
	gradient.FullySupported = gradientFillIsFullySupported(gradFill, gradient)
	return gradient, true
}

func gradientFillIsFullySupported(gradFill *xmlNode, gradient gradientPaint) bool {
	if len(gradient.Stops) < 2 {
		return false
	}
	if flip := attrValue(gradFill.Attrs, "flip"); flip != "" && flip != "none" {
		return false
	}
	if tileRect := firstChild(gradFill, "tileRect"); tileRect != nil && len(tileRect.Attrs) > 0 {
		return false
	}
	if gradient.Path != "" && gradient.Path != "circle" && gradient.Path != "rect" {
		return false
	}
	if gradient.HasAngle || firstChild(gradFill, "lin") != nil {
		return true
	}
	return true
}

func parseHexColor(value string) (color.RGBA, bool) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(value) != 6 {
		return color.RGBA{}, false
	}
	parsed, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: uint8(parsed >> 16),
		G: uint8(parsed >> 8),
		B: uint8(parsed),
		A: 255,
	}, true
}
