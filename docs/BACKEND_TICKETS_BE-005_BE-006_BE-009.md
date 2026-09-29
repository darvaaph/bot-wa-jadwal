# Backend Completion Tickets: BE-005, BE-006, dan BE-009

Dokumen ini merinci tiga pekerjaan backend yang masih diperlukan sebelum integrasi frontend dan rilis produksi. Setiap ticket memuat konteks, batas pekerjaan, acceptance criteria, skenario pengujian, dan prompt yang dapat langsung diberikan kepada coding agent.

## Ringkasan

| ID | Judul | Prioritas | Dependensi | Hasil utama |
|---|---|---:|---|---|
| BE-005 | Durable Notification Outbox | Tinggi | Tidak ada | Intent notifikasi tidak hilang ketika channel atau bot belum siap |
| BE-006 | Centralized Schedule Conflict Engine | Tinggi | BE-005 tidak wajib | Preview, penyimpanan, dan publikasi memakai hasil konflik yang sama |
| BE-009 | Canonical and Atomic Audit Writer | Tinggi | Sebaiknya setelah BE-005 dan BE-006 | Mutasi wajib dan audit berhasil atau gagal dalam satu transaksi |

Urutan implementasi yang disarankan:

1. BE-005
2. BE-006
3. BE-009

## Aturan umum implementasi

- Gunakan Go standar dan SQLite yang sudah dipakai repository.
- Jangan menambahkan frontend build step, npm, atau `package.json`.
- Pertahankan kontrak API v1 kecuali perubahan kontrak dicatat pada dokumentasi API dan collection pengujian.
- Semua query memakai parameter `?`.
- Perubahan skema harus aman untuk database baru dan database lama.
- Jangan mengubah file runtime dalam `storage/`.
- Tambahkan regression test sebelum atau bersama implementasi.
- Jalankan `gofmt` hanya pada file Go yang diubah.
- Sebelum selesai, jalankan:

```powershell
go test -count=1 ./...
go vet ./...
go build -o bin/bot ./cmd/bot
git diff --check
```

Referensi kontrak utama:

- [API v1](api/API_V1.md)
- [Business Rules](product/BUSINESS_RULES.md)
- [Functional Requirements](product/FUNCTIONAL_REQUIREMENTS.md)
- [Data Model](product/DATA_MODEL.md)
- [Backend Schema](spec/be-v1-schema.md)

---

## BE-005: Durable Notification Outbox

### Status

Belum dikerjakan.

### Masalah

Publikasi data web tidak boleh bergantung pada kesiapan WhatsApp. Saat ini `queueNotification` berhenti tanpa menyimpan intent jika kelas belum memiliki channel aktif. Akibatnya, publikasi dapat berhasil tetapi pesan yang seharusnya dikirim tidak memiliki rekaman outbox dan tidak dapat dipulihkan otomatis.

Ada juga ketidakkonsistenan skema:

- `internal/database/schema.sql` mendefinisikan `whatsapp_channel_id` sebagai `NOT NULL`.
- `internal/database/migrate_v1.go` sudah mengizinkan `whatsapp_channel_id` bernilai `NULL`.
- Test worker lama masih mengharapkan pesan tanpa channel ditolak.
- Worker dan service notifikasi belum memakai satu aturan klaim, retry, dan rekonsiliasi yang sama.

Kondisi ini bertentangan dengan BR-NOTIF-001, BR-NOTIF-002, BR-NOTIF-004, FR-NOTIF-004, dan aturan bahwa kegagalan WhatsApp tidak membatalkan publikasi web.

### Tujuan

Setiap intent notifikasi tersimpan secara durable pada transaksi mutasi bisnis yang memicunya. Intent tetap tersimpan ketika channel belum tersedia, bot offline, proses berhenti mendadak, atau pengiriman perlu dicoba ulang.

### Keputusan desain

Gunakan status publik yang sudah didokumentasikan. Pesan tanpa channel disimpan sebagai:

- `status = 'PENDING'`
- `whatsapp_channel_id = NULL`
- `scheduled_at = NULL`

`scheduled_at = NULL` berarti pesan menunggu konfigurasi, bukan siap diklaim worker. Setelah channel aktif tersedia, proses rekonsiliasi mengisi `whatsapp_channel_id` dan `scheduled_at = CURRENT_TIMESTAMP` secara atomik.

Keputusan ini menghindari penambahan status API baru. Jika implementer memilih status `PENDING_CONFIGURATION`, implementer wajib memperbarui seluruh dokumen kontrak, constraint SQLite, response API, frontend contract, dan collection Postman serta Insomnia.

