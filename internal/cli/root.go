package cli

import (
	"context"
	"fmt"
	"io"

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
	cmd.AddCommand(stubCommand("inspect", "Inspect a .pptx deck and return structured facts."))
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

func stubCommand(name string, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s is not implemented yet; current checkpoint is repository foundation", name)
		},
	}
	cmd.Flags().Bool("json", false, "emit stable machine-readable JSON")
	return cmd
}
