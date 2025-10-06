# MARC Cataloging Evaluation

The `cataloger-eval` CLI tool enables systematic evaluation of MARC record generation quality by comparing LLM-generated records against professional catalog records.

## Overview

The evaluation workflow consists of three steps:

1. **Fetch**: Query your library catalog (VuFind/FOLIO) to build an evaluation dataset
2. **Run**: Generate MARC records from images and compare them to reference records
3. **Report**: Analyze results and generate detailed comparison reports

## Installation

Build the eval CLI:

```bash
go build -o cataloger-eval ./cmd/eval
```

## Commands

### `eval fetch` - Build Evaluation Dataset

Fetches MARC records and associated images from your library catalog.

```bash
eval fetch --catalog vufind \
           --url https://catalog.yourlibrary.edu \
           --limit 100 \
           --output ./eval_data
```

**Options**:
- `--catalog`: Catalog type (`vufind` or `folio`)
- `--url`: Catalog URL (required)
- `--limit`: Number of records to fetch (default: 100)
- `--output`: Output directory for dataset (default: `./eval_data`)
- `--api-key`: API key for FOLIO authentication (optional)

**Output**:
- `dataset.json`: Metadata for all fetched records
- `images/`: Downloaded cover, title page, and copyright images

### `eval run` - Run Evaluation

Generates MARC records from images and compares them to reference records.

```bash
eval run --dataset ./eval_data \
         --provider ollama \
         --model mistral-small3.2:24b \
         --concurrency 4 \
         --output ./eval_results
```

**Options**:
- `--dataset`: Dataset directory (default: `./eval_data`)
- `--provider`: LLM provider (`ollama` or `openai`, default: `ollama`)
- `--model`: Model name (uses environment defaults if not specified)
- `--concurrency`: Number of concurrent evaluations (default: 1)
- `--output`: Output directory for results (default: `./eval_results`)

**Output**:
- `results.json`: Complete evaluation results with comparison scores

### `eval report` - Generate Report

Generates detailed comparison reports from evaluation results.

```bash
# Text report (default)
eval report --results ./eval_results

# JSON output
eval report --results ./eval_results --format json

# CSV for spreadsheet analysis
eval report --results ./eval_results --format csv > results.csv
```

**Options**:
- `--results`: Results directory (default: `./eval_results`)
- `--format`: Output format (`text`, `json`, or `csv`, default: `text`)

## Comparison Methodology

### Field Weighting

MARC fields are weighted by cataloging importance:

| Field | Weight | Description |
|-------|--------|-------------|
| 020 | 0.05 | ISBN |
| 100 | 0.15 | Main entry - personal name |
| 110 | 0.15 | Main entry - corporate name |
| 245 | 0.25 | Title statement (most important) |
| 250 | 0.05 | Edition statement |
| 260 | 0.15 | Publication info (pre-RDA) |
| 264 | 0.15 | Publication info (RDA) |
| 300 | 0.05 | Physical description |
| 650 | 0.10 | Subject headings |
| 700 | 0.05 | Added entry - personal name |

### Similarity Scoring

Field content is compared using **Levenshtein distance** normalized to a 0-1 similarity score:
- `1.0` = Identical
- `0.8-0.99` = Very similar (minor differences)
- `0.5-0.79` = Moderately similar
- `0.0-0.49` = Significantly different

### Overall Score

The overall score is calculated as a weighted average:

```
overall_score = Σ (field_weight × field_similarity) / Σ field_weight
```

## Example Workflow

### 1. Fetch Records from VuFind

```bash
eval fetch --catalog vufind \
           --url https://find.lehigh.edu \
           --limit 50 \
           --output ./lehigh_eval
```

### 2. Run Evaluation with Ollama

```bash
eval run --dataset ./lehigh_eval \
         --provider ollama \
         --model mistral-small3.2:24b \
         --concurrency 2 \
         --output ./lehigh_results
```

### 3. View Text Report

```bash
eval report --results ./lehigh_results
```

**Example Output**:
```
========================================
MARC Cataloging Evaluation Report
========================================
Provider: ollama
Model:    mistral-small3.2:24b

========================================
Evaluation Summary
========================================
Total Records:      50
Successful Evals:   48
Failed Evals:       2

Average Score:      82.45%
Median Score:       85.20%
Min Score:          45.30%
Max Score:          98.70%

Field Accuracies:
  020: 92.30%
  100: 78.50%
  245: 91.20%
  260: 75.80%
  ...
```

### 4. Export to CSV for Analysis

```bash
eval report --results ./lehigh_results --format csv > analysis.csv
```

## VuFind Integration Notes

The VuFind client expects:
- **Search API**: `/api/v1/search` endpoint
- **MARC Export**: `/Record/{id}/Export?style=MARC` endpoint
- **Cover Images**: Configured in VuFind (MARC 856 fields or cover image service)

### Customization

If your VuFind instance has custom endpoints, modify `internal/catalog/client.go`:

```go
// Example: Custom MARC export endpoint
marcURL := fmt.Sprintf("%s/api/custom/marc/%s", c.BaseURL, recordID)
```

## FOLIO Integration Notes

FOLIO support is planned but not yet fully implemented. The client expects:
- **Okapi Gateway**: Standard FOLIO authentication
- **Source Record Storage API**: `/source-storage/stream/marc-record-identifiers`
- **X-Okapi-Token**: Authentication header

## Troubleshooting

### "No image available for cataloging"

Ensure your catalog records have image URLs. For VuFind:
1. Check MARC 856 fields contain image URLs
2. Verify cover image service is configured
3. Manually add image URLs to dataset.json if needed

### Low Similarity Scores

Common causes:
- Different cataloging standards (AACR2 vs RDA)
- Abbreviated vs full author names
- Different subject heading schemes
- Publisher name variations

Examine detailed field differences in the text report to identify patterns.

### API Errors

For VuFind:
- Verify URL is accessible: `curl https://catalog.example.edu/api/v1/search`
- Check API is enabled in VuFind config

For FOLIO:
- Ensure `--api-key` is provided
- Verify Okapi token is valid

## Advanced Usage

### Filtering Specific Fields

Modify `DefaultFieldWeights` in `internal/evaluation/comparison.go` to focus on specific fields:

```go
var CustomFieldWeights = []FieldWeight{
    {"245", 0.50}, // Title - very important
    {"100", 0.30}, // Author
    {"260", 0.20}, // Publisher
}
```

### Custom Similarity Threshold

Adjust comparison logic in `compareFieldContent()` to use custom thresholds for different field types.

## Future Enhancements

- Full FOLIO API implementation
- Support for MARCXML format
- Batch processing with checkpointing
- nDCG scoring for ranked field importance
- Subject heading vocabulary validation
- Multi-language cataloging evaluation
- Integration with OpenRefine for data cleaning

## See Also

- [MARC 21 Format for Bibliographic Data](https://www.loc.gov/marc/bibliographic/)
- [VuFind API Documentation](https://vufind.org/wiki/development:architecture:api)
- [FOLIO Source Record Storage](https://wiki.folio.org/display/DD/Source+Record+Storage)
