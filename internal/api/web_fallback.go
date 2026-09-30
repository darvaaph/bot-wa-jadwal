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
