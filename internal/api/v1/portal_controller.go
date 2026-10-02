package v1

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/ratelimit"
)

// CreatePortalSessionRequest merepresentasikan payload pembuatan sesi portal mahasiswa
type CreatePortalSessionRequest struct {
	Code string `json:"code"`
}

// RotatePortalCodeRequest merepresentasikan payload rotasi kode portal oleh pengelola
type RotatePortalCodeRequest struct {
	Code string `json:"code"`
}

// PortalController melayani seluruh endpoint publik/mahasiswa dan rotasi kode portal
type PortalController struct {
	db            *sql.DB
	portalService *portal.Service
	rlManager     *middleware.RateLimitManager
	secManager    *middleware.SecurityManager
}

// NewPortalController membuat instance PortalController baru
func NewPortalController(db *sql.DB, portalService *portal.Service, rlManager *middleware.RateLimitManager, secManager *middleware.SecurityManager) *PortalController {
	return &PortalController{
		db:            db,
		portalService: portalService,
		rlManager:     rlManager,
		secManager:    secManager,
	}
}

// RegisterRoutes mendaftarkan seluruh endpoint portal ke ServeMux
func (c *PortalController) RegisterRoutes(mux *http.ServeMux, auth *middleware.AuthManager) {
	mux.HandleFunc("POST /api/v1/portal/{slug}/session", c.CreateSession)
	mux.HandleFunc("GET /api/v1/portal/{slug}/summary", c.Summary)
	mux.HandleFunc("GET /api/v1/portal/{slug}/schedule", c.Schedule)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks", c.Tasks)
	mux.HandleFunc("GET /api/v1/portal/{slug}/tasks/{id}", c.TaskDetail)
	mux.HandleFunc("GET /api/v1/portal/{slug}/changes", c.Changes)
	mux.HandleFunc("GET /api/v1/portal/{slug}/materials", c.Materials)
	mux.HandleFunc("POST /api/v1/classes/{slug}/portal-code/rotate", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.RotateCode)))
}

func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// ExtractPortalToken mengekstrak token portal dari header atau query param.
func ExtractPortalToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-Portal-Token"))
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("portal_token"))
	}
	return token
}

func (c *PortalController) clientSource(r *http.Request) string {
	var trusted []string
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	return middleware.ClientSource(r, trusted)
}

func (c *PortalController) portalAccessAllowed(w http.ResponseWriter, r *http.Request, classID int64) bool {
	var mode string
	err := c.db.QueryRow(`
		SELECT portal_access_mode
		FROM class_settings
		WHERE class_id = ?;
	`, classID).Scan(&mode)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat pengaturan kelas")
		return false
	}

	if mode == "LINK" {
		return true
	}

	token := ExtractPortalToken(r)

	if token == "" {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Kelas ini memerlukan kode akses portal")
		return false
	}

	if c.portalService == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Layanan portal belum siap")
		return false
	}
	if err := c.portalService.ValidateSession(r.Context(), classID, token); err != nil {
		if !errors.Is(err, portal.ErrInvalidCode) {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi sesi portal")
			return false
		}
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi portal tidak valid atau telah kedaluwarsa")
		return false
	}

	return true
}

