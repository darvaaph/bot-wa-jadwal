# Panduan Onboarding & Handoff Tim Pengembang (v2.0)
# Proyek: Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard

Dokumen ini adalah **panduan handoff resmi** yang disusun untuk Project Manager (PM) sebagai bahan presentasi dan orientasi (*kickoff meeting*) bagi seluruh anggota tim pengembang baru (**UI/UX Designer**, **Frontend Developer**, **Backend Developer**, dan **QA / Software Tester**).

---

## Daftar Isi
1. [Visi & Latar Belakang Proyek](#1-visi--latar-belakang-proyek)
2. [Kondisi Sistem Saat Ini (Current State)](#2-kondisi-sistem-saat-ini-current-state)
3. [Target Produk Versi 2.0 (Target State)](#3-target-produk-versi-20-target-state)
4. [Struktur Tim & Pembagian Peran (Buddy System)](#4-struktur-tim--pembagian-peran-buddy-system)
5. [Lingkungan Pengembangan Lokal (Local Dev Setup)](#5-lingkungan-pengembangan-lokal-local-dev-setup)
6. [Alur Kerja Rekayasa (Git Flow & Standar Vibe Coding)](#6-alur-kerja-rekayasa-git-flow--standar-vibe-coding)
7. [Roadmap 6 Minggu & Target Milestone](#7-roadmap-6-minggu--target-milestone)
8. [Panduan Berbicara PM Saat Kickoff Meeting (15 Menit)](#8-panduan-berbicara-pm-saat-kickoff-meeting-15-menit)

---

## 1. Visi & Latar Belakang Proyek

### Mengapa Proyek Ini Ada?
Aplikasi ini bermula sebagai bot WhatsApp mandiri untuk membantu koordinasi jadwal perkuliahan, catatan tugas, dan tautan kuliah di kelas. Saat ini bot telah melayani kebutuhan kelas secara aktif setiap hari.

### Mengapa Kita Naik ke Versi 2.0?
Interaksi berbasis teks WhatsApp memiliki keterbatasan:
1. **Pencatatan Tugas Rawan Typo:** Format perintah panjang seperti `!tugas tambah ...` rentan salah ketik karakter pemisah.
2. **Visibilitas Status Bot Terbatas:** Jika bot membutuhkan pemindaian ulang QR Code, administrator harus membuka terminal server Linux (SSH).
3. **Kebutuhan Antarmuka Visual:** Ketua Tingkat (Komti) dan pengurus kelas membutuhkan cara cepat (< 30 detik) mencatat tugas dosen dan menggeser jam kuliah langsung dari browser ponsel pintar (*smartphone*) saat berada di kampus.

**Tujuan v2.0:** Membangun **Web Admin Dashboard responsif** yang disematkan langsung ke dalam biner bot, didukung REST API terstruktur, serta dikelola secara tim dengan standar industri perangkat lunak.

---

## 2. Kondisi Sistem Saat Ini (Current State)

Fondasi inti sistem telah selesai dibangun pada Fase A dan beroperasi dengan status stabil:

* **Bahasa Pemrograman:** Go (Golang) v1.22+ dengan struktur standar modular di direktori `internal/` dan satu titik masuk utama di `cmd/bot/main.go`.
* **Basis Data:** SQLite pool terpadu dengan mode WAL (*Write-Ahead Logging*) di `storage/tugas.db`.
* **Klien WhatsApp:** Library `whatsmeow` dengan sesi login terisolasi di `storage/sesi_bot.db`. Pemblokiran nomor atau penggantian akun tidak akan menghapus data tugas maupun jadwal kelas.
* **Mesin Multi-Kelas:** Mampu menangani hingga 19 kurikulum kelas perkuliahan melalui berkas master JSON di `data/jadwal/*.json`.
* **Server Produksi Live:** Berjalan 24/7 pada server Microsoft Azure Linux VM yang dikelola oleh service manager `systemd`.
* **Integritas Pengujian:** Seluruh paket internal dilengkapi unit test otomatis dengan tingkat kelulusan 100% (`go test ./...` berstatus PASS).

---

## 3. Target Produk Versi 2.0 (Target State)

Pada Versi 2.0, tim akan melengkapi sistem dengan lapisan antarmuka visual dan jembatan data:

```text
                                  PENGGUNA
                                     │
          ┌──────────────────────────┴──────────────────────────┐
          ▼                                                     ▼
    CHAT WHATSAPP                                        WEB DASHBOARD
  (Mahasiswa & Dosen)                                  (Komti & Pengurus)
  • Pengingat Pagi 06:00                               • Buka di HP / Laptop
  • Query !jadwal / !tugas                             • Form Pop-up Tambah Tugas
  • Akses !link kuliah                                 • Geser Jam Kuliah Visual
          │                                            • Monitor Bot & Scan QR Web
          │                                                   │
          ▼                                                   ▼
  [ whatsmeow Client ]                                 [ REST API (/api/...) ]
          │                                                   │
          └──────────────────────────┬────────────────────────┘
                                     ▼
                             CORE ENGINE (GO)
                                     │
                                     ▼
                         SQLite (storage/tugas.db)
```

### Komponen Utama yang Dibangun:
1. **Web Admin Dashboard (`web/`):**
   * Antarmuka tunggal (*Single Page Application*) bernuansa *Academic Dark Mode*.
   * Dibuat menggunakan **HTML5, Tailwind CSS, dan Alpine.js**.
   * Wajib **Mobile-First / Responsive** (nyaman dioperasikan dengan satu jempol di layar HP).
2. **REST API Gateway (`internal/api/`):**
   * Endpoint CRUD Catatan Tugas: `GET /api/tasks`, `POST /api/tasks`, `DELETE /api/tasks/{id}`.
   * Endpoint Jadwal & Kelas: `GET /api/classes`, `GET /api/schedule`.
   * Endpoint Pemantauan Gateway: `GET /api/status`, `GET /api/health`.
3. **Sistem Embedded Tanpa Node.js:**
   * Seluruh aset web statis disatukan ke dalam biner Go menggunakan `embed.FS` (`web/embed.go`).
   * Server di Azure tidak memerlukan instalasi Node.js, npm, maupun Nginx tambahan.

---

## 4. Struktur Tim & Pembagian Peran (Buddy System)

Tim pengembang terdiri dari 8 orang yang dipasangkan berdua (*Buddy System*) untuk memastikan setiap peran memiliki rekan diskusi dan tidak ada hambatan kerja tunggal (*single point of failure*):

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                      PROJECT MANAGER / TECH LEAD                        │
│                   (Koordinasi Sprint, PR Review, DevOps)                │
└──────┬────────────────────┬────────────────────┬────────────────────┬───┘
       │                    │                    │                    │
       ▼                    ▼                    ▼                    ▼
     UI/UX               FRONTEND             BACKEND            QA / TESTER
   (2 Orang)            (2 Orang)            (2 Orang)            (2 Orang)
   • Lead Layout        • Layout Shell       • Tasks API          • Bot Scenarios
   • Component Forms    • Form Modals        • Schedule API       • Web UI Edge Cases
```

### Rincian Tanggung Jawab:
* **UI/UX Designer (2 Orang):**
  * *Orang 1:* Merancang Design System Tokens, Layout Utama Desktop (1440px), dan Mobile (390px) di Figma.
  * *Orang 2:* Merancang form modal interaktif, komponen notifikasi toast, dan variasi *empty/error states*.
* **Frontend Developer (2 Orang):**
  * *Orang 1:* Mengembangkan kerangka navigasi tab di `web/index.html` dan kartu jadwal kuliah mingguan.
  * *Orang 2:* Mengembangkan komponen kartu tugas, modal pop-up tambah tugas, dan konsumsi API via `web/js/api.js`.
* **Backend Developer (2 Orang):**
  * *Orang 1:* Mengembangkan endpoint CRUD tugas di file terpisah `internal/api/tasks_handler.go`.
  * *Orang 2:* Mengembangkan endpoint kurikulum dan jadwal di file terpisah `internal/api/schedule_handler.go`.
* **QA / Software Tester (2 Orang):**
  * *Orang 1:* Menyusun matriks pengujian fitur bot WhatsApp dan mengeksekusi uji skenario ekstrem di grup.
  * *Orang 2:* Menyusun matriks pengujian Web Dashboard pada berbagai ukuran layar HP dan melaporkan bug di GitHub Issues.

---

## 5. Lingkungan Pengembangan Lokal (Local Dev Setup)

Untuk menjaga keamanan server produksi di Azure, tim wajib mematuhi panduan eksekusi lokal:

### Prasyarat:
* Go versi 1.22+ ([Unduh di golang.org](https://golang.org))
* Git terpasang pada komputer masing-masing.

### Perintah Menjalankan Aplikasi:

| Kebutuhan Pengujian | Perintah Eksekusi | Keterangan Keamanan |
| :--- | :--- | :--- |
| **Frontend & UI/UX** | `go run ./cmd/bot -web-only` | **Sangat Direkomendasikan.** Dashboard aktif di `http://localhost:8080` tanpa menyambung ke WhatsApp. Server Azure aman 100%. |
| **Backend & QA (Testing WA)**| `go run ./cmd/bot -session storage/sesi_dev.db` | Menggunakan file sesi terpisah. Scan QR dengan nomor cadangan. Server Azure tidak terganggu. |
| **Server Produksi Azure** | `go run ./cmd/bot` *(atau service systemd)* | Menjalankan bot WhatsApp utama kelas dan server web secara live. |

---

## 6. Alur Kerja Rekayasa (Git Flow & Standar Vibe Coding)

### Aturan Emas Git:
1. **Dilarang keras push langsung ke branch `main`!**
2. Setiap pekerjaan wajib dibuatkan branch baru dari `main`:
   * Fitur baru: `feat/nama-fitur` (contoh: `feat/tasks-api-crud`, `feat/dashboard-navbar`).
   * Perbaikan bug: `fix/nama-bug` (contoh: `fix/mobile-table-overflow`).
3. Seluruh merge wajib melalui **Pull Request (PR)** dengan minimal 1 persetujuan (*approval*) dari Project Manager.
4. Gunakan format pesan commit standar (*Conventional Commits*):  
   `feat: ...`, `fix: ...`, `docs: ...`, `refactor: ...`, `test: ...`.

### Standar Kerja dengan Bantuan AI (Vibe Coding Guidelines):
Jika anggota tim menggunakan alat bantu AI (Cursor, Copilot, ChatGPT, Claude):
* **Gunakan Scope Prompting:** Berikan instruksi yang sempit dan sebutkan nama file secara spesifik. Dilarang meminta AI merombak seluruh proyek sekaligus.
* **Verifikasi Unit Test:** Sebelum membuka PR, developer wajib menjalankan perintah `go test ./...` di terminal. Jangan ajukan PR jika pengujian berstatus merah (*FAIL*).
* **Pemisahan Berkas:** Developer yang bekerja berpasangan dilarang mengedit berkas yang sama untuk menghindari konflik kode (*merge conflict*).

---

## 7. Roadmap 6 Minggu & Target Milestone

Proyek diselesaikan dalam 3 siklus Sprint (masing-masing 2 minggu) dengan alokasi waktu fleksibel 3–5 jam per minggu:

```text
[ Minggu 1 - 2 ] ───────────▶ [ Minggu 3 - 4 ] ───────────▶ [ Minggu 5 - 6 ] ───────────▶ RILIS v2.0
    SPRINT 1                      SPRINT 2                      SPRINT 3
(Fondasi & Desain)            (Integrasi & API)            (Hardening & Deploy)
```

### Milestone 1: Sprint 1 — Fondasi Sistem & Desain (Minggu 1 – 2)
* UI/UX menyelesaikan desain Figma resolusi Desktop (1440px) dan Mobile (390px).
* Backend menyelesaikan endpoint `/api/tasks` dan `/api/schedule` dengan unit test 100% lulus.
* Frontend menyelesaikan kerangka responsif dan komponen statis di `web/`.
* QA menyelesaikan dokumen skenario pengujian (*test matrix*).

### Milestone 2: Sprint 2 — Integrasi Sistem & Gateway WA (Minggu 3 – 4)
* Frontend tersambung penuh dengan REST API Backend nyata.
* Fitur tambah dan hapus tugas dari web tersimpan langsung ke SQLite.
* Panel pemantauan WhatsApp Gateway di web mulai membaca status koneksi.
* QA mengeksekusi uji fungsional dan mencatat seluruh temuan bug di GitHub Issues.

### Milestone 3: Sprint 3 — Deployment Azure & Peluncuran Resmi (Minggu 5 – 6)
* Penyelesaian seluruh catatan bug (Zero Critical Bugs).
* Biner tunggal Go di-deploy ke server Linux Azure VM via `systemd`.
* Pengujian penerimaan akhir (*User Acceptance Test*) oleh perwakilan kelas.
* Peluncuran resmi dan sosialisasi penggunaan kepada seluruh mahasiswa.

---

## 8. Panduan Berbicara PM Saat Kickoff Meeting (15 Menit)

Gunakan poin-poin berikut saat memimpin pertemuan perdana (*Kickoff Meeting*) bersama tim:

1. **Menit 00–03 (Pembuka & Visi):**  
   *"Halo teman-teman, selamat datang di tim pengembang bot-jadwal v2.0. Proyek ini kita buat untuk mempermudah kelas kita sendiri. Kita akan upgrade bot chat teks menjadi Web Admin Dashboard yang modern dan bisa dibuka dari HP."*
2. **Menit 03–07 (Demo Sistem yang Ada & Arsitektur):**  
   *Tunjukkan bot yang sedang berjalan di WhatsApp dan perlihatkan folder `web/index.html` via browser.*  
   *"Secara mesin, backend Go dan SQLite kita sudah stabil dan berjalan di server Azure. Tugas kita bersama adalah melengkapi antarmuka web dan API-nya."*
3. **Menit 07–11 (Penjelasan Peran & Alur Kerja):**  
   *"Kita bekerja berpasangan (2 orang per peran). Teman-teman tidak perlu khawatir kodingannya bentrok, karena wilayah filenya sudah kita bagi terpisah. Kita menggunakan GitHub Projects untuk memantau kartu tugas."*
4. **Menit 11–13 (Komitmen & Suasana Belajar):**  
   *"Proyek ini adalah ruang belajar bersama. Komitmen kita santai sekitar 3–5 jam per minggu. Kalau ada error, blocker, atau bingung pakai AI/Git, langsung tanyakan di grup tim. Jangan dipendam sendiri."*
5. **Menit 13–15 (Langkah Pertama / Next Actions):**  
   *"Setelah meeting ini, silakan cek undangan kolaborator di GitHub masing-masing, baca berkas `docs/CONTRIBUTING.md`, dan ambil tiket pertama kalian di kolom Ready pada GitHub Projects. Mari kita bangun portofolio hebat bersama!"*
