package v1

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/audit"
)

// CreateSemesterRequest adalah payload pembuatan semester baru
type CreateSemesterRequest struct {
	AcademicYear     string `json:"academic_year"`
	Term             string `json:"term"`
	StartsOn         string `json:"starts_on"`
	EndsOn           string `json:"ends_on"`
	SourceSemesterID *int64 `json:"source_semester_id,omitempty"`
}

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

// CreateMaterialRequest adalah payload pembuatan materi baru
type CreateMaterialRequest struct {
	ClassSlug    string  `json:"class_slug"`
	OfferingID   *int64  `json:"offering_id,omitempty"`
	TaskID       *int64  `json:"task_id,omitempty"`
	Title        string  `json:"title"`
	MaterialType string  `json:"material_type"` // DOCUMENT, MEETING, REPOSITORY, PORTAL, OTHER
	URL          *string `json:"url,omitempty"`
	Description  *string `json:"description,omitempty"`
}

// RoomCandidateItem merepresentasikan ruangan yang tersedia untuk digunakan
type RoomCandidateItem struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	RoomType string `json:"room_type"`
}

// RoomConfirmationRequest payload pengajuan/konfirmasi ruangan ke pihak TU
type RoomConfirmationRequest struct {
	RoomID             int64   `json:"room_id"`
	ConfirmationStatus string  `json:"confirmation_status"` // PENDING, CONFIRMED, REJECTED
	ExternalContact    *string `json:"external_contact,omitempty"`
	Note               *string `json:"note,omitempty"`
}

// AcademicController mengelola semester, kurikulum import, materi kuliah, dan alokasi ruangan
type AcademicController struct {
	db *sql.DB
}

// NewAcademicController membuat instance baru AcademicController
func NewAcademicController(db *sql.DB) *AcademicController {
	return &AcademicController{db: db}
}

// GetClassSemesters menangani GET /api/v1/classes/{slug}/semesters
func (c *AcademicController) GetClassSemesters(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	u, _ := common.GetAuthContext(r)
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kelas ditolak")
		return
	}

	rows, err := c.db.Query(`
		SELECT id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, version
		FROM semesters
		WHERE class_id = ?
		ORDER BY starts_on DESC;
	`, classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat semester")
		return
	}
	defer rows.Close()

	var semesters []map[string]any
	for rows.Next() {
		var id int64
		var year, term, startsOn, endsOn, status string
		var publishedAt, activatedAt common.DBTimestamp
		var version int

		if err := rows.Scan(&id, &year, &term, &startsOn, &endsOn, &status, &publishedAt, &activatedAt, &version); err == nil {
			semesters = append(semesters, map[string]any{
				"id":            id,
				"academic_year": year,
				"term":          term,
				"starts_on":     startsOn,
				"ends_on":       endsOn,
				"status":        status,
				"published_at":  publishedAt.RFC3339(),
				"activated_at":  activatedAt.RFC3339(),
				"version":       version,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, semesters)
}

// CreateClassSemester menangani POST /api/v1/classes/{slug}/semesters
func (c *AcademicController) CreateClassSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang membuat semester")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang membuat semester untuk kelas penugasannya")
		return
	}

	var req CreateSemesterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	if strings.TrimSpace(req.AcademicYear) == "" || strings.TrimSpace(req.Term) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "academic_year dan term wajib diisi")
		return
	}

	startDate, errStart := time.Parse("2006-01-02", req.StartsOn)
	endDate, errEnd := time.Parse("2006-01-02", req.EndsOn)
	if errStart != nil || errEnd != nil || !endDate.After(startDate) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format tanggal harus YYYY-MM-DD dan ends_on harus setelah starts_on")
		return
	}

	var semID int64
	svc := academic.NewSemesterService(c.db)
	semID, err = svc.CreateDraft(r.Context(), academic.Actor{
		UserID:           u.UserID,
		RoleAssignmentID: u.ActiveAssignmentID,
	}, classID, academic.DraftInput{
		AcademicYear:     req.AcademicYear,
		Term:             req.Term,
		StartsOn:         req.StartsOn,
		EndsOn:           req.EndsOn,
		SourceSemesterID: req.SourceSemesterID,
	})
	if err != nil {
		switch {
		case errors.Is(err, academic.ErrInvalidInput):
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format tanggal harus YYYY-MM-DD, ends_on harus setelah starts_on, dan semester sumber harus sekelas")
		case errors.Is(err, academic.ErrNotFound):
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester sumber tidak ditemukan")
		default:
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan semester: %v", err))
		}
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":     semID,
		"status": "DRAFT",
	})
}

