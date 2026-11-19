package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/database"
	"Nouriko/internal/handler"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

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

func doAuthedJSON(t *testing.T, app *fiber.App, method, target, token string, body string) *http.Response {
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

func TestSession3_AuthAndOnboardingGateAllEndpoints(t *testing.T) {
	repo := newFakeRepo()
	app := newSession2TestApp(t, repo, nil, nil, 20)

	unboardedID := uuid.NewString()
	_, _ = repo.UpsertProfile(context.Background(), database.UpsertProfileParams{
		ID:       unboardedID,
		Language: "id",
		Theme:    "light",
	})
	unboardedToken := signTestJWT(t, unboardedID, "unboarded@example.com")

	validPayload := `{"name":"Nasi Telur","health_score":75,"calories":500,"protein_g":20,"carbs_g":60,"fat_g":15,"fiber_g":4,"input_source":"ingredients"}`
	sampleUUID := uuid.NewString()

	endpoints := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"POST /meals", http.MethodPost, "/api/v1/meals", validPayload},
		{"DELETE /meals/:id", http.MethodDelete, "/api/v1/meals/" + sampleUUID, ""},
		{"GET /meals/summary", http.MethodGet, "/api/v1/meals/summary", ""},
		{"GET /meals/history", http.MethodGet, "/api/v1/meals/history", ""},
	}

	for _, ep := range endpoints {
		t.Run(ep.name+" unauthenticated", func(t *testing.T) {
			resp := doAuthedJSON(t, app, ep.method, ep.path, "", ep.body)
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
			var errResp map[string]string
			mustReadJSON(t, resp, &errResp)
			if errResp["code"] != "UNAUTHORIZED" {
				t.Fatalf("expected UNAUTHORIZED, got %q", errResp["code"])
			}
		})

		t.Run(ep.name+" onboarding incomplete", func(t *testing.T) {
			resp := doAuthedJSON(t, app, ep.method, ep.path, unboardedToken, ep.body)
			if resp.StatusCode != fiber.StatusForbidden {
				t.Fatalf("expected 403, got %d", resp.StatusCode)
			}
			var errResp map[string]string
			mustReadJSON(t, resp, &errResp)
			if errResp["code"] != "ONBOARDING_REQUIRED" {
				t.Fatalf("expected ONBOARDING_REQUIRED, got %q", errResp["code"])
			}
		})
	}
}

