package api

import (
	"net/http"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
)

// UserContext menyimpan data sesi dan konteks penugasan peran aktif dari token pengguna.
type UserContext = common.UserContext

// GetAuthContext mengambil konteks pengguna terautentikasi dari request context.
func GetAuthContext(r *http.Request) (*UserContext, bool) {
	return common.GetAuthContext(r)
}

// RequireAuth memverifikasi token Bearer pengguna, masa berlaku sesi (idle & absolute), serta session_version.
func (s *Server) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	if s.authManager == nil {
		s.authManager = middleware.NewAuthManager(s.v1DB)
	}
	return s.authManager.RequireAuth(next)
}

// RequireRole memastikan pengguna terautentikasi memiliki salah satu peran yang diizinkan.
func (s *Server) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.RequireRole(roles...)
}
