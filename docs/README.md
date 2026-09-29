# 📚 Pusat Navigasi Dokumentasi (Documentation Index)
# Bot Jadwal v3.0

Selamat datang di pusat dokumentasi resmi **Bot Jadwal v3.0**. Dokumen di repositori ini telah ditata secara modular agar setiap anggota tim (Frontend, Backend, UI/UX, QA, DevOps, dan Pengurus Kelas) dapat menemukan informasi yang dibutuhkan dengan cepat tanpa tersesat.

---

## 🗺️ Peta Direktori & Struktur Dokumentasi

```text
docs/
├── README.md                      # Indeks navigasi ini
├── PRD.md                         # Product Requirements Document kanonis (Visi, Scope, Target)
├── TEAM_ONBOARDING.md             # Panduan kickoff, buddy system, dan onboarding pengembang baru
├── CONTRIBUTING.md                # Standar kontribusi Git, Conventional Commits, dan etika PR
├── CONFIG_AND_SECURITY.md         # Referensi variabel lingkungan, CSP, HSTS, CORS, dan secret
├── DEPLOYMENT.md                  # Runbook operasional Azure VM, SOP deploy 6-langkah, dan rollback
├── PANDUAN_PENGGUNAAN.md          # Panduan lengkap perintah chat bot WhatsApp dan Web Dashboard
├── RELEASE_NOTES_v3.0.0.md        # Catatan rilis resmi versi 3.0.0
│
├── adr/                           # Architectural Decision Records (ADR-0001 s.d. ADR-0010)
│   ├── 0001-scope-v1-minimal.md
│   ├── ...
│   └── 0010-cors-and-security-headers.md
│
├── api/                           # Spesifikasi dan panduan teknis REST API v1
│   ├── API_V1.md                  # Kontrak resmi REST API v1 (Endpoints, Envelopes, Codes)
│   ├── DOKUMENTASI_API_FRONTEND.md# Panduan integrasi API v1 untuk Frontend Developer
│   ├── PANDUAN_TESTING_POSTMAN.md # Panduan pengujian API via Postman
│   ├── PANDUAN_TESTING_INSOMNIA.md# Panduan pengujian API via Insomnia
│   └── collections/               # Berkas JSON koleksi API siap impor (Postman & Insomnia)
│
├── product/                       # Spesifikasi fungsional dan model data kanonis
│   ├── PRODUCT_DEFINITION.md      # Arah produk, peran, dan aturan bisnis utama
│   ├── FUNCTIONAL_REQUIREMENTS.md # 48 Kebutuhan Fungsional kanonis (FR-*)
│   ├── ACCESS_CONTROL.md          # Matriks wewenang peran (Admin, KM, PJ, Mahasiswa)
│   ├── DATA_MODEL.md              # Model data relasional target v3.0.2
│   ├── ERD.md                     # Diagram relasi entitas Mermaid
│   └── TRACEABILITY.md            # Matriks ketertelusuran kebutuhan ke kode
│
├── design/                        # Desain antarmuka, alur pengguna, dan token UI
│   ├── SCREEN_INVENTORY.md        # Inventaris layar dashboard dan portal
│   └── flows/                     # Diagram alur interaksi pengguna
│
├── qa/                            # Laporan jaminan kualitas dan bukti pengujian
│   ├── QA_REPORT_QA-001_QA-014.md # Laporan hasil eksekusi 14 tiket QA
│   └── EVIDENCE_MANIFEST.md       # Dossier bukti rilis dan hash commit
│
├── ops/                           # Operasional gerbang produksi dan cutover
│   └── PRODUCTION_GATE_SIGN_OFF.md# Catatan sign-off Gate A s.d. Gate G komite rilis
│
└── archive/                       # Arsip dokumen historis (Discovery & Sprint Task)
    ├── tickets/                   # Berkas tiket kerja sprint (BE-005 s.d. BE-014, QA-001..014)
    └── user-interviews/           # Transkrip wawancara 11 pengguna awal fase discovery
```

---

