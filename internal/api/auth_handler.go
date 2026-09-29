package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward compatibility
type LoginRequest = v1.LoginRequest
type RoleAssignmentItem = v1.RoleAssignmentItem
type LoginResponse = v1.LoginResponse
type SwitchContextRequest = v1.SwitchContextRequest
type UpdateClassStatusRequest = v1.UpdateClassStatusRequest
type InvitationRequest = v1.InvitationRequest
type AcceptInvitationRequest = v1.AcceptInvitationRequest

// handleLogin mendelegasikan proses autentikasi pengurus ke AuthController.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.Login(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleLogout mendelegasikan proses pembatalan sesi ke AuthController.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.Logout(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetMe mendelegasikan retrieval profil dan konteks aktif ke AuthController.
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.GetMe(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleSwitchContext mendelegasikan perpindahan peran/konteks sesi ke AuthController.
func (s *Server) handleSwitchContext(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.SwitchContext(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetV1ClassesAccess mendelegasikan listing kelas yang dapat diakses ke AuthController.
func (s *Server) handleGetV1ClassesAccess(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.GetClassesAccess(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetV1Classes mendelegasikan listing kelas (ADMIN) ke AuthController.
func (s *Server) handleGetV1Classes(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.GetClasses(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handlePatchV1ClassStatus mendelegasikan perubahan lifecycle status kelas ke AuthController.
func (s *Server) handlePatchV1ClassStatus(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.PatchClassStatus(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateInvitation mendelegasikan pembuatan token undangan pengurus ke AuthController.
func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.CreateInvitation(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAcceptInvitation mendelegasikan klaim token undangan ke AuthController.
func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authController != nil {
		s.authController.AcceptInvitation(w, r)
		return
	}
	http.Error(w, "Auth controller belum diinisialisasi", http.StatusInternalServerError)
}
