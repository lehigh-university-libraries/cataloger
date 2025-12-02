package cmd

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cataloger",
		Short: "Book metadata extraction tool with LLM-powered metadata generation",
		Long: `Cataloger is a tool for extracting metadata from book images using LLMs.

It supports a powerful CLI for evaluating metadata extraction accuracy against professional catalog records.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load .env file if present (ignore errors)
			_ = godotenv.Load()
		},
	}

	// Add subcommands
	cmd.AddCommand(newEvalCmd())

	return cmd
}
