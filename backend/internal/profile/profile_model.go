// Package profile manages user profiles, onboarding anthropometrics,
// daily nutrition targets, and UI preferences (language, theme).
package profile

import (
	"errors"
	"strings"
	"time"
)

// ErrInvalidInput indicates a malformed or out-of-range profile field.
var ErrInvalidInput = errors.New("invalid input")

// Profile represents a row in the profiles table.
type Profile struct {
	ID                 string
	ProfileURL         string
	HeightCM           *float64
	WeightKG           *float64
	Age                *int16
	Gender             *string
	ActivityLevel      *string
	DailyCalorieTarget *int
	TargetProteinG     *float64
	TargetCarbsG       *float64
	TargetFatG         *float64
	TargetFiberG       *float64
	Language           string
	Theme              string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// OnboardingCompleted derives onboarding completion status from the required
// anthropometric fields and computed daily calorie target (zero-drift rule).
func (p *Profile) OnboardingCompleted() bool {
	if p == nil {
		return false
	}
	return p.HeightCM != nil &&
		p.WeightKG != nil &&
		p.Age != nil &&
		p.Gender != nil &&
		p.ActivityLevel != nil &&
		p.DailyCalorieTarget != nil
}

// UpsertProfileParams defines inputs for idempotently creating or ensuring a profile row.
type UpsertProfileParams struct {
	ID         string
	ProfileURL string
	Language   string
	Theme      string
}

// UpdateOnboardingParams defines inputs for completing user onboarding.
type UpdateOnboardingParams struct {
	UserID             string
	HeightCM           float64
	WeightKG           float64
	Age                int16
	Gender             string
	ActivityLevel      string
	DailyCalorieTarget int
	TargetProteinG     float64
	TargetCarbsG       float64
	TargetFatG         float64
	TargetFiberG       float64
}

// UpdateProfileParams defines optional fields for partial profile updates.
// Nil fields are preserved in the database via COALESCE.
type UpdateProfileParams struct {
	UserID             string
	ProfileURL         *string
	HeightCM           *float64
	WeightKG           *float64
	Age                *int16
	Gender             *string
	ActivityLevel      *string
	DailyCalorieTarget *int
	TargetProteinG     *float64
	TargetCarbsG       *float64
	TargetFatG         *float64
	TargetFiberG       *float64
	Language           *string
	Theme              *string
}

// DTO is the canonical JSON representation of a user profile returned by
// /api/v1/auth/login and /api/v1/profile* endpoints.
type DTO struct {
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

// ToDTO maps a Profile and verified email into a canonical JSON DTO.
func ToDTO(p *Profile, email string) DTO {
	if p == nil {
		return DTO{Email: email}
	}
	return DTO{
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

// ValidateLanguage normalizes and validates UI language ("id" or "en").
func ValidateLanguage(raw string, defaultIfEmpty bool) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" && defaultIfEmpty {
		return "id", nil
	}
	if v != "id" && v != "en" {
		return "", ErrInvalidInput
	}
	return v, nil
}

// ValidateTheme normalizes and validates UI theme ("light" or "dark").
func ValidateTheme(raw string, defaultIfEmpty bool) (string, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" && defaultIfEmpty {
		return "light", nil
	}
	if v != "light" && v != "dark" {
		return "", ErrInvalidInput
	}
	return v, nil
}
