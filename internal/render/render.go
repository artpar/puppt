package render

import (
	"bytes"
	"compress/zlib"
	"context"
	"embed"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

const (
	commandName            = "render"
	emuPerInch             = 914400
	defaultOutputDPI       = 72
	defaultSlideCX         = 12192000
	defaultSlideCY         = 6858000
	defaultTextInsetXEMU   = 91440
	defaultTextInsetYEMU   = 45720
	defaultTextTabPixels   = defaultOutputDPI
	customBezierSegments   = 48
	unsupportedCode        = "render_unsupported_object"
	partialUnsupportedCode = "render_partial_object"
	diagramDataRelType     = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/diagramData"
	diagramDrawingRelType  = "http://schemas.microsoft.com/office/2007/relationships/diagramDrawing"
	themeRelType           = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme"
)

var displayP3CICPChunkData = []byte{12, 13, 0, 1}

//go:embed assets/fonts/carlito/*.ttf
var bundledFontFS embed.FS

type Options struct {
	SlideNumber int
	OutputPath  string
	DPI         int
}

type slideSize struct {
	CX int64
	CY int64
}

type backgroundPaint struct {
	Color       color.RGBA
	HasGradient bool
	Gradient    gradientPaint
	Part        string
}

type gradientPaint struct {
	Stops          []gradientStop
	Path           string
	Angle          int64
	HasAngle       bool
	HasScaled      bool
	Scaled         bool
	HasFillRect    bool
	FillRect       relativeRect
	FullySupported bool
}

type gradientStop struct {
	Position int64
	Color    color.RGBA
}

type relativeRect struct {
	Left   int64
	Top    int64
	Right  int64
	Bottom int64
}

type textStyle struct {
	FontSize        int
	Bold            bool
	HasTextColor    bool
	TextColor       color.RGBA
	TextAlign       string
	ParagraphStyles map[int]paragraphStyle
}

type paragraphStyle struct {
	HasMarginLeft    bool
	MarginLeft       int64
	HasMarginRight   bool
	MarginRight      int64
	HasIndent        bool
	Indent           int64
	FontFamily       string
	FontSize         int
	HasSpaceBefore   bool
	SpaceBefore      int
	SpaceBeforePct   int
	HasSpaceAfter    bool
	SpaceAfter       int
	SpaceAfterPct    int
	HasLineSpacing   bool
	LineSpacingPct   int
	HasDefaultTab    bool
	DefaultTabSize   int64
	Bullet           string
	BulletFontFamily string
	BulletFontTx     bool
	BulletFontSize   int
	BulletSizePct    int
	BulletSizeTx     bool
	HasAutoNumber    bool
	AutoNumberType   string
	AutoNumberStart  int
	HasBulletColor   bool
	BulletColor      color.RGBA
	BulletColorTx    bool
	Bold             bool
	Italic           bool
	HasCharSpacing   bool
	CharSpacing      int
	TextAlign        string
	HasTextColor     bool
	TextColor        color.RGBA
	NoBullet         bool
}

type themeFonts struct {
	MajorLatin string
	MajorEA    string
	MajorCS    string
	MinorLatin string
	MinorEA    string
	MinorCS    string
}

type themeFillStyles struct {
	Styles           []*xmlNode
	BackgroundStyles []*xmlNode
}

type themeLineStyles struct {
	Styles []*xmlNode
}

type backgroundStyleResolver func(idx int64, placeholderColor color.RGBA) (backgroundPaint, bool)

type slideElement struct {
	Kind                       string
	ID                         string
	Name                       string
	Text                       string
	TextParagraphs             []textParagraph
	HasTable                   bool
	Table                      tableModel
	EmbedID                    string
	SVGEmbedID                 string
	DiagramDataID              string
	OffX                       int64
	OffY                       int64
	ExtCX                      int64
	ExtCY                      int64
	HasTransform               bool
	HasTextTransform           bool
	TextOffX                   int64
	TextOffY                   int64
	TextExtCX                  int64
	TextExtCY                  int64
	HasCrop                    bool
	CropLeft                   int64
	CropTop                    int64
	CropRight                  int64
	CropBottom                 int64
	FlipH                      bool
	FlipV                      bool
	HasImageAlphaModFix        bool
	ImageAlphaModFixPct        int64
	HasBlipRotWithShape        bool
	BlipRotWithShape           bool
	BWMode                     string
	HasRotation                bool
	Rotation                   int
	Rendered                   bool
	UnsupportedNote            string
	IsPlaceholder              bool
	PlaceholderType            string
	PlaceholderIdx             string
	PrstGeom                   string
	PrstGeomAdjustments        map[string]int64
	HasFill                    bool
	FillColor                  color.RGBA
	HasFillGradient            bool
	FillGradient               gradientPaint
	NoFill                     bool
	HasLine                    bool
	LineColor                  color.RGBA
	NoLine                     bool
	LineWidth                  int64
	HasLineWidth               bool
	LineDash                   string
	HasLineDash                bool
	LineCap                    string
	HasLineCap                 bool
	LineAlign                  string
	HasLineAlign               bool
	HasLineMarker              bool
	HeadLineMarker             string
	HeadLineMarkerWidth        string
	HeadLineMarkerLength       string
	TailLineMarker             string
	TailLineMarkerWidth        string
	TailLineMarkerLength       string
	HasShadow                  bool
	HasEffectProperties        bool
	ShadowColor                color.RGBA
	ShadowBlur                 int64
	ShadowDistance             int64
	ShadowDirection            int64
	ShadowAlignment            string
	HasShadowRotateWithShape   bool
	ShadowRotateWithShape      bool
	HasShadowScaleX            bool
	ShadowScaleX               int64
	HasShadowScaleY            bool
	ShadowScaleY               int64
	HasShadowSkewX             bool
	ShadowSkewX                int64
	HasShadowSkewY             bool
	ShadowSkewY                int64
	HasSoftEdge                bool
	SoftEdgeRadius             int64
	HasShape3D                 bool
	Shape3DFeatures            []string
	CustomPath                 []pathPoint
	CustomPathCommands         []pathCommand
	CustomPathUnsupported      []string
	FontFamily                 string
	FontSize                   int
	FontPointScale             float64
	Italic                     bool
	HasTextColor               bool
	TextColor                  color.RGBA
	TextAlign                  string
	PlaceholderParagraphStyles map[int]paragraphStyle
	HasBodyProperties          bool
	TextAnchor                 string
	HasTextWrap                bool
	TextWrap                   string
	HasTextHorizontalOverflow  bool
	TextHorizontalOverflow     string
	HasTextVerticalOverflow    bool
	TextVerticalOverflow       string
	HasTextVertical            bool
	TextVertical               string
	HasTextBodyRotation        bool
	TextBodyRotation           int
	HasTextColumns             bool
	TextColumnCount            int
	HasTextAnchorCenter        bool
	TextAnchorCenter           bool
	IncludeFirstLastSpacing    bool
	HasFirstLastSpacing        bool
	HasNoAutofit               bool
	HasShapeAutofit            bool
	HasNormAutofit             bool
	HasFontScalePct            bool
	FontScalePct               int
	HasLineSpacingReductionPct bool
	LineSpacingReductionPct    int
	InsetLeft                  int64
	InsetTop                   int64
	InsetRight                 int64
	InsetBottom                int64
	HasInsets                  bool
}

type textParagraph struct {
	Text             string
	Bullet           string
	HasAutoNumber    bool
	Level            int
	TextAlign        string
	FontFamily       string
	BulletFontFamily string
	BulletFontTx     bool
	FontSize         int
	Bold             bool
	Italic           bool
	HasCharSpacing   bool
	CharSpacing      int
	HasTextColor     bool
	TextColor        color.RGBA
	NoBullet         bool
	BulletFontSize   int
	BulletSizePct    int
	BulletSizeTx     bool
	HasBulletColor   bool
	BulletColor      color.RGBA
	BulletColorTx    bool
	HasMarginLeft    bool
	MarginLeft       int64
	HasMarginRight   bool
	MarginRight      int64
	HasIndent        bool
	Indent           int64
	HasSpaceBefore   bool
	SpaceBefore      int
	SpaceBeforePct   int
	HasSpaceAfter    bool
	SpaceAfter       int
	SpaceAfterPct    int
	HasLineSpacing   bool
	LineSpacingPct   int
	TabStops         []int64
	HasDefaultTab    bool
	DefaultTabSize   int64
	Runs             []textRun
}

type textRun struct {
	Text              string
	FieldType         string
	FontFamily        string
	FontSize          int
	HasBold           bool
	Bold              bool
	HasItalic         bool
	Italic            bool
	Underline         bool
	HasUnderlineColor bool
	UnderlineColor    color.RGBA
	Strike            string
	Baseline          int
	HasCharSpacing    bool
	CharSpacing       int
	HasKern           bool
	KernMinFontSize   int
	HasTextColor      bool
	TextColor         color.RGBA
	HasHighlightColor bool
	HighlightColor    color.RGBA
}

type tableModel struct {
	Columns             []int64
	Rows                []tableRow
	StyleID             string
	FirstRow            bool
	FirstCol            bool
	LastRow             bool
	LastCol             bool
	BandRow             bool
	BandCol             bool
	UnsupportedFeatures []string
}

type tableRow struct {
	Height    int64
	HasHeight bool
	Cells     []tableCell
}

type tableCell struct {
	Text           string
	TextParagraphs []textParagraph
	ColSpan        int
	HMerge         bool
	RowSpan        int
	VMerge         bool
	FontSize       int
	HasFontSize    bool
	HasTextColor   bool
	TextColor      color.RGBA
	TextAlign      string
	TextAnchor     string
	HasFill        bool
	FillColor      color.RGBA
	NoFill         bool
	HasMargins     bool
	MarginLeft     int64
	MarginRight    int64
	MarginTop      int64
	MarginBottom   int64
	BorderLeft     tableCellBorder
	BorderRight    tableCellBorder
	BorderTop      tableCellBorder
	BorderBottom   tableCellBorder
}

type headerFooterSettings struct {
	SlideNumber    bool
	HasSlideNumber bool
	DateTime       bool
	HasDateTime    bool
	Footer         bool
	HasFooter      bool
	Header         bool
	HasHeader      bool
}

type tableCellBorder struct {
	Specified bool
	HasLine   bool
	NoLine    bool
	Color     color.RGBA
	Width     int64
	Dash      string
	Cap       string
	Align     string
	Join      string
	Compound  string
}

type tableStyleSet struct {
	DefaultID string
	Styles    map[string]tableStyle
}

type tableStyle struct {
	ID                  string
	Name                string
	HasBackground       bool
	Background          backgroundPaint
	HasBackgroundEffect bool
	BackgroundEffect    themeEffectStyle
	Regions             map[string]tableStyleRegion
}

type tableStyleRegion struct {
	HasFill      bool
	NoFill       bool
	FillColor    color.RGBA
	HasTextColor bool
	TextColor    color.RGBA
	FontFamily   string
	HasBold      bool
	Bold         bool
	HasItalic    bool
	Italic       bool
	Borders      tableStyleBorders
}

type tableStyleBorders struct {
	Left    tableCellBorder
	Right   tableCellBorder
	Top     tableCellBorder
	Bottom  tableCellBorder
	InsideH tableCellBorder
	InsideV tableCellBorder
}

type xmlNode struct {
	Name     string
	Attrs    []xml.Attr
	Children []*xmlNode
	Text     string
}

type renderTransform struct {
	ScaleX  float64
	ScaleY  float64
	OffsetX float64
	OffsetY float64
}

type pathPoint struct {
	X float64
	Y float64
}

type pathCommand struct {
	Kind   string
	Points []pathPoint
}

type themeColors map[string]color.RGBA

type themeEffectStyles struct {
	Styles []*xmlNode
}

type themeEffectStyle struct {
	HasShadow                bool
	ShadowColor              color.RGBA
	ShadowBlur               int64
	ShadowDistance           int64
	ShadowDirection          int64
	ShadowAlignment          string
	HasShadowRotateWithShape bool
	ShadowRotateWithShape    bool
	HasShadowScaleX          bool
	ShadowScaleX             int64
	HasShadowScaleY          bool
	ShadowScaleY             int64
	HasShadowSkewX           bool
	ShadowSkewX              int64
	HasShadowSkewY           bool
	ShadowSkewY              int64
	HasShape3D               bool
	Shape3DFeatures          []string
}

// Render writes a PNG for one slide and returns the stable command result used
// by the CLI. Unsupported visible objects are reported explicitly.
func Render(ctx context.Context, inputPath string, options Options) (model.CommandResult, error) {
	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       commandName,
		Status:        "error",
		Input:         inputPath,
		Warnings:      []model.Warning{},
		Errors:        []model.ErrorItem{},
		Unsupported:   []model.SkipItem{},
		Summary:       model.Summary{Human: "Render failed."},
	}
	if options.OutputPath != "" {
		result.Output = &options.OutputPath
	}
	if options.OutputPath == "" {
		return result, errors.New("render output path is required")
	}

	pkg, err := pptx.Open(ctx, inputPath)
	if err != nil {
		return result, err
	}
	if options.SlideNumber < 1 || options.SlideNumber > len(pkg.SlideParts) {
		return result, fmt.Errorf("slide %d out of range 1..%d", options.SlideNumber, len(pkg.SlideParts))
	}

	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	dpi := normalizeOutputDPI(options.DPI)
	width := emuToPixelsAtDPI(size.CX, dpi)
	height := emuToPixelsAtDPI(size.CY, dpi)
	if width <= 0 || height <= 0 {
		return result, fmt.Errorf("invalid slide size %dx%d EMU", size.CX, size.CY)
	}

	slidePart := pkg.SlideParts[options.SlideNumber-1]
	theme := packageThemeColors(pkg)
	fonts := packageThemeFonts(pkg)
	themeForPart := func(part string) themeColors {
		return themeColorsForPart(pkg, part, theme)
	}
	fontsForPart := func(part string) themeFonts {
		return themeFontsForPart(pkg, part, fonts)
	}
	renderParts := inheritedRenderParts(pkg, slidePart)
	paintParts := visibleRenderParts(pkg, slidePart, renderParts)
	placeholderSources := inheritedPlaceholderSourcesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	textStyles := inheritedTextStylesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	background := inheritedBackgroundWithThemeResolver(pkg, renderParts, themeForPart)
	headerFooter := inheritedHeaderFooterSettings(pkg, renderParts)
	if !presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) && slideUsesTitleLayout(pkg, slidePart) {
		headerFooter = headerFooterSettings{}
	}
	inheritedHeaderFooterPart := inheritedHeaderFooterRenderPart(pkg, paintParts, slidePart, headerFooter)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var unsupported []model.SkipItem
	if background.HasGradient {
		drawGradientBackground(img, background.Gradient)
		if !background.Gradient.FullySupported {
			unsupported = append(unsupported, unsupportedItem(background.Part, partialUnsupportedCode, "slide background gradient was rendered with simplified layout"))
		}
	} else {
		draw.Draw(img, img.Bounds(), &image.Uniform{C: background.Color}, image.Point{}, draw.Src)
	}

	for _, renderPart := range paintParts {
		partTheme := themeForPart(renderPart)
		partFonts := fontsForPart(renderPart)
		partLineStyles := themeLineStylesForPart(pkg, renderPart)
		effectStyles := themeEffectStylesForPart(pkg, renderPart)
		fillStyles := themeFillStylesForPart(pkg, renderPart)
		tableStyles := packageTableStyles(pkg, partTheme, partFonts, fillStyles, partLineStyles, effectStyles)
		elements := collectSlideElementsWithThemeEffectsAndFills(pkg.Parts[renderPart], partTheme, effectStyles, fillStyles, partLineStyles)
		if renderPart != slidePart {
			elements = filterInheritedPlaceholdersForRender(elements, placeholderSources, headerFooter, renderPart == inheritedHeaderFooterPart)
		} else {
			elements = resolveSlidePlaceholders(elements, placeholderSources)
			elements = applyInheritedTextStyles(elements, textStyles)
		}
		elements = applyInheritedTableTextStyles(elements, textStyles)
		elements = applyThemeFontFamilies(elements, partFonts)
		elements = resolveTextFields(elements, options.SlideNumber)
		unsupported = append(unsupported, renderElements(pkg, renderPart, size, img, elements, tableStyles)...)
		unsupported = append(unsupported, unsupportedItems(renderPart, elements)...)
		unsupported = append(unsupported, timingUnsupportedItems(renderPart, pkg.Parts[renderPart], elements)...)
	}
	applyDisplayP3OutputTransform(img)
	if err := writePNGWithDPI(options.OutputPath, img, dpi); err != nil {
		return result, err
	}

	result.Status = "ok"
	result.Summary = model.Summary{Human: fmt.Sprintf("Rendered slide %d to %s.", options.SlideNumber, options.OutputPath)}
	if len(unsupported) > 0 {
		result.Status = "partial"
		result.Summary = model.Summary{Human: fmt.Sprintf("Rendered slide %d with %d unsupported object(s).", options.SlideNumber, len(unsupported))}
		result.Unsupported = unsupported
	}
	result.Render = &model.Render{
		SlideNumber: options.SlideNumber,
		SlidePart:   slidePart,
		Width:       width,
		Height:      height,
	}
	return result, nil
}

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
	background := firstDescendant(root, "bg")
	if background == nil {
		return backgroundPaint{}, false
	}
	solidFill := firstDescendant(background, "solidFill")
	if solidFill != nil {
		if c, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			return backgroundPaint{Color: c}, true
		}
	}
	gradFill := firstDescendant(background, "gradFill")
	if gradFill != nil {
		if gradient, ok := parseGradientFill(gradFill, theme); ok {
			return backgroundPaint{Color: gradient.Stops[0].Color, HasGradient: true, Gradient: gradient}, true
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

func parseTableModel(tableNode *xmlNode, theme themeColors) tableModel {
	table := tableModel{UnsupportedFeatures: tableUnsupportedFeatureMessages(tableNode)}
	if properties := firstChild(tableNode, "tblPr"); properties != nil {
		table.FirstRow = attrValue(properties.Attrs, "firstRow") == "1"
		table.FirstCol = attrValue(properties.Attrs, "firstCol") == "1"
		table.LastRow = attrValue(properties.Attrs, "lastRow") == "1"
		table.LastCol = attrValue(properties.Attrs, "lastCol") == "1"
		table.BandRow = attrValue(properties.Attrs, "bandRow") == "1"
		table.BandCol = attrValue(properties.Attrs, "bandCol") == "1"
		if styleID := firstChild(properties, "tableStyleId"); styleID != nil {
			table.StyleID = strings.TrimSpace(styleID.Text)
		}
	}
	if grid := firstChild(tableNode, "tblGrid"); grid != nil {
		for _, gridCol := range childrenByName(grid, "gridCol") {
			if width := parseIntAttr(gridCol.Attrs, "w"); width > 0 {
				table.Columns = append(table.Columns, width)
			}
		}
	}
	for _, rowNode := range childrenByName(tableNode, "tr") {
		row := tableRow{}
		if attrValue(rowNode.Attrs, "h") != "" {
			row.HasHeight = true
			row.Height = parseIntAttr(rowNode.Attrs, "h")
		}
		for _, cellNode := range childrenByName(rowNode, "tc") {
			row.Cells = append(row.Cells, parseTableCell(cellNode, theme))
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}

func tableUnsupportedFeatureMessages(tableNode *xmlNode) []string {
	messages := map[string]bool{}
	collectTableUnsupportedFeatureMessages(tableNode, messages)
	return sortedKeys(messages)
}

func collectTableUnsupportedFeatureMessages(node *xmlNode, messages map[string]bool) {
	switch node.Name {
	case "gradFill", "blipFill", "pattFill", "grpFill":
		messages["uses a non-solid cell fill that was not rendered"] = true
	case "effectDag":
		if len(node.Children) > 0 {
			messages["uses effects that were not rendered"] = true
		}
	case "effectLst":
		if effectListHasVisibleEffects(node) {
			messages["uses effects that were not rendered"] = true
		}
	case "ln", "lnL", "lnR", "lnT", "lnB", "left", "right", "top", "bottom", "insideH", "insideV":
		collectTableLineUnsupportedFeatureMessages(node, messages)
	}
	for _, child := range node.Children {
		collectTableUnsupportedFeatureMessages(child, messages)
	}
}

func effectListHasVisibleEffects(node *xmlNode) bool {
	for _, child := range node.Children {
		switch child.Name {
		case "blur", "fillOverlay", "glow", "innerShdw", "outerShdw", "prstShdw", "reflection", "softEdge":
			return true
		}
	}
	return false
}

func collectTableLineUnsupportedFeatureMessages(line *xmlNode, messages map[string]bool) {
	if cap := attrValue(line.Attrs, "cap"); cap != "" && cap != "flat" && cap != "sq" && cap != "rnd" {
		messages["uses border line caps that were not rendered"] = true
	}
	if cmpd := attrValue(line.Attrs, "cmpd"); !isSupportedTableCompoundLine(cmpd) {
		messages["uses compound border lines that were not rendered"] = true
	}
	for _, name := range []string{"headEnd", "tailEnd"} {
		marker := firstChild(line, name)
		if marker == nil {
			continue
		}
		markerType := attrValue(marker.Attrs, "type")
		if markerType != "" && markerType != "none" {
			messages["uses border line end decorations that were not rendered"] = true
		}
	}
}

func parseTableCell(cellNode *xmlNode, theme themeColors) tableCell {
	cellElement := slideElement{}
	parseTextProperties(cellNode, &cellElement, theme)
	cell := tableCell{
		Text:           strings.TrimSpace(textFromNode(cellNode)),
		TextParagraphs: textParagraphsFromNodeWithTheme(cellNode, theme),
		ColSpan:        int(parseIntAttr(cellNode.Attrs, "gridSpan")),
		HMerge:         attrValue(cellNode.Attrs, "hMerge") == "1",
		RowSpan:        int(parseIntAttr(cellNode.Attrs, "rowSpan")),
		VMerge:         attrValue(cellNode.Attrs, "vMerge") == "1",
		FontSize:       cellElement.FontSize,
		HasTextColor:   cellElement.HasTextColor,
		TextColor:      cellElement.TextColor,
		TextAlign:      cellElement.TextAlign,
		TextAnchor:     cellElement.TextAnchor,
	}
	if cell.RowSpan <= 0 {
		cell.RowSpan = 1
	}
	if cell.ColSpan <= 0 {
		cell.ColSpan = 1
	}
	if cell.FontSize > 0 {
		cell.HasFontSize = true
	}
	if cell.FontSize == 0 {
		if size := textParagraphsFontSize(cell.TextParagraphs); size > 0 {
			cell.FontSize = size
			cell.HasFontSize = true
		}
	}
	if cell.FontSize == 0 {
		cell.FontSize = 1200
	}
	if cell.TextAlign == "" {
		cell.TextAlign = textParagraphsTextAlign(cell.TextParagraphs)
	}
	if textParagraphsHaveRunColor(cell.TextParagraphs) {
		cell.HasTextColor = false
		cell.TextColor = color.RGBA{}
	}
	if cellProperties := firstChild(cellNode, "tcPr"); cellProperties != nil {
		if anchor := attrValue(cellProperties.Attrs, "anchor"); anchor != "" {
			cell.TextAnchor = anchor
		}
		if margins, ok := parseTableCellMargins(cellProperties.Attrs); ok {
			cell.HasMargins = true
			cell.MarginLeft = margins.Left
			cell.MarginTop = margins.Top
			cell.MarginRight = margins.Right
			cell.MarginBottom = margins.Bottom
		}
		if firstChild(cellProperties, "noFill") != nil {
			cell.NoFill = true
		}
		if solidFill := firstChild(cellProperties, "solidFill"); solidFill != nil {
			if fill, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				cell.HasFill = true
				cell.FillColor = fill
			}
		}
		cell.BorderLeft = parseTableCellBorder(cellProperties, "lnL", theme)
		cell.BorderRight = parseTableCellBorder(cellProperties, "lnR", theme)
		cell.BorderTop = parseTableCellBorder(cellProperties, "lnT", theme)
		cell.BorderBottom = parseTableCellBorder(cellProperties, "lnB", theme)
	}
	return cell
}

func parseTableCellBorder(cellProperties *xmlNode, name string, theme themeColors) tableCellBorder {
	line := firstChild(cellProperties, name)
	if line == nil {
		return tableCellBorder{}
	}
	return parseTableLineNode(line, theme, true)
}

func parseTableLineNode(line *xmlNode, theme themeColors, specified bool) tableCellBorder {
	border := tableCellBorder{
		Specified: specified,
		Width:     parseIntAttr(line.Attrs, "w"),
		Cap:       attrValue(line.Attrs, "cap"),
		Align:     attrValue(line.Attrs, "algn"),
		Compound:  attrValue(line.Attrs, "cmpd"),
	}
	if border.Width == 0 {
		border.Width = 9525
	}
	if firstChild(line, "noFill") != nil {
		border.NoLine = true
		return border
	}
	if solidFill := firstChild(line, "solidFill"); solidFill != nil {
		if lineColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			border.HasLine = true
			border.Color = lineColor
		}
	}
	if dash := firstChild(line, "prstDash"); dash != nil {
		if value := attrValue(dash.Attrs, "val"); value != "" && value != "solid" {
			border.Dash = value
		}
	}
	if firstChild(line, "round") != nil {
		border.Join = "round"
	} else if firstChild(line, "bevel") != nil {
		border.Join = "bevel"
	} else if firstChild(line, "miter") != nil {
		border.Join = "miter"
	}
	return border
}

func packageTableStyles(pkg *pptx.Package, theme themeColors, fonts themeFonts, fillStyles themeFillStyles, lineStyles themeLineStyles, effectStyles themeEffectStyles) tableStyleSet {
	if data, ok := pkg.Parts["ppt/tableStyles.xml"]; ok {
		return parseTableStyles(data, theme, fonts, fillStyles, lineStyles, effectStyles)
	}
	return tableStyleSet{}
}

func parseTableStyles(data []byte, theme themeColors, fonts themeFonts, fillStyles themeFillStyles, lineStyles themeLineStyles, effectStyles themeEffectStyles) tableStyleSet {
	root, err := parseXMLNode(data)
	if err != nil {
		return tableStyleSet{}
	}
	styles := tableStyleSet{
		DefaultID: strings.TrimSpace(attrValue(root.Attrs, "def")),
		Styles:    map[string]tableStyle{},
	}
	for _, node := range childrenByName(root, "tblStyle") {
		style := tableStyle{
			ID:      strings.TrimSpace(attrValue(node.Attrs, "styleId")),
			Name:    attrValue(node.Attrs, "styleName"),
			Regions: map[string]tableStyleRegion{},
		}
		for _, child := range node.Children {
			if child.Name == "tblBg" {
				if background, ok := parseTableStyleBackgroundFill(child, theme, fillStyles); ok {
					style.HasBackground = true
					style.Background = background
				}
				if effects, ok := parseTableStyleBackgroundEffect(child, theme, effectStyles); ok {
					style.HasBackgroundEffect = true
					style.BackgroundEffect = effects
				}
				continue
			}
			if !isTableStyleRegionName(child.Name) {
				continue
			}
			style.Regions[child.Name] = parseTableStyleRegion(child, theme, fonts, lineStyles)
		}
		if style.ID != "" {
			styles.Styles[normalizedTableStyleID(style.ID)] = style
		}
	}
	return styles
}

func parseTableStyleBackgroundFill(node *xmlNode, theme themeColors, fillStyles themeFillStyles) (backgroundPaint, bool) {
	if fillRef := firstChild(node, "fillRef"); fillRef != nil && attrValue(fillRef.Attrs, "idx") != "0" {
		placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(fillRef, theme)
		if hasPlaceholderColor {
			if paint, ok := fillStyles.Style(parseIntAttr(fillRef.Attrs, "idx"), themeWithPlaceholderColor(theme, placeholderColor)); ok {
				return paint, true
			}
			return backgroundPaint{Color: placeholderColor}, true
		}
	}
	if solidFill := firstChild(node, "solidFill"); solidFill != nil {
		return backgroundPaintFromFillNode(solidFill, theme)
	}
	if gradFill := firstChild(node, "gradFill"); gradFill != nil {
		return backgroundPaintFromFillNode(gradFill, theme)
	}
	return backgroundPaint{}, false
}

func parseTableStyleBackgroundEffect(node *xmlNode, theme themeColors, effectStyles themeEffectStyles) (themeEffectStyle, bool) {
	if effectRef := firstChild(node, "effectRef"); effectRef != nil && attrValue(effectRef.Attrs, "idx") != "0" {
		styleTheme := theme
		if placeholderColor, ok := colorFromColorNodeWithTheme(effectRef, theme); ok {
			styleTheme = themeWithPlaceholderColor(theme, placeholderColor)
		}
		return effectStyles.Style(parseIntAttr(effectRef.Attrs, "idx"), styleTheme)
	}
	if effect := firstChild(node, "effect"); effect != nil {
		return parseThemeEffectStyle(effect, theme)
	}
	return themeEffectStyle{}, false
}

func isTableStyleRegionName(name string) bool {
	switch name {
	case "wholeTbl", "band1H", "band2H", "band1V", "band2V", "firstCol", "lastCol", "firstRow", "lastRow", "neCell", "nwCell", "seCell", "swCell":
		return true
	default:
		return false
	}
}

func parseTableStyleRegion(node *xmlNode, theme themeColors, fonts themeFonts, lineStyles themeLineStyles) tableStyleRegion {
	var region tableStyleRegion
	if textStyle := firstChild(node, "tcTxStyle"); textStyle != nil {
		if rawBold := attrValue(textStyle.Attrs, "b"); rawBold != "" {
			region.HasBold = true
			region.Bold = boolAttrOn(rawBold)
		}
		if rawItalic := attrValue(textStyle.Attrs, "i"); rawItalic != "" {
			region.HasItalic = true
			region.Italic = boolAttrOn(rawItalic)
		}
		if fontRef := firstChild(textStyle, "fontRef"); fontRef != nil {
			region.FontFamily = tableStyleFontFamily(fontRef, fonts)
		}
		if region.FontFamily == "" {
			region.FontFamily = tableStyleDirectFontFamily(textStyle)
		}
		if textColor, ok := colorFromColorNodeWithTheme(textStyle, theme); ok {
			region.HasTextColor = true
			region.TextColor = textColor
		}
	}
	if cellStyle := firstChild(node, "tcStyle"); cellStyle != nil {
		if fill := firstChild(cellStyle, "fill"); fill != nil {
			if firstChild(fill, "noFill") != nil {
				region.NoFill = true
			} else if solidFill := firstChild(fill, "solidFill"); solidFill != nil {
				if fillColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
					region.HasFill = true
					region.FillColor = fillColor
				}
			}
		}
		if fillRef := firstChild(cellStyle, "fillRef"); fillRef != nil {
			if fillColor, ok := colorFromColorNodeWithTheme(fillRef, theme); ok {
				region.HasFill = true
				region.FillColor = fillColor
			}
		}
		if borders := firstChild(cellStyle, "tcBdr"); borders != nil {
			region.Borders = parseTableStyleBorders(borders, theme, lineStyles)
		}
	}
	return region
}

func tableStyleFontFamily(fontRef *xmlNode, fonts themeFonts) string {
	switch attrValue(fontRef.Attrs, "idx") {
	case "major":
		return fonts.MajorLatin
	case "minor":
		return fonts.MinorLatin
	default:
		return ""
	}
}

func tableStyleDirectFontFamily(textStyle *xmlNode) string {
	font := firstChild(textStyle, "font")
	if font == nil {
		return ""
	}
	if typeface := typefaceFromChild(font, "latin"); typeface != "" {
		return typeface
	}
	if typeface := typefaceFromChild(font, "ea"); typeface != "" {
		return typeface
	}
	return typefaceFromChild(font, "cs")
}

func parseTableStyleBorders(node *xmlNode, theme themeColors, lineStyles themeLineStyles) tableStyleBorders {
	return tableStyleBorders{
		Left:    parseTableStyleBorder(node, "left", theme, lineStyles),
		Right:   parseTableStyleBorder(node, "right", theme, lineStyles),
		Top:     parseTableStyleBorder(node, "top", theme, lineStyles),
		Bottom:  parseTableStyleBorder(node, "bottom", theme, lineStyles),
		InsideH: parseTableStyleBorder(node, "insideH", theme, lineStyles),
		InsideV: parseTableStyleBorder(node, "insideV", theme, lineStyles),
	}
}

func parseTableStyleBorder(parent *xmlNode, name string, theme themeColors, lineStyles themeLineStyles) tableCellBorder {
	edge := firstChild(parent, name)
	if edge == nil {
		return tableCellBorder{}
	}
	line := firstChild(edge, "ln")
	if line == nil {
		if lineRef := firstChild(edge, "lnRef"); lineRef != nil {
			return parseTableStyleLineReference(lineRef, theme, lineStyles)
		}
		return tableCellBorder{Specified: true, NoLine: true}
	}
	return parseTableLineNode(line, theme, true)
}

func parseTableStyleLineReference(lineRef *xmlNode, theme themeColors, lineStyles themeLineStyles) tableCellBorder {
	if attrValue(lineRef.Attrs, "idx") == "0" {
		return tableCellBorder{Specified: true, NoLine: true}
	}
	placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(lineRef, theme)
	if hasPlaceholderColor {
		if border, ok := lineStyles.Style(parseIntAttr(lineRef.Attrs, "idx"), themeWithPlaceholderColor(theme, placeholderColor)); ok {
			return border
		}
		return tableCellBorder{Specified: true, HasLine: true, Color: placeholderColor, Width: 9525}
	}
	if border, ok := lineStyles.Style(parseIntAttr(lineRef.Attrs, "idx"), theme); ok {
		return border
	}
	return tableCellBorder{Specified: true, NoLine: true}
}

func boolAttrOn(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

func normalizedTableStyleID(styleID string) string {
	return strings.ToLower(strings.TrimSpace(styleID))
}

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

type textInsets struct {
	Left   int64
	Top    int64
	Right  int64
	Bottom int64
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

func colorFromSolidFill(node *xmlNode) (color.RGBA, bool) {
	return colorFromSolidFillWithTheme(node, defaultThemeColors())
}

func colorFromSolidFillWithTheme(node *xmlNode, theme themeColors) (color.RGBA, bool) {
	return colorFromColorNodeWithTheme(node, theme)
}

func colorFromColorNode(node *xmlNode) (color.RGBA, bool) {
	return colorFromColorNodeWithTheme(node, defaultThemeColors())
}

func colorFromColorNodeWithTheme(node *xmlNode, theme themeColors) (color.RGBA, bool) {
	if srgb := firstChild(node, "srgbClr"); srgb != nil {
		if c, ok := parseHexColor(attrValue(srgb.Attrs, "val")); ok {
			return applyColorModifiers(c, srgb), true
		}
	}
	if scrgb := firstChild(node, "scrgbClr"); scrgb != nil {
		if c, ok := parseScRGBColor(scrgb); ok {
			return applyColorModifiers(c, scrgb), true
		}
	}
	if scheme := firstChild(node, "schemeClr"); scheme != nil {
		if c, ok := schemeColorWithTheme(attrValue(scheme.Attrs, "val"), theme); ok {
			return applyColorModifiers(c, scheme), true
		}
	}
	if sys := firstChild(node, "sysClr"); sys != nil {
		if c, ok := parseHexColor(attrValue(sys.Attrs, "lastClr")); ok {
			return applyColorModifiers(c, sys), true
		}
	}
	if preset := firstChild(node, "prstClr"); preset != nil {
		if c, ok := presetColor(attrValue(preset.Attrs, "val")); ok {
			return applyColorModifiers(c, preset), true
		}
	}
	return color.RGBA{}, false
}

func presetColor(value string) (color.RGBA, bool) {
	switch value {
	case "black":
		return color.RGBA{A: 255}, true
	case "red":
		return color.RGBA{R: 255, A: 255}, true
	case "white":
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	default:
		return color.RGBA{}, false
	}
}

func parseScRGBColor(node *xmlNode) (color.RGBA, bool) {
	r, okR := parseScRGBLinearAttr(node, "r")
	g, okG := parseScRGBLinearAttr(node, "g")
	b, okB := parseScRGBLinearAttr(node, "b")
	if !okR || !okG || !okB {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: linearToSRGBByte(r),
		G: linearToSRGBByte(g),
		B: linearToSRGBByte(b),
		A: 255,
	}, true
}

func parseScRGBLinearAttr(node *xmlNode, name string) (float64, bool) {
	raw := attrValue(node.Attrs, name)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return clampFloat(float64(value)/100000, 0, 1), true
}

func applyColorModifiers(c color.RGBA, node *xmlNode) color.RGBA {
	pendingLumMod := int64(100000)
	pendingLumOff := int64(0)
	hasPendingLuminance := false
	flushLuminance := func() {
		if !hasPendingLuminance {
			return
		}
		c = applyLuminanceModifier(c, pendingLumMod, pendingLumOff)
		pendingLumMod = 100000
		pendingLumOff = 0
		hasPendingLuminance = false
	}
	for _, child := range node.Children {
		switch child.Name {
		case "lumMod":
			if hasPendingLuminance && pendingLumOff != 0 {
				flushLuminance()
			}
			pendingLumMod = pendingLumMod * parsePercentAttr(child.Attrs, "val") / 100000
			hasPendingLuminance = true
		case "shade":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyShadeModifier(c, value)
		case "lumOff":
			pendingLumOff += parsePercentAttr(child.Attrs, "val")
			hasPendingLuminance = true
		case "alpha":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = scaleColorChannel(c.A, value)
		case "alphaOff":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c.A = offsetColorChannel(c.A, value)
		case "hueOff":
			flushLuminance()
			value := parseIntAttr(child.Attrs, "val")
			c = applyHueOffset(c, value)
		case "tint":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applyTintModifier(c, value)
		case "satMod":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applySaturationModifier(c, value)
		case "satOff":
			flushLuminance()
			value := parsePercentAttr(child.Attrs, "val")
			c = applySaturationOffset(c, value)
		}
	}
	flushLuminance()
	return c
}

func applyLuminanceModifier(c color.RGBA, mod int64, off int64) color.RGBA {
	if mod == 0 && off == 0 {
		c.R = 0
		c.G = 0
		c.B = 0
		return c
	}
	h, s, l := rgbToHSL(c)
	l = l*float64(mod)/100000 + float64(off)/100000
	if l < 0 {
		l = 0
	} else if l > 1 {
		l = 1
	}
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func scaleColorChannel(channel uint8, value int64) uint8 {
	return clampColor(int64(channel) * value / 100000)
}

func offsetColorChannel(channel uint8, value int64) uint8 {
	return clampColor(int64(math.Round(float64(channel) + float64(value)*255/100000)))
}

func applyTintModifier(c color.RGBA, value int64) color.RGBA {
	c.R = blendSRGBChannelLinear(c.R, 255, value)
	c.G = blendSRGBChannelLinear(c.G, 255, value)
	c.B = blendSRGBChannelLinear(c.B, 255, value)
	return c
}

func applyShadeModifier(c color.RGBA, value int64) color.RGBA {
	c.R = blendSRGBChannelLinear(c.R, 0, value)
	c.G = blendSRGBChannelLinear(c.G, 0, value)
	c.B = blendSRGBChannelLinear(c.B, 0, value)
	return c
}

func blendSRGBChannelLinear(channel uint8, target uint8, value int64) uint8 {
	if value < 0 {
		value = 0
	} else if value > 100000 {
		value = 100000
	}
	t := float64(value) / 100000
	linear := srgbByteToLinear(channel)*t + srgbByteToLinear(target)*(1-t)
	return linearToSRGBByte(linear)
}

func applySaturationModifier(c color.RGBA, value int64) color.RGBA {
	if value == 100000 {
		return c
	}
	h, s, l := rgbToHSL(c)
	s *= float64(value) / 100000
	if s < 0 {
		s = 0
	} else if s > 1 {
		s = 1
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func applySaturationOffset(c color.RGBA, value int64) color.RGBA {
	if value == 0 {
		return c
	}
	h, s, l := rgbToHSL(c)
	s += float64(value) / 100000
	if s < 0 {
		s = 0
	} else if s > 1 {
		s = 1
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func applyHueOffset(c color.RGBA, value int64) color.RGBA {
	if value == 0 {
		return c
	}
	h, s, l := rgbToHSL(c)
	h = math.Mod(h+float64(value)/60000, 360)
	if h < 0 {
		h += 360
	}
	r, g, b := hslToRGB(h, s, l)
	c.R = r
	c.G = g
	c.B = b
	return c
}

func rgbToHSL(c color.RGBA) (float64, float64, float64) {
	r := float64(c.R) / 255
	g := float64(c.G) / 255
	b := float64(c.B) / 255
	maxChannel := math.Max(r, math.Max(g, b))
	minChannel := math.Min(r, math.Min(g, b))
	l := (maxChannel + minChannel) / 2
	if maxChannel == minChannel {
		return 0, 0, l
	}
	delta := maxChannel - minChannel
	s := delta / (1 - math.Abs(2*l-1))
	var h float64
	switch maxChannel {
	case r:
		h = math.Mod((g-b)/delta, 6)
	case g:
		h = (b-r)/delta + 2
	default:
		h = (r-g)/delta + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

func hslToRGB(h float64, s float64, l float64) (uint8, uint8, uint8) {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r1, g1, b1 float64
	switch {
	case h < 60:
		r1, g1, b1 = c, x, 0
	case h < 120:
		r1, g1, b1 = x, c, 0
	case h < 180:
		r1, g1, b1 = 0, c, x
	case h < 240:
		r1, g1, b1 = 0, x, c
	case h < 300:
		r1, g1, b1 = x, 0, c
	default:
		r1, g1, b1 = c, 0, x
	}
	return roundUnitColorChannel(r1 + m),
		roundUnitColorChannel(g1 + m),
		roundUnitColorChannel(b1 + m)
}

func roundUnitColorChannel(value float64) uint8 {
	return clampColor(int64(math.Floor(value*255 + 0.5 + 1e-9)))
}

func clampColor(value int64) uint8 {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return uint8(value)
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

func textFromNode(node *xmlNode) string {
	if node.Name == "br" {
		return "\n"
	}
	if node.Name == "tab" {
		return "\t"
	}
	var output strings.Builder
	if node.Name == "t" || node.Name == "fld" {
		output.WriteString(node.Text)
	}
	for _, child := range node.Children {
		output.WriteString(textFromNode(child))
	}
	if node.Name == "p" && output.Len() > 0 {
		output.WriteByte('\n')
	}
	return output.String()
}

func textParagraphsFromNode(node *xmlNode) []textParagraph {
	return textParagraphsFromNodeWithTheme(node, defaultThemeColors())
}

func textParagraphsFromNodeWithTheme(node *xmlNode, theme themeColors) []textParagraph {
	var output []textParagraph
	styles := paragraphStylesFromListStyle(firstDescendant(node, "lstStyle"), theme)
	autoCounters := map[int]int{}
	for _, paragraphNode := range descendantsByName(node, "p") {
		text := strings.TrimSpace(textFromNode(paragraphNode))
		paragraph := textParagraph{Text: text}
		paragraph.Runs = paragraphTextRunsWithTheme(paragraphNode, theme)
		if pPr := firstChild(paragraphNode, "pPr"); pPr != nil {
			paragraph.Level = int(parseIntAttr(pPr.Attrs, "lvl"))
			paragraph.TextAlign = attrValue(pPr.Attrs, "algn")
		}
		style := styles[paragraph.Level]
		applyParagraphStyle(&paragraph, style)
		hasLocalBulletChoice := false
		localAutoNumberApplied := false
		if pPr := firstChild(paragraphNode, "pPr"); pPr != nil {
			if value := attrValue(pPr.Attrs, "marL"); value != "" {
				paragraph.MarginLeft = parseIntAttr(pPr.Attrs, "marL")
				paragraph.HasMarginLeft = true
			}
			if value := attrValue(pPr.Attrs, "marR"); value != "" {
				paragraph.MarginRight = parseIntAttr(pPr.Attrs, "marR")
				paragraph.HasMarginRight = true
			}
			if value := attrValue(pPr.Attrs, "indent"); value != "" {
				paragraph.Indent = parseIntAttr(pPr.Attrs, "indent")
				paragraph.HasIndent = true
			}
			if value := attrValue(pPr.Attrs, "defTabSz"); value != "" {
				paragraph.DefaultTabSize = parseIntAttr(pPr.Attrs, "defTabSz")
				paragraph.HasDefaultTab = paragraph.DefaultTabSize > 0
			}
			if spcBef := firstChild(pPr, "spcBef"); spcBef != nil {
				paragraph.HasSpaceBefore = true
				paragraph.SpaceBefore, paragraph.SpaceBeforePct = parseSpacingValue(spcBef)
			}
			if spcAft := firstChild(pPr, "spcAft"); spcAft != nil {
				paragraph.HasSpaceAfter = true
				paragraph.SpaceAfter, paragraph.SpaceAfterPct = parseSpacingValue(spcAft)
			}
			if lnSpc := firstChild(pPr, "lnSpc"); lnSpc != nil {
				paragraph.HasLineSpacing = true
				paragraph.LineSpacingPct = parseSpacingPercent(lnSpc)
			}
			paragraph.TabStops = parseParagraphTabStops(pPr)
			if bulletColorNode := firstChild(pPr, "buClr"); bulletColorNode != nil {
				if bulletColor, ok := colorFromColorNodeWithTheme(bulletColorNode, theme); ok {
					paragraph.HasBulletColor = true
					paragraph.BulletColor = bulletColor
				}
			}
			if firstChild(pPr, "buClrTx") != nil {
				paragraph.BulletColorTx = true
				paragraph.HasBulletColor = false
				paragraph.BulletColor = color.RGBA{}
			}
			paragraph.BulletFontFamily = bulletFontFamilyFromProperties(pPr)
			if firstChild(pPr, "buFontTx") != nil {
				paragraph.BulletFontTx = true
				paragraph.BulletFontFamily = ""
			}
			if bullet := firstChild(pPr, "buChar"); bullet != nil {
				hasLocalBulletChoice = true
				paragraph.Bullet = normalizeBulletCharForFont(attrValue(bullet.Attrs, "char"), paragraph.BulletFontFamily)
				paragraph.NoBullet = false
			}
			if autoNum := firstChild(pPr, "buAutoNum"); autoNum != nil {
				hasLocalBulletChoice = true
				localAutoNumberApplied = true
				paragraph.HasAutoNumber = true
				if startAt := int(parseIntAttr(autoNum.Attrs, "startAt")); startAt > 0 {
					autoCounters[paragraph.Level] = startAt
				} else {
					autoCounters[paragraph.Level]++
				}
				for level := paragraph.Level + 1; level < 9; level++ {
					delete(autoCounters, level)
				}
				paragraph.Bullet = autoNumberBullet(attrValue(autoNum.Attrs, "type"), autoCounters[paragraph.Level])
				paragraph.NoBullet = false
			}
			if firstChild(pPr, "buNone") != nil {
				hasLocalBulletChoice = true
				paragraph.NoBullet = true
				paragraph.Bullet = ""
			} else if paragraph.Bullet == "" && paragraph.Level > 0 && !style.HasAutoNumber {
				paragraph.Bullet = "•"
			}
			applyBulletSizePropertiesToParagraph(&paragraph, pPr)
			if paragraph.NoBullet {
				if bulletSize := firstChild(pPr, "buSzPts"); bulletSize != nil {
					if size := parseIntAttr(bulletSize.Attrs, "val"); size > 0 {
						paragraph.FontSize = int(size)
					}
				}
			}
			if defRPr := firstChild(pPr, "defRPr"); defRPr != nil {
				applyRunPropertiesToParagraphDefaults(&paragraph, defRPr, theme)
			}
		}
		if !localAutoNumberApplied && !hasLocalBulletChoice && !paragraph.NoBullet && paragraph.Bullet == "" && style.HasAutoNumber {
			if style.AutoNumberStart > 0 && autoCounters[paragraph.Level] == 0 {
				autoCounters[paragraph.Level] = style.AutoNumberStart
			} else {
				autoCounters[paragraph.Level]++
			}
			for level := paragraph.Level + 1; level < 9; level++ {
				delete(autoCounters, level)
			}
			paragraph.Bullet = autoNumberBullet(style.AutoNumberType, autoCounters[paragraph.Level])
			paragraph.HasAutoNumber = true
		}
		if endParaRPr := firstChild(paragraphNode, "endParaRPr"); endParaRPr != nil && !textRunsHaveRunMetricProperties(paragraph.Runs) {
			applyRunPropertiesToParagraphDefaults(&paragraph, endParaRPr, theme)
		}
		if size := textRunsFontSize(paragraph.Runs); size > 0 {
			paragraph.FontSize = size
		}
		if len(paragraph.Runs) > 0 {
			if textRunsAllBold(paragraph.Runs) {
				paragraph.Bold = true
			}
			if textRunsAllItalic(paragraph.Runs) {
				paragraph.Italic = true
			}
		}
		if paragraph.Text == "" && len(paragraph.Runs) == 0 {
			paragraph.NoBullet = true
			paragraph.Bullet = ""
		}
		output = append(output, paragraph)
	}
	return output
}

func parseParagraphTabStops(pPr *xmlNode) []int64 {
	tabList := firstChild(pPr, "tabLst")
	if tabList == nil {
		return nil
	}
	var stops []int64
	for _, tab := range childrenByName(tabList, "tab") {
		pos := parseIntAttr(tab.Attrs, "pos")
		if pos <= 0 {
			continue
		}
		stops = append(stops, pos)
	}
	if len(stops) == 0 {
		return nil
	}
	sort.Slice(stops, func(i, j int) bool { return stops[i] < stops[j] })
	return stops
}

func autoNumberBullet(kind string, index int) string {
	if index < 1 {
		index = 1
	}
	switch kind {
	case "alphaLcPeriod":
		return alphaNumber(index, false) + "."
	case "alphaUcPeriod":
		return alphaNumber(index, true) + "."
	case "alphaLcParenR":
		return alphaNumber(index, false) + ")"
	case "alphaUcParenR":
		return alphaNumber(index, true) + ")"
	case "arabicParenR":
		return fmt.Sprintf("%d)", index)
	default:
		return fmt.Sprintf("%d.", index)
	}
}

func alphaNumber(index int, upper bool) string {
	var chars []byte
	for index > 0 {
		index--
		chars = append([]byte{byte('a' + index%26)}, chars...)
		index /= 26
	}
	if upper {
		for idx := range chars {
			chars[idx] = byte(unicode.ToUpper(rune(chars[idx])))
		}
	}
	return string(chars)
}

func applyParagraphStyle(paragraph *textParagraph, style paragraphStyle) {
	if style.HasMarginLeft && !paragraph.HasMarginLeft {
		paragraph.HasMarginLeft = true
		paragraph.MarginLeft = style.MarginLeft
	}
	if style.HasMarginRight && !paragraph.HasMarginRight {
		paragraph.HasMarginRight = true
		paragraph.MarginRight = style.MarginRight
	}
	if style.HasIndent && !paragraph.HasIndent {
		paragraph.HasIndent = true
		paragraph.Indent = style.Indent
	}
	if style.HasDefaultTab && !paragraph.HasDefaultTab {
		paragraph.HasDefaultTab = true
		paragraph.DefaultTabSize = style.DefaultTabSize
	}
	if !paragraph.HasSpaceBefore && style.HasSpaceBefore {
		paragraph.HasSpaceBefore = true
		paragraph.SpaceBefore = style.SpaceBefore
		paragraph.SpaceBeforePct = style.SpaceBeforePct
	}
	if !paragraph.HasSpaceAfter && style.HasSpaceAfter {
		paragraph.HasSpaceAfter = true
		paragraph.SpaceAfter = style.SpaceAfter
		paragraph.SpaceAfterPct = style.SpaceAfterPct
	}
	if !paragraph.HasLineSpacing && style.HasLineSpacing {
		paragraph.HasLineSpacing = true
		paragraph.LineSpacingPct = style.LineSpacingPct
	}
	if paragraph.BulletSizeTx {
		paragraph.BulletFontSize = 0
		paragraph.BulletSizePct = 0
	} else if paragraph.BulletFontSize == 0 {
		paragraph.BulletFontSize = style.BulletFontSize
	}
	if paragraph.BulletSizeTx {
		// Local buSzTx blocks inherited fixed or percentage bullet sizing.
	} else if paragraph.BulletSizePct == 0 {
		paragraph.BulletSizePct = style.BulletSizePct
	}
	if style.BulletSizeTx && paragraph.BulletFontSize == 0 && paragraph.BulletSizePct == 0 {
		paragraph.BulletSizeTx = true
	}
	if paragraph.FontFamily == "" {
		paragraph.FontFamily = concreteParagraphFontFamily(style.FontFamily)
	}
	if paragraph.FontSize == 0 {
		paragraph.FontSize = style.FontSize
	}
	if !paragraph.Bold {
		paragraph.Bold = style.Bold
	}
	if !paragraph.Italic {
		paragraph.Italic = style.Italic
	}
	if !paragraph.HasCharSpacing && style.HasCharSpacing {
		paragraph.HasCharSpacing = true
		paragraph.CharSpacing = style.CharSpacing
	}
	if paragraph.TextAlign == "" {
		paragraph.TextAlign = style.TextAlign
	}
	if !paragraph.HasTextColor && style.HasTextColor {
		paragraph.HasTextColor = true
		paragraph.TextColor = style.TextColor
	}
	if paragraph.NoBullet {
		paragraph.Bullet = ""
		return
	}
	if style.NoBullet {
		paragraph.NoBullet = true
		paragraph.Bullet = ""
	} else if style.Bullet != "" && (paragraph.Bullet == "" || paragraph.Bullet == "•") {
		paragraph.Bullet = style.Bullet
	}
	if paragraph.BulletFontFamily == "" && !paragraph.BulletFontTx {
		paragraph.BulletFontFamily = style.BulletFontFamily
	}
	if style.BulletFontTx && paragraph.BulletFontFamily == "" && !paragraph.BulletFontTx {
		paragraph.BulletFontTx = true
		paragraph.BulletFontFamily = ""
	}
	if style.HasBulletColor && !paragraph.HasBulletColor && !paragraph.BulletColorTx {
		paragraph.HasBulletColor = true
		paragraph.BulletColor = style.BulletColor
	}
	if style.BulletColorTx && !paragraph.HasBulletColor && !paragraph.BulletColorTx {
		paragraph.BulletColorTx = true
		paragraph.HasBulletColor = false
		paragraph.BulletColor = color.RGBA{}
	}
}

func paragraphStylesFromListStyle(listStyle *xmlNode, theme themeColors) map[int]paragraphStyle {
	styles := map[int]paragraphStyle{}
	if listStyle == nil {
		return styles
	}
	for _, child := range listStyle.Children {
		level, ok := paragraphLevelFromStyleNode(child.Name)
		if !ok {
			continue
		}
		styles[level] = parseParagraphStyle(child, theme)
	}
	return styles
}

func paragraphLevelFromStyleNode(name string) (int, bool) {
	if name == "defPPr" {
		return 0, true
	}
	if !strings.HasPrefix(name, "lvl") || !strings.HasSuffix(name, "pPr") {
		return 0, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(name, "lvl"), "pPr")
	level, err := strconv.Atoi(raw)
	if err != nil || level < 1 {
		return 0, false
	}
	return level - 1, true
}

func parseParagraphStyle(node *xmlNode, theme themeColors) paragraphStyle {
	var style paragraphStyle
	style.TextAlign = attrValue(node.Attrs, "algn")
	if value := attrValue(node.Attrs, "marL"); value != "" {
		style.HasMarginLeft = true
		style.MarginLeft = parseIntAttr(node.Attrs, "marL")
	}
	if value := attrValue(node.Attrs, "marR"); value != "" {
		style.HasMarginRight = true
		style.MarginRight = parseIntAttr(node.Attrs, "marR")
	}
	if value := attrValue(node.Attrs, "indent"); value != "" {
		style.HasIndent = true
		style.Indent = parseIntAttr(node.Attrs, "indent")
	}
	if value := attrValue(node.Attrs, "defTabSz"); value != "" {
		style.DefaultTabSize = parseIntAttr(node.Attrs, "defTabSz")
		style.HasDefaultTab = style.DefaultTabSize > 0
	}
	style.BulletFontFamily = bulletFontFamilyFromProperties(node)
	if bullet := firstChild(node, "buChar"); bullet != nil {
		style.Bullet = normalizeBulletCharForFont(attrValue(bullet.Attrs, "char"), style.BulletFontFamily)
	}
	if bulletColorNode := firstChild(node, "buClr"); bulletColorNode != nil {
		if bulletColor, ok := colorFromColorNodeWithTheme(bulletColorNode, theme); ok {
			style.HasBulletColor = true
			style.BulletColor = bulletColor
		}
	}
	if firstChild(node, "buClrTx") != nil {
		style.BulletColorTx = true
		style.HasBulletColor = false
		style.BulletColor = color.RGBA{}
	}
	if firstChild(node, "buFontTx") != nil {
		style.BulletFontTx = true
		style.BulletFontFamily = ""
	}
	if autoNum := firstChild(node, "buAutoNum"); autoNum != nil {
		style.HasAutoNumber = true
		style.Bullet = ""
		style.NoBullet = false
		style.AutoNumberType = attrValue(autoNum.Attrs, "type")
		if startAt := int(parseIntAttr(autoNum.Attrs, "startAt")); startAt > 0 {
			style.AutoNumberStart = startAt
		}
	}
	if firstChild(node, "buNone") != nil {
		style.NoBullet = true
		style.Bullet = ""
	}
	if spcBef := firstChild(node, "spcBef"); spcBef != nil {
		style.HasSpaceBefore = true
		style.SpaceBefore, style.SpaceBeforePct = parseSpacingValue(spcBef)
	}
	if spcAft := firstChild(node, "spcAft"); spcAft != nil {
		style.HasSpaceAfter = true
		style.SpaceAfter, style.SpaceAfterPct = parseSpacingValue(spcAft)
	}
	if lnSpc := firstChild(node, "lnSpc"); lnSpc != nil {
		style.HasLineSpacing = true
		style.LineSpacingPct = parseSpacingPercent(lnSpc)
	}
	if defRPr := firstChild(node, "defRPr"); defRPr != nil {
		style.FontFamily = concreteParagraphFontFamily(latinTypefaceFromRunProperties(defRPr))
		if size := parseIntAttr(defRPr.Attrs, "sz"); size > 0 {
			style.FontSize = int(size)
		}
		style.Bold = attrValue(defRPr.Attrs, "b") == "1"
		style.Italic = attrValue(defRPr.Attrs, "i") == "1"
		if value := attrValue(defRPr.Attrs, "spc"); value != "" {
			style.HasCharSpacing = true
			style.CharSpacing = int(parseIntAttr(defRPr.Attrs, "spc"))
		}
		if solidFill := firstChild(defRPr, "solidFill"); solidFill != nil {
			if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				style.HasTextColor = true
				style.TextColor = textColor
			}
		}
	}
	applyBulletSizePropertiesToParagraphStyle(&style, node)
	return style
}

func applyBulletSizePropertiesToParagraphStyle(style *paragraphStyle, node *xmlNode) {
	if firstChild(node, "buSzTx") != nil {
		style.BulletSizeTx = true
		style.BulletFontSize = 0
		style.BulletSizePct = 0
		return
	}
	if bulletSize := firstChild(node, "buSzPts"); bulletSize != nil {
		if size := int(parseIntAttr(bulletSize.Attrs, "val")); size > 0 {
			style.BulletSizeTx = false
			style.BulletFontSize = size
			style.BulletSizePct = 0
		}
	}
	if bulletSize := firstChild(node, "buSzPct"); bulletSize != nil && style.BulletFontSize == 0 {
		if pct := int(parseIntAttr(bulletSize.Attrs, "val")); pct > 0 {
			style.BulletSizeTx = false
			style.BulletSizePct = pct
		}
	}
}

func applyBulletSizePropertiesToParagraph(paragraph *textParagraph, node *xmlNode) {
	if firstChild(node, "buSzTx") != nil {
		paragraph.BulletSizeTx = true
		paragraph.BulletFontSize = 0
		paragraph.BulletSizePct = 0
		return
	}
	if bulletSize := firstChild(node, "buSzPts"); bulletSize != nil {
		if size := int(parseIntAttr(bulletSize.Attrs, "val")); size > 0 {
			paragraph.BulletSizeTx = false
			paragraph.BulletFontSize = size
			paragraph.BulletSizePct = 0
		}
	}
	if bulletSize := firstChild(node, "buSzPct"); bulletSize != nil && paragraph.BulletFontSize == 0 {
		if pct := int(parseIntAttr(bulletSize.Attrs, "val")); pct > 0 {
			paragraph.BulletSizeTx = false
			paragraph.BulletSizePct = pct
		}
	}
}

func normalizeBulletChar(raw string) string {
	return normalizeBulletCharForFont(raw, "")
}

func normalizeBulletCharForFont(raw string, fontFamily string) string {
	font := strings.ToLower(strings.TrimSpace(fontFamily))
	if strings.Contains(font, "wingdings") && exactFontFamilyAvailable(fontFamily) {
		if mapped := legacySymbolBulletPrivateUseChar(raw); mapped != "" {
			return mapped
		}
		return raw
	}
	switch raw {
	case "§":
		return "▪"
	case "Ø":
		if strings.Contains(font, "wingdings") {
			return "¬"
		}
		return raw
	case "\uf075":
		return "▶"
	default:
		return raw
	}
}

func legacySymbolBulletPrivateUseChar(raw string) string {
	runes := []rune(raw)
	if len(runes) != 1 || runes[0] > 0xff {
		return ""
	}
	return string(rune(0xf000) + runes[0])
}

func bulletFontFamilyFromProperties(node *xmlNode) string {
	if bulletFont := firstChild(node, "buFont"); bulletFont != nil {
		return attrValue(bulletFont.Attrs, "typeface")
	}
	return ""
}

func parseSpacingPixels(node *xmlNode) int {
	pixels, _ := parseSpacingValue(node)
	return pixels
}

func parseSpacingValue(node *xmlNode) (int, int) {
	if spcPts := firstChild(node, "spcPts"); spcPts != nil {
		points100 := parseIntAttr(spcPts.Attrs, "val")
		if points100 <= 0 {
			return 0, 0
		}
		return int(math.Round(float64(points100) / 100 * defaultOutputDPI / 72)), 0
	}
	if spcPct := firstChild(node, "spcPct"); spcPct != nil {
		pct := int(parsePercentAttr(spcPct.Attrs, "val"))
		if pct <= 0 {
			return 0, 0
		}
		return 0, pct
	}
	return 0, 0
}

func parseSpacingPercent(node *xmlNode) int {
	if spcPct := firstChild(node, "spcPct"); spcPct != nil {
		value := int(parsePercentAttr(spcPct.Attrs, "val"))
		if value > 0 {
			return value
		}
	}
	return 0
}

func paragraphTextRuns(paragraphNode *xmlNode) []textRun {
	return paragraphTextRunsWithTheme(paragraphNode, defaultThemeColors())
}

func paragraphTextRunsWithTheme(paragraphNode *xmlNode, theme themeColors) []textRun {
	var runs []textRun
	for _, child := range paragraphNode.Children {
		switch child.Name {
		case "r", "fld":
			text := textFromNode(child)
			if text == "" {
				continue
			}
			runs = append(runs, textRunFromNodeWithTheme(child, text, theme))
		case "br":
			runs = append(runs, textRunFromNodeWithTheme(child, "\n", theme))
		}
	}
	return trimTextRuns(runs)
}

func textRunFromNode(node *xmlNode, text string) textRun {
	return textRunFromNodeWithTheme(node, text, defaultThemeColors())
}

func textRunFromNodeWithTheme(node *xmlNode, text string, theme themeColors) textRun {
	run := textRun{Text: text}
	if node.Name == "fld" {
		run.FieldType = attrValue(node.Attrs, "type")
	}
	if rPr := firstChild(node, "rPr"); rPr != nil {
		applyRunPropertiesToRun(&run, rPr, text, theme)
	}
	return run
}

func resolveTextFields(elements []slideElement, slideNumber int) []slideElement {
	if slideNumber <= 0 {
		return elements
	}
	for index := range elements {
		if textParagraphsContainFields(elements[index].TextParagraphs) {
			elements[index].TextParagraphs = resolveTextParagraphFields(elements[index].TextParagraphs, slideNumber)
			elements[index].Text = textFromParagraphs(elements[index].TextParagraphs)
		}
		if elements[index].HasTable {
			for rowIndex := range elements[index].Table.Rows {
				for cellIndex := range elements[index].Table.Rows[rowIndex].Cells {
					cell := &elements[index].Table.Rows[rowIndex].Cells[cellIndex]
					if textParagraphsContainFields(cell.TextParagraphs) {
						cell.TextParagraphs = resolveTextParagraphFields(cell.TextParagraphs, slideNumber)
						cell.Text = textFromParagraphs(cell.TextParagraphs)
					}
				}
			}
		}
	}
	return elements
}

func textParagraphsContainFields(paragraphs []textParagraph) bool {
	for _, paragraph := range paragraphs {
		for _, run := range paragraph.Runs {
			if run.FieldType != "" {
				return true
			}
		}
	}
	return false
}

func resolveTextParagraphFields(paragraphs []textParagraph, slideNumber int) []textParagraph {
	for paragraphIndex := range paragraphs {
		for runIndex := range paragraphs[paragraphIndex].Runs {
			run := &paragraphs[paragraphIndex].Runs[runIndex]
			if strings.EqualFold(run.FieldType, "slidenum") {
				run.Text = strconv.Itoa(slideNumber)
			}
		}
		paragraphs[paragraphIndex].Text = textFromRuns(paragraphs[paragraphIndex].Runs)
	}
	return paragraphs
}

func textFromParagraphs(paragraphs []textParagraph) string {
	var parts []string
	for _, paragraph := range paragraphs {
		text := strings.TrimSpace(paragraph.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

func textFromRuns(runs []textRun) string {
	var builder strings.Builder
	for _, run := range runs {
		builder.WriteString(run.Text)
	}
	return strings.TrimSpace(builder.String())
}

func applyRunPropertiesToParagraph(paragraph *textParagraph, rPr *xmlNode, theme themeColors) {
	var run textRun
	applyRunPropertiesToRun(&run, rPr, "", theme)
	paragraph.FontFamily = concreteParagraphFontFamily(run.FontFamily)
	paragraph.FontSize = run.FontSize
	paragraph.Bold = run.Bold
	paragraph.Italic = run.Italic
	paragraph.HasCharSpacing = run.HasCharSpacing
	paragraph.CharSpacing = run.CharSpacing
	paragraph.HasTextColor = run.HasTextColor
	paragraph.TextColor = run.TextColor
}

func applyRunPropertiesToParagraphDefaults(paragraph *textParagraph, rPr *xmlNode, theme themeColors) {
	var run textRun
	applyRunPropertiesToRun(&run, rPr, "", theme)
	if paragraph.FontFamily == "" {
		paragraph.FontFamily = concreteParagraphFontFamily(run.FontFamily)
	}
	if paragraph.FontSize == 0 {
		paragraph.FontSize = run.FontSize
	}
	if !paragraph.Bold {
		paragraph.Bold = run.Bold
	}
	if !paragraph.Italic {
		paragraph.Italic = run.Italic
	}
	if !paragraph.HasCharSpacing && run.HasCharSpacing {
		paragraph.HasCharSpacing = true
		paragraph.CharSpacing = run.CharSpacing
	}
	if !paragraph.HasTextColor && run.HasTextColor {
		paragraph.HasTextColor = true
		paragraph.TextColor = run.TextColor
	}
}

func concreteParagraphFontFamily(fontFamily string) string {
	trimmed := strings.TrimSpace(fontFamily)
	return trimmed
}

func applyRunPropertiesToRun(run *textRun, rPr *xmlNode, text string, theme themeColors) {
	run.FontSize = int(parseIntAttr(rPr.Attrs, "sz"))
	run.FontFamily = typefaceFromRunPropertiesForText(rPr, text)
	if value := attrValue(rPr.Attrs, "b"); value != "" {
		run.HasBold = true
		run.Bold = boolAttrOn(value)
	}
	if value := attrValue(rPr.Attrs, "i"); value != "" {
		run.HasItalic = true
		run.Italic = boolAttrOn(value)
	}
	run.Underline = runPropertiesUnderline(rPr)
	run.Strike = textStrikeType(attrValue(rPr.Attrs, "strike"))
	run.Baseline = int(parseIntAttr(rPr.Attrs, "baseline"))
	if value := attrValue(rPr.Attrs, "kern"); value != "" {
		run.HasKern = true
		run.KernMinFontSize = int(parseIntAttr(rPr.Attrs, "kern"))
	}
	if value := attrValue(rPr.Attrs, "spc"); value != "" {
		run.HasCharSpacing = true
		run.CharSpacing = int(parseIntAttr(rPr.Attrs, "spc"))
	}
	if solidFill := firstChild(rPr, "solidFill"); solidFill != nil {
		if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			run.HasTextColor = true
			run.TextColor = textColor
		}
	}
	if highlight := firstChild(rPr, "highlight"); highlight != nil {
		if highlightColor, ok := colorFromColorNodeWithTheme(highlight, theme); ok {
			run.HasHighlightColor = true
			run.HighlightColor = highlightColor
		}
	}
	if underlineFill := firstChild(rPr, "uFill"); underlineFill != nil {
		if solidFill := firstChild(underlineFill, "solidFill"); solidFill != nil {
			if underlineColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				run.HasUnderlineColor = true
				run.UnderlineColor = underlineColor
			}
		} else if underlineColor, ok := colorFromColorNodeWithTheme(underlineFill, theme); ok {
			run.HasUnderlineColor = true
			run.UnderlineColor = underlineColor
		}
	}
}

func typefaceFromRunPropertiesForText(rPr *xmlNode, text string) string {
	if textUsesSymbolTypeface(text) {
		if typeface := typefaceFromChild(rPr, "sym"); typeface != "" {
			return typeface
		}
	}
	if typeface := latinTypefaceFromRunProperties(rPr); typeface != "" {
		return typeface
	}
	if !textNeedsAlternateTypeface(text) {
		return ""
	}
	for _, name := range []string{"ea", "cs", "sym"} {
		if typeface := typefaceFromChild(rPr, name); typeface != "" {
			return typeface
		}
	}
	return ""
}

func latinTypefaceFromRunProperties(rPr *xmlNode) string {
	return typefaceFromChild(rPr, "latin")
}

func typefaceFromChild(node *xmlNode, name string) string {
	child := firstChild(node, name)
	if child == nil {
		return ""
	}
	typeface := attrValue(child.Attrs, "typeface")
	if strings.TrimSpace(typeface) == "" {
		return ""
	}
	return typeface
}

func textNeedsAlternateTypeface(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func textUsesSymbolTypeface(text string) bool {
	hasSymbol := false
	for _, r := range text {
		if isPrivateUseRune(r) {
			hasSymbol = true
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		return false
	}
	return hasSymbol
}

func isPrivateUseRune(r rune) bool {
	return (r >= '\uE000' && r <= '\uF8FF') || (r >= '\U000F0000' && r <= '\U000FFFFD') || (r >= '\U00100000' && r <= '\U0010FFFD')
}

func isUnderlineStyle(value string) bool {
	return value != "" && value != "none"
}

func runPropertiesUnderline(rPr *xmlNode) bool {
	if value := attrValue(rPr.Attrs, "u"); value != "" {
		return isUnderlineStyle(value)
	}
	if underline := firstChild(rPr, "uLn"); underline != nil {
		return firstChild(underline, "noFill") == nil
	}
	return false
}

func textStrikeType(value string) string {
	switch value {
	case "sngStrike", "dblStrike":
		return value
	default:
		return ""
	}
}

func textParagraphsHaveRunColor(paragraphs []textParagraph) bool {
	for _, paragraph := range paragraphs {
		for _, run := range paragraph.Runs {
			if run.HasTextColor {
				return true
			}
		}
	}
	return false
}

func textRunsHaveRunMetricProperties(runs []textRun) bool {
	for _, run := range runs {
		if run.FontSize != 0 || strings.TrimSpace(run.FontFamily) != "" || run.HasBold || run.HasItalic || run.Underline || run.Strike != "" || run.Baseline != 0 || run.HasCharSpacing || run.HasKern {
			return true
		}
	}
	return false
}

func trimTextRuns(runs []textRun) []textRun {
	start := 0
	for start < len(runs) {
		runs[start].Text = strings.TrimLeft(runs[start].Text, "\r\n")
		if runs[start].Text != "" {
			break
		}
		start++
	}
	end := len(runs)
	for end > start {
		runs[end-1].Text = strings.TrimRight(runs[end-1].Text, "\r\n")
		if runs[end-1].Text != "" {
			break
		}
		end--
	}
	if start >= end {
		return nil
	}
	return runs[start:end]
}

func textRunsFontSize(runs []textRun) int {
	fontSize := 0
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		if run.FontSize <= 0 {
			return 0
		}
		if fontSize != 0 && fontSize != run.FontSize {
			return 0
		}
		fontSize = run.FontSize
	}
	return fontSize
}

func textParagraphsFontSize(paragraphs []textParagraph) int {
	fontSize := 0
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Text) == "" {
			continue
		}
		size := paragraph.FontSize
		if size <= 0 {
			size = textRunsFontSize(paragraph.Runs)
		}
		if size <= 0 {
			return 0
		}
		if fontSize != 0 && fontSize != size {
			return 0
		}
		fontSize = size
	}
	return fontSize
}

func textParagraphsTextAlign(paragraphs []textParagraph) string {
	align := ""
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Text) == "" || paragraph.TextAlign == "" {
			continue
		}
		if align != "" && align != paragraph.TextAlign {
			return ""
		}
		align = paragraph.TextAlign
	}
	return align
}

func textRunsAllBold(runs []textRun) bool {
	seenTextRun := false
	allBold := true
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		seenTextRun = true
		if !run.Bold {
			allBold = false
		}
	}
	return seenTextRun && allBold
}

func textRunsAllItalic(runs []textRun) bool {
	seenTextRun := false
	allItalic := true
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		seenTextRun = true
		if !run.Italic {
			allItalic = false
		}
	}
	return seenTextRun && allItalic
}

func firstDescendant(node *xmlNode, name string) *xmlNode {
	if node.Name == name {
		return node
	}
	for _, child := range node.Children {
		if found := firstDescendant(child, name); found != nil {
			return found
		}
	}
	return nil
}

func descendantsByName(node *xmlNode, name string) []*xmlNode {
	var output []*xmlNode
	if node.Name == name {
		output = append(output, node)
	}
	for _, child := range node.Children {
		output = append(output, descendantsByName(child, name)...)
	}
	return output
}

func firstChild(node *xmlNode, name string) *xmlNode {
	for _, child := range node.Children {
		if child.Name == name {
			return child
		}
	}
	return nil
}

func childrenByName(node *xmlNode, name string) []*xmlNode {
	var output []*xmlNode
	for _, child := range node.Children {
		if child.Name == name {
			output = append(output, child)
		}
	}
	return output
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

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
	if !merged.Bold {
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
		if attrValue(defRPr.Attrs, "b") == "1" {
			style.Bold = true
		}
		if solidFill := firstChild(defRPr, "solidFill"); solidFill != nil {
			if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				style.HasTextColor = true
				style.TextColor = textColor
			}
		}
	}
	return style, style.FontSize > 0 || style.Bold || style.HasTextColor || style.TextAlign != "" || len(style.ParagraphStyles) > 0
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
	if !merged.Bold {
		merged.Bold = base.Bold
	}
	if !merged.Italic {
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
		element.TextParagraphs = []textParagraph{{Text: strings.TrimSpace(element.Text), Bold: true}}
		return
	}
	for index := range element.TextParagraphs {
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

func parseIntAttr(attrs []xml.Attr, name string) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			value, _ := strconv.ParseInt(attr.Value, 10, 64)
			return value
		}
	}
	return 0
}

func parsePercentAttr(attrs []xml.Attr, name string) int64 {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return parsePercentValue(attr.Value)
		}
	}
	return 0
}

func parsePercentValue(value string) int64 {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "%") {
		parsed, _ := strconv.ParseInt(trimmed, 10, 64)
		return parsed
	}
	parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(trimmed, "%")), 64)
	if err != nil {
		return 0
	}
	return int64(math.Round(parsed * 1000))
}

