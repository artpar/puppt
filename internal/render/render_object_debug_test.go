package render

import (
	"archive/zip"
	"context"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/artpar/puppt/internal/model"
)

func TestRenderObjectDebugRecordsArtifactsAndIsolationModes(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "objects.pptx")
	if err := writeObjectDebugPPTX(deckPath); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(dir, "slide.png")
	debug := &ObjectDebugOptions{ArtifactDir: filepath.Join(dir, "artifacts")}
	result, err := Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected fully supported render, got status=%s unsupported=%+v", result.Status, result.Unsupported)
	}
	if len(debug.Records) != 2 {
		t.Fatalf("expected two painted object records, got %+v", debug.Records)
	}
	first := debug.Records[0]
	if first.ZOrder != 1 || first.SlidePart != "ppt/slides/slide1.xml" || first.SourcePart != "ppt/slides/slide1.xml" || first.CNvPrID != "2" || first.CNvPrName != "Red Rect" || first.Kind != "sp" {
		t.Fatalf("unexpected first object attribution: %+v", first)
	}
	if first.CNvPrDescription != "Primary rectangle" || first.CNvPrTitle != "Red shape" || first.CNvPrCreationID != "{RED-RECT-CREATION}" {
		t.Fatalf("unexpected first object cNvPr metadata: %+v", first)
	}
	if first.XMLPath != `/p:sld/p:cSld/p:spTree/p:sp[.//p:cNvPr/@id="2"]` {
		t.Fatalf("unexpected first object XML path: %+v", first)
	}
	if first.Bounds.CX != emuPerInch || first.PixelBounds.MaxX < 70 || first.OutputPixelBounds == nil || !first.Painted {
		t.Fatalf("expected first object bounds and painted output bounds, got %+v", first)
	}
	if first.FractionalBounds.MaxX <= first.FractionalBounds.MinX || first.FractionalBounds.MaxY <= first.FractionalBounds.MinY {
		t.Fatalf("expected fractional pixel bounds, got %+v", first.FractionalBounds)
	}
	if first.ResolvedStyle.Geometry != "rect" || first.ResolvedStyle.Fill == "" || first.ResolvedStyle.NoLine != true {
		t.Fatalf("expected resolved style summary for first object, got %+v", first.ResolvedStyle)
	}
	for _, path := range []string{
		first.BeforeArtifactPath,
		first.ObjectArtifactPath,
		first.ThroughArtifactPath,
		debug.Records[1].BeforeArtifactPath,
		debug.Records[1].ObjectArtifactPath,
		debug.Records[1].ThroughArtifactPath,
	} {
		if path == "" {
			t.Fatalf("expected object artifact path in records: %+v", debug.Records)
		}
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected object artifact %s: %v", path, err)
		}
	}

	objectOnlyPath := filepath.Join(dir, "object-only.png")
	_, err = Render(context.Background(), deckPath, Options{
		SlideNumber: 1,
		OutputPath:  objectOnlyPath,
		ObjectDebug: &ObjectDebugOptions{Mode: ObjectDebugRenderObjectOnly, TargetZOrder: 2},
	})
	if err != nil {
		t.Fatalf("object-only render failed: %v", err)
	}
	objectOnly := decodePNG(t, objectOnlyPath)
	if got := color.RGBAModel.Convert(objectOnly.At(10, 10)).(color.RGBA); got.A != 0 {
		t.Fatalf("object-only target 2 should leave object 1 transparent, got %#v", got)
	}
	if got := color.RGBAModel.Convert(objectOnly.At(90, 10)).(color.RGBA); got.A == 0 {
		t.Fatalf("object-only target 2 should paint object 2, got %#v", got)
	}

	flatBackgroundPath := filepath.Join(dir, "object-only-flat.png")
	flatBackground := color.RGBA{R: 17, G: 34, B: 51, A: 255}
	_, err = Render(context.Background(), deckPath, Options{
		SlideNumber: 1,
		OutputPath:  flatBackgroundPath,
		ObjectDebug: &ObjectDebugOptions{Mode: ObjectDebugRenderObjectOnly, TargetZOrder: 2, HasFlatBackground: true, FlatBackground: flatBackground},
	})
	if err != nil {
		t.Fatalf("object-only flat-background render failed: %v", err)
	}
	objectOnlyFlat := decodePNG(t, flatBackgroundPath)
	if got := color.RGBAModel.Convert(objectOnlyFlat.At(10, 10)).(color.RGBA); !objectDebugColorNear(got, flatBackground, 5) {
		t.Fatalf("object-only flat background should replace skipped object 1, got %#v", got)
	}
	if got := color.RGBAModel.Convert(objectOnlyFlat.At(90, 10)).(color.RGBA); objectDebugColorNear(got, flatBackground, 5) {
		t.Fatalf("object-only flat background should still paint object 2, got %#v", got)
	}

	beforePath := filepath.Join(dir, "before.png")
	_, err = Render(context.Background(), deckPath, Options{
		SlideNumber: 1,
		OutputPath:  beforePath,
		ObjectDebug: &ObjectDebugOptions{Mode: ObjectDebugRenderBefore, TargetZOrder: 2},
	})
	if err != nil {
		t.Fatalf("before render failed: %v", err)
	}
	before := decodePNG(t, beforePath)
	if got := color.RGBAModel.Convert(before.At(10, 10)).(color.RGBA); got == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("before target 2 should include object 1, got %#v", got)
	}
	if got := color.RGBAModel.Convert(before.At(90, 10)).(color.RGBA); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("before target 2 should omit object 2, got %#v", got)
	}

	throughPath := filepath.Join(dir, "through.png")
	_, err = Render(context.Background(), deckPath, Options{
		SlideNumber: 1,
		OutputPath:  throughPath,
		ObjectDebug: &ObjectDebugOptions{Mode: ObjectDebugRenderThrough, TargetZOrder: 1},
	})
	if err != nil {
		t.Fatalf("through render failed: %v", err)
	}
	through := decodePNG(t, throughPath)
	if got := color.RGBAModel.Convert(through.At(10, 10)).(color.RGBA); got == (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("through target 1 should include object 1, got %#v", got)
	}
	if got := color.RGBAModel.Convert(through.At(90, 10)).(color.RGBA); got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("through target 1 should omit object 2, got %#v", got)
	}
}

