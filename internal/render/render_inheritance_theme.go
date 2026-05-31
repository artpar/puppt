package render

import (
	"encoding/xml"
	"image/color"
	"sort"
	"strings"

	"github.com/artpar/puppt/internal/pptx"
)

func filterInheritedPlaceholders(elements []slideElement) []slideElement {
	return filterInheritedPlaceholdersForRender(elements, nil, defaultHeaderFooterSettings(), false)
}

func filterInheritedPlaceholdersForRender(elements []slideElement, sources map[string]slideElement, settings headerFooterSettings, keepHeaderFooter bool) []slideElement {
	filtered := make([]slideElement, 0, len(elements))
	for _, element := range elements {
		if element.IsPlaceholder {
			if keepHeaderFooter && headerFooterPlaceholderEnabled(element.PlaceholderType, settings) {
				filtered = append(filtered, resolveInheritedHeaderFooterPlaceholder(element, sources))
			}
			continue
		}
		filtered = append(filtered, element)
	}
	return filtered
}

func resolveInheritedHeaderFooterPlaceholder(element slideElement, sources map[string]slideElement) slideElement {
	if source, ok := placeholderSource(element, sources); ok {
		if strings.TrimSpace(element.Text) == "" {
			element.Text = source.Text
			element.TextParagraphs = cloneTextParagraphs(source.TextParagraphs)
		}
		merged := mergePlaceholderSource(source, element)
		applyParagraphStylesToElement(&merged, source.PlaceholderParagraphStyles)
		applyInheritedBodyBullets(&merged)
		return merged
	}
	return element
}

func inheritedHeaderFooterRenderPart(pkg *pptx.Package, paintParts []string, slidePart string, settings headerFooterSettings) string {
	for index := len(paintParts) - 1; index >= 0; index-- {
		part := paintParts[index]
		if part == slidePart {
			continue
		}
		elements := collectSlideElements(pkg.Parts[part])
		for _, element := range elements {
			if element.IsPlaceholder && headerFooterPlaceholderEnabled(element.PlaceholderType, settings) {
				return part
			}
		}
	}
	return ""
}

func defaultHeaderFooterSettings() headerFooterSettings {
	return headerFooterSettings{}
}

func inheritedHeaderFooterSettings(pkg *pptx.Package, renderParts []string) headerFooterSettings {
	settings := defaultHeaderFooterSettings()
	for _, part := range renderParts {
		partSettings := parseHeaderFooterSettings(pkg.Parts[part])
		if partSettings.HasSlideNumber {
			settings.HasSlideNumber = true
			settings.SlideNumber = partSettings.SlideNumber
		}
		if partSettings.HasDateTime {
			settings.HasDateTime = true
			settings.DateTime = partSettings.DateTime
		}
		if partSettings.HasFooter {
			settings.HasFooter = true
			settings.Footer = partSettings.Footer
		}
		if partSettings.HasHeader {
			settings.HasHeader = true
			settings.Header = partSettings.Header
		}
	}
	return settings
}

func parseHeaderFooterSettings(data []byte) headerFooterSettings {
	root, err := parseXMLNode(data)
	if err != nil {
		return headerFooterSettings{}
	}
	hf := firstDescendant(root, "hf")
	if hf == nil {
		return headerFooterSettings{}
	}
	settings := headerFooterSettings{
		SlideNumber:    true,
		HasSlideNumber: true,
		DateTime:       true,
		HasDateTime:    true,
		Footer:         true,
		HasFooter:      true,
		Header:         true,
		HasHeader:      true,
	}
	if value := attrValue(hf.Attrs, "sldNum"); value != "" {
		settings.SlideNumber = boolAttrOn(value)
	}
	if value := attrValue(hf.Attrs, "dt"); value != "" {
		settings.DateTime = boolAttrOn(value)
	}
	if value := attrValue(hf.Attrs, "ftr"); value != "" {
		settings.Footer = boolAttrOn(value)
	}
	if value := attrValue(hf.Attrs, "hdr"); value != "" {
		settings.Header = boolAttrOn(value)
	}
	return settings
}

func headerFooterPlaceholderEnabled(placeholderType string, settings headerFooterSettings) bool {
	switch placeholderType {
	case "sldNum":
		return settings.SlideNumber
	case "dt":
		return settings.DateTime
	case "ftr":
		return settings.Footer
	case "hdr":
		return settings.Header
	default:
		return false
	}
}

func inheritedPlaceholderSources(pkg *pptx.Package, renderParts []string, slidePart string, theme themeColors) map[string]slideElement {
	return inheritedPlaceholderSourcesWithThemeResolver(pkg, renderParts, slidePart, func(string) themeColors { return theme })
}

func inheritedPlaceholderSourcesWithThemeResolver(pkg *pptx.Package, renderParts []string, slidePart string, themeForPart func(string) themeColors) map[string]slideElement {
	sources := make(map[string]slideElement)
	for _, renderPart := range renderParts {
		if renderPart == slidePart {
			continue
		}
		for _, element := range collectSlideElementsWithTheme(pkg.Parts[renderPart], themeForPart(renderPart)) {
			for _, key := range placeholderKeys(element) {
				if existing, ok := sources[key]; ok {
					sources[key] = mergePlaceholderSource(existing, element)
				} else {
					sources[key] = element
				}
			}
		}
	}
	return sources
}

