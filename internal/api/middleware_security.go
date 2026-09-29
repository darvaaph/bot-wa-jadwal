package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bot-jadwal/internal/ratelimit"
)

func hmacNew(key []byte) hash.Hash { return hmac.New(sha256.New, key) }

// SecurityOptions menyalurkan konfigurasi security BE-013 ke Server.
type SecurityOptions struct {
	Env               string
	AuthHashKey       string
	AllowedOrigins    []string
	TrustedProxyCIDRs []string
	PublicBaseURL     string
}

// SetSecurityOptions menerapkan konfigurasi security dari Config.
// Dipanggil sebelum Start; nilai origin dinormalisasi ke exact-match.
func (s *Server) SetSecurityOptions(opt SecurityOptions) {
	s.env = strings.ToLower(strings.TrimSpace(opt.Env))
	if s.env == "" {
		s.env = "development"
	}
	s.authHashKey = []byte(opt.AuthHashKey)
	s.publicBaseURL = strings.TrimSpace(opt.PublicBaseURL)
	s.allowedOrigins = nil
	seen := map[string]bool{}
	for _, o := range opt.AllowedOrigins {
		n := normalizeOriginValue(o)
		if n != "" && !seen[n] {
			seen[n] = true
			s.allowedOrigins = append(s.allowedOrigins, n)
		}
	}
	s.trustedProxyCIDRs = nil
	for _, c := range opt.TrustedProxyCIDRs {
		if trimmed := strings.TrimSpace(c); trimmed != "" {
			s.trustedProxyCIDRs = append(s.trustedProxyCIDRs, trimmed)
		}
	}
	// Catatan: hash penyimpanan token/kode portal tetap SHA-256 plain demi
	// kompatibilitas baris lama (token berentropi tinggi); HMAC keyed hanya
	// untuk fingerprint rate limiter (BE-012).
	s.buildLimiter()
}

func normalizeOriginValue(origin string) string {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if port := u.Port(); port != "" {
		return strings.ToLower(u.Scheme) + "://" + host + ":" + port
	}
	return strings.ToLower(u.Scheme) + "://" + host
}

// isProduction melaporkan environment production eksplisit.
func (s *Server) isProduction() bool { return s.env == "production" }

// fingerprint menghitung HMAC-SHA256 namespaced bila AuthHashKey tersedia,
// atau fallback SHA-256 plain (development/test tanpa key). Key tidak pernah
// dipakai untuk enkripsi password atau sebagai token.
func (s *Server) fingerprint(namespace, value string) string {
	if len(s.authHashKey) >= 32 {
		mac := hmacNew(s.authHashKey)
		_, _ = mac.Write([]byte(namespace))
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(value))
		return hex.EncodeToString(mac.Sum(nil))
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

// SetSecureCookies mengaktifkan atribut Secure pada seluruh cookie autentikasi.
func (s *Server) SetSecureCookies(enabled bool) {
	s.secureCookies = enabled
}

// setAuthCookie menyatukan atribut cookie login, switch context, dan logout:
// HttpOnly, Path /, SameSite Lax, Secure mengikuti environment, tanpa Domain.
func (s *Server) setAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookie menghapus cookie dengan atribut yang cocok dengan setter.
func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// noStore menandai response pembawa token agar tidak di-cache.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}

// isAllowedCORSMethod memeriksa apakah HTTP method diizinkan untuk CORS preflight.
func isAllowedCORSMethod(m string) bool {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead:
		return true
	default:
		return false
	}
}

// isAllowedCORSHeader memeriksa apakah HTTP header diizinkan untuk CORS preflight.
func isAllowedCORSHeader(h string) bool {
	switch strings.ToLower(strings.TrimSpace(h)) {
	case "content-type", "authorization", "idempotency-key", "x-portal-token", "x-requested-with", "accept", "origin":
		return true
	default:
		return false
	}
}