// CreateSession menangani POST /api/v1/portal/{slug}/session
func (c *PortalController) CreateSession(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if c.db == nil || c.portalService == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req CreatePortalSessionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "code wajib diisi")
		return
	}

	var classID int64
	err := c.db.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?`, r.PathValue("slug")).Scan(&classID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi kode portal")
			return
		}
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Kode portal tidak valid")
		return
	}

	session, err := c.portalService.VerifyCode(r.Context(), classID, req.Code, c.clientSource(r))
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrRateLimited):
			middleware.LimitExceeded(w, 15*time.Minute)
			return
		case errors.Is(err, portal.ErrUnavailable):
			common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
			return
		case errors.Is(err, portal.ErrInvalidInput):
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "code wajib diisi")
		case errors.Is(err, portal.ErrInvalidCode), errors.Is(err, portal.ErrNotFound):
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Kode portal tidak valid")
		default:
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membuat sesi portal")
		}
		return
	}

	noStore(w)
	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"portal_token": session.Token,
		"expires_at":   session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

// RotateCode menangani POST /api/v1/classes/{slug}/portal-code/rotate
func (c *PortalController) RotateCode(w http.ResponseWriter, r *http.Request) {
	if c.db == nil || c.portalService == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req RotatePortalCodeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	var classID int64
	err := c.db.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?`, r.PathValue("slug")).Scan(&classID)
	if errors.Is(err, sql.ErrNoRows) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Anda tidak dapat merotasi kode portal kelas lain")
		return
	}

	rotateSubject := fmt.Sprintf("rotate:%d:class:%d", u.UserID, classID)
	var trusted []string
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	rotateSource := middleware.ClientSource(r, trusted)
	if c.rlManager != nil {
		if !c.rlManager.CheckSensitiveLimit(w, r, trusted, ratelimit.PolicyPortalRotate, rotateSubject) {
			return
		}
	}

	result, err := c.portalService.RotateCode(r.Context(), portal.RotationRequest{
		ClassID:             classID,
		Code:                req.Code,
		ActorUserID:         u.UserID,
		ActorRoleAssignment: u.ActiveAssignmentID,
	})
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrInvalidInput):
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "code harus terdiri dari 6 sampai 128 karakter, atau kosong untuk dibuat otomatis")
		case errors.Is(err, portal.ErrNotFound):
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pengaturan portal kelas tidak ditemukan")
		case errors.Is(err, portal.ErrConflict):
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Pengaturan portal berubah. Muat ulang lalu coba kembali")
		default:
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal merotasi kode portal")
		}
		return
	}

	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyPortalRotate, rotateSubject, rotateSource, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"portal_code":         result.Code,
		"portal_code_version": result.Version,
		"portal_access_mode":  "CODE",
		"reveal_once":         true,
	})
}

// GetClassSettings menangani GET /api/v1/classes/{slug}/settings (BE-005).
// KM hanya kelasnya; System Admin global. Tanpa audit (operasi baca).
func (c *PortalController) GetClassSettings(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	var out struct {
		ClassID           int64          `json:"-"`
		Timezone          string         `json:"timezone"`
		PortalAccessMode  string         `json:"portal_access_mode"`
		PortalCodeVersion int            `json:"portal_code_version"`
		Morning           sql.NullString `json:"-"`
		Afternoon         sql.NullString `json:"-"`
		Replacement       sql.NullInt64  `json:"-"`
		Version           int            `json:"-"`
	}
	err := c.db.QueryRow(`
		SELECT c.id, cs.timezone, cs.portal_access_mode, cs.portal_code_version,
		       cs.morning_reminder_time, cs.afternoon_reminder_time,
		       cs.replacement_reminder_minutes, cs.version
		FROM classes c
		JOIN class_settings cs ON cs.class_id = c.id
		WHERE c.slug = ?;
	`, slug).Scan(&out.ClassID, &out.Timezone, &out.PortalAccessMode, &out.PortalCodeVersion,
		&out.Morning, &out.Afternoon, &out.Replacement, &out.Version)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat pengaturan kelas")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != out.ClassID) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	resp := map[string]any{
		"slug":                slug,
		"timezone":            out.Timezone,
		"portal_access_mode":  out.PortalAccessMode,
		"portal_code_version": out.PortalCodeVersion,
		"version":             out.Version,
	}
	if out.Morning.Valid {
		resp["morning_reminder_time"] = out.Morning.String
	}
	if out.Afternoon.Valid {
		resp["afternoon_reminder_time"] = out.Afternoon.String
	}
	if out.Replacement.Valid {
		resp["replacement_reminder_minutes"] = out.Replacement.Int64
	}
	common.WriteV1Success(w, http.StatusOK, resp)
}

// UpdateClassSettingsRequest adalah payload ubah pengaturan kelas.
type UpdateClassSettingsRequest struct {
	Version                    *int    `json:"version,omitempty"`
	Timezone                   *string `json:"timezone,omitempty"`
	MorningReminderTime        *string `json:"morning_reminder_time,omitempty"`
	AfternoonReminderTime      *string `json:"afternoon_reminder_time,omitempty"`
	ReplacementReminderMinutes *int    `json:"replacement_reminder_minutes,omitempty"`
}

