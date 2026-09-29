package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/ratelimit"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"bot-jadwal/web"
)

func hmacNew(key []byte) hash.Hash { return hmac.New(sha256.New, key) }

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
	// BE-013/BE-014: konfigurasi security eksplisit.
	env               string
	authHashKey       []byte
	allowedOrigins    []string
	trustedProxyCIDRs []string
	publicBaseURL     string
	// BE-012: rate limiter terpusat dan persisten (nil = belum di-wire).
	limiter *ratelimit.Service
}

// SetRateLimiter memasang limiter terpusat pada server.
func (s *Server) SetRateLimiter(limiter *ratelimit.Service) {
	s.limiter = limiter
	if s.portalService != nil && limiter != nil {
		s.portalService.SetRateLimiter(portalLimiterAdapter{svc: limiter})
	}
}

// buildLimiter membangun limiter terpusat di atas database v1.
// Tanpa AuthHashKey valid, service memakai kunci efemeral (fingerprint tidak
// stabil lintas restart; production wajib key valid via Config.Validate).
func (s *Server) buildLimiter() {
	if s.v1DB == nil {
		return
	}
	if limiter, err := ratelimit.NewService(s.v1DB, s.authHashKey, nil); err == nil {
		s.SetRateLimiter(limiter)
	}
}

// portalLimiterAdapter menjembatani ratelimit.Service ke antarmuka
// CodeLimiter milik portal dengan policy kode portal.
type portalLimiterAdapter struct {
	svc *ratelimit.Service
}

func (a portalLimiterAdapter) Check(ctx context.Context, subject, source string) (bool, time.Duration, error) {
	res, err := a.svc.Check(ctx, ratelimit.PolicyPortalCode, subject, source)
	if err != nil {
		return false, 0, err
	}
	return res.Allowed, res.RetryAfter, nil
}

func (a portalLimiterAdapter) Record(ctx context.Context, subject, source, outcome string) error {
	return a.svc.Record(ctx, ratelimit.PolicyPortalCode, subject, source, outcome)
}

// clientSource menurunkan source identity request: IP peer ternormalisasi
// (tanpa port) atau client hop dari header proxy tepercaya (BE-012).
func (s *Server) clientSource(r *http.Request) string {
	return ratelimit.ClientSourceHTTP(r.RemoteAddr, r.Header.Get, s.trustedProxyCIDRs)
}

// limitExceeded menulis response 429 generik dengan Retry-After tanpa
// mengungkap counter internal.
func (s *Server) limitExceeded(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(retryAfter.Seconds())
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", secs))
	s.writeV1Error(w, http.StatusTooManyRequests, CodeTooManyRequests, "Terlalu banyak percobaan. Coba lagi nanti.")
}

// checkSensitiveLimit menegakkan policy registry rate limit terpusat (BE-012)
// untuk endpoint sensitif. False = response sudah ditulis (429/503).
// Nil limiter (mode tanpa DB) dilewati.
func (s *Server) checkSensitiveLimit(w http.ResponseWriter, r *http.Request, policy ratelimit.PolicyKey, subject string) bool {
	if s.limiter == nil {
		return true
	}
	res, err := s.limiter.Check(r.Context(), policy, subject, s.clientSource(r))
	if err != nil {
		s.writeV1Error(w, http.StatusServiceUnavailable, CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
		return false
	}
	if !res.Allowed {
		s.limitExceeded(w, res.RetryAfter)
		return false
	}
	return true
}

// recordSensitiveLimit mencatat outcome limiter. Dipanggil setelah operasi
// domain commit; error operasional dicatat tanpa subject/source mentah.
func (s *Server) recordSensitiveLimit(policy ratelimit.PolicyKey, subject, source, outcome string) {
	if s.limiter == nil {
		return
	}
	if err := s.limiter.Record(context.Background(), policy, subject, source, outcome); err != nil {
		fmt.Printf("[RateLimit] gagal mencatat %s: %v\n", string(policy), err)
	}
}

// SecurityOptions menyalurkan konfigurasi security BE-013 ke Server.
type SecurityOptions struct {
	Env               string
	AuthHashKey       string
	AllowedOrigins    []string
	TrustedProxyCIDRs []string
	PublicBaseURL     string
}

// SetSecurityOptions menerapkan konfigurasi security dari Config.
// Dipanggil sebelum Start; nilai origin dinormalisasi ke exact-match.
func (s *Server) SetSecurityOptions(opt SecurityOptions) {
	s.env = strings.ToLower(strings.TrimSpace(opt.Env))
	if s.env == "" {
		s.env = "development"
	}
	s.authHashKey = []byte(opt.AuthHashKey)
	s.publicBaseURL = strings.TrimSpace(opt.PublicBaseURL)
	s.allowedOrigins = nil
	seen := map[string]bool{}
	for _, o := range opt.AllowedOrigins {
		n := normalizeOriginValue(o)
		if n != "" && !seen[n] {
			seen[n] = true
			s.allowedOrigins = append(s.allowedOrigins, n)
		}
	}
	s.trustedProxyCIDRs = nil
	for _, c := range opt.TrustedProxyCIDRs {
		if trimmed := strings.TrimSpace(c); trimmed != "" {
			s.trustedProxyCIDRs = append(s.trustedProxyCIDRs, trimmed)
		}
	}
	// Catatan: hash penyimpanan token/kode portal tetap SHA-256 plain demi
	// kompatibilitas baris lama (token berentropi tinggi); HMAC keyed hanya
	// untuk fingerprint rate limiter (BE-012).
	s.buildLimiter()
}

func normalizeOriginValue(origin string) string {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" {
		return strings.ToLower(u.Scheme) + "://" + host + ":" + port
	}
	return strings.ToLower(u.Scheme) + "://" + host
}

