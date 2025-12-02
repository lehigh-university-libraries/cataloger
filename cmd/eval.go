package cmd

import (
	"github.com/lehigh-university-libraries/cataloger/internal/evalcmd"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Metadata extraction evaluation tools",
		Long:  `Evaluation tools for measuring the accuracy of LLM-generated metadata.`,
	}

	// Add eval subcommands
	cmd.AddCommand(evalcmd.NewIBCmd())
	cmd.AddCommand(evalcmd.NewInspectCmd())
	cmd.AddCommand(evalcmd.NewDownloadImagesCmd())

	return cmd
}
