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
	HasBold         bool
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
	HasBold          bool
	Bold             bool
	HasItalic        bool
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
	HasBold          bool
	Bold             bool
	HasItalic        bool
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
