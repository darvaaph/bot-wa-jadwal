package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestTaskManager(t *testing.T) {
	testDB := "test_tugas.db"
	defer os.Remove(testDB)

	tm, err := NewTaskManager(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewTaskManager: %v", err)
	}
	defer tm.Close()

	// Gunakan waktu acuan tetap: Rabu, 9 September 2026 pukul 10:00 WIB
	refNow := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)

	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"
	otherUserJID := "628120002@s.whatsapp.net"

	// 1. Test Tambah Tugas di Grup oleh Non-Admin (Harus Ditolak)
	nonAdminReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas tambah SBD | Laporan 1 | Jumat 23:59", refNow)
	if !strings.Contains(nonAdminReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected in group, got: %s", nonAdminReply)
	}

	// 2. Test Tambah Tugas di Grup oleh Admin (Harus Berhasil dengan status urgensi)
	adminReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Jumat 23:59", refNow)
	if !strings.Contains(adminReply, "BERHASIL DITAMBAHKAN") {
		t.Errorf("Expected admin to succeed in group, got: %s", adminReply)
	}

	// 3. Test Anti-Duplikasi di Grup (Tugas serupa harus ditolak)
	dupReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Sabtu 12:00", refNow)
	if !strings.Contains(dupReply, "Tugas Serupa Sudah Terdaftar") {
		t.Errorf("Expected duplicate task to be rejected, got: %s", dupReply)
	}

	// 4. Test Validasi Format Pipa Salah
	badFormatReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Tugas Tanpa Pipa", refNow)
	if !strings.Contains(badFormatReply, "Format Penambahan Tugas Kurang Tepat") {
		t.Errorf("Expected format validation error, got: %s", badFormatReply)
	}

	// 5. Test Tambah Tugas Jatuh Tempo Hari Ini dan Besok
	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Aljabar | Kuis Hari Ini | hari ini 23:59", refNow)
	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Matdis | PR Logika | besok 14:00", refNow)

	// 6. Test Lihat Daftar Tugas Grup (!tugas) dengan Badge Urgensi
	listReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas", refNow)
	if !strings.Contains(listReply, "DAFTAR TUGAS KELAS") ||
		!strings.Contains(listReply, "SBD") ||
		!strings.Contains(listReply, "ALJABAR") ||
		!strings.Contains(listReply, "DEADLINE HARI INI") {
		t.Errorf("Expected group task list with urgency badges, got: %s", listReply)
	}

	// 7. Test Filter !tugas hari ini
	todayReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas hari ini", refNow)
	if !strings.Contains(todayReply, "DEADLINE HARI INI") || !strings.Contains(todayReply, "ALJABAR") {
		t.Errorf("Expected today's task to be Aljabar, got: %s", todayReply)
	}

	// 8. Test Filter !tugas besok
	tomorrowReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas besok", refNow)
	if !strings.Contains(tomorrowReply, "DEADLINE BESOK") || !strings.Contains(tomorrowReply, "MATDIS") {
		t.Errorf("Expected tomorrow's task to be Matdis, got: %s", tomorrowReply)
	}

	// 9. Test Pemisahan Scope (Tugas grup TIDAK boleh bocor ke DM user)
	dmListReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas", refNow)
	if !strings.Contains(dmListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected empty tasks in fresh personal DM, got: %s", dmListReply)
	}

	// 10. Test Tambah Tugas Pribadi di DM (Bebas tanpa admin)
	userReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas tambah Pribadi | Belajar Golang | Minggu 20:00", refNow)
	if !strings.Contains(userReply, "BERHASIL DITAMBAHKAN") {
		t.Errorf("Expected user to add personal task in DM, got: %s", userReply)
	}

	// Tugas user 1 tidak boleh bocor ke user 2
	user2ListReply := tm.HandleCommand(otherUserJID, false, otherUserJID, true, "!tugas", refNow)
	if !strings.Contains(user2ListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected user2 tasks to be empty, got: %s", user2ListReply)
	}

	// 11. Test Selesaikan Tugas (!tugas selesai 1)
	// Non-admin di grup mencoba menyelesaikan (Harus Ditolak)
	nonAdminDone := tm.HandleCommand(groupJID, true, userJID, false, "!tugas selesai 1", refNow)
	if !strings.Contains(nonAdminDone, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected completing task in group, got: %s", nonAdminDone)
	}

	// Admin menyelesaikan tugas 1
	adminDone := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 1", refNow)
	if !strings.Contains(adminDone, "TUGAS SELESAI") {
		t.Errorf("Expected admin to complete task 1, got: %s", adminDone)
	}

	// 12. Test Integrasi Pengingat Pagi dengan Alert Tugas (BuildMorningReminder)
	cfg, err := LoadJadwal("jadwal.json")
	if err == nil {
		reminderMsg := BuildMorningReminder(groupJID, cfg, tm, refNow)
		if !strings.Contains(reminderMsg, "PERINGATAN DEADLINE TUGAS") || !strings.Contains(reminderMsg, "ALJABAR") {
			t.Errorf("Expected morning reminder to include urgent task alert, got: %s", reminderMsg)
		}
	}

	// 13. Test Panduan Perintah (!tugas bantuan)
	helpReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas bantuan", refNow)
	if !strings.Contains(helpReply, "PANDUAN DEADLINE TRACKER TUGAS") {
		t.Errorf("Expected help guide for tasks, got: %s", helpReply)
	}

	// 14. Test Parsing Format Nama Bulan Indonesia ("5 sep 22.15" dan "8 september")
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
}
