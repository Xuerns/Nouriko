// Package auth verifies access tokens for both local mode (HS256) and
// production mode (Supabase Auth JWKS), implements dual-mode authentication
// services, and provides HTTP handlers for /api/v1/auth/* and /api/v1/account.
package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"Nouriko/internal/config"
	"Nouriko/internal/httpx"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims aliases httpx.Claims for verified JWT identity.
type Claims = httpx.Claims

type appMetadataClaims struct {
	Provider  string   `json:"provider,omitempty"`
	Providers []string `json:"providers,omitempty"`
}

type tokenClaims struct {
	Email       string             `json:"email,omitempty"`
	Role        string             `json:"role,omitempty"`
	AppMetadata *appMetadataClaims `json:"app_metadata,omitempty"`
	jwt.RegisteredClaims
}

// Verifier validates incoming Bearer JWTs according to the active APP_ENV mode.
type Verifier struct {
	mode      config.Mode
	jwtSecret []byte
	jwks      keyfunc.Keyfunc
	issuer    string
	audience  string
}

// NewVerifier initializes a Verifier for local (HS256) or production (Supabase JWKS).
func NewVerifier(ctx context.Context, cfg *config.Config) (*Verifier, error) {
	v := &Verifier{
		mode:      cfg.Mode,
		jwtSecret: []byte(cfg.Auth.JWTSecret),
		issuer:    cfg.Auth.Issuer,
		audience:  cfg.Auth.Audience,
	}

	if cfg.IsProduction() {
		jwksURL := cfg.JWKSURL()
		if jwksURL == "" {
			return nil, errors.New("supabase JWKS URL is empty")
		}
		kf, err := keyfunc.NewDefaultCtx(ctx, []string{jwksURL})
		if err != nil {
			return nil, fmt.Errorf("init supabase jwks: %w", err)
		}
		v.jwks = kf
	}

	return v, nil
}

// Verify parses and validates a raw JWT string and returns the user claims.
func (v *Verifier) Verify(rawToken string) (Claims, error) {
	if rawToken == "" {
		return Claims{}, errors.New("empty token")
	}

	opts := []jwt.ParserOption{
		jwt.WithExpirationRequired(),
		jwt.WithValidMethods(v.allowedMethods()),
	}
	if v.audience != "" {
		opts = append(opts, jwt.WithAudience(v.audience))
	}
	if v.issuer != "" {
		opts = append(opts, jwt.WithIssuer(v.issuer))
	}

	tc := &tokenClaims{}
	token, err := jwt.ParseWithClaims(rawToken, tc, v.keyFunc, opts...)
	if err != nil || !token.Valid {
		return Claims{}, fmt.Errorf("invalid token: %w", err)
	}
	if tc.Subject == "" {
		return Claims{}, errors.New("token missing subject")
	}
	if _, err := uuid.Parse(tc.Subject); err != nil {
		return Claims{}, errors.New("token subject is not a valid uuid")
	}

	return Claims{
		UserID:    tc.Subject,
		Email:     tc.Email,
		Role:      tc.Role,
		Providers: extractAppMetadataProviders(tc.AppMetadata),
	}, nil
}

func extractAppMetadataProviders(meta *appMetadataClaims) []string {
	if meta == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(meta.Providers)+1)
	out := make([]string, 0, len(meta.Providers)+1)
	add := func(raw string) {
		p := strings.ToLower(strings.TrimSpace(raw))
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range meta.Providers {
		add(p)
	}
	if meta.Provider != "" {
		add(meta.Provider)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeProviderList(providers []string) []string {
	seen := make(map[string]struct{}, len(providers))
	out := make([]string, 0, len(providers))
	for _, raw := range providers {
		p := strings.ToLower(strings.TrimSpace(raw))
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return []string{"email"}
	}
	return out
}

func (v *Verifier) allowedMethods() []string {
	if v.mode == config.ModeLocal {
		return []string{jwt.SigningMethodHS256.Alg()}
	}
	methods := []string{
		jwt.SigningMethodRS256.Alg(),
		jwt.SigningMethodES256.Alg(),
	}
	if len(v.jwtSecret) > 0 {
		methods = append(methods, jwt.SigningMethodHS256.Alg())
	}
	return methods
}

func (v *Verifier) keyFunc(token *jwt.Token) (any, error) {
	if token.Method.Alg() == jwt.SigningMethodHS256.Alg() {
		if len(v.jwtSecret) == 0 {
			return nil, errors.New("hs256 secret is not configured")
		}
		return v.jwtSecret, nil
	}
	if v.jwks == nil {
		return nil, errors.New("jwks keyfunc is not initialized")
	}
	return v.jwks.Keyfunc(token)
}

// SignLocalToken creates an HS256 JWT for local development and tests with the
// default ["email"] provider list.
func SignLocalToken(secret, userID, email, audience string, ttl time.Duration) (string, error) {
	return SignLocalTokenWithProviders(secret, userID, email, audience, []string{"email"}, ttl)
}

// SignLocalTokenWithProviders creates an HS256 JWT embedding Supabase-compatible
// app_metadata.provider and app_metadata.providers claims.
func SignLocalTokenWithProviders(secret, userID, email, audience string, providers []string, ttl time.Duration) (string, error) {
	if len(secret) < 32 {
		return "", errors.New("secret must be at least 32 characters")
	}
	normProviders := normalizeProviderList(providers)
	now := time.Now()
	claims := tokenClaims{
		Email: email,
		Role:  "authenticated",
		AppMetadata: &appMetadataClaims{
			Provider:  normProviders[0],
			Providers: normProviders,
		},
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// Middleware enforces Bearer token authentication on protected Fiber routes.
func Middleware(v *Verifier) fiber.Handler {
	return func(c fiber.Ctx) error {
		rawToken := httpx.ExtractBearerToken(c)
		if rawToken == "" {
			return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
		}

		claims, err := v.Verify(rawToken)
		if err != nil {
			return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
		}

		httpx.SetClaims(c, claims)
		return c.Next()
	}
}

// ClaimsFromContext retrieves verified user claims stored by Middleware.
func ClaimsFromContext(c fiber.Ctx) (Claims, bool) {
	return httpx.ClaimsFromContext(c)
}

// AdminClient performs privileged Supabase Auth operations (such as F13 account deletion).
type AdminClient struct {
	baseURL    string
	secretKey  string
	httpClient *http.Client
}

// NewAdminClient returns a Supabase Auth Admin API client when configured, or nil in local mode.
func NewAdminClient(cfg *config.Config) *AdminClient {
	if cfg.Supabase.URL == "" || cfg.Supabase.SecretKey == "" {
		return nil
	}
	return &AdminClient{
		baseURL:   cfg.Supabase.URL,
		secretKey: cfg.Supabase.SecretKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// DeleteUser permanently deletes a user from Supabase Auth (PRD F13).
func (a *AdminClient) DeleteUser(ctx context.Context, userID string) error {
	if a == nil {
		return errors.New("supabase admin client is not configured")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("invalid user id: %w", err)
	}

	endpoint := fmt.Sprintf("%s/auth/v1/admin/users/%s", a.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create delete user request: %w", err)
	}
	req.Header.Set("apikey", a.secretKey)
	req.Header.Set("Authorization", "Bearer "+a.secretKey)

	resp, err := a.httpClient.Do(req) // #nosec G704 -- baseURL is validated https SUPABASE_URL from server config
	if err != nil {
		return fmt.Errorf("execute delete user request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("supabase delete user failed with status %d", resp.StatusCode)
	}
	return nil
}
