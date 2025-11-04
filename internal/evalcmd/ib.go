package evalcmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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

	// Run evaluation
	results := make([]metrics.EvaluationResult, 0, len(records))

	for i, record := range records {
		slog.Info("Processing record", "index", i+1, "total", len(records), "barcode", record.BarcodeSource)

		result := evaluateRecord(record, catalogService, provider, model)
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

	// Parse extracted metadata
	var extractedMetadata metadata.BookMetadata
	if err := json.Unmarshal([]byte(metadataJSON), &extractedMetadata); err != nil {
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

	// Store comparison results (reusing existing fields for compatibility)
	result.FullComparison = &metrics.FullMARCComparison{
		Fields:           convertMetadataToMARCFields(metadataComp.Fields),
		OverallScore:     metadataComp.OverallScore,
		FieldsMatched:    metadataComp.FieldsMatched,
		FieldsMissing:    metadataComp.FieldsMissing,
		FieldsIncorrect:  metadataComp.FieldsIncorrect,
		LevenshteinTotal: metadataComp.LevenshteinTotal,
		ReferenceMARC:    fmt.Sprintf("Ground truth from Institutional Books record %s", record.BarcodeSource),
		GeneratedMARC:    metadataJSON,
	}

	slog.Info("Comparison complete",
		"barcode", record.BarcodeSource,
		"overall_score", metadataComp.OverallScore,
		"levenshtein_total", metadataComp.LevenshteinTotal,
		"fields_matched", metadataComp.FieldsMatched,
		"fields_missing", metadataComp.FieldsMissing)

	return result
}

// convertMetadataToMARCFields converts metadata field comparisons to MARC field format for compatibility
func convertMetadataToMARCFields(fields map[string]metadata.FieldComparison) map[string]metrics.FieldMatch {
	marcFields := make(map[string]metrics.FieldMatch)
	for fieldName, comp := range fields {
		marcFields[fieldName] = metrics.FieldMatch{
			Expected: comp.Expected,
			Actual:   comp.Actual,
			Score:    comp.Score,
			Method:   comp.Match,
			Notes:    comp.Notes,
		}
	}
	return marcFields
}
