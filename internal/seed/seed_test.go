package seed

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/database"
	_ "modernc.org/sqlite"
)

// setupMockCurriculum membuat berkas JSON kurikulum kecil untuk pengujian.
func setupMockCurriculum(t *testing.T, filename string) string {
	t.Helper()
	dir := t.TempDir()
	filePath := filepath.Join(dir, filename)

	content := `{
		"kampus": "D4 Semester 3 / Kelas 3A",
		"dosen": {
			"MR": "Muhammad Rizqi, M.T.",
			"TG": "Trisna Gelar, M.Kom."
		},
		"mata_kuliah": {
			"25TI2103": "Aljabar Linear",
			"25TI2104": "Sistem Basis Data"
		},
		"jadwal": [
			{
				"hari": "Senin",
				"jam": "07:00 - 08:40",
				"kode_matkul": "25TI2103",
				"nama_matkul": "Aljabar Linear (Praktikum)",
				"inisial_dosen": "MR",
				"dosen": "Muhammad Rizqi, M.T.",
				"ruang": "D102-Lab"
			},
			{
				"hari": "Selasa",
				"jam": "08:40 - 10:20",
				"kode_matkul": "25TI2103",
				"nama_matkul": "Aljabar Linear (Teori)",
				"inisial_dosen": "TG",
				"dosen": "Trisna Gelar, M.Kom.",
				"ruang": "D101-Kelas"
			},
			{
				"hari": "Rabu",
				"jam": "13:00 - 15:30",
				"kode_matkul": "25TI2104",
				"nama_matkul": "Sistem Basis Data (Praktikum)",
				"inisial_dosen": "MR",
				"dosen": "Muhammad Rizqi, M.T.",
				"ruang": "H501-Lab"
			}
		]
	}`

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Gagal menulis fixture kurikulum: %v", err)
	}
	return filePath
}

// setupLegacyMockDB membuat database SQLite lama tiruan (tugas.db) untuk pengujian migrasi warisan.
func setupLegacyMockDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "legacy_tugas.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Gagal membuat mock legacy DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_jid TEXT NOT NULL,
			class_id TEXT DEFAULT '',
			is_group BOOLEAN NOT NULL,
			matkul TEXT NOT NULL,
			deskripsi TEXT NOT NULL,
			deadline TEXT NOT NULL,
			deadline_at DATETIME,
			created_by TEXT NOT NULL,
			is_done BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE class_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_jid TEXT NOT NULL,
			is_group BOOLEAN NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'umum',
			description TEXT DEFAULT '',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel legacy: %v", err)
	}

	// Masukkan 3 tugas uji:
	// 1. "Aljabar Linear (Praktikum)" -> Cocok tepat 1 offering
	// 2. "Aljabar Linear" -> Ambigu (ada Teori & Praktikum)
	// 3. "Kalkulus Lanjut" -> Tidak ditemukan
	now := time.Now()
	_, err = db.Exec(`
		INSERT INTO tasks (scope_jid, class_id, is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at)
		VALUES
		('group@jid', 'D4-TI-SMT3-A', 1, 'Aljabar Linear (Praktikum)', 'Tugas Modul 1', 'Besok', ?, 'User1', 0, ?),
		('group@jid', 'D4-TI-SMT3-A', 1, 'Aljabar Linear', 'PR Bab 2', 'Lusa', ?, 'User2', 0, ?),
		('group@jid', 'D4-TI-SMT3-A', 1, 'Kalkulus Lanjut', 'Latihan Soal', 'Minggu Depan', ?, 'User3', 0, ?);
	`, now.Add(24*time.Hour), now, now.Add(48*time.Hour), now, now.Add(72*time.Hour), now)
	if err != nil {
		t.Fatalf("Gagal insert mock tasks: %v", err)
	}

	// Masukkan 1 tautan uji
	_, _ = db.Exec(`
		INSERT INTO class_links (scope_jid, is_group, title, url, category, description, created_by)
		VALUES ('D4-TI-SMT3-A', 1, 'Google Drive Kelas', 'https://drive.google.com/test', 'drive', 'Arsip materi', 'User1');
	`)

	return dbPath
}

