# 📜 Evidence Manifest & Release Candidate Dossier
# Versi Produk: Bot Jadwal v3.0.0 (Release Candidate 1)

Dokumen ini adalah **paket bukti teknis (Evidence Manifest)** resmi yang mengikat seluruh spesifikasi kebutuhan (*requirements*), perubahan kode sumber, hasil kompilasi, laporan pengujian QA, audit keamanan, dan kesiapan operasional rilis **Bot Jadwal v3.0.0**.

---

## 🔖 Identitas Kandidat Rilis (Release Candidate Identification)

| Atribut | Nilai Terverifikasi |
|:---|:---|
| **Versi Produk Kanonis** | `Bot Jadwal v3.0.0` |
| **Commit SHA Kandidat** | `dd08934212ca881e2befdff4a26273ed3c59b5e2` |
| **Branch Sumber** | `FE` (terintegrasi dengan backend, security, dan UI) |
| **Waktu Beku Kandidat (Freeze Timestamp)** | 2026-09-29 18:15:00 WIB |
| **Go Toolchain Version** | `go version go1.22+ windows/amd64` |
| **Target OS / Arsitektur** | `linux/amd64` (Server Azure Ubuntu 24.04 LTS) |
| **Biner Kompilasi Target** | `./cmd/bot` (`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`) |
| **Versi Skema Basis Data** | SQLite Relational Schema v3.0.2 (WAL Mode) |
| **Status Production Gate** | **GO FOR RELEASE** (Lulus Gate A s.d. Gate G) |

---

## 📑 Rekapitulasi Berkas Bukti (Evidence Pack Index)

### 1. Bukti Pengujian Otomatis (Automated Test Suite)
* **Perintah Uji:** `go test -count=1 ./...`
* **Hasil:** **100% PASS** pada 16 paket internal domain.
* **Log Raw Eksekusi:**
  ```text
  ok   bot-jadwal/internal/academic   1.419s
  ok   bot-jadwal/internal/api        12.890s
  ok   bot-jadwal/internal/audit      1.297s
  ok   bot-jadwal/internal/auth       6.727s
  ok   bot-jadwal/internal/bot        1.145s
  ok   bot-jadwal/internal/chat       0.723s
  ok   bot-jadwal/internal/config     0.612s
  ok   bot-jadwal/internal/database   2.139s
  ok   bot-jadwal/internal/link       1.289s
  ok   bot-jadwal/internal/notify     1.865s
  ok   bot-jadwal/internal/ratelimit  1.591s
  ok   bot-jadwal/internal/reminder   0.422s
  ok   bot-jadwal/internal/schedule   1.997s
  ok   bot-jadwal/internal/seed       1.872s
  ok   bot-jadwal/internal/task       1.858s
  ok   bot-jadwal/internal/util       0.600s
  ```
* **Statistik Cacat:** 0 Blocker, 0 Critical, 0 Major Defect (Zero S0/S1/S2 Defects).

### 2. Bukti Analisis Kode Statis (Static Code Analysis)
* **Perintah:** `go vet ./...`
* **Hasil:** **PASS (0 Temuan / Clean Exit Code 0)**.
* **Format Kode:** Seluruh berkas Go terformat rapi sesuai standar `gofmt`.

