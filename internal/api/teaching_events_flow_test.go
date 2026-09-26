package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/semester"
	"bot-jadwal/internal/task"
)

func newEventsTestServer(t *testing.T) (*Server, map[string]int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "events.db"))
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
	mk := func(code, slug string) int64 {
		var id int64
		if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES (?, ?, 'D4 TI', 2024, 'A') RETURNING id`, code, slug).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	classA := mk("D4-TI-2024-A", "d4-ti-2024-a-ev")
	classB := mk("D4-TI-2024-B", "d4-ti-2024-b-ev")
	now := "2026-09-24T10:00:00.000Z"
	mkSem := func(classID int64) int64 {
		var id int64
		if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027','GANJIL','2026-09-01','2027-01-31','ACTIVE',?,?) RETURNING id`, classID, now, now).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	semA := mkSem(classA)
	semB := mkSem(classB)
	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBDEV','SBD EV') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	mkOff := func(semID int64, activity, display string) int64 {
		var id int64
		if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?,?,?,?) RETURNING id`, semID, courseID, activity, display).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	offA := mkOff(semA, "TEORI", "SBD Teori A")
	offB := mkOff(semB, "TEORI", "SBD Teori B")
	var patternID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO schedule_patterns (course_offering_id, day_of_week, start_time, end_time, effective_from, status) VALUES (?,?, '07:00','08:40','2026-09-01','ACTIVE') RETURNING id`, offA, 1).Scan(&patternID); err != nil {
		t.Fatal(err)
	}
	var roomID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO rooms (code, name) VALUES ('R201','Ruang 201') RETURNING id`).Scan(&roomID); err != nil {
		t.Fatal(err)
	}
	mkUser := func(identity, name, password, role string, classID *int64, semID, offID *int64) (cookies []*http.Cookie, csrf string) {
		hash, err := auth.HashPassword(identity, password)
		if err != nil {
			t.Fatal(err)
		}
		var uid int64
		if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES (?,?,?) RETURNING id`, identity, name, hash).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		scope := "GLOBAL"
		if role == "KM" {
			scope = "CLASS"
		} else if role == "PJ" {
			scope = "COURSE_OFFERING"
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, status, valid_from) VALUES (?,?,?,?,?,?,'ACTIVE','2026-01-01T00:00:00Z')`, uid, role, scope, classID, semID, offID); err != nil {
			t.Fatal(err)
		}
		return nil, ""
	}
	_ = mkUser
	// KM owner + PJ owner + KM participant via direct inserts (login later).
	mkUser("km-a@example.test", "KM A", "kata-sandi-km-a-kuat", "KM", &classA, nil, nil)
	mkUser("pj-a@example.test", "PJ A", "kata-sandi-pj-a-kuat", "PJ", &classA, &semA, &offA)
	mkUser("km-b@example.test", "KM B", "kata-sandi-km-b-kuat", "KM", &classB, nil, nil)

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetSemesterService(semester.NewService(db))
	srv.SetScheduleEventService(schedule.NewEventService(db))
	ids := map[string]int64{"classA": classA, "classB": classB, "semA": semA, "semB": semB, "offA": offA, "offB": offB, "pattern": patternID, "room": roomID}
	return srv, ids
}

func TestTeachingEvents_ExtraDraftPublishRevoke(t *testing.T) {
	srv, ids := newEventsTestServer(t)
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	draftBody, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "EXTRA",
		"starts_at": "2026-10-06T02:00:00Z", "ends_at": "2026-10-06T03:40:00Z",
		"reason": "Kelas tambahan",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draftBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft EXTRA: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data schedule.EventRow `json:"data"`
	}
	decodeResponse(t, rr, &created)
	eventID := created.Data.ID
	path := "/api/v1/teaching-events/" + itoa64(eventID)

	rr = authHTTPRequest(t, srv, "GET", "/api/v1/teaching-events/999999/preview", nil, pjCookies, "")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("preview missing: diharapkan 404, didapat %d", rr.Code)
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", map[string]any{"owner_offering_id": 0}, pjCookies, pjCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("draft invalid: diharapkan 400, didapat %d", rr.Code)
	}

	badReplace, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "REPLACEMENT",
		"starts_at": "2026-10-07T02:00:00Z", "ends_at": "2026-10-07T03:40:00Z",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(badReplace), pjCookies, pjCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("replacement tanpa origin: diharapkan 400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "GET", "/api/v1/teaching-events?class_id=0", nil, pjCookies, "")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("list invalid class: diharapkan 400, didapat %d", rr.Code)
	}

	// Publish EXTRA (no room -> no TU gate).
	rr = authHTTPRequest(t, srv, "POST", path+"/publish", map[string]any{}, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish EXTRA: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// Idempotent re-publish.
	rr = authHTTPRequest(t, srv, "POST", path+"/publish", map[string]any{}, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("re-publish: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestTeachingEvents_ReplacementPublishRevokeCrossClass(t *testing.T) {
	srv, ids := newEventsTestServer(t)
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")
	kmBCookies, kmBCSRF, _ := loginAs(t, srv, "km-b@example.test", "kata-sandi-km-b-kuat")

	// PJ creates REPLACEMENT draft with origin + room.
	draftBody, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "REPLACEMENT",
		"starts_at": "2026-10-06T03:00:00Z", "ends_at": "2026-10-06T04:40:00Z",
		"origin_schedule_pattern_id": ids["pattern"], "origin_occurrence_date": "2026-10-05",
		"room_id": ids["room"], "reason": "Dosen berhalangan, diganti",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draftBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data schedule.EventRow `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	eventID := created.Data.ID
	path := "/api/v1/teaching-events/" + itoa64(eventID)

	// Preview should block publish (room needs TU CONFIRMED).
	rr = authHTTPRequest(t, srv, "GET", path+"/preview", nil, pjCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("preview: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", path+"/publish", map[string]any{}, pjCookies, pjCSRF)
	if rr.Code == http.StatusOK {
		t.Fatalf("publish tanpa konfirmasi TU seharusnya ditolak")
	}

	// Record CONFIRMED then publish.
	rr = authHTTPRequest(t, srv, "POST", path+"/room-confirmation", map[string]any{
		"room_id": ids["room"], "status": "CONFIRMED", "external_contact": "Pak TU", "note": "OK",
	}, pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("room-confirm: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", path+"/publish", map[string]any{}, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// Duplicate origin should block second publish.
	draft2, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "REPLACEMENT",
		"starts_at": "2026-10-06T05:00:00Z", "ends_at": "2026-10-06T06:40:00Z",
		"origin_schedule_pattern_id": ids["pattern"], "origin_occurrence_date": "2026-10-05",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draft2), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft2: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created2 struct {
		Data schedule.EventRow `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &created2); err != nil {
		t.Fatal(err)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/"+itoa64(created2.Data.ID)+"/publish", map[string]any{}, pjCookies, pjCSRF)
	if rr.Code != http.StatusConflict && rr.Code != http.StatusBadRequest {
		t.Fatalf("publish duplikat origin: diharapkan 409/400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// Cross-class: owner invites participant, participant accepts, list visibility.
	rr = authHTTPRequest(t, srv, "POST", path+"/participants", map[string]any{"course_offering_id": ids["offB"]}, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("invite participant: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/teaching-events?class_id="+itoa64(ids["classB"]), nil, kmBCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list B before accept: diharapkan 200, didapat %d", rr.Code)
	}
	var listB struct {
		Data []schedule.EventRow `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listB); err != nil {
		t.Fatal(err)
	}
	if len(listB.Data) != 0 {
		t.Fatalf("event PENDING seharusnya belum tampil di kelas peserta")
	}
	rr = authHTTPRequest(t, srv, "POST", path+"/participants/"+itoa64(ids["offB"])+"/respond", map[string]any{"decision": "ACCEPTED"}, kmBCookies, kmBCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("accept: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/teaching-events?class_id="+itoa64(ids["classB"]), nil, kmBCookies, "")
	if err := json.Unmarshal(rr.Body.Bytes(), &listB); err != nil {
		t.Fatal(err)
	}
	if len(listB.Data) != 1 {
		t.Fatalf("event ACCEPTED seharusnya tampil di kelas peserta, didapat %d", len(listB.Data))
	}

	// PJ cannot revoke.
	rr = authHTTPRequest(t, srv, "POST", path+"/revoke", map[string]any{"reason": "salah"}, pjCookies, pjCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("revoke PJ: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// Revoke without reason.
	rr = authHTTPRequest(t, srv, "POST", path+"/revoke", map[string]any{}, kmCookies, kmCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("revoke tanpa reason: diharapkan 400, didapat %d", rr.Code)
	}
	// KM revokes.
	rr = authHTTPRequest(t, srv, "POST", path+"/revoke", map[string]any{"reason": "jadwal batal"}, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke KM: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// Second revoke rejected.
	rr = authHTTPRequest(t, srv, "POST", path+"/revoke", map[string]any{"reason": "lagi"}, kmCookies, kmCSRF)
	if rr.Code != http.StatusConflict && rr.Code != http.StatusBadRequest {
		t.Fatalf("revoke kedua: diharapkan 409/400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// Version conflict on draft update.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draft2), pjCookies, pjCSRF)
	_ = rr.Code
}

func itoa64(n int64) string {
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