func mergePlaceholderSource(base slideElement, override slideElement) slideElement {
	merged := override
	if !merged.HasTransform || merged.ExtCX <= 0 || merged.ExtCY <= 0 {
		merged.HasTransform = base.HasTransform
		merged.OffX = base.OffX
		merged.OffY = base.OffY
		merged.ExtCX = base.ExtCX
		merged.ExtCY = base.ExtCY
	}
	if merged.PrstGeom == "" {
		merged.PrstGeom = base.PrstGeom
	}
	inheritPlaceholderVisualProperties(&merged, base)
	if !merged.HasInsets {
		merged.HasInsets = base.HasInsets
		merged.InsetLeft = base.InsetLeft
		merged.InsetTop = base.InsetTop
		merged.InsetRight = base.InsetRight
		merged.InsetBottom = base.InsetBottom
	}
	if merged.TextAlign == "" {
		merged.TextAlign = base.TextAlign
	}
	if shouldInheritPlaceholderTextAnchor(merged) && base.TextAnchor != "" {
		merged.TextAnchor = base.TextAnchor
	} else if shouldDefaultCenterTitleTextAnchor(merged) {
		merged.TextAnchor = "ctr"
	}
	inheritPlaceholderBodyTextProperties(&merged, base)
	if !merged.HasFirstLastSpacing && !merged.IncludeFirstLastSpacing {
		merged.IncludeFirstLastSpacing = base.IncludeFirstLastSpacing
		merged.HasFirstLastSpacing = base.HasFirstLastSpacing
	}
	if !merged.HasBodyProperties && !merged.HasShapeAutofit {
		merged.HasShapeAutofit = base.HasShapeAutofit
	}
	if !merged.HasBodyProperties && merged.FontScalePct == 0 {
		merged.FontScalePct = base.FontScalePct
		merged.HasFontScalePct = base.HasFontScalePct
	}
	if !merged.HasBodyProperties && !merged.HasNormAutofit {
		merged.HasNormAutofit = base.HasNormAutofit
	}
	if !merged.HasBodyProperties && merged.LineSpacingReductionPct == 0 {
		merged.LineSpacingReductionPct = base.LineSpacingReductionPct
		merged.HasLineSpacingReductionPct = base.HasLineSpacingReductionPct
	}
	if merged.PlaceholderType == "" {
		merged.PlaceholderType = base.PlaceholderType
	}
	if merged.FontSize == 0 {
		merged.FontSize = base.FontSize
	}
	if !merged.HasTextColor {
		merged.HasTextColor = base.HasTextColor
		merged.TextColor = base.TextColor
	}
	merged.PlaceholderParagraphStyles = mergeParagraphStyleMaps(base.PlaceholderParagraphStyles, merged.PlaceholderParagraphStyles)
	return merged
}

func inheritedTextStyles(pkg *pptx.Package, renderParts []string, slidePart string, theme themeColors) map[string]textStyle {
	return inheritedTextStylesWithThemeResolver(pkg, renderParts, slidePart, func(string) themeColors { return theme })
}

func inheritedTextStylesWithThemeResolver(pkg *pptx.Package, renderParts []string, slidePart string, themeForPart func(string) themeColors) map[string]textStyle {
	styles := presentationDefaultTextStyles(pkg, themeForPart(pkg.PresentationPath))
	for _, renderPart := range renderParts {
		if renderPart == slidePart {
			continue
		}
		for key, style := range parseTextStyles(pkg.Parts[renderPart], themeForPart(renderPart)) {
			styles[key] = mergeTextStyle(styles[key], style)
		}
	}
	return styles
}

func presentationDefaultTextStyles(pkg *pptx.Package, theme themeColors) map[string]textStyle {
	styles := map[string]textStyle{}
	if pkg == nil || pkg.PresentationPath == "" {
		return styles
	}
	style, ok := parsePresentationDefaultTextStyle(pkg.Parts[pkg.PresentationPath], theme)
	if !ok {
		return styles
	}
	styles["default"] = style
	return styles
}

func parsePresentationDefaultTextStyle(data []byte, theme themeColors) (textStyle, bool) {
	root, err := parseXMLNode(data)
	if err != nil {
		return textStyle{}, false
	}
	defaultTextStyle := firstDescendant(root, "defaultTextStyle")
	if defaultTextStyle == nil {
		return textStyle{}, false
	}
	return parseTextStyle(defaultTextStyle, theme)
}

func mergeTextStyle(base textStyle, override textStyle) textStyle {
	merged := override
	if merged.FontSize == 0 {
		merged.FontSize = base.FontSize
	}
	if !merged.HasBold && !merged.Bold {
		merged.HasBold = base.HasBold
		merged.Bold = base.Bold
	}
	if !merged.HasTextColor {
		merged.HasTextColor = base.HasTextColor
		merged.TextColor = base.TextColor
	}
	if merged.TextAlign == "" {
		merged.TextAlign = base.TextAlign
	}
	merged.ParagraphStyles = mergeParagraphStyleMaps(base.ParagraphStyles, merged.ParagraphStyles)
	return merged
}

func parseTextStyles(data []byte, theme themeColors) map[string]textStyle {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	txStyles := firstDescendant(root, "txStyles")
	if txStyles == nil {
		return nil
	}
	styles := map[string]textStyle{}
	if title := firstChild(txStyles, "titleStyle"); title != nil {
		if style, ok := parseTextStyle(title, theme); ok {
			styles["title"] = style
			styles["ctrTitle"] = style
		}
	}
	if body := firstChild(txStyles, "bodyStyle"); body != nil {
		if style, ok := parseTextStyle(body, theme); ok {
			styles["body"] = style
		}
	}
	if other := firstChild(txStyles, "otherStyle"); other != nil {
		if style, ok := parseTextStyle(other, theme); ok {
			styles["default"] = style
		}
	}
	return styles
}

