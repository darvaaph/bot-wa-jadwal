package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDraft_CreateUsesSemesterStart(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, err := db.Exec(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status) VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type) VALUES (2, 2, 1, 'Struktur Data (Teori) Draf', 'TEORI')`)
	if err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"offering_id":2,"day_of_week":1,"start_time":"08:00","duration_min":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data := decodeData(t, w)
	if data["effective_from"] != "2025-09-01" {
		t.Fatalf("effective_from harus starts_on semester draf: %v", data)
	}
}

func TestDraft_PatchInPlaceNoReason(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status) VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT')`)
	_, _ = db.Exec(`INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type) VALUES (2, 2, 1, 'Struktur Data Draf', 'TEORI')`)
	_, _ = db.Exec(`INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from) VALUES (10, 2, 1, 1, '08:00', '09:40', '2025-09-01')`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"day_of_week":2,"start_time":"10:00","duration_min":100,"version":1}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/10", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	data := decodeData(t, w)
	if v, _ := data["version"].(float64); v != 2 {
		t.Fatalf("version harus 2: %v", data)
	}
	if data["id"] != data["replaces_pattern_id"] {
		t.Fatalf("draf harus in-place (id sama): %v", data)
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM schedule_patterns WHERE course_offering_id=2`).Scan(&count)
	if count != 1 {
		t.Fatalf("draf tidak boleh buat baris baru: count=%d", count)
	}
	var notif int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE entity_type='SCHEDULE_PATTERN'`).Scan(&notif)
	if notif != 0 {
		t.Fatalf("draf tidak boleh kirim WA: %d", notif)
	}
}

func TestDraft_DeleteHard(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status) VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT')`)
	_, _ = db.Exec(`INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type) VALUES (2, 2, 1, 'Struktur Data Draf', 'TEORI')`)
	_, _ = db.Exec(`INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from) VALUES (10, 2, 1, 1, '08:00', '09:40', '2025-09-01')`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedule/patterns/10?version=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM schedule_patterns WHERE id=10`).Scan(&count)
	if count != 0 {
		t.Fatalf("draf harus hapus baris sungguhan")
	}
}

func TestDraft_DeleteBlockedByEventRef(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status) VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT')`)
	_, _ = db.Exec(`INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type) VALUES (2, 2, 1, 'Struktur Data Draf', 'TEORI')`)
	_, _ = db.Exec(`INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from) VALUES (10, 2, 1, 1, '08:00', '09:40', '2025-09-01')`)
	_, _ = db.Exec(`INSERT INTO teaching_events (id, origin_schedule_pattern_id, origin_occurrence_date, event_kind, starts_at, ends_at, lifecycle_status, version) VALUES (10, 10, '2025-09-08', 'REPLACEMENT', '2025-09-08T10:00:00+07:00', '2025-09-08T11:40:00+07:00', 'DRAFT', 1)`)
	_, _ = db.Exec(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (10, 2, 'OWNER', 'ACCEPTED')`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/schedule/patterns/10?version=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 saat dirujuk event, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDraft_PreviewAllowsDraft(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status) VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT')`)
	_, _ = db.Exec(`INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type) VALUES (2, 2, 1, 'Struktur Data Draf', 'TEORI')`)
	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"offering_id":2,"day_of_week":1,"start_time":"08:00","duration_min":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedule/patterns/preview", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview draf expected 200, got %d: %s", w.Code, w.Body.String())
	}
}
