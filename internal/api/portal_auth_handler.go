package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/portal"
)

const portalTokenCookie = "portal_token"

func portalTokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(portalTokenCookie); err == nil && strings.TrimSpace(c.Value) != "" {
		return strings.TrimSpace(c.Value)
	}
	if h := strings.TrimSpace(r.Header.Get("X-Portal-Token")); h != "" {
		return h
	}
	return ""
}

func (s *Server) portalAuthorized(r *http.Request, classID int64) bool {
	if s.portalService == nil {
		return false
	}
	return s.portalService.ValidateSession(r.Context(), classID, portalTokenFromRequest(r)) == nil
}

func (s *Server) handlePortalVerifyCode(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil || s.portalService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan portal belum tersedia"})
		return
	}
	slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	cls, err := s.academicRepo.GetClassBySlug(r.Context(), slug)
	if err != nil || cls == nil {
		portalNotFound(w, s)
		return
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil || strings.TrimSpace(payload.Code) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field code wajib diisi"})
		return
	}
	token, err := s.portalService.VerifyCode(r.Context(), cls.ID, payload.Code, requestSource(r))
	if err != nil {
		switch err {
		case portal.ErrRateLimited:
			s.writeJSON(w, http.StatusTooManyRequests, map[string]string{"status": "error", "error": "Terlalu banyak percobaan kode, coba lagi nanti"})
		case portal.ErrInvalidCode:
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Kode kelas tidak valid"})
		default:
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memverifikasi kode kelas"})
		}
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: portalTokenCookie, Value: token, Path: "/",
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(30 * 24 * time.Hour), MaxAge: 30 * 24 * 3600,
	})
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"authorized": true, "token": token}})
}

func (s *Server) handlePortalAccess(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		portalNotFound(w, s)
		return
	}
	slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	cls, err := s.academicRepo.GetClassBySlug(r.Context(), slug)
	if err != nil || cls == nil {
		portalNotFound(w, s)
		return
	}
	var mode string
	if err := s.academicRepo.DB().QueryRowContext(r.Context(),
		`SELECT portal_access_mode FROM class_settings WHERE class_id = ?`, cls.ID).Scan(&mode); err != nil {
		mode = "LINK"
	}
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode == "" {
		mode = "LINK"
	}
	authorized := true
	if mode == "CODE" {
		authorized = s.portalAuthorized(r, cls.ID)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{
		"mode": mode, "authorized": authorized,
	}})
}

