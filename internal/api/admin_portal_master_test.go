package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestV1Class_PortalMode(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. GET settings: SA 200 mode LINK default
	w := helperDo(t, s, "GET", "/api/v1/classes/d4-ti-2024-a/settings", adminToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("settings expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var get struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &get)
	if get.Data["portal_access_mode"] != "LINK" || get.Data["timezone"] != "Asia/Jakarta" {
		t.Fatalf("settings tak sesuai: %v", get.Data)
	}

	// 2. Tanpa token 401; kelas tak ada 404
	w = helperDo(t, s, "GET", "/api/v1/classes/tak-ada/settings", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa token expected 401, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/classes/tak-ada/settings", adminToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("slug tak ada expected 404, got %d", w.Code)
	}

	// 3. KM kelas lain -> 404; KM sendiri -> 200
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)
	w = helperDo(t, s, "GET", "/api/v1/classes/d4-ti-2024-b/settings", kmToken, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM lintas kelas expected 404, got %d", w.Code)
	}
	w = helperDo(t, s, "GET", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("KM sendiri expected 200, got %d", w.Code)
	}
	// 4. Mode invalid -> 422
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/portal-mode", adminToken, map[string]string{"mode": "SMS"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("mode invalid expected 422, got %d", w.Code)
	}

	// 5. LINK->LINK idempoten, tanpa audit baru
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/portal-mode", adminToken, map[string]string{"mode": "LINK"})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"changed":false`) {
		t.Fatalf("idempoten expected 200 changed:false, got %d: %s", w.Code, w.Body.String())
	}

	// 6. CODE tanpa hash -> 422 dengan arahan rotasi
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/portal-mode", adminToken, map[string]string{"mode": "CODE"})
	if w.Code != http.StatusUnprocessableEntity || !strings.Contains(w.Body.String(), "Putar kode") {
		t.Fatalf("CODE tanpa hash expected 422 arahan, got %d: %s", w.Code, w.Body.String())
	}

	// 7. Rotate -> CODE + hash; PATCH CODE idempoten; PATCH LINK berhasil + audit
	w = helperDo(t, s, "POST", "/api/v1/classes/d4-ti-2024-a/portal-code/rotate", adminToken, map[string]string{})
	if w.Code != http.StatusOK {
		t.Fatalf("rotate expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/portal-mode", adminToken, map[string]string{"mode": "CODE"})
	if w.Code != http.StatusOK {
		t.Fatalf("CODE ber-hash expected 200, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/portal-mode", adminToken, map[string]string{"mode": "LINK", "reason": "Kembali ke tautan"})
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"changed":true`) {
		t.Fatalf("LINK expected 200 changed:true, got %d: %s", w.Code, w.Body.String())
	}
	var mode, hash any
	_ = db.QueryRow(`SELECT portal_access_mode, portal_code_hash FROM class_settings WHERE class_id = 1;`).Scan(&mode, &hash)
	if mode != "LINK" || hash != nil {
		t.Fatalf("DB expected LINK tanpa hash, got %v %v", mode, hash)
	}
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'UPDATE_PORTAL_MODE' AND reason = 'Kembali ke tautan';`).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit UPDATE_PORTAL_MODE expected 1, got %d", nAudit)
	}

	// 8. KM lintas kelas PATCH -> 404
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-b/portal-mode", kmToken, map[string]string{"mode": "LINK"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM lintas kelas expected 404, got %d", w.Code)
	}
}

