# MARC Cataloging Evaluation

The `cataloger-eval` CLI tool enables systematic evaluation of MARC record generation quality by comparing LLM-generated records against professional catalog records.

## Overview

The evaluation workflow consists of three steps:

1. **Fetch**: Harvest MARC records from OAI-PMH endpoints to build an evaluation dataset
2. **Run**: Generate MARC records from images and compare them to reference records
3. **Report**: Analyze results and generate detailed comparison reports

**Key Features**:
- Incremental saving during harvest (records saved immediately, not at end)
- Resumption token support with configurable sleep delays
- Filters for books with ISBN only
- Automatic exclusion of deleted and suppressed records

## Installation

Build the eval CLI:

```bash
go build -o cataloger-eval ./cmd/eval
```

## Commands

### `eval fetch` - Build Evaluation Dataset

Harvests MARC records from OAI-PMH endpoints. Records are saved incrementally as they're fetched, so you can monitor progress in real-time.

```bash
eval fetch --url https://folio.example.edu/oai \
           --prefix marc21 \
           --limit 100 \
           --sleep 2 \
           --output ./eval_data
```

**Options**:
- `--url`: OAI-PMH base URL (required, can also use `OAI_PMH_URL` env var)
- `--prefix`: OAI-PMH metadata prefix (default: `marc21`)
- `--limit`: Number of records to save (default: 100)
- `--sleep`: Seconds to sleep between resumption token requests (default: 0, no sleep)
- `--output`: Output directory for dataset (default: `./eval_data`)
- `--exclude`: MARC tag to exclude (can be specified multiple times, e.g., `--exclude 655 --exclude 880`)

**Filtering**:
The fetch command automatically filters records to include only:
- Books (Leader type 'a' or 't', bib level 'm')
- Records with ISBN (020$a field present)
- Non-deleted records (Leader status ≠ 'd')
- Non-suppressed records (999$i ≠ 1)

**Incremental Saving**:
Records are saved to `dataset.json` immediately as they're harvested. You can stop the harvest at any time with Ctrl+C and resume later. Check progress:

```bash
# Monitor progress in real-time
watch -n 1 'cat ./eval_data/dataset.json | jq ".items | length"'
```

**Resumption Token Sleep**:
Use `--sleep` to be polite to the OAI-PMH server and avoid rate limiting:
- `--sleep 1`: Wait 1 second between batches (typical OAI-PMH batch = 100-500 records)
- `--sleep 2`: Wait 2 seconds (recommended for production servers)
- `--sleep 5`: Wait 5 seconds (very conservative)

**Output**:
- `dataset.json`: Metadata for all fetched records (saved incrementally)

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

### 1. Fetch Records from OAI-PMH

```bash
eval fetch --url https://folio.lehigh.edu/oai \
           --limit 50 \
           --sleep 2 \
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

## OAI-PMH Integration Notes

The fetch command uses standard OAI-PMH protocol:
- **ListRecords**: Harvests records with automatic resumption token handling
- **Metadata Prefix**: Typically `marc21` or `marcxml`
- **Resumption Tokens**: Automatically handled with configurable sleep delays

### Supported Catalogs

Any system with OAI-PMH support:
- **FOLIO**: `/oai` endpoint (e.g., `https://folio.example.edu/oai`)
- **VuFind**: `/OAI/Server` endpoint (e.g., `https://vufind.example.edu/OAI/Server`)
- **Koha**: `/cgi-bin/koha/oai.pl` endpoint
- **DSpace**: `/oai/request` endpoint
- **EPrints**: `/cgi/oai2` endpoint

### Testing OAI-PMH Endpoints

Verify your OAI-PMH endpoint:

```bash
# List available metadata formats
curl "https://folio.example.edu/oai?verb=ListMetadataFormats"

# Get a sample record
curl "https://folio.example.edu/oai?verb=ListRecords&metadataPrefix=marc21&set=books"
```

## Troubleshooting

### OAI-PMH Harvest Issues

**Connection refused / timeout**:
- Verify URL is accessible: `curl "https://folio.example.edu/oai?verb=Identify"`
- Check firewall rules allow access
- Try increasing sleep delay with `--sleep 5`

**No records saved**:
- Check if records have ISBNs (fetch only saves books with ISBN)
- Verify metadata prefix is correct (try `--prefix marcxml` if `marc21` fails)
- Look at debug logs: `LOG_LEVEL=DEBUG ./cataloger-eval fetch ...`

**Rate limiting / HTTP 429**:
- Increase `--sleep` parameter (try 5-10 seconds)
- Reduce concurrent harvesting if running multiple fetches

### Incremental Save Issues

**dataset.json not updating**:
- File is only written after each record processes successfully
- Check disk space: `df -h`
- Verify write permissions: `ls -la ./eval_data/`

**Corrupted dataset.json**:
- The file is rewritten completely on each save for consistency
- If harvest crashes mid-write, delete `dataset.json` and restart
- Consider backing up: `cp eval_data/dataset.json eval_data/dataset.json.bak`

### Low Similarity Scores

Common causes:
- Different cataloging standards (AACR2 vs RDA)
- Abbreviated vs full author names
- Different subject heading schemes
- Publisher name variations

Examine detailed field differences in the text report to identify patterns.

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

- Resume capability for interrupted harvests
- Parallel OAI-PMH harvesting
- Image fetching from external sources (OpenLibrary, Google Books)
- Support for MARCXML format variations
- nDCG scoring for ranked field importance
- Subject heading vocabulary validation
- Multi-language cataloging evaluation
- Integration with OpenRefine for data cleaning

## See Also

- [MARC 21 Format for Bibliographic Data](https://www.loc.gov/marc/bibliographic/)
- [OAI-PMH Protocol Specification](https://www.openarchives.org/pmh/)
- [FOLIO OAI-PMH Documentation](https://wiki.folio.org/display/FOLIOtips/FOLIO+OAI-PMH)
- [VuFind OAI Server](https://vufind.org/wiki/configuration:oai_server)
