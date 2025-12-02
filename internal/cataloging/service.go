package cataloging

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

// GenerateMARCFromImage generates a MARC record from a book title page image
func (s *Service) GenerateMARCFromImage(imagePath, provider, model string) (string, error) {
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
		return s.generateWithOpenAI(imagePath, model)
	case "ollama":
		return s.generateWithOllama(imagePath, model)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
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
		model = s.getDefaultModel(provider)
	}

	switch provider {
	case "openai":
		return s.extractMetadataWithOpenAI(ocrText, model)
	case "ollama":
		return s.extractMetadataWithOllama(ocrText, model)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
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

func (s *Service) generateWithOllama(imagePath, model string) (string, error) {
	// Get Ollama host
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
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Prepare Ollama request
	prompt := s.buildMARCPromptForImage()

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"images": []string{base64Image},
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1, // Low temperature for consistent, factual output
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call Ollama API
	resp, err := http.Post(
		ollamaHost+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var ollamaResp struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	rawResponse := ollamaResp.Response
	marcRecord := s.extractMARCFromResponse(rawResponse)
	slog.Info("Generated MARC record", "length", len(marcRecord))
	return marcRecord, nil
}

// buildMARCPrompt generates a MARC cataloging prompt based on the source type
func (s *Service) buildMARCPrompt(sourceType string) string {
	// Source-specific instructions
	var sourceInstructions string
	var missingFieldNote string
	var generationNote string

	switch sourceType {
	case "ocr":
		sourceInstructions = "Carefully analyze ALL information in the OCR text including:"
		missingFieldNote = "If any information is missing from the OCR text, note it as \"[not available in OCR]\""
		generationNote = "Include a 500 note explaining that this record was \"Generated from OCR text of title page\""
	case "image":
		sourceInstructions = "Carefully examine ALL information visible on the title page including:"
		missingFieldNote = "If any information is not visible on the title page, note it as \"[not visible on title page]\""
		generationNote = "Include a 500 note explaining that this record was \"Generated from title page image\""
	default:
		sourceInstructions = "Carefully examine ALL information in the source material including:"
		missingFieldNote = "If any information is not available, note it as \"[not available]\""
		generationNote = "Include a 500 note explaining how this record was generated"
	}

	return fmt.Sprintf(`You are an expert cataloging librarian working for The Library of Congress with over 30 years of experience creating MARC (Machine-Readable Cataloging) records. You are recognized internationally for your expertise in bibliographic description and have trained countless librarians in proper MARC cataloging practices.

Your task is to analyze the book title page %s and create a complete, professional MARC 21 bibliographic record that meets Library of Congress cataloging standards.

INSTRUCTIONS:
1. %s
   - Main title and subtitle
   - Author(s) and their roles (author, editor, translator, etc.)
   - Publisher name and location
   - Publication date
   - Edition statement (if present)
   - Series information (if present)
   - %s
   - Any other relevant bibliographic details

2. Create a MARC record using standard MARC 21 format with the following key fields:
   - Leader (record structure)
   - 008 (fixed-length data elements)
   - 020 (ISBN if available)
   - 100/110/111 (Main entry - personal/corporate/meeting name)
   - 245 (Title statement with proper indicators)
   - 250 (Edition statement)
   - 260/264 (Publication information)
   - 300 (Physical description - you may indicate "to be determined" for pagination)
   - 490/8XX (Series statement if applicable)
   - 6XX (Subject headings - provide appropriate LCSH terms based on title/content)
   - 700 (Added entries for additional authors/contributors)

3. Format your response as a proper MARC record using the mnemonic format (tag, indicators, subfields).

4. Follow these cataloging best practices:
   - Use proper capitalization (only first word and proper nouns in titles)
   - Include correct MARC indicators for each field
   - Use appropriate subfield codes ($a, $b, $c, etc.)
   - %s
   - Make reasonable inferences for subject headings based on the title and content
   - %s

5. If you identify any special characteristics (facsimile edition, reprint, translation, etc.), make sure to include appropriate MARC fields and notes.

OUTPUT FORMAT:
Respond with ONLY a JSON object in the following format:

{
  "marc": "=LDR  [leader string]\n=008  [fixed field data]\n=020  [ISBN data]\n...",
  "notes": "Brief cataloger's notes with any observations or uncertainties"
}

The "marc" field should contain the complete MARC record with each field on a new line (using \n), in numerical order, using the format =TAG  INDICATORS$SUBFIELDS.

The "notes" field should contain a brief summary of any observations, uncertainties, or special characteristics identified during cataloging.

Be thorough, precise, and follow Library of Congress Rule Interpretations (LCRIs) and MARC 21 standards exactly.`,
		map[string]string{"ocr": "OCR text", "image": "image provided"}[sourceType],
		sourceInstructions,
		map[string]string{"ocr": "ISBN (if present)", "image": "ISBN (if visible)"}[sourceType],
		missingFieldNote,
		generationNote,
	)
}

func (s *Service) buildMARCPromptForImage() string {
	return s.buildMARCPrompt("image")
}

// extractMARCFromResponse parses the JSON response and extracts the MARC record
// Falls back to returning the full response if JSON parsing fails
func (s *Service) extractMARCFromResponse(response string) string {
	// Try to parse as JSON
	var result struct {
		MARC  string `json:"marc"`
		Notes string `json:"notes"`
	}

	// Trim any markdown code blocks
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("Failed to parse JSON response, using raw output", "error", err)
		// Fallback: try to extract MARC record section from non-JSON response
		return extractMARCFromPlainText(response)
	}

	// Successfully parsed JSON
	if result.MARC == "" {
		slog.Warn("JSON parsed but MARC field is empty, using raw output")
		return response
	}

	slog.Debug("Successfully extracted MARC from JSON response", "notes", result.Notes)
	return result.MARC
}

// extractMARCFromPlainText attempts to extract MARC from plain text response
// This is a fallback for when the LLM doesn't return proper JSON
func extractMARCFromPlainText(response string) string {
	// Look for "MARC RECORD:" section
	if idx := strings.Index(response, "MARC RECORD:"); idx != -1 {
		response = response[idx+len("MARC RECORD:"):]
	}

	// Look for "=LDR" or "=001" as start of MARC
	lines := strings.Split(response, "\n")
	var marcLines []string
	inMARC := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Start collecting when we see a MARC field
		if strings.HasPrefix(line, "=") || (inMARC && len(line) > 0) {
			inMARC = true
			marcLines = append(marcLines, line)
		}
		// Stop if we hit a section marker like "CATALOGER'S NOTES" or "Notes:"
		if inMARC && (strings.HasPrefix(strings.ToUpper(line), "CATALOGER") ||
			strings.HasPrefix(strings.ToUpper(line), "NOTES:") ||
			strings.HasPrefix(line, "---")) {
			break
		}
	}

	if len(marcLines) > 0 {
		return strings.Join(marcLines, "\n")
	}

	// Last resort: return the whole response
	return response
}

