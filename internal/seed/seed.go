package seed

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/database"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/util"
	_ "modernc.org/sqlite"
)

// Manifest merepresentasikan konfigurasi pemetaan seed pilot dari JSON ke database target v1.
type Manifest struct {
	Classes []ClassMapping `json:"classes"`
}

// ClassMapping merepresentasikan pemetaan satu kelas pilot.
type ClassMapping struct {
	ClassCode        string `json:"class_code"`
	Slug             string `json:"slug"`
	StudyProgram     string `json:"study_program"`
	CohortYear       int    `json:"cohort_year"`
	GroupLabel       string `json:"group_label"`
	SourceFile       string `json:"source_file"`
	LegacyClassID    string `json:"legacy_class_id"`
	AcademicYear     string `json:"academic_year"`
	Term             string `json:"term"`
	StartsOn         string `json:"starts_on"`
	EndsOn           string `json:"ends_on"`
	Timezone         string `json:"timezone"`
	PortalAccessMode string `json:"portal_access_mode"`
}

// SourceCounts mencatat jumlah item yang dibaca dari sumber data.
type SourceCounts struct {
	Courses   int `json:"courses"`
	Lecturers int `json:"lecturers"`
	Schedules int `json:"schedules"`
	Tasks     int `json:"tasks"`
	Links     int `json:"links"`
}

// TargetCounts mencatat jumlah baris yang tersimpan pada tabel target.
type TargetCounts struct {
	Classes           int `json:"classes"`
	Semesters         int `json:"semesters"`
	Courses           int `json:"courses"`
	Lecturers         int `json:"lecturers"`
	CourseOfferings   int `json:"course_offerings"`
	OfferingLecturers int `json:"offering_lecturers"`
	SchedulePatterns  int `json:"schedule_patterns"`
	Tasks             int `json:"tasks"`
	Materials         int `json:"materials"`
}

// FailedRow mencatat rincian baris sumber yang gagal diimpor atau tidak valid.
type FailedRow struct {
	SourceFile string `json:"source_file"`
	RowNumber  int    `json:"row_number"`
	Field      string `json:"field"`
	ErrorCode  string `json:"error_code"`
	Message    string `json:"message"`
	Severity   string `json:"severity"` // "ERROR" atau "WARNING"
}

// ResolutionQueueItem mencatat entri yang memerlukan penelaahan manusia karena tidak cocok otomatis.
type ResolutionQueueItem struct {
	SourceType  string `json:"source_type"` // e.g. "TASK", "LINK"
	SourceID    string `json:"source_id"`
	Subject     string `json:"subject"`
	Description string `json:"description"`
	Reason      string `json:"reason"`
	Status      string `json:"status"` // "PENDING"
}

// ClassSeedReport mencatat hasil impor dan rekonsiliasi satu kelas pilot.
type ClassSeedReport struct {
	ClassCode       string                `json:"class_code"`
	Slug            string                `json:"slug"`
	SourceFile      string                `json:"source_file"`
	SourceChecksum  string                `json:"source_checksum"`
	SourceCounts    SourceCounts          `json:"source_counts"`
	TargetCounts    TargetCounts          `json:"target_counts"`
	FailedRows      []FailedRow           `json:"failed_rows"`
	ResolutionQueue []ResolutionQueueItem `json:"resolution_queue"`
}

// SeedReport adalah laporan keseluruhan proses seed.
type SeedReport struct {
	ManifestChecksum string            `json:"manifest_checksum"`
	Classes          []ClassSeedReport `json:"classes"`
	GeneratedAt      time.Time         `json:"generated_at"`
}

// LoadManifest membaca dan mengurai manifest dari path file.
func LoadManifest(manifestPath string) (*Manifest, error) {
	resolvedPath := util.FindDataDir(manifestPath)
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file manifest: %w", err)
	}

	var manifest Manifest
	// Coba decode sebagai objek { "classes": [...] }
	if err := json.Unmarshal(data, &manifest); err == nil && len(manifest.Classes) > 0 {
		return &manifest, nil
	}

	// Coba decode sebagai array langsung [ {...}, {...} ]
	var classes []ClassMapping
	if err := json.Unmarshal(data, &classes); err == nil && len(classes) > 0 {
		return &Manifest{Classes: classes}, nil
	}

	return nil, fmt.Errorf("format manifest JSON tidak valid atau tidak memiliki kelas")
}

