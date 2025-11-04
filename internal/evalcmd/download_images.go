package evalcmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
	"github.com/lehigh-university-libraries/cataloger/internal/images"
)

// Number of pages to download per book (easy to change)
const DEFAULT_PAGES_PER_BOOK = 10

func executeDownloadImages(datasetPath, outputDir string, sampleSize int, verbose bool) error {
	slog.Info("Starting image download", "dataset", datasetPath, "output", outputDir, "sample", sampleSize)

	// Load Institutional Books dataset
	slog.Info("Loading Institutional Books dataset...")
	loader := dataset.NewLoader(datasetPath)

	var records []dataset.InstitutionalBooksRecord
	var err error

	if sampleSize > 0 {
		records, err = loader.LoadSample(sampleSize)
	} else {
		records, err = loader.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	slog.Info("Loaded dataset records", "count", len(records))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize image fetcher
	fetcher := images.NewFetcher()

	successCount := 0
	skipCount := 0
	errorCount := 0

	for i, record := range records {
		slog.Info("Processing record", "index", i+1, "total", len(records), "barcode", record.BarcodeSource)

		// Get ISBN from record
		isbn := record.GetISBN()
		if isbn == "" {
			slog.Warn("No ISBN found for record", "barcode", record.BarcodeSource)
			skipCount++
			continue
		}

		cleanISBN := images.CleanISBN(isbn)
		slog.Info("Processing book", "barcode", record.BarcodeSource, "isbn", cleanISBN, "title", record.TitleSource)

		// Create directory for this book (use barcode as unique identifier)
		bookDir := filepath.Join(outputDir, record.BarcodeSource)
		if err := os.MkdirAll(bookDir, 0755); err != nil {
			slog.Error("Failed to create book directory", "barcode", record.BarcodeSource, "error", err)
			errorCount++
			continue
		}

		// Check if images already exist - if so, skip
		existingImages, _ := filepath.Glob(filepath.Join(bookDir, "page_*.jpg"))
		if len(existingImages) > 0 {
			slog.Info("Images already exist, skipping", "barcode", record.BarcodeSource, "count", len(existingImages))
			skipCount++
			continue
		}

		// Download pages using Google Books
		pagesDownloaded, err := images.DownloadGoogleBooksPages(fetcher, cleanISBN, bookDir, DEFAULT_PAGES_PER_BOOK)
		if err != nil {
			slog.Warn("Failed to download pages", "isbn", cleanISBN, "barcode", record.BarcodeSource, "error", err)
			errorCount++
			continue
		}

		if pagesDownloaded == 0 {
			slog.Warn("No pages downloaded", "isbn", cleanISBN, "barcode", record.BarcodeSource)
			errorCount++
			continue
		}

		slog.Info("Downloaded pages", "isbn", cleanISBN, "barcode", record.BarcodeSource, "pages", pagesDownloaded)
		successCount++
	}

	fmt.Printf("\nImage download complete!\n")
	fmt.Printf("  Successfully processed: %d\n", successCount)
	fmt.Printf("  Skipped (no ISBN or already exists): %d\n", skipCount)
	fmt.Printf("  Errors: %d\n", errorCount)
	fmt.Printf("  Output location: %s\n", outputDir)
	fmt.Printf("\nEach book directory contains:\n")
	fmt.Printf("  - page_1.jpg, page_2.jpg, ...: First %d pages from Google Books preview\n", DEFAULT_PAGES_PER_BOOK)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Run evaluation: cataloger eval run --dataset %s --provider ollama\n", outputDir)

	return nil
}
