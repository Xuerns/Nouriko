package meals

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

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func TestDefaultMealTimeWIB(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, MealTimeSnack},
		{4, MealTimeSnack},
		{5, MealTimeBreakfast},
		{10, MealTimeBreakfast},
		{11, MealTimeLunch},
		{14, MealTimeLunch},
		{15, MealTimeSnack},
		{17, MealTimeSnack},
		{18, MealTimeDinner},
		{21, MealTimeDinner},
		{22, MealTimeSnack},
		{23, MealTimeSnack},
	}

	for _, tc := range tests {
		ts := time.Date(2026, 11, 19, tc.hour, 30, 0, 0, wibZone)
		got := DefaultMealTimeWIB(ts)
		if got != tc.want {
			t.Errorf("hour %02d WIB: want %q, got %q", tc.hour, tc.want, got)
		}
	}
}

func TestMealHealthLabel(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{0, "low"},
		{39, "low"},
		{40, "fair"},
		{69, "fair"},
		{70, "good"},
		{100, "good"},
	}
	for _, tc := range cases {
		if got := MealHealthLabel(tc.score); got != tc.want {
			t.Errorf("score %d: want %q, got %q", tc.score, tc.want, got)
		}
	}
}

func TestValidateCreateMealRequest_NaNInfAndBounds(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, wibZone)
	intPtr := func(v int) *int { return &v }
	floatPtr := func(v float64) *float64 { return &v }

	baseReq := func() *createMealRequest {
		return &createMealRequest{
			Name:        "Nasi Telur Dadar",
			HealthScore: intPtr(75),
			Calories:    intPtr(500),
			ProteinG:    floatPtr(22.5),
			CarbsG:      floatPtr(60.0),
			FatG:        floatPtr(15.0),
			FiberG:      floatPtr(4.5),
			InputSource: InputSourceIngredients,
		}
	}

	// Valid defaults
	params, ok := validateCreateMealRequest("user-1", baseReq(), now)
	if !ok {
		t.Fatal("expected valid request")
	}
	if params.MealDate != "2026-11-19" {
		t.Fatalf("expected default WIB date 2026-11-19, got %q", params.MealDate)
	}
	if params.MealTime != MealTimeLunch {
		t.Fatalf("expected default WIB lunch, got %q", params.MealTime)
	}

	// NaN macro
	nanReq := baseReq()
	nanReq.ProteinG = floatPtr(math.NaN())
	if _, ok := validateCreateMealRequest("user-1", nanReq, now); ok {
		t.Fatal("expected NaN protein_g to be rejected")
	}

	// +Inf macro
	posInfReq := baseReq()
	posInfReq.CarbsG = floatPtr(math.Inf(1))
	if _, ok := validateCreateMealRequest("user-1", posInfReq, now); ok {
		t.Fatal("expected +Inf carbs_g to be rejected")
	}

	// -Inf macro
	negInfReq := baseReq()
	negInfReq.FatG = floatPtr(math.Inf(-1))
	if _, ok := validateCreateMealRequest("user-1", negInfReq, now); ok {
		t.Fatal("expected -Inf fat_g to be rejected")
	}

	// > maxMacroG
	overMacro := baseReq()
	overMacro.FiberG = floatPtr(10000.0)
	if _, ok := validateCreateMealRequest("user-1", overMacro, now); ok {
		t.Fatal("expected fiber_g > 9999.99 to be rejected")
	}

	// Name > 200 runes
	longName := baseReq()
	longName.Name = strings.Repeat("a", 201)
	if _, ok := validateCreateMealRequest("user-1", longName, now); ok {
		t.Fatal("expected name > 200 runes to be rejected")
	}

	// Past date (< todayWIB) must be rejected
	pastDate := baseReq()
	pastDate.MealDate = "2026-11-18"
	if _, ok := validateCreateMealRequest("user-1", pastDate, now); ok {
		t.Fatal("expected meal_date < today to be rejected")
	}

	// Far future date (> todayWIB + maxFutureMealDays)
	futureDate := baseReq()
	futureDate.MealDate = "2026-12-20"
	if _, ok := validateCreateMealRequest("user-1", futureDate, now); ok {
		t.Fatal("expected meal_date > today+30d to be rejected")
	}

	// Today + 1 day (tomorrow) and today + 30 days are allowed for meal planning
	tomorrowDate := baseReq()
	tomorrowDate.MealDate = "2026-11-20"
	if _, ok := validateCreateMealRequest("user-1", tomorrowDate, now); !ok {
		t.Fatal("expected meal_date == today+1d to be allowed")
	}

	maxFutureDate := baseReq()
	maxFutureDate.MealDate = "2026-12-19"
	if _, ok := validateCreateMealRequest("user-1", maxFutureDate, now); !ok {
		t.Fatal("expected meal_date == today+30d to be allowed")
	}
}

