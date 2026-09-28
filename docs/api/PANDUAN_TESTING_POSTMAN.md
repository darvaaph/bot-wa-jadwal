# Panduan Pengujian REST API Bot Jadwal v3.0

Dokumen ini adalah panduan teknis resmi untuk menguji seluruh endpoint REST API Bot Jadwal v3.0 menggunakan **Postman**. Koleksi dirancang mengikuti standar industri rekayasa perangkat lunak: dikelompokkan secara tegas berdasarkan **peran aktor (Role-Based)** dan domain operasional sistem, bebas dari istilah klise (*anti AI slop*).

---

## 1. Berkas Koleksi Postman

Berkas koleksi siap diimpor berada di dua lokasi:
* `bot-jadwal-v1.postman_collection.json` (root repositori)
* `docs/api/bot-jadwal-v1.postman_collection.json`

### Variabel Bawaan Koleksi (Collection Variables)
Saat koleksi diimpor ke Postman, variabel berikut sudah terkonfigurasi otomatis:

| Variabel | Nilai Default | Penjelasan |
|---|---|---|
| `baseUrl` | `http://localhost:8080` | Alamat host lokal server Go |
| `classSlug` | `d4-ti-2024-a` | Slug kelas pilot resmi (D4 TI Angkatan 2024 Kelas A) |
| `token` | *(Otomatis terisi)* | Token Bearer sesi aktif saat ini |
| `tokenAdmin` | *(Otomatis terisi)* | Token Bearer khusus peran System Administrator |
| `tokenKM` | *(Otomatis terisi)* | Token Bearer tersimpan khusus peran Ketua Murid |
| `tokenPJ` | *(Otomatis terisi)* | Token Bearer tersimpan khusus peran PJ Mata Kuliah |
| `currentTaskId` | `1` | ID tugas yang sedang diuji |
| `currentEventId` | `1` | ID kegiatan perkuliahan yang sedang diuji |
| `currentBackupId` | `1` | ID berkas cadangan database yang sedang diuji |
| `currentBatchId` | `1` | ID batch kurikulum impor yang sedang diuji |

> **Fitur Otomatisasi Token:** Begitu Anda menekan tombol **Send** pada request `[Auth] Login Admin`, `[Auth] Login KM`, atau `[Auth] Login PJ`, script uji bawaan koleksi akan otomatis menangkap token dari respons JSON dan menyimpannya ke variabel terkait (`{{tokenAdmin}}`, `{{tokenKM}}`, atau `{{tokenPJ}}`) sekaligus memperbarui `{{token}}`. Anda tidak perlu menyalin token secara manual.

---

## 2. Prasyarat & Menjalankan Server

### Langkah 1: Pastikan Database Target v1 Terisi
Jika berkas `storage/bot_v1.db` belum ada atau ingin di-reset ulang ke kondisi awal, jalankan perintah seed di terminal:
```bash
go run ./cmd/seed-v1
```
Perintah ini otomatis mengimpor kurikulum 2 kelas pilot (`d4-ti-2024-a` dan `d4-ti-2024-b`) serta menyiapkan 3 akun demo pengurus resmi.

### Langkah 2: Jalankan Server HTTP
Jalankan server backend dalam mode pengujian web/API:
```bash
go run ./cmd/bot -web-only
```
*Port `8080` akan aktif dan siap menerima panggilan dari Postman.*

---

## 3. Akun Pengujian Bawaan (*Seed Credentials*)

Akun demo berikut disiapkan pada database untuk mempermudah pengujian hak akses berjenjang (RBAC):

| Peran | Nomor WhatsApp (`identity_key`) | Kata Sandi | Cakupan Wewenang (*Scope*) |
|---|---|---|---|
| **System Administrator** | `+6281111111111` | `password123` | **GLOBAL** (Seluruh sistem, lintas kelas, operasi darurat & DevOps) |
| **Ketua Murid (KM)** | `+6281234567890` | `password123` | **CLASS** (Penuh pada kelas `d4-ti-2024-a`) |
| **PJ Mata Kuliah** | `+6281298765432` | `password123` | **OFFERING** (Khusus mata kuliah penugasan: Sistem Operasi Praktikum) |
| **Mahasiswa** | *Tanpa akun* | *Tanpa sandi* | Akses publik hanya-baca via slug portal `/api/v1/portal/:slug` |

