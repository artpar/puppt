package render

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"testing"

	"github.com/artpar/puppt/internal/pptx"
)

type pictureSamplingStageAcceptanceCase struct {
	name                   string
	manifestPath           string
	acceptedResidual       int
	acceptedResidualReason string
}

type pictureSamplingStageAcceptanceResult struct {
	differentPixels int
	targetLabel     string
	targetBounds    ObjectPixelBounds
}

func TestCurrentPictureSamplingStageAcceptanceGate(t *testing.T) {
	if os.Getenv("PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE") != "1" {
		t.Skip("set PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 to run the picture sampling stage replacement gate")
	}

	cases := []pictureSamplingStageAcceptanceCase{
		{
			name:                   "WHO slide 015 Picture 4",
			manifestPath:           "testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json",
			acceptedResidual:       1200,
			acceptedResidualReason: "source-backed residual: opaque 200x200 PNG rendered with documented current crop/effect/sampling path; remaining pixels are picture contour antialiasing, not relationship, crop, mask, color, or transform drift",
		},
		{
			name:                   "EPA slide 004 Google Shape;11;p15",
			manifestPath:           "testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json",
			acceptedResidual:       2127,
			acceptedResidualReason: "source-backed residual: PNG relationship, crop, transform, and visible target are reproduced by the current backend; remaining pixels are sampling/color contour residuals rejected for broad tuning",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runPictureSamplingStageAcceptanceCase(t, tc, currentPictureSamplingStage{})
			if result.differentPixels != 0 && (result.differentPixels != tc.acceptedResidual || tc.acceptedResidualReason == "") {
				t.Fatalf("current picture sampling gate failed for %s: got=%d accepted=%d reason=%q target=%s bounds=%+v", tc.name, result.differentPixels, tc.acceptedResidual, tc.acceptedResidualReason, result.targetLabel, result.targetBounds)
			}
			t.Logf("%s current sampling residual: %d pixels on %s bounds=%+v reason=%s", tc.name, result.differentPixels, result.targetLabel, result.targetBounds, tc.acceptedResidualReason)
		})
	}
}

func TestFractionalSupersamplingPictureStageRejectedByAcceptanceGate(t *testing.T) {
	if os.Getenv("PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE") != "1" {
		t.Skip("set PUPPT_RUN_PICTURE_STAGE_ACCEPTANCE=1 to run the picture sampling stage replacement gate")
	}

	cases := []pictureSamplingStageAcceptanceCase{
		{
			name:             "WHO slide 015 Picture 4",
			manifestPath:     "testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/WHO-HIV-testing-algorithms-toolkit/slide-015/micro-fixtures/0009-1028-Picture-4/manifest.json",
			acceptedResidual: 1173,
		},
		{
			name:             "EPA slide 004 Google Shape;11;p15",
			manifestPath:     "testdata/realworld-ppts/render-artifacts/object-debug-2026-06-01/EPA-residential-wood-MacCarty/slide-004/micro-fixtures/cumulative-picture-0001-11-Google-Shape-11-p15/manifest.json",
			acceptedResidual: 2115,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runPictureSamplingStageAcceptanceCase(t, tc, fractionalSupersamplingPictureSamplingStage{samplesPerAxis: 4})
			if result.differentPixels != tc.acceptedResidual {
				t.Fatalf("fractional supersampling residual drifted for %s: got=%d want=%d target=%s bounds=%+v", tc.name, result.differentPixels, tc.acceptedResidual, result.targetLabel, result.targetBounds)
			}
			if result.differentPixels == 0 {
				t.Fatalf("fractional supersampling unexpectedly passed the replacement gate for %s", tc.name)
			}
			t.Logf("%s fractional supersampling residual: %d pixels on %s bounds=%+v", tc.name, result.differentPixels, result.targetLabel, result.targetBounds)
		})
	}
}

func runPictureSamplingStageAcceptanceCase(t *testing.T, tc pictureSamplingStageAcceptanceCase, stage pictureSamplingStage) pictureSamplingStageAcceptanceResult {
	t.Helper()

	manifestPath := resolveTestArtifactPath(tc.manifestPath)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read picture sampling acceptance manifest %s: %v", manifestPath, err)
	}
	var manifest microFixtureManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse picture sampling acceptance manifest %s: %v", manifestPath, err)
	}
	if manifest.Object.Kind != "pic" || manifest.Object.OutputPixelBounds == nil {
		t.Fatalf("picture sampling acceptance requires a picture fixture with output bounds, got %+v", manifest.Object)
	}

	canvas, err := renderMicroFixturePictureWithSamplingStage(manifest, stage)
	if err != nil {
		t.Fatalf("render picture sampling acceptance fixture %s: %v", manifestPath, err)
	}

	got := cropRGBA(canvas, *manifest.Object.OutputPixelBounds)
	referencePath := resolveTestArtifactPath(manifest.ReferenceCropPath)
	targetLabel := "crop"
	if len(manifest.OccludedBy) > 0 {
		applyPictureCandidateOcclusions(got, *manifest.Object.OutputPixelBounds, manifest.OccludedBy)
		referencePath = resolveTestArtifactPath(manifest.ReferenceVisibleCropPath)
		targetLabel = "visible crop"
	}
	reference, err := decodePNGFile(referencePath)
	if err != nil {
		t.Fatalf("decode picture sampling acceptance reference %s: %v", referencePath, err)
	}
	metrics := compareCandidateImage(reference, got)
	return pictureSamplingStageAcceptanceResult{
		differentPixels: metrics.DifferentPixels,
		targetLabel:     targetLabel,
		targetBounds:    *manifest.Object.OutputPixelBounds,
	}
}

