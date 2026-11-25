package config

import (
	"strings"
	"testing"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoadLocalDefaults(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":         "local",
		"DATABASE_URL":    "postgres://u:p@localhost:5432/db?sslmode=disable",
		"AUTH_JWT_SECRET": testSecret,
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Mode != ModeLocal {
		t.Fatalf("expected local mode, got %q", cfg.Mode)
	}
	if !*cfg.Database.MigrateOnStart {
		t.Fatal("expected migrations to run on start in local mode")
	}
	if cfg.IsProduction() {
		t.Fatal("local mode must not report production")
	}
	if cfg.AppBaseURL != "http://localhost:3000" {
		t.Fatalf("expected default AppBaseURL, got %q", cfg.AppBaseURL)
	}
	if cfg.AuthLimits.Max != 15 {
		t.Fatalf("expected default AuthLimits.Max=15, got %d", cfg.AuthLimits.Max)
	}
	if cfg.AILimits.Max != 20 {
		t.Fatalf("expected default AILimits.Max=20, got %d", cfg.AILimits.Max)
	}
}

func TestLoadLocalRejectsShortSecret(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":         "local",
		"DATABASE_URL":    "postgres://u:p@localhost:5432/db",
		"AUTH_JWT_SECRET": "short",
	})

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "AUTH_JWT_SECRET") {
		t.Fatalf("expected AUTH_JWT_SECRET error, got %v", err)
	}
}

func TestLoadProduction(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DATABASE_URL":         "postgres://u:p@pooler.supabase.com:6543/postgres?sslmode=require",
		"SUPABASE_URL":         "https://abc.supabase.co/",
		"SUPABASE_SECRET_KEY":  "sb_secret_x",
		"GEMINI_API_KEY":       "key",
		"CORS_ALLOWED_ORIGINS": "https://nouriko.app",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Auth.Issuer != "https://abc.supabase.co/auth/v1" {
		t.Fatalf("unexpected issuer: %s", cfg.Auth.Issuer)
	}
	if cfg.JWKSURL() != "https://abc.supabase.co/auth/v1/.well-known/jwks.json" {
		t.Fatalf("unexpected JWKS URL: %s", cfg.JWKSURL())
	}
	if *cfg.Database.MigrateOnStart {
		t.Fatal("expected migrations to be disabled on start in production mode")
	}
}

func TestLoadProductionRequiresSupabaseSettings(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"DATABASE_URL":         "postgres://u:p@db:5432/postgres?sslmode=disable",
		"CORS_ALLOWED_ORIGINS": "*",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{"SUPABASE_URL", "SUPABASE_SECRET_KEY", "GEMINI_API_KEY", "sslmode", "CORS_ALLOWED_ORIGINS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("expected error to mention %s, got: %v", want, err)
		}
	}
}

func TestLoadRejectsUnknownMode(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":      "staging",
		"DATABASE_URL": "postgres://u:p@localhost:5432/db",
	})

	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "APP_ENV") {
		t.Fatalf("expected APP_ENV error, got %v", err)
	}
}

func TestSupabaseAuthAPIKeyFallbackAndAppBaseURLTrim(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":              "production",
		"APP_BASE_URL":         "https://nouriko.app/",
		"DATABASE_URL":         "postgres://u:p@pooler.supabase.com:6543/postgres?sslmode=require",
		"SUPABASE_URL":         "https://abc.supabase.co/",
		"SUPABASE_SECRET_KEY":  "sb_secret_x",
		"GEMINI_API_KEY":       "key",
		"CORS_ALLOWED_ORIGINS": "https://nouriko.app",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppBaseURL != "https://nouriko.app" {
		t.Fatalf("expected trimmed AppBaseURL, got %q", cfg.AppBaseURL)
	}
	if cfg.Supabase.AuthAPIKey() != "sb_secret_x" {
		t.Fatalf("expected fallback to SecretKey, got %q", cfg.Supabase.AuthAPIKey())
	}

	t.Setenv("SUPABASE_ANON_KEY", "sb_anon_y")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Supabase.AuthAPIKey() != "sb_anon_y" {
		t.Fatalf("expected AnonKey preference, got %q", cfg.Supabase.AuthAPIKey())
	}
}

func TestVercelOriginDefaults(t *testing.T) {
	setEnv(t, map[string]string{
		"APP_ENV":                       "production",
		"DATABASE_URL":                  "postgres://u:p@pooler.supabase.com:6543/postgres?sslmode=require",
		"SUPABASE_URL":                  "https://abc.supabase.co",
		"SUPABASE_SECRET_KEY":           "sb_secret_x",
		"GEMINI_API_KEY":                "key",
		"VERCEL_ENV":                    "production",
		"VERCEL_PROJECT_PRODUCTION_URL": "nouriko.vercel.app",
		"VERCEL_URL":                    "nouriko-abc123.vercel.app",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppBaseURL != "https://nouriko.vercel.app" {
		t.Fatalf("expected production Vercel AppBaseURL, got %q", cfg.AppBaseURL)
	}
	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "https://nouriko.vercel.app" || cfg.CORSOrigins[1] != "https://nouriko-abc123.vercel.app" {
		t.Fatalf("unexpected Vercel CORSOrigins: %v", cfg.CORSOrigins)
	}

	t.Setenv("VERCEL_ENV", "preview")
	cfg, err = Load()
	if err != nil {
		t.Fatalf("unexpected error in preview: %v", err)
	}
	if cfg.AppBaseURL != "https://nouriko-abc123.vercel.app" {
		t.Fatalf("expected preview Vercel AppBaseURL, got %q", cfg.AppBaseURL)
	}
}
