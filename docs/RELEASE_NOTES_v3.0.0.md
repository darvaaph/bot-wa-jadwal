# 📦 Catatan Rilis Resmi (Release Notes) — Bot Jadwal v3.0.0
# Tanggal Rilis: 29 September 2026

Selamat datang di **Bot Jadwal v3.0.0**! Rilis ini merupakan tonggak pembaruan arsitektur terbesar yang menggabungkan asisten perkuliahan cerdas WhatsApp Multi-Device dengan Web Admin Dashboard responsif ke dalam satu biner Go mandiri (*single binary embedded*).

---

## 🌟 Ringkasan Eksekutif

Bot Jadwal v3.0.0 beralih dari bot chat skrip monolitik menjadi platform manajemen akademik multi-kelas modern dengan dukungan:
- **Arsitektur Decoupled & REST API v1:** Standar API terbuka di bawah rute `/api/v1/` dengan amplop terstruktur `{ "status": "success", "data": ... }`.
- **Manajemen Akses Berbasis Peran (RBAC):** Pemisahan wewenang ketat untuk *System Administrator*, *Ketua Murid (KM)*, dan *Penanggung Jawab Matkul (PJ)*.
- **Web Admin Dashboard Responsif:** UI bertema *Academic Dark Mode* menggunakan Tailwind CSS & Alpine.js via CDN tanpa memerlukan Node.js atau proses build frontend tambahan.
- **Portal Publik Mahasiswa:** Akses baca jadwal dan tugas tanpa akun melalui URL kanonis atau proteksi kode sesi 8-digit.
- **Ketahanan Operasional & Keamanan Standar Industri:** Rate limiting terpusat berbasis database SQLite, perlindungan CSRF, header keamanan HSTS/CSP, dan fail-closed startup validation.

---

## 🚀 Fitur Baru & Peningkatan Utama

### 1. 👥 Manajemen Akses, Akun, & Portal Mahasiswa
- **Autentikasi Aman:** Dukungan token Bearer dan cookie HTTP-only `bv1` bertanda `Secure; SameSite=Lax`.
- **Multi-Tenant Class Isolation:** Isolasi data ketat antar-kelas perkuliahan. KM dan PJ hanya dapat mengelola data kelas yang ditugaskan kepada mereka.
- **Sistem Undangan & Pergantian Konteks:** Alur undangan pengurus kelas via token sekali pakai dan pertukaran konteks tugas aktif (`/api/v1/auth/switch-context`).
- **Portal Mahasiswa Mandiri (`/portal/:slug`):** Mahasiswa dapat melihat jadwal hari ini, mata kuliah berlangsung (*now/next event*), daftar tugas aktif, dan materi perkuliahan tanpa login.

### 2. 📅 Mesin Jadwal & Kalender Akademik (Schedule Engine)
- **Dukungan 19 Kelas Perkuliahan:** Membaca berkas kurikulum master JSON per kelas di `data/jadwal/*.json`.
- **Penyesuaian Jadwal Fleksibel (Overrides):** Akomodasi pergeseran jam kuliah (*reschedule*), kelas pengganti (*replacement*), kelas ditiadakan (*cancelled*), dan penetapan hari libur nasional.
- **Pendeteksi Bentrok Jadwal Otomatis:** Memvalidasi tumpang tindih waktu perkuliahan sebelum perubahan disimpan.

### 3. 📝 Manajemen Tugas & Pelacak Tenggat Waktu (Task Tracker)
- **Lifecycle Tugas Lengkap:** Alur status tugas terkelola (`DRAFT` ➔ `PUBLISHED` ➔ `REVIEW` ➔ `COMPLETED` ➔ `ARCHIVED`).
- **Optimistic Locking (HTTP 409):** Menggunakan field integer `version` untuk mencegah penimpaan data tanpa sengaja ketika dua pengurus mengedit tugas bersamaan.
- **Tautan Materi Kuliah:** Kemampuan melampirkan link materi (Google Drive, slide, dokumen PDF) langsung pada kartu tugas.