func TestRenderObjectDebugNormalModeDoesNotChangePixels(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "objects.pptx")
	if err := writeObjectDebugPPTX(deckPath); err != nil {
		t.Fatal(err)
	}

	normalPath := filepath.Join(dir, "normal.png")
	result, err := Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: normalPath})
	if err != nil {
		t.Fatalf("normal render failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected fully supported normal render, got status=%s unsupported=%+v", result.Status, result.Unsupported)
	}

	debugPath := filepath.Join(dir, "debug.png")
	debug := &ObjectDebugOptions{}
	result, err = Render(context.Background(), deckPath, Options{SlideNumber: 1, OutputPath: debugPath, ObjectDebug: debug})
	if err != nil {
		t.Fatalf("debug render failed: %v", err)
	}
	if result.Status != "ok" {
		t.Fatalf("expected fully supported debug render, got status=%s unsupported=%+v", result.Status, result.Unsupported)
	}
	if len(debug.Records) != 2 {
		t.Fatalf("expected normal debug mode to record two painted objects, got %+v", debug.Records)
	}

	diff, err := comparePNG(debugPath, normalPath)
	if err != nil {
		t.Fatalf("compare debug and normal render: %v", err)
	}
	if diff.DifferentPixels != 0 {
		t.Fatalf("expected normal debug mode to preserve rendered pixels, got diff %+v", diff)
	}
}

func objectDebugColorNear(got color.RGBA, want color.RGBA, tolerance int) bool {
	return absInt(int(got.R)-int(want.R)) <= tolerance &&
		absInt(int(got.G)-int(want.G)) <= tolerance &&
		absInt(int(got.B)-int(want.B)) <= tolerance &&
		absInt(int(got.A)-int(want.A)) <= tolerance
}