func TestSession3_CreateListDeleteMealAndValidation(t *testing.T) {
	repo := newFakeRepo()
	app := newSession2TestApp(t, repo, nil, nil, 20)
	_, token := seedOnboardedUser(t, repo)

	wibZone := time.FixedZone("WIB", 7*3600)
	todayWIB := time.Now().In(wibZone).Format("2006-01-02")

	// 1. Create meal with omitted meal_date & meal_time -> defaults applied, returns 201 + health_label.
	createPayload := `{
		"name": "  Nasi Telur Dadar + Tumis Kangkung  ",
		"health_score": 78,
		"calories": 520,
		"protein_g": 22.556,
		"carbs_g": 64.0,
		"fat_g": 16.0,
		"fiber_g": 5.5,
		"price_idr": 12000,
		"input_source": "ingredients"
	}`
	resp := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", token, createPayload)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var createRes struct {
		Meal handler.SavedMealDTO `json:"meal"`
	}
	mustReadJSON(t, resp, &createRes)

	if createRes.Meal.ID == "" {
		t.Fatal("expected generated meal ID")
	}
	if createRes.Meal.Name != "Nasi Telur Dadar + Tumis Kangkung" {
		t.Fatalf("expected trimmed name, got %q", createRes.Meal.Name)
	}
	if createRes.Meal.MealDate != todayWIB {
		t.Fatalf("expected default WIB date %s, got %s", todayWIB, createRes.Meal.MealDate)
	}
	if createRes.Meal.MealTime == "" {
		t.Fatal("expected default meal_time to be populated")
	}
	if createRes.Meal.HealthLabel != "good" {
		t.Fatalf("expected health_label good, got %q", createRes.Meal.HealthLabel)
	}
	if createRes.Meal.ProteinG != 22.56 {
		t.Fatalf("expected rounded protein_g 22.56, got %v", createRes.Meal.ProteinG)
	}
	if createRes.Meal.PriceIDR == nil || *createRes.Meal.PriceIDR != 12000 {
		t.Fatalf("expected price_idr 12000, got %v", createRes.Meal.PriceIDR)
	}

	// 2. Payload > 16 KiB -> 413 PAYLOAD_TOO_LARGE.
	oversized := `{"name":"` + strings.Repeat("x", 17*1024) + `","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`
	resp413 := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", token, oversized)
	if resp413.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp413.StatusCode)
	}
	var err413 map[string]string
	mustReadJSON(t, resp413, &err413)
	if err413["code"] != "PAYLOAD_TOO_LARGE" {
		t.Fatalf("expected PAYLOAD_TOO_LARGE, got %q", err413["code"])
	}

	// 3. Invalid payloads -> 400 INVALID_INPUT.
	invalidPayloads := []struct {
		name string
		body string
	}{
		{"malformed json", `{bad`},
		{"empty name", `{"name":"   ","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"health_score > 100", `{"name":"Mie","health_score":101,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"negative health_score", `{"name":"Mie","health_score":-1,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"calories out of range", `{"name":"Mie","health_score":50,"calories":25000,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"negative macro", `{"name":"Mie","health_score":50,"calories":100,"protein_g":-1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"price out of range", `{"name":"Mie","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"price_idr":10000001,"input_source":"budget"}`},
		{"invalid input_source", `{"name":"Mie","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"manual"}`},
		{"invalid meal_time", `{"name":"Mie","meal_time":"brunch","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"invalid meal_date", `{"name":"Mie","meal_date":"2026-13-45","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
		{"meal_date before 2020", `{"name":"Mie","meal_date":"2019-12-31","health_score":50,"calories":100,"protein_g":1,"carbs_g":1,"fat_g":1,"fiber_g":1,"input_source":"budget"}`},
	}

	for _, tc := range invalidPayloads {
		t.Run("invalid "+tc.name, func(t *testing.T) {
			r := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", token, tc.body)
			if r.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d", tc.name, r.StatusCode)
			}
			var e map[string]string
			mustReadJSON(t, r, &e)
			if e["code"] != "INVALID_INPUT" {
				t.Fatalf("expected INVALID_INPUT, got %q", e["code"])
			}
		})
	}

	// 4. DELETE invalid UUID -> 400 INVALID_INPUT.
	respBadID := doAuthedJSON(t, app, http.MethodDelete, "/api/v1/meals/not-a-uuid", token, "")
	if respBadID.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", respBadID.StatusCode)
	}

	// 5. DELETE created meal -> 200 {"deleted": true}.
	respDel := doAuthedJSON(t, app, http.MethodDelete, "/api/v1/meals/"+createRes.Meal.ID, token, "")
	if respDel.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", respDel.StatusCode)
	}
	var delBody map[string]bool
	mustReadJSON(t, respDel, &delBody)
	if !delBody["deleted"] {
		t.Fatal("expected deleted: true")
	}

	// 6. Deleting again -> 404 NOT_FOUND.
	respDel2 := doAuthedJSON(t, app, http.MethodDelete, "/api/v1/meals/"+createRes.Meal.ID, token, "")
	if respDel2.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", respDel2.StatusCode)
	}
}

func TestSession3_IDORProtectionAcrossAllEndpoints(t *testing.T) {
	repo := newFakeRepo()
	app := newSession2TestApp(t, repo, nil, nil, 20)

	_, tokenA := seedOnboardedUser(t, repo)
	_, tokenB := seedOnboardedUser(t, repo)

	targetDate := "2026-11-18"
	payload := `{
		"meal_date": "2026-11-18",
		"meal_time": "lunch",
		"name": "Ayam Bakar User A",
		"health_score": 82,
		"calories": 600,
		"protein_g": 38,
		"carbs_g": 55,
		"fat_g": 18,
		"fiber_g": 6,
		"price_idr": 18000,
		"input_source": "budget"
	}`

	respCreate := doAuthedJSON(t, app, http.MethodPost, "/api/v1/meals", tokenA, payload)
	if respCreate.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", respCreate.StatusCode)
	}
	var created struct {
		Meal handler.SavedMealDTO `json:"meal"`
	}
	mustReadJSON(t, respCreate, &created)

	// User B attempts to delete User A's meal -> 404 NOT_FOUND (never 403 existence leak).
	respIDORDel := doAuthedJSON(t, app, http.MethodDelete, "/api/v1/meals/"+created.Meal.ID, tokenB, "")
	if respIDORDel.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for cross-user delete, got %d", respIDORDel.StatusCode)
	}
	var errIDOR map[string]string
	mustReadJSON(t, respIDORDel, &errIDOR)
	if errIDOR["code"] != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", errIDOR["code"])
	}

	// User B summary must NOT contain User A's meal.
	respSumB := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date="+targetDate, tokenB, "")
	if respSumB.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for User B summary, got %d", respSumB.StatusCode)
	}
	var sumB handler.DailySummaryResponse
	mustReadJSON(t, respSumB, &sumB)
	if len(sumB.Meals) != 0 || sumB.Totals.MealCount != 0 {
		t.Fatalf("expected User B summary to have 0 meals, got %d", len(sumB.Meals))
	}

	// User B history must NOT contain User A's meal.
	respHistB := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-11-12&to=2026-11-18", tokenB, "")
	if respHistB.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for User B history, got %d", respHistB.StatusCode)
	}
	var histB handler.MealHistoryResponse
	mustReadJSON(t, respHistB, &histB)
	if len(histB.Days) != 0 {
		t.Fatalf("expected User B history to have 0 days, got %d", len(histB.Days))
	}

	// User A's meal must still exist intact.
	respSumA := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date="+targetDate, tokenA, "")
	var sumA handler.DailySummaryResponse
	mustReadJSON(t, respSumA, &sumA)
	if len(sumA.Meals) != 1 || sumA.Meals[0].ID != created.Meal.ID {
		t.Fatalf("expected User A's meal to remain intact, got %+v", sumA.Meals)
	}
}

