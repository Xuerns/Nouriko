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

	FindOrCreateGoogleUser(ctx context.Context, params UpsertGoogleUserParams) (*LocalUser, []string, error)
	GetUserProviders(ctx context.Context, userID string) ([]string, error)
	GetUserIdentity(ctx context.Context, userID, provider string) (*LocalUserIdentity, error)
	ConsumeOAuthNonce(ctx context.Context, nonceHash string, expiresAt time.Time) (bool, error)
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
	var passwordHash *string
	if params.PasswordHash != "" {
		pw := params.PasswordHash
		passwordHash = &pw
	}
	var row pgx.Row
	if strings.TrimSpace(params.ID) != "" {
		const query = `
			INSERT INTO local_users (id, email, password_hash)
			VALUES ($1::UUID, $2::TEXT, $3::TEXT)
			RETURNING ` + localUserColumns
		row = r.q.QueryRow(ctx, query, params.ID, email, passwordHash)
	} else {
		const query = `
			INSERT INTO local_users (email, password_hash)
			VALUES ($1::TEXT, $2::TEXT)
			RETURNING ` + localUserColumns
		row = r.q.QueryRow(ctx, query, email, passwordHash)
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

const localIdentityColumns = `id, user_id, provider, provider_user_id, email, created_at`

func scanLocalUserIdentity(row pgx.Row) (*LocalUserIdentity, error) {
	var id LocalUserIdentity
	err := row.Scan(
		&id.ID,
		&id.UserID,
		&id.Provider,
		&id.ProviderUserID,
		&id.Email,
		&id.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}
	return &id, nil
}

// GetUserIdentity fetches a linked OAuth identity for the given user and provider.
func (r *LocalAuthRepo) GetUserIdentity(ctx context.Context, userID, provider string) (*LocalUserIdentity, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	const query = `
		SELECT ` + localIdentityColumns + `
		FROM local_user_identities
		WHERE user_id = $1::UUID
		  AND provider = $2::TEXT
	`
	id, err := scanLocalUserIdentity(r.q.QueryRow(ctx, query, userID, strings.ToLower(strings.TrimSpace(provider))))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("get user identity: %w", err)
	}
	return id, nil
}

func (r *LocalAuthRepo) getIdentityByProviderSub(ctx context.Context, provider, providerUserID string) (*LocalUserIdentity, error) {
	const query = `
		SELECT ` + localIdentityColumns + `
		FROM local_user_identities
		WHERE provider = $1::TEXT
		  AND provider_user_id = $2::TEXT
	`
	id, err := scanLocalUserIdentity(r.q.QueryRow(ctx, query, provider, providerUserID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("get identity by provider sub: %w", err)
	}
	return id, nil
}

// GetUserProviders returns the active authentication providers for a local user:
// "email" when password_hash is present plus any linked OAuth providers.
func (r *LocalAuthRepo) GetUserProviders(ctx context.Context, userID string) ([]string, error) {
	if r == nil || r.q == nil {
		return nil, fmt.Errorf("local auth repo is not initialized")
	}

	user, err := r.GetLocalUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	providers := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	if user.HasPassword() {
		providers = append(providers, "email")
		seen["email"] = struct{}{}
	}

	const query = `
		SELECT provider
		FROM local_user_identities
		WHERE user_id = $1::UUID
		ORDER BY created_at ASC, provider ASC
	`
	rows, err := r.q.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user identity providers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan user identity provider: %w", err)
		}
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		providers = append(providers, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user identity providers: %w", err)
	}

	return providers, nil
}

