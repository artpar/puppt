package render

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	"github.com/go-text/typesetting/di"
	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/fixed"
)

func TestFilterInheritedPlaceholdersDropsDisabledSlideNumberPlaceholder(t *testing.T) {
	settings := defaultHeaderFooterSettings()
	settings.SlideNumber = false
	got := filterInheritedPlaceholdersForRender([]slideElement{{
		Kind:            "sp",
		Text:            "‹#›",
		IsPlaceholder:   true,
		PlaceholderType: "sldNum",
	}}, nil, settings, true)
	if len(got) != 0 {
		t.Fatalf("disabled slide-number placeholder should not render, got %+v", got)
	}
}

func TestInheritedHeaderFooterSettingsHonorsExplicitFalse(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"><p:hf sldNum="0" dt="0"/></p:sldMaster>`),
	}}
	got := inheritedHeaderFooterSettings(pkg, []string{"ppt/slideMasters/slideMaster1.xml"})
	if got.SlideNumber || got.DateTime || !got.Footer || !got.Header {
		t.Fatalf("unexpected inherited header/footer settings: %+v", got)
	}
}

func TestInheritedHeaderFooterSettingsTreatsMissingElementAsDisabled(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"/>`),
	}}
	got := inheritedHeaderFooterSettings(pkg, []string{"ppt/slideMasters/slideMaster1.xml"})
	if got.SlideNumber || got.DateTime || got.Footer || got.Header {
		t.Fatalf("missing hf element should not enable inherited placeholders, got %+v", got)
	}
}

func TestPresentationShowsSpecialPlaceholdersOnTitleSlideDefaultsOn(t *testing.T) {
	pkg := &pptx.Package{
		PresentationPath: "ppt/presentation.xml",
		Parts: map[string][]byte{
			"ppt/presentation.xml": []byte(`<p:presentation xmlns:p="p"/>`),
		},
	}
	if !presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) {
		t.Fatal("missing showSpecialPlsOnTitleSld should default to showing special placeholders")
	}
}

func TestPresentationShowsSpecialPlaceholdersOnTitleSlideReadsFalse(t *testing.T) {
	pkg := &pptx.Package{
		PresentationPath: "ppt/presentation.xml",
		Parts: map[string][]byte{
			"ppt/presentation.xml": []byte(`<p:presentation xmlns:p="p" showSpecialPlsOnTitleSld="0"/>`),
		},
	}
	if presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) {
		t.Fatal("showSpecialPlsOnTitleSld=0 should hide special placeholders on title slides")
	}
}

func TestSlideUsesTitleLayoutReadsLayoutType(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`),
		"ppt/slides/_rels/slide2.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout2.xml"/>
</Relationships>`),
		"ppt/slideLayouts/slideLayout1.xml": []byte(`<p:sldLayout xmlns:p="p" type="title"/>`),
		"ppt/slideLayouts/slideLayout2.xml": []byte(`<p:sldLayout xmlns:p="p" type="obj"/>`),
	}}
	if !slideUsesTitleLayout(pkg, "ppt/slides/slide1.xml") {
		t.Fatal("expected title layout to be detected")
	}
	if slideUsesTitleLayout(pkg, "ppt/slides/slide2.xml") {
		t.Fatal("non-title layout should not be treated as a title slide")
	}
}

func TestFormatUnsupportedSummarySortsDeterministically(t *testing.T) {
	got := formatUnsupportedSummary(map[string]int{
		"later alphabetically":   2,
		"earlier alphabetically": 2,
		"most common":            3,
	}, 2)
	want := "3x most common; 2x earlier alphabetically"
	if got != want {
		t.Fatalf("unexpected unsupported summary: got=%q want=%q", got, want)
	}
	if got := formatUnsupportedSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty unsupported summary, got %q", got)
	}
}

func TestFormatSlideDiffSummarySortsDeterministically(t *testing.T) {
	got := formatSlideDiffSummary([]realWorldSlideDiff{
		{Label: "deck slide 002", DifferentPixels: 20, Unsupported: 3, Status: "partial"},
		{Label: "deck slide 001", DifferentPixels: 20, Unsupported: 2, Status: "partial"},
		{Label: "deck slide 003", DifferentPixels: 30, Unsupported: 1, Status: "partial"},
	}, 2)
	want := "deck slide 003=30px/1 unsupported/partial; deck slide 001=20px/2 unsupported/partial"
	if got != want {
		t.Fatalf("unexpected slide diff summary: got=%q want=%q", got, want)
	}
	if got := formatSlideDiffSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty slide diff summary, got %q", got)
	}
}

func TestResolveTestOutputPathAcceptsRepoRelativeParents(t *testing.T) {
	got := resolveTestOutputPath(filepath.Join("docs", "renderer-diagnostic-output.json"))
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("expected resolved output parent to exist, got path %q: %v", got, err)
	}
}

func TestExtractRawObjectXMLFindsShapeByCNvPr(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld>
    <p:spTree>
      <p:sp><p:nvSpPr><p:cNvPr id="2" name="Title 1"/></p:nvSpPr><p:txBody><a:p><a:r><a:t>wrong</a:t></a:r></a:p></p:txBody></p:sp>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="7" name="Freeform 6"/></p:nvSpPr>
        <p:spPr><a:custGeom><a:pathLst/></a:custGeom></p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)
	raw, err := extractRawObjectXML(data, "sp", "7", "Freeform 6")
	if err != nil {
		t.Fatalf("extract raw object: %v", err)
	}
	if !strings.HasPrefix(raw, "<p:sp>") && !strings.HasPrefix(raw, "<p:sp\n") {
		t.Fatalf("expected raw shape XML to start at p:sp, got %q", raw[:min(len(raw), 32)])
	}
	if strings.Contains(raw, `name="Title 1"`) || !strings.Contains(raw, `name="Freeform 6"`) || !strings.Contains(raw, "<a:custGeom>") {
		t.Fatalf("unexpected extracted object XML: %s", raw)
	}
}

func TestExtractRawObjectXMLForRecordUsesZOrderForDuplicateCNvPr(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <p:sp><p:nvSpPr><p:cNvPr id="7" name="Freeform 6"/></p:nvSpPr><p:spPr><a:xfrm><a:off x="0" y="0"/></a:xfrm></p:spPr></p:sp>
      <p:pic><p:nvPicPr><p:cNvPr id="9" name="Picture 8"/></p:nvPicPr></p:pic>
      <p:sp><p:nvSpPr><p:cNvPr id="7" name="Freeform 6"/></p:nvSpPr><p:spPr><a:xfrm><a:off x="365" y="19"/></a:xfrm></p:spPr></p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)
	raw, err := extractRawObjectXMLForRecord(data, objectFailureRecord{
		Kind:      "sp",
		CNvPrID:   "7",
		CNvPrName: "Freeform 6",
		ZOrder:    3,
	})
	if err != nil {
		t.Fatalf("extract duplicate raw object: %v", err)
	}
	if !strings.Contains(raw, `x="365"`) || strings.Contains(raw, `x="0"`) {
		t.Fatalf("expected z-order extraction to select the second duplicate shape, got %s", raw)
	}
}

func TestExtractRawSlideBackgroundXMLFindsDirectCSldBackground(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld>
    <p:bg><p:bgPr><a:solidFill><a:schemeClr val="bg1"/></a:solidFill></p:bgPr></p:bg>
    <p:spTree><p:sp><p:nvSpPr><p:cNvPr id="2" name="Shape"/></p:nvSpPr></p:sp></p:spTree>
  </p:cSld>
</p:sld>`)
	raw := extractRawSlideBackgroundXML(data)
	if !strings.HasPrefix(raw, "<p:bg>") || !strings.Contains(raw, `val="bg1"`) {
		t.Fatalf("expected raw slide background XML, got %q", raw)
	}
}

func TestShapeObjectFixtureBackgroundXMLPrefersActualSlideBackground(t *testing.T) {
	slideData := []byte(`<p:sld xmlns:p="p" xmlns:a="a"><p:cSld><p:bg><p:bgPr><a:solidFill><a:schemeClr val="bg1"/></a:solidFill></p:bgPr></p:bg></p:cSld></p:sld>`)
	sourceData := []byte(`<p:sldMaster xmlns:p="p" xmlns:a="a"><p:cSld><p:bg><p:bgPr><a:gradFill><a:gsLst/></a:gradFill></p:bgPr></p:bg></p:cSld></p:sldMaster>`)
	got := shapeObjectFixtureBackgroundXML(slideData, sourceData)
	if !strings.Contains(got, `val="bg1"`) || strings.Contains(got, "gradFill") {
		t.Fatalf("expected actual slide background to win, got %s", got)
	}
}

func TestShapeObjectFixtureCopiesTableStylesForTableGraphicFrames(t *testing.T) {
	parts := map[string][]byte{}
	source := []byte(`<a:tblStyleLst xmlns:a="a"/>`)
	pkg := &pptx.Package{Parts: map[string][]byte{"ppt/tableStyles.xml": source}}
	addShapeObjectFixturePackageDependencies(parts, pkg, []objectFailureRecord{{
		Kind:          "graphicFrame",
		ResolvedStyle: ObjectStyleSummary{Table: true},
	}})
	if !bytes.Equal(parts["ppt/tableStyles.xml"], source) {
		t.Fatalf("expected table graphic-frame fixture to copy source table styles, got %q", parts["ppt/tableStyles.xml"])
	}
	if !strings.Contains(shapeObjectFixtureContentTypes(true), "presentationml.tableStyles+xml") {
		t.Fatalf("expected fixture content types to include table style override")
	}

	nonTableParts := map[string][]byte{}
	addShapeObjectFixturePackageDependencies(nonTableParts, pkg, []objectFailureRecord{{Kind: "sp"}})
	if _, ok := nonTableParts["ppt/tableStyles.xml"]; ok {
		t.Fatalf("did not expect non-table fixture to copy table styles")
	}
}

func TestPictureObjectFixtureSlidePreservesRawPictureXML(t *testing.T) {
	rawPicture := `<p:pic><p:nvPicPr><p:cNvPr id="1028" name="Picture 4" descr="Scale Up Icons"/></p:nvPicPr><p:blipFill><a:blip r:embed="rId5"><a:extLst><a:ext uri="{28A0092B-C50C-407E-A947-70E740481C1C}"><a14:useLocalDpi xmlns:a14="http://schemas.microsoft.com/office/drawing/2010/main" val="0"/></a:ext></a:extLst></a:blip><a:srcRect/><a:stretch><a:fillRect/></a:stretch></p:blipFill><p:spPr bwMode="auto"><a:noFill/></p:spPr></p:pic>`
	got := pictureObjectFixtureSlide(rawPicture)
	for _, want := range []string{`descr="Scale Up Icons"`, `r:embed="rId5"`, `useLocalDpi`, `<a:srcRect/>`, `bwMode="auto"`, `<a:noFill/>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected raw picture fixture slide to preserve %s, got %s", want, got)
		}
	}
	if strings.Contains(got, `cstate="print"`) || strings.Contains(got, `r:embed="rId1"`) {
		t.Fatalf("raw picture fixture slide should not synthesize a replacement blip: %s", got)
	}
}

func TestPictureObjectFixtureSlideRelationshipsUsesSourceRelationshipID(t *testing.T) {
	got := pictureObjectFixtureSlideRelationships("ppt/media/object.png", "rId5")
	if !strings.Contains(got, `Id="rId5"`) || !strings.Contains(got, `Target="../media/object.png"`) {
		t.Fatalf("expected picture fixture relationship to preserve source id, got %s", got)
	}
}

func TestStripNonPlaceholderObjectsInPartKeepsDependencyPlaceholders(t *testing.T) {
	data := []byte(`<p:sldLayout xmlns:p="p" xmlns:a="a">
  <p:cSld>
    <p:bg><p:bgPr><a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></p:bgPr></p:bg>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/></p:nvGrpSpPr>
      <p:grpSpPr/>
      <p:sp><p:nvSpPr><p:cNvPr id="2" name="Title 1"/><p:nvPr><p:ph type="ctrTitle"/></p:nvPr></p:nvSpPr><p:txBody><a:p><a:r><a:t>source</a:t></a:r></a:p></p:txBody></p:sp>
      <p:pic><p:nvPicPr><p:cNvPr id="7" name="Logo"/></p:nvPicPr></p:pic>
      <p:sp><p:nvSpPr><p:cNvPr id="8" name="Decor"/></p:nvSpPr></p:sp>
    </p:spTree>
  </p:cSld>
</p:sldLayout>`)
	got := string(stripNonPlaceholderObjectsInPart(data))
	if !strings.Contains(got, `name="Title 1"`) || !strings.Contains(got, `<p:ph type="ctrTitle"`) {
		t.Fatalf("expected placeholder source to remain: %s", got)
	}
	if strings.Contains(got, `name="Logo"`) || strings.Contains(got, `name="Decor"`) {
		t.Fatalf("expected non-placeholder render objects to be stripped: %s", got)
	}
	if !strings.Contains(got, "<p:bg>") || !strings.Contains(got, "<p:grpSpPr/>") {
		t.Fatalf("expected dependency scaffolding to remain: %s", got)
	}
}

func TestMicroFixtureOcclusionsUsesLaterZOrderIntersections(t *testing.T) {
	targetBounds := ObjectPixelBounds{MinX: 10, MinY: 20, MaxX: 30, MaxY: 40}
	occlusions := microFixtureOcclusions(objectFailureRecord{
		CNvPrID:           "2",
		CNvPrName:         "Title",
		ZOrder:            4,
		OutputPixelBounds: &targetBounds,
	}, []objectFailureRecord{{
		CNvPrID:           "1",
		CNvPrName:         "Earlier",
		ZOrder:            3,
		OutputPixelBounds: &ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 20, MaxY: 30},
	}, {
		CNvPrID:           "9",
		CNvPrName:         "Later",
		Kind:              "pic",
		ZOrder:            6,
		OutputPixelBounds: &ObjectPixelBounds{MinX: 25, MinY: 35, MaxX: 50, MaxY: 60},
	}, {
		CNvPrID:           "10",
		CNvPrName:         "Outside",
		ZOrder:            7,
		OutputPixelBounds: &ObjectPixelBounds{MinX: 50, MinY: 50, MaxX: 60, MaxY: 60},
	}})
	if len(occlusions) != 1 {
		t.Fatalf("expected one later overlapping occlusion, got %+v", occlusions)
	}
	if occlusions[0].CNvPrID != "9" || occlusions[0].Bounds.MinX != 25 || occlusions[0].Bounds.MaxX != 30 || occlusions[0].Bounds.MinY != 35 || occlusions[0].Bounds.MaxY != 40 {
		t.Fatalf("unexpected occlusion: %+v", occlusions[0])
	}
	if occlusions[0].MaskPaddingPixels != 1 {
		t.Fatalf("expected antialias padding on occlusion mask, got %+v", occlusions[0])
	}
	if !pointOccluded(24, 34, occlusions) {
		t.Fatalf("expected occlusion mask padding to cover one pixel outside the raw intersection")
	}
}

func TestMicroFixtureOcclusionsUseSourceBoundsForLaterTextBoxes(t *testing.T) {
	targetBounds := ObjectPixelBounds{MinX: 10, MinY: 20, MaxX: 80, MaxY: 60}
	occlusions := microFixtureOcclusions(objectFailureRecord{
		CNvPrID:           "2",
		CNvPrName:         "Picture",
		ZOrder:            4,
		OutputPixelBounds: &targetBounds,
	}, []objectFailureRecord{{
		CNvPrID:           "9",
		CNvPrName:         "Flow Label",
		Kind:              "sp",
		ZOrder:            6,
		PixelBounds:       ObjectPixelBounds{MinX: 20, MinY: 25, MaxX: 70, MaxY: 45},
		OutputPixelBounds: &ObjectPixelBounds{MinX: 35, MinY: 32, MaxX: 45, MaxY: 38},
	}})
	if len(occlusions) != 1 {
		t.Fatalf("expected one later overlapping text-box occlusion, got %+v", occlusions)
	}
	if got, want := occlusions[0].Bounds, (ObjectPixelBounds{MinX: 20, MinY: 25, MaxX: 70, MaxY: 45}); got != want {
		t.Fatalf("occlusion should use source-authored bounds, got %+v want %+v", got, want)
	}
}

func TestMicroFixtureUnderpaintsUsesEarlierZOrderIntersections(t *testing.T) {
	targetBounds := ObjectPixelBounds{MinX: 10, MinY: 20, MaxX: 30, MaxY: 40}
	underpaints := microFixtureUnderpaints(objectFailureRecord{
		CNvPrID:           "2",
		CNvPrName:         "Title",
		ZOrder:            4,
		OutputPixelBounds: &targetBounds,
	}, []objectFailureRecord{{
		CNvPrID:            "1",
		CNvPrName:          "Earlier",
		Kind:               "sp",
		ZOrder:             3,
		OutputPixelBounds:  &ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 20, MaxY: 30},
		ObjectArtifactPath: "earlier-object.png",
	}, {
		CNvPrID:           "9",
		CNvPrName:         "Later",
		ZOrder:            6,
		OutputPixelBounds: &ObjectPixelBounds{MinX: 25, MinY: 35, MaxX: 50, MaxY: 60},
	}, {
		CNvPrID:           "10",
		CNvPrName:         "Outside",
		ZOrder:            2,
		OutputPixelBounds: &ObjectPixelBounds{MinX: 50, MinY: 50, MaxX: 60, MaxY: 60},
	}})
	if len(underpaints) != 1 {
		t.Fatalf("expected one earlier overlapping underpaint, got %+v", underpaints)
	}
	if underpaints[0].CNvPrID != "1" || underpaints[0].Bounds.MinX != 10 || underpaints[0].Bounds.MaxX != 20 || underpaints[0].Bounds.MinY != 20 || underpaints[0].Bounds.MaxY != 30 {
		t.Fatalf("unexpected underpaint: %+v", underpaints[0])
	}
	if underpaints[0].ObjectArtifactPath != "earlier-object.png" {
		t.Fatalf("expected underpaint object artifact path, got %+v", underpaints[0])
	}
}

func TestMicroFixtureTargetScopeDiagnosticSplitsDiffByObjectMask(t *testing.T) {
	dir := t.TempDir()
	gotPath := filepath.Join(dir, "got-crop.png")
	referencePath := filepath.Join(dir, "reference-crop.png")
	maskPath := filepath.Join(dir, "object.png")

	got := image.NewRGBA(image.Rect(0, 0, 3, 1))
	reference := image.NewRGBA(image.Rect(0, 0, 3, 1))
	mask := image.NewRGBA(image.Rect(0, 0, 5, 1))
	for x := 0; x < 3; x++ {
		got.SetRGBA(x, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		reference.SetRGBA(x, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	reference.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})
	mask.SetRGBA(1, 0, color.RGBA{A: 128})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(maskPath, mask); err != nil {
		t.Fatal(err)
	}

	crop := ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 3, MaxY: 0}
	scope, err := microFixtureTargetScopeDiagnostic(gotPath, referencePath, objectFailureRecord{
		OutputPixelBounds:  &crop,
		ObjectArtifactPath: maskPath,
	}, nil)
	if err != nil {
		t.Fatalf("target scope diagnostic failed: %v", err)
	}
	if scope.CropPixels != 3 || scope.ComparedPixels != 3 || scope.ObjectMaskPixels != 1 {
		t.Fatalf("unexpected target scope dimensions/mask: %+v", scope)
	}
	if scope.ObjectMaskPartialAlphaPixels != 1 || scope.ObjectMaskLowAlphaPixels != 0 || scope.ObjectMaskMidAlphaPixels != 1 || scope.ObjectMaskHighAlphaPixels != 0 || scope.ObjectMaskFullAlphaPixels != 0 {
		t.Fatalf("expected one partial-alpha object mask pixel, got %+v", scope)
	}
	if scope.ObjectMaskPartialAlphaDarkPixels != 1 || scope.ObjectMaskPartialAlphaLightPixels != 0 || scope.ObjectMaskPartialAlphaOtherPixels != 0 {
		t.Fatalf("expected black partial-alpha object mask pixel, got %+v", scope)
	}
	if scope.DifferentPixels != 2 || scope.DifferentPixelsInsideObjectMask != 1 || scope.DifferentPixelsOutsideObjectMask != 1 {
		t.Fatalf("expected one differing pixel inside and one outside object mask, got %+v", scope)
	}
	if scope.DifferentPixelsInsidePartialAlphaObjectMask != 1 || scope.DifferentPixelsInsideMidAlphaObjectMask != 1 || scope.DifferentPixelsInsideFullAlphaObjectMask != 0 {
		t.Fatalf("expected differing pixel inside partial-alpha object mask, got %+v", scope)
	}
	if scope.DifferentPixelsInsideDarkPartialAlphaObjectMask != 1 || scope.DifferentPixelsInsideLightPartialAlphaObjectMask != 0 || scope.DifferentPixelsInsideOtherPartialAlphaObjectMask != 0 {
		t.Fatalf("expected differing pixel inside dark partial-alpha object mask, got %+v", scope)
	}
}

func TestMicroFixtureTargetScopeDiagnosticCountsPartialAlphaOverUnderpaint(t *testing.T) {
	dir := t.TempDir()
	gotPath := filepath.Join(dir, "got-crop.png")
	referencePath := filepath.Join(dir, "reference-crop.png")
	maskPath := filepath.Join(dir, "object.png")
	underpaintPath := filepath.Join(dir, "underpaint.png")

	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	mask := image.NewRGBA(image.Rect(0, 0, 3, 1))
	underpaint := image.NewRGBA(image.Rect(0, 0, 3, 1))
	for x := 0; x < 2; x++ {
		mask.SetRGBA(x+1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 128})
	}
	got.SetRGBA(0, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	reference.SetRGBA(0, 0, color.RGBA{R: 240, G: 240, B: 240, A: 255})
	got.SetRGBA(1, 0, color.RGBA{R: 240, G: 240, B: 240, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	underpaint.SetRGBA(1, 0, color.RGBA{A: 255})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(maskPath, mask); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(underpaintPath, underpaint); err != nil {
		t.Fatal(err)
	}

	crop := ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 2, MaxY: 0}
	scope, err := microFixtureTargetScopeDiagnostic(gotPath, referencePath, objectFailureRecord{
		OutputPixelBounds:  &crop,
		ObjectArtifactPath: maskPath,
	}, []microFixtureUnderpaint{{
		CNvPrID:            "7",
		CNvPrName:          "Earlier",
		ZOrder:             1,
		Bounds:             ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 1, MaxY: 0},
		ObjectArtifactPath: underpaintPath,
	}})
	if err != nil {
		t.Fatalf("target scope diagnostic failed: %v", err)
	}
	if scope.ObjectMaskPartialAlphaPixels != 2 || scope.ObjectMaskMidAlphaPixels != 2 || scope.ObjectMaskPartialAlphaPixelsOverUnderpaint != 1 {
		t.Fatalf("unexpected partial-alpha underpaint scope: %+v", scope)
	}
	if scope.ObjectMaskPartialAlphaLightPixels != 2 || scope.ObjectMaskPartialAlphaDarkPixels != 0 || scope.ObjectMaskPartialAlphaOtherPixels != 0 {
		t.Fatalf("expected light partial-alpha object mask pixels, got %+v", scope)
	}
	if scope.DifferentPixelsInsidePartialAlphaObjectMask != 2 || scope.DifferentPixelsInsideMidAlphaObjectMask != 2 || scope.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint != 1 || scope.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint != 1 {
		t.Fatalf("unexpected partial-alpha diff underpaint scope: %+v", scope)
	}
	if scope.DifferentPixelsInsideLightPartialAlphaObjectMask != 2 || scope.DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceDarker != 1 || scope.DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceLighter != 1 {
		t.Fatalf("expected light partial-alpha direction buckets, got %+v", scope)
	}
	if scope.DifferentPixelsReferenceDarker != 1 || scope.DifferentPixelsReferenceLighter != 1 {
		t.Fatalf("expected one darker and one lighter reference pixel, got %+v", scope)
	}
	if scope.ReferenceRGBDeltaSum8 != 0 || scope.ReferenceRGBAbsoluteDeltaSum8 != 90 {
		t.Fatalf("expected balanced RGB delta sum with 90 absolute delta, got %+v", scope)
	}
	if scope.DifferentBounds == nil || scope.DifferentBounds.MinX != 0 || scope.DifferentBounds.MaxX != 1 || scope.DifferentBounds.MinY != 0 || scope.DifferentBounds.MaxY != 0 {
		t.Fatalf("expected total diff bounds across both pixels, got %+v", scope)
	}
	if scope.ReferenceDarkerBounds == nil || scope.ReferenceDarkerBounds.MinX != 0 || scope.ReferenceDarkerBounds.MaxX != 0 {
		t.Fatalf("expected darker bounds on the first pixel, got %+v", scope)
	}
	if scope.ReferenceLighterBounds == nil || scope.ReferenceLighterBounds.MinX != 1 || scope.ReferenceLighterBounds.MaxX != 1 {
		t.Fatalf("expected lighter bounds on the second pixel, got %+v", scope)
	}
	if scope.DifferentPixelsInsidePartialAlphaObjectMaskReferenceDarker != 1 || scope.DifferentPixelsInsidePartialAlphaObjectMaskReferenceLighter != 1 {
		t.Fatalf("expected partial-alpha direction buckets, got %+v", scope)
	}
	if scope.DifferentPixelsInsideMidAlphaObjectMaskReferenceDarker != 1 || scope.DifferentPixelsInsideMidAlphaObjectMaskReferenceLighter != 1 {
		t.Fatalf("expected mid-alpha direction buckets, got %+v", scope)
	}
	if len(scope.TopDifferentRows) != 1 || scope.TopDifferentRows[0].Index != 0 || scope.TopDifferentRows[0].Count != 2 {
		t.Fatalf("expected row hotspot to include both diffs, got %+v", scope.TopDifferentRows)
	}
	if len(scope.TopDifferentColumns) != 2 || scope.TopDifferentColumns[0].Index != 0 || scope.TopDifferentColumns[1].Index != 1 {
		t.Fatalf("expected column hotspots for both diffs, got %+v", scope.TopDifferentColumns)
	}
	if len(scope.TopReferenceDarkerColumns) != 1 || scope.TopReferenceDarkerColumns[0].Index != 0 || len(scope.TopReferenceLighterColumns) != 1 || scope.TopReferenceLighterColumns[0].Index != 1 {
		t.Fatalf("expected direction column hotspots, got darker=%+v lighter=%+v", scope.TopReferenceDarkerColumns, scope.TopReferenceLighterColumns)
	}
	if len(scope.TopReferenceRGBDeltaSums8) != 2 || scope.TopReferenceRGBDeltaSums8[0].Delta != -45 || scope.TopReferenceRGBDeltaSums8[1].Delta != 45 {
		t.Fatalf("expected signed RGB delta hotspots, got %+v", scope.TopReferenceRGBDeltaSums8)
	}
	if len(scope.TopGotColors) != 2 || scope.TopGotColors[0].RGBA != "#F0F0F0/FF" || scope.TopGotColors[1].RGBA != "#FFFFFF/FF" {
		t.Fatalf("expected dominant got colors, got %+v", scope.TopGotColors)
	}
	if len(scope.TopReferenceColors) != 2 || scope.TopReferenceColors[0].RGBA != "#F0F0F0/FF" || scope.TopReferenceColors[1].RGBA != "#FFFFFF/FF" {
		t.Fatalf("expected dominant reference colors, got %+v", scope.TopReferenceColors)
	}
	if len(scope.TopDifferentGotColors) != 2 || scope.TopDifferentGotColors[0].RGBA != "#F0F0F0/FF" || scope.TopDifferentGotColors[1].RGBA != "#FFFFFF/FF" {
		t.Fatalf("expected differing got color buckets, got %+v", scope.TopDifferentGotColors)
	}
	if len(scope.TopDifferentReferenceColors) != 2 || scope.TopDifferentReferenceColors[0].RGBA != "#F0F0F0/FF" || scope.TopDifferentReferenceColors[1].RGBA != "#FFFFFF/FF" {
		t.Fatalf("expected differing reference color buckets, got %+v", scope.TopDifferentReferenceColors)
	}
}

func TestMicroFixtureShapeFillHeightSearchRanksReferenceFillAndHeight(t *testing.T) {
	got := image.NewRGBA(image.Rect(0, 0, 4, 3))
	reference := image.NewRGBA(image.Rect(0, 0, 4, 3))
	oldFill := color.RGBA{R: 224, G: 235, B: 246, A: 255}
	newFill := color.RGBA{R: 225, G: 235, B: 245, A: 255}
	text := color.RGBA{R: 47, G: 110, B: 186, A: 255}
	draw.Draw(got, got.Bounds(), &image.Uniform{C: oldFill}, image.Point{}, draw.Src)
	draw.Draw(reference, reference.Bounds(), &image.Uniform{C: newFill}, image.Point{}, draw.Src)
	for x := 0; x < 4; x++ {
		reference.SetRGBA(x, 2, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	}
	got.SetRGBA(1, 1, text)
	reference.SetRGBA(1, 1, text)

	artifact := searchMicroFixtureShapeFillHeight(got, reference, objectFailureRecord{
		PixelBounds: ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 3, MaxY: 1},
	})
	if len(artifact.Candidates) == 0 {
		t.Fatal("expected fill/height candidates")
	}
	best := artifact.Candidates[0]
	if best.DifferentPixels != 0 || best.FillColor != "#E1EBF5/FF" || best.HeightPixels != 2 {
		t.Fatalf("expected reference fill and geometry height to be ranked first, got %+v", best)
	}
}

func TestMicroFixtureShapeResidualTextProfileClassifiesTextFillAndWhite(t *testing.T) {
	got := image.NewRGBA(image.Rect(0, 0, 4, 1))
	reference := image.NewRGBA(image.Rect(0, 0, 4, 1))
	fill := color.RGBA{R: 225, G: 235, B: 245, A: 255}
	text := color.RGBA{R: 47, G: 110, B: 186, A: 255}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	draw.Draw(got, got.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)
	draw.Draw(reference, reference.Bounds(), &image.Uniform{C: fill}, image.Point{}, draw.Src)
	got.SetRGBA(0, 0, text)
	reference.SetRGBA(1, 0, text)
	reference.SetRGBA(2, 0, white)
	got.SetRGBA(3, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	reference.SetRGBA(3, 0, color.RGBA{R: 20, G: 30, B: 40, A: 255})

	profile := shapeResidualTextProfile(got, reference, text, fill)
	if profile.DifferentPixels != 4 {
		t.Fatalf("expected four residual differences, got %+v", profile)
	}
	if profile.EitherTextLikeDifferentPixels != 2 || profile.GotTextLikeDifferentPixels != 1 || profile.ReferenceTextLikeDifferentPixels != 1 {
		t.Fatalf("expected text-like residual split, got %+v", profile)
	}
	if profile.ReferenceWhiteLikeDifferentPixels != 1 || profile.GotFillLikeDifferentPixels != 2 || profile.ReferenceFillLikeDifferentPixels != 1 {
		t.Fatalf("expected fill/white residual split, got %+v", profile)
	}
	if profile.GotOtherDifferentPixels != 1 || profile.ReferenceOtherDifferentPixels != 1 {
		t.Fatalf("expected one uncategorized residual on each side, got %+v", profile)
	}
}

func TestMicroFixtureShadowAlphaDiagnosticEstimatesReferenceAlphaDirection(t *testing.T) {
	dir := t.TempDir()
	gotPath := filepath.Join(dir, "got-crop.png")
	referencePath := filepath.Join(dir, "reference-crop.png")
	backgroundPath := filepath.Join(dir, "before-crop.png")
	maskPath := filepath.Join(dir, "object.png")

	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	background := image.NewRGBA(image.Rect(0, 0, 2, 1))
	mask := image.NewRGBA(image.Rect(0, 0, 2, 1))
	background.SetRGBA(0, 0, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	background.SetRGBA(1, 0, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	got.SetRGBA(0, 0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	reference.SetRGBA(0, 0, color.RGBA{R: 80, G: 80, B: 80, A: 255})
	got.SetRGBA(1, 0, color.RGBA{R: 80, G: 80, B: 80, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	mask.SetRGBA(0, 0, color.RGBA{A: 128})
	mask.SetRGBA(1, 0, color.RGBA{A: 128})
	for path, img := range map[string]*image.RGBA{
		gotPath:        got,
		referencePath:  reference,
		backgroundPath: background,
		maskPath:       mask,
	} {
		if err := writePNG(path, img); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	bounds := ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 0}
	scope, err := microFixtureShadowAlphaDiagnostic(gotPath, referencePath, backgroundPath, objectFailureRecord{
		OutputPixelBounds:  &bounds,
		ObjectArtifactPath: maskPath,
	})
	if err != nil {
		t.Fatalf("shadow alpha diagnostic failed: %v", err)
	}
	if scope.ShadowMaskPixels != 2 || scope.AnalyzedPixels != 2 {
		t.Fatalf("expected two analyzed partial-alpha pixels, got %+v", scope)
	}
	if scope.ReferenceAlphaGreaterPixels != 1 || scope.ReferenceAlphaLessPixels != 1 {
		t.Fatalf("expected one greater and one lower reference alpha, got %+v", scope)
	}
	if scope.ReferenceAlphaGreaterDeltaSum8 != 25 || scope.ReferenceAlphaLessDeltaSum8 != -25 {
		t.Fatalf("expected per-direction delta sums, got %+v", scope)
	}
	if scope.ReferenceAlphaGreaterBounds == nil || scope.ReferenceAlphaGreaterBounds.MinX != 0 || scope.ReferenceAlphaGreaterBounds.MaxX != 0 {
		t.Fatalf("expected greater alpha bounds at first pixel, got %+v", scope.ReferenceAlphaGreaterBounds)
	}
	if scope.ReferenceAlphaLessBounds == nil || scope.ReferenceAlphaLessBounds.MinX != 1 || scope.ReferenceAlphaLessBounds.MaxX != 1 {
		t.Fatalf("expected less alpha bounds at second pixel, got %+v", scope.ReferenceAlphaLessBounds)
	}
	if scope.ReferenceAlphaGreaterCentroid == nil || scope.ReferenceAlphaGreaterCentroid.X != 0 || scope.ReferenceAlphaGreaterCentroid.Y != 0 {
		t.Fatalf("expected greater alpha centroid at first pixel, got %+v", scope.ReferenceAlphaGreaterCentroid)
	}
	if scope.ReferenceAlphaLessCentroid == nil || scope.ReferenceAlphaLessCentroid.X != 1 || scope.ReferenceAlphaLessCentroid.Y != 0 {
		t.Fatalf("expected less alpha centroid at second pixel, got %+v", scope.ReferenceAlphaLessCentroid)
	}
	if scope.ReferenceAlphaDeltaSum8 != 0 || scope.ReferenceAlphaAbsoluteDeltaSum8 != 50 {
		t.Fatalf("expected balanced alpha delta with absolute 50, got %+v", scope)
	}
	if len(scope.TopReferenceAlphaDeltaSums8) != 2 || scope.TopReferenceAlphaDeltaSums8[0].Delta != -25 || scope.TopReferenceAlphaDeltaSums8[1].Delta != 25 {
		t.Fatalf("expected alpha delta buckets, got %+v", scope.TopReferenceAlphaDeltaSums8)
	}
	if len(scope.TopReferenceAlphaGreaterColumns) != 1 || scope.TopReferenceAlphaGreaterColumns[0].Index != 0 || len(scope.TopReferenceAlphaLessColumns) != 1 || scope.TopReferenceAlphaLessColumns[0].Index != 1 {
		t.Fatalf("expected alpha direction columns, got greater=%+v less=%+v", scope.TopReferenceAlphaGreaterColumns, scope.TopReferenceAlphaLessColumns)
	}
}

func TestWriteShadowAlphaCorrectionHeatmapMarksDirection(t *testing.T) {
	dir := t.TempDir()
	gotPath := filepath.Join(dir, "got-crop.png")
	referencePath := filepath.Join(dir, "reference-crop.png")
	backgroundPath := filepath.Join(dir, "before-crop.png")
	maskPath := filepath.Join(dir, "object.png")
	heatmapPath := filepath.Join(dir, "heatmap.png")

	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	background := image.NewRGBA(image.Rect(0, 0, 2, 1))
	mask := image.NewRGBA(image.Rect(0, 0, 2, 1))
	background.SetRGBA(0, 0, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	background.SetRGBA(1, 0, color.RGBA{R: 100, G: 100, B: 100, A: 255})
	got.SetRGBA(0, 0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	reference.SetRGBA(0, 0, color.RGBA{R: 80, G: 80, B: 80, A: 255})
	got.SetRGBA(1, 0, color.RGBA{R: 80, G: 80, B: 80, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{R: 90, G: 90, B: 90, A: 255})
	mask.SetRGBA(0, 0, color.RGBA{A: 128})
	mask.SetRGBA(1, 0, color.RGBA{A: 128})
	for path, img := range map[string]*image.RGBA{
		gotPath:        got,
		referencePath:  reference,
		backgroundPath: background,
		maskPath:       mask,
	} {
		if err := writePNG(path, img); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	bounds := ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 0}
	if err := writeShadowAlphaCorrectionHeatmap(gotPath, referencePath, backgroundPath, heatmapPath, objectFailureRecord{
		OutputPixelBounds:  &bounds,
		ObjectArtifactPath: maskPath,
	}); err != nil {
		t.Fatalf("write heatmap: %v", err)
	}
	heatmap := decodePNG(t, heatmapPath)
	needMore := color.RGBAModel.Convert(heatmap.At(0, 0)).(color.RGBA)
	needLess := color.RGBAModel.Convert(heatmap.At(1, 0)).(color.RGBA)
	if needMore.R == 0 || needMore.B != 0 || needMore.A == 0 {
		t.Fatalf("expected first pixel to mark more alpha in red, got %#v", needMore)
	}
	if needLess.B == 0 || needLess.R != 0 || needLess.A == 0 {
		t.Fatalf("expected second pixel to mark less alpha in blue, got %#v", needLess)
	}
}

func TestMicroFixtureUnderpaintChainSummaryForScopesReportsDeltas(t *testing.T) {
	got := microFixtureUnderpaintChainSummaryForScopes(microFixtureTargetScope{
		DifferentPixels: 540,
		DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint:    92,
		DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint: 448,
	}, microFixtureTargetScope{
		ComparedPixels:  321300,
		DifferentPixels: 536,
		DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint:    88,
		DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint: 448,
		ReferenceRGBDeltaSum8:         -1082,
		ReferenceRGBAbsoluteDeltaSum8: 1100,
	})
	if got.ObjectOnlyDifferentPixels != 540 || got.ChainDifferentPixels != 536 || got.DifferentPixelsDelta != -4 {
		t.Fatalf("unexpected chain total summary: %+v", got)
	}
	if got.ObjectOnlyUnderpaintedPartialAlphaDifferentPixels != 92 || got.ChainUnderpaintedPartialAlphaDifferentPixels != 88 || got.UnderpaintedPartialAlphaDifferentPixelsDelta != -4 {
		t.Fatalf("unexpected underpainted summary: %+v", got)
	}
	if got.ObjectOnlyPlainPartialAlphaDifferentPixels != 448 || got.ChainPlainPartialAlphaDifferentPixels != 448 || got.PlainPartialAlphaDifferentPixelsDelta != 0 {
		t.Fatalf("unexpected plain partial-alpha summary: %+v", got)
	}
	if got.ChainReferenceRGBDeltaSum8 != -1082 || got.ChainReferenceRGBAbsoluteDeltaSum8 != 1100 {
		t.Fatalf("unexpected chain delta sums: %+v", got)
	}
}

func TestCleanMicroFixtureOwnershipFailureExcludesUnderpaintConfoundedEdges(t *testing.T) {
	clean := microFixtureOwnershipRecord{
		DifferentPixels:                  12,
		DifferentPixelsInsideObjectMask:  12,
		DifferentPixelsOutsideObjectMask: 0,
	}
	if !isCleanMicroFixtureOwnershipFailure(clean) {
		t.Fatalf("expected object-owned failure with no underpaint overlap to be clean")
	}
	underpainted := clean
	underpainted.PartialAlphaOverUnderpaintPixels = 1
	if isCleanMicroFixtureOwnershipFailure(underpainted) {
		t.Fatalf("expected partial-alpha underpaint overlap to disqualify a clean ownership failure")
	}
	contaminated := clean
	contaminated.DifferentPixelsOutsideObjectMask = 1
	if isCleanMicroFixtureOwnershipFailure(contaminated) {
		t.Fatalf("expected outside-mask pixels to disqualify a clean ownership failure")
	}
}

func TestWriteNonUnderpaintedTargetPNGUsesUnderpaintMaskAlpha(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "source.png")
	targetPath := filepath.Join(dir, "target.png")
	underpaintPath := filepath.Join(dir, "underpaint.png")

	source := image.NewRGBA(image.Rect(0, 0, 3, 1))
	source.SetRGBA(0, 0, color.RGBA{R: 10, A: 255})
	source.SetRGBA(1, 0, color.RGBA{R: 20, A: 255})
	source.SetRGBA(2, 0, color.RGBA{R: 30, A: 255})
	underpaint := image.NewRGBA(image.Rect(0, 0, 5, 1))
	underpaint.SetRGBA(2, 0, color.RGBA{A: 255})
	if err := writePNG(sourcePath, source); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(underpaintPath, underpaint); err != nil {
		t.Fatal(err)
	}
	if err := writeNonUnderpaintedTargetPNG(sourcePath, targetPath, ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 3, MaxY: 0}, []microFixtureUnderpaint{{
		Bounds:             ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 3, MaxY: 0},
		ObjectArtifactPath: underpaintPath,
	}}); err != nil {
		t.Fatalf("write non-underpaint target: %v", err)
	}
	got, err := decodePNGFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, a0 := got.At(0, 0).RGBA()
	_, _, _, a1 := got.At(1, 0).RGBA()
	_, _, _, a2 := got.At(2, 0).RGBA()
	if a0 == 0 || a1 != 0 || a2 == 0 {
		t.Fatalf("expected only the masked underpaint pixel to become transparent, got alpha=%04x,%04x,%04x", a0, a1, a2)
	}
}

func TestWriteMicroFixtureGeometryArtifactsUsesFullObjectPixelBounds(t *testing.T) {
	dir := t.TempDir()
	gotPath := filepath.Join(dir, "got.png")
	referencePath := filepath.Join(dir, "reference.png")
	got := image.NewRGBA(image.Rect(0, 0, 4, 2))
	reference := image.NewRGBA(image.Rect(0, 0, 4, 2))
	draw.Draw(got, got.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(reference, reference.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	got.SetRGBA(2, 1, color.RGBA{G: 255, A: 255})
	reference.SetRGBA(2, 1, color.RGBA{B: 255, A: 255})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}

	artifacts := writeMicroFixtureGeometryArtifacts(t, "testdata/realworld-ppts/example.pptx", 1, dir, gotPath, referencePath, objectFailureRecord{
		CNvPrID:     "2",
		CNvPrName:   "Picture",
		PixelBounds: ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 3, MaxY: 1},
	})
	for _, path := range []string{artifacts.gotPath, artifacts.referencePath, artifacts.diffPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected geometry artifact %s: %v", path, err)
		}
	}
	diff, err := comparePNG(artifacts.gotPath, artifacts.referencePath)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Width != 3 || diff.Height != 2 || diff.DifferentPixels != 1 {
		t.Fatalf("unexpected geometry crop diff: %+v", diff)
	}
}

func TestWriteMicroFixtureSourceArtifactsCropsBeforeAndThroughRenders(t *testing.T) {
	dir := t.TempDir()
	beforePath := filepath.Join(dir, "before.png")
	throughPath := filepath.Join(dir, "through.png")
	referencePath := filepath.Join(dir, "reference-crop.png")
	before := image.NewRGBA(image.Rect(0, 0, 4, 2))
	through := image.NewRGBA(image.Rect(0, 0, 4, 2))
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	draw.Draw(before, before.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(through, through.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(reference, reference.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	before.SetRGBA(1, 1, color.RGBA{R: 255, A: 255})
	through.SetRGBA(2, 1, color.RGBA{G: 255, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})
	if err := writePNG(beforePath, before); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(throughPath, through); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}

	crop := ObjectPixelBounds{MinX: 1, MinY: 1, MaxX: 2, MaxY: 1}
	referenceVisiblePath := filepath.Join(dir, "reference-visible-crop.png")
	if err := writeVisibleCroppedPNG(referencePath, referenceVisiblePath, ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 0}, []microFixtureOcclusion{{
		Bounds: ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 1, MaxY: 0},
	}}); err != nil {
		t.Fatal(err)
	}
	artifacts := writeMicroFixtureSourceArtifacts(t, "testdata/realworld-ppts/example.pptx", 1, dir, referencePath, microFixtureVisibleArtifacts{
		referencePath: referenceVisiblePath,
		occlusions: []microFixtureOcclusion{{
			Bounds: ObjectPixelBounds{MinX: 2, MinY: 1, MaxX: 2, MaxY: 1},
		}},
	}, objectFailureRecord{
		CNvPrID:             "2",
		CNvPrName:           "Rectangle",
		OutputPixelBounds:   &crop,
		BeforeArtifactPath:  beforePath,
		ThroughArtifactPath: throughPath,
	})
	for _, path := range []string{artifacts.beforePath, artifacts.throughPath, artifacts.throughDiffPath, artifacts.throughVisiblePath, artifacts.throughVisibleDiffPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected source artifact %s: %v", path, err)
		}
	}
	beforeCrop := decodePNG(t, artifacts.beforePath)
	if got := color.RGBAModel.Convert(beforeCrop.At(0, 0)).(color.RGBA); got.R != 255 || got.G != 0 || got.B != 0 {
		t.Fatalf("expected before crop to preserve source underpaint pixel, got %#v", got)
	}
	diff, err := comparePNG(artifacts.throughPath, referencePath)
	if err != nil {
		t.Fatal(err)
	}
	if diff.Width != 2 || diff.Height != 1 || diff.DifferentPixels != 1 {
		t.Fatalf("unexpected source-through diff: %+v", diff)
	}
	visibleDiff, err := comparePNG(artifacts.throughVisiblePath, referenceVisiblePath)
	if err != nil {
		t.Fatal(err)
	}
	if visibleDiff.DifferentPixels != 0 {
		t.Fatalf("expected occluded source-through visible crop to match masked reference, got %+v", visibleDiff)
	}
}

func TestRenderMicroFixtureWithObjectDebugWritesFixtureRecords(t *testing.T) {
	dir := t.TempDir()
	fixturePath := filepath.Join(dir, "fixture.pptx")
	if err := writeObjectDebugPPTX(fixturePath); err != nil {
		t.Fatal(err)
	}
	recordsPath := filepath.Join(dir, "fixture-objects.json")
	_, err := renderMicroFixtureWithObjectDebug(fixturePath, filepath.Join(dir, "got.png"), recordsPath, filepath.Join(dir, "fixture-objects"))
	if err != nil {
		t.Fatalf("render micro-fixture with debug: %v", err)
	}
	data, err := os.ReadFile(recordsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"before_artifact_path"`) || !strings.Contains(string(data), `"object_artifact_path"`) || !strings.Contains(string(data), `"through_artifact_path"`) {
		t.Fatalf("fixture object records did not include debug artifact paths: %s", data)
	}
}

func TestWriteMicroFixtureSourceObjectXMLWritesRawObject(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "objects.pptx")
	if err := writeObjectDebugPPTX(deckPath); err != nil {
		t.Fatal(err)
	}
	path, summary := writeMicroFixtureSourceObjectXML(t, deckPath, 1, dir, objectFailureRecord{
		SourcePart: "ppt/slides/slide1.xml",
		Kind:       "sp",
		CNvPrID:    "2",
		CNvPrName:  "Red Rect",
		ZOrder:     1,
	})
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "<p:sp>") || !strings.Contains(text, `name="Red Rect"`) || strings.Contains(text, `name="Blue Rect"`) {
		t.Fatalf("unexpected source object XML: %s", text)
	}
	if summary == nil || summary.CNvPrID != "2" || summary.CNvPrName != "Red Rect" || summary.Transform.CX != 914400 || summary.Transform.CY != 914400 {
		t.Fatalf("unexpected source object summary: %+v", summary)
	}
}

func TestMicroFixtureSourceObjectSummaryParsesCustomPathAndShadow(t *testing.T) {
	raw := `<p:sp>
  <p:nvSpPr><p:cNvPr id="7" name="Freeform 6"/></p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="337167"/><a:ext cx="2963007" cy="923192"/></a:xfrm>
    <a:custGeom><a:pathLst><a:path w="2963007" h="923192">
      <a:moveTo><a:pt x="0" y="0"/></a:moveTo>
      <a:lnTo><a:pt x="2039815" y="0"/></a:lnTo>
      <a:lnTo><a:pt x="2963007" y="923192"/></a:lnTo>
      <a:lnTo><a:pt x="0" y="923192"/></a:lnTo>
      <a:close/>
    </a:path></a:pathLst></a:custGeom>
    <a:effectLst><a:outerShdw blurRad="127000" dist="63500" dir="1800000" algn="tl" rotWithShape="0"/></a:effectLst>
  </p:spPr>
</p:sp>`
	summary, err := microFixtureSourceObjectSummaryFromXML(raw, "sp")
	if err != nil {
		t.Fatalf("parse source object summary: %v", err)
	}
	if summary.Kind != "sp" || summary.CNvPrID != "7" || summary.CNvPrName != "Freeform 6" {
		t.Fatalf("unexpected object identity: %+v", summary)
	}
	if summary.Transform.X != 0 || summary.Transform.Y != 337167 || summary.Transform.CX != 2963007 || summary.Transform.CY != 923192 {
		t.Fatalf("unexpected transform: %+v", summary.Transform)
	}
	if summary.CustomPath == nil || summary.CustomPath.Width != 2963007 || summary.CustomPath.Height != 923192 || len(summary.CustomPath.Points) != 4 {
		t.Fatalf("unexpected custom path: %+v", summary.CustomPath)
	}
	if point := summary.CustomPath.Points[1]; point.Command != "lnTo" || point.X != 2039815 || point.Y != 0 {
		t.Fatalf("unexpected second point: %+v", point)
	}
	if summary.Shadow == nil || summary.Shadow.BlurRadius != 127000 || summary.Shadow.Distance != 63500 || summary.Shadow.Direction != 1800000 || summary.Shadow.Alignment != "tl" || summary.Shadow.RotateWithShape != "0" {
		t.Fatalf("unexpected shadow summary: %+v", summary.Shadow)
	}
}

func TestMicroFixtureShadowRenderSummaryRecordsDerivedPixelGeometry(t *testing.T) {
	summary, ok := microFixtureShadowRenderSummaryForCanvas(slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 100, 100), objectFailureRecord{
		PixelBounds: ObjectPixelBounds{MinX: 10, MinY: 20, MaxX: 29, MaxY: 39},
		ResolvedStyle: ObjectStyleSummary{
			Shadow:          true,
			ShadowColor:     "#000000/66",
			ShadowDistance:  100,
			ShadowDirection: 0,
			ShadowBlur:      50,
			CustomPathCoordinates: []ObjectFloatPoint{
				{X: 0, Y: 0},
				{X: 0.5, Y: 1},
				{X: 1, Y: 0},
			},
		},
	})
	if !ok {
		t.Fatal("expected shadow render summary")
	}
	if summary.Offset.X != 10 || summary.Offset.Y != 0 || summary.BlurPixels != 5 {
		t.Fatalf("unexpected offset/blur: %+v", summary)
	}
	if summary.ShadowBounds.MinX != 20 || summary.ShadowBounds.MaxX != 39 || summary.ShadowBounds.MinY != 20 || summary.ShadowBounds.MaxY != 39 {
		t.Fatalf("unexpected shadow bounds: %+v", summary.ShadowBounds)
	}
	if summary.PaintBounds.MinX != 15 || summary.PaintBounds.MaxX != 44 || summary.PaintBounds.MinY != 15 || summary.PaintBounds.MaxY != 44 {
		t.Fatalf("unexpected paint bounds: %+v", summary.PaintBounds)
	}
	if len(summary.TargetCustomPathPixelPoints) != 3 || summary.TargetCustomPathPixelPoints[1].X != 20 || summary.TargetCustomPathPixelPoints[1].Y != 40 {
		t.Fatalf("unexpected target custom path pixel points: %+v", summary.TargetCustomPathPixelPoints)
	}
	if len(summary.ShadowCustomPathPixelPoints) != 3 || summary.ShadowCustomPathPixelPoints[1].X != 30 || summary.ShadowCustomPathPixelPoints[1].Y != 40 {
		t.Fatalf("unexpected shadow custom path pixel points: %+v", summary.ShadowCustomPathPixelPoints)
	}
}

func TestMicroFixturePackagePartsRecordsDeterministicZipEntries(t *testing.T) {
	dir := t.TempDir()
	deckPath := filepath.Join(dir, "objects.pptx")
	if err := writeObjectDebugPPTX(deckPath); err != nil {
		t.Fatal(err)
	}
	parts, err := microFixturePackageParts(deckPath)
	if err != nil {
		t.Fatalf("read package parts: %v", err)
	}
	if len(parts) != 5 {
		t.Fatalf("expected five package parts, got %+v", parts)
	}
	if parts[0].Name != "[Content_Types].xml" || parts[0].Size <= 0 || len(parts[0].SHA256) != 64 {
		t.Fatalf("unexpected first package part: %+v", parts[0])
	}
	if parts[0].Reason == "" {
		t.Fatalf("expected package part reason, got %+v", parts[0])
	}
	if parts[1].Name != "_rels/.rels" {
		t.Fatalf("expected sorted package parts, got %+v", parts)
	}
}

func TestMicroFixtureSamplingForPictureRecordsGeometryAndCropMath(t *testing.T) {
	outputBounds := ObjectPixelBounds{MinX: 15, MinY: 23, MaxX: 24, MaxY: 42}
	sampling := microFixtureSamplingForPicture(microFixtureSourceImage{
		Width:  200,
		Height: 100,
	}, objectFailureRecord{
		PixelBounds:       ObjectPixelBounds{MinX: 10, MinY: 20, MaxX: 29, MaxY: 69},
		FractionalBounds:  ObjectFloatBounds{MinX: 9.25, MinY: 20.5, MaxX: 29.75, MaxY: 70.25},
		OutputPixelBounds: &outputBounds,
	})
	if sampling == nil {
		t.Fatal("expected sampling diagnostic")
	}
	if sampling.IntegerGeometryWidth != 20 || sampling.IntegerGeometryHeight != 50 {
		t.Fatalf("unexpected integer geometry size: %+v", sampling)
	}
	if sampling.OutputCropOffsetX != 5 || sampling.OutputCropOffsetY != 3 || sampling.OutputCropWidth != 10 || sampling.OutputCropHeight != 20 {
		t.Fatalf("unexpected output crop diagnostic: %+v", sampling)
	}
	if sampling.FractionalOffsetX != -0.75 || sampling.FractionalOffsetY != 0.5 {
		t.Fatalf("unexpected fractional offset: %+v", sampling)
	}
	if sampling.SourceToGeometryScaleX != 0.1025 || sampling.SourceToGeometryScaleY != 0.4975 {
		t.Fatalf("unexpected source scale: %+v", sampling)
	}
}

func TestMicroFixtureManifestComparison(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_MICRO_FIXTURE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_MICRO_FIXTURE_MANIFEST to verify one extracted object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debugDir := microFixtureDebugDir()
	var debug *ObjectDebugOptions
	if debugDir != "" {
		outputDir = debugDir
		debug = &ObjectDebugOptions{}
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("create micro-fixture debug dir: %v", err)
		}
	}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := outputPath
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "crop"
	if len(manifest.OccludedBy) > 0 {
		targetLabel = "visible crop"
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if manifest.Object.OutputPixelBounds == nil {
			t.Fatal("micro-fixture manifest has occlusions but no output pixel bounds")
		}
		if err := writeVisibleCroppedPNG(outputPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write rerendered visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
	} else {
		gotTargetPath = filepath.Join(outputDir, "got-crop.png")
		if manifest.Object.OutputPixelBounds == nil {
			t.Fatal("micro-fixture manifest has no output pixel bounds")
		}
		if err := writeCroppedPNG(outputPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
			t.Fatalf("write rerendered crop: %v", err)
		}
	}
	diff, err := comparePNG(gotTargetPath, referenceTargetPath)
	if err != nil {
		t.Fatalf("compare rerendered micro-fixture %s: %v", targetLabel, err)
	}
	if debugDir != "" {
		if err := writeJSONFile(filepath.Join(outputDir, "micro-fixture-diff.json"), diff); err != nil {
			t.Fatalf("write micro-fixture debug diff: %v", err)
		}
		if err := writeDiffPNG(gotTargetPath, referenceTargetPath, filepath.Join(outputDir, "micro-fixture-diff.png")); err != nil {
			t.Fatalf("write micro-fixture debug diff image: %v", err)
		}
		scope, err := microFixtureTargetScopeDiagnostic(gotTargetPath, referenceTargetPath, manifest.Object, manifest.UnderpaintedBy)
		if err != nil {
			t.Fatalf("write micro-fixture debug target scope: %v", err)
		}
		scope.TargetCompared = filepath.Base(gotTargetPath) + " vs " + filepath.Base(referenceTargetPath)
		if err := writeJSONFile(filepath.Join(outputDir, "target-scope.json"), scope); err != nil {
			t.Fatalf("write micro-fixture debug target scope: %v", err)
		}
		summaryObject := manifest.Object
		if debug != nil {
			if err := writeJSONFile(filepath.Join(outputDir, "fixture-objects.json"), debug.Records); err != nil {
				t.Fatalf("write micro-fixture debug object records: %v", err)
			}
			if record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object); ok {
				summaryObject = objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
				if err := writeJSONFile(filepath.Join(outputDir, "current-object.json"), summaryObject); err != nil {
					t.Fatalf("write micro-fixture debug current object: %v", err)
				}
			}
		}
		fixturePkg, err := pptx.Open(context.Background(), fixturePath)
		if err != nil {
			t.Fatalf("open micro-fixture for debug shadow render summary: %v", err)
		}
		size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
		if summary, ok := microFixtureShadowRenderSummaryForCanvas(size, decodePNG(t, outputPath).Bounds(), summaryObject); ok {
			if err := writeJSONFile(filepath.Join(outputDir, "shadow-render-summary.json"), summary); err != nil {
				t.Fatalf("write micro-fixture debug shadow render summary: %v", err)
			}
		}
	}
	if diff.DifferentPixels != 0 {
		t.Fatalf("micro-fixture %s mismatch for %s slide %d object %s %q: %d differing pixel(s), bounds=%+v; schema=%s; source=%s %s; got=%s; reference=%s", targetLabel, manifest.DeckInput, manifest.SlideNumber, manifest.Object.CNvPrID, manifest.Object.CNvPrName, diff.DifferentPixels, diff.DifferentBounds, strings.Join(microFixtureManifestSchemaAnchors(manifest), ", "), microFixtureManifestSourcePart(manifest), microFixtureManifestSourcePath(manifest), gotTargetPath, referenceTargetPath)
	}
}

func microFixtureManifestSchemaAnchors(manifest microFixtureManifest) []string {
	if len(manifest.SpecFixture.SchemaAnchors) > 0 {
		return append([]string(nil), manifest.SpecFixture.SchemaAnchors...)
	}
	return schemaAnchorsForFixtureObject(manifest.Object)
}

func microFixtureManifestSourcePart(manifest microFixtureManifest) string {
	if manifest.SpecFixture.SourceXMLPart != "" {
		return manifest.SpecFixture.SourceXMLPart
	}
	return manifest.Object.SourcePart
}

func microFixtureManifestSourcePath(manifest microFixtureManifest) string {
	if manifest.SpecFixture.SourceXMLPath != "" {
		return manifest.SpecFixture.SourceXMLPath
	}
	return manifest.Object.XMLPath
}

func TestMicroFixtureShapeObjectProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_OBJECT_PROFILE_MANIFEST to profile one extracted shape object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "sp" && manifest.Object.Kind != "cxnSp" {
		t.Fatalf("shape profile requires a shape object, got kind=%q", manifest.Object.Kind)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape profile requires output pixel bounds")
	}

	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write shape profile visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	} else if err := writeCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape profile crop: %v", err)
	}
	diff, err := comparePNG(gotTargetPath, referenceTargetPath)
	if err != nil {
		t.Fatalf("compare shape profile target: %v", err)
	}
	scope, err := microFixtureTargetScopeDiagnostic(gotTargetPath, referenceTargetPath, manifest.Object, manifest.UnderpaintedBy)
	if err != nil {
		t.Fatalf("analyze shape profile target scope: %v", err)
	}
	scope.TargetCompared = targetCompared

	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open shape profile fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("shape profile fixture has no slides")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	canvas := decodePNG(t, renderPath).Bounds()
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	element, ok := findMicroFixtureSlideElement(elements, manifest.Object)
	if !ok {
		t.Fatalf("shape profile object not found in fixture elements: %s %q", manifest.Object.CNvPrID, manifest.Object.CNvPrName)
	}
	geometryTarget := elementPixelTarget(element, size, canvas)
	textTarget := geometryTarget
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		if adjusted, err := shapeAutofitTarget(element, textTarget, size, canvas); err == nil {
			textTarget = adjusted
		}
	}
	dpi := renderDPIForCanvas(size, canvas)
	textBoundsBefore := textBounds(geometryTarget, element, size, canvas)
	textBoundsAfter := textBounds(textTarget, element, size, canvas)
	measuredWidth, measuredHeight, measureErr := measuredElementTextSize(element, textBoundsBefore, dpi)
	profile := microFixtureShapeObjectProfile{
		ManifestPath:        manifestPath,
		FixturePath:         fixturePath,
		DeckInput:           manifest.DeckInput,
		SlideNumber:         manifest.SlideNumber,
		CNvPrID:             manifest.Object.CNvPrID,
		CNvPrName:           manifest.Object.CNvPrName,
		Kind:                manifest.Object.Kind,
		HasShapeAutofit:     element.HasShapeAutofit,
		HasNormalAutofit:    element.HasNormAutofit,
		HasNoAutofit:        element.HasNoAutofit,
		TextWrap:            element.TextWrap,
		Canvas:              microFixtureSize{Width: canvas.Dx(), Height: canvas.Dy()},
		GeometryTarget:      pixelBoundsFromRect(geometryTarget),
		TextTarget:          pixelBoundsFromRect(textTarget),
		TextBoundsBeforeFit: pixelBoundsFromRect(textBoundsBefore),
		TextBoundsAfterFit:  pixelBoundsFromRect(textBoundsAfter),
		FillColor:           formatObjectColor(element.FillColor),
		TextColor:           manifest.Object.ResolvedStyle.TextColor,
		Diff:                diff,
		TargetScope:         scope,
	}
	if measureErr != nil {
		profile.MeasureError = measureErr.Error()
	} else {
		profile.MeasuredTextWidth = measuredWidth
		profile.MeasuredTextHeight = measuredHeight
	}
	t.Logf("shape profile %s slide %d object %s %q: diff=%d geometry=%+v text_target=%+v fill=%s top_ref=%+v top_got=%+v", profile.DeckInput, profile.SlideNumber, profile.CNvPrID, profile.CNvPrName, profile.Diff.DifferentPixels, profile.GeometryTarget, profile.TextTarget, profile.FillColor, firstColorCount(profile.TargetScope.TopReferenceColors), firstColorCount(profile.TargetScope.TopGotColors))
	if outputPath := os.Getenv("PUPPT_SHAPE_OBJECT_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write shape object profile: %v", err)
		}
	}
}

func TestMicroFixtureShapeTextStrokeProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_TEXT_STROKE_PROFILE_MANIFEST to profile text/stroke residuals for one shape micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "sp" && manifest.Object.Kind != "cxnSp" {
		t.Fatalf("shape text/stroke profile requires a shape object, got kind=%q", manifest.Object.Kind)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape text/stroke profile requires output pixel bounds")
	}

	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write shape text/stroke visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	} else if err := writeCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape text/stroke crop: %v", err)
	}
	got, err := decodePNGFile(gotTargetPath)
	if err != nil {
		t.Fatalf("decode shape text/stroke got crop: %v", err)
	}
	reference, err := decodePNGFile(referenceTargetPath)
	if err != nil {
		t.Fatalf("decode shape text/stroke reference crop: %v", err)
	}
	textColor, ok := parseObjectColorRGBA(manifest.Object.ResolvedStyle.TextColor)
	if !ok {
		textColor = dominantResidualTextColor(reference, color.RGBA{})
	}
	fillColor, ok := dominantImageColor(got)
	if !ok {
		t.Fatal("shape text/stroke profile could not determine dominant got fill color")
	}
	profile := microFixtureShapeTextStrokeProfile(got, reference, textColor, fillColor)
	profile.ManifestPath = manifestPath
	profile.FixturePath = fixturePath
	profile.TargetCompared = targetCompared
	profile.TextColor = formatObjectColor(textColor)
	profile.FillColor = formatObjectColor(fillColor)
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open shape text/stroke fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("shape text/stroke fixture has no slide parts")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	canvas := decodePNG(t, renderPath).Bounds()
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	if element, ok := findMicroFixtureSlideElement(elements, manifest.Object); ok {
		anchorMetrics, err := shapeTextAnchorMetrics(element, size, canvas)
		if err != nil {
			t.Fatalf("measure shape text anchor metrics: %v", err)
		}
		profile.AnchorMetrics = anchorMetrics
		profile.AnchorCandidates = shapeTextAnchorCandidates(got, reference, element, size, canvas, *manifest.Object.OutputPixelBounds, textColor, fillColor, *anchorMetrics)
		profile.FontCandidates = shapeTextFontCandidates(got, reference, element, size, canvas, *manifest.Object.OutputPixelBounds, textColor, fillColor)
	}
	t.Logf("shape text/stroke profile: diff=%d got_text=%+v ref_text=%+v edge=%d text_like=%d best_shift=%+v", profile.Baseline.DifferentPixels, profile.GotTextMask.Bounds, profile.ReferenceTextMask.Bounds, profile.Edge.DifferentPixels, profile.TextLikeDifferentPixels, firstShapeTextShiftCandidate(profile.ShiftCandidates))
	if outputPath := os.Getenv("PUPPT_SHAPE_TEXT_STROKE_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write shape text/stroke profile: %v", err)
		}
	}
}

func TestMicroFixtureShapeFillHeightSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_FILL_HEIGHT_SEARCH_MANIFEST to search fill/height candidates for one shape micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "sp" && manifest.Object.Kind != "cxnSp" {
		t.Fatalf("shape fill/height search requires a shape object, got kind=%q", manifest.Object.Kind)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape fill/height search requires output pixel bounds")
	}

	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write shape fill/height visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	} else if err := writeCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape fill/height crop: %v", err)
	}
	got, err := decodePNGFile(gotTargetPath)
	if err != nil {
		t.Fatalf("decode shape fill/height got crop: %v", err)
	}
	reference, err := decodePNGFile(referenceTargetPath)
	if err != nil {
		t.Fatalf("decode shape fill/height reference crop: %v", err)
	}
	artifact := searchMicroFixtureShapeFillHeight(got, reference, manifest.Object)
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		best := artifact.Candidates[0]
		t.Logf("shape fill/height search: baseline=%d best=%s/%s/%dpx -> %d", artifact.Baseline.DifferentPixels, best.Name, best.FillColor, best.HeightPixels, best.DifferentPixels)
	}
	if outputPath := os.Getenv("PUPPT_SHAPE_FILL_HEIGHT_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shape fill/height search: %v", err)
		}
	}
}

func TestMicroFixtureShapeResidualTextProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_MANIFEST to profile residual shape/text pixels after fill-height normalization")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape residual profile requires output pixel bounds")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write residual profile visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	} else if err := writeCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
		t.Fatalf("write residual profile crop: %v", err)
	}
	got, err := decodePNGFile(gotTargetPath)
	if err != nil {
		t.Fatalf("decode residual profile got crop: %v", err)
	}
	reference, err := decodePNGFile(referenceTargetPath)
	if err != nil {
		t.Fatalf("decode residual profile reference crop: %v", err)
	}
	search := searchMicroFixtureShapeFillHeight(got, reference, manifest.Object)
	if len(search.Candidates) == 0 {
		t.Fatal("shape residual profile requires fill/height candidates")
	}
	best := search.Candidates[0]
	currentFill, ok := dominantImageColor(got)
	if !ok {
		t.Fatal("shape residual profile could not determine current fill")
	}
	bestFill, ok := parseObjectColorRGBA(best.FillColor)
	if !ok {
		t.Fatalf("shape residual profile could not parse best fill color %q", best.FillColor)
	}
	normalized := renderShapeFillHeightCandidate(got, currentFill, bestFill, best.HeightPixels)
	textColor := dominantResidualTextColor(reference, bestFill)
	profile := microFixtureShapeResidualTextProfileArtifact{
		ManifestPath:         manifestPath,
		FixturePath:          fixturePath,
		TargetCompared:       targetCompared,
		Basis:                "diagnostic only: classify residual pixels after applying the best fill/height candidate to the current crop",
		NormalizedCandidate:  best,
		TextColor:            formatObjectColor(textColor),
		NormalizedTargetDiff: compareImages(normalized, reference),
	}
	profile.Residual = shapeResidualTextProfile(normalized, reference, textColor, bestFill)
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open residual profile fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("residual profile fixture has no slide parts")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	canvas := decodePNG(t, renderPath).Bounds()
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	if element, ok := findMicroFixtureSlideElement(elements, manifest.Object); ok {
		profile.ParsedTextCandidates = shapeParsedTextCandidates(normalized, reference, element, size, canvas, *manifest.Object.OutputPixelBounds, textColor, bestFill)
	}
	t.Logf("shape residual text profile: normalized=%d either_text=%d both_text=%d ref_fill=%d ref_white=%d", profile.NormalizedTargetDiff.DifferentPixels, profile.Residual.EitherTextLikeDifferentPixels, profile.Residual.BothTextLikeDifferentPixels, profile.Residual.ReferenceFillLikeDifferentPixels, profile.Residual.ReferenceWhiteLikeDifferentPixels)
	if outputPath := os.Getenv("PUPPT_SHAPE_RESIDUAL_TEXT_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write shape residual text profile: %v", err)
		}
	}
}

func TestMicroFixtureShapeLuminanceColorSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_MANIFEST to compare DrawingML luminance color candidates for one shape micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceObjectXMLPath == "" {
		t.Fatal("luminance color search requires source object XML")
	}
	sourceXML, err := os.ReadFile(resolveTestArtifactPath(manifest.SourceObjectXMLPath))
	if err != nil {
		t.Fatalf("read source object XML: %v", err)
	}
	sourceRoot, err := parseXMLNode(sourceXML)
	if err != nil {
		t.Fatalf("parse source object XML: %v", err)
	}
	solidFill := firstDescendant(sourceRoot, "solidFill")
	if solidFill == nil {
		t.Fatal("source object has no solid fill")
	}
	scheme := firstChild(solidFill, "schemeClr")
	if scheme == nil {
		t.Fatal("source object solid fill is not a scheme color")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open luminance fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("luminance fixture has no slide parts")
	}
	theme := themeColorsForPart(pkg, pkg.SlideParts[0], packageThemeColors(pkg))
	slot := attrValue(scheme.Attrs, "val")
	base, ok := schemeColorWithTheme(slot, theme)
	if !ok {
		t.Fatalf("resolve scheme color %q", slot)
	}
	lumMod, lumOff := luminanceModifiersFromColorNode(scheme)
	referenceColor, ok := dominantReferenceFillColor(manifest)
	if !ok {
		t.Fatal("could not determine reference dominant fill color")
	}
	gotColor, ok := dominantGotFillColor(manifest)
	if !ok {
		t.Fatal("could not determine got dominant fill color")
	}
	candidates := shapeLuminanceColorCandidates(base, lumMod, lumOff, referenceColor, gotColor)
	artifact := microFixtureShapeLuminanceColorSearchArtifact{
		ManifestPath:      manifestPath,
		FixturePath:       fixturePath,
		SchemeSlot:        slot,
		BaseColor:         formatObjectColor(base),
		LumMod:            lumMod,
		LumOff:            lumOff,
		GotDominant:       formatObjectColor(gotColor),
		ReferenceDominant: formatObjectColor(referenceColor),
		Candidates:        candidates,
	}
	t.Logf("shape luminance color search: base=%s mod=%d off=%d got=%s ref=%s best=%+v", artifact.BaseColor, lumMod, lumOff, artifact.GotDominant, artifact.ReferenceDominant, firstShapeLuminanceColorCandidate(candidates))
	if outputPath := os.Getenv("PUPPT_SHAPE_LUMINANCE_COLOR_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shape luminance color search: %v", err)
		}
	}
}

func TestMicroFixtureTableStyleColorProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_TABLE_STYLE_COLOR_PROFILE_MANIFEST to profile one table micro-fixture's source style colors")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "graphicFrame" {
		t.Fatalf("table style color profile requires a graphicFrame manifest, got %s", manifest.Object.Kind)
	}
	deckPath := realWorldDeckPath(manifest.DeckInput)
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatalf("open source deck: %v", err)
	}
	sourceData, ok := pkg.Parts[manifest.Object.SourcePart]
	if !ok {
		t.Fatalf("source part %s not found", manifest.Object.SourcePart)
	}
	rawObject, err := extractRawObjectXMLForRecord(sourceData, manifest.Object)
	if err != nil {
		t.Fatalf("extract table object XML: %v", err)
	}
	root, err := parseXMLNode([]byte(rawObject))
	if err != nil {
		t.Fatalf("parse table object XML: %v", err)
	}
	tableNode := firstDescendant(root, "tbl")
	if tableNode == nil {
		t.Fatal("graphicFrame source object has no a:tbl payload")
	}
	theme := themeColorsForPart(pkg, manifest.Object.SourcePart, packageThemeColors(pkg))
	fonts := themeFontsForPart(pkg, manifest.Object.SourcePart, packageThemeFonts(pkg))
	fillStyles := themeFillStylesForPart(pkg, manifest.Object.SourcePart)
	lineStyles := themeLineStylesForPart(pkg, manifest.Object.SourcePart)
	effectStyles := themeEffectStylesForPart(pkg, manifest.Object.SourcePart)
	tableStyles := packageTableStyles(pkg, theme, fonts, fillStyles, lineStyles, effectStyles)
	table := parseTableModel(tableNode, theme)
	artifact := microFixtureTableStyleColorProfile{
		ManifestPath:              manifestPath,
		DeckInput:                 manifest.DeckInput,
		SlideNumber:               manifest.SlideNumber,
		CNvPrID:                   manifest.Object.CNvPrID,
		CNvPrName:                 manifest.Object.CNvPrName,
		SchemaAnchors:             microFixtureManifestSchemaAnchors(manifest),
		SourceXMLPart:             manifest.Object.SourcePart,
		SourceXMLPath:             manifest.Object.XMLPath,
		TableStyleID:              table.StyleID,
		FirstRow:                  table.FirstRow,
		BandRow:                   table.BandRow,
		TopGotColors:              append([]microFixtureColorCount{}, manifest.TargetScope.TopGotColors...),
		TopReferenceColors:        append([]microFixtureColorCount{}, manifest.TargetScope.TopReferenceColors...),
		TopDifferentGotColors:     append([]microFixtureColorCount{}, manifest.TargetScope.TopDifferentGotColors...),
		TopDifferentReference:     append([]microFixtureColorCount{}, manifest.TargetScope.TopDifferentReferenceColors...),
		TargetDifferentPixels:     manifest.TargetScope.DifferentPixels,
		ReferenceRGBDeltaSum8:     manifest.TargetScope.ReferenceRGBDeltaSum8,
		ReferenceRGBAbsoluteDelta: manifest.TargetScope.ReferenceRGBAbsoluteDeltaSum8,
	}
	if style, ok := tableStyleForTable(table, tableStyles); ok {
		artifact.TableStyleName = style.Name
	}
	for _, sample := range []struct {
		label       string
		rowIndex    int
		columnIndex int
	}{
		{label: "first-row", rowIndex: 0, columnIndex: 0},
		{label: "band1-row", rowIndex: 1, columnIndex: 1},
		{label: "band2-row", rowIndex: 2, columnIndex: 1},
	} {
		if sample.rowIndex >= len(table.Rows) || sample.columnIndex >= len(table.Rows[sample.rowIndex].Cells) {
			continue
		}
		cell := table.Rows[sample.rowIndex].Cells[sample.columnIndex]
		style := resolvedTableCellStyle(table, tableStyles, sample.rowIndex, sample.columnIndex)
		cellFill, hasCellFill := tableCellFill(style, cell)
		textColor, hasTextColor := tableCellTextColor(style)
		border := tableEdgeBorder(style.Borders, tableEdgeBottom, sample.rowIndex, sample.columnIndex, len(table.Rows), tableColumnCount(table))
		artifact.Samples = append(artifact.Samples, microFixtureTableStyleColorSample{
			Label:             sample.label,
			RowIndex:          sample.rowIndex,
			ColumnIndex:       sample.columnIndex,
			RegionNames:       tableStyleRegionNamesForCell(table, sample.rowIndex, sample.columnIndex),
			HasFill:           hasCellFill,
			FillColor:         formatObjectColor(cellFill),
			DisplayP3Fill:     formatObjectColor(displayP3RGBA(cellFill)),
			HasTextColor:      hasTextColor,
			TextColor:         formatObjectColor(textColor),
			DisplayP3Text:     formatObjectColor(displayP3RGBA(textColor)),
			BottomBorderLine:  border.HasLine && !border.NoLine,
			BottomBorder:      formatObjectColor(border.Color),
			DisplayP3Border:   formatObjectColor(displayP3RGBA(border.Color)),
			BottomBorderWidth: border.Width,
		})
	}
	t.Logf("table style color profile: style=%s samples=%d diff=%d", artifact.TableStyleID, len(artifact.Samples), artifact.TargetDifferentPixels)
	if outputPath := os.Getenv("PUPPT_TABLE_STYLE_COLOR_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write table style color profile: %v", err)
		}
	}
}

func TestMicroFixtureShapeVectorBackendProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_MANIFEST to profile one shape micro-fixture with the draw2d vector backend candidate")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "sp" && manifest.Object.Kind != "cxnSp" {
		t.Fatalf("shape vector backend profile requires a shape object, got kind=%q", manifest.Object.Kind)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape vector backend profile requires output pixel bounds")
	}

	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			t.Fatalf("write shape vector backend visible crop: %v", err)
		}
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	} else if err := writeCroppedPNG(renderPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape vector backend crop: %v", err)
	}
	got, err := decodePNGFile(gotTargetPath)
	if err != nil {
		t.Fatalf("decode shape vector backend got crop: %v", err)
	}
	reference, err := decodePNGFile(referenceTargetPath)
	if err != nil {
		t.Fatalf("decode shape vector backend reference crop: %v", err)
	}
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open shape vector backend fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("shape vector backend fixture has no slide parts")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	canvas := decodePNG(t, renderPath).Bounds()
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	element, ok := findMicroFixtureSlideElement(elements, manifest.Object)
	if !ok {
		t.Fatalf("find target element %s %q in vector backend fixture", manifest.Object.CNvPrID, manifest.Object.CNvPrName)
	}
	artifact := microFixtureShapeVectorBackendProfileArtifact{
		ManifestPath:   manifestPath,
		FixturePath:    fixturePath,
		TargetCompared: targetCompared,
		Basis:          "diagnostic only: draw2d renders parsed DrawingML shape fill/stroke, then Puppt current text renderer draws parsed text",
		Baseline:       compareImages(got, reference),
	}
	if outputPath := os.Getenv("PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT"); outputPath != "" {
		candidates, err := shapeVectorBackendCandidates(got, reference, element, size, canvas, *manifest.Object.OutputPixelBounds, manifest.OccludedBy, filepath.Dir(resolveTestArtifactPath(outputPath)))
		if err != nil {
			t.Fatalf("shape vector backend candidates: %v", err)
		}
		artifact.Candidates = candidates
	} else {
		candidates, err := shapeVectorBackendCandidates(got, reference, element, size, canvas, *manifest.Object.OutputPixelBounds, manifest.OccludedBy, "")
		if err != nil {
			t.Fatalf("shape vector backend candidates: %v", err)
		}
		artifact.Candidates = candidates
	}
	t.Logf("shape vector backend profile: baseline=%d best=%+v", artifact.Baseline.DifferentPixels, firstShapeVectorBackendCandidate(artifact.Candidates))
	if outputPath := os.Getenv("PUPPT_SHAPE_VECTOR_BACKEND_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shape vector backend profile: %v", err)
		}
	}
}

func TestMicroFixtureShapeTextShapingProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHAPE_TEXT_SHAPING_PROFILE_MANIFEST to profile one shape micro-fixture with the go-text shaping backend candidate")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "sp" && manifest.Object.Kind != "cxnSp" {
		t.Fatalf("shape text shaping profile requires a shape object, got kind=%q", manifest.Object.Kind)
	}
	if manifest.Object.OutputPixelBounds == nil {
		t.Fatal("shape text shaping profile requires output pixel bounds")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open text shaping fixture: %v", err)
	}
	if len(pkg.SlideParts) == 0 {
		t.Fatal("text shaping fixture has no slide parts")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	canvas := decodePNG(t, renderPath).Bounds()
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	element, ok := findMicroFixtureSlideElement(elements, manifest.Object)
	if !ok {
		t.Fatalf("find target element %s %q in text shaping fixture", manifest.Object.CNvPrID, manifest.Object.CNvPrName)
	}
	artifact, err := shapeTextShapingProfile(manifestPath, fixturePath, element, size, canvas, *manifest.Object.OutputPixelBounds)
	if err != nil {
		t.Fatalf("shape text shaping profile: %v", err)
	}
	t.Logf("shape text shaping profile: lines=%d segments=%d max_delta=%d", len(artifact.Lines), artifact.SegmentCount, artifact.MaxAdvanceDeltaPixels)
	if outputPath := os.Getenv("PUPPT_SHAPE_TEXT_SHAPING_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write text shaping profile: %v", err)
		}
	}
}

func TestMicroFixtureTargetOwnershipSummary(t *testing.T) {
	root := os.Getenv("PUPPT_MICRO_FIXTURE_ROOT")
	if root == "" {
		t.Skip("set PUPPT_MICRO_FIXTURE_ROOT to summarize extracted object micro-fixture ownership")
	}
	root = resolveTestArtifactPath(root)
	summary, err := summarizeMicroFixtureTargetOwnership(root)
	if err != nil {
		t.Fatalf("summarize micro-fixture ownership: %v", err)
	}
	if outputPath := os.Getenv("PUPPT_MICRO_FIXTURE_OWNERSHIP_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, summary); err != nil {
			t.Fatalf("write micro-fixture ownership summary: %v", err)
		}
	}
	t.Logf("micro-fixture ownership: total=%d scoped=%d clean_failures=%d contaminated_failures=%d partial_underpaint_failures=%d", summary.TotalManifests, summary.ManifestsWithTargetScope, len(summary.CleanFailures), len(summary.ContaminatedFailures), len(summary.PartialUnderpaintFailures))
}

func TestRendererProductionFailureScoreboard(t *testing.T) {
	root := os.Getenv("PUPPT_RENDERER_SCOREBOARD_ROOT")
	if root == "" {
		t.Skip("set PUPPT_RENDERER_SCOREBOARD_ROOT to summarize renderer primitive failure groups")
	}
	root = resolveTestArtifactPath(root)
	scoreboard, err := buildRendererProductionFailureScoreboard(root)
	if err != nil {
		t.Fatalf("build renderer production failure scoreboard: %v", err)
	}
	t.Logf("renderer production scoreboard: slides=%d total_slide_diff=%d object_groups=%d clean_failures=%d", scoreboard.SlideCount, scoreboard.TotalSlideDifferentPixels, len(scoreboard.ObjectOverlapByPrimitive), scoreboard.CleanFixtureFailureCount)
	if outputPath := os.Getenv("PUPPT_RENDERER_SCOREBOARD_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, scoreboard); err != nil {
			t.Fatalf("write renderer production failure scoreboard: %v", err)
		}
	}
}

func TestMicroFixtureShadowPhaseSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHADOW_PHASE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHADOW_PHASE_SEARCH_MANIFEST to search one extracted shadowed object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceBeforeCropPath == "" {
		t.Fatal("micro-fixture manifest has no source-before crop for shadow alpha search")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{ArtifactDir: filepath.Join(outputDir, "objects")}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.ObjectArtifactPath == "" {
		object.ObjectArtifactPath = manifest.Object.ObjectArtifactPath
	}
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		t.Fatalf("current object is not a shadowed custom-path fixture target: %+v", object)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	fixturePkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open micro-fixture for shadow phase search: %v", err)
	}
	size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
	artifact, err := searchMicroFixtureShadowPhase(referencePath, resolveTestArtifactPath(manifest.SourceBeforeCropPath), object, size, decodePNG(t, outputPath).Bounds())
	if err != nil {
		t.Fatalf("search shadow phase: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate shadow alpha vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best shadow phase candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_SHADOW_PHASE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shadow phase search output: %v", err)
		}
	}
}

func TestMicroFixtureShadowCompositeSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHADOW_COMPOSITE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHADOW_COMPOSITE_SEARCH_MANIFEST to search one extracted shadowed object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceBeforeCropPath == "" {
		t.Fatal("micro-fixture manifest has no source-before crop for shadow composite search")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.ObjectArtifactPath == "" {
		object.ObjectArtifactPath = manifest.Object.ObjectArtifactPath
	}
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		t.Fatalf("current object is not a shadowed custom-path fixture target: %+v", object)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	fixturePkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open micro-fixture for shadow composite search: %v", err)
	}
	size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
	artifact, err := searchMicroFixtureShadowComposite(referencePath, resolveTestArtifactPath(manifest.SourceBeforeCropPath), object, manifest.OccludedBy, size, decodePNG(t, outputPath).Bounds(), outputDir)
	if err != nil {
		t.Fatalf("search shadow composite: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate composite crop vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best shadow composite candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_SHADOW_COMPOSITE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shadow composite search output: %v", err)
		}
	}
}

func TestMicroFixtureShadowParameterSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHADOW_PARAMETER_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHADOW_PARAMETER_SEARCH_MANIFEST to search one extracted shadowed object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceBeforeCropPath == "" {
		t.Fatal("micro-fixture manifest has no source-before crop for shadow parameter search")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 || !object.ResolvedStyle.Shadow {
		t.Fatalf("current object is not a shadowed custom-path fixture target: %+v", object)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	fixturePkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open micro-fixture for shadow parameter search: %v", err)
	}
	size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
	artifact, err := searchMicroFixtureShadowParameters(referencePath, resolveTestArtifactPath(manifest.SourceBeforeCropPath), object, manifest.OccludedBy, size, decodePNG(t, outputPath).Bounds(), outputDir)
	if err != nil {
		t.Fatalf("search shadow parameters: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate shadow parameter composite crop vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best shadow parameter candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_SHADOW_PARAMETER_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shadow parameter search output: %v", err)
		}
	}
}

func TestMicroFixtureShadowKernelSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHADOW_KERNEL_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHADOW_KERNEL_SEARCH_MANIFEST to search one extracted shadowed object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceBeforeCropPath == "" {
		t.Fatal("micro-fixture manifest has no source-before crop for shadow kernel search")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 || !object.ResolvedStyle.Shadow {
		t.Fatalf("current object is not a shadowed custom-path fixture target: %+v", object)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	fixturePkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open micro-fixture for shadow kernel search: %v", err)
	}
	size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
	artifact, err := searchMicroFixtureShadowKernels(referencePath, resolveTestArtifactPath(manifest.SourceBeforeCropPath), object, manifest.OccludedBy, size, decodePNG(t, outputPath).Bounds(), outputDir)
	if err != nil {
		t.Fatalf("search shadow kernels: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate shadow kernel composite crop vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best shadow kernel candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_SHADOW_KERNEL_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shadow kernel search output: %v", err)
		}
	}
}

func TestMicroFixtureShadowGeometrySearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_SHADOW_GEOMETRY_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_SHADOW_GEOMETRY_SEARCH_MANIFEST to search one extracted shadowed object micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.SourceBeforeCropPath == "" {
		t.Fatal("micro-fixture manifest has no source-before crop for shadow geometry search")
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 || !object.ResolvedStyle.Shadow {
		t.Fatalf("current object is not a shadowed custom-path fixture target: %+v", object)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	fixturePkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		t.Fatalf("open micro-fixture for shadow geometry search: %v", err)
	}
	size := parseSlideSize(fixturePkg.Parts[fixturePkg.PresentationPath])
	artifact, err := searchMicroFixtureShadowGeometry(referencePath, resolveTestArtifactPath(manifest.SourceBeforeCropPath), object, manifest.OccludedBy, size, decodePNG(t, outputPath).Bounds(), outputDir)
	if err != nil {
		t.Fatalf("search shadow geometry: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate fractional shadow geometry composite crop vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best shadow geometry candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_SHADOW_GEOMETRY_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write shadow geometry search output: %v", err)
		}
	}
}

func TestMicroFixtureRectEdgeBlendSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_RECT_EDGE_BLEND_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_RECT_EDGE_BLEND_SEARCH_MANIFEST to search one extracted rectangle micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	outputDir := t.TempDir()
	debug := &ObjectDebugOptions{}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: outputPath, ObjectDebug: debug}); err != nil {
		t.Fatalf("render micro-fixture %s: %v", fixturePath, err)
	}
	record, ok := findMicroFixtureDebugRecord(debug.Records, manifest.Object)
	if !ok {
		t.Fatalf("current object not found in debug records: %+v", debug.Records)
	}
	object := objectFailureRecordFromPaintedObject(manifest.DeckInput, manifest.SlideNumber, record)
	if object.OutputPixelBounds == nil || object.ResolvedStyle.Geometry != "rect" {
		t.Fatalf("current object is not a rectangle fixture target: %+v", object)
	}
	beforePath := filepath.Join(outputDir, "before.png")
	if _, err := Render(context.Background(), fixturePath, Options{
		SlideNumber: 1,
		OutputPath:  beforePath,
		ObjectDebug: &ObjectDebugOptions{
			Mode:         ObjectDebugRenderBefore,
			TargetZOrder: object.ZOrder,
		},
	}); err != nil {
		t.Fatalf("render micro-fixture before object %d: %v", object.ZOrder, err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "reference-crop.png"
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "reference-visible-crop.png"
	}
	nonUnderpaintReferencePath := ""
	if manifest.NonUnderpaintReferenceCropPath != "" {
		nonUnderpaintReferencePath = resolveTestArtifactPath(manifest.NonUnderpaintReferenceCropPath)
	}
	artifact, err := searchMicroFixtureRectEdgeBlend(referencePath, nonUnderpaintReferencePath, beforePath, object, manifest.OccludedBy, manifest.UnderpaintedBy, outputDir)
	if err != nil {
		t.Fatalf("search rectangle edge blend: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = "candidate rectangle over before-object crop vs " + targetLabel
	if len(artifact.Candidates) > 0 {
		t.Logf("best rectangle edge blend candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_RECT_EDGE_BLEND_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write rectangle edge blend search output: %v", err)
		}
	}
}

func TestMicroFixturePictureResampleSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_RESAMPLE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_RESAMPLE_SEARCH_MANIFEST to search one extracted picture micro-fixture")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureResample(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture resample: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture resample candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_RESAMPLE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture resample search output: %v", err)
		}
	}
}

func TestMicroFixturePictureEdgeSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_EDGE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_EDGE_SEARCH_MANIFEST to search one extracted picture micro-fixture edge coverage")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture edge crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture edge visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureEdges(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture edges: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture edge candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_EDGE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture edge search output: %v", err)
		}
	}
}

func TestMicroFixturePictureGammaSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_GAMMA_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_GAMMA_SEARCH_MANIFEST to search one extracted picture micro-fixture transfer function")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture transfer crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture transfer visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureGamma(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture transfer function: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture transfer candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_GAMMA_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture transfer search output: %v", err)
		}
	}
}

func TestMicroFixturePictureKernelSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_KERNEL_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_KERNEL_SEARCH_MANIFEST to search one extracted picture micro-fixture scaler kernel")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture kernel crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture kernel visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureKernels(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture scaler kernels: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture kernel candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_KERNEL_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture kernel search output: %v", err)
		}
	}
}

func TestMicroFixturePictureAreaSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_AREA_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_AREA_SEARCH_MANIFEST to search one extracted picture micro-fixture area resampling")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture area crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture area visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureArea(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture area resampling: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture area candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_AREA_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture area search output: %v", err)
		}
	}
}

func TestMicroFixturePicturePhaseSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_PHASE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_PHASE_SEARCH_MANIFEST to search one extracted picture micro-fixture sampling phase")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture phase crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture phase visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePicturePhase(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture sampling phase: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture phase candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_PHASE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture phase search output: %v", err)
		}
	}
}

func TestMicroFixturePictureFractionalBoundsSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_MANIFEST to search one extracted picture micro-fixture fractional geometry")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture fractional-bounds crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture fractional-bounds visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureFractionalBounds(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture fractional bounds: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture fractional-bounds candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_FRACTIONAL_BOUNDS_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture fractional-bounds search output: %v", err)
		}
	}
}

func TestMicroFixturePictureSourceModelSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_SOURCE_MODEL_SEARCH_MANIFEST to search one extracted picture micro-fixture source image model")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	sources, err := microFixturePictureSourceModelVariants(fixturePath)
	if err != nil {
		t.Fatalf("decode fixture picture source model variants: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture source model crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture source model visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureSourceModels(referencePath, sources, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture source models: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture source model candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_SOURCE_MODEL_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture source model search output: %v", err)
		}
	}
}

func TestMicroFixturePicturePNGMetadataProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_PNG_METADATA_PROFILE_MANIFEST to profile one extracted picture micro-fixture PNG metadata")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" {
		t.Fatalf("micro-fixture is not a picture: %+v", manifest.Object)
	}
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	artifact, err := microFixturePicturePNGMetadataProfile(fixturePath)
	if err != nil {
		t.Fatalf("profile picture PNG metadata: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = fixturePath
	t.Logf("picture PNG metadata: part=%s bytes=%d ihdr=%dx%d bit_depth=%d color_type=%d chunks=%d gamma=%v icc=%t phys=%v", artifact.MediaPart, artifact.ByteSize, artifact.Width, artifact.Height, artifact.BitDepth, artifact.ColorType, len(artifact.Chunks), artifact.Gamma, artifact.HasICCP, artifact.PhysicalPixelsPerUnit)
	if outputPath := os.Getenv("PUPPT_PICTURE_PNG_METADATA_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture PNG metadata profile output: %v", err)
		}
	}
}

func TestMicroFixturePicturePipelineProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_PIPELINE_PROFILE_MANIFEST to split one extracted picture micro-fixture render pipeline")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	artifact, err := microFixturePicturePipelineProfile(manifestPath, manifest)
	if err != nil {
		t.Fatalf("profile picture pipeline: %v", err)
	}
	t.Logf("picture pipeline profile: target=%s got_delta=%d reference_delta=%d sampling=%s", artifact.TargetCompared, artifact.Output.DiffAgainstGot.DifferentPixels, artifact.Output.DiffAgainstReference.DifferentPixels, artifact.Sampling.Scaler)
	if outputPath := os.Getenv("PUPPT_PICTURE_PIPELINE_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture pipeline profile output: %v", err)
		}
	}
}

func TestMicroFixturePictureContourCoverageSearch(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_MANIFEST to search one extracted picture micro-fixture contour coverage model")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	source, err := microFixturePictureSourceImage(resolveTestArtifactPath(manifest.FixturePath))
	if err != nil {
		t.Fatalf("decode picture source: %v", err)
	}
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "candidate picture contour coverage crop vs reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.ReferenceVisibleCropPath != "" {
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "candidate picture contour coverage visible crop vs reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	artifact, err := searchMicroFixturePictureContourCoverage(referencePath, source, manifest.Object, occlusions, t.TempDir())
	if err != nil {
		t.Fatalf("search picture contour coverage: %v", err)
	}
	artifact.ManifestPath = manifestPath
	artifact.FixturePath = resolveTestArtifactPath(manifest.FixturePath)
	artifact.TargetCompared = targetCompared
	if len(artifact.Candidates) > 0 {
		t.Logf("best picture contour coverage candidate: %+v", artifact.Candidates[0])
	}
	if outputPath := os.Getenv("PUPPT_PICTURE_CONTOUR_COVERAGE_SEARCH_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, artifact); err != nil {
			t.Fatalf("write picture contour coverage search output: %v", err)
		}
	}
}

func TestMicroFixturePictureResidualProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_RESIDUAL_PROFILE_MANIFEST to profile one extracted picture micro-fixture residual")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	source, err := microFixturePictureSourceImage(resolveTestArtifactPath(manifest.FixturePath))
	if err != nil {
		t.Fatalf("decode picture source: %v", err)
	}
	gotPath := resolveTestArtifactPath(manifest.GotCropPath)
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	if manifest.GotVisibleCropPath != "" && manifest.ReferenceVisibleCropPath != "" {
		gotPath = resolveTestArtifactPath(manifest.GotVisibleCropPath)
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
	}
	profile, err := microFixturePictureResidualProfile(gotPath, referencePath, source)
	if err != nil {
		t.Fatalf("profile picture residual: %v", err)
	}
	profile.ManifestPath = manifestPath
	profile.FixturePath = resolveTestArtifactPath(manifest.FixturePath)
	profile.TargetCompared = filepath.Base(gotPath) + " vs " + filepath.Base(referencePath)
	t.Logf("picture residual profile: differing=%d grayscale=%d edge=%d pure_bw=%d", profile.DifferentPixels, profile.GrayscaleDifferentPixels, profile.EdgeCoverageDifferentPixels, profile.PureBlackWhiteDifferentPixels)
	if outputPath := os.Getenv("PUPPT_PICTURE_RESIDUAL_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write picture residual profile: %v", err)
		}
	}
}

func TestMicroFixturePictureSourceCorrespondenceProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_MANIFEST to profile one extracted picture micro-fixture residual against source coordinates")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	source, err := microFixturePictureSourceImage(resolveTestArtifactPath(manifest.FixturePath))
	if err != nil {
		t.Fatalf("decode picture source: %v", err)
	}
	gotPath := resolveTestArtifactPath(manifest.GotCropPath)
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	if manifest.GotVisibleCropPath != "" && manifest.ReferenceVisibleCropPath != "" {
		gotPath = resolveTestArtifactPath(manifest.GotVisibleCropPath)
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
	}
	profile, err := microFixturePictureSourceCorrespondenceProfile(gotPath, referencePath, source, manifest.Object)
	if err != nil {
		t.Fatalf("profile picture source correspondence: %v", err)
	}
	profile.ManifestPath = manifestPath
	profile.FixturePath = resolveTestArtifactPath(manifest.FixturePath)
	profile.TargetCompared = filepath.Base(gotPath) + " vs " + filepath.Base(referencePath)
	t.Logf("picture source correspondence profile: differing=%d source_bounds=%+v mixed_3x3=%d nearest_antialias=%d", profile.DifferentPixels, profile.SourceCoordinateBounds, profile.Mixed3x3SourceNeighborhoodPixels, profile.NearestSourceAntialiasPixels)
	if outputPath := os.Getenv("PUPPT_PICTURE_SOURCE_CORRESPONDENCE_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write picture source correspondence profile: %v", err)
		}
	}
}

func TestMicroFixturePictureEdgeGeometryProfile(t *testing.T) {
	manifestPath := os.Getenv("PUPPT_PICTURE_EDGE_GEOMETRY_PROFILE_MANIFEST")
	if manifestPath == "" {
		t.Skip("set PUPPT_PICTURE_EDGE_GEOMETRY_PROFILE_MANIFEST to profile one extracted picture micro-fixture edge geometry")
	}
	manifestPath = resolveTestArtifactPath(manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read micro-fixture manifest: %v", err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse micro-fixture manifest: %v", err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("micro-fixture is not a picture with output pixel bounds: %+v", manifest.Object)
	}
	gotPath := resolveTestArtifactPath(manifest.GotCropPath)
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := filepath.Base(gotPath) + " vs " + filepath.Base(referencePath)
	if manifest.GotVisibleCropPath != "" && manifest.ReferenceVisibleCropPath != "" {
		gotPath = resolveTestArtifactPath(manifest.GotVisibleCropPath)
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = filepath.Base(gotPath) + " vs " + filepath.Base(referencePath)
	}
	profile, err := microFixturePictureEdgeGeometryProfile(gotPath, referencePath, manifest.Object)
	if err != nil {
		t.Fatalf("profile picture edge geometry: %v", err)
	}
	profile.ManifestPath = manifestPath
	profile.FixturePath = resolveTestArtifactPath(manifest.FixturePath)
	profile.TargetCompared = targetCompared
	t.Logf("picture edge geometry profile: differing=%d bounds=%+v top_row=%+v top_col=%+v", profile.DifferentPixels, profile.DifferentBounds, firstAxisDeltaCount(profile.TopRows), firstAxisDeltaCount(profile.TopColumns))
	if outputPath := os.Getenv("PUPPT_PICTURE_EDGE_GEOMETRY_PROFILE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, profile); err != nil {
			t.Fatalf("write picture edge geometry profile: %v", err)
		}
	}
}

func findMicroFixtureDebugRecord(records []PaintedObject, object objectFailureRecord) (PaintedObject, bool) {
	for _, record := range records {
		if record.Kind == object.Kind && record.CNvPrID == object.CNvPrID && record.CNvPrName == object.CNvPrName {
			return record, true
		}
	}
	if len(records) == 1 {
		return records[0], true
	}
	return PaintedObject{}, false
}

func resolvedRenderElementsForPart(pkg *pptx.Package, slidePart string, renderPart string) []slideElement {
	theme := packageThemeColors(pkg)
	fonts := packageThemeFonts(pkg)
	themeForPart := func(part string) themeColors {
		return themeColorsForPart(pkg, part, theme)
	}
	fontsForPart := func(part string) themeFonts {
		return themeFontsForPart(pkg, part, fonts)
	}
	renderParts := inheritedRenderParts(pkg, slidePart)
	placeholderSources := inheritedPlaceholderSourcesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	textStyles := inheritedTextStylesWithThemeResolver(pkg, renderParts, slidePart, themeForPart)
	partTheme := themeForPart(renderPart)
	partFonts := fontsForPart(renderPart)
	partLineStyles := themeLineStylesForPart(pkg, renderPart)
	effectStyles := themeEffectStylesForPart(pkg, renderPart)
	fillStyles := themeFillStylesForPart(pkg, renderPart)
	tableStyles := packageTableStyles(pkg, partTheme, partFonts, fillStyles, partLineStyles, effectStyles)
	_ = tableStyles
	elements := collectSlideElementsWithThemeEffectsAndFills(pkg.Parts[renderPart], partTheme, effectStyles, fillStyles, partLineStyles)
	if renderPart == slidePart {
		elements = resolveSlidePlaceholders(elements, placeholderSources)
		elements = applyInheritedTextStyles(elements, textStyles)
	}
	elements = applyInheritedTableTextStyles(elements, textStyles)
	elements = applyThemeFontFamilies(elements, partFonts)
	elements = resolveTextFields(elements, 1)
	return elements
}

func findMicroFixtureSlideElement(elements []slideElement, object objectFailureRecord) (slideElement, bool) {
	for _, element := range elements {
		if element.Kind == object.Kind && element.ID == object.CNvPrID && element.Name == object.CNvPrName {
			return element, true
		}
	}
	if len(elements) == 1 {
		return elements[0], true
	}
	return slideElement{}, false
}

func elementPixelTarget(element slideElement, size slideSize, canvas image.Rectangle) image.Rectangle {
	return image.Rect(
		scaleEMU(element.OffX, size.CX, canvas.Dx()),
		scaleEMU(element.OffY, size.CY, canvas.Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, canvas.Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, canvas.Dy()),
	).Intersect(canvas)
}

func objectFailureRecordFromPaintedObject(deckInput string, slideNumber int, record PaintedObject) objectFailureRecord {
	return objectFailureRecord{
		DeckInput:         deckInput,
		SlideNumber:       slideNumber,
		SlidePart:         record.SlidePart,
		SourcePart:        record.SourcePart,
		XMLPath:           record.XMLPath,
		CNvPrID:           record.CNvPrID,
		CNvPrName:         record.CNvPrName,
		Kind:              record.Kind,
		ZOrder:            record.ZOrder,
		Bounds:            record.Bounds,
		PixelBounds:       record.PixelBounds,
		FractionalBounds:  record.FractionalBounds,
		OutputPixelBounds: record.OutputPixelBounds,
		ResolvedStyle:     record.ResolvedStyle,
	}
}

func microFixtureDebugDir() string {
	dir := os.Getenv("PUPPT_MICRO_FIXTURE_DEBUG_DIR")
	if dir == "" || filepath.IsAbs(dir) {
		return dir
	}
	return filepath.Join("..", "..", dir)
}

func TestRealWorldGoldenComparison(t *testing.T) {
	if os.Getenv("PUPPT_RUN_REALWORLD_RENDER_TESTS") != "1" {
		t.Skip("set PUPPT_RUN_REALWORLD_RENDER_TESTS=1 to run the 61-slide renderer golden comparison")
	}

	referenceRoot := realWorldReferenceRoot()
	manifestPath := filepath.Join(referenceRoot, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest referenceManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	total := 0
	failures := 0
	totalDifferentPixels := int64(0)
	worstDifferentPixels := int64(0)
	worstLabel := ""
	unsupportedCounts := map[string]int{}
	var slideDiffs []realWorldSlideDiff
	for _, deck := range manifest.Decks {
		if deck.Input == "" {
			t.Fatalf("manifest deck missing input: %+v", deck)
		}
		for index, slide := range deck.Slides {
			total++
			outputPath := filepath.Join(t.TempDir(), filepath.Base(deck.Input)+"-slide.png")
			result, err := Render(context.Background(), filepath.Join("..", "..", deck.Input), Options{
				SlideNumber: index + 1,
				OutputPath:  outputPath,
				DPI:         referenceDPIForSlide(slide.Width),
			})
			if err != nil {
				t.Fatalf("render %s slide %d: %v", deck.Input, index+1, err)
			}
			referencePath := filepath.Join(referenceRoot, slide.File)
			diff, err := comparePNG(outputPath, referencePath)
			if err != nil {
				t.Fatalf("compare %s slide %d: %v", deck.Input, index+1, err)
			}
			if diff.Width != slide.Width || diff.Height != slide.Height {
				t.Fatalf("manifest dimensions disagree for %s: manifest=%dx%d reference=%dx%d", slide.File, slide.Width, slide.Height, diff.Width, diff.Height)
			}
			if diff.GotWidth != slide.Width || diff.GotHeight != slide.Height {
				t.Fatalf("render dimensions disagree for %s slide %d: got=%dx%d reference=%dx%d", deck.Input, index+1, diff.GotWidth, diff.GotHeight, slide.Width, slide.Height)
			}
			if diff.DifferentPixels != 0 {
				if artifactRoot := realWorldArtifactRoot(); artifactRoot != "" {
					debug := &ObjectDebugOptions{ArtifactDir: filepath.Join(realWorldSlideArtifactDir(artifactRoot, deck.Input, index+1), "objects")}
					result, err = Render(context.Background(), filepath.Join("..", "..", deck.Input), Options{
						SlideNumber: index + 1,
						OutputPath:  outputPath,
						DPI:         referenceDPIForSlide(slide.Width),
						ObjectDebug: debug,
					})
					if err != nil {
						t.Fatalf("debug render %s slide %d: %v", deck.Input, index+1, err)
					}
					writeRealWorldDiffArtifacts(t, outputPath, referencePath, deck.Input, index+1, result, diff, debug)
				} else {
					writeRealWorldDiffArtifacts(t, outputPath, referencePath, deck.Input, index+1, result, diff, nil)
				}
				failures++
				totalDifferentPixels += int64(diff.DifferentPixels)
				for _, item := range result.Unsupported {
					unsupportedCounts[item.Message]++
				}
				if int64(diff.DifferentPixels) > worstDifferentPixels {
					worstDifferentPixels = int64(diff.DifferentPixels)
					worstLabel = fmt.Sprintf("%s slide %03d", deck.Input, index+1)
				}
				slideDiffs = append(slideDiffs, realWorldSlideDiff{
					Label:           fmt.Sprintf("%s slide %03d", deck.Input, index+1),
					DifferentPixels: diff.DifferentPixels,
					Unsupported:     len(result.Unsupported),
					Status:          result.Status,
				})
				if failures <= 10 {
					t.Logf("%s slide %03d: %d differing pixel(s), unsupported=%d, status=%s", deck.Input, index+1, diff.DifferentPixels, len(result.Unsupported), result.Status)
				}
			}
		}
	}
	if total != 61 {
		t.Fatalf("expected 61 real-world reference slides, got %d", total)
	}
	if failures != 0 {
		t.Fatalf("%d of %d real-world slides differ from %s references; total differing pixels=%d; worst=%s with %d differing pixels; worst slides: %s; top unsupported rendering gaps: %s", failures, total, referenceRoot, totalDifferentPixels, worstLabel, worstDifferentPixels, formatSlideDiffSummary(slideDiffs, 8), formatUnsupportedSummary(unsupportedCounts, 8))
	}
}

type realWorldSlideDiff struct {
	Label           string
	DifferentPixels int
	Unsupported     int
	Status          string
}

func formatSlideDiffSummary(items []realWorldSlideDiff, limit int) string {
	if len(items) == 0 || limit <= 0 {
		return "none"
	}
	items = append([]realWorldSlideDiff(nil), items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].DifferentPixels != items[j].DifferentPixels {
			return items[i].DifferentPixels > items[j].DifferentPixels
		}
		return items[i].Label < items[j].Label
	})
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%s=%dpx/%d unsupported/%s", item.Label, item.DifferentPixels, item.Unsupported, item.Status))
	}
	return builder.String()
}

func formatUnsupportedSummary(counts map[string]int, limit int) string {
	if len(counts) == 0 || limit <= 0 {
		return "none"
	}
	items := make([]unsupportedSummaryItem, 0, len(counts))
	for message, count := range counts {
		if count <= 0 {
			continue
		}
		items = append(items, unsupportedSummaryItem{Message: message, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Message < items[j].Message
	})
	if len(items) == 0 {
		return "none"
	}
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%dx %s", item.Count, item.Message))
	}
	return builder.String()
}

type unsupportedSummaryItem struct {
	Message string
	Count   int
}

func realWorldReferenceRoot() string {
	if root := os.Getenv("PUPPT_REALWORLD_REFERENCE_ROOT"); root != "" {
		return root
	}
	return filepath.Join("..", "..", "testdata", "realworld-ppts", "reference-renders", "manual-using-apple-note")
}

func realWorldArtifactRoot() string {
	root := os.Getenv("PUPPT_REALWORLD_ARTIFACT_DIR")
	if root == "" {
		return ""
	}
	if filepath.IsAbs(root) {
		return root
	}
	return filepath.Join("..", "..", root)
}

func TestRealWorldArtifactRootResolvesRelativeToRepoRoot(t *testing.T) {
	t.Setenv("PUPPT_REALWORLD_ARTIFACT_DIR", "testdata/realworld-ppts/render-artifacts/example")
	got := filepath.Clean(realWorldArtifactRoot())
	want := filepath.Clean("../../testdata/realworld-ppts/render-artifacts/example")
	if got != want {
		t.Fatalf("unexpected relative artifact root: got=%q want=%q", got, want)
	}
}

func TestMicroFixtureDebugDirResolvesRelativeToRepoRoot(t *testing.T) {
	t.Setenv("PUPPT_MICRO_FIXTURE_DEBUG_DIR", "testdata/realworld-ppts/render-artifacts/example")
	got := filepath.Clean(microFixtureDebugDir())
	want := filepath.Clean("../../testdata/realworld-ppts/render-artifacts/example")
	if got != want {
		t.Fatalf("unexpected relative micro-fixture debug dir: got=%q want=%q", got, want)
	}
}

func TestWriteRealWorldDiffArtifactsWritesMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PUPPT_REALWORLD_ARTIFACT_DIR", dir)
	gotPath := filepath.Join(dir, "got-source.png")
	referencePath := filepath.Join(dir, "reference-source.png")
	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	got.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	got.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}
	objectPath := filepath.Join(dir, "object.png")
	beforePath := filepath.Join(dir, "before.png")
	throughPath := filepath.Join(dir, "through.png")
	if err := writePNG(objectPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(beforePath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(throughPath, got); err != nil {
		t.Fatal(err)
	}

	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       commandName,
		Status:        "partial",
		Unsupported: []model.SkipItem{{
			Code:    partialUnsupportedCode,
			Message: "shape text was rendered with simplified layout",
			Part:    "ppt/slides/slide1.xml",
		}},
	}
	diff, err := comparePNG(gotPath, referencePath)
	if err != nil {
		t.Fatal(err)
	}
	debug := &ObjectDebugOptions{Records: []PaintedObject{{
		SlidePart:           "ppt/slides/slide1.xml",
		SourcePart:          "ppt/slides/slide1.xml",
		XMLPath:             `/p:sld/p:cSld/p:spTree/p:sp[.//p:cNvPr/@id="2"]`,
		CNvPrID:             "2",
		CNvPrName:           "Rectangle 1",
		Kind:                "sp",
		ZOrder:              1,
		PixelBounds:         ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 1, MaxY: 0},
		OutputPixelBounds:   &ObjectPixelBounds{MinX: 1, MinY: 0, MaxX: 1, MaxY: 0},
		BeforeArtifactPath:  beforePath,
		ObjectArtifactPath:  objectPath,
		ThroughArtifactPath: throughPath,
		ResolvedStyle:       ObjectStyleSummary{Geometry: "rect", Fill: "#00FF00/FF"},
		Painted:             true,
	}}}
	writeRealWorldDiffArtifacts(t, gotPath, referencePath, "testdata/realworld-ppts/example.pptx", 1, result, diff, debug)

	slideDir := filepath.Join(dir, "example", "slide-001")
	for _, name := range []string{"got.png", "reference.png", "diff.png", "result.json", "diff.json", "objects.json", "object-attribution.json"} {
		if _, err := os.Stat(filepath.Join(slideDir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(slideDir, "diff.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"different_pixels": 1`) {
		t.Fatalf("diff artifact did not include pixel count: %s", data)
	}
	if !strings.Contains(string(data), `"max_channel_delta_8bit": 255`) || !strings.Contains(string(data), `"different_bounds"`) {
		t.Fatalf("diff artifact did not include channel and bounds diagnostics: %s", data)
	}
	data, err = os.ReadFile(filepath.Join(slideDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "simplified layout") {
		t.Fatalf("result artifact did not include unsupported details: %s", data)
	}
	data, err = os.ReadFile(filepath.Join(slideDir, "object-attribution.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"cnv_pr_name": "Rectangle 1"`) || !strings.Contains(string(data), `"xml_path":`) || !strings.Contains(string(data), `"pixel_bounds":`) || !strings.Contains(string(data), `"before_artifact_path":`) || !strings.Contains(string(data), `"overlap_diff_pixels": 1`) || !strings.Contains(string(data), `"suspected_renderer_gap":`) {
		t.Fatalf("object attribution artifact did not include object failure details: %s", data)
	}
	if !strings.Contains(string(data), `"cumulative_probes":`) || !strings.Contains(string(data), `"largest_cumulative_delta":`) || !strings.Contains(string(data), `"cumulative_diff_delta_pixels": 1`) {
		t.Fatalf("object attribution artifact did not include cumulative z-order probes: %s", data)
	}
}

func TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls(t *testing.T) {
	forbidden := []string{
		"Apple Notes",
		"manual-using-apple-note",
		"reference-renders",
		"LibreOffice",
		"soffice",
		"PowerPoint",
		"Keynote",
		"Google Slides",
		"chromedp",
		"playwright",
		"selenium",
		"exec.Command",
		"WHO-HIV-testing-algorithms-toolkit",
		"EPA-generate-2021-presentation",
		"EPA-metal-coil-NESHAP-2018",
		"EPA-residential-wood-MacCarty",
	}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Errorf("%s contains forbidden renderer implementation string %q", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan renderer implementation: %v", err)
	}
}

func writeRealWorldDiffArtifacts(t *testing.T, gotPath string, referencePath string, deckInput string, slideNumber int, result model.CommandResult, diff imageDiff, debug *ObjectDebugOptions) {
	t.Helper()
	artifactRoot := realWorldArtifactRoot()
	if artifactRoot == "" {
		return
	}
	slideDir := realWorldSlideArtifactDir(artifactRoot, deckInput, slideNumber)
	if err := os.MkdirAll(slideDir, 0o755); err != nil {
		t.Fatalf("create artifact dir %s: %v", slideDir, err)
	}
	for _, item := range []struct {
		source string
		name   string
	}{
		{source: gotPath, name: "got.png"},
		{source: referencePath, name: "reference.png"},
	} {
		if err := copyFile(item.source, filepath.Join(slideDir, item.name)); err != nil {
			t.Fatalf("write %s artifact for %s slide %d: %v", item.name, deckInput, slideNumber, err)
		}
	}
	if err := writeDiffPNG(gotPath, referencePath, filepath.Join(slideDir, "diff.png")); err != nil {
		t.Fatalf("write diff artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "result.json"), result); err != nil {
		t.Fatalf("write result artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "diff.json"), realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write diff metadata artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if debug != nil {
		if err := writeJSONFile(filepath.Join(slideDir, "objects.json"), debug.Records); err != nil {
			t.Fatalf("write object attribution artifact for %s slide %d: %v", deckInput, slideNumber, err)
		}
		objectAttribution, err := buildObjectAttributionArtifact(deckInput, slideNumber, gotPath, referencePath, diff, debug.Records)
		if err != nil {
			t.Fatalf("build object attribution artifact for %s slide %d: %v", deckInput, slideNumber, err)
		}
		if err := writeJSONFile(filepath.Join(slideDir, "object-attribution.json"), objectAttribution); err != nil {
			t.Fatalf("write object attribution artifact for %s slide %d: %v", deckInput, slideNumber, err)
		}
		writeTopPictureMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeLargestCumulativePictureMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeTopShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeLargestCumulativeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeLargestCumulativeConnectorMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeTopTableMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
		writeLargestCumulativeTableMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, objectAttribution)
	}
}

func realWorldSlideArtifactDir(artifactRoot string, deckInput string, slideNumber int) string {
	label := strings.TrimSuffix(filepath.Base(deckInput), filepath.Ext(deckInput))
	return filepath.Join(artifactRoot, label, fmt.Sprintf("slide-%03d", slideNumber))
}

func copyFile(sourcePath string, targetPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

type realWorldDiffArtifact struct {
	DeckInput   string    `json:"deck_input"`
	SlideNumber int       `json:"slide_number"`
	Diff        imageDiff `json:"diff"`
}

type microFixtureManifest struct {
	DeckInput                         string                             `json:"deck_input"`
	SlideNumber                       int                                `json:"slide_number"`
	Object                            objectFailureRecord                `json:"object"`
	SpecFixture                       specFixtureManifest                `json:"spec_fixture"`
	FixturePath                       string                             `json:"fixture_path"`
	SourceObjectXMLPath               string                             `json:"source_object_xml_path,omitempty"`
	SourceObjectSummary               *microFixtureSourceObjectSummary   `json:"source_object_summary,omitempty"`
	FixtureParts                      []microFixturePackagePart          `json:"fixture_parts,omitempty"`
	FixtureObjectsPath                string                             `json:"fixture_objects_path,omitempty"`
	SourceImage                       microFixtureSourceImage            `json:"source_image,omitempty"`
	Sampling                          *microFixtureSampling              `json:"sampling,omitempty"`
	GotCropPath                       string                             `json:"got_crop_path"`
	ReferenceCropPath                 string                             `json:"reference_crop_path"`
	DiffPath                          string                             `json:"diff_path"`
	GotGeometryCropPath               string                             `json:"got_geometry_crop_path,omitempty"`
	ReferenceGeometryCropPath         string                             `json:"reference_geometry_crop_path,omitempty"`
	GeometryDiffPath                  string                             `json:"geometry_diff_path,omitempty"`
	SourceBeforeCropPath              string                             `json:"source_before_crop_path,omitempty"`
	SourceThroughCropPath             string                             `json:"source_through_crop_path,omitempty"`
	SourceThroughDiffPath             string                             `json:"source_through_diff_path,omitempty"`
	SourceThroughVisibleCropPath      string                             `json:"source_through_visible_crop_path,omitempty"`
	SourceThroughVisibleDiffPath      string                             `json:"source_through_visible_diff_path,omitempty"`
	GotVisibleCropPath                string                             `json:"got_visible_crop_path,omitempty"`
	ReferenceVisibleCropPath          string                             `json:"reference_visible_crop_path,omitempty"`
	VisibleDiffPath                   string                             `json:"visible_diff_path,omitempty"`
	UnderpaintChainFixturePath        string                             `json:"underpaint_chain_fixture_path,omitempty"`
	UnderpaintChainGotCropPath        string                             `json:"underpaint_chain_got_crop_path,omitempty"`
	UnderpaintChainDiffPath           string                             `json:"underpaint_chain_diff_path,omitempty"`
	UnderpaintChainGotVisibleCropPath string                             `json:"underpaint_chain_got_visible_crop_path,omitempty"`
	UnderpaintChainVisibleDiffPath    string                             `json:"underpaint_chain_visible_diff_path,omitempty"`
	UnderpaintChainTargetScopePath    string                             `json:"underpaint_chain_target_scope_path,omitempty"`
	UnderpaintChainTargetScope        microFixtureTargetScope            `json:"underpaint_chain_target_scope,omitempty"`
	UnderpaintChainSummary            microFixtureUnderpaintChainSummary `json:"underpaint_chain_summary,omitempty"`
	NonUnderpaintGotCropPath          string                             `json:"non_underpaint_got_crop_path,omitempty"`
	NonUnderpaintReferenceCropPath    string                             `json:"non_underpaint_reference_crop_path,omitempty"`
	NonUnderpaintDiffPath             string                             `json:"non_underpaint_diff_path,omitempty"`
	NonUnderpaintTargetScope          microFixtureTargetScope            `json:"non_underpaint_target_scope,omitempty"`
	ShadowAlphaScopePath              string                             `json:"shadow_alpha_scope_path,omitempty"`
	ShadowAlphaCorrectionHeatmapPath  string                             `json:"shadow_alpha_correction_heatmap_path,omitempty"`
	ShadowAlphaScope                  microFixtureShadowAlphaScope       `json:"shadow_alpha_scope,omitempty"`
	ShadowRenderSummary               *microFixtureShadowRenderSummary   `json:"shadow_render_summary,omitempty"`
	OccludedBy                        []microFixtureOcclusion            `json:"occluded_by,omitempty"`
	UnderpaintedBy                    []microFixtureUnderpaint           `json:"underpainted_by,omitempty"`
	TargetScopePath                   string                             `json:"target_scope_path,omitempty"`
	TargetScope                       microFixtureTargetScope            `json:"target_scope,omitempty"`
	Acceptance                        string                             `json:"acceptance"`
}

type specFixtureManifest struct {
	SchemaAnchors             []string `json:"schema_anchors"`
	SourceXMLPart             string   `json:"source_xml_part"`
	SourceXMLPath             string   `json:"source_xml_path,omitempty"`
	ExpectedSemanticModel     string   `json:"expected_semantic_model"`
	ExpectedRenderPrimitive   string   `json:"expected_render_primitive"`
	ExpectedUnsupportedRecord []string `json:"expected_unsupported_records,omitempty"`
}

type microFixtureOcclusion struct {
	CNvPrID           string            `json:"cnv_pr_id,omitempty"`
	CNvPrName         string            `json:"cnv_pr_name,omitempty"`
	Kind              string            `json:"kind"`
	ZOrder            int               `json:"z_order"`
	Bounds            ObjectPixelBounds `json:"bounds"`
	MaskPaddingPixels int               `json:"mask_padding_pixels,omitempty"`
}

type microFixturePackagePart struct {
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Reason string `json:"reason,omitempty"`
}

type microFixtureSourceObjectSummary struct {
	Kind       string                         `json:"kind,omitempty"`
	CNvPrID    string                         `json:"cnv_pr_id,omitempty"`
	CNvPrName  string                         `json:"cnv_pr_name,omitempty"`
	Transform  ObjectEMUPointBounds           `json:"transform_emu,omitempty"`
	CustomPath *microFixtureSourceCustomPath  `json:"custom_path,omitempty"`
	Shadow     *microFixtureSourceOuterShadow `json:"outer_shadow,omitempty"`
}

type microFixtureSourceCustomPath struct {
	Width  int64                         `json:"width"`
	Height int64                         `json:"height"`
	Points []microFixtureSourcePathPoint `json:"points,omitempty"`
}

type microFixtureSourcePathPoint struct {
	Command string `json:"command"`
	X       int64  `json:"x"`
	Y       int64  `json:"y"`
}

type microFixtureSourceOuterShadow struct {
	BlurRadius      int64  `json:"blur_radius"`
	Distance        int64  `json:"distance"`
	Direction       int64  `json:"direction"`
	Alignment       string `json:"alignment,omitempty"`
	RotateWithShape string `json:"rotate_with_shape,omitempty"`
}

type microFixtureShadowRenderSummary struct {
	Basis                       string              `json:"basis,omitempty"`
	Canvas                      microFixtureSize    `json:"canvas_pixels"`
	TargetBounds                ObjectPixelBounds   `json:"target_bounds"`
	Offset                      microFixturePoint   `json:"offset_pixels"`
	BlurPixels                  int                 `json:"blur_pixels"`
	ShadowBounds                ObjectPixelBounds   `json:"shadow_bounds"`
	TargetCustomPathPixelPoints []microFixturePoint `json:"target_custom_path_pixel_points,omitempty"`
	ShadowCustomPathPixelPoints []microFixturePoint `json:"shadow_custom_path_pixel_points,omitempty"`
	PaintBounds                 ObjectPixelBounds   `json:"paint_bounds"`
}

type microFixtureShadowPhaseSearchArtifact struct {
	ManifestPath   string                             `json:"manifest_path,omitempty"`
	FixturePath    string                             `json:"fixture_path,omitempty"`
	TargetCompared string                             `json:"target_compared,omitempty"`
	Basis          string                             `json:"basis,omitempty"`
	AnalyzedPixels int                                `json:"analyzed_pixels"`
	Baseline       *microFixtureShadowPhaseCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureShadowPhaseCandidate `json:"candidates"`
}

type microFixtureShadowCompositeSearchArtifact struct {
	ManifestPath   string                                 `json:"manifest_path,omitempty"`
	FixturePath    string                                 `json:"fixture_path,omitempty"`
	TargetCompared string                                 `json:"target_compared,omitempty"`
	Basis          string                                 `json:"basis,omitempty"`
	Baseline       *microFixtureShadowCompositeCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureShadowCompositeCandidate `json:"candidates"`
}

type microFixtureShadowParameterSearchArtifact struct {
	ManifestPath   string                                 `json:"manifest_path,omitempty"`
	FixturePath    string                                 `json:"fixture_path,omitempty"`
	TargetCompared string                                 `json:"target_compared,omitempty"`
	Basis          string                                 `json:"basis,omitempty"`
	Baseline       *microFixtureShadowParameterCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureShadowParameterCandidate `json:"candidates"`
}

type microFixtureShadowKernelSearchArtifact struct {
	ManifestPath   string                              `json:"manifest_path,omitempty"`
	FixturePath    string                              `json:"fixture_path,omitempty"`
	TargetCompared string                              `json:"target_compared,omitempty"`
	Basis          string                              `json:"basis,omitempty"`
	Baseline       *microFixtureShadowKernelCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureShadowKernelCandidate `json:"candidates"`
}

type microFixtureShadowGeometrySearchArtifact struct {
	ManifestPath   string                                `json:"manifest_path,omitempty"`
	FixturePath    string                                `json:"fixture_path,omitempty"`
	TargetCompared string                                `json:"target_compared,omitempty"`
	Basis          string                                `json:"basis,omitempty"`
	Baseline       *microFixtureShadowGeometryCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureShadowGeometryCandidate `json:"candidates"`
}

type microFixtureRectEdgeBlendSearchArtifact struct {
	ManifestPath   string                               `json:"manifest_path,omitempty"`
	FixturePath    string                               `json:"fixture_path,omitempty"`
	TargetCompared string                               `json:"target_compared,omitempty"`
	Basis          string                               `json:"basis,omitempty"`
	Baseline       *microFixtureRectEdgeBlendCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixtureRectEdgeBlendCandidate `json:"candidates"`
}

type microFixturePictureResampleSearchArtifact struct {
	ManifestPath   string                                 `json:"manifest_path,omitempty"`
	FixturePath    string                                 `json:"fixture_path,omitempty"`
	TargetCompared string                                 `json:"target_compared,omitempty"`
	Basis          string                                 `json:"basis,omitempty"`
	Baseline       *microFixturePictureResampleCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureResampleCandidate `json:"candidates"`
}

type microFixturePictureEdgeSearchArtifact struct {
	ManifestPath   string                             `json:"manifest_path,omitempty"`
	FixturePath    string                             `json:"fixture_path,omitempty"`
	TargetCompared string                             `json:"target_compared,omitempty"`
	Basis          string                             `json:"basis,omitempty"`
	Baseline       *microFixturePictureEdgeCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureEdgeCandidate `json:"candidates"`
}

type microFixturePictureGammaSearchArtifact struct {
	ManifestPath   string                              `json:"manifest_path,omitempty"`
	FixturePath    string                              `json:"fixture_path,omitempty"`
	TargetCompared string                              `json:"target_compared,omitempty"`
	Basis          string                              `json:"basis,omitempty"`
	Baseline       *microFixturePictureGammaCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureGammaCandidate `json:"candidates"`
}

type microFixturePictureKernelSearchArtifact struct {
	ManifestPath   string                               `json:"manifest_path,omitempty"`
	FixturePath    string                               `json:"fixture_path,omitempty"`
	TargetCompared string                               `json:"target_compared,omitempty"`
	Basis          string                               `json:"basis,omitempty"`
	Baseline       *microFixturePictureKernelCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureKernelCandidate `json:"candidates"`
}

type microFixturePictureAreaSearchArtifact struct {
	ManifestPath   string                             `json:"manifest_path,omitempty"`
	FixturePath    string                             `json:"fixture_path,omitempty"`
	TargetCompared string                             `json:"target_compared,omitempty"`
	Basis          string                             `json:"basis,omitempty"`
	Baseline       *microFixturePictureAreaCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureAreaCandidate `json:"candidates"`
}

type microFixturePicturePhaseSearchArtifact struct {
	ManifestPath   string                              `json:"manifest_path,omitempty"`
	FixturePath    string                              `json:"fixture_path,omitempty"`
	TargetCompared string                              `json:"target_compared,omitempty"`
	Basis          string                              `json:"basis,omitempty"`
	Baseline       *microFixturePicturePhaseCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePicturePhaseCandidate `json:"candidates"`
}

type microFixturePictureFractionalBoundsSearchArtifact struct {
	ManifestPath   string                                           `json:"manifest_path,omitempty"`
	FixturePath    string                                           `json:"fixture_path,omitempty"`
	TargetCompared string                                           `json:"target_compared,omitempty"`
	Basis          string                                           `json:"basis,omitempty"`
	Baseline       *microFixturePictureFractionalBoundsCandidate    `json:"baseline,omitempty"`
	Candidates     []microFixturePictureFractionalBoundsCandidate   `json:"candidates"`
	TargetBounds   microFixturePictureFractionalBoundsTargetSummary `json:"target_bounds"`
}

type microFixturePictureSourceModelSearchArtifact struct {
	ManifestPath   string                                         `json:"manifest_path,omitempty"`
	FixturePath    string                                         `json:"fixture_path,omitempty"`
	TargetCompared string                                         `json:"target_compared,omitempty"`
	Basis          string                                         `json:"basis,omitempty"`
	Baseline       *microFixturePictureSourceModelCandidate       `json:"baseline,omitempty"`
	Candidates     []microFixturePictureSourceModelCandidate      `json:"candidates"`
	SourceModels   []microFixturePictureSourceModelVariantSummary `json:"source_models,omitempty"`
}

type microFixturePictureContourCoverageSearchArtifact struct {
	ManifestPath   string                                        `json:"manifest_path,omitempty"`
	FixturePath    string                                        `json:"fixture_path,omitempty"`
	TargetCompared string                                        `json:"target_compared,omitempty"`
	Basis          string                                        `json:"basis,omitempty"`
	Baseline       *microFixturePictureContourCoverageCandidate  `json:"baseline,omitempty"`
	Candidates     []microFixturePictureContourCoverageCandidate `json:"candidates"`
}

type microFixtureShadowCompositeCandidate struct {
	ShiftX                        float64          `json:"shift_x"`
	ShiftY                        float64          `json:"shift_y"`
	SampleX                       float64          `json:"sample_x"`
	SampleY                       float64          `json:"sample_y"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int              `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int              `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int              `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int              `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixtureShadowParameterCandidate struct {
	Name                          string           `json:"name"`
	BlurPixels                    int              `json:"blur_pixels"`
	Alpha                         int              `json:"alpha"`
	OffsetX                       int              `json:"offset_x"`
	OffsetY                       int              `json:"offset_y"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int              `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int              `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int              `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int              `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixtureShadowKernelCandidate struct {
	Name                          string           `json:"name"`
	Kernel                        string           `json:"kernel"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int              `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int              `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int              `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int              `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixtureShadowGeometryCandidate struct {
	Name                          string             `json:"name"`
	RectSource                    string             `json:"rect_source"`
	OffsetSource                  string             `json:"offset_source"`
	SampleX                       float64            `json:"sample_x"`
	SampleY                       float64            `json:"sample_y"`
	TargetRect                    ObjectFloatBounds  `json:"target_rect"`
	ShadowRect                    ObjectFloatBounds  `json:"shadow_rect"`
	MaskBounds                    *ObjectPixelBounds `json:"mask_bounds,omitempty"`
	DifferentPixels               int                `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds   `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64              `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int                `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int                `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int                `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int                `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int                `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixtureRectEdgeBlendCandidate struct {
	Name                                       string           `json:"name"`
	CoverageQuantization                       string           `json:"coverage_quantization"`
	BlendQuantization                          string           `json:"blend_quantization"`
	DifferentPixels                            int              `json:"different_pixels"`
	DifferentBounds                            *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit              int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit                        int              `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels                      int              `json:"reference_darker_pixels"`
	ReferenceLighterPixels                     int              `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit                   int              `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit              int              `json:"reference_rgb_absolute_delta_8bit"`
	NonUnderpaintDifferentPixels               int              `json:"non_underpaint_different_pixels,omitempty"`
	NonUnderpaintDifferentBounds               *imageDiffBounds `json:"non_underpaint_different_bounds,omitempty"`
	NonUnderpaintTotalAbsoluteChannelDelta8Bit int64            `json:"non_underpaint_total_absolute_channel_delta_8bit,omitempty"`
}

type microFixturePictureResampleCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	OutputColor                   string            `json:"output_color"`
	Scaler                        string            `json:"scaler"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureEdgeCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	SourceFilter                  string            `json:"source_filter"`
	OutputFilter                  string            `json:"output_filter"`
	Scaler                        string            `json:"scaler"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureGammaCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	TransferMode                  string            `json:"transfer_mode"`
	Scaler                        string            `json:"scaler"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureKernelCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	Kernel                        string            `json:"kernel"`
	Support                       float64           `json:"support,omitempty"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureAreaCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	AreaMode                      string            `json:"area_mode"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePicturePhaseCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	SourcePhaseX                  float64           `json:"source_phase_x"`
	SourcePhaseY                  float64           `json:"source_phase_y"`
	TargetPhaseX                  float64           `json:"target_phase_x"`
	TargetPhaseY                  float64           `json:"target_phase_y"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureFractionalBoundsCandidate struct {
	Name                          string            `json:"name"`
	SourceColor                   string            `json:"source_color"`
	Sampler                       string            `json:"sampler"`
	SamplesPerAxis                int               `json:"samples_per_axis,omitempty"`
	TargetOffset                  ObjectFloatBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureFractionalBoundsTargetSummary struct {
	Fractional ObjectFloatBounds `json:"fractional"`
	Rounded    ObjectPixelBounds `json:"rounded"`
}

type microFixturePictureSourceModelCandidate struct {
	Name                          string            `json:"name"`
	SourceModel                   string            `json:"source_model"`
	Scaler                        string            `json:"scaler"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureContourCoverageCandidate struct {
	Name                          string            `json:"name"`
	TargetRounding                string            `json:"target_rounding"`
	TargetOffset                  ObjectPixelBounds `json:"target_offset"`
	Threshold                     int               `json:"threshold"`
	SamplesPerAxis                int               `json:"samples_per_axis"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	ReferenceDarkerPixels         int               `json:"reference_darker_pixels"`
	ReferenceLighterPixels        int               `json:"reference_lighter_pixels"`
	ReferenceRGBDeltaSum8Bit      int               `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta8Bit int               `json:"reference_rgb_absolute_delta_8bit"`
}

type microFixturePictureSourceModelVariantSummary struct {
	Name        string `json:"name"`
	GoType      string `json:"go_type"`
	ColorModel  string `json:"color_model"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	UniqueColor int    `json:"unique_color_count"`
}

type microFixturePicturePNGMetadataProfileArtifact struct {
	ManifestPath          string                         `json:"manifest_path,omitempty"`
	FixturePath           string                         `json:"fixture_path,omitempty"`
	Basis                 string                         `json:"basis,omitempty"`
	MediaPart             string                         `json:"media_part"`
	ByteSize              int                            `json:"byte_size"`
	SHA256                string                         `json:"sha256"`
	Width                 int                            `json:"width"`
	Height                int                            `json:"height"`
	BitDepth              int                            `json:"bit_depth"`
	ColorType             int                            `json:"color_type"`
	ColorTypeName         string                         `json:"color_type_name"`
	CompressionMethod     int                            `json:"compression_method"`
	FilterMethod          int                            `json:"filter_method"`
	InterlaceMethod       int                            `json:"interlace_method"`
	HasPalette            bool                           `json:"has_palette,omitempty"`
	PaletteEntries        int                            `json:"palette_entries,omitempty"`
	HasTransparency       bool                           `json:"has_transparency,omitempty"`
	TransparencyBytes     int                            `json:"transparency_bytes,omitempty"`
	HasGamma              bool                           `json:"has_gamma,omitempty"`
	Gamma                 *float64                       `json:"gamma,omitempty"`
	HasSRGB               bool                           `json:"has_srgb,omitempty"`
	SRGBRenderingIntent   *int                           `json:"srgb_rendering_intent,omitempty"`
	HasICCP               bool                           `json:"has_iccp,omitempty"`
	ICCProfileName        string                         `json:"icc_profile_name,omitempty"`
	ICCCompressionMethod  *int                           `json:"icc_compression_method,omitempty"`
	HasPhysicalPixelSize  bool                           `json:"has_physical_pixel_size,omitempty"`
	PhysicalPixelsPerUnit *microFixturePNGPhysicalPixels `json:"physical_pixels_per_unit,omitempty"`
	Chunks                []microFixturePNGChunk         `json:"chunks"`
}

type microFixturePNGPhysicalPixels struct {
	X    uint32 `json:"x"`
	Y    uint32 `json:"y"`
	Unit uint8  `json:"unit"`
}

type microFixturePNGChunk struct {
	Type     string `json:"type"`
	Length   uint32 `json:"length"`
	CRCValid bool   `json:"crc_valid"`
}

type microFixturePicturePipelineProfileArtifact struct {
	ManifestPath   string                                    `json:"manifest_path,omitempty"`
	FixturePath    string                                    `json:"fixture_path,omitempty"`
	TargetCompared string                                    `json:"target_compared,omitempty"`
	Basis          string                                    `json:"basis"`
	SourceDecode   microFixturePicturePipelineSourceDecode   `json:"source_decode"`
	Color          microFixturePicturePipelineColorStage     `json:"color"`
	Crop           microFixturePicturePipelineCropStage      `json:"crop"`
	Transform      microFixturePicturePipelineTransformStage `json:"transform"`
	Sampling       microFixturePicturePipelineSamplingStage  `json:"sampling"`
	Output         microFixturePicturePipelineOutputStage    `json:"output"`
}

type microFixturePicturePipelineSourceDecode struct {
	MediaPart       string                         `json:"media_part,omitempty"`
	ContentType     string                         `json:"content_type,omitempty"`
	RelationshipID  string                         `json:"relationship_id,omitempty"`
	Relationship    string                         `json:"relationship,omitempty"`
	GoType          string                         `json:"go_type,omitempty"`
	PartialFallback string                         `json:"partial_fallback,omitempty"`
	Stats           microFixturePictureSourceStats `json:"stats"`
	TopColors       []microFixtureColorCount       `json:"top_colors,omitempty"`
}

type microFixturePicturePipelineColorStage struct {
	Basis                        string                         `json:"basis"`
	DecodedSRGBStats             microFixturePictureSourceStats `json:"decoded_srgb_stats"`
	DisplayP3ConvertedStats      microFixturePictureSourceStats `json:"display_p3_converted_stats"`
	DisplayP3ChangedPixels       int                            `json:"display_p3_changed_pixels"`
	DisplayP3AbsoluteDelta8Bit   int64                          `json:"display_p3_absolute_delta_8bit"`
	DisplayP3MaxChannelDelta8Bit int                            `json:"display_p3_max_channel_delta_8bit"`
}

type microFixturePicturePipelineCropStage struct {
	Basis        string                         `json:"basis"`
	CropLeft     int64                          `json:"crop_left,omitempty"`
	CropTop      int64                          `json:"crop_top,omitempty"`
	CropRight    int64                          `json:"crop_right,omitempty"`
	CropBottom   int64                          `json:"crop_bottom,omitempty"`
	SourceBounds ObjectPixelBounds              `json:"source_bounds"`
	CropBounds   ObjectPixelBounds              `json:"crop_bounds"`
	CroppedStats microFixturePictureSourceStats `json:"cropped_stats"`
}

type microFixturePicturePipelineTransformStage struct {
	Basis          string                         `json:"basis"`
	FlipH          bool                           `json:"flip_h,omitempty"`
	FlipV          bool                           `json:"flip_v,omitempty"`
	AlphaModFixPct int64                          `json:"alpha_mod_fix_pct,omitempty"`
	Applied        bool                           `json:"applied"`
	OutputBounds   ObjectPixelBounds              `json:"output_bounds"`
	OutputStats    microFixturePictureSourceStats `json:"output_stats"`
}

type microFixturePicturePipelineSamplingStage struct {
	Basis                         string            `json:"basis"`
	Scaler                        string            `json:"scaler"`
	Canvas                        microFixtureSize  `json:"canvas"`
	AbsoluteTarget                ObjectPixelBounds `json:"absolute_target"`
	CropRelativeTarget            ObjectPixelBounds `json:"crop_relative_target"`
	FractionalSourceBounds        ObjectFloatBounds `json:"fractional_source_bounds"`
	SourceToTargetScaleX          float64           `json:"source_to_target_scale_x,omitempty"`
	SourceToTargetScaleY          float64           `json:"source_to_target_scale_y,omitempty"`
	PreOutputDiffAgainstGot       imageDiff         `json:"pre_output_diff_against_got"`
	PreOutputDiffAgainstReference imageDiff         `json:"pre_output_diff_against_reference"`
}

type microFixturePicturePipelineOutputStage struct {
	Basis                string    `json:"basis"`
	DisplayP3Output      bool      `json:"display_p3_output"`
	OcclusionMaskApplied bool      `json:"occlusion_mask_applied,omitempty"`
	OccludedPixels       int       `json:"occluded_pixels,omitempty"`
	DiffAgainstGot       imageDiff `json:"diff_against_got"`
	DiffAgainstReference imageDiff `json:"diff_against_reference"`
}

type microFixturePictureResidualProfileArtifact struct {
	ManifestPath                      string                         `json:"manifest_path,omitempty"`
	FixturePath                       string                         `json:"fixture_path,omitempty"`
	TargetCompared                    string                         `json:"target_compared,omitempty"`
	Basis                             string                         `json:"basis,omitempty"`
	Source                            microFixturePictureSourceStats `json:"source"`
	CropWidth                         int                            `json:"crop_width"`
	CropHeight                        int                            `json:"crop_height"`
	DifferentPixels                   int                            `json:"different_pixels"`
	DifferentBounds                   *imageDiffBounds               `json:"different_bounds,omitempty"`
	GrayscaleDifferentPixels          int                            `json:"grayscale_different_pixels"`
	EdgeCoverageDifferentPixels       int                            `json:"edge_coverage_different_pixels"`
	PureBlackWhiteDifferentPixels     int                            `json:"pure_black_white_different_pixels"`
	ColoredDifferentPixels            int                            `json:"colored_different_pixels"`
	GotAntialiasDifferentPixels       int                            `json:"got_antialias_different_pixels"`
	ReferenceAntialiasDifferentPixels int                            `json:"reference_antialias_different_pixels"`
	GotHardDifferentPixels            int                            `json:"got_hard_different_pixels"`
	ReferenceHardDifferentPixels      int                            `json:"reference_hard_different_pixels"`
	TopGotLumaBuckets                 []microFixtureLumaBucket       `json:"top_got_luma_buckets,omitempty"`
	TopReferenceLumaBuckets           []microFixtureLumaBucket       `json:"top_reference_luma_buckets,omitempty"`
	TopReferenceMinusGotLumaBuckets   []microFixtureDeltaCount       `json:"top_reference_minus_got_luma_buckets,omitempty"`
	TopSourceColors                   []microFixtureColorCount       `json:"top_source_colors,omitempty"`
}

type microFixturePictureSourceCorrespondenceProfileArtifact struct {
	ManifestPath                     string                                 `json:"manifest_path,omitempty"`
	FixturePath                      string                                 `json:"fixture_path,omitempty"`
	TargetCompared                   string                                 `json:"target_compared,omitempty"`
	Basis                            string                                 `json:"basis,omitempty"`
	Source                           microFixturePictureSourceStats         `json:"source"`
	CropWidth                        int                                    `json:"crop_width"`
	CropHeight                       int                                    `json:"crop_height"`
	TargetMode                       string                                 `json:"target_mode"`
	TargetRelativeBounds             ObjectPixelBounds                      `json:"target_relative_bounds"`
	DifferentPixels                  int                                    `json:"different_pixels"`
	DifferentBounds                  *imageDiffBounds                       `json:"different_bounds,omitempty"`
	SourceCoordinateBounds           *ObjectPixelBounds                     `json:"source_coordinate_bounds,omitempty"`
	NearestSourceHardPixels          int                                    `json:"nearest_source_hard_pixels"`
	NearestSourceAntialiasPixels     int                                    `json:"nearest_source_antialias_pixels"`
	NearestSourceBlackPixels         int                                    `json:"nearest_source_black_pixels"`
	NearestSourceWhitePixels         int                                    `json:"nearest_source_white_pixels"`
	NearestSourceGrayPixels          int                                    `json:"nearest_source_gray_pixels"`
	Mixed3x3SourceNeighborhoodPixels int                                    `json:"mixed_3x3_source_neighborhood_pixels"`
	Solid3x3SourceNeighborhoodPixels int                                    `json:"solid_3x3_source_neighborhood_pixels"`
	ReferenceDarkerPixels            int                                    `json:"reference_darker_pixels"`
	ReferenceLighterPixels           int                                    `json:"reference_lighter_pixels"`
	ReferenceMinusGotLumaSum         int                                    `json:"reference_minus_got_luma_sum"`
	TopNearestSourceColors           []microFixtureColorCount               `json:"top_nearest_source_colors,omitempty"`
	TopNearestSourceLumaBuckets      []microFixtureLumaBucket               `json:"top_nearest_source_luma_buckets,omitempty"`
	TopSourceRows                    []microFixtureAxisDeltaCount           `json:"top_source_rows,omitempty"`
	TopSourceColumns                 []microFixtureAxisDeltaCount           `json:"top_source_columns,omitempty"`
	TopSourcePixels                  []microFixtureSourcePixelResidualCount `json:"top_source_pixels,omitempty"`
	TopReferenceMinusGotLumaBuckets  []microFixtureDeltaCount               `json:"top_reference_minus_got_luma_buckets,omitempty"`
}

type microFixtureSourcePixelResidualCount struct {
	X                        int    `json:"x"`
	Y                        int    `json:"y"`
	Count                    int    `json:"count"`
	RGBA                     string `json:"rgba"`
	Luma                     int    `json:"luma"`
	Mixed3x3                 bool   `json:"mixed_3x3"`
	GotHardPixels            int    `json:"got_hard_pixels"`
	GotAntialiasPixels       int    `json:"got_antialias_pixels"`
	ReferenceHardPixels      int    `json:"reference_hard_pixels"`
	ReferenceAntialiasPixels int    `json:"reference_antialias_pixels"`
	ReferenceDarkerPixels    int    `json:"reference_darker_pixels"`
	ReferenceLighterPixels   int    `json:"reference_lighter_pixels"`
	ReferenceMinusGotLumaSum int    `json:"reference_minus_got_luma_sum"`
}

type microFixturePictureEdgeGeometryProfileArtifact struct {
	ManifestPath                    string                       `json:"manifest_path,omitempty"`
	FixturePath                     string                       `json:"fixture_path,omitempty"`
	TargetCompared                  string                       `json:"target_compared,omitempty"`
	Basis                           string                       `json:"basis,omitempty"`
	CropWidth                       int                          `json:"crop_width"`
	CropHeight                      int                          `json:"crop_height"`
	TargetRelativeFractionalBounds  ObjectFloatBounds            `json:"target_relative_fractional_bounds"`
	TargetRelativeOutputBounds      ObjectPixelBounds            `json:"target_relative_output_bounds"`
	DifferentPixels                 int                          `json:"different_pixels"`
	DifferentBounds                 *imageDiffBounds             `json:"different_bounds,omitempty"`
	CropLeftEdgePixels              int                          `json:"crop_left_edge_pixels"`
	CropRightEdgePixels             int                          `json:"crop_right_edge_pixels"`
	CropTopEdgePixels               int                          `json:"crop_top_edge_pixels"`
	CropBottomEdgePixels            int                          `json:"crop_bottom_edge_pixels"`
	NearCropEdgePixels              int                          `json:"near_crop_edge_pixels"`
	InteriorPixels                  int                          `json:"interior_pixels"`
	GotHardPixels                   int                          `json:"got_hard_pixels"`
	GotAntialiasPixels              int                          `json:"got_antialias_pixels"`
	ReferenceHardPixels             int                          `json:"reference_hard_pixels"`
	ReferenceAntialiasPixels        int                          `json:"reference_antialias_pixels"`
	ReferenceDarkerPixels           int                          `json:"reference_darker_pixels"`
	ReferenceLighterPixels          int                          `json:"reference_lighter_pixels"`
	ReferenceMinusGotLumaSum        int                          `json:"reference_minus_got_luma_sum"`
	TopRows                         []microFixtureAxisDeltaCount `json:"top_rows,omitempty"`
	TopColumns                      []microFixtureAxisDeltaCount `json:"top_columns,omitempty"`
	TopReferenceMinusGotLumaBuckets []microFixtureDeltaCount     `json:"top_reference_minus_got_luma_buckets,omitempty"`
}

type microFixturePictureSourceStats struct {
	Width        int `json:"width"`
	Height       int `json:"height"`
	UniqueColors int `json:"unique_colors"`
	OpaquePixels int `json:"opaque_pixels"`
	AlphaPixels  int `json:"alpha_pixels"`
}

type microFixtureLumaBucket struct {
	Luma  int `json:"luma"`
	Count int `json:"count"`
}

type microFixtureColorCount struct {
	RGBA  string `json:"rgba"`
	Count int    `json:"count"`
}

type microFixtureTargetOwnershipSummary struct {
	Root                      string                        `json:"root"`
	TotalManifests            int                           `json:"total_manifests"`
	ManifestsWithTargetScope  int                           `json:"manifests_with_target_scope"`
	CleanFailures             []microFixtureOwnershipRecord `json:"clean_failures,omitempty"`
	ContaminatedFailures      []microFixtureOwnershipRecord `json:"contaminated_failures,omitempty"`
	PartialUnderpaintFailures []microFixtureOwnershipRecord `json:"partial_underpaint_failures,omitempty"`
	UnscopedManifests         []string                      `json:"unscoped_manifests,omitempty"`
}

type microFixtureOwnershipRecord struct {
	ManifestPath                     string `json:"manifest_path"`
	DeckInput                        string `json:"deck_input"`
	SlideNumber                      int    `json:"slide_number"`
	Kind                             string `json:"kind"`
	CNvPrID                          string `json:"cnv_pr_id"`
	CNvPrName                        string `json:"cnv_pr_name"`
	DifferentPixels                  int    `json:"different_pixels"`
	DifferentPixelsInsideObjectMask  int    `json:"different_pixels_inside_object_mask"`
	DifferentPixelsOutsideObjectMask int    `json:"different_pixels_outside_object_mask"`
	PartialAlphaDifferentPixels      int    `json:"partial_alpha_different_pixels,omitempty"`
	PartialAlphaOverUnderpaintPixels int    `json:"partial_alpha_over_underpaint_pixels,omitempty"`
	NonUnderpaintDifferentPixels     int    `json:"non_underpaint_different_pixels,omitempty"`
	Warning                          string `json:"warning,omitempty"`
}

type rendererProductionFailureScoreboard struct {
	Root                      string                          `json:"root"`
	Basis                     string                          `json:"basis"`
	SlideCount                int                             `json:"slide_count"`
	TotalSlideDifferentPixels int                             `json:"total_slide_different_pixels"`
	ObjectOverlapByPrimitive  []rendererPrimitiveScore        `json:"object_overlap_by_primitive,omitempty"`
	CleanFixturesByKind       []rendererCleanFixtureScore     `json:"clean_fixtures_by_kind,omitempty"`
	CleanFixturesByObjectName []rendererCleanFixtureNameScore `json:"clean_fixtures_by_object_name,omitempty"`
	TopSlides                 []rendererSlideScore            `json:"top_slides,omitempty"`
	TopCleanFailures          []microFixtureOwnershipRecord   `json:"top_clean_failures,omitempty"`
	AttributionArtifactCount  int                             `json:"attribution_artifact_count"`
	CleanFixtureFailureCount  int                             `json:"clean_fixture_failure_count,omitempty"`
	OwnershipSummaryPath      string                          `json:"ownership_summary_path,omitempty"`
}

type rendererPrimitiveScore struct {
	Kind              string `json:"kind"`
	SuspectedGap      string `json:"suspected_gap"`
	ObjectCount       int    `json:"object_count"`
	OverlapDiffPixels int    `json:"overlap_diff_pixels"`
}

type rendererCleanFixtureScore struct {
	Kind            string `json:"kind"`
	FailureCount    int    `json:"failure_count"`
	DifferentPixels int    `json:"different_pixels"`
}

type rendererCleanFixtureNameScore struct {
	Kind            string `json:"kind"`
	CNvPrName       string `json:"cnv_pr_name"`
	FailureCount    int    `json:"failure_count"`
	DifferentPixels int    `json:"different_pixels"`
}

type rendererSlideScore struct {
	DeckInput       string `json:"deck_input"`
	SlideNumber     int    `json:"slide_number"`
	DifferentPixels int    `json:"different_pixels"`
}

type microFixtureShadowPhaseCandidate struct {
	ShiftX                 float64 `json:"shift_x"`
	ShiftY                 float64 `json:"shift_y"`
	SampleX                float64 `json:"sample_x"`
	SampleY                float64 `json:"sample_y"`
	DifferentPixels        int     `json:"different_pixels"`
	ReferenceAlphaGreater  int     `json:"reference_alpha_greater"`
	ReferenceAlphaLess     int     `json:"reference_alpha_less"`
	ReferenceAlphaDeltaSum int     `json:"reference_alpha_delta_sum_8bit"`
	AbsoluteAlphaDeltaSum  int     `json:"absolute_alpha_delta_sum_8bit"`
}

type microFixturePoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type microFixtureSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type microFixtureUnderpaint struct {
	CNvPrID            string            `json:"cnv_pr_id,omitempty"`
	CNvPrName          string            `json:"cnv_pr_name,omitempty"`
	Kind               string            `json:"kind"`
	ZOrder             int               `json:"z_order"`
	Bounds             ObjectPixelBounds `json:"bounds"`
	ObjectArtifactPath string            `json:"object_artifact_path,omitempty"`
}

type microFixtureUnderpaintChainSummary struct {
	ObjectOnlyDifferentPixels                         int `json:"object_only_different_pixels,omitempty"`
	ChainDifferentPixels                              int `json:"chain_different_pixels,omitempty"`
	DifferentPixelsDelta                              int `json:"different_pixels_delta,omitempty"`
	ObjectOnlyUnderpaintedPartialAlphaDifferentPixels int `json:"object_only_underpainted_partial_alpha_different_pixels,omitempty"`
	ChainUnderpaintedPartialAlphaDifferentPixels      int `json:"chain_underpainted_partial_alpha_different_pixels,omitempty"`
	UnderpaintedPartialAlphaDifferentPixelsDelta      int `json:"underpainted_partial_alpha_different_pixels_delta,omitempty"`
	ObjectOnlyPlainPartialAlphaDifferentPixels        int `json:"object_only_plain_partial_alpha_different_pixels,omitempty"`
	ChainPlainPartialAlphaDifferentPixels             int `json:"chain_plain_partial_alpha_different_pixels,omitempty"`
	PlainPartialAlphaDifferentPixelsDelta             int `json:"plain_partial_alpha_different_pixels_delta,omitempty"`
	ChainReferenceRGBDeltaSum8                        int `json:"chain_reference_rgb_delta_sum_8bit,omitempty"`
	ChainReferenceRGBAbsoluteDeltaSum8                int `json:"chain_reference_rgb_absolute_delta_sum_8bit,omitempty"`
}

type microFixtureSourceImage struct {
	Part   string `json:"part,omitempty"`
	Format string `json:"format,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type microFixtureSampling struct {
	IntegerGeometryWidth     int     `json:"integer_geometry_width"`
	IntegerGeometryHeight    int     `json:"integer_geometry_height"`
	FractionalGeometryWidth  float64 `json:"fractional_geometry_width,omitempty"`
	FractionalGeometryHeight float64 `json:"fractional_geometry_height,omitempty"`
	FractionalOffsetX        float64 `json:"fractional_offset_x,omitempty"`
	FractionalOffsetY        float64 `json:"fractional_offset_y,omitempty"`
	OutputCropOffsetX        int     `json:"output_crop_offset_x,omitempty"`
	OutputCropOffsetY        int     `json:"output_crop_offset_y,omitempty"`
	OutputCropWidth          int     `json:"output_crop_width,omitempty"`
	OutputCropHeight         int     `json:"output_crop_height,omitempty"`
	SourceToGeometryScaleX   float64 `json:"source_to_geometry_scale_x,omitempty"`
	SourceToGeometryScaleY   float64 `json:"source_to_geometry_scale_y,omitempty"`
}

type microFixtureTargetScope struct {
	Basis                                                            string                   `json:"basis,omitempty"`
	ObjectMaskPath                                                   string                   `json:"object_mask_path,omitempty"`
	TargetCompared                                                   string                   `json:"target_compared,omitempty"`
	CropPixels                                                       int                      `json:"crop_pixels,omitempty"`
	ComparedPixels                                                   int                      `json:"compared_pixels,omitempty"`
	ObjectMaskPixels                                                 int                      `json:"object_mask_pixels,omitempty"`
	ObjectMaskFullAlphaPixels                                        int                      `json:"object_mask_full_alpha_pixels,omitempty"`
	ObjectMaskPartialAlphaPixels                                     int                      `json:"object_mask_partial_alpha_pixels,omitempty"`
	ObjectMaskLowAlphaPixels                                         int                      `json:"object_mask_low_alpha_pixels,omitempty"`
	ObjectMaskMidAlphaPixels                                         int                      `json:"object_mask_mid_alpha_pixels,omitempty"`
	ObjectMaskHighAlphaPixels                                        int                      `json:"object_mask_high_alpha_pixels,omitempty"`
	ObjectMaskPartialAlphaDarkPixels                                 int                      `json:"object_mask_partial_alpha_dark_pixels,omitempty"`
	ObjectMaskPartialAlphaLightPixels                                int                      `json:"object_mask_partial_alpha_light_pixels,omitempty"`
	ObjectMaskPartialAlphaOtherPixels                                int                      `json:"object_mask_partial_alpha_other_pixels,omitempty"`
	ObjectMaskPartialAlphaPixelsOverUnderpaint                       int                      `json:"object_mask_partial_alpha_pixels_over_underpaint,omitempty"`
	DifferentPixels                                                  int                      `json:"different_pixels,omitempty"`
	DifferentBounds                                                  *imageDiffBounds         `json:"different_bounds,omitempty"`
	DifferentPixelsReferenceDarker                                   int                      `json:"different_pixels_reference_darker,omitempty"`
	ReferenceDarkerBounds                                            *imageDiffBounds         `json:"reference_darker_bounds,omitempty"`
	DifferentPixelsReferenceLighter                                  int                      `json:"different_pixels_reference_lighter,omitempty"`
	ReferenceLighterBounds                                           *imageDiffBounds         `json:"reference_lighter_bounds,omitempty"`
	ReferenceRGBDeltaSum8                                            int                      `json:"reference_rgb_delta_sum_8bit,omitempty"`
	ReferenceRGBAbsoluteDeltaSum8                                    int                      `json:"reference_rgb_absolute_delta_sum_8bit,omitempty"`
	DifferentPixelsInsideObjectMask                                  int                      `json:"different_pixels_inside_object_mask,omitempty"`
	DifferentPixelsInsideObjectMaskReferenceDarker                   int                      `json:"different_pixels_inside_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideObjectMaskReferenceLighter                  int                      `json:"different_pixels_inside_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideFullAlphaObjectMask                         int                      `json:"different_pixels_inside_full_alpha_object_mask,omitempty"`
	DifferentPixelsInsidePartialAlphaObjectMask                      int                      `json:"different_pixels_inside_partial_alpha_object_mask,omitempty"`
	DifferentPixelsInsidePartialAlphaObjectMaskReferenceDarker       int                      `json:"different_pixels_inside_partial_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsidePartialAlphaObjectMaskReferenceLighter      int                      `json:"different_pixels_inside_partial_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideLowAlphaObjectMask                          int                      `json:"different_pixels_inside_low_alpha_object_mask,omitempty"`
	DifferentPixelsInsideLowAlphaObjectMaskReferenceDarker           int                      `json:"different_pixels_inside_low_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideLowAlphaObjectMaskReferenceLighter          int                      `json:"different_pixels_inside_low_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideMidAlphaObjectMask                          int                      `json:"different_pixels_inside_mid_alpha_object_mask,omitempty"`
	DifferentPixelsInsideMidAlphaObjectMaskReferenceDarker           int                      `json:"different_pixels_inside_mid_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideMidAlphaObjectMaskReferenceLighter          int                      `json:"different_pixels_inside_mid_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideHighAlphaObjectMask                         int                      `json:"different_pixels_inside_high_alpha_object_mask,omitempty"`
	DifferentPixelsInsideHighAlphaObjectMaskReferenceDarker          int                      `json:"different_pixels_inside_high_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideHighAlphaObjectMaskReferenceLighter         int                      `json:"different_pixels_inside_high_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideDarkPartialAlphaObjectMask                  int                      `json:"different_pixels_inside_dark_partial_alpha_object_mask,omitempty"`
	DifferentPixelsInsideDarkPartialAlphaObjectMaskReferenceDarker   int                      `json:"different_pixels_inside_dark_partial_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideDarkPartialAlphaObjectMaskReferenceLighter  int                      `json:"different_pixels_inside_dark_partial_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideLightPartialAlphaObjectMask                 int                      `json:"different_pixels_inside_light_partial_alpha_object_mask,omitempty"`
	DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceDarker  int                      `json:"different_pixels_inside_light_partial_alpha_object_mask_reference_darker,omitempty"`
	DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceLighter int                      `json:"different_pixels_inside_light_partial_alpha_object_mask_reference_lighter,omitempty"`
	DifferentPixelsInsideOtherPartialAlphaObjectMask                 int                      `json:"different_pixels_inside_other_partial_alpha_object_mask,omitempty"`
	DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint        int                      `json:"different_pixels_inside_partial_alpha_object_mask_over_underpaint,omitempty"`
	DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint     int                      `json:"different_pixels_inside_partial_alpha_object_mask_without_underpaint,omitempty"`
	DifferentPixelsOutsideObjectMask                                 int                      `json:"different_pixels_outside_object_mask,omitempty"`
	OutsideObjectMaskRatio                                           float64                  `json:"outside_object_mask_ratio,omitempty"`
	TopDifferentRows                                                 []microFixtureAxisCount  `json:"top_different_rows,omitempty"`
	TopDifferentColumns                                              []microFixtureAxisCount  `json:"top_different_columns,omitempty"`
	TopReferenceDarkerRows                                           []microFixtureAxisCount  `json:"top_reference_darker_rows,omitempty"`
	TopReferenceLighterRows                                          []microFixtureAxisCount  `json:"top_reference_lighter_rows,omitempty"`
	TopReferenceDarkerColumns                                        []microFixtureAxisCount  `json:"top_reference_darker_columns,omitempty"`
	TopReferenceLighterColumns                                       []microFixtureAxisCount  `json:"top_reference_lighter_columns,omitempty"`
	TopReferenceRGBDeltaSums8                                        []microFixtureDeltaCount `json:"top_reference_rgb_delta_sums_8bit,omitempty"`
	TopGotColors                                                     []microFixtureColorCount `json:"top_got_colors,omitempty"`
	TopReferenceColors                                               []microFixtureColorCount `json:"top_reference_colors,omitempty"`
	TopDifferentGotColors                                            []microFixtureColorCount `json:"top_different_got_colors,omitempty"`
	TopDifferentReferenceColors                                      []microFixtureColorCount `json:"top_different_reference_colors,omitempty"`
	Warning                                                          string                   `json:"warning,omitempty"`
}

type microFixtureShadowAlphaScope struct {
	Basis                           string                   `json:"basis,omitempty"`
	ObjectMaskPath                  string                   `json:"object_mask_path,omitempty"`
	BackgroundPath                  string                   `json:"background_path,omitempty"`
	TargetCompared                  string                   `json:"target_compared,omitempty"`
	ComparedPixels                  int                      `json:"compared_pixels,omitempty"`
	ShadowMaskPixels                int                      `json:"shadow_mask_pixels,omitempty"`
	AnalyzedPixels                  int                      `json:"analyzed_pixels,omitempty"`
	ReferenceAlphaGreaterPixels     int                      `json:"reference_alpha_greater_pixels,omitempty"`
	ReferenceAlphaGreaterBounds     *imageDiffBounds         `json:"reference_alpha_greater_bounds,omitempty"`
	ReferenceAlphaGreaterDeltaSum8  int                      `json:"reference_alpha_greater_delta_sum_8bit,omitempty"`
	ReferenceAlphaGreaterCentroid   *microFixtureFloatPoint  `json:"reference_alpha_greater_centroid,omitempty"`
	ReferenceAlphaLessPixels        int                      `json:"reference_alpha_less_pixels,omitempty"`
	ReferenceAlphaLessBounds        *imageDiffBounds         `json:"reference_alpha_less_bounds,omitempty"`
	ReferenceAlphaLessDeltaSum8     int                      `json:"reference_alpha_less_delta_sum_8bit,omitempty"`
	ReferenceAlphaLessCentroid      *microFixtureFloatPoint  `json:"reference_alpha_less_centroid,omitempty"`
	ReferenceAlphaDeltaSum8         int                      `json:"reference_alpha_delta_sum_8bit,omitempty"`
	ReferenceAlphaAbsoluteDeltaSum8 int                      `json:"reference_alpha_absolute_delta_sum_8bit,omitempty"`
	TopReferenceAlphaDeltaSums8     []microFixtureDeltaCount `json:"top_reference_alpha_delta_sums_8bit,omitempty"`
	TopReferenceAlphaGreaterRows    []microFixtureAxisCount  `json:"top_reference_alpha_greater_rows,omitempty"`
	TopReferenceAlphaLessRows       []microFixtureAxisCount  `json:"top_reference_alpha_less_rows,omitempty"`
	TopReferenceAlphaGreaterColumns []microFixtureAxisCount  `json:"top_reference_alpha_greater_columns,omitempty"`
	TopReferenceAlphaLessColumns    []microFixtureAxisCount  `json:"top_reference_alpha_less_columns,omitempty"`
	Warning                         string                   `json:"warning,omitempty"`
}

type microFixtureAxisCount struct {
	Index int `json:"index"`
	Count int `json:"count"`
}

type microFixtureAxisDeltaCount struct {
	Index                    int `json:"index"`
	Count                    int `json:"count"`
	GotHardPixels            int `json:"got_hard_pixels,omitempty"`
	GotAntialiasPixels       int `json:"got_antialias_pixels,omitempty"`
	ReferenceHardPixels      int `json:"reference_hard_pixels,omitempty"`
	ReferenceAntialiasPixels int `json:"reference_antialias_pixels,omitempty"`
	ReferenceDarkerPixels    int `json:"reference_darker_pixels,omitempty"`
	ReferenceLighterPixels   int `json:"reference_lighter_pixels,omitempty"`
	ReferenceMinusGotLumaSum int `json:"reference_minus_got_luma_sum,omitempty"`
}

type microFixtureDeltaCount struct {
	Delta int `json:"delta"`
	Count int `json:"count"`
}

type microFixtureFloatPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type objectAttributionArtifact struct {
	DeckInput              string                  `json:"deck_input"`
	SlideNumber            int                     `json:"slide_number"`
	FullDiff               imageDiff               `json:"full_diff"`
	BinarySearch           objectBinarySearch      `json:"binary_search"`
	CumulativeProbes       []objectCumulativeProbe `json:"cumulative_probes,omitempty"`
	LargestCumulativeDelta *objectCumulativeProbe  `json:"largest_cumulative_delta,omitempty"`
	Objects                []objectFailureRecord   `json:"objects"`
}

type objectBinarySearch struct {
	Method                   string `json:"method"`
	ThresholdDifferentPixels int    `json:"threshold_different_pixels"`
	TargetZOrder             int    `json:"target_z_order"`
	ProbeZOrders             []int  `json:"probe_z_orders"`
}

type objectCumulativeProbe struct {
	ZOrder                    int    `json:"z_order"`
	CNvPrID                   string `json:"cnv_pr_id,omitempty"`
	CNvPrName                 string `json:"cnv_pr_name,omitempty"`
	Kind                      string `json:"kind"`
	CumulativeDifferentPixels int    `json:"cumulative_different_pixels"`
	CumulativeDiffDeltaPixels int    `json:"cumulative_diff_delta_pixels"`
	AbsoluteDeltaPixels       int    `json:"absolute_delta_pixels"`
	ThroughArtifactPath       string `json:"through_artifact_path,omitempty"`
	SuspectedRendererGap      string `json:"suspected_renderer_gap"`
}

type objectFailureRecord struct {
	DeckInput                   string               `json:"deck_input"`
	SlideNumber                 int                  `json:"slide_number"`
	SlidePart                   string               `json:"slide_part"`
	SourcePart                  string               `json:"source_part"`
	XMLPath                     string               `json:"xml_path,omitempty"`
	CNvPrID                     string               `json:"cnv_pr_id,omitempty"`
	CNvPrName                   string               `json:"cnv_pr_name,omitempty"`
	Kind                        string               `json:"kind"`
	ZOrder                      int                  `json:"z_order"`
	Bounds                      ObjectEMUPointBounds `json:"bounds_emu,omitempty"`
	PixelBounds                 ObjectPixelBounds    `json:"pixel_bounds,omitempty"`
	FractionalBounds            ObjectFloatBounds    `json:"fractional_pixel_bounds,omitempty"`
	OutputPixelBounds           *ObjectPixelBounds   `json:"output_pixel_bounds,omitempty"`
	ResolvedStyle               ObjectStyleSummary   `json:"resolved_style"`
	ObservedDiff                string               `json:"observed_diff"`
	OverlapDiffPixels           int                  `json:"overlap_diff_pixels"`
	ObjectArtifactChangedPixels int                  `json:"object_artifact_changed_pixels"`
	CumulativeDifferentPixels   int                  `json:"cumulative_different_pixels,omitempty"`
	CumulativeDiffDeltaPixels   int                  `json:"cumulative_diff_delta_pixels,omitempty"`
	SuspectedRendererGap        string               `json:"suspected_renderer_gap"`
	BeforeArtifactPath          string               `json:"before_artifact_path,omitempty"`
	ObjectArtifactPath          string               `json:"object_artifact_path,omitempty"`
	ThroughArtifactPath         string               `json:"through_artifact_path,omitempty"`
}

type microFixtureShapeObjectProfile struct {
	ManifestPath        string                  `json:"manifest_path"`
	FixturePath         string                  `json:"fixture_path"`
	DeckInput           string                  `json:"deck_input"`
	SlideNumber         int                     `json:"slide_number"`
	CNvPrID             string                  `json:"cnv_pr_id,omitempty"`
	CNvPrName           string                  `json:"cnv_pr_name,omitempty"`
	Kind                string                  `json:"kind"`
	HasShapeAutofit     bool                    `json:"has_shape_autofit,omitempty"`
	HasNormalAutofit    bool                    `json:"has_normal_autofit,omitempty"`
	HasNoAutofit        bool                    `json:"has_no_autofit,omitempty"`
	TextWrap            string                  `json:"text_wrap,omitempty"`
	Canvas              microFixtureSize        `json:"canvas"`
	GeometryTarget      ObjectPixelBounds       `json:"geometry_target"`
	TextTarget          ObjectPixelBounds       `json:"text_target"`
	TextBoundsBeforeFit ObjectPixelBounds       `json:"text_bounds_before_fit"`
	TextBoundsAfterFit  ObjectPixelBounds       `json:"text_bounds_after_fit"`
	MeasuredTextWidth   int                     `json:"measured_text_width,omitempty"`
	MeasuredTextHeight  int                     `json:"measured_text_height,omitempty"`
	MeasureError        string                  `json:"measure_error,omitempty"`
	FillColor           string                  `json:"fill_color,omitempty"`
	TextColor           string                  `json:"text_color,omitempty"`
	Diff                imageDiff               `json:"diff"`
	TargetScope         microFixtureTargetScope `json:"target_scope"`
}

type microFixtureShapeFillHeightSearchArtifact struct {
	ManifestPath   string                                 `json:"manifest_path,omitempty"`
	FixturePath    string                                 `json:"fixture_path,omitempty"`
	TargetCompared string                                 `json:"target_compared,omitempty"`
	Basis          string                                 `json:"basis"`
	Baseline       imageDiff                              `json:"baseline"`
	Candidates     []microFixtureShapeFillHeightCandidate `json:"candidates"`
}

type microFixtureShapeFillHeightCandidate struct {
	Name                          string           `json:"name"`
	FillColor                     string           `json:"fill_color"`
	HeightPixels                  int              `json:"height_pixels"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
}

type microFixtureShapeTextStrokeProfileArtifact struct {
	ManifestPath                  string                                     `json:"manifest_path,omitempty"`
	FixturePath                   string                                     `json:"fixture_path,omitempty"`
	TargetCompared                string                                     `json:"target_compared,omitempty"`
	Basis                         string                                     `json:"basis"`
	TextColor                     string                                     `json:"text_color"`
	FillColor                     string                                     `json:"fill_color"`
	TextTolerance                 int                                        `json:"text_tolerance"`
	EdgeBandPixels                int                                        `json:"edge_band_pixels"`
	Baseline                      imageDiff                                  `json:"baseline"`
	GotTextMask                   microFixtureShapeMaskProfile               `json:"got_text_mask"`
	ReferenceTextMask             microFixtureShapeMaskProfile               `json:"reference_text_mask"`
	ReferenceTopMinusGotTop       int                                        `json:"reference_top_minus_got_top,omitempty"`
	ReferenceCenterMinusGotCenter int                                        `json:"reference_center_minus_got_center,omitempty"`
	Edge                          microFixtureShapeEdgeProfile               `json:"edge"`
	TextLikeDifferentPixels       int                                        `json:"text_like_different_pixels,omitempty"`
	NonTextDifferentPixels        int                                        `json:"non_text_different_pixels,omitempty"`
	ShiftCandidates               []microFixtureShapeTextShiftCandidate      `json:"shift_candidates,omitempty"`
	ReconstructionCandidates      []microFixtureShapeReconstructionCandidate `json:"reconstruction_candidates,omitempty"`
	FontCandidates                []microFixtureShapeTextFontCandidate       `json:"font_candidates,omitempty"`
	CoverageCandidates            []microFixtureShapeTextCoverageCandidate   `json:"coverage_candidates,omitempty"`
	AnchorMetrics                 *microFixtureShapeTextAnchorMetrics        `json:"anchor_metrics,omitempty"`
	AnchorCandidates              []microFixtureShapeTextAnchorCandidate     `json:"anchor_candidates,omitempty"`
}

type microFixtureShapeMaskProfile struct {
	Pixels     int                     `json:"pixels"`
	Bounds     *imageDiffBounds        `json:"bounds,omitempty"`
	TopRows    []microFixtureAxisCount `json:"top_rows,omitempty"`
	TopColumns []microFixtureAxisCount `json:"top_columns,omitempty"`
}

type microFixtureShapeEdgeProfile struct {
	DifferentPixels  int                     `json:"different_pixels"`
	TopBandPixels    int                     `json:"top_band_pixels,omitempty"`
	BottomBandPixels int                     `json:"bottom_band_pixels,omitempty"`
	LeftBandPixels   int                     `json:"left_band_pixels,omitempty"`
	RightBandPixels  int                     `json:"right_band_pixels,omitempty"`
	TopRows          []microFixtureAxisCount `json:"top_rows,omitempty"`
	TopColumns       []microFixtureAxisCount `json:"top_columns,omitempty"`
}

type microFixtureShapeTextShiftCandidate struct {
	Name                          string           `json:"name"`
	ShiftY                        int              `json:"shift_y"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
}

type microFixtureShapeReconstructionCandidate struct {
	Name                          string           `json:"name"`
	ShiftY                        int              `json:"shift_y,omitempty"`
	EdgeBandPixels                int              `json:"edge_band_pixels,omitempty"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
}

type microFixtureShapeTextFontCandidate struct {
	Name                          string                       `json:"name"`
	FontFamily                    string                       `json:"font_family"`
	ResolvedFont                  string                       `json:"resolved_font,omitempty"`
	ShiftY                        int                          `json:"shift_y"`
	DifferentPixels               int                          `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds             `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64                        `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int                          `json:"max_channel_delta_8bit"`
	TextMask                      microFixtureShapeMaskProfile `json:"text_mask"`
}

type microFixtureShapeTextCoverageCandidate struct {
	Name                          string                       `json:"name"`
	ShiftY                        int                          `json:"shift_y,omitempty"`
	EdgeBandPixels                int                          `json:"edge_band_pixels,omitempty"`
	Mode                          string                       `json:"mode"`
	DifferentPixels               int                          `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds             `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64                        `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int                          `json:"max_channel_delta_8bit"`
	TextMask                      microFixtureShapeMaskProfile `json:"text_mask"`
}

type microFixtureShapeTextAnchorMetrics struct {
	DPI                 int                               `json:"dpi"`
	TextAnchor          string                            `json:"text_anchor,omitempty"`
	TextBounds          ObjectPixelBounds                 `json:"text_bounds"`
	FontFamily          string                            `json:"font_family,omitempty"`
	FontSize            int                               `json:"font_size,omitempty"`
	LineCount           int                               `json:"line_count"`
	CurrentAnchorHeight int                               `json:"current_anchor_height"`
	LineBoxAnchorHeight int                               `json:"line_box_anchor_height"`
	CurrentTop          int                               `json:"current_top"`
	LineBoxTop          int                               `json:"line_box_top"`
	LineBoxShiftY       int                               `json:"line_box_shift_y"`
	Lines               []microFixtureShapeTextLineMetric `json:"lines,omitempty"`
}

type microFixtureShapeTextLineMetric struct {
	Ascent      int  `json:"ascent"`
	Descent     int  `json:"descent"`
	Height      int  `json:"height"`
	SpaceBefore int  `json:"space_before,omitempty"`
	SpaceAfter  int  `json:"space_after,omitempty"`
	HasText     bool `json:"has_text,omitempty"`
}

type microFixtureShapeTextAnchorCandidate struct {
	Name                          string                       `json:"name"`
	ShiftY                        int                          `json:"shift_y"`
	DifferentPixels               int                          `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds             `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64                        `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int                          `json:"max_channel_delta_8bit"`
	TextMask                      microFixtureShapeMaskProfile `json:"text_mask"`
}

type microFixtureShapeResidualTextProfileArtifact struct {
	ManifestPath         string                                 `json:"manifest_path,omitempty"`
	FixturePath          string                                 `json:"fixture_path,omitempty"`
	TargetCompared       string                                 `json:"target_compared,omitempty"`
	Basis                string                                 `json:"basis"`
	NormalizedCandidate  microFixtureShapeFillHeightCandidate   `json:"normalized_candidate"`
	TextColor            string                                 `json:"text_color"`
	NormalizedTargetDiff imageDiff                              `json:"normalized_target_diff"`
	Residual             microFixtureShapeResidualTextProfile   `json:"residual"`
	ParsedTextCandidates []microFixtureShapeParsedTextCandidate `json:"parsed_text_candidates,omitempty"`
}

type microFixtureShapeParsedTextCandidate struct {
	Name                          string                       `json:"name"`
	TextBounds                    ObjectPixelBounds            `json:"text_bounds"`
	DifferentPixels               int                          `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds             `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64                        `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int                          `json:"max_channel_delta_8bit"`
	TextMask                      microFixtureShapeMaskProfile `json:"text_mask"`
}

type microFixtureShapeLuminanceColorSearchArtifact struct {
	ManifestPath      string                                     `json:"manifest_path,omitempty"`
	FixturePath       string                                     `json:"fixture_path,omitempty"`
	SchemeSlot        string                                     `json:"scheme_slot"`
	BaseColor         string                                     `json:"base_color"`
	LumMod            int64                                      `json:"lum_mod"`
	LumOff            int64                                      `json:"lum_off"`
	GotDominant       string                                     `json:"got_dominant"`
	ReferenceDominant string                                     `json:"reference_dominant"`
	Candidates        []microFixtureShapeLuminanceColorCandidate `json:"candidates"`
}

type microFixtureShapeLuminanceColorCandidate struct {
	Name                      string `json:"name"`
	InternalColor             string `json:"internal_color"`
	OutputColor               string `json:"output_color"`
	OutputDistanceToReference int    `json:"output_distance_to_reference"`
	OutputDistanceToGot       int    `json:"output_distance_to_got"`
}

type microFixtureTableStyleColorProfile struct {
	ManifestPath              string                              `json:"manifest_path,omitempty"`
	DeckInput                 string                              `json:"deck_input"`
	SlideNumber               int                                 `json:"slide_number"`
	CNvPrID                   string                              `json:"cnv_pr_id,omitempty"`
	CNvPrName                 string                              `json:"cnv_pr_name,omitempty"`
	SchemaAnchors             []string                            `json:"schema_anchors,omitempty"`
	SourceXMLPart             string                              `json:"source_xml_part"`
	SourceXMLPath             string                              `json:"source_xml_path"`
	TableStyleID              string                              `json:"table_style_id,omitempty"`
	TableStyleName            string                              `json:"table_style_name,omitempty"`
	FirstRow                  bool                                `json:"first_row"`
	BandRow                   bool                                `json:"band_row"`
	Samples                   []microFixtureTableStyleColorSample `json:"samples,omitempty"`
	TopGotColors              []microFixtureColorCount            `json:"top_got_colors,omitempty"`
	TopReferenceColors        []microFixtureColorCount            `json:"top_reference_colors,omitempty"`
	TopDifferentGotColors     []microFixtureColorCount            `json:"top_different_got_colors,omitempty"`
	TopDifferentReference     []microFixtureColorCount            `json:"top_different_reference_colors,omitempty"`
	TargetDifferentPixels     int                                 `json:"target_different_pixels"`
	ReferenceRGBDeltaSum8     int                                 `json:"reference_rgb_delta_sum_8bit"`
	ReferenceRGBAbsoluteDelta int                                 `json:"reference_rgb_absolute_delta_sum_8bit"`
}

type microFixtureTableStyleColorSample struct {
	Label             string   `json:"label"`
	RowIndex          int      `json:"row_index"`
	ColumnIndex       int      `json:"column_index"`
	RegionNames       []string `json:"region_names,omitempty"`
	HasFill           bool     `json:"has_fill"`
	FillColor         string   `json:"fill_color,omitempty"`
	DisplayP3Fill     string   `json:"display_p3_fill,omitempty"`
	HasTextColor      bool     `json:"has_text_color"`
	TextColor         string   `json:"text_color,omitempty"`
	DisplayP3Text     string   `json:"display_p3_text,omitempty"`
	BottomBorderLine  bool     `json:"bottom_border_line"`
	BottomBorder      string   `json:"bottom_border,omitempty"`
	DisplayP3Border   string   `json:"display_p3_border,omitempty"`
	BottomBorderWidth int64    `json:"bottom_border_width_emu,omitempty"`
}

type microFixtureShapeVectorBackendProfileArtifact struct {
	ManifestPath   string                                    `json:"manifest_path,omitempty"`
	FixturePath    string                                    `json:"fixture_path,omitempty"`
	TargetCompared string                                    `json:"target_compared,omitempty"`
	Basis          string                                    `json:"basis"`
	Baseline       imageDiff                                 `json:"baseline"`
	Candidates     []microFixtureShapeVectorBackendCandidate `json:"candidates,omitempty"`
}

type microFixtureShapeVectorBackendCandidate struct {
	Name                          string            `json:"name"`
	Backend                       string            `json:"backend"`
	Base                          string            `json:"base,omitempty"`
	Layers                        string            `json:"layers,omitempty"`
	Geometry                      string            `json:"geometry,omitempty"`
	ShapeBounds                   ObjectPixelBounds `json:"shape_bounds"`
	DifferentPixels               int               `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds  `json:"different_bounds,omitempty"`
	TotalAbsoluteChannelDelta8Bit int64             `json:"total_absolute_channel_delta_8bit"`
	MaxChannelDelta8Bit           int               `json:"max_channel_delta_8bit"`
	CandidatePath                 string            `json:"candidate_path,omitempty"`
}

type microFixtureShapeTextShapingProfileArtifact struct {
	ManifestPath          string                        `json:"manifest_path,omitempty"`
	FixturePath           string                        `json:"fixture_path,omitempty"`
	Basis                 string                        `json:"basis"`
	DPI                   int                           `json:"dpi"`
	TextBounds            ObjectPixelBounds             `json:"text_bounds"`
	LineCount             int                           `json:"line_count"`
	SegmentCount          int                           `json:"segment_count"`
	MaxAdvanceDeltaPixels int                           `json:"max_advance_delta_pixels"`
	Lines                 []microFixtureTextShapingLine `json:"lines,omitempty"`
}

type microFixtureTextShapingLine struct {
	Index    int                              `json:"index"`
	Segments []microFixtureTextShapingSegment `json:"segments,omitempty"`
}

type microFixtureTextShapingSegment struct {
	Text                string                         `json:"text"`
	FontFamily          string                         `json:"font_family,omitempty"`
	ResolvedFont        string                         `json:"resolved_font,omitempty"`
	FontSize            int                            `json:"font_size,omitempty"`
	DefaultedFontSize   bool                           `json:"defaulted_font_size,omitempty"`
	Bold                bool                           `json:"bold,omitempty"`
	Italic              bool                           `json:"italic,omitempty"`
	CurrentWidthPixels  int                            `json:"current_width_pixels"`
	ShapedAdvancePixels float64                        `json:"shaped_advance_pixels"`
	AdvanceDeltaPixels  float64                        `json:"advance_delta_pixels"`
	GlyphCount          int                            `json:"glyph_count"`
	Glyphs              []microFixtureTextShapingGlyph `json:"glyphs,omitempty"`
	Error               string                         `json:"error,omitempty"`
}

type microFixtureTextShapingGlyph struct {
	GlyphID        uint32  `json:"glyph_id"`
	TextIndex      int     `json:"text_index"`
	XAdvancePixels float64 `json:"x_advance_pixels"`
	XOffsetPixels  float64 `json:"x_offset_pixels,omitempty"`
	WidthPixels    float64 `json:"width_pixels,omitempty"`
	HeightPixels   float64 `json:"height_pixels,omitempty"`
}

type microFixtureShapeResidualTextProfile struct {
	DifferentPixels                     int                      `json:"different_pixels"`
	TextTolerance                       int                      `json:"text_tolerance"`
	FillTolerance                       int                      `json:"fill_tolerance"`
	ReferenceTextLikeDifferentPixels    int                      `json:"reference_text_like_different_pixels,omitempty"`
	GotTextLikeDifferentPixels          int                      `json:"got_text_like_different_pixels,omitempty"`
	EitherTextLikeDifferentPixels       int                      `json:"either_text_like_different_pixels,omitempty"`
	BothTextLikeDifferentPixels         int                      `json:"both_text_like_different_pixels,omitempty"`
	ReferenceFillLikeDifferentPixels    int                      `json:"reference_fill_like_different_pixels,omitempty"`
	GotFillLikeDifferentPixels          int                      `json:"got_fill_like_different_pixels,omitempty"`
	ReferenceWhiteLikeDifferentPixels   int                      `json:"reference_white_like_different_pixels,omitempty"`
	GotWhiteLikeDifferentPixels         int                      `json:"got_white_like_different_pixels,omitempty"`
	ReferenceOtherDifferentPixels       int                      `json:"reference_other_different_pixels,omitempty"`
	GotOtherDifferentPixels             int                      `json:"got_other_different_pixels,omitempty"`
	TopDifferentRows                    []microFixtureAxisCount  `json:"top_different_rows,omitempty"`
	TopDifferentColumns                 []microFixtureAxisCount  `json:"top_different_columns,omitempty"`
	TopDifferentGotColors               []microFixtureColorCount `json:"top_different_got_colors,omitempty"`
	TopDifferentReferenceColors         []microFixtureColorCount `json:"top_different_reference_colors,omitempty"`
	TopTextLikeRows                     []microFixtureAxisCount  `json:"top_text_like_rows,omitempty"`
	TopTextLikeColumns                  []microFixtureAxisCount  `json:"top_text_like_columns,omitempty"`
	TopReferenceMinusGotLumaBuckets     []microFixtureDeltaCount `json:"top_reference_minus_got_luma_buckets,omitempty"`
	TopReferenceMinusGotTextLumaBuckets []microFixtureDeltaCount `json:"top_reference_minus_got_text_luma_buckets,omitempty"`
}

func buildObjectAttributionArtifact(deckInput string, slideNumber int, gotPath string, referencePath string, fullDiff imageDiff, records []PaintedObject) (objectAttributionArtifact, error) {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return objectAttributionArtifact{}, err
	}
	reference, err := decodePNGFile(referencePath)
	if err != nil {
		return objectAttributionArtifact{}, err
	}
	attribution := objectAttributionArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		FullDiff:    fullDiff,
		Objects:     make([]objectFailureRecord, 0, len(records)),
	}
	previousCumulativeDiff := 0
	cumulativeByZOrder := make(map[int]int, len(records))
	for _, record := range records {
		overlap := diffPixelsInObjectBounds(got, reference, record)
		cumulativeDiff := 0
		cumulativeDelta := 0
		if record.ThroughArtifactPath != "" {
			if diff, err := comparePNG(record.ThroughArtifactPath, referencePath); err == nil {
				cumulativeDiff = diff.DifferentPixels
				cumulativeDelta = cumulativeDiff - previousCumulativeDiff
				cumulativeByZOrder[record.ZOrder] = cumulativeDiff
				probe := objectCumulativeProbe{
					ZOrder:                    record.ZOrder,
					CNvPrID:                   record.CNvPrID,
					CNvPrName:                 record.CNvPrName,
					Kind:                      record.Kind,
					CumulativeDifferentPixels: cumulativeDiff,
					CumulativeDiffDeltaPixels: cumulativeDelta,
					AbsoluteDeltaPixels:       absInt(cumulativeDelta),
					ThroughArtifactPath:       record.ThroughArtifactPath,
					SuspectedRendererGap:      suspectedRendererGap(record),
				}
				attribution.CumulativeProbes = append(attribution.CumulativeProbes, probe)
				if attribution.LargestCumulativeDelta == nil || probe.AbsoluteDeltaPixels > attribution.LargestCumulativeDelta.AbsoluteDeltaPixels {
					candidate := probe
					attribution.LargestCumulativeDelta = &candidate
				}
				previousCumulativeDiff = cumulativeDiff
			}
		}
		changedPixels := 0
		if record.ObjectArtifactPath != "" {
			changedPixels = nonTransparentPNGPixelCount(record.ObjectArtifactPath)
		}
		failure := objectFailureRecord{
			DeckInput:                   deckInput,
			SlideNumber:                 slideNumber,
			SlidePart:                   record.SlidePart,
			SourcePart:                  record.SourcePart,
			XMLPath:                     record.XMLPath,
			CNvPrID:                     record.CNvPrID,
			CNvPrName:                   record.CNvPrName,
			Kind:                        record.Kind,
			ZOrder:                      record.ZOrder,
			Bounds:                      record.Bounds,
			PixelBounds:                 record.PixelBounds,
			FractionalBounds:            record.FractionalBounds,
			OutputPixelBounds:           record.OutputPixelBounds,
			ResolvedStyle:               record.ResolvedStyle,
			OverlapDiffPixels:           overlap,
			ObjectArtifactChangedPixels: changedPixels,
			CumulativeDifferentPixels:   cumulativeDiff,
			CumulativeDiffDeltaPixels:   cumulativeDelta,
			SuspectedRendererGap:        suspectedRendererGap(record),
			BeforeArtifactPath:          record.BeforeArtifactPath,
			ObjectArtifactPath:          record.ObjectArtifactPath,
			ThroughArtifactPath:         record.ThroughArtifactPath,
		}
		failure.ObservedDiff = fmt.Sprintf("%d full-slide diff pixel(s) overlap this object's output bounds; object artifact paints %d pixel(s)", overlap, changedPixels)
		attribution.Objects = append(attribution.Objects, failure)
	}
	attribution.BinarySearch = binarySearchCumulativeObject(records, fullDiff, cumulativeByZOrder)
	sort.SliceStable(attribution.Objects, func(i, j int) bool {
		if attribution.Objects[i].OverlapDiffPixels != attribution.Objects[j].OverlapDiffPixels {
			return attribution.Objects[i].OverlapDiffPixels > attribution.Objects[j].OverlapDiffPixels
		}
		if absInt(attribution.Objects[i].CumulativeDiffDeltaPixels) != absInt(attribution.Objects[j].CumulativeDiffDeltaPixels) {
			return absInt(attribution.Objects[i].CumulativeDiffDeltaPixels) > absInt(attribution.Objects[j].CumulativeDiffDeltaPixels)
		}
		return attribution.Objects[i].ZOrder < attribution.Objects[j].ZOrder
	})
	return attribution, nil
}

func binarySearchCumulativeObject(records []PaintedObject, fullDiff imageDiff, cumulativeByZOrder map[int]int) objectBinarySearch {
	result := objectBinarySearch{
		Method:                   "binary search over cumulative through-X renders; predicate is cumulative different_pixels >= half of final full-slide different_pixels",
		ThresholdDifferentPixels: max(1, fullDiff.DifferentPixels/2),
	}
	if len(records) == 0 {
		return result
	}
	low, high := 0, len(records)-1
	target := 0
	for low <= high {
		mid := low + (high-low)/2
		zOrder := records[mid].ZOrder
		result.ProbeZOrders = append(result.ProbeZOrders, zOrder)
		if cumulativeByZOrder[zOrder] >= result.ThresholdDifferentPixels {
			target = zOrder
			high = mid - 1
		} else {
			low = mid + 1
		}
	}
	result.TargetZOrder = target
	return result
}

func diffPixelsInObjectBounds(got image.Image, reference image.Image, record PaintedObject) int {
	bounds, ok := objectDiffBounds(record)
	if !ok {
		return 0
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	rect := image.Rect(bounds.MinX, bounds.MinY, bounds.MaxX+1, bounds.MaxY+1).
		Intersect(image.Rect(0, 0, gotBounds.Dx(), gotBounds.Dy())).
		Intersect(image.Rect(0, 0, referenceBounds.Dx(), referenceBounds.Dy()))
	if rect.Empty() {
		return 0
	}
	count := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr != wr || gg != wg || gb != wb || ga != wa {
				count++
			}
		}
	}
	return count
}

func objectDiffBounds(record PaintedObject) (ObjectPixelBounds, bool) {
	if record.OutputPixelBounds != nil {
		return *record.OutputPixelBounds, true
	}
	if record.PixelBounds != (ObjectPixelBounds{}) {
		return record.PixelBounds, true
	}
	return ObjectPixelBounds{}, false
}

func nonTransparentPNGPixelCount(path string) int {
	img, err := decodePNGFile(path)
	if err != nil {
		return 0
	}
	bounds := img.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, alpha := img.At(x, y).RGBA()
			if alpha != 0 {
				count++
			}
		}
	}
	return count
}

func suspectedRendererGap(record PaintedObject) string {
	style := record.ResolvedStyle
	switch {
	case style.Table:
		return "table layout or inherited table text styling"
	case style.Text != "":
		return "text shaping, font metrics, paragraph layout, or text anchoring"
	case style.GradientFill:
		return "gradient fill geometry or color interpolation"
	case style.Shadow:
		return "shadow geometry, blur, alpha, or transform"
	case record.Kind == "pic":
		return "picture crop, resampling, color management, or media transform"
	case style.Geometry != "":
		return "shape geometry, fill, line, clipping, or antialiasing"
	default:
		return "unclassified renderer gap"
	}
}

func writeTopPictureMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	for _, object := range attribution.Objects {
		if object.Kind != "pic" || object.ResolvedStyle.EmbedID == "" || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
			continue
		}
		writePictureMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "")
		return
	}
}

func writeLargestCumulativePictureMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	if attribution.LargestCumulativeDelta == nil || attribution.LargestCumulativeDelta.Kind != "pic" {
		return
	}
	object, ok := findObjectFailureRecord(
		attribution.Objects,
		attribution.LargestCumulativeDelta.Kind,
		attribution.LargestCumulativeDelta.ZOrder,
		attribution.LargestCumulativeDelta.CNvPrID,
		attribution.LargestCumulativeDelta.CNvPrName,
	)
	if !ok || object.ResolvedStyle.EmbedID == "" || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
		return
	}
	writePictureMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "cumulative-picture")
}

func writePictureMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, object objectFailureRecord, attribution objectAttributionArtifact, prefix string) {
	t.Helper()
	label := fmt.Sprintf("%04d-%s-%s", object.ZOrder, object.CNvPrID, object.CNvPrName)
	if prefix != "" {
		label = fmt.Sprintf("%s-%s", prefix, label)
	}
	microDir := filepath.Join(slideDir, "micro-fixtures", sanitizeObjectArtifactName(label))
	if err := os.MkdirAll(microDir, 0o755); err != nil {
		t.Fatalf("create micro-fixture dir for %s slide %d: %v", deckInput, slideNumber, err)
	}
	fixturePath := filepath.Join(microDir, "fixture.pptx")
	sourceObjectXMLPath, sourceObjectSummary := writeMicroFixtureSourceObjectXML(t, deckInput, slideNumber, microDir, object)
	sourceImage, err := writePictureObjectFixture(deckInput, fixturePath, object)
	if err != nil {
		t.Fatalf("write picture micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	fixtureParts := readMicroFixturePackageParts(t, fixturePath)
	renderPath := filepath.Join(microDir, "got.png")
	fixtureObjectsPath := filepath.Join(microDir, "fixture-objects.json")
	if _, err := renderMicroFixtureWithObjectDebug(fixturePath, renderPath, fixtureObjectsPath, filepath.Join(microDir, "fixture-objects")); err != nil {
		t.Fatalf("render picture micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	gotCropPath := filepath.Join(microDir, "got-crop.png")
	referenceCropPath := filepath.Join(microDir, "reference-crop.png")
	if err := writeCroppedPNG(renderPath, gotCropPath, *object.OutputPixelBounds); err != nil {
		t.Fatalf("write micro-fixture got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeCroppedPNG(referencePath, referenceCropPath, *object.OutputPixelBounds); err != nil {
		t.Fatalf("write micro-fixture reference crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	diff, err := comparePNG(gotCropPath, referenceCropPath)
	if err != nil {
		t.Fatalf("compare micro-fixture crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(filepath.Join(microDir, "micro-diff.json"), realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write micro-fixture diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	visibleArtifacts := writeMicroFixtureVisibleArtifacts(t, deckInput, slideNumber, microDir, renderPath, referencePath, object, attribution)
	geometryArtifacts := writeMicroFixtureGeometryArtifacts(t, deckInput, slideNumber, microDir, renderPath, referencePath, object)
	sourceArtifacts := writeMicroFixtureSourceArtifacts(t, deckInput, slideNumber, microDir, referenceCropPath, visibleArtifacts, object)
	underpaints := microFixtureUnderpaints(object, attribution.Objects)
	targetScopePath, targetScope := writeMicroFixtureTargetScope(t, deckInput, slideNumber, microDir, gotCropPath, referenceCropPath, visibleArtifacts, object, underpaints)
	if err := writeJSONFile(filepath.Join(microDir, "manifest.json"), microFixtureManifest{
		DeckInput:                    deckInput,
		SlideNumber:                  slideNumber,
		Object:                       object,
		SpecFixture:                  specFixtureForObject(object),
		FixturePath:                  fixturePath,
		SourceObjectXMLPath:          sourceObjectXMLPath,
		SourceObjectSummary:          sourceObjectSummary,
		FixtureParts:                 fixtureParts,
		FixtureObjectsPath:           fixtureObjectsPath,
		SourceImage:                  sourceImage,
		Sampling:                     microFixtureSamplingForPicture(sourceImage, object),
		GotCropPath:                  gotCropPath,
		ReferenceCropPath:            referenceCropPath,
		DiffPath:                     filepath.Join(microDir, "micro-diff.json"),
		GotGeometryCropPath:          geometryArtifacts.gotPath,
		ReferenceGeometryCropPath:    geometryArtifacts.referencePath,
		GeometryDiffPath:             geometryArtifacts.diffPath,
		SourceBeforeCropPath:         sourceArtifacts.beforePath,
		SourceThroughCropPath:        sourceArtifacts.throughPath,
		SourceThroughDiffPath:        sourceArtifacts.throughDiffPath,
		SourceThroughVisibleCropPath: sourceArtifacts.throughVisiblePath,
		SourceThroughVisibleDiffPath: sourceArtifacts.throughVisibleDiffPath,
		GotVisibleCropPath:           visibleArtifacts.gotPath,
		ReferenceVisibleCropPath:     visibleArtifacts.referencePath,
		VisibleDiffPath:              visibleArtifacts.diffPath,
		OccludedBy:                   visibleArtifacts.occlusions,
		UnderpaintedBy:               underpaints,
		TargetScopePath:              targetScopePath,
		TargetScope:                  targetScope,
		Acceptance:                   microFixtureAcceptance(visibleArtifacts.occlusions),
	}); err != nil {
		t.Fatalf("write micro-fixture manifest for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
}

func writeTopShapeMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	for _, object := range attribution.Objects {
		if object.Kind != "sp" || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
			continue
		}
		writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "shape")
		writeUnderpaintShapeMicroFixtures(t, deckInput, slideNumber, slideDir, referencePath, object, attribution)
		return
	}
}

func writeLargestCumulativeShapeMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	if attribution.LargestCumulativeDelta == nil || attribution.LargestCumulativeDelta.Kind != "sp" {
		return
	}
	object, ok := findObjectFailureRecord(
		attribution.Objects,
		attribution.LargestCumulativeDelta.Kind,
		attribution.LargestCumulativeDelta.ZOrder,
		attribution.LargestCumulativeDelta.CNvPrID,
		attribution.LargestCumulativeDelta.CNvPrName,
	)
	if !ok || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
		return
	}
	writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "cumulative-shape")
	writeUnderpaintShapeMicroFixtures(t, deckInput, slideNumber, slideDir, referencePath, object, attribution)
}

func writeLargestCumulativeConnectorMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	if attribution.LargestCumulativeDelta == nil || attribution.LargestCumulativeDelta.Kind != "cxnSp" {
		return
	}
	object, ok := findObjectFailureRecord(
		attribution.Objects,
		attribution.LargestCumulativeDelta.Kind,
		attribution.LargestCumulativeDelta.ZOrder,
		attribution.LargestCumulativeDelta.CNvPrID,
		attribution.LargestCumulativeDelta.CNvPrName,
	)
	if !ok || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
		return
	}
	writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "cumulative-connector")
}

func writeTopTableMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	for _, object := range attribution.Objects {
		if object.Kind != "graphicFrame" || !object.ResolvedStyle.Table || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
			continue
		}
		writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "table")
		return
	}
}

func writeLargestCumulativeTableMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, attribution objectAttributionArtifact) {
	t.Helper()
	if attribution.LargestCumulativeDelta == nil || attribution.LargestCumulativeDelta.Kind != "graphicFrame" {
		return
	}
	object, ok := findObjectFailureRecord(
		attribution.Objects,
		attribution.LargestCumulativeDelta.Kind,
		attribution.LargestCumulativeDelta.ZOrder,
		attribution.LargestCumulativeDelta.CNvPrID,
		attribution.LargestCumulativeDelta.CNvPrName,
	)
	if !ok || !object.ResolvedStyle.Table || object.Bounds.CX <= 0 || object.Bounds.CY <= 0 || object.OutputPixelBounds == nil {
		return
	}
	writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, "cumulative-table")
}

func writeUnderpaintShapeMicroFixtures(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, object objectFailureRecord, attribution objectAttributionArtifact) {
	t.Helper()
	writeUnderpaintShapeMicroFixturesSeen(t, deckInput, slideNumber, slideDir, referencePath, object, attribution, map[string]bool{})
}

func writeUnderpaintShapeMicroFixturesSeen(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, object objectFailureRecord, attribution objectAttributionArtifact, seen map[string]bool) {
	t.Helper()
	for _, underpaint := range microFixtureUnderpaints(object, attribution.Objects) {
		if underpaint.Kind != "sp" {
			continue
		}
		underpaintObject, ok := findObjectFailureRecord(attribution.Objects, underpaint.Kind, underpaint.ZOrder, underpaint.CNvPrID, underpaint.CNvPrName)
		if !ok || underpaintObject.Bounds.CX <= 0 || underpaintObject.Bounds.CY <= 0 || underpaintObject.OutputPixelBounds == nil {
			continue
		}
		key := objectFailureRecordKey(underpaintObject)
		if seen[key] {
			continue
		}
		seen[key] = true
		writeShapeMicroFixture(t, deckInput, slideNumber, slideDir, referencePath, underpaintObject, attribution, "underpaint-shape")
		writeUnderpaintShapeMicroFixturesSeen(t, deckInput, slideNumber, slideDir, referencePath, underpaintObject, attribution, seen)
	}
}

func findObjectFailureRecord(objects []objectFailureRecord, kind string, zOrder int, id string, name string) (objectFailureRecord, bool) {
	for _, object := range objects {
		if object.Kind == kind && object.ZOrder == zOrder && object.CNvPrID == id && object.CNvPrName == name {
			return object, true
		}
	}
	return objectFailureRecord{}, false
}

func objectFailureRecordKey(object objectFailureRecord) string {
	return fmt.Sprintf("%s/%d/%s/%s/%s", object.SourcePart, object.ZOrder, object.Kind, object.CNvPrID, object.CNvPrName)
}

func specFixtureForObject(object objectFailureRecord) specFixtureManifest {
	spec := specFixtureManifest{
		SchemaAnchors:             schemaAnchorsForFixtureObject(object),
		SourceXMLPart:             object.SourcePart,
		SourceXMLPath:             object.XMLPath,
		ExpectedSemanticModel:     expectedSemanticModelForFixtureObject(object),
		ExpectedRenderPrimitive:   expectedRenderPrimitiveForFixtureObject(object),
		ExpectedUnsupportedRecord: expectedUnsupportedRecordsForFixtureObject(object),
	}
	return spec
}

func schemaAnchorsForFixtureObject(object objectFailureRecord) []string {
	switch object.Kind {
	case "pic":
		return []string{"pml.xsd:1245 CT_Picture", "dml-picture.xsd:14 CT_Picture", "dml-main.xsd:1502 CT_BlipFillProperties", "dml-main.xsd:2223 CT_ShapeProperties"}
	case "cxnSp":
		return []string{"pml.xsd:1228 CT_Connector", "dml-main.xsd:2223 CT_ShapeProperties", "dml-main.xsd:2206 CT_LineProperties"}
	case "graphicFrame":
		if object.ResolvedStyle.Table {
			return []string{"pml.xsd:1263 CT_GraphicalObjectFrame", "dml-main.xsd:842 CT_GraphicalObjectData", "dml-main.xsd:2423 CT_Table", "dml-main.xsd:2386 CT_TableCell", "dml-main.xsd:2347 CT_TableCellProperties"}
		}
		return []string{"pml.xsd:1263 CT_GraphicalObjectFrame", "dml-main.xsd:842 CT_GraphicalObjectData"}
	case "sp":
		return []string{"pml.xsd:1209 CT_Shape", "dml-main.xsd:2223 CT_ShapeProperties", "dml-main.xsd:2206 CT_LineProperties"}
	default:
		return []string{"pml.xsd:1282 CT_GroupShape"}
	}
}

func expectedSemanticModelForFixtureObject(object objectFailureRecord) string {
	switch object.Kind {
	case "pic":
		return "source PresentationML picture with resolved blip relationship, transform, crop, geometry, line, effects, and unsupported blip/effect records"
	case "cxnSp":
		return "source PresentationML connector with resolved transform, geometry, line properties, style references, and unsupported marker/line records"
	case "graphicFrame":
		if object.ResolvedStyle.Table {
			return "source PresentationML graphic frame with resolved DrawingML table grid, rows, cells, spans, styles, text, fills, borders, and unsupported table records"
		}
		return "source PresentationML graphic frame with resolved graphical object payload and unsupported records"
	case "sp":
		return "source PresentationML shape with resolved transform, geometry, fill, line, text body, effects, style references, and unsupported records"
	default:
		return "source PresentationML object with provenance and explicit unsupported records"
	}
}

func expectedRenderPrimitiveForFixtureObject(object objectFailureRecord) string {
	switch object.Kind {
	case "pic":
		return "picture/media primitive with source relationship, media part, crop, transform, sampling target, masks/effects, and unsupported records"
	case "cxnSp":
		return "connector/vector primitive with geometry, stroke, marker, transform, clipping, and unsupported records"
	case "graphicFrame":
		if object.ResolvedStyle.Table {
			return "table primitive with frame transform, grid, rows, cells, spans, style regions, fills, borders, text layout inputs, and unsupported records"
		}
		return "graphic-frame primitive preserving graphical object provenance and unsupported records"
	case "sp":
		if object.ResolvedStyle.Text != "" {
			return "shape/text primitive with geometry, fill, stroke, text layout inputs, effects, clipping, and unsupported records"
		}
		return "shape/vector primitive with geometry, fill, stroke, effects, clipping, and unsupported records"
	default:
		return "unsupported render primitive preserving source provenance"
	}
}

func expectedUnsupportedRecordsForFixtureObject(object objectFailureRecord) []string {
	var records []string
	for _, item := range object.ResolvedStyle.CustomPathUnsupported {
		if item == "" {
			continue
		}
		records = append(records, item)
	}
	for _, item := range object.ResolvedStyle.ImageUnsupported {
		if item == "" {
			continue
		}
		records = append(records, item)
	}
	for _, item := range object.ResolvedStyle.EffectUnsupported {
		if item == "" {
			continue
		}
		records = append(records, item)
	}
	for _, item := range object.ResolvedStyle.TableUnsupported {
		if item == "" {
			continue
		}
		records = append(records, item)
	}
	sort.Strings(records)
	return records
}

func TestExpectedUnsupportedRecordsIgnoreSupportedImageMetadata(t *testing.T) {
	object := objectFailureRecord{
		ResolvedStyle: ObjectStyleSummary{
			ImageEffects:          []string{"fillMode=stretch", "rotWithShape=false", "alphaModFix=50000"},
			ImageUnsupported:      []string{"blip effect alphaMod was not rendered"},
			EffectUnsupported:     []string{"effectDag node blend was not rendered"},
			CustomPathUnsupported: []string{"custom geometry command close was not rendered"},
			TableUnsupported:      []string{"uses effects that were not rendered"},
		},
	}

	got := expectedUnsupportedRecordsForFixtureObject(object)
	want := []string{
		"blip effect alphaMod was not rendered",
		"custom geometry command close was not rendered",
		"effectDag node blend was not rendered",
		"uses effects that were not rendered",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected expected unsupported records:\ngot  %#v\nwant %#v", got, want)
	}
}

func writeShapeMicroFixture(t *testing.T, deckInput string, slideNumber int, slideDir string, referencePath string, object objectFailureRecord, attribution objectAttributionArtifact, prefix string) {
	t.Helper()
	microDir := filepath.Join(slideDir, "micro-fixtures", sanitizeObjectArtifactName(fmt.Sprintf("%s-%04d-%s-%s", prefix, object.ZOrder, object.CNvPrID, object.CNvPrName)))
	if err := os.MkdirAll(microDir, 0o755); err != nil {
		t.Fatalf("create shape micro-fixture dir for %s slide %d: %v", deckInput, slideNumber, err)
	}
	fixturePath := filepath.Join(microDir, "fixture.pptx")
	sourceObjectXMLPath, sourceObjectSummary := writeMicroFixtureSourceObjectXML(t, deckInput, slideNumber, microDir, object)
	if err := writeShapeObjectFixture(deckInput, fixturePath, object); err != nil {
		t.Fatalf("write shape micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	fixtureParts := readMicroFixturePackageParts(t, fixturePath)
	renderPath := filepath.Join(microDir, "got.png")
	fixtureObjectsPath := filepath.Join(microDir, "fixture-objects.json")
	if _, err := renderMicroFixtureWithObjectDebug(fixturePath, renderPath, fixtureObjectsPath, filepath.Join(microDir, "fixture-objects")); err != nil {
		t.Fatalf("render shape micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	gotCropPath := filepath.Join(microDir, "got-crop.png")
	referenceCropPath := filepath.Join(microDir, "reference-crop.png")
	if err := writeCroppedPNG(renderPath, gotCropPath, *object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape micro-fixture got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeCroppedPNG(referencePath, referenceCropPath, *object.OutputPixelBounds); err != nil {
		t.Fatalf("write shape micro-fixture reference crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	diff, err := comparePNG(gotCropPath, referenceCropPath)
	if err != nil {
		t.Fatalf("compare shape micro-fixture crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(filepath.Join(microDir, "micro-diff.json"), realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write shape micro-fixture diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	visibleArtifacts := writeMicroFixtureVisibleArtifacts(t, deckInput, slideNumber, microDir, renderPath, referencePath, object, attribution)
	geometryArtifacts := writeMicroFixtureGeometryArtifacts(t, deckInput, slideNumber, microDir, renderPath, referencePath, object)
	sourceArtifacts := writeMicroFixtureSourceArtifacts(t, deckInput, slideNumber, microDir, referenceCropPath, visibleArtifacts, object)
	underpaints := microFixtureUnderpaints(object, attribution.Objects)
	targetScopePath, targetScope := writeMicroFixtureTargetScope(t, deckInput, slideNumber, microDir, gotCropPath, referenceCropPath, visibleArtifacts, object, underpaints)
	underpaintChainArtifacts := writeMicroFixtureUnderpaintChainArtifacts(t, deckInput, slideNumber, microDir, referenceCropPath, visibleArtifacts, object, underpaints, attribution)
	nonUnderpaintArtifacts := writeMicroFixtureNonUnderpaintArtifacts(t, deckInput, slideNumber, microDir, gotCropPath, referenceCropPath, visibleArtifacts, object, underpaints)
	shadowAlphaArtifacts := writeMicroFixtureShadowAlphaArtifacts(t, deckInput, slideNumber, microDir, gotCropPath, referenceCropPath, visibleArtifacts, sourceArtifacts, object)
	shadowRenderSummary := writeMicroFixtureShadowRenderSummary(t, deckInput, renderPath, object)
	if err := writeJSONFile(filepath.Join(microDir, "manifest.json"), microFixtureManifest{
		DeckInput:                         deckInput,
		SlideNumber:                       slideNumber,
		Object:                            object,
		SpecFixture:                       specFixtureForObject(object),
		FixturePath:                       fixturePath,
		SourceObjectXMLPath:               sourceObjectXMLPath,
		SourceObjectSummary:               sourceObjectSummary,
		FixtureParts:                      fixtureParts,
		FixtureObjectsPath:                fixtureObjectsPath,
		GotCropPath:                       gotCropPath,
		ReferenceCropPath:                 referenceCropPath,
		DiffPath:                          filepath.Join(microDir, "micro-diff.json"),
		GotGeometryCropPath:               geometryArtifacts.gotPath,
		ReferenceGeometryCropPath:         geometryArtifacts.referencePath,
		GeometryDiffPath:                  geometryArtifacts.diffPath,
		SourceBeforeCropPath:              sourceArtifacts.beforePath,
		SourceThroughCropPath:             sourceArtifacts.throughPath,
		SourceThroughDiffPath:             sourceArtifacts.throughDiffPath,
		SourceThroughVisibleCropPath:      sourceArtifacts.throughVisiblePath,
		SourceThroughVisibleDiffPath:      sourceArtifacts.throughVisibleDiffPath,
		GotVisibleCropPath:                visibleArtifacts.gotPath,
		ReferenceVisibleCropPath:          visibleArtifacts.referencePath,
		VisibleDiffPath:                   visibleArtifacts.diffPath,
		UnderpaintChainFixturePath:        underpaintChainArtifacts.fixturePath,
		UnderpaintChainGotCropPath:        underpaintChainArtifacts.gotCropPath,
		UnderpaintChainDiffPath:           underpaintChainArtifacts.diffPath,
		UnderpaintChainGotVisibleCropPath: underpaintChainArtifacts.gotVisiblePath,
		UnderpaintChainVisibleDiffPath:    underpaintChainArtifacts.visibleDiffPath,
		UnderpaintChainTargetScopePath:    underpaintChainArtifacts.targetScopePath,
		UnderpaintChainTargetScope:        underpaintChainArtifacts.targetScope,
		UnderpaintChainSummary:            microFixtureUnderpaintChainSummaryForScopes(targetScope, underpaintChainArtifacts.targetScope),
		NonUnderpaintGotCropPath:          nonUnderpaintArtifacts.gotPath,
		NonUnderpaintReferenceCropPath:    nonUnderpaintArtifacts.referencePath,
		NonUnderpaintDiffPath:             nonUnderpaintArtifacts.diffPath,
		NonUnderpaintTargetScope:          nonUnderpaintArtifacts.targetScope,
		ShadowAlphaScopePath:              shadowAlphaArtifacts.scopePath,
		ShadowAlphaCorrectionHeatmapPath:  shadowAlphaArtifacts.heatmapPath,
		ShadowAlphaScope:                  shadowAlphaArtifacts.scope,
		ShadowRenderSummary:               shadowRenderSummary,
		OccludedBy:                        visibleArtifacts.occlusions,
		UnderpaintedBy:                    underpaints,
		TargetScopePath:                   targetScopePath,
		TargetScope:                       targetScope,
		Acceptance:                        microFixtureAcceptance(visibleArtifacts.occlusions),
	}); err != nil {
		t.Fatalf("write shape micro-fixture manifest for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
}

type microFixtureVisibleArtifacts struct {
	gotPath       string
	referencePath string
	diffPath      string
	occlusions    []microFixtureOcclusion
}

type microFixtureGeometryArtifacts struct {
	gotPath       string
	referencePath string
	diffPath      string
}

type microFixtureSourceArtifacts struct {
	beforePath             string
	throughPath            string
	throughDiffPath        string
	throughVisiblePath     string
	throughVisibleDiffPath string
}

type microFixtureUnderpaintChainArtifacts struct {
	fixturePath     string
	gotCropPath     string
	diffPath        string
	gotVisiblePath  string
	visibleDiffPath string
	targetScopePath string
	targetScope     microFixtureTargetScope
}

type microFixtureNonUnderpaintArtifacts struct {
	gotPath       string
	referencePath string
	diffPath      string
	targetScope   microFixtureTargetScope
}

type microFixtureShadowAlphaArtifacts struct {
	scopePath   string
	heatmapPath string
	scope       microFixtureShadowAlphaScope
}

func writeMicroFixtureSourceObjectXML(t *testing.T, deckInput string, slideNumber int, microDir string, object objectFailureRecord) (string, *microFixtureSourceObjectSummary) {
	t.Helper()
	deckPath := realWorldDeckPath(deckInput)
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatalf("open deck for source object XML %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	sourceData, ok := pkg.Parts[object.SourcePart]
	if !ok {
		t.Fatalf("source part %s not found for source object XML %s slide %d object %s", object.SourcePart, deckInput, slideNumber, object.CNvPrID)
	}
	rawObject, err := extractRawObjectXMLForRecord(sourceData, object)
	if err != nil {
		t.Fatalf("extract source object XML for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	path := filepath.Join(microDir, "source-object.xml")
	if err := os.WriteFile(path, []byte(rawObject), 0o644); err != nil {
		t.Fatalf("write source object XML for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	summary, err := microFixtureSourceObjectSummaryFromXML(rawObject, object.Kind)
	if err != nil {
		t.Fatalf("parse source object summary for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	return path, summary
}

func microFixtureSourceObjectSummaryFromXML(rawObject string, kind string) (*microFixtureSourceObjectSummary, error) {
	wrapped := `<root xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` + rawObject + `</root>`
	root, err := parseXMLNode([]byte(wrapped))
	if err != nil {
		return nil, err
	}
	summary := &microFixtureSourceObjectSummary{Kind: kind}
	if cNvPr := firstDescendant(root, "cNvPr"); cNvPr != nil {
		summary.CNvPrID = attrValue(cNvPr.Attrs, "id")
		summary.CNvPrName = attrValue(cNvPr.Attrs, "name")
	}
	if xfrm := firstDescendant(root, "xfrm"); xfrm != nil {
		if off := firstChild(xfrm, "off"); off != nil {
			summary.Transform.X = parseIntAttr(off.Attrs, "x")
			summary.Transform.Y = parseIntAttr(off.Attrs, "y")
		}
		if ext := firstChild(xfrm, "ext"); ext != nil {
			summary.Transform.CX = parseIntAttr(ext.Attrs, "cx")
			summary.Transform.CY = parseIntAttr(ext.Attrs, "cy")
		}
	}
	if pathNode := firstDescendant(root, "path"); pathNode != nil {
		customPath := &microFixtureSourceCustomPath{
			Width:  parseIntAttr(pathNode.Attrs, "w"),
			Height: parseIntAttr(pathNode.Attrs, "h"),
		}
		for _, command := range pathNode.Children {
			switch command.Name {
			case "moveTo", "lnTo", "cubicBezTo", "quadBezTo":
				for _, pt := range childrenByName(command, "pt") {
					customPath.Points = append(customPath.Points, microFixtureSourcePathPoint{
						Command: command.Name,
						X:       parseIntAttr(pt.Attrs, "x"),
						Y:       parseIntAttr(pt.Attrs, "y"),
					})
				}
			}
		}
		summary.CustomPath = customPath
	}
	if shadow := firstDescendant(root, "outerShdw"); shadow != nil {
		summary.Shadow = &microFixtureSourceOuterShadow{
			BlurRadius:      parseIntAttr(shadow.Attrs, "blurRad"),
			Distance:        parseIntAttr(shadow.Attrs, "dist"),
			Direction:       parseIntAttr(shadow.Attrs, "dir"),
			Alignment:       attrValue(shadow.Attrs, "algn"),
			RotateWithShape: attrValue(shadow.Attrs, "rotWithShape"),
		}
	}
	return summary, nil
}

func readMicroFixturePackageParts(t *testing.T, fixturePath string) []microFixturePackagePart {
	t.Helper()
	parts, err := microFixturePackageParts(fixturePath)
	if err != nil {
		t.Fatalf("read micro-fixture package parts for %s: %v", fixturePath, err)
	}
	return parts
}

func microFixturePackageParts(fixturePath string) ([]microFixturePackagePart, error) {
	reader, err := zip.OpenReader(fixturePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	parts := make([]microFixturePackagePart, 0, len(reader.File))
	for _, file := range reader.File {
		handle, err := file.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(handle)
		closeErr := handle.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		sum := sha256.Sum256(data)
		parts = append(parts, microFixturePackagePart{
			Name:   file.Name,
			Size:   int64(len(data)),
			SHA256: fmt.Sprintf("%x", sum),
			Reason: microFixturePackagePartReason(file.Name),
		})
	}
	sort.Slice(parts, func(i, j int) bool {
		return parts[i].Name < parts[j].Name
	})
	return parts, nil
}

func microFixturePackagePartReason(name string) string {
	switch {
	case name == "[Content_Types].xml":
		return "package content type declarations required by OPC"
	case name == "_rels/.rels":
		return "package root relationship to the presentation part"
	case name == "ppt/presentation.xml":
		return "minimal presentation part preserving slide size"
	case name == "ppt/_rels/presentation.xml.rels":
		return "presentation relationship to the single fixture slide"
	case name == "ppt/slides/slide1.xml":
		return "single extracted fixture slide with background and target object"
	case name == "ppt/slides/_rels/slide1.xml.rels":
		return "fixture slide relationships to required layout or media"
	case strings.HasPrefix(name, "ppt/slideLayouts/_rels/"):
		return "layout relationship to required slide master"
	case strings.HasPrefix(name, "ppt/slideLayouts/"):
		return "stripped source slide layout dependency"
	case strings.HasPrefix(name, "ppt/slideMasters/_rels/"):
		return "master relationship to required theme"
	case strings.HasPrefix(name, "ppt/slideMasters/"):
		return "stripped source slide master dependency"
	case strings.HasPrefix(name, "ppt/theme/"):
		return "theme dependency for scheme colors and inherited styles"
	case strings.HasPrefix(name, "ppt/media/"):
		return "media dependency for extracted picture object"
	default:
		return "fixture package dependency"
	}
}

func summarizeMicroFixtureTargetOwnership(root string) (microFixtureTargetOwnershipSummary, error) {
	summary := microFixtureTargetOwnershipSummary{Root: root}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "manifest.json" {
			return nil
		}
		summary.TotalManifests++
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest microFixtureManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}
		record := microFixtureOwnershipRecord{
			ManifestPath:                     path,
			DeckInput:                        manifest.DeckInput,
			SlideNumber:                      manifest.SlideNumber,
			Kind:                             manifest.Object.Kind,
			CNvPrID:                          manifest.Object.CNvPrID,
			CNvPrName:                        manifest.Object.CNvPrName,
			DifferentPixels:                  manifest.TargetScope.DifferentPixels,
			DifferentPixelsInsideObjectMask:  manifest.TargetScope.DifferentPixelsInsideObjectMask,
			DifferentPixelsOutsideObjectMask: manifest.TargetScope.DifferentPixelsOutsideObjectMask,
			PartialAlphaDifferentPixels:      manifest.TargetScope.DifferentPixelsInsidePartialAlphaObjectMask,
			PartialAlphaOverUnderpaintPixels: manifest.TargetScope.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint,
			NonUnderpaintDifferentPixels:     manifest.NonUnderpaintTargetScope.DifferentPixels,
			Warning:                          manifest.TargetScope.Warning,
		}
		if record.DifferentPixels == 0 && manifest.TargetScope.ComparedPixels == 0 {
			summary.UnscopedManifests = append(summary.UnscopedManifests, path)
			return nil
		}
		summary.ManifestsWithTargetScope++
		if record.DifferentPixels == 0 {
			return nil
		}
		if record.DifferentPixelsOutsideObjectMask > 0 {
			summary.ContaminatedFailures = append(summary.ContaminatedFailures, record)
		}
		if record.PartialAlphaOverUnderpaintPixels > 0 {
			summary.PartialUnderpaintFailures = append(summary.PartialUnderpaintFailures, record)
		}
		if isCleanMicroFixtureOwnershipFailure(record) {
			summary.CleanFailures = append(summary.CleanFailures, record)
		}
		return nil
	})
	if err != nil {
		return microFixtureTargetOwnershipSummary{}, err
	}
	sortMicroFixtureOwnershipRecords(summary.CleanFailures)
	sortMicroFixtureOwnershipRecords(summary.ContaminatedFailures)
	sortMicroFixtureOwnershipRecords(summary.PartialUnderpaintFailures)
	sort.Strings(summary.UnscopedManifests)
	return summary, nil
}

func buildRendererProductionFailureScoreboard(root string) (rendererProductionFailureScoreboard, error) {
	scoreboard := rendererProductionFailureScoreboard{
		Root:  root,
		Basis: "object overlap groups summarize attributed object bounds and can exceed full-slide pixels because objects overlap; clean fixture groups summarize isolated object-crop failures only",
	}
	primitiveGroups := map[string]*rendererPrimitiveScore{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != "object-attribution.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var attribution objectAttributionArtifact
		if err := json.Unmarshal(data, &attribution); err != nil {
			return err
		}
		scoreboard.AttributionArtifactCount++
		scoreboard.SlideCount++
		scoreboard.TotalSlideDifferentPixels += attribution.FullDiff.DifferentPixels
		scoreboard.TopSlides = append(scoreboard.TopSlides, rendererSlideScore{
			DeckInput:       attribution.DeckInput,
			SlideNumber:     attribution.SlideNumber,
			DifferentPixels: attribution.FullDiff.DifferentPixels,
		})
		for _, object := range attribution.Objects {
			key := object.Kind + "\x00" + object.SuspectedRendererGap
			group := primitiveGroups[key]
			if group == nil {
				group = &rendererPrimitiveScore{Kind: object.Kind, SuspectedGap: object.SuspectedRendererGap}
				primitiveGroups[key] = group
			}
			group.ObjectCount++
			group.OverlapDiffPixels += object.OverlapDiffPixels
		}
		return nil
	})
	if err != nil {
		return rendererProductionFailureScoreboard{}, err
	}
	for _, group := range primitiveGroups {
		scoreboard.ObjectOverlapByPrimitive = append(scoreboard.ObjectOverlapByPrimitive, *group)
	}
	sort.Slice(scoreboard.ObjectOverlapByPrimitive, func(i int, j int) bool {
		if scoreboard.ObjectOverlapByPrimitive[i].OverlapDiffPixels != scoreboard.ObjectOverlapByPrimitive[j].OverlapDiffPixels {
			return scoreboard.ObjectOverlapByPrimitive[i].OverlapDiffPixels > scoreboard.ObjectOverlapByPrimitive[j].OverlapDiffPixels
		}
		if scoreboard.ObjectOverlapByPrimitive[i].ObjectCount != scoreboard.ObjectOverlapByPrimitive[j].ObjectCount {
			return scoreboard.ObjectOverlapByPrimitive[i].ObjectCount > scoreboard.ObjectOverlapByPrimitive[j].ObjectCount
		}
		return scoreboard.ObjectOverlapByPrimitive[i].SuspectedGap < scoreboard.ObjectOverlapByPrimitive[j].SuspectedGap
	})
	sort.Slice(scoreboard.TopSlides, func(i int, j int) bool {
		if scoreboard.TopSlides[i].DifferentPixels != scoreboard.TopSlides[j].DifferentPixels {
			return scoreboard.TopSlides[i].DifferentPixels > scoreboard.TopSlides[j].DifferentPixels
		}
		if scoreboard.TopSlides[i].DeckInput != scoreboard.TopSlides[j].DeckInput {
			return scoreboard.TopSlides[i].DeckInput < scoreboard.TopSlides[j].DeckInput
		}
		return scoreboard.TopSlides[i].SlideNumber < scoreboard.TopSlides[j].SlideNumber
	})
	if len(scoreboard.TopSlides) > 25 {
		scoreboard.TopSlides = scoreboard.TopSlides[:25]
	}

	ownershipPath := filepath.Join(root, "micro-fixture-ownership-summary.json")
	if data, err := os.ReadFile(ownershipPath); err == nil {
		var ownership microFixtureTargetOwnershipSummary
		if err := json.Unmarshal(data, &ownership); err != nil {
			return rendererProductionFailureScoreboard{}, err
		}
		scoreboard.OwnershipSummaryPath = ownershipPath
		scoreboard.CleanFixtureFailureCount = len(ownership.CleanFailures)
		cleanByKind := map[string]*rendererCleanFixtureScore{}
		cleanByName := map[string]*rendererCleanFixtureNameScore{}
		scoreboard.TopCleanFailures = append(scoreboard.TopCleanFailures, ownership.CleanFailures...)
		for _, record := range ownership.CleanFailures {
			kindGroup := cleanByKind[record.Kind]
			if kindGroup == nil {
				kindGroup = &rendererCleanFixtureScore{Kind: record.Kind}
				cleanByKind[record.Kind] = kindGroup
			}
			kindGroup.FailureCount++
			kindGroup.DifferentPixels += record.DifferentPixels

			nameKey := record.Kind + "\x00" + record.CNvPrName
			nameGroup := cleanByName[nameKey]
			if nameGroup == nil {
				nameGroup = &rendererCleanFixtureNameScore{Kind: record.Kind, CNvPrName: record.CNvPrName}
				cleanByName[nameKey] = nameGroup
			}
			nameGroup.FailureCount++
			nameGroup.DifferentPixels += record.DifferentPixels
		}
		for _, group := range cleanByKind {
			scoreboard.CleanFixturesByKind = append(scoreboard.CleanFixturesByKind, *group)
		}
		for _, group := range cleanByName {
			scoreboard.CleanFixturesByObjectName = append(scoreboard.CleanFixturesByObjectName, *group)
		}
		sort.Slice(scoreboard.CleanFixturesByKind, func(i int, j int) bool {
			if scoreboard.CleanFixturesByKind[i].DifferentPixels != scoreboard.CleanFixturesByKind[j].DifferentPixels {
				return scoreboard.CleanFixturesByKind[i].DifferentPixels > scoreboard.CleanFixturesByKind[j].DifferentPixels
			}
			return scoreboard.CleanFixturesByKind[i].Kind < scoreboard.CleanFixturesByKind[j].Kind
		})
		sort.Slice(scoreboard.CleanFixturesByObjectName, func(i int, j int) bool {
			if scoreboard.CleanFixturesByObjectName[i].DifferentPixels != scoreboard.CleanFixturesByObjectName[j].DifferentPixels {
				return scoreboard.CleanFixturesByObjectName[i].DifferentPixels > scoreboard.CleanFixturesByObjectName[j].DifferentPixels
			}
			if scoreboard.CleanFixturesByObjectName[i].FailureCount != scoreboard.CleanFixturesByObjectName[j].FailureCount {
				return scoreboard.CleanFixturesByObjectName[i].FailureCount > scoreboard.CleanFixturesByObjectName[j].FailureCount
			}
			return scoreboard.CleanFixturesByObjectName[i].CNvPrName < scoreboard.CleanFixturesByObjectName[j].CNvPrName
		})
		sort.Slice(scoreboard.TopCleanFailures, func(i int, j int) bool {
			if scoreboard.TopCleanFailures[i].DifferentPixels != scoreboard.TopCleanFailures[j].DifferentPixels {
				return scoreboard.TopCleanFailures[i].DifferentPixels > scoreboard.TopCleanFailures[j].DifferentPixels
			}
			if scoreboard.TopCleanFailures[i].DeckInput != scoreboard.TopCleanFailures[j].DeckInput {
				return scoreboard.TopCleanFailures[i].DeckInput < scoreboard.TopCleanFailures[j].DeckInput
			}
			return scoreboard.TopCleanFailures[i].SlideNumber < scoreboard.TopCleanFailures[j].SlideNumber
		})
		if len(scoreboard.CleanFixturesByObjectName) > 25 {
			scoreboard.CleanFixturesByObjectName = scoreboard.CleanFixturesByObjectName[:25]
		}
		if len(scoreboard.TopCleanFailures) > 25 {
			scoreboard.TopCleanFailures = scoreboard.TopCleanFailures[:25]
		}
	} else if !os.IsNotExist(err) {
		return rendererProductionFailureScoreboard{}, err
	}
	return scoreboard, nil
}

func isCleanMicroFixtureOwnershipFailure(record microFixtureOwnershipRecord) bool {
	return record.DifferentPixelsInsideObjectMask == record.DifferentPixels &&
		record.DifferentPixelsOutsideObjectMask == 0 &&
		record.PartialAlphaOverUnderpaintPixels == 0
}

func sortMicroFixtureOwnershipRecords(records []microFixtureOwnershipRecord) {
	sort.Slice(records, func(i int, j int) bool {
		if records[i].DifferentPixelsInsideObjectMask != records[j].DifferentPixelsInsideObjectMask {
			return records[i].DifferentPixelsInsideObjectMask < records[j].DifferentPixelsInsideObjectMask
		}
		if records[i].DifferentPixels != records[j].DifferentPixels {
			return records[i].DifferentPixels < records[j].DifferentPixels
		}
		if records[i].DeckInput != records[j].DeckInput {
			return records[i].DeckInput < records[j].DeckInput
		}
		if records[i].SlideNumber != records[j].SlideNumber {
			return records[i].SlideNumber < records[j].SlideNumber
		}
		return records[i].CNvPrName < records[j].CNvPrName
	})
}

func microFixtureUnderpaintChainSummaryForScopes(objectOnly microFixtureTargetScope, chain microFixtureTargetScope) microFixtureUnderpaintChainSummary {
	if chain.DifferentPixels == 0 && chain.ComparedPixels == 0 && chain.ObjectMaskPixels == 0 {
		return microFixtureUnderpaintChainSummary{}
	}
	return microFixtureUnderpaintChainSummary{
		ObjectOnlyDifferentPixels:                         objectOnly.DifferentPixels,
		ChainDifferentPixels:                              chain.DifferentPixels,
		DifferentPixelsDelta:                              chain.DifferentPixels - objectOnly.DifferentPixels,
		ObjectOnlyUnderpaintedPartialAlphaDifferentPixels: objectOnly.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint,
		ChainUnderpaintedPartialAlphaDifferentPixels:      chain.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint,
		UnderpaintedPartialAlphaDifferentPixelsDelta:      chain.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint - objectOnly.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint,
		ObjectOnlyPlainPartialAlphaDifferentPixels:        objectOnly.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint,
		ChainPlainPartialAlphaDifferentPixels:             chain.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint,
		PlainPartialAlphaDifferentPixelsDelta:             chain.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint - objectOnly.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint,
		ChainReferenceRGBDeltaSum8:                        chain.ReferenceRGBDeltaSum8,
		ChainReferenceRGBAbsoluteDeltaSum8:                chain.ReferenceRGBAbsoluteDeltaSum8,
	}
}

func renderMicroFixtureWithObjectDebug(fixturePath string, renderPath string, recordsPath string, artifactDir string) (model.CommandResult, error) {
	debug := &ObjectDebugOptions{ArtifactDir: artifactDir}
	result, err := Render(context.Background(), fixturePath, Options{
		SlideNumber: 1,
		OutputPath:  renderPath,
		ObjectDebug: debug,
	})
	if err != nil {
		return result, err
	}
	if recordsPath != "" {
		if err := writeJSONFile(recordsPath, debug.Records); err != nil {
			return result, err
		}
	}
	return result, nil
}

func microFixtureSamplingForPicture(sourceImage microFixtureSourceImage, object objectFailureRecord) *microFixtureSampling {
	if sourceImage.Width <= 0 || sourceImage.Height <= 0 || object.PixelBounds == (ObjectPixelBounds{}) {
		return nil
	}
	sampling := &microFixtureSampling{
		IntegerGeometryWidth:  object.PixelBounds.MaxX - object.PixelBounds.MinX + 1,
		IntegerGeometryHeight: object.PixelBounds.MaxY - object.PixelBounds.MinY + 1,
	}
	if object.OutputPixelBounds != nil {
		sampling.OutputCropOffsetX = object.OutputPixelBounds.MinX - object.PixelBounds.MinX
		sampling.OutputCropOffsetY = object.OutputPixelBounds.MinY - object.PixelBounds.MinY
		sampling.OutputCropWidth = object.OutputPixelBounds.MaxX - object.OutputPixelBounds.MinX + 1
		sampling.OutputCropHeight = object.OutputPixelBounds.MaxY - object.OutputPixelBounds.MinY + 1
	}
	if object.FractionalBounds != (ObjectFloatBounds{}) {
		sampling.FractionalGeometryWidth = object.FractionalBounds.MaxX - object.FractionalBounds.MinX
		sampling.FractionalGeometryHeight = object.FractionalBounds.MaxY - object.FractionalBounds.MinY
		sampling.FractionalOffsetX = object.FractionalBounds.MinX - float64(object.PixelBounds.MinX)
		sampling.FractionalOffsetY = object.FractionalBounds.MinY - float64(object.PixelBounds.MinY)
		sampling.SourceToGeometryScaleX = sampling.FractionalGeometryWidth / float64(sourceImage.Width)
		sampling.SourceToGeometryScaleY = sampling.FractionalGeometryHeight / float64(sourceImage.Height)
	}
	return sampling
}

func writeMicroFixtureSourceArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, referenceCropPath string, visibleArtifacts microFixtureVisibleArtifacts, object objectFailureRecord) microFixtureSourceArtifacts {
	t.Helper()
	if object.OutputPixelBounds == nil {
		return microFixtureSourceArtifacts{}
	}
	artifacts := microFixtureSourceArtifacts{}
	if object.BeforeArtifactPath != "" {
		beforePath := filepath.Join(microDir, "source-before-crop.png")
		if err := writeCroppedPNG(resolveTestArtifactPath(object.BeforeArtifactPath), beforePath, *object.OutputPixelBounds); err != nil {
			t.Fatalf("write source before crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
		}
		artifacts.beforePath = beforePath
	}
	if object.ThroughArtifactPath != "" {
		throughPath := filepath.Join(microDir, "source-through-crop.png")
		if err := writeCroppedPNG(resolveTestArtifactPath(object.ThroughArtifactPath), throughPath, *object.OutputPixelBounds); err != nil {
			t.Fatalf("write source through crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
		}
		artifacts.throughPath = throughPath
		if referenceCropPath != "" {
			throughDiffPath := filepath.Join(microDir, "source-through-diff.json")
			diff, err := comparePNG(throughPath, referenceCropPath)
			if err != nil {
				t.Fatalf("compare source through crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
			}
			if err := writeJSONFile(throughDiffPath, realWorldDiffArtifact{
				DeckInput:   deckInput,
				SlideNumber: slideNumber,
				Diff:        diff,
			}); err != nil {
				t.Fatalf("write source through diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
			}
			artifacts.throughDiffPath = throughDiffPath
		}
		if visibleArtifacts.referencePath != "" && len(visibleArtifacts.occlusions) > 0 {
			throughVisiblePath := filepath.Join(microDir, "source-through-visible-crop.png")
			if err := writeVisibleCroppedPNG(resolveTestArtifactPath(object.ThroughArtifactPath), throughVisiblePath, *object.OutputPixelBounds, visibleArtifacts.occlusions); err != nil {
				t.Fatalf("write source through visible crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
			}
			artifacts.throughVisiblePath = throughVisiblePath
			throughVisibleDiffPath := filepath.Join(microDir, "source-through-visible-diff.json")
			diff, err := comparePNG(throughVisiblePath, visibleArtifacts.referencePath)
			if err != nil {
				t.Fatalf("compare source through visible crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
			}
			if err := writeJSONFile(throughVisibleDiffPath, realWorldDiffArtifact{
				DeckInput:   deckInput,
				SlideNumber: slideNumber,
				Diff:        diff,
			}); err != nil {
				t.Fatalf("write source through visible diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
			}
			artifacts.throughVisibleDiffPath = throughVisibleDiffPath
		}
	}
	return artifacts
}

func writeMicroFixtureGeometryArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, renderPath string, referencePath string, object objectFailureRecord) microFixtureGeometryArtifacts {
	t.Helper()
	if object.PixelBounds == (ObjectPixelBounds{}) {
		return microFixtureGeometryArtifacts{}
	}
	gotGeometryCropPath := filepath.Join(microDir, "got-geometry-crop.png")
	referenceGeometryCropPath := filepath.Join(microDir, "reference-geometry-crop.png")
	if err := writeCroppedPNG(renderPath, gotGeometryCropPath, object.PixelBounds); err != nil {
		t.Fatalf("write geometry micro-fixture got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeCroppedPNG(referencePath, referenceGeometryCropPath, object.PixelBounds); err != nil {
		t.Fatalf("write geometry micro-fixture reference crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	geometryDiffPath := filepath.Join(microDir, "geometry-diff.json")
	diff, err := comparePNG(gotGeometryCropPath, referenceGeometryCropPath)
	if err != nil {
		t.Fatalf("compare geometry micro-fixture crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(geometryDiffPath, realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write geometry micro-fixture diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	return microFixtureGeometryArtifacts{
		gotPath:       gotGeometryCropPath,
		referencePath: referenceGeometryCropPath,
		diffPath:      geometryDiffPath,
	}
}

func writeMicroFixtureVisibleArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, renderPath string, referencePath string, object objectFailureRecord, attribution objectAttributionArtifact) microFixtureVisibleArtifacts {
	t.Helper()
	occlusions := microFixtureOcclusions(object, attribution.Objects)
	if len(occlusions) == 0 || object.OutputPixelBounds == nil {
		return microFixtureVisibleArtifacts{}
	}
	gotVisibleCropPath := filepath.Join(microDir, "got-visible-crop.png")
	referenceVisibleCropPath := filepath.Join(microDir, "reference-visible-crop.png")
	if err := writeVisibleCroppedPNG(renderPath, gotVisibleCropPath, *object.OutputPixelBounds, occlusions); err != nil {
		t.Fatalf("write visible micro-fixture got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeVisibleCroppedPNG(referencePath, referenceVisibleCropPath, *object.OutputPixelBounds, occlusions); err != nil {
		t.Fatalf("write visible micro-fixture reference crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	visibleDiffPath := filepath.Join(microDir, "visible-micro-diff.json")
	diff, err := comparePNG(gotVisibleCropPath, referenceVisibleCropPath)
	if err != nil {
		t.Fatalf("compare visible micro-fixture crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(visibleDiffPath, realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write visible micro-fixture diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	return microFixtureVisibleArtifacts{
		gotPath:       gotVisibleCropPath,
		referencePath: referenceVisibleCropPath,
		diffPath:      visibleDiffPath,
		occlusions:    occlusions,
	}
}

func writeMicroFixtureTargetScope(t *testing.T, deckInput string, slideNumber int, microDir string, gotCropPath string, referenceCropPath string, visibleArtifacts microFixtureVisibleArtifacts, object objectFailureRecord, underpaints []microFixtureUnderpaint) (string, microFixtureTargetScope) {
	t.Helper()
	targetGotPath := gotCropPath
	targetReferencePath := referenceCropPath
	targetCompared := "got-crop.png vs reference-crop.png"
	if visibleArtifacts.gotPath != "" && visibleArtifacts.referencePath != "" {
		targetGotPath = visibleArtifacts.gotPath
		targetReferencePath = visibleArtifacts.referencePath
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
	}
	scope, err := microFixtureTargetScopeDiagnostic(targetGotPath, targetReferencePath, object, underpaints)
	if err != nil {
		t.Fatalf("analyze micro-fixture target scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	scope.TargetCompared = targetCompared
	if scope.DifferentPixelsOutsideObjectMask > 0 {
		scope.Warning = fmt.Sprintf("%d differing pixel(s) are outside the current object artifact alpha mask; inspect target scope before treating this as an object-only renderer failure", scope.DifferentPixelsOutsideObjectMask)
	} else if scope.DifferentPixelsInsidePartialAlphaObjectMask > 0 {
		scope.Warning = fmt.Sprintf("%d differing pixel(s) are inside partial-alpha object mask pixels; inspect background/underpaint before treating this as a full-coverage object renderer failure", scope.DifferentPixelsInsidePartialAlphaObjectMask)
	}
	targetScopePath := filepath.Join(microDir, "target-scope.json")
	if err := writeJSONFile(targetScopePath, scope); err != nil {
		t.Fatalf("write micro-fixture target scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	return targetScopePath, scope
}

func writeMicroFixtureNonUnderpaintArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, gotCropPath string, referenceCropPath string, visibleArtifacts microFixtureVisibleArtifacts, object objectFailureRecord, underpaints []microFixtureUnderpaint) microFixtureNonUnderpaintArtifacts {
	t.Helper()
	if object.OutputPixelBounds == nil || len(underpaints) == 0 {
		return microFixtureNonUnderpaintArtifacts{}
	}
	targetGotPath := gotCropPath
	targetReferencePath := referenceCropPath
	if visibleArtifacts.gotPath != "" && visibleArtifacts.referencePath != "" {
		targetGotPath = visibleArtifacts.gotPath
		targetReferencePath = visibleArtifacts.referencePath
	}
	gotPath := filepath.Join(microDir, "non-underpaint-got-crop.png")
	referencePath := filepath.Join(microDir, "non-underpaint-reference-crop.png")
	if err := writeNonUnderpaintedTargetPNG(targetGotPath, gotPath, *object.OutputPixelBounds, underpaints); err != nil {
		t.Fatalf("write non-underpaint got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeNonUnderpaintedTargetPNG(targetReferencePath, referencePath, *object.OutputPixelBounds, underpaints); err != nil {
		t.Fatalf("write non-underpaint reference crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	diffPath := filepath.Join(microDir, "non-underpaint-diff.json")
	diff, err := comparePNG(gotPath, referencePath)
	if err != nil {
		t.Fatalf("compare non-underpaint crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(diffPath, realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write non-underpaint diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	scope, err := microFixtureTargetScopeDiagnostic(gotPath, referencePath, object, nil)
	if err != nil {
		t.Fatalf("analyze non-underpaint target scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	scope.TargetCompared = filepath.Base(gotPath) + " vs " + filepath.Base(referencePath)
	return microFixtureNonUnderpaintArtifacts{
		gotPath:       gotPath,
		referencePath: referencePath,
		diffPath:      diffPath,
		targetScope:   scope,
	}
}

func writeMicroFixtureShadowAlphaArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, gotCropPath string, referenceCropPath string, visibleArtifacts microFixtureVisibleArtifacts, sourceArtifacts microFixtureSourceArtifacts, object objectFailureRecord) microFixtureShadowAlphaArtifacts {
	t.Helper()
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" || !object.ResolvedStyle.Shadow || sourceArtifacts.beforePath == "" {
		return microFixtureShadowAlphaArtifacts{}
	}
	targetGotPath := gotCropPath
	targetReferencePath := referenceCropPath
	if visibleArtifacts.gotPath != "" && visibleArtifacts.referencePath != "" {
		targetGotPath = visibleArtifacts.gotPath
		targetReferencePath = visibleArtifacts.referencePath
	}
	scope, err := microFixtureShadowAlphaDiagnostic(targetGotPath, targetReferencePath, sourceArtifacts.beforePath, object)
	if err != nil {
		t.Fatalf("analyze shadow alpha scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	scope.TargetCompared = filepath.Base(targetGotPath) + " vs " + filepath.Base(targetReferencePath)
	scopePath := filepath.Join(microDir, "shadow-alpha-scope.json")
	if err := writeJSONFile(scopePath, scope); err != nil {
		t.Fatalf("write shadow alpha scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	heatmapPath := filepath.Join(microDir, "shadow-alpha-correction-heatmap.png")
	if err := writeShadowAlphaCorrectionHeatmap(targetGotPath, targetReferencePath, sourceArtifacts.beforePath, heatmapPath, object); err != nil {
		t.Fatalf("write shadow alpha correction heatmap for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	return microFixtureShadowAlphaArtifacts{
		scopePath:   scopePath,
		heatmapPath: heatmapPath,
		scope:       scope,
	}
}

func microFixtureShadowAlphaDiagnostic(gotCropPath string, referenceCropPath string, backgroundCropPath string, object objectFailureRecord) (microFixtureShadowAlphaScope, error) {
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" {
		return microFixtureShadowAlphaScope{}, nil
	}
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return microFixtureShadowAlphaScope{}, err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowAlphaScope{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowAlphaScope{}, err
	}
	mask, err := decodePNGFile(resolveTestArtifactPath(object.ObjectArtifactPath))
	if err != nil {
		return microFixtureShadowAlphaScope{}, err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	backgroundBounds := background.Bounds()
	maskBounds := mask.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx(), backgroundBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy(), backgroundBounds.Dy())
	scope := microFixtureShadowAlphaScope{
		Basis:          "estimated black shadow alpha over the rendered before-object crop; diagnostic only",
		ObjectMaskPath: object.ObjectArtifactPath,
		BackgroundPath: backgroundCropPath,
		ComparedPixels: width * height,
	}
	greaterRows := make([]int, height)
	lessRows := make([]int, height)
	greaterColumns := make([]int, width)
	lessColumns := make([]int, width)
	deltaCounts := make(map[int]int)
	var greaterSumX int
	var greaterSumY int
	var lessSumX int
	var lessSumY int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if fullX < 0 || fullY < 0 || fullX >= maskBounds.Dx() || fullY >= maskBounds.Dy() {
				continue
			}
			_, _, _, maskAlpha := mask.At(maskBounds.Min.X+fullX, maskBounds.Min.Y+fullY).RGBA()
			if maskAlpha == 0 || maskAlpha == 0xffff {
				continue
			}
			scope.ShadowMaskPixels++
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			br, bg, bb, ba := background.At(backgroundBounds.Min.X+x, backgroundBounds.Min.Y+y).RGBA()
			if ga == 0 || ra == 0 || ba == 0 {
				continue
			}
			backgroundLuma := averageRGB8(br, bg, bb)
			if backgroundLuma == 0 {
				continue
			}
			gotAlpha := estimateBlackOverlayAlpha8(backgroundLuma, averageRGB8(gr, gg, gb))
			referenceAlpha := estimateBlackOverlayAlpha8(backgroundLuma, averageRGB8(rr, rg, rb))
			delta := referenceAlpha - gotAlpha
			if delta == 0 {
				continue
			}
			scope.AnalyzedPixels++
			scope.ReferenceAlphaDeltaSum8 += delta
			scope.ReferenceAlphaAbsoluteDeltaSum8 += absInt(delta)
			deltaCounts[delta]++
			if delta > 0 {
				scope.ReferenceAlphaGreaterPixels++
				scope.ReferenceAlphaGreaterDeltaSum8 += delta
				includeImageDiffBounds(&scope.ReferenceAlphaGreaterBounds, x, y)
				greaterSumX += x
				greaterSumY += y
				greaterRows[y]++
				greaterColumns[x]++
			} else {
				scope.ReferenceAlphaLessPixels++
				scope.ReferenceAlphaLessDeltaSum8 += delta
				includeImageDiffBounds(&scope.ReferenceAlphaLessBounds, x, y)
				lessSumX += x
				lessSumY += y
				lessRows[y]++
				lessColumns[x]++
			}
		}
	}
	if scope.ReferenceAlphaGreaterPixels > 0 {
		scope.ReferenceAlphaGreaterCentroid = &microFixtureFloatPoint{
			X: float64(greaterSumX) / float64(scope.ReferenceAlphaGreaterPixels),
			Y: float64(greaterSumY) / float64(scope.ReferenceAlphaGreaterPixels),
		}
	}
	if scope.ReferenceAlphaLessPixels > 0 {
		scope.ReferenceAlphaLessCentroid = &microFixtureFloatPoint{
			X: float64(lessSumX) / float64(scope.ReferenceAlphaLessPixels),
			Y: float64(lessSumY) / float64(scope.ReferenceAlphaLessPixels),
		}
	}
	scope.TopReferenceAlphaDeltaSums8 = topMicroFixtureDeltaCounts(deltaCounts, 12)
	scope.TopReferenceAlphaGreaterRows = topMicroFixtureAxisCounts(greaterRows, 8)
	scope.TopReferenceAlphaLessRows = topMicroFixtureAxisCounts(lessRows, 8)
	scope.TopReferenceAlphaGreaterColumns = topMicroFixtureAxisCounts(greaterColumns, 8)
	scope.TopReferenceAlphaLessColumns = topMicroFixtureAxisCounts(lessColumns, 8)
	if scope.AnalyzedPixels == 0 && scope.ShadowMaskPixels > 0 {
		scope.Warning = "partial-alpha object pixels were present, but none had a non-zero estimated shadow alpha delta"
	}
	return scope, nil
}

func writeShadowAlphaCorrectionHeatmap(gotCropPath string, referenceCropPath string, backgroundCropPath string, outputPath string, object objectFailureRecord) error {
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" {
		return nil
	}
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return err
	}
	mask, err := decodePNGFile(resolveTestArtifactPath(object.ObjectArtifactPath))
	if err != nil {
		return err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	backgroundBounds := background.Bounds()
	maskBounds := mask.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx(), backgroundBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy(), backgroundBounds.Dy())
	heatmap := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if fullX < 0 || fullY < 0 || fullX >= maskBounds.Dx() || fullY >= maskBounds.Dy() {
				continue
			}
			_, _, _, maskAlpha := mask.At(maskBounds.Min.X+fullX, maskBounds.Min.Y+fullY).RGBA()
			if maskAlpha == 0 || maskAlpha == 0xffff {
				continue
			}
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			br, bg, bb, ba := background.At(backgroundBounds.Min.X+x, backgroundBounds.Min.Y+y).RGBA()
			if ga == 0 || ra == 0 || ba == 0 {
				continue
			}
			backgroundLuma := averageRGB8(br, bg, bb)
			if backgroundLuma == 0 {
				continue
			}
			gotAlpha := estimateBlackOverlayAlpha8(backgroundLuma, averageRGB8(gr, gg, gb))
			referenceAlpha := estimateBlackOverlayAlpha8(backgroundLuma, averageRGB8(rr, rg, rb))
			delta := referenceAlpha - gotAlpha
			if delta == 0 {
				continue
			}
			intensity := min(255, maxInt(32, absInt(delta)*16))
			if delta > 0 {
				heatmap.SetRGBA(x, y, color.RGBA{R: 255, A: uint8(intensity)})
			} else {
				heatmap.SetRGBA(x, y, color.RGBA{B: 255, A: uint8(intensity)})
			}
		}
	}
	return writePNG(outputPath, heatmap)
}

func searchMicroFixtureShadowPhase(referenceCropPath string, backgroundCropPath string, object objectFailureRecord, size slideSize, canvas image.Rectangle) (microFixtureShadowPhaseSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		return microFixtureShadowPhaseSearchArtifact{}, fmt.Errorf("shadow phase search requires a custom-path object with an object artifact")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowPhaseSearchArtifact{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowPhaseSearchArtifact{}, err
	}
	objectMask, err := decodePNGFile(resolveTestArtifactPath(object.ObjectArtifactPath))
	if err != nil {
		return microFixtureShadowPhaseSearchArtifact{}, err
	}
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowPhaseSearchArtifact{}, fmt.Errorf("object pixel bounds are outside the canvas")
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	shadowBounds := targetBounds.Add(shadowOffset(element, size, canvas.Dx()))
	blur := shadowBlurPixels(element, size, canvas.Dx())
	shadowAlpha := parseObjectColorAlpha(object.ResolvedStyle.ShadowColor)
	if shadowAlpha == 0 {
		return microFixtureShadowPhaseSearchArtifact{}, fmt.Errorf("shadow color has no alpha: %q", object.ResolvedStyle.ShadowColor)
	}

	referenceBounds := reference.Bounds()
	backgroundBounds := background.Bounds()
	maskBounds := objectMask.Bounds()
	width := min(referenceBounds.Dx(), backgroundBounds.Dx())
	height := min(referenceBounds.Dy(), backgroundBounds.Dy())
	samples := make([]shadowPhaseSample, 0, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if fullX < 0 || fullY < 0 || fullX >= maskBounds.Dx() || fullY >= maskBounds.Dy() {
				continue
			}
			mr, mg, mb, maskAlpha := objectMask.At(maskBounds.Min.X+fullX, maskBounds.Min.Y+fullY).RGBA()
			if maskAlpha == 0 || maskAlpha == 0xffff || partialAlphaObjectMaskTone(mr, mg, mb, maskAlpha) != 1 {
				continue
			}
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			br, bg, bb, ba := background.At(backgroundBounds.Min.X+x, backgroundBounds.Min.Y+y).RGBA()
			if ra == 0 || ba == 0 {
				continue
			}
			backgroundLuma := averageRGB8(br, bg, bb)
			if backgroundLuma == 0 {
				continue
			}
			samples = append(samples, shadowPhaseSample{
				X:              fullX,
				Y:              fullY,
				ReferenceAlpha: estimateBlackOverlayAlpha8(backgroundLuma, averageRGB8(rr, rg, rb)),
			})
		}
	}
	artifact := microFixtureShadowPhaseSearchArtifact{
		Basis:          "candidate custom-path shadow alpha masks compared only on current dark partial-alpha object pixels; diagnostic only",
		AnalyzedPixels: len(samples),
	}
	if len(samples) == 0 {
		return artifact, nil
	}
	shiftValues := []float64{-1, -0.75, -0.5, -0.25, 0, 0.25, 0.5, 0.75, 1}
	sampleValues := []float64{0, 0.5}
	for _, shiftX := range shiftValues {
		for _, shiftY := range shiftValues {
			for _, sampleX := range sampleValues {
				for _, sampleY := range sampleValues {
					mask := customPathShadowPhaseMask(canvas, shadowBounds, object.ResolvedStyle.CustomPathCoordinates, uint8(shadowAlpha), blur, shiftX, shiftY, sampleX, sampleY)
					candidate := microFixtureShadowPhaseCandidate{
						ShiftX:  shiftX,
						ShiftY:  shiftY,
						SampleX: sampleX,
						SampleY: sampleY,
					}
					for _, sample := range samples {
						alpha := int(mask.alphaAt(sample.X, sample.Y))
						delta := sample.ReferenceAlpha - alpha
						if delta == 0 {
							continue
						}
						candidate.DifferentPixels++
						candidate.ReferenceAlphaDeltaSum += delta
						candidate.AbsoluteAlphaDeltaSum += absInt(delta)
						if delta > 0 {
							candidate.ReferenceAlphaGreater++
						} else {
							candidate.ReferenceAlphaLess++
						}
					}
					if shiftX == 0 && shiftY == 0 && sampleX == 0 && sampleY == 0 {
						baseline := candidate
						artifact.Baseline = &baseline
					}
					artifact.Candidates = append(artifact.Candidates, candidate)
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].AbsoluteAlphaDeltaSum != artifact.Candidates[j].AbsoluteAlphaDeltaSum {
			return artifact.Candidates[i].AbsoluteAlphaDeltaSum < artifact.Candidates[j].AbsoluteAlphaDeltaSum
		}
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		return absInt(artifact.Candidates[i].ReferenceAlphaDeltaSum) < absInt(artifact.Candidates[j].ReferenceAlphaDeltaSum)
	})
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixtureShadowComposite(referenceCropPath string, backgroundCropPath string, object objectFailureRecord, occlusions []microFixtureOcclusion, size slideSize, canvas image.Rectangle, outputDir string) (microFixtureShadowCompositeSearchArtifact, error) {
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		return microFixtureShadowCompositeSearchArtifact{}, fmt.Errorf("shadow composite search requires a custom-path object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowCompositeSearchArtifact{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowCompositeSearchArtifact{}, err
	}
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowCompositeSearchArtifact{}, fmt.Errorf("object pixel bounds are outside the canvas")
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	shadowBounds := targetBounds.Add(shadowOffset(element, size, canvas.Dx()))
	blur := shadowBlurPixels(element, size, canvas.Dx())
	shadowAlpha := parseObjectColorAlpha(object.ResolvedStyle.ShadowColor)
	if shadowAlpha == 0 {
		return microFixtureShadowCompositeSearchArtifact{}, fmt.Errorf("shadow color has no alpha: %q", object.ResolvedStyle.ShadowColor)
	}
	fill, ok := parseObjectColorRGBA(object.ResolvedStyle.Fill)
	if !ok {
		return microFixtureShadowCompositeSearchArtifact{}, fmt.Errorf("fill color could not be parsed: %q", object.ResolvedStyle.Fill)
	}
	artifact := microFixtureShadowCompositeSearchArtifact{
		Basis: "candidate custom-path shadow masks composited over source-before crop, then current custom-path fill, compared to fixture reference crop; diagnostic only",
	}
	shiftValues := []float64{-1, -0.75, -0.5, -0.25, 0, 0.25, 0.5, 0.75, 1}
	sampleValues := []float64{0, 0.5}
	for _, shiftX := range shiftValues {
		for _, shiftY := range shiftValues {
			for _, sampleX := range sampleValues {
				for _, sampleY := range sampleValues {
					mask := customPathShadowPhaseMask(canvas, shadowBounds, object.ResolvedStyle.CustomPathCoordinates, uint8(shadowAlpha), blur, shiftX, shiftY, sampleX, sampleY)
					candidateImage := renderShadowCompositeCandidate(background, object, mask, fill, occlusions)
					metrics := compareCandidateImage(reference, candidateImage)
					candidate := microFixtureShadowCompositeCandidate{
						ShiftX:                        shiftX,
						ShiftY:                        shiftY,
						SampleX:                       sampleX,
						SampleY:                       sampleY,
						DifferentPixels:               metrics.DifferentPixels,
						DifferentBounds:               metrics.DifferentBounds,
						TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
						MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
						ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
						ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
						ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
						ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
					}
					if shiftX == 0 && shiftY == 0 && sampleX == 0 && sampleY == 0 {
						baseline := candidate
						artifact.Baseline = &baseline
					}
					artifact.Candidates = append(artifact.Candidates, candidate)
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return absInt(artifact.Candidates[i].ReferenceRGBDeltaSum8Bit) < absInt(artifact.Candidates[j].ReferenceRGBDeltaSum8Bit)
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		mask := customPathShadowPhaseMask(canvas, shadowBounds, object.ResolvedStyle.CustomPathCoordinates, uint8(shadowAlpha), blur, best.ShiftX, best.ShiftY, best.SampleX, best.SampleY)
		if err := writePNG(filepath.Join(outputDir, "shadow-composite-best.png"), renderShadowCompositeCandidate(background, object, mask, fill, occlusions)); err != nil {
			return microFixtureShadowCompositeSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixtureShadowParameters(referenceCropPath string, backgroundCropPath string, object objectFailureRecord, occlusions []microFixtureOcclusion, size slideSize, canvas image.Rectangle, outputDir string) (microFixtureShadowParameterSearchArtifact, error) {
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		return microFixtureShadowParameterSearchArtifact{}, fmt.Errorf("shadow parameter search requires a custom-path object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowParameterSearchArtifact{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowParameterSearchArtifact{}, err
	}
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowParameterSearchArtifact{}, fmt.Errorf("object pixel bounds are outside the canvas")
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	baseOffset := shadowOffset(element, size, canvas.Dx())
	baseBlur := shadowBlurPixels(element, size, canvas.Dx())
	baseAlpha := parseObjectColorAlpha(object.ResolvedStyle.ShadowColor)
	if baseAlpha == 0 {
		return microFixtureShadowParameterSearchArtifact{}, fmt.Errorf("shadow color has no alpha: %q", object.ResolvedStyle.ShadowColor)
	}
	fill, ok := parseObjectColorRGBA(object.ResolvedStyle.Fill)
	if !ok {
		return microFixtureShadowParameterSearchArtifact{}, fmt.Errorf("fill color could not be parsed: %q", object.ResolvedStyle.Fill)
	}
	artifact := microFixtureShadowParameterSearchArtifact{
		Basis: "candidate outer-shadow blur, alpha, and pixel offset around authored outerShdw values, composited over source-before crop and compared to fixture reference crop; diagnostic only",
	}
	blurValues := uniqueIntsInRange([]int{baseBlur - 4, baseBlur - 2, baseBlur - 1, baseBlur, baseBlur + 1, baseBlur + 2, baseBlur + 4}, 0, 64)
	alphaValues := uniqueIntsInRange([]int{baseAlpha - 24, baseAlpha - 16, baseAlpha - 8, baseAlpha, baseAlpha + 8, baseAlpha + 16, baseAlpha + 24}, 0, 255)
	offsetDeltas := []image.Point{
		{},
		{X: -1}, {X: 1}, {Y: -1}, {Y: 1},
		{X: -1, Y: -1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 1},
	}
	for _, blur := range blurValues {
		for _, alpha := range alphaValues {
			for _, delta := range offsetDeltas {
				offset := image.Point{X: baseOffset.X + delta.X, Y: baseOffset.Y + delta.Y}
				candidateImage := renderShadowParameterCandidate(background, object, targetBounds, fill, offset, blur, alpha, occlusions)
				metrics := compareCandidateImage(reference, candidateImage)
				name := fmt.Sprintf("blur_%d_alpha_%d_offset_%+d_%+d", blur, alpha, delta.X, delta.Y)
				candidate := microFixtureShadowParameterCandidate{
					Name:                          name,
					BlurPixels:                    blur,
					Alpha:                         alpha,
					OffsetX:                       offset.X,
					OffsetY:                       offset.Y,
					DifferentPixels:               metrics.DifferentPixels,
					DifferentBounds:               metrics.DifferentBounds,
					TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
					MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
					ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
					ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
					ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
					ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
				}
				if blur == baseBlur && alpha == baseAlpha && delta == (image.Point{}) {
					baseline := candidate
					artifact.Baseline = &baseline
				}
				artifact.Candidates = append(artifact.Candidates, candidate)
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return absInt(artifact.Candidates[i].ReferenceRGBDeltaSum8Bit) < absInt(artifact.Candidates[j].ReferenceRGBDeltaSum8Bit)
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		if err := writePNG(filepath.Join(outputDir, "shadow-parameter-best.png"), renderShadowParameterCandidate(background, object, targetBounds, fill, image.Point{X: best.OffsetX, Y: best.OffsetY}, best.BlurPixels, best.Alpha, occlusions)); err != nil {
			return microFixtureShadowParameterSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixtureShadowKernels(referenceCropPath string, backgroundCropPath string, object objectFailureRecord, occlusions []microFixtureOcclusion, size slideSize, canvas image.Rectangle, outputDir string) (microFixtureShadowKernelSearchArtifact, error) {
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		return microFixtureShadowKernelSearchArtifact{}, fmt.Errorf("shadow kernel search requires a custom-path object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowKernelSearchArtifact{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowKernelSearchArtifact{}, err
	}
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowKernelSearchArtifact{}, fmt.Errorf("object pixel bounds are outside the canvas")
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	offset := shadowOffset(element, size, canvas.Dx())
	blur := shadowBlurPixels(element, size, canvas.Dx())
	alpha := parseObjectColorAlpha(object.ResolvedStyle.ShadowColor)
	if alpha == 0 {
		return microFixtureShadowKernelSearchArtifact{}, fmt.Errorf("shadow color has no alpha: %q", object.ResolvedStyle.ShadowColor)
	}
	fill, ok := parseObjectColorRGBA(object.ResolvedStyle.Fill)
	if !ok {
		return microFixtureShadowKernelSearchArtifact{}, fmt.Errorf("fill color could not be parsed: %q", object.ResolvedStyle.Fill)
	}
	artifact := microFixtureShadowKernelSearchArtifact{
		Basis: "candidate shadow blur kernels using authored outerShdw alpha, offset, and blur radius, composited over source-before crop and compared to fixture reference crop; diagnostic only",
	}
	kernels := []string{"gaussian_sigma_half", "gaussian_sigma_third", "gaussian_sigma_quarter", "box_once", "box_three_pass"}
	for _, kernel := range kernels {
		candidateImage := renderShadowKernelCandidate(background, object, targetBounds, fill, offset, blur, alpha, kernel, occlusions)
		metrics := compareCandidateImage(reference, candidateImage)
		candidate := microFixtureShadowKernelCandidate{
			Name:                          kernel,
			Kernel:                        kernel,
			DifferentPixels:               metrics.DifferentPixels,
			DifferentBounds:               metrics.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
			ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
			ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
			ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
			ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
		}
		if kernel == "gaussian_sigma_half" {
			baseline := candidate
			artifact.Baseline = &baseline
		}
		artifact.Candidates = append(artifact.Candidates, candidate)
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return absInt(artifact.Candidates[i].ReferenceRGBDeltaSum8Bit) < absInt(artifact.Candidates[j].ReferenceRGBDeltaSum8Bit)
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		if err := writePNG(filepath.Join(outputDir, "shadow-kernel-best.png"), renderShadowKernelCandidate(background, object, targetBounds, fill, offset, blur, alpha, best.Kernel, occlusions)); err != nil {
			return microFixtureShadowKernelSearchArtifact{}, err
		}
	}
	return artifact, nil
}

func searchMicroFixtureShadowGeometry(referenceCropPath string, backgroundCropPath string, object objectFailureRecord, occlusions []microFixtureOcclusion, size slideSize, canvas image.Rectangle, outputDir string) (microFixtureShadowGeometrySearchArtifact, error) {
	if object.OutputPixelBounds == nil || len(object.ResolvedStyle.CustomPathCoordinates) < 3 {
		return microFixtureShadowGeometrySearchArtifact{}, fmt.Errorf("shadow geometry search requires a custom-path object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureShadowGeometrySearchArtifact{}, err
	}
	background, err := decodePNGFile(backgroundCropPath)
	if err != nil {
		return microFixtureShadowGeometrySearchArtifact{}, err
	}
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowGeometrySearchArtifact{}, fmt.Errorf("object pixel bounds are outside the canvas")
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	integerOffset := shadowOffset(element, size, canvas.Dx())
	fractionalOffset := shadowOffsetFloat(element, size, canvas.Dx())
	blur := shadowBlurPixels(element, size, canvas.Dx())
	alpha := parseObjectColorAlpha(object.ResolvedStyle.ShadowColor)
	if alpha == 0 {
		return microFixtureShadowGeometrySearchArtifact{}, fmt.Errorf("shadow color has no alpha: %q", object.ResolvedStyle.ShadowColor)
	}
	fill, ok := parseObjectColorRGBA(object.ResolvedStyle.Fill)
	if !ok {
		return microFixtureShadowGeometrySearchArtifact{}, fmt.Errorf("fill color could not be parsed: %q", object.ResolvedStyle.Fill)
	}
	artifact := microFixtureShadowGeometrySearchArtifact{
		Basis: "candidate custom-path shadow masks using integer and fractional target bounds, composited over source-before crop and compared to fixture reference crop; diagnostic only",
	}
	type rectCandidate struct {
		name string
		rect floatRect
	}
	rectCandidates := []rectCandidate{
		{name: "pixel_bounds_current", rect: floatRectFromImageRect(targetBounds)},
	}
	if object.FractionalBounds != (ObjectFloatBounds{}) {
		fractional := floatRectFromObjectFloatBounds(object.FractionalBounds)
		rectCandidates = append(rectCandidates,
			rectCandidate{name: "fractional_exact", rect: fractional},
			rectCandidate{name: "fractional_floor_ceil", rect: floatRectFromImageRect(floatRectPixelBounds(fractional).Intersect(canvas))},
			rectCandidate{name: "fractional_round", rect: roundedFloatRect(fractional)},
		)
	}
	type offsetCandidate struct {
		name   string
		offset microFixtureFloatPoint
	}
	offsetCandidates := []offsetCandidate{
		{name: "integer_offset", offset: microFixtureFloatPoint{X: float64(integerOffset.X), Y: float64(integerOffset.Y)}},
		{name: "fractional_offset", offset: fractionalOffset},
	}
	sampleValues := []float64{0, 0.5}
	for _, rectCandidate := range rectCandidates {
		if rectCandidate.rect.MaxX <= rectCandidate.rect.MinX || rectCandidate.rect.MaxY <= rectCandidate.rect.MinY {
			continue
		}
		for _, offsetCandidate := range offsetCandidates {
			shadowRect := offsetFloatRect(rectCandidate.rect, offsetCandidate.offset)
			for _, sampleX := range sampleValues {
				for _, sampleY := range sampleValues {
					mask := customPathShadowGeometryMask(canvas, shadowRect, object.ResolvedStyle.CustomPathCoordinates, uint8(alpha), blur, sampleX, sampleY)
					candidateImage := renderShadowCompositeCandidate(background, object, mask, fill, occlusions)
					metrics := compareCandidateImage(reference, candidateImage)
					candidate := microFixtureShadowGeometryCandidate{
						Name:                          rectCandidate.name + "/" + offsetCandidate.name,
						RectSource:                    rectCandidate.name,
						OffsetSource:                  offsetCandidate.name,
						SampleX:                       sampleX,
						SampleY:                       sampleY,
						TargetRect:                    objectFloatBoundsFromFloatRect(rectCandidate.rect),
						ShadowRect:                    objectFloatBoundsFromFloatRect(shadowRect),
						DifferentPixels:               metrics.DifferentPixels,
						DifferentBounds:               metrics.DifferentBounds,
						TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
						MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
						ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
						ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
						ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
						ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
					}
					if !mask.bounds.Empty() {
						bounds := pixelBoundsFromRect(mask.bounds)
						candidate.MaskBounds = &bounds
					}
					if rectCandidate.name == "pixel_bounds_current" && offsetCandidate.name == "integer_offset" && sampleX == 0 && sampleY == 0 {
						baseline := candidate
						artifact.Baseline = &baseline
					}
					artifact.Candidates = append(artifact.Candidates, candidate)
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return absInt(artifact.Candidates[i].ReferenceRGBDeltaSum8Bit) < absInt(artifact.Candidates[j].ReferenceRGBDeltaSum8Bit)
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		mask := customPathShadowGeometryMask(canvas, floatRectFromObjectFloatBounds(best.ShadowRect), object.ResolvedStyle.CustomPathCoordinates, uint8(alpha), blur, best.SampleX, best.SampleY)
		if err := writePNG(filepath.Join(outputDir, "shadow-geometry-best.png"), renderShadowCompositeCandidate(background, object, mask, fill, occlusions)); err != nil {
			return microFixtureShadowGeometrySearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func renderShadowParameterCandidate(background image.Image, object objectFailureRecord, targetBounds image.Rectangle, fill color.RGBA, offset image.Point, blur int, alpha int, occlusions []microFixtureOcclusion) *image.RGBA {
	bounds := background.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), background, bounds.Min, draw.Src)
	shadowBounds := targetBounds.Add(offset)
	mask := customPathShadowPhaseMask(output.Bounds().Add(image.Point{X: object.OutputPixelBounds.MinX, Y: object.OutputPixelBounds.MinY}), shadowBounds, object.ResolvedStyle.CustomPathCoordinates, uint8(alpha), blur, 0, 0, 0, 0)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if shadowAlpha := mask.alphaAt(fullX, fullY); shadowAlpha != 0 {
				blendPixel(output, x, y, color.RGBA{A: shadowAlpha})
			}
		}
	}
	relativeTargetBounds := image.Rect(
		object.PixelBounds.MinX-object.OutputPixelBounds.MinX,
		object.PixelBounds.MinY-object.OutputPixelBounds.MinY,
		object.PixelBounds.MaxX-object.OutputPixelBounds.MinX+1,
		object.PixelBounds.MaxY-object.OutputPixelBounds.MinY+1,
	).Intersect(output.Bounds())
	drawPolygon(output, relativeTargetBounds, objectFloatPointsToPathPoints(object.ResolvedStyle.CustomPathCoordinates), fill)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return output
}

func renderShadowKernelCandidate(background image.Image, object objectFailureRecord, targetBounds image.Rectangle, fill color.RGBA, offset image.Point, blur int, alpha int, kernel string, occlusions []microFixtureOcclusion) *image.RGBA {
	bounds := background.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), background, bounds.Min, draw.Src)
	shadowBounds := targetBounds.Add(offset)
	mask := customPathShadowKernelMask(output.Bounds().Add(image.Point{X: object.OutputPixelBounds.MinX, Y: object.OutputPixelBounds.MinY}), shadowBounds, object.ResolvedStyle.CustomPathCoordinates, uint8(alpha), blur, kernel)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if shadowAlpha := mask.alphaAt(fullX, fullY); shadowAlpha != 0 {
				blendPixel(output, x, y, color.RGBA{A: shadowAlpha})
			}
		}
	}
	relativeTargetBounds := image.Rect(
		object.PixelBounds.MinX-object.OutputPixelBounds.MinX,
		object.PixelBounds.MinY-object.OutputPixelBounds.MinY,
		object.PixelBounds.MaxX-object.OutputPixelBounds.MinX+1,
		object.PixelBounds.MaxY-object.OutputPixelBounds.MinY+1,
	).Intersect(output.Bounds())
	drawPolygon(output, relativeTargetBounds, objectFloatPointsToPathPoints(object.ResolvedStyle.CustomPathCoordinates), fill)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return output
}

func customPathShadowKernelMask(canvas image.Rectangle, shapeBounds image.Rectangle, points []ObjectFloatPoint, alpha uint8, blur int, kernel string) shadowPhaseMask {
	maskBounds := shapeBounds
	if blur > 0 {
		maskBounds = maskBounds.Inset(-blur)
	}
	maskBounds = maskBounds.Intersect(canvas)
	if maskBounds.Empty() || alpha == 0 {
		return shadowPhaseMask{}
	}
	polygon := make([]ObjectFloatPoint, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, ObjectFloatPoint{
			X: float64(shapeBounds.Min.X) + point.X*float64(shapeBounds.Dx()),
			Y: float64(shapeBounds.Min.Y) + point.Y*float64(shapeBounds.Dy()),
		})
	}
	width := maskBounds.Dx()
	height := maskBounds.Dy()
	alphaMask := make([]uint8, width*height)
	for y := maskBounds.Min.Y; y < maskBounds.Max.Y; y++ {
		for x := maskBounds.Min.X; x < maskBounds.Max.X; x++ {
			if pointInShadowPhasePolygon(float64(x), float64(y), polygon) {
				alphaMask[(y-maskBounds.Min.Y)*width+x-maskBounds.Min.X] = alpha
			}
		}
	}
	mask := shadowPhaseMask{bounds: maskBounds, alpha: alphaMask, width: width}
	if blur <= 0 {
		return mask
	}
	switch kernel {
	case "gaussian_sigma_third":
		mask.alpha = gaussianBlurAlphaWithSigma(mask.alpha, width, height, blur, float64(blur)/3)
	case "gaussian_sigma_quarter":
		mask.alpha = gaussianBlurAlphaWithSigma(mask.alpha, width, height, blur, float64(blur)/4)
	case "box_once":
		mask.alpha = boxBlurAlpha(mask.alpha, width, height, blur)
	case "box_three_pass":
		mask.alpha = boxBlurAlpha(mask.alpha, width, height, blur)
		mask.alpha = boxBlurAlpha(mask.alpha, width, height, blur)
		mask.alpha = boxBlurAlpha(mask.alpha, width, height, blur)
	default:
		mask.alpha = gaussianBlurAlpha(mask.alpha, width, height, blur)
	}
	return mask
}

func customPathShadowGeometryMask(canvas image.Rectangle, shapeBounds floatRect, points []ObjectFloatPoint, alpha uint8, blur int, sampleX float64, sampleY float64) shadowPhaseMask {
	maskBounds := floatRectPixelBounds(shapeBounds)
	if blur > 0 {
		maskBounds = maskBounds.Inset(-blur)
	}
	maskBounds = maskBounds.Intersect(canvas)
	if maskBounds.Empty() || alpha == 0 {
		return shadowPhaseMask{}
	}
	polygon := make([]ObjectFloatPoint, 0, len(points))
	width := shapeBounds.MaxX - shapeBounds.MinX
	height := shapeBounds.MaxY - shapeBounds.MinY
	for _, point := range points {
		polygon = append(polygon, ObjectFloatPoint{
			X: shapeBounds.MinX + point.X*width,
			Y: shapeBounds.MinY + point.Y*height,
		})
	}
	maskWidth := maskBounds.Dx()
	maskHeight := maskBounds.Dy()
	mask := make([]uint8, maskWidth*maskHeight)
	for y := maskBounds.Min.Y; y < maskBounds.Max.Y; y++ {
		for x := maskBounds.Min.X; x < maskBounds.Max.X; x++ {
			if pointInShadowPhasePolygon(float64(x)+sampleX, float64(y)+sampleY, polygon) {
				mask[(y-maskBounds.Min.Y)*maskWidth+x-maskBounds.Min.X] = alpha
			}
		}
	}
	if blur > 0 {
		mask = gaussianBlurAlpha(mask, maskWidth, maskHeight, blur)
	}
	return shadowPhaseMask{bounds: maskBounds, alpha: mask, width: maskWidth}
}

func shadowOffsetFloat(element slideElement, size slideSize, outputWidth int) microFixtureFloatPoint {
	distance := scaleEMUFloat(element.ShadowDistance, size.CX, outputWidth)
	if distance == 0 && element.ShadowDistance > 0 {
		distance = 1
	}
	angle := float64(element.ShadowDirection) / 60000 * math.Pi / 180
	return microFixtureFloatPoint{
		X: math.Cos(angle) * distance,
		Y: math.Sin(angle) * distance,
	}
}

func floatRectFromObjectFloatBounds(bounds ObjectFloatBounds) floatRect {
	return floatRect{MinX: bounds.MinX, MinY: bounds.MinY, MaxX: bounds.MaxX, MaxY: bounds.MaxY}
}

func objectFloatBoundsFromFloatRect(rect floatRect) ObjectFloatBounds {
	return ObjectFloatBounds{MinX: rect.MinX, MinY: rect.MinY, MaxX: rect.MaxX, MaxY: rect.MaxY}
}

func offsetFloatRect(rect floatRect, offset microFixtureFloatPoint) floatRect {
	return floatRect{
		MinX: rect.MinX + offset.X,
		MinY: rect.MinY + offset.Y,
		MaxX: rect.MaxX + offset.X,
		MaxY: rect.MaxY + offset.Y,
	}
}

func roundedFloatRect(rect floatRect) floatRect {
	return floatRect{
		MinX: math.Round(rect.MinX),
		MinY: math.Round(rect.MinY),
		MaxX: math.Round(rect.MaxX),
		MaxY: math.Round(rect.MaxY),
	}
}

func gaussianBlurAlphaWithSigma(src []uint8, width int, height int, radius int, sigma float64) []uint8 {
	if radius <= 0 || width <= 0 || height <= 0 {
		dst := make([]uint8, len(src))
		copy(dst, src)
		return dst
	}
	if sigma < 0.5 {
		sigma = 0.5
	}
	kernel := gaussianKernelWithSigma(radius, sigma)
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

func gaussianKernelWithSigma(radius int, sigma float64) []float64 {
	if radius <= 0 {
		return []float64{1}
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

func uniqueIntsInRange(values []int, minValue int, maxValue int) []int {
	seen := map[int]bool{}
	var result []int
	for _, value := range values {
		if value < minValue {
			value = minValue
		}
		if value > maxValue {
			value = maxValue
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func renderShadowCompositeCandidate(background image.Image, object objectFailureRecord, mask shadowPhaseMask, fill color.RGBA, occlusions []microFixtureOcclusion) *image.RGBA {
	bounds := background.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), background, bounds.Min, draw.Src)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if alpha := mask.alphaAt(fullX, fullY); alpha != 0 {
				blendPixel(output, x, y, color.RGBA{A: alpha})
			}
		}
	}
	targetBounds := image.Rect(
		object.PixelBounds.MinX-object.OutputPixelBounds.MinX,
		object.PixelBounds.MinY-object.OutputPixelBounds.MinY,
		object.PixelBounds.MaxX-object.OutputPixelBounds.MinX+1,
		object.PixelBounds.MaxY-object.OutputPixelBounds.MinY+1,
	).Intersect(output.Bounds())
	drawPolygon(output, targetBounds, objectFloatPointsToPathPoints(object.ResolvedStyle.CustomPathCoordinates), fill)
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return output
}

func searchMicroFixtureRectEdgeBlend(referenceCropPath string, nonUnderpaintReferencePath string, beforePath string, object objectFailureRecord, occlusions []microFixtureOcclusion, underpaints []microFixtureUnderpaint, outputDir string) (microFixtureRectEdgeBlendSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.ResolvedStyle.Geometry != "rect" {
		return microFixtureRectEdgeBlendSearchArtifact{}, fmt.Errorf("rectangle edge blend search requires a rectangle object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureRectEdgeBlendSearchArtifact{}, err
	}
	var nonUnderpaintReference image.Image
	var underpaintMasks []microFixtureUnderpaintMask
	if nonUnderpaintReferencePath != "" && len(underpaints) > 0 {
		nonUnderpaintReference, err = decodePNGFile(nonUnderpaintReferencePath)
		if err != nil {
			return microFixtureRectEdgeBlendSearchArtifact{}, err
		}
		underpaintMasks, err = loadUnderpaintMasks(underpaints)
		if err != nil {
			return microFixtureRectEdgeBlendSearchArtifact{}, err
		}
	}
	before, err := decodePNGFile(beforePath)
	if err != nil {
		return microFixtureRectEdgeBlendSearchArtifact{}, err
	}
	background := cropImageToObjectBounds(before, *object.OutputPixelBounds)
	fill, ok := parseObjectColorRGBA(object.ResolvedStyle.Fill)
	if !ok {
		return microFixtureRectEdgeBlendSearchArtifact{}, fmt.Errorf("fill color could not be parsed: %q", object.ResolvedStyle.Fill)
	}
	artifact := microFixtureRectEdgeBlendSearchArtifact{
		Basis: "candidate rectangle fractional-edge coverage and source-over rounding over before-object crop, compared to fixture reference crop; diagnostic only. When present, non-underpaint metrics mask earlier-object pixels and isolate the target rectangle edge.",
	}
	modes := []struct {
		name                 string
		coverageQuantization string
		blendQuantization    string
	}{
		{name: "current", coverageQuantization: "round", blendQuantization: "round"},
		{name: "blend_floor", coverageQuantization: "round", blendQuantization: "floor"},
		{name: "coverage_floor", coverageQuantization: "floor", blendQuantization: "round"},
		{name: "coverage_floor_blend_floor", coverageQuantization: "floor", blendQuantization: "floor"},
		{name: "coverage_ceil", coverageQuantization: "ceil", blendQuantization: "round"},
	}
	for _, mode := range modes {
		candidateImage := renderRectEdgeBlendCandidate(background, object, fill, mode.coverageQuantization, mode.blendQuantization, occlusions)
		metrics := compareCandidateImage(reference, candidateImage)
		candidate := microFixtureRectEdgeBlendCandidate{
			Name:                          mode.name,
			CoverageQuantization:          mode.coverageQuantization,
			BlendQuantization:             mode.blendQuantization,
			DifferentPixels:               metrics.DifferentPixels,
			DifferentBounds:               metrics.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
			ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
			ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
			ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
			ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
		}
		if nonUnderpaintReference != nil {
			nonUnderpaintCandidate := cloneRGBA(candidateImage)
			applyUnderpaintMaskToCandidate(nonUnderpaintCandidate, *object.OutputPixelBounds, underpaintMasks)
			nonUnderpaintMetrics := compareCandidateImage(nonUnderpaintReference, nonUnderpaintCandidate)
			candidate.NonUnderpaintDifferentPixels = nonUnderpaintMetrics.DifferentPixels
			candidate.NonUnderpaintDifferentBounds = nonUnderpaintMetrics.DifferentBounds
			candidate.NonUnderpaintTotalAbsoluteChannelDelta8Bit = nonUnderpaintMetrics.TotalAbsoluteChannelDelta8Bit
		}
		if mode.name == "current" {
			baseline := candidate
			artifact.Baseline = &baseline
		}
		artifact.Candidates = append(artifact.Candidates, candidate)
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		if err := writePNG(filepath.Join(outputDir, "rect-edge-blend-best.png"), renderRectEdgeBlendCandidate(background, object, fill, best.CoverageQuantization, best.BlendQuantization, occlusions)); err != nil {
			return microFixtureRectEdgeBlendSearchArtifact{}, err
		}
	}
	return artifact, nil
}

func cropImageToObjectBounds(source image.Image, crop ObjectPixelBounds) *image.RGBA {
	sourceBounds := source.Bounds()
	rect := image.Rect(crop.MinX, crop.MinY, crop.MaxX+1, crop.MaxY+1).Intersect(image.Rect(0, 0, sourceBounds.Dx(), sourceBounds.Dy()))
	if rect.Empty() {
		rect = image.Rect(0, 0, 1, 1)
	}
	output := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			output.Set(x, y, source.At(sourceBounds.Min.X+rect.Min.X+x, sourceBounds.Min.Y+rect.Min.Y+y))
		}
	}
	return output
}

func renderRectEdgeBlendCandidate(background image.Image, object objectFailureRecord, fill color.RGBA, coverageQuantization string, blendQuantization string, occlusions []microFixtureOcclusion) *image.RGBA {
	bounds := background.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), background, bounds.Min, draw.Src)
	rect := floatRect{
		MinX: object.FractionalBounds.MinX - float64(object.OutputPixelBounds.MinX),
		MinY: object.FractionalBounds.MinY - float64(object.OutputPixelBounds.MinY),
		MaxX: object.FractionalBounds.MaxX - float64(object.OutputPixelBounds.MinX),
		MaxY: object.FractionalBounds.MaxY - float64(object.OutputPixelBounds.MinY),
	}
	paintBounds := floatRectPixelBounds(rect).Intersect(output.Bounds())
	for y := paintBounds.Min.Y; y < paintBounds.Max.Y; y++ {
		for x := paintBounds.Min.X; x < paintBounds.Max.X; x++ {
			coverage := floatRectPixelCoverage(float64(x), float64(y), rect)
			if coverage <= 0 {
				continue
			}
			layer := fill
			layer.A = quantizeCoverageAlpha(fill.A, coverage, coverageQuantization)
			if layer.A == 0 {
				continue
			}
			blendRectCandidatePixel(output, x, y, layer, blendQuantization)
		}
	}
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return output
}

func applyUnderpaintMaskToCandidate(candidate *image.RGBA, crop ObjectPixelBounds, underpaintMasks []microFixtureUnderpaintMask) {
	for y := candidate.Bounds().Min.Y; y < candidate.Bounds().Max.Y; y++ {
		for x := candidate.Bounds().Min.X; x < candidate.Bounds().Max.X; x++ {
			fullX := crop.MinX + x
			fullY := crop.MinY + y
			if pointUnderpainted(fullX, fullY, underpaintMasks) {
				candidate.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
}

func quantizeCoverageAlpha(alpha uint8, coverage float64, quantization string) uint8 {
	if alpha == 0 || coverage <= 0 {
		return 0
	}
	if coverage >= 1 {
		return alpha
	}
	value := float64(alpha) * coverage
	switch quantization {
	case "floor":
		return uint8(math.Floor(value))
	case "ceil":
		return uint8(math.Ceil(value))
	default:
		return uint8(math.Round(value))
	}
}

func blendRectCandidatePixel(img *image.RGBA, x int, y int, src color.RGBA, quantization string) {
	if quantization == "floor" {
		blendPixelFloor(img, x, y, src)
		return
	}
	blendPixel(img, x, y, src)
}

func blendPixelFloor(img *image.RGBA, x int, y int, src color.RGBA) {
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
		R: uint8((int(src.R)*alpha + int(dst.R)*invAlpha) / 255),
		G: uint8((int(src.G)*alpha + int(dst.G)*invAlpha) / 255),
		B: uint8((int(src.B)*alpha + int(dst.B)*invAlpha) / 255),
		A: uint8(alpha + int(dst.A)*invAlpha/255),
	})
}

func microFixturePictureSourceImage(fixturePath string) (image.Image, error) {
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		return nil, err
	}
	var mediaParts []string
	for part := range pkg.Parts {
		if strings.HasPrefix(part, "ppt/media/") {
			mediaParts = append(mediaParts, part)
		}
	}
	sort.Strings(mediaParts)
	for _, part := range mediaParts {
		source, err := decodeImage(part, pkg.ContentTypes.ForPart(part), pkg.Parts[part])
		if err == nil {
			return source, nil
		}
	}
	return nil, fmt.Errorf("fixture has no decodable media parts")
}

type microFixturePictureSourceVariant struct {
	Name  string
	Image image.Image
}

func microFixturePictureSourceVariants(fixturePath string) ([]microFixturePictureSourceVariant, error) {
	converted, err := microFixturePictureSourceImage(fixturePath)
	if err != nil {
		return nil, err
	}
	variants := []microFixturePictureSourceVariant{{Name: "converted_icc", Image: converted}}
	if raw, err := microFixtureRawPNGSourceImage(fixturePath); err == nil {
		variants = append(variants, microFixturePictureSourceVariant{Name: "raw_png", Image: raw})
	}
	return variants, nil
}

func microFixturePictureSourceModelVariants(fixturePath string) ([]microFixturePictureSourceVariant, error) {
	baseVariants, err := microFixturePictureSourceVariants(fixturePath)
	if err != nil {
		return nil, err
	}
	variants := make([]microFixturePictureSourceVariant, 0, len(baseVariants)*3)
	for _, variant := range baseVariants {
		variants = append(variants, variant)
		variants = append(variants, microFixturePictureSourceVariant{Name: variant.Name + "_rgba", Image: imageToRGBA(variant.Image)})
		variants = append(variants, microFixturePictureSourceVariant{Name: variant.Name + "_nrgba", Image: imageToNRGBA(variant.Image)})
	}
	return variants, nil
}

func microFixturePictureSourceModelVariantSummaries(variants []microFixturePictureSourceVariant) []microFixturePictureSourceModelVariantSummary {
	summaries := make([]microFixturePictureSourceModelVariantSummary, 0, len(variants))
	for _, variant := range variants {
		bounds := variant.Image.Bounds()
		stats := microFixturePictureSourceStatsForImage(variant.Image)
		summaries = append(summaries, microFixturePictureSourceModelVariantSummary{
			Name:        variant.Name,
			GoType:      fmt.Sprintf("%T", variant.Image),
			ColorModel:  fmt.Sprintf("%T", variant.Image.ColorModel()),
			Width:       bounds.Dx(),
			Height:      bounds.Dy(),
			UniqueColor: stats.UniqueColors,
		})
	}
	return summaries
}

func microFixtureRawPNGSourceImage(fixturePath string) (image.Image, error) {
	_, data, err := microFixtureRawPNGSourceData(fixturePath)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func microFixtureRawPNGSourceData(fixturePath string) (string, []byte, error) {
	reader, err := zip.OpenReader(fixturePath)
	if err != nil {
		return "", nil, err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if !strings.HasPrefix(file.Name, "ppt/media/") || !strings.HasSuffix(strings.ToLower(file.Name), ".png") {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			return "", nil, err
		}
		data, readErr := io.ReadAll(handle)
		closeErr := handle.Close()
		if readErr != nil {
			return "", nil, readErr
		}
		if closeErr != nil {
			return "", nil, closeErr
		}
		return file.Name, data, nil
	}
	return "", nil, fmt.Errorf("fixture has no raw PNG media part")
}

func microFixturePicturePNGMetadataProfile(fixturePath string) (microFixturePicturePNGMetadataProfileArtifact, error) {
	part, data, err := microFixtureRawPNGSourceData(fixturePath)
	if err != nil {
		return microFixturePicturePNGMetadataProfileArtifact{}, err
	}
	const pngSignature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(pngSignature) || string(data[:len(pngSignature)]) != pngSignature {
		return microFixturePicturePNGMetadataProfileArtifact{}, fmt.Errorf("media part %s is not a PNG stream", part)
	}
	sum := sha256.Sum256(data)
	artifact := microFixturePicturePNGMetadataProfileArtifact{
		Basis:     "profiles raw PNG chunks from an attributed picture fixture to rule source metadata in or out before renderer changes; diagnostic only",
		MediaPart: part,
		ByteSize:  len(data),
		SHA256:    fmt.Sprintf("%x", sum[:]),
	}
	offset := len(pngSignature)
	for offset < len(data) {
		if offset+12 > len(data) {
			return microFixturePicturePNGMetadataProfileArtifact{}, fmt.Errorf("truncated PNG chunk header at byte %d", offset)
		}
		length := binary.BigEndian.Uint32(data[offset : offset+4])
		chunkType := string(data[offset+4 : offset+8])
		chunkStart := offset + 8
		chunkEnd := chunkStart + int(length)
		crcStart := chunkEnd
		crcEnd := crcStart + 4
		if chunkEnd < chunkStart || crcEnd > len(data) {
			return microFixturePicturePNGMetadataProfileArtifact{}, fmt.Errorf("truncated PNG chunk %s at byte %d", chunkType, offset)
		}
		chunkData := data[chunkStart:chunkEnd]
		recordedCRC := binary.BigEndian.Uint32(data[crcStart:crcEnd])
		calculatedCRC := crc32.ChecksumIEEE(data[offset+4 : chunkEnd])
		artifact.Chunks = append(artifact.Chunks, microFixturePNGChunk{
			Type:     chunkType,
			Length:   length,
			CRCValid: recordedCRC == calculatedCRC,
		})
		switch chunkType {
		case "IHDR":
			if len(chunkData) != 13 {
				return microFixturePicturePNGMetadataProfileArtifact{}, fmt.Errorf("invalid IHDR length %d", len(chunkData))
			}
			artifact.Width = int(binary.BigEndian.Uint32(chunkData[0:4]))
			artifact.Height = int(binary.BigEndian.Uint32(chunkData[4:8]))
			artifact.BitDepth = int(chunkData[8])
			artifact.ColorType = int(chunkData[9])
			artifact.ColorTypeName = pngColorTypeName(artifact.ColorType)
			artifact.CompressionMethod = int(chunkData[10])
			artifact.FilterMethod = int(chunkData[11])
			artifact.InterlaceMethod = int(chunkData[12])
		case "PLTE":
			artifact.HasPalette = true
			artifact.PaletteEntries = len(chunkData) / 3
		case "tRNS":
			artifact.HasTransparency = true
			artifact.TransparencyBytes = len(chunkData)
		case "gAMA":
			if len(chunkData) == 4 {
				value := float64(binary.BigEndian.Uint32(chunkData)) / 100000
				artifact.HasGamma = true
				artifact.Gamma = &value
			}
		case "sRGB":
			if len(chunkData) == 1 {
				value := int(chunkData[0])
				artifact.HasSRGB = true
				artifact.SRGBRenderingIntent = &value
			}
		case "iCCP":
			artifact.HasICCP = true
			nameEnd := bytes.IndexByte(chunkData, 0)
			if nameEnd >= 0 {
				artifact.ICCProfileName = string(chunkData[:nameEnd])
				if nameEnd+1 < len(chunkData) {
					value := int(chunkData[nameEnd+1])
					artifact.ICCCompressionMethod = &value
				}
			}
		case "pHYs":
			if len(chunkData) == 9 {
				artifact.HasPhysicalPixelSize = true
				artifact.PhysicalPixelsPerUnit = &microFixturePNGPhysicalPixels{
					X:    binary.BigEndian.Uint32(chunkData[0:4]),
					Y:    binary.BigEndian.Uint32(chunkData[4:8]),
					Unit: chunkData[8],
				}
			}
		}
		offset = crcEnd
		if chunkType == "IEND" {
			break
		}
	}
	return artifact, nil
}

func microFixturePicturePipelineProfile(manifestPath string, manifest microFixtureManifest) (microFixturePicturePipelineProfileArtifact, error) {
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	if len(pkg.SlideParts) == 0 {
		return microFixturePicturePipelineProfileArtifact{}, fmt.Errorf("picture pipeline fixture has no slide parts")
	}
	outputDir, err := os.MkdirTemp("", "puppt-picture-pipeline-*")
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	defer os.RemoveAll(outputDir)
	renderPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), fixturePath, Options{SlideNumber: 1, OutputPath: renderPath}); err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	renderedImage, err := decodePNGFile(renderPath)
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	rendered := imageToRGBA(renderedImage)
	canvas := rendered.Bounds()
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	element, ok := findMicroFixtureSlideElement(elements, manifest.Object)
	if !ok {
		return microFixturePicturePipelineProfileArtifact{}, fmt.Errorf("target picture %s %q not found in fixture", manifest.Object.CNvPrID, manifest.Object.CNvPrName)
	}
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	relationshipByID := make(map[string]pptx.Relationship, len(relationships))
	for _, relationship := range relationships {
		relationshipByID[relationship.ID] = relationship
	}
	relationship, ok := relationshipByID[element.EmbedID]
	if !ok {
		return microFixturePicturePipelineProfileArtifact{}, fmt.Errorf("target picture relationship %q not found", element.EmbedID)
	}
	source, mediaPart, partialUnsupported := pictureSourceImage(pkg, slidePart, &element, relationshipByID, relationship)
	if source == nil {
		return microFixturePicturePipelineProfileArtifact{}, fmt.Errorf("target picture source %s could not be decoded: %v", mediaPart, partialUnsupported)
	}

	gotPath := resolveTestArtifactPath(manifest.GotCropPath)
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "pipeline output crop vs got-crop.png/reference-crop.png"
	occlusions := []microFixtureOcclusion(nil)
	if manifest.GotVisibleCropPath != "" && manifest.ReferenceVisibleCropPath != "" {
		gotPath = resolveTestArtifactPath(manifest.GotVisibleCropPath)
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "pipeline output visible crop vs got-visible-crop.png/reference-visible-crop.png"
		occlusions = manifest.OccludedBy
	}
	gotCrop, err := decodePNGFile(gotPath)
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}
	referenceCrop, err := decodePNGFile(referencePath)
	if err != nil {
		return microFixturePicturePipelineProfileArtifact{}, err
	}

	sourceP3 := imageToRGBA(source)
	applyDisplayP3OutputTransform(sourceP3)
	colorMetrics := compareCandidateImage(source, sourceP3)
	cropBounds := sourceCropRect(source.Bounds(), element)
	cropped := cropImageToRGBA(source, cropBounds)
	pictureImage, pictureBounds := pictureSourceForElement(source, element)
	transformed := cropImageToRGBA(pictureImage, pictureBounds)

	target := elementPixelTarget(element, size, canvas)
	preOutputCanvas := image.NewRGBA(canvas)
	draw.Draw(preOutputCanvas, preOutputCanvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	softEdgeRendered := drawPictureRaster(preOutputCanvas, target, pictureImage, pictureBounds, element, size)
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, preOutputCanvas.Bounds().Dx())
		if normalizedRotationDegrees(element.Rotation) == 0 {
			drawPictureOutline(preOutputCanvas, target, element, lineWidth)
		}
	}
	_ = softEdgeRendered
	outputCrop := *manifest.Object.OutputPixelBounds
	preOutputCrop := cropImageToRGBA(preOutputCanvas, image.Rect(outputCrop.MinX, outputCrop.MinY, outputCrop.MaxX+1, outputCrop.MaxY+1))
	postOutputCanvas := cloneRGBA(preOutputCanvas)
	applyDisplayP3OutputTransform(postOutputCanvas)
	postOutputCrop := cropImageToRGBA(postOutputCanvas, image.Rect(outputCrop.MinX, outputCrop.MinY, outputCrop.MaxX+1, outputCrop.MaxY+1))
	occludedPixels := applyPicturePipelineOcclusions(postOutputCrop, outputCrop, occlusions)

	partial := ""
	if partialUnsupported != nil {
		partial = partialUnsupported.Error()
	}
	artifact := microFixturePicturePipelineProfileArtifact{
		ManifestPath:   manifestPath,
		FixturePath:    fixturePath,
		TargetCompared: targetCompared,
		Basis:          "splits the current picture rendering path into source decode, color conversion, source crop, transform/effects, sampling, and final output stages for an attributed picture micro-fixture; diagnostic only",
		SourceDecode: microFixturePicturePipelineSourceDecode{
			MediaPart:      mediaPart,
			ContentType:    pkg.ContentTypes.ForPart(mediaPart),
			RelationshipID: element.EmbedID,
			Relationship:   relationship.Target,
			GoType:         fmt.Sprintf("%T", source),
			Stats:          microFixturePictureSourceStatsForImage(source),
			TopColors:      topMicroFixtureSourceColors(source, 12),
		},
		Color: microFixturePicturePipelineColorStage{
			Basis:                        "current source decode is treated as sRGB bytes; final renderer output is converted to Display P3 after rasterization",
			DecodedSRGBStats:             microFixturePictureSourceStatsForImage(source),
			DisplayP3ConvertedStats:      microFixturePictureSourceStatsForImage(sourceP3),
			DisplayP3ChangedPixels:       colorMetrics.DifferentPixels,
			DisplayP3AbsoluteDelta8Bit:   colorMetrics.TotalAbsoluteChannelDelta8Bit,
			DisplayP3MaxChannelDelta8Bit: colorMetrics.MaxChannelDelta8Bit,
		},
		Crop: microFixturePicturePipelineCropStage{
			Basis:        "source crop uses DrawingML srcRect percentages; absent values default to zero per CT_RelativeRect",
			CropLeft:     element.CropLeft,
			CropTop:      element.CropTop,
			CropRight:    element.CropRight,
			CropBottom:   element.CropBottom,
			SourceBounds: objectPixelBoundsFromRect(source.Bounds()),
			CropBounds:   objectPixelBoundsFromRect(cropBounds),
			CroppedStats: microFixturePictureSourceStatsForImage(cropped),
		},
		Transform: microFixturePicturePipelineTransformStage{
			Basis:          "current transform stage applies flipH, flipV, and alphaModFix before sampling",
			FlipH:          element.FlipH,
			FlipV:          element.FlipV,
			AlphaModFixPct: element.ImageAlphaModFixPct,
			Applied:        element.FlipH || element.FlipV || shouldApplyImageAlphaModFix(element),
			OutputBounds:   objectPixelBoundsFromRect(pictureBounds),
			OutputStats:    microFixturePictureSourceStatsForImage(transformed),
		},
		Sampling: microFixturePicturePipelineSamplingStage{
			Basis:                         "current PNG path samples the transformed source into the integer EMU target with pictureScaler",
			Scaler:                        picturePipelineScalerName(pictureImage, pictureBounds),
			Canvas:                        microFixtureSize{Width: canvas.Dx(), Height: canvas.Dy()},
			AbsoluteTarget:                objectPixelBoundsFromRect(target),
			CropRelativeTarget:            objectPixelBoundsFromRect(target.Sub(image.Pt(outputCrop.MinX, outputCrop.MinY))),
			FractionalSourceBounds:        manifest.Object.FractionalBounds,
			SourceToTargetScaleX:          float64(pictureBounds.Dx()) / float64(max(1, target.Dx())),
			SourceToTargetScaleY:          float64(pictureBounds.Dy()) / float64(max(1, target.Dy())),
			PreOutputDiffAgainstGot:       compareImages(preOutputCrop, gotCrop),
			PreOutputDiffAgainstReference: compareImages(preOutputCrop, referenceCrop),
		},
		Output: microFixturePicturePipelineOutputStage{
			Basis:                "current renderer applies Display P3 conversion after all objects; visible fixture crops additionally blank later-object occlusions",
			DisplayP3Output:      true,
			OcclusionMaskApplied: len(occlusions) > 0,
			OccludedPixels:       occludedPixels,
			DiffAgainstGot:       compareImages(postOutputCrop, gotCrop),
			DiffAgainstReference: compareImages(postOutputCrop, referenceCrop),
		},
	}
	if partial != "" {
		artifact.SourceDecode.PartialFallback = partial
	}
	return artifact, nil
}

func objectPixelBoundsFromRect(rect image.Rectangle) ObjectPixelBounds {
	return ObjectPixelBounds{MinX: rect.Min.X, MinY: rect.Min.Y, MaxX: rect.Max.X - 1, MaxY: rect.Max.Y - 1}
}

func cropImageToRGBA(source image.Image, rect image.Rectangle) *image.RGBA {
	bounds := source.Bounds()
	rect = rect.Intersect(bounds)
	if rect.Empty() {
		return image.NewRGBA(image.Rectangle{})
	}
	output := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(output, output.Bounds(), source, rect.Min, draw.Src)
	return output
}

func applyPicturePipelineOcclusions(output *image.RGBA, crop ObjectPixelBounds, occlusions []microFixtureOcclusion) int {
	if len(occlusions) == 0 {
		return 0
	}
	occluded := 0
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := crop.MinX + x
			fullY := crop.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
				occluded++
			}
		}
	}
	return occluded
}

func picturePipelineScalerName(source image.Image, bounds image.Rectangle) string {
	if _, ok := source.(*image.YCbCr); ok && bounds.In(source.Bounds()) {
		return "catmull_rom"
	}
	return "approx_bilinear"
}

func pngColorTypeName(colorType int) string {
	switch colorType {
	case 0:
		return "grayscale"
	case 2:
		return "truecolor"
	case 3:
		return "indexed-color"
	case 4:
		return "grayscale-alpha"
	case 6:
		return "truecolor-alpha"
	default:
		return "unknown"
	}
}

func microFixturePictureResidualProfile(gotCropPath string, referenceCropPath string, source image.Image) (microFixturePictureResidualProfileArtifact, error) {
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return microFixturePictureResidualProfileArtifact{}, err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureResidualProfileArtifact{}, err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	artifact := microFixturePictureResidualProfileArtifact{
		Basis:      "classifies picture crop residual pixels by grayscale, hard black/white, and antialias edge coverage; diagnostic only",
		Source:     microFixturePictureSourceStatsForImage(source),
		CropWidth:  width,
		CropHeight: height,
	}
	gotLumaBuckets := map[int]int{}
	referenceLumaBuckets := map[int]int{}
	deltaLumaBuckets := map[int]int{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			artifact.DifferentPixels++
			includeImageDiffBounds(&artifact.DifferentBounds, x, y)
			gotGray := colorIsGrayscale(gotColor)
			referenceGray := colorIsGrayscale(referenceColor)
			gotLuma := colorLuma8(gotColor)
			referenceLuma := colorLuma8(referenceColor)
			gotLumaBuckets[gotLuma]++
			referenceLumaBuckets[referenceLuma]++
			deltaLumaBuckets[referenceLuma-gotLuma]++
			if gotGray && referenceGray {
				artifact.GrayscaleDifferentPixels++
				gotAntialias := lumaIsAntialias(gotLuma)
				referenceAntialias := lumaIsAntialias(referenceLuma)
				if gotAntialias || referenceAntialias {
					artifact.EdgeCoverageDifferentPixels++
				}
				if gotAntialias {
					artifact.GotAntialiasDifferentPixels++
				} else {
					artifact.GotHardDifferentPixels++
				}
				if referenceAntialias {
					artifact.ReferenceAntialiasDifferentPixels++
				} else {
					artifact.ReferenceHardDifferentPixels++
				}
				if lumaIsHardBlackWhite(gotLuma) && lumaIsHardBlackWhite(referenceLuma) {
					artifact.PureBlackWhiteDifferentPixels++
				}
			} else {
				artifact.ColoredDifferentPixels++
			}
		}
	}
	artifact.TopGotLumaBuckets = topMicroFixtureLumaBuckets(gotLumaBuckets, 12)
	artifact.TopReferenceLumaBuckets = topMicroFixtureLumaBuckets(referenceLumaBuckets, 12)
	artifact.TopReferenceMinusGotLumaBuckets = topMicroFixtureDeltaCountsFromMap(deltaLumaBuckets, 12)
	artifact.TopSourceColors = topMicroFixtureSourceColors(source, 12)
	return artifact, nil
}

func microFixturePictureSourceCorrespondenceProfile(gotCropPath string, referenceCropPath string, source image.Image, object objectFailureRecord) (microFixturePictureSourceCorrespondenceProfileArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, fmt.Errorf("picture source correspondence profile requires a picture object")
	}
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, err
	}
	sourceBounds := source.Bounds()
	if sourceBounds.Empty() {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, fmt.Errorf("picture source is empty")
	}
	targetModes := pictureResampleTargetModes(object)
	if len(targetModes) == 0 {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, fmt.Errorf("picture source correspondence profile requires a target mode")
	}
	targetMode := targetModes[0]
	targetWidth := targetMode.bounds.MaxX - targetMode.bounds.MinX + 1
	targetHeight := targetMode.bounds.MaxY - targetMode.bounds.MinY + 1
	if targetWidth <= 0 || targetHeight <= 0 {
		return microFixturePictureSourceCorrespondenceProfileArtifact{}, fmt.Errorf("invalid target bounds: %+v", targetMode.bounds)
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	artifact := microFixturePictureSourceCorrespondenceProfileArtifact{
		Basis:                "maps differing picture crop pixels to nearest source PNG pixels under the current round-target scale geometry; diagnostic only",
		Source:               microFixturePictureSourceStatsForImage(source),
		CropWidth:            width,
		CropHeight:           height,
		TargetMode:           targetMode.name,
		TargetRelativeBounds: targetMode.bounds,
	}
	sourceColorCounts := map[uint32]int{}
	sourceLumaCounts := map[int]int{}
	deltaLumaCounts := map[int]int{}
	sourceRowCounts := map[int]*microFixtureAxisDeltaCount{}
	sourceColumnCounts := map[int]*microFixtureAxisDeltaCount{}
	sourcePixelCounts := map[int64]*microFixtureSourcePixelResidualCount{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			sourceX, sourceY := nearestSourcePixelForPictureTarget(x, y, targetMode.bounds, targetWidth, targetHeight, sourceBounds)
			sourceColor := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+sourceX, sourceBounds.Min.Y+sourceY)).(color.RGBA)
			sourceLuma := colorLuma8(sourceColor)
			gotLuma := colorLuma8(gotColor)
			referenceLuma := colorLuma8(referenceColor)
			delta := referenceLuma - gotLuma
			mixed3x3 := sourceNeighborhoodHasMixedLuma(source, sourceBounds, sourceX, sourceY, 1)

			artifact.DifferentPixels++
			includeImageDiffBounds(&artifact.DifferentBounds, x, y)
			includeObjectPixelBounds(&artifact.SourceCoordinateBounds, sourceX, sourceY)
			sourceColorCounts[colorKey(sourceColor)]++
			sourceLumaCounts[sourceLuma]++
			deltaLumaCounts[delta]++
			addPictureAxisDelta(sourceRowCounts, sourceY, gotLuma, referenceLuma)
			addPictureAxisDelta(sourceColumnCounts, sourceX, gotLuma, referenceLuma)
			addPictureSourceCorrespondenceStats(&artifact, sourceLuma, gotLuma, referenceLuma, mixed3x3)
			addPictureSourcePixelResidual(sourcePixelCounts, sourceX, sourceY, sourceColor, sourceLuma, gotLuma, referenceLuma, mixed3x3)
		}
	}
	artifact.TopNearestSourceColors = topMicroFixtureColorCountsFromMap(sourceColorCounts, 12)
	artifact.TopNearestSourceLumaBuckets = topMicroFixtureLumaBuckets(sourceLumaCounts, 12)
	artifact.TopSourceRows = topMicroFixtureAxisDeltaCounts(sourceRowCounts, 20)
	artifact.TopSourceColumns = topMicroFixtureAxisDeltaCounts(sourceColumnCounts, 20)
	artifact.TopSourcePixels = topMicroFixtureSourcePixelResidualCounts(sourcePixelCounts, 20)
	artifact.TopReferenceMinusGotLumaBuckets = topMicroFixtureDeltaCountsFromMap(deltaLumaCounts, 20)
	return artifact, nil
}

func nearestSourcePixelForPictureTarget(x int, y int, target ObjectPixelBounds, targetWidth int, targetHeight int, sourceBounds image.Rectangle) (int, int) {
	sourceX := ((float64(x-target.MinX) + 0.5) * float64(sourceBounds.Dx()) / float64(targetWidth)) - 0.5
	sourceY := ((float64(y-target.MinY) + 0.5) * float64(sourceBounds.Dy()) / float64(targetHeight)) - 0.5
	return clampInt(int(math.Round(sourceX)), 0, sourceBounds.Dx()-1), clampInt(int(math.Round(sourceY)), 0, sourceBounds.Dy()-1)
}

func sourceNeighborhoodHasMixedLuma(source image.Image, bounds image.Rectangle, x int, y int, radius int) bool {
	if radius <= 0 {
		return false
	}
	var first *int
	for yy := max(0, y-radius); yy <= min(bounds.Dy()-1, y+radius); yy++ {
		for xx := max(0, x-radius); xx <= min(bounds.Dx()-1, x+radius); xx++ {
			luma := colorLuma8(color.RGBAModel.Convert(source.At(bounds.Min.X+xx, bounds.Min.Y+yy)).(color.RGBA))
			if first == nil {
				value := luma
				first = &value
				continue
			}
			if luma != *first {
				return true
			}
		}
	}
	return false
}

func addPictureSourceCorrespondenceStats(artifact *microFixturePictureSourceCorrespondenceProfileArtifact, sourceLuma int, gotLuma int, referenceLuma int, mixed3x3 bool) {
	if lumaIsHardBlackWhite(sourceLuma) {
		artifact.NearestSourceHardPixels++
	} else {
		artifact.NearestSourceAntialiasPixels++
	}
	switch sourceLuma {
	case 0:
		artifact.NearestSourceBlackPixels++
	case 255:
		artifact.NearestSourceWhitePixels++
	default:
		artifact.NearestSourceGrayPixels++
	}
	if mixed3x3 {
		artifact.Mixed3x3SourceNeighborhoodPixels++
	} else {
		artifact.Solid3x3SourceNeighborhoodPixels++
	}
	delta := referenceLuma - gotLuma
	artifact.ReferenceMinusGotLumaSum += delta
	if delta < 0 {
		artifact.ReferenceDarkerPixels++
	} else if delta > 0 {
		artifact.ReferenceLighterPixels++
	}
}

func addPictureSourcePixelResidual(counts map[int64]*microFixtureSourcePixelResidualCount, x int, y int, sourceColor color.RGBA, sourceLuma int, gotLuma int, referenceLuma int, mixed3x3 bool) {
	key := int64(y)<<32 | int64(x)
	count := counts[key]
	if count == nil {
		count = &microFixtureSourcePixelResidualCount{
			X:        x,
			Y:        y,
			RGBA:     colorKeyString(colorKey(sourceColor)),
			Luma:     sourceLuma,
			Mixed3x3: mixed3x3,
		}
		counts[key] = count
	}
	count.Count++
	if lumaIsHardBlackWhite(gotLuma) {
		count.GotHardPixels++
	} else {
		count.GotAntialiasPixels++
	}
	if lumaIsHardBlackWhite(referenceLuma) {
		count.ReferenceHardPixels++
	} else {
		count.ReferenceAntialiasPixels++
	}
	delta := referenceLuma - gotLuma
	count.ReferenceMinusGotLumaSum += delta
	if delta < 0 {
		count.ReferenceDarkerPixels++
	} else if delta > 0 {
		count.ReferenceLighterPixels++
	}
}

func microFixturePictureEdgeGeometryProfile(gotCropPath string, referenceCropPath string, object objectFailureRecord) (microFixturePictureEdgeGeometryProfileArtifact, error) {
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return microFixturePictureEdgeGeometryProfileArtifact{}, err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureEdgeGeometryProfileArtifact{}, err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	outputBounds := ObjectPixelBounds{}
	if object.OutputPixelBounds != nil {
		outputBounds = *object.OutputPixelBounds
	}
	artifact := microFixturePictureEdgeGeometryProfileArtifact{
		Basis:      "classifies picture residual pixels by crop-edge location, row/column concentration, and hard-vs-antialias luma state; diagnostic only",
		CropWidth:  width,
		CropHeight: height,
		TargetRelativeFractionalBounds: ObjectFloatBounds{
			MinX: object.FractionalBounds.MinX - float64(outputBounds.MinX),
			MinY: object.FractionalBounds.MinY - float64(outputBounds.MinY),
			MaxX: object.FractionalBounds.MaxX - float64(outputBounds.MinX),
			MaxY: object.FractionalBounds.MaxY - float64(outputBounds.MinY),
		},
		TargetRelativeOutputBounds: ObjectPixelBounds{
			MinX: outputBounds.MinX - outputBounds.MinX,
			MinY: outputBounds.MinY - outputBounds.MinY,
			MaxX: outputBounds.MaxX - outputBounds.MinX,
			MaxY: outputBounds.MaxY - outputBounds.MinY,
		},
	}
	rowCounts := map[int]*microFixtureAxisDeltaCount{}
	columnCounts := map[int]*microFixtureAxisDeltaCount{}
	deltaLumaBuckets := map[int]int{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			gotLuma := colorLuma8(gotColor)
			referenceLuma := colorLuma8(referenceColor)
			delta := referenceLuma - gotLuma
			artifact.DifferentPixels++
			includeImageDiffBounds(&artifact.DifferentBounds, x, y)
			if x == 0 {
				artifact.CropLeftEdgePixels++
			}
			if x == width-1 {
				artifact.CropRightEdgePixels++
			}
			if y == 0 {
				artifact.CropTopEdgePixels++
			}
			if y == height-1 {
				artifact.CropBottomEdgePixels++
			}
			if x <= 1 || y <= 1 || x >= width-2 || y >= height-2 {
				artifact.NearCropEdgePixels++
			} else {
				artifact.InteriorPixels++
			}
			addPictureAxisDelta(rowCounts, y, gotLuma, referenceLuma)
			addPictureAxisDelta(columnCounts, x, gotLuma, referenceLuma)
			addPictureEdgeGeometryStats(&artifact, gotLuma, referenceLuma)
			deltaLumaBuckets[delta]++
		}
	}
	artifact.TopRows = topMicroFixtureAxisDeltaCounts(rowCounts, 20)
	artifact.TopColumns = topMicroFixtureAxisDeltaCounts(columnCounts, 20)
	artifact.TopReferenceMinusGotLumaBuckets = topMicroFixtureDeltaCountsFromMap(deltaLumaBuckets, 20)
	return artifact, nil
}

func addPictureAxisDelta(counts map[int]*microFixtureAxisDeltaCount, index int, gotLuma int, referenceLuma int) {
	count := counts[index]
	if count == nil {
		count = &microFixtureAxisDeltaCount{Index: index}
		counts[index] = count
	}
	count.Count++
	if lumaIsHardBlackWhite(gotLuma) {
		count.GotHardPixels++
	} else {
		count.GotAntialiasPixels++
	}
	if lumaIsHardBlackWhite(referenceLuma) {
		count.ReferenceHardPixels++
	} else {
		count.ReferenceAntialiasPixels++
	}
	delta := referenceLuma - gotLuma
	count.ReferenceMinusGotLumaSum += delta
	if delta < 0 {
		count.ReferenceDarkerPixels++
	} else if delta > 0 {
		count.ReferenceLighterPixels++
	}
}

func addPictureEdgeGeometryStats(artifact *microFixturePictureEdgeGeometryProfileArtifact, gotLuma int, referenceLuma int) {
	if lumaIsHardBlackWhite(gotLuma) {
		artifact.GotHardPixels++
	} else {
		artifact.GotAntialiasPixels++
	}
	if lumaIsHardBlackWhite(referenceLuma) {
		artifact.ReferenceHardPixels++
	} else {
		artifact.ReferenceAntialiasPixels++
	}
	delta := referenceLuma - gotLuma
	artifact.ReferenceMinusGotLumaSum += delta
	if delta < 0 {
		artifact.ReferenceDarkerPixels++
	} else if delta > 0 {
		artifact.ReferenceLighterPixels++
	}
}

func topMicroFixtureAxisDeltaCounts(counts map[int]*microFixtureAxisDeltaCount, limit int) []microFixtureAxisDeltaCount {
	items := make([]microFixtureAxisDeltaCount, 0, len(counts))
	for _, count := range counts {
		items = append(items, *count)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Index < items[j].Index
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func firstAxisDeltaCount(counts []microFixtureAxisDeltaCount) microFixtureAxisDeltaCount {
	if len(counts) == 0 {
		return microFixtureAxisDeltaCount{}
	}
	return counts[0]
}

func firstColorCount(counts []microFixtureColorCount) microFixtureColorCount {
	if len(counts) == 0 {
		return microFixtureColorCount{}
	}
	return counts[0]
}

func searchMicroFixtureShapeFillHeight(got image.Image, reference image.Image, object objectFailureRecord) microFixtureShapeFillHeightSearchArtifact {
	baseline := compareImages(got, reference)
	artifact := microFixtureShapeFillHeightSearchArtifact{
		Basis:    "diagnostic only: replace current dominant fill-like pixels and optionally stop painting below candidate height while preserving current text coverage",
		Baseline: baseline,
	}
	gotColors := topMicroFixtureSourceColors(got, 8)
	referenceColors := topMicroFixtureSourceColors(reference, 8)
	if len(gotColors) == 0 {
		return artifact
	}
	currentFill, ok := parseObjectColorRGBA(gotColors[0].RGBA)
	if !ok {
		return artifact
	}
	fillCandidates := []color.RGBA{currentFill}
	seenFills := map[string]bool{formatObjectColor(currentFill): true}
	for _, item := range referenceColors {
		candidate, ok := parseObjectColorRGBA(item.RGBA)
		if !ok {
			continue
		}
		key := formatObjectColor(candidate)
		if seenFills[key] {
			continue
		}
		seenFills[key] = true
		fillCandidates = append(fillCandidates, candidate)
		if len(fillCandidates) >= 6 {
			break
		}
	}
	gotBounds := got.Bounds()
	outputHeight := gotBounds.Dy()
	heights := []int{outputHeight}
	if object.PixelBounds.MaxY >= object.PixelBounds.MinY {
		geometryHeight := object.PixelBounds.MaxY - object.PixelBounds.MinY + 1
		if geometryHeight > 0 && geometryHeight != outputHeight {
			heights = append(heights, geometryHeight)
		}
	}
	for _, height := range []int{outputHeight - 1, outputHeight - 2, outputHeight - 3, outputHeight - 4, outputHeight - 5, outputHeight - 6} {
		if height > 0 && !intInSlice(heights, height) {
			heights = append(heights, height)
		}
	}
	for _, fill := range fillCandidates {
		for _, height := range heights {
			candidate := renderShapeFillHeightCandidate(got, currentFill, fill, height)
			diff := compareImages(candidate, reference)
			artifact.Candidates = append(artifact.Candidates, microFixtureShapeFillHeightCandidate{
				Name:                          "dominant-fill-replacement",
				FillColor:                     formatObjectColor(fill),
				HeightPixels:                  height,
				DifferentPixels:               diff.DifferentPixels,
				DifferentBounds:               diff.DifferentBounds,
				TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
				MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
			})
		}
	}
	sort.Slice(artifact.Candidates, func(i int, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		if artifact.Candidates[i].HeightPixels != artifact.Candidates[j].HeightPixels {
			return artifact.Candidates[i].HeightPixels < artifact.Candidates[j].HeightPixels
		}
		return artifact.Candidates[i].FillColor < artifact.Candidates[j].FillColor
	})
	return artifact
}

func microFixtureShapeTextStrokeProfile(got image.Image, reference image.Image, textColor color.RGBA, fillColor color.RGBA) microFixtureShapeTextStrokeProfileArtifact {
	const textTolerance = 96
	const edgeBandPixels = 2
	artifact := microFixtureShapeTextStrokeProfileArtifact{
		Basis:             "diagnostic only: separates text-mask bounds from rectangle edge-band residuals and tests small vertical text-mask shifts",
		TextTolerance:     textTolerance,
		EdgeBandPixels:    edgeBandPixels,
		Baseline:          compareImages(got, reference),
		GotTextMask:       shapeTextMaskProfile(got, textColor, textTolerance),
		ReferenceTextMask: shapeTextMaskProfile(reference, textColor, textTolerance),
	}
	if artifact.GotTextMask.Bounds != nil && artifact.ReferenceTextMask.Bounds != nil {
		artifact.ReferenceTopMinusGotTop = artifact.ReferenceTextMask.Bounds.MinY - artifact.GotTextMask.Bounds.MinY
		gotCenter := (artifact.GotTextMask.Bounds.MinY + artifact.GotTextMask.Bounds.MaxY) / 2
		referenceCenter := (artifact.ReferenceTextMask.Bounds.MinY + artifact.ReferenceTextMask.Bounds.MaxY) / 2
		artifact.ReferenceCenterMinusGotCenter = referenceCenter - gotCenter
	}
	artifact.Edge = shapeEdgeProfile(got, reference, edgeBandPixels)
	artifact.TextLikeDifferentPixels, artifact.NonTextDifferentPixels = shapeTextLikeDiffCounts(got, reference, textColor, textTolerance)
	artifact.ShiftCandidates = shapeTextShiftCandidates(got, reference, textColor, fillColor, textTolerance)
	bestShift := 0
	if len(artifact.ShiftCandidates) > 0 {
		bestShift = artifact.ShiftCandidates[0].ShiftY
	}
	artifact.ReconstructionCandidates = shapeTextStrokeReconstructionCandidates(got, reference, textColor, fillColor, textTolerance, edgeBandPixels, bestShift)
	artifact.CoverageCandidates = shapeTextCoverageCandidates(got, reference, textColor, fillColor, textTolerance, edgeBandPixels, bestShift)
	return artifact
}

func shapeTextMaskProfile(img image.Image, textColor color.RGBA, tolerance int) microFixtureShapeMaskProfile {
	bounds := img.Bounds()
	rows := make([]int, bounds.Dy())
	columns := make([]int, bounds.Dx())
	profile := microFixtureShapeMaskProfile{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !shapeTextLikePixel(color.RGBAModel.Convert(img.At(x, y)).(color.RGBA), textColor, tolerance) {
				continue
			}
			profile.Pixels++
			rows[y-bounds.Min.Y]++
			columns[x-bounds.Min.X]++
			includeImageDiffBounds(&profile.Bounds, x-bounds.Min.X, y-bounds.Min.Y)
		}
	}
	profile.TopRows = topMicroFixtureAxisCounts(rows, 8)
	profile.TopColumns = topMicroFixtureAxisCounts(columns, 8)
	return profile
}

func shapeTextLikePixel(c color.RGBA, textColor color.RGBA, tolerance int) bool {
	if c.A < 240 {
		return false
	}
	return maxColorChannelDistance(c, textColor) <= tolerance
}

func shapeTextLikeDiffCounts(got image.Image, reference image.Image, textColor color.RGBA, tolerance int) (int, int) {
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	var textLike int
	var nonText int
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			if shapeTextLikePixel(gotColor, textColor, tolerance) || shapeTextLikePixel(referenceColor, textColor, tolerance) {
				textLike++
			} else {
				nonText++
			}
		}
	}
	return textLike, nonText
}

func shapeEdgeProfile(got image.Image, reference image.Image, band int) microFixtureShapeEdgeProfile {
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	rows := make([]int, height)
	columns := make([]int, width)
	profile := microFixtureShapeEdgeProfile{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			inTop := y < band
			inBottom := y >= height-band
			inLeft := x < band
			inRight := x >= width-band
			if !inTop && !inBottom && !inLeft && !inRight {
				continue
			}
			profile.DifferentPixels++
			rows[y]++
			columns[x]++
			if inTop {
				profile.TopBandPixels++
			}
			if inBottom {
				profile.BottomBandPixels++
			}
			if inLeft {
				profile.LeftBandPixels++
			}
			if inRight {
				profile.RightBandPixels++
			}
		}
	}
	profile.TopRows = topMicroFixtureAxisCounts(rows, 8)
	profile.TopColumns = topMicroFixtureAxisCounts(columns, 8)
	return profile
}

type shapeTextMaskPixel struct {
	x int
	y int
	c color.RGBA
}

func shapeTextShiftCandidates(got image.Image, reference image.Image, textColor color.RGBA, fillColor color.RGBA, tolerance int) []microFixtureShapeTextShiftCandidate {
	var candidates []microFixtureShapeTextShiftCandidate
	for shiftY := -8; shiftY <= 8; shiftY++ {
		candidateImage := renderShapeTextShiftCandidate(got, textColor, fillColor, tolerance, shiftY)
		diff := compareImages(candidateImage, reference)
		candidates = append(candidates, microFixtureShapeTextShiftCandidate{
			Name:                          fmt.Sprintf("text-mask-shift-y_%+d", shiftY),
			ShiftY:                        shiftY,
			DifferentPixels:               diff.DifferentPixels,
			DifferentBounds:               diff.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
		})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].ShiftY < candidates[j].ShiftY
	})
	if len(candidates) > 9 {
		return candidates[:9]
	}
	return candidates
}

func renderShapeTextShiftCandidate(source image.Image, textColor color.RGBA, fillColor color.RGBA, tolerance int, shiftY int) *image.RGBA {
	output := imageToRGBA(source)
	bounds := output.Bounds()
	var pixels []shapeTextMaskPixel
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := output.RGBAAt(x, y)
			if !shapeTextLikePixel(c, textColor, tolerance) {
				continue
			}
			pixels = append(pixels, shapeTextMaskPixel{x: x, y: y, c: c})
			output.SetRGBA(x, y, fillColor)
		}
	}
	for _, pixel := range pixels {
		y := pixel.y + shiftY
		if y < bounds.Min.Y || y >= bounds.Max.Y {
			continue
		}
		output.SetRGBA(pixel.x, y, pixel.c)
	}
	return output
}

func firstShapeTextShiftCandidate(candidates []microFixtureShapeTextShiftCandidate) microFixtureShapeTextShiftCandidate {
	if len(candidates) == 0 {
		return microFixtureShapeTextShiftCandidate{}
	}
	return candidates[0]
}

func shapeTextCoverageCandidates(got image.Image, reference image.Image, textColor color.RGBA, fillColor color.RGBA, tolerance int, edgeBand int, shiftY int) []microFixtureShapeTextCoverageCandidate {
	modes := []string{"none", "dilate4", "dilate8", "lighten4_35", "lighten8_35", "lighten8_50"}
	var candidates []microFixtureShapeTextCoverageCandidate
	for _, replaceEdge := range []bool{false, true} {
		for _, mode := range modes {
			candidateImage := renderShapeTextShiftCandidate(got, textColor, fillColor, tolerance, shiftY)
			applyShapeTextCoverageMode(candidateImage, textColor, fillColor, tolerance, mode)
			if replaceEdge {
				replaceShapeEdgeBandFromReference(candidateImage, reference, edgeBand)
			}
			diff := compareImages(candidateImage, reference)
			name := fmt.Sprintf("text-mask-shift-y_%+d/%s", shiftY, mode)
			if replaceEdge {
				name += "/oracle-edge-band"
			}
			candidates = append(candidates, microFixtureShapeTextCoverageCandidate{
				Name:                          name,
				ShiftY:                        shiftY,
				EdgeBandPixels:                edgeBandIf(replaceEdge, edgeBand),
				Mode:                          mode,
				DifferentPixels:               diff.DifferentPixels,
				DifferentBounds:               diff.DifferentBounds,
				TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
				MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
				TextMask:                      shapeTextMaskProfile(candidateImage, textColor, tolerance),
			})
		}
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].Name < candidates[j].Name
	})
	if len(candidates) > 12 {
		return candidates[:12]
	}
	return candidates
}

func applyShapeTextCoverageMode(img *image.RGBA, textColor color.RGBA, fillColor color.RGBA, tolerance int, mode string) {
	if img == nil || mode == "none" {
		return
	}
	bounds := img.Bounds()
	source := imageToRGBA(img)
	includeDiagonals := strings.Contains(mode, "8")
	lighten := strings.HasPrefix(mode, "lighten")
	amount := 1.0
	if mode == "lighten4_35" || mode == "lighten8_35" {
		amount = 0.35
	} else if mode == "lighten8_50" {
		amount = 0.50
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if shapeTextLikePixel(source.RGBAAt(x, y), textColor, tolerance) {
				continue
			}
			if !shapePixelAdjacentToText(source, x, y, textColor, tolerance, includeDiagonals) {
				continue
			}
			current := source.RGBAAt(x, y)
			if maxColorChannelDistance(current, fillColor) > 96 && !shapeTextLikePixel(current, textColor, tolerance*2) {
				continue
			}
			if lighten {
				img.SetRGBA(x, y, blendRGBA(current, textColor, amount))
			} else {
				img.SetRGBA(x, y, textColor)
			}
		}
	}
}

func shapePixelAdjacentToText(img *image.RGBA, x int, y int, textColor color.RGBA, tolerance int, includeDiagonals bool) bool {
	bounds := img.Bounds()
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if !includeDiagonals && dx != 0 && dy != 0 {
				continue
			}
			sx := x + dx
			sy := y + dy
			if sx < bounds.Min.X || sx >= bounds.Max.X || sy < bounds.Min.Y || sy >= bounds.Max.Y {
				continue
			}
			if shapeTextLikePixel(img.RGBAAt(sx, sy), textColor, tolerance) {
				return true
			}
		}
	}
	return false
}

func blendRGBA(left color.RGBA, right color.RGBA, amount float64) color.RGBA {
	if amount <= 0 {
		return left
	}
	if amount >= 1 {
		return right
	}
	return color.RGBA{
		R: floatChannelToByte(float64(left.R)*(1-amount) + float64(right.R)*amount),
		G: floatChannelToByte(float64(left.G)*(1-amount) + float64(right.G)*amount),
		B: floatChannelToByte(float64(left.B)*(1-amount) + float64(right.B)*amount),
		A: max(left.A, right.A),
	}
}

func shapeTextStrokeReconstructionCandidates(got image.Image, reference image.Image, textColor color.RGBA, fillColor color.RGBA, tolerance int, edgeBand int, shiftY int) []microFixtureShapeReconstructionCandidate {
	inputs := []struct {
		name        string
		shiftY      int
		replaceEdge bool
	}{
		{name: "current"},
		{name: fmt.Sprintf("text-mask-shift-y_%+d", shiftY), shiftY: shiftY},
		{name: "oracle-edge-band", replaceEdge: true},
		{name: fmt.Sprintf("text-mask-shift-y_%+d-plus-oracle-edge-band", shiftY), shiftY: shiftY, replaceEdge: true},
	}
	candidates := make([]microFixtureShapeReconstructionCandidate, 0, len(inputs))
	for _, input := range inputs {
		candidateImage := imageToRGBA(got)
		if input.shiftY != 0 {
			candidateImage = renderShapeTextShiftCandidate(candidateImage, textColor, fillColor, tolerance, input.shiftY)
		}
		if input.replaceEdge {
			replaceShapeEdgeBandFromReference(candidateImage, reference, edgeBand)
		}
		diff := compareImages(candidateImage, reference)
		candidates = append(candidates, microFixtureShapeReconstructionCandidate{
			Name:                          input.name,
			ShiftY:                        input.shiftY,
			EdgeBandPixels:                edgeBandIf(input.replaceEdge, edgeBand),
			DifferentPixels:               diff.DifferentPixels,
			DifferentBounds:               diff.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
		})
	}
	return candidates
}

func edgeBandIf(enabled bool, band int) int {
	if enabled {
		return band
	}
	return 0
}

func replaceShapeEdgeBandFromReference(target *image.RGBA, reference image.Image, band int) {
	if target == nil || band <= 0 {
		return
	}
	targetBounds := target.Bounds()
	referenceBounds := reference.Bounds()
	width := min(targetBounds.Dx(), referenceBounds.Dx())
	height := min(targetBounds.Dy(), referenceBounds.Dy())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if y >= band && y < height-band && x >= band && x < width-band {
				continue
			}
			c := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			target.SetRGBA(targetBounds.Min.X+x, targetBounds.Min.Y+y, c)
		}
	}
}

func shapeTextFontCandidates(got image.Image, reference image.Image, element slideElement, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds, textColor color.RGBA, fillColor color.RGBA) []microFixtureShapeTextFontCandidate {
	const textTolerance = 96
	families := shapeTextCandidateFamilies(element)
	target := elementPixelTarget(element, size, canvas)
	textRect := textBounds(target, element, size, canvas)
	relativeTextRect := image.Rect(
		textRect.Min.X-crop.MinX,
		textRect.Min.Y-crop.MinY,
		textRect.Max.X-crop.MinX,
		textRect.Max.Y-crop.MinY,
	)
	dpi := renderDPIForCanvas(size, canvas)
	var candidates []microFixtureShapeTextFontCandidate
	for _, family := range families {
		for _, shiftY := range []int{0, -2, -3, -4, -5} {
			candidateImage := renderShapeTextFontCandidate(got, element, family, relativeTextRect, dpi, textColor, fillColor, textTolerance, shiftY)
			diff := compareImages(candidateImage, reference)
			candidate := microFixtureShapeTextFontCandidate{
				Name:                          fmt.Sprintf("%s/shift-y_%+d", family, shiftY),
				FontFamily:                    family,
				ResolvedFont:                  resolvedShapeTextFontLabel(family, shapeElementBold(element), element.Italic),
				ShiftY:                        shiftY,
				DifferentPixels:               diff.DifferentPixels,
				DifferentBounds:               diff.DifferentBounds,
				TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
				MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
				TextMask:                      shapeTextMaskProfile(candidateImage, textColor, textTolerance),
			}
			candidates = append(candidates, candidate)
		}
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].Name < candidates[j].Name
	})
	if len(candidates) > 20 {
		return candidates[:20]
	}
	return candidates
}

func shapeTextCandidateFamilies(element slideElement) []string {
	seen := map[string]bool{}
	var families []string
	add := func(family string) {
		family = strings.TrimSpace(family)
		if family == "" || seen[strings.ToLower(family)] {
			return
		}
		seen[strings.ToLower(family)] = true
		families = append(families, family)
	}
	add(element.FontFamily)
	for _, paragraph := range element.TextParagraphs {
		add(paragraph.FontFamily)
		for _, run := range paragraph.Runs {
			add(run.FontFamily)
		}
	}
	for _, family := range []string{"Calibri", "Carlito", "Arial", "Helvetica Neue", "Helvetica", "Aptos"} {
		add(family)
	}
	return families
}

func renderShapeTextFontCandidate(source image.Image, element slideElement, family string, bounds image.Rectangle, dpi int, textColor color.RGBA, fillColor color.RGBA, tolerance int, shiftY int) *image.RGBA {
	output := imageToRGBA(source)
	imageBounds := output.Bounds()
	for y := imageBounds.Min.Y; y < imageBounds.Max.Y; y++ {
		for x := imageBounds.Min.X; x < imageBounds.Max.X; x++ {
			if shapeTextLikePixel(output.RGBAAt(x, y), textColor, tolerance) {
				output.SetRGBA(x, y, fillColor)
			}
		}
	}
	candidate := shapeElementWithFontFamily(element, family)
	_ = drawShapeTextWithDPI(output, bounds, candidate, dpi)
	if shiftY != 0 {
		output = renderShapeTextShiftCandidate(output, textColor, fillColor, tolerance, shiftY)
	}
	return output
}

func shapeTextAnchorMetrics(element slideElement, size slideSize, canvas image.Rectangle) (*microFixtureShapeTextAnchorMetrics, error) {
	target := elementPixelTarget(element, size, canvas)
	textTarget := target
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		if adjusted, err := shapeAutofitTarget(element, textTarget, size, canvas); err == nil {
			textTarget = adjusted
		}
	}
	bounds := textBounds(textTarget, element, size, canvas)
	dpi := renderDPIForCanvas(size, canvas)
	candidate := element
	if shouldFitNormalAutofit(candidate) {
		candidate = fitNormalAutofitElement(candidate, bounds, dpi)
	}
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return nil, err
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return nil, err
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return nil, err
	}
	measured, err := measureTextRenderLines(faces, lines, candidate.FontSize)
	if err != nil {
		return nil, err
	}
	currentHeight := measuredTextAnchorHeight(measured, candidate.TextAnchor)
	lineBoxHeight := measuredTextHeight(measured)
	currentTop := anchoredTextTop(bounds, currentHeight, candidate.TextAnchor)
	lineBoxTop := anchoredTextTop(bounds, lineBoxHeight, candidate.TextAnchor)
	metrics := &microFixtureShapeTextAnchorMetrics{
		DPI:                 dpi,
		TextAnchor:          candidate.TextAnchor,
		TextBounds:          pixelBoundsFromRect(bounds),
		FontFamily:          candidate.FontFamily,
		FontSize:            candidate.FontSize,
		LineCount:           len(measured),
		CurrentAnchorHeight: currentHeight,
		LineBoxAnchorHeight: lineBoxHeight,
		CurrentTop:          currentTop,
		LineBoxTop:          lineBoxTop,
		LineBoxShiftY:       lineBoxTop - currentTop,
		Lines:               make([]microFixtureShapeTextLineMetric, 0, len(measured)),
	}
	for _, line := range measured {
		metrics.Lines = append(metrics.Lines, microFixtureShapeTextLineMetric{
			Ascent:      line.Ascent,
			Descent:     line.Descent,
			Height:      line.Height,
			SpaceBefore: line.SpaceBefore,
			SpaceAfter:  line.SpaceAfter,
			HasText:     line.HasText,
		})
	}
	return metrics, nil
}

func shapeTextAnchorCandidates(got image.Image, reference image.Image, element slideElement, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds, textColor color.RGBA, fillColor color.RGBA, metrics microFixtureShapeTextAnchorMetrics) []microFixtureShapeTextAnchorCandidate {
	const textTolerance = 96
	textRect := image.Rect(
		metrics.TextBounds.MinX-crop.MinX,
		metrics.TextBounds.MinY-crop.MinY,
		metrics.TextBounds.MaxX-crop.MinX+1,
		metrics.TextBounds.MaxY-crop.MinY+1,
	)
	inputs := []struct {
		name   string
		shiftY int
	}{
		{name: "current-visible-anchor", shiftY: 0},
		{name: "line-box-anchor", shiftY: metrics.LineBoxShiftY},
	}
	dpi := metrics.DPI
	if dpi == 0 {
		dpi = renderDPIForCanvas(size, canvas)
	}
	candidates := make([]microFixtureShapeTextAnchorCandidate, 0, len(inputs))
	for _, input := range inputs {
		candidateImage := renderShapeTextAnchorCandidate(got, element, textRect, dpi, textColor, fillColor, textTolerance, input.shiftY)
		diff := compareImages(candidateImage, reference)
		candidates = append(candidates, microFixtureShapeTextAnchorCandidate{
			Name:                          input.name,
			ShiftY:                        input.shiftY,
			DifferentPixels:               diff.DifferentPixels,
			DifferentBounds:               diff.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
			TextMask:                      shapeTextMaskProfile(candidateImage, textColor, textTolerance),
		})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].Name < candidates[j].Name
	})
	return candidates
}

func renderShapeTextAnchorCandidate(source image.Image, element slideElement, bounds image.Rectangle, dpi int, textColor color.RGBA, fillColor color.RGBA, tolerance int, shiftY int) *image.RGBA {
	output := imageToRGBA(source)
	imageBounds := output.Bounds()
	for y := imageBounds.Min.Y; y < imageBounds.Max.Y; y++ {
		for x := imageBounds.Min.X; x < imageBounds.Max.X; x++ {
			if shapeTextLikePixel(output.RGBAAt(x, y), textColor, tolerance) {
				output.SetRGBA(x, y, fillColor)
			}
		}
	}
	if shiftY != 0 {
		bounds = bounds.Add(image.Pt(0, shiftY))
	}
	_ = drawShapeTextWithDPI(output, bounds, element, dpi)
	return output
}

func shapeParsedTextCandidates(source image.Image, reference image.Image, element slideElement, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds, textColor color.RGBA, fillColor color.RGBA) []microFixtureShapeParsedTextCandidate {
	target := elementPixelTarget(element, size, canvas)
	textTarget := target
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		if adjusted, err := shapeAutofitTarget(element, textTarget, size, canvas); err == nil {
			textTarget = adjusted
		}
	}
	inputs := []struct {
		name   string
		bounds image.Rectangle
	}{
		{name: "source-geometry-text-bounds", bounds: textBounds(target, element, size, canvas)},
		{name: "shape-autofit-text-bounds", bounds: textBounds(textTarget, element, size, canvas)},
	}
	dpi := renderDPIForCanvas(size, canvas)
	const textTolerance = 96
	seen := map[image.Rectangle]bool{}
	candidates := make([]microFixtureShapeParsedTextCandidate, 0, len(inputs))
	for _, input := range inputs {
		if input.bounds.Empty() || seen[input.bounds] {
			continue
		}
		seen[input.bounds] = true
		relativeBounds := image.Rect(
			input.bounds.Min.X-crop.MinX,
			input.bounds.Min.Y-crop.MinY,
			input.bounds.Max.X-crop.MinX,
			input.bounds.Max.Y-crop.MinY,
		)
		candidateImage := renderShapeParsedTextCandidate(source, element, relativeBounds, dpi, textColor, fillColor, textTolerance)
		diff := compareImages(candidateImage, reference)
		candidates = append(candidates, microFixtureShapeParsedTextCandidate{
			Name:                          input.name,
			TextBounds:                    pixelBoundsFromRect(input.bounds),
			DifferentPixels:               diff.DifferentPixels,
			DifferentBounds:               diff.DifferentBounds,
			TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
			MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
			TextMask:                      shapeTextMaskProfile(candidateImage, textColor, textTolerance),
		})
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].Name < candidates[j].Name
	})
	return candidates
}

func renderShapeParsedTextCandidate(source image.Image, element slideElement, bounds image.Rectangle, dpi int, textColor color.RGBA, fillColor color.RGBA, tolerance int) *image.RGBA {
	output := imageToRGBA(source)
	imageBounds := output.Bounds()
	for y := imageBounds.Min.Y; y < imageBounds.Max.Y; y++ {
		for x := imageBounds.Min.X; x < imageBounds.Max.X; x++ {
			if shapeTextLikePixel(output.RGBAAt(x, y), textColor, tolerance) {
				output.SetRGBA(x, y, fillColor)
			}
		}
	}
	_ = drawShapeTextWithDPI(output, bounds, element, dpi)
	return output
}

func shapeVectorBackendCandidates(got image.Image, reference image.Image, element slideElement, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds, occlusions []microFixtureOcclusion, outputDir string) ([]microFixtureShapeVectorBackendCandidate, error) {
	target := elementPixelTarget(element, size, canvas)
	targets := []struct {
		name   string
		bounds image.Rectangle
	}{
		{name: "source-geometry-target", bounds: target},
	}
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		if adjusted, err := shapeAutofitTarget(element, target, size, canvas); err == nil && adjusted != target {
			targets = append(targets, struct {
				name   string
				bounds image.Rectangle
			}{name: "shape-autofit-target", bounds: adjusted})
		}
	}
	candidates := make([]microFixtureShapeVectorBackendCandidate, 0, len(targets)*4)
	for _, target := range targets {
		for _, candidateSpec := range []struct {
			base   string
			layers string
		}{
			{base: "current-crop", layers: "fill-stroke-text"},
			{base: "white", layers: "fill"},
			{base: "white", layers: "fill-stroke"},
			{base: "white", layers: "fill-stroke-text"},
		} {
			candidateImage, ok := renderShapeVectorBackendCandidate(got, element, target.bounds, size, canvas, crop, occlusions, candidateSpec.base, candidateSpec.layers)
			if !ok {
				continue
			}
			diff := compareImages(candidateImage, reference)
			name := fmt.Sprintf("%s/%s/%s", target.name, candidateSpec.base, candidateSpec.layers)
			candidatePath := ""
			if outputDir != "" {
				candidatePath = filepath.Join(outputDir, "shape-vector-backend-"+sanitizeObjectArtifactName(name)+".png")
				if err := writePNG(candidatePath, candidateImage); err != nil {
					return nil, err
				}
			}
			candidates = append(candidates, microFixtureShapeVectorBackendCandidate{
				Name:                          name,
				Backend:                       "draw2d",
				Base:                          candidateSpec.base,
				Layers:                        candidateSpec.layers,
				Geometry:                      element.PrstGeom,
				ShapeBounds:                   pixelBoundsFromRect(target.bounds),
				DifferentPixels:               diff.DifferentPixels,
				DifferentBounds:               diff.DifferentBounds,
				TotalAbsoluteChannelDelta8Bit: diff.TotalAbsoluteChannelDelta8Bit,
				MaxChannelDelta8Bit:           diff.MaxChannelDelta8Bit,
				CandidatePath:                 candidatePath,
			})
		}
	}
	sort.Slice(candidates, func(i int, j int) bool {
		if candidates[i].DifferentPixels != candidates[j].DifferentPixels {
			return candidates[i].DifferentPixels < candidates[j].DifferentPixels
		}
		if candidates[i].TotalAbsoluteChannelDelta8Bit != candidates[j].TotalAbsoluteChannelDelta8Bit {
			return candidates[i].TotalAbsoluteChannelDelta8Bit < candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return candidates[i].Name < candidates[j].Name
	})
	return candidates, nil
}

func renderShapeVectorBackendCandidate(source image.Image, element slideElement, absoluteTarget image.Rectangle, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds, occlusions []microFixtureOcclusion, base string, layers string) (*image.RGBA, bool) {
	if absoluteTarget.Empty() || element.PrstGeom != "rect" {
		return nil, false
	}
	element = shapeElementWithDisplayP3Colors(element)
	output := shapeVectorBackendBaseImage(source, base)
	relativeTarget := image.Rect(
		absoluteTarget.Min.X-crop.MinX,
		absoluteTarget.Min.Y-crop.MinY,
		absoluteTarget.Max.X-crop.MinX,
		absoluteTarget.Max.Y-crop.MinY,
	)
	if relativeTarget.Empty() {
		return nil, false
	}
	gc := draw2dimg.NewGraphicContext(output)
	gc.SetDPI(renderDPIForCanvas(size, canvas))
	gc.SetLineJoin(draw2d.MiterJoin)
	if shapeVectorLayerEnabled(layers, "fill") && !element.NoFill && element.HasFill {
		gc.SetFillColor(element.FillColor)
		gc.Fill(draw2DRectPath(relativeTarget))
	}
	if shapeVectorLayerEnabled(layers, "stroke") && element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, canvas.Dx())
		if lineWidth > 0 {
			strokeRect := alignedStrokeRect(relativeTarget, lineWidth, element.LineAlign)
			gc.SetStrokeColor(element.LineColor)
			gc.SetLineWidth(float64(lineWidth))
			gc.SetLineCap(draw2DLineCap(element.LineCap))
			if element.LineDash != "" {
				gc.SetLineDash(draw2DLineDash(element.LineDash, lineWidth), 0)
			}
			gc.Stroke(draw2DRectPath(strokeRect))
		}
	}
	if shapeVectorLayerEnabled(layers, "text") && element.Text != "" && elementShouldRenderText(element) {
		absoluteTextBounds := textBounds(absoluteTarget, element, size, canvas)
		relativeTextBounds := image.Rect(
			absoluteTextBounds.Min.X-crop.MinX,
			absoluteTextBounds.Min.Y-crop.MinY,
			absoluteTextBounds.Max.X-crop.MinX,
			absoluteTextBounds.Max.Y-crop.MinY,
		)
		dpi := renderDPIForCanvas(size, canvas)
		rotation := normalizedRotationDegrees(element.Rotation)
		switch rotation {
		case 90, 180, 270:
			if err := drawRotatedShapeText(output, relativeTextBounds, element, rotation, dpi); err != nil {
				return nil, false
			}
		default:
			if err := drawShapeTextWithDPI(output, relativeTextBounds, element, dpi); err != nil {
				return nil, false
			}
		}
	}
	applyPictureCandidateOcclusions(output, crop, occlusions)
	return output, true
}

func shapeVectorBackendBaseImage(source image.Image, base string) *image.RGBA {
	switch base {
	case "white":
		bounds := source.Bounds()
		output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
		return output
	default:
		return imageToRGBA(source)
	}
}

func shapeVectorLayerEnabled(layers string, layer string) bool {
	switch layer {
	case "fill":
		return strings.Contains(layers, "fill")
	case "stroke":
		return strings.Contains(layers, "stroke")
	case "text":
		return strings.Contains(layers, "text")
	default:
		return false
	}
}

func draw2DRectPath(rect image.Rectangle) *draw2d.Path {
	path := &draw2d.Path{}
	path.MoveTo(float64(rect.Min.X), float64(rect.Min.Y))
	path.LineTo(float64(rect.Max.X), float64(rect.Min.Y))
	path.LineTo(float64(rect.Max.X), float64(rect.Max.Y))
	path.LineTo(float64(rect.Min.X), float64(rect.Max.Y))
	path.Close()
	return path
}

func draw2DLineCap(cap string) draw2d.LineCap {
	switch cap {
	case "rnd":
		return draw2d.RoundCap
	case "sq":
		return draw2d.SquareCap
	default:
		return draw2d.ButtCap
	}
}

func draw2DLineDash(dash string, width int) []float64 {
	pattern := lineDashPatternPixels(dash, width)
	if len(pattern) == 0 {
		return nil
	}
	out := make([]float64, 0, len(pattern))
	for _, value := range pattern {
		out = append(out, float64(value))
	}
	return out
}

func shapeElementWithDisplayP3Colors(element slideElement) slideElement {
	if element.HasFill {
		element.FillColor = displayP3RGBA(element.FillColor)
	}
	if element.HasLine {
		element.LineColor = displayP3RGBA(element.LineColor)
	}
	if element.HasTextColor {
		element.TextColor = displayP3RGBA(element.TextColor)
	}
	for paragraphIndex := range element.TextParagraphs {
		if element.TextParagraphs[paragraphIndex].HasTextColor {
			element.TextParagraphs[paragraphIndex].TextColor = displayP3RGBA(element.TextParagraphs[paragraphIndex].TextColor)
		}
		if element.TextParagraphs[paragraphIndex].HasBulletColor {
			element.TextParagraphs[paragraphIndex].BulletColor = displayP3RGBA(element.TextParagraphs[paragraphIndex].BulletColor)
		}
		for runIndex := range element.TextParagraphs[paragraphIndex].Runs {
			run := &element.TextParagraphs[paragraphIndex].Runs[runIndex]
			if run.HasTextColor {
				run.TextColor = displayP3RGBA(run.TextColor)
			}
			if run.HasUnderlineColor {
				run.UnderlineColor = displayP3RGBA(run.UnderlineColor)
			}
			if run.HasHighlightColor {
				run.HighlightColor = displayP3RGBA(run.HighlightColor)
			}
		}
	}
	return element
}

func displayP3RGBA(c color.RGBA) color.RGBA {
	r, g, b := srgbToDisplayP3(c.R, c.G, c.B)
	return color.RGBA{R: r, G: g, B: b, A: c.A}
}

func firstShapeVectorBackendCandidate(candidates []microFixtureShapeVectorBackendCandidate) microFixtureShapeVectorBackendCandidate {
	if len(candidates) == 0 {
		return microFixtureShapeVectorBackendCandidate{}
	}
	return candidates[0]
}

func shapeTextShapingProfile(manifestPath string, fixturePath string, element slideElement, size slideSize, canvas image.Rectangle, crop ObjectPixelBounds) (microFixtureShapeTextShapingProfileArtifact, error) {
	target := elementPixelTarget(element, size, canvas)
	textTarget := target
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		if adjusted, err := shapeAutofitTarget(element, textTarget, size, canvas); err == nil {
			textTarget = adjusted
		}
	}
	bounds := textBounds(textTarget, element, size, canvas)
	dpi := renderDPIForCanvas(size, canvas)
	candidate := element
	if shouldFitNormalAutofit(candidate) {
		candidate = fitNormalAutofitElement(candidate, bounds, dpi)
	}
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return microFixtureShapeTextShapingProfileArtifact{}, err
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return microFixtureShapeTextShapingProfileArtifact{}, err
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return microFixtureShapeTextShapingProfileArtifact{}, err
	}
	artifact := microFixtureShapeTextShapingProfileArtifact{
		ManifestPath: manifestPath,
		FixturePath:  fixturePath,
		Basis:        "diagnostic only: compare current x/image text segment advances with go-text HarfBuzz-shaped advances using Puppt-resolved font bytes",
		DPI:          dpi,
		TextBounds: imageToObjectPixelBounds(image.Rect(
			bounds.Min.X-crop.MinX,
			bounds.Min.Y-crop.MinY,
			bounds.Max.X-crop.MinX,
			bounds.Max.Y-crop.MinY,
		)),
		LineCount: len(lines),
		Lines:     make([]microFixtureTextShapingLine, 0, len(lines)),
	}
	for lineIndex, line := range lines {
		lineProfile := microFixtureTextShapingLine{Index: lineIndex}
		for _, segment := range line.Segments {
			if segment.Marker != "" || segment.Text == "" {
				continue
			}
			segmentProfile := shapeTextSegmentShapingProfile(candidate, faces, segment, line.TabStops, dpi)
			lineProfile.Segments = append(lineProfile.Segments, segmentProfile)
			artifact.SegmentCount++
			if delta := int(math.Round(math.Abs(segmentProfile.AdvanceDeltaPixels))); delta > artifact.MaxAdvanceDeltaPixels {
				artifact.MaxAdvanceDeltaPixels = delta
			}
		}
		artifact.Lines = append(artifact.Lines, lineProfile)
	}
	return artifact, nil
}

func shapeTextSegmentShapingProfile(element slideElement, faces *fontFaceCache, segment textLineSegment, tabStops []int, dpi int) microFixtureTextShapingSegment {
	family := segment.FontFamily
	if family == "" {
		family = element.FontFamily
	}
	fontSize := segment.FontSize
	if fontSize == 0 {
		fontSize = element.FontSize
	}
	defaultedFontSize := false
	if fontSize <= 0 {
		fontSize = 1800
		defaultedFontSize = true
	}
	bold := segment.Bold
	italic := segment.Italic
	profile := microFixtureTextShapingSegment{
		Text:              segment.Text,
		FontFamily:        family,
		FontSize:          fontSize,
		DefaultedFontSize: defaultedFontSize,
		Bold:              bold,
		Italic:            italic,
	}
	segmentFace, err := faces.GetForFamily(family, fontSize, bold, italic)
	if err != nil {
		profile.Error = err.Error()
		return profile
	}
	profile.CurrentWidthPixels = measureTextSegmentWithTabsAndSpacingAtDPI(faceWithSegmentKerning(segmentFace, segment), segment.Text, 0, dpi, tabStops, segment.CharSpacing)
	source, err := resolveFontSource(family, bold, italic)
	if err != nil {
		profile.Error = err.Error()
		return profile
	}
	profile.ResolvedFont = source.Label
	face, err := goTextFaceFromFontSource(source)
	if err != nil {
		profile.Error = err.Error()
		return profile
	}
	pointSize := fallbackFontPointSizeWithScaleAndFamily(fontSize, bold, italic, element.FontPointScale, family)
	pixelSize := pointSize * float64(normalizeOutputDPI(dpi)) / 72
	input := shaping.Input{
		Text:      []rune(segment.Text),
		RunStart:  0,
		RunEnd:    len([]rune(segment.Text)),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      fixed.Int26_6(math.Round(pixelSize * 64)),
		Script:    language.Latin,
		Language:  language.DefaultLanguage(),
	}
	output := (&shaping.HarfbuzzShaper{}).Shape(input)
	if spacing := textCharacterSpacingPixelsAtDPI(segment.CharSpacing, dpi); spacing != 0 {
		outputs := []shaping.Output{output}
		shaping.AddSpacing(outputs, input.Text, 0, fixed.I(spacing))
		output = outputs[0]
	}
	profile.ShapedAdvancePixels = fixedToFloat64(output.Advance)
	profile.AdvanceDeltaPixels = profile.ShapedAdvancePixels - float64(profile.CurrentWidthPixels)
	profile.GlyphCount = len(output.Glyphs)
	limit := minInt(len(output.Glyphs), 24)
	profile.Glyphs = make([]microFixtureTextShapingGlyph, 0, limit)
	for _, glyph := range output.Glyphs[:limit] {
		profile.Glyphs = append(profile.Glyphs, microFixtureTextShapingGlyph{
			GlyphID:        uint32(glyph.GlyphID),
			TextIndex:      glyph.TextIndex(),
			XAdvancePixels: fixedToFloat64(glyph.Advance),
			XOffsetPixels:  fixedToFloat64(glyph.XOffset),
			WidthPixels:    fixedToFloat64(glyph.Width),
			HeightPixels:   fixedToFloat64(glyph.Height),
		})
	}
	return profile
}

func goTextFaceFromFontSource(source fontSource) (*gtfont.Face, error) {
	faces, err := gtfont.ParseTTC(bytes.NewReader(source.Data))
	if err != nil {
		return nil, err
	}
	if len(faces) == 0 {
		return nil, errors.New("font collection has no faces")
	}
	return faces[0], nil
}

func fixedToFloat64(value fixed.Int26_6) float64 {
	return float64(value) / 64
}

func imageToObjectPixelBounds(rect image.Rectangle) ObjectPixelBounds {
	if rect.Empty() {
		return ObjectPixelBounds{}
	}
	return ObjectPixelBounds{
		MinX: rect.Min.X,
		MinY: rect.Min.Y,
		MaxX: rect.Max.X - 1,
		MaxY: rect.Max.Y - 1,
	}
}

func shapeElementWithFontFamily(element slideElement, family string) slideElement {
	element.FontFamily = family
	for paragraphIndex := range element.TextParagraphs {
		element.TextParagraphs[paragraphIndex].FontFamily = family
		for runIndex := range element.TextParagraphs[paragraphIndex].Runs {
			element.TextParagraphs[paragraphIndex].Runs[runIndex].FontFamily = family
		}
	}
	return element
}

func shapeElementBold(element slideElement) bool {
	for _, paragraph := range element.TextParagraphs {
		if paragraph.Bold {
			return true
		}
		for _, run := range paragraph.Runs {
			if run.Bold {
				return true
			}
		}
	}
	return false
}

func resolvedShapeTextFontLabel(family string, bold bool, italic bool) string {
	source, err := resolveFontSource(family, bold, italic)
	if err != nil {
		return err.Error()
	}
	return source.Label
}

func shapeResidualTextProfile(got image.Image, reference image.Image, textColor color.RGBA, fillColor color.RGBA) microFixtureShapeResidualTextProfile {
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	profile := microFixtureShapeResidualTextProfile{
		TextTolerance: 28,
		FillTolerance: 6,
	}
	differentRows := make([]int, height)
	differentColumns := make([]int, width)
	textRows := make([]int, height)
	textColumns := make([]int, width)
	gotColorCounts := map[uint32]int{}
	referenceColorCounts := map[uint32]int{}
	lumaDeltas := map[int]int{}
	textLumaDeltas := map[int]int{}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			if gotColor == referenceColor {
				continue
			}
			profile.DifferentPixels++
			differentRows[y]++
			differentColumns[x]++
			gotColorCounts[colorKey(gotColor)]++
			referenceColorCounts[colorKey(referenceColor)]++
			lumaDelta := colorLuma8(referenceColor) - colorLuma8(gotColor)
			lumaDeltas[lumaDelta]++
			gotText := maxColorChannelDistance(gotColor, textColor) <= profile.TextTolerance
			referenceText := maxColorChannelDistance(referenceColor, textColor) <= profile.TextTolerance
			gotFill := maxColorChannelDistance(gotColor, fillColor) <= profile.FillTolerance
			referenceFill := maxColorChannelDistance(referenceColor, fillColor) <= profile.FillTolerance
			gotWhite := maxColorChannelDistance(gotColor, white) <= profile.FillTolerance
			referenceWhite := maxColorChannelDistance(referenceColor, white) <= profile.FillTolerance
			if gotText {
				profile.GotTextLikeDifferentPixels++
			}
			if referenceText {
				profile.ReferenceTextLikeDifferentPixels++
			}
			if gotText || referenceText {
				profile.EitherTextLikeDifferentPixels++
				textRows[y]++
				textColumns[x]++
				textLumaDeltas[lumaDelta]++
			}
			if gotText && referenceText {
				profile.BothTextLikeDifferentPixels++
			}
			if gotFill {
				profile.GotFillLikeDifferentPixels++
			}
			if referenceFill {
				profile.ReferenceFillLikeDifferentPixels++
			}
			if gotWhite {
				profile.GotWhiteLikeDifferentPixels++
			}
			if referenceWhite {
				profile.ReferenceWhiteLikeDifferentPixels++
			}
			if !gotText && !gotFill && !gotWhite {
				profile.GotOtherDifferentPixels++
			}
			if !referenceText && !referenceFill && !referenceWhite {
				profile.ReferenceOtherDifferentPixels++
			}
		}
	}
	profile.TopDifferentRows = topMicroFixtureAxisCounts(differentRows, 8)
	profile.TopDifferentColumns = topMicroFixtureAxisCounts(differentColumns, 8)
	profile.TopDifferentGotColors = topMicroFixtureColorCountsFromMap(gotColorCounts, 12)
	profile.TopDifferentReferenceColors = topMicroFixtureColorCountsFromMap(referenceColorCounts, 12)
	profile.TopTextLikeRows = topMicroFixtureAxisCounts(textRows, 8)
	profile.TopTextLikeColumns = topMicroFixtureAxisCounts(textColumns, 8)
	profile.TopReferenceMinusGotLumaBuckets = topMicroFixtureDeltaCountsFromMap(lumaDeltas, 12)
	profile.TopReferenceMinusGotTextLumaBuckets = topMicroFixtureDeltaCountsFromMap(textLumaDeltas, 12)
	return profile
}

func dominantImageColor(img image.Image) (color.RGBA, bool) {
	colors := topMicroFixtureSourceColors(img, 1)
	if len(colors) == 0 {
		return color.RGBA{}, false
	}
	return parseObjectColorRGBA(colors[0].RGBA)
}

func dominantResidualTextColor(reference image.Image, fill color.RGBA) color.RGBA {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for _, item := range topMicroFixtureSourceColors(reference, 12) {
		candidate, ok := parseObjectColorRGBA(item.RGBA)
		if !ok {
			continue
		}
		if maxColorChannelDistance(candidate, fill) <= 6 || maxColorChannelDistance(candidate, white) <= 6 {
			continue
		}
		return candidate
	}
	return color.RGBA{}
}

func luminanceModifiersFromColorNode(node *xmlNode) (int64, int64) {
	mod := int64(100000)
	off := int64(0)
	for _, child := range node.Children {
		switch child.Name {
		case "lumMod":
			mod = mod * parsePercentAttr(child.Attrs, "val") / 100000
		case "lumOff":
			off += parsePercentAttr(child.Attrs, "val")
		}
	}
	return mod, off
}

func dominantReferenceFillColor(manifest microFixtureManifest) (color.RGBA, bool) {
	path := resolveTestArtifactPath(manifest.ReferenceCropPath)
	if manifest.ReferenceVisibleCropPath != "" {
		path = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
	}
	img, err := decodePNGFile(path)
	if err != nil {
		return color.RGBA{}, false
	}
	return dominantImageColor(img)
}

func dominantGotFillColor(manifest microFixtureManifest) (color.RGBA, bool) {
	path := resolveTestArtifactPath(manifest.GotCropPath)
	if manifest.GotVisibleCropPath != "" {
		path = resolveTestArtifactPath(manifest.GotVisibleCropPath)
	}
	img, err := decodePNGFile(path)
	if err != nil {
		return color.RGBA{}, false
	}
	return dominantImageColor(img)
}

func shapeLuminanceColorCandidates(base color.RGBA, mod int64, off int64, reference color.RGBA, got color.RGBA) []microFixtureShapeLuminanceColorCandidate {
	candidates := []struct {
		name string
		c    color.RGBA
	}{
		{name: "current-hsl", c: applyLuminanceModifier(base, mod, off)},
		{name: "encoded-rgb-round", c: applyEncodedRGBLuminance(base, mod, off, math.Round)},
		{name: "encoded-rgb-floor", c: applyEncodedRGBLuminance(base, mod, off, math.Floor)},
		{name: "encoded-rgb-ceil", c: applyEncodedRGBLuminance(base, mod, off, math.Ceil)},
		{name: "linear-rgb-round", c: applyLinearRGBLuminance(base, mod, off, math.Round)},
	}
	seen := map[string]bool{}
	output := make([]microFixtureShapeLuminanceColorCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.name + "/" + formatObjectColor(candidate.c)
		if seen[key] {
			continue
		}
		seen[key] = true
		r, g, b := srgbToDisplayP3(candidate.c.R, candidate.c.G, candidate.c.B)
		out := color.RGBA{R: r, G: g, B: b, A: candidate.c.A}
		output = append(output, microFixtureShapeLuminanceColorCandidate{
			Name:                      candidate.name,
			InternalColor:             formatObjectColor(candidate.c),
			OutputColor:               formatObjectColor(out),
			OutputDistanceToReference: maxColorChannelDistance(out, reference),
			OutputDistanceToGot:       maxColorChannelDistance(out, got),
		})
	}
	sort.Slice(output, func(i int, j int) bool {
		if output[i].OutputDistanceToReference != output[j].OutputDistanceToReference {
			return output[i].OutputDistanceToReference < output[j].OutputDistanceToReference
		}
		if output[i].OutputDistanceToGot != output[j].OutputDistanceToGot {
			return output[i].OutputDistanceToGot < output[j].OutputDistanceToGot
		}
		return output[i].Name < output[j].Name
	})
	return output
}

func applyEncodedRGBLuminance(c color.RGBA, mod int64, off int64, round func(float64) float64) color.RGBA {
	c.R = clampColor(int64(round(float64(c.R)*float64(mod)/100000 + 255*float64(off)/100000)))
	c.G = clampColor(int64(round(float64(c.G)*float64(mod)/100000 + 255*float64(off)/100000)))
	c.B = clampColor(int64(round(float64(c.B)*float64(mod)/100000 + 255*float64(off)/100000)))
	return c
}

func applyLinearRGBLuminance(c color.RGBA, mod int64, off int64, round func(float64) float64) color.RGBA {
	r := srgbByteToLinear(c.R)*float64(mod)/100000 + float64(off)/100000
	g := srgbByteToLinear(c.G)*float64(mod)/100000 + float64(off)/100000
	b := srgbByteToLinear(c.B)*float64(mod)/100000 + float64(off)/100000
	c.R = uint8(round(clampFloat(r, 0, 1) * 255))
	c.G = uint8(round(clampFloat(g, 0, 1) * 255))
	c.B = uint8(round(clampFloat(b, 0, 1) * 255))
	return c
}

func firstShapeLuminanceColorCandidate(candidates []microFixtureShapeLuminanceColorCandidate) microFixtureShapeLuminanceColorCandidate {
	if len(candidates) == 0 {
		return microFixtureShapeLuminanceColorCandidate{}
	}
	return candidates[0]
}

func renderShapeFillHeightCandidate(src image.Image, oldFill color.RGBA, newFill color.RGBA, height int) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if y >= height {
				dst.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				continue
			}
			c := color.RGBAModel.Convert(src.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.RGBA)
			if maxColorChannelDistance(c, oldFill) <= 3 {
				dst.SetRGBA(x, y, newFill)
			} else {
				dst.SetRGBA(x, y, c)
			}
		}
	}
	return dst
}

func maxColorChannelDistance(a color.RGBA, b color.RGBA) int {
	return max(max(absInt(int(a.R)-int(b.R)), absInt(int(a.G)-int(b.G))), max(absInt(int(a.B)-int(b.B)), absInt(int(a.A)-int(b.A))))
}

func intInSlice(values []int, needle int) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func includeObjectPixelBounds(bounds **ObjectPixelBounds, x int, y int) {
	if *bounds == nil {
		*bounds = &ObjectPixelBounds{MinX: x, MinY: y, MaxX: x, MaxY: y}
		return
	}
	if x < (*bounds).MinX {
		(*bounds).MinX = x
	}
	if y < (*bounds).MinY {
		(*bounds).MinY = y
	}
	if x > (*bounds).MaxX {
		(*bounds).MaxX = x
	}
	if y > (*bounds).MaxY {
		(*bounds).MaxY = y
	}
}

func microFixturePictureSourceStatsForImage(img image.Image) microFixturePictureSourceStats {
	if img == nil {
		return microFixturePictureSourceStats{}
	}
	bounds := img.Bounds()
	colors := map[uint32]int{}
	stats := microFixturePictureSourceStats{Width: bounds.Dx(), Height: bounds.Dy()}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA)
			colors[colorKey(c)]++
			if c.A == 255 {
				stats.OpaquePixels++
			} else {
				stats.AlphaPixels++
			}
		}
	}
	stats.UniqueColors = len(colors)
	return stats
}

func topMicroFixtureSourceColors(img image.Image, limit int) []microFixtureColorCount {
	if img == nil || limit <= 0 {
		return nil
	}
	counts := map[uint32]int{}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			counts[colorKey(color.RGBAModel.Convert(img.At(x, y)).(color.RGBA))]++
		}
	}
	items := make([]microFixtureColorCount, 0, len(counts))
	for key, count := range counts {
		items = append(items, microFixtureColorCount{RGBA: colorKeyString(key), Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].RGBA < items[j].RGBA
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func topMicroFixtureColorCountsFromMap(counts map[uint32]int, limit int) []microFixtureColorCount {
	items := make([]microFixtureColorCount, 0, len(counts))
	for key, count := range counts {
		items = append(items, microFixtureColorCount{RGBA: colorKeyString(key), Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].RGBA < items[j].RGBA
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func topMicroFixtureSourcePixelResidualCounts(counts map[int64]*microFixtureSourcePixelResidualCount, limit int) []microFixtureSourcePixelResidualCount {
	items := make([]microFixtureSourcePixelResidualCount, 0, len(counts))
	for _, count := range counts {
		items = append(items, *count)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		if items[i].Y != items[j].Y {
			return items[i].Y < items[j].Y
		}
		return items[i].X < items[j].X
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func topMicroFixtureLumaBuckets(counts map[int]int, limit int) []microFixtureLumaBucket {
	items := make([]microFixtureLumaBucket, 0, len(counts))
	for luma, count := range counts {
		items = append(items, microFixtureLumaBucket{Luma: luma, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Luma < items[j].Luma
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func topMicroFixtureDeltaCountsFromMap(counts map[int]int, limit int) []microFixtureDeltaCount {
	items := make([]microFixtureDeltaCount, 0, len(counts))
	for delta, count := range counts {
		items = append(items, microFixtureDeltaCount{Delta: delta, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Delta < items[j].Delta
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func colorIsGrayscale(c color.RGBA) bool {
	return c.R == c.G && c.G == c.B
}

func colorLuma8(c color.RGBA) int {
	return int((uint16(c.R) + uint16(c.G) + uint16(c.B)) / 3)
}

func lumaIsAntialias(luma int) bool {
	return luma > 0 && luma < 255
}

func lumaIsHardBlackWhite(luma int) bool {
	return luma == 0 || luma == 255
}

func colorKey(c color.RGBA) uint32 {
	return uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A)
}

func colorKeyString(key uint32) string {
	return fmt.Sprintf("#%02X%02X%02X/%02X", uint8(key>>24), uint8(key>>16), uint8(key>>8), uint8(key))
}

func searchMicroFixturePictureEdges(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureEdgeSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureEdgeSearchArtifact{}, fmt.Errorf("picture edge search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureEdgeSearchArtifact{}, fmt.Errorf("picture edge search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureEdgeSearchArtifact{}, err
	}
	artifact := microFixturePictureEdgeSearchArtifact{
		Basis: "candidate edge reconstruction filters for an attributed opaque black-white PNG icon, compared against the extracted object acceptance crop; diagnostic only",
	}
	scalers := []struct {
		name   string
		scaler xdraw.Scaler
	}{
		{name: "approx_bilinear", scaler: xdraw.ApproxBiLinear},
		{name: "bilinear", scaler: xdraw.BiLinear},
		{name: "catmull_rom", scaler: xdraw.CatmullRom},
	}
	sourceFilters := []string{"none", "source_box_1", "source_gaussian_1", "source_hard_edge_box_1", "source_hard_edge_gaussian_1"}
	outputFilters := []string{"none", "output_box_1", "output_gaussian_1", "output_hard_edge_box_1", "output_hard_edge_gaussian_1"}
	targets := pictureResampleTargetModes(object)
	for _, source := range sources {
		for _, sourceFilter := range sourceFilters {
			filteredSource := filteredPictureSource(source.Image, sourceFilter)
			for _, scaler := range scalers {
				for _, target := range targets {
					for _, outputFilter := range outputFilters {
						candidateImage := renderPictureEdgeCandidate(filteredSource, *object.OutputPixelBounds, target.bounds, scaler.scaler, outputFilter, occlusions)
						metrics := compareCandidateImage(reference, candidateImage)
						candidate := microFixturePictureEdgeCandidate{
							Name:                          source.Name + "/" + sourceFilter + "/" + scaler.name + "/" + target.name + "/" + outputFilter,
							SourceColor:                   source.Name,
							SourceFilter:                  sourceFilter,
							OutputFilter:                  outputFilter,
							Scaler:                        scaler.name,
							TargetRounding:                target.name,
							TargetOffset:                  target.bounds,
							DifferentPixels:               metrics.DifferentPixels,
							DifferentBounds:               metrics.DifferentBounds,
							TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
							MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
							ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
							ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
							ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
							ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
						}
						if source.Name == "converted_icc" && sourceFilter == "none" && scaler.name == "approx_bilinear" && target.name == "round" && outputFilter == "none" {
							baseline := candidate
							artifact.Baseline = &baseline
						}
						artifact.Candidates = append(artifact.Candidates, candidate)
					}
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		scaler := pictureResampleScalerByName(best.Scaler)
		if err := writePNG(filepath.Join(outputDir, "picture-edge-best.png"), renderPictureEdgeCandidate(filteredPictureSource(source, best.SourceFilter), *object.OutputPixelBounds, best.TargetOffset, scaler, best.OutputFilter, occlusions)); err != nil {
			return microFixturePictureEdgeSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureGamma(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureGammaSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureGammaSearchArtifact{}, fmt.Errorf("picture transfer search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureGammaSearchArtifact{}, fmt.Errorf("picture transfer search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureGammaSearchArtifact{}, err
	}
	artifact := microFixturePictureGammaSearchArtifact{
		Basis: "candidate transfer functions applied around picture resampling for an attributed opaque black-white PNG icon, compared against the extracted object acceptance crop; diagnostic only",
	}
	scalers := []struct {
		name   string
		scaler xdraw.Scaler
	}{
		{name: "approx_bilinear", scaler: xdraw.ApproxBiLinear},
		{name: "bilinear", scaler: xdraw.BiLinear},
		{name: "catmull_rom", scaler: xdraw.CatmullRom},
	}
	transferModes := []string{"srgb_byte", "linear_srgb", "gamma_18", "gamma_20", "gamma_22", "gamma_24"}
	targets := pictureResampleTargetModes(object)
	for _, source := range sources {
		for _, transferMode := range transferModes {
			for _, scaler := range scalers {
				for _, target := range targets {
					candidateImage := renderPictureGammaCandidate(source.Image, *object.OutputPixelBounds, target.bounds, scaler.scaler, transferMode, occlusions)
					metrics := compareCandidateImage(reference, candidateImage)
					candidate := microFixturePictureGammaCandidate{
						Name:                          source.Name + "/" + transferMode + "/" + scaler.name + "/" + target.name,
						SourceColor:                   source.Name,
						TransferMode:                  transferMode,
						Scaler:                        scaler.name,
						TargetRounding:                target.name,
						TargetOffset:                  target.bounds,
						DifferentPixels:               metrics.DifferentPixels,
						DifferentBounds:               metrics.DifferentBounds,
						TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
						MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
						ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
						ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
						ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
						ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
					}
					if source.Name == "converted_icc" && transferMode == "srgb_byte" && scaler.name == "approx_bilinear" && target.name == "round" {
						baseline := candidate
						artifact.Baseline = &baseline
					}
					artifact.Candidates = append(artifact.Candidates, candidate)
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		scaler := pictureResampleScalerByName(best.Scaler)
		if err := writePNG(filepath.Join(outputDir, "picture-gamma-best.png"), renderPictureGammaCandidate(source, *object.OutputPixelBounds, best.TargetOffset, scaler, best.TransferMode, occlusions)); err != nil {
			return microFixturePictureGammaSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureKernels(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureKernelSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureKernelSearchArtifact{}, fmt.Errorf("picture kernel search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureKernelSearchArtifact{}, fmt.Errorf("picture kernel search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureKernelSearchArtifact{}, err
	}
	artifact := microFixturePictureKernelSearchArtifact{
		Basis: "candidate scaler kernels and target endpoint rounding for an attributed opaque black-white PNG icon, compared against the extracted object acceptance crop; diagnostic only",
	}
	kernels := pictureKernelSearchModes()
	targets := pictureResampleTargetModes(object)
	for _, source := range sources {
		for _, kernel := range kernels {
			for _, target := range targets {
				candidateImage := renderPictureResampleCandidate(source.Image, *object.OutputPixelBounds, target.bounds, kernel.scaler, true, occlusions)
				metrics := compareCandidateImage(reference, candidateImage)
				candidate := microFixturePictureKernelCandidate{
					Name:                          source.Name + "/" + kernel.name + "/" + target.name,
					SourceColor:                   source.Name,
					Kernel:                        kernel.name,
					Support:                       kernel.support,
					TargetRounding:                target.name,
					TargetOffset:                  target.bounds,
					DifferentPixels:               metrics.DifferentPixels,
					DifferentBounds:               metrics.DifferentBounds,
					TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
					MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
					ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
					ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
					ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
					ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
				}
				if source.Name == "converted_icc" && kernel.name == "approx_bilinear" && target.name == "round" {
					baseline := candidate
					artifact.Baseline = &baseline
				}
				artifact.Candidates = append(artifact.Candidates, candidate)
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		scaler := pictureKernelSearchScalerByName(best.Kernel)
		if err := writePNG(filepath.Join(outputDir, "picture-kernel-best.png"), renderPictureResampleCandidate(source, *object.OutputPixelBounds, best.TargetOffset, scaler, true, occlusions)); err != nil {
			return microFixturePictureKernelSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureArea(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureAreaSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureAreaSearchArtifact{}, fmt.Errorf("picture area search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureAreaSearchArtifact{}, fmt.Errorf("picture area search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureAreaSearchArtifact{}, err
	}
	artifact := microFixturePictureAreaSearchArtifact{
		Basis: "candidate area-resampling modes for an attributed opaque black-white PNG icon, compared against the extracted object acceptance crop; diagnostic only",
	}
	areaModes := []string{"area_srgb_byte", "area_linear_srgb", "area_gamma_20"}
	targets := pictureResampleTargetModes(object)
	for _, source := range sources {
		for _, areaMode := range areaModes {
			for _, target := range targets {
				candidateImage := renderPictureAreaCandidate(source.Image, *object.OutputPixelBounds, target.bounds, areaMode, occlusions)
				metrics := compareCandidateImage(reference, candidateImage)
				candidate := microFixturePictureAreaCandidate{
					Name:                          source.Name + "/" + areaMode + "/" + target.name,
					SourceColor:                   source.Name,
					AreaMode:                      areaMode,
					TargetRounding:                target.name,
					TargetOffset:                  target.bounds,
					DifferentPixels:               metrics.DifferentPixels,
					DifferentBounds:               metrics.DifferentBounds,
					TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
					MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
					ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
					ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
					ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
					ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
				}
				if source.Name == "converted_icc" && areaMode == "area_srgb_byte" && target.name == "round" {
					baseline := candidate
					artifact.Baseline = &baseline
				}
				artifact.Candidates = append(artifact.Candidates, candidate)
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		if err := writePNG(filepath.Join(outputDir, "picture-area-best.png"), renderPictureAreaCandidate(source, *object.OutputPixelBounds, best.TargetOffset, best.AreaMode, occlusions)); err != nil {
			return microFixturePictureAreaSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePicturePhase(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePicturePhaseSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePicturePhaseSearchArtifact{}, fmt.Errorf("picture phase search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePicturePhaseSearchArtifact{}, fmt.Errorf("picture phase search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePicturePhaseSearchArtifact{}, err
	}
	artifact := microFixturePicturePhaseSearchArtifact{
		Basis: "candidate source and destination subpixel sampling phases for an attributed opaque black-white PNG icon, compared against the extracted object acceptance crop; diagnostic only",
	}
	phaseValues := []float64{-0.5, -0.25, 0, 0.25, 0.5}
	targets := picturePhaseTargetModes(object)
	for _, source := range sources {
		for _, target := range targets {
			for _, sourcePhaseX := range phaseValues {
				for _, sourcePhaseY := range phaseValues {
					for _, targetPhaseX := range phaseValues {
						for _, targetPhaseY := range phaseValues {
							candidateImage := renderPicturePhaseCandidate(source.Image, *object.OutputPixelBounds, target.bounds, sourcePhaseX, sourcePhaseY, targetPhaseX, targetPhaseY, occlusions)
							metrics := compareCandidateImage(reference, candidateImage)
							candidate := microFixturePicturePhaseCandidate{
								Name:                          fmt.Sprintf("%s/%s/src_%+.2f_%+.2f/dst_%+.2f_%+.2f", source.Name, target.name, sourcePhaseX, sourcePhaseY, targetPhaseX, targetPhaseY),
								SourceColor:                   source.Name,
								TargetRounding:                target.name,
								TargetOffset:                  target.bounds,
								SourcePhaseX:                  sourcePhaseX,
								SourcePhaseY:                  sourcePhaseY,
								TargetPhaseX:                  targetPhaseX,
								TargetPhaseY:                  targetPhaseY,
								DifferentPixels:               metrics.DifferentPixels,
								DifferentBounds:               metrics.DifferentBounds,
								TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
								MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
								ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
								ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
								ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
								ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
							}
							if source.Name == "converted_icc" && target.name == "round" && sourcePhaseX == 0 && sourcePhaseY == 0 && targetPhaseX == 0 && targetPhaseY == 0 {
								baseline := candidate
								artifact.Baseline = &baseline
							}
							artifact.Candidates = append(artifact.Candidates, candidate)
						}
					}
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		if err := writePNG(filepath.Join(outputDir, "picture-phase-best.png"), renderPicturePhaseCandidate(source, *object.OutputPixelBounds, best.TargetOffset, best.SourcePhaseX, best.SourcePhaseY, best.TargetPhaseX, best.TargetPhaseY, occlusions)); err != nil {
			return microFixturePicturePhaseSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureFractionalBounds(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureFractionalBoundsSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureFractionalBoundsSearchArtifact{}, fmt.Errorf("picture fractional-bounds search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureFractionalBoundsSearchArtifact{}, fmt.Errorf("picture fractional-bounds search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureFractionalBoundsSearchArtifact{}, err
	}
	rounded := pictureResampleTargetModes(object)[0].bounds
	fractional := pictureFractionalTargetBounds(object)
	artifact := microFixturePictureFractionalBoundsSearchArtifact{
		Basis: "candidate source-backed rendering against the object's actual fractional DrawingML bounds, compared against the extracted object acceptance crop; diagnostic only",
		TargetBounds: microFixturePictureFractionalBoundsTargetSummary{
			Fractional: fractional,
			Rounded:    rounded,
		},
	}
	current := renderPictureResampleCandidate(sources[0].Image, *object.OutputPixelBounds, rounded, xdraw.ApproxBiLinear, true, occlusions)
	currentMetrics := compareCandidateImage(reference, current)
	artifact.Baseline = &microFixturePictureFractionalBoundsCandidate{
		Name:                          sources[0].Name + "/current_approx_bilinear/round",
		SourceColor:                   sources[0].Name,
		Sampler:                       "current_approx_bilinear",
		TargetOffset:                  floatBoundsFromPixelBounds(rounded),
		DifferentPixels:               currentMetrics.DifferentPixels,
		DifferentBounds:               currentMetrics.DifferentBounds,
		TotalAbsoluteChannelDelta8Bit: currentMetrics.TotalAbsoluteChannelDelta8Bit,
		MaxChannelDelta8Bit:           currentMetrics.MaxChannelDelta8Bit,
		ReferenceDarkerPixels:         currentMetrics.ReferenceDarkerPixels,
		ReferenceLighterPixels:        currentMetrics.ReferenceLighterPixels,
		ReferenceRGBDeltaSum8Bit:      currentMetrics.ReferenceRGBDeltaSum8Bit,
		ReferenceRGBAbsoluteDelta8Bit: currentMetrics.ReferenceRGBAbsoluteDelta8Bit,
	}
	samplers := []struct {
		name    string
		samples int
		nearest bool
	}{
		{name: "nearest_center", samples: 1, nearest: true},
		{name: "bilinear_center", samples: 1},
		{name: "bilinear_2x", samples: 2},
		{name: "bilinear_3x", samples: 3},
		{name: "bilinear_4x", samples: 4},
		{name: "bilinear_8x", samples: 8},
	}
	for _, source := range sources {
		for _, sampler := range samplers {
			candidateImage := renderPictureFractionalBoundsCandidate(source.Image, *object.OutputPixelBounds, fractional, sampler.samples, sampler.nearest, occlusions)
			metrics := compareCandidateImage(reference, candidateImage)
			candidate := microFixturePictureFractionalBoundsCandidate{
				Name:                          source.Name + "/" + sampler.name,
				SourceColor:                   source.Name,
				Sampler:                       sampler.name,
				SamplesPerAxis:                sampler.samples,
				TargetOffset:                  fractional,
				DifferentPixels:               metrics.DifferentPixels,
				DifferentBounds:               metrics.DifferentBounds,
				TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
				MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
				ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
				ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
				ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
				ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
			}
			artifact.Candidates = append(artifact.Candidates, candidate)
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		if err := writePNG(filepath.Join(outputDir, "picture-fractional-bounds-best.png"), renderPictureFractionalBoundsCandidate(source, *object.OutputPixelBounds, best.TargetOffset, best.SamplesPerAxis, best.Sampler == "nearest_center", occlusions)); err != nil {
			return microFixturePictureFractionalBoundsSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureSourceModels(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureSourceModelSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureSourceModelSearchArtifact{}, fmt.Errorf("picture source model search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureSourceModelSearchArtifact{}, fmt.Errorf("picture source model search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureSourceModelSearchArtifact{}, err
	}
	artifact := microFixturePictureSourceModelSearchArtifact{
		Basis:        "candidate decoded source image models for an attributed paletted PNG icon, compared against the extracted object acceptance crop; diagnostic only",
		SourceModels: microFixturePictureSourceModelVariantSummaries(sources),
	}
	scalers := []struct {
		name   string
		scaler xdraw.Scaler
	}{
		{name: "approx_bilinear", scaler: xdraw.ApproxBiLinear},
		{name: "bilinear", scaler: xdraw.BiLinear},
		{name: "catmull_rom", scaler: xdraw.CatmullRom},
	}
	targets := pictureResampleTargetModes(object)
	for _, source := range sources {
		for _, scaler := range scalers {
			for _, target := range targets {
				candidateImage := renderPictureResampleCandidate(source.Image, *object.OutputPixelBounds, target.bounds, scaler.scaler, true, occlusions)
				metrics := compareCandidateImage(reference, candidateImage)
				candidate := microFixturePictureSourceModelCandidate{
					Name:                          source.Name + "/" + scaler.name + "/" + target.name,
					SourceModel:                   source.Name,
					Scaler:                        scaler.name,
					TargetRounding:                target.name,
					TargetOffset:                  target.bounds,
					DifferentPixels:               metrics.DifferentPixels,
					DifferentBounds:               metrics.DifferentBounds,
					TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
					MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
					ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
					ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
					ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
					ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
				}
				if source.Name == "converted_icc" && scaler.name == "approx_bilinear" && target.name == "round" {
					baseline := candidate
					artifact.Baseline = &baseline
				}
				artifact.Candidates = append(artifact.Candidates, candidate)
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceModel {
				source = candidateSource.Image
				break
			}
		}
		scaler := pictureResampleScalerByName(best.Scaler)
		if err := writePNG(filepath.Join(outputDir, "picture-source-model-best.png"), renderPictureResampleCandidate(source, *object.OutputPixelBounds, best.TargetOffset, scaler, true, occlusions)); err != nil {
			return microFixturePictureSourceModelSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func searchMicroFixturePictureContourCoverage(referenceCropPath string, source image.Image, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureContourCoverageSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureContourCoverageSearchArtifact{}, fmt.Errorf("picture contour coverage search requires a picture object")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureContourCoverageSearchArtifact{}, err
	}
	artifact := microFixturePictureContourCoverageSearchArtifact{
		Basis: "candidate reconstructs opaque grayscale picture icons as a thresholded source-luminance contour and supersamples destination coverage; diagnostic only",
	}
	targets := pictureResampleTargetModes(object)
	thresholds := []int{64, 96, 112, 128, 144, 160, 192}
	samples := []int{2, 3, 4, 6, 8}
	current := renderPictureResampleCandidate(source, *object.OutputPixelBounds, targets[0].bounds, xdraw.ApproxBiLinear, true, occlusions)
	currentMetrics := compareCandidateImage(reference, current)
	artifact.Baseline = &microFixturePictureContourCoverageCandidate{
		Name:                          "current_approx_bilinear",
		TargetRounding:                targets[0].name,
		TargetOffset:                  targets[0].bounds,
		DifferentPixels:               currentMetrics.DifferentPixels,
		DifferentBounds:               currentMetrics.DifferentBounds,
		TotalAbsoluteChannelDelta8Bit: currentMetrics.TotalAbsoluteChannelDelta8Bit,
		MaxChannelDelta8Bit:           currentMetrics.MaxChannelDelta8Bit,
		ReferenceDarkerPixels:         currentMetrics.ReferenceDarkerPixels,
		ReferenceLighterPixels:        currentMetrics.ReferenceLighterPixels,
		ReferenceRGBDeltaSum8Bit:      currentMetrics.ReferenceRGBDeltaSum8Bit,
		ReferenceRGBAbsoluteDelta8Bit: currentMetrics.ReferenceRGBAbsoluteDelta8Bit,
	}
	for _, target := range targets {
		for _, threshold := range thresholds {
			for _, sampleCount := range samples {
				candidateImage := renderPictureContourCoverageCandidate(source, *object.OutputPixelBounds, target.bounds, threshold, sampleCount, occlusions)
				metrics := compareCandidateImage(reference, candidateImage)
				candidate := microFixturePictureContourCoverageCandidate{
					Name:                          fmt.Sprintf("%s/threshold_%03d/%dx", target.name, threshold, sampleCount),
					TargetRounding:                target.name,
					TargetOffset:                  target.bounds,
					Threshold:                     threshold,
					SamplesPerAxis:                sampleCount,
					DifferentPixels:               metrics.DifferentPixels,
					DifferentBounds:               metrics.DifferentBounds,
					TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
					MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
					ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
					ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
					ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
					ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
				}
				artifact.Candidates = append(artifact.Candidates, candidate)
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		left, right := artifact.Candidates[i], artifact.Candidates[j]
		if left.DifferentPixels != right.DifferentPixels {
			return left.DifferentPixels < right.DifferentPixels
		}
		if left.TotalAbsoluteChannelDelta8Bit != right.TotalAbsoluteChannelDelta8Bit {
			return left.TotalAbsoluteChannelDelta8Bit < right.TotalAbsoluteChannelDelta8Bit
		}
		return left.Name < right.Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		if err := writePNG(filepath.Join(outputDir, "picture-contour-coverage-best.png"), renderPictureContourCoverageCandidate(source, *object.OutputPixelBounds, best.TargetOffset, best.Threshold, best.SamplesPerAxis, occlusions)); err != nil {
			return microFixturePictureContourCoverageSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

func picturePhaseTargetModes(object objectFailureRecord) []pictureResampleTargetMode {
	modes := pictureResampleTargetModes(object)
	if len(modes) <= 2 {
		return modes
	}
	return modes[:2]
}

func renderPictureContourCoverageCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, threshold int, samplesPerAxis int, occlusions []microFixtureOcclusion) *image.RGBA {
	output := image.NewRGBA(image.Rect(0, 0, crop.MaxX-crop.MinX+1, crop.MaxY-crop.MinY+1))
	draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	if source == nil || source.Bounds().Empty() || samplesPerAxis <= 0 {
		return output
	}
	targetRect := image.Rect(target.MinX, target.MinY, target.MaxX+1, target.MaxY+1).Intersect(output.Bounds())
	targetWidth := target.MaxX - target.MinX + 1
	targetHeight := target.MaxY - target.MinY + 1
	if targetRect.Empty() || targetWidth <= 0 || targetHeight <= 0 {
		return output
	}
	sourceBounds := source.Bounds()
	totalSamples := samplesPerAxis * samplesPerAxis
	for y := targetRect.Min.Y; y < targetRect.Max.Y; y++ {
		for x := targetRect.Min.X; x < targetRect.Max.X; x++ {
			inkSamples := 0
			for sy := 0; sy < samplesPerAxis; sy++ {
				sampleY := float64(y-target.MinY) + (float64(sy)+0.5)/float64(samplesPerAxis)
				sourceY := sampleY*float64(sourceBounds.Dy())/float64(targetHeight) - 0.5
				for sx := 0; sx < samplesPerAxis; sx++ {
					sampleX := float64(x-target.MinX) + (float64(sx)+0.5)/float64(samplesPerAxis)
					sourceX := sampleX*float64(sourceBounds.Dx())/float64(targetWidth) - 0.5
					if pictureBilinearSampleLuma(source, sourceBounds, sourceX, sourceY) < threshold {
						inkSamples++
					}
				}
			}
			coverage := float64(inkSamples) / float64(totalSamples)
			luma := floatChannelToByte(255 * (1 - coverage))
			output.SetRGBA(x, y, color.RGBA{R: luma, G: luma, B: luma, A: 255})
		}
	}
	applyPictureCandidateOcclusions(output, crop, occlusions)
	return output
}

func pictureBilinearSampleLuma(source image.Image, sourceBounds image.Rectangle, x float64, y float64) int {
	return colorLuma8(pictureBilinearSample(source, sourceBounds, x, y))
}

func renderPicturePhaseCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, sourcePhaseX float64, sourcePhaseY float64, targetPhaseX float64, targetPhaseY float64, occlusions []microFixtureOcclusion) *image.RGBA {
	output := image.NewRGBA(image.Rect(0, 0, crop.MaxX-crop.MinX+1, crop.MaxY-crop.MinY+1))
	draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	drawPicturePhaseResample(output, image.Rect(target.MinX, target.MinY, target.MaxX+1, target.MaxY+1), source, source.Bounds(), sourcePhaseX, sourcePhaseY, targetPhaseX, targetPhaseY)
	applyDisplayP3OutputTransform(output)
	applyPictureCandidateOcclusions(output, crop, occlusions)
	return output
}

func drawPicturePhaseResample(dst *image.RGBA, target image.Rectangle, source image.Image, sourceBounds image.Rectangle, sourcePhaseX float64, sourcePhaseY float64, targetPhaseX float64, targetPhaseY float64) {
	paintTarget := target.Intersect(dst.Bounds())
	if target.Empty() || paintTarget.Empty() || source == nil || sourceBounds.Empty() {
		return
	}
	fullTargetWidth := target.Dx()
	fullTargetHeight := target.Dy()
	if fullTargetWidth <= 0 || fullTargetHeight <= 0 {
		return
	}
	for y := paintTarget.Min.Y; y < paintTarget.Max.Y; y++ {
		sourceY := ((float64(y-target.Min.Y) + 0.5 + targetPhaseY) * float64(sourceBounds.Dy()) / float64(fullTargetHeight)) - 0.5 + sourcePhaseY
		for x := paintTarget.Min.X; x < paintTarget.Max.X; x++ {
			sourceX := ((float64(x-target.Min.X) + 0.5 + targetPhaseX) * float64(sourceBounds.Dx()) / float64(fullTargetWidth)) - 0.5 + sourcePhaseX
			dst.SetRGBA(x, y, pictureBilinearSample(source, sourceBounds, sourceX, sourceY))
		}
	}
}

func pictureBilinearSample(source image.Image, sourceBounds image.Rectangle, x float64, y float64) color.RGBA {
	x = clampFloat64(x, 0, float64(sourceBounds.Dx()-1))
	y = clampFloat64(y, 0, float64(sourceBounds.Dy()-1))
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := clampInt(x0+1, 0, sourceBounds.Dx()-1)
	y1 := clampInt(y0+1, 0, sourceBounds.Dy()-1)
	fx := x - float64(x0)
	fy := y - float64(y0)
	c00 := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+x0, sourceBounds.Min.Y+y0)).(color.RGBA)
	c10 := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+x1, sourceBounds.Min.Y+y0)).(color.RGBA)
	c01 := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+x0, sourceBounds.Min.Y+y1)).(color.RGBA)
	c11 := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+x1, sourceBounds.Min.Y+y1)).(color.RGBA)
	return color.RGBA{
		R: bilinearChannel(c00.R, c10.R, c01.R, c11.R, fx, fy),
		G: bilinearChannel(c00.G, c10.G, c01.G, c11.G, fx, fy),
		B: bilinearChannel(c00.B, c10.B, c01.B, c11.B, fx, fy),
		A: bilinearChannel(c00.A, c10.A, c01.A, c11.A, fx, fy),
	}
}

func bilinearChannel(c00 uint8, c10 uint8, c01 uint8, c11 uint8, fx float64, fy float64) uint8 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bottom := float64(c01)*(1-fx) + float64(c11)*fx
	return floatChannelToByte(top*(1-fy) + bottom*fy)
}

func pictureFractionalTargetBounds(object objectFailureRecord) ObjectFloatBounds {
	crop := *object.OutputPixelBounds
	return ObjectFloatBounds{
		MinX: object.FractionalBounds.MinX - float64(crop.MinX),
		MinY: object.FractionalBounds.MinY - float64(crop.MinY),
		MaxX: object.FractionalBounds.MaxX - float64(crop.MinX),
		MaxY: object.FractionalBounds.MaxY - float64(crop.MinY),
	}
}

func floatBoundsFromPixelBounds(bounds ObjectPixelBounds) ObjectFloatBounds {
	return ObjectFloatBounds{
		MinX: float64(bounds.MinX),
		MinY: float64(bounds.MinY),
		MaxX: float64(bounds.MaxX + 1),
		MaxY: float64(bounds.MaxY + 1),
	}
}

func renderPictureFractionalBoundsCandidate(source image.Image, crop ObjectPixelBounds, target ObjectFloatBounds, samplesPerAxis int, nearest bool, occlusions []microFixtureOcclusion) *image.RGBA {
	output := image.NewRGBA(image.Rect(0, 0, crop.MaxX-crop.MinX+1, crop.MaxY-crop.MinY+1))
	draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	if source == nil || source.Bounds().Empty() || target.MaxX <= target.MinX || target.MaxY <= target.MinY {
		return output
	}
	if samplesPerAxis <= 0 {
		samplesPerAxis = 1
	}
	sourceBounds := source.Bounds()
	targetWidth := target.MaxX - target.MinX
	targetHeight := target.MaxY - target.MinY
	totalSamples := samplesPerAxis * samplesPerAxis
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			var red float64
			var green float64
			var blue float64
			var alpha float64
			for sy := 0; sy < samplesPerAxis; sy++ {
				sampleY := float64(y) + (float64(sy)+0.5)/float64(samplesPerAxis)
				for sx := 0; sx < samplesPerAxis; sx++ {
					sampleX := float64(x) + (float64(sx)+0.5)/float64(samplesPerAxis)
					sample := color.RGBA{R: 255, G: 255, B: 255, A: 255}
					if sampleX >= target.MinX && sampleX < target.MaxX && sampleY >= target.MinY && sampleY < target.MaxY {
						sourceX := ((sampleX - target.MinX) * float64(sourceBounds.Dx()) / targetWidth) - 0.5
						sourceY := ((sampleY - target.MinY) * float64(sourceBounds.Dy()) / targetHeight) - 0.5
						if nearest {
							sourceX = math.Round(sourceX)
							sourceY = math.Round(sourceY)
						}
						sample = pictureBilinearSample(source, sourceBounds, sourceX, sourceY)
					}
					red += float64(sample.R)
					green += float64(sample.G)
					blue += float64(sample.B)
					alpha += float64(sample.A)
				}
			}
			output.SetRGBA(x, y, color.RGBA{
				R: floatChannelToByte(red / float64(totalSamples)),
				G: floatChannelToByte(green / float64(totalSamples)),
				B: floatChannelToByte(blue / float64(totalSamples)),
				A: floatChannelToByte(alpha / float64(totalSamples)),
			})
		}
	}
	applyDisplayP3OutputTransform(output)
	applyPictureCandidateOcclusions(output, crop, occlusions)
	return output
}

func renderPictureAreaCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, areaMode string, occlusions []microFixtureOcclusion) *image.RGBA {
	workingSource := source
	switch areaMode {
	case "area_linear_srgb":
		workingSource = pictureTransferToWorking(source, "linear_srgb")
	case "area_gamma_20":
		workingSource = pictureTransferToWorking(source, "gamma_20")
	}
	output := image.NewRGBA(image.Rect(0, 0, crop.MaxX-crop.MinX+1, crop.MaxY-crop.MinY+1))
	draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	drawPictureAreaResample(output, image.Rect(target.MinX, target.MinY, target.MaxX+1, target.MaxY+1), workingSource, workingSource.Bounds())
	switch areaMode {
	case "area_linear_srgb":
		applyPictureTransferFromWorking(output, "linear_srgb")
	case "area_gamma_20":
		applyPictureTransferFromWorking(output, "gamma_20")
	}
	applyDisplayP3OutputTransform(output)
	applyPictureCandidateOcclusions(output, crop, occlusions)
	return output
}

func drawPictureAreaResample(dst *image.RGBA, target image.Rectangle, source image.Image, sourceBounds image.Rectangle) {
	paintTarget := target.Intersect(dst.Bounds())
	if target.Empty() || paintTarget.Empty() || source == nil || sourceBounds.Empty() {
		return
	}
	fullTargetWidth := target.Dx()
	fullTargetHeight := target.Dy()
	if fullTargetWidth <= 0 || fullTargetHeight <= 0 {
		return
	}
	for y := paintTarget.Min.Y; y < paintTarget.Max.Y; y++ {
		sourceY0 := float64(y-target.Min.Y) * float64(sourceBounds.Dy()) / float64(fullTargetHeight)
		sourceY1 := float64(y+1-target.Min.Y) * float64(sourceBounds.Dy()) / float64(fullTargetHeight)
		for x := paintTarget.Min.X; x < paintTarget.Max.X; x++ {
			sourceX0 := float64(x-target.Min.X) * float64(sourceBounds.Dx()) / float64(fullTargetWidth)
			sourceX1 := float64(x+1-target.Min.X) * float64(sourceBounds.Dx()) / float64(fullTargetWidth)
			dst.SetRGBA(x, y, pictureAreaSample(source, sourceBounds, sourceX0, sourceY0, sourceX1, sourceY1))
		}
	}
}

func pictureAreaSample(source image.Image, sourceBounds image.Rectangle, sourceX0 float64, sourceY0 float64, sourceX1 float64, sourceY1 float64) color.RGBA {
	if sourceX1 <= sourceX0 || sourceY1 <= sourceY0 {
		return color.RGBA{}
	}
	minX := clampInt(int(math.Floor(sourceX0)), 0, sourceBounds.Dx()-1)
	maxX := clampInt(int(math.Ceil(sourceX1))-1, 0, sourceBounds.Dx()-1)
	minY := clampInt(int(math.Floor(sourceY0)), 0, sourceBounds.Dy()-1)
	maxY := clampInt(int(math.Ceil(sourceY1))-1, 0, sourceBounds.Dy()-1)
	var red float64
	var green float64
	var blue float64
	var alpha float64
	var total float64
	for y := minY; y <= maxY; y++ {
		overlapY := minFloat64(sourceY1, float64(y+1)) - maxFloat64(sourceY0, float64(y))
		if overlapY <= 0 {
			continue
		}
		for x := minX; x <= maxX; x++ {
			overlapX := minFloat64(sourceX1, float64(x+1)) - maxFloat64(sourceX0, float64(x))
			if overlapX <= 0 {
				continue
			}
			weight := overlapX * overlapY
			c := color.RGBAModel.Convert(source.At(sourceBounds.Min.X+x, sourceBounds.Min.Y+y)).(color.RGBA)
			red += float64(c.R) * weight
			green += float64(c.G) * weight
			blue += float64(c.B) * weight
			alpha += float64(c.A) * weight
			total += weight
		}
	}
	if total <= 0 {
		return color.RGBA{}
	}
	return color.RGBA{
		R: floatChannelToByte(red / total),
		G: floatChannelToByte(green / total),
		B: floatChannelToByte(blue / total),
		A: floatChannelToByte(alpha / total),
	}
}

func minFloat64(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func clampFloat64(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

type pictureKernelSearchMode struct {
	name    string
	support float64
	scaler  xdraw.Scaler
}

func pictureKernelSearchModes() []pictureKernelSearchMode {
	return []pictureKernelSearchMode{
		{name: "approx_bilinear", scaler: xdraw.ApproxBiLinear},
		{name: "bilinear", scaler: xdraw.BiLinear},
		{name: "catmull_rom", scaler: xdraw.CatmullRom},
		{name: "box_0_5", support: 0.5, scaler: pictureCustomKernel(0.5, func(float64) float64 { return 1 })},
		{name: "linear_0_75", support: 0.75, scaler: pictureLinearKernel(0.75)},
		{name: "linear_1_0", support: 1.0, scaler: pictureLinearKernel(1.0)},
		{name: "linear_1_25", support: 1.25, scaler: pictureLinearKernel(1.25)},
		{name: "bspline", support: 2.0, scaler: pictureCubicKernel(1, 0)},
		{name: "mitchell", support: 2.0, scaler: pictureCubicKernel(1.0/3.0, 1.0/3.0)},
		{name: "cubic_sharp", support: 2.0, scaler: pictureCubicKernel(0, 0.75)},
		{name: "lanczos2", support: 2.0, scaler: pictureLanczosKernel(2)},
		{name: "lanczos3", support: 3.0, scaler: pictureLanczosKernel(3)},
		{name: "gaussian_0_50", support: 2.0, scaler: pictureGaussianKernel(2, 0.50)},
		{name: "gaussian_0_75", support: 2.0, scaler: pictureGaussianKernel(2, 0.75)},
	}
}

func pictureKernelSearchScalerByName(name string) xdraw.Scaler {
	for _, mode := range pictureKernelSearchModes() {
		if mode.name == name {
			return mode.scaler
		}
	}
	return xdraw.ApproxBiLinear
}

func pictureCustomKernel(support float64, at func(float64) float64) *xdraw.Kernel {
	return &xdraw.Kernel{
		Support: support,
		At: func(t float64) float64 {
			t = math.Abs(t)
			if t >= support {
				return 0
			}
			return at(t)
		},
	}
}

func pictureLinearKernel(support float64) *xdraw.Kernel {
	return pictureCustomKernel(support, func(t float64) float64 {
		return 1 - t/support
	})
}

func pictureCubicKernel(b float64, c float64) *xdraw.Kernel {
	return pictureCustomKernel(2, func(t float64) float64 {
		if t < 1 {
			return ((12-9*b-6*c)*t*t*t + (-18+12*b+6*c)*t*t + (6 - 2*b)) / 6
		}
		return ((-b-6*c)*t*t*t + (6*b+30*c)*t*t + (-12*b-48*c)*t + (8*b + 24*c)) / 6
	})
}

func pictureLanczosKernel(a int) *xdraw.Kernel {
	support := float64(a)
	return pictureCustomKernel(support, func(t float64) float64 {
		if t == 0 {
			return 1
		}
		return pictureSinc(t) * pictureSinc(t/support)
	})
}

func pictureGaussianKernel(support float64, sigma float64) *xdraw.Kernel {
	return pictureCustomKernel(support, func(t float64) float64 {
		return math.Exp(-(t * t) / (2 * sigma * sigma))
	})
}

func pictureSinc(value float64) float64 {
	if value == 0 {
		return 1
	}
	value *= math.Pi
	return math.Sin(value) / value
}

func renderPictureGammaCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, scaler xdraw.Scaler, transferMode string, occlusions []microFixtureOcclusion) *image.RGBA {
	if transferMode == "srgb_byte" {
		return renderPictureResampleCandidate(source, crop, target, scaler, true, occlusions)
	}
	workingSource := pictureTransferToWorking(source, transferMode)
	output := renderPictureResampleCandidate(workingSource, crop, target, scaler, false, occlusions)
	applyPictureTransferFromWorking(output, transferMode)
	applyDisplayP3OutputTransform(output)
	return output
}

func pictureTransferToWorking(source image.Image, transferMode string) *image.RGBA {
	if source == nil {
		return image.NewRGBA(image.Rectangle{})
	}
	bounds := source.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			output.SetRGBA(x-bounds.Min.X, y-bounds.Min.Y, color.RGBA{
				R: pictureTransferChannelToWorking(c.R, transferMode),
				G: pictureTransferChannelToWorking(c.G, transferMode),
				B: pictureTransferChannelToWorking(c.B, transferMode),
				A: c.A,
			})
		}
	}
	return output
}

func applyPictureTransferFromWorking(img *image.RGBA, transferMode string) {
	if img == nil {
		return
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		offset := img.PixOffset(bounds.Min.X, y)
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.Pix[offset] = pictureTransferChannelFromWorking(img.Pix[offset], transferMode)
			img.Pix[offset+1] = pictureTransferChannelFromWorking(img.Pix[offset+1], transferMode)
			img.Pix[offset+2] = pictureTransferChannelFromWorking(img.Pix[offset+2], transferMode)
			offset += 4
		}
	}
}

func pictureTransferChannelToWorking(value uint8, transferMode string) uint8 {
	switch transferMode {
	case "linear_srgb":
		return floatUnitToByte(srgbByteToLinear(value))
	case "gamma_18":
		return floatUnitToByte(math.Pow(float64(value)/255, 1.8))
	case "gamma_20":
		return floatUnitToByte(math.Pow(float64(value)/255, 2.0))
	case "gamma_22":
		return floatUnitToByte(math.Pow(float64(value)/255, 2.2))
	case "gamma_24":
		return floatUnitToByte(math.Pow(float64(value)/255, 2.4))
	default:
		return value
	}
}

func pictureTransferChannelFromWorking(value uint8, transferMode string) uint8 {
	switch transferMode {
	case "linear_srgb":
		return linearToSRGBByte(float64(value) / 255)
	case "gamma_18":
		return floatUnitToByte(math.Pow(float64(value)/255, 1.0/1.8))
	case "gamma_20":
		return floatUnitToByte(math.Sqrt(float64(value) / 255))
	case "gamma_22":
		return floatUnitToByte(math.Pow(float64(value)/255, 1.0/2.2))
	case "gamma_24":
		return floatUnitToByte(math.Pow(float64(value)/255, 1.0/2.4))
	default:
		return value
	}
}

func floatUnitToByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}

func filteredPictureSource(source image.Image, filter string) image.Image {
	switch filter {
	case "source_box_1":
		return blurPictureImage(source, "box", 1)
	case "source_gaussian_1":
		return blurPictureImage(source, "gaussian", 1)
	case "source_hard_edge_box_1":
		return smoothHardPictureEdges(source, "box")
	case "source_hard_edge_gaussian_1":
		return smoothHardPictureEdges(source, "gaussian")
	default:
		return source
	}
}

func renderPictureEdgeCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, scaler xdraw.Scaler, outputFilter string, occlusions []microFixtureOcclusion) *image.RGBA {
	output := renderPictureResampleCandidate(source, crop, target, scaler, true, occlusions)
	switch outputFilter {
	case "output_box_1":
		return blurPictureImage(output, "box", 1)
	case "output_gaussian_1":
		return blurPictureImage(output, "gaussian", 1)
	case "output_hard_edge_box_1":
		return smoothHardPictureEdges(output, "box")
	case "output_hard_edge_gaussian_1":
		return smoothHardPictureEdges(output, "gaussian")
	default:
		return output
	}
}

func smoothHardPictureEdges(source image.Image, kernel string) *image.RGBA {
	base := imageToRGBA(source)
	blurred := blurPictureImage(base, kernel, 1)
	bounds := base.Bounds()
	output := image.NewRGBA(bounds)
	draw.Draw(output, bounds, base, bounds.Min, draw.Src)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			current := base.RGBAAt(x, y)
			if !colorIsGrayscale(current) || !lumaIsHardBlackWhite(colorLuma8(current)) {
				continue
			}
			if !pictureOutputNeighborhoodHasMixedLuma(base, x, y) {
				continue
			}
			output.SetRGBA(x, y, blurred.RGBAAt(x, y))
		}
	}
	return output
}

func pictureOutputNeighborhoodHasMixedLuma(img *image.RGBA, x int, y int) bool {
	bounds := img.Bounds()
	center := colorLuma8(img.RGBAAt(x, y))
	for yy := maxInt(bounds.Min.Y, y-1); yy < minInt(bounds.Max.Y, y+2); yy++ {
		for xx := maxInt(bounds.Min.X, x-1); xx < minInt(bounds.Max.X, x+2); xx++ {
			if colorLuma8(img.RGBAAt(xx, yy)) != center {
				return true
			}
		}
	}
	return false
}

func blurPictureImage(source image.Image, kernel string, radius int) *image.RGBA {
	if source == nil {
		return image.NewRGBA(image.Rectangle{})
	}
	if radius <= 0 {
		return imageToRGBA(source)
	}
	bounds := source.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}
	weights := pictureBlurKernel(kernel, radius)
	tmp := make([]float64, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			for offset := -radius; offset <= radius; offset++ {
				sampleX := clampInt(x+offset, 0, width-1)
				c := color.RGBAModel.Convert(source.At(bounds.Min.X+sampleX, bounds.Min.Y+y)).(color.RGBA)
				weight := weights[offset+radius]
				index := (y*width + x) * 4
				tmp[index] += float64(c.R) * weight
				tmp[index+1] += float64(c.G) * weight
				tmp[index+2] += float64(c.B) * weight
				tmp[index+3] += float64(c.A) * weight
			}
		}
	}
	output := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var channels [4]float64
			for offset := -radius; offset <= radius; offset++ {
				sampleY := clampInt(y+offset, 0, height-1)
				weight := weights[offset+radius]
				index := (sampleY*width + x) * 4
				channels[0] += tmp[index] * weight
				channels[1] += tmp[index+1] * weight
				channels[2] += tmp[index+2] * weight
				channels[3] += tmp[index+3] * weight
			}
			output.SetRGBA(x, y, color.RGBA{
				R: floatChannelToByte(channels[0]),
				G: floatChannelToByte(channels[1]),
				B: floatChannelToByte(channels[2]),
				A: floatChannelToByte(channels[3]),
			})
		}
	}
	return output
}

func pictureBlurKernel(name string, radius int) []float64 {
	if radius <= 0 {
		return []float64{1}
	}
	if name == "gaussian" {
		return gaussianKernelWithSigma(radius, max(0.5, float64(radius)/2))
	}
	kernel := make([]float64, radius*2+1)
	value := 1.0 / float64(len(kernel))
	for index := range kernel {
		kernel[index] = value
	}
	return kernel
}

func imageToRGBA(source image.Image) *image.RGBA {
	bounds := source.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), source, bounds.Min, draw.Src)
	return output
}

func imageToNRGBA(source image.Image) *image.NRGBA {
	bounds := source.Bounds()
	output := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(output, output.Bounds(), source, bounds.Min, draw.Src)
	return output
}

func floatChannelToByte(value float64) uint8 {
	if value <= 0 {
		return 0
	}
	if value >= 255 {
		return 255
	}
	return uint8(math.Round(value))
}

func clampInt(value int, minValue int, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func searchMicroFixturePictureResample(referenceCropPath string, sources []microFixturePictureSourceVariant, object objectFailureRecord, occlusions []microFixtureOcclusion, outputDir string) (microFixturePictureResampleSearchArtifact, error) {
	if object.OutputPixelBounds == nil || object.Kind != "pic" {
		return microFixturePictureResampleSearchArtifact{}, fmt.Errorf("picture resample search requires a picture object")
	}
	if len(sources) == 0 {
		return microFixturePictureResampleSearchArtifact{}, fmt.Errorf("picture resample search requires at least one source variant")
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixturePictureResampleSearchArtifact{}, err
	}
	artifact := microFixturePictureResampleSearchArtifact{
		Basis: "candidate picture source color handling, scalers, and target endpoint rounding compared against the extracted object acceptance crop; diagnostic only",
	}
	scalers := []struct {
		name   string
		scaler xdraw.Scaler
	}{
		{name: "nearest", scaler: xdraw.NearestNeighbor},
		{name: "approx_bilinear", scaler: xdraw.ApproxBiLinear},
		{name: "bilinear", scaler: xdraw.BiLinear},
		{name: "catmull_rom", scaler: xdraw.CatmullRom},
	}
	targets := pictureResampleTargetModes(object)
	outputModes := []string{"display_p3", "none"}
	for _, source := range sources {
		for _, scaler := range scalers {
			for _, target := range targets {
				for _, outputMode := range outputModes {
					candidateImage := renderPictureResampleCandidate(source.Image, *object.OutputPixelBounds, target.bounds, scaler.scaler, outputMode == "display_p3", occlusions)
					metrics := compareCandidateImage(reference, candidateImage)
					candidate := microFixturePictureResampleCandidate{
						Name:                          source.Name + "/" + scaler.name + "/" + target.name + "/" + outputMode,
						SourceColor:                   source.Name,
						OutputColor:                   outputMode,
						Scaler:                        scaler.name,
						TargetRounding:                target.name,
						TargetOffset:                  target.bounds,
						DifferentPixels:               metrics.DifferentPixels,
						DifferentBounds:               metrics.DifferentBounds,
						TotalAbsoluteChannelDelta8Bit: metrics.TotalAbsoluteChannelDelta8Bit,
						MaxChannelDelta8Bit:           metrics.MaxChannelDelta8Bit,
						ReferenceDarkerPixels:         metrics.ReferenceDarkerPixels,
						ReferenceLighterPixels:        metrics.ReferenceLighterPixels,
						ReferenceRGBDeltaSum8Bit:      metrics.ReferenceRGBDeltaSum8Bit,
						ReferenceRGBAbsoluteDelta8Bit: metrics.ReferenceRGBAbsoluteDelta8Bit,
					}
					if source.Name == "converted_icc" && scaler.name == "approx_bilinear" && target.name == "round" && outputMode == "display_p3" {
						baseline := candidate
						artifact.Baseline = &baseline
					}
					artifact.Candidates = append(artifact.Candidates, candidate)
				}
			}
		}
	}
	sort.Slice(artifact.Candidates, func(i, j int) bool {
		if artifact.Candidates[i].DifferentPixels != artifact.Candidates[j].DifferentPixels {
			return artifact.Candidates[i].DifferentPixels < artifact.Candidates[j].DifferentPixels
		}
		if artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit != artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit {
			return artifact.Candidates[i].TotalAbsoluteChannelDelta8Bit < artifact.Candidates[j].TotalAbsoluteChannelDelta8Bit
		}
		return artifact.Candidates[i].Name < artifact.Candidates[j].Name
	})
	if len(artifact.Candidates) > 0 && outputDir != "" {
		best := artifact.Candidates[0]
		scaler := pictureResampleScalerByName(best.Scaler)
		source := sources[0].Image
		for _, candidateSource := range sources {
			if candidateSource.Name == best.SourceColor {
				source = candidateSource.Image
				break
			}
		}
		if err := writePNG(filepath.Join(outputDir, "picture-resample-best.png"), renderPictureResampleCandidate(source, *object.OutputPixelBounds, best.TargetOffset, scaler, best.OutputColor == "display_p3", occlusions)); err != nil {
			return microFixturePictureResampleSearchArtifact{}, err
		}
	}
	if len(artifact.Candidates) > 20 {
		artifact.Candidates = artifact.Candidates[:20]
	}
	return artifact, nil
}

type pictureResampleTargetMode struct {
	name   string
	bounds ObjectPixelBounds
}

func pictureResampleTargetModes(object objectFailureRecord) []pictureResampleTargetMode {
	roundRect := ObjectPixelBounds{
		MinX: int(math.Round(object.FractionalBounds.MinX)),
		MinY: int(math.Round(object.FractionalBounds.MinY)),
		MaxX: int(math.Round(object.FractionalBounds.MaxX)) - 1,
		MaxY: int(math.Round(object.FractionalBounds.MaxY)) - 1,
	}
	floorCeilRect := ObjectPixelBounds{
		MinX: int(math.Floor(object.FractionalBounds.MinX)),
		MinY: int(math.Floor(object.FractionalBounds.MinY)),
		MaxX: int(math.Ceil(object.FractionalBounds.MaxX)) - 1,
		MaxY: int(math.Ceil(object.FractionalBounds.MaxY)) - 1,
	}
	floorFloorRect := ObjectPixelBounds{
		MinX: int(math.Floor(object.FractionalBounds.MinX)),
		MinY: int(math.Floor(object.FractionalBounds.MinY)),
		MaxX: int(math.Floor(object.FractionalBounds.MaxX)) - 1,
		MaxY: int(math.Floor(object.FractionalBounds.MaxY)) - 1,
	}
	ceilCeilRect := ObjectPixelBounds{
		MinX: int(math.Ceil(object.FractionalBounds.MinX)),
		MinY: int(math.Ceil(object.FractionalBounds.MinY)),
		MaxX: int(math.Ceil(object.FractionalBounds.MaxX)) - 1,
		MaxY: int(math.Ceil(object.FractionalBounds.MaxY)) - 1,
	}
	return []pictureResampleTargetMode{
		{name: "round", bounds: pictureResampleRelativeBounds(roundRect, *object.OutputPixelBounds)},
		{name: "floor_ceil", bounds: pictureResampleRelativeBounds(floorCeilRect, *object.OutputPixelBounds)},
		{name: "floor_floor", bounds: pictureResampleRelativeBounds(floorFloorRect, *object.OutputPixelBounds)},
		{name: "ceil_ceil", bounds: pictureResampleRelativeBounds(ceilCeilRect, *object.OutputPixelBounds)},
		{name: "round_shift_down", bounds: pictureResampleOffsetBounds(pictureResampleRelativeBounds(roundRect, *object.OutputPixelBounds), 0, 1)},
		{name: "round_shift_up", bounds: pictureResampleOffsetBounds(pictureResampleRelativeBounds(roundRect, *object.OutputPixelBounds), 0, -1)},
	}
}

func pictureResampleRelativeBounds(bounds ObjectPixelBounds, crop ObjectPixelBounds) ObjectPixelBounds {
	return ObjectPixelBounds{
		MinX: bounds.MinX - crop.MinX,
		MinY: bounds.MinY - crop.MinY,
		MaxX: bounds.MaxX - crop.MinX,
		MaxY: bounds.MaxY - crop.MinY,
	}
}

func pictureResampleOffsetBounds(bounds ObjectPixelBounds, dx int, dy int) ObjectPixelBounds {
	return ObjectPixelBounds{
		MinX: bounds.MinX + dx,
		MinY: bounds.MinY + dy,
		MaxX: bounds.MaxX + dx,
		MaxY: bounds.MaxY + dy,
	}
}

func renderPictureResampleCandidate(source image.Image, crop ObjectPixelBounds, target ObjectPixelBounds, scaler xdraw.Scaler, displayP3Output bool, occlusions []microFixtureOcclusion) *image.RGBA {
	output := image.NewRGBA(image.Rect(0, 0, crop.MaxX-crop.MinX+1, crop.MaxY-crop.MinY+1))
	draw.Draw(output, output.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	targetRect := image.Rect(target.MinX, target.MinY, target.MaxX+1, target.MaxY+1)
	scaler.Scale(output, targetRect, source, source.Bounds(), xdraw.Over, nil)
	if displayP3Output {
		applyDisplayP3OutputTransform(output)
	}
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := crop.MinX + x
			fullY := crop.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
	return output
}

func applyPictureCandidateOcclusions(output *image.RGBA, crop ObjectPixelBounds, occlusions []microFixtureOcclusion) {
	for y := output.Bounds().Min.Y; y < output.Bounds().Max.Y; y++ {
		for x := output.Bounds().Min.X; x < output.Bounds().Max.X; x++ {
			fullX := crop.MinX + x
			fullY := crop.MinY + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
			}
		}
	}
}

func pictureResampleScalerByName(name string) xdraw.Scaler {
	switch name {
	case "nearest":
		return xdraw.NearestNeighbor
	case "bilinear":
		return xdraw.BiLinear
	case "catmull_rom":
		return xdraw.CatmullRom
	default:
		return xdraw.ApproxBiLinear
	}
}

type microFixtureCandidateImageMetrics struct {
	DifferentPixels               int
	DifferentBounds               *imageDiffBounds
	TotalAbsoluteChannelDelta8Bit int64
	MaxChannelDelta8Bit           int
	ReferenceDarkerPixels         int
	ReferenceLighterPixels        int
	ReferenceRGBDeltaSum8Bit      int
	ReferenceRGBAbsoluteDelta8Bit int
}

func compareCandidateImage(reference image.Image, got image.Image) microFixtureCandidateImageMetrics {
	referenceBounds := reference.Bounds()
	gotBounds := got.Bounds()
	width := min(referenceBounds.Dx(), gotBounds.Dx())
	height := min(referenceBounds.Dy(), gotBounds.Dy())
	metrics := microFixtureCandidateImageMetrics{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr == wr && gg == wg && gb == wb && ga == wa {
				continue
			}
			metrics.DifferentPixels++
			includeImageDiffBounds(&metrics.DifferentBounds, x, y)
			metrics.addCandidateChannelDelta8Bit(gr, wr)
			metrics.addCandidateChannelDelta8Bit(gg, wg)
			metrics.addCandidateChannelDelta8Bit(gb, wb)
			metrics.addCandidateChannelDelta8Bit(ga, wa)
			delta := referenceBrightnessDelta8(gr, gg, gb, wr, wg, wb)
			metrics.ReferenceRGBDeltaSum8Bit += delta
			metrics.ReferenceRGBAbsoluteDelta8Bit += absInt(delta)
			if darker, lighter := referenceDeltaDirection(delta); darker {
				metrics.ReferenceDarkerPixels++
			} else if lighter {
				metrics.ReferenceLighterPixels++
			}
		}
	}
	return metrics
}

func (metrics *microFixtureCandidateImageMetrics) addCandidateChannelDelta8Bit(got uint32, want uint32) {
	delta := absInt(int(got>>8) - int(want>>8))
	metrics.TotalAbsoluteChannelDelta8Bit += int64(delta)
	if delta > metrics.MaxChannelDelta8Bit {
		metrics.MaxChannelDelta8Bit = delta
	}
}

func objectFloatPointsToPathPoints(points []ObjectFloatPoint) []pathPoint {
	path := make([]pathPoint, 0, len(points))
	for _, point := range points {
		path = append(path, pathPoint{X: point.X, Y: point.Y})
	}
	return path
}

type shadowPhaseSample struct {
	X              int
	Y              int
	ReferenceAlpha int
}

type shadowPhaseMask struct {
	bounds image.Rectangle
	alpha  []uint8
	width  int
}

func (mask shadowPhaseMask) alphaAt(x int, y int) uint8 {
	if !image.Pt(x, y).In(mask.bounds) {
		return 0
	}
	return mask.alpha[(y-mask.bounds.Min.Y)*mask.width+x-mask.bounds.Min.X]
}

func customPathShadowPhaseMask(canvas image.Rectangle, shapeBounds image.Rectangle, points []ObjectFloatPoint, alpha uint8, blur int, shiftX float64, shiftY float64, sampleX float64, sampleY float64) shadowPhaseMask {
	maskBounds := shapeBounds
	if blur > 0 {
		maskBounds = maskBounds.Inset(-blur)
	}
	maskBounds = maskBounds.Intersect(canvas)
	if maskBounds.Empty() || alpha == 0 {
		return shadowPhaseMask{}
	}
	polygon := make([]ObjectFloatPoint, 0, len(points))
	for _, point := range points {
		polygon = append(polygon, ObjectFloatPoint{
			X: float64(shapeBounds.Min.X) + point.X*float64(shapeBounds.Dx()) + shiftX,
			Y: float64(shapeBounds.Min.Y) + point.Y*float64(shapeBounds.Dy()) + shiftY,
		})
	}
	width := maskBounds.Dx()
	height := maskBounds.Dy()
	mask := make([]uint8, width*height)
	for y := maskBounds.Min.Y; y < maskBounds.Max.Y; y++ {
		for x := maskBounds.Min.X; x < maskBounds.Max.X; x++ {
			if pointInShadowPhasePolygon(float64(x)+sampleX, float64(y)+sampleY, polygon) {
				mask[(y-maskBounds.Min.Y)*width+x-maskBounds.Min.X] = alpha
			}
		}
	}
	if blur > 0 {
		mask = gaussianBlurAlpha(mask, width, height, blur)
	}
	return shadowPhaseMask{bounds: maskBounds, alpha: mask, width: width}
}

func pointInShadowPhasePolygon(x float64, y float64, polygon []ObjectFloatPoint) bool {
	inside := false
	j := len(polygon) - 1
	for i := range polygon {
		yi := polygon[i].Y
		yj := polygon[j].Y
		if (yi > y) != (yj > y) {
			xIntersect := (polygon[j].X-polygon[i].X)*(y-yi)/(yj-yi) + polygon[i].X
			if x < xIntersect {
				inside = !inside
			}
		}
		j = i
	}
	return inside
}

func parseObjectColorAlpha(value string) int {
	index := strings.LastIndex(value, "/")
	if index < 0 || index == len(value)-1 {
		return 0
	}
	alpha, err := strconv.ParseInt(value[index+1:], 16, 0)
	if err != nil || alpha < 0 {
		return 0
	}
	if alpha > 255 {
		return 255
	}
	return int(alpha)
}

func parseObjectColorRGBA(value string) (color.RGBA, bool) {
	if len(value) != len("#RRGGBB/AA") || value[0] != '#' || value[7] != '/' {
		return color.RGBA{}, false
	}
	red, err := strconv.ParseUint(value[1:3], 16, 8)
	if err != nil {
		return color.RGBA{}, false
	}
	green, err := strconv.ParseUint(value[3:5], 16, 8)
	if err != nil {
		return color.RGBA{}, false
	}
	blue, err := strconv.ParseUint(value[5:7], 16, 8)
	if err != nil {
		return color.RGBA{}, false
	}
	alpha, err := strconv.ParseUint(value[8:10], 16, 8)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: uint8(alpha)}, true
}

func writeMicroFixtureShadowRenderSummary(t *testing.T, deckInput string, renderPath string, object objectFailureRecord) *microFixtureShadowRenderSummary {
	t.Helper()
	if !object.ResolvedStyle.Shadow || object.ResolvedStyle.ShadowColor == "" {
		return nil
	}
	deckPath := realWorldDeckPath(deckInput)
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		t.Fatalf("open deck for shadow render summary %s object %s: %v", deckInput, object.CNvPrID, err)
	}
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	rendered, err := decodePNGFile(renderPath)
	if err != nil {
		t.Fatalf("decode render for shadow render summary %s object %s: %v", deckInput, object.CNvPrID, err)
	}
	summary, ok := microFixtureShadowRenderSummaryForCanvas(size, rendered.Bounds(), object)
	if !ok {
		return nil
	}
	return &summary
}

func microFixtureShadowRenderSummaryForCanvas(size slideSize, canvas image.Rectangle, object objectFailureRecord) (microFixtureShadowRenderSummary, bool) {
	if !object.ResolvedStyle.Shadow || object.PixelBounds == (ObjectPixelBounds{}) || size.CX <= 0 || size.CY <= 0 || canvas.Empty() {
		return microFixtureShadowRenderSummary{}, false
	}
	element := slideElement{
		ShadowDistance:  object.ResolvedStyle.ShadowDistance,
		ShadowDirection: object.ResolvedStyle.ShadowDirection,
		ShadowBlur:      object.ResolvedStyle.ShadowBlur,
	}
	offset := shadowOffset(element, size, canvas.Dx())
	blur := shadowBlurPixels(element, size, canvas.Dx())
	targetBounds := objectPixelBoundsToRect(object.PixelBounds).Intersect(canvas)
	if targetBounds.Empty() {
		return microFixtureShadowRenderSummary{}, false
	}
	shadowBounds := targetBounds.Add(offset)
	paintBounds := shadowBounds
	if blur > 0 {
		paintBounds = paintBounds.Inset(-blur)
	}
	paintBounds = paintBounds.Intersect(canvas)
	return microFixtureShadowRenderSummary{
		Basis:                       "renderer-derived shadow pixel geometry before mask rasterization; diagnostic only",
		Canvas:                      microFixtureSize{Width: canvas.Dx(), Height: canvas.Dy()},
		TargetBounds:                pixelBoundsFromRect(targetBounds),
		Offset:                      microFixturePoint{X: offset.X, Y: offset.Y},
		BlurPixels:                  blur,
		ShadowBounds:                pixelBoundsFromRect(shadowBounds),
		TargetCustomPathPixelPoints: customPathPixelPoints(targetBounds, object.ResolvedStyle.CustomPathCoordinates),
		ShadowCustomPathPixelPoints: customPathPixelPoints(shadowBounds, object.ResolvedStyle.CustomPathCoordinates),
		PaintBounds:                 pixelBoundsFromRect(paintBounds),
	}, true
}

func customPathPixelPoints(bounds image.Rectangle, points []ObjectFloatPoint) []microFixturePoint {
	if bounds.Empty() || len(points) == 0 {
		return nil
	}
	pixelPoints := make([]microFixturePoint, 0, len(points))
	for _, point := range points {
		pixelPoints = append(pixelPoints, microFixturePoint{
			X: bounds.Min.X + int(math.Round(point.X*float64(bounds.Dx()))),
			Y: bounds.Min.Y + int(math.Round(point.Y*float64(bounds.Dy()))),
		})
	}
	return pixelPoints
}

func objectPixelBoundsToRect(bounds ObjectPixelBounds) image.Rectangle {
	return image.Rect(bounds.MinX, bounds.MinY, bounds.MaxX+1, bounds.MaxY+1)
}

func averageRGB8(r uint32, g uint32, b uint32) int {
	return int((r>>8)+(g>>8)+(b>>8)) / 3
}

func estimateBlackOverlayAlpha8(backgroundLuma int, visibleLuma int) int {
	if backgroundLuma <= 0 || visibleLuma >= backgroundLuma {
		return 0
	}
	alpha := int(math.Round(float64(backgroundLuma-visibleLuma) * 255 / float64(backgroundLuma)))
	if alpha < 0 {
		return 0
	}
	if alpha > 255 {
		return 255
	}
	return alpha
}

func writeMicroFixtureUnderpaintChainArtifacts(t *testing.T, deckInput string, slideNumber int, microDir string, referenceCropPath string, visibleArtifacts microFixtureVisibleArtifacts, object objectFailureRecord, underpaints []microFixtureUnderpaint, attribution objectAttributionArtifact) microFixtureUnderpaintChainArtifacts {
	t.Helper()
	if object.OutputPixelBounds == nil || len(underpaints) == 0 {
		return microFixtureUnderpaintChainArtifacts{}
	}
	var chainObjects []objectFailureRecord
	for _, underpaint := range underpaints {
		if underpaint.Kind != "sp" {
			continue
		}
		underpaintObject, ok := findObjectFailureRecord(attribution.Objects, underpaint.Kind, underpaint.ZOrder, underpaint.CNvPrID, underpaint.CNvPrName)
		if !ok || underpaintObject.Bounds.CX <= 0 || underpaintObject.Bounds.CY <= 0 {
			continue
		}
		chainObjects = append(chainObjects, underpaintObject)
	}
	if len(chainObjects) == 0 {
		return microFixtureUnderpaintChainArtifacts{}
	}
	chainObjects = append(chainObjects, object)
	sort.SliceStable(chainObjects, func(i, j int) bool {
		if chainObjects[i].ZOrder != chainObjects[j].ZOrder {
			return chainObjects[i].ZOrder < chainObjects[j].ZOrder
		}
		return chainObjects[i].CNvPrID < chainObjects[j].CNvPrID
	})
	fixturePath := filepath.Join(microDir, "underpaint-chain-fixture.pptx")
	if err := writeShapeObjectsFixture(deckInput, fixturePath, object, chainObjects); err != nil {
		t.Fatalf("write underpaint-chain micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	renderPath := filepath.Join(microDir, "underpaint-chain-got.png")
	if _, err := renderMicroFixtureWithObjectDebug(fixturePath, renderPath, "", filepath.Join(microDir, "underpaint-chain-objects")); err != nil {
		t.Fatalf("render underpaint-chain micro-fixture for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	gotCropPath := filepath.Join(microDir, "underpaint-chain-got-crop.png")
	if err := writeCroppedPNG(renderPath, gotCropPath, *object.OutputPixelBounds); err != nil {
		t.Fatalf("write underpaint-chain got crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	diffPath := filepath.Join(microDir, "underpaint-chain-diff.json")
	diff, err := comparePNG(gotCropPath, referenceCropPath)
	if err != nil {
		t.Fatalf("compare underpaint-chain crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	if err := writeJSONFile(diffPath, realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write underpaint-chain diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	targetGotPath := gotCropPath
	targetReferencePath := referenceCropPath
	artifacts := microFixtureUnderpaintChainArtifacts{
		fixturePath: fixturePath,
		gotCropPath: gotCropPath,
		diffPath:    diffPath,
	}
	if visibleArtifacts.referencePath != "" {
		gotVisiblePath := filepath.Join(microDir, "underpaint-chain-got-visible-crop.png")
		if err := writeVisibleCroppedPNG(renderPath, gotVisiblePath, *object.OutputPixelBounds, visibleArtifacts.occlusions); err != nil {
			t.Fatalf("write underpaint-chain visible crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
		}
		visibleDiffPath := filepath.Join(microDir, "underpaint-chain-visible-diff.json")
		visibleDiff, err := comparePNG(gotVisiblePath, visibleArtifacts.referencePath)
		if err != nil {
			t.Fatalf("compare underpaint-chain visible crop for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
		}
		if err := writeJSONFile(visibleDiffPath, realWorldDiffArtifact{
			DeckInput:   deckInput,
			SlideNumber: slideNumber,
			Diff:        visibleDiff,
		}); err != nil {
			t.Fatalf("write underpaint-chain visible diff for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
		}
		artifacts.gotVisiblePath = gotVisiblePath
		artifacts.visibleDiffPath = visibleDiffPath
		targetGotPath = gotVisiblePath
		targetReferencePath = visibleArtifacts.referencePath
	}
	scope, err := microFixtureTargetScopeDiagnostic(targetGotPath, targetReferencePath, object, underpaints)
	if err != nil {
		t.Fatalf("analyze underpaint-chain target scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	scope.TargetCompared = filepath.Base(targetGotPath) + " vs " + filepath.Base(targetReferencePath)
	targetScopePath := filepath.Join(microDir, "underpaint-chain-target-scope.json")
	if err := writeJSONFile(targetScopePath, scope); err != nil {
		t.Fatalf("write underpaint-chain target scope for %s slide %d object %s: %v", deckInput, slideNumber, object.CNvPrID, err)
	}
	artifacts.targetScopePath = targetScopePath
	artifacts.targetScope = scope
	return artifacts
}

func microFixtureTargetScopeDiagnostic(gotCropPath string, referenceCropPath string, object objectFailureRecord, underpaints []microFixtureUnderpaint) (microFixtureTargetScope, error) {
	if object.OutputPixelBounds == nil || object.ObjectArtifactPath == "" {
		return microFixtureTargetScope{}, nil
	}
	got, err := decodePNGFile(gotCropPath)
	if err != nil {
		return microFixtureTargetScope{}, err
	}
	reference, err := decodePNGFile(referenceCropPath)
	if err != nil {
		return microFixtureTargetScope{}, err
	}
	mask, err := decodePNGFile(resolveTestArtifactPath(object.ObjectArtifactPath))
	if err != nil {
		return microFixtureTargetScope{}, err
	}
	underpaintMasks, err := loadUnderpaintMasks(underpaints)
	if err != nil {
		return microFixtureTargetScope{}, err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	maskBounds := mask.Bounds()
	width := min(gotBounds.Dx(), referenceBounds.Dx())
	height := min(gotBounds.Dy(), referenceBounds.Dy())
	scope := microFixtureTargetScope{
		Basis:          "current object artifact alpha mask; diagnostic only, not an acceptance mask",
		ObjectMaskPath: object.ObjectArtifactPath,
		CropPixels:     gotBounds.Dx() * gotBounds.Dy(),
		ComparedPixels: width * height,
	}
	differentRows := make([]int, height)
	differentColumns := make([]int, width)
	referenceDarkerRows := make([]int, height)
	referenceLighterRows := make([]int, height)
	referenceDarkerColumns := make([]int, width)
	referenceLighterColumns := make([]int, width)
	referenceDeltaCounts := make(map[int]int)
	differentGotColorCounts := make(map[uint32]int)
	differentReferenceColorCounts := make(map[uint32]int)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			fullX := object.OutputPixelBounds.MinX + x
			fullY := object.OutputPixelBounds.MinY + y
			insideMask := false
			partialAlphaMask := false
			alpha8 := uint32(0)
			partialAlphaMaskTone := 0
			if fullX >= 0 && fullY >= 0 && fullX < maskBounds.Dx() && fullY < maskBounds.Dy() {
				mr, mg, mb, alpha := mask.At(maskBounds.Min.X+fullX, maskBounds.Min.Y+fullY).RGBA()
				insideMask = alpha != 0
				partialAlphaMask = alpha != 0 && alpha != 0xffff
				alpha8 = alpha >> 8
				if partialAlphaMask {
					partialAlphaMaskTone = partialAlphaObjectMaskTone(mr, mg, mb, alpha)
				}
			}
			underpainted := pointUnderpainted(fullX, fullY, underpaintMasks)
			if insideMask {
				scope.ObjectMaskPixels++
				if partialAlphaMask {
					scope.ObjectMaskPartialAlphaPixels++
					switch {
					case alpha8 <= 80:
						scope.ObjectMaskLowAlphaPixels++
					case alpha8 <= 200:
						scope.ObjectMaskMidAlphaPixels++
					default:
						scope.ObjectMaskHighAlphaPixels++
					}
					switch partialAlphaMaskTone {
					case 1:
						scope.ObjectMaskPartialAlphaDarkPixels++
					case 2:
						scope.ObjectMaskPartialAlphaLightPixels++
					default:
						scope.ObjectMaskPartialAlphaOtherPixels++
					}
					if underpainted {
						scope.ObjectMaskPartialAlphaPixelsOverUnderpaint++
					}
				} else {
					scope.ObjectMaskFullAlphaPixels++
				}
			}
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr == wr && gg == wg && gb == wb && ga == wa {
				continue
			}
			gotColor := color.RGBAModel.Convert(got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y)).(color.RGBA)
			referenceColor := color.RGBAModel.Convert(reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y)).(color.RGBA)
			differentGotColorCounts[colorKey(gotColor)]++
			differentReferenceColorCounts[colorKey(referenceColor)]++
			scope.DifferentPixels++
			includeImageDiffBounds(&scope.DifferentBounds, x, y)
			differentRows[y]++
			differentColumns[x]++
			referenceDelta8 := referenceBrightnessDelta8(gr, gg, gb, wr, wg, wb)
			scope.ReferenceRGBDeltaSum8 += referenceDelta8
			scope.ReferenceRGBAbsoluteDeltaSum8 += absInt(referenceDelta8)
			referenceDeltaCounts[referenceDelta8]++
			referenceDarker, referenceLighter := referenceDeltaDirection(referenceDelta8)
			if referenceDarker {
				scope.DifferentPixelsReferenceDarker++
				includeImageDiffBounds(&scope.ReferenceDarkerBounds, x, y)
				referenceDarkerRows[y]++
				referenceDarkerColumns[x]++
			} else if referenceLighter {
				scope.DifferentPixelsReferenceLighter++
				includeImageDiffBounds(&scope.ReferenceLighterBounds, x, y)
				referenceLighterRows[y]++
				referenceLighterColumns[x]++
			}
			if insideMask {
				scope.DifferentPixelsInsideObjectMask++
				if referenceDarker {
					scope.DifferentPixelsInsideObjectMaskReferenceDarker++
				} else if referenceLighter {
					scope.DifferentPixelsInsideObjectMaskReferenceLighter++
				}
				if partialAlphaMask {
					scope.DifferentPixelsInsidePartialAlphaObjectMask++
					if referenceDarker {
						scope.DifferentPixelsInsidePartialAlphaObjectMaskReferenceDarker++
					} else if referenceLighter {
						scope.DifferentPixelsInsidePartialAlphaObjectMaskReferenceLighter++
					}
					switch {
					case alpha8 <= 80:
						scope.DifferentPixelsInsideLowAlphaObjectMask++
						if referenceDarker {
							scope.DifferentPixelsInsideLowAlphaObjectMaskReferenceDarker++
						} else if referenceLighter {
							scope.DifferentPixelsInsideLowAlphaObjectMaskReferenceLighter++
						}
					case alpha8 <= 200:
						scope.DifferentPixelsInsideMidAlphaObjectMask++
						if referenceDarker {
							scope.DifferentPixelsInsideMidAlphaObjectMaskReferenceDarker++
						} else if referenceLighter {
							scope.DifferentPixelsInsideMidAlphaObjectMaskReferenceLighter++
						}
					default:
						scope.DifferentPixelsInsideHighAlphaObjectMask++
						if referenceDarker {
							scope.DifferentPixelsInsideHighAlphaObjectMaskReferenceDarker++
						} else if referenceLighter {
							scope.DifferentPixelsInsideHighAlphaObjectMaskReferenceLighter++
						}
					}
					switch partialAlphaMaskTone {
					case 1:
						scope.DifferentPixelsInsideDarkPartialAlphaObjectMask++
						if referenceDarker {
							scope.DifferentPixelsInsideDarkPartialAlphaObjectMaskReferenceDarker++
						} else if referenceLighter {
							scope.DifferentPixelsInsideDarkPartialAlphaObjectMaskReferenceLighter++
						}
					case 2:
						scope.DifferentPixelsInsideLightPartialAlphaObjectMask++
						if referenceDarker {
							scope.DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceDarker++
						} else if referenceLighter {
							scope.DifferentPixelsInsideLightPartialAlphaObjectMaskReferenceLighter++
						}
					default:
						scope.DifferentPixelsInsideOtherPartialAlphaObjectMask++
					}
					if underpainted {
						scope.DifferentPixelsInsidePartialAlphaObjectMaskOverUnderpaint++
					} else {
						scope.DifferentPixelsInsidePartialAlphaObjectMaskWithoutUnderpaint++
					}
				} else {
					scope.DifferentPixelsInsideFullAlphaObjectMask++
				}
			} else {
				scope.DifferentPixelsOutsideObjectMask++
			}
		}
	}
	if scope.DifferentPixels > 0 {
		scope.OutsideObjectMaskRatio = float64(scope.DifferentPixelsOutsideObjectMask) / float64(scope.DifferentPixels)
	}
	scope.TopDifferentRows = topMicroFixtureAxisCounts(differentRows, 8)
	scope.TopDifferentColumns = topMicroFixtureAxisCounts(differentColumns, 8)
	scope.TopReferenceDarkerRows = topMicroFixtureAxisCounts(referenceDarkerRows, 8)
	scope.TopReferenceLighterRows = topMicroFixtureAxisCounts(referenceLighterRows, 8)
	scope.TopReferenceDarkerColumns = topMicroFixtureAxisCounts(referenceDarkerColumns, 8)
	scope.TopReferenceLighterColumns = topMicroFixtureAxisCounts(referenceLighterColumns, 8)
	scope.TopReferenceRGBDeltaSums8 = topMicroFixtureDeltaCounts(referenceDeltaCounts, 12)
	scope.TopGotColors = topMicroFixtureSourceColors(got, 12)
	scope.TopReferenceColors = topMicroFixtureSourceColors(reference, 12)
	scope.TopDifferentGotColors = topMicroFixtureColorCountsFromMap(differentGotColorCounts, 12)
	scope.TopDifferentReferenceColors = topMicroFixtureColorCountsFromMap(differentReferenceColorCounts, 12)
	return scope, nil
}

func partialAlphaObjectMaskTone(r uint32, g uint32, b uint32, a uint32) int {
	luma := averageUnpremultipliedRGB8(r, g, b, a)
	switch {
	case luma <= 64:
		return 1
	case luma >= 192:
		return 2
	default:
		return 3
	}
}

func averageUnpremultipliedRGB8(r uint32, g uint32, b uint32, a uint32) int {
	if a == 0 {
		return 0
	}
	red := min(255, int(math.Round(float64(r)*255/float64(a))))
	green := min(255, int(math.Round(float64(g)*255/float64(a))))
	blue := min(255, int(math.Round(float64(b)*255/float64(a))))
	return (red + green + blue) / 3
}

func includeImageDiffBounds(bounds **imageDiffBounds, x int, y int) {
	if *bounds == nil {
		*bounds = &imageDiffBounds{MinX: x, MinY: y, MaxX: x, MaxY: y}
		return
	}
	if x < (*bounds).MinX {
		(*bounds).MinX = x
	}
	if y < (*bounds).MinY {
		(*bounds).MinY = y
	}
	if x > (*bounds).MaxX {
		(*bounds).MaxX = x
	}
	if y > (*bounds).MaxY {
		(*bounds).MaxY = y
	}
}

func topMicroFixtureAxisCounts(counts []int, limit int) []microFixtureAxisCount {
	var ranked []microFixtureAxisCount
	for index, count := range counts {
		if count == 0 {
			continue
		}
		ranked = append(ranked, microFixtureAxisCount{Index: index, Count: count})
	}
	sort.Slice(ranked, func(i int, j int) bool {
		if ranked[i].Count != ranked[j].Count {
			return ranked[i].Count > ranked[j].Count
		}
		return ranked[i].Index < ranked[j].Index
	})
	if len(ranked) > limit {
		return ranked[:limit]
	}
	return ranked
}

func topMicroFixtureDeltaCounts(counts map[int]int, limit int) []microFixtureDeltaCount {
	ranked := make([]microFixtureDeltaCount, 0, len(counts))
	for delta, count := range counts {
		if delta == 0 || count == 0 {
			continue
		}
		ranked = append(ranked, microFixtureDeltaCount{Delta: delta, Count: count})
	}
	sort.Slice(ranked, func(i int, j int) bool {
		if ranked[i].Count != ranked[j].Count {
			return ranked[i].Count > ranked[j].Count
		}
		return ranked[i].Delta < ranked[j].Delta
	})
	if len(ranked) > limit {
		return ranked[:limit]
	}
	return ranked
}

func referenceBrightnessDelta8(gr, gg, gb, wr, wg, wb uint32) int {
	gotSum := int(gr>>8) + int(gg>>8) + int(gb>>8)
	referenceSum := int(wr>>8) + int(wg>>8) + int(wb>>8)
	return referenceSum - gotSum
}

func referenceDeltaDirection(delta int) (darker bool, lighter bool) {
	switch {
	case delta < 0:
		return true, false
	case delta > 0:
		return false, true
	default:
		return false, false
	}
}

func microFixtureAcceptance(occlusions []microFixtureOcclusion) string {
	if len(occlusions) > 0 {
		return "got-visible-crop.png must match reference-visible-crop.png exactly before this object-level renderer change can be accepted; visible crops mask later z-order occlusions plus the recorded antialias padding"
	}
	return "got-crop.png must match reference-crop.png exactly before this object-level renderer change can be accepted"
}

func microFixtureOcclusions(object objectFailureRecord, objects []objectFailureRecord) []microFixtureOcclusion {
	if object.OutputPixelBounds == nil {
		return nil
	}
	var occlusions []microFixtureOcclusion
	for _, candidate := range objects {
		candidateBounds, ok := microFixtureOcclusionCandidateBounds(candidate)
		if candidate.ZOrder <= object.ZOrder || !ok {
			continue
		}
		intersection, ok := intersectObjectPixelBounds(*object.OutputPixelBounds, candidateBounds)
		if !ok {
			continue
		}
		occlusions = append(occlusions, microFixtureOcclusion{
			CNvPrID:           candidate.CNvPrID,
			CNvPrName:         candidate.CNvPrName,
			Kind:              candidate.Kind,
			ZOrder:            candidate.ZOrder,
			Bounds:            intersection,
			MaskPaddingPixels: 1,
		})
	}
	sort.SliceStable(occlusions, func(i, j int) bool {
		if occlusions[i].ZOrder != occlusions[j].ZOrder {
			return occlusions[i].ZOrder < occlusions[j].ZOrder
		}
		return occlusions[i].CNvPrID < occlusions[j].CNvPrID
	})
	return occlusions
}

func microFixtureOcclusionCandidateBounds(candidate objectFailureRecord) (ObjectPixelBounds, bool) {
	if candidate.PixelBounds != (ObjectPixelBounds{}) {
		return candidate.PixelBounds, true
	}
	if candidate.OutputPixelBounds != nil {
		return *candidate.OutputPixelBounds, true
	}
	return ObjectPixelBounds{}, false
}

func microFixtureUnderpaints(object objectFailureRecord, objects []objectFailureRecord) []microFixtureUnderpaint {
	if object.OutputPixelBounds == nil {
		return nil
	}
	var underpaints []microFixtureUnderpaint
	for _, candidate := range objects {
		if candidate.ZOrder >= object.ZOrder || candidate.OutputPixelBounds == nil {
			continue
		}
		intersection, ok := intersectObjectPixelBounds(*object.OutputPixelBounds, *candidate.OutputPixelBounds)
		if !ok {
			continue
		}
		underpaints = append(underpaints, microFixtureUnderpaint{
			CNvPrID:            candidate.CNvPrID,
			CNvPrName:          candidate.CNvPrName,
			Kind:               candidate.Kind,
			ZOrder:             candidate.ZOrder,
			Bounds:             intersection,
			ObjectArtifactPath: candidate.ObjectArtifactPath,
		})
	}
	sort.SliceStable(underpaints, func(i, j int) bool {
		if underpaints[i].ZOrder != underpaints[j].ZOrder {
			return underpaints[i].ZOrder < underpaints[j].ZOrder
		}
		return underpaints[i].CNvPrID < underpaints[j].CNvPrID
	})
	return underpaints
}

type microFixtureUnderpaintMask struct {
	bounds ObjectPixelBounds
	mask   image.Image
}

func loadUnderpaintMasks(underpaints []microFixtureUnderpaint) ([]microFixtureUnderpaintMask, error) {
	var masks []microFixtureUnderpaintMask
	for _, underpaint := range underpaints {
		if underpaint.ObjectArtifactPath == "" {
			continue
		}
		mask, err := decodePNGFile(resolveTestArtifactPath(underpaint.ObjectArtifactPath))
		if err != nil {
			return nil, err
		}
		masks = append(masks, microFixtureUnderpaintMask{
			bounds: underpaint.Bounds,
			mask:   mask,
		})
	}
	return masks, nil
}

func pointUnderpainted(x int, y int, masks []microFixtureUnderpaintMask) bool {
	for _, candidate := range masks {
		if x < candidate.bounds.MinX || x > candidate.bounds.MaxX || y < candidate.bounds.MinY || y > candidate.bounds.MaxY {
			continue
		}
		maskBounds := candidate.mask.Bounds()
		if x < 0 || y < 0 || x >= maskBounds.Dx() || y >= maskBounds.Dy() {
			continue
		}
		_, _, _, alpha := candidate.mask.At(maskBounds.Min.X+x, maskBounds.Min.Y+y).RGBA()
		if alpha != 0 {
			return true
		}
	}
	return false
}

func intersectObjectPixelBounds(a ObjectPixelBounds, b ObjectPixelBounds) (ObjectPixelBounds, bool) {
	intersection := ObjectPixelBounds{
		MinX: max(a.MinX, b.MinX),
		MinY: max(a.MinY, b.MinY),
		MaxX: min(a.MaxX, b.MaxX),
		MaxY: min(a.MaxY, b.MaxY),
	}
	if intersection.MinX > intersection.MaxX || intersection.MinY > intersection.MaxY {
		return ObjectPixelBounds{}, false
	}
	return intersection, true
}

func writePictureObjectFixture(deckInput string, fixturePath string, object objectFailureRecord) (microFixtureSourceImage, error) {
	deckPath := realWorldDeckPath(deckInput)
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		return microFixtureSourceImage{}, err
	}
	relationships, err := pkg.RelationshipsForPart(object.SourcePart)
	if err != nil {
		return microFixtureSourceImage{}, err
	}
	var imageRel pptx.Relationship
	found := false
	for _, relationship := range relationships {
		if relationship.ID == object.ResolvedStyle.EmbedID {
			imageRel = relationship
			found = true
			break
		}
	}
	if !found {
		return microFixtureSourceImage{}, fmt.Errorf("relationship %s not found in %s", object.ResolvedStyle.EmbedID, object.SourcePart)
	}
	sourceMediaPart := pptx.ResolveTargetPart(object.SourcePart, imageRel.Target)
	mediaData, ok := pkg.Parts[sourceMediaPart]
	if !ok {
		return microFixtureSourceImage{}, fmt.Errorf("media part %s not found", sourceMediaPart)
	}
	sourceImage := microFixtureSourceImage{Part: sourceMediaPart}
	if config, format, err := image.DecodeConfig(bytes.NewReader(mediaData)); err == nil {
		sourceImage.Format = format
		sourceImage.Width = config.Width
		sourceImage.Height = config.Height
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(sourceMediaPart)), ".")
	if extension == "" {
		extension = "png"
	}
	mediaPart := "ppt/media/object." + extension
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	if size.CX <= 0 || size.CY <= 0 {
		size = slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	}
	sourceData, ok := pkg.Parts[object.SourcePart]
	if !ok {
		return microFixtureSourceImage{}, fmt.Errorf("source part %s not found", object.SourcePart)
	}
	rawObject, err := extractRawObjectXMLForRecord(sourceData, object)
	if err != nil {
		return microFixtureSourceImage{}, err
	}
	parts := []struct {
		name string
		data []byte
	}{
		{name: "[Content_Types].xml", data: []byte(pictureObjectFixtureContentTypes(extension))},
		{name: "_rels/.rels", data: []byte(pictureObjectFixtureRootRelationships())},
		{name: "ppt/presentation.xml", data: []byte(pictureObjectFixturePresentation(size))},
		{name: "ppt/_rels/presentation.xml.rels", data: []byte(pictureObjectFixturePresentationRelationships())},
		{name: "ppt/slides/slide1.xml", data: []byte(pictureObjectFixtureSlide(rawObject))},
		{name: "ppt/slides/_rels/slide1.xml.rels", data: []byte(pictureObjectFixtureSlideRelationships(mediaPart, object.ResolvedStyle.EmbedID))},
		{name: mediaPart, data: mediaData},
	}
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		return microFixtureSourceImage{}, err
	}
	file, err := os.Create(fixturePath)
	if err != nil {
		return microFixtureSourceImage{}, err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	for _, part := range parts {
		writer, err := archive.Create(part.name)
		if err != nil {
			archive.Close()
			return microFixtureSourceImage{}, err
		}
		if _, err := writer.Write(part.data); err != nil {
			archive.Close()
			return microFixtureSourceImage{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return microFixtureSourceImage{}, err
	}
	return sourceImage, nil
}

func writeShapeObjectFixture(deckInput string, fixturePath string, object objectFailureRecord) error {
	return writeShapeObjectsFixture(deckInput, fixturePath, object, []objectFailureRecord{object})
}

func writeShapeObjectsFixture(deckInput string, fixturePath string, backgroundObject objectFailureRecord, objects []objectFailureRecord) error {
	deckPath := realWorldDeckPath(deckInput)
	pkg, err := pptx.Open(context.Background(), deckPath)
	if err != nil {
		return err
	}
	sourceData, ok := pkg.Parts[backgroundObject.SourcePart]
	if !ok {
		return fmt.Errorf("source part %s not found", backgroundObject.SourcePart)
	}
	slideData := sourceData
	if data, ok := pkg.Parts[backgroundObject.SlidePart]; ok {
		slideData = data
	}
	var rawObjects []string
	for _, object := range objects {
		objectSourceData, ok := pkg.Parts[object.SourcePart]
		if !ok {
			return fmt.Errorf("source part %s not found", object.SourcePart)
		}
		rawObject, err := extractRawObjectXMLForRecord(objectSourceData, object)
		if err != nil {
			return err
		}
		rawObjects = append(rawObjects, rawObject)
	}
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	if size.CX <= 0 || size.CY <= 0 {
		size = slideSize{CX: defaultSlideCX, CY: defaultSlideCY}
	}
	hasTableStyles := shapeObjectFixtureNeedsTableStyles(objects) && pkg.Parts["ppt/tableStyles.xml"] != nil
	parts := map[string][]byte{
		"[Content_Types].xml":              []byte(shapeObjectFixtureContentTypes(hasTableStyles)),
		"_rels/.rels":                      []byte(pictureObjectFixtureRootRelationships()),
		"ppt/presentation.xml":             []byte(pictureObjectFixturePresentation(size)),
		"ppt/_rels/presentation.xml.rels":  []byte(pictureObjectFixturePresentationRelationships()),
		"ppt/slides/slide1.xml":            []byte(shapeObjectFixtureSlide(shapeObjectFixtureBackgroundXML(slideData, sourceData), rawObjects)),
		"ppt/slides/_rels/slide1.xml.rels": []byte(shapeObjectFixtureSlideRelationships("")),
	}
	addShapeObjectFixturePackageDependencies(parts, pkg, objects)
	layoutPart := firstRelationshipTarget(pkg, backgroundObject.SlidePart, pptx.SlideLayoutRelType)
	if layoutPart != "" {
		if data, ok := pkg.Parts[layoutPart]; ok {
			parts[layoutPart] = stripNonPlaceholderObjectsInPart(data)
			parts["ppt/slides/_rels/slide1.xml.rels"] = []byte(shapeObjectFixtureSlideRelationships(layoutPart))
		}
	}
	masterPart := ""
	if layoutPart != "" {
		masterPart = firstRelationshipTarget(pkg, layoutPart, pptx.SlideMasterRelType)
	}
	if masterPart != "" {
		if data, ok := pkg.Parts[masterPart]; ok {
			parts[masterPart] = stripNonPlaceholderObjectsInPart(data)
			parts[pptx.RelationshipsPartFor(layoutPart)] = []byte(shapeObjectFixtureLayoutRelationships(masterPart))
		}
	}
	themePart := ""
	if masterPart != "" {
		themePart = firstRelationshipTarget(pkg, masterPart, themeRelType)
	}
	if themePart != "" {
		if data, ok := pkg.Parts[themePart]; ok {
			parts[themePart] = data
			parts[pptx.RelationshipsPartFor(masterPart)] = []byte(shapeObjectFixtureMasterRelationships(themePart))
		}
	} else {
		for partName, data := range pkg.Parts {
			if strings.HasPrefix(partName, "ppt/theme/") && strings.HasSuffix(partName, ".xml") {
				parts[partName] = data
			}
		}
	}
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		return err
	}
	return pptx.Write(context.Background(), &pptx.Package{Parts: parts}, fixturePath)
}

func addShapeObjectFixturePackageDependencies(parts map[string][]byte, pkg *pptx.Package, objects []objectFailureRecord) {
	if !shapeObjectFixtureNeedsTableStyles(objects) {
		return
	}
	if data, ok := pkg.Parts["ppt/tableStyles.xml"]; ok {
		parts["ppt/tableStyles.xml"] = data
	}
}

func shapeObjectFixtureNeedsTableStyles(objects []objectFailureRecord) bool {
	for _, object := range objects {
		if object.Kind == "graphicFrame" && object.ResolvedStyle.Table {
			return true
		}
	}
	return false
}

func shapeObjectFixtureBackgroundXML(slideData []byte, sourceData []byte) string {
	if background := extractRawSlideBackgroundXML(slideData); background != "" {
		return background
	}
	return extractRawSlideBackgroundXML(sourceData)
}

func realWorldDeckPath(deckInput string) string {
	if filepath.IsAbs(deckInput) {
		return deckInput
	}
	return filepath.Join("..", "..", deckInput)
}

func resolveTestArtifactPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	repoRelative := filepath.Join("..", "..", path)
	if _, err := os.Stat(repoRelative); err == nil {
		return repoRelative
	}
	return path
}

func resolveTestOutputPath(path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(filepath.Dir(path)); err == nil {
		return path
	}
	repoRelative := filepath.Join("..", "..", path)
	if _, err := os.Stat(filepath.Dir(repoRelative)); err == nil {
		return repoRelative
	}
	return path
}

func extractRawSlideBackgroundXML(data []byte) string {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []string
	bgDepth := 0
	bgStart := int64(0)
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch item := token.(type) {
		case xml.StartElement:
			stack = append(stack, item.Name.Local)
			depth := len(stack)
			if bgDepth == 0 && item.Name.Local == "bg" && depth >= 2 && stack[depth-2] == "cSld" {
				bgDepth = depth
				bgStart = rawXMLStartOffset(data, decoder.InputOffset())
			}
		case xml.EndElement:
			depth := len(stack)
			if bgDepth > 0 && depth == bgDepth && item.Name.Local == "bg" {
				return string(data[bgStart:decoder.InputOffset()])
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return ""
}

func extractRawObjectXMLForRecord(data []byte, object objectFailureRecord) (string, error) {
	if object.ZOrder > 0 {
		raw, err := extractRawObjectXMLByZOrder(data, object.Kind, object.CNvPrID, object.CNvPrName, object.ZOrder)
		if err == nil {
			return raw, nil
		}
	}
	return extractRawObjectXML(data, object.Kind, object.CNvPrID, object.CNvPrName)
}

func extractRawObjectXMLByZOrder(data []byte, kind string, cnvPrID string, cnvPrName string, zOrder int) (string, error) {
	type candidate struct {
		start   int64
		end     int64
		kind    string
		matched bool
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []string
	spTreeDepth := 0
	candidateDepth := 0
	current := candidate{}
	renderableIndex := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch item := token.(type) {
		case xml.StartElement:
			stack = append(stack, item.Name.Local)
			depth := len(stack)
			if item.Name.Local == "spTree" && spTreeDepth == 0 {
				spTreeDepth = depth
				continue
			}
			if spTreeDepth > 0 && candidateDepth == 0 && depth == spTreeDepth+1 && renderableObjectElement(item.Name.Local) {
				candidateDepth = depth
				current = candidate{
					start: rawXMLStartOffset(data, decoder.InputOffset()),
					kind:  item.Name.Local,
				}
				continue
			}
			if candidateDepth > 0 && item.Name.Local == "cNvPr" && cnvPrMatches(item.Attr, cnvPrID, cnvPrName) {
				current.matched = true
			}
		case xml.EndElement:
			depth := len(stack)
			if candidateDepth > 0 && depth == candidateDepth && renderableObjectElement(item.Name.Local) {
				renderableIndex++
				current.end = decoder.InputOffset()
				if renderableIndex == zOrder && current.kind == kind && current.matched {
					return string(data[current.start:current.end]), nil
				}
				candidateDepth = 0
				current = candidate{}
			}
			if spTreeDepth > 0 && depth == spTreeDepth && item.Name.Local == "spTree" {
				spTreeDepth = 0
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return "", fmt.Errorf("%s object cNvPr id=%q name=%q z-order=%d not found", kind, cnvPrID, cnvPrName, zOrder)
}

func extractRawObjectXML(data []byte, kind string, cnvPrID string, cnvPrName string) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var startOffset int64
	depth := 0
	matched := false
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch item := token.(type) {
		case xml.StartElement:
			if depth == 0 {
				if item.Name.Local != kind {
					continue
				}
				startOffset = rawXMLStartOffset(data, decoder.InputOffset())
				depth = 1
				matched = false
				continue
			}
			depth++
			if item.Name.Local == "cNvPr" && cnvPrMatches(item.Attr, cnvPrID, cnvPrName) {
				matched = true
			}
		case xml.EndElement:
			if depth == 0 {
				continue
			}
			depth--
			if depth == 0 && item.Name.Local == kind {
				if matched {
					return string(data[startOffset:decoder.InputOffset()]), nil
				}
				startOffset = 0
			}
		}
	}
	return "", fmt.Errorf("%s object cNvPr id=%q name=%q not found", kind, cnvPrID, cnvPrName)
}

func rawXMLStartOffset(data []byte, endOffset int64) int64 {
	if endOffset <= 0 || endOffset > int64(len(data)) {
		return 0
	}
	start := bytes.LastIndexByte(data[:endOffset], '<')
	if start < 0 {
		return 0
	}
	return int64(start)
}

func cnvPrMatches(attrs []xml.Attr, id string, name string) bool {
	attrID := attrValue(attrs, "id")
	attrName := attrValue(attrs, "name")
	if id != "" && attrID != id {
		return false
	}
	if name != "" && attrName != name {
		return false
	}
	return id != "" || name != ""
}

func stripNonPlaceholderObjectsInPart(data []byte) []byte {
	type span struct {
		start int64
		end   int64
	}
	var spans []span
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var stack []string
	spTreeDepth := 0
	candidateDepth := 0
	candidateStart := int64(0)
	candidateHasPlaceholder := false
	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch item := token.(type) {
		case xml.StartElement:
			stack = append(stack, item.Name.Local)
			depth := len(stack)
			if item.Name.Local == "spTree" && spTreeDepth == 0 {
				spTreeDepth = depth
				continue
			}
			if spTreeDepth > 0 && candidateDepth == 0 && depth == spTreeDepth+1 && renderableObjectElement(item.Name.Local) {
				candidateDepth = depth
				candidateStart = rawXMLStartOffset(data, decoder.InputOffset())
				candidateHasPlaceholder = false
				continue
			}
			if candidateDepth > 0 && item.Name.Local == "ph" {
				candidateHasPlaceholder = true
			}
		case xml.EndElement:
			depth := len(stack)
			if candidateDepth > 0 && depth == candidateDepth && renderableObjectElement(item.Name.Local) {
				if !candidateHasPlaceholder {
					spans = append(spans, span{start: candidateStart, end: decoder.InputOffset()})
				}
				candidateDepth = 0
				candidateStart = 0
				candidateHasPlaceholder = false
			}
			if spTreeDepth > 0 && depth == spTreeDepth && item.Name.Local == "spTree" {
				spTreeDepth = 0
			}
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
	if len(spans) == 0 {
		return data
	}
	output := make([]byte, 0, len(data))
	cursor := int64(0)
	for _, remove := range spans {
		output = append(output, data[cursor:remove.start]...)
		cursor = remove.end
	}
	output = append(output, data[cursor:]...)
	return output
}

func renderableObjectElement(name string) bool {
	switch name {
	case "sp", "cxnSp", "pic", "graphicFrame", "grpSp":
		return true
	default:
		return false
	}
}

func pictureObjectFixtureContentTypes(extension string) string {
	contentType := "image/png"
	if extension == "jpg" || extension == "jpeg" {
		contentType = "image/jpeg"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="%s" ContentType="%s"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
</Types>`, extension, contentType)
}

func shapeObjectFixtureContentTypes(hasTableStyles bool) string {
	tableStylesOverride := ""
	if hasTableStyles {
		tableStylesOverride = `
  <Override PartName="/ppt/tableStyles.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.tableStyles+xml"/>`
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/slides/slide1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>%s
</Types>`, tableStylesOverride)
}

func pictureObjectFixtureRootRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`
}

func pictureObjectFixturePresentation(size slideSize) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst><p:sldId id="256" r:id="rId1"/></p:sldIdLst>
  <p:sldSz cx="%d" cy="%d"/>
</p:presentation>`, size.CX, size.CY)
}

func pictureObjectFixturePresentationRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide1.xml"/>
</Relationships>`
}

func pictureObjectFixtureSlide(rawPictureXML string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
      %s
    </p:spTree>
  </p:cSld>
</p:sld>`, rawPictureXML)
}

func shapeObjectFixtureSlide(rawBackground string, rawObjects []string) string {
	if rawBackground != "" {
		rawBackground += "\n    "
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld>
    %s
    <p:spTree>
      <p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr>
      <p:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="0" cy="0"/><a:chOff x="0" y="0"/><a:chExt cx="0" cy="0"/></a:xfrm></p:grpSpPr>
      %s
    </p:spTree>
  </p:cSld>
</p:sld>`, rawBackground, strings.Join(rawObjects, "\n      "))
}

func shapeObjectFixtureSlideRelationships(layoutPart string) string {
	if layoutPart == "" {
		return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>`
	}
	target := "../slideLayouts/" + filepath.Base(layoutPart)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="%s" Target="%s"/>
</Relationships>`, pptx.SlideLayoutRelType, xmlEscape(target))
}

func shapeObjectFixtureLayoutRelationships(masterPart string) string {
	target := "../slideMasters/" + filepath.Base(masterPart)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="%s" Target="%s"/>
</Relationships>`, pptx.SlideMasterRelType, xmlEscape(target))
}

func shapeObjectFixtureMasterRelationships(themePart string) string {
	target := "../theme/" + filepath.Base(themePart)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="%s" Target="%s"/>
</Relationships>`, themeRelType, xmlEscape(target))
}

func pictureObjectFixtureSlideRelationships(mediaPart string, relationshipID string) string {
	if relationshipID == "" {
		relationshipID = "rId1"
	}
	target := "../media/" + filepath.Base(mediaPart)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="%s"/>
</Relationships>`, xmlEscape(relationshipID), xmlEscape(target))
}

func writeCroppedPNG(sourcePath string, targetPath string, crop ObjectPixelBounds) error {
	source, err := decodePNGFile(sourcePath)
	if err != nil {
		return err
	}
	sourceBounds := source.Bounds()
	rect := image.Rect(crop.MinX, crop.MinY, crop.MaxX+1, crop.MaxY+1).Intersect(image.Rect(0, 0, sourceBounds.Dx(), sourceBounds.Dy()))
	if rect.Empty() {
		rect = image.Rect(0, 0, 1, 1)
	}
	output := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			output.Set(x, y, source.At(sourceBounds.Min.X+rect.Min.X+x, sourceBounds.Min.Y+rect.Min.Y+y))
		}
	}
	return writePNG(targetPath, output)
}

func writeVisibleCroppedPNG(sourcePath string, targetPath string, crop ObjectPixelBounds, occlusions []microFixtureOcclusion) error {
	source, err := decodePNGFile(sourcePath)
	if err != nil {
		return err
	}
	sourceBounds := source.Bounds()
	rect := image.Rect(crop.MinX, crop.MinY, crop.MaxX+1, crop.MaxY+1).Intersect(image.Rect(0, 0, sourceBounds.Dx(), sourceBounds.Dy()))
	if rect.Empty() {
		rect = image.Rect(0, 0, 1, 1)
	}
	output := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	for y := 0; y < rect.Dy(); y++ {
		for x := 0; x < rect.Dx(); x++ {
			fullX := rect.Min.X + x
			fullY := rect.Min.Y + y
			if pointOccluded(fullX, fullY, occlusions) {
				output.SetRGBA(x, y, color.RGBA{})
				continue
			}
			output.Set(x, y, source.At(sourceBounds.Min.X+fullX, sourceBounds.Min.Y+fullY))
		}
	}
	return writePNG(targetPath, output)
}

func writeNonUnderpaintedTargetPNG(sourcePath string, targetPath string, crop ObjectPixelBounds, underpaints []microFixtureUnderpaint) error {
	source, err := decodePNGFile(sourcePath)
	if err != nil {
		return err
	}
	underpaintMasks, err := loadUnderpaintMasks(underpaints)
	if err != nil {
		return err
	}
	sourceBounds := source.Bounds()
	output := image.NewRGBA(image.Rect(0, 0, sourceBounds.Dx(), sourceBounds.Dy()))
	for y := 0; y < sourceBounds.Dy(); y++ {
		for x := 0; x < sourceBounds.Dx(); x++ {
			fullX := crop.MinX + x
			fullY := crop.MinY + y
			if pointUnderpainted(fullX, fullY, underpaintMasks) {
				output.SetRGBA(x, y, color.RGBA{})
				continue
			}
			output.Set(x, y, source.At(sourceBounds.Min.X+x, sourceBounds.Min.Y+y))
		}
	}
	return writePNG(targetPath, output)
}

func pointOccluded(x int, y int, occlusions []microFixtureOcclusion) bool {
	for _, occlusion := range occlusions {
		if x >= occlusion.Bounds.MinX-occlusion.MaskPaddingPixels && x <= occlusion.Bounds.MaxX+occlusion.MaskPaddingPixels && y >= occlusion.Bounds.MinY-occlusion.MaskPaddingPixels && y <= occlusion.Bounds.MaxY+occlusion.MaskPaddingPixels {
			return true
		}
	}
	return false
}

func xmlEscape(value string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return replacer.Replace(value)
}

func writeJSONFile(targetPath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(targetPath, data, 0o644)
}

func writeJSONOutputFile(targetPath string, value any) error {
	return writeJSONFile(resolveTestOutputPath(targetPath), value)
}

func writeDiffPNG(gotPath string, referencePath string, targetPath string) error {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return err
	}
	reference, err := decodePNGFile(referencePath)
	if err != nil {
		return err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := max(gotBounds.Dx(), referenceBounds.Dx())
	height := max(gotBounds.Dy(), referenceBounds.Dy())
	if width <= 0 || height <= 0 {
		return writePNG(targetPath, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	}
	diff := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotInside := x < gotBounds.Dx() && y < gotBounds.Dy()
			referenceInside := x < referenceBounds.Dx() && y < referenceBounds.Dy()
			if !gotInside || !referenceInside {
				diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
				continue
			}
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr == rr && gg == rg && gb == rb && ga == ra {
				diff.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				continue
			}
			diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	return writePNG(targetPath, diff)
}

type referenceManifest struct {
	Decks []struct {
		Input  string `json:"input"`
		Slides []struct {
			File   string `json:"file"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"slides"`
	} `json:"decks"`
}

type imageDiff struct {
	Width                         int              `json:"width"`
	Height                        int              `json:"height"`
	GotWidth                      int              `json:"got_width"`
	GotHeight                     int              `json:"got_height"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
	Perceptual                    perceptualMetric `json:"perceptual"`
}

type imageDiffBounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

type perceptualMetric struct {
	Basis                     string  `json:"basis"`
	ComparedPixels            int     `json:"compared_pixels"`
	MeanAbsoluteLumaDelta8Bit float64 `json:"mean_absolute_luma_delta_8bit"`
	LumaSimilarity            float64 `json:"luma_similarity"`
	RMSChannelDelta8Bit       float64 `json:"rms_channel_delta_8bit"`
	ChannelRMSSimilarity      float64 `json:"channel_rms_similarity"`
	DifferentPixelRatio       float64 `json:"different_pixel_ratio"`
}

func referenceDPIForSlide(width int) int {
	if width <= 0 {
		return defaultOutputDPI
	}
	return normalizeOutputDPI(int(math.Round(float64(width) * emuPerInch / defaultSlideCX)))
}

func comparePNG(gotPath string, wantPath string) (imageDiff, error) {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return imageDiff{}, err
	}
	want, err := decodePNGFile(wantPath)
	if err != nil {
		return imageDiff{}, err
	}
	return compareImages(got, want), nil
}

func compareImages(got image.Image, want image.Image) imageDiff {
	gotBounds := got.Bounds()
	wantBounds := want.Bounds()
	diff := imageDiff{Width: wantBounds.Dx(), Height: wantBounds.Dy(), GotWidth: gotBounds.Dx(), GotHeight: gotBounds.Dy()}
	if gotBounds.Dx() != wantBounds.Dx() || gotBounds.Dy() != wantBounds.Dy() {
		diff.DifferentPixels = max(gotBounds.Dx(), wantBounds.Dx()) * max(gotBounds.Dy(), wantBounds.Dy())
		if diff.DifferentPixels > 0 {
			diff.DifferentBounds = &imageDiffBounds{MinX: 0, MinY: 0, MaxX: max(gotBounds.Dx(), wantBounds.Dx()) - 1, MaxY: max(gotBounds.Dy(), wantBounds.Dy()) - 1}
		}
		diff.Perceptual = perceptualMetric{
			Basis:                "not comparable because image dimensions differ",
			ComparedPixels:       0,
			LumaSimilarity:       0,
			ChannelRMSSimilarity: 0,
			DifferentPixelRatio:  1,
		}
		return diff
	}
	var lumaDeltaSum float64
	var channelSquareDeltaSum float64
	for y := 0; y < wantBounds.Dy(); y++ {
		for x := 0; x < wantBounds.Dx(); x++ {
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := want.At(wantBounds.Min.X+x, wantBounds.Min.Y+y).RGBA()
			gr8, gg8, gb8 := rgba16To8Bit(gr), rgba16To8Bit(gg), rgba16To8Bit(gb)
			wr8, wg8, wb8 := rgba16To8Bit(wr), rgba16To8Bit(wg), rgba16To8Bit(wb)
			lumaDeltaSum += math.Abs(perceptualLuma8Bit(gr8, gg8, gb8) - perceptualLuma8Bit(wr8, wg8, wb8))
			for _, pair := range [][2]int{{gr8, wr8}, {gg8, wg8}, {gb8, wb8}} {
				delta := float64(pair[0] - pair[1])
				channelSquareDeltaSum += delta * delta
			}
			if gr != wr || gg != wg || gb != wb || ga != wa {
				diff.DifferentPixels++
				if diff.DifferentBounds == nil {
					diff.DifferentBounds = &imageDiffBounds{MinX: x, MinY: y, MaxX: x, MaxY: y}
				} else {
					if x < diff.DifferentBounds.MinX {
						diff.DifferentBounds.MinX = x
					}
					if y < diff.DifferentBounds.MinY {
						diff.DifferentBounds.MinY = y
					}
					if x > diff.DifferentBounds.MaxX {
						diff.DifferentBounds.MaxX = x
					}
					if y > diff.DifferentBounds.MaxY {
						diff.DifferentBounds.MaxY = y
					}
				}
				diff.addChannelDelta8Bit(gr, wr)
				diff.addChannelDelta8Bit(gg, wg)
				diff.addChannelDelta8Bit(gb, wb)
				diff.addChannelDelta8Bit(ga, wa)
			}
		}
	}
	pixels := wantBounds.Dx() * wantBounds.Dy()
	if pixels > 0 {
		meanLumaDelta := lumaDeltaSum / float64(pixels)
		rmsChannelDelta := math.Sqrt(channelSquareDeltaSum / float64(pixels*3))
		diff.Perceptual = perceptualMetric{
			Basis:                     "deterministic luma and RGB-RMS image similarity; validation/triage only",
			ComparedPixels:            pixels,
			MeanAbsoluteLumaDelta8Bit: roundFloat(meanLumaDelta, 6),
			LumaSimilarity:            roundFloat(1-(meanLumaDelta/255), 9),
			RMSChannelDelta8Bit:       roundFloat(rmsChannelDelta, 6),
			ChannelRMSSimilarity:      roundFloat(1-(rmsChannelDelta/255), 9),
			DifferentPixelRatio:       roundFloat(float64(diff.DifferentPixels)/float64(pixels), 9),
		}
	}
	return diff
}

func perceptualLuma8Bit(r int, g int, b int) float64 {
	return 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
}

func roundFloat(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}

func (diff *imageDiff) addChannelDelta8Bit(got uint32, want uint32) {
	delta := absInt(rgba16To8Bit(got) - rgba16To8Bit(want))
	if delta > diff.MaxChannelDelta8Bit {
		diff.MaxChannelDelta8Bit = delta
	}
	diff.TotalAbsoluteChannelDelta8Bit += int64(delta)
}

func rgba16To8Bit(value uint32) int {
	return int(value >> 8)
}
