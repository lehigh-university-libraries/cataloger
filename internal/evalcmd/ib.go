package evalcmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/cataloging"
	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
	"github.com/lehigh-university-libraries/cataloger/internal/eval/metadata"
	"github.com/lehigh-university-libraries/cataloger/internal/eval/metrics"
	resultsutil "github.com/lehigh-university-libraries/cataloger/internal/eval/results"
)

func executeIB(datasetPath, outputJSON, outputReport string, sampleSize int, provider, model string, verbose bool) error {
	// Set up logging
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	slog.Info("Starting cataloger evaluation",
		"dataset", datasetPath,
		"sample_size", sampleSize,
		"provider", provider,
		"model", model)

	// Load dataset
	loader := dataset.NewLoader(datasetPath)
	var err error

	// Load records
	var records []dataset.InstitutionalBooksRecord

	if sampleSize > 0 {
		slog.Info("Loading sample from dataset", "limit", sampleSize)
		records, err = loader.LoadSample(sampleSize)
	} else {
		slog.Info("Loading full dataset")
		records, err = loader.Load()
	}

	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	slog.Info("Dataset loaded", "records", len(records))

	// Initialize cataloging service
	catalogService := cataloging.NewService()

	if model == "" {
		model = catalogService.GetDefaultModel(provider)
	}

	// Run evaluation
	results := make([]metrics.EvaluationResult, 0, len(records))

	for i, record := range records {
		slog.Info("Processing record", "index", i+1, "total", len(records), "barcode", record.BarcodeSource)

		result := evaluateRecord(record, catalogService, provider, model)
		if result.Error != "" {
			slog.Warn("Record processing failed", "barcode", record.BarcodeSource, "error", result.Error)
		}
		results = append(results, result)

		// Print progress
		if (i+1)%10 == 0 {
			fmt.Printf("Progress: %d/%d records processed\n", i+1, len(records))
		}
	}

	// Aggregate results
	slog.Info("Aggregating results")
	aggregated := metrics.AggregateEvaluationResults(results, provider, model)

	// Print summary
	aggregated.PrintSummary()

	// Save results
	slog.Info("Saving results", "json", outputJSON, "report", outputReport)

	if err := aggregated.SaveToJSON(outputJSON); err != nil {
		fmt.Printf("Warning: Failed to save JSON results: %v\n", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", outputJSON)
	}

	if err := aggregated.SaveDetailedReport(outputReport); err != nil {
		fmt.Printf("Warning: Failed to save detailed report: %v\n", err)
	} else {
		fmt.Printf("Detailed report saved to: %s\n", outputReport)
	}

	// Save results in YAML format (HTR-style)
	if err := resultsutil.SaveToYAML(provider, model, datasetPath, sampleSize, aggregated.Results); err != nil {
		fmt.Printf("Warning: Failed to save YAML results: %v\n", err)
	}

	slog.Info("Evaluation complete")
	return nil
}

// evaluateRecord evaluates a single dataset record
func evaluateRecord(record dataset.InstitutionalBooksRecord, service *cataloging.Service, provider, model string) metrics.EvaluationResult {
	startTime := time.Now()

	result := metrics.EvaluationResult{
		Barcode: record.BarcodeSource,
		Title:   record.TitleSource,
		Author:  record.AuthorSource,
	}

	// Get title page OCR text
	titlePageText := record.GetTitlePageText()
	if titlePageText == "" {
		result.Error = "No OCR text available for title page"
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Extract metadata from OCR using LLM
	metadataJSON, err := service.ExtractMetadataFromOCR(titlePageText, provider, model)
	if err != nil {
		result.Error = fmt.Sprintf("Metadata extraction failed: %v", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	// Clean the JSON response
	cleanedJSON := cleanJSON(metadataJSON)

	// Parse extracted metadata
	var extractedMetadata metadata.BookMetadata
	if err := json.Unmarshal([]byte(cleanedJSON), &extractedMetadata); err != nil {
		result.Error = fmt.Sprintf("Failed to parse metadata JSON: %v", err)
		result.ProcessingTime = time.Since(startTime)
		slog.Warn("Failed to parse metadata JSON", "barcode", record.BarcodeSource, "json", metadataJSON, "error", err)
		return result
	}

	// Store the extracted metadata JSON for reference
	result.GeneratedMetadata = metadataJSON
	result.ProcessingTime = time.Since(startTime)

	slog.Debug("Extracted metadata from LLM",
		"barcode", record.BarcodeSource,
		"title", extractedMetadata.Title,
		"author", extractedMetadata.Author)

	// Perform field-by-field metadata comparison with Levenshtein distance
	metadataComp := metadata.CompareMetadata(record, extractedMetadata)

	// Store comparison results
	result.FullComparison = metadataComp

	slog.Info("Comparison complete",
		"barcode", record.BarcodeSource,
		"overall_score", metadataComp.OverallScore,
		"levenshtein_total", metadataComp.LevenshteinTotal,
		"fields_matched", metadataComp.FieldsMatched,
		"fields_missing", metadataComp.FieldsMissing)

	return result
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return s
}

