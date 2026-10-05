package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1Portal_Semesters(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Daftar semester kelas ada
	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/semesters", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET semesters expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON: %v", err)
	}
	if len(body.Data) == 0 {
		t.Fatalf("daftar semester kosong")
	}
	for _, sm := range body.Data {
		for _, kunci := range []string{"id", "academic_year", "term", "starts_on", "ends_on", "status"} {
			if _, ok := sm[kunci]; !ok {
				t.Errorf("semester tanpa field %s: %v", kunci, sm)
			}
		}
	}

	// 2. Kelas tidak ada -> 404
	req = httptest.NewRequest("GET", "/api/v1/portal/kelas-fiktif/semesters", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("kelas fiktif expected 404, got %d", w.Code)
	}
}
