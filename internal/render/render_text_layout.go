package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func drawShapeText(img *image.RGBA, bounds image.Rectangle, element slideElement) error {
	return drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI)
}

func drawShapeTextWithDPI(img *image.RGBA, bounds image.Rectangle, element slideElement, dpi int) error {
	clip := shapeTextClipRect(element, bounds, img.Bounds())
	if clip == img.Bounds() {
		return drawShapeTextLayerWithDPI(img, bounds, element, dpi)
	}
	if clip.Empty() {
		return nil
	}
	layer := image.NewRGBA(img.Bounds())
	if err := drawShapeTextLayerWithDPI(layer, bounds, element, dpi); err != nil {
		return err
	}
	draw.Draw(img, clip, layer, clip.Min, draw.Over)
	return nil
}

func shapeTextClipRect(element slideElement, bounds image.Rectangle, canvas image.Rectangle) image.Rectangle {
	clip := canvas
	if !shapeTextHorizontalOverflowAllowed(element) {
		clip.Min.X = maxInt(clip.Min.X, bounds.Min.X)
		clip.Max.X = minInt(clip.Max.X, bounds.Max.X)
	}
	if !shapeTextVerticalOverflowAllowed(element) {
		clip.Min.Y = maxInt(clip.Min.Y, bounds.Min.Y)
		clip.Max.Y = minInt(clip.Max.Y, bounds.Max.Y)
	}
	return clip.Intersect(canvas)
}

func shapeTextHorizontalOverflowAllowed(element slideElement) bool {
	switch strings.TrimSpace(element.TextHorizontalOverflow) {
	case "", "overflow":
		return true
	default:
		return false
	}
}

func shapeTextVerticalOverflowAllowed(element slideElement) bool {
	switch strings.TrimSpace(element.TextVerticalOverflow) {
	case "", "overflow":
		return true
	default:
		return false
	}
}

func drawShapeTextLayerWithDPI(img *image.RGBA, bounds image.Rectangle, element slideElement, dpi int) error {
	dpi = normalizeOutputDPI(dpi)
	if shouldFitNormalAutofit(element) {
		element = fitNormalAutofitElement(element, bounds, dpi)
	}
	element = scaledTextElement(element, dpi)
	faces := newFontFaceCacheWithDPI(element.Italic, element.FontFamily, dpi, element.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(element.FontSize, false)
	if err != nil {
		return err
	}
	boldFace, err := faces.Get(element.FontSize, true)
	if err != nil {
		return err
	}

	textColor := color.RGBA{A: 255}
	if element.HasTextColor {
		textColor = element.TextColor
	}
	drawer := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(textColor),
		Face: face,
	}
	maxWidth := bounds.Dx()
	lines, err := textRenderLinesForElement(faces, face, boldFace, element, maxWidth, dpi)
	if err != nil {
		return err
	}
	measured, err := measureTextRenderLines(faces, lines, element.FontSize)
	if err != nil {
		return err
	}
	if element.TextAnchorCenter {
		width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
		if err != nil {
			return err
		}
		bounds = anchorCenteredTextBounds(bounds, width)
	}
	y := anchoredTextTop(bounds, measuredTextAnchorHeight(measured, element.TextAnchor), element.TextAnchor)
	verticalLimit := bounds.Max.Y
	if shapeTextVerticalOverflowAllowed(element) {
		verticalLimit = img.Bounds().Max.Y
	}
	for _, line := range lines {
		if len(measured) == 0 {
			return nil
		}
		current := measured[0]
		measured = measured[1:]
		y += current.SpaceBefore
		baseline := y + current.Ascent
		if y > verticalLimit {
			return nil
		}
		textAlign := line.TextAlign
		if textAlign == "" {
			textAlign = element.TextAlign
		}
		if len(line.Segments) > 0 {
			lineBounds := textLineBounds(bounds, line)
			x, err := alignedSegmentedTextXAtDPI(faces, face, boldFace, line.Segments, lineBounds, textAlign, dpi, line.TabStops)
			if err != nil {
				return err
			}
			if line.HasXOffset && (textAlign == "" || textAlign == "l") {
				x = bounds.Min.X + line.XOffset
			}
			justifyExtra := 0
			justifyRemainder := 0
			if line.Justify && textAlign == "just" {
				lineWidth, err := measureStyledSegmentsAtDPI(faces, face, boldFace, line.Segments, dpi, line.TabStops)
				if err != nil {
					return err
				}
				if spaceCount := textLineSpaceCount(line.Segments); spaceCount > 0 && lineWidth < lineBounds.Dx() {
					extra := lineBounds.Dx() - lineWidth
					justifyExtra = extra / spaceCount
					justifyRemainder = extra % spaceCount
				}
			}
			lineStart := x
			for _, segment := range line.Segments {
				fontSize := segment.FontSize
				if fontSize == 0 {
					fontSize = element.FontSize
				}
				segmentFace, err := faces.GetForFamily(segment.FontFamily, fontSize, segment.Bold, segment.Italic)
				if err != nil {
					return err
				}
				segmentFace = faceWithSegmentKerning(segmentFace, segment)
				if segment.HasTextColor {
					drawer.Src = image.NewUniform(segment.TextColor)
				} else {
					drawer.Src = image.NewUniform(textColor)
				}
				if segment.Marker != "" {
					markerColor := textColor
					if segment.HasTextColor {
						markerColor = segment.TextColor
					}
					drawTextMarker(img, segment.Marker, x, baseline-current.Ascent/2, markerPixelSizeAtDPI(segment, element.FontSize, dpi), markerColor)
					x += markerSegmentWidthAtDPI(segment, element.FontSize, dpi)
					continue
				}
				drawer.Face = segmentFace
				segmentWidth := measureTextSegmentWithTabsAndSpacingAtDPI(segmentFace, segment.Text, 0, dpi, line.TabStops, segment.CharSpacing)
				if justifyExtra > 0 || justifyRemainder > 0 {
					segmentSpaces := textSpaceCount(segment.Text)
					segmentWidth += segmentSpaces * justifyExtra
					segmentWidth += minInt(segmentSpaces, justifyRemainder)
				}
				segmentBaseline := baseline - segmentBaselineShiftAtDPI(segment, element.FontSize, dpi)
				if segment.HasHighlightColor {
					drawTextHighlight(img, segmentFace, x, segmentBaseline, segmentWidth, segment.HighlightColor)
				}
				segmentStart := x
				if justifyExtra > 0 || justifyRemainder > 0 {
					x = drawJustifiedTextSegment(drawer, segmentFace, segment.Text, x, segmentBaseline, justifyExtra, &justifyRemainder, segment.CharSpacing, dpi)
				} else {
					x = drawTextSegmentWithTabsAndSpacingAtDPI(drawer, segmentFace, segment.Text, x, lineStart, segmentBaseline, dpi, line.TabStops, segment.CharSpacing)
				}
				if segment.Underline {
					drawTextUnderline(img, segmentFace, segmentStart, segmentBaseline, x-segmentStart, underlineColorForSegment(segment, textColor))
				}
				if segment.Strike != "" {
					drawTextStrikethrough(img, segmentFace, segmentStart, segmentBaseline, x-segmentStart, segment.Strike, textColorForSegment(segment, textColor))
				}
			}
		} else {
			drawer.Src = image.NewUniform(textColor)
			drawer.Face = current.Face
			drawer.Dot = fixed.Point26_6{X: fixed.I(alignedTextX(current.Face, line.Text, textLineBounds(bounds, line), textAlign)), Y: fixed.I(baseline)}
			drawer.DrawString(line.Text)
		}
		y += current.Height
	}
	return nil
}

func shouldFitNormalAutofit(element slideElement) bool {
	return element.HasNormAutofit
}

func drawTextHighlight(img *image.RGBA, face font.Face, x int, baseline int, width int, c color.RGBA) {
	if width <= 0 {
		return
	}
	metrics := face.Metrics()
	top := baseline - metrics.Ascent.Ceil()
	bottom := baseline + metrics.Descent.Ceil()
	if bottom <= top {
		bottom = top + metrics.Height.Ceil()
	}
	rect := image.Rect(x, top, x+width, bottom).Intersect(img.Bounds())
	if rect.Empty() {
		return
	}
	draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
}

