# Documentation Tickets dan Production Gate

## Tujuan

Dokumen ini mendefinisikan pekerjaan dokumentasi dan gerbang produksi Bot Jadwal. Dokumentasi dinyatakan selesai hanya bila sesuai dengan implementasi yang berjalan dan dapat digunakan orang lain tanpa instruksi lisan. Production gate dinyatakan lulus hanya bila bukti teknis, QA, keamanan, recovery, dan kesiapan operasional tersedia untuk satu commit kandidat yang sama.

Dokumen acuan:

- [API v1](api/API_V1.md)
- [Panduan QA](api/PANDUAN_TESTING_QA.md)
- [QA Tickets](QA_TICKETS_QA-001_QA-014.md)
- [Functional Requirements](product/FUNCTIONAL_REQUIREMENTS.md)
- [Access Control](product/ACCESS_CONTROL.md)
- [Traceability Matrix](product/TRACEABILITY.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Contributing Guide](CONTRIBUTING.md)
- [Team Onboarding](TEAM_ONBOARDING.md)
- [Panduan Penggunaan](PANDUAN_PENGGUNAAN.md)
- [Backend Security Tickets](BACKEND_TICKETS_BE-012_BE-013_BE-014.md)

## Baseline repository

Temuan baseline yang harus diverifikasi dan diselesaikan melalui ticket di bawah:

- Dokumentasi tersebar pada README, `docs/api`, `docs/product`, panduan penggunaan, onboarding, dan deployment.
- Label versi belum konsisten. Beberapa dokumen menyebut v2.0, sedangkan panduan QA dan functional requirements menyebut v3.0.
- Beberapa contoh perintah masih memakai `go run .` atau `go build ... .`, sementara entry point resmi repository adalah `./cmd/bot`.
- Contoh direktori lokal pada deployment guide tidak sama dengan nama workspace saat ini.
- Deployment guide memuat langkah manual penggantian binary, tetapi belum mendefinisikan artefak immutable, checksum, health verification, batas waktu observasi, dan rollback lengkap.
- Struktur file pada deployment guide harus diperiksa karena code fence dapat membuat bagian berikutnya dirender sebagai blok kode.
- Klaim seperti “aman 100%” tidak dapat menjadi kontrol produksi dan harus diganti dengan risiko serta verifikasi yang konkret.
- Laporan QA yang menyatakan `GO` tetap harus ditautkan ke raw evidence, commit SHA, environment, dan waktu eksekusi sebelum diterima production gate.
- Perubahan lokal pada `docs/DEPLOYMENT.md`, ADR keamanan, dan laporan QA sedang ada di working tree. Ticket ini tidak menganggap perubahan tersebut final sebelum direview dan di-commit.

## Ringkasan backlog

| ID | Judul | Prioritas | Owner utama | Keluaran |
|---|---|---:|---|---|
| DOC-001 | Inventaris, Hierarki, Versi, dan Ownership Dokumentasi | P0 | Tech Lead | Peta dokumen kanonis |
| DOC-002 | Sinkronisasi API dan Panduan Integrasi Frontend | P0 | Backend + Frontend | Dokumentasi kontrak final |
| DOC-003 | Dokumentasi Konfigurasi, Security, dan Secret | P0 | Backend + Operations | Environment reference aman |
| DOC-004 | Runbook Deployment, Backup, Restore, dan Rollback | P0 | Operations | Runbook teruji |
| DOC-005 | Panduan Operator, Pengurus, dan Dukungan | P1 | Product + Operations | Panduan berbasis peran |
| DOC-006 | Traceability, Release Notes, dan Paket Evidence | P0 | QA Lead + Tech Lead | Release documentation pack |
| OPS-001 | Staging Production Rehearsal | P0 | Operations + QA | Bukti rehearsal |
| OPS-002 | Production Cutover dan Rollback Control | P0 | Release Manager | Checklist cutover |
| OPS-003 | Post-Deploy Verification dan Incident Readiness | P0 | Operations | Bukti stabilisasi dan respons |
| PROD-GATE-001 | Final Production Go/No-Go | P0 | Release Committee | Keputusan tertulis |

## Aturan umum

1. Dokumentasi menjelaskan perilaku yang benar-benar tersedia, bukan fitur yang direncanakan seolah sudah aktif.
2. Contoh tidak boleh memuat alamat, akun, token, cookie, hash key, session file, atau secret nyata.
3. Setiap perintah harus dijalankan pada shell dan sistem operasi yang disebutkan.
4. Setiap runbook harus memiliki prasyarat, langkah verifikasi, failure path, rollback, owner, dan escalation path.
5. Istilah produk, status, field, route, dan error code harus sama dengan implementasi serta kontrak API.
6. Satu dokumen ditetapkan sebagai sumber kanonis untuk setiap topik. Dokumen lain menautkannya, bukan menyalin isi yang mudah drift.
7. Production gate menerima evidence, bukan pernyataan “sudah dites” tanpa output atau referensi.
8. Semua evidence release harus menyebut commit SHA, environment, timestamp, executor, dan hasil.
9. Perubahan setelah commit kandidat dibekukan membatalkan evidence terdampak sampai impact analysis dan re-test selesai.
10. `GO` tidak dapat diberikan bila rollback belum dipraktikkan atau backup belum diverifikasi.

---

## DOC-001: Inventaris, Hierarki, Versi, dan Ownership Dokumentasi

### Status

- Prioritas: P0
- Owner: Tech Lead
- Reviewer: Product, Backend, Frontend, QA, Operations
- Dependensi: keputusan versi produk dan owner tiap domain

### Tujuan

