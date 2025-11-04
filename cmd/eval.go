package cmd

import (
	"github.com/lehigh-university-libraries/cataloger/internal/evalcmd"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "MARC cataloging evaluation tools",
		Long: `Evaluation tools for measuring the accuracy of LLM-generated MARC records.

Supports fetching records from OAI-PMH endpoints, enriching datasets with images,
running evaluations against professional catalog records, and generating detailed
comparison reports.`,
	}

	// Add eval subcommands
	cmd.AddCommand(evalcmd.NewFetchCmd())
	cmd.AddCommand(evalcmd.NewEnrichCmd())
	cmd.AddCommand(evalcmd.NewRunCmd())
	cmd.AddCommand(evalcmd.NewReportCmd())
	cmd.AddCommand(evalcmd.NewIBCmd())
	cmd.AddCommand(evalcmd.NewInspectCmd())
	cmd.AddCommand(evalcmd.NewDownloadImagesCmd())

	return cmd
}
