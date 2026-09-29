# ADR-0007 — Hentikan Penulisan Tugas Legacy

- Status: Accepted; menggantikan bagian write-task pada ADR-0002.
- Tanggal: 2026-09-29
- Konteks: Dashboard aktif dan bot untuk kelas pilot sudah menggunakan model tugas v1. `POST` dan `DELETE /api/tasks` masih menulis tabel lama di `tugas.db`, sehingga perubahan tidak terlihat di portal v1 dan berpotensi menghasilkan dua sumber kebenaran.
- Keputusan: `GET /api/tasks` dipertahankan sebagai shim baca dengan header `Deprecation: true`; ketika database v1 tersedia, data dipetakan dari task `PUBLISHED` yang aktif pada model canonical. Fallback baca tabel lama hanya berlaku bila database v1 tidak tersedia. `POST /api/tasks` dan `DELETE /api/tasks/:id` selalu mengembalikan `410 Gone` dengan pengganti `/api/v1/tasks` dan tidak melakukan mutasi apa pun.
- Konsekuensi: Klien lama tetap dapat membaca selama masa cutover, tetapi harus bermigrasi ke API v1 untuk membuat atau menghapus tugas. Tidak ada dual-write dan tidak ada physical delete melalui HTTP legacy.