---

## 4. Struktur Folder Koleksi Postman

Koleksi dibagi ke dalam 7 folder logis terpisah:

```text
Bot Jadwal v3.0 - API Testing Suite
├── 01. Otentikasi & Akun
│   ├── [Auth] Login Admin (System Administrator)
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
├── 03. Penanggung Jawab Mata Kuliah (PJ)
│   ├── [PJ] Simpan Draf Tugas Baru
│   ├── [PJ] Terbitkan Tugas (Publish)
│   ├── [PJ] Perbarui Tugas (Optimistic Locking)
│   ├── [PJ] Ajukan Draf Kuliah Pengganti
│   ├── [PJ] Cari Kandidat Ruangan Kosong
│   ├── [PJ] Periksa Bentrok Jadwal (Preview Conflict)
│   ├── [PJ] Terbitkan Jadwal Pengganti (Publish)
│   └── [PJ] Tambah Materi / Slide Perkuliahan
├── 04. Ketua Murid (Pemeriksaan & Kontrol Kelas)
│   ├── [KM] Lihat Antrean Tugas Butuh Review
│   ├── [KM] Setujui Tugas (Approve Review)
│   ├── [KM] Minta Revisi Tugas (Kembalikan ke Draf)
│   ├── [KM] Batalkan Tugas Sepihak (Revoke Task)
│   ├── [KM] Cabut Jadwal Pengganti (Revoke Event)
│   ├── [KM] Catat Konfirmasi Ruangan TU
│   ├── [KM] Validasi Batch Kurikulum (Import Validate)
│   ├── [KM] Terapkan Batch Kurikulum (Import Apply)
│   ├── [KM] Aktivasi Semester Baru
│   ├── [KM] Ubah Status Kelas (Active/Inactive/Archived)
│   ├── [KM] Buat Undangan PJ Mata Kuliah Baru
│   ├── [KM] Buat Cadangan Basis Data (Backup Kelas)
│   ├── [KM] Log Audit Perubahan Kelas
│   └── [KM] Antrean Siaran WhatsApp Kelas
├── 05. Administrator Sistem (Operasi Global & Pemulihan)
│   ├── [Admin] Telemetri Global Sistem
│   ├── [Admin] Penangguhan Akun Pengguna (Suspend)
│   ├── [Admin] Pemulihan Akun Pengguna (Recover)
│   ├── [Admin] Rekomendasi Ruangan Kosong Global
│   ├── [Admin] Antrean Siaran WhatsApp Global
│   ├── [Admin] Coba Ulang Siaran (Retry Notification)
│   ├── [Admin] Log Audit Global (Lintas Seluruh Kelas)
│   ├── [Admin] Buat Cadangan Basis Data Global
│   ├── [Admin] Verifikasi & Pemulihan Cadangan (Restore)
│   └── [Admin] Ubah Status Operasional Kelas
├── 06. Sistem & Telemetri Server
│   ├── Health Check HTTP Server
│   └── Telemetri WhatsApp & Kelas Aktif
└── 07. Legacy Shim (Kompatibilitas Bot Lama)
    ├── [Legacy] Daftar Kelas Lama
    ├── [Legacy] Jadwal Kuliah
    └── [Legacy] Daftar Tugas
```

---

## 5. Skenario Pengujian Bertahap

Untuk memverifikasi alur kerja aplikasi secara menyeluruh dari hulu ke hilir, jalankan request sesuai urutan skenario berikut:

### Skenario A: Mahasiswa Mengakses Portal Kelas (Tanpa Login)
Mahasiswa tidak memerlukan otentikasi (read-only):
1. Buka folder **`02. Mahasiswa (Portal Publik)`**.
2. Jalankan `[Portal] Ringkasan Hari Ini (Summary)` $\rightarrow$ Verifikasi respon HTTP 200 memuat `now_event`, `next_event`, dan `nearest_tasks`.
3. Jalankan `[Portal] Jadwal Kuliah Efektif` $\rightarrow$ Verifikasi daftar jadwal memuat gabungan pola mingguan reguler dan jadwal pengganti.
4. Jalankan `[Portal] Daftar Tugas Terbit` $\rightarrow$ Verifikasi hanya tugas berstatus `PUBLISHED` yang ditampilkan.

