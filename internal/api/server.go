package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/web"
)

// Server mengelola HTTP REST API untuk Web Admin Dashboard dan API v1
type Server struct {
	httpServer    *http.Server
	botClient     *bot.BotClient
	classManager  *schedule.ClassManager
	taskManager   *task.TaskManager
	v1DB          *sql.DB
	portalService *portal.Service
	storageDir    string
	secureCookies bool
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
	if len(v1DB) > 0 && v1DB[0] != nil {
		s.v1DB = v1DB[0]
		s.portalService = portal.NewService(v1DB[0])
	}

	// Registrasi Route API Scaffolding (Legacy Shim dengan header Deprecation: true)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.handleStatus)

	// Registrasi Route Jadwal & Kelas Legacy
	mux.HandleFunc("GET /api/classes", s.handleClasses)
	mux.HandleFunc("GET /api/schedule", s.handleSchedule)
	// Registrasi Route API Tugas Legacy
	mux.HandleFunc("GET /api/tasks", s.handleGetTasks)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", s.handleDeleteTask)

	// Route API v1 (Lapis L2 & Fitur Lanjutan)

	// 1. Auth & Konteks (§1)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.RequireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/me", s.RequireAuth(s.handleGetMe))
	mux.HandleFunc("POST /api/v1/auth/switch-context", s.RequireAuth(s.handleSwitchContext))
	mux.HandleFunc("GET /api/v1/classes", s.handleGetV1ClassesAccess)
	mux.HandleFunc("PATCH /api/v1/classes/{slug}", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handlePatchV1ClassStatus)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/portal-code/rotate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRotatePortalCode)))
	mux.HandleFunc("POST /api/v1/invitations", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateInvitation)))
	mux.HandleFunc("POST /api/v1/invitations/accept", s.handleAcceptInvitation)

	// 2. Portal Mahasiswa (§2)
	mux.HandleFunc("POST /api/v1/portal/{slug}/session", s.handleCreatePortalSession)
	mux.HandleFunc("GET /api/v1/portal/{slug}/summary", s.handlePortalSummary)
	mux.HandleFunc("GET /api/v1/portal/{slug}/schedule", s.handlePortalSchedule)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks", s.handlePortalTasks)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks/{id}", s.handlePortalTaskDetail)
	mux.HandleFunc("GET /api/v1/portal/{slug}/changes", s.handlePortalChanges)
	mux.HandleFunc("GET /api/v1/portal/{slug}/materials", s.handlePortalMaterials)

	// 3. Semester & Offering (§3)
	mux.HandleFunc("GET /api/v1/classes/{slug}/semesters", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetClassSemesters)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateClassSemester)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters/{id}/activate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleActivateSemester)))
	mux.HandleFunc("GET /api/v1/semesters/{id}/offerings", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetSemesterOfferings)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-validate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportValidate)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-apply", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportApply)))

	// 4. Jadwal: Pola & Kejadian (§4)
	mux.HandleFunc("GET /api/v1/schedule/patterns", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1Patterns)))
	mux.HandleFunc("POST /api/v1/schedule/patterns", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1Pattern)))
	mux.HandleFunc("PATCH /api/v1/schedule/patterns/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePatchV1Pattern)))
	mux.HandleFunc("POST /api/v1/teaching-events", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1TeachingEvent)))
	mux.HandleFunc("GET /api/v1/teaching-events", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1TeachingEvents)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/preview", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePreviewV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/publish", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePublishV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/revoke", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRevokeV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/participation", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleParticipationV1TeachingEvent)))

	// 5. Tugas & Review (§5)
	mux.HandleFunc("GET /api/v1/tasks", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1Tasks)))
	mux.HandleFunc("POST /api/v1/tasks", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1Task)))
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1TaskDetail)))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePatchV1Task)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/reviews", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleReviewV1Task)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/complete", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCompleteV1Task)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/archive", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleArchiveV1Task)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/restore", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleRestoreV1Task)))

	// 6. Materi (§6)
	mux.HandleFunc("GET /api/v1/materials", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1Materials)))
	mux.HandleFunc("POST /api/v1/materials", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1Material)))

	// 7. Fitur Lanjutan v1.1+ (Ruangan, Notifikasi, Audit, Backup/Restore, Admin)
	mux.HandleFunc("GET /api/v1/rooms/candidates", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetRoomCandidates)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/room-confirmations", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateRoomConfirmation)))
	mux.HandleFunc("GET /api/v1/notifications", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetNotifications)))
	mux.HandleFunc("POST /api/v1/notifications/{id}/retry", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRetryNotification)))
	mux.HandleFunc("GET /api/v1/audit", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetAuditLogs)))
	mux.HandleFunc("POST /api/v1/backups", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateBackup)))
	mux.HandleFunc("POST /api/v1/restores", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleRestoreBackup)))
	mux.HandleFunc("GET /api/v1/admin/status", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleGetAdminStatus)))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/suspend", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleAdminSuspendUser)))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/recover", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleAdminRecoverUser)))

	// Fallback untuk route API v1 yang belum diimplementasikan
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, r *http.Request) {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Endpoint API v1 tidak ditemukan")
	})

	// Fallback untuk route legacy API yang belum diimplementasikan
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		s.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Endpoint belum tersedia (dijadwalkan pada Fase B)",
		})
	})

	// Menyajikan aset web statis (Dashboard Admin) dari web.Files embedded
	mux.Handle("/", http.FileServer(http.FS(web.Files)))

	handler := s.corsMiddleware(s.recoveryMiddleware(mux))

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