// ActivateSemester menangani POST /api/v1/classes/{slug}/semesters/{id}/activate
func (c *AcademicController) ActivateSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang mengaktifkan semester")
		return
	}
	var confirm struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&confirm); err != nil || !confirm.Confirm {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "confirm:true wajib disertakan")
		return
	}

	semIDStr := r.PathValue("id")
	semID, _ := strconv.ParseInt(semIDStr, 10, 64)

	slug := r.PathValue("slug")
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang mengaktifkan semester kelas penugasannya")
		return
	}

	var curStatus string
	if err := c.db.QueryRow(`SELECT status FROM semesters WHERE id = ? AND class_id = ?;`, semID, classID).Scan(&curStatus); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	}
	if curStatus != "DRAFT" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Hanya semester DRAFT yang dapat diaktifkan")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi aktivasi")
		return
	}
	defer tx.Rollback()

	// 1. Arsipkan semester aktif lama
	if _, err = tx.Exec(`
		UPDATE semesters
		SET status = 'ARCHIVED', archived_at = CURRENT_TIMESTAMP
		WHERE class_id = ? AND status = 'ACTIVE';
	`, classID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengarsipkan semester aktif")
		return
	}

	// 2. Aktifkan semester baru
	res, err := tx.Exec(`
		UPDATE semesters
		SET status = 'ACTIVE', published_at = CURRENT_TIMESTAMP, activated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND class_id = ? AND status = 'DRAFT';
	`, semID, classID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengaktifkan semester")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	}

	correlationID := fmt.Sprintf("activate-semester-%d-%d", semID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			SemesterID:    &semID,
			Action:        "ACTIVATE_SEMESTER",
			EntityType:    "SEMESTER",
			EntityID:      &semID,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit aktivasi semester")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit aktivasi semester")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"semester_id": semID,
		"status":      "ACTIVE",
	})
}

// GetSemesterOfferings menangani GET /api/v1/semesters/{id}/offerings
func (c *AcademicController) GetSemesterOfferings(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	semIDStr := r.PathValue("id")
	semID, _ := strconv.ParseInt(semIDStr, 10, 64)
	u, _ := common.GetAuthContext(r)
	var classID int64
	if err := c.db.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses semester ditolak")
		return
	}

	rows, err := c.db.Query(`
		SELECT co.id, c.code, co.display_name, co.activity_type
		FROM course_offerings co
		JOIN courses c ON co.course_id = c.id
		WHERE co.semester_id = ? AND co.status = 'ACTIVE'
		ORDER BY co.display_name;
	`, semID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat mata kuliah offering")
		return
	}
	defer rows.Close()

	var offerings []map[string]any
	for rows.Next() {
		var id int64
		var code, displayName, actType string
		if err := rows.Scan(&id, &code, &displayName, &actType); err == nil {
			lecturers := GetOfferingLecturers(c.db, id)
			offerings = append(offerings, map[string]any{
				"id":            id,
				"course_code":   code,
				"display_name":  displayName,
				"activity_type": actType,
				"lecturers":     lecturers,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, offerings)
}

// PreviewSemester menangani GET /api/v1/classes/{slug}/semesters/{id}/preview
func (c *AcademicController) PreviewSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang meninjau semester")
		return
	}
	slug := r.PathValue("slug")
	semID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if semID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID semester tidak valid")
		return
	}
	var classID int64
	if err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang meninjau semester kelas penugasannya")
		return
	}
	svc := academic.NewSemesterService(c.db)
	p, err := svc.Preview(r.Context(), classID, semID)
	if err != nil {
		if errors.Is(err, academic.ErrNotFound) {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
			return
		}
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal meninjau semester")
		return
	}
	common.WriteV1Success(w, http.StatusOK, p)
}

// DeleteDraftSemester menangani DELETE /api/v1/classes/{slug}/semesters/{id}
func (c *AcademicController) DeleteDraftSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang menghapus draf semester")
		return
	}
	slug := r.PathValue("slug")
	semID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if semID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID semester tidak valid")
		return
	}
	var classID int64
	if err := c.db.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang menghapus draf semester kelas penugasannya")
		return
	}
	svc := academic.NewSemesterService(c.db)
	err := svc.DeleteDraft(r.Context(), academic.Actor{
		UserID:           u.UserID,
		RoleAssignmentID: u.ActiveAssignmentID,
	}, classID, semID)
	if err != nil {
		switch {
		case errors.Is(err, academic.ErrNotFound):
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester draf tidak ditemukan")
		case errors.Is(err, academic.ErrInvalidState):
			common.WriteV1Error(w, http.StatusConflict, common.CodeValidation, "Hanya semester berstatus DRAFT yang dapat dihapus")
		case errors.Is(err, academic.ErrInvalidInput):
			common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Semester tidak sesuai dengan kelas yang dipilih")
		default:
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menghapus semester draf: %v", err))
		}
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"deleted": true,
		"id":      semID,
	})
}

