package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"strings"
	"unicode/utf8"

	"Nouriko/internal/ai"
	"Nouriko/internal/auth"
	"Nouriko/internal/database"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

const (
	maxChatMessageRunes  = 1500
	maxConversationTurns = 6
	maxTurnSummaryRunes  = 300
	maxPhotoBytes        = 5 * 1024 * 1024 // 5 MiB
	maxChatJSONBytes     = 64 * 1024       // 64 KiB
)

// ChatAnalyzer defines the service contract required by ChatHandler.
type ChatAnalyzer interface {
	Analyze(ctx context.Context, input ai.AnalyzeInput) (*ai.ChatAnalyzeResponse, error)
}

// ChatHandler handles POST /api/v1/chat/analyze requests (JSON and multipart photo).
type ChatHandler struct {
	repo    database.ProfileStore
	service ChatAnalyzer
	logger  *slog.Logger
}

// NewChatHandler constructs a ChatHandler.
func NewChatHandler(repo database.ProfileStore, service ChatAnalyzer, logger *slog.Logger) *ChatHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &ChatHandler{
		repo:    repo,
		service: service,
		logger:  logger,
	}
}

type chatJSONRequest struct {
	Message             string                `json:"message"`
	Language            string                `json:"language"`
	ConversationContext []ai.ConversationTurn `json:"conversation_context"`
}

// Analyze handles POST /api/v1/chat/analyze (auth + onboarding required).
func (h *ChatHandler) Analyze(c fiber.Ctx) error {
	claims, ok := auth.ClaimsFromContext(c)
	if !ok {
		return writeErrorCode(c, fiber.StatusUnauthorized, "UNAUTHORIZED")
	}

	profile, err := h.repo.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return writeErrorCode(c, fiber.StatusForbidden, "ONBOARDING_REQUIRED")
		}
		return handleServiceError(c, h.logger, err)
	}
	if !profile.OnboardingCompleted() {
		return writeErrorCode(c, fiber.StatusForbidden, "ONBOARDING_REQUIRED")
	}

	parsed, statusCode, errCode := parseChatRequest(c, profile.Language)
	if errCode != "" {
		return writeErrorCode(c, statusCode, errCode)
	}
	parsed.Profile = profile

	if h.service == nil {
		return writeErrorCode(c, fiber.StatusServiceUnavailable, "AI_UNAVAILABLE")
	}

	resp, err := h.service.Analyze(c.Context(), parsed)
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrUnavailable):
			return writeErrorCode(c, fiber.StatusServiceUnavailable, "AI_UNAVAILABLE")
		case errors.Is(err, ai.ErrPhotoUnrecognized):
			return writeErrorCode(c, fiber.StatusUnprocessableEntity, "PHOTO_UNRECOGNIZED")
		case errors.Is(err, context.DeadlineExceeded):
			return writeErrorCode(c, fiber.StatusGatewayTimeout, "AI_TIMEOUT")
		default:
			h.logger.Error("chat analyze failed",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
			return writeErrorCode(c, fiber.StatusInternalServerError, "INTERNAL_ERROR")
		}
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func parseChatRequest(c fiber.Ctx, profileLang string) (ai.AnalyzeInput, int, string) {
	var (
		message     string
		language    string
		convTurns   []ai.ConversationTurn
		photoBytes  []byte
		photoMIME   string
		contentType = strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	)

	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, err := c.MultipartForm()
		if err != nil {
			return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
		}
		defer func() { _ = form.RemoveAll() }()
		if vals := form.Value["message"]; len(vals) > 0 {
			message = vals[0]
		}
		if vals := form.Value["language"]; len(vals) > 0 {
			language = vals[0]
		}
		if vals := form.Value["conversation_context"]; len(vals) > 0 && strings.TrimSpace(vals[0]) != "" {
			if len(vals[0]) > maxChatJSONBytes {
				return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
			}
			if err := json.Unmarshal([]byte(vals[0]), &convTurns); err != nil {
				return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
			}
		}
		if files := form.File["photo"]; len(files) > 0 {
			if len(files) > 1 {
				return ai.AnalyzeInput{}, fiber.StatusBadRequest, "PHOTO_INVALID"
			}
			b, mime, err := readAndValidatePhoto(files[0])
			if err != nil {
				return ai.AnalyzeInput{}, fiber.StatusBadRequest, "PHOTO_INVALID"
			}
			photoBytes = b
			photoMIME = mime
		}
	} else {
		if len(c.Body()) > maxChatJSONBytes {
			return ai.AnalyzeInput{}, fiber.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE"
		}
		var body chatJSONRequest
		if err := c.Bind().JSON(&body); err != nil {
			return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
		}
		message = body.Message
		language = body.Language
		convTurns = body.ConversationContext
	}

	message = strings.TrimSpace(message)
	if message == "" && len(photoBytes) == 0 {
		return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
	}
	if utf8.RuneCountInString(message) > maxChatMessageRunes {
		return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
	}

	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		language = strings.ToLower(strings.TrimSpace(profileLang))
	}
	if language == "" {
		language = "id"
	}
	if language != "id" && language != "en" {
		return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
	}

	if len(convTurns) > maxConversationTurns {
		return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
	}
	cleanTurns := make([]ai.ConversationTurn, 0, len(convTurns))
	for _, t := range convTurns {
		role := strings.ToLower(strings.TrimSpace(t.Role))
		if role != "user" && role != "assistant" {
			return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
		}
		summary := strings.TrimSpace(t.Summary)
		if utf8.RuneCountInString(summary) > maxTurnSummaryRunes {
			return ai.AnalyzeInput{}, fiber.StatusBadRequest, "INVALID_INPUT"
		}
		if summary != "" {
			cleanTurns = append(cleanTurns, ai.ConversationTurn{
				Role:    role,
				Summary: summary,
			})
		}
	}

	return ai.AnalyzeInput{
		Message:             message,
		PhotoBytes:          photoBytes,
		PhotoMIMEType:       photoMIME,
		Language:            language,
		ConversationContext: cleanTurns,
	}, 0, ""
}

func readAndValidatePhoto(fh *multipart.FileHeader) ([]byte, string, error) {
	if fh == nil || fh.Size > maxPhotoBytes {
		return nil, "", errors.New("photo exceeds max size")
	}

	f, err := fh.Open()
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = f.Close() }()

	// Read strictly into memory with a hard cap (never written to disk or DB).
	data, err := io.ReadAll(io.LimitReader(f, maxPhotoBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 || len(data) > maxPhotoBytes {
		return nil, "", errors.New("photo empty or exceeds 5 MiB")
	}

	mime, ok := DetectPhotoMIME(data)
	if !ok {
		return nil, "", errors.New("unsupported photo format")
	}
	return data, mime, nil
}

// DetectPhotoMIME validates image format strictly by magic bytes (JPEG, PNG, WebP),
// ignoring client-supplied Content-Type headers and file extensions.
func DetectPhotoMIME(b []byte) (string, bool) {
	if bytes.HasPrefix(b, []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg", true
	}
	if bytes.HasPrefix(b, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		return "image/png", true
	}
	if len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP" {
		return "image/webp", true
	}
	return "", false
}
