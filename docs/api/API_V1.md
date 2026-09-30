# API v1 — Spesifikasi Beku (L0)

> Status: FROZEN untuk eksekusi horizontal (per ADR-0005). Perubahan butuh ADR baru.
> Basis: `docs/product/FUNCTIONAL_REQUIREMENTS.md` v3.0.1, `DATA_MODEL.md` v3.0.2, `INFORMATION_ARCHITECTURE.md` v2.0.
> Legacy: 7 endpoint lama tetap hidup sebagai shim `Deprecation: true` (per ADR-0002).

## 0. Konvensi global

- Base: `/api/v1`. JSON saja, UTF-8. Waktu tulis = UTC RFC3339 (`deadline_at`, `starts_at`); tampil = zona `class_settings.timezone` (default `Asia/Jakarta`).
- Envelope sukses: `{ "status":"success", "data":{...} }` atau `"data":[...]` + opsional `"meta":{page,per_page,total}`.
- Envelope gagal: `{ "status":"error", "error":{ "code":"STRING", "message":"id...", "details":{...} } }`.
  Kode: `UNAUTHENTICATED 401 | FORBIDDEN 403 | NOT_FOUND 404 | VALIDATION 422 | VERSION_CONFLICT 409 | GONE_ARCHIVED 410 | TOO_MANY_REQUESTS 429 | NOT_IMPLEMENTED 501`.
- Optimistic locking: semua PATCH/POST-publish kirim `version`; mismatch → `409 + {current_version, current_data}`; input user tidak boleh hilang (FE wajib tampilkan diff).
- Scope diambil dari sesi server, bukan dari payload. `class_id/semester_id/offering_id` di URL diverifikasi lawan `active_role_assignment_id`.
- Audit: publish/revoke/delete/restore/assign-role selalu tulis `audit_logs` (tak ada endpoint tulis audit langsung).

## 1. Auth & konteks (`FR-ACCESS-002 s.d 007`)

| Method & Path | Auth | Body / Query | 200 | Catatan |
|---|---|---|---|---|
| `POST /api/v1/auth/login` | publik, rate-limit | `{identity_key, password}` | `{token, token_type:"Bearer", expires_at, assignments:[{id,role,class_slug,semester_id,offering_id}], need_context_choice:bool}` | 5 gagal/15 mnt → 429 + `login_attempts`; pesan gagal generik |
| `POST /api/v1/auth/logout` | Bearer | `{}` | `{revoked:true}` | Cabut token aktif |
| `GET /api/v1/auth/me` | Bearer | — | `{user, active_assignment, classes}` | Setiap halaman pengelola wajib panggil untuk tampilkan konteks aktif |
| `POST /api/v1/auth/switch-context` | Bearer | `{role_assignment_id}` | `{token_baru, expires_at}` | Rotasi token, ganti konteks tanpa login ulang |
| `GET /api/v1/classes` | Bearer KM/Admin atau portal-token | — | `{classes:[{slug,code,program,cohort,group,status}]}` | KM: kelas konteks aktif; Admin: seluruh kelas; portal-token: hanya kelas token; PJ ditolak |
| `POST /api/v1/classes/:slug/portal-code/rotate` | KM kelas terkait/Admin | `{code?}` | `{portal_code,portal_code_version,portal_access_mode:"CODE",reveal_once:true}` | Tanpa `code`, server membuat kode 8 digit; kode hanya ditampilkan sekali; versi naik dan sesi lama dicabut atomik |
| `POST /api/v1/invitations` | KM/Admin sesuai scope | `{role, class_slug, semester_id?, offering_id?, invited_identity_key}` | `{invitation_id, expires_at}` | Scope dikunci server; kirim ulang → revoke lama |
| `POST /api/v1/invitations/accept` | token undangan | `{token, password?, display_name?}` | `{user_id, assignment_id}` | Token sekali pakai |

Header: `Authorization: Bearer <token>`. Cookie `bv1` httpOnly opsional sebagai fallback Alpine.

## 2. Portal baca (mahasiswa, `FR-ACCESS-001`)

Mode default `LINK` (tanpa kode). Mode `CODE`: `X-Portal-Token` atau `?portal_token=` berisi sesi portal.

