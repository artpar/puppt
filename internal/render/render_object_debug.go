package render

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/artpar/puppt/internal/model"
)

func (debug *ObjectDebugOptions) nextObjectZOrder() int {
	if debug == nil {
		return 0
	}
	debug.nextZOrder++
	return debug.nextZOrder
}

func (debug *ObjectDebugOptions) shouldPaintObject(zOrder int) bool {
	if debug == nil {
		return true
	}
	if debug.TargetZOrder <= 0 {
		return true
	}
	switch debug.Mode {
	case ObjectDebugRenderBefore:
		return zOrder < debug.TargetZOrder
	case ObjectDebugRenderObjectOnly:
		return zOrder == debug.TargetZOrder
	case ObjectDebugRenderThrough:
		return zOrder <= debug.TargetZOrder
	default:
		return true
	}
}

func objectDebugUsesOwnBackground(debug *ObjectDebugOptions) bool {
	return debug != nil && debug.Mode == ObjectDebugRenderObjectOnly
}

func objectDebugBackgroundColor(debug *ObjectDebugOptions) (color.RGBA, bool) {
	if debug == nil || !debug.HasFlatBackground {
		return color.RGBA{}, false
	}
	return debug.FlatBackground, true
}

func appendPaintedObjectRecord(debug *ObjectDebugOptions, record PaintedObject) {
	if debug == nil {
		return
	}
	debug.Records = append(debug.Records, record)
}

func paintedObjectRecord(slidePart string, sourcePart string, element slideElement, zOrder int, size slideSize, canvas image.Rectangle, before *image.RGBA, after *image.RGBA, painted bool, unsupported []model.SkipItem) PaintedObject {
	record := PaintedObject{
		SlidePart:        slidePart,
		SourcePart:       sourcePart,
		XMLPath:          objectXMLPath(element),
		CNvPrID:          element.ID,
		CNvPrName:        element.Name,
		CNvPrDescription: element.Description,
		CNvPrTitle:       element.Title,
		CNvPrCreationID:  element.CreationID,
		Kind:             element.Kind,
		ZOrder:           zOrder,
		Bounds:           objectEMUPointBounds(element),
		ResolvedStyle:    objectStyleSummary(element),
		PixelBounds:      objectPixelBounds(element, size, canvas),
		FractionalBounds: objectFractionalPixelBounds(element, size, canvas),
		Unsupported:      objectUnsupportedSummary(unsupported),
		Painted:          painted,
	}
	if before != nil && after != nil {
		if bounds, ok := changedPixelBounds(before, after); ok {
			record.OutputPixelBounds = &bounds
		}
	}
	return record
}

func objectUnsupportedSummary(items []model.SkipItem) []ObjectUnsupported {
	if len(items) == 0 {
		return nil
	}
	summary := make([]ObjectUnsupported, 0, len(items))
	for _, item := range items {
		summary = append(summary, ObjectUnsupported{
			Code:    item.Code,
			Part:    item.Part,
			Message: item.Message,
		})
	}
	return summary
}

func objectXMLPath(element slideElement) string {
	container := "p:spTree"
	node := element.Kind
	switch element.Kind {
	case "sp":
		node = "p:sp"
	case "cxnSp":
		node = "p:cxnSp"
	case "pic":
		node = "p:pic"
	case "graphicFrame":
		node = "p:graphicFrame"
	}
	predicate := ""
	if element.ID != "" {
		predicate = fmt.Sprintf("[.//p:cNvPr/@id=%q]", element.ID)
	} else if element.Name != "" {
		predicate = fmt.Sprintf("[.//p:cNvPr/@name=%q]", element.Name)
	}
	return fmt.Sprintf("/p:sld/p:cSld/%s/%s%s", container, node, predicate)
}

func objectEMUPointBounds(element slideElement) ObjectEMUPointBounds {
	if !element.HasTransform {
		return ObjectEMUPointBounds{}
	}
	return ObjectEMUPointBounds{
		X:  element.OffX,
		Y:  element.OffY,
		CX: element.ExtCX,
		CY: element.ExtCY,
	}
}

func objectPixelBounds(element slideElement, size slideSize, canvas image.Rectangle) ObjectPixelBounds {
	if !element.HasTransform || size.CX <= 0 || size.CY <= 0 || canvas.Dx() <= 0 || canvas.Dy() <= 0 {
		return ObjectPixelBounds{}
	}
	if isLineGeometry(element.PrstGeom) && (element.ExtCX != 0 || element.ExtCY != 0) {
		x1, y1, x2, y2 := lineEndpointsForElement(element, size, canvas)
		width := emuLineWidthToPixels(element.LineWidth, size.CX, canvas.Dx())
		pad := max(1, width)
		return pixelBoundsFromRect(image.Rect(min(x1, x2)-pad, min(y1, y2)-pad, max(x1, x2)+pad+1, max(y1, y2)+pad+1).Intersect(canvas))
	}
	return renderElementClippedPixelBounds(element, size, canvas)
}

