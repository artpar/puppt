package render

import (
	"bytes"
	"compress/zlib"
	"encoding/xml"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
)

func renderPicture(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	primitive, err := renderPicturePrimitiveFromElement(pkg, slidePart, size, img.Bounds(), *element, relationships)
	if err != nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, err.Error())}
	}
	relationshipID := primitive.RelationshipID
	if relationshipID == "" {
		relationshipID = primitive.LinkRelationshipID
	}
	relationship := relationships[relationshipID]

	source, targetPart, partialUnsupported := pictureSourceImage(pkg, slidePart, element, relationships, relationship)
	if source == nil {
		return []model.SkipItem{pictureUnsupported(slidePart, element, fmt.Sprintf("picture object %q uses unsupported image data %q: %v", elementLabel(*element), targetPart, partialUnsupported))}
	}
	element.ImageMediaPart = targetPart
	element.ImageContentType = primitive.ContentType
	sourceBounds := source.Bounds()
	element.ImageWidth = sourceBounds.Dx()
	element.ImageHeight = sourceBounds.Dy()
	unsupported := currentPictureBackend{}.RenderPicture(pictureBackendInput{
		SlidePart:          slidePart,
		Size:               size,
		Canvas:             img,
		Primitive:          primitive,
		Source:             source,
		TargetPart:         targetPart,
		PartialUnsupported: partialUnsupported,
	})
	element.Rendered = true
	return unsupported
}

type pictureBackend interface {
	RenderPicture(input pictureBackendInput) []model.SkipItem
}

type pictureBackendInput struct {
	SlidePart          string
	Size               slideSize
	Canvas             *image.RGBA
	Primitive          renderPicturePrimitive
	Source             image.Image
	TargetPart         string
	PartialUnsupported error
}

type pictureSamplingStage interface {
	Draw(input pictureSamplingInput) bool
}

type pictureSamplingInput struct {
	Canvas       *image.RGBA
	Target       image.Rectangle
	Source       image.Image
	SourceBounds image.Rectangle
	Primitive    renderPicturePrimitive
	Size         slideSize
	OutputWidth  int
}

type currentPictureSamplingStage struct{}

type currentPictureBackend struct {
	sampler pictureSamplingStage
}

func (backend currentPictureBackend) RenderPicture(input pictureBackendInput) []model.SkipItem {
	primitive := input.Primitive
	img := input.Canvas
	size := input.Size
	target := imageRectFromObjectPixelBounds(primitive.Target)
	var unsupported []model.SkipItem
	if primitive.HasEffectTransform && (primitive.EffectTransformOffsetX != 0 || primitive.EffectTransformOffsetY != 0) {
		xfrmUnsupported, xfrmRendered := backend.drawPicturePrimitiveWithEffectTransform(input)
		unsupported = append(unsupported, xfrmUnsupported...)
		if xfrmRendered {
			return unsupported
		}
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q xfrm effect was not rendered", picturePrimitiveLabel(primitive))))
		return unsupported
	}
	if primitive.HasRelativeOffset && (primitive.RelativeOffsetX != 0 || primitive.RelativeOffsetY != 0) {
		relOffUnsupported, relOffRendered := backend.drawPicturePrimitiveWithRelativeOffset(input)
		unsupported = append(unsupported, relOffUnsupported...)
		if relOffRendered {
			return unsupported
		}
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q relOff effect was not rendered", picturePrimitiveLabel(primitive))))
		return unsupported
	}
	if primitive.HasShadow {
		for _, message := range picturePrimitiveShadowTransformUnsupportedMessages(primitive) {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", picturePrimitiveLabel(primitive), message)))
		}
		if drawPicturePrimitiveShadow(img, target, primitive, size) {
			// Supported picture shadows are painted before the image so the image occludes the inner shadow area.
		} else if primitive.ShadowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q outer shadow geometry was not rendered", picturePrimitiveLabel(primitive))))
		}
	}
	if primitive.HasGlow {
		if drawPicturePrimitiveGlow(img, target, primitive, size) {
			// Supported picture glows are painted before the image so the image occludes the inner glow area.
		} else if primitive.GlowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q glow geometry was not rendered", picturePrimitiveLabel(primitive))))
		}
	}
	for _, message := range picturePrimitiveShape3DUnsupportedMessages(primitive) {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", picturePrimitiveLabel(primitive), message)))
	}
	for _, message := range primitive.EffectUnsupported {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", picturePrimitiveLabel(primitive), message)))
	}
	for _, message := range primitive.ImageUnsupported {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", picturePrimitiveLabel(primitive), message)))
	}
	if primitive.BlipFillMode == "tile" && (primitive.HasSoftEdge || len(primitive.CustomPath) >= 3) {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q tiled blip fill with mask/soft edge was rendered as stretched image", picturePrimitiveLabel(primitive))))
	}
	pictureImage, pictureBounds := pictureSourceForPrimitive(input.Source, primitive)
	softEdgeRendered := false
	blurRendered := false
	fillOverlayRendered := false
	innerShadowRendered := false
	reflectionRendered := false
	alphaOutsetRendered := false
	sourceBlurRendered := false
	if primitive.HasReflection {
		softEdgeRendered, reflectionRendered = backend.drawPicturePrimitiveWithReflection(img, target, pictureImage, pictureBounds, primitive, size)
		if reflectionRendered {
			innerShadowRendered = primitive.HasInnerShadow
			fillOverlayRendered = primitive.HasFillOverlay
			blurRendered = primitive.HasBlur && primitive.BlurRadius > 0
			alphaOutsetRendered = primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0
		}
	} else if primitive.HasInnerShadow {
		softEdgeRendered, innerShadowRendered = backend.drawPicturePrimitiveWithInnerShadow(img, target, pictureImage, pictureBounds, primitive, size)
		if innerShadowRendered {
			fillOverlayRendered = primitive.HasFillOverlay
			blurRendered = primitive.HasBlur && primitive.BlurRadius > 0
			alphaOutsetRendered = primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0
		}
	} else if primitive.HasFillOverlay {
		softEdgeRendered, fillOverlayRendered = backend.drawPicturePrimitiveWithFillOverlay(img, target, pictureImage, pictureBounds, primitive, size)
		if fillOverlayRendered {
			alphaOutsetRendered = primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0
		}
	} else if primitive.HasBlur && primitive.BlurRadius > 0 {
		softEdgeRendered, blurRendered = backend.drawPicturePrimitiveWithBlur(img, target, pictureImage, pictureBounds, primitive, size)
		if blurRendered {
			alphaOutsetRendered = primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0
		}
	} else if primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0 {
		softEdgeRendered, alphaOutsetRendered = backend.drawPicturePrimitiveWithAlphaOutset(img, target, pictureImage, pictureBounds, primitive, size)
	} else if primitive.HasSourceBlur && primitive.SourceBlurRadius > 0 {
		softEdgeRendered, sourceBlurRendered = backend.drawPicturePrimitiveWithSourceBlur(img, target, pictureImage, pictureBounds, primitive, size)
	} else {
		softEdgeRendered = backend.samplingStage().Draw(pictureSamplingInput{
			Canvas:       img,
			Target:       target,
			Source:       pictureImage,
			SourceBounds: pictureBounds,
			Primitive:    primitive,
			Size:         size,
			OutputWidth:  img.Bounds().Dx(),
		})
	}
	if primitive.HasLine && !primitive.NoLine && !primitive.HasReflection && !primitive.HasInnerShadow && !primitive.HasBlur && !primitive.HasAlphaOutset && !primitive.HasFillOverlay && !primitive.HasSourceBlur {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		if primitive.RotationDegrees == 0 {
			drawPicturePrimitiveOutline(img, target, primitive, lineWidth)
		}
	}
	if primitive.HasBlur && !blurRendered {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q blur effect was not rendered", picturePrimitiveLabel(primitive))))
	}
	if primitive.HasSourceBlur && primitive.SourceBlurRadius > 0 && !sourceBlurRendered {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q blip blur effect was not rendered with combined object effects", picturePrimitiveLabel(primitive))))
	}
	if primitive.HasAlphaOutset && primitive.AlphaOutsetRadius > 0 && !alphaOutsetRendered {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q alphaOutset effect was not rendered", picturePrimitiveLabel(primitive))))
	}
	if primitive.HasFillOverlay && !fillOverlayRendered {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q fillOverlay effect was not rendered", picturePrimitiveLabel(primitive))))
	}
	if primitive.HasInnerShadow && !innerShadowRendered && primitive.InnerShadowColor.A != 0 {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q inner shadow effect was not rendered", picturePrimitiveLabel(primitive))))
	}
	if primitive.HasReflection && !reflectionRendered {
		unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q reflection effect was not rendered", picturePrimitiveLabel(primitive))))
	}

	if len(primitive.CustomPath) >= 3 && len(primitive.CustomPathUnsupported) > 0 {
		for _, message := range primitive.CustomPathUnsupported {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q %s", picturePrimitiveLabel(primitive), message)))
		}
	}
	if primitive.HasSoftEdge {
		if !softEdgeRendered {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q soft edge was not rendered", picturePrimitiveLabel(primitive))))
		}
	}
	if input.PartialUnsupported != nil {
		if strings.EqualFold(path.Ext(input.TargetPart), ".svg") {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from SVG because fallback raster could not be decoded: %v", picturePrimitiveLabel(primitive), input.PartialUnsupported)))
		} else {
			unsupported = append(unsupported, unsupportedItem(input.SlidePart, partialUnsupportedCode, fmt.Sprintf("picture object %q rendered from fallback raster because SVG image could not be decoded: %v", picturePrimitiveLabel(primitive), input.PartialUnsupported)))
		}
	}
	return unsupported
}

