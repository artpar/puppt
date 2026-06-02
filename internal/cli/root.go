package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	createworkflow "github.com/artpar/puppt/internal/create"
	editworkflow "github.com/artpar/puppt/internal/edit"
	inspectworkflow "github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/model"
	renderworkflow "github.com/artpar/puppt/internal/render"
	"github.com/artpar/puppt/internal/report"
	reviewworkflow "github.com/artpar/puppt/internal/review"
	validateworkflow "github.com/artpar/puppt/internal/validate"
	"github.com/spf13/cobra"
)

const (
	appName       = "puppt"
	schemaVersion = "puppt.v1"
)

var version = "dev"

// Execute runs the Puppt CLI with explicit streams so command behavior is testable.
func Execute(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) error {
	cmd := NewRootCommand()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	return cmd.ExecuteContext(ctx)
}

// NewRootCommand creates the top-level CLI command. Business logic belongs in
// internal workflow packages; this package should stay as command wiring.
func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           appName,
		Short:         "Inspect, edit, create, validate, and review PowerPoint .pptx files.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newInspectCommand())
	cmd.AddCommand(newPlanCommand())
	cmd.AddCommand(newEditCommand())
	cmd.AddCommand(newCreateCommand())
	cmd.AddCommand(newValidateCommand())
	cmd.AddCommand(newReviewCommand())
	cmd.AddCommand(newRenderCommand())

	return cmd
}

func newRenderCommand() *cobra.Command {
	var slideNumber int
	var slideRange string
	var allSlides bool
	var outputPath string
	var outputDPI int
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "render <input.pptx>",
		Short: "Render .pptx slides to PNG images.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			slides, err := renderSlideSelection(cmd.Context(), args[0], slideNumber, slideRange, allSlides)
			if err != nil {
				return err
			}
			result, err := renderSelectedSlides(cmd.Context(), args[0], slides, outputPath, outputDPI)
			if err != nil {
				return err
			}
			if emitJSON {
				return report.WriteJSON(cmd.OutOrStdout(), result)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human)
			return err
		},
	}
	cmd.Flags().IntVar(&slideNumber, "slide", 0, "1-based slide number to render")
	cmd.Flags().StringVar(&slideRange, "slides", "", "1-based slide range/list to render, e.g. 1-3,5")
	cmd.Flags().BoolVar(&allSlides, "all", false, "render all slides")
	cmd.Flags().StringVar(&outputPath, "out", "", "path to write rendered PNG, output directory, or template containing {slide}")
	cmd.Flags().IntVar(&outputDPI, "dpi", 72, "output PNG resolution in pixels per inch")
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	cmd.MarkFlagRequired("out")
	return cmd
}

func renderSlideSelection(ctx context.Context, inputPath string, slideNumber int, slideRange string, allSlides bool) ([]int, error) {
	selectedModes := 0
	if slideNumber > 0 {
		selectedModes++
	}
	if strings.TrimSpace(slideRange) != "" {
		selectedModes++
	}
	if allSlides {
		selectedModes++
	}
	if selectedModes != 1 {
		return nil, errors.New("exactly one of --slide, --slides, or --all is required")
	}
	if slideNumber > 0 {
		return []int{slideNumber}, nil
	}
	if allSlides {
		count, err := renderworkflow.SlideCount(ctx, inputPath)
		if err != nil {
			return nil, err
		}
		slides := make([]int, count)
		for index := range slides {
			slides[index] = index + 1
		}
		return slides, nil
	}
	return parseSlideRange(slideRange)
}

func parseSlideRange(value string) ([]int, error) {
	var slides []int
	seen := map[int]bool{}
	for _, rawItem := range strings.Split(value, ",") {
		item := strings.TrimSpace(rawItem)
		if item == "" {
			return nil, fmt.Errorf("invalid slide range %q", value)
		}
		startText, endText, hasRange := strings.Cut(item, "-")
		start, err := parsePositiveSlideNumber(startText)
		if err != nil {
			return nil, fmt.Errorf("invalid slide range %q: %w", value, err)
		}
		end := start
		if hasRange {
			end, err = parsePositiveSlideNumber(endText)
			if err != nil {
				return nil, fmt.Errorf("invalid slide range %q: %w", value, err)
			}
			if end < start {
				return nil, fmt.Errorf("invalid slide range %q: range %d-%d is descending", value, start, end)
			}
		}
		for slide := start; slide <= end; slide++ {
			if seen[slide] {
				continue
			}
			seen[slide] = true
			slides = append(slides, slide)
		}
	}
	if len(slides) == 0 {
		return nil, fmt.Errorf("invalid slide range %q", value)
	}
	return slides, nil
}

