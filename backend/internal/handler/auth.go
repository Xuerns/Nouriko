package handler

import (
	"log/slog"

	"Nouriko/internal/auth"

	"github.com/gofiber/fiber/v3"
)

// AuthHandler handles authentication and account security endpoints.
type AuthHandler struct {
	service auth.Service
	logger  *slog.Logger
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(service auth.Service, logger *slog.Logger) *AuthHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuthHandler{
		service: service,
		logger:  logger,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Language string `json:"language"`
	Theme    string `json:"theme"`
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	res, err := h.service.Register(c.Context(), auth.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		Language: req.Language,
		Theme:    req.Theme,
	})
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

type verifyEmailRequest struct {
	Token string `json:"token"`
}

// VerifyEmail handles POST /api/v1/auth/verify-email.
func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req verifyEmailRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	if err := h.service.VerifyEmail(c.Context(), req.Token); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "verified"})
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req loginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	res, err := h.service.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"access_token": res.AccessToken,
		"token_type":   res.TokenType,
		"expires_in":   res.ExpiresIn,
		"profile":      ToProfileDTO(res.Profile, res.Email),
	})
}

// Logout handles POST /api/v1/auth/logout (auth required).
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	if _, ok := auth.ClaimsFromContext(c); !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	if err := h.service.Logout(c.Context(), extractBearerToken(c)); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "logged_out"})
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// ForgotPassword handles POST /api/v1/auth/forgot-password.
func (h *AuthHandler) ForgotPassword(c fiber.Ctx) error {
	var req forgotPasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	if err := h.service.ForgotPassword(c.Context(), req.Email); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "ok"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// ResetPassword handles POST /api/v1/auth/reset-password.
func (h *AuthHandler) ResetPassword(c fiber.Ctx) error {
	var req resetPasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	if err := h.service.ResetPassword(c.Context(), req.Token, req.NewPassword); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "password_reset"})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles POST /api/v1/auth/change-password (auth required).
func (h *AuthHandler) ChangePassword(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	var req changePasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	if err := h.service.ChangePassword(
		c.Context(),
		claims.UserID,
		claims.Email,
		extractBearerToken(c),
		req.CurrentPassword,
		req.NewPassword,
	); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "password_changed"})
}

type deleteAccountRequest struct {
	Password string `json:"password"`
}

// DeleteAccount handles DELETE /api/v1/account (auth required, PRD F13).
func (h *AuthHandler) DeleteAccount(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	var req deleteAccountRequest
	if err := c.Bind().JSON(&req); err != nil {
		return writeErrorCode(c, fiber.StatusBadRequest, "INVALID_INPUT")
	}

	if err := h.service.DeleteAccount(c.Context(), claims.UserID, claims.Email, req.Password); err != nil {
		return handleServiceError(c, h.logger, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"status": "account_deleted"})
}
