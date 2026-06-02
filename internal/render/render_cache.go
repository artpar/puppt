package render

import (
	"fmt"
	"image"
	"image/color"
	"sort"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

type renderCache struct {
	pkg *pptx.Package

	xmlNodes               map[string]*xmlNode
	relationships          map[string][]pptx.Relationship
	relationshipByID       map[string]map[string]pptx.Relationship
	themePartByRenderPart  map[string]string
	colorMaps              map[string]map[string]string
	colorMapsKnown         map[string]bool
	themeColors            map[string]themeColors
	themeFonts             map[string]themeFonts
	themeEffectStyles      map[string]themeEffectStyles
	themeFillStyles        map[string]themeFillStyles
	themeLineStyles        map[string]themeLineStyles
	tableStyles            map[string]tableStyleSet
	images                 map[string]cachedRenderImage
	headerFooterSettings   map[string]headerFooterSettings
	textStyles             map[string]map[string]textStyle
	elements               map[string][]slideElement
	backgrounds            map[string]backgroundPaint
	packageThemeParts      []string
	packageThemePartsReady bool
	packageColors          themeColors
	packageFonts           themeFonts
	packageThemeEffects    themeEffectStyles
	packageThemeFills      themeFillStyles
	packageThemeLines      themeLineStyles
}

func newRenderCache(pkg *pptx.Package) *renderCache {
	return &renderCache{
		pkg:                   pkg,
		xmlNodes:              map[string]*xmlNode{},
		relationships:         map[string][]pptx.Relationship{},
		relationshipByID:      map[string]map[string]pptx.Relationship{},
		themePartByRenderPart: map[string]string{},
		colorMaps:             map[string]map[string]string{},
		colorMapsKnown:        map[string]bool{},
		themeColors:           map[string]themeColors{},
		themeFonts:            map[string]themeFonts{},
		themeEffectStyles:     map[string]themeEffectStyles{},
		themeFillStyles:       map[string]themeFillStyles{},
		themeLineStyles:       map[string]themeLineStyles{},
		tableStyles:           map[string]tableStyleSet{},
		images:                map[string]cachedRenderImage{},
		headerFooterSettings:  map[string]headerFooterSettings{},
		textStyles:            map[string]map[string]textStyle{},
		elements:              map[string][]slideElement{},
		backgrounds:           map[string]backgroundPaint{},
	}
}

type cachedRenderImage struct {
	image image.Image
	err   error
}

func (cache *renderCache) partData(part string) ([]byte, bool) {
	if cache == nil || cache.pkg == nil {
		return nil, false
	}
	data, ok := cache.pkg.Parts[part]
	return data, ok
}

func (cache *renderCache) xmlNodeForPart(part string) (*xmlNode, bool) {
	if cache == nil {
		return nil, false
	}
	if root, ok := cache.xmlNodes[part]; ok {
		return root, root != nil
	}
	data, ok := cache.partData(part)
	if !ok {
		cache.xmlNodes[part] = nil
		return nil, false
	}
	root, err := parseXMLNode(data)
	if err != nil {
		cache.xmlNodes[part] = nil
		return nil, false
	}
	cache.xmlNodes[part] = root
	return root, true
}

func (cache *renderCache) relationshipsForPart(part string) ([]pptx.Relationship, error) {
	if cache == nil || cache.pkg == nil {
		return nil, fmt.Errorf("missing package")
	}
	if relationships, ok := cache.relationships[part]; ok {
		return relationships, nil
	}
	relationships, err := cache.pkg.RelationshipsForPart(part)
	if err != nil {
		return nil, err
	}
	cache.relationships[part] = relationships
	return relationships, nil
}

func (cache *renderCache) relationshipsByIDForPart(part string) (map[string]pptx.Relationship, error) {
	if relationships, ok := cache.relationshipByID[part]; ok {
		return relationships, nil
	}
	relationships, err := cache.relationshipsForPart(part)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]pptx.Relationship, len(relationships))
	for _, relationship := range relationships {
		byID[relationship.ID] = relationship
	}
	cache.relationshipByID[part] = byID
	return byID, nil
}

