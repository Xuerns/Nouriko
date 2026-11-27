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
	"Nouriko/internal/profile"

	"github.com/google/uuid"
)

const serviceTestJWTSecret = "0123456789abcdef0123456789abcdef"

// memAuthRepo is an in-memory LocalAuthRepository + profile.Repository for testing.
type memAuthRepo struct {
	mu             sync.Mutex
	users          map[string]*LocalUser
	identities     map[string]*LocalUserIdentity // key: provider + ":" + provider_user_id
	tokens         map[string]*LocalAuthToken
	consumedNonces map[string]time.Time
	profiles       map[string]*profile.Profile
}

func newMemAuthRepo() *memAuthRepo {
	return &memAuthRepo{
		users:          make(map[string]*LocalUser),
		identities:     make(map[string]*LocalUserIdentity),
		tokens:         make(map[string]*LocalAuthToken),
		consumedNonces: make(map[string]time.Time),
		profiles:       make(map[string]*profile.Profile),
	}
}

func (m *memAuthRepo) ConsumeOAuthNonce(_ context.Context, nonceHash string, expiresAt time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.consumedNonces == nil {
		m.consumedNonces = make(map[string]time.Time)
	}
	now := time.Now()
	for k, exp := range m.consumedNonces {
		if !exp.After(now) {
			delete(m.consumedNonces, k)
		}
	}
	if exp, ok := m.consumedNonces[nonceHash]; ok && exp.After(now) {
		return false, nil
	}
	m.consumedNonces[nonceHash] = expiresAt
	return true, nil
}

func cloneLocalUser(u *LocalUser) *LocalUser {
	cp := *u
	if u.PasswordHash != nil {
		h := *u.PasswordHash
		cp.PasswordHash = &h
	}
	if u.LockedUntil != nil {
		t := *u.LockedUntil
		cp.LockedUntil = &t
	}
	return &cp
}

