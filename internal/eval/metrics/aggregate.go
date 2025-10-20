package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// EvaluationResult represents the results for a single book evaluation
type EvaluationResult struct {
	Barcode        string
	Title          string
	Author         string
	GeneratedMARC  string
	Comparison     *MARCComparison
	ProcessingTime time.Duration
	Error          string // If generation failed
}

// AggregateResults represents aggregated evaluation metrics
type AggregateResults struct {
	TotalRecords int
	SuccessCount int
	FailureCount int

	// Field-level statistics
	TitleAccuracy   FieldStats
	AuthorAccuracy  FieldStats
	DateAccuracy    FieldStats
	ISBNAccuracy    FieldStats
	SubjectAccuracy FieldStats

	// Overall
	OverallAccuracy float64

	// Timing
	AverageProcessingTime time.Duration
	TotalProcessingTime   time.Duration

	// Detailed results
	Results []EvaluationResult

	// Metadata
	EvaluationDate time.Time
	Provider       string
	Model          string
	SampleSize     int
}

// FieldStats contains statistics for a specific MARC field
type FieldStats struct {
	ExactMatches  int
	FuzzyMatches  int
	NoMatches     int
	MissingFields int
	AverageScore  float64
	Scores        []float64
}

// AggregateEvaluationResults aggregates multiple evaluation results
func AggregateEvaluationResults(results []EvaluationResult, provider, model string) *AggregateResults {
	agg := &AggregateResults{
		TotalRecords:   len(results),
		Results:        results,
		EvaluationDate: time.Now(),
		Provider:       provider,
		Model:          model,
		SampleSize:     len(results),
	}

	// Initialize field stats
	agg.TitleAccuracy = FieldStats{Scores: []float64{}}
	agg.AuthorAccuracy = FieldStats{Scores: []float64{}}
	agg.DateAccuracy = FieldStats{Scores: []float64{}}
	agg.ISBNAccuracy = FieldStats{Scores: []float64{}}
	agg.SubjectAccuracy = FieldStats{Scores: []float64{}}

	totalOverallScore := 0.0
	var totalDuration time.Duration
	var successDuration time.Duration

	for _, result := range results {
		totalDuration += result.ProcessingTime

		if result.Error != "" {
			agg.FailureCount++
			continue
		}

		agg.SuccessCount++
		successDuration += result.ProcessingTime

		if result.Comparison == nil {
			continue
		}

		// Aggregate title stats
		aggregateFieldStats(&agg.TitleAccuracy, result.Comparison.TitleMatch)

		// Aggregate author stats
		aggregateFieldStats(&agg.AuthorAccuracy, result.Comparison.AuthorMatch)

		// Aggregate date stats
		aggregateFieldStats(&agg.DateAccuracy, result.Comparison.DateMatch)

		// Aggregate ISBN stats
		aggregateFieldStats(&agg.ISBNAccuracy, result.Comparison.ISBNMatch)

		// Aggregate subject stats
		aggregateFieldStats(&agg.SubjectAccuracy, result.Comparison.SubjectMatch)

		// Overall score
		totalOverallScore += result.Comparison.OverallScore
	}

	// Calculate averages
	if agg.SuccessCount > 0 {
		agg.TitleAccuracy.AverageScore = calculateAverage(agg.TitleAccuracy.Scores)
		agg.AuthorAccuracy.AverageScore = calculateAverage(agg.AuthorAccuracy.Scores)
		agg.DateAccuracy.AverageScore = calculateAverage(agg.DateAccuracy.Scores)
		agg.ISBNAccuracy.AverageScore = calculateAverage(agg.ISBNAccuracy.Scores)
		agg.SubjectAccuracy.AverageScore = calculateAverage(agg.SubjectAccuracy.Scores)
		agg.OverallAccuracy = totalOverallScore / float64(agg.SuccessCount)
		agg.AverageProcessingTime = successDuration / time.Duration(agg.SuccessCount)
	}

	agg.TotalProcessingTime = totalDuration

	return agg
}

// aggregateFieldStats updates field statistics
func aggregateFieldStats(stats *FieldStats, match FieldMatch) {
	stats.Scores = append(stats.Scores, match.Score)

	switch match.Method {
	case "exact":
		stats.ExactMatches++
	case "fuzzy_high", "fuzzy_medium", "substring":
		stats.FuzzyMatches++
	case "no_match":
		stats.NoMatches++
	case "actual_missing", "expected_missing", "both_missing":
		stats.MissingFields++
	}
}

// calculateAverage calculates the average of a slice of scores
func calculateAverage(scores []float64) float64 {
	if len(scores) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, score := range scores {
		sum += score
	}

	return sum / float64(len(scores))
}

