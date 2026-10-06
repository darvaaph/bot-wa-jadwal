package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

	if rr.Code != http.StatusFound {
		t.Errorf("Expected status 302 for / -> /login.html, got %d", rr.Code)
	}
	if loc := rr.Header().Get("Location"); loc != "/login.html" {
		t.Errorf("Expected Location /login.html, got %q", loc)
	}

	reqLogin := httptest.NewRequest("GET", "/login.html", nil)
	rrLogin := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrLogin, reqLogin)
	if rrLogin.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /login.html, got %d", rrLogin.Code)
	}

	reqIndex := httptest.NewRequest("GET", "/index.html", nil)
	rrIndex := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrIndex, reqIndex)
	if rrIndex.Code != http.StatusFound {
		t.Errorf("Expected status 302 for /index.html -> /login.html, got %d", rrIndex.Code)
	}
	if loc := rrIndex.Header().Get("Location"); loc != "/login.html" {
		t.Errorf("Expected Location /login.html, got %q", loc)
	}

	assertLoginRedirect := func(method, target, wantLoc string) {
		t.Helper()
		req := httptest.NewRequest(method, target, nil)
		rr := httptest.NewRecorder()
		server.httpServer.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Errorf("%s %s: status = %d, mau 302", method, target, rr.Code)
		}
		if loc := rr.Header().Get("Location"); loc != wantLoc {
			t.Errorf("%s %s: Location = %q, mau %q", method, target, loc, wantLoc)
		}
	}
	assertLoginRedirect("GET", "/?role=sa", "/login.html?role=sa")
	assertLoginRedirect("GET", "/index.html?role=sa", "/login.html?role=sa")
	assertLoginRedirect("HEAD", "/", "/login.html")
	assertLoginRedirect("HEAD", "/index.html", "/login.html")

	reqPortal := httptest.NewRequest("GET", "/c/kelas-coba", nil)
	rrPortal := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrPortal, reqPortal)
	if rrPortal.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /c/kelas-coba (portal index.html), got %d", rrPortal.Code)
	}

	reqJS := httptest.NewRequest("GET", "/js/vendor/tailwindcss.play.js", nil)
	rrJS := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrJS, reqJS)
	t.Logf("tailwindcss.play.js Status: %d, Content-Type: %q", rrJS.Code, rrJS.Header().Get("Content-Type"))

	reqCSS := httptest.NewRequest("GET", "/css/style.css", nil)
	rrCSS := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrCSS, reqCSS)
	t.Logf("style.css Status: %d, Content-Type: %q", rrCSS.Code, rrCSS.Header().Get("Content-Type"))

	reqTailwind := httptest.NewRequest("GET", "/css/tailwind.css", nil)
	rrTailwind := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rrTailwind, reqTailwind)
	if rrTailwind.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /css/tailwind.css, got %d", rrTailwind.Code)
	}
	t.Logf("tailwind.css Status: %d, Content-Type: %q, Length: %d", rrTailwind.Code, rrTailwind.Header().Get("Content-Type"), rrTailwind.Body.Len())
}
