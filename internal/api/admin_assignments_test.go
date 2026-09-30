package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func helperSwitchContext(t *testing.T, s *Server, token string, assignmentID int64) string {
	t.Helper()
	body, _ := json.Marshal(map[string]int64{"role_assignment_id": assignmentID})
	req := httptest.NewRequest("POST", "/api/v1/auth/switch-context", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("switch-context gagal, status: %d, body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Data.Token == "" {
		t.Fatalf("gagal parsing token switch-context: %v", err)
	}
	return resp.Data.Token
}

func helperDo(t *testing.T, s *Server, method, target, token string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if payload != nil {
		body, _ := json.Marshal(payload)
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	return w
}

// seedSecondClass menambahkan kelas 2 + semester + offering + PJ user 2.
func seedSecondClass(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode)
		VALUES (2, 'Asia/Jakarta', 'LINK');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 1, 'Struktur Data (Teori)', 'TEORI');
		INSERT INTO role_assignments (id, user_id, role, scope_type, class_id, semester_id, course_offering_id, status)
		VALUES (5, 2, 'PJ', 'COURSE_OFFERING', 2, 2, 2, 'ACTIVE');
	`)
	if err != nil {
		t.Fatalf("gagal seed kelas 2: %v", err)
	}
}

// seedInvitations memasukkan undangan uji dan mengembalikan id-nya:
// KM pending kelas 1, PJ pending kelas 1, KM pending kelas 2, PJ accepted kelas 1,
// PJ pending kedaluwarsa kelas 1.
func seedInvitations(t *testing.T, db *sql.DB) (kmPending, pjPending, otherPending, accepted, expired int64) {
	t.Helper()
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339)
	rows := []struct {
		hash, key, role, scope string
		class, sem, off        any
		status, exp            string
		by                     int64
	}{
		{"h-km-1", "+6281000000001", "KM", "CLASS", 1, nil, nil, "PENDING", future, 3},
		{"h-pj-1", "+6281000000002", "PJ", "COURSE_OFFERING", 1, 1, 1, "PENDING", future, 1},
		{"h-km-2", "+6281000000003", "KM", "CLASS", 2, nil, nil, "PENDING", future, 3},
		{"h-pj-acc", "+6281000000004", "PJ", "COURSE_OFFERING", 1, 1, 1, "ACCEPTED", future, 1},
		{"h-pj-exp", "+6281000000005", "PJ", "COURSE_OFFERING", 1, 1, 1, "PENDING", past, 1},
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		acceptedAt := "NULL"
		if r.status == "ACCEPTED" {
			acceptedAt = "CURRENT_TIMESTAMP"
		}
		var id int64
		err := db.QueryRow(`
			INSERT INTO role_invitations
				(token_hash, invited_identity_key, role, scope_type, class_id, semester_id,
				 course_offering_id, status, expires_at, invited_by_user_id, accepted_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, `+acceptedAt+`)
			RETURNING id;
		`, r.hash, r.key, r.role, r.scope, r.class, r.sem, r.off, r.status, r.exp, r.by).Scan(&id)
		if err != nil {
			t.Fatalf("gagal seed undangan %s: %v", r.hash, err)
		}
		ids = append(ids, id)
	}
	return ids[0], ids[1], ids[2], ids[3], ids[4]
}

func TestV1Admin_GetAssignments(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	_ = db

	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token -> 401
	w := helperDo(t, s, "GET", "/api/v1/admin/assignments", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}

	// 2. Token PJ (user 2, satu assignment PJ) -> 403
	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	w = helperDo(t, s, "GET", "/api/v1/admin/assignments", pjToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("token PJ expected 403, got %d", w.Code)
	}

	// 3. Token admin -> 200, 4 penugasan seed, tanpa bocor token/hash
	w = helperDo(t, s, "GET", "/api/v1/admin/assignments", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("token admin expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "token_hash") || strings.Contains(w.Body.String(), "password_hash") {
		t.Fatalf("respons membocorkan rahasia")
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON: %v", err)
	}
	if len(body.Data) != 4 {
		t.Fatalf("expected 4 penugasan seed, got %d", len(body.Data))
	}
	for _, a := range body.Data {
		for _, k := range []string{"id", "user_id", "identity_key", "role", "scope_type", "status", "valid_from"} {
			if _, ok := a[k]; !ok {
				t.Fatalf("field %s hilang pada %v", k, a)
			}
		}
	}
	// Penugasan PJ wajib memuat label cakupan
	for _, a := range body.Data {
		if a["role"] == "PJ" {
			if a["class_slug"] != "d4-ti-2024-a" {
				t.Errorf("PJ tanpa class_slug benar: %v", a)
			}
			if a["course_code"] != "TI201" {
				t.Errorf("PJ tanpa course_code benar: %v", a)
			}
		}
		if a["role"] == "SYSTEM_ADMIN" {
			if _, ok := a["class_id"]; ok {
				t.Errorf("SA global tak boleh punya class_id: %v", a)
			}
		}
	}

	// 4. Filter role + status + class_slug
	w = helperDo(t, s, "GET", "/api/v1/admin/assignments?role=KM", adminToken, nil)
	var f1 struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f1)
	if len(f1.Data) != 1 {
		t.Fatalf("filter role=KM expected 1, got %d", len(f1.Data))
	}
	w = helperDo(t, s, "GET", "/api/v1/admin/assignments?class_slug=d4-ti-2024-a", adminToken, nil)
	var f2 struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f2)
	if len(f2.Data) != 3 {
		t.Fatalf("filter class_slug expected 3, got %d", len(f2.Data))
	}
}

func TestV1Admin_SuspendAssignment(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	pjToken := helperLogin(t, s, "+6281298765432", "password123")

	// 1. Tanpa alasan -> 422
	w := helperDo(t, s, "POST", "/api/v1/admin/assignments/2/suspend", adminToken, map[string]string{"reason": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("tanpa alasan expected 422, got %d", w.Code)
	}

	// 2. Suspend PJ id=2 -> 200, sesi PJ tercabut
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/2/suspend", adminToken, map[string]string{"reason": "Pelanggaran kebijakan"})
	if w.Code != http.StatusOK {
		t.Fatalf("suspend expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM role_assignments WHERE id = 2;`).Scan(&status)
	if status != "SUSPENDED" {
		t.Fatalf("status DB expected SUSPENDED, got %s", status)
	}
	w = helperDo(t, s, "GET", "/api/v1/auth/me", pjToken, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sesi PJ expected 401 setelah suspend, got %d", w.Code)
	}
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'SUSPEND_ASSIGNMENT' AND entity_id = 2;`).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit SUSPEND_ASSIGNMENT expected 1, got %d", nAudit)
	}

	// 3. Suspend ganda -> 409
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/2/suspend", adminToken, map[string]string{"reason": "Lagi"})
	if w.Code != http.StatusConflict {
		t.Fatalf("suspend ganda expected 409, got %d", w.Code)
	}

	// 4. Revoke dari SUSPENDED -> 200 REVOKED
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/2/revoke", adminToken, map[string]string{"reason": "Berhenti"})
	if w.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 5. Ubah yang sudah REVOKED -> 409
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/2/suspend", adminToken, map[string]string{"reason": "Lagi"})
	if w.Code != http.StatusConflict {
		t.Fatalf("ubah REVOKED expected 409, got %d", w.Code)
	}

	// 6. ID tak ada -> 404
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/999/suspend", adminToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("id tak ada expected 404, got %d", w.Code)
	}
}

func TestV1Admin_AssignmentSelfAndLastKMGuard(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. SA tak dapat ubah penugasan aktifnya sendiri (id=4) -> 400
	w := helperDo(t, s, "POST", "/api/v1/admin/assignments/4/suspend", adminToken, map[string]string{"reason": "Iseng"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("self-suspend expected 400, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Cabut satu-satunya KM kelas (id=1) tanpa force -> 409
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/1/revoke", adminToken, map[string]string{"reason": "Ganti"})
	if w.Code != http.StatusConflict {
		t.Fatalf("last-KM tanpa force expected 409, got %d, body: %s", w.Code, w.Body.String())
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM role_assignments WHERE id = 1;`).Scan(&status)
	if status != "ACTIVE" {
		t.Fatalf("guard gagal: status berubah jadi %s", status)
	}

	// 3. Dengan force (insiden keamanan) -> 200
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/1/revoke", adminToken, map[string]any{"reason": "Insiden keamanan", "force": true})
	if w.Code != http.StatusOK {
		t.Fatalf("last-KM force expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestV1Admin_AssignmentKMScope(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1) // konteks KM kelas 1

	// 1. KM tangguhkan PJ kelasnya (id=2) -> 200
	w := helperDo(t, s, "POST", "/api/v1/admin/assignments/2/suspend", kmToken, map[string]string{"reason": "Rotasi PJ"})
	if w.Code != http.StatusOK {
		t.Fatalf("KM suspend PJ expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. KM cabut penugasan KM (id=1) -> 403
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/1/revoke", kmToken, map[string]string{"reason": "Ambil alih"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM revoke KM expected 403, got %d, body: %s", w.Code, w.Body.String())
	}

	// 3. KM ubah PJ kelas lain (id=5) -> 404 tanpa ungkap keberadaan
	w = helperDo(t, s, "POST", "/api/v1/admin/assignments/5/suspend", kmToken, map[string]string{"reason": "Iseng"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM lintas kelas expected 404, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestV1Admin_GetInvitations(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	_, _, _, _, _ = seedInvitations(t, db)

	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token -> 401; token PJ -> 403
	w := helperDo(t, s, "GET", "/api/v1/admin/invitations", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}
	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations", pjToken, nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("token PJ expected 403, got %d", w.Code)
	}

	// 2. SA -> 200, 5 undangan, tanpa token_hash, is_expired benar
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("SA expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "token_hash") {
		t.Fatalf("respons membocorkan token_hash")
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Data) != 5 {
		t.Fatalf("expected 5 undangan, got %d", len(body.Data))
	}
	byKey := map[string]map[string]any{}
	for _, inv := range body.Data {
		byKey[inv["invited_identity_key"].(string)] = inv
	}
	if byKey["+6281000000005"]["is_expired"] != true {
		t.Errorf("undangan kedaluwarsa expected is_expired=true: %v", byKey["+6281000000005"])
	}
	if byKey["+6281000000001"]["is_expired"] != false {
		t.Errorf("undangan aktif expected is_expired=false: %v", byKey["+6281000000001"])
	}
	if byKey["+6281000000001"]["class_slug"] != "d4-ti-2024-a" {
		t.Errorf("undangan tanpa class_slug benar: %v", byKey["+6281000000001"])
	}

	// 3. Filter status=PENDING -> 4, status=EXPIRED -> 1 (derivasi)
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations?status=PENDING", adminToken, nil)
	var f struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	if len(f.Data) != 4 {
		t.Fatalf("filter PENDING expected 4, got %d", len(f.Data))
	}
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations?status=EXPIRED", adminToken, nil)
	var fx struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &fx)
	if len(fx.Data) != 1 || fx.Data[0]["invited_identity_key"] != "+6281000000005" {
		t.Fatalf("filter EXPIRED expected 1 (kedaluwarsa), got %v", fx.Data)
	}

	// 4. KM hanya lihat kelasnya (3: KM pending, PJ pending, PJ accepted, PJ expired)
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations", kmToken, nil)
	var k struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &k)
	if len(k.Data) != 4 {
		t.Fatalf("KM expected 4 undangan kelasnya, got %d", len(k.Data))
	}
	for _, inv := range k.Data {
		if inv["invited_identity_key"] == "+6281000000003" {
			t.Fatalf("KM melihat undangan kelas lain")
		}
	}

	// 5. KM filter slug kelas lain -> 404
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations?class_slug=d4-ti-2024-b", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM slug lain expected 404, got %d", w.Code)
	}
}

