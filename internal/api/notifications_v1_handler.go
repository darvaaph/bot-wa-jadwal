package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type NotificationResponseItem = v1.NotificationResponseItem

// handleGetNotifications menangani GET /api/v1/notifications
func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetNotifications(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleRetryNotification menangani POST /api/v1/notifications/{id}/retry
func (s *Server) handleRetryNotification(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RetryNotification(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}
