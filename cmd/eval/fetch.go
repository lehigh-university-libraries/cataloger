package main

import (
	"encoding/xml"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Jpmcrespo/goharvest/oai"
	"github.com/hectorcorrea/marcli/pkg/marc"
	"github.com/lehigh-university-libraries/cataloger/internal/evaluation"
)

func executeFetch(oaiURL, metadataPrefix, outputDir string, limit int, excludeTags []string, sleepSeconds int) error {
	slog.Info("Starting OAI-PMH harvest", "url", oaiURL, "prefix", metadataPrefix, "limit", limit, "sleep", sleepSeconds)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	recordCount := 0
	savedCount := 0
	skippedCount := 0
	filteredCount := 0
	batchCount := 0

	// Harvest records using resumption tokens with batch callback for sleep control
	slog.Info("Harvesting records from OAI-PMH endpoint...")
	request := &oai.Request{
		BaseURL:        oaiURL,
		MetadataPrefix: metadataPrefix,
	}

	request.Harvest(func(response *oai.Response) {
		// Process each record in the batch
		for _, record := range response.ListRecords.Records {
			// Stop if we've reached the limit
			if savedCount >= limit {
				return
			}

			recordCount++

			// Get metadata bytes
			metadataBytes := []byte(record.Metadata.Body)

			// Parse MARC record
			marcRecord, err := parseMARCRecord(metadataBytes)
			if err != nil {
				slog.Warn("Failed to parse MARC record", "id", record.Header.Identifier, "error", err)
				skippedCount++
				continue
			}

			// Skip deleted records (leader position 5 = 'd')
			if isDeletedRecord(marcRecord) {
				slog.Debug("Skipping deleted record", "id", record.Header.Identifier)
				filteredCount++
				continue
			}

			// Skip suppressed records (999$i = 1)
			if isSuppressedRecord(marcRecord) {
				slog.Debug("Skipping suppressed record (999$i = 1)", "id", record.Header.Identifier)
				filteredCount++
				continue
			}

			// Filter by exclude tags
			if shouldExcludeRecord(marcRecord, excludeTags) {
				slog.Debug("Filtered out record by exclude tags", "id", record.Header.Identifier)
				filteredCount++
				continue
			}

			// Only save records that are books with ISBN
			if !isBookWithISBN(marcRecord) {
				slog.Debug("Skipping record (not a book with ISBN)", "id", record.Header.Identifier)
				skippedCount++
				continue
			}

			savedCount++
			slog.Info("Processing record", "id", record.Header.Identifier, "saved", savedCount)

			// Save the record
			item := evaluation.DatasetItem{
				ID:            record.Header.Identifier,
				ReferenceMARC: string(metadataBytes),
			}

			if err := evaluation.AppendDatasetItem(item, outputDir); err != nil {
				slog.Error("Failed to save dataset item", "id", record.Header.Identifier, "error", err)
				continue
			}
		}

		// Sleep between batches if requested and there's a resumption token
		if sleepSeconds > 0 && response.ListRecords.ResumptionToken != "" {
			batchCount++
			slog.Info("Sleeping between resumption token requests", "batch", batchCount, "seconds", sleepSeconds)
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
		}
	})

	slog.Info("Dataset created successfully",
		"saved", savedCount,
		"skipped", skippedCount,
		"filtered", filteredCount,
		"total_processed", recordCount,
		"batches", batchCount)

	fmt.Printf("\nDataset created successfully!\n")
	fmt.Printf("  Records saved: %d\n", savedCount)
	fmt.Printf("  Records skipped (not books with ISBN): %d\n", skippedCount)
	fmt.Printf("  Records filtered (excluded tags): %d\n", filteredCount)
	fmt.Printf("  Total processed: %d\n", recordCount)
	fmt.Printf("  Location: %s\n", outputDir)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Enrich dataset with images: eval enrich --dataset %s\n", outputDir)
	fmt.Printf("  2. Run evaluation: eval run --dataset %s\n", outputDir)

	return nil
}

// parseMARCRecord parses a MARC record from MARCXML or ISO2709 bytes
func parseMARCRecord(data []byte) (*marc.Record, error) {
	// Check if it's MARCXML
	dataStr := string(data)
	if strings.Contains(dataStr, "<marc:record") || strings.Contains(dataStr, "<record") {
		return parseMARCXML(data)
	}

	// Otherwise try ISO2709 binary format
	return parseMARCBinary(data)
}

