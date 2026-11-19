package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/profile"
)

// SupabaseService implements Service for APP_ENV=production using Supabase Auth
// REST endpoints (/auth/v1/*), AdminClient.DeleteUser, and PostgreSQL profiles.
type SupabaseService struct {
	baseURL    string
	apiKey     string
	appBaseURL string
	profiles   profile.Repository
	admin      *AdminClient
	httpClient *http.Client
	logger     *slog.Logger
}

// Compile-time interface check.
var _ Service = (*SupabaseService)(nil)

// NewSupabaseService creates a production Supabase-backed Service.
func NewSupabaseService(cfg *config.Config, profileRepo profile.Repository, admin *AdminClient, logger *slog.Logger, httpClient *http.Client) *SupabaseService {
	if logger == nil {
		logger = slog.Default()
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	var baseURL, apiKey, appBaseURL string
	if cfg != nil {
		baseURL = strings.TrimRight(cfg.Supabase.URL, "/")
		apiKey = cfg.Supabase.AuthAPIKey()
		appBaseURL = strings.TrimRight(cfg.AppBaseURL, "/")
	}
	return &SupabaseService{
		baseURL:    baseURL,
		apiKey:     apiKey,
		appBaseURL: appBaseURL,
		profiles:   profileRepo,
		admin:      admin,
		httpClient: httpClient,
		logger:     logger,
	}
}

type supabaseErrorBody struct {
	Error            string `json:"error"`
	ErrorCode        string `json:"error_code"`
	ErrorDescription string `json:"error_description"`
	Message          string `json:"msg"`
	MsgAlt           string `json:"message"`
}

func (e supabaseErrorBody) combined() string {
	return strings.ToLower(strings.Join([]string{e.Error, e.ErrorCode, e.ErrorDescription, e.Message, e.MsgAlt}, " "))
}

type supabaseUser struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Identities []json.RawMessage `json:"identities"`
}

type supabaseSignupResponse struct {
	ID         string            `json:"id"`
	Email      string            `json:"email"`
	Identities []json.RawMessage `json:"identities"`
	User       *supabaseUser     `json:"user"`
}

type supabaseTokenResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int          `json:"expires_in"`
	User        supabaseUser `json:"user"`
}

// Register creates a user in Supabase Auth and provisions their profiles row with
// a DiceBear avatar URL.
func (s *SupabaseService) Register(ctx context.Context, in RegisterInput) (*RegisterResult, error) {
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

	endpoint := fmt.Sprintf("%s/auth/v1/signup?redirect_to=%s", s.baseURL, url.QueryEscape(s.appBaseURL+"/verify-email"))
	status, rawBody, err := s.doJSON(ctx, http.MethodPost, endpoint, "", map[string]any{
		"email":    email,
		"password": in.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("supabase signup request: %w", err)
	}

	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, mapSupabaseAuthError(status, rawBody, ErrInvalidInput)
	}

	var parsed supabaseSignupResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode supabase signup response: %w", err)
	}

	userID := parsed.ID
	userEmail := parsed.Email
	identities := parsed.Identities
	if parsed.User != nil {
		if parsed.User.ID != "" {
			userID = parsed.User.ID
		}
		if parsed.User.Email != "" {
			userEmail = parsed.User.Email
		}
		if parsed.User.Identities != nil {
			identities = parsed.User.Identities
		}
	}

	// Supabase returns an empty identities array when email confirmation is enabled
	// and the email is already registered.
	if identities != nil && len(identities) == 0 {
		return nil, ErrEmailAlreadyExists
	}
	if userID == "" {
		return nil, errors.New("supabase signup returned empty user id")
	}
	if userEmail == "" {
		userEmail = email
	}

	if _, err := s.profiles.UpsertProfile(ctx, profile.UpsertProfileParams{
		ID:         userID,
		ProfileURL: DiceBearAvatarURL(userID),
		Language:   lang,
		Theme:      theme,
	}); err != nil {
		return nil, fmt.Errorf("upsert profile after supabase signup: %w", err)
	}

	return &RegisterResult{
		Status: "verification_required",
		UserID: userID,
		Email:  userEmail,
	}, nil
}

// VerifyEmail verifies a Supabase email confirmation token hash.
func (s *SupabaseService) VerifyEmail(ctx context.Context, rawToken string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return ErrInvalidInput
	}

	endpoint := fmt.Sprintf("%s/auth/v1/verify", s.baseURL)
	for _, verifyType := range []string{"signup", "email"} {
		status, _, err := s.doJSON(ctx, http.MethodPost, endpoint, "", map[string]any{
			"token_hash": token,
			"type":       verifyType,
		})
		if err != nil {
			return fmt.Errorf("supabase verify request: %w", err)
		}
		if status >= http.StatusOK && status < http.StatusMultipleChoices {
			return nil
		}
		if status == http.StatusTooManyRequests {
			return ErrTooManyRequests
		}
		if status >= http.StatusInternalServerError {
			return fmt.Errorf("supabase verify upstream status %d", status)
		}
	}

	return ErrInvalidOrExpiredToken
}

