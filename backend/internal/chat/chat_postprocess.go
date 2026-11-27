package chat

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	maxIntroRunes             = 240
	maxOffTopicNoticeRunes    = 200
	maxBudgetExceededRunes    = 240
	maxMealNameRunes          = 100
	maxMealPortionRunes       = 80
	maxMealNoteRunes          = 160
	maxCookingStepRunes       = 140
	maxCookingSteps           = 6
	maxMissingIngredientRunes = 60
	maxMissingIngredients     = 8
	maxDetectedItemRunes      = 80
	maxDetectedItems          = 15
	maxShoppingNameRunes      = 80
	maxShoppingUnitRunes      = 40
	maxShoppingItems          = 15
	maxStandardMeals          = 5
	maxPhotoFoodMeals         = 1
)

// PostProcessOptions carries contextual metadata for deterministic backend enforcement.
type PostProcessOptions struct {
	Language          string // "id" | "en"
	HasPhoto          bool
	DetectedBudgetIDR int
	ReferencePrices   []FoodPrice
}

// DeriveHealthLabel maps a clamped [0, 100] health score to its deterministic label.
func DeriveHealthLabel(score int) string {
	switch {
	case score <= 39:
		return HealthLabelLow
	case score <= 69:
		return HealthLabelFair
	default:
		return HealthLabelGood
	}
}

