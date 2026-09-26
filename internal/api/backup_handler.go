package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if s.backupService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan backup belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat membuat backup"})
		return
	}
	var payload struct {
		ClassID    int64  `json:"class_id"`
		SemesterID *int64 `json:"semester_id"`
		Reason     string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil || payload.ClassID <= 0 || strings.TrimSpace(payload.Reason) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field class_id dan reason wajib diisi"})
		return
	}
	var userID int64 = 1
	if ok {
		userID = principal.UserID
	}
	rec, err := s.backupService.Create(r.Context(), payload.ClassID, payload.SemesterID, userID, payload.Reason)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": rec})
}

func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	if s.backupService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan backup belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat melihat backup"})
		return
	}
	var classID *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("class_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
			return
		}
		classID = &parsed
	}
	items, err := s.backupService.List(r.Context(), classID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar backup"})
		return
	}
	if items == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": []any{}})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if s.backupService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan backup belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat menjalankan restore"})
		return
	}
	backupID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || backupID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID backup tidak valid"})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if strings.TrimSpace(payload.Reason) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field reason wajib diisi untuk restore"})
		return
	}
	var userID int64 = 1
	if ok {
		userID = principal.UserID
	}
	if err := s.backupService.Restore(r.Context(), backupID, userID, payload.Reason); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Restore selesai dengan titik pemulihan pra-restore"})
}
