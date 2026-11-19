package meals

import (
	"context"
	"fmt"

	"Nouriko/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SavedMealStore defines persistence operations for the saved_meals table.
type SavedMealStore interface {
	CreateSavedMeal(ctx context.Context, params CreateSavedMealParams) (SavedMeal, error)
	DeleteSavedMeal(ctx context.Context, userID, mealID string) error
	ListSavedMealsByDate(ctx context.Context, userID, date string) ([]SavedMeal, error)
	ListMealHistory(ctx context.Context, userID, fromDate, toDate string, limit, offset int) ([]SavedMeal, error)
	GetWeeklyHealthScores(ctx context.Context, userID, fromDate, toDate string) ([]DailyHealthScorePoint, error)
}

// Store implements SavedMealStore backed by PostgreSQL (pgx/v5).
type Store struct {
	q database.DBTX
}

// Compile-time check ensuring Store implements SavedMealStore.
var _ SavedMealStore = (*Store)(nil)

// NewStore creates a PostgreSQL-backed saved meals Store from a connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: pool}
}

// NewStoreWithDBTX creates a saved meals Store bound to an existing DBTX.
func NewStoreWithDBTX(q database.DBTX) *Store {
	return &Store{q: q}
}

const (
	savedMealColumns = `id::TEXT, user_id::TEXT, TO_CHAR(meal_date, 'YYYY-MM-DD'), meal_time, name, health_score, calories, protein_g, carbs_g, fat_g, fiber_g, price_idr, input_source, created_at`

	defaultMealHistoryLimit = 500
	maxMealHistoryLimit     = 1000
)

func normalizeMealHistoryLimit(limit int) int {
	if limit <= 0 {
		return defaultMealHistoryLimit
	}
	if limit > maxMealHistoryLimit {
		return maxMealHistoryLimit
	}
	return limit
}

func normalizeMealHistoryOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func scanSavedMeal(row pgx.Row) (SavedMeal, error) {
	var m SavedMeal
	err := row.Scan(
		&m.ID,
		&m.UserID,
		&m.MealDate,
		&m.MealTime,
		&m.Name,
		&m.HealthScore,
		&m.Calories,
		&m.ProteinG,
		&m.CarbsG,
		&m.FatG,
		&m.FiberG,
		&m.PriceIDR,
		&m.InputSource,
		&m.CreatedAt,
	)
	if err != nil {
		return SavedMeal{}, err
	}
	return m, nil
}

