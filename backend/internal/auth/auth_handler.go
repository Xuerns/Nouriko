package auth

import (
	"errors"
	"log/slog"

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"
	"Nouriko/internal/profile"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// Handler handles authentication and account security HTTP endpoints.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler creates a new auth Handler.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) handleAuthError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	case errors.Is(err, ErrWeakPassword):
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeWeakPassword)
	case errors.Is(err, ErrInvalidOrExpiredToken), errors.Is(err, database.ErrTokenInvalidOrExpired):
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeLinkInvalidOrExpired)
	case errors.Is(err, ErrInvalidCredentials):
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeAuthFailed)
	case errors.Is(err, ErrUnauthorized):
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	case errors.Is(err, ErrEmailNotVerified):
		return httpx.WriteError(c, fiber.StatusForbidden, httpx.CodeEmailNotVerified)
	case errors.Is(err, ErrNotFound), errors.Is(err, database.ErrNotFound):
		return httpx.WriteError(c, fiber.StatusNotFound, httpx.CodeNotFound)
	case errors.Is(err, ErrEmailAlreadyExists), errors.Is(err, database.ErrConflict):
		return httpx.WriteError(c, fiber.StatusConflict, httpx.CodeEmailAlreadyExists)
	case errors.Is(err, ErrTooManyRequests):
		return httpx.WriteError(c, fiber.StatusTooManyRequests, httpx.CodeTooManyRequests)
	default:
		if h.logger != nil {
			h.logger.Error("internal auth handler error",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
		}
		return httpx.WriteError(c, fiber.StatusInternalServerError, httpx.CodeInternalError)
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	res, err := h.service.Register(c.Context(), RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Language: req.Language,
		Theme:    req.Theme,
	})
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *Handler) VerifyEmail(c fiber.Ctx) error {
	var req verifyEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.service.VerifyEmail(c.Context(), req.Token); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "verified"})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	res, err := h.service.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": res.AccessToken,
		"token_type":   res.TokenType,
		"expires_in":   res.ExpiresIn,
		"profile":      profile.ToDTO(res.Profile, res.Email),
	})
}

// Logout handles POST /api/v1/auth/logout (auth required).
func (h *Handler) Logout(c fiber.Ctx) error {
	if _, ok := httpx.ClaimsFromContext(c); !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	if err := h.service.Logout(c.Context(), httpx.ExtractBearerToken(c)); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "logged_out"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
func (h *Handler) ForgotPassword(c fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.service.ForgotPassword(c.Context(), req.Email); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword handles POST /api/v1/auth/reset-password.
func (h *Handler) ResetPassword(c fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.service.ResetPassword(c.Context(), req.Token, req.NewPassword); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "password_reset"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles POST /api/v1/auth/change-password (auth required).
func (h *Handler) ChangePassword(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	var req changePasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.service.ChangePassword(
		c.Context(),
		claims.UserID,
		claims.Email,
		httpx.ExtractBearerToken(c),
		req.CurrentPassword,
		req.NewPassword,
	); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "password_changed"})
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount handles DELETE /api/v1/account (auth required, PRD F13).
func (h *Handler) DeleteAccount(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	var req deleteAccountRequest
	if err := c.Bind().JSON(&req); err != nil {
		return httpx.WriteError(c, fiber.StatusBadRequest, httpx.CodeInvalidInput)
	}

	if err := h.service.DeleteAccount(c.Context(), claims.UserID, claims.Email, req.Password); err != nil {
		return h.handleAuthError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "account_deleted"})
}
