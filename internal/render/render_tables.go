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

func parseTableModel(tableNode *xmlNode, theme themeColors) tableModel {
	table := tableModel{UnsupportedFeatures: tableUnsupportedFeatureMessages(tableNode)}
	if properties := firstChild(tableNode, "tblPr"); properties != nil {
		table.FirstRow = attrValue(properties.Attrs, "firstRow") == "1"
		table.FirstCol = attrValue(properties.Attrs, "firstCol") == "1"
		table.LastRow = attrValue(properties.Attrs, "lastRow") == "1"
		table.LastCol = attrValue(properties.Attrs, "lastCol") == "1"
		table.BandRow = attrValue(properties.Attrs, "bandRow") == "1"
		table.BandCol = attrValue(properties.Attrs, "bandCol") == "1"
		if styleID := firstChild(properties, "tableStyleId"); styleID != nil {
			table.StyleID = strings.TrimSpace(styleID.Text)
		}
	}
	if grid := firstChild(tableNode, "tblGrid"); grid != nil {
		for _, gridCol := range childrenByName(grid, "gridCol") {
			if width := parseIntAttr(gridCol.Attrs, "w"); width > 0 {
				table.Columns = append(table.Columns, width)
			}
		}
	}
	for _, rowNode := range childrenByName(tableNode, "tr") {
		row := tableRow{}
		if attrValue(rowNode.Attrs, "h") != "" {
			row.HasHeight = true
			row.Height = parseIntAttr(rowNode.Attrs, "h")
		}
		for _, cellNode := range childrenByName(rowNode, "tc") {
			row.Cells = append(row.Cells, parseTableCell(cellNode, theme))
		}
		table.Rows = append(table.Rows, row)
	}
	return table
}

func tableUnsupportedFeatureMessages(tableNode *xmlNode) []string {
	messages := map[string]bool{}
	collectTableUnsupportedFeatureMessages(tableNode, messages)
	return sortedKeys(messages)
}

func collectTableUnsupportedFeatureMessages(node *xmlNode, messages map[string]bool) {
	switch node.Name {
	case "gradFill", "blipFill", "pattFill", "grpFill":
		messages["uses a non-solid cell fill that was not rendered"] = true
	case "effectDag":
		if len(node.Children) > 0 {
			messages["uses effects that were not rendered"] = true
		}
	case "effectLst":
		if effectListHasVisibleEffects(node) {
			messages["uses effects that were not rendered"] = true
		}
	case "ln", "lnL", "lnR", "lnT", "lnB", "left", "right", "top", "bottom", "insideH", "insideV":
		collectTableLineUnsupportedFeatureMessages(node, messages)
	}
	for _, child := range node.Children {
		collectTableUnsupportedFeatureMessages(child, messages)
	}
}

func effectListHasVisibleEffects(node *xmlNode) bool {
	for _, child := range node.Children {
		switch child.Name {
		case "blur", "fillOverlay", "glow", "innerShdw", "outerShdw", "prstShdw", "reflection", "softEdge":
			return true
		}
	}
	return false
}

func collectTableLineUnsupportedFeatureMessages(line *xmlNode, messages map[string]bool) {
	if cap := attrValue(line.Attrs, "cap"); cap != "" && cap != "flat" && cap != "sq" && cap != "rnd" {
		messages["uses border line caps that were not rendered"] = true
	}
	if cmpd := attrValue(line.Attrs, "cmpd"); !isSupportedTableCompoundLine(cmpd) {
		messages["uses compound border lines that were not rendered"] = true
	}
	if firstChild(line, "noFill") == nil && firstChild(line, "bevel") != nil {
		messages["uses border line joins that were not rendered"] = true
	}
	for _, name := range []string{"headEnd", "tailEnd"} {
		marker := firstChild(line, name)
		if marker == nil {
			continue
		}
		markerType := attrValue(marker.Attrs, "type")
		if markerType != "" && markerType != "none" {
			messages["uses border line end decorations that were not rendered"] = true
		}
	}
}

func parseTableCell(cellNode *xmlNode, theme themeColors) tableCell {
	cellElement := slideElement{}
	parseTextProperties(cellNode, &cellElement, theme)
	cell := tableCell{
		Text:           strings.TrimSpace(textFromNode(cellNode)),
		TextParagraphs: textParagraphsFromNodeWithTheme(cellNode, theme),
		ColSpan:        int(parseIntAttr(cellNode.Attrs, "gridSpan")),
		HMerge:         attrValue(cellNode.Attrs, "hMerge") == "1",
		RowSpan:        int(parseIntAttr(cellNode.Attrs, "rowSpan")),
		VMerge:         attrValue(cellNode.Attrs, "vMerge") == "1",
		FontSize:       cellElement.FontSize,
		HasTextColor:   cellElement.HasTextColor,
		TextColor:      cellElement.TextColor,
		TextAlign:      cellElement.TextAlign,
		TextAnchor:     cellElement.TextAnchor,
	}
	if cell.RowSpan <= 0 {
		cell.RowSpan = 1
	}
	if cell.ColSpan <= 0 {
		cell.ColSpan = 1
	}
	if cell.FontSize > 0 {
		cell.HasFontSize = true
	}
	if cell.FontSize == 0 {
		if size := textParagraphsFontSize(cell.TextParagraphs); size > 0 {
			cell.FontSize = size
			cell.HasFontSize = true
		}
	}
	if cell.FontSize == 0 {
		cell.FontSize = 1200
	}
	if cell.TextAlign == "" {
		cell.TextAlign = textParagraphsTextAlign(cell.TextParagraphs)
	}
	if textParagraphsHaveRunColor(cell.TextParagraphs) {
		cell.HasTextColor = false
		cell.TextColor = color.RGBA{}
	}
	if cellProperties := firstChild(cellNode, "tcPr"); cellProperties != nil {
		if anchor := attrValue(cellProperties.Attrs, "anchor"); anchor != "" {
			cell.TextAnchor = anchor
		}
		if margins, ok := parseTableCellMargins(cellProperties.Attrs); ok {
			cell.HasMargins = true
			cell.MarginLeft = margins.Left
			cell.MarginTop = margins.Top
			cell.MarginRight = margins.Right
			cell.MarginBottom = margins.Bottom
		}
		if firstChild(cellProperties, "noFill") != nil {
			cell.NoFill = true
		}
		if solidFill := firstChild(cellProperties, "solidFill"); solidFill != nil {
			if fill, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				cell.HasFill = true
				cell.FillColor = fill
			}
		}
		cell.BorderLeft = parseTableCellBorder(cellProperties, "lnL", theme)
		cell.BorderRight = parseTableCellBorder(cellProperties, "lnR", theme)
		cell.BorderTop = parseTableCellBorder(cellProperties, "lnT", theme)
		cell.BorderBottom = parseTableCellBorder(cellProperties, "lnB", theme)
	}
	return cell
}

