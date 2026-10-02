package v1

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/ratelimit"
	"golang.org/x/crypto/bcrypt"
)

// generateSecureToken menghasilkan token acak 32-byte dengan awalan "bv1_"
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bv1_" + hex.EncodeToString(b), nil
}

// computeHash menghitung hash SHA-256 string
func computeHash(s string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(h[:])
}

// LoginRequest merepresentasikan payload login pengurus
type LoginRequest struct {
	IdentityKey string `json:"identity_key"`
	Password    string `json:"password"`
}

// Kebijakan rate limit login (registry terpusat BE-012, angka sesuai ADR-0009).
const (
	loginMaxFailures = 5
	loginWindow      = 15 * time.Minute
	loginBlockPeriod = 15 * time.Minute
)

// RoleAssignmentItem merepresentasikan penugasan peran pengurus
type RoleAssignmentItem struct {
	ID           int64  `json:"id"`
	Role         string `json:"role"`
	ClassSlug    string `json:"class_slug,omitempty"`
	SemesterID   *int64 `json:"semester_id,omitempty"`
	OfferingID   *int64 `json:"offering_id,omitempty"`
	OfferingName string `json:"offering_name,omitempty"`
}

// LoginResponse adalah payload data respons login sukses
type LoginResponse struct {
	Token             string               `json:"token"`
	TokenType         string               `json:"token_type"`
	ExpiresAt         time.Time            `json:"expires_at"`
	Assignments       []RoleAssignmentItem `json:"assignments"`
	NeedContextChoice bool                 `json:"need_context_choice"`
}

// SwitchContextRequest adalah payload untuk ganti peran/konteks
type SwitchContextRequest struct {
	RoleAssignmentID int64 `json:"role_assignment_id"`
}

// UpdateClassStatusRequest adalah payload perubahan status kelas
type UpdateClassStatusRequest struct {
	Status string `json:"status"` // ACTIVE, INACTIVE, ARCHIVED
}

// InvitationRequest adalah payload pembuatan undangan
type InvitationRequest struct {
	Role               string `json:"role"` // KM atau PJ
	ClassSlug          string `json:"class_slug"`
	SemesterID         *int64 `json:"semester_id,omitempty"`
	OfferingID         *int64 `json:"offering_id,omitempty"`
	InvitedIdentityKey string `json:"invited_identity_key"`
}

// AcceptInvitationRequest adalah payload menerima undangan
type AcceptInvitationRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

type RecoveryRequest struct {
	IdentityKey string `json:"identity_key"`
}

type RecoveryConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

// RecoverySender is the minimum WhatsApp capability needed by password
// recovery. BotClient and test doubles implement this interface.
type RecoverySender interface {
	SendText(ctx context.Context, jid, text string) (string, error)
}

// AuthController mengelola seluruh endpoint autentikasi, sesi pengurus, hak akses kelas, dan undangan
type AuthController struct {
	db              *sql.DB
	secManager      *middleware.SecurityManager
	rlManager       *middleware.RateLimitManager
	portalService   *portal.Service
	recoveryService *auth.Service
	recoverySender  RecoverySender
	publicBaseURL   string
}

// ConfigureRecovery wires the auth domain service and verified WhatsApp
// delivery after server security configuration has been loaded.
func (c *AuthController) ConfigureRecovery(service *auth.Service, sender RecoverySender, publicBaseURL string) {
	c.recoveryService = service
	c.recoverySender = sender
	c.publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
}

func (c *AuthController) SetRecoverySender(sender RecoverySender) {
	c.recoverySender = sender
}

// NewAuthController membuat instance baru AuthController
func NewAuthController(db *sql.DB, secManager *middleware.SecurityManager, rlManager *middleware.RateLimitManager, portalService *portal.Service) *AuthController {
	return &AuthController{
		db:            db,
		secManager:    secManager,
		rlManager:     rlManager,
		portalService: portalService,
	}
}

// RegisterRoutes mendaftarkan seluruh endpoint auth ke ServeMux
func (c *AuthController) RegisterRoutes(mux *http.ServeMux, auth *middleware.AuthManager) {
	mux.HandleFunc("POST /api/v1/auth/login", c.Login)
	mux.HandleFunc("POST /api/v1/auth/recovery/request", c.RequestRecovery)
	mux.HandleFunc("POST /api/v1/auth/recovery/confirm", c.ConfirmRecovery)
	mux.HandleFunc("POST /api/v1/auth/logout", auth.RequireAuth(c.Logout))
	mux.HandleFunc("GET /api/v1/auth/me", auth.RequireAuth(c.GetMe))
	mux.HandleFunc("POST /api/v1/auth/switch-context", auth.RequireAuth(c.SwitchContext))
	mux.HandleFunc("GET /api/v1/classes", c.GetClassesAccess)
	mux.HandleFunc("PATCH /api/v1/classes/{slug}", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.PatchClassStatus)))
	mux.HandleFunc("POST /api/v1/invitations", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.CreateInvitation)))
	mux.HandleFunc("POST /api/v1/invitations/accept", c.AcceptInvitation)
}

