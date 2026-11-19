package chat

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

	"Nouriko/internal/database"
	"Nouriko/internal/httpx"
	"Nouriko/internal/profile"

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

// ProfileGetter defines the minimal profile read contract required by Handler.
type ProfileGetter interface {
	GetProfileByID(ctx context.Context, userID string) (*profile.Profile, error)
}

// ChatAnalyzer defines the service contract required by Handler.
type ChatAnalyzer interface {
	Analyze(ctx context.Context, input AnalyzeInput) (*ChatAnalyzeResponse, error)
}

// Handler handles POST /api/v1/chat/analyze requests (JSON and multipart photo).
type Handler struct {
	profiles ProfileGetter
	service  ChatAnalyzer
	logger   *slog.Logger
}

// NewHandler constructs a chat Handler.
func NewHandler(profiles ProfileGetter, service ChatAnalyzer, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		profiles: profiles,
		service:  service,
		logger:   logger,
	}
}

type chatJSONRequest struct {
	Message             string             `json:"message"`
	Language            string             `json:"language"`
	ConversationContext []ConversationTurn `json:"conversation_context"`
}

// Analyze handles POST /api/v1/chat/analyze (auth + onboarding required).
func (h *Handler) Analyze(c fiber.Ctx) error {
	claims, ok := httpx.ClaimsFromContext(c)
	if !ok {
		return httpx.WriteError(c, fiber.StatusUnauthorized, httpx.CodeUnauthorized)
	}

	prof, err := h.profiles.GetProfileByID(c.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return httpx.WriteError(c, fiber.StatusForbidden, httpx.CodeOnboardingRequired)
		}
		return httpx.HandleDBError(c, h.logger, err)
	}
	if !prof.OnboardingCompleted() {
		return httpx.WriteError(c, fiber.StatusForbidden, httpx.CodeOnboardingRequired)
	}

	parsed, statusCode, errCode := parseChatRequest(c, prof.Language)
	if errCode != "" {
		return httpx.WriteError(c, statusCode, errCode)
	}
	parsed.Profile = prof

	if h.service == nil {
		return httpx.WriteError(c, fiber.StatusServiceUnavailable, httpx.CodeAIUnavailable)
	}

	resp, err := h.service.Analyze(c.Context(), parsed)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnavailable):
			h.logger.Warn("chat analyze unavailable",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
			return httpx.WriteError(c, fiber.StatusServiceUnavailable, httpx.CodeAIUnavailable)
		case errors.Is(err, ErrPhotoUnrecognized):
			return httpx.WriteError(c, fiber.StatusUnprocessableEntity, httpx.CodePhotoUnrecognized)
		case errors.Is(err, context.DeadlineExceeded):
			h.logger.Warn("chat analyze timeout",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
			return httpx.WriteError(c, fiber.StatusGatewayTimeout, httpx.CodeAITimeout)
		default:
			h.logger.Error("chat analyze failed",
				"request_id", requestid.FromContext(c),
				"path", c.Path(),
				"error", err,
			)
			return httpx.WriteError(c, fiber.StatusInternalServerError, httpx.CodeInternalError)
		}
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

func parseChatRequest(c fiber.Ctx, profileLang string) (AnalyzeInput, int, string) {
	var (
		message     string
		language    string
		convTurns   []ConversationTurn
		photoBytes  []byte
		photoMIME   string
		contentType = strings.ToLower(strings.TrimSpace(c.Get(fiber.HeaderContentType)))
	)

	if strings.HasPrefix(contentType, "multipart/form-data") {
		form, err := c.MultipartForm()
		if err != nil {
			return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
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
				return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
			}
			if err := json.Unmarshal([]byte(vals[0]), &convTurns); err != nil {
				return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
			}
		}
		if files := form.File["photo"]; len(files) > 0 {
			if len(files) > 1 {
				return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodePhotoInvalid
			}
			b, mime, err := readAndValidatePhoto(files[0])
			if err != nil {
				return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodePhotoInvalid
			}
			photoBytes = b
			photoMIME = mime
		}
	} else {
		if len(c.Body()) > maxChatJSONBytes {
			return AnalyzeInput{}, fiber.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge
		}
		var body chatJSONRequest
		if err := c.Bind().JSON(&body); err != nil {
			return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
		}
		message = body.Message
		language = body.Language
		convTurns = body.ConversationContext
	}

	message = strings.TrimSpace(message)
	if message == "" && len(photoBytes) == 0 {
		return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
	}
	if utf8.RuneCountInString(message) > maxChatMessageRunes {
		return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
	}

	language = strings.ToLower(strings.TrimSpace(language))
	if language == "" {
		language = strings.ToLower(strings.TrimSpace(profileLang))
	}
	if language == "" {
		language = "id"
	}
	if language != "id" && language != "en" {
		return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
	}

	if len(convTurns) > maxConversationTurns {
		return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
	}
	cleanTurns := make([]ConversationTurn, 0, len(convTurns))
	for _, t := range convTurns {
		role := strings.ToLower(strings.TrimSpace(t.Role))
		if role != "user" && role != "assistant" {
			return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
		}
		summary := strings.TrimSpace(t.Summary)
		if utf8.RuneCountInString(summary) > maxTurnSummaryRunes {
			return AnalyzeInput{}, fiber.StatusBadRequest, httpx.CodeInvalidInput
		}
		if summary != "" {
			cleanTurns = append(cleanTurns, ConversationTurn{
				Role:    role,
				Summary: summary,
			})
		}
	}

	return AnalyzeInput{
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