Menetapkan dokumen kanonis, menghapus konflik versi dan instruksi, serta memastikan setiap dokumen memiliki audience, owner, status, dan tanggal review.

### Ruang lingkup

- README dan pusat navigasi dokumentasi.
- Requirement, business rules, access control, data model, API, design, QA, deployment, user guide, onboarding, dan ADR.
- Status `Draft`, `Approved`, `Deprecated`, atau `Archived`.
- Versi produk dan versi dokumen.
- Owner serta review cadence.
- Tautan rusak, duplikasi, instruksi lama, dan istilah tidak konsisten.

### Keputusan struktur minimum

| Topik | Sumber kanonis yang disarankan |
|---|---|
| Tujuan dan scope produk | `docs/PRD.md` dan `docs/product/PRODUCT_DEFINITION.md` |
| Requirement | `docs/product/USER_REQUIREMENTS.md` dan `FUNCTIONAL_REQUIREMENTS.md` |
| Aturan bisnis dan akses | `BUSINESS_RULES.md` dan `ACCESS_CONTROL.md` |
| Kontrak API | `docs/api/API_V1.md` |
| Pengujian | `docs/QA_TICKETS_QA-001_QA-014.md` dan laporan evidence per run |
| Deployment dan operasi | `docs/DEPLOYMENT.md` serta runbook yang ditetapkan DOC-004 |
| Keputusan arsitektur | `docs/adr/*.md` |
| Penggunaan aplikasi | `docs/PANDUAN_PENGGUNAAN.md` |

### Acceptance criteria

- [ ] Satu versi produk kanonis ditetapkan dan dipakai pada seluruh dokumen aktif.
- [ ] Setiap dokumen memiliki purpose, audience, status, owner, versi atau tanggal review bila relevan.
- [ ] README menyediakan jalur navigasi untuk developer, QA, operator, dan pengguna.
- [ ] Tidak ada dua dokumen aktif yang memberikan instruksi bertentangan tanpa penjelasan precedence.
- [ ] Dokumen deprecated menunjuk penggantinya.
- [ ] Seluruh tautan relatif Markdown valid.
- [ ] Istilah role, status lifecycle, nama entity, dan singkatan konsisten.
- [ ] Rencana fitur dibedakan jelas dari fitur yang telah tersedia.
- [ ] Emoji dan gaya visual tidak menghalangi pencarian atau pemahaman heading.

### Checklist review

- [ ] Cari semua label `v1`, `v2`, `v3`, `draft`, `selesai`, dan `production-ready`.
- [ ] Cari semua perintah `go run`, `go build`, `go test`, dan path project.
- [ ] Cari semua route API, nama environment variable, dan nama service.
- [ ] Periksa broken links dan heading anchor.
- [ ] Periksa code fence yang tidak seimbang.
- [ ] Periksa klaim absolut tanpa evidence.
- [ ] Periksa duplikasi credential contoh dan data pribadi.

### Evidence wajib

- Inventaris dokumen dan owner.
- Daftar konflik yang diselesaikan.
- Hasil pemeriksaan tautan dan Markdown.
- Daftar dokumen deprecated beserta pengganti.

### Definition of Done

- [ ] Seluruh dokumen aktif tercatat dalam inventaris.
- [ ] Versi dan status konsisten.
- [ ] Tidak ada broken link atau unbalanced code fence.
- [ ] Reviewer lintas fungsi menyetujui sumber kanonis.

### Prompt implementasi

```text
Kerjakan DOC-001 pada repository Bot Jadwal. Inventaris seluruh Markdown, JSON collection, ADR, dan panduan operasional. Tetapkan purpose, audience, status, owner, versi, dan sumber kanonis untuk tiap topik. Cari konflik versi produk, istilah, route, environment variable, perintah Go, path project, service name, serta instruksi yang saling bertentangan. Periksa relative links dan code fence. Bedakan fitur implemented dari planned. Jangan menghapus dokumen historis tanpa menggantinya dengan status deprecated dan tautan pengganti. Hasilkan peta dokumentasi dan daftar perubahan yang dapat direview.
```

---

## DOC-002: Sinkronisasi API dan Panduan Integrasi Frontend

### Status

- Prioritas: P0
- Owner: Backend dan Frontend Lead
- Reviewer: QA
- Dependensi: DOC-001, kontrak API final, QA-002

### Tujuan

Memastikan frontend dan integrator dapat menggunakan API v1 tanpa membaca handler atau menebak perilaku error.

### Ruang lingkup

- Route, method, auth, role, scope, request, response, status, error code, dan side effect.
- Session bearer/cookie, CSRF, CORS, idempotency, optimistic lock, pagination, filter, dan timezone.
- Lifecycle semester, schedule, task, notification, audit, backup, invitation, dan recovery.
- Collection Postman/Insomnia dan contoh request tersensor.
- Deprecated legacy shim dan tanggal atau kondisi penghentian.
- Integrasi state frontend untuk `401`, `403`, `404`, `409`, `422`, `429`, dan `5xx`.

### Acceptance criteria

- [ ] Semua route yang terdaftar benar-benar terdaftar pada server.
- [ ] Semua route server yang ditujukan untuk frontend terdokumentasi.
- [ ] Setiap mutasi menyebut auth, scope, precondition, idempotency, audit, dan notification side effect bila ada.
- [ ] Request serta response example sesuai schema aktual dan tidak memuat secret nyata.
- [ ] Error code memiliki arti serta tindakan frontend yang jelas.
- [ ] Aturan optimistic locking dan response conflict terdokumentasi.
- [ ] Timestamp menyebut format dan timezone.
- [ ] Collection memakai environment variable dan dapat dijalankan dari fixture bersih.
- [ ] Legacy endpoint memiliki header serta status deprecation yang benar.
- [ ] Dokumentasi dan collection cocok dengan hasil QA-002.

