package auth

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

type fakeRows struct {
	pgx.Rows
	values [][]any
	idx    int
	closed bool
}

func (r *fakeRows) Close()                                       { r.closed = true }
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.NewCommandTag("SELECT") }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if r.idx >= len(r.values) {
		return false
	}
	r.idx++
	return true
}
func (r *fakeRows) Scan(dest ...any) error {
	row := r.values[r.idx-1]
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = row[i].(string)
		default:
			return errors.New("unsupported fakeRows scan type")
		}
	}
	return nil
}
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte    { return nil }
func (r *fakeRows) Conn() *pgx.Conn        { return nil }

type fakeDBTX struct {
	execCalls     []recordedCall
	queryCalls    []recordedCall
	queryRowCalls []recordedCall
	execFn        func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryFn       func(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	queryRowFn    func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *fakeDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.execCalls = append(f.execCalls, recordedCall{sql: sql, args: args})
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (f *fakeDBTX) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	f.queryCalls = append(f.queryCalls, recordedCall{sql: sql, args: args})
	if f.queryFn != nil {
		return f.queryFn(ctx, sql, args...)
	}
	return nil, errors.New("unexpected Query call")
}

func (f *fakeDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	f.queryRowCalls = append(f.queryRowCalls, recordedCall{sql: sql, args: args})
	if f.queryRowFn != nil {
		return f.queryRowFn(ctx, sql, args...)
	}
	return fakeRow{}
}

func TestLocalAuthRepo_Users(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("CreateLocalUser normalizes email and maps unique violation to ErrConflict", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, _ string, _ ...any) pgx.Row {
				return fakeRow{scanFn: func(dest ...any) error {
					hash := "$2a$12$hash"
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = "raka@example.com"
					*(dest[2].(**string)) = &hash
					*(dest[3].(*bool)) = false
					*(dest[4].(*int16)) = 0
					*(dest[5].(**time.Time)) = nil
					*(dest[6].(*time.Time)) = now
					*(dest[7].(*time.Time)) = now
					return nil
				}}
			},
		}
		repo := NewLocalAuthRepoWithDBTX(db)
		u, err := repo.CreateLocalUser(ctx, CreateLocalUserParams{
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
		_, err = repo.CreateLocalUser(ctx, CreateLocalUserParams{
			ID:           userID,
			Email:        "raka@example.com",
			PasswordHash: "$2a$12$hash",
		})
		if !errors.Is(err, database.ErrConflict) {
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
					hash := "$2a$12$hash"
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = "raka@example.com"
					*(dest[2].(**string)) = &hash
					*(dest[3].(*bool)) = true
					*(dest[4].(*int16)) = 5
					*(dest[5].(**time.Time)) = &lockedUntil
					*(dest[6].(*time.Time)) = now
					*(dest[7].(*time.Time)) = now
					return nil
				}}
			},
		}
		repo := NewLocalAuthRepoWithDBTX(db)
		u, err := repo.RecordFailedLogin(ctx, userID, 5, 15*time.Minute)
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
		repo := NewLocalAuthRepoWithDBTX(db)
		if err := repo.ResetLoginLockout(ctx, userID); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("ResetLoginLockout expected ErrNotFound, got %v", err)
		}
		if err := repo.MarkEmailVerified(ctx, userID); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("MarkEmailVerified expected ErrNotFound, got %v", err)
		}
		if err := repo.UpdatePasswordHash(ctx, userID, "newhash", true); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("UpdatePasswordHash expected ErrNotFound, got %v", err)
		}
		if err := repo.DeleteLocalUser(ctx, userID); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("DeleteLocalUser expected ErrNotFound, got %v", err)
		}
	})
}

