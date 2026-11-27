package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/config"

	"github.com/gofiber/fiber/v3"
)

func newAuthHandlerTestApp(t *testing.T) (*fiber.App, *memAuthRepo) {
	return newAuthHandlerTestAppWithGoogle(t, nil)
}

func newAuthHandlerTestAppWithGoogle(t *testing.T, gv GoogleTokenVerifier) (*fiber.App, *memAuthRepo) {
	t.Helper()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret: testJWTSecret,
			Audience:  "authenticated",
		},
	}
	if gv != nil {
		cfg.Auth.GoogleClientID = "test-google-client-id"
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	repo := newMemAuthRepo()
	verifier, err := NewVerifier(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	svc := NewService(cfg, repo, repo, nil, gv, logger)
	h := NewHandler(svc, logger)

	app := fiber.New()
	api := app.Group("/api/v1")
	api.Post("/auth/register", h.Register)
	api.Post("/auth/verify-email", h.VerifyEmail)
	api.Post("/auth/login", h.Login)
	api.Post("/auth/google", h.GoogleLogin)
	api.Post("/auth/forgot-password", h.ForgotPassword)
	api.Post("/auth/reset-password", h.ResetPassword)

	authMW := Middleware(verifier)
	api.Post("/auth/logout", authMW, h.Logout)
	api.Post("/auth/change-password", authMW, h.ChangePassword)
	api.Delete("/account", authMW, h.DeleteAccount)

	return app, repo
}

func doAuthJSONReq(t *testing.T, app *fiber.App, method, path, bearer string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp, decoded
}

func assertAuthError(t *testing.T, resp *http.Response, body map[string]any, wantStatus int, wantCode string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected status %d, got %d (body=%+v)", wantStatus, resp.StatusCode, body)
	}
	if len(body) != 1 || body["code"] != wantCode {
		t.Fatalf("PRD error violation: expected ONLY {code:%q}, got %+v", wantCode, body)
	}
}

func TestAuthHandler_FullAuthAndAccountLifecycle(t *testing.T) {
	app, repo := newAuthHandlerTestApp(t)

	// 1. Register weak password -> 400 WEAK_PASSWORD
	resp, body := doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "password123",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "WEAK_PASSWORD")

	// 2. Register valid -> 201 Created
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
		"language": "id",
		"theme":    "light",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%+v)", resp.StatusCode, body)
	}
	userID, _ := body["user_id"].(string)
	devToken, _ := body["dev_verification_token"].(string)
	if userID == "" || devToken == "" {
		t.Fatalf("expected user_id and dev_verification_token, got %+v", body)
	}

	// Duplicate register -> 409 EMAIL_ALREADY_EXISTS
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	assertAuthError(t, resp, body, http.StatusConflict, "EMAIL_ALREADY_EXISTS")

	// 3. Login before verification -> 403 EMAIL_NOT_VERIFIED
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	assertAuthError(t, resp, body, http.StatusForbidden, "EMAIL_NOT_VERIFIED")

	// 4. Verify email -> 200 verified; second use -> 400 INVALID_OR_EXPIRED_TOKEN
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{
		"token": devToken,
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "verified" {
		t.Fatalf("verify-email failed: %d %+v", resp.StatusCode, body)
	}
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{
		"token": devToken,
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN")

	// 5. Login -> 200 with JWT + profile DTO (DiceBear URL & onboarding_completed=false)
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %+v", resp.StatusCode, body)
	}
	accessToken, _ := body["access_token"].(string)
	profileMap, _ := body["profile"].(map[string]any)
	if accessToken == "" || profileMap["onboarding_completed"] != false {
		t.Fatalf("unexpected login response: %+v", body)
	}
	if profileMap["profile_url"] != DiceBearAvatarURL(userID) {
		t.Fatalf("unexpected DiceBear avatar: %v", profileMap["profile_url"])
	}

	// 6. Forgot password -> always 200 {"status":"ok"}
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "unknown@example.com",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("forgot-password unknown failed: %d %+v", resp.StatusCode, body)
	}
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "raka@example.com",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "ok" || len(body) != 1 {
		t.Fatalf("forgot-password known must return identical response, got %d %+v", resp.StatusCode, body)
	}

	// 7. Reset password
	rawReset, hashReset, _ := GenerateRandomToken()
	_, _ = repo.CreateAuthToken(context.Background(), CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hashReset,
		Purpose:   TokenPurposeResetPassword,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "tehtarikhangat",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_reset" {
		t.Fatalf("reset-password failed: %d %+v", resp.StatusCode, body)
	}

	// 8. Change password
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/change-password", accessToken, map[string]any{
		"current_password": "wrongpassword",
		"new_password":     "nasigorengspesial",
	})
	assertAuthError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/change-password", accessToken, map[string]any{
		"current_password": "tehtarikhangat",
		"new_password":     "nasigorengspesial",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_changed" {
		t.Fatalf("change-password failed: %d %+v", resp.StatusCode, body)
	}

	// 9. Logout
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/logout", accessToken, nil)
	if resp.StatusCode != http.StatusOK || body["status"] != "logged_out" {
		t.Fatalf("logout failed: %d %+v", resp.StatusCode, body)
	}

	// 10. Delete account -> wrong password rejected, correct password deletes permanently
	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "wrongpassword",
	})
	assertAuthError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "nasigorengspesial",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "account_deleted" {
		t.Fatalf("delete account failed: %d %+v", resp.StatusCode, body)
	}
}