### Matriks integrasi minimum

| Area | Dokumentasi wajib |
|---|---|
| Auth | login, logout, me, context switch, expiry, recovery |
| Portal | mode LINK/CODE, rotation, summary, schedule, task, material |
| Semester | draft, import, preview, activation, archive |
| Schedule | pattern, event, conflict, publish, revoke, participant, room |
| Task | draft, publish, version, review, complete, archive, restore |
| Notification | status, retry eligibility, idempotency, stale message |
| Audit | scope, filter, before/after, redaction |
| Backup | scope, checksum, verify/restore semantics |
| Security | CORS, CSRF, cookie, rate limit, retry-after |

### Definition of Done

- [ ] Backend, Frontend, dan QA menandatangani kontrak final.
- [ ] Tidak ada drift P0/P1 antara spec, collection, dan handler.
- [ ] Frontend error-state matrix menautkan error code resmi.
- [ ] Contoh dapat dijalankan pada environment QA.

### Prompt implementasi

```text
Kerjakan DOC-002. Bandingkan API_V1.md, DOKUMENTASI_API_FRONTEND.md, Postman, Insomnia, route registration, handler, dan hasil QA-002. Dokumentasikan seluruh endpoint frontend-facing dengan auth, scope, request, response, error code, idempotency, optimistic locking, timezone, audit, dan notification side effect. Tambahkan integration guidance untuk 401, 403, 404, 409, 422, 429, dan 5xx. Tandai legacy shim dengan kebijakan deprecation. Jalankan contoh terhadap fixture bersih. Jangan mendokumentasikan perilaku hanya berdasarkan asumsi atau nama handler.
```

---

## DOC-003: Dokumentasi Konfigurasi, Security, dan Secret

### Status

- Prioritas: P0
- Owner: Backend dan Operations
- Reviewer: Security dan QA
- Dependensi: BE-012, BE-013, BE-014, DOC-001

### Tujuan

Menyediakan referensi konfigurasi yang aman untuk development, test, staging, dan production tanpa memasukkan secret ke repository.

### Ruang lingkup

- Seluruh environment variable aktual yang dibaca aplikasi.
- Required/optional, default, format, contoh aman, dan validation behavior.
- Perbedaan development, test, staging, dan production.
- Secret generation, injection, access, rotation, revocation, dan recovery.
- Cookie, origin, proxy trust, HTTPS, HSTS, CSP, CSRF, rate limit, dan public base URL.
- Storage paths, database, session WhatsApp, log redaction, serta retention.

### Acceptance criteria

- [ ] Daftar environment variable berasal dari pencarian kode, bukan daftar lama.
- [ ] Setiap variable memiliki type, required state, allowed value, default, dan contoh nonsecret.
- [ ] Variable production yang wajib gagal aman ketika kosong atau invalid.
- [ ] Secret tidak disarankan melalui command history, source file, screenshot, atau Git.
- [ ] Prosedur rotasi menjelaskan dampak pada session, limiter, token, dan downtime.
- [ ] Trusted proxy menjelaskan CIDR yang benar dan risiko spoofing.
- [ ] Allowed origins menggunakan exact scheme, host, dan port.
- [ ] Cookie serta CSRF policy cocok dengan arsitektur aktual.
- [ ] CSP memuat dependency CDN aktual dan relaxation yang masih diperlukan.
- [ ] Tidak ada value production nyata dalam dokumentasi atau history perubahan ticket.
- [ ] Redaction checklist mencakup application log, journal, audit, error response, dan QA evidence.

### Tabel konfigurasi minimum

| Field | Wajib dijelaskan |
|---|---|
| Nama | Sama persis dengan kode |
| Environment | dev/test/staging/production |
| Required | ya/tidak dan kondisi |
| Type/format | boolean, URL, CIDR, duration, path, secret |
| Default | nilai aktual atau “tidak ada” |
| Safe example | placeholder tanpa secret |
| Startup behavior | fallback atau fail closed |
| Rotation impact | restart, session revoke, data migration |

### Definition of Done

- [ ] Config reference cocok dengan test config dan startup validation.
- [ ] Security serta Operations menyetujui secret lifecycle.
- [ ] Tidak ada secret nyata ditemukan pada dokumen aktif.
- [ ] Staging dapat dikonfigurasi hanya dari dokumentasi tersebut.

### Prompt implementasi

```text
Kerjakan DOC-003. Cari seluruh pembacaan environment variable dan config default pada kode. Buat config reference untuk development, test, staging, dan production dengan type, required state, format, default, safe example, startup behavior, dan rotation impact. Dokumentasikan HMAC key, secure cookie, allowed origins, trusted proxy CIDRs, public base URL, database/storage paths, session WhatsApp, HTTPS/HSTS, CSP, CSRF, rate limits, log redaction, dan retention. Verifikasi nama variable melalui code search dan test startup. Jangan menaruh secret nyata atau menyarankan penyimpanan secret di Git/systemd unit yang world-readable.
```

---

## DOC-004: Runbook Deployment, Backup, Restore, dan Rollback

### Status

- Prioritas: P0
- Owner: Operations
- Reviewer: Backend, QA, Security
- Dependensi: DOC-003, keputusan final BE-011

### Tujuan

Mengubah deployment guide menjadi runbook operasional yang dapat dipraktikkan, diaudit, dan dibalik tanpa kehilangan data.

### Ruang lingkup