// ValidateManifest memeriksa apakah manifest memenuhi seluruh persyaratan L1.
// Mengembalikan error komprehensif jika ada field wajib yang hilang atau nilai tidak valid.
func ValidateManifest(manifest *Manifest) error {
	if manifest == nil || len(manifest.Classes) == 0 {
		return fmt.Errorf("manifest kosong atau tidak memuat kelas pilot")
	}

	var allErrors []string

	for idx, cm := range manifest.Classes {
		prefix := fmt.Sprintf("kelas[%d] (%s)", idx, cm.ClassCode)
		var missing []string

		if strings.TrimSpace(cm.StudyProgram) == "" {
			missing = append(missing, "study_program")
		}
		if cm.CohortYear <= 0 {
			missing = append(missing, "cohort_year")
		}
		if strings.TrimSpace(cm.GroupLabel) == "" {
			missing = append(missing, "group_label")
		}
		if strings.TrimSpace(cm.Slug) == "" {
			missing = append(missing, "slug")
		}
		if strings.TrimSpace(cm.ClassCode) == "" {
			missing = append(missing, "class_code")
		}
		if strings.TrimSpace(cm.SourceFile) == "" {
			missing = append(missing, "source_file")
		}
		if strings.TrimSpace(cm.AcademicYear) == "" {
			missing = append(missing, "academic_year")
		}
		if strings.TrimSpace(cm.Term) == "" {
			missing = append(missing, "term")
		}
		if strings.TrimSpace(cm.StartsOn) == "" {
			missing = append(missing, "starts_on")
		}
		if strings.TrimSpace(cm.EndsOn) == "" {
			missing = append(missing, "ends_on")
		}

		if len(missing) > 0 {
			allErrors = append(allErrors, fmt.Sprintf("%s: field wajib hilang: %s", prefix, strings.Join(missing, ", ")))
			continue
		}

		// Validasi format tahun angkatan
		if cm.CohortYear < 2000 || cm.CohortYear > 2100 {
			allErrors = append(allErrors, fmt.Sprintf("%s: cohort_year '%d' tidak valid (harus antara 2000-2100)", prefix, cm.CohortYear))
		}

		// Validasi format tanggal
		startDate, errStart := time.Parse("2006-01-02", cm.StartsOn)
		if errStart != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: format starts_on harus YYYY-MM-DD", prefix))
		}
		endDate, errEnd := time.Parse("2006-01-02", cm.EndsOn)
		if errEnd != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: format ends_on harus YYYY-MM-DD", prefix))
		}
		if errStart == nil && errEnd == nil {
			if !endDate.After(startDate) {
				allErrors = append(allErrors, fmt.Sprintf("%s: ends_on (%s) harus setelah starts_on (%s)", prefix, cm.EndsOn, cm.StartsOn))
			}
		}

		// Validasi keberadaan file kurikulum sumber
		if _, err := os.Stat(util.FindDataDir(cm.SourceFile)); err != nil {
			allErrors = append(allErrors, fmt.Sprintf("%s: berkas kurikulum sumber tidak ditemukan: %s", prefix, cm.SourceFile))
		}
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("validasi manifest gagal:\n- %s", strings.Join(allErrors, "\n- "))
	}

	return nil
}

// ComputeFileHash menghitung SHA-256 checksum dari sebuah berkas.
func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SeedOptions memuat opsi parameter eksekusi seed.
type SeedOptions struct {
	ManifestPath string
	TargetDB     *sql.DB
	LegacyDBPath string
}

