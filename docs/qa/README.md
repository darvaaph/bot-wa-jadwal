# Panduan Uji Happy Path Bot Jadwal

Dokumen ini untuk QA. Bahasanya dibuat sederhana tanpa kode teknis.

## 1. Tujuan dan batas

Tujuan tahap ini adalah memastikan alur utama berjalan dari buat kelas sampai portal tampil dan koreksi terkirim.

Batas tahap ini:

- Termasuk: buat kelas, undang KM, buat dan aktifkan semester, undang PJ, buat dan terbitkan tugas, setuju dan koreksi tugas, jadwal rutin, jadwal pengganti, cabut jadwal, portal mahasiswa, pesan WA, uji gangguan ringan.
- Belum termasuk: backup dan restore, kasus keamanan peran yang dalam, cek tampilan HP.

## 2. Kelas uji dan pembagian

| QA | Kelas | Fokus |
|---|---|---|
| Fatih | D4-TI-2025-A | Fondasi: kelas, KM, semester, undangan PJ |
| Jaya | D4-TI-2025-B | Isi: tugas, jadwal, portal, WA asli, gangguan |

Aturan:

- Tiap QA bangun kelasnya sendiri dari awal.
- Urutan wajib dari atas ke bawah karena PJ butuh semester dan undangan yang sudah jadi.
- Dua kelas berbeda supaya tidak saling menimpa.
- Perangkat yang dipakai adalah laptop.
- Pesan WA asli hanya dikirim ke grup khusus uji, bukan grup kelas asli.

## 3. Siapkan sebelum mulai

1. Atur ulang data contoh:

```bash
go run ./cmd/seed
```

2. Jalankan server:

```bash
go run ./cmd/bot -web-only
```

3. Buka alamat uji di laptop, misal `http://localhost:8080`.
4. Siapkan akun System Admin untuk mulai bangun kelas.
5. Siapkan satu grup WA khusus uji untuk bagian pesan.

## 4. Daftar cek Fatih, kelas D4-TI-2025-A

Fatih catat detail bagian ini. Bagian isi cukup dilewatkan sekali agar rantai jalan.

| No | Yang dilakukan | Yang dicek | Foto |
|---|---|---|---|
| F1 | Masuk sebagai System Admin lalu buat kelas D4-TI-2025-A dengan prodi, angkatan, rombel, nama tampilan, dan mode akses. | Kelas muncul di daftar. Nama yang sama ditolak. | Foto form dan daftar kelas. |
| F2 | Undang KM untuk D4-TI-2025-A. Masukkan nama dan nomor calon KM lalu buka tautan undangan dan buat akun. | Nama tercatat sebagai pengurus kelas. Tautan tidak bisa dipakai dua kali. | Foto halaman undangan dan daftar pengurus. |
| F3 | Buat semester draf di D4-TI-2025-A dengan tanggal mulai dan selesai. Isi mata kuliah lewat impor file, salin semester lama, atau isi manual. Buka pratinjau lalu aktifkan. | Jumlah mata kuliah dan peringatan bentrok tampil di pratinjau. Semester lama jadi arsip baca saja. Data draf belum tampil di portal. | Foto pratinjau dan status semester. |
| F4 | Undang PJ untuk satu mata kuliah di D4-TI-2025-A. Calon PJ buka tautan dan buat kata sandi. | PJ hanya lihat mata kuliah itu. PJ tidak bisa ganti kelas atau mata kuliah dari halaman undangan. | Foto batas akses dan dashboard PJ. |
| F5 | Lewatkan rantai sekali. Buat dan terbitkan satu tugas, buat satu jadwal pengganti, lalu buka portal D4-TI-2025-A. | Portal tampil. | Satu foto portal sebagai bukti rantai jalan. |
| F6 | Kumpulkan laporan fondasi. | Laporan ada langkah, hasil, dan foto. | - |

## 5. Daftar cek Jaya, kelas D4-TI-2025-B

