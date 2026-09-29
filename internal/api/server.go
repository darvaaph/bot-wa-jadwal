package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/api/legacy"
	"bot-jadwal/internal/api/middleware"
	v1 "bot-jadwal/internal/api/v1"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/ratelimit"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
)

// Server mengelola HTTP REST API untuk Web Admin Dashboard dan API v1
type Server struct {
	httpServer     *http.Server
	botClient      *bot.BotClient
	classManager   *schedule.ClassManager
	taskManager    *task.TaskManager
	v1DB           *sql.DB
	portalService  *portal.Service
	legacyHandler  *legacy.Handler
	secManager     *middleware.SecurityManager
	authManager    *middleware.AuthManager
	rlManager          *middleware.RateLimitManager
	taskController     *v1.TaskController
	scheduleController *v1.ScheduleController
	portalController   *v1.PortalController
	authController     *v1.AuthController
	academicController *v1.AcademicController
	adminController    *v1.AdminController
	storageDir         string
	secureCookies bool
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

// queueNotification mendaftarkan pesan notifikasi siaran ke tabel notification_messages.
// BE-005 durable outbox: intent tetap disimpan PENDING dengan channel NULL
// ketika kelas belum punya kanal aktif; worker tidak mengklaimnya sampai
// ReconcilePendingChannels mengisi channel + scheduled_at.
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

	// Cari kanal WhatsApp default yang aktif untuk kelas ini jika ada (boleh kosong).
	var channelID any
	var scheduledAt any
	var ch int64
	if err := s.v1DB.QueryRow(`
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&ch); err == nil {
		channelID = ch
		scheduledAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if _, err := s.v1DB.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING;
	`, classID, channelID, eventType, entityType, entityID, idempotencyKey, string(payloadBytes), scheduledAt, userID); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			fmt.Printf("[Notifikasi] gagal menyimpan outbox %s/%s/%d: %v\n", eventType, entityType, entityID, err)
		}
	}
}

// queueNotificationTx menyimpan intent dalam transaksi bisnis yang sama.
func (s *Server) queueNotificationTx(ctx context.Context, tx *sql.Tx, classID int64, eventType, entityType string, entityID int64, payload map[string]any, idempotencyKey string, triggeredByUserID ...int64) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		keySource := fmt.Sprintf("%s:%s:%d:%s", eventType, entityType, entityID, payloadBytes)
		keyHash := sha256.Sum256([]byte(keySource))
		idempotencyKey = hex.EncodeToString(keyHash[:])
	}
	var userID any
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = triggeredByUserID[0]
	}
	var channelID any
	var scheduledAt any
	var ch int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&ch); err == nil {
		channelID = ch
		scheduledAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING;
	`, classID, channelID, eventType, entityType, entityID, idempotencyKey, string(payloadBytes), scheduledAt, userID)
	return err
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
