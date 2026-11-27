package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	defaultGoogleJWKSURL        = "https://www.googleapis.com/oauth2/v3/certs"
	maxGoogleIDTokenBytes       = 8192
	minGoogleNonceLen           = 16
	maxGoogleNonceLen           = 256
	googleMaxIssuedAtFutureSkew = time.Minute
)

// GoogleIdentity holds verified identity claims extracted from a Google ID token.
type GoogleIdentity struct {
	Subject       string
	Email         string
	Name          string
	EmailVerified bool
	IssuedAt      time.Time
	ExpiresAt     time.Time
}

// GoogleTokenVerifier validates Google Identity Services (GIS) ID tokens and
// their associated cryptographic SHA-256 nonces.
type GoogleTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken, rawNonce string) (*GoogleIdentity, error)
}

type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	var rawBool bool
	if err := json.Unmarshal(data, &rawBool); err == nil {
		*b = flexBool(rawBool)
		return nil
	}
	var rawStr string
	if err := json.Unmarshal(data, &rawStr); err == nil {
		switch strings.ToLower(strings.TrimSpace(rawStr)) {
		case "true":
			*b = true
			return nil
		case "false":
			*b = false
			return nil
		}
	}
	return fmt.Errorf("invalid boolean claim")
}

type googleIDTokenClaims struct {
	Email           string   `json:"email"`
	EmailVerified   flexBool `json:"email_verified"`
	Name            string   `json:"name"`
	Nonce           string   `json:"nonce"`
	AuthorizedParty string   `json:"azp"`
	jwt.RegisteredClaims
}

// GoogleJWKSVerifier verifies Google ID tokens using RS256 and Google's public JWKS.
type GoogleJWKSVerifier struct {
	clientID string
	jwksURL  string
	bgCtx    context.Context

	mu   sync.Mutex
	jwks keyfunc.Keyfunc
	now  func() time.Time
}

// Compile-time interface check.
var _ GoogleTokenVerifier = (*GoogleJWKSVerifier)(nil)

// NewGoogleJWKSVerifier constructs a GoogleJWKSVerifier bound to clientID.
// The JWKS client is initialized lazily on first use and retries after transient errors.
func NewGoogleJWKSVerifier(ctx context.Context, clientID string) *GoogleJWKSVerifier {
	return newGoogleJWKSVerifierWithURL(ctx, clientID, defaultGoogleJWKSURL)
}

func newGoogleJWKSVerifierWithURL(ctx context.Context, clientID, jwksURL string) *GoogleJWKSVerifier {
	if ctx == nil {
		ctx = context.Background()
	}
	return &GoogleJWKSVerifier{
		clientID: strings.TrimSpace(clientID),
		jwksURL:  strings.TrimSpace(jwksURL),
		bgCtx:    ctx,
		now:      time.Now,
	}
}

func newGoogleJWKSVerifierWithKeyfunc(clientID string, kf keyfunc.Keyfunc) *GoogleJWKSVerifier {
	return &GoogleJWKSVerifier{
		clientID: strings.TrimSpace(clientID),
		jwksURL:  defaultGoogleJWKSURL,
		bgCtx:    context.Background(),
		jwks:     kf,
		now:      time.Now,
	}
}

func (v *GoogleJWKSVerifier) getKeyfunc() (keyfunc.Keyfunc, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.jwks != nil {
		return v.jwks, nil
	}
	if v.jwksURL == "" {
		return nil, ErrOAuthUnavailable
	}

	kf, err := keyfunc.NewDefaultCtx(v.bgCtx, []string{v.jwksURL})
	if err != nil {
		return nil, fmt.Errorf("%w: init google jwks: %v", ErrOAuthUnavailable, err)
	}
	v.jwks = kf
	return kf, nil
}

// VerifyIDToken validates a Google ID token signature (RS256), audience/azp, issuer,
// expiration, issued-at, verified email, and constant-time SHA-256 nonce match.
func (v *GoogleJWKSVerifier) VerifyIDToken(ctx context.Context, idToken, rawNonce string) (*GoogleIdentity, error) {
	if v == nil || v.clientID == "" {
		return nil, ErrOAuthUnavailable
	}
	if err := validateGoogleTokenInputs(idToken, rawNonce); err != nil {
		return nil, err
	}
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}

	kf, err := v.getKeyfunc()
	if err != nil {
		return nil, err
	}

	nowFn := v.now
	if nowFn == nil {
		nowFn = time.Now
	}

	claims := &googleIDTokenClaims{}
	token, err := jwt.ParseWithClaims(
		idToken,
		claims,
		kf.Keyfunc,
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(googleMaxIssuedAtFutureSkew),
		jwt.WithAudience(v.clientID),
		jwt.WithTimeFunc(nowFn),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	// OIDC Core 3.1.3.7: enforce authorized party (azp) when multiple audiences or azp are present.
	if len(claims.Audience) > 1 && claims.AuthorizedParty != v.clientID {
		return nil, ErrInvalidCredentials
	}
	if claims.AuthorizedParty != "" && claims.AuthorizedParty != v.clientID {
		return nil, ErrInvalidCredentials
	}

	if claims.Issuer != "https://accounts.google.com" && claims.Issuer != "accounts.google.com" {
		return nil, ErrInvalidCredentials
	}
	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		return nil, ErrInvalidCredentials
	}
	if !bool(claims.EmailVerified) {
		return nil, ErrInvalidCredentials
	}
	if claims.IssuedAt == nil || claims.IssuedAt.Time.IsZero() {
		return nil, ErrInvalidCredentials
	}
	if claims.IssuedAt.Time.After(nowFn().Add(googleMaxIssuedAtFutureSkew)) {
		return nil, ErrInvalidCredentials
	}

	email, err := NormalizeAndValidateEmail(claims.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	expectedNonceHex := oauthReplayKey(rawNonce)
	if claims.Nonce == "" || subtle.ConstantTimeCompare([]byte(expectedNonceHex), []byte(claims.Nonce)) != 1 {
		return nil, ErrInvalidCredentials
	}

	var expiresAt time.Time
	if claims.ExpiresAt != nil {
		expiresAt = claims.ExpiresAt.Time
	}

	return &GoogleIdentity{
		Subject:       subject,
		Email:         email,
		Name:          claims.Name,
		EmailVerified: true,
		IssuedAt:      claims.IssuedAt.Time,
		ExpiresAt:     expiresAt,
	}, nil
}

func validateGoogleTokenInputs(idToken, rawNonce string) error {
	if idToken == "" || len(idToken) > maxGoogleIDTokenBytes {
		return ErrInvalidInput
	}
	if rawNonce == "" || len(rawNonce) < minGoogleNonceLen || len(rawNonce) > maxGoogleNonceLen {
		return ErrInvalidInput
	}
	return nil
}
