package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"
	"Nouriko/internal/nutrition"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
	"google.golang.org/genai"
)

type mockAnalyzer struct {
	fn func(ctx context.Context, req AnalyzeRequest) (*RawAIResponse, error)
}

func (m *mockAnalyzer) Analyze(ctx context.Context, req AnalyzeRequest) (*RawAIResponse, error) {
	return m.fn(ctx, req)
}

type mockFoodLookup struct {
	fn func(ctx context.Context, name string) (*nutrition.MacroData, bool, error)
}

func (m *mockFoodLookup) Lookup(ctx context.Context, name string) (*nutrition.MacroData, bool, error) {
	return m.fn(ctx, name)
}

type memChatProfileStore struct {
	mu       sync.Mutex
	profiles map[string]*profile.Profile
}

func newMemChatProfileStore() *memChatProfileStore {
	return &memChatProfileStore{profiles: make(map[string]*profile.Profile)}
}

func (m *memChatProfileStore) seed(p *profile.Profile) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *p
	m.profiles[p.ID] = &cp
}

func (m *memChatProfileStore) GetProfileByID(_ context.Context, userID string) (*profile.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *p
	return &cp, nil
}

func seedOnboardedChatUser(store *memChatProfileStore) string {
	uid := uuid.NewString()
	h, w, age, gender, activity, kcal := 170.0, 65.0, int16(21), "male", "moderate", 2200
	prot, carbs, fat, fiber := 110.0, 275.0, 73.3, 30.8
	store.seed(&profile.Profile{
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

func newChatHandlerTestApp(profiles ProfileGetter, analyzer Analyzer, foodLookup nutrition.FoodLookup, aiLimitMax int) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 10 * 1024 * 1024,
	})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := NewChatService(analyzer, &stubFoodPriceStore{}, foodLookup, logger)
	h := NewHandler(profiles, svc, logger)

	injectAuth := func(c fiber.Ctx) error {
		tok := httpx.ExtractBearerToken(c)
		if tok != "" {
			httpx.SetClaims(c, httpx.Claims{UserID: tok, Email: "user@example.com", Role: "authenticated"})
		}
		return c.Next()
	}

	aiUserLimiter := limiter.New(limiter.Config{
		Max:        aiLimitMax,
		Expiration: time.Minute,
		KeyGenerator: func(c fiber.Ctx) string {
			if claims, ok := httpx.ClaimsFromContext(c); ok && claims.UserID != "" {
				return "ai:" + claims.UserID
			}
			return "ai:ip:" + c.IP()
		},
		LimitReached: func(c fiber.Ctx) error {
			return httpx.WriteError(c, fiber.StatusTooManyRequests, httpx.CodeTooManyRequests)
		},
	})

	app.Post("/api/v1/chat/analyze", injectAuth, aiUserLimiter, h.Analyze)
	return app
}

func TestChatAnalyze_AuthOnboardingAndUnavailableGuards(t *testing.T) {
	store := newMemChatProfileStore()
	app := newChatHandlerTestApp(store, nil, nil, 20)

	// 1. Unauthenticated -> 401 UNAUTHORIZED.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"budget 15rb"}`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	// 2. Authenticated without onboarding -> 403 ONBOARDING_REQUIRED.
	unboardedID := uuid.NewString()
	store.seed(&profile.Profile{ID: unboardedID, Language: "id", Theme: "light"})

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"budget 15rb"}`))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+unboardedID)
	resp2, err := app.Test(req2, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp2.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp2.StatusCode)
	}
	var errBody map[string]string
	_ = json.NewDecoder(resp2.Body).Decode(&errBody)
	if errBody["code"] != "ONBOARDING_REQUIRED" {
		t.Fatalf("expected ONBOARDING_REQUIRED, got %q", errBody["code"])
	}

	// 3. Onboarded user but Analyzer is nil -> 503 AI_UNAVAILABLE.
	uid := seedOnboardedChatUser(store)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"budget 15rb"}`))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+uid)
	resp3, err := app.Test(req3, fiber.TestConfig{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp3.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", resp3.StatusCode)
	}
	errBody = nil
	_ = json.NewDecoder(resp3.Body).Decode(&errBody)
	if errBody["code"] != "AI_UNAVAILABLE" {
		t.Fatalf("expected AI_UNAVAILABLE, got %q", errBody["code"])
	}
}

