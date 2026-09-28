package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/audit"
)

func (s *Server) auditDB() *sql.DB {
	if s.academicRepo != nil {
		return s.academicRepo.DB()
	}
	return nil
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	db := s.auditDB()
	if db == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan audit belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	}
	q := r.URL.Query()
	var classID *int64
	if raw := strings.TrimSpace(q.Get("class_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
			return
		}
		classID = &parsed
	}
	f := audit.Filter{
		EntityType: strings.TrimSpace(q.Get("entity_type")),
		Action:     strings.TrimSpace(q.Get("action")),
		From:       strings.TrimSpace(q.Get("from")),
		To:         strings.TrimSpace(q.Get("to")),
	}
	if raw := strings.TrimSpace(q.Get("entity_id")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			f.EntityID = &parsed
		}
	}
	if raw := strings.TrimSpace(q.Get("actor_id")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			f.ActorID = &parsed
		}
	}
	if raw := strings.TrimSpace(q.Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			f.Limit = parsed
		}
	}
	if principal != nil && !principal.IsSystemAdmin() {
		if principal.Role == "KM" {
			if principal.ClassID == nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			if classID != nil && *classID != *principal.ClassID {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			f.ClassID = principal.ClassID
			items, err := audit.ListScoped(r.Context(), db, f, false, principal.ClassID, nil, nil)
			if err != nil {
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil audit"})
				return
			}
			s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
			return
		}
		// PJ: must scope to own offering.
		if principal.Role == "PJ" {
			if principal.ClassID == nil || principal.CourseOfferingID == nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			offering := *principal.CourseOfferingID
			if raw := strings.TrimSpace(q.Get("course_offering_id")); raw != "" {
				parsed, err := strconv.ParseInt(raw, 10, 64)
				if err != nil || parsed != offering {
					s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
					return
				}
			}
			if classID != nil && *classID != *principal.ClassID {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			f.ClassID = principal.ClassID
			items, err := audit.ListScoped(r.Context(), db, f, false, nil, &principal.UserID, &offering)
			if err != nil {
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil audit"})
				return
			}
			s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
			return
		}
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	if principal != nil && classID != nil {
		f.ClassID = classID
	} else if classID != nil {
		f.ClassID = classID
	}
	items, err := audit.ListScoped(r.Context(), db, f, true, nil, nil, nil)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil audit"})
		return
	}
	if items == nil {
		items = []audit.Item{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat melihat status sistem"})
		return
	}
	db := s.auditDB()
	out := map[string]any{"bot_connection": "uninitialized"}
	if s.botClient != nil {
		out["bot_connection"] = s.botClient.Status()
	}
	if db != nil {
		var classes, activeSem, pending, failed, rooms, users int
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM classes`).Scan(&classes)
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM semesters WHERE status='ACTIVE'`).Scan(&activeSem)
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM notification_messages WHERE status='PENDING'`).Scan(&pending)
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM notification_messages WHERE status='FAILED'`).Scan(&failed)
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM rooms WHERE status='ACTIVE'`).Scan(&rooms)
		_ = db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM users WHERE status='ACTIVE'`).Scan(&users)
		var version int
		_ = db.QueryRowContext(r.Context(), `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version)
		out["classes"] = classes
		out["active_semesters"] = activeSem
		out["pending_notifications"] = pending
		out["failed_notifications"] = failed
		out["active_rooms"] = rooms
		out["active_users"] = users
		out["schema_version"] = version
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": out})
}
