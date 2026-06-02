package render

import (
	"embed"
	"errors"
	"fmt"
	"image"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

//go:embed assets/fonts/carlito/*.ttf
var bundledFontFS embed.FS
var resolvedFontSourceCache sync.Map
var parsedOpenTypeFontCache sync.Map

func fontResolutionUnsupportedMessage(element slideElement) string {
	messages := fontResolutionUnsupportedMessages(element)
	if len(messages) == 0 {
		return ""
	}
	return messages[0]
}

func fontResolutionUnsupportedMessages(element slideElement) []string {
	seen := map[string]bool{}
	seenMessages := map[string]bool{}
	var messages []string
	appendMessage := func(key string, message string) {
		if message == "" || seenMessages[message] {
			return
		}
		if key != "" {
			seen[key] = true
		}
		seenMessages[message] = true
		messages = append(messages, message)
	}
	if message := fontResolutionUnsupportedMessageForFamily(element.FontFamily, false, element.Italic); message != "" {
		appendMessage(fontResolutionMessageKey(element.FontFamily, false, element.Italic), message)
	}
	for _, paragraph := range element.TextParagraphs {
		key := fontResolutionMessageKey(paragraph.FontFamily, paragraph.Bold, paragraph.Italic)
		if key != "" && !seen[key] {
			if message := fontResolutionUnsupportedMessageForFamily(paragraph.FontFamily, paragraph.Bold, paragraph.Italic); message != "" {
				appendMessage(key, message)
			}
		}
		for _, run := range paragraph.Runs {
			bold := resolvedRunBold(run, paragraph)
			italic := resolvedRunItalic(run, paragraph)
			key := fontResolutionMessageKey(run.FontFamily, bold, italic)
			if key == "" || seen[key] {
				continue
			}
			if message := fontResolutionUnsupportedMessageForFamily(run.FontFamily, bold, italic); message != "" {
				appendMessage(key, message)
			}
		}
	}
	return messages
}

func elementShouldReportFontResolution(element slideElement) bool {
	return elementShouldRenderText(element)
}

func elementShouldRenderText(element slideElement) bool {
	if strings.TrimSpace(element.Text) == "" {
		return false
	}
	return !elementHasOnlyTinyImagePlaceholderMarkerText(element)
}

func elementHasOnlyTinyImagePlaceholderMarkerText(element slideElement) bool {
	if element.EmbedID == "" {
		return false
	}
	if !element.IsPlaceholder && !strings.Contains(strings.ToLower(element.Name), "placeholder") {
		return false
	}
	if strings.TrimSpace(element.Text) != "." {
		return false
	}
	size := maxElementTextFontSize(element)
	return size > 0 && size <= 100
}

func maxElementTextFontSize(element slideElement) int {
	maxSize := element.FontSize
	for _, paragraph := range element.TextParagraphs {
		if strings.TrimSpace(paragraph.Text) != "" && paragraph.FontSize > maxSize {
			maxSize = paragraph.FontSize
		}
		for _, run := range paragraph.Runs {
			if strings.TrimSpace(run.Text) != "" && run.FontSize > maxSize {
				maxSize = run.FontSize
			}
		}
	}
	return maxSize
}

func textLayoutUnsupportedMessages(element slideElement) []string {
	return textLayoutUnsupportedMessagesForTarget(element, image.Rectangle{}, defaultOutputDPI)
}

func textLayoutUnsupportedMessagesForTarget(element slideElement, bounds image.Rectangle, dpi int) []string {
	messages := staticTextUnsupportedMessages(element)
	if normalAutofitRequiresSimplifiedSizing(element, bounds, dpi) {
		messages = append(messages, "normal autofit was rendered with simplified sizing")
	}
	if !shapeAutofitLayoutSupported(element) {
		messages = append(messages, "shape autofit was rendered with simplified sizing")
	}
	return messages
}

func staticTextUnsupportedMessages(element slideElement) []string {
	var messages []string
	if element.TextWrap != "" && element.TextWrap != "square" && element.TextWrap != "none" {
		messages = append(messages, fmt.Sprintf("text body wrap mode %q was not rendered", element.TextWrap))
	}
	if element.TextHorizontalOverflow != "" && element.TextHorizontalOverflow != "overflow" && element.TextHorizontalOverflow != "clip" {
		messages = append(messages, fmt.Sprintf("text horizontal overflow mode %q was not rendered", element.TextHorizontalOverflow))
	}
	if element.TextVerticalOverflow != "" && element.TextVerticalOverflow != "overflow" && element.TextVerticalOverflow != "clip" {
		if element.TextVerticalOverflow == "ellipsis" {
			messages = append(messages, "text vertical overflow ellipsis was rendered as clipped")
		} else {
			messages = append(messages, fmt.Sprintf("text vertical overflow mode %q was not rendered", element.TextVerticalOverflow))
		}
	}
	if element.TextVertical != "" && element.TextVertical != "horz" {
		messages = append(messages, fmt.Sprintf("text body vertical mode %q was not rendered", element.TextVertical))
	}
	if element.HasTextBodyRotation && element.TextBodyRotation != 0 {
		messages = append(messages, "text body rotation was not rendered")
	}
	if element.TextColumnCount > 1 {
		messages = append(messages, "text body columns were not rendered")
	}
	if element.HasTextRightToLeftColumns && element.TextRightToLeftColumns && element.TextColumnCount > 1 {
		messages = append(messages, "text body right-to-left column order was rendered left-to-right (dml-main.xsd:2637 CT_TextBodyProperties@rtlCol)")
	}
	if len(element.Text3DFeatures) > 0 {
		features := append([]string{}, element.Text3DFeatures...)
		sort.Strings(features)
		messages = append(messages, fmt.Sprintf("%s were not rendered", strings.Join(features, ", ")))
	}
	if elementContainsAuthoredRTLParagraph(element) {
		messages = append(messages, "paragraph rtl=1 was rendered with left-to-right fallback (dml-main.xsd:3013 CT_TextParagraphProperties@rtl)")
	}
	if elementContainsRTLText(element) {
		messages = append(messages, "bidirectional/RTL text was rendered with left-to-right fallback (dml-main.xsd CT_TextParagraph/CT_RegularTextRun)")
	}
	return messages
}

func normalAutofitRequiresSimplifiedSizing(element slideElement, bounds image.Rectangle, dpi int) bool {
	if !element.HasNormAutofit {
		return false
	}
	if bounds.Empty() {
		return true
	}
	if element.HasFontScalePct {
		return false
	}
	startScale := element.FontScalePct
	if startScale <= 0 {
		startScale = 100000
	}
	maxLines := normalAutofitMaxSoftLines(element)
	_, ok := largestFittingNormalAutofitScale(element, bounds, startScale, maxLines, dpi)
	return !ok
}

func shapeAutofitLayoutSupported(element slideElement) bool {
	if !element.HasShapeAutofit {
		return true
	}
	if strings.TrimSpace(element.Text) == "" {
		return true
	}
	return normalizedRotationDegrees(element.Rotation) == 0
}

func fontResolutionUnsupportedMessageForFamily(fontFamily string, bold bool, italic bool) string {
	resolvedFamily := strings.TrimSpace(fontFamily)
	if resolvedFamily == "" || exactFontFamilyStyleAvailable(resolvedFamily, bold, italic) {
		return ""
	}
	if supportedFontSubstituteAvailable(resolvedFamily, bold, italic) {
		if fontSubstitutionShouldReport(resolvedFamily) {
			return fmt.Sprintf("text requested font family %q but rendered with a metric-compatible substitute font", resolvedFamily)
		}
		return ""
	}
	return fmt.Sprintf("text requested font family %q but rendered with a generic fallback font", resolvedFamily)
}

func fontSubstitutionShouldReport(fontFamily string) bool {
	switch normalizedFontFamily(fontFamily) {
	case "calibri", "calibri light":
		return true
	default:
		return false
	}
}

func fontResolutionMessageKey(fontFamily string, bold bool, italic bool) string {
	family := normalizedFontFamily(fontFamily)
	if family == "" {
		return ""
	}
	if !bold && !italic {
		return family
	}
	return family + ":" + fontStyleKey(bold, italic)
}

func openFontFace(fontSize int, bold bool, italic bool, pointScale float64, fontFamily string) (font.Face, error) {
	return openFontFaceWithDPI(fontSize, bold, italic, pointScale, fontFamily, defaultOutputDPI)
}

func openFontFaceWithDPI(fontSize int, bold bool, italic bool, pointScale float64, fontFamily string, dpi int) (font.Face, error) {
	source, err := cachedResolvedFontSource(fontFamily, bold, italic)
	if err != nil {
		return nil, errors.New("no supported font found")
	}
	parsed, err := cachedParsedFontData(source, bold, italic)
	if err != nil {
		return nil, err
	}
	if fontSize <= 0 {
		fontSize = 1800
	}
	return opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    fallbackFontPointSizeWithScaleAndFamily(fontSize, bold, italic, pointScale, fontFamily),
		DPI:     float64(normalizeOutputDPI(dpi)),
		Hinting: font.HintingNone,
	})
}

