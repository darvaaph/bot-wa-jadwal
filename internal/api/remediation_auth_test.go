package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func remediationRequest(s *Server, method, path, token string, payload any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(payload)
	r := httptest.NewRequest(method, path, bytes.NewReader(b))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, r)
	return w
}

func TestRemediation_InvitationExistingIdentity(t *testing.T) {
	db, s := setupV1TestEnv(t)
	km := helperLogin(t, s, "+6281234567890", "password123")
	admin := helperLogin(t, s, "+6281111111111", "password123")
	var before string
	db.QueryRow(`SELECT password_hash || display_name || status || session_version FROM users WHERE id=3`).Scan(&before)
	w := remediationRequest(s, "POST", "/api/v1/invitations", km, map[string]any{"role": "PJ", "class_slug": "d4-ti-2024-a", "semester_id": 1, "offering_id": 1, "invited_identity_key": "+6281111111111"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var res struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	w = remediationRequest(s, "POST", "/api/v1/invitations/accept", km, map[string]any{"token": res.Data.Token, "password": "attacker-password", "display_name": "Attacker"})
	if w.Code != 401 && w.Code != 403 {
		t.Fatalf("takeover accepted: %d %s", w.Code, w.Body.String())
	}
	w = remediationRequest(s, "POST", "/api/v1/invitations/accept", km, map[string]any{"token": res.Data.Token, "password": "password123"})
	if w.Code != 401 {
		t.Fatal("different logged identity accepted", w.Code, w.Body.String())
	}
	var after string
	db.QueryRow(`SELECT password_hash || display_name || status || session_version FROM users WHERE id=3`).Scan(&after)
	if before != after {
		t.Fatal("existing account changed")
	}
	if w = remediationRequest(s, "GET", "/api/v1/auth/me", admin, nil); w.Code != 200 {
		t.Fatal("existing session changed", w.Body.String())
	}
	var sessionsBefore, sessionsAfter int
	db.QueryRow(`SELECT COUNT(*) FROM user_sessions WHERE user_id=3`).Scan(&sessionsBefore)
	w = remediationRequest(s, "POST", "/api/v1/invitations/accept", admin, map[string]any{"token": res.Data.Token, "display_name": "Ignored"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	db.QueryRow(`SELECT password_hash || display_name || status || session_version FROM users WHERE id=3`).Scan(&after)
	if before != after {
		t.Fatal("accept changed existing account")
	}
	db.QueryRow(`SELECT COUNT(*) FROM user_sessions WHERE user_id=3`).Scan(&sessionsAfter)
	if sessionsBefore != sessionsAfter {
		t.Fatal("invitation created a new existing-account session")
	}
}

func TestRemediation_InvitationScopeChain(t *testing.T) {
	db, s := setupV1TestEnv(t)
	_, err := db.Exec(`INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(2,'B','b','TI',2024,'B'); INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(2,2,'2024/2025','GANJIL','2024-09-01','2025-01-31'); INSERT INTO course_offerings(id,semester_id,course_id,display_name,activity_type) VALUES(2,2,1,'Other','TEORI');`)
	if err != nil {
		t.Fatal(err)
	}
	km := helperLogin(t, s, "+6281234567890", "password123")
	w := remediationRequest(s, "POST", "/api/v1/invitations", km, map[string]any{"role": "PJ", "class_slug": "d4-ti-2024-a", "semester_id": 2, "offering_id": 2, "invited_identity_key": "+6281000000001"})
	if w.Code < 400 {
		t.Fatal("cross-class invite accepted", w.Body.String())
	}
	hash := sha256.Sum256([]byte("cross-class-old-token"))
	if _, err := db.Exec(`INSERT INTO role_invitations(token_hash,invited_identity_key,role,scope_type,class_id,semester_id,course_offering_id,status,expires_at,invited_by_user_id) VALUES(?, '+6281000000001','PJ','COURSE_OFFERING',1,2,2,'PENDING',datetime('now','+1 day'),1)`, fmt.Sprintf("%x", hash)); err != nil {
		t.Fatal(err)
	}
	w = remediationRequest(s, "POST", "/api/v1/invitations/accept", "", map[string]any{"token": "cross-class-old-token", "password": "safe-test-password"})
	if w.Code != 403 {
		t.Fatal("old malformed invitation accepted", w.Code, w.Body.String())
	}
	_, err = db.Exec(`UPDATE role_assignments SET semester_id=2,course_offering_id=2 WHERE id=2`)
	if err != nil {
		t.Fatal(err)
	}
	pj := helperLogin(t, s, "+6281298765432", "password123")
	w = remediationRequest(s, "GET", "/api/v1/tasks", pj, nil)
	if w.Code != 403 {
		t.Fatal("invalid assignment usable", w.Code, w.Body.String())
	}
}

func TestRemediation_ConcurrentLoginCheckAndRecord(t *testing.T) {
	db, s := setupV1TestEnv(t)
	var wg sync.WaitGroup
	codes := make(chan int, 40)
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := remediationRequest(s, "POST", "/api/v1/auth/login", "", map[string]any{"identity_key": "+6281000099999", "password": "wrong"})
			codes <- w.Code
		}()
	}
	wg.Wait()
	close(codes)
	unauthorized, blocked := 0, 0
	for code := range codes {
		switch code {
		case 401:
			unauthorized++
		case 429:
			blocked++
		default:
			t.Fatal(code)
		}
	}
	if unauthorized != 5 || blocked != 35 {
		t.Fatal("check/record raced", unauthorized, blocked)
	}
	if _, err := db.Exec(`UPDATE login_attempts SET attempted_at=datetime('now','-31 minutes')`); err != nil {
		t.Fatal(err)
	}
	w := remediationRequest(s, "POST", "/api/v1/auth/login", "", map[string]any{"identity_key": "+6281000099999", "password": "wrong"})
	if w.Code != 401 {
		t.Fatal("expired block still active", w.Code)
	}
}

func TestRemediation_LoginBlockedRequestsDoNotReset(t *testing.T) {
	_, s := setupV1TestEnv(t)
	for i := 0; i < 70; i++ {
		w := remediationRequest(s, "POST", "/api/v1/auth/login", "", map[string]any{"identity_key": "+6281999999999", "password": "wrong"})
		if i >= 5 && w.Code != 429 {
			t.Fatalf("attempt %d bypassed block: %d", i, w.Code)
		}
	}
}

func TestRemediation_SwitchPreservesAbsoluteAndIsAtomic(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	created := time.Now().UTC().Add(-23 * time.Hour).Format(time.RFC3339)
	expires := time.Now().UTC().Add(time.Hour).Truncate(time.Second).Format(time.RFC3339)
	if _, err := db.Exec(`UPDATE user_sessions SET created_at=?,absolute_expires_at=?`, created, expires); err != nil {
		t.Fatal(err)
	}
	w := remediationRequest(s, "POST", "/api/v1/auth/switch-context", token, map[string]any{"role_assignment_id": 3})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var latest string
	db.QueryRow(`SELECT absolute_expires_at FROM user_sessions ORDER BY id DESC LIMIT 1`).Scan(&latest)
	a, _ := time.Parse(time.RFC3339, expires)
	b, _ := time.Parse(time.RFC3339, latest)
	if b.After(a) {
		t.Fatal("session extended", latest, expires)
	}
	var res struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if _, err := db.Exec(`CREATE TRIGGER fail_rotation BEFORE UPDATE OF token_hash ON user_sessions BEGIN SELECT RAISE(ABORT,'injected'); END; CREATE TRIGGER fail_insert BEFORE INSERT ON user_sessions BEGIN SELECT RAISE(ABORT,'injected'); END;`); err != nil {
		t.Fatal(err)
	}
	w = remediationRequest(s, "POST", "/api/v1/auth/switch-context", res.Data.Token, map[string]any{"role_assignment_id": 1})
	if w.Code < 400 {
		t.Fatal("expected storage failure")
	}
	w = remediationRequest(s, "GET", "/api/v1/auth/me", res.Data.Token, nil)
	if w.Code != 200 {
		t.Fatal("failed rotation revoked old session", w.Code, w.Body.String())
	}
}

func TestRemediation_SwitchAdminLimitsBothDirections(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`INSERT INTO role_assignments(id,user_id,role,scope_type) VALUES(9,1,'SYSTEM_ADMIN','GLOBAL')`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	created := time.Now().UTC().Add(-7 * time.Hour)
	if _, err := db.Exec(`UPDATE user_sessions SET created_at=?,absolute_expires_at=?`, created.Format(time.RFC3339), created.Add(24*time.Hour).Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	var adminExpiry time.Time
	for _, assignment := range []int{9, 1} {
		w := remediationRequest(s, "POST", "/api/v1/auth/switch-context", token, map[string]any{"role_assignment_id": assignment})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var res struct {
			Data struct {
				Token   string    `json:"token"`
				Expires time.Time `json:"expires_at"`
			} `json:"data"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		if res.Data.Expires.After(created.Add(8 * time.Hour)) {
			t.Fatal("Admin absolute policy extended", res.Data.Expires)
		}
		if assignment == 9 {
			adminExpiry = res.Data.Expires
		} else if !res.Data.Expires.Equal(adminExpiry) {
			t.Fatal("leaving Admin extended expiry")
		}
		token = res.Data.Token
	}
}
