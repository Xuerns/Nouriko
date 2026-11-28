package nutrition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"Nouriko/internal/config"
)

const (
	foodAPITimeout       = 3 * time.Second
	foodAPIMaxBodyBytes  = 1 << 20 // 1 MiB
	foodAPICacheMaxItems = 500
	foodAPICacheTTL      = time.Hour
	foodAPIMaxCalories   = 2500
	foodAPISourceLabel   = "food_api"
)

// MacroData holds macro nutrition values retrieved from an external Food API.
type MacroData struct {
	FoodName string
	Calories int
	ProteinG float64
	CarbsG   float64
	FatG     float64
	FiberG   float64
	Source   string // "food_api"
}

// FoodLookup resolves nutrition data for a food item by name.
type FoodLookup interface {
	Lookup(ctx context.Context, foodName string) (*MacroData, bool, error)
}

type foodCacheEntry struct {
	data      *MacroData
	found     bool
	expiresAt time.Time
}

// OpenFoodFactsClient implements FoodLookup against Open Food Facts (or a compatible
// FOOD_API_BASE_URL) with SSRF protections, strict timeouts, response capping, and
// a bounded thread-safe in-memory TTL cache.
type OpenFoodFactsClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	mu       sync.RWMutex
	cache    map[string]foodCacheEntry
	maxItems int
	ttl      time.Duration
	nowFunc  func() time.Time
}

// Compile-time interface check.
var _ FoodLookup = (*OpenFoodFactsClient)(nil)

// NewOpenFoodFactsClient constructs an SSRF-hardened OpenFoodFactsClient from runtime config.
func NewOpenFoodFactsClient(cfg *config.Config) (*OpenFoodFactsClient, error) {
	rawBase := ""
	apiKey := ""
	isProd := false
	if cfg != nil {
		isProd = cfg.IsProduction()
		rawBase = strings.TrimSpace(cfg.Food.BaseURL)
		apiKey = strings.TrimSpace(cfg.Food.APIKey)
	}
	if rawBase == "" {
		return nil, errors.New("FOOD_API_BASE_URL is required")
	}

	validatedURL, err := validateFoodAPIBaseURL(rawBase, isProd)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout:   2 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	if isProd {
		dialer.Control = blockPrivateIPControl
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: foodAPITimeout,
	}

	httpClient := &http.Client{
		Timeout:   foodAPITimeout,
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &OpenFoodFactsClient{
		baseURL:    validatedURL,
		apiKey:     apiKey,
		httpClient: httpClient,
		cache:      make(map[string]foodCacheEntry),
		maxItems:   foodAPICacheMaxItems,
		ttl:        foodAPICacheTTL,
		nowFunc:    time.Now,
	}, nil
}

func validateFoodAPIBaseURL(raw string, isProd bool) (string, error) {
	u, err := url.Parse(strings.TrimRight(raw, "/"))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid FOOD_API_BASE_URL")
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		if isProd {
			if ip, parseErr := netip.ParseAddr(u.Hostname()); parseErr == nil && isBlockedIP(ip) {
				return "", fmt.Errorf("FOOD_API_BASE_URL host is not allowed in production")
			}
		}
		return u.String(), nil
	case "http":
		if isProd {
			return "", fmt.Errorf("FOOD_API_BASE_URL must use https in production")
		}
		host := strings.ToLower(u.Hostname())
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return "", fmt.Errorf("FOOD_API_BASE_URL http scheme is only allowed for loopback hosts")
		}
		return u.String(), nil
	default:
		return "", fmt.Errorf("FOOD_API_BASE_URL must use https")
	}
}

func blockPrivateIPControl(_ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("blocked invalid dial address: %w", err)
	}
	ip, err := netip.ParseAddr(host)
	if err != nil {
		return fmt.Errorf("blocked non-ip dial address: %w", err)
	}
	if isBlockedIP(ip) {
		return errors.New("outbound request to private or local IP is blocked")
	}
	return nil
}

var cgnatPrefix = netip.MustParsePrefix("100.64.0.0/10")

// isBlockedIP returns true for non-global-unicast, loopback, private, CGNAT,
// link-local, unspecified, multicast, and unique-local addresses
// (including IPv4-mapped IPv6).
func isBlockedIP(addr netip.Addr) bool {
	if !addr.IsValid() {
		return true
	}
	ip := addr.Unmap()
	return !ip.IsGlobalUnicast() ||
		ip.IsLoopback() ||
		ip.IsPrivate() ||
		cgnatPrefix.Contains(ip) ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() ||
		ip.IsInterfaceLocalMulticast()
}

