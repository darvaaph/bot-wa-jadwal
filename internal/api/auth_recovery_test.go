package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type fakeRecoverySender struct {
	to      string
	text    string
	err     error
	invoked int
}

func (f *fakeRecoverySender) SendText(_ context.Context, jid, text string) (string, error) {
	f.invoked++
	f.to = jid
	f.text = text
	if f.err != nil {
		return "", f.err
	}
	return "wa-message-id", nil
}

func configureRecoveryTestServer(s *Server, sender v1RecoverySenderForTest) {
	s.SetSecurityOptions(SecurityOptions{
		Env:           "test",
		AuthHashKey:   "0123456789abcdef0123456789abcdef",
		PublicBaseURL: "https://jadwal.example.test",
	})
	s.SetRecoverySender(sender)
}

// Local alias keeps the setup helper independent from the concrete bot client.
type v1RecoverySenderForTest interface {
	SendText(context.Context, string, string) (string, error)
}

func recoveryRequest(t *testing.T, s *Server, identity, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"identity_key": identity})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/recovery/request", bytes.NewReader(body))
	req.RemoteAddr = remoteAddr
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	return w
}

func recoveryTokenFromMessage(t *testing.T, message string) string {
	t.Helper()
	const marker = "token="
	idx := strings.Index(message, marker)
	if idx < 0 {
		t.Fatalf("pesan WhatsApp tidak memuat token: %q", message)
	}
	raw := strings.Fields(message[idx+len(marker):])[0]
	token, err := url.QueryUnescape(raw)
	if err != nil {
		t.Fatalf("token recovery tidak valid: %v", err)
	}
	return token
}

func TestV1AuthRecovery_RequestAndConfirm(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	sender := &fakeRecoverySender{}
	configureRecoveryTestServer(s, sender)

	w := recoveryRequest(t, s, "081234567890", "192.0.2.10:4123")
	if w.Code != http.StatusAccepted {
		t.Fatalf("request recovery expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if sender.invoked != 1 || sender.to != "6281234567890@s.whatsapp.net" {
		t.Fatalf("tujuan WhatsApp mismatch: invoked=%d to=%q", sender.invoked, sender.to)
	}
	if !strings.Contains(sender.text, "https://jadwal.example.test/login.html?mode=recovery&token=") {
		t.Fatalf("tautan recovery mismatch: %q", sender.text)
	}
	token := recoveryTokenFromMessage(t, sender.text)

	oldToken := helperLogin(t, s, "+6281234567890", "password123")
	body, _ := json.Marshal(map[string]string{"token": token, "new_password": "new-password-123"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/recovery/confirm", bytes.NewReader(body))
	req.RemoteAddr = "192.0.2.10:4123"
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("confirm recovery expected 200, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+oldToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sesi lama expected revoked/401, got %d: %s", w.Code, w.Body.String())
	}
	_ = helperLogin(t, s, "+6281234567890", "new-password-123")

	body, _ = json.Marshal(map[string]string{"token": token, "new_password": "another-password-123"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/recovery/confirm", bytes.NewReader(body))
	req.RemoteAddr = "192.0.2.10:4123"
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("token sekali pakai expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestV1AuthRecovery_GenericResponseAndDeliveryFailureInvalidatesToken(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	sender := &fakeRecoverySender{err: errors.New("bot offline")}
	configureRecoveryTestServer(s, sender)

	known := recoveryRequest(t, s, "+6281234567890", "192.0.2.11:4123")
	unknown := recoveryRequest(t, s, "+6289999999999", "192.0.2.11:4123")
	if known.Code != http.StatusAccepted || unknown.Code != http.StatusAccepted {
		t.Fatalf("respons generik expected 202, got known=%d unknown=%d", known.Code, unknown.Code)
	}
	if known.Body.String() != unknown.Body.String() {
		t.Fatalf("respons known/unknown berbeda: known=%q unknown=%q", known.Body.String(), unknown.Body.String())
	}
	var unused int
	if err := db.QueryRow(`SELECT COUNT(*) FROM recovery_tokens WHERE used_at IS NULL`).Scan(&unused); err != nil {
		t.Fatal(err)
	}
	if unused != 0 {
		t.Fatalf("token gagal kirim harus diinvalkan, unused=%d", unused)
	}
}

func TestV1AuthRecovery_RequestRateLimit(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	configureRecoveryTestServer(s, &fakeRecoverySender{})

	for i := 0; i < 5; i++ {
		w := recoveryRequest(t, s, "+6289999999999", "192.0.2.12:4123")
		if w.Code != http.StatusAccepted {
			t.Fatalf("request %d expected 202, got %d: %s", i+1, w.Code, w.Body.String())
		}
	}
	w := recoveryRequest(t, s, "+6289999999999", "192.0.2.12:4123")
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("request keenam expected 429 + Retry-After, got %d headers=%v", w.Code, w.Header())
	}
}