// FindOrCreateGoogleUser atomically resolves a verified Google identity into a
// local_users row and its active providers, applying pre-account takeover
// mitigation if an unverified local password account existed for the same email.
func (r *LocalAuthRepo) FindOrCreateGoogleUser(ctx context.Context, params UpsertGoogleUserParams) (*LocalUser, []string, error) {
	if r == nil || r.q == nil {
		return nil, nil, fmt.Errorf("local auth repo is not initialized")
	}

	email := strings.ToLower(strings.TrimSpace(params.Email))
	providerUserID := strings.TrimSpace(params.ProviderUserID)
	if email == "" || providerUserID == "" {
		return nil, nil, ErrInvalidInput
	}

	var (
		user      *LocalUser
		providers []string
	)

	run := func(txRepo *LocalAuthRepo) error {
		// 1. Existing Google identity lookup by (provider, provider_user_id).
		existingIdentity, err := txRepo.getIdentityByProviderSub(ctx, OAuthProviderGoogle, providerUserID)
		if err == nil {
			u, uErr := txRepo.GetLocalUserByID(ctx, existingIdentity.UserID)
			if uErr != nil {
				return uErr
			}
			if u.FailedLoginAttempts > 0 || u.LockedUntil != nil {
				if err := txRepo.ResetLoginLockout(ctx, u.ID); err != nil {
					return err
				}
				u.FailedLoginAttempts = 0
				u.LockedUntil = nil
			}
			provs, pErr := txRepo.GetUserProviders(ctx, u.ID)
			if pErr != nil {
				return pErr
			}
			user = u
			providers = provs
			return nil
		}
		if !errors.Is(err, database.ErrNotFound) {
			return err
		}

		// 2. Email lookup for linking or pre-takeover mitigation (locked FOR UPDATE inside tx).
		const lookupUserForUpdateQuery = `
			SELECT ` + localUserColumns + `
			FROM local_users
			WHERE LOWER(email) = LOWER($1::TEXT)
			FOR UPDATE
		`
		u, err := scanLocalUser(txRepo.q.QueryRow(ctx, lookupUserForUpdateQuery, email))
		switch {
		case err == nil:
			if !u.EmailVerified {
				// Pre-account takeover mitigation: clear unverified pre-registered
				// password and invalidate all unused verification/reset tokens.
				const clearUnverifiedQuery = `
					UPDATE local_users
					SET password_hash = NULL, email_verified = TRUE, failed_login_attempts = 0, locked_until = NULL, updated_at = NOW()
					WHERE id = $1::UUID
					RETURNING ` + localUserColumns
				u, err = scanLocalUser(txRepo.q.QueryRow(ctx, clearUnverifiedQuery, u.ID))
				if err != nil {
					return fmt.Errorf("clear unverified local password on google link: %w", err)
				}
				const clearPendingAuthQuery = `
					UPDATE local_auth_tokens
					SET used_at = NOW()
					WHERE user_id = $1::UUID
					  AND used_at IS NULL
				`
				if _, err := txRepo.q.Exec(ctx, clearPendingAuthQuery, u.ID); err != nil {
					return fmt.Errorf("invalidate unverified user tokens on google link: %w", err)
				}
			} else if u.FailedLoginAttempts > 0 || u.LockedUntil != nil {
				if err := txRepo.ResetLoginLockout(ctx, u.ID); err != nil {
					return err
				}
				u.FailedLoginAttempts = 0
				u.LockedUntil = nil
			}

			const insertIdentityQuery = `
				INSERT INTO local_user_identities (user_id, provider, provider_user_id, email)
				VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::TEXT)
				ON CONFLICT (provider, provider_user_id) DO NOTHING
			`
			tag, err := txRepo.q.Exec(ctx, insertIdentityQuery, u.ID, OAuthProviderGoogle, providerUserID, email)
			if err != nil {
				if database.IsUniqueViolation(err) {
					return database.ErrConflict
				}
				return fmt.Errorf("insert linked google identity: %w", err)
			}
			if tag.RowsAffected() == 0 {
				// Verify whether another concurrent tx bound this provider subject to a different user.
				existing, lookupErr := txRepo.getIdentityByProviderSub(ctx, OAuthProviderGoogle, providerUserID)
				if lookupErr != nil || existing.UserID != u.ID {
					return database.ErrConflict
				}
			}

		case errors.Is(err, database.ErrNotFound):
			// 3. Create brand-new passwordless verified local_users row + identity.
			const insertUserQuery = `
				INSERT INTO local_users (email, password_hash, email_verified)
				VALUES ($1::TEXT, NULL, TRUE)
				RETURNING ` + localUserColumns
			u, err = scanLocalUser(txRepo.q.QueryRow(ctx, insertUserQuery, email))
			if err != nil {
				if database.IsUniqueViolation(err) {
					return database.ErrConflict
				}
				return fmt.Errorf("create google local user: %w", err)
			}
			const insertIdentityQuery = `
				INSERT INTO local_user_identities (user_id, provider, provider_user_id, email)
				VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::TEXT)
			`
			if _, err := txRepo.q.Exec(ctx, insertIdentityQuery, u.ID, OAuthProviderGoogle, providerUserID, email); err != nil {
				if database.IsUniqueViolation(err) {
					return database.ErrConflict
				}
				return fmt.Errorf("insert new google identity: %w", err)
			}

		default:
			return err
		}

		provs, pErr := txRepo.GetUserProviders(ctx, u.ID)
		if pErr != nil {
			return pErr
		}
		user = u
		providers = provs
		return nil
	}

	if r.pool != nil {
		if err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
			return run(NewLocalAuthRepoWithDBTX(tx))
		}); err != nil {
			return nil, nil, err
		}
		return user, providers, nil
	}

	if err := run(r); err != nil {
		return nil, nil, err
	}
	return user, providers, nil
}

// ConsumeOAuthNonce atomically records a single-use OAuth nonce hash in
// oauth_consumed_nonces, returning true only on the first consumption.
func (r *LocalAuthRepo) ConsumeOAuthNonce(ctx context.Context, nonceHash string, expiresAt time.Time) (bool, error) {
	if r == nil || r.q == nil {
		return false, fmt.Errorf("local auth repo is not initialized")
	}

	const pruneExpiredQuery = `DELETE FROM oauth_consumed_nonces WHERE expires_at <= NOW()`
	_, _ = r.q.Exec(ctx, pruneExpiredQuery)

	const consumeQuery = `
		INSERT INTO oauth_consumed_nonces (nonce_hash, expires_at)
		VALUES ($1::TEXT, $2::TIMESTAMPTZ)
		ON CONFLICT (nonce_hash) DO NOTHING
	`
	tag, err := r.q.Exec(ctx, consumeQuery, nonceHash, expiresAt)
	if err != nil {
		return false, fmt.Errorf("consume oauth nonce: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
