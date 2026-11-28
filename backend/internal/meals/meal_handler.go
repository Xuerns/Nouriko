package meals

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	maxMealJSONBytes   = 16 * 1024 // 16 KiB
	maxMealNameRunes   = 200
	minHealthScore     = 0
	maxHealthScore     = 100
	minCalories        = 0
	maxCalories        = 20000
	minMacroG          = 0.0
	maxMacroG          = 9999.99
	minPriceIDR        = 0
	maxPriceIDR        = 10000000
	defaultHistoryDays = 30
	minHistoryDays     = 1
	maxHistoryDays     = 90
	maxFutureMealDays  = 30
	historyFetchLimit  = 1000
	dateLayout         = "2006-01-02"
)

var (
	// wibZone uses a fixed UTC+7 offset so minimal containers without tzdata never fail.
	wibZone     = time.FixedZone("WIB", 7*3600)
	minMealDate = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
)

// ProfileGetter defines the minimal profile read contract required by Handler.
type ProfileGetter interface {
	GetProfileByID(ctx context.Context, userID string) (*profile.Profile, error)
}

// Handler handles saved meals, daily summary, and meal history endpoints (F5, F6, F9).
type Handler struct {
	meals    SavedMealStore
	profiles ProfileGetter
	logger   *slog.Logger
	now      func() time.Time
}

// NewHandler creates a new meals Handler.
func NewHandler(mealStore SavedMealStore, profiles ProfileGetter, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		meals:    mealStore,
		profiles: profiles,
		logger:   logger,
		now:      time.Now,
	}
}

// SavedMealDTO is the canonical JSON representation of a saved meal.
type SavedMealDTO struct {
	ID          string    `json:"id"`
	MealDate    string    `json:"meal_date"`
	MealTime    string    `json:"meal_time"`
	Name        string    `json:"name"`
	HealthScore int       `json:"health_score"`
	HealthLabel string    `json:"health_label"`
	Calories    int       `json:"calories"`
	ProteinG    float64   `json:"protein_g"`
	CarbsG      float64   `json:"carbs_g"`
	FatG        float64   `json:"fat_g"`
	FiberG      float64   `json:"fiber_g"`
	PriceIDR    *int      `json:"price_idr"`
	InputSource string    `json:"input_source"`
	CreatedAt   time.Time `json:"created_at"`
}

// DailySummaryTotals aggregates nutrition, spending, and health score for one day.
type DailySummaryTotals struct {
	Calories           int     `json:"calories"`
	ProteinG           float64 `json:"protein_g"`
	CarbsG             float64 `json:"carbs_g"`
	FatG               float64 `json:"fat_g"`
	FiberG             float64 `json:"fiber_g"`
	TotalSpendingIDR   int64   `json:"total_spending_idr"`
	MealCount          int     `json:"meal_count"`
	AverageHealthScore int     `json:"average_health_score"`
	HealthLabel        *string `json:"health_label"`
}

// DailySummaryTargets exposes the user's daily nutrition targets from their profile.
type DailySummaryTargets struct {
	DailyCalorieTarget int     `json:"daily_calorie_target"`
	TargetProteinG     float64 `json:"target_protein_g"`
	TargetCarbsG       float64 `json:"target_carbs_g"`
	TargetFatG         float64 `json:"target_fat_g"`
	TargetFiberG       float64 `json:"target_fiber_g"`
}

// DailySummaryProgress reports integer completion percentages against daily targets.
type DailySummaryProgress struct {
	Calories int `json:"calories"`
	ProteinG int `json:"protein_g"`
	CarbsG   int `json:"carbs_g"`
	FatG     int `json:"fat_g"`
	FiberG   int `json:"fiber_g"`
}

// DailySummaryResponse is the JSON response for GET /api/v1/meals/summary.
type DailySummaryResponse struct {
	Date            string               `json:"date"`
	Meals           []SavedMealDTO       `json:"meals"`
	Totals          DailySummaryTotals   `json:"totals"`
	Targets         DailySummaryTargets  `json:"targets"`
	ProgressPercent DailySummaryProgress `json:"progress_percent"`
}

