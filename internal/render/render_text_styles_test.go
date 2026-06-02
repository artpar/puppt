package render

import (
	"image"
	"image/color"
	"image/draw"
	"slices"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func TestParseTextStylesNormalizesOfficeSymbolBullets(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buClr><a:srgbClr val="70AD47"/></a:buClr>
        <a:buFont typeface="Wingdings 3"/>
        <a:buSzPct val="80000"/>
        <a:buChar char="&#xF075;"/>
        <a:defRPr sz="1800" b="1"><a:solidFill><a:srgbClr val="112233"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
	</p:sldMaster>`), defaultThemeColors())
	style := styles["body"].ParagraphStyles[0]
	if style.Bullet != "▶" {
		t.Fatalf("unexpected symbol bullet, got %+v", style)
	}
	if style.BulletFontFamily != "Wingdings 3" {
		t.Fatalf("expected paragraph bullet font family, got %+v", style)
	}
	if style.FontSize != 1800 {
		t.Fatalf("expected paragraph defRPr font size, got %+v", style)
	}
	if style.FontFamily != "Arial" {
		t.Fatalf("expected paragraph defRPr font family, got %+v", style)
	}
	if style.BulletSizePct != 80000 {
		t.Fatalf("expected paragraph bullet size percent, got %+v", style)
	}
	if !style.HasBold || !style.Bold || !style.HasTextColor || style.TextColor != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 0xff}) {
		t.Fatalf("expected paragraph defRPr bold and text color, got %+v", style)
	}
	if !style.HasBulletColor || style.BulletColor != (color.RGBA{R: 0x70, G: 0xad, B: 0x47, A: 0xff}) {
		t.Fatalf("expected parsed bullet color, got %+v", style)
	}
}

func TestParseTextStylesCapturesBulletFollowTextProperties(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buClrTx/>
        <a:buFontTx/>
        <a:buChar char="•"/>
        <a:defRPr sz="1800"><a:solidFill><a:srgbClr val="112233"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())
	style := styles["body"].ParagraphStyles[0]
	if !style.BulletColorTx || !style.BulletFontTx || style.HasBulletColor || style.BulletFontFamily != "" {
		t.Fatalf("expected bullet follow-text properties, got %+v", style)
	}
}

func TestParseTextStylesCapturesBulletSizeFollowText(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buSzTx/>
        <a:buChar char="•"/>
        <a:defRPr sz="2200"/>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())
	style := styles["body"].ParagraphStyles[0]
	if !style.BulletSizeTx || style.BulletFontSize != 0 || style.BulletSizePct != 0 {
		t.Fatalf("expected buSzTx to make bullet size follow text, got %+v", style)
	}
}

func TestParseTextStylesCapturesAutoNumberBulletProperties(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
  <p:txStyles>
    <p:bodyStyle>
      <a:lvl1pPr>
        <a:buAutoNum type="arabicParenR" startAt="2"/>
        <a:defRPr sz="1800"/>
      </a:lvl1pPr>
    </p:bodyStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())

	style := styles["body"].ParagraphStyles[0]
	if !style.HasAutoNumber || style.AutoNumberType != "arabicParenR" || style.AutoNumberStart != 2 || style.Bullet != "" || style.NoBullet {
		t.Fatalf("expected auto-number bullet properties, got %+v", style)
	}
}

func TestMergeParagraphStyleRespectsAutoNumberPrecedence(t *testing.T) {
	autoNumber := paragraphStyle{
		HasAutoNumber:   true,
		AutoNumberType:  "alphaLcPeriod",
		AutoNumberStart: 3,
	}

	fromBase := mergeParagraphStyle(autoNumber, paragraphStyle{})
	if !fromBase.HasAutoNumber || fromBase.AutoNumberType != "alphaLcPeriod" || fromBase.AutoNumberStart != 3 {
		t.Fatalf("expected empty override to inherit auto-number style, got %+v", fromBase)
	}

	explicitBullet := mergeParagraphStyle(autoNumber, paragraphStyle{Bullet: "•"})
	if explicitBullet.HasAutoNumber || explicitBullet.Bullet != "•" {
		t.Fatalf("explicit bullet should block inherited auto-number style, got %+v", explicitBullet)
	}

	autoOverride := mergeParagraphStyle(paragraphStyle{Bullet: "•"}, autoNumber)
	if !autoOverride.HasAutoNumber || autoOverride.Bullet != "" {
		t.Fatalf("auto-number override should block inherited bullet style, got %+v", autoOverride)
	}
}

func TestMergeParagraphStyleHonorsExplicitNonBoldOverride(t *testing.T) {
	got := mergeParagraphStyle(
		paragraphStyle{HasBold: true, Bold: true, HasItalic: true, Italic: true},
		paragraphStyle{HasBold: true, Bold: false, HasItalic: true, Italic: false},
	)
	if !got.HasBold || got.Bold || !got.HasItalic || got.Italic {
		t.Fatalf("explicit false text style should block inherited true style: %+v", got)
	}
}

func TestApplyParagraphStyleHonorsExplicitNonBoldParagraph(t *testing.T) {
	paragraph := textParagraph{HasBold: true, Bold: false, HasItalic: true, Italic: false}
	applyParagraphStyle(&paragraph, paragraphStyle{HasBold: true, Bold: true, HasItalic: true, Italic: true})
	if paragraph.Bold || paragraph.Italic {
		t.Fatalf("explicit paragraph non-bold/non-italic should block inherited true style: %+v", paragraph)
	}
}

func TestApplyParagraphStyleKeepsLocalExplicitBulletProperties(t *testing.T) {
	paragraph := textParagraph{
		BulletFontFamily: "Arial",
		HasBulletColor:   true,
		BulletColor:      color.RGBA{R: 1, G: 2, B: 3, A: 255},
	}
	applyParagraphStyle(&paragraph, paragraphStyle{
		BulletFontTx:  true,
		BulletColorTx: true,
	})
	if paragraph.BulletFontTx || paragraph.BulletFontFamily != "Arial" {
		t.Fatalf("local explicit bullet font should win over inherited buFontTx: %+v", paragraph)
	}
	if paragraph.BulletColorTx || !paragraph.HasBulletColor || paragraph.BulletColor.R != 1 {
		t.Fatalf("local explicit bullet color should win over inherited buClrTx: %+v", paragraph)
	}
}

func TestApplyParagraphStyleKeepsLocalBulletSizeFollowText(t *testing.T) {
	paragraph := textParagraph{
		Bullet:       "•",
		FontSize:     2200,
		BulletSizeTx: true,
	}

	applyParagraphStyle(&paragraph, paragraphStyle{BulletSizePct: 80000})

	if !paragraph.BulletSizeTx || paragraph.BulletSizePct != 0 || paragraph.BulletFontSize != 0 {
		t.Fatalf("local buSzTx should block inherited bullet size, got %+v", paragraph)
	}
	if got := bulletSegmentFontSize(paragraph); got != 2200 {
		t.Fatalf("buSzTx bullet should follow paragraph font size, got %d", got)
	}
}

func TestApplyParagraphStyleInheritsBulletSizeFollowText(t *testing.T) {
	paragraph := textParagraph{
		Bullet:   "•",
		FontSize: 1800,
	}

	applyParagraphStyle(&paragraph, paragraphStyle{BulletSizeTx: true})

	if !paragraph.BulletSizeTx || paragraph.BulletSizePct != 0 || paragraph.BulletFontSize != 0 {
		t.Fatalf("expected inherited buSzTx bullet size, got %+v", paragraph)
	}
	if got := bulletSegmentFontSize(paragraph); got != 1800 {
		t.Fatalf("inherited buSzTx bullet should follow paragraph font size, got %d", got)
	}
}

func TestTextParagraphsFromNodeUsesParagraphDefaultRunProperties(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800" b="1"><a:solidFill><a:srgbClr val="336699"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
    <a:r><a:rPr><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:rPr><a:t> Red</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "Arial" || got[0].FontSize != 1800 || !got[0].Bold || !got[0].HasTextColor || got[0].TextColor != (color.RGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Fatalf("paragraph default run properties were not applied: %+v", got[0])
	}
	defaultSegment := runToSegment(got[0].Runs[0], got[0])
	if defaultSegment.FontFamily != "Arial" || !defaultSegment.Bold || !defaultSegment.HasTextColor || defaultSegment.TextColor != got[0].TextColor {
		t.Fatalf("paragraph defaults were not carried to unstyled run: %+v", defaultSegment)
	}
	redSegment := runToSegment(got[0].Runs[1], got[0])
	if !redSegment.HasTextColor || redSegment.TextColor.R != 0xff || redSegment.TextColor.G != 0 || redSegment.TextColor.B != 0 {
		t.Fatalf("explicit run color should win over paragraph default: %+v", redSegment)
	}
}

func TestTextParagraphsFromNodeHonorsExplicitParagraphDefaultNonBold(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:lstStyle>
    <a:lvl1pPr><a:defRPr sz="1800" b="1" i="1"/></a:lvl1pPr>
  </a:lstStyle>
  <a:p>
    <a:pPr><a:defRPr b="0" i="0"/></a:pPr>
    <a:r><a:rPr/><a:t>Not bold</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if segment.Bold || segment.Italic {
		t.Fatalf("explicit paragraph default b=0/i=0 should block inherited true style: %+v", segment)
	}
}

func TestTextParagraphsFromNodeDoesNotUseDefaultAlternateTypefaceWithoutRunText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "" {
		t.Fatalf("paragraph default alternate typeface should not apply without script-specific text: %+v", got[0])
	}
}

func TestParagraphDefaultRunPropertiesPreserveThemeFontTokens(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr sz="1800"><a:latin typeface="+mn-lt"/></a:defRPr></a:pPr>
    <a:r><a:rPr/><a:t>Defaulted</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraphs: %+v", got)
	}
	if got[0].FontFamily != "+mn-lt" {
		t.Fatalf("theme font token should be preserved until theme resolution: %+v", got[0])
	}
}

func TestApplyThemeFontFamiliesResolvesParagraphDefaults(t *testing.T) {
	got := applyThemeFontFamilies([]slideElement{{
		Text: "Defaulted",
		TextParagraphs: []textParagraph{{
			FontFamily: "Arial",
			Runs: []textRun{
				{Text: "Defaulted"},
				{Text: " Explicit", FontFamily: "+mj-lt"},
			},
		}},
	}}, themeFonts{MajorLatin: "Trebuchet MS", MinorLatin: "Arial"})
	paragraph := got[0].TextParagraphs[0]
	if paragraph.FontFamily != "Arial" || paragraph.Runs[1].FontFamily != "Trebuchet MS" {
		t.Fatalf("theme font families were not resolved: %+v", paragraph)
	}
	if segment := runToSegment(paragraph.Runs[0], paragraph); segment.FontFamily != "Arial" {
		t.Fatalf("paragraph font family was not used for unstyled run: %+v", segment)
	}
}

func TestInheritedTextStylesResolveThemeFontFamiliesAfterApplication(t *testing.T) {
	elements := []slideElement{{
		Text:            "Title",
		TextParagraphs:  []textParagraph{{Text: "Title"}},
		IsPlaceholder:   true,
		PlaceholderType: "ctrTitle",
	}}
	elements = applyInheritedTextStyles(elements, map[string]textStyle{
		"ctrTitle": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {FontFamily: "+mj-lt"},
			},
		},
	})
	got := applyThemeFontFamilies(elements, themeFonts{MajorLatin: "Calibri Light", MinorLatin: "Calibri"})
	if got[0].TextParagraphs[0].FontFamily != "Calibri Light" {
		t.Fatalf("inherited theme font token was not resolved after style application: %+v", got[0].TextParagraphs[0])
	}
}

func TestApplyInheritedTextStylesPreservesExplicitParagraphFontSize(t *testing.T) {
	elements := []slideElement{{
		Text:            "Nested",
		TextParagraphs:  []textParagraph{{Text: "Nested", Level: 1, FontSize: 1400}},
		IsPlaceholder:   true,
		PlaceholderType: "body",
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"body": {
			ParagraphStyles: map[int]paragraphStyle{
				1: {FontSize: 1600},
			},
		},
	})
	if got[0].TextParagraphs[0].FontSize != 1400 {
		t.Fatalf("inherited paragraph style overrode explicit font size: %+v", got[0].TextParagraphs[0])
	}
}

func TestApplyInheritedTextStylesDoesNotOverrideElementTextColor(t *testing.T) {
	elements := []slideElement{{
		Text:         "Styled",
		HasTextColor: true,
		TextColor:    color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		TextParagraphs: []textParagraph{{
			Text:     "Styled",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Styled", FontSize: 1800}},
		}},
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"default": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {HasTextColor: true, TextColor: color.RGBA{A: 0xff}, FontSize: 2200},
			},
		},
	})
	if got[0].TextParagraphs[0].FontSize != 1800 {
		t.Fatalf("explicit paragraph font size should still win over inherited style, got %+v", got[0].TextParagraphs[0])
	}
	if got[0].TextParagraphs[0].HasTextColor {
		t.Fatalf("inherited paragraph color should not override element text color, got %+v", got[0].TextParagraphs[0])
	}
	segment := runToSegment(got[0].TextParagraphs[0].Runs[0], textParagraphWithElementDefaults(got[0].TextParagraphs[0], got[0]))
	if !segment.HasTextColor || segment.TextColor != (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}) {
		t.Fatalf("expected rendered run to inherit element text color, got %+v", segment)
	}
}

func TestParseTextStylesReadsMasterTitleDefaults(t *testing.T) {
	styles := parseTextStyles([]byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
	  <p:txStyles>
    <p:titleStyle>
      <a:lvl1pPr algn="ctr">
        <a:defRPr sz="4400" b="1">
          <a:solidFill><a:srgbClr val="0070C0"/></a:solidFill>
        </a:defRPr>
      </a:lvl1pPr>
    </p:titleStyle>
  </p:txStyles>
</p:sldMaster>`), defaultThemeColors())
	got, ok := styles["ctrTitle"]
	if !ok {
		t.Fatalf("expected ctrTitle style, got %+v", styles)
	}
	if got.FontSize != 4400 || !got.Bold || got.TextAlign != "ctr" || !got.HasTextColor || got.TextColor.R != 0x00 || got.TextColor.G != 0x70 || got.TextColor.B != 0xc0 {
		t.Fatalf("unexpected parsed title style: %+v", got)
	}
}

