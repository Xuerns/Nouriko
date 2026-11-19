package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"Nouriko/internal/auth"
	"Nouriko/internal/config"
	"Nouriko/internal/database"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const testLocalSecret = "0123456789abcdef0123456789abcdef"

type fakeRepo struct {
	mu         sync.Mutex
	users      map[string]*database.LocalUser
	tokens     map[string]*database.LocalAuthToken
	profiles   map[string]*database.Profile
	savedMeals map[string]database.SavedMeal
	mealSeq    int64
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:      make(map[string]*database.LocalUser),
		tokens:     make(map[string]*database.LocalAuthToken),
		profiles:   make(map[string]*database.Profile),
		savedMeals: make(map[string]database.SavedMeal),
	}
}

func (f *fakeRepo) CreateLocalUser(_ context.Context, p database.CreateLocalUserParams) (*database.LocalUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	email := strings.ToLower(strings.TrimSpace(p.Email))
	for _, u := range f.users {
		if strings.ToLower(u.Email) == email {
			return nil, database.ErrConflict
		}
	}
	id := p.ID
	if id == "" {
		id = uuid.NewString()
	}
	now := time.Now()
	u := &database.LocalUser{ID: id, Email: email, PasswordHash: p.PasswordHash, CreatedAt: now, UpdatedAt: now}
	f.users[id] = u
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) GetLocalUserByEmail(_ context.Context, email string) (*database.LocalUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	target := strings.ToLower(strings.TrimSpace(email))
	for _, u := range f.users {
		if strings.ToLower(u.Email) == target {
			cp := *u
			return &cp, nil
		}
	}
	return nil, database.ErrNotFound
}

func (f *fakeRepo) GetLocalUserByID(_ context.Context, userID string) (*database.LocalUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) RecordFailedLogin(_ context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*database.LocalUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= maxAttempts {
		until := time.Now().Add(lockDuration)
		u.LockedUntil = &until
	}
	cp := *u
	return &cp, nil
}

func (f *fakeRepo) ResetLoginLockout(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.FailedLoginAttempts = 0
	u.LockedUntil = nil
	return nil
}

func (f *fakeRepo) MarkEmailVerified(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.EmailVerified = true
	return nil
}