// SeedClasses menjalankan proses seed pilot untuk seluruh kelas yang didefinisikan dalam manifest.
func SeedClasses(manifest *Manifest, manifestChecksum string, targetDB *sql.DB, legacyDBPath string) (*SeedReport, error) {
	if err := ValidateManifest(manifest); err != nil {
		return nil, err
	}

	// Pastikan migrasi v1 telah dijalankan pada database target
	if err := database.MigrateV1(targetDB); err != nil {
		return nil, fmt.Errorf("gagal migrasi skema v1 pada database target: %w", err)
	}

	// Jika legacy DB diberikan dan ada di disk, catat hash untuk verifikasi byte-identical
	var legacyHashBefore string
	if legacyDBPath != "" {
		if fi, err := os.Stat(legacyDBPath); err == nil && !fi.IsDir() {
			hash, err := ComputeFileHash(legacyDBPath)
			if err == nil {
				legacyHashBefore = hash
			}
		}
	}

	report := &SeedReport{
		ManifestChecksum: manifestChecksum,
		GeneratedAt:      time.Now(),
		Classes:          make([]ClassSeedReport, 0, len(manifest.Classes)),
	}

	for _, cm := range manifest.Classes {
		classReport, err := seedSingleClass(cm, targetDB, legacyDBPath)
		if err != nil {
			return nil, fmt.Errorf("gagal melakukan seed untuk kelas '%s': %w", cm.ClassCode, err)
		}
		report.Classes = append(report.Classes, *classReport)
	}

	// Verifikasi bahwa legacy DB tetap tidak berubah (byte-identical)
	if legacyHashBefore != "" {
		legacyHashAfter, err := ComputeFileHash(legacyDBPath)
		if err == nil && legacyHashBefore != legacyHashAfter {
			return nil, fmt.Errorf("PELANGGARAN KEAMANAN: database lama terubah selama proses seed (hash mismatch)")
		}
	}

	return report, nil
}

