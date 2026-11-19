package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX abstracts pgxpool.Pool and pgx.Tx so repository methods can execute
// both standalone and inside transactions.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// LocalUserStore defines persistence operations for the local_users table.
type LocalUserStore interface {
	CreateLocalUser(ctx context.Context, params CreateLocalUserParams) (*LocalUser, error)
	GetLocalUserByEmail(ctx context.Context, email string) (*LocalUser, error)
	GetLocalUserByID(ctx context.Context, userID string) (*LocalUser, error)
	RecordFailedLogin(ctx context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*LocalUser, error)
	ResetLoginLockout(ctx context.Context, userID string) error
	MarkEmailVerified(ctx context.Context, userID string) error
	UpdatePasswordHash(ctx context.Context, userID, passwordHash string, resetLockout bool) error
	DeleteLocalUser(ctx context.Context, userID string) error
}

// LocalAuthTokenStore defines persistence operations for the local_auth_tokens table.
type LocalAuthTokenStore interface {
	CreateAuthToken(ctx context.Context, params CreateAuthTokenParams) (*LocalAuthToken, error)
	InvalidateUserTokens(ctx context.Context, userID, purpose string) error
	ConsumeAuthToken(ctx context.Context, tokenHash, purpose string) (*LocalAuthToken, error)
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}

// ProfileStore defines persistence operations for the profiles table.
type ProfileStore interface {
	UpsertProfile(ctx context.Context, params UpsertProfileParams) (*Profile, error)
	GetProfileByID(ctx context.Context, userID string) (*Profile, error)
	UpdateOnboarding(ctx context.Context, params UpdateOnboardingParams) (*Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*Profile, error)
	DeleteProfile(ctx context.Context, userID string) error
}

// FoodPriceStore defines read operations for the food_prices reference table.
type FoodPriceStore interface {
	ListFoodPricesByBudget(ctx context.Context, maxBudgetIDR int, category string, limit int) ([]FoodPrice, error)
	SearchFoodPrices(ctx context.Context, query string, category string, limit int) ([]FoodPrice, error)
	ListAllFoodPrices(ctx context.Context, category string, limit int) ([]FoodPrice, error)
}

// Repository aggregates all persistence interfaces and atomic multi-table
// transaction helpers.
type Repository interface {
	LocalUserStore
	LocalAuthTokenStore
	ProfileStore
	FoodPriceStore
	VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error)
	ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error)
	DeleteLocalAccount(ctx context.Context, userID string) error
}

// Store implements Repository backed by PostgreSQL (pgx/v5).
type Store struct {
	pool *pgxpool.Pool
	q    DBTX
}

// Compile-time check ensuring Store implements Repository.
var _ Repository = (*Store)(nil)

// NewStore creates a PostgreSQL-backed Store from a connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
		q:    pool,
	}
}

// NewStoreWithDBTX creates a Store bound to an existing DBTX (such as a pgx.Tx
// or a test double).
func NewStoreWithDBTX(q DBTX) *Store {
	return &Store{
		q: q,
	}
}

// WithTx executes fn inside a database transaction. If fn returns an error or
// panics, the transaction is rolled back.
func (s *Store) WithTx(ctx context.Context, fn func(txStore *Store) error) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	txStore := NewStoreWithDBTX(tx)
	if err := fn(txStore); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// VerifyEmailWithToken atomically consumes a single-use verify_email token and
// marks the associated local_users row as email_verified = TRUE.
func (s *Store) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	run := func(st *Store) error {
		tok, err := st.ConsumeAuthToken(ctx, tokenHash, TokenPurposeVerifyEmail)
		if err != nil {
			return err
		}
		if err := st.MarkEmailVerified(ctx, tok.UserID); err != nil {
			return err
		}
		userID = tok.UserID
		return nil
	}

	if s.pool != nil {
		if err := s.WithTx(ctx, run); err != nil {
			return "", err
		}
		return userID, nil
	}

	if err := run(s); err != nil {
		return "", err
	}
	return userID, nil
}

// ResetPasswordWithToken atomically consumes a single-use reset_password token,
// updates the user's password hash, and clears any active login lockout.
func (s *Store) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	var userID string
	run := func(st *Store) error {
		tok, err := st.ConsumeAuthToken(ctx, tokenHash, TokenPurposeResetPassword)
		if err != nil {
			return err
		}
		if err := st.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
			return err
		}
		userID = tok.UserID
		return nil
	}

	if s.pool != nil {
		if err := s.WithTx(ctx, run); err != nil {
			return "", err
		}
		return userID, nil
	}

	if err := run(s); err != nil {
		return "", err
	}
	return userID, nil
}

// DeleteLocalAccount atomically deletes a user's profile (cascading saved_meals)
// and local_users record (cascading local_auth_tokens) in a single transaction.
func (s *Store) DeleteLocalAccount(ctx context.Context, userID string) error {
	run := func(st *Store) error {
		if err := st.DeleteProfile(ctx, userID); err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := st.DeleteLocalUser(ctx, userID); err != nil {
			return err
		}
		return nil
	}

	if s.pool != nil {
		return s.WithTx(ctx, run)
	}
	return run(s)
}