`POST /api/v1/portal/:slug/session` menukar `{code}` dengan `{portal_token, expires_at}`. Token mentah hanya dikirim pada respons ini; server menyimpan hash token. Kode salah dan slug yang tidak dikenal sama-sama menghasilkan pesan `401` generik. Lima kegagalan dalam 15 menit membatasi sumber dan kelas tersebut selama 15 menit.

| Method & Path | Respon `data` |
|---|---|
| `GET /api/v1/portal/:slug/summary?date=YYYY-MM-DD` | `{class, date, now_event, next_event, today:[...], changes_today:[...], nearest_tasks:[...3]}` |
| `GET /api/v1/portal/:slug/schedule?date=YYYY-MM-DD` | `{class, date, items:[{id, kind:REGULER\|PENGGANTI\|TAMBAHAN\|LIBUR\|DIBATALKAN, offering, title, activity_type, starts_at, ends_at, room, lecturers[], source:{pattern_id, event_id}}]}` — gabungan patterns + events `PUBLISHED`, label non-warna |
| `GET /api/v1/portal/:slug/tasks?group=hari_ini\|minggu_ini\|mendatang\|terlewat&offering=&q=` | `[{id, offering, title, instructions, deadline_at, submission_text, submission_url, version}]` urut deadline terdekat; overdue tetap muncul |
| `GET /api/v1/portal/:slug/tasks/:id` | Detail penuh + materi terkait |
| `GET /api/v1/portal/:slug/changes?since=` | Teaching events + koreksi terbit, terbaru dulu |
| `GET /api/v1/portal/:slug/materials?offering=` | `[{id, title, material_type, url, description}]` |

## 3. Semester & offering (KM/Admin, `FR-SEM`, `FR-CLASS`)

| Method & Path | Body penting | Status |
|---|---|---|
| `GET /api/v1/classes/:slug/semesters` | — | `DRAFT/ACTIVE/ARCHIVED` + `published_at` |
| `POST /api/v1/classes/:slug/semesters` | `{academic_year, term, starts_on, ends_on, source:{type:manual\|copy_from\|import_batch}}` | Buat `DRAFT`; `ends_on>starts_on` |
| `POST /api/v1/classes/:slug/semesters/:id/activate` | `{confirm:true}` | Atomik: arsipkan lama + `published_at` baru; PJ ditolak 403 |
| `POST /api/v1/semesters/:id/import-validate` | batch JSON kurikulum | `{batch_id, summary, errors:[{row,field,code,message,severity}]}`; `ERROR` blokir aktivasi |
| `POST /api/v1/semesters/:id/import-apply` | `{batch_id}` | Transaksional; gagal → rollback penuh |
| `GET /api/v1/semesters/:id/offerings` | — | `[{id, course_code, display_name, activity_type, lecturers[], pj_assignment}]` |

## 4. Jadwal: pola + kejadian (`FR-SCH-001 s.d 008`)

| Method & Path | Body penting | Aturan |
|---|---|---|
| `GET /api/v1/schedule/patterns?offering_id=&day=` | — | Filter offering sesuai scope PJ |
| `POST /api/v1/schedule/patterns` | `{offering_id, day_of_week:1-7, start_time, duration_min, room_id?, lecturer_ids[]}` | Server hitung `end_time`; cek konflik |
| `PATCH /api/v1/schedule/patterns/:id` | `{..., version}` | Permanen via versi baru (pola lama `effective_until`=hari ini inklusif, pola baru `effective_from`=besok; response `{id, replaces_pattern_id, version, effective_from, effective_until:null}`; BE-007) |
| `POST /api/v1/teaching-events` | `{owner_offering_id, event_kind, starts_at, ends_at, origin_pattern_id?, origin_date?, participant_offering_ids[], room_id?, reason?}` | Buat `DRAFT`; `REPLACEMENT/SESSION_CANCELLED` wajib `origin_*`; tanggal dalam semester owner |
| `GET /api/v1/teaching-events?scope=mine&status=draft\|published\|revoked&from=&to=` | — | Tab Draf/Terbit/Dicabut |
| `POST /api/v1/teaching-events/:id/preview` | `{}` | `{old, new, kind, conflicts:[{type, message, blocking}], room_note:"perlu konfirmasi TU"}`; blocking → tolak publish |
| `POST /api/v1/teaching-events/:id/publish` | `{version, conflict_override_reason?}` | PJ hanya offering-nya; `OWNER` tepat satu; idempoten via `Idempotency-Key`; WA gagal ≠ batal publish |
| `POST /api/v1/teaching-events/:id/revoke` | `{reason, version}` | KM saja; `REVOKED`; portal kembali ke jadwal berlaku; siaran koreksi jika sudah tersiar |
| `POST /api/v1/teaching-events/:id/participation` | `{action:accept\|decline\|leave}` | KM peserta; `PENDING→ACCEPTED/DECLINED`, `ACCEPTED→REMOVED`; tanggal dalam semester peserta |