// SetSecureCookies mengaktifkan atribut Secure pada seluruh cookie autentikasi.
func (s *Server) SetSecureCookies(enabled bool) {
	s.secureCookies = enabled
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

// corsMiddleware memungkinkan Web Dashboard (UI/UX) diakses lintas port saat masa pengembangan
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			allow := false
			if originURL, err := url.Parse(origin); err == nil {
				originHost := originURL.Hostname()
				reqHost := r.Host
				if h, _, err := net.SplitHostPort(r.Host); err == nil {
					reqHost = h
				}
				if originHost == reqHost {
					allow = true
				}
			}
			if allow {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Portal-Token")

		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")

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

// Start menjalankan HTTP Server di background goroutine
func (s *Server) Start() error {
	fmt.Printf("🌐 [Web API] Server REST API aktif di http://localhost%s\n", s.httpServer.Addr)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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

// queueNotification mendaftarkan pesan notifikasi siaran ke tabel notification_messages
func (s *Server) queueNotification(classID int64, eventType, entityType string, entityID int64, payload map[string]any, triggeredByUserID ...int64) {
	if s.v1DB == nil {
		return
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[Notifikasi] payload %s/%s/%d tidak valid: %v\n", eventType, entityType, entityID, err)
		return
	}
	keySource := fmt.Sprintf("%s:%s:%d:%s", eventType, entityType, entityID, payloadBytes)
	keyHash := sha256.Sum256([]byte(keySource))
	idempotencyKey := hex.EncodeToString(keyHash[:])

	var userID any
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = triggeredByUserID[0]
	}

	// Cari kanal WhatsApp default yang aktif untuk kelas ini jika ada
	var channelID sql.NullInt64
	if err := s.v1DB.QueryRow(`
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&channelID); err != nil || !channelID.Valid {
		fmt.Printf("[Notifikasi] kanal aktif kelas %d tidak tersedia untuk %s/%s/%d\n", classID, eventType, entityType, entityID)
		return
	}

	if _, err := s.v1DB.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', CURRENT_TIMESTAMP, ?);
	`, classID, channelID.Int64, eventType, entityType, entityID, idempotencyKey, string(payloadBytes), userID); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			fmt.Printf("[Notifikasi] gagal menyimpan outbox %s/%s/%d: %v\n", eventType, entityType, entityID, err)
		}
	}
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
