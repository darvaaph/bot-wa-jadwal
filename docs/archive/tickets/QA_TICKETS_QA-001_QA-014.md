# QA Tickets: QA-001 sampai QA-014

## Tujuan

Dokumen ini memecah pekerjaan Quality Assurance Bot Jadwal menjadi ticket yang dapat dikerjakan, ditinjau, dan dinyatakan lulus secara terpisah. Cakupan mencakup REST API v1, database, bot WhatsApp, dashboard web, keamanan, kompatibilitas, reliabilitas, dan User Acceptance Test.

Dokumen acuan utama:

- [Functional Requirements](product/FUNCTIONAL_REQUIREMENTS.md)
- [Business Rules](product/BUSINESS_RULES.md)
- [Access Control](product/ACCESS_CONTROL.md)
- [Traceability Matrix](product/TRACEABILITY.md)
- [API v1](api/API_V1.md)
- [Panduan Pengujian QA](api/PANDUAN_TESTING_QA.md)
- [Plan API v1](api/PLAN_V1.md)
- [Backend Tickets BE-005, BE-006, BE-009](BACKEND_TICKETS_BE-005_BE-006_BE-009.md)
- [Backend Tickets BE-007, BE-008, BE-010, BE-011](BACKEND_TICKETS_BE-007_BE-008_BE-010_BE-011.md)
- [Backend Tickets BE-012 sampai BE-014](BACKEND_TICKETS_BE-012_BE-013_BE-014.md)

## Ringkasan backlog

| ID | Judul | Prioritas | Dependensi utama | Keluaran utama |
|---|---|---:|---|---|
| QA-001 | Baseline Lingkungan, Data Uji, dan Traceability | P0 | Spec disepakati | Matriks requirement dan fixture deterministik |
| QA-002 | Validasi Kontrak REST API v1 | P0 | QA-001 | Contract suite dan laporan drift |
| QA-003 | Auth, Sesi, Undangan, dan Pemulihan Akun | P0 | BE auth dan BE-013 | Bukti lifecycle identitas dan sesi |
| QA-004 | RBAC, Scope, dan Isolasi Data Kelas | P0 | QA-003 | Matriks akses negatif lintas peran dan kelas |
| QA-005 | Kelas, Semester, Impor, Aktivasi, dan Arsip | P0 | BE semester stabil | Bukti transaksi dan lifecycle semester |
| QA-006 | Jadwal, Konflik, Ruangan, dan Lintas Kelas | P0 | BE-006 dan BE-007 | Regression suite domain jadwal |
| QA-007 | Tugas, Review, Materi, dan Optimistic Lock | P0 | BE-008 | Regression suite lifecycle tugas |
| QA-008 | Portal Mahasiswa dan Kompatibilitas Legacy | P1 | QA-005 sampai QA-007 | Bukti read model dan shim kompatibel |
| QA-009 | Notifikasi, Outbox, Retry, dan Bot WhatsApp | P0 | BE-005 dan BE-010 | Bukti idempotensi serta recovery pengiriman |
| QA-010 | Audit, Backup, Restore, Soft Delete, dan Recovery | P0 | BE-009 dan BE-011 | Bukti integritas serta atomicity operasi |
| QA-011 | Verifikasi Keamanan Produksi | P0 | BE-012 sampai BE-014 | Laporan rate limit, secret, cookie, CORS, CSP |
| QA-012 | Integrasi Frontend, Mobile, State, dan Aksesibilitas | P0 | QA-002 sampai QA-011 | E2E browser dan laporan aksesibilitas |
| QA-013 | Reliabilitas, Concurrency, dan Failure Recovery | P1 | Fitur inti stabil | Stress terarah dan laporan fault injection |
| QA-014 | Full Regression, UAT Pilot, dan Release Sign-off | P0 | QA-001 sampai QA-013 | Paket bukti go/no-go |

## Definisi prioritas dan severity

### Prioritas ticket

- `P0`: wajib lulus sebelum pilot atau rilis.
- `P1`: wajib lulus sebelum rilis umum, kecuali ada risk acceptance tertulis.
- `P2`: peningkatan kualitas yang dapat dijadwalkan setelah pilot jika tidak memengaruhi keamanan atau integritas data.

### Severity defect

| Severity | Kriteria | Contoh | Kebijakan rilis |
|---|---|---|---|
| S0 Blocker | Sistem tidak dapat diuji atau data rusak luas | migrasi gagal, server tidak start, restore merusak database | Rilis berhenti |
| S1 Critical | Kebocoran akses, kehilangan data, duplikasi bisnis, auth bypass | PJ membaca kelas lain, publish ganda, token lama tetap valid | Rilis berhenti |
| S2 Major | Alur utama gagal tanpa workaround aman | review tugas gagal, portal tidak menampilkan publikasi | Wajib diperbaiki sebelum rilis |
| S3 Minor | Fungsi sekunder salah dengan workaround jelas | filter salah urut, pesan validasi kurang tepat | Dapat diterima dengan owner dan tenggat |
| S4 Cosmetic | Tidak memengaruhi hasil atau aksesibilitas | jarak visual kecil, teks tidak konsisten | Tidak memblokir kecuali berulang luas |

Defect aksesibilitas yang menghalangi alur keyboard, pembaca layar, atau pengguna mobile minimal berstatus S2. Kebocoran data, bypass otorisasi, secret exposure, dan korupsi backup selalu minimal S1.

## Aturan umum pelaksanaan QA

1. Gunakan database dan sesi terisolasi. Jangan menyentuh `storage/sesi_bot.db` atau data produksi.
2. Simpan seed, zona waktu, waktu sistem, commit SHA, konfigurasi nonrahasia, dan versi browser pada laporan.
3. Jangan menaruh password, bearer token, portal code, cookie, recovery token, atau nilai secret dalam screenshot dan log.
4. Setiap kasus uji harus menunjuk minimal satu `FR-*`, aturan bisnis, atau kontrak API.
5. Setiap defect harus memiliki langkah reproduksi, expected result, actual result, severity, environment, dan bukti.
6. Ulangi kasus yang gagal setelah perbaikan dan jalankan regression pada area terdampak.
7. Jangan mengubah ekspektasi test hanya agar sesuai implementasi. Jika spec salah atau ambigu, buka decision ticket dan perbarui spec lebih dahulu.
8. Jalankan pengujian dengan `-count=1` untuk mencegah cache menyamarkan hasil.
9. Bukti manual harus dapat diulang oleh reviewer tanpa pengetahuan tersirat.
10. Ticket QA tidak selesai hanya karena happy path lulus.

---

## QA-001: Baseline Lingkungan, Data Uji, dan Traceability

### Status

- Prioritas: P0
- Owner: QA
- Reviewer: Backend, Frontend, Product
- Dependensi: dokumen requirement dan API v1 disepakati

### Tujuan

Membuat baseline pengujian yang deterministik sehingga hasil antar mesin, antar tester, dan antar commit dapat dibandingkan.

### Ruang lingkup

- Inventaris seluruh 48 `FR-*` dan petakan ke ticket QA serta ID test case.
- Tetapkan fixture minimal untuk dua kelas, dua semester, tiga peran, beberapa offering, jadwal bentrok, tugas pada seluruh lifecycle, dan notifikasi pada seluruh status.
- Pisahkan data untuk unit test, API test, browser E2E, bot smoke test, dan UAT.
- Tetapkan prosedur reset database dan seed tanpa menyentuh runtime production.
- Catat timezone `Asia/Jakarta` dan data UTC pembanding.
- Tetapkan format evidence dan defect report.

### Acceptance criteria

