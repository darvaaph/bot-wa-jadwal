package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/auth"
)

func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	var payload struct {
		Role             string `json:"role"`
		ClassID          *int64 `json:"class_id"`
		SemesterID       *int64 `json:"semester_id"`
		CourseOfferingID *int64 `json:"course_offering_id"`
		IdentityKey      string `json:"identity_key"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	inv, token, err := s.authService.CreateInvitation(r.Context(), *principal, auth.InvitationInput{
		Role: payload.Role, ClassID: payload.ClassID, SemesterID: payload.SemesterID,
		CourseOfferingID: payload.CourseOfferingID, IdentityKey: payload.IdentityKey,
	})
	if err != nil {
		if errors.Is(err, auth.ErrAccessDenied) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Undangan tidak valid: peran, cakupan, atau identitas tidak sesuai"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{
		"id": inv.ID, "role": inv.Role, "scope_type": inv.ScopeType,
		"class_id": inv.ClassID, "semester_id": inv.SemesterID, "course_offering_id": inv.CourseOfferingID,
		"identity_key": inv.IdentityKey, "status": inv.Status, "expires_at": inv.ExpiresAt,
		"token": token,
	}})
}

func (s *Server) handleListInvitations(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	var classID *int64
	if raw := strings.TrimSpace(r.URL.Query().Get("class_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter class_id tidak valid"})
			return
		}
		classID = &parsed
	}
	items, err := s.authService.ListInvitations(r.Context(), *principal, classID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar undangan"})
		return
	}
	if items == nil {
		items = []auth.Invitation{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleGetInvitationByToken(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	token := strings.TrimSpace(r.PathValue("token"))
	inv, err := s.authService.GetInvitationByToken(r.Context(), token)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Undangan tidak valid, kedaluwarsa, atau sudah digunakan"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{
		"role": inv.Role, "scope_type": inv.ScopeType, "class_id": inv.ClassID,
		"semester_id": inv.SemesterID, "course_offering_id": inv.CourseOfferingID,
		"identity_key": inv.IdentityKey, "status": inv.Status, "expires_at": inv.ExpiresAt,
	}})
}

func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	token := strings.TrimSpace(r.PathValue("token"))
	var payload struct {
		DisplayName string `json:"display_name"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	user, assignment, err := s.authService.AcceptInvitation(r.Context(), token, payload.DisplayName, payload.Password)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Undangan tidak valid atau kata sandi tidak memenuhi syarat (min 12 karakter)"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{
		"user_id": user.ID, "role_assignment_id": assignment.ID, "role": assignment.Role,
	}})
}

func (s *Server) handleRevokeInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	invitationID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || invitationID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID undangan tidak valid"})
		return
	}
	if err := s.authService.RevokeInvitation(r.Context(), *principal, invitationID); err != nil {
		if errors.Is(err, auth.ErrAccessDenied) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Undangan tidak dapat dicabut"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Undangan dicabut"})
}

func (s *Server) handleResendInvitation(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	invitationID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || invitationID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID undangan tidak valid"})
		return
	}
	token, err := s.authService.ResendInvitation(r.Context(), *principal, invitationID)
	if err != nil {
		if errors.Is(err, auth.ErrAccessDenied) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Undangan tidak dapat dikirim ulang"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"token": token}})
}