func (c *AuthController) clientSource(r *http.Request) string {
	var trusted []string
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	return middleware.ClientSource(r, trusted)
}

func (c *AuthController) fingerprint(namespace, value string) string {
	if c.secManager != nil {
		return c.secManager.Fingerprint(namespace, value)
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

func (c *AuthController) setAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	if c.secManager != nil {
		c.secManager.SetAuthCookie(w, token, expires)
	}
}

func (c *AuthController) clearAuthCookie(w http.ResponseWriter) {
	if c.secManager != nil {
		c.secManager.ClearAuthCookie(w)
	}
}

func (c *AuthController) loginBlocked(identityHash, sourceHash string) (bool, time.Duration, error) {
	now := time.Now().UTC()
	rows, err := c.db.Query(`
		SELECT outcome, attempted_at FROM login_attempts
		WHERE identity_hash = ? AND source_hash = ?
		ORDER BY id DESC LIMIT 32;
	`, identityHash, sourceHash)
	if err != nil {
		return false, 0, err
	}
	defer rows.Close()
	failures := 0
	var lastFailure time.Time
	for rows.Next() {
		var outcome, attempted string
		if err := rows.Scan(&outcome, &attempted); err != nil {
			return false, 0, err
		}
		ts, err := parseAttemptTime(attempted)
		if err != nil {
			continue
		}
		if outcome == "SUCCESS" {
			break
		}
		if outcome != "FAILURE" {
			continue
		}
		if now.Sub(ts) > loginWindow+loginBlockPeriod {
			break
		}
		if lastFailure.IsZero() {
			lastFailure = ts
		}
		if now.Sub(ts) <= loginWindow {
			failures++
		}
	}
	if err := rows.Err(); err != nil {
		return false, 0, err
	}
	if failures < loginMaxFailures {
		return false, 0, nil
	}
	until := lastFailure.Add(loginBlockPeriod)
	if now.Before(until) {
		return true, until.Sub(now), nil
	}
	return false, 0, nil
}

func (c *AuthController) recordLoginAttempt(userID *int64, identityHash, sourceHash, outcome string) error {
	var uid any
	if userID != nil {
		uid = *userID
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := c.db.Exec(`INSERT INTO login_attempts (
		user_id, identity_hash, source_hash, outcome, attempted_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`, uid, identityHash, sourceHash, outcome, now, now, now)
	return err
}

func parseAttemptTime(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if ts, err := time.Parse(layout, trimmed); err == nil {
			return ts.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("format waktu percobaan tidak dikenal")
}

// Login menangani POST /api/v1/auth/login
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	cleanIdentity := auth.NormalizeIdentity(req.IdentityKey)
	if cleanIdentity == "" || req.Password == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nomor WhatsApp (identity_key) dan kata sandi wajib diisi")
		return
	}

	source := c.clientSource(r)
	identityHash := c.fingerprint("identity", cleanIdentity)
	sourceHash := c.fingerprint("source", source)

	blocked, retryAfter, err := c.loginBlocked(identityHash, sourceHash)
	if err != nil {
		common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
		return
	}
	if blocked {
		if err := c.recordLoginAttempt(nil, identityHash, sourceHash, "BLOCKED"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
			return
		}
		middleware.LimitExceeded(w, retryAfter)
		return
	}

	var (
		userID         int64
		identityKey    string
		displayName    string
		passwordHash   string
		status         string
		sessionVersion int
	)

	err = c.db.QueryRow(`
		SELECT id, identity_key, display_name, password_hash, status, session_version
		FROM users
		WHERE identity_key = ?;
	`, cleanIdentity).Scan(&userID, &identityKey, &displayName, &passwordHash, &status, &sessionVersion)

	if err == sql.ErrNoRows {
		if err := c.recordLoginAttempt(nil, identityHash, sourceHash, "FAILURE"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
			return
		}
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Kredensial tidak valid")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memproses login")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		if err := c.recordLoginAttempt(&userID, identityHash, sourceHash, "FAILURE"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
			return
		}
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Kredensial tidak valid")
		return
	}

	if status != "ACTIVE" {
		if err := c.recordLoginAttempt(&userID, identityHash, sourceHash, "FAILURE"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
			return
		}
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akun Anda saat ini dinonaktifkan")
		return
	}

	if err := c.recordLoginAttempt(&userID, identityHash, sourceHash, "SUCCESS"); err != nil {
		common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan autentikasi tidak tersedia. Coba lagi nanti.")
		return
	}

	rows, err := c.db.Query(`
		SELECT ra.id, ra.role, c.slug, ra.semester_id, ra.course_offering_id, COALESCE(co.display_name, '')
		FROM role_assignments ra
		LEFT JOIN classes c ON ra.class_id = c.id
		LEFT JOIN course_offerings co ON ra.course_offering_id = co.id
		WHERE ra.user_id = ? AND ra.status = 'ACTIVE'
		  AND julianday(ra.valid_from) <= julianday('now')
		  AND (ra.valid_until IS NULL OR julianday(ra.valid_until) > julianday('now'));
	`, userID)

	var assignments []RoleAssignmentItem
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a RoleAssignmentItem
			var slug sql.NullString
			var semID, offID sql.NullInt64
			var offName string
			if err := rows.Scan(&a.ID, &a.Role, &slug, &semID, &offID, &offName); err == nil {
				if slug.Valid {
					a.ClassSlug = slug.String
				}
				if semID.Valid {
					a.SemesterID = &semID.Int64
				}
				if offID.Valid {
					a.OfferingID = &offID.Int64
				}
				a.OfferingName = offName
				assignments = append(assignments, a)
			}
		}
	}

	if len(assignments) == 0 {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Pengguna tidak memiliki penugasan peran aktif")
		return
	}

	var activeAssignmentID sql.NullInt64
	needContextChoice := len(assignments) > 1
	activeRole := assignments[0].Role
	activeAssignmentID = sql.NullInt64{Int64: assignments[0].ID, Valid: true}

	absTTL := 24 * time.Hour
	if activeRole == "SYSTEM_ADMIN" {
		absTTL = 8 * time.Hour
	}
	expiresAt := time.Now().Add(absTTL)

	token, err := generateSecureToken()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token autentikasi")
		return
	}
	tokenHash := computeHash(token)

	_, err = c.db.Exec(`
		INSERT INTO user_sessions (
			user_id, active_role_assignment_id, token_hash, session_version,
			created_at, last_seen_at, absolute_expires_at
		)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?);
	`, userID, activeAssignmentID, tokenHash, sessionVersion, expiresAt.UTC().Format(time.RFC3339))
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan sesi login")
		return
	}

	_, _ = c.db.Exec(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?;`, userID)

	noStore(w)
	c.setAuthCookie(w, token, expiresAt)

	common.WriteV1Success(w, http.StatusOK, LoginResponse{
		Token:             token,
		TokenType:         "Bearer",
		ExpiresAt:         expiresAt,
		Assignments:       assignments,
		NeedContextChoice: needContextChoice,
	})
}

// Logout menangani POST /api/v1/auth/logout
func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	_, _ = c.db.Exec(`
		UPDATE user_sessions
		SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'USER_LOGOUT'
		WHERE id = ?;
	`, u.SessionID)

	c.clearAuthCookie(w)
	common.WriteV1Success(w, http.StatusOK, map[string]bool{"revoked": true})
}

// GetMe menangani GET /api/v1/auth/me
func (c *AuthController) GetMe(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var classes []map[string]any
	rows, err := c.db.Query(`
		SELECT c.slug, c.code, c.study_program, c.cohort_year, c.group_label, c.status
		FROM classes c
		ORDER BY c.code;
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slug, code, prog, grp, st string
			var cohort int
			if err := rows.Scan(&slug, &code, &prog, &cohort, &grp, &st); err == nil {
				classes = append(classes, map[string]any{
					"slug":    slug,
					"code":    code,
					"program": prog,
					"cohort":  cohort,
					"group":   grp,
					"status":  st,
				})
			}
		}
	}

	userData := map[string]any{
		"id":           u.UserID,
		"identity_key": u.IdentityKey,
		"display_name": u.DisplayName,
		"status":       u.UserStatus,
	}

	var activeAssignmentData any
	if u.ActiveAssignmentID > 0 {
		var offeringName string
		if u.ActiveCourseOfferingID.Valid {
			_ = c.db.QueryRow(`SELECT display_name FROM course_offerings WHERE id = ?;`, u.ActiveCourseOfferingID.Int64).Scan(&offeringName)
		}
		activeAssignmentData = map[string]any{
			"id":         u.ActiveAssignmentID,
			"role":       u.ActiveRole,
			"scope_type": u.ActiveScopeType,
			"class_id":   u.ActiveClassID.Int64,
			"class_slug": u.ActiveClassSlug,
			"semester_id": func() any {
				if u.ActiveSemesterID.Valid {
					return u.ActiveSemesterID.Int64
				}
				return nil
			}(),
			"offering_id": func() any {
				if u.ActiveCourseOfferingID.Valid {
					return u.ActiveCourseOfferingID.Int64
				}
				return nil
			}(),
			"offering_name": offeringName,
		}
	}

	assignments := make([]RoleAssignmentItem, 0)
	assignmentRows, err := c.db.QueryContext(r.Context(), `
		SELECT ra.id, ra.role, c.slug, ra.semester_id, ra.course_offering_id,
		       COALESCE(co.display_name, '')
		FROM role_assignments ra
		LEFT JOIN classes c ON c.id = ra.class_id
		LEFT JOIN course_offerings co ON co.id = ra.course_offering_id
		WHERE ra.user_id = ? AND ra.status = 'ACTIVE'
		  AND (ra.valid_from IS NULL OR julianday(ra.valid_from) <= julianday('now'))
		  AND (ra.valid_until IS NULL OR julianday(ra.valid_until) > julianday('now'))
		ORDER BY ra.id`, u.UserID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat penugasan aktif")
		return
	}
	defer assignmentRows.Close()
	for assignmentRows.Next() {
		var item RoleAssignmentItem
		var classSlug sql.NullString
		var semesterID, offeringID sql.NullInt64
		if err := assignmentRows.Scan(&item.ID, &item.Role, &classSlug, &semesterID, &offeringID, &item.OfferingName); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca penugasan aktif")
			return
		}
		if classSlug.Valid {
			item.ClassSlug = classSlug.String
		}
		if semesterID.Valid {
			item.SemesterID = &semesterID.Int64
		}
		if offeringID.Valid {
			item.OfferingID = &offeringID.Int64
		}
		assignments = append(assignments, item)
	}
	if err := assignmentRows.Err(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat penugasan aktif")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user":              userData,
		"active_assignment": activeAssignmentData,
		"assignments":       assignments,
		"classes":           classes,
	})
}

