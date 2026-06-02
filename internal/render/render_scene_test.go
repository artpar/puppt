package render

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

func TestRenderPicturePrimitiveFromElementPreservesResolvedSourceFields(t *testing.T) {
	pkg := &pptx.Package{
		ContentTypes: pptx.ContentTypes{
			Defaults: map[string]string{"png": "image/png"},
		},
	}
	element := slideElement{
		Kind:                  "pic",
		ID:                    "1028",
		Name:                  "Picture 4",
		Description:           "Scale Up Icons",
		Title:                 "Icon",
		CreationID:            "{PICTURE-CREATION}",
		NonVisualProperties:   []string{"decorative=true", "hidden=false"},
		NonVisualLocks:        []string{"picLocks.noChangeArrowheads", "picLocks.noChangeAspect"},
		EmbedID:               "rId5",
		OffX:                  8595453,
		OffY:                  4567248,
		ExtCX:                 1419721,
		ExtCY:                 1419721,
		HasTransform:          true,
		CropLeft:              1000,
		CropTop:               2000,
		CropRight:             3000,
		CropBottom:            4000,
		FlipH:                 true,
		HasImageAlphaModFix:   true,
		ImageAlphaModFixPct:   50000,
		BlipCompressionState:  "print",
		HasBlipRotWithShape:   true,
		BlipRotWithShape:      false,
		HasRotation:           true,
		Rotation:              5400000,
		HasSoftEdge:           true,
		SoftEdgeRadius:        12700,
		CustomPath:            []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
		CustomPathCommands:    []pathCommand{{Kind: "moveTo"}, {Kind: "lnTo"}, {Kind: "close"}},
		CustomPathUnsupported: []string{"custom path arc command was not rendered"},
		HasLine:               true,
		LineWidth:             12700,
		LineColor:             color.RGBA{B: 255, A: 255},
		LineDash:              "dash",
		LineAlign:             "ctr",
		LineCap:               "rnd",
		HasShadow:             true,
		ShadowColor:           color.RGBA{R: 10, G: 20, B: 30, A: 128},
		ShadowBlur:            2000,
		ShadowDistance:        3000,
		ShadowDirection:       5400000,
		HasShadowScaleX:       true,
		ShadowScaleX:          90000,
		HasShape3D:            true,
		Shape3DFeatures:       []string{"bevelT"},
	}
	primitive, err := renderPicturePrimitiveFromElement(pkg, "ppt/slides/slide15.xml", slideSize{CX: defaultSlideCX, CY: defaultSlideCY}, image.Rect(0, 0, 960, 540), element, map[string]pptx.Relationship{
		"rId5": {ID: "rId5", Type: pptx.ImageRelType, Target: "../media/image17.png"},
	})
	if err != nil {
		t.Fatalf("build picture primitive: %v", err)
	}

	if primitive.ObjectKind != "pic" || primitive.ID != "1028" || primitive.Name != "Picture 4" || primitive.RelationshipID != "rId5" {
		t.Fatalf("primitive did not preserve object identity: %+v", primitive)
	}
	if len(primitive.NonVisualLocks) != 2 || primitive.NonVisualLocks[0] != "picLocks.noChangeArrowheads" || primitive.NonVisualLocks[1] != "picLocks.noChangeAspect" {
		t.Fatalf("primitive did not preserve picture lock metadata: %+v", primitive.NonVisualLocks)
	}
	if primitive.Provenance.SourcePart != "ppt/slides/slide15.xml" || len(primitive.Provenance.SchemaAnchors) == 0 || primitive.Provenance.XMLPath == "" {
		t.Fatalf("primitive did not preserve provenance: %+v", primitive.Provenance)
	}
	if primitive.Provenance.Description != "Scale Up Icons" || primitive.Provenance.Title != "Icon" {
		t.Fatalf("primitive did not preserve cNvPr description/title provenance: %+v", primitive.Provenance)
	}
	if primitive.Provenance.CreationID != "{PICTURE-CREATION}" {
		t.Fatalf("primitive did not preserve cNvPr creationId provenance: %+v", primitive.Provenance)
	}
	if len(primitive.Provenance.NonVisualProperties) != 2 || primitive.Provenance.NonVisualProperties[0] != "decorative=true" || primitive.Provenance.NonVisualProperties[1] != "hidden=false" {
		t.Fatalf("primitive did not preserve cNvPr boolean provenance: %+v", primitive.Provenance.NonVisualProperties)
	}
	if primitive.SourcePart != "ppt/slides/slide15.xml" || primitive.MediaPart != "ppt/media/image17.png" || primitive.ContentType != "image/png" {
		t.Fatalf("primitive did not resolve package source fields: %+v", primitive)
	}
	if primitive.Target != (ObjectPixelBounds{MinX: 677, MinY: 360, MaxX: 788, MaxY: 470}) {
		t.Fatalf("unexpected integer target: %+v", primitive.Target)
	}
	if primitive.Crop != (relativeRect{Left: 1000, Top: 2000, Right: 3000, Bottom: 4000}) {
		t.Fatalf("primitive did not preserve crop percentages: %+v", primitive.Crop)
	}
	if !primitive.FlipH || primitive.FlipV || !primitive.HasAlphaModFix || primitive.AlphaModFixPct != 50000 {
		t.Fatalf("primitive did not preserve transform/effect fields: %+v", primitive)
	}
	if primitive.BlipCompressionState != "print" {
		t.Fatalf("primitive did not preserve blip compression metadata: %+v", primitive)
	}
	if primitive.RotationDegrees != 90 || primitive.RotatesWithShape {
		t.Fatalf("primitive did not normalize rotation/rotWithShape: %+v", primitive)
	}
	if !primitive.HasSoftEdge || primitive.SoftEdgeRadius != 12700 || !primitive.HasCustomMask || primitive.CustomMaskPoints != 3 || primitive.CustomMaskCommands != 3 {
		t.Fatalf("primitive did not preserve mask/soft-edge fields: %+v", primitive)
	}
	if len(primitive.CustomPathUnsupported) != 1 || primitive.CustomPathUnsupported[0] == "" {
		t.Fatalf("primitive did not preserve custom path unsupported fields: %+v", primitive)
	}
	if !primitive.HasLine || primitive.NoLine || primitive.LineWidth != 12700 || primitive.LineColor != (color.RGBA{B: 255, A: 255}) || primitive.LineDash != "dash" || primitive.LineAlign != "ctr" || primitive.LineCap != "rnd" {
		t.Fatalf("primitive did not preserve line fields: %+v", primitive)
	}
	if !primitive.HasShadow || primitive.ShadowColor != (color.RGBA{R: 10, G: 20, B: 30, A: 128}) || primitive.ShadowBlur != 2000 || primitive.ShadowDistance != 3000 || primitive.ShadowDirection != 5400000 || !primitive.HasShadowScaleX || primitive.ShadowScaleX != 90000 {
		t.Fatalf("primitive did not preserve shadow fields: %+v", primitive)
	}
	if !primitive.HasShape3D || len(primitive.Shape3DFeatures) != 1 || primitive.Shape3DFeatures[0] != "bevelT" {
		t.Fatalf("primitive did not preserve shape 3-D fields: %+v", primitive)
	}
}

