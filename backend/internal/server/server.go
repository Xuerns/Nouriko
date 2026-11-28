// Package server configures the Fiber v3 HTTP server, middleware stack,
// health endpoints, context route wiring, and PRD-compliant error handling.
package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"Nouriko/internal/auth"
	"Nouriko/internal/chat"
	"Nouriko/internal/config"
	"Nouriko/internal/httpx"
	"Nouriko/internal/meals"
	"Nouriko/internal/nutrition"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Pinger checks readiness of a backing dependency such as PostgreSQL.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Dependencies groups runtime collaborators injected into the HTTP server per domain context.
type Dependencies struct {
	Config        *config.Config
	Logger        *slog.Logger
	DB            Pinger
	ProfileRepo   profile.Repository
	Auth          *auth.Verifier
	AuthService   auth.Service
	FoodPriceRepo chat.FoodPriceStore
	Gemini        *chat.Client
	Analyzer      chat.Analyzer
	FoodLookup    nutrition.FoodLookup
	MealRepo      meals.SavedMealStore
}

// New builds and configures the Fiber v3 application.
func New(deps Dependencies) *fiber.App {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	app := fiber.New(fiber.Config{
		AppName:      "Nouriko API",
		BodyLimit:    10 * 1024 * 1024, // 10 MiB for food photo uploads
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 75 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: newErrorHandler(logger),
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     deps.Config.CORSOrigins,
		AllowMethods:     []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete, fiber.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		AllowCredentials: true,
	}))
	app.Use(requestLogger(logger))

	// Health endpoints are exempt from rate limiting so container probes never fail.
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
			"mode":   deps.Config.Mode,
		})
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		if deps.DB == nil {
			return httpx.WriteError(c, fiber.StatusServiceUnavailable, httpx.CodeServiceUnavailable)
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		if err := deps.DB.Ping(ctx); err != nil {
			logger.Warn("readiness check failed", "error", err)
			return httpx.WriteError(c, fiber.StatusServiceUnavailable, httpx.CodeServiceUnavailable)
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ready"})
	})

	// Rate-limited API group for application routes.
	api := app.Group("/api/v1", limiter.New(limiter.Config{
		Max:        deps.Config.Limits.Max,
		Expiration: deps.Config.Limits.Window,
		LimitReached: func(c fiber.Ctx) error {
			return httpx.WriteError(c, fiber.StatusTooManyRequests, httpx.CodeTooManyRequests)
		},
	}))

	var activeAnalyzer chat.Analyzer
	if deps.Analyzer != nil {
		activeAnalyzer = deps.Analyzer
	} else if deps.Gemini != nil {
		activeAnalyzer = deps.Gemini
	}

	api.Get("/status", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":              "ok",
			"mode":                deps.Config.Mode,
			"ai_enabled":          activeAnalyzer != nil,
			"google_auth_enabled": deps.Config.GoogleAuthEnabled(),
		})
	})

	authMax := deps.Config.AuthLimits.Max
	authWindow := deps.Config.AuthLimits.Window
	if authMax < 1 {
		authMax = 15
	}
	if authWindow <= 0 {
		authWindow = time.Minute
	}
	authStrictLimiter := limiter.New(limiter.Config{
		Max:        authMax,
		Expiration: authWindow,
		LimitReached: func(c fiber.Ctx) error {
			return httpx.WriteError(c, fiber.StatusTooManyRequests, httpx.CodeTooManyRequests)
		},
	})

	if deps.AuthService != nil {
		authHandler := auth.NewHandler(deps.AuthService, logger)
		api.Post("/auth/register", authStrictLimiter, authHandler.Register)
		api.Post("/auth/verify-email", authStrictLimiter, authHandler.VerifyEmail)
		api.Post("/auth/login", authStrictLimiter, authHandler.Login)
		api.Post("/auth/google", authStrictLimiter, authHandler.GoogleLogin)
		api.Post("/auth/forgot-password", authStrictLimiter, authHandler.ForgotPassword)
		api.Post("/auth/reset-password", authStrictLimiter, authHandler.ResetPassword)

		if deps.Auth != nil {
			authMW := auth.Middleware(deps.Auth)
			api.Post("/auth/logout", authMW, authHandler.Logout)
			api.Post("/auth/change-password", authStrictLimiter, authMW, authHandler.ChangePassword)
			api.Delete("/account", authStrictLimiter, authMW, authHandler.DeleteAccount)
		}
	}

	if deps.Auth != nil {
		authMW := auth.Middleware(deps.Auth)

		if deps.ProfileRepo != nil {
			profileHandler := profile.NewHandler(deps.ProfileRepo, logger)
			api.Get("/profile", authMW, profileHandler.GetProfile)
			api.Put("/profile/onboarding", authMW, profileHandler.CompleteOnboarding)
			api.Patch("/profile", authMW, profileHandler.UpdateProfile)

			aiMax := deps.Config.AILimits.Max
			aiWindow := deps.Config.AILimits.Window
			if aiMax < 1 {
				aiMax = 20
			}
			if aiWindow <= 0 {
				aiWindow = time.Minute
			}
			aiUserLimiter := limiter.New(limiter.Config{
				Max:        aiMax,
				Expiration: aiWindow,
				KeyGenerator: func(c fiber.Ctx) string {
					if claims, ok := httpx.ClaimsFromContext(c); ok && claims.UserID != "" {
						return "ai:" + claims.UserID
					}
					return "ai:ip:" + c.IP()
				},
				LimitReached: func(c fiber.Ctx) error {
					return httpx.WriteError(c, fiber.StatusTooManyRequests, httpx.CodeTooManyRequests)
				},
			})

			chatService := chat.NewChatService(activeAnalyzer, deps.FoodPriceRepo, deps.FoodLookup, logger)
			chatHandler := chat.NewHandler(deps.ProfileRepo, chatService, logger)
			api.Post("/chat/analyze", authMW, aiUserLimiter, chatHandler.Analyze)

			if deps.MealRepo != nil {
				mealHandler := meals.NewHandler(deps.MealRepo, deps.ProfileRepo, logger)
				api.Post("/meals", authMW, mealHandler.CreateMeal)
				api.Get("/meals/summary", authMW, mealHandler.GetDailySummary)
				api.Get("/meals/history", authMW, mealHandler.GetMealHistory)
				api.Delete("/meals/:id", authMW, mealHandler.DeleteMeal)
			}
		}
	}

	return app
}

func requestLogger(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		logger.Info("http request",
			"request_id", requestid.FromContext(c),
			"method", c.Method(),
			"path", c.Path(),
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		return err
	}
}

// newErrorHandler returns generic error codes to clients and logs technical
// details only on the server (PRD "Aturan Pesan Error").
func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		code := httpx.CodeInternalError

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			status = fiberErr.Code
			switch status {
			case fiber.StatusBadRequest:
				code = httpx.CodeInvalidInput
			case fiber.StatusUnauthorized:
				code = httpx.CodeUnauthorized
			case fiber.StatusForbidden:
				code = httpx.CodeForbidden
			case fiber.StatusNotFound:
				code = httpx.CodeNotFound
			case fiber.StatusMethodNotAllowed:
				code = httpx.CodeMethodNotAllowed
			case fiber.StatusRequestEntityTooLarge:
				code = httpx.CodePayloadTooLarge
			case fiber.StatusTooManyRequests:
				code = httpx.CodeTooManyRequests
			default:
				code = httpx.CodeInternalError
			}
		}

		if status >= fiber.StatusInternalServerError {
			logger.Error("unhandled server error",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"status", status,
				"error", err,
			)
		}

		return httpx.WriteError(c, status, code)
	}
}