---

### Skenario B: Tata Kelola Tugas Dua Lapis (PJ Terbitkan $\rightarrow$ KM Review $\rightarrow$ Mahasiswa Baca)
Menguji arsitektur penerbitan instan oleh PJ dan pengawasan retrospektif oleh KM:

1. **Login sebagai PJ**:
   * Buka folder **`01. Otentikasi & Akun`** $\rightarrow$ Jalankan `[Auth] Login PJ (Penanggung Jawab)`.
   * *Variabel `{{tokenPJ}}` dan `{{token}}` otomatis terisi token PJ.*
2. **PJ Terbitkan Tugas Baru**:
   * Buka folder **`03. Penanggung Jawab Mata Kuliah (PJ)`**.
   * Jalankan `[PJ] Terbitkan Tugas (Publish)`.
   * Respon HTTP 201 Created. Tugas terbit dengan `publication_status: "PUBLISHED"` dan `review_state: "NOT_REVIEWED"`. ID tugas otomatis tersimpan ke `{{currentTaskId}}`.
3. **Mahasiswa Langsung Melihat Tugas**:
   * Buka folder **`02. Mahasiswa (Portal Publik)`** $\rightarrow$ Jalankan `[Portal] Daftar Tugas Terbit`.
   * Tugas baru langsung tampil tanpa menunggu persetujuan KM.
4. **KM Meninjau & Menyetujui Tugas**:
   * Buka folder **`01. Otentikasi & Akun`** $\rightarrow$ Jalankan `[Auth] Login KM (Ketua Murid)`.
   * *Variabel `{{tokenKM}}` dan `{{token}}` otomatis berganti ke token KM.*
   * Buka folder **`04. Ketua Murid`** $\rightarrow$ Jalankan `[KM] Lihat Antrean Tugas Butuh Review`.
   * Jalankan `[KM] Setujui Tugas (Approve Review)` $\rightarrow$ Respon HTTP 200 OK, status tugas berubah menjadi `review_state: "APPROVED"`.

---

### Skenario C: Penjadwalan Kuliah Pengganti & Konfirmasi Ruangan
1. Pastikan Anda telah menjalankan `[Auth] Login PJ`.
2. Buka folder **`03. Penanggung Jawab Mata Kuliah (PJ)`**.
3. Jalankan `[PJ] Ajukan Draf Kuliah Pengganti` $\rightarrow$ Respon HTTP 201 Created, ID kegiatan perkuliahan tersimpan ke `{{currentEventId}}`.
4. Jalankan `[PJ] Cari Kandidat Ruangan Kosong` $\rightarrow$ Server mencari seluruh ruangan aktif yang tidak bentrok pada rentang waktu yang diajukan.
5. Jalankan `[PJ] Periksa Bentrok Jadwal (Preview Conflict)` $\rightarrow$ Backend mensimulasikan jadwal dan mengonfirmasi apakah ada tabrakan dengan jadwal kelas lain.
6. Jalankan `[PJ] Terbitkan Jadwal Pengganti (Publish)` $\rightarrow$ Jadwal terbit resmi ke portal dan memicu antrean notifikasi WhatsApp.
7. Buka folder **`04. Ketua Murid`** $\rightarrow$ Jalankan `[KM] Catat Konfirmasi Ruangan TU`:
   * Mencatat nomor ruangan, status `CONFIRMED`, dan nama petugas TU yang menyetujui peminjaman ruangan.

---

### Skenario D: Penegakan Otorisasi & Keamanan (RBAC)
Uji keandalan aturan otorisasi server:
1. **Akses Tanpa Token**:
   * Buka salah satu request di folder PJ atau KM.
   * Kosongkan isi header `Authorization` $\rightarrow$ Kirim request.
   * **Hasil yang diharapkan:** HTTP `401 Unauthorized` dengan kode error `"UNAUTHENTICATED"`.