func TestRenderPicturePrimitiveFromElementAllowsShapeBlipFill(t *testing.T) {
	pkg := &pptx.Package{ContentTypes: pptx.ContentTypes{Defaults: map[string]string{"png": "image/png"}}}
	primitive, err := renderPicturePrimitiveFromElement(pkg, "ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), slideElement{
		Kind:         "sp",
		ID:           "7",
		Name:         "Shape Picture Fill",
		EmbedID:      "rIdImg",
		HasTransform: true,
		ExtCX:        500,
		ExtCY:        500,
	}, map[string]pptx.Relationship{
		"rIdImg": {ID: "rIdImg", Type: pptx.ImageRelType, Target: "../media/fill.png"},
	})
	if err != nil {
		t.Fatalf("build shape blip-fill primitive: %v", err)
	}
	if primitive.ObjectKind != "sp" || primitive.MediaPart != "ppt/media/fill.png" {
		t.Fatalf("unexpected shape blip-fill primitive: %+v", primitive)
	}
}

func TestCurrentPictureBackendUsesSamplingStage(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 2, 2))
	sampler := &recordingPictureSampler{paint: color.RGBA{R: 12, G: 34, B: 56, A: 255}}
	unsupported := currentPictureBackend{sampler: sampler}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: 1000, CY: 1000},
		Canvas:    canvas,
		Primitive: renderPicturePrimitive{
			ObjectKind:       "pic",
			ID:               "7",
			Name:             "Picture",
			Target:           ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1},
			RotatesWithShape: true,
		},
		Source: image.NewRGBA(image.Rect(0, 0, 1, 1)),
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected no unsupported records, got %+v", unsupported)
	}
	if sampler.calls != 1 {
		t.Fatalf("expected backend to call sampling stage once, got %d", sampler.calls)
	}
	if got := canvas.RGBAAt(0, 0); got != sampler.paint {
		t.Fatalf("expected sampling stage paint, got %#v want %#v", got, sampler.paint)
	}
	if sampler.last.Primitive.ID != "7" || sampler.last.Target != image.Rect(0, 0, 2, 2) || sampler.last.OutputWidth != 2 {
		t.Fatalf("backend did not pass primitive sampling input: %+v", sampler.last)
	}
}

