# DESIGN.md — Nouriko Design System

Dokumen ini menjadi panduan desain antarmuka (UI/UX) dan standar implementasi komponen untuk aplikasi web **Nouriko** (*Menu Bergizi Anak Kos*). Seluruh keputusan warna, tipografi, kelengkungan sudut (*border-radius*), dan pemilihan komponen dirancang agar terasa **soft, hangat, ramah, dan tenang**, sekaligus mematuhi kebijakan **Anti-AI-Slop** dari `SKILL.md` serta kebijakan **komponen siap pakai (*use-existing-first*)**.

---

## 1. Filosofi Visual: "Soft Organic Calm"

Nouriko digunakan setiap hari oleh mahasiswa dan anak kos dalam kondisi lapar, lelah, atau terbatas secara budget. Antarmuka harus terasa lembut dan bersahabat—bukan seperti aplikasi medis yang kaku atau dashboard *tech* yang tajam.

* **Hangat & Lembut (*Soft*):** Menggunakan latar belakang putih gading hangat (*warm alabaster*) pada mode terang dan arang hijau-gelap lembut (*warm deep charcoal*) pada mode gelap—bukan putih murni `#FFFFFF` yang menyilaukan atau hitam pekat `#000000` yang keras.
* **Sudut Melengkung Luwes (*Generous Soft Radius*):** Menggunakan kelengkungan sudut yang besar (`rounded-2xl`, `rounded-3xl`, dan `rounded-full`) secara terstruktur agar setiap kartu, tombol, dan area chat terasa empuk, organik, dan tidak kaku.
* **Flat & Matte (Tanpa Bayangan):** Kedalaman visual dibentuk murni melalui lapisan warna permukaan yang lembut (*tonal layering*) dan garis pembatas `1px` yang sangat halus—tanpa `box-shadow`, tanpa *drop-shadow*, dan tanpa *glow*.
* **Terinspirasi dari Logo:** Menurunkan saturasi warna hijau daun dan oranye hangat dari logo Nouriko menjadi palet *botanical sage* dan *soft terracotta/apricot*.
* **Jelas & Tanpa Basa-Basi:** Mengikuti aturan PRD, tidak ada teks penjelas panjang atau dasar hukum di UI utama.

---

## 2. Kepatuhan Anti-AI-Slop (`SKILL.md`) & Pengecualian Resmi

Seluruh implementasi UI **wajib** mematuhi aturan `SKILL.md` dengan satu pengecualian resmi yang diaktifkan melalui dokumen ini:

1. **Pengecualian Resmi Border-Radius (`DESIGN.md` Override):** Sesuai klausul pengecualian pada `SKILL.md` Bagian 5, dokumen ini **secara eksplisit melonggarkan batas `rounded-xl`** dan mewajibkan penggunaan `rounded-2xl`, `rounded-3xl`, dan `rounded-full` sebagaimana diatur rinci di **Bagian 5**.
2. **Tanpa Eyebrow Pill & Titik Indikator (Tetap Berlaku):** Meskipun `rounded-full` diizinkan untuk tombol, chip, dan badge skor, **dilarang keras** membuat label *eyebrow* berbentuk kapsul di atas judul hero/section yang memiliki titik bulat (*dot indicator*) di kiri maupun kanan. Eyebrow tetap berupa teks polos.
3. **Tanpa Border Terang/Neon (Tetap Berlaku):** Semua kartu dan kontainer hanya memakai *border* netral `1px` yang sangat lembut atau tanpa *border* (hanya kontras latar solid).
4. **Tanpa Shadow & Glow (Tetap Berlaku):** Saat menggunakan komponen dari **shadcn/ui** atau **React Bits**, semua kelas bawaan `shadow-xs`, `shadow-sm`, `shadow-md`, `shadow-lg`, atau efek *glow* **wajib dihapus/dioverride menjadi `shadow-none`**.
5. **Tanpa Animasi Pulse/Ping/Bounce (Tetap Berlaku):** Hindari `animate-pulse`, `animate-ping`, dan `animate-bounce`. Jika memakai komponen `Skeleton` dari shadcn/ui, **hapus `animate-pulse`** (ganti dengan permukaan statis lembut atau transisi fade halus).
6. **Tanpa Contained/Boxed Icon Header (Tetap Berlaku):** Dilarang membungkus ikon di dalam kotak berwarna/ber-border di sebelah atau di atas judul. Gunakan judul teks polos atau ikon Lucide polos tanpa wadah kotak.