func parentElement(stack []string) string {
	if len(stack) < 2 {
		return ""
	}
	return stack[len(stack)-2]
}

func renderElements(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, elements []slideElement, tableStyles tableStyleSet) []model.SkipItem {
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return []model.SkipItem{{
			Code:    unsupportedCode,
			Message: fmt.Sprintf("slide relationships could not be parsed for rendering: %v", err),
			Part:    pptx.RelationshipsPartFor(slidePart),
		}}
	}
	relationshipByID := make(map[string]pptx.Relationship, len(relationships))
	for _, relationship := range relationships {
		relationshipByID[relationship.ID] = relationship
	}

	var unsupported []model.SkipItem
	for index := range elements {
		var items []model.SkipItem
		switch elements[index].Kind {
		case "pic":
			items = renderPicture(pkg, slidePart, size, img, &elements[index], relationshipByID)
		case "sp", "cxnSp":
			if elements[index].EmbedID != "" {
				items = append(items, renderPicture(pkg, slidePart, size, img, &elements[index], relationshipByID)...)
			}
			items = append(items, renderShape(slidePart, size, img, &elements[index])...)
		case "graphicFrame":
			items = renderGraphicFrame(pkg, slidePart, size, img, &elements[index], relationshipByID, tableStyles)
		}
		unsupported = append(unsupported, items...)
	}
	return unsupported
}

