package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"bot-jadwal/internal/task"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "api_test.db")
	// Handler tugas lama memakai tabel pra-v3 yang terpisah dari schema target.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Gagal inisialisasi test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	tm, err := task.NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi TaskManager: %v", err)
	}

	return NewServer(":8080", nil, nil, tm)
}

func performRequest(t *testing.T, s *Server, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}

	rr := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(rr, req)
	return rr
}

func decodeResponse(t *testing.T, rr *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.NewDecoder(rr.Body).Decode(out); err != nil {
		t.Fatalf("Gagal decode respons JSON: %v", err)
	}
}

func TestTaskHandler_GetTasks_Success(t *testing.T) {
	s := newTestServer(t)

	_, _, err := s.taskManager.AddWebTask("SISTEM BASIS DATA", "Latihan ERD halaman 40", "Jumat 11 sep 23:59", "web-dashboard", time.Now())
	if err != nil {
		t.Fatalf("Gagal menambahkan tugas uji: %v", err)
	}
	_, _, err = s.taskManager.AddWebTask("JARINGAN KOMPUTER", "Subnetting VLSM", "Senin 14 Sep 12:00", "web-dashboard", time.Now())
	if err != nil {
		t.Fatalf("Gagal menambahkan tugas uji: %v", err)
	}

	rr := performRequest(t, s, "GET", "/api/tasks", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Status string             `json:"status"`
		Data   []TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("Expected 2 tasks, got %d", len(resp.Data))
	}

	matkuls := make(map[string]bool)
	for _, item := range resp.Data {
		matkuls[item.Matkul] = true
		if item.Deadline == "" {
			t.Errorf("Expected non-empty deadline label for task %q", item.Matkul)
		}
		if item.IsDone {
			t.Errorf("Expected is_done false for task %q", item.Matkul)
		}
	}
	if !matkuls["SISTEM BASIS DATA"] {
		t.Errorf("Expected 'SISTEM BASIS DATA' in response data")
	}
	if !matkuls["JARINGAN KOMPUTER"] {
		t.Errorf("Expected 'JARINGAN KOMPUTER' in response data")
	}

	for _, item := range resp.Data {
		if item.Matkul == "SISTEM BASIS DATA" && item.Deskripsi != "Latihan ERD halaman 40" {
			t.Errorf("Unexpected deskripsi for SISTEM BASIS DATA: %s", item.Deskripsi)
		}
		if item.Matkul == "SISTEM BASIS DATA" && item.ID <= 0 {
			t.Errorf("Expected positive id, got %d", item.ID)
		}
	}
}

func TestTaskHandler_GetTasks_Empty(t *testing.T) {
	s := newTestServer(t)

	rr := performRequest(t, s, "GET", "/api/tasks?class=TI-2A", nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Status string             `json:"status"`
		Data   []TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}
	if len(resp.Data) != 0 {
		t.Errorf("Expected empty data array, got %d items", len(resp.Data))
	}
	if resp.Data == nil {
		t.Errorf("Expected data to be an empty array, got null")
	}
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	s := newTestServer(t)

	body := []byte(`{
		"matkul": "JARINGAN KOMPUTER",
		"deskripsi": "Subnetting VLSM",
		"deadline": "Senin, 14 Sep 12:00 WIB"
	}`)

	rr := performRequest(t, s, "POST", "/api/tasks", body)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Status string           `json:"status"`
		Data   TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}
	if resp.Data.ID <= 0 {
		t.Errorf("Expected positive task id, got %d", resp.Data.ID)
	}
	if resp.Data.Matkul != "JARINGAN KOMPUTER" {
		t.Errorf("Expected matkul 'JARINGAN KOMPUTER', got '%s'", resp.Data.Matkul)
	}
	if resp.Data.IsDone {
		t.Errorf("Expected is_done false for new task")
	}

	items, err := s.taskManager.GetAllActiveTasks(time.Now())
	if err != nil {
		t.Fatalf("Gagal membaca tugas: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 task in database, got %d", len(items))
	}
}

func TestTaskHandler_CreateTask_MissingMatkul(t *testing.T) {
	s := newTestServer(t)

	body := []byte(`{"matkul": "", "deskripsi": "Subnetting VLSM"}`)
	rr := performRequest(t, s, "POST", "/api/tasks", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}

	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", resp.Status)
	}
	if resp.Error == "" {
		t.Errorf("Expected non-empty error message")
	}
}

func TestTaskHandler_CreateTask_MissingDeskripsi(t *testing.T) {
	s := newTestServer(t)

	body := []byte(`{"matkul": "SISTEM BASIS DATA", "deskripsi": "  "}`)
	rr := performRequest(t, s, "POST", "/api/tasks", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}

	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", resp.Status)
	}
	if resp.Error == "" {
		t.Errorf("Expected non-empty error message")
	}
}