func TestSession3_DailySummaryAggregationEmptyDayAndZeroTargetSafety(t *testing.T) {
	repo := newFakeRepo()
	app := newSession2TestApp(t, repo, nil, nil, 20)
	userID, token := seedOnboardedUser(t, repo)

	// 1. Empty day -> meals: [] (not null), meal_count: 0, average_health_score: 0, health_label: null.
	respEmpty := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-10", token, "")
	if respEmpty.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", respEmpty.StatusCode)
	}
	var sumEmpty handler.DailySummaryResponse
	rawEmpty := mustReadJSON(t, respEmpty, &sumEmpty)
	if !strings.Contains(rawEmpty, `"meals":[]`) {
		t.Fatalf("expected empty JSON array meals:[], got %s", rawEmpty)
	}
	if !strings.Contains(rawEmpty, `"health_label":null`) {
		t.Fatalf("expected health_label:null on empty day, got %s", rawEmpty)
	}
	if sumEmpty.Totals.MealCount != 0 || sumEmpty.Totals.AverageHealthScore != 0 || sumEmpty.Totals.HealthLabel != nil {
		t.Fatalf("unexpected empty totals: %+v", sumEmpty.Totals)
	}

	// 2. Invalid date query param -> 400 INVALID_INPUT.
	respBadDate := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date=19-11-2026", token, "")
	if respBadDate.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid date, got %d", respBadDate.StatusCode)
	}

	// 3. Seed 2 meals on 2026-11-18 (one with price, one with nil price).
	p1 := 15000
	_, _ = repo.CreateSavedMeal(context.Background(), database.CreateSavedMealParams{
		UserID: userID, MealDate: "2026-11-18", MealTime: "breakfast",
		Name: "Bubur Ayam", HealthScore: 65, Calories: 450,
		ProteinG: 18.2, CarbsG: 62.5, FatG: 12.1, FiberG: 3.4,
		PriceIDR: &p1, InputSource: "budget",
	})
	_, _ = repo.CreateSavedMeal(context.Background(), database.CreateSavedMealParams{
		UserID: userID, MealDate: "2026-11-18", MealTime: "dinner",
		Name: "Tumis Tempe + Nasi", HealthScore: 84, Calories: 650,
		ProteinG: 36.8, CarbsG: 75.0, FatG: 24.55, FiberG: 12.0,
		PriceIDR: nil, InputSource: "ingredients",
	})

	respSum := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-18", token, "")
	if respSum.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", respSum.StatusCode)
	}
	var sum handler.DailySummaryResponse
	mustReadJSON(t, respSum, &sum)

	if sum.Totals.MealCount != 2 {
		t.Fatalf("expected meal_count 2, got %d", sum.Totals.MealCount)
	}
	if sum.Totals.Calories != 1100 {
		t.Fatalf("expected calories 1100, got %d", sum.Totals.Calories)
	}
	if sum.Totals.ProteinG != 55.0 || sum.Totals.CarbsG != 137.5 || sum.Totals.FatG != 36.65 || sum.Totals.FiberG != 15.4 {
		t.Fatalf("unexpected macro totals: %+v", sum.Totals)
	}
	if sum.Totals.TotalSpendingIDR != 15000 {
		t.Fatalf("expected total_spending_idr 15000, got %d", sum.Totals.TotalSpendingIDR)
	}
	// (65 + 84) / 2 = 74.5 -> rounds to 75 -> "good"
	if sum.Totals.AverageHealthScore != 75 || sum.Totals.HealthLabel == nil || *sum.Totals.HealthLabel != "good" {
		t.Fatalf("expected avg 75 (good), got %+v", sum.Totals)
	}
	// Targets from seedOnboardedUser: 2200 kcal, 110 P, 275 C, 73.3 F, 30.8 Fiber -> all ~50%
	if sum.ProgressPercent.Calories != 50 || sum.ProgressPercent.ProteinG != 50 || sum.ProgressPercent.CarbsG != 50 || sum.ProgressPercent.FatG != 50 || sum.ProgressPercent.FiberG != 50 {
		t.Fatalf("expected all progress_percent == 50, got %+v", sum.ProgressPercent)
	}

	// 4. Division-by-zero safety: set user's macro targets to 0 and verify no panic / 0%.
	zeroInt := 0
	zeroFloat := 0.0
	_, _ = repo.UpdateProfile(context.Background(), database.UpdateProfileParams{
		UserID:             userID,
		DailyCalorieTarget: &zeroInt,
		TargetProteinG:     &zeroFloat,
		TargetCarbsG:       &zeroFloat,
		TargetFatG:         &zeroFloat,
		TargetFiberG:       &zeroFloat,
	})
	respZero := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-18", token, "")
	if respZero.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 with zero targets, got %d", respZero.StatusCode)
	}
	var sumZero handler.DailySummaryResponse
	mustReadJSON(t, respZero, &sumZero)
	if sumZero.ProgressPercent.Calories != 0 || sumZero.ProgressPercent.ProteinG != 0 || sumZero.ProgressPercent.CarbsG != 0 || sumZero.ProgressPercent.FatG != 0 || sumZero.ProgressPercent.FiberG != 0 {
		t.Fatalf("expected 0 progress_percent on zero targets, got %+v", sumZero.ProgressPercent)
	}
}