## 5. Tugas + review (`FR-TASK-001 s.d 006`)

| Method & Path | Body penting | Aturan |
|---|---|---|
| `GET /api/v1/tasks?offering_id=&tab=aktif\|draf\|review\|selesai\|terlewat\|arsip` | — | Portal baca pakai §2; ini untuk pengelola + review |
| `POST /api/v1/tasks` | `{offering_id, title, instructions?, deadline_at, task_type?, submission_text?, submission_url?, save_as:draft\|published}` | Draf boleh tak lengkap; publish wajib `title+instructions+deadline_at+salah satu submission_*`. Publish PJ → `PUBLISHED/NOT_REVIEWED`; publish KM → + review `APPROVED` |
| `GET /api/v1/tasks/:id` | — | `{task, reviews[], versions_info, notifications[]}` |
| `PATCH /api/v1/tasks/:id` | `{..., version}` | Naikkan `version`; PJ → `NOT_REVIEWED`; KM → + `APPROVED` baru; deadline/tempat berubah → flag butuh notif update |
| `POST /api/v1/tasks/:id/reviews` | `{decision:APPROVED\|CHANGES_REQUESTED\|REVOKED, note?, task_version}` | KM saja; `CHANGES_REQUESTED→DRAFT`, `REVOKED→REVOKED` + tarik dari portal; `task_version` harus = versi aktif; review menaikkan `version` (first-writer-wins, 409 bila stale; BE-008) |
| `POST /api/v1/tasks/:id/complete` `.../archive` `.../restore` | `{version}` | `completed_at/archived_at` terpisah dari `publication_status`; soft-delete + restore beraudit |

## 6. Materi (`FR-TASK-007`, baca dulu)

- `GET /api/v1/materials?class_slug=&offering_id=` → list (portal + pengelola).
- `POST /api/v1/materials` `{class_slug, offering_id?, task_id?, title, material_type, url, description}` → PJ hanya offering-nya; KM boleh umum. (Tulis di v1; jika kepepet, tulis geser ke v1.1 tanpa ubah baca.)

## 7. Fitur Lanjutan v1 (Ruangan, Notifikasi, Audit, Backup, Admin)

