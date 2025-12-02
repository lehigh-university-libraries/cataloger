package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/lehigh-university-libraries/cataloger/internal/providers"
)

// Ollama is a provider for Ollama
type Ollama struct{}

// New returns a new Ollama provider
func New() *Ollama {
	return &Ollama{}
}

// ExtractText extracts text from the given prompt using Ollama
func (o *Ollama) ExtractText(ctx context.Context, config providers.Config) (string, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	url := ollamaURL + "/api/generate"

	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  config.Model,
		"prompt": config.Prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": config.Temperature,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("received non-200 status code: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response body: %w", err)
	}

	return response.Response, nil
}
