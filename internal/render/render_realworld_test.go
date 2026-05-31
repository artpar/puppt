package render

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

func TestFilterInheritedPlaceholdersDropsDisabledSlideNumberPlaceholder(t *testing.T) {
	settings := defaultHeaderFooterSettings()
	settings.SlideNumber = false
	got := filterInheritedPlaceholdersForRender([]slideElement{{
		Kind:            "sp",
		Text:            "‹#›",
		IsPlaceholder:   true,
		PlaceholderType: "sldNum",
	}}, nil, settings, true)
	if len(got) != 0 {
		t.Fatalf("disabled slide-number placeholder should not render, got %+v", got)
	}
}

func TestInheritedHeaderFooterSettingsHonorsExplicitFalse(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"><p:hf sldNum="0" dt="0"/></p:sldMaster>`),
	}}
	got := inheritedHeaderFooterSettings(pkg, []string{"ppt/slideMasters/slideMaster1.xml"})
	if got.SlideNumber || got.DateTime || !got.Footer || !got.Header {
		t.Fatalf("unexpected inherited header/footer settings: %+v", got)
	}
}

func TestInheritedHeaderFooterSettingsTreatsMissingElementAsDisabled(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slideMasters/slideMaster1.xml": []byte(`<p:sldMaster xmlns:p="p"/>`),
	}}
	got := inheritedHeaderFooterSettings(pkg, []string{"ppt/slideMasters/slideMaster1.xml"})
	if got.SlideNumber || got.DateTime || got.Footer || got.Header {
		t.Fatalf("missing hf element should not enable inherited placeholders, got %+v", got)
	}
}

func TestPresentationShowsSpecialPlaceholdersOnTitleSlideDefaultsOn(t *testing.T) {
	pkg := &pptx.Package{
		PresentationPath: "ppt/presentation.xml",
		Parts: map[string][]byte{
			"ppt/presentation.xml": []byte(`<p:presentation xmlns:p="p"/>`),
		},
	}
	if !presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) {
		t.Fatal("missing showSpecialPlsOnTitleSld should default to showing special placeholders")
	}
}

func TestPresentationShowsSpecialPlaceholdersOnTitleSlideReadsFalse(t *testing.T) {
	pkg := &pptx.Package{
		PresentationPath: "ppt/presentation.xml",
		Parts: map[string][]byte{
			"ppt/presentation.xml": []byte(`<p:presentation xmlns:p="p" showSpecialPlsOnTitleSld="0"/>`),
		},
	}
	if presentationShowsSpecialPlaceholdersOnTitleSlide(pkg) {
		t.Fatal("showSpecialPlsOnTitleSld=0 should hide special placeholders on title slides")
	}
}

func TestSlideUsesTitleLayoutReadsLayoutType(t *testing.T) {
	pkg := &pptx.Package{Parts: map[string][]byte{
		"ppt/slides/_rels/slide1.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
</Relationships>`),
		"ppt/slides/_rels/slide2.xml.rels": []byte(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout2.xml"/>
</Relationships>`),
		"ppt/slideLayouts/slideLayout1.xml": []byte(`<p:sldLayout xmlns:p="p" type="title"/>`),
		"ppt/slideLayouts/slideLayout2.xml": []byte(`<p:sldLayout xmlns:p="p" type="obj"/>`),
	}}
	if !slideUsesTitleLayout(pkg, "ppt/slides/slide1.xml") {
		t.Fatal("expected title layout to be detected")
	}
	if slideUsesTitleLayout(pkg, "ppt/slides/slide2.xml") {
		t.Fatal("non-title layout should not be treated as a title slide")
	}
}

func TestFormatUnsupportedSummarySortsDeterministically(t *testing.T) {
	got := formatUnsupportedSummary(map[string]int{
		"later alphabetically":   2,
		"earlier alphabetically": 2,
		"most common":            3,
	}, 2)
	want := "3x most common; 2x earlier alphabetically"
	if got != want {
		t.Fatalf("unexpected unsupported summary: got=%q want=%q", got, want)
	}
	if got := formatUnsupportedSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty unsupported summary, got %q", got)
	}
}

