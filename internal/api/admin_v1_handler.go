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

// handleGetAdminAssignments menangani GET /api/v1/admin/assignments
func (s *Server) handleGetAdminAssignments(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetAssignments(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAdminSuspendAssignment menangani POST /api/v1/admin/assignments/{id}/suspend
func (s *Server) handleAdminSuspendAssignment(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.SuspendAssignment(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAdminRevokeAssignment menangani POST /api/v1/admin/assignments/{id}/revoke
func (s *Server) handleAdminRevokeAssignment(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RevokeAssignment(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetAdminInvitations menangani GET /api/v1/admin/invitations
func (s *Server) handleGetAdminInvitations(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetInvitations(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetNotificationAttempts menangani GET /api/v1/notifications/{id}/attempts
func (s *Server) handleGetNotificationAttempts(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetNotificationAttempts(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetBackups menangani GET /api/v1/backups
func (s *Server) handleGetBackups(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetBackups(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleAdminRevokeInvitation menangani POST /api/v1/admin/invitations/{id}/revoke
func (s *Server) handleAdminRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RevokeInvitation(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleEnterSupport menangani POST /api/v1/admin/support/enter
func (s *Server) handleEnterSupport(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.EnterSupport(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleExitSupport menangani POST /api/v1/admin/support/exit
func (s *Server) handleExitSupport(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.ExitSupport(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetActiveSupport menangani GET /api/v1/admin/support/active
func (s *Server) handleGetActiveSupport(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetActiveSupport(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleTestBotMessage menangani POST /api/v1/admin/bot/test-message
func (s *Server) handleTestBotMessage(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.TestBotMessage(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateProposal menangani POST /api/v1/master/proposals
func (s *Server) handleCreateProposal(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.CreateProposal(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetProposals menangani GET /api/v1/master/proposals
func (s *Server) handleGetProposals(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetProposals(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleApproveProposal menangani POST /api/v1/master/proposals/{id}/approve
func (s *Server) handleApproveProposal(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.ApproveProposal(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleRejectProposal menangani POST /api/v1/master/proposals/{id}/reject
func (s *Server) handleRejectProposal(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RejectProposal(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetChannels menangani GET /api/v1/whatsapp-channels
func (s *Server) handleGetChannels(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.GetChannels(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleLinkChannel menangani POST /api/v1/whatsapp-channels
func (s *Server) handleLinkChannel(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.LinkChannel(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleRevokeChannel menangani POST /api/v1/whatsapp-channels/{id}/revoke
func (s *Server) handleRevokeChannel(w http.ResponseWriter, r *http.Request) {
	if s.adminController != nil {
		s.adminController.RevokeChannel(w, r)
		return
	}
	http.Error(w, "Admin controller belum diinisialisasi", http.StatusInternalServerError)
}