func parseTableCellBorder(cellProperties *xmlNode, name string, theme themeColors) tableCellBorder {
	line := firstChild(cellProperties, name)
	if line == nil {
		return tableCellBorder{}
	}
	return parseTableLineNode(line, theme, true)
}

func parseTableLineNode(line *xmlNode, theme themeColors, specified bool) tableCellBorder {
	border := tableCellBorder{
		Specified: specified,
		Width:     parseIntAttr(line.Attrs, "w"),
		Cap:       attrValue(line.Attrs, "cap"),
		Align:     attrValue(line.Attrs, "algn"),
		Compound:  attrValue(line.Attrs, "cmpd"),
	}
	if border.Width == 0 {
		border.Width = 9525
	}
	if firstChild(line, "noFill") != nil {
		border.NoLine = true
		return border
	}
	if solidFill := firstChild(line, "solidFill"); solidFill != nil {
		if lineColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			border.HasLine = true
			border.Color = lineColor
		}
	}
	if dash := firstChild(line, "prstDash"); dash != nil {
		if value := attrValue(dash.Attrs, "val"); value != "" && value != "solid" {
			border.Dash = value
		}
	}
	if firstChild(line, "round") != nil {
		border.Join = "round"
	} else if firstChild(line, "bevel") != nil {
		border.Join = "bevel"
	} else if firstChild(line, "miter") != nil {
		border.Join = "miter"
	}
	return border
}

func packageTableStyles(pkg *pptx.Package, theme themeColors, fonts themeFonts, fillStyles themeFillStyles, lineStyles themeLineStyles, effectStyles themeEffectStyles) tableStyleSet {
	if data, ok := pkg.Parts["ppt/tableStyles.xml"]; ok {
		return parseTableStyles(data, theme, fonts, fillStyles, lineStyles, effectStyles)
	}
	return tableStyleSet{}
}

func parseTableStyles(data []byte, theme themeColors, fonts themeFonts, fillStyles themeFillStyles, lineStyles themeLineStyles, effectStyles themeEffectStyles) tableStyleSet {
	root, err := parseXMLNode(data)
	if err != nil {
		return tableStyleSet{}
	}
	styles := tableStyleSet{
		DefaultID: strings.TrimSpace(attrValue(root.Attrs, "def")),
		Styles:    map[string]tableStyle{},
	}
	for _, node := range childrenByName(root, "tblStyle") {
		style := tableStyle{
			ID:      strings.TrimSpace(attrValue(node.Attrs, "styleId")),
			Name:    attrValue(node.Attrs, "styleName"),
			Regions: map[string]tableStyleRegion{},
		}
		for _, child := range node.Children {
			if child.Name == "tblBg" {
				if background, ok := parseTableStyleBackgroundFill(child, theme, fillStyles); ok {
					style.HasBackground = true
					style.Background = background
				}
				if effects, ok := parseTableStyleBackgroundEffect(child, theme, effectStyles); ok {
					style.HasBackgroundEffect = true
					style.BackgroundEffect = effects
				}
				continue
			}
			if !isTableStyleRegionName(child.Name) {
				continue
			}
			style.Regions[child.Name] = parseTableStyleRegion(child, theme, fonts, lineStyles)
		}
		if style.ID != "" {
			styles.Styles[normalizedTableStyleID(style.ID)] = style
		}
	}
	return styles
}

func parseTableStyleBackgroundFill(node *xmlNode, theme themeColors, fillStyles themeFillStyles) (backgroundPaint, bool) {
	if fillRef := firstChild(node, "fillRef"); fillRef != nil && attrValue(fillRef.Attrs, "idx") != "0" {
		placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(fillRef, theme)
		if hasPlaceholderColor {
			if paint, ok := fillStyles.Style(parseIntAttr(fillRef.Attrs, "idx"), themeWithPlaceholderColor(theme, placeholderColor)); ok {
				return paint, true
			}
			return backgroundPaint{Color: placeholderColor}, true
		}
	}
	if solidFill := firstChild(node, "solidFill"); solidFill != nil {
		return backgroundPaintFromFillNode(solidFill, theme)
	}
	if gradFill := firstChild(node, "gradFill"); gradFill != nil {
		return backgroundPaintFromFillNode(gradFill, theme)
	}
	return backgroundPaint{}, false
}

