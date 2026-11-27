package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"Nouriko/internal/nutrition"
)

const (
	foodEnrichmentTimeout     = 3500 * time.Millisecond
	foodEnrichmentConcurrency = 5
)

// ChatService orchestrates the full AI nutrition chat and photo analysis pipeline:
// budget extraction -> FoodPriceStore query -> Analyzer -> PostProcess -> FoodLookup enrichment.
type ChatService struct {
	analyzer   Analyzer
	foodPrices FoodPriceStore
	foodLookup nutrition.FoodLookup
	logger     *slog.Logger
}

// NewChatService constructs a ChatService with injectable dependencies.
func NewChatService(
	analyzer Analyzer,
	foodPrices FoodPriceStore,
	foodLookup nutrition.FoodLookup,
	logger *slog.Logger,
) *ChatService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChatService{
		analyzer:   analyzer,
		foodPrices: foodPrices,
		foodLookup: foodLookup,
		logger:     logger,
	}
}

// Analyze executes the end-to-end chat/photo analysis workflow.
func (s *ChatService) Analyze(ctx context.Context, input AnalyzeInput) (*ChatAnalyzeResponse, error) {
	if s == nil || s.analyzer == nil {
		return nil, ErrUnavailable
	}

	lang := input.Language
	if lang != "en" {
		lang = "id"
	}

	if refusal, blocked := DetectUnrealisticRequest(input.Message, len(input.PhotoBytes) > 0, lang); blocked {
		return refusal, nil
	}

	detectedBudget := ExtractBudgetIDR(input.Message)
	refPrices := s.fetchReferencePrices(ctx, detectedBudget)

	req := AnalyzeRequest{
		SystemInstruction: BuildSystemInstruction(lang),
		UserPrompt:        BuildUserPrompt(input, detectedBudget, refPrices),
		PhotoBytes:        input.PhotoBytes,
		PhotoMIMEType:     input.PhotoMIMEType,
	}

	raw, err := s.analyzer.Analyze(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := PostProcess(raw, PostProcessOptions{
		Language:          lang,
		HasPhoto:          len(input.PhotoBytes) > 0,
		DetectedBudgetIDR: detectedBudget,
		ReferencePrices:   refPrices,
	})
	if err != nil {
		return nil, err
	}

	if s.foodLookup != nil && len(resp.Meals) > 0 {
		s.enrichMealsWithFoodAPI(ctx, resp.Meals)
	}

	return resp, nil
}

func (s *ChatService) fetchReferencePrices(ctx context.Context, detectedBudget int) []FoodPrice {
	if s.foodPrices == nil {
		return nil
	}

	if detectedBudget > 0 {
		items, err := s.foodPrices.ListFoodPricesByBudget(ctx, detectedBudget, "", defaultFoodPriceLimit)
		if err != nil {
			s.logger.Warn("failed to load budget food prices", "error", err)
			return nil
		}
		if len(items) > 0 {
			return items
		}
		// If nothing fits the budget, fetch the cheapest overall items so
		// PostProcess can suggest the closest cheapest option in budget_exceeded_notice.
		fallback, err := s.foodPrices.ListAllFoodPrices(ctx, "", 20)
		if err != nil {
			s.logger.Warn("failed to load fallback food prices", "error", err)
			return nil
		}
		return fallback
	}

	items, err := s.foodPrices.ListAllFoodPrices(ctx, "", defaultFoodPriceLimit)
	if err != nil {
		s.logger.Warn("failed to load reference food prices", "error", err)
		return nil
	}
	return items
}

func (s *ChatService) enrichMealsWithFoodAPI(ctx context.Context, meals []MealOption) {
	enrichCtx, cancel := context.WithTimeout(ctx, foodEnrichmentTimeout)
	defer cancel()

	sem := make(chan struct{}, foodEnrichmentConcurrency)
	var wg sync.WaitGroup

	for i := range meals {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-enrichCtx.Done():
				return
			}

			data, found, err := s.foodLookup.Lookup(enrichCtx, meals[idx].Name)
			if err != nil {
				s.logger.Debug("food api lookup failed, keeping ai estimate", "meal", meals[idx].Name, "error", err)
				return
			}
			if !found || data == nil {
				return
			}

			meals[idx].Calories = data.Calories
			meals[idx].ProteinG = data.ProteinG
			meals[idx].CarbsG = data.CarbsG
			meals[idx].FatG = data.FatG
			meals[idx].FiberG = data.FiberG
			meals[idx].MacroSource = MacroSourceFoodAPI
			meals[idx].IsMacroEstimated = false
		}(i)
	}

	wg.Wait()
}
