package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIServer_Health(t *testing.T) {
	server := NewServer(":8080", nil, nil)

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
	server := NewServer(":8080", nil, nil)

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
	server := NewServer(":8080", nil, nil)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for index.html, got %d", rr.Code)
	}
}

