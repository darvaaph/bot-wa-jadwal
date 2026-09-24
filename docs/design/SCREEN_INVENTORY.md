# Inventaris Layar pen.dev

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.2.0 |
| Status | Approved untuk perencanaan desain |
| Pemilik | Tim Bot Jadwal |
| Terakhir diperbarui | 24 September 2026 |
| Panduan global | [Panduan Desain pen.dev](PENCIL_CONTEXT.md) |
| Data contoh | [Data Contoh untuk Prototype](PROTOTYPE_DATA.md) |

Dokumen ini membagi desain menjadi batch yang dapat dikerjakan dan diperiksa secara terpisah. Satu baris dapat menghasilkan beberapa frame untuk peran, ukuran layar, atau keadaan berbeda.

## 1. Prioritas

- `P0`: diperlukan untuk vertical slice pertama dan validasi fondasi aplikasi.
- `P1`: diperlukan untuk MVP lengkap.
- `P2`: fase lanjutan atau operasi yang jarang digunakan.

## 2. Fondasi dan Akses Akun

| ID | Layar | Aktor | Prioritas | Variasi utama |
|---|---|---|:---:|---|
| SCR-AUTH-001 | Login | PJ, KM, System Admin | P0 | Default, kredensial gagal, diblokir sementara, loading |
| SCR-AUTH-002 | Pilih Kelas dan Peran | Pengurus dengan beberapa akses | P0 | Satu kelas, beberapa kelas, PJ dan KM, System Admin |
| SCR-AUTH-003 | Aktivasi Undangan | Calon pengurus | P1 | Akun baru, akun lama, token kedaluwarsa |
| SCR-AUTH-004 | Lupa dan Atur Ulang Kata Sandi | Pengurus | P1 | Permintaan terkirim, token tidak valid, berhasil |
| SCR-SHELL-001 | Kerangka Area Pengelola | PJ, KM | P0 | Navigasi bawah mobile, sidebar desktop, pindah kelas atau peran |
| SCR-SHELL-002 | App Shell System Admin | System Admin | P1 | Desktop utama, mobile inspeksi |

## 3. Portal Mahasiswa

| ID | Layar | Prioritas | Variasi utama |
|---|---|:---:|---|
| SCR-PORTAL-001 | Masukkan Kode Kelas | P0 | Default, kode salah, dibatasi sementara |
| SCR-PORTAL-002 | Ringkasan Kelas | P0 | Ada jadwal hari ini, tidak ada jadwal, perubahan terbaru |
| SCR-PORTAL-003 | Jadwal | P1 | Harian, mingguan, sesi diganti, sesi dibatalkan |
| SCR-PORTAL-004 | Daftar Tugas | P0 | Hari Ini, Minggu Ini, Mendatang, Terlewat, filter |
| SCR-PORTAL-005 | Detail Tugas | P0 | Aktif, selesai, terlewat, publikasi dicabut |
| SCR-PORTAL-006 | Mata Kuliah dan Materi | P1 | Materi umum dan materi per mata kuliah kelas |
| SCR-PORTAL-007 | Perubahan Terbaru | P1 | Jadwal, tugas, dan koreksi |
| SCR-PORTAL-008 | Arsip Semester | P1 | Daftar semester dan mode hanya-baca |

## 4. Area Pengelola

| ID | Layar | Aktor | Prioritas | Variasi utama |
|---|---|---|:---:|---|
| SCR-APP-001 | Ringkasan PJ | PJ | P0 | Draf, publikasi terbaru, mata kuliah penugasan |
| SCR-APP-002 | Ringkasan KM | KM | P0 | Perlu diperiksa, undangan kelas peserta, konflik, notifikasi gagal |
| SCR-TASK-001 | Daftar Tugas Pengelola | PJ, KM | P0 | Aktif, Draf, Perlu diperiksa KM, Selesai, Terlewat, Arsip |
| SCR-TASK-002 | Form Tugas | PJ, KM | P0 | Tambah, ubah, validasi gagal, ada versi lebih baru |
| SCR-TASK-003 | Pratinjau Publikasi Tugas | PJ, KM | P0 | Publikasi PJ dan publikasi KM |
| SCR-TASK-004 | Detail dan Riwayat Tugas | PJ, KM | P0 | Belum diperiksa, disetujui, koreksi, dicabut |
| SCR-TASK-005 | Antrean Pemeriksaan KM | KM | P0 | Daftar, kosong, versi berubah |
| SCR-TASK-006 | Dialog Hasil Pemeriksaan | KM | P0 | Setujui, minta koreksi, batalkan |
| SCR-SCH-001 | Pola Jadwal Reguler | PJ, KM | P1 | Daftar hari, filter, tambah, versi baru |
| SCR-SCH-002 | Form Perubahan Jadwal | PJ, KM | P1 | Pengganti, tambahan, libur, pembatalan sesi |
| SCR-SCH-003 | Pratinjau dan Konflik Jadwal | PJ, KM | P1 | Tanpa konflik, konflik pemblokir, pengecualian beralasan |
| SCR-SCH-004 | Detail Perubahan Jadwal | PJ, KM | P1 | Draf, terbit, dicabut, status notifikasi |
| SCR-SCH-005 | Partisipasi Lintas Kelas | KM | P1 | Pending, diterima, ditolak, dilepas |
| SCR-ROOM-001 | Kandidat dan Konfirmasi Ruangan | PJ, KM | P2 | Kandidat, pending TU, confirmed, rejected |
| SCR-MAT-001 | Materi | PJ, KM | P2 | Umum kelas, per mata kuliah kelas, arsip |
| SCR-MEMBER-001 | Anggota dan Peran | KM | P1 | PJ aktif, undangan, ditangguhkan, dicabut |
| SCR-SEM-001 | Daftar Semester | PJ, KM | P1 | Aktif, draf, arsip |
| SCR-SEM-002 | Persiapan Semester | KM | P1 | Input manual, salin, impor JSON, error impor |
| SCR-AUDIT-001 | Riwayat Perubahan | PJ, KM | P1 | Filter pelaku, objek, tindakan, waktu |
| SCR-SET-001 | Pengaturan Kelas | KM | P1 | Mode portal, rotasi kode, waktu pengingat |

