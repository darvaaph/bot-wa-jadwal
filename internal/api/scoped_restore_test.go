package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
)

func TestScopedRestore_ClassAcademicData(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	s.SetStorageDir(t.TempDir())
	seedSecondClass(t, db)
	admin := helperLogin(t, s, "+6281111111111", "password123")
	created := helperDo(t, s, "POST", "/api/v1/backups", admin, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "sebelum uji restore"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	var response struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET title='Judul baru' WHERE id=1; UPDATE users SET display_name='Nama terbaru' WHERE id=1;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET deleted_at=CURRENT_TIMESTAMP, deleted_by_user_id=1 WHERE id=1; INSERT INTO tasks(course_offering_id,created_by_user_id,title,publication_status) VALUES(1,1,'Tugas sesudah backup','DRAFT'); UPDATE course_offerings SET display_name='Kelas lain tetap' WHERE id=2;`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO lecturers(id,code,full_name) VALUES(99,'DOSEN-BARU','Dosen Baru'); INSERT INTO offering_lecturers(course_offering_id,lecturer_id,responsibility) VALUES(1,99,'ASSISTANT')`); err != nil {
		t.Fatal(err)
	}
	preview := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{})
	if preview.Code != http.StatusOK {
		t.Fatalf("preview: %d %s", preview.Code, preview.Body.String())
	}
	var p struct {
		Data struct {
			Token      string `json:"preview_token"`
			CanRestore bool   `json:"can_restore"`
		} `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Data.CanRestore || p.Data.Token == "" {
		t.Fatalf("bad preview: %s", preview.Body.String())
	}
	if _, err := db.Exec(`UPDATE course_offerings SET display_name='Nama berubah setelah pratinjau' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	stale := helperDo(t, s, "POST", "/api/v1/backups/1/restore-execute", admin, map[string]any{"reason": "uji stale", "preview_token": p.Data.Token})
	if stale.Code != http.StatusConflict {
		t.Fatalf("stale: %d %s", stale.Code, stale.Body.String())
	}
	preview = helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{})
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	executed := helperDo(t, s, "POST", "/api/v1/backups/1/restore-execute", admin, map[string]any{"reason": "uji restore", "preview_token": p.Data.Token})
	if executed.Code != http.StatusOK {
		t.Fatalf("execute: %d %s", executed.Code, executed.Body.String())
	}
	var title, name string
	if err := db.QueryRow(`SELECT title FROM tasks WHERE id=1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT display_name FROM users WHERE id=1`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if title != "Tugas Algoritma 1" || name != "Nama terbaru" {
		t.Fatalf("scope restore salah: title=%s user=%s", title, name)
	}
	var deleted any
	if err := db.QueryRow(`SELECT deleted_at FROM tasks WHERE id=1`).Scan(&deleted); err != nil {
		t.Fatal(err)
	}
	if deleted != nil {
		t.Fatalf("task dari backup tetap tersembunyi: %v", deleted)
	}
	var newCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE title='Tugas sesudah backup' AND deleted_at IS NULL`).Scan(&newCount); err != nil {
		t.Fatal(err)
	}
	if newCount != 0 {
		t.Fatalf("task sesudah backup masih aktif")
	}
	var activeLecturer int
	if err := db.QueryRow(`SELECT COUNT(*) FROM offering_lecturers WHERE course_offering_id=1 AND lecturer_id=99 AND superseded_at IS NULL`).Scan(&activeLecturer); err != nil {
		t.Fatal(err)
	}
	if activeLecturer != 0 {
		t.Fatal("dosen yang ditambahkan setelah backup masih aktif")
	}
	var other string
	if err := db.QueryRow(`SELECT display_name FROM course_offerings WHERE id=2`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if other != "Kelas lain tetap" {
		t.Fatalf("kelas lain berubah: %s", other)
	}
	var corrections int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE class_id=1 AND event_type='ACADEMIC_RESTORE_CORRECTION' AND status='PENDING'`).Scan(&corrections); err != nil {
		t.Fatal(err)
	}
	if corrections != 1 {
		t.Fatalf("koreksi restore harus satu, got %d", corrections)
	}
}

