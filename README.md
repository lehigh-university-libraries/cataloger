# Cataloger

A tool for evaluating LLM-based metadata extraction from book images.

## Quick Start

```bash
# Build
make build

# Evaluate with Institutional Books dataset
./cataloger eval ib --sample 10
```

## Features

- ✅ **Evaluation Tools** - Systematic quality assessment with the Institutional Books 1.0 dataset.
- ✅ **Multi-Provider** - Ollama (local), OpenAI, and Google Gemini support.

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

**Google Gemini**
```bash
GEMINI_API_KEY=your-api-key
GEMINI_MODEL=gemini-3.0-pro-preview
```

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
./cataloger eval ib --sample 10

# Specific file
./cataloger eval ib \
  --dataset ./institutional-books-1.0/data/train-00001-of-09831.parquet \
  --sample 10

# With OpenAI
./cataloger eval ib --provider openai --model gpt-4o --sample 10

# With Gemini
./cataloger eval ib --provider gemini --model gemini-3.0-pro-preview --sample 10

# Verbose output for debugging
./cataloger eval ib --verbose --sample 5
```

**Batch Evaluation**

```bash
# Test first 5 files
for i in {0..4}; do
  padded=$(printf "%05d" $i)
  ./cataloger eval ib \
    --dataset ./institutional-books-1.0/data/train-${padded}-of-09831.parquet \
    --sample 50 \
    --output-json results_${padded}.json
done

# Aggregate results
jq '.OverallAccuracy' results_*.json
```

See [docs/PARTIAL_DATASET.md](./docs/PARTIAL_DATASET.md) for more download patterns.

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
│   └── eval/                 # Eval commands (internal use)
├── internal/
│   ├── cataloging/           # Metadata extraction
│   ├── eval/
│   │   └── dataset/          # Dataset loaders (Parquet, JSONL)
│   ├── evalcmd/              # Eval CLI implementation
│   └── ocr/                  # OCR extraction
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
- **[docs/PARTIAL_DATADATASET.md](./docs/PARTIAL_DATASET.md)** - Git LFS selective download guide
- **[docs/GO_CONVENTIONS.md](./docs/GO_CONVENTIONS.md)** - Go coding standards

## License

Apache 2.0
