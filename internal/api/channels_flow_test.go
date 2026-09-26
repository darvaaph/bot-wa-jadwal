package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/chat"
	"bot-jadwal/internal/database"
)

func newChannelsTestServer(t *testing.T) (*Server, *chat.ChatSettingsManager, int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "channels.db"))
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
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-ch','D4 TI',2024,'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, classID); err != nil {
		t.Fatal(err)
	}
	kmHash, err := auth.HashPassword("km-a@example.test", "kata-sandi-km-a-kuat")
	if err != nil {
		t.Fatal(err)
	}
	var kmID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('km-a@example.test','KM A',?) RETURNING id`, kmHash).Scan(&kmID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, status, valid_from) VALUES (?, 'KM','CLASS',?,'ACTIVE','2026-01-01T00:00:00Z')`, kmID, classID); err != nil {
		t.Fatal(err)
	}

	mgr, err := chat.NewChatSettingsManager(db)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetChatRefresher(mgr)
	return srv, mgr, classID
}

func TestAdminChannels_LinkUnlinkFlow(t *testing.T) {
	srv, mgr, classID := newChannelsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")

	groupJID := "120363777@g.us"

	// Non-admin ditolak di semua endpoint kanal.
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/v1/admin/channels"},
		{"GET", "/api/v1/admin/chats/unlinked"},
		{"POST", "/api/v1/admin/channels"},
		{"POST", "/api/v1/admin/channels/1/revoke"},
	} {
		rr := authHTTPRequest(t, srv, tc.method, tc.path, nil, kmCookies, kmCSRF)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("%s %s oleh KM: diharapkan 403, didapat %d: %s", tc.method, tc.path, rr.Code, rr.Body.String())
		}
	}

	rr := authHTTPRequest(t, srv, "GET", "/api/v1/admin/channels", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list channels: diharapkan 200, didapat %d", rr.Code)
	}

	// Chat terlihat bot tapi belum tertaut muncul di unlinked.
	mgr.NoteSeenChat(groupJID, "", true)
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/admin/chats/unlinked", nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list unlinked: diharapkan 200, didapat %d", rr.Code)
	}
	var unlinked struct {
		Data []map[string]any `json:"data"`
	}
	decodeResponse(t, rr, &unlinked)
	if len(unlinked.Data) != 1 {
		t.Fatalf("unlinked diharapkan 1, didapat %+v", unlinked.Data)
	}

	// Validasi payload.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/channels", map[string]any{"jid": groupJID}, adminCookies, adminCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("link tanpa class: diharapkan 400, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/channels", map[string]any{"jid": groupJID, "class_id": 9999}, adminCookies, adminCSRF)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("link kelas asing: diharapkan 404, didapat %d", rr.Code)
	}

	// Tautkan: cache bot harus langsung konsisten (refresh).
	linkBody, _ := json.Marshal(map[string]any{"jid": groupJID, "class_id": classID})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/channels", json.RawMessage(linkBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("link: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var linked struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &linked)
	if got := mgr.GetClass(groupJID); got == "" {
		t.Fatalf("cache bot tidak me-refresh setelah link")
	}

	// Tertaut tidak lagi muncul di unlinked.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/admin/chats/unlinked", nil, adminCookies, "")
	decodeResponse(t, rr, &unlinked)
	if len(unlinked.Data) != 0 {
		t.Fatalf("unlinked harus kosong setelah link, didapat %+v", unlinked.Data)
	}

	// Audit LINK tercatat.
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/audit?entity_type=WHATSAPP_CHANNEL", nil, adminCookies, "")
	var audits struct {
		Data []map[string]any `json:"data"`
	}
	decodeResponse(t, rr, &audits)
	if len(audits.Data) == 0 {
		t.Fatalf("audit LINK hilang")
	}

	// Lepas: status REVOKED (riwayat), cache bersih.
	revokeBody, _ := json.Marshal(map[string]any{"reason": "grup arsip"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/admin/channels/%d/revoke", linked.Data.ID), json.RawMessage(revokeBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	if got := mgr.GetClass(groupJID); got != "" {
		t.Fatalf("cache bot tidak me-refresh setelah revoke, got: %s", got)
	}
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/admin/channels/%d/revoke", linked.Data.ID), json.RawMessage(revokeBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusConflict {
		t.Fatalf("revoke ganda: diharapkan 409, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/channels/999999/revoke", json.RawMessage(revokeBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("revoke hilang: diharapkan 404, didapat %d", rr.Code)
	}

	// Taut ulang setelah revoke (upsert menghidupkan kembali).
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/channels", json.RawMessage(linkBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("re-link: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	if got := mgr.GetClass(groupJID); got == "" {
		t.Fatalf("cache bot tidak me-refresh setelah re-link")
	}
}
