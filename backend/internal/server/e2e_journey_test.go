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
	"Nouriko/internal/chat"
	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/meals"
	"Nouriko/internal/nutrition"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testLocalSecret = "0123456789abcdef0123456789abcdef"

// e2eMemStore implements all domain repositories in memory for full-journey E2E tests.
type e2eMemStore struct {
	mu         sync.Mutex
	users      map[string]*auth.LocalUser
	tokens     map[string]*auth.LocalAuthToken
	profiles   map[string]*profile.Profile
	savedMeals map[string]meals.SavedMeal
	mealSeq    int64
}

func newE2EMemStore() *e2eMemStore {
	return &e2eMemStore{
		users:      make(map[string]*auth.LocalUser),
		tokens:     make(map[string]*auth.LocalAuthToken),
		profiles:   make(map[string]*profile.Profile),
		savedMeals: make(map[string]meals.SavedMeal),
	}
}

func (f *e2eMemStore) CreateLocalUser(_ context.Context, p auth.CreateLocalUserParams) (*auth.LocalUser, error) {
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
	u := &auth.LocalUser{ID: id, Email: email, PasswordHash: p.PasswordHash, CreatedAt: now, UpdatedAt: now}
	f.users[id] = u
	cp := *u
	return &cp, nil
}

func (f *e2eMemStore) GetLocalUserByEmail(_ context.Context, email string) (*auth.LocalUser, error) {
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

func (f *e2eMemStore) GetLocalUserByID(_ context.Context, userID string) (*auth.LocalUser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *e2eMemStore) RecordFailedLogin(_ context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*auth.LocalUser, error) {
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

func (f *e2eMemStore) ResetLoginLockout(_ context.Context, userID string) error {
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

func (f *e2eMemStore) MarkEmailVerified(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.EmailVerified = true
	return nil
}

func (f *e2eMemStore) UpdatePasswordHash(_ context.Context, userID, passwordHash string, resetLockout bool) error {
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

func (f *e2eMemStore) DeleteLocalUser(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.users[userID]; !ok {
		return database.ErrNotFound
	}
	delete(f.users, userID)
	return nil
}

func (f *e2eMemStore) CreateAuthToken(_ context.Context, p auth.CreateAuthTokenParams) (*auth.LocalAuthToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	tok := &auth.LocalAuthToken{ID: uuid.NewString(), UserID: p.UserID, TokenHash: p.TokenHash, Purpose: p.Purpose, ExpiresAt: p.ExpiresAt, CreatedAt: time.Now()}
	f.tokens[p.TokenHash] = tok
	cp := *tok
	return &cp, nil
}

func (f *e2eMemStore) InvalidateUserTokens(_ context.Context, userID, purpose string) error {
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

func (f *e2eMemStore) ConsumeAuthToken(_ context.Context, tokenHash, purpose string) (*auth.LocalAuthToken, error) {
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

func (f *e2eMemStore) DeleteExpiredTokens(_ context.Context) (int64, error) { return 0, nil }

func (f *e2eMemStore) UpsertProfile(_ context.Context, p profile.UpsertProfileParams) (*profile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.profiles[p.ID]; ok {
		cp := *existing
		return &cp, nil
	}
	now := time.Now()
	prof := &profile.Profile{ID: p.ID, ProfileURL: p.ProfileURL, Language: p.Language, Theme: p.Theme, CreatedAt: now, UpdatedAt: now}
	f.profiles[p.ID] = prof
	cp := *prof
	return &cp, nil
}

func (f *e2eMemStore) GetProfileByID(_ context.Context, userID string) (*profile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prof, ok := f.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *prof
	return &cp, nil
}

func (f *e2eMemStore) UpdateOnboarding(_ context.Context, p profile.UpdateOnboardingParams) (*profile.Profile, error) {
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

func (f *e2eMemStore) UpdateProfile(_ context.Context, p profile.UpdateProfileParams) (*profile.Profile, error) {
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

func (f *e2eMemStore) DeleteProfile(_ context.Context, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.profiles[userID]; !ok {
		return database.ErrNotFound
	}
	delete(f.profiles, userID)
	return nil
}

func (f *e2eMemStore) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	tok, err := f.ConsumeAuthToken(ctx, tokenHash, auth.TokenPurposeVerifyEmail)
	if err != nil {
		return "", err
	}
	if err := f.MarkEmailVerified(ctx, tok.UserID); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (f *e2eMemStore) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tok, err := f.ConsumeAuthToken(ctx, tokenHash, auth.TokenPurposeResetPassword)
	if err != nil {
		return "", err
	}
	if err := f.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (f *e2eMemStore) DeleteLocalAccount(ctx context.Context, userID string) error {
	f.mu.Lock()
	for id, meal := range f.savedMeals {
		if meal.UserID == userID {
			delete(f.savedMeals, id)
		}
	}
	for hash, tok := range f.tokens {
		if tok.UserID == userID {
			delete(f.tokens, hash)
		}
	}
	f.mu.Unlock()
	_ = f.DeleteProfile(ctx, userID)
	return f.DeleteLocalUser(ctx, userID)
}

func (f *e2eMemStore) ListFoodPricesByBudget(_ context.Context, _ int, _ string, _ int) ([]chat.FoodPrice, error) {
	return []chat.FoodPrice{}, nil
}

func (f *e2eMemStore) SearchFoodPrices(_ context.Context, _ string, _ string, _ int) ([]chat.FoodPrice, error) {
	return []chat.FoodPrice{}, nil
}

func (f *e2eMemStore) ListAllFoodPrices(_ context.Context, _ string, _ int) ([]chat.FoodPrice, error) {
	return []chat.FoodPrice{}, nil
}

func (f *e2eMemStore) CreateSavedMeal(_ context.Context, p meals.CreateSavedMealParams) (meals.SavedMeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mealSeq++
	var priceCopy *int
	if p.PriceIDR != nil {
		v := *p.PriceIDR
		priceCopy = &v
	}
	meal := meals.SavedMeal{
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

func (f *e2eMemStore) DeleteSavedMeal(_ context.Context, userID, mealID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	meal, ok := f.savedMeals[mealID]
	if !ok || meal.UserID != userID {
		return database.ErrNotFound
	}
	delete(f.savedMeals, mealID)
	return nil
}

func (f *e2eMemStore) ListSavedMealsByDate(_ context.Context, userID, date string) ([]meals.SavedMeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]meals.SavedMeal, 0)
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

func (f *e2eMemStore) ListMealHistory(_ context.Context, userID, fromDate, toDate string, limit, offset int) ([]meals.SavedMeal, error) {
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
	out := make([]meals.SavedMeal, 0)
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
		return []meals.SavedMeal{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (f *e2eMemStore) GetWeeklyHealthScores(_ context.Context, userID, fromDate, toDate string) ([]meals.DailyHealthScorePoint, error) {
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
	out := make([]meals.DailyHealthScorePoint, 0, len(dates))
	for _, d := range dates {
		a := byDate[d]
		out = append(out, meals.DailyHealthScorePoint{
			Date:               d,
			AverageHealthScore: int(math.Round(float64(a.sum) / float64(a.count))),
			MealCount:          a.count,
		})
	}
	return out, nil
}

type mockAnalyzer struct {
	fn func(ctx context.Context, req chat.AnalyzeRequest) (*chat.RawAIResponse, error)
}

func (m *mockAnalyzer) Analyze(ctx context.Context, req chat.AnalyzeRequest) (*chat.RawAIResponse, error) {
	return m.fn(ctx, req)
}

type mockFoodLookup struct {
	fn func(ctx context.Context, name string) (*nutrition.MacroData, bool, error)
}

func (m *mockFoodLookup) Lookup(ctx context.Context, name string) (*nutrition.MacroData, bool, error) {
	return m.fn(ctx, name)
}

func signTestJWT(t *testing.T, userID, email string) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"role":  "authenticated",
		"aud":   "authenticated",
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(testLocalSecret))
	if err != nil {
		t.Fatalf("signTestJWT: %v", err)
	}
	return signed
}

func seedOnboardedUser(t *testing.T, repo *e2eMemStore) (string, string) {
	t.Helper()
	uid := uuid.NewString()
	email := "kos+" + uid[:8] + "@example.com"
	_, err := repo.UpsertProfile(context.Background(), profile.UpsertProfileParams{
		ID:         uid,
		ProfileURL: auth.DiceBearAvatarURL(uid),
		Language:   "id",
		Theme:      "light",
	})
	if err != nil {
		t.Fatalf("UpsertProfile: %v", err)
	}
	_, err = repo.UpdateOnboarding(context.Background(), profile.UpdateOnboardingParams{
		UserID:             uid,
		HeightCM:           170,
		WeightKG:           65,
		Age:                21,
		Gender:             "male",
		ActivityLevel:      "moderate",
		DailyCalorieTarget: 2200,
		TargetProteinG:     110,
		TargetCarbsG:       275,
		TargetFatG:         73.3,
		TargetFiberG:       30.8,
	})
	if err != nil {
		t.Fatalf("UpdateOnboarding: %v", err)
	}
	return uid, signTestJWT(t, uid, email)
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

func mustReadJSON(t *testing.T, resp *http.Response, dst any) string {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if dst != nil {
		if err := json.Unmarshal(body, dst); err != nil {
			t.Fatalf("unmarshal response (%s): %v", string(body), err)
		}
	}
	return string(body)
}

func doAuthedJSON(t *testing.T, app *fiber.App, method, target, token, body string) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, target, err)
	}
	return resp
}

func newFullE2ETestApp(t *testing.T, analyzer chat.Analyzer, foodLookup nutrition.FoodLookup) (*fiber.App, *e2eMemStore) {
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
		AILimits: config.AIRateLimit{
			Max:    50,
			Window: time.Minute,
		},
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	repo := newE2EMemStore()
	verifier, err := auth.NewVerifier(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	authSvc := auth.NewService(cfg, repo, repo, nil, logger)
	app := New(Dependencies{
		Config:        cfg,
		Logger:        logger,
		DB:            stubPinger{},
		ProfileRepo:   repo,
		Auth:          verifier,
		AuthService:   authSvc,
		FoodPriceRepo: repo,
		Analyzer:      analyzer,
		FoodLookup:    foodLookup,
		MealRepo:      repo,
	})
	return app, repo
}

func TestE2E_FullUserJourney(t *testing.T) {
	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, req chat.AnalyzeRequest) (*chat.RawAIResponse, error) {
			if !strings.Contains(req.SystemInstruction, "English") {
				t.Errorf("expected English language directive in SystemInstruction, got %q", req.SystemInstruction)
			}
			return &chat.RawAIResponse{
				Intent:    chat.IntentBudgetMeals,
				Intro:     "Here are nutritious meals under Rp15,000.",
				BudgetIDR: 15000,
				Meals: []chat.MealOption{
					{
						Name:              "Rice with Omelette and Tempe",
						Portion:           "1 plate",
						HealthScore:       76,
						Calories:          510,
						ProteinG:          24.5,
						CarbsG:            62.0,
						FatG:              16.0,
						FiberG:            5.2,
						EstimatedPriceIDR: 12000,
						IsPriceEstimated:  false,
						Note:              "Balanced protein and fiber for a budget meal.",
					},
					{
						Name:              "Grilled Salmon Bento",
						Portion:           "1 box",
						HealthScore:       88,
						Calories:          600,
						ProteinG:          38.0,
						CarbsG:            55.0,
						FatG:              20.0,
						FiberG:            6.0,
						EstimatedPriceIDR: 22000,
						IsPriceEstimated:  true,
						Note:              "Exceeds budget and must be filtered out.",
					},
				},
			}, nil
		},
	}

	foodLookup := &mockFoodLookup{
		fn: func(_ context.Context, _ string) (*nutrition.MacroData, bool, error) {
			return nil, false, nil
		},
	}

	app, repo := newFullE2ETestApp(t, analyzer, foodLookup)

	const (
		email       = "raka.e2e@example.com"
		initialPass = "nasigorenghangat"
		updatedPass = "tempeorekmantap9"
	)

	// 1. Register (POST /api/v1/auth/register)
	resp, regBody := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    email,
		"password": initialPass,
		"language": "id",
		"theme":    "light",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("step 1 register: expected 201, got %d (%+v)", resp.StatusCode, regBody)
	}
	userID, _ := regBody["user_id"].(string)
	verifyToken, _ := regBody["dev_verification_token"].(string)
	if userID == "" || verifyToken == "" {
		t.Fatalf("step 1 register: missing user_id or dev_verification_token: %+v", regBody)
	}
	expectedAvatar := auth.DiceBearAvatarURL(userID)
	profInRepo, err := repo.GetProfileByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("step 1 profile lookup: %v", err)
	}
	if profInRepo.ProfileURL != expectedAvatar {
		t.Fatalf("step 1 DiceBear profile_url: want %q, got %q", expectedAvatar, profInRepo.ProfileURL)
	}

	// 2. Verify Email (POST /api/v1/auth/verify-email)
	resp, verifyBody := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/verify-email", "", map[string]any{
		"token": verifyToken,
	})
	if resp.StatusCode != http.StatusOK || verifyBody["status"] != "verified" {
		t.Fatalf("step 2 verify-email: expected 200 verified, got %d (%+v)", resp.StatusCode, verifyBody)
	}

	// 3. Login (POST /api/v1/auth/login) -> onboarding_completed == false
	resp, loginBody := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": initialPass,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 3 login: expected 200, got %d (%+v)", resp.StatusCode, loginBody)
	}
	accessToken, _ := loginBody["access_token"].(string)
	loginProfile, _ := loginBody["profile"].(map[string]any)
	if accessToken == "" || loginProfile["onboarding_completed"] != false {
		t.Fatalf("step 3 login: expected access_token and onboarding_completed=false, got %+v", loginBody)
	}
	if loginProfile["profile_url"] != expectedAvatar {
		t.Fatalf("step 3 login: expected profile_url %q, got %v", expectedAvatar, loginProfile["profile_url"])
	}

	// 4. Access /api/v1/chat/analyze and /api/v1/meals before onboarding -> 403 ONBOARDING_REQUIRED
	resp, errBody := doJSONReq(t, app, http.MethodPost, "/api/v1/chat/analyze", accessToken, map[string]any{
		"message": "budget 15000",
	})
	assertGenericError(t, resp, errBody, http.StatusForbidden, "ONBOARDING_REQUIRED")

	resp, errBody = doJSONReq(t, app, http.MethodPost, "/api/v1/meals", accessToken, map[string]any{
		"name":         "Nasi Telur",
		"health_score": 70,
		"calories":     450,
		"protein_g":    18,
		"carbs_g":      55,
		"fat_g":        14,
		"fiber_g":      3,
		"input_source": "budget",
	})
	assertGenericError(t, resp, errBody, http.StatusForbidden, "ONBOARDING_REQUIRED")

	resp, errBody = doJSONReq(t, app, http.MethodGet, "/api/v1/meals/summary", accessToken, nil)
	assertGenericError(t, resp, errBody, http.StatusForbidden, "ONBOARDING_REQUIRED")

	resp, errBody = doJSONReq(t, app, http.MethodGet, "/api/v1/meals/history", accessToken, nil)
	assertGenericError(t, resp, errBody, http.StatusForbidden, "ONBOARDING_REQUIRED")

	// 5. Onboarding (PUT /api/v1/profile/onboarding) -> verify Mifflin-St Jeor targets
	expectedTargets, err := nutrition.CalculateTargets(nutrition.Anthropometrics{
		HeightCM:      170,
		WeightKG:      65,
		Age:           20,
		Gender:        "male",
		ActivityLevel: "moderate",
	})
	if err != nil {
		t.Fatalf("CalculateTargets: %v", err)
	}
	resp, onbBody := doJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", accessToken, map[string]any{
		"height_cm":      170,
		"weight_kg":      65,
		"age":            20,
		"gender":         "male",
		"activity_level": "moderate",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 5 onboarding: expected 200, got %d (%+v)", resp.StatusCode, onbBody)
	}
	onbProfile, _ := onbBody["profile"].(map[string]any)
	if onbProfile["onboarding_completed"] != true {
		t.Fatalf("step 5 onboarding: expected onboarding_completed=true, got %+v", onbProfile)
	}
	if int(onbProfile["daily_calorie_target"].(float64)) != expectedTargets.DailyCalorieTarget {
		t.Fatalf("step 5 calorie target: want %d, got %v", expectedTargets.DailyCalorieTarget, onbProfile["daily_calorie_target"])
	}
	if onbProfile["target_protein_g"].(float64) != expectedTargets.TargetProteinG {
		t.Fatalf("step 5 protein target: want %v, got %v", expectedTargets.TargetProteinG, onbProfile["target_protein_g"])
	}
	if onbProfile["target_carbs_g"].(float64) != expectedTargets.TargetCarbsG {
		t.Fatalf("step 5 carbs target: want %v, got %v", expectedTargets.TargetCarbsG, onbProfile["target_carbs_g"])
	}
	if onbProfile["target_fat_g"].(float64) != expectedTargets.TargetFatG {
		t.Fatalf("step 5 fat target: want %v, got %v", expectedTargets.TargetFatG, onbProfile["target_fat_g"])
	}
	if onbProfile["target_fiber_g"].(float64) != expectedTargets.TargetFiberG {
		t.Fatalf("step 5 fiber target: want %v, got %v", expectedTargets.TargetFiberG, onbProfile["target_fiber_g"])
	}

	// 6. Update language and theme (PATCH /api/v1/profile)
	resp, patchBody := doJSONReq(t, app, http.MethodPatch, "/api/v1/profile", accessToken, map[string]any{
		"language": "en",
		"theme":    "dark",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 6 patch profile: expected 200, got %d (%+v)", resp.StatusCode, patchBody)
	}
	patchedProfile, _ := patchBody["profile"].(map[string]any)
	if patchedProfile["language"] != "en" || patchedProfile["theme"] != "dark" {
		t.Fatalf("step 6 patch profile: expected en/dark, got %+v", patchedProfile)
	}

	// 7. Chat analyze (POST /api/v1/chat/analyze) -> verify meal card and 0-over-budget filter
	chatResp := doAuthedJSON(t, app, http.MethodPost, "/api/v1/chat/analyze", accessToken, `{"message":"I have 15000 rupiah for lunch"}`)
	if chatResp.StatusCode != http.StatusOK {
		t.Fatalf("step 7 chat analyze: expected 200, got %d", chatResp.StatusCode)
	}
	var chatOut chat.ChatAnalyzeResponse
	mustReadJSON(t, chatResp, &chatOut)
	if len(chatOut.Meals) != 1 {
		t.Fatalf("step 7 chat analyze: expected 1 within-budget meal (over-budget filtered), got %d (%+v)", len(chatOut.Meals), chatOut.Meals)
	}
	selectedMeal := chatOut.Meals[0]
	if selectedMeal.Name != "Rice with Omelette and Tempe" {
		t.Fatalf("step 7 meal name: got %q", selectedMeal.Name)
	}
	if selectedMeal.HealthLabel != "good" {
		t.Fatalf("step 7 health_label: want good, got %q", selectedMeal.HealthLabel)
	}
	if selectedMeal.EstimatedPriceIDR != 12000 {
		t.Fatalf("step 7 estimated_price_idr: want 12000, got %d", selectedMeal.EstimatedPriceIDR)
	}

	// 8. Save meal -> Daily summary -> Meal history & 7-day trend
	savePayloadBytes, err := json.Marshal(map[string]any{
		"meal_time":    "lunch",
		"name":         selectedMeal.Name,
		"health_score": selectedMeal.HealthScore,
		"calories":     selectedMeal.Calories,
		"protein_g":    selectedMeal.ProteinG,
		"carbs_g":      selectedMeal.CarbsG,
		"fat_g":        selectedMeal.FatG,
		"fiber_g":      selectedMeal.FiberG,
		"price_idr":    selectedMeal.EstimatedPriceIDR,
		"input_source": meals.InputSourceBudget,
	})
	if err != nil {
		t.Fatalf("marshal save meal payload: %v", err)
	}
	saveResp := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", accessToken, string(savePayloadBytes))
	if saveResp.StatusCode != http.StatusCreated {
		t.Fatalf("step 8 save meal: expected 201, got %d", saveResp.StatusCode)
	}
	var saveOut struct {
		Meal meals.SavedMealDTO `json:"meal"`
	}
	mustReadJSON(t, saveResp, &saveOut)
	savedMealID := saveOut.Meal.ID
	if savedMealID == "" {
		t.Fatal("step 8 save meal: missing meal id")
	}

	summaryResp := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary", accessToken, "")
	if summaryResp.StatusCode != http.StatusOK {
		t.Fatalf("step 8 summary: expected 200, got %d", summaryResp.StatusCode)
	}
	var summaryOut meals.DailySummaryResponse
	mustReadJSON(t, summaryResp, &summaryOut)
	if summaryOut.Totals.MealCount != 1 {
		t.Fatalf("step 8 summary meal_count: want 1, got %d", summaryOut.Totals.MealCount)
	}
	if summaryOut.Totals.Calories != 510 || summaryOut.Totals.TotalSpendingIDR != 12000 {
		t.Fatalf("step 8 summary totals mismatch: %+v", summaryOut.Totals)
	}
	if summaryOut.Totals.HealthLabel == nil || *summaryOut.Totals.HealthLabel != "good" {
		t.Fatalf("step 8 summary health_label: want good, got %v", summaryOut.Totals.HealthLabel)
	}
	if summaryOut.Targets.DailyCalorieTarget != expectedTargets.DailyCalorieTarget {
		t.Fatalf("step 8 summary calorie target mismatch: want %d, got %d", expectedTargets.DailyCalorieTarget, summaryOut.Targets.DailyCalorieTarget)
	}

	histResp := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?days=7", accessToken, "")
	if histResp.StatusCode != http.StatusOK {
		t.Fatalf("step 8 history: expected 200, got %d", histResp.StatusCode)
	}
	var histOut meals.MealHistoryResponse
	mustReadJSON(t, histResp, &histOut)
	if len(histOut.Days) != 1 || histOut.Days[0].TotalSpendingIDR != 12000 {
		t.Fatalf("step 8 history days mismatch: %+v", histOut.Days)
	}
	if len(histOut.WeeklyTrend) != 7 {
		t.Fatalf("step 8 weekly_trend: want 7 points, got %d", len(histOut.WeeklyTrend))
	}
	lastTrend := histOut.WeeklyTrend[6]
	if lastTrend.MealCount != 1 || lastTrend.AverageHealthScore == nil || *lastTrend.AverageHealthScore != 76 {
		t.Fatalf("step 8 weekly_trend today mismatch: %+v", lastTrend)
	}

	// 9. Delete meal -> verify empty summary
	delResp := doAuthedJSON(t, app, http.MethodDelete, "/api/v1/meals/"+savedMealID, accessToken, "")
	if delResp.StatusCode != http.StatusOK {
		t.Fatalf("step 9 delete meal: expected 200, got %d", delResp.StatusCode)
	}
	summaryAfterDelResp := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary", accessToken, "")
	if summaryAfterDelResp.StatusCode != http.StatusOK {
		t.Fatalf("step 9 summary after delete: expected 200, got %d", summaryAfterDelResp.StatusCode)
	}
	var summaryAfterDel meals.DailySummaryResponse
	mustReadJSON(t, summaryAfterDelResp, &summaryAfterDel)
	if summaryAfterDel.Totals.MealCount != 0 || summaryAfterDel.Totals.Calories != 0 || summaryAfterDel.Totals.TotalSpendingIDR != 0 || summaryAfterDel.Totals.HealthLabel != nil {
		t.Fatalf("step 9 expected empty summary after delete, got %+v", summaryAfterDel.Totals)
	}

	// Save one meal again to confirm cascade cleanup when account is deleted.
	saveResp2 := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", accessToken, string(savePayloadBytes))
	if saveResp2.StatusCode != http.StatusCreated {
		t.Fatalf("step 9b re-save meal: expected 201, got %d", saveResp2.StatusCode)
	}

	// 10. Change password -> Delete account -> verify relogin fails and all data deleted
	resp, cpBody := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/change-password", accessToken, map[string]any{
		"current_password": initialPass,
		"new_password":     updatedPass,
	})
	if resp.StatusCode != http.StatusOK || cpBody["status"] != "password_changed" {
		t.Fatalf("step 10 change-password: expected 200 password_changed, got %d (%+v)", resp.StatusCode, cpBody)
	}

	// Old password must no longer work.
	resp, errBody = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": initialPass,
	})
	assertGenericError(t, resp, errBody, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	// New password works.
	resp, reloginBody := doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": updatedPass,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("step 10 relogin with new password: expected 200, got %d (%+v)", resp.StatusCode, reloginBody)
	}
	newToken, _ := reloginBody["access_token"].(string)

	// Delete account with new password.
	resp, delAccBody := doJSONReq(t, app, http.MethodDelete, "/api/v1/account", newToken, map[string]any{
		"password": updatedPass,
	})
	if resp.StatusCode != http.StatusOK || delAccBody["status"] != "account_deleted" {
		t.Fatalf("step 10 delete account: expected 200 account_deleted, got %d (%+v)", resp.StatusCode, delAccBody)
	}

	// Login after deletion must fail.
	resp, errBody = doJSONReq(t, app, http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": updatedPass,
	})
	assertGenericError(t, resp, errBody, http.StatusUnauthorized, "INVALID_CREDENTIALS")

	// Verify profile, user, and saved meals are completely removed.
	if _, err := repo.GetProfileByID(context.Background(), userID); err == nil {
		t.Fatal("step 10 expected profile to be deleted")
	}
	if _, err := repo.GetLocalUserByID(context.Background(), userID); err == nil {
		t.Fatal("step 10 expected local user to be deleted")
	}
	remainingMeals, err := repo.ListMealHistory(context.Background(), userID, "2020-01-01", "2099-12-31", 100, 0)
	if err != nil {
		t.Fatalf("step 10 list remaining meals: %v", err)
	}
	if len(remainingMeals) != 0 {
		t.Fatalf("step 10 expected 0 remaining saved meals after account delete, got %d", len(remainingMeals))
	}

	// Regression [LOW-01]: pre-deletion JWT must NOT resurrect profile via PUT /api/v1/profile/onboarding.
	resp, errBody = doJSONReq(t, app, http.MethodPut, "/api/v1/profile/onboarding", newToken, map[string]any{
		"height_cm":      170,
		"weight_kg":      65,
		"age":            20,
		"gender":         "male",
		"activity_level": "moderate",
	})
	assertGenericError(t, resp, errBody, http.StatusNotFound, "NOT_FOUND")
	if _, err := repo.GetProfileByID(context.Background(), userID); err == nil {
		t.Fatal("step 10 regression: expected profile to remain deleted after onboarding attempt with stale JWT")
	}
}

