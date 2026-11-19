package server

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/ai"
	"Nouriko/internal/auth"
	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/handler"
	"Nouriko/internal/nutrition"

	"github.com/gofiber/fiber/v3"
)

func newFullE2ETestApp(t *testing.T, analyzer ai.Analyzer, foodLookup nutrition.FoodLookup) (*fiber.App, *fakeRepo) {
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
		Analyzer:    analyzer,
		FoodLookup:  foodLookup,
	})
	return app, repo
}

func TestE2E_FullUserJourney(t *testing.T) {
	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, req ai.AnalyzeRequest) (*ai.RawAIResponse, error) {
			if !strings.Contains(req.SystemInstruction, "English") {
				t.Errorf("expected English language directive in SystemInstruction, got %q", req.SystemInstruction)
			}
			return &ai.RawAIResponse{
				Intent:    ai.IntentBudgetMeals,
				Intro:     "Here are nutritious meals under Rp15,000.",
				BudgetIDR: 15000,
				Meals: []ai.MealOption{
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
	var chatOut ai.ChatAnalyzeResponse
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
		"input_source": database.InputSourceBudget,
	})
	if err != nil {
		t.Fatalf("marshal save meal payload: %v", err)
	}
	saveResp := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", accessToken, string(savePayloadBytes))
	if saveResp.StatusCode != http.StatusCreated {
		t.Fatalf("step 8 save meal: expected 201, got %d", saveResp.StatusCode)
	}
	var saveOut struct {
		Meal handler.SavedMealDTO `json:"meal"`
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
	var summaryOut handler.DailySummaryResponse
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
	var histOut handler.MealHistoryResponse
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
	var summaryAfterDel handler.DailySummaryResponse
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