// RequestRecovery handles POST /api/v1/auth/recovery/request. Its accepted
// response is deliberately identical for known and unknown identities.
func (c *AuthController) RequestRecovery(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	var req RecoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.IdentityKey) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "identity_key wajib diisi")
		return
	}

	identity := auth.NormalizeIdentity(req.IdentityKey)
	source := c.clientSource(r)
	trusted := []string(nil)
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trusted, ratelimit.PolicyRecoveryRequest, identity) {
		return
	}
	recordAttempt := func() bool {
		if c.rlManager == nil || c.rlManager.Limiter() == nil {
			return true
		}
		if err := c.rlManager.Limiter().Record(r.Context(), ratelimit.PolicyRecoveryRequest, identity, source, "FAILURE"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return false
		}
		return true
	}
	accepted := func() {
		common.WriteV1Success(w, http.StatusAccepted, map[string]string{
			"message": "Jika akun ditemukan, petunjuk pemulihan akan dikirim melalui WhatsApp.",
		})
	}

	if c.recoveryService == nil {
		if !recordAttempt() {
			return
		}
		accepted()
		return
	}
	token, err := c.recoveryService.RequestRecovery(r.Context(), identity, "WHATSAPP", "self-service WhatsApp recovery")
	if err != nil {
		if !errors.Is(err, auth.ErrAuthenticationFailed) && !errors.Is(err, auth.ErrTooFrequent) && !errors.Is(err, auth.ErrInvalidInput) {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return
		}
		if !recordAttempt() {
			return
		}
		accepted()
		return
	}

	delivered := false
	if c.recoverySender != nil && auth.IsValidPhoneIdentity(identity) {
		jid := strings.TrimPrefix(identity, "+") + "@s.whatsapp.net"
		resetPath := "/login.html?mode=recovery&token=" + url.QueryEscape(token)
		resetURL := resetPath
		if c.publicBaseURL != "" {
			resetURL = c.publicBaseURL + resetPath
		}
		message := "Permintaan pemulihan kata sandi Bot Jadwal diterima. Buka tautan berikut dalam 1 jam:\n" + resetURL + "\n\nAbaikan pesan ini jika Anda tidak meminta pemulihan."
		_, sendErr := c.recoverySender.SendText(r.Context(), jid, message)
		delivered = sendErr == nil
	}
	if !delivered {
		if err := c.recoveryService.InvalidateRecoveryToken(r.Context(), token); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return
		}
	}
	if !recordAttempt() {
		return
	}
	accepted()
}