### 3. Matriks Ketertelusuran Kebutuhan (Traceability Matrix)
* **Lokasi Dokumen:** [`docs/product/TRACEABILITY.md`](file:///f:/Project/bot-wa-jadwal/docs/product/TRACEABILITY.md)
* **Cakupan:** 48 Kebutuhan Fungsional (`FR-ACCESS-001..007`, `FR-CLASS-001..005`, `FR-SEM-001..005`, `FR-SCH-001..008`, `FR-ROOM-001..004`, `FR-TASK-001..008`, `FR-NOTIF-001..005`, `FR-AUDIT-001..002`, `FR-UX-001..002`, `FR-OPS-001..002`) terpetakan 100% ke unit test dan REST API handler.

### 4. Laporan Final Quality Assurance (QA-001 s.d. QA-014)
* **Lokasi Dokumen:** [`docs/qa/QA_REPORT_QA-001_QA-014.md`](file:///f:/Project/bot-wa-jadwal/docs/qa/QA_REPORT_QA-001_QA-014.md)
* **Status Backlog QA:** Seluruh 14 tiket QA diselesaikan dan berstatus **LULUS**.
* **Pengujian Khusus:**
  - QA-011: Verifikasi Keamanan Produksi (BE-012..BE-014) ➔ **LULUS**
  - QA-012: Aksesibilitas Web & Mobile Responsive 390px ➔ **LULUS**
  - QA-013: Ketahanan Reliabilitas & Auto-Reconnect Goroutine ➔ **LULUS**
  - QA-014: Full Regression & Pilot Acceptance ➔ **LULUS**

### 5. Bukti Audit Keamanan & Hardening (BE-012 s.d. BE-014)
* **Lokasi Dokumen:** [`docs/CONFIG_AND_SECURITY.md`](file:///f:/Project/bot-wa-jadwal/docs/CONFIG_AND_SECURITY.md)
* **Unit Test Keamanan:** `internal/api/ratelimit_endpoints_test.go` (BE-012) dan `internal/api/security_headers_cors_test.go` (BE-014) lulus tanpa kegagalan.
* **Fitur Terverifikasi:**
  - Rate Limiting Persistent SQLite (`security_attempts`) sesuai ADR-0009.
  - Fail-Closed Startup Validation pada konfigurasi produksi `internal/config/config.go`.
  - Proteksi Anti-CSRF mutasi cookie via ADR-0010.
  - Exact-Origin CORS, CSP tanpa wildcard, HSTS 1 tahun, dan `Cache-Control: no-store`.

### 6. Bukti Kesiapan Operasional & Disaster Recovery (DOC-004, OPS-001)
* **Lokasi Dokumen:** [`docs/DEPLOYMENT.md`](file:///f:/Project/bot-wa-jadwal/docs/DEPLOYMENT.md)
* **Prosedur:**
  - SOP Deploy 6-Langkah teruji secara simulasi di lingkungan lokal/staging.
  - Verifikasi integritas checksum biner via SHA-256 (`sha256sum -c`).
  - Pre-deploy database backup snapshot terisolasi di `storage/backups/`.
  - SOP Rollback Darurat (Rollback Runbook) siap pakai dengan batas waktu observasi 15 menit.

---

## 🔒 Pernyataan Integritas & Ketiadaan Data Rahasia (Sanitization Guarantee)

Seluruh bukti teknis, log pengujian, dan berkas dokumentasi dalam repositori ini telah diaudit dan **dinyatakan bersih dari:**
1. Kunci rahasia HMAC produksi (`BOT_JADWAL_AUTH_HASH_KEY` asli).
2. Kata sandi akun administrator produksi (`BOT_JADWAL_ADMIN_PASSWORD`).
3. Sesi WhatsApp produksi (`storage/sesi_bot.db`).
4. Data pribadi nyata mahasiswa atau nomor telepon privat dosen.
5. Alamat IP atau kredensial SSH privat server cloud.

---

## ✍️ Tanda Tangan Komite Rilis (Release Committee Sign-Off)

| Peran | Nama / Perwakilan | Keputusan | Tanggal |
|:---|:---|:---:|:---:|
| **Project Manager / Tech Lead** | Darva Aryasatya Putra Hermawan | **APPROVED (GO)** | 29 September 2026 |
| **Backend & Security Lead** | Darva Aryasatya Putra Hermawan | **APPROVED (GO)** | 29 September 2026 |
| **Frontend & UI Lead** | Tim Pengembang Frontend | **APPROVED (GO)** | 29 September 2026 |
| **QA & Testing Lead** | Tim Quality Assurance | **APPROVED (GO)** | 29 September 2026 |
| **DevOps & Cloud Lead** | Tim Infrastruktur Server | **APPROVED (GO)** | 29 September 2026 |
