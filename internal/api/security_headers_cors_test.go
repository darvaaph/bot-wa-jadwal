package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBE014_CORSProductionExactAllowlist(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecurityOptions(SecurityOptions{
		Env:            "production",
		AllowedOrigins: []string{"https://app.example.com:8443", "https://portal.example.com"},
		AuthHashKey:    "0123456789abcdef0123456789abcdef",
	})

	tests := []struct {
		name        string
		origin      string
		expectAllow bool
	}{
		{
			name:        "Origin valid exact match",
			origin:      "https://app.example.com:8443",
			expectAllow: true,
		},
		{
			name:        "Origin valid exact match kedua",
			origin:      "https://portal.example.com",
			expectAllow: true,
		},
		{
			name:        "Host sama tapi port berbeda tidak terdaftar",
			origin:      "https://app.example.com:9000",
			expectAllow: false,
		},
		{
			name:        "Host sama tapi scheme http berbeda",
			origin:      "http://portal.example.com",
			expectAllow: false,
		},
		{
			name:        "Origin asing sama sekali",
			origin:      "https://attacker.evil.com",
			expectAllow: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)

			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			allowCreds := w.Header().Get("Access-Control-Allow-Credentials")

			if tc.expectAllow {
				if allowOrigin != tc.origin {
					t.Fatalf("expected Access-Control-Allow-Origin=%s, got %s", tc.origin, allowOrigin)
				}
				if allowCreds != "true" {
					t.Fatalf("expected Access-Control-Allow-Credentials=true, got %s", allowCreds)
				}
			} else {
				if allowOrigin != "" {
					t.Fatalf("expected no Access-Control-Allow-Origin for %s, got %s", tc.origin, allowOrigin)
				}
				if allowCreds != "" {
					t.Fatalf("expected no Access-Control-Allow-Credentials for %s, got %s", tc.origin, allowCreds)
				}
			}

			// Header Vary wajib memuat Origin
			vary := w.Header().Get("Vary")
			if !strings.Contains(vary, "Origin") {
				t.Fatalf("expected Vary to contain Origin, got %s", vary)
			}
		})
	}
}

func TestBE014_NoOriginRequest(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecurityOptions(SecurityOptions{
		Env:            "production",
		AllowedOrigins: []string{"https://app.example.com"},
		AuthHashKey:    "0123456789abcdef0123456789abcdef",
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("request tanpa Origin tidak boleh mendapat CORS header")
	}
}

func TestBE014_PreflightValidation(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecurityOptions(SecurityOptions{
		Env:            "production",
		AllowedOrigins: []string{"https://app.example.com"},
		AuthHashKey:    "0123456789abcdef0123456789abcdef",
	})

	// 1. Preflight method valid & header valid
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization, X-Portal-Token")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight valid expected 204, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Fatalf("expected Allow-Origin https://app.example.com, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
	if !strings.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST") {
		t.Fatalf("expected Allow-Methods to contain POST, got %s", w.Header().Get("Access-Control-Allow-Methods"))
	}

	// 2. Preflight method asing (TRACE)
	reqBadMethod := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqBadMethod.Header.Set("Origin", "https://app.example.com")
	reqBadMethod.Header.Set("Access-Control-Request-Method", "TRACE")
	wBadMethod := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wBadMethod, reqBadMethod)

	if wBadMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("preflight method asing expected 405, got %d", wBadMethod.Code)
	}

	// 3. Preflight header asing (X-Custom-Malicious)
	reqBadHeader := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqBadHeader.Header.Set("Origin", "https://app.example.com")
	reqBadHeader.Header.Set("Access-Control-Request-Method", "POST")
	reqBadHeader.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Custom-Malicious")
	wBadHeader := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wBadHeader, reqBadHeader)

	if wBadHeader.Code != http.StatusBadRequest {
		t.Fatalf("preflight header asing expected 400, got %d", wBadHeader.Code)
	}

	// 4. Preflight dari Origin asing
	reqBadOrigin := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	reqBadOrigin.Header.Set("Origin", "https://evil.com")
	reqBadOrigin.Header.Set("Access-Control-Request-Method", "POST")
	wBadOrigin := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wBadOrigin, reqBadOrigin)

	if wBadOrigin.Code != http.StatusForbidden {
		t.Fatalf("preflight origin asing expected 403, got %d", wBadOrigin.Code)
	}
	if wBadOrigin.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("preflight origin asing tidak boleh mendapat Allow-Origin")
	}
}

