package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

const (
	defaultOAuthReplayTTL = 2 * time.Hour
	minOAuthReplayTTL     = time.Hour
	maxOAuthReplayTTL     = 24 * time.Hour
)

// OAuthReplayStore enforces single-use consumption of verified OAuth nonces across
// endpoints (LoginWithGoogle and DeleteAccount) and serverless instances.
type OAuthReplayStore interface {
	ConsumeOAuthNonce(ctx context.Context, nonceHash string, expiresAt time.Time) (bool, error)
}

func oauthReplayKey(rawNonce string) string {
	sum := sha256.Sum256([]byte(rawNonce))
	return hex.EncodeToString(sum[:])
}

func oauthReplayExpiry(identity *GoogleIdentity, now time.Time) time.Time {
	if identity != nil && !identity.ExpiresAt.IsZero() && identity.ExpiresAt.After(now) {
		ttl := identity.ExpiresAt.Sub(now)
		if ttl < minOAuthReplayTTL {
			return now.Add(minOAuthReplayTTL)
		}
		if ttl > maxOAuthReplayTTL {
			return now.Add(maxOAuthReplayTTL)
		}
		return identity.ExpiresAt
	}
	return now.Add(defaultOAuthReplayTTL)
}

// memOAuthReplayStore is an in-memory fallback OAuthReplayStore used when no
// PostgreSQL-backed store is injected (such as isolated unit tests).
type memOAuthReplayStore struct {
	mu      sync.Mutex
	entries map[string]time.Time
	now     func() time.Time
}

func newMemOAuthReplayStore() *memOAuthReplayStore {
	return &memOAuthReplayStore{
		entries: make(map[string]time.Time),
		now:     time.Now,
	}
}

func (m *memOAuthReplayStore) ConsumeOAuthNonce(_ context.Context, nonceHash string, expiresAt time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if m.now != nil {
		now = m.now()
	}

	for k, exp := range m.entries {
		if !exp.After(now) {
			delete(m.entries, k)
		}
	}

	if existingExp, exists := m.entries[nonceHash]; exists && existingExp.After(now) {
		return false, nil
	}
	m.entries[nonceHash] = expiresAt
	return true, nil
}