// isOriginAllowed memeriksa apakah origin diizinkan berdasarkan exact-match allowlist.
// Production HANYA mengizinkan origin yang terdaftar eksplisit di AllowedOrigins atau PublicBaseURL.
// Development & Test mengizinkan origin localhost eksplisit.
func (s *Server) isOriginAllowed(origin string) bool {
	norm := normalizeOriginValue(origin)
	if norm == "" {
		return false
	}
	for _, allowed := range s.allowedOrigins {
		if norm == allowed {
			return true
		}
	}
	if s.publicBaseURL != "" && norm == normalizeOriginValue(s.publicBaseURL) {
		return true
	}
	if s.isProduction() {
		return false
	}
	if u, err := url.Parse(norm); err == nil {
		h := strings.ToLower(u.Hostname())
		if h == "localhost" || h == "127.0.0.1" || h == "::1" {
			return true
		}
	}
	return false
}

// addVaryOrigin menggabungkan 'Origin' ke header Vary tanpa menimpa nilai yang ada.
func addVaryOrigin(w http.ResponseWriter) {
	existing := w.Header().Values("Vary")
	for _, v := range existing {
		for _, part := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(part), "origin") {
				return
			}
		}
	}
	if len(existing) == 0 {
		w.Header().Set("Vary", "Origin")
	} else {
		w.Header().Add("Vary", "Origin")
	}
}

// isRequestHTTPS memeriksa apakah request berjalan di atas HTTPS (langsung atau via trusted proxy).
func (s *Server) isRequestHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	peer := ratelimit.PeerIP(r.RemoteAddr)
	if ratelimit.PeerTrusted(peer, s.trustedProxyCIDRs) {
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	return false
}

// corsMiddleware menegakkan CORS exact-origin, CSRF check, dan HTTP security headers (BE-014).
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Security Headers dasar & Content Security Policy (BE-014)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self';")

		if s.isProduction() && s.isRequestHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		origin := r.Header.Get("Origin")

		// 2. CORS Preflight (OPTIONS dengan Access-Control-Request-Method)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if origin == "" || !s.isOriginAllowed(origin) {
				addVaryOrigin(w)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			reqMethod := r.Header.Get("Access-Control-Request-Method")
			if !isAllowedCORSMethod(reqMethod) {
				addVaryOrigin(w)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			reqHeaders := r.Header.Get("Access-Control-Request-Headers")
			if reqHeaders != "" {
				for _, h := range strings.Split(reqHeaders, ",") {
					if !isAllowedCORSHeader(h) {
						addVaryOrigin(w)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
				}
			}
			addVaryOrigin(w)
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Portal-Token")
			w.Header().Set("Access-Control-Max-Age", "86400")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 3. Response CORS untuk request non-preflight dengan header Origin
		if origin != "" {
			addVaryOrigin(w)
			if s.isOriginAllowed(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		// 4. OPTIONS non-preflight sederhana
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 5. CSRF Protection untuk cookie-authenticated mutations (POST/PUT/PATCH/DELETE)
		// Bearer-only client (bot WA / API token) dilewati tanpa CSRF check.
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			cookie, err := r.Cookie("bv1")
			hasCookie := err == nil && strings.TrimSpace(cookie.Value) != ""
			hasBearer := strings.HasPrefix(strings.ToLower(r.Header.Get("Authorization")), "bearer ")
			if hasCookie && !hasBearer {
				if origin != "" {
					if !s.isOriginAllowed(origin) {
						s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Origin tidak diizinkan untuk mutasi kredensial cookie")
						return
					}
				} else {
					referer := r.Header.Get("Referer")
					if referer != "" {
						refURL, err := url.Parse(referer)
						if err != nil || !s.isOriginAllowed(refURL.Scheme+"://"+refURL.Host) {
							s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Referer tidak diizinkan untuk mutasi kredensial cookie")
							return
						}
					} else if s.isProduction() {
						s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Origin atau Referer wajib untuk mutasi kredensial cookie")
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware menangani panic HTTP agar server web tidak crash
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("⚠️ [HTTP Panic] %v\n", rec)
				s.writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Terjadi kesalahan internal server",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
