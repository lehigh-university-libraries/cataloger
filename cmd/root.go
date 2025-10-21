package cmd

import (
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cataloger",
		Short: "MARC cataloging tool with LLM-powered metadata generation",
		Long: `Cataloger is a web-based book metadata cataloging tool that generates
MARC records from images of book title pages using vision-capable LLMs.

It supports both a web interface for interactive cataloging and a powerful
CLI for evaluating cataloging accuracy against professional catalog records.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load .env file if present (ignore errors)
			_ = godotenv.Load()
		},
	}

	// Add subcommands
	cmd.AddCommand(newServeCmd())
	cmd.AddCommand(newEvalCmd())

	return cmd
}