func (s *Service) generateWithOpenAI(imagePath, model string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Read and encode image
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image: %w", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(imageData)

	// Prepare OpenAI request
	prompt := s.buildMARCPromptForImage()

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
		"max_tokens":  4000,
		"temperature": 0.1,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Call OpenAI API
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openAI API returned status %d: %s", resp.StatusCode, string(body))
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
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	rawResponse := openaiResp.Choices[0].Message.Content
	marcRecord := s.extractMARCFromResponse(rawResponse)
	slog.Info("Generated MARC record", "provider", "openai", "model", model, "length", len(marcRecord))
	return marcRecord, nil
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

// extractMetadataWithOllama extracts metadata using Ollama
func (s *Service) extractMetadataWithOllama(ocrText, model string) (string, error) {
	ollamaHost := os.Getenv("OLLAMA_URL")
	if ollamaHost == "" {
		ollamaHost = os.Getenv("OLLAMA_HOST")
	}
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	systemPrompt := s.buildMetadataExtractionPrompt()
	userPrompt := fmt.Sprintf("Here is the OCR text from a book title page:\n\n%s\n\nExtract the bibliographic metadata as JSON.", ocrText)

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": systemPrompt + "\n\n" + userPrompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1,
		},
		"format": "json", // Request JSON format
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata request: %w", err)
	}

	resp, err := http.Post(
		ollamaHost+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	slog.Info("Extracted metadata", "provider", "ollama", "length", len(ollamaResp.Response))
	return ollamaResp.Response, nil
}

// extractMetadataWithOpenAI extracts metadata using OpenAI
func (s *Service) extractMetadataWithOpenAI(ocrText, model string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	systemPrompt := s.buildMetadataExtractionPrompt()
	userPrompt := fmt.Sprintf("Here is the OCR text from a book title page:\n\n%s\n\nExtract the bibliographic metadata as JSON.", ocrText)

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": systemPrompt,
			},
			{
				"role":    "user",
				"content": userPrompt,
			},
		},
		"max_tokens":      1000,
		"temperature":     0.1,
		"response_format": map[string]string{"type": "json_object"}, // Request JSON format
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openAI API returned status %d: %s", resp.StatusCode, string(body))
	}

	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return "", fmt.Errorf("failed to decode OpenAI response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	metadataJSON := openaiResp.Choices[0].Message.Content
	slog.Info("Extracted metadata", "provider", "openai", "model", model, "length", len(metadataJSON))
	return metadataJSON, nil
}
