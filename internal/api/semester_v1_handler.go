package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type CreateSemesterRequest = v1.CreateSemesterRequest

// handleGetClassSemesters menangani GET /api/v1/classes/{slug}/semesters
func (s *Server) handleGetClassSemesters(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.GetClassSemesters(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateClassSemester menangani POST /api/v1/classes/{slug}/semesters
func (s *Server) handleCreateClassSemester(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.CreateClassSemester(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleActivateSemester menangani POST /api/v1/classes/{slug}/semesters/{id}/activate
func (s *Server) handleActivateSemester(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.ActivateSemester(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleGetSemesterOfferings menangani GET /api/v1/semesters/{id}/offerings
func (s *Server) handleGetSemesterOfferings(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.GetSemesterOfferings(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handlePreviewSemester menangani GET /api/v1/classes/{slug}/semesters/{id}/preview
func (s *Server) handlePreviewSemester(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.PreviewSemester(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleCreateSemesterOffering menangani POST /api/v1/semesters/{id}/offerings
func (s *Server) handleCreateSemesterOffering(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.CreateSemesterOffering(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleDeleteDraftSemester menangani DELETE /api/v1/classes/{slug}/semesters/{id}
func (s *Server) handleDeleteDraftSemester(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.DeleteDraftSemester(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}
