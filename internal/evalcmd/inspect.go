package evalcmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
	"github.com/spf13/cobra"
)

// NewInspectCmd creates the inspect command
func NewInspectCmd() *cobra.Command {
	var datasetPath string
	var limit int
	var interactive bool
	var showOCR bool
	var showMetadata bool

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect dataset records (useful for examining OCR text)",
		Long: `Inspect records from a parquet or jsonl dataset file.

This command is useful for examining OCR text, metadata, and determining
appropriate character/page limits for sending to LLMs.`,
		Example: `  # Inspect first 5 records interactively
  cataloger eval inspect --dataset ./data.parquet --limit 5 --interactive

  # Show only OCR text
  cataloger eval inspect --dataset ./data.parquet --metadata=false

  # Inspect all records (no limit)
  cataloger eval inspect --dataset ./data.parquet --limit 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if datasetPath == "" {
				return fmt.Errorf("--dataset is required")
			}
			return executeInspect(datasetPath, limit, interactive, showOCR, showMetadata)
		},
	}

	cmd.Flags().StringVar(&datasetPath, "dataset", "", "Path to parquet or jsonl dataset file (required)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Number of records to inspect (0 for all)")
	cmd.Flags().BoolVar(&interactive, "interactive", false, "Pause after each record (press Enter to continue)")
	cmd.Flags().BoolVar(&showOCR, "ocr", true, "Show OCR text")
	cmd.Flags().BoolVar(&showMetadata, "metadata", true, "Show metadata (title, author, date, ISBN)")

	_ = cmd.MarkFlagRequired("dataset")

	return cmd
}

func executeInspect(datasetPath string, limit int, interactive, showOCR, showMetadata bool) error {
	loader := dataset.NewLoader(datasetPath)

	var records []dataset.InstitutionalBooksRecord
	var err error

	if limit > 0 {
		records, err = loader.LoadSample(limit)
	} else {
		records, err = loader.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	fmt.Printf("Loaded %d records from %s\n", len(records), datasetPath)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for i, record := range records {
		fmt.Printf("RECORD %d/%d\n", i+1, len(records))
		fmt.Println(strings.Repeat("-", 80))

		if showMetadata {
			fmt.Printf("Barcode:       %s\n", record.BarcodeSource)
			fmt.Printf("Title:         %s\n", record.TitleSource)
			fmt.Printf("Author:        %s\n", record.AuthorSource)
			fmt.Printf("Date:          %s\n", record.GetPrimaryDate())
			fmt.Printf("ISBN:          %s\n", record.GetISBN())
			fmt.Printf("Pages:         %d\n", len(record.TextByPageSource))
			fmt.Println()
		}

		if showOCR {
			ocrText := record.GetTitlePageText()
			fmt.Printf("OCR Text Length: %d characters\n", len(ocrText))
			fmt.Printf("OCR Text Length: %d words (approx)\n", len(strings.Fields(ocrText)))
			fmt.Println()

			// Show first 2000 characters with indicator if truncated
			displayText := ocrText
			truncated := false
			if len(displayText) > 2000 {
				displayText = displayText[:2000]
				truncated = true
			}

			fmt.Println("OCR TEXT (first 10 pages):")
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println(displayText)
			if truncated {
				fmt.Printf("\n[... truncated, showing first 2000 of %d characters ...]\n", len(ocrText))
			}
			fmt.Println(strings.Repeat("-", 80))
		}

		fmt.Println()

		if interactive {
			fmt.Print("Press Enter to continue to next record (or Ctrl+C to quit)...")
			_, _ = reader.ReadString('\n')
			fmt.Println()
		} else {
			fmt.Println()
		}
	}

	return nil
}