// Login authenticates a user via Supabase Auth password grant and ensures their
// profile row exists.
func (s *SupabaseService) Login(ctx context.Context, rawEmail, password string) (*LoginResult, error) {
	email, err := NormalizeAndValidateEmail(rawEmail)
	if err != nil || password == "" {
		return nil, ErrInvalidInput
	}

	tokResp, err := s.passwordGrant(ctx, email, password)
	if err != nil {
		return nil, err
	}

	userID := tokResp.User.ID
	userEmail := tokResp.User.Email
	if userEmail == "" {
		userEmail = email
	}

	prof, err := s.profiles.GetProfileByID(ctx, userID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			prof, err = s.profiles.UpsertProfile(ctx, profile.UpsertProfileParams{
				ID:         userID,
				ProfileURL: DiceBearAvatarURL(userID),
				Language:   "id",
				Theme:      "light",
			})
		}
		if err != nil {
			return nil, fmt.Errorf("ensure profile on supabase login: %w", err)
		}
	}

	expiresIn := tokResp.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}

	return &LoginResult{
		AccessToken: tokResp.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
		UserID:      userID,
		Email:       userEmail,
		Profile:     prof,
	}, nil
}

// Logout performs a best-effort Supabase Auth session logout.
func (s *SupabaseService) Logout(ctx context.Context, accessToken string) error {
	if strings.TrimSpace(accessToken) == "" {
		return nil
	}
	endpoint := fmt.Sprintf("%s/auth/v1/logout", s.baseURL)
	status, _, err := s.doJSON(ctx, http.MethodPost, endpoint, accessToken, nil)
	if err != nil {
		s.logger.Warn("supabase logout request failed", "error", err)
		return nil
	}
	if status >= http.StatusInternalServerError {
		s.logger.Warn("supabase logout returned server error", "status", status)
	}
	return nil
}

// ForgotPassword triggers Supabase Auth recovery email without revealing whether
// the email exists.
func (s *SupabaseService) ForgotPassword(ctx context.Context, rawEmail string) error {
	email, err := NormalizeAndValidateEmail(rawEmail)
	if err != nil {
		return ErrInvalidInput
	}

	endpoint := fmt.Sprintf("%s/auth/v1/recover?redirect_to=%s", s.baseURL, url.QueryEscape(s.appBaseURL+"/reset-password"))
	status, _, err := s.doJSON(ctx, http.MethodPost, endpoint, "", map[string]any{
		"email": email,
	})
	if err != nil {
		return fmt.Errorf("supabase recover request: %w", err)
	}
	if status == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}
	if status >= http.StatusInternalServerError {
		return fmt.Errorf("supabase recover upstream status %d", status)
	}
	return nil
}

// ResetPassword exchanges a single-use Supabase recovery token hash via
// /auth/v1/verify (type: "recovery") for a short-lived recovery session, updates
// the password via /auth/v1/user, and immediately revokes the recovery session.
func (s *SupabaseService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return ErrInvalidInput
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	verifyEndpoint := fmt.Sprintf("%s/auth/v1/verify", s.baseURL)
	verifyStatus, verifyBody, err := s.doJSON(ctx, http.MethodPost, verifyEndpoint, "", map[string]any{
		"token_hash": token,
		"type":       "recovery",
	})
	if err != nil {
		return fmt.Errorf("supabase verify recovery request: %w", err)
	}
	if verifyStatus == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}
	if verifyStatus >= http.StatusInternalServerError {
		return fmt.Errorf("supabase verify recovery upstream status %d", verifyStatus)
	}
	if verifyStatus < http.StatusOK || verifyStatus >= http.StatusMultipleChoices {
		return ErrInvalidOrExpiredToken
	}

	var recoveryTok supabaseTokenResponse
	if err := json.Unmarshal(verifyBody, &recoveryTok); err != nil || strings.TrimSpace(recoveryTok.AccessToken) == "" {
		return ErrInvalidOrExpiredToken
	}
	defer func() {
		_ = s.Logout(ctx, recoveryTok.AccessToken)
	}()

	endpoint := fmt.Sprintf("%s/auth/v1/user", s.baseURL)
	status, rawBody, err := s.doJSON(ctx, http.MethodPut, endpoint, recoveryTok.AccessToken, map[string]any{
		"password": newPassword,
	})
	if err != nil {
		return fmt.Errorf("supabase reset password request: %w", err)
	}
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return nil
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return ErrInvalidOrExpiredToken
	}
	return mapSupabaseAuthError(status, rawBody, ErrInvalidOrExpiredToken)
}

