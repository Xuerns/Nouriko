package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/profile"

	"github.com/google/uuid"
)

const (
	maxFailedLoginAttempts int16 = 5
	loginLockoutDuration         = 15 * time.Minute
	verifyEmailTokenTTL          = 24 * time.Hour
	resetPasswordTokenTTL        = 30 * time.Minute
	localAccessTokenTTL          = 24 * time.Hour
)

type keyedLockEntry struct {
	mu   sync.Mutex
	refs int
}

type keyedLocker struct {
	mu      sync.Mutex
	entries map[string]*keyedLockEntry
}

// lock acquires an exclusive per-key mutex and returns an unlock function that
// releases the lock and deletes the map entry once no waiters remain.
func (k *keyedLocker) lock(key string) func() {
	k.mu.Lock()
	if k.entries == nil {
		k.entries = make(map[string]*keyedLockEntry)
	}
	entry, ok := k.entries[key]
	if !ok {
		entry = &keyedLockEntry{}
		k.entries[key] = entry
	}
	entry.refs++
	k.mu.Unlock()

	entry.mu.Lock()

	return func() {
		entry.mu.Unlock()
		k.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(k.entries, key)
		}
		k.mu.Unlock()
	}
}

// LocalService implements Service for APP_ENV=local using PostgreSQL tables
// (local_users, local_auth_tokens, profiles) and HS256 JWTs.
type LocalService struct {
	repo       LocalAuthRepository
	profiles   profile.Repository
	jwtSecret  string
	audience   string
	appBaseURL string
	logger     *slog.Logger
	now        func() time.Time
	loginLocks keyedLocker
}

// Compile-time interface check.
var _ Service = (*LocalService)(nil)

// NewLocalService constructs a LocalService backed by localRepo and profileRepo.
func NewLocalService(cfg *config.Config, localRepo LocalAuthRepository, profileRepo profile.Repository, logger *slog.Logger) *LocalService {
	if logger == nil {
		logger = slog.Default()
	}
	var jwtSecret, audience, appBaseURL string
	if cfg != nil {
		jwtSecret = cfg.Auth.JWTSecret
		audience = cfg.Auth.Audience
		appBaseURL = cfg.AppBaseURL
	}
	if audience == "" {
		audience = "authenticated"
	}
	if appBaseURL == "" {
		appBaseURL = "http://localhost:3000"
	}
	return &LocalService{
		repo:       localRepo,
		profiles:   profileRepo,
		jwtSecret:  jwtSecret,
		audience:   audience,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
		logger:     logger,
		now:        time.Now,
	}
}

// Register creates a local user, generates a DiceBear avatar profile, and issues
// a single-use email verification token.
func (s *LocalService) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
	email, err := NormalizeAndValidateEmail(in.Email)
	if err != nil {
		return nil, ErrInvalidInput
	}
	lang, err := ValidateLanguage(in.Language, true)
	if err != nil {
		return nil, ErrInvalidInput
	}
	theme, err := ValidateTheme(in.Theme, true)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if err := ValidatePassword(in.Password); err != nil {
		return nil, err
	}

	passwordHash, err := HashPassword(in.Password)
	if err != nil {
		return nil, fmt.Errorf("local register hash: %w", err)
	}

	userID := uuid.NewString()
	user, err := s.repo.CreateLocalUser(ctx, CreateLocalUserParams{
		ID:           userID,
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		if errors.Is(err, database.ErrConflict) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("create local user: %w", err)
	}

	if _, err := s.profiles.UpsertProfile(ctx, profile.UpsertProfileParams{
		ID:         user.ID,
		ProfileURL: DiceBearAvatarURL(user.ID),
		Language:   lang,
		Theme:      theme,
	}); err != nil {
		_ = s.repo.DeleteLocalUser(ctx, user.ID)
		return nil, fmt.Errorf("upsert profile on register: %w", err)
	}

	rawToken, tokenHash, err := GenerateRandomToken()
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.CreateAuthToken(ctx, CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Purpose:   TokenPurposeVerifyEmail,
		ExpiresAt: s.now().Add(verifyEmailTokenTTL),
	}); err != nil {
		return nil, fmt.Errorf("create verify_email token: %w", err)
	}

	s.logger.Info("local email verification link generated",
		"user_id", user.ID,
		"verify_url", fmt.Sprintf("%s/verify-email?token=%s", s.appBaseURL, rawToken),
	)

	return &RegisterResult{
		Status:               "verification_required",
		UserID:               user.ID,
		Email:                user.Email,
		DevVerificationToken: rawToken,
	}, nil
}

// VerifyEmail atomically consumes a single-use verify_email token and marks the
// user's email as verified.
func (s *LocalService) VerifyEmail(ctx context.Context, rawToken string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return ErrInvalidInput
	}

	tokenHash := HashToken(token)
	if _, err := s.repo.VerifyEmailWithToken(ctx, tokenHash); err != nil {
		if errors.Is(err, database.ErrTokenInvalidOrExpired) || errors.Is(err, database.ErrNotFound) {
			return ErrInvalidOrExpiredToken
		}
		return fmt.Errorf("verify email with token: %w", err)
	}

	return nil
}

