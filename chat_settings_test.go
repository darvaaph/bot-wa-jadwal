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

	// 1. Tes !daftarkelas awal (status belum diatur)
	daftarResp := csm.HandleCommand(groupJID, true, userJID, false, "!daftarkelas", classMgr)
	if !strings.Contains(daftarResp, "DAFTAR KELAS PERKULIAHAN") || !strings.Contains(daftarResp, "Belum Diatur") {
		t.Errorf("Respon !daftarkelas awal harus menunjukkan status 'Belum Diatur': %s", daftarResp)
	}
	if strings.Contains(daftarResp, "(Aktif)") {
		t.Errorf("Respon !daftarkelas awal tidak boleh menandai kelas apa pun sebagai aktif: %s", daftarResp)
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

	// 5. Tes !setkelas oleh admin di grup dengan kelas valid (3B -> D4-TI-SMT3-B) -> harus sukses
	adminSetResp := csm.HandleCommand(groupJID, true, userJID, true, "!setkelas 3B", classMgr)
	if !strings.Contains(adminSetResp, "BERHASIL DIATUR") || !strings.Contains(adminSetResp, "D4-TI-SMT3-B") {
		t.Errorf("Admin set kelas harus sukses, got: %s", adminSetResp)
	}
	if class := csm.GetClass(groupJID); class != "D4-TI-SMT3-B" {
		t.Errorf("Kelas aktif harus D4-TI-SMT3-B, got: %s", class)
	}

	// 6. Tes !kelas setelah disetel ke 3B -> harus menampilkan D4-TI-SMT3-B sebagai aktif
	kelasResp := csm.HandleCommand(groupJID, true, userJID, false, "!kelas", classMgr)
	if !strings.Contains(kelasResp, "D4-TI-SMT3-B") || !strings.Contains(kelasResp, "Aktif") {
		t.Errorf("Respon !kelas harus menunjukkan D4-TI-SMT3-B aktif: %s", kelasResp)
	}

	// 7. Tes !setkelas di DM pribadi (isGroup = false, isAdmin = false) dengan input bebas "smt 3 a" -> harus sukses
	dmResp := csm.HandleCommand(userJID, false, userJID, false, "!setkelas smt 3 a", classMgr)
	if !strings.Contains(dmResp, "BERHASIL DIATUR") || !strings.Contains(dmResp, "D4-TI-SMT3-A") {
		t.Errorf("User di DM harus bisa set kelas miliknya, got: %s", dmResp)
	}
	if class := csm.GetClass(userJID); class != "D4-TI-SMT3-A" {
		t.Errorf("Kelas DM harus D4-TI-SMT3-A, got: %s", class)
	}

	// 8. Tes !resetkelas oleh non-admin di grup -> harus ditolak
	nonAdminReset := csm.HandleCommand(groupJID, true, userJID, false, "!resetkelas", classMgr)
	if !strings.Contains(nonAdminReset, "AKSES DITOLAK") {
		t.Errorf("Non-admin reset kelas harus ditolak, got: %s", nonAdminReset)
	}

	// 9. Tes !resetkelas oleh admin di grup -> harus sukses kembali ke status belum diatur
	adminReset := csm.HandleCommand(groupJID, true, userJID, true, "!resetkelas", classMgr)
	if !strings.Contains(adminReset, "DIRESET") || !strings.Contains(adminReset, "Belum Diatur") {
		t.Errorf("Admin reset kelas harus sukses dan menyebut Belum Diatur, got: %s", adminReset)
	}
	if class := csm.GetClass(groupJID); class != "" {
		t.Errorf("Setelah reset kelas harus kosong (unconfigured), got: %s", class)
	}

	// 10. Verifikasi status !daftarkelas setelah reset kembali menjadi 'Belum Diatur'
	postResetDaftar := csm.HandleCommand(groupJID, true, userJID, false, "!daftarkelas", classMgr)
	if !strings.Contains(postResetDaftar, "Belum Diatur") || strings.Contains(postResetDaftar, "(Aktif)") {
		t.Errorf("Setelah reset, !daftarkelas harus kembali ke status Belum Diatur: %s", postResetDaftar)
	}
}

