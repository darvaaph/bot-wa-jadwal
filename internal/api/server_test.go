package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/database"
)

func TestAPIServer_Health(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/health", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp HealthResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
}

func TestAPIServer_Status(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/status", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var resp StatusResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", resp.Status)
	}
	if resp.BotConnection != "uninitialized" {
		t.Errorf("Expected bot_connection 'uninitialized', got '%s'", resp.BotConnection)
	}
}

func TestAPIServer_WebStatic(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for index.html, got %d", rr.Code)
	}
}

func TestAPIServer_AcademicEndpoints(t *testing.T) {
	testDB := "test_api_academic.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	db, err := database.InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	jadwalPath := filepath.Join("..", "..", "jadwal.json")
	if _, err := os.Stat(jadwalPath); os.IsNotExist(err) {
		jadwalPath = "jadwal.json"
	}

	if err := academic.SeedFromJSON(ctx, db, jadwalPath); err != nil {
		t.Fatalf("Gagal menyemai data akademik: %v", err)
	}

	repo := academic.NewRepository(db)
	server := NewServer(":8080", nil, nil, nil, repo)

	req := httptest.NewRequest("GET", "/api/academic/classes", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for /api/academic/classes, got %d", rr.Code)
	}

	var classesResp struct {
		Status string           `json:"status"`
		Data   []academic.Class `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&classesResp); err != nil {
		t.Fatalf("Gagal decode classes JSON: %v", err)
	}
	if len(classesResp.Data) == 0 {
		t.Fatal("Data kelas akademik tidak boleh kosong")
	}

	targetClassID := classesResp.Data[0].ID

	reqCourse := httptest.NewRequest("GET", fmt.Sprintf("/api/academic/classes/%d/courses", targetClassID), nil)
	rrCourse := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrCourse, reqCourse)

	if rrCourse.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for courses endpoint, got %d", rrCourse.Code)
	}

	var coursesResp struct {
		Status string            `json:"status"`
		Data   []academic.Course `json:"data"`
	}
	if err := json.NewDecoder(rrCourse.Body).Decode(&coursesResp); err != nil {
		t.Fatalf("Gagal decode courses JSON: %v", err)
	}
	if len(coursesResp.Data) == 0 {
		t.Fatal("Daftar mata kuliah tidak boleh kosong")
	}

	reqLegacy := httptest.NewRequest("GET", "/api/classes", nil)
	rrLegacy := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrLegacy, reqLegacy)

	if rrLegacy.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for /api/classes, got %d", rrLegacy.Code)
	}

	var legacyResp ClassesResponse
	if err := json.NewDecoder(rrLegacy.Body).Decode(&legacyResp); err != nil {
		t.Fatalf("Gagal decode legacy classes JSON: %v", err)
	}
	if legacyResp.Data.TotalClasses == 0 {
		t.Error("Diharapkan total classes > 0")
	}
}

func TestAcademicEndpointDoesNotLeakDatabaseError(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "closed.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	repo := academic.NewRepository(db)
	if err := db.Close(); err != nil {
		t.Fatalf("Gagal menutup database uji: %v", err)
	}

	server := NewServer(":8080", nil, nil, nil, repo)
	req := httptest.NewRequest(http.MethodGet, "/api/academic/classes", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", rr.Code)
	}
	body := strings.ToLower(rr.Body.String())
	if strings.Contains(body, "database is closed") || strings.Contains(body, "sql:") {
		t.Fatalf("Respons membocorkan detail database: %s", rr.Body.String())
	}
}

func TestAcademicCoursesRejectsNonPositiveClassID(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil, &academic.Repository{})
	req := httptest.NewRequest(http.MethodGet, "/api/academic/classes/-1/courses", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}