| Method & Path | Body penting / Query | Aturan & Akses |
|---|---|---|
| `GET /api/v1/rooms/candidates` | `?starts_at=&ends_at=` | Auth; Cari ruangan yang tidak bentrok dengan jadwal lain |
| `POST /api/v1/teaching-events/:id/room-confirmations` | `{notes?, confirmed_room_id?}` | KM / Admin; Konfirmasi kesiapan ruangan TU |
| `GET /api/v1/notifications` | `?status=PENDING\|SENT\|FAILED&limit=` | KM / Admin; Antrean siaran pesan WhatsApp |
| `POST /api/v1/notifications/:id/retry` | — | KM / Admin; Jadwalkan ulang pesan `FAILED`/`CANCELLED` menjadi `PENDING`. Response `{id, status:"PENDING", scheduled_at, retry_scheduled:true}` tanpa `attempt_number`; attempt hanya dibuat worker saat delivery (BE-010) |
| `GET /api/v1/audit` | `?entity_type=&action=&entity_id=&actor=&since=&until=&class_slug=&limit=` | KM (kelasnya) / Admin; Rekam jejak audit trail perubahan sistem. `since/until` RFC3339 atau YYYY-MM-DD (presisi detik, UTC); `actor` = ID numerik atau identity_key; `entity_id` numerik |
| `POST /api/v1/backups` | `{class_slug?, semester_id?, reason?}` | KM (kelasnya) / Admin; Snapshot basis data aman via `VACUUM INTO`. `semester_id` opsional, wajib milik kelas; nama berkas unik |
| `GET /api/v1/backups` | `?class_slug=&status=&limit=` | KM (kelasnya) / Admin; Daftar cadangan tanpa path internal (`artifact_ref` tak dikembalikan) |
| `POST /api/v1/restores` | `{backup_id, reason!}` | Admin; Verify-only (ADR-0008): verifikasi path dalam storage backup, checksum (`422 CHECKSUM_MISMATCH`), format SQLite, schema, scope kelas + semester + relasi (`foreign_key_check`); tandai `VERIFIED`; response `{backup_id, class_id, semester_id?, status:"VERIFIED", checksum, restore_performed:false}` tanpa path internal. Database aktif tidak diganti |
| `GET /api/v1/admin/status` | — | Admin; Telemetri runtime, koneksi bot, dan status migrasi |
| `GET /api/v1/admin/assignments` | `?status=&role=&class_slug=&limit=` | Admin; Daftar Penugasan Peran + scope |
| `POST /api/v1/admin/assignments/:id/suspend` | `{reason!, force?}` | Admin (+KM untuk PJ kelasnya); cabut sesi penugasan; guard KM-terakhir (`409` kecuali `force`) |
| `POST /api/v1/admin/assignments/:id/revoke` | `{reason!, force?}` | Sama dengan suspend; status akhir `REVOKED` |
| `GET /api/v1/admin/invitations` | `?status=&role=&class_slug=` | Admin (+KM kelasnya); Daftar Undangan tanpa token (`EXPIRED` derivasi) |
| `POST /api/v1/admin/invitations/:id/revoke` | `{reason!}` | Admin (+KM untuk PJ kelasnya); hanya `PENDING` |
| `POST /api/v1/admin/support/enter` | `{class_slug!, reason! min 10}` | Admin; Hibah dukungan 60 menit, tutup hibah lama; audit `SUPPORT_ENTER` |
| `POST /api/v1/admin/support/exit` | `{reason?}` | Admin; Tutup hibah aktif; audit `SUPPORT_EXIT` |
| `GET /api/v1/admin/support/active` | — | Admin; Hibah aktif atau `null`; kedaluwarsa ditandai `EXPIRED` |
| `POST /api/v1/admin/users/:id/suspend` | `{reason?}` | Admin; Bekukan pengguna dan cabut seluruh sesi aktif |
| `POST /api/v1/admin/users/:id/recover` | `{reason?}` | Admin; Pulihkan akun yang sebelumnya dibekukan |

## 8. Shim legacy → v1

| Lama | Shim |
|---|---|
| `GET /api/health`, `/api/status`, `/api/classes` | Tetap + header `Deprecation: true`; `/api/status` tambah `v1:"/api/v1/portal/:slug/summary"` |
| `GET /api/schedule?class=&day=` | Terjemahkan `class→slug`, `day→date`; baca dari patterns+events DB baru (kelas pilot), fallback JSON lama di luar pilot |
| `GET /api/tasks?class=` | Petakan `class→offering` pilot; field lama `matkul/deskripsi/deadline` diisi dari `display_name/title/deadline_at` |
| `POST /api/tasks`, `DELETE /api/tasks/{id}` | `410 Gone` + tautan pengganti `/api/v1/tasks`; write legacy dihentikan agar tidak ada dua sumber data (ADR-0007) |

## 9. Contoh

```http
POST /api/v1/auth/login
{"identity_key":"+62812xxxx","password":"***"}
→ {"status":"success","data":{"token":"bv1_...","assignments":[{"id":7,"role":"PJ","class_slug":"d4-ti-2024-a","offering_id":12}],"need_context_choice":false}}

GET /api/v1/portal/d4-ti-2024-a/schedule?date=2026-10-05
→ {"status":"success","data":{"class":"d4-ti-2024-a","date":"2026-10-05","items":[{"id":"ev_9","kind":"PENGGANTI","title":"Basis Data","starts_at":"2026-10-05T02:00:00Z","room":"R-301","lecturers":["B. Santoso"]}]}}

POST /api/v1/tasks
Authorization: Bearer bv1_...
{"offering_id":12,"title":"Normalisasi 1NF-3NF","instructions":"Kerjakan modul h.42","deadline_at":"2026-10-10T16:59:00Z","submission_url":"https://classroom/...","save_as":"published"}
→ 201 {"status":"success","data":{"id":55,"publication_status":"PUBLISHED","review_state":"NOT_REVIEWED","version":1}}
```