func drawTextUnderline(img *image.RGBA, face font.Face, x int, baseline int, width int, c color.RGBA) {
	if width <= 0 || c.A == 0 {
		return
	}
	metrics := face.Metrics()
	y := baseline + maxInt(1, metrics.Descent.Ceil()/3)
	lineWidth := maxInt(1, metrics.Height.Ceil()/16)
	drawLine(img, x, y, x+width-1, y, c, lineWidth)
}

func drawTextStrikethrough(img *image.RGBA, face font.Face, x int, baseline int, width int, strike string, c color.RGBA) {
	if width <= 0 || c.A == 0 || strike == "" {
		return
	}
	metrics := face.Metrics()
	top := baseline - metrics.Ascent.Ceil()
	bottom := baseline + metrics.Descent.Ceil()
	if bottom <= top {
		bottom = top + metrics.Height.Ceil()
	}
	lineWidth := maxInt(1, metrics.Height.Ceil()/16)
	center := top + (bottom-top)/2
	if strike == "dblStrike" {
		gap := maxInt(2, lineWidth*2)
		drawLine(img, x, center-gap/2, x+width-1, center-gap/2, c, lineWidth)
		drawLine(img, x, center+gap/2, x+width-1, center+gap/2, c, lineWidth)
		return
	}
	drawLine(img, x, center, x+width-1, center, c, lineWidth)
}

func textColorForSegment(segment textLineSegment, fallback color.RGBA) color.RGBA {
	if segment.HasTextColor {
		return segment.TextColor
	}
	return fallback
}

func underlineColorForSegment(segment textLineSegment, fallback color.RGBA) color.RGBA {
	if segment.HasUnderlineColor {
		return segment.UnderlineColor
	}
	return textColorForSegment(segment, fallback)
}

func fitNormalAutofitElement(element slideElement, bounds image.Rectangle, dpiOverride ...int) slideElement {
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return element
	}
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	startScale := 100000
	if element.FontScalePct > 0 && element.FontScalePct < startScale {
		startScale = element.FontScalePct
	}
	bestScale := startScale
	maxLines := normalAutofitMaxSoftLines(element)
	if scale, ok := largestFittingNormalAutofitScale(element, bounds, startScale, maxLines, dpi); ok {
		bestScale = scale
	}
	if bestScale != element.FontScalePct {
		element.FontScalePct = bestScale
	}
	return element
}

func largestFittingNormalAutofitScale(element slideElement, bounds image.Rectangle, startScale int, maxLines int, dpi int) (int, bool) {
	if startScale < minimumNormalAutofitFontScalePct {
		startScale = minimumNormalAutofitFontScalePct
	}
	if textFitsAtScale(element, bounds, startScale, maxLines, dpi) {
		return startScale, true
	}
	if !textFitsAtScale(element, bounds, minimumNormalAutofitFontScalePct, maxLines, dpi) {
		return 0, false
	}
	low := minimumNormalAutofitFontScalePct
	high := startScale - 1
	best := low
	for low <= high {
		mid := low + (high-low)/2
		if textFitsAtScale(element, bounds, mid, maxLines, dpi) {
			best = mid
			low = mid + 1
			continue
		}
		high = mid - 1
	}
	return best, true
}

const (
	minimumNormalAutofitFontScalePct = 1000
)

func normalAutofitHasAuthoredScale(element slideElement) bool {
	return element.HasNormAutofit && (element.HasFontScalePct || element.HasLineSpacingReductionPct)
}

func normalAutofitMaxSoftLines(element slideElement) int {
	if len(element.TextParagraphs) != 1 {
		return 0
	}
	if element.TextWrap != "none" {
		count := explicitTextLineCount(element)
		if count > 1 {
			return count
		}
		return 0
	}
	count := explicitTextLineCount(element)
	if count < 1 {
		return 1
	}
	return count
}

func explicitTextLineCount(element slideElement) int {
	if len(element.TextParagraphs) != 1 {
		return 0
	}
	count := 1
	if strings.Contains(element.Text, "\n") {
		count = maxInt(count, strings.Count(element.Text, "\n")+1)
	}
	if strings.Contains(element.TextParagraphs[0].Text, "\n") {
		count = maxInt(count, strings.Count(element.TextParagraphs[0].Text, "\n")+1)
	}
	for _, run := range element.TextParagraphs[0].Runs {
		if strings.Contains(run.Text, "\n") {
			count = maxInt(count, strings.Count(run.Text, "\n")+1)
		}
	}
	return count
}

func drawTextSegmentWithTabs(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int) int {
	return drawTextSegmentWithTabsAtDPI(drawer, face, text, x, lineStart, baseline, defaultOutputDPI, nil)
}

func drawTextSegmentWithTabsAtDPI(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int, dpi int, tabStops []int) int {
	return drawTextSegmentWithTabsAndSpacingAtDPI(drawer, face, text, x, lineStart, baseline, dpi, tabStops, 0)
}

func drawTextSegmentWithTabsAndSpacingAtDPI(drawer *font.Drawer, face font.Face, text string, x int, lineStart int, baseline int, dpi int, tabStops []int, charSpacing int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	if !strings.Contains(text, "\t") {
		return drawTextWithCharacterSpacing(drawer, face, text, x, baseline, spacingPixels)
	}
	parts := strings.Split(text, "\t")
	for index, part := range parts {
		if part != "" {
			x = drawTextWithCharacterSpacing(drawer, face, part, x, baseline, spacingPixels)
		}
		if index < len(parts)-1 {
			x += textTabAdvanceAtDPI(x-lineStart, dpi, tabStops)
		}
	}
	return x
}

func drawTextWithCharacterSpacing(drawer *font.Drawer, face font.Face, text string, x int, baseline int, spacingPixels int) int {
	if spacingPixels == 0 || utf8.RuneCountInString(text) <= 1 {
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(text)
		return x + measureString(face, text)
	}
	runes := []rune(text)
	for index, value := range runes {
		chunk := string(value)
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(chunk)
		x += measureString(face, chunk)
		if index < len(runes)-1 {
			x += spacingPixels
		}
	}
	return x
}

func drawJustifiedTextSegment(drawer *font.Drawer, face font.Face, text string, x int, baseline int, extraPerSpace int, extraRemainder *int, charSpacing int, dpi int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	runes := []rune(text)
	for index, value := range runes {
		chunk := string(value)
		drawer.Dot = fixed.Point26_6{X: fixed.I(x), Y: fixed.I(baseline)}
		drawer.DrawString(chunk)
		x += measureString(face, chunk)
		if index < len(runes)-1 {
			x += spacingPixels
		}
		if value == ' ' {
			x += extraPerSpace
			if extraRemainder != nil && *extraRemainder > 0 {
				x++
				*extraRemainder--
			}
		}
	}
	return x
}

func textHeightFitsAtScale(element slideElement, bounds image.Rectangle, scale int, maxLines int, dpi int) bool {
	candidate := element
	candidate.FontScalePct = scale
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return false
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return false
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return false
	}
	if maxLines > 0 && len(lines) > maxLines {
		return false
	}
	measured, err := measureTextRenderLines(faces, lines, candidate.FontSize)
	if err != nil {
		return false
	}
	return measuredTextHeight(measured) <= bounds.Dy()
}

func textFitsAtScale(element slideElement, bounds image.Rectangle, scale int, maxLines int, dpi int) bool {
	candidate := element
	candidate.FontScalePct = scale
	candidate = scaledTextElement(candidate, dpi)
	faces := newFontFaceCacheWithDPI(candidate.Italic, candidate.FontFamily, dpi, candidate.FontPointScale)
	defer faces.Close()
	face, err := faces.Get(candidate.FontSize, false)
	if err != nil {
		return false
	}
	boldFace, err := faces.Get(candidate.FontSize, true)
	if err != nil {
		return false
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, candidate, bounds.Dx(), dpi)
	if err != nil {
		return false
	}
	if maxLines > 0 && len(lines) > maxLines {
		return false
	}
	measured, err := measureTextRenderLines(faces, lines, candidate.FontSize)
	if err != nil {
		return false
	}
	if measuredTextHeight(measured) > bounds.Dy() {
		return false
	}
	width, err := measuredTextRenderLinesWidth(faces, face, boldFace, lines, dpi)
	if err != nil {
		return false
	}
	return width <= bounds.Dx()
}

