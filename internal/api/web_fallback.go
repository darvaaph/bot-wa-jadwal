package api

import (
	"net/http"
	"path"
	"strings"

	"bot-jadwal/web"
)

// isWebPagePath melaporkan apakah path meminta halaman web.
// Hanya untuk path ini 404 polos FileServer diganti halaman 404 kustom.
func isWebPagePath(p string) bool {
	if p == "" || p == "/" {
		return false
	}
	if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/c/") {
		return false
	}
	ext := path.Ext(p)
	return ext == "" || ext == ".html"
}

// redirectToLogin mengalihkan ke /login.html dengan query string dipertahankan.
// Dipakai bersama oleh handler "/" dan "/index.html" agar semantik GET/HEAD sama.
func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	target := "/login.html"
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// serveWebNotFound menyajikan halaman 404 kustom dengan status 404.
func serveWebNotFound(w http.ResponseWriter, r *http.Request) {
	page, err := web.Files.ReadFile("404.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write(page)
}
