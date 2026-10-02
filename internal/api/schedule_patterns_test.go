package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func setPatternSemesterCurrent(t *testing.T, db *sql.DB) {
	t.Helper()
	now := time.Now()
	_, err := db.Exec(`UPDATE semesters SET starts_on=?, ends_on=? WHERE id=1`,
		now.AddDate(0, 0, -7).Format("2006-01-02"), now.AddDate(0, 0, 90).Format("2006-01-02"))
	if err != nil {
		t.Fatal(err)
	}
}

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
	setPatternSemesterCurrent(t, db)
	// Isolasi dari seed event EXTRA (CURRENT_TIMESTAMP) agar uji pola
	// deterministik terhadap jam dinding.
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1,"reason":"Perubahan rutin dosen"}`
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
	setPatternSemesterCurrent(t, db)
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	ok := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1,"reason":"Perubahan rutin dosen"}`
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
	setPatternSemesterCurrent(t, db)
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":3,"start_time":"11:00","duration_min":100,"version":1,"reason":"Perubahan rutin dosen"}`
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

func TestBE007_CreatePreviewChecksLaterSemesterOccurrence(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	setPatternSemesterCurrent(t, db)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	first := time.Now().In(loc)
	for first.Weekday() != time.Wednesday {
		first = first.AddDate(0, 0, 1)
	}
	later := first.AddDate(0, 0, 7).Format("2006-01-02")
	_, err = db.Exec(`UPDATE teaching_events SET starts_at=?, ends_at=? WHERE id=1`, later+"T10:00:00+07:00", later+"T11:00:00+07:00")
	if err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"offering_id":1,"day_of_week":3,"start_time":"10:00","duration_min":60}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns/preview", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || decodeData(t, w)["can_publish"] != false {
		t.Fatalf("preview must block later conflict: %d %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("create must block later conflict: %d %s", w.Code, w.Body.String())
	}
}

func TestBE007_EffectiveRangeHistory(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	setPatternSemesterCurrent(t, db)
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1,"reason":"Perubahan rutin dosen"}`
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

func TestPatternPermanent_PreviewPublishSelectedDate(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	setPatternSemesterCurrent(t, db)
	_, _ = db.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = 1`)
	_, _ = db.Exec(`DELETE FROM teaching_events WHERE id = 1`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	effective := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1,"effective_from":"` + effective + `","reason":"Dosen mengganti hari kuliah"}`
	preview := httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns/1/preview", strings.NewReader(body))
	preview.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, preview)
	if w.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", w.Code, w.Body.String())
	}
	var before int
	_ = db.QueryRow(`SELECT COUNT(*) FROM schedule_patterns`).Scan(&before)
	if before != 1 {
		t.Fatalf("preview menulis data: %d", before)
	}
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", w.Code, w.Body.String())
	}
	var oldUntil, newFrom string
	_ = db.QueryRow(`SELECT effective_until FROM schedule_patterns WHERE id=1`).Scan(&oldUntil)
	_ = db.QueryRow(`SELECT effective_from FROM schedule_patterns WHERE id != 1`).Scan(&newFrom)
	parsed, _ := time.Parse("2006-01-02", effective)
	if !strings.HasPrefix(oldUntil, parsed.AddDate(0, 0, -1).Format("2006-01-02")) || !strings.HasPrefix(newFrom, effective) {
		t.Fatalf("tanggal versi: old=%s new=%s", oldUntil, newFrom)
	}
	var messages int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE entity_type='SCHEDULE_PATTERN'`).Scan(&messages)
	if messages != 1 {
		t.Fatalf("notifikasi permanen: %d", messages)
	}
}
