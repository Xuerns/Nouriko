// Package nutrition provides deterministic daily calorie and macronutrient
// target calculations using the Mifflin-St Jeor equation (PRD F8).
package nutrition

import (
	"errors"
	"math"
)

// Validation bounds for anthropometric inputs and custom nutrition targets.
const (
	MinHeightCM = 100.0
	MaxHeightCM = 250.0

	MinWeightKG = 25.0
	MaxWeightKG = 300.0

	MinAge = 13
	MaxAge = 100

	MinDailyCalorieTarget = 800
	MaxDailyCalorieTarget = 6000

	MinProteinG = 10.0
	MaxProteinG = 500.0

	MinCarbsG = 20.0
	MaxCarbsG = 1000.0

	MinFatG = 10.0
	MaxFatG = 300.0

	MinFiberG = 5.0
	MaxFiberG = 150.0
)

// Supported gender values.
const (
	GenderMale   = "male"
	GenderFemale = "female"
)

// Supported activity level values.
const (
	ActivityLight    = "light"
	ActivityModerate = "moderate"
	ActivityHeavy    = "heavy"
)

// Activity multipliers applied to Mifflin-St Jeor BMR.
const (
	multiplierLight    = 1.375
	multiplierModerate = 1.55
	multiplierHeavy    = 1.725
)

// ErrInvalidInput indicates that one or more anthropometric or nutrition values
// fall outside the supported range or enum values.
var ErrInvalidInput = errors.New("invalid nutrition input")

// Anthropometrics holds the required inputs for Mifflin-St Jeor target computation.
type Anthropometrics struct {
	HeightCM      float64
	WeightKG      float64
	Age           int
	Gender        string
	ActivityLevel string
}

// Targets holds the computed daily calorie and macronutrient targets.
type Targets struct {
	DailyCalorieTarget int
	TargetProteinG     float64
	TargetCarbsG       float64
	TargetFatG         float64
	TargetFiberG       float64
}

// CalculateTargets validates anthropometric inputs and returns daily calorie and
// default macronutrient targets (Protein 20%, Carbs 50%, Fat 30%, Fiber 14g/1000kcal).
func CalculateTargets(in Anthropometrics) (Targets, error) {
	if !ValidateHeightCM(in.HeightCM) ||
		!ValidateWeightKG(in.WeightKG) ||
		!ValidateAge(in.Age) ||
		!ValidateGender(in.Gender) ||
		!ValidateActivityLevel(in.ActivityLevel) {
		return Targets{}, ErrInvalidInput
	}

	bmr := 10.0*in.WeightKG + 6.25*in.HeightCM - 5.0*float64(in.Age)
	if in.Gender == GenderMale {
		bmr += 5.0
	} else {
		bmr -= 161.0
	}

	var multiplier float64
	switch in.ActivityLevel {
	case ActivityLight:
		multiplier = multiplierLight
	case ActivityModerate:
		multiplier = multiplierModerate
	case ActivityHeavy:
		multiplier = multiplierHeavy
	default:
		return Targets{}, ErrInvalidInput
	}

	dailyKcal := int(math.Round(bmr * multiplier))
	if dailyKcal < MinDailyCalorieTarget {
		dailyKcal = MinDailyCalorieTarget
	}

	return CalculateMacrosFromCalories(dailyKcal), nil
}

// CalculateMacrosFromCalories derives default macro targets from a daily calorie target.
func CalculateMacrosFromCalories(dailyKcal int) Targets {
	kcal := float64(dailyKcal)
	return Targets{
		DailyCalorieTarget: dailyKcal,
		TargetProteinG:     round1((kcal * 0.20) / 4.0),
		TargetCarbsG:       round1((kcal * 0.50) / 4.0),
		TargetFatG:         round1((kcal * 0.30) / 9.0),
		TargetFiberG:       round1((kcal / 1000.0) * 14.0),
	}
}

// ValidateHeightCM checks whether height in cm is finite and within [100, 250].
func ValidateHeightCM(v float64) bool {
	return isFiniteInRange(v, MinHeightCM, MaxHeightCM)
}

// ValidateWeightKG checks whether weight in kg is finite and within [25, 300].
func ValidateWeightKG(v float64) bool {
	return isFiniteInRange(v, MinWeightKG, MaxWeightKG)
}

// ValidateAge checks whether age is within [13, 100].
func ValidateAge(v int) bool {
	return v >= MinAge && v <= MaxAge
}

// ValidateGender checks whether gender is "male" or "female".
func ValidateGender(v string) bool {
	return v == GenderMale || v == GenderFemale
}

// ValidateActivityLevel checks whether activity level is "light", "moderate", or "heavy".
func ValidateActivityLevel(v string) bool {
	return v == ActivityLight || v == ActivityModerate || v == ActivityHeavy
}

// ValidateDailyCalorieTarget checks whether daily_calorie_target is within [800, 6000].
func ValidateDailyCalorieTarget(v int) bool {
	return v >= MinDailyCalorieTarget && v <= MaxDailyCalorieTarget
}

// ValidateProteinG checks whether target_protein_g is finite and within [10, 500].
func ValidateProteinG(v float64) bool {
	return isFiniteInRange(v, MinProteinG, MaxProteinG)
}

// ValidateCarbsG checks whether target_carbs_g is finite and within [20, 1000].
func ValidateCarbsG(v float64) bool {
	return isFiniteInRange(v, MinCarbsG, MaxCarbsG)
}

// ValidateFatG checks whether target_fat_g is finite and within [10, 300].
func ValidateFatG(v float64) bool {
	return isFiniteInRange(v, MinFatG, MaxFatG)
}

// ValidateFiberG checks whether target_fiber_g is finite and within [5, 150].
func ValidateFiberG(v float64) bool {
	return isFiniteInRange(v, MinFiberG, MaxFiberG)
}

func isFiniteInRange(v, minVal, maxVal float64) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return false
	}
	return v >= minVal && v <= maxVal
}

func round1(v float64) float64 {
	return math.Round(v*10.0) / 10.0
}