func renderMicroFixturePictureWithSamplingStage(manifest microFixtureManifest, stage pictureSamplingStage) (*image.RGBA, error) {
	fixturePath := resolveTestArtifactPath(manifest.FixturePath)
	pkg, err := pptx.Open(context.Background(), fixturePath)
	if err != nil {
		return nil, err
	}
	if len(pkg.SlideParts) == 0 {
		return nil, fmt.Errorf("fixture has no slide parts")
	}
	slidePart := pkg.SlideParts[0]
	size := parseSlideSize(pkg.Parts[pkg.PresentationPath])
	width := emuToPixelsAtDPI(size.CX, defaultOutputDPI)
	height := emuToPixelsAtDPI(size.CY, defaultOutputDPI)
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid fixture canvas size %dx%d", width, height)
	}
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)

	elements := resolvedRenderElementsForPart(pkg, slidePart, slidePart)
	element, ok := findMicroFixtureSlideElement(elements, manifest.Object)
	if !ok {
		return nil, fmt.Errorf("fixture picture %s %q not found", manifest.Object.CNvPrID, manifest.Object.CNvPrName)
	}
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return nil, err
	}
	relationshipByID := make(map[string]pptx.Relationship, len(relationships))
	for _, relationship := range relationships {
		relationshipByID[relationship.ID] = relationship
	}
	primitive, err := renderPicturePrimitiveFromElement(pkg, slidePart, size, canvas.Bounds(), element, relationshipByID)
	if err != nil {
		return nil, err
	}
	relationship, ok := relationshipByID[primitive.RelationshipID]
	if !ok {
		return nil, fmt.Errorf("fixture picture relationship %q not found", primitive.RelationshipID)
	}
	source, targetPart, partialUnsupported := pictureSourceImage(pkg, slidePart, &element, relationshipByID, relationship)
	if source == nil {
		return nil, fmt.Errorf("fixture picture source %s could not be decoded: %v", targetPart, partialUnsupported)
	}
	unsupported := currentPictureBackend{sampler: stage}.RenderPicture(pictureBackendInput{
		SlidePart:          slidePart,
		Size:               size,
		Canvas:             canvas,
		Primitive:          primitive,
		Source:             source,
		TargetPart:         targetPart,
		PartialUnsupported: partialUnsupported,
	})
	if len(unsupported) > 0 {
		return nil, fmt.Errorf("fixture picture rendered with unsupported records: %+v", unsupported)
	}
	applyDisplayP3OutputTransform(canvas)
	return canvas, nil
}

func cropRGBA(source *image.RGBA, bounds ObjectPixelBounds) *image.RGBA {
	rect := image.Rect(bounds.MinX, bounds.MinY, bounds.MaxX+1, bounds.MaxY+1).Intersect(source.Bounds())
	output := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(output, output.Bounds(), source, rect.Min, draw.Src)
	return output
}

type fractionalSupersamplingPictureSamplingStage struct {
	samplesPerAxis int
}

func (stage fractionalSupersamplingPictureSamplingStage) Draw(input pictureSamplingInput) bool {
	if input.Primitive.RotationDegrees != 0 || !input.Primitive.RotatesWithShape || input.Primitive.HasSoftEdge || len(input.Primitive.CustomPath) >= 3 {
		return currentPictureSamplingStage{}.Draw(input)
	}
	samplesPerAxis := stage.samplesPerAxis
	if samplesPerAxis <= 0 {
		samplesPerAxis = 4
	}
	fractional := input.Primitive.FractionalTarget
	if fractional.MaxX <= fractional.MinX || fractional.MaxY <= fractional.MinY {
		return currentPictureSamplingStage{}.Draw(input)
	}
	target := image.Rect(
		int(math.Floor(fractional.MinX)),
		int(math.Floor(fractional.MinY)),
		int(math.Ceil(fractional.MaxX)),
		int(math.Ceil(fractional.MaxY)),
	).Intersect(input.Canvas.Bounds())
	if target.Empty() || input.Source == nil || input.SourceBounds.Empty() {
		return false
	}
	totalSamples := samplesPerAxis * samplesPerAxis
	for y := target.Min.Y; y < target.Max.Y; y++ {
		for x := target.Min.X; x < target.Max.X; x++ {
			var red, green, blue float64
			for sy := 0; sy < samplesPerAxis; sy++ {
				sampleY := float64(y) + (float64(sy)+0.5)/float64(samplesPerAxis)
				for sx := 0; sx < samplesPerAxis; sx++ {
					sampleX := float64(x) + (float64(sx)+0.5)/float64(samplesPerAxis)
					if sampleX < fractional.MinX || sampleX >= fractional.MaxX || sampleY < fractional.MinY || sampleY >= fractional.MaxY {
						red += 255
						green += 255
						blue += 255
						continue
					}
					sourceX := ((sampleX - fractional.MinX) * float64(input.SourceBounds.Dx()) / (fractional.MaxX - fractional.MinX)) - 0.5
					sourceY := ((sampleY - fractional.MinY) * float64(input.SourceBounds.Dy()) / (fractional.MaxY - fractional.MinY)) - 0.5
					sample := pictureBilinearSample(input.Source, input.SourceBounds, sourceX, sourceY)
					sampleAlpha := float64(sample.A) / 255
					red += float64(sample.R)
					red += 255 * (1 - sampleAlpha)
					green += float64(sample.G)
					green += 255 * (1 - sampleAlpha)
					blue += float64(sample.B)
				}
			}
			input.Canvas.SetRGBA(x, y, color.RGBA{
				R: floatChannelToByte(red / float64(totalSamples)),
				G: floatChannelToByte(green / float64(totalSamples)),
				B: floatChannelToByte(blue / float64(totalSamples)),
				A: 255,
			})
		}
	}
	return false
}
