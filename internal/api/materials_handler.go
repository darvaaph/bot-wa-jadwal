package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/task"
)

func (s *Server) handleCreateMaterialV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul materi belum diinisialisasi"})
		return
	}
	var input task.CreateMaterialInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	if !hasPrincipal {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
		// Mode dev: izinkan tanpa sesi, pakai CreatedBy default.
		if input.CreatedByUserID <= 0 {
			input.CreatedByUserID = 1
		}
	} else {
		if !principal.IsSystemAdmin() {
			if principal.ClassID == nil || *principal.ClassID != input.ClassID {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			if input.CourseOfferingID != nil && principal.Role == "PJ" {
				if principal.CourseOfferingID == nil || *principal.CourseOfferingID != *input.CourseOfferingID {
					s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "PJ hanya dapat membuat materi pada offering penugasannya"})
					return
				}
			}
		}
		input.CreatedByUserID = principal.UserID
		input.Actor = task.ActorInfo{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID, ClassID: &input.ClassID}
	}
	created, err := s.taskRepo.CreateMaterial(r.Context(), input)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": created})
}

func (s *Server) handleListMaterialsV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul materi belum diinisialisasi"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("class_id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
		return
	}
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	var offeringID *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("course_offering_id")); raw != "" {
		parsed, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || parsed <= 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter course_offering_id tidak valid"})
			return
		}
		offeringID = &parsed
	}
	items, err := s.taskRepo.ListMaterialsByClass(r.Context(), classID, offeringID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar materi"})
		return
	}
	if items == nil {
		items = []task.Material{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleUpdateMaterialV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul materi belum diinisialisasi"})
		return
	}
	materialID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || materialID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID materi tidak valid"})
		return
	}
	var payload struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		URL         string  `json:"url"`
		Visibility  string  `json:"visibility"`
		Status      string  `json:"status"`
		Version     int     `json:"version"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if payload.Version < 1 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field version wajib diisi untuk deteksi konflik"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	var actor task.ActorInfo
	if hasPrincipal {
		actor = task.ActorInfo{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	// Pre-write scope check: resolve ownership before mutating.
	if hasPrincipal && !principal.IsSystemAdmin() {
		scope, scopeErr := s.taskRepo.GetMaterialScope(r.Context(), materialID)
		if scopeErr != nil {
			if errors.Is(scopeErr, task.ErrNotFound) {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Materi tidak ditemukan"})
			} else {
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan materi"})
			}
			return
		}
		if principal.ClassID == nil || *principal.ClassID != scope.ClassID {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if principal.Role == "PJ" && (scope.CourseOfferingID == nil || principal.CourseOfferingID == nil || *principal.CourseOfferingID != *scope.CourseOfferingID) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "PJ hanya dapat mengubah materi pada offering penugasannya"})
			return
		}
	}
	updated, updateErr := s.taskRepo.UpdateMaterial(r.Context(), materialID, task.UpdateMaterialInput{
		Title: payload.Title, Description: payload.Description, URL: payload.URL,
		Visibility: payload.Visibility, Status: payload.Status, ExpectedVersion: payload.Version,
	}, actor)
	if updateErr != nil {
		if errors.Is(updateErr, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Materi tidak ditemukan"})
			return
		}
		if errors.Is(updateErr, task.ErrVersionConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi materi sudah berubah, muat ulang sebelum menyimpan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": updateErr.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": updated})
}

func (s *Server) handleDeleteMaterialV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul materi belum diinisialisasi"})
		return
	}
	materialID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || materialID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID materi tidak valid"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	var actor task.ActorInfo
	if hasPrincipal {
		actor = task.ActorInfo{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID}
		// Pre-write scope check: resolve ownership before mutating.
		if !principal.IsSystemAdmin() {
			scope, scopeErr := s.taskRepo.GetMaterialScope(r.Context(), materialID)
			if scopeErr != nil {
				if errors.Is(scopeErr, task.ErrNotFound) {
					s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Materi tidak ditemukan"})
				} else {
					s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan materi"})
				}
				return
			}
			if principal.ClassID == nil || *principal.ClassID != scope.ClassID {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			if principal.Role == "PJ" && (scope.CourseOfferingID == nil || principal.CourseOfferingID == nil || *principal.CourseOfferingID != *scope.CourseOfferingID) {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "PJ hanya dapat menghapus materi pada offering penugasannya"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := s.taskRepo.SoftDeleteMaterial(r.Context(), materialID, actor); err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Materi tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menghapus materi"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Materi dihapus (soft-delete)"})
}
