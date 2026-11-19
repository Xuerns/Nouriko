// Package config loads and validates runtime configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
)

// Mode selects which infrastructure the backend talks to.
type Mode string

const (
	// ModeLocal uses the PostgreSQL container from docker-compose and HS256 dev tokens.
	ModeLocal Mode = "local"
	// ModeProduction uses Supabase Postgres and Supabase Auth (JWKS).
	ModeProduction Mode = "production"
)

// Config holds every setting the backend needs at runtime.
type Config struct {
	Mode            Mode          `env:"APP_ENV" envDefault:"local"`
	Port            string        `env:"PORT" envDefault:"8080"`
	AppBaseURL      string        `env:"APP_BASE_URL" envDefault:"http://localhost:3000"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
	CORSOrigins     []string      `env:"CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://localhost:3000"`

	Database   Database
	Auth       Auth
	Supabase   Supabase
	Gemini     Gemini
	Food       FoodAPI
	Limits     RateLimit
	AuthLimits AuthRateLimit
	AILimits   AIRateLimit
}

// Database configures the PostgreSQL connection pool.
type Database struct {
	URL            string `env:"DATABASE_URL"`
	MaxConns       int32  `env:"DATABASE_MAX_CONNS" envDefault:"10"`
	MigrateOnStart *bool  `env:"DATABASE_MIGRATE_ON_START"`
}

// Auth configures how incoming access tokens are verified.
type Auth struct {
	// JWTSecret is the HS256 secret. Required in local mode, optional in
	// production for Supabase projects still on the legacy JWT secret.
	JWTSecret string `env:"AUTH_JWT_SECRET"`
	Audience  string `env:"AUTH_JWT_AUDIENCE" envDefault:"authenticated"`
	Issuer    string `env:"AUTH_JWT_ISSUER"`
}

// Supabase holds project settings used in production mode.
type Supabase struct {
	URL       string `env:"SUPABASE_URL"`
	AnonKey   string `env:"SUPABASE_ANON_KEY"`
	SecretKey string `env:"SUPABASE_SECRET_KEY"`
}

// AuthAPIKey returns the key used for user-scoped Supabase Auth REST calls,
// preferring SUPABASE_ANON_KEY and falling back to SUPABASE_SECRET_KEY.
func (s Supabase) AuthAPIKey() string {
	if s.AnonKey != "" {
		return s.AnonKey
	}
	return s.SecretKey
}

// Gemini configures the Gemini API client.
type Gemini struct {
	APIKey  string        `env:"GEMINI_API_KEY"`
	Model   string        `env:"GEMINI_MODEL" envDefault:"gemini-2.5-flash"`
	Timeout time.Duration `env:"GEMINI_TIMEOUT" envDefault:"15s"`
}

// FoodAPI configures the nutrition data provider (provider not decided yet).
type FoodAPI struct {
	BaseURL string `env:"FOOD_API_BASE_URL"`
	APIKey  string `env:"FOOD_API_KEY"`
}

// RateLimit configures per-client request limits.
type RateLimit struct {
	Max    int           `env:"RATE_LIMIT_MAX" envDefault:"60"`
	Window time.Duration `env:"RATE_LIMIT_WINDOW" envDefault:"1m"`
}

// AuthRateLimit configures stricter per-client limits for sensitive auth endpoints.
type AuthRateLimit struct {
	Max    int           `env:"AUTH_RATE_LIMIT_MAX" envDefault:"15"`
	Window time.Duration `env:"AUTH_RATE_LIMIT_WINDOW" envDefault:"1m"`
}

// AIRateLimit configures per-user limits for AI analysis endpoints.
type AIRateLimit struct {
	Max    int           `env:"AI_RATE_LIMIT_MAX" envDefault:"20"`
	Window time.Duration `env:"AI_RATE_LIMIT_WINDOW" envDefault:"1m"`
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) applyDefaults() {
	c.AppBaseURL = strings.TrimRight(c.AppBaseURL, "/")
	c.Supabase.URL = strings.TrimRight(c.Supabase.URL, "/")
	if c.Auth.Issuer == "" && c.Mode == ModeProduction && c.Supabase.URL != "" {
		c.Auth.Issuer = c.Supabase.URL + "/auth/v1"
	}
	if c.Database.MigrateOnStart == nil {
		v := c.Mode == ModeLocal
		c.Database.MigrateOnStart = &v
	}
}

// Validate checks that the settings required by the selected mode are present.
func (c *Config) Validate() error {
	var errs []error

	if c.Database.URL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if c.Database.MaxConns < 1 {
		errs = append(errs, errors.New("DATABASE_MAX_CONNS must be at least 1"))
	}
	if c.Limits.Max < 1 || c.Limits.Window <= 0 {
		errs = append(errs, errors.New("RATE_LIMIT_MAX and RATE_LIMIT_WINDOW must be positive"))
	}
	if c.AuthLimits.Max < 1 || c.AuthLimits.Window <= 0 {
		errs = append(errs, errors.New("AUTH_RATE_LIMIT_MAX and AUTH_RATE_LIMIT_WINDOW must be positive"))
	}
	if c.AILimits.Max < 1 || c.AILimits.Window <= 0 {
		errs = append(errs, errors.New("AI_RATE_LIMIT_MAX and AI_RATE_LIMIT_WINDOW must be positive"))
	}

	switch c.Mode {
	case ModeLocal:
		if len(c.Auth.JWTSecret) < 32 {
			errs = append(errs, errors.New("AUTH_JWT_SECRET must be at least 32 characters in local mode"))
		}
	case ModeProduction:
		errs = append(errs, c.validateProduction()...)
	default:
		errs = append(errs, fmt.Errorf("APP_ENV must be %q or %q, got %q", ModeLocal, ModeProduction, c.Mode))
	}

	return errors.Join(errs...)
}

func (c *Config) validateProduction() []error {
	var errs []error

	u, err := url.Parse(c.Supabase.URL)
	if c.Supabase.URL == "" || err != nil || u.Scheme != "https" || u.Host == "" {
		errs = append(errs, errors.New("SUPABASE_URL must be a valid https URL in production mode"))
	}
	if c.Supabase.SecretKey == "" {
		errs = append(errs, errors.New("SUPABASE_SECRET_KEY is required in production mode"))
	}
	if c.Gemini.APIKey == "" {
		errs = append(errs, errors.New("GEMINI_API_KEY is required in production mode"))
	}
	if c.Auth.JWTSecret != "" && len(c.Auth.JWTSecret) < 32 {
		errs = append(errs, errors.New("AUTH_JWT_SECRET must be at least 32 characters when set"))
	}
	if c.Database.URL != "" && !hasTLS(c.Database.URL) {
		errs = append(errs, errors.New("DATABASE_URL must use sslmode=require (or stricter) in production mode"))
	}
	for _, o := range c.CORSOrigins {
		if o == "*" {
			errs = append(errs, errors.New("CORS_ALLOWED_ORIGINS must not be * in production mode"))
		}
	}

	return errs
}

// JWKSURL returns the Supabase Auth JWKS endpoint.
func (c *Config) JWKSURL() string {
	if c.Supabase.URL == "" {
		return ""
	}
	return c.Supabase.URL + "/auth/v1/.well-known/jwks.json"
}

// IsProduction reports whether the backend runs against Supabase.
func (c *Config) IsProduction() bool {
	return c.Mode == ModeProduction
}

func hasTLS(dsn string) bool {
	u, err := url.Parse(dsn)
	if err != nil {
		return false
	}
	switch u.Query().Get("sslmode") {
	case "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}