// Kasus 1: Seed Kelas pilot menghasilkan Kelas + Semester aktif + offering + pola yang bisa diquery.
func TestSeed_Case1_PilotClassGeneratesQueryableRows(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPath,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	report, err := SeedClasses(manifest, "checksum-abc", db, "")
	if err != nil {
		t.Fatalf("SeedClasses gagal: %v", err)
	}

	if len(report.Classes) != 1 {
		t.Fatalf("Expected 1 class report, got %d", len(report.Classes))
	}

	// 1. Verifikasi Kelas
	var classCount int
	var slug, status string
	err = db.QueryRow("SELECT COUNT(*), slug, status FROM classes WHERE code = 'D4-TI-2024-A'").Scan(&classCount, &slug, &status)
	if err != nil || classCount != 1 || slug != "d4-ti-2024-a" || status != "ACTIVE" {
		t.Errorf("Verifikasi classes gagal: count=%d, slug=%s, status=%s, err=%v", classCount, slug, status, err)
	}

	// 2. Verifikasi Semester Aktif
	var semCount int
	var term, semStatus string
	err = db.QueryRow("SELECT COUNT(*), term, status FROM semesters WHERE academic_year = '2026/2027' AND status = 'ACTIVE'").Scan(&semCount, &term, &semStatus)
	if err != nil || semCount != 1 || term != "GANJIL" || semStatus != "ACTIVE" {
		t.Errorf("Verifikasi semesters gagal: count=%d, term=%s, status=%s, err=%v", semCount, term, semStatus, err)
	}

	// 3. Verifikasi Course Offerings & Pola Jadwal
	var offeringCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM course_offerings").Scan(&offeringCount)
	if offeringCount < 2 {
		t.Errorf("Expected at least 2 course offerings, got %d", offeringCount)
	}

	var patternCount int
	_ = db.QueryRow("SELECT COUNT(*) FROM schedule_patterns WHERE status = 'ACTIVE'").Scan(&patternCount)
	if patternCount != 3 {
		t.Errorf("Expected 3 schedule patterns, got %d", patternCount)
	}
}

// Kasus 2: Jalan ulang tidak menggandakan data (idempotensi).
func TestSeed_Case2_ReRunIsIdempotent(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPath,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	// Eksekusi pertama
	_, err = SeedClasses(manifest, "chk-1", db, "")
	if err != nil {
		t.Fatalf("Seed pertama gagal: %v", err)
	}

	var countClasses1, countSemesters1, countCourses1, countOfferings1, countPatterns1 int
	_ = db.QueryRow("SELECT COUNT(*) FROM classes").Scan(&countClasses1)
	_ = db.QueryRow("SELECT COUNT(*) FROM semesters").Scan(&countSemesters1)
	_ = db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&countCourses1)
	_ = db.QueryRow("SELECT COUNT(*) FROM course_offerings").Scan(&countOfferings1)
	_ = db.QueryRow("SELECT COUNT(*) FROM schedule_patterns").Scan(&countPatterns1)

	// Eksekusi kedua (jalan ulang dengan konfigurasi yang sama)
	_, err = SeedClasses(manifest, "chk-2", db, "")
	if err != nil {
		t.Fatalf("Seed kedua (jalan ulang) gagal: %v", err)
	}

	var countClasses2, countSemesters2, countCourses2, countOfferings2, countPatterns2 int
	_ = db.QueryRow("SELECT COUNT(*) FROM classes").Scan(&countClasses2)
	_ = db.QueryRow("SELECT COUNT(*) FROM semesters").Scan(&countSemesters2)
	_ = db.QueryRow("SELECT COUNT(*) FROM courses").Scan(&countCourses2)
	_ = db.QueryRow("SELECT COUNT(*) FROM course_offerings").Scan(&countOfferings2)
	_ = db.QueryRow("SELECT COUNT(*) FROM schedule_patterns").Scan(&countPatterns2)

	if countClasses1 != countClasses2 ||
		countSemesters1 != countSemesters2 ||
		countCourses1 != countCourses2 ||
		countOfferings1 != countOfferings2 ||
		countPatterns1 != countPatterns2 {
		t.Errorf("Idempotensi gagal! Sebelum: (%d, %d, %d, %d, %d), Sesudah: (%d, %d, %d, %d, %d)",
			countClasses1, countSemesters1, countCourses1, countOfferings1, countPatterns1,
			countClasses2, countSemesters2, countCourses2, countOfferings2, countPatterns2)
	}
}

