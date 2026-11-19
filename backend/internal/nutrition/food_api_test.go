package nutrition

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"Nouriko/internal/config"
)

func TestOpenFoodFactsClient_ServingAndFallback100g(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Errorf("expected Bearer header, got %q", got)
		}
		term := r.URL.Query().Get("search_terms")
		w.Header().Set("Content-Type", "application/json")
		switch term {
		case "oatmeal instan":
			_, _ = w.Write([]byte(`{
				"products": [{
					"product_name": "Oatmeal Instan",
					"nutriments": {
						"energy-kcal_serving": 150,
						"proteins_serving": 5.25,
						"carbohydrates_serving": 27.1,
						"fat_serving": 2.6,
						"fiber_serving": 4.0,
						"energy-kcal_100g": 380
					}
				}]
			}`))
		case "tempe goreng":
			_, _ = w.Write([]byte(`{
				"products": [{
					"product_name": "Tempe Goreng",
					"nutriments": {
						"energy-kcal_100g": 225.4,
						"proteins_100g": 18.2,
						"carbohydrates_100g": 9.4,
						"fat_100g": 12.1,
						"fiber_100g": 3.5
					}
				}]
			}`))
		default:
			_, _ = w.Write([]byte(`{"products": []}`))
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		Mode: config.ModeLocal,
		Food: config.FoodAPI{
			BaseURL: srv.URL,
			APIKey:  "test-secret",
		},
	}
	client, err := NewOpenFoodFactsClient(cfg, nil)
	if err != nil {
		t.Fatalf("NewOpenFoodFactsClient error: %v", err)
	}

	// Serving values preferred over 100g.
	m, found, err := client.Lookup(context.Background(), "  Oatmeal   Instan ")
	if err != nil || !found || m == nil {
		t.Fatalf("expected hit, got data=%v found=%v err=%v", m, found, err)
	}
	if m.Calories != 150 || m.ProteinG != 5.3 || m.Source != "food_api" {
		t.Fatalf("unexpected macro data: %+v", m)
	}

	// Second lookup should hit cache (no extra HTTP call).
	m2, found2, err2 := client.Lookup(context.Background(), "oatmeal instan")
	if err2 != nil || !found2 || m2.Calories != 150 {
		t.Fatalf("expected cache hit, got data=%v found=%v err=%v", m2, found2, err2)
	}
	if hits.Load() != 1 {
		t.Fatalf("expected 1 upstream hit due to cache, got %d", hits.Load())
	}

	// Fallback to 100g when serving is absent.
	m3, found3, err3 := client.Lookup(context.Background(), "Tempe Goreng")
	if err3 != nil || !found3 || m3.Calories != 225 || m3.ProteinG != 18.2 {
		t.Fatalf("expected 100g fallback hit, got data=%+v found=%v err=%v", m3, found3, err3)
	}

	// Miss returns (nil, false, nil) and is cached.
	m4, found4, err4 := client.Lookup(context.Background(), "nasi pecel spesial")
	if err4 != nil || found4 || m4 != nil {
		t.Fatalf("expected miss, got data=%v found=%v err=%v", m4, found4, err4)
	}
	_, found5, err5 := client.Lookup(context.Background(), "nasi pecel spesial")
	if err5 != nil || found5 {
		t.Fatalf("expected cached miss, got found=%v err=%v", found5, err5)
	}
	if hits.Load() != 3 {
		t.Fatalf("expected 3 total upstream hits, got %d", hits.Load())
	}
}

func TestOpenFoodFactsClient_RedirectBlockedAndErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("search_terms") {
		case "redirect":
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/", http.StatusFound)
		case "servererror":
			w.WriteHeader(http.StatusInternalServerError)
		case "oversized":
			w.Header().Set("Content-Type", "application/json")
			// Write > 1 MiB of invalid trailing JSON so LimitReader caps it cleanly.
			_, _ = w.Write([]byte(`{"products":[{"product_name":"` + strings.Repeat("a", foodAPIMaxBodyBytes+1024) + `"}]}`))
		}
	}))
	defer srv.Close()

	client, err := NewOpenFoodFactsClient(&config.Config{
		Mode: config.ModeLocal,
		Food: config.FoodAPI{BaseURL: srv.URL},
	}, nil)
	if err != nil {
		t.Fatalf("NewOpenFoodFactsClient error: %v", err)
	}

	if _, found, err := client.Lookup(context.Background(), "redirect"); err == nil || found {
		t.Fatalf("expected redirect to fail, got found=%v err=%v", found, err)
	}
	if _, found, err := client.Lookup(context.Background(), "servererror"); err == nil || found {
		t.Fatalf("expected 500 to return error, got found=%v err=%v", found, err)
	}
	if _, found, err := client.Lookup(context.Background(), "oversized"); err == nil || found {
		t.Fatalf("expected truncated >1MiB JSON to fail cleanly, got found=%v err=%v", found, err)
	}
}

