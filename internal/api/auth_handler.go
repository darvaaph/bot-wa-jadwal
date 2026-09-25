package api

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/auth"
)

const (
	authSessionCookie = "bot_jadwal_session"
	authCSRFCookie    = "bot_jadwal_csrf"
	authRequestLimit  = 64 << 10
	authTimeout       = 5 * time.Second
)

type principalContextKey struct{}

type assignmentResponse struct {
	ID               int64  `json:"id"`
	Role             string `json:"role"`
	ScopeType        string `json:"scope_type"`
	ClassID          *int64 `json:"class_id,omitempty"`
	SemesterID       *int64 `json:"semester_id,omitempty"`
	CourseOfferingID *int64 `json:"course_offering_id,omitempty"`
}

func assignmentDTO(value auth.RoleAssignment) assignmentResponse {
	return assignmentResponse{
		ID:               value.ID,
		Role:             value.Role,
		ScopeType:        value.ScopeType,
		ClassID:          value.ClassID,
		SemesterID:       value.SemesterID,
		CourseOfferingID: value.CourseOfferingID,
	}
}

func assignmentsDTO(values []auth.RoleAssignment) []assignmentResponse {
	result := make([]assignmentResponse, 0, len(values))
	for _, value := range values {
		result = append(result, assignmentDTO(value))
	}
	return result
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if s.authService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan autentikasi belum dikonfigurasi"})
		return
	}

	var input struct {
		IdentityKey      string `json:"identity_key"`
		Password         string `json:"password"`
		RoleAssignmentID int64  `json:"role_assignment_id,omitempty"`
	}
	if err := decodeLimitedJSON(w, r, &input); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format permintaan tidak valid"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), authTimeout)
	defer cancel()
	identity, err := s.authService.Authenticate(ctx, auth.LoginInput{
		IdentityKey: input.IdentityKey,
		Password:    input.Password,
		Source:      requestSource(r),
	})
	if err != nil {
		status := http.StatusUnauthorized
		if !errors.Is(err, auth.ErrAuthenticationFailed) {
			status = http.StatusInternalServerError
		}
		s.writeJSON(w, status, map[string]string{"status": "error", "error": "Identitas atau kata sandi tidak valid"})
		return
	}
	if len(identity.Assignments) == 0 {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Akun tidak memiliki akses pengurus aktif"})
		return
	}

	assignmentID := input.RoleAssignmentID
	if assignmentID == 0 {
		assignmentID = identity.Assignments[0].ID
	}
	credential, err := s.authService.CreateSession(ctx, identity.User.ID, assignmentID)
	if err != nil {
		status := http.StatusForbidden
		if !errors.Is(err, auth.ErrAccessDenied) {
			status = http.StatusInternalServerError
		}
		s.writeJSON(w, status, map[string]string{"status": "error", "error": "Konteks akses tidak tersedia"})
		return
	}
	csrfToken, err := newCSRFCredential()
	if err != nil {
		_ = s.authService.RevokeSession(ctx, credential.Token, "CSRF_TOKEN_FAILURE")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal membuat sesi"})
		return
	}
	s.setAuthCookies(w, credential.Token, csrfToken, credential.Session.AbsoluteExpiresAt)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"user": map[string]any{
				"id":           identity.User.ID,
				"display_name": identity.User.DisplayName,
			},
			"active_role_assignment_id":  credential.Session.ActiveRoleAssignmentID,
			"assignments":                assignmentsDTO(identity.Assignments),
			"context_selection_required": len(identity.Assignments) > 1 && input.RoleAssignmentID == 0,
			"absolute_expires_at":        credential.Session.AbsoluteExpiresAt,
		},
	})
}

func (s *Server) handleAuthSession(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	principal, ok := s.authenticateRequest(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), authTimeout)
	defer cancel()
	assignments, err := s.authService.ListActiveAssignments(ctx, principal.UserID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat konteks akses"})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"user": map[string]any{
				"id":           principal.UserID,
				"display_name": principal.DisplayName,
			},
			"active_role_assignment_id": principal.RoleAssignmentID,
			"assignments":               assignmentsDTO(assignments),
		},
	})
}

