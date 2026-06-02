package render

import (
	"encoding/xml"
	"image/color"

	"golang.org/x/image/font"
)

type Options struct {
	SlideNumber int
	OutputPath  string
	DPI         int
	ObjectDebug *ObjectDebugOptions
}

type ObjectDebugRenderMode string

const (
	ObjectDebugRenderNormal     ObjectDebugRenderMode = ""
	ObjectDebugRenderBefore     ObjectDebugRenderMode = "before"
	ObjectDebugRenderObjectOnly ObjectDebugRenderMode = "object"
	ObjectDebugRenderThrough    ObjectDebugRenderMode = "through"
)

type ObjectDebugOptions struct {
	Mode              ObjectDebugRenderMode
	TargetZOrder      int
	HasFlatBackground bool
	FlatBackground    color.RGBA
	ArtifactDir       string
	Records           []PaintedObject
	nextZOrder        int
}

type PaintedObject struct {
	SlidePart           string               `json:"slide_part"`
	SourcePart          string               `json:"source_part"`
	XMLPath             string               `json:"xml_path,omitempty"`
	CNvPrID             string               `json:"cnv_pr_id,omitempty"`
	CNvPrName           string               `json:"cnv_pr_name,omitempty"`
	CNvPrDescription    string               `json:"cnv_pr_description,omitempty"`
	CNvPrTitle          string               `json:"cnv_pr_title,omitempty"`
	CNvPrCreationID     string               `json:"cnv_pr_creation_id,omitempty"`
	Kind                string               `json:"kind"`
	ZOrder              int                  `json:"z_order"`
	Bounds              ObjectEMUPointBounds `json:"bounds_emu,omitempty"`
	ResolvedStyle       ObjectStyleSummary   `json:"resolved_style"`
	PixelBounds         ObjectPixelBounds    `json:"pixel_bounds,omitempty"`
	FractionalBounds    ObjectFloatBounds    `json:"fractional_pixel_bounds,omitempty"`
	OutputPixelBounds   *ObjectPixelBounds   `json:"output_pixel_bounds,omitempty"`
	BeforeArtifactPath  string               `json:"before_artifact_path,omitempty"`
	ObjectArtifactPath  string               `json:"object_artifact_path,omitempty"`
	ThroughArtifactPath string               `json:"through_artifact_path,omitempty"`
	Unsupported         []ObjectUnsupported  `json:"unsupported,omitempty"`
	Painted             bool                 `json:"painted"`
}

type ObjectUnsupported struct {
	Code    string `json:"code"`
	Part    string `json:"part,omitempty"`
	Message string `json:"message"`
}

type ObjectEMUPointBounds struct {
	X  int64 `json:"x"`
	Y  int64 `json:"y"`
	CX int64 `json:"cx"`
	CY int64 `json:"cy"`
}

type ObjectPixelBounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

type ObjectFloatBounds struct {
	MinX float64 `json:"min_x"`
	MinY float64 `json:"min_y"`
	MaxX float64 `json:"max_x"`
	MaxY float64 `json:"max_y"`
}

type ObjectFloatPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ObjectStyleSummary struct {
	Geometry                string             `json:"geometry,omitempty"`
	CustomPathPoints        int                `json:"custom_path_points,omitempty"`
	CustomPathCoordinates   []ObjectFloatPoint `json:"custom_path_coordinates,omitempty"`
	CustomPathBounds        *ObjectFloatBounds `json:"custom_path_bounds,omitempty"`
	CustomPathCommands      int                `json:"custom_path_commands,omitempty"`
	CustomPathUnsupported   []string           `json:"custom_path_unsupported,omitempty"`
	Fill                    string             `json:"fill,omitempty"`
	PatternFill             string             `json:"pattern_fill,omitempty"`
	Line                    string             `json:"line,omitempty"`
	Text                    string             `json:"text,omitempty"`
	FontFamily              string             `json:"font_family,omitempty"`
	FontFamilies            []string           `json:"font_families,omitempty"`
	FontSize                int                `json:"font_size,omitempty"`
	ParagraphFontSize       int                `json:"paragraph_font_size,omitempty"`
	Bold                    bool               `json:"bold,omitempty"`
	Italic                  bool               `json:"italic,omitempty"`
	TextColor               string             `json:"text_color,omitempty"`
	TextAlign               string             `json:"text_align,omitempty"`
	TextBox                 bool               `json:"text_box,omitempty"`
	TextBodyProperties      []string           `json:"text_body_properties,omitempty"`
	TextParagraphProperties []string           `json:"text_paragraph_properties,omitempty"`
	Description             string             `json:"description,omitempty"`
	Title                   string             `json:"title,omitempty"`
	CreationID              string             `json:"creation_id,omitempty"`
	NonVisualProperties     []string           `json:"non_visual_properties,omitempty"`
	NonVisualLocks          []string           `json:"non_visual_locks,omitempty"`
	Placeholder             string             `json:"placeholder,omitempty"`
	EmbedID                 string             `json:"embed_id,omitempty"`
	SVGEmbedID              string             `json:"svg_embed_id,omitempty"`
	Image                   string             `json:"image,omitempty"`
	ImageCrop               string             `json:"image_crop,omitempty"`
	ImageEffects            []string           `json:"image_effects,omitempty"`
	ImageUnsupported        []string           `json:"image_unsupported,omitempty"`
	EffectUnsupported       []string           `json:"effect_unsupported,omitempty"`
	Table                   bool               `json:"table,omitempty"`
	TableStyleID            string             `json:"table_style_id,omitempty"`
	TableProperties         []string           `json:"table_properties,omitempty"`
	TableColumnIDs          []string           `json:"table_column_ids,omitempty"`
	TableRowIDs             []string           `json:"table_row_ids,omitempty"`
	TableUnsupported        []string           `json:"table_unsupported,omitempty"`
	Shadow                  bool               `json:"shadow,omitempty"`
	ShadowColor             string             `json:"shadow_color,omitempty"`
	ShadowBlur              int64              `json:"shadow_blur_emu,omitempty"`
	ShadowDistance          int64              `json:"shadow_distance_emu,omitempty"`
	ShadowDirection         int64              `json:"shadow_direction,omitempty"`
	ShadowAlignment         string             `json:"shadow_alignment,omitempty"`
	ShadowScaleX            int64              `json:"shadow_scale_x,omitempty"`
	ShadowScaleY            int64              `json:"shadow_scale_y,omitempty"`
	ShadowSkewX             int64              `json:"shadow_skew_x,omitempty"`
	ShadowSkewY             int64              `json:"shadow_skew_y,omitempty"`
	GradientFill            bool               `json:"gradient_fill,omitempty"`
	NoFill                  bool               `json:"no_fill,omitempty"`
	NoLine                  bool               `json:"no_line,omitempty"`
}

type slideSize struct {
	CX int64
	CY int64
}

