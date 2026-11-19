package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type memProfileRepo struct {
	mu       sync.Mutex
	profiles map[string]*Profile
}

func newMemProfileRepo() *memProfileRepo {
	return &memProfileRepo{profiles: make(map[string]*Profile)}
}

func (m *memProfileRepo) UpsertProfile(_ context.Context, p UpsertProfileParams) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.profiles[p.ID]; ok {
		cp := *existing
		return &cp, nil
	}
	now := time.Now()
	prof := &Profile{ID: p.ID, ProfileURL: p.ProfileURL, Language: p.Language, Theme: p.Theme, CreatedAt: now, UpdatedAt: now}
	m.profiles[p.ID] = prof
	cp := *prof
	return &cp, nil
}

func (m *memProfileRepo) GetProfileByID(_ context.Context, userID string) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *prof
	return &cp, nil
}

func (m *memProfileRepo) UpdateOnboarding(_ context.Context, p UpdateOnboardingParams) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[p.UserID]
	if !ok {
		return nil, database.ErrNotFound
	}
	prof.HeightCM = &p.HeightCM
	prof.WeightKG = &p.WeightKG
	prof.Age = &p.Age
	prof.Gender = &p.Gender
	prof.ActivityLevel = &p.ActivityLevel
	prof.DailyCalorieTarget = &p.DailyCalorieTarget
	prof.TargetProteinG = &p.TargetProteinG
	prof.TargetCarbsG = &p.TargetCarbsG
	prof.TargetFatG = &p.TargetFatG
	prof.TargetFiberG = &p.TargetFiberG
	cp := *prof
	return &cp, nil
}

func (m *memProfileRepo) UpdateProfile(_ context.Context, p UpdateProfileParams) (*Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[p.UserID]
	if !ok {
		return nil, database.ErrNotFound
	}
	if p.HeightCM != nil {
		prof.HeightCM = p.HeightCM
	}
	if p.WeightKG != nil {
		prof.WeightKG = p.WeightKG
	}
	if p.Age != nil {
		prof.Age = p.Age
	}
	if p.Gender != nil {
		prof.Gender = p.Gender
	}
	if p.ActivityLevel != nil {
		prof.ActivityLevel = p.ActivityLevel
	}
	if p.DailyCalorieTarget != nil {
		prof.DailyCalorieTarget = p.DailyCalorieTarget
	}
	if p.TargetProteinG != nil {
		prof.TargetProteinG = p.TargetProteinG
	}
	if p.TargetCarbsG != nil {
		prof.TargetCarbsG = p.TargetCarbsG
	}
	if p.TargetFatG != nil {
		prof.TargetFatG = p.TargetFatG
	}
	if p.TargetFiberG != nil {
		prof.TargetFiberG = p.TargetFiberG
	}
	if p.Language != nil {
		prof.Language = *p.Language
	}
	if p.Theme != nil {
		prof.Theme = *p.Theme
	}
	cp := *prof
	return &cp, nil
}

func (m *memProfileRepo) DeleteProfile(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[userID]; !ok {
		return database.ErrNotFound
	}
	delete(m.profiles, userID)
	return nil
}

func newProfileTestApp(repo Repository, claims *httpx.Claims) *fiber.App {
	app := fiber.New()
	h := NewHandler(repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	injectClaims := func(c fiber.Ctx) error {
		if claims != nil && c.Get("Authorization") == "Bearer valid-token" {
			httpx.SetClaims(c, *claims)
		}
		return c.Next()
	}
	app.Get("/api/v1/profile", injectClaims, h.GetProfile)
	app.Put("/api/v1/profile/onboarding", injectClaims, h.CompleteOnboarding)
	app.Patch("/api/v1/profile", injectClaims, h.UpdateProfile)
	return app
}

func doProfileJSONReq(t *testing.T, app *fiber.App, method, path, bearer string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp, decoded
}

func assertProfileError(t *testing.T, resp *http.Response, body map[string]any, wantStatus int, wantCode string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected status %d, got %d (body=%+v)", wantStatus, resp.StatusCode, body)
	}
	if len(body) != 1 || body["code"] != wantCode {
		t.Fatalf("PRD error violation: expected ONLY {code:%q}, got %+v", wantCode, body)
	}
}

