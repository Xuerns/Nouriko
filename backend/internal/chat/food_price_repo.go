package chat

import (
	"context"
	"fmt"
	"strings"

	"Nouriko/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// FoodPriceStore defines read operations for the food_prices reference table.
type FoodPriceStore interface {
	ListFoodPricesByBudget(ctx context.Context, maxBudgetIDR int, category string, limit int) ([]FoodPrice, error)
	ListAllFoodPrices(ctx context.Context, category string, limit int) ([]FoodPrice, error)
}

// FoodPriceRepo implements FoodPriceStore backed by PostgreSQL (pgx/v5).
type FoodPriceRepo struct {
	q database.DBTX
}

// Compile-time check ensuring FoodPriceRepo implements FoodPriceStore.
var _ FoodPriceStore = (*FoodPriceRepo)(nil)

// NewFoodPriceRepo creates a PostgreSQL-backed FoodPriceRepo from a connection pool.
func NewFoodPriceRepo(pool *pgxpool.Pool) *FoodPriceRepo {
	return &FoodPriceRepo{q: pool}
}

// NewFoodPriceRepoWithDBTX creates a FoodPriceRepo bound to an existing DBTX.
func NewFoodPriceRepoWithDBTX(q database.DBTX) *FoodPriceRepo {
	return &FoodPriceRepo{q: q}
}

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
func (r *FoodPriceRepo) ListFoodPricesByBudget(ctx context.Context, maxBudgetIDR int, category string, limit int) ([]FoodPrice, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("food price store is not initialized")
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
		rows, err = r.q.Query(ctx, query, maxBudgetIDR, cat, effectiveLimit)
	} else {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			WHERE price_idr <= $1::INTEGER
			ORDER BY price_idr ASC, name ASC
			LIMIT $2::INTEGER
		`
		rows, err = r.q.Query(ctx, query, maxBudgetIDR, effectiveLimit)
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

// ListAllFoodPrices lists reference food prices, optionally filtered by category.
func (r *FoodPriceRepo) ListAllFoodPrices(ctx context.Context, category string, limit int) ([]FoodPrice, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("food price store is not initialized")
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
		rows, err = r.q.Query(ctx, query, cat, effectiveLimit)
	} else {
		const query = `
			SELECT ` + foodPriceColumns + `
			FROM food_prices
			ORDER BY category ASC, price_idr ASC, name ASC
			LIMIT $1::INTEGER
		`
		rows, err = r.q.Query(ctx, query, effectiveLimit)
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
