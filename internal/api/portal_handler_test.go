package api

import (
	"context"
	"database/sql"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/database"
)

func newPortalTestServer(t *testing.T) (*Server, string, *sql.DB) {
	t.Helper()

	db, err := database.InitDB(filepath.Join(t.TempDir(), "portal.db"))
	if err != nil {
		t.Fatalf("gagal membuka database uji: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	var userID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('pj-portal', 'PJ Portal', 'hash') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("gagal membuat user: %v", err)
	}
	var classID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A', 'd4-ti-2024-a', 'D4 TI', 2024, 'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatalf("gagal membuat kelas: %v", err)
	}
	now := "2026-09-24T10:00:00.000Z"
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode, replacement_reminder_minutes) VALUES (?, 'Asia/Jakarta', 'LINK', 60)`, classID); err != nil {
		t.Fatalf("gagal membuat pengaturan kelas: %v", err)
	}
	var semID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31', 'ACTIVE', ?, ?) RETURNING id`, classID, now, now).Scan(&semID); err != nil {
		t.Fatalf("gagal membuat semester: %v", err)
	}
	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBD', 'Sistem Basis Data') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatalf("gagal membuat course: %v", err)
	}
	var offeringID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?, ?, 'Teori', 'Sistem Basis Data Teori') RETURNING id`, semID, courseID).Scan(&offeringID); err != nil {
		t.Fatalf("gagal membuat offering: %v", err)
	}
	var lecturerID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO lecturers (code, full_name) VALUES ('DSN', 'Dosen Contoh') RETURNING id`).Scan(&lecturerID); err != nil {
		t.Fatalf("gagal membuat dosen: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility) VALUES (?, ?, 'PRIMARY')`, offeringID, lecturerID); err != nil {
		t.Fatalf("gagal menautkan dosen: %v", err)
	}
	var roomID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO rooms (code, name) VALUES ('R201', 'Ruang 201') RETURNING id`).Scan(&roomID); err != nil {
		t.Fatalf("gagal membuat ruangan: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO schedule_patterns (course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status) VALUES (?, ?, 1, '07:00', '08:40', '2026-09-01', 'ACTIVE')`, offeringID, roomID); err != nil {
		t.Fatalf("gagal membuat pola jadwal: %v", err)
	}

	var eventID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO teaching_events (event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, published_by_user_id, published_at) VALUES ('EXTRA', '2026-09-30T08:00:00.000Z', '2026-09-30T10:00:00.000Z', ?, 'Kelas tambahan pengganti', 'PUBLISHED', ?, ?) RETURNING id`, roomID, userID, now).Scan(&eventID); err != nil {
		t.Fatalf("gagal membuat event: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (?, ?, 'OWNER', 'ACCEPTED')`, eventID, offeringID); err != nil {
		t.Fatalf("gagal menautkan offering pemilik: %v", err)
	}

	for _, task := range []struct {
		title    string
		deadline string
	}{
		{"Tugas Dekat", "2026-09-25T10:00:00.000Z"},
		{"Tugas Jauh", "2026-10-20T10:00:00.000Z"},
	} {
		if _, err := db.ExecContext(ctx, `INSERT INTO tasks (course_offering_id, title, instructions, deadline_at, task_type, submission_text, publication_status, review_state, reviewed_version, created_by_user_id, published_at, version) VALUES (?, ?, 'Kerjakan dengan benar.', ?, 'INDIVIDUAL', 'Via LMS', 'PUBLISHED', 'APPROVED', 1, ?, ?, 1)`, offeringID, task.title, task.deadline, userID, now); err != nil {
			t.Fatalf("gagal membuat tugas: %v", err)
		}
	}

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	return srv, "d4-ti-2024-a", db
}

