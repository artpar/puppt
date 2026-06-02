package render

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

func textFromNode(node *xmlNode) string {
	if node.Name == "br" {
		return "\n"
	}
	if node.Name == "tab" {
		return "\t"
	}
	var output strings.Builder
	if node.Name == "t" || node.Name == "fld" {
		output.WriteString(node.Text)
	}
	for _, child := range node.Children {
		output.WriteString(textFromNode(child))
	}
	if node.Name == "p" && output.Len() > 0 {
		output.WriteByte('\n')
	}
	return output.String()
}

func textParagraphsFromNode(node *xmlNode) []textParagraph {
	return textParagraphsFromNodeWithTheme(node, defaultThemeColors())
}

func textParagraphsFromNodeWithTheme(node *xmlNode, theme themeColors) []textParagraph {
	var output []textParagraph
	styles := paragraphStylesFromListStyle(firstDescendant(node, "lstStyle"), theme)
	autoCounters := map[int]int{}
	for _, paragraphNode := range descendantsByName(node, "p") {
		text := strings.TrimSpace(textFromNode(paragraphNode))
		paragraph := textParagraph{Text: text}
		paragraph.Runs = paragraphTextRunsWithTheme(paragraphNode, theme)
		if pPr := firstChild(paragraphNode, "pPr"); pPr != nil {
			paragraph.Level = int(parseIntAttr(pPr.Attrs, "lvl"))
			paragraph.TextAlign = attrValue(pPr.Attrs, "algn")
			paragraph.FontAlign = attrValue(pPr.Attrs, "fontAlgn")
			applyParagraphPropertyFlags(&paragraph, pPr)
		}
		style := styles[paragraph.Level]
		applyParagraphStyle(&paragraph, style)
		hasLocalBulletChoice := false
		localAutoNumberApplied := false
		if pPr := firstChild(paragraphNode, "pPr"); pPr != nil {
			if value := attrValue(pPr.Attrs, "marL"); value != "" {
				paragraph.MarginLeft = parseIntAttr(pPr.Attrs, "marL")
				paragraph.HasMarginLeft = true
			}
			if value := attrValue(pPr.Attrs, "marR"); value != "" {
				paragraph.MarginRight = parseIntAttr(pPr.Attrs, "marR")
				paragraph.HasMarginRight = true
			}
			if value := attrValue(pPr.Attrs, "indent"); value != "" {
				paragraph.Indent = parseIntAttr(pPr.Attrs, "indent")
				paragraph.HasIndent = true
			}
			if value := attrValue(pPr.Attrs, "defTabSz"); value != "" {
				paragraph.DefaultTabSize = parseIntAttr(pPr.Attrs, "defTabSz")
				paragraph.HasDefaultTab = paragraph.DefaultTabSize > 0
			}
			if spcBef := firstChild(pPr, "spcBef"); spcBef != nil {
				paragraph.HasSpaceBefore = true
				paragraph.SpaceBefore, paragraph.SpaceBeforePct = parseSpacingValue(spcBef)
			}
			if spcAft := firstChild(pPr, "spcAft"); spcAft != nil {
				paragraph.HasSpaceAfter = true
				paragraph.SpaceAfter, paragraph.SpaceAfterPct = parseSpacingValue(spcAft)
			}
			if lnSpc := firstChild(pPr, "lnSpc"); lnSpc != nil {
				paragraph.HasLineSpacing = true
				paragraph.LineSpacingPct = parseSpacingPercent(lnSpc)
			}
			paragraph.TabStops = parseParagraphTabStops(pPr)
			if bulletColorNode := firstChild(pPr, "buClr"); bulletColorNode != nil {
				if bulletColor, ok := colorFromColorNodeWithTheme(bulletColorNode, theme); ok {
					paragraph.HasBulletColor = true
					paragraph.BulletColor = bulletColor
				}
			}
			if firstChild(pPr, "buClrTx") != nil {
				paragraph.BulletColorTx = true
				paragraph.HasBulletColor = false
				paragraph.BulletColor = color.RGBA{}
			}
			paragraph.BulletFontFamily = bulletFontFamilyFromProperties(pPr)
			if firstChild(pPr, "buFontTx") != nil {
				paragraph.BulletFontTx = true
				paragraph.BulletFontFamily = ""
			}
			if bullet := firstChild(pPr, "buChar"); bullet != nil {
				hasLocalBulletChoice = true
				paragraph.Bullet = normalizeBulletCharForFont(attrValue(bullet.Attrs, "char"), paragraph.BulletFontFamily)
				paragraph.NoBullet = false
			}
			if autoNum := firstChild(pPr, "buAutoNum"); autoNum != nil {
				hasLocalBulletChoice = true
				localAutoNumberApplied = true
				paragraph.HasAutoNumber = true
				if startAt := int(parseIntAttr(autoNum.Attrs, "startAt")); startAt > 0 {
					autoCounters[paragraph.Level] = startAt
				} else {
					autoCounters[paragraph.Level]++
				}
				for level := paragraph.Level + 1; level < 9; level++ {
					delete(autoCounters, level)
				}
				paragraph.Bullet = autoNumberBullet(attrValue(autoNum.Attrs, "type"), autoCounters[paragraph.Level])
				paragraph.NoBullet = false
			}
			if firstChild(pPr, "buNone") != nil {
				hasLocalBulletChoice = true
				paragraph.NoBullet = true
				paragraph.Bullet = ""
			} else if paragraph.Bullet == "" && paragraph.Level > 0 && !style.HasAutoNumber {
				paragraph.Bullet = "•"
			}
			applyBulletSizePropertiesToParagraph(&paragraph, pPr)
			if paragraph.NoBullet {
				if bulletSize := firstChild(pPr, "buSzPts"); bulletSize != nil {
					if size := parseIntAttr(bulletSize.Attrs, "val"); size > 0 {
						paragraph.FontSize = int(size)
					}
				}
			}
			if defRPr := firstChild(pPr, "defRPr"); defRPr != nil {
				applyRunPropertiesToParagraphDefaults(&paragraph, defRPr, theme)
			}
		}
		if !localAutoNumberApplied && !hasLocalBulletChoice && !paragraph.NoBullet && paragraph.Bullet == "" && style.HasAutoNumber {
			if style.AutoNumberStart > 0 && autoCounters[paragraph.Level] == 0 {
				autoCounters[paragraph.Level] = style.AutoNumberStart
			} else {
				autoCounters[paragraph.Level]++
			}
			for level := paragraph.Level + 1; level < 9; level++ {
				delete(autoCounters, level)
			}
			paragraph.Bullet = autoNumberBullet(style.AutoNumberType, autoCounters[paragraph.Level])
			paragraph.HasAutoNumber = true
		}
		if endParaRPr := firstChild(paragraphNode, "endParaRPr"); endParaRPr != nil && !textRunsHaveRunMetricProperties(paragraph.Runs) {
			applyRunPropertiesToParagraphDefaults(&paragraph, endParaRPr, theme)
		}
		if size := textRunsFontSize(paragraph.Runs); size > 0 {
			paragraph.FontSize = size
		}
		if len(paragraph.Runs) > 0 {
			if textRunsAllBold(paragraph.Runs) {
				paragraph.HasBold = true
				paragraph.Bold = true
			}
			if textRunsAllItalic(paragraph.Runs) {
				paragraph.HasItalic = true
				paragraph.Italic = true
			}
		}
		if paragraph.Text == "" && len(paragraph.Runs) == 0 && !hasLocalBulletChoice {
			paragraph.NoBullet = true
			paragraph.Bullet = ""
		}
		output = append(output, paragraph)
	}
	return output
}