---

## 3. Kebijakan Komponen Siap Pakai (*Use-Existing-First*)

Untuk menjaga konsistensi, kecepatan pengerjaan, dan aksesibilitas (keyboard navigation, ARIA, screen reader), pengembang dan AI **dilarang membangun komponen UI standar secara manual dari nol** apabila sudah tersedia di ekosistem React/Tailwind.

### 3.1. Hierarki Sumber Komponen (Urutan Prioritas)

1. **Prioritas 1 — `shadcn/ui` (Wajib untuk Seluruh Primitif UI):**
   * Sumber utama untuk seluruh elemen interaktif, form, overlay, navigasi, data display, dan notifikasi.
   * Dipasang langsung via CLI (`npx shadcn@latest add ...`) dan dikustomisasi hanya pada variabel tema (`globals.css`), `border-radius`, serta penghapusan `shadow-*`.
2. **Prioritas 2 — `React Bits` (`reactbits.dev`) & Ekosistem Kompatibel:**
   * Digunakan untuk interaksi halus, transisi masuk konten yang lembut di Landing Page, atau komponen ekspresif yang tidak tersedia di `shadcn/ui` (misalnya `FadeContent`, `AnimatedList` dengan transisi sangat lembut, atau `Vaul` / `Drawer` untuk bottom sheet mobile).
   * **Syarat Filter:** Hanya pilih komponen yang tenang, mendukung `prefers-reduced-motion`, tidak menggunakan animasi looping (`pulse`/`bounce`), dan tidak memiliki efek neon/glow.
3. **Prioritas 3 — Pustaka Pendukung Resmi Ekosistem shadcn:**
   * **Ikon:** `lucide-react` (polos, stroke `1.75` agar terasa lebih *soft* dibanding stroke tebal).
   * **Notifikasi Toast:** `sonner` (integrasi resmi `shadcn/ui`).
   * **Tema Terang/Gelap:** `next-themes`.
   * **Grafik (F9 Riwayat Mingguan):** `shadcn/ui Charts` (`recharts`).
4. **Prioritas 4 — Komposisi Custom (Hanya Jika Tidak Tersedia):**
   * Komponen spesifik domain seperti `MealResultCard` (F4) atau `ChatComposer` (F1) **tidak dibuat dari elemen HTML mentah**, melainkan **dikomposisikan dari primitif `shadcn/ui`** (`Card`, `Badge`, `Collapsible`, `Button`, `Textarea`, `Select`, `Separator`).

### 3.2. Peta Pemetaan Komponen PRD ke Pustaka Siap Pakai

