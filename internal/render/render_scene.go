package render

import (
	"fmt"
	"image"
	"image/color"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

type renderPrimitiveKind string

const (
	renderPrimitivePicture      renderPrimitiveKind = "picture"
	renderPrimitiveShape        renderPrimitiveKind = "shape"
	renderPrimitiveConnector    renderPrimitiveKind = "connector"
	renderPrimitiveGraphicFrame renderPrimitiveKind = "graphicFrame"
	renderPrimitiveGroup        renderPrimitiveKind = "group"
	renderPrimitiveUnsupported  renderPrimitiveKind = "unsupported"
)

type renderScene struct {
	SlidePart  string
	SourcePart string
	Canvas     ObjectPixelBounds
	Primitives []renderPrimitive
}

type renderPrimitive struct {
	Kind         renderPrimitiveKind
	ZOrder       int
	Provenance   renderPrimitiveProvenance
	Picture      *renderPicturePrimitive
	Shape        *renderShapePrimitive
	Connector    *renderConnectorPrimitive
	GraphicFrame *renderGraphicFramePrimitive
	Group        *renderGroupPrimitive
	Unsupported  *renderUnsupportedPrimitive
}

type renderPrimitiveProvenance struct {
	ObjectKind          string   `json:"object_kind"`
	ID                  string   `json:"id,omitempty"`
	Name                string   `json:"name,omitempty"`
	Description         string   `json:"description,omitempty"`
	Title               string   `json:"title,omitempty"`
	CreationID          string   `json:"creation_id,omitempty"`
	NonVisualProperties []string `json:"non_visual_properties,omitempty"`
	SourcePart          string   `json:"source_part"`
	XMLPath             string   `json:"xml_path,omitempty"`
	RelationshipIDs     []string `json:"relationship_ids,omitempty"`
	SchemaAnchors       []string `json:"schema_anchors"`
}

type renderPicturePrimitive struct {
	Provenance                renderPrimitiveProvenance
	ObjectKind                string
	ID                        string
	Name                      string
	SourcePart                string
	RelationshipID            string
	LinkRelationshipID        string
	SVGRelationshipID         string
	NonVisualLocks            []string
	MediaPart                 string
	ContentType               string
	Target                    ObjectPixelBounds
	FractionalTarget          ObjectFloatBounds
	Crop                      relativeRect
	BlipFillMode              string
	BlipTileOffsetX           int64
	BlipTileOffsetY           int64
	BlipTileScaleX            int64
	BlipTileScaleY            int64
	BlipTileFlip              string
	BlipTileAlignment         string
	BlipCompressionState      string
	FlipH                     bool
	FlipV                     bool
	HasAlphaModFix            bool
	AlphaModFixPct            int64
	HasAlphaModulate          bool
	AlphaModulatePct          int64
	HasAlphaBiLevel           bool
	AlphaBiLevelThreshold     int64
	HasAlphaCeiling           bool
	HasAlphaFloor             bool
	HasAlphaInverse           bool
	HasAlphaReplace           bool
	AlphaReplacePct           int64
	HasBiLevel                bool
	BiLevelThreshold          int64
	HasGrayscale              bool
	HasLuminance              bool
	LuminanceBright           int64
	LuminanceContrast         int64
	HasHSL                    bool
	HSLHue                    int64
	HSLSaturation             int64
	HSLLuminance              int64
	HasTint                   bool
	TintHue                   int64
	TintAmount                int64
	HasSourceBlur             bool
	SourceBlurRadius          int64
	SourceBlurGrow            bool
	HasSourceFillOverlay      bool
	SourceFillOverlay         backgroundPaint
	SourceFillOverlayBlend    string
	HasColorChange            bool
	ColorChangeFrom           color.RGBA
	ColorChangeTo             color.RGBA
	ColorChangeUseAlpha       bool
	HasColorReplace           bool
	ColorReplace              color.RGBA
	HasDuotone                bool
	DuotoneDark               color.RGBA
	DuotoneLight              color.RGBA
	ImageUnsupported          []string
	RotationDegrees           int
	RotatesWithShape          bool
	HasSoftEdge               bool
	SoftEdgeRadius            int64
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
	HasCustomMask             bool
	CustomPath                []pathPoint
	CustomPathCommands        []pathCommand
	CustomPathUnsupported     []string
	CustomMaskPoints          int
	CustomMaskCommands        int
	HasLine                   bool
	NoLine                    bool
	LineWidth                 int64
	LineColor                 color.RGBA
	LineDash                  string
	LineAlign                 string
	LineCap                   string
	LineJoin                  string
	LineCompound              string
	HasShadow                 bool
	ShadowColor               color.RGBA
	ShadowBlur                int64
	ShadowDistance            int64
	ShadowDirection           int64
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
	HasShape3D                bool
	Shape3DFeatures           []string
	EffectUnsupported         []string
}

type renderShapePrimitive struct {
	Provenance          renderPrimitiveProvenance
	Target              ObjectPixelBounds
	FractionalTarget    ObjectFloatBounds
	Geometry            string
	GeometryAdjustments map[string]int64
	CustomPath          renderPathPrimitive
	Fill                renderFillPrimitive
	Stroke              renderStrokePrimitive
	RotationDegrees     int
	FlipH               bool
	FlipV               bool
	NonVisualLocks      []string
	Text                *renderTextPrimitive
	Effect              *renderEffectPrimitive
	Unsupported         []string
}

type renderConnectorPrimitive struct {
	Provenance       renderPrimitiveProvenance
	Target           ObjectPixelBounds
	FractionalTarget ObjectFloatBounds
	Geometry         string
	NonVisualLocks   []string
	Stroke           renderStrokePrimitive
	Start            image.Point
	End              image.Point
	Effect           *renderEffectPrimitive
	Unsupported      []string
}

type renderGraphicFramePrimitive struct {
	Provenance       renderPrimitiveProvenance
	Target           ObjectPixelBounds
	FractionalTarget ObjectFloatBounds
	PayloadKind      string
	PayloadURI       string
	RelationshipID   string
	PayloadPart      string
	PayloadType      string
	NonVisualLocks   []string
	Text             *renderTextPrimitive
	Table            *renderTablePrimitive
	Diagram          *renderDiagramPrimitive
	Unsupported      []string
}

type renderGroupPrimitive struct {
	Provenance       renderPrimitiveProvenance
	Target           ObjectPixelBounds
	FractionalTarget ObjectFloatBounds
	NonVisualLocks   []string
	Unsupported      []string
}

type renderPathPrimitive struct {
	Provenance    renderPrimitiveProvenance
	Preset        string
	Points        []pathPoint
	Subpaths      [][]pathPoint
	Fills         []bool
	Strokes       []bool
	Commands      []pathCommand
	Unsupported   []string
	SchemaAnchors []string
}

type renderTextPrimitive struct {
	Provenance                 renderPrimitiveProvenance
	Text                       string
	Paragraphs                 []textParagraph
	FontFamily                 string
	FontSize                   int
	FontPointScale             float64
	Italic                     bool
	HasTextColor               bool
	TextColor                  color.RGBA
	Align                      string
	IsTextBox                  bool
	Anchor                     string
	Wrap                       string
	HorizontalOverflow         string
	VerticalOverflow           string
	Insets                     renderTextInsets
	HasShapeAutofit            bool
	HasNormAutofit             bool
	HasNoAutofit               bool
	HasFontScalePct            bool
	FontScalePct               int
	HasLineSpacingReductionPct bool
	LineSpacingReductionPct    int
	HasFirstLastSpacing        bool
	IncludeFirstLastSpacing    bool
	HasTextVertical            bool
	TextVertical               string
	HasTextBodyRotation        bool
	TextBodyRotation           int
	HasTextColumns             bool
	TextColumnCount            int
	HasRTLColumns              bool
	RTLColumns                 bool
	FontResolution             []string
	Unsupported                []string
	SchemaAnchors              []string
}

type renderTextInsets struct {
	Left   int64
	Top    int64
	Right  int64
	Bottom int64
}

type renderTablePrimitive struct {
	Provenance          renderPrimitiveProvenance
	Columns             []int64
	ColumnIDs           []string
	Rows                []tableRow
	StyleID             string
	FirstRow            bool
	FirstCol            bool
	LastRow             bool
	LastCol             bool
	BandRow             bool
	BandCol             bool
	UnsupportedFeatures []string
	SchemaAnchors       []string
}

type renderDiagramPrimitive struct {
	Provenance    renderPrimitiveProvenance
	DataRelID     string
	DataPart      string
	SchemaAnchors []string
}

type renderEffectPrimitive struct {
	Provenance                renderPrimitiveProvenance
	HasEffectProperties       bool
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
	HasSoftEdge               bool
	SoftEdgeRadius            int64
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
	HasGlow                   bool
	GlowColor                 color.RGBA
	GlowRadius                int64
	HasShape3D                bool
	Shape3DFeatures           []string
	Unsupported               []string
	SchemaAnchors             []string
}

type renderFillPrimitive struct {
	HasFill       bool
	Color         color.RGBA
	NoFill        bool
	HasGradient   bool
	Gradient      gradientPaint
	HasPattern    bool
	Pattern       patternPaint
	Unsupported   []string
	SchemaAnchors []string
}

type renderStrokePrimitive struct {
	HasLine          bool
	NoLine           bool
	Color            color.RGBA
	Width            int64
	HasWidth         bool
	Dash             string
	HasDash          bool
	Cap              string
	HasCap           bool
	Align            string
	HasAlign         bool
	HasMarker        bool
	HeadMarker       string
	HeadMarkerWidth  string
	HeadMarkerLength string
	TailMarker       string
	TailMarkerWidth  string
	TailMarkerLength string
	Join             string
	HasJoin          bool
	Compound         string
	HasCompound      bool
}

type renderUnsupportedPrimitive struct {
	Provenance renderPrimitiveProvenance
	Reason     string
}

type shapeBackend interface {
	RenderShape(input shapeBackendInput) []model.SkipItem
}

type shapeBackendInput struct {
	SlidePart string
	Size      slideSize
	Canvas    *image.RGBA
	Primitive renderShapePrimitive
}

type connectorBackend interface {
	RenderConnector(input connectorBackendInput) []model.SkipItem
}

type connectorBackendInput struct {
	SlidePart string
	Size      slideSize
	Canvas    *image.RGBA
	Primitive renderConnectorPrimitive
}

type graphicFrameBackend interface {
	RenderGraphicFrame(input graphicFrameBackendInput) []model.SkipItem
}

type graphicFrameBackendInput struct {
	SlidePart string
	Size      slideSize
	Canvas    *image.RGBA
	Primitive renderGraphicFramePrimitive
}

func renderSceneFromElements(pkg *pptx.Package, slidePart string, sourcePart string, size slideSize, canvas image.Rectangle, elements []slideElement, relationships map[string]pptx.Relationship) (renderScene, []error) {
	scene := renderScene{
		SlidePart:  slidePart,
		SourcePart: sourcePart,
		Canvas:     objectPixelBoundsFromImageRect(canvas),
		Primitives: make([]renderPrimitive, 0, len(elements)),
	}
	var errs []error
	for index, element := range elements {
		zOrder := index + 1
		switch element.Kind {
		case "pic":
			picture, err := renderPicturePrimitiveFromElement(pkg, sourcePart, size, canvas, element, relationships)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitivePicture, ZOrder: zOrder, Provenance: picture.Provenance, Picture: &picture})
		case "sp":
			if element.EmbedID != "" || element.LinkID != "" {
				picture, err := renderPicturePrimitiveFromElement(pkg, sourcePart, size, canvas, element, relationships)
				if err != nil {
					errs = append(errs, err)
				} else {
					scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitivePicture, ZOrder: zOrder, Provenance: picture.Provenance, Picture: &picture})
				}
			}
			shape := renderShapePrimitiveFromElement(sourcePart, size, canvas, element)
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitiveShape, ZOrder: zOrder, Provenance: shape.Provenance, Shape: &shape})
		case "cxnSp":
			if element.EmbedID != "" || element.LinkID != "" {
				picture, err := renderPicturePrimitiveFromElement(pkg, sourcePart, size, canvas, element, relationships)
				if err != nil {
					errs = append(errs, err)
				} else {
					scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitivePicture, ZOrder: zOrder, Provenance: picture.Provenance, Picture: &picture})
				}
			}
			connector := renderConnectorPrimitiveFromElement(sourcePart, size, canvas, element)
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitiveConnector, ZOrder: zOrder, Provenance: connector.Provenance, Connector: &connector})
		case "graphicFrame":
			frame, err := renderGraphicFramePrimitiveFromElement(sourcePart, size, canvas, element, relationships)
			if err != nil {
				errs = append(errs, err)
			}
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitiveGraphicFrame, ZOrder: zOrder, Provenance: frame.Provenance, GraphicFrame: &frame})
		case "grpSp":
			group := renderGroupPrimitiveFromElement(sourcePart, size, canvas, element)
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitiveGroup, ZOrder: zOrder, Provenance: group.Provenance, Group: &group})
		default:
			unsupported := renderUnsupportedPrimitiveFromElement(sourcePart, element)
			scene.Primitives = append(scene.Primitives, renderPrimitive{Kind: renderPrimitiveUnsupported, ZOrder: zOrder, Provenance: unsupported.Provenance, Unsupported: &unsupported})
		}
	}
	return scene, errs
}

