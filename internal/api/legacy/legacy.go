package legacy

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/internal/util"
)

// TaskResponseItem adalah representasi tugas pada respons JSON API legacy
type TaskResponseItem struct {
	ID        int    `json:"id"`
	ClassID   string `json:"class_id,omitempty"`
	Matkul    string `json:"matkul"`
	Deskripsi string `json:"deskripsi"`
	Deadline  string `json:"deadline"`
	IsDone    bool   `json:"is_done"`
}

// ClassesResponse adalah format balasan untuk endpoint GET /api/classes
type ClassesResponse struct {
	Status string              `json:"status"`
	Data   ClassesDataResponse `json:"data"`
}

// ClassesDataResponse adalah payload data kelas yang terdaftar
type ClassesDataResponse struct {
	DefaultClass string   `json:"default_class"`
	TotalClasses int      `json:"total_classes"`
	Classes      []string `json:"classes"`
}

// ScheduleItemResponse merepresentasikan entri jadwal perkuliahan individual untuk respons API
type ScheduleItemResponse struct {
	Hari   string `json:"hari"`
	Jam    string `json:"jam"`
	Matkul string `json:"matkul"`
	Dosen  string `json:"dosen"`
	Ruang  string `json:"ruang"`
}

// ScheduleResponse adalah format balasan untuk endpoint GET /api/schedule
type ScheduleResponse struct {
	Status string                 `json:"status"`
	Class  string                 `json:"class"`
	Day    string                 `json:"day"`
	Data   []ScheduleItemResponse `json:"data"`
}

// Handler melayani seluruh endpoint shim legacy /api/*
type Handler struct {
	classManager *schedule.ClassManager
	taskManager  *task.TaskManager
	v1DB         *sql.DB
}

// NewHandler membuat instance handler rute legacy
func NewHandler(classManager *schedule.ClassManager, taskManager *task.TaskManager, v1DB *sql.DB) *Handler {
	return &Handler{
		classManager: classManager,
		taskManager:  taskManager,
		v1DB:         v1DB,
	}
}

// RegisterRoutes mendaftarkan seluruh endpoint legacy ke ServeMux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/classes", h.HandleClasses)
	mux.HandleFunc("GET /api/schedule", h.HandleSchedule)
	mux.HandleFunc("GET /api/tasks", h.HandleGetTasks)
	mux.HandleFunc("POST /api/tasks", h.HandleCreateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.HandleDeleteTask)
}

// HandleClasses menyajikan daftar seluruh kode kelas kanonikal beserta kelas default (Legacy Shim)
func (h *Handler) HandleClasses(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	defaultClass := ""
	classes := make([]string, 0)

	if h.classManager != nil {
		classes = h.classManager.ListClasses()
		if classes == nil {
			classes = make([]string, 0)
		}
		defaultClass = h.classManager.GetDefaultClassID()
	}

	resp := ClassesResponse{
		Status: "success",
		Data: ClassesDataResponse{
			DefaultClass: defaultClass,
			TotalClasses: len(classes),
			Classes:      classes,
		},
	}

	common.WriteJSON(w, http.StatusOK, resp)
}