// UpdateClassSettings menangani PATCH /api/v1/classes/{slug}/settings.
func (c *PortalController) UpdateClassSettings(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang mengubah pengaturan kelas")
		return
	}
	slug := strings.TrimSpace(r.PathValue("slug"))
	var req UpdateClassSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	var classID int64
	var curVersion int
	var curTz string
	var curMorning, curAfternoon sql.NullString
	var curRepl sql.NullInt64
	if err := c.db.QueryRow(`SELECT c.id, cs.version, cs.timezone, cs.morning_reminder_time, cs.afternoon_reminder_time, cs.replacement_reminder_minutes
		FROM classes c JOIN class_settings cs ON cs.class_id = c.id WHERE c.slug = ?;`, slug).
		Scan(&classID, &curVersion, &curTz, &curMorning, &curAfternoon, &curRepl); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca pengaturan kelas")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if req.Version != nil && *req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{"current_version": curVersion})
		return
	}
	sets := []string{}
	args := []any{}
	isHHMM := func(s string) bool {
		if len(s) != 5 || s[2] != ':' {
			return false
		}
		h, herr := strconv.Atoi(s[:2])
		m, merr := strconv.Atoi(s[3:])
		return herr == nil && merr == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
	}
	newTz := curTz
	if req.Timezone != nil {
		tz := strings.TrimSpace(*req.Timezone)
		if _, err := time.LoadLocation(tz); err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "timezone tidak dikenal (format IANA, mis. Asia/Jakarta)")
			return
		}
		sets = append(sets, "timezone = ?")
		args = append(args, tz)
		newTz = tz
	}
	if req.MorningReminderTime != nil {
		t := strings.TrimSpace(*req.MorningReminderTime)
		if !isHHMM(t) {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "morning_reminder_time harus HH:MM")
			return
		}
		sets = append(sets, "morning_reminder_time = ?")
		args = append(args, t)
	}
	if req.AfternoonReminderTime != nil {
		t := strings.TrimSpace(*req.AfternoonReminderTime)
		if !isHHMM(t) {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "afternoon_reminder_time harus HH:MM")
			return
		}
		sets = append(sets, "afternoon_reminder_time = ?")
		args = append(args, t)
	}
	if req.ReplacementReminderMinutes != nil {
		if *req.ReplacementReminderMinutes < 0 || *req.ReplacementReminderMinutes > 1440 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "replacement_reminder_minutes harus 0-1440")
			return
		}
		sets = append(sets, "replacement_reminder_minutes = ?")
		args = append(args, *req.ReplacementReminderMinutes)
	}
	if len(sets) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tidak ada field yang diubah")
		return
	}
	sets = append(sets, "version = version + 1")
	args = append(args, classID, curVersion)
	res, err := c.db.Exec(`UPDATE class_settings SET `+strings.Join(sets, ", ")+` WHERE class_id = ? AND version = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan pengaturan kelas")
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data berubah saat menyimpan")
		return
	}
	beforeJSON := fmt.Sprintf(`{"timezone":%q}`, curTz)
	afterJSON := fmt.Sprintf(`{"timezone":%q}`, newTz)
	uid := u.UserID
	var raid *int64
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	_ = audit.Write(r.Context(), c.db, audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		ClassID:       &classID,
		Action:        "UPDATE_CLASS_SETTINGS",
		EntityType:    "CLASS_SETTINGS",
		EntityID:      &classID,
		BeforeJSON:    &beforeJSON,
		AfterJSON:     &afterJSON,
		CorrelationID: fmt.Sprintf("class-settings-%d-%d", classID, time.Now().UnixNano()),
	})
	common.WriteV1Success(w, http.StatusOK, map[string]any{"slug": slug, "version": curVersion + 1})
}

// SetPortalModeRequest adalah payload ganti mode Portal Kelas.
type SetPortalModeRequest struct {
	Mode   string  `json:"mode"`
	Reason *string `json:"reason,omitempty"`
}

// SetPortalMode menangani PATCH /api/v1/classes/{slug}/portal-mode (BE-005).
// LINK selalu bisa (membersihkan hash kode). CODE wajib sudah punya hash aktif —
// bila belum, putar kode portal dulu (rotate otomatis mengaktifkan CODE).
func (c *PortalController) SetPortalMode(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req SetPortalModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	mode := strings.ToUpper(strings.TrimSpace(req.Mode))
	if mode != "LINK" && mode != "CODE" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Mode harus LINK atau CODE")
		return
	}
	reason := ""
	if req.Reason != nil {
		reason = strings.TrimSpace(*req.Reason)
	}

	slug := strings.TrimSpace(r.PathValue("slug"))
	var classID int64
	var curMode string
	var hasHash bool
	err := c.db.QueryRow(`
		SELECT c.id, cs.portal_access_mode, cs.portal_code_hash IS NOT NULL
		FROM classes c
		JOIN class_settings cs ON cs.class_id = c.id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &curMode, &hasHash)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	if curMode == mode {
		common.WriteV1Success(w, http.StatusOK, map[string]any{
			"slug": slug, "portal_access_mode": curMode, "changed": false,
		})
		return
	}
	if mode == "CODE" && !hasHash {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Belum ada kode aktif. Putar kode portal dulu untuk mengaktifkan mode CODE")
		return
	}

	modeSubject := fmt.Sprintf("portal-mode:%d:class:%d", u.UserID, classID)
	var trusted []string
	if c.secManager != nil {
		trusted = c.secManager.TrustedProxyCIDRs()
	}
	modeSource := middleware.ClientSource(r, trusted)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trusted, ratelimit.PolicyPortalRotate, modeSubject) {
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi mode portal")
		return
	}
	defer tx.Rollback()
	if mode == "LINK" {
		if _, err := tx.Exec(`UPDATE class_settings SET portal_access_mode='LINK',
			portal_code_hash=NULL, portal_code_version=portal_code_version+1, version=version+1,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id=?`, classID); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah mode portal")
			return
		}
	} else {
		if _, err := tx.Exec(`UPDATE class_settings SET portal_access_mode='CODE', version=version+1,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id=?`, classID); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah mode portal")
			return
		}
	}
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"portal_access_mode":%q}`, curMode)
		afterJSON := fmt.Sprintf(`{"portal_access_mode":%q}`, mode)
		correlationID := fmt.Sprintf("portal-mode-%d-%d", classID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "UPDATE_PORTAL_MODE",
			EntityType:    "CLASS",
			EntityID:      &classID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit mode portal")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit mode portal")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyPortalRotate, modeSubject, modeSource, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"slug": slug, "portal_access_mode": mode, "changed": true,
	})
}

// Summary menangani GET /api/v1/portal/{slug}/summary
func (c *PortalController) Summary(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := c.db.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	dateParam := r.URL.Query().Get("date")
	var targetDate time.Time
	if dateParam != "" {
		t, err := time.ParseInLocation("2006-01-02", dateParam, loc)
		if err == nil {
			targetDate = t
		}
	}
	if targetDate.IsZero() {
		targetDate = time.Now().In(loc)
	}
	dateStr := targetDate.Format("2006-01-02")
	dayOfWeek := int(targetDate.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	scheduleItems, _ := c.getScheduleForDate(classID, 0, targetDate, dayOfWeek)

	nowTimeStr := targetDate.Format("15:04")
	var nowEvent any
	var nextEvent any

	for _, item := range scheduleItems {
		start := item["starts_at"].(string)
		end := item["ends_at"].(string)

		if nowTimeStr >= start && nowTimeStr <= end {
			nowEvent = item
		} else if nowTimeStr < start && nextEvent == nil {
			nextEvent = item
		}
	}

	var changesToday []map[string]any
	for _, item := range scheduleItems {
		if item["kind"] != "REGULER" {
			changesToday = append(changesToday, item)
		}
	}

	var nearestTasks []map[string]any
	taskRows, err := c.db.Query(`
		SELECT t.id, co.display_name, t.title, t.deadline_at, t.submission_url
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE'
		  AND t.publication_status = 'PUBLISHED'
		  AND t.deleted_at IS NULL
		  AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC
		LIMIT 3;
	`, classID)

	if err == nil {
		defer taskRows.Close()
		for taskRows.Next() {
			var tID int64
			var offering, title string
			var deadlineAt common.DBTimestamp
			var subURL sql.NullString
			if err := taskRows.Scan(&tID, &offering, &title, &deadlineAt, &subURL); err == nil {
				nearestTasks = append(nearestTasks, map[string]any{
					"id":             tID,
					"offering":       offering,
					"title":          title,
					"deadline_at":    deadlineAt.RFC3339(),
					"submission_url": subURL.String,
				})
			}
		}
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"class":         slug,
		"date":          dateStr,
		"now_event":     nowEvent,
		"next_event":    nextEvent,
		"today":         scheduleItems,
		"changes_today": changesToday,
		"nearest_tasks": nearestTasks,
	})
}

// Schedule menangani GET /api/v1/portal/{slug}/schedule
func (c *PortalController) Schedule(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := c.db.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	dateParam := r.URL.Query().Get("date")
	var targetDate time.Time
	if dateParam != "" {
		t, err := time.ParseInLocation("2006-01-02", dateParam, loc)
		if err == nil {
			targetDate = t
		}
	}
	if targetDate.IsZero() {
		targetDate = time.Now().In(loc)
	}
	dateStr := targetDate.Format("2006-01-02")
	dayOfWeek := int(targetDate.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	items, err := c.getScheduleForDate(classID, semesterID, targetDate, dayOfWeek)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat jadwal")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"class": slug,
		"date":  dateStr,
		"items": items,
	})
}

// Tasks menangani GET /api/v1/portal/{slug}/tasks
func (c *PortalController) Tasks(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := c.db.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	query := `
		SELECT t.id, co.display_name, t.title, t.instructions, t.deadline_at,
		       COALESCE(t.submission_text, ''), COALESCE(t.submission_url, ''), t.version
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE sem.class_id = ? AND sem.id = ?
		  AND t.publication_status = 'PUBLISHED'
		  AND t.deleted_at IS NULL
	`
	args := []any{classID, semesterID}

	group := r.URL.Query().Get("group")
	if group != "" && group != "hari_ini" && group != "minggu_ini" && group != "mendatang" && group != "terlewat" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "group harus hari_ini, minggu_ini, mendatang, atau terlewat")
		return
	}
	now := time.Now().In(loc)
	if group == "hari_ini" {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC().Format("2006-01-02T15:04:05Z")
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc).UTC().Format("2006-01-02T15:04:05Z")
		query += " AND t.deadline_at BETWEEN ? AND ?"
		args = append(args, start, end)
	} else if group == "minggu_ini" {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startWeek := now.AddDate(0, 0, -weekday+1)
		endWeek := startWeek.AddDate(0, 0, 6)
		start := time.Date(startWeek.Year(), startWeek.Month(), startWeek.Day(), 0, 0, 0, 0, loc).UTC().Format("2006-01-02T15:04:05Z")
		end := time.Date(endWeek.Year(), endWeek.Month(), endWeek.Day(), 23, 59, 59, 999999999, loc).UTC().Format("2006-01-02T15:04:05Z")
		query += " AND t.deadline_at BETWEEN ? AND ?"
		args = append(args, start, end)
	} else if group == "mendatang" {
		query += " AND t.deadline_at > ?"
		args = append(args, now.UTC().Format("2006-01-02T15:04:05Z"))
	} else if group == "terlewat" {
		query += " AND t.deadline_at < ? AND t.completed_at IS NULL"
		args = append(args, now.UTC().Format("2006-01-02T15:04:05Z"))
	}

	offering := r.URL.Query().Get("offering")
	if offering != "" {
		if offID, err := strconv.ParseInt(offering, 10, 64); err == nil {
			query += " AND co.id = ?"
			args = append(args, offID)
		} else {
			query += " AND co.display_name = ?"
			args = append(args, offering)
		}
	}

	q := r.URL.Query().Get("q")
	if q != "" {
		query += " AND (t.title LIKE '%' || ? || '%' OR t.instructions LIKE '%' || ? || '%')"
		args = append(args, q, q)
	}

	query += " ORDER BY t.deadline_at ASC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat tugas portal")
		return
	}
	defer rows.Close()

	var tasks []map[string]any
	for rows.Next() {
		var id int64
		var offeringName, title, instructions, subText, subURL string
		var deadlineAt common.DBTimestamp
		var version int

		if err := rows.Scan(&id, &offeringName, &title, &instructions, &deadlineAt, &subText, &subURL, &version); err == nil {
			tasks = append(tasks, map[string]any{
				"id":              id,
				"offering":        offeringName,
				"title":           title,
				"instructions":    instructions,
				"deadline_at":     deadlineAt.RFC3339(),
				"submission_text": subText,
				"submission_url":  subURL,
				"version":         version,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, tasks)
}

// TaskDetail menangani GET /api/v1/portal/{slug}/tasks/{id}
func (c *PortalController) TaskDetail(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	taskIDStr := r.PathValue("id")
	taskID, _ := strconv.ParseInt(taskIDStr, 10, 64)

	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}

	var (
		id           int64
		offeringID   int64
		offeringName string
		title        string
		instructions string
		deadlineAt   common.DBTimestamp
		taskType     sql.NullString
		subText      sql.NullString
		subURL       sql.NullString
		version      int
		completedAt  common.DBTimestamp
	)

	err = c.db.QueryRow(`
		SELECT t.id, co.id, co.display_name, t.title, t.instructions, t.deadline_at,
		       t.task_type, t.submission_text, t.submission_url, t.version, t.completed_at
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.id = ? AND sem.class_id = ? AND sem.id = ?
		  AND t.publication_status = 'PUBLISHED' AND t.deleted_at IS NULL;
	`, taskID, classID, semesterID).Scan(
		&id, &offeringID, &offeringName, &title, &instructions, &deadlineAt,
		&taskType, &subText, &subURL, &version, &completedAt,
	)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tugas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat detail tugas")
		return
	}

	var materials []map[string]any
	matRows, err := c.db.Query(`
		SELECT id, title, material_type, url, description
		FROM materials
		WHERE (task_id = ? OR (task_id IS NULL AND course_offering_id = ?))
		  AND status = 'ACTIVE' AND visibility = 'CLASS_ACCESS' AND deleted_at IS NULL;
	`, id, offeringID)
	if err == nil {
		defer matRows.Close()
		for matRows.Next() {
			var mID int64
			var mTitle, mType string
			var mURL, mDesc sql.NullString
			if err := matRows.Scan(&mID, &mTitle, &mType, &mURL, &mDesc); err == nil {
				materials = append(materials, map[string]any{
					"id":            mID,
					"title":         mTitle,
					"material_type": mType,
					"url":           mURL.String,
					"description":   mDesc.String,
				})
			}
		}
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"task": map[string]any{
			"id":              id,
			"offering_id":     offeringID,
			"offering":        offeringName,
			"title":           title,
			"instructions":    instructions,
			"deadline_at":     deadlineAt.RFC3339(),
			"task_type":       taskType.String,
			"submission_text": subText.String,
			"submission_url":  subURL.String,
			"version":         version,
			"is_completed":    completedAt.Valid,
		},
		"materials": materials,
	})
}

// Changes menangani GET /api/v1/portal/{slug}/changes
func (c *PortalController) Changes(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}

	query := `
		SELECT te.id, te.event_kind, co.display_name, te.starts_at, te.ends_at,
		       COALESCE(r.code, ''), COALESCE(te.reason, ''), te.published_at
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE sem.class_id = ? AND sem.id = ? AND te.lifecycle_status = 'PUBLISHED'
	`
	args := []any{classID, semesterID}

	since := r.URL.Query().Get("since")
	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "since harus berformat RFC3339")
			return
		}
		query += " AND te.published_at >= ?"
		args = append(args, since)
	}

	query += " ORDER BY te.published_at DESC LIMIT 50;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat riwayat perubahan")
		return
	}
	defer rows.Close()

	var changes []map[string]any
	for rows.Next() {
		var id int64
		var kind, offering, roomCode, reason string
		var startsAt, endsAt common.DBTimestamp
		var publishedAt common.DBTimestamp

		if err := rows.Scan(&id, &kind, &offering, &startsAt, &endsAt, &roomCode, &reason, &publishedAt); err == nil {
			changes = append(changes, map[string]any{
				"id":           id,
				"event_kind":   kind,
				"offering":     offering,
				"starts_at":    startsAt.RFC3339(),
				"ends_at":      endsAt.RFC3339(),
				"room":         roomCode,
				"reason":       reason,
				"published_at": publishedAt.RFC3339(),
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, changes)
}

// Semesters menangani GET /api/v1/portal/{slug}/semesters
func (c *PortalController) Semesters(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	rows, err := c.db.Query(`
		SELECT id, academic_year, term, starts_on, ends_on, status,
		       COALESCE(published_at, ''), COALESCE(activated_at, ''), COALESCE(archived_at, '')
		FROM semesters
		WHERE class_id = ?
		  AND status IN ('ACTIVE', 'ARCHIVED')
		  AND published_at IS NOT NULL
		ORDER BY starts_on DESC;
	`, classID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat semester")
		return
	}
	defer rows.Close()

	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var year, term, startsOn, endsOn, status, publishedAt, activatedAt, archivedAt string
		if err := rows.Scan(&id, &year, &term, &startsOn, &endsOn, &status, &publishedAt, &activatedAt, &archivedAt); err == nil {
			out = append(out, map[string]any{
				"id": id, "academic_year": year, "term": term,
				"starts_on": startsOn, "ends_on": endsOn, "status": status,
				"published_at": publishedAt, "activated_at": activatedAt, "archived_at": archivedAt,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, out)
}

// Materials menangani GET /api/v1/portal/{slug}/materials
func (c *PortalController) Materials(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !c.portalAccessAllowed(w, r, classID) {
		return
	}

	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}

	query := `
		SELECT m.id, m.title, m.material_type, COALESCE(m.url, ''), COALESCE(m.description, '')
		FROM materials m
		LEFT JOIN tasks mt ON mt.id = m.task_id
		LEFT JOIN course_offerings co ON co.id = COALESCE(m.course_offering_id, mt.course_offering_id)
		WHERE m.class_id = ?
		  AND m.status = 'ACTIVE'
		  AND m.visibility = 'CLASS_ACCESS'
		  AND m.deleted_at IS NULL
		  AND (m.task_id IS NULL OR mt.publication_status = 'PUBLISHED')
		  AND (co.semester_id = ? OR (m.course_offering_id IS NULL AND m.task_id IS NULL))
	`
	args := []any{classID, semesterID}

	offering := r.URL.Query().Get("offering")
	if offering != "" {
		if offID, err := strconv.ParseInt(offering, 10, 64); err == nil {
			query += " AND co.id = ?"
			args = append(args, offID)
		} else {
			query += " AND co.display_name = ?"
			args = append(args, offering)
		}
	}

	query += " ORDER BY m.created_at DESC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat materi kelas")
		return
	}
	defer rows.Close()

	var materials []map[string]any
	for rows.Next() {
		var id int64
		var title, matType, urlStr, desc string
		if err := rows.Scan(&id, &title, &matType, &urlStr, &desc); err == nil {
			materials = append(materials, map[string]any{
				"id":            id,
				"title":         title,
				"material_type": matType,
				"url":           urlStr,
				"description":   desc,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, materials)
}

// resolvePortalSemester memilih semester aktif secara default dan hanya menerima
// semester yang pernah dipublikasikan ketika semester_id diberikan eksplisit.
func (c *PortalController) resolvePortalSemester(w http.ResponseWriter, r *http.Request, classID int64) (int64, bool) {
	rawID := strings.TrimSpace(r.URL.Query().Get("semester_id"))
	if rawID == "" {
		var semesterID int64
		err := c.db.QueryRow(`
			SELECT id
			FROM semesters
			WHERE class_id = ? AND status = 'ACTIVE' AND published_at IS NOT NULL
		`, classID).Scan(&semesterID)
		if errors.Is(err, sql.ErrNoRows) {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester aktif tidak ditemukan")
			return 0, false
		}
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat semester")
			return 0, false
		}
		return semesterID, true
	}

	semesterID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || semesterID <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "semester_id harus berupa bilangan bulat positif")
		return 0, false
	}

	err = c.db.QueryRow(`
		SELECT id
		FROM semesters
		WHERE id = ? AND class_id = ?
		  AND status IN ('ACTIVE', 'ARCHIVED')
		  AND published_at IS NOT NULL
	`, semesterID, classID).Scan(&semesterID)
	if errors.Is(err, sql.ErrNoRows) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Semester tidak ditemukan")
		return 0, false
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat semester")
		return 0, false
	}

	return semesterID, true
}

func (c *PortalController) getScheduleForDate(classID, semesterID int64, targetDate time.Time, dayOfWeek int) ([]map[string]any, error) {
	dateStr := targetDate.Format("2006-01-02")

	patternRows, err := c.db.Query(`
		SELECT sp.id, co.id, co.display_name, c.name, co.activity_type,
		       sp.start_time, sp.end_time, COALESCE(r.code, ''), sp.effective_from, sp.effective_until,
		       COALESCE(sp.meeting_link, '')
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		WHERE sem.class_id = ?
		  AND ((? = 0 AND sem.status = 'ACTIVE') OR sem.id = ?)
		  AND sp.status = 'ACTIVE'
		  AND sp.day_of_week = ?
		  AND (sp.effective_from IS NULL OR sp.effective_from <= ?)
		  AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
		ORDER BY sp.start_time ASC;
	`, classID, semesterID, semesterID, dayOfWeek, dateStr, dateStr)

	var items []map[string]any
	if err != nil {
		return nil, err
	}
	defer patternRows.Close()

	for patternRows.Next() {
		var patternID, offID int64
		var offDisplay, courseName, actType, startTime, endTime, roomCode, meetingLink string
		var effFrom, effUntil sql.NullString

		if err := patternRows.Scan(&patternID, &offID, &offDisplay, &courseName, &actType, &startTime, &endTime, &roomCode, &effFrom, &effUntil, &meetingLink); err == nil {
			lecturers := c.getOfferingLecturers(offID)
			items = append(items, map[string]any{
				"id":            fmt.Sprintf("pat_%d", patternID),
				"kind":          "REGULER",
				"offering":      offDisplay,
				"title":         courseName,
				"activity_type": actType,
				"starts_at":     startTime,
				"ends_at":       endTime,
				"room":          roomCode,
				"meeting_link":  meetingLink,
				"lecturers":     lecturers,
				"source": map[string]any{
					"pattern_id": patternID,
				},
			})
		}
	}

	eventRows, err := c.db.Query(`
		SELECT te.id, te.event_kind, co.id, co.display_name, c.name, co.activity_type,
		       strftime('%H:%M', te.starts_at) as start_time,
		       strftime('%H:%M', te.ends_at) as end_time,
		       COALESCE(r.code, ''), te.origin_schedule_pattern_id, COALESCE(te.meeting_link, '')
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE sem.class_id = ?
		  AND ((? = 0 AND sem.status = 'ACTIVE') OR sem.id = ?)
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND date(te.starts_at) = ?;
	`, classID, semesterID, semesterID, dateStr)

	if err == nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var eventID, offID int64
			var eventKind, offDisplay, courseName, actType, startTime, endTime, roomCode, meetingLink string
			var originPatID sql.NullInt64

			if err := eventRows.Scan(&eventID, &eventKind, &offID, &offDisplay, &courseName, &actType, &startTime, &endTime, &roomCode, &originPatID, &meetingLink); err == nil {
				kindMap := map[string]string{
					"REPLACEMENT":       "PENGGANTI",
					"EXTRA":             "TAMBAHAN",
					"HOLIDAY":           "LIBUR",
					"SESSION_CANCELLED": "DIBATALKAN",
				}
				kindLabel, ok := kindMap[eventKind]
				if !ok {
					kindLabel = eventKind
				}

				lecturers := c.getOfferingLecturers(offID)

				items = append(items, map[string]any{
					"id":            fmt.Sprintf("ev_%d", eventID),
					"kind":          kindLabel,
					"offering":      offDisplay,
					"title":         courseName,
					"activity_type": actType,
					"starts_at":     startTime,
					"ends_at":       endTime,
					"room":          roomCode,
					"meeting_link":  meetingLink,
					"lecturers":     lecturers,
					"source": map[string]any{
						"event_id":   eventID,
						"pattern_id": originPatID.Int64,
					},
				})
			}
		}
	}

	return items, nil
}

func (c *PortalController) getOfferingLecturers(offeringID int64) []string {
	return GetOfferingLecturers(c.db, offeringID)
}

// GetOfferingLecturers mengembalikan daftar nama dosen untuk suatu offering.
func GetOfferingLecturers(db *sql.DB, offeringID int64) []string {
	var lecturers []string
	if db == nil {
		return lecturers
	}
	rows, err := db.Query(`
		SELECT l.full_name
		FROM offering_lecturers ol
		JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE ol.course_offering_id = ?;
	`, offeringID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				lecturers = append(lecturers, name)
			}
		}
	}
	return lecturers
}