| Fitur PRD | Kebutuhan UI | Komponen Siap Pakai (Wajib Dipakai) | Catatan Kustomisasi Soft |
| --- | --- | --- | --- |
| **F0 & F15** Auth & Keamanan Sandi | Form login/daftar, error, kekuatan sandi | `shadcn`: `Card`, `Form`, `Input`, `Label`, `Button`, `Alert`, `Progress`, `Avatar` | `Card` `rounded-3xl shadow-none`, `Button` `rounded-full`, `Input` `rounded-2xl` |
| **F1** Antarmuka Chat Tunggal | Area pesan, input teks + foto, prompt saran | `shadcn`: `ScrollArea`, `Textarea`, `Button`, `Badge`, `Tooltip` + `React Bits`: `FadeContent` (opsional untuk masuknya kartu hasil) | Composer `rounded-3xl`, starter chips `rounded-full`, bubble chat `rounded-3xl` |
| **F2, F3, F4** Kartu Hasil Makanan | Kartu rekomendasi, skor, makro, cara buat, simpan | `shadcn`: `Card`, `Badge`, `Collapsible` (atau `Accordion`), `Separator`, `Select`, `Button` | `Card` `rounded-3xl`, blok makro `rounded-2xl`, `Badge` skor `rounded-full` (tanpa dot) |
| **F5** Simpan ke Menu Hari Ini | Aksi simpan, pilih waktu makan, umpan balik | `shadcn`: `Button`, `Select` (atau `DropdownMenu`), `Sonner` (toast) | Toast `rounded-2xl shadow-none border` |
| **F6** Ringkasan Harian | Target kalori, makro, daftar tersimpan, hapus | `shadcn`: `Card`, `Progress`, `Separator`, `AlertDialog` (konfirmasi hapus) | `Progress` `rounded-full h-2.5`, daftar item tanpa shadow |
| **F7** Analisis Foto | Lampiran foto, pratinjau, koreksi nama/porsi | `shadcn`: `Button`, `Dialog` (preview/koreksi), `Input`, `Badge` | Thumbnail `rounded-2xl`, tombol foto `rounded-full` |
| **F8** Onboarding & Profil | Isian data tubuh, gender, aktivitas, ubah profil | `shadcn`: `Form`, `Input`, `RadioGroup`, `Select`, `Tabs`, `Button`, `Avatar` | Pilihan aktivitas memakai `RadioGroup` berbentuk kartu `rounded-2xl` |
| **F9** Riwayat Menu | Pilih tanggal, daftar harian, grafik mingguan | `shadcn`: `Calendar` / `Popover` (atau `Accordion`), `Chart` (`recharts`) | Bar/line chart memakai warna `--brand-primary`, sudut bar `radius={8}` |
| **F10** Landing Page | Hero, 3 fitur, 3 langkah, CTA, footer | `shadcn`: `Button`, `Card`, `Separator` + `React Bits`: `FadeContent` / `SplitText` (halus, 1x jalan saat tampil) | Dilarang *glow*, dilarang *eyebrow pill* bertitik, dilarang *boxed icon* |
| **F11 & F12** Bahasa & Tema | Ganti ID/EN & Terang/Gelap di header | `shadcn`: `DropdownMenu` atau `ToggleGroup`, `Button` + `next-themes` | Tombol `rounded-full` di header |
| **F13** Hapus Akun | Modal konfirmasi dengan input kata sandi | `shadcn`: `AlertDialog` (atau `Dialog`), `Input`, `Button` | Modal `rounded-3xl shadow-none`, overlay `bg-black/35` |
| **F14** Kebijakan Privasi | Halaman bacaan panjang 2 bahasa | Tipografi rapi + `shadcn`: `Separator`, `Accordion` (opsional untuk ringkas) | Lebar bacaan `max-w-2xl` |
| **Navigasi Mobile** | Menu bawah / drawer opsi tambahan | `shadcn`: `Sheet` atau `Drawer` (`vaul`) | Sudut atas `rounded-t-3xl` |

---

## 4. Sistem Warna (Light & Dark Mode)

Semua pasangan warna teks dan latar belakang memenuhi standar kontras **WCAG AA** (minimal `4.5:1` untuk teks normal dan `3:1` untuk teks besar/elemen UI). Variabel di bawah dipetakan langsung ke token CSS `shadcn/ui` di `globals.css`.

### 4.1. Permukaan & Netral (*Surfaces & Neutrals*)

| Token Desain | Mapping `shadcn/ui` | Mode Terang (Light) | Mode Gelap (Dark) |
| --- | --- | --- | --- |
| `--bg-canvas` | `--background` | `#FAF8F5` (Warm Alabaster) | `#141715` (Soft Forest Charcoal) |
| `--bg-surface` | `--card` / `--popover` | `#FFFFFF` (Soft Pure White) | `#1B1F1C` (Muted Deep Slate) |
| `--bg-subtle` | `--muted` / `--secondary` | `#F2EFE9` (Warm Oat) | `#232824` (Soft Moss Gray) |
| `--bg-hover` | `--accent` | `#EAE6DE` | `#2B312C` |
| `--border-default` | `--border` / `--input` | `#E6E2D9` | `#2C322D` |
| `--border-strong` | `--ring` (netral) | `#CFC9BC` | `#3E4740` |
| `--text-primary` | `--foreground` | `#1F2622` (Deep Pine Ink) | `#F0EFEA` (Soft Warm White) |
| `--text-secondary` | `--muted-foreground` | `#5A635D` | `#A4ADA6` |
| `--text-muted` | Placeholder & label estimasi | `#848D87` | `#747E77` |

### 4.2. Warna Merek (*Brand Accents* — Adaptasi Lembut dari Logo)

