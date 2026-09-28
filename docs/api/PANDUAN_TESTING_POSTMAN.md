# Panduan Pengujian REST API Bot Jadwal v3.0

Dokumen ini adalah panduan teknis untuk menguji seluruh endpoint REST API Bot Jadwal v3.0 menggunakan **Postman**. Koleksi dirancang mengikuti standar industri rekayasa perangkat lunak: dikelompokkan secara tegas berdasarkan **peran aktor (Role-Based)** dan alur kerja nyata di kampus, bebas dari istilah klise (*no AI slop*).

---

## 1. Berkas Koleksi Postman

Berkas koleksi siap diimpor berada di:
* `bot-jadwal-v1.postman_collection.json` (root proyek)
* `docs/api/bot-jadwal-v1.postman_collection.json`

### Variabel Bawaan Koleksi (Collection Variables)
Saat koleksi diimpor, variabel berikut sudah terpasang otomatis:

| Variabel | Nilai Default | Penjelasan |
|---|---|---|
| `baseUrl` | `http://localhost:8080` | Alamat host lokal server Go |
| `classSlug` | `d4-ti-2024-a` | Slug kelas pilot resmi (D4 TI Angkatan 2024 Kelas A) |
| `token` | *(Otomatis terisi)* | Token Bearer aktif untuk request pengurus |
| `tokenKM` | *(Otomatis terisi)* | Token Bearer tersimpan khusus peran Ketua Murid |
| `tokenPJ` | *(Otomatis terisi)* | Token Bearer tersimpan khusus peran PJ Mata Kuliah |
| `currentTaskId` | `1` | ID tugas yang sedang diuji |
| `currentEventId` | `1` | ID kejadian perkuliahan yang sedang diuji |

> **Fitur Otomatis:** Begitu Anda menekan tombol **Send** pada request `[Auth] Login KM` atau `[Auth] Login PJ`, script uji bawaan koleksi akan otomatis menangkap token dari respons dan menyimpannya ke variabel `{{token}}`. Anda tidak perlu menyalin token secara manual.

---

## 2. Prasyarat & Menjalankan Server

### Langkah 1: Pastikan Database Target v1 Terisi
Jika berkas `storage/bot_v1.db` belum ada atau ingin di-reset ulang, jalankan perintah seed pilot di terminal:
```bash
go run ./cmd/seed-v1
```
Perintah ini otomatis mengimpor jadwal resmi 2 kelas pilot (`D4-TI-2024-A` dan `D4-TI-2024-B`) serta menyiapkan akun demo pengurus.

### Langkah 2: Jalankan Server HTTP
Jalankan server backend dalam mode pengujian API:
```bash
go run ./cmd/bot -web-only
```
*Port `8080` akan aktif dan siap menerima panggilan dari Postman.*

---

## 3. Akun Pengujian Bawaan (*Seed Credentials*)

Akun berikut telah disiapkan pada database untuk mempermudah pengujian hak akses (RBAC):

| Peran | Nomor WhatsApp (`identity_key`) | Kata Sandi | Cakupan Wewenang (*Scope*) |
|---|---|---|---|
| **Ketua Murid (KM)** | `+6281234567890` | `password123` | Seluruh mata kuliah di kelas `d4-ti-2024-a` |
| **PJ Mata Kuliah** | `+6281298765432` | `password123` | Khusus mata kuliah penugasan (Aljabar Linear) |
| **Mahasiswa** | *Tanpa akun* | *Tanpa sandi* | Akses publik hanya-baca via slug portal `/c/:slug` |

---

## 4. Struktur Folder Koleksi Postman

Koleksi dibagi ke dalam 6 folder logis:

