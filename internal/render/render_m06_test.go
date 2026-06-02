package render

import (
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
)

func TestM06CustomGeometrySupportsQuadArcAndMultiplePaths(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a">
  <a:pathLst>
    <a:path w="100" h="100">
      <a:moveTo><a:pt x="0" y="100"/></a:moveTo>
      <a:quadBezTo><a:pt x="50" y="0"/><a:pt x="100" y="100"/></a:quadBezTo>
      <a:lnTo><a:pt x="0" y="100"/></a:lnTo>
      <a:close/>
    </a:path>
    <a:path w="100" h="100" stroke="0">
      <a:moveTo><a:pt x="0" y="50"/></a:moveTo>
      <a:arcTo wR="25" hR="25" stAng="0" swAng="5400000"/>
      <a:lnTo><a:pt x="0" y="100"/></a:lnTo>
      <a:close/>
    </a:path>
  </a:pathLst>
</a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}

	paths, fills, strokes, commands, unsupported := parseCustomGeometryPathsCommandsWithDiagnostics(root)
	if len(unsupported) != 0 {
		t.Fatalf("expected quad/arc/multiple custom paths to be supported, got %+v", unsupported)
	}
	if len(paths) != 2 || len(fills) != 2 || len(strokes) != 2 || !fills[0] || !fills[1] || !strokes[0] || strokes[1] {
		t.Fatalf("unexpected custom path metadata: paths=%d fills=%+v strokes=%+v", len(paths), fills, strokes)
	}
	var kinds []string
	for _, command := range commands {
		kinds = append(kinds, command.Kind)
	}
	if !slices.Contains(kinds, "quadBezTo") || !slices.Contains(kinds, "arcTo") {
		t.Fatalf("expected quad and arc commands to be preserved, got %+v", kinds)
	}
}

func TestM06ShapeLineParsesCustomDashJoinCompoundAndMarkers(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
  <p:nvSpPr><p:cNvPr id="7" name="Stroke Shape"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
    <a:prstGeom prst="triangle"/>
    <a:ln w="38100" cap="rnd" algn="ctr" cmpd="tri">
      <a:solidFill><a:srgbClr val="FF0000"/></a:solidFill>
      <a:custDash><a:ds d="200000" sp="100000"/></a:custDash>
      <a:round/>
      <a:headEnd type="diamond" w="lg" len="sm"/>
      <a:tailEnd type="oval" w="sm" len="lg"/>
    </a:ln>
  </p:spPr>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}

	element := parseSlideElementNodeWithTheme(root, renderTransform{ScaleX: 1, ScaleY: 1}, defaultThemeColors())
	if !element.HasLine || element.LineDash != "cust:200000/100000" || element.LineJoin != "round" || element.LineCompound != "tri" {
		t.Fatalf("expected parsed M06 line semantics, got %+v", element)
	}
	primitive := renderShapePrimitiveFromElement("ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 96, 96), element)
	if primitive.Stroke.Dash != "cust:200000/100000" || primitive.Stroke.Join != "round" || primitive.Stroke.Compound != "tri" || primitive.Stroke.HeadMarker != "diamond" || primitive.Stroke.TailMarker != "oval" {
		t.Fatalf("primitive did not preserve M06 stroke fields: %+v", primitive.Stroke)
	}
}