type recordingPictureSampler struct {
	calls int
	last  pictureSamplingInput
	paint color.RGBA
}

func (sampler *recordingPictureSampler) Draw(input pictureSamplingInput) bool {
	sampler.calls++
	sampler.last = input
	input.Canvas.SetRGBA(input.Target.Min.X, input.Target.Min.Y, sampler.paint)
	return false
}

func TestRenderSceneFromElementsKeepsPictureZOrderAndErrors(t *testing.T) {
	pkg := &pptx.Package{ContentTypes: pptx.ContentTypes{Defaults: map[string]string{"png": "image/png"}}}
	elements := []slideElement{
		{Kind: "sp", ID: "2", Name: "Title", HasTransform: true, ExtCX: 1000, ExtCY: 1000},
		{Kind: "pic", ID: "3", Name: "Missing Relationship", EmbedID: "missing", HasTransform: true, ExtCX: 1000, ExtCY: 1000},
		{Kind: "pic", ID: "4", Name: "Picture", EmbedID: "rId1", HasTransform: true, ExtCX: 1000, ExtCY: 1000},
	}
	scene, errs := renderSceneFromElements(pkg, "ppt/slides/slide1.xml", "ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), elements, map[string]pptx.Relationship{
		"rId1": {ID: "rId1", Type: pptx.ImageRelType, Target: "../media/object.png"},
	})
	if len(errs) != 1 {
		t.Fatalf("expected one primitive conversion error, got %d: %v", len(errs), errs)
	}
	if len(scene.Primitives) != 2 {
		t.Fatalf("expected shape plus picture primitive, got %+v", scene.Primitives)
	}
	if scene.Primitives[0].Kind != renderPrimitiveShape || scene.Primitives[0].ZOrder != 1 || scene.Primitives[0].Shape == nil {
		t.Fatalf("unexpected first scene primitive: %+v", scene.Primitives[0])
	}
	if scene.Primitives[1].Kind != renderPrimitivePicture || scene.Primitives[1].ZOrder != 3 || scene.Primitives[1].Picture == nil {
		t.Fatalf("unexpected second scene primitive: %+v", scene.Primitives[1])
	}
	if scene.Primitives[1].Picture.MediaPart != "ppt/media/object.png" {
		t.Fatalf("unexpected picture media part: %+v", scene.Primitives[1].Picture)
	}
}