func TestChatSettings_Onboarding(t *testing.T) {
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal init DB: %v", err)
	}
	defer db.Close()

	csm, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init CSM: %v", err)
	}

	// 1. Uji pesan onboarding untuk Grup
	groupPrompt := csm.GetOnboardingPrompt(true)
	if !strings.Contains(groupPrompt, "KELAS BELUM DIATUR") || !strings.Contains(groupPrompt, "Admin Grup") || !strings.Contains(groupPrompt, "!setkelas") {
		t.Errorf("Onboarding prompt grup tidak valid: %s", groupPrompt)
	}

	// 2. Uji pesan onboarding untuk DM Pribadi
	dmPrompt := csm.GetOnboardingPrompt(false)
	if !strings.Contains(dmPrompt, "KELAS BELUM DIATUR") || !strings.Contains(dmPrompt, "Chat pribadi") || !strings.Contains(dmPrompt, "!setkelas") {
		t.Errorf("Onboarding prompt DM tidak valid: %s", dmPrompt)
	}

	// 3. Uji menu belum terkonfigurasi untuk Grup
	groupMenu := csm.BuildUnconfiguredMenu(true)
	if !strings.Contains(groupMenu, "STATUS: KELAS BELUM DIATUR") || !strings.Contains(groupMenu, "Admin Grup") {
		t.Errorf("Unconfigured menu grup tidak valid: %s", groupMenu)
	}

	// 4. Uji menu belum terkonfigurasi untuk DM Pribadi
	dmMenu := csm.BuildUnconfiguredMenu(false)
	if !strings.Contains(dmMenu, "STATUS: KELAS BELUM DIATUR") || !strings.Contains(dmMenu, "chat pribadi") {
		t.Errorf("Unconfigured menu DM tidak valid: %s", dmMenu)
	}
}

func TestChatSettings_BackwardCompatibilityMigration(t *testing.T) {
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

	g1 := "120363001@g.us"
	g2 := "120363002@g.us"
	g3 := "120363003@g.us"
	g4 := "120363004@g.us"

	// 1. Simulasikan data lama di SQLite: "3A", "3B", "D4-TI-1A", "D3-TI-1A"
	_ = csm.SetClass(g1, "3A")
	_ = csm.SetClass(g2, "3B")
	_ = csm.SetClass(g3, "D4-TI-1A")
	_ = csm.SetClass(g4, "D3-TI-1A")

	// 2. Verifikasi sebelum migrasi: GetClassOrDefault tetap bekerja via alias cerdas tanpa error
	cfg1 := classMgr.GetClassOrDefault(csm.GetClass(g1))
	if cfg1 == nil || !strings.Contains(cfg1.Kampus, "3A") {
		t.Errorf("Sebelum migrasi, g1 (3A) harus tetap bisa mendapatkan jadwal 3A: %v", cfg1)
	}

	// 3. Jalankan penyelarasan otomatis (SyncWithClassManager)
	upgraded := csm.SyncWithClassManager(classMgr)
	if upgraded != 4 {
		t.Errorf("Harus meng-upgrade 4 setelan lama, tapi got: %d", upgraded)
	}

	// 4. Verifikasi setelah migrasi: seluruh record otomatis berubah menjadi ID kanonikal eksplisit
	if c1 := csm.GetClass(g1); c1 != "D4-TI-SMT3-A" {
		t.Errorf("g1 harus ter-upgrade ke D4-TI-SMT3-A, got: %s", c1)
	}
	if c2 := csm.GetClass(g2); c2 != "D4-TI-SMT3-B" {
		t.Errorf("g2 harus ter-upgrade ke D4-TI-SMT3-B, got: %s", c2)
	}
	if c3 := csm.GetClass(g3); c3 != "D4-TI-SMT1-A" {
		t.Errorf("g3 harus ter-upgrade ke D4-TI-SMT1-A, got: %s", c3)
	}
	if c4 := csm.GetClass(g4); c4 != "D3-TI-SMT1-A" {
		t.Errorf("g4 harus ter-upgrade ke D3-TI-SMT1-A, got: %s", c4)
	}

	// 5. Verifikasi bahwa perintah !kelas menampilkan status aktif yang benar
	resp1 := csm.HandleCommand(g1, true, "user@s.whatsapp.net", false, "!kelas", classMgr)
	if !strings.Contains(resp1, "D4-TI-SMT3-A") || !strings.Contains(resp1, "Aktif") {
		t.Errorf("Format kelas g1 harus menampilkan D4-TI-SMT3-A aktif: %s", resp1)
	}

	// 6. Verifikasi bahwa tidak ada chat yang perlu disetel ulang
	if csm.GetClass(g1) == "" {
		t.Errorf("Chat tidak boleh kembali ke unconfigured")
	}
}


