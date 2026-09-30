package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"testing"
)

func seedNotifications(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		INSERT INTO notification_messages
			(id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			 idempotency_key, payload_json, status, scheduled_at, created_at)
		VALUES
		(1, 1, 1, 'DAILY_DIGEST', 'TEACHING_EVENT', 1, 'kunci-1', '{}', 'FAILED', '2024-05-01T06:00:00Z', '2024-05-01T06:00:00.000Z'),
		(2, 1, NULL, 'TASK_REMINDER', 'TASK', 1, 'kunci-2', '{}', 'PENDING', NULL, '2024-08-20T06:00:00.000Z');
		INSERT INTO notification_attempts
			(notification_message_id, attempt_number, started_at, finished_at, result, error_message)
		VALUES
		(1, 1, '2024-05-01T06:00:01Z', '2024-05-01T06:00:02Z', 'FAILED', 'timeout koneksi'),
		(1, 2, '2024-05-01T07:00:01Z', '2024-05-01T07:00:02Z', 'FAILED', 'penerima tak merespons');
	`)
	if err != nil {
		t.Fatalf("gagal seed notifikasi: %v", err)
	}
}

func getNotifList(t *testing.T, s *Server, token, qs string) (int, []map[string]any, string) {
	t.Helper()
	w := helperDo(t, s, "GET", "/api/v1/notifications"+qs, token, nil)
	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body.Data, w.Body.String()
}

func TestV1Notifications_FiltersAndDetail(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedNotifications(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Semua + field detail baru
	code, items, raw := getNotifList(t, s, adminToken, "")
	if code != http.StatusOK || len(items) != 2 {
		t.Fatalf("expected 200 + 2 item, got %d: %s", code, raw)
	}
	byID := map[float64]map[string]any{}
	for _, it := range items {
		byID[it["id"].(float64)] = it
	}
	m1 := byID[1]
	if m1["idempotency_key"] != "kunci-1" {
		t.Errorf("tanpa idempotency_key: %v", m1)
	}
	if m1["channel_jid"] != "120363000000000001@g.us" {
		t.Errorf("tanpa channel_jid penerima: %v", m1)
	}
	if m1["attempt_count"] != float64(2) {
		t.Errorf("attempt_count expected 2: %v", m1)
	}
	if m1["last_attempt_error"] != "penerima tak merespons" {
		t.Errorf("tanpa galat terakhir: %v", m1)
	}
	if m1["last_attempt_at"] == nil {
		t.Errorf("tanpa waktu percobaan terakhir: %v", m1)
	}
	if _, ok := byID[2]["channel_jid"]; ok {
		t.Errorf("tanpa kanal tak boleh ada channel_jid: %v", byID[2])
	}

	// 2. Filter class_id / event_type / status
	if _, items, _ := getNotifList(t, s, adminToken, "?class_id=1"); len(items) != 2 {
		t.Fatalf("class_id=1 expected 2, got %d", len(items))
	}
	if _, items, _ := getNotifList(t, s, adminToken, "?class_id=999"); len(items) != 0 {
		t.Fatalf("class_id asing expected 0, got %d", len(items))
	}
	if code, _, _ := getNotifList(t, s, adminToken, "?class_id=nol"); code != http.StatusUnprocessableEntity {
		t.Fatalf("class_id invalid expected 422, got %d", code)
	}
	if _, items, _ := getNotifList(t, s, adminToken, "?event_type=DAILY_DIGEST"); len(items) != 1 {
		t.Fatalf("event_type expected 1, got %d", len(items))
	}

	// 3. Filter waktu
	if _, items, _ := getNotifList(t, s, adminToken, "?since=2024-06-01"); len(items) != 1 {
		t.Fatalf("since expected 1, got %d", len(items))
	}
	if _, items, _ := getNotifList(t, s, adminToken, "?until=2024-06-01"); len(items) != 1 {
		t.Fatalf("until expected 1, got %d", len(items))
	}
	if code, _, _ := getNotifList(t, s, adminToken, "?since=kapan"); code != http.StatusUnprocessableEntity {
		t.Fatalf("since invalid expected 422, got %d", code)
	}

	// 4. KM terlingkup kelasnya; class_id asing -> 404 generik
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	if code, _, _ := getNotifList(t, s, kmToken, "?class_id=999"); code != http.StatusNotFound {
		t.Fatalf("KM class_id asing expected 404, got %d", code)
	}
	if _, items, _ := getNotifList(t, s, kmToken, "?status=FAILED"); len(items) != 1 {
		t.Fatalf("KM FAILED expected 1, got %d", len(items))
	}

	// 5. Retry KM atas pesan kelas lain -> 404 (tanpa ungkap keberadaan)
	seedSecondClass(t, db)
	_, err := db.Exec(`
		INSERT INTO notification_messages
			(id, class_id, event_type, entity_type, entity_id, idempotency_key,
			 payload_json, status, created_at)
		VALUES (3, 2, 'DAILY_DIGEST', 'TEACHING_EVENT', 1, 'kunci-3', '{}', 'FAILED',
		        '2024-05-02T06:00:00.000Z');
	`)
	if err != nil {
		t.Fatalf("gagal seed pesan kelas 2: %v", err)
	}
	w := helperDo(t, s, "POST", "/api/v1/notifications/3/retry", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM retry asing expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Tanpa token 401
	if code, _, _ := getNotifList(t, s, "", ""); code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", code)
	}
}

func TestV1Notifications_Attempts(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedNotifications(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Riwayat 2 percobaan berurutan + field lengkap
	w := helperDo(t, s, "GET", "/api/v1/notifications/1/attempts", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(body.Data))
	}
	if body.Data[0]["attempt_number"] != float64(1) || body.Data[1]["attempt_number"] != float64(2) {
		t.Fatalf("urutan salah: %v", body.Data)
	}
	if body.Data[1]["error_message"] != "penerima tak merespons" {
		t.Fatalf("tanpa galat: %v", body.Data[1])
	}
	if body.Data[0]["started_at"] == nil || body.Data[0]["result"] != "FAILED" {
		t.Fatalf("field tak lengkap: %v", body.Data[0])
	}

	// 2. Tanpa percobaan -> []
	w = helperDo(t, s, "GET", "/api/v1/notifications/2/attempts", adminToken, nil)
	var empty struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &empty)
	if w.Code != http.StatusOK || len(empty.Data) != 0 {
		t.Fatalf("expected 200 [], got %d: %s", w.Code, w.Body.String())
	}

	// 3. ID tak ada -> 404; ID invalid -> 400; tanpa token -> 401
	w = helperDo(t, s, "GET", "/api/v1/notifications/999/attempts", adminToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("tak ada expected 404, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/notifications/nol/attempts", adminToken, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid expected 400, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/notifications/1/attempts", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}

	// 4. KM kelas lain -> 404
	seedSecondClass(t, db)
	_, err := db.Exec(`
		INSERT INTO notification_messages
			(id, class_id, event_type, entity_type, entity_id, idempotency_key,
			 payload_json, status, created_at)
		VALUES (3, 2, 'DAILY_DIGEST', 'TEACHING_EVENT', 1, 'kunci-3', '{}', 'FAILED',
		        '2024-05-02T06:00:00.000Z');
	`)
	if err != nil {
		t.Fatalf("gagal seed pesan kelas 2: %v", err)
	}
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "GET", "/api/v1/notifications/3/attempts", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM asing expected 404, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "GET", "/api/v1/notifications/1/attempts", kmToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("KM sendiri expected 200, got %d", w.Code)
	}
}
