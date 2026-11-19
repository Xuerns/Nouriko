package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"Nouriko/internal/database"
	"Nouriko/internal/profile"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LocalAuthRepository defines persistence operations for local_users,
// local_auth_tokens, and atomic multi-table local auth transactions.
type LocalAuthRepository interface {
	CreateLocalUser(ctx context.Context, params CreateLocalUserParams) (*LocalUser, error)
	GetLocalUserByEmail(ctx context.Context, email string) (*LocalUser, error)
	GetLocalUserByID(ctx context.Context, userID string) (*LocalUser, error)
	RecordFailedLogin(ctx context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*LocalUser, error)
	ResetLoginLockout(ctx context.Context, userID string) error
	MarkEmailVerified(ctx context.Context, userID string) error
	UpdatePasswordHash(ctx context.Context, userID, passwordHash string, resetLockout bool) error
	DeleteLocalUser(ctx context.Context, userID string) error

	CreateAuthToken(ctx context.Context, params CreateAuthTokenParams) (*LocalAuthToken, error)
	InvalidateUserTokens(ctx context.Context, userID, purpose string) error
	ConsumeAuthToken(ctx context.Context, tokenHash, purpose string) (*LocalAuthToken, error)
	DeleteExpiredTokens(ctx context.Context) (int64, error)

	VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error)
	ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error)
	DeleteLocalAccount(ctx context.Context, userID string) error
}

// LocalAuthRepo implements LocalAuthRepository backed by PostgreSQL (pgx/v5).
type LocalAuthRepo struct {
	pool *pgxpool.Pool
	q    database.DBTX
}

// Compile-time check ensuring LocalAuthRepo implements LocalAuthRepository.
var _ LocalAuthRepository = (*LocalAuthRepo)(nil)

// NewLocalAuthRepo creates a PostgreSQL-backed LocalAuthRepo from a connection pool.
func NewLocalAuthRepo(pool *pgxpool.Pool) *LocalAuthRepo {
	return &LocalAuthRepo{
		pool: pool,
		q:    pool,
	}
}

// NewLocalAuthRepoWithDBTX creates a LocalAuthRepo bound to an existing DBTX
// (such as a pgx.Tx or a test double).
func NewLocalAuthRepoWithDBTX(q database.DBTX) *LocalAuthRepo {
	return &LocalAuthRepo{
		q: q,
	}
}

const localUserColumns = `id, email, password_hash, email_verified, failed_login_attempts, locked_until, created_at, updated_at`

func scanLocalUser(row pgx.Row) (*LocalUser, error) {
	var u LocalUser
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.EmailVerified,
		&u.FailedLoginAttempts,
		&u.LockedUntil,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateLocalUser inserts a new local_users row with normalized lowercase email.
func (r *LocalAuthRepo) CreateLocalUser(ctx context.Context, params CreateLocalUserParams) (*LocalUser, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))
	var row pgx.Row
	if strings.TrimSpace(params.ID) != "" {
		const query = `
			INSERT INTO local_users (id, email, password_hash)
			VALUES ($1::UUID, $2::TEXT, $3::TEXT)
			RETURNING ` + localUserColumns
		row = r.q.QueryRow(ctx, query, params.ID, email, params.PasswordHash)
	} else {
		const query = `
			INSERT INTO local_users (email, password_hash)
			VALUES ($1::TEXT, $2::TEXT)
			RETURNING ` + localUserColumns
		row = r.q.QueryRow(ctx, query, email, params.PasswordHash)
	}

	u, err := scanLocalUser(row)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, database.ErrConflict
		}
		return nil, fmt.Errorf("create local user: %w", err)
	}
	return u, nil
}

// GetLocalUserByEmail fetches a local_users row by case-insensitive email.
func (r *LocalAuthRepo) GetLocalUserByEmail(ctx context.Context, email string) (*LocalUser, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		SELECT ` + localUserColumns + `
		FROM local_users
		WHERE LOWER(email) = LOWER($1::TEXT)
	`
	u, err := scanLocalUser(r.q.QueryRow(ctx, query, strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("get local user by email: %w", err)
	}
	return u, nil
}

// GetLocalUserByID fetches a local_users row by UUID.
func (r *LocalAuthRepo) GetLocalUserByID(ctx context.Context, userID string) (*LocalUser, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		SELECT ` + localUserColumns + `
		FROM local_users
		WHERE id = $1::UUID
	`
	u, err := scanLocalUser(r.q.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("get local user by id: %w", err)
	}
	return u, nil
}