func TestM06RendersSchemaLineEndMarkerTypes(t *testing.T) {
	for _, marker := range []string{"triangle", "stealth", "diamond", "oval", "arrow"} {
		t.Run(marker, func(t *testing.T) {
			size := slideSize{CX: emuPerInch, CY: emuPerInch}
			img := image.NewRGBA(image.Rect(0, 0, 96, 96))
			element := slideElement{
				Kind:           "cxnSp",
				Name:           "Marker Connector",
				PrstGeom:       "straightConnector1",
				HasTransform:   true,
				OffX:           emuPerInch / 8,
				OffY:           emuPerInch / 2,
				ExtCX:          emuPerInch * 3 / 4,
				HasLine:        true,
				LineColor:      color.RGBA{R: 255, A: 255},
				LineWidth:      38100,
				HasLineMarker:  true,
				TailLineMarker: marker,
			}

			unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
			if len(unsupported) != 0 || !element.Rendered {
				t.Fatalf("unexpected marker render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
			}
			if countOpaquePixelsWithRed(img) == 0 {
				t.Fatal("expected marker/line rendering to paint red pixels")
			}
		})
	}
}

func TestM06ReportsUnknownLineMarkerType(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:           "cxnSp",
		Name:           "Unknown Marker Connector",
		PrstGeom:       "straightConnector1",
		HasTransform:   true,
		ExtCX:          emuPerInch,
		HasLine:        true,
		LineColor:      color.RGBA{R: 255, A: 255},
		LineWidth:      9525,
		HasLineMarker:  true,
		TailLineMarker: "unsupportedMarker",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "line markers") || !element.Rendered {
		t.Fatalf("expected partial unsupported unknown marker, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
}

func TestM06RendersCompoundConnectorAndCustomDash(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:            "cxnSp",
		Name:            "Double Dashed Connector",
		PrstGeom:        "straightConnector1",
		HasTransform:    true,
		OffX:            emuPerInch / 8,
		OffY:            emuPerInch / 2,
		ExtCX:           emuPerInch * 3 / 4,
		HasLine:         true,
		LineColor:       color.RGBA{B: 255, A: 255},
		LineWidth:       76200,
		LineDash:        "cust:200000/100000",
		HasLineCompound: true,
		LineCompound:    "dbl",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("unexpected compound dashed connector render result: unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	top := img.RGBAAt(48, 44)
	bottom := img.RGBAAt(48, 52)
	center := img.RGBAAt(48, 48)
	if top.B == 0 || bottom.B == 0 {
		t.Fatalf("expected double compound line to paint both offset strokes, top=%#v bottom=%#v", top, bottom)
	}
	if center.A != 0 {
		t.Fatalf("expected double compound gap to remain unpainted, center=%#v", center)
	}
}

func TestM06RendersRoundJoinForRectOutline(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	base := slideElement{
		Kind:         "sp",
		Name:         "Round Join Rectangle",
		PrstGeom:     "rect",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		NoFill:       true,
		HasLine:      true,
		LineColor:    color.RGBA{R: 255, A: 255},
		LineWidth:    emuPerInch / 10,
	}

	plain := base
	plainImg := image.NewRGBA(image.Rect(0, 0, 96, 96))
	if unsupported := renderShape("ppt/slides/slide1.xml", size, plainImg, &plain); len(unsupported) != 0 || !plain.Rendered {
		t.Fatalf("unexpected plain rect render result: unsupported=%+v rendered=%v", unsupported, plain.Rendered)
	}
	if got := plainImg.RGBAAt(20, 20); got.A != 0 {
		t.Fatalf("plain rectangle outline should not paint outside the legacy inward stroke bounds, got %#v", got)
	}

	roundJoin := base
	roundJoin.HasLineJoin = true
	roundJoin.LineJoin = "round"
	roundImg := image.NewRGBA(image.Rect(0, 0, 96, 96))
	if unsupported := renderShape("ppt/slides/slide1.xml", size, roundImg, &roundJoin); len(unsupported) != 0 || !roundJoin.Rendered {
		t.Fatalf("unexpected round-join rect render result: unsupported=%+v rendered=%v", unsupported, roundJoin.Rendered)
	}
	if got := roundImg.RGBAAt(20, 20); got.R == 0 || got.A == 0 {
		t.Fatalf("round line join should paint the centered rounded corner stroke, got %#v", got)
	}
}

func TestM06CustomGeometryReportsExactUnsupportedCommand(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:custGeom xmlns:a="a"><a:pathLst><a:path w="100" h="100"><a:moveTo><a:pt x="0" y="0"/></a:moveTo><a:weirdTo/><a:lnTo><a:pt x="100" y="0"/></a:lnTo></a:path></a:pathLst></a:custGeom>`))
	if err != nil {
		t.Fatal(err)
	}

	_, unsupported := parseCustomGeometryPathWithDiagnostics(root)
	if len(unsupported) == 0 || !strings.Contains(strings.Join(unsupported, "; "), "unsupported weirdTo command") {
		t.Fatalf("expected exact unsupported custom command, got %+v", unsupported)
	}
}

func countOpaquePixelsWithRed(img *image.RGBA) int {
	count := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 0 && c.A > 0 {
				count++
			}
		}
	}
	return count
}
