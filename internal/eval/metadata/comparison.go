package metadata

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
)

// CompareMetadata performs field-by-field comparison using Levenshtein distance
func CompareMetadata(reference dataset.InstitutionalBooksRecord, extracted BookMetadata) *MetadataComparison {
	comparison := &MetadataComparison{
		Fields: make(map[string]FieldComparison),
	}

	totalScore := 0.0
	fieldCount := 0
	totalLevenshtein := 0

	// Compare Title
	titleComp := compareField("title", reference.TitleSource, extracted.Title)
	comparison.Fields["title"] = titleComp
	totalScore += titleComp.Score
	totalLevenshtein += titleComp.Distance
	fieldCount++
	if titleComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if titleComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if extracted.Title == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Compare Author
	authorComp := compareField("author", reference.AuthorSource, extracted.Author)
	comparison.Fields["author"] = authorComp
	totalScore += authorComp.Score
	totalLevenshtein += authorComp.Distance
	fieldCount++
	if authorComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if authorComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if extracted.Author == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Compare Publication Date
	dateComp := compareField("date", reference.Date1Source, extracted.PublicationDate)
	comparison.Fields["date"] = dateComp
	totalScore += dateComp.Score
	totalLevenshtein += dateComp.Distance
	fieldCount++
	if dateComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if dateComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if extracted.PublicationDate == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Compare ISBN
	isbnRef := ""
	if len(reference.IdentifiersSource.ISBN) > 0 {
		isbnRef = reference.IdentifiersSource.ISBN[0]
	}
	isbnExt := ""
	if len(extracted.ISBN) > 0 {
		isbnExt = extracted.ISBN[0]
	}
	isbnComp := compareField("isbn", isbnRef, isbnExt)
	comparison.Fields["isbn"] = isbnComp
	totalScore += isbnComp.Score
	totalLevenshtein += isbnComp.Distance
	fieldCount++
	if isbnComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if isbnComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if isbnExt == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Compare Language
	langComp := compareField("language", reference.LanguageSource, extracted.Language)
	comparison.Fields["language"] = langComp
	totalScore += langComp.Score
	totalLevenshtein += langComp.Distance
	fieldCount++
	if langComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if langComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if extracted.Language == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Compare Subject
	subjectComp := compareField("subject", reference.TopicOrSubjectSource, extracted.Subject)
	comparison.Fields["subject"] = subjectComp
	totalScore += subjectComp.Score
	totalLevenshtein += subjectComp.Distance
	fieldCount++
	if subjectComp.Score > 0.8 {
		comparison.FieldsMatched++
	} else if subjectComp.Score > 0.5 {
		comparison.FieldsIncorrect++
	} else if extracted.Subject == "" {
		comparison.FieldsMissing++
	} else {
		comparison.FieldsIncorrect++
	}

	// Calculate overall score
	if fieldCount > 0 {
		comparison.OverallScore = totalScore / float64(fieldCount)
	}
	comparison.LevenshteinTotal = totalLevenshtein

	return comparison
}

// compareField compares a single field using Levenshtein distance
func compareField(fieldName, expected, actual string) FieldComparison {
	comp := FieldComparison{
		FieldName: fieldName,
		Expected:  expected,
		Actual:    actual,
	}

	// Normalize for comparison
	expNorm := normalizeText(expected)
	actNorm := normalizeText(actual)

	// Handle empty fields
	if expNorm == "" && actNorm == "" {
		comp.Score = 0.5
		comp.Distance = 0
		comp.Match = "both_empty"
		comp.Notes = "Both fields are empty"
		return comp
	}

	if expNorm == "" {
		comp.Score = 0.0
		comp.Distance = len(actNorm)
		comp.Match = "no_reference"
		comp.Notes = "No reference value (ground truth missing)"
		return comp
	}

	if actNorm == "" {
		comp.Score = 0.0
		comp.Distance = len(expNorm)
		comp.Match = "missing"
		comp.Notes = "Field missing from extracted metadata"
		return comp
	}

	// Calculate Levenshtein distance
	distance := levenshteinDistance(expNorm, actNorm)
	comp.Distance = distance

	// Exact match
	if expNorm == actNorm {
		comp.Score = 1.0
		comp.Match = "exact"
		comp.Notes = "Exact match"
		return comp
	}

	// Calculate similarity score
	maxLen := max(len(expNorm), len(actNorm))
	similarity := 1.0 - (float64(distance) / float64(maxLen))
	comp.Score = similarity

	// Classify match quality
	if similarity > 0.9 {
		comp.Match = "fuzzy_high"
		comp.Notes = fmt.Sprintf("Very high similarity (%.1f%%), Levenshtein: %d", similarity*100, distance)
	} else if similarity > 0.7 {
		comp.Match = "fuzzy_medium"
		comp.Notes = fmt.Sprintf("Medium similarity (%.1f%%), Levenshtein: %d", similarity*100, distance)
	} else if similarity > 0.5 {
		comp.Match = "fuzzy_low"
		comp.Notes = fmt.Sprintf("Low similarity (%.1f%%), Levenshtein: %d", similarity*100, distance)
	} else {
		comp.Match = "no_match"
		comp.Notes = fmt.Sprintf("Poor match (%.1f%%), Levenshtein: %d", similarity*100, distance)
	}

	return comp
}

// normalizeText normalizes text for comparison
func normalizeText(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	// Remove common punctuation for comparison
	re := regexp.MustCompile(`[^\w\s]`)
	text = re.ReplaceAllString(text, "")

	return strings.TrimSpace(text)
}

// levenshteinDistance calculates the Levenshtein distance between two strings
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	rows := len(s1) + 1
	cols := len(s2) + 1
	matrix := make([][]int, rows)
	for i := range matrix {
		matrix[i] = make([]int, cols)
	}

	// Initialize first row and column
	for i := 0; i < rows; i++ {
		matrix[i][0] = i
	}
	for j := 0; j < cols; j++ {
		matrix[0][j] = j
	}

	// Fill the matrix
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			deletion := matrix[i-1][j] + 1
			insertion := matrix[i][j-1] + 1
			substitution := matrix[i-1][j-1] + cost

			matrix[i][j] = min(deletion, min(insertion, substitution))
		}
	}

	return matrix[rows-1][cols-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