### Ruang lingkup

1. Selaraskan definisi tabel `notification_messages` pada semua jalur pembuatan database.
2. Tambahkan migrasi SQLite untuk database lama yang masih memiliki `whatsapp_channel_id NOT NULL`.
3. Buat satu service outbox sebagai satu-satunya jalur enqueue notifikasi bisnis.
4. Sediakan operasi enqueue yang menerima `*sql.Tx` agar mutasi domain dan intent notifikasi berada pada transaksi yang sama.
5. Simpan intent meskipun channel aktif belum tersedia.
6. Tambahkan reconciler yang memasangkan pesan tanpa channel ketika channel kelas menjadi aktif.
7. Pastikan worker hanya mengklaim pesan yang memiliki channel, memiliki `scheduled_at`, dan siap dikirim.
8. Satukan aturan retry, stale processing recovery, attempt logging, idempotency, cancel, dan supersede.
9. Pastikan API antrean memperlihatkan pesan yang menunggu channel tanpa mengklaim bahwa pesan tersebut gagal dikirim.
10. Hapus atau delegasikan implementasi enqueue yang duplikat setelah semua caller memakai service yang sama.

### Di luar ruang lingkup

- Mengganti provider WhatsApp.
- Menambahkan message broker eksternal.
- Mengubah format pesan bisnis yang tidak berkaitan dengan durability.
- Menjamin exactly-once pada provider eksternal. Targetnya exactly-once secara logis di database dan at-least-once pada transport.

### Rancangan teknis minimum

Service outbox sebaiknya memiliki operasi dengan bentuk setara berikut:

```go
type EnqueueRequest struct {
	ClassID          int64
	EventType        string
	EntityType       string
	EntityID         int64
	IdempotencyKey   string
	Payload          any
	TriggeredByUserID *int64
}

func (s *Service) EnqueueTx(ctx context.Context, tx *sql.Tx, req EnqueueRequest) (int64, error)
func (s *Service) ReconcilePendingChannels(ctx context.Context, classID *int64) (int64, error)
```

Nama dan package boleh disesuaikan, tetapi transaksi dan perilakunya harus sama.

Aturan idempotensi:

- Key mewakili satu kejadian bisnis dan tujuan pesan.
- Pemanggilan ulang dengan key yang sama mengembalikan message ID yang sama.
- Pemanggilan ulang tidak membuat baris pesan atau attempt baru sebelum worker benar-benar mencoba mengirim.
- Perubahan data sebelum pengiriman membatalkan atau mengganti pesan lama sesuai BR-NOTIF-004.
- Hubungan koreksi memakai `supersedes_message_id` secara konsisten.

Aturan worker:

- Klaim hanya baris `PENDING` atau `FAILED` yang memiliki channel aktif dan `scheduled_at <= now`.
- Klaim memakai update kondisional agar dua worker tidak mengirim baris yang sama secara bersamaan.
- Setiap percobaan nyata membuat satu `notification_attempts`.
- Crash saat `PROCESSING` dapat dipulihkan ke `PENDING` setelah batas stale yang terdokumentasi.
- Retry manual menjadwalkan ulang baris yang sama, bukan membuat pesan bisnis baru.
- Channel `DISCONNECTED` atau `REVOKED` tidak boleh dipakai untuk pengiriman baru.

### File yang kemungkinan berubah

- `internal/database/schema.sql`
- `internal/database/migrate_v1.go`
- `internal/database/migrations/*.sql`
- `internal/notify/notify.go`
- `internal/bot/notification_worker.go`
- `internal/api/server.go`
- `internal/api/notifications_v1_handler.go`
- Test pada package `database`, `notify`, `bot`, dan `api`
- Dokumentasi API jika response antrean ditambah field konfigurasi turunan

### Acceptance criteria

