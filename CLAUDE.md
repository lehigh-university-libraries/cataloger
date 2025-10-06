# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cataloger is a web-based book metadata cataloging tool that generates MARC records from images of book title pages using vision-capable LLMs (Ollama or OpenAI).

**Current Status**: MVP complete with working MARC generation from title page images.

## 📚 Critical Documentation References
- **Go Conventions**: `./docs/GO_CONVENTIONS.md` 📋
- **Project Architecture**: `./docs/ARCHITECTURE.md`
- **Security Policy**: `./SECURITY.md`
- **Contributing Guide**: `./CONTRIBUTING.md`

## Current Features (Implemented)

### ✅ Image Upload
- File upload (up to 10MB)
- URL upload
- Three image types: cover, title_page, copyright
- MD5-based file storage to prevent duplicates

### ✅ MARC Generation
- Automatic MARC record generation from title page images
- Multi-provider support (Ollama, OpenAI)
- Expert-level prompt designed for Library of Congress standards
- Low temperature (0.1) for consistent, factual output

### ✅ Provider Configuration
- **Ollama**: Local or remote instances via OLLAMA_URL
- **OpenAI**: GPT-4o, GPT-4o-mini, GPT-4-turbo
- Dynamic model selection per upload
- Provider/model info stored with sessions

### ✅ Session Management
- In-memory session storage
- View uploaded images and generated MARC records
- Click sessions to view details in modal

### ✅ Frontend
- Clean dark-themed UI
- Provider and model dropdowns
- Image type selection
- Real-time session list
- Modal display of images and MARC records

## Architecture

### Backend (Go)
- **main.go**: HTTP server on port 8888
- **internal/cataloging/service.go**: MARC generation with multi-provider support
  - `GenerateMARCFromImage()`: Main entry point
  - `generateWithOllama()`: Ollama API integration
  - `generateWithOpenAI()`: OpenAI Vision API integration
  - `buildMARCPrompt()`: Expert cataloger prompt
- **internal/handlers/**: HTTP endpoints
  - `upload.go`: File and URL upload with validation
  - `sessions.go`: Session CRUD operations
  - `image_processing.go`: Image download and processing
  - `static.go`: Static file serving
  - `common.go`: Shared utilities and session creation
- **internal/models/models.go**: Data structures
  - `CatalogSession`: Session with images, MARC, provider/model
  - `ImageItem`: Image metadata
- **internal/storage/**: In-memory session store
- **internal/utils/**: MD5 hashing and error handling

### Frontend (Vanilla JS)
- **static/index.html**: Single-page app
- **static/script.js**: Upload, session management, modal display
- **static/styles.css**: Dark theme with provider selection UI

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
```

## Development Commands

### Local Development
```bash
# Run server
go run main.go

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

### Testing
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
```

## Security Features

- File size limit: 10MB
- Image type validation
- Path traversal prevention (MD5 filenames)
- API keys in environment variables only
- `.env` file gitignored

See `SECURITY.md` for complete security considerations.

## MARC Generation Prompt

The system uses a detailed prompt that positions the LLM as an expert Library of Congress cataloging librarian with 30+ years experience. The prompt:

- Emphasizes Library of Congress standards and MARC 21 format
- Requests all standard MARC fields (Leader, 008, 020, 100, 245, 260/264, 300, 490, 6XX, 700)
- Uses low temperature (0.1) for consistency
- Includes cataloger's notes for observations
- Handles edge cases (facsimiles, reprints, translations)

## Known Limitations

- **No authentication** - suitable for internal use only
- **In-memory storage** - sessions lost on restart
- **No rate limiting** - could be abused in production
- **Basic validation** - no advanced security hardening
- **Single image per session** - multi-image support planned
- **No MARC editing** - only view generated records
- **No MARC export** - copy/paste only

## Roadmap

### Phase 2: Enhanced Metadata (Planned)
- Multi-image sessions (cover + title + copyright)
- MARC record editing interface
- MARC export (ISO 2709, MARCXML, JSON)
- Template matching for edge cases
- Subject classification using embeddings

### Phase 3: Evaluation (Planned)
- nDCG scoring CLI (similar to ../htr)
- Compare against professional catalogs
- Precision/recall metrics
- Test dataset management

### Phase 4: Production Ready (Planned)
- Authentication and authorization
- Rate limiting
- Database persistence
- Session expiration
- Audit logging
- Azure/Gemini provider support
- Multi-language support (Spanish, Ukrainian)

## Contributing

See `CONTRIBUTING.md` for:
- Development setup
- Code standards
- Pull request process
- How to add new providers

## File Structure

```
cataloger/
├── main.go                      # Entry point
├── internal/
│   ├── cataloging/
│   │   └── service.go          # MARC generation with LLMs
│   ├── handlers/
│   │   ├── common.go           # Shared handler utilities
│   │   ├── upload.go           # File/URL upload
│   │   ├── sessions.go         # Session CRUD
│   │   ├── image_processing.go # Image handling
│   │   └── static.go           # Static files
│   ├── models/
│   │   └── models.go           # Data structures
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
│   └── GO_CONVENTIONS.md       # Go style guide
├── SECURITY.md                 # Security considerations
├── CONTRIBUTING.md             # Contribution guide
├── README.md                   # User-facing docs
├── sample.env                  # Example configuration
└── uploads/                    # Uploaded images (gitignored)
```

## Tips for Working with This Codebase

1. **Adding a provider**: Update `cataloging/service.go` with new case in switch statement
2. **Changing prompt**: Edit `buildMARCPrompt()` in `cataloging/service.go`
3. **Frontend changes**: Edit `static/` files directly, no build step needed
4. **Testing locally**: Use Ollama for faster iteration (no API costs)
5. **Production**: Use OpenAI for better quality MARC records

## Common Issues

**"model not found"**: Make sure model is pulled in Ollama or specified correctly
**"connection refused"**: Check OLLAMA_URL points to running instance
**"API key error"**: Verify OPENAI_API_KEY is set in .env
**File upload fails**: Check file size < 10MB and is valid image format