func (cache *renderCache) firstRelationshipTarget(sourcePart string, relationshipType string) string {
	relationships, err := cache.relationshipsForPart(sourcePart)
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

func (cache *renderCache) inheritedRenderParts(slidePart string) []string {
	var parts []string
	layoutPart := cache.firstRelationshipTarget(slidePart, pptx.SlideLayoutRelType)
	masterPart := ""
	if layoutPart != "" {
		masterPart = cache.firstRelationshipTarget(layoutPart, pptx.SlideMasterRelType)
	}
	for _, part := range []string{masterPart, layoutPart, slidePart} {
		if part == "" {
			continue
		}
		if _, ok := cache.partData(part); ok {
			parts = append(parts, part)
		}
	}
	return parts
}

func (cache *renderCache) visibleRenderParts(slidePart string, parts []string) []string {
	if !cache.layoutHidesMasterShapes(slidePart) {
		return parts
	}
	layoutPart := cache.firstRelationshipTarget(slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return parts
	}
	masterPart := cache.firstRelationshipTarget(layoutPart, pptx.SlideMasterRelType)
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

func (cache *renderCache) layoutHidesMasterShapes(slidePart string) bool {
	layoutPart := cache.firstRelationshipTarget(slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return false
	}
	root, ok := cache.xmlNodeForPart(layoutPart)
	if !ok {
		return false
	}
	value := strings.ToLower(strings.TrimSpace(attrValue(root.Attrs, "showMasterSp")))
	return value == "0" || value == "false"
}

func (cache *renderCache) presentationShowsSpecialPlaceholdersOnTitleSlide() bool {
	root, ok := cache.xmlNodeForPart(cache.pkg.PresentationPath)
	if !ok {
		return true
	}
	value := strings.TrimSpace(attrValue(root.Attrs, "showSpecialPlsOnTitleSld"))
	if value == "" {
		return true
	}
	return boolAttrOn(value)
}

func (cache *renderCache) slideUsesTitleLayout(slidePart string) bool {
	layoutPart := cache.firstRelationshipTarget(slidePart, pptx.SlideLayoutRelType)
	if layoutPart == "" {
		return false
	}
	root, ok := cache.xmlNodeForPart(layoutPart)
	if !ok {
		return false
	}
	return strings.TrimSpace(attrValue(root.Attrs, "type")) == "title"
}

func (cache *renderCache) packageThemePartNames() []string {
	if cache.packageThemePartsReady {
		return cache.packageThemeParts
	}
	paths := make([]string, 0, len(cache.pkg.Parts))
	for part := range cache.pkg.Parts {
		if strings.HasPrefix(part, "ppt/theme/") && strings.HasSuffix(part, ".xml") {
			paths = append(paths, part)
		}
	}
	sort.Strings(paths)
	cache.packageThemeParts = paths
	cache.packageThemePartsReady = true
	return paths
}

func (cache *renderCache) packageThemeColors() themeColors {
	if cache.packageColors != nil {
		return cache.packageColors
	}
	for _, part := range cache.packageThemePartNames() {
		if colors := cache.parseThemeColors(part); len(colors) > 0 {
			cache.packageColors = colors
			return colors
		}
	}
	cache.packageColors = defaultThemeColors()
	return cache.packageColors
}

func (cache *renderCache) packageThemeFonts() themeFonts {
	if cache.packageFonts.MajorLatin != "" || cache.packageFonts.MinorLatin != "" {
		return cache.packageFonts
	}
	for _, part := range cache.packageThemePartNames() {
		if fonts := cache.parseThemeFonts(part); fonts.MajorLatin != "" || fonts.MinorLatin != "" {
			cache.packageFonts = fonts
			return fonts
		}
	}
	return themeFonts{}
}

func (cache *renderCache) packageThemeEffectStyles() themeEffectStyles {
	if len(cache.packageThemeEffects.Styles) > 0 {
		return cache.packageThemeEffects
	}
	for _, part := range cache.packageThemePartNames() {
		if styles := cache.parseThemeEffectStyles(part); len(styles.Styles) > 0 {
			cache.packageThemeEffects = styles
			return styles
		}
	}
	return themeEffectStyles{}
}

func (cache *renderCache) packageThemeFillStyles() themeFillStyles {
	if len(cache.packageThemeFills.Styles) > 0 || len(cache.packageThemeFills.BackgroundStyles) > 0 {
		return cache.packageThemeFills
	}
	for _, part := range cache.packageThemePartNames() {
		if styles := cache.parseThemeFillStyles(part); len(styles.Styles) > 0 {
			cache.packageThemeFills = styles
			return styles
		}
	}
	return themeFillStyles{}
}

func (cache *renderCache) packageThemeLineStyles() themeLineStyles {
	if len(cache.packageThemeLines.Styles) > 0 {
		return cache.packageThemeLines
	}
	for _, part := range cache.packageThemePartNames() {
		if styles := cache.parseThemeLineStyles(part); len(styles.Styles) > 0 {
			cache.packageThemeLines = styles
			return styles
		}
	}
	return themeLineStyles{}
}

func (cache *renderCache) themePartForRenderPart(renderPart string) string {
	if themePart, ok := cache.themePartByRenderPart[renderPart]; ok {
		return themePart
	}
	original := renderPart
	if strings.HasPrefix(renderPart, "ppt/slides/") {
		layoutPart := cache.firstRelationshipTarget(renderPart, pptx.SlideLayoutRelType)
		if layoutPart == "" {
			cache.themePartByRenderPart[original] = ""
			return ""
		}
		renderPart = layoutPart
	}
	if strings.HasPrefix(renderPart, "ppt/slideLayouts/") {
		masterPart := cache.firstRelationshipTarget(renderPart, pptx.SlideMasterRelType)
		if masterPart == "" {
			cache.themePartByRenderPart[original] = ""
			return ""
		}
		renderPart = masterPart
	}
	themePart := cache.firstRelationshipTarget(renderPart, themeRelType)
	cache.themePartByRenderPart[original] = themePart
	return themePart
}

func (cache *renderCache) colorMapForRenderPart(renderPart string) map[string]string {
	if cache.colorMapsKnown[renderPart] {
		return cache.colorMaps[renderPart]
	}
	mapping := cache.computeColorMapForRenderPart(renderPart)
	cache.colorMaps[renderPart] = mapping
	cache.colorMapsKnown[renderPart] = true
	return mapping
}

func (cache *renderCache) computeColorMapForRenderPart(renderPart string) map[string]string {
	if mapping, ok := cache.parseColorMapOverride(renderPart); ok {
		return mapping
	}
	if strings.HasPrefix(renderPart, "ppt/slideMasters/") {
		return cache.parseMasterColorMap(renderPart)
	}
	if strings.HasPrefix(renderPart, "ppt/slides/") {
		layoutPart := cache.firstRelationshipTarget(renderPart, pptx.SlideLayoutRelType)
		if layoutPart == "" {
			return nil
		}
		renderPart = layoutPart
	}
	if strings.HasPrefix(renderPart, "ppt/slideLayouts/") {
		if mapping, ok := cache.parseColorMapOverride(renderPart); ok {
			return mapping
		}
		masterPart := cache.firstRelationshipTarget(renderPart, pptx.SlideMasterRelType)
		if masterPart == "" {
			return nil
		}
		return cache.parseMasterColorMap(masterPart)
	}
	return nil
}

func (cache *renderCache) themeColorsForPart(renderPart string, fallback themeColors) themeColors {
	if colors, ok := cache.themeColors[renderPart]; ok {
		return colors
	}
	themePart := cache.themePartForRenderPart(renderPart)
	var colors themeColors
	if themePart == "" {
		colors = fallback
	} else if parsed := cache.parseThemeColors(themePart); len(parsed) > 0 {
		colors = parsed
	} else {
		colors = fallback
	}
	if mapped := applyThemeColorMap(colors, cache.colorMapForRenderPart(renderPart)); len(mapped) > 0 {
		cache.themeColors[renderPart] = mapped
		return mapped
	}
	cache.themeColors[renderPart] = colors
	return colors
}

func (cache *renderCache) themeFontsForPart(renderPart string, fallback themeFonts) themeFonts {
	if fonts, ok := cache.themeFonts[renderPart]; ok {
		return fonts
	}
	themePart := cache.themePartForRenderPart(renderPart)
	if themePart == "" {
		cache.themeFonts[renderPart] = fallback
		return fallback
	}
	fonts := cache.parseThemeFonts(themePart)
	if fonts.MajorLatin == "" && fonts.MinorLatin == "" {
		fonts = fallback
	}
	cache.themeFonts[renderPart] = fonts
	return fonts
}

func (cache *renderCache) themeEffectStylesForPart(renderPart string) themeEffectStyles {
	if styles, ok := cache.themeEffectStyles[renderPart]; ok {
		return styles
	}
	themePart := cache.themePartForRenderPart(renderPart)
	var styles themeEffectStyles
	if themePart == "" {
		styles = themeEffectStyles{}
	} else {
		styles = cache.parseThemeEffectStyles(themePart)
	}
	cache.themeEffectStyles[renderPart] = styles
	return styles
}

func (cache *renderCache) themeFillStylesForPart(renderPart string) themeFillStyles {
	if styles, ok := cache.themeFillStyles[renderPart]; ok {
		return styles
	}
	themePart := cache.themePartForRenderPart(renderPart)
	var styles themeFillStyles
	if themePart == "" {
		styles = cache.packageThemeFillStyles()
	} else {
		styles = cache.parseThemeFillStyles(themePart)
	}
	cache.themeFillStyles[renderPart] = styles
	return styles
}

func (cache *renderCache) themeLineStylesForPart(renderPart string) themeLineStyles {
	if styles, ok := cache.themeLineStyles[renderPart]; ok {
		return styles
	}
	themePart := cache.themePartForRenderPart(renderPart)
	var styles themeLineStyles
	if themePart == "" {
		styles = cache.packageThemeLineStyles()
	} else {
		styles = cache.parseThemeLineStyles(themePart)
	}
	cache.themeLineStyles[renderPart] = styles
	return styles
}

func (cache *renderCache) themeBackgroundFillForPart(renderPart string, idx int64, placeholderColor color.RGBA, theme themeColors) (backgroundPaint, bool) {
	themePart := cache.themePartForRenderPart(renderPart)
	if themePart == "" {
		for _, part := range cache.packageThemePartNames() {
			if paint, ok := cache.parseThemeFillStyles(part).Style(idx, themeWithPlaceholderColor(theme, placeholderColor)); ok {
				return paint, true
			}
		}
		return backgroundPaint{}, false
	}
	if idx < 1001 {
		return backgroundPaint{}, false
	}
	return cache.parseThemeFillStyles(themePart).Style(idx, themeWithPlaceholderColor(theme, placeholderColor))
}

func (cache *renderCache) parseThemeColors(part string) themeColors {
	if colors, ok := cache.themeColors[part]; ok {
		return colors
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return nil
	}
	colors := parseThemeColorsFromRoot(root)
	cache.themeColors[part] = colors
	return colors
}

func (cache *renderCache) parseThemeFonts(part string) themeFonts {
	if fonts, ok := cache.themeFonts[part]; ok {
		return fonts
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return themeFonts{}
	}
	fonts := parseThemeFontsFromRoot(root)
	cache.themeFonts[part] = fonts
	return fonts
}

func (cache *renderCache) parseThemeEffectStyles(part string) themeEffectStyles {
	if styles, ok := cache.themeEffectStyles[part]; ok {
		return styles
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return themeEffectStyles{}
	}
	styles := parseThemeEffectStylesFromRoot(root)
	cache.themeEffectStyles[part] = styles
	return styles
}

func (cache *renderCache) parseThemeFillStyles(part string) themeFillStyles {
	if styles, ok := cache.themeFillStyles[part]; ok {
		return styles
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return themeFillStyles{}
	}
	styles := parseThemeFillStylesFromRoot(root)
	cache.themeFillStyles[part] = styles
	return styles
}

func (cache *renderCache) parseThemeLineStyles(part string) themeLineStyles {
	if styles, ok := cache.themeLineStyles[part]; ok {
		return styles
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return themeLineStyles{}
	}
	styles := parseThemeLineStylesFromRoot(root)
	cache.themeLineStyles[part] = styles
	return styles
}

func (cache *renderCache) parseColorMapOverride(part string) (map[string]string, bool) {
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return nil, false
	}
	return parseColorMapOverrideFromRoot(root)
}

func (cache *renderCache) parseMasterColorMap(part string) map[string]string {
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return nil
	}
	return parseMasterColorMapFromRoot(root)
}

