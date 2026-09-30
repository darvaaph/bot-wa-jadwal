package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1MeetingLink_PatternAndEvent(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. Buat pola dengan tautan daring
	pbody, _ := json.Marshal(map[string]any{
		"offering_id": 1, "day_of_week": 6, "start_time": "08:00",
		"duration_min": 100, "meeting_link": "https://meet.example.com/kelas-1",
	})
	req := httptest.NewRequest("POST", "/api/v1/schedule/patterns", bytes.NewReader(pbody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST patterns expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Daftar pola memuat tautan
	req = httptest.NewRequest("GET", "/api/v1/schedule/patterns", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("https://meet.example.com/kelas-1")) {
		t.Fatalf("daftar pola tidak memuat meeting_link: %d %s", w.Code, w.Body.String())
	}

	// 3. Buat draf perubahan dengan tautan
	ebody, _ := json.Marshal(map[string]any{
		"owner_offering_id": 1, "event_kind": "EXTRA",
		"starts_at": "2024-10-05T09:00:00+07:00", "ends_at": "2024-10-05T11:00:00+07:00",
		"reason": "Kelas tambahan", "meeting_link": "https://meet.example.com/tambahan",
	})
	req = httptest.NewRequest("POST", "/api/v1/teaching-events", bytes.NewReader(ebody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST events expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data["id"] == nil {
		t.Fatalf("respons draf tidak memuat id: %v %s", err, w.Body.String())
	}
	eventID := int(created.Data["id"].(float64))

	// 4. Pratinjau memuat tautan
	req = httptest.NewRequest("POST", "/api/v1/teaching-events/"+itoa2(eventID)+"/preview", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("https://meet.example.com/tambahan")) {
		t.Fatalf("preview tidak memuat meeting_link: %d %s", w.Code, w.Body.String())
	}

	// 5. Daftar events memuat tautan
	req = httptest.NewRequest("GET", "/api/v1/teaching-events", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("https://meet.example.com/tambahan")) {
		t.Fatalf("daftar events tidak memuat meeting_link: %d %s", w.Code, w.Body.String())
	}
}

func itoa2(n int) string {
	if n == 0 {
		return "0"
	}
	out := ""
	for n > 0 {
		out = string(rune('0'+n%10)) + out
		n /= 10
	}
	return out
}
