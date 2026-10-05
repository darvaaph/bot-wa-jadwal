package api

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
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
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (1, 1, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
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
		INSERT INTO whatsapp_channels (id, class_id, jid, channel_type, display_name, status)
		VALUES (1, 1, '120363000000000001@g.us', 'GROUP', 'Kelas D4-TI-2024-A', 'ACTIVE');
	`)
	if err != nil {
		t.Fatalf("Gagal insert courses/offerings: %v", err)
	}

	// 4. Masukkan Pengguna (KM, PJ, dan SYSTEM_ADMIN) dengan password bcrypt: "password123"
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

	// 5. Masukkan Pola Jadwal & Teaching Event
	todayStr := time.Now().Format("2006-01-02")
	_, err = db.Exec(`
		INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from)
		VALUES (1, 1, 1, 1, '08:00', '09:40', ?);
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, published_by_user_id, published_at, version)
		VALUES (1, 'EXTRA', CURRENT_TIMESTAMP, datetime(CURRENT_TIMESTAMP, '+2 hours'), 1, 'Kuliah Tambahan', 'PUBLISHED', 1, CURRENT_TIMESTAMP, 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (1, 1, 'OWNER', 'ACCEPTED');
	`, todayStr)
	if err != nil {
		t.Fatalf("Gagal insert schedule items: %v", err)
	}

	// 6. Masukkan Penugasan Peran (KM, PJ, dan SYSTEM_ADMIN)
	_, err = db.Exec(`
		INSERT INTO role_assignments (id, user_id, role, scope_type, class_id, semester_id, course_offering_id, status)
		VALUES (1, 1, 'KM', 'CLASS', 1, NULL, NULL, 'ACTIVE'),
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
		                  deadline_at, task_type, publication_status, review_state, reviewed_version, submission_url, published_at, version)
		VALUES (1, 1, 2, 'Tugas Algoritma 1', 'Kerjakan latihan soal bab 1',
		        ?, 'INDIVIDUAL', 'PUBLISHED', 'APPROVED', 1, 'https://classroom.google.com', CURRENT_TIMESTAMP, 1);
	`, futureDeadline)
	if err != nil {
		t.Fatalf("Gagal insert initial task: %v", err)
	}

	// 8. Masukkan Materi Uji Awal
	_, err = db.Exec(`
		INSERT INTO materials (id, class_id, course_offering_id, title, material_type, url, description, created_by_user_id)
		VALUES (1, 1, 1, 'Slide Pertemuan 1', 'DOCUMENT', 'https://example.com/slide1.pdf', 'Pengenalan', 1);
	`)
	if err != nil {
		t.Fatalf("Gagal insert initial material: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	server := NewServer(":0", nil, nil, nil, db)
	server.SetStorageDir(t.TempDir())
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

// 1. Auth & Context Tests

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

func TestV1Auth_LoginHonorsSecureCookieConfig(t *testing.T) {
	_, s := setupV1TestEnv(t)
	s.SetSecureCookies(true)

	body := `{"identity_key":"+6281234567890","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	if len(cookies) == 0 || !cookies[0].Secure {
		t.Fatalf("cookie auth harus memiliki atribut Secure saat konfigurasi aktif")
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

func TestV1ManagementCollections_RequireAuthentication(t *testing.T) {
	_, s := setupV1TestEnv(t)

	for _, path := range []string{"/api/v1/classes", "/api/v1/materials"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("GET %s tanpa autentikasi: expected 401, got %d; body=%s", path, w.Code, w.Body.String())
			}
		})
	}
}

func TestV1Classes_AccessIsScopedForKMAdminPJAndPortal(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode)
		VALUES (2, 'Asia/Jakarta', 'LINK');
	`); err != nil {
		t.Fatalf("Gagal membuat kelas kedua: %v", err)
	}

	requestClasses := func(token, portalToken string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/classes", nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		if portalToken != "" {
			req.Header.Set("X-Portal-Token", portalToken)
		}
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		return w
	}
	classSlugs := func(w *httptest.ResponseRecorder) []string {
		var resp struct {
			Data struct {
				Classes []struct {
					Slug string `json:"slug"`
				} `json:"classes"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Respons classes tidak valid: %v", err)
		}
		result := make([]string, 0, len(resp.Data.Classes))
		for _, class := range resp.Data.Classes {
			result = append(result, class.Slug)
		}
		return result
	}

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w := requestClasses(kmToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("KM classes expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if slugs := classSlugs(w); len(slugs) != 1 || slugs[0] != "d4-ti-2024-a" {
		t.Fatalf("KM hanya boleh melihat kelas aktifnya, got %v", slugs)
	}

	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	w = requestClasses(adminToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("Admin classes expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if slugs := classSlugs(w); len(slugs) != 2 {
		t.Fatalf("Admin harus melihat seluruh kelas, got %v", slugs)
	}

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	w = requestClasses(pjToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("PJ classes expected 403, got %d: %s", w.Code, w.Body.String())
	}

	configurePortalCode(t, db, "123456")
	portalToken, _ := createPortalSession(t, s, "d4-ti-2024-a", "123456")
	w = requestClasses("", portalToken)
	if w.Code != http.StatusOK {
		t.Fatalf("Portal token classes expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if slugs := classSlugs(w); len(slugs) != 1 || slugs[0] != "d4-ti-2024-a" {
		t.Fatalf("Portal token hanya boleh melihat kelas terikat, got %v", slugs)
	}

	w = requestClasses("", "portal-token-invalid")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Portal token invalid expected 401, got %d", w.Code)
	}
}

func TestV1Auth_GetMe_Success(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	if _, err := db.Exec(`UPDATE role_assignments
		SET valid_from = datetime('now', '-2 hours'), valid_until = datetime('now', '-1 hour')
		WHERE id = 3`); err != nil {
		t.Fatalf("gagal menyiapkan assignment kedaluwarsa: %v", err)
	}

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
			Assignments []RoleAssignmentItem `json:"assignments"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Data.User.DisplayName != "Ketua Murid" {
		t.Errorf("Display name mismatch: %s", resp.Data.User.DisplayName)
	}
	if resp.Data.ActiveAssignment.Role != "KM" {
		t.Errorf("Role expected KM, got %s", resp.Data.ActiveAssignment.Role)
	}
	if len(resp.Data.Assignments) != 1 {
		t.Fatalf("assignments aktif expected 1, got %d: %#v", len(resp.Data.Assignments), resp.Data.Assignments)
	}
	if resp.Data.Assignments[0].Role != "KM" {
		t.Fatalf("assignments role mismatch: %#v", resp.Data.Assignments)
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

func TestV1Auth_SwitchContext_ExpiredAssignmentForbidden(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	token := helperLogin(t, s, "+6281234567890", "password123")
	if _, err := db.Exec(`UPDATE role_assignments
		SET valid_from = datetime('now', '-2 hours'), valid_until = datetime('now', '-1 hour')
		WHERE id = 3`); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]int64{"role_assignment_id": 3})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/switch-context", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("switch ke assignment kedaluwarsa expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// 2. Portal Read Endpoints Tests

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

	// 5b. Detail mata kuliah menggabungkan jadwal, tugas, dan materi terbit.
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/courses", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Struktur Data") || !strings.Contains(w.Body.String(), "Slide Pertemuan 1") {
		t.Errorf("Portal courses expected 200 with academic detail, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 6. Kelas tidak ada -> 404
	req = httptest.NewRequest("GET", "/api/v1/portal/kelas-fiktif/summary", nil)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Portal nonexistent class expected 404, got %d", w.Code)
	}
}

func TestLegacyTasks_GetReadsCanonicalV1Model(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`INSERT INTO tasks (
		course_offering_id, created_by_user_id, title, instructions, deadline_at,
		publication_status, review_state, version
	) VALUES (1, 2, 'Draf Tidak Boleh Bocor', 'Instruksi', ?, 'DRAFT', 'NOT_REVIEWED', 1)`,
		time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("Gagal membuat tugas draf: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/tasks?class=D4-TI-2024-A", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Legacy GET expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Deprecation") != "true" || !strings.Contains(w.Header().Get("Link"), "/api/v1/tasks") {
		t.Fatal("Legacy GET harus memiliki header deprecation dan successor-version")
	}

	var resp struct {
		Data []TaskResponseItem `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Respons legacy GET tidak valid: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Fatalf("Legacy GET harus membaca hanya tugas PUBLISHED dari v1, got %d", len(resp.Data))
	}
	item := resp.Data[0]
	if item.ID != 1 || item.ClassID != "D4-TI-2024-A" || item.Matkul != "Struktur Data (Teori)" || item.Deskripsi != "Tugas Algoritma 1" {
		t.Fatalf("Pemetaan legacy dari v1 tidak sesuai: %+v", item)
	}
}

