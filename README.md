# Cataloger

A web-based book metadata cataloging tool that generates MARC records from images of book title pages using LLM vision models.

## Quick Start

### Web Server

```bash
make serve

# Or with Docker
docker compose up --build -d

# Server runs on http://localhost:8888
```

### Evaluation CLI

```bash
# Build
make build

# Evaluate with Institutional Books dataset
./cataloger eval eval-ib --sample 10

# Fetch evaluation data from OAI-PMH
./cataloger eval fetch --url https://folio.example.edu/oai --limit 100

# Run evaluation on custom dataset
./cataloger eval run --dataset ./eval_data --provider ollama

# Generate report
./cataloger eval report --results ./eval_results
```

## Features

- ✅ **Web Interface** - Upload book images via file or URL
- ✅ **MARC Generation** - Automatic MARC record generation from title pages
- ✅ **Multi-Provider** - Ollama (local) and OpenAI support
- ✅ **Evaluation Tools** - Systematic quality assessment with multiple dataset sources
- ✅ **Incremental Harvesting** - OAI-PMH support with resumption tokens

## Configuration

Create a `.env` file (see `sample.env`):

**Ollama (default)**
```bash
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=mistral-small3.2:24b
```

**OpenAI**
```bash
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o
```

## Web Interface Usage

1. Open http://localhost:8888
2. Select provider (Ollama or OpenAI) and model
3. Select image type (Cover, Title Page, or Copyright Page)
4. Upload an image file or enter an image URL
5. View generated MARC record in the session modal

## Evaluation

### Institutional Books 1.0 Dataset

The fastest way to evaluate is with the Institutional Books dataset (983K public domain books).

**Setup (One Time)**

```bash
# 1. Install Git LFS
brew install git-lfs && git lfs install

# 2. Accept terms at: https://huggingface.co/datasets/instdin/institutional-books-1.0

# 3. Clone WITHOUT downloading files (dataset is 1TB!)
GIT_LFS_SKIP_SMUDGE=1 git clone https://huggingface.co/datasets/instdin/institutional-books-1.0

# 4. Download just what you need (each file ~100MB with ~100 books)
cd institutional-books-1.0
git lfs pull --include="data/train-00000-of-09831.parquet"
git lfs pull --include="data/train-00001-of-09831.parquet"
cd ..
```

**Run Evaluation**

```bash
# Build
go build -o cataloger .

# Quick test with defaults (first parquet file)
./cataloger eval eval-ib --sample 10

# Specific file
./cataloger eval eval-ib \
  --dataset ./institutional-books-1.0/data/train-00001-of-09831.parquet \
  --sample 10

# With OpenAI
./cataloger eval eval-ib --provider openai --model gpt-4o --sample 10

# Verbose output for debugging
./cataloger eval eval-ib --verbose --sample 5
```

**Batch Evaluation**

```bash
# Test first 5 files
for i in {0..4}; do
  padded=$(printf "%05d" $i)
  ./cataloger eval eval-ib \
    --dataset ./institutional-books-1.0/data/train-${padded}-of-09831.parquet \
    --sample 50 \
    --output-json results_${padded}.json
done

# Aggregate results
jq '.OverallAccuracy' results_*.json
```

**Download Patterns**

```bash
cd institutional-books-1.0

# Single file (~100MB)
git lfs pull --include="data/train-00000-of-09831.parquet"

# First 10 files (~1GB)
git lfs pull --include="data/train-0000[0-9]-of-09831.parquet"

# Specific range (files 0-4)
for i in {0..4}; do
  git lfs pull --include="data/train-$(printf "%05d" $i)-of-09831.parquet"
done

# Evenly distributed sample across dataset (30 files ~3GB)
for i in 0 100 200 400 800 1600 3200 6400 9830; do
  padded=$(printf "%05d" $i)
  git lfs pull --include="data/train-${padded}-of-09831.parquet"
done
```