func TestPortal_Summary(t *testing.T) {
	s, slug, _ := newPortalTestServer(t)

	rr := performRequest(t, s, "GET", "/api/portal/"+slug+"/summary", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Kelas         string           `json:"kelas"`
			JadwalHariIni []map[string]any `json:"jadwal_hari_ini"`
			TugasTerdekat []map[string]any `json:"tugas_terdekat"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &resp)
	if resp.Status != "success" || resp.Data.Kelas != "D4-TI-2024-A" {
		t.Errorf("ringkasan tidak sesuai: %+v", resp.Data)
	}
	if len(resp.Data.TugasTerdekat) != 2 {
		t.Errorf("diharapkan 2 tugas terdekat, didapat %d", len(resp.Data.TugasTerdekat))
	}
}

func TestPortal_ScheduleEffective(t *testing.T) {
	s, slug, _ := newPortalTestServer(t)

	rr := performRequest(t, s, "GET", "/api/portal/"+slug+"/schedule?date=2026-09-28", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("jadwal senin: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var monday struct {
		Status string `json:"status"`
		Data   struct {
			Jadwal []map[string]any `json:"jadwal"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &monday)
	if len(monday.Data.Jadwal) != 1 || monday.Data.Jadwal[0]["label"] != "Reguler" {
		t.Errorf("jadwal senin seharusnya 1 sesi Reguler, didapat %+v", monday.Data.Jadwal)
	}

	rr = performRequest(t, s, "GET", "/api/portal/"+slug+"/schedule?date=2026-09-30", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("jadwal rabu: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var wednesday struct {
		Status string `json:"status"`
		Data   struct {
			Jadwal []map[string]any `json:"jadwal"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &wednesday)
	if len(wednesday.Data.Jadwal) != 1 || wednesday.Data.Jadwal[0]["label"] != "Kelas Tambahan" {
		t.Errorf("jadwal rabu seharusnya 1 Kelas Tambahan, didapat %+v", wednesday.Data.Jadwal)
	}

	rr = performRequest(t, s, "GET", "/api/portal/"+slug+"/schedule?date=2026-13-99", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("tanggal invalid: diharapkan 400, didapat %d", rr.Code)
	}
}

func TestPortal_TasksGroups(t *testing.T) {
	s, slug, _ := newPortalTestServer(t)

	rr := performRequest(t, s, "GET", "/api/portal/"+slug+"/tasks", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Tugas []map[string]any `json:"tugas"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &resp)
	if len(resp.Data.Tugas) != 2 {
		t.Fatalf("diharapkan 2 tugas, didapat %d", len(resp.Data.Tugas))
	}
	for _, item := range resp.Data.Tugas {
		if _, ok := item["review_state"]; ok {
			t.Errorf("portal membocorkan review_state internal: %+v", item)
		}
	}

	rr = performRequest(t, s, "GET", "/api/portal/"+slug+"/tasks?group=bogus", nil)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("group invalid: diharapkan 400, didapat %d", rr.Code)
	}
}

func TestPortal_ChangesAndSemesters(t *testing.T) {
	s, slug, _ := newPortalTestServer(t)

	rr := performRequest(t, s, "GET", "/api/portal/"+slug+"/changes", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("changes: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var changes struct {
		Status string `json:"status"`
		Data   struct {
			Perubahan []map[string]any `json:"perubahan"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &changes)
	if len(changes.Data.Perubahan) != 1 || changes.Data.Perubahan[0]["label"] != "Kelas Tambahan" {
		t.Errorf("perubahan seharusnya 1 Kelas Tambahan, didapat %+v", changes.Data.Perubahan)
	}

	rr = performRequest(t, s, "GET", "/api/portal/"+slug+"/semesters", nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("semesters: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var sems struct {
		Status string `json:"status"`
		Data   struct {
			Semester []map[string]any `json:"semester"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &sems)
	if len(sems.Data.Semester) != 1 {
		t.Errorf("diharapkan 1 semester terbit, didapat %d", len(sems.Data.Semester))
	}
}

func TestPortal_UnknownSlug(t *testing.T) {
	s, _, _ := newPortalTestServer(t)

	rr := performRequest(t, s, "GET", "/api/portal/kelas-tidak-ada/summary", nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("slug tak dikenal: diharapkan 404, didapat %d", rr.Code)
	}
}

func TestPortal_CodeModeDenied(t *testing.T) {
	s, slug, db := newPortalTestServer(t)

	if _, err := db.ExecContext(context.Background(), `UPDATE class_settings SET portal_access_mode = 'CODE', portal_code_hash = 'hash-contoh' WHERE class_id = (SELECT id FROM classes WHERE slug = ?)`, slug); err != nil {
		t.Fatalf("gagal mengaktifkan mode CODE: %v", err)
	}

	rr := performRequest(t, s, "GET", "/api/portal/"+slug+"/summary", nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("mode CODE tanpa kode: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
}
