package main

import (
	"strings"
	"testing"
	"time"
)

func TestReminder_MultiClassResolution(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB in-memory: %v", err)
	}
	defer db.Close()

	settingsMgr, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi ChatSettingsManager: %v", err)
	}

	classMgr, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	groupA := "120363001@g.us"
	groupB := "120363002@g.us"
	groupDefault := "120363003@g.us"

	// Tautkan grup A ke 3A, grup B ke 3B, grup default tidak ditautkan
	if err := settingsMgr.SetClass(groupA, "3A"); err != nil {
		t.Fatalf("Gagal set class groupA: %v", err)
	}
	if err := settingsMgr.SetClass(groupB, "3B"); err != nil {
		t.Fatalf("Gagal set class groupB: %v", err)
	}

	// Buat waktu referensi hari Senin pukul 06:30 WIB (2026-09-07)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	seninPagi := time.Date(2026, 9, 7, 6, 30, 0, 0, loc)

	// Resolusi untuk Grup A
	cfgA := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupA))
	msgA := BuildMorningReminder(groupA, cfgA, nil, seninPagi)

	// Resolusi untuk Grup B
	cfgB := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupB))
	msgB := BuildMorningReminder(groupB, cfgB, nil, seninPagi)

	// Resolusi untuk Grup Default
	cfgDef := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupDefault))
	msgDef := BuildMorningReminder(groupDefault, cfgDef, nil, seninPagi)

	// Verifikasi bahwa pesan pengingat Grup A berisi jadwal 3A (misal: Aljabar Linear)
	if !strings.Contains(msgA, "Aljabar Linear") {
		t.Errorf("Pengingat pagi Grup A harus berisi Aljabar Linear (Senin 3A): %s", msgA)
	}

	// Verifikasi bahwa pesan pengingat Grup B berisi jadwal 3B (misal: Sistem Operasi)
	if !strings.Contains(msgB, "Sistem Operasi") {
		t.Errorf("Pengingat pagi Grup B harus berisi Sistem Operasi (Senin 3B): %s", msgB)
	}

	// Verifikasi bahwa grup default menggunakan kelas 3A
	if !strings.Contains(msgDef, "Aljabar Linear") {
		t.Errorf("Pengingat pagi Grup Default harus menggunakan kelas default 3A: %s", msgDef)
	}
}

func TestBuildMorningReminder_WithMeetingLinks(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB in-memory: %v", err)
	}
	defer db.Close()

	classMgr, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	linkMgr, err := NewLinkManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi LinkManager: %v", err)
	}

	groupJID := "120363009@g.us"
	cfg := classMgr.GetDefaultClass()

	loc, _ := time.LoadLocation("Asia/Jakarta")
	seninPagi := time.Date(2026, 9, 7, 6, 30, 0, 0, loc)

	// Tambahkan tautan Google Meet dan Google Drive
	_, err = linkMgr.AddLink(groupJID, true, "Zoom Kuliah Daring SBD", "https://zoom.us/j/999111222", "ID Passcode 1234", "admin@s.whatsapp.net")
	if err != nil {
		t.Fatalf("Gagal tambah zoom link: %v", err)
	}
	_, err = linkMgr.AddLink(groupJID, true, "Drive Modul Kuliah", "https://drive.google.com/drive/folders/xyz", "Modul lengkap", "admin@s.whatsapp.net")
	if err != nil {
		t.Fatalf("Gagal tambah drive link: %v", err)
	}

	// Bangun pengingat pagi
	msg := BuildMorningReminder(groupJID, cfg, nil, seninPagi, linkMgr)

	// Verifikasi bagian tautan daring muncul dan berisi zoom link
	if !strings.Contains(msg, "TAUTAN KULIAH DARING HARI INI:") {
		t.Errorf("Harus menampilkan header tautan daring: %s", msg)
	}
	if !strings.Contains(msg, "Zoom Kuliah Daring SBD") || !strings.Contains(msg, "https://zoom.us/j/999111222") {
		t.Errorf("Harus memuat zoom link yang aktif: %s", msg)
	}
	// Drive tidak boleh masuk ke section meeting hari ini
	if strings.Contains(msg, "Drive Modul Kuliah") {
		t.Errorf("Tautan drive tidak boleh masuk ke section kuliah daring: %s", msg)
	}
}