func (s *Server) handleUpdateClassSettings(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil || s.portalService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kelas tidak valid"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() && (principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	} else if !principal.IsSystemAdmin() && principal.Role != "KM" {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM yang dapat mengubah pengaturan kelas"})
		return
	}
	var payload struct {
		PortalAccessMode           *string `json:"portal_access_mode"`
		PortalCode                 *string `json:"portal_code"`
		MeetingLinkVisibility      *string `json:"meeting_link_visibility"`
		MorningReminderTime        *string `json:"morning_reminder_time"`
		AfternoonReminderTime      *string `json:"afternoon_reminder_time"`
		ReplacementReminderMinutes *int    `json:"replacement_reminder_minutes"`
		Timezone                   *string `json:"timezone"`
		Version                    int     `json:"version"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if payload.Version < 1 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field version wajib diisi untuk deteksi konflik"})
		return
	}
	// Pre-check before any write so a stale version fails fast without
	// partially applying code/mode rotation.
	var currentVersion int
	if err := s.academicRepo.DB().QueryRowContext(r.Context(), `SELECT version FROM class_settings WHERE class_id = ?`, classID).Scan(&currentVersion); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Pengaturan kelas tidak ditemukan"})
		return
	}
	if currentVersion != payload.Version {
		s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi pengaturan sudah berubah, muat ulang sebelum menyimpan"})
		return
	}
	if payload.PortalCode != nil && strings.TrimSpace(*payload.PortalCode) != "" {
		if err := s.portalService.SetClassCode(r.Context(), classID, *payload.PortalCode); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Kode kelas tidak valid"})
			return
		}
		payload.Version++
	}
	if payload.PortalAccessMode != nil {
		mode := strings.ToUpper(strings.TrimSpace(*payload.PortalAccessMode))
		if mode != "LINK" && mode != "CODE" {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "portal_access_mode harus LINK atau CODE"})
			return
		}
		if mode == "CODE" && (payload.PortalCode == nil || strings.TrimSpace(*payload.PortalCode) == "") {
			var existingCode sql.NullString
			_ = s.academicRepo.DB().QueryRowContext(r.Context(),
				`SELECT portal_code_hash FROM class_settings WHERE class_id = ?`, classID).Scan(&existingCode)
			if !existingCode.Valid || strings.TrimSpace(existingCode.String) == "" {
				s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Mode CODE memerlukan portal_code baru"})
				return
			}
		}
		if err := s.portalService.SetAccessMode(r.Context(), classID, mode); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Gagal mengubah mode akses portal"})
			return
		}
		payload.Version++
	}
	updates := []string{}
	args := []any{}
	if payload.MeetingLinkVisibility != nil {
		v := strings.ToUpper(strings.TrimSpace(*payload.MeetingLinkVisibility))
		if v != "VALID_CLASS_ACCESS" && v != "WHATSAPP_ONLY" {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "meeting_link_visibility tidak valid"})
			return
		}
		updates = append(updates, "meeting_link_visibility = ?")
		args = append(args, v)
	}
	if payload.MorningReminderTime != nil {
		v := strings.TrimSpace(*payload.MorningReminderTime)
		if v != "" && !validHHMM(v) {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "morning_reminder_time harus HH:MM"})
			return
		}
		if v == "" {
			updates = append(updates, "morning_reminder_time = NULL")
		} else {
			updates = append(updates, "morning_reminder_time = ?")
			args = append(args, v)
		}
	}
	if payload.AfternoonReminderTime != nil {
		v := strings.TrimSpace(*payload.AfternoonReminderTime)
		if v != "" && !validHHMM(v) {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "afternoon_reminder_time harus HH:MM"})
			return
		}
		if v == "" {
			updates = append(updates, "afternoon_reminder_time = NULL")
		} else {
			updates = append(updates, "afternoon_reminder_time = ?")
			args = append(args, v)
		}
	}
	if payload.ReplacementReminderMinutes != nil {
		if *payload.ReplacementReminderMinutes < 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "replacement_reminder_minutes tidak valid"})
			return
		}
		updates = append(updates, "replacement_reminder_minutes = ?")
		args = append(args, *payload.ReplacementReminderMinutes)
	}
	if payload.Timezone != nil {
		tz := strings.TrimSpace(*payload.Timezone)
		if tz == "" {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "timezone tidak valid"})
			return
		}
		if _, err := time.LoadLocation(tz); err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "timezone tidak dikenal"})
			return
		}
		updates = append(updates, "timezone = ?")
		args = append(args, tz)
	}
	if len(updates) > 0 {
		updates = append(updates, "version = version + 1")
		query := "UPDATE class_settings SET " + strings.Join(updates, ", ") + ", updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id = ? AND version = ?"
		args = append(args, classID, payload.Version)
		res, err := s.academicRepo.DB().ExecContext(r.Context(), query, args...)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menyimpan pengaturan kelas"})
			return
		}
		if n, _ := res.RowsAffected(); n != 1 {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi pengaturan sudah berubah, muat ulang sebelum menyimpan"})
			return
		}
	}
	s.writeClassSettingsAudit(r, classID, payload)
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Pengaturan kelas disimpan, sesi portal lama otomatis dicabut"})
}

func (s *Server) handleGetClassSettings(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kelas tidak valid"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() && (principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	} else if !principal.IsSystemAdmin() && principal.Role != "KM" {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM yang dapat melihat pengaturan kelas"})
		return
	}
	var timezone, mode, visibility string
	var morning, afternoon sql.NullString
	var replacement, version int
	err = s.academicRepo.DB().QueryRowContext(r.Context(), `SELECT timezone, portal_access_mode,
		meeting_link_visibility, morning_reminder_time, afternoon_reminder_time,
		replacement_reminder_minutes, version FROM class_settings WHERE class_id = ?`, classID).
		Scan(&timezone, &mode, &visibility, &morning, &afternoon, &replacement, &version)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Pengaturan kelas tidak ditemukan"})
		return
	}
	data := map[string]any{
		"timezone": timezone, "portal_access_mode": mode,
		"meeting_link_visibility":      visibility,
		"replacement_reminder_minutes": replacement, "version": version,
	}
	if morning.Valid {
		data["morning_reminder_time"] = morning.String
	}
	if afternoon.Valid {
		data["afternoon_reminder_time"] = afternoon.String
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": data})
}

// writeClassSettingsAudit records one UPDATE row for class settings changes.
// Best-effort: settings are already committed; a failed audit insert must not
// mask the success response (failures are surfaced via server logs by callers
// that need guarantees; critical paths use in-transaction audit instead).
func (s *Server) writeClassSettingsAudit(r *http.Request, classID int64, payload struct {
	PortalAccessMode           *string `json:"portal_access_mode"`
	PortalCode                 *string `json:"portal_code"`
	MeetingLinkVisibility      *string `json:"meeting_link_visibility"`
	MorningReminderTime        *string `json:"morning_reminder_time"`
	AfternoonReminderTime      *string `json:"afternoon_reminder_time"`
	ReplacementReminderMinutes *int    `json:"replacement_reminder_minutes"`
	Timezone                   *string `json:"timezone"`
	Version                    int     `json:"version"`
}) {
	changed := []string{}
	if payload.PortalAccessMode != nil {
		changed = append(changed, "portal_access_mode")
	}
	if payload.PortalCode != nil && strings.TrimSpace(*payload.PortalCode) != "" {
		changed = append(changed, "portal_code(rotated)")
	}
	if payload.MeetingLinkVisibility != nil {
		changed = append(changed, "meeting_link_visibility")
	}
	if payload.MorningReminderTime != nil || payload.AfternoonReminderTime != nil || payload.ReplacementReminderMinutes != nil {
		changed = append(changed, "reminder_times")
	}
	if payload.Timezone != nil {
		changed = append(changed, "timezone")
	}
	if len(changed) == 0 {
		return
	}
	principal, _ := principalFromRequest(r)
	var actorUser, actorAssignment any
	actorType := "SYSTEM"
	if principal != nil {
		actorUser = principal.UserID
		actorAssignment = principal.RoleAssignmentID
		actorType = "USER"
	}
	after := `{"changed":["` + strings.Join(changed, `","`) + `"]}`
	corr := audit.NewCorrelationID()
	_, _ = s.academicRepo.DB().ExecContext(r.Context(), `INSERT INTO audit_logs (
		class_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'UPDATE', 'CLASS_SETTING', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, actorUser, actorAssignment, actorType, classID, after, corr)
}

func validHHMM(v string) bool {
	if len(v) != 5 || v[2] != ':' {
		return false
	}
	h, err1 := strconv.Atoi(v[:2])
	m, err2 := strconv.Atoi(v[3:])
	return err1 == nil && err2 == nil && h >= 0 && h <= 23 && m >= 0 && m <= 59
}
