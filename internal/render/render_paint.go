package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

func renderElements(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, elements []slideElement, tableStyles tableStyleSet) []model.SkipItem {
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return []model.SkipItem{{
			Code:    unsupportedCode,
			Message: fmt.Sprintf("slide relationships could not be parsed for rendering: %v", err),
			Part:    pptx.RelationshipsPartFor(slidePart),
		}}
	}
	relationshipByID := make(map[string]pptx.Relationship, len(relationships))
	for _, relationship := range relationships {
		relationshipByID[relationship.ID] = relationship
	}

	var unsupported []model.SkipItem
	for index := range elements {
		var items []model.SkipItem
		switch elements[index].Kind {
		case "pic":
			items = renderPicture(pkg, slidePart, size, img, &elements[index], relationshipByID)
		case "sp", "cxnSp":
			if elements[index].EmbedID != "" {
				items = append(items, renderPicture(pkg, slidePart, size, img, &elements[index], relationshipByID)...)
			}
			items = append(items, renderShape(slidePart, size, img, &elements[index])...)
		case "graphicFrame":
			items = renderGraphicFrame(pkg, slidePart, size, img, &elements[index], relationshipByID, tableStyles)
		}
		unsupported = append(unsupported, items...)
	}
	return unsupported
}

func renderGraphicFrame(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship, tableStyles tableStyleSet) []model.SkipItem {
	if element.DiagramDataID != "" {
		return renderDiagramGraphicFrame(pkg, slidePart, size, img, element, relationships)
	}
	if element.HasTable {
		return renderTableGraphicFrame(slidePart, size, img, element, tableStyles)
	}
	if element.Text == "" || !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return nil
	}
	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	).Intersect(img.Bounds())
	if target.Empty() {
		return nil
	}
	if err := drawShapeTextWithDPI(img, textBounds(target, *element, size, img.Bounds()), *element, renderDPIForCanvas(size, img.Bounds())); err != nil {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q text could not be rendered: %v", elementLabel(*element), err))}
	}
	element.Rendered = true
	return []model.SkipItem{unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q text was rendered with simplified layout", elementLabel(*element)))}
}

func renderDiagramGraphicFrame(pkg *pptx.Package, slidePart string, size slideSize, img *image.RGBA, element *slideElement, relationships map[string]pptx.Relationship) []model.SkipItem {
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 {
		return nil
	}
	drawingPart, ok, err := diagramDrawingPart(pkg, slidePart, element.DiagramDataID, relationships)
	if err != nil {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q diagram could not be resolved: %v", elementLabel(*element), err))}
	}
	if !ok {
		return nil
	}
	diagramElements := diagramDrawingElements(pkg, slidePart, drawingPart)
	diagramElements = fitDiagramElementsToFrame(diagramElements, *element)
	var unsupported []model.SkipItem
	renderedSupportedElement := false
	for index := range diagramElements {
		if diagramElements[index].Kind != "sp" && diagramElements[index].Kind != "cxnSp" {
			if diagramElements[index].Kind != "" {
				unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q diagram contains %s content that was not rendered", elementLabel(*element), objectKindLabel(diagramElements[index].Kind))))
			}
			continue
		}
		unsupported = append(unsupported, renderShape(slidePart, size, img, &diagramElements[index])...)
		if diagramElements[index].Rendered {
			renderedSupportedElement = true
		}
	}
	element.Rendered = renderedSupportedElement
	return unsupported
}

func diagramDrawingElements(pkg *pptx.Package, slidePart, drawingPart string) []slideElement {
	colors := themeColorsForPart(pkg, slidePart, packageThemeColors(pkg))
	fonts := themeFontsForPart(pkg, slidePart, packageThemeFonts(pkg))
	effectStyles := themeEffectStylesForPart(pkg, slidePart)
	fillStyles := themeFillStylesForPart(pkg, slidePart)
	lineStyles := themeLineStylesForPart(pkg, slidePart)
	elements := collectSlideElementsWithThemeEffectsAndFills(pkg.Parts[drawingPart], colors, effectStyles, fillStyles, lineStyles)
	return applyThemeFontFamilies(elements, fonts)
}