func TestProfileHandler_GetOnboardingAndPatch(t *testing.T) {
	repo := newMemProfileRepo()
	userID := uuid.NewString()
	_, _ = repo.UpsertProfile(context.Background(), UpsertProfileParams{
		ID:         userID,
		ProfileURL: "https://api.dicebear.com/10.x/slice/svg?seed=" + userID,
		Language:   "id",
		Theme:      "light",
	})
	claims := &httpx.Claims{UserID: userID, Email: "raka@example.com", Role: "authenticated"}
	app := newProfileTestApp(repo, claims)

	// 1. Missing auth -> 401 UNAUTHORIZED
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/profile"},
		{http.MethodPut, "/api/v1/profile/onboarding"},
		{http.MethodPatch, "/api/v1/profile"},
	} {
		resp, body := doProfileJSONReq(t, app, tc.method, tc.path, "", nil)
		assertProfileError(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED")
	}

	// 2. GET /api/v1/profile -> 200
	resp, body := doProfileJSONReq(t, app, http.MethodGet, "/api/v1/profile", "valid-token", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /profile failed: %d %+v", resp.StatusCode, body)
	}
	profileMap := body["profile"].(map[string]any)
	if profileMap["onboarding_completed"] != false {
		t.Fatalf("expected onboarding_completed=false, got %+v", profileMap)
	}

	// 3. PUT /api/v1/profile/onboarding -> 400 on invalid input, 200 on valid
	resp, body = doProfileJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", "valid-token", map[string]any{
		"height_cm": 50,
	})
	assertProfileError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")

	resp, body = doProfileJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", "valid-token", map[string]any{
		"height_cm":      170,
		"weight_kg":      65,
		"age":            20,
		"gender":         "male",
		"activity_level": "moderate",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /profile/onboarding failed: %d %+v", resp.StatusCode, body)
	}
	profileMap = body["profile"].(map[string]any)
	if profileMap["onboarding_completed"] != true || int(profileMap["daily_calorie_target"].(float64)) != 2507 {
		t.Fatalf("unexpected onboarding profile: %+v", profileMap)
	}

	// 4. PATCH /api/v1/profile -> updates preferences + recalculates on anthro change + honors explicit override
	resp, body = doProfileJSONReq(t, app, http.MethodPatch, "/api/v1/profile", "valid-token", map[string]any{
		"language":       "en",
		"theme":          "dark",
		"activity_level": "heavy",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /profile failed: %d %+v", resp.StatusCode, body)
	}
	profileMap = body["profile"].(map[string]any)
	// BMR 1617.5 * 1.725 = 2790.1875 -> 2790
	if profileMap["language"] != "en" || profileMap["theme"] != "dark" || int(profileMap["daily_calorie_target"].(float64)) != 2790 {
		t.Fatalf("unexpected recalculated profile: %+v", profileMap)
	}

	resp, body = doProfileJSONReq(t, app, http.MethodPatch, "/api/v1/profile", "valid-token", map[string]any{
		"weight_kg":            68,
		"daily_calorie_target": 2600,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH /profile with override failed: %d %+v", resp.StatusCode, body)
	}
	profileMap = body["profile"].(map[string]any)
	if int(profileMap["daily_calorie_target"].(float64)) != 2600 {
		t.Fatalf("expected custom daily_calorie_target=2600 to override recalculation, got %v", profileMap["daily_calorie_target"])
	}

	// 5. PATCH /api/v1/profile rejects invalid language, theme, and out-of-range targets
	invalidPatches := []map[string]any{
		{"language": "jp"},
		{"theme": "sepia"},
		{"daily_calorie_target": 100},
		{"target_protein_g": 9999},
		{"target_carbs_g": 5},
		{"target_fat_g": 500},
		{"target_fiber_g": 200},
	}
	for _, patch := range invalidPatches {
		resp, body = doProfileJSONReq(t, app, http.MethodPatch, "/api/v1/profile", "valid-token", patch)
		assertProfileError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")
	}

	// 6. Deleted profile -> 404 NOT_FOUND
	_ = repo.DeleteProfile(context.Background(), userID)
	resp, body = doProfileJSONReq(t, app, http.MethodGet, "/api/v1/profile", "valid-token", nil)
	assertProfileError(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}