func TestApplyInheritedTextStylesDoesNotOverrideExplicitRunSize(t *testing.T) {
	elements := []slideElement{{
		Text:            "Title",
		IsPlaceholder:   true,
		PlaceholderType: "ctrTitle",
		FontSize:        3200,
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"ctrTitle": {
			FontSize:     4400,
			HasTextColor: true,
			TextColor:    color.RGBA{B: 255, A: 255},
			TextAlign:    "ctr",
		},
	})
	if got[0].FontSize != 3200 {
		t.Fatalf("inherited style overrode explicit font size: %+v", got[0])
	}
	if !got[0].HasTextColor || got[0].TextColor.B != 255 || got[0].TextAlign != "ctr" {
		t.Fatalf("inherited missing properties were not applied: %+v", got[0])
	}
}

func TestApplyInheritedTextStylesAppliesTitleButSkipsBodyPlaceholders(t *testing.T) {
	elements := []slideElement{
		{
			Text:            "Title",
			TextParagraphs:  []textParagraph{{Text: "Title"}},
			IsPlaceholder:   true,
			PlaceholderType: "title",
		},
		{
			Text:            "Body",
			IsPlaceholder:   true,
			PlaceholderType: "body",
		},
	}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"title": {
			FontSize:     4400,
			Bold:         true,
			HasTextColor: true,
			TextColor:    color.RGBA{B: 255, A: 255},
			ParagraphStyles: map[int]paragraphStyle{
				0: {HasLineSpacing: true, LineSpacingPct: 90000},
			},
		},
		"body": {
			FontSize:     2800,
			HasTextColor: true,
			TextColor:    color.RGBA{R: 255, A: 255},
		},
	})
	if got[0].FontSize != 4400 || !got[0].HasTextColor || got[0].TextColor.B != 255 || !got[0].TextParagraphs[0].Bold || got[0].TextParagraphs[0].LineSpacingPct != 90000 {
		t.Fatalf("title placeholder was not styled by inherited title fallback: %+v", got)
	}
	if got[1].FontSize != 0 || got[1].HasTextColor {
		t.Fatalf("body placeholder was unexpectedly styled by title-only fallback: %+v", got)
	}
}

func TestApplyInheritedTextStylesAppliesDefaultParagraphStyleToNonPlaceholderShapes(t *testing.T) {
	elements := []slideElement{{
		Text: "Bullet shape",
		TextParagraphs: []textParagraph{
			{Text: "Local", Level: 0, HasMarginLeft: true, MarginLeft: 285750, HasIndent: true, Indent: -285750},
			{Text: "Default", Level: 1, Bullet: "▪"},
		},
	}}
	got := applyInheritedTextStyles(elements, map[string]textStyle{
		"default": {
			ParagraphStyles: map[int]paragraphStyle{
				0: {HasMarginLeft: true, MarginLeft: 0},
				1: {HasMarginLeft: true, MarginLeft: 457200, HasDefaultTab: true, DefaultTabSize: 914400},
			},
		},
	})
	if got[0].TextParagraphs[0].MarginLeft != 285750 || got[0].TextParagraphs[0].Indent != -285750 {
		t.Fatalf("local paragraph geometry should win over default style, got %+v", got[0].TextParagraphs[0])
	}
	if !got[0].TextParagraphs[1].HasMarginLeft || got[0].TextParagraphs[1].MarginLeft != 457200 {
		t.Fatalf("default otherStyle paragraph margin was not inherited: %+v", got[0].TextParagraphs[1])
	}
	if !got[0].TextParagraphs[1].HasDefaultTab || got[0].TextParagraphs[1].DefaultTabSize != 914400 {
		t.Fatalf("default otherStyle tab size was not inherited: %+v", got[0].TextParagraphs[1])
	}
}

func TestInheritedTextStylesUsePresentationDefaultAsBase(t *testing.T) {
	pkg := &pptx.Package{
		PresentationPath: "ppt/presentation.xml",
		Parts: map[string][]byte{
			"ppt/presentation.xml": []byte(`<p:presentation xmlns:p="p" xmlns:a="a">
			  <p:defaultTextStyle>
			    <a:lvl1pPr marR="914400"><a:defRPr sz="1400"><a:solidFill><a:srgbClr val="112233"/></a:solidFill><a:latin typeface="Arial"/></a:defRPr></a:lvl1pPr>
			  </p:defaultTextStyle>
			</p:presentation>`),
			"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p" xmlns:a="a">
			  <p:txStyles><p:otherStyle><a:lvl1pPr algn="ctr"><a:defRPr sz="1800"/></a:lvl1pPr></p:otherStyle></p:txStyles>
			</p:sldMaster>`),
			"ppt/slides/slide1.xml": []byte(`<p:sld xmlns:p="p" xmlns:a="a"/>`),
		},
	}

	styles := inheritedTextStylesWithThemeResolver(pkg, []string{"ppt/slideMasters/slideMaster1.xml", "ppt/slides/slide1.xml"}, "ppt/slides/slide1.xml", func(string) themeColors {
		return defaultThemeColors()
	})
	style := styles["default"].ParagraphStyles[0]
	if style.FontSize != 1800 || style.TextAlign != "ctr" {
		t.Fatalf("master style should override presentation default font size and alignment, got %+v", style)
	}
	if !style.HasMarginRight || style.MarginRight != 914400 || style.FontFamily != "Arial" || !style.HasTextColor || style.TextColor != (color.RGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}) {
		t.Fatalf("presentation default properties should remain as base values, got %+v", style)
	}
}

func TestParseBodyPropertiesReadsTextAnchor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a" anchor="ctr" wrap="square" horzOverflow="clip" vertOverflow="overflow" vert="eaVert" rot="5400000" numCol="2" rtlCol="1" anchorCtr="1" spcFirstLastPara="1"><a:spAutoFit/><a:normAutofit fontScale="85000" lnSpcReduction="20000"/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if element.TextAnchor != "ctr" || !element.HasTextWrap || element.TextWrap != "square" {
		t.Fatalf("unexpected body properties: %+v", element)
	}
	if !element.HasTextHorizontalOverflow || element.TextHorizontalOverflow != "clip" || !element.HasTextVerticalOverflow || element.TextVerticalOverflow != "overflow" {
		t.Fatalf("expected text overflow properties: %+v", element)
	}
	if !element.HasTextVertical || element.TextVertical != "eaVert" || !element.HasTextBodyRotation || element.TextBodyRotation != 5400000 || !element.HasTextColumns || element.TextColumnCount != 2 || !element.HasTextAnchorCenter || !element.TextAnchorCenter {
		t.Fatalf("expected text layout body properties: %+v", element)
	}
	if !element.HasTextRightToLeftColumns || !element.TextRightToLeftColumns {
		t.Fatalf("expected rtlCol body property: %+v", element)
	}
	if !element.HasFirstLastSpacing || !element.IncludeFirstLastSpacing {
		t.Fatalf("expected first/last paragraph spacing flag: %+v", element)
	}
	if !element.HasNormAutofit {
		t.Fatalf("expected normal autofit to be detected: %+v", element)
	}
	if !element.HasShapeAutofit {
		t.Fatalf("expected shape autofit to be detected: %+v", element)
	}
	if !element.HasFontScalePct || element.FontScalePct != 85000 {
		t.Fatalf("unexpected autofit font scale: %+v", element)
	}
	if !element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 20000 {
		t.Fatalf("unexpected autofit line spacing reduction: %+v", element)
	}
}

func TestParseBodyPropertiesReadsText3DMetadata(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a">
	  <a:scene3d><a:camera prst="perspectiveFront"/><a:lightRig rig="soft" dir="br"/></a:scene3d>
	  <a:sp3d z="63500"><a:bevelT/></a:sp3d>
	  <a:flatTx z="12700"/>
	</a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	for _, feature := range []string{
		"text 3-D scene camera perspectiveFront",
		"text 3-D scene light rig soft/br",
		"text 3-D z offset",
		"text 3-D top bevel",
		"text 3-D flat text z offset",
	} {
		if !slices.Contains(element.Text3DFeatures, feature) {
			t.Fatalf("expected text 3-D feature %q, got %+v", feature, element.Text3DFeatures)
		}
	}
}

func TestParseBodyPropertiesReadsNormalAutofitPercentStrings(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a"><a:normAutofit fontScale="92.000%" lnSpcReduction="20%"/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasNormAutofit || !element.HasFontScalePct || element.FontScalePct != 92000 {
		t.Fatalf("expected normal-autofit fontScale percent string to parse, got %+v", element)
	}
	if !element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 20000 {
		t.Fatalf("expected normal-autofit line-spacing reduction percent string to parse, got %+v", element)
	}
}

func TestParseBodyPropertiesReadsExplicitFirstLastSpacingOff(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a" spcFirstLastPara="0"/>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasFirstLastSpacing || element.IncludeFirstLastSpacing {
		t.Fatalf("expected explicit false first/last paragraph spacing flag: %+v", element)
	}
}

func TestParseBodyPropertiesReadsNoAutofitChoice(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a"><a:noAutofit/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasBodyProperties || !element.HasNoAutofit {
		t.Fatalf("expected explicit DrawingML noAutofit state, got %+v", element)
	}
	if element.HasShapeAutofit || element.HasNormAutofit || element.HasFontScalePct || element.FontScalePct != 0 || element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 0 {
		t.Fatalf("noAutofit should not leave active autofit properties, got %+v", element)
	}
}

func TestParseBodyPropertiesNoAutofitSuppressesOtherAutofitChoices(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:bodyPr xmlns:a="a"><a:spAutoFit/><a:normAutofit fontScale="85000" lnSpcReduction="20000"/><a:noAutofit/></a:bodyPr>`))
	if err != nil {
		t.Fatal(err)
	}
	var element slideElement
	parseBodyProperties(root, &element)
	if !element.HasNoAutofit {
		t.Fatalf("expected explicit noAutofit state, got %+v", element)
	}
	if element.HasShapeAutofit || element.HasNormAutofit || element.HasFontScalePct || element.FontScalePct != 0 || element.HasLineSpacingReductionPct || element.LineSpacingReductionPct != 0 {
		t.Fatalf("noAutofit should win over other malformed autofit choices, got %+v", element)
	}
}

func TestFallbackFontPointSizeKeepsThirtyTwoPointText(t *testing.T) {
	got := fallbackFontPointSize(3200, false, false)
	want := 32.0
	if got != want {
		t.Fatalf("expected 32pt text to keep its DrawingML point size: got %v want %v", got, want)
	}
}

