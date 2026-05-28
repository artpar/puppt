package validate

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/artpar/puppt/internal/fixtures"
)

func TestValidateAcceptsFixtureDeck(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WriteMinimalPPTX(deckPath, []string{"ppt/slides/slide1.xml"}); err != nil {
		t.Fatal(err)
	}

	result, err := Validate(context.Background(), deckPath)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if result.Status != "ok" || result.Validation == nil || !result.Validation.Valid {
		t.Fatalf("unexpected validation result: %+v", result)
	}
}

func TestValidateReportsMissingRelationshipTarget(t *testing.T) {
	deckPath := filepath.Join(t.TempDir(), "deck.pptx")
	if err := fixtures.WritePPTX(deckPath, fixtures.PPTXOptions{
		Slides: []fixtures.Slide{{PartName: "ppt/slides/slide1.xml", Text: "Slide"}},
		ExtraParts: []fixtures.ExtraPart{{
			Name: "ppt/slides/_rels/slide1.xml.rels",
			Data: []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/missing.png"/>
</Relationships>
`),
		}},
	}); err != nil {
		t.Fatal(err)
	}

	result, err := Validate(context.Background(), deckPath)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if result.Status != "invalid" || result.Validation == nil || result.Validation.Valid {
		t.Fatalf("unexpected validation result: %+v", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Code != "missing_relationship_target" {
		t.Fatalf("unexpected errors: %+v", result.Errors)
	}
}
