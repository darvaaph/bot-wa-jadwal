# Spec: Skema Target v1 di Database Baru (tanpa Seed Data)

> Tahap: L1 item 1 dari `docs/api/PLAN_V1.md` (`feat/be-v1-schema`).
> Glosarium memakai `CONTEXT.md`. Menghormati ADR-0001 s.d 0005.
> Status publish tracker: TERBLOKIR — vocabulary tracker/label belum disediakan.
> Pengguna perlu menjalankan `/setup-matt-pocock-skills` agar spec ini bisa diterbitkan dengan label `ready-for-agent`.

## Seam pengujian yang diusulkan

Satu seam utama (tertinggi yang memungkinkan):

- **Seam S1 — entri migrasi paket database melawan SQLite terisolasi.**
  Perilaku eksternal yang diuji: setelah migrasi dijalankan pada database kosong,
  seluruh tabel target ada, foreign key aktif, status CHECK menolak nilai liar,
  indeks unik menolak duplikat, migrasi dapat dijalankan ulang tanpa error
  (idempoten), dan database lama tidak disentuh.
  Prior art: pola `db_test.go` (InitDB + temp file + asersi SQL) dan
  `task_test.go` (manajer dibangun di atas `*sql.DB` temp, tanpa file produksi).
  Seam baru memang diperlukan (fungsi migrasi belum ada), tetapi ditaruh setinggi
  mungkin: satu entri di paket database, bukan per-tabel.
- **Seam S2 (kecil, murni) — validasi manifest migrasi tanpa database.**
  Hanya jika slice seed ikut tersentuh; untuk spec ini S2 opsional/ditunda.
  Ideal = satu seam (S1). Jika seam ini tidak sesuai ekspektasi, koreksi sebelum
  dikerjakan agen.

## Problem Statement

Backend hari ini menyimpan Tugas dan tautan dengan pemilik `scope_jid`,
jadwal hidup di file JSON per Kelas, dan tidak ada tabel untuk
Penugasan Peran, Undangan, Sesi Pengurus, Sesi Portal, Semester Kelas,
Course Offering, Pola Jadwal, Kejadian Perkuliahan, review Tugas versi,
Materi terikat Kelas, Kanal WhatsApp, antrean Notifikasi idempoten, maupun
Riwayat Perubahan. Akibatnya tidak satu pun endpoint `/api/v1` dari
`docs/api/API_V1.md` (auth, Portal Kelas baca, publish Tugas, publish
Kejadian Perkuliahan) bisa dibangun: tidak ada tempat menyimpan datanya,
tidak ada constraint yang menegakkan isolasi Kelas, dan tidak ada fondasi
untuk mencabut sesi atau mengaudit publikasi. Pengembang butuh skema target
yang berdiri di database baru yang bersih, terverifikasi oleh test, sebelum
lapis endpoint dimulai.

## Solution

Sediakan migrasi skema satu-kali yang membangun seluruh tabel target v1 di
database baru yang kosong, dengan foreign key, batasan status, keunikan, dan
indeks sesuai `docs/product/DATA_MODEL.md`. Migrasi harus idempoten
(aman dijalankan ulang), tidak menyentuh database lama, dan terbukti lewat
test yang memeriksa perilaku database dari luar (bukan detail SQL per baris).
Setelah slice ini selesai, slice seed pilot dan slice endpoint dapat dibangun
di atas fondasi yang sudah terkunci.

## User Stories