func TestCalcProgressPercent_DivisionByZeroSafety(t *testing.T) {
	if got := calcProgressPercent(500, 0); got != 0 {
		t.Fatalf("expected 0 for zero target, got %d", got)
	}
	if got := calcProgressPercent(500, -10); got != 0 {
		t.Fatalf("expected 0 for negative target, got %d", got)
	}
	if got := calcProgressPercent(0, 2000); got != 0 {
		t.Fatalf("expected 0 for zero actual, got %d", got)
	}
	if got := calcProgressPercent(520, 2100); got != 25 {
		t.Fatalf("expected 25, got %d", got)
	}
}

func TestIsValidUUID(t *testing.T) {
	if !isValidUUID("b7c0e9f0-1234-4abc-8def-0123456789ab") {
		t.Fatal("expected canonical 36-char UUID to be valid")
	}
	for _, bad := range []string{
		"",
		"not-a-uuid",
		"b7c0e9f012344abc8def0123456789ab", // 32-char without hyphens
		"urn:uuid:b7c0e9f0-1234-4abc-8def-0123456789ab",
	} {
		if isValidUUID(bad) {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}

// --- HTTP handler integration tests (migrated from server/session3_test.go) ---

type memMealsTestStore struct {
	mu         sync.Mutex
	profiles   map[string]*profile.Profile
	savedMeals map[string]SavedMeal
	mealSeq    int64
}

func newMemMealsTestStore() *memMealsTestStore {
	return &memMealsTestStore{
		profiles:   make(map[string]*profile.Profile),
		savedMeals: make(map[string]SavedMeal),
	}
}

func (m *memMealsTestStore) seedProfile(p *profile.Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *p
	m.profiles[p.ID] = &cp
}

func (m *memMealsTestStore) GetProfileByID(_ context.Context, userID string) (*profile.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func (m *memMealsTestStore) CreateSavedMeal(_ context.Context, p CreateSavedMealParams) (SavedMeal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mealSeq++
	var priceCopy *int
	if p.PriceIDR != nil {
		v := *p.PriceIDR
		priceCopy = &v
	}
	meal := SavedMeal{
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
		CreatedAt:   time.Now().UTC().Add(time.Duration(m.mealSeq) * time.Millisecond),
	}
	m.savedMeals[meal.ID] = meal
	return meal, nil
}

func (m *memMealsTestStore) DeleteSavedMeal(_ context.Context, userID, mealID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	meal, ok := m.savedMeals[mealID]
	if !ok || meal.UserID != userID {
		return database.ErrNotFound
	}
	delete(m.savedMeals, mealID)
	return nil
}

func (m *memMealsTestStore) ListSavedMealsByDate(_ context.Context, userID, date string) ([]SavedMeal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SavedMeal, 0)
	for _, item := range m.savedMeals {
		if item.UserID == userID && item.MealDate == date {
			out = append(out, item)
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

func (m *memMealsTestStore) ListMealHistory(_ context.Context, userID, fromDate, toDate string, limit, offset int) ([]SavedMeal, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 500
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]SavedMeal, 0)
	for _, item := range m.savedMeals {
		if item.UserID == userID && item.MealDate >= fromDate && item.MealDate <= toDate {
			out = append(out, item)
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
		return []SavedMeal{}, nil
	}
	end := offset + limit
	if end > len(out) {
		end = len(out)
	}
	return out[offset:end], nil
}

func (m *memMealsTestStore) GetWeeklyHealthScores(_ context.Context, userID, fromDate, toDate string) ([]DailyHealthScorePoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	type agg struct {
		sum   int
		count int
	}
	byDate := make(map[string]*agg)
	for _, item := range m.savedMeals {
		if item.UserID == userID && item.MealDate >= fromDate && item.MealDate <= toDate {
			a, ok := byDate[item.MealDate]
			if !ok {
				a = &agg{}
				byDate[item.MealDate] = a
			}
			a.sum += int(item.HealthScore)
			a.count++
		}
	}
	dates := make([]string, 0, len(byDate))
	for d := range byDate {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]DailyHealthScorePoint, 0, len(dates))
	for _, d := range dates {
		a := byDate[d]
		out = append(out, DailyHealthScorePoint{
			Date:               d,
			AverageHealthScore: int(math.Round(float64(a.sum) / float64(a.count))),
			MealCount:          a.count,
		})
	}
	return out, nil
}

func seedOnboardedMealsUser(store *memMealsTestStore) string {
	uid := uuid.NewString()
	h, w, age, gender, activity, kcal := 170.0, 65.0, int16(21), "male", "moderate", 2200
	prot, carbs, fat, fiber := 110.0, 275.0, 73.3, 30.8
	store.seedProfile(&profile.Profile{
		ID:                 uid,
		HeightCM:           &h,
		WeightKG:           &w,
		Age:                &age,
		Gender:             &gender,
		ActivityLevel:      &activity,
		DailyCalorieTarget: &kcal,
		TargetProteinG:     &prot,
		TargetCarbsG:       &carbs,
		TargetFatG:         &fat,
		TargetFiberG:       &fiber,
		Language:           "id",
		Theme:              "light",
	})
	return uid
}

func newMealsHandlerTestApp(store *memMealsTestStore) *fiber.App {
	app := fiber.New()
	h := NewHandler(store, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	injectAuth := func(c fiber.Ctx) error {
		tok := httpx.ExtractBearerToken(c)
		if tok != "" {
			httpx.SetClaims(c, httpx.Claims{UserID: tok, Email: "user@example.com", Role: "authenticated"})
		}
		return c.Next()
	}
	app.Post("/api/v1/meals", injectAuth, h.CreateMeal)
	app.Get("/api/v1/meals/summary", injectAuth, h.GetDailySummary)
	app.Get("/api/v1/meals/history", injectAuth, h.GetMealHistory)
	app.Delete("/api/v1/meals/:id", injectAuth, h.DeleteMeal)
	return app
}

func mustReadMealJSON(t *testing.T, resp *http.Response, dst any) string {
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

func doMealReq(t *testing.T, app *fiber.App, method, target, token, body string) *http.Response {
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
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test %s %s: %v", method, target, err)
	}
	return resp
}

func TestMealHandler_AuthAndOnboardingGateAllEndpoints(t *testing.T) {
	store := newMemMealsTestStore()
	app := newMealsHandlerTestApp(store)

	unboardedID := uuid.NewString()
	store.seedProfile(&profile.Profile{ID: unboardedID, Language: "id", Theme: "light"})

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
			resp := doMealReq(t, app, ep.method, ep.path, "", ep.body)
			if resp.StatusCode != fiber.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", resp.StatusCode)
			}
			var errResp map[string]string
			mustReadMealJSON(t, resp, &errResp)
			if errResp["code"] != "UNAUTHORIZED" {
				t.Fatalf("expected UNAUTHORIZED, got %q", errResp["code"])
			}
		})

		t.Run(ep.name+" onboarding incomplete", func(t *testing.T) {
			resp := doMealReq(t, app, ep.method, ep.path, unboardedID, ep.body)
			if resp.StatusCode != fiber.StatusForbidden {
				t.Fatalf("expected 403, got %d", resp.StatusCode)
			}
			var errResp map[string]string
			mustReadMealJSON(t, resp, &errResp)
			if errResp["code"] != "ONBOARDING_REQUIRED" {
				t.Fatalf("expected ONBOARDING_REQUIRED, got %q", errResp["code"])
			}
		})
	}
}

