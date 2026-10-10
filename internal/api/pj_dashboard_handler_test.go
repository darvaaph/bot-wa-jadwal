package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPJDashboard_AccessControl(t *testing.T) {
	_, server := setupV1TestEnv(t)

	// 1. Unauthenticated request -> 401
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/pj/dashboard", nil)
	unauthW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("Unauthenticated expected 401, got %d", unauthW.Code)
	}

	// 2. KM role access -> 403 (hanya PJ dan SYSTEM_ADMIN yang diizinkan)
	kmToken := helperLogin(t, server, "+6281234567890", "password123")
	kmReq := httptest.NewRequest(http.MethodGet, "/api/v1/pj/dashboard", nil)
	kmReq.Header.Set("Authorization", "Bearer "+kmToken)
	kmW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(kmW, kmReq)
	if kmW.Code != http.StatusForbidden {
		t.Fatalf("KM expected 403 on PJ dashboard, got %d: %s", kmW.Code, kmW.Body.String())
	}

	// 3. PJ role access -> 200 OK
	pjToken := helperLogin(t, server, "+6281298765432", "password123")
	pjReq := httptest.NewRequest(http.MethodGet, "/api/v1/pj/dashboard", nil)
	pjReq.Header.Set("Authorization", "Bearer "+pjToken)
	pjW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(pjW, pjReq)
	if pjW.Code != http.StatusOK {
		t.Fatalf("PJ expected 200, got %d: %s", pjW.Code, pjW.Body.String())
	}
}

func TestPJDashboard_DataIntegrityAndCorrection(t *testing.T) {
	db, server := setupV1TestEnv(t)

	for _, query := range []string{
		"DELETE FROM teaching_event_offerings",
		"DELETE FROM teaching_events",
		"DELETE FROM schedule_patterns",
		"DELETE FROM task_reviews",
		"DELETE FROM tasks",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatal(err)
		}
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	today := time.Now().In(loc)
	weekday := (int(today.Weekday())+6)%7 + 1
	yesterday := today.AddDate(0, 0, -1).Format("2006-01-02")

	// Pastikan course 2 dan course_offering 2 tersedia di semester 1
	if _, err := db.Exec(`INSERT OR IGNORE INTO courses (id, code, name) VALUES (2, 'TI202', 'Basis Data');
		INSERT OR IGNORE INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 1, 2, 'Basis Data (Teori)', 'TEORI');`); err != nil {
		t.Fatal(err)
	}

	// Setup 2 jadwal kuliah hari ini: 1 sesi offering 1 (PJ), 1 sesi offering 2 (Matkul lain)
	if _, err := db.Exec(`INSERT INTO schedule_patterns
		(id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from)
		VALUES (21, 1, 1, ?, '10:30', '12:10', ?)`, weekday, yesterday); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schedule_patterns
		(id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from)
		VALUES (22, 2, 1, ?, '13:00', '14:40', ?)`, weekday, yesterday); err != nil {
		t.Fatal(err)
	}

	// Setup 1 tugas terbit mendekati tenggat
	futureDeadline := today.Add(26 * time.Hour).UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO tasks
		(id, course_offering_id, created_by_user_id, title, instructions, deadline_at,
		 submission_url, publication_status, review_state, published_at)
		VALUES (101, 1, 2, 'Praktikum Query Agregasi', 'Instruksi praktikum', ?,
		        'https://example.com/sub', 'PUBLISHED', 'APPROVED', CURRENT_TIMESTAMP)`, futureDeadline); err != nil {
		t.Fatal(err)
	}

	// Setup 1 tugas yang memerlukan perbaikan (CHANGES_REQUESTED)
	if _, err := db.Exec(`INSERT INTO tasks
		(id, course_offering_id, created_by_user_id, title, instructions, deadline_at,
		 submission_url, publication_status, review_state)
		VALUES (102, 1, 2, 'Praktikum Normalisasi BCNF', 'Instruksi', ?,
		        'https://example.com/sub', 'DRAFT', 'CHANGES_REQUESTED')`, futureDeadline); err != nil {
		t.Fatal(err)
	}
	// Catat review penolakan dari KM (User 1 = Ketua Murid)
	if _, err := db.Exec(`INSERT INTO task_reviews
		(task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note)
		VALUES (102, 1, 1, 1, 'CHANGES_REQUESTED', 'Perbaiki format instruksi dan tanggal deadline')`); err != nil {
		t.Fatal(err)
	}

	// Setup 1 tugas draf baru yang menunggu review
	if _, err := db.Exec(`INSERT INTO tasks
		(id, course_offering_id, created_by_user_id, title, instructions, deadline_at,
		 submission_url, publication_status, review_state)
		VALUES (103, 1, 2, 'Tugas Mandiri 3', 'Instruksi tugas mandiri', ?,
		        'https://example.com/sub', 'DRAFT', 'NOT_REVIEWED')`, futureDeadline); err != nil {
		t.Fatal(err)
	}

	pjToken := helperLogin(t, server, "+6281298765432", "password123")
	pjReq := httptest.NewRequest(http.MethodGet, "/api/v1/pj/dashboard?offering_id=1", nil)
	pjReq.Header.Set("Authorization", "Bearer "+pjToken)
	pjW := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(pjW, pjReq)

	if pjW.Code != http.StatusOK {
		t.Fatalf("status %d: %s", pjW.Code, pjW.Body.String())
	}

	var result struct {
		Data pjDashboardData `json:"data"`
	}
	if err := json.Unmarshal(pjW.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	got := result.Data

	// Verifikasi Konteks
	if got.Class.Slug != "d4-ti-2024-a" || got.Offering.ID != 1 {
		t.Fatalf("Konteks offering/kelas salah: %+v", got)
	}

	// Verifikasi Banner Koreksi
	if !got.Correction.Needed || got.Correction.Count != 1 || got.Correction.TaskID != 102 {
		t.Fatalf("Data koreksi tidak sesuai: %+v", got.Correction)
	}
	if got.Correction.TaskTitle != "Praktikum Normalisasi BCNF" {
		t.Fatalf("Judul tugas koreksi salah: %s", got.Correction.TaskTitle)
	}
	if got.Correction.ReviewerName != "Ketua Murid" {
		t.Fatalf("Reviewer koreksi salah: %s", got.Correction.ReviewerName)
	}

	// Verifikasi Metrik
	if got.Metrics.PublishedCount != 1 || got.Metrics.NearCount != 1 {
		t.Fatalf("Metrik tugas terbit salah: %+v", got.Metrics)
	}
	if got.Metrics.PendingReviewCount != 1 {
		t.Fatalf("Metrik pending review salah: %+v", got.Metrics)
	}
	if got.Metrics.TodayTotalSessions != 2 || got.Metrics.TodayPJSessions != 1 {
		t.Fatalf("Metrik sesi hari ini salah: %+v", got.Metrics)
	}

	// Verifikasi Jadwal Hari Ini
	if len(got.Schedule.Today) != 2 {
		t.Fatalf("Jumlah sesi hari ini salah: %d", len(got.Schedule.Today))
	}
	if !got.Schedule.Today[0].IsMyCourse && !got.Schedule.Today[1].IsMyCourse {
		t.Fatalf("Harus ada 1 sesi dengan flag IsMyCourse=true: %+v", got.Schedule.Today)
	}
}