func scaledTextElement(element slideElement, dpiOverride ...int) slideElement {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	applyFontScale := element.FontScalePct > 0 && element.FontScalePct != 100000
	applyLineSpacingReduction := element.LineSpacingReductionPct > 0
	scaleForDPI := dpi != defaultOutputDPI
	if !scaleForDPI && !applyFontScale && !applyLineSpacingReduction {
		return element
	}
	element.TextParagraphs = cloneTextParagraphs(element.TextParagraphs)
	if scaleForDPI {
		for paragraphIndex := range element.TextParagraphs {
			element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceBefore, dpi)
			element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceAfter, dpi)
		}
	}
	if applyLineSpacingReduction {
		for paragraphIndex := range element.TextParagraphs {
			element.TextParagraphs[paragraphIndex].LineSpacingPct = reducedLineSpacing(element.TextParagraphs[paragraphIndex].LineSpacingPct, element.LineSpacingReductionPct)
		}
	}
	if !applyFontScale {
		return element
	}
	element.FontSize = scaledFontSize(element.FontSize, element.FontScalePct)
	for paragraphIndex := range element.TextParagraphs {
		element.TextParagraphs[paragraphIndex].FontSize = scaledFontSize(element.TextParagraphs[paragraphIndex].FontSize, element.FontScalePct)
		element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixels(element.TextParagraphs[paragraphIndex].SpaceBefore, element.FontScalePct)
		element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixels(element.TextParagraphs[paragraphIndex].SpaceAfter, element.FontScalePct)
		for runIndex := range element.TextParagraphs[paragraphIndex].Runs {
			element.TextParagraphs[paragraphIndex].Runs[runIndex].FontSize = scaledFontSize(element.TextParagraphs[paragraphIndex].Runs[runIndex].FontSize, element.FontScalePct)
		}
	}
	return element
}

func cloneTextParagraphs(paragraphs []textParagraph) []textParagraph {
	if len(paragraphs) == 0 {
		return nil
	}
	cloned := make([]textParagraph, len(paragraphs))
	copy(cloned, paragraphs)
	for index := range cloned {
		if len(cloned[index].Runs) > 0 {
			cloned[index].Runs = append([]textRun(nil), cloned[index].Runs...)
		}
		if len(cloned[index].TabStops) > 0 {
			cloned[index].TabStops = append([]int64(nil), cloned[index].TabStops...)
		}
	}
	return cloned
}

func scaleParagraphSpacingForDPI(element slideElement, dpi int) slideElement {
	element.TextParagraphs = cloneTextParagraphs(element.TextParagraphs)
	for paragraphIndex := range element.TextParagraphs {
		element.TextParagraphs[paragraphIndex].SpaceBefore = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceBefore, dpi)
		element.TextParagraphs[paragraphIndex].SpaceAfter = scalePixelsForDPI(element.TextParagraphs[paragraphIndex].SpaceAfter, dpi)
	}
	return element
}

func scalePixelsForDPI(value int, dpi int) int {
	if value == 0 || dpi == defaultOutputDPI {
		return value
	}
	return int(math.Round(float64(value) * float64(dpi) / defaultOutputDPI))
}

func scalePixels(value int, scalePct int) int {
	if value == 0 || scalePct <= 0 || scalePct == 100000 {
		return value
	}
	return int(math.Round(float64(value) * float64(scalePct) / 100000))
}

func scaledFontSize(fontSize int, scalePct int) int {
	if fontSize <= 0 || scalePct <= 0 || scalePct == 100000 {
		return fontSize
	}
	scaled := int(math.Round(float64(fontSize) * float64(scalePct) / 100000))
	if scaled < 100 {
		return 100
	}
	return scaled
}

func reducedLineSpacing(lineSpacingPct int, reductionPct int) int {
	if reductionPct <= 0 || lineSpacingPct <= 0 {
		return lineSpacingPct
	}
	reduced := lineSpacingPct - reductionPct
	if reduced < 1000 {
		return 1000
	}
	return reduced
}

func alignedSegmentedTextX(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, bounds image.Rectangle, align string) (int, error) {
	return alignedSegmentedTextXAtDPI(faces, face, boldFace, segments, bounds, align, defaultOutputDPI, nil)
}

func alignedSegmentedTextXAtDPI(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, bounds image.Rectangle, align string, dpi int, tabStops []int) (int, error) {
	width, err := measureStyledSegmentsAtDPI(faces, face, boldFace, segments, dpi, tabStops)
	if err != nil {
		return 0, err
	}
	switch align {
	case "ctr":
		return bounds.Min.X + (bounds.Dx()-width)/2, nil
	case "r":
		return bounds.Max.X - width, nil
	default:
		return bounds.Min.X, nil
	}
}

func textRenderLinesForElement(faces *fontFaceCache, face font.Face, boldFace font.Face, element slideElement, maxWidth int, dpiOverride ...int) ([]textRenderLine, error) {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	if len(element.TextParagraphs) == 0 {
		lines := textLayoutPlainLines(face, element.Text, maxWidth, element.TextWrap)
		output := make([]textRenderLine, 0, len(lines))
		for _, line := range lines {
			output = append(output, textRenderLine{Text: line})
		}
		return output, nil
	}
	var output []textRenderLine
	pendingSpaceAfter := 0
	pendingSpaceAfterPct := 0
	for _, paragraph := range element.TextParagraphs {
		var paragraphLines []textRenderLine
		if len(paragraph.Runs) > 0 {
			lines, err := textRenderLinesForStyledParagraph(faces, face, boldFace, paragraph, maxWidth, element.TextWrap, dpi)
			if err != nil {
				return nil, err
			}
			paragraphLines = lines
		} else {
			paragraphFace := face
			if paragraph.FontSize != 0 {
				var err error
				paragraphFace, err = faces.Get(paragraph.FontSize, paragraph.Bold, paragraph.Italic)
				if err != nil {
					return nil, err
				}
			}
			for index, line := range layoutParagraphLines(paragraphFace, paragraph, maxWidth, element.TextWrap) {
				renderLine := textRenderLine{Text: line, Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize, TextAlign: paragraph.TextAlign, LineSpacingPct: paragraph.LineSpacingPct, TabStops: paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)}
				if index == 0 {
					renderLine.SpaceBefore = paragraph.SpaceBefore
					renderLine.SpaceBeforePct = paragraph.SpaceBeforePct
				}
				paragraphLines = append(paragraphLines, renderLine)
			}
		}
		if len(paragraphLines) == 0 {
			continue
		}
		if len(output) == 0 && !element.IncludeFirstLastSpacing {
			paragraphLines[0].SpaceBefore = 0
			paragraphLines[0].SpaceBeforePct = 0
		} else if len(output) > 0 {
			paragraphLines[0].SpaceBefore += pendingSpaceAfter
			paragraphLines[0].SpaceBeforePct += pendingSpaceAfterPct
		}
		output = append(output, paragraphLines...)
		pendingSpaceAfter = paragraph.SpaceAfter
		pendingSpaceAfterPct = paragraph.SpaceAfterPct
	}
	if len(output) > 0 && element.IncludeFirstLastSpacing {
		output[len(output)-1].SpaceAfter = pendingSpaceAfter
		output[len(output)-1].SpaceAfterPct = pendingSpaceAfterPct
	}
	return output, nil
}

