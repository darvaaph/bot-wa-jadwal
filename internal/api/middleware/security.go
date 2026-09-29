package middleware

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

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/ratelimit"
)

func hmacNew(key []byte) hash.Hash { return hmac.New(sha256.New, key) }

// SecurityOptions menyalurkan konfigurasi security BE-013 ke middleware.
type SecurityOptions struct {
	Env               string
	AuthHashKey       string
	AllowedOrigins    []string
	TrustedProxyCIDRs []string
	PublicBaseURL     string
	SecureCookies     bool
}

// SecurityManager mengatur proteksi CORS, security headers, CSRF, dan cookie autentikasi.
type SecurityManager struct {
	env               string
	authHashKey       []byte
	allowedOrigins    []string
	trustedProxyCIDRs []string
	publicBaseURL     string
	secureCookies     bool
}

// NewSecurityManager membuat instance baru SecurityManager dengan konfigurasi aman.
func NewSecurityManager(opt SecurityOptions) *SecurityManager {
	sm := &SecurityManager{}
	sm.ApplyOptions(opt)
	return sm
}

// ApplyOptions memperbarui opsi keamanan.
func (sm *SecurityManager) ApplyOptions(opt SecurityOptions) {
	sm.env = strings.ToLower(strings.TrimSpace(opt.Env))
	if sm.env == "" {
		sm.env = "development"
	}
	sm.authHashKey = []byte(opt.AuthHashKey)
	sm.publicBaseURL = strings.TrimSpace(opt.PublicBaseURL)
	sm.secureCookies = opt.SecureCookies

	sm.allowedOrigins = nil
	seen := map[string]bool{}
	for _, o := range opt.AllowedOrigins {
		n := NormalizeOriginValue(o)
		if n != "" && !seen[n] {
			seen[n] = true
			sm.allowedOrigins = append(sm.allowedOrigins, n)
		}
	}

	sm.trustedProxyCIDRs = nil
	for _, c := range opt.TrustedProxyCIDRs {
		if trimmed := strings.TrimSpace(c); trimmed != "" {
			sm.trustedProxyCIDRs = append(sm.trustedProxyCIDRs, trimmed)
		}
	}
}

// SetSecureCookies mengatur status secure flag pada cookie.
func (sm *SecurityManager) SetSecureCookies(enabled bool) {
	sm.secureCookies = enabled
}

// IsProduction melaporkan apakah environment saat ini adalah production.
func (sm *SecurityManager) IsProduction() bool {
	return sm.env == "production"
}

// AuthHashKey mengembalikan key hash HMAC yang tersimpan.
func (sm *SecurityManager) AuthHashKey() []byte {
	return sm.authHashKey
}

// TrustedProxyCIDRs mengembalikan daftar CIDR proxy tepercaya.
func (sm *SecurityManager) TrustedProxyCIDRs() []string {
	return sm.trustedProxyCIDRs
}

// Fingerprint menghitung HMAC-SHA256 namespaced bila AuthHashKey tersedia,
// atau fallback SHA-256 plain (development/test tanpa key).
func (sm *SecurityManager) Fingerprint(namespace, value string) string {
	if len(sm.authHashKey) >= 32 {
		mac := hmacNew(sm.authHashKey)
		_, _ = mac.Write([]byte(namespace))
		_, _ = mac.Write([]byte{0})
		_, _ = mac.Write([]byte(value))
		return hex.EncodeToString(mac.Sum(nil))
	}
	sum := sha256.Sum256([]byte(namespace + "\x00" + value))
	return hex.EncodeToString(sum[:])
}

// SetAuthCookie menyatukan atribut cookie login, switch context, dan logout.
func (sm *SecurityManager) SetAuthCookie(w http.ResponseWriter, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   sm.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearAuthCookie menghapus cookie dengan atribut yang cocok dengan setter.
func (sm *SecurityManager) ClearAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   sm.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

// NormalizeOriginValue menstandarisasi origin string ke format scheme://host(:port).
func NormalizeOriginValue(origin string) string {
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

// IsOriginAllowed memeriksa apakah origin diizinkan berdasarkan exact-match allowlist.
func (sm *SecurityManager) IsOriginAllowed(origin string) bool {
	norm := NormalizeOriginValue(origin)
	if norm == "" {
		return false
	}
	for _, allowed := range sm.allowedOrigins {
		if norm == allowed {
			return true
		}
	}
	if sm.publicBaseURL != "" && norm == NormalizeOriginValue(sm.publicBaseURL) {
		return true
	}
	if sm.IsProduction() {
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

// IsRequestHTTPS memeriksa apakah request berjalan di atas HTTPS.
func (sm *SecurityManager) IsRequestHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	peer := ratelimit.PeerIP(r.RemoteAddr)
	if ratelimit.PeerTrusted(peer, sm.trustedProxyCIDRs) {
		if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			return true
		}
	}
	return false
}

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

func isAllowedCORSMethod(m string) bool {
	switch strings.ToUpper(strings.TrimSpace(m)) {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions, http.MethodHead:
		return true
	default:
		return false
	}
}

func isAllowedCORSHeader(h string) bool {
	switch strings.ToLower(strings.TrimSpace(h)) {
	case "content-type", "authorization", "idempotency-key", "x-portal-token", "x-requested-with", "accept", "origin":
		return true
	default:
		return false
	}
}

// CORS menegakkan CORS exact-origin, CSRF check, dan HTTP security headers (BE-014).
func (sm *SecurityManager) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Security Headers dasar & Content Security Policy (BE-014)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.tailwindcss.com https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self';")

		if sm.IsProduction() && sm.IsRequestHTTPS(r) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		origin := r.Header.Get("Origin")

		// 2. CORS Preflight (OPTIONS dengan Access-Control-Request-Method)
		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			if origin == "" || !sm.IsOriginAllowed(origin) {
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
			if sm.IsOriginAllowed(origin) {
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
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			cookie, err := r.Cookie("bv1")
			hasCookie := err == nil && strings.TrimSpace(cookie.Value) != ""
			hasBearer := strings.HasPrefix(strings.ToLower(r.Header.Get("Authorization")), "bearer ")
			if hasCookie && !hasBearer {
				if origin != "" {
					if !sm.IsOriginAllowed(origin) {
						common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Origin tidak diizinkan untuk mutasi kredensial cookie")
						return
					}
				} else {
					referer := r.Header.Get("Referer")
					if referer != "" {
						refURL, err := url.Parse(referer)
						if err != nil || !sm.IsOriginAllowed(refURL.Scheme+"://"+refURL.Host) {
							common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Referer tidak diizinkan untuk mutasi kredensial cookie")
							return
						}
					} else if sm.IsProduction() {
						common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Origin atau Referer wajib untuk mutasi kredensial cookie")
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery menangani panic HTTP agar server web tidak crash.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("⚠️ [HTTP Panic] %v\n", rec)
				common.WriteJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "Terjadi kesalahan internal server",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
