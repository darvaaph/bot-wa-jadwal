package main

import (
	"strings"
	"testing"
)

func TestChatSettings_CRUD(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB in-memory: %v", err)
	}
	defer db.Close()

	mgr, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi ChatSettingsManager: %v", err)
	}

	groupA := "120363001@g.us"
	groupB := "120363002@g.us"

	// 1. Cek nilai awal (harus kosong / belum diatur)
	if class := mgr.GetClass(groupA); class != "" {
		t.Errorf("Expected empty class for groupA, got: %s", class)
	}

	// 2. Set kelas groupA ke 3A (dengan input santai "kelas 3a")
	if err := mgr.SetClass(groupA, "kelas 3a"); err != nil {
		t.Fatalf("Gagal SetClass: %v", err)
	}

	// 3. Verifikasi GetClass
	if class := mgr.GetClass(groupA); class != "3A" {
		t.Errorf("Expected 3A, got: %s", class)
	}

	// 4. Set kelas groupB ke 3B
	if err := mgr.SetClass(groupB, "3b"); err != nil {
		t.Fatalf("Gagal SetClass groupB: %v", err)
	}
	if class := mgr.GetClass(groupB); class != "3B" {
		t.Errorf("Expected 3B, got: %s", class)
	}

	// 5. Update kelas groupA menjadi TI-3C
	if err := mgr.SetClass(groupA, "TI-3C"); err != nil {
		t.Fatalf("Gagal update SetClass groupA: %v", err)
	}
	if class := mgr.GetClass(groupA); class != "TI-3C" {
		t.Errorf("Expected TI-3C after update, got: %s", class)
	}

	// 6. Validasi jumlah setelan
	if count := mgr.CountSettings(); count != 2 {
		t.Errorf("Expected count=2, got: %d", count)
	}

	// 7. DeleteClass
	if err := mgr.DeleteClass(groupA); err != nil {
		t.Fatalf("Gagal DeleteClass: %v", err)
	}
	if class := mgr.GetClass(groupA); class != "" {
		t.Errorf("Expected empty class after delete, got: %s", class)
	}
	if count := mgr.CountSettings(); count != 1 {
		t.Errorf("Expected count=1 after delete, got: %d", count)
	}
}

func TestChatSettings_CacheWarmup(t *testing.T) {
	// Gunakan database yang persisten pada instance DB
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB: %v", err)
	}
	defer db.Close()

	mgr1, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init mgr1: %v", err)
	}

	testGroup := "120363099@g.us"
	if err := mgr1.SetClass(testGroup, "3A"); err != nil {
		t.Fatalf("Gagal set class: %v", err)
	}

	// Buat instance manager baru pada database yang sama untuk mensimulasikan bot restart
	mgr2, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init mgr2: %v", err)
	}

	// Verifikasi bahwa cache langsung terisi dari database
	if class := mgr2.GetClass(testGroup); class != "3A" {
		t.Errorf("Expected warm cache class 3A, got: %s", class)
	}
}

func TestChatSettings_Validation(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal init DB: %v", err)
	}
	defer db.Close()

	mgr, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init mgr: %v", err)
	}

	// Coba set dengan string kosong
	if err := mgr.SetClass("group@g.us", "   "); err == nil {
		t.Errorf("SetClass dengan nama kelas kosong seharusnya menghasilkan error")
	}

	// Init dengan nil DB harus error
	if _, err := NewChatSettingsManager(nil); err == nil {
		t.Errorf("NewChatSettingsManager(nil) seharusnya menghasilkan error")
	}
}

