package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	foodPriceColumns      = `id, name, price_idr, unit, category, created_at, updated_at`
	defaultFoodPriceLimit = 40
	maxFoodPriceLimit     = 100
)

func normalizeFoodPriceLimit(limit int) int {
	if limit <= 0 {
		return defaultFoodPriceLimit
	}
	if limit > maxFoodPriceLimit {
		return maxFoodPriceLimit
	}
	return limit
}

// escapeLikePattern escapes PostgreSQL LIKE/ILIKE wildcard characters (%, _)
// and the escape character (\) so user-provided search text is matched literally.
func escapeLikePattern(raw string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(raw)
}

func scanFoodPrices(rows pgx.Rows) ([]FoodPrice, error) {
	defer rows.Close()

	items := make([]FoodPrice, 0)
	for rows.Next() {
		var item FoodPrice
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.PriceIDR,
			&item.Unit,
			&item.Category,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListFoodPricesByBudget returns food price items with price_idr <= maxBudgetIDR,
// optionally filtered by category ("meal" or "ingredient"), ordered by price_idr ASC, name ASC.
func (s *Store) ListFoodPricesByBudget(ctx context.Context, maxBudgetIDR int, category string, limit int) ([]FoodPrice, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}
	if maxBudgetIDR < 0 {
		return []FoodPrice{}, nil
	}

	effectiveLimit := normalizeFoodPriceLimit(limit)
	cat := strings.TrimSpace(category)

	var (
		rows pgx.Rows
		err  error
	)
	if cat != "" {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE price_idr <= $1::INTEGER
			  AND category = $2::TEXT
			ORDER BY price_idr ASC, name ASC
			LIMIT $3::INTEGER
		`
		rows, err = s.q.Query(ctx, query, maxBudgetIDR, cat, effectiveLimit)
	} else {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE price_idr <= $1::INTEGER
			ORDER BY price_idr ASC, name ASC
			LIMIT $2::INTEGER
		`
		rows, err = s.q.Query(ctx, query, maxBudgetIDR, effectiveLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("list food prices by budget: %w", err)
	}

	items, err := scanFoodPrices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan food prices by budget: %w", err)
	}
	return items, nil
}

// SearchFoodPrices performs a case-insensitive substring search on food_prices.name
// with LIKE wildcard escaping, optionally filtered by category.
func (s *Store) SearchFoodPrices(ctx context.Context, query string, category string, limit int) ([]FoodPrice, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []FoodPrice{}, nil
	}

	pattern := "%" + escapeLikePattern(trimmed) + "%"
	effectiveLimit := normalizeFoodPriceLimit(limit)
	cat := strings.TrimSpace(category)

	var (
		rows pgx.Rows
		err  error
	)
	if cat != "" {
		const sqlQuery = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE name ILIKE $1::TEXT ESCAPE '\'
			  AND category = $2::TEXT
			ORDER BY price_idr ASC, name ASC
			LIMIT $3::INTEGER
		`
		rows, err = s.q.Query(ctx, sqlQuery, pattern, cat, effectiveLimit)
	} else {
		const sqlQuery = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE name ILIKE $1::TEXT ESCAPE '\'
			ORDER BY price_idr ASC, name ASC
			LIMIT $2::INTEGER
		`
		rows, err = s.q.Query(ctx, sqlQuery, pattern, effectiveLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("search food prices: %w", err)
	}

	items, err := scanFoodPrices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan search food prices: %w", err)
	}
	return items, nil
}

// ListAllFoodPrices lists reference food prices, optionally filtered by category.
func (s *Store) ListAllFoodPrices(ctx context.Context, category string, limit int) ([]FoodPrice, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	effectiveLimit := normalizeFoodPriceLimit(limit)
	cat := strings.TrimSpace(category)

	var (
		rows pgx.Rows
		err  error
	)
	if cat != "" {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE category = $1::TEXT
			ORDER BY category ASC, price_idr ASC, name ASC
			LIMIT $2::INTEGER
		`
		rows, err = s.q.Query(ctx, query, cat, effectiveLimit)
	} else {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			ORDER BY category ASC, price_idr ASC, name ASC
			LIMIT $1::INTEGER
		`
		rows, err = s.q.Query(ctx, query, effectiveLimit)
	}
	if err != nil {
		return nil, fmt.Errorf("list all food prices: %w", err)
	}

	items, err := scanFoodPrices(rows)
	if err != nil {
		return nil, fmt.Errorf("scan all food prices: %w", err)
	}
	return items, nil
}