func parsePositiveSlideNumber(value string) (int, error) {
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, err
	}
	if number < 1 {
		return 0, fmt.Errorf("slide number must be >= 1")
	}
	return number, nil
}

func renderSelectedSlides(ctx context.Context, inputPath string, slides []int, outputPath string, dpi int) (model.CommandResult, error) {
	if outputPath == "" {
		return model.CommandResult{}, errors.New("render output path is required")
	}
	if len(slides) == 1 {
		slideOutput := outputPath
		if slideOutputPlaceholder.MatchString(outputPath) {
			var err error
			slideOutput, err = renderOutputPathForSlide(inputPath, outputPath, slides[0], true)
			if err != nil {
				return model.CommandResult{}, err
			}
		}
		return renderworkflow.Render(ctx, inputPath, renderworkflow.Options{
			SlideNumber: slides[0],
			OutputPath:  slideOutput,
			DPI:         dpi,
		})
	}
	result := model.CommandResult{
		SchemaVersion: model.SchemaVersion,
		Command:       "render",
		Status:        "ok",
		Input:         inputPath,
		Output:        &outputPath,
		Warnings:      []model.Warning{},
		Errors:        []model.ErrorItem{},
		Unsupported:   []model.SkipItem{},
		Summary:       model.Summary{Human: fmt.Sprintf("Rendered %d slides to %s.", len(slides), outputPath)},
	}
	for _, slide := range slides {
		slideOutput, err := renderOutputPathForSlide(inputPath, outputPath, slide, true)
		if err != nil {
			return result, err
		}
		slideResult, err := renderworkflow.Render(ctx, inputPath, renderworkflow.Options{
			SlideNumber: slide,
			OutputPath:  slideOutput,
			DPI:         dpi,
		})
		if err != nil {
			return result, err
		}
		result.Outputs = append(result.Outputs, slideOutput)
		if slideResult.Render != nil {
			result.Renders = append(result.Renders, *slideResult.Render)
		}
		result.Unsupported = append(result.Unsupported, slideResult.Unsupported...)
		if slideResult.Status != "ok" {
			result.Status = slideResult.Status
		}
	}
	if len(result.Unsupported) > 0 {
		result.Status = "partial"
		result.Summary = model.Summary{Human: fmt.Sprintf("Rendered %d slides with %d unsupported object(s).", len(slides), len(result.Unsupported))}
	}
	return result, nil
}

var slideOutputPlaceholder = regexp.MustCompile(`\{slide(?::0?([0-9]+))?\}`)

func renderOutputPathForSlide(inputPath string, outputPath string, slide int, multiple bool) (string, error) {
	if !multiple {
		return outputPath, nil
	}
	if slideOutputPlaceholder.MatchString(outputPath) {
		path := slideOutputPlaceholder.ReplaceAllStringFunc(outputPath, func(match string) string {
			submatches := slideOutputPlaceholder.FindStringSubmatch(match)
			if len(submatches) == 2 && submatches[1] != "" {
				width, err := strconv.Atoi(submatches[1])
				if err == nil && width > 0 {
					return fmt.Sprintf("%0*d", width, slide)
				}
			}
			return strconv.Itoa(slide)
		})
		return ensureRenderOutputParent(path)
	}
	if info, err := os.Stat(outputPath); err == nil && info.IsDir() {
		outputDir := filepath.Join(outputPath, deckOutputDirectoryName(inputPath))
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return "", err
		}
		return filepath.Join(outputDir, fmt.Sprintf("slide-%03d.png", slide)), nil
	}
	if strings.EqualFold(filepath.Ext(outputPath), ".png") {
		return "", errors.New("multiple-slide render requires --out to be a directory or a template containing {slide}")
	}
	outputDir := filepath.Join(outputPath, deckOutputDirectoryName(inputPath))
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(outputDir, fmt.Sprintf("slide-%03d.png", slide)), nil
}

