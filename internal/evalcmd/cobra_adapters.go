package evalcmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)


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
		Long: `Evaluate metadata extraction using the Institutional Books 1.0 dataset from HuggingFace.

This dataset contains OCR text from book title pages with ground truth metadata.
The evaluation compares LLM-generated metadata fields against the reference metadata.

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
	cmd.Flags().StringVar(&provider, "provider", "ollama", "LLM provider (ollama, openai, or gemini)")
	cmd.Flags().StringVar(&model, "model", "", "Model name (defaults to provider's default)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose logging")

	return cmd
}

// NewDownloadImagesCmd creates the download-images command for downloading book page images
func NewDownloadImagesCmd() *cobra.Command {
	var datasetPath string
	var outputDir string
	var sampleSize int
	var verbose bool

	cmd := &cobra.Command{
		Use:   "download-images",
		Short: "Download book page images from Google Books for Institutional Books dataset",
		Long: `Download the first N pages from Google Books for each book in the Institutional Books dataset.

This command uses the ISBNs from the Institutional Books dataset to fetch preview page images
from Google Books. Images are stored in directories named by barcode for easy reference.

The number of pages to download per book is configurable via the DEFAULT_PAGES_PER_BOOK constant
(currently set to 10 pages per book).`,
		Example: `  # Download images for 10 books
  cataloger eval download-images --dataset ./institutional-books-1.0/data/train-00000-of-09831.parquet --sample 10

  # Download images for 100 books with verbose logging
  cataloger eval download-images --dataset ./institutional-books-1.0/data/train-00000-of-09831.parquet --sample 100 --verbose

  # Download images for all books in the parquet file
  cataloger eval download-images --dataset ./institutional-books-1.0/data/train-00000-of-09831.parquet --sample -1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if datasetPath == "" {
				return fmt.Errorf("--dataset is required")
			}

			// Check if dataset file exists
			if _, err := os.Stat(datasetPath); os.IsNotExist(err) {
				return fmt.Errorf("dataset file not found: %s\n\nPlease clone the dataset first:\n  git clone https://huggingface.co/datasets/instdin/institutional-books-1.0", datasetPath)
			}

			return executeDownloadImages(datasetPath, outputDir, sampleSize, verbose)
		},
	}

	cmd.Flags().StringVar(&datasetPath, "dataset", "", "Path to Institutional Books parquet file (required)")
	cmd.Flags().StringVar(&outputDir, "output", "./book_images", "Output directory for downloaded images")
	cmd.Flags().IntVar(&sampleSize, "sample", 10, "Number of books to process (-1 for all)")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Verbose logging")

	_ = cmd.MarkFlagRequired("dataset")
	return cmd
}