// PrintSummary prints a human-readable summary of the evaluation
func (a *AggregateResults) PrintSummary() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("CATALOGER EVALUATION SUMMARY")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Evaluation Date: %s\n", a.EvaluationDate.Format("2006-01-02 15:04:05"))
	fmt.Printf("Provider: %s\n", a.Provider)
	fmt.Printf("Model: %s\n", a.Model)
	fmt.Printf("Sample Size: %d records\n", a.SampleSize)
	fmt.Println()

	fmt.Println("PROCESSING STATISTICS")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Total Records: %d\n", a.TotalRecords)
	fmt.Printf("Successful: %d (%.1f%%)\n", a.SuccessCount, float64(a.SuccessCount)/float64(a.TotalRecords)*100)
	fmt.Printf("Failed: %d (%.1f%%)\n", a.FailureCount, float64(a.FailureCount)/float64(a.TotalRecords)*100)
	fmt.Printf("Average Processing Time: %s\n", a.AverageProcessingTime)
	fmt.Printf("Total Processing Time: %s\n", a.TotalProcessingTime)
	fmt.Println()

	fmt.Println("FIELD-LEVEL ACCURACY")
	fmt.Println(strings.Repeat("-", 70))
	printFieldStats("Title", a.TitleAccuracy)
	printFieldStats("Author", a.AuthorAccuracy)
	printFieldStats("Date", a.DateAccuracy)
	printFieldStats("ISBN", a.ISBNAccuracy)
	printFieldStats("Subject", a.SubjectAccuracy)
	fmt.Println()

	fmt.Println("OVERALL SCORE")
	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("Overall Accuracy: %.2f%% (%.3f)\n", a.OverallAccuracy*100, a.OverallAccuracy)
	fmt.Println(strings.Repeat("=", 70))
}

// printFieldStats prints statistics for a single field
func printFieldStats(fieldName string, stats FieldStats) {
	fmt.Printf("\n%s:\n", fieldName)
	fmt.Printf("  Average Score: %.2f%% (%.3f)\n", stats.AverageScore*100, stats.AverageScore)
	fmt.Printf("  Exact Matches: %d\n", stats.ExactMatches)
	fmt.Printf("  Fuzzy Matches: %d\n", stats.FuzzyMatches)
	fmt.Printf("  No Matches: %d\n", stats.NoMatches)
	fmt.Printf("  Missing Fields: %d\n", stats.MissingFields)
}

// SaveToJSON saves the aggregate results to a JSON file
func (a *AggregateResults) SaveToJSON(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(a); err != nil {
		return fmt.Errorf("failed to encode results to JSON: %w", err)
	}

	return nil
}

// SaveDetailedReport saves a detailed report with individual results
func (a *AggregateResults) SaveDetailedReport(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "CATALOGER EVALUATION DETAILED REPORT\n")
	fmt.Fprintf(file, "Generated: %s\n", a.EvaluationDate.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Provider: %s, Model: %s\n", a.Provider, a.Model)
	separator := strings.Repeat("=", 80)
	fmt.Fprintf(file, "%s\n\n", separator)

	// Write individual results
	dash := strings.Repeat("-", 80)
	for i, result := range a.Results {
		fmt.Fprintf(file, "RECORD %d: %s\n", i+1, result.Barcode)
		fmt.Fprintf(file, "%s\n", dash)
		fmt.Fprintf(file, "Title: %s\n", result.Title)
		fmt.Fprintf(file, "Author: %s\n", result.Author)
		fmt.Fprintf(file, "Processing Time: %s\n", result.ProcessingTime)

		if result.Error != "" {
			fmt.Fprintf(file, "ERROR: %s\n", result.Error)
		} else if result.Comparison != nil {
			fmt.Fprintf(file, "\nField Comparisons:\n")
			fmt.Fprintf(file, "  Title:   %.2f (%s) - Expected: %s, Actual: %s\n",
				result.Comparison.TitleMatch.Score,
				result.Comparison.TitleMatch.Method,
				result.Comparison.TitleMatch.Expected,
				result.Comparison.TitleMatch.Actual)

			fmt.Fprintf(file, "  Author:  %.2f (%s) - Expected: %s, Actual: %s\n",
				result.Comparison.AuthorMatch.Score,
				result.Comparison.AuthorMatch.Method,
				result.Comparison.AuthorMatch.Expected,
				result.Comparison.AuthorMatch.Actual)

			fmt.Fprintf(file, "  Date:    %.2f (%s) - Expected: %s, Actual: %s\n",
				result.Comparison.DateMatch.Score,
				result.Comparison.DateMatch.Method,
				result.Comparison.DateMatch.Expected,
				result.Comparison.DateMatch.Actual)

			fmt.Fprintf(file, "  ISBN:    %.2f (%s) - Expected: %s, Actual: %s\n",
				result.Comparison.ISBNMatch.Score,
				result.Comparison.ISBNMatch.Method,
				result.Comparison.ISBNMatch.Expected,
				result.Comparison.ISBNMatch.Actual)

			fmt.Fprintf(file, "\nOverall Score: %.2f%%\n", result.Comparison.OverallScore*100)
		}

		fmt.Fprintf(file, "\n%s\n\n", separator)
	}

	return nil
}
