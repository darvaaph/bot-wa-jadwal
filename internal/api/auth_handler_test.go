package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/task"
)

func newAuthHTTPTestServer(t *testing.T) (*Server, *sql.DB, *auth.Service, int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "auth-http.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	service, err := auth.NewService(db, auth.Config{HashKey: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	user, assignment, err := service.ProvisionInitialSystemAdmin(context.Background(), auth.ProvisionInput{
		IdentityKey: "admin@example.test",
		DisplayName: "Admin Pengujian",
		Password:    "kata-sandi-yang-kuat",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(":8080", nil, nil, nil)
	server.SetAuthService(service, false)
	if user.ID <= 0 {
		t.Fatal("user provisioning tidak valid")
	}
	return server, db, service, assignment.ID
}

func authHTTPRequest(t *testing.T, server *Server, method, path string, body any, cookies []*http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, payload)
	request.RemoteAddr = "198.51.100.20:54321"
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		request.Header.Set("X-CSRF-Token", csrf)
	}
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	recorder := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(recorder, request)
	return recorder
}

func cookieNamed(t *testing.T, response *httptest.ResponseRecorder, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s tidak ditemukan", name)
	return nil
}

func TestAuthHTTP_LoginSessionAndLogout(t *testing.T) {
	server, _, _, assignmentID := newAuthHTTPTestServer(t)
	login := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"identity_key":       "ADMIN@example.test",
		"password":           "kata-sandi-yang-kuat",
		"role_assignment_id": assignmentID,
	}, nil, "")
	if login.Code != http.StatusOK {
		t.Fatalf("login: ingin 200, dapat %d: %s", login.Code, login.Body.String())
	}
	sessionCookie := cookieNamed(t, login, authSessionCookie)
	csrfCookie := cookieNamed(t, login, authCSRFCookie)
	if !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("atribut cookie sesi tidak aman: %+v", sessionCookie)
	}
	if csrfCookie.HttpOnly || csrfCookie.Value == "" {
		t.Fatalf("cookie CSRF tidak dapat dipakai double-submit: %+v", csrfCookie)
	}

	session := authHTTPRequest(t, server, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{sessionCookie}, "")
	if session.Code != http.StatusOK {
		t.Fatalf("session: ingin 200, dapat %d: %s", session.Code, session.Body.String())
	}

	withoutCSRF := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/logout", nil, []*http.Cookie{sessionCookie, csrfCookie}, "")
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("logout tanpa CSRF: ingin 403, dapat %d", withoutCSRF.Code)
	}
	logout := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/logout", nil, []*http.Cookie{sessionCookie, csrfCookie}, csrfCookie.Value)
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout: ingin 204, dapat %d: %s", logout.Code, logout.Body.String())
	}
	invalid := authHTTPRequest(t, server, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{sessionCookie}, "")
	if invalid.Code != http.StatusUnauthorized {
		t.Fatalf("sesi setelah logout: ingin 401, dapat %d", invalid.Code)
	}
}

