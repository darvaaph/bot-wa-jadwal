package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1Portal_LegacySlugResolution(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	// 1. Uji slug kanonikal: d4-ti-2024-a
	reqCanonical := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-2024-a/summary", nil)
	wCanonical := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wCanonical, reqCanonical)
	if wCanonical.Code != http.StatusOK {
		t.Fatalf("Canonical slug d4-ti-2024-a expected 200, got %d, body: %s", wCanonical.Code, wCanonical.Body.String())
	}

	var respCanonical struct {
		Data struct {
			Class string `json:"class"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wCanonical.Body.Bytes(), &respCanonical); err != nil {
		t.Fatalf("Unmarshal canonical response failed: %v", err)
	}
	if respCanonical.Data.Class != "d4-ti-2024-a" {
		t.Errorf("Expected canonical class 'd4-ti-2024-a', got '%s'", respCanonical.Data.Class)
	}

	// 2. Uji slug legacy huruf kecil: d4-ti-smt1-a (pada semester aktif 2024/2025, SMT1 angkatan 2024)
	reqLegacy := httptest.NewRequest("GET", "/api/v1/portal/d4-ti-smt1-a/summary", nil)
	wLegacy := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wLegacy, reqLegacy)
	if wLegacy.Code != http.StatusOK {
		t.Fatalf("Legacy slug d4-ti-smt1-a expected 200, got %d, body: %s", wLegacy.Code, wLegacy.Body.String())
	}
	var respLegacy struct {
		Data struct {
			Class string `json:"class"`
		} `json:"data"`
	}
	if err := json.Unmarshal(wLegacy.Body.Bytes(), &respLegacy); err != nil {
		t.Fatalf("Unmarshal legacy response failed: %v", err)
	}
	if respLegacy.Data.Class != "d4-ti-2024-a" {
		t.Errorf("Expected resolved canonical class 'd4-ti-2024-a', got '%s'", respLegacy.Data.Class)
	}

	// 3. Uji endpoint portal lain dengan legacy slug: tasks, semesters, schedule
	endpoints := []string{
		"/api/v1/portal/d4-ti-smt1-a/tasks",
		"/api/v1/portal/d4-ti-smt1-a/semesters",
		"/api/v1/portal/d4-ti-smt1-a/schedule",
	}
	for _, ep := range endpoints {
		req := httptest.NewRequest("GET", ep, nil)
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("GET %s expected 200, got %d, body: %s", ep, w.Code, w.Body.String())
		}
	}

	// 4. Uji kelas fiktif yang tidak ada -> 404
	reqNotFound := httptest.NewRequest("GET", "/api/v1/portal/kelas-tak-terdaftar-999/summary", nil)
	wNotFound := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(wNotFound, reqNotFound)
	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("Unknown class expected 404, got %d", wNotFound.Code)
	}
}