- [ ] Seluruh `FR-*` memiliki minimal satu test case positif dan satu negatif, atau alasan tertulis bila tidak relevan.
- [ ] Semua peran memiliki akun uji terisolasi dan scope yang diketahui.
- [ ] Tersedia sedikitnya dua kelas agar uji lintas kelas bukan simulasi satu tenant.
- [ ] Tersedia akun multi-role dan akun tanpa assignment aktif.
- [ ] Reset dan seed dapat diulang dengan hasil ID atau lookup key yang deterministik.
- [ ] Kredensial fixture hanya berlaku di lingkungan test dan tidak masuk konfigurasi production.
- [ ] Baseline menyimpan commit SHA, versi Go, OS, browser, mode server, serta checksum collection.
- [ ] Traceability tidak memiliki requirement tanpa test owner.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-BASE-001 | Seed database bersih dua kali | Hasil logis sama dan tidak menggandakan data |
| TC-BASE-002 | Jalankan test pada database sementara | Tidak ada file session production berubah |
| TC-BASE-003 | Resolusi akun berdasarkan role dan scope | Semua actor cocok dengan fixture |
| TC-BASE-004 | Validasi tanggal UTC dan WIB | Konversi konsisten pada batas hari |
| TC-BASE-005 | Audit traceability | Seluruh 48 FR terpetakan |
| TC-BASE-006 | Bersihkan hasil test | Hanya artefak test yang dihapus |

### Evidence wajib

- Matriks `FR -> TC -> ticket -> status`.
- Manifest fixture tanpa secret.
- Log seed dan reset.
- Daftar environment serta versi alat.

### Definition of Done

- [ ] Baseline direview Backend, Frontend, dan Product.
- [ ] Tester kedua dapat menjalankan setup tanpa instruksi lisan.
- [ ] Tidak ada data atau secret production dalam fixture.
- [ ] Seluruh ticket QA berikutnya merujuk baseline yang sama.

### Prompt eksekusi

```text
Kerjakan QA-001 pada repository Bot Jadwal. Baca Functional Requirements, Business Rules, Access Control, Traceability, API v1, dan panduan QA. Buat inventaris seluruh FR dan petakan masing-masing ke ID test case serta ticket QA-001 sampai QA-014. Siapkan rancangan fixture deterministik untuk dua kelas, dua semester, Admin, KM, PJ, user multi-role, user tanpa assignment, portal LINK/CODE, lifecycle jadwal, lifecycle tugas, serta seluruh status notifikasi. Pastikan database dan session test terisolasi dari storage production. Dokumentasikan perintah setup, reset, seed, versi environment, timezone, dan format evidence. Jangan memasukkan credential atau token nyata. Laporkan gap requirement dan data uji yang masih memerlukan keputusan. Jangan mengubah kontrak produk agar test lulus.
```

---

## QA-002: Validasi Kontrak REST API v1

### Status

- Prioritas: P0
- Owner: QA Backend
- Reviewer: Backend dan Frontend
- Dependensi: QA-001

### Tujuan

Memastikan route, method, status code, envelope, field, tipe data, pagination, filtering, dan error code aktual sesuai API v1 serta kebutuhan frontend.

### Ruang lingkup

- Semua endpoint dalam [API v1](api/API_V1.md).
- Koleksi Postman dan Insomnia.
- Header autentikasi, `Content-Type`, idempotency, deprecation, dan cache policy.
- Error untuk body kosong, JSON rusak, field asing, tipe salah, identifier tidak valid, serta resource tidak ditemukan.
- Konsistensi timestamp RFC3339 dan identifier.
- Drift antara dokumentasi, collection, handler, dan response aktual.

### Acceptance criteria

- [ ] Seluruh route terdokumentasi dapat dipanggil dengan method yang benar.
- [ ] Method yang tidak didukung tidak mengeksekusi mutasi.
- [ ] Response sukses dan gagal memakai envelope kanonis.
- [ ] Error code stabil dan dapat dipakai frontend tanpa parsing pesan manusia.
- [ ] `401`, `403`, `404`, `409`, `422`, dan `429` dibedakan sesuai sebab.
- [ ] Tidak ada response yang mengandung field sensitif atau field internal tak terdokumentasi.
- [ ] Collection tidak bergantung pada ID sisa run sebelumnya.
- [ ] Collection dapat dijalankan berulang setelah reset fixture.
- [ ] Setiap drift dicatat sebagai defect kontrak, bukan diterima diam-diam.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-API-001 | Seluruh happy path API v1 | Status, envelope, dan schema sesuai spec |
| TC-API-002 | JSON malformed dan body kosong | Error terstruktur, server tidak panic |
| TC-API-003 | Tipe field salah dan field wajib hilang | `422` dengan field error berguna |
| TC-API-004 | Method salah | Tidak ada mutasi dan status konsisten |
| TC-API-005 | ID/slug tidak ada | `404` tanpa bocor resource lain |
| TC-API-006 | Header idempotency hilang/duplikat | Sesuai kebijakan endpoint |
| TC-API-007 | Timestamp dan timezone | RFC3339 konsisten, UTC/WIB benar |
| TC-API-008 | Collection run kedua | Tidak gagal akibat state tersembunyi |

### Evidence wajib

- Export hasil collection.
- Daftar endpoint dan status lulus/gagal.
- Contoh response yang sudah disensor.
- Laporan drift spec terhadap implementasi.

### Definition of Done

- [ ] Semua endpoint memiliki contract assertion otomatis atau alasan manual.
- [ ] Tidak ada drift P0/P1 terbuka.
- [ ] Backend dan Frontend menyetujui envelope serta error code final.
- [ ] Panduan QA diselaraskan dengan kontrak final.

### Prompt eksekusi

```text
Kerjakan QA-002. Bandingkan docs/api/API_V1.md, koleksi Postman, koleksi Insomnia, route server, dan response aktual. Jalankan seluruh endpoint dengan fixture QA-001. Verifikasi method, status, envelope, nama field, tipe, timestamp, error code, auth header, idempotency, deprecation, dan cache policy. Tambahkan assertion collection bila aman tanpa menambah npm atau frontend build step. Catat setiap drift dengan endpoint, request tersensor, expected, actual, severity, dan dokumen sumber. Jangan mengubah spec mengikuti bug implementasi. Ulangi collection dari database bersih untuk membuktikan repeatability.
```

---

## QA-003: Auth, Sesi, Undangan, dan Pemulihan Akun

### Status

- Prioritas: P0
- Owner: QA Security
- Reviewer: Backend dan Security
- Dependensi: QA-001, QA-002, BE-013
- Requirement: `FR-ACCESS-002`, `FR-ACCESS-003`, `FR-ACCESS-005`, `FR-ACCESS-006`, `FR-ACCESS-007`

### Tujuan

Memastikan seluruh lifecycle identitas dan session aman, konsisten, serta tidak meninggalkan token lama yang masih dapat digunakan.

### Cakupan uji

- Login berhasil, gagal, lockout, dan pesan generik.
- Logout, expiry idle, expiry absolut, revoke, dan suspend.
- Switch context dan rotasi session.
- Undangan create, resend, accept, expire, revoke, dan single use.
- Recovery manual, password reset, dan peningkatan `session_version`.
- Cookie dan bearer bila keduanya didukung.
- Multi-device dan batas sesi paralel.

### Acceptance criteria

- [ ] Login valid hanya membuat sesi untuk assignment aktif.
- [ ] Respons login gagal tidak membedakan akun tidak ada, password salah, atau akun nonaktif.
- [ ] Logout membuat token lama langsung tidak valid.
- [ ] Switch context menghasilkan token baru dan mencabut atau membatasi token lama sesuai kebijakan.
- [ ] Token undangan dan recovery hanya dapat dipakai sekali.
- [ ] Resend undangan membatalkan token sebelumnya.
- [ ] Suspend dan recovery mencabut sesi yang diwajibkan kebijakan.
- [ ] Expiry PJ/KM dan Admin mengikuti requirement.
- [ ] Token mentah tidak muncul pada database audit, log, atau response sesudah fase pembuatan.
- [ ] Operasi simultan accept/recover hanya menghasilkan satu keberhasilan.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-AUTH-001 | Login valid setiap role | Sesi dan context benar |
| TC-AUTH-002 | Akun tidak ada vs password salah | Response eksternal setara |
| TC-AUTH-003 | Lockout dan cooldown | Batas, durasi, serta audit benar |
| TC-AUTH-004 | Logout lalu reuse token | Ditolak |
| TC-AUTH-005 | Switch ke assignment sendiri | Token baru hanya memiliki scope baru |
| TC-AUTH-006 | Switch ke assignment orang lain | Ditolak tanpa bocor resource |
| TC-AUTH-007 | Accept invitation dua kali | Tepat satu berhasil |
| TC-AUTH-008 | Resend lalu pakai token lama | Token lama ditolak |
| TC-AUTH-009 | Recovery lalu pakai semua sesi lama | Sesi yang diwajibkan telah dicabut |
| TC-AUTH-010 | Idle dan absolute expiry | Expiry tepat pada batas |
| TC-AUTH-011 | Suspend diri sendiri sebagai Admin | Ditolak sesuai kontrak |
| TC-AUTH-012 | Request paralel memakai token yang dirotasi | Tidak menghasilkan dua sesi aktif tak sah |