2. **PJ Mengubah Data di Luar Wewenang**:
   * Login sebagai **PJ**.
   * Coba jalankan `[KM] Aktivasi Semester Baru` atau `[KM] Setujui Tugas`.
   * **Hasil yang diharapkan:** HTTP `403 Forbidden` dengan kode error `"FORBIDDEN"`.
3. **Pencegahan Brute-Force Login**:
   * Kirim request `[Auth] Login KM` dengan password salah sebanyak 5 kali berturut-turut.
   * **Hasil yang diharapkan:** Pada percobaan ke-6, server menolak dengan HTTP `429 Too Many Requests`.

---

### Skenario E: Tata Kelola Kurikulum & Cadangan oleh KM
1. Login sebagai **KM**.
2. Buka folder **`04. Ketua Murid`**.
3. Jalankan `[KM] Validasi Batch Kurikulum (Import Validate)` $\rightarrow$ Backend memvalidasi integritas data matkul, dosen pengampu, dan pola jadwal. ID batch tersimpan ke `{{currentBatchId}}`.
4. Jalankan `[KM] Terapkan Batch Kurikulum (Import Apply)` $\rightarrow$ Seluruh data offering dan jadwal tersimpan secara atomik ke database.
5. Jalankan `[KM] Buat Cadangan Basis Data (Backup Kelas)` $\rightarrow$ Backend menghasilkan snapshot SQLite terisolasi dengan checksum SHA-256. ID cadangan tersimpan ke `{{currentBackupId}}`.
6. Jalankan `[KM] Aktivasi Semester Baru` $\rightarrow$ Semester baru menjadi aktif dan semester lama diarsipkan secara aman.
7. Jalankan `[KM] Log Audit Perubahan Kelas` $\rightarrow$ Memastikan seluruh aksi di atas tercatat rapi pada audit log kelas.

---

### Skenario F: Operasi Global & Pemulihan oleh System Administrator
1. Buka folder **`01. Otentikasi & Akun`** $\rightarrow$ Jalankan `[Auth] Login Admin (System Administrator)`.
   * *Variabel `{{tokenAdmin}}` otomatis terisi token Admin.*
2. Buka folder **`05. Administrator Sistem (Operasi Global & Pemulihan)`**:
3. Jalankan `[Admin] Telemetri Global Sistem` $\rightarrow$ Memantau metrik total kelas, pengguna, tugas, jadwal, antrean pesan, dan status bot WA.
4. Jalankan `[Admin] Penangguhan Akun Pengguna (Suspend)` $\rightarrow$ Menangguhkan akun bermasalah dan mencabut semua token sesi aktifnya secara instan.
5. Jalankan `[Admin] Pemulihan Akun Pengguna (Recover)` $\rightarrow$ Mengaktifkan kembali akun dan mereset kata sandi baru.
6. Jalankan `[Admin] Antrean Siaran WhatsApp Global` $\rightarrow$ Memantau pesan siaran seluruh kelas.
7. Jalankan `[Admin] Log Audit Global (Lintas Seluruh Kelas)` $\rightarrow$ Melihat seluruh rekaman audit sistem dari semua kelas tanpa filter.
8. Jalankan `[Admin] Verifikasi & Pemulihan Cadangan (Restore)` $\rightarrow$ Memverifikasi integritas checksum SHA-256 fisik berkas di disk sebelum proses pemulihan bencana (*disaster recovery*).

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
    "message": "Deskripsi kesalahan teknis yang manusiawi",
    "details": { ... }
  }
}
```

Daftar Kode Error Resmi:
* `UNAUTHENTICATED` (401): Token tidak ada, tidak valid, atau kedaluwarsa.
* `FORBIDDEN` (403): Pengguna terotentikasi tetapi tidak memiliki wewenang pada cakupan target.
* `NOT_FOUND` (404): Entitas atau endpoint tidak ditemukan.
* `VALIDATION` (422 / 400): Parameter atau input data tidak memenuhi batasan logika bisnis.
* `VERSION_CONFLICT` (409): Konflik konkurensi (optimistic locking) karena data di server telah diperbarui oleh pengurus lain.
* `TOO_MANY_REQUESTS` (429): Frekuensi request melebihi ambang batas rate limit.