func parseParagraphTabStops(pPr *xmlNode) []int64 {
	tabList := firstChild(pPr, "tabLst")
	if tabList == nil {
		return nil
	}
	var stops []int64
	for _, tab := range childrenByName(tabList, "tab") {
		pos := parseIntAttr(tab.Attrs, "pos")
		if pos <= 0 {
			continue
		}
		stops = append(stops, pos)
	}
	if len(stops) == 0 {
		return nil
	}
	sort.Slice(stops, func(i, j int) bool { return stops[i] < stops[j] })
	return stops
}

func autoNumberBullet(kind string, index int) string {
	if index < 1 {
		index = 1
	}
	switch kind {
	case "alphaLcParenBoth":
		return "(" + alphaNumber(index, false) + ")"
	case "alphaUcParenBoth":
		return "(" + alphaNumber(index, true) + ")"
	case "alphaLcPeriod":
		return alphaNumber(index, false) + "."
	case "alphaUcPeriod":
		return alphaNumber(index, true) + "."
	case "alphaLcParenR":
		return alphaNumber(index, false) + ")"
	case "alphaUcParenR":
		return alphaNumber(index, true) + ")"
	case "arabicParenBoth":
		return fmt.Sprintf("(%d)", index)
	case "arabicParenR":
		return fmt.Sprintf("%d)", index)
	case "arabicPlain":
		return fmt.Sprintf("%d", index)
	case "romanLcParenBoth":
		return "(" + romanNumber(index, false) + ")"
	case "romanUcParenBoth":
		return "(" + romanNumber(index, true) + ")"
	case "romanLcParenR":
		return romanNumber(index, false) + ")"
	case "romanUcParenR":
		return romanNumber(index, true) + ")"
	case "romanLcPeriod":
		return romanNumber(index, false) + "."
	case "romanUcPeriod":
		return romanNumber(index, true) + "."
	default:
		return fmt.Sprintf("%d.", index)
	}
}

func alphaNumber(index int, upper bool) string {
	var chars []byte
	for index > 0 {
		index--
		chars = append([]byte{byte('a' + index%26)}, chars...)
		index /= 26
	}
	if upper {
		for idx := range chars {
			chars[idx] = byte(unicode.ToUpper(rune(chars[idx])))
		}
	}
	return string(chars)
}

