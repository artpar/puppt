package inspect

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/artpar/puppt/internal/model"
	"github.com/artpar/puppt/internal/pptx"
)

// Inspect reads a .pptx deck and returns stable, agent-facing inspection facts.
func Inspect(ctx context.Context, filePath string) (*model.CommandResult, error) {
	pkg, err := pptx.Open(ctx, filePath)
	if err != nil {
		return nil, err
	}

	slides := make([]model.Slide, 0, len(pkg.SlideParts))
	repeated := make(map[string]int)
	metadata, err := inspectMetadata(pkg)
	if err != nil {
		return nil, err
	}

	for index, slidePart := range pkg.SlideParts {
		blocks, err := visibleTextBlocks(slidePart, pkg.Parts[slidePart])
		if err != nil {
			return nil, err
		}
		notes, images, layout, slideWarnings, err := inspectSlideRelationships(pkg, slidePart)
		if err != nil {
			return nil, err
		}
		for _, block := range blocks {
			if block.Text != "" {
				repeated[block.Text]++
			}
		}

		title := ""
		if len(blocks) > 0 {
			title = blocks[0].Text
		}

		slides = append(slides, model.Slide{
			Number:      index + 1,
			ID:          slidePart,
			Part:        slidePart,
			Layout:      layout,
			Title:       title,
			VisibleText: blocks,
			Notes:       notes,
			Images:      images,
			Warnings:    slideWarnings,
		})
	}

	warnings := []model.Warning{
		{
			Code:    "inspection_partial",
			Message: "inspection currently covers text shapes, notes, image refs, layouts, and metadata; advanced non-text object extraction and unsupported-feature warnings are not complete yet",
		},
	}

	return &model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "inspect",
		Status:        "ok",
		Input:         filePath,
		Output:        nil,
		Warnings:      warnings,
		Errors:        []model.ErrorItem{},
		Summary: model.Summary{
			Human: fmt.Sprintf("Found %d slides.", len(slides)),
		},
		Inspection: &model.Inspection{
			PresentationPart: pkg.PresentationPath,
			PartCount:        len(pkg.Parts),
			SlideCount:       len(slides),
			Metadata:         metadata,
			Slides:           slides,
			RepeatedText:     repeatedText(repeated),
		},
	}, nil
}

func visibleTextBlocks(slidePart string, data []byte) ([]model.TextBlock, error) {
	blocks, err := shapeTextBlocks(slidePart, data)
	if err != nil {
		return nil, err
	}
	if len(blocks) > 0 {
		return blocks, nil
	}
	return textBlocks(slidePart, slidePart+"#text-1", data)
}

func textBlocks(partName string, objectID string, data []byte) ([]model.TextBlock, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var runs []string

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse text %s: %w", partName, err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "t" {
			continue
		}

		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return nil, fmt.Errorf("parse text run %s: %w", partName, err)
		}
		if value != "" {
			runs = append(runs, value)
		}
	}

	if len(runs) == 0 {
		return []model.TextBlock{}, nil
	}

	text := strings.Join(runs, "")
	return []model.TextBlock{
		{
			ObjectID: objectID,
			Text:     text,
			Runs:     runs,
		},
	}, nil
}

type slideTextXML struct {
	CommonSlideData commonSlideDataXML `xml:"cSld"`
}

type commonSlideDataXML struct {
	ShapeTree shapeTreeXML `xml:"spTree"`
}

type shapeTreeXML struct {
	Shapes []shapeXML `xml:"sp"`
}

type shapeXML struct {
	NonVisual nonVisualShapeXML `xml:"nvSpPr"`
	TextBody  textBodyXML       `xml:"txBody"`
}

type nonVisualShapeXML struct {
	Properties nonVisualPropertiesXML `xml:"cNvPr"`
}

type nonVisualPropertiesXML struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr"`
}

type textBodyXML struct {
	Paragraphs []paragraphXML `xml:"p"`
}

type paragraphXML struct {
	Runs []runXML `xml:"r"`
}

type runXML struct {
	Text string `xml:"t"`
}