func configurePortalCode(t *testing.T, db *sql.DB, code string) {
	t.Helper()
	hash := sha256.Sum256([]byte(strings.TrimSpace(code)))
	if _, err := db.Exec(`UPDATE class_settings
		SET portal_access_mode = 'CODE', portal_code_hash = ?, portal_code_version = 2
		WHERE class_id = 1`, hex.EncodeToString(hash[:])); err != nil {
		t.Fatalf("Gagal mengatur kode portal: %v", err)
	}
}

func createPortalSession(t *testing.T, s *Server, slug, code string) (string, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest("POST", "/api/v1/portal/"+slug+"/session", bytes.NewReader(body))
	req.RemoteAddr = "198.51.100.10:4567"
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create portal session expected 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Token     string `json:"portal_token"`
			ExpiresAt string `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Respons sesi portal tidak valid: %v", err)
	}
	if resp.Data.Token == "" || resp.Data.ExpiresAt == "" {
		t.Fatalf("Respons sesi portal tidak lengkap: %s", w.Body.String())
	}
	return resp.Data.Token, resp.Data.ExpiresAt
}

func TestV1PortalSession_ExchangeStoresOnlyTokenHash(t *testing.T) {
	db, s := setupV1TestEnv(t)
	configurePortalCode(t, db, "123456")

	token, expiresAt := createPortalSession(t, s, "d4-ti-2024-a", "123456")
	if _, err := time.Parse(time.RFC3339, expiresAt); err != nil {
		t.Fatalf("expires_at harus RFC3339: %q", expiresAt)
	}

	var storedHash string
	var version int
	if err := db.QueryRow(`SELECT token_hash, access_code_version FROM portal_sessions`).Scan(&storedHash, &version); err != nil {
		t.Fatalf("Sesi portal tidak tersimpan: %v", err)
	}
	expectedHash := sha256.Sum256([]byte(token))
	if storedHash == token || storedHash != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("Database harus menyimpan hash token, bukan token mentah")
	}
	if version != 2 {
		t.Fatalf("access_code_version expected 2, got %d", version)
	}

	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	req.Header.Set("X-Portal-Token", token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Token portal valid expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestV1PortalSession_RejectsInvalidExpiredAndRevokedTokens(t *testing.T) {
	t.Run("invalid code and unknown class use generic response", func(t *testing.T) {
		db, s := setupV1TestEnv(t)
		configurePortalCode(t, db, "123456")

		for _, tc := range []struct {
			slug string
			code string
		}{
			{slug: "d4-ti-2024-a", code: "000000"},
			{slug: "kelas-tidak-ada", code: "123456"},
		} {
			body, _ := json.Marshal(map[string]string{"code": tc.code})
			req := httptest.NewRequest("POST", "/api/v1/portal/"+tc.slug+"/session", bytes.NewReader(body))
			req.RemoteAddr = "198.51.100.11:4567"
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "Kode portal tidak valid") {
				t.Fatalf("Respons kode invalid harus generik, got %d: %s", w.Code, w.Body.String())
			}
		}
	})

	for _, tc := range []struct {
		name   string
		update string
	}{
		{name: "expired", update: `UPDATE portal_sessions SET created_at = '1999-01-01T00:00:00Z', expires_at = '2000-01-01T00:00:00Z'`},
		{name: "revoked", update: `UPDATE portal_sessions SET revoked_at = CURRENT_TIMESTAMP`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, s := setupV1TestEnv(t)
			configurePortalCode(t, db, "123456")
			token, _ := createPortalSession(t, s, "d4-ti-2024-a", "123456")
			if _, err := db.Exec(tc.update); err != nil {
				t.Fatalf("Gagal mengubah sesi portal: %v", err)
			}

			req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
			req.Header.Set("X-Portal-Token", token)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnauthorized {
				t.Fatalf("Token %s expected 401, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestV1PortalSession_RateLimitsFailedCodes(t *testing.T) {
	db, s := setupV1TestEnv(t)
	configurePortalCode(t, db, "123456")

	for attempt := 1; attempt <= 6; attempt++ {
		body := bytes.NewBufferString(`{"code":"000000"}`)
		req := httptest.NewRequest("POST", "/api/v1/portal/d4-ti-2024-a/session", body)
		req.RemoteAddr = "203.0.113.7:9999"
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)

		expected := http.StatusUnauthorized
		if attempt == 6 {
			expected = http.StatusTooManyRequests
		}
		if w.Code != expected {
			t.Fatalf("Percobaan %d expected %d, got %d. Body: %s", attempt, expected, w.Code, w.Body.String())
		}
	}
}

func rotatePortalCode(t *testing.T, s *Server, token, slug, payload string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/classes/"+slug+"/portal-code/rotate", strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	return w
}

func TestV1PortalCodeRotate_RevokesOldSessionsAndAuditsWithoutCode(t *testing.T) {
	db, s := setupV1TestEnv(t)
	configurePortalCode(t, db, "123456")
	oldToken, _ := createPortalSession(t, s, "d4-ti-2024-a", "123456")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	w := rotatePortalCode(t, s, kmToken, "d4-ti-2024-a", `{"code":"654321"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("Rotate portal code expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Code       string `json:"portal_code"`
			Version    int    `json:"portal_code_version"`
			RevealOnce bool   `json:"reveal_once"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Respons rotasi tidak valid: %v", err)
	}
	if resp.Data.Code != "654321" || resp.Data.Version != 3 || !resp.Data.RevealOnce {
		t.Fatalf("Respons rotasi tidak sesuai: %s", w.Body.String())
	}

	var mode, storedCodeHash string
	var version int
	if err := db.QueryRow(`SELECT portal_access_mode, portal_code_hash, portal_code_version
		FROM class_settings WHERE class_id = 1`).Scan(&mode, &storedCodeHash, &version); err != nil {
		t.Fatalf("Gagal membaca pengaturan portal: %v", err)
	}
	newCodeHash := sha256.Sum256([]byte("654321"))
	if mode != "CODE" || version != 3 || storedCodeHash != hex.EncodeToString(newCodeHash[:]) || storedCodeHash == "654321" {
		t.Fatalf("Pengaturan portal setelah rotasi tidak benar")
	}

	var revokedAt sql.NullString
	if err := db.QueryRow(`SELECT revoked_at FROM portal_sessions WHERE token_hash = ?`, func() string {
		h := sha256.Sum256([]byte(oldToken))
		return hex.EncodeToString(h[:])
	}()).Scan(&revokedAt); err != nil {
		t.Fatalf("Gagal membaca sesi lama: %v", err)
	}
	if !revokedAt.Valid {
		t.Fatal("Sesi portal versi lama harus dicabut secara eksplisit")
	}

	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	req.Header.Set("X-Portal-Token", oldToken)
	portalResponse := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(portalResponse, req)
	if portalResponse.Code != http.StatusUnauthorized {
		t.Fatalf("Token lama expected 401, got %d", portalResponse.Code)
	}
	createPortalSession(t, s, "d4-ti-2024-a", "654321")

	var actorUserID, actorAssignmentID int64
	var beforeJSON, afterJSON, correlationID string
	if err := db.QueryRow(`SELECT actor_user_id, actor_role_assignment_id, before_json, after_json, correlation_id
		FROM audit_logs WHERE action = 'ROTATE_PORTAL_CODE' ORDER BY id DESC LIMIT 1`).
		Scan(&actorUserID, &actorAssignmentID, &beforeJSON, &afterJSON, &correlationID); err != nil {
		t.Fatalf("Audit rotasi tidak ditemukan: %v", err)
	}
	if actorUserID != 1 || actorAssignmentID != 1 || strings.TrimSpace(correlationID) == "" {
		t.Fatalf("Aktor audit rotasi tidak lengkap")
	}
	if strings.Contains(beforeJSON+afterJSON, "654321") || strings.Contains(beforeJSON+afterJSON, storedCodeHash) {
		t.Fatal("Kode portal atau hash-nya tidak boleh masuk audit")
	}
}

func TestV1PortalCodeRotate_EnforcesScopeAndAllowsAdminGeneratedCode(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode)
		VALUES (2, 'Asia/Jakarta', 'LINK');
	`); err != nil {
		t.Fatalf("Gagal membuat kelas kedua: %v", err)
	}

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w := rotatePortalCode(t, s, kmToken, "d4-ti-2024-b", `{"code":"654321"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM lintas kelas expected 403, got %d. Body: %s", w.Code, w.Body.String())
	}

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	w = rotatePortalCode(t, s, pjToken, "d4-ti-2024-a", `{"code":"654321"}`)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PJ rotate portal code expected 403, got %d", w.Code)
	}

	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	w = rotatePortalCode(t, s, adminToken, "d4-ti-2024-b", `{}`)
	if w.Code != http.StatusOK {
		t.Fatalf("Admin rotate portal code expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Code string `json:"portal_code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil || len(resp.Data.Code) != 8 {
		t.Fatalf("Kode portal otomatis harus 8 digit: %s", w.Body.String())
	}
}

func TestV1PortalCodeRotate_RollsBackWhenAuditFails(t *testing.T) {
	db, s := setupV1TestEnv(t)
	configurePortalCode(t, db, "123456")
	oldToken, _ := createPortalSession(t, s, "d4-ti-2024-a", "123456")
	if _, err := db.Exec(`CREATE TRIGGER fail_portal_rotation_audit
		BEFORE INSERT ON audit_logs
		WHEN NEW.action = 'ROTATE_PORTAL_CODE'
		BEGIN SELECT RAISE(ABORT, 'forced audit failure'); END;`); err != nil {
		t.Fatalf("Gagal membuat trigger uji: %v", err)
	}
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	w := rotatePortalCode(t, s, kmToken, "d4-ti-2024-a", `{"code":"654321"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Audit gagal expected 500, got %d. Body: %s", w.Code, w.Body.String())
	}

	var version int
	var codeHash string
	if err := db.QueryRow(`SELECT portal_code_version, portal_code_hash FROM class_settings WHERE class_id = 1`).Scan(&version, &codeHash); err != nil {
		t.Fatalf("Gagal membaca pengaturan portal: %v", err)
	}
	originalHash := sha256.Sum256([]byte("123456"))
	if version != 2 || codeHash != hex.EncodeToString(originalHash[:]) {
		t.Fatal("Perubahan kode harus rollback ketika audit gagal")
	}

	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	req.Header.Set("X-Portal-Token", oldToken)
	portalResponse := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(portalResponse, req)
	if portalResponse.Code != http.StatusOK {
		t.Fatalf("Sesi lama harus tetap valid setelah rollback, got %d", portalResponse.Code)
	}
}

// 3. Tasks Management & Optimistic Locking Tests

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
	req := httptest.NewRequest("POST", "/api/v1/tasks/1/complete", strings.NewReader(`{"version":1}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Complete expected 200, got %d", w.Code)
	}

	// 2. Archive
	req = httptest.NewRequest("POST", "/api/v1/tasks/1/archive", strings.NewReader(`{"version":2}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Archive expected 200, got %d", w.Code)
	}

	// 3. Restore
	req = httptest.NewRequest("POST", "/api/v1/tasks/1/restore", strings.NewReader(`{"version":3}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Restore expected 200, got %d", w.Code)
	}
}

func TestV1Materials_PatchAndArchive(t *testing.T) {
	_, s := setupV1TestEnv(t)

	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	doReq := func(method, target, body string) *httptest.ResponseRecorder {
		t.Helper()
		var reader *strings.Reader
		if body == "" {
			reader = strings.NewReader("")
		} else {
			reader = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, target, reader)
		req.Header.Set("Authorization", "Bearer "+kmToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		return w
	}

	// PATCH versi salah -> 409
	w := doReq("PATCH", "/api/v1/materials/1", `{"version":99,"title":"X"}`)
	if w.Code != http.StatusConflict {
		t.Errorf("PATCH versi salah expected 409, got %d", w.Code)
	}

	// PATCH judul -> 200, version 2
	w = doReq("PATCH", "/api/v1/materials/1", `{"version":1,"title":"Slide Revisi"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	var patched struct {
		Data struct {
			Version int `json:"version"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &patched)
	if patched.Data.Version != 2 {
		t.Errorf("PATCH version expected 2, got %d", patched.Data.Version)
	}

	// GET memuat version + judul baru
	w = doReq("GET", "/api/v1/materials?class_slug=d4-ti-2024-a", "")
	var listed struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &listed)
	if len(listed.Data) == 0 {
		t.Fatalf("GET materials expected >= 1 item")
	}
	if v, _ := listed.Data[0]["version"].(float64); v != 2 {
		t.Errorf("GET version expected 2, got %v", listed.Data[0]["version"])
	}
	if listed.Data[0]["title"] != "Slide Revisi" {
		t.Errorf("GET title expected revisi, got %v", listed.Data[0]["title"])
	}

	// DELETE tanpa version -> 422
	w = doReq("DELETE", "/api/v1/materials/1", "")
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("DELETE tanpa version expected 422, got %d", w.Code)
	}

	// DELETE versi benar -> 200 ARCHIVED
	w = doReq("DELETE", "/api/v1/materials/1?version=2", "")
	if w.Code != http.StatusOK {
		t.Fatalf("DELETE expected 200, got %d; body=%s", w.Code, w.Body.String())
	}

	// GET tak lagi tampil
	w = doReq("GET", "/api/v1/materials?class_slug=d4-ti-2024-a", "")
	listed.Data = nil
	_ = json.Unmarshal(w.Body.Bytes(), &listed)
	if len(listed.Data) != 0 {
		t.Errorf("materi terarsip tak boleh tampil, got %d item", len(listed.Data))
	}

	// PATCH materi terarsip -> 422
	w = doReq("PATCH", "/api/v1/materials/1", `{"version":3,"title":"Y"}`)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("PATCH terarsip expected 422, got %d", w.Code)
	}
}