func cachedResolvedFontSource(fontFamily string, bold bool, italic bool) (fontSource, error) {
	key := normalizedFontFamily(fontFamily) + ":" + fontStyleKey(bold, italic)
	if cached, ok := resolvedFontSourceCache.Load(key); ok {
		return cached.(fontSource), nil
	}
	source, err := resolveFontSource(fontFamily, bold, italic)
	if err != nil {
		return fontSource{}, err
	}
	actual, _ := resolvedFontSourceCache.LoadOrStore(key, source)
	return actual.(fontSource), nil
}

func cachedParsedFontData(source fontSource, bold bool, italic bool) (*opentype.Font, error) {
	if source.Label == "" {
		return parseFontData(source.Data, bold, italic)
	}
	key := source.Label + ":" + fontStyleKey(bold, italic)
	if cached, ok := parsedOpenTypeFontCache.Load(key); ok {
		return cached.(*opentype.Font), nil
	}
	parsed, err := parseFontData(source.Data, bold, italic)
	if err != nil {
		return nil, err
	}
	actual, _ := parsedOpenTypeFontCache.LoadOrStore(key, parsed)
	return actual.(*opentype.Font), nil
}

func resolveFontSource(fontFamily string, bold bool, italic bool) (fontSource, error) {
	if fontPath := firstExistingPath(exactFontCandidatesForFamily(fontFamily, bold, italic)); fontPath != "" {
		return readFontPath(fontPath)
	}
	if source, ok := substituteFontSourceForFamily(fontFamily, bold, italic); ok {
		return source, nil
	}
	if fontPath := firstExistingPath(fontCandidates(bold, italic)); fontPath != "" {
		return readFontPath(fontPath)
	}
	return fontSource{}, errors.New("no supported font found")
}