func TestFormatSlideDiffSummarySortsDeterministically(t *testing.T) {
	got := formatSlideDiffSummary([]realWorldSlideDiff{
		{Label: "deck slide 002", DifferentPixels: 20, Unsupported: 3, Status: "partial"},
		{Label: "deck slide 001", DifferentPixels: 20, Unsupported: 2, Status: "partial"},
		{Label: "deck slide 003", DifferentPixels: 30, Unsupported: 1, Status: "partial"},
	}, 2)
	want := "deck slide 003=30px/1 unsupported/partial; deck slide 001=20px/2 unsupported/partial"
	if got != want {
		t.Fatalf("unexpected slide diff summary: got=%q want=%q", got, want)
	}
	if got := formatSlideDiffSummary(nil, 8); got != "none" {
		t.Fatalf("expected empty slide diff summary, got %q", got)
	}
}

func TestRealWorldGoldenComparison(t *testing.T) {
	if os.Getenv("PUPPT_RUN_REALWORLD_RENDER_TESTS") != "1" {
		t.Skip("set PUPPT_RUN_REALWORLD_RENDER_TESTS=1 to run the 61-slide renderer golden comparison")
	}

	referenceRoot := realWorldReferenceRoot()
	manifestPath := filepath.Join(referenceRoot, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest referenceManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	total := 0
	failures := 0
	totalDifferentPixels := int64(0)
	worstDifferentPixels := int64(0)
	worstLabel := ""
	unsupportedCounts := map[string]int{}
	var slideDiffs []realWorldSlideDiff
	for _, deck := range manifest.Decks {
		if deck.Input == "" {
			t.Fatalf("manifest deck missing input: %+v", deck)
		}
		for index, slide := range deck.Slides {
			total++
			outputPath := filepath.Join(t.TempDir(), filepath.Base(deck.Input)+"-slide.png")
			result, err := Render(context.Background(), filepath.Join("..", "..", deck.Input), Options{
				SlideNumber: index + 1,
				OutputPath:  outputPath,
				DPI:         referenceDPIForSlide(slide.Width),
			})
			if err != nil {
				t.Fatalf("render %s slide %d: %v", deck.Input, index+1, err)
			}
			referencePath := filepath.Join(referenceRoot, slide.File)
			diff, err := comparePNG(outputPath, referencePath)
			if err != nil {
				t.Fatalf("compare %s slide %d: %v", deck.Input, index+1, err)
			}
			if diff.Width != slide.Width || diff.Height != slide.Height {
				t.Fatalf("manifest dimensions disagree for %s: manifest=%dx%d reference=%dx%d", slide.File, slide.Width, slide.Height, diff.Width, diff.Height)
			}
			if diff.GotWidth != slide.Width || diff.GotHeight != slide.Height {
				t.Fatalf("render dimensions disagree for %s slide %d: got=%dx%d reference=%dx%d", deck.Input, index+1, diff.GotWidth, diff.GotHeight, slide.Width, slide.Height)
			}
			if diff.DifferentPixels != 0 {
				writeRealWorldDiffArtifacts(t, outputPath, referencePath, deck.Input, index+1, result, diff)
				failures++
				totalDifferentPixels += int64(diff.DifferentPixels)
				for _, item := range result.Unsupported {
					unsupportedCounts[item.Message]++
				}
				if int64(diff.DifferentPixels) > worstDifferentPixels {
					worstDifferentPixels = int64(diff.DifferentPixels)
					worstLabel = fmt.Sprintf("%s slide %03d", deck.Input, index+1)
				}
				slideDiffs = append(slideDiffs, realWorldSlideDiff{
					Label:           fmt.Sprintf("%s slide %03d", deck.Input, index+1),
					DifferentPixels: diff.DifferentPixels,
					Unsupported:     len(result.Unsupported),
					Status:          result.Status,
				})
				if failures <= 10 {
					t.Logf("%s slide %03d: %d differing pixel(s), unsupported=%d, status=%s", deck.Input, index+1, diff.DifferentPixels, len(result.Unsupported), result.Status)
				}
			}
		}
	}
	if total != 61 {
		t.Fatalf("expected 61 real-world reference slides, got %d", total)
	}
	if failures != 0 {
		t.Fatalf("%d of %d real-world slides differ from %s references; total differing pixels=%d; worst=%s with %d differing pixels; worst slides: %s; top unsupported rendering gaps: %s", failures, total, referenceRoot, totalDifferentPixels, worstLabel, worstDifferentPixels, formatSlideDiffSummary(slideDiffs, 8), formatUnsupportedSummary(unsupportedCounts, 8))
	}
}

type realWorldSlideDiff struct {
	Label           string
	DifferentPixels int
	Unsupported     int
	Status          string
}

func formatSlideDiffSummary(items []realWorldSlideDiff, limit int) string {
	if len(items) == 0 || limit <= 0 {
		return "none"
	}
	items = append([]realWorldSlideDiff(nil), items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].DifferentPixels != items[j].DifferentPixels {
			return items[i].DifferentPixels > items[j].DifferentPixels
		}
		return items[i].Label < items[j].Label
	})
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%s=%dpx/%d unsupported/%s", item.Label, item.DifferentPixels, item.Unsupported, item.Status))
	}
	return builder.String()
}