func parseTextStyle(styleNode *xmlNode, theme themeColors) (textStyle, bool) {
	style := textStyle{ParagraphStyles: paragraphStylesFromListStyle(styleNode, theme)}
	paragraphProperties := firstLevelParagraphProperties(styleNode)
	if paragraphProperties == nil {
		return style, len(style.ParagraphStyles) > 0
	}
	style.TextAlign = attrValue(paragraphProperties.Attrs, "algn")
	if defRPr := firstDescendant(paragraphProperties, "defRPr"); defRPr != nil {
		if size := parseIntAttr(defRPr.Attrs, "sz"); size > 0 {
			style.FontSize = int(size)
		}
		if value := attrValue(defRPr.Attrs, "b"); value != "" {
			style.HasBold = true
			style.Bold = boolAttrOn(value)
		}
		if solidFill := firstChild(defRPr, "solidFill"); solidFill != nil {
			if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				style.HasTextColor = true
				style.TextColor = textColor
			}
		}
	}
	return style, style.FontSize > 0 || style.HasBold || style.HasTextColor || style.TextAlign != "" || len(style.ParagraphStyles) > 0
}

func firstLevelParagraphProperties(styleNode *xmlNode) *xmlNode {
	for _, child := range styleNode.Children {
		if child.Name == "defPPr" || (strings.HasPrefix(child.Name, "lvl") && strings.HasSuffix(child.Name, "pPr")) {
			return child
		}
	}
	return nil
}

func resolveSlidePlaceholders(elements []slideElement, sources map[string]slideElement) []slideElement {
	for index := range elements {
		element := &elements[index]
		if !element.IsPlaceholder {
			continue
		}
		source, ok := placeholderSource(*element, sources)
		if !ok {
			continue
		}
		if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
			element.HasTransform = source.HasTransform
			element.OffX = source.OffX
			element.OffY = source.OffY
			element.ExtCX = source.ExtCX
			element.ExtCY = source.ExtCY
		}
		if element.PrstGeom == "" {
			element.PrstGeom = source.PrstGeom
		}
		inheritPlaceholderVisualProperties(element, source)
		if !element.HasInsets {
			element.HasInsets = source.HasInsets
			element.InsetLeft = source.InsetLeft
			element.InsetTop = source.InsetTop
			element.InsetRight = source.InsetRight
			element.InsetBottom = source.InsetBottom
		}
		if element.TextAlign == "" {
			element.TextAlign = source.TextAlign
		}
		if shouldInheritPlaceholderTextAnchor(*element) && source.TextAnchor != "" {
			element.TextAnchor = source.TextAnchor
		} else if shouldDefaultCenterTitleTextAnchor(*element) {
			element.TextAnchor = "ctr"
		}
		inheritPlaceholderBodyTextProperties(element, source)
		if !element.HasFirstLastSpacing && !element.IncludeFirstLastSpacing {
			element.IncludeFirstLastSpacing = source.IncludeFirstLastSpacing
			element.HasFirstLastSpacing = source.HasFirstLastSpacing
		}
		if !element.HasBodyProperties && !element.HasShapeAutofit {
			element.HasShapeAutofit = source.HasShapeAutofit
		}
		if !element.HasBodyProperties && element.FontScalePct == 0 {
			element.FontScalePct = source.FontScalePct
			element.HasFontScalePct = source.HasFontScalePct
		}
		if !element.HasBodyProperties && !element.HasNormAutofit {
			element.HasNormAutofit = source.HasNormAutofit
		}
		if !element.HasBodyProperties && element.LineSpacingReductionPct == 0 {
			element.LineSpacingReductionPct = source.LineSpacingReductionPct
			element.HasLineSpacingReductionPct = source.HasLineSpacingReductionPct
		}
		if element.PlaceholderType == "" {
			element.PlaceholderType = source.PlaceholderType
		}
		if element.FontSize == 0 {
			element.FontSize = source.FontSize
		}
		if !element.HasTextColor {
			element.HasTextColor = source.HasTextColor
			element.TextColor = source.TextColor
		}
		applyParagraphStylesToElement(element, source.PlaceholderParagraphStyles)
		applyInheritedBodyBullets(element)
	}
	return elements
}

func inheritPlaceholderVisualProperties(element *slideElement, source slideElement) {
	if element == nil {
		return
	}
	if !element.HasFill && !element.NoFill {
		element.HasFill = source.HasFill
		element.FillColor = source.FillColor
		element.HasFillGradient = source.HasFillGradient
		element.FillGradient = source.FillGradient
		element.NoFill = source.NoFill
	}
	if !element.HasLine && !element.NoLine {
		element.HasLine = source.HasLine
		element.LineColor = source.LineColor
		element.HasLineWidth = source.HasLineWidth
		element.LineWidth = source.LineWidth
		element.HasLineDash = source.HasLineDash
		element.LineDash = source.LineDash
		element.HasLineCap = source.HasLineCap
		element.LineCap = source.LineCap
		element.HasLineAlign = source.HasLineAlign
		element.LineAlign = source.LineAlign
		element.NoLine = source.NoLine
	}
	if !element.HasShadow && !element.HasEffectProperties {
		element.HasShadow = source.HasShadow
		element.ShadowColor = source.ShadowColor
		element.ShadowBlur = source.ShadowBlur
		element.ShadowDistance = source.ShadowDistance
		element.ShadowDirection = source.ShadowDirection
		element.ShadowAlignment = source.ShadowAlignment
		element.HasShadowRotateWithShape = source.HasShadowRotateWithShape
		element.ShadowRotateWithShape = source.ShadowRotateWithShape
		element.HasShadowScaleX = source.HasShadowScaleX
		element.ShadowScaleX = source.ShadowScaleX
		element.HasShadowScaleY = source.HasShadowScaleY
		element.ShadowScaleY = source.ShadowScaleY
		element.HasShadowSkewX = source.HasShadowSkewX
		element.ShadowSkewX = source.ShadowSkewX
		element.HasShadowSkewY = source.HasShadowSkewY
		element.ShadowSkewY = source.ShadowSkewY
		element.HasEffectProperties = source.HasEffectProperties
		element.HasSoftEdge = source.HasSoftEdge
		element.SoftEdgeRadius = source.SoftEdgeRadius
		element.HasShape3D = source.HasShape3D
		element.Shape3DFeatures = append([]string{}, source.Shape3DFeatures...)
	}
}

