package evalcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewFetchCmd creates the fetch command for harvesting OAI-PMH records
func NewFetchCmd() *cobra.Command {
	var oaiURL string
	var metadataPrefix string
	var outputDir string
	var limit int
	var excludeTags []string
	var sleepSeconds int

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch records from OAI-PMH endpoint to build evaluation dataset",
		Long: `Harvest MARC records from an OAI-PMH endpoint to build an evaluation dataset.

Automatically filters for books with ISBNs and excludes deleted/suppressed records.
Supports incremental saving and resumption token sleep delays.`,
		Example: `  # Fetch 100 records from FOLIO
  cataloger eval fetch --url https://folio.example.edu/oai --limit 100 --output ./dataset

  # Fetch with resumption token delay
  cataloger eval fetch --url https://folio.example.edu/oai --limit 500 --sleep 2

  # Exclude specific MARC tags
  cataloger eval fetch --url https://folio.example.edu/oai --limit 100 --exclude 856,999`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if oaiURL == "" {
				return fmt.Errorf("--url is required")
			}
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0")
			}
			return executeFetch(oaiURL, metadataPrefix, outputDir, limit, excludeTags, sleepSeconds)
		},
	}

	cmd.Flags().StringVar(&oaiURL, "url", "", "OAI-PMH endpoint URL (required)")
	cmd.Flags().StringVar(&metadataPrefix, "prefix", "marc21", "Metadata prefix for OAI-PMH")
	cmd.Flags().StringVar(&outputDir, "output", "./dataset", "Output directory for dataset")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of records to fetch")
	cmd.Flags().StringSliceVar(&excludeTags, "exclude", []string{}, "MARC tags to exclude (comma-separated)")
	cmd.Flags().IntVar(&sleepSeconds, "sleep", 0, "Seconds to sleep between resumption token requests")

	_ = cmd.MarkFlagRequired("url")
	return cmd
}

// NewEnrichCmd creates the enrich command for adding images and MARCXML to datasets
func NewEnrichCmd() *cobra.Command {
	var datasetDir string
	var outputDir string

	cmd := &cobra.Command{
		Use:   "enrich",
		Short: "Enrich dataset with images and MARCXML for each ISBN",
		Long: `Enrich a fetched dataset with images and well-formatted MARCXML.

For each record with an ISBN, this command:
- Creates an ISBN-based directory
- Saves the MARCXML record as marc.xml
- Fetches cover, title page, and copyright page images from Open Library
- Includes 3-second rate limiting to be respectful to Open Library`,
		Example: `  # Enrich dataset with images
  cataloger eval enrich --dataset ./dataset --output ./enriched

  # Use same directory for input and output
  cataloger eval enrich --dataset ./dataset`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if datasetDir == "" {
				return fmt.Errorf("--dataset is required")
			}
			if outputDir == "" {
				outputDir = datasetDir
			}
			return executeEnrich(datasetDir, outputDir)
		},
	}

	cmd.Flags().StringVar(&datasetDir, "dataset", "", "Path to dataset directory (required)")
	cmd.Flags().StringVar(&outputDir, "output", "", "Output directory (defaults to dataset directory)")

	_ = cmd.MarkFlagRequired("dataset")
	return cmd
}

// NewRunCmd creates the run command for evaluating MARC generation accuracy
func NewRunCmd() *cobra.Command {
	var datasetDir string
	var provider string
	var model string
	var outputDir string
	var concurrency int

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run evaluation on dataset",
		Long: `Run LLM-based MARC generation on a dataset and compare against reference records.

For each item in the dataset:
- Generates MARC from title page image using specified LLM
- Compares generated MARC with reference MARC from catalog
- Calculates field-by-field accuracy scores
- Produces detailed evaluation results`,
		Example: `  # Run evaluation with OpenAI
  cataloger eval run --dataset ./enriched --provider openai --model gpt-4o --output ./results

  # Run with Ollama using 4 concurrent workers
  cataloger eval run --dataset ./enriched --provider ollama --concurrency 4`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if datasetDir == "" {
				return fmt.Errorf("--dataset is required")
			}
			if provider == "" {
				return fmt.Errorf("--provider is required")
			}
			return executeRun(datasetDir, provider, model, outputDir, concurrency)
		},
	}

	cmd.Flags().StringVar(&datasetDir, "dataset", "", "Path to enriched dataset directory (required)")
	cmd.Flags().StringVar(&provider, "provider", "", "LLM provider (ollama or openai) (required)")
	cmd.Flags().StringVar(&model, "model", "", "Model name (defaults to env var or provider default)")
	cmd.Flags().StringVar(&outputDir, "output", "./results", "Output directory for evaluation results")
	cmd.Flags().IntVar(&concurrency, "concurrency", 1, "Number of concurrent evaluations")

	_ = cmd.MarkFlagRequired("dataset")
	_ = cmd.MarkFlagRequired("provider")
	return cmd
}

