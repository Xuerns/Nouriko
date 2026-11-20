package profile

import (
	"log/slog"

	"Nouriko/internal/httpx"
	"Nouriko/internal/nutrition"

	"github.com/gofiber/fiber/v3"
)

// Handler handles user profile, onboarding, and preference endpoints.
type Handler struct {
	repo   Repository
	logger *slog.Logger
}

// NewHandler creates a new profile Handler.
func NewHandler(repo Repository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		repo:   repo,
		logger: logger,
	}
}

// GetProfile handles GET /api/v1/profile (auth required).
func (h *Handler) GetProfile(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	p, err := h.repo.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToDTO(p, claims.Email),
	})
}

type onboardingRequest struct {
	DisplayName   *string  `json:"display_name"`
	HeightCM      *float64 `json:"height_cm"`
	WeightKG      *float64 `json:"weight_kg"`
	Age           *int16   `json:"age"`
	Gender        *string  `json:"gender"`
	ActivityLevel *string  `json:"activity_level"`
}

// CompleteOnboarding handles PUT /api/v1/profile/onboarding (auth required, PRD F8).
func (h *Handler) CompleteOnboarding(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	var req onboardingRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}
	if req.HeightCM == nil || req.WeightKG == nil || req.Age == nil || req.Gender == nil || req.ActivityLevel == nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	targets, err := nutrition.CalculateTargets(nutrition.Anthropometrics{
		HeightCM:      *req.HeightCM,
		WeightKG:      *req.WeightKG,
		Age:           int(*req.Age),
		Gender:        *req.Gender,
		ActivityLevel: *req.ActivityLevel,
	})
	if err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	var displayName *string
	if req.DisplayName != nil {
		cleaned, nameErr := ValidateDisplayName(*req.DisplayName)
		if nameErr != nil {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
		displayName = &cleaned
	}

	params := UpdateOnboardingParams{
		UserID:             claims.UserID,
		DisplayName:        displayName,
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
		return httpx.HandleDBError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToDTO(updated, claims.Email),
	})
}

type updateProfileRequest struct {
	DisplayName        *string  `json:"display_name"`
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
func (h *Handler) UpdateProfile(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	var req updateProfileRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	params, err := validateProfilePatch(claims.UserID, &req)
	if err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	current, err := h.repo.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
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
				return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
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
		return httpx.HandleDBError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"profile": ToDTO(updated, claims.Email),
	})
}

func validateProfilePatch(userID string, req *updateProfileRequest) (UpdateProfileParams, error) {
	params := UpdateProfileParams{
		UserID:        userID,
		HeightCM:      req.HeightCM,
		WeightKG:      req.WeightKG,
		Age:           req.Age,
		Gender:        req.Gender,
		ActivityLevel: req.ActivityLevel,
	}

	if req.DisplayName != nil {
		cleaned, err := ValidateDisplayName(*req.DisplayName)
		if err != nil {
			return UpdateProfileParams{}, ErrInvalidInput
		}
		params.DisplayName = &cleaned
	}
	if req.HeightCM != nil && !nutrition.ValidateHeightCM(*req.HeightCM) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.WeightKG != nil && !nutrition.ValidateWeightKG(*req.WeightKG) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.Age != nil && !nutrition.ValidateAge(int(*req.Age)) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.Gender != nil && !nutrition.ValidateGender(*req.Gender) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.ActivityLevel != nil && !nutrition.ValidateActivityLevel(*req.ActivityLevel) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.DailyCalorieTarget != nil && !nutrition.ValidateDailyCalorieTarget(*req.DailyCalorieTarget) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.TargetProteinG != nil && !nutrition.ValidateProteinG(*req.TargetProteinG) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.TargetCarbsG != nil && !nutrition.ValidateCarbsG(*req.TargetCarbsG) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.TargetFatG != nil && !nutrition.ValidateFatG(*req.TargetFatG) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.TargetFiberG != nil && !nutrition.ValidateFiberG(*req.TargetFiberG) {
		return UpdateProfileParams{}, ErrInvalidInput
	}
	if req.Language != nil {
		lang, err := ValidateLanguage(*req.Language, false)
		if err != nil {
			return UpdateProfileParams{}, ErrInvalidInput
		}
		params.Language = &lang
	}
	if req.Theme != nil {
		theme, err := ValidateTheme(*req.Theme, false)
		if err != nil {
			return UpdateProfileParams{}, ErrInvalidInput
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