func romanNumber(index int, upper bool) string {
	type romanToken struct {
		value int
		lower string
		upper string
	}
	tokens := []romanToken{
		{1000, "m", "M"},
		{900, "cm", "CM"},
		{500, "d", "D"},
		{400, "cd", "CD"},
		{100, "c", "C"},
		{90, "xc", "XC"},
		{50, "l", "L"},
		{40, "xl", "XL"},
		{10, "x", "X"},
		{9, "ix", "IX"},
		{5, "v", "V"},
		{4, "iv", "IV"},
		{1, "i", "I"},
	}
	var output strings.Builder
	for _, token := range tokens {
		for index >= token.value {
			if upper {
				output.WriteString(token.upper)
			} else {
				output.WriteString(token.lower)
			}
			index -= token.value
		}
	}
	return output.String()
}

func applyParagraphStyle(paragraph *textParagraph, style paragraphStyle) {
	if style.HasMarginLeft && !paragraph.HasMarginLeft {
		paragraph.HasMarginLeft = true
		paragraph.MarginLeft = style.MarginLeft
	}
	if style.HasMarginRight && !paragraph.HasMarginRight {
		paragraph.HasMarginRight = true
		paragraph.MarginRight = style.MarginRight
	}
	if style.HasIndent && !paragraph.HasIndent {
		paragraph.HasIndent = true
		paragraph.Indent = style.Indent
	}
	if style.HasDefaultTab && !paragraph.HasDefaultTab {
		paragraph.HasDefaultTab = true
		paragraph.DefaultTabSize = style.DefaultTabSize
	}
	if !paragraph.HasSpaceBefore && style.HasSpaceBefore {
		paragraph.HasSpaceBefore = true
		paragraph.SpaceBefore = style.SpaceBefore
		paragraph.SpaceBeforePct = style.SpaceBeforePct
	}
	if !paragraph.HasSpaceAfter && style.HasSpaceAfter {
		paragraph.HasSpaceAfter = true
		paragraph.SpaceAfter = style.SpaceAfter
		paragraph.SpaceAfterPct = style.SpaceAfterPct
	}
	if !paragraph.HasLineSpacing && style.HasLineSpacing {
		paragraph.HasLineSpacing = true
		paragraph.LineSpacingPct = style.LineSpacingPct
	}
	if paragraph.BulletSizeTx {
		paragraph.BulletFontSize = 0
		paragraph.BulletSizePct = 0
	} else if paragraph.BulletFontSize == 0 {
		paragraph.BulletFontSize = style.BulletFontSize
	}
	if paragraph.BulletSizeTx {
		// Local buSzTx blocks inherited fixed or percentage bullet sizing.
	} else if paragraph.BulletSizePct == 0 {
		paragraph.BulletSizePct = style.BulletSizePct
	}
	if style.BulletSizeTx && paragraph.BulletFontSize == 0 && paragraph.BulletSizePct == 0 {
		paragraph.BulletSizeTx = true
	}
	if paragraph.FontFamily == "" {
		paragraph.FontFamily = concreteParagraphFontFamily(style.FontFamily)
	}
	if paragraph.FontSize == 0 {
		paragraph.FontSize = style.FontSize
	}
	if !paragraph.HasBold && !paragraph.Bold && (style.HasBold || style.Bold) {
		paragraph.HasBold = style.HasBold
		paragraph.Bold = style.Bold
	}
	if !paragraph.HasItalic && !paragraph.Italic && (style.HasItalic || style.Italic) {
		paragraph.HasItalic = style.HasItalic
		paragraph.Italic = style.Italic
	}
	if !paragraph.HasTextCaps && style.HasTextCaps {
		paragraph.HasTextCaps = true
		paragraph.TextCaps = style.TextCaps
	}
	if !paragraph.HasCharSpacing && style.HasCharSpacing {
		paragraph.HasCharSpacing = true
		paragraph.CharSpacing = style.CharSpacing
	}
	if paragraph.TextAlign == "" {
		paragraph.TextAlign = style.TextAlign
	}
	if paragraph.FontAlign == "" {
		paragraph.FontAlign = style.FontAlign
	}
	if !paragraph.HasRTL && style.HasRTL {
		paragraph.HasRTL = true
		paragraph.RTL = style.RTL
	}
	if !paragraph.HasEALineBreak && style.HasEALineBreak {
		paragraph.HasEALineBreak = true
		paragraph.EALineBreak = style.EALineBreak
	}
	if !paragraph.HasLatinLineBreak && style.HasLatinLineBreak {
		paragraph.HasLatinLineBreak = true
		paragraph.LatinLineBreak = style.LatinLineBreak
	}
	if !paragraph.HasHangingPunct && style.HasHangingPunct {
		paragraph.HasHangingPunct = true
		paragraph.HangingPunct = style.HangingPunct
	}
	if !paragraph.HasTextColor && style.HasTextColor {
		paragraph.HasTextColor = true
		paragraph.TextColor = style.TextColor
	}
	if paragraph.NoBullet {
		paragraph.Bullet = ""
		return
	}
	if style.NoBullet {
		paragraph.NoBullet = true
		paragraph.Bullet = ""
	} else if style.Bullet != "" && (paragraph.Bullet == "" || paragraph.Bullet == "•") {
		paragraph.Bullet = style.Bullet
	}
	if paragraph.BulletFontFamily == "" && !paragraph.BulletFontTx {
		paragraph.BulletFontFamily = style.BulletFontFamily
	}
	if style.BulletFontTx && paragraph.BulletFontFamily == "" && !paragraph.BulletFontTx {
		paragraph.BulletFontTx = true
		paragraph.BulletFontFamily = ""
	}
	if style.HasBulletColor && !paragraph.HasBulletColor && !paragraph.BulletColorTx {
		paragraph.HasBulletColor = true
		paragraph.BulletColor = style.BulletColor
	}
	if style.BulletColorTx && !paragraph.HasBulletColor && !paragraph.BulletColorTx {
		paragraph.BulletColorTx = true
		paragraph.HasBulletColor = false
		paragraph.BulletColor = color.RGBA{}
	}
}

