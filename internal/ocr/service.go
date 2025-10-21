package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
)

// Service handles OCR extraction from images
type Service struct{}

// NewService creates a new OCR service
func NewService() *Service {
	return &Service{}
}

// ExtractTextFromImage extracts text from an image using LLM vision capabilities
// This is faster and more reliable than traditional OCR for title pages
func (s *Service) ExtractTextFromImage(imagePath, provider, model string) (string, error) {
	// Set defaults if not provided
	if provider == "" {
		provider = os.Getenv("CATALOGING_PROVIDER")
		if provider == "" {
			provider = "ollama"
		}
	}

	if model == "" {
		model = s.getDefaultModel(provider)
	}

	switch provider {
	case "openai":
		return s.extractWithOpenAI(imagePath, model)
	case "ollama":
		return s.extractWithOllama(imagePath, model)
	default:
		return "", fmt.Errorf("unsupported OCR provider: %s", provider)
	}
}

func (s *Service) getDefaultModel(provider string) string {
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
	default:
		return ""
	}
}

func (s *Service) buildOCRPrompt() string {
	return `You are performing OCR (Optical Character Recognition) on a book title page image.

Your task is to extract ALL visible text from the image exactly as it appears, preserving:
- Line breaks and formatting
- Capitalization
- Punctuation
- Special characters
- Order of text elements

INSTRUCTIONS:
1. Read the image carefully from top to bottom
2. Transcribe every piece of visible text
3. Preserve the original line breaks
4. Do not add any interpretation, commentary, or explanations
5. Do not skip any text, no matter how small or decorative
6. If text is partially obscured or unclear, transcribe what you can see and use [?] for illegible portions

OUTPUT FORMAT:
Provide ONLY the extracted text. Do not include phrases like "Here is the text:" or "The image contains:".
Start immediately with the transcribed text from the title page.

Example output:
THE ADVENTURES OF
TOM SAWYER

By Mark Twain

New York
Harper & Brothers Publishers
1876`
}

func (s *Service) extractWithOllama(imagePath, model string) (string, error) {
	ollamaHost := os.Getenv("OLLAMA_URL")
	if ollamaHost == "" {
		ollamaHost = os.Getenv("OLLAMA_HOST")
	}
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image for OCR: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Prepare Ollama request for OCR
	prompt := s.buildOCRPrompt()

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"images": []string{base64Image},
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.0, // Zero temperature for exact OCR
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OCR request: %w", err)
	}

	// Call Ollama API
	resp, err := http.Post(
		ollamaHost+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API for OCR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama OCR API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama OCR response: %w", err)
	}

	slog.Info("Extracted OCR text", "provider", "ollama", "length", len(ollamaResp.Response))
	return ollamaResp.Response, nil
}

func (s *Service) extractWithOpenAI(imagePath, model string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image for OCR: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Prepare OpenAI request for OCR
	prompt := s.buildOCRPrompt()

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": prompt,
					},
					{
						"type": "image_url",
						"image_url": map[string]string{
							"url": "data:image/jpeg;base64," + base64Image,
						},
					},
				},
			},
		},
		"max_tokens":  2000,
		"temperature": 0.0, // Zero temperature for exact OCR
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal OCR request: %w", err)
	}

	// Call OpenAI API
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create OCR request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API for OCR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openAI OCR API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI OCR response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no OCR response from OpenAI")
	}

	ocrText := openaiResp.Choices[0].Message.Content
	slog.Info("Extracted OCR text", "provider", "openai", "model", model, "length", len(ocrText))
	return ocrText, nil
}