- Build artefak Linux dari commit kandidat.
- Version metadata, checksum, storage, transfer, permission, dan ownership.
- Pre-deploy backup dan verification.
- Database migration strategy.
- Stop/start/restart service serta health verification.
- Reverse proxy, TLS, firewall, dan timezone.
- Cutover binary/config/database.
- Rollback binary, config, dan database.
- WhatsApp session safety.
- Verify-only atau full restore semantics.
- Failure path dan escalation.

### Acceptance criteria

- [ ] Semua command memakai entry point `./cmd/bot` dan path yang benar.
- [ ] Build mencatat commit SHA, Go version, target OS/arch, dan checksum.
- [ ] Artefak lama tidak ditimpa sebelum versi baru lolos verification.
- [ ] Backup dibuat dan diverifikasi sebelum perubahan data atau migrasi.
- [ ] Runbook menjelaskan siapa yang boleh deploy dan siapa yang memberi go/no-go.
- [ ] Service unit, working directory, environment source, user, file permission, dan restart policy terdokumentasi.
- [ ] Health check memeriksa proses, HTTP, database, bot status, dan critical smoke flow.
- [ ] Rollback memiliki trigger, langkah, batas waktu keputusan, serta owner.
- [ ] Rollback binary dan config tidak otomatis mengembalikan schema secara berbahaya.
- [ ] Strategi migration backward compatibility atau restore point dinyatakan eksplisit.
- [ ] WhatsApp session tidak disalin atau dicabut tanpa prosedur khusus.
- [ ] Runbook tidak memakai klaim absolut seperti “aman 100%”.
- [ ] Seluruh langkah telah dipraktikkan pada staging.

### Struktur runbook minimum

1. Scope dan audience.
2. Prerequisite serta required access.
3. Input release: commit, artefak, checksum, config version.
4. Preflight.
5. Backup dan verify.
6. Deploy/cutover.
7. Migration.
8. Health dan smoke verification.
9. Observation window.
10. Rollback triggers.
11. Rollback procedure.
12. Failure escalation.
13. Evidence capture dan closeout.

### Definition of Done

- [ ] Operator kedua menyelesaikan rehearsal tanpa instruksi lisan.
- [ ] Rollback drill berhasil pada staging.
- [ ] Backup hasil rehearsal dapat diverifikasi sesuai semantics final.
- [ ] Semua command dan path telah diuji.

### Prompt implementasi

```text
Kerjakan DOC-004 dengan mengaudit docs/DEPLOYMENT.md terhadap codebase dan deployment target aktual. Perbaiki entry point, project path, build target, service unit, environment source, storage path, migration, proxy, TLS, firewall, timezone, backup, restore semantics, dan WhatsApp session safety. Tambahkan artefak immutable, commit SHA, checksum, preflight, health checks, observation window, rollback triggers, rollback langkah demi langkah, owner, escalation, serta evidence. Hapus klaim absolut yang tidak dapat dibuktikan. Praktikkan deploy dan rollback pada staging, bukan production, lalu catat hasilnya.
```

---

## DOC-005: Panduan Operator, Pengurus, dan Dukungan

### Status

- Prioritas: P1
- Owner: Product dan Operations
- Reviewer: KM, PJ, Admin, Support, QA
- Dependensi: fitur dan UI final

### Tujuan

Menyediakan panduan berbasis tugas untuk pengguna dan operator, termasuk error recovery serta batas kewenangan.

### Audience

- Mahasiswa portal.
- PJ mata kuliah.
- KM.
- System Administrator.
- Operator bot dan support.

### Ruang lingkup

- Login, invitation, context switch, logout, dan recovery.
- Semester, schedule, task, material, room confirmation, notification, audit, dan backup sesuai role.
- Status loading, validation, denied, conflict, rate limited, offline, serta session expired.
- Operasi berisiko dan konfirmasi.
- Cara melaporkan masalah tanpa membagikan credential.
- Batas fitur serta known limitations.
- Accessibility dan mobile instructions bila diperlukan.

### Acceptance criteria

- [ ] Setiap role hanya melihat instruksi untuk kewenangannya.
- [ ] Nama menu, tombol, status, dan field sama dengan UI final.
- [ ] Panduan menjelaskan outcome serta cara pulih, bukan hanya urutan klik.
- [ ] Error umum memiliki langkah berikutnya yang aman.
- [ ] Panduan tidak meminta pengguna mengirim password, token, cookie, atau database.
- [ ] Operasi publish, revoke, suspend, backup, dan recovery memuat dampak.
- [ ] Screenshot memakai data dummy dan tidak cepat basi bila teks sudah cukup.
- [ ] Satu pengguna per role dapat menyelesaikan alur utama hanya dengan panduan.
- [ ] Contact/escalation path tidak memuat kontak pribadi bila dokumen bersifat publik.

### Definition of Done

- [ ] Usability walkthrough dilakukan untuk KM, PJ, Admin, dan operator.
- [ ] Semua blocker dokumentasi diperbaiki.
- [ ] Known limitations selaras dengan release notes.
- [ ] QA menyetujui kecocokan panduan dengan UI.

### Prompt implementasi

```text
Kerjakan DOC-005. Audit panduan penggunaan terhadap UI dan role final. Tulis task-based guidance untuk mahasiswa, PJ, KM, System Admin, operator bot, dan support. Cakup login/invitation/context, semester, jadwal, tugas, materi, ruangan, notifikasi, audit, backup, recovery, error state, offline, conflict, rate limit, dan session expiry sesuai role. Gunakan nama menu serta status aktual. Jelaskan dampak operasi berisiko dan recovery path. Jangan meminta pengguna membagikan credential, token, cookie, session file, atau database. Validasi melalui walkthrough satu pengguna per role.
```

---

## DOC-006: Traceability, Release Notes, dan Paket Evidence

### Status