func paragraphStylesFromListStyle(listStyle *xmlNode, theme themeColors) map[int]paragraphStyle {
	styles := map[int]paragraphStyle{}
	if listStyle == nil {
		return styles
	}
	for _, child := range listStyle.Children {
		level, ok := paragraphLevelFromStyleNode(child.Name)
		if !ok {
			continue
		}
		styles[level] = parseParagraphStyle(child, theme)
	}
	return styles
}

func paragraphLevelFromStyleNode(name string) (int, bool) {
	if name == "defPPr" {
		return 0, true
	}
	if !strings.HasPrefix(name, "lvl") || !strings.HasSuffix(name, "pPr") {
		return 0, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(name, "lvl"), "pPr")
	level, err := strconv.Atoi(raw)
	if err != nil || level < 1 {
		return 0, false
	}
	return level - 1, true
}

func parseParagraphStyle(node *xmlNode, theme themeColors) paragraphStyle {
	var style paragraphStyle
	style.TextAlign = attrValue(node.Attrs, "algn")
	style.FontAlign = attrValue(node.Attrs, "fontAlgn")
	applyParagraphStylePropertyFlags(&style, node)
	if value := attrValue(node.Attrs, "marL"); value != "" {
		style.HasMarginLeft = true
		style.MarginLeft = parseIntAttr(node.Attrs, "marL")
	}
	if value := attrValue(node.Attrs, "marR"); value != "" {
		style.HasMarginRight = true
		style.MarginRight = parseIntAttr(node.Attrs, "marR")
	}
	if value := attrValue(node.Attrs, "indent"); value != "" {
		style.HasIndent = true
		style.Indent = parseIntAttr(node.Attrs, "indent")
	}
	if value := attrValue(node.Attrs, "defTabSz"); value != "" {
		style.DefaultTabSize = parseIntAttr(node.Attrs, "defTabSz")
		style.HasDefaultTab = style.DefaultTabSize > 0
	}
	style.BulletFontFamily = bulletFontFamilyFromProperties(node)
	if bullet := firstChild(node, "buChar"); bullet != nil {
		style.Bullet = normalizeBulletCharForFont(attrValue(bullet.Attrs, "char"), style.BulletFontFamily)
	}
	if bulletColorNode := firstChild(node, "buClr"); bulletColorNode != nil {
		if bulletColor, ok := colorFromColorNodeWithTheme(bulletColorNode, theme); ok {
			style.HasBulletColor = true
			style.BulletColor = bulletColor
		}
	}
	if firstChild(node, "buClrTx") != nil {
		style.BulletColorTx = true
		style.HasBulletColor = false
		style.BulletColor = color.RGBA{}
	}
	if firstChild(node, "buFontTx") != nil {
		style.BulletFontTx = true
		style.BulletFontFamily = ""
	}
	if autoNum := firstChild(node, "buAutoNum"); autoNum != nil {
		style.HasAutoNumber = true
		style.Bullet = ""
		style.NoBullet = false
		style.AutoNumberType = attrValue(autoNum.Attrs, "type")
		if startAt := int(parseIntAttr(autoNum.Attrs, "startAt")); startAt > 0 {
			style.AutoNumberStart = startAt
		}
	}
	if firstChild(node, "buNone") != nil {
		style.NoBullet = true
		style.Bullet = ""
	}
	if spcBef := firstChild(node, "spcBef"); spcBef != nil {
		style.HasSpaceBefore = true
		style.SpaceBefore, style.SpaceBeforePct = parseSpacingValue(spcBef)
	}
	if spcAft := firstChild(node, "spcAft"); spcAft != nil {
		style.HasSpaceAfter = true
		style.SpaceAfter, style.SpaceAfterPct = parseSpacingValue(spcAft)
	}
	if lnSpc := firstChild(node, "lnSpc"); lnSpc != nil {
		style.HasLineSpacing = true
		style.LineSpacingPct = parseSpacingPercent(lnSpc)
	}
	if defRPr := firstChild(node, "defRPr"); defRPr != nil {
		style.FontFamily = concreteParagraphFontFamily(latinTypefaceFromRunProperties(defRPr))
		if size := parseIntAttr(defRPr.Attrs, "sz"); size > 0 {
			style.FontSize = int(size)
		}
		if value := attrValue(defRPr.Attrs, "b"); value != "" {
			style.HasBold = true
			style.Bold = boolAttrOn(value)
		}
		if value := attrValue(defRPr.Attrs, "i"); value != "" {
			style.HasItalic = true
			style.Italic = boolAttrOn(value)
		}
		if value := textCapsType(attrValue(defRPr.Attrs, "cap")); value != "" {
			style.HasTextCaps = true
			style.TextCaps = value
		}
		if value := attrValue(defRPr.Attrs, "spc"); value != "" {
			style.HasCharSpacing = true
			style.CharSpacing = int(parseIntAttr(defRPr.Attrs, "spc"))
		}
		if solidFill := firstChild(defRPr, "solidFill"); solidFill != nil {
			if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				style.HasTextColor = true
				style.TextColor = textColor
			}
		}
	}
	applyBulletSizePropertiesToParagraphStyle(&style, node)
	return style
}

