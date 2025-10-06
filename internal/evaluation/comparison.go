package evaluation

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/hectorcorrea/marcli/pkg/marc"
)

// ComparisonResult represents the comparison between two MARC records
type ComparisonResult struct {
	OverallScore     float64            `json:"overall_score"`     // 0.0 to 1.0
	FieldScores      map[string]float64 `json:"field_scores"`      // Per-field scores
	MissingFields    []string           `json:"missing_fields"`    // Fields in reference but not in generated
	ExtraFields      []string           `json:"extra_fields"`      // Fields in generated but not in reference
	FieldDifferences []FieldDiff        `json:"field_differences"` // Detailed field differences
}

// FieldDiff represents a difference in a specific field
type FieldDiff struct {
	Tag        string  `json:"tag"`
	Reference  string  `json:"reference"`
	Generated  string  `json:"generated"`
	Similarity float64 `json:"similarity"`
}

// FieldWeight represents the importance weight of a MARC field for scoring
type FieldWeight struct {
	Tag    string
	Weight float64
}

// Standard field weights based on cataloging importance
var DefaultFieldWeights = []FieldWeight{
	{"020", 0.05}, // ISBN - exact match expected
	{"100", 0.15}, // Main entry - personal name
	{"110", 0.15}, // Main entry - corporate name
	{"245", 0.25}, // Title statement - most important
	{"250", 0.05}, // Edition statement
	{"260", 0.15}, // Publication info (older)
	{"264", 0.15}, // Publication info (RDA)
	{"300", 0.05}, // Physical description
	{"650", 0.10}, // Subject headings
	{"700", 0.05}, // Added entry - personal name
}

// CompareRecords compares two MARC records and returns a detailed comparison
func CompareRecords(referenceMARC, generatedMARC string) (*ComparisonResult, error) {
	// Parse both MARC records from raw binary data
	refRecord, err := parseMARCRecord([]byte(referenceMARC))
	if err != nil {
		return nil, fmt.Errorf("failed to parse reference MARC: %w", err)
	}

	genRecord, err := parseMARCRecord([]byte(generatedMARC))
	if err != nil {
		return nil, fmt.Errorf("failed to parse generated MARC: %w", err)
	}

	result := &ComparisonResult{
		FieldScores:      make(map[string]float64),
		MissingFields:    []string{},
		ExtraFields:      []string{},
		FieldDifferences: []FieldDiff{},
	}

	// Create maps of fields for comparison
	refFields := groupFieldsByTag(refRecord.Fields)
	genFields := groupFieldsByTag(genRecord.Fields)

	// Compare weighted fields
	var totalWeight float64
	var weightedScore float64

	for _, fw := range DefaultFieldWeights {
		totalWeight += fw.Weight

		refFieldData := refFields[fw.Tag]
		genFieldData := genFields[fw.Tag]

		if len(refFieldData) == 0 && len(genFieldData) == 0 {
			// Neither has this field - perfect match
			weightedScore += fw.Weight
			result.FieldScores[fw.Tag] = 1.0
			continue
		}

		if len(refFieldData) == 0 {
			// Extra field in generated
			result.ExtraFields = append(result.ExtraFields, fw.Tag)
			result.FieldScores[fw.Tag] = 0.5 // Partial credit for having data
			weightedScore += fw.Weight * 0.5
			continue
		}

		if len(genFieldData) == 0 {
			// Missing field in generated
			result.MissingFields = append(result.MissingFields, fw.Tag)
			result.FieldScores[fw.Tag] = 0.0
			continue
		}

		// Compare field content
		score := compareFieldContent(fw.Tag, refFieldData, genFieldData, result)
		result.FieldScores[fw.Tag] = score
		weightedScore += fw.Weight * score
	}

	if totalWeight > 0 {
		result.OverallScore = weightedScore / totalWeight
	}

	return result, nil
}

// groupFieldsByTag groups fields by their tag
func groupFieldsByTag(fields []marc.Field) map[string][]marc.Field {
	grouped := make(map[string][]marc.Field)
	for _, field := range fields {
		grouped[field.Tag] = append(grouped[field.Tag], field)
	}
	return grouped
}

// compareFieldContent compares the content of fields with the same tag
func compareFieldContent(tag string, refFields, genFields []marc.Field, result *ComparisonResult) float64 {
	// For simplicity, compare first occurrence of each field
	// In a more sophisticated version, we'd do optimal matching
	refField := refFields[0]
	genField := genFields[0]

	refContent := fieldToString(refField)
	genContent := fieldToString(genField)

	similarity := stringSimilarity(refContent, genContent)

	result.FieldDifferences = append(result.FieldDifferences, FieldDiff{
		Tag:        tag,
		Reference:  refContent,
		Generated:  genContent,
		Similarity: similarity,
	})

	return similarity
}

// fieldToString converts a MARC field to a comparable string
func fieldToString(field marc.Field) string {
	if field.IsControlField() {
		return field.Value
	}

	var parts []string
	for _, subfield := range field.SubFields {
		parts = append(parts, fmt.Sprintf("$%s %s", subfield.Code, subfield.Value))
	}
	return strings.Join(parts, " ")
}

// stringSimilarity calculates similarity between two strings using Levenshtein distance
// Returns a score from 0.0 (completely different) to 1.0 (identical)
func stringSimilarity(s1, s2 string) float64 {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))

	if s1 == s2 {
		return 1.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - (float64(distance) / maxLen)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// parseMARCRecord parses a MARC record from raw bytes
func parseMARCRecord(data []byte) (*marc.Record, error) {
	// Write to temporary file since marcli requires *os.File
	tmpFile, err := os.CreateTemp("", "marc-*.mrc")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	// Seek to beginning
	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	// Create MarcFile
	marcFile := marc.NewMarcFile(tmpFile)

	// Read first record
	if !marcFile.Scan() {
		if err := marcFile.Err(); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		return nil, fmt.Errorf("no MARC record found")
	}

	record, err := marcFile.Record()
	if err != nil {
		return nil, fmt.Errorf("record parse error: %w", err)
	}

	return &record, nil
}