func inheritPlaceholderBodyTextProperties(element *slideElement, source slideElement) {
	if element == nil {
		return
	}
	if !element.HasTextWrap {
		element.HasTextWrap = source.HasTextWrap
		element.TextWrap = source.TextWrap
	}
	if !element.HasTextHorizontalOverflow {
		element.HasTextHorizontalOverflow = source.HasTextHorizontalOverflow
		element.TextHorizontalOverflow = source.TextHorizontalOverflow
	}
	if !element.HasTextVerticalOverflow {
		element.HasTextVerticalOverflow = source.HasTextVerticalOverflow
		element.TextVerticalOverflow = source.TextVerticalOverflow
	}
	if !element.HasTextVertical {
		element.HasTextVertical = source.HasTextVertical
		element.TextVertical = source.TextVertical
	}
	if !element.HasTextBodyRotation {
		element.HasTextBodyRotation = source.HasTextBodyRotation
		element.TextBodyRotation = source.TextBodyRotation
	}
	if !element.HasTextColumns {
		element.HasTextColumns = source.HasTextColumns
		element.TextColumnCount = source.TextColumnCount
	}
	if !element.HasTextAnchorCenter {
		element.HasTextAnchorCenter = source.HasTextAnchorCenter
		element.TextAnchorCenter = source.TextAnchorCenter
	}
}

func mergeParagraphStyleMaps(base map[int]paragraphStyle, override map[int]paragraphStyle) map[int]paragraphStyle {
	if len(base) == 0 && len(override) == 0 {
		return nil
	}
	merged := make(map[int]paragraphStyle, len(base)+len(override))
	for level, style := range base {
		merged[level] = style
	}
	for level, style := range override {
		merged[level] = mergeParagraphStyle(merged[level], style)
	}
	return merged
}

func mergeParagraphStyle(base paragraphStyle, override paragraphStyle) paragraphStyle {
	merged := override
	if !merged.HasMarginLeft {
		merged.HasMarginLeft = base.HasMarginLeft
		merged.MarginLeft = base.MarginLeft
	}
	if !merged.HasMarginRight {
		merged.HasMarginRight = base.HasMarginRight
		merged.MarginRight = base.MarginRight
	}
	if !merged.HasIndent {
		merged.HasIndent = base.HasIndent
		merged.Indent = base.Indent
	}
	if merged.FontFamily == "" {
		merged.FontFamily = base.FontFamily
	}
	if merged.FontSize == 0 {
		merged.FontSize = base.FontSize
	}
	if !merged.HasSpaceBefore {
		merged.HasSpaceBefore = base.HasSpaceBefore
		merged.SpaceBefore = base.SpaceBefore
		merged.SpaceBeforePct = base.SpaceBeforePct
	}
	if !merged.HasSpaceAfter {
		merged.HasSpaceAfter = base.HasSpaceAfter
		merged.SpaceAfter = base.SpaceAfter
		merged.SpaceAfterPct = base.SpaceAfterPct
	}
	if !merged.HasLineSpacing {
		merged.HasLineSpacing = base.HasLineSpacing
		merged.LineSpacingPct = base.LineSpacingPct
	}
	if !merged.HasDefaultTab {
		merged.HasDefaultTab = base.HasDefaultTab
		merged.DefaultTabSize = base.DefaultTabSize
	}
	if merged.Bullet == "" && !merged.NoBullet && !merged.HasAutoNumber {
		merged.Bullet = base.Bullet
		merged.NoBullet = base.NoBullet
	}
	if !merged.HasAutoNumber && merged.Bullet == "" && !merged.NoBullet {
		merged.HasAutoNumber = base.HasAutoNumber
		merged.AutoNumberType = base.AutoNumberType
		merged.AutoNumberStart = base.AutoNumberStart
	}
	if merged.BulletFontFamily == "" {
		merged.BulletFontFamily = base.BulletFontFamily
	}
	if merged.BulletFontTx {
		merged.BulletFontFamily = ""
	} else if merged.BulletFontFamily == "" {
		merged.BulletFontTx = base.BulletFontTx
	}
	if merged.BulletSizeTx {
		merged.BulletFontSize = 0
		merged.BulletSizePct = 0
	} else if merged.BulletFontSize == 0 {
		merged.BulletFontSize = base.BulletFontSize
	}
	if merged.BulletSizeTx {
		// Local buSzTx blocks inherited fixed or percentage bullet sizing.
	} else if merged.BulletSizePct == 0 {
		merged.BulletSizePct = base.BulletSizePct
	}
	if base.BulletSizeTx && merged.BulletFontSize == 0 && merged.BulletSizePct == 0 {
		merged.BulletSizeTx = true
	}
	if !merged.HasBulletColor {
		merged.HasBulletColor = base.HasBulletColor
		merged.BulletColor = base.BulletColor
	}
	if merged.BulletColorTx {
		merged.HasBulletColor = false
		merged.BulletColor = color.RGBA{}
	} else if !merged.HasBulletColor {
		merged.BulletColorTx = base.BulletColorTx
	}
	if !merged.HasBold && !merged.Bold {
		merged.HasBold = base.HasBold
		merged.Bold = base.Bold
	}
	if !merged.HasItalic && !merged.Italic {
		merged.HasItalic = base.HasItalic
		merged.Italic = base.Italic
	}
	if !merged.HasCharSpacing {
		merged.HasCharSpacing = base.HasCharSpacing
		merged.CharSpacing = base.CharSpacing
	}
	if merged.TextAlign == "" {
		merged.TextAlign = base.TextAlign
	}
	if !merged.HasTextColor {
		merged.HasTextColor = base.HasTextColor
		merged.TextColor = base.TextColor
	}
	return merged
}

