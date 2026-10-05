package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/backup"
)

type backupRequestInput struct {
	ClassSlug  string `json:"class_slug"`
	SemesterID *int64 `json:"semester_id,omitempty"`
	Reason     string `json:"reason"`
}

func (c *AdminController) CreateBackupRequest(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" {
		common.WriteV1Error(w, 403, common.CodeForbidden, "Hanya KM dapat mengajukan cadangan kelas")
		return
	}
	var in backupRequestInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(strings.TrimSpace(in.Reason)) < 5 {
		common.WriteV1Error(w, 422, common.CodeValidation, "Kelas dan alasan minimal lima karakter wajib diisi")
		return
	}
	var classID int64
	if err := c.db.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug=?`, strings.TrimSpace(in.ClassSlug)).Scan(&classID); err != nil || !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
		common.WriteV1Error(w, 404, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if in.SemesterID != nil {
		var belongs bool
		if err := c.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM semesters WHERE id=? AND class_id=?)`, *in.SemesterID, classID).Scan(&belongs); err != nil || !belongs {
			common.WriteV1Error(w, 404, common.CodeNotFound, "Semester tidak ditemukan di kelas ini")
			return
		}
	}
	tx, err := c.db.BeginTx(r.Context(), nil)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membuat permintaan")
		return
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(r.Context(), `INSERT INTO backup_requests (class_id, semester_id, requested_by_user_id, reason)
		VALUES (?, ?, ?, ?) RETURNING id`, classID, in.SemesterID, u.UserID, strings.TrimSpace(in.Reason)).Scan(&id)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal menyimpan permintaan")
		return
	}
	uid := u.UserID
	if err := audit.Write(r.Context(), tx, audit.Entry{Actor: audit.Actor{Type: "USER", UserID: &uid}, ClassID: &classID,
		SemesterID: in.SemesterID, Action: "REQUEST_BACKUP", EntityType: "BACKUP_REQUEST", EntityID: &id,
		Reason: strings.TrimSpace(in.Reason), CorrelationID: fmt.Sprintf("backup-request-%d", id)}); err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal mencatat audit permintaan")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal menyimpan permintaan")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "status": "PENDING", "class_id": classID, "semester_id": in.SemesterID})
}

func (c *AdminController) ListBackupRequests(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	query := `SELECT br.id, br.class_id, cl.slug, br.semester_id, br.reason, br.status,
		br.requested_by_user_id, br.backup_id, br.created_at, br.decided_at
		FROM backup_requests br JOIN classes cl ON cl.id=br.class_id`
	args := []any{}
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, 403, common.CodeForbidden, "Konteks kelas diperlukan")
			return
		}
		query += ` WHERE br.class_id=?`
		args = append(args, u.ActiveClassID.Int64)
	} else if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, 403, common.CodeForbidden, "Tidak diizinkan")
		return
	}
	query += ` ORDER BY br.id DESC LIMIT 100`
	rows, err := c.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat permintaan")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, classID, requestedBy int64
		var slug, reason, status, created string
		var semesterID, backupID sql.NullInt64
		var decided sql.NullString
		if err := rows.Scan(&id, &classID, &slug, &semesterID, &reason, &status, &requestedBy, &backupID, &created, &decided); err != nil {
			continue
		}
		item := map[string]any{"id": id, "class_id": classID, "class_slug": slug, "reason": reason, "status": status, "requested_by_user_id": requestedBy, "created_at": created}
		if semesterID.Valid {
			item["semester_id"] = semesterID.Int64
		}
		if backupID.Valid {
			item["backup_id"] = backupID.Int64
		}
		if decided.Valid {
			item["decided_at"] = decided.String
		}
		out = append(out, item)
	}
	common.WriteV1Success(w, 200, out)
}

func (c *AdminController) ExecuteBackupRequest(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, 403, common.CodeForbidden, "Hanya System Admin dapat mengeksekusi cadangan")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, 400, common.CodeValidation, "ID permintaan tidak valid")
		return
	}
	var classID int64
	var semesterID sql.NullInt64
	var reason, status string
	err = c.db.QueryRowContext(r.Context(), `SELECT class_id,semester_id,reason,status FROM backup_requests WHERE id=?`, id).
		Scan(&classID, &semesterID, &reason, &status)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, 404, common.CodeNotFound, "Permintaan tidak ditemukan")
		return
	}
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat permintaan")
		return
	}
	if status != "PENDING" {
		common.WriteV1Error(w, 409, common.CodeVersionConflict, "Permintaan sudah diproses")
		return
	}
	res, err := c.db.ExecContext(r.Context(), `UPDATE backup_requests SET status='PROCESSING' WHERE id=? AND status='PENDING'`, id)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memulai eksekusi")
		return
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		common.WriteV1Error(w, 409, common.CodeVersionConflict, "Permintaan sudah diproses")
		return
	}
	var sem *int64
	if semesterID.Valid {
		sem = &semesterID.Int64
	}
	svc := backup.NewService(c.db, filepath.Join(c.getStorageDir(), "backups"))
	record, err := svc.Create(r.Context(), classID, sem, u.UserID, reason)
	if err != nil {
		_, _ = c.db.ExecContext(r.Context(), `UPDATE backup_requests SET status='FAILED', decided_at=? WHERE id=?`, time.Now().UTC().Format(time.RFC3339Nano), id)
		common.WriteV1Error(w, 500, "BACKUP_FAILED", "Gagal membuat cadangan akademik")
		return
	}
	_, err = c.db.ExecContext(r.Context(), `UPDATE backup_requests SET status='EXECUTED', executed_by_user_id=?, backup_id=?, decided_at=? WHERE id=? AND status='PROCESSING'`,
		u.UserID, record.ID, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Cadangan dibuat tetapi status permintaan gagal diperbarui")
		return
	}
	common.WriteV1Success(w, 200, map[string]any{"id": id, "status": "EXECUTED", "backup_id": record.ID, "checksum": record.Checksum})
}