- Prioritas: P0
- Owner: QA Lead dan Tech Lead
- Reviewer: Product dan Release Manager
- Dependensi: DOC-001 sampai DOC-005, QA-014

### Tujuan

Mengikat requirement, perubahan kode, ticket, test, defect, keputusan, dan artefak release pada satu commit kandidat.

### Ruang lingkup

- Matriks `requirement -> implementation -> test -> evidence -> status`.
- Changelog/release notes yang berorientasi dampak pengguna dan operator.
- Breaking/deprecated behavior.
- Database migration dan config changes.
- Security changes dan secret rotation impact.
- Known issues, workaround, owner, serta target perbaikan.
- Raw test output, collection result, UAT, security report, backup/rollback rehearsal.
- Software Bill of Materials hanya jika pipeline telah mendukung; jangan mengklaim tersedia bila belum ada.

### Acceptance criteria

- [ ] Setiap 48 `FR-*` memiliki implementation reference dan test evidence.
- [ ] Setiap BE/FE/QA ticket dalam release memiliki status serta commit/PR reference.
- [ ] Release notes menyebut perubahan user-facing, operator-facing, security, config, dan migration.
- [ ] Known issue memiliki severity, impact, workaround, owner, dan target.
- [ ] Evidence menyebut commit SHA yang sama dengan artefak release.
- [ ] Output gagal tidak dihapus dari laporan; hasil rerun ditautkan terpisah.
- [ ] Screenshot dan log telah disensor.
- [ ] QA `GO` tidak diterima bila hanya berupa tabel status tanpa raw evidence.
- [ ] Paket release dapat diperiksa tanpa akses ke mesin pembuatnya.

### Paket evidence minimum

```text
release-evidence/
  manifest.md
  commit-and-build.txt
  checksums.txt
  test-results/
  api-contract-results/
  browser-e2e/
  security-review/
  backup-rollback-rehearsal/
  uat/
  known-issues.md
  approvals.md
```

Direktori di atas adalah struktur konseptual. Lokasi penyimpanan final harus ditetapkan tim dan tidak boleh memuat secret atau data pribadi.

### Definition of Done

- [ ] Traceability lengkap dan direview.
- [ ] Release notes cocok dengan diff kandidat.
- [ ] Semua evidence dapat dibuka oleh reviewer berwenang.
- [ ] Manifest memiliki checksum atau referensi immutable yang disepakati.

### Prompt implementasi

```text
Kerjakan DOC-006 untuk satu commit kandidat. Perbarui traceability dari 48 FR ke file implementasi, ticket, test case, hasil, dan evidence. Susun release notes dari diff aktual, termasuk user impact, operator impact, security, config, migration, deprecation, known issues, workaround, owner, dan rollback impact. Bangun manifest evidence yang menautkan raw Go test/vet/build output, API contract result, browser E2E, security review, UAT, serta backup/rollback rehearsal. Pastikan seluruh evidence memakai commit SHA kandidat yang sama dan telah disensor. Jangan menerima status PASS atau GO tanpa bukti yang dapat ditinjau.
```

---

## OPS-001: Staging Production Rehearsal

### Status

- Prioritas: P0
- Owner: Operations dan QA
- Reviewer: Backend, Security, Release Manager
- Dependensi: DOC-002 sampai DOC-004, QA-001 sampai QA-013

### Tujuan

Membuktikan bahwa artefak, konfigurasi, migrasi, deployment, backup, rollback, dan smoke tests bekerja pada staging yang merepresentasikan production.

### Kesetaraan staging minimum

- Sistem operasi dan arsitektur target setara.
- Service manager dan working directory setara.
- Reverse proxy, HTTPS, trusted proxy, origin, dan security headers setara secara struktur.
- Database engine, schema baseline, storage permission, dan migration path setara.
- Secret berbeda dari production tetapi format serta injection method setara.
- WhatsApp memakai test session/sender atau adapter aman.
- Dataset disensor dan cukup untuk critical flows.

### Acceptance criteria

- [ ] Artefak dibangun dari commit kandidat dan checksum diverifikasi di staging.
- [ ] Production-mode startup validation lulus dengan secret staging.
- [ ] Migration dari snapshot baseline staging lulus.
- [ ] Semua P0 smoke tests lulus setelah deploy.
- [ ] Backup sebelum deploy dapat diverifikasi.
- [ ] Rollback binary/config berhasil.
- [ ] Rollback atau recovery database mengikuti strategi yang disetujui dan berhasil dipraktikkan.
- [ ] Restart service mempertahankan state yang harus durable.
- [ ] Proxy/client identity, CORS, cookie, CSP, dan HSTS behavior diverifikasi.
- [ ] Outbox pulih setelah restart tanpa duplicate business message.
- [ ] Durasi langkah dan downtime aktual dicatat.
- [ ] Semua deviation staging terhadap production didokumentasikan beserta risikonya.

### Evidence wajib

- Commit, build metadata, dan checksum.
- Config manifest tanpa secret.
- Migration dan service logs yang disensor.
- Before/after health checks.
- Backup verification.
- Rollback transcript.
- Security header/CORS/cookie evidence.
- Critical E2E dan bot smoke result.
- Durasi deploy, recovery, dan observation.

### Definition of Done

- [ ] Rehearsal dijalankan dari runbook, bukan langkah improvisasi.
- [ ] Operator kedua dapat mengulang hasil.
- [ ] Semua P0 defect ditutup dan rehearsal terdampak diulang.
- [ ] Release Manager menyetujui hasil rehearsal.

### Prompt implementasi

