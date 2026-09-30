package v1

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/ratelimit"
	"golang.org/x/crypto/bcrypt"
)

// BotStatusProvider menyediakan status koneksi bot WhatsApp
type BotStatusProvider interface {
	Status() string
}

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

// SuspendUserRequest payload penangguhan akun pengguna
type SuspendUserRequest struct {
	Reason string `json:"reason"`
}

// RecoverUserRequest payload pemulihan akun pengguna
type RecoverUserRequest struct {
	NewPassword *string `json:"new_password,omitempty"`
	Reason      *string `json:"reason,omitempty"`
}

// AdminUserItem merepresentasikan satu pengguna beserta ringkasan penugasannya
type AdminUserItem struct {
	ID          int64    `json:"id"`
	IdentityKey string   `json:"identity_key"`
	DisplayName string   `json:"display_name"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
}

// GetUsers menangani GET /api/v1/admin/users
func (c *AdminController) GetUsers(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang melihat daftar pengguna")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	query := `SELECT id, identity_key, display_name, status FROM users WHERE (1=1)`
	var args []any
	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY id ASC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat daftar pengguna")
		return
	}
	defer rows.Close()

	users := []AdminUserItem{}
	for rows.Next() {
		var item AdminUserItem
		if err := rows.Scan(&item.ID, &item.IdentityKey, &item.DisplayName, &item.Status); err != nil {
			continue
		}
		roleRows, err := c.db.Query(`
			SELECT DISTINCT ra.role FROM role_assignments ra
			WHERE ra.user_id = ? AND ra.status = 'ACTIVE' ORDER BY ra.role;
		`, item.ID)
		if err == nil {
			for roleRows.Next() {
				var role string
				if err := roleRows.Scan(&role); err == nil {
					item.Roles = append(item.Roles, role)
				}
			}
			roleRows.Close()
		}
		if item.Roles == nil {
			item.Roles = []string{}
		}
		users = append(users, item)
	}

	common.WriteV1Success(w, http.StatusOK, users)
}

// AuditLogResponseItem merepresentasikan catatan riwayat audit sistem
type AuditLogResponseItem struct {
	ID          int64   `json:"id"`
	ClassID     *int64  `json:"class_id,omitempty"`
	ClassSlug   *string `json:"class_slug,omitempty"`
	ActorUserID *int64  `json:"actor_user_id,omitempty"`
	ActorName   *string `json:"actor_name,omitempty"`
	ActorRole   *string `json:"actor_role,omitempty"`
	Action      string  `json:"action"`
	EntityType  *string `json:"entity_type,omitempty"`
	EntityID    *int64  `json:"entity_id,omitempty"`
	BeforeJSON  *string `json:"before_json,omitempty"`
	AfterJSON   *string `json:"after_json,omitempty"`
	Reason      *string `json:"reason,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// BackupRequest merepresentasikan payload pembuatan backup on-demand.
// SemesterID opsional: bila diisi, dicatat sebagai cakupan dan diverifikasi
// saat verifikasi (BE-010/BE-011). Snapshot berkas tetap per kelas.
type BackupRequest struct {
	ClassSlug  *string `json:"class_slug,omitempty"`
	SemesterID *int64  `json:"semester_id,omitempty"`
	Reason     *string `json:"reason,omitempty"`
}

// BackupResponseItem merepresentasikan catatan cadangan basis data.
// artifact_ref tidak dikembalikan ke client (jalur internal).
// Status v1: CREATING, READY, VERIFIED, FAILED (tanpa RESTORING per ADR-0008).
type BackupResponseItem struct {
	ID         int64   `json:"id"`
	ClassID    *int64  `json:"class_id,omitempty"`
	ClassSlug  *string `json:"class_slug,omitempty"`
	SemesterID *int64  `json:"semester_id,omitempty"`
	Checksum   string  `json:"checksum"`
	Status     string  `json:"status"` // CREATING, READY, VERIFIED, FAILED
	Reason     *string `json:"reason,omitempty"`
	CreatedBy  *string `json:"created_by,omitempty"`
	CreatedAt  string  `json:"created_at"`
	VerifiedAt *string `json:"verified_at,omitempty"`
}

// RestoreRequest merepresentasikan payload permintaan pemulihan database
type RestoreRequest struct {
	BackupID int64   `json:"backup_id"`
	Reason   *string `json:"reason,omitempty"`
}

// NotificationResponseItem merepresentasikan pesan notifikasi dalam antrean siaran WhatsApp
type NotificationResponseItem struct {
	ID           int64   `json:"id"`
	ClassID      int64   `json:"class_id"`
	EventType    string  `json:"event_type"`
	EntityType   *string `json:"entity_type,omitempty"`
	EntityID     *int64  `json:"entity_id,omitempty"`
	Status       string  `json:"status"` // PENDING, PROCESSING, SENT, FAILED, CANCELLED
	PayloadJSON  string  `json:"payload_json"`
	ScheduledAt  *string `json:"scheduled_at,omitempty"`
	SentAt       *string `json:"sent_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	AttemptCount int     `json:"attempt_count"`
}

// AdminController mengelola telemetri sistem, penangguhan/pemulihan akun, audit logs, backup & restore, dan notifikasi outbox
type AdminController struct {
	db            *sql.DB
	botClient     BotStatusProvider
	secManager    *middleware.SecurityManager
	rlManager     *middleware.RateLimitManager
	getStorageDir func() string
}

// NewAdminController membuat instance baru AdminController
func NewAdminController(db *sql.DB, botClient BotStatusProvider, secManager *middleware.SecurityManager, rlManager *middleware.RateLimitManager, getStorageDir func() string) *AdminController {
	if getStorageDir == nil {
		getStorageDir = func() string { return "storage" }
	}
	return &AdminController{
		db:            db,
		botClient:     botClient,
		secManager:    secManager,
		rlManager:     rlManager,
		getStorageDir: getStorageDir,
	}
}

// GetAdminStatus menangani GET /api/v1/admin/status
func (c *AdminController) GetAdminStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang mengakses status sistem global")
		return
	}

	botStatus := "uninitialized"
	if c.botClient != nil {
		botStatus = c.botClient.Status()
	}

	resp := AdminStatusResponse{
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		BotConnection: botStatus,
	}

	// Hitung Kelas
	_ = c.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'INACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'ARCHIVED' THEN 1 ELSE 0 END), 0)
		FROM classes;
	`).Scan(&resp.ClassesSummary.Total, &resp.ClassesSummary.Active, &resp.ClassesSummary.Inactive, &resp.ClassesSummary.Archived)

	// Hitung Pengguna
	_ = c.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN status = 'ACTIVE' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'SUSPENDED' THEN 1 ELSE 0 END), 0)
		FROM users;
	`).Scan(&resp.UsersSummary.Total, &resp.UsersSummary.Active, &resp.UsersSummary.Suspended)

	// Hitung Tugas
	_ = c.db.QueryRow(`
		SELECT COUNT(*),
		       COALESCE(SUM(CASE WHEN publication_status = 'PUBLISHED' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN publication_status = 'DRAFT' THEN 1 ELSE 0 END), 0)
		FROM tasks
		WHERE deleted_at IS NULL;
	`).Scan(&resp.TasksSummary.Total, &resp.TasksSummary.Published, &resp.TasksSummary.Draft)

	// Hitung Jadwal
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM schedule_patterns WHERE status = 'ACTIVE';`).Scan(&resp.EventsSummary.TotalPatterns)
	_ = c.db.QueryRow(`SELECT COUNT(*) FROM teaching_events;`).Scan(&resp.EventsSummary.TotalEvents)

	common.WriteV1Success(w, http.StatusOK, resp)
}