## 5. System Admin

| ID | Layar | Prioritas | Variasi utama |
|---|---|:---:|---|
| SCR-ADMIN-001 | Ringkasan Sistem | P1 | Normal, antrean gagal, bot offline |
| SCR-ADMIN-002 | Kelas dan Detail Kelas | P1 | Daftar, buat, mode dukungan |
| SCR-ADMIN-003 | Pengguna dan Penugasan | P1 | Aktif, ditangguhkan, pemulihan |
| SCR-ADMIN-004 | Master Mata Kuliah | P2 | Aktif dan nonaktif |
| SCR-ADMIN-005 | Master Ruangan | P2 | Aktif, nonaktif, usulan koreksi |
| SCR-ADMIN-006 | Antrean Notifikasi | P1 | Pending, Processing, Sent, Failed |
| SCR-ADMIN-007 | Backup dan Pemulihan | P2 | Pilih scope, validasi, konfirmasi restore |
| SCR-ADMIN-008 | Audit Global | P2 | Filter kelas, pelaku, tindakan, waktu |
| SCR-ADMIN-009 | Status Sistem dan WhatsApp | P1 | Terhubung, terputus, proses sambung ulang |

## 6. Urutan Batch Desain

| Batch | Isi | Tujuan review |
|---|---|---|
| 0 | Fondasi desain, SCR-SHELL-001, SCR-AUTH-001, SCR-AUTH-002 | Menetapkan kerangka, ringkasan kelas dan peran, breakpoint, serta komponen dasar |
| 1 | SCR-TASK-001 sampai SCR-TASK-006, SCR-PORTAL-004, SCR-PORTAL-005 | Menguji satu siklus lengkap PJ, KM, dan mahasiswa |
| 2 | SCR-PORTAL-001 sampai SCR-PORTAL-003 serta SCR-APP-001 sampai SCR-APP-002 | Memvalidasi pintu masuk dan ringkasan tiap aktor |
| 3 | SCR-PORTAL-003 dan SCR-SCH-001 sampai SCR-SCH-005 | Memvalidasi jadwal efektif mahasiswa, pola reguler, konflik, publikasi, dan lintas kelas |
| 4 | Semester, anggota, audit, dan pengaturan | Melengkapi tata kelola kelas |
| 5 | System Admin dan fitur P2 | Melengkapi operasi global dan fase lanjutan |

Jangan memulai batch berikutnya sebelum struktur navigasi, terminologi, dan komponen yang dipakai ulang pada batch aktif telah diperiksa.

## 7. Syarat Layar Siap Diperiksa

Satu layar dianggap siap diperiksa jika:

1. pengguna, kelas, semester, dan peran yang sedang dikelola terlihat;
2. tindakan utama serta tindakan berisiko dapat dibedakan;
3. mobile dan desktop tersedia;
4. state relevan tersedia;
5. copy memakai Bahasa Indonesia, istilah yang mudah dipahami, dan padanan pada `PENCIL_CONTEXT.md`;
6. variasi izin tidak menampilkan aksi yang dilarang;
7. frame mencantumkan ID layar dan requirement sumber;
8. keputusan visual yang belum final diberi label eksplorasi.

## 8. Changelog

### 1.2.0, 24 September 2026

- Mengganti nama layar dan variasi teknis dengan istilah yang menjelaskan tindakan pengguna.
- Menyelaraskan nama layar perubahan jadwal dengan panduan bahasa antarmuka.

### 1.1.0, 24 September 2026

- Memasukkan tampilan jadwal mahasiswa ke batch desain jadwal.

### 1.0.0, 23 September 2026

- Membagi seluruh ruang aplikasi menjadi inventaris layar yang dapat digenerate bertahap.
- Menetapkan prioritas, variasi state, batch desain, dan definition of done frame.
