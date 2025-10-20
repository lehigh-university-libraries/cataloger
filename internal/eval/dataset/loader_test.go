package dataset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	path := "./test.parquet"
	loader := NewLoader(path)

	if loader.datasetPath != path {
		t.Errorf("Expected path %s, got %s", path, loader.datasetPath)
	}
}

func TestGetTitlePageText(t *testing.T) {
	tests := []struct {
		name     string
		record   InstitutionalBooksRecord
		expected string
	}{
		{
			name: "uses gen text when available",
			record: InstitutionalBooksRecord{
				TextByPageGen:    []string{"Page 1 gen", "Page 2 gen"},
				TextByPageSource: []string{"Page 1 src", "Page 2 src"},
			},
			expected: "Page 1 gen\n\n---PAGE BREAK---\n\nPage 2 gen\n\n---PAGE BREAK---\n\n",
		},
		{
			name: "falls back to source text",
			record: InstitutionalBooksRecord{
				TextByPageSource: []string{"Page 1 src", "Page 2 src"},
			},
			expected: "Page 1 src\n\n---PAGE BREAK---\n\nPage 2 src\n\n---PAGE BREAK---\n\n",
		},
		{
			name: "limits to 10 pages",
			record: InstitutionalBooksRecord{
				TextByPageSource: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"},
			},
			expected: "1\n\n---PAGE BREAK---\n\n2\n\n---PAGE BREAK---\n\n3\n\n---PAGE BREAK---\n\n4\n\n---PAGE BREAK---\n\n5\n\n---PAGE BREAK---\n\n6\n\n---PAGE BREAK---\n\n7\n\n---PAGE BREAK---\n\n8\n\n---PAGE BREAK---\n\n9\n\n---PAGE BREAK---\n\n10\n\n---PAGE BREAK---\n\n",
		},
		{
			name:     "returns empty for no pages",
			record:   InstitutionalBooksRecord{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.record.GetTitlePageText()
			if result != tt.expected {
				t.Errorf("Expected:\n%s\nGot:\n%s", tt.expected, result)
			}
		})
	}
}

func TestGetPrimaryDate(t *testing.T) {
	tests := []struct {
		name     string
		record   InstitutionalBooksRecord
		expected string
	}{
		{
			name: "returns date1 when available",
			record: InstitutionalBooksRecord{
				Date1Source: "1920",
				Date2Source: "1925",
			},
			expected: "1920",
		},
		{
			name: "falls back to date2",
			record: InstitutionalBooksRecord{
				Date2Source: "1925",
			},
			expected: "1925",
		},
		{
			name:     "returns empty when no dates",
			record:   InstitutionalBooksRecord{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.record.GetPrimaryDate()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetISBN(t *testing.T) {
	tests := []struct {
		name     string
		record   InstitutionalBooksRecord
		expected string
	}{
		{
			name: "returns first ISBN",
			record: InstitutionalBooksRecord{
				IdentifiersSource: Identifiers{
					ISBN: []string{"978-0-123456-78-9", "978-0-987654-32-1"},
				},
			},
			expected: "978-0-123456-78-9",
		},
		{
			name:     "returns empty for no ISBNs",
			record:   InstitutionalBooksRecord{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.record.GetISBN()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestLoadJSONLSample(t *testing.T) {
	// Create temporary JSONL file
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")

	// Write test data
	testData := `{"barcode_src":"123","title_src":"Test Book","author_src":"Test Author","date1_src":"2020","text_by_page_src":["Page 1"]}
{"barcode_src":"456","title_src":"Another Book","author_src":"Another Author","date1_src":"2021","text_by_page_src":["Page 1"]}
{"barcode_src":"789","title_src":"Third Book","author_src":"Third Author","date1_src":"2022","text_by_page_src":["Page 1"]}
`
	err := os.WriteFile(jsonlPath, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewLoader(jsonlPath)

	// Test loading sample
	records, err := loader.LoadSample(2)
	if err != nil {
		t.Fatalf("LoadSample failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}

	if records[0].BarcodeSource != "123" {
		t.Errorf("Expected barcode 123, got %s", records[0].BarcodeSource)
	}

	if records[0].TitleSource != "Test Book" {
		t.Errorf("Expected title 'Test Book', got %s", records[0].TitleSource)
	}

	if records[1].BarcodeSource != "456" {
		t.Errorf("Expected barcode 456, got %s", records[1].BarcodeSource)
	}
}

func TestLoadJSONL(t *testing.T) {
	// Create temporary JSONL file
	tmpDir := t.TempDir()
	jsonlPath := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"barcode_src":"123","title_src":"Test Book","author_src":"Test Author"}
{"barcode_src":"456","title_src":"Another Book","author_src":"Another Author"}
`
	err := os.WriteFile(jsonlPath, []byte(testData), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	loader := NewLoader(jsonlPath)

	// Test loading all records
	records, err := loader.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
}

func TestLoadUnsupportedFormat(t *testing.T) {
	loader := NewLoader("test.txt")

	_, err := loader.Load()
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}

	_, err = loader.LoadSample(10)
	if err == nil {
		t.Error("Expected error for unsupported format in LoadSample, got nil")
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	loader := NewLoader("/nonexistent/path/file.jsonl")

	_, err := loader.Load()
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}

	_, err = loader.LoadSample(10)
	if err == nil {
		t.Error("Expected error for non-existent file in LoadSample, got nil")
	}
}
