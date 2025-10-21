# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cataloger is a web-based book metadata cataloging tool that generates MARC records from images of book title pages using vision-capable LLMs (Ollama or OpenAI).

**Current Status**: MVP complete with working MARC generation and comprehensive evaluation tools.

## 📚 Critical Documentation References
- **README**: `./README.md` - Main project documentation, quick start, and evaluation guide
- **Go Conventions**: `./docs/GO_CONVENTIONS.md` 📋
- **Project Architecture**: `./docs/ARCHITECTURE.md`
- **Evaluation Guide**: `./docs/EVALUATION.md`
- **Dataset Setup**: `./docs/PARTIAL_DATASET.md`

## Current Features (Implemented)

### ✅ Web Interface
- File upload (up to 10MB)
- URL upload
- Three image types: cover, title_page, copyright
- MD5-based file storage to prevent duplicates
- Dark-themed UI with provider/model selection
- Real-time session list with modal display

### ✅ MARC Generation
- Automatic MARC record generation from title page images
- Multi-provider support (Ollama, OpenAI)
- Expert-level prompt designed for Library of Congress standards
- Low temperature (0.1) for consistent, factual output
- Provider/model info stored with sessions

### ✅ Evaluation CLI
- **Institutional Books 1.0 Dataset Support** (983K books)
  - Parquet file reading with debug logging
  - Fast loading (sub-second for 100 records)
  - OCR text extraction for evaluation
- **OAI-PMH Harvesting**
  - Incremental dataset saving (records saved immediately)
  - Resumption token support with configurable sleep delays
  - Automatic filtering (books with ISBN only, excludes deleted/suppressed)
- **MARC Comparison**
  - Field-by-field comparison with weighted scoring
  - Levenshtein distance for similarity measurement
  - Text, JSON, and CSV report formats
- **Concurrent Processing**
  - Configurable concurrency for faster evaluation

## Architecture

### Unified Binary

The project uses a single `cataloger` binary with subcommands:

```bash
cataloger serve           # Web server (default)
cataloger eval <cmd>      # Evaluation commands
```

### Backend (Go)

