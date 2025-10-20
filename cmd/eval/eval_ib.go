package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/cataloging"
	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
	"github.com/lehigh-university-libraries/cataloger/internal/eval/metrics"
)

func evalIBCmd() {
	fs := flag.NewFlagSet("eval-ib", flag.ExitOnError)

	// Define flags
	datasetPath := fs.String("dataset", "./institutional-books-1.0/data/train-00000-of-09831.parquet", "Path to Institutional Books parquet file")
	outputJSON := fs.String("output-json", "eval_results.json", "Path to output JSON results file")
	outputReport := fs.String("output-report", "eval_report.txt", "Path to output detailed report file")
	sampleSize := fs.Int("sample", 10, "Number of records to evaluate (use -1 for all)")
	provider := fs.String("provider", "ollama", "LLM provider (ollama or openai)")
	model := fs.String("model", "", "Model name (defaults to provider's default)")
	verbose := fs.Bool("verbose", false, "Verbose logging")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Printf("Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Set up logging
	logLevel := slog.LevelInfo
	if *verbose {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// Check if dataset file exists
	if _, err := os.Stat(*datasetPath); os.IsNotExist(err) {
		fmt.Printf("Dataset file not found: %s\n\nPlease clone the dataset first:\n  git clone https://huggingface.co/datasets/instdin/institutional-books-1.0\n", *datasetPath)
		os.Exit(1)
	}

	slog.Info("Starting cataloger evaluation",
		"dataset", *datasetPath,
		"sample_size", *sampleSize,
		"provider", *provider,
		"model", *model)

	// Load dataset
	loader := dataset.NewLoader(*datasetPath)
	var err error

	// Load records
	var records []dataset.InstitutionalBooksRecord

	if *sampleSize > 0 {
		slog.Info("Loading sample from dataset", "limit", *sampleSize)
		records, err = loader.LoadSample(*sampleSize)
	} else {
		slog.Info("Loading full dataset")
		records, err = loader.Load()
	}

	if err != nil {
		fmt.Printf("Failed to load dataset: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Dataset loaded", "records", len(records))

	// Initialize cataloging service
	catalogService := cataloging.NewService()

	// Run evaluation
	results := make([]metrics.EvaluationResult, 0, len(records))

	for i, record := range records {
		slog.Info("Processing record", "index", i+1, "total", len(records), "barcode", record.BarcodeSource)

		result := evaluateRecord(record, catalogService, *provider, *model)
		results = append(results, result)

		// Print progress
		if (i+1)%10 == 0 {
			fmt.Printf("Progress: %d/%d records processed\n", i+1, len(records))
		}
	}

	// Aggregate results
	slog.Info("Aggregating results")
	aggregated := metrics.AggregateEvaluationResults(results, *provider, *model)

	// Print summary
	aggregated.PrintSummary()

	// Save results
	slog.Info("Saving results", "json", *outputJSON, "report", *outputReport)

	if err := aggregated.SaveToJSON(*outputJSON); err != nil {
		fmt.Printf("Warning: Failed to save JSON results: %v\n", err)
	} else {
		fmt.Printf("\nResults saved to: %s\n", *outputJSON)
	}

	if err := aggregated.SaveDetailedReport(*outputReport); err != nil {
		fmt.Printf("Warning: Failed to save detailed report: %v\n", err)
	} else {
		fmt.Printf("Detailed report saved to: %s\n", *outputReport)
	}

	slog.Info("Evaluation complete")
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

	// Generate MARC record from OCR
	generatedMARC, err := service.GenerateMARCFromOCR(titlePageText, provider, model)
	if err != nil {
		result.Error = fmt.Sprintf("MARC generation failed: %v", err)
		result.ProcessingTime = time.Since(startTime)
		return result
	}

	result.GeneratedMARC = generatedMARC
	result.ProcessingTime = time.Since(startTime)

	// Compare against ground truth
	comparison := metrics.CompareMARCFields(
		generatedMARC,
		record.TitleSource,
		record.AuthorSource,
		record.GetPrimaryDate(),
		record.GetISBN(),
		record.TopicOrSubjectSource,
	)

	result.Comparison = comparison

	return result
}
