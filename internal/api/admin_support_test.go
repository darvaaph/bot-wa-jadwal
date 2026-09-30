package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestV1Support_Lifecycle(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token -> 401
	w := helperDo(t, s, "POST", "/api/v1/admin/support/enter", "", map[string]string{
		"class_slug": "d4-ti-2024-a", "reason": "Pemulihan akses KM kelas",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}

	// 2. KM -> 403
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w = helperDo(t, s, "POST", "/api/v1/admin/support/enter", kmToken, map[string]string{
		"class_slug": "d4-ti-2024-a", "reason": "Pemulihan akses KM kelas",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM expected 403, got %d", w.Code)
	}

	// 3. Alasan pendek -> 422; slug kosong -> 422; kelas tak ada -> 404
	w = helperDo(t, s, "POST", "/api/v1/admin/support/enter", adminToken, map[string]string{
		"class_slug": "d4-ti-2024-a", "reason": "pendek",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("alasan pendek expected 422, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/support/enter", adminToken, map[string]string{
		"class_slug": "tak-ada", "reason": "Pemulihan akses KM kelas",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("kelas tak ada expected 404, got %d", w.Code)
	}

	// 4. Enter valid -> 201 + audit SUPPORT_ENTER
	w = helperDo(t, s, "POST", "/api/v1/admin/support/enter", adminToken, map[string]string{
		"class_slug": "d4-ti-2024-a", "reason": "Pemulihan akses KM kelas",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("enter expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var entered struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &entered)
	if entered.Data["status"] != "ACTIVE" || entered.Data["class_slug"] != "d4-ti-2024-a" {
		t.Fatalf("grant tak sesuai: %v", entered.Data)
	}
	if entered.Data["expires_at"] == nil || entered.Data["reason"] != "Pemulihan akses KM kelas" {
		t.Fatalf("grant tanpa expiry/alasan: %v", entered.Data)
	}
	var nEnter int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'SUPPORT_ENTER';`).Scan(&nEnter)
	if nEnter != 1 {
		t.Fatalf("audit SUPPORT_ENTER expected 1, got %d", nEnter)
	}

	// 5. Active -> grant; enter kedua menutup yang pertama
	w = helperDo(t, s, "GET", "/api/v1/admin/support/active", adminToken, nil)
	var act struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &act)
	if act.Data == nil || act.Data["reason"] != "Pemulihan akses KM kelas" {
		t.Fatalf("active tak sesuai: %v", act.Data)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/support/enter", adminToken, map[string]string{
		"class_slug": "d4-ti-2024-a", "reason": "Insiden keamanan sesi KM",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("enter kedua expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var nActive int
	_ = db.QueryRow(`SELECT COUNT(*) FROM support_grants WHERE user_id = 3 AND status = 'ACTIVE';`).Scan(&nActive)
	if nActive != 1 {
		t.Fatalf("expected tepat 1 grant aktif, got %d", nActive)
	}
	var nSupersede int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'SUPPORT_EXIT';`).Scan(&nSupersede)
	if nSupersede != 1 {
		t.Fatalf("audit supersede SUPPORT_EXIT expected 1, got %d", nSupersede)
	}

	// 6. Exit -> 200 CLOSED + audit; exit lagi -> 404
	w = helperDo(t, s, "POST", "/api/v1/admin/support/exit", adminToken, map[string]string{"reason": "Selesai"})
	if w.Code != http.StatusOK {
		t.Fatalf("exit expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var nExit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'SUPPORT_EXIT';`).Scan(&nExit)
	if nExit != 2 {
		t.Fatalf("audit SUPPORT_EXIT expected 2 (supersede + exit), got %d", nExit)
	}
	w = helperDo(t, s, "POST", "/api/v1/admin/support/exit", adminToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("exit tanpa aktif expected 404, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/admin/support/active", adminToken, nil)
	if !json.Valid(w.Body.Bytes()) {
		t.Fatalf("active bukan JSON: %s", w.Body.String())
	}
	var act2 struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &act2)
	if act2.Data != nil {
		t.Fatalf("active harus null setelah exit: %v", act2.Data)
	}
}

func TestV1Support_Expiry(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	past := time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO support_grants (user_id, class_id, reason, status, expires_at)
		VALUES (3, 1, 'Dukungan kedaluwarsa uji', 'ACTIVE', ?);
	`, past)
	if err != nil {
		t.Fatalf("gagal seed grant kedaluwarsa: %v", err)
	}
	w := helperDo(t, s, "GET", "/api/v1/admin/support/active", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("active expected 200, got %d", w.Code)
	}
	var act struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &act)
	if act.Data != nil {
		t.Fatalf("grant kedaluwarsa harus null: %v", act.Data)
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM support_grants WHERE user_id = 3;`).Scan(&status)
	if status != "EXPIRED" {
		t.Fatalf("status expected EXPIRED, got %s", status)
	}
}

func TestV1Audit_TimeActorEntityFilters(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	_, err := db.Exec(`
		INSERT INTO audit_logs (actor_user_id, actor_type, action, entity_type, entity_id, reason, correlation_id, created_at) VALUES
		(3, 'USER', 'SUSPEND_USER', 'USER', 2, 'r1', 'c1', '2024-05-01T10:00:00.000Z'),
		(1, 'USER', 'CREATE_BACKUP', 'BACKUP_RECORD', 1, 'r2', 'c2', '2024-06-15T10:00:00.000Z'),
		(3, 'USER', 'SUSPEND_USER', 'USER', 5, 'r3', 'c3', '2024-07-20T10:00:00.000Z');
	`)
	if err != nil {
		t.Fatalf("gagal seed audit: %v", err)
	}
	get := func(qs string) []map[string]any {
		t.Helper()
		w := helperDo(t, s, "GET", "/api/v1/audit"+qs, adminToken, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s expected 200, got %d: %s", qs, w.Code, w.Body.String())
		}
		var body struct {
			Data []map[string]any `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return body.Data
	}

	if got := get("?since=2024-06-01"); len(got) != 2 {
		t.Fatalf("since expected 2, got %d", len(got))
	}
	if got := get("?until=2024-06-01"); len(got) != 1 {
		t.Fatalf("until expected 1, got %d", len(got))
	}
	if got := get("?since=2024-06-01&until=2024-06-30"); len(got) != 1 {
		t.Fatalf("rentang expected 1, got %d", len(got))
	}
	if got := get("?actor=%2B6281111111111"); len(got) != 2 {
		t.Fatalf("actor identity expected 2, got %d", len(got))
	}
	if got := get("?actor=1"); len(got) != 1 {
		t.Fatalf("actor id expected 1, got %d", len(got))
	}
	if got := get("?entity_id=2"); len(got) != 1 {
		t.Fatalf("entity_id expected 1, got %d", len(got))
	}
	if got := get("?action=SUSPEND_USER&since=2024-07-01"); len(got) != 1 {
		t.Fatalf("kombinasi expected 1, got %d", len(got))
	}

	w := helperDo(t, s, "GET", "/api/v1/audit?since=bukan-waktu", adminToken, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("since invalid expected 422, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/audit?entity_id=nol", adminToken, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("entity_id invalid expected 422, got %d", w.Code)
	}
}

func TestV1Audit_KMForeignSlug404(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)

	w := helperDo(t, s, "GET", "/api/v1/audit?class_slug=d4-ti-2024-b", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM slug asing expected 404, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "GET", "/api/v1/audit?class_slug=d4-ti-2024-a", kmToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("KM slug sendiri expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestV1Audit_PJScope(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)

	_, err := db.Exec(`
		INSERT INTO audit_logs (actor_user_id, actor_type, action, entity_type, entity_id, class_id, reason, correlation_id, created_at) VALUES
		(2, 'USER', 'UPDATE_CLASS', 'CLASS', 1, 1, 'aksi sendiri', 'pj-1', '2024-05-01T10:00:00.000Z'),
		(1, 'USER', 'UPDATE_TASK', 'TASK', 1, 1, 'tugas matkul PJ', 'pj-2', '2024-05-02T10:00:00.000Z'),
		(1, 'USER', 'UPDATE_TASK', 'TASK', 999, 1, 'tugas matkul lain', 'pj-3', '2024-05-03T10:00:00.000Z'),
		(1, 'USER', 'ACTIVATE_SEMESTER', 'SEMESTER', 1, 1, 'semester', 'pj-4', '2024-05-04T10:00:00.000Z'),
		(1, 'USER', 'PUBLISH_EVENT', 'TEACHING_EVENT', 1, 1, 'event matkul PJ', 'pj-5', '2024-05-05T10:00:00.000Z');
	`)
	if err != nil {
		t.Fatalf("gagal seed audit PJ: %v", err)
	}
	pjToken := helperLogin(t, s, "+6281298765432", "password123")

	get := func(qs string) (int, []map[string]any) {
		t.Helper()
		w := helperDo(t, s, "GET", "/api/v1/audit"+qs, pjToken, nil)
		var body struct {
			Data []map[string]any `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return w.Code, body.Data
	}

	// Tindakan sendiri + entitas matkulnya (TASK 1, EVENT 1); bukan yang lain.
	code, items := get("")
	if code != http.StatusOK {
		t.Fatalf("PJ expected 200, got %d", code)
	}
	actions := map[string]bool{}
	for _, it := range items {
		actions[it["action"].(string)] = true
	}
	for _, want := range []string{"UPDATE_CLASS", "UPDATE_TASK", "PUBLISH_EVENT"} {
		if !actions[want] {
			t.Errorf("PJ harus melihat %s: %v", want, actions)
		}
	}
	if actions["ACTIVATE_SEMESTER"] {
		t.Errorf("PJ tak boleh melihat audit semester: %v", actions)
	}
	if len(items) != 3 {
		t.Fatalf("PJ expected 3 baris, got %d", len(items))
	}

	// Slug asing -> 404; filter aksi tetap dalam scope.
	if code, _ := get("?class_slug=d4-ti-2024-b"); code != http.StatusNotFound {
		t.Fatalf("PJ slug asing expected 404, got %d", code)
	}
	if code, items := get("?action=UPDATE_TASK"); code != http.StatusOK || len(items) != 1 {
		t.Fatalf("PJ filter aksi expected 1, got %d (%d)", len(items), code)
	}
}