- [ ] Publikasi jadwal atau tugas tetap berhasil ketika kelas belum mempunyai channel aktif.
- [ ] Publikasi tersebut menghasilkan tepat satu baris `notification_messages` dengan channel dan jadwal kirim kosong.
- [ ] Ketika channel kelas diaktifkan, reconciler menghubungkan pesan tertunda ke channel dan menjadwalkannya.
- [ ] Worker tidak mengklaim pesan tanpa channel atau tanpa `scheduled_at`.
- [ ] Bot offline tidak menghilangkan intent dan tidak membatalkan transaksi domain.
- [ ] Pemanggilan enqueue berulang dengan idempotency key yang sama tidak membuat duplikat.
- [ ] Dua worker yang berjalan bersamaan tidak sama-sama berhasil mengklaim message ID yang sama.
- [ ] Retry manual menggunakan message ID yang sama dan menambah attempt hanya ketika pengiriman dilakukan.
- [ ] Pesan usang dapat menjadi `CANCELLED` atau `SUPERSEDED` dan tidak terkirim setelah data berubah.
- [ ] Database baru dan database hasil migrasi memiliki constraint yang sama.
- [ ] Tidak ada lagi jalur enqueue bisnis yang diam-diam berhenti ketika channel tidak tersedia.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Publish tanpa channel | Domain tersimpan, outbox `PENDING`, channel `NULL`, jadwal `NULL` |
| Channel diaktifkan setelah publish | Reconciler mengisi channel dan jadwal tepat sekali |
| Enqueue key yang sama dua kali | Satu message ID dan satu baris outbox |
| Dua worker mengklaim baris sama | Hanya satu worker mendapat klaim |
| Worker crash setelah klaim | Reaper mengembalikan pesan stale ke antrean |
| Provider gagal | Status dan attempt menyimpan error, domain tetap terbit |
| Retry pesan gagal | Baris yang sama dijadwalkan ulang |
| Data berubah sebelum kirim | Pesan lama dibatalkan atau digantikan |
| Channel revoked sebelum klaim | Pesan tidak dikirim melalui channel tersebut |
| Migrasi database lama | Data lama utuh dan channel nullable sesuai desain |

### Definition of Done

- Seluruh acceptance criteria memiliki test otomatis.
- Tidak ada perbedaan definisi tabel antara `schema.sql` dan migrasi Go.
- Seluruh producer memakai service outbox yang sama.
- Worker lama dan service `internal/notify` tidak menerapkan aturan bisnis yang saling bertentangan.
- Test race dijalankan pada environment dengan CGO aktif jika tersedia.
- Dokumen API dan collection pengujian diperbarui jika bentuk response berubah.

### Prompt implementasi

```text
Kerjakan ticket BE-005 Durable Notification Outbox pada repository bot-wa-jadwal.

Baca lebih dahulu AGENTS.md, docs/api/API_V1.md, docs/product/BUSINESS_RULES.md bagian BR-NOTIF-001 sampai BR-NOTIF-005, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-NOTIF-001 sampai FR-NOTIF-006, docs/product/DATA_MODEL.md bagian notifikasi, serta docs/spec/be-v1-schema.md. Perlakukan dokumen tersebut sebagai kontrak.

Masalah yang harus diperbaiki: intent notifikasi saat ini dapat hilang ketika kelas belum memiliki WhatsApp channel aktif. Selain itu, internal/database/schema.sql mewajibkan whatsapp_channel_id, sedangkan skema v1 di Go mengizinkan NULL. Ada lebih dari satu implementasi enqueue dan worker dengan aturan yang belum seragam.

Implementasikan durable transactional outbox dengan ketentuan berikut:
1. Simpan intent walaupun channel belum tersedia menggunakan status PENDING, whatsapp_channel_id NULL, dan scheduled_at NULL.
2. Buat migration aman untuk database lama serta selaraskan schema.sql dan migrate_v1.go.
3. Buat satu service enqueue yang dapat berjalan di dalam *sql.Tx. Mutasi bisnis dan enqueue wajib commit atau rollback bersama.
4. Idempotency key yang sama harus mengembalikan message yang sama tanpa membuat duplikat.
5. Tambahkan reconciler untuk mengisi channel dan scheduled_at ketika channel aktif tersedia.
6. Worker hanya boleh mengklaim pesan yang mempunyai channel aktif dan sudah terjadwal.
7. Pertahankan attempt history, stale PROCESSING recovery, retry manual pada baris yang sama, serta CANCELLED dan SUPERSEDED untuk pesan usang.
8. Hilangkan jalur enqueue duplikat atau delegasikan semuanya ke service yang sama.
9. Jangan menambahkan broker atau dependency eksternal.

Kerjakan secara test-first. Tambahkan integration test dengan database SQLite sementara untuk publish tanpa channel, rekonsiliasi channel, idempotensi, concurrent claim, provider failure, retry, supersede, stale recovery, dan migration compatibility. Jangan menyentuh storage produksi. Jalankan gofmt pada file yang berubah, lalu go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check.

Laporkan file yang berubah, keputusan transaksi, bukti test, serta risiko yang masih tersisa. Jangan melakukan commit atau push kecuali diminta.
```

