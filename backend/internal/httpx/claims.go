// Package httpx provides shared HTTP request context helpers and PRD-compliant
// generic error responses for Fiber v3 handlers across all domain contexts.
package httpx

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

type localsKey string

const claimsLocalsKey localsKey = "auth_claims"

// Claims holds the verified user identity extracted from a Bearer JWT.
type Claims struct {
	UserID string
	Email  string
	Role   string
}

// SetClaims stores verified user claims on the Fiber request context.
func SetClaims(c fiber.Ctx, claims Claims) {
	c.Locals(claimsLocalsKey, claims)
}

// ClaimsFromContext retrieves verified user claims stored by auth middleware.
func ClaimsFromContext(c fiber.Ctx) (Claims, bool) {
	val := c.Locals(claimsLocalsKey)
	claims, ok := val.(Claims)
	return claims, ok
}

// ExtractBearerToken extracts the raw Bearer token from the Authorization header.
// It clones the string so callers never retain Fiber's reusable request buffer.
func ExtractBearerToken(c fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.Clone(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
}