// CreateSemesterOfferingRequest adalah payload tambah offering manual ke semester DRAFT
type CreateSemesterOfferingRequest struct {
	CourseCode    string   `json:"course_code"`
	ActivityType  string   `json:"activity_type"`
	DisplayName   string   `json:"display_name"`
	LecturerCodes []string `json:"lecturer_codes,omitempty"`
}

// CreateSemesterOffering menangani POST /api/v1/semesters/{id}/offerings
func (c *AcademicController) CreateSemesterOffering(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang menambah mata kuliah")
		return
	}
	semID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if semID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID semester tidak valid")
		return
	}
	var status string
	var classID int64
	if err := c.db.QueryRow(`SELECT class_id, status FROM semesters WHERE id = ?;`, semID).Scan(&classID, &status); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca semester")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang menambah mata kuliah kelas penugasannya")
		return
	}
	if status != "DRAFT" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Hanya semester DRAFT yang dapat ditambah mata kuliah")
		return
	}
	var req CreateSemesterOfferingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	code := strings.TrimSpace(req.CourseCode)
	actType := strings.ToUpper(strings.TrimSpace(req.ActivityType))
	display := strings.TrimSpace(req.DisplayName)
	if code == "" || display == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "course_code dan display_name wajib diisi")
		return
	}
	if actType != "TEORI" && actType != "PRAKTIKUM" && actType != "PRAKTIK" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "activity_type harus TEORI/PRAKTIKUM/PRAKTIK")
		return
	}
	var courseID int64
	if err := c.db.QueryRow(`SELECT id FROM courses WHERE code = ?;`, code).Scan(&courseID); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "course_code tidak dikenal di master mata kuliah")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()
	var offeringID int64
	if err := tx.QueryRow(`
		INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
		VALUES (?, ?, ?, ?, 'ACTIVE') RETURNING id;
	`, semID, courseID, actType, display).Scan(&offeringID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menambah mata kuliah")
		return
	}
	for _, lc := range req.LecturerCodes {
		lc = strings.TrimSpace(lc)
		if lc == "" {
			continue
		}
		var lectID int64
		if err := tx.QueryRow(`SELECT id FROM lecturers WHERE code = ?;`, lc).Scan(&lectID); err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, fmt.Sprintf("Kode dosen tidak dikenal: %s", lc))
			return
		}
		if _, err := tx.Exec(`INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility) VALUES (?, ?, 'PRIMARY') ON CONFLICT(course_offering_id, lecturer_id) DO UPDATE SET superseded_at=NULL;`, offeringID, lectID); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menautkan dosen")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan mata kuliah")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": offeringID, "status": "ACTIVE"})
}