func TestMealHandler_CreateListDeleteMealAndValidation(t *testing.T) {
	store := newMemMealsTestStore()
	app := newMealsHandlerTestApp(store)
	uid := seedOnboardedMealsUser(store)

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
	resp := doMealReq(t, app, http.MethodPost, "/api/v1/meals", uid, createPayload)
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}
	var createRes struct {
		Meal SavedMealDTO `json:"meal"`
	}
	mustReadMealJSON(t, resp, &createRes)

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
	resp413 := doMealReq(t, app, http.MethodPost, "/api/v1/meals", uid, oversized)
	if resp413.StatusCode != fiber.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", resp413.StatusCode)
	}
	var err413 map[string]string
	mustReadMealJSON(t, resp413, &err413)
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
			r := doMealReq(t, app, http.MethodPost, "/api/v1/meals", uid, tc.body)
			if r.StatusCode != fiber.StatusBadRequest {
				t.Fatalf("expected 400 for %s, got %d", tc.name, r.StatusCode)
			}
			var e map[string]string
			mustReadMealJSON(t, r, &e)
			if e["code"] != "INVALID_INPUT" {
				t.Fatalf("expected INVALID_INPUT, got %q", e["code"])
			}
		})
	}

	// 4. DELETE invalid UUID -> 400 INVALID_INPUT.
	respBadID := doMealReq(t, app, http.MethodDelete, "/api/v1/meals/not-a-uuid", uid, "")
	if respBadID.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid UUID, got %d", respBadID.StatusCode)
	}

	// 5. DELETE created meal -> 200 {"deleted": true}.
	respDel := doMealReq(t, app, http.MethodDelete, "/api/v1/meals/"+createRes.Meal.ID, uid, "")
	if respDel.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", respDel.StatusCode)
	}
	var delBody map[string]bool
	mustReadMealJSON(t, respDel, &delBody)
	if !delBody["deleted"] {
		t.Fatal("expected deleted: true")
	}

	// 6. Deleting again -> 404 NOT_FOUND.
	respDel2 := doMealReq(t, app, http.MethodDelete, "/api/v1/meals/"+createRes.Meal.ID, uid, "")
	if respDel2.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 on second delete, got %d", respDel2.StatusCode)
	}
}

