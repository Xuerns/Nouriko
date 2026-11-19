package chat

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

const (
	minValidBudgetIDR = 1000
	maxValidBudgetIDR = 10_000_000
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