func renderGraphicFrame(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship, tableStyles tableStyleSet) []model.SkipItem {
	if element.DiagramDataID != "" {
		return renderDiagramGraphicFrame(pkg, slidePart, size, img, element, relationships)
	}
	if element.HasTable {
		return renderTableGraphicFrame(slidePart, size, img, element, tableStyles)
	}
	if element.Text == "" || !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return nil
	}
	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	).Intersect(img.Bounds())
	if target.Empty() {
		return nil
	}
	if err := drawShapeTextWithDPI(img, textBounds(target, *element, size, img.Bounds()), *element, renderDPIForCanvas(size, img.Bounds())); err != nil {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q text could not be rendered: %v", elementLabel(*element), err))}
	}
	element.Rendered = true
	return []model.SkipItem{unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q text was rendered with simplified layout", elementLabel(*element)))}
}

func renderTableGraphicFrame(slidePart string, size slideSize, img *image.RGBA, element *slideElement, tableStyles tableStyleSet) []model.SkipItem {
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 || len(element.Table.Rows) == 0 {
		return nil
	}
	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	).Intersect(img.Bounds())
	if target.Empty() {
		return nil
	}
	columnOffsets := tableGridOffsets(tableColumnWeights(element.Table), target.Min.X, target.Max.X, element.OffX, element.ExtCX, size.CX, img.Bounds().Dx())
	rowOffsets := tableRowOffsets(element.Table, target.Min.Y, target.Max.Y, element.OffY, element.ExtCY, size.CY, img.Bounds().Dy())
	style, hasStyle := tableStyleForTable(element.Table, tableStyles)
	backgroundEffectRendered := true
	if hasStyle {
		if style.HasBackgroundEffect && style.BackgroundEffect.HasShadow {
			backgroundElement := slideElement{
				PrstGeom:        "rect",
				HasShadow:       true,
				ShadowColor:     style.BackgroundEffect.ShadowColor,
				ShadowBlur:      style.BackgroundEffect.ShadowBlur,
				ShadowDistance:  style.BackgroundEffect.ShadowDistance,
				ShadowDirection: style.BackgroundEffect.ShadowDirection,
			}
			backgroundEffectRendered = drawShapeShadow(img, target, backgroundElement, size)
		}
		if style.HasBackground {
			drawTableBackgroundPaint(img, target, style.Background)
		}
	}
	for rowIndex, row := range element.Table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			if cellRect.Empty() {
				continue
			}
			style := resolvedTableCellStyle(element.Table, tableStyles, rowIndex, columnIndex)
			if fill, ok := tableCellFill(style, cell); ok {
				drawTableCellFill(img, cellRect, fill)
			}
		}
	}
	drawTableBorders(img, target, size, element.Table, tableStyles, columnOffsets, rowOffsets)
	var failures []string
	fontMessages := map[string]bool{}
	for rowIndex, row := range element.Table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || cell.Text == "" || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			textRect := tableCellTextRect(cellRect, cell, size, img.Bounds())
			if textRect.Empty() {
				textRect = cellRect
			}
			hasTextColor := cell.HasTextColor
			textColor := cell.TextColor
			style := resolvedTableCellStyle(element.Table, tableStyles, rowIndex, columnIndex)
			if color, ok := tableCellTextColor(style); ok && !hasTextColor {
				hasTextColor = true
				textColor = color
			}
			textParagraphs := cell.TextParagraphs
			if color, ok := tableCellTextColor(style); ok {
				textParagraphs = tableTextParagraphsWithColor(textParagraphs, cell.Text, color)
			}
			if tableCellTextBold(style) {
				textParagraphs = tableTextParagraphsWithBold(textParagraphs, cell.Text)
			}
			fontFamily := tableCellTextFontFamily(style)
			cellElement := slideElement{
				Text:           cell.Text,
				TextParagraphs: textParagraphs,
				FontFamily:     fontFamily,
				Italic:         tableCellTextItalic(style),
				FontSize:       cell.FontSize,
				HasTextColor:   hasTextColor,
				TextColor:      textColor,
				TextAlign:      cell.TextAlign,
				TextAnchor:     tableCellTextAnchor(cell),
			}
			if err := drawShapeTextWithDPI(img, textRect, cellElement, renderDPIForCanvas(size, img.Bounds())); err != nil {
				failures = append(failures, err.Error())
			}
			for _, message := range fontResolutionUnsupportedMessages(cellElement) {
				fontMessages[message] = true
			}
		}
	}
	element.Rendered = true
	if len(failures) > 0 {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q table text could not be rendered: %s", elementLabel(*element), strings.Join(failures, "; ")))}
	}
	unsupported := make([]model.SkipItem, 0, len(fontMessages)+1)
	for _, message := range sortedKeys(fontMessages) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table %s", elementLabel(*element), message)))
	}
	if hasStyle && style.HasBackgroundEffect && style.BackgroundEffect.HasShadow && !backgroundEffectRendered && style.BackgroundEffect.ShadowColor.A != 0 {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table background shadow geometry was not rendered", elementLabel(*element))))
	}
	for _, message := range element.Table.UnsupportedFeatures {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table %s", elementLabel(*element), message)))
	}
	return unsupported
}

func drawTableBackgroundPaint(img *image.RGBA, target image.Rectangle, paint backgroundPaint) {
	if target.Empty() {
		return
	}
	if paint.HasGradient {
		drawGradientRect(img, target, paint.Gradient, false)
		return
	}
	if paint.Color.A == 0 {
		return
	}
	drawTableCellFill(img, target, paint.Color)
}

func tableCellTextAnchor(cell tableCell) string {
	return cell.TextAnchor
}

func drawTableCellFill(img *image.RGBA, rect image.Rectangle, fill color.RGBA) {
	op := draw.Src
	if fill.A < 255 {
		op = draw.Over
	}
	draw.Draw(img, rect, &image.Uniform{C: fill}, image.Point{}, op)
}

func drawTableBorders(img *image.RGBA, target image.Rectangle, size slideSize, table tableModel, tableStyles tableStyleSet, columnOffsets []int, rowOffsets []int) {
	rowCount := len(table.Rows)
	columnCount := tableColumnCount(table)
	for rowIndex, row := range table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			if cellRect.Empty() {
				continue
			}
			style := resolvedTableCellStyle(table, tableStyles, rowIndex, columnIndex)
			drawTableCellBorderWithDefault(img, size, target, cellRect, cell.BorderTop, tableEdgeTop, tableEdgeBorder(style.Borders, tableEdgeTop, rowIndex, columnIndex, rowCount, columnCount), true)
			drawTableCellBorderWithDefault(img, size, target, cellRect, cell.BorderBottom, tableEdgeBottom, tableEdgeBorder(style.Borders, tableEdgeBottom, rowIndex, columnIndex, rowCount, columnCount), true)
			drawTableCellBorderWithDefault(img, size, target, cellRect, cell.BorderLeft, tableEdgeLeft, tableEdgeBorder(style.Borders, tableEdgeLeft, rowIndex, columnIndex, rowCount, columnCount), true)
			drawTableCellBorderWithDefault(img, size, target, cellRect, cell.BorderRight, tableEdgeRight, tableEdgeBorder(style.Borders, tableEdgeRight, rowIndex, columnIndex, rowCount, columnCount), true)
		}
	}
}

const (
	tableEdgeTop = iota
	tableEdgeBottom
	tableEdgeLeft
	tableEdgeRight
)

func tableEdgeBorder(borders tableStyleBorders, edge int, rowIndex int, columnIndex int, rowCount int, columnCount int) tableCellBorder {
	switch edge {
	case tableEdgeTop:
		if rowIndex > 0 && borders.InsideH.Specified {
			return borders.InsideH
		}
		if borders.Top.Specified {
			return borders.Top
		}
	case tableEdgeBottom:
		if rowCount <= 0 || rowIndex < rowCount-1 {
			if borders.InsideH.Specified {
				return borders.InsideH
			}
		}
		if borders.Bottom.Specified {
			return borders.Bottom
		}
		if borders.InsideH.Specified {
			return borders.InsideH
		}
	case tableEdgeLeft:
		if columnIndex > 0 && borders.InsideV.Specified {
			return borders.InsideV
		}
		if borders.Left.Specified {
			return borders.Left
		}
	case tableEdgeRight:
		if columnCount <= 0 || columnIndex < columnCount-1 {
			if borders.InsideV.Specified {
				return borders.InsideV
			}
		}
		if borders.Right.Specified {
			return borders.Right
		}
		if borders.InsideV.Specified {
			return borders.InsideV
		}
	}
	return defaultTableGridBorder()
}

func defaultTableGridBorder() tableCellBorder {
	return tableCellBorder{
		Specified: true,
		HasLine:   true,
		Color:     color.RGBA{R: 90, G: 90, B: 90, A: 255},
		Width:     9525,
	}
}

func drawTableCellBorder(img *image.RGBA, size slideSize, tableRect image.Rectangle, rect image.Rectangle, border tableCellBorder, edge int) {
	if !border.Specified || border.NoLine || !border.HasLine {
		return
	}
	width := emuLineWidthToPixels(border.Width, size.CX, img.Bounds().Dx())
	switch edge {
	case tableEdgeTop:
		drawTableBorderLine(img, rect.Min.X, rect.Min.Y, rect.Max.X-1, rect.Min.Y, border.Color, width, border.Dash, border.Compound, border.Cap, true)
	case tableEdgeBottom:
		y := rect.Max.Y
		if y >= tableRect.Max.Y {
			y = rect.Max.Y - 1
		}
		drawTableBorderLine(img, rect.Min.X, y, rect.Max.X-1, y, border.Color, width, border.Dash, border.Compound, border.Cap, true)
	case tableEdgeLeft:
		drawTableBorderLine(img, rect.Min.X, rect.Min.Y, rect.Min.X, rect.Max.Y-1, border.Color, width, border.Dash, border.Compound, border.Cap, false)
	case tableEdgeRight:
		x := rect.Max.X
		if x >= tableRect.Max.X {
			x = rect.Max.X - 1
		}
		drawTableBorderLine(img, x, rect.Min.Y, x, rect.Max.Y-1, border.Color, width, border.Dash, border.Compound, border.Cap, false)
	}
}

func isSupportedTableCompoundLine(compound string) bool {
	return compound == "" || compound == "sng" || compound == "dbl"
}

func drawTableBorderLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, compound string, cap string, horizontal bool) {
	if compound != "dbl" {
		drawStyledLine(img, x0, y0, x1, y1, c, width, dash, cap)
		return
	}
	strokeWidth, firstOffset, secondOffset := doubleCompoundLineMetrics(width)
	if horizontal {
		drawStyledLine(img, x0, y0+firstOffset, x1, y1+firstOffset, c, strokeWidth, dash, cap)
		drawStyledLine(img, x0, y0+secondOffset, x1, y1+secondOffset, c, strokeWidth, dash, cap)
		return
	}
	drawStyledLine(img, x0+firstOffset, y0, x1+firstOffset, y1, c, strokeWidth, dash, cap)
	drawStyledLine(img, x0+secondOffset, y0, x1+secondOffset, y1, c, strokeWidth, dash, cap)
}

func doubleCompoundLineMetrics(width int) (int, int, int) {
	if width < 1 {
		width = 1
	}
	strokeWidth := width / 3
	if strokeWidth < 1 {
		strokeWidth = 1
	}
	gap := width - (2 * strokeWidth)
	if gap < 1 {
		gap = 1
	}
	separation := strokeWidth + gap
	firstOffset := -(separation / 2)
	secondOffset := firstOffset + separation
	if firstOffset == secondOffset {
		secondOffset++
	}
	return strokeWidth, firstOffset, secondOffset
}

func drawTableCellBorderWithDefault(img *image.RGBA, size slideSize, tableRect image.Rectangle, rect image.Rectangle, border tableCellBorder, edge int, defaultBorder tableCellBorder, hasDefaultBorder bool) {
	if border.Specified {
		drawTableCellBorder(img, size, tableRect, rect, border, edge)
		return
	}
	if hasDefaultBorder {
		drawTableCellBorder(img, size, tableRect, rect, defaultBorder, edge)
	}
}

func tableCellTextRect(cellRect image.Rectangle, cell tableCell, size slideSize, imageBounds image.Rectangle) image.Rectangle {
	if !cell.HasMargins {
		return image.Rect(
			cellRect.Min.X+scaleEMU(defaultTableCellHorizontalMarginEMU, size.CX, imageBounds.Dx()),
			cellRect.Min.Y+scaleEMU(defaultTableCellVerticalMarginEMU, size.CY, imageBounds.Dy()),
			cellRect.Max.X-scaleEMU(defaultTableCellHorizontalMarginEMU, size.CX, imageBounds.Dx()),
			cellRect.Max.Y-scaleEMU(defaultTableCellVerticalMarginEMU, size.CY, imageBounds.Dy()),
		)
	}
	left := scaleEMU(cell.MarginLeft, size.CX, imageBounds.Dx())
	right := scaleEMU(cell.MarginRight, size.CX, imageBounds.Dx())
	top := scaleEMU(cell.MarginTop, size.CY, imageBounds.Dy())
	bottom := scaleEMU(cell.MarginBottom, size.CY, imageBounds.Dy())
	return image.Rect(cellRect.Min.X+left, cellRect.Min.Y+top, cellRect.Max.X-right, cellRect.Max.Y-bottom)
}

const (
	defaultTableCellHorizontalMarginEMU = 91440
	defaultTableCellVerticalMarginEMU   = 45720
)

func tableCellRect(columnOffsets []int, rowOffsets []int, rowIndex int, columnIndex int, cell tableCell) image.Rectangle {
	rowEnd := rowIndex + cell.RowSpan
	if rowEnd >= len(rowOffsets) {
		rowEnd = len(rowOffsets) - 1
	}
	if rowEnd <= rowIndex {
		rowEnd = rowIndex + 1
	}
	columnEnd := columnIndex + cell.ColSpan
	if columnEnd >= len(columnOffsets) {
		columnEnd = len(columnOffsets) - 1
	}
	if columnEnd <= columnIndex {
		columnEnd = columnIndex + 1
	}
	return image.Rect(columnOffsets[columnIndex], rowOffsets[rowIndex], columnOffsets[columnEnd], rowOffsets[rowEnd])
}

func tableCellFill(style tableStyleRegion, cell tableCell) (color.RGBA, bool) {
	if cell.NoFill {
		return color.RGBA{}, false
	}
	if cell.HasFill {
		return cell.FillColor, true
	}
	if style.NoFill {
		return color.RGBA{}, false
	}
	if style.HasFill {
		return style.FillColor, true
	}
	return color.RGBA{}, false
}

func tableCellTextColor(style tableStyleRegion) (color.RGBA, bool) {
	return style.TextColor, style.HasTextColor
}

func tableCellTextBold(style tableStyleRegion) bool {
	return style.HasBold && style.Bold
}

func tableCellTextItalic(style tableStyleRegion) bool {
	return style.HasItalic && style.Italic
}