// Lookup searches for a food item and returns its macro data if found.
// Cache hits are returned immediately. Upstream errors return (nil, false, err)
// without caching the failure so callers can gracefully fall back to AI estimates.
func (c *OpenFoodFactsClient) Lookup(ctx context.Context, foodName string) (*MacroData, bool, error) {
	if c == nil {
		return nil, false, nil
	}

	normalized := normalizeFoodName(foodName)
	if normalized == "" {
		return nil, false, nil
	}

	if data, found, ok := c.cacheGet(normalized); ok {
		return data, found, nil
	}

	reqCtx, cancel := context.WithTimeout(ctx, foodAPITimeout)
	defer cancel()

	q := url.Values{}
	q.Set("search_terms", normalized)
	q.Set("search_simple", "1")
	q.Set("action", "process")
	q.Set("json", "1")
	q.Set("page_size", "3")
	q.Set("fields", "product_name,nutriments")

	endpoint := c.baseURL + "/cgi/search.pl?" + q.Encode()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, false, fmt.Errorf("build food api request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Nouriko/1.0")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req) // #nosec G704 -- baseURL validated and private IPs blocked in production
	if err != nil {
		return nil, false, fmt.Errorf("food api request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("food api returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, foodAPIMaxBodyBytes))
	if err != nil {
		return nil, false, fmt.Errorf("read food api body: %w", err)
	}

	data, found, err := parseOpenFoodFactsResponse(body, foodName)
	if err != nil {
		return nil, false, err
	}

	c.cacheSet(normalized, data, found)
	return cloneMacroData(data), found, nil
}

type offSearchResponse struct {
	Products []offProduct `json:"products"`
}

type offProduct struct {
	ProductName string         `json:"product_name"`
	Nutriments  map[string]any `json:"nutriments"`
}

func parseOpenFoodFactsResponse(body []byte, fallbackName string) (*MacroData, bool, error) {
	var parsed offSearchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, false, fmt.Errorf("decode food api response: %w", err)
	}

	for _, prod := range parsed.Products {
		if len(prod.Nutriments) == 0 {
			continue
		}
		if m, ok := extractMacroFromNutriments(prod.Nutriments, prod.ProductName, fallbackName); ok {
			return m, true, nil
		}
	}

	return nil, false, nil
}

func extractMacroFromNutriments(n map[string]any, productName, fallbackName string) (*MacroData, bool) {
	// Prefer per-serving values if valid; otherwise fall back to per-100g.
	if m, ok := extractWithSuffix(n, "_serving", productName, fallbackName); ok {
		return m, true
	}
	return extractWithSuffix(n, "_100g", productName, fallbackName)
}

func extractWithSuffix(n map[string]any, suffix, productName, fallbackName string) (*MacroData, bool) {
	kcal, ok := parseNutrimentFloat(n["energy-kcal"+suffix])
	if !ok || kcal <= 0 || kcal > foodAPIMaxCalories {
		return nil, false
	}

	protein, _ := parseNutrimentFloat(n["proteins"+suffix])
	carbs, _ := parseNutrimentFloat(n["carbohydrates"+suffix])
	fat, _ := parseNutrimentFloat(n["fat"+suffix])
	fiber, _ := parseNutrimentFloat(n["fiber"+suffix])

	if protein < 0 || carbs < 0 || fat < 0 || fiber < 0 {
		return nil, false
	}

	name := strings.TrimSpace(productName)
	if name == "" {
		name = strings.TrimSpace(fallbackName)
	}

	return &MacroData{
		FoodName: name,
		Calories: int(math.Round(kcal)),
		ProteinG: round1(protein),
		CarbsG:   round1(carbs),
		FatG:     round1(fat),
		FiberG:   round1(fiber),
		Source:   foodAPISourceLabel,
	}, true
}

func parseNutrimentFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		if math.IsNaN(t) || math.IsInf(t, 0) {
			return 0, false
		}
		return t, true
	case int:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return f, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func normalizeFoodName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(name)), " "))
}

func cloneMacroData(m *MacroData) *MacroData {
	if m == nil {
		return nil
	}
	cp := *m
	return &cp
}

func (c *OpenFoodFactsClient) cacheGet(key string) (*MacroData, bool, bool) {
	now := c.nowFunc()
	c.mu.RLock()
	entry, ok := c.cache[key]
	c.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		return nil, false, false
	}
	return cloneMacroData(entry.data), entry.found, true
}

func (c *OpenFoodFactsClient) cacheSet(key string, data *MacroData, found bool) {
	now := c.nowFunc()
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxItems {
		// Evict expired entries first.
		for k, v := range c.cache {
			if now.After(v.expiresAt) {
				delete(c.cache, k)
			}
		}
		// If still full, evict the entry expiring soonest.
		if len(c.cache) >= c.maxItems {
			var oldestKey string
			var oldestTime time.Time
			first := true
			for k, v := range c.cache {
				if first || v.expiresAt.Before(oldestTime) {
					oldestKey = k
					oldestTime = v.expiresAt
					first = false
				}
			}
			if !first {
				delete(c.cache, oldestKey)
			}
		}
	}

	c.cache[key] = foodCacheEntry{
		data:      cloneMacroData(data),
		found:     found,
		expiresAt: now.Add(c.ttl),
	}
}