func TestLocalAuthRepo_TokensAndAccountDelete(t *testing.T) {
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
		repo := NewLocalAuthRepoWithDBTX(db)
		_, err := repo.ConsumeAuthToken(ctx, "sha256hash", TokenPurposeVerifyEmail)
		if !errors.Is(err, database.ErrTokenInvalidOrExpired) {
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
		repo := NewLocalAuthRepoWithDBTX(db)

		gotUserID, err := repo.VerifyEmailWithToken(ctx, "verify-hash")
		if err != nil || gotUserID != userID {
			t.Fatalf("VerifyEmailWithToken failed: userID=%q err=%v", gotUserID, err)
		}

		gotUserID, err = repo.ResetPasswordWithToken(ctx, "reset-hash", "$2a$12$newhash")
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
		repo := NewLocalAuthRepoWithDBTX(db)
		deleted, err := repo.DeleteExpiredTokens(ctx)
		if err != nil || deleted != 4 {
			t.Fatalf("expected 4 deleted tokens, got %d (err=%v)", deleted, err)
		}
	})

	t.Run("DeleteLocalAccount deletes profile then local_users", func(t *testing.T) {
		db := &fakeDBTX{
			execFn: func(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
				return pgconn.NewCommandTag("DELETE 1"), nil
			},
		}
		repo := NewLocalAuthRepoWithDBTX(db)
		if err := repo.DeleteLocalAccount(ctx, userID); err != nil {
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

	t.Run("Nil repo returns error instead of panicking", func(t *testing.T) {
		var r *LocalAuthRepo
		if _, err := r.GetLocalUserByID(ctx, userID); err == nil {
			t.Fatal("expected error from nil repo")
		}
	})
}

func TestLocalAuthRepo_GoogleOAuthTransactions(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"

	t.Run("FindOrCreateGoogleUser clears unverified password and invalidates all tokens (pre-takeover mitigation)", func(t *testing.T) {
		var clearedPassword, invalidatedTokens, insertedIdentity bool
		oldHash := "$2a$12$attackerhash"

		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
				switch {
				case strings.Contains(sql, "FROM local_user_identities"):
					return fakeRow{} // not linked yet
				case strings.Contains(sql, "WHERE LOWER(email)"):
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "victim@gmail.com"
						*(dest[2].(**string)) = &oldHash
						*(dest[3].(*bool)) = false // unverified!
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				case strings.Contains(sql, "SET password_hash = NULL, email_verified = TRUE"):
					clearedPassword = true
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "victim@gmail.com"
						*(dest[2].(**string)) = nil
						*(dest[3].(*bool)) = true
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				case strings.Contains(sql, "WHERE id = $1::UUID"):
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "victim@gmail.com"
						*(dest[2].(**string)) = nil
						*(dest[3].(*bool)) = true
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				default:
					return fakeRow{}
				}
			},
			execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
				if strings.Contains(sql, "UPDATE local_auth_tokens") && strings.Contains(sql, "used_at IS NULL") {
					invalidatedTokens = true
				}
				if strings.Contains(sql, "INSERT INTO local_user_identities") {
					insertedIdentity = true
				}
				return pgconn.NewCommandTag("UPDATE 1"), nil
			},
			queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
				return &fakeRows{values: [][]any{{"google"}}}, nil
			},
		}

		repo := NewLocalAuthRepoWithDBTX(db)
		u, providers, err := repo.FindOrCreateGoogleUser(ctx, UpsertGoogleUserParams{
			Email:          "victim@gmail.com",
			ProviderUserID: "google-sub-victim",
		})
		if err != nil {
			t.Fatalf("FindOrCreateGoogleUser failed: %v", err)
		}
		if !clearedPassword || !invalidatedTokens || !insertedIdentity {
			t.Fatalf("expected pre-takeover mitigation steps: cleared=%v invalidated=%v inserted=%v", clearedPassword, invalidatedTokens, insertedIdentity)
		}
		if u.HasPassword() || !u.EmailVerified {
			t.Fatalf("expected passwordless verified user, got %+v", u)
		}
		if len(providers) != 1 || providers[0] != "google" {
			t.Fatalf("expected providers=[google], got %v", providers)
		}
	})

	t.Run("FindOrCreateGoogleUser creates new passwordless verified user when absent", func(t *testing.T) {
		var insertedUser, insertedIdentity bool
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
				switch {
				case strings.Contains(sql, "FROM local_user_identities"):
					return fakeRow{}
				case strings.Contains(sql, "WHERE LOWER(email)"):
					return fakeRow{}
				case strings.Contains(sql, "INSERT INTO local_users"):
					insertedUser = true
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "new@gmail.com"
						*(dest[2].(**string)) = nil
						*(dest[3].(*bool)) = true
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				case strings.Contains(sql, "WHERE id = $1::UUID"):
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "new@gmail.com"
						*(dest[2].(**string)) = nil
						*(dest[3].(*bool)) = true
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				default:
					return fakeRow{}
				}
			},
			execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
				if strings.Contains(sql, "INSERT INTO local_user_identities") {
					insertedIdentity = true
				}
				return pgconn.NewCommandTag("INSERT 1"), nil
			},
			queryFn: func(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
				return &fakeRows{values: [][]any{{"google"}}}, nil
			},
		}

		repo := NewLocalAuthRepoWithDBTX(db)
		u, providers, err := repo.FindOrCreateGoogleUser(ctx, UpsertGoogleUserParams{
			Email:          "new@gmail.com",
			ProviderUserID: "google-sub-new",
		})
		if err != nil {
			t.Fatalf("FindOrCreateGoogleUser failed: %v", err)
		}
		if !insertedUser || !insertedIdentity {
			t.Fatalf("expected user + identity inserts, got user=%v identity=%v", insertedUser, insertedIdentity)
		}
		if u.HasPassword() || !u.EmailVerified || len(providers) != 1 || providers[0] != "google" {
			t.Fatalf("unexpected new google user: u=%+v providers=%v", u, providers)
		}
	})

	t.Run("FindOrCreateGoogleUser returns ErrConflict when provider_user_id is concurrently bound to another user", func(t *testing.T) {
		otherUserID := "660e8400-e29b-41d4-a716-446655440000"
		identityLookupCalls := 0
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, _ ...any) pgx.Row {
				switch {
				case strings.Contains(sql, "FROM local_user_identities"):
					identityLookupCalls++
					if identityLookupCalls == 1 {
						return fakeRow{} // Step 1 misses before concurrent tx commits
					}
					// Step 2 re-check after ON CONFLICT DO NOTHING sees sub bound to otherUserID!
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = "770e8400-e29b-41d4-a716-446655440000"
						*(dest[1].(*string)) = otherUserID
						*(dest[2].(*string)) = OAuthProviderGoogle
						*(dest[3].(*string)) = "google-sub-race"
						*(dest[4].(*string)) = "race@gmail.com"
						*(dest[5].(*time.Time)) = now
						return nil
					}}
				case strings.Contains(sql, "WHERE LOWER(email)"):
					return fakeRow{scanFn: func(dest ...any) error {
						*(dest[0].(*string)) = userID
						*(dest[1].(*string)) = "race@gmail.com"
						*(dest[2].(**string)) = nil
						*(dest[3].(*bool)) = true
						*(dest[4].(*int16)) = 0
						*(dest[5].(**time.Time)) = nil
						*(dest[6].(*time.Time)) = now
						*(dest[7].(*time.Time)) = now
						return nil
					}}
				default:
					return fakeRow{}
				}
			},
			execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
				if strings.Contains(sql, "INSERT INTO local_user_identities") {
					return pgconn.NewCommandTag("INSERT 0 0"), nil
				}
				return pgconn.NewCommandTag("UPDATE 1"), nil
			},
		}

		repo := NewLocalAuthRepoWithDBTX(db)
		if _, _, err := repo.FindOrCreateGoogleUser(ctx, UpsertGoogleUserParams{
			Email:          "race@gmail.com",
			ProviderUserID: "google-sub-race",
		}); !errors.Is(err, database.ErrConflict) {
			t.Fatalf("expected database.ErrConflict on concurrent sub divergence, got %v", err)
		}
	})

	t.Run("ConsumeOAuthNonce enforces single-use insertion", func(t *testing.T) {
		inserted := false
		db := &fakeDBTX{
			execFn: func(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
				if strings.Contains(sql, "DELETE FROM oauth_consumed_nonces") {
					return pgconn.NewCommandTag("DELETE 0"), nil
				}
				if strings.Contains(sql, "INSERT INTO oauth_consumed_nonces") {
					if !inserted {
						inserted = true
						return pgconn.NewCommandTag("INSERT 0 1"), nil
					}
					return pgconn.NewCommandTag("INSERT 0 0"), nil
				}
				return pgconn.NewCommandTag("OK"), nil
			},
		}

		repo := NewLocalAuthRepoWithDBTX(db)
		first, err := repo.ConsumeOAuthNonce(ctx, "noncehash123", now.Add(time.Hour))
		if err != nil || !first {
			t.Fatalf("expected first ConsumeOAuthNonce=true, got %v err=%v", first, err)
		}
		second, err := repo.ConsumeOAuthNonce(ctx, "noncehash123", now.Add(time.Hour))
		if err != nil || second {
			t.Fatalf("expected second ConsumeOAuthNonce=false, got %v err=%v", second, err)
		}
	})
}
