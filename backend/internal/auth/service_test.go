package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"Nouriko/internal/config"
	"Nouriko/internal/database"

	"github.com/google/uuid"
)

const serviceTestJWTSecret = "0123456789abcdef0123456789abcdef"

// memRepo is an in-memory database.Repository for testing LocalService and SupabaseService.
type memRepo struct {
	mu       sync.Mutex
	users    map[string]*database.LocalUser
	tokens   map[string]*database.LocalAuthToken
	profiles map[string]*database.Profile
}

func newMemRepo() *memRepo {
	return &memRepo{
		users:    make(map[string]*database.LocalUser),
		tokens:   make(map[string]*database.LocalAuthToken),
		profiles: make(map[string]*database.Profile),
	}
}

func (m *memRepo) CreateLocalUser(_ context.Context, p database.CreateLocalUserParams) (*database.LocalUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := strings.ToLower(strings.TrimSpace(p.Email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == email {
			return nil, database.ErrConflict
		}
	}
	id := p.ID
	if id == "" {
		id = uuid.NewString()
	}
	now := time.Now()
	u := &database.LocalUser{
		ID:           id,
		Email:        email,
		PasswordHash: p.PasswordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.users[id] = u
	cp := *u
	return &cp, nil
}

func (m *memRepo) GetLocalUserByEmail(_ context.Context, email string) (*database.LocalUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	target := strings.ToLower(strings.TrimSpace(email))
	for _, u := range m.users {
		if strings.ToLower(u.Email) == target {
			cp := *u
			return &cp, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *memRepo) GetLocalUserByID(_ context.Context, userID string) (*database.LocalUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memRepo) RecordFailedLogin(_ context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*database.LocalUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	u.FailedLoginAttempts++
	if u.FailedLoginAttempts >= maxAttempts {
		until := time.Now().Add(lockDuration)
		u.LockedUntil = &until
	}
	cp := *u
	return &cp, nil
}

func (m *memRepo) ResetLoginLockout(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.FailedLoginAttempts = 0
	u.LockedUntil = nil
	return nil
}

func (m *memRepo) MarkEmailVerified(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.EmailVerified = true
	return nil
}

func (m *memRepo) UpdatePasswordHash(_ context.Context, userID, passwordHash string, resetLockout bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.PasswordHash = passwordHash
	if resetLockout {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (m *memRepo) DeleteLocalUser(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[userID]; !ok {
		return database.ErrNotFound
	}
	delete(m.users, userID)
	for k, tok := range m.tokens {
		if tok.UserID == userID {
			delete(m.tokens, k)
		}
	}
	return nil
}

func (m *memRepo) CreateAuthToken(_ context.Context, p database.CreateAuthTokenParams) (*database.LocalAuthToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok := &database.LocalAuthToken{
		ID:        uuid.NewString(),
		UserID:    p.UserID,
		TokenHash: p.TokenHash,
		Purpose:   p.Purpose,
		ExpiresAt: p.ExpiresAt,
		CreatedAt: time.Now(),
	}
	m.tokens[p.TokenHash] = tok
	cp := *tok
	return &cp, nil
}

func (m *memRepo) InvalidateUserTokens(_ context.Context, userID, purpose string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, tok := range m.tokens {
		if tok.UserID == userID && tok.Purpose == purpose && tok.UsedAt == nil {
			tok.UsedAt = &now
		}
	}
	return nil
}

func (m *memRepo) ConsumeAuthToken(_ context.Context, tokenHash, purpose string) (*database.LocalAuthToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.tokens[tokenHash]
	if !ok || tok.Purpose != purpose || tok.UsedAt != nil || !tok.ExpiresAt.After(time.Now()) {
		return nil, database.ErrTokenInvalidOrExpired
	}
	now := time.Now()
	tok.UsedAt = &now
	cp := *tok
	return &cp, nil
}

func (m *memRepo) DeleteExpiredTokens(_ context.Context) (int64, error) { return 0, nil }

func (m *memRepo) UpsertProfile(_ context.Context, p database.UpsertProfileParams) (*database.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.profiles[p.ID]; ok {
		cp := *existing
		return &cp, nil
	}
	now := time.Now()
	prof := &database.Profile{
		ID:         p.ID,
		ProfileURL: p.ProfileURL,
		Language:   p.Language,
		Theme:      p.Theme,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	m.profiles[p.ID] = prof
	cp := *prof
	return &cp, nil
}

func (m *memRepo) GetProfileByID(_ context.Context, userID string) (*database.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *prof
	return &cp, nil
}

func (m *memRepo) UpdateOnboarding(_ context.Context, p database.UpdateOnboardingParams) (*database.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[p.UserID]
	if !ok {
		return nil, database.ErrNotFound
	}
	prof.HeightCM = &p.HeightCM
	prof.WeightKG = &p.WeightKG
	prof.Age = &p.Age
	prof.Gender = &p.Gender
	prof.ActivityLevel = &p.ActivityLevel
	prof.DailyCalorieTarget = &p.DailyCalorieTarget
	prof.TargetProteinG = &p.TargetProteinG
	prof.TargetCarbsG = &p.TargetCarbsG
	prof.TargetFatG = &p.TargetFatG
	prof.TargetFiberG = &p.TargetFiberG
	cp := *prof
	return &cp, nil
}

func (m *memRepo) UpdateProfile(_ context.Context, p database.UpdateProfileParams) (*database.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[p.UserID]
	if !ok {
		return nil, database.ErrNotFound
	}
	if p.HeightCM != nil {
		prof.HeightCM = p.HeightCM
	}
	if p.WeightKG != nil {
		prof.WeightKG = p.WeightKG
	}
	if p.Age != nil {
		prof.Age = p.Age
	}
	if p.Gender != nil {
		prof.Gender = p.Gender
	}
	if p.ActivityLevel != nil {
		prof.ActivityLevel = p.ActivityLevel
	}
	if p.DailyCalorieTarget != nil {
		prof.DailyCalorieTarget = p.DailyCalorieTarget
	}
	if p.TargetProteinG != nil {
		prof.TargetProteinG = p.TargetProteinG
	}
	if p.TargetCarbsG != nil {
		prof.TargetCarbsG = p.TargetCarbsG
	}
	if p.TargetFatG != nil {
		prof.TargetFatG = p.TargetFatG
	}
	if p.TargetFiberG != nil {
		prof.TargetFiberG = p.TargetFiberG
	}
	if p.Language != nil {
		prof.Language = *p.Language
	}
	if p.Theme != nil {
		prof.Theme = *p.Theme
	}
	cp := *prof
	return &cp, nil
}

func (m *memRepo) DeleteProfile(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[userID]; !ok {
		return database.ErrNotFound
	}
	delete(m.profiles, userID)
	return nil
}

func (m *memRepo) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	tok, err := m.ConsumeAuthToken(ctx, tokenHash, database.TokenPurposeVerifyEmail)
	if err != nil {
		return "", err
	}
	if err := m.MarkEmailVerified(ctx, tok.UserID); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (m *memRepo) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tok, err := m.ConsumeAuthToken(ctx, tokenHash, database.TokenPurposeResetPassword)
	if err != nil {
		return "", err
	}
	if err := m.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (m *memRepo) DeleteLocalAccount(ctx context.Context, userID string) error {
	_ = m.DeleteProfile(ctx, userID)
	return m.DeleteLocalUser(ctx, userID)
}

func (m *memRepo) ListFoodPricesByBudget(_ context.Context, _ int, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func (m *memRepo) SearchFoodPrices(_ context.Context, _ string, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func (m *memRepo) ListAllFoodPrices(_ context.Context, _ string, _ int) ([]database.FoodPrice, error) {
	return []database.FoodPrice{}, nil
}

func TestLocalServiceFullLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepo()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret: serviceTestJWTSecret,
			Audience:  "authenticated",
		},
	}
	svc := NewLocalService(cfg, repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	// 1. Weak password rejected.
	if _, err := svc.Register(ctx, RegisterInput{Email: "raka@example.com", Password: "password123"}); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}

	// 2. Register succeeds and creates DiceBear avatar profile + dev token.
	reg, err := svc.Register(ctx, RegisterInput{
		Email:    "Raka@Example.com",
		Password: "kopisusupagi",
		Language: "en",
		Theme:    "dark",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if reg.DevVerificationToken == "" {
		t.Fatal("expected dev_verification_token in local mode")
	}
	if reg.Email != "raka@example.com" {
		t.Fatalf("expected normalized email, got %q", reg.Email)
	}

	prof, err := repo.GetProfileByID(ctx, reg.UserID)
	if err != nil {
		t.Fatalf("profile not created: %v", err)
	}
	if prof.ProfileURL != DiceBearAvatarURL(reg.UserID) {
		t.Fatalf("unexpected DiceBear URL: %q", prof.ProfileURL)
	}
	if prof.Language != "en" || prof.Theme != "dark" {
		t.Fatalf("unexpected preferences: lang=%q theme=%q", prof.Language, prof.Theme)
	}

	// 3. Duplicate email rejected.
	if _, err := svc.Register(ctx, RegisterInput{Email: "raka@example.com", Password: "kopisusupagi"}); !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	// 4. Login blocked before email verification.
	if _, err := svc.Login(ctx, "raka@example.com", "kopisusupagi"); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}

	// 5. Verify email once; second attempt rejected (single-use).
	if err := svc.VerifyEmail(ctx, reg.DevVerificationToken); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	if err := svc.VerifyEmail(ctx, reg.DevVerificationToken); !errors.Is(err, ErrInvalidOrExpiredToken) {
		t.Fatalf("expected ErrInvalidOrExpiredToken on reuse, got %v", err)
	}

	// 6. Login succeeds and returns valid JWT.
	loginRes, err := svc.Login(ctx, "raka@example.com", "kopisusupagi")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loginRes.AccessToken == "" || loginRes.TokenType != "Bearer" || loginRes.ExpiresIn != 86400 {
		t.Fatalf("unexpected login result: %+v", loginRes)
	}

	// 7. Unknown email returns ErrInvalidCredentials.
	if _, err := svc.Login(ctx, "nobody@example.com", "kopisusupagi"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown email, got %v", err)
	}

	// 8. Lockout after 5 failed attempts.
	for i := 0; i < 5; i++ {
		if _, err := svc.Login(ctx, "raka@example.com", "wrongpassword"); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d: expected ErrInvalidCredentials, got %v", i+1, err)
		}
	}
	if _, err := svc.Login(ctx, "raka@example.com", "kopisusupagi"); !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("expected ErrTooManyRequests when locked, got %v", err)
	}

	// 9. ForgotPassword is non-enumerating and issues a single-use reset token that also clears lockout.
	if err := svc.ForgotPassword(ctx, "unknown@example.com"); err != nil {
		t.Fatalf("ForgotPassword unknown email must return nil, got %v", err)
	}
	if err := svc.ForgotPassword(ctx, "raka@example.com"); err != nil {
		t.Fatalf("ForgotPassword known email: %v", err)
	}

	// Inject a known reset token directly to test ResetPassword end-to-end.
	rawReset, hashReset, _ := GenerateRandomToken()
	_, _ = repo.CreateAuthToken(ctx, database.CreateAuthTokenParams{
		UserID:    reg.UserID,
		TokenHash: hashReset,
		Purpose:   database.TokenPurposeResetPassword,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	})
	if err := svc.ResetPassword(ctx, rawReset, "tehtarikhangat"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	if err := svc.ResetPassword(ctx, rawReset, "tehtarikhangat2"); !errors.Is(err, ErrInvalidOrExpiredToken) {
		t.Fatalf("expected ErrInvalidOrExpiredToken on reset token reuse, got %v", err)
	}

	// Lockout cleared and new password works.
	if _, err := svc.Login(ctx, "raka@example.com", "tehtarikhangat"); err != nil {
		t.Fatalf("Login after reset: %v", err)
	}

	// 10. ChangePassword validates current password and updates hash.
	if err := svc.ChangePassword(ctx, reg.UserID, reg.Email, "", "wrongcurr", "nasigorengspesial"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if err := svc.ChangePassword(ctx, reg.UserID, reg.Email, "", "tehtarikhangat", "nasigorengspesial"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	// 11. DeleteAccount requires correct password and removes user + profile.
	if err := svc.DeleteAccount(ctx, reg.UserID, reg.Email, "wrongpass"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on delete, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, reg.UserID, reg.Email, "nasigorengspesial"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if _, err := repo.GetProfileByID(ctx, reg.UserID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("expected profile deleted, got %v", err)
	}
	if _, err := repo.GetLocalUserByID(ctx, reg.UserID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("expected local user deleted, got %v", err)
	}
}

func TestLocalServiceConcurrentLoginLockout(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepo()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret: serviceTestJWTSecret,
			Audience:  "authenticated",
		},
	}
	svc := NewLocalService(cfg, repo, slog.New(slog.NewJSONHandler(io.Discard, nil)))

	reg, err := svc.Register(ctx, RegisterInput{
		Email:    "burst@example.com",
		Password: "kopisusupagi",
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := svc.VerifyEmail(ctx, reg.DevVerificationToken); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}

	const totalAttempts = 10
	errs := make([]error, totalAttempts)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < totalAttempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			<-start
			_, errs[idx] = svc.Login(ctx, "burst@example.com", "wrongpassword")
		}(i)
	}
	close(start)
	wg.Wait()

	var invalidCreds, tooMany int
	for _, err := range errs {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			invalidCreds++
		case errors.Is(err, ErrTooManyRequests):
			tooMany++
		default:
			t.Fatalf("unexpected error from concurrent login: %v", err)
		}
	}
	if invalidCreds != int(maxFailedLoginAttempts) {
		t.Fatalf("expected exactly %d ErrInvalidCredentials before lockout, got %d (tooMany=%d)", maxFailedLoginAttempts, invalidCreds, tooMany)
	}
	if tooMany != totalAttempts-int(maxFailedLoginAttempts) {
		t.Fatalf("expected %d ErrTooManyRequests after lockout, got %d", totalAttempts-int(maxFailedLoginAttempts), tooMany)
	}

	// Ensure keyedLocker cleaned up map entries after all goroutines finished.
	svc.loginLocks.mu.Lock()
	remaining := len(svc.loginLocks.entries)
	svc.loginLocks.mu.Unlock()
	if remaining != 0 {
		t.Fatalf("expected keyedLocker entries to be cleaned up, got %d remaining", remaining)
	}
}

func TestSupabaseServiceProductionModeMock(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	var deletedAdminUserID string
	var logoutTokens []string
	var resetCallSequence []string
	var userPutCalled bool

	mockSupabase := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/signup":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["email"] == "exists@example.com" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"id":"` + userID + `","email":"exists@example.com","identities":[]}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + userID + `","email":"` + body["email"] + `","identities":[{"id":"1"}]}`))

		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/verify":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["type"] == "recovery" {
				if body["token_hash"] == "good-recovery-hash" {
					resetCallSequence = append(resetCallSequence, "verify:recovery")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"access_token":"recovery-jwt","token_type":"bearer","expires_in":300,"user":{"id":"` + userID + `","email":"raka@example.com"}}`))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error_code":"otp_expired"}`))
				return
			}
			if body["token_hash"] == "valid-hash" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"tok"}`))
				return
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code":"otp_expired"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/token":
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			switch body["password"] {
			case "unverified-pass":
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"email_not_confirmed","msg":"Email not confirmed"}`))
			case "kopisusupagi":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"sb-jwt","token_type":"bearer","expires_in":3600,"user":{"id":"` + userID + `","email":"` + body["email"] + `"}}`))
			default:
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"invalid_credentials","msg":"Invalid login credentials"}`))
			}

		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/logout":
			authHdr := r.Header.Get("Authorization")
			logoutTokens = append(logoutTokens, authHdr)
			if authHdr == "Bearer recovery-jwt" {
				resetCallSequence = append(resetCallSequence, "logout:recovery")
			}
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/recover":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

		case r.Method == http.MethodPut && r.URL.Path == "/auth/v1/user":
			userPutCalled = true
			if r.Header.Get("Authorization") == "Bearer recovery-jwt" {
				resetCallSequence = append(resetCallSequence, "put:user")
			}
			if r.Header.Get("Authorization") == "Bearer bad-recovery" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"msg":"invalid token"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + userID + `"}`))

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/auth/v1/admin/users/"):
			deletedAdminUserID = strings.TrimPrefix(r.URL.Path, "/auth/v1/admin/users/")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSupabase.Close()

	cfg := &config.Config{
		Mode:       config.ModeProduction,
		AppBaseURL: "https://nouriko.app",
		Supabase: config.Supabase{
			URL:       mockSupabase.URL,
			AnonKey:   "sb_anon_test",
			SecretKey: "sb_secret_test",
		},
	}
	repo := newMemRepo()
	admin := NewAdminClient(cfg)
	svc := NewSupabaseService(cfg, repo, admin, slog.New(slog.NewJSONHandler(io.Discard, nil)), mockSupabase.Client())

	// Register new user -> no dev token in production.
	reg, err := svc.Register(ctx, RegisterInput{Email: "raka@example.com", Password: "kopisusupagi"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if reg.DevVerificationToken != "" {
		t.Fatal("production register must NOT return dev_verification_token")
	}

	// Duplicate user via empty identities.
	if _, err := svc.Register(ctx, RegisterInput{Email: "exists@example.com", Password: "kopisusupagi"}); !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}

	// Verify email.
	if err := svc.VerifyEmail(ctx, "valid-hash"); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	if err := svc.VerifyEmail(ctx, "expired-hash"); !errors.Is(err, ErrInvalidOrExpiredToken) {
		t.Fatalf("expected ErrInvalidOrExpiredToken, got %v", err)
	}

	// Login unconfirmed / wrong / valid.
	if _, err := svc.Login(ctx, "raka@example.com", "unverified-pass"); !errors.Is(err, ErrEmailNotVerified) {
		t.Fatalf("expected ErrEmailNotVerified, got %v", err)
	}
	if _, err := svc.Login(ctx, "raka@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	loginRes, err := svc.Login(ctx, "raka@example.com", "kopisusupagi")
	if err != nil || loginRes.AccessToken != "sb-jwt" {
		t.Fatalf("Login failed: res=%+v err=%v", loginRes, err)
	}

	// Logout.
	if err := svc.Logout(ctx, "sb-jwt"); err != nil || len(logoutTokens) == 0 || logoutTokens[0] != "Bearer sb-jwt" {
		t.Fatalf("Logout failed: err=%v tokens=%v", err, logoutTokens)
	}

	// Forgot & Reset password.
	if err := svc.ForgotPassword(ctx, "raka@example.com"); err != nil {
		t.Fatalf("ForgotPassword: %v", err)
	}

	// Security regression [MED-01]: regular session JWT must be rejected and MUST NOT reach PUT /auth/v1/user.
	userPutCalled = false
	if err := svc.ResetPassword(ctx, "sb-jwt", "tehtarikhangat"); !errors.Is(err, ErrInvalidOrExpiredToken) {
		t.Fatalf("expected ErrInvalidOrExpiredToken for regular session JWT, got %v", err)
	}
	if userPutCalled {
		t.Fatal("PUT /auth/v1/user must not be called when /auth/v1/verify rejects non-recovery token")
	}

	// Valid recovery token hash executes: POST /auth/v1/verify (type: recovery) -> PUT /auth/v1/user -> POST /auth/v1/logout.
	resetCallSequence = nil
	if err := svc.ResetPassword(ctx, "good-recovery-hash", "tehtarikhangat"); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}
	expectedSeq := "verify:recovery,put:user,logout:recovery"
	if got := strings.Join(resetCallSequence, ","); got != expectedSeq {
		t.Fatalf("unexpected ResetPassword call sequence: got %q, want %q", got, expectedSeq)
	}

	// Change password rejects mismatched userID and succeeds for matching userID.
	if err := svc.ChangePassword(ctx, uuid.NewString(), "raka@example.com", "sb-jwt", "kopisusupagi", "tehtarikhangat"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on userID mismatch in ChangePassword, got %v", err)
	}
	if err := svc.ChangePassword(ctx, userID, "raka@example.com", "sb-jwt", "kopisusupagi", "tehtarikhangat"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}

	// DeleteAccount verifies password + userID, deletes profile, calls AdminClient.DeleteUser.
	if err := svc.DeleteAccount(ctx, userID, "raka@example.com", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on delete, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, uuid.NewString(), "raka@example.com", "kopisusupagi"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on userID mismatch in DeleteAccount, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, userID, "raka@example.com", "kopisusupagi"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if deletedAdminUserID != userID {
		t.Fatalf("expected AdminClient.DeleteUser(%s), got %q", userID, deletedAdminUserID)
	}
	if _, err := repo.GetProfileByID(ctx, userID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("expected profile deleted, got %v", err)
	}
}