func TestOpenFoodFactsClient_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
			_, _ = w.Write([]byte(`{"products":[]}`))
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	client, err := NewOpenFoodFactsClient(&config.Config{
		Mode: config.ModeLocal,
		Food: config.FoodAPI{BaseURL: srv.URL},
	}, nil)
	if err != nil {
		t.Fatalf("NewOpenFoodFactsClient error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if _, found, err := client.Lookup(ctx, "slow item"); err == nil || found {
		t.Fatalf("expected timeout error, got found=%v err=%v", found, err)
	}
}

func TestOpenFoodFactsClient_CacheBoundedEvictionAndTTL(t *testing.T) {
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)
	client := &OpenFoodFactsClient{
		cache:    make(map[string]foodCacheEntry),
		maxItems: 2,
		ttl:      10 * time.Minute,
		nowFunc:  func() time.Time { return now },
	}

	client.cacheSet("a", &MacroData{FoodName: "a", Calories: 100}, true)
	now = now.Add(time.Minute)
	client.cacheSet("b", &MacroData{FoodName: "b", Calories: 200}, true)
	now = now.Add(time.Minute)
	client.cacheSet("c", &MacroData{FoodName: "c", Calories: 300}, true)

	if len(client.cache) != 2 {
		t.Fatalf("expected cache bounded at 2, got %d", len(client.cache))
	}
	if _, _, ok := client.cacheGet("a"); ok {
		t.Fatal("expected oldest entry 'a' to be evicted")
	}
	if d, found, ok := client.cacheGet("b"); !ok || !found || d.Calories != 200 {
		t.Fatalf("expected 'b' in cache, got d=%v found=%v ok=%v", d, found, ok)
	}

	// Advance past TTL -> expired.
	now = now.Add(15 * time.Minute)
	if _, _, ok := client.cacheGet("b"); ok {
		t.Fatal("expected expired entry 'b' to miss")
	}
}

func TestValidateFoodAPIBaseURLAndSSRF(t *testing.T) {
	if _, err := validateFoodAPIBaseURL("http://world.openfoodfacts.org", true); err == nil {
		t.Fatal("expected http to be rejected in production")
	}
	if _, err := validateFoodAPIBaseURL("https://169.254.169.254", true); err == nil {
		t.Fatal("expected metadata IP to be rejected in production")
	}
	if _, err := validateFoodAPIBaseURL("http://example.com", false); err == nil {
		t.Fatal("expected non-loopback http to be rejected even in local mode")
	}
	if _, err := validateFoodAPIBaseURL("http://127.0.0.1:8089", false); err != nil {
		t.Fatalf("expected loopback http to be allowed in local mode: %v", err)
	}
	if _, err := validateFoodAPIBaseURL("https://world.openfoodfacts.org", true); err != nil {
		t.Fatalf("expected public https URL to be allowed in production: %v", err)
	}

	blocked := []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.5.4",
		"192.168.1.1",
		"169.254.169.254",
		"100.64.0.1",
		"100.127.255.254",
		"::ffff:100.64.0.1",
		"255.255.255.255",
		"::1",
		"fc00::1",
		"fe80::1",
		"::ffff:127.0.0.1",
		"::ffff:169.254.169.254",
		"0.0.0.0",
	}
	for _, raw := range blocked {
		ip := netip.MustParseAddr(raw)
		if !isBlockedIP(ip) {
			t.Errorf("expected IP %s to be blocked", raw)
		}
	}

	allowed := []string{
		"1.1.1.1",
		"8.8.8.8",
		"213.36.253.214",
		"2606:4700:4700::1111",
	}
	for _, raw := range allowed {
		ip := netip.MustParseAddr(raw)
		if isBlockedIP(ip) {
			t.Errorf("expected public IP %s to be allowed", raw)
		}
	}
}
