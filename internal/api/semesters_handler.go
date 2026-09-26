package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/semester"
)

func authScope(classID, semesterID, offeringID int64) auth.Scope {
	return auth.Scope{ClassID: classID, SemesterID: semesterID, CourseOfferingID: offeringID}
}

func semesterActor(r *http.Request) semester.Actor {
	principal, ok := principalFromRequest(r)
	if !ok {
		return semester.Actor{}
	}
	return semester.Actor{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID}
}

func semesterService(r *http.Request, s *Server) (*semester.Service, bool) {
	if s.semesterService == nil {
		return nil, false
	}
	return s.semesterService, true
}

func requireSemesterManager(r *http.Request, classID int64) error {
	principal, ok := principalFromRequest(r)
	if !ok {
		return errors.New("unauthorized")
	}
	if principal.IsSystemAdmin() {
		return nil
	}
	if principal.Role == "KM" && principal.ClassID != nil && *principal.ClassID == classID {
		return nil
	}
	return semester.ErrForbidden
}

func (s *Server) handleCreateClass(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	if s.authService != nil {
		if !hasPrincipal || !principal.IsSystemAdmin() {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat membuat kelas"})
			return
		}
	}
	var payload struct {
		Code         string `json:"code"`
		Slug         string `json:"slug"`
		StudyProgram string `json:"study_program"`
		CohortYear   int    `json:"cohort_year"`
		GroupLabel   string `json:"group_label"`
		Timezone     string `json:"timezone"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	id, err := svc.CreateClass(r.Context(), semesterActor(r), payload.Code, payload.Slug, payload.StudyProgram, payload.CohortYear, payload.GroupLabel, payload.Timezone)
	if err != nil {
		if errors.Is(err, semester.ErrConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Identitas atau slug kelas sudah digunakan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data kelas tidak valid"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{"id": id}})
}

func (s *Server) handleListSemesters(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kelas tidak valid"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	items, err := svc.ListSemesters(r.Context(), classID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar semester"})
		return
	}
	if items == nil {
		items = []semester.SemesterListItem{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleCreateSemesterDraft(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kelas tidak valid"})
		return
	}
	if s.authService != nil {
		if err := requireSemesterManager(r, classID); err != nil {
			if err.Error() == "unauthorized" {
				s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			} else {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM atau System Admin yang dapat membuat semester"})
			}
			return
		}
	}
	var payload struct {
		AcademicYear     string `json:"academic_year"`
		Term             string `json:"term"`
		StartsOn         string `json:"starts_on"`
		EndsOn           string `json:"ends_on"`
		SourceSemesterID *int64 `json:"source_semester_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	id, err := svc.CreateDraft(r.Context(), semesterActor(r), classID, semester.DraftInput{
		AcademicYear: payload.AcademicYear, Term: payload.Term,
		StartsOn: payload.StartsOn, EndsOn: payload.EndsOn, SourceSemesterID: payload.SourceSemesterID,
	})
	if err != nil {
		if errors.Is(err, semester.ErrConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Semester tahun/term tersebut sudah ada di kelas ini"})
			return
		}
		if errors.Is(err, semester.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Semester sumber tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data semester tidak valid (cek tanggal mulai < selesai)"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{"id": id, "status": "DRAFT"}})
}

func (s *Server) handleSemesterPreview(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err1 := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	semID, err2 := strconv.ParseInt(strings.TrimSpace(r.PathValue("sid")), 10, 64)
	if err1 != nil || err2 != nil || classID <= 0 || semID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter tidak valid"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	preview, err := svc.Preview(r.Context(), classID, semID)
	if err != nil {
		if errors.Is(err, semester.ErrNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Semester tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat preview semester"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": preview})
}

func (s *Server) handleSemesterActivate(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err1 := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	semID, err2 := strconv.ParseInt(strings.TrimSpace(r.PathValue("sid")), 10, 64)
	if err1 != nil || err2 != nil || classID <= 0 || semID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter tidak valid"})
		return
	}
	if s.authService != nil {
		if err := requireSemesterManager(r, classID); err != nil {
			if err.Error() == "unauthorized" {
				s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			} else {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "PJ tidak dapat mengaktifkan semester"})
			}
			return
		}
	}
	preview, err := svc.Preview(r.Context(), classID, semID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Semester tidak ditemukan"})
		return
	}
	if !preview.CanActivate {
		s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Semester belum siap diaktifkan: " + strings.Join(preview.Blockers, "; ")})
		return
	}
	var payload struct {
		Version int `json:"version"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if payload.Version < 1 {
		// Fall back to the previewed version so callers that only previewed
		// can still activate without an extra round-trip.
		payload.Version = preview.Semester.Version
	}
	if err := svc.Activate(r.Context(), semesterActor(r), classID, semID, payload.Version); err != nil {
		if errors.Is(err, semester.ErrInvalidState) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Hanya semester DRAFT yang dapat diaktifkan"})
			return
		}
		if errors.Is(err, semester.ErrVersion) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi semester sudah berubah, muat ulang preview sebelum mengaktifkan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengaktifkan semester"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Semester diaktifkan, semester lama diarsipkan"})
}

func (s *Server) handleSemesterImport(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kelas tidak valid"})
		return
	}
	var actorID int64 = 1
	if s.authService != nil {
		if err := requireSemesterManager(r, classID); err != nil {
			if err.Error() == "unauthorized" {
				s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			} else {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM atau System Admin yang dapat mengimpor"})
			}
			return
		}
		if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
			actorID = principal.UserID
		}
	}
	var payload struct {
		AcademicYear string          `json:"academic_year"`
		Term         string          `json:"term"`
		StartsOn     string          `json:"starts_on"`
		EndsOn       string          `json:"ends_on"`
		SemesterID   *int64          `json:"semester_id"`
		Data         json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if len(payload.Data) == 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field data wajib diisi"})
		return
	}
	semID, importErrs, err := svc.ImportJSON(r.Context(), semesterActor(r), classID, payload.SemesterID,
		payload.AcademicYear, payload.Term, payload.StartsOn, payload.EndsOn, payload.Data, actorID)
	if importErrs == nil {
		importErrs = []semester.ImportError{}
	}
	if err != nil {
		s.writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"status": "error", "error": "Impor ditolak: perbaiki kesalahan pemblokir", "errors": importErrs, "semester_id": semID,
		})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"semester_id": semID, "warnings": importErrs}})
}

func (s *Server) handleAddOffering(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	classID, err1 := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	semID, err2 := strconv.ParseInt(strings.TrimSpace(r.PathValue("sid")), 10, 64)
	if err1 != nil || err2 != nil || classID <= 0 || semID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter tidak valid"})
		return
	}
	if s.authService != nil {
		if err := requireSemesterManager(r, classID); err != nil {
			if err.Error() == "unauthorized" {
				s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			} else {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM atau System Admin yang dapat menambah mata kuliah"})
			}
			return
		}
	}
	var payload struct {
		CourseCode   string `json:"course_code"`
		CourseName   string `json:"course_name"`
		ActivityType string `json:"activity_type"`
		DisplayName  string `json:"display_name"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	id, err := svc.AddOffering(r.Context(), semesterActor(r), classID, semID, payload.CourseCode, payload.CourseName, payload.ActivityType, payload.DisplayName)
	if err != nil {
		if errors.Is(err, semester.ErrConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Offering tersebut sudah ada di semester ini"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data offering tidak valid"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{"id": id}})
}