func (m *memAuthRepo) CreateLocalUser(_ context.Context, p CreateLocalUserParams) (*LocalUser, error) {
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
	var pwHash *string
	if p.PasswordHash != "" {
		h := p.PasswordHash
		pwHash = &h
	}
	u := &LocalUser{
		ID:           id,
		Email:        email,
		PasswordHash: pwHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	m.users[id] = u
	return cloneLocalUser(u), nil
}

func (m *memAuthRepo) GetLocalUserByEmail(_ context.Context, email string) (*LocalUser, error) {
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

func (m *memAuthRepo) GetLocalUserByID(_ context.Context, userID string) (*LocalUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memAuthRepo) RecordFailedLogin(_ context.Context, userID string, maxAttempts int16, lockDuration time.Duration) (*LocalUser, error) {
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

func (m *memAuthRepo) ResetLoginLockout(_ context.Context, userID string) error {
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

func (m *memAuthRepo) MarkEmailVerified(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	u.EmailVerified = true
	return nil
}

func (m *memAuthRepo) UpdatePasswordHash(_ context.Context, userID, passwordHash string, resetLockout bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return database.ErrNotFound
	}
	h := passwordHash
	u.PasswordHash = &h
	if resetLockout {
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
	}
	return nil
}

func (m *memAuthRepo) DeleteLocalUser(_ context.Context, userID string) error {
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
	for k, id := range m.identities {
		if id.UserID == userID {
			delete(m.identities, k)
		}
	}
	return nil
}

func (m *memAuthRepo) CreateAuthToken(_ context.Context, p CreateAuthTokenParams) (*LocalAuthToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok := &LocalAuthToken{
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

func (m *memAuthRepo) InvalidateUserTokens(_ context.Context, userID, purpose string) error {
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

func (m *memAuthRepo) ConsumeAuthToken(_ context.Context, tokenHash, purpose string) (*LocalAuthToken, error) {
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

func (m *memAuthRepo) DeleteExpiredTokens(_ context.Context) (int64, error) { return 0, nil }

func (m *memAuthRepo) UpsertProfile(_ context.Context, p profile.UpsertProfileParams) (*profile.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.profiles[p.ID]; ok {
		if existing.ProfileURL == "" {
			existing.ProfileURL = p.ProfileURL
		}
		if existing.DisplayName == "" && strings.TrimSpace(p.DisplayName) != "" {
			existing.DisplayName = strings.TrimSpace(p.DisplayName)
		}
		cp := *existing
		return &cp, nil
	}
	now := time.Now()
	prof := &profile.Profile{
		ID:          p.ID,
		ProfileURL:  p.ProfileURL,
		DisplayName: strings.TrimSpace(p.DisplayName),
		Language:    p.Language,
		Theme:       p.Theme,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	m.profiles[p.ID] = prof
	cp := *prof
	return &cp, nil
}

func (m *memAuthRepo) GetProfileByID(_ context.Context, userID string) (*profile.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[userID]
	if !ok {
		return nil, database.ErrNotFound
	}
	cp := *prof
	return &cp, nil
}

func (m *memAuthRepo) UpdateOnboarding(_ context.Context, p profile.UpdateOnboardingParams) (*profile.Profile, error) {
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

func (m *memAuthRepo) UpdateProfile(_ context.Context, p profile.UpdateProfileParams) (*profile.Profile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prof, ok := m.profiles[p.UserID]
	if !ok {
		return nil, database.ErrNotFound
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

func (m *memAuthRepo) DeleteProfile(_ context.Context, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.profiles[userID]; !ok {
		return database.ErrNotFound
	}
	delete(m.profiles, userID)
	return nil
}

func (m *memAuthRepo) VerifyEmailWithToken(ctx context.Context, tokenHash string) (string, error) {
	tok, err := m.ConsumeAuthToken(ctx, tokenHash, TokenPurposeVerifyEmail)
	if err != nil {
		return "", err
	}
	if err := m.MarkEmailVerified(ctx, tok.UserID); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (m *memAuthRepo) ResetPasswordWithToken(ctx context.Context, tokenHash, newPasswordHash string) (string, error) {
	tok, err := m.ConsumeAuthToken(ctx, tokenHash, TokenPurposeResetPassword)
	if err != nil {
		return "", err
	}
	if err := m.UpdatePasswordHash(ctx, tok.UserID, newPasswordHash, true); err != nil {
		return "", err
	}
	return tok.UserID, nil
}

func (m *memAuthRepo) DeleteLocalAccount(ctx context.Context, userID string) error {
	_ = m.DeleteProfile(ctx, userID)
	return m.DeleteLocalUser(ctx, userID)
}

func (m *memAuthRepo) providersLocked(userID string) []string {
	u, ok := m.users[userID]
	if !ok {
		return nil
	}
	out := make([]string, 0, 2)
	if u.HasPassword() {
		out = append(out, "email")
	}
	for _, id := range m.identities {
		if id.UserID == userID {
			out = append(out, id.Provider)
		}
	}
	return out
}

func (m *memAuthRepo) GetUserProviders(_ context.Context, userID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[userID]; !ok {
		return nil, database.ErrNotFound
	}
	return m.providersLocked(userID), nil
}

func (m *memAuthRepo) GetUserIdentity(_ context.Context, userID, provider string) (*LocalUserIdentity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range m.identities {
		if id.UserID == userID && id.Provider == provider {
			cp := *id
			return &cp, nil
		}
	}
	return nil, database.ErrNotFound
}

func (m *memAuthRepo) FindOrCreateGoogleUser(_ context.Context, p UpsertGoogleUserParams) (*LocalUser, []string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	email := strings.ToLower(strings.TrimSpace(p.Email))
	sub := strings.TrimSpace(p.ProviderUserID)
	key := OAuthProviderGoogle + ":" + sub

	if existingID, ok := m.identities[key]; ok {
		u, uOk := m.users[existingID.UserID]
		if !uOk {
			return nil, nil, database.ErrNotFound
		}
		u.FailedLoginAttempts = 0
		u.LockedUntil = nil
		return cloneLocalUser(u), m.providersLocked(u.ID), nil
	}

	var matched *LocalUser
	for _, u := range m.users {
		if strings.ToLower(u.Email) == email {
			matched = u
			break
		}
	}

	now := time.Now()
	if matched != nil {
		// Prevent binding a different Google sub to an already-linked user.
		for _, id := range m.identities {
			if id.UserID == matched.ID && id.Provider == OAuthProviderGoogle && id.ProviderUserID != sub {
				return nil, nil, database.ErrConflict
			}
		}
		if !matched.EmailVerified {
			matched.PasswordHash = nil
			matched.EmailVerified = true
			matched.FailedLoginAttempts = 0
			matched.LockedUntil = nil
			for _, tok := range m.tokens {
				if tok.UserID == matched.ID && tok.UsedAt == nil {
					t := now
					tok.UsedAt = &t
				}
			}
		} else {
			matched.FailedLoginAttempts = 0
			matched.LockedUntil = nil
		}
		m.identities[key] = &LocalUserIdentity{
			ID:             uuid.NewString(),
			UserID:         matched.ID,
			Provider:       OAuthProviderGoogle,
			ProviderUserID: sub,
			Email:          email,
			CreatedAt:      now,
		}
		return cloneLocalUser(matched), m.providersLocked(matched.ID), nil
	}

	u := &LocalUser{
		ID:            uuid.NewString(),
		Email:         email,
		PasswordHash:  nil,
		EmailVerified: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	m.users[u.ID] = u
	m.identities[key] = &LocalUserIdentity{
		ID:             uuid.NewString(),
		UserID:         u.ID,
		Provider:       OAuthProviderGoogle,
		ProviderUserID: sub,
		Email:          email,
		CreatedAt:      now,
	}
	return cloneLocalUser(u), m.providersLocked(u.ID), nil
}

type mockGoogleVerifier struct {
	fn func(ctx context.Context, idToken, rawNonce string) (*GoogleIdentity, error)
}

func (m *mockGoogleVerifier) VerifyIDToken(ctx context.Context, idToken, rawNonce string) (*GoogleIdentity, error) {
	return m.fn(ctx, idToken, rawNonce)
}

func TestLocalServiceFullLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := newMemAuthRepo()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret: serviceTestJWTSecret,
			Audience:  "authenticated",
		},
	}
	svc := NewLocalService(cfg, repo, repo, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))

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
	_, _ = repo.CreateAuthToken(ctx, CreateAuthTokenParams{
		UserID:    reg.UserID,
		TokenHash: hashReset,
		Purpose:   TokenPurposeResetPassword,
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
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{UserID: reg.UserID, Email: reg.Email, Password: "wrongpass", HasPassword: true}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on delete, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{UserID: reg.UserID, Email: reg.Email, Password: "nasigorengspesial", HasPassword: true}); err != nil {
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
	repo := newMemAuthRepo()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret: serviceTestJWTSecret,
			Audience:  "authenticated",
		},
	}
	svc := NewLocalService(cfg, repo, repo, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))

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
	repo := newMemAuthRepo()
	admin := NewAdminClient(cfg)
	svc := NewSupabaseService(cfg, repo, admin, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), mockSupabase.Client())

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
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{UserID: userID, Email: "raka@example.com", Password: "wrong", HasPassword: true}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on delete, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{UserID: uuid.NewString(), Email: "raka@example.com", Password: "kopisusupagi", HasPassword: true}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on userID mismatch in DeleteAccount, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{UserID: userID, Email: "raka@example.com", Password: "kopisusupagi", HasPassword: true}); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if deletedAdminUserID != userID {
		t.Fatalf("expected AdminClient.DeleteUser(%s), got %q", userID, deletedAdminUserID)
	}
	if _, err := repo.GetProfileByID(ctx, userID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("expected profile deleted, got %v", err)
	}
}

func TestLocalServiceGoogleSignInLifecycleAndSecurity(t *testing.T) {
	ctx := context.Background()
	repo := newMemAuthRepo()
	cfg := &config.Config{
		Mode:       config.ModeLocal,
		AppBaseURL: "http://localhost:3000",
		Auth: config.Auth{
			JWTSecret:      serviceTestJWTSecret,
			Audience:       "authenticated",
			GoogleClientID: "test-google-client-id",
		},
	}
	fixedNow := time.Now().UTC().Truncate(time.Second)
	var currentIdentity *GoogleIdentity
	gv := &mockGoogleVerifier{
		fn: func(_ context.Context, idToken, _ string) (*GoogleIdentity, error) {
			if idToken == "bad-token" {
				return nil, ErrInvalidCredentials
			}
			cp := *currentIdentity
			return &cp, nil
		},
	}

	svc := NewLocalService(cfg, repo, repo, gv, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	svc.now = func() time.Time { return fixedNow }
	jwtVerifier, err := NewVerifier(ctx, cfg)
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	const (
		nonceLogin1 = "0123456789abcdef0123456789abcde1"
		nonceLogin2 = "0123456789abcdef0123456789abcde2"
		nonceReauth = "0123456789abcdef0123456789abcde3"
		nonceVictim = "0123456789abcdef0123456789abcde4"
		nonceLinked = "0123456789abcdef0123456789abcde5"
	)

	// 1. Unconfigured service returns ErrOAuthUnavailable.
	disabledSvc := NewLocalService(cfg, repo, repo, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if _, err := disabledSvc.LoginWithGoogle(ctx, GoogleLoginInput{IDToken: "tok", Nonce: nonceLogin1}); !errors.Is(err, ErrOAuthUnavailable) {
		t.Fatalf("expected ErrOAuthUnavailable when verifier is nil, got %v", err)
	}

	// 2. Brand-new Google user signup provisions profile + sanitized DisplayName + google-only JWT.
	currentIdentity = &GoogleIdentity{
		Subject:       "sub-new-user",
		Email:         "dina@gmail.com",
		Name:          "  Dina\x00 Kusuma  ",
		EmailVerified: true,
		IssuedAt:      fixedNow.Add(-30 * time.Second),
		ExpiresAt:     fixedNow.Add(time.Hour),
	}
	res, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{
		IDToken:  "valid-id-token",
		Nonce:    nonceLogin1,
		Language: "en",
		Theme:    "dark",
	})
	if err != nil {
		t.Fatalf("LoginWithGoogle new user: %v", err)
	}
	if res.HasPassword || len(res.AuthProviders) != 1 || res.AuthProviders[0] != "google" {
		t.Fatalf("unexpected new google user auth metadata: hasPassword=%v providers=%v", res.HasPassword, res.AuthProviders)
	}
	if res.Profile.DisplayName != "Dina Kusuma" || res.Profile.ProfileURL != DiceBearAvatarURL(res.UserID) {
		t.Fatalf("unexpected google profile: %+v", res.Profile)
	}
	claims, err := jwtVerifier.Verify(res.AccessToken)
	if err != nil || claims.HasPassword() {
		t.Fatalf("expected valid google-only JWT with HasPassword()==false, claims=%+v err=%v", claims, err)
	}

	// 2b. Replaying the exact same (id_token, nonce) on LoginWithGoogle MUST be rejected.
	if _, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{
		IDToken: "valid-id-token",
		Nonce:   nonceLogin1,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on replayed LoginWithGoogle nonce, got %v", err)
	}

	// 3. Password login & ChangePassword on Google-only account are rejected without leaking existence.
	if _, err := svc.Login(ctx, "dina@gmail.com", "anypassword123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for password login on google-only user, got %v", err)
	}
	if err := svc.ChangePassword(ctx, res.UserID, "dina@gmail.com", "", "anypassword", "newpassword123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for ChangePassword on google-only user, got %v", err)
	}

	// 4. Repeat login with a fresh nonce returns same user and preserves custom display_name.
	repo.profiles[res.UserID].DisplayName = "Custom Dina"
	res2, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{IDToken: "valid-id-token-2", Nonce: nonceLogin2})
	if err != nil || res2.UserID != res.UserID || res2.Profile.DisplayName != "Custom Dina" {
		t.Fatalf("repeat google login unexpected result: res=%+v err=%v", res2, err)
	}

	// 5. DeleteAccount rejects replayed login nonce, stale iat (>5m), and mismatched sub, then succeeds with fresh token.
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        res.UserID,
		Email:         "dina@gmail.com",
		GoogleIDToken: "valid-id-token-2",
		Nonce:         nonceLogin2,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials when replaying login nonce on DeleteAccount, got %v", err)
	}

	currentIdentity.IssuedAt = fixedNow.Add(-6 * time.Minute)
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        res.UserID,
		Email:         "dina@gmail.com",
		GoogleIDToken: "stale-token",
		Nonce:         nonceReauth,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for stale Google re-auth token, got %v", err)
	}

	currentIdentity.IssuedAt = fixedNow.Add(-10 * time.Second)
	currentIdentity.Subject = "sub-other-person"
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        res.UserID,
		Email:         "dina@gmail.com",
		GoogleIDToken: "other-sub-token",
		Nonce:         nonceReauth,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for mismatched Google sub, got %v", err)
	}

	currentIdentity.Subject = "sub-new-user"
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        res.UserID,
		Email:         "dina@gmail.com",
		GoogleIDToken: "fresh-valid-token",
		Nonce:         nonceReauth,
		HasPassword:   false,
	}); err != nil {
		t.Fatalf("expected fresh Google re-auth DeleteAccount to succeed, got %v", err)
	}
	if _, err := repo.GetLocalUserByID(ctx, res.UserID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("expected google user deleted, got %v", err)
	}

	// 6. Pre-account takeover mitigation: unverified local registration + Google login clears password & invalidates old verify token.
	regUnverified, err := svc.Register(ctx, RegisterInput{
		Email:    "victim@gmail.com",
		Password: "attackerpassword9",
	})
	if err != nil {
		t.Fatalf("Register unverified: %v", err)
	}
	currentIdentity = &GoogleIdentity{
		Subject:       "sub-victim",
		Email:         "victim@gmail.com",
		Name:          "Real Owner",
		EmailVerified: true,
		IssuedAt:      fixedNow,
		ExpiresAt:     fixedNow.Add(time.Hour),
	}
	victimRes, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{IDToken: "victim-tok", Nonce: nonceVictim})
	if err != nil {
		t.Fatalf("LoginWithGoogle victim: %v", err)
	}
	if victimRes.UserID != regUnverified.UserID {
		t.Fatalf("expected same user id after pre-takeover mitigation, got %s vs %s", victimRes.UserID, regUnverified.UserID)
	}
	if victimRes.HasPassword {
		t.Fatal("expected pre-registered unverified password to be cleared (HasPassword == false)")
	}
	if _, err := svc.Login(ctx, "victim@gmail.com", "attackerpassword9"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("attacker password must NOT work after Google takeover mitigation, got %v", err)
	}
	if err := svc.VerifyEmail(ctx, regUnverified.DevVerificationToken); !errors.Is(err, ErrInvalidOrExpiredToken) {
		t.Fatalf("old verify token must be invalidated, got %v", err)
	}

	// 7. Safe linking to already-verified local password account keeps password + links google.
	regVerified, err := svc.Register(ctx, RegisterInput{
		Email:    "verified@gmail.com",
		Password: "kopisusupagi",
	})
	if err != nil {
		t.Fatalf("Register verified: %v", err)
	}
	if err := svc.VerifyEmail(ctx, regVerified.DevVerificationToken); err != nil {
		t.Fatalf("VerifyEmail: %v", err)
	}
	currentIdentity = &GoogleIdentity{
		Subject:       "sub-verified",
		Email:         "verified@gmail.com",
		Name:          "Verified User",
		EmailVerified: true,
		IssuedAt:      fixedNow,
		ExpiresAt:     fixedNow.Add(time.Hour),
	}
	linkedRes, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{IDToken: "linked-tok", Nonce: nonceLinked})
	if err != nil {
		t.Fatalf("LoginWithGoogle verified link: %v", err)
	}
	if !linkedRes.HasPassword || len(linkedRes.AuthProviders) != 2 {
		t.Fatalf("expected linked account to retain password and have 2 providers, got hasPassword=%v providers=%v", linkedRes.HasPassword, linkedRes.AuthProviders)
	}
	if _, err := svc.Login(ctx, "verified@gmail.com", "kopisusupagi"); err != nil {
		t.Fatalf("existing password must still work after verified Google link, got %v", err)
	}
}