func TestMealHandler_IDORProtectionAcrossAllEndpoints(t *testing.T) {
	store := newMemMealsTestStore()
	app := newMealsHandlerTestApp(store)

	uidA := seedOnboardedMealsUser(store)
	uidB := seedOnboardedMealsUser(store)

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

	respCreate := doMealReq(t, app, http.MethodPost, "/api/v1/meals", uidA, payload)
	if respCreate.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", respCreate.StatusCode)
	}
	var created struct {
		Meal SavedMealDTO `json:"meal"`
	}
	mustReadMealJSON(t, respCreate, &created)

	// User B attempts to delete User A's meal -> 404 NOT_FOUND (never 403 existence leak).
	respIDORDel := doMealReq(t, app, http.MethodDelete, "/api/v1/meals/"+created.Meal.ID, uidB, "")
	if respIDORDel.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404 for cross-user delete, got %d", respIDORDel.StatusCode)
	}
	var errIDOR map[string]string
	mustReadMealJSON(t, respIDORDel, &errIDOR)
	if errIDOR["code"] != "NOT_FOUND" {
		t.Fatalf("expected NOT_FOUND, got %q", errIDOR["code"])
	}

	// User B summary must NOT contain User A's meal.
	respSumB := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date="+targetDate, uidB, "")
	if respSumB.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for User B summary, got %d", respSumB.StatusCode)
	}
	var sumB DailySummaryResponse
	mustReadMealJSON(t, respSumB, &sumB)
	if len(sumB.Meals) != 0 || sumB.Totals.MealCount != 0 {
		t.Fatalf("expected User B summary to have 0 meals, got %d", len(sumB.Meals))
	}

	// User B history must NOT contain User A's meal.
	respHistB := doMealReq(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-11-12&to=2026-11-18", uidB, "")
	if respHistB.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for User B history, got %d", respHistB.StatusCode)
	}
	var histB MealHistoryResponse
	mustReadMealJSON(t, respHistB, &histB)
	if len(histB.Days) != 0 {
		t.Fatalf("expected User B history to have 0 days, got %d", len(histB.Days))
	}

	// User A's meal must still exist intact.
	respSumA := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date="+targetDate, uidA, "")
	var sumA DailySummaryResponse
	mustReadMealJSON(t, respSumA, &sumA)
	if len(sumA.Meals) != 1 || sumA.Meals[0].ID != created.Meal.ID {
		t.Fatalf("expected User A's meal to remain intact, got %+v", sumA.Meals)
	}
}