Jaya bangun fondasi D4-TI-2025-B sekadar lewat tanpa foto detail, lalu catat detail bagian isi.

| No | Yang dilakukan | Yang dicek | Foto |
|---|---|---|---|
| J0 | Buat kelas D4-TI-2025-B, KM, semester aktif, dan PJ sekadar lewat. | Rantai fondasi jalan. | Tidak wajib. |
| J1 | Masuk sebagai PJ lalu buat tugas draf dengan judul, instruksi, tenggat, jenis, dan tempat pengumpulan. | Draf tidak muncul di portal. | Foto halaman tugas dan portal. |
| J2 | Buka pratinjau tugas lalu terbitkan. | Tugas muncul di portal dengan isi lengkap. | Foto sebelum dan sesudah terbit. |
| J3 | Masuk sebagai KM lalu setujui satu tugas. Uji satu koreksi dengan catatan. | Tugas koreksi turun jadi draf dan hilang dari portal. Riwayat catat pelaku. | Foto status dan riwayat. |
| J4 | Tambah jadwal rutin dengan hari, jam mulai, durasi, ruangan, dan dosen. | Jam selesai terhitung otomatis. Bentrok beri peringatan sebelum simpan. | Foto form dan peringatan. |
| J5 | Pilih satu sesi lalu buat jadwal pengganti dengan tanggal, jam baru, ruangan, dan keterangan. Buka layar perbandingan lama dan baru lalu terbitkan. | Portal beri label pengganti. | Foto perbandingan dan portal. |
| J6 | Sambungkan bot asli ke grup khusus uji untuk D4-TI-2025-B. | Pesan perubahan masuk satu kali. | Foto pesan dan status antrean. |
| J7 | Masuk sebagai KM lalu cabut jadwal yang salah dengan alasan. | Portal kembali ke jadwal semula. Riwayat lama tetap ada. Pesan koreksi masuk. | Foto portal dan riwayat. |
| J8 | Buka portal D4-TI-2025-B tanpa login. Cek ringkasan hari ini, jadwal, tugas, mata kuliah, riwayat perubahan, dan arsip. | Tidak ada tombol ubah. Tugas draf tidak bocor. | Foto tiap tab. |
| J9 | Uji gangguan. Terbitkan satu tugas saat bot mati. Cek portal lalu nyalakan bot lagi. Uji dua pengurus ubah data yang sama. | Portal tetap tampil saat bot mati dan status pesan tulis menunggu. Pesan terkirim satu kali setelah bot nyala. Simpanan kedua saat edit bersamaan ditolak dan ketikan tidak hilang. | Foto tiap hasil. |
| J10 | Kumpulkan laporan isi. | Laporan ada langkah, hasil, dan foto. | - |

## 6. Tukar baca sebelum tutup

- Fatih buka portal D4-TI-2025-B dan ulangi cek tugas serta jadwal dari sisi mahasiswa.
- Jaya uji satu undangan yang dicabut atau kedaluwarsa di D4-TI-2025-A dan pastikan ditolak.
- Tulis temuan tambahan di laporan masing-masing.

## 7. Format tiap temuan

Tulis dengan bentuk ini:

```markdown
Judul: ringkas, sebut halaman dan kejadian
Langkah ulangi:
1. Masuk sebagai ...
2. Buka ...
3. Isi ... lalu simpan
Hasil yang diharapkan: ...
Hasil yang terjadi: ...
Foto: lampirkan foto layar laptop
Kelas: D4-TI-2025-A atau D4-TI-2025-B
```

## 8. Definisi selesai tahap ini

Tahap ini selesai saat:

- Laporan Fatih dan Jaya terkumpul.
- Tiap langkah ada status lulus atau gagal plus foto.
- Tiap bug ada langkah ulangi dan foto.

Laporan yang terkumpul dipakai untuk putuskan langkah berikutnya, walau masih ada yang gagal.