func TestChatSettings_HandleCommand(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal init DB: %v", err)
	}
	defer db.Close()

	csm, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init CSM: %v", err)
	}

	classMgr, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal init ClassManager: %v", err)
	}

	groupJID := "120363001@g.us"
	userJID := "628123456789@s.whatsapp.net"

	// 1. Tes !daftarkelas awal
	daftarResp := csm.HandleCommand(groupJID, true, userJID, false, "!daftarkelas", classMgr)
	if !strings.Contains(daftarResp, "DAFTAR KELAS PERKULIAHAN") || !strings.Contains(daftarResp, "3A") {
		t.Errorf("Respon !daftarkelas tidak valid: %s", daftarResp)
	}

	// 2. Tes !setkelas tanpa argumen
	helpResp := csm.HandleCommand(groupJID, true, userJID, true, "!setkelas", classMgr)
	if !strings.Contains(helpResp, "Panduan Penggunaan !setkelas") {
		t.Errorf("Expected panduan !setkelas, got: %s", helpResp)
	}

	// 3. Tes !setkelas oleh non-admin di grup -> harus ditolak
	nonAdminResp := csm.HandleCommand(groupJID, true, userJID, false, "!setkelas 3B", classMgr)
	if !strings.Contains(nonAdminResp, "AKSES DITOLAK") {
		t.Errorf("Non-admin di grup harus ditolak, got: %s", nonAdminResp)
	}

	// 4. Tes !setkelas oleh admin di grup dengan kelas yang tidak ada -> harus error
	invalidClassResp := csm.HandleCommand(groupJID, true, userJID, true, "!setkelas 99Z", classMgr)
	if !strings.Contains(invalidClassResp, "Tidak Ditemukan") {
		t.Errorf("Kelas tidak ada harus menghasilkan error, got: %s", invalidClassResp)
	}

	// 5. Tes !setkelas oleh admin di grup dengan kelas valid (3B) -> harus sukses
	adminSetResp := csm.HandleCommand(groupJID, true, userJID, true, "!setkelas 3B", classMgr)
	if !strings.Contains(adminSetResp, "BERHASIL DIATUR") || !strings.Contains(adminSetResp, "3B") {
		t.Errorf("Admin set kelas harus sukses, got: %s", adminSetResp)
	}
	if class := csm.GetClass(groupJID); class != "3B" {
		t.Errorf("Kelas aktif harus 3B, got: %s", class)
	}

	// 6. Tes !kelas setelah disetel ke 3B -> harus menampilkan 3B sebagai aktif
	kelasResp := csm.HandleCommand(groupJID, true, userJID, false, "!kelas", classMgr)
	if !strings.Contains(kelasResp, "3B") || !strings.Contains(kelasResp, "Aktif") {
		t.Errorf("Respon !kelas harus menunjukkan 3B aktif: %s", kelasResp)
	}

	// 7. Tes !setkelas di DM pribadi (isGroup = false, isAdmin = false) -> harus bisa set sendiri
	dmResp := csm.HandleCommand(userJID, false, userJID, false, "!setkelas 3A", classMgr)
	if !strings.Contains(dmResp, "BERHASIL DIATUR") || !strings.Contains(dmResp, "3A") {
		t.Errorf("User di DM harus bisa set kelas miliknya, got: %s", dmResp)
	}
	if class := csm.GetClass(userJID); class != "3A" {
		t.Errorf("Kelas DM harus 3A, got: %s", class)
	}

	// 8. Tes !resetkelas oleh non-admin di grup -> harus ditolak
	nonAdminReset := csm.HandleCommand(groupJID, true, userJID, false, "!resetkelas", classMgr)
	if !strings.Contains(nonAdminReset, "AKSES DITOLAK") {
		t.Errorf("Non-admin reset kelas harus ditolak, got: %s", nonAdminReset)
	}

	// 9. Tes !resetkelas oleh admin di grup -> harus sukses kembali ke default
	adminReset := csm.HandleCommand(groupJID, true, userJID, true, "!resetkelas", classMgr)
	if !strings.Contains(adminReset, "DIRESET") {
		t.Errorf("Admin reset kelas harus sukses, got: %s", adminReset)
	}
	if class := csm.GetClass(groupJID); class != "" {
		t.Errorf("Setelah reset kelas harus kosong (default), got: %s", class)
	}
}

