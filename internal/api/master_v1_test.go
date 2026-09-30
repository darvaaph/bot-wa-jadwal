package api

import (
	"bytes"
	"encoding/json"
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
}
