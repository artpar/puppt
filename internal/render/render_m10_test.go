package render

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
)

func TestM10CollectSlideElementsReportsUnsupportedVisibleShapeEffects(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld>
    <p:spTree>
      <p:sp>
        <p:nvSpPr><p:cNvPr id="10" name="Effect Shape"/></p:nvSpPr>
        <p:spPr>
          <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
          <a:prstGeom prst="rect"/>
          <a:effectLst>
            <a:outerShdw blurRad="12700" dist="12700" dir="5400000"><a:srgbClr val="000000"/></a:outerShdw>
            <a:softEdge rad="12700"/>
            <a:glow rad="25400"><a:srgbClr val="FF0000"/></a:glow>
            <a:innerShdw blurRad="12700" dist="12700" dir="0"><a:srgbClr val="000000"/></a:innerShdw>
            <a:reflection stA="50000"/>
          </a:effectLst>
        </p:spPr>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasShadow || !got.HasSoftEdge || !got.HasGlow || !got.HasInnerShadow || !got.HasReflection {
		t.Fatalf("expected supported outer shadow, soft edge, glow, inner shadow, and reflection to parse, got %+v", got)
	}
	if len(got.EffectUnsupported) != 0 {
		t.Fatalf("expected all visible effects in this fixture to parse as supported static effects, got %+v", got.EffectUnsupported)
	}
	for _, unexpected := range []string{"glow", "innerShdw", "outerShdw", "reflection", "softEdge"} {
		if strings.Contains(strings.Join(got.EffectUnsupported, " "), unexpected) {
			t.Fatalf("supported effect %q should not be reported unsupported: %+v", unexpected, got.EffectUnsupported)
		}
	}
}