func TestObjectStyleSummaryIncludesResolvedParagraphTextStyle(t *testing.T) {
	summary := objectStyleSummary(slideElement{
		Text:                "Hello",
		FontFamily:          "Calibri Light",
		FontSize:            6000,
		IsTextBox:           true,
		Description:         "Diagram description",
		Title:               "Lifecycle chart",
		CreationID:          "{TEXT-CREATION}",
		HasHidden:           true,
		Hidden:              false,
		HasDecorative:       true,
		Decorative:          true,
		NonVisualProperties: []string{"decorative=true", "hidden=false"},
		NonVisualLocks:      []string{"spLocks.noTextEdit"},
		HasTable:            true,
		Table: tableModel{
			StyleID:  "{TABLE-STYLE}",
			FirstRow: true,
			BandRow:  true,
			ColumnIDs: []string{
				"col-a",
				"col-b",
			},
			Rows: []tableRow{{ID: "row-a"}, {ID: "row-b"}},
		},
		HasTextWrap:                true,
		TextWrap:                   "square",
		HasShapeAutofit:            true,
		HasNormAutofit:             true,
		HasFontScalePct:            true,
		FontScalePct:               85000,
		HasLineSpacingReductionPct: true,
		LineSpacingReductionPct:    20000,
		HasFirstLastSpacing:        true,
		IncludeFirstLastSpacing:    false,
		HasTextRightToLeftColumns:  true,
		TextRightToLeftColumns:     false,
		TextParagraphs: []textParagraph{{
			Text:              "Hello",
			FontFamily:        "Calibri",
			BulletFontFamily:  "Arial",
			FontSize:          4400,
			HasBold:           true,
			Bold:              true,
			HasTextColor:      true,
			TextColor:         color.RGBA{R: 0x00, G: 0x70, B: 0xC0, A: 0xFF},
			HasRTL:            true,
			HasEALineBreak:    true,
			EALineBreak:       true,
			HasLatinLineBreak: true,
			HasHangingPunct:   true,
			HangingPunct:      true,
			Runs:              []textRun{{Text: "Hello", FontFamily: "Arial", FontSize: 4400}},
		}},
	})
	if summary.FontFamily != "Arial" || !slices.Equal(summary.FontFamilies, []string{"Arial", "Calibri"}) {
		t.Fatalf("expected authored text font family in object summary, got %+v", summary)
	}
	if summary.FontSize != 6000 || summary.ParagraphFontSize != 4400 || !summary.Bold || summary.TextColor != "#0070C0/FF" || !summary.TextBox {
		t.Fatalf("expected resolved paragraph style in object summary, got %+v", summary)
	}
	if summary.Description != "Diagram description" || summary.Title != "Lifecycle chart" {
		t.Fatalf("expected cNvPr description/title metadata in object summary, got %+v", summary)
	}
	if summary.CreationID != "{TEXT-CREATION}" {
		t.Fatalf("expected cNvPr creationId metadata in object summary, got %+v", summary)
	}
	for _, want := range []string{"decorative=true", "hidden=false"} {
		if !slices.Contains(summary.NonVisualProperties, want) {
			t.Fatalf("expected cNvPr boolean property %q in object summary, got %+v", want, summary.NonVisualProperties)
		}
	}
	if len(summary.NonVisualLocks) != 1 || summary.NonVisualLocks[0] != "spLocks.noTextEdit" {
		t.Fatalf("expected non-visual lock metadata in object summary, got %+v", summary.NonVisualLocks)
	}
	if !summary.Table || !slices.Equal(summary.TableColumnIDs, []string{"col-a", "col-b"}) || !slices.Equal(summary.TableRowIDs, []string{"row-a", "row-b"}) {
		t.Fatalf("expected table row/column ids in object summary, got %+v", summary)
	}
	if summary.TableStyleID != "{TABLE-STYLE}" {
		t.Fatalf("expected table style id in object summary, got %+v", summary)
	}
	for _, want := range []string{"firstRow=true", "bandRow=true"} {
		if !slices.Contains(summary.TableProperties, want) {
			t.Fatalf("expected table property %q in object summary, got %+v", want, summary.TableProperties)
		}
	}
	for _, want := range []string{"wrap=square", "spAutoFit=true", "normAutofit=true", "fontScale=85000", "lnSpcReduction=20000", "spcFirstLastPara=false", "rtlCol=false"} {
		if !slices.Contains(summary.TextBodyProperties, want) {
			t.Fatalf("expected text body property %q in object summary, got %+v", want, summary.TextBodyProperties)
		}
	}
	for _, want := range []string{"rtl=false", "eaLnBrk=true", "latinLnBrk=false", "hangingPunct=true"} {
		if !slices.Contains(summary.TextParagraphProperties, want) {
			t.Fatalf("expected paragraph property %q in object summary, got %+v", want, summary.TextParagraphProperties)
		}
	}
}

