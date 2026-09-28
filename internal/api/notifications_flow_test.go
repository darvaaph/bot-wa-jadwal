package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/notify"
	"bot-jadwal/internal/task"
)

type stubSender struct {
	fail bool
}

func (s *stubSender) SendText(ctx context.Context, jid, text string) (string, error) {
	if s.fail {
		return "", errStubOffline
	}
	return "stub-1", nil
}

var errStubOffline = errOffline()

func errOffline() error { return &stubError{"WA offline"} }

type stubError struct{ msg string }

func (e *stubError) Error() string { return e.msg }

func newNotificationsTestServer(t *testing.T, failSend bool) (*Server, int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "notif-api.db"))
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
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-nt','D4 TI',2024,'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, classID); err != nil {
		t.Fatal(err)
	}
	hash, _ := auth.HashPassword("km-a@example.test", "kata-sandi-km-a-kuat")
	var kmID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('km-a@example.test','KM A',?) RETURNING id`, hash).Scan(&kmID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, status, valid_from) VALUES (?, 'KM','CLASS',?,'ACTIVE','2026-01-01T00:00:00Z')`, kmID, classID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status) VALUES (?, '120363@test@g.us','GROUP','Grup A','ACTIVE')`, classID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO notification_messages (class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at) VALUES (?, 1, 'DAILY_SUMMARY','CLASS',?, 'daily:test:1', '{"text":"pagi"}','PENDING', ?)`, classID, classID, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetNotifyService(notify.NewService(db), &stubSender{fail: failSend})
	return srv, classID
}

func TestNotifications_ListRetryProcess(t *testing.T) {
	srv, classID := newNotificationsTestServer(t, false)
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	rr := authHTTPRequest(t, srv, "GET", "/api/v1/notifications?class_id=1", nil, kmCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/notifications?class_id=999", nil, kmCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("list lintas kelas: diharapkan 403, didapat %d", rr.Code)
	}
	_ = classID

	// Process as KM forbidden; admin ok.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/notifications/process", nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("process KM: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/notifications/process", nil, adminCookies, adminCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("process admin: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var proc struct {
		Data struct {
			Sent   int `json:"sent"`
			Failed int `json:"failed"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &proc); err != nil {
		t.Fatal(err)
	}
	if proc.Data.Sent != 1 {
		t.Fatalf("sent diharapkan 1, didapat %+v", proc.Data)
	}

	// Retry SENT -> 404.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/notifications/1/retry", nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("retry SENT: diharapkan 404, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestNotifications_FailedRetry(t *testing.T) {
	srv, _ := newNotificationsTestServer(t, true)
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/admin/notifications/process", nil, adminCookies, adminCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("process fail: diharapkan 200, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/notifications?class_id=1&status=FAILED", nil, kmCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list FAILED: diharapkan 200, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/notifications/1/retry", nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("retry: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}
