package gemini

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/lehigh-university-libraries/cataloger/internal/providers"
	"google.golang.org/api/option"
)

// Gemini is a provider for Google Gemini
type Gemini struct{}

// New returns a new Gemini provider
func New() *Gemini {
	return &Gemini{}
}

// ExtractText extracts text from the given prompt using Gemini
func (g *Gemini) ExtractText(ctx context.Context, config providers.Config) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", fmt.Errorf("failed to create new gemini client: %w", err)
	}
	defer client.Close()

	model := client.GenerativeModel(config.Model)
	model.SetTemperature(float32(config.Temperature))

	resp, err := model.GenerateContent(ctx, genai.Text(config.Prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned from Gemini")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("empty content returned from Gemini")
	}

	if txt, ok := candidate.Content.Parts[0].(genai.Text); ok {
		return string(txt), nil
	}

	return "", fmt.Errorf("unexpected response format from Gemini")
}
