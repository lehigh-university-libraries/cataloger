package main

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/lehigh-university-libraries/cataloger/internal/catalog"
	"github.com/lehigh-university-libraries/cataloger/internal/evaluation"
)

func executeFetch(catalogType, catalogURL, apiKey, outputDir string, limit int) error {
	slog.Info("Starting dataset fetch", "catalog", catalogType, "url", catalogURL, "limit", limit)

	// Create catalog client
	client := catalog.NewClient(catalogType, catalogURL, apiKey)

	// Fetch records from catalog
	slog.Info("Fetching records from catalog...")
	records, err := client.FetchRecords(limit)
	if err != nil {
		return fmt.Errorf("failed to fetch records: %w", err)
	}

	slog.Info("Fetched records", "count", len(records))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create images directory
	imagesDir := filepath.Join(outputDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create images directory: %w", err)
	}

	// Build dataset
	dataset := &evaluation.Dataset{
		Items: make([]evaluation.DatasetItem, 0, len(records)),
	}

	for i, record := range records {
		slog.Info("Processing record", "id", record.ID, "progress", fmt.Sprintf("%d/%d", i+1, len(records)))

		item := evaluation.DatasetItem{
			ID:            record.ID,
			ReferenceMARC: record.MARCRecord,
		}

		// Download images if URLs are available
		if record.CoverImageURL != "" {
			imagePath := filepath.Join(imagesDir, fmt.Sprintf("%s_cover.jpg", record.ID))
			if err := downloadImage(record.CoverImageURL, imagePath); err != nil {
				slog.Warn("Failed to download cover image", "id", record.ID, "error", err)
			} else {
				item.CoverImagePath = imagePath
			}
		}

		if record.TitlePageURL != "" {
			imagePath := filepath.Join(imagesDir, fmt.Sprintf("%s_titlepage.jpg", record.ID))
			if err := downloadImage(record.TitlePageURL, imagePath); err != nil {
				slog.Warn("Failed to download title page image", "id", record.ID, "error", err)
			} else {
				item.TitlePagePath = imagePath
			}
		}

		if record.CopyrightURL != "" {
			imagePath := filepath.Join(imagesDir, fmt.Sprintf("%s_copyright.jpg", record.ID))
			if err := downloadImage(record.CopyrightURL, imagePath); err != nil {
				slog.Warn("Failed to download copyright image", "id", record.ID, "error", err)
			} else {
				item.CopyrightPagePath = imagePath
			}
		}

		dataset.Items = append(dataset.Items, item)
	}

	// Save dataset
	slog.Info("Saving dataset", "output", outputDir)
	if err := evaluation.SaveDataset(dataset, outputDir); err != nil {
		return fmt.Errorf("failed to save dataset: %w", err)
	}

	slog.Info("Dataset created successfully", "items", len(dataset.Items))
	fmt.Printf("\nDataset created successfully!\n")
	fmt.Printf("  Records: %d\n", len(dataset.Items))
	fmt.Printf("  Location: %s\n", outputDir)
	fmt.Printf("\nNext step: Run evaluation with:\n")
	fmt.Printf("  eval run --dataset %s\n", outputDir)

	return nil
}

func downloadImage(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image fetch returned status %d", resp.StatusCode)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}

	return nil
}
