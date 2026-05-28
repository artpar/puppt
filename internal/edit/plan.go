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

const statusUnsupported = "unsupported"

// Plan reads an edit spec and returns the non-mutating edit plan.
func Plan(ctx context.Context, deckPath string, specPath string) (*model.CommandResult, error) {
	spec, err := readSpec(specPath)
	if err != nil {
		return nil, err
	}
	if message := validateOperationTarget(spec); message != "" {
		plan := &model.EditPlan{
			Operation: spec.Operation,
			Target:    spec.Target,
			Matches:   []model.TargetMatch{},
			Status:    statusUnsupported,
			Message:   message,
		}
		return &model.CommandResult{
			SchemaVersion: model.SchemaVersion,
			Command:       "plan",
			Status:        statusUnsupported,
			Input:         deckPath,
			Output:        nil,
			Warnings:      []model.Warning{},
			Errors:        []model.ErrorItem{},
			Summary:       model.Summary{Human: "Edit operation is unsupported for the requested target."},
			Plan:          plan,
			Unsupported: []model.SkipItem{{
				Code:    "unsupported_operation_target",
				Message: message,
			}},
		}, nil
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

func validateOperationTarget(spec model.EditSpec) string {
	allowedTargets := map[string]map[string]bool{
		"replace_text": {
			"title":        true,
			"visible_text": true,
			"object_id":    true,
		},
		"update_notes": {
			"notes": true,
		},
		"update_metadata": {
			"metadata": true,
		},
		"replace_image": {
			"object_id": true,
		},
		"slide_add": {
			"slide_number": true,
		},
		"slide_delete": {
			"slide_number": true,
		},
		"slide_move": {
			"slide_number": true,
		},
		"slide_duplicate": {
			"slide_number": true,
		},
	}

	targets, ok := allowedTargets[spec.Operation]
	if !ok {
		return fmt.Sprintf("unsupported operation %q", spec.Operation)
	}
	if !targets[spec.Target.Type] {
		return fmt.Sprintf("operation %q does not support target type %q", spec.Operation, spec.Target.Type)
	}
	if spec.Operation == "replace_text" && spec.Replacement == "" {
		return "replace_text requires replacement"
	}
	if spec.Operation == "update_notes" && spec.Replacement == "" {
		return "update_notes requires replacement"
	}
	if spec.Operation == "update_metadata" && spec.Target.Property == "" {
		return "update_metadata requires target property"
	}
	return ""
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
	case statusUnsupported:
		return "Edit operation is unsupported for the requested target."
	default:
		return plan.Message
	}
}
