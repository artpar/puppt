package render

import (
	"image/color"
	"testing"
)

func TestColorFromColorNodeAppliesDrawingMLModifiers(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent5"><a:lumMod val="20000"/><a:lumOff val="80000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 219 || got.G != 238 || got.B != 244 || got.A != 255 {
		t.Fatalf("unexpected modified color: %#v", got)
	}
}

func TestColorFromColorNodeAcceptsDrawingMLPercentStringModifiers(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent5"><a:lumMod val="20%"/><a:lumOff val="80.000%"/><a:alpha val="50%"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 219 || got.G != 238 || got.B != 244 || got.A != 127 {
		t.Fatalf("unexpected percent-string modified color: %#v", got)
	}
}

func TestColorFromColorNodeRoundsLuminanceModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent6"><a:lumMod val="50000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent6": {R: 0x70, G: 0xad, B: 0x47, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 56, G: 87, B: 36, A: 255}) {
		t.Fatalf("unexpected rounded luminance color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesLuminanceInHSLSpace(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:lumMod val="50000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 32, G: 56, B: 100, A: 255}) {
		t.Fatalf("unexpected HSL luminance color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLTint(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:tint val="20000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 233, G: 235, B: 245, A: 255}) {
		t.Fatalf("unexpected tinted color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLTintInLinearLight(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="00FF00"><a:tint val="50%"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 0xbc, G: 0xff, B: 0xbc, A: 255}) {
		t.Fatalf("unexpected linear-light tinted color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesDrawingMLShade(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:schemeClr val="accent1"><a:shade val="40000"/></a:schemeClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNodeWithTheme(root, themeColors{"accent1": {R: 0x44, G: 0x72, B: 0xc4, A: 0xff}})
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 42, G: 73, B: 129, A: 255}) {
		t.Fatalf("unexpected shaded color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesAlphaModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="804020"><a:alpha val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0x80 || got.G != 0x40 || got.B != 0x20 || got.A != 127 {
		t.Fatalf("unexpected alpha-modified color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesAlphaOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="804020"><a:alphaOff val="-10000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0x80 || got.G != 0x40 || got.B != 0x20 || got.A != 230 {
		t.Fatalf("unexpected alpha-offset color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesSaturationModifier(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="806060"><a:satMod val="200000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R <= 0x80 || got.G >= 0x60 || got.B >= 0x60 || got.A != 255 {
		t.Fatalf("expected saturation modifier to widen channel contrast, got %#v", got)
	}
}

func TestColorFromColorNodeAppliesSaturationOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="806060"><a:satOff val="-50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R >= 0x80 || got.G <= 0x60 || got.B <= 0x60 || got.A != 255 {
		t.Fatalf("expected saturation offset to narrow channel contrast, got %#v", got)
	}
}

func TestColorFromColorNodeAppliesHueOffset(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="FF0000"><a:hueOff val="7200000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 0 || got.G != 255 || got.B != 0 || got.A != 255 {
		t.Fatalf("unexpected hue-offset color: %#v", got)
	}
}

func TestColorFromColorNodeAppliesModifiersInDocumentOrder(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="808080"><a:tint val="50000"/><a:shade val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	reversedRoot, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:srgbClr val="808080"><a:shade val="50000"/><a:tint val="50000"/></a:srgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	reversed, ok := colorFromColorNode(reversedRoot)
	if !ok {
		t.Fatal("reversed color was not parsed")
	}
	if got == reversed {
		t.Fatalf("modifier order should affect color: ordered=%#v reversed=%#v", got, reversed)
	}
	if got != (color.RGBA{R: 150, G: 150, B: 150, A: 255}) {
		t.Fatalf("unexpected ordered color: %#v", got)
	}
}

func TestColorFromColorNodeParsesScRGB(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:scrgbClr r="50000" g="0" b="100000"><a:alpha val="50000"/></a:scrgbClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got.R != 188 || got.G != 0 || got.B != 255 || got.A != 127 {
		t.Fatalf("unexpected scrgb color: %#v", got)
	}
}

func TestColorFromColorNodeClampsScRGBBeforeSRGBTransfer(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:scrgbClr r="-50000" g="150000" b="25000"/></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 0, G: 255, B: 137, A: 255}) {
		t.Fatalf("unexpected clamped scrgb color: %#v", got)
	}
}

func TestColorFromColorNodeParsesSystemColorLastColor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:sysClr val="windowText" lastClr="123456"><a:alpha val="50000"/></a:sysClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 0x12, G: 0x34, B: 0x56, A: 127}) {
		t.Fatalf("unexpected system color: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetRed(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:prstClr val="red"><a:alpha val="25000"/></a:prstClr></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 255, A: 63}) {
		t.Fatalf("unexpected preset red: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetBlack(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:effectLst xmlns:a="a"><a:outerShdw><a:prstClr val="black"><a:alpha val="40000"/></a:prstClr></a:outerShdw></a:effectLst>`))
	if err != nil {
		t.Fatal(err)
	}
	shadow := firstDescendant(root, "outerShdw")
	got, ok := colorFromColorNode(shadow)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{A: 102}) {
		t.Fatalf("unexpected preset black: %#v", got)
	}
}

func TestColorFromColorNodeParsesPresetWhite(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:solidFill xmlns:a="a"><a:prstClr val="white"/></a:solidFill>`))
	if err != nil {
		t.Fatal(err)
	}
	got, ok := colorFromColorNode(root)
	if !ok {
		t.Fatal("color was not parsed")
	}
	if got != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
		t.Fatalf("unexpected preset white: %#v", got)
	}
}
