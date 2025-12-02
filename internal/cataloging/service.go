package cataloging

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/lehigh-university-libraries/cataloger/internal/gemini"
	"github.com/lehigh-university-libraries/cataloger/internal/ollama"
	"github.com/lehigh-university-libraries/cataloger/internal/openai"
	"github.com/lehigh-university-libraries/cataloger/internal/providers"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// initProvider initializes an LLM provider based on the provider type
func (s *Service) initProvider(providerType string) (providers.Provider, error) {
	switch providerType {
	case "ollama":
		return ollama.New(), nil
	case "openai":
		return openai.New(), nil
	case "gemini":
		return gemini.New(), nil
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", providerType)
	}
}

// ExtractMetadataFromOCR extracts bibliographic metadata from OCR text
func (s *Service) ExtractMetadataFromOCR(ocrText, provider, model string) (string, error) {
	// Set defaults if not provided
	if provider == "" {
		provider = os.Getenv("CATALOGING_PROVIDER")
		if provider == "" {
			provider = "ollama"
		}
	}

	if model == "" {
		model = s.GetDefaultModel(provider)
	}

	// Initialize provider
	llmProvider, err := s.initProvider(provider)
	if err != nil {
		return "", err
	}

	// Build prompt
	systemPrompt := s.buildMetadataExtractionPrompt()
	userPrompt := fmt.Sprintf("Here is the OCR text from a book title page:\n\n%s\n\nExtract the bibliographic metadata as JSON.", ocrText)
	fullPrompt := systemPrompt + "\n\n" + userPrompt

	// Create config
	config := providers.Config{
		Model:       model,
		Temperature: 0.1,
		Prompt:      fullPrompt,
	}

	// Extract metadata using provider
	ctx := context.Background()
	metadataJSON, err := llmProvider.ExtractText(ctx, config)
	if err != nil {
		return "", fmt.Errorf("failed to extract metadata with %s: %w", provider, err)
	}

	slog.Info("Extracted metadata", "provider", provider, "model", model, "length", len(metadataJSON))
	return metadataJSON, nil
}

func (s *Service) GetDefaultModel(provider string) string {
	switch provider {
	case "openai":
		model := os.Getenv("OPENAI_MODEL")
		if model == "" {
			return "gpt-4o"
		}
		return model
	case "ollama":
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			return "mistral-small3.2:24b"
		}
		return model
	case "gemini":
		model := os.Getenv("GEMINI_MODEL")
		if model == "" {
			return "gemini-1.5-flash-latest"
		}
		return model
	default:
		return ""
	}
}

// buildMetadataExtractionPrompt creates a prompt for extracting bibliographic metadata
func (s *Service) buildMetadataExtractionPrompt() string {
	return `You are an expert bibliographic metadata cataloger. Extract structured metadata from the OCR text of a book title page.

INSTRUCTIONS:
1. Carefully analyze ALL information in the OCR text
2. Extract the following bibliographic fields:
   - title: Full title of the work (include subtitle if present)
   - author: Primary author(s) name(s)
   - publisher: Publisher name
   - publication_date: Year of publication
   - publication_city: City where published
   - edition: Edition statement (if present, e.g., "2nd ed.", "Rev. ed.")
   - isbn: ISBN numbers (array, if present)
   - language: Primary language of the work (ISO 639-3 code if possible, or full name)
   - subject: Main subject or topic
   - genre: Genre or form (e.g., "Fiction", "Biography", "Reference")
   - series: Series information (if part of a series)

3. For missing fields, use empty string "" or empty array [] for ISBN
4. Be precise and extract exactly what is shown in the OCR text
5. Do not invent or infer information that isn't present

OUTPUT FORMAT:
Respond with ONLY a JSON object:

{
  "title": "...",
  "author": "...",
  "publisher": "...",
  "publication_date": "...",
  "publication_city": "...",
  "edition": "...",
  "isbn": ["..."],
  "language": "...",
  "subject": "...",
  "genre": "...",
  "series": "...",
  "notes": "Any observations or uncertainties"
}

Be thorough and accurate. Extract only what is clearly present in the OCR text.`
}
