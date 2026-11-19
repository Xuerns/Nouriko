package chat

import "time"

// Supported food_prices category values.
const (
	FoodPriceCategoryMeal       = "meal"
	FoodPriceCategoryIngredient = "ingredient"
)

// FoodPrice represents a row in the food_prices reference table.
type FoodPrice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PriceIDR  int       `json:"price_idr"`
	Unit      string    `json:"unit"`
	Category  string    `json:"category"` // "meal" | "ingredient"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
