# Data Contoh untuk Prototype

## Status Dokumen

| Atribut | Nilai |
|---|---|
| Versi | 1.0.0 |
| Status | Data fiktif untuk eksplorasi UI/UX |
| Tanggal acuan | Kamis, 24 September 2026 |
| Cakupan | Portal mahasiswa, area pengelola, tugas, dan jadwal |

Semua isi dokumen ini adalah data contoh. Nama, kelas, mata kuliah, ruangan, dan kejadian tidak mewakili kampus atau orang nyata. Tampilkan label `Data contoh` pada frame yang memakai data ini. Jangan tampilkan label tersebut berulang pada setiap kartu atau baris.

## 1. Pengguna dan Pilihan Akses

| Field | Nilai contoh |
|---|---|
| Nama tampilan | Pengguna Contoh |
| Inisial | PC |
| Akses utama | Ketua Murid, D4 TI 2024 A |
| Akses lain | PJ Pemrograman Lanjut, D4 TI 2024 A |
| Semester aktif | Semester 3, 2026/2027 Ganjil |

Pilihan pada layar `Pilih Kelas dan Peran`:

1. `Ketua Murid` untuk `D4 TI 2024 A`, Semester 3.
2. `PJ Mata Kuliah` untuk `Pemrograman Lanjut`, D4 TI 2024 A, Semester 3.

Ringkasan yang selalu terlihat setelah masuk:

```text
Sedang mengelola
D4 TI 2024 A · Semester 3
Ketua Murid
[Pindah kelas atau peran]
```

## 2. Mata Kuliah

| Kode contoh | Mata kuliah | Jenis | PJ |
|---|---|---|---|
| PLP-T | Pemrograman Lanjut | Teori | Pengguna Contoh A |
| PLP-P | Pemrograman Lanjut | Praktik | Pengguna Contoh B |
| SBD-T | Sistem Basis Data | Teori | Pengguna Contoh C |
| ALIN-T | Aljabar Linear | Teori | Pengguna Contoh D |
| OS-T | Sistem Operasi | Teori | Pengguna Contoh E |

Nama pengguna di atas adalah placeholder yang sengaja diberi kata `Contoh`. Jangan menggantinya dengan nama orang yang tampak nyata.

## 3. Jadwal Reguler

| Hari | Waktu | Mata kuliah | Jenis | Pengajar | Ruangan | Berlaku |
|---|---|---|---|---|---|---|
| Senin | 08.00-09.40 WIB | Pemrograman Lanjut | Teori | Dosen Contoh A | R. Teori 3 | 26 Agu-18 Des 2026 |
| Selasa | 10.00-11.40 WIB | Sistem Basis Data | Teori | Dosen Contoh B | R. Teori 2 | 26 Agu-18 Des 2026 |
| Rabu | 13.00-15.30 WIB | Pemrograman Lanjut | Praktik | Dosen Contoh C | Lab 301 | 26 Agu-18 Des 2026 |
| Kamis | 08.00-09.40 WIB | Aljabar Linear | Teori | Dosen Contoh D | R. Teori 1 | 26 Agu-18 Des 2026 |
| Jumat | 09.00-10.40 WIB | Sistem Operasi | Teori | Dosen Contoh E | R. Teori 4 | 26 Agu-18 Des 2026 |

Gunakan data ini untuk keadaan berisi. Untuk keadaan kosong, jangan hapus identitas kelas dan semester. Tampilkan penjelasan serta tindakan yang sesuai dengan hak akses pengguna.

## 4. Perubahan Jadwal

### Perubahan yang sudah terbit

| Field | Nilai contoh |
|---|---|
| Jenis | Kelas Pengganti |
| Mata kuliah | Sistem Basis Data, Teori |
| Jadwal semula | Selasa, 29 September 2026 · 10.00-11.40 WIB · R. Teori 2 |
| Jadwal baru | Rabu, 30 September 2026 · 15.30-17.10 WIB · Lab 302 |
| Alasan | Dosen menghadiri kegiatan program studi |
| Status publikasi | Terbit |
| Status WhatsApp | Menunggu dikirim |

### Draf kelas tambahan

| Field | Nilai contoh |
|---|---|
| Jenis | Kelas Tambahan |
| Mata kuliah | Pemrograman Lanjut, Praktik |
| Waktu | Sabtu, 3 Oktober 2026 · 09.00-11.30 WIB |
| Ruangan | Lab 301, belum dikonfirmasi TU |
| Status publikasi | Draf |

### Contoh konflik