```text
Kerjakan OPS-001 menggunakan commit kandidat dan runbook DOC-004. Siapkan staging yang setara secara struktur dengan production tanpa memakai secret atau data production. Verifikasi checksum artefak, production startup validation, migration, service permissions, reverse proxy, HTTPS, CORS, cookie, CSP, HSTS, database, outbox, bot test adapter, dan critical E2E. Buat serta verifikasi backup, praktikkan rollback binary/config/database sesuai strategi, lalu ulangi health check. Catat durasi, downtime, deviation, log tersensor, dan setiap langkah yang tidak dapat diulang. Jangan menjalankan langkah ini pada production.
```

---

## OPS-002: Production Cutover dan Rollback Control

### Status

- Prioritas: P0
- Owner: Release Manager
- Executor: Operations
- Approver: Product, Tech Lead, QA Lead, Security
- Dependensi: OPS-001 lulus, PROD-GATE-001 preflight lulus

### Tujuan

Menjalankan cutover production secara terkendali dengan artefak yang sama seperti rehearsal dan kemampuan rollback yang nyata.

### Pre-cutover checklist

- [ ] Commit kandidat dibekukan.
- [ ] Artefak dan checksum sama dengan staging rehearsal.
- [ ] Semua approval production gate tercatat.
- [ ] Maintenance window dan komunikasi disetujui.
- [ ] Release owner, operator, verifier, dan rollback owner hadir.
- [ ] Backup production baru selesai dan terverifikasi.
- [ ] Kapasitas storage untuk backup, binary lama, dan migration cukup.
- [ ] Secret/config production tersedia melalui jalur aman.
- [ ] Monitoring dan log access siap.
- [ ] Rollback triggers serta decision deadline dibaca bersama.
- [ ] Tidak ada incident aktif yang membuat deploy berisiko.

### Cutover controls

1. Catat start time dan baseline health.
2. Verifikasi checksum artefak sebelum instalasi.
3. Ambil serta verifikasi backup.
4. Hentikan atau drain service sesuai runbook.
5. Pasang artefak dan config tanpa menghapus versi terakhir.
6. Jalankan migration sesuai strategi.
7. Start service dan lakukan health check berlapis.
8. Jalankan smoke test read-only lebih dahulu.
9. Jalankan satu critical mutation dengan data uji yang disetujui.
10. Verifikasi audit dan outbox side effect.
11. Mulai observation window.
12. Putuskan continue atau rollback sebelum deadline.

### Rollback trigger minimum

- Service gagal stabil atau restart loop.
- Migration gagal atau invariant database rusak.
- Login atau portal P0 gagal.
- Cross-tenant leak, auth bypass, atau security header/cookie policy salah.
- Duplicate/lost notification pada smoke flow.
- Error rate atau latency memburuk melewati threshold yang ditetapkan sebelum deploy.
- Backup tidak dapat diverifikasi.
- Operator kehilangan observability yang diperlukan untuk menilai kesehatan.

### Acceptance criteria

- [ ] Hanya artefak yang disetujui production gate yang dipasang.
- [ ] Semua command dan hasil penting dicatat tanpa secret.
- [ ] Tidak ada perubahan manual di luar runbook tanpa incident/change record.
- [ ] Health dan smoke tests lulus.
- [ ] Rollback dimulai segera ketika trigger terpenuhi.
- [ ] Keputusan continue/rollback memiliki approver dan timestamp.
- [ ] Binary/config/database versi sebelum deploy tetap dapat dipulihkan sesuai strategi.

### Definition of Done

- [ ] Cutover selesai atau rollback selesai dengan state diketahui.
- [ ] Evidence cutover tersimpan.
- [ ] Stakeholder menerima status aktual.
- [ ] Handoff ke OPS-003 dilakukan.

### Prompt implementasi

```text
Jalankan OPS-002 hanya setelah PROD-GATE-001 memberi GO tertulis. Gunakan artefak dan checksum yang sama dengan staging rehearsal. Ikuti pre-cutover checklist dan DOC-004 tanpa improvisasi tersembunyi. Catat baseline, backup verification, stop/drain, install, migration, start, health checks, smoke tests, audit/outbox side effects, serta observation window. Terapkan rollback trigger yang telah disetujui dan catat approver serta timestamp keputusan. Jangan menampilkan secret atau session data pada evidence.
```

---

## OPS-003: Post-Deploy Verification dan Incident Readiness

### Status

- Prioritas: P0
- Owner: Operations
- Reviewer: QA Lead, Backend, Product
- Dependensi: OPS-002

### Tujuan

Memastikan release tetap sehat setelah cutover dan tim siap menangani kegagalan tanpa menebak langkah.

### Cakupan verifikasi

- Service uptime/restart, HTTP health, database connectivity, storage, dan migration version.
- Login, portal, critical PJ/KM flow, audit, outbox, serta bot connection.
- Error logs, rate limit behavior, latency baseline, dan notification backlog.
- Backup schedule serta restore verification berikutnya.
- On-call owner, severity, communication, escalation, dan incident log.
- Post-deploy review dan closure.

### Observation window

Durasi dan threshold harus ditetapkan sebelum deploy berdasarkan pola penggunaan aktual. Dokumen ini tidak menetapkan angka palsu. Minimal harus ada:

- pemeriksaan segera setelah start;
- pemeriksaan setelah critical smoke test;
- pemeriksaan pada satu siklus worker/reminder relevan;
- pemeriksaan setelah trafik pengguna pertama;
- keputusan closure atau perpanjangan observasi.

### Acceptance criteria

