package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/database"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// setupV1TestEnv membuat database SQLite terisolasi, menjalankan migrasi v1,
// serta menyisipkan data kelas, pengguna, penugasan, dan tugas uji.
func setupV1TestEnv(t *testing.T) (*sql.DB, *Server) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "v1_test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}

	if err := database.MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}

	// 1. Masukkan Kelas & Settings
	_, err = db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (1, 'D4-TI-2024-A', 'd4-ti-2024-a', 'D4 Teknik Informatika', 2024, 'A', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode)
		VALUES (1, 'Asia/Jakarta', 'LINK');
	`)
	if err != nil {
		t.Fatalf("Gagal insert class: %v", err)
	}

	// 2. Masukkan Semester
	_, err = db.Exec(`
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at)
		VALUES (1, 1, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("Gagal insert semester: %v", err)
	}

	// 3. Masukkan Mata Kuliah, Offering, dan Ruangan
	_, err = db.Exec(`
		INSERT INTO courses (id, code, name) VALUES (1, 'TI201', 'Struktur Data');
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (1, 1, 1, 'Struktur Data (Teori)', 'TEORI');
		INSERT INTO rooms (id, code, name) VALUES (1, 'R-301', 'Ruang Kelas 301');
	`)
	if err != nil {
		t.Fatalf("Gagal insert courses/offerings: %v", err)
	}

	// 4. Masukkan Pola Jadwal & Teaching Event
	todayStr := time.Now().Format("2006-01-02")
	_, err = db.Exec(`
		INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from)
		VALUES (1, 1, 1, 1, '08:00', '09:40', ?);
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, version)
		VALUES (1, 'EXTRA', CURRENT_TIMESTAMP, datetime(CURRENT_TIMESTAMP, '+2 hours'), 1, 'Kuliah Tambahan', 'PUBLISHED', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (1, 1, 'OWNER', 'ACCEPTED');
	`, todayStr)
	if err != nil {
		t.Fatalf("Gagal insert schedule items: %v", err)
	}

	// 5. Masukkan Pengguna (KM, PJ, dan SYSTEM_ADMIN) dengan password bcrypt: "password123"
	pwdHash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	_, err = db.Exec(`
		INSERT INTO users (id, identity_key, display_name, password_hash, status)
		VALUES (1, '+6281234567890', 'Ketua Murid', ?, 'ACTIVE'),
		       (2, '+6281298765432', 'Penanggung Jawab', ?, 'ACTIVE'),
		       (3, '+6281111111111', 'Admin Utama', ?, 'ACTIVE');
	`, string(pwdHash), string(pwdHash), string(pwdHash))
	if err != nil {
		t.Fatalf("Gagal insert users: %v", err)
	}

	// 6. Masukkan Penugasan Peran (KM, PJ, dan SYSTEM_ADMIN)
	_, err = db.Exec(`
		INSERT INTO role_assignments (id, user_id, role, scope_type, class_id, semester_id, course_offering_id, status)
		VALUES (1, 1, 'KM', 'CLASS', 1, 1, NULL, 'ACTIVE'),
		       (2, 2, 'PJ', 'COURSE_OFFERING', 1, 1, 1, 'ACTIVE'),
		       (3, 1, 'PJ', 'COURSE_OFFERING', 1, 1, 1, 'ACTIVE'),
		       (4, 3, 'SYSTEM_ADMIN', 'GLOBAL', NULL, NULL, NULL, 'ACTIVE');
	`)
	if err != nil {
		t.Fatalf("Gagal insert role assignments: %v", err)
	}

	// 7. Masukkan Tugas Uji Awal
	futureDeadline := time.Now().Add(48 * time.Hour).UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO tasks (id, course_offering_id, created_by_user_id, title, instructions,
		                  deadline_at, task_type, publication_status, review_state, version)
		VALUES (1, 1, 2, 'Tugas Algoritma 1', 'Kerjakan latihan soal bab 1',
		        ?, 'INDIVIDUAL', 'PUBLISHED', 'APPROVED', 1);
	`, futureDeadline)
	if err != nil {
		t.Fatalf("Gagal insert initial task: %v", err)
	}

	// 8. Masukkan Materi Uji Awal
	_, err = db.Exec(`
		INSERT INTO materials (id, class_id, course_offering_id, title, material_type, url, description)
		VALUES (1, 1, 1, 'Slide Pertemuan 1', 'DOCUMENT', 'https://example.com/slide1.pdf', 'Pengenalan');
	`)
	if err != nil {
		t.Fatalf("Gagal insert initial material: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	server := NewServer(":0", nil, nil, nil, db)
	return db, server
}

// helperLogin melakukan login dan mengembalikan token auth Bearer
func helperLogin(t *testing.T, s *Server, identityKey, password string) string {
	t.Helper()
	payload := map[string]string{
		"identity_key": identityKey,
		"password":     password,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Login gagal, status: %d, body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || resp.Data.Token == "" {
		t.Fatalf("Gagal parsing token respons: %v", err)
	}

	return resp.Data.Token
}

// ============================================================================
// 1. Auth & Context Tests
// ============================================================================

func TestV1Auth_LoginSuccess(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	payload := map[string]string{
		"identity_key": "+6281234567890",
		"password":     "password123",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Token       string `json:"token"`
			TokenType   string `json:"token_type"`
			Assignments []any  `json:"assignments"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal gagal: %v", err)
	}

	if resp.Status != "success" || resp.Data.TokenType != "Bearer" {
		t.Errorf("Format respons tidak sesuai: %+v", resp)
	}
	if len(resp.Data.Token) < 10 || resp.Data.Token[:4] != "bv1_" {
		t.Errorf("Format token tidak diawali 'bv1_': %s", resp.Data.Token)
	}
}