func formatUnsupportedSummary(counts map[string]int, limit int) string {
	if len(counts) == 0 || limit <= 0 {
		return "none"
	}
	items := make([]unsupportedSummaryItem, 0, len(counts))
	for message, count := range counts {
		if count <= 0 {
			continue
		}
		items = append(items, unsupportedSummaryItem{Message: message, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count != items[j].Count {
			return items[i].Count > items[j].Count
		}
		return items[i].Message < items[j].Message
	})
	if len(items) == 0 {
		return "none"
	}
	if len(items) > limit {
		items = items[:limit]
	}
	var builder strings.Builder
	for index, item := range items {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(fmt.Sprintf("%dx %s", item.Count, item.Message))
	}
	return builder.String()
}

type unsupportedSummaryItem struct {
	Message string
	Count   int
}

func realWorldReferenceRoot() string {
	if root := os.Getenv("PUPPT_REALWORLD_REFERENCE_ROOT"); root != "" {
		return root
	}
	return filepath.Join("..", "..", "testdata", "realworld-ppts", "reference-renders", "manual-using-apple-note")
}

func TestWriteRealWorldDiffArtifactsWritesMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PUPPT_REALWORLD_ARTIFACT_DIR", dir)
	gotPath := filepath.Join(dir, "got-source.png")
	referencePath := filepath.Join(dir, "reference-source.png")
	got := image.NewRGBA(image.Rect(0, 0, 2, 1))
	got.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	got.SetRGBA(1, 0, color.RGBA{G: 255, A: 255})
	reference := image.NewRGBA(image.Rect(0, 0, 2, 1))
	reference.SetRGBA(0, 0, color.RGBA{R: 255, A: 255})
	reference.SetRGBA(1, 0, color.RGBA{B: 255, A: 255})
	if err := writePNG(gotPath, got); err != nil {
		t.Fatal(err)
	}
	if err := writePNG(referencePath, reference); err != nil {
		t.Fatal(err)
	}

	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       commandName,
		Status:        "partial",
		Unsupported: []model.SkipItem{{
			Code:    partialUnsupportedCode,
			Message: "shape text was rendered with simplified layout",
			Part:    "ppt/slides/slide1.xml",
		}},
	}
	diff, err := comparePNG(gotPath, referencePath)
	if err != nil {
		t.Fatal(err)
	}
	writeRealWorldDiffArtifacts(t, gotPath, referencePath, "testdata/realworld-ppts/example.pptx", 1, result, diff)

	slideDir := filepath.Join(dir, "example", "slide-001")
	for _, name := range []string{"got.png", "reference.png", "diff.png", "result.json", "diff.json"} {
		if _, err := os.Stat(filepath.Join(slideDir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(slideDir, "diff.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"different_pixels": 1`) {
		t.Fatalf("diff artifact did not include pixel count: %s", data)
	}
	if !strings.Contains(string(data), `"max_channel_delta_8bit": 255`) || !strings.Contains(string(data), `"different_bounds"`) {
		t.Fatalf("diff artifact did not include channel and bounds diagnostics: %s", data)
	}
	data, err = os.ReadFile(filepath.Join(slideDir, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "simplified layout") {
		t.Fatalf("result artifact did not include unsupported details: %s", data)
	}
}

func TestRendererImplementationHasNoTargetDeckHardcodesOrExternalRendererCalls(t *testing.T) {
	forbidden := []string{
		"Apple Notes",
		"manual-using-apple-note",
		"reference-renders",
		"LibreOffice",
		"soffice",
		"PowerPoint",
		"Keynote",
		"Google Slides",
		"chromedp",
		"playwright",
		"selenium",
		"exec.Command",
		"WHO-HIV-testing-algorithms-toolkit",
		"EPA-generate-2021-presentation",
		"EPA-metal-coil-NESHAP-2018",
		"EPA-residential-wood-MacCarty",
	}
	err := filepath.WalkDir(".", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, value := range forbidden {
			if strings.Contains(string(data), value) {
				t.Errorf("%s contains forbidden renderer implementation string %q", path, value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan renderer implementation: %v", err)
	}
}

func writeRealWorldDiffArtifacts(t *testing.T, gotPath string, referencePath string, deckInput string, slideNumber int, result model.CommandResult, diff imageDiff) {
	t.Helper()
	artifactRoot := os.Getenv("PUPPT_REALWORLD_ARTIFACT_DIR")
	if artifactRoot == "" {
		return
	}
	label := strings.TrimSuffix(filepath.Base(deckInput), filepath.Ext(deckInput))
	slideDir := filepath.Join(artifactRoot, label, fmt.Sprintf("slide-%03d", slideNumber))
	if err := os.MkdirAll(slideDir, 0o755); err != nil {
		t.Fatalf("create artifact dir %s: %v", slideDir, err)
	}
	for _, item := range []struct {
		source string
		name   string
	}{
		{source: gotPath, name: "got.png"},
		{source: referencePath, name: "reference.png"},
	} {
		if err := copyFile(item.source, filepath.Join(slideDir, item.name)); err != nil {
			t.Fatalf("write %s artifact for %s slide %d: %v", item.name, deckInput, slideNumber, err)
		}
	}
	if err := writeDiffPNG(gotPath, referencePath, filepath.Join(slideDir, "diff.png")); err != nil {
		t.Fatalf("write diff artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "result.json"), result); err != nil {
		t.Fatalf("write result artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
	if err := writeJSONFile(filepath.Join(slideDir, "diff.json"), realWorldDiffArtifact{
		DeckInput:   deckInput,
		SlideNumber: slideNumber,
		Diff:        diff,
	}); err != nil {
		t.Fatalf("write diff metadata artifact for %s slide %d: %v", deckInput, slideNumber, err)
	}
}

func copyFile(sourcePath string, targetPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

type realWorldDiffArtifact struct {
	DeckInput   string    `json:"deck_input"`
	SlideNumber int       `json:"slide_number"`
	Diff        imageDiff `json:"diff"`
}

func writeJSONFile(targetPath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(targetPath, data, 0o644)
}

func writeDiffPNG(gotPath string, referencePath string, targetPath string) error {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return err
	}
	reference, err := decodePNGFile(referencePath)
	if err != nil {
		return err
	}
	gotBounds := got.Bounds()
	referenceBounds := reference.Bounds()
	width := max(gotBounds.Dx(), referenceBounds.Dx())
	height := max(gotBounds.Dy(), referenceBounds.Dy())
	if width <= 0 || height <= 0 {
		return writePNG(targetPath, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	}
	diff := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			gotInside := x < gotBounds.Dx() && y < gotBounds.Dy()
			referenceInside := x < referenceBounds.Dx() && y < referenceBounds.Dy()
			if !gotInside || !referenceInside {
				diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
				continue
			}
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			rr, rg, rb, ra := reference.At(referenceBounds.Min.X+x, referenceBounds.Min.Y+y).RGBA()
			if gr == rr && gg == rg && gb == rb && ga == ra {
				diff.SetRGBA(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
				continue
			}
			diff.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	return writePNG(targetPath, diff)
}

type referenceManifest struct {
	Decks []struct {
		Input  string `json:"input"`
		Slides []struct {
			File   string `json:"file"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"slides"`
	} `json:"decks"`
}

type imageDiff struct {
	Width                         int              `json:"width"`
	Height                        int              `json:"height"`
	GotWidth                      int              `json:"got_width"`
	GotHeight                     int              `json:"got_height"`
	DifferentPixels               int              `json:"different_pixels"`
	DifferentBounds               *imageDiffBounds `json:"different_bounds,omitempty"`
	MaxChannelDelta8Bit           int              `json:"max_channel_delta_8bit"`
	TotalAbsoluteChannelDelta8Bit int64            `json:"total_absolute_channel_delta_8bit"`
}

type imageDiffBounds struct {
	MinX int `json:"min_x"`
	MinY int `json:"min_y"`
	MaxX int `json:"max_x"`
	MaxY int `json:"max_y"`
}

func referenceDPIForSlide(width int) int {
	if width <= 0 {
		return defaultOutputDPI
	}
	return normalizeOutputDPI(int(math.Round(float64(width) * emuPerInch / defaultSlideCX)))
}

func comparePNG(gotPath string, wantPath string) (imageDiff, error) {
	got, err := decodePNGFile(gotPath)
	if err != nil {
		return imageDiff{}, err
	}
	want, err := decodePNGFile(wantPath)
	if err != nil {
		return imageDiff{}, err
	}
	gotBounds := got.Bounds()
	wantBounds := want.Bounds()
	diff := imageDiff{Width: wantBounds.Dx(), Height: wantBounds.Dy(), GotWidth: gotBounds.Dx(), GotHeight: gotBounds.Dy()}
	if gotBounds.Dx() != wantBounds.Dx() || gotBounds.Dy() != wantBounds.Dy() {
		diff.DifferentPixels = max(gotBounds.Dx(), wantBounds.Dx()) * max(gotBounds.Dy(), wantBounds.Dy())
		if diff.DifferentPixels > 0 {
			diff.DifferentBounds = &imageDiffBounds{MinX: 0, MinY: 0, MaxX: max(gotBounds.Dx(), wantBounds.Dx()) - 1, MaxY: max(gotBounds.Dy(), wantBounds.Dy()) - 1}
		}
		return diff, nil
	}
	for y := 0; y < wantBounds.Dy(); y++ {
		for x := 0; x < wantBounds.Dx(); x++ {
			gr, gg, gb, ga := got.At(gotBounds.Min.X+x, gotBounds.Min.Y+y).RGBA()
			wr, wg, wb, wa := want.At(wantBounds.Min.X+x, wantBounds.Min.Y+y).RGBA()
			if gr != wr || gg != wg || gb != wb || ga != wa {
				diff.DifferentPixels++
				if diff.DifferentBounds == nil {
					diff.DifferentBounds = &imageDiffBounds{MinX: x, MinY: y, MaxX: x, MaxY: y}
				} else {
					if x < diff.DifferentBounds.MinX {
						diff.DifferentBounds.MinX = x
					}
					if y < diff.DifferentBounds.MinY {
						diff.DifferentBounds.MinY = y
					}
					if x > diff.DifferentBounds.MaxX {
						diff.DifferentBounds.MaxX = x
					}
					if y > diff.DifferentBounds.MaxY {
						diff.DifferentBounds.MaxY = y
					}
				}
				diff.addChannelDelta8Bit(gr, wr)
				diff.addChannelDelta8Bit(gg, wg)
				diff.addChannelDelta8Bit(gb, wb)
				diff.addChannelDelta8Bit(ga, wa)
			}
		}
	}
	return diff, nil
}

func (diff *imageDiff) addChannelDelta8Bit(got uint32, want uint32) {
	delta := absInt(rgba16To8Bit(got) - rgba16To8Bit(want))
	if delta > diff.MaxChannelDelta8Bit {
		diff.MaxChannelDelta8Bit = delta
	}
	diff.TotalAbsoluteChannelDelta8Bit += int64(delta)
}

func rgba16To8Bit(value uint32) int {
	return int(value >> 8)
}
