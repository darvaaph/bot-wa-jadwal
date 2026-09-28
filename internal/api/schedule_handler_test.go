package api

import (
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/util"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()
	cm, err := schedule.NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal memuat ClassManager untuk pengujian: %v", err)
	}
	return NewServer(":8080", nil, cm, nil)
}

func TestHandleClasses_Success(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/classes", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200, didapat: %d", rr.Code)
	}

	var resp ClassesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Ekspektasi status 'success', didapat: '%s'", resp.Status)
	}

	if resp.Data.DefaultClass == "" {
		t.Error("Ekspektasi default_class tidak kosong")
	}

	if resp.Data.TotalClasses == 0 {
		t.Error("Ekspektasi total_classes > 0")
	}

	if len(resp.Data.Classes) != resp.Data.TotalClasses {
		t.Errorf("Panjang slice classes (%d) tidak cocok dengan total_classes (%d)", len(resp.Data.Classes), resp.Data.TotalClasses)
	}
}

func TestHandleClasses_NilClassManager(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/classes", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200, didapat: %d", rr.Code)
	}

	var resp ClassesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Ekspektasi status 'success', didapat: '%s'", resp.Status)
	}

	if resp.Data.TotalClasses != 0 || len(resp.Data.Classes) != 0 {
		t.Errorf("Ekspektasi classes kosong saat classManager nil, didapat: %v", resp.Data.Classes)
	}
}

func TestHandleSchedule_DefaultClass_FullWeek(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/schedule", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200, didapat: %d", rr.Code)
	}

	var resp ScheduleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Ekspektasi status 'success', didapat: '%s'", resp.Status)
	}

	expectedDefault := server.classManager.GetDefaultClassID()
	if resp.Class != expectedDefault {
		t.Errorf("Ekspektasi class '%s', didapat: '%s'", expectedDefault, resp.Class)
	}

	if resp.Day != "all" {
		t.Errorf("Ekspektasi day 'all' untuk jadwal seminggu, didapat: '%s'", resp.Day)
	}

	if len(resp.Data) == 0 {
		t.Error("Ekspektasi data jadwal tidak kosong untuk kelas default")
	}

	first := resp.Data[0]
	if first.Hari == "" || first.Jam == "" || first.Matkul == "" || first.Ruang == "" {
		t.Errorf("Field pada item jadwal belum lengkap: %+v", first)
	}
}

func TestHandleSchedule_FilterDay(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/schedule?class=D4-TI-SMT3-A&day=Senin", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200, didapat: %d", rr.Code)
	}

	var resp ScheduleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Ekspektasi status 'success', didapat: '%s'", resp.Status)
	}

	if resp.Class != "D4-TI-SMT3-A" {
		t.Errorf("Ekspektasi class 'D4-TI-SMT3-A', didapat: '%s'", resp.Class)
	}

	if resp.Day != "Senin" {
		t.Errorf("Ekspektasi day 'Senin', didapat: '%s'", resp.Day)
	}

	for _, item := range resp.Data {
		if !strings.EqualFold(item.Hari, "Senin") {
			t.Errorf("Ditemukan item di luar hari Senin: %+v", item)
		}
	}
}

func TestHandleSchedule_FilterToday(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/schedule?class=D4-TI-SMT3-A&day=today", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200, didapat: %d", rr.Code)
	}

	var resp ScheduleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	expectedDay := util.GetHariIndonesia(time.Now())
	if resp.Day != expectedDay {
		t.Errorf("Ekspektasi day '%s', didapat: '%s'", expectedDay, resp.Day)
	}

	for _, item := range resp.Data {
		if !strings.EqualFold(item.Hari, expectedDay) {
			t.Errorf("Ditemukan item di luar hari %s: %+v", expectedDay, item)
		}
	}
}

func TestHandleSchedule_AliasClass(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/schedule?class=3A&day=Senin", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Ekspektasi HTTP 200 untuk alias kelas, didapat: %d", rr.Code)
	}

	var resp ScheduleResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Gagal decode JSON respons: %v", err)
	}

	if resp.Class != "D4-TI-SMT3-A" {
		t.Errorf("Ekspektasi alias '3A' ter-resolve ke 'D4-TI-SMT3-A', didapat: '%s'", resp.Class)
	}
}

func TestHandleSchedule_ClassNotFound_404(t *testing.T) {
	server := setupTestServer(t)

	req := httptest.NewRequest("GET", "/api/schedule?class=KELAS_TIDAK_ADA_999", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("Ekspektasi HTTP 404 untuk kelas tidak dikenal, didapat: %d", rr.Code)
	}

	var errResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Gagal decode JSON respons error: %v", err)
	}

	if errResp["status"] != "error" {
		t.Errorf("Ekspektasi status 'error', didapat: '%s'", errResp["status"])
	}

	if errResp["message"] != "Kelas tidak ditemukan" {
		t.Errorf("Ekspektasi pesan 'Kelas tidak ditemukan', didapat: '%s'", errResp["message"])
	}
}

func TestHandleSchedule_NilClassManager_404(t *testing.T) {
	server := NewServer(":8080", nil, nil, nil)

	req := httptest.NewRequest("GET", "/api/schedule?class=D4-TI-SMT3-A", nil)
	rr := httptest.NewRecorder()

	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("Ekspektasi HTTP 404 saat classManager nil, didapat: %d", rr.Code)
	}

	var errResp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&errResp); err != nil {
		t.Fatalf("Gagal decode JSON respons error: %v", err)
	}

	if errResp["status"] != "error" {
		t.Errorf("Ekspektasi status 'error', didapat: '%s'", errResp["status"])
	}

	if errResp["message"] != "Kelas tidak ditemukan" {
		t.Errorf("Ekspektasi pesan 'Kelas tidak ditemukan', didapat: '%s'", errResp["message"])
	}
}
