# TICKET-002: Tugas Review & Lifecycle (FR-TASK-003..006, SCR-TASK-005/006)

## Scope
KM review retrospektif tugas PJ: Setujui / Minta Koreksi / Batalkan (wajib catatan, terikat versi). Tugas: selesai (`completed_at`), arsip (`archived_at`), pulihkan.

## Backend (P1)
- [ ] `POST /api/v1/tasks/{id}/reviews` — **SUDAH ADA** (decision: APPROVED/CHANGES_REQUESTED/REVOKED + note + task_version)
- [ ] `POST /api/v1/tasks/{id}/complete` — **SUDAH ADA** (set `completed_at`)
- [ ] `POST /api/v1/tasks/{id}/archive` — **SUDAH ADA** (set `archived_at`, soft-delete)
- [ ] `POST /api/v1/tasks/{id}/restore` — **SUDAH ADA** (hapus `archived_at`)
- [ ] `GET /api/v1/tasks/{id}` — **PERBAIKI**: return `completed_at`/`archived_at` asli (bukan `is_completed`/`is_archived` boolean saja) — `task_controller.go:477-478` perlu update

## Frontend (P1)
- [ ] `view-tugas.html` + `view-antrean.html`: tombol **Setujui / Minta Koreksi / Batalkan** di antrean + detail
- [ ] Dialog konfirmasi: "Minta Koreksi" → `CHANGES_REQUESTED` (wajib catatan, tugas jadi DRAFT), "Batalkan" → `REVOKED` (wajib catatan, tarik dari portal, batalkan notif)
- [ ] `app-km.js`: `setujuiTugas()`, `kirimReview()`, `putuskanDetail()` — **SUDAH ADA** (review)
- [ ] `app-km.js`: `tandaiSelesai()`, `arsipkanTugas()`, `pulihkanTugas()` — **SUDAH ADA** (lifecycle)
- [ ] Perbaiki `bukaDetailTugas()` baca `completed_at`/`archived_at` asli dari detail response
- [ ] Fix `antrean` getter: hapus filter `!completed_at` (review queue = `PUBLISHED` + `NOT_REVIEWED` saja)

## Acceptance Criteria
- KM review tugas terbit: Setujui (auto-approve versi baru KM) / Minta Koreksi (→ DRAFT) / Batalkan (→ REVOKED)
- Semua aksi wajib catatan + terikat `task_version` (optimistic locking)
- Tandai Selesai → tampil chip "Selesai", tombol Ubah/Arsip hilang
- Arsipkan → pindah ke tab Arsip, tombol Pulihkan muncul
- Pulihkan dari arsip → kembali ke status sebelumnya
- Antrean hanya tugas `PUBLISHED` + `NOT_REVIEWED` (tanpa filter `completed_at`)

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-TASK-003..006
- `docs/product/USER_FLOWS.md` UF-TASK-002/003
- `docs/design/SCREEN_INVENTORY.md` SCR-TASK-005/006
- `internal/api/v1/task_controller.go` (review logic sudah ada)