func objectFractionalPixelBounds(element slideElement, size slideSize, canvas image.Rectangle) ObjectFloatBounds {
	if !element.HasTransform || size.CX <= 0 || size.CY <= 0 || canvas.Dx() <= 0 || canvas.Dy() <= 0 {
		return ObjectFloatBounds{}
	}
	return elementFractionalTarget(element, size, canvas)
}

func pixelBoundsFromRect(rect image.Rectangle) ObjectPixelBounds {
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

func changedPixelBounds(before *image.RGBA, after *image.RGBA) (ObjectPixelBounds, bool) {
	if before == nil || after == nil || before.Bounds() != after.Bounds() {
		return ObjectPixelBounds{}, false
	}
	bounds := before.Bounds()
	var changed image.Rectangle
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if before.RGBAAt(x, y) == after.RGBAAt(x, y) {
				continue
			}
			point := image.Pt(x, y)
			if changed.Empty() {
				changed = image.Rectangle{Min: point, Max: point.Add(image.Pt(1, 1))}
			} else {
				changed = changed.Union(image.Rectangle{Min: point, Max: point.Add(image.Pt(1, 1))})
			}
		}
	}
	if changed.Empty() {
		return ObjectPixelBounds{}, false
	}
	return pixelBoundsFromRect(changed), true
}

func objectStyleSummary(element slideElement) ObjectStyleSummary {
	summary := ObjectStyleSummary{
		Geometry:                element.PrstGeom,
		FontFamily:              objectTextFontFamily(element),
		FontFamilies:            objectTextFontFamilies(element),
		FontSize:                element.FontSize,
		ParagraphFontSize:       textParagraphsFontSize(element.TextParagraphs),
		Bold:                    objectTextBold(element),
		Italic:                  objectTextItalic(element),
		TextAlign:               element.TextAlign,
		TextBox:                 element.IsTextBox,
		TextBodyProperties:      objectTextBodyPropertiesSummary(element),
		TextParagraphProperties: objectTextParagraphPropertiesSummary(element.TextParagraphs),
		Description:             element.Description,
		Title:                   element.Title,
		CreationID:              element.CreationID,
		NonVisualProperties:     append([]string{}, element.NonVisualProperties...),
		NonVisualLocks:          append([]string{}, element.NonVisualLocks...),
		EmbedID:                 element.EmbedID,
		SVGEmbedID:              element.SVGEmbedID,
		Image:                   objectImageSummary(element),
		ImageCrop:               objectImageCropSummary(element),
		ImageEffects:            objectImageEffectSummary(element),
		ImageUnsupported:        append([]string{}, element.ImageUnsupported...),
		EffectUnsupported:       append([]string{}, element.EffectUnsupported...),
		Table:                   element.HasTable,
		TableStyleID:            element.Table.StyleID,
		TableProperties:         tablePropertiesSummary(element.Table),
		TableColumnIDs:          append([]string{}, element.Table.ColumnIDs...),
		TableRowIDs:             tableRowIDs(element.Table.Rows),
		TableUnsupported:        append([]string{}, element.Table.UnsupportedFeatures...),
		Shadow:                  element.HasShadow,
		GradientFill:            element.HasFillGradient,
		NoFill:                  element.NoFill,
		NoLine:                  element.NoLine,
	}
	if element.HasFill {
		summary.Fill = formatObjectColor(element.FillColor)
	}
	if element.HasPatternFill {
		summary.PatternFill = fmt.Sprintf("%s/%s/%s", element.PatternFill.Preset, formatObjectColor(element.PatternFill.Foreground), formatObjectColor(element.PatternFill.Background))
	}
	if len(element.CustomPath) > 0 {
		if summary.Geometry == "" {
			summary.Geometry = "customPath"
		}
		customPath := transformedPathPoints(element.CustomPath, element)
		summary.CustomPathPoints = len(customPath)
		summary.CustomPathCommands = len(element.CustomPathCommands)
		summary.CustomPathCoordinates = make([]ObjectFloatPoint, 0, len(customPath))
		for _, point := range customPath {
			summary.CustomPathCoordinates = append(summary.CustomPathCoordinates, ObjectFloatPoint{X: point.X, Y: point.Y})
		}
		if bounds, ok := normalizedPathBounds(customPath); ok {
			summary.CustomPathBounds = &bounds
		}
		summary.CustomPathUnsupported = append([]string{}, element.CustomPathUnsupported...)
	}
	if element.HasLine {
		summary.Line = fmt.Sprintf("%s/%d", formatObjectColor(element.LineColor), element.LineWidth)
	}
	if element.HasShadow {
		summary.ShadowColor = formatObjectColor(element.ShadowColor)
		summary.ShadowBlur = element.ShadowBlur
		summary.ShadowDistance = element.ShadowDistance
		summary.ShadowDirection = element.ShadowDirection
		summary.ShadowAlignment = element.ShadowAlignment
		if element.HasShadowScaleX {
			summary.ShadowScaleX = element.ShadowScaleX
		}
		if element.HasShadowScaleY {
			summary.ShadowScaleY = element.ShadowScaleY
		}
		if element.HasShadowSkewX {
			summary.ShadowSkewX = element.ShadowSkewX
		}
		if element.HasShadowSkewY {
			summary.ShadowSkewY = element.ShadowSkewY
		}
	}
	if element.Text != "" {
		summary.Text = summarizeObjectText(element.Text)
	}
	if textColor, ok := objectTextColor(element); ok {
		summary.TextColor = formatObjectColor(textColor)
	}
	if element.IsPlaceholder {
		summary.Placeholder = element.PlaceholderType
		if element.PlaceholderIdx != "" {
			summary.Placeholder += "#" + element.PlaceholderIdx
		}
	}
	return summary
}

