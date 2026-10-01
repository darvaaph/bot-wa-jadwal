# TICKET-006: Audit Filter + Before/After (FR-AUDIT-001/002, SCR-AUDIT-001)

## Scope
KM lihat riwayat perubahan kelas dengan filter pelaku/objek/tindakan/waktu + perbandingan before/after.

## Backend Status
- **SUDAH ADA**: `GET /api/v1/audit?since=&until=&actor=&entity_type=&action=&limit=&offset=` — filter server-side, KM dibatasi ke kelasnya (`admin_controller.go:536-757`)
- Response: `AuditLogResponseItem` { action, actor_name, actor_role, entity_type, entity_id, before_json, after_json, created_at }

## Frontend Status
- `view-monitor.html` (tab `log`): list dasar 50 item, `labelAksiAudit()`, `fmtWaktuID()` — **BELUM ADA FILTER**

## Required (P1)
- [ ] **Frontend `view-monitor.html` (tab `log`)**:
  - Filter form: Pelaku (select: KM/PJ/SA/Sistem), Objek (select: TASK/TEACHING_EVENT/PATTERN/ROOM_CONFIRMATION/USER/ASSIGNMENT), Tindakan (select: CREATE/UPDATE/DELETE/PUBLISH/REVOKE/ARCHIVE/RESTORE/APPROVE/REJECT/COMPLETE/SUSPEND/RECOVER), Waktu (datetime-local since/until)
  - Tombol "Terapkan" → `loadAuditLog({since, until, actor, entity_type, action})`
  - Tombol "Reset" → hapus filter + reload
  - Chip filter aktif (seperti System Admin antrean)
  - Pagination "Muat 50 lagi" (server-side offset)
- [ ] **Frontend `app-km.js`**: `loadAuditLog(params)` accept filter object, `auditList`, `auditLoading`, `auditError`, `auditHasMore`, `loadAuditLogMore()`
- [ ] **Detail row**: expand → tampilkan `before_json` / `after_json` formatted (pretty JSON atau diff highlight)
- [ ] **Dashboard preview** (`auditPreview` getter): tetap 5 terbaru tanpa filter

## Acceptance Criteria
- Filter pelaku/objek/tindakan/waktu bekerja server-side (cepat, tidak client-side filter)
- Pagination "Muat 50 lagi" tanpa duplikat
- Detail expand menampilkan before/after JSON (highlight field yang berubah)
- Chip filter aktif + tombol reset
- Dashboard preview tidak terpengaruh filter tab log

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-AUDIT-001/002
- `docs/design/SCREEN_INVENTORY.md` SCR-AUDIT-001
- `internal/api/v1/admin_controller.go` (GetAudit)