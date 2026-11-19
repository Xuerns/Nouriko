package nutrition

import (
	"errors"
	"math"
	"testing"
)

func TestCalculateTargetsMifflinStJeor(t *testing.T) {
	tests := []struct {
		name      string
		in        Anthropometrics
		wantKcal  int
		wantProt  float64
		wantCarbs float64
		wantFat   float64
		wantFiber float64
	}{
		{
			name: "male moderate",
			in: Anthropometrics{
				HeightCM:      170,
				WeightKG:      65,
				Age:           20,
				Gender:        GenderMale,
				ActivityLevel: ActivityModerate,
			},
			// BMR = 650 + 1062.5 - 100 + 5 = 1617.5; * 1.55 = 2507.125 -> 2507
			wantKcal:  2507,
			wantProt:  125.4, // 2507*0.20/4 = 125.35 -> 125.4
			wantCarbs: 313.4, // 2507*0.50/4 = 313.375 -> 313.4
			wantFat:   83.6,  // 2507*0.30/9 = 83.566... -> 83.6
			wantFiber: 35.1,  // 2.507*14 = 35.098 -> 35.1
		},
		{
			name: "female light",
			in: Anthropometrics{
				HeightCM:      160,
				WeightKG:      55,
				Age:           22,
				Gender:        GenderFemale,
				ActivityLevel: ActivityLight,
			},
			// BMR = 550 + 1000 - 110 - 161 = 1279; * 1.375 = 1758.625 -> 1759
			wantKcal:  1759,
			wantProt:  88.0,  // 1759*0.20/4 = 87.95 -> 88.0
			wantCarbs: 219.9, // 1759*0.50/4 = 219.875 -> 219.9
			wantFat:   58.6,  // 1759*0.30/9 = 58.633... -> 58.6
			wantFiber: 24.6,  // 1.759*14 = 24.626 -> 24.6
		},
		{
			name: "male heavy",
			in: Anthropometrics{
				HeightCM:      180,
				WeightKG:      75,
				Age:           25,
				Gender:        GenderMale,
				ActivityLevel: ActivityHeavy,
			},
			// BMR = 750 + 1125 - 125 + 5 = 1755; * 1.725 = 3027.375 -> 3027
			wantKcal:  3027,
			wantProt:  151.4,
			wantCarbs: 378.4,
			wantFat:   100.9,
			wantFiber: 42.4,
		},
		{
			name: "female heavy",
			in: Anthropometrics{
				HeightCM:      165,
				WeightKG:      60,
				Age:           30,
				Gender:        GenderFemale,
				ActivityLevel: ActivityHeavy,
			},
			// BMR = 600 + 1031.25 - 150 - 161 = 1320.25; * 1.725 = 2277.43125 -> 2277
			wantKcal:  2277,
			wantProt:  113.9,
			wantCarbs: 284.6,
			wantFat:   75.9,
			wantFiber: 31.9,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CalculateTargets(tc.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DailyCalorieTarget != tc.wantKcal {
				t.Errorf("kcal: got %d, want %d", got.DailyCalorieTarget, tc.wantKcal)
			}
			if got.TargetProteinG != tc.wantProt {
				t.Errorf("protein: got %.1f, want %.1f", got.TargetProteinG, tc.wantProt)
			}
			if got.TargetCarbsG != tc.wantCarbs {
				t.Errorf("carbs: got %.1f, want %.1f", got.TargetCarbsG, tc.wantCarbs)
			}
			if got.TargetFatG != tc.wantFat {
				t.Errorf("fat: got %.1f, want %.1f", got.TargetFatG, tc.wantFat)
			}
			if got.TargetFiberG != tc.wantFiber {
				t.Errorf("fiber: got %.1f, want %.1f", got.TargetFiberG, tc.wantFiber)
			}
		})
	}
}

func TestCalculateMacrosFromCalories2000(t *testing.T) {
	got := CalculateMacrosFromCalories(2000)
	if got.DailyCalorieTarget != 2000 || got.TargetProteinG != 100.0 || got.TargetCarbsG != 250.0 || got.TargetFatG != 66.7 || got.TargetFiberG != 28.0 {
		t.Fatalf("unexpected 2000 kcal macros: %+v", got)
	}
}

func TestCalculateTargetsValidationErrors(t *testing.T) {
	valid := Anthropometrics{
		HeightCM:      170,
		WeightKG:      65,
		Age:           20,
		Gender:        GenderMale,
		ActivityLevel: ActivityModerate,
	}

	cases := []struct {
		name   string
		mutate func(*Anthropometrics)
	}{
		{"height too low", func(a *Anthropometrics) { a.HeightCM = 99.9 }},
		{"height too high", func(a *Anthropometrics) { a.HeightCM = 250.1 }},
		{"height NaN", func(a *Anthropometrics) { a.HeightCM = math.NaN() }},
		{"weight too low", func(a *Anthropometrics) { a.WeightKG = 24.9 }},
		{"weight too high", func(a *Anthropometrics) { a.WeightKG = 300.1 }},
		{"weight Inf", func(a *Anthropometrics) { a.WeightKG = math.Inf(1) }},
		{"age too low", func(a *Anthropometrics) { a.Age = 12 }},
		{"age too high", func(a *Anthropometrics) { a.Age = 101 }},
		{"invalid gender", func(a *Anthropometrics) { a.Gender = "other" }},
		{"invalid activity", func(a *Anthropometrics) { a.ActivityLevel = "sedentary" }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := valid
			tc.mutate(&in)
			if _, err := CalculateTargets(in); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestCustomTargetValidators(t *testing.T) {
	if !ValidateDailyCalorieTarget(800) || !ValidateDailyCalorieTarget(6000) || ValidateDailyCalorieTarget(799) || ValidateDailyCalorieTarget(6001) {
		t.Fatal("daily calorie validator bounds mismatch")
	}
	if !ValidateProteinG(10) || !ValidateProteinG(500) || ValidateProteinG(9.9) || ValidateProteinG(500.1) {
		t.Fatal("protein validator bounds mismatch")
	}
	if !ValidateCarbsG(20) || !ValidateCarbsG(1000) || ValidateCarbsG(19.9) || ValidateCarbsG(1000.1) {
		t.Fatal("carbs validator bounds mismatch")
	}
	if !ValidateFatG(10) || !ValidateFatG(300) || ValidateFatG(9.9) || ValidateFatG(300.1) {
		t.Fatal("fat validator bounds mismatch")
	}
	if !ValidateFiberG(5) || !ValidateFiberG(150) || ValidateFiberG(4.9) || ValidateFiberG(150.1) {
		t.Fatal("fiber validator bounds mismatch")
	}
}