```text
Bot Jadwal v3.0 - API Testing Suite
├── 01. Otentikasi & Akun
│   ├── [Auth] Login KM (Ketua Murid)
│   ├── [Auth] Login PJ (Penanggung Jawab)
│   ├── [Auth] Profil Saya & Konteks Aktif (/me)
│   ├── [Auth] Ganti Konteks Peran (Switch Context)
│   └── [Auth] Logout Sesi
├── 02. Mahasiswa (Portal Publik)
│   ├── [Portal] Ringkasan Hari Ini (Summary)
│   ├── [Portal] Jadwal Kuliah Efektif
│   ├── [Portal] Daftar Tugas Terbit
│   ├── [Portal] Detail Tugas & Materi Terkait
│   ├── [Portal] Riwayat Perubahan Jadwal (Changes)
│   └── [Portal] Direktori Materi & Link Perkuliahan
├── 03. PJ Mata Kuliah (Pengelolaan Tugas & Sesi)
│   ├── [PJ] Simpan Draf Tugas Baru
│   ├── [PJ] Terbitkan Tugas (Publish)
│   ├── [PJ] Perbarui Tugas (Optimistic Locking)
│   ├── [PJ] Ajukan Draf Kuliah Pengganti
│   ├── [PJ] Periksa Bentrok Jadwal (Preview Conflict)
│   ├── [PJ] Terbitkan Jadwal Pengganti (Publish)
│   └── [PJ] Tambah Materi / Slide Perkuliahan
├── 04. Ketua Murid (Pemeriksaan & Kontrol Kelas)
│   ├── [KM] Lihat Antrean Tugas Butuh Review
│   ├── [KM] Setujui Tugas (Approve Review)
│   ├── [KM] Minta Revisi Tugas (Kembalikan ke Draf)
│   ├── [KM] Batalkan Tugas Sepihak (Revoke Task)
│   ├── [KM] Cabut Jadwal Pengganti (Revoke Event)
│   ├── [KM] Buat Undangan PJ Mata Kuliah Baru
│   └── [KM] Aktivasi Semester Baru
├── 05. Sistem & Fitur v1.1+
│   ├── Health Check HTTP Server
│   ├── Telemetri WhatsApp & Kelas Aktif
│   ├── [Admin] Telemetri Global Sistem
│   ├── [Admin] Penangguhan Akun Pengguna (Suspend)
│   ├── [Admin] Pemulihan Akun Pengguna (Recover)
│   ├── [Audit] Log Audit Terpusat
│   ├── [Backup] Buat Cadangan Basis Data (Backup)
│   ├── [Restore] Verifikasi & Pemulihan Cadangan
│   ├── [Ruangan] Rekomendasi Ruangan Kosong
│   ├── [Ruangan] Catat Konfirmasi Ruangan TU
│   ├── [Notifikasi] Antrean Siaran WhatsApp
│   ├── [Notifikasi] Coba Ulang Siaran (Retry)
│   ├── [Impor] Validasi Batch Kurikulum
│   └── [Impor] Terapkan Batch Kurikulum (Apply)
└── 06. Legacy Shim (Kompatibilitas Bot Lama)
    ├── [Legacy] Daftar Kelas Lama
    ├── [Legacy] Jadwal Kuliah
    └── [Legacy] Daftar Tugas
```

---

## 5. Skenario Pengujian Bertahap

Untuk memverifikasi alur kerja aplikasi secara menyeluruh, jalankan request sesuai urutan skenario berikut:

### Skenario A: Mahasiswa Melihat Informasi Kelas (Tanpa Login)
Mahasiswa tidak memerlukan otentikasi. Endpoint ini dapat langsung ditembak kapan saja:
1. Buka folder **`02. Mahasiswa (Portal Publik)`**.
2. Jalankan `[Portal] Ringkasan Hari Ini (Summary)` $\rightarrow$ Verifikasi respon HTTP 200 memuat `now_event`, `next_event`, dan `nearest_tasks`.
3. Jalankan `[Portal] Jadwal Kuliah Efektif` $\rightarrow$ Verifikasi daftar jadwal memuat gabungan pola reguler dan kuliah pengganti.
4. Jalankan `[Portal] Daftar Tugas Terbit` $\rightarrow$ Verifikasi daftar tugas yang berstatus `PUBLISHED`.

---

### Skenario B: Siklus Hidup Tugas (PJ Terbitkan $\rightarrow$ KM Review $\rightarrow$ Mahasiswa Baca)
Menguji tata kelola tugas dua lapis (publikasi langsung PJ dan peninjauan retrospektif KM):

1. **Login sebagai PJ**:
   * Buka folder **`01. Otentikasi & Akun`**.
   * Jalankan `[Auth] Login PJ (Penanggung Jawab)`.
   * *Variabel `{{token}}` otomatis berisi token PJ.*
2. **PJ Terbitkan Tugas Baru**:
   * Buka folder **`03. PJ Mata Kuliah`**.
   * Jalankan `[PJ] Terbitkan Tugas (Publish)`.
   * Respon HTTP 201 Created. Tugas terbit dengan `publication_status: "PUBLISHED"` dan `review_state: "NOT_REVIEWED"`. ID tugas otomatis tersimpan ke `{{currentTaskId}}`.
3. **Mahasiswa Langsung Melihat Tugas di Portal**:
   * Buka folder **`02. Mahasiswa (Portal Publik)`**.
   * Jalankan `[Portal] Daftar Tugas Terbit`. Tugas baru langsung muncul tanpa menunggu KM.
4. **KM Meninjau Tugas**:
   * Buka folder **`01. Otentikasi & Akun`** $\rightarrow$ Jalankan `[Auth] Login KM (Ketua Murid)`.
   * *Variabel `{{token}}` sekarang berganti ke token KM.*
   * Buka folder **`04. Ketua Murid`** $\rightarrow$ Jalankan `[KM] Lihat Antrean Tugas Butuh Review`.
   * Jalankan `[KM] Setujui Tugas (Approve Review)` $\rightarrow$ Respon HTTP 200, tugas berubah menjadi `review_state: "APPROVED"`.

---

