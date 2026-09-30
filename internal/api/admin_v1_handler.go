package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type AdminStatusResponse = v1.AdminStatusResponse
type SuspendUserRequest = v1.SuspendUserRequest
type RecoverUserRequest = v1.RecoverUserRequest

// handleGetAdminStatus menangani GET /api/v1/admin/status
func (s *Server) handleGetAdminStatus(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetAdminStatus(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAdminSuspendUser menangani POST /api/v1/admin/users/{id}/suspend
func (s *Server) handleAdminSuspendUser(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.SuspendUser(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAdminRecoverUser menangani POST /api/v1/admin/users/{id}/recover
func (s *Server) handleAdminRecoverUser(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RecoverUser(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetAdminUsers menangani GET /api/v1/admin/users
func (s *Server) handleGetAdminUsers(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetUsers(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}
