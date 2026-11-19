// Package chat implements the AI nutrition chat, food photo analysis pipeline,
// anak kos food price repository, and HTTP handler for POST /api/v1/chat/analyze.
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

	temp := float32(0.3)
	genCfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(req.SystemInstruction, genai.RoleUser),
		Temperature:       &temp,
		ResponseMIMEType:  "application/json",
		ResponseSchema:    chatResponseSchema(),
	}

	contents := []*genai.Content{
		genai.NewContentFromParts(parts, genai.RoleUser),
	}

	resp, err := c.SDK.Models.GenerateContent(callCtx, c.Model, contents, genCfg)
	if err != nil {
		return nil, fmt.Errorf("gemini generate content: %w", err)
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

func chatResponseSchema() *genai.Schema {
	mealSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"name":                {Type: genai.TypeString},
			"portion":             {Type: genai.TypeString},
			"health_score":        {Type: genai.TypeInteger},
			"health_label":        {Type: genai.TypeString, Enum: []string{HealthLabelLow, HealthLabelFair, HealthLabelGood}},
			"calories":            {Type: genai.TypeInteger},
			"protein_g":           {Type: genai.TypeNumber},
			"carbs_g":             {Type: genai.TypeNumber},
			"fat_g":               {Type: genai.TypeNumber},
			"fiber_g":             {Type: genai.TypeNumber},
			"estimated_price_idr": {Type: genai.TypeInteger},
			"is_price_estimated":  {Type: genai.TypeBoolean},
			"note":                {Type: genai.TypeString},
			"cooking_steps":       {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"missing_ingredients": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"input_source":        {Type: genai.TypeString, Enum: []string{InputSourceIngredients, InputSourceBudget, InputSourcePhoto}},
		},
		Required: []string{
			"name", "portion", "health_score", "health_label",
			"calories", "protein_g", "carbs_g", "fat_g", "fiber_g",
			"estimated_price_idr", "is_price_estimated", "note",
			"cooking_steps", "missing_ingredients", "input_source",
		},
	}

	shoppingSchema := &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"name":                {Type: genai.TypeString},
			"unit":                {Type: genai.TypeString},
			"estimated_price_idr": {Type: genai.TypeInteger},
			"is_price_estimated":  {Type: genai.TypeBoolean},
			"calories":            {Type: genai.TypeInteger},
			"protein_g":           {Type: genai.TypeNumber},
			"carbs_g":             {Type: genai.TypeNumber},
			"fat_g":               {Type: genai.TypeNumber},
			"fiber_g":             {Type: genai.TypeNumber},
		},
		Required: []string{
			"name", "unit", "estimated_price_idr", "is_price_estimated",
			"calories", "protein_g", "carbs_g", "fat_g", "fiber_g",
		},
	}

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"intent": {
				Type: genai.TypeString,
				Enum: []string{
					IntentIngredients,
					IntentBudgetMeals,
					IntentBudgetIngredients,
					IntentPhotoFood,
					IntentPhotoIngredients,
					IntentOffTopic,
				},
			},
			"intro":                  {Type: genai.TypeString},
			"off_topic_notice":       {Type: genai.TypeString},
			"detected_items":         {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
			"needs_clarification":    {Type: genai.TypeBoolean},
			"budget_idr":             {Type: genai.TypeInteger},
			"budget_exceeded_notice": {Type: genai.TypeString},
			"meals":                  {Type: genai.TypeArray, Items: mealSchema},
			"shopping_list":          {Type: genai.TypeArray, Items: shoppingSchema},
		},
		Required: []string{"intent", "intro", "detected_items", "needs_clarification", "meals", "shopping_list"},
	}
}
