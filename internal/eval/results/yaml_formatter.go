package results

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/eval/metrics"
	"gopkg.in/yaml.v3"
)

// EvalConfig represents the configuration section of the eval YAML
type EvalConfig struct {
	Provider    string   `yaml:"provider"`
	Model       string   `yaml:"model"`
	Prompt      string   `yaml:"prompt"`
	Temperature float64  `yaml:"temperature"`
	DatasetPath string   `yaml:"datasetpath"`
	SampleSize  int      `yaml:"samplesize"`
	Timestamp   string   `yaml:"timestamp"`
}

// EvalResult represents a single evaluation result
type EvalResult struct {
	Identifier       string             `yaml:"identifier"`
	Title            string             `yaml:"title"`
	Author           string             `yaml:"author,omitempty"`
	ProviderResponse string             `yaml:"providerresponse"`
	ReferenceMARC    string             `yaml:"referencemarc"`
	OverallScore     float64            `yaml:"overallscore"`
	LevenshteinTotal int                `yaml:"levenshteintotal"`
	FieldsMatched    int                `yaml:"fieldsmatched"`
	FieldsMissing    int                `yaml:"fieldsmissing"`
	FieldsIncorrect  int                `yaml:"fieldsincorrect"`
	FieldScores      map[string]float64 `yaml:"fieldscores"`
}

// EvalSpec represents the complete evaluation specification
type EvalSpec struct {
	Config  EvalConfig   `yaml:"config"`
	Results []EvalResult `yaml:"results"`
}

// SaveToYAML saves evaluation results to a YAML file in evals/ directory
func SaveToYAML(provider, model, datasetPath string, sampleSize int, results []metrics.EvaluationResult) error {
	// Create evals directory
	if err := os.MkdirAll("evals", 0755); err != nil {
		return fmt.Errorf("failed to create evals directory: %w", err)
	}

	// Generate timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")

	// Create eval spec
	spec := EvalSpec{
		Config: EvalConfig{
			Provider:    provider,
			Model:       model,
			Prompt:      "Generate MARC record from title page OCR text",
			Temperature: 0.1,
			DatasetPath: datasetPath,
			SampleSize:  sampleSize,
			Timestamp:   timestamp,
		},
		Results: make([]EvalResult, 0, len(results)),
	}

	// Convert results
	for _, r := range results {
		if r.Error != "" {
			continue // Skip failed evaluations
		}

		evalResult := EvalResult{
			Identifier:       r.Barcode,
			Title:            r.Title,
			Author:           r.Author,
			ProviderResponse: r.GeneratedMARC,
			ReferenceMARC:    r.ReferenceMARC,
		}

		// Add full comparison metrics if available
		if r.FullComparison != nil {
			evalResult.OverallScore = r.FullComparison.OverallScore
			evalResult.LevenshteinTotal = r.FullComparison.LevenshteinTotal
			evalResult.FieldsMatched = r.FullComparison.FieldsMatched
			evalResult.FieldsMissing = r.FullComparison.FieldsMissing
			evalResult.FieldsIncorrect = r.FullComparison.FieldsIncorrect

			// Extract field scores
			evalResult.FieldScores = make(map[string]float64)
			for tag, match := range r.FullComparison.Fields {
				evalResult.FieldScores[tag] = match.Score
			}
		}

		spec.Results = append(spec.Results, evalResult)
	}

	// Generate filename
	filename := fmt.Sprintf("evals/%s-%s.yaml", model, timestamp)

	// Write YAML
	data, err := yaml.Marshal(&spec)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}

	absPath, _ := filepath.Abs(filename)
	fmt.Printf("\n✅ Evaluation results saved to: %s\n", absPath)

	return nil
}