func (s *Server) handleAuthSwitchContext(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if !s.requireCSRF(w, r) {
		return
	}
	token, ok := sessionToken(r)
	if !ok || s.authService == nil {
		s.clearAuthCookies(w)
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return
	}
	var input struct {
		RoleAssignmentID int64 `json:"role_assignment_id"`
	}
	if err := decodeLimitedJSON(w, r, &input); err != nil || input.RoleAssignmentID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Konteks akses tidak valid"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), authTimeout)
	defer cancel()
	credential, err := s.authService.SwitchContext(ctx, token, input.RoleAssignmentID)
	if err != nil {
		status := http.StatusForbidden
		if errors.Is(err, auth.ErrInvalidSession) {
			status = http.StatusUnauthorized
			s.clearAuthCookies(w)
		}
		s.writeJSON(w, status, map[string]string{"status": "error", "error": "Konteks akses tidak tersedia"})
		return
	}
	csrfToken, err := newCSRFCredential()
	if err != nil {
		_ = s.authService.RevokeSession(ctx, credential.Token, "CSRF_TOKEN_FAILURE")
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal merotasi sesi"})
		return
	}
	s.setAuthCookies(w, credential.Token, csrfToken, credential.Session.AbsoluteExpiresAt)
	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"active_role_assignment_id": credential.Session.ActiveRoleAssignmentID,
			"absolute_expires_at":       credential.Session.AbsoluteExpiresAt,
		},
	})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	setNoStore(w)
	if !s.requireCSRF(w, r) {
		return
	}
	token, ok := sessionToken(r)
	if ok && s.authService != nil {
		ctx, cancel := context.WithTimeout(r.Context(), authTimeout)
		defer cancel()
		_ = s.authService.RevokeSession(ctx, token, "LOGOUT")
	}
	s.clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) authenticateRequest(w http.ResponseWriter, r *http.Request) (*auth.Principal, bool) {
	token, ok := sessionToken(r)
	if !ok || s.authService == nil {
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return nil, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), authTimeout)
	defer cancel()
	principal, err := s.authService.ValidateSession(ctx, token)
	if err != nil {
		s.clearAuthCookies(w)
		s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
		return nil, false
	}
	return principal, true
}

func (s *Server) requireAuthentication(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, ok := s.authenticateRequest(w, r)
		if !ok {
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), principalContextKey{}, principal)))
	}
}

func (s *Server) authenticateIfConfigured(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authService == nil {
			next(w, r)
			return
		}
		s.requireAuthentication(next)(w, r)
	}
}

func (s *Server) authenticateMutationIfConfigured(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authService == nil {
			next(w, r)
			return
		}
		if !s.requireCSRF(w, r) {
			return
		}
		s.requireAuthentication(next)(w, r)
	}
}

func (s *Server) disableLegacyMutationWhenAuthConfigured(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.authService != nil {
			s.writeJSON(w, http.StatusGone, map[string]string{
				"status": "error",
				"error":  "Endpoint mutasi lama dinonaktifkan; gunakan API v1 yang memeriksa sesi dan cakupan",
			})
			return
		}
		next(w, r)
	}
}

func principalFromRequest(r *http.Request) (*auth.Principal, bool) {
	principal, ok := r.Context().Value(principalContextKey{}).(*auth.Principal)
	return principal, ok
}

func setNoStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
}

func decodeLimitedJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, authRequestLimit)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("permintaan hanya boleh memuat satu objek JSON")
	}
	return nil
}

func requestSource(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func sessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(authSessionCookie)
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return "", false
	}
	return cookie.Value, true
}

func newCSRFCredential() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (s *Server) requireCSRF(w http.ResponseWriter, r *http.Request) bool {
	cookie, err := r.Cookie(authCSRFCookie)
	header := r.Header.Get("X-CSRF-Token")
	if err != nil || cookie.Value == "" || header == "" || len(cookie.Value) != len(header) || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Token CSRF tidak valid"})
		return false
	}
	return true
}

func (s *Server) setAuthCookies(w http.ResponseWriter, session, csrf string, expires time.Time) {
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{Name: authSessionCookie, Value: session, Path: "/", HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: maxAge})
	http.SetCookie(w, &http.Cookie{Name: authCSRFCookie, Value: csrf, Path: "/", HttpOnly: false, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: maxAge})
}

func (s *Server) clearAuthCookies(w http.ResponseWriter) {
	for _, name := range []string{authSessionCookie, authCSRFCookie} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: name == authSessionCookie, Secure: s.secureCookies, SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	}
}
