package render

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func TestM05ColorResolutionSupportsHSLSystemAndChannelTransforms(t *testing.T) {
	hslRoot, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:hslClr hue="7200000" sat="100000" lum="50000"/></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	hsl, ok := colorFromColorNode(hslRoot)
	if !ok || hsl != (color.RGBA{G: 255, A: 255}) {
		t.Fatalf("expected HSL green, got=%#v ok=%v", hsl, ok)
	}

	sysRoot, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:sysClr val="windowText"><a:alphaMod val="50000"/></a:sysClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	sys, ok := colorFromColorNode(sysRoot)
	if !ok || sys != (color.RGBA{A: 127}) {
		t.Fatalf("expected system color fallback with alphaMod, got=%#v ok=%v", sys, ok)
	}

	channelRoot, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="808080"><a:redMod val="50000"/><a:greenOff val="10000"/><a:blue val="100000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	channel, ok := colorFromColorNode(channelRoot)
	if !ok || channel != (color.RGBA{R: 64, G: 154, B: 255, A: 255}) {
		t.Fatalf("expected RGB channel transforms, got=%#v ok=%v", channel, ok)
	}
}

func TestM05PatternFillParsesRendersAndLowersPaintPrimitive(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
  <p:nvSpPr><p:cNvPr id="2" name="Pattern Rectangle"/></p:nvSpPr>
  <p:spPr>
    <a:xfrm><a:off x="0" y="0"/><a:ext cx="1000" cy="1000"/></a:xfrm>
    <a:prstGeom prst="rect"/>
    <a:pattFill prst="pct50">
      <a:fgClr><a:srgbClr val="FF0000"/></a:fgClr>
      <a:bgClr><a:srgbClr val="0000FF"/></a:bgClr>
    </a:pattFill>
  </p:spPr>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	element := parseSlideElementNodeWithTheme(root, renderTransform{ScaleX: 1, ScaleY: 1}, defaultThemeColors())
	if !element.HasPatternFill || !element.HasFill || element.PatternFill.Preset != "pct50" {
		t.Fatalf("expected pattern paint to parse into resolved fill, got %+v", element)
	}
	primitive := renderShapePrimitiveFromElement("ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, image.Rect(0, 0, 4, 4), element)
	if !primitive.Fill.HasPattern || primitive.Fill.Pattern.Foreground != (color.RGBA{R: 255, A: 255}) || primitive.Fill.Pattern.Background != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("expected pattern fill primitive, got %+v", primitive.Fill)
	}

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	unsupported := renderShape("ppt/slides/slide1.xml", slideSize{CX: 1000, CY: 1000}, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected rectangle pattern fill to render without unsupported records, unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(0, 0); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("expected foreground pattern pixel, got %#v", got)
	}
	if got := img.RGBAAt(1, 0); got != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("expected background pattern pixel, got %#v", got)
	}
}

func TestM05GroupFillResolvesChildGroupFill(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld><p:spTree>
    <p:grpSp>
      <p:grpSpPr>
        <a:xfrm><a:off x="0" y="0"/><a:ext cx="1000" cy="1000"/><a:chOff x="0" y="0"/><a:chExt cx="1000" cy="1000"/></a:xfrm>
        <a:solidFill><a:schemeClr val="accent1"/></a:solidFill>
      </p:grpSpPr>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="3" name="Group Fill Child"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="1000" cy="1000"/></a:xfrm>
          <a:prstGeom prst="rect"/>
          <a:grpFill/>
        </p:spPr>
      </p:sp>
    </p:grpSp>
  </p:spTree></p:cSld>
</p:sld>`)
	elements := collectSlideElementsWithTheme(data, themeColors{"accent1": {R: 10, G: 20, B: 30, A: 255}})
	if len(elements) != 1 {
		t.Fatalf("expected one grouped element, got %+v", elements)
	}
	if !elements[0].HasFill || elements[0].FillColor != (color.RGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Fatalf("expected child grpFill to resolve from group fill, got %+v", elements[0])
	}
}

func TestM05BackgroundPatternFillParses(t *testing.T) {
	data := []byte(`<p:sld xmlns:p="p" xmlns:a="a">
  <p:cSld><p:bg><p:bgPr>
    <a:pattFill prst="horz">
      <a:fgClr><a:prstClr val="yellow"/></a:fgClr>
      <a:bgClr><a:sysClr val="windowText"/></a:bgClr>
    </a:pattFill>
  </p:bgPr></p:bg></p:cSld>
</p:sld>`)
	paint, ok := parseSlideBackgroundPaintWithTheme(data, defaultThemeColors())
	if !ok || !paint.HasPattern || paint.Pattern.Preset != "horz" {
		t.Fatalf("expected pattern background paint, got=%+v ok=%v", paint, ok)
	}
	if paint.Pattern.Foreground != (color.RGBA{R: 255, G: 255, A: 255}) || paint.Pattern.Background != (color.RGBA{A: 255}) {
		t.Fatalf("unexpected pattern colors: %+v", paint.Pattern)
	}
}

func TestM05DirectFillTakesPrecedenceOverStyleFill(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
  <p:nvSpPr><p:cNvPr id="4" name="Direct Fill"/></p:nvSpPr>
  <p:spPr><a:solidFill><a:srgbClr val="112233"/></a:solidFill></p:spPr>
  <p:style><a:fillRef idx="1"><a:schemeClr val="accent1"/></a:fillRef></p:style>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	fillStyles := parseThemeFillStyles([]byte(`<a:theme xmlns:a="a"><a:themeElements><a:fmtScheme><a:fillStyleLst>
  <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
</a:fillStyleLst></a:fmtScheme></a:themeElements></a:theme>`))
	element := parseSlideElementNodeWithThemeEffectsAndFills(root, renderTransform{ScaleX: 1, ScaleY: 1}, themeColors{"accent1": {R: 200, A: 255}}, themeEffectStyles{}, fillStyles, themeLineStyles{})
	if element.FillColor != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}) {
		t.Fatalf("expected direct fill to win over style fill, got %+v", element)
	}
	if len(element.PaintUnsupported) != 0 || strings.TrimSpace(element.UnsupportedNote) != "" {
		t.Fatalf("direct/style fill resolution should not create unsupported records, got %+v", element)
	}
}