func tableCellTextFontFamily(style tableStyleRegion) string {
	return style.FontFamily
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key, ok := range values {
		if ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func resolvedTableCellStyle(table tableModel, styles tableStyleSet, rowIndex int, columnIndex int) tableStyleRegion {
	style, ok := tableStyleForTable(table, styles)
	if !ok {
		return tableStyleRegion{}
	}
	var resolved tableStyleRegion
	for _, regionName := range tableStyleRegionNamesForCell(table, rowIndex, columnIndex) {
		region, ok := style.Regions[regionName]
		if !ok {
			continue
		}
		mergeTableStyleRegion(&resolved, region)
	}
	return resolved
}

func tableStyleForTable(table tableModel, styles tableStyleSet) (tableStyle, bool) {
	if len(styles.Styles) == 0 {
		return tableStyle{}, false
	}
	if table.StyleID != "" {
		if style, ok := styles.Styles[normalizedTableStyleID(table.StyleID)]; ok {
			return style, true
		}
	}
	if styles.DefaultID != "" {
		if style, ok := styles.Styles[normalizedTableStyleID(styles.DefaultID)]; ok {
			return style, true
		}
	}
	return tableStyle{}, false
}

func tableStyleRegionNamesForCell(table tableModel, rowIndex int, columnIndex int) []string {
	rowCount := len(table.Rows)
	columnCount := tableColumnCount(table)
	names := []string{"wholeTbl"}
	if table.BandRow {
		switch tableBandIndex(rowIndex, table.FirstRow) {
		case 0:
			names = append(names, "band1H")
		case 1:
			names = append(names, "band2H")
		}
	}
	if table.BandCol {
		switch tableBandIndex(columnIndex, table.FirstCol) {
		case 0:
			names = append(names, "band1V")
		case 1:
			names = append(names, "band2V")
		}
	}
	if table.FirstCol && columnIndex == 0 {
		names = append(names, "firstCol")
	}
	if table.LastCol && columnCount > 0 && columnIndex == columnCount-1 {
		names = append(names, "lastCol")
	}
	if table.FirstRow && rowIndex == 0 {
		names = append(names, "firstRow")
	}
	if table.LastRow && rowCount > 0 && rowIndex == rowCount-1 {
		names = append(names, "lastRow")
	}
	if table.FirstRow && table.FirstCol && rowIndex == 0 && columnIndex == 0 {
		names = append(names, "nwCell")
	}
	if table.FirstRow && table.LastCol && rowIndex == 0 && columnCount > 0 && columnIndex == columnCount-1 {
		names = append(names, "neCell")
	}
	if table.LastRow && table.FirstCol && rowCount > 0 && rowIndex == rowCount-1 && columnIndex == 0 {
		names = append(names, "swCell")
	}
	if table.LastRow && table.LastCol && rowCount > 0 && columnCount > 0 && rowIndex == rowCount-1 && columnIndex == columnCount-1 {
		names = append(names, "seCell")
	}
	return names
}

func tableBandIndex(index int, skipFirst bool) int {
	if skipFirst {
		index--
	}
	if index < 0 {
		return -1
	}
	return index % 2
}

func tableColumnCount(table tableModel) int {
	columnCount := len(table.Columns)
	for _, row := range table.Rows {
		if len(row.Cells) > columnCount {
			columnCount = len(row.Cells)
		}
	}
	return columnCount
}

func mergeTableStyleRegion(dst *tableStyleRegion, src tableStyleRegion) {
	if src.NoFill {
		dst.NoFill = true
		dst.HasFill = false
		dst.FillColor = color.RGBA{}
	} else if src.HasFill {
		dst.NoFill = false
		dst.HasFill = true
		dst.FillColor = src.FillColor
	}
	if src.HasTextColor {
		dst.HasTextColor = true
		dst.TextColor = src.TextColor
	}
	if src.FontFamily != "" {
		dst.FontFamily = src.FontFamily
	}
	if src.HasBold {
		dst.HasBold = true
		dst.Bold = src.Bold
	}
	if src.HasItalic {
		dst.HasItalic = true
		dst.Italic = src.Italic
	}
	mergeTableBorder(&dst.Borders.Left, src.Borders.Left)
	mergeTableBorder(&dst.Borders.Right, src.Borders.Right)
	mergeTableBorder(&dst.Borders.Top, src.Borders.Top)
	mergeTableBorder(&dst.Borders.Bottom, src.Borders.Bottom)
	mergeTableBorder(&dst.Borders.InsideH, src.Borders.InsideH)
	mergeTableBorder(&dst.Borders.InsideV, src.Borders.InsideV)
}

func mergeTableBorder(dst *tableCellBorder, src tableCellBorder) {
	if src.Specified {
		*dst = src
	}
}

func tableTextParagraphsWithBold(paragraphs []textParagraph, fallbackText string) []textParagraph {
	if len(paragraphs) == 0 {
		if strings.TrimSpace(fallbackText) == "" {
			return nil
		}
		return []textParagraph{{Text: strings.TrimSpace(fallbackText), Bold: true}}
	}
	output := make([]textParagraph, len(paragraphs))
	copy(output, paragraphs)
	for paragraphIndex := range output {
		output[paragraphIndex].Bold = true
		runs := make([]textRun, len(output[paragraphIndex].Runs))
		copy(runs, output[paragraphIndex].Runs)
		for runIndex := range runs {
			runs[runIndex].Bold = true
		}
		output[paragraphIndex].Runs = runs
	}
	return output
}

func tableTextParagraphsWithColor(paragraphs []textParagraph, fallbackText string, textColor color.RGBA) []textParagraph {
	if len(paragraphs) == 0 {
		if strings.TrimSpace(fallbackText) == "" {
			return nil
		}
		return []textParagraph{{Text: strings.TrimSpace(fallbackText), HasTextColor: true, TextColor: textColor}}
	}
	output := make([]textParagraph, len(paragraphs))
	copy(output, paragraphs)
	for paragraphIndex := range output {
		output[paragraphIndex].HasTextColor = true
		output[paragraphIndex].TextColor = textColor
		runs := make([]textRun, len(output[paragraphIndex].Runs))
		copy(runs, output[paragraphIndex].Runs)
		output[paragraphIndex].Runs = runs
	}
	return output
}

func tableColumnWeights(table tableModel) []int64 {
	columnCount := len(table.Columns)
	for _, row := range table.Rows {
		if len(row.Cells) > columnCount {
			columnCount = len(row.Cells)
		}
	}
	if columnCount == 0 {
		return nil
	}
	weights := make([]int64, columnCount)
	for index := range weights {
		if index < len(table.Columns) && table.Columns[index] > 0 {
			weights[index] = table.Columns[index]
		} else {
			weights[index] = 1
		}
	}
	return weights
}

func tableRowWeights(table tableModel) []int64 {
	if len(table.Rows) == 0 {
		return nil
	}
	weights := make([]int64, len(table.Rows))
	for index, row := range table.Rows {
		if row.HasHeight {
			weights[index] = row.Height
		} else {
			weights[index] = 1
		}
	}
	return weights
}

func tableRowOffsets(table tableModel, min int, max int, originEMU int64, frameEMU int64, slideEMU int64, canvasPixels int) []int {
	weights := tableRowWeights(table)
	if len(weights) <= 1 || !table.FirstRow || !tableFirstRowHasSpanningCells(table) || frameEMU <= 0 {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 || total >= frameEMU {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	headerEnd := scaleEMU(originEMU+weights[0], slideEMU, canvasPixels)
	if headerEnd <= min || headerEnd >= max {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	offsets := make([]int, 0, len(weights)+1)
	offsets = append(offsets, min, headerEnd)
	bodyOffsets := proportionalOffsets(weights[1:], headerEnd, max)
	offsets = append(offsets, bodyOffsets[1:]...)
	return offsets
}

func tableFirstRowHasSpanningCells(table tableModel) bool {
	if len(table.Rows) == 0 {
		return false
	}
	for _, cell := range table.Rows[0].Cells {
		if cell.ColSpan > 1 || cell.HMerge {
			return true
		}
	}
	return false
}

func tableGridOffsets(weights []int64, min int, max int, originEMU int64, frameEMU int64, slideEMU int64, canvasPixels int) []int {
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total > 0 && frameEMU > 0 && total == frameEMU {
		offsets := make([]int, len(weights)+1)
		offsets[0] = min
		running := int64(0)
		for index, weight := range weights {
			running += weight
			offsets[index+1] = scaleEMU(originEMU+running, slideEMU, canvasPixels)
		}
		return offsets
	}
	return proportionalOffsets(weights, min, max)
}

func proportionalOffsets(weights []int64, min int, max int) []int {
	offsets := make([]int, len(weights)+1)
	offsets[0] = min
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 {
		total = int64(len(weights))
		for index := range weights {
			weights[index] = 1
		}
	}
	span := max - min
	running := int64(0)
	for index, weight := range weights {
		running += weight
		offsets[index+1] = min + int(math.Round(float64(span)*float64(running)/float64(total)))
	}
	offsets[len(offsets)-1] = max
	return offsets
}

func renderDiagramGraphicFrame(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return nil
	}
	drawingPart, ok, err := diagramDrawingPart(pkg, slidePart, element.DiagramDataID, relationships)
	if err != nil {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q diagram could not be resolved: %v", elementLabel(*element), err))}
	}
	if !ok {
		return nil
	}
	diagramElements := diagramDrawingElements(pkg, slidePart, drawingPart)
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

func diagramDrawingElements(pkg *pptx.Package, slidePart, drawingPart string) []slideElement {
	colors := themeColorsForPart(pkg, slidePart, packageThemeColors(pkg))
	fonts := themeFontsForPart(pkg, slidePart, packageThemeFonts(pkg))
	effectStyles := themeEffectStylesForPart(pkg, slidePart)
	fillStyles := themeFillStylesForPart(pkg, slidePart)
	lineStyles := themeLineStylesForPart(pkg, slidePart)
	elements := collectSlideElementsWithThemeEffectsAndFills(pkg.Parts[drawingPart], colors, effectStyles, fillStyles, lineStyles)
	return applyThemeFontFamilies(elements, fonts)
}

func diagramDrawingPart(pkg *pptx.Package, slidePart string, diagramDataID string, relationships map[string]pptx.Relationship) (string, bool, error) {
	dataRel, ok := relationships[diagramDataID]
	if !ok || dataRel.Type != diagramDataRelType || (dataRel.TargetMode != "" && !strings.EqualFold(dataRel.TargetMode, "Internal")) {
		return "", false, nil
	}
	dataPart := pptx.ResolveTargetPart(slidePart, dataRel.Target)
	data, ok := pkg.Parts[dataPart]
	if !ok {
		return "", false, fmt.Errorf("diagram data part %s is missing", dataPart)
	}
	root, err := parseXMLNode(data)
	if err != nil {
		return "", false, fmt.Errorf("parse diagram data %s: %w", dataPart, err)
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
	if _, ok := pkg.Parts[drawingPart]; !ok {
		return "", false, fmt.Errorf("diagram drawing part %s is missing", drawingPart)
	}
	return drawingPart, true, nil
}

func fitDiagramElementsToFrame(elements []slideElement, frame slideElement) []slideElement {
	maxX, maxY := diagramElementExtents(elements)
	if maxX <= 0 || maxY <= 0 {
		return elements
	}
	scaleX := int64(1)
	scaleY := int64(1)
	denomX := int64(1)
	denomY := int64(1)
	if maxX > frame.ExtCX {
		scaleX = frame.ExtCX
		denomX = maxX
	}
	if maxY > frame.ExtCY {
		scaleY = frame.ExtCY
		denomY = maxY
	}
	for index := range elements {
		if !elements[index].HasTransform {
			continue
		}
		elements[index].OffX = frame.OffX + elements[index].OffX*scaleX/denomX
		elements[index].OffY = frame.OffY + elements[index].OffY*scaleY/denomY
		elements[index].ExtCX = elements[index].ExtCX * scaleX / denomX
		elements[index].ExtCY = elements[index].ExtCY * scaleY / denomY
		if elements[index].HasTextTransform {
			elements[index].TextOffX = frame.OffX + elements[index].TextOffX*scaleX/denomX
			elements[index].TextOffY = frame.OffY + elements[index].TextOffY*scaleY/denomY
			elements[index].TextExtCX = elements[index].TextExtCX * scaleX / denomX
			elements[index].TextExtCY = elements[index].TextExtCY * scaleY / denomY
		}
	}
	return elements
}

func diagramElementExtents(elements []slideElement) (int64, int64) {
	var maxX int64
	var maxY int64
	for _, element := range elements {
		if !element.HasTransform {
			continue
		}
		if right := element.OffX + element.ExtCX; right > maxX {
			maxX = right
		}
		if bottom := element.OffY + element.ExtCY; bottom > maxY {
			maxY = bottom
		}
		if element.HasTextTransform {
			if right := element.TextOffX + element.TextExtCX; right > maxX {
				maxX = right
			}
			if bottom := element.TextOffY + element.TextExtCY; bottom > maxY {
				maxY = bottom
			}
		}
	}
	return maxX, maxY
}

func renderShape(slidePart string, size slideSize, img *image.RGBA, element *slideElement) []model.SkipItem {
	if !element.HasTransform {
		return nil
	}
	var unsupported []model.SkipItem
	rendered := element.Rendered
	if isLineGeometry(element.PrstGeom) && element.HasLine && !element.NoLine && (element.ExtCX != 0 || element.ExtCY != 0) {
		startX, startY, endX, endY := lineEndpointsForElement(*element, size, img.Bounds())
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		drawStyledLine(img, startX, startY, endX, endY, element.LineColor, lineWidth, element.LineDash, element.LineCap)
		markerPartial := false
		if element.HeadLineMarker != "" {
			if element.HeadLineMarker == "triangle" {
				drawLineTriangleMarker(img, startX, startY, startX-endX, startY-endY, element.LineColor, lineWidth, element.HeadLineMarkerWidth, element.HeadLineMarkerLength)
			} else {
				markerPartial = true
			}
		}
		if element.TailLineMarker != "" {
			if element.TailLineMarker == "triangle" {
				drawLineTriangleMarker(img, endX, endY, endX-startX, endY-startY, element.LineColor, lineWidth, element.TailLineMarkerWidth, element.TailLineMarkerLength)
			} else {
				markerPartial = true
			}
		}
		rendered = true
		if markerPartial {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("connector object %q line markers were not rendered", elementLabel(*element))))
		}
	}
	if element.ExtCX <= 0 || element.ExtCY <= 0 {
		element.Rendered = rendered
		return unsupported
	}
	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	)
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		adjusted, err := shapeAutofitTarget(*element, target, size, img.Bounds())
		if err == nil {
			target = adjusted
		}
	}
	target = target.Intersect(img.Bounds())
	if target.Empty() {
		element.Rendered = rendered
		return unsupported
	}
	if element.HasShadow {
		for _, message := range shadowTransformUnsupportedMessages(*element) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
		}
		if drawShapeShadow(img, target, *element, size) {
			rendered = true
		} else if element.ShadowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q outer shadow geometry was not rendered", elementLabel(*element))))
		}
	}
	for _, message := range shape3DUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
	}
	for _, message := range shapeSoftEdgeUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
	}
	gradientFillRendered := false
	if element.HasFillGradient && !element.NoFill {
		switch element.PrstGeom {
		case "rect", "":
			drawGradientRect(img, target, element.FillGradient, false)
			rendered = true
			gradientFillRendered = true
		case "roundRect":
			drawGradientRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopLeft: true, TopRight: true, BottomLeft: true, BottomRight: true}, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		case "round1Rect":
			drawGradientRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopRight: true}, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		case "ellipse":
			drawGradientEllipse(img, target, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		}
	}
	if element.HasFillGradient && !gradientFillRendered && !element.NoFill {
		if points, ok := presetPolygonPointsForElement(*element); ok {
			drawGradientPolygon(img, target, points, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		}
	}
	if element.HasFillGradient && !gradientFillRendered && !element.NoFill && len(element.CustomPath) >= 3 {
		drawGradientPolygon(img, target, transformedPathPoints(element.CustomPath, *element), element.FillGradient)
		rendered = true
		gradientFillRendered = true
	}
	if element.HasFill && !element.NoFill && !gradientFillRendered {
		switch element.PrstGeom {
		case "rect":
			fillShapeRect(img, target, element.FillColor)
			rendered = true
		case "roundRect":
			fillRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopLeft: true, TopRight: true, BottomLeft: true, BottomRight: true}, element.FillColor)
			rendered = true
		case "round1Rect":
			fillRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopRight: true}, element.FillColor)
			rendered = true
		}
	}
	if points, ok := presetPolygonPointsForElement(*element); ok && element.HasFill && !element.NoFill {
		drawPolygon(img, target, points, element.FillColor)
		rendered = true
	}
	if element.PrstGeom == "ellipse" && element.HasFill && !element.NoFill {
		drawEllipse(img, target, element.FillColor)
		rendered = true
	}
	if (element.PrstGeom == "curvedDownArrow" || element.PrstGeom == "curvedUpArrow") && element.HasFill && !element.NoFill {
		drawCurvedArrow(img, target, *element, element.FillColor)
		rendered = true
	}
	if element.PrstGeom == "rightBrace" && element.HasFill && !element.NoFill {
		drawPolygon(img, target, rightBracePresetPath(*element), element.FillColor)
		rendered = true
	}
	if len(element.CustomPath) >= 3 && element.HasFill && !element.NoFill {
		drawPolygon(img, target, transformedPathPoints(element.CustomPath, *element), element.FillColor)
		rendered = true
	}
	if element.HasFillGradient && !gradientFillRendered {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q gradient fill was rendered as a solid fallback", elementLabel(*element))))
	} else if element.HasFillGradient && !element.FillGradient.FullySupported {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q gradient fill was rendered with simplified layout", elementLabel(*element))))
	}
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		switch element.PrstGeom {
		case "rect", "roundRect", "round1Rect", "":
			drawStyledRectOutlineAligned(img, target, element.LineColor, lineWidth, element.LineDash, element.LineAlign)
			rendered = true
		case "ellipse":
			drawEllipseOutline(img, target, element.LineColor, lineWidth)
			rendered = true
		case "triangle", "rightArrow", "notchedRightArrow", "chevron":
			if points, ok := presetPolygonPointsForElement(*element); ok {
				drawPolygonOutline(img, target, points, element.LineColor, lineWidth)
				rendered = true
			}
		case "curvedDownArrow", "curvedUpArrow":
			if points := curvedArrowPresetOutlinePath(*element); len(points) >= 2 {
				drawOpenPathOutline(img, target, points, element.LineColor, lineWidth)
				rendered = true
			}
		case "rightBrace":
			drawRightBrace(img, target, *element, element.LineColor, lineWidth)
			rendered = true
		}
		if len(element.CustomPath) >= 3 {
			drawPolygonOutline(img, target, transformedPathPoints(element.CustomPath, *element), element.LineColor, lineWidth)
			rendered = true
		}
	}
	if element.Text != "" {
		for _, message := range fontResolutionUnsupportedMessages(*element) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
		}
		for _, message := range textLayoutUnsupportedMessagesForTarget(*element, textBounds(target, *element, size, img.Bounds()), renderDPIForCanvas(size, img.Bounds())) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
		}
		if err := drawShapeTextForElement(img, target, *element, size); err != nil {
			if rendered {
				element.Rendered = true
			}
			return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("shape object %q text could not be rendered: %v", elementLabel(*element), err))}
		}
		rendered = true
	}
	element.Rendered = rendered
	return unsupported
}

func lineEndpointsForElement(element slideElement, size slideSize, bounds image.Rectangle) (int, int, int, int) {
	left := scaleEMU(element.OffX, size.CX, bounds.Dx())
	top := scaleEMU(element.OffY, size.CY, bounds.Dy())
	right := scaleEMU(element.OffX+element.ExtCX, size.CX, bounds.Dx())
	bottom := scaleEMU(element.OffY+element.ExtCY, size.CY, bounds.Dy())
	startX, startY := left, top
	endX, endY := right, bottom
	if element.FlipH {
		startX, endX = endX, startX
	}
	if element.FlipV {
		startY, endY = endY, startY
	}
	return startX, startY, endX, endY
}

func drawShapeTextForElement(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) error {
	bounds := textBounds(target, element, size, img.Bounds())
	dpi := renderDPIForCanvas(size, img.Bounds())
	rotation := normalizedRotationDegrees(element.Rotation)
	switch rotation {
	case 90, 180, 270:
		return drawRotatedShapeText(img, bounds, element, rotation, dpi)
	default:
		return drawShapeTextWithDPI(img, bounds, element, dpi)
	}
}

func shapeAutofitTarget(element slideElement, target image.Rectangle, size slideSize, canvas image.Rectangle) (image.Rectangle, error) {
	if target.Empty() {
		return target, nil
	}
	bounds := textBounds(target, element, size, canvas)
	if bounds.Empty() || bounds.Dy() <= 0 {
		return target, nil
	}
	dpi := renderDPIForCanvas(size, canvas)
	width, height, err := measuredElementTextSize(element, bounds, dpi)
	if err != nil {
		return target, err
	}
	if element.TextWrap == "none" && width > bounds.Dx() {
		target.Max.X += width - bounds.Dx()
	}
	if height > 0 && height != bounds.Dy() {
		target.Max.Y += height - bounds.Dy()
	}
	return target, nil
}

func measuredElementTextHeight(element slideElement, bounds image.Rectangle, dpi int) (int, error) {
	_, height, err := measuredElementTextSize(element, bounds, dpi)
	return height, err
}

func measuredElementTextSize(element slideElement, bounds image.Rectangle, dpi int) (int, int, error) {
	if shouldFitNormalAutofit(element) {
		element = fitNormalAutofitElement(element, bounds, dpi)
	}
	element = scaledTextElement(element, dpi)
	faces := newFontFaceCacheWithDPI(element.Italic, element.FontFamily, dpi, element.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(element.FontSize, false)
	if err != nil {
		return 0, 0, err
	}
	boldFace, err := faces.Get(element.FontSize, true)
	if err != nil {
		return 0, 0, err
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, element, bounds.Dx(), dpi)
	if err != nil {
		return 0, 0, err
	}
	measured, err := measureTextRenderLines(faces, lines, element.FontSize)
	if err != nil {
		return 0, 0, err
	}
	width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
	if err != nil {
		return 0, 0, err
	}
	return width, measuredTextHeight(measured), nil
}

func normalizedRotationDegrees(rotation int) int {
	if rotation == 0 {
		return 0
	}
	degrees := int(math.Round(float64(rotation) / 60000))
	degrees %= 360
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

func drawRotatedShapeText(img *image.RGBA, bounds image.Rectangle, element slideElement, rotation int, dpi int) error {
	if bounds.Empty() {
		return nil
	}
	temp := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if err := drawShapeTextWithDPI(temp, temp.Bounds(), element, dpi); err != nil {
		return err
	}
	rotated := rotateRGBA(temp, rotation)
	center := image.Point{X: bounds.Min.X + bounds.Dx()/2, Y: bounds.Min.Y + bounds.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
	return nil
}

func rotateRGBA(src *image.RGBA, rotation int) *image.RGBA {
	bounds := src.Bounds()
	switch rotation {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(bounds.Max.Y-1-y, x-bounds.Min.X, src.RGBAAt(x, y))
			}
		}
		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(bounds.Max.X-1-x, bounds.Max.Y-1-y, src.RGBAAt(x, y))
			}
		}
		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(y-bounds.Min.Y, bounds.Max.X-1-x, src.RGBAAt(x, y))
			}
		}
		return dst
	default:
		return rotateRGBAArbitrary(src, float64(rotation))
	}
}

func rotateRGBAArbitrary(src *image.RGBA, degrees float64) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	radians := degrees * math.Pi / 180
	sin, cos := math.Sin(radians), math.Cos(radians)
	outputWidth := int(math.Ceil(math.Abs(float64(width)*cos) + math.Abs(float64(height)*sin)))
	outputHeight := int(math.Ceil(math.Abs(float64(width)*sin) + math.Abs(float64(height)*cos)))
	if outputWidth <= 0 || outputHeight <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	sourceCenterX := float64(bounds.Min.X) + float64(width-1)/2
	sourceCenterY := float64(bounds.Min.Y) + float64(height-1)/2
	outputCenterX := float64(outputWidth-1) / 2
	outputCenterY := float64(outputHeight-1) / 2
	for y := 0; y < outputHeight; y++ {
		for x := 0; x < outputWidth; x++ {
			dx := float64(x) - outputCenterX
			dy := float64(y) - outputCenterY
			sourceX := sourceCenterX + dx*cos + dy*sin
			sourceY := sourceCenterY - dx*sin + dy*cos
			if sourceX < float64(bounds.Min.X) || sourceX > float64(bounds.Max.X-1) || sourceY < float64(bounds.Min.Y) || sourceY > float64(bounds.Max.Y-1) {
				continue
			}
			dst.SetRGBA(x, y, bilinearRGBAAt(src, sourceX, sourceY))
		}
	}
	return dst
}

func bilinearRGBAAt(src *image.RGBA, x float64, y float64) color.RGBA {
	bounds := src.Bounds()
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := min(x0+1, bounds.Max.X-1)
	y1 := min(y0+1, bounds.Max.Y-1)
	fx := x - float64(x0)
	fy := y - float64(y0)
	c00 := src.RGBAAt(x0, y0)
	c10 := src.RGBAAt(x1, y0)
	c01 := src.RGBAAt(x0, y1)
	c11 := src.RGBAAt(x1, y1)
	return color.RGBA{
		R: interpolateBilinearChannel(c00.R, c10.R, c01.R, c11.R, fx, fy),
		G: interpolateBilinearChannel(c00.G, c10.G, c01.G, c11.G, fx, fy),
		B: interpolateBilinearChannel(c00.B, c10.B, c01.B, c11.B, fx, fy),
		A: interpolateBilinearChannel(c00.A, c10.A, c01.A, c11.A, fx, fy),
	}
}

func interpolateBilinearChannel(c00 uint8, c10 uint8, c01 uint8, c11 uint8, fx float64, fy float64) uint8 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bottom := float64(c01)*(1-fx) + float64(c11)*fx
	return clampColor(int64(math.Round(top*(1-fy) + bottom*fy)))
}

func drawRGBAAt(dst *image.RGBA, target image.Rectangle, src *image.RGBA) {
	clipped := target.Intersect(dst.Bounds())
	if clipped.Empty() {
		return
	}
	sourcePoint := image.Point{
		X: src.Bounds().Min.X + clipped.Min.X - target.Min.X,
		Y: src.Bounds().Min.Y + clipped.Min.Y - target.Min.Y,
	}
	draw.Draw(dst, clipped, src, sourcePoint, draw.Over)
}

func isRectGeometry(geometry string) bool {
	return geometry == "rect" || geometry == "roundRect" || geometry == "round1Rect"
}

func isLineGeometry(geometry string) bool {
	return geometry == "line" || geometry == "straightConnector1"
}

func isFilledPresetPolygonGeometry(geometry string) bool {
	_, ok := presetPolygonPoints(geometry)
	return ok
}

func presetPolygonPoints(geometry string) ([]pathPoint, bool) {
	switch geometry {
	case "triangle":
		return []pathPoint{{X: 0.5, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, true
	case "rightArrow":
		return []pathPoint{
			{X: 0, Y: 0.25},
			{X: 0.65, Y: 0.25},
			{X: 0.65, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.65, Y: 1},
			{X: 0.65, Y: 0.75},
			{X: 0, Y: 0.75},
		}, true
	case "notchedRightArrow":
		return []pathPoint{
			{X: 0, Y: 0},
			{X: 0.78, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.78, Y: 1},
			{X: 0, Y: 1},
			{X: 0.18, Y: 0.5},
		}, true
	case "chevron":
		return []pathPoint{
			{X: 0, Y: 0},
			{X: 0.75, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.75, Y: 1},
			{X: 0, Y: 1},
			{X: 0.25, Y: 0.5},
		}, true
	default:
		return nil, false
	}
}

func presetPolygonPointsForElement(element slideElement) ([]pathPoint, bool) {
	switch element.PrstGeom {
	case "rightArrow":
		return transformedPathPoints(rightArrowPresetPoints(element), element), true
	case "chevron":
		return transformedPathPoints(chevronPresetPoints(element), element), true
	case "notchedRightArrow":
		return transformedPathPoints(notchedRightArrowPresetPoints(element), element), true
	}
	points, ok := presetPolygonPoints(element.PrstGeom)
	if !ok {
		return nil, false
	}
	if element.PrstGeom == "triangle" {
		if adj, ok := element.PrstGeomAdjustments["adj"]; ok {
			points[0].X = clampPathCoordinate(float64(adj) / 100000)
		}
	}
	return transformedPathPoints(points, element), true
}

func chevronPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj := presetAdjustment(element, "adj", 50000)
	maxAdj := 100000 * w / ss
	a := clampFloat(adj, 0, maxAdj)
	x1 := ss * a / 100000
	x2 := w - x1
	return []pathPoint{
		{X: 0, Y: 0},
		{X: x2 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x2 / w, Y: 1},
		{X: 0, Y: 1},
		{X: x1 / w, Y: 0.5},
	}
}

func rightArrowPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj1 := presetAdjustment(element, "adj1", 50000)
	adj2 := presetAdjustment(element, "adj2", 50000)
	a1 := clampFloat(adj1, 0, 100000)
	maxAdj2 := 100000 * w / ss
	a2 := clampFloat(adj2, 0, maxAdj2)
	dx1 := ss * a2 / 100000
	x1 := w - dx1
	dy1 := h * a1 / 200000
	y1 := 0.5 - dy1/h
	y2 := 0.5 + dy1/h
	return []pathPoint{
		{X: 0, Y: y1},
		{X: x1 / w, Y: y1},
		{X: x1 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x1 / w, Y: 1},
		{X: x1 / w, Y: y2},
		{X: 0, Y: y2},
	}
}

func notchedRightArrowPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj1 := presetAdjustment(element, "adj1", 50000)
	adj2 := presetAdjustment(element, "adj2", 50000)
	a1 := clampFloat(adj1, 0, 100000)
	maxAdj2 := 100000 * w / ss
	a2 := clampFloat(adj2, 0, maxAdj2)
	dx2 := ss * a2 / 100000
	x2 := w - dx2
	dy1 := h * a1 / 200000
	x1 := 0.0
	if h > 0 {
		x1 = dy1 * dx2 / (h / 2)
	}
	y1 := 0.5 - dy1/h
	y2 := 0.5 + dy1/h
	return []pathPoint{
		{X: 0, Y: y1},
		{X: x2 / w, Y: y1},
		{X: x2 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x2 / w, Y: 1},
		{X: x2 / w, Y: y2},
		{X: 0, Y: y2},
		{X: x1 / w, Y: 0.5},
	}
}

func positiveGeometryDimensions(element slideElement) (float64, float64) {
	w := float64(element.ExtCX)
	h := float64(element.ExtCY)
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	return w, h
}