func TestParseTextPropertiesKeepsRunSizeOverEndParagraphDefault(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
  <p:txBody>
    <a:p>
      <a:r><a:rPr sz="1400"><a:solidFill><a:srgbClr val="112233"/></a:solidFill></a:rPr><a:t>1</a:t></a:r>
      <a:endParaRPr sz="2000"><a:solidFill><a:srgbClr val="445566"/></a:solidFill></a:endParaRPr>
    </a:p>
  </p:txBody>
</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	got := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if got.FontSize != 0 {
		t.Fatalf("endParaRPr font size should not promote to shape fallback, got %+v", got)
	}
	if got.HasTextColor {
		t.Fatalf("expected direct run text color to stay run-scoped, got %+v", got)
	}
	if len(got.TextParagraphs) != 1 || len(got.TextParagraphs[0].Runs) != 1 {
		t.Fatalf("expected one text run, got %+v", got.TextParagraphs)
	}
	run := got.TextParagraphs[0].Runs[0]
	if run.FontSize != 1400 {
		t.Fatalf("expected run font size to stay run-scoped, got %+v", run)
	}
	if !run.HasTextColor || run.TextColor.R != 0x11 || run.TextColor.G != 0x22 || run.TextColor.B != 0x33 {
		t.Fatalf("expected run text color to win over endParaRPr, got %+v", run)
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphDefaultForParagraphOnly(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr/><a:t>Default sized</a:t></a:r>
    <a:endParaRPr sz="2000" b="1"/>
  </a:p>
  <a:p>
    <a:r><a:rPr/><a:t>Plain</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph parse: %+v", got)
	}
	if got[0].FontSize != 2000 || !got[0].Bold {
		t.Fatalf("expected endParaRPr to become first paragraph default, got %+v", got[0])
	}
	if got[1].FontSize != 0 || got[1].Bold {
		t.Fatalf("endParaRPr leaked to sibling paragraph: %+v", got[1])
	}
}

func TestParseThemeFontsMapsLatinScheme(t *testing.T) {
	fonts := parseThemeFonts([]byte(`<a:theme xmlns:a="a">
  <a:themeElements>
    <a:fontScheme name="Facet">
      <a:majorFont><a:latin typeface="Trebuchet MS"/><a:ea typeface="Yu Gothic"/><a:cs typeface="Times New Roman"/></a:majorFont>
      <a:minorFont><a:latin typeface="Arial"/><a:ea typeface="MS Gothic"/><a:cs typeface="Tahoma"/></a:minorFont>
    </a:fontScheme>
  </a:themeElements>
</a:theme>`))
	if fonts.MajorLatin != "Trebuchet MS" || fonts.MinorLatin != "Arial" {
		t.Fatalf("unexpected theme fonts: %+v", fonts)
	}
	if fonts.MajorEA != "Yu Gothic" || fonts.MajorCS != "Times New Roman" || fonts.MinorEA != "MS Gothic" || fonts.MinorCS != "Tahoma" {
		t.Fatalf("unexpected non-Latin theme fonts: %+v", fonts)
	}
}

func TestApplyThemeFontFamiliesUsesMajorForTitles(t *testing.T) {
	elements := []slideElement{
		{Text: "Title", IsPlaceholder: true, PlaceholderType: "title"},
		{Text: "Body", IsPlaceholder: true, PlaceholderType: "body"},
		{Text: "Fixed", FontFamily: "Existing"},
		{Text: "ElementToken", FontFamily: "+mn-lt"},
		{Text: "Runs", TextParagraphs: []textParagraph{{Runs: []textRun{
			{Text: "Major", FontFamily: "+mj-lt"},
			{Text: "Minor", FontFamily: "+mn-lt"},
			{Text: "MajorEA", FontFamily: "+mj-ea"},
			{Text: "MinorCS", FontFamily: "+mn-cs"},
		}}}},
		{Text: "Bullet", TextParagraphs: []textParagraph{{Bullet: "•", BulletFontFamily: "+mj-cs"}}},
	}
	got := applyThemeFontFamilies(elements, themeFonts{
		MajorLatin: "Trebuchet MS",
		MajorEA:    "Yu Gothic",
		MajorCS:    "Times New Roman",
		MinorLatin: "Arial",
		MinorEA:    "MS Gothic",
		MinorCS:    "Tahoma",
	})
	if got[0].FontFamily != "Trebuchet MS" || got[1].FontFamily != "Arial" || got[2].FontFamily != "Existing" || got[3].FontFamily != "Arial" {
		t.Fatalf("unexpected font family application: %+v", got)
	}
	if got[4].TextParagraphs[0].Runs[0].FontFamily != "Trebuchet MS" || got[4].TextParagraphs[0].Runs[1].FontFamily != "Arial" {
		t.Fatalf("unexpected run font family application: %+v", got[4].TextParagraphs[0].Runs)
	}
	if got[4].TextParagraphs[0].Runs[2].FontFamily != "Yu Gothic" || got[4].TextParagraphs[0].Runs[3].FontFamily != "Tahoma" {
		t.Fatalf("unexpected non-Latin run font family application: %+v", got[4].TextParagraphs[0].Runs)
	}
	if got[5].TextParagraphs[0].BulletFontFamily != "Times New Roman" {
		t.Fatalf("unexpected bullet font family application: %+v", got[5].TextParagraphs[0])
	}
}

func TestShapeAutofitTargetExpandsHeightForText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		Text:            "First\nSecond",
		FontSize:        4800,
		TextParagraphs: []textParagraph{{
			Text:     "First",
			FontSize: 4800,
			Runs:     []textRun{{Text: "First", FontSize: 4800}},
		}, {
			Text:     "Second",
			FontSize: 4800,
			Runs:     []textRun{{Text: "Second", FontSize: 4800}},
		}},
	}, image.Rect(10, 20, 210, 30), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dy() <= 10 {
		t.Fatalf("expected shape target to grow, got %+v", got)
	}
	if got.Min.Y != 20 || got.Min.X != 10 || got.Max.X != 210 {
		t.Fatalf("unexpected horizontal or top adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetShrinksHeightToText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		Text:            "Short",
		FontSize:        1800,
		TextParagraphs: []textParagraph{{
			Text:     "Short",
			FontSize: 1800,
			Runs:     []textRun{{Text: "Short", FontSize: 1800}},
		}},
	}, image.Rect(10, 20, 210, 220), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dy() >= 200 || got.Dy() <= 0 {
		t.Fatalf("expected shape target to shrink to measured text, got %+v", got)
	}
	if got.Min.Y != 20 || got.Min.X != 10 || got.Max.X != 210 {
		t.Fatalf("unexpected horizontal or top adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetExpandsNoWrapWidthForText(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		TextWrap:        "none",
		Text:            "This heading is intentionally wider than the original box",
		FontSize:        2400,
		TextParagraphs: []textParagraph{{
			Text:     "This heading is intentionally wider than the original box",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This heading is intentionally wider than the original box", FontSize: 2400}},
		}},
	}, image.Rect(10, 20, 90, 70), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Dx() <= 80 {
		t.Fatalf("expected no-wrap shape target to grow horizontally, got %+v", got)
	}
	if got.Min.X != 10 || got.Min.Y != 20 {
		t.Fatalf("unexpected top-left adjustment: %+v", got)
	}
}

func TestShapeAutofitTargetDoesNotExpandWrappedWidth(t *testing.T) {
	got, err := shapeAutofitTarget(slideElement{
		HasShapeAutofit: true,
		TextWrap:        "square",
		Text:            "This heading is intentionally wider than the original box",
		FontSize:        2400,
		TextParagraphs: []textParagraph{{
			Text:     "This heading is intentionally wider than the original box",
			FontSize: 2400,
			Runs:     []textRun{{Text: "This heading is intentionally wider than the original box", FontSize: 2400}},
		}},
	}, image.Rect(10, 20, 90, 70), slideSize{CX: 12192000, CY: 6858000}, image.Rect(0, 0, 960, 540))
	if err != nil {
		t.Fatal(err)
	}
	if got.Min.X != 10 || got.Max.X != 90 {
		t.Fatalf("expected wrapped shape target to preserve horizontal bounds, got %+v", got)
	}
}

func TestFitNormalAutofitElementScalesTextToBounds(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:  true,
		PlaceholderType: "title",
		FontScalePct:    90000,
		FontSize:        4000,
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 420, 65))
	if got.FontScalePct == 0 || got.FontScalePct >= 90000 {
		t.Fatalf("expected normal autofit to select a reduced font scale, got %+v", got)
	}
}

func TestFitNormalAutofitElementUsesAuthoredScaleAsProbeStart(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:  true,
		HasFontScalePct: true,
		FontScalePct:    90000,
		FontFamily:      "Carlito",
		FontSize:        4000,
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 420, 65))
	if got.FontScalePct == 0 || got.FontScalePct > 90000 {
		t.Fatalf("authored normal-autofit fontScale should cap the probe start, got %+v", got)
	}
}

func TestFitNormalAutofitElementCanScaleBelowFiftyPercent(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit: true,
		FontFamily:     "Carlito",
		FontSize:       4000,
		TextWrap:       "none",
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}, image.Rect(0, 0, 180, 30))
	if got.FontScalePct >= 50000 || got.FontScalePct < minimumNormalAutofitFontScalePct {
		t.Fatalf("expected normal autofit to use the supported scale range below 50%%, got %+v", got)
	}
	if !textFitsAtScale(got, image.Rect(0, 0, 180, 30), got.FontScalePct, normalAutofitMaxSoftLines(got), defaultOutputDPI) {
		t.Fatalf("selected scale should fit in the target bounds, got %+v", got)
	}
}

func TestFitNormalAutofitElementSelectsLargestFittingScale(t *testing.T) {
	element := slideElement{
		HasNormAutofit: true,
		FontFamily:     "Carlito",
		FontSize:       4000,
		TextWrap:       "none",
		TextParagraphs: []textParagraph{{
			Text:     "Wide Heading With Several Words",
			FontSize: 4000,
			Runs: []textRun{{
				Text:     "Wide Heading With Several Words",
				FontSize: 4000,
			}},
		}},
	}
	bounds := image.Rect(0, 0, 180, 30)
	got := fitNormalAutofitElement(element, bounds)

	if got.FontScalePct <= minimumNormalAutofitFontScalePct || got.FontScalePct >= 100000 {
		t.Fatalf("expected a derived normal-autofit scale within supported bounds, got %+v", got)
	}
	if !textFitsAtScale(got, bounds, got.FontScalePct, normalAutofitMaxSoftLines(got), defaultOutputDPI) {
		t.Fatalf("selected normal-autofit scale should fit, got %+v", got)
	}
	if textFitsAtScale(got, bounds, got.FontScalePct+1, normalAutofitMaxSoftLines(got), defaultOutputDPI) {
		t.Fatalf("selected normal-autofit scale should be the largest fitting scale, got %+v", got)
	}
}

func TestFitNormalAutofitElementPreservesAuthoredLineSpacingReduction(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit:             true,
		HasLineSpacingReductionPct: true,
		LineSpacingReductionPct:    10000,
		FontFamily:                 "Carlito",
		FontSize:                   2400,
		TextWrap:                   "square",
		TextParagraphs: []textParagraph{{
			Text:           "Short body",
			FontSize:       2400,
			LineSpacingPct: 90000,
			Runs: []textRun{{
				Text:     "Short body",
				FontSize: 2400,
			}},
		}},
	}, image.Rect(0, 0, 500, 100))
	if !got.HasLineSpacingReductionPct || got.LineSpacingReductionPct != 10000 {
		t.Fatalf("authored line spacing reduction should be preserved: %+v", got)
	}
}

func TestNormalAutofitMaxSoftLinesHonorsWrapNoneAndHardBreaks(t *testing.T) {
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "square",
		Text:     "Single line title",
		TextParagraphs: []textParagraph{{
			Text: "Single line title",
			Runs: []textRun{{Text: "Single line title"}},
		}},
	}); got != 0 {
		t.Fatalf("wrapping text without hard breaks should not cap soft lines, got %d", got)
	}
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "none",
		Text:     "Single line title",
		TextParagraphs: []textParagraph{{
			Text: "Single line title",
			Runs: []textRun{{Text: "Single line title"}},
		}},
	}); got != 1 {
		t.Fatalf(`expected wrap="none" text without hard breaks to require single-line fit, got %d`, got)
	}
	if got := normalAutofitMaxSoftLines(slideElement{
		TextWrap: "square",
		Text:     "Line one\nLine two",
		TextParagraphs: []textParagraph{{
			Text: "Line one\nLine two",
			Runs: []textRun{{Text: "Line one\nLine two"}},
		}},
	}); got != 2 {
		t.Fatalf("hard breaks should cap normal-autofit soft lines to authored line count, got %d", got)
	}
}