// ConfirmRecovery handles POST /api/v1/auth/recovery/confirm.
func (c *AuthController) ConfirmRecovery(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	var req RecoveryConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" || req.NewPassword == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "token dan new_password wajib diisi")
		return
	}
	source := c.clientSource(r)
	subject := computeHash(req.Token)
	trusted := []string(nil)
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trusted, ratelimit.PolicyRecoveryConfirm, subject) {
		return
	}
	if c.recoveryService == nil {
		common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan pemulihan tidak tersedia")
		return
	}
	err := c.recoveryService.ConfirmRecovery(r.Context(), req.Token, req.NewPassword)
	outcome := "SUCCESS"
	if err != nil {
		outcome = "FAILURE"
	}
	if c.rlManager != nil && c.rlManager.Limiter() != nil {
		if recordErr := c.rlManager.Limiter().Record(r.Context(), ratelimit.PolicyRecoveryConfirm, subject, source, outcome); recordErr != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return
		}
	}
	if err != nil {
		if errors.Is(err, auth.ErrInvalidInput) {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Token pemulihan atau kata sandi tidak valid")
			return
		}
		common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]bool{"password_reset": true})
}

// SwitchContext menangani POST /api/v1/auth/switch-context
func (c *AuthController) SwitchContext(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req SwitchContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoleAssignmentID <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "role_assignment_id wajib disertakan")
		return
	}

	var role string
	err := c.db.QueryRow(`
		SELECT role FROM role_assignments
		WHERE id = ? AND user_id = ? AND status = 'ACTIVE'
		  AND julianday(valid_from) <= julianday('now')
		  AND (valid_until IS NULL OR julianday(valid_until) > julianday('now'));
	`, req.RoleAssignmentID, u.UserID).Scan(&role)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Penugasan peran tidak ditemukan atau Anda tidak memiliki akses")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi penugasan peran")
		return
	}

	_, _ = c.db.Exec(`
		UPDATE user_sessions
		SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'SWITCH_CONTEXT'
		WHERE id = ?;
	`, u.SessionID)

	absTTL := 24 * time.Hour
	if role == "SYSTEM_ADMIN" {
		absTTL = 8 * time.Hour
	}
	expiresAt := time.Now().Add(absTTL)

	newToken, err := generateSecureToken()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token baru")
		return
	}
	tokenHash := computeHash(newToken)

	_, err = c.db.Exec(`
		INSERT INTO user_sessions (
			user_id, active_role_assignment_id, token_hash, session_version,
			created_at, last_seen_at, absolute_expires_at
		)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?);
	`, u.UserID, req.RoleAssignmentID, tokenHash, u.SessionVersion, expiresAt.UTC().Format(time.RFC3339))
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan sesi baru")
		return
	}

	noStore(w)
	c.setAuthCookie(w, newToken, expiresAt)

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"token":      newToken,
		"token_type": "Bearer",
		"expires_at": expiresAt,
	})
}