func diagramDrawingPart(pkg *pptx.Package, slidePart string, diagramDataID string, relationships map[string]pptx.Relationship) (string, bool, error) {
	dataRel, ok := relationships[diagramDataID]
	if !ok || dataRel.Type != diagramDataRelType || (dataRel.TargetMode != "" && !strings.EqualFold(dataRel.TargetMode, "Internal")) {
		return "", false, nil
	}
	dataPart := pptx.ResolveTargetPart(slidePart, dataRel.Target)
	data, ok := pkg.Parts[dataPart]
	if !ok {
		return "", false, fmt.Errorf("diagram data part %s is missing", dataPart)
	}
	root, err := parseXMLNode(data)
	if err != nil {
		return "", false, fmt.Errorf("parse diagram data %s: %w", dataPart, err)
	}
	ext := firstDescendant(root, "dataModelExt")
	if ext == nil {
		return "", false, nil
	}
	drawingID := attrValue(ext.Attrs, "relId")
	if drawingID == "" {
		return "", false, nil
	}
	drawingRel, ok := relationships[drawingID]
	if !ok || drawingRel.Type != diagramDrawingRelType || (drawingRel.TargetMode != "" && !strings.EqualFold(drawingRel.TargetMode, "Internal")) {
		return "", false, nil
	}
	drawingPart := pptx.ResolveTargetPart(slidePart, drawingRel.Target)
	if _, ok := pkg.Parts[drawingPart]; !ok {
		return "", false, fmt.Errorf("diagram drawing part %s is missing", drawingPart)
	}
	return drawingPart, true, nil
}

func fitDiagramElementsToFrame(elements []slideElement, frame slideElement) []slideElement {
	maxX, maxY := diagramElementExtents(elements)
	if maxX <= 0 || maxY <= 0 {
		return elements
	}
	scaleX := int64(1)
	scaleY := int64(1)
	denomX := int64(1)
	denomY := int64(1)
	if maxX > frame.ExtCX {
		scaleX = frame.ExtCX
		denomX = maxX
	}
	if maxY > frame.ExtCY {
		scaleY = frame.ExtCY
		denomY = maxY
	}
	for index := range elements {
		if !elements[index].HasTransform {
			continue
		}
		elements[index].OffX = frame.OffX + elements[index].OffX*scaleX/denomX
		elements[index].OffY = frame.OffY + elements[index].OffY*scaleY/denomY
		elements[index].ExtCX = elements[index].ExtCX * scaleX / denomX
		elements[index].ExtCY = elements[index].ExtCY * scaleY / denomY
		if elements[index].HasTextTransform {
			elements[index].TextOffX = frame.OffX + elements[index].TextOffX*scaleX/denomX
			elements[index].TextOffY = frame.OffY + elements[index].TextOffY*scaleY/denomY
			elements[index].TextExtCX = elements[index].TextExtCX * scaleX / denomX
			elements[index].TextExtCY = elements[index].TextExtCY * scaleY / denomY
		}
	}
	return elements
}

func diagramElementExtents(elements []slideElement) (int64, int64) {
	var maxX int64
	var maxY int64
	for _, element := range elements {
		if !element.HasTransform {
			continue
		}
		if right := element.OffX + element.ExtCX; right > maxX {
			maxX = right
		}
		if bottom := element.OffY + element.ExtCY; bottom > maxY {
			maxY = bottom
		}
		if element.HasTextTransform {
			if right := element.TextOffX + element.TextExtCX; right > maxX {
				maxX = right
			}
			if bottom := element.TextOffY + element.TextExtCY; bottom > maxY {
				maxY = bottom
			}
		}
	}
	return maxX, maxY
}