func TestTaskHandler_CreateTask_InvalidJSON(t *testing.T) {
	s := newTestServer(t)

	body := []byte(`{"matkul": "SISTEM BASIS DATA", deskripsi: invalid}`)
	rr := performRequest(t, s, "POST", "/api/tasks", body)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}

func TestTaskHandler_DeleteTask_Success(t *testing.T) {
	s := newTestServer(t)

	id, _, err := s.taskManager.AddWebTask("SISTEM BASIS DATA", "Latihan ERD", "Jumat 11 sep 23:59", "web-dashboard", time.Now())
	if err != nil {
		t.Fatalf("Gagal menambahkan tugas uji: %v", err)
	}

	rr := performRequest(t, s, "DELETE", "/api/tasks/"+strconv.FormatInt(id, 10), nil)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr.Code)
	}

	var resp struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", resp.Status)
	}
	if resp.Message == "" {
		t.Errorf("Expected non-empty success message")
	}

	items, err := s.taskManager.GetAllActiveTasks(time.Now())
	if err != nil {
		t.Fatalf("Gagal membaca tugas: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected no active tasks after delete, got %d", len(items))
	}
}

func TestTaskHandler_DeleteTask_NotFound(t *testing.T) {
	s := newTestServer(t)

	rr := performRequest(t, s, "DELETE", "/api/tasks/9999", nil)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("Expected status 404, got %d", rr.Code)
	}

	var resp struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", resp.Status)
	}
}

func TestTaskHandler_DeleteTask_InvalidID(t *testing.T) {
	s := newTestServer(t)

	rr := performRequest(t, s, "DELETE", "/api/tasks/abc", nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}

func TestTaskHandler_DeleteTask_NegativeID(t *testing.T) {
	s := newTestServer(t)

	rr := performRequest(t, s, "DELETE", "/api/tasks/-1", nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}

func TestTaskHandler_UninitializedTaskManager(t *testing.T) {
	s := NewServer(":8080", nil, nil, nil)

	rr := performRequest(t, s, "GET", "/api/tasks", nil)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rr.Code)
	}
}

func TestTaskHandler_CreateTask_WithClassID(t *testing.T) {
	s := newTestServer(t)

	body := []byte(`{
		"class_id": "D4-TI-3A",
		"matkul": "SISTEM OPERASI",
		"deskripsi": "Praktikum Shell Scripting",
		"deadline": "Besok 23:59"
	}`)

	rr := performRequest(t, s, "POST", "/api/tasks", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Status string           `json:"status"`
		Data   TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr, &resp)

	if resp.Data.ClassID != "D4-TI-3A" {
		t.Errorf("Expected ClassID 'D4-TI-3A', got '%s'", resp.Data.ClassID)
	}
	if resp.Data.Matkul != "SISTEM OPERASI" {
		t.Errorf("Expected Matkul 'SISTEM OPERASI', got '%s'", resp.Data.Matkul)
	}
}

func TestTaskHandler_GetTasks_FilterByClass(t *testing.T) {
	s := newTestServer(t)

	_, _, err := s.taskManager.AddWebTask("SBD", "Tugas 3A", "besok 23:59", "web-dashboard", time.Now(), "D4-TI-3A")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	_, _, err = s.taskManager.AddWebTask("ALPRO", "Tugas 1A", "lusa 12:00", "web-dashboard", time.Now(), "D3-TI-1A")
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	rr3A := performRequest(t, s, "GET", "/api/tasks?class=D4-TI-3A", nil)
	if rr3A.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", rr3A.Code)
	}
	var resp3A struct {
		Status string             `json:"status"`
		Data   []TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr3A, &resp3A)
	if len(resp3A.Data) != 1 {
		t.Fatalf("Expected 1 task for D4-TI-3A, got %d", len(resp3A.Data))
	}
	if resp3A.Data[0].Matkul != "SBD" || resp3A.Data[0].ClassID != "D4-TI-3A" {
		t.Errorf("Unexpected task content for 3A: %+v", resp3A.Data[0])
	}

	rr1A := performRequest(t, s, "GET", "/api/tasks?class=D3-TI-1A", nil)
	var resp1A struct {
		Status string             `json:"status"`
		Data   []TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rr1A, &resp1A)
	if len(resp1A.Data) != 1 {
		t.Fatalf("Expected 1 task for D3-TI-1A, got %d", len(resp1A.Data))
	}
	if resp1A.Data[0].Matkul != "ALPRO" || resp1A.Data[0].ClassID != "D3-TI-1A" {
		t.Errorf("Unexpected task content for 1A: %+v", resp1A.Data[0])
	}

	rrAll := performRequest(t, s, "GET", "/api/tasks", nil)
	var respAll struct {
		Status string             `json:"status"`
		Data   []TaskResponseItem `json:"data"`
	}
	decodeResponse(t, rrAll, &respAll)
	if len(respAll.Data) != 2 {
		t.Fatalf("Expected 2 tasks in total, got %d", len(respAll.Data))
	}
}
