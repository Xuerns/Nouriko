package database

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
}

func TestProfileOnboardingCompleted(t *testing.T) {
	var nilProfile *Profile
	if nilProfile.OnboardingCompleted() {
		t.Fatal("expected nil profile to have OnboardingCompleted() == false")
	}

	h := 170.0
	w := 65.0
	age := int16(20)
	gender := "male"
	activity := "moderate"
	kcal := 2523

	incomplete := &Profile{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		HeightCM: &h,
		WeightKG: &w,
		Age:      &age,
		Gender:   &gender,
		Language: "id",
		Theme:    "light",
	}
	if incomplete.OnboardingCompleted() {
		t.Fatal("expected incomplete profile to return OnboardingCompleted() == false")
	}

	complete := &Profile{
		ID:                 "550e8400-e29b-41d4-a716-446655440000",
		HeightCM:           &h,
		WeightKG:           &w,
		Age:                &age,
		Gender:             &gender,
		ActivityLevel:      &activity,
		DailyCalorieTarget: &kcal,
		Language:           "id",
		Theme:              "light",
	}
	if !complete.OnboardingCompleted() {
		t.Fatal("expected complete profile to return OnboardingCompleted() == true")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if !isUniqueViolation(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("expected SQLSTATE 23505 to be recognized as unique violation")
	}
	if isUniqueViolation(&pgconn.PgError{Code: "23503"}) {
		t.Fatal("expected SQLSTATE 23503 not to be recognized as unique violation")
	}
	if isUniqueViolation(errors.New("generic error")) {
		t.Fatal("expected generic error not to be recognized as unique violation")
	}
}

// --- Lightweight DBTX test double for verifying SQL queries & error mapping ---

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanFn == nil {
		return pgx.ErrNoRows
	}
	return r.scanFn(dest...)
}

type recordedCall struct {
	sql  string
	args []any
}

type fakeDBTX struct {
	execCalls     []recordedCall
	queryRowCalls []recordedCall
	execFn        func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryRowFn    func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *fakeDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execCalls = append(f.execCalls, recordedCall{sql: sql, args: args})
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (f *fakeDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query call")
}

func (f *fakeDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowCalls = append(f.queryRowCalls, recordedCall{sql: sql, args: args})
	if f.queryRowFn != nil {
		return f.queryRowFn(ctx, sql, args...)
	}
	return fakeRow{}
}

func TestStoreLocalUserRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("CreateLocalUser normalizes email and maps unique violation to ErrConflict", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = "raka@example.com"
					*(dest[2].(*string)) = "$2a$12$hash"
					*(dest[3].(*bool)) = false
					*(dest[4].(*int16)) = 0
					*(dest[5].(**time.Time)) = nil
					*(dest[6].(*time.Time)) = now
					*(dest[7].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		u, err := store.CreateLocalUser(ctx, CreateLocalUserParams{
			Email:        "  Raka@Example.COM  ",
			PasswordHash: "$2a$12$hash",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Email != "raka@example.com" {
			t.Fatalf("expected normalized email, got %q", u.Email)
		}
		if gotArg, ok := db.queryRowCalls[0].args[0].(string); !ok || gotArg != "raka@example.com" {
			t.Fatalf("expected normalized email arg, got %#v", db.queryRowCalls[0].args[0])
		}

		// Conflict case
		db.queryRowFn = func(_ context.Context, _ string, _ ...any) pgx.Row {
			return fakeRow{scanFn: func(_ ...any) error {
				return &pgconn.PgError{Code: "23505"}
			}}
		}
		_, err = store.CreateLocalUser(ctx, CreateLocalUserParams{
			ID:           userID,
			Email:        "raka@example.com",
			PasswordHash: "$2a$12$hash",
		})
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("expected ErrConflict, got %v", err)
		}
	})

	t.Run("RecordFailedLogin passes maxAttempts and lockDuration ms", func(t *testing.T) {
		lockedUntil := now.Add(15 * time.Minute)
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, args ...any) pgx.Row {
				if !strings.Contains(sql, "failed_login_attempts + 1") {
					t.Errorf("expected atomic increment SQL, got %s", sql)
				}
				if len(args) != 3 || args[1] != int16(5) || args[2] != int64(15*time.Minute/time.Millisecond) {
					t.Errorf("unexpected RecordFailedLogin args: %#v", args)
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = "raka@example.com"
					*(dest[2].(*string)) = "$2a$12$hash"
					*(dest[3].(*bool)) = true
					*(dest[4].(*int16)) = 5
					*(dest[5].(**time.Time)) = &lockedUntil
					*(dest[6].(*time.Time)) = now
					*(dest[7].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		u, err := store.RecordFailedLogin(ctx, userID, 5, 15*time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.FailedLoginAttempts != 5 || u.LockedUntil == nil {
			t.Fatalf("expected 5 failed attempts and non-nil locked_until, got %+v", u)
		}
	})

	t.Run("ResetLoginLockout, MarkEmailVerified, UpdatePasswordHash, DeleteLocalUser map 0 rows to ErrNotFound", func(t *testing.T) {
		db := &fakeDBTX{
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 0"), nil
			},
		}
		store := NewStoreWithDBTX(db)
		if err := store.ResetLoginLockout(ctx, userID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("ResetLoginLockout expected ErrNotFound, got %v", err)
		}
		if err := store.MarkEmailVerified(ctx, userID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("MarkEmailVerified expected ErrNotFound, got %v", err)
		}
		if err := store.UpdatePasswordHash(ctx, userID, "newhash", true); !errors.Is(err, ErrNotFound) {
			t.Fatalf("UpdatePasswordHash expected ErrNotFound, got %v", err)
		}
		if err := store.DeleteLocalUser(ctx, userID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("DeleteLocalUser expected ErrNotFound, got %v", err)
		}
	})
}

func TestStoreAuthTokenRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("ConsumeAuthToken uses single-use atomic UPDATE and maps ErrNoRows to ErrTokenInvalidOrExpired", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, args ...any) pgx.Row {
				if !strings.Contains(sql, "used_at IS NULL") || !strings.Contains(sql, "expires_at > NOW()") {
					t.Errorf("ConsumeAuthToken SQL missing single-use or expiry guard: %s", sql)
				}
				if len(args) != 2 || args[0] != "sha256hash" || args[1] != TokenPurposeVerifyEmail {
					t.Errorf("unexpected ConsumeAuthToken args: %#v", args)
				}
				return fakeRow{scanFn: func(_ ...any) error {
					return pgx.ErrNoRows
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		_, err := store.ConsumeAuthToken(ctx, "sha256hash", TokenPurposeVerifyEmail)
		if !errors.Is(err, ErrTokenInvalidOrExpired) {
			t.Fatalf("expected ErrTokenInvalidOrExpired, got %v", err)
		}
	})

	t.Run("VerifyEmailWithToken and ResetPasswordWithToken coordinate token consumption and user update", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = "tok-1"
					*(dest[1].(*string)) = userID
					*(dest[2].(*string)) = args[0].(string)
					*(dest[3].(*string)) = args[1].(string)
					*(dest[4].(*time.Time)) = now.Add(time.Hour)
					*(dest[5].(**time.Time)) = &now
					*(dest[6].(*time.Time)) = now
					return nil
				}}
			},
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("UPDATE 1"), nil
			},
		}
		store := NewStoreWithDBTX(db)

		gotUserID, err := store.VerifyEmailWithToken(ctx, "verify-hash")
		if err != nil || gotUserID != userID {
			t.Fatalf("VerifyEmailWithToken failed: userID=%q err=%v", gotUserID, err)
		}

		gotUserID, err = store.ResetPasswordWithToken(ctx, "reset-hash", "$2a$12$newhash")
		if err != nil || gotUserID != userID {
			t.Fatalf("ResetPasswordWithToken failed: userID=%q err=%v", gotUserID, err)
		}
		if !strings.Contains(db.execCalls[1].sql, "failed_login_attempts = 0") {
			t.Fatalf("expected ResetPasswordWithToken to clear login lockout, got SQL: %s", db.execCalls[1].sql)
		}
	})

	t.Run("DeleteExpiredTokens returns rows affected", func(t *testing.T) {
		db := &fakeDBTX{
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 4"), nil
			},
		}
		store := NewStoreWithDBTX(db)
		deleted, err := store.DeleteExpiredTokens(ctx)
		if err != nil || deleted != 4 {
			t.Fatalf("expected 4 deleted tokens, got %d (err=%v)", deleted, err)
		}
	})
}

func TestStoreProfileRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"
	avatarURL := "https://api.dicebear.com/10.x/slice/svg?seed=" + userID

	t.Run("UpsertProfile applies defaults for language and theme", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
				if args[2] != "id" || args[3] != "light" {
					t.Errorf("expected default language=id and theme=light, got %#v %#v", args[2], args[3])
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = avatarURL
					*(dest[12].(*string)) = "id"
					*(dest[13].(*string)) = "light"
					*(dest[14].(*time.Time)) = now
					*(dest[15].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		p, err := store.UpsertProfile(ctx, UpsertProfileParams{
			ID:         userID,
			ProfileURL: avatarURL,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ProfileURL != avatarURL || p.OnboardingCompleted() {
			t.Fatalf("unexpected profile result: %+v", p)
		}
	})

	t.Run("UpdateProfile uses explicit PostgreSQL casts for COALESCE compatibility with Supavisor", func(t *testing.T) {
		lang := "en"
		theme := "dark"
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, args ...any) pgx.Row {
				if !strings.Contains(sql, "COALESCE($13::TEXT, language)") || !strings.Contains(sql, "COALESCE($14::TEXT, theme)") {
					t.Errorf("expected explicit type casts in UpdateProfile SQL: %s", sql)
				}
				if len(args) != 14 {
					t.Errorf("expected 14 args for UpdateProfile, got %d", len(args))
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = avatarURL
					*(dest[12].(*string)) = lang
					*(dest[13].(*string)) = theme
					*(dest[14].(*time.Time)) = now
					*(dest[15].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		p, err := store.UpdateProfile(ctx, UpdateProfileParams{
			UserID:   userID,
			Language: &lang,
			Theme:    &theme,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Language != "en" || p.Theme != "dark" {
			t.Fatalf("expected updated language/theme, got %+v", p)
		}
	})

	t.Run("DeleteLocalAccount deletes profile then local_users", func(t *testing.T) {
		db := &fakeDBTX{
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		}
		store := NewStoreWithDBTX(db)
		if err := store.DeleteLocalAccount(ctx, userID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(db.execCalls) != 2 {
			t.Fatalf("expected 2 delete statements, got %d", len(db.execCalls))
		}
		if !strings.Contains(db.execCalls[0].sql, "DELETE FROM profiles") {
			t.Errorf("expected first delete on profiles, got %s", db.execCalls[0].sql)
		}
		if !strings.Contains(db.execCalls[1].sql, "DELETE FROM local_users") {
			t.Errorf("expected second delete on local_users, got %s", db.execCalls[1].sql)
		}
	})

	t.Run("Nil store returns error instead of panicking", func(t *testing.T) {
		var s *Store
		if _, err := s.GetProfileByID(ctx, userID); err == nil {
			t.Fatal("expected error from nil store")
		}
		if err := s.WithTx(ctx, func(_ *Store) error { return nil }); err == nil {
			t.Fatal("expected error from nil store WithTx")
		}
	})
}