func renderShape(slidePart string, size slideSize, img *image.RGBA, element *slideElement) []model.SkipItem {
	if !element.HasTransform {
		return nil
	}
	var unsupported []model.SkipItem
	rendered := element.Rendered
	if isLineGeometry(element.PrstGeom) && element.HasLine && !element.NoLine && (element.ExtCX != 0 || element.ExtCY != 0) {
		startX, startY, endX, endY := lineEndpointsForElement(*element, size, img.Bounds())
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		drawStyledLine(img, startX, startY, endX, endY, element.LineColor, lineWidth, element.LineDash, element.LineCap)
		markerPartial := false
		if element.HeadLineMarker != "" {
			if element.HeadLineMarker == "triangle" {
				drawLineTriangleMarker(img, startX, startY, startX-endX, startY-endY, element.LineColor, lineWidth, element.HeadLineMarkerWidth, element.HeadLineMarkerLength)
			} else {
				markerPartial = true
			}
		}
		if element.TailLineMarker != "" {
			if element.TailLineMarker == "triangle" {
				drawLineTriangleMarker(img, endX, endY, endX-startX, endY-startY, element.LineColor, lineWidth, element.TailLineMarkerWidth, element.TailLineMarkerLength)
			} else {
				markerPartial = true
			}
		}
		rendered = true
		if markerPartial {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("connector object %q line markers were not rendered", elementLabel(*element))))
		}
	}
	if element.ExtCX <= 0 || element.ExtCY <= 0 {
		element.Rendered = rendered
		return unsupported
	}
	target := image.Rect(
		scaleEMU(element.OffX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY, size.CY, img.Bounds().Dy()),
		scaleEMU(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		scaleEMU(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	)
	targetFloat := floatRect{
		MinX: scaleEMUFloat(element.OffX, size.CX, img.Bounds().Dx()),
		MinY: scaleEMUFloat(element.OffY, size.CY, img.Bounds().Dy()),
		MaxX: scaleEMUFloat(element.OffX+element.ExtCX, size.CX, img.Bounds().Dx()),
		MaxY: scaleEMUFloat(element.OffY+element.ExtCY, size.CY, img.Bounds().Dy()),
	}
	if element.HasShapeAutofit && element.Text != "" && normalizedRotationDegrees(element.Rotation) == 0 {
		adjusted, err := shapeAutofitTarget(*element, target, size, img.Bounds())
		if err == nil {
			target = adjusted
			targetFloat = floatRectFromImageRect(target)
		}
	}
	target = target.Intersect(img.Bounds())
	if target.Empty() {
		element.Rendered = rendered
		return unsupported
	}
	if element.HasShadow {
		for _, message := range shadowTransformUnsupportedMessages(*element) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
		}
		if drawShapeShadow(img, target, *element, size) {
			rendered = true
		} else if element.ShadowColor.A != 0 {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q outer shadow geometry was not rendered", elementLabel(*element))))
		}
	}
	for _, message := range shape3DUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
	}
	for _, message := range shapeSoftEdgeUnsupportedMessages(*element) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
	}
	gradientFillRendered := false
	if element.HasFillGradient && !element.NoFill {
		switch element.PrstGeom {
		case "rect", "":
			drawGradientRect(img, target, element.FillGradient, false)
			rendered = true
			gradientFillRendered = true
		case "roundRect":
			drawGradientRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopLeft: true, TopRight: true, BottomLeft: true, BottomRight: true}, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		case "round1Rect":
			drawGradientRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopRight: true}, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		case "ellipse":
			drawGradientEllipse(img, target, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		}
	}
	if element.HasFillGradient && !gradientFillRendered && !element.NoFill {
		if points, ok := presetPolygonPointsForElement(*element); ok {
			drawGradientPolygon(img, target, points, element.FillGradient)
			rendered = true
			gradientFillRendered = true
		}
	}
	if element.HasFillGradient && !gradientFillRendered && !element.NoFill && len(element.CustomPath) >= 3 {
		drawGradientPolygon(img, target, transformedPathPoints(element.CustomPath, *element), element.FillGradient)
		rendered = true
		gradientFillRendered = true
	}
	if element.HasFill && !element.NoFill && !gradientFillRendered {
		switch element.PrstGeom {
		case "rect":
			fillShapeRectWithFloatBounds(img, floatRectPixelBounds(targetFloat).Intersect(img.Bounds()), targetFloat, element.FillColor)
			rendered = true
		case "roundRect":
			fillRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopLeft: true, TopRight: true, BottomLeft: true, BottomRight: true}, element.FillColor)
			rendered = true
		case "round1Rect":
			fillRoundRect(img, target, roundRectRadius(target, element.PrstGeomAdjustments), roundedCorners{TopRight: true}, element.FillColor)
			rendered = true
		}
	}
	if points, ok := presetPolygonPointsForElement(*element); ok && element.HasFill && !element.NoFill {
		drawPolygon(img, target, points, element.FillColor)
		rendered = true
	}
	if element.PrstGeom == "ellipse" && element.HasFill && !element.NoFill {
		drawEllipse(img, target, element.FillColor)
		rendered = true
	}
	if (element.PrstGeom == "curvedDownArrow" || element.PrstGeom == "curvedUpArrow") && element.HasFill && !element.NoFill {
		drawCurvedArrow(img, target, *element, element.FillColor)
		rendered = true
	}
	if element.PrstGeom == "rightBrace" && element.HasFill && !element.NoFill {
		drawPolygon(img, target, rightBracePresetPath(*element), element.FillColor)
		rendered = true
	}
	if len(element.CustomPath) >= 3 && element.HasFill && !element.NoFill {
		drawPolygon(img, target, transformedPathPoints(element.CustomPath, *element), element.FillColor)
		rendered = true
	}
	if element.HasFillGradient && !gradientFillRendered {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q gradient fill was rendered as a solid fallback", elementLabel(*element))))
	} else if element.HasFillGradient && !element.FillGradient.FullySupported {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q gradient fill was rendered with simplified layout", elementLabel(*element))))
	}
	if element.HasLine && !element.NoLine {
		lineWidth := emuLineWidthToPixels(element.LineWidth, size.CX, img.Bounds().Dx())
		switch element.PrstGeom {
		case "rect", "roundRect", "round1Rect", "":
			drawStyledRectOutlineAlignedWithCap(img, target, element.LineColor, lineWidth, element.LineDash, element.LineAlign, element.LineCap)
			rendered = true
		case "ellipse":
			drawEllipseOutline(img, target, element.LineColor, lineWidth)
			rendered = true
		case "triangle", "rightArrow", "notchedRightArrow", "chevron":
			if points, ok := presetPolygonPointsForElement(*element); ok {
				drawPolygonOutline(img, target, points, element.LineColor, lineWidth)
				rendered = true
			}
		case "curvedDownArrow", "curvedUpArrow":
			if points := curvedArrowPresetOutlinePath(*element); len(points) >= 2 {
				drawOpenPathOutline(img, target, points, element.LineColor, lineWidth)
				rendered = true
			}
		case "rightBrace":
			drawRightBrace(img, target, *element, element.LineColor, lineWidth)
			rendered = true
		}
		if len(element.CustomPath) >= 3 {
			drawPolygonOutline(img, target, transformedPathPoints(element.CustomPath, *element), element.LineColor, lineWidth)
			rendered = true
		}
	}
	if element.Text != "" && elementShouldRenderText(*element) {
		if elementShouldReportFontResolution(*element) {
			for _, message := range fontResolutionUnsupportedMessages(*element) {
				unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
			}
		}
		for _, message := range textLayoutUnsupportedMessagesForTarget(*element, textBounds(target, *element, size, img.Bounds()), renderDPIForCanvas(size, img.Bounds())) {
			unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("shape object %q %s", elementLabel(*element), message)))
		}
		if err := drawShapeTextForElement(img, target, *element, size); err != nil {
			if rendered {
				element.Rendered = true
			}
			return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("shape object %q text could not be rendered: %v", elementLabel(*element), err))}
		}
		rendered = true
	}
	element.Rendered = rendered
	return unsupported
}