| Token Desain | Mapping `shadcn/ui` | Mode Terang (Light) | Mode Gelap (Dark) |
| --- | --- | --- | --- |
| `--brand-primary` | `--primary` | `#2F7D60` (Soft Botanical Green) | `#5BB58E` (Soft Sage Mint) |
| `--brand-primary-hover` | Hover tombol utama | `#26674F` | `#6EC29E` |
| `--brand-primary-text` | `--primary-foreground` | `#FAF8F5` | `#101A15` |
| `--brand-tint` | Latar sorotan aktif lembut | `#E6F3ED` | `#1E332A` |
| `--brand-accent` | Aksen sekunder hangat (daun oranye logo) | `#D4723C` (Soft Terracotta) | `#E59160` (Warm Apricot) |
| `--brand-accent-tint` | Latar sorotan hangat (budget/catatan) | `#FBEFE8` | `#33231A` |

### 4.3. Semantik Skor Kesehatan (F4) & Status

Digunakan sebagai latar isi datar lembut (*soft flat fill*) + teks kontras tinggi—**tanpa border menyala** dan **tanpa titik indikator**.

| Kategori | Skor / Kondisi | Light (Bg / Text) | Dark (Bg / Text) |
| --- | --- | --- | --- |
| **Baik** | 70 – 100 | `#E6F3ED` / `#1E5C45` | `#1C3329` / `#7ED4AE` |
| **Cukup** | 40 – 69 | `#F9F1DF` / `#7D5614` | `#332B1A` / `#E6BE73` |
| **Rendah** | 0 – 39 | `#F9E8E6` / `#8C322A` | `#36201E` / `#E8928A` |
| **Error / Hapus** | `--destructive` (versi lembut) | `#F9E8E6` / `#9E2F26` | `#36201E` / `#F08A82` |

### 4.4. Aksen Makro (Untuk `Progress` & Label Ringkasan Harian F6)

* **Kalori:** `#2F7D60` (Light) / `#5BB58E` (Dark)
* **Protein:** `#4B7B9C` (Light) / `#7BAED1` (Dark)
* **Karbohidrat:** `#C78A3B` (Light) / `#DFAA62` (Dark)
* **Lemak:** `#C46D5E` (Light) / `#DE8D7F` (Dark)
* **Serat:** `#6A8E58` (Light) / `#93B880` (Dark)

---

## 5. Geometri, Radius Sudut (*Soft Override*), & Spasi

### 5.1. Skala Border-Radius (Otorisasi Resmi Melampaui `rounded-xl`)

Bagian ini secara resmi melonggarkan batas `rounded-xl` dari `SKILL.md` demi mewujudkan karakter antarmuka yang **soft dan ramah**:

| Kelompok Komponen | Kelas Tailwind Wajib | Nilai Piksel | Contoh Penerapan |
| --- | --- | --- | --- |
| **Permukaan Besar & Overlay** | `rounded-3xl` | `24px` | Kartu Hasil Makanan (F4), Kartu Ringkasan Harian (F6), `Card` Auth/Onboarding, `Dialog`/`AlertDialog`, Composer Chat (F1) |
| **Gelembung Chat** | `rounded-3xl` (+ `rounded-br-md` / `rounded-bl-md`) | `24px` (`6px` di sudut ekor) | Pesan pengguna & pesan balasan teks singkat AI |
| **Sub-Kontainer, Input, & Media** | `rounded-2xl` | `16px` | Blok grid makro di dalam kartu, `Input`, `Textarea`, `Select`, pratinjau foto, kotak `Alert` error, Toast `Sonner`, opsi `RadioGroup` |
| **Tombol, Chip, Badge, & Progress** | `rounded-full` | `9999px` | Semua `Button` (CTA, sekunder, ikon), Starter Prompt Chips, `Badge` skor kesehatan (tanpa dot), `Progress` bar, `Tabs` trigger, `ToggleGroup`, `Avatar` |
| **Mobile Bottom Sheet / Drawer** | `rounded-t-3xl` | `24px` (atas) | `Drawer` / `Sheet` bawah di layar HP |

> **Catatan Kepatuhan:** `rounded-full` **dilarang** dipakai sebagai *eyebrow badge* bertitik indikator di atas judul (sesuai `SKILL.md` Bagian 1). Gunakan `rounded-full` hanya untuk elemen interaktif (tombol/chip/tab), indikator progres, avatar, dan badge data (seperti skor kesehatan `82 · Baik`) tanpa titik bulat.