func TestSupabaseServiceGoogleSignInAndReauthDelete(t *testing.T) {
	ctx := context.Background()
	userID := uuid.NewString()
	const (
		nonceLogin       = "1123456789abcdef0123456789abcde1"
		nonceStale       = "1123456789abcdef0123456789abcde2"
		nonceWrongUser   = "1123456789abcdef0123456789abcde3"
		nonceHasPassword = "1123456789abcdef0123456789abcde4"
		nonceDelete      = "1123456789abcdef0123456789abcde5"
	)

	var (
		idTokenGrantCalled bool
		tempSessionRevoked bool
		deletedAdminID     string
	)

	mockSupabase := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/token" && r.URL.Query().Get("grant_type") == "id_token":
			idTokenGrantCalled = true
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["provider"] != "google" || len(body["nonce"]) < minGoogleNonceLen {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"invalid_credentials"}`))
				return
			}
			switch body["id_token"] {
			case "wrong-user-id-tok":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"temp-sb-jwt","token_type":"bearer","expires_in":3600,"user":{"id":"` + uuid.NewString() + `","email":"other@gmail.com","app_metadata":{"provider":"google","providers":["google"]}}}`))
				return
			case "now-has-password-tok":
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"access_token":"temp-sb-jwt","token_type":"bearer","expires_in":3600,"user":{"id":"` + userID + `","email":"siti@gmail.com","app_metadata":{"provider":"google","providers":["google","email"]}}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"google-sb-jwt","token_type":"bearer","expires_in":3600,"user":{"id":"` + userID + `","email":"siti@gmail.com","app_metadata":{"provider":"google","providers":["google"]},"user_metadata":{"full_name":"Siti Rahma"}}}`))

		case r.Method == http.MethodPost && r.URL.Path == "/auth/v1/logout":
			if r.Header.Get("Authorization") == "Bearer google-sb-jwt" || r.Header.Get("Authorization") == "Bearer temp-sb-jwt" {
				tempSessionRevoked = true
			}
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/auth/v1/admin/users/"):
			deletedAdminID = strings.TrimPrefix(r.URL.Path, "/auth/v1/admin/users/")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSupabase.Close()

	fixedNow := time.Now().UTC().Truncate(time.Second)
	verifiedIssuedAt := fixedNow.Add(-20 * time.Second)
	gv := &mockGoogleVerifier{
		fn: func(_ context.Context, _ string, _ string) (*GoogleIdentity, error) {
			return &GoogleIdentity{
				Subject:       "google-sub-siti",
				Email:         "siti@gmail.com",
				Name:          "Siti Rahma",
				EmailVerified: true,
				IssuedAt:      verifiedIssuedAt,
				ExpiresAt:     fixedNow.Add(time.Hour),
			}, nil
		},
	}

	cfg := &config.Config{
		Mode:       config.ModeProduction,
		AppBaseURL: "https://nouriko.app",
		Auth: config.Auth{
			GoogleClientID: "prod-google-client-id",
		},
		Supabase: config.Supabase{
			URL:       mockSupabase.URL,
			AnonKey:   "sb_anon_test",
			SecretKey: "sb_secret_test",
		},
	}
	repo := newMemAuthRepo()
	admin := NewAdminClient(cfg)
	svc := NewSupabaseService(cfg, repo, admin, gv, slog.New(slog.NewJSONHandler(io.Discard, nil)), mockSupabase.Client())
	svc.now = func() time.Time { return fixedNow }

	// 1. LoginWithGoogle succeeds, calls id_token grant, creates profile with display_name.
	res, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{
		IDToken:  "valid-google-id-token",
		Nonce:    nonceLogin,
		Language: "id",
		Theme:    "dark",
	})
	if err != nil {
		t.Fatalf("Supabase LoginWithGoogle failed: %v", err)
	}
	if !idTokenGrantCalled || res.AccessToken != "google-sb-jwt" || res.HasPassword || len(res.AuthProviders) != 1 || res.AuthProviders[0] != "google" {
		t.Fatalf("unexpected Supabase LoginWithGoogle result: %+v", res)
	}
	if res.Profile.DisplayName != "Siti Rahma" {
		t.Fatalf("expected backfilled display_name=Siti Rahma, got %q", res.Profile.DisplayName)
	}

	// 1b. Replaying login nonce on LoginWithGoogle or DeleteAccount MUST be rejected.
	if _, err := svc.LoginWithGoogle(ctx, GoogleLoginInput{
		IDToken: "valid-google-id-token",
		Nonce:   nonceLogin,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials on replayed Supabase LoginWithGoogle nonce, got %v", err)
	}
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "valid-google-id-token",
		Nonce:         nonceLogin,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials when replaying login nonce on Supabase DeleteAccount, got %v", err)
	}

	// 2. DeleteAccount rejects stale iat (>5m) before calling Supabase.
	verifiedIssuedAt = fixedNow.Add(-10 * time.Minute)
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "stale-google-id-token",
		Nonce:         nonceStale,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for stale Google re-auth, got %v", err)
	}

	// 3. DeleteAccount rejects mismatched user.id and revokes temp session.
	verifiedIssuedAt = fixedNow.Add(-15 * time.Second)
	tempSessionRevoked = false
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "wrong-user-id-tok",
		Nonce:         nonceWrongUser,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for user mismatch, got %v", err)
	}
	if !tempSessionRevoked {
		t.Fatal("expected temporary Supabase session to be revoked on mismatch")
	}

	// 3b. DeleteAccount rejects stale JWT has_password=false when live Supabase user has email password.
	tempSessionRevoked = false
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "now-has-password-tok",
		Nonce:         nonceHasPassword,
		HasPassword:   false,
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput when live Supabase user now has email password, got %v", err)
	}
	if !tempSessionRevoked {
		t.Fatal("expected temporary Supabase session to be revoked on password-drift rejection")
	}

	// 4. DeleteAccount succeeds with fresh token + matching user.id.
	tempSessionRevoked = false
	if err := svc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "fresh-google-id-token",
		Nonce:         nonceDelete,
		HasPassword:   false,
	}); err != nil {
		t.Fatalf("Supabase Google DeleteAccount failed: %v", err)
	}
	if !tempSessionRevoked || deletedAdminID != userID {
		t.Fatalf("expected temp session revoked (%v) and admin delete (%q == %q)", tempSessionRevoked, deletedAdminID, userID)
	}

	// 5. Missing GOOGLE_CLIENT_ID or nil googleVerifier fails closed with ErrOAuthUnavailable.
	disabledCfg := *cfg
	disabledCfg.Auth.GoogleClientID = ""
	disabledSvc := NewSupabaseService(&disabledCfg, repo, admin, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), mockSupabase.Client())
	if _, err := disabledSvc.LoginWithGoogle(ctx, GoogleLoginInput{IDToken: "tok", Nonce: nonceLogin}); !errors.Is(err, ErrOAuthUnavailable) {
		t.Fatalf("expected ErrOAuthUnavailable when GOOGLE_CLIENT_ID is empty, got %v", err)
	}
	nilVerifierSvc := NewSupabaseService(cfg, repo, admin, nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), mockSupabase.Client())
	if err := nilVerifierSvc.DeleteAccount(ctx, DeleteAccountInput{
		UserID:        userID,
		Email:         "siti@gmail.com",
		GoogleIDToken: "tok",
		Nonce:         nonceLogin,
		HasPassword:   false,
	}); !errors.Is(err, ErrOAuthUnavailable) {
		t.Fatalf("expected ErrOAuthUnavailable when googleVerifier is nil, got %v", err)
	}
}