func parseTableStyleBackgroundEffect(node *xmlNode, theme themeColors, effectStyles themeEffectStyles) (themeEffectStyle, bool) {
	if effectRef := firstChild(node, "effectRef"); effectRef != nil && attrValue(effectRef.Attrs, "idx") != "0" {
		styleTheme := theme
		if placeholderColor, ok := colorFromColorNodeWithTheme(effectRef, theme); ok {
			styleTheme = themeWithPlaceholderColor(theme, placeholderColor)
		}
		return effectStyles.Style(parseIntAttr(effectRef.Attrs, "idx"), styleTheme)
	}
	if effect := firstChild(node, "effect"); effect != nil {
		return parseThemeEffectStyle(effect, theme)
	}
	return themeEffectStyle{}, false
}

func isTableStyleRegionName(name string) bool {
	switch name {
	case "wholeTbl", "band1H", "band2H", "band1V", "band2V", "firstCol", "lastCol", "firstRow", "lastRow", "neCell", "nwCell", "seCell", "swCell":
		return true
	default:
		return false
	}
}

func parseTableStyleRegion(node *xmlNode, theme themeColors, fonts themeFonts, lineStyles themeLineStyles) tableStyleRegion {
	var region tableStyleRegion
	if textStyle := firstChild(node, "tcTxStyle"); textStyle != nil {
		if rawBold := attrValue(textStyle.Attrs, "b"); rawBold != "" {
			region.HasBold = true
			region.Bold = boolAttrOn(rawBold)
		}
		if rawItalic := attrValue(textStyle.Attrs, "i"); rawItalic != "" {
			region.HasItalic = true
			region.Italic = boolAttrOn(rawItalic)
		}
		if fontRef := firstChild(textStyle, "fontRef"); fontRef != nil {
			region.FontFamily = tableStyleFontFamily(fontRef, fonts)
		}
		if region.FontFamily == "" {
			region.FontFamily = tableStyleDirectFontFamily(textStyle)
		}
		if textColor, ok := colorFromColorNodeWithTheme(textStyle, theme); ok {
			region.HasTextColor = true
			region.TextColor = textColor
		}
	}
	if cellStyle := firstChild(node, "tcStyle"); cellStyle != nil {
		if fill := firstChild(cellStyle, "fill"); fill != nil {
			if firstChild(fill, "noFill") != nil {
				region.NoFill = true
			} else if solidFill := firstChild(fill, "solidFill"); solidFill != nil {
				if fillColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
					region.HasFill = true
					region.FillColor = fillColor
				}
			}
		}
		if fillRef := firstChild(cellStyle, "fillRef"); fillRef != nil {
			if fillColor, ok := colorFromColorNodeWithTheme(fillRef, theme); ok {
				region.HasFill = true
				region.FillColor = fillColor
			}
		}
		if borders := firstChild(cellStyle, "tcBdr"); borders != nil {
			region.Borders = parseTableStyleBorders(borders, theme, lineStyles)
		}
	}
	return region
}

func tableStyleFontFamily(fontRef *xmlNode, fonts themeFonts) string {
	switch attrValue(fontRef.Attrs, "idx") {
	case "major":
		return fonts.MajorLatin
	case "minor":
		return fonts.MinorLatin
	default:
		return ""
	}
}

func tableStyleDirectFontFamily(textStyle *xmlNode) string {
	font := firstChild(textStyle, "font")
	if font == nil {
		return ""
	}
	if typeface := typefaceFromChild(font, "latin"); typeface != "" {
		return typeface
	}
	if typeface := typefaceFromChild(font, "ea"); typeface != "" {
		return typeface
	}
	return typefaceFromChild(font, "cs")
}

func parseTableStyleBorders(node *xmlNode, theme themeColors, lineStyles themeLineStyles) tableStyleBorders {
	return tableStyleBorders{
		Left:    parseTableStyleBorder(node, "left", theme, lineStyles),
		Right:   parseTableStyleBorder(node, "right", theme, lineStyles),
		Top:     parseTableStyleBorder(node, "top", theme, lineStyles),
		Bottom:  parseTableStyleBorder(node, "bottom", theme, lineStyles),
		InsideH: parseTableStyleBorder(node, "insideH", theme, lineStyles),
		InsideV: parseTableStyleBorder(node, "insideV", theme, lineStyles),
	}
}

func parseTableStyleBorder(parent *xmlNode, name string, theme themeColors, lineStyles themeLineStyles) tableCellBorder {
	edge := firstChild(parent, name)
	if edge == nil {
		return tableCellBorder{}
	}
	line := firstChild(edge, "ln")
	if line == nil {
		if lineRef := firstChild(edge, "lnRef"); lineRef != nil {
			return parseTableStyleLineReference(lineRef, theme, lineStyles)
		}
		return tableCellBorder{Specified: true, NoLine: true}
	}
	return parseTableLineNode(line, theme, true)
}

func parseTableStyleLineReference(lineRef *xmlNode, theme themeColors, lineStyles themeLineStyles) tableCellBorder {
	if attrValue(lineRef.Attrs, "idx") == "0" {
		return tableCellBorder{Specified: true, NoLine: true}
	}
	placeholderColor, hasPlaceholderColor := colorFromColorNodeWithTheme(lineRef, theme)
	if hasPlaceholderColor {
		if border, ok := lineStyles.Style(parseIntAttr(lineRef.Attrs, "idx"), themeWithPlaceholderColor(theme, placeholderColor)); ok {
			return border
		}
		return tableCellBorder{Specified: true, HasLine: true, Color: placeholderColor, Width: 9525}
	}
	if border, ok := lineStyles.Style(parseIntAttr(lineRef.Attrs, "idx"), theme); ok {
		return border
	}
	return tableCellBorder{Specified: true, NoLine: true}
}