### 5.2. Sistem Spasi & Tata Letak (Mobile-First)

* **Token Dasar shadcn:** Set `--radius: 1rem` (`16px`) di `globals.css` sebagai basis default komponen, lalu terapkan `rounded-3xl` / `rounded-full` sesuai tabel 5.1.
* **Breakpoint:** Mobile-first mulai `360px`, Tablet `768px`, Desktop `1024px`.
* **Padding Halaman:** `16px` (`px-4`) di mobile, `24px` (`px-6`) di desktop.
* **Lebar Maksimal Konten:**
  * Halaman Chat (F1): `max-w-2xl` (`672px`).
  * Ringkasan Harian (F6) & Riwayat (F9): `max-w-3xl` (`768px`).
  * Onboarding & Form Auth (F0, F8): `max-w-md` (`448px`).
  * Landing Page (F10): `max-w-5xl` (`1024px`).

---

## 6. Tipografi

* **Font Utama:** `Plus Jakarta Sans` (Google Fonts) — geometris lembut dengan lengkungan organik yang sangat nyaman dibaca di layar HP.
* **Angka Presisi:** Seluruh nilai gizi (`420 kkal`, `18g`), skor (`82/100`), dan harga (`Rp15.000`) wajib menggunakan kelas Tailwind `tabular-nums`.

| Peran | Mobile / Desktop | Bobot | Line Height | Tracking |
| --- | --- | --- | --- | --- |
| **Display (Landing Hero)** | `28px` / `40px` | `600` (SemiBold) | `1.2` | `-0.02em` |
| **Page Title (H1)** | `22px` / `26px` | `600` (SemiBold) | `1.25` | `-0.015em` |
| **Section Title (H2)** | `18px` / `20px` | `600` (SemiBold) | `1.3` | `-0.01em` |
| **Card Title (H3)** | `16px` | `600` (SemiBold) | `1.35` | `0` |
| **Body Default** | `15px` | `400` (Regular) | `1.6` | `0` |
| **Body Small / Meta** | `13px` | `400` / `500` | `1.5` | `0` |
| **Label Mikro / Eyebrow** | `11px` | `600` (SemiBold) | `1.4` | `+0.06em` (Uppercase, teks polos) |

---

## 7. Spesifikasi Pola Halaman Utama

### 7.1. Header & Navigasi
* Tinggi `64px`, latar `--bg-canvas`, garis bawah `1px solid var(--border-default)`, `shadow-none`.
* **Kiri:** Logo Nouriko (`28px`) + teks `Nouriko` (`font-semibold`).
* **Tengah (Desktop) / Bottom Nav (Mobile):** Tab/tautan `rounded-full` (aktif memakai latar `--brand-tint` + teks `--brand-primary`) untuk **Chat**, **Hari Ini**, dan **Riwayat**.
* **Kanan:** Tombol `rounded-full` ganti bahasa (`ID`/`EN`), tombol ikon `rounded-full` ganti tema, dan `Avatar` (DiceBear gaya Slice) `34px`.

### 7.2. Antarmuka Chat Tunggal (F1)
* **Empty State:** Sapaan singkat hangat (misal: *"Mau makan apa hari ini?"*) diikuti 3–4 tombol `Button` (`variant="outline"`, `rounded-full` atau `rounded-2xl` untuk teks panjang) sebagai contoh prompt yang bisa langsung diketuk.
* **Gelembung & Kartu:** Pesan pengguna berada di kanan (`bg-muted rounded-3xl rounded-br-md`). Balasan AI menampilkan pengantar 1–2 kalimat polos diikuti deretan **Kartu Hasil Makanan (F4)**.
* **Indikator Loading:** Teks status ramah (*"Menyiapkan rekomendasi..."*) dengan transisi halus—**dilarang** memakai `animate-pulse` atau `animate-bounce`.
* **Composer Sticky Bottom:** Wadah `rounded-3xl`, latar `--bg-surface`, `border 1px solid var(--border-default)`. Berisi tombol ikon lampirkan foto (`Button size="icon" variant="ghost" className="rounded-full"`), `Textarea` auto-grow tanpa border internal, dan tombol kirim bulat (`Button size="icon" className="rounded-full"`).