// parseMARCXML parses MARCXML into a marc.Record
func parseMARCXML(data []byte) (*marc.Record, error) {
	var xmlRec marc.XmlRecord

	// Wrap in XML document if needed
	xmlStr := string(data)
	if !strings.HasPrefix(strings.TrimSpace(xmlStr), "<?xml") {
		xmlStr = `<?xml version="1.0" encoding="UTF-8"?>` + "\n" + xmlStr
	}

	// Handle MARC namespace
	xmlStr = strings.ReplaceAll(xmlStr, "<marc:record", "<record")
	xmlStr = strings.ReplaceAll(xmlStr, "</marc:record>", "</record>")
	xmlStr = strings.ReplaceAll(xmlStr, "<marc:leader>", "<leader>")
	xmlStr = strings.ReplaceAll(xmlStr, "</marc:leader>", "</leader>")
	xmlStr = strings.ReplaceAll(xmlStr, "<marc:controlfield", "<controlfield")
	xmlStr = strings.ReplaceAll(xmlStr, "</marc:controlfield>", "</controlfield>")
	xmlStr = strings.ReplaceAll(xmlStr, "<marc:datafield", "<datafield")
	xmlStr = strings.ReplaceAll(xmlStr, "</marc:datafield>", "</datafield>")
	xmlStr = strings.ReplaceAll(xmlStr, "<marc:subfield", "<subfield")
	xmlStr = strings.ReplaceAll(xmlStr, "</marc:subfield>", "</subfield>")

	if err := xml.Unmarshal([]byte(xmlStr), &xmlRec); err != nil {
		return nil, fmt.Errorf("failed to parse MARCXML: %w", err)
	}

	// Convert XmlRecord to Record
	rec := &marc.Record{}

	// Parse leader
	leader, err := marc.NewLeader([]byte(xmlRec.Leader))
	if err != nil {
		// Non-fatal: some XML records have invalid leaders
		slog.Debug("Invalid leader in XML record", "error", err)
	}
	rec.Leader = leader
	rec.Data = []byte(xmlStr) // Store original XML as data

	// Convert control fields
	for _, control := range xmlRec.ControlFields {
		field := marc.Field{Tag: control.Tag, Value: control.Value}
		rec.Fields = append(rec.Fields, field)
	}

	// Convert data fields
	for _, data := range xmlRec.DataFields {
		field := marc.Field{
			Tag:        data.Tag,
			Indicator1: data.Ind1,
			Indicator2: data.Ind2,
		}
		for _, sub := range data.SubFields {
			subfield := marc.SubField{Code: sub.Code, Value: sub.Value}
			field.SubFields = append(field.SubFields, subfield)
		}
		rec.Fields = append(rec.Fields, field)
	}

	return rec, nil
}

// parseMARCBinary parses ISO2709 binary MARC
func parseMARCBinary(data []byte) (*marc.Record, error) {
	tmpFile, err := os.CreateTemp("", "marc-*.mrc")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek: %w", err)
	}

	marcFile := marc.NewMarcFile(tmpFile)

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

// isDeletedRecord checks if a MARC record is marked as deleted (leader position 5 = 'd')
func isDeletedRecord(record *marc.Record) bool {
	return record.Leader.Status == 'd'
}

// isSuppressedRecord checks if a MARC record is suppressed from discovery (999$i = 1)
func isSuppressedRecord(record *marc.Record) bool {
	for _, field := range record.Fields {
		if field.Tag == "999" {
			for _, subfield := range field.SubFields {
				if subfield.Code == "i" && subfield.Value == "1" {
					return true
				}
			}
		}
	}
	return false
}

// isBookWithISBN checks if a MARC record is a book and has an ISBN
func isBookWithISBN(record *marc.Record) bool {
	// Check Leader Type (position 6)
	// 'a' = Language material (books)
	// 't' = Manuscript language material
	if record.Leader.Type != 'a' && record.Leader.Type != 't' {
		return false
	}

	// Check Leader BibLevel (position 7)
	// 'm' = Monograph/Item
	if record.Leader.BibLevel != 'm' {
		return false
	}

	// Check for ISBN in field 020
	for _, field := range record.Fields {
		if field.Tag == "020" {
			for _, subfield := range field.SubFields {
				if subfield.Code == "a" && len(subfield.Value) > 0 {
					return true
				}
			}
		}
	}

	return false
}

// shouldExcludeRecord checks if a record contains any of the excluded tags
func shouldExcludeRecord(record *marc.Record, excludeTags []string) bool {
	if len(excludeTags) == 0 {
		return false
	}

	for _, field := range record.Fields {
		for _, excludeTag := range excludeTags {
			if field.Tag == excludeTag {
				return true
			}
		}
	}

	return false
}

// extractISBNFromMARC extracts the first ISBN (020$a) from a MARC record
func extractISBNFromMARC(record *marc.Record) string {
	for _, field := range record.Fields {
		if field.Tag == "020" {
			for _, subfield := range field.SubFields {
				if subfield.Code == "a" && len(subfield.Value) > 0 {
					// Return the first ISBN found
					// ISBN may have additional text after space, take just the number part
					parts := strings.Fields(subfield.Value)
					if len(parts) > 0 {
						return parts[0]
					}
				}
			}
		}
	}
	return ""
}