// SemesterImportValidate menangani POST /api/v1/semesters/{id}/import-validate
func (c *AcademicController) SemesterImportValidate(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang memvalidasi impor kurikulum")
		return
	}

	semIDStr := r.PathValue("id")
	semID, err := strconv.ParseInt(semIDStr, 10, 64)
	if err != nil || semID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID semester tidak valid")
		return
	}

	var classID int64
	err = c.db.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi semester")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang memvalidasi impor untuk kelas miliknya")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca body permintaan")
		return
	}

	hasher := sha256.New()
	hasher.Write(bodyBytes)
	checksum := hex.EncodeToString(hasher.Sum(nil))

	var payload CurriculumImportPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid: "+err.Error())
		return
	}

	sourceType := strings.TrimSpace(payload.SourceType)
	if sourceType == "" {
		sourceType = "JSON"
	}

	errorsList := []ImportErrorRecord{}

	// 1. Validasi Mata Kuliah
	courseCodes := make(map[string]bool)
	for i, course := range payload.Courses {
		code := strings.TrimSpace(course.Code)
		name := strings.TrimSpace(course.Name)
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
			var exists int
			_ = c.db.QueryRow(`SELECT COUNT(*) FROM courses WHERE code = ?;`, cCode).Scan(&exists)
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
				_ = c.db.QueryRow(`SELECT COUNT(*) FROM lecturers WHERE code = ?;`, lClean).Scan(&lExists)
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
			var offExists int
			_ = c.db.QueryRow(`
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
			_ = c.db.QueryRow(`SELECT COUNT(*) FROM rooms WHERE code = ?;`, rCode).Scan(&roomExists)
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

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi validasi impor")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO import_batches (
			class_id, semester_id, source_type, source_checksum, status, created_by_user_id, summary_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);
	`, classID, semID, sourceType, checksum, batchStatus, u.UserID, string(summaryJSONBytes))
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan rekam batch impor: %v", err))
		return
	}

	batchID, _ := res.LastInsertId()

	useNewImportErrCols := hasImportErrNewCols(tx)
	for _, e := range errorsList {
		if useNewImportErrCols {
			_, _ = tx.Exec(`
				INSERT INTO import_errors (batch_id, source_location, field_name, error_code, message, severity)
				VALUES (?, ?, ?, ?, ?, ?);
			`, batchID, fmt.Sprintf("row %d", e.RowNumber), e.Field, e.ErrorCode, e.Message, e.Severity)
		} else {
			_, _ = tx.Exec(`
				INSERT INTO import_errors (batch_id, row_number, field, error_code, message, severity)
				VALUES (?, ?, ?, ?, ?, ?);
			`, batchID, e.RowNumber, e.Field, e.ErrorCode, e.Message, e.Severity)
		}
	}

	_ = tx.Commit()

	common.WriteV1Success(w, http.StatusOK, map[string]any{
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

// SemesterImportApply menangani POST /api/v1/semesters/{id}/import-apply
func (c *AcademicController) SemesterImportApply(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang menerapkan kurikulum")
		return
	}

	semIDStr := r.PathValue("id")
	semID, err := strconv.ParseInt(semIDStr, 10, 64)
	if err != nil || semID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID semester tidak valid")
		return
	}

	var classID int64
	err = c.db.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi semester")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang menerapkan impor untuk kelas miliknya")
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
		err = c.db.QueryRow(`
			SELECT id, status, summary_json
			FROM import_batches
			WHERE id = ? AND semester_id = ?;
		`, *req.BatchID, semID).Scan(&batchID, &batchStatus, &summaryJSON)
	} else {
		err = c.db.QueryRow(`
			SELECT id, status, summary_json
			FROM import_batches
			WHERE semester_id = ? AND status = 'READY'
			ORDER BY id DESC LIMIT 1;
		`, semID).Scan(&batchID, &batchStatus, &summaryJSON)
	}

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Batch impor dengan status READY tidak ditemukan untuk semester ini")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat batch impor")
		return
	}

	if batchStatus != "READY" {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, fmt.Sprintf("Batch impor tidak dapat diterapkan karena statusnya %s (harus READY)", batchStatus))
		return
	}

	// Parse payload yang tersimpan di summary_json
	var summaryData struct {
		Payload CurriculumImportPayload `json:"payload"`
	}
	if err := json.Unmarshal([]byte(summaryJSON), &summaryData); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DATA_CORRUPT", "Gagal membaca berkas payload tersimpan")
		return
	}

	payload := summaryData.Payload

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi impor")
		return
	}
	defer tx.Rollback()

	// 1. Terapkan Master Mata Kuliah
	courseIDMap := make(map[string]int64)
	for _, course := range payload.Courses {
		code := strings.TrimSpace(course.Code)
		name := strings.TrimSpace(course.Name)
		var cid int64
		err := tx.QueryRow(`
			INSERT INTO courses (code, name, status)
			VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET name = excluded.name, status = 'ACTIVE'
			RETURNING id;
		`, code, name).Scan(&cid)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan mata kuliah %s: %v", code, err))
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
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan dosen %s: %v", code, err))
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
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan offering %s (%s): %v", cCode, actType, err))
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
					ON CONFLICT(course_offering_id, lecturer_id) DO UPDATE SET responsibility='PRIMARY', superseded_at=NULL;
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
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mengimpor pola jadwal: %v", err))
			return
		}
		patternsImported++
	}

	// 5. Perbarui Status Batch Impor menjadi APPLIED
	if _, err := tx.Exec(`UPDATE import_batches SET status = 'APPLIED' WHERE id = ?;`, batchID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status batch impor")
		return
	}

	// 6. Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"batch_id":%d,"courses":%d,"lecturers":%d,"offerings":%d,"patterns":%d}`,
			batchID, len(payload.Courses), len(payload.Lecturers), len(payload.Offerings), patternsImported)
		correlationID := fmt.Sprintf("apply-import-%d-%d", batchID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			SemesterID:    &semID,
			Action:        "APPLY_CURRICULUM_IMPORT",
			EntityType:    "IMPORT_BATCH",
			EntityID:      &batchID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit impor")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal meresmikan penerapan transaksi impor kurikulum")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"batch_id":           batchID,
		"status":             "APPLIED",
		"courses_imported":   len(payload.Courses),
		"lecturers_imported": len(payload.Lecturers),
		"offerings_imported": len(payload.Offerings),
		"patterns_imported":  patternsImported,
		"message":            "Kurikulum dan pola perkuliahan berhasil diimpor dan siap digunakan",
	})
}