// ChangePassword verifies currentPassword via Supabase password grant and then
// updates the password via /auth/v1/user.
func (s *SupabaseService) ChangePassword(ctx context.Context, userID, email, accessToken, currentPassword, newPassword string) error {
	if currentPassword == "" {
		return ErrInvalidInput
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	tokResp, err := s.passwordGrant(ctx, email, currentPassword)
	if err != nil {
		if errors.Is(err, ErrEmailNotVerified) {
			return ErrInvalidCredentials
		}
		return err
	}
	if tokResp.User.ID != userID {
		return ErrInvalidCredentials
	}

	bearer := accessToken
	if tokResp.AccessToken != "" {
		bearer = tokResp.AccessToken
	}

	endpoint := fmt.Sprintf("%s/auth/v1/user", s.baseURL)
	status, rawBody, err := s.doJSON(ctx, http.MethodPut, endpoint, bearer, map[string]any{
		"password": newPassword,
	})
	if err != nil {
		return fmt.Errorf("supabase change password request: %w", err)
	}
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return nil
	}
	return mapSupabaseAuthError(status, rawBody, ErrUnauthorized)
}

// DeleteAccount verifies the user's password via Supabase password grant, deletes
// their profile row (cascading saved_meals), and permanently deletes the user via
// Supabase Admin API (PRD F13).
func (s *SupabaseService) DeleteAccount(ctx context.Context, userID, email, password string) error {
	if password == "" {
		return ErrInvalidInput
	}

	tokResp, err := s.passwordGrant(ctx, email, password)
	if err != nil {
		if errors.Is(err, ErrEmailNotVerified) {
			return ErrInvalidCredentials
		}
		return err
	}
	if tokResp.User.ID != userID {
		return ErrInvalidCredentials
	}

	if err := s.profiles.DeleteProfile(ctx, userID); err != nil && !errors.Is(err, database.ErrNotFound) {
		return fmt.Errorf("delete profile before supabase user deletion: %w", err)
	}

	if err := s.admin.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("supabase admin delete user: %w", err)
	}

	return nil
}

func (s *SupabaseService) passwordGrant(ctx context.Context, email, password string) (*supabaseTokenResponse, error) {
	endpoint := fmt.Sprintf("%s/auth/v1/token?grant_type=password", s.baseURL)
	status, rawBody, err := s.doJSON(ctx, http.MethodPost, endpoint, "", map[string]any{
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("supabase password grant request: %w", err)
	}

	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, mapSupabaseAuthError(status, rawBody, ErrInvalidCredentials)
	}

	var parsed supabaseTokenResponse
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode supabase token response: %w", err)
	}
	if parsed.AccessToken == "" || parsed.User.ID == "" {
		return nil, errors.New("supabase token response missing access_token or user id")
	}

	return &parsed, nil
}

func (s *SupabaseService) doJSON(ctx context.Context, method, endpoint, bearerToken string, payload any) (int, []byte, error) {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("apikey", s.apiKey)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	resp, err := s.httpClient.Do(req) // #nosec G704 -- baseURL is validated https SUPABASE_URL from server config
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read response body: %w", err)
	}

	return resp.StatusCode, rawBody, nil
}

func mapSupabaseAuthError(status int, rawBody []byte, fallback error) error {
	if status == http.StatusTooManyRequests {
		return ErrTooManyRequests
	}
	if status >= http.StatusInternalServerError {
		return fmt.Errorf("supabase upstream error status %d", status)
	}

	var errBody supabaseErrorBody
	_ = json.Unmarshal(rawBody, &errBody)
	msg := errBody.combined()

	switch {
	case strings.Contains(msg, "email_not_confirmed") || strings.Contains(msg, "email not confirmed"):
		return ErrEmailNotVerified
	case strings.Contains(msg, "user_already_exists") || strings.Contains(msg, "already registered") || strings.Contains(msg, "email_exists"):
		return ErrEmailAlreadyExists
	case strings.Contains(msg, "weak_password") || strings.Contains(msg, "password should be") || strings.Contains(msg, "password is known to be weak"):
		return ErrWeakPassword
	case strings.Contains(msg, "invalid_credentials") || strings.Contains(msg, "invalid login credentials"):
		return ErrInvalidCredentials
	case strings.Contains(msg, "otp_expired") || strings.Contains(msg, "expired") || strings.Contains(msg, "invalid token"):
		return ErrInvalidOrExpiredToken
	default:
		return fallback
	}
}