func applyInheritedBodyBullets(element *slideElement) {
	if !isBodyLikePlaceholder(*element) {
		return
	}
	for index := range element.TextParagraphs {
		if element.TextParagraphs[index].NoBullet || element.TextParagraphs[index].Bullet != "" {
			continue
		}
		element.TextParagraphs[index].Bullet = "•"
	}
}

func shouldInheritPlaceholderTextAnchor(element slideElement) bool {
	return element.TextAnchor == ""
}

func shouldDefaultCenterTitleTextAnchor(element slideElement) bool {
	return element.TextAnchor == "" && element.HasBodyProperties && element.PlaceholderType == "ctrTitle" && element.FontScalePct > 0
}

func isBodyLikePlaceholder(element slideElement) bool {
	if !element.IsPlaceholder {
		return false
	}
	if element.PlaceholderType == "body" {
		return true
	}
	if strings.Contains(strings.ToLower(element.Name), "content placeholder") {
		return true
	}
	return element.PlaceholderIdx == "1" && element.PlaceholderType == ""
}

func applyThemeFontFamilies(elements []slideElement, fonts themeFonts) []slideElement {
	for index := range elements {
		for paragraphIndex := range elements[index].TextParagraphs {
			paragraph := &elements[index].TextParagraphs[paragraphIndex]
			if family := resolveThemeTypeface(paragraph.FontFamily, fonts); family != "" {
				paragraph.FontFamily = family
			}
			if family := resolveThemeTypeface(paragraph.BulletFontFamily, fonts); family != "" {
				paragraph.BulletFontFamily = family
			}
			for runIndex := range elements[index].TextParagraphs[paragraphIndex].Runs {
				run := &paragraph.Runs[runIndex]
				if family := resolveThemeTypeface(run.FontFamily, fonts); family != "" {
					run.FontFamily = family
				}
			}
		}
		for rowIndex := range elements[index].Table.Rows {
			for cellIndex := range elements[index].Table.Rows[rowIndex].Cells {
				for paragraphIndex := range elements[index].Table.Rows[rowIndex].Cells[cellIndex].TextParagraphs {
					paragraph := &elements[index].Table.Rows[rowIndex].Cells[cellIndex].TextParagraphs[paragraphIndex]
					if family := resolveThemeTypeface(paragraph.FontFamily, fonts); family != "" {
						paragraph.FontFamily = family
					}
					if family := resolveThemeTypeface(paragraph.BulletFontFamily, fonts); family != "" {
						paragraph.BulletFontFamily = family
					}
					for runIndex := range paragraph.Runs {
						run := &paragraph.Runs[runIndex]
						if family := resolveThemeTypeface(run.FontFamily, fonts); family != "" {
							run.FontFamily = family
						}
					}
				}
			}
		}
		if elements[index].FontFamily != "" {
			if family := resolveThemeTypeface(elements[index].FontFamily, fonts); family != "" {
				elements[index].FontFamily = family
			}
			continue
		}
		if isTitleLikePlaceholder(elements[index]) && fonts.MajorLatin != "" {
			elements[index].FontFamily = fonts.MajorLatin
			continue
		}
		if fonts.MinorLatin != "" {
			elements[index].FontFamily = fonts.MinorLatin
		}
	}
	return elements
}

func resolveThemeTypeface(typeface string, fonts themeFonts) string {
	switch strings.ToLower(strings.TrimSpace(typeface)) {
	case "+mj-lt":
		return fonts.MajorLatin
	case "+mj-ea":
		return fonts.MajorEA
	case "+mj-cs":
		return fonts.MajorCS
	case "+mn-lt":
		return fonts.MinorLatin
	case "+mn-ea":
		return fonts.MinorEA
	case "+mn-cs":
		return fonts.MinorCS
	default:
		return ""
	}
}

func fontRefTypeface(idx string) string {
	switch strings.ToLower(strings.TrimSpace(idx)) {
	case "major":
		return "+mj-lt"
	case "minor":
		return "+mn-lt"
	default:
		return ""
	}
}

func isTitleLikePlaceholder(element slideElement) bool {
	return element.IsPlaceholder && (element.PlaceholderType == "title" || element.PlaceholderType == "ctrTitle")
}

func applyInheritedTextStyles(elements []slideElement, styles map[string]textStyle) []slideElement {
	for index := range elements {
		if elements[index].Text == "" {
			continue
		}
		if isBodyLikePlaceholder(elements[index]) {
			if style, ok := styles["body"]; ok {
				applyParagraphStylesToElement(&elements[index], style.ParagraphStyles)
			}
		}
		style, ok := inheritedTextStyleForElement(elements[index], styles)
		if !ok {
			continue
		}
		applyParagraphStylesToElement(&elements[index], style.ParagraphStyles)
		if elements[index].TextAlign == "" {
			elements[index].TextAlign = style.TextAlign
		}
		if elements[index].FontSize == 0 {
			elements[index].FontSize = style.FontSize
		}
		if !elements[index].HasTextColor && style.HasTextColor {
			elements[index].HasTextColor = true
			elements[index].TextColor = style.TextColor
		}
		if style.Bold {
			applyStyleBoldToParagraphs(&elements[index])
		}
	}
	return elements
}

func applyInheritedTableTextStyles(elements []slideElement, styles map[string]textStyle) []slideElement {
	style, ok := styles["default"]
	if !ok {
		return elements
	}
	for elementIndex := range elements {
		if !elements[elementIndex].HasTable {
			continue
		}
		for rowIndex := range elements[elementIndex].Table.Rows {
			for cellIndex := range elements[elementIndex].Table.Rows[rowIndex].Cells {
				cell := &elements[elementIndex].Table.Rows[rowIndex].Cells[cellIndex]
				applyParagraphStylesToTableCell(cell, style.ParagraphStyles)
				if !cell.HasFontSize && style.FontSize > 0 {
					cell.FontSize = style.FontSize
				}
			}
		}
	}
	return elements
}

