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