// GetMaterials menangani GET /api/v1/materials
func (c *AcademicController) GetMaterials(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	classSlug := r.URL.Query().Get("class_slug")
	offeringParam := r.URL.Query().Get("offering_id")

	query := `
		SELECT m.id, m.class_id, c.slug, m.course_offering_id, m.task_id, m.title,
		       m.material_type, COALESCE(m.url, ''), COALESCE(m.description, ''),
		       COALESCE(m.created_at, ''), m.version
		FROM materials m
		JOIN classes c ON m.class_id = c.id
		WHERE m.status = 'ACTIVE' AND m.deleted_at IS NULL
	`
	var args []any

	if classSlug != "" {
		query += " AND c.slug = ?"
		args = append(args, classSlug)
	}
	if offeringParam != "" {
		offID, _ := strconv.ParseInt(offeringParam, 10, 64)
		if offID > 0 {
			query += " AND m.course_offering_id = ?"
			args = append(args, offID)
		}
	}

	query += " ORDER BY m.created_at DESC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat materi")
		return
	}
	defer rows.Close()

	var materials []map[string]any
	for rows.Next() {
		var id, classID int64
		var slug, title, matType, urlStr, desc, created string
		var version int
		var offID, taskID sql.NullInt64

		if err := rows.Scan(&id, &classID, &slug, &offID, &taskID, &title, &matType, &urlStr, &desc, &created, &version); err == nil {
			materials = append(materials, map[string]any{
				"id":         id,
				"class_slug": slug,
				"offering_id": func() any {
					if offID.Valid {
						return offID.Int64
					}
					return nil
				}(),
				"task_id": func() any {
					if taskID.Valid {
						return taskID.Int64
					}
					return nil
				}(),
				"title":         title,
				"material_type": matType,
				"url":           urlStr,
				"description":   desc,
				"created_at":    created,
				"version":       version,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, materials)
}

// CreateMaterial menangani POST /api/v1/materials
func (c *AcademicController) CreateMaterial(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.ClassSlug) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "title dan class_slug wajib diisi")
		return
	}

	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, req.ClassSlug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang menambahkan materi untuk kelas penugasannya")
		return
	}

	// Batasan peran: PJ hanya boleh menambah materi untuk offering penugasannya
	if u.ActiveRole == "PJ" {
		if req.OfferingID == nil || !u.ActiveCourseOfferingID.Valid || *req.OfferingID != u.ActiveCourseOfferingID.Int64 {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya berwenang menambahkan materi untuk offering penugasannya")
			return
		}
	}
	if req.OfferingID != nil {
		var offeringClassID int64
		if err := c.db.QueryRow(`
			SELECT sem.class_id FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?;
		`, *req.OfferingID).Scan(&offeringClassID); err != nil || offeringClassID != classID {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "offering_id tidak berada pada class_slug yang dipilih")
			return
		}
	}
	if req.TaskID != nil {
		var taskClassID, taskOfferingID int64
		if err := c.db.QueryRow(`
			SELECT sem.class_id, t.course_offering_id FROM tasks t
			JOIN course_offerings co ON co.id = t.course_offering_id
			JOIN semesters sem ON sem.id = co.semester_id WHERE t.id = ?;
		`, *req.TaskID).Scan(&taskClassID, &taskOfferingID); err != nil || taskClassID != classID || (req.OfferingID != nil && taskOfferingID != *req.OfferingID) {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "task_id tidak konsisten dengan kelas atau offering")
			return
		}
	}

	matType := strings.ToUpper(strings.TrimSpace(req.MaterialType))
	if matType == "" {
		matType = "OTHER"
	}

	urlStr := ""
	if req.URL != nil {
		urlStr = strings.TrimSpace(*req.URL)
	}
	descStr := ""
	if req.Description != nil {
		descStr = *req.Description
	}
	var descArg any
	if descStr != "" {
		descArg = descStr
	}

	var matID int64
	err = c.db.QueryRow(`
		INSERT INTO materials (
			class_id, course_offering_id, task_id, title, material_type,
			url, description, visibility, status, version, created_by_user_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'CLASS_ACCESS', 'ACTIVE', 1, ?)
		RETURNING id;
	`, classID, req.OfferingID, req.TaskID, req.Title, matType, urlStr, descArg, u.UserID).Scan(&matID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan materi: %v", err))
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":     matID,
		"status": "ACTIVE",
	})
}

