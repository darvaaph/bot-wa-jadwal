package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func be011Login(t *testing.T, s *Server) string {
	t.Helper()
	return helperLogin(t, s, "+6281111111111", "password123")
}

func checksumOf(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256Hex(b)
}

func be011MakeBackup(t *testing.T, s *Server, token string) int64 {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/backups", strings.NewReader(`{"class_slug":"d4-ti-2024-a","reason":"cadangan rutin"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("backup expected 201, got %d: %s", w.Code, w.Body.String())
	}
	return 1
}

func TestBE011_RestoreRequiresReason(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	s.SetStorageDir(t.TempDir())
	be011MakeBackup(t, s, token)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("tanpa reason expected 422, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE011_RestoreVerifyOnlyContract(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	dir := t.TempDir()
	s.SetStorageDir(dir)
	be011MakeBackup(t, s, token)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1,"reason":"verifikasi rutin"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"restore_performed":false`) {
		t.Fatalf("response wajib restore_performed:false: %s", body)
	}
	if strings.Contains(body, "backups/") || strings.Contains(body, ".db") {
		t.Fatalf("response tidak boleh membocorkan path internal: %s", body)
	}
	var status string
	_ = db.QueryRow(`SELECT status FROM backup_records WHERE id=1`).Scan(&status)
	if status != "VERIFIED" {
		t.Fatalf("status harus VERIFIED, got %s", status)
	}
	// Idempotent repeat.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1,"reason":"verifikasi rutin"}`))
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("repeat expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestBE011_RestoreChecksumMismatch(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	s.SetStorageDir(t.TempDir())
	be011MakeBackup(t, s, token)
	var artifact string
	_ = db.QueryRow(`SELECT artifact_ref FROM backup_records WHERE id=1`).Scan(&artifact)
	f, err := os.OpenFile(artifact, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("corrupt")
	f.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1,"reason":"cek rusak"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "CHECKSUM_MISMATCH") {
		t.Fatalf("expected 422 CHECKSUM_MISMATCH, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE011_RestoreArtifactHilangGenerik(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	s.SetStorageDir(t.TempDir())
	be011MakeBackup(t, s, token)
	var artifact string
	_ = db.QueryRow(`SELECT artifact_ref FROM backup_records WHERE id=1`).Scan(&artifact)
	_ = os.Remove(artifact)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1,"reason":"cek hilang"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), artifact) || strings.Contains(w.Body.String(), ".db") {
		t.Fatalf("404 tidak boleh membocorkan path: %s", w.Body.String())
	}
}

func TestBE011_RestorePathEscapeDitolak(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	s.SetStorageDir(t.TempDir())
	_, err := db.Exec(`INSERT INTO backup_records (id, class_id, artifact_ref, checksum, status, created_by_user_id, reason) VALUES (99, 1, ?, 'abc', 'READY', 3, 'uji')`, filepath.Join("..", "evil.db"))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":99,"reason":"cek escape"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code == http.StatusOK {
		t.Fatalf("path escape harus ditolak, got 200: %s", w.Body.String())
	}
}

func TestBE011_RestoreNonAdminDitolak(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	_ = db
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":1,"reason":"coba"}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("non-admin expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestBE011_RestoreNonSQLiteDitolak(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	token := be011Login(t, s)
	dir := t.TempDir()
	s.SetStorageDir(dir)
	if err := os.MkdirAll(filepath.Join(dir, "backups"), 0755); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(dir, "backups", "fake.db")
	if err := os.WriteFile(fake, []byte("bukan sqlite"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`INSERT INTO backup_records (id, class_id, artifact_ref, checksum, status, created_by_user_id, reason) VALUES (98, 1, ?, ?, 'READY', 3, 'uji')`, fake, checksumOf(t, fake))
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/restores", strings.NewReader(`{"backup_id":98,"reason":"cek format"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("non-sqlite expected 422, got %d: %s", w.Code, w.Body.String())
	}
}
