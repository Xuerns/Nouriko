package chat

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type recordedCall struct {
	sql  string
	args []any
}

type fakeRow struct{}

func (fakeRow) Scan(_ ...any) error { return pgx.ErrNoRows }

// fakeRows implements pgx.Rows for testing multi-row query methods on FoodPriceRepo.
type fakeRows struct {
	pgx.Rows
	items   []FoodPrice
	idx     int
	closed  bool
	scanErr error
	rowsErr error
}

func (r *fakeRows) Close()                                       { r.closed = true }
func (r *fakeRows) Err() error                                   { return r.rowsErr }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.closed || r.idx >= len(r.items) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	item := r.items[r.idx-1]
	*(dest[0].(*string)) = item.ID
	*(dest[1].(*string)) = item.Name
	*(dest[2].(*int)) = item.PriceIDR
	*(dest[3].(*string)) = item.Unit
	*(dest[4].(*string)) = item.Category
	*(dest[5].(*time.Time)) = item.CreatedAt
	*(dest[6].(*time.Time)) = item.UpdatedAt
	return nil
}
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

type queryFakeDBTX struct {
	queryCalls []recordedCall
	queryFn    func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (f *queryFakeDBTX) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("unexpected Exec call")
}

func (f *queryFakeDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queryCalls = append(f.queryCalls, recordedCall{sql: sql, args: args})
	if f.queryFn != nil {
		return f.queryFn(ctx, sql, args...)
	}
	return &fakeRows{}, nil
}

func (f *queryFakeDBTX) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return fakeRow{}
}

func TestNormalizeFoodPriceLimit(t *testing.T) {
	if got := normalizeFoodPriceLimit(-10); got != defaultFoodPriceLimit {
		t.Errorf("normalizeFoodPriceLimit(-10) = %d, want %d", got, defaultFoodPriceLimit)
	}
	if got := normalizeFoodPriceLimit(0); got != defaultFoodPriceLimit {
		t.Errorf("normalizeFoodPriceLimit(0) = %d, want %d", got, defaultFoodPriceLimit)
	}
	if got := normalizeFoodPriceLimit(25); got != 25 {
		t.Errorf("normalizeFoodPriceLimit(25) = %d, want 25", got)
	}
	if got := normalizeFoodPriceLimit(250); got != maxFoodPriceLimit {
		t.Errorf("normalizeFoodPriceLimit(250) = %d, want %d", got, maxFoodPriceLimit)
	}
}

func TestFoodPriceRepo(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	sampleRows := []FoodPrice{
		{
			ID:        "11111111-1111-1111-1111-111111111111",
			Name:      "Nasi Putih",
			PriceIDR:  4000,
			Unit:      "1 porsi",
			Category:  FoodPriceCategoryMeal,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "22222222-2222-2222-2222-222222222222",
			Name:      "Nasi Telor Dadar",
			PriceIDR:  10000,
			Unit:      "1 porsi",
			Category:  FoodPriceCategoryMeal,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	t.Run("ListFoodPricesByBudget with category uses explicit casts and default limit", func(t *testing.T) {
		rows := &fakeRows{items: sampleRows}
		db := &queryFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if !strings.Contains(sql, "price_idr <= $1::INTEGER") ||
					!strings.Contains(sql, "category = $2::TEXT") ||
					!strings.Contains(sql, "ORDER BY price_idr ASC, name ASC") ||
					!strings.Contains(sql, "LIMIT $3::INTEGER") {
					t.Errorf("unexpected SQL for ListFoodPricesByBudget: %s", sql)
				}
				if len(args) != 3 || args[0] != 15000 || args[1] != "meal" || args[2] != defaultFoodPriceLimit {
					t.Errorf("unexpected args: %#v", args)
				}
				return rows, nil
			},
		}

		store := NewFoodPriceRepoWithDBTX(db)
		got, err := store.ListFoodPricesByBudget(ctx, 15000, " meal ", 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !rows.closed {
			t.Errorf("expected rows.Close() to be called")
		}
		if len(got) != 2 || got[0].Name != "Nasi Putih" || got[1].PriceIDR != 10000 {
			t.Fatalf("unexpected rows returned: %+v", got)
		}
	})

	t.Run("ListFoodPricesByBudget without category clamps limit and skips negative budget", func(t *testing.T) {
		db := &queryFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if strings.Contains(sql, "AND category =") || !strings.Contains(sql, "LIMIT $2::INTEGER") {
					t.Errorf("unexpected SQL without category: %s", sql)
				}
				if len(args) != 2 || args[0] != 12000 || args[1] != maxFoodPriceLimit {
					t.Errorf("unexpected args: %#v", args)
				}
				return &fakeRows{}, nil
			},
		}
		store := NewFoodPriceRepoWithDBTX(db)

		empty, err := store.ListFoodPricesByBudget(ctx, -500, "", 10)
		if err != nil {
			t.Fatalf("unexpected error for negative budget: %v", err)
		}
		if len(empty) != 0 || len(db.queryCalls) != 0 {
			t.Fatalf("expected negative budget to short-circuit without DB query")
		}

		got, err := store.ListFoodPricesByBudget(ctx, 12000, "", 999)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("expected empty slice, got %+v", got)
		}
	})

	t.Run("ListAllFoodPrices supports optional category and propagates errors", func(t *testing.T) {
		db := &queryFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if len(args) == 2 {
					if !strings.Contains(sql, "WHERE category = $1::TEXT") || !strings.Contains(sql, "LIMIT $2::INTEGER") {
						t.Errorf("unexpected ListAllFoodPrices SQL with category: %s", sql)
					}
					return &fakeRows{items: sampleRows}, nil
				}
				if !strings.Contains(sql, "LIMIT $1::INTEGER") {
					t.Errorf("unexpected ListAllFoodPrices SQL without category: %s", sql)
				}
				return &fakeRows{rowsErr: errors.New("cursor failed")}, nil
			},
		}
		store := NewFoodPriceRepoWithDBTX(db)

		got, err := store.ListAllFoodPrices(ctx, "meal", 50)
		if err != nil || len(got) != 2 {
			t.Fatalf("expected 2 items, got %d (err=%v)", len(got), err)
		}

		if _, err := store.ListAllFoodPrices(ctx, "", 50); err == nil {
			t.Fatalf("expected rows.Err() to be propagated")
		}
	})

	t.Run("Nil store returns error instead of panicking", func(t *testing.T) {
		var s *FoodPriceRepo
		if _, err := s.ListFoodPricesByBudget(ctx, 10000, "", 10); err == nil {
			t.Fatal("expected error from nil store ListFoodPricesByBudget")
		}
		if _, err := s.ListAllFoodPrices(ctx, "", 10); err == nil {
			t.Fatal("expected error from nil store ListAllFoodPrices")
		}
	})
}
