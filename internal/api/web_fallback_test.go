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