func TestTextRenderLinesPreserveDrawingMLBreakRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:bodyPr><a:normAutofit/></a:bodyPr><a:lstStyle/><a:p><a:r><a:rPr sz="4400"/><a:t> Welcome to </a:t></a:r><a:br><a:rPr sz="4400"/></a:br><a:r><a:rPr sz="4400"/><a:t>GENERATE: The Game of Energy Choices</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := textParagraphsFromNode(root)
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(4400, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(4400, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{TextParagraphs: paragraphs, FontSize: 4400}, 900)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected DrawingML break to create two render lines, got %d: %+v", len(lines), lines)
	}
	if !strings.Contains(lines[1].Text, "GENERATE") {
		t.Fatalf("expected second line to preserve following run text, got %+v", lines[1])
	}
}

func TestTextRenderLinesPreserveDrawingMLBreakRunMetrics(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:bodyPr/><a:lstStyle/><a:p><a:r><a:rPr sz="1600"/><a:t>Small</a:t></a:r><a:br><a:rPr sz="4800" b="1"/></a:br><a:r><a:rPr sz="1600"/><a:t>Next</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := textParagraphsFromNode(root)
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(1600, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(1600, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{TextParagraphs: paragraphs, FontSize: 1600}, 900)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected DrawingML break to create two render lines, got %d: %+v", len(lines), lines)
	}
	if len(lines[0].Segments) != 2 || lines[0].Segments[1].Text != "" || lines[0].Segments[1].FontSize != 4800 || !lines[0].Segments[1].Bold {
		t.Fatalf("expected break run properties to stay on the preceding line as metric segment, got %+v", lines[0])
	}
	measured, err := measureTextRenderLines(faces, lines, 1600)
	if err != nil {
		t.Fatal(err)
	}
	if measured[0].Height <= measured[1].Height {
		t.Fatalf("expected break run metrics to affect first line height, got %+v", measured)
	}
}

func TestTextRenderLinesPreserveAuthoredEmptyParagraphs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="2200"/><a:t>First</a:t></a:r></a:p>
  <a:p><a:pPr><a:spcBef><a:spcPts val="0"/></a:spcBef><a:spcAft><a:spcPts val="0"/></a:spcAft></a:pPr><a:endParaRPr sz="2200"/></a:p>
  <a:p><a:r><a:rPr sz="2200"/><a:t>Second</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 3 || paragraphs[1].Text != "" || len(paragraphs[1].Runs) != 0 || paragraphs[1].FontSize != 2200 {
		t.Fatalf("expected empty authored paragraph with endParaRPr metrics, got %+v", paragraphs)
	}
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(2200, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(2200, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{TextParagraphs: paragraphs, FontSize: 2200}, 900)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected empty authored paragraph to produce a blank render line, got %d: %+v", len(lines), lines)
	}
	if lines[1].Text != "" || lines[1].FontSize != 2200 {
		t.Fatalf("expected blank line to preserve paragraph metrics, got %+v", lines[1])
	}
	measured, err := measureTextRenderLines(faces, lines, 2200)
	if err != nil {
		t.Fatal(err)
	}
	if len(measured) != 3 || measured[1].HasText || measured[1].Height <= 0 {
		t.Fatalf("expected blank paragraph to reserve vertical advance, got %+v", measured)
	}
}

func TestTextRenderLinesPreserveExplicitEmptyBulletParagraphs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr marL="285750" indent="-285750">
      <a:buFont typeface="Arial"/>
      <a:buChar char="•"/>
      <a:defRPr/>
    </a:pPr>
    <a:endParaRPr sz="2200"/>
  </a:p>
  <a:p>
    <a:pPr><a:defRPr/></a:pPr>
    <a:endParaRPr sz="2200"/>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	paragraphs := textParagraphsFromNode(root)
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(2200, false)
	if err != nil {
		t.Fatal(err)
	}
	boldFace, err := faces.Get(2200, true)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := textRenderLinesForElement(faces, face, boldFace, slideElement{TextParagraphs: paragraphs, FontSize: 2200}, 900)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected empty bullet and blank paragraphs to render as two lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "• " {
		t.Fatalf("expected explicit empty bullet paragraph to render bullet prefix, got %+v", lines[0])
	}
	if lines[1].Text != "" {
		t.Fatalf("expected empty paragraph without local bullet choice to stay blank, got %+v", lines[1])
	}
}

func TestFitNormalAutofitAllowsWrappingWithinHardBreakLines(t *testing.T) {
	got := fitNormalAutofitElement(slideElement{
		HasNormAutofit: true,
		FontScalePct:   90000,
		FontFamily:     "Carlito",
		FontSize:       4000,
		Text:           "Residual Risk and Technology Review of\nSurface Coating NESHAP",
		TextParagraphs: []textParagraph{{
			Text:     "Residual Risk and Technology Review of\nSurface Coating NESHAP",
			FontSize: 4000,
			Runs: []textRun{
				{Text: "Residual Risk and Technology Review of ", FontSize: 4000},
				{Text: "\n", FontSize: 4000},
				{Text: "Surface Coating NESHAP", FontSize: 4000},
			},
		}},
	}, image.Rect(0, 0, 520, 150))
	if got.FontScalePct == 90000 {
		t.Fatalf("hard-break segments that soft-wrap should trigger normal autofit scaling, got %+v", got)
	}
}

func TestFitNormalAutofitDoesNotMutateParagraphFontSizes(t *testing.T) {
	element := slideElement{
		HasNormAutofit: true,
		FontScalePct:   90000,
		FontFamily:     "Carlito",
		FontSize:       4000,
		TextParagraphs: []textParagraph{{
			TextAlign: "ctr",
			FontSize:  4000,
			Runs: []textRun{
				{Text: "Residual Risk and Technology Review of ", FontSize: 4000},
				{Text: "\n", FontSize: 4000},
				{Text: "Surface Coating NESHAP ", FontSize: 4000},
				{Text: "\n"},
			},
		}},
	}

	got := fitNormalAutofitElement(element, image.Rect(0, 0, 700, 200))
	if element.TextParagraphs[0].FontSize != 4000 || element.TextParagraphs[0].Runs[0].FontSize != 4000 {
		t.Fatalf("normal-autofit probing mutated source text sizes: %+v", element.TextParagraphs[0])
	}

	scaled := scaledTextElement(got)
	if scaled.FontSize != 3600 || scaled.TextParagraphs[0].FontSize != 3600 || scaled.TextParagraphs[0].Runs[0].FontSize != 3600 {
		t.Fatalf("expected explicit 90%% normal-autofit to scale text once, got element=%+v paragraph=%+v", scaled, scaled.TextParagraphs[0])
	}
}

func TestScaleParagraphSpacingForDPIDoesNotMutateSourceParagraphs(t *testing.T) {
	element := slideElement{TextParagraphs: []textParagraph{{
		SpaceBefore: 9,
		SpaceAfter:  18,
		TabStops:    []int64{914400},
		Runs:        []textRun{{Text: "Title", FontSize: 2400}},
	}}}

	got := scaleParagraphSpacingForDPI(element, 96)
	if element.TextParagraphs[0].SpaceBefore != 9 || element.TextParagraphs[0].SpaceAfter != 18 || element.TextParagraphs[0].Runs[0].FontSize != 2400 {
		t.Fatalf("DPI spacing scaling mutated source paragraphs: %+v", element.TextParagraphs[0])
	}
	got.TextParagraphs[0].Runs[0].FontSize = 1200
	got.TextParagraphs[0].TabStops[0] = 1
	if element.TextParagraphs[0].Runs[0].FontSize != 2400 || element.TextParagraphs[0].TabStops[0] != 914400 {
		t.Fatalf("DPI spacing scaling reused nested paragraph slices: %+v", element.TextParagraphs[0])
	}
}

func TestMeasuredTextHeightIncludesInkExtentsWhenLineSpacingIsTight(t *testing.T) {
	got := measuredTextHeight([]measuredTextLine{
		{Ascent: 39, Descent: 10, Height: 36},
		{Ascent: 39, Descent: 10, Height: 36},
		{Ascent: 39, Descent: 10, Height: 36},
	})
	if got != 121 {
		t.Fatalf("expected ink extents to exceed tight line advances, got %d", got)
	}
}

func TestMeasuredTextAnchorHeightUsesVisibleInkBoxForCenteredText(t *testing.T) {
	lines := []measuredTextLine{{
		Ascent:      30,
		Descent:     8,
		Height:      48,
		HasText:     true,
		SpaceBefore: 2,
		SpaceAfter:  3,
	}}

	if got := measuredTextAnchorHeight(lines, "ctr"); got != 43 {
		t.Fatalf("centered text anchor should use visible ink height, got %d", got)
	}
	if got := measuredTextAnchorHeight(lines, "b"); got != 43 {
		t.Fatalf("bottom text anchor should use visible ink height, got %d", got)
	}
	if got := measuredTextAnchorHeight(lines, ""); got != 53 {
		t.Fatalf("top anchored text should keep full line advance, got %d", got)
	}
}

func TestMeasuredTextAnchorHeightKeepsEmptyParagraphAdvance(t *testing.T) {
	lines := []measuredTextLine{{
		Ascent:      30,
		Descent:     8,
		Height:      48,
		SpaceBefore: 2,
		SpaceAfter:  3,
	}}

	if got := measuredTextAnchorHeight(lines, "ctr"); got != 53 {
		t.Fatalf("centered empty paragraph should use authored line advance, got %d", got)
	}
}

func TestDrawShapeTextDoesNotDropBottomAnchoredLineByBaseline(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 320, 44))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		FontFamily: "Carlito",
		FontSize:   2400,
		TextAnchor: "b",
		TextParagraphs: []textParagraph{{
			Runs: []textRun{
				{Text: "First line", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "\n", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "Second line", FontSize: 2400, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
			},
		}},
	}
	if err := drawShapeTextWithDPI(img, img.Bounds(), element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	if got := countNonWhitePixelsBelow(img, 26); got == 0 {
		t.Fatal("expected bottom-anchored second line to render when its line box intersects the bounds")
	}
}

func TestDrawShapeTextClipsGlyphsToTextBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 220, 70))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	element := slideElement{
		FontFamily:              "Carlito",
		FontSize:                3200,
		TextAnchor:              "b",
		TextVerticalOverflow:    "clip",
		HasTextVerticalOverflow: true,
		TextParagraphs: []textParagraph{{
			Runs: []textRun{
				{Text: "First", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "\n", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
				{Text: "Second", FontSize: 3200, HasTextColor: true, TextColor: color.RGBA{B: 255, A: 255}},
			},
		}},
	}
	bounds := image.Rect(0, 0, 220, 44)
	if err := drawShapeTextWithDPI(img, bounds, element, defaultOutputDPI); err != nil {
		t.Fatal(err)
	}
	if got := countNonWhitePixelsBelow(img, 44); got != 0 {
		t.Fatalf("expected text drawing to be clipped at the text bounds, got %d painted pixel(s) below", got)
	}
	if got := countNonWhitePixelsBelow(img, 26); got == 0 {
		t.Fatal("expected the bottom line to remain visible inside the clipped text bounds")
	}
}

func countNonWhitePixelsBelow(img *image.RGBA, minY int) int {
	count := 0
	bounds := img.Bounds()
	for y := maxInt(bounds.Min.Y, minY); y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.RGBAAt(x, y) != (color.RGBA{R: 255, G: 255, B: 255, A: 255}) {
				count++
			}
		}
	}
	return count
}

func TestShouldFitNormalAutofitUsesImplicitScaleWhenRequested(t *testing.T) {
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "title"}) {
		t.Fatal("regular title normal-autofit should derive a scale when requested")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "title", FontScalePct: 90000}) {
		t.Fatal("expected title normal-autofit with explicit fontScale to fit")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "ctrTitle"}) {
		t.Fatal("centered title normal-autofit should derive a scale when requested")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, PlaceholderType: "ctrTitle", FontScalePct: 90000}) {
		t.Fatal("expected centered title normal-autofit with explicit fontScale to fit")
	}
	if !shouldFitNormalAutofit(slideElement{HasNormAutofit: true, LineSpacingReductionPct: 10000}) {
		t.Fatal("content normal-autofit should derive a scale when requested")
	}
	if shouldFitNormalAutofit(slideElement{}) {
		t.Fatal("normal-autofit should not run when the text body did not request it")
	}
}

