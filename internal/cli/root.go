package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	createworkflow "github.com/artpar/puppt/internal/create"
	editworkflow "github.com/artpar/puppt/internal/edit"
	inspectworkflow "github.com/artpar/puppt/internal/inspect"
	"github.com/artpar/puppt/internal/report"
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
	cmd.AddCommand(stubCommand("validate", "Validate a .pptx deck for structure and expected content."))
	cmd.AddCommand(stubCommand("review", "Summarize deck changes for agents and human reviewers."))

	return cmd
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

func stubCommand(name string, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s is not implemented yet", name)
		},
	}
	cmd.Flags().Bool("json", false, "emit stable machine-readable JSON")
	return cmd
}
