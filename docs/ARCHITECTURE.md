
## Architecture

### Backend (Go)
- **main.go**: HTTP server setup listening on port 8888
- **internal/handlers/**: HTTP request handlers
  - `common.go`: Handler struct and shared utilities
  - `upload.go`: Image upload and URL processing
  - `image_processing.go`: Image download and dimension extraction
  - `sessions.go`: Session management endpoints
  - `static.go`: Static file serving
- **internal/models/models.go**: Data structures
  - `CatalogSession`: Represents a cataloging session with uploaded images
  - `ImageItem`: Represents an uploaded book image (cover/title_page/copyright)
- **internal/storage/**: In-memory session storage
- **internal/utils/**: Error handling and image utilities

### Frontend (Vanilla JS)
- **static/index.html**: Simple upload interface
- **static/script.js**: Upload handling for files and URLs
- **static/styles.css**: Dark theme styling

### Current Data Flow
1. User uploads image via file or URL
2. Image downloaded/read and saved to `uploads/` directory
3. Image dimensions extracted
4. Session created with image metadata
5. Session stored in-memory
6. Session ID returned to client
7. Sessions displayed in list on homepage

### Planned Architecture (Future)

#### Backend Additions (Planned)
- **internal/cataloging/service.go**: Core cataloging pipeline
  - Image analysis using vision models (VIT, SigLIP)
  - Title page template matching for different formats and languages
  - LLM-based metadata extraction from cover, title page, and copyright statement
  - Subject classification using embeddings (Qwen 0.6b) and zero-shot classification
  - MARC record generation and validation
- **internal/providers/service.go**: Multi-provider LLM abstraction
  - Wraps github.com/lehigh-university-libraries/htr package
  - Supports: OpenAI, Azure OpenAI, Google Gemini, Ollama
  - Default provider: Ollama (can be changed via CATALOGING_PROVIDER env var)
- **internal/classification/**: Subject and image classification
  - Embedding-based classification for controlled vocabularies
  - VIT fine-tuning for title page format detection
  - Zero-shot classification for subjects
- **internal/metrics/**: nDCG scoring for evaluation against professional catalogs
- **cmd/eval/**: CLI for evaluating MARC generation against known catalogs

#### Frontend Additions (Planned)
- MARC record editing interface
- Template selection for different title page formats
- Language selection (English, Spanish, Ukrainian, etc.)
- Real-time validation
- Export functionality

### Planned Cataloging Pipeline Flow
1. Multi-image upload (cover + title page + copyright statement)
2. Template matching to identify title page format (e.g., facsimile vs. original)
3. Vision model analysis to extract visual features
4. LLM-based metadata extraction from images
5. Subject classification using embeddings and zero-shot classification
6. MARC record generation with controlled vocabulary mapping
7. Validation and human review interface
8. Export MARC or save to repository

### Key Design Decisions

#### Why In-Memory Storage?
Currently using in-memory storage for simplicity in early development. Future versions may add:
- Database persistence (PostgreSQL/SQLite)
- Session expiration
- User authentication

#### Why Classification over Generation for Subjects?
Classification models provide:
- Consistent results for controlled vocabularies
- No hallucinations
- Faster inference
- Easier evaluation

#### Title Page Template Matching
The planned template matching system will identify:
- Original vs. facsimile editions
- Historical reproductions (18th/19th century census reports, etc.)
- Different language layouts (English, Spanish, Ukrainian, etc.)
- Publisher and imprint variations