### 7.3. Kartu Hasil Makanan (F4)
Dikomposisikan dari `shadcn` `Card` (`rounded-3xl shadow-none p-5`):
* **Header:** Nama makanan + porsi di kiri; `Badge` skor kesehatan (`rounded-full`, warna semantik Bagian 4.3, tanpa dot) di kanan, misal `82 · Baik`.
* **Meta Harga:** Harga estimasi (`Rp12.000`) sejajar dengan label halus `Estimasi`.
* **Grid Makro:** Wadah `bg-muted rounded-2xl p-3` dengan 5 item (Kalori, Protein, Karbo, Lemak, Serat) berformat `tabular-nums`.
* **Langkah & Bahan Beli (Jika Ada):** Menggunakan `shadcn` `Collapsible` (*"Lihat cara buat"*) agar chat tetap ringkas.
* **Footer Aksi:** Pemilih waktu makan opsional (`Select` `rounded-full`) dan tombol **Simpan** (`Button` `rounded-full`). Setelah disimpan, berubah menjadi **Tersimpan** dengan latar `--brand-tint` dan teks `--brand-primary`.

### 7.4. Ringkasan Harian (F6) & Riwayat (F9)
* Memakai `shadcn` `Card` (`rounded-3xl shadow-none`) untuk progres kalori dan pengeluaran hari ini, dilengkapi `Progress` (`rounded-full`).
* Daftar makanan tersimpan menggunakan pemisah `Separator` tipis dan tombol hapus ikon Lucide polos yang memicu `AlertDialog` (`rounded-3xl shadow-none`).
* Grafik mingguan opsional (F9) memakai `shadcn/ui Charts` (`recharts`) dengan tampilan datar tanpa grid berat.

### 7.5. Onboarding, Auth, & Hapus Akun (F0, F8, F13, F15)
* Seluruh input memakai `shadcn` `Input` (`h-11 rounded-2xl shadow-none`) dengan `Label` di atas input.
* **Onboarding (F8):** Bersih tanpa teks penjelasan privasi; tingkat aktivitas dipilih memakai `RadioGroup` berbentuk kartu lembut `rounded-2xl`.
* **Hapus Akun (F13):** Menggunakan `AlertDialog` (`rounded-3xl shadow-none`) dengan peringatan singkat tidak bisa dibatalkan, input kata sandi, serta tombol **Batal** dan **Hapus** (`rounded-full`).

### 7.6. Landing Page (F10)
* Hero yang lega dengan judul hangat, subjudul 1–2 kalimat, dan tombol CTA `rounded-full`. Dapat menggunakan animasi masuk satu kali yang sangat halus dari **React Bits** (`FadeContent`), tanpa *glow* dan tanpa *eyebrow pill* bertitik.
* Bagian 3 kemampuan utama & 3 langkah cara kerja memakai `shadcn` `Card` (`rounded-3xl shadow-none`) dengan angka tipografi polos (`01`, `02`, `03`)—tanpa *boxed icon*.

---

## 8. Gerak, Aksesibilitas, & Dua Bahasa (ID / EN)

* **Gerak Lembut (*Soft Motion*):** Transisi warna `transition-colors duration-150 ease-out`, buka-tutup `Collapsible`/`Dialog` `200ms`. Jika memakai komponen **React Bits**, pastikan hanya animasi masuk satu kali (*trigger once*), berdurasi singkat (`300ms–450ms`), dan otomatis nonaktif saat `prefers-reduced-motion: reduce`.
* **Target Sentuh:** Minimal `44×44px` pada seluruh tombol, chip, dan input mobile.
* **Fokus Keyboard:** Status `:focus-visible` yang jelas (`ring-2 ring-[var(--brand-primary)] ring-offset-2`).
* **Pesan Error (Sesuai PRD):** Menggunakan `shadcn` `Alert` (`rounded-2xl shadow-none border-0 bg-[var(--error-bg)] text-[var(--error-text)]`), bahasa ramah sehari-hari tanpa istilah teknis, dan `role="alert"`.
* **Fleksibilitas Teks (ID ↔ EN):** Hindari `width` tetap pada `Button` dan `Badge` agar peralihan bahasa Indonesia dan Inggris tidak terpotong.