func TestRenderShapePrimitiveFromElementPreservesGeometryTextAndEffects(t *testing.T) {
	element := slideElement{
		Kind:                       "sp",
		ID:                         "6",
		Name:                       "Rectangle 5",
		Description:                "Annotated rectangle",
		CreationID:                 "{SHAPE-CREATION}",
		NonVisualProperties:        []string{"hidden=true"},
		HasTransform:               true,
		OffX:                       10,
		OffY:                       20,
		ExtCX:                      300,
		ExtCY:                      400,
		PrstGeom:                   "rect",
		PrstGeomAdjustments:        map[string]int64{"adj": 25000},
		HasFill:                    true,
		FillColor:                  color.RGBA{R: 1, G: 2, B: 3, A: 255},
		HasLine:                    true,
		HasLineWidth:               true,
		LineWidth:                  12700,
		LineColor:                  color.RGBA{R: 4, G: 5, B: 6, A: 255},
		HasLineMarker:              true,
		TailLineMarker:             "triangle",
		Text:                       "Ordering Test Kits",
		IsTextBox:                  true,
		NonVisualLocks:             []string{"spLocks.noTextEdit"},
		TextParagraphs:             []textParagraph{{Text: "Ordering Test Kits", FontSize: 4000, HasRTL: true, HasEALineBreak: true, EALineBreak: true, HasLatinLineBreak: true, HasHangingPunct: true, HangingPunct: true}},
		FontFamily:                 "Calibri",
		FontSize:                   40,
		TextAnchor:                 "ctr",
		HasShapeAutofit:            true,
		HasNormAutofit:             true,
		HasFontScalePct:            true,
		FontScalePct:               85000,
		HasLineSpacingReductionPct: true,
		LineSpacingReductionPct:    20000,
		HasFirstLastSpacing:        true,
		IncludeFirstLastSpacing:    false,
		HasEffectProperties:        true,
		HasShadow:                  true,
		ShadowColor:                color.RGBA{A: 128},
		ShadowBlur:                 12700,
		HasSoftEdge:                true,
		SoftEdgeRadius:             6350,
		HasShape3D:                 true,
		Shape3DFeatures:            []string{"bevelT"},
		CustomPath:                 []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}},
		CustomPathCommands:         []pathCommand{{Kind: "moveTo"}, {Kind: "lnTo"}, {Kind: "close"}},
		CustomPathUnsupported:      []string{"custom path arc command was not rendered"},
		HasTextVerticalOverflow:    true,
		TextVerticalOverflow:       "clip",
	}
	primitive := renderShapePrimitiveFromElement("ppt/slides/slide12.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), element)
	if primitive.Provenance.ID != "6" || primitive.Provenance.SourcePart != "ppt/slides/slide12.xml" || primitive.Provenance.XMLPath == "" || primitive.Provenance.Description != "Annotated rectangle" || primitive.Provenance.CreationID != "{SHAPE-CREATION}" {
		t.Fatalf("shape primitive did not preserve provenance: %+v", primitive.Provenance)
	}
	if len(primitive.Provenance.NonVisualProperties) != 1 || primitive.Provenance.NonVisualProperties[0] != "hidden=true" {
		t.Fatalf("shape primitive did not preserve cNvPr boolean provenance: %+v", primitive.Provenance.NonVisualProperties)
	}
	if primitive.Geometry != "rect" || primitive.GeometryAdjustments["adj"] != 25000 || len(primitive.CustomPath.Points) != 3 || len(primitive.CustomPath.Commands) != 3 {
		t.Fatalf("shape primitive did not preserve geometry/path: %+v", primitive)
	}
	if len(primitive.NonVisualLocks) != 1 || primitive.NonVisualLocks[0] != "spLocks.noTextEdit" {
		t.Fatalf("shape primitive did not preserve lock metadata: %+v", primitive.NonVisualLocks)
	}
	if !primitive.Fill.HasFill || primitive.Fill.Color != (color.RGBA{R: 1, G: 2, B: 3, A: 255}) {
		t.Fatalf("shape primitive did not preserve fill: %+v", primitive.Fill)
	}
	if !primitive.Stroke.HasLine || primitive.Stroke.Width != 12700 || primitive.Stroke.TailMarker != "triangle" {
		t.Fatalf("shape primitive did not preserve stroke: %+v", primitive.Stroke)
	}
	if primitive.Text == nil || primitive.Text.Text != "Ordering Test Kits" || !primitive.Text.IsTextBox || primitive.Text.Anchor != "ctr" || primitive.Text.VerticalOverflow != "clip" || !primitive.Text.HasShapeAutofit {
		t.Fatalf("shape primitive did not preserve text: %+v", primitive.Text)
	}
	if !primitive.Text.HasNormAutofit || !primitive.Text.HasFontScalePct || primitive.Text.FontScalePct != 85000 || !primitive.Text.HasLineSpacingReductionPct || primitive.Text.LineSpacingReductionPct != 20000 || !primitive.Text.HasFirstLastSpacing || primitive.Text.IncludeFirstLastSpacing {
		t.Fatalf("shape primitive did not preserve text body autofit/spacing properties: %+v", primitive.Text)
	}
	if len(primitive.Text.Paragraphs) != 1 || !primitive.Text.Paragraphs[0].HasRTL || primitive.Text.Paragraphs[0].RTL || !primitive.Text.Paragraphs[0].HasLatinLineBreak || primitive.Text.Paragraphs[0].LatinLineBreak {
		t.Fatalf("shape primitive did not preserve paragraph property flags: %+v", primitive.Text.Paragraphs)
	}
	if primitive.Effect == nil || !primitive.Effect.HasShadow || !primitive.Effect.HasSoftEdge || !primitive.Effect.HasShape3D || len(primitive.Effect.Shape3DFeatures) != 1 {
		t.Fatalf("shape primitive did not preserve effects: %+v", primitive.Effect)
	}
	if len(primitive.Unsupported) == 0 {
		t.Fatalf("shape primitive should preserve unsupported records")
	}
}

