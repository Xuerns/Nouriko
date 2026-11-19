// Package ai wraps the Gemini API client used for nutrition recommendations
// and food photo analysis (PRD Section 8).
package ai

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"Nouriko/internal/config"

	"google.golang.org/genai"
)

// Client wraps the Google GenAI SDK client with Nouriko's model and timeout settings.
type Client struct {
	SDK     *genai.Client
	Model   string
	Timeout time.Duration
}

// NewGeminiClient initializes the Gemini API client.
// In local mode, an empty GEMINI_API_KEY returns (nil, nil) so the server can
// boot without external credentials.
func NewGeminiClient(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.Gemini.APIKey == "" {
		if cfg.IsProduction() {
			return nil, fmt.Errorf("GEMINI_API_KEY is required in production mode")
		}
		return nil, nil
	}

	sdk, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.Gemini.APIKey,
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Timeout: cfg.Gemini.Timeout,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init gemini client: %w", err)
	}

	return &Client{
		SDK:     sdk,
		Model:   cfg.Gemini.Model,
		Timeout: cfg.Gemini.Timeout,
	}, nil
}
