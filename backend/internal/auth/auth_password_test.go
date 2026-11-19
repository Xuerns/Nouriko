package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestValidatePassword(t *testing.T) {
	validPasswords := []string{
		"kopisusupagi",
		"mysecurepassword",
		"nasi-goreng-2026",
		"1234567a",
		strings.Repeat("a", 71) + "b", // 72 bytes exact
	}
	for _, pw := range validPasswords {
		if err := ValidatePassword(pw); err != nil {
			t.Errorf("expected password %q to be valid, got %v", pw, err)
		}
	}

	weakPasswords := []string{
		"",
		"short",
		"1234567",
		"password",
		"Password123",
		"12345678",
		"qwerty123",
		"indonesia",
		"aaaaaaaa",
		"        ",
		strings.Repeat("a", 73), // > 72 bytes
	}
	for _, pw := range weakPasswords {
		if err := ValidatePassword(pw); !errors.Is(err, ErrWeakPassword) {
			t.Errorf("expected password %q to return ErrWeakPassword, got %v", pw, err)
		}
	}
}

func TestHashAndVerifyPasswordCost12(t *testing.T) {
	hash, err := HashPassword("kopisusupagi")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("bcrypt.Cost: %v", err)
	}
	if cost != BcryptCost {
		t.Fatalf("expected cost %d, got %d", BcryptCost, cost)
	}
	if !VerifyPassword(hash, "kopisusupagi") {
		t.Fatal("expected VerifyPassword to succeed")
	}
	if VerifyPassword(hash, "wrongpassword") {
		t.Fatal("expected VerifyPassword to fail on wrong password")
	}
}

func TestGenerateRandomTokenAndHash(t *testing.T) {
	raw1, hash1, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken: %v", err)
	}
	raw2, hash2, err := GenerateRandomToken()
	if err != nil {
		t.Fatalf("GenerateRandomToken: %v", err)
	}
	if len(raw1) != 64 || len(hash1) != 64 {
		t.Fatalf("expected 64-char hex strings, got raw=%d hash=%d", len(raw1), len(hash1))
	}
	if raw1 == raw2 || hash1 == hash2 {
		t.Fatal("expected unique random tokens")
	}
	if HashToken(raw1) != hash1 {
		t.Fatal("HashToken mismatch")
	}
}

func TestNormalizeAndValidateEmail(t *testing.T) {
	got, err := NormalizeAndValidateEmail("  Raka@Example.COM ")
	if err != nil || got != "raka@example.com" {
		t.Fatalf("expected raka@example.com, got %q (err=%v)", got, err)
	}

	invalid := []string{"", "plain", "a@b", "a@.com", "a@b.", "Raka <raka@example.com>", "a b@example.com"}
	for _, email := range invalid {
		if _, err := NormalizeAndValidateEmail(email); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput for %q, got %v", email, err)
		}
	}
}

func TestDiceBearAvatarURL(t *testing.T) {
	id := "550e8400-e29b-41d4-a716-446655440000"
	want := "https://api.dicebear.com/10.x/slice/svg?seed=550e8400-e29b-41d4-a716-446655440000"
	if got := DiceBearAvatarURL(id); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
