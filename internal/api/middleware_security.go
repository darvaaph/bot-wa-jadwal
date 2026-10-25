package api

import (
	"net/http"

	"bot-jadwal/internal/api/middleware"
)

// SecurityOptions menyalurkan konfigurasi security BE-013 ke Server.
type SecurityOptions = middleware.SecurityOptions

// SetSecurityOptions menerapkan konfigurasi security dari Config.
func (s *Server) SetSecurityOptions(opt SecurityOptions) {
	if s.secManager == nil {
		s.secManager = middleware.NewSecurityManager(opt)
	} else {
		s.secManager.ApplyOptions(opt)
	}
	s.env = opt.Env
	s.authHashKey = s.secManager.AuthHashKey()
	s.publicBaseURL = opt.PublicBaseURL
	s.allowedOrigins = opt.AllowedOrigins
	s.trustedProxyCIDRs = s.secManager.TrustedProxyCIDRs()
	s.buildLimiter()
	s.configureRecovery()
}

// SetSecureCookies mengaktifkan atribut Secure pada seluruh cookie autentikasi.
func (s *Server) SetSecureCookies(enabled bool) {
	s.secureCookies = enabled
	if s.secManager != nil {
		s.secManager.SetSecureCookies(enabled)
	}
}

// corsMiddleware menegakkan CORS exact-origin, CSRF check, dan HTTP security headers (BE-014).
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	if s.secManager != nil {
		return s.secManager.CORS(next)
	}
	return next
}

// recoveryMiddleware menangani panic HTTP agar server web tidak crash.
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return middleware.Recovery(next)
}

// compressMiddleware mengompresi respons jika klien mendukung header Accept-Encoding: gzip.
func (s *Server) compressMiddleware(next http.Handler) http.Handler {
	return middleware.Gzip(next)
}