- **main.go**: Unified entry point, routes to serve or eval
- **cmd/serve/**: Web server implementation (internal)
- **cmd/eval/**: Eval CLI implementation (internal)
- **internal/evalcmd/**: Eval command package (used by main.go)
- **internal/cataloging/service.go**: MARC generation with multi-provider support
  - `GenerateMARCFromImage()`: Main entry point
  - `GenerateMARCFromOCR()`: Generate from OCR text
  - `generateWithOllama()`: Ollama API integration
  - `generateWithOpenAI()`: OpenAI Vision API integration
  - `buildMARCPrompt()`: Expert cataloger prompt
- **internal/ocr/service.go**: LLM-based OCR extraction
  - `ExtractTextFromImage()`: OCR extraction entry point
  - Temperature 0.0 for accurate text extraction
- **internal/evaluation/**: MARC comparison engine
  - `comparison.go`: Field-by-field comparison with Levenshtein distance
  - `dataset.go`: Dataset and results persistence with incremental save support
- **internal/eval/dataset/**: Dataset loaders
  - `loader.go`: Parquet and JSONL loading with debug logging
  - `models.go`: InstitutionalBooksRecord struct with parquet tags
- **internal/handlers/**: HTTP endpoints
  - `upload.go`: File and URL upload with validation
  - `sessions.go`: Session CRUD operations
  - `image_processing.go`: Image download and processing
  - `static.go`: Static file serving
  - `common.go`: Shared utilities and session creation
- **internal/models/models.go**: Data structures
  - `CatalogSession`: Session with images, MARC, provider/model
  - `ImageItem`: Image metadata with OCR text field
- **internal/storage/**: In-memory session store
- **internal/utils/**: MD5 hashing and error handling

### Frontend (Vanilla JS)
- **static/index.html**: Single-page app
- **static/script.js**: Upload, session management, modal display
- **static/styles.css**: Dark theme with provider selection UI

## Commands

### Web Server

```bash
go run main.go              # Default: starts web server
go run main.go serve        # Explicit server start
```

### Evaluation

```bash
# Institutional Books evaluation
cataloger eval eval-ib --sample 10 --verbose

# OAI-PMH harvest
cataloger eval fetch --url https://folio.edu/oai --limit 100 --sleep 2

# Enrich dataset
cataloger eval enrich --dataset ./eval_data --output ./enriched_data

# Run evaluation
cataloger eval run --dataset ./eval_data --provider ollama --concurrency 4

# Generate report
cataloger eval report --results ./eval_results --format text
```

## API Endpoints

- `POST /api/upload` - Upload image (multipart or JSON with URL)
- `GET /api/sessions` - List all sessions
- `GET /api/sessions/{id}` - Get session details
- `PUT /api/sessions/{id}` - Update session
- `GET /healthcheck` - Health check
- `GET /static/*` - Static files
- `GET /static/uploads/*` - Uploaded images

## Environment Variables

```bash
# Provider Configuration
OLLAMA_URL=http://localhost:11434          # or remote URL
OLLAMA_MODEL=mistral-small3.2:24b         # default Ollama model

OPENAI_API_KEY=sk-proj-...                # OpenAI API key
OPENAI_MODEL=gpt-4o                       # default OpenAI model

# Optional
CATALOGING_PROVIDER=ollama                # default provider (ollama|openai)
LOG_LEVEL=DEBUG                           # logging level
```

## Development Commands

### Local Development
```bash
# Run server
go run main.go

# Run with specific command
go run main.go serve
go run main.go eval eval-ib --sample 5

# Format code
gofmt -w .

# Lint
golangci-lint run

# Visit app
open http://localhost:8888
```

### Docker
```bash
# Build
docker build -t cataloger .

# Run
docker run -p 8888:8888 --env-file .env cataloger
```

### Build
```bash
# Build unified binary
go build -o cataloger .

# Test commands
./cataloger --help
./cataloger serve &
./cataloger eval eval-ib --sample 5
```

## Testing

### Manual Testing
```bash
# Upload via curl (file)
curl -X POST -F "file=@test.jpg" \
  -F "image_type=title_page" \
  -F "provider=ollama" \
  -F "model=mistral-small3.2:24b" \
  http://localhost:8888/api/upload

# Upload via curl (URL)
curl -X POST http://localhost:8888/api/upload \
  -H "Content-Type: application/json" \
  -d '{"image_url":"https://example.com/title.jpg","image_type":"title_page","provider":"openai","model":"gpt-4o"}'

# Eval with verbose logging
./cataloger eval eval-ib --verbose --sample 2
```

## Security Features

- File size limit: 10MB
- Image type validation
- Path traversal prevention (MD5 filenames)
- API keys in environment variables only
- `.env` file gitignored

### Security Considerations
- File size limits prevent DoS attacks
- MD5 hashing prevents path traversal
- Environment variables keep credentials secure
- No sensitive data in git repository

## MARC Generation Prompt

The system uses a detailed prompt that positions the LLM as an expert Library of Congress cataloging librarian with 30+ years experience. The prompt:

- Emphasizes Library of Congress standards and MARC 21 format
- Requests all standard MARC fields (Leader, 008, 020, 100, 245, 260/264, 300, 490, 6XX, 700)
- Uses low temperature (0.1) for consistency
- Includes cataloger's notes for observations
- Handles edge cases (facsimiles, reprints, translations)

## Dataset Structure

### Institutional Books 1.0

- **Size**: ~1TB total, 983,004 books
- **Format**: 9,831 Parquet files (~100 books each, ~100MB per file)
- **Fields**: barcode, title, author, dates, OCR text (page-by-page), identifiers (ISBN/LCCN/OCLC)
- **Access**: Gated dataset, requires HuggingFace account and terms acceptance
- **Loading**: Parquet files loaded with `parquet-go` library, supports sampling

The loader includes debug logging to verify correct data reading:
- File stats (size, row count)
- Batch reading progress
- Sample data from first record
- Total load time

## Known Limitations

- **No authentication** - suitable for internal use only
- **In-memory storage** - sessions lost on restart
- **No rate limiting** - could be abused in production
- **Basic validation** - no advanced security hardening
- **Single image per session** - multi-image support planned
- **No MARC editing** - only view generated records
- **No MARC export** - copy/paste only (export planned for Phase 4)

## Roadmap

### Phase 1: Image Upload ✅ (Complete)
- File and URL upload
- Basic session management
- Simple web interface

### Phase 2: MARC Generation ✅ (Complete)
- LLM vision model integration
- Multi-provider support (Ollama, OpenAI)
- Provider/model selection in UI

### Phase 3: Evaluation ✅ (Complete)
- Institutional Books 1.0 dataset support
- OAI-PMH harvesting from any catalog
- Incremental dataset building with real-time progress
- Field-weighted MARC comparison
- Comprehensive evaluation reports
- Concurrent processing

### Phase 4: Advanced Features (Planned)
- Multi-language support (Spanish, Ukrainian)
- Subject classification using embeddings (Qwen 0.6b)
- MARC export (ISO 2709, MARCXML, JSON)
- Azure/Gemini provider support
- Record editing interface
- Multi-image sessions
- Database persistence
- Authentication and rate limiting
- nDCG scoring for ranked field importance

## Contributing

Follow the Go conventions in `docs/GO_CONVENTIONS.md`:
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Add tests for new features
- Document public APIs

## File Structure

```
cataloger/
├── main.go                      # Unified entry point
├── cmd/
│   ├── serve/main.go           # Web server (internal)
│   └── eval/                   # Eval commands (internal, for reference)
│       ├── main.go             # CLI routing
│       ├── fetch.go            # OAI-PMH harvest
│       ├── enrich.go           # Dataset enrichment
│       ├── run.go              # Batch evaluation
│       ├── report.go           # Report generation
│       └── eval_ib.go          # Institutional Books evaluator
├── internal/
│   ├── cataloging/
│   │   └── service.go          # MARC generation with LLMs
│   ├── ocr/
│   │   └── service.go          # OCR extraction with LLMs
│   ├── evaluation/
│   │   ├── comparison.go       # Field comparison with Levenshtein
│   │   └── dataset.go          # Dataset persistence (incremental)
│   ├── eval/
│   │   └── dataset/
│   │       ├── loader.go       # Parquet/JSONL loader with debug logging
│   │       └── models.go       # InstitutionalBooksRecord struct
│   ├── evalcmd/                # Eval CLI package (used by main.go)
│   │   ├── main.go             # Command routing
│   │   ├── fetch.go            # OAI-PMH implementation
│   │   ├── enrich.go           # Enrichment implementation
│   │   ├── run.go              # Evaluation runner
│   │   ├── report.go           # Report generator
│   │   └── eval_ib.go          # IB evaluator
│   ├── handlers/
│   │   ├── common.go           # Shared handler utilities
│   │   ├── upload.go           # File/URL upload
│   │   ├── sessions.go         # Session CRUD
│   │   ├── image_processing.go # Image handling
│   │   └── static.go           # Static files
│   ├── models/
│   │   └── models.go           # Data structures (with OCRText field)
│   ├── storage/
│   │   └── storage.go          # In-memory store
│   └── utils/
│       ├── files.go            # MD5 hashing
│       └── helper.go           # Error handling
├── static/
│   ├── index.html              # Main UI
│   ├── script.js               # Frontend logic
│   └── styles.css              # Dark theme
├── docs/
│   ├── ARCHITECTURE.md         # Detailed architecture
│   ├── EVALUATION.md           # Evaluation guide
│   ├── PARTIAL_DATASET.md      # Git LFS selective checkout
│   └── GO_CONVENTIONS.md       # Go style guide
├── README.md                   # Main documentation
└── uploads/                    # Uploaded images (gitignored)
```

## Tips for Working with This Codebase

1. **Adding a provider**: Update `cataloging/service.go` with new case in switch statement
2. **Changing prompt**: Edit `buildMARCPrompt()` in `cataloging/service.go`
3. **Frontend changes**: Edit `static/` files directly, no build step needed
4. **Testing locally**: Use Ollama for faster iteration (no API costs)
5. **Production**: Use OpenAI for better quality MARC records
6. **Debugging dataset loading**: Use `--verbose` flag to see debug logs
7. **New eval command**: Add to `internal/evalcmd/` and update switch in `main.go`

## Common Issues

**"model not found"**: Make sure model is pulled in Ollama or specified correctly
**"connection refused"**: Check OLLAMA_URL points to running instance
**"API key error"**: Verify OPENAI_API_KEY is set in .env
**File upload fails**: Check file size < 10MB and is valid image format
**Empty parquet data**: Ensure struct has `parquet:"field_name"` tags matching schema
**Slow evaluation**: Check LLM response time, not dataset loading (loading is sub-second)
