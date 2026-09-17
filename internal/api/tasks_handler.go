package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TaskResponseItem adalah representasi tugas pada respons JSON API
type TaskResponseItem struct {
	ID        int    `json:"id"`
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
	IsDone    bool   `json:"is_done"`
}

// CreateTaskRequest adalah payload form pembuatan tugas dari Web Dashboard
type CreateTaskRequest struct {
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
}

const webCreatorJID = "web-dashboard"

// handleGetTasks menangani GET /api/tasks - daftar seluruh tugas aktif
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	if s.taskManager == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	_ = r.URL.Query().Get("class")

	items, err := s.taskManager.GetAllActiveTasks(time.Now())
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Format JSON tidak valid",
		})
		return
	}

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
		req.Matkul, req.Deskripsi, req.Deadline, webCreatorJID, time.Now(),
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