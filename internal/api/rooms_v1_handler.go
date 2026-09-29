package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type RoomCandidateItem = v1.RoomCandidateItem
type RoomConfirmationRequest = v1.RoomConfirmationRequest

// handleGetRoomCandidates menangani GET /api/v1/rooms/candidates
func (s *Server) handleGetRoomCandidates(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.GetRoomCandidates(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateRoomConfirmation menangani POST /api/v1/teaching-events/{id}/room-confirmations
func (s *Server) handleCreateRoomConfirmation(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.CreateRoomConfirmation(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}