func (f *fakeRepo) UpdatePasswordHash(_ context.Context, userID, passwordHash string, resetLockout bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.PasswordHash = passwordHash
	if resetLockout {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (f *fakeRepo) DeleteLocalUser(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[userID]; !ok {
		return database.ErrNotFound
	}
	delete(f.users, userID)
	return nil
}

func (f *fakeRepo) CreateAuthToken(_ context.Context, p database.CreateAuthTokenParams) (*database.LocalAuthToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tok := &database.LocalAuthToken{ID: uuid.NewString(), UserID: p.UserID, TokenHash: p.TokenHash, Purpose: p.Purpose, ExpiresAt: p.ExpiresAt, CreatedAt: time.Now()}
	f.tokens[p.TokenHash] = tok
	cp := *tok
	return &cp, nil
}

func (f *fakeRepo) InvalidateUserTokens(_ context.Context, userID, purpose string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	for _, tok := range f.tokens {
		if tok.UserID == userID && tok.Purpose == purpose && tok.UsedAt == nil {
			tok.UsedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) ConsumeAuthToken(_ context.Context, tokenHash, purpose string) (*database.LocalAuthToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tok, ok := f.tokens[tokenHash]
	if !ok || tok.Purpose != purpose || tok.UsedAt != nil || !tok.ExpiresAt.After(time.Now()) {
		return nil, database.ErrTokenInvalidOrExpired
	}
	now := time.Now()
	tok.UsedAt = &now
	cp := *tok
	return &cp, nil
}

func (f *fakeRepo) DeleteExpiredTokens(_ context.Context) (int64, error) { return 0, nil }

func (f *fakeRepo) UpsertProfile(_ context.Context, p database.UpsertProfileParams) (*database.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.profiles[p.ID]; ok {
		cp := *existing
		return &cp, nil
	}
	now := time.Now()
	prof := &database.Profile{ID: p.ID, ProfileURL: p.ProfileURL, Language: p.Language, Theme: p.Theme, CreatedAt: now, UpdatedAt: now}
	f.profiles[p.ID] = prof
	cp := *prof
	return &cp, nil
}

func (f *fakeRepo) GetProfileByID(_ context.Context, userID string) (*database.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prof, ok := f.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *prof
	return &cp, nil
}

func (f *fakeRepo) UpdateOnboarding(_ context.Context, p database.UpdateOnboardingParams) (*database.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prof, ok := f.profiles[p.UserID]
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

func (f *fakeRepo) UpdateProfile(_ context.Context, p database.UpdateProfileParams) (*database.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prof, ok := f.profiles[p.UserID]
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

func (f *fakeRepo) DeleteProfile(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.profiles[userID]; !ok {
		return database.ErrNotFound
	}
	delete(f.profiles, userID)
	return nil
}

func (f *fakeRepo) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	tok, err := f.ConsumeAuthToken(ctx, tokenHash, database.TokenPurposeVerifyEmail)
	if err != nil {
		return "", err
	}
	if err := f.MarkEmailVerified(ctx, tok.UserID); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (f *fakeRepo) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tok, err := f.ConsumeAuthToken(ctx, tokenHash, database.TokenPurposeResetPassword)
	if err != nil {
		return "", err
	}
	if err := f.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (f *fakeRepo) DeleteLocalAccount(ctx context.Context, userID string) error {
	_ = f.DeleteProfile(ctx, userID)
	return f.DeleteLocalUser(ctx, userID)
}

func (f *fakeRepo) ListFoodPricesByBudget(_ context.Context, _ int, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func (f *fakeRepo) SearchFoodPrices(_ context.Context, _ string, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func (f *fakeRepo) ListAllFoodPrices(_ context.Context, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func (f *fakeRepo) CreateSavedMeal(_ context.Context, p database.CreateSavedMealParams) (database.SavedMeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mealSeq++
	var priceCopy *int
	if p.PriceIDR != nil {
		v := *p.PriceIDR
		priceCopy = &v
	}
	meal := database.SavedMeal{
		ID:          uuid.NewString(),
		UserID:      p.UserID,
		MealDate:    p.MealDate,
		MealTime:    p.MealTime,
		Name:        p.Name,
		HealthScore: p.HealthScore,
		Calories:    p.Calories,
		ProteinG:    p.ProteinG,
		CarbsG:      p.CarbsG,
		FatG:        p.FatG,
		FiberG:      p.FiberG,
		PriceIDR:    priceCopy,
		InputSource: p.InputSource,
		CreatedAt:   time.Now().UTC().Add(time.Duration(f.mealSeq) * time.Millisecond),
	}
	f.savedMeals[meal.ID] = meal
	return meal, nil
}

func (f *fakeRepo) DeleteSavedMeal(_ context.Context, userID, mealID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	meal, ok := f.savedMeals[mealID]
	if !ok || meal.UserID != userID {
		return database.ErrNotFound
	}
	delete(f.savedMeals, mealID)
	return nil
}

func (f *fakeRepo) ListSavedMealsByDate(_ context.Context, userID, date string) ([]database.SavedMeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]database.SavedMeal, 0)
	for _, m := range f.savedMeals {
		if m.UserID == userID && m.MealDate == date {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	return out, nil
}

func (f *fakeRepo) ListMealHistory(_ context.Context, userID, fromDate, toDate string, limit, offset int) ([]database.SavedMeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]database.SavedMeal, 0)
	for _, m := range f.savedMeals {
		if m.UserID == userID && m.MealDate >= fromDate && m.MealDate <= toDate {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MealDate != out[j].MealDate {
			return out[i].MealDate > out[j].MealDate
		}
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.Before(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	if offset >= len(out) {
		return []database.SavedMeal{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (f *fakeRepo) GetWeeklyHealthScores(_ context.Context, userID, fromDate, toDate string) ([]database.DailyHealthScorePoint, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	type agg struct {
		sum   int
		count int
	}
	byDate := make(map[string]*agg)
	for _, m := range f.savedMeals {
		if m.UserID == userID && m.MealDate >= fromDate && m.MealDate <= toDate {
			a, ok := byDate[m.MealDate]
			if !ok {
				a = &agg{}
				byDate[m.MealDate] = a
			}
			a.sum += int(m.HealthScore)
			a.count++
		}
	}
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]database.DailyHealthScorePoint, 0, len(dates))
	for _, d := range dates {
		a := byDate[d]
		out = append(out, database.DailyHealthScorePoint{
			Date:               d,
			AverageHealthScore: int(math.Round(float64(a.sum) / float64(a.count))),
			MealCount:          a.count,
		})
	}
	return out, nil
}

func newSession1TestApp(t *testing.T) (*fiber.App, *fakeRepo) {
	t.Helper()
	cfg := &config.Config{
		Mode:        config.ModeLocal,
		Port:        "8080",
		AppBaseURL:  "http://localhost:3000",
		CORSOrigins: []string{"http://localhost:3000"},
		Auth: config.Auth{
			JWTSecret: testLocalSecret,
			Audience:  "authenticated",
		},
		Limits: config.RateLimit{
			Max:    200,
			Window: time.Minute,
		},
		AuthLimits: config.AuthRateLimit{
			Max:    100,
			Window: time.Minute,
		},
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	repo := newFakeRepo()
	verifier, err := auth.NewVerifier(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	authSvc := auth.NewService(cfg, repo, nil, logger)
	app := New(Dependencies{
		Config:      cfg,
		Logger:      logger,
		DB:          stubPinger{},
		Repo:        repo,
		Auth:        verifier,
		AuthService: authSvc,
	})
	return app, repo
}

func doJSONReq(t *testing.T, app *fiber.App, method, path, bearer string, body any) (*http.Response, map[string]any) {
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
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp, decoded
}

func assertGenericError(t *testing.T, resp *http.Response, body map[string]any, wantStatus int, wantCode string) {
	t.Helper()
	if resp.StatusCode != wantStatus {
		t.Fatalf("expected status %d, got %d (body=%+v)", wantStatus, resp.StatusCode, body)
	}
	if len(body) != 1 || body["code"] != wantCode {
		t.Fatalf("PRD error violation: expected ONLY {code:%q}, got %+v", wantCode, body)
	}
}

func TestSession1AllElevenEndpoints(t *testing.T) {
	app, repo := newSession1TestApp(t)

	// 1. Register weak password -> 400 WEAK_PASSWORD
	resp, body := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "password123",
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "WEAK_PASSWORD")

	// 2. Register valid -> 201 Created
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
		"language": "id",
		"theme":    "light",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%+v)", resp.StatusCode, body)
	}
	userID, _ := body["user_id"].(string)
	devToken, _ := body["dev_verification_token"].(string)
	if userID == "" || devToken == "" {
		t.Fatalf("expected user_id and dev_verification_token, got %+v", body)
	}

	// Duplicate register -> 409 EMAIL_ALREADY_EXISTS
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	assertGenericError(t, resp, body, http.StatusConflict, "EMAIL_ALREADY_EXISTS")

	// 3. Login before verification -> 403 EMAIL_NOT_VERIFIED
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	assertGenericError(t, resp, body, http.StatusForbidden, "EMAIL_NOT_VERIFIED")

	// 4. Verify email -> 200 verified; second use -> 400 INVALID_OR_EXPIRED_TOKEN
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{
		"token": devToken,
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "verified" {
		t.Fatalf("verify-email failed: %d %+v", resp.StatusCode, body)
	}
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{
		"token": devToken,
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN")

	// 5. Login -> 200 with JWT + ProfileDTO (DiceBear URL & onboarding_completed=false)
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "raka@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: %d %+v", resp.StatusCode, body)
	}
	accessToken, _ := body["access_token"].(string)
	profileMap, _ := body["profile"].(map[string]any)
	if accessToken == "" || profileMap["onboarding_completed"] != false {
		t.Fatalf("unexpected login response: %+v", body)
	}
	if profileMap["profile_url"] != auth.DiceBearAvatarURL(userID) {
		t.Fatalf("unexpected DiceBear avatar: %v", profileMap["profile_url"])
	}

	// 6. GET /api/v1/profile -> 401 without token, 200 with token
	resp, body = doJSONReq(t, app, http.MethodGet, "/api/v1/profile", "", nil)
	assertGenericError(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED")

	resp, body = doJSONReq(t, app, http.MethodGet, "/api/v1/profile", accessToken, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /profile failed: %d %+v", resp.StatusCode, body)
	}

	// 7. PUT /api/v1/profile/onboarding -> 400 on invalid input, 200 on valid
	resp, body = doJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", accessToken, map[string]any{
		"height_cm": 50,
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")

	resp, body = doJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", accessToken, map[string]any{
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

	// 8. PATCH /api/v1/profile -> updates preferences + recalculates on anthro change + honors explicit override
	resp, body = doJSONReq(t, app, http.MethodPatch, "/api/v1/profile", accessToken, map[string]any{
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

	resp, body = doJSONReq(t, app, http.MethodPatch, "/api/v1/profile", accessToken, map[string]any{
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

	// 9. POST /api/v1/auth/forgot-password -> always 200 {"status":"ok"}
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "unknown@example.com",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "ok" {
		t.Fatalf("forgot-password unknown failed: %d %+v", resp.StatusCode, body)
	}
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "raka@example.com",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "ok" || len(body) != 1 {
		t.Fatalf("forgot-password known must return identical response, got %d %+v", resp.StatusCode, body)
	}

	// 10. POST /api/v1/auth/reset-password
	rawReset, hashReset, _ := auth.GenerateRandomToken()
	_, _ = repo.CreateAuthToken(context.Background(), database.CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hashReset,
		Purpose:   database.TokenPurposeResetPassword,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "tehtarikhangat",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_reset" {
		t.Fatalf("reset-password failed: %d %+v", resp.StatusCode, body)
	}

	// 11. POST /api/v1/auth/change-password
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/change-password", accessToken, map[string]any{
		"current_password": "wrongpassword",
		"new_password":     "nasigorengspesial",
	})
	assertGenericError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/change-password", accessToken, map[string]any{
		"current_password": "tehtarikhangat",
		"new_password":     "nasigorengspesial",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_changed" {
		t.Fatalf("change-password failed: %d %+v", resp.StatusCode, body)
	}

	// 12. POST /api/v1/auth/logout
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/logout", accessToken, nil)
	if resp.StatusCode != http.StatusOK || body["status"] != "logged_out" {
		t.Fatalf("logout failed: %d %+v", resp.StatusCode, body)
	}

	// 13. DELETE /api/v1/account -> wrong password rejected, correct password deletes permanently
	resp, body = doJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "wrongpassword",
	})
	assertGenericError(t, resp, body, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	resp, body = doJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "nasigorengspesial",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "account_deleted" {
		t.Fatalf("delete account failed: %d %+v", resp.StatusCode, body)
	}

	resp, body = doJSONReq(t, app, http.MethodGet, "/api/v1/profile", accessToken, nil)
	assertGenericError(t, resp, body, http.StatusNotFound, "NOT_FOUND")
}

func TestSession1EdgeCasesAndStrictErrorContract(t *testing.T) {
	app, repo := newSession1TestApp(t)

	// 1. All protected endpoints reject missing Bearer token with 401 UNAUTHORIZED
	protectedCases := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/auth/logout"},
		{http.MethodPost, "/api/v1/auth/change-password"},
		{http.MethodGet, "/api/v1/profile"},
		{http.MethodPut, "/api/v1/profile/onboarding"},
		{http.MethodPatch, "/api/v1/profile"},
		{http.MethodDelete, "/api/v1/account"},
	}
	for _, tc := range protectedCases {
		resp, body := doJSONReq(t, app, tc.method, tc.path, "", nil)
		assertGenericError(t, resp, body, http.StatusUnauthorized, "UNAUTHORIZED")
	}

	// 2. Malformed JSON returns 400 INVALID_INPUT
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader("{bad-json"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("malformed json test: %v", err)
	}
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	assertGenericError(t, resp, decoded, http.StatusBadRequest, "INVALID_INPUT")

	// 3. Forgot password malformed email -> 400 INVALID_INPUT
	resp, body := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/forgot-password", "", map[string]any{
		"email": "not-an-email",
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")

	// Register, verify, and login a user for authenticated edge cases
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    "edge@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register edge user: %d %+v", resp.StatusCode, body)
	}
	userID := body["user_id"].(string)
	devToken := body["dev_verification_token"].(string)

	resp, _ = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{"token": devToken})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("verify edge user: %d", resp.StatusCode)
	}
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    "edge@example.com",
		"password": "kopisusupagi",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login edge user: %d", resp.StatusCode)
	}
	accessToken := body["access_token"].(string)

	// 4. PATCH /api/v1/profile rejects invalid language, theme, and out-of-range targets
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
		resp, body = doJSONReq(t, app, http.MethodPatch, "/api/v1/profile", accessToken, patch)
		assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")
	}

	// 5. Reset password single-use and weak password rejection
	rawReset, hashReset, _ := auth.GenerateRandomToken()
	_, _ = repo.CreateAuthToken(context.Background(), database.CreateAuthTokenParams{
		UserID:    userID,
		TokenHash: hashReset,
		Purpose:   database.TokenPurposeResetPassword,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "password",
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "WEAK_PASSWORD")

	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "nasikuninghangat",
	})
	if resp.StatusCode != http.StatusOK || body["status"] != "password_reset" {
		t.Fatalf("reset password expected 200, got %d %+v", resp.StatusCode, body)
	}
	// Second use of same reset token must fail
	resp, body = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/reset-password", "", map[string]any{
		"token":        rawReset,
		"new_password": "nasikuninghangat2",
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_OR_EXPIRED_TOKEN")

	// 6. DELETE /api/v1/account empty password -> 400 INVALID_INPUT
	resp, body = doJSONReq(t, app, http.MethodDelete, "/api/v1/account", accessToken, map[string]any{
		"password": "",
	})
	assertGenericError(t, resp, body, http.StatusBadRequest, "INVALID_INPUT")
}