// RecordFailedLogin atomically increments failed_login_attempts and sets
// locked_until when the updated count reaches or exceeds maxAttempts.
func (r *LocalAuthRepo) RecordFailedLogin(ctx context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*LocalUser, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		UPDATE local_users
		SET
			failed_login_attempts = LEAST(failed_login_attempts + 1, 32767),
			locked_until = CASE
				WHEN LEAST(failed_login_attempts + 1, 32767) >= $2::SMALLINT THEN NOW() + ($3::BIGINT * INTERVAL '1 millisecond')
				ELSE locked_until
			END,
			updated_at = NOW()
		WHERE id = $1::UUID
		RETURNING ` + localUserColumns

	u, err := scanLocalUser(r.q.QueryRow(ctx, query, userID, maxAttempts, lockDuration.Milliseconds()))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("record failed login: %w", err)
	}
	return u, nil
}

// ResetLoginLockout clears failed_login_attempts and locked_until after a
// successful authentication.
func (r *LocalAuthRepo) ResetLoginLockout(ctx context.Context, userID string) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		UPDATE local_users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1::UUID
	`
	tag, err := r.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("reset login lockout: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// MarkEmailVerified marks a local user's email address as verified.
func (r *LocalAuthRepo) MarkEmailVerified(ctx context.Context, userID string) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		UPDATE local_users
		SET email_verified = TRUE, updated_at = NOW()
		WHERE id = $1::UUID
	`
	tag, err := r.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// UpdatePasswordHash updates the stored bcrypt password hash for a local user
// and optionally resets any active lockout counters.
func (r *LocalAuthRepo) UpdatePasswordHash(ctx context.Context, userID, passwordHash string, resetLockout bool) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	const queryUpdateWithReset = `
		UPDATE local_users
		SET password_hash = $2::TEXT, failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1::UUID
	`
	const queryUpdateHashOnly = `
		UPDATE local_users
		SET password_hash = $2::TEXT, updated_at = NOW()
		WHERE id = $1::UUID
	`

	query := queryUpdateHashOnly
	if resetLockout {
		query = queryUpdateWithReset
	}

	tag, err := r.q.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// DeleteLocalUser deletes a local_users row (cascading local_auth_tokens).
func (r *LocalAuthRepo) DeleteLocalUser(ctx context.Context, userID string) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	const query = `DELETE FROM local_users WHERE id = $1::UUID`
	tag, err := r.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete local user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

const authRecordColumns = `id, user_id, token_hash, purpose, expires_at, used_at, created_at`

func scanLocalAuthToken(row pgx.Row) (*LocalAuthToken, error) {
	var t LocalAuthToken
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.TokenHash,
		&t.Purpose,
		&t.ExpiresAt,
		&t.UsedAt,
		&t.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateAuthToken inserts a new single-use hashed token into local_auth_tokens.
func (r *LocalAuthRepo) CreateAuthToken(ctx context.Context, params CreateAuthTokenParams) (*LocalAuthToken, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		INSERT INTO local_auth_tokens (user_id, token_hash, purpose, expires_at)
		VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::TIMESTAMPTZ)
		RETURNING ` + authRecordColumns

	t, err := scanLocalAuthToken(r.q.QueryRow(ctx, query, params.UserID, params.TokenHash, params.Purpose, params.ExpiresAt))
	if err != nil {
		if database.IsUniqueViolation(err) {
			return nil, database.ErrConflict
		}
		return nil, fmt.Errorf("create auth token: %w", err)
	}
	return t, nil
}

// InvalidateUserTokens marks all unused tokens for the given user and purpose
// as used so only the most recently issued token remains valid.
func (r *LocalAuthRepo) InvalidateUserTokens(ctx context.Context, userID, purpose string) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		UPDATE local_auth_tokens
		SET used_at = NOW()
		WHERE user_id = $1::UUID
		  AND purpose = $2::TEXT
		  AND used_at IS NULL
	`
	if _, err := r.q.Exec(ctx, query, userID, purpose); err != nil {
		return fmt.Errorf("invalidate user tokens: %w", err)
	}
	return nil
}

// ConsumeAuthToken atomically marks a matching unused, unexpired token as used
// and returns it. Returns database.ErrTokenInvalidOrExpired if no matching row exists.
func (r *LocalAuthRepo) ConsumeAuthToken(ctx context.Context, tokenHash, purpose string) (*LocalAuthToken, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		UPDATE local_auth_tokens
		SET used_at = NOW()
		WHERE token_hash = $1::TEXT
		  AND purpose = $2::TEXT
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING ` + authRecordColumns

	t, err := scanLocalAuthToken(r.q.QueryRow(ctx, query, tokenHash, purpose))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrTokenInvalidOrExpired
		}
		return nil, fmt.Errorf("consume auth token: %w", err)
	}
	return t, nil
}

