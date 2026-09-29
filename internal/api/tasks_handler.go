package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/task"
)

// TaskResponseItem adalah representasi tugas pada respons JSON API
type TaskResponseItem struct {
	ID        int    `json:"id"`
	ClassID   string `json:"class_id,omitempty"`
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
	IsDone    bool   `json:"is_done"`
}

// handleGetTasks menangani GET /api/tasks - daftar seluruh tugas aktif (dapat difilter per kelas, Legacy Shim)
func (s *Server) handleGetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/v1/tasks>; rel="successor-version"`)

	classQuery := strings.TrimSpace(r.URL.Query().Get("class"))
	if classQuery == "" {
		classQuery = strings.TrimSpace(r.URL.Query().Get("class_id"))
	}

	if s.v1DB != nil {
		data, err := s.getLegacyTaskViewFromV1(r.Context(), classQuery)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{
				"status": "error",
				"error":  "Gagal mengambil daftar tugas",
			})
			return
		}
		s.writeJSON(w, http.StatusOK, map[string]any{
			"status": "success",
			"data":   data,
		})
		return
	}

	if s.taskManager == nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
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

func (s *Server) getLegacyTaskViewFromV1(ctx context.Context, classQuery string) ([]TaskResponseItem, error) {
	query := `
		SELECT t.id, cl.code, co.display_name, t.title, t.deadline_at
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		JOIN classes cl ON cl.id = sem.class_id
		WHERE t.publication_status = 'PUBLISHED'
		  AND t.completed_at IS NULL
		  AND t.archived_at IS NULL
		  AND t.deleted_at IS NULL`
	args := []any{}
	if classQuery != "" {
		query += ` AND (cl.code = ? OR cl.slug = ?)`
		args = append(args, classQuery, classQuery)
	}
	query += ` ORDER BY t.deadline_at ASC, t.id ASC`

	rows, err := s.v1DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskResponseItem, 0)
	for rows.Next() {
		var item TaskResponseItem
		var deadline dbTimestamp
		if err := rows.Scan(&item.ID, &item.ClassID, &item.Matkul, &item.Deskripsi, &deadline); err != nil {
			return nil, err
		}
		item.Deadline = deadline.Time.Format(time.RFC3339)
		item.IsDone = false
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// handleCreateTask menangani POST /api/tasks - menyimpan catatan tugas baru (Legacy Shim)
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	s.writeLegacyTaskGone(w)
}

// handleDeleteTask menangani DELETE /api/tasks/{id} - menghapus tugas berdasarkan ID (Legacy Shim)
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request) {
	s.writeLegacyTaskGone(w)
}

func (s *Server) writeLegacyTaskGone(w http.ResponseWriter) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/v1/tasks>; rel="successor-version"`)
	s.writeJSON(w, http.StatusGone, map[string]string{
		"status":      "error",
		"error":       "Endpoint tulis tugas legacy telah dihentikan; gunakan API v1 dengan autentikasi",
		"replacement": "/api/v1/tasks",
	})
}
