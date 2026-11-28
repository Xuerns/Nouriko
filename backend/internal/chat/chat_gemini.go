// Package chat implements the AI nutrition chat, food photo analysis pipeline,
// anak kos food price repository, and HTTP handler for POST /api/v1/chat/analyze.
package chat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"Nouriko/internal/config"

	"google.golang.org/genai"
)

// Client wraps the Google GenAI SDK client with Nouriko's model and timeout settings.
type Client struct {
	SDK           *genai.Client
	Model         string
	ThinkingLevel genai.ThinkingLevel
	Timeout       time.Duration
}

// Compile-time interface check.
var _ Analyzer = (*Client)(nil)

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

	attempts := int32(2)
	initialDelay := 1.0
	maxDelay := 3.0
	expBase := 2.0

	// Force IPv4 ("tcp4") dialing so broken or unrouted IPv6 paths inside Docker/ISP
	// environments never stall connection establishment for 20-35 seconds before fallback.
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, "tcp4", addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	sdk, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.Gemini.APIKey,
		Backend: genai.BackendGeminiAPI,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   cfg.Gemini.Timeout,
		},
		HTTPOptions: genai.HTTPOptions{
			RetryOptions: &genai.HTTPRetryOptions{
				Attempts:     &attempts,
				InitialDelay: &initialDelay,
				MaxDelay:     &maxDelay,
				ExpBase:      &expBase,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("init gemini client: %w", err)
	}

	return &Client{
		SDK:           sdk,
		Model:         "gemini-3.5-flash-lite",
		ThinkingLevel: resolveThinkingLevel(cfg.Gemini.ThinkingLevel),
		Timeout:       cfg.Gemini.Timeout,
	}, nil
}

// resolveThinkingLevel maps the configured thinking level for gemini-3.5-flash-lite
// (minimal, low, medium, high), defaulting to minimal.
func resolveThinkingLevel(explicitLevel string) genai.ThinkingLevel {
	switch strings.ToLower(strings.TrimSpace(explicitLevel)) {
	case "low":
		return genai.ThinkingLevelLow
	case "medium":
		return genai.ThinkingLevelMedium
	case "high":
		return genai.ThinkingLevelHigh
	default:
		return genai.ThinkingLevelMinimal
	}
}

// Analyze sends the structured prompt and optional in-memory photo bytes to Gemini
// using strict JSON output mode and a response schema.
func (c *Client) Analyze(ctx context.Context, req AnalyzeRequest) (*RawAIResponse, error) {
	if c == nil || c.SDK == nil {
		return nil, ErrUnavailable
	}

	callCtx := ctx
	var cancel context.CancelFunc
	if c.Timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}

	parts := make([]*genai.Part, 0, 2)
	parts = append(parts, genai.NewPartFromText(req.UserPrompt))
	if len(req.PhotoBytes) > 0 && req.PhotoMIMEType != "" {
		parts = append(parts, genai.NewPartFromBytes(req.PhotoBytes, req.PhotoMIMEType))
	}

	genCfg := &genai.GenerateContentConfig{
		SystemInstruction:  genai.NewContentFromText(req.SystemInstruction, genai.RoleUser),
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: chatResponseJSONSchema(),
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingLevel: c.ThinkingLevel,
		},
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	resp, err := c.SDK.Models.GenerateContent(callCtx, c.Model, contents, genCfg)
	if err != nil {
		return nil, mapGeminiError(err)
	}
	if resp == nil {
		return nil, fmt.Errorf("gemini returned empty response")
	}

	rawText := stripJSONCodeFence(resp.Text())
	if rawText == "" {
		return nil, fmt.Errorf("gemini returned blank json payload")
	}

	var parsed RawAIResponse
	if err := json.Unmarshal([]byte(rawText), &parsed); err != nil {
		return nil, fmt.Errorf("decode gemini json: %w", err)
	}
	return &parsed, nil
}

