# Panduan Pengujian QA (Software Quality Assurance) — Bot Jadwal v3.0

Dokumen ini disusun khusus bagi **Tim QA / Software Tester** untuk melakukan pengujian fungsionalitas, keamanan otorisasi (RBAC), integritas data, dan penanganan kasus batas (*edge cases*) pada REST API Bot Jadwal v3.0.

Koleksi Postman telah dilengkapi dengan kartu instruksi pengujian (*QA Cards*), script assertions otomatis (`pm.test`), dan penangkapan token sesi otomatis di setiap endpoint.

---

## 1. Lingkungan & Akun Uji Coba

### Menjalankan Server Backend
```bash
# 1. Pastikan database bersih dan terisi akun demo
go run ./cmd/seed

# 2. Jalankan server dalam mode API testing
go run ./cmd/bot -web-only
```
*Host pengujian:* `http://localhost:8080`

### Kredensial Uji Coba (Bawaan Database)
| Peran (*Actor*) | Nomor WhatsApp (`identity_key`) | Kata Sandi | Variabel Postman | Batasan Wewenang (*Scope*) |
|---|---|---|---|---|
| **System Administrator** | `+6281111111111` | `password123` | `{{tokenAdmin}}` | **GLOBAL** (Operasi sistem, telemetri, suspend, restore) |
| **Ketua Murid (KM)** | `+6281234567890` | `password123` | `{{tokenKM}}` | **CLASS** (Penuh pada kelas `d4-ti-2024-a`) |
| **PJ Mata Kuliah** | `+6281298765432` | `password123` | `{{tokenPJ}}` | **OFFERING** (Mata kuliah Sistem Operasi Praktikum) |
| **Mahasiswa** | *Tanpa Akun* | *Tanpa Sandi* | *Tanpa Token* | Akses publik hanya-baca via `/api/v1/portal/:slug` |

---

## 2. Cara Kerja Fitur Otomatis Postman bagi QA

Setiap kali Anda menjalankan request login di folder **`01. Otentikasi & Akun`**, Postman secara otomatis:
1. Menangkap token Bearer dari respons JSON.
2. Menyimpan token ke variabel peran masing-masing:
   * `[Auth] Login Admin` $\rightarrow$ mengisi `{{tokenAdmin}}` dan `{{token}}`.
   * `[Auth] Login KM` $\rightarrow$ mengisi `{{tokenKM}}` dan `{{token}}`.
   * `[Auth] Login PJ` $\rightarrow$ mengisi `{{tokenPJ}}` dan `{{token}}`.
3. Menyimpan ID penting saat request mutasi dijalankan:
   * Menerbitkan tugas baru $\rightarrow$ otomatis mengisi `{{currentTaskId}}`.
   * Mengajukan kuliah pengganti $\rightarrow$ otomatis mengisi `{{currentEventId}}`.
   * Memvalidasi batch kurikulum $\rightarrow$ otomatis mengisi `{{currentBatchId}}`.
   * Membuat snapshot backup $\rightarrow$ otomatis mengisi `{{currentBackupId}}`.

> **Keuntungan bagi QA:** Anda cukup mengeksekusi request login ketiga akun demo satu kali di awal sesi pengujian. Setelah itu, seluruh folder pengujian dapat dijalankan tanpa perlu menyalin-tempel token Bearer atau ID secara manual.

---

## 3. Matriks Pengujian QA (QA Test Matrix)

Gunakan matriks berikut sebagai daftar periksa (*checklist*) pengujian formal:

