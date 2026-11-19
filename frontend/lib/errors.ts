import type { Language } from "@/lib/types";

export type ErrorCode =
  | "INVALID_CREDENTIALS"
  | "EMAIL_NOT_VERIFIED"
  | "EMAIL_ALREADY_EXISTS"
  | "WEAK_PASSWORD"
  | "INVALID_OR_EXPIRED_TOKEN"
  | "INVALID_INPUT"
  | "TOO_MANY_REQUESTS"
  | "UNAUTHORIZED"
  | "NOT_FOUND"
  | "NETWORK_ERROR"
  | "INTERNAL_ERROR";

const ERROR_MESSAGES: Record<Language, Record<ErrorCode, string>> = {
  id: {
    INVALID_CREDENTIALS:
      "Email atau kata sandi tidak cocok. Periksa kembali, lalu coba masuk lagi.",
    EMAIL_NOT_VERIFIED:
      "Email kamu belum diverifikasi. Periksa kotak masuk emailmu dan buka tautan verifikasi terlebih dahulu.",
    EMAIL_ALREADY_EXISTS:
      "Email ini sudah terdaftar. Silakan masuk atau gunakan email lain.",
    WEAK_PASSWORD:
      "Kata sandi minimal 8 karakter dan tidak boleh terlalu umum. Pilih kata sandi lain.",
    INVALID_OR_EXPIRED_TOKEN:
      "Tautan sudah tidak berlaku atau kedaluwarsa. Minta tautan baru untuk melanjutkan.",
    INVALID_INPUT:
      "Periksa kembali isian formulir kamu, lalu coba kirim lagi.",
    TOO_MANY_REQUESTS:
      "Kamu mengirim terlalu banyak permintaan. Tunggu sebentar, lalu coba lagi.",
    UNAUTHORIZED: "Sesi kamu sudah berakhir. Silakan masuk kembali.",
    NOT_FOUND: "Data yang kamu cari tidak ditemukan.",
    NETWORK_ERROR:
      "Koneksi internetmu sepertinya terputus. Periksa koneksi, lalu muat ulang halaman.",
    INTERNAL_ERROR:
      "Terjadi kendala di sisi kami. Coba lagi sebentar lagi, dan muat ulang halaman jika masih berlanjut.",
  },
  en: {
    INVALID_CREDENTIALS:
      "Email or password does not match. Please check and try again.",
    EMAIL_NOT_VERIFIED:
      "Your email is not verified yet. Please check your inbox and open the verification link first.",
    EMAIL_ALREADY_EXISTS:
      "This email is already registered. Please sign in or use another email.",
    WEAK_PASSWORD:
      "Password must be at least 8 characters and not too common. Please choose another password.",
    INVALID_OR_EXPIRED_TOKEN:
      "This link is invalid or has expired. Please request a new link to continue.",
    INVALID_INPUT: "Please check your form inputs and try submitting again.",
    TOO_MANY_REQUESTS:
      "You have made too many requests. Please wait a moment and try again.",
    UNAUTHORIZED: "Your session has ended. Please sign in again.",
    NOT_FOUND: "The requested data could not be found.",
    NETWORK_ERROR:
      "Your internet connection appears to be offline. Please check your connection and reload.",
    INTERNAL_ERROR:
      "Something went wrong on our side. Please try again shortly, and reload if it continues.",
  },
};

export function getFriendlyErrorMessage(
  code: string | undefined,
  language: Language
): string {
  const dict = ERROR_MESSAGES[language];
  if (code && Object.hasOwn(dict, code)) {
    return dict[code as ErrorCode];
  }
  return dict.INTERNAL_ERROR;
}
