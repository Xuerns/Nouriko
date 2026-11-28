package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"Nouriko/internal/auth"
	"Nouriko/internal/chat"
	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/meals"
	"Nouriko/internal/nutrition"
	"Nouriko/internal/profile"
	"Nouriko/internal/server"

	"github.com/gofiber/fiber/v3"
	"github.com/joho/godotenv"
)

const defaultPort = "8080"

func main() {
	// Allows the distroless container to run its own healthcheck without curl/wget.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(portFromEnv()); err != nil {
			slog.Error("healthcheck failed", "error", err)
			os.Exit(1)
		}
		return
	}

	// Load .env if present (useful for local non-Docker development); ignored in containers/prod.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	if err := run(cfg, logger); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if cfg.Database.MigrateOnStart != nil && *cfg.Database.MigrateOnStart {
		logger.Info("applying database migrations")
		if err := db.RunMigrations(ctx); err != nil {
			return err
		}
	}

	verifier, err := auth.NewVerifier(ctx, cfg)
	if err != nil {
		return err
	}

	geminiClient, err := chat.NewGeminiClient(ctx, cfg)
	if err != nil {
		return err
	}

	var foodLookup nutrition.FoodLookup
	if strings.TrimSpace(cfg.Food.BaseURL) != "" {
		foodClient, err := nutrition.NewOpenFoodFactsClient(cfg)
		if err != nil {
			return err
		}
		foodLookup = foodClient
	} else {
		logger.Info("food api enrichment disabled (FOOD_API_BASE_URL not set)")
	}

	profileRepo := profile.NewStore(db.Pool)
	localAuthRepo := auth.NewLocalAuthRepo(db.Pool)
	foodPriceRepo := chat.NewFoodPriceRepo(db.Pool)
	mealRepo := meals.NewStore(db.Pool)

	adminClient := auth.NewAdminClient(cfg)
	var googleVerifier auth.GoogleTokenVerifier
	if cfg.GoogleAuthEnabled() {
		googleVerifier = auth.NewGoogleJWKSVerifier(ctx, cfg.Auth.GoogleClientID)
	}
	authService := auth.NewService(cfg, localAuthRepo, profileRepo, adminClient, googleVerifier, logger)

	app := server.New(server.Dependencies{
		Config:        cfg,
		Logger:        logger,
		DB:            db,
		ProfileRepo:   profileRepo,
		Auth:          verifier,
		AuthService:   authService,
		FoodPriceRepo: foodPriceRepo,
		Gemini:        geminiClient,
		FoodLookup:    foodLookup,
		MealRepo:      mealRepo,
	})

	errCh := make(chan error, 1)
	go func() {
		addr := ":" + cfg.Port
		logger.Info("server listening", "addr", addr, "mode", cfg.Mode)
		if err := app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true}); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	logger.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return app.ShutdownWithContext(shutdownCtx)
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

func runHealthcheck(p string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:"+p+"/healthz", nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req) // #nosec G704 -- loopback healthcheck against local container port
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return errors.New("unexpected status: " + resp.Status)
	}
	return nil
}

func portFromEnv() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}