1. As a backend developer, I want a single migration entry that creates all v1 tables on an empty database, so that setup is one call, not dozens of manual statements.
2. As a backend developer, I want to re-run the migration safely on an already-migrated database, so that deploys and retries never corrupt the schema.
3. As a backend developer, I want the old database left untouched, so that the WhatsApp bot keeps running during the transition.
4. As a backend developer, I want foreign keys enforced, so that orphan rows (e.g. a Tugas without its Course Offering) are impossible.
5. As a System Admin, I want each Kelas to allow at most one active Semester Kelas, so that Portal Kelas never shows two conflicting "current" semesters.
6. As a System Admin, I want Kelas identity (program, cohort, group) unique, so that duplicate classes cannot be created silently.
7. As a Ketua Murid (KM), I want my Penugasan Peran scoped to my Kelas and surviving semester changes, so that I am not re-invited every semester.
8. As a PJ Mata Kuliah, I want my Penugasan Peran bound to one Course Offering in one Semester Kelas, so that I can only change my own subject.
9. As a PJ Mata Kuliah, I want an Undangan that cannot be retargeted by the recipient, so that invite links cannot escalate privileges.
10. As a pengurus, I want my Sesi Pengurus revocable from the server, so that a lost device stops working immediately.
11. As a pengurus with several assignments, I want context switching to rotate my token, so that a stale token cannot act under the old context.
12. As a Mahasiswa, I want Portal Kelas readable without an account and without granting any write power, so that class info stays public yet safe.
13. As a KM, I want rotating the portal code to invalidate old Sesi Portal, so that a leaked link or code stops working.
14. As a PJ Mata Kuliah, I want to save a Tugas as draft before it is complete, so that I never lose half-written work.
15. As a PJ Mata Kuliah, I want publishing a Tugas to record me as publisher and mark it needing KM review without blocking publication, so that students see it now and review happens retrospectively.
16. As a KM, I want review decisions (approve, request changes, revoke) appended per Tugas version, so that history is never overwritten.
17. As a Mahasiswa, I want overdue Tugas to still appear as overdue rather than vanish, so that I see what I missed.
18. As a PJ Mata Kuliah, I want concurrent edits detected via version, so that I never silently overwrite a colleague's change.
19. As a PJ Mata Kuliah, I want Pola Jadwal versioned instead of overwritten, so that permanent changes keep history.
20. As a PJ Mata Kuliah, I want a Kejadian Perkuliahan draft invisible to the Portal until published, so that tentative changes never leak.
21. As a KM, I want revoking a publication to require a reason and keep the original record, so that corrections stay traceable.
22. As a KM of a participating class, I want cross-class event participation tracked per offering (pending/accepted/declined/removed), so that my class only shows events we accepted.
23. As a PJ Mata Kuliah, I want room confirmation recorded with recorder, time, and source before publishing when a room is needed, so that unconfirmed rooms are never presented as certain.
24. As a KM, I want Materi scoped to my Kelas (optionally to one Course Offering), so that general class files do not need a fake subject.
25. As a System Admin, I want Notifikasi queued with idempotency keys separate from academic data, so that retries never double-send and bot outages never cancel web publication.
26. As a System Admin, I want Riwayat Perubahan append-only with actor, scope, before/after, and reason, so that every publish, revoke, delete, restore, and role change is auditable.
27. As a QA tester, I want constraint violations rejected by the database itself, so that authorization and lifecycle bugs surface even if application code has gaps.
28. As a QA tester, I want an isolated test database per run, so that tests never touch production or session files.

## Implementation Decisions

- Modul yang dibangun: satu entri migrasi skema v1 di paket database,
  ditambah (jika belum ada) pemisah konfigurasi nama database baru vs lama
  di paket config. Tidak ada perubahan handler API, bot WhatsApp, atau
  dashboard web pada slice ini.
- Modul yang dimodifikasi: paket database saja. Paket tugas, tautan, jadwal,
  chat, reminder, dan bot tidak disentuh; mereka tetap menunjuk database lama
  sampai cutover diumumkan pada slice L3.
- Cakupan tabel (kelompok, sesuai DATA_MODEL): identitas dan akses (users,
  role_assignments, role_invitations, login_attempts, user_sessions,
  portal_sessions, recovery_tokens); akademik (classes, class_settings,
  semesters, courses, course_offerings, lecturers, offering_lecturers);
  jadwal dan ruangan (rooms, schedule_patterns, teaching_events,
  teaching_event_offerings, room_confirmations); tugas dan materi (tasks,
  task_reviews, materials); WhatsApp (whatsapp_channels,
  notification_messages, notification_attempts); operasional (audit_logs,
  import_batches, import_errors, backup_records).
- Keputusan dari ADR yang dihormati: database baru yang bersih berisi skema
  murni tanpa kolom warisan `scope_jid` (ADR-0004); tabel sesi opaque
  mengikuti ADR-0003 (token hash, active assignment, session version);
  tidak ada endpoint pada slice ini (ADR-0005, lapis L1 sebelum L2).
- Keputusan dari protyping: tidak ada; tidak ada snippet prototipe yang
  perlu di-inline.
- Aturan status ditegakkan di database (CHECK atau tabel referensi, bukan
  hanya di kode): role ACTIVE/SUSPENDED/REVOKED; undangan
  PENDING/ACCEPTED/EXPIRED/REVOKED; semester DRAFT/ACTIVE/ARCHIVED; pola
  ACTIVE/SUPERSEDED/ARCHIVED/DELETED; kejadian DRAFT/PUBLISHED/REVOKED;
  publikasi Tugas DRAFT/PUBLISHED/REVOKED; review NOT_REVIEWED plus keputusan
  APPROVED/CHANGES_REQUESTED/REVOKED; konfirmasi ruangan
  PENDING/CONFIRMED/REJECTED; notifikasi
  PENDING/PROCESSING/SENT/FAILED/CANCELLED/SUPERSEDED.
