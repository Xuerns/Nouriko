package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"Nouriko/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository defines persistence operations for the profiles table.
type Repository interface {
	UpsertProfile(ctx context.Context, params UpsertProfileParams) (*Profile, error)
	GetProfileByID(ctx context.Context, userID string) (*Profile, error)
	UpdateOnboarding(ctx context.Context, params UpdateOnboardingParams) (*Profile, error)
	UpdateProfile(ctx context.Context, params UpdateProfileParams) (*Profile, error)
	DeleteProfile(ctx context.Context, userID string) error
}

// Store implements Repository backed by PostgreSQL (pgx/v5).
type Store struct {
	q database.DBTX
}

// Compile-time check ensuring Store implements Repository.
var _ Repository = (*Store)(nil)

// NewStore creates a PostgreSQL-backed profile Store from a connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{q: pool}
}

// NewStoreWithDBTX creates a profile Store bound to an existing DBTX (such as a pgx.Tx or test double).
func NewStoreWithDBTX(q database.DBTX) *Store {
	return &Store{q: q}
}

const profileColumns = `id, profile_url, height_cm, weight_kg, age, gender, activity_level, daily_calorie_target, target_protein_g, target_carbs_g, target_fat_g, target_fiber_g, language, theme, created_at, updated_at`

func scanProfile(row pgx.Row) (*Profile, error) {
	var p Profile
	err := row.Scan(
		&p.ID,
		&p.ProfileURL,
		&p.HeightCM,
		&p.WeightKG,
		&p.Age,
		&p.Gender,
		&p.ActivityLevel,
		&p.DailyCalorieTarget,
		&p.TargetProteinG,
		&p.TargetCarbsG,
		&p.TargetFatG,
		&p.TargetFiberG,
		&p.Language,
		&p.Theme,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, database.ErrNotFound
		}
		return nil, err
	}
	return &p, nil
}

// UpsertProfile creates a profiles row if absent, or returns the existing row
// (backfilling profile_url if it was empty).
func (s *Store) UpsertProfile(ctx context.Context, params UpsertProfileParams) (*Profile, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("profile store is not initialized")
	}

	language := strings.TrimSpace(params.Language)
	if language == "" {
		language = "id"
	}
	theme := strings.TrimSpace(params.Theme)
	if theme == "" {
		theme = "light"
	}

	const query = `
		INSERT INTO profiles (id, profile_url, language, theme)
		VALUES ($1::UUID, $2::TEXT, $3::TEXT, $4::TEXT)
		ON CONFLICT (id) DO UPDATE
		SET
			profile_url = CASE WHEN profiles.profile_url = '' THEN EXCLUDED.profile_url ELSE profiles.profile_url END,
			updated_at = NOW()
		RETURNING ` + profileColumns

	p, err := scanProfile(s.q.QueryRow(ctx, query, params.ID, params.ProfileURL, language, theme))
	if err != nil {
		return nil, fmt.Errorf("upsert profile: %w", err)
	}
	return p, nil
}

// GetProfileByID fetches a user's profile by UUID.
func (s *Store) GetProfileByID(ctx context.Context, userID string) (*Profile, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("profile store is not initialized")
	}

	const query = `
		SELECT ` + profileColumns + `
		FROM profiles
		WHERE id = $1::UUID
	`
	p, err := scanProfile(s.q.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("get profile by id: %w", err)
	}
	return p, nil
}

// UpdateOnboarding updates all anthropometric inputs and computed daily macro
// targets for a user's profile.
func (s *Store) UpdateOnboarding(ctx context.Context, params UpdateOnboardingParams) (*Profile, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("profile store is not initialized")
	}

	const query = `
		UPDATE profiles
		SET
			height_cm = $2::NUMERIC(5, 2),
			weight_kg = $3::NUMERIC(5, 2),
			age = $4::SMALLINT,
			gender = $5::TEXT,
			activity_level = $6::TEXT,
			daily_calorie_target = $7::INTEGER,
			target_protein_g = $8::NUMERIC(6, 2),
			target_carbs_g = $9::NUMERIC(6, 2),
			target_fat_g = $10::NUMERIC(6, 2),
			target_fiber_g = $11::NUMERIC(6, 2),
			updated_at = NOW()
		WHERE id = $1::UUID
		RETURNING ` + profileColumns

	p, err := scanProfile(s.q.QueryRow(
		ctx,
		query,
		params.UserID,
		params.HeightCM,
		params.WeightKG,
		params.Age,
		params.Gender,
		params.ActivityLevel,
		params.DailyCalorieTarget,
		params.TargetProteinG,
		params.TargetCarbsG,
		params.TargetFatG,
		params.TargetFiberG,
	))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("update onboarding: %w", err)
	}
	return p, nil
}

// UpdateProfile partially updates non-nil fields on a user's profile using
// COALESCE with explicit PostgreSQL type casts for Supavisor compatibility.
func (s *Store) UpdateProfile(ctx context.Context, params UpdateProfileParams) (*Profile, error) {
	if s == nil || s.q == nil {
		return nil, fmt.Errorf("profile store is not initialized")
	}

	const query = `
		UPDATE profiles
		SET
			profile_url = COALESCE($2::TEXT, profile_url),
			height_cm = COALESCE($3::NUMERIC(5, 2), height_cm),
			weight_kg = COALESCE($4::NUMERIC(5, 2), weight_kg),
			age = COALESCE($5::SMALLINT, age),
			gender = COALESCE($6::TEXT, gender),
			activity_level = COALESCE($7::TEXT, activity_level),
			daily_calorie_target = COALESCE($8::INTEGER, daily_calorie_target),
			target_protein_g = COALESCE($9::NUMERIC(6, 2), target_protein_g),
			target_carbs_g = COALESCE($10::NUMERIC(6, 2), target_carbs_g),
			target_fat_g = COALESCE($11::NUMERIC(6, 2), target_fat_g),
			target_fiber_g = COALESCE($12::NUMERIC(6, 2), target_fiber_g),
			language = COALESCE($13::TEXT, language),
			theme = COALESCE($14::TEXT, theme),
			updated_at = NOW()
		WHERE id = $1::UUID
		RETURNING ` + profileColumns

	p, err := scanProfile(s.q.QueryRow(
		ctx,
		query,
		params.UserID,
		params.ProfileURL,
		params.HeightCM,
		params.WeightKG,
		params.Age,
		params.Gender,
		params.ActivityLevel,
		params.DailyCalorieTarget,
		params.TargetProteinG,
		params.TargetCarbsG,
		params.TargetFatG,
		params.TargetFiberG,
		params.Language,
		params.Theme,
	))
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, database.ErrNotFound
		}
		return nil, fmt.Errorf("update profile: %w", err)
	}
	return p, nil
}

// DeleteProfile deletes a user's profile row, cascading to saved_meals via
// ON DELETE CASCADE.
func (s *Store) DeleteProfile(ctx context.Context, userID string) error {
	if s == nil || s.q == nil {
		return fmt.Errorf("profile store is not initialized")
	}

	const query = `DELETE FROM profiles WHERE id = $1::UUID`
	tag, err := s.q.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}
