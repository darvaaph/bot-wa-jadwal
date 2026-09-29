package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CourseImportItem merepresentasikan mata kuliah dalam batch impor kurikulum
type CourseImportItem struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// LecturerImportItem merepresentasikan dosen dalam batch impor kurikulum
type LecturerImportItem struct {
	Code     string `json:"code"`
	FullName string `json:"full_name"`
}

// OfferingImportItem merepresentasikan offering mata kuliah dalam semester
type OfferingImportItem struct {
	CourseCode    string   `json:"course_code"`
	ActivityType  string   `json:"activity_type"` // e.g. TEORI, PRAKTIKUM
	DisplayName   string   `json:"display_name"`
	LecturerCodes []string `json:"lecturer_codes,omitempty"`
}

// SchedulePatternImportItem merepresentasikan pola jadwal perkuliahan mingguan
type SchedulePatternImportItem struct {
	CourseCode   string  `json:"course_code"`
	ActivityType string  `json:"activity_type"`
	DayOfWeek    int     `json:"day_of_week"` // 1 = Senin s.d 7 = Minggu
	StartTime    string  `json:"start_time"`  // Format HH:MM
	DurationMin  int     `json:"duration_min"`
	RoomCode     *string `json:"room_code,omitempty"`
}

// CurriculumImportPayload adalah struktur data lengkap batch kurikulum yang diimpor
type CurriculumImportPayload struct {
	SourceType       string                      `json:"source_type"` // JSON, SIASAT, EXCEL
	Courses          []CourseImportItem          `json:"courses"`
	Lecturers        []LecturerImportItem        `json:"lecturers"`
	Offerings        []OfferingImportItem        `json:"offerings"`
	SchedulePatterns []SchedulePatternImportItem `json:"schedule_patterns"`
}

