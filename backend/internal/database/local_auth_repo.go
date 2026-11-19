package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

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
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// CreateLocalUser inserts a new local_users row with normalized lowercase email.
func (s *Store) CreateLocalUser(ctx context.Context, params CreateLocalUserParams) (*LocalUser, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))
	var row pgx.Row
	if strings.TrimSpace(params.ID) != "" {
		const query = `
			INSERT INTO local_users (id, email, password_hash)
			VALUES ($1::UUID, $2::TEXT, $3::TEXT)
			RETURNING ` + localUserColumns
		row = s.q.QueryRow(ctx, query, params.ID, email, params.PasswordHash)
	} else {
		const query = `
			INSERT INTO local_users (email, password_hash)
			VALUES ($1::TEXT, $2::TEXT)
			RETURNING ` + localUserColumns
		row = s.q.QueryRow(ctx, query, email, params.PasswordHash)
	}

	u, err := scanLocalUser(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create local user: %w", err)
	}
	return u, nil
}

// GetLocalUserByEmail fetches a local_users row by case-insensitive email.
func (s *Store) GetLocalUserByEmail(ctx context.Context, email string) (*LocalUser, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	const query = `
		SELECT ` + localUserColumns + `
		FROM local_users
		WHERE LOWER(email) = LOWER($1::TEXT)
	`
	u, err := scanLocalUser(s.q.QueryRow(ctx, query, strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get local user by email: %w", err)
	}
	return u, nil
}

// GetLocalUserByID fetches a local_users row by UUID.
func (s *Store) GetLocalUserByID(ctx context.Context, userID string) (*LocalUser, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	const query = `
		SELECT ` + localUserColumns + `
		FROM local_users
		WHERE id = $1::UUID
	`
	u, err := scanLocalUser(s.q.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get local user by id: %w", err)
	}
	return u, nil
}

// RecordFailedLogin atomically increments failed_login_attempts and sets
// locked_until when the updated count reaches or exceeds maxAttempts.
func (s *Store) RecordFailedLogin(ctx context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*LocalUser, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
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

	u, err := scanLocalUser(s.q.QueryRow(ctx, query, userID, maxAttempts, lockDuration.Milliseconds()))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("record failed login: %w", err)
	}
	return u, nil
}

// ResetLoginLockout clears failed_login_attempts and locked_until after a
// successful authentication.
func (s *Store) ResetLoginLockout(ctx context.Context, userID string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("database store is not initialized")
	}

	const query = `
		UPDATE local_users
		SET failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
		WHERE id = $1::UUID
	`
	tag, err := s.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("reset login lockout: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkEmailVerified marks a local user's email address as verified.
func (s *Store) MarkEmailVerified(ctx context.Context, userID string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("database store is not initialized")
	}

	const query = `
		UPDATE local_users
		SET email_verified = TRUE, updated_at = NOW()
		WHERE id = $1::UUID
	`
	tag, err := s.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("mark email verified: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdatePasswordHash updates the stored bcrypt password hash for a local user
// and optionally resets any active lockout counters.
func (s *Store) UpdatePasswordHash(ctx context.Context, userID, passwordHash string, resetLockout bool) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("database store is not initialized")
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

	tag, err := s.q.Exec(ctx, query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteLocalUser deletes a local_users row (cascading local_auth_tokens).
func (s *Store) DeleteLocalUser(ctx context.Context, userID string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("database store is not initialized")
	}

	const query = `DELETE FROM local_users WHERE id = $1::UUID`
	tag, err := s.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete local user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
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
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

// CreateAuthToken inserts a new single-use hashed token into local_auth_tokens.
func (s *Store) CreateAuthToken(ctx context.Context, params CreateAuthTokenParams) (*LocalAuthToken, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	const query = `
		INSERT INTO local_auth_tokens (user_id, token_hash, purpose, expires_at)
		VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::TIMESTAMPTZ)
		RETURNING ` + authRecordColumns

	t, err := scanLocalAuthToken(s.q.QueryRow(ctx, query, params.UserID, params.TokenHash, params.Purpose, params.ExpiresAt))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, fmt.Errorf("create auth token: %w", err)
	}
	return t, nil
}

// InvalidateUserTokens marks all unused tokens for the given user and purpose
// as used so only the most recently issued token remains valid.
func (s *Store) InvalidateUserTokens(ctx context.Context, userID, purpose string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("database store is not initialized")
	}

	const query = `
		UPDATE local_auth_tokens
		SET used_at = NOW()
		WHERE user_id = $1::UUID
		  AND purpose = $2::TEXT
		  AND used_at IS NULL
	`
	if _, err := s.q.Exec(ctx, query, userID, purpose); err != nil {
		return fmt.Errorf("invalidate user tokens: %w", err)
	}
	return nil
}

// ConsumeAuthToken atomically marks a matching unused, unexpired token as used
// and returns it. Returns ErrTokenInvalidOrExpired if no matching row exists.
func (s *Store) ConsumeAuthToken(ctx context.Context, tokenHash, purpose string) (*LocalAuthToken, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("database store is not initialized")
	}

	const query = `
		UPDATE local_auth_tokens
		SET used_at = NOW()
		WHERE token_hash = $1::TEXT
		  AND purpose = $2::TEXT
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING ` + authRecordColumns

	t, err := scanLocalAuthToken(s.q.QueryRow(ctx, query, tokenHash, purpose))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrTokenInvalidOrExpired
		}
		return nil, fmt.Errorf("consume auth token: %w", err)
	}
	return t, nil
}

// DeleteExpiredTokens deletes all expired rows from local_auth_tokens and
// returns the number of removed rows.
func (s *Store) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	if s == nil || s.q == nil {
		return 0, fmt.Errorf("database store is not initialized")
	}

	const query = `DELETE FROM local_auth_tokens WHERE expires_at <= NOW()`
	tag, err := s.q.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete expired tokens: %w", err)
	}
	return tag.RowsAffected(), nil
}
