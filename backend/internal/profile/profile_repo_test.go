package profile

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestProfileOnboardingCompleted(t *testing.T) {
	var nilProfile *Profile
	if nilProfile.OnboardingCompleted() {
		t.Fatal("expected nil profile to have OnboardingCompleted() == false")
	}

	h := 170.0
	w := 65.0
	age := int16(20)
	gender := "male"
	activity := "moderate"
	kcal := 2523

	incomplete := &Profile{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		HeightCM: &h,
		WeightKG: &w,
		Age:      &age,
		Gender:   &gender,
		Language: "id",
		Theme:    "light",
	}
	if incomplete.OnboardingCompleted() {
		t.Fatal("expected incomplete profile to return OnboardingCompleted() == false")
	}

	complete := &Profile{
		ID:                 "550e8400-e29b-41d4-a716-446655440000",
		HeightCM:           &h,
		WeightKG:           &w,
		Age:                &age,
		Gender:             &gender,
		ActivityLevel:      &activity,
		DailyCalorieTarget: &kcal,
		Language:           "id",
		Theme:              "light",
	}
	if !complete.OnboardingCompleted() {
		t.Fatal("expected complete profile to return OnboardingCompleted() == true")
	}
}

type fakeRow struct {
	scanFn func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scanFn == nil {
		return pgx.ErrNoRows
	}
	return r.scanFn(dest...)
}

type fakeDBTX struct {
	execFn     func(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *fakeDBTX) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if f.execFn != nil {
		return f.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (f *fakeDBTX) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	return nil, errors.New("unexpected Query call")
}

func (f *fakeDBTX) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if f.queryRowFn != nil {
		return f.queryRowFn(ctx, sql, args...)
	}
	return fakeRow{}
}

func TestStoreProfileRepository(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	userID := "550e8400-e29b-41d4-a716-446655440000"
	avatarURL := "https://api.dicebear.com/10.x/slice/svg?seed=" + userID

	t.Run("UpsertProfile applies defaults for language and theme", func(t *testing.T) {
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, _ string, args ...any) pgx.Row {
				if args[2] != "id" || args[3] != "light" {
					t.Errorf("expected default language=id and theme=light, got %#v %#v", args[2], args[3])
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = avatarURL
					*(dest[12].(*string)) = "id"
					*(dest[13].(*string)) = "light"
					*(dest[14].(*time.Time)) = now
					*(dest[15].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		p, err := store.UpsertProfile(ctx, UpsertProfileParams{
			ID:         userID,
			ProfileURL: avatarURL,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ProfileURL != avatarURL || p.OnboardingCompleted() {
			t.Fatalf("unexpected profile result: %+v", p)
		}
	})

	t.Run("UpdateProfile uses explicit PostgreSQL casts for COALESCE compatibility with Supavisor", func(t *testing.T) {
		lang := "en"
		theme := "dark"
		db := &fakeDBTX{
			queryRowFn: func(_ context.Context, sql string, args ...any) pgx.Row {
				if !strings.Contains(sql, "COALESCE($13::TEXT, language)") || !strings.Contains(sql, "COALESCE($14::TEXT, theme)") {
					t.Errorf("expected explicit type casts in UpdateProfile SQL: %s", sql)
				}
				if len(args) != 14 {
					t.Errorf("expected 14 args for UpdateProfile, got %d", len(args))
				}
				return fakeRow{scanFn: func(dest ...any) error {
					*(dest[0].(*string)) = userID
					*(dest[1].(*string)) = avatarURL
					*(dest[12].(*string)) = lang
					*(dest[13].(*string)) = theme
					*(dest[14].(*time.Time)) = now
					*(dest[15].(*time.Time)) = now
					return nil
				}}
			},
		}
		store := NewStoreWithDBTX(db)
		p, err := store.UpdateProfile(ctx, UpdateProfileParams{
			UserID:   userID,
			Language: &lang,
			Theme:    &theme,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Language != "en" || p.Theme != "dark" {
			t.Fatalf("expected updated language/theme, got %+v", p)
		}
	})

	t.Run("Nil store returns error instead of panicking", func(t *testing.T) {
		var s *Store
		if _, err := s.GetProfileByID(ctx, userID); err == nil {
			t.Fatal("expected error from nil store")
		}
	})
}