func applyParagraphStylesToTableCell(cell *tableCell, styles map[int]paragraphStyle) {
	if cell == nil || len(styles) == 0 {
		return
	}
	for index := range cell.TextParagraphs {
		style, ok := styles[cell.TextParagraphs[index].Level]
		if !ok {
			continue
		}
		applyParagraphStyle(&cell.TextParagraphs[index], style)
	}
	if cell.TextAlign == "" {
		cell.TextAlign = textParagraphsTextAlign(cell.TextParagraphs)
	}
}

func applyParagraphStylesToElement(element *slideElement, styles map[int]paragraphStyle) {
	if len(styles) == 0 {
		return
	}
	for index := range element.TextParagraphs {
		style, ok := styles[element.TextParagraphs[index].Level]
		if !ok {
			continue
		}
		applyParagraphStyle(&element.TextParagraphs[index], style)
	}
}

func applyStyleBoldToParagraphs(element *slideElement) {
	if len(element.TextParagraphs) == 0 && strings.TrimSpace(element.Text) != "" {
		element.TextParagraphs = []textParagraph{{Text: strings.TrimSpace(element.Text), HasBold: true, Bold: true}}
		return
	}
	for index := range element.TextParagraphs {
		element.TextParagraphs[index].HasBold = true
		element.TextParagraphs[index].Bold = true
	}
}

func inheritedTextStyleForElement(element slideElement, styles map[string]textStyle) (textStyle, bool) {
	for _, key := range placeholderKeys(element) {
		if strings.HasPrefix(key, "type:") {
			placeholderType := strings.TrimPrefix(key, "type:")
			if placeholderType != "ctrTitle" && placeholderType != "title" {
				continue
			}
			if style, ok := styles[placeholderType]; ok {
				return style, true
			}
		}
	}
	return textStyle{}, false
}

func placeholderKey(element slideElement) string {
	keys := placeholderKeys(element)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func placeholderSource(element slideElement, sources map[string]slideElement) (slideElement, bool) {
	for _, key := range placeholderKeys(element) {
		source, ok := sources[key]
		if ok {
			return source, true
		}
	}
	return slideElement{}, false
}

func placeholderKeys(element slideElement) []string {
	var keys []string
	if element.PlaceholderType != "" {
		keys = append(keys, "type:"+element.PlaceholderType)
	}
	if element.PlaceholderIdx != "" {
		keys = append(keys, "idx:"+element.PlaceholderIdx)
	}
	return keys
}

func packageThemeColors(pkg *pptx.Package) themeColors {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if colors := parseThemeColors(pkg.Parts[part]); len(colors) > 0 {
			return colors
		}
	}
	return defaultThemeColors()
}

func packageThemeFonts(pkg *pptx.Package) themeFonts {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if fonts := parseThemeFonts(pkg.Parts[part]); fonts.MajorLatin != "" || fonts.MinorLatin != "" {
			return fonts
		}
	}
	return themeFonts{}
}

func themeColorsForPart(pkg *pptx.Package, renderPart string, fallback themeColors) themeColors {
	themePart := themePartForRenderPart(pkg, renderPart)
	var colors themeColors
	if themePart == "" {
		colors = fallback
	} else if parsed := parseThemeColors(pkg.Parts[themePart]); len(parsed) > 0 {
		colors = parsed
	} else {
		colors = fallback
	}
	if mapped := applyThemeColorMap(colors, colorMapForRenderPart(pkg, renderPart)); len(mapped) > 0 {
		return mapped
	}
	return colors
}

func themeFontsForPart(pkg *pptx.Package, renderPart string, fallback themeFonts) themeFonts {
	themePart := themePartForRenderPart(pkg, renderPart)
	if themePart == "" {
		return fallback
	}
	if fonts := parseThemeFonts(pkg.Parts[themePart]); fonts.MajorLatin != "" || fonts.MinorLatin != "" {
		return fonts
	}
	return fallback
}

func packageThemeEffectStyles(pkg *pptx.Package) themeEffectStyles {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if styles := parseThemeEffectStyles(pkg.Parts[part]); len(styles.Styles) > 0 {
			return styles
		}
	}
	return themeEffectStyles{}
}

func themeEffectStylesForPart(pkg *pptx.Package, renderPart string) themeEffectStyles {
	themePart := themePartForRenderPart(pkg, renderPart)
	if themePart == "" {
		return themeEffectStyles{}
	}
	return parseThemeEffectStyles(pkg.Parts[themePart])
}

func themeFillStylesForPart(pkg *pptx.Package, renderPart string) themeFillStyles {
	themePart := themePartForRenderPart(pkg, renderPart)
	if themePart == "" {
		return packageThemeFillStyles(pkg)
	}
	return parseThemeFillStyles(pkg.Parts[themePart])
}

func themeBackgroundFillForPart(pkg *pptx.Package, renderPart string, idx int64, placeholderColor color.RGBA, theme themeColors) (backgroundPaint, bool) {
	themePart := themePartForRenderPart(pkg, renderPart)
	if themePart == "" {
		return packageThemeBackgroundFill(pkg, idx, placeholderColor, theme)
	}
	return parseThemeBackgroundFill(pkg.Parts[themePart], idx, placeholderColor, theme)
}

func packageThemeFillStyles(pkg *pptx.Package) themeFillStyles {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if styles := parseThemeFillStyles(pkg.Parts[part]); len(styles.Styles) > 0 {
			return styles
		}
	}
	return themeFillStyles{}
}

func packageThemeLineStyles(pkg *pptx.Package) themeLineStyles {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if styles := parseThemeLineStyles(pkg.Parts[part]); len(styles.Styles) > 0 {
			return styles
		}
	}
	return themeLineStyles{}
}