---

## BE-006: Centralized Schedule Conflict Engine

### Status

Belum dikerjakan.

### Masalah

Pemeriksaan konflik jadwal saat ini tersebar pada handler pembuatan pola, perubahan pola, preview event, dan publish event. Query yang terpisah mudah menghasilkan perbedaan keputusan. Sebuah event dapat lolos saat preview tetapi ditolak saat publish, atau sebaliknya.

Duplikasi juga menyulitkan pengujian aturan berikut:

- overlap ruangan;
- overlap dosen;
- overlap kelas atau offering;
- pola reguler versus teaching event terbit;
- tanggal efektif versi pola;
- event lintas kelas;
- pengecualian terhadap entitas yang sedang diedit;
- blocking conflict versus conflict yang boleh dioverride;
- batas semester dan zona waktu kelas pemilik.

Kondisi ini berisiko melanggar BR-SCH-002, BR-SCH-007, BR-SCH-009, FR-SCH-002, FR-SCH-004, FR-SCH-005, dan FR-SCH-006.

### Tujuan

Semua operasi jadwal memakai satu conflict engine. Candidate input yang sama dan snapshot database yang sama harus menghasilkan daftar konflik yang sama pada create, patch, preview, dan publish.

### Ruang lingkup

1. Buat package atau service khusus konflik jadwal di bawah `internal/schedule`.
2. Definisikan candidate dan hasil konflik sebagai tipe domain, bukan `map[string]any` di handler.
3. Gunakan satu definisi overlap interval: `[start, end)`. Sesi yang berakhir tepat saat sesi berikutnya mulai tidak bentrok.
4. Normalisasi waktu berdasarkan timezone kelas pemilik dan bandingkan dalam UTC.
5. Periksa pola reguler yang efektif pada tanggal candidate.
6. Periksa teaching event berstatus `PUBLISHED`; abaikan `DRAFT` dan `REVOKED` sebagai sumber benturan aktif.
7. Dukung pengecualian ID ketika mengubah pola atau event yang sama.
8. Kembalikan data konflik yang dapat digunakan langsung oleh endpoint preview dan validation error.
9. Pisahkan deteksi konflik dari kebijakan publish. Engine mendeteksi fakta; policy menentukan blocking dan kebutuhan alasan override.
10. Gunakan engine yang sama pada create pattern, patch pattern, create event, preview event, publish event, dan room candidates.

### Di luar ruang lingkup

- Membuat UI penyelesaian konflik.
- Sinkronisasi kalender eksternal.
- Menganggap hasil pemeriksaan ruangan sebagai konfirmasi resmi TU.
- Mengubah lifecycle teaching event selain yang diperlukan untuk validasi konflik.

### Model domain minimum

```go
type Candidate struct {
	OwnerClassID      int64
	OwnerOfferingID   int64
	ParticipantIDs    []int64
	RoomID            *int64
	LecturerIDs       []int64
	StartsAt          time.Time
	EndsAt            time.Time
	ExcludePatternID  *int64
	ExcludeEventID    *int64
}

type Conflict struct {
	Code          string
	Blocking      bool
	EntityType    string
	EntityID      int64
	StartsAt      time.Time
	EndsAt        time.Time
	Message       string
}
```

Nama field dapat disesuaikan. Hasil harus stabil, terurut, dan cukup rinci agar frontend dapat menjelaskan konflik tanpa menebak data lain.

### Kebijakan konflik minimum

| Kode | Kondisi | Kebijakan minimum |
|---|---|---|
| `CLASS_TIME_OVERLAP` | Kelas pemilik memiliki sesi lain pada interval sama | Blocking |
| `ROOM_TIME_OVERLAP` | Ruangan yang sama dipakai sesi lain | Blocking |
| `LECTURER_TIME_OVERLAP` | Dosen yang sama mengajar sesi lain | Blocking |
| `OFFERING_TIME_OVERLAP` | Offering yang sama memiliki sesi lain | Blocking |
| `PARTICIPANT_TIME_OVERLAP` | Kelas peserta memiliki sesi lain | Ditentukan eksplisit oleh policy dan dokumentasi |
| `ROOM_CONFIRMATION_REQUIRED` | Ruangan belum dikonfirmasi TU | Nonblocking, perlu penjelasan |
| `OUTSIDE_OWNER_SEMESTER` | Event di luar semester owner | Validation error, tidak dapat dioverride |
| `OUTSIDE_PARTICIPANT_SEMESTER` | Event di luar semester peserta | Validation error, tidak dapat dioverride |

