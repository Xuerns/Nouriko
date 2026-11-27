package auth

import "errors"

// Domain errors returned by AuthService implementations and mapped to PRD generic
// error codes by HTTP handlers.
var (
	ErrInvalidInput          = errors.New("invalid input")
	ErrWeakPassword          = errors.New("weak password")
	ErrInvalidOrExpiredToken = errors.New("invalid or expired token")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrEmailNotVerified      = errors.New("email not verified")
	ErrEmailAlreadyExists    = errors.New("email already exists")
	ErrTooManyRequests       = errors.New("too many requests")
	ErrNotFound              = errors.New("not found")
	ErrOAuthUnavailable      = errors.New("oauth unavailable")
)
