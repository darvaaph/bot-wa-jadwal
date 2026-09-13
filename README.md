# 🤖 Bot WhatsApp Jadwal Kuliah & Web Admin Dashboard v2.0

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Frontend](https://img.shields.io/badge/Frontend-Tailwind_CDN_%2B_Alpine.js-38B2AC?style=flat&logo=tailwindcss)](https://tailwindcss.com)
[![Database](https://img.shields.io/badge/Database-SQLite_WAL-003B57?style=flat&logo=sqlite)](https://sqlite.org)
[![Library](https://img.shields.io/badge/WhatsApp-whatsmeow-25D366?style=flat&logo=whatsapp)](https://github.com/tulir/whatsmeow)
[![Server](https://img.shields.io/badge/Cloud-Microsoft_Azure-0078D4?style=flat&logo=microsoftazure)](https://azure.microsoft.com)

Asisten perkuliahan cerdas berbasis **Bot WhatsApp Multi-Device** dan **Web Admin Dashboard Responsif** yang disematkan langsung ke dalam satu biner Go (*single binary embedded* via `embed.FS`). 

Sistem ini membantu mahasiswa dan pengurus kelas memantau jadwal perkuliahan harian, melacak tenggat waktu tugas (*deadline tracker*), mengelola tautan kuliah daring, mengakomodasi jadwal kuliah pengganti (*schedule overrides*), serta mengirimkan broadcast pengingat pagi otomatis setiap hari kuliah pukul **06:00 WIB**.

---

## 📚 Pusat Navigasi Dokumentasi Tim (`docs/` & `.agent/`)

Seluruh acuan kerja, arsitektur, dan panduan operasional tim telah didokumentasikan secara rapi dan modular:

| Dokumen | Deskripsi & Target Pembaca |
| :--- | :--- |
| 🚀 [**docs/TEAM_ONBOARDING.md**](docs/TEAM_ONBOARDING.md) | **Panduan Onboarding & Kickoff Meeting Tim** *(Wajib untuk seluruh tim baru: visi v2.0, roadmap sprint, dan panduan PM).* |
| 📱 [**docs/DASHBOARD_PRD.md**](docs/DASHBOARD_PRD.md) | **Spesifikasi Produk Web Dashboard** *(Wajib untuk UI/UX & Frontend: gambaran 4 tab, data field, tombol aksi, dan variasi state).* |
| 🤝 [**docs/CONTRIBUTING.md**](docs/CONTRIBUTING.md) | **Panduan Standar Kontribusi & Git Flow** *(Aturan branching, Conventional Commits, dan ceklis Pull Request).* |
| 🏗️ [**docs/ARCHITECTURE.md**](docs/ARCHITECTURE.md) | **Cetak Biru Arsitektur Teknis** *(Diagram alur data, SQLite WAL mode, multi-kelas engine, dan isolasi sesi).* |
| ☁️ [**docs/DEPLOYMENT.md**](docs/DEPLOYMENT.md) | **Panduan Server Azure & DevOps** *(Operasional server Linux Azure VM, service systemd, dan rilis produksi).* |
| 📖 [**docs/PANDUAN_PENGGUNAAN.md**](docs/PANDUAN_PENGGUNAAN.md) | **Panduan Lengkap Perintah Bot** *(Cheat sheet perintah WhatsApp, format tanggal alami, dan otorisasi admin grup).* |
| 🤖 [**.agent/skills/vibe-coding-guide/**](.agent/skills/vibe-coding-guide/SKILL.md) | **Guardrails Vibe Coding AI** *(Aturan anti-bloat: dilarang npm, Go single binary, dan protokol tutor edukasi).* |
| 🌿 [**.agent/skills/git-team-flow/**](.agent/skills/git-team-flow/SKILL.md) | **Panduan Kolaborasi Git Tim** *(Penamaan branch per tiket, etika commit, dan larangan auto-commit liar oleh AI).* |

---

## 👥 Alur Kerja Tim & Metodologi (Scrumban)

Proyek ini dikembangkan secara kolaboratif oleh 8 pengembang mahasiswa menggunakan metodologi **Scrumban (Hybrid Agile-Kanban)** melalui **GitHub Projects** dan **Milestones**.

```text
┌────────────────────────────────────────────────────────────────────────┐
│                      PROJECT MANAGER / TECH LEAD                       │
│             (Koordinasi Sprint, Pull Request Review, DevOps)           │
└──────┬────────────────────┬────────────────────┬───────────────────┬───┘
       │                    │                    │                   │
       ▼                    ▼                    ▼                   ▼
    🎨 UI/UX             💻 FRONTEND          ⚙️ BACKEND          🧪 QA TESTER
    (2 Orang)            (2 Orang)            (2 Orang)           (2 Orang)
```

### 1. Papan Kanban GitHub Projects
Seluruh aktivitas tugas dipantau melalui kartu digital (tiket) di papan Kanban:
- `Todo`: Tiket yang siap dikerjakan pada sprint aktif.
- `In Progress`: Tiket yang sedang dikerjakan secara aktif oleh penanggung jawab (*Assignee*).
- `In Review`: Kodingan atau desain telah selesai dan sedang diperiksa oleh Project Manager.
- `Done`: Pull Request berhasil di-merge ke branch `main` atau desain/dokumen telah disetujui.

### 2. Tiga Siklus Sprint (Roadmap 6 Minggu)
1. **Sprint 1 (Minggu 1 – 2): Fondasi & Desain Inti**
   - *UI/UX:* Moodboard, Design Tokens, Wireframe, dan Mockup Hi-Fi Halaman Ringkasan & Jadwal di Figma.
   - *Backend:* Endpoint REST API `/api/tasks` dan `/api/schedule` di Go + unit test 100% lulus.
   - *Frontend:* Shell dashboard responsif, navigasi tab Alpine.js, dan tabel jadwal (data mock).
   - *QA Tester:* Perancangan dokumen Master Test Cases di Google Spreadsheet (Tab Bot & Tab Web).
2. **Sprint 2 (Minggu 3 – 4): Integrasi Sistem & WhatsApp Gateway**
   - *Frontend & Backend:* Menyambungkan tombol-tombol web ke REST API Go nyata (tersimpan di SQLite).
   - *UI/UX & Frontend:* Antarmuka telemetri status bot dan pemindai QR Code di dashboard.
   - *QA Tester:* Eksekusi uji fungsional di sandbox WhatsApp dan pengetesan form di HP fisik.
3. **Sprint 3 (Minggu 5 – 6): UAT, Hardening, & Deployment Azure**
   - *Seluruh Tim:* Penyelesaian seluruh catatan bug (Zero Blocker Bugs).
   - *QA & PM:* End-to-End User Acceptance Test (UAT) dan simulasi disaster recovery.
   - *DevOps / PM:* Deployment biner Go ke server Linux Azure VM via `systemd` dan rilis resmi ke kelas.

---

## 💻 Panduan Menjalankan Aplikasi di Komputer Lokal

Untuk menjaga keamanan server produksi di Azure, **gunakan perintah eksekusi yang sesuai dengan peran Anda**:

```text
                  ┌─────────────────────────────────────────────────────────┐
                  │                ALUR PENGEMBANGAN LOKAL                  │
                  └─────────────────────────────────────────────────────────┘
                                               │
                        ┌──────────────────────┴──────────────────────┐
                        ▼                                             ▼
             [OPSI 1: Mode Web-Only]                      [OPSI 2: Mode Sesi-Dev]
           Untuk: Frontend & UI/UX                      Untuk: Backend & QA Tester
      ---------------------------------            ------------------------------------
      go run ./cmd/bot -web-only                   go run ./cmd/bot -session storage/sesi_dev.db
      ---------------------------------            ------------------------------------
      • Web aktif di localhost:8080                • Web aktif di localhost:8080
      • Database lokal tugas.db aktif              • Database lokal tugas.db aktif
      • WhatsApp MATI TOTAL                        • WhatsApp aktif pakai nomor testing pribadi
      • Bebas scan QR, sangat cepat                • Muncul QR code dev di terminal
```

### 1. Opsi 1: `go run ./cmd/bot -web-only` *(Khusus Frontend & UI/UX)*
- Menjalankan **Web Dashboard & REST API lokal** di `http://localhost:8080`.
- **Fitur WhatsApp dimatikan total** (tidak ada scan QR code).
- Server Azure produksi aman 100% dan laptop tidak memerlukan koneksi WhatsApp.

### 2. Opsi 2: `go run ./cmd/bot -session storage/sesi_dev.db` *(Khusus Backend & QA)*
- Menjalankan bot WhatsApp dengan **file sesi dev terpisah** (`storage/sesi_dev.db`).
- Terminal akan memunculkan QR Code untuk di-scan menggunakan **nomor WhatsApp testing cadangan** di grup pengujian sandbox.
- **Dilarang keras menyentuh `storage/sesi_bot.db`** agar tidak berebut koneksi dengan server Azure!

### 3. Menjalankan Pengujian Unit (Wajib Sebelum Pull Request)
Sebelum membuka PR, developer wajib memastikan seluruh unit test lulus:
```bash
go test -v ./...
```

---

## 🛠️ Toolkit Resmi Tim (100% Gratis & Ringan)

| Divisi | Toolkit Utama | Fungsi di Proyek Ini |
| :--- | :--- | :--- |
| 🎨 **UI/UX Designer** | **Figma** + Plugin *Heroicons/Lucide* & *Contrast Checker* | Merancang wireframe, mockup Hi-Fi, dan design tokens. |
| 💻 **Frontend Dev** | **Code Editor** + **Chrome DevTools (F12)** + **Figma** | Mengoding di `web/`, cek responsive 390px mobile, dan inspeksi token desain. |
| ⚙️ **Backend Dev** | **Go Toolchain** + **Insomnia** + **DB Browser for SQLite** | Membuat endpoint API, uji request via Insomnia, dan verifikasi tabel database. |
| 🧪 **QA Tester** | **Google Spreadsheet** + **DB Browser** + **WhatsApp Sandbox** | Menyusun master test cases (Tab 1 & 2), verifikasi database, dan uji skenario. |
| 👑 **Project Manager** | **GitHub Projects** + **Terminal SSH Azure** | Manajemen papan Kanban, code review PR, dan deployment rilis. |

---

## 🛡️ Aturan Emas Arsitektur (Hard Guardrails)

Seluruh pengembang wajib mematuhi batasan teknis berikut:

1. **Zero-Build Frontend (Dilarang Menginstal NPM / Node.js):**
   - JANGAN PERNAH menjalankan `npm init`, `npm install`, atau memasukkan `package.json` dan `node_modules`.
   - Stack frontend murni **HTML5 + Tailwind CSS (CDN) + Alpine.js (CDN)** di folder `web/` agar tersemat langsung ke biner Go via `embed.FS`.
2. **Modular Go Backend:**
   - Gunakan perutean standar Go v1.22+ (`http.ServeMux`).
   - Pisahkan handler API di `internal/api/`: `tasks_handler.go`, `schedule_handler.go`, `bot_handler.go`.
3. **Keamanan Database SQLite:**
   - Database aplikasi berada di `storage/tugas.db` dengan mode WAL (`PRAGMA journal_mode=WAL`).
   - **Wajib gunakan Prepared Statement (`?`)** pada setiap query SQL untuk mencegah celah **SQL Injection**.
4. **Etika Git & Anti-Konflik:**
   - **Dilarang push langsung ke branch `main`!**
   - Buat branch fitur per tiket: `feat/<kode-tiket>-<deskripsi>` (contoh: `feat/be-1-tasks-api`).
   - Gunakan format pesan commit standar (*Conventional Commits*): `feat: ...`, `fix: ...`, `docs: ...`.
   - AI Agent dilarang melakukan auto-commit mandiri; seluruh commit dijalankan sendiri oleh developer.

---

## 📂 Struktur Direktori Proyek

```text
bot-jadwal/
├── cmd/
│   └── bot/
│       └── main.go          # Titik masuk utama aplikasi (CLI flags, bot WA, & REST API server)
├── web/                     # Antarmuka Web Admin Dashboard (Single Binary Embedded)
│   ├── embed.go             # Go embed.FS pengemas aset web ke dalam biner tunggal
│   ├── index.html           # Halaman utama SPA Dashboard
│   ├── css/style.css        # Styling custom pendukung
│   └── js/                  # api.js (REST client & mock) & app.js (state reaktif Alpine.js)
├── internal/
│   ├── config/              # Konfigurasi aplikasi & manajemen path storage
│   ├── database/            # Inisialisasi SQLite connection pool WAL mode
│   ├── util/                # Helper waktu WIB, parser tanggal alami, & path resolver
│   ├── schedule/            # ScheduleEngine, ClassManager, & OverrideManager
│   ├── task/                # TaskManager SQLite & kalkulasi deadline
│   ├── link/                # LinkManager SQLite tautan perkuliahan
│   ├── chat/                # ChatSettingsManager & GroupAdminResolver
│   ├── reminder/            # Cron broadcast pagi otomatis (06:00 WIB)
│   ├── bot/                 # whatsmeow client & message dispatcher
│   └── api/                 # HTTP REST API server untuk Web Admin Dashboard
├── data/
│   ├── jadwal/              # Berkas master JSON kurikulum per kelas (19 kelas)
│   └── jadwal.json          # Berkas kurikulum default
├── storage/                 # Direktori runtime terisolasi (di-ignore oleh git)
│   ├── tugas.db             # Database SQLite utama aplikasi (tugas, link, setting kelas)
│   ├── sesi_bot.db          # Database sesi login WhatsMeow produksi (Azure VM)
│   └── reminder_groups.json # Data preferensi broadcast pengingat
├── docs/                    # Dokumentasi lengkap sistem & tim
│   ├── TEAM_ONBOARDING.md   # Panduan kickoff meeting tim pengembang
│   ├── DASHBOARD_PRD.md     # Spesifikasi fitur Web Dashboard untuk UI/UX & Frontend
│   ├── CONTRIBUTING.md      # Panduan kontribusi, git workflow, dan DoD
│   ├── ARCHITECTURE.md      # Cetak biru arsitektur teknis sistem
│   ├── DEPLOYMENT.md        # Panduan operasional server Azure & DevOps
│   ├── PANDUAN_PENGGUNAAN.md# Panduan lengkap fitur bot untuk pengguna
│   ├── PRD.md               # Spesifikasi awal produk bot
│   └── TODO.md              # Roadmap & catatan pengembangan
├── .agent/                  # Custom Agent Skills untuk bimbingan tim & AI
│   └── skills/
│       ├── vibe-coding-guide/  # Panduan guardrails arsitektur & edukasi konsep
│       ├── git-team-flow/      # Panduan alur Git, branch, dan PR
│       └── ticket-creator/     # Standar pembuatan tiket backlog
├── .github/                 # Template kolaborasi GitHub
│   ├── pull_request_template.md # Template standar Pull Request
│   └── ISSUE_TEMPLATE/      # Template Bug Report & Feature Request
└── README.md                # Dokumentasi utama proyek
```

---

## 📱 Ringkasan Perintah Chat Bot (Cheat Sheet)

*(Gunakan prefix `!`, `/`, atau `#` di grup WhatsApp. Di chat pribadi bisa diketik langsung).*

### 1. Jadwal Perkuliahan
* `!jadwal` / `!jadwal hari ini` ➔ Menampilkan jadwal kuliah hari ini.
* `!jadwal besok` ➔ Menampilkan jadwal kuliah besok.
* `!jadwal sekarang` ➔ Menampilkan mata kuliah yang sedang berlangsung saat ini.
* `!jadwal senin` s/d `!jadwal jumat` ➔ Menampilkan jadwal pada hari tertentu.
* `!seminggu` ➔ Menampilkan jadwal lengkap Senin sampai Jumat.
* `!dosen [nama/kode]` ➔ Mencari jadwal berdasarkan inisial atau nama dosen.
* `!ruang [nama]` ➔ Mencari jadwal berdasarkan nama ruangan/lab.

### 2. Manajemen Tugas & Deadline Tracker
* `!tugas` ➔ Menampilkan daftar tugas aktif beserta hitung mundur (*countdown*).
* `!tugas tambah <Nama Tugas> | <Matkul> | <Deadline>` ➔ Menambah tugas baru *(Khusus Admin Grup / Ketua Kelas)*.  
  *Contoh:* `!tugas tambah Laporan Praktikum | Basis Data | Jumat 23:59`
* `!tugas selesai <ID>` ➔ Menandai tugas sebagai selesai.
* `!tugas hapus <ID>` ➔ Menghapus tugas dari sistem.

### 3. Tautan Penting Kelas (!link)
* `!link` / `!tautan` ➔ Menampilkan seluruh link penting (Drive, Zoom/Meet, Portal).
* `!drive` ➔ Menampilkan link penyimpanan materi Google Drive kelas.
* `!zoom` / `!meet` ➔ Menampilkan link perkuliahan daring aktif.

### 4. Pengingat Pagi & Pengaturan Kelas
* `!reminder on` / `!reminder off` ➔ Mengaktifkan / mematikan broadcast jadwal pagi (**06:00 WIB**).
* `!daftarkelas` ➔ Menampilkan 19 pilihan kelas yang tersedia.
* `!setkelas <KODE_KELAS>` ➔ Mengatur kelas untuk grup tersebut *(Contoh: `!setkelas D4-TI-1A`)*.
