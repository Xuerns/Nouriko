package database

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeRows implements pgx.Rows for testing multi-row query methods on Store.
type fakeRows struct {
	pgx.Rows
	items   []FoodPrice
	idx     int
	closed  bool
	scanErr error
	rowsErr error
}

func (r *fakeRows) Close()                        { r.closed = true }
func (r *fakeRows) Err() error                    { return r.rowsErr }
func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("SELECT") }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
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

func TestMigration00003SeedFoodPrices(t *testing.T) {
	raw, err := fs.ReadFile(migrationsFS, "migrations/00003_seed_food_prices.sql")
	if err != nil {
		t.Fatalf("failed to read embedded 00003_seed_food_prices.sql: %v", err)
	}
	content := string(raw)

	upMarker := "-- +goose Up"
	downMarker := "-- +goose Down"
	upIdx := strings.Index(content, upMarker)
	downIdx := strings.Index(content, downMarker)
	if upIdx == -1 || downIdx == -1 || downIdx <= upIdx {
		t.Fatalf("00003_seed_food_prices.sql must contain '-- +goose Up' followed by '-- +goose Down'")
	}

	upSQL := content[upIdx:downIdx]
	downSQL := content[downIdx:]

	requiredUpSnippets := []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_food_prices_name_category",
		"ON food_prices (name, category)",
		"INSERT INTO food_prices (name, price_idr, unit, category) VALUES",
		"ON CONFLICT (name, category) DO UPDATE",
		"price_idr = EXCLUDED.price_idr",
		"unit = EXCLUDED.unit",
		"updated_at = NOW()",
	}
	for _, snippet := range requiredUpSnippets {
		if !strings.Contains(upSQL, snippet) {
			t.Errorf("00003 Up SQL missing required snippet: %q", snippet)
		}
	}

	// Parse seeded tuples: ('Name', 12000, '1 porsi', 'meal')
	upTupleRe := regexp.MustCompile(`\('([^']+)',\s*(\d+),\s*'([^']+)',\s*'(meal|ingredient)'\)`)
	upMatches := upTupleRe.FindAllStringSubmatch(upSQL, -1)
	if len(upMatches) != 55 {
		t.Fatalf("expected exactly 55 seeded items in 00003 Up SQL, found %d", len(upMatches))
	}

	mealCount := 0
	ingredientCount := 0
	upKeys := make(map[string]struct{}, len(upMatches))
	for _, m := range upMatches {
		name := m[1]
		category := m[4]
		key := name + "|" + category
		if _, dup := upKeys[key]; dup {
			t.Errorf("duplicate seeded item found in 00003 Up SQL: %s", key)
		}
		upKeys[key] = struct{}{}
		switch category {
		case FoodPriceCategoryMeal:
			mealCount++
		case FoodPriceCategoryIngredient:
			ingredientCount++
		}
	}
	if mealCount != 22 {
		t.Errorf("expected 22 meal items, got %d", mealCount)
	}
	if ingredientCount != 33 {
		t.Errorf("expected 33 ingredient items, got %d", ingredientCount)
	}

	// Verify Down migration safety and symmetry.
	if strings.Contains(strings.ToUpper(downSQL), "DROP TABLE") {
		t.Errorf("00003 Down SQL must not drop food_prices table")
	}
	if !strings.Contains(downSQL, "DROP INDEX IF EXISTS uq_food_prices_name_category;") {
		t.Errorf("00003 Down SQL must drop uq_food_prices_name_category")
	}

	downTupleRe := regexp.MustCompile(`\('([^']+)',\s*'(meal|ingredient)'\)`)
	downMatches := downTupleRe.FindAllStringSubmatch(downSQL, -1)
	if len(downMatches) != 55 {
		t.Fatalf("expected 55 delete tuples in 00003 Down SQL, found %d", len(downMatches))
	}
	for _, m := range downMatches {
		key := m[1] + "|" + m[2]
		if _, ok := upKeys[key]; !ok {
			t.Errorf("00003 Down SQL deletes unknown key not in Up SQL: %s", key)
		}
	}
}

