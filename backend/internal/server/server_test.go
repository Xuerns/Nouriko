package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"Nouriko/internal/config"
)

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

func testConfig() *config.Config {
	return &config.Config{
		Mode:        config.ModeLocal,
		Port:        "8080",
		CORSOrigins: []string{"http://localhost:3000"},
		Limits: config.RateLimit{
			Max:    60,
			Window: time.Minute,
		},
	}
}

func TestHealthz(t *testing.T) {
	app := New(Dependencies{
		Config: testConfig(),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" || body["mode"] != "local" {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestReadyzHealthyAndUnhealthy(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	okApp := New(Dependencies{Config: testConfig(), Logger: logger, DB: stubPinger{}})
	resp, err := okApp.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 when db healthy, got %d", resp.StatusCode)
	}

	failApp := New(Dependencies{Config: testConfig(), Logger: logger, DB: stubPinger{err: errors.New("db down")}})
	resp, err = failApp.Test(httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when db down, got %d", resp.StatusCode)
	}
}

func TestNotFoundUsesGenericCode(t *testing.T) {
	app := New(Dependencies{
		Config: testConfig(),
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
		DB:     stubPinger{},
	})

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "NOT_FOUND" {
		t.Fatalf("expected generic NOT_FOUND code, got %+v", body)
	}
}
