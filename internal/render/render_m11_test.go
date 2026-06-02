package render

import (
	"image"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
)

func TestM11CollectSlideElementsClassifiesChartGraphicFrame(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:c="http://schemas.openxmlformats.org/drawingml/2006/chart" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld><p:spTree><p:graphicFrame>
    <p:nvGraphicFramePr><p:cNvPr id="7" name="Chart 1"/></p:nvGraphicFramePr>
    <p:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm>
    <a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/chart"><c:chart r:id="rIdChart"/></a:graphicData></a:graphic>
  </p:graphicFrame></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one chart graphic frame, got %+v", elements)
	}
	got := elements[0]
	if got.Kind != "graphicFrame" || got.GraphicPayloadKind != "chart" || got.PayloadRelationshipID != "rIdChart" {
		t.Fatalf("expected chart payload metadata, got %+v", got)
	}
	if got.GraphicPayloadURI != "http://schemas.openxmlformats.org/drawingml/2006/chart" {
		t.Fatalf("expected chart graphicData URI, got %+v", got)
	}
}

func TestM11RenderGraphicFrameReportsChartPayload(t *testing.T) {
	pkg := &pptx.Package{}
	element := slideElement{
		Kind:                  "graphicFrame",
		ID:                    "7",
		Name:                  "Chart 1",
		HasTransform:          true,
		ExtCX:                 emuPerInch,
		ExtCY:                 emuPerInch,
		GraphicPayloadKind:    "chart",
		GraphicPayloadURI:     "http://schemas.openxmlformats.org/drawingml/2006/chart",
		PayloadRelationshipID: "rIdChart",
	}
	relationships := map[string]pptx.Relationship{
		"rIdChart": {ID: "rIdChart", Type: chartRelType, Target: "../charts/chart1.xml"},
	}

	partial := renderGraphicFrame(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, image.NewRGBA(image.Rect(0, 0, 96, 96)), &element, relationships, tableStyleSet{})
	if len(partial) != 1 || partial[0].Code != partialUnsupportedCode || !strings.Contains(partial[0].Message, "chart payload was preserved") || !strings.Contains(partial[0].Message, "chart graphics are not rendered yet") || !strings.Contains(partial[0].Message, "ppt/charts/chart1.xml") {
		t.Fatalf("expected precise chart partial-render report, got %+v", partial)
	}
	if element.Rendered {
		t.Fatal("partial chart payload should not be marked rendered")
	}
}

func TestM11CollectSlideElementsClassifiesUnsupportedPayloadObjects(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld><p:spTree>
    <p:contentPart r:id="rIdContent"><p:nvContentPartPr><p:cNvPr id="8" name="Model 1"/></p:nvContentPartPr><p:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm></p:contentPart>
    <p:oleObj r:id="rIdOle" progId="Excel.Sheet.12"><p:pic><p:nvPicPr><p:cNvPr id="9" name="OLE Preview"/></p:nvPicPr><p:blipFill><a:blip r:embed="rIdPreview"/></p:blipFill><p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm></p:spPr></p:pic></p:oleObj>
    <p:control r:id="rIdControl" name="Button1"/>
    <p:audioFile r:link="rIdAudio"/>
    <p:videoFile r:link="rIdVideo"/>
  </p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	got := map[string]slideElement{}
	for _, element := range elements {
		got[element.Kind+"."+element.GraphicPayloadKind] = element
	}
	for _, want := range []string{"contentPart.content part", "oleObj.OLE object", "control.control", "audioFile.audio", "videoFile.video"} {
		if _, ok := got[want]; !ok {
			t.Fatalf("expected payload element %s in %+v", want, elements)
		}
	}
	if got["oleObj.OLE object"].OLEProgID != "Excel.Sheet.12" || got["oleObj.OLE object"].PayloadRelationshipID != "rIdOle" {
		t.Fatalf("expected OLE relationship/progId metadata, got %+v", got["oleObj.OLE object"])
	}
	if got["pic."].EmbedID != "rIdPreview" {
		t.Fatalf("expected nested OLE preview picture to remain renderable, got %+v", elements)
	}
}