// PostProcess validates, clamps, truncates, enforces deterministic health_label,
// applies the strict 0-over-budget filter, and enforces photo/off-topic rules.
func PostProcess(raw *RawAIResponse, opts PostProcessOptions) (*ChatAnalyzeResponse, error) {
	if raw == nil {
		return nil, fmt.Errorf("nil ai response")
	}

	lang := opts.Language
	if lang != "en" {
		lang = "id"
	}

	intent := normalizeIntent(raw.Intent, opts.HasPhoto, opts.DetectedBudgetIDR)

	out := &ChatAnalyzeResponse{
		Intent:             intent,
		Intro:              truncateRunes(strings.TrimSpace(raw.Intro), maxIntroRunes),
		OffTopicNotice:     truncateRunes(strings.TrimSpace(raw.OffTopicNotice), maxOffTopicNoticeRunes),
		DetectedItems:      sanitizeStringSlice(raw.DetectedItems, maxDetectedItems, maxDetectedItemRunes),
		NeedsClarification: raw.NeedsClarification,
		Meals:              []MealOption{},
		ShoppingList:       []ShoppingListItem{},
	}

	// Off-topic responses must never return meals, shopping lists, or budget notices.
	if intent == IntentOffTopic {
		out.DetectedItems = []string{}
		out.NeedsClarification = false
		out.OffTopicNotice = ""
		if out.Intro == "" {
			if lang == "en" {
				out.Intro = "Sorry, I can only help with food recommendations, budget meal ideas, and nutrition information."
			} else {
				out.Intro = "Maaf, saya hanya bisa membantu seputar rekomendasi makanan, resep hemat, dan informasi gizi."
			}
		}
		return out, nil
	}

	// Determine effective budget (backend parser takes precedence when present).
	effectiveBudget := opts.DetectedBudgetIDR
	if effectiveBudget <= 0 && raw.BudgetIDR > 0 {
		effectiveBudget = clampInt(raw.BudgetIDR, 0, maxValidBudgetIDR)
	}

	inputSource := deriveInputSource(intent)
	sanitizedMeals := make([]MealOption, 0, len(raw.Meals))
	for _, m := range raw.Meals {
		name := truncateRunes(strings.TrimSpace(m.Name), maxMealNameRunes)
		if name == "" {
			continue
		}
		score := clampInt(m.HealthScore, 0, 100)
		// Always treat raw LLM macro outputs as estimated; only enrichMealsWithFoodAPI
		// may promote a meal to MacroSourceFoodAPI / IsMacroEstimated: false.
		macroSource := MacroSourceEstimated
		isMacroEst := true
		sanitizedMeals = append(sanitizedMeals, MealOption{
			Name:               name,
			Portion:            truncateRunes(strings.TrimSpace(m.Portion), maxMealPortionRunes),
			HealthScore:        score,
			HealthLabel:        DeriveHealthLabel(score),
			Calories:           maxInt(m.Calories, 0),
			ProteinG:           round1NonNeg(m.ProteinG),
			CarbsG:             round1NonNeg(m.CarbsG),
			FatG:               round1NonNeg(m.FatG),
			FiberG:             round1NonNeg(m.FiberG),
			EstimatedPriceIDR:  maxInt(m.EstimatedPriceIDR, 0),
			IsPriceEstimated:   m.IsPriceEstimated,
			IsMacroEstimated:   isMacroEst,
			MacroSource:        macroSource,
			Note:               truncateRunes(strings.TrimSpace(m.Note), maxMealNoteRunes),
			CookingSteps:       sanitizeStringSlice(m.CookingSteps, maxCookingSteps, maxCookingStepRunes),
			MissingIngredients: sanitizeStringSlice(m.MissingIngredients, maxMissingIngredients, maxMissingIngredientRunes),
			InputSource:        inputSource,
		})
	}

	// Sanitize shopping list (only retained for budget_ingredients).
	sanitizedShopping := make([]ShoppingListItem, 0)
	if intent == IntentBudgetIngredients {
		for _, item := range raw.ShoppingList {
			if len(sanitizedShopping) >= maxShoppingItems {
				break
			}
			name := truncateRunes(strings.TrimSpace(item.Name), maxShoppingNameRunes)
			if name == "" {
				continue
			}
			sanitizedShopping = append(sanitizedShopping, ShoppingListItem{
				Name:              name,
				Unit:              truncateRunes(strings.TrimSpace(item.Unit), maxShoppingUnitRunes),
				EstimatedPriceIDR: maxInt(item.EstimatedPriceIDR, 0),
				IsPriceEstimated:  item.IsPriceEstimated,
				Calories:          maxInt(item.Calories, 0),
				ProteinG:          round1NonNeg(item.ProteinG),
				CarbsG:            round1NonNeg(item.CarbsG),
				FatG:              round1NonNeg(item.FatG),
				FiberG:            round1NonNeg(item.FiberG),
			})
		}
	}

	// Strict Budget Filter: 0 options exceeding budget (or zero-priced fake budget meals) may be returned.
	var removedMeals []MealOption
	if effectiveBudget > 0 && intent != IntentPhotoFood {
		out.BudgetIDR = effectiveBudget
		kept := make([]MealOption, 0, len(sanitizedMeals))
		for _, m := range sanitizedMeals {
			if m.EstimatedPriceIDR > effectiveBudget {
				removedMeals = append(removedMeals, m)
				continue
			}
			if intent == IntentBudgetMeals && m.EstimatedPriceIDR <= 0 {
				continue
			}
			kept = append(kept, m)
		}
		sanitizedMeals = kept

		if intent == IntentBudgetIngredients && len(sanitizedShopping) > 0 {
			trimmedShop := make([]ShoppingListItem, 0, len(sanitizedShopping))
			runningTotal := 0
			for _, item := range sanitizedShopping {
				if item.EstimatedPriceIDR > effectiveBudget {
					continue
				}
				if runningTotal+item.EstimatedPriceIDR <= effectiveBudget {
					trimmedShop = append(trimmedShop, item)
					runningTotal += item.EstimatedPriceIDR
				}
			}
			sanitizedShopping = trimmedShop
		}

		if len(sanitizedMeals) == 0 {
			out.BudgetExceededNotice = buildBudgetExceededNotice(
				lang,
				effectiveBudget,
				intent,
				removedMeals,
				opts.ReferencePrices,
			)
		}
	} else if raw.BudgetExceededNotice != "" {
		out.BudgetExceededNotice = truncateRunes(strings.TrimSpace(raw.BudgetExceededNotice), maxBudgetExceededRunes)
	}

	// Enforce per-intent meal count caps.
	maxMeals := maxStandardMeals
	if intent == IntentPhotoFood {
		maxMeals = maxPhotoFoodMeals
	}
	if len(sanitizedMeals) > maxMeals {
		sanitizedMeals = sanitizedMeals[:maxMeals]
	}

	out.Meals = sanitizedMeals
	out.ShoppingList = sanitizedShopping

	// Photo unrecognized guard: uploaded photo has no food/ingredients and no clarification needed.
	if opts.HasPhoto &&
		(intent == IntentPhotoFood || intent == IntentPhotoIngredients) &&
		len(out.Meals) == 0 &&
		len(out.DetectedItems) == 0 &&
		!out.NeedsClarification {
		return nil, ErrPhotoUnrecognized
	}

	return out, nil
}

func normalizeIntent(raw string, hasPhoto bool, detectedBudget int) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case IntentIngredients:
		return IntentIngredients
	case IntentBudgetMeals:
		return IntentBudgetMeals
	case IntentBudgetIngredients:
		return IntentBudgetIngredients
	case IntentPhotoFood:
		if hasPhoto {
			return IntentPhotoFood
		}
	case IntentPhotoIngredients:
		if hasPhoto {
			return IntentPhotoIngredients
		}
	case IntentOffTopic:
		return IntentOffTopic
	}
	if hasPhoto {
		return IntentPhotoFood
	}
	if detectedBudget > 0 {
		return IntentBudgetMeals
	}
	return IntentIngredients
}