func TestV1Auth_LoginInvalidPassword_Generic401(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	payload := map[string]string{
		"identity_key": "+6281234567890",
		"password":     "wrong_password",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Status expected 401, got %d", w.Code)
	}

	var resp V1ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != CodeUnauthenticated {
		t.Errorf("Error code expected %s, got %s", CodeUnauthenticated, resp.Error.Code)
	}
}

func TestV1Auth_LoginRateLimitLockout_429(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	payload := map[string]string{
		"identity_key": "+6281234567890",
		"password":     "wrong_password",
	}
	body, _ := json.Marshal(payload)

	// Lakukan 5 kali percobaan gagal
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Percobaan %d status expected 401, got %d", i+1, w.Code)
		}
	}

	// Percobaan ke-6 harus ditolak dengan 429
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("Status expected 429 TOO_MANY_REQUESTS setelah 5 kali gagal, got %d", w.Code)
	}
}

func TestV1Auth_RequireAuth_Rejection(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Tanpa header auth
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Tanpa token expected 401, got %d", w.Code)
	}

	// 2. Token palsu
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer bv1_invalid_token_12345")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Token palsu expected 401, got %d", w.Code)
	}
}

func TestV1Auth_GetMe_Success(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	token := helperLogin(t, s, "+6281234567890", "password123")

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /api/v1/auth/me expected 200, got %d", w.Code)
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			User struct {
				DisplayName string `json:"display_name"`
			} `json:"user"`
			ActiveAssignment struct {
				Role string `json:"role"`
			} `json:"active_assignment"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.User.DisplayName != "Ketua Murid" {
		t.Errorf("Display name mismatch: %s", resp.Data.User.DisplayName)
	}
	if resp.Data.ActiveAssignment.Role != "KM" {
		t.Errorf("Role expected KM, got %s", resp.Data.ActiveAssignment.Role)
	}
}

func TestV1Auth_SwitchContext_TokenRotation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	token := helperLogin(t, s, "+6281234567890", "password123")

	// Switch ke assignment ID 3 (peran PJ yang juga dimiliki User 1)
	payload := map[string]int64{"role_assignment_id": 3}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/v1/auth/switch-context", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Switch context expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	newToken := resp.Data.Token

	if newToken == "" || newToken == token {
		t.Errorf("Token baru harus dihasilkan dan berbeda dari token lama")
	}

	// Pastikan konteks aktif sekarang adalah PJ
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+newToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	var meResp struct {
		Data struct {
			ActiveAssignment struct {
				Role string `json:"role"`
			} `json:"active_assignment"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &meResp)
	if meResp.Data.ActiveAssignment.Role != "PJ" {
		t.Errorf("Konteks aktif setelah switch expected PJ, got %s", meResp.Data.ActiveAssignment.Role)
	}
}

