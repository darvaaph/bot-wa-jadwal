# TICKET-005: Semester Import / Copy / Preview + Arsip Detail (FR-SEM-001..004, SCR-SEM-001/002)

## Scope
Lengkapi `view-semester.html` + backend yang sudah ada tapi belum di-wire: import JSON validate/apply, salin semester, preview konflik, detail arsip read-only.

## Backend Status (Sudah Ada, Belum Di-wire)
- `POST /api/v1/semesters/{id}/import-validate` — validasi per-baris, error/warning, checksum
- `POST /api/v1/semesters/{id}/import-apply` — apply batch (transactional upsert courses/lecturers/offerings/patterns)
- `SemesterService.Preview()` — logic ada di `semester_lifecycle.go:348` (conflicts, warnings, blockers, can_activate)
- `SemesterService.copySemesterStructure()` — ada tapi tidak di-expose via endpoint
- **BELUM ADA ENDPOINT**: `GET /api/v1/classes/{slug}/semesters/{id}/preview`, `POST /api/v1/classes/{slug}/semesters` + `source_semester_id`, `POST /api/v1/semesters/{id}/offerings` (manual CRUD)

## Frontend Status
- `view-semester.html`: list + buat draf manual + aktivasi (sudah OK)
- **BELUM**: Import JSON (file upload + preview error per baris), Salin semester, Preview modal, Arsip detail

## Required (P1)
- [ ] **Backend**: `GET /api/v1/classes/{slug}/semesters/{id}/preview` → expose `SemesterService.Preview()`
- [ ] **Backend**: `POST /api/v1/classes/{slug}/semesters` + body `source_semester_id` → call `copySemesterStructure()`
- [ ] **Backend**: `POST /api/v1/semesters/{id}/offerings` — manual CRUD offering (hanya semester DRAFT)
- [ ] **Frontend**: Import JSON (file input + `API.importSemester()` → preview error per baris → apply)
- [ ] **Frontend**: Salin semester (pilih semester sumber → POST `source_semester_id`)
- [ ] **Frontend**: Preview semester modal (offerings, patterns, lecturers, rooms, conflicts, warnings, blockers, can_activate)
- [ ] **Frontend**: Detail arsip read-only (klik semester ARSIP → modal: jadwal/tugas/materi semester tsb)

## Acceptance Criteria
- Import JSON: upload file → preview error/warning per baris → apply all-or-nothing
- Salin semester: pilih sumber → copy struktur (matkul/dosen/offerings/patterns) tanpa tugas/publikasi/PJ
- Preview: conflicts (blocking/non-blocking), warnings, blockers, can_activate flag
- Arsip detail: read-only jadwal pola + event + tugas + materi semester arsip
- Semua validasi client + server konsisten

## Related
- `docs/product/FUNCTIONAL_REQUIREMENTS.md` FR-SEM-001..004
- `docs/product/USER_FLOWS.md` UF-SEM-001/002
- `docs/design/SCREEN_INVENTORY.md` SCR-SEM-001/002
- `internal/academic/semester_lifecycle.go` (Preview & Copy logic)
- `internal/academic/semester_import.go` (Validate/Apply logic)