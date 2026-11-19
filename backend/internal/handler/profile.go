package handler

import (
	"log/slog"

	"Nouriko/internal/auth"
	"Nouriko/internal/database"
	"Nouriko/internal/nutrition"

	"github.com/gofiber/fiber/v3"
)

// ProfileHandler handles user profile, onboarding, and preference endpoints.
type ProfileHandler struct {
	repo   database.ProfileStore
	logger *slog.Logger
}

// NewProfileHandler creates a new ProfileHandler.
func NewProfileHandler(repo database.ProfileStore, logger *slog.Logger) *ProfileHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ProfileHandler{
		repo:   repo,
		logger: logger,
	}
}

// GetProfile handles GET /api/v1/profile (auth required).
func (h *ProfileHandler) GetProfile(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	profile, err := h.repo.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToProfileDTO(profile, claims.Email),
	})
}

type onboardingRequest struct {
	HeightCM      *float64 `json:"height_cm"`
	WeightKG      *float64 `json:"weight_kg"`
	Age           *int16   `json:"age"`
	Gender        *string  `json:"gender"`
	ActivityLevel *string  `json:"activity_level"`
}

// CompleteOnboarding handles PUT /api/v1/profile/onboarding (auth required, PRD F8).
func (h *ProfileHandler) CompleteOnboarding(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	var req onboardingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}
	if req.HeightCM == nil || req.WeightKG == nil || req.Age == nil || req.Gender == nil || req.ActivityLevel == nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	targets, err := nutrition.CalculateTargets(nutrition.Anthropometrics{
		HeightCM:      *req.HeightCM,
		WeightKG:      *req.WeightKG,
		Age:           int(*req.Age),
		Gender:        *req.Gender,
		ActivityLevel: *req.ActivityLevel,
	})
	if err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	params := database.UpdateOnboardingParams{
		UserID:             claims.UserID,
		HeightCM:           *req.HeightCM,
		WeightKG:           *req.WeightKG,
		Age:                *req.Age,
		Gender:             *req.Gender,
		ActivityLevel:      *req.ActivityLevel,
		DailyCalorieTarget: targets.DailyCalorieTarget,
		TargetProteinG:     targets.TargetProteinG,
		TargetCarbsG:       targets.TargetCarbsG,
		TargetFatG:         targets.TargetFatG,
		TargetFiberG:       targets.TargetFiberG,
	}

	updated, err := h.repo.UpdateOnboarding(c.Context(), params)
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToProfileDTO(updated, claims.Email),
	})
}

type updateProfileRequest struct {
	HeightCM           *float64 `json:"height_cm"`
	WeightKG           *float64 `json:"weight_kg"`
	Age                *int16   `json:"age"`
	Gender             *string  `json:"gender"`
	ActivityLevel      *string  `json:"activity_level"`
	DailyCalorieTarget *int     `json:"daily_calorie_target"`
	TargetProteinG     *float64 `json:"target_protein_g"`
	TargetCarbsG       *float64 `json:"target_carbs_g"`
	TargetFatG         *float64 `json:"target_fat_g"`
	TargetFiberG       *float64 `json:"target_fiber_g"`
	Language           *string  `json:"language"`
	Theme              *string  `json:"theme"`
}

// UpdateProfile handles PATCH /api/v1/profile (auth required, PRD F8/F11/F12).
func (h *ProfileHandler) UpdateProfile(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	var req updateProfileRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	params, err := validateProfilePatch(claims.UserID, &req)
	if err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	current, err := h.repo.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	anthroChanged := req.HeightCM != nil || req.WeightKG != nil || req.Age != nil || req.Gender != nil || req.ActivityLevel != nil
	if anthroChanged {
		height := coalesceFloat64(req.HeightCM, current.HeightCM)
		weight := coalesceFloat64(req.WeightKG, current.WeightKG)
		age := coalesceInt16(req.Age, current.Age)
		gender := coalesceString(req.Gender, current.Gender)
		activity := coalesceString(req.ActivityLevel, current.ActivityLevel)

		if height != nil && weight != nil && age != nil && gender != nil && activity != nil {
			targets, calcErr := nutrition.CalculateTargets(nutrition.Anthropometrics{
				HeightCM:      *height,
				WeightKG:      *weight,
				Age:           int(*age),
				Gender:        *gender,
				ActivityLevel: *activity,
			})
			if calcErr != nil {
				return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
			}

			params.DailyCalorieTarget = &targets.DailyCalorieTarget
			params.TargetProteinG = &targets.TargetProteinG
			params.TargetCarbsG = &targets.TargetCarbsG
			params.TargetFatG = &targets.TargetFatG
			params.TargetFiberG = &targets.TargetFiberG
		}
	}

	// Explicit custom target overrides provided in the same request take precedence.
	if req.DailyCalorieTarget != nil {
		params.DailyCalorieTarget = req.DailyCalorieTarget
	}
	if req.TargetProteinG != nil {
		params.TargetProteinG = req.TargetProteinG
	}
	if req.TargetCarbsG != nil {
		params.TargetCarbsG = req.TargetCarbsG
	}
	if req.TargetFatG != nil {
		params.TargetFatG = req.TargetFatG
	}
	if req.TargetFiberG != nil {
		params.TargetFiberG = req.TargetFiberG
	}

	updated, err := h.repo.UpdateProfile(c.Context(), params)
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToProfileDTO(updated, claims.Email),
	})
}

func validateProfilePatch(userID string, req *updateProfileRequest) (database.UpdateProfileParams, error) {
	params := database.UpdateProfileParams{
		UserID:        userID,
		HeightCM:      req.HeightCM,
		WeightKG:      req.WeightKG,
		Age:           req.Age,
		Gender:        req.Gender,
		ActivityLevel: req.ActivityLevel,
	}

	if req.HeightCM != nil && !nutrition.ValidateHeightCM(*req.HeightCM) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.WeightKG != nil && !nutrition.ValidateWeightKG(*req.WeightKG) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.Age != nil && !nutrition.ValidateAge(int(*req.Age)) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.Gender != nil && !nutrition.ValidateGender(*req.Gender) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.ActivityLevel != nil && !nutrition.ValidateActivityLevel(*req.ActivityLevel) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.DailyCalorieTarget != nil && !nutrition.ValidateDailyCalorieTarget(*req.DailyCalorieTarget) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.TargetProteinG != nil && !nutrition.ValidateProteinG(*req.TargetProteinG) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.TargetCarbsG != nil && !nutrition.ValidateCarbsG(*req.TargetCarbsG) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.TargetFatG != nil && !nutrition.ValidateFatG(*req.TargetFatG) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.TargetFiberG != nil && !nutrition.ValidateFiberG(*req.TargetFiberG) {
		return database.UpdateProfileParams{}, auth.ErrInvalidInput
	}
	if req.Language != nil {
		lang, err := auth.ValidateLanguage(*req.Language, false)
		if err != nil {
			return database.UpdateProfileParams{}, auth.ErrInvalidInput
		}
		params.Language = &lang
	}
	if req.Theme != nil {
		theme, err := auth.ValidateTheme(*req.Theme, false)
		if err != nil {
			return database.UpdateProfileParams{}, auth.ErrInvalidInput
		}
		params.Theme = &theme
	}

	return params, nil
}

func coalesceFloat64(a, b *float64) *float64 {
	if a != nil {
		return a
	}
	return b
}

func coalesceInt16(a, b *int16) *int16 {
	if a != nil {
		return a
	}
	return b
}

func coalesceString(a, b *string) *string {
	if a != nil {
		return a
	}
	return b
}