Jika dokumentasi produk belum menentukan blocking untuk konflik peserta, jangan menebak. Catat keputusan sebagai pembaruan business rule sebelum mengubah kontrak publish.

Aturan override:

- Blocking conflict selalu menolak publish.
- Nonblocking conflict hanya dapat dilanjutkan dengan `conflict_override_reason` yang tidak kosong.
- Alasan override disimpan pada teaching event dan audit log.
- Alasan tidak menghapus daftar konflik dari response preview.

### File yang kemungkinan berubah

- Package baru atau file baru pada `internal/schedule`
- `internal/api/schedule_v1_handler.go`
- Handler room candidates
- Test pada `internal/schedule` dan `internal/api`
- Dokumentasi API jika struktur detail konflik diperjelas

### Acceptance criteria

- [ ] Preview dan publish menghasilkan kode konflik yang sama untuk candidate dan snapshot database yang sama.
- [ ] Create dan patch pattern memakai engine yang sama.
- [ ] Room candidates memakai definisi overlap yang sama.
- [ ] Interval bersebelahan tidak dianggap overlap.
- [ ] Room, lecturer, offering, dan class overlap terdeteksi terhadap pola efektif dan event terbit.
- [ ] Event `DRAFT` dan `REVOKED` tidak dianggap sesi aktif.
- [ ] Entitas yang sedang diedit dapat dikecualikan tanpa menyembunyikan konflik lain.
- [ ] Event di luar semester owner atau participant ditolak.
- [ ] Blocking conflict tidak dapat dipublikasikan walaupun alasan override diberikan.
- [ ] Nonblocking conflict memerlukan alasan override yang tidak kosong.
- [ ] Hasil konflik terurut deterministik dan memiliki code, blocking, entity, interval, serta message.
- [ ] Semua query tetap mematuhi scope role dan class dari sesi server.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Sesi A berakhir pukul 10.00, sesi B mulai 10.00 | Tidak bentrok |
| Sesi A berakhir pukul 10.01, sesi B mulai 10.00 | Bentrok |
| Room sama, class berbeda | `ROOM_TIME_OVERLAP` |
| Lecturer sama, room berbeda | `LECTURER_TIME_OVERLAP` |
| Class sama, offering berbeda | `CLASS_TIME_OVERLAP` |
| Konflik dengan event DRAFT | Tidak dihitung sebagai sesi aktif |
| Konflik dengan event REVOKED | Tidak dihitung sebagai sesi aktif |
| Patch entitas yang sama tanpa perubahan waktu | Tidak bentrok dengan dirinya sendiri |
| Pattern sudah melewati `effective_until` | Tidak dihitung pada tanggal candidate |
| Candidate sebelum `effective_from` | Pattern belum berlaku |
| Owner di luar semester | Ditolak tanpa override |
| Participant di luar semester | Ditolak tanpa override |
| Nonblocking tanpa alasan | Publish ditolak dengan validation error |
| Nonblocking dengan alasan | Publish boleh, alasan tersimpan dan teraudit |
| Preview lalu publish tanpa perubahan data | Daftar konflik identik |

### Definition of Done

- Semua handler jadwal menggunakan engine yang sama.
- Tidak ada query konflik bisnis yang tersisa di handler selain pemanggilan service.
- Klasifikasi blocking memiliki sumber aturan yang dapat ditunjuk.
- Unit test engine dan integration test endpoint lulus.
- Format error atau response preview terdokumentasi dan konsisten.
- Pengujian mencakup timezone, semester, versi pola, dan event lintas kelas.

### Prompt implementasi

