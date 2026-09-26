package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/notify"
)

func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	if s.notifyService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan notifikasi belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("class_id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
		return
	}
	if principal, ok := principalFromRequest(r); ok && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status != "" {
		switch status {
		case "PENDING", "PROCESSING", "SENT", "FAILED", "CANCELLED", "SUPERSEDED":
		default:
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter status tidak valid"})
			return
		}
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	items, err := s.notifyService.List(r.Context(), classID, status, limit)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil antrean notifikasi"})
		return
	}
	if items == nil {
		items = []notify.MessageItem{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleRetryNotification(w http.ResponseWriter, r *http.Request) {
	if s.notifyService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan notifikasi belum tersedia"})
		return
	}
	messageID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || messageID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID notifikasi tidak valid"})
		return
	}
	if principal, ok := principalFromRequest(r); ok {
		if !principal.IsSystemAdmin() {
			if principal.Role != "KM" || principal.ClassID == nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM atau System Admin yang dapat mencoba ulang"})
				return
			}
			msgClass, err := s.notifyService.GetMessageClass(r.Context(), messageID)
			if err != nil {
				s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Notifikasi gagal tidak ditemukan atau tidak dalam status FAILED"})
				return
			}
			if msgClass != *principal.ClassID {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := s.notifyService.Retry(r.Context(), messageID); err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Notifikasi gagal tidak ditemukan atau tidak dalam status FAILED"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Notifikasi dijadwalkan ulang"})
}

func (s *Server) handleProcessNotifications(w http.ResponseWriter, r *http.Request) {
	if s.notifyService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan notifikasi belum tersedia"})
		return
	}
	if principal, ok := principalFromRequest(r); ok {
		if !principal.IsSystemAdmin() {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat memicu pengiriman"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	sent, failed, err := s.notifyService.ProcessDue(r.Context(), s.notifySender, 20, time.Now().UTC())
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memproses antrean"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"sent": sent, "failed": failed}})
}
