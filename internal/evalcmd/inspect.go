package evalcmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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

			// Create a context that gets canceled on an interrupt signal (Ctrl+C)
			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop() // Ensure the signal handler is cleaned up

			return executeInspect(ctx, datasetPath, limit, interactive, showOCR, showMetadata)
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

func executeInspect(ctx context.Context, datasetPath string, limit int, interactive, showOCR, showMetadata bool) error {
	loader := dataset.NewLoader(datasetPath)

	var records []dataset.InstitutionalBooksRecord
	var err error

	// Note: Loading itself doesn't respect context here, but we'll assume it's fast.
	// A more complex implementation would involve passing context to the Loader.
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
		// Check for context cancellation (e.g., Ctrl+C) at the start of each iteration
		select {
		case <-ctx.Done():
			fmt.Println("\nInspection interrupted.")
			return nil // Return nil for a clean exit
		default:
			// Continue processing the record
		}

		fmt.Printf("RECORD %d/%d\n", i+1, len(records))
		fmt.Println(strings.Repeat("-", 80))

		if showMetadata {
			// Core identifiers
			fmt.Printf("Barcode:        %s\n", record.BarcodeSource)

			// Bibliographic metadata
			fmt.Printf("Title:          %s\n", record.TitleSource)
			fmt.Printf("Author:         %s\n", record.AuthorSource)
			fmt.Printf("Date1:          %s\n", record.Date1Source)
			fmt.Printf("Date2:          %s\n", record.Date2Source)
			fmt.Printf("Date Types:     %s\n", record.DateTypesSource)

			// Additional metadata
			fmt.Printf("Language:       %s\n", record.LanguageSource)
			fmt.Printf("Topic/Subject: %s\n", record.TopicOrSubjectSource)
			fmt.Printf("Genre/Form:     %s\n", record.GenreOrFormSource)
			fmt.Printf("General Note:   %s\n", record.GeneralNoteSource)

			// Identifiers
			if len(record.IdentifiersSource.ISBN) > 0 {
				fmt.Printf("ISBN(s):        %s\n", strings.Join(record.IdentifiersSource.ISBN, ", "))
			}
			if len(record.IdentifiersSource.LCCN) > 0 {
				fmt.Printf("LCCN(s):        %s\n", strings.Join(record.IdentifiersSource.LCCN, ", "))
			}
			if len(record.IdentifiersSource.OCLC) > 0 {
				fmt.Printf("OCLC(s):        %s\n", strings.Join(record.IdentifiersSource.OCLC, ", "))
			}

			// HathiTrust data
			if record.HathitrustDataExt.URL != "" {
				fmt.Printf("HathiTrust URL: %s\n", record.HathitrustDataExt.URL)
				fmt.Printf("Rights Code:    %s\n", record.HathitrustDataExt.RightsCode)
				fmt.Printf("Reason Code:    %s\n", record.HathitrustDataExt.ReasonCode)
				fmt.Printf("Last Check:     %s\n", record.HathitrustDataExt.LastCheck)
			}

			// Statistics
			fmt.Printf("Page Count:     %d\n", record.PageCountSource)
			fmt.Printf("Token Count:    %d\n", record.TokenCountGen)
			fmt.Printf("Pages w/ OCR:   %d\n", len(record.TextByPageSource))
			fmt.Println()
		}

		if showOCR {
			ocrText := record.GetTitlePageText()
			fmt.Printf("OCR Text Length: %d characters\n", len(ocrText))
			fmt.Printf("OCR Text Length: %d words (approx)\n", len(strings.Fields(ocrText)))
			fmt.Println()

			// Show first 500 characters with indicator if truncated
			displayText := ocrText
			truncated := false
			maxChars := 500
			if len(displayText) > maxChars {
				displayText = displayText[:maxChars]
				truncated = true
			}

			fmt.Println("OCR TEXT PREVIEW (first 10 pages):")
			fmt.Println(strings.Repeat("-", 80))
			fmt.Println(displayText)
			if truncated {
				fmt.Printf("\n[... truncated, showing first %d of %d characters ...]\n", maxChars, len(ocrText))
			}
			fmt.Println(strings.Repeat("-", 80))
		}

		fmt.Println()

		if interactive {
			fmt.Print("Press Enter to continue to next record (or Ctrl+C to quit)...")

			// Channel to signal user input
			inputCh := make(chan struct{})
			// Goroutine to wait for Enter key
			go func() {
				_, _ = reader.ReadString('\n')
				close(inputCh)
			}()

			// Wait for either user input (Enter) or context cancellation (Ctrl+C)
			select {
			case <-ctx.Done():
				// Context was canceled
				fmt.Println("\nInspection interrupted.")
				return nil // Clean exit
			case <-inputCh:
				// User pressed Enter
				fmt.Println()
			}
		} else {
			fmt.Println()
		}
	}

	return nil
}
