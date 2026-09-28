# ADR-0001 — Scope API v1: Auth + Jadwal Efektif + Tugas Publish Minimal

- Status: Accepted
- Tanggal: 2026-09-28
- Konteks: `FR-*` v3 menuntut ~8 domain (auth, kelas, semester, jadwal, tugas, materi, notifikasi, ruangan, audit). Membangun semuanya sekaligus memblokir frontend berminggu-minggu. Opsi: (A) full MVP, (B) slice minimal, (C) rapikan lama tanpa auth.
- Keputusan: Pilih (B). API v1 = auth pengurus + portal baca + pola jadwal + teaching event draf/publish/revoke + tugas draf/publish/review + materi baca. Ruangan kandidat, antrean notifikasi admin, audit global, backup/restore = v1.1+.
- Konsekuensi: Frontend Portal + Kelola Tugas/Jadwal bisa jalan duluan. Endpoint `FR-ROOM/FR-NOTIF-004/FR-AUDIT-002/FR-OPS-002` awalnya direncanakan balas `501`, namun telah selesai diimplementasikan penuh pada Lapis L2.

### Addendum (2026-09-28)
Seluruh endpoint lanjutan yang sebelumnya dialokasikan untuk v1.1 (`/api/v1/rooms/candidates`, `/api/v1/teaching-events/:id/room-confirmations`, `/api/v1/notifications`, `/api/v1/audit`, `/api/v1/backups`, `/api/v1/restores`, `/api/v1/admin/*`) telah diimplementasikan penuh pada branch `darva-sio` dengan unit test dan integrasi Insomnia/Postman lengkap untuk mempercepat pengujian QA.
