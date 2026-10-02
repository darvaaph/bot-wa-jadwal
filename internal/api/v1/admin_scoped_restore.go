package v1

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/backup"
)

func (c *AdminController) scopedRestoreInput(w http.ResponseWriter, r *http.Request) (int64, bool) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return 0, false
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang dapat memulihkan cadangan")
		return 0, false
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "ID cadangan tidak valid")
		return 0, false
	}
	return id, true
}

func scopedRestoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, backup.ErrNotFound):
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, err.Error())
	case errors.Is(err, backup.ErrInvalidInput):
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, err.Error())
	case errors.Is(err, backup.ErrMismatch):
		common.WriteV1Error(w, http.StatusUnprocessableEntity, "BACKUP_MISMATCH", "Checksum, versi, atau cakupan cadangan tidak cocok")
	case errors.Is(err, backup.ErrCrossClass):
		common.WriteV1Error(w, http.StatusConflict, "CROSS_CLASS_EVENT", err.Error())
	case errors.Is(err, backup.ErrStalePreview):
		common.WriteV1Error(w, http.StatusConflict, "STALE_PREVIEW", err.Error())
	default:
		common.WriteV1Error(w, http.StatusInternalServerError, "RESTORE_FAILED", "Pemulihan gagal; data aktif tidak diubah")
	}
}

func (c *AdminController) PreviewScopedRestore(w http.ResponseWriter, r *http.Request) {
	id, ok := c.scopedRestoreInput(w, r)
	if !ok {
		return
	}
	svc := backup.NewService(c.db, filepath.Join(c.getStorageDir(), "backups"))
	preview, err := svc.Preview(r.Context(), id)
	if err != nil {
		scopedRestoreError(w, err)
		return
	}
	common.WriteV1Success(w, http.StatusOK, preview)
}

func (c *AdminController) ExecuteScopedRestore(w http.ResponseWriter, r *http.Request) {
	id, ok := c.scopedRestoreInput(w, r)
	if !ok {
		return
	}
	var req struct {
		Reason       string `json:"reason"`
		PreviewToken string `json:"preview_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" || req.PreviewToken == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan dan token pratinjau wajib diisi")
		return
	}
	u, _ := common.GetAuthContext(r)
	svc := backup.NewService(c.db, filepath.Join(c.getStorageDir(), "backups"))
	prePointID, err := svc.Execute(r.Context(), id, u.UserID, req.Reason, req.PreviewToken)
	if err != nil {
		scopedRestoreError(w, err)
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"backup_id": id, "pre_restore_backup_id": prePointID, "status": "RESTORED"})
}