// SuspendUser menangani POST /api/v1/admin/users/{id}/suspend
func (c *AdminController) SuspendUser(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang menangguhkan akun pengguna")
		return
	}

	targetUserIDStr := r.PathValue("id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil || targetUserID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID pengguna tidak valid")
		return
	}

	if targetUserID == u.UserID {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Tidak dapat menangguhkan akun Anda sendiri")
		return
	}

	// BE-012: batasi abuse mutasi admin tanpa mengubah authorization.
	limitSubject := fmt.Sprintf("admin:%d", u.UserID)
	var trustedProxies []string
	if c.secManager != nil {
		trustedProxies = c.secManager.TrustedProxyCIDRs()
	}
	limitSource := middleware.ClientSource(r, trustedProxies)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trustedProxies, ratelimit.PolicyAdminMutation, limitSubject) {
		return
	}

	var req SuspendUserRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "Penangguhan administratif oleh System Admin"
	}

	var curStatus string
	err = c.db.QueryRow(`SELECT status FROM users WHERE id = ?;`, targetUserID).Scan(&curStatus)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pengguna tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pengguna")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi penangguhan")
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
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menangguhkan akun pengguna")
		return
	}

	// Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, curStatus)
		afterJSON := `{"status":"SUSPENDED"}`
		correlationID := fmt.Sprintf("suspend-user-%d-%d", targetUserID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			Action:        "SUSPEND_USER",
			EntityType:    "USER",
			EntityID:      &targetUserID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit penangguhan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit penangguhan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyAdminMutation, limitSubject, limitSource, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user_id": targetUserID,
		"status":  "SUSPENDED",
		"message": "Akun berhasil ditangguhkan dan seluruh sesi aktif telah dicabut",
	})
}