// seedSingleClass mengeksekusi seed untuk satu kelas pilot dalam transaksi database tunggal.
func seedSingleClass(cm ClassMapping, targetDB *sql.DB, legacyDBPath string) (*ClassSeedReport, error) {
	sourceHash, err := ComputeFileHash(util.FindDataDir(cm.SourceFile))
	if err != nil {
		sourceHash = "unknown"
	}

	cfg, err := schedule.LoadJadwal(cm.SourceFile)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca berkas kurikulum JSON '%s': %w", cm.SourceFile, err)
	}

	tx, err := targetDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	classReport := &ClassSeedReport{
		ClassCode:      cm.ClassCode,
		Slug:           cm.Slug,
		SourceFile:     cm.SourceFile,
		SourceChecksum: sourceHash,
		SourceCounts: SourceCounts{
			Courses:   len(cfg.MataKuliah),
			Lecturers: len(cfg.Dosen),
			Schedules: len(cfg.Jadwal),
		},
	}

	// 1. Tulis atau perbarui Kelas permanen
	var classID int64
	err = tx.QueryRow(`
		INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET
			slug = excluded.slug,
			study_program = excluded.study_program,
			cohort_year = excluded.cohort_year,
			group_label = excluded.group_label,
			status = 'ACTIVE'
		RETURNING id;
	`, cm.ClassCode, cm.Slug, cm.StudyProgram, cm.CohortYear, cm.GroupLabel).Scan(&classID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan data kelas: %w", err)
	}

	var seedActorID int64
	err = tx.QueryRow(`
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES ('system:seed', 'Seed Import', 'disabled', 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET display_name = excluded.display_name
		RETURNING id;
	`).Scan(&seedActorID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyiapkan pelaku seed: %w", err)
	}

	// 2. Tulis atau perbarui Pengaturan Kelas (class_settings)
	timezone := strings.TrimSpace(cm.Timezone)
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}
	portalMode := strings.ToUpper(strings.TrimSpace(cm.PortalAccessMode))
	if portalMode == "" {
		portalMode = "LINK"
	}
	_, err = tx.Exec(`
		INSERT INTO class_settings (class_id, timezone, portal_access_mode, version)
		VALUES (?, ?, ?, 1)
		ON CONFLICT(class_id) DO UPDATE SET
			timezone = excluded.timezone,
			portal_access_mode = excluded.portal_access_mode,
			portal_code_hash = CASE WHEN excluded.portal_access_mode = 'LINK' THEN NULL ELSE portal_code_hash END;
	`, classID, timezone, portalMode)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan pengaturan kelas: %w", err)
	}

	// 3. Tulis atau perbarui Semester Aktif
	// Pastikan hanya satu semester aktif untuk kelas ini
	_, _ = tx.Exec(`
		UPDATE semesters
		SET status = 'ARCHIVED', archived_at = COALESCE(archived_at, CURRENT_TIMESTAMP)
		WHERE class_id = ? AND status = 'ACTIVE';
	`, classID)

	var semesterID int64
	err = tx.QueryRow(`
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, version)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 1)
		ON CONFLICT(class_id, academic_year, term) DO UPDATE SET
			starts_on = excluded.starts_on,
			ends_on = excluded.ends_on,
			status = 'ACTIVE',
			published_at = COALESCE(semesters.published_at, CURRENT_TIMESTAMP),
			activated_at = CURRENT_TIMESTAMP,
			archived_at = NULL
		RETURNING id;
	`, classID, cm.AcademicYear, cm.Term, cm.StartsOn, cm.EndsOn).Scan(&semesterID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan semester aktif: %w", err)
	}

	// 4. Impor Master Mata Kuliah (courses) secara idempoten
	courseIDMap := make(map[string]int64)
	for code, name := range cfg.MataKuliah {
		cleanCode := strings.TrimSpace(code)
		cleanName := strings.TrimSpace(name)
		var cID int64
		err = tx.QueryRow(`
			INSERT INTO courses (code, name, status)
			VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, status = 'ACTIVE'
			RETURNING id;
		`, cleanCode, cleanName).Scan(&cID)
		if err != nil {
			return nil, fmt.Errorf("gagal menyimpan master mata kuliah '%s': %w", cleanCode, err)
		}
		courseIDMap[cleanCode] = cID
	}

	// 5. Impor Master Dosen (lecturers) secara idempoten
	lecturerIDMap := make(map[string]int64)
	for inisial, fullName := range cfg.Dosen {
		cleanCode := strings.TrimSpace(inisial)
		cleanName := strings.TrimSpace(fullName)
		var lID int64
		err = tx.QueryRow(`
			INSERT INTO lecturers (code, full_name, status)
			VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, status = 'ACTIVE'
			RETURNING id;
		`, cleanCode, cleanName).Scan(&lID)
		if err != nil {
			return nil, fmt.Errorf("gagal menyimpan master dosen '%s': %w", cleanCode, err)
		}
		lecturerIDMap[cleanCode] = lID
	}

	// 6. Impor Course Offerings, Offering Lecturers, dan Pola Jadwal (schedule_patterns)
	type offeringKey struct {
		CourseID     int64
		ActivityType string
	}
	offeringIDMap := make(map[offeringKey]int64)

	for rowIdx, item := range cfg.Jadwal {
		cleanCourseCode := strings.TrimSpace(item.KodeMatkul)
		courseID, exists := courseIDMap[cleanCourseCode]
		if !exists {
			// Jika belum terdaftar di map mata_kuliah, buat master baru dari item
			cleanName := cleanSubjectName(item.NamaMatkul)
			var cID int64
			err = tx.QueryRow(`
				INSERT INTO courses (code, name, status)
				VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET name = excluded.name, status = 'ACTIVE'
				RETURNING id;
			`, cleanCourseCode, cleanName).Scan(&cID)
			if err != nil {
				classReport.FailedRows = append(classReport.FailedRows, FailedRow{
					SourceFile: cm.SourceFile,
					RowNumber:  rowIdx + 1,
					Field:      "kode_matkul",
					ErrorCode:  "COURSE_CREATE_FAIL",
					Message:    fmt.Sprintf("Gagal membuat course '%s': %v", cleanCourseCode, err),
					Severity:   "ERROR",
				})
				continue
			}
			courseID = cID
			courseIDMap[cleanCourseCode] = cID
		}

		activityType := detectActivityType(item.NamaMatkul)
		offKey := offeringKey{CourseID: courseID, ActivityType: activityType}
		offeringID, offExists := offeringIDMap[offKey]

		if !offExists {
			displayName := strings.TrimSpace(item.NamaMatkul)
			if displayName == "" {
				displayName = cfg.MataKuliah[cleanCourseCode]
			}
			var oID int64
			err = tx.QueryRow(`
				INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status, version)
				VALUES (?, ?, ?, ?, 'ACTIVE', 1)
				ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET
					display_name = excluded.display_name,
					status = 'ACTIVE'
				RETURNING id;
			`, semesterID, courseID, activityType, displayName).Scan(&oID)
			if err != nil {
				classReport.FailedRows = append(classReport.FailedRows, FailedRow{
					SourceFile: cm.SourceFile,
					RowNumber:  rowIdx + 1,
					Field:      "course_offering",
					ErrorCode:  "OFFERING_CREATE_FAIL",
					Message:    fmt.Sprintf("Gagal membuat course offering: %v", err),
					Severity:   "ERROR",
				})
				continue
			}
			offeringID = oID
			offeringIDMap[offKey] = oID
		}

		// Hubungkan Dosen ke Offering
		cleanInisial := strings.TrimSpace(item.InisialDosen)
		if cleanInisial != "" {
			lecturerID, lExists := lecturerIDMap[cleanInisial]
			if !lExists {
				var lID int64
				lecturerName := strings.TrimSpace(item.Dosen)
				if lecturerName == "" {
					lecturerName = cleanInisial
				}
				err = tx.QueryRow(`
					INSERT INTO lecturers (code, full_name, status)
					VALUES (?, ?, 'ACTIVE')
					ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, status = 'ACTIVE'
					RETURNING id;
				`, cleanInisial, lecturerName).Scan(&lID)
				if err == nil {
					lecturerID = lID
					lecturerIDMap[cleanInisial] = lID
					lExists = true
				}
			}
			if lExists {
				_, _ = tx.Exec(`
					INSERT OR IGNORE INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
					VALUES (?, ?, 'PRIMARY');
				`, offeringID, lecturerID)
			}
		}

		// Validasi Pola Waktu Jadwal
		dayOfWeek := parseDayOfWeek(item.Hari)
		if dayOfWeek < 1 || dayOfWeek > 7 {
			classReport.FailedRows = append(classReport.FailedRows, FailedRow{
				SourceFile: cm.SourceFile,
				RowNumber:  rowIdx + 1,
				Field:      "hari",
				ErrorCode:  "INVALID_DAY",
				Message:    fmt.Sprintf("Nama hari tidak valid '%s'", item.Hari),
				Severity:   "ERROR",
			})
			continue
		}

		startTime, endTime, errTime := parseJam(item.Jam)
		if errTime != nil {
			classReport.FailedRows = append(classReport.FailedRows, FailedRow{
				SourceFile: cm.SourceFile,
				RowNumber:  rowIdx + 1,
				Field:      "jam",
				ErrorCode:  "INVALID_TIME_FORMAT",
				Message:    fmt.Sprintf("Format jam tidak valid '%s': %v", item.Jam, errTime),
				Severity:   "ERROR",
			})
			continue
		}

		if endTime <= startTime {
			classReport.FailedRows = append(classReport.FailedRows, FailedRow{
				SourceFile: cm.SourceFile,
				RowNumber:  rowIdx + 1,
				Field:      "jam",
				ErrorCode:  "TIME_END_NOT_AFTER_START",
				Message:    fmt.Sprintf("Waktu selesai (%s) harus setelah waktu mulai (%s)", endTime, startTime),
				Severity:   "ERROR",
			})
			continue
		}

		// Cari room_id jika terdaftar di master rooms, jangan buat baru (User Story 13)
		var roomID sql.NullInt64
		cleanRoom := strings.TrimSpace(item.Ruang)
		if cleanRoom != "" {
			var rID int64
			err = tx.QueryRow(`SELECT id FROM rooms WHERE code = ?;`, cleanRoom).Scan(&rID)
			if err == nil {
				roomID = sql.NullInt64{Int64: rID, Valid: true}
			}
		}

		// Simpan Pola Jadwal secara idempoten
		var existingPatternID int64
		pErr := tx.QueryRow(`
			SELECT id FROM schedule_patterns
			WHERE course_offering_id = ? AND day_of_week = ? AND start_time = ?;
		`, offeringID, dayOfWeek, startTime).Scan(&existingPatternID)
		if pErr == nil {
			_, err = tx.Exec(`
				UPDATE schedule_patterns
				SET end_time = ?, room_id = ?, effective_from = ?, effective_until = ?, status = 'ACTIVE'
				WHERE id = ?;
			`, endTime, roomID, cm.StartsOn, cm.EndsOn, existingPatternID)
		} else {
			_, err = tx.Exec(`
				INSERT INTO schedule_patterns (
					course_offering_id, room_id, day_of_week, start_time, end_time,
					effective_from, effective_until, status, version
				)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE', 1);
			`, offeringID, roomID, dayOfWeek, startTime, endTime, cm.StartsOn, cm.EndsOn)
		}
		if err != nil {
			classReport.FailedRows = append(classReport.FailedRows, FailedRow{
				SourceFile: cm.SourceFile,
				RowNumber:  rowIdx + 1,
				Field:      "schedule_pattern",
				ErrorCode:  "PATTERN_INSERT_FAIL",
				Message:    fmt.Sprintf("Gagal menyimpan pola jadwal: %v", err),
				Severity:   "ERROR",
			})
		}
	}

	// 7. Migrasi Data Warisan (Legacy Tasks & Links) jika legacy DB tersedia
	if legacyDBPath != "" {
		if _, statErr := os.Stat(legacyDBPath); statErr == nil {
			migrateLegacyData(tx, cm, classID, semesterID, seedActorID, legacyDBPath, classReport)
		}
	}

	// 8. Hitung Jumlah Baris Target Aktual
	_ = tx.QueryRow(`SELECT COUNT(*) FROM classes WHERE id = ?;`, classID).Scan(&classReport.TargetCounts.Classes)
	_ = tx.QueryRow(`SELECT COUNT(*) FROM semesters WHERE id = ?;`, semesterID).Scan(&classReport.TargetCounts.Semesters)
	_ = tx.QueryRow(`SELECT COUNT(*) FROM courses;`).Scan(&classReport.TargetCounts.Courses)
	_ = tx.QueryRow(`SELECT COUNT(*) FROM lecturers;`).Scan(&classReport.TargetCounts.Lecturers)
	_ = tx.QueryRow(`SELECT COUNT(*) FROM course_offerings WHERE semester_id = ?;`, semesterID).Scan(&classReport.TargetCounts.CourseOfferings)
	_ = tx.QueryRow(`
		SELECT COUNT(*) FROM offering_lecturers ol
		JOIN course_offerings co ON ol.course_offering_id = co.id
		WHERE co.semester_id = ?;
	`, semesterID).Scan(&classReport.TargetCounts.OfferingLecturers)
	_ = tx.QueryRow(`
		SELECT COUNT(*) FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		WHERE co.semester_id = ?;
	`, semesterID).Scan(&classReport.TargetCounts.SchedulePatterns)
	_ = tx.QueryRow(`
		SELECT COUNT(*) FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		WHERE co.semester_id = ?;
	`, semesterID).Scan(&classReport.TargetCounts.Tasks)
	_ = tx.QueryRow(`SELECT COUNT(*) FROM materials WHERE class_id = ?;`, classID).Scan(&classReport.TargetCounts.Materials)

	// 9. Catat ke import_batches dan import_errors
	summaryJSON, _ := json.Marshal(map[string]interface{}{
		"source_counts":    classReport.SourceCounts,
		"target_counts":    classReport.TargetCounts,
		"failed_rows":      len(classReport.FailedRows),
		"resolution_queue": len(classReport.ResolutionQueue),
		"checksum":         sourceHash,
	})

	var batchID int64
	err = tx.QueryRow(`
		INSERT INTO import_batches (
			class_id, semester_id, source_type, source_checksum, status,
			created_by_user_id, summary_json
		)
		VALUES (?, ?, 'CURRICULUM_SEED', ?, 'APPLIED', ?, ?)
		RETURNING id;
	`, classID, semesterID, sourceHash, seedActorID, string(summaryJSON)).Scan(&batchID)
	if err == nil {
		for _, f := range classReport.FailedRows {
			_, _ = tx.Exec(`
				INSERT INTO import_errors (batch_id, source_location, field_name, error_code, message, severity)
				VALUES (?, ?, ?, ?, ?, ?);
			`, batchID, fmt.Sprintf("row:%d", f.RowNumber), f.Field, f.ErrorCode, f.Message, f.Severity)
		}
		for _, q := range classReport.ResolutionQueue {
			_, _ = tx.Exec(`
				INSERT INTO import_errors (batch_id, source_location, field_name, error_code, message, severity)
				VALUES (?, ?, ?, 'RESOLUTION_QUEUE', ?, 'WARNING');
			`, batchID, fmt.Sprintf("legacy:%s:%s", q.SourceType, q.SourceID), q.SourceType,
				fmt.Sprintf("[%s] %s: %s", q.SourceID, q.Subject, q.Reason))
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal melakukan commit transaksi seed: %w", err)
	}

	return classReport, nil
}

