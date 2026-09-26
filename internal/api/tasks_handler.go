package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/task"
)

type TaskResponseItem struct {
	ID        int    `json:"id"`
	ClassID   string `json:"class_id,omitempty"`
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
	IsDone    bool   `json:"is_done"`
}

type CreateTaskRequest struct {
	ClassID   string `json:"class_id"`
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
}

const webCreatorJID = "web-dashboard"

// handleGetTasks menangani GET /api/tasks - daftar seluruh tugas aktif (dapat difilter per kelas)
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	if s.taskManager == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	classQuery := strings.TrimSpace(r.URL.Query().Get("class"))
	if classQuery == "" {
		classQuery = strings.TrimSpace(r.URL.Query().Get("class_id"))
	}

	var items []task.TaskItem
	var err error
	if classQuery != "" {
		items, err = s.taskManager.GetTasksByClassID(classQuery, time.Now())
	} else {
		items, err = s.taskManager.GetAllActiveTasks(time.Now())
	}
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal mengambil daftar tugas",
		})
		return
	}

	data := make([]TaskResponseItem, 0, len(items))
	for _, item := range items {
		data = append(data, TaskResponseItem{
			ID:        item.ID,
			ClassID:   item.ClassID,
			Matkul:    item.Matkul,
			Deskripsi: item.Deskripsi,
			Deadline:  item.Deadline,
			IsDone:    item.IsDone,
		})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   data,
	})
}

// handleCreateTask menangani POST /api/tasks - menyimpan catatan tugas baru
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if s.taskManager == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Format JSON tidak valid",
		})
		return
	}

	req.ClassID = strings.TrimSpace(req.ClassID)
	req.Matkul = strings.TrimSpace(req.Matkul)
	req.Deskripsi = strings.TrimSpace(req.Deskripsi)

	if req.Matkul == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Field matkul wajib diisi",
		})
		return
	}

	if req.Deskripsi == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Field deskripsi wajib diisi",
		})
		return
	}

	id, deadlineLabel, err := s.taskManager.AddWebTask(
		req.Matkul, req.Deskripsi, req.Deadline, webCreatorJID, time.Now(), req.ClassID,
	)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal menyimpan tugas",
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status": "success",
		"data": TaskResponseItem{
			ID:        int(id),
			ClassID:   req.ClassID,
			Matkul:    req.Matkul,
			Deskripsi: req.Deskripsi,
			Deadline:  deadlineLabel,
			IsDone:    false,
		},
	})
}

// handleDeleteTask menangani DELETE /api/tasks/{id} - menghapus tugas berdasarkan ID
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if s.taskManager == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	idStr := r.PathValue("id")
	taskID, err := strconv.Atoi(idStr)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "ID tugas tidak valid",
		})
		return
	}

	ok, err := s.taskManager.DeleteTaskByID(taskID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal menghapus tugas",
		})
		return
	}

	if !ok {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"status": "error",
			"error":  "Tugas tidak ditemukan",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Tugas berhasil dihapus",
	})
}

func (s *Server) handleListTasksV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	classID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("class_id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Parameter class_id tidak valid",
		})
		return
	}
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusFilter != "" {
		switch statusFilter {
		case "DRAFT", "PUBLISHED", "REVOKED":
		default:
			s.writeJSON(w, http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  "Parameter status tidak valid",
			})
			return
		}
	}

	items, err := s.taskRepo.ListTasksByClass(r.Context(), classID, statusFilter)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal mengambil daftar tugas",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   items,
	})
}

func (s *Server) handleCreateTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	var input task.CreateTaskInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Format JSON tidak valid",
		})
		return
	}

	if input.CourseOfferingID <= 0 || strings.TrimSpace(input.Title) == "" || strings.TrimSpace(input.DeadlineAt) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Field course_offering_id, title, dan deadline_at wajib diisi",
		})
		return
	}
	if principal, ok := principalFromRequest(r); ok {
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), input.CourseOfferingID)
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
		input.CreatedByUserID = principal.UserID
		input.Actor = actorInfo(principal, scope)
	}

	created, err := s.taskRepo.CreateTask(r.Context(), input)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal menyimpan tugas",
		})
		return
	}

	s.writeJSON(w, http.StatusCreated, map[string]any{
		"status": "success",
		"data":   created,
	})
}

func (s *Server) handleReviewTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "ID tugas tidak valid",
		})
		return
	}

	var payload struct {
		Decision                 string  `json:"decision"`
		Note                     *string `json:"note"`
		ReviewerRoleAssignmentID int64   `json:"reviewer_role_assignment_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Format JSON tidak valid",
		})
		return
	}

	payload.Decision = strings.ToUpper(strings.TrimSpace(payload.Decision))
	if payload.Decision == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Field decision wajib diisi",
		})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	var reviewScope task.AcademicScope
	if hasPrincipal {
		scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
		if err != nil {
			if errors.Is(err, task.ErrNotFound) {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa cakupan tugas"})
			return
		}
		if err := s.authService.RequireOfferingMutation(r.Context(), *principal, auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil || !principal.CanReviewClass(scope.ClassID) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		payload.ReviewerRoleAssignmentID = principal.RoleAssignmentID
		reviewScope = scope
	} else if payload.ReviewerRoleAssignmentID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field reviewer_role_assignment_id wajib diisi"})
		return
	}

	err = s.taskRepo.SubmitReview(r.Context(), task.ReviewTaskInput{
		TaskID:                   taskID,
		Decision:                 payload.Decision,
		Note:                     payload.Note,
		ReviewerRoleAssignmentID: payload.ReviewerRoleAssignmentID,
		Actor:                    actorInfo(principal, reviewScope),
	})
	if err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{
				"status": "error",
				"error":  "Tugas tidak ditemukan",
			})
			return
		}
		if errors.Is(err, task.ErrInvalidState) {
			s.writeJSON(w, http.StatusConflict, map[string]string{
				"status": "error",
				"error":  "Tugas REVOKED tidak dapat direview, publikasikan ulang",
			})
			return
		}
		if errors.Is(err, task.ErrValidation) {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{
				"status": "error",
				"error":  "Data minimum tugas belum lengkap untuk persetujuan",
			})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Gagal menyimpan review tugas",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Review tugas berhasil disimpan",
	})
}

func (s *Server) handleCompleteTaskV1(w http.ResponseWriter, r *http.Request) {
	if s.taskRepo == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	taskID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || taskID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "ID tugas tidak valid",
		})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	var completeScope task.AcademicScope
	if hasPrincipal {
		scope, err := s.taskRepo.GetTaskScope(r.Context(), taskID)
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
		completeScope = scope
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}

	if err := s.taskRepo.CompleteTask(r.Context(), taskID, actorInfo(principal, completeScope)); err != nil {
		if errors.Is(err, task.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{
				"status": "error",
				"error":  "Tugas tidak ditemukan",
			})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Gagal menyelesaikan tugas",
		})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"message": "Tugas ditandai selesai",
	})
}