```text
Kerjakan ticket BE-006 Centralized Schedule Conflict Engine pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian jadwal, docs/product/BUSINESS_RULES.md bagian BR-SCH-001 sampai BR-SCH-009, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-SCH-001 sampai FR-SCH-008, dan docs/product/DATA_MODEL.md bagian schedule_patterns, teaching_events, serta teaching_event_offerings. Jangan mengubah aturan bisnis tanpa memperbarui dokumen sumbernya.

Masalah yang harus diperbaiki: query konflik tersebar di beberapa handler sehingga create, patch, preview, publish, dan room candidates dapat memberi keputusan berbeda.

Implementasikan satu conflict engine di internal/schedule dengan ketentuan berikut:
1. Gunakan tipe Candidate dan Conflict yang eksplisit. Jangan mengembalikan map generik dari domain service.
2. Gunakan interval setengah terbuka [start, end). Interval yang hanya bersentuhan pada batas tidak bentrok.
3. Tafsirkan input dengan timezone kelas pemilik dan bandingkan dalam UTC.
4. Periksa class, offering, room, dan lecturer terhadap schedule pattern yang efektif serta teaching event PUBLISHED.
5. Abaikan DRAFT dan REVOKED sebagai sesi aktif.
6. Dukung exclude pattern ID dan exclude event ID untuk operasi update.
7. Pisahkan fakta konflik dari kebijakan blocking dan override.
8. Blocking conflict selalu menolak publish. Nonblocking conflict memerlukan conflict_override_reason yang tidak kosong serta harus teraudit.
9. Gunakan engine yang sama pada create pattern, patch pattern, create event, preview event, publish event, dan room candidates.
10. Hasil harus deterministik dan memuat code, blocking, entity type, entity ID, interval, dan message.

Jangan menebak klasifikasi konflik peserta lintas kelas jika dokumen belum tegas. Catat keputusan yang dibutuhkan dan perbarui business rule sebelum menerapkannya.

Kerjakan secara test-first. Buat table-driven unit test untuk overlap interval, effective range pattern, status event, pengecualian self, semester, timezone, room, lecturer, class, offering, participant, blocking, dan override. Tambahkan integration test yang membuktikan preview dan publish menghasilkan konflik yang sama. Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check.

Laporkan file yang berubah, matriks aturan konflik final, bukti test, perubahan kontrak jika ada, dan risiko yang masih tersisa. Jangan melakukan commit atau push kecuali diminta.
```

---

## BE-009: Canonical and Atomic Audit Writer

### Status

Belum dikerjakan.

### Masalah

Penulisan audit masih dilakukan dengan banyak `INSERT INTO audit_logs` langsung dari handler. Setiap caller membangun action, actor, before JSON, after JSON, reason, dan correlation ID sendiri. Pola ini menimbulkan beberapa risiko:

- mutasi berhasil tetapi audit gagal atau tidak ditulis;
- audit sukses tercatat walaupun mutasi kemudian gagal;
- snapshot before dan after memakai bentuk berbeda antar endpoint;
- actor role context hilang atau salah pada pengguna multi-role;
- correlation ID tidak stabil;
- secret, token, kode portal, hash, atau data sensitif masuk ke JSON audit;
- action dan entity type tidak memiliki kosakata kanonis;
- pengujian rollback harus dibuat ulang pada setiap handler.

Kondisi ini berisiko melanggar BR-AUDIT-001, BR-AUDIT-002, FR-AUDIT-001, FR-AUDIT-002, dan FR-AUDIT-003.

### Tujuan

Semua mutasi yang wajib diaudit memakai satu writer. Mutasi dan audit berjalan pada transaksi yang sama. Snapshot, identitas pelaku, reason, serta correlation ID memiliki bentuk kanonis dan aman dari data sensitif.

### Ruang lingkup

1. Buat package audit terpusat, misalnya `internal/audit`.
2. Writer harus menerima executor transaksi dan context.
3. Definisikan tipe actor, scope, entity, action, before, after, reason, dan correlation ID.
4. Validasi field wajib sebelum insert.
5. Serialisasi JSON secara konsisten dari struct atau map yang telah disanitasi.
6. Redaksi atau tolak field sensitif sebelum data masuk audit.
7. Buat correlation ID di batas request atau business operation, lalu gunakan ID yang sama untuk semua audit dan outbox terkait.
8. Migrasikan semua mutasi wajib audit ke writer baru.
9. Pastikan kegagalan insert audit membatalkan mutasi domain.
10. Pertahankan trigger append-only pada `audit_logs`.
11. Selaraskan query baca audit dengan cakupan PJ, KM, dan System Admin sesuai dokumentasi.

### Di luar ruang lingkup

- Menghapus audit lama.
- Membuat mekanisme retensi fisik baru.
- Menyimpan request atau response HTTP mentah.
- Merekam password, token, cookie, recovery code, portal code, atau hash rahasia.

### Rancangan teknis minimum

```go
type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Actor struct {
	Type             string
	UserID           *int64
	RoleAssignmentID *int64
	Context           any
}

type Entry struct {
	Actor         Actor
	ClassID       *int64
	SemesterID    *int64
	Action        string
	EntityType    string
	EntityID      *int64
	Before        any
	After         any
	Reason        string
	CorrelationID string
}

func (w *Writer) Write(ctx context.Context, exec Executor, entry Entry) error
```