### Definition of Done

- [ ] Semua lifecycle memiliki happy path, negative path, expiry, replay, dan concurrency test.
- [ ] Tidak ada S0 sampai S2 terbuka.
- [ ] Bukti database membenarkan pencabutan tanpa memuat token mentah.
- [ ] Log keamanan telah diperiksa untuk redaction.

### Prompt eksekusi

```text
Kerjakan QA-003 berdasarkan FR-ACCESS-002/003/005/006/007 dan konfigurasi BE-013. Uji login, lockout, logout, expiry idle/absolut, switch-context, undangan, resend, revoke, accept, recovery, suspend, serta multi-device. Sertakan replay dan request paralel. Periksa state database sebelum dan sesudah tanpa merekam raw token. Verifikasi respons gagal generik, rotasi session, session_version, single-use, dan redaction log. Laporkan setiap sesi lama yang masih valid sebagai S1. Jangan gunakan credential production.
```

---

## QA-004: RBAC, Scope, dan Isolasi Data Kelas

### Status

- Prioritas: P0
- Owner: QA Security
- Reviewer: Backend, Product, Security
- Dependensi: QA-003
- Requirement: `FR-ACCESS-004`, `FR-ACCESS-007`, `FR-CLASS-001`, `FR-CLASS-002`

### Tujuan

Membuktikan bahwa Admin, KM, PJ, dan mahasiswa hanya dapat membaca atau mengubah resource dalam scope yang diizinkan, termasuk ketika identifier resource lain diketahui.

### Ruang lingkup

- Matriks actor x action x resource x scope.
- Isolasi dua kelas dan dua offering.
- Direct object reference melalui path, query, dan body.
- Nested resource dengan parent-child mismatch.
- Filter, pagination, count, export, audit, backup, dan notifikasi.
- Assignment suspended, revoked, expired, serta multi-role.
- Konsistensi `403` atau `404` sesuai kebijakan anti-enumeration.

### Acceptance criteria

- [ ] PJ hanya mengelola offering assignment aktif miliknya.
- [ ] KM hanya mengelola kelas assignment aktif miliknya.
- [ ] Data kelas lain tidak muncul pada list, detail, count, filter, audit, backup, portal, atau notifikasi.
- [ ] Mengganti ID pada path, query, atau body tidak melewati scope check.
- [ ] Parent ID yang sah dengan child ID kelas lain ditolak.
- [ ] Revoked atau suspended assignment langsung kehilangan akses baru.
- [ ] Pergantian context tidak menggabungkan privilege dua role.
- [ ] Penolakan tidak mengungkap keberadaan resource di luar scope.
- [ ] Percobaan ilegal tidak menghasilkan mutasi atau audit sukses.

### Test matrix minimum

| ID | Actor | Target | Ekspektasi |
|---|---|---|---|
| TC-RBAC-001 | PJ A | Offering A | Operasi sesuai izin berhasil |
| TC-RBAC-002 | PJ A | Offering B | Ditolak |
| TC-RBAC-003 | KM A | Kelas A | Operasi kelas berhasil |
| TC-RBAC-004 | KM A | Kelas B | Ditolak tanpa kebocoran |
| TC-RBAC-005 | Mahasiswa portal A | Resource kelas B | Tidak terlihat |
| TC-RBAC-006 | User multi-role context PJ | Operasi KM | Ditolak sampai switch context |
| TC-RBAC-007 | Assignment revoked | Resource lama | Ditolak langsung |
| TC-RBAC-008 | Child B pada parent A | Nested endpoint | Ditolak dan tanpa mutasi |
| TC-RBAC-009 | Filter/count/export | Kelas lain | Tidak ada metadata bocor |
| TC-RBAC-010 | Actor tanpa assignment | Area pengelola | Ditolak |

### Definition of Done

- [ ] Matriks seluruh endpoint mutasi dan data sensitif telah diuji.
- [ ] Minimal dua tenant digunakan.
- [ ] Tidak ada S1 atau S2 terbuka.
- [ ] Bukti mencakup response dan state database sebelum/sesudah.

### Prompt eksekusi

```text
Kerjakan QA-004 sebagai audit dinamis RBAC dan tenant isolation. Bangun matriks actor x endpoint x resource untuk Admin, KM kelas A/B, PJ offering A/B, user multi-role, user revoked, dan mahasiswa portal. Uji penggantian identifier pada path, query, dan body, nested resource mismatch, list, detail, count, filter, audit, backup, serta notification queue. Pastikan penolakan tidak membocorkan keberadaan resource dan tidak menulis mutasi atau audit sukses. Gunakan dua kelas nyata dari fixture. Klasifikasikan setiap cross-tenant leak atau privilege escalation sebagai S1.
```

---

## QA-005: Kelas, Semester, Impor, Aktivasi, dan Arsip

### Status

- Prioritas: P0
- Owner: QA Functional
- Reviewer: Backend dan Product
- Dependensi: QA-001 sampai QA-004
- Requirement: `FR-CLASS-001`, `FR-SEM-001` sampai `FR-SEM-004`

### Tujuan

Memastikan lifecycle kelas dan semester mempertahankan integritas data, validasi impor, serta aturan tepat satu semester aktif per kelas.

### Cakupan uji

- Create/update status kelas.
- Semester draft, tanggal valid, preview, activate, archive.
- Validasi impor per row dan field.
- Apply import secara atomik.
- Aktivasi simultan dan rollback transaksi.
- Akses semester terbit dan arsip melalui portal.
- Data draft atau gagal impor tidak bocor.

### Acceptance criteria

- [ ] `end_date` selalu setelah `start_date`.
- [ ] Draft tidak menggantikan semester aktif.
- [ ] Kesalahan impor memiliki row dan field yang jelas.
- [ ] Error blocking mencegah apply atau aktivasi.
- [ ] Import gagal tidak meninggalkan data aktif atau parsial.
- [ ] Satu kelas tidak pernah memiliki dua semester aktif, termasuk pada request paralel.
- [ ] Aktivasi mengisi `published_at` dan mengarsipkan versi sebelumnya sesuai aturan.
- [ ] PJ tidak dapat mengaktifkan semester.
- [ ] Semester arsip tetap terbaca bila pernah dipublikasikan dan tidak dapat dimutasi biasa.
- [ ] Data draft yang belum diterbitkan tidak terlihat di portal.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-SEM-001 | Buat draft valid | Tidak mengubah active semester |
| TC-SEM-002 | Rentang tanggal invalid | `422`, tidak tersimpan |
| TC-SEM-003 | Import valid | Preview dan apply konsisten |
| TC-SEM-004 | Import campuran valid/invalid | Error per row, tidak ada apply parsial |
| TC-SEM-005 | Apply batch belum tervalidasi | Ditolak |
| TC-SEM-006 | Dua aktivasi paralel | Tepat satu active |
| TC-SEM-007 | Aktivasi gagal di tengah transaksi | State lama utuh |
| TC-SEM-008 | Mutasi semester arsip | Ditolak |
| TC-SEM-009 | Portal semester arsip published | Read-only dan lengkap |
| TC-SEM-010 | Portal draft | Tidak terlihat |