// OfferingInfo memuat ringkasan offering untuk pencocokan tugas warisan.
type OfferingInfo struct {
	ID          int64
	CourseCode  string
	CourseName  string
	DisplayName string
}

// migrateLegacyData memindahkan tugas dan tautan lama ke skema v1 dengan antrean resolusi.
func migrateLegacyData(
	tx *sql.Tx,
	cm ClassMapping,
	classID int64,
	semesterID int64,
	seedActorID int64,
	legacyDBPath string,
	classReport *ClassSeedReport,
) {
	// Buka legacy DB secara read-only murni
	legacyDB, err := sql.Open("sqlite", legacyDBPath)
	if err != nil {
		return
	}
	defer legacyDB.Close()
	_, _ = legacyDB.Exec("PRAGMA query_only = ON;")

	// 1. Ambil seluruh course offering aktif kelas ini untuk pencocokan tugas
	var offerings []OfferingInfo
	rows, err := tx.Query(`
		SELECT co.id, c.code, c.name, co.display_name
		FROM course_offerings co
		JOIN courses c ON co.course_id = c.id
		WHERE co.semester_id = ?;
	`, semesterID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var off OfferingInfo
			if err := rows.Scan(&off.ID, &off.CourseCode, &off.CourseName, &off.DisplayName); err == nil {
				offerings = append(offerings, off)
			}
		}
	}

	// 2. Baca tugas dari legacy database
	legacyClassID := cm.LegacyClassID
	if legacyClassID == "" {
		legacyClassID = cm.ClassCode
	}

	taskRows, err := legacyDB.Query(`
		SELECT id, matkul, deskripsi, deadline, deadline_at, is_done, created_at
		FROM tasks
		WHERE UPPER(class_id) = UPPER(?) OR UPPER(class_id) = UPPER(?);
	`, legacyClassID, cm.ClassCode)
	if err == nil {
		defer taskRows.Close()
		for taskRows.Next() {
			classReport.SourceCounts.Tasks++
			var tID int
			var matkul, deskripsi, deadline string
			var deadlineAt sql.NullTime
			var isDone bool
			var createdAt time.Time

			if err := taskRows.Scan(&tID, &matkul, &deskripsi, &deadline, &deadlineAt, &isDone, &createdAt); err != nil {
				continue
			}

			// Lakukan pencocokan eksak ke offering
			matched := matchOffering(matkul, offerings)

			if len(matched) == 0 {
				classReport.ResolutionQueue = append(classReport.ResolutionQueue, ResolutionQueueItem{
					SourceType:  "TASK",
					SourceID:    strconv.Itoa(tID),
					Subject:     matkul,
					Description: deskripsi,
					Reason:      "mata kuliah tidak ditemukan pada course offerings semester",
					Status:      "PENDING",
				})
				continue
			}

			if len(matched) > 1 {
				classReport.ResolutionQueue = append(classReport.ResolutionQueue, ResolutionQueueItem{
					SourceType:  "TASK",
					SourceID:    strconv.Itoa(tID),
					Subject:     matkul,
					Description: deskripsi,
					Reason:      fmt.Sprintf("ambigu: mata kuliah cocok dengan %d offering", len(matched)),
					Status:      "PENDING",
				})
				continue
			}

			// Tepat 1 offering cocok. Data lama tidak menyimpan tempat pengumpulan,
			// sehingga tugas masuk sebagai DRAFT untuk dilengkapi sebelum diterbitkan.
			targetOffering := matched[0]
			var taskDeadline time.Time
			if deadlineAt.Valid {
				taskDeadline = deadlineAt.Time
			} else {
				taskDeadline = time.Now().Add(24 * time.Hour)
			}

			title := strings.TrimSpace(deskripsi)
			if title == "" {
				title = matkul
			}

			var completedAt sql.NullTime
			if isDone {
				completedAt = sql.NullTime{Time: createdAt, Valid: true}
			}

			// Idempotensi tugas
			var existingTaskID int64
			chkErr := tx.QueryRow(`
				SELECT id FROM tasks
				WHERE course_offering_id = ? AND title = ? AND deadline_at = ?;
			`, targetOffering.ID, title, taskDeadline).Scan(&existingTaskID)
			if chkErr != nil {
				_, _ = tx.Exec(`
					INSERT INTO tasks (
						course_offering_id, title, instructions, deadline_at,
						publication_status, review_state, created_by_user_id,
						version, created_at, completed_at
					)
					VALUES (?, ?, ?, ?, 'DRAFT', 'NOT_REVIEWED', ?, 1, ?, ?);
				`, targetOffering.ID, title, deskripsi, taskDeadline, seedActorID, createdAt, completedAt)
			}
		}
	}

	// 3. Baca tautan dari legacy database (class_links)
	linkRows, err := legacyDB.Query(`
		SELECT id, title, url, category, description, created_at
		FROM class_links
		WHERE UPPER(scope_jid) = UPPER(?) OR UPPER(scope_jid) = UPPER(?);
	`, legacyClassID, cm.ClassCode)
	if err == nil {
		defer linkRows.Close()
		for linkRows.Next() {
			classReport.SourceCounts.Links++
			var lID int
			var title, urlStr, category, desc string
			var createdAt time.Time
			if err := linkRows.Scan(&lID, &title, &urlStr, &category, &desc, &createdAt); err != nil {
				continue
			}

			matType := mapCategoryToMaterialType(category)

			// Idempotensi materi
			var existingMaterialID int64
			chkErr := tx.QueryRow(`
				SELECT id FROM materials
				WHERE class_id = ? AND title = ? AND url = ?;
			`, classID, title, urlStr).Scan(&existingMaterialID)
			if chkErr != nil {
				_, _ = tx.Exec(`
					INSERT INTO materials (
						class_id, title, material_type, url, description,
						visibility, status, created_by_user_id, version, created_at
					)
					VALUES (?, ?, ?, ?, ?, 'CLASS_ACCESS', 'ACTIVE', ?, 1, ?);
				`, classID, title, matType, urlStr, desc, seedActorID, createdAt)
			}
		}
	}
}

