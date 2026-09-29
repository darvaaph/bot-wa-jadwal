package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type CreateMaterialRequest = v1.CreateMaterialRequest

// handleGetV1Materials menangani GET /api/v1/materials
func (s *Server) handleGetV1Materials(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.GetMaterials(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateV1Material menangani POST /api/v1/materials
func (s *Server) handleCreateV1Material(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.CreateMaterial(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}