// HandleSchedule menyajikan jadwal perkuliahan berdasarkan kelas dan filter hari (Legacy Shim)
func (h *Handler) HandleSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	if h.classManager == nil {
		common.WriteJSON(w, http.StatusNotFound, map[string]string{
			"status":  "error",
			"message": "Kelas tidak ditemukan",
		})
		return
	}

	classQuery := strings.TrimSpace(r.URL.Query().Get("class"))
	if classQuery == "" {
		classQuery = h.classManager.GetDefaultClassID()
	}

	cfg, ok := h.classManager.GetClass(classQuery)
	if !ok || cfg == nil {
		common.WriteJSON(w, http.StatusNotFound, map[string]string{
			"status":  "error",
			"message": "Kelas tidak ditemukan",
		})
		return
	}

	canonicalClassID := h.classManager.ResolveClassID(classQuery)
	if canonicalClassID == "" {
		canonicalClassID = classQuery
	}

	dayQuery := strings.TrimSpace(r.URL.Query().Get("day"))
	var targetDay string
	var respDay string

	if strings.EqualFold(dayQuery, "today") || strings.EqualFold(dayQuery, "hari ini") {
		targetDay = util.GetHariIndonesia(time.Now())
		respDay = targetDay
	} else if dayQuery != "" && !strings.EqualFold(dayQuery, "all") && !strings.EqualFold(dayQuery, "semua") {
		targetDay = normalizeDayInput(dayQuery)
		respDay = targetDay
	} else {
		respDay = "all"
	}

	items := make([]ScheduleItemResponse, 0)
	for _, item := range cfg.Jadwal {
		if targetDay != "" && !strings.EqualFold(item.Hari, targetDay) {
			continue
		}

		matkul := item.NamaMatkul
		if matkul == "" {
			if m, exists := cfg.MataKuliah[item.KodeMatkul]; exists && m != "" {
				matkul = m
			} else {
				matkul = item.KodeMatkul
			}
		}

		dosen := item.Dosen
		if dosen == "" {
			if d, exists := cfg.Dosen[item.InisialDosen]; exists && d != "" {
				dosen = d
			} else {
				dosen = item.InisialDosen
			}
		}

		items = append(items, ScheduleItemResponse{
			Hari:   item.Hari,
			Jam:    item.Jam,
			Matkul: matkul,
			Dosen:  dosen,
			Ruang:  item.Ruang,
		})
	}

	resp := ScheduleResponse{
		Status: "success",
		Class:  canonicalClassID,
		Day:    respDay,
		Data:   items,
	}

	common.WriteJSON(w, http.StatusOK, resp)
}

// HandleGetTasks menangani GET /api/tasks - daftar seluruh tugas aktif (dapat difilter per kelas, Legacy Shim)
func (h *Handler) HandleGetTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/v1/tasks>; rel="successor-version"`)

	classQuery := strings.TrimSpace(r.URL.Query().Get("class"))
	if classQuery == "" {
		classQuery = strings.TrimSpace(r.URL.Query().Get("class_id"))
	}

	if h.v1DB != nil {
		data, err := h.GetLegacyTaskViewFromV1(r.Context(), classQuery)
		if err != nil {
			common.WriteJSON(w, http.StatusInternalServerError, map[string]string{
				"status": "error",
				"error":  "Gagal mengambil daftar tugas",
			})
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]any{
			"status": "success",
			"data":   data,
		})
		return
	}

	if h.taskManager == nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "error",
			"error":  "Modul tugas belum diinisialisasi",
		})
		return
	}

	var items []task.TaskItem
	var err error
	if classQuery != "" {
		items, err = h.taskManager.GetTasksByClassID(classQuery, time.Now())
	} else {
		items, err = h.taskManager.GetAllActiveTasks(time.Now())
	}
	if err != nil {
		common.WriteJSON(w, http.StatusInternalServerError, map[string]string{
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

	common.WriteJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   data,
	})
}

// GetLegacyTaskViewFromV1 mengambil data tugas legacy dari database v1
func (h *Handler) GetLegacyTaskViewFromV1(ctx context.Context, classQuery string) ([]TaskResponseItem, error) {
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

	rows, err := h.v1DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]TaskResponseItem, 0)
	for rows.Next() {
		var item TaskResponseItem
		var deadline common.DBTimestamp
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

// HandleCreateTask menangani POST /api/tasks - menyimpan catatan tugas baru (Legacy Shim)
func (h *Handler) HandleCreateTask(w http.ResponseWriter, r *http.Request) {
	h.writeLegacyTaskGone(w)
}

// HandleDeleteTask menangani DELETE /api/tasks/{id} - menghapus tugas berdasarkan ID (Legacy Shim)
func (h *Handler) HandleDeleteTask(w http.ResponseWriter, r *http.Request) {
	h.writeLegacyTaskGone(w)
}

func (h *Handler) writeLegacyTaskGone(w http.ResponseWriter) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Link", `</api/v1/tasks>; rel="successor-version"`)
	common.WriteJSON(w, http.StatusGone, map[string]string{
		"status":      "error",
		"error":       "Endpoint tulis tugas legacy telah dihentikan; gunakan API v1 dengan autentikasi",
		"replacement": "/api/v1/tasks",
	})
}

func normalizeDayInput(input string) string {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "senin":
		return "Senin"
	case "selasa":
		return "Selasa"
	case "rabu":
		return "Rabu"
	case "kamis":
		return "Kamis"
	case "jumat", "jum'at":
		return "Jumat"
	case "sabtu":
		return "Sabtu"
	case "minggu":
		return "Minggu"
	default:
		return input
	}
}
