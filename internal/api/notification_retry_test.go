package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestBE010_RetryFailedSchedulesWithoutAttempt(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := helperLogin(t, s, "+6281234567890", "password123")
	_, err := db.Exec(`INSERT INTO notification_messages (id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at) VALUES (900, 1, 1, 'TASK_REMINDER', 'CLASS', 1, 'be010-900', '{}', 'FAILED', CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/900/retry", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "attempt_number") {
		t.Fatalf("response tidak boleh mengklaim attempt: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "retry_scheduled") {
		t.Fatalf("response wajib retry_scheduled: %s", w.Body.String())
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notification_attempts WHERE notification_message_id=900`).Scan(&n)
	if n != 0 {
		t.Fatalf("retry tidak boleh membuat attempt, got %d", n)
	}
	var st string
	_ = db.QueryRow(`SELECT status FROM notification_messages WHERE id=900`).Scan(&st)
	if st != "PENDING" {
		t.Fatalf("status harus PENDING, got %s", st)
	}
}

func TestBE010_RetryRejectedStatuses(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := helperLogin(t, s, "+6281234567890", "password123")
	cases := []struct {
		id     int64
		status string
		key    string
	}{
		{901, "SENT", "be010-901"}, {902, "PROCESSING", "be010-902"}, {903, "PENDING", "be010-903"}, {904, "SUPERSEDED", "be010-904"},
	}
	for _, tc := range cases {
		sent := "CURRENT_TIMESTAMP"
		if tc.status != "SENT" {
			sent = "NULL"
		}
		if _, err := db.Exec(`INSERT INTO notification_messages (id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, sent_at) VALUES (?, 1, 1, 'TASK_REMINDER', 'CLASS', 1, ?, '{}', ?, CURRENT_TIMESTAMP, `+sent+`)`, tc.id, tc.key, tc.status); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/"+itoa(tc.id)+"/retry", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("status %s expected 400, got %d: %s", tc.status, w.Code, w.Body.String())
		}
	}
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [32]byte
	p := len(b)
	for n > 0 {
		p--
		b[p] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		p--
		b[p] = '-'
	}
	return string(b[p:])
}

func TestBE010_ConcurrentRetrySingleWinner(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := helperLogin(t, s, "+6281234567890", "password123")
	_, err := db.Exec(`INSERT INTO notification_messages (id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at) VALUES (910, 1, 1, 'TASK_REMINDER', 'CLASS', 1, 'be010-910', '{}', 'FAILED', CURRENT_TIMESTAMP)`)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	codes := make([]int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/910/retry", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			codes[idx] = w.Code
		}(i)
	}
	wg.Wait()
	ok, conflict := 0, 0
	for _, c := range codes {
		if c == http.StatusOK {
			ok++
		} else if c == http.StatusConflict || c == http.StatusBadRequest {
			conflict++
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("satu sukses satu conflict diharapkan, got %v", codes)
	}
}
