package api

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
	"strings"
	"time"
)

// BackupRequest merepresentasikan payload pembuatan backup on-demand
type BackupRequest struct {
	ClassSlug *string `json:"class_slug,omitempty"`
	Reason    *string `json:"reason,omitempty"`
}

// BackupResponseItem merepresentasikan catatan cadangan basis data
type BackupResponseItem struct {
	ID          int64   `json:"id"`
	ClassID     *int64  `json:"class_id,omitempty"`
	ArtifactRef string  `json:"artifact_ref"`
	Checksum    string  `json:"checksum"`
	Status      string  `json:"status"` // CREATING, READY, RESTORING, VERIFIED, FAILED
	Reason      *string `json:"reason,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// handleCreateBackup menangani POST /api/v1/backups
func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang memicu pencadangan database")
		return
	}

	var req BackupRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	backupDir := filepath.Join(s.getStorageDir(), "backups")
	_ = os.MkdirAll(backupDir, 0755)

	timestamp := time.Now().Format("20060102_150405")
	backupFileName := fmt.Sprintf("backup_v1_%s.db", timestamp)
	backupFilePath := filepath.Join(backupDir, backupFileName)

	// Lakukan VACUUM INTO untuk membuat snapshot SQLite secara aman tanpa mengunci penulisan
	_, err := s.v1DB.Exec(fmt.Sprintf("VACUUM INTO '%s';", filepath.ToSlash(backupFilePath)))
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", fmt.Sprintf("Gagal membuat snapshot database: %v", err))
		return
	}

	// Hitung checksum berkas hasil backup
	f, err := os.Open(backupFilePath)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Gagal membaca berkas hasil cadangan")
		return
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "BACKUP_FAILED", "Gagal menghitung checksum cadangan")
		return
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))

	var classID sql.NullInt64
	if u.ActiveClassID.Valid {
		classID = u.ActiveClassID
	}

	res, err := s.v1DB.Exec(`
		INSERT INTO backup_records (
			class_id, artifact_ref, checksum, status, created_by_user_id, reason, created_at
		) VALUES (?, ?, ?, 'READY', ?, ?, CURRENT_TIMESTAMP);
	`, classID, backupFilePath, checksum, u.UserID, req.Reason)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan rekam cadangan: %v", err))
		return
	}

	backupID, _ := res.LastInsertId()

	// Catat audit_logs
	_, _ = s.v1DB.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, after_json)
		VALUES (?, ?, ?, 'CREATE_BACKUP', 'BACKUP_RECORD', ?, ?);
	`, classID, u.UserID, u.ActiveAssignmentID, backupID,
		fmt.Sprintf(`{"artifact_ref":%q,"checksum":%q}`, backupFilePath, checksum),
	)

	s.writeV1Success(w, http.StatusCreated, BackupResponseItem{
		ID:          backupID,
		ArtifactRef: backupFilePath,
		Checksum:    checksum,
		Status:      "READY",
		Reason:      req.Reason,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
}

// RestoreRequest merepresentasikan payload permintaan pemulihan database
type RestoreRequest struct {
	BackupID int64   `json:"backup_id"`
	Reason   *string `json:"reason,omitempty"`
}

// handleRestoreBackup menangani POST /api/v1/restores
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya System Admin yang berwenang melakukan verifikasi dan pemulihan database")
		return
	}

	var req RestoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.BackupID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID cadangan (backup_id) wajib diisi")
		return
	}

	var (
		artifactRef string
		expectedChk string
		curStatus   string
	)

	err := s.v1DB.QueryRow(`
		SELECT artifact_ref, checksum, status
		FROM backup_records
		WHERE id = ?;
	`, req.BackupID).Scan(&artifactRef, &expectedChk, &curStatus)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Catatan cadangan tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi catatan cadangan")
		return
	}

	// Cek fisik file di disk
	f, err := os.Open(artifactRef)
	if err != nil {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, fmt.Sprintf("Berkas fisik cadangan tidak ditemukan di disk: %s", artifactRef))
		return
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "IO_ERROR", "Gagal membaca berkas cadangan untuk validasi")
		return
	}
	actualChk := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(actualChk, expectedChk) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, "CHECKSUM_MISMATCH", "Integritas berkas cadangan rusak: checksum tidak cocok")
		return
	}

	// Update status cadangan menjadi VERIFIED
	_, _ = s.v1DB.Exec(`
		UPDATE backup_records
		SET status = 'VERIFIED', verified_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`, req.BackupID)

	// Catat audit_logs
	_, _ = s.v1DB.Exec(`
		INSERT INTO audit_logs (actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, after_json)
		VALUES (?, ?, 'VERIFY_RESTORE_BACKUP', 'BACKUP_RECORD', ?, ?);
	`, u.UserID, u.ActiveAssignmentID, req.BackupID,
		fmt.Sprintf(`{"artifact_ref":%q,"status":"VERIFIED"}`, artifactRef),
	)

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"backup_id":    req.BackupID,
		"status":       "VERIFIED",
		"artifact_ref": artifactRef,
		"checksum":     actualChk,
		"message":      "Integritas berkas cadangan terverifikasi aman dan siap digunakan untuk pemulihan operasional",
	})
}
