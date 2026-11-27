package chat

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	minValidBudgetIDR         = 1000
	minRealisticMealBudgetIDR = 4000
	maxRequestedMealCount     = 10
	maxValidBudgetIDR         = 10_000_000
)

var (
	// Matches shorthand thousands like "15rb", "15 rb", "12,5rb", "12.5k", "20ribu", "20 ribu".
	reBudgetShorthand = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(?:(?:rp\.?|idr)\s*)?(\d{1,4}(?:[.,]\d{1,2})?)\s*(rb|ribu|k)(?:[^\p{L}\p{N}]|$)`)

	// Matches explicit currency prefix like "Rp 15.000", "rp15000", "Rp. 25,000", "IDR 20.000".
	reBudgetCurrency = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(?:rp\.?|idr)\s*(\d{1,3}(?:[.,]\d{3})+|\d{4,7})(?:[^\p{L}\p{N}]|$)`)

	// Matches dot-grouped thousands like "15.000" or "25.000" (standard Indonesian Rupiah formatting).
	reBudgetDotted = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(\d{1,3}(?:\.\d{3})+)(?:[^\p{L}\p{N}]|$)`)

	// Matches plain integer preceded by a budget cue keyword like "budget 15000", "uang 25000", "max 20000".
	reBudgetKeyword = regexp.MustCompile(`(?i)(?:budget|bujet|uang|duit|modal|dana|harga|maks(?:imal)?|max|under|below|sekitar|cuma|hanya|punya|ada)\s*(?:di\s*bawah\s*|kurang\s*dari\s*|sekitar\s*|cuma\s*)?(?:rp\.?\s*|idr\s*)?(\d{4,7})(?:[^\p{L}\p{N}]|$)`)

	// Matches explicit tiny amounts under Rp1.000 such as "Rp 500", "500 perak", "500 rupiah", "budget 200", "0,5rb".
	reTinyBudgetCurrency = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(?:(?:rp\.?|idr)\s*(\d{1,3})|(\d{1,3})\s*(?:perak|rupiah|idr)|(?:budget|bujet|uang|duit|modal|dana)\s*(?:rp\.?\s*|idr\s*)?(\d{1,3}))(?:[^\p{L}\p{N}]|$)`)

	// Matches zero-money / free-food cues in Indonesian and English.
	reNoMoneyCue = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(?:(?:ga|gak|ngga|nggak|tidak|tak|tdk|ndak|kagak|gk)\s*(?:punya|ada)\s*(?:uang|duit|budget|bujet|modal|dana)|tanpa\s*(?:uang|duit|budget|bujet|modal|biaya)|(?:uang|duit|budget|bujet|modal)\s*(?:kosong|habis|nol|0)|(?:rp\.?|idr)\s*0|0\s*(?:rb|ribu|k|rupiah|perak|idr)|makanan\s*gratis|no\s*money|zero\s*budget|have\s*no\s*(?:money|budget)|free\s*(?:food|meal|meals)|\$0)(?:[^\p{L}\p{N}]|$)`)

	// Matches contrast cues indicating the user actually has ingredients at home despite no money.
	reHasIngredientsContrast = regexp.MustCompile(`(?i)(?:tapi|tetapi|namun|cuma|cuman|hanya|but|only)\s+(?:ada|punya|have)\b`)

	// Matches explicit requested meal counts like "50 makanan", "20 menu", "15 meals".
	reRequestedMealCount = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(\d{1,4})\s*(?:makanan|menu|porsi|hidangan|lauk|opsi|pilihan|meal|meals|dish|dishes|options?)(?:[^\p{L}\p{N}]|$)`)

	// Matches explicit combined multi-meal phrases like "3x makan", "3 kali makan", "3 porsi", "untuk 3 hari".
	reCombinedMealCount = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(\d{1,3})\s*(?:x\s*makan|kali\s*makan|porsi|hari|days?)(?:[^\p{L}\p{N}]|$)`)
)

// ExtractBudgetIDR parses a user's free-form Indonesian or English message and returns
// the detected Rupiah budget, or 0 if no realistic budget is found.
func ExtractBudgetIDR(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}

	// 1. Shorthand thousands ("15rb", "12,5rb", "20k", "15 ribu").
	if m := reBudgetShorthand.FindStringSubmatch(trimmed); len(m) >= 2 {
		if val := parseShorthandThousands(m[1]); isValidBudget(val) {
			return val
		}
	}

	// 2. Explicit currency prefix ("Rp 15.000", "rp25000", "IDR 20.000").
	if m := reBudgetCurrency.FindStringSubmatch(trimmed); len(m) >= 2 {
		if val := parseGroupedIDR(m[1]); isValidBudget(val) {
			return val
		}
	}

	// 3. Budget cue keyword + plain integer ("budget 15000", "uang 20000", "under 15000").
	if m := reBudgetKeyword.FindStringSubmatch(trimmed); len(m) >= 2 {
		if val := parseGroupedIDR(m[1]); isValidBudget(val) {
			return val
		}
	}

	// 4. Standard Indonesian dot-grouped thousands ("15.000", "25.000"), ignoring calorie/weight units.
	if m := reBudgetDotted.FindStringSubmatchIndex(trimmed); len(m) >= 4 {
		numStr := trimmed[m[2]:m[3]]
		after := strings.ToLower(strings.TrimSpace(trimmed[m[3]:]))
		if !hasNonCurrencyUnitPrefix(after) {
			if val := parseGroupedIDR(numStr); isValidBudget(val) {
				return val
			}
		}
	}

	return 0
}

func parseShorthandThousands(raw string) int {
	normalized := strings.ReplaceAll(raw, ",", ".")
	f, err := strconv.ParseFloat(normalized, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	return int(math.Round(f * 1000))
}

func parseGroupedIDR(raw string) int {
	digits := strings.NewReplacer(".", "", ",", "").Replace(raw)
	n, err := strconv.Atoi(digits)
	if err != nil {
		return 0
	}
	return n
}

func hasNonCurrencyUnitPrefix(after string) bool {
	for _, unit := range []string{"kcal", "kkal", "kalori", "cal", "gram", "gr", "kg", "ml", "liter", "langkah", "steps"} {
		if strings.HasPrefix(after, unit) {
			return true
		}
	}
	return false
}

func isValidBudget(v int) bool {
	return v >= minValidBudgetIDR && v <= maxValidBudgetIDR
}

// ExtractRequestedMealCount parses an explicit count of requested meals/menus
// (e.g. "3 makanan", "5 menu", "4 meals"), or returns 0 if none is specified.
func ExtractRequestedMealCount(text string) int {
	m := reRequestedMealCount.FindStringSubmatch(strings.TrimSpace(text))
	if len(m) < 2 {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func extractSubMinimumBudgetIDR(text string) (int, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0, false
	}
	if m := reBudgetShorthand.FindStringSubmatch(trimmed); len(m) >= 2 {
		val := parseShorthandThousands(m[1])
		if val >= 0 && val < minRealisticMealBudgetIDR {
			return val, true
		}
	}
	if m := reTinyBudgetCurrency.FindStringSubmatch(trimmed); len(m) >= 4 {
		for _, raw := range m[1:4] {
			if raw == "" {
				continue
			}
			if val, err := strconv.Atoi(raw); err == nil && val >= 0 && val < minRealisticMealBudgetIDR {
				return val, true
			}
		}
	}
	return 0, false
}

// DetectUnrealisticRequest deterministically refuses impossible or unrealistic
// budget/quantity requests (e.g., Rp0 / no money, budgets below Rp4.000, or
// impossible budget-per-meal counts) with a polite "Maaf, ..." / "Sorry, ..."
// response and zero hallucinated meal cards.
func DetectUnrealisticRequest(text string, hasPhoto bool, lang string) (*ChatAnalyzeResponse, bool) {
	if hasPhoto {
		return nil, false
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, false
	}

	buildRefusal := func(introID, introEN string, budgetIDR int) *ChatAnalyzeResponse {
		intro := introID
		if lang == "en" {
			intro = introEN
		}
		return &ChatAnalyzeResponse{
			Intent:        IntentBudgetMeals,
			Intro:         truncateRunes(intro, maxIntroRunes),
			DetectedItems: []string{},
			BudgetIDR:     budgetIDR,
			Meals:         []MealOption{},
			ShoppingList:  []ShoppingListItem{},
		}
	}

	// 1. Zero / no-money cues (unless user explicitly says they already have ingredients to cook).
	if reNoMoneyCue.MatchString(trimmed) && !reHasIngredientsContrast.MatchString(trimmed) {
		return buildRefusal(
			"Maaf, Nouriko belum bisa memberikan rekomendasi makanan jika budget Rp0 atau tanpa uang. Jika kamu sudah punya bahan di kos, sebutkan saja bahannya agar saya bantu buatkan ide masakannya.",
			"Sorry, Nouriko cannot recommend purchasable meals with a Rp0 budget. If you already have ingredients at home, tell me what you have and I can suggest recipes.",
			0,
		), true
	}

	// 2. Sub-minimum budget (< Rp4.000, including Rp100-Rp3.999 and shorthand like "1rb", "500 perak").
	detectedBudget := ExtractBudgetIDR(trimmed)
	if detectedBudget == 0 {
		if tinyBudget, ok := extractSubMinimumBudgetIDR(trimmed); ok {
			if tinyBudget == 0 {
				return buildRefusal(
					"Maaf, Nouriko belum bisa memberikan rekomendasi makanan jika budget Rp0 atau tanpa uang. Jika kamu sudah punya bahan di kos, sebutkan saja bahannya agar saya bantu buatkan ide masakannya.",
					"Sorry, Nouriko cannot recommend purchasable meals with a Rp0 budget. If you already have ingredients at home, tell me what you have and I can suggest recipes.",
					0,
				), true
			}
			budgetStr := formatIDR(tinyBudget)
			return buildRefusal(
				"Maaf, dengan budget "+budgetStr+" belum memungkinkan untuk mendapatkan rekomendasi makanan yang layak. Coba gunakan budget minimal sekitar Rp10.000, atau sebutkan bahan yang sudah kamu punya.",
				"Sorry, a "+budgetStr+" budget is too low to realistically buy a proper meal. Try a budget starting around Rp10.000, or tell me what ingredients you already have.",
				tinyBudget,
			), true
		}
	} else if detectedBudget < minRealisticMealBudgetIDR {
		budgetStr := formatIDR(detectedBudget)
		return buildRefusal(
			"Maaf, dengan budget "+budgetStr+" belum memungkinkan untuk mendapatkan rekomendasi makanan yang layak. Coba gunakan budget minimal sekitar Rp10.000, atau sebutkan bahan yang sudah kamu punya.",
			"Sorry, a "+budgetStr+" budget is too low to realistically buy a proper meal. Try a budget starting around Rp10.000, or tell me what ingredients you already have.",
			detectedBudget,
		), true
	}

	// 3. Unrealistic requested meal count (> 10 meals or combined budget per meal < Rp4.000).
	requestedCount := ExtractRequestedMealCount(trimmed)
	if requestedCount > maxRequestedMealCount {
		return buildRefusal(
			"Maaf, Nouriko hanya bisa memberikan maksimal 5 rekomendasi makanan dalam sekali permintaan. Coba minta 1 sampai 5 menu agar rekomendasinya lebih tepat.",
			"Sorry, Nouriko can only provide up to 5 meal recommendations per request. Please ask for 1 to 5 meals so the suggestions stay practical.",
			detectedBudget,
		), true
	}
	if m := reCombinedMealCount.FindStringSubmatch(trimmed); len(m) >= 2 {
		if combinedCount, err := strconv.Atoi(m[1]); err == nil && detectedBudget >= minRealisticMealBudgetIDR && combinedCount >= 2 && detectedBudget/combinedCount < minRealisticMealBudgetIDR {
			budgetStr := formatIDR(detectedBudget)
			return buildRefusal(
				"Maaf, dengan budget "+budgetStr+" belum memungkinkan untuk memenuhi kebutuhan "+strconv.Itoa(combinedCount)+" porsi sekaligus. Coba sesuaikan jumlah porsi atau naikkan budget kamu.",
				"Sorry, a "+budgetStr+" budget is not enough to realistically cover "+strconv.Itoa(combinedCount)+" meals combined. Try adjusting the count or increasing your budget.",
				detectedBudget,
			), true
		}
	}

	return nil, false
}