func lineEndpointsForElement(element slideElement, size slideSize, bounds image.Rectangle) (int, int, int, int) {
	left := scaleEMU(element.OffX, size.CX, bounds.Dx())
	top := scaleEMU(element.OffY, size.CY, bounds.Dy())
	right := scaleEMU(element.OffX+element.ExtCX, size.CX, bounds.Dx())
	bottom := scaleEMU(element.OffY+element.ExtCY, size.CY, bounds.Dy())
	startX, startY := left, top
	endX, endY := right, bottom
	if element.FlipH {
		startX, endX = endX, startX
	}
	if element.FlipV {
		startY, endY = endY, startY
	}
	return startX, startY, endX, endY
}

func drawShapeTextForElement(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) error {
	bounds := textBounds(target, element, size, img.Bounds())
	dpi := renderDPIForCanvas(size, img.Bounds())
	rotation := normalizedRotationDegrees(element.Rotation)
	switch rotation {
	case 90, 180, 270:
		return drawRotatedShapeText(img, bounds, element, rotation, dpi)
	default:
		return drawShapeTextWithDPI(img, bounds, element, dpi)
	}
}

func shapeAutofitTarget(element slideElement, target image.Rectangle, size slideSize, canvas image.Rectangle) (image.Rectangle, error) {
	if target.Empty() {
		return target, nil
	}
	bounds := textBounds(target, element, size, canvas)
	if bounds.Empty() || bounds.Dy() <= 0 {
		return target, nil
	}
	dpi := renderDPIForCanvas(size, canvas)
	width, height, err := measuredElementTextSize(element, bounds, dpi)
	if err != nil {
		return target, err
	}
	if element.TextWrap == "none" && width > bounds.Dx() {
		target.Max.X += width - bounds.Dx()
	}
	if height > 0 && height != bounds.Dy() {
		target.Max.Y += height - bounds.Dy()
	}
	return target, nil
}