func TestScaledTextElementAppliesNormalAutofitFontScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		FontScalePct:            85000,
		FontSize:                2000,
		LineSpacingReductionPct: 20000,
		TextParagraphs: []textParagraph{{
			FontSize:       1000,
			SpaceBefore:    10,
			SpaceAfter:     20,
			LineSpacingPct: 90000,
			Runs: []textRun{
				{Text: "A", FontSize: 1200},
			},
		}},
	})
	if got.FontSize != 1700 || got.TextParagraphs[0].FontSize != 850 || got.TextParagraphs[0].Runs[0].FontSize != 1020 {
		t.Fatalf("unexpected scaled text sizes: %+v", got)
	}
	if got.TextParagraphs[0].LineSpacingPct != 70000 {
		t.Fatalf("unexpected reduced line spacing: %+v", got)
	}
	if got.TextParagraphs[0].SpaceBefore != 9 || got.TextParagraphs[0].SpaceAfter != 17 {
		t.Fatalf("unexpected scaled paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementAppliesNormalAutofitLineSpacingReductionWithoutFontScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		LineSpacingReductionPct: 10000,
		FontSize:                2000,
		TextParagraphs: []textParagraph{{
			FontSize:       2000,
			SpaceBefore:    8,
			SpaceAfter:     12,
			LineSpacingPct: 90000,
			Runs: []textRun{{
				Text:     "A",
				FontSize: 2000,
			}},
		}},
	})
	if got.FontSize != 2000 || got.TextParagraphs[0].FontSize != 2000 || got.TextParagraphs[0].Runs[0].FontSize != 2000 {
		t.Fatalf("line spacing reduction must not scale font sizes: %+v", got)
	}
	if got.TextParagraphs[0].LineSpacingPct != 80000 {
		t.Fatalf("line spacing reduction should apply independently of font scaling: %+v", got)
	}
	if got.TextParagraphs[0].SpaceBefore != 8 || got.TextParagraphs[0].SpaceAfter != 12 {
		t.Fatalf("line spacing reduction must not scale paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementScalesParagraphSpacingForDPI(t *testing.T) {
	got := scaledTextElement(slideElement{
		TextParagraphs: []textParagraph{{
			SpaceBefore: 9,
			SpaceAfter:  12,
		}},
	}, 96)
	if got.TextParagraphs[0].SpaceBefore != 12 || got.TextParagraphs[0].SpaceAfter != 16 {
		t.Fatalf("unexpected dpi-scaled paragraph spacing: %+v", got)
	}
}

func TestScaledTextElementAppliesLineSpacingReductionAtFullScale(t *testing.T) {
	got := scaledTextElement(slideElement{
		FontScalePct:            100000,
		LineSpacingReductionPct: 10000,
		TextParagraphs: []textParagraph{{
			Text:           "Body",
			FontSize:       2400,
			LineSpacingPct: 100000,
		}},
	})
	if got.TextParagraphs[0].LineSpacingPct != 90000 {
		t.Fatalf("line spacing reduction should apply when fontScale is 100%%: %+v", got)
	}
}

func TestScaledTextElementDoesNotInventPercentageLineSpacingForReduction(t *testing.T) {
	got := scaledTextElement(slideElement{
		LineSpacingReductionPct: 10000,
		TextParagraphs: []textParagraph{{
			Text:     "Body",
			FontSize: 2400,
		}},
	})
	if got.TextParagraphs[0].LineSpacingPct != 0 {
		t.Fatalf("line spacing reduction applies only to percentage line spacing: %+v", got)
	}
}

func TestAnchoredTextStartYCentersLines(t *testing.T) {
	got := anchoredTextStartY(image.Rect(0, 10, 100, 110), 2, 20, 12, "ctr")
	want := 52
	if got != want {
		t.Fatalf("unexpected centered text y: got=%d want=%d", got, want)
	}
}

func TestTextFromNodePreservesParagraphBreaks(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>First</a:t></a:r></a:p><a:p><a:r><a:t>Second</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(textFromNode(root)); got != "First\nSecond" {
		t.Fatalf("unexpected paragraph text: %q", got)
	}
}

func TestTextFromNodePreservesTabs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a"><a:p><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p></p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(textFromNode(root)); got != "Cost\tTotal" {
		t.Fatalf("unexpected tabbed text: %q", got)
	}
	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || len(paragraphs[0].Runs) != 1 || paragraphs[0].Runs[0].Text != "Cost\tTotal" {
		t.Fatalf("expected tab in text run, got %+v", paragraphs)
	}
}

