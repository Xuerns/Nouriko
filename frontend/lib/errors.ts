import type { Language } from "@/lib/types";

export type ErrorCode =
  | "INVALID_CREDENTIALS"
  | "EMAIL_NOT_VERIFIED"
  | "EMAIL_ALREADY_EXISTS"
  | "WEAK_PASSWORD"
  | "INVALID_OR_EXPIRED_TOKEN"
  | "INVALID_INPUT"
  | "PHOTO_INVALID"
  | "PHOTO_UNRECOGNIZED"
  | "PAYLOAD_TOO_LARGE"
  | "AI_UNAVAILABLE"
  | "AI_TIMEOUT"
  | "ONBOARDING_REQUIRED"
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
    PHOTO_INVALID:
      "Foto tidak dapat diproses. Pilih gambar JPG, PNG, atau WebP dengan ukuran maksimal 5 MB.",
    PHOTO_UNRECOGNIZED:
      "Makanan atau bahan pada foto belum bisa dikenali. Coba ambil foto yang lebih jelas atau tulis nama makanannya di chat.",
    PAYLOAD_TOO_LARGE:
      "Ukuran foto atau pesan terlalu besar. Pilih foto yang lebih kecil lalu coba kirim lagi.",
    AI_UNAVAILABLE:
      "Layanan rekomendasi sedang sibuk atau belum tersedia saat ini. Coba kirim ulang sebentar lagi.",
    AI_TIMEOUT:
      "Permintaan membutuhkan waktu terlalu lama. Periksa koneksi kamu, lalu coba kirim ulang.",
    ONBOARDING_REQUIRED:
      "Lengkapi data profil kamu terlebih dahulu sebelum menggunakan chat nutrisi.",
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
    PHOTO_INVALID:
      "This photo could not be processed. Please choose a JPG, PNG, or WebP image up to 5 MB.",
    PHOTO_UNRECOGNIZED:
      "The food or ingredients in this photo could not be recognized. Try a clearer photo or type the food name in chat.",
    PAYLOAD_TOO_LARGE:
      "The photo or message is too large. Please choose a smaller photo and try again.",
    AI_UNAVAILABLE:
      "The recommendation service is temporarily unavailable. Please try sending your message again shortly.",
    AI_TIMEOUT:
      "The request took too long to complete. Please check your connection and try sending again.",
    ONBOARDING_REQUIRED:
      "Please complete your profile details first before using the nutrition chat.",
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
