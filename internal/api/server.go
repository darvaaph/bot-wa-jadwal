package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/web"
)

type Server struct {
	httpServer   *http.Server
	botClient    *bot.BotClient
	classManager *schedule.ClassManager
	taskManager  *task.TaskManager
	taskRepo     *task.Repository
	academicRepo *academic.Repository
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

type StatusResponse struct {
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	BotConnection string    `json:"bot_connection"`
	TotalClasses  int       `json:"total_classes"`
	DefaultClass  string    `json:"default_class"`
	Classes       []string  `json:"classes"`
}

var startTime = time.Now()

const academicQueryTimeout = 3 * time.Second

func (s *Server) writeAcademicQueryError(w http.ResponseWriter, err error, message string) {
	status := http.StatusInternalServerError
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		message = "Waktu pemrosesan data akademik habis"
	} else if errors.Is(err, context.Canceled) {
		status = http.StatusRequestTimeout
		message = "Permintaan data akademik dibatalkan"
	}
	s.writeJSON(w, status, map[string]string{
		"status": "error",
		"error":  message,
	})
}

// NewServer membuat instance baru HTTP API server dengan middleware CORS dan logging
func NewServer(addr string, botClient *bot.BotClient, classManager *schedule.ClassManager, taskManager *task.TaskManager, academicRepo ...*academic.Repository) *Server {
	mux := http.NewServeMux()

	var repo *academic.Repository
	if len(academicRepo) > 0 {
		repo = academicRepo[0]
	}

	s := &Server{
		botClient:    botClient,
		classManager: classManager,
		taskManager:  taskManager,
		academicRepo: repo,
	}

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)

	mux.HandleFunc("GET /api/academic/classes", s.handleAcademicClasses)
	mux.HandleFunc("GET /api/academic/classes/{id}/courses", s.handleAcademicCourses)

	mux.HandleFunc("GET /api/classes", s.handleClasses)
	mux.HandleFunc("GET /api/schedule", s.handleSchedule)
	mux.HandleFunc("GET /api/tasks", s.handleGetTasks)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.handleDeleteTask)

	mux.HandleFunc("GET /api/v1/tasks", s.handleListTasksV1)
	mux.HandleFunc("POST /api/v1/tasks", s.handleCreateTaskV1)
	mux.HandleFunc("POST /api/v1/tasks/{id}/reviews", s.handleReviewTaskV1)
	mux.HandleFunc("PATCH /api/v1/tasks/{id}/complete", s.handleCompleteTaskV1)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Endpoint belum tersedia (dijadwalkan pada Fase B)",
		})
	})

	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	handler := s.corsMiddleware(s.recoveryMiddleware(mux))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	botStatus := "uninitialized"
	if s.botClient != nil {
		botStatus = s.botClient.Status()
	}

	totalClasses := 0
	defaultClass := ""
	var classes []string
	if s.classManager != nil {
		classes = s.classManager.ListClasses()
		totalClasses = len(classes)
		defaultClass = s.classManager.GetDefaultClassID()
	}

	resp := StatusResponse{
		Status:        "ok",
		Timestamp:     time.Now(),
		BotConnection: botStatus,
		TotalClasses:  totalClasses,
		DefaultClass:  defaultClass,
		Classes:       classes,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// corsMiddleware memungkinkan Web Dashboard (UI/UX) diakses lintas port saat masa pengembangan
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware menangani panic HTTP agar server web tidak crash
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("⚠️ [HTTP Panic] %v\n", rec)
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Terjadi kesalahan internal server",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	fmt.Printf("🌐 [Web API] Server REST API aktif di http://localhost%s\n", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("⚠️ [Web API] Server berhenti dengan pesan: %v\n", err)
		}
	}()
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) SetAcademicRepo(repo *academic.Repository) {
	s.academicRepo = repo
}

func (s *Server) SetTaskRepo(repo *task.Repository) {
	s.taskRepo = repo
}

func (s *Server) handleAcademicClasses(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"status": "success",
			"data":   []any{},
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), academicQueryTimeout)
	defer cancel()

	classes, err := s.academicRepo.GetClasses(ctx)
	if err != nil {
		log.Printf("academic classes query failed: %v", err)
		s.writeAcademicQueryError(w, err, "Gagal mengambil data kelas")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   classes,
	})
}

func (s *Server) handleAcademicCourses(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{
			"status": "success",
			"data":   []any{},
		})
		return
	}

	idStr := r.PathValue("id")
	classID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "error",
			"error":  "Parameter ID kelas tidak valid",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), academicQueryTimeout)
	defer cancel()

	courses, err := s.academicRepo.GetCoursesByClassID(ctx, classID)
	if err != nil {
		log.Printf("academic courses query failed for class ID %d: %v", classID, err)
		s.writeAcademicQueryError(w, err, "Gagal mengambil daftar mata kuliah")
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data":   courses,
	})
}