func TestAuthHandler_EdgeCasesAndStrictErrorContract(t *testing.T) {
	app, repo := newAuthHandlerTestApp(t)

	// 1. Protected auth/account endpoints reject missing Bearer token with 401 UNAUTHORIZED
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/auth/change-password"},
		{http.MethodDelete, "/api/v1/account"},
	} {
		resp, body := doAuthJSONReq(t, app, tc.method, tc.path, "", nil)
		assertAuthError(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED")
	}

	// 2. Malformed JSON returns 400 INVALID_INPUT
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{bad-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("malformed json test: %v", err)
	}
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	assertAuthError(t, resp, decoded, http.StatusBadRequest, "INVALID_INPUT")

	// 3. Forgot password malformed email -> 400 INVALID_INPUT
	resp, body := doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "not-an-email",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")

	// 4. Register, verify, login
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "edge@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register edge user: %d %+v", resp.StatusCode, body)
	}
	userID := body["user_id"].(string)
	devToken := body["dev_verification_token"].(string)

	resp, _ = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{"token": devToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify edge user: %d", resp.StatusCode)
	}
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "edge@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login edge user: %d", resp.StatusCode)
	}
	accessToken := body["access_token"].(string)

	// 5. Reset password single-use and weak password rejection
	rawReset, hashReset, _ := GenerateRandomToken()
	_, _ = repo.CreateAuthToken(context.Background(), CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hashReset,
		Purpose:   TokenPurposeResetPassword,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "password",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "WEAK_PASSWORD")

	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "nasikuninghangat",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_reset" {
		t.Fatalf("reset password expected 200, got %d %+v", resp.StatusCode, body)
	}
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "nasikuninghangat2",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN")

	// 6. DELETE /api/v1/account empty password -> 400 INVALID_INPUT
	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")
}

func TestAuthHandler_GoogleSignInAndReauthDelete(t *testing.T) {
	const validNonce = "0123456789abcdef0123456789abcdef"

	// 1. Unconfigured Google auth -> 503 OAUTH_UNAVAILABLE
	disabledApp, _ := newAuthHandlerTestAppWithGoogle(t, nil)
	resp, body := doAuthJSONReq(t, disabledApp, http.MethodPost, "/api/v1/auth/google", "", map[string]any{
		"id_token": "some-token",
		"nonce":    validNonce,
	})
	assertAuthError(t, resp, body, http.StatusServiceUnavailable, "OAUTH_UNAVAILABLE")

	// 2. Configured Google auth
	now := time.Now().UTC()
	gv := &mockGoogleVerifier{
		fn: func(_ context.Context, idToken, _ string) (*GoogleIdentity, error) {
			switch idToken {
			case "valid-token":
				return &GoogleIdentity{
					Subject:       "google-sub-handler",
					Email:         "budi@gmail.com",
					Name:          "Budi Santoso",
					EmailVerified: true,
					IssuedAt:      now.Add(-15 * time.Second),
				}, nil
			case "stale-token":
				return &GoogleIdentity{
					Subject:       "google-sub-handler",
					Email:         "budi@gmail.com",
					Name:          "Budi Santoso",
					EmailVerified: true,
					IssuedAt:      now.Add(-10 * time.Minute),
				}, nil
			default:
				return nil, ErrInvalidCredentials
			}
		},
	}
	app, _ := newAuthHandlerTestAppWithGoogle(t, gv)

	// Invalid input (short nonce) -> 400 INVALID_INPUT
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]any{
		"id_token": "valid-token",
		"nonce":    "too-short",
	})
	assertAuthError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")

	// Invalid Google token -> 401 INVALID_CREDENTIALS
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]any{
		"id_token": "bad-token",
		"nonce":    validNonce,
	})
	assertAuthError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	// Valid Google login -> 200 OK with auth_providers=["google"] and has_password=false
	resp, body = doAuthJSONReq(t, app, http.MethodPost, "/api/v1/auth/google", "", map[string]any{
		"id_token": "valid-token",
		"nonce":    validNonce,
		"language": "en",
		"theme":    "dark",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for google login, got %d (%+v)", resp.StatusCode, body)
	}
	accessToken, _ := body["access_token"].(string)
	profileMap, _ := body["profile"].(map[string]any)
	if accessToken == "" || profileMap["has_password"] != false || profileMap["display_name"] != "Budi Santoso" {
		t.Fatalf("unexpected google login response: %+v", body)
	}
	provs, _ := profileMap["auth_providers"].([]any)
	if len(provs) != 1 || provs[0] != "google" {
		t.Fatalf("expected auth_providers=[google], got %+v", profileMap["auth_providers"])
	}

	// DELETE /api/v1/account with stale Google token -> 401 INVALID_CREDENTIALS
	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"google_id_token": "stale-token",
		"nonce":           validNonce,
	})
	assertAuthError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	// DELETE /api/v1/account replaying the login nonce -> 401 INVALID_CREDENTIALS
	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"google_id_token": "valid-token",
		"nonce":           validNonce,
	})
	assertAuthError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	// DELETE /api/v1/account with fresh Google token & fresh nonce -> 200 account_deleted
	const freshReauthNonce = "fedcba9876543210fedcba9876543210"
	resp, body = doAuthJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"google_id_token": "valid-token",
		"nonce":           freshReauthNonce,
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "account_deleted" {
		t.Fatalf("expected 200 account_deleted for Google re-auth delete, got %d (%+v)", resp.StatusCode, body)
	}
}