func TestChatAnalyze_JSONSuccessAndStrictValidation(t *testing.T) {
	store := newMemChatProfileStore()
	uid := seedOnboardedChatUser(store)

	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, _ AnalyzeRequest) (*RawAIResponse, error) {
			return &RawAIResponse{
				Intent: IntentBudgetMeals,
				Intro:  "Ini pilihan menu hemat di bawah Rp15.000.",
				Meals: []MealOption{
					{
						Name:              "Nasi Telor Dadar",
						Portion:           "1 porsi",
						HealthScore:       68,
						Calories:          450,
						ProteinG:          18,
						CarbsG:            58,
						FatG:              14,
						FiberG:            3,
						EstimatedPriceIDR: 10000,
						Note:              "Protein murah dan praktis.",
					},
					{
						Name:              "Nasi Rendang Mahal",
						Portion:           "1 porsi",
						HealthScore:       62,
						EstimatedPriceIDR: 22000, // exceeds 15000 -> must be filtered out
					},
				},
			}, nil
		},
	}

	foodLookup := &mockFoodLookup{
		fn: func(_ context.Context, name string) (*nutrition.MacroData, bool, error) {
			if name == "Nasi Telor Dadar" {
				return &nutrition.MacroData{
					FoodName: name,
					Calories: 460,
					ProteinG: 19.2,
					CarbsG:   56.0,
					FatG:     15.0,
					FiberG:   3.4,
					Source:   "food_api",
				}, true, nil
			}
			return nil, false, nil
		},
	}

	app := newChatHandlerTestApp(store, analyzer, foodLookup, 20)

	// Happy path JSON request.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{
		"message": "budget 15rb buat makan siang",
		"conversation_context": [{"role":"user","summary":"lagi cari makan siang"}]
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+uid)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var out ChatAnalyzeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Intent != IntentBudgetMeals || out.BudgetIDR != 15000 {
		t.Fatalf("unexpected intent/budget: %+v", out)
	}
	if len(out.Meals) != 1 || out.Meals[0].Name != "Nasi Telor Dadar" {
		t.Fatalf("expected only 1 within-budget meal, got %+v", out.Meals)
	}
	if out.Meals[0].HealthLabel != HealthLabelFair || out.Meals[0].MacroSource != MacroSourceFoodAPI || out.Meals[0].IsMacroEstimated {
		t.Fatalf("expected deterministic health_label + food_api enrichment, got %+v", out.Meals[0])
	}

	// Empty message -> 400 INVALID_INPUT.
	reqEmpty := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"   "}`))
	reqEmpty.Header.Set("Content-Type", "application/json")
	reqEmpty.Header.Set("Authorization", "Bearer "+uid)
	respEmpty, _ := app.Test(reqEmpty)
	if respEmpty.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for empty message, got %d", respEmpty.StatusCode)
	}

	// Too many conversation turns (> 6) -> 400 INVALID_INPUT.
	reqTurns := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{
		"message": "halo",
		"conversation_context": [
			{"role":"user","summary":"1"},{"role":"assistant","summary":"2"},
			{"role":"user","summary":"3"},{"role":"assistant","summary":"4"},
			{"role":"user","summary":"5"},{"role":"assistant","summary":"6"},
			{"role":"user","summary":"7"}
		]
	}`))
	reqTurns.Header.Set("Content-Type", "application/json")
	reqTurns.Header.Set("Authorization", "Bearer "+uid)
	respTurns, _ := app.Test(reqTurns)
	if respTurns.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for >6 conversation turns, got %d", respTurns.StatusCode)
	}
}

func buildMultipartPhotoRequest(t *testing.T, tok, message, filename, headerMIME string, photoData []byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if message != "" {
		_ = w.WriteField("message", message)
	}
	if photoData != nil {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", `form-data; name="photo"; filename="`+filename+`"`)
		h.Set("Content-Type", headerMIME)
		part, err := w.CreatePart(h)
		if err != nil {
			t.Fatalf("CreatePart: %v", err)
		}
		_, _ = part.Write(photoData)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tok)
	return req
}

