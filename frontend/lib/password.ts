export type PasswordStrength = "empty" | "weak" | "fair" | "strong";

const COMMON_PASSWORDS = new Set([
  "12345678",
  "123456789",
  "1234567890",
  "password",
  "password1",
  "password123",
  "qwerty123",
  "qwertyui",
  "iloveyou",
  "indonesia",
  "rahasia123",
  "admin123",
  "letmein1",
  "welcome1",
  "abc12345",

]);

export function evaluatePasswordStrength(password: string): PasswordStrength {
  if (!password) {
    return "empty";
  }

  const trimmedLower = password.trim().toLowerCase();
  const uniqueChars = new Set(password).size;

  if (
    password.length < 8 ||
    COMMON_PASSWORDS.has(trimmedLower) ||
    uniqueChars <= 2
  ) {
    return "weak";
  }

  let score = 0;
  if (password.length >= 10) score += 1;
  if (password.length >= 14) score += 1;
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score += 1;
  if (/\d/.test(password)) score += 1;
  if (/[^a-zA-Z0-9]/.test(password)) score += 1;

  if (score >= 3) {
    return "strong";
  }
  return "fair";
}
