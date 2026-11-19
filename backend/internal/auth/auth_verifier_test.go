package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Nouriko/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func newLocalVerifier(t *testing.T) *Verifier {
	t.Helper()
	cfg := &config.Config{
		Mode: config.ModeLocal,
		Auth: config.Auth{
			JWTSecret: testJWTSecret,
			Audience:  "authenticated",
		},
	}
	v, err := NewVerifier(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error creating verifier: %v", err)
	}
	return v
}

func TestVerifyLocalTokenSuccess(t *testing.T) {
	v := newLocalVerifier(t)
	uid := uuid.NewString()

	token, err := SignLocalToken(testJWTSecret, uid, "raka@example.com", "authenticated", time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	claims, err := v.Verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.UserID != uid || claims.Email != "raka@example.com" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyLocalTokenRejectsExpired(t *testing.T) {
	v := newLocalVerifier(t)
	token, err := SignLocalToken(testJWTSecret, uuid.NewString(), "raka@example.com", "authenticated", -time.Minute)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := v.Verify(token); err == nil {
		t.Fatal("expected expired token to fail")
	}
}

func TestMiddlewareEnforcesBearerAuth(t *testing.T) {
	v := newLocalVerifier(t)
	app := fiber.New()
	app.Get("/me", Middleware(v), func(c fiber.Ctx) error {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			return fiber.ErrUnauthorized
		}
		return c.JSON(fiber.Map{"user_id": claims.UserID})
	})

	// Missing header -> 401
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	// Valid Bearer token -> 200
	token, err := SignLocalToken(testJWTSecret, uuid.NewString(), "raka@example.com", "authenticated", time.Hour)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