func (cache *renderCache) slideBackgroundPaint(renderPart string, theme themeColors) (backgroundPaint, bool) {
	if paint, ok := cache.backgrounds[renderPart]; ok {
		return paint, true
	}
	root, ok := cache.xmlNodeForPart(renderPart)
	if !ok {
		return backgroundPaint{}, false
	}
	resolveStyle := func(idx int64, placeholderColor color.RGBA) (backgroundPaint, bool) {
		return cache.themeBackgroundFillForPart(renderPart, idx, placeholderColor, theme)
	}
	paint, ok := parseSlideBackgroundPaintFromRootWithThemeAndResolver(root, theme, resolveStyle)
	if ok {
		cache.backgrounds[renderPart] = paint
	}
	return paint, ok
}

func (cache *renderCache) inheritedBackground(renderParts []string, themeForPart func(string) themeColors) backgroundPaint {
	background := backgroundPaint{Color: color.RGBA{R: 255, G: 255, B: 255, A: 255}}
	for _, renderPart := range renderParts {
		if paint, ok := cache.slideBackgroundPaint(renderPart, themeForPart(renderPart)); ok {
			paint.Part = renderPart
			background = paint
		}
	}
	return background
}

func (cache *renderCache) headerFooterSettingsForPart(part string) headerFooterSettings {
	if settings, ok := cache.headerFooterSettings[part]; ok {
		return settings
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return headerFooterSettings{}
	}
	settings := parseHeaderFooterSettingsFromRoot(root)
	cache.headerFooterSettings[part] = settings
	return settings
}