func measuredElementTextHeight(element slideElement, bounds image.Rectangle, dpi int) (int, error) {
	_, height, err := measuredElementTextSize(element, bounds, dpi)
	return height, err
}

func measuredElementTextSize(element slideElement, bounds image.Rectangle, dpi int) (int, int, error) {
	if shouldFitNormalAutofit(element) {
		element = fitNormalAutofitElement(element, bounds, dpi)
	}
	element = scaledTextElement(element, dpi)
	faces := newFontFaceCacheWithDPI(element.Italic, element.FontFamily, dpi, element.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(element.FontSize, false)
	if err != nil {
		return 0, 0, err
	}
	boldFace, err := faces.Get(element.FontSize, true)
	if err != nil {
		return 0, 0, err
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, element, bounds.Dx(), dpi)
	if err != nil {
		return 0, 0, err
	}
	measured, err := measureTextRenderLines(faces, lines, element.FontSize)
	if err != nil {
		return 0, 0, err
	}
	width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
	if err != nil {
		return 0, 0, err
	}
	return width, measuredTextHeight(measured), nil
}

func normalizedRotationDegrees(rotation int) int {
	if rotation == 0 {
		return 0
	}
	degrees := int(math.Round(float64(rotation) / 60000))
	degrees %= 360
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

func drawRotatedShapeText(img *image.RGBA, bounds image.Rectangle, element slideElement, rotation int, dpi int) error {
	if bounds.Empty() {
		return nil
	}
	temp := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	if err := drawShapeTextWithDPI(temp, temp.Bounds(), element, dpi); err != nil {
		return err
	}
	rotated := rotateRGBA(temp, rotation)
	center := image.Point{X: bounds.Min.X + bounds.Dx()/2, Y: bounds.Min.Y + bounds.Dy()/2}
	dst := image.Rect(center.X-rotated.Bounds().Dx()/2, center.Y-rotated.Bounds().Dy()/2, center.X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(), center.Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy())
	drawRGBAAt(img, dst, rotated)
	return nil
}

func rotateRGBA(src *image.RGBA, rotation int) *image.RGBA {
	bounds := src.Bounds()
	switch rotation {
	case 90:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(bounds.Max.Y-1-y, x-bounds.Min.X, src.RGBAAt(x, y))
			}
		}
		return dst
	case 180:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(bounds.Max.X-1-x, bounds.Max.Y-1-y, src.RGBAAt(x, y))
			}
		}
		return dst
	case 270:
		dst := image.NewRGBA(image.Rect(0, 0, bounds.Dy(), bounds.Dx()))
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				dst.SetRGBA(y-bounds.Min.Y, bounds.Max.X-1-x, src.RGBAAt(x, y))
			}
		}
		return dst
	default:
		return rotateRGBAArbitrary(src, float64(rotation))
	}
}

func rotateRGBAArbitrary(src *image.RGBA, degrees float64) *image.RGBA {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	radians := degrees * math.Pi / 180
	sin, cos := math.Sin(radians), math.Cos(radians)
	outputWidth := int(math.Ceil(math.Abs(float64(width)*cos) + math.Abs(float64(height)*sin)))
	outputHeight := int(math.Ceil(math.Abs(float64(width)*sin) + math.Abs(float64(height)*cos)))
	if outputWidth <= 0 || outputHeight <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 0, 0))
	}
	dst := image.NewRGBA(image.Rect(0, 0, outputWidth, outputHeight))
	sourceCenterX := float64(bounds.Min.X) + float64(width-1)/2
	sourceCenterY := float64(bounds.Min.Y) + float64(height-1)/2
	outputCenterX := float64(outputWidth-1) / 2
	outputCenterY := float64(outputHeight-1) / 2
	for y := 0; y < outputHeight; y++ {
		for x := 0; x < outputWidth; x++ {
			dx := float64(x) - outputCenterX
			dy := float64(y) - outputCenterY
			sourceX := sourceCenterX + dx*cos + dy*sin
			sourceY := sourceCenterY - dx*sin + dy*cos
			if sourceX < float64(bounds.Min.X) || sourceX > float64(bounds.Max.X-1) || sourceY < float64(bounds.Min.Y) || sourceY > float64(bounds.Max.Y-1) {
				continue
			}
			dst.SetRGBA(x, y, bilinearRGBAAt(src, sourceX, sourceY))
		}
	}
	return dst
}

