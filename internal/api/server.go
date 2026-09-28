package api

import (
	"context"
	"database/sql"
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
	"bot-jadwal/internal/backup"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/notify"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/rooms"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/semester"
	"bot-jadwal/internal/task"
	"bot-jadwal/web"
)

// Server mengelola HTTP REST API untuk Web Admin Dashboard dan API v1
type Server struct {
	httpServer   *http.Server
	botClient    *bot.BotClient
	classManager *schedule.ClassManager
	taskManager  *task.TaskManager
	v1DB         *sql.DB
	storageDir   string
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
	V1            string    `json:"v1,omitempty"`
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
func NewServer(addr string, botClient *bot.BotClient, classManager *schedule.ClassManager, taskManager *task.TaskManager, v1DB ...*sql.DB) *Server {
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
	if len(v1DB) > 0 && v1DB[0] != nil {
		s.v1DB = v1DB[0]
	}

	// Registrasi Route API Scaffolding (Legacy Shim dengan header Deprecation: true)
	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/status", s.authenticateIfConfigured(s.handleStatus))

	mux.HandleFunc("GET /api/academic/classes", s.authenticateIfConfigured(s.handleAcademicClasses))
	mux.HandleFunc("GET /api/academic/classes/{id}/courses", s.authenticateIfConfigured(s.handleAcademicCourses))
	mux.HandleFunc("POST /api/v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("GET /api/v1/auth/session", s.handleAuthSession)
	mux.HandleFunc("POST /api/v1/auth/switch-context", s.handleAuthSwitchContext)

	// Registrasi Route Jadwal & Kelas Legacy
	mux.HandleFunc("GET /api/classes", s.handleClasses)
	mux.HandleFunc("GET /api/schedule", s.handleSchedule)
	// Registrasi Route API Tugas Legacy
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
	mux.HandleFunc("POST /api/portal/{slug}/verify-code", s.handlePortalVerifyCode)
	mux.HandleFunc("GET /api/portal/{slug}/access", s.handlePortalAccess)

	mux.HandleFunc("POST /api/v1/invitations", s.authenticateMutationIfConfigured(s.handleCreateInvitation))
	mux.HandleFunc("GET /api/v1/invitations", s.authenticateIfConfigured(s.handleListInvitations))
	mux.HandleFunc("GET /api/v1/invitations/{token}", s.handleGetInvitationByToken)
	mux.HandleFunc("POST /api/v1/invitations/{token}/accept", s.handleAcceptInvitation)
	mux.HandleFunc("POST /api/v1/invitations/{id}/revoke", s.authenticateMutationIfConfigured(s.handleRevokeInvitation))
	mux.HandleFunc("POST /api/v1/invitations/{id}/resend", s.authenticateMutationIfConfigured(s.handleResendInvitation))

	mux.HandleFunc("GET /api/v1/role-assignments", s.authenticateIfConfigured(s.handleListRoleAssignments))
	mux.HandleFunc("PATCH /api/v1/role-assignments/{id}", s.authenticateMutationIfConfigured(s.handleUpdateRoleAssignment))

	mux.HandleFunc("POST /api/v1/auth/recovery/request", s.handleRequestRecovery)
	mux.HandleFunc("POST /api/v1/auth/recovery/confirm", s.handleConfirmRecovery)
	mux.HandleFunc("POST /api/v1/admin/recovery/issue", s.authenticateMutationIfConfigured(s.handleIssueRecovery))

	mux.HandleFunc("PATCH /api/v1/classes/{id}/settings", s.authenticateMutationIfConfigured(s.handleUpdateClassSettings))
	mux.HandleFunc("GET /api/v1/classes/{id}/settings", s.authenticateIfConfigured(s.handleGetClassSettings))

	mux.HandleFunc("POST /api/v1/classes", s.authenticateMutationIfConfigured(s.handleCreateClass))
	mux.HandleFunc("GET /api/v1/classes/{id}/semesters", s.authenticateIfConfigured(s.handleListSemesters))
	mux.HandleFunc("POST /api/v1/classes/{id}/semesters/draft", s.authenticateMutationIfConfigured(s.handleCreateSemesterDraft))
	mux.HandleFunc("POST /api/v1/classes/{id}/semesters/import", s.authenticateMutationIfConfigured(s.handleSemesterImport))
	mux.HandleFunc("GET /api/v1/classes/{id}/semesters/{sid}/preview", s.authenticateIfConfigured(s.handleSemesterPreview))
	mux.HandleFunc("POST /api/v1/classes/{id}/semesters/{sid}/activate", s.authenticateMutationIfConfigured(s.handleSemesterActivate))
	mux.HandleFunc("POST /api/v1/classes/{id}/semesters/{sid}/offerings", s.authenticateMutationIfConfigured(s.handleAddOffering))
	mux.HandleFunc("POST /api/v1/patterns", s.authenticateMutationIfConfigured(s.handleAddPattern))

	mux.HandleFunc("POST /api/v1/teaching-events/draft", s.authenticateMutationIfConfigured(s.handleCreateTeachingEventDraft))
	mux.HandleFunc("PUT /api/v1/teaching-events/{id}", s.authenticateMutationIfConfigured(s.handleUpdateTeachingEventDraft))
	mux.HandleFunc("DELETE /api/v1/teaching-events/{id}", s.authenticateMutationIfConfigured(s.handleDeleteTeachingEventDraft))
	mux.HandleFunc("GET /api/v1/teaching-events", s.authenticateIfConfigured(s.handleListTeachingEvents))
	mux.HandleFunc("GET /api/v1/teaching-events/{id}", s.authenticateIfConfigured(s.handleGetTeachingEventDetail))
	mux.HandleFunc("GET /api/v1/teaching-events/{id}/preview", s.authenticateIfConfigured(s.handlePreviewTeachingEvent))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/publish", s.authenticateMutationIfConfigured(s.handlePublishTeachingEvent))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/revoke", s.authenticateMutationIfConfigured(s.handleRevokeTeachingEvent))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/participants", s.authenticateMutationIfConfigured(s.handleAddEventParticipant))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/participants/{offeringId}/respond", s.authenticateMutationIfConfigured(s.handleRespondEventParticipant))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/room-confirmation", s.authenticateMutationIfConfigured(s.handleRecordRoomConfirmation))

	mux.HandleFunc("GET /api/v1/notifications", s.authenticateIfConfigured(s.handleListNotifications))
	mux.HandleFunc("POST /api/v1/notifications/{id}/retry", s.authenticateMutationIfConfigured(s.handleRetryNotification))
	mux.HandleFunc("POST /api/v1/admin/notifications/process", s.authenticateMutationIfConfigured(s.handleProcessNotifications))

	mux.HandleFunc("GET /api/v1/rooms", s.authenticateIfConfigured(s.handleListRooms))
	mux.HandleFunc("POST /api/v1/admin/rooms", s.authenticateMutationIfConfigured(s.handleCreateRoom))
	mux.HandleFunc("PUT /api/v1/admin/rooms/{id}", s.authenticateMutationIfConfigured(s.handleUpdateRoom))
	mux.HandleFunc("GET /api/v1/rooms/availability", s.authenticateIfConfigured(s.handleRoomAvailability))
	mux.HandleFunc("POST /api/v1/rooms/proposals", s.authenticateMutationIfConfigured(s.handleRoomProposal))

	mux.HandleFunc("GET /api/v1/audit", s.authenticateIfConfigured(s.handleListAudit))

	mux.HandleFunc("POST /api/v1/admin/backups", s.authenticateMutationIfConfigured(s.handleCreateBackup))
	mux.HandleFunc("GET /api/v1/admin/backups", s.authenticateIfConfigured(s.handleListBackups))
	mux.HandleFunc("POST /api/v1/admin/backups/{id}/restore", s.authenticateMutationIfConfigured(s.handleRestoreBackup))
	mux.HandleFunc("GET /api/v1/admin/system-status", s.authenticateIfConfigured(s.handleSystemStatus))

	mux.HandleFunc("GET /api/v1/admin/channels", s.authenticateIfConfigured(s.handleListChannels))
	mux.HandleFunc("GET /api/v1/admin/chats/unlinked", s.authenticateIfConfigured(s.handleListUnlinkedChats))
	mux.HandleFunc("POST /api/v1/admin/channels", s.authenticateMutationIfConfigured(s.handleLinkChannel))
	mux.HandleFunc("POST /api/v1/admin/channels/{id}/revoke", s.authenticateMutationIfConfigured(s.handleRevokeChannel))

	// Route API v1 (Lapis L2 & Fitur Lanjutan)

	// 1. Auth & Konteks (§1)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/v1/auth/logout", s.RequireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/me", s.RequireAuth(s.handleGetMe))
	mux.HandleFunc("POST /api/v1/auth/switch-context", s.RequireAuth(s.handleSwitchContext))
	mux.HandleFunc("GET /api/v1/classes", s.handleGetV1Classes)
	mux.HandleFunc("PATCH /api/v1/classes/{slug}", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handlePatchV1ClassStatus)))
	mux.HandleFunc("POST /api/v1/invitations", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateInvitation)))
	mux.HandleFunc("POST /api/v1/invitations/accept", s.handleAcceptInvitation)

	// 2. Portal Mahasiswa (§2)
	mux.HandleFunc("GET /api/v1/portal/{slug}/summary", s.handlePortalSummary)
	mux.HandleFunc("GET /api/v1/portal/{slug}/schedule", s.handlePortalSchedule)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks", s.handlePortalTasks)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks/{id}", s.handlePortalTaskDetail)
	mux.HandleFunc("GET /api/v1/portal/{slug}/changes", s.handlePortalChanges)
	mux.HandleFunc("GET /api/v1/portal/{slug}/materials", s.handlePortalMaterials)

	// 3. Semester & Offering (§3)
	mux.HandleFunc("GET /api/v1/classes/{slug}/semesters", s.RequireAuth(s.handleGetClassSemesters))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateClassSemester)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters/{id}/activate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleActivateSemester)))
	mux.HandleFunc("GET /api/v1/semesters/{id}/offerings", s.RequireAuth(s.handleGetSemesterOfferings))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-validate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportValidate)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-apply", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportApply)))

	// 4. Jadwal: Pola & Kejadian (§4)
	mux.HandleFunc("GET /api/v1/schedule/patterns", s.RequireAuth(s.handleGetV1Patterns))
	mux.HandleFunc("POST /api/v1/schedule/patterns", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1Pattern)))
	mux.HandleFunc("PATCH /api/v1/schedule/patterns/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePatchV1Pattern)))
	mux.HandleFunc("POST /api/v1/teaching-events", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1TeachingEvent)))
	mux.HandleFunc("GET /api/v1/teaching-events", s.RequireAuth(s.handleGetV1TeachingEvents))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/preview", s.RequireAuth(s.handlePreviewV1TeachingEvent))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/publish", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePublishV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/revoke", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRevokeV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/participation", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleParticipationV1TeachingEvent)))

	// 5. Tugas & Review (§5)
	mux.HandleFunc("GET /api/v1/tasks", s.RequireAuth(s.handleGetV1Tasks))
	mux.HandleFunc("POST /api/v1/tasks", s.RequireAuth(s.handleCreateV1Task))
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.RequireAuth(s.handleGetV1TaskDetail))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", s.RequireAuth(s.handlePatchV1Task))
	mux.HandleFunc("POST /api/v1/tasks/{id}/reviews", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleReviewV1Task)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/complete", s.RequireAuth(s.handleCompleteV1Task))
	mux.HandleFunc("POST /api/v1/tasks/{id}/archive", s.RequireAuth(s.handleArchiveV1Task))
	mux.HandleFunc("POST /api/v1/tasks/{id}/restore", s.RequireAuth(s.handleRestoreV1Task))

	// 6. Materi (§6)
	mux.HandleFunc("GET /api/v1/materials", s.handleGetV1Materials)
	mux.HandleFunc("POST /api/v1/materials", s.RequireAuth(s.handleCreateV1Material))

	// 7. Fitur Lanjutan v1.1+ (Ruangan, Notifikasi, Audit, Backup/Restore, Admin)
	mux.HandleFunc("GET /api/v1/rooms/candidates", s.RequireAuth(s.handleGetRoomCandidates))
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

// queueNotification mendaftarkan pesan notifikasi siaran ke tabel notification_messages
func (s *Server) queueNotification(classID int64, eventType, entityType string, entityID int64, payload map[string]any, triggeredByUserID ...int64) {
	if s.v1DB == nil {
		return
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		payloadBytes = []byte("{}")
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%d:%d", eventType, entityType, entityID, time.Now().UnixNano())

	var userID any
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = triggeredByUserID[0]
	}

	// Cari kanal WhatsApp default yang aktif untuk kelas ini jika ada
	var channelID sql.NullInt64
	_ = s.v1DB.QueryRow(`
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&channelID)

	_, _ = s.v1DB.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?);
	`, classID, func() any {
		if channelID.Valid {
			return channelID.Int64
		}
		return nil
	}(), eventType, entityType, entityID, idempotencyKey, string(payloadBytes), userID)
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
