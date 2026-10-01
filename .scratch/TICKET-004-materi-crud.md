# TICKET-004: Materi CRUD (FR-TASK-007, SCR-MAT-001)

## Scope
List + filter + tambah materi (umum kelas / per-offering). Ubah/Arsip tunda (butuh backend).

## Backend Status
- **SUDAH ADA**: `GET /api/v1/materials?class_slug=&offering_id=`, `POST /api/v1/materials` (return `created_at`)
- **BELUM ADA**: `PATCH /api/v1/materials/{id}`, `DELETE /api/v1/materials/{id}` (soft delete `archived_at`)

## Frontend Status
- **SUDAH ADA**: `view-materi.html` — list (linkable judul + subjudul "Matkul · sumber" + label jenis + tanggal), filter panel kiri (Semua / Umum / per matkul + hitungan), sort terbaru/terlama, form tambah (judul, offeringId optional, jenis, URL, deskripsi)

## Required (P2)
- [ ] **Backend**: `PATCH /api/v1/materials/{id}` (title, material_type, url, description, version optimistic locking)
- [ ] **Backend**: `DELETE /api/v1/materials/{id}` (soft delete → `archived_at`, `deleted_at`, version)
- [ ] **Frontend**: Tombol **Ubah** di baris materi (modal prefill + version check)
- [ ] **Frontend**: Tombol **Arsipkan** di baris materi (confirm + version)
- [ ] **Frontend**: State `materiForm` support edit mode (`editMateriId`, `editMateriVersion`)

## Acceptance Criteria
- List materi: filter Semua / Umum / per matkul (hitungan real-time)
- Sort terbaru/terlama berdasarkan `created_at`
- Tambah materi: umum (offeringId kosong) / per matkul, link URL clickable
- Ubah materi: modal prefill, version conflict → banner "Versi berubah, muat ulang"
- Arsipkan materi: pindah ke arsip (tidak tampil default), bisa pulihkan
- Semua aksi terikat version (optimistic locking 409)

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-TASK-007
- `docs/design/SCREEN_INVENTORY.md` SCR-MAT-001
- `internal/api/v1/academic_controller.go` (GetMaterials/CreateMaterials)