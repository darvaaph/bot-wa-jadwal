package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestBE008_ReviewStaleVersion(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/reviews",
		strings.NewReader(`{"decision":"APPROVED","task_version":999}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("review stale expected 409, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE008_ReviewNoteRequired(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	for _, body := range []string{
		`{"decision":"CHANGES_REQUESTED","task_version":1}`,
		`{"decision":"REVOKED","note":"   ","task_version":1}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/reviews", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("note kosong expected 422, got %d: %s", w.Code, w.Body.String())
		}
	}
}

func TestBE008_ConcurrentReviewSingleWinner(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/reviews",
				strings.NewReader(`{"decision":"APPROVED","task_version":1}`))
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			codes[idx] = w.Code
		}(i)
	}
	wg.Wait()
	wins := 0
	for _, c := range codes {
		if c == http.StatusOK {
			wins++
		}
	}
	if wins != 1 {
		t.Fatalf("tepat satu review menang diharapkan, got %v", codes)
	}
}

func TestBE008_StateChangeConflictHasData(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/complete",
		strings.NewReader(`{"version":999}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "current_data") {
		t.Fatalf("conflict wajib memuat current_data: %s", w.Body.String())
	}
}

func TestReviewTask_BroadcastWhatsAppFlag(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")

	// Review dengan broadcast_whatsapp = false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/reviews",
		strings.NewReader(`{"decision":"APPROVED","task_version":1,"broadcast_whatsapp":false}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE entity_type = 'TASK' AND entity_id = 1;`).Scan(&count)
	if count != 0 {
		t.Errorf("expected 0 notifications when broadcast_whatsapp=false, got %d", count)
	}

	// Insert task baru versi 1 untuk ditinjau dengan broadcast_whatsapp = true
	_, err := db.Exec(`
		INSERT INTO tasks (id, course_offering_id, title, instructions, deadline_at, submission_url, publication_status, published_at, review_state, created_by_user_id, version)
		VALUES (200, 1, 'Tugas WA Broadcast', 'Cek grup WA', '2026-10-25T23:59:59Z', 'https://example.com', 'PUBLISHED', '2026-10-20T10:00:00Z', 'NOT_REVIEWED', 1, 1);
	`)
	if err != nil {
		t.Fatalf("setup task 200 failed: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/200/reviews",
		strings.NewReader(`{"decision":"APPROVED","task_version":1,"broadcast_whatsapp":true}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE entity_type = 'TASK' AND entity_id = 200;`).Scan(&count)
	if count != 1 {
		t.Errorf("expected 1 notification when broadcast_whatsapp=true, got %d", count)
	}
}

func TestTaskLifecycle_EditAndReviewVersionProgression(t *testing.T) {
	db, s := setupV1TestEnv(t)
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	pjToken := helperLogin(t, s, "+6281298765432", "password123")

	// 1. Setup task 300 versi 1 (NOT_REVIEWED)
	_, err := db.Exec(`
		INSERT INTO tasks (id, course_offering_id, title, instructions, deadline_at, submission_url, publication_status, published_at, review_state, created_by_user_id, version)
		VALUES (300, 1, 'Tugas Siklus Versi', 'Instruksi awal', '2026-10-25T23:59:59Z', 'https://example.com', 'PUBLISHED', '2026-10-20T10:00:00Z', 'NOT_REVIEWED', 1, 1);
	`)
	if err != nil {
		t.Fatalf("setup task 300 failed: %v", err)
	}

	// 2. KM menyetujui v1
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/300/reviews",
		strings.NewReader(`{"decision":"APPROVED","task_version":1,"broadcast_whatsapp":false}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("review approve v1 failed: %d %s", w.Code, w.Body.String())
	}

	var vAfterApprove int
	_ = db.QueryRow(`SELECT version FROM tasks WHERE id = 300;`).Scan(&vAfterApprove)
	if vAfterApprove != 1 {
		t.Fatalf("review v1 tidak boleh menaikkan versi tugas, expected 1 got %d", vAfterApprove)
	}

	// 3. PJ mengubah tugas (PATCH) dari v1 -> harus naik ke v2
	patchReq := httptest.NewRequest(http.MethodPatch, "/api/v1/tasks/300",
		strings.NewReader(`{"title":"Tugas Siklus Versi Edit 1","instructions":"Instruksi revisi 1","version":1}`))
	patchReq.Header.Set("Authorization", "Bearer "+pjToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, patchReq)
	if w.Code != http.StatusOK {
		t.Fatalf("patch task v1 failed: %d %s", w.Code, w.Body.String())
	}

	var vAfterPatch1 int
	var revStateAfterPatch1 string
	_ = db.QueryRow(`SELECT version, review_state FROM tasks WHERE id = 300;`).Scan(&vAfterPatch1, &revStateAfterPatch1)
	if vAfterPatch1 != 2 {
		t.Fatalf("patch v1 wajib menaikkan versi ke 2, got %d", vAfterPatch1)
	}
	if revStateAfterPatch1 != "NOT_REVIEWED" {
		t.Fatalf("setelah diedit PJ, review_state wajib kembali ke NOT_REVIEWED, got %s", revStateAfterPatch1)
	}

	// 4. KM menyetujui v2 -> versi harus tetap 2
	req = httptest.NewRequest(http.MethodPost, "/api/v1/tasks/300/reviews",
		strings.NewReader(`{"decision":"APPROVED","task_version":2,"broadcast_whatsapp":false}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("review approve v2 failed: %d %s", w.Code, w.Body.String())
	}

	var vFinal int
	var revStateFinal string
	_ = db.QueryRow(`SELECT version, review_state FROM tasks WHERE id = 300;`).Scan(&vFinal, &revStateFinal)
	if vFinal != 2 {
		t.Fatalf("setelah diedit 1x dan diapprove, versi akhir wajib 2 (bukan 6), got %d", vFinal)
	}
	if revStateFinal != "APPROVED" {
		t.Fatalf("setelah diapprove KM, review_state wajib APPROVED, got %s", revStateFinal)
	}
}