func readFontPath(fontPath string) (fontSource, error) {
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return fontSource{}, err
	}
	return fontSource{Data: data, Label: fontPath}, nil
}

func fallbackFontPointSize(fontSize int, bold bool, italic bool) float64 {
	return fallbackFontPointSizeWithScale(fontSize, bold, italic, 0)
}

func fallbackFontPointSizeWithScale(fontSize int, bold bool, italic bool, pointScale float64) float64 {
	return fallbackFontPointSizeWithScaleAndFamily(fontSize, bold, italic, pointScale, "")
}

func fallbackFontPointSizeWithScaleAndFamily(fontSize int, bold bool, italic bool, pointScale float64, fontFamily string) float64 {
	if pointScale > 0 {
		return float64(fontSize) / 100 * pointScale
	}
	return float64(fontSize) / 100
}

func parseFontData(data []byte, bold bool, italic bool) (*opentype.Font, error) {
	parsed, err := opentype.Parse(data)
	if err == nil {
		return parsed, nil
	}
	collection, collectionErr := opentype.ParseCollection(data)
	if collectionErr != nil {
		return nil, err
	}
	return fontFromCollection(collection, bold, italic)
}

func fontFromCollection(collection *opentype.Collection, bold bool, italic bool) (*opentype.Font, error) {
	bestScore := -1
	var best *opentype.Font
	for index := 0; index < collection.NumFonts(); index++ {
		fontItem, err := collection.Font(index)
		if err != nil {
			continue
		}
		score := fontCollectionStyleScore(fontItem, bold, italic)
		if score > bestScore {
			bestScore = score
			best = fontItem
		}
	}
	if best == nil {
		return nil, errors.New("font collection has no usable fonts")
	}
	return best, nil
}

func fontCollectionStyleScore(fontItem *opentype.Font, bold bool, italic bool) int {
	subfamily, _ := fontItem.Name(nil, sfnt.NameIDSubfamily)
	full, _ := fontItem.Name(nil, sfnt.NameIDFull)
	name := strings.ToLower(subfamily + " " + full)
	hasBold := strings.Contains(name, "bold")
	hasItalic := strings.Contains(name, "italic") || strings.Contains(name, "oblique")
	score := 0
	if hasBold == bold {
		score += 2
	}
	if hasItalic == italic {
		score += 2
	}
	if !bold && !italic && strings.Contains(name, "regular") {
		score++
	}
	return score
}