func (cache *renderCache) inheritedHeaderFooterSettings(renderParts []string) headerFooterSettings {
	settings := defaultHeaderFooterSettings()
	for _, part := range renderParts {
		partSettings := cache.headerFooterSettingsForPart(part)
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

func (cache *renderCache) textStylesForPart(part string, theme themeColors) map[string]textStyle {
	if styles, ok := cache.textStyles[part]; ok {
		return styles
	}
	root, ok := cache.xmlNodeForPart(part)
	if !ok {
		return nil
	}
	styles := parseTextStylesFromRoot(root, theme)
	cache.textStyles[part] = styles
	return styles
}

func (cache *renderCache) inheritedTextStyles(renderParts []string, slidePart string, themeForPart func(string) themeColors) map[string]textStyle {
	styles := map[string]textStyle{}
	if root, ok := cache.xmlNodeForPart(cache.pkg.PresentationPath); ok {
		if style, ok := parsePresentationDefaultTextStyleFromRoot(root, themeForPart(cache.pkg.PresentationPath)); ok {
			styles["default"] = style
		}
	}
	for _, renderPart := range renderParts {
		if renderPart == slidePart {
			continue
		}
		for key, style := range cache.textStylesForPart(renderPart, themeForPart(renderPart)) {
			styles[key] = mergeTextStyle(styles[key], style)
		}
	}
	return styles
}

func (cache *renderCache) elementsForPart(renderPart string, theme themeColors, effectStyles themeEffectStyles, fillStyles themeFillStyles, lineStyles themeLineStyles) []slideElement {
	root, ok := cache.xmlNodeForPart(renderPart)
	if !ok {
		return nil
	}
	return collectSlideElementsFromRootWithThemeEffectsAndFills(root, theme, effectStyles, fillStyles, lineStyles)
}

func (cache *renderCache) inheritedPlaceholderSources(renderParts []string, slidePart string, themeForPart func(string) themeColors) map[string]slideElement {
	sources := make(map[string]slideElement)
	for _, renderPart := range renderParts {
		if renderPart == slidePart {
			continue
		}
		for _, element := range cache.elementsForPart(renderPart, themeForPart(renderPart), themeEffectStyles{}, themeFillStyles{}, themeLineStyles{}) {
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

func (cache *renderCache) inheritedHeaderFooterRenderPart(paintParts []string, slidePart string, settings headerFooterSettings) string {
	for index := len(paintParts) - 1; index >= 0; index-- {
		part := paintParts[index]
		if part == slidePart {
			continue
		}
		for _, element := range cache.elementsForPart(part, defaultThemeColors(), themeEffectStyles{}, themeFillStyles{}, themeLineStyles{}) {
			if element.IsPlaceholder && headerFooterPlaceholderEnabled(element.PlaceholderType, settings) {
				return part
			}
		}
	}
	return ""
}

func (cache *renderCache) packageTableStyles(renderPart string, theme themeColors, fonts themeFonts, fillStyles themeFillStyles, lineStyles themeLineStyles, effectStyles themeEffectStyles) tableStyleSet {
	if styles, ok := cache.tableStyles[renderPart]; ok {
		return styles
	}
	root, ok := cache.xmlNodeForPart("ppt/tableStyles.xml")
	if !ok {
		return tableStyleSet{}
	}
	styles := parseTableStylesFromRoot(root, theme, fonts, fillStyles, lineStyles, effectStyles)
	cache.tableStyles[renderPart] = styles
	return styles
}

func (cache *renderCache) renderElementsWithDebug(slidePart string, sourcePart string, size slideSize, img *image.RGBA, elements []slideElement, tableStyles tableStyleSet, debug *ObjectDebugOptions) []model.SkipItem {
	relationshipByID, err := cache.relationshipsByIDForPart(sourcePart)
	if err != nil {
		return []model.SkipItem{{
			Code:    unsupportedCode,
			Message: fmt.Sprintf("slide relationships could not be parsed for rendering: %v", err),
			Part:    pptx.RelationshipsPartFor(sourcePart),
		}}
	}
	if debug != nil {
		_, _ = renderSceneFromElements(cache.pkg, slidePart, sourcePart, size, img.Bounds(), elements, relationshipByID)
	}

	var unsupported []model.SkipItem
	for index := range elements {
		zOrder := debug.nextObjectZOrder()
		shouldPaint := debug.shouldPaintObject(zOrder)
		var before *image.RGBA
		if debug != nil {
			before = cloneRGBA(img)
		}
		var items []model.SkipItem
		if shouldPaint {
			items = renderOneElementWithCache(cache, sourcePart, size, img, &elements[index], relationshipByID, tableStyles)
		}
		record := paintedObjectRecord(slidePart, sourcePart, elements[index], zOrder, size, img.Bounds(), before, img, shouldPaint && elements[index].Rendered, items)
		if debug != nil && debug.ArtifactDir != "" && shouldPaint {
			objectImage := image.NewRGBA(img.Bounds())
			objectElement := elements[index]
			_ = renderOneElementWithCache(cache, sourcePart, size, objectImage, &objectElement, relationshipByID, tableStyles)
			writeObjectDebugArtifacts(debug, renderDPIForCanvas(size, img.Bounds()), &record, before, objectImage, img)
		}
		appendPaintedObjectRecord(debug, record)
		unsupported = append(unsupported, items...)
	}
	return unsupported
}

func renderOneElementWithCache(cache *renderCache, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship, tableStyles tableStyleSet) []model.SkipItem {
	switch element.Kind {
	case "pic":
		return renderPictureWithCache(cache, slidePart, size, img, element, relationships)
	case "sp", "cxnSp":
		var items []model.SkipItem
		if element.EmbedID != "" {
			items = append(items, renderPictureWithCache(cache, slidePart, size, img, element, relationships)...)
		}
		items = append(items, renderShape(slidePart, size, img, element)...)
		return items
	case "graphicFrame":
		return renderGraphicFrameWithCache(cache, slidePart, size, img, element, relationships, tableStyles)
	default:
		return renderUnsupportedPayloadElement(cache.pkg, slidePart, element, relationships)
	}
}

func renderPictureWithCache(cache *renderCache, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	primitive, err := renderPicturePrimitiveFromElement(cache.pkg, slidePart, size, img.Bounds(), *element, relationships)
	if err != nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, err.Error())}
	}
	relationshipID := primitive.RelationshipID
	if relationshipID == "" {
		relationshipID = primitive.LinkRelationshipID
	}
	relationship := relationships[relationshipID]

	source, targetPart, partialUnsupported := pictureSourceImageWithCache(cache, slidePart, element, relationships, relationship)
	if source == nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported image data %q: %v", elementLabel(*element), targetPart, partialUnsupported))}
	}
	element.ImageMediaPart = targetPart
	element.ImageContentType = primitive.ContentType
	sourceBounds := source.Bounds()
	element.ImageWidth = sourceBounds.Dx()
	element.ImageHeight = sourceBounds.Dy()
	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart:          slidePart,
		Size:               size,
		Canvas:             img,
		Primitive:          primitive,
		Source:             source,
		TargetPart:         targetPart,
		PartialUnsupported: partialUnsupported,
	})
	element.Rendered = true
	return unsupported
}

