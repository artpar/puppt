package render

import (
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
)

func TestM07ParsesBlipFillModeLinkAndEffects(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:pic xmlns:p="p" xmlns:a="a" xmlns:r="r">
		<p:nvPicPr><p:cNvPr id="7" name="Linked Picture"/></p:nvPicPr>
		<p:blipFill rotWithShape="0">
			<a:blip r:link="rIdLink" cstate="print">
				<a:alphaRepl a="50000"/>
				<a:biLevel thresh="42000"/>
				<a:grayscl/>
				<a:lum bright="10000" contrast="-20000"/>
				<a:hsl hue="7200000" sat="10000" lum="-5000"/>
				<a:tint hue="7200000" amt="50000"/>
				<a:blur rad="101600" grow="0"/>
				<a:fillOverlay blend="screen"><a:solidFill><a:srgbClr val="0000FF"/></a:solidFill></a:fillOverlay>
				<a:clrRepl><a:srgbClr val="336699"/></a:clrRepl>
				<a:alphaMod><a:cont><a:alphaModFix amt="50000"/></a:cont></a:alphaMod>
			</a:blip>
			<a:srcRect l="-10000" t="20000" r="30000" b="-40000"/>
			<a:tile tx="914400" ty="457200" sx="50000" sy="200000" flip="xy" algn="ctr"/>
		</p:blipFill>
		<p:spPr><a:xfrm><a:off x="0" y="0"/><a:ext cx="914400" cy="914400"/></a:xfrm></p:spPr>
	</p:pic>`))
	if err != nil {
		t.Fatal(err)
	}

	element := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if element.EmbedID != "" || element.LinkID != "rIdLink" {
		t.Fatalf("expected linked image relationship, got embed=%q link=%q", element.EmbedID, element.LinkID)
	}
	if element.BlipCompressionState != "print" {
		t.Fatalf("expected blip compression state metadata, got %+v", element)
	}
	if element.BlipFillMode != "tile" || element.BlipTileScaleX != 50000 || element.BlipTileScaleY != 200000 || element.BlipTileFlip != "xy" || element.BlipTileAlignment != "ctr" {
		t.Fatalf("expected tile fill properties, got %+v", element)
	}
	if !element.HasCrop || element.CropLeft != -10000 || element.CropTop != 20000 || element.CropRight != 30000 || element.CropBottom != -40000 {
		t.Fatalf("expected signed srcRect crop/padding, got %+v", element)
	}
	if !element.HasImageAlphaReplace || element.ImageAlphaReplacePct != 50000 || !element.HasImageBiLevel || element.ImageBiLevelThreshold != 42000 || !element.HasImageGrayscale {
		t.Fatalf("expected parsed alpha/color effects, got %+v", element)
	}
	if !element.HasImageLuminance || element.ImageLuminanceBright != 10000 || element.ImageLuminanceContrast != -20000 || !element.HasImageColorReplace {
		t.Fatalf("expected parsed luminance/color replace effects, got %+v", element)
	}
	if !element.HasImageHSL || element.ImageHSLHue != 7200000 || element.ImageHSLSaturation != 10000 || element.ImageHSLLuminance != -5000 {
		t.Fatalf("expected parsed HSL effect, got %+v", element)
	}
	if !element.HasImageTint || element.ImageTintHue != 7200000 || element.ImageTintAmount != 50000 {
		t.Fatalf("expected parsed tint effect, got %+v", element)
	}
	if !element.HasImageBlur || element.ImageBlurRadius != 101600 || element.ImageBlurGrow {
		t.Fatalf("expected parsed blip blur effect, got %+v", element)
	}
	if !element.HasImageFillOverlay || element.ImageFillOverlayBlend != "screen" || element.ImageFillOverlay.Color != (color.RGBA{B: 255, A: 255}) {
		t.Fatalf("expected parsed blip fillOverlay effect, got %+v", element)
	}
	if !element.HasImageAlphaModulate || element.ImageAlphaModulatePct != 50000 {
		t.Fatalf("expected parsed blip alphaMod effect, got %+v", element)
	}
	if len(element.ImageUnsupported) != 0 {
		t.Fatalf("expected all parsed blip effects to be supported, got %+v", element.ImageUnsupported)
	}
}

func TestM07ReportsUnsupportedBlipAlphaModContainer(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:blip xmlns:a="a"><a:alphaMod/></a:blip>`))
	if err != nil {
		t.Fatal(err)
	}

	var element slideElement
	parseBlipEffects(root, &element)
	if len(element.ImageUnsupported) != 1 || !strings.Contains(element.ImageUnsupported[0], "alphaMod container") {
		t.Fatalf("expected unsupported alphaMod container diagnostic, got %+v", element.ImageUnsupported)
	}
}