// isProduction melaporkan environment production eksplisit.
func (s *Server) isProduction() bool { return s.env == "production" }

// fingerprint menghitung HMAC-SHA256 namespaced bila AuthHashKey tersedia,
// atau fallback SHA-256 plain (development/test tanpa key). Key tidak pernah
// dipakai untuk enkripsi password atau sebagai token.
func (s *Server) fingerprint(namespace, value string) string {
	if len(s.authHashKey) >= 32 {
		mac := hmacNew(s.authHashKey)
		_, _ = mac.Write([]byte(namespace))
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(value))
		return hex.EncodeToString(mac.Sum(nil))
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

// setAuthCookie menyatukan atribut cookie login, switch context, dan logout:
// HttpOnly, Path /, SameSite Lax, Secure mengikuti environment, tanpa Domain.
func (s *Server) setAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookie menghapus cookie dengan atribut yang cocok dengan setter.
func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// noStore menandai response pembawa token agar tidak di-cache.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
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
		// Limiter default (kunci efemeral hingga SetSecurityOptions memberi
		// key eksplisit). Tabel security_attempts tersedia via migrasi 008.
		s.buildLimiter()
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

// isAllowedCORSMethod memeriksa apakah HTTP method diizinkan untuk CORS preflight.
func isAllowedCORSMethod(m string) bool {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead:
		return true
	default:
		return false
	}
}

// isAllowedCORSHeader memeriksa apakah HTTP header diizinkan untuk CORS preflight.
func isAllowedCORSHeader(h string) bool {
	switch strings.ToLower(strings.TrimSpace(h)) {
	case "content-type", "authorization", "idempotency-key", "x-portal-token", "x-requested-with", "accept", "origin":
		return true
	default:
		return false
	}
}

// isOriginAllowed memeriksa apakah origin diizinkan berdasarkan exact-match allowlist.
// Production HANYA mengizinkan origin yang terdaftar eksplisit di AllowedOrigins atau PublicBaseURL.
// Development & Test mengizinkan origin localhost eksplisit.
func (s *Server) isOriginAllowed(origin string) bool {
	norm := normalizeOriginValue(origin)
	if norm == "" {
		return false
	}
	for _, allowed := range s.allowedOrigins {
		if norm == allowed {
			return true
		}
	}
	if s.publicBaseURL != "" && norm == normalizeOriginValue(s.publicBaseURL) {
		return true
	}
	if s.isProduction() {
		return false
	}
	if u, err := url.Parse(norm); err == nil {
		h := strings.ToLower(u.Hostname())
		if h == "localhost" || h == "127.0.0.1" || h == "::1" {
			return true
		}
	}
	return false
}

// addVaryOrigin menggabungkan 'Origin' ke header Vary tanpa menimpa nilai yang ada.
func addVaryOrigin(w http.ResponseWriter) {
	existing := w.Header().Values("Vary")
	for _, v := range existing {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "origin") {
				return
			}
		}
	}
	if len(existing) == 0 {
		w.Header().Set("Vary", "Origin")
	} else {
		w.Header().Add("Vary", "Origin")
	}
}

// isRequestHTTPS memeriksa apakah request berjalan di atas HTTPS (langsung atau via trusted proxy).
func (s *Server) isRequestHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	peer := ratelimit.PeerIP(r.RemoteAddr)
	if ratelimit.PeerTrusted(peer, s.trustedProxyCIDRs) {
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	return false
}

// corsMiddleware menegakkan CORS exact-origin, CSRF check, dan HTTP security headers (BE-014).
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Security Headers dasar & Content Security Policy (BE-014)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self';")

		if s.isProduction() && s.isRequestHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		origin := r.Header.Get("Origin")

		// 2. CORS Preflight (OPTIONS dengan Access-Control-Request-Method)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if origin == "" || !s.isOriginAllowed(origin) {
				addVaryOrigin(w)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			reqMethod := r.Header.Get("Access-Control-Request-Method")
			if !isAllowedCORSMethod(reqMethod) {
				addVaryOrigin(w)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				for _, h := range strings.Split(reqHeaders, ",") {
					if !isAllowedCORSHeader(h) {
						addVaryOrigin(w)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}
			}
			addVaryOrigin(w)
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Portal-Token")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 3. Response CORS untuk request non-preflight dengan header Origin
		if origin != "" {
			addVaryOrigin(w)
			if s.isOriginAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		// 4. OPTIONS non-preflight sederhana
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 5. CSRF Protection untuk cookie-authenticated mutations (POST/PUT/PATCH/DELETE)
		// Bearer-only client (bot WA / API token) dilewati tanpa CSRF check.
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			cookie, err := r.Cookie("bv1")
			hasCookie := err == nil && strings.TrimSpace(cookie.Value) != ""
			hasBearer := strings.HasPrefix(strings.ToLower(r.Header.Get("Authorization")), "bearer ")
			if hasCookie && !hasBearer {
				if origin != "" {
					if !s.isOriginAllowed(origin) {
						s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Origin tidak diizinkan untuk mutasi kredensial cookie")
						return
					}
				} else {
					referer := r.Header.Get("Referer")
					if referer != "" {
						refURL, err := url.Parse(referer)
						if err != nil || !s.isOriginAllowed(refURL.Scheme+"://"+refURL.Host) {
							s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Referer tidak diizinkan untuk mutasi kredensial cookie")
							return
						}
					} else if s.isProduction() {
						s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Origin atau Referer wajib untuk mutasi kredensial cookie")
						return
					}
				}
			}
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