func TestRenderConnectorPrimitiveFromElementPreservesLineEndpointsAndMarkers(t *testing.T) {
	element := slideElement{
		Kind:           "cxnSp",
		ID:             "32",
		Name:           "Straight Connector 31",
		HasTransform:   true,
		OffX:           0,
		OffY:           0,
		ExtCX:          1000,
		ExtCY:          500,
		PrstGeom:       "line",
		FlipH:          true,
		HasLine:        true,
		LineWidth:      12700,
		LineColor:      color.RGBA{R: 255, A: 255},
		HasLineMarker:  true,
		HeadLineMarker: "triangle",
		TailLineMarker: "diamond",
	}
	primitive := renderConnectorPrimitiveFromElement("ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), element)
	if primitive.Provenance.ID != "32" || primitive.Geometry != "line" {
		t.Fatalf("connector primitive did not preserve identity/geometry: %+v", primitive)
	}
	if primitive.Start != (image.Point{X: 100, Y: 0}) || primitive.End != (image.Point{X: 0, Y: 50}) {
		t.Fatalf("connector primitive did not preserve transformed endpoints: start=%+v end=%+v", primitive.Start, primitive.End)
	}
	if !primitive.Stroke.HasLine || primitive.Stroke.HeadMarker != "triangle" || primitive.Stroke.TailMarker != "diamond" {
		t.Fatalf("connector primitive did not preserve stroke markers: %+v", primitive.Stroke)
	}
}

func TestRenderGraphicFramePrimitiveFromElementPreservesTableAndDiagramErrors(t *testing.T) {
	tableFrame, err := renderGraphicFramePrimitiveFromElement("ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), slideElement{
		Kind:       "graphicFrame",
		ID:         "9",
		Name:       "Table",
		Title:      "Table title",
		CreationID: "{TABLE-CREATION}",
		NonVisualLocks: []string{
			"graphicFrameLocks.noGrp",
		},
		HasTransform: true,
		ExtCX:        1000,
		ExtCY:        500,
		HasTable:     true,
		Table: tableModel{
			Columns:             []int64{100, 200},
			Rows:                []tableRow{{Height: 300, Cells: []tableCell{{Text: "A"}}}},
			StyleID:             "{style}",
			FirstRow:            true,
			UnsupportedFeatures: []string{"cell bevel"},
		},
	}, nil)
	if err != nil {
		t.Fatalf("table graphic frame should not error: %v", err)
	}
	if tableFrame.Table == nil || len(tableFrame.Table.Columns) != 2 || tableFrame.Table.StyleID != "{style}" || len(tableFrame.Table.UnsupportedFeatures) != 1 {
		t.Fatalf("graphic frame primitive did not preserve table: %+v", tableFrame.Table)
	}
	if len(tableFrame.NonVisualLocks) != 1 || tableFrame.NonVisualLocks[0] != "graphicFrameLocks.noGrp" {
		t.Fatalf("graphic frame primitive did not preserve lock metadata: %+v", tableFrame.NonVisualLocks)
	}
	if tableFrame.Provenance.Title != "Table title" || tableFrame.Provenance.CreationID != "{TABLE-CREATION}" {
		t.Fatalf("graphic frame primitive did not preserve cNvPr title provenance: %+v", tableFrame.Provenance)
	}

	diagramFrame, err := renderGraphicFramePrimitiveFromElement("ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), slideElement{
		Kind:          "graphicFrame",
		ID:            "10",
		Name:          "Diagram",
		HasTransform:  true,
		ExtCX:         1000,
		ExtCY:         500,
		DiagramDataID: "rIdDiagram",
	}, map[string]pptx.Relationship{})
	if err == nil {
		t.Fatalf("expected missing diagram relationship error")
	}
	if diagramFrame.Diagram != nil || len(diagramFrame.Unsupported) == 0 {
		t.Fatalf("diagram error should be preserved as unsupported conversion evidence: %+v", diagramFrame)
	}
}