func TestV1Tasks_DetailIncludesTimestamps(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	getDetail := func() map[string]any {
		t.Helper()
		req := httptest.NewRequest("GET", "/api/v1/tasks/1", nil)
		req.Header.Set("Authorization", "Bearer "+kmToken)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("GET task detail expected 200, got %d; body=%s", w.Code, w.Body.String())
		}
		var resp struct {
			Data struct {
				Task map[string]any `json:"task"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("respons bukan JSON: %v", err)
		}
		return resp.Data.Task
	}

	// Awal: belum selesai/diarsip -> null + boolean false
	task := getDetail()
	if task["completed_at"] != nil {
		t.Errorf("completed_at awal expected null, got %v", task["completed_at"])
	}
	if task["archived_at"] != nil {
		t.Errorf("archived_at awal expected null, got %v", task["archived_at"])
	}

	// Complete -> completed_at terisi
	req := httptest.NewRequest("POST", "/api/v1/tasks/1/complete", strings.NewReader(`{"version":1}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Complete expected 200, got %d", w.Code)
	}
	task = getDetail()
	completed, _ := task["completed_at"].(string)
	if completed == "" {
		t.Errorf("completed_at expected terisi setelah complete, got %v", task["completed_at"])
	}

	// Archive -> archived_at terisi
	req = httptest.NewRequest("POST", "/api/v1/tasks/1/archive", strings.NewReader(`{"version":2}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Archive expected 200, got %d", w.Code)
	}
	task = getDetail()
	archived, _ := task["archived_at"].(string)
	if archived == "" {
		t.Errorf("archived_at expected terisi setelah archive, got %v", task["archived_at"])
	}
}

// 4. Fitur Lanjutan Tests (Status Kelas, Ruangan, Notifikasi, Audit, Backup, Admin, Impor)

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

func TestV1Classes_CreateClass(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. Non-admin harus ditolak 403
	body, _ := json.Marshal(map[string]any{
		"name":          "D4 TI 2025 C",
		"study_program": "D4 Teknik Informatika",
		"cohort_year":   2025,
		"group_label":   "C",
	})
	req := httptest.NewRequest("POST", "/api/v1/classes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM buat kelas expected 403, got %d", w.Code)
	}

	// 2. Admin berhasil membuat kelas
	req = httptest.NewRequest("POST", "/api/v1/classes", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Admin buat kelas expected 201, got %d, body: %s", w.Code, w.Body.String())
	}

	// 3. Verifikasi kelas tersimpan di database
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM classes WHERE slug = 'd4-ti-2025-c';`).Scan(&count)
	if count != 1 {
		t.Fatalf("Kelas baru tidak ditemukan di database")
	}
}

func TestV1Rooms_CandidatesAndConfirmation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	draftStart := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	draftEnd := time.Now().Add(26 * time.Hour).UTC().Format(time.RFC3339)
	_, err := db.Exec(`
		INSERT INTO rooms (id, code, name) VALUES (2, 'R-302', 'Ruang Kelas 302');
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, reason, lifecycle_status, version)
		VALUES (2, 'EXTRA', ?, ?, 'Draf untuk konfirmasi TU', 'DRAFT', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (2, 1, 'OWNER', 'ACCEPTED');
	`, draftStart, draftEnd)
	if err != nil {
		t.Fatalf("Gagal menyiapkan event draf: %v", err)
	}

	// 1. Cari kandidat ruangan
	startsAt := time.Now().Add(1 * time.Hour).UTC().Format(time.RFC3339)
	endsAt := time.Now().Add(3 * time.Hour).UTC().Format(time.RFC3339)
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/rooms/candidates?starts_at=%s&ends_at=%s", startsAt, endsAt), nil)
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET room candidates expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. Konfirmasi ruangan TU
	body, _ := json.Marshal(map[string]any{
		"room_id":             2,
		"confirmation_status": "CONFIRMED",
		"external_contact":    "TU Gedung A",
		"note":                "Disetujui staf TU",
	})
	req = httptest.NewRequest("POST", "/api/v1/teaching-events/2/room-confirmations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST room confirmation expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var roomID int64
	var version int
	if err := db.QueryRow(`SELECT room_id, version FROM teaching_events WHERE id = 2`).Scan(&roomID, &version); err != nil {
		t.Fatalf("Gagal membaca event setelah konfirmasi: %v", err)
	}
	if roomID != 2 || version != 2 {
		t.Fatalf("Konfirmasi harus memperbarui room/version secara atomik, got room=%d version=%d", roomID, version)
	}
	var auditCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'CREATE_ROOM_CONFIRMATION' AND entity_type = 'ROOM_CONFIRMATION'`).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("Konfirmasi harus menghasilkan satu audit log, count=%d err=%v", auditCount, err)
	}
}

func TestV1TeachingEventDetail_ScopeAndResponse(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	_, err := db.Exec(`
		INSERT INTO rooms (id, code, name) VALUES (2, 'R-302', 'Ruang Kelas 302');
		INSERT INTO room_confirmations
			(teaching_event_id, room_id, confirmation_status, external_contact, note, recorded_by_user_id, recorded_at, confirmed_at)
		VALUES (1, 1, 'CONFIRMED', 'TU Gedung A', 'Tersedia', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO courses (id, code, name) VALUES (2, 'TI202', 'Basis Data');
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 2, 'Basis Data (Teori)', 'TEORI');
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, lifecycle_status, version)
		VALUES (2, 'EXTRA', '2024-10-01T02:00:00Z', '2024-10-01T04:00:00Z', 'DRAFT', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (2, 2, 'OWNER', 'ACCEPTED'), (2, 1, 'PARTICIPANT', 'PENDING');
	`)
	if err != nil {
		t.Fatalf("Gagal menyiapkan detail event lintas kelas: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/teaching-events/1", nil)
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("PJ owner GET detail expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Data struct {
			Event struct {
				OfferingID int64 `json:"offering_id"`
			} `json:"event"`
			Participations []map[string]any `json:"participations"`
			Confirmations  []struct {
				RecordedBy struct {
					ID          int64  `json:"id"`
					DisplayName string `json:"display_name"`
				} `json:"recorded_by"`
			} `json:"confirmations"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Respons detail bukan JSON valid: %v", err)
	}
	if response.Data.Event.OfferingID != 1 || len(response.Data.Participations) != 1 || len(response.Data.Confirmations) != 1 {
		t.Fatalf("Respons detail tidak lengkap: %s", w.Body.String())
	}
	if response.Data.Confirmations[0].RecordedBy.ID != 1 || response.Data.Confirmations[0].RecordedBy.DisplayName == "" {
		t.Fatalf("Pelaku konfirmasi tidak terisi: %s", w.Body.String())
	}

	// KM kelas peserta boleh membaca undangan PENDING agar dapat merespons partisipasi.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/teaching-events/2", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("KM participant GET detail expected 200, got %d body=%s", w.Code, w.Body.String())
	}

	// PJ hanya boleh membaca event yang offering OWNER-nya sama dengan konteks aktif.
	req = httptest.NewRequest(http.MethodGet, "/api/v1/teaching-events/2", nil)
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PJ out-of-scope GET detail expected 403, got %d body=%s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/teaching-events/999", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("Admin missing event expected 404, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestV1RoomConfirmation_PJOwnerScopeAndDraftOnly(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	body := []byte(`{"room_id":1,"confirmation_status":"PENDING","external_contact":"TU"}`)

	// Event terbit ditolak; konfirmasi manual wajib terikat pada draf.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teaching-events/1/room-confirmations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("Confirmation on published event expected 409, got %d body=%s", w.Code, w.Body.String())
	}

	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO courses (id, code, name) VALUES (2, 'TI202', 'Basis Data');
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 2, 'Basis Data (Teori)', 'TEORI');
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, lifecycle_status, version)
		VALUES (2, 'EXTRA', '2024-10-01T02:00:00Z', '2024-10-01T04:00:00Z', 'DRAFT', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (2, 2, 'OWNER', 'ACCEPTED');
	`)
	if err != nil {
		t.Fatalf("Gagal menyiapkan event out-of-scope: %v", err)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/teaching-events/2/room-confirmations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("PJ out-of-scope confirmation expected 403, got %d body=%s", w.Code, w.Body.String())
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM room_confirmations WHERE teaching_event_id = 2`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("Out-of-scope confirmation must not be persisted, count=%d err=%v", count, err)
	}
}

