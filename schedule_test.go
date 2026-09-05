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

	// 1. Test GetByHari
	seninResult := cfg.GetByHari("senin")
	if !strings.Contains(seninResult, "Aljabar Linear") {
		t.Errorf("Expected 'Aljabar Linear' in Senin schedule, got: %s", seninResult)
	}

	// 2. Test SearchDosen yang ada jadwal
	dosenResult := cfg.SearchDosen("MR")
	if !strings.Contains(dosenResult, "Muhammad Rizqi Sholahuddin") {
		t.Errorf("Expected dosen Muhammad Rizqi Sholahuddin for MR, got: %s", dosenResult)
	}

	// 3. Test SearchDosen master JTK tanpa jadwal di kelas ini
	dosenMasterResult := cfg.SearchDosen("PH")
	if !strings.Contains(dosenMasterResult, "Priyanto Hidayatullah") {
		t.Errorf("Expected master dosen Priyanto for PH, got: %s", dosenMasterResult)
	}

	// 4. Test SearchRuangan
	ruangResult := cfg.SearchRuangan("Lab")
	if !strings.Contains(ruangResult, "Lab") {
		t.Errorf("Expected Lab in results, got: %s", ruangResult)
	}

	// 5. Test Menu
	menuResult := cfg.GetMenu()
	if !strings.Contains(menuResult, "!hari ini") || !strings.Contains(menuResult, "!seminggu") {
		t.Errorf("Expected menu to contain !hari ini and !seminggu, got: %s", menuResult)
	}

	// 6. Test GetJadwalSeminggu
	semingguResult := cfg.GetJadwalSeminggu()
	if !strings.Contains(semingguResult, "JADWAL SENIN - JUMAT") ||
		!strings.Contains(semingguResult, "SENIN") ||
		!strings.Contains(semingguResult, "JUMAT") {
		t.Errorf("GetJadwalSeminggu invalid result, got: %s", semingguResult)
	}

	// 7. Test ProcessMessage - Pintasan Cepat
	cases := []struct {
		input       string
		mustContain string
	}{
		{"!senin", "Aljabar Linear"},
		{"/senin", "Aljabar Linear"},
		{"senin", "Aljabar Linear"},
		{"!hari ini", "JADWAL"},
		{"hariini", "JADWAL"},
		{"!today", "JADWAL"},
		{"!besok", "JADWAL"},
		{"besok", "JADWAL"},
		{"!seminggu", "JADWAL SENIN - JUMAT"},
		{"seminggu", "JADWAL SENIN - JUMAT"},
		{"!senin-jumat", "JADWAL SENIN - JUMAT"},
		{"senin-jumat", "JADWAL SENIN - JUMAT"},
		{"!semua", "JADWAL SENIN - JUMAT"},
		{"!all", "JADWAL SENIN - JUMAT"},
		{"!jadwal seminggu", "JADWAL SENIN - JUMAT"},
		{"!jadwal", "JADWAL"},
		{"!jadwal selasa", "Matematika Diskrit Lanjut"},
		{"!dosen MR", "Muhammad Rizqi Sholahuddin"},
		{"dosen MR", "Muhammad Rizqi Sholahuddin"},
		{"!ruang D105", "D105-Kelas"},
		{"!cari sistem", "Sistem"},
		{"!menu", "Jadwal Kuliah"},
		{"menu", "Jadwal Kuliah"},
		{"!perintah_aneh", "tidak dikenali"},
	}

	for _, tc := range cases {
		res := cfg.ProcessMessage(tc.input)
		if !strings.Contains(res, tc.mustContain) {
			t.Errorf("ProcessMessage('%s') expected to contain '%s', got: '%s'", tc.input, tc.mustContain, res)
		}
	}

	// Test pesan non-perintah tidak boleh membalas (harus empty string)
	randomMsg := cfg.ProcessMessage("halo lagi ngapain?")
	if randomMsg != "" {
		t.Errorf("Expected empty reply for normal chat, got: %s", randomMsg)
	}
}