func TestObjectStyleSummaryIncludesShadowParameters(t *testing.T) {
	summary := objectStyleSummary(slideElement{
		HasShadow:       true,
		ShadowColor:     color.RGBA{A: 102},
		ShadowBlur:      127000,
		ShadowDistance:  63500,
		ShadowDirection: 1800000,
		ShadowAlignment: "tl",
		HasShadowScaleX: true,
		ShadowScaleX:    120000,
		HasShadowScaleY: true,
		ShadowScaleY:    80000,
		HasShadowSkewX:  true,
		ShadowSkewX:     60000,
		HasShadowSkewY:  true,
		ShadowSkewY:     -60000,
	})
	if !summary.Shadow || summary.ShadowColor != "#000000/66" || summary.ShadowBlur != 127000 || summary.ShadowDistance != 63500 || summary.ShadowDirection != 1800000 || summary.ShadowAlignment != "tl" {
		t.Fatalf("expected direct shadow metrics in object summary, got %+v", summary)
	}
	if summary.ShadowScaleX != 120000 || summary.ShadowScaleY != 80000 || summary.ShadowSkewX != 60000 || summary.ShadowSkewY != -60000 {
		t.Fatalf("expected shadow transform metrics in object summary, got %+v", summary)
	}
}

func TestObjectStyleSummaryIncludesCustomPathDetails(t *testing.T) {
	summary := objectStyleSummary(slideElement{
		CustomPath: []pathPoint{
			{X: 0, Y: 0},
			{X: 0.75, Y: 0.1},
			{X: 1, Y: 1},
			{X: 0.25, Y: 0.8},
		},
		CustomPathCommands: []pathCommand{
			{Kind: "moveTo"},
			{Kind: "lnTo"},
			{Kind: "lnTo"},
			{Kind: "close"},
		},
		CustomPathUnsupported: []string{"custom geometry uses unsupported arcTo command"},
	})
	if summary.Geometry != "customPath" || summary.CustomPathPoints != 4 || summary.CustomPathCommands != 4 {
		t.Fatalf("expected custom path summary, got %+v", summary)
	}
	if len(summary.CustomPathCoordinates) != 4 || summary.CustomPathCoordinates[1].X != 0.75 || summary.CustomPathCoordinates[1].Y != 0.1 {
		t.Fatalf("expected custom path coordinates, got %+v", summary.CustomPathCoordinates)
	}
	if summary.CustomPathBounds == nil || summary.CustomPathBounds.MinX != 0 || summary.CustomPathBounds.MinY != 0 || summary.CustomPathBounds.MaxX != 1 || summary.CustomPathBounds.MaxY != 1 {
		t.Fatalf("expected normalized custom path bounds, got %+v", summary.CustomPathBounds)
	}
	if len(summary.CustomPathUnsupported) != 1 || summary.CustomPathUnsupported[0] != "custom geometry uses unsupported arcTo command" {
		t.Fatalf("expected custom path unsupported diagnostics, got %+v", summary.CustomPathUnsupported)
	}
}

