package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/auth"
	"Nouriko/internal/config"

	"github.com/gofiber/fiber/v3"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func testConfig() *config.Config {
	return &config.Config{
		Mode:        config.ModeLocal,
		Port:        "8080",
		CORSOrigins: []string{"http://localhost:3000"},
		Limits: config.RateLimit{
			Max:    60,
			Window: time.Minute,
		},
	}
}

func TestHealthz(t *testing.T) {
	app := New(Dependencies{
		Config: testConfig(),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" || body["mode"] != "local" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestReadyzHealthyAndUnhealthy(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	okApp := New(Dependencies{Config: testConfig(), Logger: logger, DB: stubPinger{}})
	resp, err := okApp.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when db healthy, got %d", resp.StatusCode)
	}

	failApp := New(Dependencies{Config: testConfig(), Logger: logger, DB: stubPinger{err: errors.New("db down")}})
	resp, err = failApp.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when db down, got %d", resp.StatusCode)
	}
}

func TestNotFoundUsesGenericCode(t *testing.T) {
	app := New(Dependencies{
		Config: testConfig(),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "NOT_FOUND" {
		t.Fatalf("expected generic NOT_FOUND code, got %+v", body)
	}
}

type stubAuthService struct{}

func (stubAuthService) Register(context.Context, auth.RegisterInput) (*auth.RegisterResult, error) {
	return &auth.RegisterResult{Status: "verification_required"}, nil
}

func (stubAuthService) VerifyEmail(context.Context, string) error {
	return auth.ErrInvalidOrExpiredToken
}

func (stubAuthService) Login(context.Context, string, string) (*auth.LoginResult, error) {
	return nil, auth.ErrInvalidCredentials
}

func (stubAuthService) LoginWithGoogle(context.Context, auth.GoogleLoginInput) (*auth.LoginResult, error) {
	return nil, auth.ErrInvalidCredentials
}

func (stubAuthService) Logout(context.Context, string) error {
	return nil
}

func (stubAuthService) ForgotPassword(context.Context, string) error {
	return nil
}

func (stubAuthService) ResetPassword(context.Context, string, string) error {
	return auth.ErrInvalidOrExpiredToken
}

func (stubAuthService) ChangePassword(context.Context, string, string, string, string, string) error {
	return auth.ErrInvalidCredentials
}

func (stubAuthService) DeleteAccount(context.Context, auth.DeleteAccountInput) error {
	return auth.ErrInvalidCredentials
}

func TestAuthStrictRateLimiterAppliedToSensitiveEndpoints(t *testing.T) {
	const testSecret = "0123456789abcdef0123456789abcdef"
	token, err := auth.SignLocalToken(testSecret, "550e8400-e29b-41d4-a716-446655440000", "raka@example.com", "authenticated", time.Hour)
	if err != nil {
		t.Fatalf("SignLocalToken: %v", err)
	}

	endpoints := []struct {
		name     string
		method   string
		path     string
		payload  string
		withAuth bool
	}{
		{
			name:    "google-login",
			method:  http.MethodPost,
			path:    "/api/v1/auth/google",
			payload: `{"id_token":"bad-token","nonce":"0123456789abcdef"}`,
		},
		{
			name:    "verify-email",
			method:  http.MethodPost,
			path:    "/api/v1/auth/verify-email",
			payload: `{"token":"bad-token"}`,
		},
		{
			name:    "reset-password",
			method:  http.MethodPost,
			path:    "/api/v1/auth/reset-password",
			payload: `{"token":"bad-token","new_password":"kopisusupagi"}`,
		},
		{
			name:     "change-password",
			method:   http.MethodPost,
			path:     "/api/v1/auth/change-password",
			payload:  `{"current_password":"wrong","new_password":"kopisusupagi"}`,
			withAuth: true,
		},
		{
			name:     "delete-account",
			method:   http.MethodDelete,
			path:     "/api/v1/account",
			payload:  `{"password":"wrong"}`,
			withAuth: true,
		},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Auth = config.Auth{
				JWTSecret: testSecret,
				Audience:  "authenticated",
			}
			cfg.Limits = config.RateLimit{Max: 60, Window: time.Minute}
			cfg.AuthLimits = config.AuthRateLimit{Max: 2, Window: time.Minute}

			verifier, err := auth.NewVerifier(context.Background(), cfg)
			if err != nil {
				t.Fatalf("NewVerifier: %v", err)
			}

			app := New(Dependencies{
				Config:      cfg,
				Logger:      slog.New(slog.NewJSONHandler(io.Discard, nil)),
				DB:          stubPinger{},
				Auth:        verifier,
				AuthService: stubAuthService{},
			})

			for i := 0; i < 3; i++ {
				req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.payload))
				req.Header.Set("Content-Type", "application/json")
				if tc.withAuth {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
				if err != nil {
					t.Fatalf("attempt %d app.Test: %v", i+1, err)
				}
				if i < 2 && resp.StatusCode == http.StatusTooManyRequests {
					t.Fatalf("attempt %d unexpectedly rate limited", i+1)
				}
				if i == 2 {
					if resp.StatusCode != http.StatusTooManyRequests {
						t.Fatalf("attempt 3 expected 429 Too Many Requests, got %d", resp.StatusCode)
					}
					var body map[string]string
					if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
						t.Fatalf("decode 429 body: %v", err)
					}
					if body["code"] != "TOO_MANY_REQUESTS" {
						t.Fatalf("expected TOO_MANY_REQUESTS code, got %+v", body)
					}
				}
			}
		})
	}
}

func TestStatusReportsGoogleAuthEnabled(t *testing.T) {
	cfg := testConfig()
	app := New(Dependencies{
		Config: cfg,
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if body["google_auth_enabled"] != false {
		t.Fatalf("expected google_auth_enabled=false, got %+v", body)
	}

	cfg.Auth.GoogleClientID = "client-id.apps.googleusercontent.com"
	app2 := New(Dependencies{
		Config: cfg,
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})
	resp2, err := app2.Test(httptest.NewRequest(http.MethodGet, "/api/v1/status", nil))
	if err != nil {
		t.Fatalf("app2.Test: %v", err)
	}
	var body2 map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode status 2: %v", err)
	}
	if body2["google_auth_enabled"] != true {
		t.Fatalf("expected google_auth_enabled=true, got %+v", body2)
	}
}
