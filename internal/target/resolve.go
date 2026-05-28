package target

import (
	"fmt"
	"strings"

	"github.com/artpar/puppt/internal/model"
)

const (
	StatusReady     = "ready"
	StatusNoMatch   = "no_match"
	StatusAmbiguous = "ambiguous"
)

// Resolve maps an edit target onto inspected deck facts without mutating the deck.
func Resolve(inspection *model.Inspection, spec model.TargetSpec) ([]model.TargetMatch, string, string) {
	if inspection == nil {
		return nil, StatusNoMatch, "inspection is missing"
	}

	var matches []model.TargetMatch
	switch spec.Type {
	case "slide_number":
		matches = matchSlideNumber(inspection, spec.SlideNumber)
	case "title":
		matches = matchTitle(inspection, spec.Text)
	case "visible_text":
		matches = matchVisibleText(inspection, spec.Text)
	case "object_id":
		matches = matchObjectID(inspection, spec.ObjectID)
	case "image":
		matches = matchImages(inspection, spec.SlideNumber)
	case "notes":
		matches = matchNotes(inspection, spec.SlideNumber)
	case "metadata":
		if spec.Property != "" {
			matches = []model.TargetMatch{{Kind: "metadata", Property: spec.Property}}
		}
	default:
		return nil, StatusNoMatch, fmt.Sprintf("unsupported target type %q", spec.Type)
	}

	if len(matches) == 0 {
		return matches, StatusNoMatch, "no matching target found"
	}
	if spec.Scope == "deck" {
		return matches, StatusReady, fmt.Sprintf("matched %d targets", len(matches))
	}
	if len(matches) > 1 {
		return matches, StatusAmbiguous, fmt.Sprintf("matched %d targets; specify a narrower target or deck scope", len(matches))
	}
	return matches, StatusReady, "matched 1 target"
}

func matchImages(inspection *model.Inspection, slideNumber int) []model.TargetMatch {
	var matches []model.TargetMatch
	for _, slide := range inspection.Slides {
		if slideNumber != 0 && slide.Number != slideNumber {
			continue
		}
		for _, image := range slide.Images {
			matches = append(matches, model.TargetMatch{
				SlideNumber: slide.Number,
				SlideID:     slide.ID,
				ObjectID:    image.ObjectID,
				Kind:        "image",
			})
		}
	}
	return matches
}

func matchSlideNumber(inspection *model.Inspection, slideNumber int) []model.TargetMatch {
	for _, slide := range inspection.Slides {
		if slide.Number == slideNumber {
			return []model.TargetMatch{{
				SlideNumber: slide.Number,
				SlideID:     slide.ID,
				Kind:        "slide",
				Text:        slide.Title,
			}}
		}
	}
	return nil
}

func matchTitle(inspection *model.Inspection, title string) []model.TargetMatch {
	var matches []model.TargetMatch
	for _, slide := range inspection.Slides {
		if slide.Title == title {
			objectID := ""
			if len(slide.VisibleText) > 0 {
				objectID = slide.VisibleText[0].ObjectID
			}
			matches = append(matches, model.TargetMatch{
				SlideNumber: slide.Number,
				SlideID:     slide.ID,
				ObjectID:    objectID,
				Kind:        "title",
				Text:        slide.Title,
			})
		}
	}
	return matches
}

func matchVisibleText(inspection *model.Inspection, text string) []model.TargetMatch {
	var matches []model.TargetMatch
	for _, slide := range inspection.Slides {
		for _, block := range slide.VisibleText {
			if strings.Contains(block.Text, text) {
				matches = append(matches, model.TargetMatch{
					SlideNumber: slide.Number,
					SlideID:     slide.ID,
					ObjectID:    block.ObjectID,
					Kind:        "visible_text",
					Text:        block.Text,
				})
			}
		}
	}
	return matches
}

func matchObjectID(inspection *model.Inspection, objectID string) []model.TargetMatch {
	for _, slide := range inspection.Slides {
		for _, block := range slide.VisibleText {
			if block.ObjectID == objectID {
				return []model.TargetMatch{{
					SlideNumber: slide.Number,
					SlideID:     slide.ID,
					ObjectID:    block.ObjectID,
					Kind:        "visible_text",
					Text:        block.Text,
				}}
			}
		}
		for _, block := range slide.Notes {
			if block.ObjectID == objectID {
				return []model.TargetMatch{{
					SlideNumber: slide.Number,
					SlideID:     slide.ID,
					ObjectID:    block.ObjectID,
					Kind:        "notes",
					Text:        block.Text,
				}}
			}
		}
		for _, media := range slide.Media {
			if media.ObjectID == objectID {
				return []model.TargetMatch{{
					SlideNumber: slide.Number,
					SlideID:     slide.ID,
					ObjectID:    media.ObjectID,
					Kind:        media.Kind,
				}}
			}
		}
	}
	return nil
}

func matchNotes(inspection *model.Inspection, slideNumber int) []model.TargetMatch {
	for _, slide := range inspection.Slides {
		if slide.Number == slideNumber {
			return []model.TargetMatch{{
				SlideNumber: slide.Number,
				SlideID:     slide.ID,
				Kind:        "notes",
			}}
		}
	}
	return nil
}
