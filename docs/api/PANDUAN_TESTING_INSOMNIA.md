# Panduan Pengujian REST API Insomnia — Bot Jadwal v3.0

Dokumen teknis ini memandu pengujian endpoint REST API Bot Jadwal v3.0 menggunakan **Insomnia REST Client**. Koleksi disusun dalam format native **Insomnia v4** dengan pembagian wewenang peran aktor, konfigurasi environment variabel, dan otomatisasi ekstraksi token (*Response Chaining*).

---

## 1. Berkas Koleksi Insomnia

Koleksi pengujian tersedia pada:
* `bot-jadwal-v1.insomnia_collection.json` (root repositori)
* `docs/api/bot-jadwal-v1.insomnia_collection.json`

Insomnia juga mendukung impor langsung berkas Postman `bot-jadwal-v1.postman_collection.json`. Namun, berkas native `.insomnia_collection.json` telah dikonfigurasi dengan *template tag* Nunjucks untuk integrasi token otomatis.

---

## 2. Prosedur Impor

1. Buka aplikasi **Insomnia**.
2. Pada panel navigasi utama, pilih menu **Import** atau seret (*drag and drop*) berkas `bot-jadwal-v1.insomnia_collection.json` ke dalam jendela Insomnia.
3. Konfirmasi impor ke workspace aktif atau buat workspace baru bernama **Bot Jadwal v3.0 - API Testing Suite**.

Seluruh 7 folder dan 49 request akan dimuat ke dalam workspace.

---

## 3. Konfigurasi Environment

Insomnia menyediakan dua sub-environment di bawah *Base Environment*:

```text
Base Environment
├── Auto Chaining (Otomatis)
└── Manual Token (Pengujian Negatif)
```

### A. Sub-Environment: Auto Chaining (Otomatis)
Sub-environment ini mengekstrak nilai token dari respons login menggunakan template tag Nunjucks:
1. Pilih sub-environment **Auto Chaining (Otomatis)** pada menu dropdown environment (kiri atas atau via `Ctrl + E`).
2. Jalankan request login pada folder `01. Otentikasi & Akun`:
   * `[Auth] Login Admin (System Administrator)`
   * `[Auth] Login KM (Ketua Murid)`
   * `[Auth] Login PJ (Penanggung Jawab)`
3. Token dari properti `data.token` pada respons JSON otomatis disimpan dalam memori Insomnia dan diteruskan ke header `Authorization: Bearer ...` pada setiap request berikutnya.
4. Penguji tidak perlu menyalin token secara manual antar request.

### B. Sub-Environment: Manual Token (Pengujian Negatif)
Sub-environment ini digunakan untuk pengujian validasi otentikasi dan otorisasi:
1. Pilih sub-environment **Manual Token (Pengujian Negatif)**.
2. Buka dialog *Manage Environments* (`Ctrl + E`).
3. Masukkan nilai secara manual untuk skenario uji:
   * Mengosongkan variabel `tokenKM` untuk menguji respon `401 UNAUTHENTICATED`.
   * Memasukkan token dengan format salah atau kadaluarsa.
   * Menukar token peran PJ ke wewenang KM untuk memvalidasi respon `403 FORBIDDEN`.

---

## 4. Prasyarat dan Eksekusi Server

Sebelum memulai pengujian, inisialisasi database dan jalankan server backend lokal:

```bash
# 1. Inisialisasi data kelas pilot dan akun pengurus demo
go run ./cmd/seed-v1

# 2. Jalankan server HTTP lokal
go run ./cmd/bot -web-only
```
*Host pengujian:* `http://localhost:8080`