// RecoverUser menangani POST /api/v1/admin/users/{id}/recover
func (c *AdminController) RecoverUser(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang memulihkan akun pengguna")
		return
	}

	targetUserIDStr := r.PathValue("id")
	targetUserID, err := strconv.ParseInt(targetUserIDStr, 10, 64)
	if err != nil || targetUserID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID pengguna tidak valid")
		return
	}

	// BE-012: batasi abuse mutasi admin tanpa mengubah authorization.
	recoverSubject := fmt.Sprintf("admin:%d", u.UserID)
	var trustedProxies []string
	if c.secManager != nil {
		trustedProxies = c.secManager.TrustedProxyCIDRs()
	}
	recoverSource := middleware.ClientSource(r, trustedProxies)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trustedProxies, ratelimit.PolicyAdminMutation, recoverSubject) {
		return
	}

	var req RecoverUserRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != "" && len(strings.TrimSpace(*req.NewPassword)) < 12 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kata sandi baru minimal 12 karakter")
		return
	}

	var curStatus string
	err = c.db.QueryRow(`SELECT status FROM users WHERE id = ?;`, targetUserID).Scan(&curStatus)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pengguna tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pengguna")
		return
	}

	var updateQuery string
	var args []any
	if req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(strings.TrimSpace(*req.NewPassword)), bcrypt.DefaultCost)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Gagal mengenkripsi kata sandi baru")
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

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi pemulihan")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(updateQuery, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulihkan akun pengguna")
		return
	}

	// Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, curStatus)
		afterJSON := `{"status":"ACTIVE"}`
		recoverReason := ""
		if req.Reason != nil {
			recoverReason = *req.Reason
		}
		correlationID := fmt.Sprintf("recover-user-%d-%d", targetUserID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			Action:        "RECOVER_USER",
			EntityType:    "USER",
			EntityID:      &targetUserID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        recoverReason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit pemulihan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit pemulihan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyAdminMutation, recoverSubject, recoverSource, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user_id": targetUserID,
		"status":  "ACTIVE",
		"message": "Akun pengguna berhasil dipulihkan",
	})
}