func boolAttrOn(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on":
		return true
	default:
		return false
	}
}

func normalizedTableStyleID(styleID string) string {
	return strings.ToLower(strings.TrimSpace(styleID))
}

func renderTableGraphicFrame(slidePart string, size slideSize, img *image.RGBA, element *slideElement, tableStyles tableStyleSet) []model.SkipItem {
	if !element.HasTransform || element.ExtCX <= 0 || element.ExtCY <= 0 || len(element.Table.Rows) == 0 {
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
	columnOffsets := tableGridOffsets(tableColumnWeights(element.Table), target.Min.X, target.Max.X, element.OffX, element.ExtCX, size.CX, img.Bounds().Dx())
	rowOffsets := tableRowOffsets(element.Table, target.Min.Y, target.Max.Y, element.OffY, element.ExtCY, size.CY, img.Bounds().Dy())
	style, hasStyle := tableStyleForTable(element.Table, tableStyles)
	backgroundEffectRendered := true
	if hasStyle {
		if style.HasBackgroundEffect && style.BackgroundEffect.HasShadow {
			backgroundElement := slideElement{
				PrstGeom:        "rect",
				HasShadow:       true,
				ShadowColor:     style.BackgroundEffect.ShadowColor,
				ShadowBlur:      style.BackgroundEffect.ShadowBlur,
				ShadowDistance:  style.BackgroundEffect.ShadowDistance,
				ShadowDirection: style.BackgroundEffect.ShadowDirection,
			}
			backgroundEffectRendered = drawShapeShadow(img, target, backgroundElement, size)
		}
		if style.HasBackground {
			drawTableBackgroundPaint(img, target, style.Background)
		}
	}
	for rowIndex, row := range element.Table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			if cellRect.Empty() {
				continue
			}
			style := resolvedTableCellStyle(element.Table, tableStyles, rowIndex, columnIndex)
			if fill, ok := tableCellFill(style, cell); ok {
				drawTableCellFill(img, cellRect, fill)
			}
		}
	}
	drawTableBorders(img, target, size, element.Table, tableStyles, columnOffsets, rowOffsets)
	var failures []string
	fontMessages := map[string]bool{}
	for rowIndex, row := range element.Table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || cell.Text == "" || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			textRect := tableCellTextRect(cellRect, cell, size, img.Bounds())
			if textRect.Empty() {
				textRect = cellRect
			}
			hasTextColor := cell.HasTextColor
			textColor := cell.TextColor
			style := resolvedTableCellStyle(element.Table, tableStyles, rowIndex, columnIndex)
			if color, ok := tableCellTextColor(style); ok && !hasTextColor {
				hasTextColor = true
				textColor = color
			}
			textParagraphs := cell.TextParagraphs
			if color, ok := tableCellTextColor(style); ok {
				textParagraphs = tableTextParagraphsWithColor(textParagraphs, cell.Text, color)
			}
			if tableCellTextBold(style) {
				textParagraphs = tableTextParagraphsWithBold(textParagraphs, cell.Text)
			}
			fontFamily := tableCellTextFontFamily(style)
			cellElement := slideElement{
				Text:           cell.Text,
				TextParagraphs: textParagraphs,
				FontFamily:     fontFamily,
				Italic:         tableCellTextItalic(style),
				FontSize:       cell.FontSize,
				HasTextColor:   hasTextColor,
				TextColor:      textColor,
				TextAlign:      cell.TextAlign,
				TextAnchor:     tableCellTextAnchor(cell),
			}
			if err := drawShapeTextWithDPI(img, textRect, cellElement, renderDPIForCanvas(size, img.Bounds())); err != nil {
				failures = append(failures, err.Error())
			}
			for _, message := range fontResolutionUnsupportedMessages(cellElement) {
				fontMessages[message] = true
			}
		}
	}
	element.Rendered = true
	if len(failures) > 0 {
		return []model.SkipItem{unsupportedItem(slidePart, unsupportedCode, fmt.Sprintf("graphic frame object %q table text could not be rendered: %s", elementLabel(*element), strings.Join(failures, "; ")))}
	}
	unsupported := make([]model.SkipItem, 0, len(fontMessages)+1)
	for _, message := range sortedKeys(fontMessages) {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table %s", elementLabel(*element), message)))
	}
	if hasStyle && style.HasBackgroundEffect && style.BackgroundEffect.HasShadow && !backgroundEffectRendered && style.BackgroundEffect.ShadowColor.A != 0 {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table background shadow geometry was not rendered", elementLabel(*element))))
	}
	for _, message := range element.Table.UnsupportedFeatures {
		unsupported = append(unsupported, unsupportedItem(slidePart, partialUnsupportedCode, fmt.Sprintf("graphic frame object %q table %s", elementLabel(*element), message)))
	}
	return unsupported
}

func drawTableBackgroundPaint(img *image.RGBA, target image.Rectangle, paint backgroundPaint) {
	if target.Empty() {
		return
	}
	if paint.HasGradient {
		drawGradientRect(img, target, paint.Gradient, false)
		return
	}
	if paint.Color.A == 0 {
		return
	}
	drawTableCellFill(img, target, paint.Color)
}

func tableCellTextAnchor(cell tableCell) string {
	return cell.TextAnchor
}