func TestM10CollectSlideElementsReportsEffectDag(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="11" name="Effect DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:glow rad="25400"><a:srgbClr val="00FF00"/></a:glow></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one effectDag shape, got %+v", elements)
	}
	got := elements[0]
	if !got.HasGlow || got.GlowRadius != 25400 || got.GlowColor.G != 255 {
		t.Fatalf("expected simple effectDag glow to flatten into supported effect state, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset") {
		t.Fatalf("expected effectDag simplified diagnostic, got %+v", got.EffectUnsupported)
	}
	if containsString(got.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("simple supported effectDag should not report unsupported graph nodes, got %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsFlattensAlphaOutsetEffectDag(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="17" name="Complex Effect DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:alphaOutset rad="12700"/></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one alphaOutset effectDag shape, got %+v", elements)
	}
	got := elements[0]
	if !got.HasAlphaOutset || got.AlphaOutsetRadius != 12700 {
		t.Fatalf("expected alphaOutset effectDag to flatten into supported effect state, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset") {
		t.Fatalf("expected flattened effectDag diagnostic, got %+v", got.EffectUnsupported)
	}
	if containsString(got.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("supported alphaOutset effectDag should not report unsupported graph nodes, got %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsFlattensRelativeOffsetEffectDag(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="22" name="Relative Offset DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:relOff tx="25000" ty="-12500"/></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one relOff effectDag shape, got %+v", elements)
	}
	got := elements[0]
	if !got.HasRelativeOffset || got.RelativeOffsetX != 25000 || got.RelativeOffsetY != -12500 {
		t.Fatalf("expected relOff effectDag to flatten into supported effect state, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset") {
		t.Fatalf("expected flattened effectDag diagnostic, got %+v", got.EffectUnsupported)
	}
	if containsString(got.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("supported relOff effectDag should not report unsupported graph nodes, got %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsFlattensTransformEffectDag(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="24" name="Transform Effect DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:xfrm tx="91440" ty="-45720"/></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one xfrm effectDag shape, got %+v", elements)
	}
	got := elements[0]
	if !got.HasEffectTransform || got.EffectTransformOffsetX != 91440 || got.EffectTransformOffsetY != -45720 {
		t.Fatalf("expected xfrm effectDag to flatten into supported translation state, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset") {
		t.Fatalf("expected flattened effectDag diagnostic, got %+v", got.EffectUnsupported)
	}
	if containsString(got.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("supported xfrm translation effectDag should not report unsupported graph nodes, got %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsReportsUnsupportedEffectDagNodes(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="17" name="Complex Effect DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:tint/></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 || !containsString(elements[0].EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("expected unsupported effectDag node diagnostic, got %+v", elements)
	}
}

func TestM10CollectSlideElementsFlattensSupportedBlendEffectDagChild(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="19" name="Blend Effect DAG Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectDag><a:cont><a:blend blend="screen"><a:cont><a:glow rad="25400"><a:srgbClr val="00FF00"/></a:glow></a:cont></a:blend></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one effectDag shape, got %+v", elements)
	}
	got := elements[0]
	if !got.HasGlow || got.GlowRadius != 25400 || got.GlowColor.G != 255 {
		t.Fatalf("expected supported blend child to flatten into glow effect state, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effectDag effect graph was rendered as a flattened supported effect subset") || !containsString(got.EffectUnsupported, "effectDag effect graph has unsupported ordering or effect nodes") {
		t.Fatalf("expected flattened plus graph-partial diagnostics, got %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsParsesBlurEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="13" name="Blurred Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:blur rad="91440"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasBlur || got.BlurRadius != 91440 || !got.BlurGrow {
		t.Fatalf("expected blur radius and schema-default grow=true to parse, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "blur") {
		t.Fatalf("supported blur effect should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsParsesFillOverlayEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="14" name="Overlay Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:fillOverlay blend="mult"><a:solidFill><a:srgbClr val="00FF00"/></a:solidFill></a:fillOverlay></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasFillOverlay || got.FillOverlayBlend != "mult" || got.FillOverlay.Color.G != 255 {
		t.Fatalf("expected fillOverlay blend and source fill to parse, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "fillOverlay") {
		t.Fatalf("supported fillOverlay effect should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsParsesInnerShadowEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="15" name="Inner Shadow Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:innerShdw blurRad="91440" dist="45720" dir="5400000"><a:srgbClr val="000000"><a:alpha val="60000"/></a:srgbClr></a:innerShdw></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasInnerShadow || got.InnerShadowBlur != 91440 || got.InnerShadowDistance != 45720 || got.InnerShadowDirection != 5400000 || got.InnerShadowColor.A == 0 {
		t.Fatalf("expected inner shadow source properties to parse, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "innerShdw") {
		t.Fatalf("supported innerShdw effect should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsParsesReflectionEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="16" name="Reflection Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:reflection blurRad="91440" stA="60000" stPos="0" endA="0" endPos="100000" dist="0"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasReflection || got.ReflectionBlur != 91440 || got.ReflectionStartAlpha != 60000 || got.ReflectionEndPosition != 100000 || got.ReflectionAlignment != "b" {
		t.Fatalf("expected reflection source properties to parse with defaults, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "reflection") {
		t.Fatalf("supported reflection effect should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsParsesPresetShadowAsSimplifiedShadow(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="12" name="Preset Shadow Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:prstShdw prst="shdw1" dist="12700" dir="5400000"><a:srgbClr val="000000"><a:alpha val="50000"/></a:srgbClr></a:prstShdw></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasShadow || got.ShadowDistance != 12700 || got.ShadowDirection != 5400000 || got.ShadowColor.A == 0 {
		t.Fatalf("expected preset shadow to map to source-backed shadow fields, got %+v", got)
	}
	if !containsString(got.EffectUnsupported, "effect prstShdw preset style was rendered as a simplified offset shadow") {
		t.Fatalf("expected preset shadow simplification diagnostic, got %+v", got.EffectUnsupported)
	}
}

func TestM10RenderShapeReportsUnsupportedEffectsButRendersSoftEdge(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:              "sp",
		Name:              "Soft Shape With Glow",
		PrstGeom:          "rect",
		HasTransform:      true,
		ExtCX:             emuPerInch / 2,
		ExtCY:             emuPerInch / 2,
		HasFill:           true,
		FillColor:         color.RGBA{B: 255, A: 255},
		HasSoftEdge:       true,
		SoftEdgeRadius:    203200,
		EffectUnsupported: []string{"effect glow was not rendered"},
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "effect glow") || !element.Rendered {
		t.Fatalf("expected one unsupported glow report with rendered soft edge, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	_, _, _, edgeAlpha := img.At(0, 0).RGBA()
	if edgeAlpha == 0 || edgeAlpha == 0xffff {
		t.Fatalf("expected soft edge alpha to be applied despite unsupported glow report, got alpha=%04x", edgeAlpha)
	}
}

func TestM10RenderShapePaintsGlowEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Glowing Shape",
		PrstGeom:     "rect",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
		HasGlow:      true,
		GlowColor:    color.RGBA{G: 255, A: 160},
		GlowRadius:   127000,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported glow render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(20, 48); got.G == 0 || got.A == 0 {
		t.Fatalf("expected green glow outside shape bounds, got %+v", got)
	}
}

func TestM10RenderShapePaintsFlattenedEffectDagGlow(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="18" name="Effect DAG Glow Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="228600" y="228600"/><a:ext cx="457200" cy="457200"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:solidFill><a:srgbClr val="FF0000"/></a:solidFill>
      <a:effectDag><a:cont><a:glow rad="127000"><a:srgbClr val="00FF00"><a:alpha val="60000"/></a:srgbClr></a:glow></a:cont></a:effectDag>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)
	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape, got %+v", elements)
	}
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &elements[0])
	if len(unsupported) != 1 || !strings.Contains(unsupported[0].Message, "flattened supported effect subset") || !elements[0].Rendered {
		t.Fatalf("expected flattened effectDag diagnostic with rendered shape, got unsupported=%+v rendered=%v", unsupported, elements[0].Rendered)
	}
	if got := img.RGBAAt(20, 48); got.G == 0 || got.A == 0 {
		t.Fatalf("expected flattened effectDag glow outside shape bounds, got %+v", got)
	}
}

func TestM10RenderShapePaintsBlurEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:         "sp",
		Name:         "Blurred Shape",
		PrstGeom:     "rect",
		HasTransform: true,
		OffX:         emuPerInch / 4,
		OffY:         emuPerInch / 4,
		ExtCX:        emuPerInch / 2,
		ExtCY:        emuPerInch / 2,
		HasFill:      true,
		FillColor:    color.RGBA{R: 255, A: 255},
		HasBlur:      true,
		BlurRadius:   emuPerInch / 10,
		BlurGrow:     true,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported blur render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	outside := img.RGBAAt(20, 48)
	if outside.R == 0 || outside.A == 0 {
		t.Fatalf("expected blurred red pixels outside shape bounds, got %+v", outside)
	}
	center := img.RGBAAt(48, 48)
	if center.R < 200 || center.A < 200 {
		t.Fatalf("expected blurred shape center to retain red fill, got %+v", center)
	}
}

func TestM10CollectSlideElementsParsesAlphaOutsetEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="21" name="Alpha Outset Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:alphaOutset rad="91440"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasAlphaOutset || got.AlphaOutsetRadius != 91440 {
		t.Fatalf("expected alphaOutset radius to parse, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "alphaOutset") {
		t.Fatalf("supported alphaOutset should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10RenderShapePaintsAlphaOutsetEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:              "sp",
		Name:              "Alpha Outset Shape",
		PrstGeom:          "rect",
		HasTransform:      true,
		OffX:              emuPerInch / 4,
		OffY:              emuPerInch / 4,
		ExtCX:             emuPerInch / 2,
		ExtCY:             emuPerInch / 2,
		HasFill:           true,
		FillColor:         color.RGBA{R: 255, A: 255},
		HasAlphaOutset:    true,
		AlphaOutsetRadius: emuPerInch / 10,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported alphaOutset render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(20, 48); got.R == 0 || got.A == 0 {
		t.Fatalf("expected alphaOutset red pixels outside shape bounds, got %+v", got)
	}
}

func TestM10CollectSlideElementsParsesRelativeOffsetEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="23" name="Relative Offset Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:relOff tx="25%" ty="-12.5%"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasRelativeOffset || got.RelativeOffsetX != 25000 || got.RelativeOffsetY != -12500 {
		t.Fatalf("expected relOff percentages to parse, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "relOff") {
		t.Fatalf("supported relOff should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10RenderShapePaintsRelativeOffsetEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:              "sp",
		Name:              "Relative Offset Shape",
		PrstGeom:          "rect",
		HasTransform:      true,
		OffX:              emuPerInch / 4,
		OffY:              emuPerInch / 4,
		ExtCX:             emuPerInch / 2,
		ExtCY:             emuPerInch / 2,
		HasFill:           true,
		FillColor:         color.RGBA{R: 255, A: 255},
		HasRelativeOffset: true,
		RelativeOffsetX:   25000,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported relOff render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(30, 48); got.A != 0 {
		t.Fatalf("expected original shape position to be transparent after relOff, got %+v", got)
	}
	if got := img.RGBAAt(80, 48); got.R == 0 || got.A == 0 {
		t.Fatalf("expected relOff red pixels at shifted shape position, got %+v", got)
	}
}

func TestM10CollectSlideElementsParsesTransformEffect(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="25" name="Transform Effect Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:xfrm tx="91440" ty="-45720"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	got := elements[0]
	if !got.HasEffectTransform || got.EffectTransformOffsetX != 91440 || got.EffectTransformOffsetY != -45720 || got.EffectTransformScaleX != 100000 || got.EffectTransformScaleY != 100000 {
		t.Fatalf("expected xfrm translation to parse with default scale, got %+v", got)
	}
	if strings.Contains(strings.Join(got.EffectUnsupported, " "), "xfrm") {
		t.Fatalf("supported xfrm translation should not be reported unsupported: %+v", got.EffectUnsupported)
	}
}

func TestM10CollectSlideElementsReportsTransformScaleSkewPartial(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<p:sld xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:cSld><p:spTree><p:sp>
    <p:nvSpPr><p:cNvPr id="26" name="Transform Scale Shape"/></p:nvSpPr>
    <p:spPr>
      <a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm>
      <a:prstGeom prst="rect"/>
      <a:effectLst><a:xfrm sx="120000" ky="600000"/></a:effectLst>
    </p:spPr>
  </p:sp></p:spTree></p:cSld>
</p:sld>`)

	elements := collectSlideElements(data)
	if len(elements) != 1 {
		t.Fatalf("expected one shape element, got %+v", elements)
	}
	if !containsString(elements[0].EffectUnsupported, "effect xfrm scale/skew transform was not rendered") {
		t.Fatalf("expected xfrm scale/skew partial diagnostic, got %+v", elements[0].EffectUnsupported)
	}
}

func TestM10RenderShapePaintsTransformTranslationEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                   "sp",
		Name:                   "Transform Effect Shape",
		PrstGeom:               "rect",
		HasTransform:           true,
		OffX:                   emuPerInch / 4,
		OffY:                   emuPerInch / 4,
		ExtCX:                  emuPerInch / 2,
		ExtCY:                  emuPerInch / 2,
		HasFill:                true,
		FillColor:              color.RGBA{R: 255, A: 255},
		HasEffectTransform:     true,
		EffectTransformScaleX:  100000,
		EffectTransformScaleY:  100000,
		EffectTransformOffsetX: emuPerInch / 10,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported xfrm translation render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(30, 48); got.A != 0 {
		t.Fatalf("expected original shape position to be transparent after xfrm translation, got %+v", got)
	}
	if got := img.RGBAAt(80, 48); got.R == 0 || got.A == 0 {
		t.Fatalf("expected xfrm-translated red pixels at shifted shape position, got %+v", got)
	}
}

func TestM10RenderShapePaintsFillOverlayEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:             "sp",
		Name:             "Overlay Shape",
		PrstGeom:         "rect",
		HasTransform:     true,
		OffX:             emuPerInch / 4,
		OffY:             emuPerInch / 4,
		ExtCX:            emuPerInch / 2,
		ExtCY:            emuPerInch / 2,
		HasFill:          true,
		FillColor:        color.RGBA{R: 255, B: 255, A: 255},
		HasFillOverlay:   true,
		FillOverlay:      backgroundPaint{Color: color.RGBA{G: 255, A: 255}},
		FillOverlayBlend: "mult",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported fillOverlay render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	got := img.RGBAAt(48, 48)
	if got.R != 0 || got.G != 0 || got.B != 0 || got.A == 0 {
		t.Fatalf("expected multiply fillOverlay to darken magenta shape with green overlay, got %+v", got)
	}
}

func TestM10FillOverlayImplementsSchemaBlendModes(t *testing.T) {
	base := color.RGBA{R: 80, G: 120, B: 200, A: 255}
	overlay := color.RGBA{R: 160, G: 90, B: 40, A: 255}
	tests := []struct {
		name string
		want color.RGBA
	}{
		{name: "over", want: overlay},
		{name: "mult", want: color.RGBA{R: 50, G: 42, B: 31, A: 255}},
		{name: "screen", want: color.RGBA{R: 190, G: 168, B: 209, A: 255}},
		{name: "darken", want: color.RGBA{R: 80, G: 90, B: 40, A: 255}},
		{name: "lighten", want: color.RGBA{R: 160, G: 120, B: 200, A: 255}},
	}
	for _, tt := range tests {
		got := fillOverlayBlendPixel(base, overlay, tt.name)
		if got != tt.want {
			t.Fatalf("blend mode %s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestM10RenderShapePaintsInnerShadowEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                "sp",
		Name:                "Inner Shadow Shape",
		PrstGeom:            "rect",
		HasTransform:        true,
		OffX:                emuPerInch / 4,
		OffY:                emuPerInch / 4,
		ExtCX:               emuPerInch / 2,
		ExtCY:               emuPerInch / 2,
		HasFill:             true,
		FillColor:           color.RGBA{R: 255, G: 255, B: 255, A: 255},
		HasInnerShadow:      true,
		InnerShadowColor:    color.RGBA{A: 220},
		InnerShadowBlur:     emuPerInch / 10,
		InnerShadowDistance: 0,
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported inner shadow render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	edge := img.RGBAAt(24, 48)
	center := img.RGBAAt(48, 48)
	if edge.R >= center.R || edge.G >= center.G || edge.B >= center.B {
		t.Fatalf("expected inner shadow to darken inside edge, got edge=%+v center=%+v", edge, center)
	}
}

func TestM10RenderShapePaintsReflectionEffect(t *testing.T) {
	size := slideSize{CX: emuPerInch, CY: emuPerInch}
	img := image.NewRGBA(image.Rect(0, 0, 96, 96))
	element := slideElement{
		Kind:                    "sp",
		Name:                    "Reflection Shape",
		PrstGeom:                "rect",
		HasTransform:            true,
		OffX:                    emuPerInch / 4,
		OffY:                    emuPerInch / 4,
		ExtCX:                   emuPerInch / 4,
		ExtCY:                   emuPerInch / 4,
		HasFill:                 true,
		FillColor:               color.RGBA{R: 255, A: 255},
		HasReflection:           true,
		ReflectionStartAlpha:    70000,
		ReflectionStartPosition: 0,
		ReflectionEndAlpha:      0,
		ReflectionEndPosition:   100000,
		ReflectionScaleY:        100000,
		ReflectionAlignment:     "b",
	}

	unsupported := renderShape("ppt/slides/slide1.xml", size, img, &element)
	if len(unsupported) != 0 || !element.Rendered {
		t.Fatalf("expected supported reflection render, got unsupported=%+v rendered=%v", unsupported, element.Rendered)
	}
	if got := img.RGBAAt(36, 49); got.R == 0 || got.A == 0 {
		t.Fatalf("expected reflected red pixels below shape bounds, got %+v", got)
	}
	if got := img.RGBAAt(36, 70); got.A >= img.RGBAAt(36, 49).A {
		t.Fatalf("expected reflection alpha to fade with distance, near=%+v far=%+v", img.RGBAAt(36, 49), got)
	}
}

func TestM10PictureBackendReportsUnsupportedShapeEffects(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 20))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	primitive := renderPicturePrimitive{
		Name:              "Picture With Unsupported Effect",
		Target:            ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 9, MaxY: 9},
		EffectUnsupported: []string{"effect reflection was not rendered"},
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 1 || unsupported[0].Code != partialUnsupportedCode || !strings.Contains(unsupported[0].Message, "effect reflection") {
		t.Fatalf("expected picture unsupported effect report, got %+v", unsupported)
	}
}

func TestM10PictureBackendPaintsGlowEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	primitive := renderPicturePrimitive{
		Name:        "Picture With Glow",
		Target:      ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasGlow:     true,
		GlowColor:   color.RGBA{B: 255, A: 160},
		GlowRadius:  emuPerInch / 10,
		ContentType: "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture glow render, got %+v", unsupported)
	}
	if got := img.RGBAAt(4, 10); got.B == 0 || got.A == 0 {
		t.Fatalf("expected blue picture glow outside image bounds, got %+v", got)
	}
}

func TestM10PictureBackendPaintsBlurEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:        "Picture With Blur",
		Target:      ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasBlur:     true,
		BlurRadius:  emuPerInch / 10,
		BlurGrow:    true,
		ContentType: "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture blur render, got %+v", unsupported)
	}
	if got := img.RGBAAt(4, 10); got.R == 0 || got.A == 0 {
		t.Fatalf("expected blurred red picture pixels outside image bounds, got %+v", got)
	}
}

func TestM10PictureBackendPaintsAlphaOutsetEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:              "Picture With Alpha Outset",
		Target:            ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasAlphaOutset:    true,
		AlphaOutsetRadius: emuPerInch / 10,
		ContentType:       "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture alphaOutset render, got %+v", unsupported)
	}
	if got := img.RGBAAt(4, 10); got.R == 0 || got.A == 0 {
		t.Fatalf("expected alphaOutset red picture pixels outside image bounds, got %+v", got)
	}
}

func TestM10PictureBackendPaintsRelativeOffsetEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:              "Picture With Relative Offset",
		Target:            ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasRelativeOffset: true,
		RelativeOffsetX:   25000,
		ContentType:       "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture relOff render, got %+v", unsupported)
	}
	if got := img.RGBAAt(7, 10); got.A != 0 {
		t.Fatalf("expected original picture position to be transparent after relOff, got %+v", got)
	}
	if got := img.RGBAAt(17, 10); got.R == 0 || got.A == 0 {
		t.Fatalf("expected relOff red picture pixels at shifted position, got %+v", got)
	}
}

func TestM10PictureBackendPaintsTransformTranslationEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:                   "Picture With Transform Effect",
		Target:                 ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasEffectTransform:     true,
		EffectTransformScaleX:  100000,
		EffectTransformScaleY:  100000,
		EffectTransformOffsetX: emuPerInch / 4,
		ContentType:            "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture xfrm translation render, got %+v", unsupported)
	}
	if got := img.RGBAAt(7, 10); got.A != 0 {
		t.Fatalf("expected original picture position to be transparent after xfrm translation, got %+v", got)
	}
	if got := img.RGBAAt(17, 10); got.R == 0 || got.A == 0 {
		t.Fatalf("expected xfrm-translated red picture pixels at shifted position, got %+v", got)
	}
}

func TestM10PictureBackendPaintsFillOverlayEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:             "Picture With Fill Overlay",
		Target:           ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasFillOverlay:   true,
		FillOverlay:      backgroundPaint{Color: color.RGBA{B: 255, A: 255}},
		FillOverlayBlend: "screen",
		ContentType:      "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture fillOverlay render, got %+v", unsupported)
	}
	got := img.RGBAAt(10, 10)
	if got.R != 255 || got.B != 255 || got.A == 0 {
		t.Fatalf("expected screen fillOverlay to add blue over red picture, got %+v", got)
	}
}

func TestM10PictureBackendPaintsInnerShadowEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 24))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:                "Picture With Inner Shadow",
		Target:              ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasInnerShadow:      true,
		InnerShadowColor:    color.RGBA{A: 220},
		InnerShadowBlur:     emuPerInch / 10,
		InnerShadowDistance: 0,
		ContentType:         "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture inner shadow render, got %+v", unsupported)
	}
	edge := img.RGBAAt(6, 10)
	center := img.RGBAAt(10, 10)
	if edge.R >= center.R || edge.G >= center.G || edge.B >= center.B {
		t.Fatalf("expected picture inner shadow to darken inside edge, got edge=%+v center=%+v", edge, center)
	}
}

func TestM10PictureBackendPaintsReflectionEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 24, 32))
	source := image.NewRGBA(image.Rect(0, 0, 4, 4))
	draw.Draw(source, source.Bounds(), &image.Uniform{C: color.RGBA{G: 255, A: 255}}, image.Point{}, draw.Src)
	primitive := renderPicturePrimitive{
		Name:                    "Picture With Reflection",
		Target:                  ObjectPixelBounds{MinX: 6, MinY: 6, MaxX: 15, MaxY: 15},
		HasReflection:           true,
		ReflectionStartAlpha:    70000,
		ReflectionStartPosition: 0,
		ReflectionEndAlpha:      0,
		ReflectionEndPosition:   100000,
		ReflectionScaleY:        100000,
		ContentType:             "image/png",
	}

	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: primitive,
		Source:    source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported picture reflection render, got %+v", unsupported)
	}
	if got := img.RGBAAt(10, 16); got.G == 0 || got.A == 0 {
		t.Fatalf("expected reflected green picture pixels below image bounds, got %+v", got)
	}
	if got := img.RGBAAt(10, 24); got.A >= img.RGBAAt(10, 16).A {
		t.Fatalf("expected picture reflection alpha to fade with distance, near=%+v far=%+v", img.RGBAAt(10, 16), got)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
