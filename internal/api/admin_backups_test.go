package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestV1Backups_CreateWithSemester(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	token := helperLogin(t, s, "+6281111111111", "password123")
	s.SetStorageDir(t.TempDir())

	// 1. Buat dengan semester milik kelas -> 201 + semester_id tersimpan
	w := helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "semester_id": 1, "reason": "sebelum migrasi",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "artifact_ref") || strings.Contains(w.Body.String(), ".db") {
		t.Fatalf("respons create membocorkan path internal: %s", w.Body.String())
	}
	var semID *int64
	_ = db.QueryRow(`SELECT semester_id FROM backup_records WHERE id = 1;`).Scan(&semID)
	if semID == nil || *semID != 1 {
		t.Fatalf("semester_id tak tersimpan: %v", semID)
	}

	// 2. Semester milik kelas lain -> 404
	w = helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "semester_id": 2, "reason": "salah scope",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("semester lain expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Tanpa semester tetap bisa (kolom nullable)
	w = helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "reason": "tanpa semester",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("tanpa semester expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// 4. semester_id<=0 -> 422 (bukan fallback diam-diam)
	w = helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "semester_id": 0, "reason": "nol",
	})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("semester 0 expected 422, got %d: %s", w.Code, w.Body.String())
	}

	// 5. KM di luar cakupan -> 404 generik (bukan 403 yang mengungkap keberadaan)
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "POST", "/api/v1/backups", kmToken, map[string]any{
		"class_slug": "d4-ti-2024-b", "reason": "intip",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM luar cakupan expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Audit create memuat alasan + semester, tanpa path internal
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'CREATE_BACKUP' AND reason = 'sebelum migrasi';`).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit CREATE_BACKUP beralasan expected 1, got %d", nAudit)
	}
	var afterJSON string
	_ = db.QueryRow(`SELECT after_json FROM audit_logs WHERE action = 'CREATE_BACKUP' AND reason = 'sebelum migrasi';`).Scan(&afterJSON)
	if strings.Contains(afterJSON, "backups/") || strings.Contains(afterJSON, ".db") {
		t.Fatalf("audit membocorkan path internal: %s", afterJSON)
	}
	if !strings.Contains(afterJSON, `"semester_id":1`) {
		t.Fatalf("audit tanpa semester_id: %s", afterJSON)
	}
}

func TestV1Backups_List(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	token := helperLogin(t, s, "+6281111111111", "password123")
	s.SetStorageDir(t.TempDir())

	helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "semester_id": 1, "reason": "rutin",
	})
	helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-b", "reason": "kelas lain",
	})

	// 1. SA -> 200, tanpa artifact_ref, ada class_slug + semester
	w := helperDo(t, s, "GET", "/api/v1/backups", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("SA expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "artifact_ref") || strings.Contains(w.Body.String(), ".db") {
		t.Fatalf("daftar membocorkan path internal")
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 cadangan, got %d", len(body.Data))
	}
	bySlug := map[string]map[string]any{}
	for _, it := range body.Data {
		bySlug[it["class_slug"].(string)] = it
	}
	if bySlug["d4-ti-2024-a"]["semester_id"] != float64(1) {
		t.Errorf("cadangan kelas a tanpa semester_id benar: %v", bySlug["d4-ti-2024-a"])
	}
	if _, ok := bySlug["d4-ti-2024-b"]["semester_id"]; ok {
		t.Errorf("cadangan tanpa semester tak boleh punya semester_id: %v", bySlug["d4-ti-2024-b"])
	}

	// 2. Filter class_slug + status
	w = helperDo(t, s, "GET", "/api/v1/backups?class_slug=d4-ti-2024-b", token, nil)
	var f struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	if len(f.Data) != 1 {
		t.Fatalf("filter kelas expected 1, got %d", len(f.Data))
	}
	w = helperDo(t, s, "GET", "/api/v1/backups?status=READY", token, nil)
	var fs struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &fs)
	if len(fs.Data) != 2 {
		t.Fatalf("filter READY expected 2, got %d", len(fs.Data))
	}

	// 3. Tanpa token -> 401
	w = helperDo(t, s, "GET", "/api/v1/backups", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}

	// 4. KM hanya kelasnya; filter slug lain -> 404
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "GET", "/api/v1/backups", kmToken, nil)
	var k struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &k)
	if len(k.Data) != 1 || k.Data[0]["class_slug"] != "d4-ti-2024-a" {
		t.Fatalf("KM expected 1 cadangan kelasnya, got %v", k.Data)
	}
	w = helperDo(t, s, "GET", "/api/v1/backups?class_slug=d4-ti-2024-b", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM slug lain expected 404, got %d", w.Code)
	}
}

func TestV1Backups_VerifySemesterScope(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	token := helperLogin(t, s, "+6281111111111", "password123")
	s.SetStorageDir(t.TempDir())

	// Backup semester-valid -> verify OK
	w := helperDo(t, s, "POST", "/api/v1/backups", token, map[string]any{
		"class_slug": "d4-ti-2024-a", "semester_id": 1, "reason": "scope valid",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "POST", "/api/v1/restores", token, map[string]any{
		"backup_id": 1, "reason": "verifikasi scope",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("verify scope valid expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"semester_id":1`) {
		t.Fatalf("respons verify harus menggema cakupan semester: %s", w.Body.String())
	}

	// Rekam dengan semester kelas lain pada artefak kelas 1 -> 422 scope semester
	var artifact, checksum string
	_ = db.QueryRow(`SELECT artifact_ref, checksum FROM backup_records WHERE id = 1;`).Scan(&artifact, &checksum)
	_, err := db.Exec(`
		INSERT INTO backup_records (class_id, semester_id, artifact_ref, checksum, status, created_by_user_id, reason)
		VALUES (1, 2, ?, ?, 'READY', 3, 'scope salah');
	`, artifact, checksum)
	if err != nil {
		t.Fatalf("gagal seed scope salah: %v", err)
	}
	w = helperDo(t, s, "POST", "/api/v1/restores", token, map[string]any{
		"backup_id": 2, "reason": "uji scope",
	})
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "semester") {
		t.Fatalf("scope semester salah expected 422 semester, got %d: %s", w.Code, w.Body.String())
	}
}