func TestV1RoomConfirmation_RejectsUnavailableRoom(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	pjToken := helperLogin(t, s, "+6281298765432", "password123")
	startsAt := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	endsAt := startsAt.Add(2 * time.Hour)
	_, err := db.Exec(`
		INSERT INTO rooms (id, code, name) VALUES (2, 'R-302', 'Ruang Kelas 302');
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, lifecycle_status, version)
		VALUES (2, 'EXTRA', ?, ?, 'DRAFT', 1),
		       (3, 'EXTRA', ?, ?, 'DRAFT', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (2, 1, 'OWNER', 'ACCEPTED'), (3, 1, 'OWNER', 'ACCEPTED');
		UPDATE teaching_events
		SET room_id = 2, lifecycle_status = 'PUBLISHED', published_by_user_id = 2, published_at = CURRENT_TIMESTAMP
		WHERE id = 3;
	`, startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339), startsAt.Format(time.RFC3339), endsAt.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("Gagal menyiapkan konflik ruangan: %v", err)
	}

	body := []byte(`{"room_id":2,"confirmation_status":"CONFIRMED","external_contact":"TU"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/teaching-events/2/room-confirmations", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("Unavailable room confirmation expected 409, got %d body=%s", w.Code, w.Body.String())
	}

	var confirmations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM room_confirmations WHERE teaching_event_id = 2`).Scan(&confirmations); err != nil || confirmations != 0 {
		t.Fatalf("Konfirmasi konflik tidak boleh tersimpan, count=%d err=%v", confirmations, err)
	}
	var roomID sql.NullInt64
	var version int
	if err := db.QueryRow(`SELECT room_id, version FROM teaching_events WHERE id = 2`).Scan(&roomID, &version); err != nil {
		t.Fatalf("Gagal memeriksa event konflik: %v", err)
	}
	if roomID.Valid || version != 1 {
		t.Fatalf("Event konflik harus tetap utuh, room=%v version=%d", roomID, version)
	}
}