func TestV1Admin_RevokeInvitation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	kmPending, pjPending, otherPending, accepted, _ := seedInvitations(t, db)

	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa alasan -> 422
	w := helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(kmPending, 10)+"/revoke", adminToken, map[string]string{"reason": ""})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("tanpa alasan expected 422, got %d", w.Code)
	}

	// 2. Cabut KM pending -> 200 + audit
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(kmPending, 10)+"/revoke", adminToken, map[string]string{"reason": "Batal rekrut"})
	if w.Code != http.StatusOK {
		t.Fatalf("revoke expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM role_invitations WHERE id = ?;`, kmPending).Scan(&status)
	if status != "REVOKED" {
		t.Fatalf("status DB expected REVOKED, got %s", status)
	}
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'REVOKE_INVITATION' AND entity_id = ?;`, kmPending).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit REVOKE_INVITATION expected 1, got %d", nAudit)
	}

	// 3. Cabut ganda -> 409; cabut ACCEPTED -> 409; id tak ada -> 404
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(kmPending, 10)+"/revoke", adminToken, map[string]string{"reason": "Lagi"})
	if w.Code != http.StatusConflict {
		t.Fatalf("revoke ganda expected 409, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(accepted, 10)+"/revoke", adminToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusConflict {
		t.Fatalf("revoke ACCEPTED expected 409, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/999/revoke", adminToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("id tak ada expected 404, got %d", w.Code)
	}

	// 4. KM cabut PJ kelasnya -> 200; KM cabut KM -> 403; KM cabut kelas lain -> 404
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(pjPending, 10)+"/revoke", kmToken, map[string]string{"reason": "Ganti kandidat"})
	if w.Code != http.StatusOK {
		t.Fatalf("KM revoke PJ expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	// undangan KM pending lain untuk uji 403
	var kmPending2 int64
	future := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	_ = db.QueryRow(`
		INSERT INTO role_invitations
			(token_hash, invited_identity_key, role, scope_type, class_id, status, expires_at, invited_by_user_id)
		VALUES ('h-km-x', '+6281000000006', 'KM', 'CLASS', 1, 'PENDING', ?, 3)
		RETURNING id;
	`, future).Scan(&kmPending2)
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(kmPending2, 10)+"/revoke", kmToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM revoke KM expected 403, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/invitations/"+strconv.FormatInt(otherPending, 10)+"/revoke", kmToken, map[string]string{"reason": "x"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM revoke kelas lain expected 404, got %d", w.Code)
	}
}