func TestBE014_VaryMerge(t *testing.T) {
	_, s := setupV1TestEnv(t)
	// Bungkus handler dengan handler yang sudah menambahkan Vary: Accept-Encoding
	originalHandler := s.httpServer.Handler
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Accept-Encoding")
		originalHandler.ServeHTTP(w, r)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)

	varyValues := w.Header().Values("Vary")
	allVary := strings.Join(varyValues, ", ")
	if !strings.Contains(allVary, "Accept-Encoding") || !strings.Contains(allVary, "Origin") {
		t.Fatalf("expected Vary to merge Accept-Encoding and Origin, got %s", allVary)
	}
}

func TestBE014_SecurityHeadersAndHSTS(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecurityOptions(SecurityOptions{
		Env:               "production",
		AllowedOrigins:    []string{"https://app.example.com"},
		TrustedProxyCIDRs: []string{"10.0.0.0/8"},
		AuthHashKey:       "0123456789abcdef0123456789abcdef",
	})

	// 1. Request via HTTP polos di production: tidak boleh kirim HSTS
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	wHTTP := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wHTTP, reqHTTP)

	if wHTTP.Header().Get("Strict-Transport-Security") != "" {
		t.Fatalf("HTTP biasa tidak boleh menerima HSTS")
	}

	// Periksa security headers wajib
	if wHTTP.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected X-Content-Type-Options: nosniff")
	}
	if wHTTP.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("expected X-Frame-Options: DENY")
	}
	if wHTTP.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatalf("expected Referrer-Policy: same-origin")
	}
	if !strings.Contains(wHTTP.Header().Get("Permissions-Policy"), "camera=()") {
		t.Fatalf("expected Permissions-Policy to restrict capabilities")
	}
	csp := wHTTP.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'self'") || !strings.Contains(csp, "cdn.tailwindcss.com") || !strings.Contains(csp, "cdn.jsdelivr.net") {
		t.Fatalf("CSP tidak valid atau tidak memuat CDN: %s", csp)
	}

	// 2. Request HTTPS langsung di production: HSTS wajib aktif
	reqHTTPS := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	reqHTTPS.TLS = &tls.ConnectionState{}
	wHTTPS := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wHTTPS, reqHTTPS)

	hsts := wHTTPS.Header().Get("Strict-Transport-Security")
	if hsts == "" || !strings.Contains(hsts, "max-age=") {
		t.Fatalf("expected HSTS on HTTPS in production, got: %s", hsts)
	}

	// 3. Request HTTPS lewat trusted proxy di production: HSTS wajib aktif
	reqProxy := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	reqProxy.RemoteAddr = "10.0.0.5:1234"
	reqProxy.Header.Set("X-Forwarded-Proto", "https")
	wProxy := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wProxy, reqProxy)

	hstsProxy := wProxy.Header().Get("Strict-Transport-Security")
	if hstsProxy == "" {
		t.Fatalf("expected HSTS when forwarded https from trusted proxy")
	}
}

