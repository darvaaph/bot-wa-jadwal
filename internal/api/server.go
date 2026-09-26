package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/web"
)

type Server struct {
	httpServer    *http.Server
	botClient     *bot.BotClient
	classManager  *schedule.ClassManager
	taskManager   *task.TaskManager
	taskRepo      *task.Repository
	academicRepo  *academic.Repository
	authService   *auth.Service
	secureCookies bool
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
	mux.HandleFunc("GET /api/status", s.authenticateIfConfigured(s.handleStatus))

	mux.HandleFunc("GET /api/academic/classes", s.authenticateIfConfigured(s.handleAcademicClasses))
	mux.HandleFunc("GET /api/academic/classes/{id}/courses", s.authenticateIfConfigured(s.handleAcademicCourses))
	mux.HandleFunc("POST /api/v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("GET /api/v1/auth/session", s.handleAuthSession)
	mux.HandleFunc("POST /api/v1/auth/switch-context", s.handleAuthSwitchContext)

	mux.HandleFunc("GET /api/classes", s.handleClasses)
	mux.HandleFunc("GET /api/schedule", s.handleSchedule)
	mux.HandleFunc("GET /api/tasks", s.handleGetTasks)
	mux.HandleFunc("POST /api/tasks", s.disableLegacyMutationWhenAuthConfigured(s.handleCreateTask))
	mux.HandleFunc("DELETE /api/tasks/{id}", s.disableLegacyMutationWhenAuthConfigured(s.handleDeleteTask))

	mux.HandleFunc("GET /api/v1/tasks", s.authenticateIfConfigured(s.handleListTasksV1))
	mux.HandleFunc("POST /api/v1/tasks", s.authenticateMutationIfConfigured(s.handleCreateTaskV1))
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.authenticateIfConfigured(s.handleGetTaskV1Detail))
	mux.HandleFunc("PUT /api/v1/tasks/{id}", s.authenticateMutationIfConfigured(s.handleUpdateTaskV1))
	mux.HandleFunc("POST /api/v1/tasks/{id}/publish", s.authenticateMutationIfConfigured(s.handlePublishTaskV1))
	mux.HandleFunc("POST /api/v1/tasks/{id}/reviews", s.authenticateMutationIfConfigured(s.handleReviewTaskV1))
	mux.HandleFunc("GET /api/v1/tasks/{id}/reviews", s.authenticateIfConfigured(s.handleListTaskReviewsV1))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}/complete", s.authenticateMutationIfConfigured(s.handleCompleteTaskV1))
	mux.HandleFunc("POST /api/v1/tasks/{id}/archive", s.authenticateMutationIfConfigured(s.handleArchiveTaskV1))
	mux.HandleFunc("POST /api/v1/tasks/{id}/unarchive", s.authenticateMutationIfConfigured(s.handleUnarchiveTaskV1))
	mux.HandleFunc("DELETE /api/v1/tasks/{id}", s.authenticateMutationIfConfigured(s.handleDeleteTaskV1))
	mux.HandleFunc("POST /api/v1/tasks/{id}/restore", s.authenticateMutationIfConfigured(s.handleRestoreTaskV1))

	mux.HandleFunc("GET /api/v1/materials", s.authenticateIfConfigured(s.handleListMaterialsV1))
	mux.HandleFunc("POST /api/v1/materials", s.authenticateMutationIfConfigured(s.handleCreateMaterialV1))
	mux.HandleFunc("PUT /api/v1/materials/{id}", s.authenticateMutationIfConfigured(s.handleUpdateMaterialV1))
	mux.HandleFunc("DELETE /api/v1/materials/{id}", s.authenticateMutationIfConfigured(s.handleDeleteMaterialV1))

	mux.HandleFunc("GET /api/portal/{slug}/summary", s.handlePortalSummary)
	mux.HandleFunc("GET /api/portal/{slug}/schedule", s.handlePortalSchedule)
	mux.HandleFunc("GET /api/portal/{slug}/tasks", s.handlePortalTasks)
	mux.HandleFunc("GET /api/portal/{slug}/changes", s.handlePortalChanges)
	mux.HandleFunc("GET /api/portal/{slug}/semesters", s.handlePortalSemesters)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Endpoint belum tersedia (dijadwalkan pada Fase B)",
		})
	})

	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	handler := s.corsMiddleware(s.recoveryMiddleware(mux))

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
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
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
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

// corsMiddleware hanya mengizinkan origin yang sama agar cookie sesi tidak dapat
// dipakai oleh situs lain. Dashboard produksi dilayani dari server ini.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if s.secureCookies {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			parsed, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(parsed.Host, r.Host) {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Origin tidak diizinkan"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")

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

func (s *Server) SetAuthService(service *auth.Service, secureCookies bool) {
	s.authService = service
	s.secureCookies = secureCookies
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
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() {
		filtered := classes[:0]
		for _, class := range classes {
			if principal.ClassID != nil && class.ID == *principal.ClassID {
				filtered = append(filtered, class)
			}
		}
		classes = filtered
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
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
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
