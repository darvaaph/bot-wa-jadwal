package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1Admin_GetUsers(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tanpa token -> 401
	req := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}

	// 2. Token KM -> 403
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	req = httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("token KM expected 403, got %d, body: %s", w.Code, w.Body.String())
	}

	// 3. Token admin -> 200 + daftar berisi admin
	req = httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("token admin expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON: %v", err)
	}
	if len(body.Data) == 0 {
		t.Fatalf("daftar pengguna kosong")
	}
	ketemuAdmin := false
	for _, u := range body.Data {
		if u["identity_key"] == "+6281111111111" {
			ketemuAdmin = true
			roles, _ := u["roles"].([]any)
			if len(roles) == 0 {
				t.Errorf("admin tanpa peran aktif")
			}
		}
		if _, ok := u["password_hash"]; ok {
			t.Errorf("respons membocorkan password_hash")
		}
	}
	if !ketemuAdmin {
		t.Errorf("admin tidak ada di daftar")
	}

	// 4. Filter status
	req = httptest.NewRequest("GET", "/api/v1/admin/users?status=ACTIVE", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("filter status expected 200, got %d", w.Code)
	}
}
