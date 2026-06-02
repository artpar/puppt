package render

import (
	"bytes"
	"errors"
	"math"
	"sync"
	"unicode"

	"github.com/go-text/typesetting/di"
	gtfont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

type textShapingInput struct {
	Text        string
	FontFamily  string
	FontSize    int
	Bold        bool
	Italic      bool
	CharSpacing int
	DPI         int
	PointScale  float64
}

type textShapingOutput struct {
	Advance   int
	Glyphs    int
	FontLabel string
}

type textShapingBackend interface {
	ShapeText(input textShapingInput) (textShapingOutput, bool)
}

type harfbuzzTextShapingBackend struct{}

var currentTextShapingBackend textShapingBackend = harfbuzzTextShapingBackend{}
var textShapingFontSourceCache sync.Map
var textShapingFaceCache sync.Map
var textShapingHarfbuzzShaperPool sync.Pool

func (harfbuzzTextShapingBackend) ShapeText(input textShapingInput) (textShapingOutput, bool) {
	if input.Text == "" || textContainsRTL(input.Text) {
		return textShapingOutput{}, false
	}
	source, err := cachedTextShapingFontSource(input.FontFamily, input.Bold, input.Italic)
	if err != nil {
		return textShapingOutput{}, false
	}
	face, err := cachedGoTextShapingFaceFromFontSource(source)
	if err != nil {
		return textShapingOutput{}, false
	}
	fontSize := input.FontSize
	if fontSize <= 0 {
		fontSize = 1800
	}
	pointSize := fallbackFontPointSizeWithScaleAndFamily(fontSize, input.Bold, input.Italic, input.PointScale, input.FontFamily)
	pixelSize := pointSize * float64(normalizeOutputDPI(input.DPI)) / 72
	runes := []rune(input.Text)
	shaper := pooledTextShapingHarfbuzzShaper()
	defer textShapingHarfbuzzShaperPool.Put(shaper)
	output := shaper.Shape(shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      fixed.Int26_6(math.Round(pixelSize * 64)),
		Script:    language.Latin,
		Language:  language.DefaultLanguage(),
	})
	if spacing := textCharacterSpacingPixelsAtDPI(input.CharSpacing, input.DPI); spacing != 0 {
		outputs := []shaping.Output{output}
		shaping.AddSpacing(outputs, runes, 0, fixed.I(spacing))
		output = outputs[0]
	}
	return textShapingOutput{
		Advance:   int(math.Round(float64(output.Advance) / 64)),
		Glyphs:    len(output.Glyphs),
		FontLabel: source.Label,
	}, true
}

func pooledTextShapingHarfbuzzShaper() *shaping.HarfbuzzShaper {
	if cached := textShapingHarfbuzzShaperPool.Get(); cached != nil {
		return cached.(*shaping.HarfbuzzShaper)
	}
	return &shaping.HarfbuzzShaper{}
}

func cachedTextShapingFontSource(fontFamily string, bold bool, italic bool) (fontSource, error) {
	key := normalizedFontFamily(fontFamily) + ":" + fontStyleKey(bold, italic)
	if cached, ok := textShapingFontSourceCache.Load(key); ok {
		return cached.(fontSource), nil
	}
	source, err := resolveFontSource(fontFamily, bold, italic)
	if err != nil {
		return fontSource{}, err
	}
	actual, _ := textShapingFontSourceCache.LoadOrStore(key, source)
	return actual.(fontSource), nil
}

func cachedGoTextShapingFaceFromFontSource(source fontSource) (*gtfont.Face, error) {
	if source.Label != "" {
		if cached, ok := textShapingFaceCache.Load(source.Label); ok {
			return cached.(*gtfont.Face), nil
		}
	}
	face, err := goTextShapingFaceFromFontSource(source)
	if err != nil {
		return nil, err
	}
	if source.Label == "" {
		return face, nil
	}
	actual, _ := textShapingFaceCache.LoadOrStore(source.Label, face)
	return actual.(*gtfont.Face), nil
}

func goTextShapingFaceFromFontSource(source fontSource) (*gtfont.Face, error) {
	faces, err := gtfont.ParseTTC(bytes.NewReader(source.Data))
	if err != nil {
		return nil, err
	}
	if len(faces) == 0 {
		return nil, errors.New("font collection has no faces")
	}
	return faces[0], nil
}

func shapeTextSegmentAdvanceAtDPI(faces *fontFaceCache, segment textLineSegment, dpi int) (int, bool) {
	if currentTextShapingBackend == nil || segment.Marker != "" || segment.Text == "" {
		return 0, false
	}
	if segment.HasKern && segment.KernMinFontSize > 0 && segment.FontSize > 0 && segment.FontSize < segment.KernMinFontSize {
		return 0, false
	}
	family := segment.FontFamily
	if family == "" && faces != nil {
		family = faces.FontFamily
	}
	pointScale := 0.0
	if faces != nil {
		pointScale = faces.PointScale
	}
	output, ok := currentTextShapingBackend.ShapeText(textShapingInput{
		Text:        segment.Text,
		FontFamily:  family,
		FontSize:    segment.FontSize,
		Bold:        segment.Bold,
		Italic:      segment.Italic,
		CharSpacing: segment.CharSpacing,
		DPI:         dpi,
		PointScale:  pointScale,
	})
	if !ok {
		return 0, false
	}
	return output.Advance, true
}

func textContainsRTL(text string) bool {
	for _, value := range text {
		if unicode.In(value,
			unicode.Arabic,
			unicode.Hebrew,
			unicode.Syriac,
			unicode.Thaana,
			unicode.Nko,
			unicode.Adlam,
			unicode.Mandaic,
			unicode.Manichaean,
			unicode.Mende_Kikakui,
			unicode.Nabataean,
			unicode.Old_South_Arabian,
			unicode.Old_North_Arabian,
			unicode.Palmyrene,
			unicode.Phoenician,
			unicode.Psalter_Pahlavi,
			unicode.Samaritan) {
			return true
		}
	}
	return false
}

func elementContainsRTLText(element slideElement) bool {
	if textContainsRTL(element.Text) {
		return true
	}
	for _, paragraph := range element.TextParagraphs {
		if textContainsRTL(paragraph.Text) {
			return true
		}
		for _, run := range paragraph.Runs {
			if textContainsRTL(run.Text) {
				return true
			}
		}
	}
	return false
}

func elementContainsAuthoredRTLParagraph(element slideElement) bool {
	for _, paragraph := range element.TextParagraphs {
		if paragraph.HasRTL && paragraph.RTL {
			return true
		}
	}
	for _, row := range element.Table.Rows {
		for _, cell := range row.Cells {
			for _, paragraph := range cell.TextParagraphs {
				if paragraph.HasRTL && paragraph.RTL {
					return true
				}
			}
		}
	}
	return false
}
