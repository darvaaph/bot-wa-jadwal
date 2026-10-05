# ADR-0009 — Pemulihan Data Akademik Bercakupan

- Status: Accepted
- Tanggal: 2026-10-02
- Acuan: `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-OPS-002 dan `docs/product/ACCESS_CONTROL.md`.

## Keputusan

Backup baru memakai paket JSON versi 2 dengan checksum SHA-256 untuk satu kelas atau satu semester kelas. Paket memuat semester, offering, dosen penawaran, pola jadwal, teaching event milik cakupan, partisipasi, konfirmasi ruangan, tugas, review, dan materi. Referensi master disimpan untuk pemeriksaan, tetapi restore tidak menulis master global, akun, izin, kanal WhatsApp, audit, atau riwayat pesan.

KM membuat permintaan backup; System Admin mengeksekusinya. Admin memuat pratinjau, meninjau jumlah data dan teaching event lintas kelas, mengisi alasan, dan mengirim token pratinjau saat eksekusi. Perubahan pada data antara pratinjau dan eksekusi menuntut pratinjau baru. Event lintas kelas menahan restore sampai keterkaitannya diselesaikan.

Eksekusi menghentikan sementara mutasi HTTP dan worker pesan, membuat paket titik pemulihan, lalu memulihkan data akademik dalam transaksi. Data baru setelah paket disembunyikan dari tampilan aktif; relasi dosen penawaran dan konfirmasi ruangan memakai `superseded_at` agar jejaknya tetap tersimpan. Pesan lama tetap sebagai riwayat, pesan tertunda yang usang dibatalkan, dan satu koreksi baru dibuat bila informasi terbit berubah. Galat membuat transaksi rollback dan gerbang mutasi dibuka kembali.

`POST /api/v1/restores` dari ADR-0008 tetap verify-only untuk kompatibilitas arsip v1. Paket v1 tidak dapat dipakai pada endpoint eksekusi baru. Status `VERIFIED` menunjukkan integritas berkas sudah diperiksa, bukan data sudah dipulihkan.
