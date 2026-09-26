package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/schedule"
)

func eventActor(r *http.Request) schedule.EventActor {
	principal, ok := principalFromRequest(r)
	if !ok {
		return schedule.EventActor{CorrelationID: audit.NewCorrelationID()}
	}
	return schedule.EventActor{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID, CorrelationID: audit.NewCorrelationID()}
}

func (s *Server) eventService() (*schedule.EventService, bool) {
	if s.scheduleEvents == nil {
		return nil, false
	}
	return s.scheduleEvents, true
}

func (s *Server) handleCreateTeachingEventDraft(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	var payload struct {
		OwnerOfferingID int64   `json:"owner_offering_id"`
		Kind            string  `json:"event_kind"`
		StartsAt        string  `json:"starts_at"`
		EndsAt          string  `json:"ends_at"`
		OriginPatternID *int64  `json:"origin_schedule_pattern_id"`
		OriginDate      *string `json:"origin_occurrence_date"`
		RoomID          *int64  `json:"room_id"`
		Reason          *string `json:"reason"`
		IsPermanent     bool    `json:"is_permanent"`
		NewStartTime    *string `json:"new_start_time"`
		NewEndTime      *string `json:"new_end_time"`
		NewRoomID       *int64  `json:"new_room_id"`
		NewDayOfWeek    *int    `json:"new_day_of_week"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if payload.OwnerOfferingID <= 0 || strings.TrimSpace(payload.Kind) == "" ||
		strings.TrimSpace(payload.StartsAt) == "" || strings.TrimSpace(payload.EndsAt) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field owner_offering_id, event_kind, starts_at, ends_at wajib diisi"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), payload.OwnerOfferingID)
		if err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if s.authService != nil {
			if err := s.authService.RequireOfferingMutation(r.Context(), *principal,
				auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	created, err := svc.CreateDraft(r.Context(), eventActor(r), schedule.CreateDraftInput{
		OwnerOfferingID: payload.OwnerOfferingID, Kind: payload.Kind,
		StartsAt: payload.StartsAt, EndsAt: payload.EndsAt,
		OriginPatternID: payload.OriginPatternID, OriginDate: payload.OriginDate,
		RoomID: payload.RoomID, Reason: payload.Reason, IsPermanent: payload.IsPermanent,
		NewStartTime: payload.NewStartTime, NewEndTime: payload.NewEndTime,
		NewRoomID: payload.NewRoomID, NewDayOfWeek: payload.NewDayOfWeek,
	})
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": created})
}

func (s *Server) handleUpdateTeachingEventDraft(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	var payload struct {
		StartsAt  *string `json:"starts_at"`
		EndsAt    *string `json:"ends_at"`
		RoomID    *int64  `json:"room_id"`
		ClearRoom bool    `json:"clear_room"`
		Reason    *string `json:"reason"`
		Version   int     `json:"version"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if payload.Version < 1 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field version wajib diisi"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		detail, detailErr := svc.GetFullDetail(r.Context(), eventID)
		if detailErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), detail.Event.OwnerOfferingID)
		if err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if s.authService != nil {
			if err := s.authService.RequireOfferingMutation(r.Context(), *principal,
				auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	updated, err := svc.UpdateDraft(r.Context(), eventActor(r), eventID, schedule.UpdateDraftInput{
		StartsAt: payload.StartsAt, EndsAt: payload.EndsAt, RoomID: payload.RoomID,
		ClearRoom: payload.ClearRoom, Reason: payload.Reason, ExpectedVersion: payload.Version,
	})
	if err != nil {
		if errors.Is(err, schedule.ErrEventVersion) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Versi data sudah berubah, muat ulang sebelum menyimpan"})
			return
		}
		if errors.Is(err, schedule.ErrEventInvalidState) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Hanya draf yang dapat diubah"})
			return
		}
		if errors.Is(err, schedule.ErrEventNotFound) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": updated})
}