func bilinearRGBAAt(src *image.RGBA, x float64, y float64) color.RGBA {
	bounds := src.Bounds()
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1 := min(x0+1, bounds.Max.X-1)
	y1 := min(y0+1, bounds.Max.Y-1)
	fx := x - float64(x0)
	fy := y - float64(y0)
	c00 := src.RGBAAt(x0, y0)
	c10 := src.RGBAAt(x1, y0)
	c01 := src.RGBAAt(x0, y1)
	c11 := src.RGBAAt(x1, y1)
	return color.RGBA{
		R: interpolateBilinearChannel(c00.R, c10.R, c01.R, c11.R, fx, fy),
		G: interpolateBilinearChannel(c00.G, c10.G, c01.G, c11.G, fx, fy),
		B: interpolateBilinearChannel(c00.B, c10.B, c01.B, c11.B, fx, fy),
		A: interpolateBilinearChannel(c00.A, c10.A, c01.A, c11.A, fx, fy),
	}
}

func interpolateBilinearChannel(c00 uint8, c10 uint8, c01 uint8, c11 uint8, fx float64, fy float64) uint8 {
	top := float64(c00)*(1-fx) + float64(c10)*fx
	bottom := float64(c01)*(1-fx) + float64(c11)*fx
	return clampColor(int64(math.Round(top*(1-fy) + bottom*fy)))
}

func drawRGBAAt(dst *image.RGBA, target image.Rectangle, src *image.RGBA) {
	clipped := target.Intersect(dst.Bounds())
	if clipped.Empty() {
		return
	}
	sourcePoint := image.Point{
		X: src.Bounds().Min.X + clipped.Min.X - target.Min.X,
		Y: src.Bounds().Min.Y + clipped.Min.Y - target.Min.Y,
	}
	draw.Draw(dst, clipped, src, sourcePoint, draw.Over)
}

func isRectGeometry(geometry string) bool {
	return geometry == "rect" || geometry == "roundRect" || geometry == "round1Rect"
}

func isLineGeometry(geometry string) bool {
	return geometry == "line" || geometry == "straightConnector1"
}

func isFilledPresetPolygonGeometry(geometry string) bool {
	_, ok := presetPolygonPoints(geometry)
	return ok
}

func presetPolygonPoints(geometry string) ([]pathPoint, bool) {
	switch geometry {
	case "triangle":
		return []pathPoint{{X: 0.5, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, true
	case "rightArrow":
		return []pathPoint{
			{X: 0, Y: 0.25},
			{X: 0.65, Y: 0.25},
			{X: 0.65, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.65, Y: 1},
			{X: 0.65, Y: 0.75},
			{X: 0, Y: 0.75},
		}, true
	case "notchedRightArrow":
		return []pathPoint{
			{X: 0, Y: 0},
			{X: 0.78, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.78, Y: 1},
			{X: 0, Y: 1},
			{X: 0.18, Y: 0.5},
		}, true
	case "chevron":
		return []pathPoint{
			{X: 0, Y: 0},
			{X: 0.75, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.75, Y: 1},
			{X: 0, Y: 1},
			{X: 0.25, Y: 0.5},
		}, true
	default:
		return nil, false
	}
}

func presetPolygonPointsForElement(element slideElement) ([]pathPoint, bool) {
	switch element.PrstGeom {
	case "rightArrow":
		return transformedPathPoints(rightArrowPresetPoints(element), element), true
	case "chevron":
		return transformedPathPoints(chevronPresetPoints(element), element), true
	case "notchedRightArrow":
		return transformedPathPoints(notchedRightArrowPresetPoints(element), element), true
	}
	points, ok := presetPolygonPoints(element.PrstGeom)
	if !ok {
		return nil, false
	}
	if element.PrstGeom == "triangle" {
		if adj, ok := element.PrstGeomAdjustments["adj"]; ok {
			points[0].X = clampPathCoordinate(float64(adj) / 100000)
		}
	}
	return transformedPathPoints(points, element), true
}

func chevronPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj := presetAdjustment(element, "adj", 50000)
	maxAdj := 100000 * w / ss
	a := clampFloat(adj, 0, maxAdj)
	x1 := ss * a / 100000
	x2 := w - x1
	return []pathPoint{
		{X: 0, Y: 0},
		{X: x2 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x2 / w, Y: 1},
		{X: 0, Y: 1},
		{X: x1 / w, Y: 0.5},
	}
}

func rightArrowPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj1 := presetAdjustment(element, "adj1", 50000)
	adj2 := presetAdjustment(element, "adj2", 50000)
	a1 := clampFloat(adj1, 0, 100000)
	maxAdj2 := 100000 * w / ss
	a2 := clampFloat(adj2, 0, maxAdj2)
	dx1 := ss * a2 / 100000
	x1 := w - dx1
	dy1 := h * a1 / 200000
	y1 := 0.5 - dy1/h
	y2 := 0.5 + dy1/h
	return []pathPoint{
		{X: 0, Y: y1},
		{X: x1 / w, Y: y1},
		{X: x1 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x1 / w, Y: 1},
		{X: x1 / w, Y: y2},
		{X: 0, Y: y2},
	}
}

