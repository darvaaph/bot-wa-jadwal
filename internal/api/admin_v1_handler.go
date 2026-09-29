package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// AdminStatusResponse merepresentasikan metrik telemetri sistem global
type AdminStatusResponse struct {
	Timestamp      string `json:"timestamp"`
	BotConnection  string `json:"bot_connection"`
	ClassesSummary struct {
		Total    int `json:"total"`
		Active   int `json:"active"`
		Inactive int `json:"inactive"`
		Archived int `json:"archived"`
	} `json:"classes_summary"`
	UsersSummary struct {
		Total     int `json:"total"`
		Active    int `json:"active"`
		Suspended int `json:"suspended"`
	} `json:"users_summary"`
	TasksSummary struct {
		Total     int `json:"total"`
		Published int `json:"published"`
		Draft     int `json:"draft"`
	} `json:"tasks_summary"`
	EventsSummary struct {
		TotalPatterns int `json:"total_patterns"`
		TotalEvents   int `json:"total_events"`
	} `json:"events_summary"`
}

// handleGetAdminStatus menangani GET /api/v1/admin/status
func (s *Server) handleGetAdminStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya System Admin yang berwenang mengakses status sistem global")
		return
	}

	botStatus := "uninitialized"
	if s.botClient != nil {
		botStatus = s.botClient.Status()
	}

	resp := AdminStatusResponse{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		BotConnection: botStatus,
	}

	// Hitung Kelas
	_ = s.v1DB.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'INACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'ARCHIVED' THEN 1 ELSE 0 END), 0)
		FROM classes;
	`).Scan(&resp.ClassesSummary.Total, &resp.ClassesSummary.Active, &resp.ClassesSummary.Inactive, &resp.ClassesSummary.Archived)

	// Hitung Pengguna
	_ = s.v1DB.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'SUSPENDED' THEN 1 ELSE 0 END), 0)
		FROM users;
	`).Scan(&resp.UsersSummary.Total, &resp.UsersSummary.Active, &resp.UsersSummary.Suspended)

	// Hitung Tugas
	_ = s.v1DB.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN publication_status = 'PUBLISHED' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN publication_status = 'DRAFT' THEN 1 ELSE 0 END), 0)
		FROM tasks
		WHERE deleted_at IS NULL;
	`).Scan(&resp.TasksSummary.Total, &resp.TasksSummary.Published, &resp.TasksSummary.Draft)

	// Hitung Jadwal
	_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM schedule_patterns WHERE status = 'ACTIVE';`).Scan(&resp.EventsSummary.TotalPatterns)
	_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM teaching_events;`).Scan(&resp.EventsSummary.TotalEvents)

	s.writeV1Success(w, http.StatusOK, resp)
}

// SuspendUserRequest payload penangguhan akun pengguna
type SuspendUserRequest struct {
	Reason string `json:"reason"`
}

// handleAdminSuspendUser menangani POST /api/v1/admin/users/{id}/suspend
func (s *Server) handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya System Admin yang berwenang menangguhkan akun pengguna")
		return
	}

	targetUserIDStr := r.PathValue("id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil || targetUserID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID pengguna tidak valid")
		return
	}

	if targetUserID == u.UserID {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Tidak dapat menangguhkan akun Anda sendiri")
		return
	}

	var req SuspendUserRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Penangguhan administratif oleh System Admin"
	}

	var curStatus string
	err = s.v1DB.QueryRow(`SELECT status FROM users WHERE id = ?;`, targetUserID).Scan(&curStatus)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Pengguna tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pengguna")
		return
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi penangguhan")
		return
	}
	defer tx.Rollback()
	// Update status user menjadi SUSPENDED dan naikkan session_version agar seluruh token sesi lama langsung invalid
	_, err = tx.Exec(`
		UPDATE users
		SET status = 'SUSPENDED', session_version = session_version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`, targetUserID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menangguhkan akun pengguna")
		return
	}

	// Catat audit_logs
	if _, err = tx.Exec(`
		INSERT INTO audit_logs (actor_type, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, before_json, after_json, reason, correlation_id)
		VALUES ('USER', ?, ?, 'SUSPEND_USER', 'USER', ?, ?, '{"status":"SUSPENDED"}', ?, ?);
	`, u.UserID, u.ActiveAssignmentID, targetUserID, fmt.Sprintf(`{"status":%q}`, curStatus), reason, fmt.Sprintf("suspend-user-%d-%d", targetUserID, time.Now().UnixNano())); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit penangguhan")
		return
	}
	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit penangguhan")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"user_id": targetUserID,
		"status":  "SUSPENDED",
		"message": "Akun berhasil ditangguhkan dan seluruh sesi aktif telah dicabut",
	})
}

// RecoverUserRequest payload pemulihan akun pengguna
type RecoverUserRequest struct {
	NewPassword *string `json:"new_password,omitempty"`
	Reason      *string `json:"reason,omitempty"`
}

// handleAdminRecoverUser menangani POST /api/v1/admin/users/{id}/recover
func (s *Server) handleAdminRecoverUser(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya System Admin yang berwenang memulihkan akun pengguna")
		return
	}

	targetUserIDStr := r.PathValue("id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil || targetUserID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID pengguna tidak valid")
		return
	}

	var req RecoverUserRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	var curStatus string
	err = s.v1DB.QueryRow(`SELECT status FROM users WHERE id = ?;`, targetUserID).Scan(&curStatus)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Pengguna tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pengguna")
		return
	}

	var updateQuery string
	var args []any
	if req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(*req.NewPassword)), bcrypt.DefaultCost)
		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Gagal mengenkripsi kata sandi baru")
			return
		}
		updateQuery = `
			UPDATE users
			SET status = 'ACTIVE', password_hash = ?, session_version = session_version + 1, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?;
		`
		args = append(args, string(hash), targetUserID)
	} else {
		updateQuery = `
			UPDATE users
			SET status = 'ACTIVE', session_version = session_version + 1, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?;
		`
		args = append(args, targetUserID)
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi pemulihan")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulihkan akun pengguna")
		return
	}

	// Catat audit_logs
	if _, err = tx.Exec(`
		INSERT INTO audit_logs (actor_type, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, before_json, after_json, reason, correlation_id)
		VALUES ('USER', ?, ?, 'RECOVER_USER', 'USER', ?, ?, '{"status":"ACTIVE"}', ?, ?);
	`, u.UserID, u.ActiveAssignmentID, targetUserID, fmt.Sprintf(`{"status":%q}`, curStatus), req.Reason, fmt.Sprintf("recover-user-%d-%d", targetUserID, time.Now().UnixNano())); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit pemulihan")
		return
	}
	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit pemulihan")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"user_id": targetUserID,
		"status":  "ACTIVE",
		"message": "Akun pengguna berhasil dipulihkan",
	})
}