### A. Modul Otentikasi & Sesi Pengurus (`01. Otentikasi & Akun`)
| ID Kasus | Nama Pengujian | Kasus Normal (Happy Path) | Kasus Negatif / Batas (Edge Cases) | Ekspektasi Kode |
|---|---|---|---|---|
| `TC-AUTH-01` | Login Admin Global | Kredensial valid menghasilkan token Admin | Password salah $\rightarrow$ pesan generic tanpa kebocoran nomor | `200` vs `401` |
| `TC-AUTH-02` | Login Ketua Murid (KM) | Kredensial valid menghasilkan token KM | Nomor WA belum terdaftar $\rightarrow$ respon 401 generic | `200` vs `401` |
| `TC-AUTH-03` | Login PJ Mata Kuliah | Kredensial valid menghasilkan token PJ | Percobaan salah 5x berturut-turut $\rightarrow$ Lockout | `200` vs `429` |
| `TC-AUTH-04` | Profil Sesi Aktif (`/me`) | Menampilkan profil, wewenang, dan nama offering aktif | Akses tanpa token Bearer atau token acak | `200` vs `401` |
| `TC-AUTH-05` | Rotasi Sesi (`switch-context`)| Menghasilkan token sesi baru untuk peran lain | Menembak role_assignment_id milik user lain | `200` vs `403` / `404` |
| `TC-AUTH-06` | Pencabutan Sesi (`logout`) | Token dicabut seketika dari database | Token yang sudah di-logout digunakan kembali | `200` vs `401` |

---

### B. Modul Portal Mahasiswa (`02. Mahasiswa - Portal Publik`)
| ID Kasus | Nama Pengujian | Kasus Normal (Happy Path) | Kasus Negatif / Batas (Edge Cases) | Ekspektasi Kode |
|---|---|---|---|---|
| `TC-PORT-01` | Kartu Ringkasan Hari Ini | Menampilkan kuliah saat ini, berikutnya, dan 3 tugas | Slug kelas fiktif `d4-ti-9999-z` | `200` vs `404` |
| `TC-PORT-02` | Jadwal Kuliah Efektif | Menggabungkan pola mingguan & jadwal pengganti | Parameter view tidak valid (misal `view=invalid`) | `200` (default) |
| `TC-PORT-03` | Filter Daftar Tugas | **HANYA** menampilkan tugas berstatus `PUBLISHED` | Tugas draf **TIDAK BOLEH** bocor ke publik | `200` |
| `TC-PORT-04` | Detail Instruksi Tugas | Menampilkan deskripsi pengumpulan dan materi | Menembak ID tugas yang masih berstatus `DRAFT` | `200` vs `404` |
| `TC-PORT-05` | Riwayat Mutasi Jadwal | Menampilkan riwayat pergantian ruang / jam kuliah | Slug kelas salah | `200` vs `404` |
| `TC-PORT-06` | Direktori Materi Kelas | Menampilkan daftar tautan Google Drive / Cloud | Slug kelas salah | `200` vs `404` |

---

### C. Modul Penanggung Jawab Mata Kuliah (`03. PJ Mata Kuliah`)
| ID Kasus | Nama Pengujian | Kasus Normal (Happy Path) | Kasus Negatif / Batas (Edge Cases) | Ekspektasi Kode |
|---|---|---|---|---|
| `TC-PJ-01` | Simpan Draf Tugas | Tugas tersimpan dengan `publication_status: 'DRAFT'` | PJ memilih offering_id mata kuliah orang lain | `201` vs `403` |
| `TC-PJ-02` | Terbitkan Tugas Langsung | Tugas langsung tampil di portal (`NOT_REVIEWED`) | Field `title` dikosongkan | `201` vs `422` |
| `TC-PJ-03` | Update Tugas (Optimistic Lock)| Mengirim versi data yang cocok (`version: 1` $\rightarrow$ `2`) | Mengirim versi usang (`version: 0`) $\rightarrow$ Konflik | `200` vs `409` |
| `TC-PJ-04` | Ajukan Kuliah Pengganti | Jadwal pengganti tersimpan sebagai `DRAFT` | Waktu selesai mendahului waktu mulai (`starts > ends`) | `201` vs `422` |
| `TC-PJ-05` | Cari Ruangan Kosong | Menampilkan ruangan aktif yang bebas bentrok | Format tanggal/jam bukan RFC3339 | `200` vs `422` |
| `TC-PJ-06` | Simulasi Bentrok Jadwal | Membandingkan jam usulan terhadap jadwal kelas lain | Event ID tidak ditemukan | `200` vs `404` |
| `TC-PJ-07` | Terbitkan Jadwal Pengganti | Event terbit resmi (`PUBLISHED`) dengan idempotency | Mengirim payload yang sama berulang kali (Idempoten) | `200`/`201` (aman) |
| `TC-PJ-08` | Tambah Materi Drive | Menautkan link materi ke offering matkul | Format URL tidak valid (bukan URL internet) | `201` vs `422` |

