package v1

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
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

// AuthController mengelola seluruh endpoint autentikasi, sesi pengurus, hak akses kelas, dan undangan
type AuthController struct {
	db            *sql.DB
	secManager    *middleware.SecurityManager
	rlManager     *middleware.RateLimitManager
	portalService *portal.Service
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

	cleanIdentity := strings.ToLower(strings.TrimSpace(req.IdentityKey))
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
		WHERE ra.user_id = ? AND ra.status = 'ACTIVE';
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

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user":              userData,
		"active_assignment": activeAssignmentData,
		"classes":           classes,
	})
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
		WHERE id = ? AND user_id = ? AND status = 'ACTIVE';
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
		SELECT slug, code, study_program, cohort_year, group_label, status
		FROM classes`
	args := []any{}
	if scopedClassID.Valid {
		query += ` WHERE id = ?`
		args = append(args, scopedClassID.Int64)
	}
	query += ` ORDER BY code`

	rows, err := c.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengambil daftar kelas")
		return
	}
	defer rows.Close()

	classes := make([]map[string]any, 0)
	for rows.Next() {
		var slug, code, prog, grp, st string
		var cohort int
		if err := rows.Scan(&slug, &code, &prog, &cohort, &grp, &st); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca daftar kelas")
			return
		}
		classes = append(classes, map[string]any{
			"slug":    slug,
			"code":    code,
			"program": prog,
			"cohort":  cohort,
			"group":   grp,
			"status":  st,
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
	if role != "KM" && role != "PJ" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Peran undangan harus KM atau PJ")
		return
	}

	cleanIdentity := strings.TrimSpace(req.InvitedIdentityKey)
	if cleanIdentity == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nomor WhatsApp (invited_identity_key) wajib diisi")
		return
	}

	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, req.ClassSlug).Scan(&classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang membuat undangan untuk kelasnya sendiri")
		return
	}

	scopeType := "CLASS"
	if role == "PJ" {
		scopeType = "COURSE_OFFERING"
		if req.SemesterID == nil || req.OfferingID == nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Undangan PJ wajib menyertakan semester_id dan offering_id")
			return
		}
	}

	_, _ = c.db.Exec(`
		UPDATE role_invitations
		SET status = 'REVOKED'
		WHERE invited_identity_key = ? AND status = 'PENDING';
	`, cleanIdentity)

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

	if len(req.Password) < 6 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kata sandi minimal 6 karakter")
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
		VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		RETURNING id;
	`, userID, role, scopeType, classID, semesterID, courseOfferingID, invID).Scan(&assignmentID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menetapkan peran: %v", err))
		return
	}

	res, err := tx.Exec(`UPDATE role_invitations SET status = 'ACCEPTED' WHERE id = ? AND status = 'PENDING';`, invID)
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

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"assignment_id": assignmentID,
	})
}
