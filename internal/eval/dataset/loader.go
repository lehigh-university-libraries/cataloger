package dataset

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// Loader handles loading of the Institutional Books dataset
type Loader struct {
	datasetPath string
}

// NewLoader creates a new dataset loader
func NewLoader(datasetPath string) *Loader {
	return &Loader{
		datasetPath: datasetPath,
	}
}

// Load loads records from a dataset file (JSONL or Parquet)
func (l *Loader) Load() ([]InstitutionalBooksRecord, error) {
	// Detect file format
	ext := strings.ToLower(filepath.Ext(l.datasetPath))

	switch ext {
	case ".parquet":
		return l.loadParquet()
	case ".jsonl", ".json":
		return l.loadJSONL()
	default:
		return nil, fmt.Errorf("unsupported file format: %s (supported: .parquet, .jsonl)", ext)
	}
}

// loadJSONL loads records from a JSONL file
func (l *Loader) loadJSONL() ([]InstitutionalBooksRecord, error) {
	slog.Debug("Opening JSONL file", "path", l.datasetPath)

	file, err := os.Open(l.datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	slog.Debug("JSONL file stats", "size_bytes", info.Size(), "size_mb", info.Size()/1024/1024)

	var records []InstitutionalBooksRecord
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large JSON lines
	const maxCapacity = 10 * 1024 * 1024 // 10MB per line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var record InstitutionalBooksRecord
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, fmt.Errorf("failed to parse JSON at line %d: %w", lineNum, err)
		}

		records = append(records, record)

		// Log first record for verification
		if lineNum == 1 {
			slog.Debug("First record sample",
				"barcode", record.BarcodeSource,
				"title", record.TitleSource,
				"author", record.AuthorSource,
				"has_ocr_pages", len(record.TextByPageSource))
		}

		// Log progress every 1000 records
		if lineNum%1000 == 0 {
			slog.Debug("Reading JSONL", "lines_read", lineNum)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dataset: %w", err)
	}

	slog.Debug("Finished reading JSONL file", "total_records", len(records), "total_lines", lineNum)

	return records, nil
}

// loadParquet loads records from a Parquet file
func (l *Loader) loadParquet() ([]InstitutionalBooksRecord, error) {
	slog.Debug("Opening Parquet file", "path", l.datasetPath)

	file, err := os.Open(l.datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	slog.Debug("Parquet file stats", "size_bytes", info.Size(), "size_mb", info.Size()/1024/1024)

	// Create parquet reader
	pf, err := parquet.OpenFile(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet: %w", err)
	}

	slog.Debug("Parquet file opened successfully", "num_rows", pf.NumRows(), "num_row_groups", len(pf.RowGroups()))

	// Read all rows
	reader := parquet.NewGenericReader[InstitutionalBooksRecord](pf)
	defer reader.Close()

	var records []InstitutionalBooksRecord
	rows := make([]InstitutionalBooksRecord, 128) // Read in batches

	batchNum := 0
	totalRead := 0

	for {
		n, err := reader.Read(rows)
		if n > 0 {
			batchNum++
			totalRead += n
			records = append(records, rows[:n]...)
			slog.Debug("Read batch from Parquet", "batch", batchNum, "rows_in_batch", n, "total_rows_read", totalRead)

			// Log first record of first batch for verification
			if batchNum == 1 && len(rows) > 0 {
				slog.Debug("First record sample",
					"barcode", rows[0].BarcodeSource,
					"title", rows[0].TitleSource,
					"author", rows[0].AuthorSource,
					"has_ocr_pages", len(rows[0].TextByPageSource))
			}
		}
		if err != nil {
			break
		}
	}

	slog.Debug("Finished reading Parquet file", "total_records", len(records), "total_batches", batchNum)

	return records, nil
}

// LoadSample loads a limited number of records (useful for testing)
func (l *Loader) LoadSample(limit int) ([]InstitutionalBooksRecord, error) {
	// Detect file format
	ext := strings.ToLower(filepath.Ext(l.datasetPath))

	switch ext {
	case ".parquet":
		return l.loadParquetSample(limit)
	case ".jsonl", ".json":
		return l.loadJSONLSample(limit)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

// loadJSONLSample loads a sample from JSONL
func (l *Loader) loadJSONLSample(limit int) ([]InstitutionalBooksRecord, error) {
	file, err := os.Open(l.datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file: %w", err)
	}
	defer file.Close()

	var records []InstitutionalBooksRecord
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large JSON lines
	const maxCapacity = 10 * 1024 * 1024 // 10MB per line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() && len(records) < limit {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var record InstitutionalBooksRecord
		if err := json.Unmarshal(line, &record); err != nil {
			// Skip malformed lines but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse JSON at line %d: %v\n", lineNum, err)
			continue
		}

		records = append(records, record)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dataset: %w", err)
	}

	return records, nil
}

// loadParquetSample loads a sample from Parquet
func (l *Loader) loadParquetSample(limit int) ([]InstitutionalBooksRecord, error) {
	slog.Debug("Opening Parquet file for sample", "path", l.datasetPath, "sample_limit", limit)

	file, err := os.Open(l.datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	slog.Debug("Parquet file stats", "size_bytes", info.Size(), "size_mb", info.Size()/1024/1024)

	pf, err := parquet.OpenFile(file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet: %w", err)
	}

	slog.Debug("Parquet file opened successfully", "num_rows", pf.NumRows(), "num_row_groups", len(pf.RowGroups()))

	reader := parquet.NewGenericReader[InstitutionalBooksRecord](pf)
	defer reader.Close()

	var records []InstitutionalBooksRecord
	rows := make([]InstitutionalBooksRecord, 128)

	batchNum := 0

	for len(records) < limit {
		n, err := reader.Read(rows)
		if n > 0 {
			batchNum++
			remaining := limit - len(records)
			if n > remaining {
				n = remaining
			}
			records = append(records, rows[:n]...)
			slog.Debug("Read batch from Parquet", "batch", batchNum, "rows_in_batch", n, "total_rows_read", len(records))

			// Log first record of first batch for verification
			if batchNum == 1 && len(rows) > 0 {
				slog.Debug("First record sample",
					"barcode", rows[0].BarcodeSource,
					"title", rows[0].TitleSource,
					"author", rows[0].AuthorSource,
					"has_ocr_pages", len(rows[0].TextByPageSource))
			}
		}
		if err != nil {
			break
		}
	}

	slog.Debug("Finished reading Parquet sample", "total_records", len(records), "total_batches", batchNum)

	return records, nil
}

// LoadWithFilter loads records matching a filter function
func (l *Loader) LoadWithFilter(filterFn func(*InstitutionalBooksRecord) bool) ([]InstitutionalBooksRecord, error) {
	file, err := os.Open(l.datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file: %w", err)
	}
	defer file.Close()

	var records []InstitutionalBooksRecord
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large JSON lines
	const maxCapacity = 10 * 1024 * 1024 // 10MB per line
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		var record InstitutionalBooksRecord
		if err := json.Unmarshal(line, &record); err != nil {
			// Skip malformed lines but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to parse JSON at line %d: %v\n", lineNum, err)
			continue
		}

		// Apply filter
		if filterFn(&record) {
			records = append(records, record)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading dataset: %w", err)
	}

	return records, nil
}