func textRenderLinesForStyledParagraph(faces *fontFaceCache, face font.Face, boldFace font.Face, paragraph textParagraph, maxWidth int, wrap string, dpi int) ([]textRenderLine, error) {
	prefix, hangingPrefix := paragraphPrefixes(paragraph)
	firstOffset, hangingOffset, hasOffset := paragraphPixelOffsetsAtDPI(paragraph, dpi)
	rightOffset := paragraphRightOffsetAtDPI(paragraph, dpi)
	tabStops := paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)
	var output []textRenderLine
	chunks := splitRunsOnBreaks(paragraph.Runs)
	for index, chunk := range chunks {
		linePrefix := prefix
		lineOffset := firstOffset
		lineTabStops := tabStops
		if index > 0 {
			linePrefix = hangingPrefix
			lineOffset = hangingOffset
		} else if tabStop, ok := hangingBulletTabStop(paragraph, firstOffset, hangingOffset); ok {
			linePrefix = paragraph.Bullet + "\t"
			lineTabStops = withAdditionalTabStop(tabStops, tabStop)
		}
		if wrap == "none" {
			output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(linePrefix, paragraph, runsToSegments(chunk, paragraph)), lineTabStops), lineOffset, rightOffset, hasOffset))
			continue
		}
		if runsContainTabs(chunk) {
			output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(linePrefix, paragraph, runsToSegments(chunk, paragraph)), lineTabStops), lineOffset, rightOffset, hasOffset))
			continue
		}
		lines, err := wrapStyledRuns(faces, face, boldFace, chunk, paragraph, linePrefix, hangingPrefix, maxWidth, firstOffset, hangingOffset, rightOffset, hasOffset, dpi, lineTabStops)
		if err != nil {
			return nil, err
		}
		output = append(output, lines...)
	}
	if len(output) > 0 {
		output[0].SpaceBefore = paragraph.SpaceBefore
		output[0].SpaceBeforePct = paragraph.SpaceBeforePct
		for index := range output {
			output[index].TextAlign = paragraph.TextAlign
			output[index].LineSpacingPct = paragraph.LineSpacingPct
		}
	}
	return output, nil
}

func hangingBulletTabStop(paragraph textParagraph, firstOffset int, hangingOffset int) (int, bool) {
	if !paragraph.HasAutoNumber || paragraph.Bullet == "" || hangingOffset <= firstOffset {
		return 0, false
	}
	if !paragraph.HasMarginLeft && !paragraph.HasIndent {
		return 0, false
	}
	return hangingOffset - firstOffset, true
}

func withAdditionalTabStop(stops []int, stop int) []int {
	if stop <= 0 {
		return stops
	}
	for _, existing := range stops {
		if existing == stop {
			return stops
		}
	}
	output := append(append([]int{}, stops...), stop)
	sort.Ints(output)
	return output
}

func paragraphPixelOffsets(paragraph textParagraph) (int, int, bool) {
	return paragraphPixelOffsetsAtDPI(paragraph, defaultOutputDPI)
}

func paragraphPixelOffsetsAtDPI(paragraph textParagraph, dpi int) (int, int, bool) {
	if !paragraph.HasMarginLeft && !paragraph.HasIndent {
		return 0, 0, false
	}
	margin := paragraph.MarginLeft
	indent := paragraph.Indent
	first := emuToPixelsAtDPI(margin+indent, dpi)
	hanging := emuToPixelsAtDPI(margin, dpi)
	if first < 0 {
		first = 0
	}
	if hanging < 0 {
		hanging = 0
	}
	return first, hanging, true
}

func paragraphRightOffsetAtDPI(paragraph textParagraph, dpi int) int {
	if !paragraph.HasMarginRight {
		return 0
	}
	right := emuToPixelsAtDPI(paragraph.MarginRight, dpi)
	if right < 0 {
		return 0
	}
	return right
}

