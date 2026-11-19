package meals

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordedCall struct {
	sql  string
	args []any
}

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanFn == nil {
		return pgx.ErrNoRows
	}
	return r.scanFn(dest...)
}

type fakeSavedMealRows struct {
	pgx.Rows
	items   []SavedMeal
	idx     int
	closed  bool
	scanErr error
	rowsErr error
}

func (r *fakeSavedMealRows) Close()                                       { r.closed = true }
func (r *fakeSavedMealRows) Err() error                                   { return r.rowsErr }
func (r *fakeSavedMealRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *fakeSavedMealRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeSavedMealRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeSavedMealRows) RawValues() [][]byte                          { return nil }
func (r *fakeSavedMealRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeSavedMealRows) Next() bool {
	if r.closed || r.idx >= len(r.items) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeSavedMealRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	item := r.items[r.idx-1]
	*(dest[0].(*string)) = item.ID
	*(dest[1].(*string)) = item.UserID
	*(dest[2].(*string)) = item.MealDate
	*(dest[3].(*string)) = item.MealTime
	*(dest[4].(*string)) = item.Name
	*(dest[5].(*int16)) = item.HealthScore
	*(dest[6].(*int)) = item.Calories
	*(dest[7].(*float64)) = item.ProteinG
	*(dest[8].(*float64)) = item.CarbsG
	*(dest[9].(*float64)) = item.FatG
	*(dest[10].(*float64)) = item.FiberG
	*(dest[11].(**int)) = item.PriceIDR
	*(dest[12].(*string)) = item.InputSource
	*(dest[13].(*time.Time)) = item.CreatedAt
	return nil
}

type fakeHealthScoreRows struct {
	pgx.Rows
	items   []DailyHealthScorePoint
	idx     int
	closed  bool
	scanErr error
	rowsErr error
}

func (r *fakeHealthScoreRows) Close()                                       { r.closed = true }
func (r *fakeHealthScoreRows) Err() error                                   { return r.rowsErr }
func (r *fakeHealthScoreRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *fakeHealthScoreRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeHealthScoreRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeHealthScoreRows) RawValues() [][]byte                          { return nil }
func (r *fakeHealthScoreRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeHealthScoreRows) Next() bool {
	if r.closed || r.idx >= len(r.items) {
		return false
	}
	r.idx++
	return true
}

func (r *fakeHealthScoreRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	item := r.items[r.idx-1]
	*(dest[0].(*string)) = item.Date
	*(dest[1].(*int)) = item.AverageHealthScore
	*(dest[2].(*int)) = item.MealCount
	return nil
}

type savedMealFakeDBTX struct {
	execCalls     []recordedCall
	queryCalls    []recordedCall
	queryRowCalls []recordedCall
	execFn        func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryFn       func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn    func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *savedMealFakeDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execCalls = append(f.execCalls, recordedCall{sql: sql, args: args})
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("DELETE 1"), nil
}

func (f *savedMealFakeDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queryCalls = append(f.queryCalls, recordedCall{sql: sql, args: args})
	if f.queryFn != nil {
		return f.queryFn(ctx, sql, args...)
	}
	return &fakeSavedMealRows{}, nil
}

func (f *savedMealFakeDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowCalls = append(f.queryRowCalls, recordedCall{sql: sql, args: args})
	if f.queryRowFn != nil {
		return f.queryRowFn(ctx, sql, args...)
	}
	return fakeRow{}
}

func TestNormalizeMealHistoryBounds(t *testing.T) {
	if got := normalizeMealHistoryLimit(-5); got != defaultMealHistoryLimit {
		t.Errorf("normalizeMealHistoryLimit(-5) = %d, want %d", got, defaultMealHistoryLimit)
	}
	if got := normalizeMealHistoryLimit(0); got != defaultMealHistoryLimit {
		t.Errorf("normalizeMealHistoryLimit(0) = %d, want %d", got, defaultMealHistoryLimit)
	}
	if got := normalizeMealHistoryLimit(250); got != 250 {
		t.Errorf("normalizeMealHistoryLimit(250) = %d, want 250", got)
	}
	if got := normalizeMealHistoryLimit(5000); got != maxMealHistoryLimit {
		t.Errorf("normalizeMealHistoryLimit(5000) = %d, want %d", got, maxMealHistoryLimit)
	}

	if got := normalizeMealHistoryOffset(-10); got != 0 {
		t.Errorf("normalizeMealHistoryOffset(-10) = %d, want 0", got)
	}
	if got := normalizeMealHistoryOffset(25); got != 25 {
		t.Errorf("normalizeMealHistoryOffset(25) = %d, want 25", got)
	}
}

func TestStoreSavedMealRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"
	mealID := "b7c0e9f0-1234-4abc-8def-0123456789ab"
	price := 12000

	sampleMeal := SavedMeal{
		ID:          mealID,
		UserID:      userID,
		MealDate:    "2026-11-19",
		MealTime:    MealTimeLunch,
		Name:        "Nasi Telur Dadar + Tumis Kangkung",
		HealthScore: 78,
		Calories:    520,
		ProteinG:    22.5,
		CarbsG:      64.0,
		FatG:        16.0,
		FiberG:      5.5,
		PriceIDR:    &price,
		InputSource: InputSourceIngredients,
		CreatedAt:   now,
	}

	t.Run("CreateSavedMeal uses explicit casts, TO_CHAR(meal_date), and returns persisted meal", func(t *testing.T) {
		db := &savedMealFakeDBTX{
			queryRowFn: func(_ context.Context, sql string, args ...any) pgx.Row {
				requiredSQL := []string{
					"INSERT INTO saved_meals",
					"$1::UUID",
					"$2::DATE",
					"$3::TEXT",
					"$4::TEXT",
					"$5::SMALLINT",
					"$6::INTEGER",
					"$7::NUMERIC(6, 2)",
					"$11::INTEGER",
					"$12::TEXT",
					"TO_CHAR(meal_date, 'YYYY-MM-DD')",
				}
				for _, snippet := range requiredSQL {
					if !strings.Contains(sql, snippet) {
						t.Errorf("CreateSavedMeal SQL missing %q: %s", snippet, sql)
					}
				}
				if len(args) != 12 {
					t.Errorf("expected 12 args, got %d", len(args))
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = sampleMeal.ID
					*(dest[1].(*string)) = sampleMeal.UserID
					*(dest[2].(*string)) = sampleMeal.MealDate
					*(dest[3].(*string)) = sampleMeal.MealTime
					*(dest[4].(*string)) = sampleMeal.Name
					*(dest[5].(*int16)) = sampleMeal.HealthScore
					*(dest[6].(*int)) = sampleMeal.Calories
					*(dest[7].(*float64)) = sampleMeal.ProteinG
					*(dest[8].(*float64)) = sampleMeal.CarbsG
					*(dest[9].(*float64)) = sampleMeal.FatG
					*(dest[10].(*float64)) = sampleMeal.FiberG
					*(dest[11].(**int)) = sampleMeal.PriceIDR
					*(dest[12].(*string)) = sampleMeal.InputSource
					*(dest[13].(*time.Time)) = sampleMeal.CreatedAt
					return nil
				}}
			},
		}

		store := NewStoreWithDBTX(db)
		got, err := store.CreateSavedMeal(ctx, CreateSavedMealParams{
			UserID:      userID,
			MealDate:    "2026-11-19",
			MealTime:    MealTimeLunch,
			Name:        sampleMeal.Name,
			HealthScore: 78,
			Calories:    520,
			ProteinG:    22.5,
			CarbsG:      64.0,
			FatG:        16.0,
			FiberG:      5.5,
			PriceIDR:    &price,
			InputSource: InputSourceIngredients,
		})
		if err != nil {
			t.Fatalf("CreateSavedMeal unexpected error: %v", err)
		}
		if got.ID != mealID || got.MealDate != "2026-11-19" || got.PriceIDR == nil || *got.PriceIDR != 12000 {
			t.Fatalf("CreateSavedMeal returned unexpected meal: %+v", got)
		}
	})

	t.Run("CreateSavedMeal propagates database error", func(t *testing.T) {
		db := &savedMealFakeDBTX{
			queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				return fakeRow{scanFn: func(_ ...any) error {
					return errors.New("db insert failed")
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		if _, err := store.CreateSavedMeal(ctx, CreateSavedMealParams{UserID: userID}); err == nil {
			t.Fatal("expected error from CreateSavedMeal")
		}
	})

	t.Run("DeleteSavedMeal enforces IDOR-safe WHERE id AND user_id and maps 0 rows to ErrNotFound", func(t *testing.T) {
		db := &savedMealFakeDBTX{
			execFn: func(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
				if !strings.Contains(sql, "WHERE id = $1::UUID AND user_id = $2::UUID") {
					t.Errorf("DeleteSavedMeal SQL missing IDOR-safe WHERE clause: %s", sql)
				}
				if len(args) != 2 || args[0] != mealID || args[1] != userID {
					t.Errorf("DeleteSavedMeal unexpected args: %#v", args)
				}
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		}
		store := NewStoreWithDBTX(db)
		if err := store.DeleteSavedMeal(ctx, userID, mealID); err != nil {
			t.Fatalf("DeleteSavedMeal unexpected error: %v", err)
		}

		// Wrong owner or nonexistent meal returns ErrNotFound.
		db.execFn = func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("DELETE 0"), nil
		}
		if err := store.DeleteSavedMeal(ctx, userID, mealID); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("DeleteSavedMeal expected ErrNotFound, got %v", err)
		}

		// Exec error propagated.
		db.execFn = func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("connection lost")
		}
		if err := store.DeleteSavedMeal(ctx, userID, mealID); err == nil || errors.Is(err, database.ErrNotFound) {
			t.Fatalf("expected non-ErrNotFound error, got %v", err)
		}
	})

	t.Run("ListSavedMealsByDate orders created_at ASC and returns non-nil slice", func(t *testing.T) {
		rows := &fakeSavedMealRows{items: []SavedMeal{sampleMeal}}
		db := &savedMealFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if !strings.Contains(sql, "TO_CHAR(meal_date, 'YYYY-MM-DD')") ||
					!strings.Contains(sql, "WHERE user_id = $1::UUID AND meal_date = $2::DATE") ||
					!strings.Contains(sql, "ORDER BY created_at ASC, id ASC") {
					t.Errorf("unexpected ListSavedMealsByDate SQL: %s", sql)
				}
				if len(args) != 2 || args[0] != userID || args[1] != "2026-11-19" {
					t.Errorf("unexpected args: %#v", args)
				}
				return rows, nil
			},
		}
		store := NewStoreWithDBTX(db)
		list, err := store.ListSavedMealsByDate(ctx, userID, "2026-11-19")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !rows.closed {
			t.Errorf("expected rows.Close() to be called")
		}
		if len(list) != 1 || list[0].ID != mealID {
			t.Fatalf("unexpected meals: %+v", list)
		}

		// Empty result returns non-nil slice.
		db.queryFn = func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &fakeSavedMealRows{}, nil
		}
		empty, err := store.ListSavedMealsByDate(ctx, userID, "2026-11-18")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if empty == nil || len(empty) != 0 {
			t.Fatalf("expected non-nil empty slice, got %#v", empty)
		}
	})

	t.Run("ListMealHistory clamps limit/offset and orders meal_date DESC, created_at ASC", func(t *testing.T) {
		rows := &fakeSavedMealRows{items: []SavedMeal{sampleMeal}}
		db := &savedMealFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if !strings.Contains(sql, "WHERE user_id = $1::UUID") ||
					!strings.Contains(sql, "AND meal_date >= $2::DATE") ||
					!strings.Contains(sql, "AND meal_date <= $3::DATE") ||
					!strings.Contains(sql, "ORDER BY meal_date DESC, created_at ASC, id ASC") ||
					!strings.Contains(sql, "LIMIT $4::INTEGER OFFSET $5::INTEGER") {
					t.Errorf("unexpected ListMealHistory SQL: %s", sql)
				}
				if len(args) != 5 || args[0] != userID || args[1] != "2026-10-21" || args[2] != "2026-11-19" || args[3] != maxMealHistoryLimit || args[4] != 0 {
					t.Errorf("unexpected ListMealHistory args: %#v", args)
				}
				return rows, nil
			},
		}
		store := NewStoreWithDBTX(db)
		list, err := store.ListMealHistory(ctx, userID, "2026-10-21", "2026-11-19", 9999, -20)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !rows.closed || len(list) != 1 {
			t.Fatalf("unexpected ListMealHistory result: len=%d closed=%v", len(list), rows.closed)
		}

		// Propagates rows.Err()
		db.queryFn = func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &fakeSavedMealRows{rowsErr: errors.New("stream error")}, nil
		}
		if _, err := store.ListMealHistory(ctx, userID, "2026-10-21", "2026-11-19", 0, 0); err == nil {
			t.Fatal("expected rows.Err() to be propagated")
		}
	})

	t.Run("GetWeeklyHealthScores aggregates ROUND(AVG(health_score)) and COUNT(*) ordered ASC", func(t *testing.T) {
		expectedPoints := []DailyHealthScorePoint{
			{Date: "2026-11-14", AverageHealthScore: 68, MealCount: 2},
			{Date: "2026-11-19", AverageHealthScore: 78, MealCount: 1},
		}
		rows := &fakeHealthScoreRows{items: expectedPoints}
		db := &savedMealFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				requiredSQL := []string{
					"TO_CHAR(meal_date, 'YYYY-MM-DD')",
					"ROUND(AVG(health_score))::INTEGER",
					"COUNT(*)::INTEGER",
					"WHERE user_id = $1::UUID",
					"AND meal_date >= $2::DATE",
					"AND meal_date <= $3::DATE",
					"GROUP BY meal_date",
					"ORDER BY meal_date ASC",
				}
				for _, snippet := range requiredSQL {
					if !strings.Contains(sql, snippet) {
						t.Errorf("GetWeeklyHealthScores SQL missing %q: %s", snippet, sql)
					}
				}
				if len(args) != 3 || args[0] != userID || args[1] != "2026-11-13" || args[2] != "2026-11-19" {
					t.Errorf("unexpected GetWeeklyHealthScores args: %#v", args)
				}
				return rows, nil
			},
		}
		store := NewStoreWithDBTX(db)
		points, err := store.GetWeeklyHealthScores(ctx, userID, "2026-11-13", "2026-11-19")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !rows.closed {
			t.Errorf("expected rows.Close() to be called")
		}
		if len(points) != 2 || points[0].AverageHealthScore != 68 || points[1].MealCount != 1 {
			t.Fatalf("unexpected points: %+v", points)
		}

		// Propagates scan error.
		db.queryFn = func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
			return &fakeHealthScoreRows{items: expectedPoints, scanErr: errors.New("scan fail")}, nil
		}
		if _, err := store.GetWeeklyHealthScores(ctx, userID, "2026-11-13", "2026-11-19"); err == nil {
			t.Fatal("expected scan error to be propagated")
		}
	})

	t.Run("Nil store returns error instead of panicking", func(t *testing.T) {
		var s *Store
		if _, err := s.CreateSavedMeal(ctx, CreateSavedMealParams{}); err == nil {
			t.Fatal("expected error from nil store CreateSavedMeal")
		}
		if err := s.DeleteSavedMeal(ctx, userID, mealID); err == nil {
			t.Fatal("expected error from nil store DeleteSavedMeal")
		}
		if _, err := s.ListSavedMealsByDate(ctx, userID, "2026-11-19"); err == nil {
			t.Fatal("expected error from nil store ListSavedMealsByDate")
		}
		if _, err := s.ListMealHistory(ctx, userID, "2026-11-01", "2026-11-19", 10, 0); err == nil {
			t.Fatal("expected error from nil store ListMealHistory")
		}
		if _, err := s.GetWeeklyHealthScores(ctx, userID, "2026-11-13", "2026-11-19"); err == nil {
			t.Fatal("expected error from nil store GetWeeklyHealthScores")
		}
	})
}