// Kasus 3: Manifest tanpa field wajib ditolak sebelum menulis ke database.
func TestSeed_Case3_ManifestMissingRequiredFieldsRejectedBeforeWrite(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	// Manifest tidak memiliki StudyProgram dan AcademicYear
	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:  "D4-TI-2024-A",
				Slug:       "d4-ti-2024-a",
				CohortYear: 2024,
				GroupLabel: "A",
				SourceFile: curriculumPath,
				// StudyProgram sengaja kosong
				// AcademicYear sengaja kosong
				Term:     "GANJIL",
				StartsOn: "2026-09-01",
				EndsOn:   "2027-01-31",
			},
		},
	}

	_, err = SeedClasses(manifest, "chk", db, "")
	if err == nil {
		t.Fatalf("Expected error for missing required fields, got nil")
	}

	// Verifikasi tidak ada baris yang tertulis di tabel classes
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM classes").Scan(&count)
	if count != 0 {
		t.Errorf("Expected 0 classes written, got %d", count)
	}
}

// Kasus 4: Kode angkatan / rentang tanggal yang tidak valid ditolak.
func TestSeed_Case4_InvalidCohortYearRejected(t *testing.T) {
	curriculumPath := setupMockCurriculum(t, "pilot_a.json")

	// 1. Tahun angkatan < 2000
	m1 := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:    "D4-TI-1995-A",
				Slug:         "d4-ti-1995-a",
				StudyProgram: "D4 TI",
				CohortYear:   1995,
				GroupLabel:   "A",
				SourceFile:   curriculumPath,
				AcademicYear: "2026/2027",
				Term:         "GANJIL",
				StartsOn:     "2026-09-01",
				EndsOn:       "2027-01-31",
			},
		},
	}
	if err := ValidateManifest(m1); err == nil {
		t.Errorf("Expected validation failure for cohort_year=1995")
	}

	// 2. Tanggal ends_on tidak setelah starts_on
	m2 := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:    "D4-TI-2024-A",
				Slug:         "d4-ti-2024-a",
				StudyProgram: "D4 TI",
				CohortYear:   2024,
				GroupLabel:   "A",
				SourceFile:   curriculumPath,
				AcademicYear: "2026/2027",
				Term:         "GANJIL",
				StartsOn:     "2026-09-01",
				EndsOn:       "2026-08-01", // Lebih awal dari starts_on!
			},
		},
	}
	if err := ValidateManifest(m2); err == nil {
		t.Errorf("Expected validation failure when ends_on <= starts_on")
	}
}

