package auth

import (
	"context"
	"log/slog"

	"Nouriko/internal/config"
	"Nouriko/internal/profile"
)

// RegisterInput holds user registration inputs.
type RegisterInput struct {
	Email    string
	Password string
	Language string
	Theme    string
}

// RegisterResult holds the registration response payload.
type RegisterResult struct {
	Status               string `json:"status"`
	UserID               string `json:"user_id"`
	Email                string `json:"email"`
	DevVerificationToken string `json:"dev_verification_token,omitempty"`
}

// GoogleLoginInput holds verified Google Identity Services login inputs.
type GoogleLoginInput struct {
	IDToken  string
	Nonce    string
	Language string
	Theme    string
}

// DeleteAccountInput holds re-authentication credentials for account deletion (PRD F13).
type DeleteAccountInput struct {
	UserID        string
	Email         string
	Password      string
	GoogleIDToken string
	Nonce         string
	HasPassword   bool
}

// LoginResult holds the authenticated session token and user profile.
type LoginResult struct {
	AccessToken   string
	TokenType     string
	ExpiresIn     int
	UserID        string
	Email         string
	AuthProviders []string
	HasPassword   bool
	Profile       *profile.Profile
}

// Service defines the unified dual-mode authentication & account security interface.
type Service interface {
	Register(ctx context.Context, in RegisterInput) (*RegisterResult, error)
	VerifyEmail(ctx context.Context, rawToken string) error
	Login(ctx context.Context, email, password string) (*LoginResult, error)
	LoginWithGoogle(ctx context.Context, in GoogleLoginInput) (*LoginResult, error)
	Logout(ctx context.Context, accessToken string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
	ChangePassword(ctx context.Context, userID, email, accessToken, currentPassword, newPassword string) error
	DeleteAccount(ctx context.Context, in DeleteAccountInput) error
}

// NewService constructs either LocalService (APP_ENV=local) or SupabaseService
// (APP_ENV=production) behind the unified Service interface.
func NewService(
	cfg *config.Config,
	localRepo LocalAuthRepository,
	profileRepo profile.Repository,
	admin *AdminClient,
	googleVerifier GoogleTokenVerifier,
	logger *slog.Logger,
) Service {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg != nil && cfg.IsProduction() {
		svc := NewSupabaseService(cfg, profileRepo, admin, googleVerifier, logger, nil)
		if localRepo != nil {
			svc.SetOAuthReplayStore(localRepo)
		}
		return svc
	}
	return NewLocalService(cfg, localRepo, profileRepo, googleVerifier, logger)
}