func applyParagraphPropertyFlags(paragraph *textParagraph, node *xmlNode) {
	if value := attrValue(node.Attrs, "rtl"); value != "" {
		paragraph.HasRTL = true
		paragraph.RTL = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "eaLnBrk"); value != "" {
		paragraph.HasEALineBreak = true
		paragraph.EALineBreak = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "latinLnBrk"); value != "" {
		paragraph.HasLatinLineBreak = true
		paragraph.LatinLineBreak = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "hangingPunct"); value != "" {
		paragraph.HasHangingPunct = true
		paragraph.HangingPunct = boolAttrOn(value)
	}
}

func applyParagraphStylePropertyFlags(style *paragraphStyle, node *xmlNode) {
	if value := attrValue(node.Attrs, "rtl"); value != "" {
		style.HasRTL = true
		style.RTL = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "eaLnBrk"); value != "" {
		style.HasEALineBreak = true
		style.EALineBreak = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "latinLnBrk"); value != "" {
		style.HasLatinLineBreak = true
		style.LatinLineBreak = boolAttrOn(value)
	}
	if value := attrValue(node.Attrs, "hangingPunct"); value != "" {
		style.HasHangingPunct = true
		style.HangingPunct = boolAttrOn(value)
	}
}

func applyBulletSizePropertiesToParagraphStyle(style *paragraphStyle, node *xmlNode) {
	if firstChild(node, "buSzTx") != nil {
		style.BulletSizeTx = true
		style.BulletFontSize = 0
		style.BulletSizePct = 0
		return
	}
	if bulletSize := firstChild(node, "buSzPts"); bulletSize != nil {
		if size := int(parseIntAttr(bulletSize.Attrs, "val")); size > 0 {
			style.BulletSizeTx = false
			style.BulletFontSize = size
			style.BulletSizePct = 0
		}
	}
	if bulletSize := firstChild(node, "buSzPct"); bulletSize != nil && style.BulletFontSize == 0 {
		if pct := int(parseIntAttr(bulletSize.Attrs, "val")); pct > 0 {
			style.BulletSizeTx = false
			style.BulletSizePct = pct
		}
	}
}

func applyBulletSizePropertiesToParagraph(paragraph *textParagraph, node *xmlNode) {
	if firstChild(node, "buSzTx") != nil {
		paragraph.BulletSizeTx = true
		paragraph.BulletFontSize = 0
		paragraph.BulletSizePct = 0
		return
	}
	if bulletSize := firstChild(node, "buSzPts"); bulletSize != nil {
		if size := int(parseIntAttr(bulletSize.Attrs, "val")); size > 0 {
			paragraph.BulletSizeTx = false
			paragraph.BulletFontSize = size
			paragraph.BulletSizePct = 0
		}
	}
	if bulletSize := firstChild(node, "buSzPct"); bulletSize != nil && paragraph.BulletFontSize == 0 {
		if pct := int(parseIntAttr(bulletSize.Attrs, "val")); pct > 0 {
			paragraph.BulletSizeTx = false
			paragraph.BulletSizePct = pct
		}
	}
}

func normalizeBulletChar(raw string) string {
	return normalizeBulletCharForFont(raw, "")
}

func normalizeBulletCharForFont(raw string, fontFamily string) string {
	font := strings.ToLower(strings.TrimSpace(fontFamily))
	switch raw {
	case "§":
		return "▪"
	case "Ø":
		if strings.Contains(font, "wingdings") {
			return "¬"
		}
		return raw
	case "\uf075":
		return "▶"
	default:
		if strings.Contains(font, "wingdings") && exactFontFamilyAvailable(fontFamily) {
			if mapped := legacySymbolBulletPrivateUseChar(raw); mapped != "" {
				return mapped
			}
		}
		return raw
	}
}