func drawTableCellFill(img *image.RGBA, rect image.Rectangle, fill color.RGBA) {
	op := draw.Src
	if fill.A < 255 {
		op = draw.Over
	}
	draw.Draw(img, rect, &image.Uniform{C: fill}, image.Point{}, op)
}

func drawTableBorders(img *image.RGBA, target image.Rectangle, size slideSize, table tableModel, tableStyles tableStyleSet, columnOffsets []int, rowOffsets []int) {
	rowCount := len(table.Rows)
	columnCount := tableColumnCount(table)
	for rowIndex, row := range table.Rows {
		for columnIndex, cell := range row.Cells {
			if cell.HMerge || cell.VMerge || columnIndex+1 >= len(columnOffsets) || rowIndex+1 >= len(rowOffsets) {
				continue
			}
			cellRect := tableCellRect(columnOffsets, rowOffsets, rowIndex, columnIndex, cell).Intersect(target)
			if cellRect.Empty() {
				continue
			}
			style := resolvedTableCellStyle(table, tableStyles, rowIndex, columnIndex)
			top := effectiveTableCellBorder(cell.BorderTop, tableEdgeBorder(style.Borders, tableEdgeTop, rowIndex, columnIndex, rowCount, columnCount), true)
			bottom := effectiveTableCellBorder(cell.BorderBottom, tableEdgeBorder(style.Borders, tableEdgeBottom, rowIndex, columnIndex, rowCount, columnCount), true)
			left := effectiveTableCellBorder(cell.BorderLeft, tableEdgeBorder(style.Borders, tableEdgeLeft, rowIndex, columnIndex, rowCount, columnCount), true)
			right := effectiveTableCellBorder(cell.BorderRight, tableEdgeBorder(style.Borders, tableEdgeRight, rowIndex, columnIndex, rowCount, columnCount), true)
			drawTableCellBorder(img, size, target, cellRect, top, tableEdgeTop)
			drawTableCellBorder(img, size, target, cellRect, bottom, tableEdgeBottom)
			drawTableCellBorder(img, size, target, cellRect, left, tableEdgeLeft)
			drawTableCellBorder(img, size, target, cellRect, right, tableEdgeRight)
			drawTableCellRoundBorderJoins(img, size, target, cellRect, top, bottom, left, right)
		}
	}
}

const (
	tableEdgeTop = iota
	tableEdgeBottom
	tableEdgeLeft
	tableEdgeRight
)

func tableEdgeBorder(borders tableStyleBorders, edge int, rowIndex int, columnIndex int, rowCount int, columnCount int) tableCellBorder {
	switch edge {
	case tableEdgeTop:
		if rowIndex > 0 && borders.InsideH.Specified {
			return borders.InsideH
		}
		if borders.Top.Specified {
			return borders.Top
		}
	case tableEdgeBottom:
		if rowCount <= 0 || rowIndex < rowCount-1 {
			if borders.InsideH.Specified {
				return borders.InsideH
			}
		}
		if borders.Bottom.Specified {
			return borders.Bottom
		}
		if borders.InsideH.Specified {
			return borders.InsideH
		}
	case tableEdgeLeft:
		if columnIndex > 0 && borders.InsideV.Specified {
			return borders.InsideV
		}
		if borders.Left.Specified {
			return borders.Left
		}
	case tableEdgeRight:
		if columnCount <= 0 || columnIndex < columnCount-1 {
			if borders.InsideV.Specified {
				return borders.InsideV
			}
		}
		if borders.Right.Specified {
			return borders.Right
		}
		if borders.InsideV.Specified {
			return borders.InsideV
		}
	}
	return defaultTableGridBorder()
}

func defaultTableGridBorder() tableCellBorder {
	return tableCellBorder{
		Specified: true,
		HasLine:   true,
		Color:     color.RGBA{R: 90, G: 90, B: 90, A: 255},
		Width:     9525,
	}
}

func drawTableCellBorder(img *image.RGBA, size slideSize, tableRect image.Rectangle, rect image.Rectangle, border tableCellBorder, edge int) {
	if !border.Specified || border.NoLine || !border.HasLine {
		return
	}
	width := emuLineWidthToPixels(border.Width, size.CX, img.Bounds().Dx())
	switch edge {
	case tableEdgeTop:
		drawTableBorderLine(img, rect.Min.X, rect.Min.Y, rect.Max.X-1, rect.Min.Y, border.Color, width, border.Dash, border.Compound, border.Cap, true)
	case tableEdgeBottom:
		y := rect.Max.Y
		if y >= tableRect.Max.Y {
			y = rect.Max.Y - 1
		}
		drawTableBorderLine(img, rect.Min.X, y, rect.Max.X-1, y, border.Color, width, border.Dash, border.Compound, border.Cap, true)
	case tableEdgeLeft:
		drawTableBorderLine(img, rect.Min.X, rect.Min.Y, rect.Min.X, rect.Max.Y-1, border.Color, width, border.Dash, border.Compound, border.Cap, false)
	case tableEdgeRight:
		x := rect.Max.X
		if x >= tableRect.Max.X {
			x = rect.Max.X - 1
		}
		drawTableBorderLine(img, x, rect.Min.Y, x, rect.Max.Y-1, border.Color, width, border.Dash, border.Compound, border.Cap, false)
	}
}

func isSupportedTableCompoundLine(compound string) bool {
	return compound == "" || compound == "sng" || compound == "dbl"
}

