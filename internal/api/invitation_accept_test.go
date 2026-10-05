package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1Invitation_AcceptReturnsSession(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. KM membuat undangan PJ
	invBody, _ := json.Marshal(map[string]any{
		"role": "PJ", "class_slug": "d4-ti-2024-a",
		"semester_id": 1, "offering_id": 1,
		"invited_identity_key": "+6281000000001",
	})
	req := httptest.NewRequest("POST", "/api/v1/invitations", bytes.NewReader(invBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("buat undangan expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var inv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &inv); err != nil || inv.Data.Token == "" {
		t.Fatalf("token undangan kosong: %v %s", err, w.Body.String())
	}

	// 2. Terima undangan -> sesi langsung
	accBody, _ := json.Marshal(map[string]string{
		"token": inv.Data.Token, "password": "katasandi Aman123", "display_name": "PJ Baru",
	})
	req = httptest.NewRequest("POST", "/api/v1/invitations/accept", bytes.NewReader(accBody))
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("accept expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var acc struct {
		Data struct {
			Token       string           `json:"token"`
			Assignments []map[string]any `json:"assignments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &acc); err != nil {
		t.Fatalf("respons accept bukan JSON: %v", err)
	}
	if acc.Data.Token == "" {
		t.Fatalf("accept tidak mengembalikan token sesi")
	}
	if len(acc.Data.Assignments) == 0 {
		t.Fatalf("accept tidak mengembalikan penugasan")
	}

	// 3. Token langsung berlaku untuk /me
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+acc.Data.Token)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me dengan token aktivasi expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 4. Token dipakai ulang -> 400 (sudah digunakan)
	req = httptest.NewRequest("POST", "/api/v1/invitations/accept", bytes.NewReader(accBody))
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("accept kedua seharusnya gagal")
	}
}
