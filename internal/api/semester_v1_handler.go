package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bot-jadwal/internal/audit"
)

// handleGetClassSemesters menangani GET /api/v1/classes/{slug}/semesters
func (s *Server) handleGetClassSemesters(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	u, _ := GetAuthContext(r)
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Akses kelas ditolak")
		return
	}

	rows, err := s.v1DB.Query(`
		SELECT id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, version
		FROM semesters
		WHERE class_id = ?
		ORDER BY starts_on DESC;
	`, classID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat semester")
		return
	}
	defer rows.Close()

	var semesters []map[string]any
	for rows.Next() {
		var id int64
		var year, term, startsOn, endsOn, status string
		var publishedAt, activatedAt dbTimestamp
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

	s.writeV1Success(w, http.StatusOK, semesters)
}

// CreateSemesterRequest adalah payload pembuatan semester baru
type CreateSemesterRequest struct {
	AcademicYear string `json:"academic_year"`
	Term         string `json:"term"`
	StartsOn     string `json:"starts_on"`
	EndsOn       string `json:"ends_on"`
}

// handleCreateClassSemester menangani POST /api/v1/classes/{slug}/semesters
func (s *Server) handleCreateClassSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang membuat semester")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang membuat semester untuk kelas penugasannya")
		return
	}

	var req CreateSemesterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	startDate, errStart := time.Parse("2006-01-02", req.StartsOn)
	endDate, errEnd := time.Parse("2006-01-02", req.EndsOn)
	if errStart != nil || errEnd != nil || !endDate.After(startDate) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Format tanggal harus YYYY-MM-DD dan ends_on harus setelah starts_on")
		return
	}

	var semID int64
	err = s.v1DB.QueryRow(`
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, version)
		VALUES (?, ?, ?, ?, ?, 'DRAFT', 1)
		RETURNING id;
	`, classID, req.AcademicYear, req.Term, req.StartsOn, req.EndsOn).Scan(&semID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan semester: %v", err))
		return
	}

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"id":     semID,
		"status": "DRAFT",
	})
}

// handleActivateSemester menangani POST /api/v1/classes/{slug}/semesters/{id}/activate
func (s *Server) handleActivateSemester(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang mengaktifkan semester")
		return
	}
	var confirm struct {
		Confirm bool `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&confirm); err != nil || !confirm.Confirm {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "confirm:true wajib disertakan")
		return
	}

	semIDStr := r.PathValue("id")
	semID, _ := strconv.ParseInt(semIDStr, 10, 64)

	slug := r.PathValue("slug")
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang mengaktifkan semester kelas penugasannya")
		return
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi aktivasi")
		return
	}
	defer tx.Rollback()

	// 1. Arsipkan semester aktif lama
	if _, err = tx.Exec(`
		UPDATE semesters
		SET status = 'ARCHIVED', archived_at = CURRENT_TIMESTAMP
		WHERE class_id = ? AND status = 'ACTIVE';
	`, classID); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengarsipkan semester aktif")
		return
	}

	// 2. Aktifkan semester baru
	res, err := tx.Exec(`
		UPDATE semesters
		SET status = 'ACTIVE', published_at = CURRENT_TIMESTAMP, activated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND class_id = ?;
	`, semID, classID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengaktifkan semester")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Semester tidak ditemukan")
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
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit aktivasi semester")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit aktivasi semester")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"semester_id": semID,
		"status":      "ACTIVE",
	})
}

// handleGetSemesterOfferings menangani GET /api/v1/semesters/{id}/offerings
func (s *Server) handleGetSemesterOfferings(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	semIDStr := r.PathValue("id")
	semID, _ := strconv.ParseInt(semIDStr, 10, 64)
	u, _ := GetAuthContext(r)
	var classID int64
	if err := s.v1DB.QueryRow(`SELECT class_id FROM semesters WHERE id = ?;`, semID).Scan(&classID); err != nil {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Semester tidak ditemukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Akses semester ditolak")
		return
	}

	rows, err := s.v1DB.Query(`
		SELECT co.id, c.code, co.display_name, co.activity_type
		FROM course_offerings co
		JOIN courses c ON co.course_id = c.id
		WHERE co.semester_id = ? AND co.status = 'ACTIVE'
		ORDER BY co.display_name;
	`, semID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat mata kuliah offering")
		return
	}
	defer rows.Close()

	var offerings []map[string]any
	for rows.Next() {
		var id int64
		var code, displayName, actType string
		if err := rows.Scan(&id, &code, &displayName, &actType); err == nil {
			lecturers := s.getOfferingLecturers(id)
			offerings = append(offerings, map[string]any{
				"id":            id,
				"course_code":   code,
				"display_name":  displayName,
				"activity_type": actType,
				"lecturers":     lecturers,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, offerings)
}