func shapeTextBlocks(slidePart string, data []byte) ([]model.TextBlock, error) {
	var slide slideTextXML
	if err := xml.NewDecoder(bytes.NewReader(data)).Decode(&slide); err != nil {
		return nil, fmt.Errorf("parse slide shapes %s: %w", slidePart, err)
	}

	blocks := make([]model.TextBlock, 0, len(slide.CommonSlideData.ShapeTree.Shapes))
	for index, shape := range slide.CommonSlideData.ShapeTree.Shapes {
		runs := shape.textRuns()
		if len(runs) == 0 {
			continue
		}
		blocks = append(blocks, model.TextBlock{
			ObjectID: shapeObjectID(slidePart, index, shape.NonVisual.Properties),
			Text:     strings.Join(runs, ""),
			Runs:     runs,
		})
	}
	return blocks, nil
}

func (s shapeXML) textRuns() []string {
	var runs []string
	for _, paragraph := range s.TextBody.Paragraphs {
		for _, run := range paragraph.Runs {
			if run.Text != "" {
				runs = append(runs, run.Text)
			}
		}
	}
	return runs
}

func shapeObjectID(slidePart string, index int, properties nonVisualPropertiesXML) string {
	if properties.ID != "" {
		return slidePart + "#shape-" + properties.ID
	}
	return fmt.Sprintf("%s#shape-%d", slidePart, index+1)
}

func inspectSlideRelationships(pkg *pptx.Package, slidePart string) ([]model.TextBlock, []model.MediaRef, string, []model.Warning, error) {
	relationships, err := pkg.RelationshipsForPart(slidePart)
	if err != nil {
		return nil, nil, "", nil, err
	}

	notes := []model.TextBlock{}
	images := []model.MediaRef{}
	layout := ""
	warnings := []model.Warning{}

	for _, relationship := range relationships {
		targetPart := pptx.ResolveTargetPart(slidePart, relationship.Target)
		switch relationship.Type {
		case pptx.NotesSlideRelType:
			data, ok := pkg.Parts[targetPart]
			if !ok {
				warnings = append(warnings, model.Warning{
					Code:    "missing_notes_part",
					Message: "slide references a notes part that is missing from the package",
					Part:    targetPart,
				})
				continue
			}
			blocks, err := noteTextBlocks(targetPart, data)
			if err != nil {
				return nil, nil, "", nil, err
			}
			notes = append(notes, blocks...)
		case pptx.ImageRelType:
			images = append(images, model.MediaRef{
				ObjectID:         slidePart + "#" + relationship.ID,
				Relationship:     relationship.ID,
				Target:           targetPart,
				ContentType:      pkg.ContentTypes.ForPart(targetPart),
				RelationshipType: relationship.Type,
			})
		case pptx.SlideLayoutRelType:
			layout = targetPart
		}
	}

	return notes, images, layout, warnings, nil
}

func noteTextBlocks(notesPart string, data []byte) ([]model.TextBlock, error) {
	blocks, err := shapeTextBlocks(notesPart, data)
	if err != nil {
		return nil, err
	}
	if len(blocks) > 0 {
		return blocks, nil
	}
	return textBlocks(notesPart, notesPart+"#notes-1", data)
}

func inspectMetadata(pkg *pptx.Package) (model.Metadata, error) {
	for _, relationship := range pkg.RootRelationships {
		if relationship.Type != pptx.CorePropertiesRelType {
			continue
		}
		targetPart := pptx.ResolveTargetPart("", relationship.Target)
		data, ok := pkg.Parts[targetPart]
		if !ok {
			return model.Metadata{}, fmt.Errorf("core properties relationship target missing: %s", targetPart)
		}
		return parseCoreProperties(data)
	}
	return model.Metadata{}, nil
}

func parseCoreProperties(data []byte) (model.Metadata, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var metadata model.Metadata

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return model.Metadata{}, fmt.Errorf("parse core properties: %w", err)
		}

		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}

		var value string
		switch start.Name.Local {
		case "title":
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return model.Metadata{}, err
			}
			metadata.Title = value
		case "creator":
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return model.Metadata{}, err
			}
			metadata.Author = value
		case "subject":
			if err := decoder.DecodeElement(&value, &start); err != nil {
				return model.Metadata{}, err
			}
			metadata.Subject = value
		}
	}
	return metadata, nil
}

func repeatedText(counts map[string]int) []model.RepeatedText {
	items := make([]model.RepeatedText, 0)
	for text, count := range counts {
		if count > 1 {
			items = append(items, model.RepeatedText{Text: text, Count: count})
		}
	}
	sort.Slice(items, func(left int, right int) bool {
		return items[left].Text < items[right].Text
	})
	return items
}
