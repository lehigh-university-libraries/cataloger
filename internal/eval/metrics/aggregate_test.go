package metrics

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAggregateEvaluationResults(t *testing.T) {
	results := []EvaluationResult{
		{
			Barcode:           "123",
			Title:             "Test Book 1",
			Author:            "Test Author 1",
			GeneratedMetadata: `{"title":"Test Book 1","author":"Test Author 1"}`,
			ProcessingTime:    5 * time.Second,
			FullComparison: &FullMARCComparison{
				Fields: map[string]FieldMatch{
					"title":   {Score: 0.9, Method: "exact"},
					"author":  {Score: 0.8, Method: "fuzzy_high"},
					"date":    {Score: 1.0, Method: "exact"},
					"isbn":    {Score: 0.7, Method: "fuzzy_medium"},
					"subject": {Score: 0.6, Method: "substring"},
				},
				OverallScore:    0.82,
				FieldsMatched:   3,
				FieldsMissing:   1,
				FieldsIncorrect: 1,
			},
		},
		{
			Barcode:           "456",
			Title:             "Test Book 2",
			Author:            "Test Author 2",
			GeneratedMetadata: `{"title":"Test Book 2","author":"Test Author 2"}`,
			ProcessingTime:    3 * time.Second,
			FullComparison: &FullMARCComparison{
				Fields: map[string]FieldMatch{
					"title":   {Score: 1.0, Method: "exact"},
					"author":  {Score: 0.9, Method: "exact"},
					"date":    {Score: 0.8, Method: "fuzzy_high"},
					"isbn":    {Score: 0.0, Method: "no_match"},
					"subject": {Score: 0.5, Method: "both_missing"},
				},
				OverallScore:    0.75,
				FieldsMatched:   2,
				FieldsMissing:   2,
				FieldsIncorrect: 1,
			},
		},
		{
			Barcode:        "789",
			Title:          "Test Book 3",
			Author:         "Test Author 3",
			Error:          "Failed to generate MARC",
			ProcessingTime: 1 * time.Second,
		},
	}

	agg := AggregateEvaluationResults(results, "ollama", "mistral-small3.2:24b")

	// Check basic stats
	if agg.TotalRecords != 3 {
		t.Errorf("Expected TotalRecords=3, got %d", agg.TotalRecords)
	}

	if agg.SuccessCount != 2 {
		t.Errorf("Expected SuccessCount=2, got %d", agg.SuccessCount)
	}

	if agg.FailureCount != 1 {
		t.Errorf("Expected FailureCount=1, got %d", agg.FailureCount)
	}

	// Check provider/model
	if agg.Provider != "ollama" {
		t.Errorf("Expected Provider=ollama, got %s", agg.Provider)
	}

	if agg.Model != "mistral-small3.2:24b" {
		t.Errorf("Expected Model=mistral-small3.2:24b, got %s", agg.Model)
	}

	// Check field stats
	if agg.TitleAccuracy.ExactMatches != 2 {
		t.Errorf("Expected TitleAccuracy.ExactMatches=2, got %d", agg.TitleAccuracy.ExactMatches)
	}

	if agg.AuthorAccuracy.ExactMatches != 1 {
		t.Errorf("Expected AuthorAccuracy.ExactMatches=1, got %d", agg.AuthorAccuracy.ExactMatches)
	}

	if agg.AuthorAccuracy.FuzzyMatches != 1 {
		t.Errorf("Expected AuthorAccuracy.FuzzyMatches=1, got %d", agg.AuthorAccuracy.FuzzyMatches)
	}

	if agg.ISBNAccuracy.NoMatches != 1 {
		t.Errorf("Expected ISBNAccuracy.NoMatches=1, got %d", agg.ISBNAccuracy.NoMatches)
	}

	if agg.SubjectAccuracy.MissingFields != 1 {
		t.Errorf("Expected SubjectAccuracy.MissingFields=1, got %d", agg.SubjectAccuracy.MissingFields)
	}

	// Check average scores
	expectedTitleAvg := (0.9 + 1.0) / 2.0
	if agg.TitleAccuracy.AverageScore != expectedTitleAvg {
		t.Errorf("Expected TitleAccuracy.AverageScore=%.2f, got %.2f",
			expectedTitleAvg, agg.TitleAccuracy.AverageScore)
	}

	// Check overall accuracy (use tolerance for floating point comparison)
	expectedOverall := (0.82 + 0.75) / 2.0
	tolerance := 0.01
	if agg.OverallAccuracy < expectedOverall-tolerance || agg.OverallAccuracy > expectedOverall+tolerance {
		t.Errorf("Expected OverallAccuracy=%.2f, got %.2f",
			expectedOverall, agg.OverallAccuracy)
	}

	// Check timing
	expectedTotal := 9 * time.Second
	if agg.TotalProcessingTime != expectedTotal {
		t.Errorf("Expected TotalProcessingTime=%s, got %s",
			expectedTotal, agg.TotalProcessingTime)
	}

	expectedAvg := 4 * time.Second // (5+3)/2 for successful ones
	if agg.AverageProcessingTime != expectedAvg {
		t.Errorf("Expected AverageProcessingTime=%s, got %s",
			expectedAvg, agg.AverageProcessingTime)
	}
}