### Definition of Done

- [ ] Lifecycle dan rollback dibuktikan dari API serta database.
- [ ] Concurrency activation test stabil pada beberapa pengulangan.
- [ ] Tidak ada S0 sampai S2 terbuka.
- [ ] Traceability FR-SEM lengkap.

### Prompt eksekusi

```text
Kerjakan QA-005 untuk FR-CLASS-001 dan FR-SEM-001 sampai 004. Uji class lifecycle, semester draft, validasi impor, preview, apply, aktivasi, arsip, dan akses portal. Gunakan import valid, invalid, campuran, duplikat, serta referensi offering yang tidak cocok. Buktikan apply dan activation atomik dengan state database sebelum/sesudah serta dua request aktivasi paralel. Pastikan draft tidak bocor dan hanya satu semester aktif per kelas. Catat setiap partial write atau dua semester aktif sebagai S1.
```

---

## QA-006: Jadwal, Konflik, Ruangan, dan Lintas Kelas

### Status

- Prioritas: P0
- Owner: QA Functional
- Reviewer: Backend, Product, perwakilan KM/PJ
- Dependensi: BE-006, BE-007, QA-005
- Requirement: `FR-SCH-001` sampai `FR-SCH-008`, `FR-ROOM-001` sampai `FR-ROOM-003`

### Tujuan

Memastikan jadwal efektif, versioning pola, konflik, publikasi, pencabutan, ruangan, dan partisipasi lintas kelas mengikuti aturan domain.

### Cakupan uji

- CRUD pola jadwal dan effective range.
- Event regular, replacement, additional, holiday, cancelled.
- Draft, preview, publish, revoke, dan correction.
- Konflik kelas, dosen, ruangan, boundary waktu, dan override reason.
- Idempotency publish dan revoke.
- Kandidat ruangan dan konfirmasi TU.
- Owner serta participant lintas kelas.
- UTC storage dan tampilan WIB pada pergantian hari.

### Acceptance criteria

- [ ] Jadwal efektif memilih versi pola yang benar untuk tanggal target.
- [ ] Waktu selesai harus setelah waktu mulai.
- [ ] Konflik blocking mencegah publish tanpa partial write.
- [ ] Konflik nonblocking hanya dapat dilanjutkan dengan alasan override yang tersimpan.
- [ ] Permanent change membuat versi pola baru dan tidak menulis ulang histori.
- [ ] Retry publish dengan idempotency key sama tidak menggandakan event atau notifikasi.
- [ ] Revoke kedua ditolak dan koreksi notifikasi dibuat sekali.
- [ ] Kandidat ruangan mengecualikan benturan yang diketahui dan menyebut freshness sumber.
- [ ] Konfirmasi TU memiliki status, petugas/sumber, waktu, catatan, dan audit.
- [ ] Participant hanya melihat event setelah accepted.
- [ ] Scope semester owner dan participant tervalidasi.
- [ ] Event lintas tengah malam atau batas timezone diproses konsisten.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-SCH-001 | Jadwal reguler tanpa override | Pola efektif tampil |
| TC-SCH-002 | Replacement pada satu tanggal | Menggantikan sesi asal saja |
| TC-SCH-003 | Additional tanpa pola asal | Valid bila field minimum lengkap |
| TC-SCH-004 | Blocking class/lecturer/room conflict | Publish ditolak |
| TC-SCH-005 | Nonblocking conflict tanpa reason | Ditolak |
| TC-SCH-006 | Nonblocking conflict dengan reason | Publish dan reason teraudit |
| TC-SCH-007 | Retry publish paralel | Satu event dan satu pesan bisnis |
| TC-SCH-008 | Permanent schedule change | Versi baru berlaku sejak effective date |
| TC-SCH-009 | Revoke dan revoke ulang | Pertama sukses, kedua ditolak |
| TC-SCH-010 | Room candidate dan stale source | Hasil serta disclaimer benar |
| TC-SCH-011 | TU confirmation lifecycle | Hanya CONFIRMED memenuhi syarat |
| TC-SCH-012 | Invite/accept/decline/remove participant | Visibility mengikuti lifecycle |
| TC-SCH-013 | Owner dan participant beda semester | Validasi periode berlaku |
| TC-SCH-014 | UTC/WIB pada batas hari | Tanggal efektif benar |

### Definition of Done

- [ ] Semua tipe event dan transisi lifecycle diuji.
- [ ] Konflik diuji pada boundary bersinggungan dan overlap nyata.
- [ ] Idempotency serta concurrency lulus.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-006 berdasarkan FR-SCH-001 sampai 008, FR-ROOM-001 sampai 003, BE-006, dan BE-007. Uji pattern versioning, seluruh event type, preview, conflict blocking/nonblocking, override reason, publish idempotency, revoke, correction, room candidates, TU confirmation, serta cross-class participation. Sertakan boundary waktu, overlap, waktu bersebelahan, UTC/WIB, semester owner/participant, retry, dan request paralel. Periksa event, offering relation, room confirmation, notification outbox, dan audit dalam satu bukti transaksi. Setiap publikasi ganda, konflik blocking lolos, atau histori tertimpa adalah S1.
```

---

## QA-007: Tugas, Review, Materi, dan Optimistic Lock

### Status

- Prioritas: P0
- Owner: QA Functional
- Reviewer: Backend, Product, perwakilan KM/PJ
- Dependensi: BE-008, QA-005
- Requirement: `FR-TASK-001` sampai `FR-TASK-007`, `FR-OPS-004`

### Tujuan

Memastikan lifecycle tugas dan materi konsisten, review atomik, versioning aman, serta portal hanya menampilkan versi yang berhak dibaca.

### Cakupan uji

- Draft, preview, publish oleh PJ/KM.
- Approve, changes requested, revoke.
- Patch dengan optimistic locking.
- Complete, archive, restore, dan overdue.
- Filter, grouping, ordering deadline.
- Materi umum kelas dan offering.
- Review terhadap versi aktif dan request paralel.

### Acceptance criteria

- [ ] Draft boleh disimpan tidak lengkap sesuai kontrak, tetapi publish memerlukan field minimum.
- [ ] Publish PJ menghasilkan `NOT_REVIEWED`; publish KM menghasilkan review `APPROVED` pada versi aktif.
- [ ] `CHANGES_REQUESTED` dan `REVOKED` membutuhkan note serta menarik tugas dari portal secara atomik.
- [ ] Review untuk versi lama ditolak dengan conflict yang dapat dipulihkan frontend.
- [ ] Update visible menaikkan version dan mempertahankan histori.
- [ ] Deadline lampau tampil sebagai overdue, bukan hilang.
- [ ] Complete, archive, restore, dan revoke tidak mencampur makna lifecycle.
- [ ] PJ hanya membuat tugas atau materi pada offering miliknya.
- [ ] Materi umum KM selalu memiliki class scope.
- [ ] Dua mutasi paralel pada version sama menghasilkan satu sukses dan satu conflict.
- [ ] Input pengguna tersedia pada response conflict sesuai kontrak frontend.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-TASK-001 | Simpan draft parsial | Draft tersimpan, tidak tampil di portal |
| TC-TASK-002 | Publish tanpa field wajib | `422`, draft tidak terbit |
| TC-TASK-003 | Publish oleh PJ | `PUBLISHED` dan `NOT_REVIEWED` |
| TC-TASK-004 | Publish oleh KM | Review `APPROVED` untuk versi aktif |
| TC-TASK-005 | Approve versi aktif | Review dan state konsisten |
| TC-TASK-006 | Changes requested tanpa note | Ditolak |
| TC-TASK-007 | Revoke | Portal menarik tugas, audit tersimpan |
| TC-TASK-008 | Update versi usang | `409`, tidak ada overwrite |
| TC-TASK-009 | Dua update paralel | Satu sukses, satu conflict |
| TC-TASK-010 | Deadline lampau | Overdue tetap terlihat |
| TC-TASK-011 | Complete/archive/restore | Status dan histori tidak hilang |
| TC-TASK-012 | Materi offering lintas scope | Ditolak |
| TC-TASK-013 | Review saat versi berubah | Review stale ditolak |

### Definition of Done

- [ ] Seluruh state transition valid dan invalid diuji.
- [ ] Atomicity review dibuktikan pada failure path.
- [ ] Concurrency test dapat diulang.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-007 berdasarkan FR-TASK-001 sampai 007, FR-OPS-004, dan BE-008. Uji draft, publish PJ/KM, review approve/changes requested/revoke, update version, complete, archive, restore, overdue, filter, ordering, dan materi. Sertakan stale review, stale edit, dua request paralel, note wajib, scope offering, serta kegagalan di tengah transaksi. Verifikasi API, portal, audit, notification outbox, dan histori version. Klasifikasikan lost update, review pada versi salah, atau partial state sebagai S1.
```