Writer harus dapat menerima `*sql.Tx`. Pemakaian langsung `*sql.DB` hanya diperbolehkan untuk kejadian sistem yang memang tidak memiliki mutasi domain terkait. Alasan ini harus terlihat pada caller dan memiliki test.

### Kosakata kanonis

Action dan entity type harus berupa konstanta. Minimal mencakup:

- create, update, delete, restore;
- publish dan revoke;
- semester activation;
- role invitation, assignment, suspension, dan revocation;
- account suspension dan recovery;
- portal code rotation;
- room confirmation;
- notification retry;
- import apply;
- backup dan restore verification;
- legacy shim jika masih ada operasi baca yang perlu dicatat.

Nama final harus mengikuti action yang sudah dipakai client atau dokumentasi. Jangan mengganti nama action yang terlihat oleh API tanpa migration atau compatibility mapping.

### Data sensitif yang dilarang

Audit tidak boleh menyimpan nilai berikut, termasuk dalam object bertingkat:

- password dan password hash;
- bearer token, session token, cookie, dan refresh token;
- recovery token atau recovery code;
- portal access code dan portal code hash;
- API key, secret, private key, atau credential provider;
- isi file backup;
- payload WhatsApp yang mengandung credential.

Sanitizer harus memakai allowlist untuk snapshot domain yang kritis. Pencocokan nama field saja tidak cukup untuk semua kasus.

### Matriks atomicity minimum

| Operasi | Mutasi dan audit wajib satu transaksi |
|---|---|
| Ubah status kelas | Ya |
| Terima undangan dan buat role assignment | Ya |
| Suspend atau recover user | Ya |
| Create atau patch schedule pattern | Ya |
| Create, publish, atau revoke teaching event | Ya |
| Create, patch, review, complete, archive, atau restore task | Ya |
| Aktivasi semester | Ya |
| Konfirmasi ruangan | Ya |
| Apply import | Ya |
| Retry notification | Ya |
| Rotasi portal code | Ya |
| Pembuatan metadata backup | Ya |
| Restore verification record | Ya |

Jika operasi juga membuat outbox, mutasi domain, audit, dan enqueue outbox harus memakai transaksi yang sama.

### File yang kemungkinan berubah

- Package baru `internal/audit`
- Handler pada `internal/api` yang menulis langsung ke `audit_logs`
- Service domain yang melakukan mutasi beraudit
- Middleware atau helper correlation ID
- Test pada package audit dan integration test API
- Dokumentasi action audit jika kosakata diperjelas

### Acceptance criteria

- [ ] Semua insert audit produksi melewati writer kanonis.
- [ ] Mutasi wajib audit dan audit insert memakai transaksi yang sama.
- [ ] Forced audit failure membuat mutasi domain rollback.
- [ ] Forced domain failure tidak meninggalkan audit sukses.
- [ ] Pengguna multi-role mencatat `actor_role_assignment_id` aktif yang benar.
- [ ] System actor memakai `actor_type = 'SYSTEM'` dan context yang cukup tanpa user palsu.
- [ ] Before dan after valid JSON dengan struktur stabil.
- [ ] Reason wajib untuk operasi yang mensyaratkannya dan tidak dapat hanya whitespace.
- [ ] Correlation ID tidak kosong dan sama untuk mutasi, audit, serta outbox yang berasal dari operasi yang sama.
- [ ] Secret dan credential tidak muncul pada audit, termasuk object bertingkat.
- [ ] Trigger database tetap menolak update dan delete pada audit log.
- [ ] Endpoint baca audit tetap menerapkan scope PJ, KM, dan System Admin.
- [ ] Action dan entity type memakai konstanta kanonis.

### Test matrix

| Skenario | Hasil yang diharapkan |
|---|---|
| Audit insert dipaksa gagal | Mutasi domain rollback |
| Mutasi domain dipaksa gagal | Tidak ada audit sukses |
| Commit transaksi gagal | Tidak ada state parsial |
| User memiliki dua role | Audit memakai assignment aktif dari sesi |
| System job menulis audit | Actor type SYSTEM, user ID kosong |
| Before atau after berisi secret bertingkat | Secret tidak tersimpan |
| Reason hanya whitespace | Ditolak untuk action yang mewajibkan reason |
| Publish membuat audit dan outbox | Ketiganya memakai correlation ID yang sama |
| Update audit log | Ditolak trigger append-only |
| Delete audit log | Ditolak trigger append-only |
| PJ membaca audit offering lain | Ditolak atau tidak muncul sesuai kontrak |
| KM membaca kelas lain | Tidak bocor |
| Admin membaca audit global | Diizinkan |