// HistoryDaySummary groups saved meals and aggregates totals for one calendar date.
type HistoryDaySummary struct {
	Date               string         `json:"date"`
	MealCount          int            `json:"meal_count"`
	TotalCalories      int            `json:"total_calories"`
	TotalProteinG      float64        `json:"total_protein_g"`
	TotalCarbsG        float64        `json:"total_carbs_g"`
	TotalFatG          float64        `json:"total_fat_g"`
	TotalFiberG        float64        `json:"total_fiber_g"`
	TotalSpendingIDR   int64          `json:"total_spending_idr"`
	AverageHealthScore int            `json:"average_health_score"`
	HealthLabel        string         `json:"health_label"`
	Meals              []SavedMealDTO `json:"meals"`
}

// WeeklyTrendPoint represents a single day in the contiguous 7-day health score trend.
type WeeklyTrendPoint struct {
	Date               string `json:"date"`
	AverageHealthScore *int   `json:"average_health_score"`
	MealCount          int    `json:"meal_count"`
}

// MealHistoryResponse is the JSON response for GET /api/v1/meals/history.
type MealHistoryResponse struct {
	From        string              `json:"from"`
	To          string              `json:"to"`
	Days        []HistoryDaySummary `json:"days"`
	WeeklyTrend []WeeklyTrendPoint  `json:"weekly_trend"`
}

type createMealRequest struct {
	MealDate    string   `json:"meal_date"`
	MealTime    string   `json:"meal_time"`
	Name        string   `json:"name"`
	HealthScore *int     `json:"health_score"`
	Calories    *int     `json:"calories"`
	ProteinG    *float64 `json:"protein_g"`
	CarbsG      *float64 `json:"carbs_g"`
	FatG        *float64 `json:"fat_g"`
	FiberG      *float64 `json:"fiber_g"`
	PriceIDR    *int     `json:"price_idr"`
	InputSource string   `json:"input_source"`
}

// CreateMeal handles POST /api/v1/meals (F5 — Save Meal).
func (h *Handler) CreateMeal(c fiber.Ctx) error {
	claims, _, ok, err := h.requireOnboardedUser(c)
	if !ok {
		return err
	}

	rawBody := c.Body()
	if len(rawBody) > maxMealJSONBytes {
		return httpx.WriteError(c, fiber.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
	}

	var req createMealRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	params, valid := validateCreateMealRequest(claims.UserID, &req, h.now())
	if !valid {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	created, err := h.meals.CreateSavedMeal(c.Context(), params)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"meal": toSavedMealDTO(created),
	})
}

// DeleteMeal handles DELETE /api/v1/meals/:id (F5 — Delete Saved Meal).
func (h *Handler) DeleteMeal(c fiber.Ctx) error {
	claims, _, ok, err := h.requireOnboardedUser(c)
	if !ok {
		return err
	}

	mealID := strings.TrimSpace(c.Params("id"))
	if !isValidUUID(mealID) {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.meals.DeleteSavedMeal(c.Context(), claims.UserID, mealID); err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"deleted": true,
	})
}