func TestRenderSceneFromElementsLowersAllPrimitiveFamilies(t *testing.T) {
	pkg := &pptx.Package{ContentTypes: pptx.ContentTypes{Defaults: map[string]string{"png": "image/png"}}}
	elements := []slideElement{
		{Kind: "pic", ID: "1", Name: "Picture", EmbedID: "rIdImage", HasTransform: true, ExtCX: 100, ExtCY: 100},
		{Kind: "sp", ID: "2", Name: "Shape", HasTransform: true, ExtCX: 100, ExtCY: 100, Text: "Text"},
		{Kind: "cxnSp", ID: "3", Name: "Connector", HasTransform: true, ExtCX: 100, ExtCY: 0, HasLine: true},
		{Kind: "graphicFrame", ID: "4", Name: "Table", HasTransform: true, ExtCX: 100, ExtCY: 100, HasTable: true, Table: tableModel{Columns: []int64{100}, ColumnIDs: []string{"col-1"}, Rows: []tableRow{{ID: "row-1", Cells: []tableCell{{Text: "A"}}}}}},
		{Kind: "grpSp", ID: "5", Name: "Group", HasTransform: true, ExtCX: 100, ExtCY: 100},
		{Kind: "contentPart", ID: "6", Name: "Unsupported"},
	}
	scene, errs := renderSceneFromElements(pkg, "ppt/slides/slide1.xml", "ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), elements, map[string]pptx.Relationship{
		"rIdImage": {ID: "rIdImage", Type: pptx.ImageRelType, Target: "../media/image.png"},
	})
	if len(errs) != 0 {
		t.Fatalf("unexpected scene conversion errors: %v", errs)
	}
	counts := map[renderPrimitiveKind]int{}
	for _, primitive := range scene.Primitives {
		counts[primitive.Kind]++
		if primitive.Provenance.SourcePart == "" || len(primitive.Provenance.SchemaAnchors) == 0 {
			t.Fatalf("primitive missing provenance/schema anchors: %+v", primitive)
		}
	}
	for _, kind := range []renderPrimitiveKind{renderPrimitivePicture, renderPrimitiveShape, renderPrimitiveConnector, renderPrimitiveGraphicFrame, renderPrimitiveGroup, renderPrimitiveUnsupported} {
		if counts[kind] != 1 {
			t.Fatalf("expected one %s primitive, got counts=%+v", kind, counts)
		}
	}
	if scene.Primitives[1].Shape.Text == nil {
		t.Fatalf("shape text primitive was not lowered: %+v", scene.Primitives[1].Shape)
	}
	if scene.Primitives[3].GraphicFrame.Table == nil {
		t.Fatalf("table primitive was not lowered: %+v", scene.Primitives[3].GraphicFrame)
	}
	if !slices.Equal(scene.Primitives[3].GraphicFrame.Table.ColumnIDs, []string{"col-1"}) || scene.Primitives[3].GraphicFrame.Table.Rows[0].ID != "row-1" {
		t.Fatalf("table primitive did not preserve row/column ids: %+v", scene.Primitives[3].GraphicFrame.Table)
	}
}

func TestRenderPrimitiveBackendsAreSwappableByFamily(t *testing.T) {
	shape := recordingShapeBackend{}
	connector := recordingConnectorBackend{}
	frame := recordingGraphicFrameBackend{}
	shape.RenderShape(shapeBackendInput{Primitive: renderShapePrimitive{Provenance: renderPrimitiveProvenance{ID: "shape"}}})
	connector.RenderConnector(connectorBackendInput{Primitive: renderConnectorPrimitive{Provenance: renderPrimitiveProvenance{ID: "connector"}}})
	frame.RenderGraphicFrame(graphicFrameBackendInput{Primitive: renderGraphicFramePrimitive{Provenance: renderPrimitiveProvenance{ID: "frame"}}})
	if shape.calls != 1 || connector.calls != 1 || frame.calls != 1 {
		t.Fatalf("expected backend interfaces to accept primitive input, got shape=%d connector=%d frame=%d", shape.calls, connector.calls, frame.calls)
	}
}

type recordingShapeBackend struct {
	calls int
}

func (backend *recordingShapeBackend) RenderShape(input shapeBackendInput) []model.SkipItem {
	backend.calls++
	return nil
}

type recordingConnectorBackend struct {
	calls int
}

func (backend *recordingConnectorBackend) RenderConnector(input connectorBackendInput) []model.SkipItem {
	backend.calls++
	return nil
}

type recordingGraphicFrameBackend struct {
	calls int
}

func (backend *recordingGraphicFrameBackend) RenderGraphicFrame(input graphicFrameBackendInput) []model.SkipItem {
	backend.calls++
	return nil
}