// Login authenticates a local user with timing-attack mitigation, brute-force
// lockout (5 failures -> 15 min lock), and email verification enforcement.
func (s *LocalService) Login(ctx context.Context, rawEmail, password string) (*LoginResult, error) {
	email, err := NormalizeAndValidateEmail(rawEmail)
	if err != nil || password == "" {
		return nil, ErrInvalidInput
	}

	unlock := s.loginLocks.lock(email)
	defer unlock()

	user, err := s.repo.GetLocalUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			DummyPasswordCheck(password)
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("lookup local user: %w", err)
	}

	if user.LockedUntil != nil && user.LockedUntil.After(s.now()) {
		return nil, ErrTooManyRequests
	}

	if !VerifyPassword(user.PasswordHash, password) {
		if _, recErr := s.repo.RecordFailedLogin(ctx, user.ID, maxFailedLoginAttempts, loginLockoutDuration); recErr != nil {
			s.logger.Warn("failed to record failed login attempt", "user_id", user.ID, "error", recErr)
		}
		return nil, ErrInvalidCredentials
	}

	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		if err := s.repo.ResetLoginLockout(ctx, user.ID); err != nil {
			s.logger.Warn("failed to reset login lockout", "user_id", user.ID, "error", err)
		}
	}

	accessToken, err := SignLocalToken(s.jwtSecret, user.ID, user.Email, s.audience, localAccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("sign local token: %w", err)
	}

	prof, err := s.profiles.GetProfileByID(ctx, user.ID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			prof, err = s.profiles.UpsertProfile(ctx, profile.UpsertProfileParams{
				ID:         user.ID,
				ProfileURL: DiceBearAvatarURL(user.ID),
				Language:   "id",
				Theme:      "light",
			})
		}
		if err != nil {
			return nil, fmt.Errorf("fetch profile on login: %w", err)
		}
	}

	return &LoginResult{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int(localAccessTokenTTL.Seconds()),
		UserID:      user.ID,
		Email:       user.Email,
		Profile:     prof,
	}, nil
}

// Logout acknowledges logout in local stateless HS256 mode.
func (s *LocalService) Logout(_ context.Context, _ string) error {
	return nil
}

// ForgotPassword issues a single-use short-lived reset token if the email exists,
// without revealing account existence to the caller.
func (s *LocalService) ForgotPassword(ctx context.Context, rawEmail string) error {
	email, err := NormalizeAndValidateEmail(rawEmail)
	if err != nil {
		return ErrInvalidInput
	}

	user, err := s.repo.GetLocalUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("lookup user for forgot password: %w", err)
	}

	if err := s.repo.InvalidateUserTokens(ctx, user.ID, TokenPurposeResetPassword); err != nil {
		return fmt.Errorf("invalidate old reset tokens: %w", err)
	}

	rawToken, tokenHash, err := GenerateRandomToken()
	if err != nil {
		return err
	}

	if _, err := s.repo.CreateAuthToken(ctx, CreateAuthTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		Purpose:   TokenPurposeResetPassword,
		ExpiresAt: s.now().Add(resetPasswordTokenTTL),
	}); err != nil {
		return fmt.Errorf("create reset_password token: %w", err)
	}

	s.logger.Info("local password reset link generated",
		"user_id", user.ID,
		"reset_url", fmt.Sprintf("%s/reset-password?token=%s", s.appBaseURL, rawToken),
	)

	return nil
}

// ResetPassword validates the new password and atomically consumes a single-use
// reset_password token to update the user's password hash and clear lockouts.
func (s *LocalService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return ErrInvalidInput
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash reset password: %w", err)
	}

	tokenHash := HashToken(token)
	if _, err := s.repo.ResetPasswordWithToken(ctx, tokenHash, newHash); err != nil {
		if errors.Is(err, database.ErrTokenInvalidOrExpired) || errors.Is(err, database.ErrNotFound) {
			return ErrInvalidOrExpiredToken
		}
		return fmt.Errorf("reset password with token: %w", err)
	}

	return nil
}

// ChangePassword verifies the current password and updates the local user's password hash.
func (s *LocalService) ChangePassword(ctx context.Context, userID, _, _, currentPassword, newPassword string) error {
	if currentPassword == "" {
		return ErrInvalidInput
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.repo.GetLocalUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("get local user for change password: %w", err)
	}

	if !VerifyPassword(user.PasswordHash, currentPassword) {
		return ErrInvalidCredentials
	}

	newHash, err := HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	if err := s.repo.UpdatePasswordHash(ctx, userID, newHash, false); err != nil {
		return fmt.Errorf("update local password hash: %w", err)
	}

	return nil
}

// DeleteAccount verifies the user's password and permanently deletes their
// profile (cascading saved_meals) and local auth record in a transaction (PRD F13).
func (s *LocalService) DeleteAccount(ctx context.Context, userID, _, password string) error {
	if password == "" {
		return ErrInvalidInput
	}

	user, err := s.repo.GetLocalUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("get local user for delete account: %w", err)
	}

	if !VerifyPassword(user.PasswordHash, password) {
		return ErrInvalidCredentials
	}

	if err := s.repo.DeleteLocalAccount(ctx, userID); err != nil {
		return fmt.Errorf("delete local account: %w", err)
	}

	return nil
}