// ============================================================================
// 2. Portal Read Endpoints Tests
// ============================================================================

func TestV1Portal_Endpoints(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Summary
	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Portal summary expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Schedule
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/schedule", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Portal schedule expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 3. Tasks
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/tasks", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Portal tasks expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 4. Task Detail
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/tasks/1", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Portal task detail expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 5. Materials
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/materials", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Portal materials expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 6. Kelas tidak ada -> 404
	req = httptest.NewRequest("GET", "/api/v1/portal/kelas-fiktif/summary", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Portal nonexistent class expected 404, got %d", w.Code)
	}
}

// ============================================================================
// 3. Tasks Management & Optimistic Locking Tests
// ============================================================================

func TestV1Tasks_CreateDraftAndPublish(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	pjToken := helperLogin(t, s, "+6281298765432", "password123")

	// 1. Buat Draft
	future := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	payloadDraft := map[string]any{
		"offering_id": 1,
		"title":       "Draft Tugas Baru",
		"deadline_at": future,
		"save_as":     "draft",
	}
	body, _ := json.Marshal(payloadDraft)

	req := httptest.NewRequest("POST", "/api/v1/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create draft expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var draftResp struct {
		Data struct {
			ID                int64  `json:"id"`
			PublicationStatus string `json:"publication_status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &draftResp)
	if draftResp.Data.PublicationStatus != "DRAFT" {
		t.Errorf("Publication status expected DRAFT, got %s", draftResp.Data.PublicationStatus)
	}
}

func TestV1Tasks_OptimisticLockingConflict_409(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Coba update task 1 dengan version basi (misal version = 99, padahal asli version = 1)
	newTitle := "Judul Versi Konflik"
	patchPayload := map[string]any{
		"version": 99,
		"title":   newTitle,
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/api/v1/tasks/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Optimistic locking conflict expected 409, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp V1ErrorResponse
	_ = json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Code != CodeVersionConflict {
		t.Errorf("Error code expected %s, got %s", CodeVersionConflict, errResp.Error.Code)
	}
}

func TestV1Tasks_PatchSuccess_IncrementVersion(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Update task 1 dengan version yang benar (version = 1)
	newTitle := "Tugas Algoritma Revisi KM"
	patchPayload := map[string]any{
		"version": 1,
		"title":   newTitle,
	}
	body, _ := json.Marshal(patchPayload)

	req := httptest.NewRequest("PATCH", "/api/v1/tasks/1", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH task expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			ID      int64  `json:"id"`
			Title   string `json:"title"`
			Version int    `json:"version"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.Version != 2 {
		t.Errorf("Versi setelah update expected 2, got %d", resp.Data.Version)
	}
	if resp.Data.Title != newTitle {
		t.Errorf("Title expected %s, got %s", newTitle, resp.Data.Title)
	}
}

func TestV1Tasks_KMReview_ChangesRequested(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	reviewPayload := map[string]any{
		"decision":     "CHANGES_REQUESTED",
		"note":         "Instruksi tugas kurang jelas, tolong perbaiki bagian soal 3.",
		"task_version": 1,
	}
	body, _ := json.Marshal(reviewPayload)

	req := httptest.NewRequest("POST", "/api/v1/tasks/1/reviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST review expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			PublicationStatus string `json:"publication_status"`
			ReviewState       string `json:"review_state"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.PublicationStatus != "DRAFT" {
		t.Errorf("Status publikasi setelah CHANGES_REQUESTED expected DRAFT, got %s", resp.Data.PublicationStatus)
	}
	if resp.Data.ReviewState != "CHANGES_REQUESTED" {
		t.Errorf("Review state expected CHANGES_REQUESTED, got %s", resp.Data.ReviewState)
	}
}

func TestV1Tasks_CompleteArchiveRestore(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. Complete
	req := httptest.NewRequest("POST", "/api/v1/tasks/1/complete", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Complete expected 200, got %d", w.Code)
	}

	// 2. Archive
	req = httptest.NewRequest("POST", "/api/v1/tasks/1/archive", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Archive expected 200, got %d", w.Code)
	}

	// 3. Restore
	req = httptest.NewRequest("POST", "/api/v1/tasks/1/restore", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Restore expected 200, got %d", w.Code)
	}
}

// ============================================================================
// 4. Fitur v1.1+ Tests (Status Kelas, Ruangan, Notifikasi, Audit, Backup, Admin, Impor)
// ============================================================================

func TestV1Classes_PatchStatus(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. Ubah status menjadi INACTIVE
	body, _ := json.Marshal(map[string]string{"status": "INACTIVE"})
	req := httptest.NewRequest("PATCH", "/api/v1/classes/d4-ti-2024-a", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH class status expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var statusInDB string
	_ = db.QueryRow(`SELECT status FROM classes WHERE slug = 'd4-ti-2024-a';`).Scan(&statusInDB)
	if statusInDB != "INACTIVE" {
		t.Errorf("DB class status expected INACTIVE, got %s", statusInDB)
	}

	// 2. Ubah status menjadi ARCHIVED
	body, _ = json.Marshal(map[string]string{"status": "ARCHIVED"})
	req = httptest.NewRequest("PATCH", "/api/v1/classes/d4-ti-2024-a", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH class status to ARCHIVED expected 200, got %d", w.Code)
	}

	// 3. Status tidak valid
	body, _ = json.Marshal(map[string]string{"status": "INVALID_STATUS"})
	req = httptest.NewRequest("PATCH", "/api/v1/classes/d4-ti-2024-a", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("Invalid status expected 422, got %d", w.Code)
	}
}

func TestV1Rooms_CandidatesAndConfirmation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. Cari kandidat ruangan
	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/rooms/candidates?starts_at=%s&ends_at=%s", startsAt, endsAt), nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET room candidates expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Konfirmasi ruangan TU
	body, _ := json.Marshal(map[string]any{
		"room_id":             1,
		"confirmation_status": "CONFIRMED",
		"note":                "Disetujui staf TU",
	})
	req = httptest.NewRequest("POST", "/api/v1/teaching-events/1/room-confirmations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST room confirmation expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestV1Notifications_ListAndRetry(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Masukkan pesan notifikasi berstatus FAILED
	_, err := db.Exec(`
		INSERT INTO notification_messages (id, class_id, event_type, idempotency_key, status)
		VALUES (10, 1, 'TASK_PUBLISHED', 'test-failed-notif', 'FAILED');
	`)
	if err != nil {
		t.Fatalf("Gagal insert notifikasi uji: %v", err)
	}

	// 1. Ambil daftar notifikasi
	req := httptest.NewRequest("GET", "/api/v1/notifications?status=FAILED", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET notifications expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Coba ulang (retry) pengiriman
	req = httptest.NewRequest("POST", "/api/v1/notifications/10/retry", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST notification retry expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var statusInDB string
	_ = db.QueryRow(`SELECT status FROM notification_messages WHERE id = 10;`).Scan(&statusInDB)
	if statusInDB != "PENDING" {
		t.Errorf("Notif status expected PENDING after retry, got %s", statusInDB)
	}
}

func TestV1Audit_ListWithScoping(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// Sisipkan rekam audit uji
	_, _ = db.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, action, entity_type, entity_id)
		VALUES (1, 1, 'TEST_ACTION', 'CLASS', 1);
	`)

	// 1. KM membaca audit kelas miliknya
	req := httptest.NewRequest("GET", "/api/v1/audit", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("KM GET audit expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Admin membaca audit global
	req = httptest.NewRequest("GET", "/api/v1/audit", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Admin GET audit expected 200, got %d", w.Code)
	}
}

func TestV1Backups_CreateAndVerifyRestore(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Buat backup (KM)
	body, _ := json.Marshal(map[string]string{"reason": "Uji cadangan berkala"})
	req := httptest.NewRequest("POST", "/api/v1/backups", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST backups expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	var backupResp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &backupResp)
	backupID := backupResp.Data.ID

	// 2. Verifikasi restore (Admin)
	restoreBody, _ := json.Marshal(map[string]any{"backup_id": backupID})
	req = httptest.NewRequest("POST", "/api/v1/restores", bytes.NewReader(restoreBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST restores expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
}

func TestV1Admin_StatusAndUserSuspendRecover(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Ambil status telemetri admin
	req := httptest.NewRequest("GET", "/api/v1/admin/status", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET admin status expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Suspend pengguna (PJ, ID 2)
	suspendBody, _ := json.Marshal(map[string]string{"reason": "Akun dinonaktifkan sementara"})
	req = httptest.NewRequest("POST", "/api/v1/admin/users/2/suspend", bytes.NewReader(suspendBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST suspend user expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Cek status di DB
	var userStatus string
	_ = db.QueryRow(`SELECT status FROM users WHERE id = 2;`).Scan(&userStatus)
	if userStatus != "SUSPENDED" {
		t.Errorf("User status expected SUSPENDED, got %s", userStatus)
	}

	// 3. Recover pengguna dengan kata sandi baru
	recoverBody, _ := json.Marshal(map[string]string{
		"new_password": "passwordBaru123",
		"reason":       "Akun telah diverifikasi kembali",
	})
	req = httptest.NewRequest("POST", "/api/v1/admin/users/2/recover", bytes.NewReader(recoverBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST recover user expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Verifikasi login dengan password baru
	pjToken := helperLogin(t, s, "+6281298765432", "passwordBaru123")
	if pjToken == "" {
		t.Errorf("Login PJ setelah pemulihan gagal")
	}
}

func TestV1Curriculum_ImportValidateAndApply(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	curriculumPayload := CurriculumImportPayload{
		SourceType: "JSON",
		Courses: []CourseImportItem{
			{Code: "IF301", Name: "Kecerdasan Buatan"},
		},
		Lecturers: []LecturerImportItem{
			{Code: "DSN002", FullName: "Prof. Agus M.Kom"},
		},
		Offerings: []OfferingImportItem{
			{
				CourseCode:    "IF301",
				ActivityType:  "TEORI",
				DisplayName:   "Kecerdasan Buatan (Teori)",
				LecturerCodes: []string{"DSN002"},
			},
		},
		SchedulePatterns: []SchedulePatternImportItem{
			{
				CourseCode:   "IF301",
				ActivityType: "TEORI",
				DayOfWeek:    2,
				StartTime:    "10:00",
				DurationMin:  100,
			},
		},
	}

	body, _ := json.Marshal(curriculumPayload)

	// 1. Validasi Impor
	req := httptest.NewRequest("POST", "/api/v1/semesters/1/import-validate", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST import-validate expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var valResp struct {
		Data struct {
			BatchID int64  `json:"batch_id"`
			Status  string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &valResp)
	if valResp.Data.Status != "READY" {
		t.Fatalf("Batch status expected READY, got %s", valResp.Data.Status)
	}

	batchID := valResp.Data.BatchID

	// 2. Terapkan Impor
	applyBody, _ := json.Marshal(map[string]any{"batch_id": batchID})
	req = httptest.NewRequest("POST", "/api/v1/semesters/1/import-apply", bytes.NewReader(applyBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST import-apply expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 3. Verifikasi mata kuliah baru tersimpan di DB
	var courseName string
	err := db.QueryRow(`SELECT name FROM courses WHERE code = 'IF301';`).Scan(&courseName)
	if err != nil || courseName != "Kecerdasan Buatan" {
		t.Errorf("Course IF301 tidak ditemukan setelah import apply: %v", err)
	}
}

// ============================================================================
// 5. Legacy Shim Tests (Header Deprecation & Backward Compatibility)
// ============================================================================

func TestLegacyShim_HeadersAndTelemetry(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Health
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Header().Get("Deprecation") != "true" {
		t.Errorf("GET /api/health expected Deprecation: true header")
	}

	// 2. Status
	req = httptest.NewRequest("GET", "/api/status", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Header().Get("Deprecation") != "true" {
		t.Errorf("GET /api/status expected Deprecation: true header")
	}

	var statusResp StatusResponse
	_ = json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp.V1 != "/api/v1/portal/:slug/summary" {
		t.Errorf("Field v1 pada status expected '/api/v1/portal/:slug/summary', got %s", statusResp.V1)
	}

	// 3. Classes
	req = httptest.NewRequest("GET", "/api/classes", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Header().Get("Deprecation") != "true" {
		t.Errorf("GET /api/classes expected Deprecation: true header")
	}
}
