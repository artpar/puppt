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

	for index, slidePart := range pkg.SlideParts {
		blocks, err := visibleTextBlocks(slidePart, pkg.Parts[slidePart])
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
			Title:       title,
			VisibleText: blocks,
			Notes:       []model.TextBlock{},
			Images:      []model.MediaRef{},
			Warnings:    []model.Warning{},
		})
	}

	warnings := []model.Warning{
		{
			Code:    "inspection_partial",
			Message: "inspection currently includes package structure, slide order, and visible slide text; notes, images, layouts, and metadata are represented but not populated yet",
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
			Metadata:         model.Metadata{},
			Slides:           slides,
			RepeatedText:     repeatedText(repeated),
		},
	}, nil
}

func visibleTextBlocks(slidePart string, data []byte) ([]model.TextBlock, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var runs []string

	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("parse slide text %s: %w", slidePart, err)
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "t" {
			continue
		}

		var value string
		if err := decoder.DecodeElement(&value, &start); err != nil {
			return nil, fmt.Errorf("parse text run %s: %w", slidePart, err)
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
			ObjectID: slidePart + "#text-1",
			Text:     text,
			Runs:     runs,
		},
	}, nil
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