### Skenario C: Deteksi Konflik & Penjadwalan Kuliah Pengganti
1. Pastikan Anda sedang login sebagai **PJ**.
2. Buka folder **`03. PJ Mata Kuliah`**.
3. Jalankan `[PJ] Ajukan Draf Kuliah Pengganti` $\rightarrow$ Respon HTTP 201, ID event tersimpan ke `{{currentEventId}}`.
4. Jalankan `[PJ] Periksa Bentrok Jadwal (Preview Conflict)`:
   * Backend membandingkan jam baru terhadap jadwal mata kuliah lain di hari yang sama.
   * Respon memuat status `conflicts: []` dan catatan konfirmasi ruangan.
5. Jalankan `[PJ] Terbitkan Jadwal Pengganti (Publish)`:
   * Header `Idempotency-Key` dikirimkan untuk mencegah duplikasi.
   * Status event menjadi `PUBLISHED`.
6. Buka folder **`02. Mahasiswa (Portal Publik)`** $\rightarrow$ Jalankan `[Portal] Jadwal Kuliah Efektif`:
   * Jadwal pengganti otomatis tersisip pada tanggal pelaksanaan.

---

### Skenario D: Penegakan Hak Akses & Keamanan (RBAC)
Uji keandalan aturan otorisasi server:
1. **Akses Tanpa Token**:
   * Buka request `[PJ] Terbitkan Tugas (Publish)`.
   * Hapus atau kosongkan isi header `Authorization` $\rightarrow$ Kirim request.
   * **Hasil yang diharapkan:** HTTP `401 Unauthorized` dengan kode error `"UNAUTHENTICATED"`.
2. **PJ Mengubah Data Di Luar Wewenang**:
   * Login sebagai **PJ**.
   * Coba jalankan `[KM] Aktivasi Semester Baru` atau `[KM] Setujui Tugas`.
   * **Hasil yang diharapkan:** HTTP `403 Forbidden` dengan kode error `"FORBIDDEN"`.
3. **Pencegahan Brute-Force Login**:
   * Kirim request `[Auth] Login KM` dengan password salah sebanyak 5 kali berturut-turut.
   * **Hasil yang diharapkan:** Pada percobaan ke-6, server menolak dengan HTTP `429 Too Many Requests`.

---

### Skenario E: Verifikasi Fitur Lanjutan v1.1+ (Ruangan, Audit, Backup, Notifikasi, Impor)
Semua fitur v1.1+ kini telah aktif sepenuhnya:
1. **Rekomendasi Ruangan Kosong**:
   * Buka folder **`05. Sistem & Fitur v1.1+`** $\rightarrow$ Jalankan `[Ruangan] Rekomendasi Ruangan Kosong`.
   * Server mencari ruangan berstatus `ACTIVE` yang tidak bentrok dengan jadwal `PUBLISHED` pada rentang waktu `starts_at` s.d `ends_at`.
2. **Konfirmasi Ruangan TU**:
   * Jalankan `[Ruangan] Catat Konfirmasi Ruangan TU` $\rightarrow$ Respon HTTP 201 Created. Kolom `room_id` pada event perkuliahan otomatis diperbarui jika status `CONFIRMED`.
3. **Pencadangan Database & Verifikasi Restore**:
   * Jalankan `[Backup] Buat Cadangan Basis Data` $\rightarrow$ Respon HTTP 201 Created dengan snapshot database SQLite di `storage/backups/` dan checksum SHA-256.
   * Jalankan `[Restore] Verifikasi & Pemulihan Cadangan` $\rightarrow$ Respon HTTP 200 OK dengan status `VERIFIED`.
4. **Validasi & Penerapan Impor Kurikulum Batch**:
   * Jalankan `[Impor] Validasi Batch Kurikulum` $\rightarrow$ Backend memvalidasi integritas kode matkul, dosen, dan offering. Respon status `READY`.
   * Jalankan `[Impor] Terapkan Batch Kurikulum (Apply)` $\rightarrow$ Kurikulum, offering, dan pola jadwal disimpan secara atomik ke database.

---

## 6. Format Respon Standar (Envelope Konvensi)

Setiap endpoint API Bot Jadwal v3.0 mematuhi struktur JSON seragam:

### Format Sukses
```json
{
  "status": "success",
  "data": { ... } // Objek atau Array
}
```

### Format Gagal
```json
{
  "status": "error",
  "error": {
    "code": "VALIDATION",
    "message": "Deskripsi kesalahan yang manusiawi",
    "details": { ... }
  }
}
```

Daftar Kode Error Resmi:
* `UNAUTHENTICATED` (401): Token tidak ada, tidak valid, atau kedaluwarsa.
* `FORBIDDEN` (403): Pengguna terotentikasi tetapi tidak memiliki wewenang pada kelas/mata kuliah target.
* `NOT_FOUND` (404): Entitas tidak ditemukan.
* `VALIDATION` (422): Input form tidak memenuhi batasan logika bisnis.
* `VERSION_CONFLICT` (409): Versi data di server lebih baru daripada versi yang dikirim (optimistic locking).
* `TOO_MANY_REQUESTS` (429): Kenaikan frekuensi request melebihi batas rate limit.
* `NOT_IMPLEMENTED` (501): Fitur direncanakan untuk fase rilis berikutnya (v1.1+).