- [ ] Tidak ada restart loop, panic, migration error, atau permission error.
- [ ] Critical user flows tetap lulus.
- [ ] Outbox backlog bergerak dan tidak menghasilkan duplikasi.
- [ ] Tidak ada kebocoran secret pada log production.
- [ ] Error dan latency dibandingkan dengan baseline yang telah ditetapkan.
- [ ] Bot connection status diketahui dan ada recovery path.
- [ ] Incident owner serta escalation path aktif selama observation window.
- [ ] Setiap anomaly memiliki severity, owner, keputusan, dan timestamp.
- [ ] Closure hanya diberikan setelah observation criteria terpenuhi.

### Incident minimum template

```text
Incident ID:
Start time / detected time:
Release commit:
Severity:
User impact:
Detection source:
Current state:
Actions taken:
Decision owner:
Rollback decision and deadline:
Evidence links:
Next update time:
Resolution and follow-up:
```

### Definition of Done

- [ ] Observation window selesai tanpa unresolved P0/P1 anomaly.
- [ ] Release status dikomunikasikan.
- [ ] Evidence dan incident record tersimpan.
- [ ] Follow-up ticket dibuat untuk setiap temuan yang tidak diselesaikan langsung.

### Prompt implementasi

```text
Kerjakan OPS-003 segera setelah cutover atau rollback. Pantau service, HTTP, database, storage, migration, login, portal, critical PJ/KM flow, audit, outbox, bot status, error log, latency, dan notification backlog selama observation window yang ditetapkan sebelum deploy. Bandingkan dengan baseline dan catat seluruh anomaly. Aktifkan incident template serta escalation path untuk setiap masalah. Jangan menutup release hanya karena process berstatus running; verifikasi outcome bisnis dan data side effect.
```

---

## PROD-GATE-001: Final Production Go/No-Go

### Prinsip

Status default adalah `NO-GO`. Status berubah menjadi `GO` hanya bila seluruh gate wajib lulus untuk satu commit kandidat dan satu artefak immutable.

### Identitas kandidat

| Field | Nilai wajib |
|---|---|
| Release/version | Diisi saat gate |
| Commit SHA | Diisi saat gate |
| Branch/tag | Diisi saat gate |
| Build timestamp | Diisi saat gate |
| Go version | Diisi saat gate |
| Target OS/arch | Diisi saat gate |
| Artifact checksum | Diisi saat gate |
| Database schema version | Diisi saat gate |
| Config manifest version | Diisi saat gate, tanpa secret |
| Staging rehearsal ID | Diisi saat gate |
| QA report/evidence ID | Diisi saat gate |

### Gate A: Source dan change control

- [ ] Working tree kandidat bersih.
- [ ] Commit kandidat telah direview dan bukan direct push tanpa review ke `main`.
- [ ] Semua perubahan dalam scope memiliki ticket atau penjelasan.
- [ ] Tidak ada conflict marker, debug bypass, fixture credential, database, session, binary, atau `.env` ter-commit.
- [ ] Generated/release artefact dapat ditelusuri ke commit SHA.
- [ ] Perubahan setelah freeze telah melalui impact analysis dan re-test.

### Gate B: Build dan automated verification

- [ ] `gofmt` bersih pada file Go berubah.
- [ ] `go test -count=1 ./...` lulus.
- [ ] `go vet ./...` lulus.
- [ ] Race test lulus pada environment yang mendukung, atau pengecualian memiliki risk acceptance tertulis.
- [ ] Linux production build lulus untuk target aktual.
- [ ] API contract collection lulus dari fixture bersih.
- [ ] Hasil memiliki raw output, timestamp, environment, dan commit SHA.

### Gate C: Functional dan integration QA

- [ ] QA-001 sampai QA-014 memiliki evidence final.
- [ ] Seluruh 48 functional requirements terpetakan.
- [ ] Critical PJ, KM, Admin, mahasiswa, dan bot flow lulus.
- [ ] Mobile 390px, keyboard, error state, offline, conflict, dan session expiry lulus.
- [ ] Tidak ada S0, S1, atau S2 terbuka.
- [ ] S3/S4 memiliki owner, workaround, dan target.
- [ ] UAT pilot ditandatangani oleh perwakilan yang ditetapkan.

### Gate D: Security dan privacy

- [ ] Auth, session rotation, invitation, recovery, dan lockout lulus.
- [ ] RBAC serta cross-class isolation lulus negative testing.
- [ ] Rate limit, source identity, trusted proxy, dan multi-instance policy lulus.
- [ ] Production startup fail-closed untuk config tidak aman.
- [ ] Cookie, CORS, CSRF, CSP, HSTS, security headers, dan no-store lulus.
- [ ] Tidak ada raw secret, token, password, code, cookie, atau session pada log/evidence/database yang dilarang.
- [ ] Dependency atau vulnerability findings yang diketahui telah dinilai.
- [ ] Security reviewer memberikan approval tertulis.

### Gate E: Data, migration, backup, dan rollback

- [ ] Migration diuji dari baseline yang mewakili production.
- [ ] Invariant database kritis lulus setelah migration.
- [ ] Backup baru dapat dibuat dan diverifikasi.
- [ ] Restore behavior sesuai keputusan BE-011 dan tidak diklaim melebihi implementasi.
- [ ] Rollback binary dan config dipraktikkan.
- [ ] Database rollback/recovery strategy dipraktikkan atau memiliki batas yang diterima tertulis.
- [ ] Retention, permission, free storage, serta owner backup diketahui.
- [ ] WhatsApp session safety telah diverifikasi.

### Gate F: Deployment dan operations readiness

- [ ] DOC-001 sampai DOC-006 selesai.
- [ ] OPS-001 staging rehearsal lulus.
- [ ] Service unit, user, working directory, restart policy, dan environment source direview.
- [ ] Reverse proxy, DNS, TLS, firewall, timezone, dan trusted proxy benar.
- [ ] Health checks serta critical smoke checklist siap.
- [ ] Monitoring/log access dan threshold observation ditetapkan.
- [ ] Maintenance window dan komunikasi disetujui.
- [ ] Release owner, operator, verifier, incident owner, dan rollback owner tersedia.
- [ ] Rollback trigger serta decision deadline telah disepakati.