// GetAuditLogs menangani GET /api/v1/audit
func (c *AdminController) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang menelaah log audit")
		return
	}

	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	actionFilter := strings.TrimSpace(r.URL.Query().Get("action"))
	entityTypeFilter := strings.TrimSpace(r.URL.Query().Get("entity_type"))

	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT al.id, al.class_id, cl.slug, al.actor_user_id, u.display_name,
		       ra.role, al.action, al.entity_type, al.entity_id,
		       al.before_json, al.after_json, al.reason, al.created_at
		FROM audit_logs al
		LEFT JOIN classes cl ON al.class_id = cl.id
		LEFT JOIN users u ON al.actor_user_id = u.id
		LEFT JOIN role_assignments ra ON al.actor_role_assignment_id = ra.id
		WHERE (1=1)
	`
	var args []any

	// Pembatasan cakupan: KM hanya dapat membaca audit kelas miliknya
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks kelas KM tidak valid")
			return
		}
		query += " AND al.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND cl.slug = ?"
		args = append(args, classSlugFilter)
	}

	if actionFilter != "" {
		query += " AND al.action = ?"
		args = append(args, actionFilter)
	}

	if entityTypeFilter != "" {
		query += " AND al.entity_type = ?"
		args = append(args, entityTypeFilter)
	}

	query += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memuat log audit: %v", err))
		return
	}
	defer rows.Close()

	var logs []AuditLogResponseItem
	for rows.Next() {
		var item AuditLogResponseItem
		var classID, actorUID, entityID sql.NullInt64
		var classSlug, actorName, actorRole, entityType, beforeJSON, afterJSON, reason sql.NullString
		var createdAt common.DBTimestamp

		if err := rows.Scan(
			&item.ID, &classID, &classSlug, &actorUID, &actorName,
			&actorRole, &item.Action, &entityType, &entityID,
			&beforeJSON, &afterJSON, &reason, &createdAt,
		); err == nil {
			if classID.Valid {
				item.ClassID = &classID.Int64
			}
			if classSlug.Valid {
				item.ClassSlug = &classSlug.String
			}
			if actorUID.Valid {
				item.ActorUserID = &actorUID.Int64
			}
			if actorName.Valid {
				item.ActorName = &actorName.String
			}
			if actorRole.Valid {
				item.ActorRole = &actorRole.String
			}
			if entityType.Valid {
				item.EntityType = &entityType.String
			}
			if entityID.Valid {
				item.EntityID = &entityID.Int64
			}
			if beforeJSON.Valid {
				item.BeforeJSON = &beforeJSON.String
			}
			if afterJSON.Valid {
				item.AfterJSON = &afterJSON.String
			}
			if reason.Valid {
				item.Reason = &reason.String
			}
			item.CreatedAt = createdAt.Time.Format(time.RFC3339)
			logs = append(logs, item)
		}
	}

	common.WriteV1Success(w, http.StatusOK, logs)
}

// CreateBackup menangani POST /api/v1/backups
func (c *AdminController) CreateBackup(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang memicu pencadangan database")
		return
	}

	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	// Urutan ini disengaja: KM di luar cakupan selalu menerima 404 yang sama
	// dengan kelas tak ada, agar keberadaan kelas lain tak terungkap (FR-ACCESS-004).
	var classID int64
	classFound := false
	if req.ClassSlug != nil && strings.TrimSpace(*req.ClassSlug) != "" {
		if err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, strings.TrimSpace(*req.ClassSlug)).Scan(&classID); err == nil {
			classFound = true
		}
		if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || !classFound || u.ActiveClassID.Int64 != classID) {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
			return
		}
		if !classFound {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
			return
		}
	} else if u.ActiveClassID.Valid {
		classID = u.ActiveClassID.Int64
	} else {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "class_slug wajib untuk System Admin")
		return
	}
	if u.ActiveRole == "KM" && u.ActiveClassID.Int64 != classID {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	// BE-010: cakupan semester opsional, wajib milik kelas yang dicadangkan.
	var semesterID sql.NullInt64
	if req.SemesterID != nil {
		if *req.SemesterID <= 0 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "semester_id tidak valid")
			return
		}
		var found int64
		if err := c.db.QueryRow(`SELECT id FROM semesters WHERE id = ? AND class_id = ?;`, *req.SemesterID, classID).Scan(&found); err != nil {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan pada kelas ini")
			return
		}
		semesterID = sql.NullInt64{Int64: found, Valid: true}
	}

	// BE-012: batasi frekuensi operasi mahal dan berisiko.
	backupSubject := fmt.Sprintf("backup:%d", u.UserID)
	var trustedProxies []string
	if c.secManager != nil {
		trustedProxies = c.secManager.TrustedProxyCIDRs()
	}
	backupSource := middleware.ClientSource(r, trustedProxies)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trustedProxies, ratelimit.PolicyBackupRestore, backupSubject) {
		return
	}

	backupDir := filepath.Join(c.getStorageDir(), "backups")
	_ = os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("backup_v1_%s_%d.db", timestamp, time.Now().UnixNano())
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// Lakukan VACUUM INTO untuk membuat snapshot SQLite secara aman tanpa mengunci penulisan
	_, err := c.db.Exec(fmt.Sprintf("VACUUM INTO '%s';", filepath.ToSlash(backupFilePath)))
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", fmt.Sprintf("Gagal membuat snapshot database: %v", err))
		return
	}

	// Hitung checksum berkas hasil backup
	f, err := os.Open(backupFilePath)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Gagal membaca berkas hasil cadangan")
		return
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Gagal menghitung checksum cadangan")
		return
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi backup")
		return
	}
	defer tx.Rollback()
	reasonStr := "Backup on-demand"
	if req.Reason != nil && strings.TrimSpace(*req.Reason) != "" {
		reasonStr = strings.TrimSpace(*req.Reason)
	}
	res, err := tx.Exec(`
		INSERT INTO backup_records (
			class_id, semester_id, artifact_ref, checksum, status, created_by_user_id, reason, created_at
		) VALUES (?, ?, ?, ?, 'READY', ?, ?, CURRENT_TIMESTAMP);
	`, classID, semesterID, backupFilePath, checksum, u.UserID, reasonStr)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan rekam cadangan: %v", err))
		return
	}

	backupID, _ := res.LastInsertId()

	var createdAt string
	_ = tx.QueryRow(`SELECT created_at FROM backup_records WHERE id = ?;`, backupID).Scan(&createdAt)
	var outSemesterID *int64
	if semesterID.Valid {
		outSemesterID = &semesterID.Int64
	}

	// Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"checksum":%q,"semester_id":%v}`, checksum, nullableInt(semesterID))
		correlationID := fmt.Sprintf("create-backup-%d-%d", backupID, time.Now().UnixNano())
		createEntry := audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "CREATE_BACKUP",
			EntityType:    "BACKUP_RECORD",
			EntityID:      &backupID,
			AfterJSON:     &afterJSON,
			Reason:        reasonStr,
			CorrelationID: correlationID,
		}
		if semesterID.Valid {
			createEntry.SemesterID = &semesterID.Int64
		}
		if err := audit.Write(r.Context(), tx, createEntry); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit backup")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit backup")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyBackupRestore, backupSubject, backupSource, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusCreated, BackupResponseItem{
		ID:         backupID,
		ClassID:    &classID,
		SemesterID: outSemesterID,
		Checksum:   checksum,
		Status:     "READY",
		Reason:     &reasonStr,
		CreatedAt:  createdAt,
	})
}