func TestCalculateAverage(t *testing.T) {
	tests := []struct {
		name     string
		scores   []float64
		expected float64
	}{
		{
			name:     "normal scores",
			scores:   []float64{0.8, 0.9, 1.0},
			expected: 0.9,
		},
		{
			name:     "empty scores",
			scores:   []float64{},
			expected: 0.0,
		},
		{
			name:     "single score",
			scores:   []float64{0.75},
			expected: 0.75,
		},
		{
			name:     "zeros",
			scores:   []float64{0.0, 0.0, 0.0},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAverage(tt.scores)
			if result != tt.expected {
				t.Errorf("calculateAverage(%v) = %.2f, want %.2f",
					tt.scores, result, tt.expected)
			}
		})
	}
}

func TestAggregateFieldStats(t *testing.T) {
	stats := FieldStats{
		Scores: []float64{},
	}

	// Test exact match
	aggregateFieldStats(&stats, FieldMatch{Score: 1.0, Method: "exact"})
	if stats.ExactMatches != 1 {
		t.Errorf("Expected ExactMatches=1, got %d", stats.ExactMatches)
	}
	if len(stats.Scores) != 1 {
		t.Errorf("Expected 1 score, got %d", len(stats.Scores))
	}

	// Test fuzzy match
	aggregateFieldStats(&stats, FieldMatch{Score: 0.8, Method: "fuzzy_high"})
	if stats.FuzzyMatches != 1 {
		t.Errorf("Expected FuzzyMatches=1, got %d", stats.FuzzyMatches)
	}

	// Test no match
	aggregateFieldStats(&stats, FieldMatch{Score: 0.0, Method: "no_match"})
	if stats.NoMatches != 1 {
		t.Errorf("Expected NoMatches=1, got %d", stats.NoMatches)
	}

	// Test missing field
	aggregateFieldStats(&stats, FieldMatch{Score: 0.5, Method: "both_missing"})
	if stats.MissingFields != 1 {
		t.Errorf("Expected MissingFields=1, got %d", stats.MissingFields)
	}
}

func TestSaveToJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "test_results.json")

	results := []EvaluationResult{
		{
			Barcode:        "123",
			Title:          "Test Book",
			ProcessingTime: 5 * time.Second,
			FullComparison: &FullMARCComparison{
				Fields: map[string]FieldMatch{
					"title":  {Score: 0.9, Method: "exact"},
					"author": {Score: 0.8, Method: "fuzzy_high"},
				},
				OverallScore:  0.85,
				FieldsMatched: 2,
			},
		},
	}

	agg := AggregateEvaluationResults(results, "ollama", "test-model")

	err := agg.SaveToJSON(jsonPath)
	if err != nil {
		t.Fatalf("SaveToJSON failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("JSON file was not created")
	}

	// Verify file has content
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	if len(content) == 0 {
		t.Error("JSON file is empty")
	}
}

func TestSaveDetailedReport(t *testing.T) {
	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "test_report.txt")

	results := []EvaluationResult{
		{
			Barcode:        "123",
			Title:          "Test Book",
			Author:         "Test Author",
			ProcessingTime: 5 * time.Second,
			FullComparison: &FullMARCComparison{
				Fields: map[string]FieldMatch{
					"title": {
						Expected: "Test Book",
						Actual:   "Test Book",
						Score:    1.0,
						Method:   "exact",
					},
					"author": {
						Expected: "Test Author",
						Actual:   "Test Author",
						Score:    1.0,
						Method:   "exact",
					},
					"date": {
						Expected: "2020",
						Actual:   "2020",
						Score:    1.0,
						Method:   "exact",
					},
					"isbn": {
						Expected: "",
						Actual:   "",
						Score:    0.5,
						Method:   "both_missing",
					},
				},
				OverallScore:  0.95,
				FieldsMatched: 3,
			},
		},
		{
			Barcode: "456",
			Title:   "Failed Book",
			Author:  "Failed Author",
			Error:   "MARC generation failed",
		},
	}

	agg := AggregateEvaluationResults(results, "ollama", "test-model")

	err := agg.SaveDetailedReport(reportPath)
	if err != nil {
		t.Fatalf("SaveDetailedReport failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Error("Report file was not created")
	}

	// Verify file has content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read report file: %v", err)
	}

	contentStr := string(content)

	// Check for expected content
	if len(content) == 0 {
		t.Error("Report file is empty")
	}

	// Check for header
	if !contains(contentStr, "CATALOGER EVALUATION DETAILED REPORT") {
		t.Error("Report missing header")
	}

	// Check for record information
	if !contains(contentStr, "RECORD 1: 123") {
		t.Error("Report missing first record")
	}

	if !contains(contentStr, "Test Book") {
		t.Error("Report missing title")
	}

	// Check for error
	if !contains(contentStr, "ERROR: MARC generation failed") {
		t.Error("Report missing error message")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[0:len(substr)] == substr || contains(s[1:], substr))))
}