### Gate G: Documentation dan support readiness

- [ ] API, config, deployment, rollback, dan user docs sesuai kandidat.
- [ ] Release notes berasal dari diff aktual.
- [ ] Known limitations dan deprecation tercatat.
- [ ] Support memiliki triage, severity, escalation, dan safe evidence guidance.
- [ ] Evidence pack dapat diakses reviewer berwenang.
- [ ] Tidak ada instruksi aktif yang bertentangan atau menunjuk path/command lama.

### Blocker otomatis

Keputusan wajib `NO-GO` bila salah satu kondisi berikut berlaku:

- Ada S0, S1, atau S2 terbuka.
- Ada auth bypass, cross-tenant leak, secret exposure, atau kehilangan/duplikasi data bisnis.
- Test, vet, build, contract suite, atau critical E2E gagal.
- Artefak production berbeda dari artefak staging tanpa rehearsal ulang.
- Commit SHA tidak diketahui atau evidence berasal dari commit lain.
- Backup belum diverifikasi.
- Rollback belum dipraktikkan.
- Migration gagal atau recovery strategy tidak jelas.
- Production config belum lulus fail-closed validation.
- Tidak ada operator atau rollback owner pada window deploy.
- Observability tidak cukup untuk mengetahui apakah deploy sehat.
- Approval hanya berupa klaim tanpa evidence yang dapat dibuka.

### Kebijakan waiver

- Waiver hanya berlaku untuk S3/S4 dan item nonsecurity yang tidak memengaruhi data integrity, access control, recovery, atau critical flow.
- Waiver wajib memuat risiko, dampak pengguna, workaround, owner, target tanggal, approver, dan alasan tidak diperbaiki sebelum release.
- S0 sampai S2, security isolation, secret exposure, backup, migration, dan rollback tidak dapat di-waive.

### Keputusan

| Pilihan | Kondisi |
|---|---|
| `GO` | Seluruh gate wajib lulus, tidak ada blocker otomatis |
| `CONDITIONAL GO` | Hanya S3/S4 dengan waiver valid dan seluruh gate kritis lulus |
| `NO-GO` | Ada blocker atau evidence wajib tidak tersedia |

### Approval record

```text
Decision: GO / CONDITIONAL GO / NO-GO
Release/version:
Commit SHA:
Artifact checksum:
Decision timestamp:
Product approver:
Tech Lead approver:
QA Lead approver:
Security approver:
Operations approver:
Open waivers:
Deployment window:
Rollback decision deadline:
Evidence manifest:
Notes:
```

### Definition of Done

- [ ] Identitas kandidat diisi lengkap.
- [ ] Setiap checklist memiliki evidence reference atau alasan `N/A` yang disetujui.
- [ ] Semua approver menandatangani kandidat yang sama.
- [ ] Keputusan dan timestamp tersimpan sebelum OPS-002 dimulai.
- [ ] Hasil gate tidak diubah setelah deploy; perubahan dicatat sebagai amendment.

### Prompt pelaksanaan gate

```text
Jalankan PROD-GATE-001 untuk satu commit dan artefak kandidat yang dibekukan. Isi identitas kandidat, lalu verifikasi Gate A sampai G menggunakan raw evidence DOC, QA, security, staging rehearsal, backup, migration, rollback, build, dan contract suite. Jangan menerima label PASS atau GO tanpa bukti yang dapat dibuka dan cocok dengan commit SHA. Terapkan blocker otomatis dan kebijakan waiver secara ketat. Catat setiap item sebagai PASS, FAIL, atau N/A dengan alasan serta approver. Hasilkan keputusan GO, CONDITIONAL GO, atau NO-GO beserta timestamp, penanda tangan, evidence manifest, deployment window, dan rollback deadline. Jangan memulai production cutover jika hasil bukan GO atau CONDITIONAL GO yang valid.
```

---

## Urutan pelaksanaan

```text
DOC-001
  -> DOC-002
  -> DOC-003
  -> DOC-004
  -> DOC-005
  -> DOC-006

QA-001 sampai QA-013 + DOC-001 sampai DOC-004
  -> OPS-001

QA-014 + DOC-001 sampai DOC-006 + OPS-001
  -> PROD-GATE-001

PROD-GATE-001 = GO
  -> OPS-002
  -> OPS-003
```

DOC-002, DOC-003, dan DOC-005 dapat berjalan paralel setelah DOC-001 menetapkan sumber kanonis. DOC-006 diselesaikan terhadap commit kandidat. OPS-002 tidak boleh dimulai hanya karena laporan QA menyatakan `GO`; `PROD-GATE-001` tetap wajib memvalidasi semua evidence dan kesiapan operasi.

## Gate penyelesaian dokumentasi dan produksi

- [ ] Enam ticket dokumentasi selesai dan direview.
- [ ] Seluruh perintah, path, variable, route, dan status sesuai implementasi.
- [ ] Tidak ada secret atau data pribadi dalam dokumen/evidence.
- [ ] Staging rehearsal dan rollback drill lulus.
- [ ] QA final dan UAT terhubung ke raw evidence kandidat.
- [ ] Artefak immutable memiliki commit SHA dan checksum.
- [ ] Backup production dapat diverifikasi sebelum cutover.
- [ ] Production gate menghasilkan keputusan tertulis.
- [ ] Cutover hanya dilakukan setelah `GO` yang valid.
- [ ] Post-deploy observation dan incident readiness selesai.