func TestM07PictureSourceAppliesBlipAlphaColorAndLuminanceEffects(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 3, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 200, G: 10, B: 10, A: 120})
	src.SetRGBA(1, 0, color.RGBA{R: 20, G: 80, B: 140, A: 200})
	src.SetRGBA(2, 0, color.RGBA{R: 10, G: 10, B: 10, A: 80})

	got, bounds := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasAlphaModFix:        true,
		AlphaModFixPct:        50000,
		HasAlphaBiLevel:       true,
		AlphaBiLevelThreshold: 30000,
		HasColorChange:        true,
		ColorChangeFrom:       color.RGBA{R: 200, G: 10, B: 10, A: 255},
		ColorChangeTo:         color.RGBA{B: 255, A: 255},
		ColorChangeUseAlpha:   false,
		HasGrayscale:          true,
		HasLuminance:          true,
		LuminanceBright:       10000,
		LuminanceContrast:     0,
	})
	if bounds != image.Rect(0, 0, 3, 1) {
		t.Fatalf("unexpected transformed bounds: %v", bounds)
	}
	first := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if first.A != 0 || first.R != first.G || first.G != first.B {
		t.Fatalf("expected first pixel to be recolored, grayscaled, brightened, and alpha-thresholded, got %#v", first)
	}
	second := color.RGBAModel.Convert(got.At(1, 0)).(color.RGBA)
	if second.A != 255 || second.R != second.G || second.G != second.B || second.R <= 80 {
		t.Fatalf("expected second pixel to keep visible alpha and brighten grayscale color, got %#v", second)
	}
}

func TestM07PictureSourceAppliesBlipAlphaModulateEffect(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 20, G: 80, B: 140, A: 200})

	got, bounds := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasAlphaModulate: true,
		AlphaModulatePct: 50000,
	})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel != (color.RGBA{R: 20, G: 80, B: 140, A: 100}) {
		t.Fatalf("expected alphaMod container scalar to modulate source alpha, got %#v", pixel)
	}
}

func TestM07PictureSourceAppliesBlipHSLEffect(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})

	got, bounds := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasHSL: true,
		HSLHue: 7200000,
	})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel.G <= 240 || pixel.R >= 20 || pixel.B >= 20 || pixel.A != 255 {
		t.Fatalf("expected HSL hue offset to rotate red toward green, got %#v", pixel)
	}
}

func TestM07PictureSourceAppliesBlipTintEffect(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})

	got, bounds := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasTint:    true,
		TintHue:    7200000,
		TintAmount: 100000,
	})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel.G <= 240 || pixel.R >= 20 || pixel.B >= 20 || pixel.A != 255 {
		t.Fatalf("expected tint to shift red toward green, got %#v", pixel)
	}
}

func TestM07PictureBackendAppliesBlipBlurEffect(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 9, 9))
	source := image.NewRGBA(image.Rect(0, 0, 9, 9))
	source.SetRGBA(4, 4, color.RGBA{R: 255, A: 255})
	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: renderPicturePrimitive{
			Name:             "Picture With Blip Blur",
			Target:           ObjectPixelBounds{MinX: 0, MinY: 0, MaxX: 9, MaxY: 9},
			HasSourceBlur:    true,
			SourceBlurRadius: emuPerInch / 9,
			SourceBlurGrow:   true,
			ContentType:      "image/png",
		},
		Source: source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported blip blur render, got %+v", unsupported)
	}
	center := img.RGBAAt(4, 4)
	neighbor := img.RGBAAt(4, 3)
	if center.A >= 255 || neighbor.A == 0 || neighbor.R == 0 {
		t.Fatalf("expected blur to spread center pixel into neighbor, center=%+v neighbor=%+v", center, neighbor)
	}
}

func TestM07PictureBackendKeepsRotatedOutlineOutsideBlipBlur(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	source := image.NewRGBA(image.Rect(0, 0, 8, 8))
	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart: "ppt/slides/slide1.xml",
		Size:      slideSize{CX: emuPerInch, CY: emuPerInch},
		Canvas:    img,
		Primitive: renderPicturePrimitive{
			Name:             "Rotated Picture With Blip Blur And Outline",
			Target:           ObjectPixelBounds{MinX: 8, MinY: 8, MaxX: 23, MaxY: 23},
			RotationDegrees:  90,
			RotatesWithShape: true,
			HasSourceBlur:    true,
			SourceBlurRadius: emuPerInch / 8,
			SourceBlurGrow:   true,
			HasLine:          true,
			LineColor:        color.RGBA{G: 255, A: 255},
			LineWidth:        emuPerInch / 8,
			ContentType:      "image/png",
		},
		Source: source,
	})
	if len(unsupported) != 0 {
		t.Fatalf("expected supported rotated blip blur with outline render, got %+v", unsupported)
	}
	pureOutlinePixels := 0
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y) == (color.RGBA{G: 255, A: 255}) {
				pureOutlinePixels++
			}
		}
	}
	if pureOutlinePixels == 0 {
		t.Fatalf("expected unblurred post-blip-effect outline pixels")
	}
}

