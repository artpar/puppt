package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

// Validate checks core package structure and relationship reachability.
func Validate(ctx context.Context, filePath string) (*model.CommandResult, error) {
	pkg, err := pptx.Open(ctx, filePath)
	if err != nil {
		return nil, err
	}

	warnings := []model.Warning{}
	errors := []model.ErrorItem{}

	for _, slidePart := range pkg.SlideParts {
		relationships, err := pkg.RelationshipsForPart(slidePart)
		if err != nil {
			errors = append(errors, model.ErrorItem{
				Code:    "invalid_relationships",
				Message: err.Error(),
				Part:    pptx.RelationshipsPartFor(slidePart),
			})
			continue
		}
		for _, relationship := range relationships {
			if relationship.TargetMode != "" && !strings.EqualFold(relationship.TargetMode, "Internal") {
				continue
			}
			switch relationship.Type {
			case pptx.NotesSlideRelType, pptx.ImageRelType, pptx.AudioRelType, pptx.VideoRelType, pptx.OLEObjectRelType, pptx.SlideLayoutRelType:
				targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
				if _, ok := pkg.Parts[targetPart]; !ok {
					errors = append(errors, model.ErrorItem{
						Code:    "missing_relationship_target",
						Message: fmt.Sprintf("relationship %s target is missing", relationship.ID),
						Part:    targetPart,
					})
				}
			}
		}
	}

	status := "ok"
	valid := len(errors) == 0
	if !valid {
		status = "invalid"
	}
	return &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "validate",
		Status:        status,
		Input:         filePath,
		Output:        nil,
		Warnings:      warnings,
		Errors:        errors,
		Summary: model.Summary{
			Human: validationSummary(valid, len(errors)),
		},
		Validation: &model.Validation{
			Valid:    valid,
			Warnings: warnings,
			Errors:   errors,
		},
	}, nil
}

func validationSummary(valid bool, errorCount int) string {
	if valid {
		return "Validation passed."
	}
	return fmt.Sprintf("Validation failed with %d error(s).", errorCount)
}