---

## QA-008: Portal Mahasiswa dan Kompatibilitas Legacy

### Status

- Prioritas: P1
- Owner: QA Integration
- Reviewer: Backend dan Frontend
- Dependensi: QA-005, QA-006, QA-007
- Requirement: `FR-ACCESS-001`, `FR-SCH-001`, `FR-TASK-004`, `FR-TASK-007`

### Tujuan

Memastikan portal hanya-baca menampilkan read model yang benar dan shim legacy mempertahankan perilaku yang memang masih dijanjikan tanpa membuka mutasi lama.

### Cakupan uji

- Portal LINK dan CODE bila keduanya aktif.
- Summary, schedule, tasks, task detail, changes, materials.
- Published, draft, revoked, archived, overdue, serta cross-class event.
- Rotasi portal code dan pencabutan sesi versi lama.
- Slug, token, dan resource lintas kelas.
- Tujuh endpoint shim legacy, header deprecation, dan larangan legacy write sesuai ADR.

### Acceptance criteria

- [ ] Portal tidak menampilkan draft, revoked, atau data sensitif.
- [ ] Archived published semester tetap dapat dibaca secara read-only sesuai kebijakan.
- [ ] Jadwal replacement/cancelled menghasilkan tampilan efektif yang benar.
- [ ] Tugas diurutkan deadline dan overdue tetap terlihat.
- [ ] Rotasi code mencabut portal session versi lama.
- [ ] Code salah tidak mengungkap apakah slug atau kelas ada.
- [ ] Portal kelas A tidak dapat membaca resource kelas B melalui ID langsung.
- [ ] Legacy read yang masih didukung cocok dengan model kanonis.
- [ ] Legacy write yang telah dipensiunkan ditolak dan tidak bermutasi.
- [ ] Response legacy memiliki header deprecation yang disepakati.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-PORT-001 | Summary pada hari ada jadwal/tugas | Data efektif dan batas jumlah benar |
| TC-PORT-002 | Replacement/cancelled event | Jadwal asal tidak tampil salah |
| TC-PORT-003 | Draft/revoked task direct ID | Tidak terlihat |
| TC-PORT-004 | Archived published semester | Read-only dapat dibaca |
| TC-PORT-005 | Rotate portal code | Sesi lama ditolak |
| TC-PORT-006 | Code/slug salah | Respons anti-enumeration konsisten |
| TC-PORT-007 | Cross-class direct ID | Ditolak atau not found |
| TC-PORT-008 | Legacy read | Data sesuai kanonis |
| TC-PORT-009 | Legacy write retired | Ditolak tanpa mutasi |
| TC-PORT-010 | Deprecation header | Hadir sesuai ADR |

### Definition of Done

- [ ] Enam read model portal telah diuji.
- [ ] Portal code lifecycle lulus.
- [ ] Semua shim memiliki keputusan pass, defect, atau deprecated.
- [ ] Tidak ada kebocoran data atau S1/S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-008 untuk portal mahasiswa dan shim legacy. Uji portal LINK/CODE, summary, schedule, tasks, task detail, changes, materials, archive read-only, rotasi portal code, serta isolasi lintas kelas. Verifikasi draft/revoked tidak bocor dan read model efektif sesuai jadwal/tugas kanonis. Jalankan seluruh endpoint legacy yang tercantum pada API v1, periksa deprecation header, parity read, dan penolakan write yang telah dipensiunkan. Jangan menganggap response 200 benar tanpa membandingkan isi dengan database kanonis.
```

---

## QA-009: Notifikasi, Outbox, Retry, dan Bot WhatsApp

### Status

- Prioritas: P0
- Owner: QA Integration
- Reviewer: Backend dan Operator Bot
- Dependensi: BE-005, BE-010, QA-006, QA-007
- Requirement: `FR-NOTIF-001` sampai `FR-NOTIF-006`, `FR-OPS-001`

### Tujuan

Membuktikan bahwa publikasi web tidak bergantung pada koneksi WhatsApp, pesan bisnis tidak hilang atau ganda, dan retry mempertahankan lifecycle yang benar.

### Cakupan uji

- Daily schedule, task reminder, schedule change, correction, dan replacement reminder.
- Durable outbox pada restart proses dan koneksi bot putus.
- Status pending, processing, sent, failed, cancelled, dan retry policy final.
- Idempotency key, lease/claim, backoff, stale processing recovery.
- Perubahan atau revoke sebelum pesan dikirim.
- Retry admin dan penolakan retry status tidak sah.
- Pemisahan kelas/channel dan timezone pengiriman.

### Acceptance criteria

- [ ] Transaksi publikasi domain dan enqueue outbox mengikuti atomicity yang disepakati.
- [ ] WhatsApp offline tidak membatalkan publikasi web.
- [ ] Restart setelah commit tidak menghilangkan pesan pending.
- [ ] Dua worker tidak mengirim pesan bisnis yang sama dua kali.
- [ ] Retry tidak membuat row pesan bisnis baru tanpa alasan lifecycle.
- [ ] Pesan usang dibatalkan atau diganti sebelum pengiriman.
- [ ] Revoke menghasilkan correction yang terhubung ke publikasi asal.
- [ ] Failed dapat dipulihkan sesuai policy; sent tidak dapat di-retry sembarang.
- [ ] Pesan memuat kelas, mata kuliah, waktu, dosen, ruang, atau deadline sesuai jenisnya.
- [ ] Data kelas A tidak dikirim ke channel kelas B.
- [ ] Waktu pengingat memakai timezone dan class settings aktif.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-NOTIF-001 | Publish saat bot online | Satu outbox dan satu send |
| TC-NOTIF-002 | Publish saat bot offline | Domain sukses, pesan tetap pending/failed recoverable |
| TC-NOTIF-003 | Restart sebelum worker send | Pesan tetap diproses setelah restart |
| TC-NOTIF-004 | Dua worker claim row sama | Maksimal satu pengiriman efektif |
| TC-NOTIF-005 | Crash setelah claim | Lease kedaluwarsa dan pesan pulih |
| TC-NOTIF-006 | Retry failed | Attempt bertambah, pesan bisnis tetap sama |
| TC-NOTIF-007 | Retry sent | Ditolak |
| TC-NOTIF-008 | Update sebelum send | Payload lama dibatalkan/diganti |
| TC-NOTIF-009 | Revoke published event | Correction terhubung dan tidak ganda |
| TC-NOTIF-010 | Reminder timezone boundary | Waktu target benar |
| TC-NOTIF-011 | Cross-class channel | Tidak ada salah tujuan |
| TC-NOTIF-012 | Task completed/archived | Tidak masuk reminder aktif |

### Definition of Done

- [ ] Seluruh jenis pesan memiliki content assertion.
- [ ] Restart, offline, concurrency, dan retry telah diuji.
- [ ] Tidak ada lost message, duplicate business message, atau cross-class delivery.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-009 berdasarkan FR-NOTIF-001 sampai 006, FR-OPS-001, BE-005, dan BE-010. Uji durable outbox, seluruh jenis pesan, bot online/offline, restart, dua worker, lease timeout, retry, stale message replacement, revoke correction, timezone, serta channel isolation. Gunakan fake sender atau nomor test, bukan penerima production. Buktikan relasi domain event, notification message, attempt, dan audit. Setiap lost message, duplicate send, atau pesan ke kelas salah adalah S1.
```

