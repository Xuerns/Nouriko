package handler

import (
	"math"
	"strings"
	"testing"
	"time"

	"Nouriko/internal/database"
)

func TestDefaultMealTimeWIB(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, database.MealTimeSnack},
		{4, database.MealTimeSnack},
		{5, database.MealTimeBreakfast},
		{10, database.MealTimeBreakfast},
		{11, database.MealTimeLunch},
		{14, database.MealTimeLunch},
		{15, database.MealTimeSnack},
		{17, database.MealTimeSnack},
		{18, database.MealTimeDinner},
		{21, database.MealTimeDinner},
		{22, database.MealTimeSnack},
		{23, database.MealTimeSnack},
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
			InputSource: database.InputSourceIngredients,
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
	if params.MealTime != database.MealTimeLunch {
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

	// Far future date (> todayWIB + 1 day)
	futureDate := baseReq()
	futureDate.MealDate = "2026-11-21"
	if _, ok := validateCreateMealRequest("user-1", futureDate, now); ok {
		t.Fatal("expected meal_date > today+1d to be rejected")
	}

	// Today + 1 day is allowed (ahead timezones)
	tomorrowDate := baseReq()
	tomorrowDate.MealDate = "2026-11-20"
	if _, ok := validateCreateMealRequest("user-1", tomorrowDate, now); !ok {
		t.Fatal("expected meal_date == today+1d to be allowed")
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