func deriveInputSource(intent string) string {
	switch intent {
	case IntentBudgetMeals, IntentBudgetIngredients:
		return InputSourceBudget
	case IntentPhotoFood, IntentPhotoIngredients:
		return InputSourcePhoto
	default:
		return InputSourceIngredients
	}
}

type priceCandidate struct {
	Name     string
	PriceIDR int
}

func buildBudgetExceededNotice(
	lang string,
	budgetIDR int,
	intent string,
	removedMeals []MealOption,
	refPrices []FoodPrice,
) string {
	preferredCat := FoodPriceCategoryMeal
	if intent == IntentBudgetIngredients {
		preferredCat = FoodPriceCategoryIngredient
	}

	seen := make(map[string]struct{})
	candidates := make([]priceCandidate, 0, len(removedMeals)+len(refPrices))

	for _, m := range removedMeals {
		if m.EstimatedPriceIDR <= 0 || m.Name == "" {
			continue
		}
		key := strings.ToLower(m.Name)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			candidates = append(candidates, priceCandidate{Name: m.Name, PriceIDR: m.EstimatedPriceIDR})
		}
	}

	for _, fp := range refPrices {
		if fp.PriceIDR <= 0 || fp.Name == "" || fp.Category != preferredCat {
			continue
		}
		key := strings.ToLower(fp.Name)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			candidates = append(candidates, priceCandidate{Name: fp.Name, PriceIDR: fp.PriceIDR})
		}
	}

	if len(candidates) == 0 {
		for _, fp := range refPrices {
			if fp.PriceIDR <= 0 || fp.Name == "" {
				continue
			}
			key := strings.ToLower(fp.Name)
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				candidates = append(candidates, priceCandidate{Name: fp.Name, PriceIDR: fp.PriceIDR})
			}
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].PriceIDR != candidates[j].PriceIDR {
			return candidates[i].PriceIDR < candidates[j].PriceIDR
		}
		return candidates[i].Name < candidates[j].Name
	})

	budgetStr := formatIDR(budgetIDR)
	if len(candidates) == 0 {
		if lang == "en" {
			return truncateRunes(fmt.Sprintf("No meal options fit within your %s budget. Try increasing your budget slightly.", budgetStr), maxBudgetExceededRunes)
		}
		return truncateRunes(fmt.Sprintf("Belum ada opsi yang pas dengan budget %s. Coba naikkan sedikit budget kamu.", budgetStr), maxBudgetExceededRunes)
	}

	if len(candidates) == 1 {
		c0 := candidates[0]
		if lang == "en" {
			return truncateRunes(fmt.Sprintf("No options fit within %s. Closest cheapest option: %s (around %s).", budgetStr, c0.Name, formatIDR(c0.PriceIDR)), maxBudgetExceededRunes)
		}
		return truncateRunes(fmt.Sprintf("Belum ada opsi yang pas dengan budget %s. Opsi termurah terdekat adalah %s (sekitar %s).", budgetStr, c0.Name, formatIDR(c0.PriceIDR)), maxBudgetExceededRunes)
	}

	c0, c1 := candidates[0], candidates[1]
	if lang == "en" {
		return truncateRunes(fmt.Sprintf("No options fit within %s. Closest cheapest options: %s (around %s) or %s (around %s).", budgetStr, c0.Name, formatIDR(c0.PriceIDR), c1.Name, formatIDR(c1.PriceIDR)), maxBudgetExceededRunes)
	}
	return truncateRunes(fmt.Sprintf("Belum ada opsi yang pas dengan budget %s. Opsi termurah terdekat adalah %s (sekitar %s) atau %s (sekitar %s).", budgetStr, c0.Name, formatIDR(c0.PriceIDR), c1.Name, formatIDR(c1.PriceIDR)), maxBudgetExceededRunes)
}

func formatIDR(amount int) string {
	if amount < 0 {
		amount = 0
	}
	s := strconv.Itoa(amount)
	n := len(s)
	if n <= 3 {
		return "Rp" + s
	}
	var b strings.Builder
	b.WriteString("Rp")
	rem := n % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if n > rem {
			b.WriteByte('.')
		}
	}
	for i := rem; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte('.')
		}
	}
	return b.String()
}

func sanitizeStringSlice(items []string, maxItems, maxRunes int) []string {
	out := make([]string, 0, min(len(items), maxItems))
	for _, s := range items {
		if len(out) >= maxItems {
			break
		}
		trimmed := truncateRunes(strings.TrimSpace(s), maxRunes)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 || utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return strings.TrimSpace(string(runes[:maxRunes]))
}

func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

func maxInt(v, floor int) int {
	if v < floor {
		return floor
	}
	return v
}

func round1NonNeg(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	return math.Round(v*10.0) / 10.0
}