### Definition of Done

- Tidak ada `INSERT INTO audit_logs` langsung pada production code di luar package audit.
- Setiap operasi pada matriks atomicity mempunyai test rollback.
- Audit writer memiliki unit test untuk validasi, serialization, redaction, dan actor context.
- Correlation ID menghubungkan domain mutation, audit, dan outbox.
- Dokumentasi action audit dan response baca tetap sinkron dengan implementasi.
- Seluruh test, vet, build, dan diff check lulus.

### Prompt implementasi

```text
Kerjakan ticket BE-009 Canonical and Atomic Audit Writer pada repository bot-wa-jadwal.

Baca AGENTS.md, docs/api/API_V1.md bagian aturan audit dan endpoint audit, docs/product/BUSINESS_RULES.md bagian BR-AUDIT-001 dan BR-AUDIT-002, docs/product/FUNCTIONAL_REQUIREMENTS.md bagian FR-AUDIT-001 sampai FR-AUDIT-003, docs/product/DATA_MODEL.md bagian audit_logs, serta migration trigger append-only. Perlakukan dokumen tersebut sebagai kontrak.

Masalah yang harus diperbaiki: production code masih menulis INSERT INTO audit_logs langsung dari banyak handler. Actor context, before dan after JSON, reason, correlation ID, redaction, serta transaksi belum dijamin konsisten.

Implementasikan writer audit kanonis di internal/audit dengan ketentuan berikut:
1. Writer menerima context, executor yang kompatibel dengan *sql.Tx, dan typed Entry.
2. Action dan entity type memakai konstanta.
3. Validasi actor, scope, entity, reason, dan correlation ID sebelum insert.
4. Serialisasi before dan after secara konsisten serta sanitasi data sensitif dengan allowlist untuk snapshot kritis.
5. Password, hash, token, cookie, recovery code, portal code, API key, secret, dan credential tidak boleh masuk audit.
6. Correlation ID dibuat sekali per operasi dan dipakai juga oleh outbox terkait.
7. Migrasikan seluruh mutasi wajib audit ke writer baru.
8. Mutasi domain dan audit wajib commit atau rollback dalam transaksi yang sama. Jika operasi membuat outbox, outbox masuk transaksi yang sama.
9. Pertahankan trigger append-only dan scope baca audit.
10. Setelah migrasi, jangan sisakan INSERT INTO audit_logs langsung pada production code di luar package audit.

Kerjakan secara test-first. Tambahkan unit test untuk validation, JSON serialization, nested-secret redaction, actor USER dan SYSTEM, serta correlation ID. Tambahkan integration test yang memaksa audit insert gagal dan membuktikan rollback domain pada setiap kelompok mutasi penting. Tambahkan test kebalikannya: domain gagal dan tidak ada audit sukses. Pastikan test scope baca audit untuk PJ, KM, dan Admin tetap lulus.

Jalankan gofmt, go test -count=1 ./..., go vet ./..., go build -o bin/bot ./cmd/bot, dan git diff --check. Laporkan file yang berubah, daftar caller yang dimigrasikan, bukti atomicity, hasil test, dan risiko yang tersisa. Jangan melakukan commit atau push kecuali diminta.
```

---

## Gate penyelesaian ketiga ticket

Ketiga ticket dinyatakan selesai jika semua kondisi berikut terpenuhi:

- [ ] Publikasi web tidak kehilangan intent notifikasi ketika WhatsApp belum siap.
- [ ] Seluruh keputusan konflik jadwal berasal dari satu engine.
- [ ] Mutasi wajib dan audit bersifat atomik.
- [ ] Idempotency dan correlation ID dapat ditelusuri dari domain mutation ke audit dan outbox.
- [ ] Tidak ada kebocoran lintas kelas atau lintas offering pada endpoint terkait.
- [ ] Dokumentasi API, business rule, data model, schema, dan collection pengujian selaras.
- [ ] `go test -count=1 ./...` lulus.
- [ ] `go vet ./...` lulus.
- [ ] `go build -o bin/bot ./cmd/bot` lulus.
- [ ] `git diff --check` lulus.
- [ ] Test race dijalankan pada environment dengan CGO aktif, atau keterbatasan environment dicatat secara eksplisit.