func TestScopedRestore_RollsBackOnInvalidPackage(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	s.SetStorageDir(t.TempDir())
	admin := helperLogin(t, s, "+6281111111111", "password123")
	created := helperDo(t, s, "POST", "/api/v1/backups", admin, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "uji rollback"})
	if created.Code != 201 {
		t.Fatalf("create: %s", created.Body.String())
	}
	var path string
	if err := db.QueryRow(`SELECT artifact_ref FROM backup_records WHERE id=1`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var packageData map[string]any
	if err := json.Unmarshal(content, &packageData); err != nil {
		t.Fatal(err)
	}
	tasks := packageData["tasks"].([]any)
	tasks[0].(map[string]any)["course_offering_id"] = float64(99999)
	modified, err := json.Marshal(packageData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, modified, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(modified)
	if _, err := db.Exec(`UPDATE backup_records SET checksum=? WHERE id=1`, fmt.Sprintf("%x", sum)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET title='Tetap aktif' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	preview := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{})
	if preview.Code != 200 {
		t.Fatalf("preview: %d %s", preview.Code, preview.Body.String())
	}
	var p struct {
		Data struct {
			Token string `json:"preview_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	result := helperDo(t, s, "POST", "/api/v1/backups/1/restore-execute", admin, map[string]any{"reason": "uji rollback", "preview_token": p.Data.Token})
	if result.Code != 500 {
		t.Fatalf("expected rollback, got %d %s", result.Code, result.Body.String())
	}
	var title string
	if err := db.QueryRow(`SELECT title FROM tasks WHERE id=1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Tetap aktif" {
		t.Fatalf("transaction partial: %s", title)
	}
}

func TestScopedRestore_PreRestorePointIsUsable(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	s.SetStorageDir(t.TempDir())
	admin := helperLogin(t, s, "+6281111111111", "password123")
	created := helperDo(t, s, "POST", "/api/v1/backups", admin, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "titik awal"})
	if created.Code != 201 {
		t.Fatalf("create: %s", created.Body.String())
	}
	if _, err := db.Exec(`UPDATE tasks SET title='Versi sebelum restore' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	preview := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{})
	var p struct {
		Data struct {
			Token string `json:"preview_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	result := helperDo(t, s, "POST", "/api/v1/backups/1/restore-execute", admin, map[string]any{"reason": "kembali ke awal", "preview_token": p.Data.Token})
	if result.Code != 200 {
		t.Fatalf("restore: %d %s", result.Code, result.Body.String())
	}
	var restored struct {
		Data struct {
			PrePoint int64 `json:"pre_restore_backup_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result.Body.Bytes(), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Data.PrePoint <= 1 {
		t.Fatalf("prepoint hilang: %s", result.Body.String())
	}
	path := fmt.Sprintf("/api/v1/backups/%d/restore-preview", restored.Data.PrePoint)
	preview = helperDo(t, s, "POST", path, admin, map[string]any{})
	if preview.Code != 200 {
		t.Fatalf("prepoint preview: %d %s", preview.Code, preview.Body.String())
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	result = helperDo(t, s, "POST", fmt.Sprintf("/api/v1/backups/%d/restore-execute", restored.Data.PrePoint), admin, map[string]any{"reason": "uji titik pemulihan", "preview_token": p.Data.Token})
	if result.Code != 200 {
		t.Fatalf("prepoint execute: %d %s", result.Code, result.Body.String())
	}
	var title string
	if err := db.QueryRow(`SELECT title FROM tasks WHERE id=1`).Scan(&title); err != nil {
		t.Fatal(err)
	}
	if title != "Versi sebelum restore" {
		t.Fatalf("prepoint tidak dapat dipakai: %s", title)
	}
}

func TestScopedRestore_RejectsCrossClassEvent(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	s.SetStorageDir(t.TempDir())
	seedSecondClass(t, db)
	admin := helperLogin(t, s, "+6281111111111", "password123")
	created := helperDo(t, s, "POST", "/api/v1/backups", admin, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "sebelum partisipasi"})
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %s", created.Body.String())
	}
	if _, err := db.Exec(`INSERT INTO teaching_event_offerings (teaching_event_id,course_offering_id,participation_role,participation_status) VALUES (1,2,'PARTICIPANT','ACCEPTED')`); err != nil {
		t.Fatal(err)
	}
	preview := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{})
	if preview.Code != http.StatusOK || !json.Valid(preview.Body.Bytes()) {
		t.Fatalf("preview: %d %s", preview.Code, preview.Body.String())
	}
	var p struct {
		Data struct {
			CanRestore bool    `json:"can_restore"`
			Blockers   []int64 `json:"cross_class_event_ids"`
		} `json:"data"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Data.CanRestore || len(p.Data.Blockers) != 1 || p.Data.Blockers[0] != 1 {
		t.Fatalf("missing blocker: %s", preview.Body.String())
	}
}

func TestBackupRequest_RoleTransitions(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	s.SetStorageDir(t.TempDir())
	km := helperLogin(t, s, "+6281234567890", "password123")
	km = helperSwitchContext(t, s, km, 1)
	pj := helperLogin(t, s, "+6281298765432", "password123")
	admin := helperLogin(t, s, "+6281111111111", "password123")
	input := map[string]any{"class_slug": "d4-ti-2024-a", "reason": "sebelum semester baru"}
	if w := helperDo(t, s, "POST", "/api/v1/backups", km, input); w.Code != 403 {
		t.Fatalf("KM direct backup: %d", w.Code)
	}
	if w := helperDo(t, s, "POST", "/api/v1/backup-requests", pj, input); w.Code != 403 {
		t.Fatalf("PJ request: %d", w.Code)
	}
	if w := helperDo(t, s, "POST", "/api/v1/backup-requests", km, input); w.Code != 201 {
		t.Fatalf("KM request: %d %s", w.Code, w.Body.String())
	}
	if w := helperDo(t, s, "POST", "/api/v1/backup-requests/1/execute", km, nil); w.Code != 403 {
		t.Fatalf("KM execute: %d", w.Code)
	}
	if w := helperDo(t, s, "POST", "/api/v1/backup-requests/1/execute", admin, nil); w.Code != 200 {
		t.Fatalf("Admin execute: %d %s", w.Code, w.Body.String())
	}
	if w := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", km, map[string]any{}); w.Code != 403 {
		t.Fatalf("KM restore preview: %d", w.Code)
	}
	if w := helperDo(t, s, "POST", "/api/v1/backups/1/restore-preview", admin, map[string]any{}); w.Code != 200 {
		t.Fatalf("Admin restore preview: %d %s", w.Code, w.Body.String())
	}
}