func TestObjectStyleSummaryIncludesImageAndTableProperties(t *testing.T) {
	summary := objectStyleSummary(slideElement{
		EmbedID:              "rId5",
		SVGEmbedID:           "rId6",
		ImageMediaPart:       "ppt/media/image8.png",
		ImageContentType:     "image/png",
		ImageWidth:           2830,
		ImageHeight:          820,
		DiagramDataID:        "rId7",
		BWMode:               "gray",
		HasCrop:              true,
		CropLeft:             1000,
		CropTop:              2000,
		CropRight:            3000,
		CropBottom:           4000,
		HasImageAlphaModFix:  true,
		ImageAlphaModFixPct:  65000,
		BlipCompressionState: "print",
		HasBlipRotWithShape:  true,
		BlipRotWithShape:     false,
		HasSoftEdge:          true,
		SoftEdgeRadius:       12700,
		HasTable:             true,
		Table: tableModel{
			UnsupportedFeatures: []string{"table cell uses diagonal border"},
		},
	})
	if summary.Image != "embed=rId5 svg=rId6 part=ppt/media/image8.png type=image/png size=2830x820 diagram=rId7 bwMode=gray" {
		t.Fatalf("expected explicit image relationship summary, got %+v", summary)
	}
	if summary.ImageCrop != "l=1000 t=2000 r=3000 b=4000" {
		t.Fatalf("expected image crop summary, got %+v", summary)
	}
	wantEffects := []string{"alphaModFix=65000", "cstate=print", "rotWithShape=false", "softEdge=12700"}
	if len(summary.ImageEffects) != len(wantEffects) {
		t.Fatalf("expected image effects %v, got %+v", wantEffects, summary.ImageEffects)
	}
	for index, want := range wantEffects {
		if summary.ImageEffects[index] != want {
			t.Fatalf("expected image effect %q at %d, got %+v", want, index, summary.ImageEffects)
		}
	}
	if !summary.Table || len(summary.TableUnsupported) != 1 || summary.TableUnsupported[0] != "table cell uses diagonal border" {
		t.Fatalf("expected table unsupported summary, got %+v", summary)
	}
}

func TestPaintedObjectRecordIncludesUnsupportedItems(t *testing.T) {
	before := solidObjectDebugTestImage(2, 2, color.RGBA{A: 255})
	after := solidObjectDebugTestImage(2, 2, color.RGBA{R: 255, A: 255})
	record := paintedObjectRecord(
		"ppt/slides/slide1.xml",
		"ppt/slides/slide1.xml",
		slideElement{Kind: "sp", ID: "2", Name: "Unsupported Shape", HasTransform: true, ExtCX: emuPerInch, ExtCY: emuPerInch},
		1,
		slideSize{CX: emuPerInch, CY: emuPerInch},
		after.Bounds(),
		before,
		after,
		true,
		[]model.SkipItem{{Code: partialUnsupportedCode, Part: "ppt/slides/slide1.xml", Message: "shape object uses unsupported bevel"}},
	)
	if len(record.Unsupported) != 1 {
		t.Fatalf("expected one object unsupported item, got %+v", record.Unsupported)
	}
	if record.Unsupported[0].Code != partialUnsupportedCode || record.Unsupported[0].Part != "ppt/slides/slide1.xml" || record.Unsupported[0].Message != "shape object uses unsupported bevel" {
		t.Fatalf("unexpected object unsupported summary: %+v", record.Unsupported)
	}
}

func solidObjectDebugTestImage(width int, height int, fill color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)
	return img
}

func writeObjectDebugPPTX(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	parts := map[string][]byte{
		"[Content_Types].xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`),
		"_rels/.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`),
		"ppt/presentation.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst>
  <p:sldSz cx="1828800" cy="914400"/>
</p:presentation>`),
		"ppt/_rels/presentation.xml.rels": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`),
		"ppt/slides/slide1.xml": []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:adec="http://schemas.microsoft.com/office/drawing/2017/decorative" xmlns:a16="http://schemas.microsoft.com/office/drawing/2014/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="2" name="Red Rect" descr="Primary rectangle" title="Red shape" hidden="0"><a:extLst><a:ext uri="{FF2B5EF4-FFF2-40B4-BE49-F238E27FC236}"><a16:creationId id="{RED-RECT-CREATION}"/></a:ext><a:ext uri="{C183D7F6-B498-43B3-948B-1728B52AA6E4}"><adec:decorative val="0"/></a:ext></a:extLst></p:cNvPr><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill><a:ln><a:noFill/></a:ln></p:spPr>
      </p:sp>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="3" name="Blue Rect"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
        <p:spPr><a:xfrm><a:off x="914400" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom><a:solidFill><a:srgbClr val="0000FF"/></a:solidFill><a:ln><a:noFill/></a:ln></p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`),
	}
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "ppt/presentation.xml", "ppt/_rels/presentation.xml.rels", "ppt/slides/slide1.xml"} {
		writer, err := archive.Create(name)
		if err != nil {
			archive.Close()
			return err
		}
		if _, err := writer.Write(parts[name]); err != nil {
			archive.Close()
			return err
		}
	}
	return archive.Close()
}