---

## QA-010: Audit, Backup, Restore, Soft Delete, dan Recovery

### Status

- Prioritas: P0
- Owner: QA Data Integrity
- Reviewer: Backend, Security, Operations
- Dependensi: BE-009, BE-011, QA-004
- Requirement: `FR-AUDIT-001` sampai `FR-AUDIT-003`, `FR-OPS-002` sampai `FR-OPS-004`

### Tujuan

Memastikan operasi sensitif memiliki jejak audit kanonis, backup terverifikasi, restore aman sesuai keputusan produk, dan recovery tidak menghapus histori.

### Cakupan uji

- Audit success/failure policy, actor, active context, class, entity, action, before/after, reason, timestamp.
- Atomicity mutasi dan audit.
- Filter audit sesuai scope.
- Redaction secret dan data sensitif.
- Backup class/global, checksum, manifest, dan status lifecycle.
- Restore verify-only atau full restore sesuai keputusan BE-011.
- Corrupt package, wrong scope, wrong version, failure rollback.
- Soft delete, restore, retention, dan optimistic lock.

### Acceptance criteria

- [ ] Setiap mutasi wajib menghasilkan audit kanonis tepat satu kali.
- [ ] Mutasi gagal tidak menghasilkan audit sukses.
- [ ] Jika audit wajib gagal, mutasi domain ikut rollback.
- [ ] Audit actor multi-role menyimpan active context yang benar.
- [ ] Before/after dapat dibandingkan dan tidak berisi secret.
- [ ] PJ dan KM hanya melihat audit sesuai scope.
- [ ] Backup checksum cocok dan file snapshot konsisten.
- [ ] Paket salah scope, corrupt, atau incompatible ditolak.
- [ ] Full restore, bila dipilih, membuat restore point dan rollback utuh saat gagal.
- [ ] Verify-only tidak mengklaim data telah dipulihkan.
- [ ] Soft delete menghilangkan data dari list aktif tetapi mempertahankan histori.
- [ ] Restore data tidak menghapus audit delete.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-AUD-001 | Mutasi sukses | Satu audit lengkap |
| TC-AUD-002 | Mutasi validation/auth failure | Tidak ada audit sukses palsu |
| TC-AUD-003 | Simulasi audit writer gagal | Domain rollback bila audit wajib |
| TC-AUD-004 | User multi-role | Actor dan context tepat |
| TC-AUD-005 | Filter audit lintas scope | Tidak bocor |
| TC-AUD-006 | Scan field sensitif | Tidak ada raw secret/token/password |
| TC-BACKUP-001 | Backup class/global | Scope, manifest, checksum benar |
| TC-BACKUP-002 | File corrupt | Verifikasi gagal aman |
| TC-BACKUP-003 | Paket scope/version salah | Ditolak |
| TC-BACKUP-004 | Restore failure injection | State awal pulih utuh |
| TC-OPS-001 | Soft delete dan list | Hilang dari aktif, histori ada |
| TC-OPS-002 | Restore soft-deleted | Data kembali, audit delete tetap ada |
| TC-OPS-003 | Concurrent version update | Satu sukses, satu conflict |

### Definition of Done

