# Cataloger

A web-based book metadata cataloging tool that generates MARC records from images of book title pages using LLM vision models.

## Current Status

**MVP Complete** - Basic MARC generation is working

The application currently supports:
- Uploading book images via web interface (file upload)
- Uploading book images via URL
- Three image types: Cover, Title Page, Copyright Page
- **MARC record generation from title page images**
- **Multi-provider support: Ollama (local) and OpenAI**
- **Provider and model selection in UI**
- Simple session management

**Future phases** will add evaluation tools, multi-language support, and advanced features.

## Quick Start

```bash
docker compose up --build -d

# Server runs on http://localhost:8888
```

## Usage

1. Open http://localhost:8888 in your browser
2. Select provider (Ollama or OpenAI) and model
3. Select image type (Cover, Title Page, or Copyright Page)
4. Either:
   - Upload an image file from your computer, OR
   - Enter an image URL
5. View generated MARC record in the session modal

## API Endpoints

- `POST /api/upload` - Upload image (form data or JSON)
- `GET /api/sessions` - List all sessions
- `GET /api/sessions/{id}` - Get session details
- `GET /healthcheck` - Health check

## Configuration

See [sample.env](./sample.env) for environment variable configuration:

**Ollama (default)**
- `OCR_PROVIDER=ollama` (or omit)
- `OLLAMA_URL=http://localhost:11434` (default)
- `OLLAMA_MODEL=mistral-small3.2:24b` (default)

**OpenAI**
- `OCR_PROVIDER=openai`
- `OPENAI_API_KEY=sk-...` (required)
- `OPENAI_MODEL=gpt-4o` (default)

## Project Vision

The full project will:
- Analyze book covers, title pages, and copyright statements
- Generate MARC metadata records suitable for library catalogs
- Use classification models for controlled vocabularies
- Support multiple languages (English, Spanish, Ukrainian, etc.)
- Handle edge cases like facsimile editions and historical reproductions
- Evaluate output quality using nDCG scoring against professional catalogs

### Why Classification Models?

For controlled vocabularies (subjects, classifications), we plan to use classification models instead of generative ones because:
- Consistent, reproducible results
- Direct mapping to controlled vocabularies
- Avoids hallucinations requiring manual correction
- Faster inference
- Easier to evaluate and improve

## Development Roadmap

### Phase 1: Image Upload ✅ (Complete)
- File and URL upload
- Basic session management
- Simple web interface

### Phase 2: MARC Generation ✅ (MVP Complete)
- LLM vision model integration (Ollama, OpenAI)
- MARC record generation from title pages
- Multi-provider support
- Provider/model selection in UI

### Phase 3: Evaluation (Planned)
- Evaluation CLI similar to ../htr
- nDCG scoring against professional catalogs
- Precision/recall metrics for metadata fields

### Phase 4: Advanced Features (Planned)
- Multi-language support
- Subject classification using embeddings (Qwen 0.6b)
- Fine-tuned models for specific use cases
- Export and repository integration

## License

Apache 2.0
