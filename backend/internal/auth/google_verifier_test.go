package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func startMockGoogleJWKS(t *testing.T, kid string, pub *rsa.PublicKey) *httptest.Server {
	t.Helper()
	nEnc := base64.RawURLEncoding.EncodeToString(pub.N.Bytes())
	eBytes := big.NewInt(int64(pub.E)).Bytes()
	eEnc := base64.RawURLEncoding.EncodeToString(eBytes)

	payload := map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"kid": kid,
				"use": "sig",
				"alg": "RS256",
				"n":   nEnc,
				"e":   eEnc,
			},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func signMockGoogleJWT(t *testing.T, priv *rsa.PrivateKey, kid string, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign mock google jwt: %v", err)
	}
	return signed
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func TestGoogleJWKSVerifier_ValidAndRejectionCases(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	const (
		kid      = "google-test-kid-1"
		clientID = "1234567890-abcdef.apps.googleusercontent.com"
		rawNonce = "0123456789abcdef0123456789abcdef"
	)

	srv := startMockGoogleJWKS(t, kid, &priv.PublicKey)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	fixedNow := time.Now().UTC().Truncate(time.Second)
	verifier := newGoogleJWKSVerifierWithURL(ctx, clientID, srv.URL)
	verifier.now = func() time.Time { return fixedNow }

	baseClaims := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss":            "https://accounts.google.com",
			"aud":            clientID,
			"sub":            "google-sub-12345",
			"email":          "Raka.Google@Gmail.com",
			"email_verified": true,
			"name":           "Raka Pratama",
			"nonce":          sha256Hex(rawNonce),
			"iat":            fixedNow.Add(-30 * time.Second).Unix(),
			"exp":            fixedNow.Add(10 * time.Minute).Unix(),
		}
	}

	t.Run("valid RS256 token with https issuer and matching SHA-256 nonce", func(t *testing.T) {
		tok := signMockGoogleJWT(t, priv, kid, baseClaims())
		id, err := verifier.VerifyIDToken(ctx, tok, rawNonce)
		if err != nil {
			t.Fatalf("expected valid token, got error: %v", err)
		}
		if id.Subject != "google-sub-12345" || id.Email != "raka.google@gmail.com" || id.Name != "Raka Pratama" || !id.EmailVerified {
			t.Fatalf("unexpected identity: %+v", id)
		}
		if !id.IssuedAt.Equal(fixedNow.Add(-30 * time.Second)) {
			t.Fatalf("unexpected IssuedAt: %v", id.IssuedAt)
		}
	})

	t.Run("valid RS256 token with accounts.google.com issuer", func(t *testing.T) {
		c := baseClaims()
		c["iss"] = "accounts.google.com"
		tok := signMockGoogleJWT(t, priv, kid, c)
		if _, err := verifier.VerifyIDToken(ctx, tok, rawNonce); err != nil {
			t.Fatalf("expected accounts.google.com issuer to succeed, got %v", err)
		}
	})

	t.Run("rejects wrong aud", func(t *testing.T) {
		c := baseClaims()
		c["aud"] = "other-client.apps.googleusercontent.com"
		tok := signMockGoogleJWT(t, priv, kid, c)
		if _, err := verifier.VerifyIDToken(ctx, tok, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejects wrong iss", func(t *testing.T) {
		c := baseClaims()
		c["iss"] = "https://evil.example.com"
		tok := signMockGoogleJWT(t, priv, kid, c)
		if _, err := verifier.VerifyIDToken(ctx, tok, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejects expired exp", func(t *testing.T) {
		c := baseClaims()
		c["exp"] = fixedNow.Add(-time.Minute).Unix()
		tok := signMockGoogleJWT(t, priv, kid, c)
		if _, err := verifier.VerifyIDToken(ctx, tok, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejects missing iat and future iat", func(t *testing.T) {
		cMissing := baseClaims()
		delete(cMissing, "iat")
		tokMissing := signMockGoogleJWT(t, priv, kid, cMissing)
		if _, err := verifier.VerifyIDToken(ctx, tokMissing, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for missing iat, got %v", err)
		}

		cFuture := baseClaims()
		cFuture["iat"] = fixedNow.Add(5 * time.Minute).Unix()
		tokFuture := signMockGoogleJWT(t, priv, kid, cFuture)
		if _, err := verifier.VerifyIDToken(ctx, tokFuture, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for future iat, got %v", err)
		}
	})

	t.Run("rejects email_verified=false", func(t *testing.T) {
		c := baseClaims()
		c["email_verified"] = false
		tok := signMockGoogleJWT(t, priv, kid, c)
		if _, err := verifier.VerifyIDToken(ctx, tok, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("rejects empty sub or invalid email", func(t *testing.T) {
		cSub := baseClaims()
		cSub["sub"] = ""
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cSub), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for empty sub, got %v", err)
		}

		cEmail := baseClaims()
		cEmail["email"] = "not-an-email"
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cEmail), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for bad email, got %v", err)
		}
	})

	t.Run("rejects nonce mismatch and missing nonce", func(t *testing.T) {
		tok := signMockGoogleJWT(t, priv, kid, baseClaims())
		if _, err := verifier.VerifyIDToken(ctx, tok, "different-nonce-value-123456"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for wrong nonce, got %v", err)
		}

		cNoNonce := baseClaims()
		delete(cNoNonce, "nonce")
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cNoNonce), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for missing nonce claim, got %v", err)
		}
	})

	t.Run("rejects non-RS256 algorithm (HS256)", func(t *testing.T) {
		hsTok := jwt.NewWithClaims(jwt.SigningMethodHS256, baseClaims())
		hsTok.Header["kid"] = kid
		signed, err := hsTok.SignedString([]byte(testJWTSecret))
		if err != nil {
			t.Fatalf("sign hs256: %v", err)
		}
		if _, err := verifier.VerifyIDToken(ctx, signed, rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for HS256 token, got %v", err)
		}
	})

	t.Run("rejects input bounds violations", func(t *testing.T) {
		validTok := signMockGoogleJWT(t, priv, kid, baseClaims())
		if _, err := verifier.VerifyIDToken(ctx, "", rawNonce); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for empty idToken, got %v", err)
		}
		if _, err := verifier.VerifyIDToken(ctx, strings.Repeat("a", maxGoogleIDTokenBytes+1), rawNonce); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for oversized idToken, got %v", err)
		}
		if _, err := verifier.VerifyIDToken(ctx, validTok, "short"); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for short nonce, got %v", err)
		}
		if _, err := verifier.VerifyIDToken(ctx, validTok, strings.Repeat("b", maxGoogleNonceLen+1)); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("expected ErrInvalidInput for oversized nonce, got %v", err)
		}
	})

	t.Run("rejects azp mismatch and multi-audience without matching azp", func(t *testing.T) {
		cMultiNoAzp := baseClaims()
		cMultiNoAzp["aud"] = []string{clientID, "other-client.apps.googleusercontent.com"}
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cMultiNoAzp), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for multi-aud without azp, got %v", err)
		}

		cBadAzp := baseClaims()
		cBadAzp["azp"] = "wrong-azp.apps.googleusercontent.com"
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cBadAzp), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials for wrong azp, got %v", err)
		}

		cMultiValidAzp := baseClaims()
		cMultiValidAzp["aud"] = []string{clientID, "other-client.apps.googleusercontent.com"}
		cMultiValidAzp["azp"] = clientID
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cMultiValidAzp), rawNonce); err != nil {
			t.Fatalf("expected valid multi-aud token with matching azp, got %v", err)
		}
	})

	t.Run("accepts minor positive iat clock skew within leeway and rejects beyond leeway", func(t *testing.T) {
		cMinorSkew := baseClaims()
		cMinorSkew["iat"] = fixedNow.Add(30 * time.Second).Unix()
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cMinorSkew), rawNonce); err != nil {
			t.Fatalf("expected token within leeway to succeed, got %v", err)
		}

		cLargeSkew := baseClaims()
		cLargeSkew["iat"] = fixedNow.Add(2 * time.Minute).Unix()
		if _, err := verifier.VerifyIDToken(ctx, signMockGoogleJWT(t, priv, kid, cLargeSkew), rawNonce); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("expected ErrInvalidCredentials beyond iat leeway, got %v", err)
		}
	})

	t.Run("unconfigured verifier returns ErrOAuthUnavailable", func(t *testing.T) {
		unconfigured := NewGoogleJWKSVerifier(ctx, "")
		if _, err := unconfigured.VerifyIDToken(ctx, "tok", rawNonce); !errors.Is(err, ErrOAuthUnavailable) {
			t.Fatalf("expected ErrOAuthUnavailable, got %v", err)
		}
	})
}
