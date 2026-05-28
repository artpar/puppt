package edit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/target"
)

// Plan reads an edit spec and returns the non-mutating edit plan.
func Plan(ctx context.Context, deckPath string, specPath string) (*model.CommandResult, error) {
	spec, err := readSpec(specPath)
	if err != nil {
		return nil, err
	}

	inspectionResult, err := inspect.Inspect(ctx, deckPath)
	if err != nil {
		return nil, err
	}

	matches, status, message := target.Resolve(inspectionResult.Inspection, spec.Target)
	plan := &model.EditPlan{
		Operation: spec.Operation,
		Target:    spec.Target,
		Matches:   matches,
		Status:    status,
		Message:   message,
	}

	resultStatus := "ok"
	if status != target.StatusReady {
		resultStatus = status
	}

	result := &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "plan",
		Status:        resultStatus,
		Input:         deckPath,
		Output:        nil,
		Warnings:      inspectionResult.Warnings,
		Errors:        []model.ErrorItem{},
		Summary: model.Summary{
			Human: humanSummary(plan),
		},
		Plan: plan,
	}

	switch status {
	case target.StatusNoMatch:
		result.Skipped = []model.SkipItem{{
			Code:    "no_match",
			Message: message,
		}}
	case target.StatusAmbiguous:
		result.Ambiguous = []model.SkipItem{{
			Code:    "ambiguous_target",
			Message: message,
		}}
	}
	return result, nil
}

func readSpec(specPath string) (model.EditSpec, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return model.EditSpec{}, err
	}

	var spec model.EditSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		return model.EditSpec{}, err
	}
	if spec.Operation == "" {
		return model.EditSpec{}, fmt.Errorf("edit spec missing operation")
	}
	if spec.Target.Type == "" {
		return model.EditSpec{}, fmt.Errorf("edit spec missing target type")
	}
	return spec, nil
}

func humanSummary(plan *model.EditPlan) string {
	switch plan.Status {
	case target.StatusReady:
		return fmt.Sprintf("Planned %s for %d target(s).", plan.Operation, len(plan.Matches))
	case target.StatusAmbiguous:
		return "Edit target is ambiguous."
	case target.StatusNoMatch:
		return "No matching edit target found."
	default:
		return plan.Message
	}
}
