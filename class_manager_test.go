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
	if len(classes) < 2 {
		t.Fatalf("Ekspektasi minimal 2 kelas (3A, 3B), ditemukan: %v", classes)
	}

	found3A := false
	found3B := false
	for _, c := range classes {
		if c == "3A" {
			found3A = true
		}
		if c == "3B" {
			found3B = true
		}
	}

	if !found3A || !found3B {
		t.Errorf("Kelas 3A atau 3B tidak ditemukan di daftar kelas: %v", classes)
	}
}

func TestClassManager_GetClass(t *testing.T) {
	cm, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	// 1. Tes pencocokan case-insensitive dan variasi prefix
	testCases := []struct {
		input       string
		expectedID  string
		shouldExist bool
	}{
		{"3A", "3A", true},
		{"3a", "3A", true},
		{"  3a  ", "3A", true},
		{"kelas 3A", "3A", true},
		{"KELAS 3a", "3A", true},
		{"kelas-3b", "3B", true},
		{"3B", "3B", true},
		{"99Z", "", false},
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
	if defaultID != "3A" {
		t.Errorf("Expected default class 3A, got: %s", defaultID)
	}

	defCfg := cm.GetDefaultClass()
	if defCfg == nil {
		t.Fatalf("Default class config tidak boleh nil")
	}

	// GetClassOrDefault dengan kelas valid
	c3b := cm.GetClassOrDefault("3B")
	if !strings.Contains(c3b.Kampus, "3B") {
		t.Errorf("GetClassOrDefault(3B) gagal mengambil kelas 3B")
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

	if !cm.HasClass("3A") {
		t.Errorf("Fallback jadwal.json harus menyediakan kelas 3A")
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

	cfg3A, _ := cm.GetClass("3A")
	if cfg3A.OverrideManager == nil {
		t.Errorf("OverrideManager harus terpasang di kelas 3A")
	}

	cfg3B, _ := cm.GetClass("3B")
	if cfg3B.OverrideManager == nil {
		t.Errorf("OverrideManager harus terpasang di kelas 3B")
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
	if count < 2 {
		t.Errorf("Expected reload minimal 2 kelas, got: %d", count)
	}
}