func drawTableBorderLine(img *image.RGBA, x0 int, y0 int, x1 int, y1 int, c color.RGBA, width int, dash string, compound string, cap string, horizontal bool) {
	if compound != "dbl" {
		drawStyledLine(img, x0, y0, x1, y1, c, width, dash, cap)
		return
	}
	strokeWidth, firstOffset, secondOffset := doubleCompoundLineMetrics(width)
	if horizontal {
		drawStyledLine(img, x0, y0+firstOffset, x1, y1+firstOffset, c, strokeWidth, dash, cap)
		drawStyledLine(img, x0, y0+secondOffset, x1, y1+secondOffset, c, strokeWidth, dash, cap)
		return
	}
	drawStyledLine(img, x0+firstOffset, y0, x1+firstOffset, y1, c, strokeWidth, dash, cap)
	drawStyledLine(img, x0+secondOffset, y0, x1+secondOffset, y1, c, strokeWidth, dash, cap)
}

func doubleCompoundLineMetrics(width int) (int, int, int) {
	if width < 1 {
		width = 1
	}
	strokeWidth := width / 3
	if strokeWidth < 1 {
		strokeWidth = 1
	}
	gap := width - (2 * strokeWidth)
	if gap < 1 {
		gap = 1
	}
	separation := strokeWidth + gap
	firstOffset := -(separation / 2)
	secondOffset := firstOffset + separation
	if firstOffset == secondOffset {
		secondOffset++
	}
	return strokeWidth, firstOffset, secondOffset
}

func effectiveTableCellBorder(border tableCellBorder, defaultBorder tableCellBorder, hasDefaultBorder bool) tableCellBorder {
	if border.Specified {
		return border
	}
	if hasDefaultBorder {
		return defaultBorder
	}
	return tableCellBorder{}
}

func drawTableCellRoundBorderJoins(img *image.RGBA, size slideSize, tableRect image.Rectangle, rect image.Rectangle, top tableCellBorder, bottom tableCellBorder, left tableCellBorder, right tableCellBorder) {
	topY := rect.Min.Y
	bottomY := rect.Max.Y
	if bottomY >= tableRect.Max.Y {
		bottomY = rect.Max.Y - 1
	}
	leftX := rect.Min.X
	rightX := rect.Max.X
	if rightX >= tableRect.Max.X {
		rightX = rect.Max.X - 1
	}
	drawTableRoundBorderJoin(img, size, leftX, topY, top, left)
	drawTableRoundBorderJoin(img, size, rightX, topY, top, right)
	drawTableRoundBorderJoin(img, size, leftX, bottomY, bottom, left)
	drawTableRoundBorderJoin(img, size, rightX, bottomY, bottom, right)
}

func drawTableRoundBorderJoin(img *image.RGBA, size slideSize, x int, y int, first tableCellBorder, second tableCellBorder) {
	for _, border := range []tableCellBorder{first, second} {
		if !tableBorderHasRenderableRoundJoin(border) {
			continue
		}
		width := emuLineWidthToPixels(border.Width, size.CX, img.Bounds().Dx())
		drawRoundLineJoin(img, x, y, border.Color, width)
	}
}

func tableBorderHasRenderableRoundJoin(border tableCellBorder) bool {
	return border.Specified && border.HasLine && !border.NoLine && border.Join == "round" && (border.Compound == "" || border.Compound == "sng")
}

func drawRoundLineJoin(img *image.RGBA, centerX int, centerY int, c color.RGBA, width int) {
	if c.A == 0 {
		return
	}
	if width < 1 {
		width = 1
	}
	radius := float64(width) / 2
	padding := int(math.Ceil(radius)) + 1
	bounds := image.Rect(centerX-padding, centerY-padding, centerX+padding+1, centerY+padding+1).Intersect(img.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := circleCoverage(float64(x), float64(y), float64(centerX), float64(centerY), radius)
			if coverage == 4 && c.A == 255 {
				img.SetRGBA(x, y, c)
			} else if coverage > 0 {
				layer := c
				layer.A = coverageAlpha(c.A, coverage)
				blendPixel(img, x, y, layer)
			}
		}
	}
}

func circleCoverage(x float64, y float64, centerX float64, centerY float64, radius float64) int {
	if radius <= 0 {
		return 0
	}
	coverage := 0
	for _, offset := range coverageSampleOffsets {
		dx := x + offset.x - centerX
		dy := y + offset.y - centerY
		if dx*dx+dy*dy <= radius*radius {
			coverage++
		}
	}
	return coverage
}

func tableCellTextRect(cellRect image.Rectangle, cell tableCell, size slideSize, imageBounds image.Rectangle) image.Rectangle {
	if !cell.HasMargins {
		return image.Rect(
			cellRect.Min.X+scaleEMU(defaultTableCellHorizontalMarginEMU, size.CX, imageBounds.Dx()),
			cellRect.Min.Y+scaleEMU(defaultTableCellVerticalMarginEMU, size.CY, imageBounds.Dy()),
			cellRect.Max.X-scaleEMU(defaultTableCellHorizontalMarginEMU, size.CX, imageBounds.Dx()),
			cellRect.Max.Y-scaleEMU(defaultTableCellVerticalMarginEMU, size.CY, imageBounds.Dy()),
		)
	}
	left := scaleEMU(cell.MarginLeft, size.CX, imageBounds.Dx())
	right := scaleEMU(cell.MarginRight, size.CX, imageBounds.Dx())
	top := scaleEMU(cell.MarginTop, size.CY, imageBounds.Dy())
	bottom := scaleEMU(cell.MarginBottom, size.CY, imageBounds.Dy())
	return image.Rect(cellRect.Min.X+left, cellRect.Min.Y+top, cellRect.Max.X-right, cellRect.Max.Y-bottom)
}