func TestMealHandler_DailySummaryAggregationEmptyDayAndZeroTargetSafety(t *testing.T) {
	store := newMemMealsTestStore()
	app := newMealsHandlerTestApp(store)
	userID := seedOnboardedMealsUser(store)

	// 1. Empty day -> meals: [] (not null), meal_count: 0, average_health_score: 0, health_label: null.
	respEmpty := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-10", userID, "")
	if respEmpty.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", respEmpty.StatusCode)
	}
	var sumEmpty DailySummaryResponse
	rawEmpty := mustReadMealJSON(t, respEmpty, &sumEmpty)
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
	respBadDate := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date=19-11-2026", userID, "")
	if respBadDate.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid date, got %d", respBadDate.StatusCode)
	}

	// 3. Seed 2 meals on 2026-11-18 (one with price, one with nil price).
	p1 := 15000
	_, _ = store.CreateSavedMeal(context.Background(), CreateSavedMealParams{
		UserID: userID, MealDate: "2026-11-18", MealTime: "breakfast",
		Name: "Bubur Ayam", HealthScore: 65, Calories: 450,
		ProteinG: 18.2, CarbsG: 62.5, FatG: 12.1, FiberG: 3.4,
		PriceIDR: &p1, InputSource: "budget",
	})
	_, _ = store.CreateSavedMeal(context.Background(), CreateSavedMealParams{
		UserID: userID, MealDate: "2026-11-18", MealTime: "dinner",
		Name: "Tumis Tempe + Nasi", HealthScore: 84, Calories: 650,
		ProteinG: 36.8, CarbsG: 75.0, FatG: 24.55, FiberG: 12.0,
		PriceIDR: nil, InputSource: "ingredients",
	})

	respSum := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-18", userID, "")
	if respSum.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", respSum.StatusCode)
	}
	var sum DailySummaryResponse
	mustReadMealJSON(t, respSum, &sum)

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
	// Targets from seedOnboardedMealsUser: 2200 kcal, 110 P, 275 C, 73.3 F, 30.8 Fiber -> all ~50%
	if sum.ProgressPercent.Calories != 50 || sum.ProgressPercent.ProteinG != 50 || sum.ProgressPercent.CarbsG != 50 || sum.ProgressPercent.FatG != 50 || sum.ProgressPercent.FiberG != 50 {
		t.Fatalf("expected all progress_percent == 50, got %+v", sum.ProgressPercent)
	}

	// 4. Division-by-zero safety: set user's macro targets to 0 and verify no panic / 0%.
	zeroInt := 0
	zeroFloat := 0.0
	prof, _ := store.GetProfileByID(context.Background(), userID)
	prof.DailyCalorieTarget = &zeroInt
	prof.TargetProteinG = &zeroFloat
	prof.TargetCarbsG = &zeroFloat
	prof.TargetFatG = &zeroFloat
	prof.TargetFiberG = &zeroFloat
	store.seedProfile(prof)

	respZero := doMealReq(t, app, http.MethodGet, "/api/v1/meals/summary?date=2026-11-18", userID, "")
	if respZero.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 with zero targets, got %d", respZero.StatusCode)
	}
	var sumZero DailySummaryResponse
	mustReadMealJSON(t, respZero, &sumZero)
	if sumZero.ProgressPercent.Calories != 0 || sumZero.ProgressPercent.ProteinG != 0 || sumZero.ProgressPercent.CarbsG != 0 || sumZero.ProgressPercent.FatG != 0 || sumZero.ProgressPercent.FiberG != 0 {
		t.Fatalf("expected 0 progress_percent on zero targets, got %+v", sumZero.ProgressPercent)
	}
}