// GetClassesAccess menangani routing akses GET /api/v1/classes (baik via token portal maupun auth pengguna)
func (c *AuthController) GetClassesAccess(w http.ResponseWriter, r *http.Request) {
	if ExtractPortalToken(r) != "" {
		c.GetClasses(w, r)
		return
	}
	middleware.NewAuthManager(c.db).RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.GetClasses))(w, r)
}

// GetClasses menyajikan daftar kelas yang dapat diakses pengguna atau token portal
func (c *AuthController) GetClasses(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var scopedClassID sql.NullInt64
	if u, ok := common.GetAuthContext(r); ok {
		switch u.ActiveRole {
		case "SYSTEM_ADMIN":
		case "KM":
			if !u.ActiveClassID.Valid {
				common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks kelas KM tidak valid")
				return
			}
			scopedClassID = u.ActiveClassID
		default:
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Peran aktif tidak dapat melihat daftar kelas")
			return
		}
	} else {
		portalToken := ExtractPortalToken(r)
		if portalToken == "" {
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi atau token portal diperlukan")
			return
		}
		if c.portalService == nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Layanan portal belum siap")
			return
		}
		classID, err := c.portalService.ResolveSession(r.Context(), portalToken)
		if err != nil {
			if errors.Is(err, portal.ErrInvalidCode) {
				common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi portal tidak valid atau telah kedaluwarsa")
			} else {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi sesi portal")
			}
			return
		}
		scopedClassID = sql.NullInt64{Int64: classID, Valid: true}
	}

	query := `
		SELECT 
			c.slug, c.code, c.study_program, c.cohort_year, c.group_label, c.status,
			COALESCE(u.display_name, ''),
			COALESCE(u.identity_key, ''),
			COALESCE((
				SELECT ri.invited_identity_key 
				FROM role_invitations ri 
				WHERE ri.class_id = c.id AND ri.role = 'KM' AND ri.status = 'PENDING' AND ri.expires_at > CURRENT_TIMESTAMP
				ORDER BY ri.created_at DESC LIMIT 1
			), '')
		FROM classes c
		LEFT JOIN (
			SELECT class_id, user_id 
			FROM role_assignments 
			WHERE role = 'KM' AND status = 'ACTIVE' 
			GROUP BY class_id
		) ra ON ra.class_id = c.id
		LEFT JOIN users u ON u.id = ra.user_id AND u.status = 'ACTIVE'`
	args := []any{}
	if scopedClassID.Valid {
		query += ` WHERE c.id = ?`
		args = append(args, scopedClassID.Int64)
	}
	query += ` ORDER BY c.code`

	rows, err := c.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengambil daftar kelas")
		return
	}
	defer rows.Close()

	classes := make([]map[string]any, 0)
	for rows.Next() {
		var slug, code, prog, grp, st, kmName, kmPhone, pendingPhone string
		var cohort int
		if err := rows.Scan(&slug, &code, &prog, &cohort, &grp, &st, &kmName, &kmPhone, &pendingPhone); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca daftar kelas")
			return
		}
		statusKM := "none"
		if kmName != "" {
			statusKM = "active"
		} else if pendingPhone != "" {
			statusKM = "pending"
		}
		classes = append(classes, map[string]any{
			"slug":          slug,
			"code":          code,
			"program":       prog,
			"cohort":        cohort,
			"group":         grp,
			"status":        st,
			"km_name":       kmName,
			"km_phone":      kmPhone,
			"status_km":     statusKM,
			"pending_phone": pendingPhone,
		})
	}
	if err := rows.Err(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca daftar kelas")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"classes": classes,
	})
}

// CreateClassRequest payload pembuatan kelas baru oleh System Admin
type CreateClassRequest struct {
	Name         string `json:"name"`
	Code         string `json:"code"`
	Slug         string `json:"slug"`
	StudyProgram string `json:"study_program"`
	CohortYear   int    `json:"cohort_year"`
	GroupLabel   string `json:"group_label"`
}