func fontCandidatesForFamily(fontFamily string, bold bool, italic bool) []string {
	exact := exactFontCandidatesForFamily(fontFamily, bold, italic)
	if firstExistingPath(exact) != "" {
		return append(exact, fontCandidates(bold, italic)...)
	}
	return fontCandidates(bold, italic)
}

func exactFontCandidatesForFamily(fontFamily string, bold bool, italic bool) []string {
	configured := configuredFontCandidatesForFamily(fontFamily, bold, italic)
	switch normalizedFontFamily(fontFamily) {
	case "calibri light":
		return append(configured, calibriFontCandidates("Calibri Light", bold, italic)...)
	case "calibri":
		return append(configured, calibriFontCandidates("Calibri", bold, italic)...)
	case "trebuchet ms":
		return append(configured, trebuchetMSFontCandidates(bold, italic)...)
	case "times new roman":
		return append(configured, timesNewRomanFontCandidates(bold, italic)...)
	case "wingdings", "wingdings 2", "wingdings 3":
		return append(configured, wingdingsFontCandidates(fontFamily)...)
	case "segoe ui symbol":
		return append(configured, segoeUISymbolFontCandidates()...)
	case "segoe ui historic":
		return append(configured, segoeUIHistoricFontCandidates()...)
	case "arial":
		var candidates []string
		switch {
		case bold && italic:
			candidates = []string{
				"/System/Library/Fonts/Supplemental/Arial Bold Italic.ttf",
				"/Library/Fonts/Arial Bold Italic.ttf",
				"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
				"/System/Library/Fonts/Supplemental/Arial Italic.ttf",
				"/System/Library/Fonts/Supplemental/Arial.ttf",
				"/Library/Fonts/Arial.ttf",
			}
		case bold:
			candidates = []string{
				"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
				"/Library/Fonts/Arial Bold.ttf",
				"/System/Library/Fonts/Supplemental/Arial.ttf",
				"/Library/Fonts/Arial.ttf",
			}
		case italic:
			candidates = []string{
				"/System/Library/Fonts/Supplemental/Arial Italic.ttf",
				"/Library/Fonts/Arial Italic.ttf",
				"/System/Library/Fonts/Supplemental/Arial.ttf",
				"/Library/Fonts/Arial.ttf",
			}
		default:
			candidates = []string{
				"/System/Library/Fonts/Supplemental/Arial.ttf",
				"/Library/Fonts/Arial.ttf",
			}
		}
		return append(configured, candidates...)
	default:
		return configured
	}
}

