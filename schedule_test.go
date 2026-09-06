package main

import (
	"os"
	"strings"
	"testing"
	"time"
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

	// 7. Test GetDaftarMatkul
	matkulResult := cfg.GetDaftarMatkul()
	if !strings.Contains(matkulResult, "DAFTAR MATA KULIAH") ||
		!strings.Contains(matkulResult, "Aljabar Linear") {
		t.Errorf("GetDaftarMatkul invalid result, got: %s", matkulResult)
	}

	// 8. Test Reload
	reloadMsg, err := cfg.Reload()
	if err != nil || !strings.Contains(reloadMsg, "berhasil dimuat ulang") {
		t.Errorf("Reload failed, err: %v, msg: %s", err, reloadMsg)
	}

	// 9. Test GetNextClass berbagai kondisi waktu
	// 9a. Saat kuliah sedang berlangsung (Selasa 07:30)
	tSelasaOngoing := time.Date(2026, 9, 1, 7, 30, 0, 0, time.Local) // 1 Sep 2026 adalah Selasa
	resOngoing := cfg.GetNextClass(tSelasaOngoing)
	if !strings.Contains(resOngoing, "SEDANG BERLANGSUNG") || !strings.Contains(resOngoing, "Matematika Diskrit Lanjut") {
		t.Errorf("GetNextClass ongoing failed, got: %s", resOngoing)
	}

	// 9b. Sebelum kuliah pertama dimulai (Selasa 06:00)
	tSelasaPagi := time.Date(2026, 9, 1, 6, 0, 0, 0, time.Local)
	resPagi := cfg.GetNextClass(tSelasaPagi)
	if !strings.Contains(resPagi, "KULIAH BERIKUTNYA") || !strings.Contains(resPagi, "Matematika Diskrit Lanjut") {
		t.Errorf("GetNextClass upcoming failed, got: %s", resPagi)
	}

	// 9c. Setelah semua kuliah selesai (Selasa 16:00)
	tSelasaSore := time.Date(2026, 9, 1, 16, 0, 0, 0, time.Local)
	resSore := cfg.GetNextClass(tSelasaSore)
	if !strings.Contains(resSore, "KULIAH HARI INI SELESAI") {
		t.Errorf("GetNextClass finished failed, got: %s", resSore)
	}

	// 9d. Hari Libur (Sabtu)
	tSabtu := time.Date(2026, 9, 5, 10, 0, 0, 0, time.Local) // 5 Sep 2026 adalah Sabtu
	resSabtu := cfg.GetNextClass(tSabtu)
	if !strings.Contains(resSabtu, "LIBUR") {
		t.Errorf("GetNextClass weekend failed, got: %s", resSabtu)
	}

	// 10. Test ProcessMessage - Pintasan Cepat & Perintah Baru
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
		{"!matkul", "DAFTAR MATA KULIAH"},
		{"matakuliah", "DAFTAR MATA KULIAH"},
		{"!reload", "berhasil dimuat ulang"},
		{"!dosen MR", "Muhammad Rizqi Sholahuddin"},
		{"dosen MR", "Muhammad Rizqi Sholahuddin"},
		{"!ruang D105", "D105-Kelas"},
		{"!cari sistem", "Sistem"},
		{"!menu", "JADWAL KULIAH"},
		{"menu", "JADWAL KULIAH"},
		{"!keyword", "DAFTAR LENGKAP KEYWORD"},
		{"keyword", "DAFTAR LENGKAP KEYWORD"},
		{"!help", "DAFTAR LENGKAP KEYWORD"},
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

	// 10b. Test Strategi Hybrid: Di grup wajib ada prefix (!), di DM bebas
	// Di Grup Chat (isGroup = true):
	if res := cfg.ProcessMessage("!senin", true); !strings.Contains(res, "Aljabar Linear") {
		t.Errorf("Expected '!senin' in group to be processed, got: %s", res)
	}
	if res := cfg.ProcessMessage("senin", true); res != "" {
		t.Errorf("Expected 'senin' without prefix in group to be IGNORED, got: %s", res)
	}
	if res := cfg.ProcessMessage("besok", true); res != "" {
		t.Errorf("Expected 'besok' without prefix in group to be IGNORED, got: %s", res)
	}
	if res := cfg.ProcessMessage("matkul", true); res != "" {
		t.Errorf("Expected 'matkul' without prefix in group to be IGNORED, got: %s", res)
	}
	if res := cfg.ProcessMessage("!matkul", true); !strings.Contains(res, "DAFTAR MATA KULIAH") {
		t.Errorf("Expected '!matkul' with prefix in group to be processed, got: %s", res)
	}

	// 11. Test ReminderManager
	rmFile := "test_reminder_groups.json"
	defer os.Remove(rmFile)
	rm := LoadReminderManager(rmFile)

	ok, addMsg := rm.AddGroup("12345@g.us", "Grup Kelas Test")
	if !ok || !strings.Contains(addMsg, "Pengingat Otomatis Diaktifkan") {
		t.Errorf("AddGroup failed, got: %s", addMsg)
	}

	statusMsg := rm.Status("12345@g.us")
	if !strings.Contains(statusMsg, "Aktif di chat/grup ini") {
		t.Errorf("Expected status to be active, got: %s", statusMsg)
	}

	ok, rmMsg := rm.RemoveGroup("12345@g.us")
	if !ok || !strings.Contains(rmMsg, "Dinonaktifkan") {
		t.Errorf("RemoveGroup failed, got: %s", rmMsg)
	}
}