// DeleteExpiredTokens deletes all expired rows from local_auth_tokens and
// returns the number of removed rows.
func (r *LocalAuthRepo) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	if r == nil || r.q == nil {
		return 0, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `DELETE FROM local_auth_tokens WHERE expires_at <= NOW()`
	tag, err := r.q.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}

// VerifyEmailWithToken atomically consumes a single-use verify_email token and
// marks the associated local_users row as email_verified = TRUE.
func (r *LocalAuthRepo) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	if r == nil || r.q == nil {
		return "", fmt.Errorf("local auth repo is not initialized")
	}

	var userID string
	run := func(txRepo *LocalAuthRepo) error {
		tok, err := txRepo.ConsumeAuthToken(ctx, tokenHash, TokenPurposeVerifyEmail)
		if err != nil {
			return err
		}
		if err := txRepo.MarkEmailVerified(ctx, tok.UserID); err != nil {
			return err
		}
		userID = tok.UserID
		return nil
	}

	if r.pool != nil {
		if err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
			return run(NewLocalAuthRepoWithDBTX(tx))
		}); err != nil {
			return "", err
		}
		return userID, nil
	}

	if err := run(r); err != nil {
		return "", err
	}
	return userID, nil
}

// ResetPasswordWithToken atomically consumes a single-use reset_password token,
// updates the user's password hash, and clears any active login lockout.
func (r *LocalAuthRepo) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	if r == nil || r.q == nil {
		return "", fmt.Errorf("local auth repo is not initialized")
	}

	var userID string
	run := func(txRepo *LocalAuthRepo) error {
		tok, err := txRepo.ConsumeAuthToken(ctx, tokenHash, TokenPurposeResetPassword)
		if err != nil {
			return err
		}
		if err := txRepo.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
			return err
		}
		userID = tok.UserID
		return nil
	}

	if r.pool != nil {
		if err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
			return run(NewLocalAuthRepoWithDBTX(tx))
		}); err != nil {
			return "", err
		}
		return userID, nil
	}

	if err := run(r); err != nil {
		return "", err
	}
	return userID, nil
}

// DeleteLocalAccount atomically deletes a user's profile (cascading saved_meals)
// and local_users record (cascading local_auth_tokens) in a single transaction.
func (r *LocalAuthRepo) DeleteLocalAccount(ctx context.Context, userID string) error {
	if r == nil || r.q == nil {
		return fmt.Errorf("local auth repo is not initialized")
	}

	run := func(q database.DBTX) error {
		profStore := profile.NewStoreWithDBTX(q)
		if err := profStore.DeleteProfile(ctx, userID); err != nil && !errors.Is(err, database.ErrNotFound) {
			return err
		}
		txAuth := NewLocalAuthRepoWithDBTX(q)
		if err := txAuth.DeleteLocalUser(ctx, userID); err != nil {
			return err
		}
		return nil
	}

	if r.pool != nil {
		return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
			return run(tx)
		})
	}
	return run(r.q)
}
