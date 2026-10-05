package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestV1Master_RoomsAndCourses(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	adminToken := helperLogin(t, s, "+6281111111111", "password123")
	kmToken := helperLogin(t, s, "+6281234567890", "password123")

	// 1. KM boleh membaca master ruangan
	req := httptest.NewRequest("GET", "/api/v1/master/rooms", nil)
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("KM GET rooms expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 2. KM tidak boleh membuat ruangan
	body, _ := json.Marshal(map[string]any{"code": "R-999", "name": "Ruang Tes"})
	req = httptest.NewRequest("POST", "/api/v1/master/rooms", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kmToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("KM POST rooms expected 403, got %d", w.Code)
	}

	// 3. Admin membuat ruangan
	req = httptest.NewRequest("POST", "/api/v1/master/rooms", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin POST rooms expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Data["id"] == nil {
		t.Fatalf("respons create room tidak memuat id: %v", err)
	}

	// 4. Kode ganda ditolak
	req = httptest.NewRequest("POST", "/api/v1/master/rooms", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code == http.StatusCreated {
		t.Fatalf("kode ganda seharusnya ditolak")
	}

	// 4b. Bulk create rooms
	bulkRoomsPayload, _ := json.Marshal(map[string]any{
		"rooms": []map[string]any{
			{"code": "BULK-R1", "name": "Bulk Room 1", "building": "Gedung D", "room_type": "TEORI", "capacity": 32},
			{"code": "BULK-R2", "name": "Bulk Room 2", "building": "Gedung H", "room_type": "LAB", "capacity": 32},
		},
	})
	bulkRoomReq := httptest.NewRequest("POST", "/api/v1/master/rooms/bulk", bytes.NewReader(bulkRoomsPayload))
	bulkRoomReq.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, bulkRoomReq)
	if w.Code != http.StatusOK {
		t.Fatalf("admin POST /rooms/bulk expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 5. Admin membuat + menonaktifkan mata kuliah
	cbody, _ := json.Marshal(map[string]string{"code": "IF999", "name": "Mata Kuliah Tes"})
	req = httptest.NewRequest("POST", "/api/v1/master/courses", bytes.NewReader(cbody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin POST courses expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var createdCourse struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createdCourse)
	courseID := int(createdCourse.Data["id"].(float64))

	pbody, _ := json.Marshal(map[string]string{"status": "INACTIVE"})
	req = httptest.NewRequest("PATCH", "/api/v1/master/courses/"+strconv.Itoa(courseID), bytes.NewReader(pbody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin PATCH course expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 6. Daftar courses memuat yang baru
	req = httptest.NewRequest("GET", "/api/v1/master/courses", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !bytes.Contains(w.Body.Bytes(), []byte("IF999")) {
		t.Fatalf("daftar courses tidak memuat IF999: %d %s", w.Code, w.Body.String())
	}

	// 7. Bulk import courses
	bulkBody, _ := json.Marshal(map[string]any{
		"courses": []map[string]string{
			{"code": "IF801", "name": "Pemrograman Go"},
			{"code": "IF802", "name": "Arsitektur Microservices"},
		},
	})
	req = httptest.NewRequest("POST", "/api/v1/master/courses/bulk", bytes.NewReader(bulkBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin POST /courses/bulk expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// 8. Sinkronisasi master courses dari data/jadwal/*.json
	syncReq := httptest.NewRequest("POST", "/api/v1/master/courses/sync-jadwal", nil)
	syncReq.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, syncReq)
	if w.Code != http.StatusOK {
		t.Fatalf("admin POST /courses/sync-jadwal expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var syncRes struct {
		Data struct {
			TotalFound  int `json:"total_found"`
			TotalSynced int `json:"total_synced"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &syncRes); err != nil || syncRes.Data.TotalSynced < 40 {
		t.Fatalf("hasil sync jadwal tidak valid: %v, totalSynced: %d", err, syncRes.Data.TotalSynced)
	}

	// 9. Master Lecturers (Dosen): Create, Get, Patch, Bulk, Sync
	lecBody, _ := json.Marshal(map[string]string{"code": "ts", "full_name": "Test Dosen, S.T., M.T."})
	req = httptest.NewRequest("POST", "/api/v1/master/lecturers", bytes.NewReader(lecBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("admin POST /master/lecturers expected 201, got %d, body: %s", w.Code, w.Body.String())
	}
	var lecCreateRes struct {
		Data struct {
			ID     int64  `json:"id"`
			Code   string `json:"code"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &lecCreateRes)
	if lecCreateRes.Data.Code != "TS" {
		t.Fatalf("dosen code harus uppercase TS, got %s", lecCreateRes.Data.Code)
	}

	// Patch Dosen (ubah nama & nonaktifkan)
	lecPatchBody, _ := json.Marshal(map[string]string{"name": "Dosen Diperbarui, S.T.", "status": "INACTIVE"})
	req = httptest.NewRequest("PATCH", fmt.Sprintf("/api/v1/master/lecturers/%d", lecCreateRes.Data.ID), bytes.NewReader(lecPatchBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin PATCH /master/lecturers expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Bulk Create Dosen
	bulkLecBody, _ := json.Marshal([]map[string]string{
		{"code": "BK1", "full_name": "Dosen Bulk 1, M.T."},
		{"code": "BK2", "full_name": "Dosen Bulk 2, M.Kom."},
	})
	req = httptest.NewRequest("POST", "/api/v1/master/lecturers/bulk", bytes.NewReader(bulkLecBody))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("admin POST /master/lecturers/bulk expected 200, got %d, body: %s", w.Code, w.Body.String())
	}

	// Sync Dosen Mandiri (47 dosen resmi dari 19 jadwal POLBAN)
	req = httptest.NewRequest("POST", "/api/v1/master/lecturers/sync-jadwal", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /master/lecturers/sync-jadwal expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var lecSyncRes struct {
		Data struct {
			TotalFound  int `json:"total_found"`
			TotalSynced int `json:"total_synced"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &lecSyncRes); err != nil || lecSyncRes.Data.TotalSynced != 47 {
		t.Fatalf("sync dosen harus tepat 47 dosen resmi POLBAN, got %d (err: %v)", lecSyncRes.Data.TotalSynced, err)
	}

	// 10. Sync-All Master (41 Matkul + 18 Ruangan + 47 Dosen = 106 Total)
	syncAllReq := httptest.NewRequest("POST", "/api/v1/master/sync-all", nil)
	syncAllReq.Header.Set("Authorization", "Bearer "+adminToken)
	w = httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, syncAllReq)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /api/v1/master/sync-all expected 200, got %d, body: %s", w.Code, w.Body.String())
	}
	var syncAllRes struct {
		Data struct {
			CoursesSynced   int `json:"courses_synced"`
			RoomsSynced     int `json:"rooms_synced"`
			LecturersSynced int `json:"lecturers_synced"`
			TotalSynced     int `json:"total_synced"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &syncAllRes); err != nil || syncAllRes.Data.TotalSynced != 106 {
		t.Fatalf("sync all harus tepat 106 (41 matkul + 18 ruangan + 47 dosen), got %d (C: %d, R: %d, L: %d)",
			syncAllRes.Data.TotalSynced, syncAllRes.Data.CoursesSynced, syncAllRes.Data.RoomsSynced, syncAllRes.Data.LecturersSynced)
	}

	// Pastikan status INACTIVE pada TS tetap terlindungi dan tidak ditimpa
	var tsStatus string
	_ = s.v1DB.QueryRow(`SELECT status FROM lecturers WHERE code = 'TS';`).Scan(&tsStatus)
	if tsStatus != "INACTIVE" {
		t.Fatalf("status dosen TS harus tetap INACTIVE setelah sync-all, got %s", tsStatus)
	}
}