// mapGeminiError translates Google GenAI SDK errors (such as 504 DEADLINE_EXCEEDED
// or 429/503 transient errors) into Nouriko's domain sentinel errors so the HTTP
// handler can return accurate status codes (504 AI_TIMEOUT / 503 AI_UNAVAILABLE).
func mapGeminiError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("gemini generate content: %w", err)
	}

	var apiErr genai.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Code == http.StatusGatewayTimeout || strings.EqualFold(apiErr.Status, "DEADLINE_EXCEEDED"):
			return fmt.Errorf("gemini generate content (%s): %w", apiErr.Error(), context.DeadlineExceeded)
		case apiErr.Code == http.StatusTooManyRequests ||
			apiErr.Code == http.StatusServiceUnavailable ||
			strings.EqualFold(apiErr.Status, "RESOURCE_EXHAUSTED") ||
			strings.EqualFold(apiErr.Status, "UNAVAILABLE"):
			return fmt.Errorf("gemini generate content (%s): %w", apiErr.Error(), ErrUnavailable)
		}
	}

	var ptrErr *genai.APIError
	if errors.As(err, &ptrErr) && ptrErr != nil {
		switch {
		case ptrErr.Code == http.StatusGatewayTimeout || strings.EqualFold(ptrErr.Status, "DEADLINE_EXCEEDED"):
			return fmt.Errorf("gemini generate content (%s): %w", ptrErr.Error(), context.DeadlineExceeded)
		case ptrErr.Code == http.StatusTooManyRequests ||
			ptrErr.Code == http.StatusServiceUnavailable ||
			strings.EqualFold(ptrErr.Status, "RESOURCE_EXHAUSTED") ||
			strings.EqualFold(ptrErr.Status, "UNAVAILABLE"):
			return fmt.Errorf("gemini generate content (%s): %w", ptrErr.Error(), ErrUnavailable)
		}
	}

	return fmt.Errorf("gemini generate content: %w", err)
}

func stripJSONCodeFence(s string) string {
	trimmed := strings.TrimSpace(s)
	if strings.HasPrefix(trimmed, "```") {
		if idx := strings.Index(trimmed, "\n"); idx >= 0 {
			trimmed = trimmed[idx+1:]
		}
		trimmed = strings.TrimSuffix(strings.TrimSpace(trimmed), "```")
	}
	return strings.TrimSpace(trimmed)
}

func chatResponseJSONSchema() map[string]any {
	mealSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":                map[string]any{"type": "string"},
			"portion":             map[string]any{"type": "string"},
			"health_score":        map[string]any{"type": "integer"},
			"health_label":        map[string]any{"type": "string", "enum": []string{HealthLabelLow, HealthLabelFair, HealthLabelGood}},
			"calories":            map[string]any{"type": "integer"},
			"protein_g":           map[string]any{"type": "number"},
			"carbs_g":             map[string]any{"type": "number"},
			"fat_g":               map[string]any{"type": "number"},
			"fiber_g":             map[string]any{"type": "number"},
			"estimated_price_idr": map[string]any{"type": "integer"},
			"is_price_estimated":  map[string]any{"type": "boolean"},
			"note":                map[string]any{"type": "string"},
			"cooking_steps":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"missing_ingredients": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"input_source":        map[string]any{"type": "string", "enum": []string{InputSourceIngredients, InputSourceBudget, InputSourcePhoto}},
		},
		"required": []string{
			"name", "portion", "health_score", "health_label",
			"calories", "protein_g", "carbs_g", "fat_g", "fiber_g",
			"estimated_price_idr", "is_price_estimated", "note",
			"cooking_steps", "missing_ingredients", "input_source",
		},
	}

	shoppingSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":                map[string]any{"type": "string"},
			"unit":                map[string]any{"type": "string"},
			"estimated_price_idr": map[string]any{"type": "integer"},
			"is_price_estimated":  map[string]any{"type": "boolean"},
			"calories":            map[string]any{"type": "integer"},
			"protein_g":           map[string]any{"type": "number"},
			"carbs_g":             map[string]any{"type": "number"},
			"fat_g":               map[string]any{"type": "number"},
			"fiber_g":             map[string]any{"type": "number"},
		},
		"required": []string{
			"name", "unit", "estimated_price_idr", "is_price_estimated",
			"calories", "protein_g", "carbs_g", "fat_g", "fiber_g",
		},
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"intent": map[string]any{
				"type": "string",
				"enum": []string{
					IntentIngredients,
					IntentBudgetMeals,
					IntentBudgetIngredients,
					IntentPhotoFood,
					IntentPhotoIngredients,
					IntentOffTopic,
				},
			},
			"intro":                  map[string]any{"type": "string"},
			"off_topic_notice":       map[string]any{"type": "string"},
			"detected_items":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"needs_clarification":    map[string]any{"type": "boolean"},
			"budget_idr":             map[string]any{"type": "integer"},
			"budget_exceeded_notice": map[string]any{"type": "string"},
			"meals":                  map[string]any{"type": "array", "items": mealSchema},
			"shopping_list":          map[string]any{"type": "array", "items": shoppingSchema},
		},
		"required": []string{"intent", "intro", "detected_items", "needs_clarification", "meals", "shopping_list"},
	}
}