// ImportErrorRecord merepresentasikan kesalahan atau peringatan pada proses validasi impor
type ImportErrorRecord struct {
	RowNumber int    `json:"row_number"`
	Field     string `json:"field"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // ERROR, WARNING
}

// ApplyImportRequest payload penerapan batch impor
type ApplyImportRequest struct {
	BatchID *int64 `json:"batch_id,omitempty"`
}

// handleSemesterImportValidate menangani POST /api/v1/semesters/{id}/import-validate
func (s *Server) handleSemesterImportValidate(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang memvalidasi impor kurikulum")
		return
	}

	semIDStr := r.PathValue("id")
	semID, err := strconv.ParseInt(semIDStr, 10, 64)
	if err != nil || semID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID semester tidak valid")
		return
	}

	var classID int64
	err = s.v1DB.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Semester tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi semester")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang memvalidasi impor untuk kelas miliknya")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Gagal membaca body permintaan")
		return
	}

	hasher := sha256.New()
	hasher.Write(bodyBytes)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	var payload CurriculumImportPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid: "+err.Error())
		return
	}

	sourceType := strings.TrimSpace(payload.SourceType)
	if sourceType == "" {
		sourceType = "JSON"
	}

	errorsList := []ImportErrorRecord{}

	// 1. Validasi Mata Kuliah
	courseCodes := make(map[string]bool)
	for i, c := range payload.Courses {
		code := strings.TrimSpace(c.Code)
		name := strings.TrimSpace(c.Name)
		if code == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "courses.code",
				ErrorCode: "REQUIRED",
				Message:   "Kode mata kuliah tidak boleh kosong",
				Severity:  "ERROR",
			})
		} else if courseCodes[code] {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "courses.code",
				ErrorCode: "DUPLICATE",
				Message:   fmt.Sprintf("Kode mata kuliah duplikat dalam berkas impor: %s", code),
				Severity:  "ERROR",
			})
		} else {
			courseCodes[code] = true
		}

		if name == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "courses.name",
				ErrorCode: "REQUIRED",
				Message:   fmt.Sprintf("Nama mata kuliah wajib diisi untuk kode %s", code),
				Severity:  "ERROR",
			})
		}
	}

	// 2. Validasi Dosen
	lecturerCodes := make(map[string]bool)
	for i, l := range payload.Lecturers {
		code := strings.TrimSpace(l.Code)
		name := strings.TrimSpace(l.FullName)
		if code == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "lecturers.code",
				ErrorCode: "REQUIRED",
				Message:   "Kode dosen tidak boleh kosong",
				Severity:  "ERROR",
			})
		} else if lecturerCodes[code] {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "lecturers.code",
				ErrorCode: "DUPLICATE",
				Message:   fmt.Sprintf("Kode dosen duplikat dalam berkas impor: %s", code),
				Severity:  "ERROR",
			})
		} else {
			lecturerCodes[code] = true
		}

		if name == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "lecturers.full_name",
				ErrorCode: "REQUIRED",
				Message:   fmt.Sprintf("Nama lengkap dosen wajib diisi untuk kode %s", code),
				Severity:  "ERROR",
			})
		}
	}

	// 3. Validasi Offering
	offeringKeys := make(map[string]bool)
	for i, off := range payload.Offerings {
		cCode := strings.TrimSpace(off.CourseCode)
		actType := strings.ToUpper(strings.TrimSpace(off.ActivityType))
		if cCode == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "offerings.course_code",
				ErrorCode: "REQUIRED",
				Message:   "Kode mata kuliah pada offering wajib diisi",
				Severity:  "ERROR",
			})
		} else if !courseCodes[cCode] {
			// Periksa apakah sudah ada di database
			var exists int
			_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM courses WHERE code = ?;`, cCode).Scan(&exists)
			if exists == 0 {
				errorsList = append(errorsList, ImportErrorRecord{
					RowNumber: i + 1,
					Field:     "offerings.course_code",
					ErrorCode: "NOT_FOUND",
					Message:   fmt.Sprintf("Mata kuliah %s tidak ditemukan dalam daftar impor maupun database", cCode),
					Severity:  "ERROR",
				})
			}
		}

		if actType == "" {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "offerings.activity_type",
				ErrorCode: "REQUIRED",
				Message:   "Jenis aktivitas offering (misal: TEORI/PRAKTIKUM) wajib diisi",
				Severity:  "ERROR",
			})
		}

		key := fmt.Sprintf("%s:%s", cCode, actType)
		if offeringKeys[key] {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "offerings",
				ErrorCode: "DUPLICATE",
				Message:   fmt.Sprintf("Offering ganda untuk mata kuliah %s dengan tipe %s", cCode, actType),
				Severity:  "ERROR",
			})
		} else {
			offeringKeys[key] = true
		}

		for _, lCode := range off.LecturerCodes {
			lClean := strings.TrimSpace(lCode)
			if !lecturerCodes[lClean] {
				var lExists int
				_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM lecturers WHERE code = ?;`, lClean).Scan(&lExists)
				if lExists == 0 {
					errorsList = append(errorsList, ImportErrorRecord{
						RowNumber: i + 1,
						Field:     "offerings.lecturer_codes",
						ErrorCode: "NOT_FOUND",
						Message:   fmt.Sprintf("Dosen %s pada offering %s tidak ditemukan", lClean, cCode),
						Severity:  "WARNING",
					})
				}
			}
		}
	}

	// 4. Validasi Pola Jadwal (Schedule Patterns)
	for i, pat := range payload.SchedulePatterns {
		cCode := strings.TrimSpace(pat.CourseCode)
		actType := strings.ToUpper(strings.TrimSpace(pat.ActivityType))
		key := fmt.Sprintf("%s:%s", cCode, actType)
		if !offeringKeys[key] {
			// Periksa apakah offering sudah ada di DB
			var offExists int
			_ = s.v1DB.QueryRow(`
				SELECT COUNT(*)
				FROM course_offerings co
				JOIN courses c ON co.course_id = c.id
				WHERE co.semester_id = ? AND c.code = ? AND co.activity_type = ?;
			`, semID, cCode, actType).Scan(&offExists)
			if offExists == 0 {
				errorsList = append(errorsList, ImportErrorRecord{
					RowNumber: i + 1,
					Field:     "schedule_patterns",
					ErrorCode: "OFFERING_NOT_FOUND",
					Message:   fmt.Sprintf("Pola jadwal merujuk offering yang tidak terdaftar: %s (%s)", cCode, actType),
					Severity:  "ERROR",
				})
			}
		}

		if pat.DayOfWeek < 1 || pat.DayOfWeek > 7 {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "schedule_patterns.day_of_week",
				ErrorCode: "RANGE_ERROR",
				Message:   "day_of_week harus berada di antara 1 (Senin) hingga 7 (Minggu)",
				Severity:  "ERROR",
			})
		}

		if pat.DurationMin <= 0 {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "schedule_patterns.duration_min",
				ErrorCode: "RANGE_ERROR",
				Message:   "duration_min harus lebih besar dari 0 menit",
				Severity:  "ERROR",
			})
		}

		cleanStartTime := strings.TrimSpace(pat.StartTime)
		_, errTime := time.Parse("15:04", cleanStartTime)
		if errTime != nil {
			_, errTime = time.Parse("15:04:05", cleanStartTime)
		}
		if errTime != nil {
			errorsList = append(errorsList, ImportErrorRecord{
				RowNumber: i + 1,
				Field:     "schedule_patterns.start_time",
				ErrorCode: "FORMAT_ERROR",
				Message:   fmt.Sprintf("Format start_time tidak valid (harus HH:MM): %s", cleanStartTime),
				Severity:  "ERROR",
			})
		}

		if pat.RoomCode != nil && strings.TrimSpace(*pat.RoomCode) != "" {
			rCode := strings.TrimSpace(*pat.RoomCode)
			var roomExists int
			_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM rooms WHERE code = ?;`, rCode).Scan(&roomExists)
			if roomExists == 0 {
				errorsList = append(errorsList, ImportErrorRecord{
					RowNumber: i + 1,
					Field:     "schedule_patterns.room_code",
					ErrorCode: "ROOM_NOT_FOUND",
					Message:   fmt.Sprintf("Ruangan %s belum terdaftar dalam master ruangan", rCode),
					Severity:  "WARNING",
				})
			}
		}
	}

	// Tentukan Status Batch Impor
	hasError := false
	for _, e := range errorsList {
		if e.Severity == "ERROR" {
			hasError = true
			break
		}
	}

	batchStatus := "READY"
	if hasError {
		batchStatus = "INVALID"
	}

	summaryMap := map[string]any{
		"source_type":      sourceType,
		"courses_count":    len(payload.Courses),
		"lecturers_count":  len(payload.Lecturers),
		"offerings_count":  len(payload.Offerings),
		"patterns_count":   len(payload.SchedulePatterns),
		"errors_count":     len(errorsList),
		"has_fatal_errors": hasError,
		"payload":          payload,
	}

	summaryJSONBytes, _ := json.Marshal(summaryMap)

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi validasi impor")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO import_batches (
			class_id, semester_id, source_type, source_checksum, status, created_by_user_id, summary_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);
	`, classID, semID, sourceType, checksum, batchStatus, u.UserID, string(summaryJSONBytes))
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan rekam batch impor: %v", err))
		return
	}

	batchID, _ := res.LastInsertId()

	for _, e := range errorsList {
		_, _ = tx.Exec(`
			INSERT INTO import_errors (batch_id, row_number, field, error_code, message, severity)
			VALUES (?, ?, ?, ?, ?, ?);
		`, batchID, e.RowNumber, e.Field, e.ErrorCode, e.Message, e.Severity)
	}

	_ = tx.Commit()

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"batch_id":         batchID,
		"status":           batchStatus,
		"checksum":         checksum,
		"courses_count":    len(payload.Courses),
		"lecturers_count":  len(payload.Lecturers),
		"offerings_count":  len(payload.Offerings),
		"patterns_count":   len(payload.SchedulePatterns),
		"errors":           errorsList,
		"has_fatal_errors": hasError,
	})
}

// handleSemesterImportApply menangani POST /api/v1/semesters/{id}/import-apply
func (s *Server) handleSemesterImportApply(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang menerapkan impor kurikulum")
		return
	}

	semIDStr := r.PathValue("id")
	semID, err := strconv.ParseInt(semIDStr, 10, 64)
	if err != nil || semID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID semester tidak valid")
		return
	}

	var classID int64
	err = s.v1DB.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Semester tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi semester")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang menerapkan impor untuk kelas miliknya")
		return
	}

	var req ApplyImportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	var (
		batchID     int64
		summaryJSON string
		batchStatus string
	)

	if req.BatchID != nil && *req.BatchID > 0 {
		err = s.v1DB.QueryRow(`
			SELECT id, status, summary_json
			FROM import_batches
			WHERE id = ? AND semester_id = ?;
		`, *req.BatchID, semID).Scan(&batchID, &batchStatus, &summaryJSON)
	} else {
		err = s.v1DB.QueryRow(`
			SELECT id, status, summary_json
			FROM import_batches
			WHERE semester_id = ? AND status = 'READY'
			ORDER BY id DESC LIMIT 1;
		`, semID).Scan(&batchID, &batchStatus, &summaryJSON)
	}

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Batch impor dengan status READY tidak ditemukan untuk semester ini")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat batch impor")
		return
	}

	if batchStatus != "READY" {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, fmt.Sprintf("Batch impor tidak dapat diterapkan karena statusnya %s (harus READY)", batchStatus))
		return
	}

	// Parse payload yang tersimpan di summary_json
	var summaryData struct {
		Payload CurriculumImportPayload `json:"payload"`
	}
	if err := json.Unmarshal([]byte(summaryJSON), &summaryData); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DATA_CORRUPT", "Gagal membaca berkas payload tersimpan")
		return
	}

	payload := summaryData.Payload

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi impor")
		return
	}
	defer tx.Rollback()

	// 1. Terapkan Master Mata Kuliah
	courseIDMap := make(map[string]int64)
	for _, c := range payload.Courses {
		code := strings.TrimSpace(c.Code)
		name := strings.TrimSpace(c.Name)
		var cid int64
		err := tx.QueryRow(`
			INSERT INTO courses (code, name, status)
			VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, status = 'ACTIVE'
			RETURNING id;
		`, code, name).Scan(&cid)
		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan mata kuliah %s: %v", code, err))
			return
		}
		courseIDMap[code] = cid
	}

	// 2. Terapkan Master Dosen
	lecturerIDMap := make(map[string]int64)
	for _, l := range payload.Lecturers {
		code := strings.TrimSpace(l.Code)
		name := strings.TrimSpace(l.FullName)
		var lid int64
		err := tx.QueryRow(`
			INSERT INTO lecturers (code, full_name, status)
			VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, status = 'ACTIVE'
			RETURNING id;
		`, code, name).Scan(&lid)
		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan dosen %s: %v", code, err))
			return
		}
		lecturerIDMap[code] = lid
	}

	// 3. Terapkan Course Offerings
	offeringIDMap := make(map[string]int64)
	for _, off := range payload.Offerings {
		cCode := strings.TrimSpace(off.CourseCode)
		actType := strings.ToUpper(strings.TrimSpace(off.ActivityType))
		displayName := strings.TrimSpace(off.DisplayName)

		cid, exists := courseIDMap[cCode]
		if !exists {
			_ = tx.QueryRow(`SELECT id FROM courses WHERE code = ?;`, cCode).Scan(&cid)
		}
		if cid == 0 {
			continue
		}

		if displayName == "" {
			_ = tx.QueryRow(`SELECT name FROM courses WHERE id = ?;`, cid).Scan(&displayName)
		}

		var offID int64
		err := tx.QueryRow(`
			INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, 'ACTIVE')
			ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET display_name = excluded.display_name, status = 'ACTIVE'
			RETURNING id;
		`, semID, cid, actType, displayName).Scan(&offID)

		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan offering %s (%s): %v", cCode, actType, err))
			return
		}

		key := fmt.Sprintf("%s:%s", cCode, actType)
		offeringIDMap[key] = offID

		// Tautkan dosen ke offering
		for _, lCode := range off.LecturerCodes {
			lClean := strings.TrimSpace(lCode)
			lid, lExists := lecturerIDMap[lClean]
			if !lExists {
				_ = tx.QueryRow(`SELECT id FROM lecturers WHERE code = ?;`, lClean).Scan(&lid)
			}
			if lid > 0 {
				_, _ = tx.Exec(`
					INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
					VALUES (?, ?, 'PRIMARY')
					ON CONFLICT(course_offering_id, lecturer_id) DO NOTHING;
				`, offID, lid)
			}
		}
	}

	// 4. Terapkan Pola Jadwal (Schedule Patterns)
	patternsImported := 0
	for _, pat := range payload.SchedulePatterns {
		cCode := strings.TrimSpace(pat.CourseCode)
		actType := strings.ToUpper(strings.TrimSpace(pat.ActivityType))
		key := fmt.Sprintf("%s:%s", cCode, actType)

		offID, exists := offeringIDMap[key]
		if !exists {
			_ = tx.QueryRow(`
				SELECT co.id
				FROM course_offerings co
				JOIN courses c ON co.course_id = c.id
				WHERE co.semester_id = ? AND c.code = ? AND co.activity_type = ?;
			`, semID, cCode, actType).Scan(&offID)
		}
		if offID == 0 {
			continue
		}

		// Hitung waktu selesai
		cleanStart := strings.TrimSpace(pat.StartTime)
		parsedTime, errTime := time.Parse("15:04", cleanStart)
		if errTime != nil {
			parsedTime, _ = time.Parse("15:04:05", cleanStart)
		}
		endTime := parsedTime.Add(time.Duration(pat.DurationMin) * time.Minute).Format("15:04")

		var roomID sql.NullInt64
		if pat.RoomCode != nil && strings.TrimSpace(*pat.RoomCode) != "" {
			var rid int64
			err := tx.QueryRow(`SELECT id FROM rooms WHERE code = ?;`, strings.TrimSpace(*pat.RoomCode)).Scan(&rid)
			if err == nil && rid > 0 {
				roomID = sql.NullInt64{Int64: rid, Valid: true}
			}
		}

		_, err := tx.Exec(`
			INSERT INTO schedule_patterns (
				course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status
			) VALUES (?, ?, ?, ?, ?, (SELECT starts_on FROM semesters WHERE id = ?), 'ACTIVE');
		`, offID, roomID, pat.DayOfWeek, cleanStart, endTime, semID)

		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mengimpor pola jadwal: %v", err))
			return
		}
		patternsImported++
	}

	// 5. Perbarui Status Batch Impor menjadi APPLIED
	if _, err := tx.Exec(`UPDATE import_batches SET status = 'APPLIED' WHERE id = ?;`, batchID); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status batch impor")
		return
	}

	// 6. Catat audit_logs
	if _, err := tx.Exec(`
		INSERT INTO audit_logs (actor_type, class_id, semester_id, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, after_json, correlation_id)
		VALUES ('USER', ?, ?, ?, ?, 'APPLY_CURRICULUM_IMPORT', 'IMPORT_BATCH', ?, ?, ?);
	`, classID, semID, u.UserID, u.ActiveAssignmentID, batchID,
		fmt.Sprintf(`{"batch_id":%d,"courses":%d,"lecturers":%d,"offerings":%d,"patterns":%d}`,
			batchID, len(payload.Courses), len(payload.Lecturers), len(payload.Offerings), patternsImported),
		fmt.Sprintf("apply-import-%d-%d", batchID, time.Now().UnixNano()),
	); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit impor")
		return
	}

	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal meresmikan penerapan transaksi impor kurikulum")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"batch_id":           batchID,
		"status":             "APPLIED",
		"courses_imported":   len(payload.Courses),
		"lecturers_imported": len(payload.Lecturers),
		"offerings_imported": len(payload.Offerings),
		"patterns_imported":  patternsImported,
		"message":            "Kurikulum dan pola perkuliahan berhasil diimpor dan siap digunakan",
	})
}