const (
	defaultTableCellHorizontalMarginEMU = 91440
	defaultTableCellVerticalMarginEMU   = 45720
)

func tableCellRect(columnOffsets []int, rowOffsets []int, rowIndex int, columnIndex int, cell tableCell) image.Rectangle {
	rowEnd := rowIndex + cell.RowSpan
	if rowEnd >= len(rowOffsets) {
		rowEnd = len(rowOffsets) - 1
	}
	if rowEnd <= rowIndex {
		rowEnd = rowIndex + 1
	}
	columnEnd := columnIndex + cell.ColSpan
	if columnEnd >= len(columnOffsets) {
		columnEnd = len(columnOffsets) - 1
	}
	if columnEnd <= columnIndex {
		columnEnd = columnIndex + 1
	}
	return image.Rect(columnOffsets[columnIndex], rowOffsets[rowIndex], columnOffsets[columnEnd], rowOffsets[rowEnd])
}

func tableCellFill(style tableStyleRegion, cell tableCell) (color.RGBA, bool) {
	if cell.NoFill {
		return color.RGBA{}, false
	}
	if cell.HasFill {
		return cell.FillColor, true
	}
	if style.NoFill {
		return color.RGBA{}, false
	}
	if style.HasFill {
		return style.FillColor, true
	}
	return color.RGBA{}, false
}

func tableCellTextColor(style tableStyleRegion) (color.RGBA, bool) {
	return style.TextColor, style.HasTextColor
}

func tableCellTextBold(style tableStyleRegion) bool {
	return style.HasBold && style.Bold
}

func tableCellTextItalic(style tableStyleRegion) bool {
	return style.HasItalic && style.Italic
}