func TestResolveTextFieldsUpdatesSlideNumberFieldRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:sp xmlns:p="p" xmlns:a="a">
	  <p:txBody><a:bodyPr/><a:lstStyle/><a:p>
	    <a:fld id="{424CEEAC-8F67-4238-9622-1B74DC6E8318}" type="slidenum"><a:rPr sz="1200"/><a:t>‹#›</a:t></a:fld>
	  </a:p></p:txBody>
	</p:sp>`))
	if err != nil {
		t.Fatal(err)
	}
	element := parseSlideElementNode(root, renderTransform{ScaleX: 1, ScaleY: 1})
	if len(element.TextParagraphs) != 1 || len(element.TextParagraphs[0].Runs) != 1 || element.TextParagraphs[0].Runs[0].FieldType != "slidenum" {
		t.Fatalf("expected slide-number field metadata to be preserved, got %+v", element.TextParagraphs)
	}

	got := resolveTextFields([]slideElement{element}, 12)
	if got[0].Text != "12" || got[0].TextParagraphs[0].Text != "12" || got[0].TextParagraphs[0].Runs[0].Text != "12" {
		t.Fatalf("expected slide-number field to resolve from render options, got %+v", got[0])
	}
}

func TestResolveTextFieldsLeavesCachedDateFieldsStable(t *testing.T) {
	paragraphs := []textParagraph{{
		Text: "8/18/2021",
		Runs: []textRun{{
			Text:      "8/18/2021",
			FieldType: "datetimeFigureOut",
		}},
	}}

	got := resolveTextFields([]slideElement{{Text: "8/18/2021", TextParagraphs: paragraphs}}, 7)
	if got[0].Text != "8/18/2021" || got[0].TextParagraphs[0].Runs[0].Text != "8/18/2021" {
		t.Fatalf("non-slide-number fields should keep cached package text, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeParsesTabStops(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:tabLst><a:tab pos="1074738" algn="l"/></a:tabLst></a:pPr><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || len(paragraphs[0].TabStops) != 1 || paragraphs[0].TabStops[0] != 1074738 {
		t.Fatalf("expected explicit paragraph tab stop, got %+v", paragraphs)
	}
	if stops := tabStopsAtDPI(paragraphs[0].TabStops, 96); len(stops) != 1 || stops[0] != 113 {
		t.Fatalf("expected tab stop to scale to 96 DPI pixels, got %+v", stops)
	}
}

func TestTextParagraphsFromNodeParsesDefaultTabSize(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:lstStyle><a:lvl1pPr defTabSz="457200"/></a:lstStyle>
	  <a:p><a:pPr lvl="0"/><a:r><a:t>Cost</a:t><a:tab/><a:t>Total</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || !paragraphs[0].HasDefaultTab || paragraphs[0].DefaultTabSize != 457200 {
		t.Fatalf("expected default tab size inherited from list style, got %+v", paragraphs)
	}
	stops := paragraphTabStopsAtDPI(paragraphs[0], 72, 160)
	if len(stops) < 3 || stops[0] != 36 || stops[1] != 72 || stops[2] != 108 {
		t.Fatalf("expected repeating half-inch default tab stops, got %+v", stops)
	}
}

func TestTextParagraphsFromNodeParsesRightMargin(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:pPr marR="914400"/><a:r><a:t>Right margin</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	paragraphs := textParagraphsFromNode(root)
	if len(paragraphs) != 1 || !paragraphs[0].HasMarginRight || paragraphs[0].MarginRight != 914400 {
		t.Fatalf("expected paragraph right margin, got %+v", paragraphs)
	}
	if got := paragraphRightOffsetAtDPI(paragraphs[0], 96); got != 96 {
		t.Fatalf("expected right margin to scale to 96 DPI pixels, got %d", got)
	}
}

func TestTextParagraphsFromNodeDetectsBulletsAndLevels(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr b="1"/><a:t>Primary energy resources</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"/><a:r><a:t>Fossil</a:t></a:r></a:p>
  <a:p><a:pPr lvl="2"><a:buSzPts val="1400"/><a:buChar char="-"/></a:pPr><a:r><a:t>Coal</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"><a:buFont typeface="Wingdings"/><a:buChar char="§"/></a:pPr><a:r><a:t>Wingdings square</a:t></a:r></a:p>
  <a:p><a:pPr><a:buNone/></a:pPr><a:r><a:t>No bullet</a:t></a:r></a:p>
  <a:p><a:pPr><a:buClrTx/><a:buFontTx/><a:buChar char="•"/></a:pPr><a:r><a:t>Follow text</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 6 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Text != "Primary energy resources" || !got[0].Bold || len(got[0].Runs) != 1 || !got[0].Runs[0].Bold {
		t.Fatalf("unexpected first paragraph: %+v", got[0])
	}
	if got[1].Text != "Fossil" || got[1].Bullet != "•" || got[1].Level != 1 {
		t.Fatalf("unexpected default bullet paragraph: %+v", got[1])
	}
	if got[2].Text != "Coal" || got[2].Bullet != "-" || got[2].Level != 2 {
		t.Fatalf("unexpected explicit bullet paragraph: %+v", got[2])
	}
	if got[2].BulletFontSize != 1400 {
		t.Fatalf("expected explicit bullet font size, got %+v", got[2])
	}
	if got[3].Text != "Wingdings square" || got[3].Bullet != "▪" || got[3].Level != 1 {
		t.Fatalf("unexpected Wingdings bullet paragraph: %+v", got[3])
	}
	if got[3].BulletFontFamily != "Wingdings" {
		t.Fatalf("expected Wingdings bullet font family to be preserved, got %+v", got[3])
	}
	if got[4].Text != "No bullet" || !got[4].NoBullet {
		t.Fatalf("unexpected no-bullet paragraph: %+v", got[4])
	}
	if !got[5].BulletColorTx || !got[5].BulletFontTx {
		t.Fatalf("expected bullet color/font to follow text, got %+v", got[5])
	}
}

func TestTextParagraphsFromNodeMapsWingdingsNotSignBullet(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buFont typeface="Wingdings"/><a:buChar char="Ø"/></a:pPr><a:r><a:t>Mapped</a:t></a:r></a:p>
  <a:p><a:pPr><a:buFont typeface="Arial"/><a:buChar char="Ø"/></a:pPr><a:r><a:t>Literal</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bullet != "¬" || got[0].BulletFontFamily != "Wingdings" {
		t.Fatalf("expected Wingdings Ø bullet to map to Unicode not sign, got %+v", got[0])
	}
	if got[1].Bullet != "Ø" || got[1].BulletFontFamily != "Arial" {
		t.Fatalf("non-Wingdings Ø bullet should stay literal, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeNumbersAutoBullets(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>First</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>Second</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod" startAt="4"/></a:pPr><a:r><a:t>Restarted</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"><a:buAutoNum type="arabicParenR" startAt="2"/></a:pPr><a:r><a:t>Nested</a:t></a:r></a:p>
  <a:p><a:pPr><a:buAutoNum type="alphaLcPeriod"/></a:pPr><a:r><a:t>Continued</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 5 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bullet != "a." || got[1].Bullet != "b." || got[2].Bullet != "d." || got[3].Bullet != "2)" || got[4].Bullet != "e." {
		t.Fatalf("unexpected auto-number bullets: %+v", got)
	}
}

func TestTextParagraphsFromNodeInheritsStyledAutoNumberBullets(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:lstStyle>
    <a:lvl1pPr>
      <a:buAutoNum type="alphaLcPeriod" startAt="3"/>
      <a:defRPr sz="1800"/>
    </a:lvl1pPr>
    <a:lvl2pPr>
      <a:buAutoNum type="arabicParenR" startAt="2"/>
      <a:defRPr sz="1600"/>
    </a:lvl2pPr>
  </a:lstStyle>
  <a:p><a:pPr/><a:r><a:t>Third</a:t></a:r></a:p>
  <a:p><a:pPr/><a:r><a:t>Fourth</a:t></a:r></a:p>
  <a:p><a:pPr lvl="1"/><a:r><a:t>Nested</a:t></a:r></a:p>
  <a:p><a:pPr/><a:r><a:t>Fifth</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 4 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bullet != "c." || got[1].Bullet != "d." || got[2].Bullet != "2)" || got[3].Bullet != "e." {
		t.Fatalf("unexpected inherited auto-number bullets: %+v", got)
	}
}

func TestAutoNumberBulletFormatsCommonDrawingMLSchemes(t *testing.T) {
	tests := []struct {
		name  string
		kind  string
		index int
		want  string
	}{
		{name: "lower alpha both parentheses", kind: "alphaLcParenBoth", index: 27, want: "(aa)"},
		{name: "upper alpha right parenthesis", kind: "alphaUcParenR", index: 28, want: "AB)"},
		{name: "upper alpha period", kind: "alphaUcPeriod", index: 2, want: "B."},
		{name: "arabic both parentheses", kind: "arabicParenBoth", index: 12, want: "(12)"},
		{name: "arabic right parenthesis", kind: "arabicParenR", index: 3, want: "3)"},
		{name: "arabic period", kind: "arabicPeriod", index: 4, want: "4."},
		{name: "arabic plain", kind: "arabicPlain", index: 5, want: "5"},
		{name: "lower roman both parentheses", kind: "romanLcParenBoth", index: 9, want: "(ix)"},
		{name: "upper roman right parenthesis", kind: "romanUcParenR", index: 14, want: "XIV)"},
		{name: "lower roman period", kind: "romanLcPeriod", index: 44, want: "xliv."},
		{name: "upper roman period", kind: "romanUcPeriod", index: 3999, want: "MMMCMXCIX."},
		{name: "minimum index", kind: "arabicPlain", index: 0, want: "1"},
		{name: "unhandled schema family falls back to arabic period", kind: "thaiNumPeriod", index: 6, want: "6."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := autoNumberBullet(test.kind, test.index); got != test.want {
				t.Fatalf("autoNumberBullet(%q, %d) = %q, want %q", test.kind, test.index, got, test.want)
			}
		})
	}
}

func TestTextParagraphsFromNodeLocalBulletChoiceBlocksStyledAutoNumber(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:lstStyle>
    <a:lvl1pPr>
      <a:buAutoNum type="arabicParenR"/>
      <a:defRPr sz="1800"/>
    </a:lvl1pPr>
  </a:lstStyle>
  <a:p><a:pPr><a:buChar char="•"/></a:pPr><a:r><a:t>Symbol</a:t></a:r></a:p>
  <a:p><a:pPr><a:buNone/></a:pPr><a:r><a:t>No bullet</a:t></a:r></a:p>
  <a:p><a:pPr/><a:r><a:t>Numbered</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 3 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bullet != "•" || got[0].NoBullet {
		t.Fatalf("local buChar should block styled auto-numbering, got %+v", got[0])
	}
	if got[1].Bullet != "" || !got[1].NoBullet {
		t.Fatalf("local buNone should block styled auto-numbering, got %+v", got[1])
	}
	if got[2].Bullet != "1)" {
		t.Fatalf("styled auto-numbering should still apply to later paragraphs, got %+v", got[2])
	}
}

func TestTextParagraphsFromNodeCapturesBulletSizeFollowText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buSzTx/><a:buChar char="•"/></a:pPr><a:r><a:rPr sz="2400"/><a:t>Follow text</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || !got[0].BulletSizeTx || got[0].BulletFontSize != 0 || got[0].BulletSizePct != 0 {
		t.Fatalf("expected paragraph buSzTx bullet size, got %+v", got)
	}
	if size := bulletSegmentFontSize(got[0]); size != 2400 {
		t.Fatalf("buSzTx bullet should follow text size, got %d", size)
	}
}

func TestTextParagraphsPreservesSingleLeadingRunSpace(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:t> Welcome</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 || got[0].Runs[0].Text != " Welcome" {
		t.Fatalf("expected single leading space to be preserved, got %+v", got)
	}
}

func TestTextParagraphsPreservesManualLeadingPadding(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:t>          Centered title</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 || got[0].Runs[0].Text != "          Centered title" {
		t.Fatalf("expected manual leading padding to be preserved, got %+v", got)
	}
}

func TestTextParagraphsFromNodeUsesNoBulletSizeAsFallbackFontSize(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:buSzPts val="4400"/><a:buNone/><a:spcAft><a:spcPts val="1200"/></a:spcAft></a:pPr><a:r><a:rPr/><a:t>Title</a:t></a:r></a:p>
  <a:p><a:pPr><a:buSzPts val="2200"/><a:buNone/></a:pPr><a:r><a:rPr sz="1800"/><a:t>Explicit</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 4400 {
		t.Fatalf("expected no-bullet paragraph size fallback, got %+v", got[0])
	}
	if got[0].SpaceAfter != 12 {
		t.Fatalf("expected paragraph after-spacing, got %+v", got[0])
	}
	if got[1].FontSize != 1800 {
		t.Fatalf("explicit run size should win over no-bullet fallback, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeParsesPercentParagraphSpacing(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:spcBef><a:spcPct val="90000"/></a:spcBef><a:spcAft><a:spcPct val="110000"/></a:spcAft></a:pPr><a:r><a:rPr sz="1800"/><a:t>Percent spacing</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || !got[0].HasSpaceBefore || got[0].SpaceBeforePct != 90000 || got[0].SpaceAfterPct != 110000 {
		t.Fatalf("expected percent paragraph spacing, got %+v", got)
	}
	if got[0].SpaceBefore != 0 || got[0].SpaceAfter != 0 {
		t.Fatalf("percent spacing should not be stored as fixed pixels, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeParsesRunCharacterSpacing(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:defRPr spc="150"/></a:pPr><a:r><a:rPr spc="250"/><a:t>Wide</a:t></a:r><a:r><a:rPr/><a:t>Default</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || !got[0].HasCharSpacing || got[0].CharSpacing != 150 {
		t.Fatalf("expected paragraph default character spacing, got %+v", got)
	}
	if len(got[0].Runs) != 2 || !got[0].Runs[0].HasCharSpacing || got[0].Runs[0].CharSpacing != 250 {
		t.Fatalf("expected run character spacing, got %+v", got[0].Runs)
	}
	if segment := runToSegment(got[0].Runs[1], got[0]); segment.CharSpacing != 150 {
		t.Fatalf("expected unstyled run to inherit paragraph character spacing, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeParsesRunCaps(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr><a:defRPr cap="all"/></a:pPr><a:r><a:rPr/><a:t>Default caps</a:t></a:r><a:r><a:rPr cap="none"/><a:t> plain</a:t></a:r><a:r><a:rPr cap="small" sz="2000"/><a:t>aB c</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || !got[0].HasTextCaps || got[0].TextCaps != "all" {
		t.Fatalf("expected paragraph default caps, got %+v", got)
	}
	if len(got[0].Runs) != 3 || got[0].Runs[0].HasTextCaps || !got[0].Runs[1].HasTextCaps || got[0].Runs[1].TextCaps != "none" || got[0].Runs[2].TextCaps != "small" {
		t.Fatalf("expected run caps to preserve explicit values, got %+v", got[0].Runs)
	}
	if segment := runToSegment(got[0].Runs[0], got[0]); segment.TextCaps != "all" {
		t.Fatalf("expected unstyled run to inherit all-caps paragraph default, got %+v", segment)
	}
	if segment := runToSegment(got[0].Runs[1], got[0]); segment.TextCaps != "" {
		t.Fatalf("expected cap=none to clear paragraph default, got %+v", segment)
	}
	if segments := runToSegments(got[0].Runs[2], got[0]); len(segments) != 3 || segments[0].Text != "A" || segments[1].Text != "B " || segments[2].Text != "C" || segments[0].FontSize >= segments[1].FontSize || segments[2].FontSize >= segments[1].FontSize {
		t.Fatalf("expected small caps to split lowercase text into smaller uppercase segments, got %+v", segments)
	}
}

func TestTextParagraphsFromNodePreservesEmptyParagraphs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="2400"/><a:t>Before</a:t></a:r></a:p>
  <a:p><a:endParaRPr sz="2400" b="1"/></a:p>
  <a:p><a:r><a:rPr sz="2400"/><a:t>After</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 3 {
		t.Fatalf("expected empty paragraph to be preserved, got %+v", got)
	}
	if got[1].Text != "" || got[1].FontSize != 2400 || !got[1].Bold {
		t.Fatalf("expected empty paragraph end properties, got %+v", got[1])
	}
	if !got[1].NoBullet {
		t.Fatalf("expected empty paragraph to reserve space without a bullet, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodePreservesExplicitEmptyBulletParagraphs(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr marL="285750" indent="-285750">
      <a:buFont typeface="Arial"/>
      <a:buChar char="•"/>
      <a:defRPr/>
    </a:pPr>
    <a:endParaRPr sz="2200"/>
  </a:p>
  <a:p>
    <a:pPr><a:buNone/><a:defRPr/></a:pPr>
    <a:endParaRPr sz="2200"/>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Text != "" || got[0].Bullet != "•" || got[0].NoBullet || got[0].FontSize != 2200 {
		t.Fatalf("expected local buChar to preserve an empty bullet paragraph, got %+v", got[0])
	}
	if !got[0].HasMarginLeft || !got[0].HasIndent {
		t.Fatalf("expected bullet paragraph offsets from source pPr, got %+v", got[0])
	}
	if got[1].Bullet != "" || !got[1].NoBullet {
		t.Fatalf("expected local buNone to keep empty paragraph unbulleted, got %+v", got[1])
	}
}

func TestTextParagraphsFromNodeDoesNotApplyEndParagraphPropertiesToExistingRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1700"/><a:t>Visible</a:t></a:r><a:endParaRPr sz="1700" b="1"/></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].Bold || len(got[0].Runs) != 1 || got[0].Runs[0].Bold {
		t.Fatalf("endParaRPr should not restyle existing text runs, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphPropertiesForUnstyledRuns(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr/><a:t>Visible</a:t></a:r><a:endParaRPr sz="1700" b="1"/></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 1700 || !got[0].Bold {
		t.Fatalf("endParaRPr should seed paragraph defaults when runs are unstyled, got %+v", got[0])
	}
}

func TestTextParagraphsFromNodeUsesEndParagraphPropertiesWhenRunsOnlySetColor(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr><a:solidFill><a:srgbClr val="FF0000"/></a:solidFill></a:rPr><a:t>Visible</a:t></a:r>
    <a:endParaRPr sz="1700" b="1"/>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	if got[0].FontSize != 1700 || !got[0].Bold {
		t.Fatalf("color-only runs should still allow endParaRPr to seed missing defaults, got %+v", got[0])
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if !segment.Bold || !segment.HasTextColor || segment.TextColor.R != 0xff {
		t.Fatalf("endParaRPr defaults should not replace run color, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesMixedRunStyles(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"/><a:t>Energy services - </a:t></a:r><a:r><a:rPr sz="1800" b="1"/><a:t>Mobility</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected mixed run parse: %+v", got)
	}
	if got[0].Bold || got[0].Italic || got[0].FontSize != 1800 {
		t.Fatalf("paragraph-level metadata should preserve only uniform values: %+v", got[0])
	}
	if got[0].Runs[0].Text != "Energy services - " || got[0].Runs[0].Bold || got[0].Runs[1].Text != "Mobility" || !got[0].Runs[1].Bold {
		t.Fatalf("unexpected run styles: %+v", got[0].Runs)
	}
}

func TestExplicitRunBoldFalseOverridesInheritedBold(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:r><a:rPr b="0"/><a:t>Normal</a:t></a:r><a:r><a:rPr/><a:t>Inherited</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	got[0].Bold = true
	normal := runToSegment(got[0].Runs[0], got[0])
	inherited := runToSegment(got[0].Runs[1], got[0])
	if normal.Bold {
		t.Fatalf("explicit b=0 run should stay non-bold under inherited bold paragraph: %+v", normal)
	}
	if !inherited.Bold {
		t.Fatalf("run without explicit bold should inherit paragraph bold: %+v", inherited)
	}
}

func TestExplicitRunItalicFalseOverridesInheritedItalic(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:p><a:r><a:rPr i="0"/><a:t>Normal</a:t></a:r><a:r><a:rPr/><a:t>Inherited</a:t></a:r></a:p>
	</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	got[0].Italic = true
	normal := runToSegment(got[0].Runs[0], got[0])
	inherited := runToSegment(got[0].Runs[1], got[0])
	if normal.Italic {
		t.Fatalf("explicit i=0 run should stay non-italic under inherited italic paragraph: %+v", normal)
	}
	if !inherited.Italic {
		t.Fatalf("run without explicit italic should inherit paragraph italic: %+v", inherited)
	}
}

func TestTextParagraphsFromNodeCapturesMixedRunItalics(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"/><a:t>Regular </a:t></a:r><a:r><a:rPr sz="1800" i="1"/><a:t>Italic</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected mixed run parse: %+v", got)
	}
	if got[0].Italic {
		t.Fatalf("mixed italic runs should not promote the whole paragraph: %+v", got[0])
	}
	if got[0].Runs[0].Italic || !got[0].Runs[1].Italic {
		t.Fatalf("unexpected run italic styles: %+v", got[0].Runs)
	}
}

func TestTextParagraphsFromNodeCapturesRunBaseline(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="3200" b="1"/><a:t>CO</a:t></a:r><a:r><a:rPr sz="3200" b="1" baseline="-25000"/><a:t>2</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[1].Baseline != -25000 {
		t.Fatalf("expected subscript baseline to be preserved, got %+v", got[0].Runs[1])
	}
	if shift := segmentBaselineShift(textLineSegment{FontSize: 3200, Baseline: -25000}, 3200); shift >= 0 {
		t.Fatalf("expected negative baseline to produce downward drawing offset, got %d", shift)
	}
	if shift := segmentBaselineShiftAtDPI(textLineSegment{FontSize: 3200, Baseline: -25000}, 3200, 96); shift != -11 {
		t.Fatalf("expected 96 DPI baseline shift to scale from point size, got %d", shift)
	}
	segment := runToSegment(got[0].Runs[1], got[0])
	if segment.FontSize >= got[0].Runs[1].FontSize || segment.BaselineFontSize != got[0].Runs[1].FontSize {
		t.Fatalf("expected baseline run to render smaller while preserving shift font size, got %+v", segment)
	}
}

func TestBaselineRunWithoutLocalSizeUsesElementFallbackForRenderSize(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr sz="2200"/><a:t>SSA</a:t></a:r>
    <a:r><a:rPr baseline="30000"/><a:t>1</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[1].FontSize != 0 || got[0].Runs[1].Baseline != 30000 {
		t.Fatalf("expected inherited-size baseline run, got %+v", got[0].Runs[1])
	}
	segment := runToSegment(got[0].Runs[1], got[0])
	if segment.FontSize != 0 || segment.BaselineFontSize != 0 {
		t.Fatalf("run segment should preserve inherited-size baseline metadata until element fallback is known, got %+v", segment)
	}
	if got := segmentRenderFontSize(segment, 2200); got != scaledBaselineRunFontSize(2200) {
		t.Fatalf("expected inherited-size baseline run to scale fallback font size, got %d", got)
	}
	if shift := segmentBaselineShift(segment, 2200); shift <= 0 {
		t.Fatalf("expected inherited-size superscript baseline to use fallback font size for shift, got %d", shift)
	}

	faces := newFontFaceCache(false, "Arial")
	defer faces.Close()
	face, err := faces.Get(2200, false)
	if err != nil {
		t.Fatal(err)
	}
	scaledFace, err := faces.Get(segmentRenderFontSize(segment, 2200), false)
	if err != nil {
		t.Fatal(err)
	}
	if scaledFace.Metrics().Ascent.Ceil() >= face.Metrics().Ascent.Ceil() {
		t.Fatalf("expected inherited-size baseline face to be smaller, scaled=%+v unscaled=%+v", scaledFace.Metrics(), face.Metrics())
	}
}

func TestTextParagraphsFromNodeCapturesParagraphFontAlign(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr fontAlgn="b"/><a:r><a:t>Bottom aligned</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || got[0].FontAlign != "b" {
		t.Fatalf("expected paragraph fontAlgn to be preserved, got %+v", got)
	}
}

func TestTextParagraphsFromNodeInheritsListStyleFontAlign(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
	  <a:lstStyle><a:lvl1pPr fontAlgn="ctr"/></a:lstStyle>
	  <a:p><a:pPr lvl="0"/><a:r><a:t>Centered metrics</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || got[0].FontAlign != "ctr" {
		t.Fatalf("expected inherited paragraph fontAlgn to be preserved, got %+v", got)
	}
}

func TestTextParagraphsFromNodeCapturesParagraphLineBreakFlags(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr rtl="0" eaLnBrk="1" latinLnBrk="0" hangingPunct="1"/><a:r><a:t>Flags</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	paragraph := got[0]
	if !paragraph.HasRTL || paragraph.RTL {
		t.Fatalf("expected authored rtl=false to be preserved, got %+v", paragraph)
	}
	if !paragraph.HasEALineBreak || !paragraph.EALineBreak {
		t.Fatalf("expected eaLnBrk=true to be preserved, got %+v", paragraph)
	}
	if !paragraph.HasLatinLineBreak || paragraph.LatinLineBreak {
		t.Fatalf("expected latinLnBrk=false to be preserved, got %+v", paragraph)
	}
	if !paragraph.HasHangingPunct || !paragraph.HangingPunct {
		t.Fatalf("expected hangingPunct=true to be preserved, got %+v", paragraph)
	}
}

func TestTextParagraphsFromNodeInheritsParagraphLineBreakFlags(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:lstStyle><a:lvl1pPr rtl="1" eaLnBrk="0" latinLnBrk="1" hangingPunct="0"/></a:lstStyle>
  <a:p><a:pPr lvl="0"/><a:r><a:t>Inherited flags</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 {
		t.Fatalf("unexpected paragraph count: %+v", got)
	}
	paragraph := got[0]
	if !paragraph.HasRTL || !paragraph.RTL || !paragraph.HasEALineBreak || paragraph.EALineBreak || !paragraph.HasLatinLineBreak || !paragraph.LatinLineBreak || !paragraph.HasHangingPunct || paragraph.HangingPunct {
		t.Fatalf("expected inherited paragraph flags, got %+v", paragraph)
	}
}

func TestSegmentFontAlignmentShiftUsesLineMetrics(t *testing.T) {
	face := testMetricsFace{
		Face: basicfont.Face7x13,
		metrics: font.Metrics{
			Ascent:  fixed.I(8),
			Descent: fixed.I(2),
			Height:  fixed.I(10),
		},
	}
	line := measuredTextLine{Ascent: 12, Descent: 4}

	if got := segmentFontAlignmentShift(face, line, "t"); got != 4 {
		t.Fatalf("expected top font alignment shift 4, got %d", got)
	}
	if got := segmentFontAlignmentShift(face, line, "ctr"); got != 1 {
		t.Fatalf("expected center font alignment shift 1, got %d", got)
	}
	if got := segmentFontAlignmentShift(face, line, "b"); got != -2 {
		t.Fatalf("expected bottom font alignment shift -2, got %d", got)
	}
	if got := segmentFontAlignmentShift(face, line, "base"); got != 0 {
		t.Fatalf("expected baseline font alignment to preserve existing baseline, got %d", got)
	}
	if got := segmentFontAlignmentShift(face, line, "auto"); got != 0 {
		t.Fatalf("expected auto font alignment to preserve existing baseline, got %d", got)
	}
}

func TestTextParagraphsFromNodeCapturesRunHighlight(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:highlight><a:srgbClr val="FFFF00"/></a:highlight></a:rPr><a:t>Marked</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	run := got[0].Runs[0]
	if !run.HasHighlightColor || run.HighlightColor.R != 0xff || run.HighlightColor.G != 0xff || run.HighlightColor.B != 0x00 {
		t.Fatalf("expected highlight color to be preserved, got %+v", run)
	}
	segment := runToSegment(run, got[0])
	if !segment.HasHighlightColor || segment.HighlightColor != run.HighlightColor {
		t.Fatalf("expected highlight color on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesRunUnderline(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800" u="sng"/><a:t>Underlined</a:t></a:r><a:r><a:rPr sz="1800" u="none"/><a:t>Plain</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 2 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if !got[0].Runs[0].Underline || got[0].Runs[1].Underline {
		t.Fatalf("expected only single underline run, got %+v", got[0].Runs)
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if !segment.Underline {
		t.Fatalf("expected underline on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesDrawingMLUnderlineStrokeAndFill(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr sz="1800"><a:uLn/><a:uFill><a:solidFill><a:srgbClr val="00FF00"/></a:solidFill></a:uFill></a:rPr><a:t>Underlined</a:t></a:r>
    <a:r><a:rPr sz="1800" u="none"><a:uLnTx/><a:uFillTx/></a:rPr><a:t>Plain</a:t></a:r>
    <a:r><a:rPr sz="1800"><a:uLn><a:noFill/></a:uLn></a:rPr><a:t>No stroke</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 3 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if !got[0].Runs[0].Underline || !got[0].Runs[0].HasUnderlineColor || got[0].Runs[0].UnderlineColor != (color.RGBA{G: 0xff, A: 0xff}) {
		t.Fatalf("expected explicit underline stroke and fill, got %+v", got[0].Runs[0])
	}
	if got[0].Runs[1].Underline || got[0].Runs[2].Underline {
		t.Fatalf("u=none and no-fill underline stroke should not underline text: %+v", got[0].Runs)
	}
	segment := runToSegment(got[0].Runs[0], got[0])
	if !segment.Underline || !segment.HasUnderlineColor || segment.UnderlineColor != got[0].Runs[0].UnderlineColor {
		t.Fatalf("expected underline stroke and fill on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesRunStrikethrough(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:r><a:rPr sz="1800" strike="sngStrike"/><a:t>Single</a:t></a:r>
    <a:r><a:rPr sz="1800" strike="dblStrike"/><a:t>Double</a:t></a:r>
    <a:r><a:rPr sz="1800" strike="noStrike"/><a:t>Plain</a:t></a:r>
  </a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 3 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].Strike != "sngStrike" || got[0].Runs[1].Strike != "dblStrike" || got[0].Runs[2].Strike != "" {
		t.Fatalf("expected DrawingML strike enum to be preserved, got %+v", got[0].Runs)
	}
	segment := runToSegment(got[0].Runs[1], got[0])
	if segment.Strike != "dblStrike" {
		t.Fatalf("expected strike on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeKeepsParagraphAlignmentScoped(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:pPr algn="ctr"/><a:r><a:t>Centered</a:t></a:r></a:p>
  <a:p><a:r><a:t>Default</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 2 {
		t.Fatalf("unexpected paragraph parse: %+v", got)
	}
	if got[0].TextAlign != "ctr" || got[1].TextAlign != "" {
		t.Fatalf("paragraph alignment leaked across paragraphs: %+v", got)
	}
	element := parseSlideElementNode(&xmlNode{Name: "sp", Children: []*xmlNode{root}}, renderTransform{ScaleX: 1, ScaleY: 1})
	if element.TextAlign != "" {
		t.Fatalf("paragraph alignment should not promote to shape alignment, got %+v", element)
	}
}

func TestTextParagraphsFromNodeInheritsListStyleParagraphAlignment(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:bodyPr/>
  <a:lstStyle><a:lvl1pPr algn="ctr"/></a:lstStyle>
  <a:p><a:pPr/><a:r><a:t>Centered by style</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || got[0].TextAlign != "ctr" {
		t.Fatalf("expected list style paragraph alignment, got %+v", got)
	}
}

func TestTextParagraphsFromNodeCapturesRunFontFamily(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:latin typeface="Trebuchet MS"/></a:rPr><a:t>Label</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	run := got[0].Runs[0]
	if run.FontFamily != "Trebuchet MS" {
		t.Fatalf("expected run font family to be preserved, got %+v", run)
	}
	segment := runToSegment(run, got[0])
	if segment.FontFamily != "Trebuchet MS" {
		t.Fatalf("expected run font family on render segment, got %+v", segment)
	}
}

func TestTextParagraphsFromNodeCapturesRunLanguage(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p>
    <a:pPr><a:defRPr lang="en-US"/></a:pPr>
    <a:r><a:rPr lang="en-GB" sz="1800"/><a:t>Label</a:t></a:r>
  </a:p>
  <a:p><a:pPr><a:defRPr lang="fr-FR"/></a:pPr><a:r><a:rPr sz="1800"/><a:t>Defaut</a:t></a:r></a:p>
  <a:p><a:endParaRPr lang="es-ES" sz="1800"/></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 3 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Language != "en-US" || got[0].Runs[0].Language != "en-GB" {
		t.Fatalf("expected paragraph and run language to be preserved, got %+v", got[0])
	}
	if segment := runToSegment(got[0].Runs[0], got[0]); segment.Language != "en-GB" {
		t.Fatalf("expected run language to win on render segment, got %+v", segment)
	}
	if segment := runToSegment(got[1].Runs[0], got[1]); got[1].Language != "fr-FR" || segment.Language != "fr-FR" {
		t.Fatalf("expected paragraph default language on render segment, paragraph=%+v segment=%+v", got[1], segment)
	}
	if got[2].Language != "es-ES" || got[2].FontSize != 1800 {
		t.Fatalf("expected endParaRPr language on empty paragraph, got %+v", got[2])
	}
}

func TestTextParagraphsFromNodeUsesExplicitAlternateTypefaceForNonLatinText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>标题</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "Arial" {
		t.Fatalf("expected run fallback typeface to be preserved, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeDoesNotUseAlternateTypefaceForLatinText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>Label</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "" {
		t.Fatalf("latin text without a latin typeface should not use alternate font slots, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeDoesNotUseAlternateTypefaceForLatinTextWithMathSymbol(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:ea typeface="Arial"/><a:cs typeface="Calibri"/></a:rPr><a:t>value ≥99%</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "" {
		t.Fatalf("math symbols in Latin text should not switch the whole run to alternate font slots, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeUsesSymbolTypefaceForPrivateUseRunText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:latin typeface="Arial"/><a:sym typeface="Wingdings"/></a:rPr><a:t>&#xF0E0;</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "Wingdings" {
		t.Fatalf("private-use symbol text should use a:sym typeface, got %+v", got[0].Runs[0])
	}
}

func TestTextParagraphsFromNodeKeepsLatinTypefaceForMixedPrivateUseText(t *testing.T) {
	root, err := parseXMLNode([]byte(`<p:txBody xmlns:p="p" xmlns:a="a">
  <a:p><a:r><a:rPr sz="1800"><a:latin typeface="Arial"/><a:sym typeface="Wingdings"/></a:rPr><a:t>Go &#xF0E0; there</a:t></a:r></a:p>