func TestV1Notifications_ListAndRetry(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Masukkan pesan notifikasi berstatus FAILED
	_, err := db.Exec(`
		INSERT INTO notification_messages (
			id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at
		)
		VALUES (10, 1, 1, 'TASK_PUBLISHED', 'TASK', 1,
		        'test-failed-notif', '{}', 'FAILED', CURRENT_TIMESTAMP);
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
	if _, err := db.Exec(`
		INSERT INTO audit_logs (actor_type, class_id, actor_user_id, action, entity_type, entity_id, correlation_id)
		VALUES ('USER', 1, 1, 'TEST_ACTION', 'CLASS', 1, 'test-audit-scope');
	`); err != nil {
		t.Fatalf("Gagal menyiapkan audit log: %v", err)
	}

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

	// 1. KM meminta backup; Admin mengeksekusi permintaan.
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	body, _ := json.Marshal(map[string]string{"class_slug": "d4-ti-2024-a", "reason": "Uji cadangan berkala"})
	req := httptest.NewRequest("POST", "/api/v1/backup-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("POST backup-requests expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	req = httptest.NewRequest("POST", "/api/v1/backup-requests/1/execute", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("execute backup request: %d %s", w.Code, w.Body.String())
	}

	var backupResp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &backupResp)
	backupID := backupResp.Data.ID

	// 2. Verifikasi restore (Admin, verify-only ADR-0008)
	restoreBody, _ := json.Marshal(map[string]any{"backup_id": backupID, "reason": "verifikasi berkala"})
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

func TestV1Semester_Preview(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	req := httptest.NewRequest("GET", "/api/v1/classes/d4-ti-2024-a/semesters/1/preview", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET preview expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			Offerings   int      `json:"offerings"`
			CanActivate bool     `json:"can_activate"`
			Blockers    []string `json:"blockers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respons bukan JSON: %v", err)
	}
	if resp.Data.Offerings < 1 {
		t.Errorf("preview offerings expected >= 1, got %d", resp.Data.Offerings)
	}
	if resp.Data.CanActivate {
		t.Errorf("semester ACTIVE tak boleh can_activate=true")
	}

	// Semester tak ada -> 404
	req = httptest.NewRequest("GET", "/api/v1/classes/d4-ti-2024-a/semesters/999/preview", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("preview semester tak ada expected 404, got %d", w.Code)
	}
}

