package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DatasetItem represents a single evaluation item
type DatasetItem struct {
	ID                string `json:"id"`
	ReferenceMARC     string `json:"reference_marc"`
	CoverImagePath    string `json:"cover_image_path,omitempty"`
	TitlePagePath     string `json:"title_page_path,omitempty"`
	CopyrightPagePath string `json:"copyright_page_path,omitempty"`
}

// Dataset represents a collection of evaluation items
type Dataset struct {
	Items []DatasetItem `json:"items"`
}

// SaveDataset saves a dataset to disk
func SaveDataset(dataset *Dataset, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	datasetPath := filepath.Join(outputDir, "dataset.json")
	file, err := os.Create(datasetPath)
	if err != nil {
		return fmt.Errorf("failed to create dataset file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(dataset); err != nil {
		return fmt.Errorf("failed to encode dataset: %w", err)
	}

	return nil
}

// AppendDatasetItem appends a single item to an existing dataset file
// Creates the dataset file if it doesn't exist
func AppendDatasetItem(item DatasetItem, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	datasetPath := filepath.Join(outputDir, "dataset.json")

	// Load existing dataset or create new one
	var dataset Dataset
	if _, err := os.Stat(datasetPath); err == nil {
		existingDataset, err := LoadDataset(outputDir)
		if err != nil {
			return fmt.Errorf("failed to load existing dataset: %w", err)
		}
		dataset = *existingDataset
	} else {
		dataset = Dataset{Items: make([]DatasetItem, 0)}
	}

	// Append new item
	dataset.Items = append(dataset.Items, item)

	// Save updated dataset
	file, err := os.Create(datasetPath)
	if err != nil {
		return fmt.Errorf("failed to create dataset file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(&dataset); err != nil {
		return fmt.Errorf("failed to encode dataset: %w", err)
	}

	return nil
}

// LoadDataset loads a dataset from disk
func LoadDataset(datasetDir string) (*Dataset, error) {
	datasetPath := filepath.Join(datasetDir, "dataset.json")
	file, err := os.Open(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file: %w", err)
	}
	defer file.Close()

	var dataset Dataset
	if err := json.NewDecoder(file).Decode(&dataset); err != nil {
		return nil, fmt.Errorf("failed to decode dataset: %w", err)
	}

	return &dataset, nil
}

// EvaluationResult represents the result of evaluating a single item
type EvaluationResult struct {
	ID               string            `json:"id"`
	GeneratedMARC    string            `json:"generated_marc"`
	ComparisonResult *ComparisonResult `json:"comparison_result"`
	Error            string            `json:"error,omitempty"`
}

// EvaluationResults represents all evaluation results
type EvaluationResults struct {
	Provider string             `json:"provider"`
	Model    string             `json:"model"`
	Results  []EvaluationResult `json:"results"`
	Summary  *EvaluationSummary `json:"summary"`
}

// EvaluationSummary contains aggregate metrics
type EvaluationSummary struct {
	TotalRecords    int                `json:"total_records"`
	SuccessfulEvals int                `json:"successful_evals"`
	FailedEvals     int                `json:"failed_evals"`
	AverageScore    float64            `json:"average_score"`
	MedianScore     float64            `json:"median_score"`
	MinScore        float64            `json:"min_score"`
	MaxScore        float64            `json:"max_score"`
	FieldAccuracies map[string]float64 `json:"field_accuracies"` // Average accuracy per field
}

// SaveResults saves evaluation results to disk
func SaveResults(results *EvaluationResults, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	resultsPath := filepath.Join(outputDir, "results.json")
	file, err := os.Create(resultsPath)
	if err != nil {
		return fmt.Errorf("failed to create results file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		return fmt.Errorf("failed to encode results: %w", err)
	}

	return nil
}

// LoadResults loads evaluation results from disk
func LoadResults(resultsDir string) (*EvaluationResults, error) {
	resultsPath := filepath.Join(resultsDir, "results.json")
	file, err := os.Open(resultsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open results file: %w", err)
	}
	defer file.Close()

	var results EvaluationResults
	if err := json.NewDecoder(file).Decode(&results); err != nil {
		return nil, fmt.Errorf("failed to decode results: %w", err)
	}

	return &results, nil
}
