// Package handler implements Fiber v3 HTTP handlers for the Nouriko REST API.
package handler

import (
	"errors"
	"log/slog"
	"strings"

	"Nouriko/internal/auth"
	"Nouriko/internal/database"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// ProfileDTO is the canonical JSON representation of a user profile returned by
// /api/v1/auth/login and /api/v1/profile* endpoints.
type ProfileDTO struct {
	ID                  string   `json:"id"`
	Email               string   `json:"email"`
	ProfileURL          string   `json:"profile_url"`
	HeightCM            *float64 `json:"height_cm"`
	WeightKG            *float64 `json:"weight_kg"`
	Age                 *int16   `json:"age"`
	Gender              *string  `json:"gender"`
	ActivityLevel       *string  `json:"activity_level"`
	DailyCalorieTarget  *int     `json:"daily_calorie_target"`
	TargetProteinG      *float64 `json:"target_protein_g"`
	TargetCarbsG        *float64 `json:"target_carbs_g"`
	TargetFatG          *float64 `json:"target_fat_g"`
	TargetFiberG        *float64 `json:"target_fiber_g"`
	Language            string   `json:"language"`
	Theme               string   `json:"theme"`
	OnboardingCompleted bool     `json:"onboarding_completed"`
}

// ToProfileDTO maps a database.Profile and verified email into a ProfileDTO.
func ToProfileDTO(p *database.Profile, email string) ProfileDTO {
	if p == nil {
		return ProfileDTO{Email: email}
	}
	return ProfileDTO{
		ID:                  p.ID,
		Email:               email,
		ProfileURL:          p.ProfileURL,
		HeightCM:            p.HeightCM,
		WeightKG:            p.WeightKG,
		Age:                 p.Age,
		Gender:              p.Gender,
		ActivityLevel:       p.ActivityLevel,
		DailyCalorieTarget:  p.DailyCalorieTarget,
		TargetProteinG:      p.TargetProteinG,
		TargetCarbsG:        p.TargetCarbsG,
		TargetFatG:          p.TargetFatG,
		TargetFiberG:        p.TargetFiberG,
		Language:            p.Language,
		Theme:               p.Theme,
		OnboardingCompleted: p.OnboardingCompleted(),
	}
}

func writeErrorCode(c fiber.Ctx, status int, code string) error {
	return c.Status(status).JSON(fiber.Map{"code": code})
}

func handleServiceError(c fiber.Ctx, logger *slog.Logger, err error) error {
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	case errors.Is(err, auth.ErrWeakPassword):
		return writeErrorCode(c, fiber.StatusBadRequest, "WEAK_PASSWORD")
	case errors.Is(err, auth.ErrInvalidOrExpiredToken), errors.Is(err, database.ErrTokenInvalidOrExpired):
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN")
	case errors.Is(err, auth.ErrInvalidCredentials):
		return writeErrorCode(c, fiber.StatusUnauthorized, "INVALID_CREDENTIALS")
	case errors.Is(err, auth.ErrUnauthorized):
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	case errors.Is(err, auth.ErrEmailNotVerified):
		return writeErrorCode(c, fiber.StatusForbidden, "EMAIL_NOT_VERIFIED")
	case errors.Is(err, auth.ErrNotFound), errors.Is(err, database.ErrNotFound):
		return writeErrorCode(c, fiber.StatusNotFound, "NOT_FOUND")
	case errors.Is(err, auth.ErrEmailAlreadyExists), errors.Is(err, database.ErrConflict):
		return writeErrorCode(c, fiber.StatusConflict, "EMAIL_ALREADY_EXISTS")
	case errors.Is(err, auth.ErrTooManyRequests):
		return writeErrorCode(c, fiber.StatusTooManyRequests, "TOO_MANY_REQUESTS")
	default:
		if logger != nil {
			logger.Error("internal handler error",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
		}
		return writeErrorCode(c, fiber.StatusInternalServerError, "INTERNAL_ERROR")
	}
}

func extractBearerToken(c fiber.Ctx) string {
	header := c.Get(fiber.HeaderAuthorization)
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}