func TestChatAnalyze_MultipartPhotoValidationAndUnrecognized(t *testing.T) {
	store := newMemChatProfileStore()
	uid := seedOnboardedChatUser(store)

	var seenMIME string
	var seenLen int
	returnUnrecognized := false

	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, req AnalyzeRequest) (*RawAIResponse, error) {
			seenMIME = req.PhotoMIMEType
			seenLen = len(req.PhotoBytes)
			if returnUnrecognized {
				return &RawAIResponse{
					Intent:             IntentPhotoFood,
					Intro:              "Tidak ditemukan makanan.",
					DetectedItems:      []string{},
					Meals:              []MealOption{},
					NeedsClarification: false,
				}, nil
			}
			return &RawAIResponse{
				Intent:        IntentPhotoFood,
				Intro:         "Ini estimasi gizi makanan di fotomu.",
				DetectedItems: []string{"Nasi", "Ayam Goreng"},
				Meals: []MealOption{
					{Name: "Nasi Ayam Goreng", Portion: "1 porsi", HealthScore: 72, Calories: 540},
				},
			}, nil
		},
	}

	app := newChatHandlerTestApp(store, analyzer, nil, 20)

	// 1. Valid PNG magic bytes (even with wrong header Content-Type) -> 200 OK.
	pngBytes := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 32)...)
	reqPNG := buildMultipartPhotoRequest(t, uid, "cek kalori ini", "meal.png", "application/octet-stream", pngBytes)
	respPNG, err := app.Test(reqPNG)
	if err != nil {
		t.Fatalf("app.Test PNG: %v", err)
	}
	if respPNG.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for valid PNG, got %d", respPNG.StatusCode)
	}
	if seenMIME != "image/png" || seenLen != len(pngBytes) {
		t.Fatalf("expected detected image/png (%d bytes), got %q (%d bytes)", len(pngBytes), seenMIME, seenLen)
	}

	// 2. Spoofed SVG disguised as image/jpeg -> 400 PHOTO_INVALID.
	svgBytes := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`)
	reqSVG := buildMultipartPhotoRequest(t, uid, "", "evil.jpg", "image/jpeg", svgBytes)
	respSVG, _ := app.Test(reqSVG)
	if respSVG.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for spoofed SVG, got %d", respSVG.StatusCode)
	}
	var errBody map[string]string
	_ = json.NewDecoder(respSVG.Body).Decode(&errBody)
	if errBody["code"] != "PHOTO_INVALID" {
		t.Fatalf("expected PHOTO_INVALID, got %q", errBody["code"])
	}

	// 3. Oversized photo (> 5 MiB) -> 400 PHOTO_INVALID.
	bigJPEG := append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, make([]byte, 5*1024*1024+16)...)
	reqBig := buildMultipartPhotoRequest(t, uid, "", "huge.jpg", "image/jpeg", bigJPEG)
	respBig, _ := app.Test(reqBig)
	if respBig.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for >5MiB photo, got %d", respBig.StatusCode)
	}

	// 4. Unrecognized photo -> 422 PHOTO_UNRECOGNIZED.
	returnUnrecognized = true
	webpBytes := append([]byte("RIFF\x00\x00\x00\x00WEBP"), make([]byte, 16)...)
	reqUnrec := buildMultipartPhotoRequest(t, uid, "", "chair.webp", "image/webp", webpBytes)
	respUnrec, _ := app.Test(reqUnrec)
	if respUnrec.StatusCode != fiber.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for unrecognized photo, got %d", respUnrec.StatusCode)
	}
	errBody = nil
	_ = json.NewDecoder(respUnrec.Body).Decode(&errBody)
	if errBody["code"] != "PHOTO_UNRECOGNIZED" {
		t.Fatalf("expected PHOTO_UNRECOGNIZED, got %q", errBody["code"])
	}
}

func TestChatAnalyze_TimeoutReturns504AITimeout(t *testing.T) {
	store := newMemChatProfileStore()
	uid := seedOnboardedChatUser(store)

	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, _ AnalyzeRequest) (*RawAIResponse, error) {
			return nil, mapGeminiError(genai.APIError{
				Code:    http.StatusGatewayTimeout,
				Message: "Deadline expired before operation could complete.",
				Status:  "DEADLINE_EXCEEDED",
			})
		},
	}

	app := newChatHandlerTestApp(store, analyzer, nil, 20)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"budget 15rb"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+uid)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != fiber.StatusGatewayTimeout {
		t.Fatalf("expected 504, got %d", resp.StatusCode)
	}
	var errBody map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&errBody)
	if errBody["code"] != "AI_TIMEOUT" {
		t.Fatalf("expected AI_TIMEOUT, got %q", errBody["code"])
	}
}

func TestMapGeminiError(t *testing.T) {
	deadlineErr := mapGeminiError(genai.APIError{
		Code:    504,
		Message: "Deadline expired before operation could complete.",
		Status:  "DEADLINE_EXCEEDED",
	})
	if !errors.Is(deadlineErr, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", deadlineErr)
	}

	quotaErr := mapGeminiError(genai.APIError{
		Code:    429,
		Message: "Quota exceeded.",
		Status:  "RESOURCE_EXHAUSTED",
	})
	if !errors.Is(quotaErr, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable, got %v", quotaErr)
	}

	unavailErr := mapGeminiError(&genai.APIError{
		Code:    503,
		Message: "Model overloaded.",
		Status:  "UNAVAILABLE",
	})
	if !errors.Is(unavailErr, ErrUnavailable) {
		t.Fatalf("expected ErrUnavailable for pointer APIError, got %v", unavailErr)
	}
}

func TestResolveThinkingLevel(t *testing.T) {
	if got := resolveThinkingLevel(""); got != genai.ThinkingLevelMinimal {
		t.Fatalf("expected default MINIMAL for gemini-3.5-flash-lite, got %q", got)
	}
	if got := resolveThinkingLevel("minimal"); got != genai.ThinkingLevelMinimal {
		t.Fatalf("expected MINIMAL, got %q", got)
	}
	if got := resolveThinkingLevel("low"); got != genai.ThinkingLevelLow {
		t.Fatalf("expected LOW, got %q", got)
	}
	if got := resolveThinkingLevel("medium"); got != genai.ThinkingLevelMedium {
		t.Fatalf("expected MEDIUM, got %q", got)
	}
	if got := resolveThinkingLevel("high"); got != genai.ThinkingLevelHigh {
		t.Fatalf("expected HIGH, got %q", got)
	}
}

func TestChatAnalyze_PerUserRateLimit(t *testing.T) {
	store := newMemChatProfileStore()
	uidA := seedOnboardedChatUser(store)
	uidB := seedOnboardedChatUser(store)

	analyzer := &mockAnalyzer{
		fn: func(_ context.Context, _ AnalyzeRequest) (*RawAIResponse, error) {
			return &RawAIResponse{
				Intent: IntentIngredients,
				Intro:  "OK",
				Meals:  []MealOption{{Name: "Telur Rebus", HealthScore: 80}},
			}, nil
		},
	}

	// Per-user AI rate limit set to 2 req/min.
	app := newChatHandlerTestApp(store, analyzer, nil, 2)

	send := func(tok string) int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/analyze", strings.NewReader(`{"message":"ada telur"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test: %v", err)
		}
		return resp.StatusCode
	}

	if got := send(uidA); got != fiber.StatusOK {
		t.Fatalf("user A req 1: expected 200, got %d", got)
	}
	if got := send(uidA); got != fiber.StatusOK {
		t.Fatalf("user A req 2: expected 200, got %d", got)
	}
	if got := send(uidA); got != fiber.StatusTooManyRequests {
		t.Fatalf("user A req 3: expected 429, got %d", got)
	}

	// User B has an independent quota and must still succeed.
	if got := send(uidB); got != fiber.StatusOK {
		t.Fatalf("user B req 1: expected 200, got %d", got)
	}
}

func TestDetectPhotoMIME(t *testing.T) {
	if mime, ok := DetectPhotoMIME([]byte{0xFF, 0xD8, 0xFF, 0xE0}); !ok || mime != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %q %v", mime, ok)
	}
	if mime, ok := DetectPhotoMIME([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}); !ok || mime != "image/png" {
		t.Fatalf("expected image/png, got %q %v", mime, ok)
	}
	if mime, ok := DetectPhotoMIME([]byte("RIFF1234WEBP")); !ok || mime != "image/webp" {
		t.Fatalf("expected image/webp, got %q %v", mime, ok)
	}
	if _, ok := DetectPhotoMIME([]byte("GIF89a...")); ok {
		t.Fatal("GIF must not be accepted")
	}
}
