package api

import (
	"net/http"
	"path"
	"strings"

	"bot-jadwal/web"
)

// registerRoutes mendaftarkan seluruh endpoint REST API (legacy shim, API v1, dan web assets)
func (s *Server) registerRoutes(mux *http.ServeMux) {
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
	mux.HandleFunc("POST /api/v1/auth/recovery/request", s.handleRequestRecovery)
	mux.HandleFunc("POST /api/v1/auth/recovery/confirm", s.handleConfirmRecovery)
	mux.HandleFunc("POST /api/v1/auth/logout", s.RequireAuth(s.handleLogout))
	mux.HandleFunc("GET /api/v1/auth/me", s.RequireAuth(s.handleGetMe))
	mux.HandleFunc("POST /api/v1/auth/switch-context", s.RequireAuth(s.handleSwitchContext))
	mux.HandleFunc("GET /api/v1/classes", s.handleGetV1ClassesAccess)
	mux.HandleFunc("POST /api/v1/classes", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleCreateV1Class)))
	mux.HandleFunc("PATCH /api/v1/classes/{slug}", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handlePatchV1ClassStatus)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/portal-code/rotate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRotatePortalCode)))
	mux.HandleFunc("GET /api/v1/classes/{slug}/settings", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetClassSettings)))
	mux.HandleFunc("PATCH /api/v1/classes/{slug}/settings", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleUpdateClassSettings)))
	mux.HandleFunc("PATCH /api/v1/classes/{slug}/portal-mode", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSetPortalMode)))
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
	mux.HandleFunc("GET /api/v1/portal/{slug}/courses", s.handlePortalCourses)
	mux.HandleFunc("GET /api/v1/portal/{slug}/semesters", s.handlePortalSemesters)

	// 3. Semester & Offering (§3)
	mux.HandleFunc("GET /api/v1/classes/{slug}/semesters", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetClassSemesters)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateClassSemester)))
	mux.HandleFunc("POST /api/v1/classes/{slug}/semesters/{id}/activate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleActivateSemester)))
	mux.HandleFunc("GET /api/v1/classes/{slug}/semesters/{id}/preview", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handlePreviewSemester)))
	mux.HandleFunc("GET /api/v1/semesters/{id}/offerings", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetSemesterOfferings)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/offerings", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleCreateSemesterOffering)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-validate", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportValidate)))
	mux.HandleFunc("POST /api/v1/semesters/{id}/import-apply", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleSemesterImportApply)))

	// 4. Jadwal: Pola & Kejadian (§4)
	mux.HandleFunc("GET /api/v1/schedule/patterns", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1Patterns)))
	mux.HandleFunc("POST /api/v1/schedule/patterns", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1Pattern)))
	mux.HandleFunc("POST /api/v1/schedule/patterns/preview", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePreviewCreateV1Pattern)))
	mux.HandleFunc("PATCH /api/v1/schedule/patterns/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePatchV1Pattern)))
	mux.HandleFunc("POST /api/v1/schedule/patterns/{id}/preview", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePreviewV1Pattern)))
	mux.HandleFunc("DELETE /api/v1/schedule/patterns/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleDeleteV1Pattern)))
	mux.HandleFunc("POST /api/v1/teaching-events", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateV1TeachingEvent)))
	mux.HandleFunc("GET /api/v1/teaching-events", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1TeachingEvents)))
	mux.HandleFunc("GET /api/v1/teaching-events/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetV1TeachingEventDetail)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/preview", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePreviewV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/publish", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePublishV1TeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/revoke", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRevokeV1TeachingEvent)))
	mux.HandleFunc("DELETE /api/v1/teaching-events/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleDeleteV1TeachingEvent)))
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
	mux.HandleFunc("PATCH /api/v1/materials/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePatchV1Material)))
	mux.HandleFunc("DELETE /api/v1/materials/{id}", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleDeleteV1Material)))

	// 7. Fitur Lanjutan v1.1+ (Ruangan, Notifikasi, Audit, Backup/Restore, Admin)
	mux.HandleFunc("GET /api/v1/rooms/candidates", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetRoomCandidates)))
	mux.HandleFunc("GET /api/v1/rooms/confirmations", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetRoomConfirmations)))
	mux.HandleFunc("GET /api/v1/publications/{entityType}/{id}/delivery", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handlePublicationDelivery)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/room-confirmations", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleCreateRoomConfirmation)))
	mux.HandleFunc("GET /api/v1/notifications", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetNotifications)))
	mux.HandleFunc("GET /api/v1/notifications/{id}/attempts", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetNotificationAttempts)))
	mux.HandleFunc("POST /api/v1/notifications/{id}/retry", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRetryNotification)))
	mux.HandleFunc("GET /api/v1/audit", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetAuditLogs)))
	mux.HandleFunc("POST /api/v1/backups", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleCreateBackup)))
	mux.HandleFunc("POST /api/v1/backup-requests", s.RequireAuth(s.RequireRole("KM")(s.handleCreateBackupRequest)))
	mux.HandleFunc("GET /api/v1/backup-requests", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleListBackupRequests)))
	mux.HandleFunc("POST /api/v1/backup-requests/{id}/execute", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleExecuteBackupRequest)))
	mux.HandleFunc("GET /api/v1/backups", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetBackups)))
	mux.HandleFunc("POST /api/v1/restores", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleRestoreBackup)))
	mux.HandleFunc("POST /api/v1/backups/{id}/restore-preview", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleScopedRestorePreview)))
	mux.HandleFunc("POST /api/v1/backups/{id}/restore-execute", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleScopedRestoreExecute)))
	mux.HandleFunc("GET /api/v1/admin/status", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleGetAdminStatus)))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/suspend", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleAdminSuspendUser)))
	mux.HandleFunc("POST /api/v1/admin/users/{id}/recover", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleAdminRecoverUser)))
	mux.HandleFunc("GET /api/v1/admin/users", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleGetAdminUsers)))
	mux.HandleFunc("GET /api/v1/admin/assignments", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN", "KM")(s.handleGetAdminAssignments)))
	mux.HandleFunc("POST /api/v1/admin/assignments/{id}/suspend", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN", "KM")(s.handleAdminSuspendAssignment)))
	mux.HandleFunc("POST /api/v1/admin/assignments/{id}/revoke", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN", "KM")(s.handleAdminRevokeAssignment)))
	mux.HandleFunc("GET /api/v1/admin/invitations", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN", "KM")(s.handleGetAdminInvitations)))
	mux.HandleFunc("POST /api/v1/admin/invitations/{id}/revoke", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN", "KM")(s.handleAdminRevokeInvitation)))
	mux.HandleFunc("POST /api/v1/admin/support/enter", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleEnterSupport)))
	mux.HandleFunc("POST /api/v1/admin/support/exit", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleExitSupport)))
	mux.HandleFunc("GET /api/v1/admin/support/active", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleGetActiveSupport)))
	mux.HandleFunc("POST /api/v1/admin/bot/test-message", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleTestBotMessage)))
	mux.HandleFunc("POST /api/v1/master/proposals", s.RequireAuth(s.RequireRole("KM")(s.handleCreateProposal)))
	mux.HandleFunc("GET /api/v1/master/proposals", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetProposals)))
	mux.HandleFunc("POST /api/v1/master/proposals/{id}/approve", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleApproveProposal)))
	mux.HandleFunc("POST /api/v1/master/proposals/{id}/reject", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleRejectProposal)))
	mux.HandleFunc("GET /api/v1/whatsapp-channels", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleGetChannels)))
	mux.HandleFunc("POST /api/v1/whatsapp-channels", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleLinkChannel)))
	mux.HandleFunc("POST /api/v1/whatsapp-channels/{id}/revoke", s.RequireAuth(s.RequireRole("KM", "SYSTEM_ADMIN")(s.handleRevokeChannel)))
	mux.HandleFunc("GET /api/v1/master/rooms", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetMasterRooms)))
	mux.HandleFunc("POST /api/v1/master/rooms", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleCreateMasterRoom)))
	mux.HandleFunc("PATCH /api/v1/master/rooms/{id}", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handlePatchMasterRoom)))
	mux.HandleFunc("POST /api/v1/master/rooms/sync-jadwal", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleSyncMasterRooms)))
	mux.HandleFunc("GET /api/v1/master/courses", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetMasterCourses)))
	mux.HandleFunc("POST /api/v1/master/courses", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleCreateMasterCourse)))
	mux.HandleFunc("POST /api/v1/master/courses/sync-jadwal", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleSyncMasterCourses)))
	mux.HandleFunc("POST /api/v1/master/courses/bulk", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleBulkCreateMasterCourses)))
	mux.HandleFunc("PATCH /api/v1/master/courses/{id}", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handlePatchMasterCourse)))
	mux.HandleFunc("GET /api/v1/master/lecturers", s.RequireAuth(s.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(s.handleGetMasterLecturers)))
	mux.HandleFunc("POST /api/v1/master/lecturers", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleCreateMasterLecturer)))
	mux.HandleFunc("POST /api/v1/master/lecturers/sync-jadwal", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleSyncMasterLecturers)))
	mux.HandleFunc("POST /api/v1/master/lecturers/bulk", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleBulkCreateMasterLecturers)))
	mux.HandleFunc("PATCH /api/v1/master/lecturers/{id}", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handlePatchMasterLecturer)))
	mux.HandleFunc("POST /api/v1/master/sync-all", s.RequireAuth(s.RequireRole("SYSTEM_ADMIN")(s.handleSyncAllMaster)))

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

	// Route akses portal mahasiswa via slug (/c/{slug})
	mux.HandleFunc("GET /c/{slug}", func(w http.ResponseWriter, r *http.Request) {
		indexFile, err := web.Files.ReadFile("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexFile)
	})

	// Alias lama: /superadmin.html dialihkan permanen ke /system-admin.html.
	mux.HandleFunc("GET /superadmin.html", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/system-admin.html", http.StatusMovedPermanently)
	})

	// Menyajikan aset web statis dari web.Files embedded.
	// Path halaman yang tidak ada (mis. salah ketik *.html) mendapat
	// halaman 404 kustom, bukan teks polos FileServer.
	webFS := http.FileServer(http.FS(web.Files))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebPagePath(r.URL.Path) {
			if _, err := web.Files.Open(strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")); err != nil {
				serveWebNotFound(w, r)
				return
			}
		}
		webFS.ServeHTTP(w, r)
	}))
}