---

### D. Modul Ketua Murid (`04. Ketua Murid - Kontrol Kelas`)
| ID Kasus | Nama Pengujian | Kasus Normal (Happy Path) | Kasus Negatif / Batas (Edge Cases) | Ekspektasi Kode |
|---|---|---|---|---|
| `TC-KM-01` | Antrean Tugas Butuh Review | Menampilkan tugas-tugas terbitan PJ (`NOT_REVIEWED`)| Ditembak menggunakan token PJ | `200` vs `403` |
| `TC-KM-02` | Persetujuan Tugas (Approve) | Tugas berubah menjadi `review_state: 'APPROVED'` | PJ mencoba menyetujui tugasnya sendiri | `200`/`201` vs `403` |
| `TC-KM-03` | Minta Revisi (Tarik ke Draf)| Tugas ditarik dari portal mahasiswa kembali ke DRAFT | Catatan revisi (`note`) dikosongkan | `200`/`201` vs `422` |
| `TC-KM-04` | Pembatalan Tugas Sepihak | Tugas resmi dibatalkan (`REVOKED`) | KM kelas lain mencoba membatalkan tugas kelas ini | `200`/`201` vs `403` |
| `TC-KM-05` | Cabut Jadwal Pengganti | Mengembalikan jam kuliah dan memicu siaran koreksi | Mencabut event yang sudah pernah dibatalkan | `200` vs `409`/`422` |
| `TC-KM-06` | Konfirmasi Ruangan TU | Status `CONFIRMED` otomatis mengupdate `room_id` | Status selain PENDING/CONFIRMED/REJECTED | `201` vs `422` |
| `TC-KM-07` | Validasi Impor Kurikulum | Memvalidasi integritas JSON kurikulum semester | Kode matkul kosong atau jam bentrok di JSON | `200` vs `422` |
| `TC-KM-08` | Terapkan Impor Kurikulum | Eksekusi transaksi atomik offering & pola jadwal | Batch ID belum pernah divalidasi | `200` vs `404`/`422` |
| `TC-KM-09` | Aktivasi Semester Baru | Semester baru aktif, semester lama diarsipkan | Tanpa flag konfirmasi `confirm: true` | `200` vs `400` |
| `TC-KM-10` | Ubah Status Kelas | Status kelas berganti (ACTIVE/INACTIVE/ARCHIVED) | Status fiktif (misal 'DELETED') | `200` vs `422` |
| `TC-KM-11` | Undang PJ Baru | Token undangan acak berdurasi 7 hari terbit | Mengundang dengan peran selain PJ | `201` vs `403`/`422` |
| `TC-KM-12` | Backup Database Kelas | Snapshot SQLite dibuat dengan checksum SHA-256 | Dijalankan oleh akun PJ | `201` vs `403` |
| `TC-KM-13` | Telaah Log Audit Kelas | Hanya menampilkan audit log milik kelas KM | Mencoba mengintip log kelas lain (terisolasi) | `200` (terfilter) |
| `TC-KM-14` | Antrean Notifikasi Kelas | Memantau status pengiriman pesan siaran kelas | Akses tanpa otentikasi | `200` vs `401` |

---

### E. Modul Administrator Sistem (`05. Administrator Sistem - Global`)
| ID Kasus | Nama Pengujian | Kasus Normal (Happy Path) | Kasus Negatif / Batas (Edge Cases) | Ekspektasi Kode |
|---|---|---|---|---|
| `TC-ADM-01`| Telemetri Sistem Global | Total kelas, user, tugas, dan status bot runtime | Dijalankan memakai token KM atau PJ | `200` vs `403` |
| `TC-ADM-02`| Penangguhan Akun (Suspend) | Status user SUSPENDED & seluruh token sesi dicabut | Admin mencoba men-suspend akunnya sendiri | `200` vs `400`/`422` |
| `TC-ADM-03`| Pemulihan Akun (Recover) | Status user ACTIVE kembali & password baru aktif | Password baru kurang dari 8 karakter | `200` vs `422` |
| `TC-ADM-04`| Rekomendasi Ruang Global | Cek ruangan kosong lintas seluruh gedung kampus | Format parameter waktu salah | `200` vs `422` |
| `TC-ADM-05`| Antrean WhatsApp Global | Memantau seluruh antrean pesan broadcast sistem | Akses oleh akun non-admin/KM | `200` vs `403` |
| `TC-ADM-06`| Retry Notifikasi Siaran | Mengulang pengiriman pesan yang FAILED/CANCELLED | Me-retry pesan yang sudah sukses terkirim (SENT) | `200` vs `400`/`409` |
| `TC-ADM-07`| Audit Trail Global | Rekaman mutasi data dari seluruh tenant/kelas | Dijalankan oleh akun PJ | `200` vs `403` |
| `TC-ADM-08`| Backup Global Sistem | Snapshot seluruh database SQLite (SHA-256) | Dijalankan oleh akun PJ | `201` vs `403` |
| `TC-ADM-09`| Verifikasi Restore | Checksum fisik berkas cadangan terverifikasi | Dijalankan oleh KM (Restore eksklusif Admin) | `200` vs `403` |
| `TC-ADM-10`| Kontrol Status Kelas Global| Menyesuaikan status kelas apa pun di server | Slug kelas tidak ditemukan | `200` vs `404` |