func legacySymbolBulletPrivateUseChar(raw string) string {
	runes := []rune(raw)
	if len(runes) != 1 || runes[0] > 0xff {
		return ""
	}
	return string(rune(0xf000) + runes[0])
}

func bulletFontFamilyFromProperties(node *xmlNode) string {
	if bulletFont := firstChild(node, "buFont"); bulletFont != nil {
		return attrValue(bulletFont.Attrs, "typeface")
	}
	return ""
}

func parseSpacingPixels(node *xmlNode) int {
	pixels, _ := parseSpacingValue(node)
	return pixels
}

func parseSpacingValue(node *xmlNode) (int, int) {
	if spcPts := firstChild(node, "spcPts"); spcPts != nil {
		points100 := parseIntAttr(spcPts.Attrs, "val")
		if points100 <= 0 {
			return 0, 0
		}
		return int(math.Round(float64(points100) / 100 * defaultOutputDPI / 72)), 0
	}
	if spcPct := firstChild(node, "spcPct"); spcPct != nil {
		pct := int(parsePercentAttr(spcPct.Attrs, "val"))
		if pct <= 0 {
			return 0, 0
		}
		return 0, pct
	}
	return 0, 0
}

func parseSpacingPercent(node *xmlNode) int {
	if spcPct := firstChild(node, "spcPct"); spcPct != nil {
		value := int(parsePercentAttr(spcPct.Attrs, "val"))
		if value > 0 {
			return value
		}
	}
	return 0
}

func paragraphTextRuns(paragraphNode *xmlNode) []textRun {
	return paragraphTextRunsWithTheme(paragraphNode, defaultThemeColors())
}

func paragraphTextRunsWithTheme(paragraphNode *xmlNode, theme themeColors) []textRun {
	var runs []textRun
	for _, child := range paragraphNode.Children {
		switch child.Name {
		case "r", "fld":
			text := textFromNode(child)
			if text == "" {
				continue
			}
			runs = append(runs, textRunFromNodeWithTheme(child, text, theme))
		case "br":
			runs = append(runs, textRunFromNodeWithTheme(child, "\n", theme))
		}
	}
	return trimTextRuns(runs)
}

func textRunFromNode(node *xmlNode, text string) textRun {
	return textRunFromNodeWithTheme(node, text, defaultThemeColors())
}

func textRunFromNodeWithTheme(node *xmlNode, text string, theme themeColors) textRun {
	run := textRun{Text: text}
	if node.Name == "fld" {
		run.FieldType = attrValue(node.Attrs, "type")
	}
	if rPr := firstChild(node, "rPr"); rPr != nil {
		applyRunPropertiesToRun(&run, rPr, text, theme)
	}
	return run
}

func resolveTextFields(elements []slideElement, slideNumber int) []slideElement {
	if slideNumber <= 0 {
		return elements
	}
	for index := range elements {
		if textParagraphsContainFields(elements[index].TextParagraphs) {
			elements[index].TextParagraphs = resolveTextParagraphFields(elements[index].TextParagraphs, slideNumber)
			elements[index].Text = textFromParagraphs(elements[index].TextParagraphs)
		}
		if elements[index].HasTable {
			for rowIndex := range elements[index].Table.Rows {
				for cellIndex := range elements[index].Table.Rows[rowIndex].Cells {
					cell := &elements[index].Table.Rows[rowIndex].Cells[cellIndex]
					if textParagraphsContainFields(cell.TextParagraphs) {
						cell.TextParagraphs = resolveTextParagraphFields(cell.TextParagraphs, slideNumber)
						cell.Text = textFromParagraphs(cell.TextParagraphs)
					}
				}
			}
		}
	}
	return elements
}

func textParagraphsContainFields(paragraphs []textParagraph) bool {
	for _, paragraph := range paragraphs {
		for _, run := range paragraph.Runs {
			if run.FieldType != "" {
				return true
			}
		}
	}
	return false
}

func resolveTextParagraphFields(paragraphs []textParagraph, slideNumber int) []textParagraph {
	for paragraphIndex := range paragraphs {
		for runIndex := range paragraphs[paragraphIndex].Runs {
			run := &paragraphs[paragraphIndex].Runs[runIndex]
			if strings.EqualFold(run.FieldType, "slidenum") {
				run.Text = strconv.Itoa(slideNumber)
			}
		}
		paragraphs[paragraphIndex].Text = textFromRuns(paragraphs[paragraphIndex].Runs)
	}
	return paragraphs
}