</p:txBody>`))
	if err != nil {
		t.Fatal(err)
	}

	got := textParagraphsFromNode(root)
	if len(got) != 1 || len(got[0].Runs) != 1 {
		t.Fatalf("unexpected run parse: %+v", got)
	}
	if got[0].Runs[0].FontFamily != "Arial" {
		t.Fatalf("latin typeface should still win for mixed text runs, got %+v", got[0].Runs[0])
	}
}

func TestDrawTextUnderlinePaintsBelowBaseline(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	face, err := openFontFace(1800, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()
	drawTextUnderline(img, face, 10, 20, 60, color.RGBA{R: 255, A: 255})
	if !hasColorPixel(img, color.RGBA{R: 255, A: 255}) {
		t.Fatal("expected underline to paint red pixels")
	}
}

func TestUnderlineColorForSegmentUsesExplicitUnderlineFill(t *testing.T) {
	segment := textLineSegment{
		HasTextColor:      true,
		TextColor:         color.RGBA{R: 0xff, A: 0xff},
		HasUnderlineColor: true,
		UnderlineColor:    color.RGBA{G: 0xff, A: 0xff},
	}
	if got := underlineColorForSegment(segment, color.RGBA{B: 0xff, A: 0xff}); got != segment.UnderlineColor {
		t.Fatalf("expected explicit underline color, got %#v", got)
	}
	segment.HasUnderlineColor = false
	if got := underlineColorForSegment(segment, color.RGBA{B: 0xff, A: 0xff}); got != segment.TextColor {
		t.Fatalf("expected underline color to follow text color, got %#v", got)
	}
}

func TestDrawTextStrikethroughPaintsThroughTextMiddle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	face, err := openFontFace(1800, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	baseline := 24
	metrics := face.Metrics()
	center := baseline - metrics.Ascent.Ceil() + (metrics.Ascent.Ceil()+metrics.Descent.Ceil())/2
	drawTextStrikethrough(img, face, 10, baseline, 60, "sngStrike", color.RGBA{R: 255, A: 255})
	if got := img.RGBAAt(20, center); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("expected single strikethrough at text middle y=%d, got %#v", center, got)
	}
}

func TestDrawTextStrikethroughPaintsDoubleStrike(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 120, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	face, err := openFontFace(1800, false, false, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	defer face.Close()

	baseline := 24
	metrics := face.Metrics()
	center := baseline - metrics.Ascent.Ceil() + (metrics.Ascent.Ceil()+metrics.Descent.Ceil())/2
	lineWidth := maxInt(1, metrics.Height.Ceil()/16)
	gap := maxInt(2, lineWidth*2)
	drawTextStrikethrough(img, face, 10, baseline, 60, "dblStrike", color.RGBA{R: 255, A: 255})
	if got := img.RGBAAt(20, center-gap/2); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("expected upper double strikethrough at y=%d, got %#v", center-gap/2, got)
	}
	if got := img.RGBAAt(20, center+gap/2); got != (color.RGBA{R: 255, A: 255}) {
		t.Fatalf("expected lower double strikethrough at y=%d, got %#v", center+gap/2, got)
	}
}

func TestTextRenderLinesCarryParagraphLineSpacing(t *testing.T) {
	lines := textLayoutStyledParagraphLines(nil, nil, []textParagraph{{
		Text:           "First",
		LineSpacingPct: 90000,
	}}, "", 200, "none")
	if len(lines) != 1 || lines[0].LineSpacingPct != 90000 {
		t.Fatalf("expected line spacing on rendered line, got %+v", lines)
	}
}

func TestApplyLineSpacingScalesHeight(t *testing.T) {
	if got := applyLineSpacing(50, 90000); got != 45 {
		t.Fatalf("unexpected scaled line height: %d", got)
	}
	if got := applyLineSpacing(50, 0); got != 50 {
		t.Fatalf("unexpected default line height: %d", got)
	}
}

func TestApplyLineSpacingUsesDrawingMLFontSizeForPercentSpacing(t *testing.T) {
	if got := applyLineSpacingAtDPI(32, 150000, 1700, 72); got != 26 {
		t.Fatalf("expected 150%% line spacing from 17pt font size, got %d", got)
	}
	if got := applyLineSpacingAtDPI(32, 150000, 1700, 96); got != 35 {
		t.Fatalf("expected 96-DPI line spacing from 17pt font size, got %d", got)
	}
	if got := applyLineSpacingAtDPI(32, 100000, 1700, 72); got != 17 {
		t.Fatalf("100%% explicit line spacing should use DrawingML font-size spacing, got %d", got)
	}
}

func TestVisibleLineAdvanceKeepsTightSpacingFromCollidingGlyphs(t *testing.T) {
	line := measuredTextLine{Ascent: 22, Descent: 6}
	if got := visibleLineAdvance(22, line); got != 28 {
		t.Fatalf("line advance shorter than ink extents should grow to 28, got %d", got)
	}
	if got := visibleLineAdvance(32, line); got != 32 {
		t.Fatalf("line advance taller than ink extents should be preserved, got %d", got)
	}
}

func TestParseSpacingPercentAcceptsDrawingMLPercentString(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:lnSpc xmlns:a="a"><a:spcPct val="92.5%"/></a:lnSpc>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := parseSpacingPercent(root); got != 92500 {
		t.Fatalf("expected DrawingML percent string to parse as thousandths, got %d", got)
	}

	root, err = parseXMLNode([]byte(`<a:spcBef xmlns:a="a"><a:spcPct val="110%"/></a:spcBef>`))
	if err != nil {
		t.Fatal(err)
	}
	_, pct := parseSpacingValue(root)
	if pct != 110000 {
		t.Fatalf("expected paragraph spacing percent string to parse as thousandths, got %d", pct)
	}
}

func TestParagraphSpacingPercentPixelsScalesFontSize(t *testing.T) {
	if got := paragraphSpacingPercentPixels(90000, 2000); got != 18 {
		t.Fatalf("expected 90%% paragraph spacing from 20pt text, got %d", got)
	}
	if got := paragraphSpacingPercentPixelsAtDPI(110000, 1800, 96); got != 26 {
		t.Fatalf("expected 110%% paragraph spacing from 18pt text at 96 DPI, got %d", got)
	}
	if got := paragraphSpacingPercentPixels(0, 2000); got != 0 {
		t.Fatalf("expected zero paragraph spacing, got %d", got)
	}
}

func TestTextCharacterSpacingUsesDrawingMLTextPointUnits(t *testing.T) {
	if got := textCharacterSpacingPixelsAtDPI(100, 72); got != 1 {
		t.Fatalf("expected one point of character spacing to equal one pixel at 72 DPI, got %d", got)
	}
	if got := textCharacterSpacingPixelsAtDPI(150, 96); got != 2 {
		t.Fatalf("expected 1.5 points of character spacing to round at 96 DPI, got %d", got)
	}
	if got := textCharacterSpacingAdvance("ABC", 2); got != 4 {
		t.Fatalf("expected spacing between characters only, got %d", got)
	}
}

type testKerningFace struct {
	font.Face
}

type testMetricsFace struct {
	font.Face
	metrics font.Metrics
}

func (face testMetricsFace) Metrics() font.Metrics {
	return face.metrics
}

func (face testKerningFace) Kern(r0 rune, r1 rune) fixed.Int26_6 {
	return fixed.I(4)
}

func TestFaceWithSegmentKerningHonorsDrawingMLKernThreshold(t *testing.T) {
	base := basicfont.Face7x13
	kerned := testKerningFace{Face: base}
	unrestricted := textLineSegment{Text: "AV", FontSize: 1100}
	disabled := textLineSegment{Text: "AV", FontSize: 1100, HasKern: true, KernMinFontSize: 1200}
	enabled := textLineSegment{Text: "AV", FontSize: 1200, HasKern: true, KernMinFontSize: 1200}

	baseWidth := measureString(base, "AV")
	if got := measureString(faceWithSegmentKerning(kerned, unrestricted), "AV"); got == baseWidth {
		t.Fatalf("expected default font kerning to remain active, got width %d", got)
	}
	if got := measureString(faceWithSegmentKerning(kerned, disabled), "AV"); got != baseWidth {
		t.Fatalf("expected DrawingML kern threshold to disable kerning below 12pt, got width %d want %d", got, baseWidth)
	}
	if got := measureString(faceWithSegmentKerning(kerned, enabled), "AV"); got == baseWidth {
		t.Fatalf("expected DrawingML kern threshold to allow kerning at 12pt, got width %d", got)
	}
}

func TestTextRunFromNodeReadsDrawingMLKernThreshold(t *testing.T) {
	root, err := parseXMLNode([]byte(`<a:r xmlns:a="a"><a:rPr kern="1200"/><a:t>AV</a:t></a:r>`))
	if err != nil {
		t.Fatal(err)
	}
	got := textRunFromNode(root, "AV")
	if !got.HasKern || got.KernMinFontSize != 1200 {
		t.Fatalf("expected run kern threshold to be parsed, got %+v", got)
	}
}

func TestMeasureStyledSegmentsIncludesCharacterSpacing(t *testing.T) {
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}
	base := measureString(face, "ABC")
	got, err := measureStyledSegmentsAtDPI(faces, face, face, []textLineSegment{{Text: "ABC", FontSize: 1800, CharSpacing: 100}}, 72)
	if err != nil {
		t.Fatal(err)
	}
	if got != base+2 {
		t.Fatalf("expected two character-spacing advances for three characters, got %d want %d", got, base+2)
	}
}

func TestLineFontSizeUsesLargestSegmentFontSize(t *testing.T) {
	got := lineFontSize(textRenderLine{FontSize: 1200, Segments: []textLineSegment{
		{Text: "small", FontSize: 1000},
		{Text: "large", FontSize: 2200},
		{Text: "fallback"},
	}}, 1800)
	if got != 2200 {
		t.Fatalf("expected largest explicit segment font size, got %d", got)
	}
}

func TestMeasureTextRenderLinesUsesFontLineMetricHeight(t *testing.T) {
	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()

	face, err := faces.Get(1800, false, false)
	if err != nil {
		t.Fatal(err)
	}
	metrics := face.Metrics()
	want := visibleLineAdvance(defaultLineMetricHeight(metrics), measuredTextLine{
		Ascent:  metrics.Ascent.Ceil(),
		Descent: metrics.Descent.Ceil(),
	})

	got, err := measureTextRenderLines(faces, []textRenderLine{{Text: "A", FontSize: 1800}}, 1800)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Height != want {
		t.Fatalf("expected font line metric height %d, got %+v", want, got)
	}
}
