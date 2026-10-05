package api

import (
	"net/http"
	"time"

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

// isProduction melaporkan environment production eksplisit.
func (s *Server) isProduction() bool {
	if s.secManager != nil {
		return s.secManager.IsProduction()
	}
	return s.env == "production"
}

// fingerprint menghitung HMAC-SHA256 namespaced bila AuthHashKey tersedia.
func (s *Server) fingerprint(namespace, value string) string {
	if s.secManager != nil {
		return s.secManager.Fingerprint(namespace, value)
	}
	return value
}

// SetSecureCookies mengaktifkan atribut Secure pada seluruh cookie autentikasi.
func (s *Server) SetSecureCookies(enabled bool) {
	s.secureCookies = enabled
	if s.secManager != nil {
		s.secManager.SetSecureCookies(enabled)
	}
}

// setAuthCookie menyatukan atribut cookie login, switch context, dan logout.
func (s *Server) setAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	if s.secManager != nil {
		s.secManager.SetAuthCookie(w, token, expires)
		return
	}
}

// clearAuthCookie menghapus cookie dengan atribut yang cocok dengan setter.
func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	if s.secManager != nil {
		s.secManager.ClearAuthCookie(w)
		return
	}
}

// noStore menandai response pembawa token agar tidak di-cache.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// isOriginAllowed memeriksa apakah origin diizinkan berdasarkan exact-match allowlist.
func (s *Server) isOriginAllowed(origin string) bool {
	if s.secManager != nil {
		return s.secManager.IsOriginAllowed(origin)
	}
	return false
}

// isRequestHTTPS memeriksa apakah request berjalan di atas HTTPS.
func (s *Server) isRequestHTTPS(r *http.Request) bool {
	if s.secManager != nil {
		return s.secManager.IsRequestHTTPS(r)
	}
	return false
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
