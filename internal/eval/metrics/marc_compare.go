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

// calculateSimilarity calculates similarity ratio (0.0 to 1.0)
// Uses a simple character-based comparison
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple approach: count matching character positions
	matches := 0
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}

	for i := 0; i < minLen; i++ {
		if s1[i] == s2[i] {
			matches++
		}
	}

	maxLen := len(s1)
	if len(s2) > maxLen {
		maxLen = len(s2)
	}

	return float64(matches) / float64(maxLen)
}