func pictureSourceImageWithCache(cache *renderCache, slidePart string, element *slideElement, relationships map[string]pptx.Relationship, fallbackRelationship pptx.Relationship) (image.Image, string, error) {
	fallback, fallbackPart, fallbackErr := fallbackPictureSourceImageWithCache(cache, slidePart, fallbackRelationship)
	if fallbackErr == nil {
		return fallback, fallbackPart, nil
	}
	if element.SVGEmbedID == "" {
		return nil, fallbackPart, fallbackErr
	}
	relationship, ok := relationships[element.SVGEmbedID]
	if !ok || relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return nil, fallbackPart, fallbackErr
	}
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	source, err := cache.decodeImagePart(targetPart)
	if err != nil {
		return nil, targetPart, fallbackErr
	}
	return source, targetPart, fallbackErr
}

func fallbackPictureSourceImageWithCache(cache *renderCache, slidePart string, relationship pptx.Relationship) (image.Image, string, error) {
	if relationship.Target == "" {
		return nil, "", fmt.Errorf("missing image relationship")
	}
	if relationship.Type != "" && relationship.Type != pptx.ImageRelType {
		return nil, "", fmt.Errorf("relationship type %q is not an image", relationship.Type)
	}
	if relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal") {
		return nil, relationship.Target, fmt.Errorf("linked image relationship target %q is external and was not fetched", relationship.Target)
	}
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	source, err := cache.decodeImagePart(targetPart)
	if err != nil {
		return nil, targetPart, err
	}
	return source, targetPart, nil
}

