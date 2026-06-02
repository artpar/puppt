package render

import (
	"strings"
	"testing"
)

type fakeM08TextShapingBackend struct {
	advance int
	inputs  []textShapingInput
}

func (backend *fakeM08TextShapingBackend) ShapeText(input textShapingInput) (textShapingOutput, bool) {
	backend.inputs = append(backend.inputs, input)
	return textShapingOutput{Advance: backend.advance, Glyphs: len([]rune(input.Text)), FontLabel: "fake"}, true
}

func TestM08MeasureStyledSegmentsUsesShapingBackend(t *testing.T) {
	previous := currentTextShapingBackend
	backend := &fakeM08TextShapingBackend{advance: 37}
	currentTextShapingBackend = backend
	defer func() { currentTextShapingBackend = previous }()

	faces := newFontFaceCache(false, "Carlito")
	defer faces.Close()
	face, err := faces.Get(1800, false)
	if err != nil {
		t.Fatal(err)
	}

	got, err := measureStyledSegmentsAtDPI(faces, face, face, []textLineSegment{{
		Text:       "office",
		FontFamily: "Carlito",
		FontSize:   1800,
	}}, defaultOutputDPI)
	if err != nil {
		t.Fatal(err)
	}
	if got != 37 {
		t.Fatalf("expected shaped advance to drive segment measurement, got %d", got)
	}
	if len(backend.inputs) != 1 || backend.inputs[0].Text != "office" || backend.inputs[0].FontFamily != "Carlito" {
		t.Fatalf("unexpected shaping inputs: %+v", backend.inputs)
	}
}

func TestM08HarfbuzzBackendShapesSupportedLTRText(t *testing.T) {
	output, ok := (harfbuzzTextShapingBackend{}).ShapeText(textShapingInput{
		Text:       "office",
		FontFamily: "Carlito",
		FontSize:   1800,
		DPI:        defaultOutputDPI,
	})
	if !ok || output.Advance <= 0 || output.Glyphs <= 0 || output.FontLabel == "" {
		t.Fatalf("expected supported LTR shaping output, ok=%v output=%+v", ok, output)
	}
}

func TestM08HarfbuzzBackendDeclinesRTLUntilBidiIsImplemented(t *testing.T) {
	if _, ok := (harfbuzzTextShapingBackend{}).ShapeText(textShapingInput{
		Text:       "אבג",
		FontFamily: "Carlito",
		FontSize:   1800,
		DPI:        defaultOutputDPI,
	}); ok {
		t.Fatalf("RTL text should not be silently treated as supported LTR shaping")
	}
}

func TestM08TextPrimitiveReportsFontResolutionAndTextUnsupportedModes(t *testing.T) {
	element := slideElement{
		Kind:                      "sp",
		ID:                        "5",
		Name:                      "M08 Text",
		Text:                      "אבג",
		FontFamily:                "Missing Office Font",
		FontSize:                  1800,
		TextVertical:              "eaVert",
		TextColumnCount:           2,
		HasTextRightToLeftColumns: true,
		TextRightToLeftColumns:    true,
		HasTextTransform:          true,
		TextParagraphs: []textParagraph{{
			Text:       "אבג",
			FontFamily: "Missing Office Font",
			FontSize:   1800,
			Runs: []textRun{{
				Text:       "אבג",
				FontFamily: "Missing Office Font",
				FontSize:   1800,
			}},
		}},
	}
	primitive := renderTextPrimitiveFromElement(renderPrimitiveProvenance{SourcePart: "ppt/slides/slide1.xml"}, element)

	if len(primitive.Paragraphs) != 1 || primitive.Paragraphs[0].Text != "אבג" {
		t.Fatalf("text primitive did not preserve paragraph source model: %+v", primitive.Paragraphs)
	}
	if !strings.Contains(strings.Join(primitive.FontResolution, "\n"), "generic fallback font") {
		t.Fatalf("expected font fallback report in text primitive, got %+v", primitive.FontResolution)
	}
	unsupported := strings.Join(primitive.Unsupported, "\n")
	if !primitive.HasRTLColumns || !primitive.RTLColumns {
		t.Fatalf("text primitive did not preserve rtlCol source metadata: %+v", primitive)
	}
	for _, want := range []string{"vertical mode", "columns", "right-to-left column order", "bidirectional/RTL"} {
		if !strings.Contains(unsupported, want) {
			t.Fatalf("expected %q in text primitive unsupported reports, got %s", want, unsupported)
		}
	}
}

func TestM08TextLayoutReportsBidiFallback(t *testing.T) {
	got := strings.Join(textLayoutUnsupportedMessages(slideElement{
		Text: "אבג",
		TextParagraphs: []textParagraph{{
			Text: "אבג",
			Runs: []textRun{{Text: "אבג"}},
		}},
	}), "\n")
	if !strings.Contains(got, "bidirectional/RTL") || !strings.Contains(got, "CT_RegularTextRun") {
		t.Fatalf("expected precise bidi fallback report, got %s", got)
	}
}

func TestM08TextLayoutReportsAuthoredRTLParagraphFallback(t *testing.T) {
	got := strings.Join(textLayoutUnsupportedMessages(slideElement{
		Text: "Latin text",
		TextParagraphs: []textParagraph{{
			Text:   "Latin text",
			HasRTL: true,
			RTL:    true,
			Runs:   []textRun{{Text: "Latin text"}},
		}},
	}), "\n")
	if !strings.Contains(got, "paragraph rtl=1") || !strings.Contains(got, "CT_TextParagraphProperties@rtl") {
		t.Fatalf("expected authored paragraph RTL fallback report, got %s", got)
	}
}
