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
	"bot-jadwal/internal/portal"
	"bot-jadwal/internal/task"
)

func newAccessTestServer(t *testing.T) (*Server, *auth.Service) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "access.db"))
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
	var classID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-acc','D4 TI',2024,'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatal(err)
	}
	now := "2026-09-24T10:00:00.000Z"
	var semID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027','GANJIL','2026-09-01','2027-01-31','ACTIVE',?,?) RETURNING id`, classID, now, now).Scan(&semID); err != nil {
		t.Fatal(err)
	}
	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBDACC','SBD Acc') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatal(err)
	}
	var offeringID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?,?,'TEORI','SBD Teori') RETURNING id`, semID, courseID).Scan(&offeringID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, classID); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetPortalService(portal.NewService(db))
	return srv, svc
}

func loginAs(t *testing.T, srv *Server, identity, password string) (cookies []*http.Cookie, csrf string, principalID int64) {
	t.Helper()
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/auth/login", map[string]any{"identity_key": identity, "password": password}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("login %s: diharapkan 200, didapat %d: %s", identity, rr.Code, rr.Body.String())
	}
	var body struct {
		Data struct {
			ActiveRoleAssignmentID int64 `json:"active_role_assignment_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	resp := rr.Result()
	for _, c := range resp.Cookies() {
		if c.Name == "bot_jadwal_session" || c.Name == "bot_jadwal_csrf" {
			cookies = append(cookies, c)
			if c.Name == "bot_jadwal_csrf" {
				csrf = c.Value
			}
		}
	}
	return cookies, csrf, body.Data.ActiveRoleAssignmentID
}

func TestAccess_PortalCodeFlow(t *testing.T) {
	srv, _ := newAccessTestServer(t)

	cookies, csrf, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	rr := authHTTPRequest(t, srv, "PATCH", "/api/v1/classes/1/settings", map[string]any{
		"portal_access_mode": "CODE", "portal_code": "kelas-rahasia-123",
	}, cookies, csrf)
	if rr.Code != http.StatusOK {
		t.Fatalf("set code: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "GET", "/api/portal/d4-ti-2024-a-acc/summary", nil, nil, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("summary tanpa kode: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/portal/d4-ti-2024-a-acc/verify-code", map[string]any{"code": "salah"}, nil, "")
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("kode salah: diharapkan 401, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/portal/d4-ti-2024-a-acc/verify-code", map[string]any{"code": "kelas-rahasia-123"}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("kode benar: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var portalCookies []*http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == "portal_token" {
			portalCookies = append(portalCookies, c)
		}
	}
	if len(portalCookies) == 0 {
		t.Fatalf("cookie portal_token tidak diset")
	}

	rr = authHTTPRequest(t, srv, "GET", "/api/portal/d4-ti-2024-a-acc/summary", nil, portalCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("summary dengan kode: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "PATCH", "/api/v1/classes/1/settings", map[string]any{"portal_code": "kode-baru-456"}, cookies, csrf)
	if rr.Code != http.StatusOK {
		t.Fatalf("rotasi kode: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/portal/d4-ti-2024-a-acc/summary", nil, portalCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("sesi lama setelah rotasi: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAccess_InvitationAndRoleLifecycle(t *testing.T) {
	srv, _ := newAccessTestServer(t)
	cookies, csrf, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/invitations", map[string]any{
		"role": "KM", "class_id": 1, "identity_key": "km1@example.test",
	}, cookies, csrf)
	if rr.Code != http.StatusCreated {
		t.Fatalf("invite KM: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var invBody struct {
		Data struct {
			ID    int64  `json:"id"`
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &invBody); err != nil {
		t.Fatal(err)
	}
	if invBody.Data.Token == "" {
		t.Fatalf("token undangan kosong")
	}

	rr = authHTTPRequest(t, srv, "GET", "/api/v1/invitations/"+invBody.Data.Token, nil, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("lookup token: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations/"+invBody.Data.Token+"/accept", map[string]any{
		"display_name": "KM Satu", "password": "kata-sandi-km-yang-kuat",
	}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("accept: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations/"+invBody.Data.Token+"/accept", map[string]any{
		"display_name": "KM Satu", "password": "kata-sandi-km-yang-kuat",
	}, nil, "")
	if rr.Code == http.StatusOK {
		t.Fatalf("reuse token: seharusnya ditolak, didapat 200")
	}

	kmCookies, kmCSRF, _ := loginAs(t, srv, "km1@example.test", "kata-sandi-km-yang-kuat")

	rr = authHTTPRequest(t, srv, "GET", "/api/v1/role-assignments?class_id=1", nil, cookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list roles: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// KM undang PJ.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations", map[string]any{
		"role": "PJ", "class_id": 1, "semester_id": 1, "course_offering_id": 1, "identity_key": "pj1@example.test",
	}, kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("invite PJ oleh KM: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// PJ tidak boleh undang PJ lain.
	var pjToken string
	{
		var tmp struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &tmp); err != nil {
			t.Fatal(err)
		}
		pjToken = tmp.Data.Token
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations/"+pjToken+"/accept", map[string]any{
		"display_name": "PJ Satu", "password": "kata-sandi-pj-yang-kuat",
	}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("accept PJ: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj1@example.test", "kata-sandi-pj-yang-kuat")
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations", map[string]any{
		"role": "PJ", "class_id": 1, "semester_id": 1, "course_offering_id": 1, "identity_key": "pj2@example.test",
	}, pjCookies, pjCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("PJ undang PJ: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAccess_RecoveryFlow(t *testing.T) {
	srv, _ := newAccessTestServer(t)

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/auth/recovery/request", map[string]any{"identity_key": "admin@example.test"}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("request recovery: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var body struct {
		Data struct {
			RecoveryToken string `json:"recovery_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Data.RecoveryToken == "" {
		t.Fatalf("recovery token kosong")
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/auth/recovery/confirm", map[string]any{
		"token": body.Data.RecoveryToken, "new_password": "kata-sandi-baru-yang-kuat",
	}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("confirm recovery: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/auth/recovery/confirm", map[string]any{
		"token": body.Data.RecoveryToken, "new_password": "kata-sandi-lain-yang-kuat",
	}, nil, "")
	if rr.Code == http.StatusOK {
		t.Fatalf("reuse recovery token: seharusnya ditolak")
	}
	if _, _, _ = loginAs(t, srv, "admin@example.test", "kata-sandi-baru-yang-kuat"); true {
		// login sukses berarti password terganti; helper sudah assert 200.
	}
}
