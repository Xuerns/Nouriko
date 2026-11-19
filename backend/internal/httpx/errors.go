package httpx

import (
	"errors"
	"log/slog"

	"Nouriko/internal/database"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Generic PRD-compliant error code constants returned in {"code": "..."} JSON responses.
const (
	CodeInvalidInput         = "INVALID_INPUT"
	CodeWeakPassword         = "WEAK_PASSWORD"
	CodeLinkInvalidOrExpired = "INVALID_OR_EXPIRED_TOKEN"
	CodeAuthFailed           = "INVALID_CREDENTIALS"
	CodeUnauthorized         = "UNAUTHORIZED"
	CodeForbidden            = "FORBIDDEN"
	CodeEmailNotVerified     = "EMAIL_NOT_VERIFIED"
	CodeOnboardingRequired   = "ONBOARDING_REQUIRED"
	CodeNotFound             = "NOT_FOUND"
	CodeMethodNotAllowed     = "METHOD_NOT_ALLOWED"
	CodeEmailAlreadyExists   = "EMAIL_ALREADY_EXISTS"
	CodePayloadTooLarge      = "PAYLOAD_TOO_LARGE"
	CodePhotoInvalid         = "PHOTO_INVALID"
	CodePhotoUnrecognized    = "PHOTO_UNRECOGNIZED"
	CodeTooManyRequests      = "TOO_MANY_REQUESTS"
	CodeInternalError        = "INTERNAL_ERROR"
	CodeServiceUnavailable   = "SERVICE_UNAVAILABLE"
	CodeAIUnavailable        = "AI_UNAVAILABLE"
	CodeAITimeout            = "AI_TIMEOUT"
)

// WriteError writes a generic PRD error response containing strictly {"code": code}.
func WriteError(c fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(fiber.Map{"code": code})
}

// HandleDBError maps database sentinel errors to generic PRD HTTP responses and
// logs unexpected internal errors with the request ID.
func HandleDBError(c fiber.Ctx, logger *slog.Logger, err error) error {
	switch {
	case errors.Is(err, database.ErrTokenInvalidOrExpired):
		return WriteError(c, fiber.StatusBadRequest, CodeLinkInvalidOrExpired)
	case errors.Is(err, database.ErrNotFound):
		return WriteError(c, fiber.StatusNotFound, CodeNotFound)
	case errors.Is(err, database.ErrConflict):
		return WriteError(c, fiber.StatusConflict, CodeEmailAlreadyExists)
	default:
		if logger != nil {
			logger.Error("internal handler error",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
		}
		return WriteError(c, fiber.StatusInternalServerError, CodeInternalError)
	}
}
