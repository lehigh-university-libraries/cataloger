package metrics

import (
	"testing"
)

func TestCompareMARCFields(t *testing.T) {
	tests := []struct {
		name          string
		generatedMARC string
		title         string
		author        string
		date          string
		isbn          string
		subject       string
		minTitleScore float64
		minOverall    float64
	}{
		{
			name: "exact matches",
			generatedMARC: `00000nam  2200000 a 4500
001 12345
245 00 $a The Great Gatsby / $c F. Scott Fitzgerald
100 1  $a Fitzgerald, F. Scott
260    $a New York : $b Scribner, $c 1925
020    $a 978-0-7432-7356-5
650  0 $a American fiction`,
			title:         "The Great Gatsby",
			author:        "F. Scott Fitzgerald",
			date:          "1925",
			isbn:          "978-0-7432-7356-5",
			subject:       "American fiction",
			minTitleScore: 0.8,
			minOverall:    0.7,
		},
		{
			name: "fuzzy matches",
			generatedMARC: `00000nam  2200000 a 4500
245 00 $a Great Gatsby
100 1  $a F Scott Fitzgerald
260    $c 1925`,
			title:         "The Great Gatsby",
			author:        "F. Scott Fitzgerald",
			date:          "1925",
			isbn:          "",
			subject:       "",
			minTitleScore: 0.5,
			minOverall:    0.4,
		},
		{
			name: "no matches",
			generatedMARC: `00000nam  2200000 a 4500
245 00 $a Different Book
100 1  $a Different Author`,
			title:         "The Great Gatsby",
			author:        "F. Scott Fitzgerald",
			date:          "1925",
			isbn:          "978-0-7432-7356-5",
			subject:       "Fiction",
			minTitleScore: 0.0,
			minOverall:    0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison := CompareMARCFields(
				tt.generatedMARC,
				tt.title,
				tt.author,
				tt.date,
				tt.isbn,
				tt.subject,
			)

			if comparison == nil {
				t.Fatal("Expected non-nil comparison")
			}

			// Check title score
			if comparison.TitleMatch.Score < tt.minTitleScore {
				t.Errorf("Title score %.2f below minimum %.2f",
					comparison.TitleMatch.Score, tt.minTitleScore)
			}

			// Check overall score
			if comparison.OverallScore < tt.minOverall {
				t.Errorf("Overall score %.2f below minimum %.2f",
					comparison.OverallScore, tt.minOverall)
			}

			// Verify field level scores exist
			if len(comparison.FieldLevelScores) == 0 {
				t.Error("Expected field level scores to be populated")
			}

			// Verify all matches have expected and actual values
			if comparison.TitleMatch.Expected != tt.title {
				t.Errorf("Title expected mismatch: got %q, want %q",
					comparison.TitleMatch.Expected, tt.title)
			}

			if comparison.AuthorMatch.Expected != tt.author {
				t.Errorf("Author expected mismatch: got %q, want %q",
					comparison.AuthorMatch.Expected, tt.author)
			}
		})
	}
}

func TestCompareMARCFields_EmptyMARC(t *testing.T) {
	comparison := CompareMARCFields("", "Title", "Author", "2020", "123", "Subject")

	if comparison == nil {
		t.Fatal("Expected non-nil comparison even for empty MARC")
	}

	// All scores should be 0 or low for missing fields
	if comparison.TitleMatch.Score > 0.3 {
		t.Errorf("Expected low title score for empty MARC, got %.2f", comparison.TitleMatch.Score)
	}
}

func TestCompareMARCFields_WeightedScoring(t *testing.T) {
	// Title and author are weighted at 30% each, so they should dominate the score
	marc := `00000nam  2200000 a 4500
245 00 $a Perfect Title Match
100 1  $a Perfect Author Match`

	comparison := CompareMARCFields(
		marc,
		"Perfect Title Match",
		"Perfect Author Match",
		"Wrong Date",    // Wrong
		"Wrong ISBN",    // Wrong
		"Wrong Subject", // Wrong
	)

	// Even with date/isbn/subject wrong, title+author perfect should give good score
	// Title (30%) + Author (30%) = 60% minimum
	if comparison.OverallScore < 0.55 {
		t.Errorf("Expected overall score >= 0.55 with perfect title+author, got %.2f",
			comparison.OverallScore)
	}
}

func TestCompareMARCFields_MissingFields(t *testing.T) {
	marc := `00000nam  2200000 a 4500
245 00 $a Some Title`

	comparison := CompareMARCFields(
		marc,
		"Some Title",
		"", // Both missing
		"", // Both missing
		"", // Both missing
		"", // Both missing
	)

	// Author, date, isbn, subject should be marked as "both_missing"
	if comparison.AuthorMatch.Method != "both_missing" {
		t.Errorf("Expected author method='both_missing', got %q", comparison.AuthorMatch.Method)
	}

	// Both missing should give partial credit (0.5)
	if comparison.AuthorMatch.Score != 0.5 {
		t.Errorf("Expected author score=0.5 for both_missing, got %.2f", comparison.AuthorMatch.Score)
	}
}

func TestFieldMatch_Structure(t *testing.T) {
	marc := `245 00 $a Test Title
100 1  $a Test Author`

	comparison := CompareMARCFields(marc, "Test Title", "Test Author", "", "", "")

	// Verify FieldMatch structure is populated correctly
	if comparison.TitleMatch.Expected == "" {
		t.Error("TitleMatch.Expected should be populated")
	}

	if comparison.TitleMatch.Actual == "" {
		t.Error("TitleMatch.Actual should be populated from MARC")
	}

	if comparison.TitleMatch.Method == "" {
		t.Error("TitleMatch.Method should be populated")
	}

	// Score should be between 0 and 1
	if comparison.TitleMatch.Score < 0 || comparison.TitleMatch.Score > 1 {
		t.Errorf("TitleMatch.Score should be between 0 and 1, got %.2f", comparison.TitleMatch.Score)
	}
}

func TestCompareMARCFields_SubfieldExtraction(t *testing.T) {
	// Test that subfields are correctly extracted
	marc := `260    $a New York : $b Scribner, $c 1925`

	comparison := CompareMARCFields(marc, "", "", "1925", "", "")

	// Date from 260$c should match
	if comparison.DateMatch.Score < 0.9 {
		t.Errorf("Expected high date score for 260$c extraction, got %.2f", comparison.DateMatch.Score)
	}
}

func TestCompareMARCFields_MultipleISBNFormats(t *testing.T) {
	tests := []struct {
		name         string
		marc         string
		expectedISBN string
		minScore     float64
	}{
		{
			name:         "hyphenated ISBN",
			marc:         "020    $a 978-0-123-45678-9",
			expectedISBN: "978-0-123-45678-9",
			minScore:     0.9,
		},
		{
			name:         "ISBN without hyphens",
			marc:         "020    $a 9780123456789",
			expectedISBN: "978-0-123-45678-9",
			minScore:     0.7, // Should still match with normalization
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comparison := CompareMARCFields(tt.marc, "", "", "", tt.expectedISBN, "")
			if comparison.ISBNMatch.Score < tt.minScore {
				t.Errorf("ISBN score %.2f below minimum %.2f",
					comparison.ISBNMatch.Score, tt.minScore)
			}
		})
	}
}
