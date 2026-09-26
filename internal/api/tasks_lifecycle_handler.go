package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/task"
)

func actorInfo(principal *auth.Principal, scope task.AcademicScope) task.ActorInfo {
	info := task.ActorInfo{CorrelationID: audit.NewCorrelationID()}
	if principal != nil {
		info.UserID = principal.UserID
		info.RoleAssignmentID = principal.RoleAssignmentID
		info.ClassID = &scope.ClassID
		info.SemesterID = &scope.SemesterID
	} else {
		info.ClassID = &scope.ClassID
		info.SemesterID = &scope.SemesterID
	}
	return info
}

func isKMPublisher(principal *auth.Principal, classID int64) bool {
	if principal == nil {
		return false
	}
	if principal.IsSystemAdmin() {
		return true
	}
	return principal.CanReviewClass(classID)
}

func (s *Server) handleGetTaskV1Detail(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			if _, ok := principalFromRequest(r); ok {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
		return
	}
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != scope.ClassID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	view, err := s.taskRepo.GetTaskByID(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil detail tugas"})
		return
	}
	reviews, err := s.taskRepo.ListReviews(r.Context(), taskID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil riwayat review"})
		return
	}
	if reviews == nil {
		reviews = []task.TaskReview{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": view, "reviews": reviews})
}

func (s *Server) handlePublishTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	var scope task.AcademicScope
	if hasPrincipal {
		scope, err = s.taskRepo.GetTaskScope(r.Context(), taskID)
		if err != nil {
			if errors.Is(err, task.ErrNotFound) {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
			return
		}
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	} else {
		// Mode dev tanpa auth (web-only): lewati scope check, pakai scope tugas.
		scope, err = s.taskRepo.GetTaskScope(r.Context(), taskID)
		if err != nil {
			if errors.Is(err, task.ErrNotFound) {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
				return
			}
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
			return
		}
	}
	published, err := s.taskRepo.PublishTask(r.Context(), taskID, actorInfo(principal, scope), isKMPublisher(principal, scope.ClassID))
	if err != nil {
		switch {
		case errors.Is(err, task.ErrNotFound):
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
		case errors.Is(err, task.ErrValidation):
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data minimum publikasi belum lengkap: judul, instruksi, deadline, dan tempat/tautan pengumpulan wajib diisi"})
		case errors.Is(err, task.ErrInvalidState):
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Status tugas tidak memungkinkan publikasi"})
		default:
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mempublikasikan tugas"})
		}
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": published})
}

func (s *Server) handleUpdateTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	var payload struct {
		Title          string  `json:"title"`
		Instructions   string  `json:"instructions"`
		DeadlineAt     string  `json:"deadline_at"`
		TaskType       string  `json:"task_type"`
		SubmissionText *string `json:"submission_text"`
		SubmissionURL  *string `json:"submission_url"`
		Version        int     `json:"version"`
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
	scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			if hasPrincipal {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			} else {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			}
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
		return
	}
	if hasPrincipal {
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	updated, err := s.taskRepo.UpdateTask(r.Context(), taskID, task.UpdateTaskInput{
		Title: payload.Title, Instructions: payload.Instructions, DeadlineAt: payload.DeadlineAt,
		TaskType: payload.TaskType, SubmissionText: payload.SubmissionText, SubmissionURL: payload.SubmissionURL,
		ExpectedVersion: payload.Version,
	}, actorInfo(principal, scope), isKMPublisher(principal, scope.ClassID))
	if err != nil {
		switch {
		case errors.Is(err, task.ErrNotFound):
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
		case errors.Is(err, task.ErrVersionConflict):
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi data sudah berubah, muat ulang sebelum menyimpan"})
		case errors.Is(err, task.ErrValidation):
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format deadline_at tidak valid (gunakan RFC3339 UTC)"})
		case errors.Is(err, task.ErrInvalidState):
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Tugas REVOKED tidak dapat diubah, publikasikan ulang"})
		default:
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengubah tugas"})
		}
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": updated})
}

func (s *Server) handleArchiveTaskV1(w http.ResponseWriter, r *http.Request) {
	s.handleArchiveToggleV1(w, r, true)
}

func (s *Server) handleUnarchiveTaskV1(w http.ResponseWriter, r *http.Request) {
	s.handleArchiveToggleV1(w, r, false)
}

func (s *Server) handleArchiveToggleV1(w http.ResponseWriter, r *http.Request, archive bool) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			if hasPrincipal {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			} else {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			}
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
		return
	}
	if hasPrincipal {
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	var result *task.Task
	if archive {
		result, err = s.taskRepo.ArchiveTask(r.Context(), taskID, actorInfo(principal, scope))
	} else {
		result, err = s.taskRepo.UnarchiveTask(r.Context(), taskID, actorInfo(principal, scope))
	}
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengubah status arsip"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": result})
}

func (s *Server) handleDeleteTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			if hasPrincipal {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			} else {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			}
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
		return
	}
	if hasPrincipal {
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := s.taskRepo.SoftDeleteTask(r.Context(), taskID, actorInfo(principal, scope)); err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menghapus tugas"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Tugas dihapus (soft-delete)"})
}

func (s *Server) handleRestoreTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if strings.TrimSpace(payload.Reason) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field reason wajib diisi untuk pemulihan"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	if !hasPrincipal && s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	// Resolve the true scope including soft-deleted rows; GetTaskScope hides them.
	scope, err := s.taskRepo.GetTaskScopeIncludingDeleted(r.Context(), taskID)
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas terhapus tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
		return
	}
	if hasPrincipal {
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	}
	restored, err := s.taskRepo.RestoreTask(r.Context(), taskID, actorInfo(principal, scope), strings.TrimSpace(payload.Reason))
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas terhapus tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": restored})
}

func (s *Server) handleListTaskReviewsV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Modul tugas belum diinisialisasi"})
		return
	}
	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID tugas tidak valid"})
		return
	}
	if principal, ok := principalFromRequest(r); ok {
		scope, scopeErr := s.taskRepo.GetTaskScope(r.Context(), taskID)
		if scopeErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
		if !principal.IsSystemAdmin() && (principal.ClassID == nil || *principal.ClassID != scope.ClassID) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	} else {
		if _, scopeErr := s.taskRepo.GetTaskScope(r.Context(), taskID); scopeErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Tugas tidak ditemukan"})
			return
		}
	}
	reviews, err := s.taskRepo.ListReviews(r.Context(), taskID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil riwayat review"})
		return
	}
	if reviews == nil {
		reviews = []task.TaskReview{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": reviews})
}