func deckOutputDirectoryName(inputPath string) string {
	base := filepath.Base(inputPath)
	extension := filepath.Ext(base)
	name := strings.TrimSuffix(base, extension)
	if name == "" {
		return "deck"
	}
	return name
}

func ensureRenderOutputParent(path string) (string, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", err
		}
	}
	return path, nil
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Puppt version information.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "%s %s (%s)\n", appName, version, schemaVersion)
			return err
		},
	}
}

func newPlanCommand() *cobra.Command {
	var editPath string
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "plan <input.pptx>",
		Short: "Plan a targeted deck edit without writing output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := editworkflow.Plan(cmd.Context(), args[0], editPath)
			if err != nil {
				return err
			}
			if emitJSON {
				if err := report.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				if result.Status != "ok" {
					return errors.New(result.Summary.Human)
				}
				return nil
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human); err != nil {
				return err
			}
			if result.Status != "ok" {
				return errors.New(result.Summary.Human)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&editPath, "edit", "", "path to edit spec JSON")
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	cmd.MarkFlagRequired("edit")
	return cmd
}

func newEditCommand() *cobra.Command {
	var editPath string
	var outputPath string
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "edit <input.pptx>",
		Short: "Apply targeted edits to a .pptx deck.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := editworkflow.Apply(cmd.Context(), args[0], editPath, outputPath)
			if err != nil {
				return err
			}
			if emitJSON {
				if err := report.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				if result.Status != "ok" {
					return errors.New(result.Summary.Human)
				}
				return nil
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human); err != nil {
				return err
			}
			if result.Status != "ok" {
				return errors.New(result.Summary.Human)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&editPath, "edit", "", "path to edit spec JSON")
	cmd.Flags().StringVar(&outputPath, "out", "", "path to write edited .pptx")
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	cmd.MarkFlagRequired("edit")
	cmd.MarkFlagRequired("out")
	return cmd
}

func newCreateCommand() *cobra.Command {
	var inputPath string
	var outputPath string
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an editable .pptx deck from structured input.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := createworkflow.Create(cmd.Context(), inputPath, outputPath)
			if err != nil {
				return err
			}
			if emitJSON {
				if err := report.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				if result.Status != "ok" {
					return errors.New(result.Summary.Human)
				}
				return nil
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human); err != nil {
				return err
			}
			if result.Status != "ok" {
				return errors.New(result.Summary.Human)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&inputPath, "input", "", "path to structured deck JSON")
	cmd.Flags().StringVar(&outputPath, "out", "", "path to write created .pptx")
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	cmd.MarkFlagRequired("input")
	cmd.MarkFlagRequired("out")
	return cmd
}

func newInspectCommand() *cobra.Command {
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "inspect <input.pptx>",
		Short: "Inspect a .pptx deck and return structured facts.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := inspectworkflow.Inspect(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if emitJSON {
				return report.WriteJSON(cmd.OutOrStdout(), result)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human)
			return err
		},
	}
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	return cmd
}

func newValidateCommand() *cobra.Command {
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "validate <input.pptx>",
		Short: "Validate a .pptx deck for structure and expected content.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := validateworkflow.Validate(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if emitJSON {
				if err := report.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				if result.Status != "ok" {
					return errors.New(result.Summary.Human)
				}
				return nil
			}
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human); err != nil {
				return err
			}
			if result.Status != "ok" {
				return errors.New(result.Summary.Human)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	return cmd
}

func newReviewCommand() *cobra.Command {
	var changesPath string
	var emitJSON bool
	cmd := &cobra.Command{
		Use:   "review <input.pptx>",
		Short: "Summarize deck changes for agents and human reviewers.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := reviewworkflow.Review(cmd.Context(), args[0], changesPath)
			if err != nil {
				return err
			}
			if emitJSON {
				if err := report.WriteJSON(cmd.OutOrStdout(), result); err != nil {
					return err
				}
				if result.Status != "ok" {
					return errors.New(result.Summary.Human)
				}
				return nil
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), result.Summary.Human)
			return err
		},
	}
	cmd.Flags().StringVar(&changesPath, "changes", "", "path to changes JSON")
	cmd.Flags().BoolVar(&emitJSON, "json", false, "emit stable machine-readable JSON")
	cmd.MarkFlagRequired("changes")
	return cmd
}