- [ ] Seluruh mutasi kritis terwakili dalam audit regression.
- [ ] Restore behavior cocok dengan keputusan final BE-011.
- [ ] Recovery drill berhasil pada database salinan.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-010 berdasarkan FR-AUDIT-001 sampai 003, FR-OPS-002 sampai 004, BE-009, dan keputusan final BE-011. Audit mutasi sukses, penolakan, kegagalan transaksi, actor multi-role, scope filter, before/after, reason, dan redaction. Uji backup class/global, checksum, corrupt package, wrong scope/version, serta restore verify-only atau full restore sesuai kontrak. Untuk full restore, lakukan failure injection dan buktikan restore point serta rollback. Uji soft delete, restore, retention, dan optimistic locking. Jangan menjalankan restore pada database atau session production.
```

---

## QA-011: Verifikasi Keamanan Produksi

### Status

- Prioritas: P0
- Owner: QA Security
- Reviewer: Backend, Operations, Security
- Dependensi: BE-012, BE-013, BE-014, QA-003, QA-004

### Tujuan

Memverifikasi hardening rate limit, source identity, secret configuration, token/cookie, CORS, CSRF, CSP, headers, cache, serta log redaction sebelum deployment production.

### Cakupan uji

- Rate limit login, portal code, invitation, recovery, retry, backup, dan endpoint sensitif lain.
- Source normalization dan trusted proxy.
- Persistent limiter serta perilaku multi-instance.
- Startup validation environment production.
- Cookie flags, lifetime, scope, rotation, dan deletion.
- Exact-origin CORS dan preflight.
- CSRF untuk cookie-authenticated mutation.
- CSP terhadap CDN yang benar-benar dipakai web.
- HSTS, frame protection, MIME sniffing, referrer, permissions, dan no-store.
- Secret/token leakage pada log, error, audit, database, dan browser storage.

### Acceptance criteria

- [ ] Perubahan source port tidak mereset rate limit.
- [ ] Forwarding header dari proxy tidak terpercaya diabaikan.
- [ ] Trusted proxy chain diproses sesuai konfigurasi.
- [ ] Restart atau instance kedua tidak menghapus limit yang wajib persisten.
- [ ] `429` konsisten dan memiliki `Retry-After` yang valid.
- [ ] Production gagal start bila hash key, allowed origin, secure cookie, atau proxy trust tidak aman.
- [ ] Cookie auth memakai `Secure`, `HttpOnly`, dan `SameSite` sesuai arsitektur final.
- [ ] Origin asing, scheme berbeda, atau port berbeda tidak mendapat credentialed CORS.
- [ ] Preflight hanya mengizinkan method dan header yang diperlukan.
- [ ] Mutation cookie-authenticated memiliki proteksi CSRF yang teruji.
- [ ] CSP tidak memakai wildcard luas dan tidak merusak halaman utama.
- [ ] HSTS hanya dikirim pada production HTTPS yang tervalidasi.
- [ ] Response auth/token sensitif memakai `Cache-Control: no-store`.
- [ ] Tidak ada raw credential, token, code, cookie, atau source identity pada log.

### Test matrix minimum

| ID | Skenario | Hasil yang diharapkan |
|---|---|---|
| TC-SEC-001 | Login failures dari source sama, port berubah | Tetap satu bucket |
| TC-SEC-002 | Spoof `X-Forwarded-For` tanpa trusted proxy | Tidak dipercaya |
| TC-SEC-003 | Trusted proxy valid | Client identity terambil benar |
| TC-SEC-004 | Restart/multi-instance limiter | Counter konsisten sesuai policy |
| TC-SEC-005 | Konfigurasi production tidak lengkap | Startup gagal jelas tanpa bocor secret |
| TC-SEC-006 | Inspect Set-Cookie | Flags, path, lifetime benar |
| TC-SEC-007 | Origin exact match | Diizinkan |
| TC-SEC-008 | Scheme/port/subdomain asing | Ditolak |
| TC-SEC-009 | CSRF cross-site mutation | Ditolak |
| TC-SEC-010 | Browser CSP smoke | Tidak ada resource wajib diblokir |
| TC-SEC-011 | Security headers dev/prod | Policy sesuai environment |
| TC-SEC-012 | Log and database secret scan | Tidak ada raw secret |
| TC-SEC-013 | Auth response cache | `no-store` hadir |
| TC-SEC-014 | Rate-limit expiry boundary | Akses pulih tepat sesuai policy |

### Definition of Done

- [ ] Seluruh acceptance criteria BE-012 sampai BE-014 memiliki bukti QA.
- [ ] Test dilakukan pada HTTP development dan HTTPS-like staging di belakang proxy yang representatif.
- [ ] Tidak ada S0 sampai S2 security defect terbuka.
- [ ] Relaxation CSP/CORS memiliki owner, alasan, dan tenggat.

### Prompt eksekusi

```text
Kerjakan QA-011 sebagai security verification untuk BE-012, BE-013, dan BE-014. Uji rate limiting dengan source port berubah, spoofed forwarding headers, trusted proxy, restart, dan multi-instance. Uji startup validation production, keyed hashing, cookie flags, session rotation, exact-origin CORS, preflight, CSRF, CSP, HSTS, Permissions-Policy, X-Content-Type-Options, frame policy, Referrer-Policy, dan Cache-Control no-store. Jalankan browser smoke test pada halaman app, KM, dan Admin. Scan log, audit, database, dan browser storage untuk raw secret/token/code. Sensor seluruh evidence. Laporkan bypass atau credential leak sebagai S1.
```

---

## QA-012: Integrasi Frontend, Mobile, State, dan Aksesibilitas

### Status

- Prioritas: P0
- Owner: QA Frontend
- Reviewer: Frontend, Product, Accessibility reviewer
- Dependensi: QA-002 sampai QA-011
- Requirement: `FR-UX-001` sampai `FR-UX-004` dan seluruh user flow utama

### Tujuan

Memastikan frontend nyata menggunakan API v1 dengan benar pada semua role, state, ukuran layar, dan metode input.

### Cakupan uji

- Login, context selection, dashboard PJ/KM/Admin, portal mahasiswa.
- Semester, schedule, task, material, notification, audit, dan operasi admin yang tersedia di UI.
- Loading, empty, success, validation error, denied, unauthenticated, offline, conflict, stale data, dan server error.
- Lebar 390px serta desktop yang didukung.
- Keyboard, focus, Escape, dialog, label, accessible name, dan status nonwarna.
- Refresh, back/forward, deep link, session expiry, dan duplicate submission.
- Tidak ada fallback data palsu atau localStorage sebagai sumber data v1.

### Acceptance criteria

- [ ] Setiap UI action memanggil endpoint dan payload sesuai contract suite.
- [ ] Pengguna tidak melihat sukses sebelum server mengonfirmasi.
- [ ] Error mempertahankan input yang masih aman digunakan.
- [ ] `401`, `403`, `404`, `409`, `422`, dan `429` memiliki state serta langkah berikutnya yang tepat.
- [ ] Conflict `409` menampilkan versi terbaru dan menjaga perubahan pengguna.
- [ ] Loading mencegah duplicate submission tanpa membuat UI macet.
- [ ] Layout 390px tidak memiliki horizontal overflow yang menghalangi tugas.
- [ ] Target sentuh utama minimal 44 x 44 piksel.
- [ ] Seluruh alur utama dapat diselesaikan dengan keyboard.
- [ ] Focus terlihat dan kembali ke trigger setelah dialog ditutup.
- [ ] Dialog dapat ditutup dengan Escape bila aman.
- [ ] Status tidak hanya dibedakan dengan warna.
- [ ] Browser console tidak memiliki error JS atau pelanggaran CSP yang tidak diterima.
- [ ] Tidak ada token atau password disimpan pada localStorage.

### Skenario E2E minimum

| ID | Alur | Hasil yang diharapkan |
|---|---|---|
| TC-E2E-001 | Login PJ, buat dan publish tugas | Portal berubah, KM melihat review |
| TC-E2E-002 | KM request changes | Tugas hilang dari portal, PJ melihat note |
| TC-E2E-003 | PJ revisi, KM approve | Versi dan review tepat |
| TC-E2E-004 | Draft, preview, publish jadwal pengganti | Jadwal efektif dan notifikasi benar |
| TC-E2E-005 | KM revoke jadwal | Portal pulih dan correction dibuat |
| TC-E2E-006 | Switch role/context | Data dan navigasi mengikuti scope baru |
| TC-E2E-007 | Session expiry saat form terbuka | Input aman, re-auth jelas |
| TC-E2E-008 | Edit bersamaan menghasilkan 409 | Konflik dapat dipulihkan |
| TC-E2E-009 | Offline saat submit | Tidak ada false success/duplicate |
| TC-E2E-010 | Keyboard-only critical flows | Dapat diselesaikan |
| TC-E2E-011 | Mobile 390px one-hand flow | Tidak overflow, kontrol terjangkau |
| TC-E2E-012 | Empty/denied/rate-limited states | Pesan dan tindakan tepat |

### Browser minimum

- Chrome versi stabil pada desktop.
- Chrome responsive emulation 390px.
- Satu browser Chromium kedua atau Edge untuk smoke compatibility.
- Browser lain hanya diwajibkan bila Product menetapkannya sebagai target dukungan.

### Definition of Done

- [ ] Semua critical user flow memiliki bukti video atau screenshot tersensor.
- [ ] State matrix lengkap untuk halaman utama.
- [ ] Keyboard dan mobile test lulus.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-012 pada frontend nyata yang terhubung ke API v1, tanpa mock dan tanpa fallback localStorage. Uji alur PJ publish tugas -> KM review -> portal update, serta jadwal draft -> preview -> publish -> revoke -> correction. Uji seluruh role dan state loading, empty, error, denied, unauthenticated, offline, conflict, stale, rate-limited, serta session expiry. Jalankan pada desktop dan 390px. Audit keyboard, focus, Escape, labels, accessible names, status nonwarna, dan target sentuh 44x44. Periksa network request, browser storage, console, serta CSP. Simpan evidence tersensor dan buat defect yang dapat direproduksi.
```

---

## QA-013: Reliabilitas, Concurrency, dan Failure Recovery

### Status

- Prioritas: P1
- Owner: QA Reliability
- Reviewer: Backend dan Operations
- Dependensi: fitur inti stabil dan QA-003 sampai QA-011

### Tujuan

Menemukan race, deadlock, partial write, resource leak, dan perilaku recovery yang tidak terlihat pada pengujian fungsional biasa.

### Ruang lingkup

- `go test -race` pada package yang mendukung dan tidak bergantung pada batas platform.
- Concurrent login, switch context, invitation accept, semester activate, schedule publish, task update/review, notification claim/retry, backup, dan restore.
- Database busy/locked, disk write failure yang dapat disimulasikan, network sender failure, process restart, dan timeout.
- Repeat test untuk area rawan flake.
- Baseline latency endpoint kritis pada fixture representatif, bukan benchmark production palsu.
- Goroutine dan connection cleanup pada shutdown.

### Acceptance criteria

- [ ] Tidak ada data race yang dapat direproduksi pada package inti.
- [ ] Operasi ber-idempotency tidak menggandakan hasil pada request paralel.
- [ ] Optimistic lock menghasilkan satu pemenang tanpa lost update.
- [ ] Database lock menghasilkan error terkontrol atau retry sesuai policy, bukan panic.
- [ ] Kegagalan dependency tidak meninggalkan partial state.
- [ ] Restart memulihkan outbox dan tidak mengaktifkan ulang sesi yang telah dicabut.
- [ ] Shutdown tidak meninggalkan row processing permanen tanpa recovery.
- [ ] Tidak ada goroutine leak yang terlihat pada test berulang terarah.
- [ ] Latency baseline dan dataset dicatat tanpa mengklaim kapasitas production.

### Test matrix minimum

| ID | Skenario | Invariant |
|---|---|---|
| TC-REL-001 | Dua accept invitation | Satu assignment aktif |
| TC-REL-002 | Dua activation semester | Satu active semester |
| TC-REL-003 | Publish schedule paralel | Satu event/outbox per key |
| TC-REL-004 | Update task version sama | Satu sukses, satu conflict |
| TC-REL-005 | Review saat task berubah | Tidak mereview versi stale |
| TC-REL-006 | Dua worker outbox | Tidak duplicate send |
| TC-REL-007 | DB locked | Error terkontrol, state utuh |
| TC-REL-008 | Sender timeout | Attempt dan retry state benar |
| TC-REL-009 | Restart saat processing | Recovery sesuai lease |
| TC-REL-010 | Repeated suite | Tidak flaky tanpa penyebab |