func themeLineStylesForPart(pkg *pptx.Package, renderPart string) themeLineStyles {
	themePart := themePartForRenderPart(pkg, renderPart)
	if themePart == "" {
		return packageThemeLineStyles(pkg)
	}
	return parseThemeLineStyles(pkg.Parts[themePart])
}

func parseThemeFillStyles(data []byte) themeFillStyles {
	root, err := parseXMLNode(data)
	if err != nil {
		return themeFillStyles{}
	}
	var styles themeFillStyles
	if list := firstDescendant(root, "fillStyleLst"); list != nil {
		styles.Styles = list.Children
	}
	if list := firstDescendant(root, "bgFillStyleLst"); list != nil {
		styles.BackgroundStyles = list.Children
	}
	return styles
}

func parseThemeLineStyles(data []byte) themeLineStyles {
	root, err := parseXMLNode(data)
	if err != nil {
		return themeLineStyles{}
	}
	list := firstDescendant(root, "lnStyleLst")
	if list == nil {
		return themeLineStyles{}
	}
	return themeLineStyles{Styles: childrenByName(list, "ln")}
}

func (styles themeFillStyles) Style(idx int64, theme themeColors) (backgroundPaint, bool) {
	if idx <= 0 || idx == 1000 {
		return backgroundPaint{}, false
	}
	if idx >= 1001 {
		styleIndex := int(idx - 1001)
		if styleIndex < 0 || styleIndex >= len(styles.BackgroundStyles) {
			return backgroundPaint{}, false
		}
		return backgroundPaintFromFillNode(styles.BackgroundStyles[styleIndex], theme)
	}
	styleIndex := int(idx - 1)
	if styleIndex < 0 || styleIndex >= len(styles.Styles) {
		return backgroundPaint{}, false
	}
	return backgroundPaintFromFillNode(styles.Styles[styleIndex], theme)
}

func (styles themeLineStyles) Style(idx int64, theme themeColors) (tableCellBorder, bool) {
	if idx <= 0 {
		return tableCellBorder{}, false
	}
	styleIndex := int(idx - 1)
	if styleIndex < 0 || styleIndex >= len(styles.Styles) {
		return tableCellBorder{}, false
	}
	border := parseTableLineNode(styles.Styles[styleIndex], theme, true)
	if border.NoLine || border.HasLine {
		return border, true
	}
	return tableCellBorder{}, false
}

func parseThemeEffectStyles(data []byte) themeEffectStyles {
	root, err := parseXMLNode(data)
	if err != nil {
		return themeEffectStyles{}
	}
	list := firstDescendant(root, "effectStyleLst")
	if list == nil {
		return themeEffectStyles{}
	}
	return themeEffectStyles{Styles: childrenByName(list, "effectStyle")}
}

func (styles themeEffectStyles) Style(idx int64, theme themeColors) (themeEffectStyle, bool) {
	if idx <= 0 {
		return themeEffectStyle{}, false
	}
	styleIndex := int(idx - 1)
	if styleIndex < 0 || styleIndex >= len(styles.Styles) {
		return themeEffectStyle{}, false
	}
	return parseThemeEffectStyle(styles.Styles[styleIndex], theme)
}

func parseThemeEffectStyle(style *xmlNode, theme themeColors) (themeEffectStyle, bool) {
	var element slideElement
	if effectList := firstChild(style, "effectLst"); effectList != nil {
		parseShapeEffects(effectList, &element, theme)
	}
	if sp3d := firstChild(style, "sp3d"); sp3d != nil {
		parseShape3DProperties(sp3d, &element)
	}
	if !element.HasShadow && !element.HasShape3D {
		return themeEffectStyle{}, false
	}
	return themeEffectStyle{
		HasShadow:                true,
		ShadowColor:              element.ShadowColor,
		ShadowBlur:               element.ShadowBlur,
		ShadowDistance:           element.ShadowDistance,
		ShadowDirection:          element.ShadowDirection,
		ShadowAlignment:          element.ShadowAlignment,
		HasShadowRotateWithShape: element.HasShadowRotateWithShape,
		ShadowRotateWithShape:    element.ShadowRotateWithShape,
		HasShadowScaleX:          element.HasShadowScaleX,
		ShadowScaleX:             element.ShadowScaleX,
		HasShadowScaleY:          element.HasShadowScaleY,
		ShadowScaleY:             element.ShadowScaleY,
		HasShadowSkewX:           element.HasShadowSkewX,
		ShadowSkewX:              element.ShadowSkewX,
		HasShadowSkewY:           element.HasShadowSkewY,
		ShadowSkewY:              element.ShadowSkewY,
		HasShape3D:               element.HasShape3D,
		Shape3DFeatures:          append([]string{}, element.Shape3DFeatures...),
	}, true
}

func packageThemeBackgroundFill(pkg *pptx.Package, idx int64, placeholderColor color.RGBA, theme themeColors) (backgroundPaint, bool) {
	paths := make([]string, 0, len(pkg.Parts))
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	for _, part := range paths {
		if paint, ok := parseThemeBackgroundFill(pkg.Parts[part], idx, placeholderColor, theme); ok {
			return paint, true
		}
	}
	return backgroundPaint{}, false
}

func parseThemeBackgroundFill(data []byte, idx int64, placeholderColor color.RGBA, theme themeColors) (backgroundPaint, bool) {
	if idx < 1001 {
		return backgroundPaint{}, false
	}
	return parseThemeFillStyles(data).Style(idx, themeWithPlaceholderColor(theme, placeholderColor))
}

func themeWithPlaceholderColor(theme themeColors, placeholderColor color.RGBA) themeColors {
	merged := themeColors{}
	for key, value := range theme {
		merged[key] = value
	}
	merged["phClr"] = placeholderColor
	return merged
}