func renderPicturePrimitiveFromElement(pkg *pptx.Package, sourcePart string, size slideSize, canvas image.Rectangle, element slideElement, relationships map[string]pptx.Relationship) (renderPicturePrimitive, error) {
	if element.Kind != "pic" && element.Kind != "sp" && element.Kind != "cxnSp" {
		return renderPicturePrimitive{}, fmt.Errorf("render picture primitive requires picture-backed element, got %q", element.Kind)
	}
	relationshipID := element.EmbedID
	if relationshipID == "" {
		relationshipID = element.LinkID
	}
	if relationshipID == "" {
		return renderPicturePrimitive{}, fmt.Errorf("picture object %q has no image relationship", elementLabel(element))
	}
	relationship, ok := relationships[relationshipID]
	if !ok {
		return renderPicturePrimitive{}, fmt.Errorf("picture object %q references missing relationship %q", elementLabel(element), relationshipID)
	}
	if relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return renderPicturePrimitive{}, fmt.Errorf("picture object %q uses unsupported relationship %q", elementLabel(element), relationship.Type)
	}
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return renderPicturePrimitive{}, fmt.Errorf("picture object %q has no renderable transform", elementLabel(element))
	}
	mediaPart := pptx.ResolveTargetPart(sourcePart, relationship.Target)
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	provenance.RelationshipIDs = appendRelationshipID(provenance.RelationshipIDs, element.EmbedID)
	provenance.RelationshipIDs = appendRelationshipID(provenance.RelationshipIDs, element.LinkID)
	provenance.RelationshipIDs = appendRelationshipID(provenance.RelationshipIDs, element.SVGEmbedID)
	return renderPicturePrimitive{
		Provenance:                provenance,
		ObjectKind:                element.Kind,
		ID:                        element.ID,
		Name:                      element.Name,
		SourcePart:                sourcePart,
		RelationshipID:            element.EmbedID,
		LinkRelationshipID:        element.LinkID,
		SVGRelationshipID:         element.SVGEmbedID,
		NonVisualLocks:            append([]string{}, element.NonVisualLocks...),
		MediaPart:                 mediaPart,
		ContentType:               pkg.ContentTypes.ForPart(mediaPart),
		Target:                    objectPixelBoundsFromImageRect(sceneElementPixelTarget(element, size, canvas)),
		FractionalTarget:          elementFractionalTarget(element, size, canvas),
		Crop:                      relativeRect{Left: element.CropLeft, Top: element.CropTop, Right: element.CropRight, Bottom: element.CropBottom},
		BlipFillMode:              element.BlipFillMode,
		BlipTileOffsetX:           element.BlipTileOffsetX,
		BlipTileOffsetY:           element.BlipTileOffsetY,
		BlipTileScaleX:            element.BlipTileScaleX,
		BlipTileScaleY:            element.BlipTileScaleY,
		BlipTileFlip:              element.BlipTileFlip,
		BlipTileAlignment:         element.BlipTileAlignment,
		BlipCompressionState:      element.BlipCompressionState,
		FlipH:                     element.FlipH,
		FlipV:                     element.FlipV,
		HasAlphaModFix:            element.HasImageAlphaModFix,
		AlphaModFixPct:            element.ImageAlphaModFixPct,
		HasAlphaModulate:          element.HasImageAlphaModulate,
		AlphaModulatePct:          element.ImageAlphaModulatePct,
		HasAlphaBiLevel:           element.HasImageAlphaBiLevel,
		AlphaBiLevelThreshold:     element.ImageAlphaBiLevelThreshold,
		HasAlphaCeiling:           element.HasImageAlphaCeiling,
		HasAlphaFloor:             element.HasImageAlphaFloor,
		HasAlphaInverse:           element.HasImageAlphaInverse,
		HasAlphaReplace:           element.HasImageAlphaReplace,
		AlphaReplacePct:           element.ImageAlphaReplacePct,
		HasBiLevel:                element.HasImageBiLevel,
		BiLevelThreshold:          element.ImageBiLevelThreshold,
		HasGrayscale:              element.HasImageGrayscale,
		HasLuminance:              element.HasImageLuminance,
		LuminanceBright:           element.ImageLuminanceBright,
		LuminanceContrast:         element.ImageLuminanceContrast,
		HasHSL:                    element.HasImageHSL,
		HSLHue:                    element.ImageHSLHue,
		HSLSaturation:             element.ImageHSLSaturation,
		HSLLuminance:              element.ImageHSLLuminance,
		HasTint:                   element.HasImageTint,
		TintHue:                   element.ImageTintHue,
		TintAmount:                element.ImageTintAmount,
		HasSourceBlur:             element.HasImageBlur,
		SourceBlurRadius:          element.ImageBlurRadius,
		SourceBlurGrow:            element.ImageBlurGrow,
		HasSourceFillOverlay:      element.HasImageFillOverlay,
		SourceFillOverlay:         element.ImageFillOverlay,
		SourceFillOverlayBlend:    element.ImageFillOverlayBlend,
		HasColorChange:            element.HasImageColorChange,
		ColorChangeFrom:           element.ImageColorChangeFrom,
		ColorChangeTo:             element.ImageColorChangeTo,
		ColorChangeUseAlpha:       element.ImageColorChangeUseAlpha,
		HasColorReplace:           element.HasImageColorReplace,
		ColorReplace:              element.ImageColorReplace,
		HasDuotone:                element.HasImageDuotone,
		DuotoneDark:               element.ImageDuotoneDark,
		DuotoneLight:              element.ImageDuotoneLight,
		ImageUnsupported:          append([]string{}, element.ImageUnsupported...),
		RotationDegrees:           normalizedRotationDegrees(element.Rotation),
		RotatesWithShape:          pictureRotatesWithShape(element),
		HasSoftEdge:               element.HasSoftEdge,
		SoftEdgeRadius:            element.SoftEdgeRadius,
		HasBlur:                   element.HasBlur,
		BlurRadius:                element.BlurRadius,
		BlurGrow:                  element.BlurGrow,
		HasAlphaOutset:            element.HasAlphaOutset,
		AlphaOutsetRadius:         element.AlphaOutsetRadius,
		HasRelativeOffset:         element.HasRelativeOffset,
		RelativeOffsetX:           element.RelativeOffsetX,
		RelativeOffsetY:           element.RelativeOffsetY,
		HasEffectTransform:        element.HasEffectTransform,
		EffectTransformScaleX:     element.EffectTransformScaleX,
		EffectTransformScaleY:     element.EffectTransformScaleY,
		EffectTransformSkewX:      element.EffectTransformSkewX,
		EffectTransformSkewY:      element.EffectTransformSkewY,
		EffectTransformOffsetX:    element.EffectTransformOffsetX,
		EffectTransformOffsetY:    element.EffectTransformOffsetY,
		HasFillOverlay:            element.HasFillOverlay,
		FillOverlay:               element.FillOverlay,
		FillOverlayBlend:          element.FillOverlayBlend,
		HasInnerShadow:            element.HasInnerShadow,
		InnerShadowColor:          element.InnerShadowColor,
		InnerShadowBlur:           element.InnerShadowBlur,
		InnerShadowDistance:       element.InnerShadowDistance,
		InnerShadowDirection:      element.InnerShadowDirection,
		HasReflection:             element.HasReflection,
		ReflectionBlur:            element.ReflectionBlur,
		ReflectionStartAlpha:      element.ReflectionStartAlpha,
		ReflectionStartPosition:   element.ReflectionStartPosition,
		ReflectionEndAlpha:        element.ReflectionEndAlpha,
		ReflectionEndPosition:     element.ReflectionEndPosition,
		ReflectionDistance:        element.ReflectionDistance,
		ReflectionDirection:       element.ReflectionDirection,
		ReflectionFadeDirection:   element.ReflectionFadeDirection,
		ReflectionScaleX:          element.ReflectionScaleX,
		ReflectionScaleY:          element.ReflectionScaleY,
		ReflectionSkewX:           element.ReflectionSkewX,
		ReflectionSkewY:           element.ReflectionSkewY,
		ReflectionAlignment:       element.ReflectionAlignment,
		HasReflectionRotate:       element.HasReflectionRotate,
		ReflectionRotateWithShape: element.ReflectionRotateWithShape,
		HasGlow:                   element.HasGlow,
		GlowColor:                 element.GlowColor,
		GlowRadius:                element.GlowRadius,
		HasCustomMask:             len(element.CustomPath) >= 3,
		CustomPath:                append([]pathPoint{}, element.CustomPath...),
		CustomPathCommands:        append([]pathCommand{}, element.CustomPathCommands...),
		CustomPathUnsupported:     append([]string{}, element.CustomPathUnsupported...),
		CustomMaskPoints:          len(element.CustomPath),
		CustomMaskCommands:        len(element.CustomPathCommands),
		HasLine:                   element.HasLine,
		NoLine:                    element.NoLine,
		LineWidth:                 element.LineWidth,
		LineColor:                 element.LineColor,
		LineDash:                  element.LineDash,
		LineAlign:                 element.LineAlign,
		LineCap:                   element.LineCap,
		LineJoin:                  element.LineJoin,
		LineCompound:              element.LineCompound,
		HasShadow:                 element.HasShadow,
		ShadowColor:               element.ShadowColor,
		ShadowBlur:                element.ShadowBlur,
		ShadowDistance:            element.ShadowDistance,
		ShadowDirection:           element.ShadowDirection,
		HasShadowRotateWithShape:  element.HasShadowRotateWithShape,
		ShadowRotateWithShape:     element.ShadowRotateWithShape,
		HasShadowScaleX:           element.HasShadowScaleX,
		ShadowScaleX:              element.ShadowScaleX,
		HasShadowScaleY:           element.HasShadowScaleY,
		ShadowScaleY:              element.ShadowScaleY,
		HasShadowSkewX:            element.HasShadowSkewX,
		ShadowSkewX:               element.ShadowSkewX,
		HasShadowSkewY:            element.HasShadowSkewY,
		ShadowSkewY:               element.ShadowSkewY,
		HasShape3D:                element.HasShape3D,
		Shape3DFeatures:           append([]string{}, element.Shape3DFeatures...),
		EffectUnsupported:         append([]string{}, element.EffectUnsupported...),
	}, nil
}