func (backend currentPictureBackend) drawPicturePrimitiveWithReflection(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasReflection = false
	inner.ReflectionBlur = 0
	inner.ReflectionDistance = 0
	inner.ReflectionDirection = 0
	softEdgeRendered := false
	if inner.HasInnerShadow {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithInnerShadow(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasFillOverlay {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithFillOverlay(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasBlur && inner.BlurRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithBlur(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasAlphaOutset && inner.AlphaOutsetRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithAlphaOutset(layer, target, pictureImage, pictureBounds, inner, size)
	} else {
		softEdgeRendered = backend.samplingStage().Draw(pictureSamplingInput{
			Canvas:       layer,
			Target:       target,
			Source:       pictureImage,
			SourceBounds: pictureBounds,
			Primitive:    inner,
			Size:         size,
			OutputWidth:  img.Bounds().Dx(),
		})
	}
	if primitive.HasLine && !primitive.NoLine && primitive.RotationDegrees == 0 {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		drawPicturePrimitiveOutline(layer, target, primitive, lineWidth)
	}
	if !applyReflection(layer, target, primitive.ReflectionParameters(), size, img.Bounds().Dx()) {
		return softEdgeRendered, false
	}
	draw.Draw(img, img.Bounds(), layer, image.Point{}, draw.Over)
	return softEdgeRendered, true
}

func (primitive renderPicturePrimitive) ReflectionParameters() reflectionParameters {
	return reflectionParameters{
		Blur:          primitive.ReflectionBlur,
		StartAlpha:    primitive.ReflectionStartAlpha,
		StartPosition: primitive.ReflectionStartPosition,
		EndAlpha:      primitive.ReflectionEndAlpha,
		EndPosition:   primitive.ReflectionEndPosition,
		Distance:      primitive.ReflectionDistance,
		Direction:     primitive.ReflectionDirection,
		ScaleY:        primitive.ReflectionScaleY,
	}
}

func (backend currentPictureBackend) drawPicturePrimitiveWithRelativeOffset(input pictureBackendInput) ([]model.SkipItem, bool) {
	target := imageRectFromObjectPixelBounds(input.Primitive.Target)
	if target.Empty() {
		return nil, false
	}
	offset := relativeOffsetPixels(target, input.Primitive.RelativeOffsetX, input.Primitive.RelativeOffsetY)
	if offset == (image.Point{}) {
		return nil, false
	}
	layer := image.NewRGBA(input.Canvas.Bounds())
	inner := input.Primitive
	inner.HasRelativeOffset = false
	inner.RelativeOffsetX = 0
	inner.RelativeOffsetY = 0
	innerInput := input
	innerInput.Canvas = layer
	innerInput.Primitive = inner
	unsupported := backend.RenderPicture(innerInput)
	drawRGBAAt(input.Canvas, layer.Bounds().Add(offset), layer)
	return unsupported, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithEffectTransform(input pictureBackendInput) ([]model.SkipItem, bool) {
	target := imageRectFromObjectPixelBounds(input.Primitive.Target)
	if target.Empty() {
		return nil, false
	}
	offset := effectTransformOffsetPixels(input.Size, input.Canvas.Bounds(), input.Primitive.EffectTransformOffsetX, input.Primitive.EffectTransformOffsetY)
	if offset == (image.Point{}) {
		return nil, false
	}
	layer := image.NewRGBA(input.Canvas.Bounds())
	inner := input.Primitive
	inner.HasEffectTransform = false
	inner.EffectTransformScaleX = 0
	inner.EffectTransformScaleY = 0
	inner.EffectTransformSkewX = 0
	inner.EffectTransformSkewY = 0
	inner.EffectTransformOffsetX = 0
	inner.EffectTransformOffsetY = 0
	innerInput := input
	innerInput.Canvas = layer
	innerInput.Primitive = inner
	unsupported := backend.RenderPicture(innerInput)
	drawRGBAAt(input.Canvas, layer.Bounds().Add(offset), layer)
	return unsupported, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithInnerShadow(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasInnerShadow = false
	inner.InnerShadowColor = color.RGBA{}
	inner.InnerShadowBlur = 0
	inner.InnerShadowDistance = 0
	inner.InnerShadowDirection = 0
	softEdgeRendered := false
	if inner.HasFillOverlay {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithFillOverlay(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasBlur && inner.BlurRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithBlur(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasAlphaOutset && inner.AlphaOutsetRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithAlphaOutset(layer, target, pictureImage, pictureBounds, inner, size)
	} else {
		softEdgeRendered = backend.samplingStage().Draw(pictureSamplingInput{
			Canvas:       layer,
			Target:       target,
			Source:       pictureImage,
			SourceBounds: pictureBounds,
			Primitive:    inner,
			Size:         size,
			OutputWidth:  img.Bounds().Dx(),
		})
	}
	if primitive.HasLine && !primitive.NoLine && primitive.RotationDegrees == 0 {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		drawPicturePrimitiveOutline(layer, target, primitive, lineWidth)
	}
	blur := innerShadowBlurPixels(primitive.InnerShadowBlur, size, img.Bounds().Dx())
	offset := innerShadowOffset(primitive.InnerShadowDistance, primitive.InnerShadowDirection, size, img.Bounds().Dx())
	pad := blur + absInt(offset.X) + absInt(offset.Y)
	crop := target.Inset(-pad).Intersect(layer.Bounds())
	if crop.Empty() {
		return softEdgeRendered, false
	}
	if !applyInnerShadow(layer, crop, primitive.InnerShadowColor, blur, offset) {
		return softEdgeRendered, false
	}
	draw.Draw(img, crop, layer, crop.Min, draw.Over)
	return softEdgeRendered, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithFillOverlay(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasFillOverlay = false
	inner.FillOverlay = backgroundPaint{}
	inner.FillOverlayBlend = ""
	softEdgeRendered := false
	if inner.HasBlur && inner.BlurRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithBlur(layer, target, pictureImage, pictureBounds, inner, size)
	} else if inner.HasAlphaOutset && inner.AlphaOutsetRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithAlphaOutset(layer, target, pictureImage, pictureBounds, inner, size)
	} else {
		softEdgeRendered = backend.samplingStage().Draw(pictureSamplingInput{
			Canvas:       layer,
			Target:       target,
			Source:       pictureImage,
			SourceBounds: pictureBounds,
			Primitive:    inner,
			Size:         size,
			OutputWidth:  img.Bounds().Dx(),
		})
	}
	if primitive.HasLine && !primitive.NoLine && primitive.RotationDegrees == 0 {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		drawPicturePrimitiveOutline(layer, target, primitive, lineWidth)
	}
	crop := target.Intersect(layer.Bounds())
	if crop.Empty() {
		return softEdgeRendered, false
	}
	applyFillOverlay(layer, crop, primitive.FillOverlay, primitive.FillOverlayBlend)
	draw.Draw(img, crop, layer, crop.Min, draw.Over)
	return softEdgeRendered, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithAlphaOutset(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	radius := alphaOutsetRadiusPixels(primitive.AlphaOutsetRadius, size, img.Bounds().Dx())
	if radius <= 0 {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasAlphaOutset = false
	inner.AlphaOutsetRadius = 0
	softEdgeRendered := false
	if inner.HasBlur && inner.BlurRadius > 0 {
		softEdgeRendered, _ = backend.drawPicturePrimitiveWithBlur(layer, target, pictureImage, pictureBounds, inner, size)
	} else {
		softEdgeRendered = backend.samplingStage().Draw(pictureSamplingInput{
			Canvas:       layer,
			Target:       target,
			Source:       pictureImage,
			SourceBounds: pictureBounds,
			Primitive:    inner,
			Size:         size,
			OutputWidth:  img.Bounds().Dx(),
		})
	}
	if primitive.HasLine && !primitive.NoLine && primitive.RotationDegrees == 0 {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		drawPicturePrimitiveOutline(layer, target, primitive, lineWidth)
	}
	crop := target.Inset(-radius).Intersect(layer.Bounds())
	if crop.Empty() {
		return softEdgeRendered, false
	}
	if !applyAlphaOutset(layer, crop, radius) {
		return softEdgeRendered, false
	}
	draw.Draw(img, crop, layer, crop.Min, draw.Over)
	return softEdgeRendered, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithBlur(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	radius := picturePrimitiveBlurRadiusPixels(primitive, size, img.Bounds().Dx())
	if radius <= 0 {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasBlur = false
	inner.BlurRadius = 0
	inner.BlurGrow = false
	softEdgeRendered := backend.samplingStage().Draw(pictureSamplingInput{
		Canvas:       layer,
		Target:       target,
		Source:       pictureImage,
		SourceBounds: pictureBounds,
		Primitive:    inner,
		Size:         size,
		OutputWidth:  img.Bounds().Dx(),
	})
	if primitive.HasLine && !primitive.NoLine && primitive.RotationDegrees == 0 {
		lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
		drawPicturePrimitiveOutline(layer, target, primitive, lineWidth)
	}
	crop := target.Inset(-radius).Intersect(layer.Bounds())
	if crop.Empty() {
		return softEdgeRendered, false
	}
	source := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(source, source.Bounds(), layer, crop.Min, draw.Src)
	blurred := gaussianBlurRGBA(source, radius)
	paint := crop
	if !primitive.BlurGrow {
		paint = paint.Intersect(target)
	}
	paint = paint.Intersect(img.Bounds())
	if paint.Empty() {
		return softEdgeRendered, false
	}
	draw.Draw(img, paint, blurred, image.Point{X: paint.Min.X - crop.Min.X, Y: paint.Min.Y - crop.Min.Y}, draw.Over)
	return softEdgeRendered, true
}

func (backend currentPictureBackend) drawPicturePrimitiveWithSourceBlur(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) (bool, bool) {
	if target.Empty() {
		return false, false
	}
	radius := picturePrimitiveSourceBlurRadiusPixels(primitive, size, img.Bounds().Dx())
	if radius <= 0 {
		return false, false
	}
	layer := image.NewRGBA(img.Bounds())
	inner := primitive
	inner.HasSourceBlur = false
	inner.SourceBlurRadius = 0
	inner.SourceBlurGrow = false
	inner.HasLine = false
	inner.NoLine = true
	softEdgeRendered := backend.samplingStage().Draw(pictureSamplingInput{
		Canvas:       layer,
		Target:       target,
		Source:       pictureImage,
		SourceBounds: pictureBounds,
		Primitive:    inner,
		Size:         size,
		OutputWidth:  img.Bounds().Dx(),
	})
	crop := target.Inset(-radius).Intersect(layer.Bounds())
	if crop.Empty() {
		return softEdgeRendered, false
	}
	source := image.NewRGBA(image.Rect(0, 0, crop.Dx(), crop.Dy()))
	draw.Draw(source, source.Bounds(), layer, crop.Min, draw.Src)
	blurred := gaussianBlurRGBA(source, radius)
	paint := crop
	if !primitive.SourceBlurGrow {
		paint = paint.Intersect(target)
	}
	paint = paint.Intersect(img.Bounds())
	if paint.Empty() {
		return softEdgeRendered, false
	}
	draw.Draw(img, paint, blurred, image.Point{X: paint.Min.X - crop.Min.X, Y: paint.Min.Y - crop.Min.Y}, draw.Over)
	drawPicturePrimitivePostEffectOutline(img, target, primitive, size)
	return softEdgeRendered, true
}

func (backend currentPictureBackend) samplingStage() pictureSamplingStage {
	if backend.sampler != nil {
		return backend.sampler
	}
	return currentPictureSamplingStage{}
}

func imageRectFromObjectPixelBounds(bounds ObjectPixelBounds) image.Rectangle {
	return image.Rect(bounds.MinX, bounds.MinY, bounds.MaxX+1, bounds.MaxY+1)
}

func picturePrimitiveLabel(primitive renderPicturePrimitive) string {
	label := strings.TrimSpace(primitive.Name)
	if label == "" {
		label = primitive.ID
	}
	if label == "" {
		label = primitive.ObjectKind
	}
	return label
}

func picturePrimitiveShadowTransformUnsupportedMessages(primitive renderPicturePrimitive) []string {
	var messages []string
	if (primitive.HasShadowScaleX && primitive.ShadowScaleX != 100000) || (primitive.HasShadowScaleY && primitive.ShadowScaleY != 100000) || (primitive.HasShadowSkewX && primitive.ShadowSkewX != 0) || (primitive.HasShadowSkewY && primitive.ShadowSkewY != 0) {
		messages = append(messages, "outer shadow scale/skew transform was not rendered")
	}
	if primitive.HasShadowRotateWithShape && !primitive.ShadowRotateWithShape && primitive.RotationDegrees != 0 {
		messages = append(messages, "outer shadow rotate-with-shape transform was not rendered")
	}
	return messages
}

func picturePrimitiveShape3DUnsupportedMessages(primitive renderPicturePrimitive) []string {
	if !primitive.HasShape3D {
		return nil
	}
	if len(primitive.Shape3DFeatures) == 0 {
		return []string{"3-D shape properties were not rendered"}
	}
	features := append([]string{}, primitive.Shape3DFeatures...)
	sort.Strings(features)
	return []string{fmt.Sprintf("%s were not rendered", strings.Join(features, ", "))}
}

func drawPictureRaster(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize) bool {
	rotation := normalizedRotationDegrees(element.Rotation)
	if !pictureRotatesWithShape(element) {
		rotation = 0
	}
	if rotation == 0 {
		return drawPictureRasterLayer(img, target, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	}
	if target.Empty() {
		return false
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	layerTarget := layer.Bounds()
	softEdgeRendered := drawPictureRasterLayer(layer, layerTarget, pictureImage, pictureBounds, element, size, img.Bounds().Dx())
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		drawPictureOutline(layer, layerTarget, element, lineWidth)
	}
	rotated := rotateRGBA(layer, rotation)
	center := image.Point{X: target.Min.X + target.Dx()/2, Y: target.Min.Y + target.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
	return softEdgeRendered
}

func drawPicturePrimitiveRaster(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize) bool {
	return currentPictureSamplingStage{}.Draw(pictureSamplingInput{
		Canvas:       img,
		Target:       target,
		Source:       pictureImage,
		SourceBounds: pictureBounds,
		Primitive:    primitive,
		Size:         size,
		OutputWidth:  img.Bounds().Dx(),
	})
}

func pictureRotatesWithShape(element slideElement) bool {
	return !element.HasBlipRotWithShape || element.BlipRotWithShape
}

func drawPictureRasterLayer(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, element slideElement, size slideSize, outputWidth int) bool {
	if element.HasSoftEdge && len(element.CustomPath) < 3 {
		scaleImageWithSoftEdge(img, target, pictureImage, pictureBounds, softEdgeRadiusPixels(element, size, outputWidth))
		return true
	}
	if len(element.CustomPath) >= 3 {
		scaleImageWithCustomMask(img, target, pictureImage, pictureBounds, element.CustomPath, element.CustomPathCommands)
		return false
	}
	scaleImage(img, target, pictureImage, pictureBounds)
	return false
}

func drawPicturePrimitiveRasterLayer(img *image.RGBA, target image.Rectangle, pictureImage image.Image, pictureBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize, outputWidth int) bool {
	if primitive.HasSoftEdge && len(primitive.CustomPath) < 3 {
		scaleImageWithSoftEdge(img, target, pictureImage, pictureBounds, picturePrimitiveSoftEdgeRadiusPixels(primitive, size, outputWidth))
		return true
	}
	if len(primitive.CustomPath) >= 3 {
		scaleImageWithCustomMask(img, target, pictureImage, pictureBounds, primitive.CustomPath, primitive.CustomPathCommands)
		return false
	}
	if primitive.BlipFillMode == "tile" {
		tileImage(img, target, pictureImage, pictureBounds, primitive, size, outputWidth)
		return false
	}
	scaleImage(img, target, pictureImage, pictureBounds)
	return false
}

func (currentPictureSamplingStage) Draw(input pictureSamplingInput) bool {
	rotation := input.Primitive.RotationDegrees
	if !input.Primitive.RotatesWithShape {
		rotation = 0
	}
	if rotation == 0 {
		return drawPicturePrimitiveRasterLayer(input.Canvas, input.Target, input.Source, input.SourceBounds, input.Primitive, input.Size, input.OutputWidth)
	}
	if input.Target.Empty() {
		return false
	}
	layer := image.NewRGBA(image.Rect(0, 0, input.Target.Dx(), input.Target.Dy()))
	layerTarget := layer.Bounds()
	softEdgeRendered := drawPicturePrimitiveRasterLayer(layer, layerTarget, input.Source, input.SourceBounds, input.Primitive, input.Size, input.OutputWidth)
	if input.Primitive.HasLine && !input.Primitive.NoLine {
		lineWidth := emuLineWidthToPixels(input.Primitive.LineWidth, input.Size.CX, input.OutputWidth)
		drawPicturePrimitiveOutline(layer, layerTarget, input.Primitive, lineWidth)
	}
	rotated := rotateRGBA(layer, rotation)
	center := image.Point{X: input.Target.Min.X + input.Target.Dx()/2, Y: input.Target.Min.Y + input.Target.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(input.Canvas, dst, rotated)
	return softEdgeRendered
}

func drawPictureOutline(img *image.RGBA, target image.Rectangle, element slideElement, lineWidth int) {
	drawStyledRectOutlineCompound(img, target, element.LineColor, lineWidth, element.LineDash, element.LineAlign, element.LineCap, element.LineJoin, element.LineCompound)
}

func drawPicturePrimitiveOutline(img *image.RGBA, target image.Rectangle, primitive renderPicturePrimitive, lineWidth int) {
	drawStyledRectOutlineCompound(img, target, primitive.LineColor, lineWidth, primitive.LineDash, primitive.LineAlign, primitive.LineCap, primitive.LineJoin, primitive.LineCompound)
}

func drawPicturePrimitivePostEffectOutline(img *image.RGBA, target image.Rectangle, primitive renderPicturePrimitive, size slideSize) {
	if !primitive.HasLine || primitive.NoLine || target.Empty() {
		return
	}
	lineWidth := emuLineWidthToPixels(primitive.LineWidth, size.CX, img.Bounds().Dx())
	rotation := primitive.RotationDegrees
	if !primitive.RotatesWithShape {
		rotation = 0
	}
	if rotation == 0 {
		drawPicturePrimitiveOutline(img, target, primitive, lineWidth)
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	drawPicturePrimitiveOutline(layer, layer.Bounds(), primitive, lineWidth)
	rotated := rotateRGBA(layer, rotation)
	center := image.Point{X: target.Min.X + target.Dx()/2, Y: target.Min.Y + target.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
}

func drawPictureShadow(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) bool {
	if element.ShadowColor.A == 0 {
		return false
	}
	offset := shadowOffset(element, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := shadowBlurPixels(element, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	if len(element.CustomPath) >= 3 {
		drawSoftPolygon(img, shadowBounds, element.CustomPath, element.ShadowColor, blur)
	} else {
		drawSoftRect(img, shadowBounds, element.ShadowColor, blur)
	}
	return true
}

func drawPicturePrimitiveShadow(img *image.RGBA, target image.Rectangle, primitive renderPicturePrimitive, size slideSize) bool {
	if primitive.ShadowColor.A == 0 {
		return false
	}
	offset := picturePrimitiveShadowOffset(primitive, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := picturePrimitiveShadowBlurPixels(primitive, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	if len(primitive.CustomPath) >= 3 {
		drawSoftPolygon(img, shadowBounds, primitive.CustomPath, primitive.ShadowColor, blur)
	} else {
		drawSoftRect(img, shadowBounds, primitive.ShadowColor, blur)
	}
	return true
}

func drawPicturePrimitiveGlow(img *image.RGBA, target image.Rectangle, primitive renderPicturePrimitive, size slideSize) bool {
	if primitive.GlowColor.A == 0 {
		return false
	}
	blur := glowRadiusPixels(primitive.GlowRadius, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(target, blur, img.Bounds()) {
		return false
	}
	if len(primitive.CustomPath) >= 3 {
		drawSoftPolygon(img, target, primitive.CustomPath, primitive.GlowColor, blur)
	} else {
		drawSoftRect(img, target, primitive.GlowColor, blur)
	}
	return true
}

func picturePrimitiveShadowOffset(primitive renderPicturePrimitive, size slideSize, outputWidth int) image.Point {
	distance := scaleEMU(primitive.ShadowDistance, size.CX, outputWidth)
	if distance == 0 && primitive.ShadowDistance > 0 {
		distance = 1
	}
	angle := float64(primitive.ShadowDirection) / 60000 * math.Pi / 180
	return image.Point{
		X: int(math.Round(math.Cos(angle) * float64(distance))),
		Y: int(math.Round(math.Sin(angle) * float64(distance))),
	}
}

func picturePrimitiveShadowBlurPixels(primitive renderPicturePrimitive, size slideSize, outputWidth int) int {
	blur := scaleEMU(primitive.ShadowBlur, size.CX, outputWidth)
	if blur < 0 {
		return 0
	}
	return blur
}

func pictureSourceImage(pkg *pptx.Package, slidePart string, element *slideElement, relationships map[string]pptx.Relationship, fallbackRelationship pptx.Relationship) (image.Image, string, error) {
	fallback, fallbackPart, fallbackErr := fallbackPictureSourceImage(pkg, slidePart, fallbackRelationship)
	if fallbackErr == nil {
		return fallback, fallbackPart, nil
	}
	if element.SVGEmbedID == "" {
		return nil, fallbackPart, fallbackErr
	}
	relationship, ok := relationships[element.SVGEmbedID]
	if !ok || relationship.Type != pptx.ImageRelType || (relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal")) {
		return nil, fallbackPart, fallbackErr
	}
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fallbackErr
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, fallbackErr
	}
	return source, targetPart, fallbackErr
}

func fallbackPictureSourceImage(pkg *pptx.Package, slidePart string, relationship pptx.Relationship) (image.Image, string, error) {
	if relationship.Target == "" {
		return nil, "", fmt.Errorf("missing image relationship")
	}
	if relationship.Type != "" && relationship.Type != pptx.ImageRelType {
		return nil, "", fmt.Errorf("relationship type %q is not an image", relationship.Type)
	}
	if relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal") {
		return nil, relationship.Target, fmt.Errorf("linked image relationship target %q is external and was not fetched", relationship.Target)
	}
	targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
	data, ok := pkg.Parts[targetPart]
	if !ok {
		return nil, targetPart, fmt.Errorf("missing image part")
	}
	source, err := decodeImage(targetPart, pkg.ContentTypes.ForPart(targetPart), data)
	if err != nil {
		return nil, targetPart, err
	}
	return source, targetPart, nil
}

func pictureUnsupported(slidePart string, element *slideElement, message string) model.SkipItem {
	element.UnsupportedNote = message
	return unsupportedItem(slidePart, unsupportedCode, message)
}

func decodeImage(partName string, contentType string, data []byte) (image.Image, error) {
	extension := strings.ToLower(path.Ext(partName))
	switch {
	case contentType == "image/png" || extension == ".png":
		return decodePNGImage(data)
	case contentType == "image/jpeg" || contentType == "image/jpg" || extension == ".jpg" || extension == ".jpeg":
		return decodeJPEGImage(data)
	case contentType == "image/gif" || extension == ".gif":
		return gif.Decode(bytes.NewReader(data))
	case contentType == "image/svg+xml" || extension == ".svg":
		return decodeSVGImage(data)
	default:
		return nil, fmt.Errorf("unsupported image content type %q", contentType)
	}
}

func decodePNGImage(data []byte) (image.Image, error) {
	source, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	profileData, ok := pngICCProfile(data)
	if !ok {
		return source, nil
	}
	profile, ok := parseICCRGBToSRGBProfile(profileData)
	if !ok {
		return source, nil
	}
	return convertICCImageToSRGB(source, profile), nil
}

func decodeJPEGImage(data []byte) (image.Image, error) {
	source, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if profileData, ok := jpegICCProfile(data); ok {
		if bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")) {
			return convertAdobeRGBImageToSRGB(source), nil
		}
		if profile, ok := parseICCRGBToSRGBProfile(profileData); ok {
			return convertICCImageToSRGB(source, profile), nil
		}
	}
	return source, nil
}

func jpegHasAdobeRGBProfile(data []byte) bool {
	profileData, ok := jpegICCProfile(data)
	return ok && (bytes.Contains(profileData, []byte("Adobe RGB (1998)")) || bytes.Contains(profileData, []byte("Adobe RGB")))
}

func jpegICCProfile(data []byte) ([]byte, bool) {
	const markerPrefix = "ICC_PROFILE\x00"
	chunks := map[int][]byte{}
	totalChunks := 0
	for offset := 0; offset+4 <= len(data); {
		if data[offset] != 0xFF {
			offset++
			continue
		}
		for offset < len(data) && data[offset] == 0xFF {
			offset++
		}
		if offset >= len(data) {
			break
		}
		marker := data[offset]
		offset++
		if marker == 0xDA || marker == 0xD9 {
			break
		}
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}
		if offset+2 > len(data) {
			return nil, false
		}
		length := int(data[offset])<<8 | int(data[offset+1])
		offset += 2
		if length < 2 || offset+length-2 > len(data) {
			return nil, false
		}
		segment := data[offset : offset+length-2]
		offset += length - 2
		if marker != 0xE2 || !bytes.HasPrefix(segment, []byte(markerPrefix)) {
			continue
		}
		if len(segment) < len(markerPrefix)+2 {
			return nil, false
		}
		sequenceNumber := int(segment[len(markerPrefix)])
		sequenceTotal := int(segment[len(markerPrefix)+1])
		if sequenceNumber == 0 || sequenceTotal == 0 || sequenceNumber > sequenceTotal {
			return nil, false
		}
		if totalChunks == 0 {
			totalChunks = sequenceTotal
		} else if totalChunks != sequenceTotal {
			return nil, false
		}
		if _, exists := chunks[sequenceNumber]; exists {
			return nil, false
		}
		chunks[sequenceNumber] = segment[len(markerPrefix)+2:]
	}
	if totalChunks == 0 || len(chunks) != totalChunks {
		return nil, false
	}
	var profile []byte
	for index := 1; index <= totalChunks; index++ {
		chunk, ok := chunks[index]
		if !ok {
			return nil, false
		}
		profile = append(profile, chunk...)
	}
	return profile, true
}

func pngICCProfile(data []byte) ([]byte, bool) {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		return nil, false
	}
	offset := 8
	for offset+8 <= len(data) {
		length := int(readUint32BE(data[offset : offset+4]))
		chunkType := string(data[offset+4 : offset+8])
		offset += 8
		if length < 0 || offset+length+4 > len(data) {
			return nil, false
		}
		chunk := data[offset : offset+length]
		offset += length + 4
		if chunkType == "IEND" {
			return nil, false
		}
		if chunkType != "iCCP" {
			continue
		}
		nameEnd := bytes.IndexByte(chunk, 0)
		if nameEnd < 0 || nameEnd+2 > len(chunk) || chunk[nameEnd+1] != 0 {
			return nil, false
		}
		reader, err := zlib.NewReader(bytes.NewReader(chunk[nameEnd+2:]))
		if err != nil {
			return nil, false
		}
		defer reader.Close()
		profile, err := io.ReadAll(reader)
		if err != nil {
			return nil, false
		}
		return profile, true
	}
	return nil, false
}

func parseICCRGBToSRGBProfile(data []byte) (iccRGBToSRGBProfile, bool) {
	if len(data) < 132 || string(data[16:20]) != "RGB " || string(data[20:24]) != "XYZ " {
		return iccRGBToSRGBProfile{}, false
	}
	tagCount := int(readUint32BE(data[128:132]))
	if tagCount < 0 || 132+tagCount*12 > len(data) {
		return iccRGBToSRGBProfile{}, false
	}
	tags := map[string][]byte{}
	for index := 0; index < tagCount; index++ {
		entry := 132 + index*12
		signature := string(data[entry : entry+4])
		offset := int(readUint32BE(data[entry+4 : entry+8]))
		size := int(readUint32BE(data[entry+8 : entry+12]))
		if offset < 0 || size < 0 || offset+size > len(data) {
			continue
		}
		tags[signature] = data[offset : offset+size]
	}
	profile := iccRGBToSRGBProfile{}
	var ok bool
	if profile.rXYZ, ok = parseICCXYZTag(tags["rXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gXYZ, ok = parseICCXYZTag(tags["gXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bXYZ, ok = parseICCXYZTag(tags["bXYZ"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.rTRC, ok = parseICCCurveTag(tags["rTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.gTRC, ok = parseICCCurveTag(tags["gTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	if profile.bTRC, ok = parseICCCurveTag(tags["bTRC"]); !ok {
		return iccRGBToSRGBProfile{}, false
	}
	return profile, true
}

func parseICCXYZTag(data []byte) ([3]float64, bool) {
	if len(data) < 20 || string(data[:4]) != "XYZ " {
		return [3]float64{}, false
	}
	return [3]float64{
		s15Fixed16(data[8:12]),
		s15Fixed16(data[12:16]),
		s15Fixed16(data[16:20]),
	}, true
}

func parseICCCurveTag(data []byte) (iccCurve, bool) {
	if len(data) < 12 || string(data[:4]) != "curv" {
		return iccCurve{}, false
	}
	count := int(readUint32BE(data[8:12]))
	if count == 0 {
		return iccCurve{gamma: 1}, true
	}
	if len(data) < 12+count*2 {
		return iccCurve{}, false
	}
	if count == 1 {
		return iccCurve{gamma: float64(readUint16BE(data[12:14])) / 256}, true
	}
	table := make([]uint16, count)
	for index := range table {
		table[index] = readUint16BE(data[12+index*2 : 14+index*2])
	}
	return iccCurve{table: table}, true
}

func convertICCImageToSRGB(source image.Image, profile iccRGBToSRGBProfile) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.NRGBAModel.Convert(source.At(x, y)).(color.NRGBA)
			r, g, b := profile.iccRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: pixel.A})
		}
	}
	return dst
}

func (profile iccRGBToSRGBProfile) iccRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := profile.rTRC.linearize(r)
	linearG := profile.gTRC.linearize(g)
	linearB := profile.bTRC.linearize(b)

	xD50 := profile.rXYZ[0]*linearR + profile.gXYZ[0]*linearG + profile.bXYZ[0]*linearB
	yD50 := profile.rXYZ[1]*linearR + profile.gXYZ[1]*linearG + profile.bXYZ[1]*linearB
	zD50 := profile.rXYZ[2]*linearR + profile.gXYZ[2]*linearG + profile.bXYZ[2]*linearB

	// ICC matrix profiles encode PCS XYZ relative to D50. Adapt to D65 before
	// applying the sRGB output matrix.
	xD65 := 0.9555766*xD50 - 0.0230393*yD50 + 0.0631636*zD50
	yD65 := -0.0282895*xD50 + 1.0099416*yD50 + 0.0210077*zD50
	zD65 := 0.0122982*xD50 - 0.0204830*yD50 + 1.3299098*zD50

	srgbR := 3.2404542*xD65 - 1.5371385*yD65 - 0.4985314*zD65
	srgbG := -0.9692660*xD65 + 1.8760108*yD65 + 0.0415560*zD65
	srgbB := 0.0556434*xD65 - 0.2040259*yD65 + 1.0572252*zD65
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func (curve iccCurve) linearize(value uint8) float64 {
	encoded := float64(value) / 255
	if len(curve.table) == 0 {
		gamma := curve.gamma
		if gamma == 0 {
			gamma = 1
		}
		return math.Pow(encoded, gamma)
	}
	position := encoded * float64(len(curve.table)-1)
	index := int(math.Floor(position))
	if index >= len(curve.table)-1 {
		return float64(curve.table[len(curve.table)-1]) / 65535
	}
	fraction := position - float64(index)
	a := float64(curve.table[index]) / 65535
	b := float64(curve.table[index+1]) / 65535
	return a + (b-a)*fraction
}

func readUint32BE(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(data[0])<<24 | uint32(data[1])<<16 | uint32(data[2])<<8 | uint32(data[3])
}

func readUint16BE(data []byte) uint16 {
	if len(data) < 2 {
		return 0
	}
	return uint16(data[0])<<8 | uint16(data[1])
}

func s15Fixed16(data []byte) float64 {
	if len(data) < 4 {
		return 0
	}
	value := int32(readUint32BE(data))
	return float64(value) / 65536
}

func convertAdobeRGBImageToSRGB(source image.Image) *image.RGBA {
	bounds := source.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pixel := color.RGBAModel.Convert(source.At(x, y)).(color.RGBA)
			pixel.R, pixel.G, pixel.B = adobeRGBToSRGB(pixel.R, pixel.G, pixel.B)
			dst.SetRGBA(x, y, pixel)
		}
	}
	return dst
}

func adobeRGBToSRGB(r uint8, g uint8, b uint8) (uint8, uint8, uint8) {
	linearR := adobeRGBByteToLinear(r)
	linearG := adobeRGBByteToLinear(g)
	linearB := adobeRGBByteToLinear(b)

	x := 0.5767309*linearR + 0.1855540*linearG + 0.1881852*linearB
	y := 0.2973769*linearR + 0.6273491*linearG + 0.0752741*linearB
	z := 0.0270343*linearR + 0.0706872*linearG + 0.9911085*linearB

	srgbR := 3.2404542*x - 1.5371385*y - 0.4985314*z
	srgbG := -0.9692660*x + 1.8760108*y + 0.0415560*z
	srgbB := 0.0556434*x - 0.2040259*y + 1.0572252*z
	return linearToSRGBByte(srgbR), linearToSRGBByte(srgbG), linearToSRGBByte(srgbB)
}

func adobeRGBByteToLinear(value uint8) float64 {
	if value == 0 {
		return 0
	}
	return math.Pow(float64(value)/255, 2.19921875)
}

func decodeSVGImage(data []byte) (image.Image, error) {
	root, err := parseXMLNode(data)
	if err != nil {
		return nil, err
	}
	if root.Name != "svg" {
		return nil, fmt.Errorf("expected svg root, got %q", root.Name)
	}
	viewBox, err := parseSVGViewBox(root)
	if err != nil {
		return nil, err
	}
	width := svgRasterDimension(viewBox.Width)
	height := svgRasterDimension(viewBox.Height)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := drawSVGNode(img, img.Bounds(), viewBox, root, parseSVGStyleRules(root), svgPaintStyle{}); err != nil {
		return nil, err
	}
	return img, nil
}

func parseSVGViewBox(root *xmlNode) (svgViewBox, error) {
	raw := attrValue(root.Attrs, "viewBox")
	values := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	if len(values) == 4 {
		var parsed [4]float64
		for index, value := range values {
			number, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return svgViewBox{}, fmt.Errorf("invalid svg viewBox %q", raw)
			}
			parsed[index] = number
		}
		if parsed[2] > 0 && parsed[3] > 0 {
			return svgViewBox{MinX: parsed[0], MinY: parsed[1], Width: parsed[2], Height: parsed[3]}, nil
		}
	}
	width, widthOK := svgLengthAttr(root.Attrs, "width")
	height, heightOK := svgLengthAttr(root.Attrs, "height")
	if widthOK && heightOK && width > 0 && height > 0 {
		return svgViewBox{Width: width, Height: height}, nil
	}
	return svgViewBox{}, fmt.Errorf("svg viewBox is missing or invalid")
}

func svgLengthAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := attrValue(attrs, name)
	value = strings.TrimSuffix(strings.TrimSpace(value), "px")
	if value == "" {
		return 0, false
	}
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgRasterDimension(value float64) int {
	dimension := int(math.Round(value))
	if dimension < 1 {
		return 1
	}
	if dimension > 2048 {
		return 2048
	}
	return dimension
}

func drawSVGNode(img *image.RGBA, bounds image.Rectangle, viewBox svgViewBox, node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) error {
	inherited = resolveSVGPaintStyle(node, styles, inherited, true)
	for _, child := range node.Children {
		switch child.Name {
		case "g", "svg":
			if err := drawSVGNode(img, bounds, viewBox, child, styles, inherited); err != nil {
				return err
			}
		case "path":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			paths, err := parseSVGPath(attrValue(child.Attrs, "d"), viewBox)
			if err != nil {
				return err
			}
			for _, points := range paths {
				drawPolygon(img, bounds, points, c)
			}
		case "rect":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			rect, ok := svgRectBounds(child, bounds, viewBox)
			if ok {
				draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Src)
			}
		case "circle", "ellipse":
			c, ok := svgNodeFill(child, styles, inherited)
			if !ok {
				continue
			}
			ellipse, ok := svgEllipseBounds(child, bounds, viewBox)
			if ok {
				drawEllipse(img, ellipse, c)
			}
		}
	}
	return nil
}

func svgNodeFill(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle) (color.RGBA, bool) {
	style := resolveSVGPaintStyle(node, styles, inherited, true)
	if style.NoFill {
		return color.RGBA{}, false
	}
	if !style.HasFill {
		style.Fill = color.RGBA{A: 255}
	}
	if style.HasOpacity {
		opacity := style.FillOpacity
		if opacity < 0 {
			opacity = 0
		}
		if opacity > 1 {
			opacity = 1
		}
		style.Fill.A = uint8(math.Round(float64(style.Fill.A) * opacity))
	}
	return style.Fill, true
}

func resolveSVGPaintStyle(node *xmlNode, styles map[string]svgPaintStyle, inherited svgPaintStyle, includePresentationAttrs bool) svgPaintStyle {
	resolved := inherited
	if includePresentationAttrs {
		mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations("fill:"+attrValue(node.Attrs, "fill")+";fill-opacity:"+attrValue(node.Attrs, "fill-opacity")))
	}
	for _, className := range strings.Fields(attrValue(node.Attrs, "class")) {
		if style, ok := styles[className]; ok {
			mergeSVGPaintStyle(&resolved, style)
		}
	}
	mergeSVGPaintStyle(&resolved, parseSVGPaintDeclarations(attrValue(node.Attrs, "style")))
	return resolved
}

func mergeSVGPaintStyle(base *svgPaintStyle, override svgPaintStyle) {
	if override.HasFill || override.NoFill {
		base.Fill = override.Fill
		base.HasFill = override.HasFill
		base.NoFill = override.NoFill
	}
	if override.HasOpacity {
		base.FillOpacity = override.FillOpacity
		base.HasOpacity = true
	}
}

func parseSVGStyleRules(root *xmlNode) map[string]svgPaintStyle {
	styles := map[string]svgPaintStyle{}
	for _, node := range descendantsByName(root, "style") {
		for _, block := range strings.Split(node.Text, "}") {
			selectorText, declarationText, ok := strings.Cut(block, "{")
			if !ok {
				continue
			}
			style := parseSVGPaintDeclarations(declarationText)
			if !style.HasFill && !style.NoFill && !style.HasOpacity {
				continue
			}
			for _, selector := range strings.Split(selectorText, ",") {
				selector = strings.TrimSpace(selector)
				if !strings.HasPrefix(selector, ".") {
					continue
				}
				className := strings.TrimSpace(strings.TrimPrefix(selector, "."))
				if className != "" {
					styles[className] = style
				}
			}
		}
	}
	return styles
}

func parseSVGPaintDeclarations(raw string) svgPaintStyle {
	var style svgPaintStyle
	for _, declaration := range strings.Split(raw, ";") {
		name, value, ok := strings.Cut(declaration, ":")
		if !ok {
			continue
		}
		name = strings.ToLower(strings.TrimSpace(name))
		value = strings.TrimSpace(value)
		switch name {
		case "fill":
			c, hasFill, noFill := parseSVGFillValue(value)
			style.Fill = c
			style.HasFill = hasFill
			style.NoFill = noFill
		case "fill-opacity":
			if opacity, ok := parseSVGOpacity(value); ok {
				style.FillOpacity = opacity
				style.HasOpacity = true
			}
		}
	}
	return style
}

func parseSVGFillValue(raw string) (color.RGBA, bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return color.RGBA{}, false, false
	}
	if strings.EqualFold(raw, "none") {
		return color.RGBA{}, false, true
	}
	var c color.RGBA
	var ok bool
	switch strings.ToLower(raw) {
	case "black":
		c, ok = color.RGBA{A: 255}, true
	case "white":
		c, ok = color.RGBA{R: 255, G: 255, B: 255, A: 255}, true
	default:
		c, ok = parseHexColor(raw)
	}
	if !ok {
		return color.RGBA{}, false, false
	}
	return c, true, false
}

func parseSVGOpacity(raw string) (float64, bool) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	return value, err == nil
}

func svgRectBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	x, xOK := svgFloatAttr(node.Attrs, "x")
	y, yOK := svgFloatAttr(node.Attrs, "y")
	width, widthOK := svgFloatAttr(node.Attrs, "width")
	height, heightOK := svgFloatAttr(node.Attrs, "height")
	if !xOK {
		x = 0
	}
	if !yOK {
		y = 0
	}
	if !widthOK || !heightOK || width <= 0 || height <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(x, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(x+width, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(y+height, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgEllipseBounds(node *xmlNode, bounds image.Rectangle, viewBox svgViewBox) (image.Rectangle, bool) {
	cx, cxOK := svgFloatAttr(node.Attrs, "cx")
	cy, cyOK := svgFloatAttr(node.Attrs, "cy")
	if !cxOK || !cyOK {
		return image.Rectangle{}, false
	}
	rx, rxOK := svgFloatAttr(node.Attrs, "rx")
	ry, ryOK := svgFloatAttr(node.Attrs, "ry")
	if node.Name == "circle" {
		r, rOK := svgFloatAttr(node.Attrs, "r")
		rx, ry, rxOK, ryOK = r, r, rOK, rOK
	}
	if !rxOK || !ryOK || rx <= 0 || ry <= 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(
		svgCoordToPixel(cx-rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy-ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
		svgCoordToPixel(cx+rx, viewBox.MinX, viewBox.Width, bounds.Min.X, bounds.Dx()),
		svgCoordToPixel(cy+ry, viewBox.MinY, viewBox.Height, bounds.Min.Y, bounds.Dy()),
	).Intersect(bounds), true
}

func svgFloatAttr(attrs []xml.Attr, name string) (float64, bool) {
	value := strings.TrimSpace(attrValue(attrs, name))
	if value == "" {
		return 0, false
	}
	value = strings.TrimSuffix(value, "px")
	number, err := strconv.ParseFloat(value, 64)
	return number, err == nil
}

func svgCoordToPixel(value float64, min float64, span float64, pixelMin int, pixelSpan int) int {
	if span == 0 {
		return pixelMin
	}
	return pixelMin + int(math.Round((value-min)/span*float64(pixelSpan)))
}

func svgPointToPathPoint(x float64, y float64, viewBox svgViewBox) pathPoint {
	return pathPoint{
		X: (x - viewBox.MinX) / viewBox.Width,
		Y: (y - viewBox.MinY) / viewBox.Height,
	}
}

func parseSVGPath(data string, viewBox svgViewBox) ([][]pathPoint, error) {
	tokens, err := tokenizeSVGPath(data)
	if err != nil {
		return nil, err
	}
	var paths [][]pathPoint
	var points []pathPoint
	var currentCommand byte
	var currentX float64
	var currentY float64
	var startX float64
	var startY float64
	index := 0
	for index < len(tokens) {
		if !tokens[index].IsNumber {
			currentCommand = tokens[index].Command
			index++
		} else if currentCommand == 0 {
			return nil, fmt.Errorf("svg path data starts with a number")
		}
		switch currentCommand {
		case 'M', 'm':
			first := true
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path move command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'm' {
					x += currentX
					y += currentY
				}
				if first {
					if len(points) >= 3 {
						paths = append(paths, points)
					}
					points = []pathPoint{svgPointToPathPoint(x, y, viewBox)}
					startX, startY = x, y
					first = false
				} else {
					points = append(points, svgPointToPathPoint(x, y, viewBox))
				}
				currentX, currentY = x, y
			}
		case 'L', 'l':
			for index < len(tokens) && tokens[index].IsNumber {
				x, y, next, ok := readSVGPathPair(tokens, index)
				if !ok {
					return nil, fmt.Errorf("svg path line command has incomplete coordinates")
				}
				index = next
				if currentCommand == 'l' {
					x += currentX
					y += currentY
				}
				points = append(points, svgPointToPathPoint(x, y, viewBox))
				currentX, currentY = x, y
			}
		case 'H', 'h':
			for index < len(tokens) && tokens[index].IsNumber {
				x := tokens[index].Number
				index++
				if currentCommand == 'h' {
					x += currentX
				}
				points = append(points, svgPointToPathPoint(x, currentY, viewBox))
				currentX = x
			}
		case 'V', 'v':
			for index < len(tokens) && tokens[index].IsNumber {
				y := tokens[index].Number
				index++
				if currentCommand == 'v' {
					y += currentY
				}
				points = append(points, svgPointToPathPoint(currentX, y, viewBox))
				currentY = y
			}
		case 'C', 'c':
			for index < len(tokens) && tokens[index].IsNumber {
				values, next, ok := readSVGPathNumbers(tokens, index, 6)
				if !ok {
					return nil, fmt.Errorf("svg path cubic command has incomplete coordinates")
				}
				index = next
				x1, y1, x2, y2, x, y := values[0], values[1], values[2], values[3], values[4], values[5]
				if currentCommand == 'c' {
					x1 += currentX
					y1 += currentY
					x2 += currentX
					y2 += currentY
					x += currentX
					y += currentY
				}
				points = append(points, flattenSVGCubic(currentX, currentY, x1, y1, x2, y2, x, y, viewBox)...)
				currentX, currentY = x, y
			}
		case 'Z', 'z':
			if len(points) >= 3 {
				paths = append(paths, points)
			}
			points = nil
			currentX, currentY = startX, startY
			currentCommand = 0
		default:
			return nil, fmt.Errorf("unsupported svg path command %q", string(currentCommand))
		}
	}
	if len(points) >= 3 {
		paths = append(paths, points)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("svg path has no closed paintable subpaths")
	}
	return paths, nil
}

func tokenizeSVGPath(data string) ([]svgPathToken, error) {
	var tokens []svgPathToken
	for index := 0; index < len(data); {
		ch := data[index]
		switch {
		case isSVGPathSeparator(ch):
			index++
		case isSVGPathCommand(ch):
			tokens = append(tokens, svgPathToken{Command: ch})
			index++
		case isSVGPathNumberStart(ch):
			start := index
			index++
			for index < len(data) && isSVGPathNumberByte(data[index], data[index-1]) {
				index++
			}
			number, err := strconv.ParseFloat(data[start:index], 64)
			if err != nil {
				return nil, fmt.Errorf("invalid svg path number %q", data[start:index])
			}
			tokens = append(tokens, svgPathToken{Number: number, IsNumber: true})
		default:
			return nil, fmt.Errorf("invalid svg path token %q", string(ch))
		}
	}
	return tokens, nil
}

func isSVGPathSeparator(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ','
}

func isSVGPathCommand(ch byte) bool {
	return strings.ContainsRune("MmLlHhVvCcZz", rune(ch))
}

func isSVGPathNumberStart(ch byte) bool {
	return (ch >= '0' && ch <= '9') || ch == '-' || ch == '+' || ch == '.'
}

func isSVGPathNumberByte(ch byte, previous byte) bool {
	if ch >= '0' && ch <= '9' {
		return true
	}
	if ch == '.' {
		return true
	}
	if ch == '-' || ch == '+' {
		return previous == 'e' || previous == 'E'
	}
	return ch == 'e' || ch == 'E'
}

func readSVGPathPair(tokens []svgPathToken, index int) (float64, float64, int, bool) {
	values, next, ok := readSVGPathNumbers(tokens, index, 2)
	if !ok {
		return 0, 0, index, false
	}
	return values[0], values[1], next, true
}

func readSVGPathNumbers(tokens []svgPathToken, index int, count int) ([]float64, int, bool) {
	if index+count > len(tokens) {
		return nil, index, false
	}
	values := make([]float64, 0, count)
	for offset := 0; offset < count; offset++ {
		token := tokens[index+offset]
		if !token.IsNumber {
			return nil, index, false
		}
		values = append(values, token.Number)
	}
	return values, index + count, true
}

func flattenSVGCubic(x0 float64, y0 float64, x1 float64, y1 float64, x2 float64, y2 float64, x3 float64, y3 float64, viewBox svgViewBox) []pathPoint {
	const segments = 12
	points := make([]pathPoint, 0, segments)
	for step := 1; step <= segments; step++ {
		t := float64(step) / segments
		inv := 1 - t
		x := inv*inv*inv*x0 + 3*inv*inv*t*x1 + 3*inv*t*t*x2 + t*t*t*x3
		y := inv*inv*inv*y0 + 3*inv*inv*t*y1 + 3*inv*t*t*y2 + t*t*t*y3
		points = append(points, svgPointToPathPoint(x, y, viewBox))
	}
	return points
}

func scaleEMU(value int64, totalEMU int64, totalPixels int) int {
	if totalEMU == 0 {
		return 0
	}
	return int(math.Round(float64(value) / float64(totalEMU) * float64(totalPixels)))
}

func scaleEMUFloat(value int64, totalEMU int64, totalPixels int) float64 {
	if totalEMU == 0 {
		return 0
	}
	return float64(value) / float64(totalEMU) * float64(totalPixels)
}

func sourceCropRect(bounds image.Rectangle, element slideElement) image.Rectangle {
	if !element.HasCrop {
		return bounds
	}
	width := bounds.Dx()
	height := bounds.Dy()
	left := bounds.Min.X + cropPixels(width, element.CropLeft)
	top := bounds.Min.Y + cropPixels(height, element.CropTop)
	right := bounds.Max.X - cropPixels(width, element.CropRight)
	bottom := bounds.Max.Y - cropPixels(height, element.CropBottom)
	cropped := image.Rect(left, top, right, bottom)
	if cropped.Empty() || cropped.Intersect(bounds).Empty() {
		return bounds
	}
	return cropped
}

func sourceCropRectForPrimitive(bounds image.Rectangle, primitive renderPicturePrimitive) image.Rectangle {
	if primitive.Crop.Left == 0 && primitive.Crop.Top == 0 && primitive.Crop.Right == 0 && primitive.Crop.Bottom == 0 {
		return bounds
	}
	width := bounds.Dx()
	height := bounds.Dy()
	left := bounds.Min.X + cropPixels(width, primitive.Crop.Left)
	top := bounds.Min.Y + cropPixels(height, primitive.Crop.Top)
	right := bounds.Max.X - cropPixels(width, primitive.Crop.Right)
	bottom := bounds.Max.Y - cropPixels(height, primitive.Crop.Bottom)
	cropped := image.Rect(left, top, right, bottom)
	if cropped.Empty() || cropped.Intersect(bounds).Empty() {
		return bounds
	}
	return cropped
}

func cropPixels(total int, percentage int64) int {
	if percentage == 0 || total == 0 {
		return 0
	}
	return int(math.Round(float64(total) * float64(percentage) / 100000))
}

func pictureSourceForElement(src image.Image, element slideElement) (image.Image, image.Rectangle) {
	srcBounds := sourceCropRect(src.Bounds(), element)
	if !element.FlipH && !element.FlipV && !shouldApplyImageEffects(element) {
		return src, srcBounds
	}
	return transformedPictureImage(src, srcBounds, element), image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}

func pictureSourceForPrimitive(src image.Image, primitive renderPicturePrimitive) (image.Image, image.Rectangle) {
	srcBounds := sourceCropRectForPrimitive(src.Bounds(), primitive)
	if !primitive.FlipH && !primitive.FlipV && !shouldApplyPrimitiveImageEffects(primitive) {
		return src, srcBounds
	}
	return transformedPicturePrimitiveImage(src, srcBounds, primitive), image.Rect(0, 0, srcBounds.Dx(), srcBounds.Dy())
}

func shouldApplyImageAlphaModFix(element slideElement) bool {
	return element.HasImageAlphaModFix && element.ImageAlphaModFixPct != 100000
}

func shouldApplyPrimitiveAlphaModFix(primitive renderPicturePrimitive) bool {
	return primitive.HasAlphaModFix && primitive.AlphaModFixPct != 100000
}

func shouldApplyImageAlphaModulate(element slideElement) bool {
	return element.HasImageAlphaModulate && element.ImageAlphaModulatePct != 100000
}

func shouldApplyPrimitiveAlphaModulate(primitive renderPicturePrimitive) bool {
	return primitive.HasAlphaModulate && primitive.AlphaModulatePct != 100000
}

func shouldApplyImageEffects(element slideElement) bool {
	return shouldApplyImageAlphaModFix(element) ||
		shouldApplyImageAlphaModulate(element) ||
		element.HasImageAlphaBiLevel ||
		element.HasImageAlphaCeiling ||
		element.HasImageAlphaFloor ||
		element.HasImageAlphaInverse ||
		element.HasImageAlphaReplace ||
		element.HasImageBiLevel ||
		element.HasImageGrayscale ||
		element.HasImageLuminance ||
		element.HasImageHSL ||
		element.HasImageTint ||
		element.HasImageFillOverlay ||
		element.HasImageColorChange ||
		element.HasImageColorReplace ||
		element.HasImageDuotone
}

func shouldApplyPrimitiveImageEffects(primitive renderPicturePrimitive) bool {
	return shouldApplyPrimitiveAlphaModFix(primitive) ||
		shouldApplyPrimitiveAlphaModulate(primitive) ||
		primitive.HasAlphaBiLevel ||
		primitive.HasAlphaCeiling ||
		primitive.HasAlphaFloor ||
		primitive.HasAlphaInverse ||
		primitive.HasAlphaReplace ||
		primitive.HasBiLevel ||
		primitive.HasGrayscale ||
		primitive.HasLuminance ||
		primitive.HasHSL ||
		primitive.HasTint ||
		primitive.HasSourceFillOverlay ||
		primitive.HasColorChange ||
		primitive.HasColorReplace ||
		primitive.HasDuotone
}

func transformedPictureImage(src image.Image, srcBounds image.Rectangle, element slideElement) *image.RGBA {
	width := srcBounds.Dx()
	height := srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y
		if element.FlipV {
			srcY = srcBounds.Max.Y - 1 - y
		}
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x
			if element.FlipH {
				srcX = srcBounds.Max.X - 1 - x
			}
			pixel := color.RGBAModel.Convert(src.At(srcX, srcY)).(color.RGBA)
			pixel = applyImageEffects(pixel, element)
			dst.SetRGBA(x, y, pixel)
		}
	}
	if element.HasImageFillOverlay {
		applyFillOverlay(dst, dst.Bounds(), element.ImageFillOverlay, element.ImageFillOverlayBlend)
	}
	return dst
}

func transformedPicturePrimitiveImage(src image.Image, srcBounds image.Rectangle, primitive renderPicturePrimitive) *image.RGBA {
	width := srcBounds.Dx()
	height := srcBounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		srcY := srcBounds.Min.Y + y
		if primitive.FlipV {
			srcY = srcBounds.Max.Y - 1 - y
		}
		for x := 0; x < width; x++ {
			srcX := srcBounds.Min.X + x
			if primitive.FlipH {
				srcX = srcBounds.Max.X - 1 - x
			}
			pixel := color.RGBAModel.Convert(src.At(srcX, srcY)).(color.RGBA)
			pixel = applyPrimitiveImageEffects(pixel, primitive)
			dst.SetRGBA(x, y, pixel)
		}
	}
	if primitive.HasSourceFillOverlay {
		applyFillOverlay(dst, dst.Bounds(), primitive.SourceFillOverlay, primitive.SourceFillOverlayBlend)
	}
	return dst
}

func applyImageAlphaModFix(c color.RGBA, element slideElement) color.RGBA {
	if shouldApplyImageAlphaModFix(element) {
		c.A = scaleColorChannel(c.A, element.ImageAlphaModFixPct)
	}
	return c
}

func applyPrimitiveAlphaModFix(c color.RGBA, primitive renderPicturePrimitive) color.RGBA {
	if shouldApplyPrimitiveAlphaModFix(primitive) {
		c.A = scaleColorChannel(c.A, primitive.AlphaModFixPct)
	}
	return c
}

func applyImageAlphaModulate(c color.RGBA, element slideElement) color.RGBA {
	if shouldApplyImageAlphaModulate(element) {
		c.A = scaleColorChannel(c.A, element.ImageAlphaModulatePct)
	}
	return c
}

func applyPrimitiveAlphaModulate(c color.RGBA, primitive renderPicturePrimitive) color.RGBA {
	if shouldApplyPrimitiveAlphaModulate(primitive) {
		c.A = scaleColorChannel(c.A, primitive.AlphaModulatePct)
	}
	return c
}

func applyImageEffects(c color.RGBA, element slideElement) color.RGBA {
	c = applyImageAlphaModFix(c, element)
	c = applyImageAlphaModulate(c, element)
	if element.HasImageAlphaBiLevel {
		c.A = alphaBiLevel(c.A, element.ImageAlphaBiLevelThreshold)
	}
	if element.HasImageAlphaCeiling {
		c.A = alphaCeiling(c.A)
	}
	if element.HasImageAlphaFloor {
		c.A = alphaFloor(c.A)
	}
	if element.HasImageAlphaInverse {
		c.A = 255 - c.A
	}
	if element.HasImageAlphaReplace {
		c.A = colorChannelFromPercent(element.ImageAlphaReplacePct)
	}
	if element.HasImageColorChange && colorMatchesBlipChange(c, element.ImageColorChangeFrom, element.ImageColorChangeUseAlpha) {
		c = replacementColorWithAlpha(element.ImageColorChangeTo, c.A, element.ImageColorChangeUseAlpha)
	}
	if element.HasImageColorReplace {
		c = replacementColorWithAlpha(element.ImageColorReplace, c.A, false)
	}
	if element.HasImageGrayscale {
		c = applyGrayscale(c)
	}
	if element.HasImageBiLevel {
		c = applyBiLevel(c, element.ImageBiLevelThreshold)
	}
	if element.HasImageLuminance {
		c = applyImageLuminance(c, element.ImageLuminanceBright, element.ImageLuminanceContrast)
	}
	if element.HasImageHSL {
		c = applyImageHSL(c, element.ImageHSLHue, element.ImageHSLSaturation, element.ImageHSLLuminance)
	}
	if element.HasImageTint {
		c = applyImageTint(c, element.ImageTintHue, element.ImageTintAmount)
	}
	if element.HasImageDuotone {
		c = applyDuotone(c, element.ImageDuotoneDark, element.ImageDuotoneLight)
	}
	return c
}

func applyPrimitiveImageEffects(c color.RGBA, primitive renderPicturePrimitive) color.RGBA {
	c = applyPrimitiveAlphaModFix(c, primitive)
	c = applyPrimitiveAlphaModulate(c, primitive)
	if primitive.HasAlphaBiLevel {
		c.A = alphaBiLevel(c.A, primitive.AlphaBiLevelThreshold)
	}
	if primitive.HasAlphaCeiling {
		c.A = alphaCeiling(c.A)
	}
	if primitive.HasAlphaFloor {
		c.A = alphaFloor(c.A)
	}
	if primitive.HasAlphaInverse {
		c.A = 255 - c.A
	}
	if primitive.HasAlphaReplace {
		c.A = colorChannelFromPercent(primitive.AlphaReplacePct)
	}
	if primitive.HasColorChange && colorMatchesBlipChange(c, primitive.ColorChangeFrom, primitive.ColorChangeUseAlpha) {
		c = replacementColorWithAlpha(primitive.ColorChangeTo, c.A, primitive.ColorChangeUseAlpha)
	}
	if primitive.HasColorReplace {
		c = replacementColorWithAlpha(primitive.ColorReplace, c.A, false)
	}
	if primitive.HasGrayscale {
		c = applyGrayscale(c)
	}
	if primitive.HasBiLevel {
		c = applyBiLevel(c, primitive.BiLevelThreshold)
	}
	if primitive.HasLuminance {
		c = applyImageLuminance(c, primitive.LuminanceBright, primitive.LuminanceContrast)
	}
	if primitive.HasHSL {
		c = applyImageHSL(c, primitive.HSLHue, primitive.HSLSaturation, primitive.HSLLuminance)
	}
	if primitive.HasTint {
		c = applyImageTint(c, primitive.TintHue, primitive.TintAmount)
	}
	if primitive.HasDuotone {
		c = applyDuotone(c, primitive.DuotoneDark, primitive.DuotoneLight)
	}
	return c
}

func alphaBiLevel(alpha uint8, threshold int64) uint8 {
	if int64(alpha)*100000/255 >= threshold {
		return 255
	}
	return 0
}

func alphaCeiling(alpha uint8) uint8 {
	if alpha > 0 {
		return 255
	}
	return 0
}

func alphaFloor(alpha uint8) uint8 {
	if alpha < 255 {
		return 0
	}
	return 255
}

func colorMatchesBlipChange(c color.RGBA, from color.RGBA, useAlpha bool) bool {
	if c.R != from.R || c.G != from.G || c.B != from.B {
		return false
	}
	return !useAlpha || c.A == from.A
}

func replacementColorWithAlpha(replacement color.RGBA, originalAlpha uint8, useReplacementAlpha bool) color.RGBA {
	if !useReplacementAlpha {
		replacement.A = originalAlpha
	}
	return replacement
}

func applyBiLevel(c color.RGBA, threshold int64) color.RGBA {
	luma := int64(math.Round(0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)))
	if luma*100000/255 >= threshold {
		c.R, c.G, c.B = 255, 255, 255
	} else {
		c.R, c.G, c.B = 0, 0, 0
	}
	return c
}

func applyImageLuminance(c color.RGBA, bright int64, contrast int64) color.RGBA {
	c.R = applyImageLuminanceChannel(c.R, bright, contrast)
	c.G = applyImageLuminanceChannel(c.G, bright, contrast)
	c.B = applyImageLuminanceChannel(c.B, bright, contrast)
	return c
}

func applyImageLuminanceChannel(channel uint8, bright int64, contrast int64) uint8 {
	value := (float64(channel)-127.5)*(1+float64(contrast)/100000) + 127.5 + 255*float64(bright)/100000
	return clampColor(int64(math.Round(value)))
}

func applyImageHSL(c color.RGBA, hue int64, saturation int64, luminance int64) color.RGBA {
	c = applyHueOffset(c, hue)
	c = applySaturationOffset(c, saturation)
	c = applyLuminanceOffset(c, luminance)
	return c
}

func applyLuminanceOffset(c color.RGBA, value int64) color.RGBA {
	if value == 0 {
		return c
	}
	h, s, l := rgbToHSL(c)
	l += float64(value) / 100000
	l = clampFloat(l, 0, 1)
	c.R, c.G, c.B = hslToRGB(h, s, l)
	return c
}

func applyImageTint(c color.RGBA, hue int64, amount int64) color.RGBA {
	if amount == 0 {
		return c
	}
	currentHue, s, l := rgbToHSL(c)
	targetHue := math.Mod(float64(hue)/60000, 360)
	if targetHue < 0 {
		targetHue += 360
	}
	delta := targetHue - currentHue
	if delta > 180 {
		delta -= 360
	} else if delta < -180 {
		delta += 360
	}
	nextHue := math.Mod(currentHue+delta*float64(amount)/100000, 360)
	if nextHue < 0 {
		nextHue += 360
	}
	c.R, c.G, c.B = hslToRGB(nextHue, s, l)
	return c
}

func applyDuotone(c color.RGBA, dark color.RGBA, light color.RGBA) color.RGBA {
	luma := (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
	return color.RGBA{
		R: clampColor(int64(math.Round(float64(dark.R)*(1-luma) + float64(light.R)*luma))),
		G: clampColor(int64(math.Round(float64(dark.G)*(1-luma) + float64(light.G)*luma))),
		B: clampColor(int64(math.Round(float64(dark.B)*(1-luma) + float64(light.B)*luma))),
		A: c.A,
	}
}

func scaleImage(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	pictureScaler(src, srcBounds).Scale(dst, target, src, srcBounds, xdraw.Over, nil)
}

func tileImage(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, primitive renderPicturePrimitive, size slideSize, outputWidth int) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() || srcBounds.Empty() {
		return
	}
	tile := scaledTileImage(src, srcBounds, primitive)
	if tile.Bounds().Empty() {
		return
	}
	offset := image.Point{
		X: scaleEMU(primitive.BlipTileOffsetX, size.CX, outputWidth),
		Y: scaleEMU(primitive.BlipTileOffsetY, size.CX, outputWidth),
	}
	start := tileStartPoint(target, tile.Bounds().Dx(), tile.Bounds().Dy(), offset, primitive.BlipTileAlignment)
	tileIndexY := 0
	for y := start.Y; y < target.Max.Y; y += tile.Bounds().Dy() {
		if y+tile.Bounds().Dy() <= target.Min.Y {
			tileIndexY++
			continue
		}
		tileIndexX := 0
		for x := start.X; x < target.Max.X; x += tile.Bounds().Dx() {
			if x+tile.Bounds().Dx() <= target.Min.X {
				tileIndexX++
				continue
			}
			current := tile
			flipH := (primitive.BlipTileFlip == "x" || primitive.BlipTileFlip == "xy") && tileIndexX%2 == 1
			flipV := (primitive.BlipTileFlip == "y" || primitive.BlipTileFlip == "xy") && tileIndexY%2 == 1
			if flipH || flipV {
				current = flippedTileImage(tile, flipH, flipV)
			}
			draw.Draw(dst, image.Rect(x, y, x+current.Bounds().Dx(), y+current.Bounds().Dy()).Intersect(target), current, image.Point{}, draw.Over)
			tileIndexX++
		}
		tileIndexY++
	}
}

func scaledTileImage(src image.Image, srcBounds image.Rectangle, primitive renderPicturePrimitive) *image.RGBA {
	scaleX := primitive.BlipTileScaleX
	if scaleX == 0 {
		scaleX = 100000
	}
	scaleY := primitive.BlipTileScaleY
	if scaleY == 0 {
		scaleY = 100000
	}
	width := int(math.Round(float64(srcBounds.Dx()) * float64(scaleX) / 100000))
	height := int(math.Round(float64(srcBounds.Dy()) * float64(scaleY) / 100000))
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	tile := image.NewRGBA(image.Rect(0, 0, width, height))
	scaleImage(tile, tile.Bounds(), src, srcBounds)
	return tile
}

func tileStartPoint(target image.Rectangle, tileWidth int, tileHeight int, offset image.Point, alignment string) image.Point {
	start := target.Min.Add(offset)
	switch alignment {
	case "t", "ctr", "b":
		start.X = target.Min.X + (target.Dx()-tileWidth)/2 + offset.X
	case "tr", "r", "br":
		start.X = target.Max.X - tileWidth + offset.X
	}
	switch alignment {
	case "l", "ctr", "r":
		start.Y = target.Min.Y + (target.Dy()-tileHeight)/2 + offset.Y
	case "bl", "b", "br":
		start.Y = target.Max.Y - tileHeight + offset.Y
	}
	for start.X > target.Min.X {
		start.X -= tileWidth
	}
	for start.Y > target.Min.Y {
		start.Y -= tileHeight
	}
	return start
}

func flippedTileImage(src *image.RGBA, flipH bool, flipV bool) *image.RGBA {
	bounds := src.Bounds()
	dst := image.NewRGBA(bounds)
	for y := 0; y < bounds.Dy(); y++ {
		srcY := y
		if flipV {
			srcY = bounds.Dy() - 1 - y
		}
		for x := 0; x < bounds.Dx(); x++ {
			srcX := x
			if flipH {
				srcX = bounds.Dx() - 1 - x
			}
			dst.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, src.RGBAAt(bounds.Min.X+srcX, bounds.Min.Y+srcY))
		}
	}
	return dst
}

func pictureScaler(src image.Image, srcBounds image.Rectangle) xdraw.Scaler {
	if _, ok := src.(*image.YCbCr); ok && srcBounds.In(src.Bounds()) {
		return xdraw.CatmullRom
	}
	return xdraw.ApproxBiLinear
}

func scaleImageWithSoftEdge(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, radius int) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() {
		return
	}
	if srcBounds.Empty() {
		return
	}
	if radius <= 0 {
		scaleImage(dst, target, src, srcBounds)
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	applySoftEdgeAlpha(layer, radius)
	for y := 0; y < layer.Bounds().Dy(); y++ {
		for x := 0; x < layer.Bounds().Dx(); x++ {
			blendPixel(dst, target.Min.X+x, target.Min.Y+y, layer.RGBAAt(x, y))
		}
	}
}

func applySoftEdgeAlpha(img *image.RGBA, radius int) {
	bounds := img.Bounds()
	if radius <= 0 || bounds.Empty() {
		return
	}
	maxRadius := min(radius, min(bounds.Dx(), bounds.Dy())/2)
	if maxRadius <= 0 {
		return
	}
	padding := maxRadius * 3
	maskWidth := bounds.Dx() + padding*2
	maskHeight := bounds.Dy() + padding*2
	mask := make([]uint8, maskWidth*maskHeight)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			mask[(y+padding)*maskWidth+x+padding] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y).A
		}
	}
	blurred := gaussianBlurAlpha(mask, maskWidth, maskHeight, maxRadius)
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			pixel := img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
			pixel.A = blurred[(y+padding)*maskWidth+x+padding]
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, pixel)
		}
	}
}

func applyAlphaOutset(img *image.RGBA, bounds image.Rectangle, radius int) bool {
	bounds = bounds.Intersect(img.Bounds())
	if radius <= 0 || bounds.Empty() {
		return false
	}
	width := bounds.Dx()
	height := bounds.Dy()
	source := make([]color.RGBA, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			source[y*width+x] = img.RGBAAt(bounds.Min.X+x, bounds.Min.Y+y)
		}
	}
	painted := false
	radiusSquared := radius * radius
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			index := y*width + x
			current := source[index]
			best := current
			for dy := -radius; dy <= radius; dy++ {
				ny := y + dy
				if ny < 0 || ny >= height {
					continue
				}
				for dx := -radius; dx <= radius; dx++ {
					if dx*dx+dy*dy > radiusSquared {
						continue
					}
					nx := x + dx
					if nx < 0 || nx >= width {
						continue
					}
					candidate := source[ny*width+nx]
					if candidate.A > best.A {
						best = candidate
					}
				}
			}
			if best.A <= current.A {
				continue
			}
			img.SetRGBA(bounds.Min.X+x, bounds.Min.Y+y, best)
			painted = true
		}
	}
	return painted
}

