package metadata

// BookMetadata represents the extracted bibliographic metadata from a book
// This matches the structure of the Institutional Books dataset for easy comparison
type BookMetadata struct {
	Title           string   `json:"title"`
	Author          string   `json:"author"`
	Publisher       string   `json:"publisher"`
	PublicationDate string   `json:"publication_date"`
	PublicationCity string   `json:"publication_city"`
	Edition         string   `json:"edition,omitempty"`
	ISBN            []string `json:"isbn,omitempty"`
	Language        string   `json:"language"`
	Subject         string   `json:"subject,omitempty"`
	Genre           string   `json:"genre,omitempty"`
	Series          string   `json:"series,omitempty"`
	Notes           string   `json:"notes,omitempty"`
}

// MetadataComparison represents field-by-field comparison of metadata
type MetadataComparison struct {
	Fields           map[string]FieldComparison
	OverallScore     float64
	FieldsMatched    int
	FieldsMissing    int
	FieldsIncorrect  int
	LevenshteinTotal int
}

// FieldComparison represents comparison for a single metadata field
type FieldComparison struct {
	FieldName string
	Expected  string
	Actual    string
	Score     float64 // 0.0 to 1.0
	Distance  int     // Levenshtein distance
	Match     string  // "exact", "fuzzy_high", "fuzzy_medium", "no_match", "missing"
	Notes     string
}