func (cache *renderCache) decodeImagePart(targetPart string) (image.Image, error) {
	if cached, ok := cache.images[targetPart]; ok {
		return cached.image, cached.err
	}
	data, ok := cache.pkg.Parts[targetPart]
	if !ok {
		err := fmt.Errorf("missing image part")
		cache.images[targetPart] = cachedRenderImage{err: err}
		return nil, err
	}
	source, err := decodeImage(targetPart, cache.pkg.ContentTypes.ForPart(targetPart), data)
	cache.images[targetPart] = cachedRenderImage{image: source, err: err}
	return source, err
}

func renderGraphicFrameWithCache(cache *renderCache, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship, tableStyles tableStyleSet) []model.SkipItem {
	if element.DiagramDataID != "" {
		return renderDiagramGraphicFrameWithCache(cache, slidePart, size, img, element, relationships)
	}
	return renderGraphicFrame(cache.pkg, slidePart, size, img, element, relationships, tableStyles)
}

func renderDiagramGraphicFrameWithCache(cache *renderCache, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return nil
	}
	drawingPart, ok, err := diagramDrawingPartWithCache(cache, slidePart, element.DiagramDataID, relationships)
	if err != nil {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q diagram could not be resolved: %v", elementLabel(*element), err))}
	}
	if !ok {
		message := fmt.Sprintf("graphic frame object %q diagram payload was preserved but SmartArt layout fallback drawing was not available", elementLabel(*element))
		element.UnsupportedNote = message
		return []model.SkipItem{unsupportedItem(slidePart, partialUnsupportedCode, message)}
	}
	diagramElements := diagramDrawingElementsWithCache(cache, slidePart, drawingPart)
	diagramElements = fitDiagramElementsToFrame(diagramElements, *element)
	var unsupported []model.SkipItem
	renderedSupportedElement := false
	for index := range diagramElements {
		if diagramElements[index].Kind != "sp" && diagramElements[index].Kind != "cxnSp" {
			if diagramElements[index].Kind != "" {
				unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q diagram contains %s content that was not rendered", elementLabel(*element), objectKindLabel(diagramElements[index].Kind))))
			}
			continue
		}
		unsupported = append(unsupported, renderShape(slidePart, size, img, &diagramElements[index])...)
		if diagramElements[index].Rendered {
			renderedSupportedElement = true
		}
	}
	element.Rendered = renderedSupportedElement
	return unsupported
}