// matchOffering mencocokkan teks matkul tugas lama dengan daftar offering semester.
func matchOffering(matkul string, offerings []OfferingInfo) []OfferingInfo {
	clean := strings.ToLower(strings.TrimSpace(matkul))
	var matched []OfferingInfo

	for _, off := range offerings {
		offCode := strings.ToLower(strings.TrimSpace(off.CourseCode))
		offName := strings.ToLower(strings.TrimSpace(off.CourseName))
		offDisp := strings.ToLower(strings.TrimSpace(off.DisplayName))

		if clean == offCode || clean == offName || clean == offDisp {
			matched = append(matched, off)
			continue
		}

		// Pencocokan jika teks lama menyebutkan jenis aktivitas eksplisit
		if strings.Contains(clean, "praktik") && strings.Contains(offDisp, "praktik") && strings.Contains(clean, offName) {
			matched = append(matched, off)
			continue
		}
		if strings.Contains(clean, "teori") && strings.Contains(offDisp, "teori") && strings.Contains(clean, offName) {
			matched = append(matched, off)
			continue
		}
	}

	return matched
}

// cleanSubjectName membersihkan suffix "(Teori)" atau "(Praktikum)" dari nama matkul.
func cleanSubjectName(raw string) string {
	s := strings.TrimSpace(raw)
	re := regexp.MustCompile(`(?i)\s*\((teori|praktikum|praktik)\)\s*$`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

// detectActivityType menentukan apakah mata kuliah adalah PRAKTIKUM atau TEORI.
func detectActivityType(raw string) string {
	lower := strings.ToLower(raw)
	if strings.Contains(lower, "praktikum") || strings.Contains(lower, "praktik") {
		return "PRAKTIKUM"
	}
	return "TEORI"
}

// parseDayOfWeek menerjemahkan nama hari Indonesia ke angka 1-7 (Senin=1 ... Minggu=7).
func parseDayOfWeek(hari string) int {
	clean := strings.ToLower(strings.TrimSpace(hari))
	switch clean {
	case "senin":
		return 1
	case "selasa":
		return 2
	case "rabu":
		return 3
	case "kamis":
		return 4
	case "jumat", "jum'at":
		return 5
	case "sabtu":
		return 6
	case "minggu", "ahad":
		return 7
	default:
		return 0
	}
}

// parseJam memecah string jam "07:00 - 08:40" menjadi "07:00" dan "08:40".
func parseJam(raw string) (string, string, error) {
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("format harus 'HH:MM - HH:MM'")
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	// Format jam harus HH:MM
	matchStart, _ := regexp.MatchString(`^\d{2}:\d{2}$`, start)
	matchEnd, _ := regexp.MatchString(`^\d{2}:\d{2}$`, end)
	if !matchStart || !matchEnd {
		return "", "", fmt.Errorf("jam harus berformat HH:MM")
	}

	return start, end, nil
}

// mapCategoryToMaterialType memetakan kategori lama ke material_type v1.
func mapCategoryToMaterialType(cat string) string {
	clean := strings.ToLower(strings.TrimSpace(cat))
	switch clean {
	case "drive":
		return "DOCUMENT"
	case "meeting", "zoom", "meet":
		return "MEETING"
	case "repo", "github", "gitlab":
		return "REPOSITORY"
	case "portal", "web":
		return "PORTAL"
	default:
		return "OTHER"
	}
}
