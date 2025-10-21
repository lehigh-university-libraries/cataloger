package dataset

// InstitutionalBooksRecord represents a record from the Institutional Books 1.0 dataset
// Dataset: https://huggingface.co/datasets/instdin/institutional-books-1.0
type InstitutionalBooksRecord struct {
	// Core identifiers
	BarcodeSource string `json:"barcode_src" parquet:"barcode_src"` // Primary key

	// Bibliographic metadata (ground truth for MARC comparison)
	TitleSource     string `json:"title_src" parquet:"title_src"`
	AuthorSource    string `json:"author_src" parquet:"author_src"`
	Date1Source     string `json:"date1_src" parquet:"date1_src"`
	Date2Source     string `json:"date2_src" parquet:"date2_src"`
	DateTypesSource string `json:"date_types_src" parquet:"date_types_src"`

	// Additional metadata for evaluation
	LanguageSource       string `json:"language_src" parquet:"language_src"`                 // ISO 639-3 code
	TopicOrSubjectSource string `json:"topic_or_subject_src" parquet:"topic_or_subject_src"` // Topic/subject info
	GenreOrFormSource    string `json:"genre_or_form_src" parquet:"genre_or_form_src"`
	GeneralNoteSource    string `json:"general_note_src" parquet:"general_note_src"`

	// Identifiers for cross-referencing
	IdentifiersSource Identifiers `json:"identifiers_src" parquet:"identifiers_src"`

	// HathiTrust data for accessing the full book
	HathitrustDataExt HathitrustData `json:"hathitrust_data_ext" parquet:"hathitrust_data_ext"`

	// OCR text (we'll use this as the input to our cataloger)
	TextByPageSource []string `json:"text_by_page_src" parquet:"text_by_page_src,list"` // Original OCR text
	TextByPageGen    []string `json:"text_by_page_gen" parquet:"text_by_page_gen,list"` // Post-processed OCR text

	// Statistics
	PageCountSource int `json:"page_count_src" parquet:"page_count_src"`
	TokenCountGen   int `json:"token_count_o200k_base_gen" parquet:"token_count_o200k_base_gen"`
}

// Identifiers contains bibliographic identifiers
type Identifiers struct {
	LCCN []string `json:"lccn" parquet:"lccn,list"`   // Library of Congress Control Numbers
	ISBN []string `json:"isbn" parquet:"isbn,list"`   // International Standard Book Numbers
	OCLC []string `json:"ocolc" parquet:"ocolc,list"` // OCLC Control Numbers
}

// HathitrustData contains rights and access information from HathiTrust
type HathitrustData struct {
	URL        string `json:"url" parquet:"url"`                 // Permalink to volume on HathiTrust
	RightsCode string `json:"rights_code" parquet:"rights_code"` // Rights determination code
	ReasonCode string `json:"reason_code" parquet:"reason_code"` // Rights determination reason
	LastCheck  string `json:"last_check" parquet:"last_check"`   // Date info was pulled
}

// GetTitlePageText returns the OCR text for the title page
// Typically the title page is within the first 10 pages
func (r *InstitutionalBooksRecord) GetTitlePageText() string {
	// Use post-processed text if available, otherwise use source
	var pages []string
	if len(r.TextByPageGen) > 0 {
		pages = r.TextByPageGen
	} else {
		pages = r.TextByPageSource
	}

	// Get first 10 pages (title page is usually pages 3-7)
	if len(pages) == 0 {
		return ""
	}

	endIdx := 10
	if len(pages) < endIdx {
		endIdx = len(pages)
	}

	// Concatenate first 10 pages with clear markers
	result := ""
	for i := 0; i < endIdx; i++ {
		result += pages[i] + "\n\n---PAGE BREAK---\n\n"
	}

	return result
}

// GetPrimaryAuthor returns the first author name (useful for evaluation)
func (r *InstitutionalBooksRecord) GetPrimaryAuthor() string {
	return r.AuthorSource
}

// GetPrimaryDate returns the primary date for the publication
func (r *InstitutionalBooksRecord) GetPrimaryDate() string {
	if r.Date1Source != "" {
		return r.Date1Source
	}
	return r.Date2Source
}

// GetISBN returns the first ISBN if available
func (r *InstitutionalBooksRecord) GetISBN() string {
	if len(r.IdentifiersSource.ISBN) > 0 {
		return r.IdentifiersSource.ISBN[0]
	}
	return ""
}
