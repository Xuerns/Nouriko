const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;

export function isValidEmailFormat(raw: string): boolean {
  const trimmed = raw.trim();
  if (!trimmed || trimmed.length > 254) return false;
  return EMAIL_PATTERN.test(trimmed);
}

export function focusFieldById(id: string): void {
  if (typeof document === "undefined") return;
  const el = document.getElementById(id);
  if (el instanceof HTMLElement) {
    el.focus();
  }
}