func notchedRightArrowPresetPoints(element slideElement) []pathPoint {
	w, h := positiveGeometryDimensions(element)
	ss := math.Min(w, h)
	adj1 := presetAdjustment(element, "adj1", 50000)
	adj2 := presetAdjustment(element, "adj2", 50000)
	a1 := clampFloat(adj1, 0, 100000)
	maxAdj2 := 100000 * w / ss
	a2 := clampFloat(adj2, 0, maxAdj2)
	dx2 := ss * a2 / 100000
	x2 := w - dx2
	dy1 := h * a1 / 200000
	x1 := 0.0
	if h > 0 {
		x1 = dy1 * dx2 / (h / 2)
	}
	y1 := 0.5 - dy1/h
	y2 := 0.5 + dy1/h
	return []pathPoint{
		{X: 0, Y: y1},
		{X: x2 / w, Y: y1},
		{X: x2 / w, Y: 0},
		{X: 1, Y: 0.5},
		{X: x2 / w, Y: 1},
		{X: x2 / w, Y: y2},
		{X: 0, Y: y2},
		{X: x1 / w, Y: 0.5},
	}
}

func positiveGeometryDimensions(element slideElement) (float64, float64) {
	w := float64(element.ExtCX)
	h := float64(element.ExtCY)
	if w <= 0 {
		w = 1
	}
	if h <= 0 {
		h = 1
	}
	return w, h
}

func presetAdjustment(element slideElement, name string, fallback float64) float64 {
	if value, ok := element.PrstGeomAdjustments[name]; ok {
		return float64(value)
	}
	return fallback
}

