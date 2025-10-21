package evalcmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewFetchCmd creates the fetch command (TODO: convert from flag.FlagSet to cobra)
func NewFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch records from OAI-PMH endpoint to build evaluation dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("fetch command not yet migrated to Cobra - please convert fetch.go")
		},
	}
}

// NewEnrichCmd creates the enrich command (TODO: convert from flag.FlagSet to cobra)
func NewEnrichCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enrich",
		Short: "Enrich dataset with images and MARCXML for each ISBN",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("enrich command not yet migrated to Cobra - please convert enrich.go")
		},
	}
}

// NewRunCmd creates the run command (TODO: convert from flag.FlagSet to cobra)
func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Run evaluation on dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("run command not yet migrated to Cobra - please convert run.go")
		},
	}
}

// NewReportCmd creates the report command (TODO: convert from flag.FlagSet to cobra)
func NewReportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Generate detailed comparison report",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("report command not yet migrated to Cobra - please convert report.go")
		},
	}
}

// NewIBCmd creates the ib command (TODO: convert from flag.FlagSet to cobra)
func NewIBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ib",
		Short: "Evaluate using Institutional Books 1.0 dataset",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("ib command not yet migrated to Cobra - please convert ib.go")
		},
	}
}
