package main

import (
	"strings"
	"testing"
)

func TestSchedule(t *testing.T) {
	cfg, err := LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal load jadwal.json: %v", err)
	}

	if len(cfg.Jadwal) == 0 {
		t.Fatalf("Jadwal kosong")
	}

	// Test GetByHari
	seninResult := cfg.GetByHari("senin")
	if !strings.Contains(seninResult, "Matematika Diskrit Dasar") {
		t.Errorf("Expected 'Matematika Diskrit Dasar' in Senin schedule, got: %s", seninResult)
	}

	// Test SearchDosen yang ada jadwal
	dosenResult := cfg.SearchDosen("YD")
	if !strings.Contains(dosenResult, "Yudi Widhiyasana") {
		t.Errorf("Expected dosen Yudi Widhiyasana for YD, got: %s", dosenResult)
	}

	// Test SearchDosen master JTK tanpa jadwal di kelas ini
	dosenMasterResult := cfg.SearchDosen("PH")
	if !strings.Contains(dosenMasterResult, "Priyanto Hidayatullah") {
		t.Errorf("Expected master dosen Priyanto for PH, got: %s", dosenMasterResult)
	}

	// Test SearchRuangan
	ruangResult := cfg.SearchRuangan("Lab")
	if !strings.Contains(ruangResult, "Lab") {
		t.Errorf("Expected Lab in results, got: %s", ruangResult)
	}

	// Test Menu
	menuResult := cfg.GetMenu()
	if !strings.Contains(menuResult, "!jadwal") {
		t.Errorf("Expected menu to contain !jadwal, got: %s", menuResult)
	}
}