func renderShapePrimitiveFromElement(sourcePart string, size slideSize, canvas image.Rectangle, element slideElement) renderShapePrimitive {
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	primitive := renderShapePrimitive{
		Provenance:          provenance,
		Target:              renderElementPixelBounds(element, size, canvas),
		FractionalTarget:    elementFractionalTarget(element, size, canvas),
		Geometry:            element.PrstGeom,
		GeometryAdjustments: cloneInt64Map(element.PrstGeomAdjustments),
		CustomPath:          renderPathPrimitiveFromElement(provenance, element),
		Fill:                renderFillPrimitiveFromElement(element),
		Stroke:              renderStrokePrimitiveFromElement(element),
		RotationDegrees:     normalizedRotationDegrees(element.Rotation),
		FlipH:               element.FlipH,
		FlipV:               element.FlipV,
		NonVisualLocks:      append([]string{}, element.NonVisualLocks...),
		Effect:              renderEffectPrimitiveFromElement(provenance, element),
		Unsupported:         renderPrimitiveUnsupportedMessages(element),
	}
	if element.Text != "" || len(element.TextParagraphs) > 0 {
		text := renderTextPrimitiveFromElement(provenance, element)
		primitive.Text = &text
	}
	return primitive
}

func renderConnectorPrimitiveFromElement(sourcePart string, size slideSize, canvas image.Rectangle, element slideElement) renderConnectorPrimitive {
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	startX, startY, endX, endY := lineEndpointsForElement(element, size, canvas)
	return renderConnectorPrimitive{
		Provenance:       provenance,
		Target:           renderElementPixelBounds(element, size, canvas),
		FractionalTarget: elementFractionalTarget(element, size, canvas),
		Geometry:         element.PrstGeom,
		NonVisualLocks:   append([]string{}, element.NonVisualLocks...),
		Stroke:           renderStrokePrimitiveFromElement(element),
		Start:            image.Point{X: startX, Y: startY},
		End:              image.Point{X: endX, Y: endY},
		Effect:           renderEffectPrimitiveFromElement(provenance, element),
		Unsupported:      renderPrimitiveUnsupportedMessages(element),
	}
}

