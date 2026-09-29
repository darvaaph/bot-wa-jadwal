package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type AuditLogResponseItem = v1.AuditLogResponseItem

// handleGetAuditLogs menangani GET /api/v1/audit
func (s *Server) handleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetAuditLogs(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}
