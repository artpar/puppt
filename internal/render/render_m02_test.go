package render

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestMicroFixtureCoverageQueueSummaryReadsGeneratedMetadata(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "renderer-coverage-summary.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read coverage queue summary %s: %v", path, err)
	}
	var summary struct {
		TotalDeclarations int                       `json:"total_declarations"`
		Queues            map[string]int            `json:"queues"`
		QueueStatusCounts map[string]map[string]int `json:"queue_status_counts"`
		Rows              []struct {
			Anchor      string `json:"anchor"`
			Declaration string `json:"declaration"`
			Status      string `json:"status"`
			Queue       string `json:"queue"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatalf("parse coverage queue summary: %v", err)
	}
	if summary.TotalDeclarations != 1007 || len(summary.Rows) != summary.TotalDeclarations {
		t.Fatalf("unexpected coverage inventory size: total=%d rows=%d", summary.TotalDeclarations, len(summary.Rows))
	}
	for _, queue := range []string{"core-static", "common-partial", "hard-rendering", "unsupported-preserve", "out-of-scope"} {
		if summary.Queues[queue] == 0 {
			t.Fatalf("expected queue %q to have declarations, got queues=%+v", queue, summary.Queues)
		}
		if summary.QueueStatusCounts[queue] == nil {
			t.Fatalf("missing queue/status counts for %q", queue)
		}
	}
	var foundPicture bool
	for _, row := range summary.Rows {
		if row.Anchor == "pml.xsd:1245" && row.Declaration == "CT_Picture" && row.Status == "Partial" && row.Queue == "common-partial" {
			foundPicture = true
			break
		}
	}
	if !foundPicture {
		t.Fatalf("expected pml.xsd:1245 CT_Picture to be tracked as common-partial")
	}
}

func TestMicroFixtureSpecFixtureManifestFormatIncludesSchemaAnchors(t *testing.T) {
	object := objectFailureRecord{
		SourcePart: "ppt/slides/slide15.xml",
		XMLPath:    `/p:sld/p:cSld/p:spTree/p:pic[.//p:cNvPr/@id="1028"]`,
		Kind:       "pic",
		CNvPrID:    "1028",
		CNvPrName:  "Picture 4",
	}
	spec := specFixtureForObject(object)
	if len(spec.SchemaAnchors) < 3 || spec.SourceXMLPart != object.SourcePart || spec.SourceXMLPath != object.XMLPath {
		t.Fatalf("unexpected picture spec fixture manifest: %+v", spec)
	}
	if !strings.Contains(spec.ExpectedSemanticModel, "resolved blip relationship") || !strings.Contains(spec.ExpectedRenderPrimitive, "picture/media primitive") {
		t.Fatalf("spec fixture should describe expected semantics and primitive, got %+v", spec)
	}
}

func TestMicroFixtureManifestsDoNotClassifySupportedImageMetadataAsUnsupported(t *testing.T) {
	root := resolveTestArtifactPath(filepath.Join("testdata", "realworld-ppts", "render-artifacts", "object-debug-2026-06-01"))
	var offenders []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil || info.IsDir() || filepath.Base(path) != "manifest.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var manifest microFixtureManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, record := range manifest.SpecFixture.ExpectedUnsupportedRecord {
			if supportedImageMetadataExpectedUnsupported(record) {
				offenders = append(offenders, fmt.Sprintf("%s: %s", path, record))
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(offenders) > 0 {
		t.Fatalf("supported image/effect metadata must not be expected unsupported records: %s", strings.Join(offenders, "\n"))
	}
}

func supportedImageMetadataExpectedUnsupported(record string) bool {
	if record == "alphaCeiling" || record == "alphaFloor" || record == "alphaInv" || record == "clrChange" || record == "clrRepl" || record == "duotone" || record == "grayscl" {
		return true
	}
	for _, prefix := range []string{
		"alphaBiLevel=",
		"alphaModFix=",
		"alphaRepl=",
		"biLevel=",
		"blur=",
		"fillMode=",
		"fillOverlay=",
		"glow=",
		"innerShdw=",
		"lum bright=",
		"reflection=",
		"rotWithShape=",
		"softEdge=",
	} {
		if strings.HasPrefix(record, prefix) {
			return true
		}
	}
	return false
}

func TestMicroFixtureCleanFailureSuite(t *testing.T) {
	root := os.Getenv("PUPPT_MICRO_FIXTURE_ROOT")
	if root == "" {
		t.Skip("set PUPPT_MICRO_FIXTURE_ROOT to run every tracked clean object micro-fixture")
	}
	root = resolveTestArtifactPath(root)
	ownership, err := summarizeMicroFixtureTargetOwnership(root)
	if err != nil {
		t.Fatalf("summarize clean fixture ownership: %v", err)
	}
	summary := microFixtureSuiteSummary{
		Root:  root,
		Basis: "rerenders every clean object fixture and compares the fixture acceptance crop; exact diff remains diagnostic and perceptual metrics are validation/triage only",
	}
	outputRoot := filepath.Join(t.TempDir(), "clean-fixture-suite")
	for _, ownershipRecord := range ownership.CleanFailures {
		record, err := runMicroFixtureSuiteManifest(resolveTestArtifactPath(ownershipRecord.ManifestPath), outputRoot)
		if err != nil {
			t.Fatalf("run clean micro-fixture %s: %v", ownershipRecord.ManifestPath, err)
		}
		summary.Records = append(summary.Records, record)
		if record.Diff.DifferentPixels == 0 {
			summary.Passed++
		} else {
			summary.Failed++
		}
	}
	summary.Total = len(summary.Records)
	sort.Slice(summary.Records, func(i, j int) bool {
		if summary.Records[i].Diff.DifferentPixels != summary.Records[j].Diff.DifferentPixels {
			return summary.Records[i].Diff.DifferentPixels > summary.Records[j].Diff.DifferentPixels
		}
		return summary.Records[i].ManifestPath < summary.Records[j].ManifestPath
	})
	if outputPath := os.Getenv("PUPPT_CLEAN_MICRO_FIXTURE_SUITE_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, summary); err != nil {
			t.Fatalf("write clean micro-fixture suite summary: %v", err)
		}
	}
	t.Logf("clean micro-fixture suite: total=%d passed=%d failed=%d", summary.Total, summary.Passed, summary.Failed)
	if summary.Failed > 0 && os.Getenv("PUPPT_ACCEPT_CLEAN_FIXTURE_FAILURES") != "1" {
		t.Fatalf("%d clean micro-fixture(s) still fail; first: %s", summary.Failed, summary.Records[0].FailureSummary())
	}
}

func TestRealWorldPerceptualMetrics(t *testing.T) {
	if os.Getenv("PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS") != "1" {
		t.Skip("set PUPPT_RUN_REALWORLD_PERCEPTUAL_METRICS=1 to render the real-world corpus and write perceptual metrics")
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
	summary := realWorldPerceptualSummary{
		ReferenceRoot: referenceRoot,
		Basis:         "deterministic luma and RGB-RMS image similarity over rendered slide PNGs; validation/triage only",
	}
	for _, deck := range manifest.Decks {
		for index, slide := range deck.Slides {
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
			record := realWorldPerceptualRecord{
				DeckInput:       deck.Input,
				SlideNumber:     index + 1,
				ReferencePath:   referencePath,
				DifferentPixels: diff.DifferentPixels,
				DiffBounds:      diff.DifferentBounds,
				Unsupported:     len(result.Unsupported),
				Status:          result.Status,
				Perceptual:      diff.Perceptual,
			}
			summary.Slides = append(summary.Slides, record)
			summary.TotalSlides++
			summary.TotalDifferentPixels += int64(diff.DifferentPixels)
			if diff.DifferentPixels > 0 {
				summary.DifferentSlides++
			}
			summary.LumaSimilaritySum += diff.Perceptual.LumaSimilarity
			summary.ChannelRMSSimilaritySum += diff.Perceptual.ChannelRMSSimilarity
		}
	}
	if summary.TotalSlides > 0 {
		summary.MeanLumaSimilarity = roundFloat(summary.LumaSimilaritySum/float64(summary.TotalSlides), 9)
		summary.MeanChannelRMSSimilarity = roundFloat(summary.ChannelRMSSimilaritySum/float64(summary.TotalSlides), 9)
	}
	sort.Slice(summary.Slides, func(i, j int) bool {
		if summary.Slides[i].Perceptual.LumaSimilarity != summary.Slides[j].Perceptual.LumaSimilarity {
			return summary.Slides[i].Perceptual.LumaSimilarity < summary.Slides[j].Perceptual.LumaSimilarity
		}
		if summary.Slides[i].DifferentPixels != summary.Slides[j].DifferentPixels {
			return summary.Slides[i].DifferentPixels > summary.Slides[j].DifferentPixels
		}
		return summary.Slides[i].ReferencePath < summary.Slides[j].ReferencePath
	})
	if outputPath := os.Getenv("PUPPT_REALWORLD_PERCEPTUAL_OUTPUT"); outputPath != "" {
		if err := writeJSONOutputFile(outputPath, summary); err != nil {
			t.Fatalf("write real-world perceptual metrics: %v", err)
		}
	}
	t.Logf("real-world perceptual metrics: slides=%d different=%d mean_luma_similarity=%.9f mean_channel_rms_similarity=%.9f total_diff=%d", summary.TotalSlides, summary.DifferentSlides, summary.MeanLumaSimilarity, summary.MeanChannelRMSSimilarity, summary.TotalDifferentPixels)
}

type realWorldPerceptualSummary struct {
	ReferenceRoot            string                      `json:"reference_root"`
	Basis                    string                      `json:"basis"`
	TotalSlides              int                         `json:"total_slides"`
	DifferentSlides          int                         `json:"different_slides"`
	TotalDifferentPixels     int64                       `json:"total_different_pixels"`
	MeanLumaSimilarity       float64                     `json:"mean_luma_similarity"`
	MeanChannelRMSSimilarity float64                     `json:"mean_channel_rms_similarity"`
	LumaSimilaritySum        float64                     `json:"-"`
	ChannelRMSSimilaritySum  float64                     `json:"-"`
	Slides                   []realWorldPerceptualRecord `json:"slides"`
}

type realWorldPerceptualRecord struct {
	DeckInput       string           `json:"deck_input"`
	SlideNumber     int              `json:"slide_number"`
	ReferencePath   string           `json:"reference_path"`
	DifferentPixels int              `json:"different_pixels"`
	DiffBounds      *imageDiffBounds `json:"different_bounds,omitempty"`
	Unsupported     int              `json:"unsupported"`
	Status          string           `json:"status"`
	Perceptual      perceptualMetric `json:"perceptual"`
}

type microFixtureSuiteSummary struct {
	Root    string                    `json:"root"`
	Basis   string                    `json:"basis"`
	Total   int                       `json:"total"`
	Passed  int                       `json:"passed"`
	Failed  int                       `json:"failed"`
	Records []microFixtureSuiteRecord `json:"records,omitempty"`
}

type microFixtureSuiteRecord struct {
	ManifestPath   string    `json:"manifest_path"`
	DeckInput      string    `json:"deck_input"`
	SlideNumber    int       `json:"slide_number"`
	Kind           string    `json:"kind"`
	CNvPrID        string    `json:"cnv_pr_id,omitempty"`
	CNvPrName      string    `json:"cnv_pr_name,omitempty"`
	SchemaAnchors  []string  `json:"schema_anchors,omitempty"`
	SourceXMLPart  string    `json:"source_xml_part,omitempty"`
	SourceXMLPath  string    `json:"source_xml_path,omitempty"`
	TargetCompared string    `json:"target_compared"`
	GotPath        string    `json:"got_path"`
	ReferencePath  string    `json:"reference_path"`
	Diff           imageDiff `json:"diff"`
	Acceptance     string    `json:"acceptance"`
}

func (record microFixtureSuiteRecord) FailureSummary() string {
	return fmt.Sprintf("%s slide %d object %s %q schema=%s source=%s %s diff=%d bounds=%+v got=%s reference=%s",
		record.DeckInput,
		record.SlideNumber,
		record.CNvPrID,
		record.CNvPrName,
		strings.Join(record.SchemaAnchors, ", "),
		record.SourceXMLPart,
		record.SourceXMLPath,
		record.Diff.DifferentPixels,
		record.Diff.DifferentBounds,
		record.GotPath,
		record.ReferencePath,
	)
}

func runMicroFixtureSuiteManifest(manifestPath string, outputRoot string) (microFixtureSuiteRecord, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return microFixtureSuiteRecord{}, err
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return microFixtureSuiteRecord{}, err
	}
	label := sanitizeObjectArtifactName(fmt.Sprintf("%s-slide-%03d-%s-%s-%s", filepath.Base(manifest.DeckInput), manifest.SlideNumber, manifest.Object.Kind, manifest.Object.CNvPrID, manifest.Object.CNvPrName))
	outputDir := filepath.Join(outputRoot, label)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return microFixtureSuiteRecord{}, err
	}
	outputPath := filepath.Join(outputDir, "got.png")
	if _, err := Render(context.Background(), resolveTestArtifactPath(manifest.FixturePath), Options{SlideNumber: 1, OutputPath: outputPath}); err != nil {
		return microFixtureSuiteRecord{}, err
	}
	gotTargetPath := filepath.Join(outputDir, "got-crop.png")
	referenceTargetPath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetCompared := "got-crop.png vs reference-crop.png"
	if len(manifest.OccludedBy) > 0 {
		gotTargetPath = filepath.Join(outputDir, "got-visible-crop.png")
		referenceTargetPath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetCompared = "got-visible-crop.png vs reference-visible-crop.png"
		if manifest.Object.OutputPixelBounds == nil {
			return microFixtureSuiteRecord{}, fmt.Errorf("manifest has occlusions but no output pixel bounds: %s", manifestPath)
		}
		if err := writeVisibleCroppedPNG(outputPath, gotTargetPath, *manifest.Object.OutputPixelBounds, manifest.OccludedBy); err != nil {
			return microFixtureSuiteRecord{}, err
		}
	} else {
		if manifest.Object.OutputPixelBounds == nil {
			return microFixtureSuiteRecord{}, fmt.Errorf("manifest has no output pixel bounds: %s", manifestPath)
		}
		if err := writeCroppedPNG(outputPath, gotTargetPath, *manifest.Object.OutputPixelBounds); err != nil {
			return microFixtureSuiteRecord{}, err
		}
	}
	diff, err := comparePNG(gotTargetPath, referenceTargetPath)
	if err != nil {
		return microFixtureSuiteRecord{}, err
	}
	return microFixtureSuiteRecord{
		ManifestPath:   manifestPath,
		DeckInput:      manifest.DeckInput,
		SlideNumber:    manifest.SlideNumber,
		Kind:           manifest.Object.Kind,
		CNvPrID:        manifest.Object.CNvPrID,
		CNvPrName:      manifest.Object.CNvPrName,
		SchemaAnchors:  microFixtureManifestSchemaAnchors(manifest),
		SourceXMLPart:  microFixtureManifestSourcePart(manifest),
		SourceXMLPath:  microFixtureManifestSourcePath(manifest),
		TargetCompared: targetCompared,
		GotPath:        gotTargetPath,
		ReferencePath:  referenceTargetPath,
		Diff:           diff,
		Acceptance:     manifest.Acceptance,
	}, nil
}