func diagramDrawingElementsWithCache(cache *renderCache, slidePart, drawingPart string) []slideElement {
	colors := cache.themeColorsForPart(slidePart, cache.packageThemeColors())
	fonts := cache.themeFontsForPart(slidePart, cache.packageThemeFonts())
	effectStyles := cache.themeEffectStylesForPart(slidePart)
	fillStyles := cache.themeFillStylesForPart(slidePart)
	lineStyles := cache.themeLineStylesForPart(slidePart)
	elements := cache.elementsForPart(drawingPart, colors, effectStyles, fillStyles, lineStyles)
	return applyThemeFontFamilies(elements, fonts)
}

func diagramDrawingPartWithCache(cache *renderCache, slidePart string, diagramDataID string, relationships map[string]pptx.Relationship) (string, bool, error) {
	dataRel, ok := relationships[diagramDataID]
	if !ok || dataRel.Type != diagramDataRelType || (dataRel.TargetMode != "" && !strings.EqualFold(dataRel.TargetMode, "Internal")) {
		return "", false, nil
	}
	dataPart := pptx.ResolveTargetPart(slidePart, dataRel.Target)
	if _, ok := cache.partData(dataPart); !ok {
		return "", false, fmt.Errorf("diagram data part %s is missing", dataPart)
	}
	root, ok := cache.xmlNodeForPart(dataPart)
	if !ok {
		return "", false, fmt.Errorf("parse diagram data %s", dataPart)
	}
	ext := firstDescendant(root, "dataModelExt")
	if ext == nil {
		return "", false, nil
	}
	drawingID := attrValue(ext.Attrs, "relId")
	if drawingID == "" {
		return "", false, nil
	}
	drawingRel, ok := relationships[drawingID]
	if !ok || drawingRel.Type != diagramDrawingRelType || (drawingRel.TargetMode != "" && !strings.EqualFold(drawingRel.TargetMode, "Internal")) {
		return "", false, nil
	}
	drawingPart := pptx.ResolveTargetPart(slidePart, drawingRel.Target)
	if _, ok := cache.partData(drawingPart); !ok {
		return "", false, fmt.Errorf("diagram drawing part %s is missing", drawingPart)
	}
	return drawingPart, true, nil
}
