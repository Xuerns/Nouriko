package database

import (
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

// Sentinel errors returned by database repositories.
var (
	ErrNotFound              = errors.New("record not found")
	ErrConflict              = errors.New("record already exists")
	ErrTokenInvalidOrExpired = errors.New("token invalid, already used, or expired")
)

// Supported local_auth_tokens purpose values.
const (
	TokenPurposeVerifyEmail   = "verify_email"
	TokenPurposeResetPassword = "reset_password"
)

// Supported food_prices category values.
const (
	FoodPriceCategoryMeal       = "meal"
	FoodPriceCategoryIngredient = "ingredient"
)

// pgUniqueViolationCode is PostgreSQL SQLSTATE 23505 (unique_violation).
const pgUniqueViolationCode = "23505"

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolationCode
}

// LocalUser represents a row in the local_users table (used when APP_ENV=local).
type LocalUser struct {
	ID                  string
	Email               string
	PasswordHash        string
	EmailVerified       bool
	FailedLoginAttempts int16
	LockedUntil         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// LocalAuthToken represents a row in the local_auth_tokens table.
type LocalAuthToken struct {
	ID        string
	UserID    string
	TokenHash string
	Purpose   string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

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

// CreateLocalUserParams defines inputs for inserting a new local user.
type CreateLocalUserParams struct {
	// ID is optional; when empty, PostgreSQL generates a UUID via gen_random_uuid().
	ID           string
	Email        string
	PasswordHash string
}

// CreateAuthTokenParams defines inputs for creating a single-use auth token.
type CreateAuthTokenParams struct {
	UserID    string
	TokenHash string
	Purpose   string
	ExpiresAt time.Time
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

// FoodPrice represents a row in the food_prices reference table.
type FoodPrice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	PriceIDR  int       `json:"price_idr"`
	Unit      string    `json:"unit"`
	Category  string    `json:"category"` // "meal" | "ingredient"
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