// PatchMaterialRequest adalah payload ubah materi
type PatchMaterialRequest struct {
	Version      int     `json:"version"`
	Title        *string `json:"title,omitempty"`
	MaterialType *string `json:"material_type,omitempty"`
	URL          *string `json:"url,omitempty"`
	Description  *string `json:"description,omitempty"`
}

// PatchMaterial menangani PATCH /api/v1/materials/{id}
func (c *AcademicController) PatchMaterial(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "PJ" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM, PJ, atau System Admin yang berwenang mengubah materi")
		return
	}
	matID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if matID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID materi tidak valid")
		return
	}
	var req PatchMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	if req.Version <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib diisi")
		return
	}
	var classID, offeringID sql.NullInt64
	var curVersion int
	var curStatus string
	if err := c.db.QueryRow(`SELECT class_id, course_offering_id, version, status FROM materials WHERE id = ? AND deleted_at IS NULL;`, matID).
		Scan(&classID, &offeringID, &curVersion, &curStatus); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Materi tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca materi")
		return
	}
	if curStatus != "ACTIVE" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Hanya materi aktif yang dapat diubah")
		return
	}
	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{"current_version": curVersion})
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || !classID.Valid || u.ActiveClassID.Int64 != classID.Int64) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang mengubah materi kelas penugasannya")
		return
	}
	if u.ActiveRole == "PJ" {
		if !offeringID.Valid || !u.ActiveCourseOfferingID.Valid || offeringID.Int64 != u.ActiveCourseOfferingID.Int64 {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya berwenang mengubah materi offering penugasannya")
			return
		}
	}
	sets := []string{}
	args := []any{}
	if req.Title != nil {
		if strings.TrimSpace(*req.Title) == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "title tidak boleh kosong")
			return
		}
		sets = append(sets, "title = ?")
		args = append(args, strings.TrimSpace(*req.Title))
	}
	if req.MaterialType != nil {
		matType := strings.ToUpper(strings.TrimSpace(*req.MaterialType))
		switch matType {
		case "DOCUMENT", "MEETING", "REPOSITORY", "PORTAL", "OTHER":
		default:
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "material_type harus DOCUMENT/MEETING/REPOSITORY/PORTAL/OTHER")
			return
		}
		sets = append(sets, "material_type = ?")
		args = append(args, matType)
	}
	if req.URL != nil {
		sets = append(sets, "url = ?")
		args = append(args, strings.TrimSpace(*req.URL))
	}
	if req.Description != nil {
		if strings.TrimSpace(*req.Description) == "" {
			sets = append(sets, "description = NULL")
		} else {
			sets = append(sets, "description = ?")
			args = append(args, *req.Description)
		}
	}
	if len(sets) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tidak ada field yang diubah")
		return
	}
	sets = append(sets, "version = version + 1")
	args = append(args, matID, curVersion)
	res, err := c.db.Exec(`UPDATE materials SET `+strings.Join(sets, ", ")+` WHERE id = ? AND version = ? AND deleted_at IS NULL;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah materi")
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data berubah saat menyimpan", map[string]any{"current_version": curVersion})
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": matID, "version": curVersion + 1})
}

// DeleteMaterial menangani DELETE /api/v1/materials/{id} (arsip lunak)
func (c *AcademicController) DeleteMaterial(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "PJ" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM, PJ, atau System Admin yang berwenang mengarsipkan materi")
		return
	}
	matID, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if matID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID materi tidak valid")
		return
	}
	version := 0
	if v := strings.TrimSpace(r.URL.Query().Get("version")); v != "" {
		version, _ = strconv.Atoi(v)
	}
	if version <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib disertakan sebagai query parameter")
		return
	}
	var classID, offeringID sql.NullInt64
	var curVersion int
	var curStatus string
	if err := c.db.QueryRow(`SELECT class_id, course_offering_id, version, status FROM materials WHERE id = ? AND deleted_at IS NULL;`, matID).
		Scan(&classID, &offeringID, &curVersion, &curStatus); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Materi tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca materi")
		return
	}
	if curStatus != "ACTIVE" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Hanya materi aktif yang dapat diarsipkan")
		return
	}
	if version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{"current_version": curVersion})
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || !classID.Valid || u.ActiveClassID.Int64 != classID.Int64) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang mengarsipkan materi kelas penugasannya")
		return
	}
	if u.ActiveRole == "PJ" {
		if !offeringID.Valid || !u.ActiveCourseOfferingID.Valid || offeringID.Int64 != u.ActiveCourseOfferingID.Int64 {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya berwenang mengarsipkan materi offering penugasannya")
			return
		}
	}
	res, err := c.db.Exec(`UPDATE materials SET status = 'ARCHIVED', version = version + 1 WHERE id = ? AND version = ? AND deleted_at IS NULL;`, matID, curVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengarsipkan materi")
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data berubah saat menyimpan")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": matID, "status": "ARCHIVED"})
}

// GetRoomCandidates menangani GET /api/v1/rooms/candidates
func (c *AcademicController) GetRoomCandidates(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	startsAtStr := strings.TrimSpace(r.URL.Query().Get("starts_at"))
	endsAtStr := strings.TrimSpace(r.URL.Query().Get("ends_at"))

	if startsAtStr == "" || endsAtStr == "" {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Parameter starts_at dan ends_at wajib disertakan (RFC3339)")
		return
	}

	startsAt, err1 := time.Parse(time.RFC3339, startsAtStr)
	endsAt, err2 := time.Parse(time.RFC3339, endsAtStr)
	if err1 != nil || err2 != nil || !endsAt.After(startsAt) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format waktu tidak valid atau ends_at mendahului starts_at")
		return
	}

	excludeEventIDStr := strings.TrimSpace(r.URL.Query().Get("exclude_event_id"))
	excludeEventID, _ := strconv.ParseInt(excludeEventIDStr, 10, 64)

	// BE-006: definisi overlap sama [start,end): te.starts_at < endsAt AND te.ends_at > startsAt.
	// Kandidat mengecualikan ruangan yang dipakai event PUBLISHED maupun pola
	// reguler efektif pada tanggal candidate.
	dateStr := startsAt.Format("2006-01-02")
	dow := int(startsAt.Weekday())
	if dow == 0 {
		dow = 7
	}
	startHM := startsAt.Format("15:04")
	endHM := endsAt.Format("15:04")
	// Cari ruangan yang TIDAK sedang digunakan oleh event PUBLISHED pada rentang waktu tersebut
	query := `
		SELECT r.id, r.code, r.name, r.capacity, r.room_type
		FROM rooms r
		WHERE r.status = 'ACTIVE'
		  AND r.id NOT IN (
		      SELECT te.room_id
		      FROM teaching_events te
		      WHERE te.room_id IS NOT NULL
		        AND te.lifecycle_status = 'PUBLISHED'
		        AND te.id != ?
		        AND te.starts_at < ? AND te.ends_at > ?
		  )
		  AND r.id NOT IN (
		      SELECT sp.room_id FROM schedule_patterns sp
		      WHERE sp.room_id IS NOT NULL AND sp.status='ACTIVE' AND sp.day_of_week=?
		        AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
		        AND sp.start_time < ? AND sp.end_time > ?
		  )
		ORDER BY r.code;
	`

	rows, err := c.db.Query(query, excludeEventID, endsAt.Format(time.RFC3339), startsAt.Format(time.RFC3339),
		dow, dateStr, dateStr, endHM, startHM)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mencari kandidat ruangan: %v", err))
		return
	}
	defer rows.Close()

	candidates := []RoomCandidateItem{}
	for rows.Next() {
		var item RoomCandidateItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Capacity, &item.RoomType); err == nil {
			candidates = append(candidates, item)
		}
	}

	common.WriteV1Success(w, http.StatusOK, candidates)
}

// CreateRoomConfirmation menangani POST /api/v1/teaching-events/{id}/room-confirmations
func (c *AcademicController) CreateRoomConfirmation(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req RoomConfirmationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.ConfirmationStatus))
	if status != "PENDING" && status != "CONFIRMED" && status != "REJECTED" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Status konfirmasi harus PENDING, CONFIRMED, atau REJECTED")
		return
	}
	if req.RoomID <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "room_id wajib diisi")
		return
	}

	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi konfirmasi ruangan")
		return
	}
	defer tx.Rollback()

	var eventClassID int64
	var lifecycle, timezone string
	var startsAt, endsAt common.DBTimestamp
	scopeQuery := `
		SELECT sem.class_id, te.lifecycle_status, te.starts_at, te.ends_at,
		       COALESCE(cs.timezone, 'Asia/Jakarta')
		FROM teaching_events te
		JOIN teaching_event_offerings teo
		  ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN class_settings cs ON cs.class_id = sem.class_id
		WHERE te.id = ?`
	scopeArgs := []any{eventID}
	switch u.ActiveRole {
	case "PJ":
		if !u.ActiveCourseOfferingID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kejadian jadwal ditolak")
			return
		}
		scopeQuery += ` AND co.id = ?`
		scopeArgs = append(scopeArgs, u.ActiveCourseOfferingID.Int64)
	case "KM":
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kejadian jadwal ditolak")
			return
		}
		scopeQuery += ` AND sem.class_id = ?`
		scopeArgs = append(scopeArgs, u.ActiveClassID.Int64)
	case "SYSTEM_ADMIN":
	default:
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kejadian jadwal ditolak")
		return
	}
	err = tx.QueryRowContext(r.Context(), scopeQuery, scopeArgs...).Scan(
		&eventClassID, &lifecycle, &startsAt, &endsAt, &timezone,
	)
	if errors.Is(err, sql.ErrNoRows) {
		if u.ActiveRole == "SYSTEM_ADMIN" {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian jadwal tidak ditemukan")
		} else {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kejadian jadwal ditolak")
		}
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi cakupan kejadian jadwal")
		return
	}
	if lifecycle != "DRAFT" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Konfirmasi TU hanya dapat dicatat pada teaching event DRAFT")
		return
	}

	var roomExists bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ? AND status = 'ACTIVE')`, req.RoomID).Scan(&roomExists); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi ruangan")
		return
	}
	if !roomExists {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Ruangan tidak ditemukan atau nonaktif")
		return
	}

	if status == "CONFIRMED" {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			loc = time.FixedZone("WIB", 7*60*60)
		}
		localStart := startsAt.Time.In(loc)
		localEnd := endsAt.Time.In(loc)
		dayOfWeek := int(localStart.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7
		}
		date := localStart.Format("2006-01-02")
		var roomConflict bool
		err = tx.QueryRowContext(r.Context(), `SELECT EXISTS(
			SELECT 1 FROM teaching_events other
			WHERE other.room_id = ? AND other.id <> ? AND other.lifecycle_status = 'PUBLISHED'
			  AND other.starts_at < ? AND other.ends_at > ?
			UNION ALL
			SELECT 1 FROM schedule_patterns sp
			WHERE sp.room_id = ? AND sp.status = 'ACTIVE' AND sp.day_of_week = ?
			  AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
			  AND sp.start_time < ? AND sp.end_time > ?
		)`, req.RoomID, eventID, endsAt.Time.Format(time.RFC3339), startsAt.Time.Format(time.RFC3339),
			req.RoomID, dayOfWeek, date, date, localEnd.Format("15:04"), localStart.Format("15:04")).Scan(&roomConflict)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa ketersediaan ruangan")
			return
		}
		if roomConflict {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Ruangan tidak lagi tersedia pada rentang waktu kejadian")
			return
		}
	}

	var externalContact, note any
	if req.ExternalContact != nil && strings.TrimSpace(*req.ExternalContact) != "" {
		externalContact = strings.TrimSpace(*req.ExternalContact)
	}
	if req.Note != nil && strings.TrimSpace(*req.Note) != "" {
		note = strings.TrimSpace(*req.Note)
	}
	var confirmedAt any
	if status == "CONFIRMED" {
		confirmedAt = time.Now().UTC().Format(time.RFC3339)
	}
	res, err := tx.Exec(`
		INSERT INTO room_confirmations (
			teaching_event_id, room_id, confirmation_status, external_contact, note, recorded_by_user_id, recorded_at, confirmed_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?);
	`, eventID, req.RoomID, status, externalContact, note, u.UserID, confirmedAt)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan konfirmasi ruangan: %v", err))
		return
	}

	confID, _ := res.LastInsertId()

	// Update room_id pada teaching_event jika dikonfirmasi
	if status == "CONFIRMED" {
		if _, err = tx.Exec(`UPDATE teaching_events SET room_id = ?, version = version + 1 WHERE id = ?;`, req.RoomID, eventID); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui ruangan kejadian")
			return
		}
	}

	// Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"event_id":%d,"room_id":%d,"status":%q}`, eventID, req.RoomID, status)
		correlationID := fmt.Sprintf("room-confirmation-%d-%d", confID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &eventClassID,
			Action:        "CREATE_ROOM_CONFIRMATION",
			EntityType:    "ROOM_CONFIRMATION",
			EntityID:      &confID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit konfirmasi ruangan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit konfirmasi ruangan")
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":                  confID,
		"teaching_event_id":   eventID,
		"room_id":             req.RoomID,
		"confirmation_status": status,
	})
}

// hasImportErrNewCols mendeteksi skema import_errors baru (source_location)
// vs legacy (row_number) agar tulis tetap jalan di kedua DB.
func hasImportErrNewCols(tx *sql.Tx) bool {
	rows, err := tx.Query(`PRAGMA table_info(import_errors);`)
	if err != nil {
		return true
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err == nil {
			if name == "source_location" {
				return true
			}
		}
	}
	return false
}
