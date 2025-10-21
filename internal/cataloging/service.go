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

// GenerateMARCFromOCR generates a MARC record from OCR text extracted from a title page
func (s *Service) GenerateMARCFromOCR(ocrText, provider, model string) (string, error) {
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
		return s.generateFromOCRWithOpenAI(ocrText, model)
	case "ollama":
		return s.generateFromOCRWithOllama(ocrText, model)
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

	slog.Info("Generated MARC record", "length", len(ollamaResp.Response))
	return ollamaResp.Response, nil
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
Provide the complete MARC record in a clear, readable format. Start with:

MARC RECORD:
=LDR  [leader string]
=008  [fixed field data]
=020  [ISBN data]
...

Include all relevant MARC fields in numerical order. After the MARC record, provide a brief "CATALOGER'S NOTES" section with any observations or uncertainties about the bibliographic information.

Be thorough, precise, and follow Library of Congress Rule Interpretations (LCRIs) and MARC 21 standards exactly.`,
		map[string]string{"ocr": "OCR text", "image": "image provided"}[sourceType],
		sourceInstructions,
		map[string]string{"ocr": "ISBN (if present)", "image": "ISBN (if visible)"}[sourceType],
		missingFieldNote,
		generationNote,
	)
}

func (s *Service) buildMARCPromptForOCR() string {
	return s.buildMARCPrompt("ocr")
}

func (s *Service) buildMARCPromptForImage() string {
	return s.buildMARCPrompt("image")
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

	marcRecord := openaiResp.Choices[0].Message.Content
	slog.Info("Generated MARC record", "provider", "openai", "model", model, "length", len(marcRecord))
	return marcRecord, nil
}

// generateFromOCRWithOllama generates MARC from OCR text using Ollama
func (s *Service) generateFromOCRWithOllama(ocrText, model string) (string, error) {
	ollamaHost := os.Getenv("OLLAMA_URL")
	if ollamaHost == "" {
		ollamaHost = os.Getenv("OLLAMA_HOST")
	}
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	// Prepare Ollama request with OCR text
	systemPrompt := s.buildMARCPromptForOCR()
	userPrompt := fmt.Sprintf("Here is the OCR text extracted from a book title page:\n\n%s\n\nPlease generate a MARC 21 record based on this text.", ocrText)

	requestBody := map[string]interface{}{
		"model":  model,
		"prompt": systemPrompt + "\n\n" + userPrompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.1,
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
		return "", fmt.Errorf("failed to call Ollama API for OCR-based MARC: %w", err)
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
		return "", fmt.Errorf("failed to decode Ollama OCR-based response: %w", err)
	}

	slog.Info("Generated MARC record from OCR", "provider", "ollama", "length", len(ollamaResp.Response))
	return ollamaResp.Response, nil
}

// generateFromOCRWithOpenAI generates MARC from OCR text using OpenAI
func (s *Service) generateFromOCRWithOpenAI(ocrText, model string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Prepare OpenAI request with OCR text
	systemPrompt := s.buildMARCPromptForOCR()
	userPrompt := fmt.Sprintf("Here is the OCR text extracted from a book title page:\n\n%s\n\nPlease generate a MARC 21 record based on this text.", ocrText)

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
		"max_tokens":  4000,
		"temperature": 0.1,
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
		return "", fmt.Errorf("failed to call OpenAI API for OCR-based MARC: %w", err)
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
		return "", fmt.Errorf("failed to decode OpenAI OCR-based response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	marcRecord := openaiResp.Choices[0].Message.Content
	slog.Info("Generated MARC record from OCR", "provider", "openai", "model", model, "length", len(marcRecord))
	return marcRecord, nil
}