func TestSmartUpcomingSchedule(t *testing.T) {
	cfg, err := LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal load jadwal.json: %v", err)
	}

	// 1. Uji akhir pekan (Sabtu): Harusnya otomatis menampilkan hari Senin dengan catatan libur akhir pekan
	tSabtu := time.Date(2026, 9, 5, 14, 0, 0, 0, time.Local) // 5 Sep 2026 adalah Sabtu
	resSabtu := cfg.ProcessMessage("!jadwal", tSabtu)
	if !strings.Contains(resSabtu, "libur akhir pekan") {
		t.Errorf("Expected note 'libur akhir pekan' on Saturday, got:\n%s", resSabtu)
	}
	if !strings.Contains(resSabtu, "Senin") || !strings.Contains(resSabtu, "Aljabar Linear") {
		t.Errorf("Expected Monday schedule on Saturday, got:\n%s", resSabtu)
	}

	// 2. Uji akhir pekan (Minggu): Harusnya otomatis menampilkan hari Senin
	tMinggu := time.Date(2026, 9, 6, 10, 0, 0, 0, time.Local) // 6 Sep 2026 adalah Minggu
	resMinggu := cfg.ProcessMessage("!jadwal", tMinggu)
	if !strings.Contains(resMinggu, "libur akhir pekan") || !strings.Contains(resMinggu, "Aljabar Linear") {
		t.Errorf("Expected Monday schedule on Sunday, got:\n%s", resMinggu)
	}

	// 3. Uji hari aktif sebelum/saat jam kuliah (Senin 08:00 WIB):
	// Harus menampilkan hari Senin langsung tanpa catatan "telah selesai"
	tSeninPagi := time.Date(2026, 9, 7, 8, 0, 0, 0, time.Local) // 7 Sep 2026 adalah Senin
	resSeninPagi := cfg.ProcessMessage("!jadwal", tSeninPagi)
	if strings.Contains(resSeninPagi, "telah selesai") {
		t.Errorf("Expected ongoing Monday to NOT have 'telah selesai' note, got:\n%s", resSeninPagi)
	}
	if !strings.Contains(resSeninPagi, "Aljabar Linear") {
		t.Errorf("Expected Monday schedule on Monday morning, got:\n%s", resSeninPagi)
	}

	// 4. Uji hari aktif setelah jam kuliah selesai (Senin 16:00 WIB, matkul terakhir Senin selesai 14:40):
	// Harus menampilkan hari Selasa dengan catatan "Perkuliahan hari ini telah selesai"
	tSeninSore := time.Date(2026, 9, 7, 16, 0, 0, 0, time.Local)
	resSeninSore := cfg.ProcessMessage("!jadwal", tSeninSore)
	if !strings.Contains(resSeninSore, "Perkuliahan hari ini telah selesai") || !strings.Contains(resSeninSore, "Selasa") {
		t.Errorf("Expected Tuesday schedule after Monday classes ended, got:\n%s", resSeninSore)
	}
	if !strings.Contains(resSeninSore, "Matematika Diskrit Lanjut") {
		t.Errorf("Expected Tuesday course on Monday evening, got:\n%s", resSeninSore)
	}

	// 5. Uji hari Jumat sore setelah perkuliahan selesai (matkul terakhir Jumat selesai 18:10):
	// Melewati Sabtu dan Minggu, langsung melompat ke Senin pekan depan
	tJumatSore := time.Date(2026, 9, 11, 19, 0, 0, 0, time.Local) // 11 Sep 2026 adalah Jumat
	resJumatSore := cfg.ProcessMessage("!jadwal", tJumatSore)
	if !strings.Contains(resJumatSore, "Perkuliahan hari ini telah selesai") {
		t.Errorf("Expected 'Perkuliahan hari ini telah selesai' note on Friday evening, got:\n%s", resJumatSore)
	}
	if !strings.Contains(resJumatSore, "Senin") || !strings.Contains(resJumatSore, "Aljabar Linear") {
		t.Errorf("Expected Monday schedule after Friday classes ended, got:\n%s", resJumatSore)
	}

	// 6. Uji pembanding perintah literal "!hari ini":
	// Di Senin sore (!hari ini) tetap harus menampilkan jadwal Senin secara penuh
	resHariIni := cfg.ProcessMessage("!hari ini", tSeninSore)
	if !strings.Contains(resHariIni, "Aljabar Linear") || strings.Contains(resHariIni, "Matematika Diskrit Lanjut") {
		t.Errorf("Expected '!hari ini' to strictly return Monday schedule even in evening, got:\n%s", resHariIni)
	}

	// 7. Uji dengan OverrideManager: Libur di hari berikutnya
	db, err := InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB memory: %v", err)
	}
	defer db.Close()

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi OverrideManager: %v", err)
	}
	cfg.SetOverrideManager(om)

	groupJID := "test_group_smart@g.us"
	userJID := "test_admin@s.whatsapp.net"

	// Set Selasa libur: "!libur besok | Libur Kuliah Lapangan" (saat tSeninSore)
	om.HandleCommand(groupJID, true, userJID, true, "!libur besok | Libur Kuliah Lapangan", cfg, tSeninSore)

	// Saat Senin sore memanggil !jadwal di grup, karena Selasa libur, harus otomatis loncat ke Rabu
	resOverride := cfg.ProcessMessage("!jadwal", true, groupJID, tSeninSore)
	if !strings.Contains(resOverride, "Perkuliahan hari ini telah selesai") {
		t.Errorf("Expected completed note, got:\n%s", resOverride)
	}
	if !strings.Contains(resOverride, "Rabu") {
		t.Errorf("Expected to jump to Wednesday because Tuesday is a holiday, got:\n%s", resOverride)
	}
}
