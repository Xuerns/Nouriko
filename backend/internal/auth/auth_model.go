package auth

import "time"

// Supported local_auth_tokens purpose values.
const (
	TokenPurposeVerifyEmail   = "verify_email"
	TokenPurposeResetPassword = "reset_password"
)

// LocalUser represents a row in the local_users table (used when APP_ENV=local).
type LocalUser struct {
	ID                  string
	Email               string
	PasswordHash        string
	EmailVerified       bool
	FailedLoginAttempts int16
	LockedUntil         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
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
