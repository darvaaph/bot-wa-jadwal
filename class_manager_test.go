package main

import (
	"strings"
	"testing"
)

func TestClassManager_LoadDirectory(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal memuat ClassManager dari data/jadwal: %v", err)
	}

	classes := cm.ListClasses()
	if len(classes) < 19 {
		t.Fatalf("Ekspektasi minimal 19 kelas resmi, ditemukan: %d kelas (%v)", len(classes), classes)
	}

	found3A := false
	found3B := false
	for _, c := range classes {
		if c == "D4-TI-SMT3-A" {
			found3A = true
		}
		if c == "D4-TI-SMT3-B" {
			found3B = true
		}
	}

	if !found3A || !found3B {
		t.Errorf("Kelas D4-TI-SMT3-A atau D4-TI-SMT3-B tidak ditemukan di daftar kelas: %v", classes)
	}
}

func TestClassManager_GetClass(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	// 1. Tes pencocokan nama kanonikal, alias case-insensitive, dan variasi input bebas
	testCases := []struct {
		input       string
		shouldExist bool
	}{
		{"D4-TI-SMT3-A", true},
		{"d4-ti-smt3-a", true},
		{"3A", true},
		{"3a", true},
		{"  3a  ", true},
		{"kelas 3A", true},
		{"KELAS 3a", true},
		{"kelas-3b", true},
		{"3B", true},
		{"D4-TI-3A", true},
		{"d4-ti-3a", true},
		{"smt 3 a", true},
		{"sem 3 a", true},
		{"semester 3 kelas a", true},
		{"d4 3 a", true},
		{"D3-TI-SMT1-A", true},
		{"d3-ti-1a", true},
		{"d3 1 a", true},
		{"d3-ti-smt5-c", true},
		{"2A", true},
		{"2a", true},
		{"kelas 2A", true},
		{"tingkat 2 a", true},
		{"d4-ti-2a", true},
		{"[`D4-TI-SMT3-A`]", true},
		{"`D4-TI-SMT3-A`", true},
		{"[D4-TI-SMT3-A]", true},
		{"*D4-TI-SMT3-A*", true},
		{"\"D4-TI-SMT3-A\"", true},
		{"99Z", false},
		{"KELAS_GHOIB", false},
	}

	for _, tc := range testCases {
		cfg, found := cm.GetClass(tc.input)
		if found != tc.shouldExist {
			t.Errorf("GetClass(%q): expected found=%v, got found=%v", tc.input, tc.shouldExist, found)
		}
		if found && cfg == nil {
			t.Errorf("GetClass(%q): config bernilai nil padahal ditemukan", tc.input)
		}
	}

	// 2. Pastikan jadwal 3A dan 3B benar-benar berbeda
	cfg3A, _ := cm.GetClass("3A")
	cfg3B, _ := cm.GetClass("3B")
	if !strings.Contains(cfg3A.Kampus, "3A") {
		t.Errorf("Jadwal 3A kampus mismatch: %s", cfg3A.Kampus)
	}
	if !strings.Contains(cfg3B.Kampus, "3B") {
		t.Errorf("Jadwal 3B kampus mismatch: %s", cfg3B.Kampus)
	}
}

func TestClassManager_DefaultClass(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	defaultID := cm.GetDefaultClassID()
	if defaultID != "D4-TI-SMT3-A" {
		t.Errorf("Expected default class D4-TI-SMT3-A, got: %s", defaultID)
	}

	defCfg := cm.GetDefaultClass()
	if defCfg == nil {
		t.Fatalf("Default class config tidak boleh nil")
	}

	// GetClassOrDefault dengan alias valid (3B)
	c3b := cm.GetClassOrDefault("3B")
	if !strings.Contains(c3b.Kampus, "3B") {
		t.Errorf("GetClassOrDefault(3B) gagal mengambil kelas 3B")
	}

	// GetClassOrDefault dengan ID kanonikal valid
	c3bKanonikal := cm.GetClassOrDefault("D4-TI-SMT3-B")
	if c3bKanonikal != c3b {
		t.Errorf("GetClassOrDefault(D4-TI-SMT3-B) harus sama dengan GetClassOrDefault(3B)")
	}

	// GetClassOrDefault dengan kelas tidak valid harus fallback ke default
	cInvalid := cm.GetClassOrDefault("KELAS_GHOIB")
	if cInvalid != defCfg {
		t.Errorf("GetClassOrDefault kelas invalid harus fallback ke default class")
	}
}

func TestClassManager_FallbackSingleFile(t *testing.T) {
	// Tes memuat direktori non-eksisten dengan fallback ke jadwal.json
	cm, err := NewClassManager("folder_tidak_ada_xyz", "jadwal.json")
	if err != nil {
		t.Fatalf("Harus sukses memuat via fallbackSingleFile: %v", err)
	}

	if !cm.HasClass("D4-TI-SMT3-A") || !cm.HasClass("3A") {
		t.Errorf("Fallback jadwal.json harus menyediakan kelas D4-TI-SMT3-A dan alias 3A")
	}
}

func TestClassManager_SetOverrideManager(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database in-memory: %v", err)
	}
	defer db.Close()

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi OverrideManager: %v", err)
	}

	cm.SetOverrideManager(om)

	cfg3A, _ := cm.GetClass("D4-TI-SMT3-A")
	if cfg3A.OverrideManager == nil {
		t.Errorf("OverrideManager harus terpasang di kelas D4-TI-SMT3-A")
	}

	cfg3B, _ := cm.GetClass("D4-TI-SMT3-B")
	if cfg3B.OverrideManager == nil {
		t.Errorf("OverrideManager harus terpasang di kelas D4-TI-SMT3-B")
	}
}

func TestClassManager_ReloadAll(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	count, errs := cm.ReloadAll()
	if len(errs) > 0 {
		t.Errorf("ReloadAll menghasilkan error: %v", errs)
	}
	if count < 19 {
		t.Errorf("Expected reload minimal 19 kelas, got: %d", count)
	}
}