func renderGraphicFramePrimitiveFromElement(sourcePart string, size slideSize, canvas image.Rectangle, element slideElement, relationships map[string]pptx.Relationship) (renderGraphicFramePrimitive, error) {
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	primitive := renderGraphicFramePrimitive{
		Provenance:       provenance,
		Target:           renderElementPixelBounds(element, size, canvas),
		FractionalTarget: elementFractionalTarget(element, size, canvas),
		PayloadKind:      element.GraphicPayloadKind,
		PayloadURI:       element.GraphicPayloadURI,
		RelationshipID:   element.PayloadRelationshipID,
		NonVisualLocks:   append([]string{}, element.NonVisualLocks...),
	}
	if element.PayloadRelationshipID != "" {
		if relationship, ok := relationships[element.PayloadRelationshipID]; ok {
			primitive.PayloadPart = pptx.ResolveTargetPart(sourcePart, relationship.Target)
			primitive.PayloadType = relationship.Type
			provenance.RelationshipIDs = appendRelationshipID(provenance.RelationshipIDs, element.PayloadRelationshipID)
			primitive.Provenance = provenance
		}
	}
	if element.Text != "" || len(element.TextParagraphs) > 0 {
		text := renderTextPrimitiveFromElement(provenance, element)
		primitive.Text = &text
	}
	if element.HasTable {
		table := renderTablePrimitiveFromElement(provenance, element)
		primitive.Table = &table
	}
	if element.DiagramDataID != "" {
		diagram, err := renderDiagramPrimitiveFromElement(sourcePart, provenance, element, relationships)
		if err != nil {
			primitive.Unsupported = append(primitive.Unsupported, err.Error())
			return primitive, err
		}
		primitive.Diagram = &diagram
	}
	return primitive, nil
}