func clampFloat(value float64, minValue float64, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func clampPathCoordinate(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func transformedPathPoints(points []pathPoint, element slideElement) []pathPoint {
	if !element.FlipH && !element.FlipV && normalizedRotationDegrees(element.Rotation) == 0 {
		return points
	}
	transformed := make([]pathPoint, 0, len(points))
	for _, point := range points {
		next := point
		if element.FlipH {
			next.X = 1 - next.X
		}
		if element.FlipV {
			next.Y = 1 - next.Y
		}
		transformed = append(transformed, next)
	}
	return rotatePathPoints(transformed, normalizedRotationDegrees(element.Rotation))
}

func rotatePathPoints(points []pathPoint, rotation int) []pathPoint {
	if rotation != 90 && rotation != 180 && rotation != 270 {
		return points
	}
	rotated := make([]pathPoint, 0, len(points))
	for _, point := range points {
		switch rotation {
		case 90:
			rotated = append(rotated, pathPoint{X: 1 - point.Y, Y: point.X})
		case 180:
			rotated = append(rotated, pathPoint{X: 1 - point.X, Y: 1 - point.Y})
		case 270:
			rotated = append(rotated, pathPoint{X: point.Y, Y: 1 - point.X})
		}
	}
	return rotated
}

func drawShapeShadow(img *image.RGBA, target image.Rectangle, element slideElement, size slideSize) bool {
	if element.ShadowColor.A == 0 {
		return false
	}
	offset := shadowOffset(element, size, img.Bounds().Dx())
	shadowBounds := target.Add(offset)
	blur := shadowBlurPixels(element, size, img.Bounds().Dx())
	if !shadowIntersectsCanvas(shadowBounds, blur, img.Bounds()) {
		return false
	}
	switch {
	case isRectGeometry(element.PrstGeom):
		drawSoftRect(img, shadowBounds, element.ShadowColor, blur)
	case element.PrstGeom == "ellipse":
		drawSoftEllipse(img, shadowBounds, element.ShadowColor, blur)
	case element.PrstGeom == "triangle":
		drawSoftPolygon(img, shadowBounds, transformedPathPoints([]pathPoint{{X: 0.5, Y: 0}, {X: 1, Y: 1}, {X: 0, Y: 1}}, element), element.ShadowColor, blur)
	case element.PrstGeom == "rightArrow":
		drawSoftPolygon(img, shadowBounds, transformedPathPoints([]pathPoint{
			{X: 0, Y: 0.25},
			{X: 0.65, Y: 0.25},
			{X: 0.65, Y: 0},
			{X: 1, Y: 0.5},
			{X: 0.65, Y: 1},
			{X: 0.65, Y: 0.75},
			{X: 0, Y: 0.75},
		}, element), element.ShadowColor, blur)
	case isFilledPresetPolygonGeometry(element.PrstGeom):
		if points, ok := presetPolygonPointsForElement(element); ok {
			drawSoftPolygon(img, shadowBounds, points, element.ShadowColor, blur)
		} else {
			return false
		}
	case len(element.CustomPath) >= 3:
		drawSoftPolygon(img, shadowBounds, transformedPathPoints(element.CustomPath, element), element.ShadowColor, blur)
	default:
		return false
	}
	return true
}

func shadowTransformUnsupportedMessages(element slideElement) []string {
	var messages []string
	if (element.HasShadowScaleX && element.ShadowScaleX != 100000) || (element.HasShadowScaleY && element.ShadowScaleY != 100000) || (element.HasShadowSkewX && element.ShadowSkewX != 0) || (element.HasShadowSkewY && element.ShadowSkewY != 0) {
		messages = append(messages, "outer shadow scale/skew transform was not rendered")
	}
	if element.HasShadowRotateWithShape && !element.ShadowRotateWithShape && normalizedRotationDegrees(element.Rotation) != 0 {
		messages = append(messages, "outer shadow rotate-with-shape transform was not rendered")
	}
	return messages
}

func shape3DUnsupportedMessages(element slideElement) []string {
	if !element.HasShape3D {
		return nil
	}
	if len(element.Shape3DFeatures) == 0 {
		return []string{"3-D shape properties were not rendered"}
	}
	features := append([]string{}, element.Shape3DFeatures...)
	sort.Strings(features)
	return []string{fmt.Sprintf("%s were not rendered", strings.Join(features, ", "))}
}

func shapeSoftEdgeUnsupportedMessages(element slideElement) []string {
	if !element.HasSoftEdge || element.SoftEdgeRadius <= 0 {
		return nil
	}
	return []string{"soft edge effect was not rendered"}
}

func shadowIntersectsCanvas(bounds image.Rectangle, blur int, canvas image.Rectangle) bool {
	if bounds.Empty() || canvas.Empty() {
		return false
	}
	if blur > 0 {
		bounds = bounds.Inset(-blur)
	}
	return !bounds.Intersect(canvas).Empty()
}

func shadowOffset(element slideElement, size slideSize, outputWidth int) image.Point {
	distance := scaleEMU(element.ShadowDistance, size.CX, outputWidth)
	if distance == 0 && element.ShadowDistance > 0 {
		distance = 1
	}
	angle := float64(element.ShadowDirection) / 60000 * math.Pi / 180
	return image.Point{
		X: int(math.Round(math.Cos(angle) * float64(distance))),
		Y: int(math.Round(math.Sin(angle) * float64(distance))),
	}
}

func shadowBlurPixels(element slideElement, size slideSize, outputWidth int) int {
	blur := scaleEMU(element.ShadowBlur, size.CX, outputWidth)
	if blur < 0 {
		return 0
	}
	return blur
}
