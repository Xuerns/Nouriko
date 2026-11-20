package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"Nouriko/internal/profile"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost is the bcrypt work factor enforced for local password hashing (PRD F15).
	BcryptCost = 12
	// MinPasswordRunes is the minimum password length in Unicode characters.
	MinPasswordRunes = 8
	// MaxPasswordBytes is the bcrypt 72-byte maximum input size.
	MaxPasswordBytes = 72
	// TokenByteLength is the entropy size (256 bits) for single-use auth tokens.
	TokenByteLength = 32
)

// commonWeakPasswords contains widely breached or trivially guessable passwords
// (stored lowercase for case-insensitive matching).
var commonWeakPasswords = map[string]struct{}{
	"12345678":      {},
	"123456789":     {},
	"1234567890":    {},
	"0123456789":    {},
	"87654321":      {},
	"987654321":     {},
	"00000000":      {},
	"11111111":      {},
	"password":      {},
	"password1":     {},
	"password12":    {},
	"password123":   {},
	"passw0rd":      {},
	"p@ssword":      {},
	"p@ssw0rd":      {},
	"qwerty12":      {},
	"qwerty123":     {},
	"qwertyui":      {},
	"qwertyuiop":    {},
	"asdfghjk":      {},
	"asdfghjkl":     {},
	"zxcvbnm1":      {},
	"abcdefgh":      {},
	"abc12345":      {},
	"abcd1234":      {},
	"iloveyou":      {},
	"admin123":      {},
	"admin1234":     {},
	"administrator": {},
	"welcome1":      {},
	"welcome123":    {},
	"letmein1":      {},
	"letmein123":    {},
	"monkey123":     {},
	"dragon123":     {},
	"football":      {},
	"baseball":      {},
	"sunshine":      {},
	"princess":      {},
	"starwars":      {},
	"trustno1":      {},
	"whatever":      {},
	"superman":      {},
	"master12":      {},
	"changeme":      {},
	"indonesia":     {},
	"katasandi":     {},
	"rahasia1":      {},
	"rahasia123":    {},
	"nouriko1":      {},
	"nouriko123":    {},
	"test1234":      {},
	"user1234":      {},
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// ValidatePassword enforces PRD F15 password rules:
// minimum 8 characters, maximum 72 bytes (bcrypt limit), no mandatory
// uppercase/symbol rules, and rejection of common/weak passwords.
func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < MinPasswordRunes {
		return ErrWeakPassword
	}
	if len(password) > MaxPasswordBytes {
		return ErrWeakPassword
	}

	normalized := strings.ToLower(strings.TrimSpace(password))
	if normalized == "" {
		return ErrWeakPassword
	}
	if _, blocked := commonWeakPasswords[normalized]; blocked {
		return ErrWeakPassword
	}
	if isAllSameRune(normalized) {
		return ErrWeakPassword
	}

	return nil
}

func isAllSameRune(s string) bool {
	var first rune
	for i, r := range s {
		if i == 0 {
			first = r
			continue
		}
		if r != first {
			return false
		}
	}
	return s != ""
}

// HashPassword hashes a plaintext password with bcrypt cost 12.
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hashed), nil
}

// VerifyPassword compares a bcrypt hash with a plaintext password in constant time.
func VerifyPassword(hash, password string) bool {
	if hash == "" || password == "" || len(password) > MaxPasswordBytes {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// DummyPasswordCheck runs a real bcrypt comparison against a dummy cost-12 hash
// to reduce timing side channels when an email does not exist on login.
func DummyPasswordCheck(password string) {
	dummyHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("nouriko-timing-mitigation-dummy-secret"), BcryptCost)
		if err == nil {
			dummyHash = h
		}
	})
	if len(dummyHash) > 0 && len(password) <= MaxPasswordBytes {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	}
}

// GenerateRandomToken generates a 256-bit cryptographically random hex token and
// its SHA-256 hex digest (only the digest is stored in the database).
func GenerateRandomToken() (rawHex string, sha256Hex string, err error) {
	buf := make([]byte, TokenByteLength)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate random token: %w", err)
	}
	rawHex = hex.EncodeToString(buf)
	sha256Hex = HashToken(rawHex)
	return rawHex, sha256Hex, nil
}

// HashToken returns the lowercase hex-encoded SHA-256 digest of a raw token.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NormalizeAndValidateEmail trims, lowercases, and validates an email address.
func NormalizeAndValidateEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" || len(email) > 254 || strings.ContainsAny(email, " \t\r\n") {
		return "", ErrInvalidInput
	}

	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", ErrInvalidInput
	}

	at := strings.LastIndex(email, "@")
	if at <= 0 || at >= len(email)-1 {
		return "", ErrInvalidInput
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidInput
	}

	return email, nil
}

// DiceBearAvatarURL builds the deterministic DiceBear Waves SVG avatar URL (PRD F0).
func DiceBearAvatarURL(userID string) string {
	return "https://api.dicebear.com/10.x/waves/svg?seed=" + url.QueryEscape(userID)
}

// ValidateLanguage normalizes and validates UI language ("id" or "en").
func ValidateLanguage(raw string, defaultIfEmpty bool) (string, error) {
	v, err := profile.ValidateLanguage(raw, defaultIfEmpty)
	if err != nil {
		return "", ErrInvalidInput
	}
	return v, nil
}

// ValidateTheme normalizes and validates UI theme ("light" or "dark").
func ValidateTheme(raw string, defaultIfEmpty bool) (string, error) {
	v, err := profile.ValidateTheme(raw, defaultIfEmpty)
	if err != nil {
		return "", ErrInvalidInput
	}
	return v, nil
}