func backgroundPaintFromFillNode(node *xmlNode, theme themeColors) (backgroundPaint, bool) {
	switch node.Name {
	case "solidFill":
		if c, ok := colorFromSolidFillWithTheme(node, theme); ok {
			return backgroundPaint{Color: c}, true
		}
	case "gradFill":
		if gradient, ok := parseGradientFill(node, theme); ok {
			return backgroundPaint{Color: gradient.Stops[0].Color, HasGradient: true, Gradient: gradient}, true
		}
	}
	return backgroundPaint{}, false
}

func parseThemeColors(data []byte) themeColors {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	scheme := firstDescendant(root, "clrScheme")
	if scheme == nil {
		return nil
	}
	colors := themeColors{}
	for _, child := range scheme.Children {
		switch child.Name {
		case "dk1", "lt1", "dk2", "lt2", "accent1", "accent2", "accent3", "accent4", "accent5", "accent6", "hlink", "folHlink":
			if c, ok := themeSlotColor(child); ok {
				colors[child.Name] = c
			}
		}
	}
	if c, ok := colors["dk1"]; ok {
		colors["tx1"] = c
	}
	if c, ok := colors["lt1"]; ok {
		colors["bg1"] = c
	}
	if c, ok := colors["dk2"]; ok {
		colors["tx2"] = c
	}
	if c, ok := colors["lt2"]; ok {
		colors["bg2"] = c
	}
	return colors
}

func parseMasterColorMap(data []byte) map[string]string {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	clrMap := firstDescendant(root, "clrMap")
	if clrMap == nil {
		return nil
	}
	return colorMapFromAttrs(clrMap.Attrs)
}

func parseColorMapOverride(data []byte) (map[string]string, bool) {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil, false
	}
	override := firstDescendant(root, "overrideClrMapping")
	if override == nil {
		return nil, false
	}
	return colorMapFromAttrs(override.Attrs), true
}

func colorMapFromAttrs(attrs []xml.Attr) map[string]string {
	mapping := map[string]string{}
	for _, key := range themeColorMapKeys() {
		value := attrValue(attrs, key)
		if value != "" {
			mapping[key] = value
		}
	}
	if len(mapping) == 0 {
		return nil
	}
	return mapping
}

func applyThemeColorMap(colors themeColors, mapping map[string]string) themeColors {
	if len(colors) == 0 || len(mapping) == 0 {
		return nil
	}
	mapped := make(themeColors, len(colors)+len(mapping))
	for key, value := range colors {
		mapped[key] = value
	}
	for _, key := range themeColorMapKeys() {
		source := mapping[key]
		if source == "" {
			continue
		}
		if c, ok := colors[source]; ok {
			mapped[key] = c
		}
	}
	return mapped
}

func themeColorMapKeys() []string {
	return []string{"bg1", "tx1", "bg2", "tx2", "accent1", "accent2", "accent3", "accent4", "accent5", "accent6", "hlink", "folHlink"}
}

func parseThemeFonts(data []byte) themeFonts {
	root, err := parseXMLNode(data)
	if err != nil {
		return themeFonts{}
	}
	scheme := firstDescendant(root, "fontScheme")
	if scheme == nil {
		return themeFonts{}
	}
	var fonts themeFonts
	if major := firstChild(scheme, "majorFont"); major != nil {
		fonts.MajorLatin = latinTypeface(major)
		fonts.MajorEA = typefaceFromChild(major, "ea")
		fonts.MajorCS = typefaceFromChild(major, "cs")
	}
	if minor := firstChild(scheme, "minorFont"); minor != nil {
		fonts.MinorLatin = latinTypeface(minor)
		fonts.MinorEA = typefaceFromChild(minor, "ea")
		fonts.MinorCS = typefaceFromChild(minor, "cs")
	}
	return fonts
}

func latinTypeface(node *xmlNode) string {
	latin := firstChild(node, "latin")
	if latin == nil {
		return ""
	}
	return attrValue(latin.Attrs, "typeface")
}

func themeSlotColor(node *xmlNode) (color.RGBA, bool) {
	if srgb := firstChild(node, "srgbClr"); srgb != nil {
		return parseHexColor(attrValue(srgb.Attrs, "val"))
	}
	if sys := firstChild(node, "sysClr"); sys != nil {
		return parseHexColor(attrValue(sys.Attrs, "lastClr"))
	}
	return color.RGBA{}, false
}

func schemeColor(value string) (color.RGBA, bool) {
	return schemeColorWithTheme(value, defaultThemeColors())
}

func schemeColorWithTheme(value string, theme themeColors) (color.RGBA, bool) {
	if c, ok := theme[value]; ok {
		return c, true
	}
	if c, ok := defaultThemeColors()[value]; ok {
		return c, true
	}
	return color.RGBA{}, false
}

func defaultThemeColors() themeColors {
	return themeColors{
		"tx1":     {A: 255},
		"dk1":     {A: 255},
		"bg1":     {R: 255, G: 255, B: 255, A: 255},
		"lt1":     {R: 255, G: 255, B: 255, A: 255},
		"tx2":     {R: 31, G: 31, B: 31, A: 255},
		"dk2":     {R: 31, G: 31, B: 31, A: 255},
		"bg2":     {R: 238, G: 238, B: 238, A: 255},
		"lt2":     {R: 238, G: 238, B: 238, A: 255},
		"accent1": {R: 79, G: 129, B: 189, A: 255},
		"accent2": {R: 192, G: 80, B: 77, A: 255},
		"accent3": {R: 155, G: 187, B: 89, A: 255},
		"accent4": {R: 128, G: 100, B: 162, A: 255},
		"accent5": {R: 75, G: 172, B: 198, A: 255},
		"accent6": {R: 247, G: 150, B: 70, A: 255},
	}
}
