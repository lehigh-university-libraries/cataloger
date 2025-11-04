package metrics

import (
	"fmt"
	"regexp"
	"strings"
)

// MARCComparison represents field-level comparison results
type MARCComparison struct {
	TitleMatch   FieldMatch
	AuthorMatch  FieldMatch
	DateMatch    FieldMatch
	ISBNMatch    FieldMatch
	SubjectMatch FieldMatch

	// Overall scores
	FieldLevelScores map[string]float64
	OverallScore     float64
}

// FullMARCComparison represents comprehensive MARC-to-MARC comparison
type FullMARCComparison struct {
	Fields           map[string]FieldMatch // Tag -> comparison
	OverallScore     float64
	FieldsMatched    int
	FieldsMissing    int
	FieldsIncorrect  int
	LevenshteinTotal int    // Total Levenshtein distance across all fields
	ReferenceMARC    string // Ground truth MARC
	GeneratedMARC    string // LLM-generated MARC
}

// FieldMatch represents the comparison result for a single field
type FieldMatch struct {
	Expected string
	Actual   string
	Score    float64 // 0.0 to 1.0
	Method   string  // "exact", "fuzzy", "partial", "missing"
	Notes    string
}

// MARCParser extracts fields from generated MARC records
type MARCParser struct{}

// NewMARCParser creates a new MARC parser
func NewMARCParser() *MARCParser {
	return &MARCParser{}
}

// ExtractTitle extracts the title from a MARC record (field 245)
func (p *MARCParser) ExtractTitle(marcRecord string) string {
	// Look for 245 field (with or without = prefix)
	re := regexp.MustCompile(`(?m)^=?245\s+.*?\$a\s*([^$\n]+)`)
	matches := re.FindStringSubmatch(marcRecord)
	if len(matches) > 1 {
		title := strings.TrimSpace(matches[1])
		// Remove trailing punctuation like / or :
		title = strings.TrimRight(title, " /:")
		return title
	}
	return ""
}

