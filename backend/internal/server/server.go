// Package server configures the Fiber v3 HTTP server, middleware stack,
// health endpoints, and PRD-compliant error handling.
package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"Nouriko/internal/ai"
	"Nouriko/internal/auth"
	"Nouriko/internal/config"
	"Nouriko/internal/database"
	"Nouriko/internal/handler"

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

// Dependencies groups runtime collaborators injected into the HTTP server.
type Dependencies struct {
	Config      *config.Config
	Logger      *slog.Logger
	DB          Pinger
	Repo        database.Repository
	Auth        *auth.Verifier
	AuthAdmin   *auth.AdminClient
	AuthService auth.Service
	Gemini      *ai.Client
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
		WriteTimeout: 30 * time.Second,
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
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"code": "SERVICE_UNAVAILABLE"})
		}
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()
		if err := deps.DB.Ping(ctx); err != nil {
			logger.Warn("readiness check failed", "error", err)
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"code": "SERVICE_UNAVAILABLE"})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ready"})
	})

	// Rate-limited API group for application routes.
	api := app.Group("/api/v1", limiter.New(limiter.Config{
		Max:        deps.Config.Limits.Max,
		Expiration: deps.Config.Limits.Window,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"code": "TOO_MANY_REQUESTS"})
		},
	}))

	api.Get("/status", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":     "ok",
			"mode":       deps.Config.Mode,
			"ai_enabled": deps.Gemini != nil,
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
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"code": "TOO_MANY_REQUESTS"})
		},
	})

	if deps.AuthService != nil {
		authHandler := handler.NewAuthHandler(deps.AuthService, logger)
		api.Post("/auth/register", authStrictLimiter, authHandler.Register)
		api.Post("/auth/verify-email", authStrictLimiter, authHandler.VerifyEmail)
		api.Post("/auth/login", authStrictLimiter, authHandler.Login)
		api.Post("/auth/forgot-password", authStrictLimiter, authHandler.ForgotPassword)
		api.Post("/auth/reset-password", authStrictLimiter, authHandler.ResetPassword)

		if deps.Auth != nil {
			authMW := auth.Middleware(deps.Auth)
			api.Post("/auth/logout", authMW, authHandler.Logout)
			api.Post("/auth/change-password", authStrictLimiter, authMW, authHandler.ChangePassword)
			api.Delete("/account", authStrictLimiter, authMW, authHandler.DeleteAccount)
		}
	}

	if deps.Repo != nil && deps.Auth != nil {
		profileHandler := handler.NewProfileHandler(deps.Repo, logger)
		authMW := auth.Middleware(deps.Auth)
		api.Get("/profile", authMW, profileHandler.GetProfile)
		api.Put("/profile/onboarding", authMW, profileHandler.CompleteOnboarding)
		api.Patch("/profile", authMW, profileHandler.UpdateProfile)
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
		code := "INTERNAL_ERROR"

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			status = fiberErr.Code
			switch status {
			case fiber.StatusBadRequest:
				code = "INVALID_INPUT"
			case fiber.StatusUnauthorized:
				code = "UNAUTHORIZED"
			case fiber.StatusForbidden:
				code = "FORBIDDEN"
			case fiber.StatusNotFound:
				code = "NOT_FOUND"
			case fiber.StatusMethodNotAllowed:
				code = "METHOD_NOT_ALLOWED"
			case fiber.StatusRequestEntityTooLarge:
				code = "PAYLOAD_TOO_LARGE"
			case fiber.StatusTooManyRequests:
				code = "TOO_MANY_REQUESTS"
			default:
				code = "INTERNAL_ERROR"
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

		return c.Status(status).JSON(fiber.Map{"code": code})
	}
}
