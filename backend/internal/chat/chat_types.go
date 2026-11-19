package chat

import (
	"context"
	"errors"

	"Nouriko/internal/profile"
)

// Sentinel errors returned by the AI analysis pipeline.
var (
	ErrUnavailable       = errors.New("ai analyzer unavailable")
	ErrPhotoUnrecognized = errors.New("photo unrecognized")
)

// Supported intent values returned in ChatAnalyzeResponse.
const (
	IntentIngredients       = "ingredients"
	IntentBudgetMeals       = "budget_meals"
	IntentBudgetIngredients = "budget_ingredients"
	IntentPhotoFood         = "photo_food"
	IntentPhotoIngredients  = "photo_ingredients"
	IntentOffTopic          = "off_topic"
)

// Supported deterministic health_label values.
const (
	HealthLabelLow  = "low"
	HealthLabelFair = "fair"
	HealthLabelGood = "good"
)

// Supported input_source values per meal card.
const (
	InputSourceIngredients = "ingredients"
	InputSourceBudget      = "budget"
	InputSourcePhoto       = "photo"
)

// Supported macro_source values per meal card.
const (
	MacroSourceFoodAPI   = "food_api"
	MacroSourceEstimated = "estimated"
)

// ConversationTurn represents a compact ephemeral turn from the active chat session.
type ConversationTurn struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Summary string `json:"summary"` // max 300 runes
}

// AnalyzeInput is the high-level input passed from the HTTP handler to ChatService.
type AnalyzeInput struct {
	Message             string
	PhotoBytes          []byte // nil if no photo; strictly in-memory, never persisted
	PhotoMIMEType       string // "image/jpeg" | "image/png" | "image/webp"
	Language            string // "id" | "en"
	ConversationContext []ConversationTurn
	Profile             *profile.Profile
}

// AnalyzeRequest is the low-level prompt + optional image payload passed to Analyzer.
type AnalyzeRequest struct {
	SystemInstruction string
	UserPrompt        string
	PhotoBytes        []byte
	PhotoMIMEType     string
}

// Analyzer is the injectable LLM backend interface implemented by *Client.
type Analyzer interface {
	Analyze(ctx context.Context, req AnalyzeRequest) (*RawAIResponse, error)
}

// MealOption represents a single structured meal recommendation or photo analysis card (PRD F4).
type MealOption struct {
	Name               string   `json:"name"`
	Portion            string   `json:"portion"`
	HealthScore        int      `json:"health_score"`
	HealthLabel        string   `json:"health_label"`
	Calories           int      `json:"calories"`
	ProteinG           float64  `json:"protein_g"`
	CarbsG             float64  `json:"carbs_g"`
	FatG               float64  `json:"fat_g"`
	FiberG             float64  `json:"fiber_g"`
	EstimatedPriceIDR  int      `json:"estimated_price_idr"`
	IsPriceEstimated   bool     `json:"is_price_estimated"`
	IsMacroEstimated   bool     `json:"is_macro_estimated"`
	MacroSource        string   `json:"macro_source"`
	Note               string   `json:"note"`
	CookingSteps       []string `json:"cooking_steps"`
	MissingIngredients []string `json:"missing_ingredients"`
	InputSource        string   `json:"input_source"`
}

// ShoppingListItem represents an affordable ingredient item in a budget_ingredients response.
type ShoppingListItem struct {
	Name              string  `json:"name"`
	Unit              string  `json:"unit"`
	EstimatedPriceIDR int     `json:"estimated_price_idr"`
	IsPriceEstimated  bool    `json:"is_price_estimated"`
	Calories          int     `json:"calories"`
	ProteinG          float64 `json:"protein_g"`
	CarbsG            float64 `json:"carbs_g"`
	FatG              float64 `json:"fat_g"`
	FiberG            float64 `json:"fiber_g"`
}

// RawAIResponse is the structured JSON decoded from the LLM before backend post-processing.
type RawAIResponse struct {
	Intent               string             `json:"intent"`
	Intro                string             `json:"intro"`
	OffTopicNotice       string             `json:"off_topic_notice,omitempty"`
	DetectedItems        []string           `json:"detected_items"`
	NeedsClarification   bool               `json:"needs_clarification"`
	BudgetIDR            int                `json:"budget_idr,omitempty"`
	BudgetExceededNotice string             `json:"budget_exceeded_notice,omitempty"`
	Meals                []MealOption       `json:"meals"`
	ShoppingList         []ShoppingListItem `json:"shopping_list"`
}

// ChatAnalyzeResponse is the validated, post-processed API response for POST /api/v1/chat/analyze.
type ChatAnalyzeResponse = RawAIResponse
