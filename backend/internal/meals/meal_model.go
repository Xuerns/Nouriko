// Package meals manages saved meals, daily nutrition summaries, and meal history
// with 7-day health score trends (PRD F5, F6, F9).
package meals

import "time"

// Supported saved_meals meal_time values.
const (
	MealTimeBreakfast = "breakfast"
	MealTimeLunch     = "lunch"
	MealTimeDinner    = "dinner"
	MealTimeSnack     = "snack"
)

// Supported saved_meals input_source values.
const (
	InputSourceIngredients = "ingredients"
	InputSourceBudget      = "budget"
	InputSourcePhoto       = "photo"
)

// SavedMeal represents a row in the saved_meals table.
type SavedMeal struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	MealDate    string    `json:"meal_date"` // YYYY-MM-DD calendar date
	MealTime    string    `json:"meal_time"` // breakfast | lunch | dinner | snack
	Name        string    `json:"name"`
	HealthScore int16     `json:"health_score"` // 0..100
	Calories    int       `json:"calories"`
	ProteinG    float64   `json:"protein_g"`
	CarbsG      float64   `json:"carbs_g"`
	FatG        float64   `json:"fat_g"`
	FiberG      float64   `json:"fiber_g"`
	PriceIDR    *int      `json:"price_idr"`
	InputSource string    `json:"input_source"` // ingredients | budget | photo
	CreatedAt   time.Time `json:"created_at"`
}

// CreateSavedMealParams defines validated inputs for inserting a saved_meals row.
type CreateSavedMealParams struct {
	UserID      string
	MealDate    string // YYYY-MM-DD
	MealTime    string
	Name        string
	HealthScore int16
	Calories    int
	ProteinG    float64
	CarbsG      float64
	FatG        float64
	FiberG      float64
	PriceIDR    *int
	InputSource string
}

// DailyHealthScorePoint represents an aggregated health score for one calendar date.
type DailyHealthScorePoint struct {
	Date               string `json:"date"`
	AverageHealthScore int    `json:"average_health_score"`
	MealCount          int    `json:"meal_count"`
}
