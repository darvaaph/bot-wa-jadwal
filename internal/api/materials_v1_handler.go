package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// handleGetV1Materials menangani GET /api/v1/materials
func (s *Server) handleGetV1Materials(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	classSlug := r.URL.Query().Get("class_slug")
	offeringParam := r.URL.Query().Get("offering_id")

	query := `
		SELECT m.id, m.class_id, c.slug, m.course_offering_id, m.task_id, m.title,
		       m.material_type, COALESCE(m.url, ''), COALESCE(m.description, '')
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

	rows, err := s.v1DB.Query(query, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat materi")
		return
	}
	defer rows.Close()

	var materials []map[string]any
	for rows.Next() {
		var id, classID int64
		var slug, title, matType, urlStr, desc string
		var offID, taskID sql.NullInt64

		if err := rows.Scan(&id, &classID, &slug, &offID, &taskID, &title, &matType, &urlStr, &desc); err == nil {
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
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, materials)
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

// handleCreateV1Material menangani POST /api/v1/materials
func (s *Server) handleCreateV1Material(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.ClassSlug) == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "title dan class_slug wajib diisi")
		return
	}

	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, req.ClassSlug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang menambahkan materi untuk kelas penugasannya")
		return
	}

	// Batasan peran: PJ hanya boleh menambah materi untuk offering penugasannya
	if u.ActiveRole == "PJ" {
		if req.OfferingID == nil || !u.ActiveCourseOfferingID.Valid || *req.OfferingID != u.ActiveCourseOfferingID.Int64 {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "PJ hanya berwenang menambahkan materi untuk offering penugasannya")
			return
		}
	}
	if req.OfferingID != nil {
		var offeringClassID int64
		if err := s.v1DB.QueryRow(`
			SELECT sem.class_id FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?;
		`, *req.OfferingID).Scan(&offeringClassID); err != nil || offeringClassID != classID {
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "offering_id tidak berada pada class_slug yang dipilih")
			return
		}
	}
	if req.TaskID != nil {
		var taskClassID, taskOfferingID int64
		if err := s.v1DB.QueryRow(`
			SELECT sem.class_id, t.course_offering_id FROM tasks t
			JOIN course_offerings co ON co.id = t.course_offering_id
			JOIN semesters sem ON sem.id = co.semester_id WHERE t.id = ?;
		`, *req.TaskID).Scan(&taskClassID, &taskOfferingID); err != nil || taskClassID != classID || (req.OfferingID != nil && taskOfferingID != *req.OfferingID) {
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "task_id tidak konsisten dengan kelas atau offering")
			return
		}
	}

	matType := strings.ToUpper(strings.TrimSpace(req.MaterialType))
	if matType == "" {
		matType = "OTHER"
	}

	var matID int64
	err = s.v1DB.QueryRow(`
		INSERT INTO materials (
			class_id, course_offering_id, task_id, title, material_type,
			url, description, visibility, status, version, created_by_user_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'CLASS_ACCESS', 'ACTIVE', 1, ?)
		RETURNING id;
	`, classID, req.OfferingID, req.TaskID, req.Title, matType, req.URL, req.Description, u.UserID).Scan(&matID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan materi: %v", err))
		return
	}

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"id":     matID,
		"status": "ACTIVE",
	})
}