func tabStopsAtDPI(stops []int64, dpi int) []int {
	if len(stops) == 0 {
		return nil
	}
	output := make([]int, 0, len(stops))
	for _, stop := range stops {
		if stop <= 0 {
			continue
		}
		output = append(output, emuToPixelsAtDPI(stop, dpi))
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func paragraphTabStopsAtDPI(paragraph textParagraph, dpi int, maxWidth int) []int {
	stops := tabStopsAtDPI(paragraph.TabStops, dpi)
	if !paragraph.HasDefaultTab || paragraph.DefaultTabSize <= 0 {
		return stops
	}
	defaultTab := emuToPixelsAtDPI(paragraph.DefaultTabSize, dpi)
	if defaultTab <= 0 {
		return stops
	}
	limit := maxWidth * 2
	if limit < defaultTab {
		limit = defaultTab
	}
	generated := make([]int, 0, len(stops)+limit/defaultTab)
	generated = append(generated, stops...)
	for stop := defaultTab; stop <= limit; stop += defaultTab {
		generated = append(generated, stop)
	}
	if len(generated) == 0 {
		return nil
	}
	sort.Ints(generated)
	output := generated[:0]
	previous := -1
	for _, stop := range generated {
		if stop <= 0 || stop == previous {
			continue
		}
		output = append(output, stop)
		previous = stop
	}
	if len(output) == 0 {
		return nil
	}
	return output
}

func splitRunsOnBreaks(runs []textRun) [][]textRun {
	var chunks [][]textRun
	var current []textRun
	for _, run := range runs {
		parts := strings.Split(run.Text, "\n")
		for index, part := range parts {
			if index > 0 {
				chunks = append(chunks, current)
				current = nil
			}
			if part == "" {
				continue
			}
			next := run
			next.Text = part
			current = append(current, next)
		}
	}
	if len(current) > 0 || len(chunks) == 0 {
		chunks = append(chunks, current)
	}
	return chunks
}

func runsContainTabs(runs []textRun) bool {
	for _, run := range runs {
		if strings.Contains(run.Text, "\t") {
			return true
		}
	}
	return false
}

func runsToSegments(runs []textRun, paragraph textParagraph) []textLineSegment {
	segments := make([]textLineSegment, 0, len(runs))
	for _, run := range runs {
		segments = append(segments, runToSegment(run, paragraph))
	}
	return segments
}

func runToSegment(run textRun, paragraph textParagraph) textLineSegment {
	fontSize := run.FontSize
	if fontSize == 0 {
		fontSize = paragraph.FontSize
	}
	baselineFontSize := fontSize
	if run.Baseline != 0 {
		fontSize = scaledBaselineRunFontSize(fontSize)
	}
	segment := textLineSegment{
		Text:              run.Text,
		FontFamily:        paragraphFontFamily(run, paragraph),
		Bold:              resolvedRunBold(run, paragraph),
		Italic:            resolvedRunItalic(run, paragraph),
		Underline:         run.Underline,
		HasUnderlineColor: run.HasUnderlineColor,
		UnderlineColor:    run.UnderlineColor,
		Strike:            run.Strike,
		FontSize:          fontSize,
		Baseline:          run.Baseline,
		BaselineFontSize:  baselineFontSize,
		CharSpacing:       resolvedRunCharSpacing(run, paragraph),
		HasKern:           run.HasKern,
		KernMinFontSize:   run.KernMinFontSize,
		HasTextColor:      run.HasTextColor,
		TextColor:         run.TextColor,
		HasHighlightColor: run.HasHighlightColor,
		HighlightColor:    run.HighlightColor,
	}
	if !segment.HasTextColor && paragraph.HasTextColor {
		segment.HasTextColor = true
		segment.TextColor = paragraph.TextColor
	}
	return segment
}

func resolvedRunBold(run textRun, paragraph textParagraph) bool {
	if run.HasBold {
		return run.Bold
	}
	if run.Bold {
		return true
	}
	return paragraph.Bold
}

func resolvedRunItalic(run textRun, paragraph textParagraph) bool {
	if run.HasItalic {
		return run.Italic
	}
	if run.Italic {
		return true
	}
	return paragraph.Italic
}

func resolvedRunCharSpacing(run textRun, paragraph textParagraph) int {
	if run.HasCharSpacing {
		return run.CharSpacing
	}
	if paragraph.HasCharSpacing {
		return paragraph.CharSpacing
	}
	return 0
}

func paragraphFontFamily(run textRun, paragraph textParagraph) string {
	if strings.TrimSpace(run.FontFamily) != "" {
		return run.FontFamily
	}
	return paragraph.FontFamily
}

func scaledBaselineRunFontSize(fontSize int) int {
	if fontSize <= 0 {
		return fontSize
	}
	scaled := int(math.Round(float64(fontSize) * 0.65))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func appendPrefixSegment(prefix string, paragraph textParagraph, segments []textLineSegment) []textLineSegment {
	if prefix == "" {
		return segments
	}
	if paragraph.Bullet == "▶" && strings.HasPrefix(prefix, paragraph.Bullet) {
		bulletSegment := textLineSegment{Marker: "triangle", FontFamily: bulletSegmentFontFamily(paragraph), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
		if paragraph.HasBulletColor {
			bulletSegment.HasTextColor = true
			bulletSegment.TextColor = paragraph.BulletColor
		} else if paragraph.BulletColorTx {
			if textColor, ok := bulletTextColor(paragraph, segments); ok {
				bulletSegment.HasTextColor = true
				bulletSegment.TextColor = textColor
			}
		}
		spacer := textLineSegment{Text: strings.TrimPrefix(prefix, paragraph.Bullet), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize}
		output := make([]textLineSegment, 0, len(segments)+2)
		output = append(output, bulletSegment, spacer)
		output = append(output, segments...)
		return output
	}
	if paragraph.Bullet != "" && strings.HasPrefix(prefix, paragraph.Bullet) {
		bulletSegment := textLineSegment{Text: paragraph.Bullet, FontFamily: bulletSegmentFontFamily(paragraph), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
		if paragraph.HasBulletColor {
			bulletSegment.HasTextColor = true
			bulletSegment.TextColor = paragraph.BulletColor
		} else if paragraph.BulletColorTx {
			if textColor, ok := bulletTextColor(paragraph, segments); ok {
				bulletSegment.HasTextColor = true
				bulletSegment.TextColor = textColor
			}
		}
		spacer := textLineSegment{Text: strings.TrimPrefix(prefix, paragraph.Bullet), Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: paragraph.FontSize}
		output := make([]textLineSegment, 0, len(segments)+2)
		output = append(output, bulletSegment, spacer)
		output = append(output, segments...)
		return output
	}
	prefixSegment := textLineSegment{Text: prefix, Bold: paragraph.Bold, Italic: paragraph.Italic, FontSize: bulletSegmentFontSize(paragraph)}
	output := make([]textLineSegment, 0, len(segments)+1)
	output = append(output, prefixSegment)
	output = append(output, segments...)
	return output
}

func bulletTextColor(paragraph textParagraph, segments []textLineSegment) (color.RGBA, bool) {
	for _, segment := range segments {
		if segment.HasTextColor {
			return segment.TextColor, true
		}
	}
	if paragraph.HasTextColor {
		return paragraph.TextColor, true
	}
	return color.RGBA{}, false
}

func bulletSegmentFontFamily(paragraph textParagraph) string {
	if paragraph.BulletFontTx {
		return paragraph.FontFamily
	}
	if symbolBulletRenderedAsUnicode(paragraph.Bullet, paragraph.BulletFontFamily) {
		return paragraph.FontFamily
	}
	if paragraph.BulletFontFamily != "" {
		return paragraph.BulletFontFamily
	}
	return paragraph.FontFamily
}

func symbolBulletRenderedAsUnicode(bullet string, fontFamily string) bool {
	font := strings.ToLower(strings.TrimSpace(fontFamily))
	if !strings.Contains(font, "wingdings") {
		return false
	}
	switch bullet {
	case "▪", "▶", "¬":
		return true
	default:
		return false
	}
}

func bulletSegmentFontSize(paragraph textParagraph) int {
	if paragraph.BulletFontSize > 0 {
		return paragraph.BulletFontSize
	}
	if paragraph.BulletSizePct > 0 && paragraph.FontSize > 0 {
		scaled := int(math.Round(float64(paragraph.FontSize) * float64(paragraph.BulletSizePct) / 100000))
		if scaled > 0 {
			return scaled
		}
	}
	return paragraph.FontSize
}

func textRenderLineFromSegments(segments []textLineSegment) textRenderLine {
	return textRenderLineFromSegmentsWithTabs(segments, nil)
}

func textRenderLineFromSegmentsWithTabs(segments []textLineSegment, tabStops []int) textRenderLine {
	var text strings.Builder
	for _, segment := range segments {
		text.WriteString(segment.Text)
	}
	return textRenderLine{Text: text.String(), Segments: segments, TabStops: tabStops}
}

func textRenderLineWithOffset(line textRenderLine, offset int, ok bool) textRenderLine {
	return textRenderLineWithOffsets(line, offset, 0, ok)
}

func textRenderLineWithOffsets(line textRenderLine, offset int, rightOffset int, ok bool) textRenderLine {
	if ok {
		line.XOffset = offset
		line.HasXOffset = true
	}
	line.RightOffset = rightOffset
	return line
}

func textLineBounds(bounds image.Rectangle, line textRenderLine) image.Rectangle {
	adjusted := bounds
	if line.HasXOffset {
		adjusted.Min.X += line.XOffset
	}
	if line.RightOffset > 0 {
		adjusted.Max.X -= line.RightOffset
	}
	if adjusted.Empty() {
		return bounds
	}
	return adjusted
}

func wrapStyledRuns(faces *fontFaceCache, face font.Face, boldFace font.Face, runs []textRun, paragraph textParagraph, firstPrefix string, hangingPrefix string, maxWidth int, firstOffset int, hangingOffset int, rightOffset int, hasOffset bool, dpi int, tabStopsOverride ...[]int) ([]textRenderLine, error) {
	fullLine := appendPrefixSegment(firstPrefix, paragraph, runsToSegments(runs, paragraph))
	tabStops := paragraphTabStopsAtDPI(paragraph, dpi, maxWidth)
	if len(tabStopsOverride) > 0 {
		tabStops = tabStopsOverride[0]
	}
	fullLineWidth, err := measureStyledSegmentsAtDPI(faces, face, boldFace, fullLine, dpi, tabStops)
	if err != nil {
		return nil, err
	}
	availableWidth := maxWidth
	if hasOffset {
		availableWidth -= firstOffset
	}
	availableWidth -= rightOffset
	if maxWidth <= 0 || fullLineWidth <= availableWidth {
		return []textRenderLine{textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(fullLine, tabStops), firstOffset, rightOffset, hasOffset)}, nil
	}

	tokens := styledWordTokens(runs, paragraph)
	if maxWidth <= 0 || len(tokens) == 0 {
		return []textRenderLine{textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(appendPrefixSegment(firstPrefix, paragraph, runsToSegments(runs, paragraph)), tabStops), firstOffset, rightOffset, hasOffset)}, nil
	}
	var output []textRenderLine
	line := appendPrefixSegment(firstPrefix, paragraph, nil)
	lineOffset := firstOffset
	hasWord := false
	for _, token := range tokens {
		candidateToken := token.segmentWithPrefix(hasWord)
		candidate := append(append([]textLineSegment{}, line...), candidateToken)
		width, err := measureStyledSegmentsAtDPI(faces, face, boldFace, candidate, dpi, tabStops)
		if err != nil {
			return nil, err
		}
		availableWidth := maxWidth
		if hasOffset {
			availableWidth -= lineOffset
		}
		availableWidth -= rightOffset
		if hasWord && width > availableWidth {
			wrappedLine := textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(line, tabStops), lineOffset, rightOffset, hasOffset)
			wrappedLine.Justify = paragraph.TextAlign == "just"
			output = append(output, wrappedLine)
			line = appendPrefixSegment(hangingPrefix, paragraph, []textLineSegment{token.segmentAtLineStart()})
			lineOffset = hangingOffset
			hasWord = true
			continue
		}
		line = candidate
		hasWord = true
	}
	if len(line) > 0 {
		output = append(output, textRenderLineWithOffsets(textRenderLineFromSegmentsWithTabs(line, tabStops), lineOffset, rightOffset, hasOffset))
	}
	return output, nil
}

func (token styledWordToken) segmentWithPrefix(hasPriorWord bool) textLineSegment {
	segment := token.Segment
	if hasPriorWord || token.PreserveLinePrefix {
		segment.Text = token.Prefix + segment.Text
	}
	return segment
}

func (token styledWordToken) segmentAtLineStart() textLineSegment {
	segment := token.Segment
	if token.PreserveLinePrefix {
		segment.Text = token.Prefix + segment.Text
	}
	return segment
}

func styledWordTokens(runs []textRun, paragraph textParagraph) []styledWordToken {
	var tokens []styledWordToken
	var pendingSpace strings.Builder
	seenToken := false
	for _, run := range runs {
		segment := runToSegment(run, paragraph)
		for _, part := range splitTextForWrapping(segment.Text) {
			if part.Space {
				pendingSpace.WriteString(part.Text)
				continue
			}
			tokenSegment := segment
			tokenSegment.Text = part.Text
			tokens = append(tokens, styledWordToken{
				Segment:            tokenSegment,
				Prefix:             pendingSpace.String(),
				PreserveLinePrefix: !seenToken,
			})
			pendingSpace.Reset()
			seenToken = true
		}
	}
	return tokens
}

func splitTextForWrapping(text string) []wrapTextPart {
	if text == "" {
		return nil
	}
	var parts []wrapTextPart
	start := 0
	inSpace := unicode.IsSpace(rune(text[0]))
	for index, value := range text {
		isSpace := unicode.IsSpace(value)
		if index > start && isSpace != inSpace {
			parts = append(parts, wrapTextPart{Text: text[start:index], Space: inSpace})
			start = index
			inSpace = isSpace
		}
	}
	parts = append(parts, wrapTextPart{Text: text[start:], Space: inSpace})
	return parts
}

func textLineSpaceCount(segments []textLineSegment) int {
	count := 0
	for _, segment := range segments {
		count += textSpaceCount(segment.Text)
	}
	return count
}

func textSpaceCount(text string) int {
	count := 0
	for _, value := range text {
		if value == ' ' {
			count++
		}
	}
	return count
}

func measureStyledSegments(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment) (int, error) {
	return measureStyledSegmentsAtDPI(faces, face, boldFace, segments, defaultOutputDPI)
}

func measureStyledSegmentsAtDPI(faces *fontFaceCache, face font.Face, boldFace font.Face, segments []textLineSegment, dpi int, tabStopsOverride ...[]int) (int, error) {
	var tabStops []int
	if len(tabStopsOverride) > 0 {
		tabStops = tabStopsOverride[0]
	}
	width := 0
	for _, segment := range segments {
		if segment.Marker != "" {
			width += markerSegmentWidthAtDPI(segment, 0, dpi)
			continue
		}
		segmentFace := face
		if segment.FontSize != 0 || segment.FontFamily != "" {
			var err error
			segmentFace, err = faces.GetForFamily(segment.FontFamily, segment.FontSize, segment.Bold, segment.Italic)
			if err != nil {
				return 0, err
			}
		} else if segment.Bold || segment.Italic {
			var err error
			segmentFace, err = faces.GetForFamily(segment.FontFamily, 0, segment.Bold, segment.Italic)
			if err != nil {
				return 0, err
			}
		}
		width = measureTextSegmentWithTabsAndSpacingAtDPI(faceWithSegmentKerning(segmentFace, segment), segment.Text, width, dpi, tabStops, segment.CharSpacing)
	}
	return width, nil
}

func (face noKerningFace) Kern(r0 rune, r1 rune) fixed.Int26_6 {
	return 0
}

func faceWithSegmentKerning(face font.Face, segment textLineSegment) font.Face {
	if face == nil || !segment.HasKern {
		return face
	}
	if segment.KernMinFontSize <= 0 {
		return face
	}
	if segment.FontSize > 0 && segment.FontSize < segment.KernMinFontSize {
		return noKerningFace{Face: face}
	}
	return face
}

func measureTextSegmentWithTabs(face font.Face, text string, currentWidth int) int {
	return measureTextSegmentWithTabsAtDPI(face, text, currentWidth, defaultOutputDPI, nil)
}

func measureTextSegmentWithTabsAtDPI(face font.Face, text string, currentWidth int, dpi int, tabStopsOverride ...[]int) int {
	return measureTextSegmentWithTabsAndSpacingAtDPI(face, text, currentWidth, dpi, firstTabStopsOverride(tabStopsOverride), 0)
}

func measureTextSegmentWithTabsAndSpacingAtDPI(face font.Face, text string, currentWidth int, dpi int, tabStops []int, charSpacing int) int {
	spacingPixels := textCharacterSpacingPixelsAtDPI(charSpacing, dpi)
	if !strings.Contains(text, "\t") {
		return currentWidth + measureString(face, text) + textCharacterSpacingAdvance(text, spacingPixels)
	}
	width := currentWidth
	parts := strings.Split(text, "\t")
	for index, part := range parts {
		width += measureString(face, part) + textCharacterSpacingAdvance(part, spacingPixels)
		if index < len(parts)-1 {
			width += textTabAdvanceAtDPI(width, dpi, tabStops)
		}
	}
	return width
}

func firstTabStopsOverride(overrides [][]int) []int {
	if len(overrides) == 0 {
		return nil
	}
	return overrides[0]
}

func textCharacterSpacingPixelsAtDPI(charSpacing int, dpi int) int {
	if charSpacing == 0 {
		return 0
	}
	return int(math.Round(float64(charSpacing) / 100 * float64(normalizeOutputDPI(dpi)) / 72))
}

func textCharacterSpacingAdvance(text string, spacingPixels int) int {
	if spacingPixels == 0 {
		return 0
	}
	count := utf8.RuneCountInString(text)
	if count <= 1 {
		return 0
	}
	return (count - 1) * spacingPixels
}

func textTabAdvance(currentWidth int) int {
	return textTabAdvanceAtDPI(currentWidth, defaultOutputDPI, nil)
}

func textTabAdvanceAtDPI(currentWidth int, dpi int, tabStops []int) int {
	for _, stop := range tabStops {
		if stop > currentWidth {
			return stop - currentWidth
		}
	}
	tabPixels := normalizeOutputDPI(dpi)
	remainder := currentWidth % tabPixels
	if remainder == 0 {
		return tabPixels
	}
	return tabPixels - remainder
}

func measuredTextRenderLinesWidth(faces *fontFaceCache, face font.Face, boldFace font.Face, lines []textRenderLine, dpiOverride ...int) (int, error) {
	dpi := defaultOutputDPI
	if len(dpiOverride) > 0 {
		dpi = normalizeOutputDPI(dpiOverride[0])
	}
	maxWidth := 0
	for _, line := range lines {
		width := 0
		if len(line.Segments) > 0 {
			var err error
			width, err = measureStyledSegmentsAtDPI(faces, face, boldFace, line.Segments, dpi, line.TabStops)
			if err != nil {
				return 0, err
			}
		} else {
			lineFace := face
			if line.FontSize != 0 {
				var err error
				lineFace, err = faces.Get(line.FontSize, line.Bold, line.Italic)
				if err != nil {
					return 0, err
				}
			} else if line.Bold || line.Italic {
				var err error
				lineFace, err = faces.Get(0, line.Bold, line.Italic)
				if err != nil {
					return 0, err
				}
			}
			width = measureString(lineFace, line.Text)
		}
		if line.HasXOffset {
			width += line.XOffset
		}
		width += line.RightOffset
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth, nil
}

func segmentBaselineShift(segment textLineSegment, fallbackFontSize int) int {
	return segmentBaselineShiftAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func segmentBaselineShiftAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	if segment.Baseline == 0 {
		return 0
	}
	fontSize := segment.FontSize
	if segment.BaselineFontSize > 0 {
		fontSize = segment.BaselineFontSize
	}
	if fontSize == 0 {
		fontSize = fallbackFontSize
	}
	if fontSize <= 0 {
		fontSize = 1800
	}
	points := float64(fontSize) / 100
	pixels := points * float64(normalizeOutputDPI(dpi)) / 72
	return int(math.Round(pixels * float64(segment.Baseline) / 100000))
}

func markerPixelSize(segment textLineSegment, fallbackFontSize int) int {
	return markerPixelSizeAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func markerPixelSizeAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	fontSize := segment.FontSize
	if fontSize == 0 {
		fontSize = fallbackFontSize
	}
	if fontSize <= 0 {
		fontSize = 1800
	}
	size := int(math.Round(float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72 * 0.45))
	if size < 6 {
		return 6
	}
	return size
}

func markerSegmentWidth(segment textLineSegment, fallbackFontSize int) int {
	return markerSegmentWidthAtDPI(segment, fallbackFontSize, defaultOutputDPI)
}

func markerSegmentWidthAtDPI(segment textLineSegment, fallbackFontSize int, dpi int) int {
	spacer := int(math.Round(float64(normalizeOutputDPI(dpi)) * 10 / defaultOutputDPI))
	if spacer < 1 {
		spacer = 1
	}
	return markerPixelSizeAtDPI(segment, fallbackFontSize, dpi) + spacer
}

func drawTextMarker(img *image.RGBA, marker string, x int, centerY int, size int, c color.RGBA) {
	switch marker {
	case "triangle":
		bounds := image.Rect(x, centerY-size/2, x+size, centerY+size/2)
		drawPolygon(img, bounds, []pathPoint{{X: 0, Y: 0}, {X: 1, Y: 0.5}, {X: 0, Y: 1}}, c)
	}
}

func newFontFaceCache(italic bool, fontFamily string, pointScale ...float64) *fontFaceCache {
	return newFontFaceCacheWithDPI(italic, fontFamily, defaultOutputDPI, pointScale...)
}

func newFontFaceCacheWithDPI(italic bool, fontFamily string, dpi int, pointScale ...float64) *fontFaceCache {
	scale := 0.0
	if len(pointScale) > 0 {
		scale = pointScale[0]
	}
	return &fontFaceCache{DefaultItalic: italic, FontFamily: fontFamily, PointScale: scale, DPI: normalizeOutputDPI(dpi), Faces: map[fontKey]font.Face{}}
}

func (cache *fontFaceCache) Get(fontSize int, bold bool, italicOverride ...bool) (font.Face, error) {
	return cache.GetForFamily("", fontSize, bold, italicOverride...)
}

func (cache *fontFaceCache) GetForFamily(fontFamily string, fontSize int, bold bool, italicOverride ...bool) (font.Face, error) {
	italic := cache.DefaultItalic
	if len(italicOverride) > 0 {
		italic = italicOverride[0]
	}
	resolvedFamily := strings.TrimSpace(fontFamily)
	if resolvedFamily == "" {
		resolvedFamily = cache.FontFamily
	}
	key := fontKey{FontSize: fontSize, Bold: bold, Italic: italic, FontFamily: resolvedFamily}
	if face, ok := cache.Faces[key]; ok {
		return face, nil
	}
	face, err := openFontFaceWithDPI(fontSize, bold, italic, cache.PointScale, resolvedFamily, cache.DPI)
	if err != nil {
		return nil, err
	}
	cache.Faces[key] = face
	return face, nil
}

func (cache *fontFaceCache) Close() {
	for _, face := range cache.Faces {
		face.Close()
	}
}

func measureTextRenderLines(faces *fontFaceCache, lines []textRenderLine, fallbackFontSize int) ([]measuredTextLine, error) {
	measured := make([]measuredTextLine, 0, len(lines))
	for _, line := range lines {
		if len(line.Segments) > 0 {
			current, err := measureSegmentedTextLine(faces, line.Segments, fallbackFontSize)
			if err != nil {
				return nil, err
			}
			current.SpaceBefore = line.SpaceBefore
			current.SpaceBeforePct = line.SpaceBeforePct
			current.SpaceAfter = line.SpaceAfter
			current.SpaceAfterPct = line.SpaceAfterPct
			current.LineSpacingPct = line.LineSpacingPct
			current.HasText = true
			fontSize := lineFontSize(line, fallbackFontSize)
			current.SpaceBefore += paragraphSpacingPercentPixelsAtDPI(current.SpaceBeforePct, fontSize, faces.DPI)
			current.SpaceAfter += paragraphSpacingPercentPixelsAtDPI(current.SpaceAfterPct, fontSize, faces.DPI)
			current.Height = visibleLineAdvance(applyLineSpacingAtDPI(current.Height, current.LineSpacingPct, fontSize, faces.DPI), current)
			measured = append(measured, current)
			continue
		}
		fontSize := line.FontSize
		if fontSize == 0 {
			fontSize = fallbackFontSize
		}
		face, err := faces.Get(fontSize, line.Bold, line.Italic)
		if err != nil {
			return nil, err
		}
		metrics := face.Metrics()
		height := defaultLineMetricHeight(metrics)
		measured = append(measured, measuredTextLine{
			Face:    face,
			Ascent:  metrics.Ascent.Ceil(),
			Descent: metrics.Descent.Ceil(),
			HasText: line.Text != "",
			Height: visibleLineAdvance(
				applyLineSpacingAtDPI(height, line.LineSpacingPct, fontSize, faces.DPI),
				measuredTextLine{Ascent: metrics.Ascent.Ceil(), Descent: metrics.Descent.Ceil()},
			),
			SpaceBefore:    line.SpaceBefore + paragraphSpacingPercentPixelsAtDPI(line.SpaceBeforePct, fontSize, faces.DPI),
			SpaceBeforePct: line.SpaceBeforePct,
			SpaceAfter:     line.SpaceAfter + paragraphSpacingPercentPixelsAtDPI(line.SpaceAfterPct, fontSize, faces.DPI),
			SpaceAfterPct:  line.SpaceAfterPct,
			LineSpacingPct: line.LineSpacingPct,
		})
	}
	return measured, nil
}

func applyLineSpacing(height int, pct int) int {
	return applyLineSpacingAtDPI(height, pct, 0, defaultOutputDPI)
}

func applyLineSpacingAtDPI(height int, pct int, fontSize int, dpi int) int {
	if pct <= 0 {
		return height
	}
	base := height
	if fontSize > 0 {
		base = int(math.Round(float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72))
		if base <= 0 {
			base = height
		}
	}
	scaled := int(math.Round(float64(base) * float64(pct) / 100000))
	if scaled < 1 {
		return 1
	}
	return scaled
}

func visibleLineAdvance(height int, line measuredTextLine) int {
	minimum := line.Ascent + line.Descent
	if minimum > height {
		return minimum
	}
	return height
}

func lineFontSize(line textRenderLine, fallbackFontSize int) int {
	size := line.FontSize
	for _, segment := range line.Segments {
		segmentSize := segment.FontSize
		if segmentSize == 0 {
			segmentSize = fallbackFontSize
		}
		if segmentSize > size {
			size = segmentSize
		}
	}
	if size <= 0 {
		return fallbackFontSize
	}
	return size
}

func measureSegmentedTextLine(faces *fontFaceCache, segments []textLineSegment, fallbackFontSize int) (measuredTextLine, error) {
	var measured measuredTextLine
	for _, segment := range segments {
		fontSize := segment.FontSize
		if fontSize == 0 {
			fontSize = fallbackFontSize
		}
		face, err := faces.GetForFamily(segment.FontFamily, fontSize, segment.Bold, segment.Italic)
		if err != nil {
			return measuredTextLine{}, err
		}
		metrics := face.Metrics()
		ascent := metrics.Ascent.Ceil()
		descent := metrics.Descent.Ceil()
		height := defaultLineMetricHeight(metrics)
		if ascent > measured.Ascent {
			measured.Ascent = ascent
		}
		if descent > measured.Descent {
			measured.Descent = descent
		}
		if height > measured.Height {
			measured.Height = height
		}
		if measured.Face == nil {
			measured.Face = face
		}
	}
	return measured, nil
}

func defaultLineMetricHeight(metrics font.Metrics) int {
	height := metrics.Height.Ceil()
	drawable := metrics.Ascent.Ceil() + metrics.Descent.Ceil()
	if height > 0 {
		return height
	}
	return drawable
}

func measuredTextHeight(lines []measuredTextLine) int {
	advanceHeight := 0
	inkHeight := 0
	for index, line := range lines {
		if index == 0 {
			inkHeight += line.SpaceBefore + line.Ascent
		} else {
			inkHeight += line.SpaceBefore + line.Height
		}
		advanceHeight += line.SpaceBefore + line.Height + line.SpaceAfter
	}
	if len(lines) > 0 {
		inkHeight += lines[len(lines)-1].Descent + lines[len(lines)-1].SpaceAfter
	}
	if inkHeight > advanceHeight {
		return inkHeight
	}
	return advanceHeight
}

func measuredTextAnchorHeight(lines []measuredTextLine, anchor string) int {
	if anchor != "ctr" && anchor != "b" {
		return measuredTextHeight(lines)
	}
	height := 0
	for _, line := range lines {
		visible := line.Ascent + line.Descent
		if !line.HasText {
			visible = line.Height
		}
		if visible <= 0 {
			visible = line.Height
		}
		height += line.SpaceBefore + visible + line.SpaceAfter
	}
	return height
}

func paragraphSpacingPercentPixels(pct int, fontSize int) int {
	return paragraphSpacingPercentPixelsAtDPI(pct, fontSize, defaultOutputDPI)
}

func paragraphSpacingPercentPixelsAtDPI(pct int, fontSize int, dpi int) int {
	if pct <= 0 || fontSize <= 0 {
		return 0
	}
	textPixels := float64(fontSize) / 100 * float64(normalizeOutputDPI(dpi)) / 72
	return int(math.Round(textPixels * float64(pct) / 100000))
}

func anchoredTextTop(bounds image.Rectangle, totalHeight int, anchor string) int {
	top := bounds.Min.Y
	switch anchor {
	case "ctr":
		top = bounds.Min.Y + (bounds.Dy()-totalHeight)/2
	case "b":
		top = bounds.Max.Y - totalHeight
	}
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	return top
}

func anchorCenteredTextBounds(bounds image.Rectangle, textWidth int) image.Rectangle {
	if textWidth <= 0 || textWidth >= bounds.Dx() {
		return bounds
	}
	left := bounds.Min.X + (bounds.Dx()-textWidth)/2
	return image.Rect(left, bounds.Min.Y, left+textWidth, bounds.Max.Y)
}

func textLayoutLines(face font.Face, text string, maxWidth int, wrap string) []string {
	return textLayoutParagraphLines(face, nil, text, maxWidth, wrap)
}

func textLayoutParagraphLines(face font.Face, paragraphs []textParagraph, fallbackText string, maxWidth int, wrap string) []string {
	lines := textLayoutStyledParagraphLines(face, face, paragraphs, fallbackText, maxWidth, wrap)
	output := make([]string, 0, len(lines))
	for _, line := range lines {
		output = append(output, line.Text)
	}
	return output
}

func textLayoutStyledParagraphLines(face font.Face, boldFace font.Face, paragraphs []textParagraph, fallbackText string, maxWidth int, wrap string) []textRenderLine {
	if len(paragraphs) == 0 {
		lines := textLayoutPlainLines(face, fallbackText, maxWidth, wrap)
		output := make([]textRenderLine, 0, len(lines))
		for _, line := range lines {
			output = append(output, textRenderLine{Text: line})
		}
		return output
	}
	var output []textRenderLine
	for _, paragraph := range paragraphs {
		paragraphFace := face
		if paragraph.Bold {
			paragraphFace = boldFace
		}
		for _, line := range layoutParagraphLines(paragraphFace, paragraph, maxWidth, wrap) {
			output = append(output, textRenderLine{Text: line, Bold: paragraph.Bold, FontSize: paragraph.FontSize, TextAlign: paragraph.TextAlign, LineSpacingPct: paragraph.LineSpacingPct})
		}
	}
	return output
}

func textLayoutPlainLines(face font.Face, text string, maxWidth int, wrap string) []string {
	var output []string
	for _, paragraph := range strings.Split(text, "\n") {
		if wrap == "none" {
			output = append(output, paragraph)
			continue
		}
		output = append(output, wrapText(face, paragraph, maxWidth)...)
	}
	return output
}

func layoutParagraphLines(face font.Face, paragraph textParagraph, maxWidth int, wrap string) []string {
	prefix, hangingPrefix := paragraphPrefixes(paragraph)
	var output []string
	for index, text := range strings.Split(paragraph.Text, "\n") {
		firstPrefix := prefix
		if index > 0 {
			firstPrefix = hangingPrefix
		}
		if wrap == "none" {
			output = append(output, firstPrefix+text)
			continue
		}
		output = append(output, wrapTextWithPrefixes(face, text, maxWidth, firstPrefix, hangingPrefix)...)
	}
	return output
}

func paragraphPrefixes(paragraph textParagraph) (string, string) {
	if paragraph.HasMarginLeft || paragraph.HasIndent {
		if paragraph.Bullet == "" {
			return "", ""
		}
		return paragraph.Bullet + " ", ""
	}
	indent := strings.Repeat("  ", paragraph.Level)
	if paragraph.Bullet == "" {
		return indent, indent
	}
	prefix := indent + paragraph.Bullet + " "
	hangingPrefix := strings.Repeat(" ", len([]rune(prefix)))
	return prefix, hangingPrefix
}

func anchoredTextStartY(bounds image.Rectangle, lineCount int, lineHeight int, ascent int, anchor string) int {
	if lineCount <= 0 {
		return bounds.Min.Y + ascent
	}
	totalHeight := lineCount * lineHeight
	top := bounds.Min.Y
	switch anchor {
	case "ctr":
		top = bounds.Min.Y + (bounds.Dy()-totalHeight)/2
	case "b":
		top = bounds.Max.Y - totalHeight
	}
	if top < bounds.Min.Y {
		top = bounds.Min.Y
	}
	return top + ascent
}

func wrapText(face font.Face, text string, maxWidth int) []string {
	return wrapTextWithPrefixes(face, text, maxWidth, "", "")
}

func wrapTextWithPrefixes(face font.Face, text string, maxWidth int, firstPrefix string, hangingPrefix string) []string {
	if maxWidth <= 0 || strings.TrimSpace(text) == "" {
		return []string{firstPrefix + text}
	}
	if measureString(face, firstPrefix+text) <= maxWidth {
		return []string{firstPrefix + text}
	}
	words := plainWordTokens(text)
	if len(words) == 0 {
		return []string{firstPrefix + text}
	}
	var lines []string
	current := words[0].textAtLineStart()
	for _, word := range words[1:] {
		candidate := current + word.textWithPrefix(true)
		prefix := firstPrefix
		if len(lines) > 0 {
			prefix = hangingPrefix
		}
		if measureString(face, prefix+candidate) <= maxWidth {
			current = candidate
			continue
		}
		prefix = firstPrefix
		if len(lines) > 0 {
			prefix = hangingPrefix
		}
		lines = append(lines, prefix+current)
		current = word.textAtLineStart()
	}
	prefix := firstPrefix
	if len(lines) > 0 {
		prefix = hangingPrefix
	}
	lines = append(lines, prefix+current)
	return lines
}

func (token plainWordToken) textWithPrefix(hasPriorWord bool) string {
	if hasPriorWord || token.PreserveLinePrefix {
		return token.Prefix + token.Text
	}
	return token.Text
}

func (token plainWordToken) textAtLineStart() string {
	if token.PreserveLinePrefix {
		return token.Prefix + token.Text
	}
	return token.Text
}

func plainWordTokens(text string) []plainWordToken {
	var tokens []plainWordToken
	var pendingSpace strings.Builder
	seenToken := false
	for _, part := range splitTextForWrapping(text) {
		if part.Space {
			pendingSpace.WriteString(part.Text)
			continue
		}
		tokens = append(tokens, plainWordToken{
			Text:               part.Text,
			Prefix:             pendingSpace.String(),
			PreserveLinePrefix: !seenToken,
		})
		pendingSpace.Reset()
		seenToken = true
	}
	return tokens
}

func alignedTextX(face font.Face, text string, bounds image.Rectangle, align string) int {
	width := measureString(face, text)
	switch align {
	case "ctr":
		return bounds.Min.X + (bounds.Dx()-width)/2
	case "r":
		return bounds.Max.X - width
	default:
		return bounds.Min.X
	}
}

func measureString(face font.Face, text string) int {
	return (&font.Drawer{Face: face}).MeasureString(text).Ceil()
}

func textBounds(bounds image.Rectangle, element slideElement, size slideSize, canvas image.Rectangle) image.Rectangle {
	if element.HasTextTransform && element.TextExtCX > 0 && element.TextExtCY > 0 {
		bounds = image.Rect(
			scaleEMU(element.TextOffX, size.CX, canvas.Dx()),
			scaleEMU(element.TextOffY, size.CY, canvas.Dy()),
			scaleEMU(element.TextOffX+element.TextExtCX, size.CX, canvas.Dx()),
			scaleEMU(element.TextOffY+element.TextExtCY, size.CY, canvas.Dy()),
		)
	}
	left := scaleEMU(defaultTextInsetXEMU, size.CX, canvas.Dx())
	top := scaleEMU(defaultTextInsetYEMU, size.CY, canvas.Dy())
	right := scaleEMU(defaultTextInsetXEMU, size.CX, canvas.Dx())
	bottom := scaleEMU(defaultTextInsetYEMU, size.CY, canvas.Dy())
	if element.HasInsets {
		left = scaleEMU(element.InsetLeft, size.CX, canvas.Dx())
		top = scaleEMU(element.InsetTop, size.CY, canvas.Dy())
		right = scaleEMU(element.InsetRight, size.CX, canvas.Dx())
		bottom = scaleEMU(element.InsetBottom, size.CY, canvas.Dy())
	}
	inset := image.Rect(bounds.Min.X+left, bounds.Min.Y+top, bounds.Max.X-right, bounds.Max.Y-bottom)
	if inset.Empty() {
		return bounds
	}
	return inset
}
