package evalcmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/lehigh-university-libraries/cataloger/internal/evaluation"
)

func executeReport(resultsDir, format string) error {
	// Load results
	results, err := evaluation.LoadResults(resultsDir)
	if err != nil {
		return fmt.Errorf("failed to load results: %w", err)
	}

	switch format {
	case "text":
		return printTextReport(results)
	case "json":
		return printJSONReport(results)
	case "csv":
		return printCSVReport(results)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func printTextReport(results *evaluation.EvaluationResults) error {
	fmt.Println("========================================")
	fmt.Printf("MARC Cataloging Evaluation Report\n")
	fmt.Println("========================================")
	fmt.Printf("Provider: %s\n", results.Provider)
	fmt.Printf("Model:    %s\n", results.Model)
	fmt.Println()

	// Print summary
	printSummary(results.Summary)

	// Print detailed results
	fmt.Println("\nDetailed Results:")
	fmt.Println("========================================")

	for i, result := range results.Results {
		fmt.Printf("\n[%d] Record ID: %s\n", i+1, result.ID)

		if result.Error != "" {
			fmt.Printf("  ❌ Error: %s\n", result.Error)
			continue
		}

		fmt.Printf("  Overall Score: %.2f%%\n", result.ComparisonResult.OverallScore*100)

		if len(result.ComparisonResult.MissingFields) > 0 {
			fmt.Printf("  Missing Fields: %v\n", result.ComparisonResult.MissingFields)
		}

		if len(result.ComparisonResult.ExtraFields) > 0 {
			fmt.Printf("  Extra Fields: %v\n", result.ComparisonResult.ExtraFields)
		}

		fmt.Println("  Field Scores:")
		var fields []string
		for field := range result.ComparisonResult.FieldScores {
			fields = append(fields, field)
		}
		sort.Strings(fields)

		for _, field := range fields {
			score := result.ComparisonResult.FieldScores[field]
			fmt.Printf("    %s: %.2f%%\n", field, score*100)
		}

		// Show field differences with low scores
		fmt.Println("  Significant Differences:")
		for _, diff := range result.ComparisonResult.FieldDifferences {
			if diff.Similarity < 0.8 {
				fmt.Printf("    %s (%.0f%% similar):\n", diff.Tag, diff.Similarity*100)
				fmt.Printf("      Reference:  %s\n", truncate(diff.Reference, 80))
				fmt.Printf("      Generated:  %s\n", truncate(diff.Generated, 80))
			}
		}
	}

	return nil
}

func printJSONReport(results *evaluation.EvaluationResults) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func printCSVReport(results *evaluation.EvaluationResults) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	header := []string{"ID", "Overall Score", "Missing Fields", "Extra Fields", "Error"}
	// Add field score columns
	for _, fw := range evaluation.DefaultFieldWeights {
		header = append(header, fmt.Sprintf("Field_%s", fw.Tag))
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, result := range results.Results {
		row := []string{
			result.ID,
		}

		if result.Error != "" {
			row = append(row, "0", "", "", result.Error)
		} else {
			row = append(row,
				fmt.Sprintf("%.4f", result.ComparisonResult.OverallScore),
				fmt.Sprintf("%v", result.ComparisonResult.MissingFields),
				fmt.Sprintf("%v", result.ComparisonResult.ExtraFields),
				"",
			)
		}

		// Add field scores
		for _, fw := range evaluation.DefaultFieldWeights {
			if result.Error != "" {
				row = append(row, "0")
			} else if score, ok := result.ComparisonResult.FieldScores[fw.Tag]; ok {
				row = append(row, fmt.Sprintf("%.4f", score))
			} else {
				row = append(row, "0")
			}
		}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
