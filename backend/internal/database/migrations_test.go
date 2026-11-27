package database

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedMigrationsValidAndPortable(t *testing.T) {
	subFS, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("failed to open embedded migrations: %v", err)
	}

	entries, err := fs.ReadDir(subFS, ".")
	if err != nil {
		t.Fatalf("failed to list embedded migrations: %v", err)
	}

	found := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, readErr := fs.ReadFile(subFS, entry.Name())
		if readErr != nil {
			t.Fatalf("failed to read %s: %v", entry.Name(), readErr)
		}
		found[entry.Name()] = string(data)
	}

	requiredFiles := []string{
		"00001_initial_schema.sql",
		"00002_auth_local.sql",
		"00003_seed_food_prices.sql",
		"00004_profile_display_name.sql",
		"00005_profile_avatar_waves.sql",
		"00006_auth_oauth.sql",
	}
	for _, name := range requiredFiles {
		content, ok := found[name]
		if !ok {
			t.Fatalf("expected embedded migration %s to exist", name)
		}
		if !strings.Contains(content, "-- +goose Up") {
			t.Errorf("%s missing '-- +goose Up' directive", name)
		}
		if !strings.Contains(content, "-- +goose Down") {
			t.Errorf("%s missing '-- +goose Down' directive", name)
		}
		beginCount := strings.Count(content, "-- +goose StatementBegin")
		endCount := strings.Count(content, "-- +goose StatementEnd")
		if beginCount != endCount {
			t.Errorf("%s mismatched StatementBegin (%d) and StatementEnd (%d)", name, beginCount, endCount)
		}
		if strings.Contains(strings.ToLower(content), "citext") {
			t.Errorf("%s must not depend on non-default citext extension", name)
		}
	}

	m2 := found["00002_auth_local.sql"]
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS local_users",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_local_users_email_lower ON local_users (LOWER(email))",
		"CREATE TABLE IF NOT EXISTS local_auth_tokens",
		"REFERENCES local_users(id) ON DELETE CASCADE",
		"CHECK (purpose IN ('verify_email', 'reset_password'))",
		"CREATE INDEX IF NOT EXISTS idx_local_auth_tokens_user_purpose",
		"CREATE INDEX IF NOT EXISTS idx_local_auth_tokens_expires_at",
		"WHERE n.nspname = 'auth' AND p.proname = 'uid'",
		"ALTER TABLE local_users ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE local_auth_tokens ENABLE ROW LEVEL SECURITY",
		"DROP TABLE IF EXISTS local_auth_tokens;",
		"DROP TABLE IF EXISTS local_users;",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(m2, snippet) {
			t.Errorf("00002_auth_local.sql missing expected SQL snippet: %q", snippet)
		}
	}

	m6 := found["00006_auth_oauth.sql"]
	expectedM6Snippets := []string{
		"ALTER TABLE local_users ALTER COLUMN password_hash DROP NOT NULL;",
		"CREATE TABLE IF NOT EXISTS local_user_identities",
		"user_id UUID NOT NULL REFERENCES local_users(id) ON DELETE CASCADE",
		"provider TEXT NOT NULL CHECK (provider IN ('google'))",
		"UNIQUE (provider, provider_user_id)",
		"UNIQUE (user_id, provider)",
		"CREATE INDEX IF NOT EXISTS idx_local_user_identities_user_id ON local_user_identities (user_id);",
		"WHERE n.nspname = 'auth' AND p.proname = 'uid'",
		"ALTER TABLE local_user_identities ENABLE ROW LEVEL SECURITY;",
		"DROP TABLE IF EXISTS local_user_identities;",
	}
	for _, snippet := range expectedM6Snippets {
		if !strings.Contains(m6, snippet) {
			t.Errorf("00006_auth_oauth.sql missing expected SQL snippet: %q", snippet)
		}
	}
}

func TestSavedMealsSchemaInInitialMigration(t *testing.T) {
	raw, err := fs.ReadFile(migrationsFS, "migrations/00001_initial_schema.sql")
	if err != nil {
		t.Fatalf("failed to read 00001_initial_schema.sql: %v", err)
	}
	content := string(raw)

	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS saved_meals",
		"user_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE",
		"meal_date DATE NOT NULL",
		"CHECK (meal_time IN ('breakfast', 'lunch', 'dinner', 'snack'))",
		"CHECK (health_score BETWEEN 0 AND 100)",
		"CHECK (input_source IN ('ingredients', 'budget', 'photo'))",
		"CREATE INDEX IF NOT EXISTS idx_saved_meals_user_date ON saved_meals (user_id, meal_date DESC)",
		"CREATE POLICY saved_meals_owner_all ON saved_meals",
		"USING (auth.uid() = user_id)",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("00001_initial_schema.sql missing expected saved_meals snippet: %q", snippet)
		}
	}
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
		case "meal":
			mealCount++
		case "ingredient":
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
