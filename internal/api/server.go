package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/api/legacy"
	"bot-jadwal/internal/api/middleware"
	v1 "bot-jadwal/internal/api/v1"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/maintenance"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/ratelimit"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
)

// Server mengelola HTTP REST API untuk Web Admin Dashboard dan API v1
type Server struct {
	httpServer         *http.Server
	botClient          *bot.BotClient
	classManager       *schedule.ClassManager
	taskManager        *task.TaskManager
	v1DB               *sql.DB
	portalService      *portal.Service
	legacyHandler      *legacy.Handler
	secManager         *middleware.SecurityManager
	authManager        *middleware.AuthManager
	rlManager          *middleware.RateLimitManager
	taskController     *v1.TaskController
	scheduleController *v1.ScheduleController
	masterController   *v1.MasterController
	portalController   *v1.PortalController
	authController     *v1.AuthController
	academicController *v1.AcademicController
	adminController    *v1.AdminController
	storageDir         string
	secureCookies      bool
	// BE-013/BE-014: konfigurasi security eksplisit.
	env               string
	authHashKey       []byte
	allowedOrigins    []string
	trustedProxyCIDRs []string
	publicBaseURL     string
	// BE-012: rate limiter terpusat dan persisten (nil = belum di-wire).
	limiter *ratelimit.Service
}

// HealthResponse adalah payload untuk endpoint /api/health
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
}

// StatusResponse adalah payload telemetri untuk endpoint /api/status
type StatusResponse struct {
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
	BotConnection string    `json:"bot_connection"`
	TotalClasses  int       `json:"total_classes"`
	DefaultClass  string    `json:"default_class"`
	Classes       []string  `json:"classes"`
	V1            string    `json:"v1,omitempty"`
}

var startTime = time.Now()

// NewServer membuat instance baru HTTP API server dengan middleware CORS dan logging
func NewServer(addr string, botClient *bot.BotClient, classManager *schedule.ClassManager, taskManager *task.TaskManager, v1DB ...*sql.DB) *Server {
	mux := http.NewServeMux()

	s := &Server{
		botClient:    botClient,
		classManager: classManager,
		taskManager:  taskManager,
	}
	s.secManager = middleware.NewSecurityManager(middleware.SecurityOptions{})
	s.rlManager = middleware.NewRateLimitManager(nil)
	if len(v1DB) > 0 && v1DB[0] != nil {
		s.v1DB = v1DB[0]
		s.portalService = portal.NewService(v1DB[0])
		s.authManager = middleware.NewAuthManager(v1DB[0])
		// Limiter default (kunci efemeral hingga SetSecurityOptions memberi
		// key eksplisit). Tabel security_attempts tersedia via migrasi 008.
		s.buildLimiter()
	} else {
		s.authManager = middleware.NewAuthManager(nil)
	}
	s.legacyHandler = legacy.NewHandler(classManager, taskManager, s.v1DB)
	s.taskController = v1.NewTaskController(s.v1DB)
	s.scheduleController = v1.NewScheduleController(s.v1DB)
	s.masterController = v1.NewMasterController(s.v1DB)
	s.portalController = v1.NewPortalController(s.v1DB, s.portalService, s.rlManager, s.secManager)
	s.authController = v1.NewAuthController(s.v1DB, s.secManager, s.rlManager, s.portalService)
	s.academicController = v1.NewAcademicController(s.v1DB)
	var botProvider v1.BotStatusProvider
	if s.botClient != nil {
		botProvider = s.botClient
	}
	s.adminController = v1.NewAdminController(s.v1DB, botProvider, s.secManager, s.rlManager, s.getStorageDir)

	// Registrasi seluruh rute (legacy shim, API v1, static web assets)
	s.registerRoutes(mux)

	handler := s.corsMiddleware(s.recoveryMiddleware(s.maintenanceMiddleware(mux)))

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return s
}

func (s *Server) maintenanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions ||
			strings.HasSuffix(r.URL.Path, "/restore-execute") {
			next.ServeHTTP(w, r)
			return
		}
		release, ok := maintenance.EnterMutation()
		if !ok {
			w.Header().Set("Retry-After", "5")
			s.writeV1Error(w, http.StatusServiceUnavailable, "MAINTENANCE", "Pemulihan data sedang berlangsung. Coba lagi beberapa saat.")
			return
		}
		defer release()
		next.ServeHTTP(w, r)
	})
}

// SetRecoverySender replaces the WhatsApp delivery adapter used by password
// recovery. Production uses BotClient; tests may inject a deterministic fake.
func (s *Server) SetRecoverySender(sender v1.RecoverySender) {
	if s.authController != nil {
		s.authController.SetRecoverySender(sender)
	}
}

func (s *Server) configureRecovery() {
	if s.authController == nil || s.v1DB == nil || len(s.authHashKey) < 32 {
		return
	}
	service, err := auth.NewService(s.v1DB, auth.Config{HashKey: s.authHashKey})
	if err != nil {
		fmt.Printf("[Auth] gagal menginisialisasi pemulihan kata sandi: %v\n", err)
		return
	}
	s.authController.ConfigureRecovery(service, s.botClient, s.publicBaseURL)
}

// Start menjalankan HTTP Server di background goroutine
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.httpServer.Addr)
	if err != nil {
		return fmt.Errorf("gagal mendengarkan pada %s: %w", s.httpServer.Addr, err)
	}
	fmt.Printf("🌐 [Web API] Server REST API aktif di http://localhost%s\n", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("⚠️ [Web API] Server berhenti dengan pesan: %v\n", err)
		}
	}()
	return nil
}

// Shutdown mematikan HTTP server secara anggun (graceful shutdown)
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

// handleHealth mengembalikan sinyal hidup (health check) server dengan shim deprecation header
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	resp := HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// handleStatus mengembalikan telemetri bot dan sistem kelas dengan shim deprecation header
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
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
		V1:            "/api/v1/portal/:slug/summary",
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// writeJSON adalah helper pengirim respon JSON seragam
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(data)
}

// SetStorageDir menentukan direktori penyimpanan berkas runtime/backup (berguna untuk pengujian terisolasi)
func (s *Server) SetStorageDir(dir string) {
	s.storageDir = dir
}

func (s *Server) getStorageDir() string {
	if s.storageDir != "" {
		return s.storageDir
	}
	return "storage"
}