---

## 4. Panduan Menjalankan Pengujian Otomatis (*Postman Collection Runner*)

Tim QA dapat menjalankan seluruh suite pengujian atau per folder dalam satu klik:

1. Buka Postman dan pilih koleksi **Bot Jadwal v3.0 - API Testing Suite**.
2. Klik tombol **Run** (atau ikon panah *Run Collection*) di pojok kanan atas.
3. Pilih folder yang ingin diuji (misalnya: `01. Otentikasi & Akun`, `03. PJ Mata Kuliah`, atau seluruh koleksi).
4. Klik tombol **Run Bot Jadwal v3.0**.
5. Postman Runner akan mengeksekusi request secara sekuensial dan menampilkan status:
   * **PASS**: Status kode, format envelope, dan data payload sesuai spesifikasi.
   * **FAIL**: Ditemukan ketidaksesuaian respon backend.

---

## 5. Kamus Kode Respon Resmi Backend

| Kode Status | Kode Error di JSON | Kapan Ini Muncul? |
|---|---|---|
| `200 OK` | - | Request query atau mutasi berhasil diproses. |
| `201 Created` | - | Data baru (tugas, event, backup, undangan) berhasil dibuat. |
| `401 Unauthorized` | `UNAUTHENTICATED` | Token Bearer tidak ada, salah format, atau telah dicabut (logout/suspend). |
| `403 Forbidden` | `FORBIDDEN` | Token valid, namun wewenang pengguna tidak mencakup entitas target. |
| `404 Not Found` | `NOT_FOUND` | Slug kelas, ID tugas, ID event, atau ID semester tidak ada di database. |
| `409 Conflict` | `VERSION_CONFLICT` | Versi data yang diedit lebih rendah daripada data di database (Optimistic Locking). |
| `422 Unprocessable`| `VALIDATION` | Data input tidak memenuhi logika bisnis (jam bentrok, title kosong, dsb). |
| `429 Too Many Req` | `TOO_MANY_REQUESTS` | Frekuensi request melebihi ambang batas rate limit keamanan. |

---

## 6. Format Standar Pelaporan Temuan Bug (Bug Report Template)

Jika Tim QA menemukan kegagalan uji (*assertion failure*) atau perilaku yang menyimpang dari spesifikasi, buat tiket issue di GitHub dengan format berikut:

````markdown
### Deskripsi Masalah
[Jelaskan secara ringkas perilaku error yang terjadi]

### ID Kasus Uji Terkait
- ID: [Misal: TC-PJ-03 / TC-KM-06]
- Endpoint: `[METHOD] /api/v1/[path]`

### Langkah Reproduksi (Steps to Reproduce)
1. Login sebagai [KM / PJ / Admin].
2. Buka request `[...]` pada koleksi Postman.
3. Masukkan payload:
```json
{
  "field": "nilai"
}
```
4. Kirim request.

### Hasil yang Diharapkan (Expected Behavior)
[Kode status dan respon yang seharusnya diterima menurut dokumen ini]

### Hasil Aktual (Actual Behavior)
[Kode status dan pesan error aktual yang diterima dari server]
```json
{
  "status": "error",
  ...
}
```
````