func scanSavedMeals(rows pgx.Rows) ([]SavedMeal, error) {
	defer rows.Close()

	list := make([]SavedMeal, 0)
	for rows.Next() {
		var m SavedMeal
		if err := rows.Scan(
			&m.ID,
			&m.UserID,
			&m.MealDate,
			&m.MealTime,
			&m.Name,
			&m.HealthScore,
			&m.Calories,
			&m.ProteinG,
			&m.CarbsG,
			&m.FatG,
			&m.FiberG,
			&m.PriceIDR,
			&m.InputSource,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return list, nil
}

func scanDailyHealthScorePoints(rows pgx.Rows) ([]DailyHealthScorePoint, error) {
	defer rows.Close()

	points := make([]DailyHealthScorePoint, 0)
	for rows.Next() {
		var p DailyHealthScorePoint
		if err := rows.Scan(
			&p.Date,
			&p.AverageHealthScore,
			&p.MealCount,
		); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return points, nil
}

// CreateSavedMeal inserts a new saved_meals row for the user and returns the persisted record.
func (s *Store) CreateSavedMeal(ctx context.Context, params CreateSavedMealParams) (SavedMeal, error) {
	if s == nil || s.q == nil {
		return SavedMeal{}, fmt.Errorf("meals store is not initialized")
	}

	const query = `
		INSERT INTO saved_meals (
			user_id, meal_date, meal_time, name, health_score,
			calories, protein_g, carbs_g, fat_g, fiber_g, price_idr, input_source
		)
		VALUES (
			$1::UUID, $2::DATE, $3::TEXT, $4::TEXT, $5::SMALLINT,
			$6::INTEGER, $7::NUMERIC(6, 2), $8::NUMERIC(6, 2), $9::NUMERIC(6, 2), $10::NUMERIC(6, 2), $11::INTEGER, $12::TEXT
		)
		RETURNING ` + savedMealColumns

	meal, err := scanSavedMeal(s.q.QueryRow(
		ctx,
		query,
		params.UserID,
		params.MealDate,
		params.MealTime,
		params.Name,
		params.HealthScore,
		params.Calories,
		params.ProteinG,
		params.CarbsG,
		params.FatG,
		params.FiberG,
		params.PriceIDR,
		params.InputSource,
	))
	if err != nil {
		return SavedMeal{}, fmt.Errorf("create saved meal: %w", err)
	}
	return meal, nil
}

// DeleteSavedMeal deletes a saved_meals row scoped strictly to both mealID and userID
// to prevent IDOR. Returns database.ErrNotFound if no matching row was deleted.
func (s *Store) DeleteSavedMeal(ctx context.Context, userID, mealID string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("meals store is not initialized")
	}

	const query = `DELETE FROM saved_meals WHERE id = $1::UUID AND user_id = $2::UUID`
	tag, err := s.q.Exec(ctx, query, mealID, userID)
	if err != nil {
		return fmt.Errorf("delete saved meal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// ListSavedMealsByDate lists all saved meals for a user on a specific YYYY-MM-DD calendar date,
// ordered by created_at ASC, id ASC (capped defensively at 500 rows).
func (s *Store) ListSavedMealsByDate(ctx context.Context, userID, date string) ([]SavedMeal, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("meals store is not initialized")
	}

	const query = `
		SELECT ` + savedMealColumns + `
		FROM saved_meals
		WHERE user_id = $1::UUID AND meal_date = $2::DATE
		ORDER BY created_at ASC, id ASC
		LIMIT 500
	`
	rows, err := s.q.Query(ctx, query, userID, date)
	if err != nil {
		return nil, fmt.Errorf("list saved meals by date: %w", err)
	}

	list, err := scanSavedMeals(rows)
	if err != nil {
		return nil, fmt.Errorf("scan saved meals by date: %w", err)
	}
	return list, nil
}

// ListMealHistory lists saved meals for a user within the inclusive [fromDate, toDate] range,
// ordered by meal_date DESC, created_at ASC, id ASC with bounded limit and offset.
func (s *Store) ListMealHistory(ctx context.Context, userID, fromDate, toDate string, limit, offset int) ([]SavedMeal, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("meals store is not initialized")
	}

	effectiveLimit := normalizeMealHistoryLimit(limit)
	effectiveOffset := normalizeMealHistoryOffset(offset)

	const query = `
		SELECT ` + savedMealColumns + `
		FROM saved_meals
		WHERE user_id = $1::UUID
		  AND meal_date >= $2::DATE
		  AND meal_date <= $3::DATE
		ORDER BY meal_date DESC, created_at ASC, id ASC
		LIMIT $4::INTEGER OFFSET $5::INTEGER
	`
	rows, err := s.q.Query(ctx, query, userID, fromDate, toDate, effectiveLimit, effectiveOffset)
	if err != nil {
		return nil, fmt.Errorf("list meal history: %w", err)
	}

	list, err := scanSavedMeals(rows)
	if err != nil {
		return nil, fmt.Errorf("scan meal history: %w", err)
	}
	return list, nil
}

// GetWeeklyHealthScores aggregates daily average health scores and meal counts for a user
// within the inclusive [fromDate, toDate] window, ordered by meal_date ASC.
func (s *Store) GetWeeklyHealthScores(ctx context.Context, userID, fromDate, toDate string) ([]DailyHealthScorePoint, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("meals store is not initialized")
	}

	const query = `
		SELECT TO_CHAR(meal_date, 'YYYY-MM-DD') AS meal_date,
		       ROUND(AVG(health_score))::INTEGER AS avg_health_score,
		       COUNT(*)::INTEGER AS meal_count
		FROM saved_meals
		WHERE user_id = $1::UUID
		  AND meal_date >= $2::DATE
		  AND meal_date <= $3::DATE
		GROUP BY meal_date
		ORDER BY meal_date ASC
	`
	rows, err := s.q.Query(ctx, query, userID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("get weekly health scores: %w", err)
	}

	points, err := scanDailyHealthScorePoints(rows)
	if err != nil {
		return nil, fmt.Errorf("scan weekly health scores: %w", err)
	}
	return points, nil
}