func TestSession3_MealHistoryGroupingRangeValidationAndWeeklyTrend(t *testing.T) {
	repo := newFakeRepo()
	app := newSession2TestApp(t, repo, nil, nil, 20)
	userID, token := seedOnboardedUser(t, repo)

	// Seed meals on 2026-11-14 (2 meals), 2026-11-17 (1 meal), 2026-11-19 (2 meals).
	// Leaves gaps on 2026-11-13, 2026-11-15, 2026-11-16, 2026-11-18 inside 7-day trend ending 2026-11-19.
	p10 := 10000
	p14 := 14000
	seedMeals := []database.CreateSavedMealParams{
		{UserID: userID, MealDate: "2026-11-14", MealTime: "lunch", Name: "Nasi Padang", HealthScore: 60, Calories: 700, ProteinG: 25, CarbsG: 80, FatG: 25, FiberG: 4, PriceIDR: &p14, InputSource: "budget"},
		{UserID: userID, MealDate: "2026-11-14", MealTime: "dinner", Name: "Gado-gado", HealthScore: 76, Calories: 450, ProteinG: 18, CarbsG: 50, FatG: 18, FiberG: 8, PriceIDR: &p10, InputSource: "photo"},
		{UserID: userID, MealDate: "2026-11-17", MealTime: "breakfast", Name: "Oatmeal Pisang", HealthScore: 88, Calories: 320, ProteinG: 12, CarbsG: 55, FatG: 6, FiberG: 9, PriceIDR: nil, InputSource: "ingredients"},
		{UserID: userID, MealDate: "2026-11-19", MealTime: "lunch", Name: "Pecel Lele", HealthScore: 70, Calories: 600, ProteinG: 30, CarbsG: 65, FatG: 20, FiberG: 5, PriceIDR: &p14, InputSource: "budget"},
		{UserID: userID, MealDate: "2026-11-19", MealTime: "snack", Name: "Gorengan", HealthScore: 30, Calories: 350, ProteinG: 5, CarbsG: 40, FatG: 18, FiberG: 2, PriceIDR: &p10, InputSource: "photo"},
	}
	for _, m := range seedMeals {
		if _, err := repo.CreateSavedMeal(context.Background(), m); err != nil {
			t.Fatalf("seed meal: %v", err)
		}
	}

	// 1. Valid range query (from/to within 90 days).
	resp := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-11-01&to=2026-11-19", token, "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var hist handler.MealHistoryResponse
	mustReadJSON(t, resp, &hist)

	if hist.From != "2026-11-01" || hist.To != "2026-11-19" {
		t.Fatalf("unexpected from/to: %s .. %s", hist.From, hist.To)
	}
	if len(hist.Days) != 3 {
		t.Fatalf("expected 3 grouped days, got %d", len(hist.Days))
	}
	// Ordered date DESC: 2026-11-19, 2026-11-17, 2026-11-14.
	if hist.Days[0].Date != "2026-11-19" || hist.Days[1].Date != "2026-11-17" || hist.Days[2].Date != "2026-11-14" {
		t.Fatalf("expected date DESC ordering, got %s, %s, %s", hist.Days[0].Date, hist.Days[1].Date, hist.Days[2].Date)
	}
	if hist.Days[0].MealCount != 2 || hist.Days[0].TotalCalories != 950 || hist.Days[0].TotalSpendingIDR != 24000 || hist.Days[0].AverageHealthScore != 50 || hist.Days[0].HealthLabel != "fair" {
		t.Fatalf("unexpected day[0] summary: %+v", hist.Days[0])
	}

	// Weekly trend must be EXACTLY 7 contiguous days ending 2026-11-19 in ASC order.
	if len(hist.WeeklyTrend) != 7 {
		t.Fatalf("expected 7 weekly_trend points, got %d", len(hist.WeeklyTrend))
	}
	expectedTrend := []struct {
		date  string
		score *int
		count int
	}{
		{"2026-11-13", nil, 0},
		{"2026-11-14", intPtrTest(68), 2},
		{"2026-11-15", nil, 0},
		{"2026-11-16", nil, 0},
		{"2026-11-17", intPtrTest(88), 1},
		{"2026-11-18", nil, 0},
		{"2026-11-19", intPtrTest(50), 2},
	}
	for i, exp := range expectedTrend {
		got := hist.WeeklyTrend[i]
		if got.Date != exp.date || got.MealCount != exp.count {
			t.Fatalf("trend[%d]: want (%s, count=%d), got (%s, count=%d)", i, exp.date, exp.count, got.Date, got.MealCount)
		}
		if exp.score == nil && got.AverageHealthScore != nil {
			t.Fatalf("trend[%d] (%s): expected nil score, got %d", i, exp.date, *got.AverageHealthScore)
		}
		if exp.score != nil && (got.AverageHealthScore == nil || *got.AverageHealthScore != *exp.score) {
			t.Fatalf("trend[%d] (%s): expected score %d, got %v", i, exp.date, *exp.score, got.AverageHealthScore)
		}
	}

	// 2. Range validation edge cases.
	badQueries := []string{
		"/api/v1/meals/history?from=2026-11-20&to=2026-11-19", // from > to
		"/api/v1/meals/history?from=2026-08-21&to=2026-11-19", // 91 days (> 90)
		"/api/v1/meals/history?days=0",
		"/api/v1/meals/history?days=91",
		"/api/v1/meals/history?days=abc",
		"/api/v1/meals/history?from=invalid",
		"/api/v1/meals/history?to=2026-99-99",
	}
	for _, q := range badQueries {
		r := doAuthedJSON(t, app, http.MethodGet, q, token, "")
		if r.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", q, r.StatusCode)
		}
	}

	// 3. Exact 90-day span is allowed (2026-08-22 to 2026-11-19 = 90 calendar days inclusive).
	resp90 := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-08-22&to=2026-11-19", token, "")
	if resp90.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for exact 90-day span, got %d", resp90.StatusCode)
	}

	// 4. Empty range returns days: [] (not null) and 7 null trend entries.
	respEmpty := doAuthedJSON(t, app, http.MethodGet, "/api/v1/meals/history?from=2025-01-01&to=2025-01-07", token, "")
	if respEmpty.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for empty history, got %d", respEmpty.StatusCode)
	}
	var emptyHist handler.MealHistoryResponse
	raw := mustReadJSON(t, respEmpty, &emptyHist)
	if !bytes.Contains([]byte(raw), []byte(`"days":[]`)) {
		t.Fatalf("expected days:[] JSON array, got %s", raw)
	}
	if len(emptyHist.WeeklyTrend) != 7 {
		t.Fatalf("expected 7 trend entries on empty history, got %d", len(emptyHist.WeeklyTrend))
	}
	for i, pt := range emptyHist.WeeklyTrend {
		if pt.AverageHealthScore != nil || pt.MealCount != 0 {
			t.Fatalf("expected empty trend[%d], got %+v", i, pt)
		}
	}
}

func intPtrTest(v int) *int {
	return &v
}
