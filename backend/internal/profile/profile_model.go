// Package profile manages user profiles, onboarding anthropometrics,
// daily nutrition targets, and UI preferences (language, theme).
package profile

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// MaxDisplayNameRunes is the maximum allowed rune length for a profile display name.
const MaxDisplayNameRunes = 50

// ErrInvalidInput indicates a malformed or out-of-range profile field.
var ErrInvalidInput = errors.New("invalid input")

// Profile represents a row in the profiles table.
type Profile struct {
	ID                 string
	ProfileURL         string
	DisplayName        string
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
	ID          string
	ProfileURL  string
	DisplayName string
	Language    string
	Theme       string
}

// UpdateOnboardingParams defines inputs for completing user onboarding.
type UpdateOnboardingParams struct {
	UserID             string
	DisplayName        *string
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
	DisplayName        *string
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
	DisplayName         string   `json:"display_name"`
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
	AuthProviders       []string `json:"auth_providers"`
	HasPassword         bool     `json:"has_password"`
}

// ToDTOWithAuth maps a Profile, verified email, and auth provider metadata into
// a canonical JSON DTO.
func ToDTOWithAuth(p *Profile, email string, providers []string, hasPassword bool) DTO {
	normProviders := normalizeAuthProviders(providers)
	if p == nil {
		return DTO{
			Email:         email,
			AuthProviders: normProviders,
			HasPassword:   hasPassword,
		}
	}
	return DTO{
		ID:                  p.ID,
		Email:               email,
		ProfileURL:          p.ProfileURL,
		DisplayName:         p.DisplayName,
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
		AuthProviders:       normProviders,
		HasPassword:         hasPassword,
	}
}

func normalizeAuthProviders(providers []string) []string {
	if len(providers) == 0 {
		return []string{"email"}
	}
	seen := make(map[string]struct{}, len(providers))
	out := make([]string, 0, len(providers))
	for _, raw := range providers {
		v := strings.ToLower(strings.TrimSpace(raw))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	if len(out) == 0 {
		return []string{"email"}
	}
	return out
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

// ValidateDisplayName normalizes and validates an optional user display name.
func ValidateDisplayName(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if utf8.RuneCountInString(v) > MaxDisplayNameRunes {
		return "", ErrInvalidInput
	}
	for _, r := range v {
		if unicode.IsControl(r) {
			return "", ErrInvalidInput
		}
	}
	return v, nil
}

// SanitizeGoogleDisplayName strips control characters, trims whitespace, and
// safely truncates a Google profile name to MaxDisplayNameRunes runes.
func SanitizeGoogleDisplayName(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))
	for _, r := range raw {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	cleaned := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(cleaned) <= MaxDisplayNameRunes {
		return cleaned
	}
	runes := []rune(cleaned)
	return strings.TrimSpace(string(runes[:MaxDisplayNameRunes]))
}
