package api

import (
	"context"
	"net/http"

	"bot-jadwal/internal/api/legacy"
)

// TaskResponseItem adalah representasi tugas pada respons JSON API legacy
type TaskResponseItem = legacy.TaskResponseItem

// handleGetTasks menangani GET /api/tasks - daftar seluruh tugas aktif (dapat difilter per kelas, Legacy Shim)
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	if s.legacyHandler != nil {
		s.legacyHandler.HandleGetTasks(w, r)
		return
	}
	legacy.NewHandler(s.classManager, s.taskManager, s.v1DB).HandleGetTasks(w, r)
}

func (s *Server) getLegacyTaskViewFromV1(ctx context.Context, classQuery string) ([]TaskResponseItem, error) {
	h := s.legacyHandler
	if h == nil {
		h = legacy.NewHandler(s.classManager, s.taskManager, s.v1DB)
	}
	return h.GetLegacyTaskViewFromV1(ctx, classQuery)
}

// handleCreateTask menangani POST /api/tasks - menyimpan catatan tugas baru (Legacy Shim)
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if s.legacyHandler != nil {
		s.legacyHandler.HandleCreateTask(w, r)
		return
	}
	legacy.NewHandler(s.classManager, s.taskManager, s.v1DB).HandleCreateTask(w, r)
}

// handleDeleteTask menangani DELETE /api/tasks/{id} - menghapus tugas berdasarkan ID (Legacy Shim)
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	if s.legacyHandler != nil {
		s.legacyHandler.HandleDeleteTask(w, r)
		return
	}
	legacy.NewHandler(s.classManager, s.taskManager, s.v1DB).HandleDeleteTask(w, r)
}
