package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func decodeData(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var env struct {
		Status string         `json:"status"`
		Data   map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	return env.Data
}

func TestBE007_PatchSuccessContract(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	// Isolasi dari seed event EXTRA (CURRENT_TIMESTAMP) agar uji pola
	// deterministik terhadap jam dinding.
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	data := decodeData(t, w)
	if _, ok := data["replaces_pattern_id"]; !ok {
		t.Fatalf("response wajib replaces_pattern_id: %v", data)
	}
	if _, ok := data["effective_from"]; !ok {
		t.Fatalf("response wajib effective_from: %v", data)
	}
	if _, ok := data["effective_until"]; !ok {
		t.Fatalf("response wajib effective_until: %v", data)
	}
	if v, _ := data["version"].(float64); v != 2 {
		t.Fatalf("version baru diharapkan 2: %v", data)
	}
}

func TestBE007_PatchStaleConflict(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	ok := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(ok))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch pertama expected 200: %s", w.Body.String())
	}
	req2 := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(ok))
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Fatalf("patch stale expected 409, got %d: %s", w2.Code, w2.Body.String())
	}
	if !strings.Contains(w2.Body.String(), "current_version") {
		t.Fatalf("conflict wajib memuat current_version: %s", w2.Body.String())
	}
}

func TestBE007_ConcurrentPatchSingleWinner(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":3,"start_time":"11:00","duration_min":100,"version":1}`
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			codes[idx] = w.Code
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, c := range codes {
		if c == http.StatusOK {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("tepat satu pemenang diharapkan, got %v", codes)
	}
}

func TestBE007_CreateLecturerInvalid(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"offering_id":1,"day_of_week":2,"start_time":"08:00","duration_min":100,"lecturer_ids":[9999]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("lecturer invalid expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE007_EffectiveRangeHistory(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch expected 200: %s", w.Body.String())
	}
	var oldUntil, newFrom string
	var newID int64
	_ = db.QueryRow(`SELECT effective_until FROM schedule_patterns WHERE id=1`).Scan(&oldUntil)
	_ = db.QueryRow(`SELECT id, effective_from FROM schedule_patterns WHERE id != 1 AND course_offering_id=1 ORDER BY id DESC LIMIT 1`).Scan(&newID, &newFrom)
	if newID == 0 || newFrom == "" {
		t.Fatalf("pola baru tidak ditemukan")
	}
	if oldUntil > newFrom {
		t.Fatalf("pola lama harus berakhir sebelum pola baru mulai: %s vs %s", oldUntil, newFrom)
	}
}