// ExtractAuthor extracts the author from a MARC record (field 100)
func (p *MARCParser) ExtractAuthor(marcRecord string) string {
	// Look for 100 field (with or without = prefix)
	re := regexp.MustCompile(`(?m)^=?100\s+.*?\$a\s*([^$\n]+)`)
	matches := re.FindStringSubmatch(marcRecord)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// ExtractDate extracts the publication date from a MARC record (field 260/264)
func (p *MARCParser) ExtractDate(marcRecord string) string {
	// Look for 260 or 264 field with $c (date) - with or without = prefix
	re := regexp.MustCompile(`(?m)^=?(260|264)\s+.*?\$c\s*([^$\n]+)`)
	matches := re.FindStringSubmatch(marcRecord)
	if len(matches) > 2 {
		return strings.TrimSpace(matches[2])
	}
	return ""
}

// ExtractISBN extracts ISBN from a MARC record (field 020)
func (p *MARCParser) ExtractISBN(marcRecord string) string {
	// Look for 020 field with $a (ISBN) - with or without = prefix
	re := regexp.MustCompile(`(?m)^=?020\s+.*?\$a\s*([^$\n]+)`)
	matches := re.FindStringSubmatch(marcRecord)
	if len(matches) > 1 {
		isbn := strings.TrimSpace(matches[1])
		// Clean ISBN (remove hyphens, spaces)
		isbn = regexp.MustCompile(`[^0-9Xx]`).ReplaceAllString(isbn, "")
		return isbn
	}
	return ""
}

// ExtractSubject extracts subject headings from a MARC record (fields 6XX)
func (p *MARCParser) ExtractSubject(marcRecord string) string {
	// Look for 6XX fields - with or without = prefix
	re := regexp.MustCompile(`(?m)^=?6[0-9]{2}\s+.*?\$a\s*([^$\n]+)`)
	matches := re.FindAllStringSubmatch(marcRecord, -1)

	var subjects []string
	for _, match := range matches {
		if len(match) > 1 {
			subjects = append(subjects, strings.TrimSpace(match[1]))
		}
	}

	return strings.Join(subjects, "; ")
}

// CompareMARCFields compares generated MARC against ground truth
func CompareMARCFields(generatedMARC, expectedTitle, expectedAuthor, expectedDate, expectedISBN, expectedSubject string) *MARCComparison {
	parser := NewMARCParser()

	comparison := &MARCComparison{
		FieldLevelScores: make(map[string]float64),
	}

	// Extract fields from generated MARC
	actualTitle := parser.ExtractTitle(generatedMARC)
	actualAuthor := parser.ExtractAuthor(generatedMARC)
	actualDate := parser.ExtractDate(generatedMARC)
	actualISBN := parser.ExtractISBN(generatedMARC)
	actualSubject := parser.ExtractSubject(generatedMARC)

	// Compare Title
	comparison.TitleMatch = compareField(expectedTitle, actualTitle, "title")
	comparison.FieldLevelScores["title"] = comparison.TitleMatch.Score

	// Compare Author
	comparison.AuthorMatch = compareField(expectedAuthor, actualAuthor, "author")
	comparison.FieldLevelScores["author"] = comparison.AuthorMatch.Score

	// Compare Date
	comparison.DateMatch = compareField(expectedDate, actualDate, "date")
	comparison.FieldLevelScores["date"] = comparison.DateMatch.Score

	// Compare ISBN
	comparison.ISBNMatch = compareField(expectedISBN, actualISBN, "isbn")
	comparison.FieldLevelScores["isbn"] = comparison.ISBNMatch.Score

	// Compare Subject
	comparison.SubjectMatch = compareField(expectedSubject, actualSubject, "subject")
	comparison.FieldLevelScores["subject"] = comparison.SubjectMatch.Score

	// Calculate overall score (weighted average)
	// Title and Author are most important
	weights := map[string]float64{
		"title":   0.30,
		"author":  0.30,
		"date":    0.20,
		"isbn":    0.10,
		"subject": 0.10,
	}

	totalScore := 0.0
	for field, weight := range weights {
		totalScore += comparison.FieldLevelScores[field] * weight
	}
	comparison.OverallScore = totalScore

	return comparison
}

// compareField performs detailed field comparison with fuzzy matching
func compareField(expected, actual, fieldName string) FieldMatch {
	match := FieldMatch{
		Expected: expected,
		Actual:   actual,
	}

	// Normalize for comparison
	expNorm := normalizeForComparison(expected)
	actNorm := normalizeForComparison(actual)

	// Handle missing fields
	if expected == "" && actual == "" {
		match.Score = 0.5
		match.Method = "both_missing"
		match.Notes = "Both fields are empty"
		return match
	}

	if expected == "" {
		match.Score = 0.0
		match.Method = "expected_missing"
		match.Notes = "Expected value is empty (no ground truth)"
		return match
	}

	if actual == "" {
		match.Score = 0.0
		match.Method = "actual_missing"
		match.Notes = "Generated MARC missing this field"
		return match
	}

	// Exact match
	if expNorm == actNorm {
		match.Score = 1.0
		match.Method = "exact"
		match.Notes = "Exact match"
		return match
	}

	// Fuzzy match - check for substring containment
	if strings.Contains(actNorm, expNorm) || strings.Contains(expNorm, actNorm) {
		match.Score = 0.8
		match.Method = "substring"
		match.Notes = "Partial match (substring found)"
		return match
	}

	// Levenshtein-based similarity
	similarity := calculateSimilarity(expNorm, actNorm)
	match.Score = similarity
	if similarity > 0.7 {
		match.Method = "fuzzy_high"
		match.Notes = fmt.Sprintf("High similarity (%.2f)", similarity)
	} else if similarity > 0.4 {
		match.Method = "fuzzy_medium"
		match.Notes = fmt.Sprintf("Medium similarity (%.2f)", similarity)
	} else {
		match.Method = "no_match"
		match.Notes = fmt.Sprintf("Low similarity (%.2f)", similarity)
	}

	return match
}

// normalizeForComparison normalizes text for comparison
func normalizeForComparison(text string) string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove punctuation
	re := regexp.MustCompile(`[^\w\s]`)
	text = re.ReplaceAllString(text, "")

	// Remove extra whitespace
	text = strings.Join(strings.Fields(text), " ")

	return strings.TrimSpace(text)
}