// CreateClass menangani POST /api/v1/classes
func (c *AuthController) CreateClass(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang membuat kelas baru")
		return
	}

	var req CreateClassRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	name := strings.TrimSpace(req.Name)
	code := strings.TrimSpace(req.Code)
	if code == "" && name != "" {
		code = strings.ToUpper(strings.ReplaceAll(name, " ", "-"))
	}
	if code == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nama atau kode kelas wajib diisi")
		return
	}

	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		slug = strings.ToLower(code)
	}

	prog := strings.TrimSpace(req.StudyProgram)
	if prog == "" {
		prog = "Teknik Informatika"
	}

	cohort := req.CohortYear
	if cohort == 0 {
		cohort = time.Now().Year()
	}

	group := strings.ToUpper(strings.TrimSpace(req.GroupLabel))
	if group == "" {
		parts := strings.Split(code, "-")
		if len(parts) > 1 && len(parts[len(parts)-1]) == 1 {
			group = parts[len(parts)-1]
		} else {
			group = "A"
		}
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi pembuatan kelas")
		return
	}
	defer tx.Rollback()

	var classID int64
	err = tx.QueryRow(`
		INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE')
		RETURNING id;
	`, code, slug, prog, cohort, group).Scan(&classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Kelas dengan kode atau kombinasi prodi/angkatan/grup sudah ada")
		return
	}

	_, err = tx.Exec(`
		INSERT INTO class_settings (class_id, timezone, portal_access_mode, portal_code_version, replacement_reminder_minutes, version)
		VALUES (?, 'Asia/Jakarta', 'LINK', 1, 60, 1)
		ON CONFLICT(class_id) DO NOTHING;
	`, classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan pengaturan default kelas")
		return
	}

	uid := u.UserID
	var raid *int64
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	afterJSON := fmt.Sprintf(`{"code":%q,"slug":%q,"study_program":%q,"cohort_year":%d,"group_label":%q,"status":"ACTIVE"}`, code, slug, prog, cohort, group)
	correlationID := fmt.Sprintf("create-class-%d-%d", classID, time.Now().UnixNano())
	_ = audit.Write(r.Context(), tx, audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		ClassID:       &classID,
		Action:        "CREATE_CLASS",
		EntityType:    "CLASS",
		EntityID:      &classID,
		AfterJSON:     &afterJSON,
		CorrelationID: correlationID,
	})

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit pembuatan kelas")
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":        classID,
		"code":      code,
		"slug":      slug,
		"program":   prog,
		"cohort":    cohort,
		"group":     group,
		"status":    "ACTIVE",
		"status_km": "none",
	})
}

// PatchClassStatus menangani PATCH /api/v1/classes/{slug}
func (c *AuthController) PatchClassStatus(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	slug := r.PathValue("slug")
	if strings.TrimSpace(slug) == "" {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Slug kelas tidak valid")
		return
	}

	var req UpdateClassStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	newStatus := strings.ToUpper(strings.TrimSpace(req.Status))
	if newStatus != "ACTIVE" && newStatus != "INACTIVE" && newStatus != "ARCHIVED" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Status harus salah satu dari: ACTIVE, INACTIVE, ARCHIVED")
		return
	}

	var classID int64
	var oldStatus string
	err := c.db.QueryRow(`SELECT id, status FROM classes WHERE slug = ?;`, slug).Scan(&classID, &oldStatus)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi kelas")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		if u.ActiveRole != "KM" || !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM kelas ini atau System Admin yang berwenang mengubah status kelas")
			return
		}
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi status kelas")
		return
	}
	defer tx.Rollback()
	_, err = tx.Exec(`UPDATE classes SET status = ? WHERE id = ?;`, newStatus, classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status kelas")
		return
	}

	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, oldStatus)
		afterJSON := fmt.Sprintf(`{"status":%q}`, newStatus)
		correlationID := fmt.Sprintf("class-status-%d-%d", classID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "UPDATE_CLASS_STATUS",
			EntityType:    "CLASS",
			EntityID:      &classID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit status kelas")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit status kelas")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"slug":   slug,
		"status": newStatus,
	})
}

