package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bot-jadwal/internal/api/common"
	v1 "bot-jadwal/internal/api/v1"
	"bot-jadwal/internal/bot"
)

func TestV1Invite_SystemAdmin(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. SA undang SA -> 201 scope GLOBAL tanpa kelas
	w := helperDo(t, s, "POST", "/api/v1/invitations", adminToken, map[string]string{
		"role": "SYSTEM_ADMIN", "invited_identity_key": "+6281000000099",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("SA invite SA expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	token, _ := created.Data["token"].(string)
	if token == "" {
		t.Fatalf("tanpa token undangan: %s", w.Body.String())
	}
	var role, scope string
	var classID any
	_ = db.QueryRow(`SELECT role, scope_type, class_id FROM role_invitations WHERE invited_identity_key = '+6281000000099';`).Scan(&role, &scope, &classID)
	if role != "SYSTEM_ADMIN" || scope != "GLOBAL" || classID != nil {
		t.Fatalf("undangan tak sesuai: %s %s %v", role, scope, classID)
	}

	// 2. KM undang SA -> 403
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w = helperDo(t, s, "POST", "/api/v1/invitations", kmToken, map[string]string{
		"role": "SYSTEM_ADMIN", "invited_identity_key": "+6281000000098",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM invite SA expected 403, got %d: %s", w.Code, w.Body.String())
	}

	// 3. SA + kelas -> 422
	w = helperDo(t, s, "POST", "/api/v1/invitations", adminToken, map[string]string{
		"role": "SYSTEM_ADMIN", "class_slug": "d4-ti-2024-a", "invited_identity_key": "+6281000000097",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("SA + kelas expected 422, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Terima undangan -> akun + penugasan GLOBAL
	w = helperDo(t, s, "POST", "/api/v1/invitations/accept", "", map[string]string{
		"token": token, "password": "password123456", "display_name": "Admin Baru",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("accept expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var aRole, aScope string
	var aClass any
	_ = db.QueryRow(`
		SELECT ra.role, ra.scope_type, ra.class_id FROM role_assignments ra
		JOIN users u ON u.id = ra.user_id WHERE u.identity_key = '+6281000000099';
	`).Scan(&aRole, &aScope, &aClass)
	if aRole != "SYSTEM_ADMIN" || aScope != "GLOBAL" || aClass != nil {
		t.Fatalf("penugasan tak sesuai: %s %s %v", aRole, aScope, aClass)
	}

	// 5. Masuk daftar undangan admin
	w = helperDo(t, s, "GET", "/api/v1/admin/invitations?role=SYSTEM_ADMIN", adminToken, nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "+6281000000099") {
		t.Fatalf("daftar expected memuat SA, got %d: %s", w.Code, w.Body.String())
	}
}

type fakeBotSender struct {
	sendErr error
}

func (f *fakeBotSender) Status() string { return "connected" }

func (f *fakeBotSender) SendText(ctx context.Context, jid string, text string) (string, error) {
	if f.sendErr != nil {
		return "", f.sendErr
	}
	if jid == "" || text == "" {
		return "", errFakeSend("input kosong")
	}
	return "mid-1", nil
}

type errFakeSend string

func (e errFakeSend) Error() string { return string(e) }

func TestV1Bot_TestMessageValidation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token 401; KM 403
	w := helperDo(t, s, "POST", "/api/v1/admin/bot/test-message", "", map[string]string{"to": "x@g.us", "text": "halo"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w = helperDo(t, s, "POST", "/api/v1/admin/bot/test-message", kmToken, map[string]string{"to": "x@g.us", "text": "halo"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM expected 403, got %d", w.Code)
	}

	// 2. Payload invalid -> 422 (tujuan, teks kosong, teks >500)
	for _, tc := range []map[string]string{
		{"to": "", "text": "halo"},
		{"to": "bukan-jid", "text": "halo"},
		{"to": "x@g.us", "text": ""},
		{"to": "x@g.us", "text": strings.Repeat("a", 501)},
	} {
		w = helperDo(t, s, "POST", "/api/v1/admin/bot/test-message", adminToken, tc)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("payload %v expected 422, got %d: %s", tc, w.Code, w.Body.String())
		}
	}

	// 3. Kanal tak terdaftar -> 404
	w = helperDo(t, s, "POST", "/api/v1/admin/bot/test-message", adminToken, map[string]string{
		"to": "999@s.whatsapp.net", "text": "halo",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("kanal asing expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Tanpa klien bot (env uji) -> 503
	w = helperDo(t, s, "POST", "/api/v1/admin/bot/test-message", adminToken, map[string]string{
		"to": "120363000000000001@g.us", "text": "halo",
	})
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("tanpa klien expected 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestV1Bot_TestMessageSend(t *testing.T) {
	db, _ := setupV1TestEnv(t)
	defer db.Close()

	newReq := func(payload map[string]string) (*httptest.ResponseRecorder, *http.Request) {
		t.Helper()
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest("POST", "/api/v1/admin/bot/test-message", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = common.WithAuthContext(req, &common.UserContext{
			UserID: 3, ActiveRole: "SYSTEM_ADMIN", ActiveAssignmentID: 4,
		})
		return httptest.NewRecorder(), req
	}

	// 1. Kirim OK -> 200 + audit
	ctrl := v1.NewAdminController(db, &fakeBotSender{}, nil, nil, func() string { return t.TempDir() })
	w, req := newReq(map[string]string{"to": "120363000000000001@g.us", "text": "uji diagnostik"})
	ctrl.TestBotMessage(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("kirim expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"message_id":"mid-1"`) {
		t.Fatalf("tanpa message_id: %s", w.Body.String())
	}
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'BOT_TEST_MESSAGE';`).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit BOT_TEST_MESSAGE expected 1, got %d", nAudit)
	}

	// 2. Gagal kirim -> 502; tidak terhubung -> 503
	ctrlFail := v1.NewAdminController(db, &fakeBotSender{sendErr: errFakeSend("kirim gagal")}, nil, nil, func() string { return t.TempDir() })
	w, req = newReq(map[string]string{"to": "120363000000000001@g.us", "text": "halo"})
	ctrlFail.TestBotMessage(w, req)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("gagal kirim expected 502, got %d: %s", w.Code, w.Body.String())
	}
	ctrlDown := v1.NewAdminController(db, &fakeBotSender{sendErr: bot.ErrNotConnected}, nil, nil, func() string { return t.TempDir() })
	w, req = newReq(map[string]string{"to": "120363000000000001@g.us", "text": "halo"})
	ctrlDown.TestBotMessage(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("tak terhubung expected 503, got %d: %s", w.Code, w.Body.String())
	}
}
