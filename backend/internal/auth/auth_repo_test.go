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

func TestLocalAuthRepo_Users(t *testing.T) {
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
