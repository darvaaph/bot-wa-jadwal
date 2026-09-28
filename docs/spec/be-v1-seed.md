# Spec: Seed Kelas Pilot beserta Manifest Mapping

> Tahap: L1 item 2 dari `docs/api/PLAN_V1.md` (`feat/be-v1-seed`), blocked by tiket 01.
> Glosarium memakai `CONTEXT.md`. Menghormati ADR-0001 s.d 0005 dan
> `docs/product/DATA_MODEL.md` §14 (strategi migrasi).
> Status: ready-for-agent (terbit lokal; tracker file `.scratch/`).

## Seam pengujian yang diusulkan

Satu seam utama (tertinggi yang memungkinkan):

- **Seam S1 — entri seed melawan database terisolasi.**
  Perilaku eksternal yang diuji: diberi manifest + berkas kurikulum JSON,
  seed menghasilkan Kelas, Semester Kelas aktif, Course Offering, dosen,
  Pola Jadwal, Tugas warisan, dan Materi warisan yang bisa diquery, plus
  laporan cocok sumber vs target dan antrean resolusi untuk yang tak cocok.
  Prior art: pola `db_test.go` (InitDB + temp file + asersi SQL) dan
  `task_test.go` (verifikasi perilaku di atas `*sql.DB` temp, tanpa file
  produksi). Validasi manifest murni (tanpa database) boleh menjadi fungsi
  kecil terpisah, tetapi tetap dihitung satu seam logis: manifest masuk,
  baris + laporan keluar.
  Jika seam ini tidak sesuai ekspektasi, koreksi sebelum dikerjakan agen.

## Problem Statement

Skema target v1 (tiket 01) adalah database kosong: benar secara struktur,
tetapi tidak ada Kelas, Semester, Course Offering, dosen, atau Pola Jadwal
di dalamnya. Tanpa data, tidak satu pun endpoint Portal Kelas, publish
Tugas, atau publish Kejadian Perkuliahan bisa diuji. Di sisi lain, data
sumber (19 berkas kurikulum JSON berisi peta dosen, peta mata kuliah, dan
daftar jadwal per Kelas) memakai kode lama seperti `D4-TI-SMT3-A` yang tidak
mengandung program, angkatan, rombel, atau tahun akademik secara eksplisit —
menebaknya dari nama berkas dilarang oleh model data. Pengembang butuh cara
yang terkontrol untuk memuat 1–2 Kelas pilot dari JSON ke model target,
dengan mapping yang ditulis manusia, hitung cocok yang bisa diaudit, dan
antrean untuk data yang tidak cocok, sebelum menyentuh 17 Kelas sisanya.

## Solution

Sediakan perintah seed yang membaca manifest mapping (ditulis manusia) plus
berkas kurikulum JSON Kelas pilot, lalu menulis Kelas permanen, pengaturan
Kelas, satu Semester aktif, master mata kuliah dan dosen, Course Offering,
relasi offering–dosen, Pola Jadwal, serta Tugas dan Materi warisan Kelas
pilot ke database baru. Setiap jalan seed mengeluarkan laporan: jumlah
sumber, jumlah target, baris gagal, checksum, dan antrean resolusi. Baris
yang tidak punya pasangan offering masuk antrean pending — tidak pernah
dipaksakan ke mata kuliah yang mirip. Seed dapat dijalankan ulang tanpa
menggandakan data.

## User Stories

1. As a backend developer, I want to seed one pilot class with a single command, so that I get a queryable database in minutes.
2. As a backend developer, I want the manifest to state program, cohort, group, academic year, and initial active semester explicitly, so that no cohort is ever guessed from a filename.
3. As a backend developer, I want re-running the seed to update rather than duplicate rows, so that retries and corrections are safe.
4. As a backend developer, I want a checksum report of source vs target counts plus failures, so that I can prove nothing was lost or invented.
5. As a backend developer, I want unmatched rows to land in a resolution queue instead of a lookalike subject, so that bad matches never corrupt data silently.
6. As a QA tester, I want only 1–2 pilot classes seeded first, so that mapping mistakes stay small and reviewable.
7. As a QA tester, I want an isolated database per seed test run, so that tests never touch production or session files.
8. As a Ketua Murid (KM), I want my pilot Kelas readable in the portal with its real schedule, so that I can verify the migration looks right.
9. As a PJ Mata Kuliah, I want my Course Offering to carry the right lecturers and schedule patterns, so that my subject is mine and complete.
10. As a PJ Mata Kuliah, I want theory and practicum to become separate offerings when the source distinguishes them, so that each has its own schedule and assignee.
11. As a System Admin, I want lecturers deduplicated into one master by code, so that one lecturer teaching two offerings stays one person.
12. As a System Admin, I want courses deduplicated into one master by code, so that cross-class references stay consistent later.
13. As a System Admin, I want rooms referenced by source text preserved without inventing a master room, so that room master management stays a later, deliberate step.
14. As a System Admin, I want legacy tasks of pilot classes carried over with their class and subject name, so that active deadlines survive the move.
15. As a System Admin, I want legacy class links of pilot classes carried over as class-scoped materials, so that Drive and meeting links keep working.
16. As a System Admin, I want the old database left byte-identical, so that the WhatsApp bot keeps serving all 19 classes during the pilot.
17. As a Mahasiswa, I want the pilot portal to show the same schedule I saw from the old source, so that I trust the new system.
18. As a backend developer, I want master lecturer/course writes to be idempotent by code, so that seeding class two never duplicates class one's masters.
19. As a backend developer, I want every seeded row traceable to its source file and row, so that a bad row can be traced and fixed at the origin.
20. As a System Admin, I want to know exactly which rows failed and why, so that fixes go into the source or manifest, not into silent defaults.

