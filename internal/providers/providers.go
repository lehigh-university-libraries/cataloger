package providers

import (
	"context"
)

// Config represents the configuration for an LLM provider
type Config struct {
	Model       string
	Temperature float64
	Prompt      string
}

// Provider defines the interface for an LLM provider
type Provider interface {
	ExtractText(ctx context.Context, config Config) (string, error)
}