func renderGroupPrimitiveFromElement(sourcePart string, size slideSize, canvas image.Rectangle, element slideElement) renderGroupPrimitive {
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	return renderGroupPrimitive{
		Provenance:       provenance,
		Target:           renderElementPixelBounds(element, size, canvas),
		FractionalTarget: elementFractionalTarget(element, size, canvas),
		NonVisualLocks:   append([]string{}, element.NonVisualLocks...),
		Unsupported:      renderPrimitiveUnsupportedMessages(element),
	}
}

func renderUnsupportedPrimitiveFromElement(sourcePart string, element slideElement) renderUnsupportedPrimitive {
	provenance := renderPrimitiveProvenanceForElement(sourcePart, element, schemaAnchorsForRenderElement(element))
	reason := fmt.Sprintf("%s object %q is not lowered into a supported render primitive", objectKindLabel(element.Kind), elementLabel(element))
	if element.GraphicPayloadKind != "" {
		reason = fmt.Sprintf("%s object %q preserves unsupported %s payload", objectKindLabel(element.Kind), elementLabel(element), element.GraphicPayloadKind)
	}
	return renderUnsupportedPrimitive{
		Provenance: provenance,
		Reason:     reason,
	}
}

func renderPathPrimitiveFromElement(provenance renderPrimitiveProvenance, element slideElement) renderPathPrimitive {
	pathProvenance := provenance
	pathProvenance.SchemaAnchors = []string{"dml-main.xsd:2062 CT_PresetGeometry2D", "dml-main.xsd:2074 CT_CustomGeometry2D", "dml-main.xsd:2042 CT_Path2D"}
	return renderPathPrimitive{
		Provenance:    pathProvenance,
		Preset:        element.PrstGeom,
		Points:        append([]pathPoint{}, element.CustomPath...),
		Subpaths:      clonePathPointSlices(element.CustomPaths),
		Fills:         append([]bool{}, element.CustomPathFills...),
		Strokes:       append([]bool{}, element.CustomPathStrokes...),
		Commands:      append([]pathCommand{}, element.CustomPathCommands...),
		Unsupported:   append([]string{}, element.CustomPathUnsupported...),
		SchemaAnchors: append([]string{}, pathProvenance.SchemaAnchors...),
	}
}

