package auth

import "time"

// Supported local_auth_tokens purpose values and OAuth providers.
const (
	TokenPurposeVerifyEmail   = "verify_email"
	TokenPurposeResetPassword = "reset_password"
	OAuthProviderGoogle       = "google"
)

// LocalUser represents a row in the local_users table (used when APP_ENV=local).
type LocalUser struct {
	ID                  string
	Email               string
	PasswordHash        *string
	EmailVerified       bool
	FailedLoginAttempts int16
	LockedUntil         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// HasPassword reports whether the local user has a non-empty bcrypt password hash.
func (u *LocalUser) HasPassword() bool {
	return u != nil && u.PasswordHash != nil && *u.PasswordHash != ""
}

// LocalUserIdentity represents a linked external OAuth identity in local_user_identities.
type LocalUserIdentity struct {
	ID             string
	UserID         string
	Provider       string
	ProviderUserID string
	Email          string
	CreatedAt      time.Time
}

// UpsertGoogleUserParams defines verified Google identity fields used to find,
// link, or create a local user account.
type UpsertGoogleUserParams struct {
	Email          string
	ProviderUserID string
}

// LocalAuthToken represents a row in the local_auth_tokens table.
type LocalAuthToken struct {
	ID        string
	UserID    string
	TokenHash string
	Purpose   string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// CreateLocalUserParams defines inputs for inserting a new local user.
type CreateLocalUserParams struct {
	// ID is optional; when empty, PostgreSQL generates a UUID via gen_random_uuid().
	ID           string
	Email        string
	PasswordHash string
}

// CreateAuthTokenParams defines inputs for creating a single-use auth token.
type CreateAuthTokenParams struct {
	UserID    string
	TokenHash string
	Purpose   string
	ExpiresAt time.Time
}