// NewReportCmd creates the report command for generating evaluation reports
func NewReportCmd() *cobra.Command {
	var resultsDir string
	var format string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate detailed comparison report",
		Long: `Generate a detailed report from evaluation results.

Supports multiple output formats:
- text: Human-readable summary with field-by-field comparisons
- json: Machine-readable JSON format for further analysis
- csv: Spreadsheet-compatible CSV format`,
		Example: `  # Generate text report
  cataloger eval report --results ./results --format text

  # Generate JSON report
  cataloger eval report --results ./results --format json > report.json

  # Generate CSV report
  cataloger eval report --results ./results --format csv > report.csv`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if resultsDir == "" {
				return fmt.Errorf("--results is required")
			}
			return executeReport(resultsDir, format)
		},
	}

	cmd.Flags().StringVar(&resultsDir, "results", "", "Path to evaluation results directory (required)")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json, csv)")

	_ = cmd.MarkFlagRequired("results")
	return cmd
}

// NewIBCmd creates the ib command for evaluating with Institutional Books dataset
func NewIBCmd() *cobra.Command {
	var datasetPath string
	var outputJSON string
	var outputReport string
	var sampleSize int
	var provider string
	var model string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "ib",
		Short: "Evaluate using Institutional Books 1.0 dataset",
		Long: `Evaluate MARC generation using the Institutional Books 1.0 dataset from HuggingFace.

This dataset contains OCR text from book title pages with ground truth metadata.
The evaluation compares LLM-generated MARC fields against the reference metadata.

Dataset: https://huggingface.co/datasets/instdin/institutional-books-1.0`,
		Example: `  # Evaluate 10 records with Ollama
  cataloger eval ib --sample 10 --provider ollama --verbose

  # Evaluate 100 records with OpenAI
  cataloger eval ib --sample 100 --provider openai --model gpt-4o

  # Evaluate full dataset (thousands of records)
  cataloger eval ib --sample -1 --provider openai`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set up logging
			logLevel := "info"
			if verbose {
				logLevel = "debug"
			}
			// Note: This would need proper integration with slog
			_ = logLevel

			// Check if dataset file exists
			if _, err := os.Stat(datasetPath); os.IsNotExist(err) {
				return fmt.Errorf("dataset file not found: %s\n\nPlease clone the dataset first:\n  git clone https://huggingface.co/datasets/instdin/institutional-books-1.0", datasetPath)
			}

			// Run the evaluation
			return executeIB(datasetPath, outputJSON, outputReport, sampleSize, provider, model, verbose)
		},
	}

	cmd.Flags().StringVar(&datasetPath, "dataset", "./institutional-books-1.0/data/train-00000-of-09831.parquet", "Path to Institutional Books parquet file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "eval_results.json", "Path to output JSON results file")
	cmd.Flags().StringVar(&outputReport, "output-report", "eval_report.txt", "Path to output detailed report file")
	cmd.Flags().IntVar(&sampleSize, "sample", 10, "Number of records to evaluate (-1 for all)")
	cmd.Flags().StringVar(&provider, "provider", "ollama", "LLM provider (ollama or openai)")
	cmd.Flags().StringVar(&model, "model", "", "Model name (defaults to provider's default)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose logging")

	return cmd
}