## Implementation Decisions

- Modul yang dibangun: perintah seed baru beserta tiga fungsi logis —
  validasi manifest, impor akademik (Kelas, pengaturan, Semester,
  master, offering, relasi dosen, Pola Jadwal), dan migrasi warisan
  (Tugas, tautan menjadi Materi). Tidak ada perubahan handler API, bot,
  atau dashboard pada slice ini.
- Modul yang dipakai (tanpa modifikasi perilaku): pemuat kurikulum JSON
  yang sudah ada sebagai pembaca sumber, dan koneksi database yang sudah
  ada sebagai tujuan tulis.
- Keputusan dari ADR yang dihormati: hanya 1–2 Kelas pilot, database lama
  read-only selama transisi (ADR-0004); tidak ada endpoint pada slice ini
  (ADR-0005, lapis L1 sebelum L2); glosarium `CONTEXT.md` untuk penamaan.
- Keputusan dari prototyping: tidak ada; tidak ada snippet yang di-inline.
- Aturan manifest: wajib berisi program studi, tahun angkatan, label
  rombel, slug portal, tahun akademik dan term semester awal, tanggal
  mulai/selesai, zona waktu Kelas, dan mode akses portal (default tautan).
  Manifest tanpa field wajib ditolak sebelum menulis satu baris pun, dengan
  daftar field yang hilang.
- Aturan impor akademik: satu Semester berstatus ACTIVE per Kelas dengan
  tanggal mulai/selesai terisi; kombinasi Kelas+tahun+term unik; master
  mata kuliah unik per kode; master dosen unik per kode/inisial; offering
  unik per semester+mata kuliah+jenis aktivitas; pola menolak akhir yang
  tidak setelah awal; ruangan sumber tak dikenal disimpan sebagai teks
  referensi, bukan master ruangan baru.
- Aturan warisan: Tugas warisan Kelas pilot ditulis dengan status Terbit
  dan Perlu Review (cerminan publikasi sebelum review retrospektif);
  baris tanpa pasangan offering masuk antrean resolusi beserta alasan;
  tautan warisan menjadi Materi lingkup Kelas (tanpa offering) kecuali
  kategorinya jelas menunjuk satu mata kuliah pilot.
- Aturan idempotensi: kunci alami (kode Kelas, kode master, offering per
  semester+mata kuliah+jenis, pola per offering+hari+jam) membuat jalan
  ulang bersifat pemutakhiran, bukan penggandaan; laporan selalu ditulis
  ulang dari hasil jalan terakhir.
- Kontrak API pada slice ini: tidak ada. Konsumen berikutnya adalah slice
  endpoint (portal baca dulu, sesuai tiket 04).

## Testing Decisions

- Definisi test yang baik untuk slice ini: hanya menguji perilaku eksternal
  (baris bisa diquery, hitung cocok, antrean terisi, jalan ulang tak
  ganda, manifest buruk ditolak sebelum tulis), bukan detail implementasi
  (urutan insert internal, teks SQL). Test yang pecah karena refactor
  internal tanpa mengubah hasil dianggap test yang buruk.
- Modul yang diuji: entri seed beserta validasi manifest (seam S1).
  Tidak menguji handler HTTP, bot, atau frontend pada slice ini.
- Prior art: pola `db_test.go` (database temp, asersi via query, cleanup)
  dan pola `task_test.go` (skenario ujung ke ujung di atas database temp).
  Test seed memakai database temp yang sudah dimigrasi skema v1, plus
  salinan kecil fixture JSON, bukan berkas produksi.
- Kasus minimum: (1) seed Kelas pilot menghasilkan Kelas+Semester
  aktif+offering+pola yang bisa diquery; (2) jalan ulang tidak menggandakan;
  (3) manifest tanpa field wajib ditolak tanpa tulis; (4) kode angkatan yang
  tidak valid ditolak; (5) mata kuliah/dosen ganda antar dua seed tidak
  ganda di master; (6) Tugas tanpa pasangan offering masuk antrean, bukan
  offering mirip; (7) laporan memuat sumber, target, gagal, dan checksum;
  (8) database lama tidak berubah.
- Kriteria lolos slice: `go test -v ./...` dan `go vet ./...` hijau,
  `gofmt` bersih, tidak ada file produksi atau sesi yang dibuat/diubah
  oleh test maupun seed.

## Out of Scope

- 17 Kelas non-pilot — gelombang berikutnya setelah pilot diverifikasi.
- Endpoint `/api/v1` apa pun dan middleware auth — lapis L2 (tiket 03–07).
- Master ruangan terkelola dan konfirmasi TU — v1.1.
- Antrean Notifikasi pengiriman, audit global UI, backup/restore —
  v1.1+ (501 terstruktur per kontrak API).
- Perubahan dashboard web, bot WhatsApp, dan cutover DSN — lapis L3
  (tiket 08).
- Penghapusan tabel atau berkas lama — hanya via migrasi terpisah setelah
  backup dan persetujuan.

## Further Notes

- Setelah slice ini, tiket 03 (auth) dapat berjalan paralel dengan
  verifikasi portal-langsung-dari-SQL; tiket 04 adalah konsumen pertama
  data ini.
- Rollback slice ini trivial: database baru belum ditunjuk siapa pun;
  hapus baris seed atau mulai dari salinan kosong dan seed ulang dengan
  manifest yang diperbaiki.
- Perintah verifikasi tiap PR: `go test -v ./...`, `go vet ./...`,
  `gofmt -w` pada file yang diubah.