func renderTextPrimitiveFromElement(provenance renderPrimitiveProvenance, element slideElement) renderTextPrimitive {
	textProvenance := provenance
	textProvenance.SchemaAnchors = []string{"dml-main.xsd:2653 CT_TextBody", "dml-main.xsd:2625 CT_TextBodyProperties"}
	return renderTextPrimitive{
		Provenance:                 textProvenance,
		Text:                       element.Text,
		Paragraphs:                 append([]textParagraph{}, element.TextParagraphs...),
		FontFamily:                 element.FontFamily,
		FontSize:                   element.FontSize,
		FontPointScale:             element.FontPointScale,
		Italic:                     element.Italic,
		HasTextColor:               element.HasTextColor,
		TextColor:                  element.TextColor,
		Align:                      element.TextAlign,
		IsTextBox:                  element.IsTextBox,
		Anchor:                     element.TextAnchor,
		Wrap:                       element.TextWrap,
		HorizontalOverflow:         element.TextHorizontalOverflow,
		VerticalOverflow:           element.TextVerticalOverflow,
		Insets:                     renderTextInsets{Left: element.InsetLeft, Top: element.InsetTop, Right: element.InsetRight, Bottom: element.InsetBottom},
		HasShapeAutofit:            element.HasShapeAutofit,
		HasNormAutofit:             element.HasNormAutofit,
		HasNoAutofit:               element.HasNoAutofit,
		HasFontScalePct:            element.HasFontScalePct,
		FontScalePct:               element.FontScalePct,
		HasLineSpacingReductionPct: element.HasLineSpacingReductionPct,
		LineSpacingReductionPct:    element.LineSpacingReductionPct,
		HasFirstLastSpacing:        element.HasFirstLastSpacing,
		IncludeFirstLastSpacing:    element.IncludeFirstLastSpacing,
		HasTextVertical:            element.HasTextVertical,
		TextVertical:               element.TextVertical,
		HasTextBodyRotation:        element.HasTextBodyRotation,
		TextBodyRotation:           element.TextBodyRotation,
		HasTextColumns:             element.HasTextColumns,
		TextColumnCount:            element.TextColumnCount,
		HasRTLColumns:              element.HasTextRightToLeftColumns,
		RTLColumns:                 element.TextRightToLeftColumns,
		FontResolution:             fontResolutionUnsupportedMessages(element),
		Unsupported:                staticTextUnsupportedMessages(element),
		SchemaAnchors:              append([]string{}, textProvenance.SchemaAnchors...),
	}
}

func renderTablePrimitiveFromElement(provenance renderPrimitiveProvenance, element slideElement) renderTablePrimitive {
	tableProvenance := provenance
	tableProvenance.SchemaAnchors = []string{"dml-main.xsd:2423 CT_Table", "pml.xsd:1263 CT_GraphicalObjectFrame"}
	return renderTablePrimitive{
		Provenance:          tableProvenance,
		Columns:             append([]int64{}, element.Table.Columns...),
		ColumnIDs:           append([]string{}, element.Table.ColumnIDs...),
		Rows:                append([]tableRow{}, element.Table.Rows...),
		StyleID:             element.Table.StyleID,
		FirstRow:            element.Table.FirstRow,
		FirstCol:            element.Table.FirstCol,
		LastRow:             element.Table.LastRow,
		LastCol:             element.Table.LastCol,
		BandRow:             element.Table.BandRow,
		BandCol:             element.Table.BandCol,
		UnsupportedFeatures: append([]string{}, element.Table.UnsupportedFeatures...),
		SchemaAnchors:       append([]string{}, tableProvenance.SchemaAnchors...),
	}
}

func renderDiagramPrimitiveFromElement(sourcePart string, provenance renderPrimitiveProvenance, element slideElement, relationships map[string]pptx.Relationship) (renderDiagramPrimitive, error) {
	diagramProvenance := provenance
	diagramProvenance.SchemaAnchors = []string{"pml.xsd:1263 CT_GraphicalObjectFrame", "dml-diagram.xsd:387 CT_RelIds", "dml-diagram.xsd:393 relIds"}
	diagramProvenance.RelationshipIDs = appendRelationshipID(diagramProvenance.RelationshipIDs, element.DiagramDataID)
	primitive := renderDiagramPrimitive{
		Provenance:    diagramProvenance,
		DataRelID:     element.DiagramDataID,
		SchemaAnchors: append([]string{}, diagramProvenance.SchemaAnchors...),
	}
	relationship, ok := relationships[element.DiagramDataID]
	if !ok {
		return primitive, fmt.Errorf("diagram object %q references missing relationship %q", elementLabel(element), element.DiagramDataID)
	}
	if relationship.Type != diagramDataRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return primitive, fmt.Errorf("diagram object %q uses unsupported relationship %q", elementLabel(element), relationship.Type)
	}
	primitive.DataPart = pptx.ResolveTargetPart(sourcePart, relationship.Target)
	return primitive, nil
}