### Kredensial Akun Pengujian Bawaan
| Peran | Nomor WhatsApp (`identity_key`) | Kata Sandi | Variabel Insomnia | Cakupan Otorisasi (*Scope*) |
|---|---|---|---|---|
| **System Administrator** | `+6281111111111` | `password123` | `{{ _.tokenAdmin }}` | **GLOBAL** (Operasi sistem, telemetri, restore) |
| **Ketua Murid (KM)** | `+6281234567890` | `password123` | `{{ _.tokenKM }}` | **CLASS** (Penuh pada kelas `d4-ti-2024-a`) |
| **PJ Mata Kuliah** | `+6281298765432` | `password123` | `{{ _.tokenPJ }}` | **OFFERING** (Mata kuliah Sistem Operasi Praktikum) |
| **Mahasiswa** | *Tanpa Akun* | *Tanpa Sandi* | *Tanpa Token* | Akses publik hanya-baca via `/api/v1/portal/:slug` |

---

## 5. Struktur Modul Koleksi

Koleksi dibagi ke dalam 7 folder berdasarkan batasan akses:

1. **01. Otentikasi & Akun**: Manajemen sesi token Bearer, rotasi konteks peran, dan terminasi sesi.
2. **02. Mahasiswa (Portal Publik)**: Akses publik hanya-baca untuk jadwal efektif, tugas aktif, dan materi.
3. **03. Penanggung Jawab Mata Kuliah (PJ)**: Pengelolaan draf tugas, publikasi tugas, draf kuliah pengganti, deteksi bentrok, dan materi matkul.
4. **04. Ketua Murid (Pemeriksaan & Kontrol Kelas)**: Peninjauan tugas, pencabutan jadwal keliru, konfirmasi ruangan TU, impor kurikulum batch, aktivasi semester, dan pencadangan database kelas.
5. **05. Administrator Sistem (Operasi Global & Pemulihan)**: Metrik telemetri global, suspensi/pemulihan akun pengguna, antrean pesan siaran, log audit menyeluruh, dan verifikasi integritas restore.
6. **06. Sistem & Telemetri Server**: Health check endpoint dan status memori bot WhatsApp.
7. **07. Legacy Shim (Kompatibilitas Bot Lama)**: Endpoint kompatibilitas v0 (`/api/classes`, `/api/schedule`, `/api/tasks`) dengan header `Deprecation: true`.

---

## 6. Panel Dokumentasi pada Request

Setiap request di Insomnia dilengkapi spesifikasi teknis pada tab **Docs** (panel tengah, sebelah tab *Body* dan *Header*):
* **Tujuan Pengujian**: Batasan fungsional endpoint.
* **Aktor dan Prasyarat**: Peran yang diizinkan dan variabel yang dibutuhkan.
* **Parameter Kunci**: Struktur payload JSON atau query parameter.
* **Kriteria Kelulusan**: Status kode HTTP dan bentuk envelope respon.
* **Kasus Negatif**: Batas validasi, skenario konflik data, dan respon penolakan akses.

---

## 7. Kamus Status Kode Respon

| Kode HTTP | Kode Error JSON | Kondisi Pemicu |
|---|---|---|
| `200 OK` | - | Permintaan query atau pembaruan berhasil diproses. |
| `201 Created` | - | Entitas baru berhasil dibuat (tugas, event, konfirmasi, backup). |
| `401 Unauthorized` | `UNAUTHENTICATED` | Token tidak disertakan, format tidak valid, atau sesi telah dicabut. |
| `403 Forbidden` | `FORBIDDEN` | Token valid tetapi peran pengguna tidak memiliki wewenang pada target. |
| `404 Not Found` | `NOT_FOUND` | Data kelas, tugas, event, atau semester tidak ditemukan. |
| `409 Conflict` | `VERSION_CONFLICT` | Versi data yang dikirim lebih rendah dari versi data server (optimistic lock). |
| `422 Unprocessable`| `VALIDATION` | Data input melanggar aturan logika bisnis (jam bertabrakan, judul kosong). |
| `429 Too Many Req` | `TOO_MANY_REQUESTS` | Frekuensi request melebihi batas rate limit keamanan. |
