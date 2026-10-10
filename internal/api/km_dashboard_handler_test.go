package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestKMDashboard_ClassScopeAndEffectiveSchedule(t *testing.T) {
	db, server := setupV1TestEnv(t)
	for _, query := range []string{
		"DELETE FROM teaching_event_offerings",
		"DELETE FROM teaching_events",
		"DELETE FROM schedule_patterns",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc)
	date := today.Format("2006-01-02")
	weekday := (int(today.Weekday())+6)%7 + 1
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")
	for _, id := range []int{11, 12} {
		if _, err := db.Exec(`INSERT INTO schedule_patterns
			(id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from)
			VALUES (?, 1, 1, ?, ?, ?, ?)`, id, weekday,
			map[int]string{11: "08:00", 12: "15:00"}[id],
			map[int]string{11: "09:40", 12: "16:40"}[id], yesterday); err != nil {
			t.Fatal(err)
		}
	}
	insertEvent := func(id int, kind string, patternID int, startHour int) {
		t.Helper()
		start := time.Date(today.Year(), today.Month(), today.Day(), startHour, 0, 0, 0, loc).UTC().Format(time.RFC3339)
		end := time.Date(today.Year(), today.Month(), today.Day(), startHour+1, 0, 0, 0, loc).UTC().Format(time.RFC3339)
		if _, err := db.Exec(`INSERT INTO teaching_events
			(id, event_kind, origin_schedule_pattern_id, origin_occurrence_date, starts_at, ends_at,
			 room_id, lifecycle_status, published_by_user_id, published_at)
			VALUES (?, ?, ?, ?, ?, ?, 1, 'PUBLISHED', 1, CURRENT_TIMESTAMP)`,
			id, kind, patternID, date, start, end); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO teaching_event_offerings
			(teaching_event_id, course_offering_id, participation_role, participation_status)
			VALUES (?, 1, 'OWNER', 'ACCEPTED')`, id); err != nil {
			t.Fatal(err)
		}
	}
	insertEvent(11, "REPLACEMENT", 11, 10)
	insertEvent(12, "SESSION_CANCELLED", 12, 15)

	if _, err := db.Exec(`INSERT INTO tasks
		(course_offering_id, created_by_user_id, title, instructions, deadline_at,
		 submission_url, publication_status, review_state, published_at)
		VALUES (1, 2, 'Perlu diperiksa', 'Instruksi', ?, 'https://example.com',
		        'PUBLISHED', 'NOT_REVIEWED', CURRENT_TIMESTAMP)`, today.Add(24*time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}

	kmToken := helperLogin(t, server, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/km/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Data kmDashboardData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	got := result.Data
	if got.Class.Slug != "d4-ti-2024-a" || got.Semester == nil || got.Semester.ID != 1 {
		t.Fatalf("konteks kelas/semester salah: %+v", got)
	}
	if got.Tasks.ActiveCount != 2 || got.Tasks.ReviewCount != 1 || got.Tasks.NearCount < 1 {
		t.Fatalf("ringkasan tugas salah: %+v", got.Tasks)
	}
	if got.Schedule.WeekCount != 1 || len(got.Schedule.Today) != 1 || got.Schedule.Today[0].Kind != "REPLACEMENT" {
		t.Fatalf("jadwal efektif salah: %+v", got.Schedule)
	}

	pjToken := helperLogin(t, server, "+6281298765432", "password123")
	pjReq := httptest.NewRequest(http.MethodGet, "/api/v1/km/dashboard", nil)
	pjReq.Header.Set("Authorization", "Bearer "+pjToken)
	pjW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(pjW, pjReq)
	if pjW.Code != http.StatusForbidden {
		t.Fatalf("PJ memperoleh dashboard KM: %d", pjW.Code)
	}
}

func TestKMDashboard_NoActiveSemester(t *testing.T) {
	db, server := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE semesters SET status = 'ARCHIVED', archived_at = CURRENT_TIMESTAMP WHERE class_id = 1`); err != nil {
		t.Fatal(err)
	}
	kmToken := helperLogin(t, server, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/km/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Data kmDashboardData `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	got := result.Data
	if got.Semester != nil || got.Tasks.ActiveCount != 0 || got.Schedule.WeekCount != 0 || got.Tasks.Priority == nil || got.Schedule.Today == nil {
		t.Fatalf("state tanpa semester salah: %+v", got)
	}
}