func textFromParagraphs(paragraphs []textParagraph) string {
	var parts []string
	for _, paragraph := range paragraphs {
		text := strings.TrimSpace(paragraph.Text)
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

func textFromRuns(runs []textRun) string {
	var builder strings.Builder
	for _, run := range runs {
		builder.WriteString(run.Text)
	}
	return strings.TrimSpace(builder.String())
}

func applyRunPropertiesToParagraph(paragraph *textParagraph, rPr *xmlNode, theme themeColors) {
	var run textRun
	applyRunPropertiesToRun(&run, rPr, "", theme)
	paragraph.FontFamily = concreteParagraphFontFamily(run.FontFamily)
	paragraph.Language = run.Language
	paragraph.FontSize = run.FontSize
	paragraph.HasBold = run.HasBold
	paragraph.Bold = run.Bold
	paragraph.HasItalic = run.HasItalic
	paragraph.Italic = run.Italic
	paragraph.HasTextCaps = run.HasTextCaps
	paragraph.TextCaps = run.TextCaps
	paragraph.HasCharSpacing = run.HasCharSpacing
	paragraph.CharSpacing = run.CharSpacing
	paragraph.HasTextColor = run.HasTextColor
	paragraph.TextColor = run.TextColor
}

func applyRunPropertiesToParagraphDefaults(paragraph *textParagraph, rPr *xmlNode, theme themeColors) {
	var run textRun
	applyRunPropertiesToRun(&run, rPr, "", theme)
	if paragraph.FontFamily == "" {
		paragraph.FontFamily = concreteParagraphFontFamily(run.FontFamily)
	}
	if paragraph.Language == "" {
		paragraph.Language = run.Language
	}
	if paragraph.FontSize == 0 {
		paragraph.FontSize = run.FontSize
	}
	if run.HasBold {
		paragraph.HasBold = true
		paragraph.Bold = run.Bold
	}
	if run.HasItalic {
		paragraph.HasItalic = true
		paragraph.Italic = run.Italic
	}
	if !paragraph.HasTextCaps && run.HasTextCaps {
		paragraph.HasTextCaps = true
		paragraph.TextCaps = run.TextCaps
	}
	if !paragraph.HasCharSpacing && run.HasCharSpacing {
		paragraph.HasCharSpacing = true
		paragraph.CharSpacing = run.CharSpacing
	}
	if !paragraph.HasTextColor && run.HasTextColor {
		paragraph.HasTextColor = true
		paragraph.TextColor = run.TextColor
	}
}

func concreteParagraphFontFamily(fontFamily string) string {
	trimmed := strings.TrimSpace(fontFamily)
	return trimmed
}

func applyRunPropertiesToRun(run *textRun, rPr *xmlNode, text string, theme themeColors) {
	run.Language = attrValue(rPr.Attrs, "lang")
	run.FontSize = int(parseIntAttr(rPr.Attrs, "sz"))
	run.FontFamily = typefaceFromRunPropertiesForText(rPr, text)
	if value := attrValue(rPr.Attrs, "b"); value != "" {
		run.HasBold = true
		run.Bold = boolAttrOn(value)
	}
	if value := attrValue(rPr.Attrs, "i"); value != "" {
		run.HasItalic = true
		run.Italic = boolAttrOn(value)
	}
	run.Underline = runPropertiesUnderline(rPr)
	run.Strike = textStrikeType(attrValue(rPr.Attrs, "strike"))
	if value := textCapsType(attrValue(rPr.Attrs, "cap")); value != "" {
		run.HasTextCaps = true
		run.TextCaps = value
	}
	run.Baseline = int(parseIntAttr(rPr.Attrs, "baseline"))
	if value := attrValue(rPr.Attrs, "kern"); value != "" {
		run.HasKern = true
		run.KernMinFontSize = int(parseIntAttr(rPr.Attrs, "kern"))
	}
	if value := attrValue(rPr.Attrs, "spc"); value != "" {
		run.HasCharSpacing = true
		run.CharSpacing = int(parseIntAttr(rPr.Attrs, "spc"))
	}
	if solidFill := firstChild(rPr, "solidFill"); solidFill != nil {
		if textColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
			run.HasTextColor = true
			run.TextColor = textColor
		}
	}
	if highlight := firstChild(rPr, "highlight"); highlight != nil {
		if highlightColor, ok := colorFromColorNodeWithTheme(highlight, theme); ok {
			run.HasHighlightColor = true
			run.HighlightColor = highlightColor
		}
	}
	if underlineFill := firstChild(rPr, "uFill"); underlineFill != nil {
		if solidFill := firstChild(underlineFill, "solidFill"); solidFill != nil {
			if underlineColor, ok := colorFromSolidFillWithTheme(solidFill, theme); ok {
				run.HasUnderlineColor = true
				run.UnderlineColor = underlineColor
			}
		} else if underlineColor, ok := colorFromColorNodeWithTheme(underlineFill, theme); ok {
			run.HasUnderlineColor = true
			run.UnderlineColor = underlineColor
		}
	}
}

func typefaceFromRunPropertiesForText(rPr *xmlNode, text string) string {
	if textUsesSymbolTypeface(text) {
		if typeface := typefaceFromChild(rPr, "sym"); typeface != "" {
			return typeface
		}
	}
	if typeface := latinTypefaceFromRunProperties(rPr); typeface != "" {
		return typeface
	}
	if !textNeedsAlternateTypeface(text) {
		return ""
	}
	for _, name := range []string{"ea", "cs", "sym"} {
		if typeface := typefaceFromChild(rPr, name); typeface != "" {
			return typeface
		}
	}
	return ""
}

func latinTypefaceFromRunProperties(rPr *xmlNode) string {
	return typefaceFromChild(rPr, "latin")
}

func typefaceFromChild(node *xmlNode, name string) string {
	child := firstChild(node, name)
	if child == nil {
		return ""
	}
	typeface := attrValue(child.Attrs, "typeface")
	if strings.TrimSpace(typeface) == "" {
		return ""
	}
	return typeface
}

func textNeedsAlternateTypeface(text string) bool {
	for _, r := range text {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func textUsesSymbolTypeface(text string) bool {
	hasSymbol := false
	for _, r := range text {
		if isPrivateUseRune(r) {
			hasSymbol = true
			continue
		}
		if unicode.IsSpace(r) {
			continue
		}
		return false
	}
	return hasSymbol
}

func isPrivateUseRune(r rune) bool {
	return (r >= '\uE000' && r <= '\uF8FF') || (r >= '\U000F0000' && r <= '\U000FFFFD') || (r >= '\U00100000' && r <= '\U0010FFFD')
}

func isUnderlineStyle(value string) bool {
	return value != "" && value != "none"
}

func runPropertiesUnderline(rPr *xmlNode) bool {
	if value := attrValue(rPr.Attrs, "u"); value != "" {
		return isUnderlineStyle(value)
	}
	if underline := firstChild(rPr, "uLn"); underline != nil {
		return firstChild(underline, "noFill") == nil
	}
	return false
}

func textStrikeType(value string) string {
	switch value {
	case "sngStrike", "dblStrike":
		return value
	default:
		return ""
	}
}

func textCapsType(value string) string {
	switch strings.TrimSpace(value) {
	case "none", "small", "all":
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func textParagraphsHaveRunColor(paragraphs []textParagraph) bool {
	for _, paragraph := range paragraphs {
		for _, run := range paragraph.Runs {
			if run.HasTextColor {
				return true
			}
		}
	}
	return false
}

func textRunsHaveRunMetricProperties(runs []textRun) bool {
	for _, run := range runs {
		if run.FontSize != 0 || strings.TrimSpace(run.FontFamily) != "" || run.HasBold || run.HasItalic || run.Underline || run.Strike != "" || run.HasTextCaps || run.Baseline != 0 || run.HasCharSpacing || run.HasKern {
			return true
		}
	}
	return false
}

func trimTextRuns(runs []textRun) []textRun {
	start := 0
	for start < len(runs) {
		runs[start].Text = strings.TrimLeft(runs[start].Text, "\r\n")
		if runs[start].Text != "" {
			break
		}
		start++
	}
	end := len(runs)
	for end > start {
		runs[end-1].Text = strings.TrimRight(runs[end-1].Text, "\r\n")
		if runs[end-1].Text != "" {
			break
		}
		end--
	}
	if start >= end {
		return nil
	}
	return runs[start:end]
}

func textRunsFontSize(runs []textRun) int {
	fontSize := 0
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		if run.FontSize <= 0 {
			return 0
		}
		if fontSize != 0 && fontSize != run.FontSize {
			return 0
		}
		fontSize = run.FontSize
	}
	return fontSize
}

func textParagraphsFontSize(paragraphs []textParagraph) int {
	fontSize := 0
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Text) == "" {
			continue
		}
		size := paragraph.FontSize
		if size <= 0 {
			size = textRunsFontSize(paragraph.Runs)
		}
		if size <= 0 {
			return 0
		}
		if fontSize != 0 && fontSize != size {
			return 0
		}
		fontSize = size
	}
	return fontSize
}

func textParagraphsTextAlign(paragraphs []textParagraph) string {
	align := ""
	for _, paragraph := range paragraphs {
		if strings.TrimSpace(paragraph.Text) == "" || paragraph.TextAlign == "" {
			continue
		}
		if align != "" && align != paragraph.TextAlign {
			return ""
		}
		align = paragraph.TextAlign
	}
	return align
}

func textRunsAllBold(runs []textRun) bool {
	seenTextRun := false
	allBold := true
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		seenTextRun = true
		if !run.Bold {
			allBold = false
		}
	}
	return seenTextRun && allBold
}

func textRunsAllItalic(runs []textRun) bool {
	seenTextRun := false
	allItalic := true
	for _, run := range runs {
		if strings.TrimSpace(run.Text) == "" {
			continue
		}
		seenTextRun = true
		if !run.Italic {
			allItalic = false
		}
	}
	return seenTextRun && allItalic
}