func softEdgeRadiusPixels(element slideElement, size slideSize, outputWidth int) int {
	radius := scaleEMU(element.SoftEdgeRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func picturePrimitiveSoftEdgeRadiusPixels(primitive renderPicturePrimitive, size slideSize, outputWidth int) int {
	radius := scaleEMU(primitive.SoftEdgeRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func picturePrimitiveBlurRadiusPixels(primitive renderPicturePrimitive, size slideSize, outputWidth int) int {
	radius := scaleEMU(primitive.BlurRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func picturePrimitiveSourceBlurRadiusPixels(primitive renderPicturePrimitive, size slideSize, outputWidth int) int {
	radius := scaleEMU(primitive.SourceBlurRadius, size.CX, outputWidth)
	if radius < 0 {
		return 0
	}
	return radius
}

func scaleImageWithCustomMask(dst *image.RGBA, target image.Rectangle, src image.Image, srcBounds image.Rectangle, points []pathPoint, commands []pathCommand) {
	target = target.Intersect(dst.Bounds())
	if target.Empty() || len(points) < 3 {
		return
	}
	if srcBounds.Empty() {
		return
	}
	layer := image.NewRGBA(image.Rect(0, 0, target.Dx(), target.Dy()))
	pictureScaler(src, srcBounds).Scale(layer, layer.Bounds(), src, srcBounds, xdraw.Over, nil)
	mask := rasterizePathMaskWithCommands(layer.Bounds(), points, commands)
	draw.DrawMask(dst, target, layer, image.Point{}, mask, image.Point{}, draw.Over)
}

func rasterizePathMask(bounds image.Rectangle, points []pathPoint) *image.Alpha {
	return rasterizePathMaskWithCommands(bounds, points, nil)
}

func rasterizePathMaskWithCommands(bounds image.Rectangle, points []pathPoint, commands []pathCommand) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if bounds.Empty() || len(points) < 3 {
		return mask
	}
	rasterizer := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	if len(commands) > 0 {
		for _, command := range commands {
			switch command.Kind {
			case "moveTo":
				if len(command.Points) == 1 {
					rasterizer.MoveTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "lnTo":
				if len(command.Points) == 1 {
					rasterizer.LineTo(maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds))
				}
			case "cubicBezTo":
				if len(command.Points) == 3 {
					rasterizer.CubeTo(
						maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds),
						maskPathX(command.Points[1], bounds), maskPathY(command.Points[1], bounds),
						maskPathX(command.Points[2], bounds), maskPathY(command.Points[2], bounds),
					)
				}
			case "quadBezTo":
				if len(command.Points) == 2 {
					rasterizer.QuadTo(
						maskPathX(command.Points[0], bounds), maskPathY(command.Points[0], bounds),
						maskPathX(command.Points[1], bounds), maskPathY(command.Points[1], bounds),
					)
				}
			case "arcTo":
				for _, point := range command.Points {
					rasterizer.LineTo(maskPathX(point, bounds), maskPathY(point, bounds))
				}
			case "close":
				rasterizer.ClosePath()
			}
		}
		rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
		return mask
	}
	for index, point := range points {
		x := maskPathX(point, bounds)
		y := maskPathY(point, bounds)
		if index == 0 {
			rasterizer.MoveTo(x, y)
		} else {
			rasterizer.LineTo(x, y)
		}
	}
	rasterizer.ClosePath()
	rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	return mask
}

func maskPathX(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.X * float64(bounds.Dx()))
}

func maskPathY(point pathPoint, bounds image.Rectangle) float32 {
	return float32(point.Y * float64(bounds.Dy()))
}