func presetAdjustment(element slideElement, name string, fallback float64) float64 {
	if value, ok := element.PrstGeomAdjustments[name]; ok {
		return float64(value)
	}
	return fallback
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampPathCoordinate(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func transformedPathPoints(points []pathPoint, element slideElement) []pathPoint {
	if !element.FlipH && !element.FlipV && normalizedRotationDegrees(element.Rotation) == 0 {
		return points
	}
	transformed := make([]pathPoint, 0, len(points))
	for _, point := range points {
		next := point
		if element.FlipH {
			next.X = 1 - next.X
		}
		if element.FlipV {
			next.Y = 1 - next.Y
		}
		transformed = append(transformed, next)
	}
	return rotatePathPoints(transformed, normalizedRotationDegrees(element.Rotation))
}

func rotatePathPoints(points []pathPoint, rotation int) []pathPoint {
	if rotation != 90 && rotation != 180 && rotation != 270 {
		return points
	}
	rotated := make([]pathPoint, 0, len(points))
	for _, point := range points {
		switch rotation {
		case 90:
			rotated = append(rotated, pathPoint{X: 1 - point.Y, Y: point.X})
		case 180:
			rotated = append(rotated, pathPoint{X: 1 - point.X, Y: 1 - point.Y})
		case 270:
			rotated = append(rotated, pathPoint{X: point.Y, Y: 1 - point.X})
		}
	}
	return rotated
}

func drawShapeShadow(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) bool {
	if element.ShadowColor.A == 0 {
		return false
	}
	offset := shadowOffset(element, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := shadowBlurPixels(element, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	switch {
	case isRectGeometry(element.PrstGeom):
		drawSoftRect(img, shadowBounds, element.ShadowColor, blur)
	case element.PrstGeom == "ellipse":
		drawSoftEllipse(img, shadowBounds, element.ShadowColor, blur)
	case element.PrstGeom == "triangle":
		drawSoftPolygon(img, shadowBounds, transformedPathPoints([]pathPoint{{X: 0.5, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, element), element.ShadowColor, blur)
	case element.PrstGeom == "rightArrow":
		drawSoftPolygon(img, shadowBounds, transformedPathPoints([]pathPoint{
			{X: 0, Y: 0.25},
			{X: 0.65, Y: 0.25},
			{X: 0.65, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.65, Y: 1},
			{X: 0.65, Y: 0.75},
			{X: 0, Y: 0.75},
		}, element), element.ShadowColor, blur)
	case isFilledPresetPolygonGeometry(element.PrstGeom):
		if points, ok := presetPolygonPointsForElement(element); ok {
			drawSoftPolygon(img, shadowBounds, points, element.ShadowColor, blur)
		} else {
			return false
		}
	case len(element.CustomPath) >= 3:
		drawSoftPolygon(img, shadowBounds, transformedPathPoints(element.CustomPath, element), element.ShadowColor, blur)
	default:
		return false
	}
	return true
}

func shadowTransformUnsupportedMessages(element slideElement) []string {
	var messages []string
	if (element.HasShadowScaleX && element.ShadowScaleX != 100000) || (element.HasShadowScaleY && element.ShadowScaleY != 100000) || (element.HasShadowSkewX && element.ShadowSkewX != 0) || (element.HasShadowSkewY && element.ShadowSkewY != 0) {
		messages = append(messages, "outer shadow scale/skew transform was not rendered")
	}
	if element.HasShadowRotateWithShape && !element.ShadowRotateWithShape && normalizedRotationDegrees(element.Rotation) != 0 {
		messages = append(messages, "outer shadow rotate-with-shape transform was not rendered")
	}
	return messages
}

func shape3DUnsupportedMessages(element slideElement) []string {
	if !element.HasShape3D {
		return nil
	}
	if len(element.Shape3DFeatures) == 0 {
		return []string{"3-D shape properties were not rendered"}
	}
	features := append([]string{}, element.Shape3DFeatures...)
	sort.Strings(features)
	return []string{fmt.Sprintf("%s were not rendered", strings.Join(features, ", "))}
}

func shapeSoftEdgeUnsupportedMessages(element slideElement) []string {
	if !element.HasSoftEdge || element.SoftEdgeRadius <= 0 {
		return nil
	}
	return []string{"soft edge effect was not rendered"}
}

func shadowIntersectsCanvas(bounds image.Rectangle, blur int, canvas image.Rectangle) bool {
	if bounds.Empty() || canvas.Empty() {
		return false
	}
	if blur > 0 {
		bounds = bounds.Inset(-blur)
	}
	return !bounds.Intersect(canvas).Empty()
}

func shadowOffset(element slideElement, size slideSize, outputWidth int) image.Point {
	distance := scaleEMU(element.ShadowDistance, size.CX, outputWidth)
	if distance == 0 && element.ShadowDistance > 0 {
		distance = 1
	}
	angle := float64(element.ShadowDirection) / 60000 * math.Pi / 180
	return image.Point{
		X: int(math.Round(math.Cos(angle) * float64(distance))),
		Y: int(math.Round(math.Sin(angle) * float64(distance))),
	}
}

func shadowBlurPixels(element slideElement, size slideSize, outputWidth int) int {
	blur := scaleEMU(element.ShadowBlur, size.CX, outputWidth)
	if blur < 0 {
		return 0
	}
	return blur
}

func drawGradientBackground(img *image.RGBA, gradient gradientPaint) {
	drawGradientRect(img, img.Bounds(), gradient, true)
}

func drawGradientRect(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint, replace bool) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	if len(gradient.Stops) == 1 {
		op := draw.Over
		if replace || gradient.Stops[0].Color.A == 255 {
			op = draw.Src
		}
		draw.Draw(img, bounds, &image.Uniform{C: gradient.Stops[0].Color}, image.Point{}, op)
		return
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			var position int64
			switch gradient.Path {
			case "circle":
				position = radialGradientPosition(bounds, x, y, gradient)
			case "rect":
				position = rectangularGradientPosition(bounds, x, y, gradient)
			default:
				position = linearGradientPosition(bounds, x, y, gradient)
			}
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			if replace || c.A == 255 {
				img.SetRGBA(x, y, c)
			} else {
				blendPixel(img, x, y, c)
			}
		}
	}
}

func drawGradientEllipse(img *image.RGBA, bounds image.Rectangle, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	radiusX := float64(bounds.Dx()) / 2
	radiusY := float64(bounds.Dy()) / 2
	if radiusX <= 0 || radiusY <= 0 {
		return
	}
	centerX := float64(bounds.Min.X) + radiusX
	centerY := float64(bounds.Min.Y) + radiusY
	drawGradientWithCoverage(img, bounds, bounds, gradient, func(x int, y int) int {
		coverage := 0
		for _, offset := range coverageSampleOffsets {
			dx := (float64(x) + offset.x - centerX) / radiusX
			dy := (float64(y) + offset.y - centerY) / radiusY
			if dx*dx+dy*dy <= 1 {
				coverage++
			}
		}
		return coverage
	})
}

func drawGradientRoundRect(img *image.RGBA, bounds image.Rectangle, radius int, corners roundedCorners, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	if radius <= 0 {
		drawGradientRect(img, bounds, gradient, false)
		return
	}
	drawGradientWithCoverage(img, bounds, bounds, gradient, func(x int, y int) int {
		return roundRectCoverage(float64(x), float64(y), bounds, radius, corners)
	})
}

func drawGradientPolygon(img *image.RGBA, bounds image.Rectangle, points []pathPoint, gradient gradientPaint) {
	bounds = bounds.Intersect(img.Bounds())
	if bounds.Empty() || len(points) < 3 || len(gradient.Stops) == 0 {
		return
	}
	polygon := polygonImagePoints(bounds, points)
	gradientBounds := pathPointBoundsRect(bounds, points).Intersect(img.Bounds())
	if gradientBounds.Empty() {
		return
	}
	drawGradientWithCoverage(img, bounds, gradientBounds, gradient, func(x int, y int) int {
		return polygonCoverage(float64(x), float64(y), polygon)
	})
}

func drawGradientWithCoverage(img *image.RGBA, paintBounds image.Rectangle, gradientBounds image.Rectangle, gradient gradientPaint, coverageAt func(x int, y int) int) {
	paintBounds = paintBounds.Intersect(img.Bounds())
	if paintBounds.Empty() || gradientBounds.Empty() || len(gradient.Stops) == 0 {
		return
	}
	for y := paintBounds.Min.Y; y < paintBounds.Max.Y; y++ {
		for x := paintBounds.Min.X; x < paintBounds.Max.X; x++ {
			coverage := coverageAt(x, y)
			if coverage <= 0 {
				continue
			}
			var position int64
			switch gradient.Path {
			case "circle":
				position = radialGradientPosition(gradientBounds, x, y, gradient)
			case "rect":
				position = rectangularGradientPosition(gradientBounds, x, y, gradient)
			default:
				position = linearGradientPosition(gradientBounds, x, y, gradient)
			}
			c := colorAtGradientPositionForPath(gradient.Stops, position, gradient.Path)
			if coverage >= 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
				continue
			}
			c.A = coverageAlpha(c.A, coverage)
			blendPixel(img, x, y, c)
		}
	}
}

func linearGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	if bounds.Dx() <= 1 && bounds.Dy() <= 1 {
		return 0
	}
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	if !gradient.HasAngle {
		height := bounds.Dy()
		if height <= 1 {
			return 0
		}
		position := (sampleY - float64(bounds.Min.Y)) / float64(height)
		if position < 0 {
			position = 0
		} else if position > 1 {
			position = 1
		}
		return int64(math.Round(position * 100000))
	}
	radians := float64(gradient.Angle) / 60000 * math.Pi / 180
	dx := math.Cos(radians)
	dy := math.Sin(radians)
	if gradient.HasScaled && gradient.Scaled {
		dx *= float64(bounds.Dx())
		dy *= float64(bounds.Dy())
	}
	corners := []struct {
		X float64
		Y float64
	}{
		{X: float64(bounds.Min.X), Y: float64(bounds.Min.Y)},
		{X: float64(bounds.Max.X - 1), Y: float64(bounds.Min.Y)},
		{X: float64(bounds.Min.X), Y: float64(bounds.Max.Y - 1)},
		{X: float64(bounds.Max.X - 1), Y: float64(bounds.Max.Y - 1)},
	}
	minProjection := math.Inf(1)
	maxProjection := math.Inf(-1)
	for _, corner := range corners {
		projection := corner.X*dx + corner.Y*dy
		if projection < minProjection {
			minProjection = projection
		}
		if projection > maxProjection {
			maxProjection = projection
		}
	}
	span := maxProjection - minProjection
	if span <= 0 {
		return 0
	}
	projection := (sampleX*dx + sampleY*dy - minProjection) / span
	if projection < 0 {
		projection = 0
	} else if projection > 1 {
		projection = 1
	}
	return int64(math.Round(projection * 100000))
}

type floatRect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

func rectangularGradientFocusRect(bounds image.Rectangle, gradient gradientPaint) floatRect {
	if !gradient.HasFillRect {
		centerX := float64(bounds.Min.X) + float64(bounds.Dx())/2
		centerY := float64(bounds.Min.Y) + float64(bounds.Dy())/2
		return floatRect{MinX: centerX, MinY: centerY, MaxX: centerX, MaxY: centerY}
	}
	width := float64(bounds.Dx())
	height := float64(bounds.Dy())
	left := float64(bounds.Min.X) + width*float64(gradient.FillRect.Left)/100000
	top := float64(bounds.Min.Y) + height*float64(gradient.FillRect.Top)/100000
	right := float64(bounds.Max.X) - width*float64(gradient.FillRect.Right)/100000
	bottom := float64(bounds.Max.Y) - height*float64(gradient.FillRect.Bottom)/100000
	if right < left {
		center := (left + right) / 2
		left = center
		right = center
	}
	if bottom < top {
		center := (top + bottom) / 2
		top = center
		bottom = center
	}
	return floatRect{MinX: left, MinY: top, MaxX: right, MaxY: bottom}
}

func rectangularGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return 0
	}
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	outer := floatRect{
		MinX: float64(bounds.Min.X),
		MinY: float64(bounds.Min.Y),
		MaxX: float64(bounds.Max.X),
		MaxY: float64(bounds.Max.Y),
	}
	inner := rectangularGradientFocusRect(bounds, gradient)
	if pointInRect(sampleX, sampleY, inner) {
		return 0
	}
	position := 0.0
	if sampleX < inner.MinX {
		position = math.Max(position, normalizedGradientDistance(inner.MinX-sampleX, inner.MinX-outer.MinX))
	} else if sampleX > inner.MaxX {
		position = math.Max(position, normalizedGradientDistance(sampleX-inner.MaxX, outer.MaxX-inner.MaxX))
	}
	if sampleY < inner.MinY {
		position = math.Max(position, normalizedGradientDistance(inner.MinY-sampleY, inner.MinY-outer.MinY))
	} else if sampleY > inner.MaxY {
		position = math.Max(position, normalizedGradientDistance(sampleY-inner.MaxY, outer.MaxY-inner.MaxY))
	}
	return int64(math.Round(clampGradientRatio(position) * 100000))
}

func normalizedGradientDistance(distance float64, span float64) float64 {
	if distance <= 0 {
		return 0
	}
	if span <= 0 {
		return 1
	}
	return distance / span
}

func pointInRect(x float64, y float64, rect floatRect) bool {
	return x >= rect.MinX && x <= rect.MaxX && y >= rect.MinY && y <= rect.MaxY
}

func gradientFocusRect(bounds image.Rectangle, gradient gradientPaint) floatRect {
	if gradient.Path != "circle" {
		return floatRect{}
	}
	outer := radialGradientOuterRect(bounds)
	if !gradient.HasFillRect {
		centerX := (outer.MinX + outer.MaxX) / 2
		centerY := (outer.MinY + outer.MaxY) / 2
		return floatRect{MinX: centerX, MinY: centerY, MaxX: centerX, MaxY: centerY}
	}
	width := outer.MaxX - outer.MinX
	height := outer.MaxY - outer.MinY
	left := outer.MinX + width*float64(gradient.FillRect.Left)/100000
	top := outer.MinY + height*float64(gradient.FillRect.Top)/100000
	right := outer.MaxX - width*float64(gradient.FillRect.Right)/100000
	bottom := outer.MaxY - height*float64(gradient.FillRect.Bottom)/100000
	if right < left {
		center := (left + right) / 2
		left = center
		right = center
	}
	if bottom < top {
		center := (top + bottom) / 2
		top = center
		bottom = center
	}
	return floatRect{MinX: left, MinY: top, MaxX: right, MaxY: bottom}
}

func radialGradientOuterRect(bounds image.Rectangle) floatRect {
	centerX := float64(bounds.Min.X) + float64(bounds.Dx())/2
	centerY := float64(bounds.Min.Y) + float64(bounds.Dy())/2
	diameter := math.Hypot(float64(bounds.Dx()), float64(bounds.Dy()))
	radius := diameter / 2
	return floatRect{
		MinX: centerX - radius,
		MinY: centerY - radius,
		MaxX: centerX + radius,
		MaxY: centerY + radius,
	}
}

func radialGradientPosition(bounds image.Rectangle, x int, y int, gradient gradientPaint) int64 {
	origin := radialGradientFocusPoint(bounds, gradient)
	sampleX := float64(x) + 0.5
	sampleY := float64(y) + 0.5
	dx := sampleX - origin.X
	dy := sampleY - origin.Y
	distance := math.Hypot(dx, dy)
	if distance <= 0 {
		return 0
	}

	outer := radialGradientOuterRect(bounds)
	inner := gradientFocusRect(bounds, gradient)
	if pointInEllipse(sampleX, sampleY, inner) {
		return 0
	}
	innerDistance := rayEllipseExitDistance(origin, dx/distance, dy/distance, inner)
	outerDistance := rayEllipseExitDistance(origin, dx/distance, dy/distance, outer)
	if outerDistance <= innerDistance {
		return 100000
	}
	position := (distance - innerDistance) / (outerDistance - innerDistance)
	if position < 0 {
		position = 0
	} else if position > 1 {
		position = 1
	}
	return int64(math.Round(position * 100000))
}

func pointInEllipse(x float64, y float64, rect floatRect) bool {
	width := rect.MaxX - rect.MinX
	height := rect.MaxY - rect.MinY
	if width <= 0 || height <= 0 {
		return false
	}
	rx := width / 2
	ry := height / 2
	cx := (rect.MinX + rect.MaxX) / 2
	cy := (rect.MinY + rect.MaxY) / 2
	nx := (x - cx) / rx
	ny := (y - cy) / ry
	return nx*nx+ny*ny <= 1
}

func rayEllipseExitDistance(origin floatPoint, unitX float64, unitY float64, rect floatRect) float64 {
	width := rect.MaxX - rect.MinX
	height := rect.MaxY - rect.MinY
	if width <= 0 || height <= 0 {
		return 0
	}
	rx := width / 2
	ry := height / 2
	cx := (rect.MinX + rect.MaxX) / 2
	cy := (rect.MinY + rect.MaxY) / 2
	ox := origin.X - cx
	oy := origin.Y - cy
	a := unitX*unitX/(rx*rx) + unitY*unitY/(ry*ry)
	if a <= 0 {
		return 0
	}
	b := 2 * (ox*unitX/(rx*rx) + oy*unitY/(ry*ry))
	c := ox*ox/(rx*rx) + oy*oy/(ry*ry) - 1
	discriminant := b*b - 4*a*c
	if discriminant < 0 {
		return 0
	}
	root := math.Sqrt(discriminant)
	first := (-b - root) / (2 * a)
	second := (-b + root) / (2 * a)
	switch {
	case first >= 0 && second >= 0:
		return math.Max(first, second)
	case second >= 0:
		return second
	case first >= 0:
		return first
	default:
		return 0
	}
}

type floatPoint struct {
	X float64
	Y float64
}

func radialGradientFocusPoint(bounds image.Rectangle, gradient gradientPaint) floatPoint {
	inner := gradientFocusRect(bounds, gradient)
	outer := radialGradientOuterRect(bounds)
	outerLeft := outer.MinX
	outerTop := outer.MinY
	outerWidth := outer.MaxX - outer.MinX
	outerHeight := outer.MaxY - outer.MinY
	innerWidth := inner.MaxX - inner.MinX
	innerHeight := inner.MaxY - inner.MinY
	point := floatPoint{X: inner.MinX, Y: inner.MinY}
	if innerWidth > 0 {
		widthDiff := outerWidth - innerWidth
		if math.Abs(widthDiff) > 2*math.SmallestNonzeroFloat64 {
			point.X += innerWidth * (inner.MinX - outerLeft) / widthDiff
		}
	}
	if innerHeight > 0 {
		heightDiff := outerHeight - innerHeight
		if math.Abs(heightDiff) > 2*math.SmallestNonzeroFloat64 {
			point.Y += innerHeight * (inner.MinY - outerTop) / heightDiff
		}
	}
	return point
}

func colorAtGradientPositionForPath(stops []gradientStop, position int64, path string) color.RGBA {
	return colorAtGradientPosition(stops, position)
}

func colorAtGradientPosition(stops []gradientStop, position int64) color.RGBA {
	if len(stops) == 0 {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	if c, ok := colorAtOfficeGammaGradientPosition(stops, position); ok {
		return c
	}
	if position <= stops[0].Position {
		return stops[0].Color
	}
	for index := 1; index < len(stops); index++ {
		right := stops[index]
		if position > right.Position {
			continue
		}
		left := stops[index-1]
		span := right.Position - left.Position
		if span <= 0 {
			return right.Color
		}
		numerator := position - left.Position
		return color.RGBA{
			R: interpolateChannel(left.Color.R, right.Color.R, numerator, span),
			G: interpolateChannel(left.Color.G, right.Color.G, numerator, span),
			B: interpolateChannel(left.Color.B, right.Color.B, numerator, span),
			A: interpolateChannel(left.Color.A, right.Color.A, numerator, span),
		}
	}
	return stops[len(stops)-1].Color
}

func colorAtOfficeGammaGradientPosition(stops []gradientStop, position int64) (color.RGBA, bool) {
	if len(stops) == 2 && stops[0].Position == 0 && stops[1].Position == 100000 {
		t := clampGradientRatio(float64(position) / 100000)
		return interpolateOfficeGammaColor(stops[0].Color, stops[1].Color, t), true
	}
	if len(stops) == 3 && stops[0].Position == 0 && stops[2].Position == 100000 && stops[0].Color == stops[2].Color {
		mid := stops[1].Position
		if mid <= 0 || mid >= 100000 {
			return color.RGBA{}, false
		}
		if position <= mid {
			t := clampGradientRatio(float64(position) / float64(mid))
			return interpolateOfficeGammaColor(stops[0].Color, stops[1].Color, t), true
		}
		t := clampGradientRatio(float64(100000-position) / float64(100000-mid))
		return interpolateOfficeGammaColor(stops[2].Color, stops[1].Color, t), true
	}
	return color.RGBA{}, false
}

func clampGradientRatio(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

func interpolateOfficeGammaColor(left color.RGBA, right color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: interpolateOfficeGammaChannel(left.R, right.R, t),
		G: interpolateOfficeGammaChannel(left.G, right.G, t),
		B: interpolateOfficeGammaChannel(left.B, right.B, t),
		A: interpolateLinearFloatChannel(left.A, right.A, t),
	}
}

func interpolateOfficeGammaChannel(left uint8, right uint8, t float64) uint8 {
	if left == right {
		return left
	}
	ratio := math.Pow(t, 1.875)
	if right > left {
		ratio = 1 - math.Pow(1-t, 1.875)
	}
	return interpolateFloatChannel(left, right, ratio)
}

func interpolateLinearFloatChannel(left uint8, right uint8, t float64) uint8 {
	return interpolateFloatChannel(left, right, t)
}

func interpolateFloatChannel(left uint8, right uint8, t float64) uint8 {
	value := float64(left) + (float64(right)-float64(left))*t
	if value < 0 {
		value = 0
	}
	if value > 255 {
		value = 255
	}
	return uint8(math.Round(value))
}

func interpolateChannel(left uint8, right uint8, numerator int64, denominator int64) uint8 {
	if denominator <= 0 {
		return right
	}
	return uint8((int64(left)*(denominator-numerator) + int64(right)*numerator + denominator/2) / denominator)
}

func drawShapeText(img *image.RGBA, bounds image.Rectangle, element slideElement) error {
	return drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI)
}

func drawShapeTextWithDPI(img *image.RGBA, bounds image.Rectangle, element slideElement, dpi int) error {
	clip := shapeTextClipRect(element, bounds, img.Bounds())
	if clip == img.Bounds() {
		return drawShapeTextLayerWithDPI(img, bounds, element, dpi)
	}
	if clip.Empty() {
		return nil
	}
	layer := image.NewRGBA(img.Bounds())
	if err := drawShapeTextLayerWithDPI(layer, bounds, element, dpi); err != nil {
		return err
	}
	draw.Draw(img, clip, layer, clip.Min, draw.Over)
	return nil
}

func shapeTextClipRect(element slideElement, bounds image.Rectangle, canvas image.Rectangle) image.Rectangle {
	clip := canvas
	if !shapeTextHorizontalOverflowAllowed(element) {
		clip.Min.X = maxInt(clip.Min.X, bounds.Min.X)
		clip.Max.X = minInt(clip.Max.X, bounds.Max.X)
	}
	if !shapeTextVerticalOverflowAllowed(element) {
		clip.Min.Y = maxInt(clip.Min.Y, bounds.Min.Y)
		clip.Max.Y = minInt(clip.Max.Y, bounds.Max.Y)
	}
	return clip.Intersect(canvas)
}

func shapeTextHorizontalOverflowAllowed(element slideElement) bool {
	switch strings.TrimSpace(element.TextHorizontalOverflow) {
	case "", "overflow":
		return true
	default:
		return false
	}
}

func shapeTextVerticalOverflowAllowed(element slideElement) bool {
	switch strings.TrimSpace(element.TextVerticalOverflow) {
	case "", "overflow":
		return true
	default:
		return false
	}
}

func drawShapeTextLayerWithDPI(img *image.RGBA, bounds image.Rectangle, element slideElement, dpi int) error {
	dpi = normalizeOutputDPI(dpi)
	if shouldFitNormalAutofit(element) {
		element = fitNormalAutofitElement(element, bounds, dpi)
	}
	element = scaledTextElement(element, dpi)
	faces := newFontFaceCacheWithDPI(element.Italic, element.FontFamily, dpi, element.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(element.FontSize, false)
	if err != nil {
		return err
	}
	boldFace, err := faces.Get(element.FontSize, true)
	if err != nil {
		return err
	}

	textColor := color.RGBA{A: 255}
	if element.HasTextColor {
		textColor = element.TextColor
	}
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
	}
	maxWidth := bounds.Dx()
	lines, err := textRenderLinesForElement(faces, face, boldFace, element, maxWidth, dpi)
	if err != nil {
		return err
	}
	measured, err := measureTextRenderLines(faces, lines, element.FontSize)
	if err != nil {
		return err
	}
	if element.TextAnchorCenter {
		width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
		if err != nil {
			return err
		}
		bounds = anchorCenteredTextBounds(bounds, width)
	}
	y := anchoredTextTop(bounds, measuredTextAnchorHeight(measured, element.TextAnchor), element.TextAnchor)
	verticalLimit := bounds.Max.Y
	if shapeTextVerticalOverflowAllowed(element) {
		verticalLimit = img.Bounds().Max.Y
	}
	for _, line := range lines {
		if len(measured) == 0 {
			return nil
		}
		current := measured[0]
		measured = measured[1:]
		y += current.SpaceBefore
		baseline := y + current.Ascent
		if y > verticalLimit {
			return nil
		}
		textAlign := line.TextAlign
		if textAlign == "" {
			textAlign = element.TextAlign
		}
		if len(line.Segments) > 0 {
			lineBounds := textLineBounds(bounds, line)
			x, err := alignedSegmentedTextXAtDPI(faces, face, boldFace, line.Segments, lineBounds, textAlign, dpi, line.TabStops)
			if err != nil {
				return err
			}
			if line.HasXOffset && (textAlign == "" || textAlign == "l") {
				x = bounds.Min.X + line.XOffset
			}
			justifyExtra := 0
			justifyRemainder := 0
			if line.Justify && textAlign == "just" {
				lineWidth, err := measureStyledSegmentsAtDPI(faces, face, boldFace, line.Segments, dpi, line.TabStops)
				if err != nil {
					return err
				}
				if spaceCount := textLineSpaceCount(line.Segments); spaceCount > 0 && lineWidth < lineBounds.Dx() {
					extra := lineBounds.Dx() - lineWidth
					justifyExtra = extra / spaceCount
					justifyRemainder = extra % spaceCount
				}
			}
			lineStart := x
			for _, segment := range line.Segments {
				fontSize := segment.FontSize
				if fontSize == 0 {
					fontSize = element.FontSize
				}
				segmentFace, err := faces.GetForFamily(segment.FontFamily, fontSize, segment.Bold, segment.Italic)
				if err != nil {
					return err
				}
				segmentFace = faceWithSegmentKerning(segmentFace, segment)
				if segment.HasTextColor {
					drawer.Src = image.NewUniform(segment.TextColor)
				} else {
					drawer.Src = image.NewUniform(textColor)
				}
				if segment.Marker != "" {
					markerColor := textColor
					if segment.HasTextColor {
						markerColor = segment.TextColor
					}
					drawTextMarker(img, segment.Marker, x, baseline-current.Ascent/2, markerPixelSizeAtDPI(segment, element.FontSize, dpi), markerColor)
					x += markerSegmentWidthAtDPI(segment, element.FontSize, dpi)
					continue
				}
				drawer.Face = segmentFace
				segmentWidth := measureTextSegmentWithTabsAndSpacingAtDPI(segmentFace, segment.Text, 0, dpi, line.TabStops, segment.CharSpacing)
				if justifyExtra > 0 || justifyRemainder > 0 {
					segmentSpaces := textSpaceCount(segment.Text)
					segmentWidth += segmentSpaces * justifyExtra
					segmentWidth += minInt(segmentSpaces, justifyRemainder)
				}
				segmentBaseline := baseline - segmentBaselineShiftAtDPI(segment, element.FontSize, dpi)
				if segment.HasHighlightColor {
					drawTextHighlight(img, segmentFace, x, segmentBaseline, segmentWidth, segment.HighlightColor)
				}
				segmentStart := x
				if justifyExtra > 0 || justifyRemainder > 0 {
					x = drawJustifiedTextSegment(drawer, segmentFace, segment.Text, x, segmentBaseline, justifyExtra, &justifyRemainder, segment.CharSpacing, dpi)
				} else {
					x = drawTextSegmentWithTabsAndSpacingAtDPI(drawer, segmentFace, segment.Text, x, lineStart, segmentBaseline, dpi, line.TabStops, segment.CharSpacing)
				}
				if segment.Underline {
					drawTextUnderline(img, segmentFace, segmentStart, segmentBaseline, x-segmentStart, underlineColorForSegment(segment, textColor))
				}
				if segment.Strike != "" {
					drawTextStrikethrough(img, segmentFace, segmentStart, segmentBaseline, x-segmentStart, segment.Strike, textColorForSegment(segment, textColor))
				}
			}
		} else {
			drawer.Src = image.NewUniform(textColor)
			drawer.Face = current.Face
			drawer.Dot = fixed.Point26_6{X: fixed.I(alignedTextX(current.Face, line.Text, textLineBounds(bounds, line), textAlign)), Y: fixed.I(baseline)}
			drawer.DrawString(line.Text)
		}
		y += current.Height
	}
	return nil
}

func shouldFitNormalAutofit(element slideElement) bool {
	return element.HasNormAutofit
}

func drawTextHighlight(img *image.RGBA, face font.Face, x int, baseline int, width int, c color.RGBA) {
	if width <= 0 {
		return
	}
	metrics := face.Metrics()
	top := baseline - metrics.Ascent.Ceil()
	bottom := baseline + metrics.Descent.Ceil()
	if bottom <= top {
		bottom = top + metrics.Height.Ceil()
	}
	rect := image.Rect(x, top, x+width, bottom).Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
}

func drawTextUnderline(img *image.RGBA, face font.Face, x int, baseline int, width int, c color.RGBA) {
	if width <= 0 || c.A == 0 {
		return
	}
	metrics := face.Metrics()
	y := baseline + maxInt(1, metrics.Descent.Ceil()/3)
	lineWidth := maxInt(1, metrics.Height.Ceil()/16)
	drawLine(img, x, y, x+width-1, y, c, lineWidth)
}

func drawTextStrikethrough(img *image.RGBA, face font.Face, x int, baseline int, width int, strike string, c color.RGBA) {
	if width <= 0 || c.A == 0 || strike == "" {
		return
	}
	metrics := face.Metrics()
	top := baseline - metrics.Ascent.Ceil()
	bottom := baseline + metrics.Descent.Ceil()
	if bottom <= top {
		bottom = top + metrics.Height.Ceil()
	}
	lineWidth := maxInt(1, metrics.Height.Ceil()/16)
	center := top + (bottom-top)/2
	if strike == "dblStrike" {
		gap := maxInt(2, lineWidth*2)
		drawLine(img, x, center-gap/2, x+width-1, center-gap/2, c, lineWidth)
		drawLine(img, x, center+gap/2, x+width-1, center+gap/2, c, lineWidth)
		return
	}
	drawLine(img, x, center, x+width-1, center, c, lineWidth)
}

func textColorForSegment(segment textLineSegment, fallback color.RGBA) color.RGBA {
	if segment.HasTextColor {
		return segment.TextColor
	}
	return fallback
}

func underlineColorForSegment(segment textLineSegment, fallback color.RGBA) color.RGBA {
	if segment.HasUnderlineColor {
		return segment.UnderlineColor
	}
	return textColorForSegment(segment, fallback)
}

func fitNormalAutofitElement(element slideElement, bounds image.Rectangle, dpiOverride ...int) slideElement {
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return element
	}
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	startScale := 100000
	if element.FontScalePct > 0 && element.FontScalePct < startScale {
		startScale = element.FontScalePct
	}
	bestScale := startScale
	maxLines := normalAutofitMaxSoftLines(element)
	if element.LineSpacingReductionPct > 0 {
		withoutLineSpacingReduction := element
		withoutLineSpacingReduction.LineSpacingReductionPct = 0
		withoutLineSpacingReduction.HasLineSpacingReductionPct = false
		if textHeightFitsAtScale(withoutLineSpacingReduction, bounds, startScale, maxLines, dpi) {
			element.LineSpacingReductionPct = 0
			element.HasLineSpacingReductionPct = false
		}
	}
	if scale, ok := largestFittingNormalAutofitScale(element, bounds, startScale, maxLines, dpi); ok {
		bestScale = scale
	}
	if bestScale != element.FontScalePct {
		element.FontScalePct = bestScale
	}
	return element
}

func largestFittingNormalAutofitScale(element slideElement, bounds image.Rectangle, startScale int, maxLines int, dpi int) (int, bool) {
	if startScale < minimumNormalAutofitFontScalePct {
		startScale = minimumNormalAutofitFontScalePct
	}
	if textFitsAtScale(element, bounds, startScale, maxLines, dpi) {
		return startScale, true
	}
	if !textFitsAtScale(element, bounds, minimumNormalAutofitFontScalePct, maxLines, dpi) {
		return 0, false
	}
	low := minimumNormalAutofitFontScalePct
	high := startScale - 1
	best := low
	for low <= high {
		mid := low + (high-low)/2
		if textFitsAtScale(element, bounds, mid, maxLines, dpi) {
			best = mid
			low = mid + 1
			continue
		}
		high = mid - 1
	}
	return best, true
}

const (
	minimumNormalAutofitFontScalePct = 1000
)

func normalAutofitHasAuthoredScale(element slideElement) bool {
	return element.HasNormAutofit && (element.HasFontScalePct || element.HasLineSpacingReductionPct)
}

func normalAutofitMaxSoftLines(element slideElement) int {
	if len(element.TextParagraphs) != 1 {
		return 0
	}
	if element.TextWrap != "none" {
		count := explicitTextLineCount(element)
		if count > 1 {
			return count
		}
		return 0
	}
	count := explicitTextLineCount(element)
	if count < 1 {
		return 1
	}
	return count
}

func explicitTextLineCount(element slideElement) int {
	if len(element.TextParagraphs) != 1 {
		return 0
	}
	count := 1
	if strings.Contains(element.Text, "\n") {
		count = maxInt(count, strings.Count(element.Text, "\n")+1)
	}
	if strings.Contains(element.TextParagraphs[0].Text, "\n") {
		count = maxInt(count, strings.Count(element.TextParagraphs[0].Text, "\n")+1)
	}
	for _, run := range element.TextParagraphs[0].Runs {
		if strings.Contains(run.Text, "\n") {
			count = maxInt(count, strings.Count(run.Text, "\n")+1)
		}
	}
	return count
}

func drawTextSegmentWithTabs(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int) int {
	return drawTextSegmentWithTabsAtDPI(drawer, face, text, x, lineStart, baseline, defaultOutputDPI, nil)
}

func drawTextSegmentWithTabsAtDPI(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int, dpi int, tabStops []int) int {
	return drawTextSegmentWithTabsAndSpacingAtDPI(drawer, face, text, x, lineStart, baseline, dpi, tabStops, 0)
}

func drawTextSegmentWithTabsAndSpacingAtDPI(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int, dpi int, tabStops []int, charSpacing int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	if !strings.Contains(text, "\t") {
		return drawTextWithCharacterSpacing(drawer, face, text, x, baseline, spacingPixels)
	}
	parts := strings.Split(text, "\t")
	for index, part := range parts {
		if part != "" {
			x = drawTextWithCharacterSpacing(drawer, face, part, x, baseline, spacingPixels)
		}
		if index < len(parts)-1 {
			x += textTabAdvanceAtDPI(x-lineStart, dpi, tabStops)
		}
	}
	return x
}

func drawTextWithCharacterSpacing(drawer *font.Drawer, face font.Face, text string, x int, baseline int, spacingPixels int) int {
	if spacingPixels == 0 || utf8.RuneCountInString(text) <= 1 {
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(text)
		return x + measureString(face, text)
	}
	runes := []rune(text)
	for index, value := range runes {
		chunk := string(value)
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(chunk)
		x += measureString(face, chunk)
		if index < len(runes)-1 {
			x += spacingPixels
		}
	}
	return x
}

func drawJustifiedTextSegment(drawer *font.Drawer, face font.Face, text string, x int, baseline int, extraPerSpace int, extraRemainder *int, charSpacing int, dpi int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	runes := []rune(text)
	for index, value := range runes {
		chunk := string(value)
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(chunk)
		x += measureString(face, chunk)
		if index < len(runes)-1 {
			x += spacingPixels
		}
		if value == ' ' {
			x += extraPerSpace
			if extraRemainder != nil && *extraRemainder > 0 {
				x++
				*extraRemainder--
			}
		}
	}
	return x
}