### 4. 🤖 Integrasi WhatsApp Multi-Device Cerdas
- **Library whatsmeow Terbaru:** Dukungan protokol multi-device WhatsApp yang stabil.
- **Watchdog Auto-Reconnect:** Goroutine pengawas di latar belakang yang otomatis menyambungkan kembali koneksi bot jika koneksi internet terputus menggunakan algoritma *Exponential Backoff*.
- **Pemisahan File Sesi Terisolasi:** File sesi WhatsApp disimpan mandiri di `storage/sesi_bot.db`, terpisah total dari database aplikasi.
- **Broadcast Pengingat Pagi (06:00 WIB):** Otomatis menyiarkan ringkasan jadwal kuliah dan tugas mendesak ke grup WhatsApp kelas setiap hari kuliah.

### 5. 🔐 Security Hardening Produksi (BE-012, BE-013, BE-014)
- **Persistent Rate Limiter Terpusat (ADR-0009):** 8 kebijakan pembatas laju terdaftar di SQLite (`security_attempts`) untuk menangkal brute force login, eksploitasi portal code, dan DoS mutasi data.
- **Fail-Closed Startup Validation:** Aplikasi menolak booting di lingkungan produksi jika variabel rahasia (`BOT_JADWAL_AUTH_HASH_KEY`, `BOT_JADWAL_SECURE_COOKIES`, dll) belum tervalidasi aman.
- **Proteksi Anti-CSRF (ADR-0010):** Validasi header `X-Requested-With` dan kecocokan exact-origin pada mutasi yang menggunakan autentikasi cookie.
- **HTTP Security Headers Lengkap:** Injeksi otomatis HSTS (`max-age=31536000`), CSP ketat tanpa wildcard, `X-Frame-Options: DENY`, dan `Cache-Control: no-store` pada seluruh endpoint API v1.

---

## 🗄️ Perubahan Model Data & Migrasi Database

- **Database Utama v3.0 (`storage/bot_v1.db`):** Menggunakan SQLite dalam mode WAL (*Write-Ahead Logging*) dengan skema terindeks penuh untuk tabel:
  - `users`, `roles`, `role_assignments`, `sessions`, `invitations`
  - `academic_semesters`, `course_offerings`, `schedule_patterns`, `schedule_events`
  - `academic_tasks`, `task_materials`, `audit_logs`
  - `security_attempts` (tabel persistent rate limiting)
- **Kompatibilitas Database Legacy (`storage/tugas.db`):** Tetap dipertahankan untuk backward-compatibility fitur chat bot WhatsApp.

---

## ⚠️ Kebijakan Deprecations & Kompatibilitas Mundur

- **Legacy REST Endpoints (ADR-0002):** 7 endpoint REST lama (`/api/tasks`, `/api/schedule`, `/api/classes`, dll.) tetap didukung sebagai shim kompatibilitas dengan menyertakan response header:
  ```http
  Deprecation: true
  Sunset: 2026-12-31
  Link: </api/v1/...>; rel="successor-version"
  ```
- **Entry Point Eksekusi:** Seluruh instruksi kompilasi dan eksekusi resmi kini menggunakan `./cmd/bot` (menggantikan instruksi usang `go run .`).

---

## 📋 Matriks Kompatibilitas Sistem

| Komponen | Persyaratan Minimum | Rekomendasi Produksi |
|:---|:---|:---|
| **Go Runtime** | Go 1.22+ | Go 1.22 atau Go 1.23 |
| **Sistem Operasi** | Linux x64 (Ubuntu 22.04 LTS) | Ubuntu Server 24.04 LTS (Azure VM) |
| **Arsitektur CPU** | amd64 (x86_64) | 2 vCPU AMD EPYC / Intel Xeon |
| **Memori RAM** | 512 MB | 1 GiB atau lebih |
| **Basis Data** | SQLite 3.35+ (Built-in WAL mode) | SQLite 3.40+ |
| **Browser Pengguna** | Chrome 90+, Firefox 90+, Safari 14+ | Browser modern versi terbaru |
| **Layar Mobile** | Layar ponsel lebar minimal 360px | Viewport 390px (standar iPhone/Android) |

---

## 👥 Kontributor & Tim Rilis v3.0

- **Project Lead & Tech Lead:** Darva Aryasatya Putra Hermawan
- **Backend Architecture & Security:** Tim Backend Bot Jadwal
- **Frontend & Web Dashboard:** Tim Frontend Web UI
- **Quality Assurance & Verification:** Tim QA & Software Testing
- **Operasional & DevOps:** Tim Cloud Infrastructure Azure
