package api

import (
	"net/http"

	v1 "bot-jadwal/internal/api/v1"
)

// Alias types untuk backward-compatibility
type CourseImportItem = v1.CourseImportItem
type LecturerImportItem = v1.LecturerImportItem
type OfferingImportItem = v1.OfferingImportItem
type SchedulePatternImportItem = v1.SchedulePatternImportItem
type CurriculumImportPayload = v1.CurriculumImportPayload
type ImportErrorRecord = v1.ImportErrorRecord
type ApplyImportRequest = v1.ApplyImportRequest

// handleSemesterImportValidate menangani POST /api/v1/semesters/{id}/import-validate
func (s *Server) handleSemesterImportValidate(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.SemesterImportValidate(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}

// handleSemesterImportApply menangani POST /api/v1/semesters/{id}/import-apply
func (s *Server) handleSemesterImportApply(w http.ResponseWriter, r *http.Request) {
	if s.academicController != nil {
		s.academicController.SemesterImportApply(w, r)
		return
	}
	http.Error(w, "Academic controller belum diinisialisasi", http.StatusInternalServerError)
}