func TestM07PictureSourceAppliesBlipFillOverlayEffect(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})

	got, bounds := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasSourceFillOverlay:   true,
		SourceFillOverlay:      backgroundPaint{Color: color.RGBA{B: 255, A: 255}},
		SourceFillOverlayBlend: "screen",
	})
	if bounds != image.Rect(0, 0, 1, 1) {
		t.Fatalf("unexpected transformed bounds: %v", bounds)
	}
	pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	if pixel != (color.RGBA{R: 255, B: 255, A: 255}) {
		t.Fatalf("expected source-space screen fillOverlay to blend blue over red, got %#v", pixel)
	}
}

func TestM07PictureSourceAppliesBlipBiLevelAndDuotone(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 10, G: 10, B: 10, A: 123})
	src.SetRGBA(1, 0, color.RGBA{R: 240, G: 240, B: 240, A: 231})

	got, _ := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasBiLevel:       true,
		BiLevelThreshold: 50000,
		HasDuotone:       true,
		DuotoneDark:      color.RGBA{A: 255},
		DuotoneLight:     color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})

	dark := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA)
	light := color.RGBAModel.Convert(got.At(1, 0)).(color.RGBA)
	if dark != (color.RGBA{A: 123}) {
		t.Fatalf("expected dark bi-level duotone pixel with preserved alpha, got %#v", dark)
	}
	if light != (color.RGBA{R: 255, G: 255, B: 255, A: 231}) {
		t.Fatalf("expected light bi-level duotone pixel with preserved alpha, got %#v", light)
	}
}

func TestM07PictureSourceAppliesBlipColorReplace(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 90})

	got, _ := pictureSourceForPrimitive(src, renderPicturePrimitive{
		HasColorReplace: true,
		ColorReplace:    color.RGBA{R: 100, G: 150, B: 200, A: 255},
	})
	if pixel := color.RGBAModel.Convert(got.At(0, 0)).(color.RGBA); pixel != (color.RGBA{R: 100, G: 150, B: 200, A: 90}) {
		t.Fatalf("expected color replacement with preserved source alpha, got %#v", pixel)
	}
}

func TestM07TileBlipFillRepeatsSourceImage(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 4, 2))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.RGBA{R: 255, G: 255, B: 255, A: 255}}, image.Point{}, draw.Src)
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	src.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})

	tileImage(dst, dst.Bounds(), src, src.Bounds(), renderPicturePrimitive{BlipFillMode: "tile"}, slideSize{CX: emuPerInch, CY: emuPerInch}, 96)
	for y := 0; y < 2; y++ {
		if got := dst.RGBAAt(0, y); got != (color.RGBA{R: 255, A: 255}) {
			t.Fatalf("row %d first tile pixel = %#v", y, got)
		}
		if got := dst.RGBAAt(1, y); got != (color.RGBA{B: 255, A: 255}) {
			t.Fatalf("row %d second tile pixel = %#v", y, got)
		}
		if got := dst.RGBAAt(2, y); got != (color.RGBA{R: 255, A: 255}) {
			t.Fatalf("row %d repeated first tile pixel = %#v", y, got)
		}
	}
}

func TestM07RenderPicturePrimitiveSupportsInternalLinkedImageRelationship(t *testing.T) {
	pkg := &pptx.Package{
		ContentTypes: pptx.ContentTypes{Defaults: map[string]string{"png": "image/png"}},
	}
	element := slideElement{
		Kind:         "pic",
		Name:         "Linked",
		LinkID:       "rIdLink",
		HasTransform: true,
		ExtCX:        emuPerInch,
		ExtCY:        emuPerInch,
	}
	primitive, err := renderPicturePrimitiveFromElement(pkg, "ppt/slides/slide1.xml", slideSize{CX: emuPerInch, CY: emuPerInch}, image.Rect(0, 0, 96, 96), element, map[string]pptx.Relationship{
		"rIdLink": {ID: "rIdLink", Type: pptx.ImageRelType, Target: "../media/linked.png"},
	})
	if err != nil {
		t.Fatalf("internal linked image relationship should be renderable: %v", err)
	}
	if primitive.RelationshipID != "" || primitive.LinkRelationshipID != "rIdLink" || primitive.MediaPart != "ppt/media/linked.png" {
		t.Fatalf("unexpected linked primitive: %+v", primitive)
	}
}