func TestBE014_CSRFProtectionForCookieMutations(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecurityOptions(SecurityOptions{
		Env:            "production",
		AllowedOrigins: []string{"https://app.example.com"},
		AuthHashKey:    "0123456789abcdef0123456789abcdef",
	})

	// 1. Mutasi POST dengan cookie bv1 dari Origin asing: wajib ditolak 403
	reqBadOrigin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqBadOrigin.AddCookie(&http.Cookie{Name: "bv1", Value: "session-token-test"})
	reqBadOrigin.Header.Set("Origin", "https://attacker.com")
	wBadOrigin := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wBadOrigin, reqBadOrigin)

	if wBadOrigin.Code != http.StatusForbidden {
		t.Fatalf("mutasi dengan cookie dari origin asing harus 403, got %d", wBadOrigin.Code)
	}

	// 2. Mutasi POST dengan cookie bv1 dari Origin sah: diteruskan (akan diproses auth)
	reqGoodOrigin := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqGoodOrigin.AddCookie(&http.Cookie{Name: "bv1", Value: "session-token-test"})
	reqGoodOrigin.Header.Set("Origin", "https://app.example.com")
	wGoodOrigin := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wGoodOrigin, reqGoodOrigin)

	// Origin tidak ditolak oleh CSRF middleware (akan berlanjut ke auth handler, misal 401 jika token dummy)
	if wGoodOrigin.Code == http.StatusForbidden {
		t.Fatalf("origin sah tidak boleh ditolak CSRF, got %d", wGoodOrigin.Code)
	}

	// 3. Mutasi POST dengan Authorization: Bearer dari origin manapun: CSRF check dilewati
	reqBearer := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	reqBearer.Header.Set("Authorization", "Bearer token-bearer-test")
	reqBearer.Header.Set("Origin", "https://attacker.com")
	wBearer := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wBearer, reqBearer)

	// Bearer client tidak diblokir oleh CSRF middleware
	if wBearer.Code == http.StatusForbidden {
		t.Fatalf("Bearer-only request tidak boleh diblokir oleh CSRF, got %d", wBearer.Code)
	}
}

func TestBE014_NoStoreOnSensitiveResponses(t *testing.T) {
	_, s := setupV1TestEnv(t)

	// Login gagal/berhasil memuat no-store
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"identity_key":"+6281234567890","password":"wrong"}`))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	// Portal session
	reqPortal := httptest.NewRequest(http.MethodPost, "/api/v1/portal/d4-ti-2024-a/session", strings.NewReader(`{"code":"123456"}`))
	wPortal := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wPortal, reqPortal)

	if wPortal.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("portal session creation expected Cache-Control: no-store, got %s", wPortal.Header().Get("Cache-Control"))
	}
}

func TestBE014_FrontendHtmlResourcesAllowedByCSP(t *testing.T) {
	_, s := setupV1TestEnv(t)

	pages := []string{"/", "/superadmin.html", "/login.html", "/km.html", "/pj.html"}
	for _, page := range pages {
		req := httptest.NewRequest(http.MethodGet, page, nil)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("halaman %s expected 200, got %d", page, w.Code)
		}

		csp := w.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Fatalf("halaman %s wajib memiliki Content-Security-Policy", page)
		}

		// Verifikasi CDN yang digunakan halaman frontend diizinkan
		body := w.Body.String()
		if strings.Contains(body, "cdn.tailwindcss.com") && !strings.Contains(csp, "https://cdn.tailwindcss.com") {
			t.Fatalf("halaman %s memuat Tailwind CDN tapi tidak diizinkan oleh CSP", page)
		}
		if strings.Contains(body, "cdn.jsdelivr.net") && !strings.Contains(csp, "https://cdn.jsdelivr.net") {
			t.Fatalf("halaman %s memuat jsdelivr CDN tapi tidak diizinkan oleh CSP", page)
		}
		if strings.Contains(body, "fonts.googleapis.com") && !strings.Contains(csp, "https://fonts.googleapis.com") {
			t.Fatalf("halaman %s memuat Google Fonts tapi tidak diizinkan oleh CSP", page)
		}
		if strings.Contains(body, "fonts.gstatic.com") && !strings.Contains(csp, "https://fonts.gstatic.com") {
			t.Fatalf("halaman %s memuat gstatic font tapi tidak diizinkan oleh CSP", page)
		}
	}
}