### Perintah verifikasi

```powershell
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
go build -o bin/bot.exe ./cmd/bot
```

Catatan: bila race detector tidak dapat dijalankan pada environment Windows yang tersedia, jalankan pada CI atau environment Go yang mendukung dan lampirkan hasilnya. Jangan menandai lulus hanya karena test dilewati.

### Definition of Done

- [ ] Concurrency invariant kritis memiliki automated test.
- [ ] Failure injection memiliki cleanup dan tidak menyentuh production state.
- [ ] Flaky test diselesaikan atau dikarantina dengan owner serta tiket, bukan diabaikan.
- [ ] Tidak ada S0 sampai S2 terbuka.

### Prompt eksekusi

```text
Kerjakan QA-013 sebagai reliability and concurrency pass. Jalankan test normal, race detector pada environment yang mendukung, repeated tests, dan failure injection terarah. Fokus pada invitation accept, session rotation, semester activation, schedule publish, task update/review, outbox claim/retry, backup, serta restore. Simulasikan database busy, sender timeout, restart saat processing, dan dua worker. Periksa invariant database dan cleanup. Catat dataset serta latency baseline tanpa menyebutnya sebagai kapasitas production. Setiap lost update, duplicate business action, deadlock, atau partial write adalah minimal S1.
```

---

## QA-014: Full Regression, UAT Pilot, dan Release Sign-off

### Status

- Prioritas: P0
- Owner: QA Lead
- Reviewer: Product, Backend, Frontend, Operations, perwakilan pengguna
- Dependensi: QA-001 sampai QA-013

### Tujuan

Menghasilkan keputusan go/no-go yang dapat diaudit untuk pilot satu kelas dan rilis berikutnya.

### Entry criteria

- Semua ticket backend dalam scope telah berada pada commit kandidat rilis.
- QA-001 sampai QA-013 selesai atau memiliki waiver tertulis.
- Database staging telah di-reset dan di-seed dari prosedur yang disetujui.
- Build kandidat memiliki commit SHA dan konfigurasi deployment tercatat.
- Rollback point tersedia sebelum UAT.

### Cakupan regression

- Semua automated Go tests.
- Seluruh collection API v1.
- Critical E2E browser flows.
- Bot WhatsApp smoke test memakai sesi dan nomor uji.
- Security smoke untuk auth, RBAC, rate limit, CORS, cookie, dan headers.
- Backup/restore drill sesuai keputusan BE-011.
- Deployment start, shutdown, restart, dan rollback.
- UAT satu kelas pilot dengan KM, PJ, dan mahasiswa perwakilan.

### Skenario UAT minimum

1. KM mengaktifkan semester pilot.
2. PJ melihat offering yang tepat dan tidak melihat offering lain.
3. PJ membuat serta menerbitkan tugas.
4. KM meninjau tugas dan meminta perubahan.
5. PJ memperbaiki, lalu KM menyetujui versi baru.
6. Mahasiswa melihat tugas yang benar pada portal.
7. PJ membuat jadwal pengganti, memeriksa konflik, dan menerbitkan.
8. KM mencabut event dan mahasiswa melihat jadwal efektif kembali.
9. Notifikasi serta koreksi masuk ke channel uji tepat satu kali.
10. Audit menampilkan actor, context, waktu, before/after, dan reason.
11. Backup kandidat dibuat dan diverifikasi.
12. Rollback deployment dipraktikkan pada staging/dev.

### Exit criteria

- [ ] `go test -count=1 ./...` lulus pada commit kandidat.
- [ ] `go vet ./...` lulus.
- [ ] Build binary lulus.
- [ ] Contract suite dan critical E2E lulus.
- [ ] Tidak ada S0, S1, atau S2 terbuka.
- [ ] S3/S4 terbuka memiliki owner, workaround, dampak, dan target penyelesaian.
- [ ] Seluruh P0 requirement memiliki evidence.
- [ ] UAT pilot ditandatangani perwakilan KM, PJ, mahasiswa, Product, dan QA.
- [ ] Backup, restore/verify, serta rollback telah dipraktikkan.
- [ ] Known limitations dan perubahan konfigurasi production terdokumentasi.
- [ ] Release decision menyebut commit SHA dan waktu keputusan.

### Paket evidence release

- Test summary per ticket QA.
- Traceability final `FR -> TC -> result -> evidence`.
- Output test, vet, dan build.
- Collection report.
- Screenshot/video E2E yang telah disensor.
- Security verification report.
- UAT sign-off.
- Backup checksum dan hasil recovery drill.
- Daftar defect terbuka serta waiver.
- Go/no-go minutes dan rollback owner.

### Aturan keputusan

- `GO`: seluruh exit criteria wajib terpenuhi.
- `CONDITIONAL GO`: hanya S3/S4 dengan waiver tertulis; tidak berlaku untuk security, data integrity, tenant isolation, atau recovery defect.
- `NO-GO`: ada S0 sampai S2, evidence tidak lengkap, rollback belum diuji, atau commit kandidat berubah setelah regression tanpa impact analysis.

### Definition of Done

- [ ] Kandidat rilis tidak berubah setelah regression final, kecuali melalui re-test terdokumentasi.
- [ ] Semua penanda tangan memahami known limitations.
- [ ] Release dan rollback owner tersedia pada window deployment.
- [ ] Laporan final tersimpan bersama commit SHA kandidat.

### Prompt eksekusi

```text
Kerjakan QA-014 pada satu commit kandidat yang dibekukan. Verifikasi entry criteria lalu jalankan full Go test, vet, build, seluruh API collection, critical browser E2E, bot WhatsApp smoke test, security smoke, backup/restore verification, restart, dan rollback drill. Fasilitasi UAT satu kelas pilot untuk KM, PJ, dan mahasiswa menggunakan skenario di ticket. Susun traceability final serta defect summary. Jangan memberi status GO bila masih ada S0, S1, S2, evidence P0 hilang, atau rollback belum diuji. Catat commit SHA, environment, signer, waktu keputusan, known limitations, dan owner deployment/rollback.
```

---

## Urutan eksekusi yang disarankan

```text
QA-001
  -> QA-002
  -> QA-003 -> QA-004
  -> QA-005
       -> QA-006 -> QA-009
       -> QA-007
       -> QA-008
  -> QA-010
  -> QA-011
  -> QA-012
  -> QA-013
  -> QA-014
```

QA-006 dan QA-007 dapat berjalan paralel setelah QA-005 stabil. QA-010 dapat dimulai setelah audit dan backup implementation stabil. QA-011 harus memakai konfigurasi yang representatif terhadap production. QA-014 selalu memakai satu commit kandidat yang dibekukan.

## Gate penyelesaian seluruh QA

- [ ] Seluruh 48 functional requirements terpetakan ke test dan evidence.
- [ ] Semua endpoint API v1 memiliki contract coverage.
- [ ] Auth, session, RBAC, dan tenant isolation lulus negative testing.
- [ ] Semester, jadwal, tugas, notifikasi, audit, serta backup lulus lifecycle dan failure-path testing.
- [ ] Security hardening BE-012 sampai BE-014 tervalidasi pada staging representatif.
- [ ] Frontend lulus integrasi API, state matrix, mobile 390px, keyboard, dan accessibility checks.
- [ ] Concurrency serta recovery invariant kritis memiliki automated coverage.
- [ ] Tidak ada S0, S1, atau S2 terbuka.
- [ ] Full regression dan UAT pilot lulus pada commit kandidat yang sama.
- [ ] Backup dan rollback pernah dipraktikkan.
- [ ] QA Lead dan stakeholder terkait memberikan keputusan go/no-go tertulis.
