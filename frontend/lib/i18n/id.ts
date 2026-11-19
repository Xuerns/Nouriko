import type { Dictionary } from "@/lib/i18n/types";

export const idDictionary: Dictionary = {
  common: {
    brand: "Nouriko",
    loading: "Memuat...",
    save: "Simpan",
    saving: "Menyimpan...",
    cancel: "Batal",
    submit: "Kirim",
    submitting: "Memproses...",
    backHome: "Kembali ke Beranda",
    privacyPolicy: "Kebijakan Privasi",
    languageLabel: "Bahasa",
    themeLabel: "Tema",
    themeLight: "Terang",
    themeDark: "Gelap",
    showPassword: "Tampilkan",
    hidePassword: "Sembunyikan",
    passwordStrengthLabel: "Kekuatan kata sandi",
    passwordWeak: "Lemah",
    passwordFair: "Cukup",
    passwordStrong: "Kuat",
    passwordMinHint: "Minimal 8 karakter, tidak boleh terlalu umum.",
    passwordMismatch: "Konfirmasi kata sandi tidak cocok.",
    emailLabel: "Email",
    emailPlaceholder: "nama@contoh.com",
    passwordLabel: "Kata Sandi",
    confirmPasswordLabel: "Konfirmasi Kata Sandi",
    newPasswordLabel: "Kata Sandi Baru",
    currentPasswordLabel: "Kata Sandi Saat Ini",
  },
  nav: {
    chat: "Chat",
    profile: "Profil",
    logout: "Keluar",
    login: "Masuk",
    register: "Daftar",
  },
  footer: {
    copyright: "Nouriko — Asisten Nutrisi Harian",
    privacyLink: "Kebijakan Privasi",
  },
  landing: {
    eyebrow: "ASISTEN NUTRISI HARIAN",
    title: "Nouriko",
    tagline:
      "Rekomendasi menu harian dan perhitungan nutrisi sesuai bahan yang ada, anggaran belanja, serta foto makanan kamu.",
    ctaRegister: "Daftar",
    ctaLogin: "Masuk",
    ctaOpenChat: "Buka Chat",
    ctaContinueOnboarding: "Lengkapi Profil",
    capabilitiesHeading: "Kemampuan Utama",
    capabilities: [
      {
        title: "Rekomendasi dari Bahan",
        description:
          "Masukkan bahan yang tersedia di dapur atau kulkas untuk mendapatkan rekomendasi menu beserta perkiraan kalori dan makronya.",
      },
      {
        title: "Sesuai Anggaran",
        description:
          "Tentukan anggaran makan harian atau mingguan agar rekomendasi menu tetap bergizi tanpa melebihi batas pengeluaran.",
      },
      {
        title: "Analisis Foto Makanan",
        description:
          "Unggah foto makanan untuk melihat perkiraan komposisi bahan, kalori, protein, karbohidrat, lemak, dan serat.",
      },
    ],
    stepsHeading: "Cara Kerja",
    steps: [
      {
        step: "01",
        title: "Buat Akun dan Isi Profil",
        description:
          "Daftar dengan email, verifikasi akun, lalu isi data tubuh agar target kalori dan makro harian dihitung otomatis.",
      },
      {
        step: "02",
        title: "Tulis Bahan, Anggaran, atau Unggah Foto",
        description:
          "Buka halaman chat dan sampaikan kebutuhan makanmu dalam bahasa sehari-hari atau kirim foto makanan.",
      },
      {
        step: "03",
        title: "Simpan dan Pantau Menu",
        description:
          "Tinjau rekomendasi nutrisi yang sesuai target harianmu dan sesuaikan profil kapan saja bila kebutuhan berubah.",
      },
    ],
  },
  register: {
    title: "Daftar Akun",
    submitButton: "Daftar",
    hasAccount: "Sudah punya akun?",
    loginLink: "Masuk",
    privacyPrefix: "Dengan mendaftar, kamu dapat membaca ketentuan data di",
    successTitle: "Periksa Email Kamu",
    successMessage:
      "Akun berhasil dibuat. Buka tautan verifikasi yang dikirim ke emailmu sebelum masuk.",
    devTokenLabel: "Tautan verifikasi lokal:",
    devVerifyLink: "Verifikasi Email Sekarang",
    goToVerify: "Masukkan Token Verifikasi",
  },
  verifyEmail: {
    title: "Verifikasi Email",
    tokenLabel: "Token Verifikasi",
    tokenPlaceholder: "Tempel token verifikasi di sini",
    submitButton: "Verifikasi Email",
    verifying: "Memverifikasi email...",
    successTitle: "Email Terverifikasi",
    successMessage: "Email kamu berhasil diverifikasi. Sekarang kamu bisa masuk.",
    goToLogin: "Lanjut ke Halaman Masuk",
  },
  login: {
    title: "Masuk",
    submitButton: "Masuk",
    forgotPasswordLink: "Lupa kata sandi?",
    noAccount: "Belum punya akun?",
    registerLink: "Daftar",
    sessionExpired: "Sesi kamu sudah berakhir. Silakan masuk kembali.",
  },
  forgotPassword: {
    title: "Lupa Kata Sandi",
    submitButton: "Kirim Tautan Reset",
    successMessage:
      "Jika email tersebut terdaftar, kami telah mengirimkan tautan untuk mengatur ulang kata sandi.",
    devResetLink: "Buka Halaman Reset Kata Sandi",
    backToLogin: "Kembali ke Masuk",
  },
  resetPassword: {
    title: "Atur Ulang Kata Sandi",
    tokenLabel: "Token Reset",
    tokenPlaceholder: "Tempel token reset di sini",
    submitButton: "Simpan Kata Sandi Baru",
    successMessage:
      "Kata sandi kamu berhasil diperbarui. Silakan masuk dengan kata sandi baru.",
    goToLogin: "Masuk Sekarang",
  },
  onboarding: {
    title: "Data Profil",
    heightLabel: "Tinggi Badan (cm)",
    weightLabel: "Berat Badan (kg)",
    ageLabel: "Usia (tahun)",
    genderLabel: "Jenis Kelamin",
    genderMale: "Laki-laki",
    genderFemale: "Perempuan",
    activityLabel: "Tingkat Aktivitas",
    activityLight: "Ringan",
    activityModerate: "Sedang",
    activityHeavy: "Berat",
    submitButton: "Simpan",
  },
  profile: {
    title: "Profil & Pengaturan Akun",
    summaryHeading: "Ringkasan Target Harian",
    avatarAlt: "Avatar profil pengguna",
    dailyCalories: "Target Kalori",
    protein: "Protein",
    carbs: "Karbohidrat",
    fat: "Lemak",
    fiber: "Serat",
    kcalUnit: "kkal",
    gramUnit: "g",
    editProfileHeading: "Data Tubuh",
    macroTargetsHeading: "Target Kalori & Makro Harian",
    recalcNote:
      "Mengubah data tubuh akan menghitung ulang target harian secara otomatis, kecuali kamu mengisi angka target khusus.",
    saveProfileButton: "Simpan Perubahan",
    profileSaved: "Profil berhasil diperbarui.",
    changePasswordHeading: "Ganti Kata Sandi",
    changePasswordButton: "Perbarui Kata Sandi",
    passwordChanged: "Kata sandi berhasil diperbarui.",
    privacyHeading: "Privasi & Data",
    privacyLinkText: "Baca Kebijakan Privasi Nouriko",
    deleteAccountHeading: "Hapus Akun",
    deleteAccountButton: "Hapus Akun",
    deleteDialogTitle: "Hapus Akun",
    deleteDialogQuestion: "Apakah kamu yakin ingin menghapus akun ini?",
    deleteDialogWarning:
      "Tindakan ini tidak dapat dibatalkan. Seluruh data akun dan riwayat tersimpan akan dihapus permanen.",
    deleteDialogPasswordLabel: "Masukkan Kata Sandi",
    deleteDialogConfirm: "Hapus",
    deleteDialogDeleting: "Menghapus...",
  },
  privacy: {
    title: "Kebijakan Privasi",
    updatedLabel: "Terakhir diperbarui: 19 November 2026",
    sections: [
      {
        heading: "1. Data yang Kami Kumpulkan",
        paragraphs: [
          "Nouriko hanya mengumpulkan data yang diperlukan agar aplikasi dapat berfungsi sesuai kebutuhan nutrisi harian kamu.",
        ],
        items: [
          "Data akun: alamat email dan kata sandi tersandi untuk autentikasi.",
          "Data profil tubuh: tinggi badan, berat badan, usia, jenis kelamin, tingkat aktivitas, serta target kalori dan makro harian.",
          "Preferensi aplikasi: pilihan bahasa (Indonesia atau Inggris) dan tema tampilan (Terang atau Gelap).",
          "Data menu tersimpan yang kamu pilih untuk disimpan secara mandiri.",
        ],
      },
      {
        heading: "2. Tujuan Penggunaan Data",
        paragraphs: [
          "Data profil tubuh digunakan untuk menghitung kebutuhan kalori dan makronutrisi harian secara personal. Alamat email digunakan untuk verifikasi akun, masuk aplikasi, dan pemulihan kata sandi. Kami tidak menjual atau membagikan data pribadimu kepada pihak ketiga untuk keperluan iklan.",
        ],
      },
      {
        heading: "3. Penyimpanan dan Retensi Data",
        paragraphs: [
          "Data akun, profil, dan menu tersimpan disimpan pada basis data yang terisolasi per akun pengguna. Percakapan chat bersifat sementara selama sesi berlangsung, dan foto makanan yang kamu unggah hanya diproses untuk analisis nutrisi tanpa disimpan permanen di server kami.",
        ],
      },
      {
        heading: "4. Hak Pengguna dan Penghapusan Akun",
        paragraphs: [
          "Kamu berhak melihat, memperbarui, dan menghapus data pribadimu kapan saja melalui halaman Profil.",
        ],
        items: [
          "Memperbarui data tubuh, target nutrisi, bahasa, tema, serta kata sandi kapan saja.",
          "Menghapus akun secara permanen melalui menu Hapus Akun di halaman Profil dengan verifikasi kata sandi. Penghapusan akun akan menghapus seluruh data profil dan menu tersimpan secara permanen.",
        ],
      },
      {
        heading: "5. Bukan Alat Medis",
        paragraphs: [
          "Informasi nutrisi, estimasi kalori, dan rekomendasi menu dari Nouriko bersifat informatif untuk membantu perencanaan makan sehari-hari. Nouriko bukan alat medis dan tidak menggantikan saran dokter, ahli gizi klinis, atau tenaga kesehatan profesional.",
        ],
      },
      {
        heading: "6. Dasar Hukum",
        paragraphs: [
          "Pengelolaan data pribadi di Nouriko mengacu pada Undang-Undang Nomor 27 Tahun 2022 tentang Pelindungan Data Pribadi (UU PDP) di Indonesia, termasuk prinsip persetujuan pengguna, pembatasan tujuan, keamanan penyimpanan, dan hak penghapusan data.",
        ],
      },
    ],
  },
  chat: {
    title: "Chat Nutrisi",
    subtitle: "Kerangka halaman chat siap digunakan untuk sesi berikutnya.",
    targetSummary: "Target harian kamu",
    inputPlaceholder:
      "Tulis bahan yang tersedia, anggaran, atau kebutuhan makanmu...",
    sendButton: "Kirim",
    shellNotice:
      "Akun dan profil kamu sudah siap. Fitur percakapan nutrisi akan diaktifkan pada tahap berikutnya.",
    editProfileLink: "Ubah Profil",
  },
  notFound: {
    title: "Halaman Tidak Ditemukan",
    description:
      "Halaman yang kamu tuju tidak tersedia atau tautannya sudah berubah.",
    backHome: "Kembali ke Beranda",
  },
};