See [docs/PARTIAL_DATASET.md](./docs/PARTIAL_DATASET.md) for more download patterns.

### OAI-PMH Evaluation

Build your own evaluation dataset from any OAI-PMH endpoint:

```bash
# Fetch records from FOLIO/VuFind/Koha
./cataloger eval fetch \
  --url https://folio.example.edu/oai \
  --limit 100 \
  --sleep 2

# Enrich with images and MARCXML
./cataloger eval enrich \
  --dataset ./eval_data \
  --output ./enriched_data

# Run evaluation
./cataloger eval run \
  --dataset ./eval_data \
  --provider ollama \
  --model mistral-small3.2:24b \
  --concurrency 4

# Generate report
./cataloger eval report --results ./eval_results --format text
```

**Features:**
- Incremental saving (records saved immediately during harvest)
- Resumption token support with configurable sleep delays
- Automatic filtering (books with ISBN only, excludes deleted/suppressed)
- Field-weighted MARC comparison
- Text, JSON, and CSV reports

See [docs/EVALUATION.md](./docs/EVALUATION.md) for detailed documentation.

## API Endpoints

- `POST /api/upload` - Upload image (multipart form or JSON)
- `GET /api/sessions` - List all sessions
- `GET /api/sessions/{id}` - Get session details
- `PUT /api/sessions/{id}` - Update session
- `GET /healthcheck` - Health check

## Development

```bash
# Run locally
go run main.go

# Format code
gofmt -w .

# Lint
golangci-lint run

# Run tests
go test ./...
```

## Project Structure

```
cataloger/
├── main.go                    # Unified CLI entry point
├── cmd/
│   ├── serve/                # Web server (internal use)
│   └── eval/                 # Eval commands (internal use)
├── internal/
│   ├── cataloging/           # MARC generation
│   ├── evaluation/           # MARC comparison engine
│   ├── eval/
│   │   └── dataset/          # Dataset loaders (Parquet, JSONL)
│   ├── evalcmd/              # Eval CLI implementation
│   ├── handlers/             # HTTP handlers
│   ├── models/               # Data structures
│   ├── ocr/                  # OCR extraction
│   └── storage/              # In-memory session store
├── static/                   # Web interface
├── docs/                     # Documentation
│   ├── ARCHITECTURE.md       # Technical architecture
│   ├── EVALUATION.md         # Evaluation guide
│   ├── PARTIAL_DATASET.md    # Git LFS selective checkout
│   └── GO_CONVENTIONS.md     # Go style guide
└── uploads/                  # Uploaded images
```

## Documentation

- **[docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)** - System architecture and design
- **[docs/EVALUATION.md](./docs/EVALUATION.md)** - Detailed evaluation documentation
- **[docs/PARTIAL_DATASET.md](./docs/PARTIAL_DATASET.md)** - Git LFS selective download guide
- **[docs/GO_CONVENTIONS.md](./docs/GO_CONVENTIONS.md)** - Go coding standards

## Roadmap

### Phase 1: Image Upload ✅
- File and URL upload
- Basic session management
- Web interface

### Phase 2: MARC Generation ✅
- LLM vision model integration
- Multi-provider support (Ollama, OpenAI)
- Provider/model selection in UI

### Phase 3: Evaluation ✅
- Institutional Books 1.0 dataset support
- OAI-PMH harvesting
- Field-weighted MARC comparison
- Concurrent evaluation processing

### Phase 4: Advanced Features (Planned)
- Multi-language support (Spanish, Ukrainian)
- Subject classification using embeddings
- MARC export (ISO 2709, MARCXML, JSON)
- Azure/Gemini provider support
- Record editing interface

## Known Limitations

- **No authentication** - suitable for internal use only
- **In-memory storage** - sessions lost on restart
- **No rate limiting** - could be abused in production
- **Single image per session** - multi-image support planned

## License

Apache 2.0
