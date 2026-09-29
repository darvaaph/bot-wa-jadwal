package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/portal"
)

type createPortalSessionRequest struct {
	Code string `json:"code"`
}

type rotatePortalCodeRequest struct {
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

func (s *Server) handleRotatePortalCode(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil || s.portalService == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req rotatePortalCodeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	var classID int64
	err := s.v1DB.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?`, r.PathValue("slug")).Scan(&classID)
	if errors.Is(err, sql.ErrNoRows) {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Anda tidak dapat merotasi kode portal kelas lain")
		return
	}

	result, err := s.portalService.RotateCode(r.Context(), portal.RotationRequest{
		ClassID:             classID,
		Code:                req.Code,
		ActorUserID:         u.UserID,
		ActorRoleAssignment: u.ActiveAssignmentID,
	})
	if err != nil {
		switch {
		case errors.Is(err, portal.ErrInvalidInput):
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "code harus terdiri dari 6 sampai 128 karakter, atau kosong untuk dibuat otomatis")
		case errors.Is(err, portal.ErrNotFound):
			s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Pengaturan portal kelas tidak ditemukan")
		case errors.Is(err, portal.ErrConflict):
			s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Pengaturan portal berubah. Muat ulang lalu coba kembali")
		default:
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal merotasi kode portal")
		}
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"portal_code":         result.Code,
		"portal_code_version": result.Version,
		"portal_access_mode":  "CODE",
		"reveal_once":         true,
	})
}