// GetDailySummary handles GET /api/v1/meals/summary?date=YYYY-MM-DD (F6 — Daily Summary).
func (h *Handler) GetDailySummary(c fiber.Ctx) error {
	claims, prof, ok, err := h.requireOnboardedUser(c)
	if !ok {
		return err
	}

	dateParam := strings.TrimSpace(c.Query("date"))
	if dateParam == "" {
		dateParam = h.now().In(wibZone).Format(dateLayout)
	} else {
		parsed, valid := parseStrictDate(dateParam)
		if !valid || parsed.Before(minMealDate) {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
	}

	list, err := h.meals.ListSavedMealsByDate(c.Context(), claims.UserID, dateParam)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	mealDTOs := make([]SavedMealDTO, 0, len(list))
	var (
		totalCalories int
		totalProtein  float64
		totalCarbs    float64
		totalFat      float64
		totalFiber    float64
		totalSpending int64
		sumScore      int
	)

	for _, m := range list {
		mealDTOs = append(mealDTOs, toSavedMealDTO(m))
		totalCalories += m.Calories
		totalProtein += m.ProteinG
		totalCarbs += m.CarbsG
		totalFat += m.FatG
		totalFiber += m.FiberG
		if m.PriceIDR != nil {
			totalSpending += int64(*m.PriceIDR)
		}
		sumScore += int(m.HealthScore)
	}

	mealCount := len(list)
	avgScore := 0
	var avgLabel *string
	if mealCount > 0 {
		avgScore = int(math.Round(float64(sumScore) / float64(mealCount)))
		lbl := MealHealthLabel(avgScore)
		avgLabel = &lbl
	}

	totals := DailySummaryTotals{
		Calories:           totalCalories,
		ProteinG:           round2(totalProtein),
		CarbsG:             round2(totalCarbs),
		FatG:               round2(totalFat),
		FiberG:             round2(totalFiber),
		TotalSpendingIDR:   totalSpending,
		MealCount:          mealCount,
		AverageHealthScore: avgScore,
		HealthLabel:        avgLabel,
	}

	targets := DailySummaryTargets{
		DailyCalorieTarget: derefInt(prof.DailyCalorieTarget),
		TargetProteinG:     round2(derefFloat(prof.TargetProteinG)),
		TargetCarbsG:       round2(derefFloat(prof.TargetCarbsG)),
		TargetFatG:         round2(derefFloat(prof.TargetFatG)),
		TargetFiberG:       round2(derefFloat(prof.TargetFiberG)),
	}

	progress := DailySummaryProgress{
		Calories: calcProgressPercent(float64(totals.Calories), float64(targets.DailyCalorieTarget)),
		ProteinG: calcProgressPercent(totals.ProteinG, targets.TargetProteinG),
		CarbsG:   calcProgressPercent(totals.CarbsG, targets.TargetCarbsG),
		FatG:     calcProgressPercent(totals.FatG, targets.TargetFatG),
		FiberG:   calcProgressPercent(totals.FiberG, targets.TargetFiberG),
	}

	return c.Status(fiber.StatusOK).JSON(DailySummaryResponse{
		Date:            dateParam,
		Meals:           mealDTOs,
		Totals:          totals,
		Targets:         targets,
		ProgressPercent: progress,
	})
}

// GetMealHistory handles GET /api/v1/meals/history?from=YYYY-MM-DD&to=YYYY-MM-DD&days=30 (F9 — Meal History).
func (h *Handler) GetMealHistory(c fiber.Ctx) error {
	claims, _, ok, err := h.requireOnboardedUser(c)
	if !ok {
		return err
	}

	toParam := strings.TrimSpace(c.Query("to"))
	var toTime time.Time
	if toParam == "" {
		todayWIBStr := h.now().In(wibZone).Format(dateLayout)
		toTime, _ = parseStrictDate(todayWIBStr)
		toParam = todayWIBStr
	} else {
		parsed, valid := parseStrictDate(toParam)
		if !valid || parsed.Before(minMealDate) {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
		toTime = parsed
	}

	daysVal := defaultHistoryDays
	daysParam := strings.TrimSpace(c.Query("days"))
	if daysParam != "" {
		parsedDays, convErr := strconv.Atoi(daysParam)
		if convErr != nil || parsedDays < minHistoryDays || parsedDays > maxHistoryDays {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
		daysVal = parsedDays
	}

	fromParam := strings.TrimSpace(c.Query("from"))
	var fromTime time.Time
	if fromParam == "" {
		fromTime = toTime.AddDate(0, 0, -(daysVal - 1))
		fromParam = fromTime.Format(dateLayout)
	} else {
		parsed, valid := parseStrictDate(fromParam)
		if !valid || parsed.Before(minMealDate) {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
		fromTime = parsed
		if fromTime.After(toTime) {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
		spanDays := int(toTime.Sub(fromTime).Hours()/24) + 1
		if spanDays > maxHistoryDays {
			return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
		}
	}

	list, err := h.meals.ListMealHistory(c.Context(), claims.UserID, fromParam, toParam, historyFetchLimit, 0)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	weekStart := toTime.AddDate(0, 0, -6)
	weekStartStr := weekStart.Format(dateLayout)
	weeklyPoints, err := h.meals.GetWeeklyHealthScores(c.Context(), claims.UserID, weekStartStr, toParam)
	if err != nil {
		return httpx.HandleDBError(c, h.logger, err)
	}

	days := groupMealsByDate(list)
	weeklyTrend := buildWeeklyTrend(weekStart, weeklyPoints)

	return c.Status(fiber.StatusOK).JSON(MealHistoryResponse{
		From:        fromParam,
		To:          toParam,
		Days:        days,
		WeeklyTrend: weeklyTrend,
	})
}

func (h *Handler) requireOnboardedUser(c fiber.Ctx) (httpx.Claims, *profile.Profile, bool, error) {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.Claims{}, nil, false, httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	prof, err := h.profiles.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httpx.Claims{}, nil, false, httpx.WriteError(c, fiber.StatusForbidden, httpx.CodeOnboardingRequired)
		}
		return httpx.Claims{}, nil, false, httpx.HandleDBError(c, h.logger, err)
	}
	if !prof.OnboardingCompleted() {
		return httpx.Claims{}, nil, false, httpx.WriteError(c, fiber.StatusForbidden, httpx.CodeOnboardingRequired)
	}

	return claims, prof, true, nil
}

func validateCreateMealRequest(userID string, req *createMealRequest, now time.Time) (CreateSavedMealParams, bool) {
	if req == nil {
		return CreateSavedMealParams{}, false
	}

	nowWIB := now.In(wibZone)
	todayWIBStr := nowWIB.Format(dateLayout)
	todayWIB, _ := parseStrictDate(todayWIBStr)
	maxAllowedDate := todayWIB.AddDate(0, 0, maxFutureMealDays)

	mealDate := strings.TrimSpace(req.MealDate)
	if mealDate == "" {
		mealDate = todayWIBStr
	} else {
		parsedDate, valid := parseStrictDate(mealDate)
		if !valid || parsedDate.Before(todayWIB) || parsedDate.After(maxAllowedDate) {
			return CreateSavedMealParams{}, false
		}
	}

	mealTime := strings.ToLower(strings.TrimSpace(req.MealTime))
	if mealTime == "" {
		mealTime = DefaultMealTimeWIB(nowWIB)
	} else if !isValidMealTime(mealTime) {
		return CreateSavedMealParams{}, false
	}

	name := strings.TrimSpace(req.Name)
	nameRunes := utf8.RuneCountInString(name)
	if nameRunes < 1 || nameRunes > maxMealNameRunes {
		return CreateSavedMealParams{}, false
	}

	if req.HealthScore == nil || *req.HealthScore < minHealthScore || *req.HealthScore > maxHealthScore {
		return CreateSavedMealParams{}, false
	}
	healthScore := int16(*req.HealthScore)

	if req.Calories == nil || *req.Calories < minCalories || *req.Calories > maxCalories {
		return CreateSavedMealParams{}, false
	}

	if !isValidMacro(req.ProteinG) || !isValidMacro(req.CarbsG) || !isValidMacro(req.FatG) || !isValidMacro(req.FiberG) {
		return CreateSavedMealParams{}, false
	}

	var priceIDR *int
	if req.PriceIDR != nil {
		if *req.PriceIDR < minPriceIDR || *req.PriceIDR > maxPriceIDR {
			return CreateSavedMealParams{}, false
		}
		v := *req.PriceIDR
		priceIDR = &v
	}

	inputSource := strings.ToLower(strings.TrimSpace(req.InputSource))
	if !isValidInputSource(inputSource) {
		return CreateSavedMealParams{}, false
	}

	return CreateSavedMealParams{
		UserID:      userID,
		MealDate:    mealDate,
		MealTime:    mealTime,
		Name:        name,
		HealthScore: healthScore,
		Calories:    *req.Calories,
		ProteinG:    round2(*req.ProteinG),
		CarbsG:      round2(*req.CarbsG),
		FatG:        round2(*req.FatG),
		FiberG:      round2(*req.FiberG),
		PriceIDR:    priceIDR,
		InputSource: inputSource,
	}, true
}

func groupMealsByDate(list []SavedMeal) []HistoryDaySummary {
	if len(list) == 0 {
		return make([]HistoryDaySummary, 0)
	}

	type dayBucket struct {
		date          string
		meals         []SavedMealDTO
		totalCalories int
		totalProtein  float64
		totalCarbs    float64
		totalFat      float64
		totalFiber    float64
		totalSpending int64
		sumScore      int
	}

	buckets := make(map[string]*dayBucket)
	dates := make([]string, 0)

	for _, m := range list {
		b, exists := buckets[m.MealDate]
		if !exists {
			b = &dayBucket{
				date:  m.MealDate,
				meals: make([]SavedMealDTO, 0),
			}
			buckets[m.MealDate] = b
			dates = append(dates, m.MealDate)
		}
		b.meals = append(b.meals, toSavedMealDTO(m))
		b.totalCalories += m.Calories
		b.totalProtein += m.ProteinG
		b.totalCarbs += m.CarbsG
		b.totalFat += m.FatG
		b.totalFiber += m.FiberG
		if m.PriceIDR != nil {
			b.totalSpending += int64(*m.PriceIDR)
		}
		b.sumScore += int(m.HealthScore)
	}

	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})

	days := make([]HistoryDaySummary, 0, len(dates))
	for _, d := range dates {
		b := buckets[d]
		count := len(b.meals)
		avgScore := int(math.Round(float64(b.sumScore) / float64(count)))
		days = append(days, HistoryDaySummary{
			Date:               b.date,
			MealCount:          count,
			TotalCalories:      b.totalCalories,
			TotalProteinG:      round2(b.totalProtein),
			TotalCarbsG:        round2(b.totalCarbs),
			TotalFatG:          round2(b.totalFat),
			TotalFiberG:        round2(b.totalFiber),
			TotalSpendingIDR:   b.totalSpending,
			AverageHealthScore: avgScore,
			HealthLabel:        MealHealthLabel(avgScore),
			Meals:              b.meals,
		})
	}

	return days
}

func buildWeeklyTrend(weekStart time.Time, points []DailyHealthScorePoint) []WeeklyTrendPoint {
	byDate := make(map[string]DailyHealthScorePoint, len(points))
	for _, pt := range points {
		byDate[pt.Date] = pt
	}

	trend := make([]WeeklyTrendPoint, 0, 7)
	for i := 0; i < 7; i++ {
		d := weekStart.AddDate(0, 0, i).Format(dateLayout)
		if pt, ok := byDate[d]; ok && pt.MealCount > 0 {
			avg := pt.AverageHealthScore
			trend = append(trend, WeeklyTrendPoint{
				Date:               d,
				AverageHealthScore: &avg,
				MealCount:          pt.MealCount,
			})
		} else {
			trend = append(trend, WeeklyTrendPoint{
				Date:               d,
				AverageHealthScore: nil,
				MealCount:          0,
			})
		}
	}
	return trend
}

func toSavedMealDTO(m SavedMeal) SavedMealDTO {
	score := int(m.HealthScore)
	return SavedMealDTO{
		ID:          m.ID,
		MealDate:    m.MealDate,
		MealTime:    m.MealTime,
		Name:        m.Name,
		HealthScore: score,
		HealthLabel: MealHealthLabel(score),
		Calories:    m.Calories,
		ProteinG:    round2(m.ProteinG),
		CarbsG:      round2(m.CarbsG),
		FatG:        round2(m.FatG),
		FiberG:      round2(m.FiberG),
		PriceIDR:    m.PriceIDR,
		InputSource: m.InputSource,
		CreatedAt:   m.CreatedAt,
	}
}

// MealHealthLabel deterministically maps a 0..100 health score to low/fair/good.
func MealHealthLabel(score int) string {
	switch {
	case score >= 70:
		return "good"
	case score >= 40:
		return "fair"
	default:
		return "low"
	}
}

// DefaultMealTimeWIB derives a deterministic meal_time from the hour in WIB (UTC+7).
func DefaultMealTimeWIB(t time.Time) string {
	hour := t.In(wibZone).Hour()
	switch {
	case hour >= 5 && hour <= 10:
		return MealTimeBreakfast
	case hour >= 11 && hour <= 14:
		return MealTimeLunch
	case hour >= 15 && hour <= 17:
		return MealTimeSnack
	case hour >= 18 && hour <= 21:
		return MealTimeDinner
	default:
		return MealTimeSnack
	}
}

func parseStrictDate(s string) (time.Time, bool) {
	if len(s) != len(dateLayout) {
		return time.Time{}, false
	}
	parsed, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, false
	}
	if parsed.Format(dateLayout) != s {
		return time.Time{}, false
	}
	return parsed, true
}

func isValidUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.String(), s)
}

func isValidMealTime(s string) bool {
	switch s {
	case MealTimeBreakfast, MealTimeLunch, MealTimeDinner, MealTimeSnack:
		return true
	default:
		return false
	}
}

func isValidInputSource(s string) bool {
	switch s {
	case InputSourceIngredients, InputSourceBudget, InputSourcePhoto:
		return true
	default:
		return false
	}
}

func isValidMacro(v *float64) bool {
	if v == nil {
		return false
	}
	val := *v
	if math.IsNaN(val) || math.IsInf(val, 0) {
		return false
	}
	return val >= minMacroG && val <= maxMacroG
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func calcProgressPercent(actual, target float64) int {
	if target <= 0 || actual <= 0 {
		return 0
	}
	return int(math.Round((actual / target) * 100))
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