type backgroundPaint struct {
	Color       color.RGBA
	HasGradient bool
	Gradient    gradientPaint
	HasPattern  bool
	Pattern     patternPaint
	Unsupported []string
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

type patternPaint struct {
	Preset     string
	Foreground color.RGBA
	Background color.RGBA
}

type relativeRect struct {
	Left   int64
	Top    int64
	Right  int64
	Bottom int64
}

type textStyle struct {
	FontSize        int
	HasBold         bool
	Bold            bool
	HasTextColor    bool
	TextColor       color.RGBA
	TextAlign       string
	ParagraphStyles map[int]paragraphStyle
}

type paragraphStyle struct {
	HasMarginLeft     bool
	MarginLeft        int64
	HasMarginRight    bool
	MarginRight       int64
	HasIndent         bool
	Indent            int64
	FontFamily        string
	FontSize          int
	HasSpaceBefore    bool
	SpaceBefore       int
	SpaceBeforePct    int
	HasSpaceAfter     bool
	SpaceAfter        int
	SpaceAfterPct     int
	HasLineSpacing    bool
	LineSpacingPct    int
	HasDefaultTab     bool
	DefaultTabSize    int64
	Bullet            string
	BulletFontFamily  string
	BulletFontTx      bool
	BulletFontSize    int
	BulletSizePct     int
	BulletSizeTx      bool
	HasAutoNumber     bool
	AutoNumberType    string
	AutoNumberStart   int
	HasBulletColor    bool
	BulletColor       color.RGBA
	BulletColorTx     bool
	HasBold           bool
	Bold              bool
	HasItalic         bool
	Italic            bool
	HasTextCaps       bool
	TextCaps          string
	HasCharSpacing    bool
	CharSpacing       int
	TextAlign         string
	FontAlign         string
	HasRTL            bool
	RTL               bool
	HasEALineBreak    bool
	EALineBreak       bool
	HasLatinLineBreak bool
	LatinLineBreak    bool
	HasHangingPunct   bool
	HangingPunct      bool
	HasTextColor      bool
	TextColor         color.RGBA
	NoBullet          bool
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
	LinkID                     string
	SVGEmbedID                 string
	ImageMediaPart             string
	ImageContentType           string
	ImageWidth                 int
	ImageHeight                int
	DiagramDataID              string
	GraphicPayloadKind         string
	GraphicPayloadURI          string
	PayloadRelationshipID      string
	OLEProgID                  string
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
	HasImageAlphaModulate      bool
	ImageAlphaModulatePct      int64
	HasImageAlphaBiLevel       bool
	ImageAlphaBiLevelThreshold int64
	HasImageAlphaCeiling       bool
	HasImageAlphaFloor         bool
	HasImageAlphaInverse       bool
	HasImageAlphaReplace       bool
	ImageAlphaReplacePct       int64
	HasImageBiLevel            bool
	ImageBiLevelThreshold      int64
	HasImageGrayscale          bool
	HasImageLuminance          bool
	ImageLuminanceBright       int64
	ImageLuminanceContrast     int64
	HasImageHSL                bool
	ImageHSLHue                int64
	ImageHSLSaturation         int64
	ImageHSLLuminance          int64
	HasImageTint               bool
	ImageTintHue               int64
	ImageTintAmount            int64
	HasImageBlur               bool
	ImageBlurRadius            int64
	ImageBlurGrow              bool
	HasImageFillOverlay        bool
	ImageFillOverlay           backgroundPaint
	ImageFillOverlayBlend      string
	HasImageColorChange        bool
	ImageColorChangeFrom       color.RGBA
	ImageColorChangeTo         color.RGBA
	ImageColorChangeUseAlpha   bool
	HasImageColorReplace       bool
	ImageColorReplace          color.RGBA
	HasImageDuotone            bool
	ImageDuotoneDark           color.RGBA
	ImageDuotoneLight          color.RGBA
	BlipFillMode               string
	BlipTileOffsetX            int64
	BlipTileOffsetY            int64
	BlipTileScaleX             int64
	BlipTileScaleY             int64
	BlipTileFlip               string
	BlipTileAlignment          string
	BlipCompressionState       string
	ImageUnsupported           []string
	HasBlipRotWithShape        bool
	BlipRotWithShape           bool
	BWMode                     string
	HasRotation                bool
	Rotation                   int
	Rendered                   bool
	UnsupportedNote            string
	PaintUnsupported           []string
	IsPlaceholder              bool
	PlaceholderType            string
	PlaceholderIdx             string
	PrstGeom                   string
	PrstGeomAdjustments        map[string]int64
	HasFill                    bool
	FillColor                  color.RGBA
	HasFillGradient            bool
	FillGradient               gradientPaint
	HasPatternFill             bool
	PatternFill                patternPaint
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
	LineJoin                   string
	HasLineJoin                bool
	LineCompound               string
	HasLineCompound            bool
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
	HasInnerShadow             bool
	InnerShadowColor           color.RGBA
	InnerShadowBlur            int64
	InnerShadowDistance        int64
	InnerShadowDirection       int64
	HasReflection              bool
	ReflectionBlur             int64
	ReflectionStartAlpha       int64
	ReflectionStartPosition    int64
	ReflectionEndAlpha         int64
	ReflectionEndPosition      int64
	ReflectionDistance         int64
	ReflectionDirection        int64
	ReflectionFadeDirection    int64
	ReflectionScaleX           int64
	ReflectionScaleY           int64
	ReflectionSkewX            int64
	ReflectionSkewY            int64
	ReflectionAlignment        string
	HasReflectionRotate        bool
	ReflectionRotateWithShape  bool
	HasSoftEdge                bool
	SoftEdgeRadius             int64
	HasBlur                    bool
	BlurRadius                 int64
	BlurGrow                   bool
	HasAlphaOutset             bool
	AlphaOutsetRadius          int64
	HasRelativeOffset          bool
	RelativeOffsetX            int64
	RelativeOffsetY            int64
	HasEffectTransform         bool
	EffectTransformScaleX      int64
	EffectTransformScaleY      int64
	EffectTransformSkewX       int64
	EffectTransformSkewY       int64
	EffectTransformOffsetX     int64
	EffectTransformOffsetY     int64
	HasFillOverlay             bool
	FillOverlay                backgroundPaint
	FillOverlayBlend           string
	HasGlow                    bool
	GlowColor                  color.RGBA
	GlowRadius                 int64
	HasShape3D                 bool
	Shape3DFeatures            []string
	EffectUnsupported          []string
	IsTextBox                  bool
	Description                string
	Title                      string
	CreationID                 string
	HasHidden                  bool
	Hidden                     bool
	HasDecorative              bool
	Decorative                 bool
	NonVisualProperties        []string
	NonVisualLocks             []string
	CustomPath                 []pathPoint
	CustomPaths                [][]pathPoint
	CustomPathFills            []bool
	CustomPathStrokes          []bool
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
	HasTextRightToLeftColumns  bool
	TextRightToLeftColumns     bool
	Text3DFeatures             []string
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
	Text              string
	Bullet            string
	HasAutoNumber     bool
	Level             int
	TextAlign         string
	FontAlign         string
	HasRTL            bool
	RTL               bool
	HasEALineBreak    bool
	EALineBreak       bool
	HasLatinLineBreak bool
	LatinLineBreak    bool
	HasHangingPunct   bool
	HangingPunct      bool
	FontFamily        string
	Language          string
	BulletFontFamily  string
	BulletFontTx      bool
	FontSize          int
	HasBold           bool
	Bold              bool
	HasItalic         bool
	Italic            bool
	HasTextCaps       bool
	TextCaps          string
	HasCharSpacing    bool
	CharSpacing       int
	HasTextColor      bool
	TextColor         color.RGBA
	NoBullet          bool
	BulletFontSize    int
	BulletSizePct     int
	BulletSizeTx      bool
	HasBulletColor    bool
	BulletColor       color.RGBA
	BulletColorTx     bool
	HasMarginLeft     bool
	MarginLeft        int64
	HasMarginRight    bool
	MarginRight       int64
	HasIndent         bool
	Indent            int64
	HasSpaceBefore    bool
	SpaceBefore       int
	SpaceBeforePct    int
	HasSpaceAfter     bool
	SpaceAfter        int
	SpaceAfterPct     int
	HasLineSpacing    bool
	LineSpacingPct    int
	TabStops          []int64
	HasDefaultTab     bool
	DefaultTabSize    int64
	Runs              []textRun
}

type textRun struct {
	Text              string
	FieldType         string
	FontFamily        string
	Language          string
	FontSize          int
	HasBold           bool
	Bold              bool
	HasItalic         bool
	Italic            bool
	Underline         bool
	HasUnderlineColor bool
	UnderlineColor    color.RGBA
	Strike            string
	HasTextCaps       bool
	TextCaps          string
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
	ColumnIDs           []string
	Rows                []tableRow
	StyleID             string
	HasBackground       bool
	NoBackground        bool
	Background          backgroundPaint
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
	ID        string
	Cells     []tableCell
}

type tableCell struct {
	Text                       string
	TextParagraphs             []textParagraph
	ColSpan                    int
	HMerge                     bool
	RowSpan                    int
	VMerge                     bool
	FontSize                   int
	HasFontSize                bool
	HasTextColor               bool
	TextColor                  color.RGBA
	TextAlign                  string
	TextAnchor                 string
	HasTextHorizontalOverflow  bool
	TextHorizontalOverflow     string
	HasTextVerticalOverflow    bool
	TextVerticalOverflow       string
	HasTextVertical            bool
	TextVertical               string
	HasTextAnchorCenter        bool
	TextAnchorCenter           bool
	HasFill                    bool
	FillColor                  color.RGBA
	FillPaint                  backgroundPaint
	NoFill                     bool
	HasMargins                 bool
	MarginLeft                 int64
	MarginRight                int64
	MarginTop                  int64
	MarginBottom               int64
	BorderLeft                 tableCellBorder
	BorderRight                tableCellBorder
	BorderTop                  tableCellBorder
	BorderBottom               tableCellBorder
	BorderTopLeftToBottomRight tableCellBorder
	BorderBottomLeftToTopRight tableCellBorder
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
	Specified        bool
	HasLine          bool
	NoLine           bool
	Color            color.RGBA
	Width            int64
	Dash             string
	Cap              string
	Align            string
	Join             string
	Compound         string
	HeadMarker       string
	HeadMarkerWidth  string
	HeadMarkerLength string
	TailMarker       string
	TailMarkerWidth  string
	TailMarkerLength string
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
	FillPaint    backgroundPaint
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
	Left                   tableCellBorder
	Right                  tableCellBorder
	Top                    tableCellBorder
	Bottom                 tableCellBorder
	InsideH                tableCellBorder
	InsideV                tableCellBorder
	TopLeftToBottomRight   tableCellBorder
	BottomLeftToTopRight   tableCellBorder
	TopOverridesInsideH    bool
	BottomOverridesInsideH bool
	LeftOverridesInsideV   bool
	RightOverridesInsideV  bool
}

type xmlNode struct {
	Name     string
	Attrs    []xml.Attr
	Children []*xmlNode
	Text     string
}

type renderTransform struct {
	ScaleX    float64
	ScaleY    float64
	OffsetX   float64
	OffsetY   float64
	GroupFill *backgroundPaint
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
	HasShadow                 bool
	ShadowColor               color.RGBA
	ShadowBlur                int64
	ShadowDistance            int64
	ShadowDirection           int64
	ShadowAlignment           string
	HasShadowRotateWithShape  bool
	ShadowRotateWithShape     bool
	HasShadowScaleX           bool
	ShadowScaleX              int64
	HasShadowScaleY           bool
	ShadowScaleY              int64
	HasShadowSkewX            bool
	ShadowSkewX               int64
	HasShadowSkewY            bool
	ShadowSkewY               int64
	HasInnerShadow            bool
	InnerShadowColor          color.RGBA
	InnerShadowBlur           int64
	InnerShadowDistance       int64
	InnerShadowDirection      int64
	HasReflection             bool
	ReflectionBlur            int64
	ReflectionStartAlpha      int64
	ReflectionStartPosition   int64
	ReflectionEndAlpha        int64
	ReflectionEndPosition     int64
	ReflectionDistance        int64
	ReflectionDirection       int64
	ReflectionFadeDirection   int64
	ReflectionScaleX          int64
	ReflectionScaleY          int64
	ReflectionSkewX           int64
	ReflectionSkewY           int64
	ReflectionAlignment       string
	HasReflectionRotate       bool
	ReflectionRotateWithShape bool
	HasGlow                   bool
	GlowColor                 color.RGBA
	GlowRadius                int64
	HasBlur                   bool
	BlurRadius                int64
	BlurGrow                  bool
	HasAlphaOutset            bool
	AlphaOutsetRadius         int64
	HasRelativeOffset         bool
	RelativeOffsetX           int64
	RelativeOffsetY           int64
	HasEffectTransform        bool
	EffectTransformScaleX     int64
	EffectTransformScaleY     int64
	EffectTransformSkewX      int64
	EffectTransformSkewY      int64
	EffectTransformOffsetX    int64
	EffectTransformOffsetY    int64
	HasFillOverlay            bool
	FillOverlay               backgroundPaint
	FillOverlayBlend          string
	HasShape3D                bool
	Shape3DFeatures           []string
	EffectUnsupported         []string
}

// Render writes a PNG for one slide and returns the stable command result used
// by the CLI. Unsupported visible objects are reported explicitly.

type textInsets struct {
	Left   int64
	Top    int64
	Right  int64
	Bottom int64
}

type floatRect struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

type floatPoint struct {
	X float64
	Y float64
}

type styledWordToken struct {
	Segment            textLineSegment
	Prefix             string
	PreserveLinePrefix bool
}

type wrapTextPart struct {
	Text  string
	Space bool
}

type noKerningFace struct {
	font.Face
}

type textRenderLine struct {
	Text           string
	Bold           bool
	Italic         bool
	FontSize       int
	TextAlign      string
	FontAlign      string
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
	Language          string
	Bold              bool
	Italic            bool
	Underline         bool
	HasUnderlineColor bool
	UnderlineColor    color.RGBA
	Strike            string
	TextCaps          string
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

type plainWordToken struct {
	Text               string
	Prefix             string
	PreserveLinePrefix bool
}

type roundedCorners struct {
	TopLeft     bool
	TopRight    bool
	BottomLeft  bool
	BottomRight bool
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

type fontSource struct {
	Data  []byte
	Label string
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

type svgPathToken struct {
	Command  byte
	Number   float64
	IsNumber bool
}
