package cli

import (
	"context"
	"fmt"
	"io"

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
	cmd.AddCommand(stubCommand("plan", "Plan a targeted deck edit without writing output."))
	cmd.AddCommand(stubCommand("edit", "Apply targeted edits to a .pptx deck."))
	cmd.AddCommand(stubCommand("create", "Create an editable .pptx deck from structured input."))
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
