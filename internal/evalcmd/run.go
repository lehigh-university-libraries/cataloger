package evalcmd

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"

	"github.com/lehigh-university-libraries/cataloger/internal/cataloging"
	"github.com/lehigh-university-libraries/cataloger/internal/evaluation"
)

func executeRun(datasetDir, provider, model, outputDir string, concurrency int) error {
	slog.Info("Starting evaluation run", "dataset", datasetDir, "provider", provider, "model", model)

	// Load dataset
	slog.Info("Loading dataset...")
	dataset, err := evaluation.LoadDataset(datasetDir)
	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	slog.Info("Dataset loaded", "items", len(dataset.Items))

	// Set defaults for model if not specified
	if model == "" {
		switch provider {
		case "ollama":
			model = os.Getenv("OLLAMA_MODEL")
			if model == "" {
				model = "mistral-small3.2:24b"
			}
		case "openai":
			model = os.Getenv("OPENAI_MODEL")
			if model == "" {
				model = "gpt-4o"
			}
		}
	}

	// Create cataloging service
	catalogService := cataloging.NewService()

	// Create results
	results := &evaluation.EvaluationResults{
		Provider: provider,
		Model:    model,
		Results:  make([]evaluation.EvaluationResult, 0, len(dataset.Items)),
	}

	// Process items with concurrency control
	slog.Info("Processing items", "concurrency", concurrency)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	resultsChan := make(chan evaluation.EvaluationResult, len(dataset.Items))

	for i, item := range dataset.Items {
		wg.Add(1)
		go func(idx int, item evaluation.DatasetItem) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			slog.Info("Processing item", "id", item.ID, "progress", fmt.Sprintf("%d/%d", idx+1, len(dataset.Items)))

			result := processItem(item, catalogService, provider, model)
			resultsChan <- result
		}(i, item)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results.Results = append(results.Results, result)
	}

	// Calculate summary statistics
	slog.Info("Calculating summary statistics...")
	results.Summary = calculateSummary(results.Results)

	// Save results
	slog.Info("Saving results", "output", outputDir)
	if err := evaluation.SaveResults(results, outputDir); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Print summary
	printSummary(results.Summary)

	fmt.Printf("\nResults saved to: %s\n", outputDir)
	fmt.Printf("\nGenerate detailed report with:\n")
	fmt.Printf("  eval report --results %s\n", outputDir)

	return nil
}

func processItem(item evaluation.DatasetItem, catalogService *cataloging.Service, provider, model string) evaluation.EvaluationResult {
	result := evaluation.EvaluationResult{
		ID: item.ID,
	}

	// Determine which image to use (prefer title page, fallback to cover)
	imagePath := item.TitlePagePath
	if imagePath == "" {
		imagePath = item.CoverImagePath
	}

	if imagePath == "" {
		result.Error = "no image available for cataloging"
		return result
	}

	// Generate MARC record
	generatedMARC, err := catalogService.GenerateMARCFromImage(imagePath, provider, model)
	if err != nil {
		result.Error = fmt.Sprintf("failed to generate MARC: %v", err)
		return result
	}

	result.GeneratedMARC = generatedMARC

	// Compare with reference MARC
	comparison, err := evaluation.CompareRecords(item.ReferenceMARC, generatedMARC)
	if err != nil {
		result.Error = fmt.Sprintf("failed to compare records: %v", err)
		return result
	}

	result.ComparisonResult = comparison

	return result
}

func calculateSummary(results []evaluation.EvaluationResult) *evaluation.EvaluationSummary {
	summary := &evaluation.EvaluationSummary{
		TotalRecords:    len(results),
		FieldAccuracies: make(map[string]float64),
	}

	var scores []float64
	fieldScores := make(map[string][]float64)

	for _, result := range results {
		if result.Error != "" {
			summary.FailedEvals++
			continue
		}

		summary.SuccessfulEvals++
		scores = append(scores, result.ComparisonResult.OverallScore)

		// Collect field scores
		for field, score := range result.ComparisonResult.FieldScores {
			fieldScores[field] = append(fieldScores[field], score)
		}
	}

	if len(scores) > 0 {
		// Calculate average score
		var total float64
		for _, score := range scores {
			total += score
		}
		summary.AverageScore = total / float64(len(scores))

		// Calculate median score
		sort.Float64s(scores)
		mid := len(scores) / 2
		if len(scores)%2 == 0 {
			summary.MedianScore = (scores[mid-1] + scores[mid]) / 2
		} else {
			summary.MedianScore = scores[mid]
		}

		// Min and max scores
		summary.MinScore = scores[0]
		summary.MaxScore = scores[len(scores)-1]

		// Calculate average field accuracies
		for field, scores := range fieldScores {
			var total float64
			for _, score := range scores {
				total += score
			}
			summary.FieldAccuracies[field] = total / float64(len(scores))
		}
	}

	return summary
}

func printSummary(summary *evaluation.EvaluationSummary) {
	fmt.Println("\n========================================")
	fmt.Println("Evaluation Summary")
	fmt.Println("========================================")
	fmt.Printf("Total Records:      %d\n", summary.TotalRecords)
	fmt.Printf("Successful Evals:   %d\n", summary.SuccessfulEvals)
	fmt.Printf("Failed Evals:       %d\n", summary.FailedEvals)
	fmt.Println()
	fmt.Printf("Average Score:      %.2f%%\n", summary.AverageScore*100)
	fmt.Printf("Median Score:       %.2f%%\n", summary.MedianScore*100)
	fmt.Printf("Min Score:          %.2f%%\n", summary.MinScore*100)
	fmt.Printf("Max Score:          %.2f%%\n", summary.MaxScore*100)
	fmt.Println()
	fmt.Println("Field Accuracies:")

	// Sort fields for consistent output
	var fields []string
	for field := range summary.FieldAccuracies {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	for _, field := range fields {
		fmt.Printf("  %s: %.2f%%\n", field, summary.FieldAccuracies[field]*100)
	}
	fmt.Println("========================================")
}