func renderEffectPrimitiveFromElement(provenance renderPrimitiveProvenance, element slideElement) *renderEffectPrimitive {
	if !element.HasEffectProperties && !element.HasShadow && !element.HasInnerShadow && !element.HasReflection && !element.HasSoftEdge && !element.HasBlur && !element.HasAlphaOutset && !element.HasRelativeOffset && !element.HasEffectTransform && !element.HasFillOverlay && !element.HasGlow && !element.HasShape3D {
		return nil
	}
	effectProvenance := provenance
	effectProvenance.SchemaAnchors = []string{"dml-main.xsd:1671 CT_EffectList", "dml-main.xsd:1689 CT_EffectProperties"}
	return &renderEffectPrimitive{
		Provenance:                effectProvenance,
		HasEffectProperties:       element.HasEffectProperties,
		HasShadow:                 element.HasShadow,
		ShadowColor:               element.ShadowColor,
		ShadowBlur:                element.ShadowBlur,
		ShadowDistance:            element.ShadowDistance,
		ShadowDirection:           element.ShadowDirection,
		ShadowAlignment:           element.ShadowAlignment,
		HasShadowRotateWithShape:  element.HasShadowRotateWithShape,
		ShadowRotateWithShape:     element.ShadowRotateWithShape,
		HasShadowScaleX:           element.HasShadowScaleX,
		ShadowScaleX:              element.ShadowScaleX,
		HasShadowScaleY:           element.HasShadowScaleY,
		ShadowScaleY:              element.ShadowScaleY,
		HasShadowSkewX:            element.HasShadowSkewX,
		ShadowSkewX:               element.ShadowSkewX,
		HasShadowSkewY:            element.HasShadowSkewY,
		ShadowSkewY:               element.ShadowSkewY,
		HasInnerShadow:            element.HasInnerShadow,
		InnerShadowColor:          element.InnerShadowColor,
		InnerShadowBlur:           element.InnerShadowBlur,
		InnerShadowDistance:       element.InnerShadowDistance,
		InnerShadowDirection:      element.InnerShadowDirection,
		HasReflection:             element.HasReflection,
		ReflectionBlur:            element.ReflectionBlur,
		ReflectionStartAlpha:      element.ReflectionStartAlpha,
		ReflectionStartPosition:   element.ReflectionStartPosition,
		ReflectionEndAlpha:        element.ReflectionEndAlpha,
		ReflectionEndPosition:     element.ReflectionEndPosition,
		ReflectionDistance:        element.ReflectionDistance,
		ReflectionDirection:       element.ReflectionDirection,
		ReflectionFadeDirection:   element.ReflectionFadeDirection,
		ReflectionScaleX:          element.ReflectionScaleX,
		ReflectionScaleY:          element.ReflectionScaleY,
		ReflectionSkewX:           element.ReflectionSkewX,
		ReflectionSkewY:           element.ReflectionSkewY,
		ReflectionAlignment:       element.ReflectionAlignment,
		HasReflectionRotate:       element.HasReflectionRotate,
		ReflectionRotateWithShape: element.ReflectionRotateWithShape,
		HasSoftEdge:               element.HasSoftEdge,
		SoftEdgeRadius:            element.SoftEdgeRadius,
		HasBlur:                   element.HasBlur,
		BlurRadius:                element.BlurRadius,
		BlurGrow:                  element.BlurGrow,
		HasAlphaOutset:            element.HasAlphaOutset,
		AlphaOutsetRadius:         element.AlphaOutsetRadius,
		HasRelativeOffset:         element.HasRelativeOffset,
		RelativeOffsetX:           element.RelativeOffsetX,
		RelativeOffsetY:           element.RelativeOffsetY,
		HasEffectTransform:        element.HasEffectTransform,
		EffectTransformScaleX:     element.EffectTransformScaleX,
		EffectTransformScaleY:     element.EffectTransformScaleY,
		EffectTransformSkewX:      element.EffectTransformSkewX,
		EffectTransformSkewY:      element.EffectTransformSkewY,
		EffectTransformOffsetX:    element.EffectTransformOffsetX,
		EffectTransformOffsetY:    element.EffectTransformOffsetY,
		HasFillOverlay:            element.HasFillOverlay,
		FillOverlay:               element.FillOverlay,
		FillOverlayBlend:          element.FillOverlayBlend,
		HasGlow:                   element.HasGlow,
		GlowColor:                 element.GlowColor,
		GlowRadius:                element.GlowRadius,
		HasShape3D:                element.HasShape3D,
		Shape3DFeatures:           append([]string{}, element.Shape3DFeatures...),
		Unsupported:               renderPrimitiveUnsupportedMessages(element),
		SchemaAnchors:             append([]string{}, effectProvenance.SchemaAnchors...),
	}
}

func renderFillPrimitiveFromElement(element slideElement) renderFillPrimitive {
	return renderFillPrimitive{
		HasFill:       element.HasFill,
		Color:         element.FillColor,
		NoFill:        element.NoFill,
		HasGradient:   element.HasFillGradient,
		Gradient:      element.FillGradient,
		HasPattern:    element.HasPatternFill,
		Pattern:       element.PatternFill,
		Unsupported:   append([]string{}, element.PaintUnsupported...),
		SchemaAnchors: []string{"dml-main.xsd:1391 CT_SolidColorFillProperties", "dml-main.xsd:1438 CT_GradientFillProperties", "dml-main.xsd:1569 CT_PatternFillProperties", "dml-main.xsd:1579 EG_FillProperties"},
	}
}

