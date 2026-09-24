package task

import (
	"bot-jadwal/internal/schedule"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func openLegacyTaskTestDB(t *testing.T, path string) *sql.DB {
	t.Helper()

	// Pengujian ini mencakup adapter tugas pra-v3, bukan schema akademik target.
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Gagal membuka database SQLite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestTaskManager(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "test_tugas.db")

	db := openLegacyTaskTestDB(t, testDB)

	tm, err := NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewTaskManager: %v", err)
	}

	cfg, err := schedule.LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal memuat jadwal.json: %v", err)
	}

	// Gunakan waktu acuan tetap: Rabu, 9 September 2026 pukul 10:00 WIB
	refNow := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)

	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"
	otherUserJID := "628120002@s.whatsapp.net"

	nonAdminReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas tambah SBD | Laporan 1 | Jumat 23:59", cfg, refNow)
	if !strings.Contains(nonAdminReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected in group, got: %s", nonAdminReply)
	}

	adminReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Jumat 23:59", cfg, refNow)
	if !strings.Contains(adminReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(adminReply, "SISTEM BASIS DATA") {
		t.Errorf("Expected admin to succeed in group with normalized matkul, got: %s", adminReply)
	}

	dupReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Sabtu 12:00", cfg, refNow)
	if !strings.Contains(dupReply, "Tugas Serupa Sudah Terdaftar") {
		t.Errorf("Expected duplicate task to be rejected, got: %s", dupReply)
	}

	badFormatReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Tugas Tanpa Pipa", cfg, refNow)
	if !strings.Contains(badFormatReply, "Format Penambahan Tugas Kurang Tepat") {
		t.Errorf("Expected format validation error, got: %s", badFormatReply)
	}

	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Aljabar | Kuis Hari Ini | hari ini 23:59", cfg, refNow)
	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Matdis | PR Logika | besok 14:00", cfg, refNow)

	listReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas", cfg, refNow)
	if !strings.Contains(listReply, "DAFTAR TUGAS KELAS") ||
		!strings.Contains(listReply, "SISTEM BASIS DATA") ||
		!strings.Contains(listReply, "ALJABAR LINEAR") ||
		!strings.Contains(listReply, "DEADLINE HARI INI") {
		t.Errorf("Expected group task list with urgency badges, got: %s", listReply)
	}

	todayReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas hari ini", cfg, refNow)
	if !strings.Contains(todayReply, "DEADLINE HARI INI") || !strings.Contains(todayReply, "ALJABAR LINEAR") {
		t.Errorf("Expected today's task to be Aljabar, got: %s", todayReply)
	}

	tomorrowReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas besok", cfg, refNow)
	if !strings.Contains(tomorrowReply, "DEADLINE BESOK") || !strings.Contains(tomorrowReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected tomorrow's task to be Matdis, got: %s", tomorrowReply)
	}

	dmListReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas", cfg, refNow)
	if !strings.Contains(dmListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected empty tasks in fresh personal DM, got: %s", dmListReply)
	}

	userReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas tambah Pribadi | Belajar Golang | Minggu 20:00", cfg, refNow)
	if !strings.Contains(userReply, "BERHASIL DITAMBAHKAN") {
		t.Errorf("Expected user to add personal task in DM, got: %s", userReply)
	}

	// Tugas user 1 tidak boleh bocor ke user 2
	user2ListReply := tm.HandleCommand(otherUserJID, false, otherUserJID, true, "!tugas", cfg, refNow)
	if !strings.Contains(user2ListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected user2 tasks to be empty, got: %s", user2ListReply)
	}

	nonAdminDone := tm.HandleCommand(groupJID, true, userJID, false, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(nonAdminDone, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected completing task in group, got: %s", nonAdminDone)
	}

	adminDone := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(adminDone, "TUGAS SELESAI") {
		t.Errorf("Expected admin to complete task 1, got: %s", adminDone)
	}

	helpReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas bantuan", cfg, refNow)
	if !strings.Contains(helpReply, "PANDUAN DEADLINE TRACKER TUGAS") {
		t.Errorf("Expected help guide for tasks, got: %s", helpReply)
	}

	// Acuan: Sabtu, 5 September 2026 pukul 20:00 WIB
	tSabtu := time.Date(2026, 9, 5, 20, 0, 0, 0, time.Local)
	targetToday, labelToday := parseDeadline("5 sep 22.15", tSabtu)
	if targetToday.Day() != 5 || targetToday.Month() != 9 || targetToday.Hour() != 22 || targetToday.Minute() != 15 {
		t.Errorf("Expected 5 Sep 22:15, got: %v", targetToday)
	}
	if !strings.Contains(labelToday, "5 Sep 22:15 WIB") {
		t.Errorf("Expected label '5 Sep 22:15 WIB', got: %s", labelToday)
	}

	badgeToday := GetUrgencyBadge(targetToday, tSabtu)
	if !strings.Contains(badgeToday, "DEADLINE HARI INI") {
		t.Errorf("Expected DEADLINE HARI INI for '5 sep 22.15', got: %s", badgeToday)
	}

	target8Sep, label8Sep := parseDeadline("8 september", tSabtu)
	if target8Sep.Day() != 8 || target8Sep.Month() != 9 || target8Sep.Hour() != 23 || target8Sep.Minute() != 59 {
		t.Errorf("Expected 8 Sep 23:59, got: %v", target8Sep)
	}
	if !strings.Contains(label8Sep, "8 Sep 23:59 WIB") {
		t.Errorf("Expected label '8 Sep 23:59 WIB', got: %s", label8Sep)
	}

	badge8Sep := GetUrgencyBadge(target8Sep, tSabtu)
	if !strings.Contains(badge8Sep, "H-3") {
		t.Errorf("Expected H-3 for '8 september' from 5 Sep, got: %s", badge8Sep)
	}

	targetJamSaja, labelJamSaja := parseDeadline("22.22", tSabtu)
	if targetJamSaja.Day() != 5 || targetJamSaja.Month() != 9 || targetJamSaja.Hour() != 22 || targetJamSaja.Minute() != 22 {
		t.Errorf("Expected 5 Sep 22:22, got: %v", targetJamSaja)
	}
	if !strings.Contains(labelJamSaja, "Hari Ini (Sabtu), 22:22 WIB") {
		t.Errorf("Expected 'Hari Ini (Sabtu), 22:22 WIB', got: %s", labelJamSaja)
	}

	badMatkulReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Kalkulus | Latihan 1 | 22.22", cfg, tSabtu)
	if !strings.Contains(badMatkulReply, "Tidak Terdaftar") || !strings.Contains(badMatkulReply, "Daftar Mata Kuliah Kelas") {
		t.Errorf("Expected unlisted matkul error with course guide, got: %s", badMatkulReply)
	}

	mtkReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah mtk | Tugas Graph | 22.22", cfg, tSabtu)
	if !strings.Contains(mtkReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(mtkReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected 'mtk' to normalize to 'Matematika Diskrit Lanjut', got: %s", mtkReply)
	}

	matematikaReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah matematika | Tugas Tree | besok 10:00", cfg, tSabtu)
	if !strings.Contains(matematikaReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(matematikaReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected 'matematika' to normalize to 'Matematika Diskrit Lanjut', got: %s", matematikaReply)
	}

	umumReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah umum | Bawa Perlengkapan Lab | 22.22", cfg, tSabtu)
	if !strings.Contains(umumReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(umumReply, "UMUM") {
		t.Errorf("Expected 'umum' task to succeed, got: %s", umumReply)
	}

	nonAdminEdit := tm.HandleCommand(groupJID, true, userJID, false, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(nonAdminEdit, "Akses Ditolak") {
		t.Errorf("Expected non-admin edit to be rejected in group, got: %s", nonAdminEdit)
	}

	adminEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(adminEdit, "BERHASIL DIPERBARUI") || !strings.Contains(adminEdit, "Minggu") {
		t.Errorf("Expected admin edit deadline to succeed, got: %s", adminEdit)
	}

	adminEditBoth := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Revisi Lapres 1 | Senin 12:00", cfg, tSabtu)
	if !strings.Contains(adminEditBoth, "BERHASIL DIPERBARUI") || !strings.Contains(adminEditBoth, "Revisi Lapres 1") {
		t.Errorf("Expected admin edit both desc and deadline to succeed, got: %s", adminEditBoth)
	}

	badIDEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 999 | besok", cfg, tSabtu)
	if !strings.Contains(badIDEdit, "tidak ditemukan") {
		t.Errorf("Expected non-existent task to report not found, got: %s", badIDEdit)
	}

	filterSBD := tm.HandleCommand(groupJID, true, userJID, false, "!tugas sbd", cfg, tSabtu)
	if !strings.Contains(filterSBD, "SISTEM BASIS DATA") || strings.Contains(filterSBD, "ALJABAR LINEAR") {
		t.Errorf("Expected filter SBD to only show SBD tasks, got:\n%s", filterSBD)
	}

	filterAljabar := tm.HandleCommand(groupJID, true, userJID, false, "!tugas aljabar", cfg, tSabtu)
	if !strings.Contains(filterAljabar, "ALJABAR LINEAR") || strings.Contains(filterAljabar, "SISTEM BASIS DATA") {
		t.Errorf("Expected filter Aljabar to only show Aljabar tasks, got:\n%s", filterAljabar)
	}

	filterSO := tm.HandleCommand(groupJID, true, userJID, false, "!tugas so", cfg, tSabtu)
	if !strings.Contains(filterSO, "Tidak ada tugas aktif") || !strings.Contains(filterSO, "SISTEM OPERASI") {
		t.Errorf("Expected empty message for SO filter, got:\n%s", filterSO)
	}

	filterMatkulSBD := tm.HandleCommand(groupJID, true, userJID, false, "!tugas matkul sbd", cfg, tSabtu)
	if !strings.Contains(filterMatkulSBD, "SISTEM BASIS DATA") {
		t.Errorf("Expected !tugas matkul sbd to work, got:\n%s", filterMatkulSBD)
	}

	riwayatResp := tm.HandleCommand(groupJID, true, userJID, false, "!tugas riwayat", cfg, tSabtu)
	if !strings.Contains(riwayatResp, "ARSIP & RIWAYAT TUGAS SELESAI") || !strings.Contains(riwayatResp, "#1") || !strings.Contains(riwayatResp, "✅") {
		t.Errorf("Expected task #1 in riwayat, got: %s", riwayatResp)
	}

	arsipResp := tm.HandleCommand(groupJID, true, userJID, false, "!tugas arsip", cfg, tSabtu)
	if !strings.Contains(arsipResp, "ARSIP & RIWAYAT TUGAS SELESAI") || !strings.Contains(arsipResp, "#1") {
		t.Errorf("Expected task #1 in arsip, got: %s", arsipResp)
	}

	doneResp2 := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 2", cfg, tSabtu)
	if !strings.Contains(doneResp2, "TUGAS SELESAI") || !strings.Contains(doneResp2, "#2") {
		t.Errorf("Expected task #2 to be completed, got: %s", doneResp2)
	}

	riwayatResp2 := tm.HandleCommand(groupJID, true, userJID, false, "!tugas riwayat", cfg, tSabtu)
	if !strings.Contains(riwayatResp2, "#2") || !strings.Contains(riwayatResp2, "#1") {
		t.Errorf("Expected both task #1 and #2 in riwayat, got: %s", riwayatResp2)
	}
}

func TestTaskTeoriPraktikumDisambiguation(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "test_teori_prak.db")

	db := openLegacyTaskTestDB(t, testDB)

	tm, err := NewTaskManager(db)
	if err != nil {
		t.Fatalf("NewTaskManager error: %v", err)
	}

	cfg, err := schedule.LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("LoadJadwal error: %v", err)
	}

	refNow := time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local)
	groupJID := "test_group@g.us"
	userJID := "628111111@s.whatsapp.net"

	// Input tanpa penanda teori atau praktikum harus tetap ambigu.
	ambiguResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah alin | Pertemuan-1 | besok 8:40", cfg, refNow)
	if !strings.Contains(ambiguResp, "Sesi Belum Spesifik") ||
		!strings.Contains(ambiguResp, "Praktikum") ||
		!strings.Contains(ambiguResp, "Teori") ||
		!strings.Contains(ambiguResp, "Muhammad Rizqi") ||
		!strings.Contains(ambiguResp, "Nurjannah") {
		t.Errorf("Expected ambiguity warning with lecturer details, got:\n%s", ambiguResp)
	}

	prakResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah alin praktikum | Pertemuan-1 | besok 8:40", cfg, refNow)
	if !strings.Contains(prakResp, "BERHASIL DITAMBAHKAN") ||
		!strings.Contains(prakResp, "ALJABAR LINEAR (PRAKTIKUM)") ||
		!strings.Contains(prakResp, "Muhammad Rizqi") {
		t.Errorf("Expected explicit praktikum task added, got:\n%s", prakResp)
	}

	teoriResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah alin teori | Pertemuan-1 | besok 8:40", cfg, refNow)
	if !strings.Contains(teoriResp, "BERHASIL DITAMBAHKAN") ||
		!strings.Contains(teoriResp, "ALJABAR LINEAR (TEORI)") ||
		!strings.Contains(teoriResp, "Nurjannah") {
		t.Errorf("Expected explicit teori task added, got:\n%s", teoriResp)
	}

	descPrakResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah alin | Laporan Praktikum Modul 2 | jumat 23:59", cfg, refNow)
	if !strings.Contains(descPrakResp, "BERHASIL DITAMBAHKAN") ||
		!strings.Contains(descPrakResp, "ALJABAR LINEAR (PRAKTIKUM)") {
		t.Errorf("Expected auto-detect praktikum from desc, got:\n%s", descPrakResp)
	}

	descTeoriResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah alin | Resume Bab 2 Transformasi Linier | jumat 23:59", cfg, refNow)
	if !strings.Contains(descTeoriResp, "BERHASIL DITAMBAHKAN") ||
		!strings.Contains(descTeoriResp, "ALJABAR LINEAR (TEORI)") {
		t.Errorf("Expected auto-detect teori from desc, got:\n%s", descTeoriResp)
	}

	// Mata kuliah dengan satu jenis sesi tidak memerlukan disambiguasi.
	aokResp := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah aok | Tugas Pipeline | besok 10:00", cfg, refNow)
	if !strings.Contains(aokResp, "BERHASIL DITAMBAHKAN") ||
		!strings.Contains(aokResp, "ARSITEKTUR DAN ORGANISASI KOMPUTER") {
		t.Errorf("Expected single session course to be added directly, got:\n%s", aokResp)
	}

	filterAll := tm.HandleCommand(groupJID, true, userJID, false, "!tugas alin", cfg, refNow)
	if !strings.Contains(filterAll, "ALJABAR LINEAR (PRAKTIKUM)") || !strings.Contains(filterAll, "ALJABAR LINEAR (TEORI)") {
		t.Errorf("Expected !tugas alin to show both, got:\n%s", filterAll)
	}

	filterPrak := tm.HandleCommand(groupJID, true, userJID, false, "!tugas alin praktikum", cfg, refNow)
	if !strings.Contains(filterPrak, "ALJABAR LINEAR (PRAKTIKUM)") || strings.Contains(filterPrak, "ALJABAR LINEAR (TEORI)") {
		t.Errorf("Expected !tugas alin praktikum to only show praktikum, got:\n%s", filterPrak)
	}
}

func TestTaskClassScopingAndTwoWaySync(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "test_class_sync.db")

	db := openLegacyTaskTestDB(t, testDB)

	tm, err := NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewTaskManager: %v", err)
	}

	cfg, err := schedule.LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal memuat jadwal.json: %v", err)
	}

	refNow := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)
	group3AJID := "1203633182@g.us"
	class3A := "D4-TI-SMT3-A"
	adminJID := "628120001@s.whatsapp.net"
	userDM := "628129999@s.whatsapp.net"

	webTaskID, _, err := tm.AddWebTask("SBD", "Tugas Web Dashboard", "Jumat 23:59", "web-dashboard", refNow, class3A)
	if err != nil {
		t.Fatalf("AddWebTask failed: %v", err)
	}

	tasks3A, err := tm.GetTasksByClassID(class3A, refNow)
	if err != nil || len(tasks3A) != 1 {
		t.Fatalf("Expected 1 task for class 3A, got %d (err: %v)", len(tasks3A), err)
	}
	if tasks3A[0].ClassID != class3A {
		t.Errorf("Expected ClassID %s, got %s", class3A, tasks3A[0].ClassID)
	}

	tasks1A, err := tm.GetTasksByClassID("D3-TI-1A", refNow)
	if err != nil || len(tasks1A) != 0 {
		t.Fatalf("Expected 0 task for class 1A, got %d", len(tasks1A))
	}

	// Tugas dari web harus terlihat dari grup WhatsApp untuk kelas yang sama.
	waListReply := tm.HandleCommand(group3AJID, true, adminJID, false, "!tugas", cfg, refNow, class3A)
	if !strings.Contains(waListReply, "Tugas Web Dashboard") {
		t.Errorf("Expected WA group 3A to see web task, got:\n%s", waListReply)
	}

	doneReply := tm.HandleCommand(group3AJID, true, adminJID, true, "!tugas selesai "+strings.TrimSpace(string(rune('0'+webTaskID))), cfg, refNow, class3A)
	if !strings.Contains(doneReply, "TUGAS SELESAI") {
		t.Errorf("Expected task to be completed by WA admin, got:\n%s", doneReply)
	}

	tasksAfterDone, err := tm.GetTasksByClassID(class3A, refNow)
	if err != nil || len(tasksAfterDone) != 0 {
		t.Errorf("Expected 0 active tasks after completion, got %d", len(tasksAfterDone))
	}

	// Tugas pribadi tidak boleh masuk ke scope kelas.
	dmReply := tm.HandleCommand(userDM, false, userDM, false, "!tugas tambah Pribadi | Beli buku catatan | besok", cfg, refNow)
	if !strings.Contains(dmReply, "BERHASIL DITAMBAHKAN") {
		t.Fatalf("Expected personal task added, got:\n%s", dmReply)
	}
	tasksClassCheck, err := tm.GetTasksByClassID(class3A, refNow)
	if err != nil || len(tasksClassCheck) != 0 {
		t.Errorf("Personal task leaked into class tasks! Found %d tasks", len(tasksClassCheck))
	}
}
