package review

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/validate"
)

// Review inspects a deck and summarizes an existing changes JSON artifact.
func Review(ctx context.Context, deckPath string, changesPath string) (*model.CommandResult, error) {
	artifact, err := readReviewArtifact(changesPath)
	if err != nil {
		return nil, err
	}
	inspectionResult, err := inspect.Inspect(ctx, deckPath)
	if err != nil {
		return nil, err
	}
	validationResult, err := validate.Validate(ctx, deckPath)
	if err != nil {
		return nil, err
	}
	validation := validationResult.Validation
	if validation == nil {
		validation = &model.Validation{Valid: false, Warnings: []model.Warning{}, Errors: []model.ErrorItem{{Code: "missing_validation", Message: "validation result missing"}}}
	}
	slideCount := 0
	if inspectionResult.Inspection != nil {
		slideCount = inspectionResult.Inspection.SlideCount
	}
	statusText := "failed"
	if validation.Valid {
		statusText = "passed"
	}
	humanSummary := fmt.Sprintf(
		"Reviewed %d slide deck with %d reported change(s) on %s; skipped %d, ambiguous %d, unsupported %d; validation %s.",
		slideCount,
		len(artifact.Changes),
		touchedSlides(artifact.Changes),
		len(artifact.Skipped),
		len(artifact.Ambiguous),
		len(artifact.Unsupported),
		statusText,
	)
	return &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "review",
		Status:        "ok",
		Input:         deckPath,
		Output:        nil,
		Warnings:      append(inspectionResult.Warnings, validation.Warnings...),
		Errors:        validation.Errors,
		Summary: model.Summary{
			Human: humanSummary,
		},
		Inspection:  inspectionResult.Inspection,
		Changes:     artifact.Changes,
		Skipped:     artifact.Skipped,
		Ambiguous:   artifact.Ambiguous,
		Unsupported: artifact.Unsupported,
		Validation:  validation,
	}, nil
}

func touchedSlides(changes []model.ChangeItem) string {
	seen := map[int]bool{}
	var slides []int
	for _, change := range changes {
		if change.SlideNumber == 0 || seen[change.SlideNumber] {
			continue
		}
		seen[change.SlideNumber] = true
		slides = append(slides, change.SlideNumber)
	}
	if len(slides) == 0 {
		return "no named slides"
	}
	items := make([]string, 0, len(slides))
	for _, slide := range slides {
		items = append(items, fmt.Sprintf("slide %d", slide))
	}
	return strings.Join(items, ", ")
}

type reviewArtifact struct {
	Changes     []model.ChangeItem
	Skipped     []model.SkipItem
	Ambiguous   []model.SkipItem
	Unsupported []model.SkipItem
}

func readReviewArtifact(changesPath string) (reviewArtifact, error) {
	data, err := os.ReadFile(changesPath)
	if err != nil {
		return reviewArtifact{}, err
	}
	var result model.CommandResult
	if err := json.Unmarshal(data, &result); err == nil && result.SchemaVersion == model.SchemaVersion {
		return reviewArtifact{
			Changes:     emptyChanges(result.Changes),
			Skipped:     emptySkips(result.Skipped),
			Ambiguous:   emptySkips(result.Ambiguous),
			Unsupported: emptySkips(result.Unsupported),
		}, nil
	}
	var changes []model.ChangeItem
	if err := json.Unmarshal(data, &changes); err != nil {
		return reviewArtifact{}, err
	}
	return reviewArtifact{Changes: emptyChanges(changes), Skipped: []model.SkipItem{}, Ambiguous: []model.SkipItem{}, Unsupported: []model.SkipItem{}}, nil
}

func emptyChanges(changes []model.ChangeItem) []model.ChangeItem {
	if changes == nil {
		return []model.ChangeItem{}
	}
	return changes
}

func emptySkips(items []model.SkipItem) []model.SkipItem {
	if items == nil {
		return []model.SkipItem{}
	}
	return items
}
