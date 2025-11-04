package marcgen

import (
	"fmt"
	"strings"
	"time"

	"github.com/lehigh-university-libraries/cataloger/internal/eval/dataset"
)

// GenerateMARCFromMetadata creates a MARC record from Institutional Books metadata
// This serves as the "ground truth" MARC for comparison
func GenerateMARCFromMetadata(record dataset.InstitutionalBooksRecord) string {
	var marc strings.Builder

	// Leader (always start with this)
	marc.WriteString("=LDR  00000nam  2200000   4500\n")

	// 008 - Fixed-Length Data Elements
	marc.WriteString(fmt.Sprintf("=008  %s%s||||%s||||||||||||%s|||||d\n",
		formatDate008(),
		record.DateTypesSource,
		record.Date1Source,
		record.LanguageSource))

	// 020 - ISBN
	if len(record.IdentifiersSource.ISBN) > 0 {
		for _, isbn := range record.IdentifiersSource.ISBN {
			marc.WriteString(fmt.Sprintf("=020  \\\\$a%s\n", isbn))
		}
	}

	// 050 - Library of Congress Call Number (from LCCN if available)
	if len(record.IdentifiersSource.LCCN) > 0 {
		marc.WriteString(fmt.Sprintf("=050  \\4$a%s\n", record.IdentifiersSource.LCCN[0]))
	}

	// 100 - Main Entry - Personal Name
	if record.AuthorSource != "" {
		// Format: Last, First
		marc.WriteString(fmt.Sprintf("=100  1\\$a%s\n", record.AuthorSource))
	}

	// 245 - Title Statement
	if record.TitleSource != "" {
		// Check if title starts with article for indicator
		indicator2 := "0"
		lowerTitle := strings.ToLower(record.TitleSource)
		if strings.HasPrefix(lowerTitle, "the ") {
			indicator2 = "4"
		} else if strings.HasPrefix(lowerTitle, "a ") || strings.HasPrefix(lowerTitle, "an ") {
			indicator2 = "2"
		}

		// Determine if we have author (indicator 1)
		indicator1 := "0"
		if record.AuthorSource != "" {
			indicator1 = "1"
		}

		marc.WriteString(fmt.Sprintf("=245  %s%s$a%s\n", indicator1, indicator2, record.TitleSource))
	}

	// 260/264 - Publication Information
	if record.Date1Source != "" {
		marc.WriteString(fmt.Sprintf("=264  \\1$c%s\n", record.Date1Source))
	}

	// 500 - General Note
	if record.GeneralNoteSource != "" {
		marc.WriteString(fmt.Sprintf("=500  \\\\$a%s\n", record.GeneralNoteSource))
	}

	// 650 - Subject Added Entry - Topical Term
	if record.TopicOrSubjectSource != "" {
		marc.WriteString(fmt.Sprintf("=650  \\0$a%s\n", record.TopicOrSubjectSource))
	}

	// 655 - Genre/Form
	if record.GenreOrFormSource != "" {
		marc.WriteString(fmt.Sprintf("=655  \\7$a%s\n", record.GenreOrFormSource))
	}

	return marc.String()
}

// formatDate008 returns the current date in YYMMDD format for 008 field positions 0-5
func formatDate008() string {
	now := time.Now()
	return now.Format("060102")
}