func TestMeals_RegressionPre2020DatesRejected(t *testing.T) {
	app, repo := newFullE2ETestApp(t, nil, nil)
	_, token := seedOnboardedUser(t, repo)

	rejectedEndpoints := []struct {
		name string
		path string
	}{
		{"summary year 0000", "/api/v1/meals/summary?date=0000-01-01"},
		{"summary year 0001", "/api/v1/meals/summary?date=0001-01-01"},
		{"summary pre-2020 boundary", "/api/v1/meals/summary?date=2019-12-31"},
		{"history to year 0000", "/api/v1/meals/history?to=0000-01-01"},
		{"history to year 0001", "/api/v1/meals/history?to=0001-01-01"},
		{"history to pre-2020 boundary", "/api/v1/meals/history?to=2019-12-31"},
		{"history from year 0001", "/api/v1/meals/history?from=0001-01-01&to=2020-01-05"},
		{"history from pre-2020 boundary", "/api/v1/meals/history?from=2019-12-31&to=2020-01-05"},
	}

	for _, tc := range rejectedEndpoints {
		t.Run(tc.name, func(t *testing.T) {
			resp := doAuthedJSON(t, app, http.MethodGet, tc.path, token, "")
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d", tc.path, resp.StatusCode)
			}
			var errBody map[string]string
			mustReadJSON(t, resp, &errBody)
			if errBody["code"] != "INVALID_INPUT" {
				t.Fatalf("expected INVALID_INPUT for %s, got %q", tc.path, errBody["code"])
			}
		})
	}

	// Verify 2020-01-01 lower bound itself is accepted.
	okSummary := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date=2020-01-01", token, "")
	if okSummary.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for 2020-01-01 summary, got %d", okSummary.StatusCode)
	}
	okHistory := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?from=2020-01-01&to=2020-01-07", token, "")
	if okHistory.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for 2020-01-01..2020-01-07 history, got %d", okHistory.StatusCode)
	}
}