func TestAuthHTTP_SwitchContextRotatesToken(t *testing.T) {
	server, db, _, adminAssignmentID := newAuthHTTPTestServer(t)
	var userID, classID, kmAssignmentID int64
	if err := db.QueryRow(`SELECT user_id FROM role_assignments WHERE id = ?`, adminAssignmentID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label)
		VALUES ('AUTH-A', 'auth-a', 'TI', 2026, 'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO role_assignments (user_id, role, scope_type, class_id)
		VALUES (?, 'KM', 'CLASS', ?) RETURNING id`, userID, classID).Scan(&kmAssignmentID); err != nil {
		t.Fatal(err)
	}

	login := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"identity_key": "admin@example.test",
		"password":     "kata-sandi-yang-kuat",
	}, nil, "")
	oldSession := cookieNamed(t, login, authSessionCookie)
	oldCSRF := cookieNamed(t, login, authCSRFCookie)
	switchResponse := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/switch-context", map[string]any{
		"role_assignment_id": kmAssignmentID,
	}, []*http.Cookie{oldSession, oldCSRF}, oldCSRF.Value)
	if switchResponse.Code != http.StatusOK {
		t.Fatalf("switch context: ingin 200, dapat %d: %s", switchResponse.Code, switchResponse.Body.String())
	}
	newSession := cookieNamed(t, switchResponse, authSessionCookie)
	if newSession.Value == oldSession.Value {
		t.Fatal("switch context tidak merotasi token sesi")
	}
	oldResult := authHTTPRequest(t, server, http.MethodGet, "/api/v1/auth/session", nil, []*http.Cookie{oldSession}, "")
	if oldResult.Code != http.StatusUnauthorized {
		t.Fatalf("token lama sesudah switch: ingin 401, dapat %d", oldResult.Code)
	}
}

func TestAuthHTTP_TaskMutationUsesSessionActorAndScope(t *testing.T) {
	server, db, _, adminAssignmentID := newAuthHTTPTestServer(t)
	server.SetTaskRepo(task.NewRepository(db))
	var userID int64
	if err := db.QueryRow(`SELECT user_id FROM role_assignments WHERE id = ?`, adminAssignmentID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	insertOffering := func(code string) (classID, semesterID, offeringID int64) {
		t.Helper()
		if err := db.QueryRow(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label)
			VALUES (?, ?, 'TI', 2026, ?) RETURNING id`, "AUTH-"+code, "auth-"+code, code).Scan(&classID); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on)
			VALUES (?, '2026/2027', 'ODD', '2026-08-01', '2026-12-31') RETURNING id`, classID).Scan(&semesterID); err != nil {
			t.Fatal(err)
		}
		var courseID int64
		if err := db.QueryRow(`INSERT INTO courses (code, name) VALUES (?, ?) RETURNING id`, "MK-"+code, "Mata Kuliah "+code).Scan(&courseID); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(`INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name)
			VALUES (?, ?, 'LECTURE', ?) RETURNING id`, semesterID, courseID, "Mata Kuliah "+code).Scan(&offeringID); err != nil {
			t.Fatal(err)
		}
		return
	}
	classID, semesterID, allowedOfferingID := insertOffering("PJA")
	_, _, deniedOfferingID := insertOffering("PJB")
	var pjAssignmentID int64
	if err := db.QueryRow(`INSERT INTO role_assignments (
		user_id, role, scope_type, class_id, semester_id, course_offering_id
	) VALUES (?, 'PJ', 'COURSE_OFFERING', ?, ?, ?) RETURNING id`, userID, classID, semesterID, allowedOfferingID).Scan(&pjAssignmentID); err != nil {
		t.Fatal(err)
	}

	login := authHTTPRequest(t, server, http.MethodPost, "/api/v1/auth/login", map[string]any{
		"identity_key":       "admin@example.test",
		"password":           "kata-sandi-yang-kuat",
		"role_assignment_id": pjAssignmentID,
	}, nil, "")
	sessionCookie := cookieNamed(t, login, authSessionCookie)
	csrfCookie := cookieNamed(t, login, authCSRFCookie)
	payload := map[string]any{
		"course_offering_id": allowedOfferingID,
		"title":              "Tugas Berizin",
		"instructions":       "Kerjakan",
		"deadline_at":        "2026-10-10T10:00:00Z",
		"task_type":          "INDIVIDUAL",
		"created_by_user_id": 999999,
	}
	withoutCSRF := authHTTPRequest(t, server, http.MethodPost, "/api/v1/tasks", payload, []*http.Cookie{sessionCookie, csrfCookie}, "")
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("mutasi tanpa CSRF: ingin 403, dapat %d", withoutCSRF.Code)
	}
	created := authHTTPRequest(t, server, http.MethodPost, "/api/v1/tasks", payload, []*http.Cookie{sessionCookie, csrfCookie}, csrfCookie.Value)
	if created.Code != http.StatusCreated {
		t.Fatalf("create task: ingin 201, dapat %d: %s", created.Code, created.Body.String())
	}
	var storedActor int64
	if err := db.QueryRow(`SELECT created_by_user_id FROM tasks WHERE title = 'Tugas Berizin'`).Scan(&storedActor); err != nil {
		t.Fatal(err)
	}
	if storedActor != userID {
		t.Fatalf("aktor berasal dari body: dapat %d, ingin user sesi %d", storedActor, userID)
	}

	payload["course_offering_id"] = deniedOfferingID
	denied := authHTTPRequest(t, server, http.MethodPost, "/api/v1/tasks", payload, []*http.Cookie{sessionCookie, csrfCookie}, csrfCookie.Value)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("PJ lintas offering: ingin 403, dapat %d: %s", denied.Code, denied.Body.String())
	}
}