// calculateSimilarity calculates similarity ratio (0.0 to 1.0) using Levenshtein distance
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	distance := levenshteinDistance(s1, s2)
	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	// Convert distance to similarity (0.0 to 1.0)
	return 1.0 - (float64(distance) / float64(maxLen))
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

	// Create a 2D slice for dynamic programming
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

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CompareMARCRecords performs comprehensive field-by-field comparison of two MARC records
func CompareMARCRecords(referenceMARC, generatedMARC string) *FullMARCComparison {
	comparison := &FullMARCComparison{
		Fields:        make(map[string]FieldMatch),
		ReferenceMARC: referenceMARC,
		GeneratedMARC: generatedMARC,
	}

	// Parse both MARC records into field maps
	refFields := parseMARCFields(referenceMARC)
	genFields := parseMARCFields(generatedMARC)

	totalScore := 0.0
	fieldCount := 0
	totalLevenshtein := 0

	// Compare each field from reference MARC
	for tag, refValue := range refFields {
		genValue, exists := genFields[tag]

		match := FieldMatch{
			Expected: refValue,
			Actual:   genValue,
		}

		if !exists {
			match.Score = 0.0
			match.Method = "missing"
			match.Notes = "Field missing from generated MARC"
			comparison.FieldsMissing++
		} else {
			// Calculate similarity using Levenshtein distance
			refNorm := normalizeForComparison(refValue)
			genNorm := normalizeForComparison(genValue)

			distance := levenshteinDistance(refNorm, genNorm)
			totalLevenshtein += distance

			if refNorm == genNorm {
				match.Score = 1.0
				match.Method = "exact"
				match.Notes = "Exact match"
				comparison.FieldsMatched++
			} else {
				similarity := calculateSimilarity(refNorm, genNorm)
				match.Score = similarity

				if similarity > 0.8 {
					match.Method = "fuzzy_high"
					match.Notes = fmt.Sprintf("High similarity (%.2f), Levenshtein distance: %d", similarity, distance)
					comparison.FieldsMatched++
				} else if similarity > 0.5 {
					match.Method = "fuzzy_medium"
					match.Notes = fmt.Sprintf("Medium similarity (%.2f), Levenshtein distance: %d", similarity, distance)
					comparison.FieldsIncorrect++
				} else {
					match.Method = "no_match"
					match.Notes = fmt.Sprintf("Low similarity (%.2f), Levenshtein distance: %d", similarity, distance)
					comparison.FieldsIncorrect++
				}
			}
		}

		comparison.Fields[tag] = match
		totalScore += match.Score
		fieldCount++
	}

	// Check for extra fields in generated MARC
	for tag, genValue := range genFields {
		if _, exists := refFields[tag]; !exists {
			comparison.Fields[tag] = FieldMatch{
				Expected: "",
				Actual:   genValue,
				Score:    0.0,
				Method:   "extra",
				Notes:    "Extra field not in reference MARC",
			}
			fieldCount++
		}
	}

	// Calculate overall score
	if fieldCount > 0 {
		comparison.OverallScore = totalScore / float64(fieldCount)
	}
	comparison.LevenshteinTotal = totalLevenshtein

	return comparison
}

// parseMARCFields parses a MARC record into a map of tag -> value
func parseMARCFields(marcRecord string) map[string]string {
	fields := make(map[string]string)

	// Match MARC fields like:
	// =245  10$aTitle$bsubtitle
	// =100  1\$aAuthor, Name
	re := regexp.MustCompile(`(?m)^=?(\d{3})\s+[^\$]*(\$.*)$`)
	matches := re.FindAllStringSubmatch(marcRecord, -1)

	for _, match := range matches {
		if len(match) > 2 {
			tag := match[1]
			value := match[2]
			fields[tag] = strings.TrimSpace(value)
		}
	}

	return fields
}