// Kasus 5: Master mata kuliah & dosen tidak ganda saat 2 kelas pilot di-seed.
func TestSeed_Case5_CoursesAndLecturersDeduplicatedAcrossSeeds(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPathA := setupMockCurriculum(t, "pilot_a.json")
	curriculumPathB := setupMockCurriculum(t, "pilot_b.json")

	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPathA,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
			{
				ClassCode:        "D4-TI-2024-B",
				Slug:             "d4-ti-2024-b",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "B",
				SourceFile:       curriculumPathB,
				LegacyClassID:    "D4-TI-SMT3-B",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	_, err = SeedClasses(manifest, "chk-ab", db, "")
	if err != nil {
		t.Fatalf("SeedClasses gagal: %v", err)
	}

	// Kedua file kurikulum memuat mata kuliah '25TI2103' dan dosen 'MR'.
	// Di tabel master, keduanya HARUS tepat 1 baris.
	var countCourse int
	_ = db.QueryRow("SELECT COUNT(*) FROM courses WHERE code = '25TI2103'").Scan(&countCourse)
	if countCourse != 1 {
		t.Errorf("Expected exactly 1 master row for course 25TI2103, got %d", countCourse)
	}

	var countLecturer int
	_ = db.QueryRow("SELECT COUNT(*) FROM lecturers WHERE code = 'MR'").Scan(&countLecturer)
	if countLecturer != 1 {
		t.Errorf("Expected exactly 1 master row for lecturer MR, got %d", countLecturer)
	}

	// Tetapi setiap kelas memiliki course_offering sendiri-sendiri
	var countOfferings int
	_ = db.QueryRow("SELECT COUNT(*) FROM course_offerings WHERE course_id = (SELECT id FROM courses WHERE code = '25TI2103')").Scan(&countOfferings)
	if countOfferings < 2 {
		t.Errorf("Expected offerings across 2 classes, got %d", countOfferings)
	}
}

// Kasus 6: Tugas tanpa pasangan offering masuk antrean, bukan ke offering yang mirip.
func TestSeed_Case6_UnmatchedTasksLandInResolutionQueue(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	legacyDBPath := setupLegacyMockDB(t)

	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPath,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	report, err := SeedClasses(manifest, "chk", db, legacyDBPath)
	if err != nil {
		t.Fatalf("SeedClasses gagal: %v", err)
	}

	classRep := report.Classes[0]

	// 1 tugas cocok tepat ("Aljabar Linear (Praktikum)") -> berhasil masuk tasks
	if classRep.TargetCounts.Tasks != 1 {
		t.Errorf("Expected 1 imported task, got %d", classRep.TargetCounts.Tasks)
	}

	// 2 tugas tidak cocok ("Aljabar Linear" [ambigu] dan "Kalkulus Lanjut" [tidak ada]) -> masuk antrean resolusi
	if len(classRep.ResolutionQueue) != 2 {
		t.Errorf("Expected 2 items in resolution queue, got %d", len(classRep.ResolutionQueue))
	}

	// Pastikan tercatat ke import_errors
	var countErrors int
	_ = db.QueryRow("SELECT COUNT(*) FROM import_errors WHERE error_code = 'RESOLUTION_QUEUE'").Scan(&countErrors)
	if countErrors != 2 {
		t.Errorf("Expected 2 resolution queue records in import_errors, got %d", countErrors)
	}
}

// Kasus 7: Laporan memuat sumber, target, gagal, dan checksum.
func TestSeed_Case7_ReportContainsCountsAndChecksum(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPath,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	report, err := SeedClasses(manifest, "manifest-sha256-test", db, "")
	if err != nil {
		t.Fatalf("SeedClasses gagal: %v", err)
	}

	if report.ManifestChecksum != "manifest-sha256-test" {
		t.Errorf("ManifestChecksum mismatch: got %s", report.ManifestChecksum)
	}

	classRep := report.Classes[0]
	if classRep.SourceChecksum == "" || classRep.SourceChecksum == "unknown" {
		t.Errorf("Expected valid source checksum, got %s", classRep.SourceChecksum)
	}
	if classRep.SourceCounts.Courses == 0 || classRep.SourceCounts.Schedules == 0 {
		t.Errorf("Expected non-zero source counts, got %+v", classRep.SourceCounts)
	}
	if classRep.TargetCounts.Courses == 0 || classRep.TargetCounts.SchedulePatterns == 0 {
		t.Errorf("Expected non-zero target counts, got %+v", classRep.TargetCounts)
	}
}

// Kasus 8: Database lama tidak berubah (byte-identical).
func TestSeed_Case8_LegacyDatabaseRemainsByteIdentical(t *testing.T) {
	targetDBPath := filepath.Join(t.TempDir(), "target_v1.db")
	db, err := database.InitDB(targetDBPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	curriculumPath := setupMockCurriculum(t, "pilot_a.json")
	legacyDBPath := setupLegacyMockDB(t)

	hashBefore, err := ComputeFileHash(legacyDBPath)
	if err != nil {
		t.Fatalf("Gagal menghitung hash sebelum seed: %v", err)
	}

	manifest := &Manifest{
		Classes: []ClassMapping{
			{
				ClassCode:        "D4-TI-2024-A",
				Slug:             "d4-ti-2024-a",
				StudyProgram:     "D4 Teknik Informatika",
				CohortYear:       2024,
				GroupLabel:       "A",
				SourceFile:       curriculumPath,
				LegacyClassID:    "D4-TI-SMT3-A",
				AcademicYear:     "2026/2027",
				Term:             "GANJIL",
				StartsOn:         "2026-09-01",
				EndsOn:           "2027-01-31",
				Timezone:         "Asia/Jakarta",
				PortalAccessMode: "LINK",
			},
		},
	}

	_, err = SeedClasses(manifest, "chk", db, legacyDBPath)
	if err != nil {
		t.Fatalf("SeedClasses gagal: %v", err)
	}

	hashAfter, err := ComputeFileHash(legacyDBPath)
	if err != nil {
		t.Fatalf("Gagal menghitung hash sesudah seed: %v", err)
	}

	if hashBefore != hashAfter {
		t.Errorf("PELANGGARAN: Database lama termodifikasi! Sebelum=%s, Sesudah=%s", hashBefore, hashAfter)
	}
}