- Constraint lintas entitas kunci: satu semester ACTIVE per Kelas; satu
  OWNER per Kejadian Perkuliahan; peserta tidak boleh dari Kelas pemilik;
  `end_time` pola harus setelah `start_time`; tanggal kejadian wajib dalam
  semester pemilik; soft-delete memakai `deleted_at` + pelaku dan
  disembunyikan dari query aktif tetapi tersedia untuk audit dan pemulihan.
- Indeks mengikuti DATA_MODEL §11 (unik kelas/slug, unik offering per
  semester, unik idempotency key, pencarian sesi per token, audit per
  kelas/entitas/pelaku). Indeks final divalidasi terhadap query nyata pada
  slice endpoint, bukan ditambah spekulatif di sini.
- Transaksi dan idempotensi migrasi: seluruh DDL memakai `IF NOT EXISTS`
  dan dapat dijalankan ulang tanpa error; verifikasi skema tidak menulis
  data bisnis.
- Kontrak API pada slice ini: tidak ada (tidak ada endpoint baru, tidak ada
  perubahan envelope). Slice endpoint (L2) adalah konsumen berikutnya.

## Testing Decisions

- Definisi test yang baik untuk slice ini: hanya menguji perilaku eksternal
  database (tabel ada, tulis valid lolos, tulis melanggar ditolak, migrasi
  ulang aman), bukan detail implementasi (teks SQL per pernyataan, urutan
  DDL internal). Test yang pecah karena refactor SQL tanpa mengubah perilaku
  dianggap test yang buruk.
- Modul yang diuji: entri migrasi paket database (seam S1). Tidak menguji
  handler HTTP, bot, atau frontend pada slice ini.
- Prior art: pola `db_test.go` (InitDB pada file temp, asersi via query,
  cleanup file `-wal`/`-shm`) dan pola `task_test.go` (bangun manajer di
  atas `*sql.DB` temp). Test baru mengikuti pola yang sama: database
  terisolasi per run, tidak pernah memakai file produksi atau sesi.
- Kasus minimum: (1) migrasi dari kosong menghasilkan semua tabel;
  (2) migrasi ulang idempoten; (3) foreign key aktif (tulis yatim ditolak);
  (4) dua semester ACTIVE satu Kelas ditolak; (5) OWNER ganda satu kejadian
  ditolak; (6) status liar ditolak CHECK; (7) idempotency key ganda ditolak;
  (8) pola dengan akhir <= awal ditolak.
- Kriteria lolos slice: `go test -v ./...` dan `go vet ./...` hijau,
  `gofmt` bersih, tidak ada file database produksi yang dibuat atau diubah
  oleh test.

## Out of Scope

- Seed data (kelas pilot, manifest mapping, impor JSON, checksum) — slice
  `feat/be-v1-seed` berikutnya.
- Endpoint `/api/v1` apa pun, middleware auth, dan shim legacy — lapis L2.
- Migrasi baris data lama (`tasks`, `class_links`, `chat_settings`,
  overrides) menjadi baris target — bagian dari slice seed, bukan slice ini.
- Notifikasi pengiriman, pencarian Ruangan Kandidat, backup/restore
  operasional, audit global UI — v1.1+ (501 terstruktur per API_V1 §7).
- Perubahan dashboard web, bot WhatsApp, dan cutover DSN — lapis L3.
- Penghapusan tabel lama — hanya via migrasi terpisah setelah backup dan
  persetujuan (per DATA_MODEL).

## Further Notes

- Setelah slice ini, urutan tetap horizontal per ADR-0005: seed pilot (L1)
  → endpoint (L2) → QA + cutover frontend sekaligus (L3). Frontend menunggu
  sampai L3; perubahan kontrak setelah L0 butuh ADR baru.
- Rollback slice ini trivial: database baru hanyalah file; hapus file dan
  tidak ada sistem yang terpengaruh karena belum ada yang menunjuknya.
- Perintah verifikasi tiap PR: `go test -v ./...`, `go vet ./...`,
  `gofmt -w` pada file yang diubah, `go run ./cmd/bot -web-only` untuk
  memastikan biner tetap hidup (bot lama tetap menunjuk database lama).
