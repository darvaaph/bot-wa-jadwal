# Bot WhatsApp Jadwal Kuliah & Web Dashboard

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Frontend](https://img.shields.io/badge/Frontend-Tailwind_%2B_Alpine.js-38B2AC?style=flat&logo=tailwindcss)](https://tailwindcss.com)
[![Database](https://img.shields.io/badge/Database-SQLite_WAL-003B57?style=flat&logo=sqlite)](https://sqlite.org)
[![Library](https://img.shields.io/badge/WhatsApp-whatsmeow-25D366?style=flat&logo=whatsapp)](https://github.com/tulir/whatsmeow)

Aplikasi bot WhatsApp dan web dashboard terpadu dalam satu biner Go (`embed.FS`). Digunakan untuk mengelola jadwal kuliah, tenggat tugas, repositori materi, dan pengingat otomatis kelas.

---

## Cara Menjalankan

Pastikan Go 1.22+ sudah terpasang.

### 1. Mode Web & API Saja (Frontend / UI)
Menjalankan dashboard dan REST API di `http://localhost:8080` tanpa WhatsApp:
```powershell
go run ./cmd/bot -web-only
```

### 2. Mode Bot WhatsApp Penuh (Backend / Testing)
Menjalankan web dan bot WhatsApp dengan sesi uji coba terisolasi:
```powershell
go run ./cmd/bot -session storage/sesi_dev.db
```
*Scan QR code di terminal dengan nomor WhatsApp testing.*

### 3. Berbagi Akses Lokal ke Teman (Cloudflare Tunnel)
Jika ingin membagikan tampilan lokal ke teman tanpa deploy:
```powershell
cloudflared tunnel --url http://localhost:8080
```
*Salin tautan `https://xxxx.trycloudflare.com` dari terminal dan bagikan ke teman.*

### 4. Build & Testing
```powershell
# Uji coba unit test
go test -v ./...

# Periksa kesalahan kode
go vet ./...

# Kompilasi aplikasi menjadi binary tunggal
go build -o bin/bot ./cmd/bot
```

### 5. Akun Admin Lokal (Opsional)
Untuk membuat akun System Admin lokal pertama kali:
```powershell
$env:BOT_JADWAL_AUTH_HASH_KEY = "kunci-acak-minimal-32-byte-disini"
$env:BOT_JADWAL_ADMIN_PASSWORD = "password-minimal-12-karakter"
go run ./cmd/provision-admin -identity "admin@example.test" -display-name "System Admin"
```

---

## Aturan Pengembangan

1. **Frontend Tanpa Build Step (No NPM / Node.js)**
   Frontend murni HTML5 dengan Tailwind dan Alpine.js vendored di `web/`. Jangan menambahkan `package.json` atau folder `node_modules`.
2. **Database SQLite WAL**
   Data disimpan di `storage/` dengan mode WAL. Selalu gunakan prepared statement (`?`) untuk query SQL.
3. **Alur Kerja Git**
   - Jangan push langsung ke branch `main`.
   - Buat branch fitur per tiket: `feat/nama-fitur` atau `fix/nama-bug`.
   - Gunakan format Conventional Commits (`feat:`, `fix:`, `docs:`).

---

## Perintah Bot WhatsApp

Gunakan awalan `!`, `/`, atau `#` di grup WhatsApp (atau ketik langsung di chat pribadi):

| Kategori | Perintah | Fungsi |
| :--- | :--- | :--- |
| **Jadwal** | `!jadwal` / `!jadwal besok` | Jadwal kuliah hari ini atau besok |
| | `!jadwal sekarang` | Mata kuliah yang sedang berlangsung |
| | `!jadwal senin` - `jumat` | Jadwal pada hari tertentu |
| | `!seminggu` | Jadwal lengkap Senin sampai Jumat |
| | `!dosen [nama]` / `!ruang [kode]` | Cari jadwal berdasarkan dosen atau ruangan |
| **Tugas** | `!tugas` | Daftar tugas aktif beserta hitung mundur tenggat |
| **Tautan** | `!link` / `!drive` / `!zoom` | Tautan perkuliahan, Google Drive, dan kelas daring |
| **Kelas** | `!daftarkelas` | Daftar kelas yang tersedia |

---

## Struktur Direktori

```text
├── cmd/bot/              # Entry point utama aplikasi (main.go)
├── internal/             # Domain logic (api, bot, schedule, task, database)
├── web/                  # Dashboard frontend (HTML, CSS, JS, partials)
├── data/jadwal/          # Data kurikulum kelas (.json)
├── storage/              # Database runtime (.db) & sesi WA (di-ignore git)
├── docs/                 # Dokumentasi produk, arsitektur, dan operasional
└── .agent/               # Panduan tim dan skill AI
```

---

## Navigasi Dokumentasi Tim

- 🧭 [Arah dan Definisi Produk](docs/product/PRODUCT_DEFINITION.md)
- ✅ [Kebutuhan Fungsional](docs/product/FUNCTIONAL_REQUIREMENTS.md)
- 🧱 [Model Data & ERD](docs/product/DATA_MODEL.md)
- ☁️ [Panduan Kesiapan Produksi](docs/ops/PRODUCTION_READINESS.md)
- 🚀 [Panduan Deployment Server Linux](docs/ops/DEPLOYMENT_GUIDE.md)