```text
Lab 301 tercatat dipakai D4 TI 2024 B pada Sabtu, 3 Oktober 2026 pukul 09.30-11.00 WIB.
Pilih waktu atau ruangan lain sebelum menerbitkan perubahan ini.
```

### Contoh hasil pencarian ruangan

| Ruangan | Status data internal | Pembaruan | Tindakan berikutnya |
|---|---|---|---|
| Lab 302 | Tidak tercatat dipakai | 24 Sep 2026, 13.20 WIB | Konfirmasi ke TU |
| R. Teori 5 | Tidak tercatat dipakai | 24 Sep 2026, 13.20 WIB | Konfirmasi ke TU |

Jangan menulis `Tersedia` sebelum pengurus mencatat konfirmasi TU. Gunakan `Tidak tercatat dipakai` agar batas data sistem tetap jelas.

## 5. Tugas

| Mata kuliah | Judul | Tenggat | Publikasi | Pemeriksaan KM | Kondisi |
|---|---|---|---|---|---|
| Sistem Basis Data | Normalisasi Skema Perpustakaan | Jumat, 25 Sep 2026 · 21.00 WIB | Terbit | Perlu diperiksa | Aktif |
| Pemrograman Lanjut | Refaktor Modul Inventaris | Senin, 28 Sep 2026 · 20.00 WIB | Draf | Belum perlu diperiksa | Draf |
| Aljabar Linear | Latihan Ruang Vektor | Kamis, 1 Okt 2026 · 18.00 WIB | Terbit | Sudah diperiksa | Aktif |
| Sistem Operasi | Ringkasan Manajemen Proses | Rabu, 23 Sep 2026 · 19.00 WIB | Terbit | Sudah diperiksa | Terlewat |

Detail tugas `Normalisasi Skema Perpustakaan`:

| Field | Nilai contoh |
|---|---|
| Instruksi | Ubah tabel transaksi ke bentuk normal ketiga. Sertakan alasan untuk setiap pemisahan tabel. |
| Jenis tugas | Individu |
| Tempat pengumpulan | Tautan LMS kelas |
| Versi | Versi 3 |
| Diterbitkan oleh | Pengguna Contoh C, PJ Mata Kuliah |
| Waktu terbit | Kamis, 24 September 2026 · 10.15 WIB |

## 6. Pesan Keadaan Layar

| Keadaan | Judul | Penjelasan | Tindakan |
|---|---|---|---|
| Belum ada tugas | Belum ada tugas untuk semester ini | Tugas yang diterbitkan akan muncul di sini. | `Tambah Tugas` untuk pengurus, tanpa tombol untuk mahasiswa |
| Filter tanpa hasil | Tidak ada tugas yang cocok | Tidak ada tugas dengan filter yang dipilih. | `Hapus Semua Filter` |
| Jadwal belum dibuat | Jadwal reguler belum dibuat | Tambahkan jadwal agar mahasiswa dapat melihat jadwal mingguannya. | `Tambah Jadwal` jika diizinkan |
| Memuat jadwal | Memuat jadwal kelas | Jadwal D4 TI 2024 A sedang dimuat. | Tidak ada tindakan |
| Gagal memuat | Jadwal belum dapat dimuat | Periksa koneksi lalu coba lagi. | `Coba Lagi` |
| Tidak memiliki akses | Anda tidak dapat membuka halaman ini | Akun ini tidak memiliki izin untuk kelas atau peran yang dipilih. | `Kembali` dan `Pindah kelas atau peran` |
| Sesi berakhir | Silakan masuk kembali | Isian Anda tetap tersimpan di perangkat ini. Masuk kembali untuk melanjutkan. | `Masuk Kembali` |
| WhatsApp tertunda | Jadwal sudah terbit di web | Pesan WhatsApp belum terkirim dan akan dicoba lagi. | `Lihat Status Pesan` |

## 7. Aturan Pemakaian

- Isi layar daftar, detail, formulir, preview, riwayat, dan notifikasi dengan data contoh yang relevan. Jangan menghasilkan frame utama yang seluruhnya kosong.
- Keadaan kosong tetap harus dibuat sebagai variasi terpisah untuk menguji panduan pengguna pertama kali.
- Jangan membuat statistik jumlah mahasiswa, tingkat penyelesaian, performa kelas, testimoni, foto profil, nama kampus, atau nama dosen.
- Gunakan tanggal acuan dokumen agar label seperti `Hari Ini` atau `Terlewat` konsisten antarframe.
- Gunakan status internal hanya dalam anotasi desain. Antarmuka memakai label Bahasa Indonesia dari `PENCIL_CONTEXT.md`.