func configuredFontCandidatesForFamily(fontFamily string, bold bool, italic bool) []string {
	entries := parseConfiguredFontMap(os.Getenv("PUPPT_FONT_MAP"))
	if len(entries) == 0 {
		return nil
	}
	keys := []string{configuredFontMapKey(fontFamily, bold, italic)}
	if bold || italic {
		keys = append(keys, configuredFontMapKey(fontFamily, false, false))
	}
	var candidates []string
	seen := map[string]bool{}
	for _, key := range keys {
		for _, candidate := range entries[key] {
			if candidate == "" || seen[candidate] {
				continue
			}
			seen[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func parseConfiguredFontMap(raw string) map[string][]string {
	entries := map[string][]string{}
	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		key = configuredFontMapRawKey(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		entries[key] = append(entries[key], value)
	}
	return entries
}

func configuredFontMapRawKey(raw string) string {
	family, style, hasStyle := strings.Cut(strings.TrimSpace(raw), ":")
	family = normalizedFontFamily(family)
	if family == "" {
		return ""
	}
	if !hasStyle {
		return family
	}
	styleKey := normalizedFontStyleKey(style)
	if styleKey == "" {
		return family
	}
	return family + ":" + styleKey
}

func configuredFontMapKey(fontFamily string, bold bool, italic bool) string {
	family := normalizedFontFamily(fontFamily)
	if family == "" || (!bold && !italic) {
		return family
	}
	return family + ":" + fontStyleKey(bold, italic)
}

func normalizedFontStyleKey(style string) string {
	style = strings.ToLower(strings.TrimSpace(style))
	style = strings.NewReplacer("-", "", "_", "", " ", "").Replace(style)
	switch style {
	case "", "regular", "normal":
		return ""
	case "bold":
		return "bold"
	case "italic", "oblique":
		return "italic"
	case "bolditalic", "boldoblique":
		return "bolditalic"
	default:
		return style
	}
}

func fontStyleKey(bold bool, italic bool) string {
	switch {
	case bold && italic:
		return "bolditalic"
	case bold:
		return "bold"
	case italic:
		return "italic"
	default:
		return ""
	}
}

func substituteFontSourceForFamily(fontFamily string, bold bool, italic bool) (fontSource, bool) {
	switch normalizedFontFamily(fontFamily) {
	case "calibri", "calibri light":
		if normalizedFontFamily(fontFamily) == "calibri light" && bold {
			bold = false
		}
		if fontPath := firstExistingPath(carlitoFontCandidates(bold, italic)); fontPath != "" {
			source, err := readFontPath(fontPath)
			if err == nil {
				return source, true
			}
		}
		source, err := readBundledFont(carlitoAssetPath(bold, italic))
		if err != nil {
			return fontSource{}, false
		}
		return source, true
	case "segoe ui historic", "segoe ui symbol":
		if source, ok := sansSerifSubstituteFontSource(bold, italic); ok {
			return source, true
		}
		return fontSource{}, false
	default:
		return fontSource{}, false
	}
}

func sansSerifSubstituteFontSource(bold bool, italic bool) (fontSource, bool) {
	if fontPath := firstExistingPath(fontCandidates(bold, italic)); fontPath != "" {
		source, err := readFontPath(fontPath)
		if err == nil {
			return source, true
		}
	}
	return fontSource{}, false
}

func carlitoFontCandidates(bold bool, italic bool) []string {
	styleName := "Regular"
	fileName := "Carlito-Regular.ttf"
	switch {
	case bold && italic:
		styleName = "BoldItalic"
		fileName = "Carlito-BoldItalic.ttf"
	case bold:
		styleName = "Bold"
		fileName = "Carlito-Bold.ttf"
	case italic:
		styleName = "Italic"
		fileName = "Carlito-Italic.ttf"
	}
	candidates := []string{
		"/System/Library/PrivateFrameworks/FontServices.framework/Versions/A/Resources/Fonts/ApplicationSupport/Carlito.ttc",
		"/Library/Fonts/Carlito.ttc",
		"/System/Library/Fonts/Supplemental/Carlito.ttc",
		"/Library/Fonts/" + fileName,
		"/System/Library/Fonts/Supplemental/" + fileName,
		"/usr/share/fonts/truetype/crosextra/" + fileName,
		"/usr/share/fonts/truetype/carlito/" + fileName,
		"/usr/share/fonts/opentype/carlito/" + fileName,
	}
	if styleName != "Regular" {
		candidates = append(candidates,
			"/Library/Fonts/Carlito-"+styleName+".ttf",
			"/System/Library/Fonts/Supplemental/Carlito-"+styleName+".ttf",
		)
	}
	return candidates
}

func supportedFontSubstituteAvailable(fontFamily string, bold bool, italic bool) bool {
	_, ok := substituteFontSourceForFamily(fontFamily, bold, italic)
	return ok
}

func exactFontFamilyAvailable(fontFamily string) bool {
	return exactFontFamilyStyleAvailable(fontFamily, false, false)
}

func exactFontFamilyStyleAvailable(fontFamily string, bold bool, italic bool) bool {
	if firstExistingPath(configuredFontCandidatesForFamily(fontFamily, false, false)) != "" {
		return true
	}
	switch normalizedFontFamily(fontFamily) {
	case "arial", "calibri", "calibri light", "times new roman", "trebuchet ms", "wingdings", "wingdings 2", "wingdings 3", "segoe ui symbol", "segoe ui historic":
		return firstExistingPath(exactFontCandidatesForFamily(fontFamily, bold, italic)) != ""
	default:
		return firstExistingPath(configuredFontCandidatesForFamily(fontFamily, bold, italic)) != ""
	}
}

func normalizedFontFamily(fontFamily string) string {
	return strings.ToLower(strings.TrimSpace(fontFamily))
}

func readBundledFont(assetPath string) (fontSource, error) {
	data, err := bundledFontFS.ReadFile(assetPath)
	if err != nil {
		return fontSource{}, err
	}
	return fontSource{Data: data, Label: strings.TrimPrefix(assetPath, "assets/fonts/")}, nil
}

func carlitoAssetPath(bold bool, italic bool) string {
	name := "Carlito-Regular.ttf"
	switch {
	case bold && italic:
		name = "Carlito-BoldItalic.ttf"
	case bold:
		name = "Carlito-Bold.ttf"
	case italic:
		name = "Carlito-Italic.ttf"
	}
	return path.Join("assets/fonts/carlito", name)
}

func segoeUISymbolFontCandidates() []string {
	return []string{
		`C:\Windows\Fonts\seguisym.ttf`,
		`C:\Windows\Fonts\Seguisym.ttf`,
		"/Windows/Fonts/seguisym.ttf",
		"/Windows/Fonts/Seguisym.ttf",
		"/Library/Fonts/Microsoft/Seguisym.ttf",
		"/Library/Fonts/Microsoft/seguisym.ttf",
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", "Seguisym.ttf"),
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", "seguisym.ttf"),
	}
}

func segoeUIHistoricFontCandidates() []string {
	return []string{
		`C:\Windows\Fonts\seguihis.ttf`,
		`C:\Windows\Fonts\Seguihis.ttf`,
		"/Windows/Fonts/seguihis.ttf",
		"/Windows/Fonts/Seguihis.ttf",
		"/Library/Fonts/Microsoft/Seguihis.ttf",
		"/Library/Fonts/Microsoft/seguihis.ttf",
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", "Seguihis.ttf"),
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", "seguihis.ttf"),
	}
}

func wingdingsFontCandidates(family string) []string {
	name := strings.TrimSpace(family)
	return []string{
		"/System/Library/Fonts/Supplemental/" + name + ".ttf",
		"/Library/Fonts/" + name + ".ttf",
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", name+".ttf"),
	}
}

func calibriFontCandidates(family string, bold bool, italic bool) []string {
	styleSuffix := ""
	switch {
	case bold && italic:
		styleSuffix = " Bold Italic"
	case bold:
		styleSuffix = " Bold"
	case italic:
		styleSuffix = " Italic"
	}
	names := []string{family + styleSuffix + ".ttf", family + styleSuffix + ".otf"}
	names = append(names, calibriOfficeFileNames(family, bold, italic)...)
	if styleSuffix != "" {
		names = append(names, family+".ttf", family+".otf")
		names = append(names, calibriOfficeFileNames(family, false, false)...)
	}
	roots := []string{
		"/Library/Fonts",
		"/Library/Fonts/Microsoft",
		"/System/Library/Fonts/Supplemental",
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts"),
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts", "Microsoft"),
		filepath.Join("/Applications", "Microsoft Word.app", "Contents", "Resources", "DFonts"),
		filepath.Join("/Applications", "Microsoft Excel.app", "Contents", "Resources", "DFonts"),
		filepath.Join(os.Getenv("HOME"), "Applications", "Microsoft Word.app", "Contents", "Resources", "DFonts"),
		filepath.Join(os.Getenv("HOME"), "Applications", "Microsoft Excel.app", "Contents", "Resources", "DFonts"),
		filepath.Join(os.Getenv("HOME"), ".cache", "puppt", "fonts", "*", "expanded", "*", "Payload", "Microsoft Word.app", "Contents", "Resources", "DFonts"),
		filepath.Join(os.Getenv("HOME"), ".cache", "puppt", "fonts", "*", "expanded", "*", "Payload", "Microsoft Excel.app", "Contents", "Resources", "DFonts"),
		filepath.Join(os.Getenv("HOME"), "Library", "Group Containers", "UBF8T346G9.Office", "FontCache", "*", "CloudFonts"),
		filepath.Join(os.Getenv("HOME"), "Library", "Group Containers", "UBF8T346G9.Office", "FontCache", "*", "CloudFonts", family),
		"/usr/local/share/fonts",
		"/usr/share/fonts",
		"/usr/share/fonts/truetype/msttcorefonts",
		filepath.Join(os.Getenv("HOME"), ".local", "share", "fonts"),
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		roots = append(roots,
			filepath.Join(localAppData, "Microsoft", "FontCache", "*", "CloudFonts"),
			filepath.Join(localAppData, "Microsoft", "FontCache", "*", "CloudFonts", family),
		)
	}
	var paths []string
	for _, root := range roots {
		for _, name := range names {
			paths = append(paths, filepath.Join(root, name))
		}
	}
	return paths
}

func calibriOfficeFileNames(family string, bold bool, italic bool) []string {
	switch normalizedFontFamily(family) {
	case "calibri light":
		switch {
		case italic:
			return []string{"calibrili.ttf"}
		default:
			return []string{"calibril.ttf"}
		}
	case "calibri":
		switch {
		case bold && italic:
			return []string{"Calibriz.ttf", "calibriz.ttf"}
		case bold:
			return []string{"Calibrib.ttf", "calibrib.ttf"}
		case italic:
			return []string{"Calibrii.ttf", "calibrii.ttf"}
		default:
			return []string{"Calibri.ttf", "calibri.ttf"}
		}
	default:
		return nil
	}
}

func trebuchetMSFontCandidates(bold bool, italic bool) []string {
	switch {
	case bold && italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Trebuchet MS Bold Italic.ttf",
			"/Library/Fonts/Trebuchet MS Bold Italic.ttf",
			"/System/Library/Fonts/Supplemental/Trebuchet MS Bold.ttf",
			"/System/Library/Fonts/Supplemental/Trebuchet MS Italic.ttf",
			"/System/Library/Fonts/Supplemental/Trebuchet MS.ttf",
			"/Library/Fonts/Trebuchet MS.ttf",
		}
	case bold:
		return []string{
			"/System/Library/Fonts/Supplemental/Trebuchet MS Bold.ttf",
			"/Library/Fonts/Trebuchet MS Bold.ttf",
			"/System/Library/Fonts/Supplemental/Trebuchet MS.ttf",
			"/Library/Fonts/Trebuchet MS.ttf",
		}
	case italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Trebuchet MS Italic.ttf",
			"/Library/Fonts/Trebuchet MS Italic.ttf",
			"/System/Library/Fonts/Supplemental/Trebuchet MS.ttf",
			"/Library/Fonts/Trebuchet MS.ttf",
		}
	default:
		return []string{
			"/System/Library/Fonts/Supplemental/Trebuchet MS.ttf",
			"/Library/Fonts/Trebuchet MS.ttf",
		}
	}
}

func timesNewRomanFontCandidates(bold bool, italic bool) []string {
	switch {
	case bold && italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Times New Roman Bold Italic.ttf",
			"/Library/Fonts/Times New Roman Bold Italic.ttf",
			"/System/Library/Fonts/Supplemental/Times New Roman Bold.ttf",
			"/System/Library/Fonts/Supplemental/Times New Roman Italic.ttf",
			"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
			"/Library/Fonts/Times New Roman.ttf",
			"/System/Library/Fonts/Times.ttc",
		}
	case bold:
		return []string{
			"/System/Library/Fonts/Supplemental/Times New Roman Bold.ttf",
			"/Library/Fonts/Times New Roman Bold.ttf",
			"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
			"/Library/Fonts/Times New Roman.ttf",
			"/System/Library/Fonts/Times.ttc",
		}
	case italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Times New Roman Italic.ttf",
			"/Library/Fonts/Times New Roman Italic.ttf",
			"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
			"/Library/Fonts/Times New Roman.ttf",
			"/System/Library/Fonts/Times.ttc",
		}
	default:
		return []string{
			"/System/Library/Fonts/Supplemental/Times New Roman.ttf",
			"/Library/Fonts/Times New Roman.ttf",
			"/System/Library/Fonts/Times.ttc",
		}
	}
}

func fontCandidates(bold bool, italic bool) []string {
	switch {
	case bold && italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Arial Bold Italic.ttf",
			"/Library/Fonts/Arial Bold Italic.ttf",
			"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
			"/System/Library/Fonts/Supplemental/Arial Italic.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
		}
	case bold:
		return []string{
			"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
			"/Library/Fonts/Arial Bold.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
		}
	case italic:
		return []string{
			"/System/Library/Fonts/Supplemental/Arial Italic.ttf",
			"/Library/Fonts/Arial Italic.ttf",
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
		}
	default:
		return []string{
			"/System/Library/Fonts/Supplemental/Arial.ttf",
			"/Library/Fonts/Arial.ttf",
			"/System/Library/Fonts/Helvetica.ttc",
		}
	}
}

func firstExistingPath(paths []string) string {
	for _, candidate := range paths {
		if strings.ContainsAny(candidate, "*?[") {
			matches, err := filepath.Glob(candidate)
			if err == nil {
				sort.Strings(matches)
				if path := firstExistingPath(matches); path != "" {
					return path
				}
			}
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