func TestM11RenderElementsReportsOLEAndRendersPreviewPicture(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rIdOle" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/oleObject" Target="../embeddings/oleObject1.bin"/>
  <Relationship Id="rIdPreview" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/image1.png"/>
</Relationships>`),
		"ppt/embeddings/oleObject1.bin": []byte("ole-bytes"),
		"ppt/media/image1.png":          redPNG(),
	}}
	elements := []slideElement{
		{Kind: "oleObj", ID: "9", Name: "Workbook", GraphicPayloadKind: "OLE object", PayloadRelationshipID: "rIdOle", OLEProgID: "Excel.Sheet.12"},
		{Kind: "pic", ID: "10", Name: "OLE Preview", EmbedID: "rIdPreview", HasTransform: true, ExtCX: emuPerInch / 2, ExtCY: emuPerInch / 2},
	}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))

	unsupported := renderElements(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, elements, tableStyleSet{})
	if len(unsupported) != 1 || unsupported[0].Code != unsupportedCode || !strings.Contains(unsupported[0].Message, "OLE object") || !strings.Contains(unsupported[0].Message, "ppt/embeddings/oleObject1.bin") {
		t.Fatalf("expected OLE unsupported report with relationship target, got %+v", unsupported)
	}
	if !elements[1].Rendered {
		t.Fatal("expected OLE preview picture to render as a normal picture")
	}
	if r, _, _, a := img.At(20, 20).RGBA(); r != 0xffff || a != 0xffff {
		t.Fatalf("expected rendered red OLE preview pixel, got r=%04x a=%04x", r, a)
	}
}

func TestM11RenderUnsupportedPayloadObjectsReportsPreciseFamilies(t *testing.T) {
	tests := []struct {
		name    string
		element slideElement
		want    string
	}{
		{name: "content", element: slideElement{Kind: "contentPart", Name: "3D Model", GraphicPayloadKind: "content part", PayloadRelationshipID: "rIdContent"}, want: "content part object"},
		{name: "control", element: slideElement{Kind: "control", Name: "Button1", GraphicPayloadKind: "control", PayloadRelationshipID: "rIdControl"}, want: "active controls are not rendered"},
		{name: "audio", element: slideElement{Kind: "audioFile", Name: "Audio", GraphicPayloadKind: "audio", PayloadRelationshipID: "rIdAudio"}, want: "rich media playback is not rendered"},
		{name: "video", element: slideElement{Kind: "videoFile", Name: "Video", GraphicPayloadKind: "video", PayloadRelationshipID: "rIdVideo"}, want: "rich media playback is not rendered"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unsupported := renderOneElement(&pptx.Package{}, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, image.NewRGBA(image.Rect(0, 0, 96, 96)), &tc.element, map[string]pptx.Relationship{}, tableStyleSet{})
			if len(unsupported) != 1 || unsupported[0].Code != unsupportedCode || !strings.Contains(unsupported[0].Message, tc.want) {
				t.Fatalf("expected precise unsupported payload report %q, got %+v", tc.want, unsupported)
			}
		})
	}
}

func TestM12CollectSlideElementsLowersLockedCanvasGraphicFrame(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:lc="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
  <p:cSld><p:spTree>
    <p:graphicFrame>
      <p:nvGraphicFramePr><p:cNvPr id="20" name="Locked Canvas Frame"/></p:nvGraphicFramePr>
      <p:xfrm><a:off x="914400" y="914400"/><a:ext cx="914400" cy="914400"/></p:xfrm>
      <a:graphic><a:graphicData uri="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
        <lc:lockedCanvas>
          <a:nvGrpSpPr><a:cNvPr id="21" name="Locked Canvas"/><a:cNvGrpSpPr/></a:nvGrpSpPr>
          <a:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/><a:chOff x="0" y="0"/><a:chExt cx="914400" cy="914400"/></a:xfrm></a:grpSpPr>
          <a:sp>
            <a:nvSpPr><a:cNvPr id="22" name="Locked Shape"/></a:nvSpPr>
            <a:spPr><a:xfrm><a:off x="91440" y="182880"/><a:ext cx="274320" cy="365760"/></a:xfrm><a:prstGeom prst="rect"/><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:spPr>
          </a:sp>
        </lc:lockedCanvas>
      </a:graphicData></a:graphic>
    </p:graphicFrame>
  </p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one lowered lockedCanvas child, got %+v", elements)
	}
	got := elements[0]
	if got.Kind != "sp" || got.ID != "22" || got.Name != "Locked Shape" {
		t.Fatalf("expected lockedCanvas shape child, got %+v", got)
	}
	if got.GraphicPayloadKind != "" {
		t.Fatalf("lockedCanvas child should not be classified as unsupported graphicData payload: %+v", got)
	}
	if got.OffX != 1005840 || got.OffY != 1097280 || got.ExtCX != 274320 || got.ExtCY != 365760 {
		t.Fatalf("unexpected lockedCanvas child transform: %+v", got)
	}
}

func TestM12CollectSlideElementsLowersLockedCanvasTextShape(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:lc="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
  <p:cSld><p:spTree>
    <p:graphicFrame>
      <p:nvGraphicFramePr><p:cNvPr id="20" name="Locked Canvas Frame"/></p:nvGraphicFramePr>
      <p:xfrm><a:off x="914400" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm>
      <a:graphic><a:graphicData uri="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
        <lc:lockedCanvas>
          <a:nvGrpSpPr><a:cNvPr id="21" name="Locked Canvas"/><a:cNvGrpSpPr/></a:nvGrpSpPr>
          <a:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/><a:chOff x="0" y="0"/><a:chExt cx="914400" cy="914400"/></a:xfrm></a:grpSpPr>
          <a:txSp>
            <a:txBody><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr sz="1200"/><a:t>Locked Text</a:t></a:r></a:p></a:txBody>
            <a:xfrm><a:off x="91440" y="182880"/><a:ext cx="457200" cy="274320"/></a:xfrm>
          </a:txSp>
        </lc:lockedCanvas>
      </a:graphicData></a:graphic>
    </p:graphicFrame>
  </p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one lowered lockedCanvas text child, got %+v", elements)
	}
	got := elements[0]
	if got.Kind != "sp" || got.Text != "Locked Text" || len(got.TextParagraphs) != 1 {
		t.Fatalf("expected lockedCanvas txSp to lower as text shape, got %+v", got)
	}
	if got.OffX != 1005840 || got.OffY != 182880 || got.ExtCX != 457200 || got.ExtCY != 274320 {
		t.Fatalf("unexpected lockedCanvas txSp transform: %+v", got)
	}
}

func TestM12RenderLockedCanvasGraphicFrameShape(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:lc="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
  <p:cSld><p:spTree>
    <p:graphicFrame>
      <p:nvGraphicFramePr><p:cNvPr id="20" name="Locked Canvas Frame"/></p:nvGraphicFramePr>
      <p:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></p:xfrm>
      <a:graphic><a:graphicData uri="http://purl.oclc.org/ooxml/drawingml/lockedCanvas">
        <lc:lockedCanvas>
          <a:nvGrpSpPr><a:cNvPr id="21" name="Locked Canvas"/><a:cNvGrpSpPr/></a:nvGrpSpPr>
          <a:grpSpPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/><a:chOff x="0" y="0"/><a:chExt cx="914400" cy="914400"/></a:xfrm></a:grpSpPr>
          <a:sp>
            <a:nvSpPr><a:cNvPr id="22" name="Locked Shape"/></a:nvSpPr>
            <a:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm><a:prstGeom prst="rect"/><a:solidFill><a:srgbClr val="00FF00"/></a:solidFill></a:spPr>
          </a:sp>
        </lc:lockedCanvas>
      </a:graphicData></a:graphic>
    </p:graphicFrame>
  </p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	unsupported := renderElements(&pptx.Package{}, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, img, elements, tableStyleSet{})
	if len(unsupported) != 0 {
		t.Fatalf("expected lockedCanvas shape to render without unsupported records, got %+v", unsupported)
	}
	if len(elements) != 1 || !elements[0].Rendered {
		t.Fatalf("expected lockedCanvas shape element to be marked rendered, got %+v", elements)
	}
	_, g, _, a := img.At(48, 48).RGBA()
	if g != 0xffff || a != 0xffff {
		t.Fatalf("expected rendered green lockedCanvas pixel, got g=%04x a=%04x", g, a)
	}
}