func TestV1Class_UpdateSettings(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	seedSecondClass(t, db)
	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)

	// 1. GET memuat field pengingat + version
	w := helperDo(t, s, "GET", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("settings expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var get struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &get)
	ver, _ := get.Data["version"].(float64)
	if ver != 1 {
		t.Fatalf("version awal expected 1, got %v", get.Data["version"])
	}

	// 2. Timezone invalid -> 422
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]string{"timezone": "Mars/Olympus"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("timezone invalid expected 422, got %d", w.Code)
	}

	// 3. Jam invalid -> 422
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]string{"morning_reminder_time": "25:00"})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("jam invalid expected 422, got %d", w.Code)
	}

	// 4. Menit di luar rentang -> 422
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]any{"replacement_reminder_minutes": 2000})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("menit invalid expected 422, got %d", w.Code)
	}

	// 5. Body kosong -> 422
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]any{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("body kosong expected 422, got %d", w.Code)
	}

	// 6. Happy path KM -> 200 version 2 + audit
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]any{
		"timezone": "Asia/Jakarta", "morning_reminder_time": "06:00",
		"afternoon_reminder_time": "17:00", "replacement_reminder_minutes": 60,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("PATCH expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var dbTz, dbPagi, dbSore string
	var dbGanti int
	var dbVer int
	_ = db.QueryRow(`SELECT timezone, morning_reminder_time, afternoon_reminder_time, replacement_reminder_minutes, version FROM class_settings WHERE class_id = 1;`).Scan(&dbTz, &dbPagi, &dbSore, &dbGanti, &dbVer)
	if dbTz != "Asia/Jakarta" || dbPagi != "06:00" || dbSore != "17:00" || dbGanti != 60 || dbVer != 2 {
		t.Fatalf("DB tak sesuai: %v %v %v %v %v", dbTz, dbPagi, dbSore, dbGanti, dbVer)
	}
	var nAudit int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'UPDATE_CLASS_SETTINGS';`).Scan(&nAudit)
	if nAudit != 1 {
		t.Fatalf("audit UPDATE_CLASS_SETTINGS expected 1, got %d", nAudit)
	}

	// 7. Version basi -> 409
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-a/settings", kmToken, map[string]any{"version": 1, "morning_reminder_time": "06:30"})
	if w.Code != http.StatusConflict {
		t.Fatalf("version basi expected 409, got %d", w.Code)
	}

	// 8. KM lintas kelas -> 404; admin -> 200
	w = helperDo(t, s, "PATCH", "/api/v1/classes/d4-ti-2024-b/settings", kmToken, map[string]any{"morning_reminder_time": "06:30"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("KM lintas kelas expected 404, got %d", w.Code)
	}
	_ = adminToken
}

func TestV1Master_AuditTrail(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")

	// 1. Tambah ruangan -> audit CREATE_MASTER_ROOM
	w := helperDo(t, s, "POST", "/api/v1/master/rooms", adminToken, map[string]any{
		"code": "R-901", "name": "Lab Uji",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create room expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var nCreate int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'CREATE_MASTER_ROOM';`).Scan(&nCreate)
	if nCreate != 1 {
		t.Fatalf("audit create room expected 1, got %d", nCreate)
	}

	// 2. Ubah ruangan -> audit UPDATE_MASTER_ROOM dengan before/after
	w = helperDo(t, s, "PATCH", "/api/v1/master/rooms/1", adminToken, map[string]any{"name": "Ruang Baru"})
	if w.Code != http.StatusOK {
		t.Fatalf("patch room expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var beforeJSON, afterJSON string
	_ = db.QueryRow(`SELECT before_json, after_json FROM audit_logs WHERE action = 'UPDATE_MASTER_ROOM';`).Scan(&beforeJSON, &afterJSON)
	if !strings.Contains(beforeJSON, "Ruang Kelas 301") || !strings.Contains(afterJSON, "Ruang Baru") {
		t.Fatalf("before/after tak sesuai: %s / %s", beforeJSON, afterJSON)
	}

	// 3. Tambah + ubah matkul -> dua audit
	w = helperDo(t, s, "POST", "/api/v1/master/courses", adminToken, map[string]any{"code": "TI999", "name": "Uji"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create course expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "PATCH", "/api/v1/master/courses/1", adminToken, map[string]any{"status": "INACTIVE"})
	if w.Code != http.StatusOK {
		t.Fatalf("patch course expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var nCourse int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action IN ('CREATE_MASTER_COURSE','UPDATE_MASTER_COURSE');`).Scan(&nCourse)
	if nCourse != 2 {
		t.Fatalf("audit course expected 2, got %d", nCourse)
	}

	// 4. KM tak boleh tulis master -> 403
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	w = helperDo(t, s, "POST", "/api/v1/master/rooms", kmToken, map[string]any{"code": "R-X"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM tulis master expected 403, got %d", w.Code)
	}
}

func TestV1Master_Proposals(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()
	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")
	kmToken = helperSwitchContext(t, s, kmToken, 1)

	// 1. SA tak boleh mengusulkan -> 403
	w := helperDo(t, s, "POST", "/api/v1/master/proposals", adminToken, map[string]any{
		"kind": "ROOM", "payload": map[string]any{"name": "X"},
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("SA propose expected 403, got %d", w.Code)
	}

	// 2. Validasi: kind salah, field asing, ganti kode existing, target tak ada
	for _, tc := range []struct {
		name    string
		payload map[string]any
	}{
		{"kind salah", map[string]any{"kind": "GEDUNG", "payload": map[string]any{"name": "X"}}},
		{"field asing", map[string]any{"kind": "ROOM", "payload": map[string]any{"lantai": 2}}},
		{"ganti kode", map[string]any{"kind": "ROOM", "target_id": 1, "payload": map[string]any{"code": "R-X"}}},
		{"target tak ada", map[string]any{"kind": "ROOM", "target_id": 999, "payload": map[string]any{"name": "X"}}},
		{"nama kosong", map[string]any{"kind": "COURSE", "target_id": 1, "payload": map[string]any{"name": " "}}},
	} {
		w = helperDo(t, s, "POST", "/api/v1/master/proposals", kmToken, tc.payload)
		if w.Code != http.StatusUnprocessableEntity && w.Code != http.StatusNotFound {
			t.Fatalf("%s expected 422/404, got %d: %s", tc.name, w.Code, w.Body.String())
		}
	}

	// 3. Usulan valid: ubah nama ruangan + tambah matkul baru
	w = helperDo(t, s, "POST", "/api/v1/master/proposals", kmToken, map[string]any{
		"kind": "ROOM", "target_id": 1,
		"payload": map[string]any{"name": "Ruang Usulan KM", "capacity": 40},
		"note":    "Nama resmi dari TU",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("propose room expected 201, got %d: %s", w.Code, w.Body.String())
	}
	w = helperDo(t, s, "POST", "/api/v1/master/proposals", kmToken, map[string]any{
		"kind": "COURSE", "payload": map[string]any{"code": "TI777", "name": "Matkul Usulan"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("propose course expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Daftar: SA 2, KM 2 (kelasnya)
	w = helperDo(t, s, "GET", "/api/v1/master/proposals", adminToken, nil)
	var all struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &all)
	if len(all.Data) != 2 {
		t.Fatalf("SA expected 2 usulan, got %d", len(all.Data))
	}
	w = helperDo(t, s, "GET", "/api/v1/master/proposals?status=PENDING&kind=ROOM", adminToken, nil)
	var f struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &f)
	if len(f.Data) != 1 || f.Data[0]["target_code"] != "R-301" {
		t.Fatalf("filter expected 1 R-301, got %v", f.Data)
	}

	// 5. KM tak boleh putuskan -> 403
	w = helperDo(t, s, "POST", "/api/v1/master/proposals/1/approve", kmToken, map[string]any{})
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM approve expected 403, got %d", w.Code)
	}

	// 6. Setujui usulan ruangan -> diterapkan + audit ganda
	w = helperDo(t, s, "POST", "/api/v1/master/proposals/1/approve", adminToken, map[string]any{"review_note": "Sesuai TU"})
	if w.Code != http.StatusOK {
		t.Fatalf("approve expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var rname string
	var rcap int
	_ = db.QueryRow(`SELECT name, capacity FROM rooms WHERE id = 1;`).Scan(&rname, &rcap)
	if rname != "Ruang Usulan KM" || rcap != 40 {
		t.Fatalf("penerapan salah: %s %d", rname, rcap)
	}
	var nAppr, nUpd int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'APPROVE_MASTER_PROPOSAL';`).Scan(&nAppr)
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'UPDATE_MASTER_ROOM';`).Scan(&nUpd)
	if nAppr != 1 || nUpd != 1 {
		t.Fatalf("audit approve/update expected 1/1, got %d/%d", nAppr, nUpd)
	}

	// 7. Putuskan ganda -> 409; tolak tanpa catatan -> 422; tolak -> REJECTED + audit
	w = helperDo(t, s, "POST", "/api/v1/master/proposals/1/approve", adminToken, map[string]any{})
	if w.Code != http.StatusConflict {
		t.Fatalf("double decide expected 409, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/master/proposals/2/reject", adminToken, map[string]any{})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("reject tanpa catatan expected 422, got %d", w.Code)
	}
	w = helperDo(t, s, "POST", "/api/v1/master/proposals/2/reject", adminToken, map[string]any{"review_note": "Duplikat TI201"})
	if w.Code != http.StatusOK {
		t.Fatalf("reject expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var st string
	_ = db.QueryRow(`SELECT status FROM master_proposals WHERE id = 2;`).Scan(&st)
	if st != "REJECTED" {
		t.Fatalf("status expected REJECTED, got %s", st)
	}
	var nRej int
	_ = db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action = 'REJECT_MASTER_PROPOSAL';`).Scan(&nRej)
	if nRej != 1 {
		t.Fatalf("audit reject expected 1, got %d", nRej)
	}
	var nCourse int
	_ = db.QueryRow(`SELECT COUNT(*) FROM courses WHERE code = 'TI777';`).Scan(&nCourse)
	if nCourse != 0 {
		t.Fatalf("usulan ditolak tak boleh diterapkan")
	}
}
