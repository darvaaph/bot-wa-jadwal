# TICKET-009: Minta Backup KM (FR-OPS-002, UF-OPS-003)

## Scope
KM request backup/restore per kelas/semester ke System Admin. `ACCESS_CONTROL.md §8.5`: "Minta backup/restore ke SA (eksekusi SA, terikat kelas/semester, tolak paket tak cocok)."

## Backend Status
- **SUDAH ADA**: `POST /api/v1/backups` (KM boleh, body `{ class_slug?, semester_id?, reason? }`), `GET /api/v1/backups`
- **SA ONLY**: `POST /api/v1/restores` (verify-only, ADR-0008)

## Frontend Status
- **BELUM ADA**: UI backup di KM area

## Required (P2)
- [ ] **Frontend `view-pengaturan.html`** (atau tab baru di dashboard): section "Cadangan Kelas"
  - Tombol "Minta Cadangan" → modal: pilih scope (kelas ini / semester aktif / semua semester) + alasan → POST `/api/v1/backups`
  - List cadangan KM kelas ini: `GET /api/v1/backups?class_slug=` → tabel: status (PENDING/READY/FAILED), created_at, size, semester_id, reason
  - Status `READY` → link download (jika backend support) atau "Siap diambil admin"
- [ ] **Frontend `app-km.js`**: `mintaBackup(scope, reason)`, `loadBackups()`, `backupList`, `backupLoading`, `backupError`
- [ ] **Backend**: Pastikan `POST /backups` return `backup_id` + status, KM hanya bisa request kelasnya (scope validation sudah ada)

## Acceptance Criteria
- KM klik "Minta Cadangan" → pilih scope + alasan → request terkirim ke SA
- List cadangan kelas KM tampil dengan status real-time
- SA eksekusi backup (manual/jadwal), status jadi READY
- KM lihat cadangan siap, koordinasi SA untuk download/restore

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-OPS-002
- `docs/product/USER_FLOWS.md` UF-OPS-003
- `docs/product/ACCESS_CONTROL.md` §8.5
- `internal/api/v1/academic_controller.go` (CreateBackup/GetBackups)