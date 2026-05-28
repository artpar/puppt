package target

import (
	"testing"

	"github.com/artpar/puppt/internal/model"
)

func TestResolveDetectsAmbiguousVisibleText(t *testing.T) {
	inspection := &model.Inspection{
		Slides: []model.Slide{
			{Number: 1, ID: "s1", VisibleText: []model.TextBlock{{ObjectID: "s1#shape-1", Text: "Repeat"}}},
			{Number: 2, ID: "s2", VisibleText: []model.TextBlock{{ObjectID: "s2#shape-1", Text: "Repeat"}}},
		},
	}

	matches, status, _ := Resolve(inspection, model.TargetSpec{Type: "visible_text", Text: "Repeat"})
	if status != StatusAmbiguous {
		t.Fatalf("unexpected status: %s", status)
	}
	if len(matches) != 2 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}

func TestResolveAllowsDeckScopeVisibleText(t *testing.T) {
	inspection := &model.Inspection{
		Slides: []model.Slide{
			{Number: 1, ID: "s1", VisibleText: []model.TextBlock{{ObjectID: "s1#shape-1", Text: "Repeat"}}},
			{Number: 2, ID: "s2", VisibleText: []model.TextBlock{{ObjectID: "s2#shape-1", Text: "Repeat"}}},
		},
	}

	matches, status, _ := Resolve(inspection, model.TargetSpec{Type: "visible_text", Scope: "deck", Text: "Repeat"})
	if status != StatusReady {
		t.Fatalf("unexpected status: %s", status)
	}
	if len(matches) != 2 {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}

func TestResolveObjectID(t *testing.T) {
	inspection := &model.Inspection{
		Slides: []model.Slide{
			{Number: 1, ID: "s1", VisibleText: []model.TextBlock{{ObjectID: "s1#shape-1", Text: "Title"}}},
		},
	}

	matches, status, _ := Resolve(inspection, model.TargetSpec{Type: "object_id", ObjectID: "s1#shape-1"})
	if status != StatusReady {
		t.Fatalf("unexpected status: %s", status)
	}
	if len(matches) != 1 || matches[0].ObjectID != "s1#shape-1" {
		t.Fatalf("unexpected matches: %+v", matches)
	}
}
