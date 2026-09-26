package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/auth"
)

func (s *Server) handleListRoleAssignments(w http.ResponseWriter, r *http.Request) {
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
	items, err := s.authService.ListRoleAssignments(r.Context(), *principal, classID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar peran"})
		return
	}
	if items == nil {
		items = []auth.RoleAssignment{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": assignmentsDTO(items)})
}

func (s *Server) handleUpdateRoleAssignment(w http.ResponseWriter, r *http.Request) {
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	assignmentID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || assignmentID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID penugasan tidak valid"})
		return
	}
	var payload struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	if err := s.authService.UpdateRoleAssignmentStatus(r.Context(), *principal, assignmentID, payload.Status, payload.Reason); err != nil {
		if errors.Is(err, auth.ErrAccessDenied) {
			s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Tindakan tidak tersedia pada cakupan aktif"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Status penugasan diperbarui"})
}

func (s *Server) handleRequestRecovery(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	var payload struct {
		IdentityKey string `json:"identity_key"`
		Method      string `json:"method"`
	}
	if err := decodeLimitedJSON(w, r, &payload); err != nil || strings.TrimSpace(payload.IdentityKey) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field identity_key wajib diisi"})
		return
	}
	token, err := s.authService.RequestRecovery(r.Context(), payload.IdentityKey, payload.Method, "")
	if err != nil {
		// Generic response to avoid account enumeration.
		s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Jika identitas terdaftar, instruksi pemulihan telah dibuat"})
		return
	}
	_ = token
	// Identical generic response: the token is never disclosed here. It is
	// relayed out-of-band (WhatsApp/admin) via the admin issue endpoint.
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Jika identitas terdaftar, instruksi pemulihan telah dibuat"})
}

// handleIssueRecovery lets a System Admin create a recovery token for manual
// relay after out-of-band identity verification. The reason is mandatory audit.
func (s *Server) handleIssueRecovery(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat menerbitkan token pemulihan"})
		return
	}
	var payload struct {
		IdentityKey string `json:"identity_key"`
		Method      string `json:"method"`
		Reason      string `json:"reason"`
	}
	if err := decodeLimitedJSON(w, r, &payload); err != nil || strings.TrimSpace(payload.IdentityKey) == "" || strings.TrimSpace(payload.Reason) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field identity_key dan reason wajib diisi"})
		return
	}
	token, err := s.authService.RequestRecovery(r.Context(), payload.IdentityKey, payload.Method, payload.Reason)
	if err != nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Identitas tidak terdaftar"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{"recovery_token": token}})
}

func (s *Server) handleConfirmRecovery(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}
	var payload struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeLimitedJSON(w, r, &payload); err != nil || strings.TrimSpace(payload.Token) == "" || payload.NewPassword == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field token dan new_password wajib diisi"})
		return
	}
	if err := s.authService.ConfirmRecovery(r.Context(), payload.Token, payload.NewPassword); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Token pemulihan tidak valid atau kata sandi tidak memenuhi syarat"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Kata sandi diperbarui, silakan login kembali"})
}
