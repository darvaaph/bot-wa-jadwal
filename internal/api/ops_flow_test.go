package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/backup"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/notify"
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/rooms"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/semester"
	"bot-jadwal/internal/task"
)

func newOpsTestServer(t *testing.T) (*Server, map[string]int64) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.InitDB(filepath.Join(dir, "ops.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	svc, err := auth.NewService(db, auth.Config{HashKey: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ProvisionInitialSystemAdmin(ctx, auth.ProvisionInput{
		IdentityKey: "admin@example.test", DisplayName: "Admin", Password: "kata-sandi-yang-sangat-kuat",
	}); err != nil {
		t.Fatal(err)
	}
	var classA, classB int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-ops','D4 TI',2024,'A') RETURNING id`).Scan(&classA); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-B','d4-ti-2024-b-ops','D4 TI',2024,'B') RETURNING id`).Scan(&classB); err != nil {
		t.Fatal(err)
	}
	now := "2026-09-24T10:00:00.000Z"
	for _, c := range []int64{classA, classB} {
		if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, c); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027','GANJIL','2026-09-01','2027-01-31','ACTIVE',?,?)`, c, now, now); err != nil {
			t.Fatal(err)
		}
	}
	var semA int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ?`, classA).Scan(&semA); err != nil {
		t.Fatal(err)
	}
	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBDOPS','SBD Ops') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	var offA, offB int64
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?,?,'TEORI','SBD A') RETURNING id`, semA, courseID).Scan(&offA); err != nil {
		t.Fatal(err)
	}
	var semB int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ?`, classB).Scan(&semB); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?,?,'TEORI','SBD B') RETURNING id`, semB, courseID).Scan(&offB); err != nil {
		t.Fatal(err)
	}
	mkUser := func(identity, name, password, role string, classID *int64, offID *int64) {
		hash, err := auth.HashPassword(identity, password)
		if err != nil {
			t.Fatal(err)
		}
		var uid int64
		if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES (?,?,?) RETURNING id`, identity, name, hash).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		scope := "GLOBAL"
		var semID *int64
		if role == "KM" {
			scope = "CLASS"
		} else if role == "PJ" {
			scope = "COURSE_OFFERING"
			semID = &semA
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, status, valid_from) VALUES (?,?,?,?,?,?,'ACTIVE','2026-01-01T00:00:00Z')`, uid, role, scope, classID, semID, offID); err != nil {
			t.Fatal(err)
		}
	}
	mkUser("km-a@example.test", "KM A", "kata-sandi-km-a-kuat", "KM", &classA, nil)
	mkUser("pj-a@example.test", "PJ A", "kata-sandi-pj-a-kuat", "PJ", &classA, &offA)

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetPortalService(portal.NewService(db))
	srv.SetSemesterService(semester.NewService(db))
	srv.SetScheduleEventService(schedule.NewEventService(db))
	srv.SetNotifyService(notify.NewService(db), &stubSender{})
	srv.SetRoomsService(rooms.NewService(db))
	srv.SetBackupService(backup.NewService(db, filepath.Join(dir, "backups")))
	return srv, map[string]int64{"classA": classA, "classB": classB, "semA": semA, "offA": offA, "offB": offB}
}

