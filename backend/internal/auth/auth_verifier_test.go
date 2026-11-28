package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Nouriko/internal/config"
	"Nouriko/internal/httpx"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
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
		claims, ok := httpx.ClaimsFromContext(c)
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

func TestVerifyLocalTokenProvidersAndHasPassword(t *testing.T) {
	v := newLocalVerifier(t)
	uid := uuid.NewString()

	// 1. Default SignLocalToken embeds ["email"] -> HasPassword() == true.
	emailTok, err := SignLocalToken(testJWTSecret, uid, "raka@example.com", "authenticated", time.Hour)
	if err != nil {
		t.Fatalf("SignLocalToken: %v", err)
	}
	claims, err := v.Verify(emailTok)
	if err != nil {
		t.Fatalf("Verify emailTok: %v", err)
	}
	if !claims.HasPassword() || len(claims.Providers) != 1 || claims.Providers[0] != "email" {
		t.Fatalf("unexpected email claims: %+v", claims)
	}

	// 2. Google-only token -> HasPassword() == false.
	googleTok, err := SignLocalTokenWithProviders(testJWTSecret, uid, "g@example.com", "authenticated", []string{"google"}, time.Hour)
	if err != nil {
		t.Fatalf("SignLocalTokenWithProviders(google): %v", err)
	}
	claims, err = v.Verify(googleTok)
	if err != nil {
		t.Fatalf("Verify googleTok: %v", err)
	}
	if claims.HasPassword() || len(claims.Providers) != 1 || claims.Providers[0] != "google" {
		t.Fatalf("unexpected google-only claims: %+v", claims)
	}

	// 3. Linked email+google token -> HasPassword() == true.
	linkedTok, err := SignLocalTokenWithProviders(testJWTSecret, uid, "linked@example.com", "authenticated", []string{"email", "google"}, time.Hour)
	if err != nil {
		t.Fatalf("SignLocalTokenWithProviders(email+google): %v", err)
	}
	claims, err = v.Verify(linkedTok)
	if err != nil {
		t.Fatalf("Verify linkedTok: %v", err)
	}
	if !claims.HasPassword() || len(claims.Providers) != 2 {
		t.Fatalf("unexpected linked claims: %+v", claims)
	}

	// 4. Legacy token without app_metadata -> backward-compatible HasPassword() == true, NormalizedProviders() == ["email"].
	now := time.Now()
	legacyJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   uid,
		"email": "legacy@example.com",
		"role":  "authenticated",
		"aud":   "authenticated",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	})
	legacyRaw, err := legacyJWT.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign legacy jwt: %v", err)
	}
	claims, err = v.Verify(legacyRaw)
	if err != nil {
		t.Fatalf("Verify legacy: %v", err)
	}
	if !claims.HasPassword() || len(claims.NormalizedProviders()) != 1 || claims.NormalizedProviders()[0] != "email" {
		t.Fatalf("unexpected legacy claims: %+v", claims)
	}

	// 5. Token with only app_metadata.provider="google" (fallback when providers array omitted).
	providerOnlyJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   uid,
		"email": "p@example.com",
		"role":  "authenticated",
		"aud":   "authenticated",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
		"app_metadata": map[string]any{
			"provider": "google",
		},
	})
	providerOnlyRaw, err := providerOnlyJWT.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("sign provider-only jwt: %v", err)
	}
	claims, err = v.Verify(providerOnlyRaw)
	if err != nil {
		t.Fatalf("Verify provider-only: %v", err)
	}
	if claims.HasPassword() || len(claims.Providers) != 1 || claims.Providers[0] != "google" {
		t.Fatalf("unexpected provider-only claims: %+v", claims)
	}
}
