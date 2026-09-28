# ADR-0001 — Scope API v1: Auth + Jadwal Efektif + Tugas Publish Minimal

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: `FR-*` v3 menuntut ~8 domain (auth, kelas, semester, jadwal, tugas, materi, notifikasi, ruangan, audit). Membangun semuanya sekaligus memblokir frontend berminggu-minggu. Opsi: (A) full MVP, (B) slice minimal, (C) rapikan lama tanpa auth.
- Keputusan: Pilih (B). API v1 = auth pengurus + portal baca + pola jadwal + teaching event draf/publish/revoke + tugas draf/publish/review + materi baca. Ruangan kandidat, antrean notifikasi admin, audit global, backup/restore = v1.1+.
- Konsekuensi: Frontend Portal + Kelola Tugas/Jadwal bisa jalan duluan. Endpoint `FR-ROOM/FR-NOTIF-004/FR-AUDIT-002/FR-OPS-002` harus balas `501 + kode fitur` yang jelas, bukan `404` generik, agar frontend tidak menebak.