func textHeightFitsAtScale(element slideElement, bounds image.Rectangle, scale int, maxLines int, dpi int) bool {
	candidate := element
	candidate.FontScalePct = scale
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return false
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return false
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return false
	}
	if maxLines > 0 && len(lines) > maxLines {
		return false
	}
	measured, err := measureTextRenderLines(faces, lines, candidate.FontSize)
	if err != nil {
		return false
	}
	return measuredTextHeight(measured) <= bounds.Dy()
}

func textFitsAtScale(element slideElement, bounds image.Rectangle, scale int, maxLines int, dpi int) bool {
	candidate := element
	candidate.FontScalePct = scale
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return false
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return false
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return false
	}
	if maxLines > 0 && len(lines) > maxLines {
		return false
	}
	measured, err := measureTextRenderLines(faces, lines, candidate.FontSize)
	if err != nil {
		return false
	}
	if measuredTextHeight(measured) > bounds.Dy() {
		return false
	}
	width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
	if err != nil {
		return false
	}
	return width <= bounds.Dx()
}

func scaledTextElement(element slideElement, dpiOverride ...int) slideElement {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	applyFontScale := element.FontScalePct > 0 && element.FontScalePct != 100000
	applyLineSpacingReduction := element.LineSpacingReductionPct > 0 && element.FontScalePct > 0 && element.FontScalePct != 100000
	scaleForDPI := dpi != defaultOutputDPI
	if !scaleForDPI && !applyFontScale && !applyLineSpacingReduction {
		return element
	}
	element.TextParagraphs = cloneTextParagraphs(element.TextParagraphs)
	if scaleForDPI {
		for paragraphIndex := range element.TextParagraphs {
			element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceBefore, dpi)
			element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceAfter, dpi)
		}
	}
	if applyLineSpacingReduction {
		for paragraphIndex := range element.TextParagraphs {
			element.TextParagraphs[paragraphIndex].LineSpacingPct = reducedLineSpacing(element.TextParagraphs[paragraphIndex].LineSpacingPct, element.LineSpacingReductionPct)
		}
	}
	if !applyFontScale {
		return element
	}
	element.FontSize = scaledFontSize(element.FontSize, element.FontScalePct)
	for paragraphIndex := range element.TextParagraphs {
		element.TextParagraphs[paragraphIndex].FontSize = scaledFontSize(element.TextParagraphs[paragraphIndex].FontSize, element.FontScalePct)
		element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixels(element.TextParagraphs[paragraphIndex].SpaceBefore, element.FontScalePct)
		element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixels(element.TextParagraphs[paragraphIndex].SpaceAfter, element.FontScalePct)
		for runIndex := range element.TextParagraphs[paragraphIndex].Runs {
			element.TextParagraphs[paragraphIndex].Runs[runIndex].FontSize = scaledFontSize(element.TextParagraphs[paragraphIndex].Runs[runIndex].FontSize, element.FontScalePct)
		}
	}
	return element
}

func cloneTextParagraphs(paragraphs []textParagraph) []textParagraph {
	if len(paragraphs) == 0 {
		return nil
	}
	cloned := make([]textParagraph, len(paragraphs))
	copy(cloned, paragraphs)
	for index := range cloned {
		if len(cloned[index].Runs) > 0 {
			cloned[index].Runs = append([]textRun(nil), cloned[index].Runs...)
		}
		if len(cloned[index].TabStops) > 0 {
			cloned[index].TabStops = append([]int64(nil), cloned[index].TabStops...)
		}
	}
	return cloned
}

func scaleParagraphSpacingForDPI(element slideElement, dpi int) slideElement {
	element.TextParagraphs = cloneTextParagraphs(element.TextParagraphs)
	for paragraphIndex := range element.TextParagraphs {
		element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceBefore, dpi)
		element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceAfter, dpi)
	}
	return element
}

func scalePixelsForDPI(value int, dpi int) int {
	if value == 0 || dpi == defaultOutputDPI {
		return value
	}
	return int(math.Round(float64(value) * float64(dpi) / defaultOutputDPI))
}

func scalePixels(value int, scalePct int) int {
	if value == 0 || scalePct <= 0 || scalePct == 100000 {
		return value
	}
	return int(math.Round(float64(value) * float64(scalePct) / 100000))
}

func scaledFontSize(fontSize int, scalePct int) int {
	if fontSize <= 0 || scalePct <= 0 || scalePct == 100000 {
		return fontSize
	}
	scaled := int(math.Round(float64(fontSize) * float64(scalePct) / 100000))
	if scaled < 100 {
		return 100
	}
	return scaled
}

func reducedLineSpacing(lineSpacingPct int, reductionPct int) int {
	if reductionPct <= 0 || lineSpacingPct <= 0 {
		return lineSpacingPct
	}
	reduced := lineSpacingPct - reductionPct
	if reduced < 1000 {
		return 1000
	}
	return reduced
}

func alignedSegmentedTextX(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, bounds image.Rectangle, align string) (int, error) {
	return alignedSegmentedTextXAtDPI(faces, face, boldFace, segments, bounds, align, defaultOutputDPI, nil)
}

func alignedSegmentedTextXAtDPI(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, bounds image.Rectangle, align string, dpi int, tabStops []int) (int, error) {
	width, err := measureStyledSegmentsAtDPI(faces, face, boldFace, segments, dpi, tabStops)
	if err != nil {
		return 0, err
	}
	switch align {
	case "ctr":
		return bounds.Min.X + (bounds.Dx()-width)/2, nil
	case "r":
		return bounds.Max.X - width, nil
	default:
		return bounds.Min.X, nil
	}
}

func textRenderLinesForElement(faces *fontFaceCache, face font.Face, boldFace font.Face, element slideElement, maxWidth int, dpiOverride ...int) ([]textRenderLine, error) {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	if len(element.TextParagraphs) == 0 {
		lines := textLayoutPlainLines(face, element.Text, maxWidth, element.TextWrap)
		output := make([]textRenderLine, 0, len(lines))
		for _, line := range lines {
			output = append(output, textRenderLine{Text: line})
		}
		return output, nil
	}
	var output []textRenderLine
	pendingSpaceAfter := 0
	pendingSpaceAfterPct := 0
	for _, paragraph := range element.TextParagraphs {
		var paragraphLines []textRenderLine
		if len(paragraph.Runs) > 0 {
			lines, err := textRenderLinesForStyledParagraph(faces, face, boldFace, paragraph, maxWidth, element.TextWrap, dpi)
			if err != nil {
				return nil, err
			}
			paragraphLines = lines
		} else {
			paragraphFace := face
			if paragraph.FontSize != 0 {
				var err error
				paragraphFace, err = faces.Get(paragraph.FontSize, paragraph.Bold, paragraph.Italic)
				if err != nil {
					return nil, err
				}
			}
			for index, line := range layoutParagraphLines(paragraphFace, paragraph, maxWidth, element.TextWrap) {
				renderLine := textRenderLine{Text: line, Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize, TextAlign: paragraph.TextAlign, LineSpacingPct: paragraph.LineSpacingPct, TabStops: paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)}
				if index == 0 {
					renderLine.SpaceBefore = paragraph.SpaceBefore
					renderLine.SpaceBeforePct = paragraph.SpaceBeforePct
				}
				paragraphLines = append(paragraphLines, renderLine)
			}
		}
		if len(paragraphLines) == 0 {
			continue
		}
		if len(output) == 0 && !element.IncludeFirstLastSpacing {
			paragraphLines[0].SpaceBefore = 0
			paragraphLines[0].SpaceBeforePct = 0
		} else if len(output) > 0 {
			paragraphLines[0].SpaceBefore += pendingSpaceAfter
			paragraphLines[0].SpaceBeforePct += pendingSpaceAfterPct
		}
		output = append(output, paragraphLines...)
		pendingSpaceAfter = paragraph.SpaceAfter
		pendingSpaceAfterPct = paragraph.SpaceAfterPct
	}
	if len(output) > 0 && element.IncludeFirstLastSpacing {
		output[len(output)-1].SpaceAfter = pendingSpaceAfter
		output[len(output)-1].SpaceAfterPct = pendingSpaceAfterPct
	}
	return output, nil
}

func textRenderLinesForStyledParagraph(faces *fontFaceCache, face font.Face, boldFace font.Face, paragraph textParagraph, maxWidth int, wrap string, dpi int) ([]textRenderLine, error) {
	prefix, hangingPrefix := paragraphPrefixes(paragraph)
	firstOffset, hangingOffset, hasOffset := paragraphPixelOffsetsAtDPI(paragraph, dpi)
	rightOffset := paragraphRightOffsetAtDPI(paragraph, dpi)
	tabStops := paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)
	var output []textRenderLine
	chunks := splitRunsOnBreaks(paragraph.Runs)
	for index, chunk := range chunks {
		linePrefix := prefix
		lineOffset := firstOffset
		lineTabStops := tabStops
		if index > 0 {
			linePrefix = hangingPrefix
			lineOffset = hangingOffset
		} else if tabStop, ok := hangingBulletTabStop(paragraph, firstOffset, hangingOffset); ok {
			linePrefix = paragraph.Bullet + "\t"
			lineTabStops = withAdditionalTabStop(tabStops, tabStop)
		}
		if wrap == "none" {
			output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(linePrefix, paragraph, runsToSegments(chunk, paragraph)), lineTabStops), lineOffset, rightOffset, hasOffset))
			continue
		}
		if runsContainTabs(chunk) {
			output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(linePrefix, paragraph, runsToSegments(chunk, paragraph)), lineTabStops), lineOffset, rightOffset, hasOffset))
			continue
		}
		lines, err := wrapStyledRuns(faces, face, boldFace, chunk, paragraph, linePrefix, hangingPrefix, maxWidth, firstOffset, hangingOffset, rightOffset, hasOffset, dpi, lineTabStops)
		if err != nil {
			return nil, err
		}
		output = append(output, lines...)
	}
	if len(output) > 0 {
		output[0].SpaceBefore = paragraph.SpaceBefore
		output[0].SpaceBeforePct = paragraph.SpaceBeforePct
		for index := range output {
			output[index].TextAlign = paragraph.TextAlign
			output[index].LineSpacingPct = paragraph.LineSpacingPct
		}
	}
	return output, nil
}

func hangingBulletTabStop(paragraph textParagraph, firstOffset int, hangingOffset int) (int, bool) {
	if !paragraph.HasAutoNumber || paragraph.Bullet == "" || hangingOffset <= firstOffset {
		return 0, false
	}
	if !paragraph.HasMarginLeft && !paragraph.HasIndent {
		return 0, false
	}
	return hangingOffset - firstOffset, true
}

func withAdditionalTabStop(stops []int, stop int) []int {
	if stop <= 0 {
		return stops
	}
	for _, existing := range stops {
		if existing == stop {
			return stops
		}
	}
	output := append(append([]int{}, stops...), stop)
	sort.Ints(output)
	return output
}

func paragraphPixelOffsets(paragraph textParagraph) (int, int, bool) {
	return paragraphPixelOffsetsAtDPI(paragraph, defaultOutputDPI)
}

func paragraphPixelOffsetsAtDPI(paragraph textParagraph, dpi int) (int, int, bool) {
	if !paragraph.HasMarginLeft && !paragraph.HasIndent {
		return 0, 0, false
	}
	margin := paragraph.MarginLeft
	indent := paragraph.Indent
	first := emuToPixelsAtDPI(margin+indent, dpi)
	hanging := emuToPixelsAtDPI(margin, dpi)
	if first < 0 {
		first = 0
	}
	if hanging < 0 {
		hanging = 0
	}
	return first, hanging, true
}

func paragraphRightOffsetAtDPI(paragraph textParagraph, dpi int) int {
	if !paragraph.HasMarginRight {
		return 0
	}
	right := emuToPixelsAtDPI(paragraph.MarginRight, dpi)
	if right < 0 {
		return 0
	}
	return right
}

