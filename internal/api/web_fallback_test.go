package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestWebUnknownHTML_ServesCustom404(t *testing.T) {
	s := newTestServer(t)

	rr := performRequest(t, s, "GET", "/systemaefaeadmin.html", nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("Status = %d, mau %d", rr.Code, http.StatusNotFound)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, mau text/html", ct)
	}
	if !strings.Contains(rr.Body.String(), "Halaman tidak ditemukan") {
		t.Fatalf("Body tidak memuat halaman 404 kustom")
	}
}

func TestWebKnownPage_StillServed(t *testing.T) {
	s := newTestServer(t)

	for _, p := range []string{"/km.html", "/404.html"} {
		rr := performRequest(t, s, "GET", p, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, mau 200", p, rr.Code)
		}
	}
}

func TestIsWebPagePath(t *testing.T) {
	for p, want := range map[string]bool{
		"/":                   false,
		"/c":                  false,
		"/c/":                 false,
		"/km.html":            true,
		"/salah-ketik.html":   true,
		"/tentang":            true,
		"/api/v1/tasks":       false,
		"/c/d4-ti-2024-a":     false,
		"/js/app-km.js":       false,
		"/assets/icons/x.svg": false,
		"/partials/a.html":    true,
	} {
		if got := isWebPagePath(p); got != want {
			t.Errorf("isWebPagePath(%q) = %v, mau %v", p, got, want)
		}
	}
}

func TestPWARoutes_ServedWithCorrectHeaders(t *testing.T) {
	s := newTestServer(t)

	// 1. Service Worker: no-cache, Service-Worker-Allowed: /
	rr := performRequest(t, s, "GET", "/sw.js", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /sw.js status = %d, mau 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Fatalf("GET /sw.js Content-Type = %q, mau javascript", ct)
	}
	if swa := rr.Header().Get("Service-Worker-Allowed"); swa != "/" {
		t.Fatalf("GET /sw.js Service-Worker-Allowed = %q, mau /", swa)
	}
	if cc := rr.Header().Get("Cache-Control"); !strings.Contains(cc, "no-cache") {
		t.Fatalf("GET /sw.js Cache-Control = %q, mau no-cache", cc)
	}

	// 2. Manifests: application/manifest+json
	for _, p := range []string{"/manifest.webmanifest", "/manifest.json"} {
		rr := performRequest(t, s, "GET", p, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, mau 200", p, rr.Code)
		}
		if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
			t.Fatalf("GET %s Content-Type = %q, mau json", p, ct)
		}
		if !strings.Contains(rr.Body.String(), "Bot Jadwal") {
			t.Fatalf("GET %s tidak memuat nama 'Bot Jadwal'", p)
		}
		if !strings.Contains(rr.Body.String(), "/assets/icons/icon-192.png") {
			t.Fatalf("GET %s tidak memuat icon-192.png", p)
		}
	}

	// 3. PWA Icons: 200 OK
	for _, icon := range []string{
		"/assets/icons/icon-192.png",
		"/assets/icons/icon-512.png",
		"/assets/icons/icon-maskable-192.png",
		"/assets/icons/icon-maskable-512.png",
		"/assets/icons/favicon.svg",
	} {
		rr := performRequest(t, s, "GET", icon, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, mau 200", icon, rr.Code)
		}
	}
}