func tablePropertiesSummary(table tableModel) []string {
	if !table.FirstRow && !table.FirstCol && !table.LastRow && !table.LastCol && !table.BandRow && !table.BandCol && !table.NoBackground && !table.HasBackground {
		return nil
	}
	var summary []string
	if table.FirstRow {
		summary = append(summary, "firstRow=true")
	}
	if table.FirstCol {
		summary = append(summary, "firstCol=true")
	}
	if table.LastRow {
		summary = append(summary, "lastRow=true")
	}
	if table.LastCol {
		summary = append(summary, "lastCol=true")
	}
	if table.BandRow {
		summary = append(summary, "bandRow=true")
	}
	if table.BandCol {
		summary = append(summary, "bandCol=true")
	}
	if table.NoBackground {
		summary = append(summary, "noFill=true")
	} else if table.HasBackground {
		summary = append(summary, "fill=true")
	}
	return summary
}

func tableRowIDs(rows []tableRow) []string {
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func objectImageSummary(element slideElement) string {
	var parts []string
	if element.EmbedID != "" {
		parts = append(parts, "embed="+element.EmbedID)
	}
	if element.LinkID != "" {
		parts = append(parts, "link="+element.LinkID)
	}
	if element.SVGEmbedID != "" {
		parts = append(parts, "svg="+element.SVGEmbedID)
	}
	if element.ImageMediaPart != "" {
		parts = append(parts, "part="+element.ImageMediaPart)
	}
	if element.ImageContentType != "" {
		parts = append(parts, "type="+element.ImageContentType)
	}
	if element.ImageWidth > 0 && element.ImageHeight > 0 {
		parts = append(parts, fmt.Sprintf("size=%dx%d", element.ImageWidth, element.ImageHeight))
	}
	if element.DiagramDataID != "" {
		parts = append(parts, "diagram="+element.DiagramDataID)
	}
	if element.BWMode != "" {
		parts = append(parts, "bwMode="+element.BWMode)
	}
	return strings.Join(parts, " ")
}

func objectImageCropSummary(element slideElement) string {
	if !element.HasCrop {
		return ""
	}
	return fmt.Sprintf("l=%d t=%d r=%d b=%d", element.CropLeft, element.CropTop, element.CropRight, element.CropBottom)
}

func objectImageEffectSummary(element slideElement) []string {
	var effects []string
	if element.HasImageAlphaModFix {
		effects = append(effects, fmt.Sprintf("alphaModFix=%d", element.ImageAlphaModFixPct))
	}
	if element.HasImageAlphaModulate {
		effects = append(effects, fmt.Sprintf("alphaMod=%d", element.ImageAlphaModulatePct))
	}
	if element.HasImageGrayscale {
		effects = append(effects, "grayscl")
	}
	if element.HasImageBiLevel {
		effects = append(effects, fmt.Sprintf("biLevel=%d", element.ImageBiLevelThreshold))
	}
	if element.HasImageLuminance {
		effects = append(effects, fmt.Sprintf("lum bright=%d contrast=%d", element.ImageLuminanceBright, element.ImageLuminanceContrast))
	}
	if element.HasImageAlphaBiLevel {
		effects = append(effects, fmt.Sprintf("alphaBiLevel=%d", element.ImageAlphaBiLevelThreshold))
	}
	if element.HasImageAlphaCeiling {
		effects = append(effects, "alphaCeiling")
	}
	if element.HasImageAlphaFloor {
		effects = append(effects, "alphaFloor")
	}
	if element.HasImageAlphaInverse {
		effects = append(effects, "alphaInv")
	}
	if element.HasImageAlphaReplace {
		effects = append(effects, fmt.Sprintf("alphaRepl=%d", element.ImageAlphaReplacePct))
	}
	if element.HasImageColorChange {
		effects = append(effects, "clrChange")
	}
	if element.HasImageColorReplace {
		effects = append(effects, "clrRepl")
	}
	if element.HasImageDuotone {
		effects = append(effects, "duotone")
	}
	if element.BlipFillMode != "" {
		effects = append(effects, "fillMode="+element.BlipFillMode)
	}
	if element.BlipCompressionState != "" {
		effects = append(effects, "cstate="+element.BlipCompressionState)
	}
	effects = append(effects, element.ImageUnsupported...)
	effects = append(effects, element.EffectUnsupported...)
	if element.HasBlipRotWithShape {
		effects = append(effects, fmt.Sprintf("rotWithShape=%t", element.BlipRotWithShape))
	}
	if element.HasSoftEdge {
		effects = append(effects, fmt.Sprintf("softEdge=%d", element.SoftEdgeRadius))
	}
	if element.HasBlur {
		effects = append(effects, fmt.Sprintf("blur=%d grow=%t", element.BlurRadius, element.BlurGrow))
	}
	if element.HasFillOverlay {
		effects = append(effects, "fillOverlay="+element.FillOverlayBlend)
	}
	if element.HasInnerShadow {
		effects = append(effects, fmt.Sprintf("innerShdw=%d/%d/%d", element.InnerShadowBlur, element.InnerShadowDistance, element.InnerShadowDirection))
	}
	if element.HasReflection {
		effects = append(effects, fmt.Sprintf("reflection=%d/%d-%d", element.ReflectionBlur, element.ReflectionStartAlpha, element.ReflectionEndAlpha))
	}
	if element.HasGlow {
		effects = append(effects, fmt.Sprintf("glow=%d", element.GlowRadius))
	}
	return effects
}

func normalizedPathBounds(points []pathPoint) (ObjectFloatBounds, bool) {
	if len(points) == 0 {
		return ObjectFloatBounds{}, false
	}
	bounds := ObjectFloatBounds{
		MinX: points[0].X,
		MinY: points[0].Y,
		MaxX: points[0].X,
		MaxY: points[0].Y,
	}
	for _, point := range points[1:] {
		if point.X < bounds.MinX {
			bounds.MinX = point.X
		}
		if point.X > bounds.MaxX {
			bounds.MaxX = point.X
		}
		if point.Y < bounds.MinY {
			bounds.MinY = point.Y
		}
		if point.Y > bounds.MaxY {
			bounds.MaxY = point.Y
		}
	}
	return bounds, true
}

func objectTextBold(element slideElement) bool {
	for _, paragraph := range element.TextParagraphs {
		if paragraph.Bold {
			return true
		}
		for _, run := range paragraph.Runs {
			if resolvedRunBold(run, paragraph) {
				return true
			}
		}
	}
	return false
}

func objectTextItalic(element slideElement) bool {
	if element.Italic {
		return true
	}
	for _, paragraph := range element.TextParagraphs {
		if paragraph.Italic {
			return true
		}
		for _, run := range paragraph.Runs {
			if resolvedRunItalic(run, paragraph) {
				return true
			}
		}
	}
	return false
}

func objectTextFontFamily(element slideElement) string {
	for _, family := range objectTextFontFamilies(element) {
		return family
	}
	return element.FontFamily
}

func objectTextFontFamilies(element slideElement) []string {
	seen := map[string]bool{}
	var families []string
	add := func(family string) {
		family = strings.TrimSpace(family)
		if family == "" || seen[family] {
			return
		}
		seen[family] = true
		families = append(families, family)
	}
	for _, paragraph := range element.TextParagraphs {
		for _, run := range paragraph.Runs {
			add(run.FontFamily)
		}
		add(paragraph.FontFamily)
		add(paragraph.BulletFontFamily)
	}
	return families
}

func objectTextColor(element slideElement) (color.RGBA, bool) {
	if element.HasTextColor {
		return element.TextColor, true
	}
	for _, paragraph := range element.TextParagraphs {
		if paragraph.HasTextColor {
			return paragraph.TextColor, true
		}
		for _, run := range paragraph.Runs {
			if run.HasTextColor {
				return run.TextColor, true
			}
		}
	}
	return color.RGBA{}, false
}

func formatObjectColor(value color.RGBA) string {
	return fmt.Sprintf("#%02X%02X%02X/%02X", value.R, value.G, value.B, value.A)
}

func summarizeObjectText(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= 80 {
		return text
	}
	return text[:77] + "..."
}

func objectTextBodyPropertiesSummary(element slideElement) []string {
	var summary []string
	if element.HasTextWrap {
		summary = append(summary, "wrap="+element.TextWrap)
	}
	if element.HasShapeAutofit {
		summary = append(summary, "spAutoFit=true")
	}
	if element.HasNormAutofit {
		summary = append(summary, "normAutofit=true")
	}
	if element.HasNoAutofit {
		summary = append(summary, "noAutofit=true")
	}
	if element.HasFontScalePct {
		summary = append(summary, fmt.Sprintf("fontScale=%d", element.FontScalePct))
	}
	if element.HasLineSpacingReductionPct {
		summary = append(summary, fmt.Sprintf("lnSpcReduction=%d", element.LineSpacingReductionPct))
	}
	if element.HasFirstLastSpacing {
		summary = append(summary, fmt.Sprintf("spcFirstLastPara=%t", element.IncludeFirstLastSpacing))
	}
	if element.HasTextRightToLeftColumns {
		summary = append(summary, fmt.Sprintf("rtlCol=%t", element.TextRightToLeftColumns))
	}
	return summary
}

func objectTextParagraphPropertiesSummary(paragraphs []textParagraph) []string {
	if len(paragraphs) == 0 {
		return nil
	}
	var summary []string
	seen := map[string]bool{}
	add := func(value string) {
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		summary = append(summary, value)
	}
	for _, paragraph := range paragraphs {
		if paragraph.HasRTL {
			add(fmt.Sprintf("rtl=%t", paragraph.RTL))
		}
		if paragraph.HasEALineBreak {
			add(fmt.Sprintf("eaLnBrk=%t", paragraph.EALineBreak))
		}
		if paragraph.HasLatinLineBreak {
			add(fmt.Sprintf("latinLnBrk=%t", paragraph.LatinLineBreak))
		}
		if paragraph.HasHangingPunct {
			add(fmt.Sprintf("hangingPunct=%t", paragraph.HangingPunct))
		}
	}
	return summary
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	if src == nil {
		return nil
	}
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}

func writeObjectDebugArtifacts(debug *ObjectDebugOptions, dpi int, record *PaintedObject, beforeImage *image.RGBA, objectImage *image.RGBA, throughImage *image.RGBA) {
	if debug == nil || debug.ArtifactDir == "" || record == nil {
		return
	}
	base := fmt.Sprintf("%04d-%s-%s", record.ZOrder, record.CNvPrID, record.CNvPrName)
	base = sanitizeObjectArtifactName(base)
	if base == "" {
		base = fmt.Sprintf("%04d-object", record.ZOrder)
	}
	beforePath := filepath.Join(debug.ArtifactDir, base+"-before.png")
	if beforeImage != nil {
		output := cloneRGBA(beforeImage)
		applyDisplayP3OutputTransform(output)
		if err := writePNGWithDPI(beforePath, output, dpi); err == nil {
			record.BeforeArtifactPath = beforePath
		}
	}
	objectPath := filepath.Join(debug.ArtifactDir, base+"-object.png")
	if objectImage != nil {
		output := cloneRGBA(objectImage)
		applyDisplayP3OutputTransform(output)
		if err := writePNGWithDPI(objectPath, output, dpi); err == nil {
			record.ObjectArtifactPath = objectPath
		}
	}
	throughPath := filepath.Join(debug.ArtifactDir, base+"-through.png")
	if throughImage != nil {
		output := cloneRGBA(throughImage)
		applyDisplayP3OutputTransform(output)
		if err := writePNGWithDPI(throughPath, output, dpi); err == nil {
			record.ThroughArtifactPath = throughPath
		}
	}
}

var objectArtifactUnsafeChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeObjectArtifactName(name string) string {
	name = strings.TrimSpace(name)
	name = objectArtifactUnsafeChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._")
	if len(name) > 120 {
		name = name[:120]
	}
	return name
}
