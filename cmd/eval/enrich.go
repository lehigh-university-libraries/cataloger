package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hectorcorrea/marcli/pkg/marc"
	"github.com/lehigh-university-libraries/cataloger/internal/evaluation"
	"github.com/lehigh-university-libraries/cataloger/internal/images"
)

func executeEnrich(datasetDir, outputDir string) error {
	slog.Info("Starting dataset enrichment", "dataset", datasetDir, "output", outputDir)

	// Load the dataset
	dataset, err := evaluation.LoadDataset(datasetDir)
	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	slog.Info("Loaded dataset", "records", len(dataset.Items))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize image fetcher
	imageFetcher := images.NewFetcher()

	successCount := 0
	skipCount := 0
	errorCount := 0

	for i, item := range dataset.Items {
		slog.Info("Processing record", "index", i+1, "total", len(dataset.Items), "id", item.ID)

		// Parse MARC record to extract ISBN
		marcRecord, err := parseMARCRecordFromString(item.ReferenceMARC)
		if err != nil {
			slog.Warn("Failed to parse MARC record", "id", item.ID, "error", err)
			errorCount++
			continue
		}

		// Extract ISBN
		isbn := extractISBNFromMARC(marcRecord)
		if isbn == "" {
			slog.Warn("No ISBN found in record", "id", item.ID)
			skipCount++
			continue
		}

		cleanISBN := images.CleanISBN(isbn)
		slog.Info("Found ISBN", "id", item.ID, "isbn", cleanISBN)

		// Create ISBN-based directory
		isbnDir := filepath.Join(outputDir, cleanISBN)
		if err := os.MkdirAll(isbnDir, 0755); err != nil {
			slog.Error("Failed to create ISBN directory", "isbn", cleanISBN, "error", err)
			errorCount++
			continue
		}

		// Check if marc.xml already exists - if so, skip this ISBN entirely
		marcPath := filepath.Join(isbnDir, "marc.xml")
		if _, err := os.Stat(marcPath); err == nil {
			slog.Info("MARC file already exists, skipping", "isbn", cleanISBN)
			skipCount++
			continue
		}

		// Write original MARCXML to file (as harvested from OAI-PMH)
		// The ReferenceMARC from OAI-PMH is already MARCXML
		if err := os.WriteFile(marcPath, []byte(item.ReferenceMARC), 0644); err != nil {
			slog.Error("Failed to write MARC file", "isbn", cleanISBN, "error", err)
			errorCount++
			continue
		}

		// Fetch images
		slog.Info("Fetching images", "isbn", cleanISBN)
		imageSet, err := imageFetcher.FetchImagesForISBN(cleanISBN, isbnDir)
		if err != nil {
			slog.Warn("Failed to fetch images", "isbn", cleanISBN, "error", err)
			// Don't count as error - MARCXML still created
		} else {
			imageCount := 0
			if imageSet.CoverPath != "" {
				imageCount++
			}
			if imageSet.TitlePagePath != "" {
				imageCount++
			}
			if imageSet.CopyrightPagePath != "" {
				imageCount++
			}
			slog.Info("Fetched images", "isbn", cleanISBN, "count", imageCount)
		}

		successCount++

		// Rate limiting - be nice to Open Library (100 req/5min = 3 seconds between requests)
		if i < len(dataset.Items)-1 {
			time.Sleep(3 * time.Second)
		}
	}

	fmt.Printf("\nDataset enrichment complete!\n")
	fmt.Printf("  Successfully processed: %d\n", successCount)
	fmt.Printf("  Skipped (no ISBN): %d\n", skipCount)
	fmt.Printf("  Errors: %d\n", errorCount)
	fmt.Printf("  Output location: %s\n", outputDir)
	fmt.Printf("\nEach ISBN directory contains:\n")
	fmt.Printf("  - marc.xml: Well-formatted MARCXML record\n")
	fmt.Printf("  - {ISBN}_cover.jpg: Book cover image (if available)\n")
	fmt.Printf("  - {ISBN}_title.jpg: Title page image (if available)\n")
	fmt.Printf("  - {ISBN}_copyright.jpg: Copyright page image (if available)\n")

	return nil
}

// parseMARCRecordFromString parses a MARC record from XML string
func parseMARCRecordFromString(marcXML string) (*marc.Record, error) {
	// Try parsing as MARCXML
	if strings.Contains(marcXML, "<marc:record") || strings.Contains(marcXML, "<record") {
		return parseMARCXML([]byte(marcXML))
	}

	// Otherwise try ISO2709 binary format
	return parseMARCBinary([]byte(marcXML))
}