func tableCellTextFontFamily(style tableStyleRegion) string {
	return style.FontFamily
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key, ok := range values {
		if ok {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func resolvedTableCellStyle(table tableModel, styles tableStyleSet, rowIndex int, columnIndex int) tableStyleRegion {
	style, ok := tableStyleForTable(table, styles)
	if !ok {
		return tableStyleRegion{}
	}
	var resolved tableStyleRegion
	for _, regionName := range tableStyleRegionNamesForCell(table, rowIndex, columnIndex) {
		region, ok := style.Regions[regionName]
		if !ok {
			continue
		}
		mergeTableStyleRegion(&resolved, region)
	}
	return resolved
}

func tableStyleForTable(table tableModel, styles tableStyleSet) (tableStyle, bool) {
	if len(styles.Styles) == 0 {
		return tableStyle{}, false
	}
	if table.StyleID != "" {
		if style, ok := styles.Styles[normalizedTableStyleID(table.StyleID)]; ok {
			return style, true
		}
	}
	if styles.DefaultID != "" {
		if style, ok := styles.Styles[normalizedTableStyleID(styles.DefaultID)]; ok {
			return style, true
		}
	}
	return tableStyle{}, false
}

func tableStyleRegionNamesForCell(table tableModel, rowIndex int, columnIndex int) []string {
	rowCount := len(table.Rows)
	columnCount := tableColumnCount(table)
	names := []string{"wholeTbl"}
	if table.BandRow {
		switch tableBandIndex(rowIndex, table.FirstRow) {
		case 0:
			names = append(names, "band1H")
		case 1:
			names = append(names, "band2H")
		}
	}
	if table.BandCol {
		switch tableBandIndex(columnIndex, table.FirstCol) {
		case 0:
			names = append(names, "band1V")
		case 1:
			names = append(names, "band2V")
		}
	}
	if table.FirstCol && columnIndex == 0 {
		names = append(names, "firstCol")
	}
	if table.LastCol && columnCount > 0 && columnIndex == columnCount-1 {
		names = append(names, "lastCol")
	}
	if table.FirstRow && rowIndex == 0 {
		names = append(names, "firstRow")
	}
	if table.LastRow && rowCount > 0 && rowIndex == rowCount-1 {
		names = append(names, "lastRow")
	}
	if table.FirstRow && table.FirstCol && rowIndex == 0 && columnIndex == 0 {
		names = append(names, "nwCell")
	}
	if table.FirstRow && table.LastCol && rowIndex == 0 && columnCount > 0 && columnIndex == columnCount-1 {
		names = append(names, "neCell")
	}
	if table.LastRow && table.FirstCol && rowCount > 0 && rowIndex == rowCount-1 && columnIndex == 0 {
		names = append(names, "swCell")
	}
	if table.LastRow && table.LastCol && rowCount > 0 && columnCount > 0 && rowIndex == rowCount-1 && columnIndex == columnCount-1 {
		names = append(names, "seCell")
	}
	return names
}

func tableBandIndex(index int, skipFirst bool) int {
	if skipFirst {
		index--
	}
	if index < 0 {
		return -1
	}
	return index % 2
}

func tableColumnCount(table tableModel) int {
	columnCount := len(table.Columns)
	for _, row := range table.Rows {
		if len(row.Cells) > columnCount {
			columnCount = len(row.Cells)
		}
	}
	return columnCount
}

func mergeTableStyleRegion(dst *tableStyleRegion, src tableStyleRegion) {
	if src.NoFill {
		dst.NoFill = true
		dst.HasFill = false
		dst.FillColor = color.RGBA{}
	} else if src.HasFill {
		dst.NoFill = false
		dst.HasFill = true
		dst.FillColor = src.FillColor
	}
	if src.HasTextColor {
		dst.HasTextColor = true
		dst.TextColor = src.TextColor
	}
	if src.FontFamily != "" {
		dst.FontFamily = src.FontFamily
	}
	if src.HasBold {
		dst.HasBold = true
		dst.Bold = src.Bold
	}
	if src.HasItalic {
		dst.HasItalic = true
		dst.Italic = src.Italic
	}
	mergeTableBorder(&dst.Borders.Left, src.Borders.Left)
	mergeTableBorder(&dst.Borders.Right, src.Borders.Right)
	mergeTableBorder(&dst.Borders.Top, src.Borders.Top)
	mergeTableBorder(&dst.Borders.Bottom, src.Borders.Bottom)
	mergeTableBorder(&dst.Borders.InsideH, src.Borders.InsideH)
	mergeTableBorder(&dst.Borders.InsideV, src.Borders.InsideV)
}

func mergeTableBorder(dst *tableCellBorder, src tableCellBorder) {
	if src.Specified {
		*dst = src
	}
}

func tableTextParagraphsWithBold(paragraphs []textParagraph, fallbackText string) []textParagraph {
	if len(paragraphs) == 0 {
		if strings.TrimSpace(fallbackText) == "" {
			return nil
		}
		return []textParagraph{{Text: strings.TrimSpace(fallbackText), Bold: true}}
	}
	output := make([]textParagraph, len(paragraphs))
	copy(output, paragraphs)
	for paragraphIndex := range output {
		output[paragraphIndex].Bold = true
		runs := make([]textRun, len(output[paragraphIndex].Runs))
		copy(runs, output[paragraphIndex].Runs)
		for runIndex := range runs {
			runs[runIndex].Bold = true
		}
		output[paragraphIndex].Runs = runs
	}
	return output
}

func tableTextParagraphsWithColor(paragraphs []textParagraph, fallbackText string, textColor color.RGBA) []textParagraph {
	if len(paragraphs) == 0 {
		if strings.TrimSpace(fallbackText) == "" {
			return nil
		}
		return []textParagraph{{Text: strings.TrimSpace(fallbackText), HasTextColor: true, TextColor: textColor}}
	}
	output := make([]textParagraph, len(paragraphs))
	copy(output, paragraphs)
	for paragraphIndex := range output {
		output[paragraphIndex].HasTextColor = true
		output[paragraphIndex].TextColor = textColor
		runs := make([]textRun, len(output[paragraphIndex].Runs))
		copy(runs, output[paragraphIndex].Runs)
		output[paragraphIndex].Runs = runs
	}
	return output
}

func tableColumnWeights(table tableModel) []int64 {
	columnCount := len(table.Columns)
	for _, row := range table.Rows {
		if len(row.Cells) > columnCount {
			columnCount = len(row.Cells)
		}
	}
	if columnCount == 0 {
		return nil
	}
	weights := make([]int64, columnCount)
	for index := range weights {
		if index < len(table.Columns) && table.Columns[index] > 0 {
			weights[index] = table.Columns[index]
		} else {
			weights[index] = 1
		}
	}
	return weights
}

func tableRowWeights(table tableModel) []int64 {
	if len(table.Rows) == 0 {
		return nil
	}
	weights := make([]int64, len(table.Rows))
	for index, row := range table.Rows {
		if row.HasHeight {
			weights[index] = row.Height
		} else {
			weights[index] = 1
		}
	}
	return weights
}

func tableRowOffsets(table tableModel, min int, max int, originEMU int64, frameEMU int64, slideEMU int64, canvasPixels int) []int {
	weights := tableRowWeights(table)
	if len(weights) <= 1 || !table.FirstRow || !tableFirstRowHasSpanningCells(table) || frameEMU <= 0 {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 || total >= frameEMU {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	headerEnd := scaleEMU(originEMU+weights[0], slideEMU, canvasPixels)
	if headerEnd <= min || headerEnd >= max {
		return tableGridOffsets(weights, min, max, originEMU, frameEMU, slideEMU, canvasPixels)
	}
	offsets := make([]int, 0, len(weights)+1)
	offsets = append(offsets, min, headerEnd)
	bodyOffsets := proportionalOffsets(weights[1:], headerEnd, max)
	offsets = append(offsets, bodyOffsets[1:]...)
	return offsets
}

func tableFirstRowHasSpanningCells(table tableModel) bool {
	if len(table.Rows) == 0 {
		return false
	}
	for _, cell := range table.Rows[0].Cells {
		if cell.ColSpan > 1 || cell.HMerge {
			return true
		}
	}
	return false
}

func tableGridOffsets(weights []int64, min int, max int, originEMU int64, frameEMU int64, slideEMU int64, canvasPixels int) []int {
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total > 0 && frameEMU > 0 && total == frameEMU {
		offsets := make([]int, len(weights)+1)
		offsets[0] = min
		running := int64(0)
		for index, weight := range weights {
			running += weight
			offsets[index+1] = scaleEMU(originEMU+running, slideEMU, canvasPixels)
		}
		return offsets
	}
	return proportionalOffsets(weights, min, max)
}

func proportionalOffsets(weights []int64, min int, max int) []int {
	offsets := make([]int, len(weights)+1)
	offsets[0] = min
	total := int64(0)
	for _, weight := range weights {
		total += weight
	}
	if total <= 0 {
		total = int64(len(weights))
		for index := range weights {
			weights[index] = 1
		}
	}
	span := max - min
	running := int64(0)
	for index, weight := range weights {
		running += weight
		offsets[index+1] = min + int(math.Round(float64(span)*float64(running)/float64(total)))
	}
	offsets[len(offsets)-1] = max
	return offsets
}