## 🧭 Jalur Baca Cepat Sesuai Peran Anda (Reading Path)

Pilihlah jalur baca di bawah ini sesuai dengan peran dan kebutuhan Anda:

### 1. 🚀 Pengembang Baru yang Baru Bergabung (New Developer Onboarding)
1. Baca [`docs/PRD.md`](PRD.md) untuk memahami masalah yang diselesaikan dan visi produk.
2. Baca [`docs/product/PRODUCT_DEFINITION.md`](product/PRODUCT_DEFINITION.md) untuk memahami aturan main sistem.
3. Baca [`docs/TEAM_ONBOARDING.md`](TEAM_ONBOARDING.md) untuk menyiapkan lingkungan kerja lokal di komputer Anda.
4. Baca [`docs/CONTRIBUTING.md`](CONTRIBUTING.md) untuk memahami alur branching Git dan penulisan commit.

### 2. ⚙️ Backend Developer
1. Baca [`docs/api/API_V1.md`](api/API_V1.md) untuk mempelajari seluruh spesifikasi rute dan error codes.
2. Baca [`docs/product/DATA_MODEL.md`](product/DATA_MODEL.md) dan [`docs/product/ERD.md`](product/ERD.md) untuk memahami tabel SQLite.
3. Baca [`docs/adr/`](adr/) untuk mengetahui alasan di balik keputusan teknis (ADR-0001 s.d. 0010).

### 3. 💻 Frontend Developer & UI/UX Designer
1. Baca [`docs/api/DOKUMENTASI_API_FRONTEND.md`](api/DOKUMENTASI_API_FRONTEND.md) untuk integrasi HTTP client ke REST API.
2. Buka [`docs/design/SCREEN_INVENTORY.md`](design/SCREEN_INVENTORY.md) dan [`docs/design/flows/`](design/flows/) untuk melihat desain alur tampilan.
3. Impor koleksi pengujian di [`docs/api/collections/`](api/collections/) ke Postman atau Insomnia.

### 4. 🧪 QA / Software Tester
1. Baca [`docs/product/FUNCTIONAL_REQUIREMENTS.md`](product/FUNCTIONAL_REQUIREMENTS.md) untuk daftar acuan ID FR.
2. Buka [`docs/api/PANDUAN_TESTING_POSTMAN.md`](api/PANDUAN_TESTING_POSTMAN.md) untuk menjalankan tes API otomatis.
3. Lihat riwayat pengujian di [`docs/qa/QA_REPORT_QA-001_QA-014.md`](qa/QA_REPORT_QA-001_QA-014.md).

### 5. ☁️ DevOps, Operator Server & Koordinator Kelas
1. Baca [`docs/DEPLOYMENT.md`](DEPLOYMENT.md) untuk panduan rilis server Azure VM, verifikasi SHA-256, dan SOP rollback.
2. Baca [`docs/CONFIG_AND_SECURITY.md`](CONFIG_AND_SECURITY.md) untuk konfigurasi environment variables dan rate limiting.
3. Baca [`docs/PANDUAN_PENGGUNAAN.md`](PANDUAN_PENGGUNAAN.md) untuk daftar lengkap perintah WhatsApp bot dan dashboard web.
4. Periksa evaluasi produksi di [`docs/ops/PRODUCTION_GATE_SIGN_OFF.md`](ops/PRODUCTION_GATE_SIGN_OFF.md).

---

## 📦 Mengapa Ada Folder `archive/`?

Folder [`docs/archive/`](archive/) memuat dokumen-dokumen yang **telah selesai dieksekusi atau bersifat historis**:
- **`archive/tickets/`**: Berkas spesifikasi tiket kerja sprint terdahulu (BE-005 s.d. BE-014, QA-001 s.d. QA-014, dan Production Gate). Berkas ini disimpan murni sebagai jejak audit rekayasa perangkat lunak, bukan sebagai panduan harian pengembang.
- **`archive/user-interviews/`**: Hasil wawancara 11 mahasiswa dan pengurus kelas saat fase riset awal produk.