// CreateInvitation menangani POST /api/v1/invitations
func (c *AuthController) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req InvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role != "KM" && role != "PJ" && role != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Peran undangan harus KM, PJ, atau SYSTEM_ADMIN")
		return
	}
	if role == "SYSTEM_ADMIN" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang mengundang System Admin")
		return
	}

	cleanIdentity := auth.NormalizeIdentity(req.InvitedIdentityKey)
	if cleanIdentity == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nomor WhatsApp (invited_identity_key) wajib diisi")
		return
	}
	if !auth.IsValidPhoneIdentity(cleanIdentity) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nomor WhatsApp tidak valid (gunakan format 08... atau +62...)")
		return
	}

	// Undangan System Admin berscope GLOBAL tanpa kelas (BE-003).
	var classID sql.NullInt64
	var scopeType string
	switch role {
	case "SYSTEM_ADMIN":
		if strings.TrimSpace(req.ClassSlug) != "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Undangan System Admin tidak memakai kelas")
			return
		}
		scopeType = "GLOBAL"
	default:
		var cid int64
		err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, req.ClassSlug).Scan(&cid)
		if err != nil {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
			return
		}
		classID = sql.NullInt64{Int64: cid, Valid: true}
	}

	if u.ActiveRole == "KM" && role != "SYSTEM_ADMIN" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID.Int64 {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang membuat undangan untuk kelasnya sendiri")
		return
	}

	if role == "PJ" {
		scopeType = "COURSE_OFFERING"
		if req.SemesterID == nil || req.OfferingID == nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Undangan PJ wajib menyertakan semester_id dan offering_id")
			return
		}
	} else if role != "SYSTEM_ADMIN" {
		scopeType = "CLASS"
	}

	var scopeClassVal, scopeSemVal, scopeOffVal any
	if classID.Valid {
		scopeClassVal = classID.Int64
	}
	if req.SemesterID != nil {
		scopeSemVal = *req.SemesterID
	}
	if req.OfferingID != nil {
		scopeOffVal = *req.OfferingID
	}

	var alreadyActive bool
	if err := c.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM users u
			JOIN role_assignments ra ON ra.user_id = u.id
			WHERE u.identity_key = ? AND ra.role = ? AND ra.scope_type = ?
			AND COALESCE(ra.class_id,0)=COALESCE(?,0)
			AND COALESCE(ra.semester_id,0)=COALESCE(?,0)
			AND COALESCE(ra.course_offering_id,0)=COALESCE(?,0)
			AND ra.status = 'ACTIVE'
		);
	`, cleanIdentity, role, scopeType, scopeClassVal, scopeSemVal, scopeOffVal).Scan(&alreadyActive); err == nil && alreadyActive {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Nomor sudah terdaftar aktif sebagai "+role+" pada cakupan ini")
		return
	}

	_, _ = c.db.Exec(`
		UPDATE role_invitations
		SET status = 'REVOKED'
		WHERE invited_identity_key = ? AND role = ? AND status = 'PENDING'
		AND COALESCE(class_id,0)=COALESCE(?,0)
		AND COALESCE(semester_id,0)=COALESCE(?,0)
		AND COALESCE(course_offering_id,0)=COALESCE(?,0);
	`, cleanIdentity, role, scopeClassVal, scopeSemVal, scopeOffVal)

	token, err := generateSecureToken()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token undangan")
		return
	}
	tokenHash := computeHash(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	var invID int64
	err = c.db.QueryRow(`
		INSERT INTO role_invitations (
			token_hash, invited_identity_key, role, scope_type, class_id, semester_id,
			course_offering_id, status, expires_at, invited_by_user_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		RETURNING id;
	`, tokenHash, cleanIdentity, role, scopeType, classID, req.SemesterID, req.OfferingID, expiresAt, u.UserID).Scan(&invID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan undangan: %v", err))
		return
	}

	noStore(w)
	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"invitation_id": invID,
		"token":         token,
		"expires_at":    expiresAt,
	})
}

// AcceptInvitation menangani POST /api/v1/invitations/accept
func (c *AuthController) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Token undangan wajib disertakan")
		return
	}

	if len(req.Password) < 12 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kata sandi minimal 12 karakter")
		return
	}

	inviteSubject := "invite:" + strings.TrimSpace(req.Token)
	inviteSource := c.clientSource(r)
	var trusted []string
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	if c.rlManager != nil {
		if !c.rlManager.CheckSensitiveLimit(w, r, trusted, ratelimit.PolicyInviteAccept, inviteSubject) {
			return
		}
	}

	recordInviteFailure := func() bool {
		if c.rlManager == nil || c.rlManager.Limiter() == nil {
			return true
		}
		if err := c.rlManager.Limiter().Record(r.Context(), ratelimit.PolicyInviteAccept, inviteSubject, inviteSource, "FAILURE"); err != nil {
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return false
		}
		return true
	}

	tokenHash := computeHash(req.Token)

	var (
		invID            int64
		identityKey      string
		role             string
		scopeType        string
		classID          sql.NullInt64
		semesterID       sql.NullInt64
		courseOfferingID sql.NullInt64
		status           string
		expiresAt        common.DBTimestamp
	)

	err := c.db.QueryRow(`
		SELECT id, invited_identity_key, role, scope_type, class_id, semester_id, course_offering_id, status, expires_at
		FROM role_invitations
		WHERE token_hash = ?;
	`, tokenHash).Scan(&invID, &identityKey, &role, &scopeType, &classID, &semesterID, &courseOfferingID, &status, &expiresAt)

	if err == sql.ErrNoRows || status != "PENDING" {
		if !recordInviteFailure() {
			return
		}
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Undangan tidak valid atau telah digunakan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi undangan")
		return
	}

	identityKey = auth.NormalizeIdentity(identityKey)

	if !expiresAt.Valid || time.Now().After(expiresAt.Time) {
		_, _ = c.db.Exec(`UPDATE role_invitations SET status = 'EXPIRED' WHERE id = ?;`, invID)
		if !recordInviteFailure() {
			return
		}
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Undangan telah kedaluwarsa")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Gagal memproses kata sandi")
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = identityKey
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var userID int64
	err = tx.QueryRow(`
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES (?, ?, ?, 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET
			password_hash = excluded.password_hash,
			display_name = excluded.display_name,
			status = 'ACTIVE'
		RETURNING id;
	`, identityKey, displayName, string(hashedPassword)).Scan(&userID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan akun: %v", err))
		return
	}

	var assignmentID int64
	err = tx.QueryRow(`
		INSERT INTO role_assignments (
			user_id, role, scope_type, class_id, semester_id, course_offering_id,
			accepted_invitation_id, status
		)
		SELECT ?, ?, ?, ?, ?, ?, ?, 'ACTIVE'
		WHERE NOT EXISTS(
			SELECT 1 FROM role_assignments
			WHERE user_id = ? AND role = ? AND scope_type = ?
			AND COALESCE(class_id,0)=COALESCE(?,0)
			AND COALESCE(semester_id,0)=COALESCE(?,0)
			AND COALESCE(course_offering_id,0)=COALESCE(?,0)
			AND status = 'ACTIVE'
		)
		RETURNING id;
	`, userID, role, scopeType, classID, semesterID, courseOfferingID, invID,
		userID, role, scopeType, classID, semesterID, courseOfferingID).Scan(&assignmentID)
	if err != nil {
		if err == sql.ErrNoRows || strings.Contains(strings.ToLower(err.Error()), "unique") {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Nomor sudah memiliki peran aktif pada cakupan ini")
			return
		}
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menetapkan peran: %v", err))
		return
	}

	res, err := tx.Exec(`UPDATE role_invitations SET status = 'ACCEPTED', accepted_at = CURRENT_TIMESTAMP WHERE id = ? AND status = 'PENDING';`, invID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status undangan")
		return
	}
	affected, _ := res.RowsAffected()
	if affected != 1 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Undangan telah digunakan")
		return
	}
	correlationID := fmt.Sprintf("accept-invitation-%d-%d", invID, time.Now().UnixNano())
	{
		var classIDPtr *int64
		if classID.Valid {
			v := classID.Int64
			classIDPtr = &v
		}
		var semesterIDPtr *int64
		if semesterID.Valid {
			v := semesterID.Int64
			semesterIDPtr = &v
		}
		afterJSON := fmt.Sprintf(`{"role":%q,"scope_type":%q,"invitation_id":%d}`, role, scopeType, invID)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &userID, RoleAssignmentID: &assignmentID},
			ClassID:       classIDPtr,
			SemesterID:    semesterIDPtr,
			Action:        "ASSIGN_ROLE",
			EntityType:    "ROLE_ASSIGNMENT",
			EntityID:      &assignmentID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit penugasan peran")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyelesaikan proses penerimaan undangan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyInviteAccept, inviteSubject, inviteSource, "SUCCESS")
	}

	// Buat sesi login langsung agar pengguna masuk ke ruang kerja tanpa login ulang.
	var sessionVersion int
	if err := c.db.QueryRow(`SELECT session_version FROM users WHERE id = ?;`, userID).Scan(&sessionVersion); err != nil {
		sessionVersion = 1
	}
	rows, err := c.db.Query(`
		SELECT ra.id, ra.role, c.slug, ra.semester_id, ra.course_offering_id, COALESCE(co.display_name, '')
		FROM role_assignments ra
		LEFT JOIN classes c ON ra.class_id = c.id
		LEFT JOIN course_offerings co ON ra.course_offering_id = co.id
		WHERE ra.user_id = ? AND ra.status = 'ACTIVE';
	`, userID)
	assignments := []RoleAssignmentItem{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a RoleAssignmentItem
			var slug sql.NullString
			var semID, offID sql.NullInt64
			var offName string
			if err := rows.Scan(&a.ID, &a.Role, &slug, &semID, &offID, &offName); err == nil {
				if slug.Valid {
					a.ClassSlug = slug.String
				}
				if semID.Valid {
					a.SemesterID = &semID.Int64
				}
				if offID.Valid {
					a.OfferingID = &offID.Int64
				}
				a.OfferingName = offName
				assignments = append(assignments, a)
			}
		}
	}

	absTTL := 24 * time.Hour
	if role == "SYSTEM_ADMIN" {
		absTTL = 8 * time.Hour
	}
	sessionExpiresAt := time.Now().Add(absTTL)

	token, err := generateSecureToken()
	if err != nil {
		common.WriteV1Success(w, http.StatusOK, map[string]any{
			"user_id":       userID,
			"assignment_id": assignmentID,
		})
		return
	}
	if _, err := c.db.Exec(`
		INSERT INTO user_sessions (
			user_id, active_role_assignment_id, token_hash, session_version,
			created_at, last_seen_at, absolute_expires_at
		)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?);
	`, userID, assignmentID, computeHash(token), sessionVersion, sessionExpiresAt.UTC().Format(time.RFC3339)); err != nil {
		common.WriteV1Success(w, http.StatusOK, map[string]any{
			"user_id":       userID,
			"assignment_id": assignmentID,
		})
		return
	}
	_, _ = c.db.Exec(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?;`, userID)

	noStore(w)
	c.setAuthCookie(w, token, sessionExpiresAt)

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user_id":             userID,
		"assignment_id":       assignmentID,
		"token":               token,
		"token_type":          "Bearer",
		"expires_at":          sessionExpiresAt.UTC().Format(time.RFC3339),
		"assignments":         assignments,
		"need_context_choice": len(assignments) > 1,
	})
}