func (s *Server) handleAddPattern(w http.ResponseWriter, r *http.Request) {
	svc, ok := semesterService(r, s)
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	var payload struct {
		CourseOfferingID int64   `json:"course_offering_id"`
		RoomCode         *string `json:"room_code"`
		DayOfWeek        int     `json:"day_of_week"`
		StartTime        string  `json:"start_time"`
		EndTime          string  `json:"end_time"`
		EffectiveFrom    string  `json:"effective_from"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if payload.CourseOfferingID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "course_offering_id wajib diisi"})
		return
	}
	if s.taskRepo != nil && s.authService != nil {
		if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
			scope, scopeErr := s.taskRepo.GetOfferingScope(r.Context(), payload.CourseOfferingID)
			if scopeErr != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
			if err := s.authService.RequireOfferingMutation(r.Context(), *principal,
				authScope(scope.ClassID, scope.SemesterID, scope.CourseOfferingID)); err != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		} else {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	}
	var roomID *int64
	if payload.RoomCode != nil && strings.TrimSpace(*payload.RoomCode) != "" {
		// Resolve room by code; create lookup only (rooms managed in BE-F).
		if s.academicRepo != nil {
			var rid int64
			err := s.academicRepo.DB().QueryRowContext(r.Context(), `SELECT id FROM rooms WHERE code = ?`, strings.TrimSpace(*payload.RoomCode)).Scan(&rid)
			if err == nil {
				roomID = &rid
			}
		}
	}
	id, err := svc.AddPattern(r.Context(), semesterActor(r), payload.CourseOfferingID, roomID, payload.DayOfWeek, payload.StartTime, payload.EndTime, payload.EffectiveFrom)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data pola jadwal tidak valid"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{"id": id}})
}
