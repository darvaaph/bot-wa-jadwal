package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/portal"
)

type createPortalSessionRequest struct {
	Code string `json:"code"`
}

func portalRequestSource(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

func (s *Server) handleCreatePortalSession(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil || s.portalService == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req createPortalSessionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "code wajib diisi")
		return
	}

	var classID int64
	err := s.v1DB.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?`, r.PathValue("slug")).Scan(&classID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi kode portal")
			return
		}
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Kode portal tidak valid")
		return
	}

	session, err := s.portalService.VerifyCode(r.Context(), classID, req.Code, portalRequestSource(r))
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrRateLimited):
			s.writeV1Error(w, http.StatusTooManyRequests, CodeTooManyRequests, "Terlalu banyak percobaan kode portal. Coba lagi nanti")
		case errors.Is(err, portal.ErrInvalidInput):
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "code wajib diisi")
		case errors.Is(err, portal.ErrInvalidCode), errors.Is(err, portal.ErrNotFound):
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Kode portal tidak valid")
		default:
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membuat sesi portal")
		}
		return
	}

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"portal_token": session.Token,
		"expires_at":   session.ExpiresAt.UTC().Format(time.RFC3339),
	})
}