func tabStopsAtDPI(stops []int64, dpi int) []int {
	if len(stops) == 0 {
		return nil
	}
	output := make([]int, 0, len(stops))
	for _, stop := range stops {
		if stop <= 0 {
			continue
		}
		output = append(output, emuToPixelsAtDPI(stop, dpi))
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func paragraphTabStopsAtDPI(paragraph textParagraph, dpi int, maxWidth int) []int {
	stops := tabStopsAtDPI(paragraph.TabStops, dpi)
	if !paragraph.HasDefaultTab || paragraph.DefaultTabSize <= 0 {
		return stops
	}
	defaultTab := emuToPixelsAtDPI(paragraph.DefaultTabSize, dpi)
	if defaultTab <= 0 {
		return stops
	}
	limit := maxWidth * 2
	if limit < defaultTab {
		limit = defaultTab
	}
	generated := make([]int, 0, len(stops)+limit/defaultTab)
	generated = append(generated, stops...)
	for stop := defaultTab; stop <= limit; stop += defaultTab {
		generated = append(generated, stop)
	}
	if len(generated) == 0 {
		return nil
	}
	sort.Ints(generated)
	output := generated[:0]
	previous := -1
	for _, stop := range generated {
		if stop <= 0 || stop == previous {
			continue
		}
		output = append(output, stop)
		previous = stop
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func splitRunsOnBreaks(runs []textRun) [][]textRun {
	var chunks [][]textRun
	var current []textRun
	for _, run := range runs {
		parts := strings.Split(run.Text, "\n")
		for index, part := range parts {
			if index > 0 {
				chunks = append(chunks, current)
				current = nil
			}
			if part == "" {
				continue
			}
			next := run
			next.Text = part
			current = append(current, next)
		}
	}
	if len(current) > 0 || len(chunks) == 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func runsContainTabs(runs []textRun) bool {
	for _, run := range runs {
		if strings.Contains(run.Text, "\t") {
			return true
		}
	}
	return false
}

func runsToSegments(runs []textRun, paragraph textParagraph) []textLineSegment {
	segments := make([]textLineSegment, 0, len(runs))
	for _, run := range runs {
		segments = append(segments, runToSegment(run, paragraph))
	}
	return segments
}

func runToSegment(run textRun, paragraph textParagraph) textLineSegment {
	fontSize := run.FontSize
	if fontSize == 0 {
		fontSize = paragraph.FontSize
	}
	baselineFontSize := fontSize
	if run.Baseline != 0 {
		fontSize = scaledBaselineRunFontSize(fontSize)
	}
	segment := textLineSegment{
		Text:              run.Text,
		FontFamily:        paragraphFontFamily(run, paragraph),
		Bold:              resolvedRunBold(run, paragraph),
		Italic:            resolvedRunItalic(run, paragraph),
		Underline:         run.Underline,
		HasUnderlineColor: run.HasUnderlineColor,
		UnderlineColor:    run.UnderlineColor,
		Strike:            run.Strike,
		FontSize:          fontSize,
		Baseline:          run.Baseline,
		BaselineFontSize:  baselineFontSize,
		CharSpacing:       resolvedRunCharSpacing(run, paragraph),
		HasKern:           run.HasKern,
		KernMinFontSize:   run.KernMinFontSize,
		HasTextColor:      run.HasTextColor,
		TextColor:         run.TextColor,
		HasHighlightColor: run.HasHighlightColor,
		HighlightColor:    run.HighlightColor,
	}
	if !segment.HasTextColor && paragraph.HasTextColor {
		segment.HasTextColor = true
		segment.TextColor = paragraph.TextColor
	}
	return segment
}

func resolvedRunBold(run textRun, paragraph textParagraph) bool {
	if run.HasBold {
		return run.Bold
	}
	if run.Bold {
		return true
	}
	return paragraph.Bold
}

func resolvedRunItalic(run textRun, paragraph textParagraph) bool {
	if run.HasItalic {
		return run.Italic
	}
	if run.Italic {
		return true
	}
	return paragraph.Italic
}

func resolvedRunCharSpacing(run textRun, paragraph textParagraph) int {
	if run.HasCharSpacing {
		return run.CharSpacing
	}
	if paragraph.HasCharSpacing {
		return paragraph.CharSpacing
	}
	return 0
}

func paragraphFontFamily(run textRun, paragraph textParagraph) string {
	if strings.TrimSpace(run.FontFamily) != "" {
		return run.FontFamily
	}
	return paragraph.FontFamily
}

func scaledBaselineRunFontSize(fontSize int) int {
	if fontSize <= 0 {
		return fontSize
	}
	scaled := int(math.Round(float64(fontSize) * 0.65))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func appendPrefixSegment(prefix string, paragraph textParagraph, segments []textLineSegment) []textLineSegment {
	if prefix == "" {
		return segments
	}
	if paragraph.Bullet == "▶" && strings.HasPrefix(prefix, paragraph.Bullet) {
		bulletSegment := textLineSegment{Marker: "triangle", FontFamily: bulletSegmentFontFamily(paragraph), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
		if paragraph.HasBulletColor {
			bulletSegment.HasTextColor = true
			bulletSegment.TextColor = paragraph.BulletColor
		} else if paragraph.BulletColorTx {
			if textColor, ok := bulletTextColor(paragraph, segments); ok {
				bulletSegment.HasTextColor = true
				bulletSegment.TextColor = textColor
			}
		}
		spacer := textLineSegment{Text: strings.TrimPrefix(prefix, paragraph.Bullet), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize}
		output := make([]textLineSegment, 0, len(segments)+2)
		output = append(output, bulletSegment, spacer)
		output = append(output, segments...)
		return output
	}
	if paragraph.Bullet != "" && strings.HasPrefix(prefix, paragraph.Bullet) {
		bulletSegment := textLineSegment{Text: paragraph.Bullet, FontFamily: bulletSegmentFontFamily(paragraph), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
		if paragraph.HasBulletColor {
			bulletSegment.HasTextColor = true
			bulletSegment.TextColor = paragraph.BulletColor
		} else if paragraph.BulletColorTx {
			if textColor, ok := bulletTextColor(paragraph, segments); ok {
				bulletSegment.HasTextColor = true
				bulletSegment.TextColor = textColor
			}
		}
		spacer := textLineSegment{Text: strings.TrimPrefix(prefix, paragraph.Bullet), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize}
		output := make([]textLineSegment, 0, len(segments)+2)
		output = append(output, bulletSegment, spacer)
		output = append(output, segments...)
		return output
	}
	prefixSegment := textLineSegment{Text: prefix, Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
	output := make([]textLineSegment, 0, len(segments)+1)
	output = append(output, prefixSegment)
	output = append(output, segments...)
	return output
}

func bulletTextColor(paragraph textParagraph, segments []textLineSegment) (color.RGBA, bool) {
	for _, segment := range segments {
		if segment.HasTextColor {
			return segment.TextColor, true
		}
	}
	if paragraph.HasTextColor {
		return paragraph.TextColor, true
	}
	return color.RGBA{}, false
}

func bulletSegmentFontFamily(paragraph textParagraph) string {
	if paragraph.BulletFontTx {
		return paragraph.FontFamily
	}
	if symbolBulletRenderedAsUnicode(paragraph.Bullet, paragraph.BulletFontFamily) {
		return paragraph.FontFamily
	}
	if paragraph.BulletFontFamily != "" {
		return paragraph.BulletFontFamily
	}
	return paragraph.FontFamily
}

func symbolBulletRenderedAsUnicode(bullet string, fontFamily string) bool {
	font := strings.ToLower(strings.TrimSpace(fontFamily))
	if !strings.Contains(font, "wingdings") {
		return false
	}
	switch bullet {
	case "▪", "▶", "¬":
		return true
	default:
		return false
	}
}

func bulletSegmentFontSize(paragraph textParagraph) int {
	if paragraph.BulletFontSize > 0 {
		return paragraph.BulletFontSize
	}
	if paragraph.BulletSizePct > 0 && paragraph.FontSize > 0 {
		scaled := int(math.Round(float64(paragraph.FontSize) * float64(paragraph.BulletSizePct) / 100000))
		if scaled > 0 {
			return scaled
		}
	}
	return paragraph.FontSize
}

func textRenderLineFromSegments(segments []textLineSegment) textRenderLine {
	return textRenderLineFromSegmentsWithTabs(segments, nil)
}

func textRenderLineFromSegmentsWithTabs(segments []textLineSegment, tabStops []int) textRenderLine {
	var text strings.Builder
	for _, segment := range segments {
		text.WriteString(segment.Text)
	}
	return textRenderLine{Text: text.String(), Segments: segments, TabStops: tabStops}
}

func textRenderLineWithOffset(line textRenderLine, offset int, ok bool) textRenderLine {
	return textRenderLineWithOffsets(line, offset, 0, ok)
}

func textRenderLineWithOffsets(line textRenderLine, offset int, rightOffset int, ok bool) textRenderLine {
	if ok {
		line.XOffset = offset
		line.HasXOffset = true
	}
	line.RightOffset = rightOffset
	return line
}

func textLineBounds(bounds image.Rectangle, line textRenderLine) image.Rectangle {
	adjusted := bounds
	if line.HasXOffset {
		adjusted.Min.X += line.XOffset
	}
	if line.RightOffset > 0 {
		adjusted.Max.X -= line.RightOffset
	}
	if adjusted.Empty() {
		return bounds
	}
	return adjusted
}

func wrapStyledRuns(faces *fontFaceCache, face font.Face, boldFace font.Face, runs []textRun, paragraph textParagraph, firstPrefix string, hangingPrefix string, maxWidth int, firstOffset int, hangingOffset int, rightOffset int, hasOffset bool, dpi int, tabStopsOverride ...[]int) ([]textRenderLine, error) {
	fullLine := appendPrefixSegment(firstPrefix, paragraph, runsToSegments(runs, paragraph))
	tabStops := paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)
	if len(tabStopsOverride) > 0 {
		tabStops = tabStopsOverride[0]
	}
	fullLineWidth, err := measureStyledSegmentsAtDPI(faces, face, boldFace, fullLine, dpi, tabStops)
	if err != nil {
		return nil, err
	}
	availableWidth := maxWidth
	if hasOffset {
		availableWidth -= firstOffset
	}
	availableWidth -= rightOffset
	if maxWidth <= 0 || fullLineWidth <= availableWidth {
		return []textRenderLine{textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(fullLine, tabStops), firstOffset, rightOffset, hasOffset)}, nil
	}

	tokens := styledWordTokens(runs, paragraph)
	if maxWidth <= 0 || len(tokens) == 0 {
		return []textRenderLine{textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(firstPrefix, paragraph, runsToSegments(runs, paragraph)), tabStops), firstOffset, rightOffset, hasOffset)}, nil
	}
	var output []textRenderLine
	line := appendPrefixSegment(firstPrefix, paragraph, nil)
	lineOffset := firstOffset
	hasWord := false
	for _, token := range tokens {
		candidateToken := token.segmentWithPrefix(hasWord)
		candidate := append(append([]textLineSegment{}, line...), candidateToken)
		width, err := measureStyledSegmentsAtDPI(faces, face, boldFace, candidate, dpi, tabStops)
		if err != nil {
			return nil, err
		}
		availableWidth := maxWidth
		if hasOffset {
			availableWidth -= lineOffset
		}
		availableWidth -= rightOffset
		if hasWord && width > availableWidth {
			wrappedLine := textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(line, tabStops), lineOffset, rightOffset, hasOffset)
			wrappedLine.Justify = paragraph.TextAlign == "just"
			output = append(output, wrappedLine)
			line = appendPrefixSegment(hangingPrefix, paragraph, []textLineSegment{token.segmentAtLineStart()})
			lineOffset = hangingOffset
			hasWord = true
			continue
		}
		line = candidate
		hasWord = true
	}
	if len(line) > 0 {
		output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(line, tabStops), lineOffset, rightOffset, hasOffset))
	}
	return output, nil
}

type styledWordToken struct {
	Segment            textLineSegment
	Prefix             string
	PreserveLinePrefix bool
}

func (token styledWordToken) segmentWithPrefix(hasPriorWord bool) textLineSegment {
	segment := token.Segment
	if hasPriorWord || token.PreserveLinePrefix {
		segment.Text = token.Prefix + segment.Text
	}
	return segment
}

func (token styledWordToken) segmentAtLineStart() textLineSegment {
	segment := token.Segment
	if token.PreserveLinePrefix {
		segment.Text = token.Prefix + segment.Text
	}
	return segment
}

func styledWordTokens(runs []textRun, paragraph textParagraph) []styledWordToken {
	var tokens []styledWordToken
	var pendingSpace strings.Builder
	seenToken := false
	for _, run := range runs {
		segment := runToSegment(run, paragraph)
		for _, part := range splitTextForWrapping(segment.Text) {
			if part.Space {
				pendingSpace.WriteString(part.Text)
				continue
			}
			tokenSegment := segment
			tokenSegment.Text = part.Text
			tokens = append(tokens, styledWordToken{
				Segment:            tokenSegment,
				Prefix:             pendingSpace.String(),
				PreserveLinePrefix: !seenToken,
			})
			pendingSpace.Reset()
			seenToken = true
		}
	}
	return tokens
}

type wrapTextPart struct {
	Text  string
	Space bool
}

func splitTextForWrapping(text string) []wrapTextPart {
	if text == "" {
		return nil
	}
	var parts []wrapTextPart
	start := 0
	inSpace := unicode.IsSpace(rune(text[0]))
	for index, value := range text {
		isSpace := unicode.IsSpace(value)
		if index > start && isSpace != inSpace {
			parts = append(parts, wrapTextPart{Text: text[start:index], Space: inSpace})
			start = index
			inSpace = isSpace
		}
	}
	parts = append(parts, wrapTextPart{Text: text[start:], Space: inSpace})
	return parts
}

func textLineSpaceCount(segments []textLineSegment) int {
	count := 0
	for _, segment := range segments {
		count += textSpaceCount(segment.Text)
	}
	return count
}

func textSpaceCount(text string) int {
	count := 0
	for _, value := range text {
		if value == ' ' {
			count++
		}
	}
	return count
}

func measureStyledSegments(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment) (int, error) {
	return measureStyledSegmentsAtDPI(faces, face, boldFace, segments, defaultOutputDPI)
}

func measureStyledSegmentsAtDPI(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, dpi int, tabStopsOverride ...[]int) (int, error) {
	var tabStops []int
	if len(tabStopsOverride) > 0 {
		tabStops = tabStopsOverride[0]
	}
	width := 0
	for _, segment := range segments {
		if segment.Marker != "" {
			width += markerSegmentWidthAtDPI(segment, 0, dpi)
			continue
		}
		segmentFace := face
		if segment.FontSize != 0 || segment.FontFamily != "" {
			var err error
			segmentFace, err = faces.GetForFamily(segment.FontFamily, segment.FontSize, segment.Bold, segment.Italic)
			if err != nil {
				return 0, err
			}
		} else if segment.Bold || segment.Italic {
			var err error
			segmentFace, err = faces.GetForFamily(segment.FontFamily, 0, segment.Bold, segment.Italic)
			if err != nil {
				return 0, err
			}
		}
		width = measureTextSegmentWithTabsAndSpacingAtDPI(faceWithSegmentKerning(segmentFace, segment), segment.Text, width, dpi, tabStops, segment.CharSpacing)
	}
	return width, nil
}

type noKerningFace struct {
	font.Face
}

func (face noKerningFace) Kern(r0 rune, r1 rune) fixed.Int26_6 {
	return 0
}

func faceWithSegmentKerning(face font.Face, segment textLineSegment) font.Face {
	if face == nil || !segment.HasKern {
		return face
	}
	if segment.KernMinFontSize <= 0 {
		return face
	}
	if segment.FontSize > 0 && segment.FontSize < segment.KernMinFontSize {
		return noKerningFace{Face: face}
	}
	return face
}

func measureTextSegmentWithTabs(face font.Face, text string, currentWidth int) int {
	return measureTextSegmentWithTabsAtDPI(face, text, currentWidth, defaultOutputDPI, nil)
}

func measureTextSegmentWithTabsAtDPI(face font.Face, text string, currentWidth int, dpi int, tabStopsOverride ...[]int) int {
	return measureTextSegmentWithTabsAndSpacingAtDPI(face, text, currentWidth, dpi, firstTabStopsOverride(tabStopsOverride), 0)
}

func measureTextSegmentWithTabsAndSpacingAtDPI(face font.Face, text string, currentWidth int, dpi int, tabStops []int, charSpacing int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	if !strings.Contains(text, "\t") {
		return currentWidth + measureString(face, text) + textCharacterSpacingAdvance(text, spacingPixels)
	}
	width := currentWidth
	parts := strings.Split(text, "\t")
	for index, part := range parts {
		width += measureString(face, part) + textCharacterSpacingAdvance(part, spacingPixels)
		if index < len(parts)-1 {
			width += textTabAdvanceAtDPI(width, dpi, tabStops)
		}
	}
	return width
}

func firstTabStopsOverride(overrides [][]int) []int {
	if len(overrides) == 0 {
		return nil
	}
	return overrides[0]
}

func textCharacterSpacingPixelsAtDPI(charSpacing int, dpi int) int {
	if charSpacing == 0 {
		return 0
	}
	return int(math.Round(float64(charSpacing) / 100 * float64(normalizeOutputDPI(dpi)) / 72))
}

func textCharacterSpacingAdvance(text string, spacingPixels int) int {
	if spacingPixels == 0 {
		return 0
	}
	count := utf8.RuneCountInString(text)
	if count <= 1 {
		return 0
	}
	return (count - 1) * spacingPixels
}

func textTabAdvance(currentWidth int) int {
	return textTabAdvanceAtDPI(currentWidth, defaultOutputDPI, nil)
}

func textTabAdvanceAtDPI(currentWidth int, dpi int, tabStops []int) int {
	for _, stop := range tabStops {
		if stop > currentWidth {
			return stop - currentWidth
		}
	}
	tabPixels := normalizeOutputDPI(dpi)
	remainder := currentWidth % tabPixels
	if remainder == 0 {
		return tabPixels
	}
	return tabPixels - remainder
}

func measuredTextRenderLinesWidth(faces *fontFaceCache, face font.Face, boldFace font.Face, lines []textRenderLine, dpiOverride ...int) (int, error) {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	maxWidth := 0
	for _, line := range lines {
		width := 0
		if len(line.Segments) > 0 {
			var err error
			width, err = measureStyledSegmentsAtDPI(faces, face, boldFace, line.Segments, dpi, line.TabStops)
			if err != nil {
				return 0, err
			}
		} else {
			lineFace := face
			if line.FontSize != 0 {
				var err error
				lineFace, err = faces.Get(line.FontSize, line.Bold, line.Italic)
				if err != nil {
					return 0, err
				}
			} else if line.Bold || line.Italic {
				var err error
				lineFace, err = faces.Get(0, line.Bold, line.Italic)
				if err != nil {
					return 0, err
				}
			}
			width = measureString(lineFace, line.Text)
		}
		if line.HasXOffset {
			width += line.XOffset
		}
		width += line.RightOffset
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth, nil
}

func segmentBaselineShift(segment textLineSegment, fallbackFontSize int) int {
	return segmentBaselineShiftAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func segmentBaselineShiftAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	if segment.Baseline == 0 {
		return 0
	}
	fontSize := segment.FontSize
	if segment.BaselineFontSize > 0 {
		fontSize = segment.BaselineFontSize
	}
	if fontSize == 0 {
		fontSize = fallbackFontSize
	}
	if fontSize <= 0 {
		fontSize = 1800
	}
	points := float64(fontSize) / 100
	pixels := points * float64(normalizeOutputDPI(dpi)) / 72
	return int(math.Round(pixels * float64(segment.Baseline) / 100000))
}

func markerPixelSize(segment textLineSegment, fallbackFontSize int) int {
	return markerPixelSizeAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func markerPixelSizeAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	fontSize := segment.FontSize
	if fontSize == 0 {
		fontSize = fallbackFontSize
	}
	if fontSize <= 0 {
		fontSize = 1800
	}
	size := int(math.Round(float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72 * 0.45))
	if size < 6 {
		return 6
	}
	return size
}

func markerSegmentWidth(segment textLineSegment, fallbackFontSize int) int {
	return markerSegmentWidthAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func markerSegmentWidthAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	spacer := int(math.Round(float64(normalizeOutputDPI(dpi)) * 10 / defaultOutputDPI))
	if spacer < 1 {
		spacer = 1
	}
	return markerPixelSizeAtDPI(segment, fallbackFontSize, dpi) + spacer
}

func drawTextMarker(img *image.RGBA, marker string, x int, centerY int, size int, c color.RGBA) {
	switch marker {
	case "triangle":
		bounds := image.Rect(x, centerY-size/2, x+size, centerY+size/2)
		drawPolygon(img, bounds, []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0.5}, {X: 0, Y: 1}}, c)
	}
}

type textRenderLine struct {
	Text           string
	Bold           bool
	Italic         bool
	FontSize       int
	TextAlign      string
	Segments       []textLineSegment
	HasXOffset     bool
	XOffset        int
	RightOffset    int
	SpaceBefore    int
	SpaceBeforePct int
	SpaceAfter     int
	SpaceAfterPct  int
	LineSpacingPct int
	TabStops       []int
	Justify        bool
}

type textLineSegment struct {
	Text              string
	Marker            string
	FontFamily        string
	Bold              bool
	Italic            bool
	Underline         bool
	HasUnderlineColor bool
	UnderlineColor    color.RGBA
	Strike            string
	FontSize          int
	Baseline          int
	BaselineFontSize  int
	CharSpacing       int
	HasKern           bool
	KernMinFontSize   int
	HasTextColor      bool
	TextColor         color.RGBA
	HasHighlightColor bool
	HighlightColor    color.RGBA
}

type measuredTextLine struct {
	Face           font.Face
	Ascent         int
	Descent        int
	Height         int
	HasText        bool
	SpaceBefore    int
	SpaceBeforePct int
	SpaceAfter     int
	SpaceAfterPct  int
	LineSpacingPct int
}

type fontKey struct {
	FontSize   int
	Bold       bool
	Italic     bool
	FontFamily string
}

type fontFaceCache struct {
	DefaultItalic bool
	FontFamily    string
	PointScale    float64
	DPI           int
	Faces         map[fontKey]font.Face
}

func newFontFaceCache(italic bool, fontFamily string, pointScale ...float64) *fontFaceCache {
	return newFontFaceCacheWithDPI(italic, fontFamily, defaultOutputDPI, pointScale...)
}

func newFontFaceCacheWithDPI(italic bool, fontFamily string, dpi int, pointScale ...float64) *fontFaceCache {
	scale := 0.0
	if len(pointScale) > 0 {
		scale = pointScale[0]
	}
	return &fontFaceCache{DefaultItalic: italic, FontFamily: fontFamily, PointScale: scale, DPI: normalizeOutputDPI(dpi), Faces: map[fontKey]font.Face{}}
}

func (cache *fontFaceCache) Get(fontSize int, bold bool, italicOverride ...bool) (font.Face, error) {
	return cache.GetForFamily("", fontSize, bold, italicOverride...)
}

func (cache *fontFaceCache) GetForFamily(fontFamily string, fontSize int, bold bool, italicOverride ...bool) (font.Face, error) {
	italic := cache.DefaultItalic
	if len(italicOverride) > 0 {
		italic = italicOverride[0]
	}
	resolvedFamily := strings.TrimSpace(fontFamily)
	if resolvedFamily == "" {
		resolvedFamily = cache.FontFamily
	}
	key := fontKey{FontSize: fontSize, Bold: bold, Italic: italic, FontFamily: resolvedFamily}
	if face, ok := cache.Faces[key]; ok {
		return face, nil
	}
	face, err := openFontFaceWithDPI(fontSize, bold, italic, cache.PointScale, resolvedFamily, cache.DPI)
	if err != nil {
		return nil, err
	}
	cache.Faces[key] = face
	return face, nil
}

func (cache *fontFaceCache) Close() {
	for _, face := range cache.Faces {
		face.Close()
	}
}

func measureTextRenderLines(faces *fontFaceCache, lines []textRenderLine, fallbackFontSize int) ([]measuredTextLine, error) {
	measured := make([]measuredTextLine, 0, len(lines))
	for _, line := range lines {
		if len(line.Segments) > 0 {
			current, err := measureSegmentedTextLine(faces, line.Segments, fallbackFontSize)
			if err != nil {
				return nil, err
			}
			current.SpaceBefore = line.SpaceBefore
			current.SpaceBeforePct = line.SpaceBeforePct
			current.SpaceAfter = line.SpaceAfter
			current.SpaceAfterPct = line.SpaceAfterPct
			current.LineSpacingPct = line.LineSpacingPct
			current.HasText = true
			fontSize := lineFontSize(line, fallbackFontSize)
			current.SpaceBefore += paragraphSpacingPercentPixelsAtDPI(current.SpaceBeforePct, fontSize, faces.DPI)
			current.SpaceAfter += paragraphSpacingPercentPixelsAtDPI(current.SpaceAfterPct, fontSize, faces.DPI)
			current.Height = visibleLineAdvance(applyLineSpacingAtDPI(current.Height, current.LineSpacingPct, fontSize, faces.DPI), current)
			measured = append(measured, current)
			continue
		}
		fontSize := line.FontSize
		if fontSize == 0 {
			fontSize = fallbackFontSize
		}
		face, err := faces.Get(fontSize, line.Bold, line.Italic)
		if err != nil {
			return nil, err
		}
		metrics := face.Metrics()
		height := defaultLineMetricHeight(metrics)
		measured = append(measured, measuredTextLine{
			Face:    face,
			Ascent:  metrics.Ascent.Ceil(),
			Descent: metrics.Descent.Ceil(),
			HasText: line.Text != "",
			Height: visibleLineAdvance(
				applyLineSpacingAtDPI(height, line.LineSpacingPct, fontSize, faces.DPI),
				measuredTextLine{Ascent: metrics.Ascent.Ceil(), Descent: metrics.Descent.Ceil()},
			),
			SpaceBefore:    line.SpaceBefore + paragraphSpacingPercentPixelsAtDPI(line.SpaceBeforePct, fontSize, faces.DPI),
			SpaceBeforePct: line.SpaceBeforePct,
			SpaceAfter:     line.SpaceAfter + paragraphSpacingPercentPixelsAtDPI(line.SpaceAfterPct, fontSize, faces.DPI),
			SpaceAfterPct:  line.SpaceAfterPct,
			LineSpacingPct: line.LineSpacingPct,
		})
	}
	return measured, nil
}

func applyLineSpacing(height int, pct int) int {
	return applyLineSpacingAtDPI(height, pct, 0, defaultOutputDPI)
}

func applyLineSpacingAtDPI(height int, pct int, fontSize int, dpi int) int {
	if pct <= 0 {
		return height
	}
	base := height
	if fontSize > 0 {
		base = int(math.Round(float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72))
		if base <= 0 {
			base = height
		}
	}
	scaled := int(math.Round(float64(base) * float64(pct) / 100000))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func visibleLineAdvance(height int, line measuredTextLine) int {
	minimum := line.Ascent + line.Descent
	if minimum > height {
		return minimum
	}
	return height
}

func lineFontSize(line textRenderLine, fallbackFontSize int) int {
	size := line.FontSize
	for _, segment := range line.Segments {
		segmentSize := segment.FontSize
		if segmentSize == 0 {
			segmentSize = fallbackFontSize
		}
		if segmentSize > size {
			size = segmentSize
		}
	}
	if size <= 0 {
		return fallbackFontSize
	}
	return size
}

func measureSegmentedTextLine(faces *fontFaceCache, segments []textLineSegment, fallbackFontSize int) (measuredTextLine, error) {
	var measured measuredTextLine
	for _, segment := range segments {
		fontSize := segment.FontSize
		if fontSize == 0 {
			fontSize = fallbackFontSize
		}
		face, err := faces.GetForFamily(segment.FontFamily, fontSize, segment.Bold, segment.Italic)
		if err != nil {
			return measuredTextLine{}, err
		}
		metrics := face.Metrics()
		ascent := metrics.Ascent.Ceil()
		descent := metrics.Descent.Ceil()
		height := defaultLineMetricHeight(metrics)
		if ascent > measured.Ascent {
			measured.Ascent = ascent
		}
		if descent > measured.Descent {
			measured.Descent = descent
		}
		if height > measured.Height {
			measured.Height = height
		}
		if measured.Face == nil {
			measured.Face = face
		}
	}
	return measured, nil
}

func defaultLineMetricHeight(metrics font.Metrics) int {
	height := metrics.Height.Ceil()
	drawable := metrics.Ascent.Ceil() + metrics.Descent.Ceil()
	if height > 0 {
		return height
	}
	return drawable
}

func measuredTextHeight(lines []measuredTextLine) int {
	advanceHeight := 0
	inkHeight := 0
	for index, line := range lines {
		if index == 0 {
			inkHeight += line.SpaceBefore + line.Ascent
		} else {
			inkHeight += line.SpaceBefore + line.Height
		}
		advanceHeight += line.SpaceBefore + line.Height + line.SpaceAfter
	}
	if len(lines) > 0 {
		inkHeight += lines[len(lines)-1].Descent + lines[len(lines)-1].SpaceAfter
	}
	if inkHeight > advanceHeight {
		return inkHeight
	}
	return advanceHeight
}

func measuredTextAnchorHeight(lines []measuredTextLine, anchor string) int {
	if anchor != "ctr" && anchor != "b" {
		return measuredTextHeight(lines)
	}
	height := 0
	for _, line := range lines {
		visible := line.Ascent + line.Descent
		if !line.HasText {
			visible = line.Height
		}
		if visible <= 0 {
			visible = line.Height
		}
		height += line.SpaceBefore + visible + line.SpaceAfter
	}
	return height
}

func paragraphSpacingPercentPixels(pct int, fontSize int) int {
	return paragraphSpacingPercentPixelsAtDPI(pct, fontSize, defaultOutputDPI)
}

func paragraphSpacingPercentPixelsAtDPI(pct int, fontSize int, dpi int) int {
	if pct <= 0 || fontSize <= 0 {
		return 0
	}
	textPixels := float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72
	return int(math.Round(textPixels * float64(pct) / 100000))
}

func anchoredTextTop(bounds image.Rectangle, totalHeight int, anchor string) int {
	top := bounds.Min.Y
	switch anchor {
	case "ctr":
		top = bounds.Min.Y + (bounds.Dy()-totalHeight)/2
	case "b":
		top = bounds.Max.Y - totalHeight
	}
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	return top
}

func anchorCenteredTextBounds(bounds image.Rectangle, textWidth int) image.Rectangle {
	if textWidth <= 0 || textWidth >= bounds.Dx() {
		return bounds
	}
	left := bounds.Min.X + (bounds.Dx()-textWidth)/2
	return image.Rect(left, bounds.Min.Y, left+textWidth, bounds.Max.Y)
}

func textLayoutLines(face font.Face, text string, maxWidth int, wrap string) []string {
	return textLayoutParagraphLines(face, nil, text, maxWidth, wrap)
}

func textLayoutParagraphLines(face font.Face, paragraphs []textParagraph, fallbackText string, maxWidth int, wrap string) []string {
	lines := textLayoutStyledParagraphLines(face, face, paragraphs, fallbackText, maxWidth, wrap)
	output := make([]string, 0, len(lines))
	for _, line := range lines {
		output = append(output, line.Text)
	}
	return output
}

func textLayoutStyledParagraphLines(face font.Face, boldFace font.Face, paragraphs []textParagraph, fallbackText string, maxWidth int, wrap string) []textRenderLine {
	if len(paragraphs) == 0 {
		lines := textLayoutPlainLines(face, fallbackText, maxWidth, wrap)
		output := make([]textRenderLine, 0, len(lines))
		for _, line := range lines {
			output = append(output, textRenderLine{Text: line})
		}
		return output
	}
	var output []textRenderLine
	for _, paragraph := range paragraphs {
		paragraphFace := face
		if paragraph.Bold {
			paragraphFace = boldFace
		}
		for _, line := range layoutParagraphLines(paragraphFace, paragraph, maxWidth, wrap) {
			output = append(output, textRenderLine{Text: line, Bold: paragraph.Bold, FontSize: paragraph.FontSize, TextAlign: paragraph.TextAlign, LineSpacingPct: paragraph.LineSpacingPct})
		}
	}
	return output
}

func textLayoutPlainLines(face font.Face, text string, maxWidth int, wrap string) []string {
	var output []string
	for _, paragraph := range strings.Split(text, "\n") {
		if wrap == "none" {
			output = append(output, paragraph)
			continue
		}
		output = append(output, wrapText(face, paragraph, maxWidth)...)
	}
	return output
}

func layoutParagraphLines(face font.Face, paragraph textParagraph, maxWidth int, wrap string) []string {
	prefix, hangingPrefix := paragraphPrefixes(paragraph)
	var output []string
	for index, text := range strings.Split(paragraph.Text, "\n") {
		firstPrefix := prefix
		if index > 0 {
			firstPrefix = hangingPrefix
		}
		if wrap == "none" {
			output = append(output, firstPrefix+text)
			continue
		}
		output = append(output, wrapTextWithPrefixes(face, text, maxWidth, firstPrefix, hangingPrefix)...)
	}
	return output
}

func paragraphPrefixes(paragraph textParagraph) (string, string) {
	if paragraph.HasMarginLeft || paragraph.HasIndent {
		if paragraph.Bullet == "" {
			return "", ""
		}
		return paragraph.Bullet + " ", ""
	}
	indent := strings.Repeat("  ", paragraph.Level)
	if paragraph.Bullet == "" {
		return indent, indent
	}
	prefix := indent + paragraph.Bullet + " "
	hangingPrefix := strings.Repeat(" ", len([]rune(prefix)))
	return prefix, hangingPrefix
}

func anchoredTextStartY(bounds image.Rectangle, lineCount int, lineHeight int, ascent int, anchor string) int {
	if lineCount <= 0 {
		return bounds.Min.Y + ascent
	}
	totalHeight := lineCount * lineHeight
	top := bounds.Min.Y
	switch anchor {
	case "ctr":
		top = bounds.Min.Y + (bounds.Dy()-totalHeight)/2
	case "b":
		top = bounds.Max.Y - totalHeight
	}
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	return top + ascent
}

func wrapText(face font.Face, text string, maxWidth int) []string {
	return wrapTextWithPrefixes(face, text, maxWidth, "", "")
}

func wrapTextWithPrefixes(face font.Face, text string, maxWidth int, firstPrefix string, hangingPrefix string) []string {
	if maxWidth <= 0 || strings.TrimSpace(text) == "" {
		return []string{firstPrefix + text}
	}
	if measureString(face, firstPrefix+text) <= maxWidth {
		return []string{firstPrefix + text}
	}
	words := plainWordTokens(text)
	if len(words) == 0 {
		return []string{firstPrefix + text}
	}
	var lines []string
	current := words[0].textAtLineStart()
	for _, word := range words[1:] {
		candidate := current + word.textWithPrefix(true)
		prefix := firstPrefix
		if len(lines) > 0 {
			prefix = hangingPrefix
		}
		if measureString(face, prefix+candidate) <= maxWidth {
			current = candidate
			continue
		}
		prefix = firstPrefix
		if len(lines) > 0 {
			prefix = hangingPrefix
		}
		lines = append(lines, prefix+current)
		current = word.textAtLineStart()
	}
	prefix := firstPrefix
	if len(lines) > 0 {
		prefix = hangingPrefix
	}
	lines = append(lines, prefix+current)
	return lines
}

type plainWordToken struct {
	Text               string
	Prefix             string
	PreserveLinePrefix bool
}

func (token plainWordToken) textWithPrefix(hasPriorWord bool) string {
	if hasPriorWord || token.PreserveLinePrefix {
		return token.Prefix + token.Text
	}
	return token.Text
}

func (token plainWordToken) textAtLineStart() string {
	if token.PreserveLinePrefix {
		return token.Prefix + token.Text
	}
	return token.Text
}

func plainWordTokens(text string) []plainWordToken {
	var tokens []plainWordToken
	var pendingSpace strings.Builder
	seenToken := false
	for _, part := range splitTextForWrapping(text) {
		if part.Space {
			pendingSpace.WriteString(part.Text)
			continue
		}
		tokens = append(tokens, plainWordToken{
			Text:               part.Text,
			Prefix:             pendingSpace.String(),
			PreserveLinePrefix: !seenToken,
		})
		pendingSpace.Reset()
		seenToken = true
	}
	return tokens
}

func alignedTextX(face font.Face, text string, bounds image.Rectangle, align string) int {
	width := measureString(face, text)
	switch align {
	case "ctr":
		return bounds.Min.X + (bounds.Dx()-width)/2
	case "r":
		return bounds.Max.X - width
	default:
		return bounds.Min.X
	}
}

func measureString(face font.Face, text string) int {
	return (&font.Drawer{Face: face}).MeasureString(text).Ceil()
}

func textBounds(bounds image.Rectangle, element slideElement, size slideSize, canvas image.Rectangle) image.Rectangle {
	if element.HasTextTransform && element.TextExtCX > 0 && element.TextExtCY > 0 {
		bounds = image.Rect(
			scaleEMU(element.TextOffX, size.CX, canvas.Dx()),
			scaleEMU(element.TextOffY, size.CY, canvas.Dy()),
			scaleEMU(element.TextOffX+element.TextExtCX, size.CX, canvas.Dx()),
			scaleEMU(element.TextOffY+element.TextExtCY, size.CY, canvas.Dy()),
		)
	}
	left := scaleEMU(defaultTextInsetXEMU, size.CX, canvas.Dx())
	top := scaleEMU(defaultTextInsetYEMU, size.CY, canvas.Dy())
	right := scaleEMU(defaultTextInsetXEMU, size.CX, canvas.Dx())
	bottom := scaleEMU(defaultTextInsetYEMU, size.CY, canvas.Dy())
	if element.HasInsets {
		left = scaleEMU(element.InsetLeft, size.CX, canvas.Dx())
		top = scaleEMU(element.InsetTop, size.CY, canvas.Dy())
		right = scaleEMU(element.InsetRight, size.CX, canvas.Dx())
		bottom = scaleEMU(element.InsetBottom, size.CY, canvas.Dy())
	}
	inset := image.Rect(bounds.Min.X+left, bounds.Min.Y+top, bounds.Max.X-right, bounds.Max.Y-bottom)
	if inset.Empty() {
		return bounds
	}
	return inset
}

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

type roundedCorners struct {
	TopLeft     bool
	TopRight    bool
	BottomLeft  bool
	BottomRight bool
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

type curvedArrowGuides struct {
	W      float64
	H      float64
	Th     float64
	WR     float64
	X3     float64
	X4     float64
	X5     float64
	X6     float64
	X7     float64
	X8     float64
	Y1     float64
	IX     float64
	IY     float64
	SwAng  float64
	MSwAng float64
	StAng  float64
	StAng2 float64
	SwAng2 float64
	SwAng3 float64
	StAng3 float64
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
	rect = alignedStrokeRect(rect, width, align)
	if dash == "" {
		drawRectOutline(img, rect, c, width)
		return
	}
	if rect.Empty() {
		return
	}
	for i := 0; i < width; i++ {
		drawStyledLineWithPatternWidth(img, rect.Min.X, rect.Min.Y+i, rect.Max.X-1, rect.Min.Y+i, c, 1, dash, "flat", width)
		drawStyledLineWithPatternWidth(img, rect.Min.X, rect.Max.Y-1-i, rect.Max.X-1, rect.Max.Y-1-i, c, 1, dash, "flat", width)
		drawStyledLineWithPatternWidth(img, rect.Min.X+i, rect.Min.Y, rect.Min.X+i, rect.Max.Y-1, c, 1, dash, "flat", width)
		drawStyledLineWithPatternWidth(img, rect.Max.X-1-i, rect.Min.Y, rect.Max.X-1-i, rect.Max.Y-1, c, 1, dash, "flat", width)
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

func drawStyledLineWithPatternWidth(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, cap string, patternWidth int) {
	if cap == "" || cap == "sq" {
		if dash == "" {
			drawLine(img, x0, y0, x1, y1, c, width)
			return
		}
		drawDashedLineLegacyWithPatternWidth(img, x0, y0, x1, y1, c, width, dash, patternWidth)
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
	length := math.Hypot(float64(dirX), float64(dirY))
	if length == 0 {
		return
	}
	unitX := float64(dirX) / length
	unitY := float64(dirY) / length
	drawLength := math.Max(float64(lineWidth)*lineEndLengthFactor(markerLength), 8)
	markerHalfWidth := math.Max(float64(lineWidth)*lineEndWidthFactor(markerWidth), 4)
	baseX := float64(tipX) - unitX*drawLength
	baseY := float64(tipY) - unitY*drawLength
	perpX := -unitY
	perpY := unitX
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

func textLayoutUnsupportedMessages(element slideElement) []string {
	return textLayoutUnsupportedMessagesForTarget(element, image.Rectangle{}, defaultOutputDPI)
}

func textLayoutUnsupportedMessagesForTarget(element slideElement, bounds image.Rectangle, dpi int) []string {
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
	if normalAutofitRequiresSimplifiedSizing(element, bounds, dpi) {
		messages = append(messages, "normal autofit was rendered with simplified sizing")
	}
	if !shapeAutofitLayoutSupported(element) {
		messages = append(messages, "shape autofit was rendered with simplified sizing")
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
	source, err := resolveFontSource(fontFamily, bold, italic)
	if err != nil {
		return nil, errors.New("no supported font found")
	}
	parsed, err := parseFontData(source.Data, bold, italic)
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

type fontSource struct {
	Data  []byte
	Label string
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
	if styleSuffix != "" {
		names = append(names, family+".ttf", family+".otf")
	}
	roots := []string{
		"/Library/Fonts",
		"/System/Library/Fonts/Supplemental",
		filepath.Join(os.Getenv("HOME"), "Library", "Fonts"),
	}
	var paths []string
	for _, root := range roots {
		for _, name := range names {
			paths = append(paths, filepath.Join(root, name))
		}
	}
	return paths
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
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func renderPicture(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	if element.EmbedID == "" {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q has no embedded image relationship", elementLabel(*element)))}
	}
	relationship, ok := relationships[element.EmbedID]
	if !ok {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q references missing relationship %q", elementLabel(*element), element.EmbedID))}
	}
	if relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported relationship %q", elementLabel(*element), relationship.Type))}
	}
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q has no renderable transform", elementLabel(*element)))}
	}

	source, targetPart, partialUnsupported := pictureSourceImage(pkg, slidePart, element, relationships, relationship)
	if source == nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported image data %q: %v", elementLabel(*element), targetPart, partialUnsupported))}
	}

	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	)
	var unsupported []model.SkipItem
	if element.HasShadow {
		for _, message := range shadowTransformUnsupportedMessages(*element) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
		}
		if drawPictureShadow(img, target, *element, size) {
			// Supported picture shadows are painted before the image so the image occludes the inner shadow area.
		} else if element.ShadowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q outer shadow geometry was not rendered", elementLabel(*element))))
		}
	}
	for _, message := range shape3DUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
	}
	pictureImage, pictureBounds := pictureSourceForElement(source, *element)
	softEdgeRendered := drawPictureRaster(img, target, pictureImage, pictureBounds, *element, size)
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		if normalizedRotationDegrees(element.Rotation) == 0 {
			drawPictureOutline(img, target, *element, lineWidth)
		}
	}
	element.Rendered = true

	if len(element.CustomPath) >= 3 && len(element.CustomPathUnsupported) > 0 {
		for _, message := range element.CustomPathUnsupported {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", elementLabel(*element), message)))
		}
	}
	if element.HasSoftEdge {
		if !softEdgeRendered {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q soft edge was not rendered", elementLabel(*element))))
		}
	}
	if partialUnsupported != nil {
		if strings.EqualFold(path.Ext(targetPart), ".svg") {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from SVG because fallback raster could not be decoded: %v", elementLabel(*element), partialUnsupported)))
		} else {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from fallback raster because SVG image could not be decoded: %v", elementLabel(*element), partialUnsupported)))
		}
	}
	return unsupported
}

func drawPictureRaster(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize) bool {
	rotation := normalizedRotationDegrees(element.Rotation)
	if !pictureRotatesWithShape(element) {
		rotation = 0
	}
	if rotation == 0 {
		return drawPictureRasterLayer(img, target, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	}
	if target.Empty() {
		return false
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	layerTarget := layer.Bounds()
	softEdgeRendered := drawPictureRasterLayer(layer, layerTarget, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		drawPictureOutline(layer, layerTarget, element, lineWidth)
	}
	rotated := rotateRGBA(layer, rotation)
	center := image.Point{X: target.Min.X + target.Dx()/2, Y: target.Min.Y + target.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
	return softEdgeRendered
}

func pictureRotatesWithShape(element slideElement) bool {
	return !element.HasBlipRotWithShape || element.BlipRotWithShape
}

func drawPictureRasterLayer(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize, outputWidth int) bool {
	if element.HasSoftEdge && len(element.CustomPath) < 3 {
		scaleImageWithSoftEdge(img, target, pictureImage, pictureBounds, softEdgeRadiusPixels(element, size, outputWidth))
		return true
	}
	if len(element.CustomPath) >= 3 {
		scaleImageWithCustomMask(img, target, pictureImage, pictureBounds, element.CustomPath, element.CustomPathCommands)
		return false
	}
	scaleImage(img, target, pictureImage, pictureBounds)
	return false
}

func drawPictureOutline(img *image.RGBA, target image.Rectangle, element slideElement, lineWidth int) {
	drawStyledRectOutlineAligned(img, target, element.LineColor, lineWidth, element.LineDash, element.LineAlign)
}

func drawPictureShadow(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) bool {
	if element.ShadowColor.A == 0 {
		return false
	}
	offset := shadowOffset(element, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := shadowBlurPixels(element, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	if len(element.CustomPath) >= 3 {
		drawSoftPolygon(img, shadowBounds, element.CustomPath, element.ShadowColor, blur)
	} else {
		drawSoftRect(img, shadowBounds, element.ShadowColor, blur)
	}
	return true
}

func pictureSourceImage(pkg *pptx.Package, slidePart string, element *slideElement, relationships map[string]pptx.Relationship, fallbackRelationship pptx.Relationship) (image.Image, string, error) {
	fallback, fallbackPart, fallbackErr := fallbackPictureSourceImage(pkg, slidePart, fallbackRelationship)
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
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fallbackErr
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, fallbackErr
	}
	return source, targetPart, fallbackErr
}

func fallbackPictureSourceImage(pkg *pptx.Package, slidePart string, relationship pptx.Relationship) (image.Image, string, error) {
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fmt.Errorf("missing image part")
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, err
	}
	return source, targetPart, nil
}

func pictureUnsupported(slidePart string, element *slideElement, message string) model.SkipItem {
	element.UnsupportedNote = message
	return unsupportedItem(slidePart, unsupportedCode, message)
}

func decodeImage(partName string, contentType string, data []byte) (image.Image, error) {
	extension := strings.ToLower(path.Ext(partName))
	switch {
	case contentType == "image/png" || extension == ".png":
		return decodePNGImage(data)
	case contentType == "image/jpeg" || contentType == "image/jpg" || extension == ".jpg" || extension == ".jpeg":
		return decodeJPEGImage(data)
	case contentType == "image/gif" || extension == ".gif":
		return gif.Decode(bytes.NewReader(data))
	case contentType == "image/svg+xml" || extension == ".svg":
		return decodeSVGImage(data)
	default:
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}
}

func decodePNGImage(data []byte) (image.Image, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	profileData, ok := pngICCProfile(data)
	if !ok {
		return source, nil
	}
	profile, ok := parseICCRGBToSRGBProfile(profileData)
	if !ok {
		return source, nil
	}
	return convertICCImageToSRGB(source, profile), nil
}

func decodeJPEGImage(data []byte) (image.Image, error) {
	source, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if profileData, ok := jpegICCProfile(data); ok {
		if bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")) {
			return convertAdobeRGBImageToSRGB(source), nil
		}
		if profile, ok := parseICCRGBToSRGBProfile(profileData); ok {
			return convertICCImageToSRGB(source, profile), nil
		}
	}
	return source, nil
}

func jpegHasAdobeRGBProfile(data []byte) bool {
	profileData, ok := jpegICCProfile(data)
	return ok && (bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")))
}

func jpegICCProfile(data []byte) ([]byte, bool) {
	const markerPrefix = "ICC_PROFILE\x00"
	chunks := map[int][]byte{}
	totalChunks := 0
	for offset := 0; offset+4 <= len(data); {
		if data[offset] != 0xFF {
			offset++
			continue
		}
		for offset < len(data) && data[offset] == 0xFF {
			offset++
		}
		if offset >= len(data) {
			break
		}
		marker := data[offset]
		offset++
		if marker == 0xDA || marker == 0xD9 {
			break
		}
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if offset+2 > len(data) {
			return nil, false
		}
		length := int(data[offset])<<8 | int(data[offset+1])
		offset += 2
		if length < 2 || offset+length-2 > len(data) {
			return nil, false
		}
		segment := data[offset : offset+length-2]
		offset += length - 2
		if marker != 0xE2 || !bytes.HasPrefix(segment, []byte(markerPrefix)) {
			continue
		}
		if len(segment) < len(markerPrefix)+2 {
			return nil, false
		}
		sequenceNumber := int(segment[len(markerPrefix)])
		sequenceTotal := int(segment[len(markerPrefix)+1])
		if sequenceNumber == 0 || sequenceTotal == 0 || sequenceNumber > sequenceTotal {
			return nil, false
		}
		if totalChunks == 0 {
			totalChunks = sequenceTotal
		} else if totalChunks != sequenceTotal {
			return nil, false
		}
		if _, exists := chunks[sequenceNumber]; exists {
			return nil, false
		}
		chunks[sequenceNumber] = segment[len(markerPrefix)+2:]
	}
	if totalChunks == 0 || len(chunks) != totalChunks {
		return nil, false
	}
	var profile []byte
	for index := 1; index <= totalChunks; index++ {
		chunk, ok := chunks[index]
		if !ok {
			return nil, false
		}
		profile = append(profile, chunk...)
	}
	return profile, true
}

func pngICCProfile(data []byte) ([]byte, bool) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return nil, false
	}
	offset := 8
	for offset+8 <= len(data) {
		length := int(readUint32BE(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length+4 > len(data) {
			return nil, false
		}
		chunk := data[offset : offset+length]
		offset += length + 4
		if chunkType == "IEND" {
			return nil, false
		}
		if chunkType != "iCCP" {
			continue
		}
		nameEnd := bytes.IndexByte(chunk, 0)
		if nameEnd < 0 || nameEnd+2 > len(chunk) || chunk[nameEnd+1] != 0 {
			return nil, false
		}
		reader, err := zlib.NewReader(bytes.NewReader(chunk[nameEnd+2:]))
		if err != nil {
			return nil, false
		}
		defer reader.Close()
		profile, err := io.ReadAll(reader)
		if err != nil {
			return nil, false
		}
		return profile, true
	}
	return nil, false
}

type iccRGBToSRGBProfile struct {
	rXYZ [3]float64
	gXYZ [3]float64
	bXYZ [3]float64
	rTRC iccCurve
	gTRC iccCurve
	bTRC iccCurve
}

type iccCurve struct {
	gamma float64
	table []uint16
}

func parseICCRGBToSRGBProfile(data []byte) (iccRGBToSRGBProfile, bool) {
	if len(data) < 132 || string(data[16:20]) != "RGB " || string(data[20:24]) != "XYZ " {
		return iccRGBToSRGBProfile{}, false
	}
	tagCount := int(readUint32BE(data[128:132]))
	if tagCount < 0 || 132+tagCount*12 > len(data) {
		return iccRGBToSRGBProfile{}, false
	}
	tags := map[string][]byte{}
	for index := 0; index < tagCount; index++ {
		entry := 132 + index*12
		signature := string(data[entry : entry+4])
		offset := int(readUint32BE(data[entry+4 : entry+8]))
		size := int(readUint32BE(data[entry+8 : entry+12]))
		if offset < 0 || size < 0 || offset+size > len(data) {
			continue
		}
		tags[signature] = data[offset : offset+size]
	}
	profile := iccRGBToSRGBProfile{}
	var ok bool
	if profile.rXYZ, ok = parseICCXYZTag(tags["rXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gXYZ, ok = parseICCXYZTag(tags["gXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bXYZ, ok = parseICCXYZTag(tags["bXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.rTRC, ok = parseICCCurveTag(tags["rTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gTRC, ok = parseICCCurveTag(tags["gTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bTRC, ok = parseICCCurveTag(tags["bTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	return profile, true
}

func parseICCXYZTag(data []byte) ([3]float64, bool) {
	if len(data) < 20 || string(data[:4]) != "XYZ " {
		return [3]float64{}, false
	}
	return [3]float64{
		s15Fixed16(data[8:12]),
		s15Fixed16(data[12:16]),
		s15Fixed16(data[16:20]),
	}, true
}

func parseICCCurveTag(data []byte) (iccCurve, bool) {
	if len(data) < 12 || string(data[:4]) != "curv" {
		return iccCurve{}, false
	}
	count := int(readUint32BE(data[8:12]))
	if count == 0 {
		return iccCurve{gamma: 1}, true
	}
	if len(data) < 12+count*2 {
		return iccCurve{}, false
	}
	if count == 1 {
		return iccCurve{gamma: float64(readUint16BE(data[12:14])) / 256}, true
	}
	table := make([]uint16, count)
	for index := range table {
		table[index] = readUint16BE(data[12+index*2 : 14+index*2])
	}
	return iccCurve{table: table}, true
}

func convertICCImageToSRGB(source image.Image, profile iccRGBToSRGBProfile) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			r, g, b := profile.iccRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: pixel.A})
		}
	}
	return dst
}

func (profile iccRGBToSRGBProfile) iccRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := profile.rTRC.linearize(r)
	linearG := profile.gTRC.linearize(g)
	linearB := profile.bTRC.linearize(b)

	xD50 := profile.rXYZ[0]*linearR + profile.gXYZ[0]*linearG + profile.bXYZ[0]*linearB
	yD50 := profile.rXYZ[1]*linearR + profile.gXYZ[1]*linearG + profile.bXYZ[1]*linearB
	zD50 := profile.rXYZ[2]*linearR + profile.gXYZ[2]*linearG + profile.bXYZ[2]*linearB

	// ICC matrix profiles encode PCS XYZ relative to D50. Adapt to D65 before
	// applying the sRGB output matrix.
	xD65 := 0.9555766*xD50 - 0.0230393*yD50 + 0.0631636*zD50
	yD65 := -0.0282895*xD50 + 1.0099416*yD50 + 0.0210077*zD50
	zD65 := 0.0122982*xD50 - 0.0204830*yD50 + 1.3299098*zD50

	srgbR := 3.2404542*xD65 - 1.5371385*yD65 - 0.4985314*zD65
	srgbG := -0.9692660*xD65 + 1.8760108*yD65 + 0.0415560*zD65
	srgbB := 0.0556434*xD65 - 0.2040259*yD65 + 1.0572252*zD65
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func (curve iccCurve) linearize(value uint8) float64 {
	encoded := float64(value) / 255
	if len(curve.table) == 0 {
		gamma := curve.gamma
		if gamma == 0 {
			gamma = 1
		}
		return math.Pow(encoded, gamma)
	}
	position := encoded * float64(len(curve.table)-1)
	index := int(math.Floor(position))
	if index >= len(curve.table)-1 {
		return float64(curve.table[len(curve.table)-1]) / 65535
	}
	fraction := position - float64(index)
	a := float64(curve.table[index]) / 65535
	b := float64(curve.table[index+1]) / 65535
	return a + (b-a)*fraction
}

func readUint32BE(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

func readUint16BE(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}

func s15Fixed16(data []byte) float64 {
	if len(data) < 4 {
		return 0
	}
	value := int32(readUint32BE(data))
	return float64(value) / 65536
}

func convertAdobeRGBImageToSRGB(source image.Image) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			pixel.R, pixel.G, pixel.B = adobeRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, pixel)
		}
	}
	return dst
}

func adobeRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := adobeRGBByteToLinear(r)
	linearG := adobeRGBByteToLinear(g)
	linearB := adobeRGBByteToLinear(b)

	x := 0.5767309*linearR + 0.1855540*linearG + 0.1881852*linearB
	y := 0.2973769*linearR + 0.6273491*linearG + 0.0752741*linearB
	z := 0.0270343*linearR + 0.0706872*linearG + 0.9911085*linearB

	srgbR := 3.2404542*x - 1.5371385*y - 0.4985314*z
	srgbG := -0.9692660*x + 1.8760108*y + 0.0415560*z
	srgbB := 0.0556434*x - 0.2040259*y + 1.0572252*z
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func adobeRGBByteToLinear(value uint8) float64 {
	if value == 0 {
		return 0
	}
	return math.Pow(float64(value)/255, 2.19921875)
}

type svgViewBox struct {
	MinX   float64
	MinY   float64
	Width  float64
	Height float64
}

type svgPaintStyle struct {
	Fill        color.RGBA
	HasFill     bool
	NoFill      bool
	FillOpacity float64
	HasOpacity  bool
}

func decodeSVGImage(data []byte) (image.Image, error) {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil, err
	}
	if root.Name != "svg" {
		return nil, fmt.Errorf("expected svg root, got %q", root.Name)
	}
	viewBox, err := parseSVGViewBox(root)
	if err != nil {
		return nil, err
	}
	width := svgRasterDimension(viewBox.Width)
	height := svgRasterDimension(viewBox.Height)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := drawSVGNode(img, img.Bounds(), viewBox, root, parseSVGStyleRules(root), svgPaintStyle{}); err != nil {
		return nil, err
	}
	return img, nil
}

func parseSVGViewBox(root *xmlNode) (svgViewBox, error) {
	raw := attrValue(root.Attrs, "viewBox")
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(values) == 4 {
		var parsed [4]float64
		for index, value := range values {
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return svgViewBox{}, fmt.Errorf("invalid svg viewBox %q", raw)
			}
			parsed[index] = number
		}
		if parsed[2] > 0 && parsed[3] > 0 {
			return svgViewBox{MinX: parsed[0], MinY: parsed[1], Width: parsed[2], Height: parsed[3]}, nil
		}
	}
	width, widthOK := svgLengthAttr(root.Attrs, "width")
	height, heightOK := svgLengthAttr(root.Attrs, "height")
	if widthOK && heightOK && width > 0 && height > 0 {
		return svgViewBox{Width: width, Height: height}, nil
	}
	return svgViewBox{}, fmt.Errorf("svg viewBox is missing or invalid")
}

func svgLengthAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := attrValue(attrs, name)
	value = strings.TrimSuffix(strings.TrimSpace(value), "px")
	if value == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgRasterDimension(value float64) int {
	dimension := int(math.Round(value))
	if dimension < 1 {
		return 1
	}
	if dimension > 2048 {
		return 2048
	}
	return dimension
}

func drawSVGNode(img *image.RGBA, bounds image.Rectangle, viewBox svgViewBox, node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) error {
	inherited = resolveSVGPaintStyle(node, styles, inherited, true)
	for _, child := range node.Children {
		switch child.Name {
		case "g", "svg":
			if err := drawSVGNode(img, bounds, viewBox, child, styles, inherited); err != nil {
				return err
			}
		case "path":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			paths, err := parseSVGPath(attrValue(child.Attrs, "d"), viewBox)
			if err != nil {
				return err
			}
			for _, points := range paths {
				drawPolygon(img, bounds, points, c)
			}
		case "rect":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			rect, ok := svgRectBounds(child, bounds, viewBox)
			if ok {
				draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
			}
		case "circle", "ellipse":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			ellipse, ok := svgEllipseBounds(child, bounds, viewBox)
			if ok {
				drawEllipse(img, ellipse, c)
			}
		}
	}
	return nil
}

func svgNodeFill(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) (color.RGBA, bool) {
	style := resolveSVGPaintStyle(node, styles, inherited, true)
	if style.NoFill {
		return color.RGBA{}, false
	}
	if !style.HasFill {
		style.Fill = color.RGBA{A: 255}
	}
	if style.HasOpacity {
		opacity := style.FillOpacity
		if opacity < 0 {
			opacity = 0
		}
		if opacity > 1 {
			opacity = 1
		}
		style.Fill.A = uint8(math.Round(float64(style.Fill.A) * opacity))
	}
	return style.Fill, true
}

func resolveSVGPaintStyle(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle, includePresentationAttrs bool) svgPaintStyle {
	resolved := inherited
	if includePresentationAttrs {
		mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations("fill:"+attrValue(node.Attrs, "fill")+";fill-opacity:"+attrValue(node.Attrs, "fill-opacity")))
	}
	for _, className := range strings.Fields(attrValue(node.Attrs, "class")) {
		if style, ok := styles[className]; ok {
			mergeSVGPaintStyle(&resolved, style)
		}
	}
	mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations(attrValue(node.Attrs, "style")))
	return resolved
}