func (s *Server) handleDeleteTeachingEventDraft(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		detail, detailErr := svc.GetFullDetail(r.Context(), eventID)
		if detailErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), detail.Event.OwnerOfferingID)
		if err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if s.authService != nil {
			if err := s.authService.RequireOfferingMutation(r.Context(), *principal,
				auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := svc.DeleteDraft(r.Context(), eventActor(r), eventID); err != nil {
		if errors.Is(err, schedule.ErrEventInvalidState) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Hanya draf yang dapat dihapus"})
			return
		}
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Draf event dihapus"})
}

func (s *Server) handleListTeachingEvents(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	classID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("class_id")), 10, 64)
	if err != nil || classID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal && !principal.IsSystemAdmin() &&
		(principal.ClassID == nil || *principal.ClassID != classID) {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
		return
	}
	items, err := svc.ListEvents(r.Context(), classID,
		strings.TrimSpace(r.URL.Query().Get("lifecycle")), strings.TrimSpace(r.URL.Query().Get("kind")))
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar event"})
		return
	}
	if items == nil {
		items = []schedule.EventRow{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleGetTeachingEventDetail(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	detail, err := svc.GetFullDetail(r.Context(), eventID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal && !principal.IsSystemAdmin() {
		visible := false
		for _, p := range detail.Participations {
			if principal.ClassID != nil && p.ClassID == *principal.ClassID && (p.Role == "OWNER" || p.Status == "ACCEPTED") {
				visible = true
				break
			}
		}
		if !visible {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": detail})
}

func (s *Server) handlePreviewTeachingEvent(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	preview, err := svc.Preview(r.Context(), eventID)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": preview})
}

func (s *Server) handlePublishTeachingEvent(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	var payload struct {
		ConflictOverrideReason *string `json:"conflict_override_reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		detail, detailErr := svc.GetFullDetail(r.Context(), eventID)
		if detailErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), detail.Event.OwnerOfferingID)
		if err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if s.authService != nil {
			if err := s.authService.RequireOfferingMutation(r.Context(), *principal,
				auth.Scope{ClassID: scope.ClassID, SemesterID: scope.SemesterID, CourseOfferingID: scope.CourseOfferingID}); err != nil {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	published, err := svc.Publish(r.Context(), eventActor(r), eventID, payload.ConflictOverrideReason)
	if err != nil {
		if errors.Is(err, schedule.ErrEventConflict) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Konflik pemblokir mencegah publikasi, periksa preview"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.enqueueScheduleChange(eventID, published)
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": published})
}

func (s *Server) handleRevokeTeachingEvent(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if strings.TrimSpace(payload.Reason) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field reason wajib diisi"})
		return
	}
	principal, hasPrincipal := principalFromRequest(r)
	if s.authService != nil {
		if !hasPrincipal {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
		detail, detailErr := svc.GetFullDetail(r.Context(), eventID)
		if detailErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		if !principal.IsSystemAdmin() && !principal.CanManageClass(detail.Event.OwnerClassID) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM yang dapat mencabut publikasi"})
			return
		}
		if principal.Role == "PJ" {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM yang dapat mencabut publikasi"})
			return
		}
	}
	revoked, err := svc.Revoke(r.Context(), eventActor(r), eventID, payload.Reason)
	if err != nil {
		if errors.Is(err, schedule.ErrEventInvalidState) {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Hanya event PUBLISHED yang dapat dicabut"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.enqueueScheduleCorrection(eventID, revoked)
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": revoked})
}

func (s *Server) handleAddEventParticipant(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	var payload struct {
		CourseOfferingID int64 `json:"course_offering_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil || payload.CourseOfferingID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "course_offering_id wajib diisi"})
		return
	}
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		detail, detailErr := svc.GetFullDetail(r.Context(), eventID)
		if detailErr != nil {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Event tidak ditemukan"})
			return
		}
		if s.authService != nil && !principal.IsSystemAdmin() && !principal.CanManageClass(detail.Event.OwnerClassID) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM pemilik yang dapat mengundang"})
			return
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := svc.AddParticipant(r.Context(), eventActor(r), eventID, payload.CourseOfferingID); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Undangan partisipasi dibuat (PENDING)"})
}

func (s *Server) handleRespondEventParticipant(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok || s.taskRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err1 := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	offeringID, err2 := strconv.ParseInt(strings.TrimSpace(r.PathValue("offeringId")), 10, 64)
	if err1 != nil || err2 != nil || eventID <= 0 || offeringID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter tidak valid"})
		return
	}
	var payload struct {
		Decision string `json:"decision"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	if principal, hasPrincipal := principalFromRequest(r); hasPrincipal {
		scope, err := s.taskRepo.GetOfferingScope(r.Context(), offeringID)
		if err != nil {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		if s.authService != nil && !principal.IsSystemAdmin() {
			if principal.ClassID == nil || *principal.ClassID != scope.ClassID || principal.Role != "KM" {
				s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM kelas peserta yang dapat merespons"})
				return
			}
		}
	} else if s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if err := svc.RespondParticipation(r.Context(), eventActor(r), eventID, offeringID, payload.Decision); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Respons partisipasi disimpan"})
}

func (s *Server) handleRecordRoomConfirmation(w http.ResponseWriter, r *http.Request) {
	svc, ok := s.eventService()
	if !ok {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan jadwal belum tersedia"})
		return
	}
	eventID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || eventID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID event tidak valid"})
		return
	}
	var payload struct {
		RoomID  int64   `json:"room_id"`
		Status  string  `json:"status"`
		Contact *string `json:"external_contact"`
		Note    *string `json:"note"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil || payload.RoomID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "room_id dan status wajib diisi"})
		return
	}
	if _, hasPrincipal := principalFromRequest(r); !hasPrincipal && s.authService != nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	recorded, err := svc.RecordRoomConfirmation(r.Context(), eventActor(r), eventID, payload.RoomID, payload.Status,
		derefStr(payload.Contact), derefStr(payload.Note))
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": recorded})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// enqueueScheduleChange best-effort posts a PRD change message; WA failure never fails publish.
func (s *Server) enqueueScheduleChange(eventID int64, published *schedule.EventRow) {
	if s.notifyService == nil || published == nil {
		return
	}
	text := "*PERUBAHAN JADWAL*\n" + published.OwnerDisplay + " (" + published.Kind + ")\n" +
		"Mulai: " + published.StartsAt + "\nSelesai: " + published.EndsAt
	if published.Reason != nil && *published.Reason != "" {
		text += "\nKeterangan: " + *published.Reason
	}
	_, _ = s.notifyService.EnqueueEventPublished(context.Background(), published.OwnerClassID, eventID, text, nil)
}

// enqueueScheduleCorrection posts a PRD correction message and supersedes pending change.
func (s *Server) enqueueScheduleCorrection(eventID int64, revoked *schedule.EventRow) {
	if s.notifyService == nil || revoked == nil {
		return
	}
	reason := ""
	if revoked.RevocationReason != nil {
		reason = *revoked.RevocationReason
	}
	text := "*KOREKSI JADWAL*\n" + revoked.OwnerDisplay + "\nPerubahan sebelumnya dicabut oleh KM.\nAlasan: " + reason
	_, _ = s.notifyService.EnqueueEventRevoked(context.Background(), revoked.OwnerClassID, eventID, text, nil)
}
