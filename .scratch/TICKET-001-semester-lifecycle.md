# TICKET-001: Semester Lifecycle (FR-SEM-001..004, UR-SEM-001..003, UF-SEM-001/002)

## Scope
KM needs full CRUD + lifecycle for semester: create draft, import JSON / copy previous / manual input, preview (validate conflicts/kelengkapan), activate (archive old atomik), arsip read-only.

## Backend (P1)
- [ ] `POST /api/v1/classes/{slug}/semesters` — create draft (sudah ada, test OK)
- [ ] `POST /api/v1/semesters/{id}/import-validate` — validasi per-baris, all-or-nothing (sudah ada, test OK)
- [ ] `POST /api/v1/semesters/{id}/import-apply` — apply batch (sudah ada, test OK)
- [ ] `GET /api/v1/classes/{slug}/semesters/{id}/preview` — **BELUM ADA** (logic `SemesterService.Preview()` ada di `semester_lifecycle.go:348` tapi tidak di-expose)
- [ ] `POST /api/v1/classes/{slug}/semesters` + `source_semester_id` — **BELUM ADA** (salin semester lama; `copySemesterStructure()` ada tapi tidak di-wire)
- [ ] `POST /api/v1/classes/{slug}/semesters/{id}/activate` — sudah ada, test OK
- [ ] `POST /api/v1/semesters/{id}/offerings` — **BELUM ADA** (offering CRUD manual untuk semester DRAFT)

## Frontend (P1)
- [ ] `view-semester.html`: tambah tombol "Impor JSON" (file upload + preview error per baris)
- [ ] `view-semester.html`: tambah tombol "Salin Semester" (pilih semester sumber)
- [ ] `view-semester.html`: preview modal (conflicts, warnings, blockers, can_activate)
- [ ] `view-semester.html`: detail arsip read-only (jadwal, tugas, materi semester arsip)
- [ ] `app-km.js`: `loadSemesters()`, `importSemester()`, `previewSemester()`, `copySemester()`, `activateSemester()`

## Acceptance Criteria
- KM bisa bikin semester draf manual / impor / salin
- Impor JSON validasi per-baris, gagal = rollback total
- Preview konflik jadwal/ruangan/dosen sebelum aktivasi
- Aktivasi = arsip semester aktif lama + set ACTIVE atomik
- Arsip hanya-baca, detail jadwal/tugas/materi termuat

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-SEM-001..004
- `docs/product/USER_FLOWS.md` UF-SEM-001/002
- `docs/design/SCREEN_INVENTORY.md` SCR-SEM-001/002
- `internal/academic/semester_lifecycle.go` (Preview logic sudah ada)