func TestV1Semester_CreateWithSource(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	body, _ := json.Marshal(map[string]any{
		"academic_year": "2025/2026", "term": "GANJIL",
		"starts_on": "2025-09-01", "ends_on": "2026-01-31",
		"source_semester_id": 1,
	})
	req := httptest.NewRequest("POST", "/api/v1/classes/d4-ti-2024-a/semesters", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST semesters+source expected 201, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	var copied, original int
	_ = db.QueryRow(`SELECT COUNT(*) FROM course_offerings WHERE semester_id = ?;`, resp.Data.ID).Scan(&copied)
	_ = db.QueryRow(`SELECT COUNT(*) FROM course_offerings WHERE semester_id = 1;`).Scan(&original)
	if original == 0 || copied != original {
		t.Errorf("salinan offerings expected %d, got %d", original, copied)
	}

	// Sumber tak ada -> 404
	body, _ = json.Marshal(map[string]any{
		"academic_year": "2025/2026", "term": "GENAP",
		"starts_on": "2026-02-01", "ends_on": "2026-06-30",
		"source_semester_id": 999,
	})
	req = httptest.NewRequest("POST", "/api/v1/classes/d4-ti-2024-a/semesters", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("source tak ada expected 404, got %d", w.Code)
	}
}

func TestV1Semester_CreateOffering(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Semester ACTIVE (id 1) -> 422
	body, _ := json.Marshal(map[string]any{
		"course_code": "TI201", "activity_type": "TEORI", "display_name": "Struktur Data (Teori)",
	})
	req := httptest.NewRequest("POST", "/api/v1/semesters/1/offerings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("offering ke semester ACTIVE expected 422, got %d", w.Code)
	}

	// Buat semester DRAFT dulu
	draftBody, _ := json.Marshal(map[string]any{
		"academic_year": "2025/2026", "term": "GANJIL",
		"starts_on": "2025-09-01", "ends_on": "2026-01-31",
	})
	req = httptest.NewRequest("POST", "/api/v1/classes/d4-ti-2024-a/semesters", bytes.NewReader(draftBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("buat draf expected 201, got %d; body=%s", w.Code, w.Body.String())
	}
	var draftResp struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &draftResp)

	// Kode matkul tak dikenal -> 422
	badBody, _ := json.Marshal(map[string]any{
		"course_code": "XX999", "activity_type": "TEORI", "display_name": "Tidak Ada",
	})
	req = httptest.NewRequest("POST", "/api/v1/semesters/"+strconv.FormatInt(draftResp.Data.ID, 10)+"/offerings", bytes.NewReader(badBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("course tak dikenal expected 422, got %d", w.Code)
	}

	// Happy path -> 201
	req = httptest.NewRequest("POST", "/api/v1/semesters/"+strconv.FormatInt(draftResp.Data.ID, 10)+"/offerings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("tambah offering expected 201, got %d; body=%s", w.Code, w.Body.String())
	}
}

// 5. Legacy Shim Tests (Header Deprecation & Backward Compatibility)

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

func TestV1Notifications_AutoQueueOnPublish(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. KM membuat tugas dengan save_as = published
	body := `{
		"offering_id": 1,
		"title": "Tugas Notifikasi Otomatis",
		"instructions": "Pastikan tugas ini masuk antrean notifikasi WhatsApp",
		"deadline_at": "2026-10-01T23:59:00Z",
		"submission_url": "https://classroom.google.com/test",
		"save_as": "published"
	}`
	req := httptest.NewRequest("POST", "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Create published task expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// Verifikasi ada baris PENDING di notification_messages
	var notifCount int
	var eventType, status string
	err := db.QueryRow(`
		SELECT COUNT(*), event_type, status
		FROM notification_messages
		WHERE event_type = 'TASK_PUBLISHED' AND status = 'PENDING';
	`).Scan(&notifCount, &eventType, &status)
	if err != nil || notifCount != 1 {
		t.Errorf("Expected 1 PENDING TASK_PUBLISHED notification, got count=%d, err=%v", notifCount, err)
	}

	// 2. Terbitkan teaching event draft
	_, _ = db.Exec(`
		INSERT INTO teaching_events (
			id, origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status
		)
		VALUES (50, 1, '2024-10-07', 'REPLACEMENT',
		        '2024-10-07T08:00:00Z', '2024-10-07T10:00:00Z', 1, 'Kuliah pengganti', 'DRAFT');
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		)
		VALUES (50, 1, 'OWNER', 'ACCEPTED');
	`)

	req = httptest.NewRequest("POST", "/api/v1/teaching-events/50/publish", strings.NewReader(`{"version":1}`))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Idempotency-Key", "publish-event-50")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Publish teaching event expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verifikasi ada notifikasi SCHEDULE_REPLACEMENT
	var schedNotifCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM notification_messages
		WHERE event_type = 'SCHEDULE_REPLACEMENT' AND entity_id = 50 AND status = 'PENDING';
	`).Scan(&schedNotifCount)
	if err != nil || schedNotifCount != 1 {
		t.Errorf("Expected 1 PENDING SCHEDULE_REPLACEMENT notification, got count=%d, err=%v", schedNotifCount, err)
	}
}

func TestV1TeachingEventPublish_RequiresVersionAndIdempotencyKey(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")

	tests := []struct {
		name string
		body string
		key  string
	}{
		{name: "version kosong", body: `{}`, key: "publish-1"},
		{name: "idempotency key kosong", body: `{"version":1}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/teaching-events/1/publish", strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Idempotency-Key", tc.key)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422, got %d; body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestV1PatternPatch_RequiresVersion(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/schedule/patterns/1", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("patch pattern tanpa version: expected 422, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestV1TeachingEventPublish_IdempotencyKeyReplay(t *testing.T) {
	db, s := setupV1TestEnv(t)
	_, err := db.Exec(`
		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, lifecycle_status, version)
		VALUES (60, 'EXTRA', '2024-10-01T08:00:00Z', '2024-10-01T10:00:00Z', 'DRAFT', 1);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (60, 1, 'OWNER', 'ACCEPTED');
	`)
	if err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	publish := func(key string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/teaching-events/60/publish", strings.NewReader(`{"version":1}`))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		return w
	}
	if w := publish("same-key"); w.Code != http.StatusOK {
		t.Fatalf("publish awal expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	if w := publish("same-key"); w.Code != http.StatusOK {
		t.Fatalf("replay key sama expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	if w := publish("different-key"); w.Code != http.StatusConflict {
		t.Fatalf("replay key berbeda expected 409, got %d; body=%s", w.Code, w.Body.String())
	}
}

// 6. Security & Negative Authorization Tests (BOLA, Portal CODE, Role Validity)

func TestV1Auth_SuspendedRoleAssignment_Rejected(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Suspend KM role assignment
	_, err := db.Exec(`UPDATE role_assignments SET status = 'SUSPENDED' WHERE id = 1;`)
	if err != nil {
		t.Fatalf("Gagal update role assignment: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403 Forbidden for suspended role assignment, got %d", w.Code)
	}
}

func TestV1Portal_CodeMode_AccessControl(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Ubah portal_access_mode ke CODE
	codeHash := sha256.Sum256([]byte("rahasia123"))
	codeHashHex := hex.EncodeToString(codeHash[:])
	_, err := db.Exec(`
		UPDATE class_settings
		SET portal_access_mode = 'CODE', portal_code_hash = ?, portal_code_version = 1
		WHERE class_id = 1;
	`, codeHashHex)
	if err != nil {
		t.Fatalf("Gagal set CODE mode: %v", err)
	}

	// 2. Akses tanpa token -> harus ditolak 401
	req := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 for portal CODE mode without token, got %d", w.Code)
	}

	// 3. Masukkan sesi portal aktif
	rawToken := "valid-portal-token-xyz"
	tokenHash := sha256.Sum256([]byte(rawToken))
	tokenHashHex := hex.EncodeToString(tokenHash[:])
	futureExpiry := time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO portal_sessions (class_id, token_hash, access_code_version, expires_at)
		VALUES (1, ?, 1, ?);
	`, tokenHashHex, futureExpiry)
	if err != nil {
		t.Fatalf("Gagal insert portal session: %v", err)
	}

	// 4. Akses dengan X-Portal-Token valid -> 200 OK
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	req.Header.Set("X-Portal-Token", rawToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 with valid portal token, got %d", w.Code)
	}

	// 5. Akses dengan token salah -> 401
	req = httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	req.Header.Set("X-Portal-Token", "wrong-token")
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 with wrong portal token, got %d", w.Code)
	}
}

func TestV1Portal_RejectsInvalidFilters(t *testing.T) {
	_, s := setupV1TestEnv(t)
	for _, path := range []string{
		"/api/v1/portal/d4-ti-2024-a/tasks?group=sembarang",
		"/api/v1/portal/d4-ti-2024-a/changes?since=bukan-timestamp",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("filter invalid %s: expected 422, got %d; body=%s", path, w.Code, w.Body.String())
		}
	}
}

func TestV1Tasks_ScopeEnforcement_BOLA(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// Buat kelas kedua dan tugas milik kelas kedua
	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'TI-B', 'd4-ti-2024-b', 'Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (2, 'Asia/Jakarta', 'LINK');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 1, 'Struktur Data Kelas B', 'TEORI');
		INSERT INTO tasks (id, course_offering_id, created_by_user_id, title, instructions, deadline_at, submission_url, published_at, publication_status, version)
		VALUES (99, 2, 1, 'Tugas Kelas B', 'Petunjuk', '2026-11-01T10:00:00Z', 'https://classroom.google.com', CURRENT_TIMESTAMP, 'PUBLISHED', 1);
	`)
	if err != nil {
		t.Fatalf("Gagal setup class 2 tasks: %v", err)
	}

	// PJ kelas A (offering 1) mencoba mengakses tugas 99 (offering 2) -> 403
	pjToken := helperLogin(t, s, "+6281298765432", "password123")

	req := httptest.NewRequest("GET", "/api/v1/tasks/99", nil)
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("PJ expected 403 when accessing task from another offering, got %d", w.Code)
	}

	// KM kelas A mencoba mereview tugas kelas B -> 403
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	reviewBody, _ := json.Marshal(map[string]any{
		"decision":     "APPROVED",
		"task_version": 1,
	})
	req = httptest.NewRequest("POST", "/api/v1/tasks/99/reviews", bytes.NewReader(reviewBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("KM expected 403 when reviewing task from another class, got %d", w.Code)
	}

	// PJ kelas A mencoba menyelesaikan tugas kelas B -> 403
	req = httptest.NewRequest("POST", "/api/v1/tasks/99/complete", strings.NewReader(`{"version":1}`))
	req.Header.Set("Authorization", "Bearer "+pjToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("PJ expected 403 when completing task from another offering, got %d", w.Code)
	}
}

func TestV1TaskLifecycle_RequiresVersion(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/complete", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("lifecycle tanpa version: expected 422, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestV1TaskReview_RequiresNoteForNegativeDecision(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/1/reviews", strings.NewReader(`{"decision":"CHANGES_REQUESTED","task_version":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("review negatif tanpa note: expected 422, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestV1ClassStatus_WritesCanonicalAudit(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/classes/d4-ti-2024-a", strings.NewReader(`{"status":"INACTIVE"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("patch status expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	var actorType, correlationID string
	if err := db.QueryRow(`SELECT actor_type, correlation_id FROM audit_logs WHERE action = 'UPDATE_CLASS_STATUS' ORDER BY id DESC LIMIT 1;`).Scan(&actorType, &correlationID); err != nil {
		t.Fatalf("audit canonical tidak tersimpan: %v", err)
	}
	if actorType != "USER" || strings.TrimSpace(correlationID) == "" {
		t.Fatalf("audit canonical tidak lengkap: actor_type=%q correlation_id=%q", actorType, correlationID)
	}
}

func TestV1SemesterAndMaterial_RejectCrossClassKM(t *testing.T) {
	db, s := setupV1TestEnv(t)
	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (2, 'Asia/Jakarta', 'LINK');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 1, 'Struktur Data Kelas B', 'TEORI');
	`)
	if err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")

	semesterReq := httptest.NewRequest(http.MethodGet, "/api/v1/classes/d4-ti-2024-b/semesters", nil)
	semesterReq.Header.Set("Authorization", "Bearer "+token)
	semesterW := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(semesterW, semesterReq)
	if semesterW.Code != http.StatusForbidden {
		t.Fatalf("semester lintas kelas expected 403, got %d", semesterW.Code)
	}

	materialBody := `{"class_slug":"d4-ti-2024-b","offering_id":2,"title":"Lintas kelas","material_type":"DOCUMENT"}`
	materialReq := httptest.NewRequest(http.MethodPost, "/api/v1/materials", strings.NewReader(materialBody))
	materialReq.Header.Set("Authorization", "Bearer "+token)
	materialW := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(materialW, materialReq)
	if materialW.Code != http.StatusForbidden {
		t.Fatalf("material lintas kelas expected 403, got %d; body=%s", materialW.Code, materialW.Body.String())
	}
}

func TestV1Materials_GetIncludesCreatedAt(t *testing.T) {
	_, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/materials?class_slug=d4-ti-2024-a", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET materials expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Gagal parsing respons materials: %v", err)
	}
	if len(resp.Data) == 0 {
		t.Fatal("GET materials expected minimal 1 item")
	}
	created, _ := resp.Data[0]["created_at"].(string)
	if created == "" {
		t.Fatal("GET materials expected created_at terisi untuk urutan terbaru/terlama")
	}
}

func TestV1Tasks_CreateRejectsOfferingOutsideActiveClass(t *testing.T) {
	db, s := setupV1TestEnv(t)
	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 1, 'Struktur Data Kelas B', 'TEORI');
	`)
	if err != nil {
		t.Fatal(err)
	}

	token := helperLogin(t, s, "+6281234567890", "password123")
	body := `{"offering_id":2,"title":"Lintas kelas","save_as":"draft"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("KM membuat task lintas kelas: expected 403, got %d; body=%s", w.Code, w.Body.String())
	}
}

func TestV1TeachingEvents_ScopeAndConflict(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Revoke dengan versi salah -> 409
	revokeBody, _ := json.Marshal(map[string]any{
		"reason":  "Dosen berhalangan",
		"version": 99, // Mismatched version
	})
	req := httptest.NewRequest("POST", "/api/v1/teaching-events/1/revoke", bytes.NewReader(revokeBody))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict for mismatched version on revoke, got %d", w.Code)
	}
}

func TestV1Schedule_DeletePattern(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Cari pattern aktif
	var patternID int64
	var version int
	err := db.QueryRow(`
		SELECT sp.id, sp.version
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		WHERE co.id = 1 AND sp.effective_until IS NULL
		LIMIT 1
	`).Scan(&patternID, &version)
	if err != nil {
		t.Fatalf("No active pattern found for test: %v", err)
	}

	// Hapus tanpa version -> 422
	req := httptest.NewRequest("DELETE", "/api/v1/schedule/patterns/"+strconv.FormatInt(patternID, 10), nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("DELETE pattern without version expected 422, got %d", w.Code)
	}

	// Hapus dengan versi salah -> 409
	req = httptest.NewRequest("DELETE", "/api/v1/schedule/patterns/"+strconv.FormatInt(patternID, 10)+"?version=99", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("DELETE pattern wrong version expected 409, got %d", w.Code)
	}

	// Hapus dengan versi benar -> 200
	req = httptest.NewRequest("DELETE", "/api/v1/schedule/patterns/"+strconv.FormatInt(patternID, 10)+"?version="+strconv.Itoa(version), nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("DELETE pattern correct version expected 200, got %d; body=%s", w.Code, w.Body.String())
	}

	// Baris tetap ada untuk audit: effective_until terisi, bukan hapus fisik
	var effUntil sql.NullString
	err = db.QueryRow(`SELECT effective_until FROM schedule_patterns WHERE id = ?`, patternID).Scan(&effUntil)
	if err != nil {
		t.Fatalf("pola terhapus fisik, riwayat audit hilang: %v", err)
	}
	if !effUntil.Valid || effUntil.String == "" {
		t.Errorf("effective_until kosong setelah hapus, pola masih dianggap aktif")
	}
}

func TestV1Schedule_DeleteTeachingEvent(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// Buat draf event untuk dihapus
	draftBody := `{"owner_offering_id":1,"event_kind":"EXTRA","starts_at":"2024-12-01T10:00:00+07:00","ends_at":"2024-12-01T12:00:00+07:00","reason":"Test hapus draf"}`
	req := httptest.NewRequest("POST", "/api/v1/teaching-events", bytes.NewReader([]byte(draftBody)))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Create draft for delete test expected 201, got %d; body=%s", w.Code, w.Body.String())
	}
	var draftResp struct {
		Data struct {
			ID      int64 `json:"id"`
			Version int   `json:"version"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &draftResp)
	eventID := draftResp.Data.ID
	eventVersion := draftResp.Data.Version

	// Hapus tanpa version -> 422
	req = httptest.NewRequest("DELETE", "/api/v1/teaching-events/"+strconv.FormatInt(eventID, 10), nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("DELETE event without version expected 422, got %d", w.Code)
	}

	// Hapus dengan versi salah -> 409
	req = httptest.NewRequest("DELETE", "/api/v1/teaching-events/"+strconv.FormatInt(eventID, 10)+"?version=99", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("DELETE event wrong version expected 409, got %d", w.Code)
	}

	// Coba hapus PUBLISHED event -> 409
	pubEventID := int64(1)
	req = httptest.NewRequest("DELETE", "/api/v1/teaching-events/"+strconv.FormatInt(pubEventID, 10)+"?version=1", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("DELETE published event expected 409, got %d", w.Code)
	}

	// Hapus draf dengan versi benar -> 200
	req = httptest.NewRequest("DELETE", "/api/v1/teaching-events/"+strconv.FormatInt(eventID, 10)+"?version="+strconv.Itoa(eventVersion), nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("DELETE draft event correct version expected 200, got %d; body=%s", w.Code, w.Body.String())
	}
}