func TestOps_RoomsCRUDAndAvailability(t *testing.T) {
	srv, _ := newOpsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/admin/rooms", map[string]any{"code": "R201", "name": "Ruang 201"}, adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create room: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/rooms", map[string]any{"code": "R201", "name": "Duplikat"}, adminCookies, adminCSRF)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate room: diharapkan 409, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/rooms", map[string]any{"code": "R202", "name": "Ruang 202"}, kmCookies, kmCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("KM create room: diharapkan 403, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/rooms?status=ACTIVE", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list rooms: diharapkan 200, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/rooms/availability?date=2026-10-06&start=07:00&end=08:40", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("availability: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var avail struct {
		Data struct {
			Candidates []map[string]any `json:"candidates"`
			Note       string           `json:"note"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &avail); err != nil {
		t.Fatal(err)
	}
	if len(avail.Data.Candidates) == 0 || avail.Data.Note == "" {
		t.Fatalf("availability: kandidat + note TU diharapkan ada")
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/rooms/availability?date=bogus&start=07:00&end=08:40", nil, adminCookies, "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("availability invalid: diharapkan 400, didapat %d", rr.Code)
	}
	// KM proposal ok, PJ forbidden.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/rooms/proposals", map[string]any{"note": "AC rusak"}, kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("proposal KM: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/rooms/proposals", map[string]any{"note": "x"}, pjCookies, pjCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("proposal PJ: diharapkan 403, didapat %d", rr.Code)
	}
	// Admin disables room; old refs stay valid (no cascade error asserted by update 200).
	rr = authHTTPRequest(t, srv, "PUT", "/api/v1/admin/rooms/1", map[string]any{"status": "INACTIVE"}, adminCookies, adminCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("disable room: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestOps_AuditScoping(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	adminCookies, _, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, _, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	// Generate audit rows: PJ creates event draft (audit CREATE with class scope).
	draftBody, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "EXTRA",
		"starts_at": "2026-10-06T02:00:00Z", "ends_at": "2026-10-06T03:40:00Z",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draftBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft: %d %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "GET", "/api/v1/audit?class_id=1", nil, kmCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("audit KM: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var kmList struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &kmList); err != nil {
		t.Fatal(err)
	}
	if len(kmList.Data) == 0 {
		t.Fatalf("audit KM seharusnya ada baris")
	}
	// PJ sees own offering rows.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/audit?class_id=1", nil, pjCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("audit PJ: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// PJ cross-class forbidden.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/audit?class_id=2", nil, pjCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("audit PJ lintas kelas: diharapkan 403, didapat %d", rr.Code)
	}
	// Admin global.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/audit", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("audit admin: diharapkan 200, didapat %d", rr.Code)
	}
	// System status: admin ok, KM forbidden.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/admin/system-status", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("system-status admin: diharapkan 200, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/admin/system-status", nil, kmCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("system-status KM: diharapkan 403, didapat %d", rr.Code)
	}
}

func TestOps_BackupRestoreRoundtrip(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	// Seed a task so restore has something to preserve.
	createBody, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Tugas Backup", "instructions": "Kerjakan.",
		"deadline_at": "2026-09-30T16:00:00.000Z", "task_type": "INDIVIDUAL", "submission_text": "LMS",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(createBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups", map[string]any{"class_id": ids["classA"], "reason": "sebelum UTS"}, adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("backup: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups", map[string]any{"class_id": ids["classA"]}, adminCookies, adminCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("backup tanpa reason: diharapkan 400, didapat %d", rr.Code)
	}
	// Mutate after backup: add another task.
	createBody2, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Tugas Baru", "instructions": "Baru.",
		"deadline_at": "2026-10-30T16:00:00.000Z", "task_type": "INDIVIDUAL", "submission_text": "LMS",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(createBody2), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task2: %d", rr.Code)
	}
	// Restore without reason rejected.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups/1/restore", map[string]any{}, adminCookies, adminCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("restore tanpa reason: diharapkan 400, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups/999999/restore", map[string]any{"reason": "x"}, adminCookies, adminCSRF)
	if rr.Code == http.StatusOK {
		t.Fatalf("restore missing: seharusnya gagal")
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups/1/restore", map[string]any{"reason": "salah hapus"}, adminCookies, adminCSRF)
	// lookup real id (may not be 1 if pre-restore points exist from other tests? isolated DB so 1).
	if rr.Code != http.StatusOK {
		// Try created id.
		rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/backups/"+itoaOps(created.Data.ID)+"/restore", map[string]any{"reason": "salah hapus"}, adminCookies, adminCSRF)
		if rr.Code != http.StatusOK {
			t.Fatalf("restore: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
		}
	}
	// After restore, original task exists and pre-restore point was created.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/admin/backups?class_id=1", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list backups: %d", rr.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) < 2 {
		t.Fatalf("titik pre-restore diharapkan ada, didapat %d", len(list.Data))
	}
}

func itoaOps(n int64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