func TestMealHandler_MealHistoryGroupingRangeValidationAndWeeklyTrend(t *testing.T) {
	store := newMemMealsTestStore()
	app := newMealsHandlerTestApp(store)
	userID := seedOnboardedMealsUser(store)

	p10 := 10000
	p14 := 14000
	seedMeals := []CreateSavedMealParams{
		{UserID: userID, MealDate: "2026-11-14", MealTime: "lunch", Name: "Nasi Padang", HealthScore: 60, Calories: 700, ProteinG: 25, CarbsG: 80, FatG: 25, FiberG: 4, PriceIDR: &p14, InputSource: "budget"},
		{UserID: userID, MealDate: "2026-11-14", MealTime: "dinner", Name: "Gado-gado", HealthScore: 76, Calories: 450, ProteinG: 18, CarbsG: 50, FatG: 18, FiberG: 8, PriceIDR: &p10, InputSource: "photo"},
		{UserID: userID, MealDate: "2026-11-17", MealTime: "breakfast", Name: "Oatmeal Pisang", HealthScore: 88, Calories: 320, ProteinG: 12, CarbsG: 55, FatG: 6, FiberG: 9, PriceIDR: nil, InputSource: "ingredients"},
		{UserID: userID, MealDate: "2026-11-19", MealTime: "lunch", Name: "Pecel Lele", HealthScore: 70, Calories: 600, ProteinG: 30, CarbsG: 65, FatG: 20, FiberG: 5, PriceIDR: &p14, InputSource: "budget"},
		{UserID: userID, MealDate: "2026-11-19", MealTime: "snack", Name: "Gorengan", HealthScore: 30, Calories: 350, ProteinG: 5, CarbsG: 40, FatG: 18, FiberG: 2, PriceIDR: &p10, InputSource: "photo"},
	}
	for _, m := range seedMeals {
		if _, err := store.CreateSavedMeal(context.Background(), m); err != nil {
			t.Fatalf("seed meal: %v", err)
		}
	}

	// 1. Valid range query (from/to within 90 days).
	resp := doMealReq(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-11-01&to=2026-11-19", userID, "")
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var hist MealHistoryResponse
	mustReadMealJSON(t, resp, &hist)

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
	intPtr := func(v int) *int { return &v }
	expectedTrend := []struct {
		date  string
		score *int
		count int
	}{
		{"2026-11-13", nil, 0},
		{"2026-11-14", intPtr(68), 2},
		{"2026-11-15", nil, 0},
		{"2026-11-16", nil, 0},
		{"2026-11-17", intPtr(88), 1},
		{"2026-11-18", nil, 0},
		{"2026-11-19", intPtr(50), 2},
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
		r := doMealReq(t, app, http.MethodGet, q, userID, "")
		if r.StatusCode != fiber.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", q, r.StatusCode)
		}
	}

	// 3. Exact 90-day span is allowed (2026-08-22 to 2026-11-19 = 90 calendar days inclusive).
	resp90 := doMealReq(t, app, http.MethodGet, "/api/v1/meals/history?from=2026-08-22&to=2026-11-19", userID, "")
	if resp90.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for exact 90-day span, got %d", resp90.StatusCode)
	}

	// 4. Empty range returns days: [] (not null) and 7 null trend entries.
	respEmpty := doMealReq(t, app, http.MethodGet, "/api/v1/meals/history?from=2025-01-01&to=2025-01-07", userID, "")
	if respEmpty.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for empty history, got %d", respEmpty.StatusCode)
	}
	var emptyHist MealHistoryResponse
	raw := mustReadMealJSON(t, respEmpty, &emptyHist)
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
