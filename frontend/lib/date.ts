import type { Language } from "@/lib/types";

const DATE_PATTERN = /^(\d{4})-(\d{2})-(\d{2})$/;

function pad2(value: number): string {
  return String(value).padStart(2, "0");
}

export function getLocalDateString(date: Date = new Date()): string {
  const year = date.getFullYear();
  const month = pad2(date.getMonth() + 1);
  const day = pad2(date.getDate());
  return `${year}-${month}-${day}`;
}

export function isValidDateString(
  value: string | null | undefined
): value is string {
  if (!value) return false;
  const match = DATE_PATTERN.exec(value);
  if (!match) return false;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  if (month < 1 || month > 12 || day < 1 || day > 31) return false;
  const parsed = new Date(year, month - 1, day);
  return (
    parsed.getFullYear() === year &&
    parsed.getMonth() === month - 1 &&
    parsed.getDate() === day
  );
}

export function parseLocalDate(dateStr: string): Date | null {
  if (!isValidDateString(dateStr)) return null;
  const [year, month, day] = dateStr.split("-").map(Number);
  return new Date(year, month - 1, day);
}

export function shiftDateString(dateStr: string, deltaDays: number): string {
  const base = parseLocalDate(dateStr) ?? new Date();
  const shifted = new Date(
    base.getFullYear(),
    base.getMonth(),
    base.getDate() + deltaDays
  );
  return getLocalDateString(shifted);
}

export function formatDisplayDate(dateStr: string, language: Language): string {
  const parsed = parseLocalDate(dateStr);
  if (!parsed) return dateStr;
  const locale = language === "id" ? "id-ID" : "en-US";
  return new Intl.DateTimeFormat(locale, {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  }).format(parsed);
}

export function formatShortChartDate(
  dateStr: string,
  language: Language
): { day: string; date: string } {
  const parsed = parseLocalDate(dateStr);
  if (!parsed) {
    return { day: "-", date: dateStr };
  }
  const locale = language === "id" ? "id-ID" : "en-US";
  const day = new Intl.DateTimeFormat(locale, { weekday: "short" }).format(
    parsed
  );
  const date = `${pad2(parsed.getDate())}/${pad2(parsed.getMonth() + 1)}`;
  return { day, date };
}