func mergeSVGPaintStyle(base *svgPaintStyle, override svgPaintStyle) {
	if override.HasFill || override.NoFill {
		base.Fill = override.Fill
		base.HasFill = override.HasFill
		base.NoFill = override.NoFill
	}
	if override.HasOpacity {
		base.FillOpacity = override.FillOpacity
		base.HasOpacity = true
	}
}

func parseSVGStyleRules(root *xmlNode) map[string]svgPaintStyle {
	styles := map[string]svgPaintStyle{}
	for _, node := range descendantsByName(root, "style") {
		for _, block := range strings.Split(node.Text, "}") {
			selectorText, declarationText, ok := strings.Cut(block, "{")
			if !ok {
				continue
			}
			style := parseSVGPaintDeclarations(declarationText)
			if !style.HasFill && !style.NoFill && !style.HasOpacity {
				continue
			}
			for _, selector := range strings.Split(selectorText, ",") {
				selector = strings.TrimSpace(selector)
				if !strings.HasPrefix(selector, ".") {
					continue
				}
				className := strings.TrimSpace(strings.TrimPrefix(selector, "."))
				if className != "" {
					styles[className] = style
				}
			}
		}
	}
	return styles
}

func parseSVGPaintDeclarations(raw string) svgPaintStyle {
	var style svgPaintStyle
	for _, declaration := range strings.Split(raw, ";") {
		name, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		switch name {
		case "fill":
			c, hasFill, noFill := parseSVGFillValue(value)
			style.Fill = c
			style.HasFill = hasFill
			style.NoFill = noFill
		case "fill-opacity":
			if opacity, ok := parseSVGOpacity(value); ok {
				style.FillOpacity = opacity
				style.HasOpacity = true
			}
		}
	}
	return style
}

func parseSVGFillValue(raw string) (color.RGBA, bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return color.RGBA{}, false, false
	}
	if strings.EqualFold(raw, "none") {
		return color.RGBA{}, false, true
	}
	var c color.RGBA
	var ok bool
	switch strings.ToLower(raw) {
	case "black":
		c, ok = color.RGBA{A: 255}, true
	case "white":
		c, ok = color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	default:
		c, ok = parseHexColor(raw)
	}
	if !ok {
		return color.RGBA{}, false, false
	}
	return c, true, false
}

func parseSVGOpacity(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return value, err == nil
}

func svgRectBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	x, xOK := svgFloatAttr(node.Attrs, "x")
	y, yOK := svgFloatAttr(node.Attrs, "y")
	width, widthOK := svgFloatAttr(node.Attrs, "width")
	height, heightOK := svgFloatAttr(node.Attrs, "height")
	if !xOK {
		x = 0
	}
	if !yOK {
		y = 0
	}
	if !widthOK || !heightOK || width <= 0 || height <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(x, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(x+width, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y+height, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgEllipseBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	cx, cxOK := svgFloatAttr(node.Attrs, "cx")
	cy, cyOK := svgFloatAttr(node.Attrs, "cy")
	if !cxOK || !cyOK {
		return image.Rectangle{}, false
	}
	rx, rxOK := svgFloatAttr(node.Attrs, "rx")
	ry, ryOK := svgFloatAttr(node.Attrs, "ry")
	if node.Name == "circle" {
		r, rOK := svgFloatAttr(node.Attrs, "r")
		rx, ry, rxOK, ryOK = r, r, rOK, rOK
	}
	if !rxOK || !ryOK || rx <= 0 || ry <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(cx-rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy-ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(cx+rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy+ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgFloatAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := strings.TrimSpace(attrValue(attrs, name))
	if value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgCoordToPixel(value float64, min float64, span float64, pixelMin int, pixelSpan int) int {
	if span == 0 {
		return pixelMin
	}
	return pixelMin + int(math.Round((value-min)/span*float64(pixelSpan)))
}

func svgPointToPathPoint(x float64, y float64, viewBox svgViewBox) pathPoint {
	return pathPoint{
		X: (x - viewBox.MinX) / viewBox.Width,
		Y: (y - viewBox.MinY) / viewBox.Height,
	}
}

type svgPathToken struct {
	Command  byte
	Number   float64
	IsNumber bool
}

func parseSVGPath(data string, viewBox svgViewBox) ([][]pathPoint, error) {
	tokens, err := tokenizeSVGPath(data)
	if err != nil {
		return nil, err
	}
	var paths [][]pathPoint
	var points []pathPoint
	var currentCommand byte
	var currentX float64
	var currentY float64
	var startX float64
	var startY float64
	index := 0
	for index < len(tokens) {
		if !tokens[index].IsNumber {
			currentCommand = tokens[index].Command
			index++
		} else if currentCommand == 0 {
			return nil, fmt.Errorf("svg path data starts with a number")
		}
		switch currentCommand {
		case 'M', 'm':
			first := true
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path move command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'm' {
					x += currentX
					y += currentY
				}
				if first {
					if len(points) >= 3 {
						paths = append(paths, points)
					}
					points = []pathPoint{svgPointToPathPoint(x, y, viewBox)}
					startX, startY = x, y
					first = false
				} else {
					points = append(points, svgPointToPathPoint(x, y, viewBox))
				}
				currentX, currentY = x, y
			}
		case 'L', 'l':
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path line command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'l' {
					x += currentX
					y += currentY
				}
				points = append(points, svgPointToPathPoint(x, y, viewBox))
				currentX, currentY = x, y
			}
		case 'H', 'h':
			for index < len(tokens) && tokens[index].IsNumber {
				x := tokens[index].Number
				index++
				if currentCommand == 'h' {
					x += currentX
				}
				points = append(points, svgPointToPathPoint(x, currentY, viewBox))
				currentX = x
			}
		case 'V', 'v':
			for index < len(tokens) && tokens[index].IsNumber {
				y := tokens[index].Number
				index++
				if currentCommand == 'v' {
					y += currentY
				}
				points = append(points, svgPointToPathPoint(currentX, y, viewBox))
				currentY = y
			}
		case 'C', 'c':
			for index < len(tokens) && tokens[index].IsNumber {
				values, next, ok := readSVGPathNumbers(tokens, index, 6)
				if !ok {
					return nil, fmt.Errorf("svg path cubic command has incomplete coordinates")
				}
				index = next
				x1, y1, x2, y2, x, y := values[0], values[1], values[2], values[3], values[4], values[5]
				if currentCommand == 'c' {
					x1 += currentX
					y1 += currentY
					x2 += currentX
					y2 += currentY
					x += currentX
					y += currentY
				}
				points = append(points, flattenSVGCubic(currentX, currentY, x1, y1, x2, y2, x, y, viewBox)...)
				currentX, currentY = x, y
			}
		case 'Z', 'z':
			if len(points) >= 3 {
				paths = append(paths, points)
			}
			points = nil
			currentX, currentY = startX, startY
			currentCommand = 0
		default:
			return nil, fmt.Errorf("unsupported svg path command %q", string(currentCommand))
		}
	}
	if len(points) >= 3 {
		paths = append(paths, points)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("svg path has no closed paintable subpaths")
	}
	return paths, nil
}

func tokenizeSVGPath(data string) ([]svgPathToken, error) {
	var tokens []svgPathToken
	for index := 0; index < len(data); {
		ch := data[index]
		switch {
		case isSVGPathSeparator(ch):
			index++
		case isSVGPathCommand(ch):
			tokens = append(tokens, svgPathToken{Command: ch})
			index++
		case isSVGPathNumberStart(ch):
			start := index
			index++
			for index < len(data) && isSVGPathNumberByte(data[index], data[index-1]) {
				index++
			}
			number, err := strconv.ParseFloat(data[start:index], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid svg path number %q", data[start:index])
			}
			tokens = append(tokens, svgPathToken{Number: number, IsNumber: true})
		default:
			return nil, fmt.Errorf("invalid svg path token %q", string(ch))
		}
	}
	return tokens, nil
}

func isSVGPathSeparator(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ','
}

func isSVGPathCommand(ch byte) bool {
	return strings.ContainsRune("MmLlHhVvCcZz", rune(ch))
}

func isSVGPathNumberStart(ch byte) bool {
	return (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' || ch == '.'
}

func isSVGPathNumberByte(ch byte, previous byte) bool {
	if ch >= '0' && ch <= '9' {
		return true
	}
	if ch == '.' {
		return true
	}
	if ch == '-' || ch == '+' {
		return previous == 'e' || previous == 'E'
	}
	return ch == 'e' || ch == 'E'
}

func readSVGPathPair(tokens []svgPathToken, index int) (float64, float64, int, bool) {
	values, next, ok := readSVGPathNumbers(tokens, index, 2)
	if !ok {
		return 0, 0, index, false
	}
	return values[0], values[1], next, true
}

func readSVGPathNumbers(tokens []svgPathToken, index int, count int) ([]float64, int, bool) {
	if index+count > len(tokens) {
		return nil, index, false
	}
	values := make([]float64, 0, count)
	for offset := 0; offset < count; offset++ {
		token := tokens[index+offset]
		if !token.IsNumber {
			return nil, index, false
		}
		values = append(values, token.Number)
	}
	return values, index + count, true
}

func flattenSVGCubic(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, viewBox svgViewBox) []pathPoint {
	const segments = 12
	points := make([]pathPoint, 0, segments)
	for step := 1; step <= segments; step++ {
		t := float64(step) / segments
		inv := 1 - t
		x := inv*inv*inv*x0 + 3*inv*inv*t*x1 + 3*inv*t*t*x2 + t*t*t*x3
		y := inv*inv*inv*y0 + 3*inv*inv*t*y1 + 3*inv*t*t*y2 + t*t*t*y3
		points = append(points, svgPointToPathPoint(x, y, viewBox))
	}
	return points
}

func scaleEMU(value int64, totalEMU int64, totalPixels int) int {
	if totalEMU == 0 {
		return 0
	}
	return int(math.Round(float64(value) / float64(totalEMU) * float64(totalPixels)))
}

func sourceCropRect(bounds image.Rectangle, element slideElement) image.Rectangle {
	if !element.HasCrop {
		return bounds
	}
	width := bounds.Dx()
	height := bounds.Dy()
	left := bounds.Min.X + cropPixels(width, element.CropLeft)
	top := bounds.Min.Y + cropPixels(height, element.CropTop)
	right := bounds.Max.X - cropPixels(width, element.CropRight)
	bottom := bounds.Max.Y - cropPixels(height, element.CropBottom)
	cropped := image.Rect(left, top, right, bottom)
	if cropped.Empty() || cropped.Intersect(bounds).Empty() {
		return bounds
	}
	return cropped
}

func cropPixels(total int, percentage int64) int {
	if percentage == 0 || total == 0 {
		return 0
	}
	return int(math.Round(float64(total) * float64(percentage) / 100000))
}

func pictureSourceForElement(src image.Image, element slideElement) (image.Image, image.Rectangle) {
	srcBounds := sourceCropRect(src.Bounds(), element)
	if !element.FlipH && !element.FlipV && !shouldApplyImageAlphaModFix(element) {
		return src, srcBounds
	}
	return transformedPictureImage(src, srcBounds, element), image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}

func shouldApplyImageAlphaModFix(element slideElement) bool {
	return element.HasImageAlphaModFix && element.ImageAlphaModFixPct > 0 && element.ImageAlphaModFixPct != 100000
}

func transformedPictureImage(src image.Image, srcBounds image.Rectangle, element slideElement) *image.RGBA {
	width := srcBounds.Dx()
	height := srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y
		if element.FlipV {
			srcY = srcBounds.Max.Y - 1 - y
		}
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x
			if element.FlipH {
				srcX = srcBounds.Max.X - 1 - x
			}
			pixel := color.RGBAModel.Convert(src.At(srcX, srcY)).(color.RGBA)
			pixel = applyImageAlphaModFix(pixel, element)
			dst.SetRGBA(x, y, pixel)
		}
	}
	return dst
}

func applyImageAlphaModFix(c color.RGBA, element slideElement) color.RGBA {
	if shouldApplyImageAlphaModFix(element) {
		c.A = scaleColorChannel(c.A, element.ImageAlphaModFixPct)
	}
	return c
}

func scaleImage(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	pictureScaler(src, srcBounds).Scale(dst, target, src, srcBounds, xdraw.Over, nil)
}

func pictureScaler(src image.Image, srcBounds image.Rectangle) xdraw.Scaler {
	if _, ok := src.(*image.YCbCr); ok && srcBounds.In(src.Bounds()) {
		return xdraw.CatmullRom
	}
	return xdraw.ApproxBiLinear
}

func scaleImageWithSoftEdge(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, radius int) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	if radius <= 0 {
		scaleImage(dst, target, src, srcBounds)
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	applySoftEdgeAlpha(layer, radius)
	for y := 0; y < layer.Bounds().Dy(); y++ {
		for x := 0; x < layer.Bounds().Dx(); x++ {
			blendPixel(dst, target.Min.X+x, target.Min.Y+y, layer.RGBAAt(x, y))
		}
	}
}

func applySoftEdgeAlpha(img *image.RGBA, radius int) {
	bounds := img.Bounds()
	if radius <= 0 || bounds.Empty() {
		return
	}
	maxRadius := min(radius, min(bounds.Dx(), bounds.Dy())/2)
	if maxRadius <= 0 {
		return
	}
	padding := maxRadius * 3
	maskWidth := bounds.Dx() + padding*2
	maskHeight := bounds.Dy() + padding*2
	mask := make([]uint8, maskWidth*maskHeight)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mask[(y+padding)*maskWidth+x+padding] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A
		}
	}
	blurred := gaussianBlurAlpha(mask, maskWidth, maskHeight, maxRadius)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixel := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			pixel.A = blurred[(y+padding)*maskWidth+x+padding]
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, pixel)
		}
	}
}

func softEdgeRadiusPixels(element slideElement, size slideSize, outputWidth int) int {
	radius := scaleEMU(element.SoftEdgeRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func scaleImageWithCustomMask(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, points []pathPoint, commands []pathCommand) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() || len(points) < 3 {
		return
	}
	if srcBounds.Empty() {
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	mask := rasterizePathMaskWithCommands(layer.Bounds(), points, commands)
	draw.DrawMask(dst, target, layer, image.Point{}, mask, image.Point{}, draw.Over)
}

func rasterizePathMask(bounds image.Rectangle, points []pathPoint) *image.Alpha {
	return rasterizePathMaskWithCommands(bounds, points, nil)
}

func rasterizePathMaskWithCommands(bounds image.Rectangle, points []pathPoint, commands []pathCommand) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if bounds.Empty() || len(points) < 3 {
		return mask
	}
	rasterizer := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	if len(commands) > 0 {
		for _, command := range commands {
			switch command.Kind {
			case "moveTo":
				if len(command.Points) == 1 {
					rasterizer.MoveTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "lnTo":
				if len(command.Points) == 1 {
					rasterizer.LineTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "cubicBezTo":
				if len(command.Points) == 3 {
					rasterizer.CubeTo(
						maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds),
						maskPathX(command.Points[1], bounds), maskPathY(command.Points[1], bounds),
						maskPathX(command.Points[2], bounds), maskPathY(command.Points[2], bounds),
					)
				}
			case "close":
				rasterizer.ClosePath()
			}
		}
		rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
		return mask
	}
	for index, point := range points {
		x := maskPathX(point, bounds)
		y := maskPathY(point, bounds)
		if index == 0 {
			rasterizer.MoveTo(x, y)
		} else {
			rasterizer.LineTo(x, y)
		}
	}
	rasterizer.ClosePath()
	rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}

func maskPathX(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.X * float64(bounds.Dx()))
}

func maskPathY(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.Y * float64(bounds.Dy()))
}

func unsupportedItems(slidePart string, elements []slideElement) []model.SkipItem {
	items := make([]model.SkipItem, 0, len(elements))
	for _, element := range elements {
		if element.Rendered {
			continue
		}
		if element.UnsupportedNote != "" {
			continue
		}
		if element.IsPlaceholder && element.Text == "" && element.EmbedID == "" {
			continue
		}
		if strings.Contains(strings.ToLower(element.Name), "placeholder") && element.Text == "" && element.EmbedID == "" {
			continue
		}
		if element.ID == "" && element.Name == "" && element.Text == "" {
			continue
		}
		message := fmt.Sprintf("%s object %q was detected but is not rendered yet", objectKindLabel(element.Kind), elementLabel(element))
		if element.Text != "" {
			message = fmt.Sprintf("%s object %q contains text and is not rendered yet", objectKindLabel(element.Kind), elementLabel(element))
		}
		items = append(items, unsupportedItem(slidePart, unsupportedCode, message))
	}
	return items
}

func timingUnsupportedItems(slidePart string, data []byte, elements []slideElement) []model.SkipItem {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil
	}
	timing := firstDescendant(root, "timing")
	if timing == nil || !timingHasAnimationBehavior(timing) || timingHasSupportedStaticVisibilityBuilds(root, timing, elements) {
		return nil
	}
	return []model.SkipItem{unsupportedItem(slidePart, partialUnsupportedCode, "slide animation timing was not evaluated for static rendering")}
}

func timingHasSupportedStaticVisibilityBuilds(root *xmlNode, timing *xmlNode, elements []slideElement) bool {
	targetIDs := partObjectIDs(root, elements)
	if len(targetIDs) == 0 {
		return false
	}
	seenVisibilityBuild := false
	for _, node := range timingBehaviorNodes(timing) {
		switch node.Name {
		case "set":
			targetID, ok := timingSetVisibilityTarget(node)
			if !ok || !targetIDs[targetID] {
				return false
			}
			seenVisibilityBuild = true
		case "animEffect":
			targetID, ok := timingStaticEntranceEffectTarget(node)
			if !ok || !targetIDs[targetID] {
				return false
			}
		case "cTn":
			if !timingContainerIsSupportedStaticEntrance(node) {
				return false
			}
		default:
			return false
		}
	}
	return seenVisibilityBuild
}

func partObjectIDs(root *xmlNode, elements []slideElement) map[string]bool {
	ids := map[string]bool{}
	for _, property := range descendantsByName(root, "cNvPr") {
		if id := attrValue(property.Attrs, "id"); id != "" {
			ids[id] = true
		}
	}
	for _, element := range elements {
		if element.ID != "" {
			ids[element.ID] = true
		}
	}
	return ids
}

func timingBehaviorNodes(node *xmlNode) []*xmlNode {
	var nodes []*xmlNode
	if timingNodeIsBehavior(node) {
		nodes = append(nodes, node)
	}
	for _, child := range node.Children {
		nodes = append(nodes, timingBehaviorNodes(child)...)
	}
	return nodes
}

func timingNodeIsBehavior(node *xmlNode) bool {
	switch node.Name {
	case "anim", "animClr", "animEffect", "animMotion", "animRot", "animScale", "cmd", "set":
		return true
	case "cTn":
		return attrValue(node.Attrs, "presetClass") != "" || attrValue(node.Attrs, "presetID") != ""
	default:
		return false
	}
}

func timingSetVisibilityTarget(node *xmlNode) (string, bool) {
	if !timingSetWritesVisibleStyle(node) {
		return "", false
	}
	target := firstDescendant(node, "spTgt")
	if target == nil {
		return "", false
	}
	return attrValue(target.Attrs, "spid"), true
}

func timingStaticEntranceEffectTarget(node *xmlNode) (string, bool) {
	if attrValue(node.Attrs, "transition") != "in" {
		return "", false
	}
	target := firstDescendant(node, "spTgt")
	if target == nil {
		return "", false
	}
	return attrValue(target.Attrs, "spid"), true
}

func timingSetWritesVisibleStyle(node *xmlNode) bool {
	attrNames := descendantsByName(node, "attrName")
	if len(attrNames) != 1 || strings.TrimSpace(attrNames[0].Text) != "style.visibility" {
		return false
	}
	value := firstDescendant(node, "strVal")
	return value != nil && strings.EqualFold(attrValue(value.Attrs, "val"), "visible")
}

func timingContainerIsSupportedStaticEntrance(node *xmlNode) bool {
	presetClass := attrValue(node.Attrs, "presetClass")
	return presetClass == "" || presetClass == "entr"
}

func timingHasAnimationBehavior(timing *xmlNode) bool {
	for _, child := range timing.Children {
		if timingNodeHasAnimationBehavior(child) {
			return true
		}
	}
	return false
}

func timingNodeHasAnimationBehavior(node *xmlNode) bool {
	if timingNodeIsBehavior(node) {
		return true
	}
	for _, child := range node.Children {
		if timingNodeHasAnimationBehavior(child) {
			return true
		}
	}
	return false
}

func unsupportedItem(slidePart string, code string, message string) model.SkipItem {
	return model.SkipItem{
		Code:    code,
		Message: message,
		Part:    slidePart,
	}
}

func elementLabel(element slideElement) string {
	label := strings.TrimSpace(element.Name)
	if label == "" {
		label = element.ID
	}
	if label == "" {
		label = element.Kind
	}
	return label
}

func objectKindLabel(kind string) string {
	switch kind {
	case "sp":
		return "shape"
	case "cxnSp":
		return "connector"
	case "pic":
		return "picture"
	case "graphicFrame":
		return "graphic frame"
	case "grpSp":
		return "group"
	default:
		return kind
	}
}

func writePNG(outputPath string, img image.Image) error {
	return writePNGWithDPI(outputPath, img, defaultOutputDPI)
}

func writePNGWithDPI(outputPath string, img image.Image, dpi int) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		return err
	}
	_, err = file.Write(pngWithOutputMetadata(data.Bytes(), normalizeOutputDPI(dpi)))
	return err
}

func pngWithOutputMetadata(data []byte, dpi int) []byte {
	if len(data) < 33 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return data
	}
	if string(data[12:16]) != "IHDR" {
		return data
	}
	pixelsPerMeter := uint32(math.Round(float64(normalizeOutputDPI(dpi)) / 0.0254))
	chunkData := make([]byte, 9)
	writeUint32BE(chunkData[0:4], pixelsPerMeter)
	writeUint32BE(chunkData[4:8], pixelsPerMeter)
	chunkData[8] = 1
	colorChunk := pngChunk("cICP", displayP3CICPChunkData)
	densityChunk := pngChunk("pHYs", chunkData)
	output := make([]byte, 0, len(data)+len(colorChunk)+len(densityChunk))
	output = append(output, data[:33]...)
	output = append(output, colorChunk...)
	output = append(output, densityChunk...)
	output = append(output, data[33:]...)
	return output
}

func pngChunk(chunkType string, data []byte) []byte {
	chunk := make([]byte, 8+len(data)+4)
	writeUint32BE(chunk[0:4], uint32(len(data)))
	copy(chunk[4:8], chunkType)
	copy(chunk[8:8+len(data)], data)
	crc := crc32.NewIEEE()
	_, _ = crc.Write(chunk[4 : 8+len(data)])
	writeUint32BE(chunk[8+len(data):], crc.Sum32())
	return chunk
}

func writeUint32BE(data []byte, value uint32) {
	if len(data) < 4 {
		return
	}
	data[0] = byte(value >> 24)
	data[1] = byte(value >> 16)
	data[2] = byte(value >> 8)
	data[3] = byte(value)
}