func TestEscapeLikePatternAndLimit(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "ayam", want: "ayam"},
		{in: "100%", want: `100\%`},
		{in: "nasi_goreng", want: `nasi\_goreng`},
		{in: `kol\kubis`, want: `kol\\kubis`},
		{in: `50%_off\test`, want: `50\%\_off\\test`},
	}
	for _, tc := range tests {
		if got := escapeLikePattern(tc.in); got != tc.want {
			t.Errorf("escapeLikePattern(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

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

func TestStoreFoodPriceRepository(t *testing.T) {
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

		store := NewStoreWithDBTX(db)
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
		store := NewStoreWithDBTX(db)

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

	t.Run("SearchFoodPrices escapes wildcards and uses ILIKE ESCAPE", func(t *testing.T) {
		db := &queryFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if !strings.Contains(sql, `name ILIKE $1::TEXT ESCAPE '\'`) ||
					!strings.Contains(sql, "category = $2::TEXT") ||
					!strings.Contains(sql, "LIMIT $3::INTEGER") {
					t.Errorf("unexpected SearchFoodPrices SQL: %s", sql)
				}
				wantPattern := `%50\%\_ayam\\goreng%`
				if len(args) != 3 || args[0] != wantPattern || args[1] != "ingredient" || args[2] != 15 {
					t.Errorf("unexpected SearchFoodPrices args: %#v (want pattern %q)", args, wantPattern)
				}
				return &fakeRows{items: sampleRows[:1]}, nil
			},
		}

		store := NewStoreWithDBTX(db)
		got, err := store.SearchFoodPrices(ctx, `  50%_ayam\goreng  `, "ingredient", 15)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 row, got %d", len(got))
		}

		// Blank query short-circuits cleanly.
		blank, err := store.SearchFoodPrices(ctx, "   ", "", 10)
		if err != nil {
			t.Fatalf("unexpected error for blank query: %v", err)
		}
		if len(blank) != 0 || len(db.queryCalls) != 1 {
			t.Fatalf("expected blank search query to short-circuit")
		}
	})

	t.Run("SearchFoodPrices without category uses 2 parameters", func(t *testing.T) {
		db := &queryFakeDBTX{
			queryFn: func(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
				if strings.Contains(sql, "AND category =") || !strings.Contains(sql, "LIMIT $2::INTEGER") {
					t.Errorf("unexpected SearchFoodPrices SQL without category: %s", sql)
				}
				if len(args) != 2 || args[0] != "%tempe%" || args[1] != defaultFoodPriceLimit {
					t.Errorf("unexpected args: %#v", args)
				}
				return &fakeRows{}, nil
			},
		}
		store := NewStoreWithDBTX(db)
		if _, err := store.SearchFoodPrices(ctx, "tempe", "", 0); err != nil {
			t.Fatalf("unexpected error: %v", err)
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
		store := NewStoreWithDBTX(db)

		got, err := store.ListAllFoodPrices(ctx, "meal", 50)
		if err != nil || len(got) != 2 {
			t.Fatalf("expected 2 items, got %d (err=%v)", len(got), err)
		}

		if _, err := store.ListAllFoodPrices(ctx, "", 50); err == nil {
			t.Fatalf("expected rows.Err() to be propagated")
		}
	})

	t.Run("Nil store returns error instead of panicking", func(t *testing.T) {
		var s *Store
		if _, err := s.ListFoodPricesByBudget(ctx, 10000, "", 10); err == nil {
			t.Fatal("expected error from nil store ListFoodPricesByBudget")
		}
		if _, err := s.SearchFoodPrices(ctx, "ayam", "", 10); err == nil {
			t.Fatal("expected error from nil store SearchFoodPrices")
		}
		if _, err := s.ListAllFoodPrices(ctx, "", 10); err == nil {
			t.Fatal("expected error from nil store ListAllFoodPrices")
		}
	})
}