func renderStrokePrimitiveFromElement(element slideElement) renderStrokePrimitive {
	return renderStrokePrimitive{
		HasLine:          element.HasLine,
		NoLine:           element.NoLine,
		Color:            element.LineColor,
		Width:            element.LineWidth,
		HasWidth:         element.HasLineWidth,
		Dash:             element.LineDash,
		HasDash:          element.HasLineDash,
		Cap:              element.LineCap,
		HasCap:           element.HasLineCap,
		Align:            element.LineAlign,
		HasAlign:         element.HasLineAlign,
		HasMarker:        element.HasLineMarker,
		HeadMarker:       element.HeadLineMarker,
		HeadMarkerWidth:  element.HeadLineMarkerWidth,
		HeadMarkerLength: element.HeadLineMarkerLength,
		TailMarker:       element.TailLineMarker,
		TailMarkerWidth:  element.TailLineMarkerWidth,
		TailMarkerLength: element.TailLineMarkerLength,
		Join:             element.LineJoin,
		HasJoin:          element.HasLineJoin,
		Compound:         element.LineCompound,
		HasCompound:      element.HasLineCompound,
	}
}

func clonePathPointSlices(paths [][]pathPoint) [][]pathPoint {
	if len(paths) == 0 {
		return nil
	}
	cloned := make([][]pathPoint, 0, len(paths))
	for _, path := range paths {
		cloned = append(cloned, append([]pathPoint{}, path...))
	}
	return cloned
}

func renderPrimitiveUnsupportedMessages(element slideElement) []string {
	var messages []string
	messages = append(messages, element.CustomPathUnsupported...)
	messages = append(messages, element.ImageUnsupported...)
	messages = append(messages, element.EffectUnsupported...)
	if element.HasShape3D {
		messages = append(messages, shape3DUnsupportedMessages(element)...)
	}
	if element.HasShadow {
		messages = append(messages, shadowTransformUnsupportedMessages(element)...)
	}
	return messages
}

func renderPrimitiveProvenanceForElement(sourcePart string, element slideElement, schemaAnchors []string) renderPrimitiveProvenance {
	return renderPrimitiveProvenance{
		ObjectKind:          element.Kind,
		ID:                  element.ID,
		Name:                element.Name,
		Description:         element.Description,
		Title:               element.Title,
		CreationID:          element.CreationID,
		NonVisualProperties: append([]string{}, element.NonVisualProperties...),
		SourcePart:          sourcePart,
		XMLPath:             renderPrimitiveXMLPath(element),
		SchemaAnchors:       append([]string{}, schemaAnchors...),
	}
}

func renderPrimitiveXMLPath(element slideElement) string {
	if element.Kind == "" {
		return ""
	}
	if element.ID != "" {
		return fmt.Sprintf(`/p:sld/p:cSld/p:spTree/p:%s[.//p:cNvPr/@id="%s"]`, element.Kind, element.ID)
	}
	if element.Name != "" {
		return fmt.Sprintf(`/p:sld/p:cSld/p:spTree/p:%s[.//p:cNvPr/@name="%s"]`, element.Kind, element.Name)
	}
	return fmt.Sprintf("/p:sld/p:cSld/p:spTree/p:%s", element.Kind)
}

func schemaAnchorsForRenderElement(element slideElement) []string {
	switch element.Kind {
	case "pic":
		return []string{"pml.xsd:1245 CT_Picture", "dml-picture.xsd:14 CT_Picture", "dml-main.xsd:1502 CT_BlipFillProperties", "dml-main.xsd:2223 CT_ShapeProperties"}
	case "sp":
		anchors := []string{"pml.xsd:1209 CT_Shape", "dml-main.xsd:2223 CT_ShapeProperties"}
		if element.HasLine {
			anchors = append(anchors, "dml-main.xsd:2206 CT_LineProperties")
		}
		if element.Text != "" || len(element.TextParagraphs) > 0 {
			anchors = append(anchors, "dml-main.xsd:2653 CT_TextBody")
		}
		return anchors
	case "cxnSp":
		return []string{"pml.xsd:1228 CT_Connector", "dml-main.xsd:2223 CT_ShapeProperties", "dml-main.xsd:2206 CT_LineProperties"}
	case "graphicFrame":
		anchors := []string{"pml.xsd:1263 CT_GraphicalObjectFrame", "dml-main.xsd:848 CT_GraphicalObject"}
		if element.HasTable {
			anchors = append(anchors, "dml-main.xsd:2423 CT_Table")
		}
		if element.DiagramDataID != "" {
			anchors = append(anchors, "dml-diagram.xsd:393 relIds")
		}
		switch element.GraphicPayloadKind {
		case "chart":
			anchors = append(anchors, "dml-chart.xsd:1448 chart")
		case "unknown graphic payload":
			anchors = append(anchors, "dml-main.xsd:842 CT_GraphicalObjectData")
		}
		return anchors
	case "contentPart":
		return []string{"pml.xsd:1293 CT_Rel", "pml.xsd:1297 contentPart"}
	case "oleObj":
		return []string{"pml.xsd:840 CT_OleObject", "pml.xsd:851 oleObj"}
	case "control":
		return []string{"pml.xsd:852 CT_Control", "pml.xsd:859 CT_ControlList"}
	case "audio", "video", "audioFile", "videoFile":
		return []string{"pml.xsd:592 CT_TLCommonMediaNodeData", "pml.xsd:602 CT_TLMediaNodeAudio", "pml.xsd:608 CT_TLMediaNodeVideo"}
	case "grpSp":
		return []string{"pml.xsd:1282 CT_GroupShape", "dml-main.xsd:2236 CT_GroupShapeProperties"}
	default:
		return []string{"pml.xsd:1282 CT_GroupShape"}
	}
}

func appendRelationshipID(ids []string, id string) []string {
	if id == "" {
		return ids
	}
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

func cloneInt64Map(values map[string]int64) map[string]int64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]int64, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func objectPixelBoundsFromImageRect(rect image.Rectangle) ObjectPixelBounds {
	return ObjectPixelBounds{MinX: rect.Min.X, MinY: rect.Min.Y, MaxX: rect.Max.X - 1, MaxY: rect.Max.Y - 1}
}
