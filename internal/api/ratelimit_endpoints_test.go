package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bot-jadwal/internal/ratelimit"
)

func TestBE012_LoginBlockAndRetryAfter(t *testing.T) {
	_, s := setupV1TestEnv(t)
	bad := `{"identity_key":"+6281234567890","password":"salah-salah-salah"}`
	var last *httptest.ResponseRecorder
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(bad))
		req.RemoteAddr = "198.51.100.7:4000"
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		last = w
		if i < 5 && w.Code == http.StatusTooManyRequests {
			t.Fatalf("percobaan %d belum boleh diblokir: %s", i+1, w.Body.String())
		}
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan ke-6 expected 429, got %d: %s", last.Code, last.Body.String())
	}
	if last.Header().Get("Retry-After") == "" {
		t.Fatalf("response 429 wajib Retry-After")
	}
	// Port berbeda tidak mengubah bucket: попыtka dengan port lain tetap diblokir.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(bad))
	req.RemoteAddr = "198.51.100.7:9999"
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("port berbeda expected tetap 429, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE012_PortalLimitPersistentAcrossRebuild(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	if err := s.portalService.SetClassCode(context.Background(), 1, "123456"); err != nil {
		t.Fatal(err)
	}
	// Kunci eksplisit bersama agar dua instance menghitung state yang sama.
	shared, err := ratelimit.NewService(db, []byte("0123456789abcdef0123456789abcdef"), nil)
	if err != nil {
		t.Fatal(err)
	}
	s.SetRateLimiter(shared)
	bad := `{"code":"000000"}`
	var last *httptest.ResponseRecorder
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/d4-ti-2024-a/session", strings.NewReader(bad))
		req.RemoteAddr = "203.0.113.9:1234"
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		last = w
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", last.Code, last.Body.String())
	}
	// Server dibuat ulang di atas database sama dengan key yang sama:
	// blokir harus bertahan.
	s2 := NewServer(":0", nil, nil, nil, db)
	s2.SetRateLimiter(shared)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/d4-ti-2024-a/session", strings.NewReader(bad))
	req.RemoteAddr = "203.0.113.9:1234"
	w := httptest.NewRecorder()
	s2.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("blokir harus bertahan lintas rebuild, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE012_InviteAcceptLimited(t *testing.T) {
	_, s := setupV1TestEnv(t)
	var last *httptest.ResponseRecorder
	for i := 0; i < 12; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/invitations/accept", strings.NewReader(`{"token":"palsu-012","password":"password123456"}`))
		req.RemoteAddr = "192.0.2.44:5000"
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		last = w
	}
	if last.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 setelah threshold, got %d: %s", last.Code, last.Body.String())
	}
}
