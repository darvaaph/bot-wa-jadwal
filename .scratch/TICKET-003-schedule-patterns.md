# TICKET-003: Schedule Patterns & Events (FR-SCH-002..008, SCR-SCH-001..005)

## Scope
Pola reguler (CRUD + versioning), Kejadian Perkuliahan draf/preview/publish/revoke, partisipasi lintas kelas, kandidat ruangan + konfirmasi TU, hapus pola/draf.

## Backend Status
- **SUDAH ADA** (test OK):
  - `GET/POST/PATCH /api/v1/schedule/patterns` — CRUD pola + versioning (`effective_until` closure)
  - `GET/POST /api/v1/teaching-events` — draf event
  - `POST /api/v1/teaching-events/{id}/preview` — conflict detection
  - `POST /api/v1/teaching-events/{id}/publish` — idempotency + conflict override
  - `POST /api/v1/teaching-events/{id}/revoke` — KM only, reason + version
  - `POST /api/v1/teaching-events/{id}/participation` — accept/decline/leave
  - `GET /api/v1/rooms/candidates?starts_at=&ends_at=` — kandidat ruangan
  - `POST /api/v1/teaching-events/{id}/room-confirmations` — TU konfirmasi
  - `DELETE /api/v1/schedule/patterns/{id}?version=` — **PERBAIKI** (soft delete via `effective_until`, bukan hard delete)
  - `DELETE /api/v1/teaching-events/{id}?version=` — sudah OK (hard delete DRAFT only)

## Frontend Status
- **SUDAH ADA** (parseked):
  - `view-jadwal.html`: pola CRUD, event draf/preview/publish/revoke, partisipasi, kandidat ruangan
  - Modal TU konfirmasi (CONFIRMED/REJECTED + nama TU + catatan)
  - Tombol Hapus pola (confirm + `API.deletePattern`) — **PERBAIKI**: call endpoint yang benar
  - Tombol Hapus draf event (DRAFT only + confirm + `API.deleteTeachingEvent`)
  - Peringatan publish jika ruangan dipilih tapi belum TU konfirmasi

## Required Fixes (P1)
- [ ] **Backend**: `DeletePattern` → soft delete via `UPDATE schedule_patterns SET effective_until = DATE_SUB(NOW(), INTERVAL 1 DAY) WHERE id=? AND version=?` (bukan hard delete). Versi lama tetap utuh untuk audit.
- [ ] **Frontend**: `hapusPola(p)` sudah call `API.deletePattern` — verifikasi endpoint benar.
- [ ] **Frontend**: `terbitkanPerubahan()` peringatan TU sudah ada — verifikasi `draftEvent.room_id` check benar.
- [ ] **Frontend**: `catatKonfirmasiRuang()` method lama masih ada tapi tidak dipakai — hapus atau comment.
- [ ] **Frontend**: Pastikan `loadPatterns()` dipanggil di `initKM()` (sudah di-refactor).

## Acceptance Criteria
- Pola reguler: tambah/ubah via version baru (tutup lama `effective_until`, buka baru `effective_from`)
- Hapus pola = non-aktifkan mulai hari ini (versi lama tetap readable), bukan hapus row
- Event draf → preview konflik → publish (idempotent) → revoke (KM only, reason) → koreksi disiarkan
- Partisipasi lintas kelas: owner buat + undang, peserta terima/tolak/lepas, invite ulang audit
- Kandidat ruangan: data internal + `source_updated_at` + wajib TU konfirmasi sebelum publish
- TU konfirmasi: simpan `room_id` + `external_contact` (nama TU) + `note` + `recorded_at`, hanya `CONFIRMED` allow publish

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-SCH-002..008
- `docs/product/USER_FLOWS.md` UF-SCH-001/003/004/005
- `docs/design/SCREEN_INVENTORY.md` SCR-SCH-001..005
- `internal/schedule/events.go` (Preview logic)