// nullableInt memformat sql.NullInt64 untuk audit JSON (null bila invalid).
func nullableInt(n sql.NullInt64) string {
	if n.Valid {
		return strconv.FormatInt(n.Int64, 10)
	}
	return "null"
}

// RestoreBackup menangani POST /api/v1/restores
func (c *AdminController) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang melakukan verifikasi dan pemulihan database")
		return
	}

	// BE-012: batasi frekuensi operasi mahal dan berisiko.
	restoreSubject := fmt.Sprintf("backup:%d", u.UserID)
	var trustedProxies []string
	if c.secManager != nil {
		trustedProxies = c.secManager.TrustedProxyCIDRs()
	}
	restoreSource := middleware.ClientSource(r, trustedProxies)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trustedProxies, ratelimit.PolicyBackupRestore, restoreSubject) {
		return
	}

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BackupID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID cadangan (backup_id) wajib diisi")
		return
	}
	// BE-011 (ADR-0008 verify-only): reason non-kosong wajib untuk jejak audit.
	if req.Reason == nil || strings.TrimSpace(*req.Reason) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "reason wajib diisi untuk verifikasi backup")
		return
	}

	var (
		artifactRef string
		expectedChk string
		curStatus   string
		classID     int64
		semesterID  sql.NullInt64
	)

	err := c.db.QueryRow(`
		SELECT artifact_ref, checksum, status, class_id, semester_id
		FROM backup_records
		WHERE id = ?;
	`, req.BackupID).Scan(&artifactRef, &expectedChk, &curStatus, &classID, &semesterID)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Catatan cadangan tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi catatan cadangan")
		return
	}

	// BE-011: artifact harus berada di storage backup yang dikonfigurasi.
	// Tolak traversal dan symlink escape; 404 generik tanpa path internal.
	backupDir := filepath.Join(c.getStorageDir(), "backups")
	absBase, err := filepath.Abs(backupDir)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi direktori cadangan")
		return
	}
	absTarget, err := filepath.Abs(artifactRef)
	if err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Berkas cadangan tidak ditemukan")
		return
	}
	if absTarget != absBase && !strings.HasPrefix(absTarget, absBase+string(os.PathSeparator)) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Berkas cadangan tidak ditemukan")
		return
	}
	if resolved, err := filepath.EvalSymlinks(absTarget); err == nil {
		if resolved != absBase && !strings.HasPrefix(resolved, absBase+string(os.PathSeparator)) {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Berkas cadangan tidak ditemukan")
			return
		}
		absTarget = resolved
	}

	// Cek fisik file di disk
	f, err := os.Open(absTarget)
	if err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Berkas cadangan tidak ditemukan")
		return
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "IO_ERROR", "Gagal membaca berkas cadangan untuk validasi")
		return
	}
	actualChk := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(actualChk, expectedChk) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, "CHECKSUM_MISMATCH", "Integritas berkas cadangan rusak: checksum tidak cocok")
		return
	}

	// BE-011: verifikasi format SQLite, kompatibilitas schema, scope metadata,
	// cakupan semester bila dicatat, dan integritas relasi — tanpa memodifikasi
	// database aktif (ADR-0008 verify-only).
	if err := verifyBackupArtifact(absTarget, classID, semesterID); err != nil {
		if err == errBackupScope || err == errBackupSchema || err == errBackupSemester || err == errBackupRelations {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, err.Error())
			return
		}
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Berkas cadangan bukan SQLite yang valid")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi verifikasi backup")
		return
	}
	defer tx.Rollback()
	// Update status cadangan menjadi VERIFIED
	if _, err = tx.Exec(`
		UPDATE backup_records
		SET status = 'VERIFIED', verified_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`, req.BackupID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status backup")
		return
	}

	// Catat audit_logs
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"status":"VERIFIED","restore_performed":false,"checksum":%q,"semester_id":%v}`, actualChk, nullableInt(semesterID))
		correlationID := fmt.Sprintf("verify-backup-%d-%d", req.BackupID, time.Now().UnixNano())
		reason := strings.TrimSpace(*req.Reason)
		verifyEntry := audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "VERIFY_RESTORE_BACKUP",
			EntityType:    "BACKUP_RECORD",
			EntityID:      &req.BackupID,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}
		if semesterID.Valid {
			verifyEntry.SemesterID = &semesterID.Int64
		}
		if err := audit.Write(r.Context(), tx, verifyEntry); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit verifikasi backup")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit verifikasi backup")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyBackupRestore, restoreSubject, restoreSource, "SUCCESS")
	}

	// BE-011 (ADR-0008): verify-only — tidak mengganti database aktif.
	// artifact_ref dan path internal tidak dikembalikan ke client.
	// Respons menggema cakupan yang diverifikasi (tinjau cakupan).
	verifyResp := map[string]any{
		"backup_id":         req.BackupID,
		"class_id":          classID,
		"status":            "VERIFIED",
		"checksum":          actualChk,
		"restore_performed": false,
		"message":           "Berkas cadangan terverifikasi (checksum, format, schema, scope). Database aktif tidak diubah.",
	}
	if semesterID.Valid {
		verifyResp["semester_id"] = semesterID.Int64
	}
	common.WriteV1Success(w, http.StatusOK, verifyResp)
}

var (
	errBackupScope     = backupVerifyError{msg: "cakupan backup tidak cocok dengan kelas yang diminta"}
	errBackupSchema    = backupVerifyError{msg: "versi schema backup tidak kompatibel"}
	errBackupSemester  = backupVerifyError{msg: "cakupan semester backup tidak cocok"}
	errBackupRelations = backupVerifyError{msg: "integritas relasi backup rusak"}
)

type backupVerifyError struct{ msg string }

func (e backupVerifyError) Error() string { return e.msg }

// verifyBackupArtifact memeriksa format SQLite, kompatibilitas schema, metadata
// scope (kelas + semester bila dicatat), dan integritas relasi tanpa memodifikasi
// database aktif. Dibuka read-only.
func verifyBackupArtifact(absPath string, classID int64, semesterID sql.NullInt64) error {
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", absPath))
	if err != nil {
		return err
	}
	defer db.Close()
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='classes'`).Scan(&name); err != nil {
		return err
	}
	var quick string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&quick); err != nil || quick != "ok" {
		return fmt.Errorf("integrity check gagal")
	}
	var maxVer int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&maxVer); err == nil {
		if maxVer > database.LatestSchemaVersion {
			return errBackupSchema
		}
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM classes WHERE id = ?`, classID).Scan(&n); err != nil || n == 0 {
		return errBackupScope
	}
	if semesterID.Valid {
		var m int
		if err := db.QueryRow(`SELECT COUNT(*) FROM semesters WHERE id = ? AND class_id = ?`, semesterID.Int64, classID).Scan(&m); err != nil || m == 0 {
			return errBackupSemester
		}
	}
	var fkViolations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&fkViolations); err != nil || fkViolations != 0 {
		return errBackupRelations
	}
	return nil
}

// GetNotifications menangani GET /api/v1/notifications
func (c *AdminController) GetNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT nm.id, nm.class_id, nm.event_type, nm.entity_type, nm.entity_id,
		       nm.status, nm.payload_json, nm.scheduled_at, nm.sent_at, nm.created_at,
		       (SELECT COUNT(*) FROM notification_attempts na WHERE na.notification_message_id = nm.id) AS attempts
		FROM notification_messages nm
		WHERE (1=1)
	`
	var args []any

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveClassID.Valid {
		query += " AND nm.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	}

	if statusFilter != "" {
		query += " AND nm.status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY nm.created_at DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memuat antrean notifikasi: %v", err))
		return
	}
	defer rows.Close()

	notifications := []NotificationResponseItem{}
	for rows.Next() {
		var (
			id          int64
			classID     int64
			eventType   string
			entityType  sql.NullString
			entityID    sql.NullInt64
			status      string
			payloadJSON string
			scheduledAt common.DBTimestamp
			sentAt      common.DBTimestamp
			createdAt   common.DBTimestamp
			attempts    int
		)

		if err := rows.Scan(&id, &classID, &eventType, &entityType, &entityID, &status, &payloadJSON, &scheduledAt, &sentAt, &createdAt, &attempts); err == nil {
			var schedStr *string
			if scheduledAt.Valid {
				formatted := scheduledAt.Time.Format(time.RFC3339)
				schedStr = &formatted
			}
			var sentStr *string
			if sentAt.Valid {
				formatted := sentAt.Time.Format(time.RFC3339)
				sentStr = &formatted
			}
			var eType *string
			if entityType.Valid {
				eType = &entityType.String
			}
			var eID *int64
			if entityID.Valid {
				eID = &entityID.Int64
			}

			notifications = append(notifications, NotificationResponseItem{
				ID:           id,
				ClassID:      classID,
				EventType:    eventType,
				EntityType:   eType,
				EntityID:     eID,
				Status:       status,
				PayloadJSON:  payloadJSON,
				ScheduledAt:  schedStr,
				SentAt:       sentStr,
				CreatedAt:    createdAt.Time.Format(time.RFC3339),
				AttemptCount: attempts,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, notifications)
}

// RetryNotification menangani POST /api/v1/notifications/{id}/retry
func (c *AdminController) RetryNotification(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	notifIDStr := r.PathValue("id")
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil || notifID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID notifikasi tidak valid")
		return
	}

	var curStatus string
	var curClassID int64
	err = c.db.QueryRow(`SELECT status, class_id FROM notification_messages WHERE id = ?;`, notifID).Scan(&curStatus, &curClassID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pesan notifikasi tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi notifikasi")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya pengurus kelas terkait yang berwenang mencoba ulang pengiriman notifikasi")
			return
		}
	}

	// BE-010 state machine: hanya FAILED dan CANCELLED yang dapat diretry.
	if curStatus != "FAILED" && curStatus != "CANCELLED" {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, fmt.Sprintf("Hanya notifikasi dengan status FAILED atau CANCELLED yang dapat dicoba ulang (status saat ini: %s)", curStatus))
		return
	}

	// Update status notifikasi menjadi PENDING
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE notification_messages
		SET status = 'PENDING', scheduled_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('FAILED', 'CANCELLED');
	`, notifID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status notifikasi")
		return
	}

	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Status notifikasi telah berubah")
		return
	}

	// Catat audit_logs
	correlationID := fmt.Sprintf("retry-notif-%d", time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, curStatus)
		afterJSON := `{"status":"PENDING"}`
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &curClassID,
			Action:        "RETRY_NOTIFICATION",
			EntityType:    "NOTIFICATION_MESSAGE",
			EntityID:      &notifID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit retry notifikasi")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memproses transaksi")
		return
	}

	var scheduledAt string
	_ = c.db.QueryRow(`SELECT COALESCE(scheduled_at,'') FROM notification_messages WHERE id=?`, notifID).Scan(&scheduledAt)
	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id":              notifID,
		"status":          "PENDING",
		"scheduled_at":    scheduledAt,
		"retry_scheduled": true,
	})
}
