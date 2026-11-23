package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBufferedJSONEndpoints_RejectOversizedBody(t *testing.T) {
	_, s := setupV1TestEnv(t)
	admin := helperLogin(t, s, "+6281111111111", "password123")
	body := strings.Repeat(" ", 10*1024*1024+1)
	for _, path := range []string{
		"/api/v1/semesters/1/import-validate",
		"/api/v1/master/rooms", "/api/v1/master/rooms/bulk",
		"/api/v1/master/courses", "/api/v1/master/courses/bulk",
		"/api/v1/master/lecturers", "/api/v1/master/lecturers/bulk",
	} {
		t.Run(path, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
			r.ContentLength = -1
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Authorization", "Bearer "+admin)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, r)
			if w.Code != http.StatusRequestEntityTooLarge || !strings.Contains(w.Body.String(), "PAYLOAD_TOO_LARGE") {
				t.Fatalf("expected 413, got %d %s", w.Code, w.Body.String())
			}
		})
	}
}
