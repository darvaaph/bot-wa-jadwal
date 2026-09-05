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

	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database SQLite: %v", err)
	}
	defer db.Close()

	tm, err := NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewTaskManager: %v", err)
	}

	cfg, err := LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal memuat jadwal.json: %v", err)
	}

	// Gunakan waktu acuan tetap: Rabu, 9 September 2026 pukul 10:00 WIB
	refNow := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)

	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"
	otherUserJID := "628120002@s.whatsapp.net"

	// 1. Test Tambah Tugas di Grup oleh Non-Admin (Harus Ditolak)
	nonAdminReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas tambah SBD | Laporan 1 | Jumat 23:59", cfg, refNow)
	if !strings.Contains(nonAdminReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected in group, got: %s", nonAdminReply)
	}

	// 2. Test Tambah Tugas di Grup oleh Admin (Harus Berhasil dengan nama matkul dinormalisasi)
	adminReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Jumat 23:59", cfg, refNow)
	if !strings.Contains(adminReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(adminReply, "SISTEM BASIS DATA") {
		t.Errorf("Expected admin to succeed in group with normalized matkul, got: %s", adminReply)
	}

	// 3. Test Anti-Duplikasi di Grup (Tugas serupa harus ditolak)
	dupReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Sabtu 12:00", cfg, refNow)
	if !strings.Contains(dupReply, "Tugas Serupa Sudah Terdaftar") {
		t.Errorf("Expected duplicate task to be rejected, got: %s", dupReply)
	}

	// 4. Test Validasi Format Pipa Salah
	badFormatReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Tugas Tanpa Pipa", cfg, refNow)
	if !strings.Contains(badFormatReply, "Format Penambahan Tugas Kurang Tepat") {
		t.Errorf("Expected format validation error, got: %s", badFormatReply)
	}

	// 5. Test Tambah Tugas Jatuh Tempo Hari Ini dan Besok
	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Aljabar | Kuis Hari Ini | hari ini 23:59", cfg, refNow)
	_ = tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Matdis | PR Logika | besok 14:00", cfg, refNow)

	// 6. Test Lihat Daftar Tugas Grup (!tugas) dengan Badge Urgensi
	listReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas", cfg, refNow)
	if !strings.Contains(listReply, "DAFTAR TUGAS KELAS") ||
		!strings.Contains(listReply, "SISTEM BASIS DATA") ||
		!strings.Contains(listReply, "ALJABAR LINEAR") ||
		!strings.Contains(listReply, "DEADLINE HARI INI") {
		t.Errorf("Expected group task list with urgency badges, got: %s", listReply)
	}

	// 7. Test Filter !tugas hari ini
	todayReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas hari ini", cfg, refNow)
	if !strings.Contains(todayReply, "DEADLINE HARI INI") || !strings.Contains(todayReply, "ALJABAR LINEAR") {
		t.Errorf("Expected today's task to be Aljabar, got: %s", todayReply)
	}

	// 8. Test Filter !tugas besok
	tomorrowReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas besok", cfg, refNow)
	if !strings.Contains(tomorrowReply, "DEADLINE BESOK") || !strings.Contains(tomorrowReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected tomorrow's task to be Matdis, got: %s", tomorrowReply)
	}

	// 9. Test Pemisahan Scope (Tugas grup TIDAK boleh bocor ke DM user)
	dmListReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas", cfg, refNow)
	if !strings.Contains(dmListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected empty tasks in fresh personal DM, got: %s", dmListReply)
	}

	// 10. Test Tambah Tugas Pribadi di DM (Bebas tanpa admin)
	userReply := tm.HandleCommand(userJID, false, userJID, true, "!tugas tambah Pribadi | Belajar Golang | Minggu 20:00", cfg, refNow)
	if !strings.Contains(userReply, "BERHASIL DITAMBAHKAN") {
		t.Errorf("Expected user to add personal task in DM, got: %s", userReply)
	}

	// Tugas user 1 tidak boleh bocor ke user 2
	user2ListReply := tm.HandleCommand(otherUserJID, false, otherUserJID, true, "!tugas", cfg, refNow)
	if !strings.Contains(user2ListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected user2 tasks to be empty, got: %s", user2ListReply)
	}

	// 11. Test Selesaikan Tugas (!tugas selesai 1)
	// Non-admin di grup mencoba menyelesaikan (Harus Ditolak)
	nonAdminDone := tm.HandleCommand(groupJID, true, userJID, false, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(nonAdminDone, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected completing task in group, got: %s", nonAdminDone)
	}

	// Admin menyelesaikan tugas 1
	adminDone := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(adminDone, "TUGAS SELESAI") {
		t.Errorf("Expected admin to complete task 1, got: %s", adminDone)
	}

	// 12. Test Integrasi Pengingat Pagi dengan Alert Tugas (BuildMorningReminder)
	reminderMsg := BuildMorningReminder(groupJID, cfg, tm, refNow)
	if !strings.Contains(reminderMsg, "PERINGATAN DEADLINE TUGAS") || !strings.Contains(reminderMsg, "ALJABAR LINEAR") {
		t.Errorf("Expected morning reminder to include urgent task alert, got: %s", reminderMsg)
	}

	// 13. Test Panduan Perintah (!tugas bantuan)
	helpReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas bantuan", cfg, refNow)
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

	// 15. Test Parsing Format Jam Saja Tanpa Tanggal ("22.22", "22:22", "jam 22.22")
	targetJamSaja, labelJamSaja := parseDeadline("22.22", tSabtu)
	if targetJamSaja.Day() != 5 || targetJamSaja.Month() != 9 || targetJamSaja.Hour() != 22 || targetJamSaja.Minute() != 22 {
		t.Errorf("Expected 5 Sep 22:22, got: %v", targetJamSaja)
	}
	if !strings.Contains(labelJamSaja, "Hari Ini (Sabtu), 22:22 WIB") {
		t.Errorf("Expected 'Hari Ini (Sabtu), 22:22 WIB', got: %s", labelJamSaja)
	}

	// 16. Test Validasi Mata Kuliah dan Alias Pintar
	// Matkul tidak terdaftar (Kalkulus) -> Harus Ditolak dengan rekomendasi daftar matkul
	badMatkulReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Kalkulus | Latihan 1 | 22.22", cfg, tSabtu)
	if !strings.Contains(badMatkulReply, "Tidak Terdaftar") || !strings.Contains(badMatkulReply, "Daftar Mata Kuliah Kelas") {
		t.Errorf("Expected unlisted matkul error with course guide, got: %s", badMatkulReply)
	}

	// Alias "mtk" -> Harus berhasil dan dinormalisasi menjadi "Matematika Diskrit Lanjut"
	mtkReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah mtk | Tugas Graph | 22.22", cfg, tSabtu)
	if !strings.Contains(mtkReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(mtkReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected 'mtk' to normalize to 'Matematika Diskrit Lanjut', got: %s", mtkReply)
	}

	// Alias "matematika" -> Harus berhasil dan dinormalisasi
	matematikaReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah matematika | Tugas Tree | besok 10:00", cfg, tSabtu)
	if !strings.Contains(matematikaReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(matematikaReply, "MATEMATIKA DISKRIT LANJUT") {
		t.Errorf("Expected 'matematika' to normalize to 'Matematika Diskrit Lanjut', got: %s", matematikaReply)
	}

	// Matkul "Umum" -> Harus diterima sebagai tugas non-matkul
	umumReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah umum | Bawa Perlengkapan Lab | 22.22", cfg, tSabtu)
	if !strings.Contains(umumReply, "BERHASIL DITAMBAHKAN") || !strings.Contains(umumReply, "UMUM") {
		t.Errorf("Expected 'umum' task to succeed, got: %s", umumReply)
	}

	// 17. Test Edit / Perpanjangan Tenggat Waktu Tugas (!tugas edit / !tugas mundur)
	// Non-admin di grup mencoba mengedit (Harus Ditolak)
	nonAdminEdit := tm.HandleCommand(groupJID, true, userJID, false, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(nonAdminEdit, "Akses Ditolak") {
		t.Errorf("Expected non-admin edit to be rejected in group, got: %s", nonAdminEdit)
	}

	// Admin mengedit tenggat saja (!tugas edit 2 | Minggu 23:59)
	adminEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(adminEdit, "BERHASIL DIPERBARUI") || !strings.Contains(adminEdit, "Minggu") {
		t.Errorf("Expected admin edit deadline to succeed, got: %s", adminEdit)
	}

	// Admin mengedit deskripsi dan tenggat sekaligus (!tugas edit 2 | Revisi Lapres 1 | Senin 12:00)
	adminEditBoth := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Revisi Lapres 1 | Senin 12:00", cfg, tSabtu)
	if !strings.Contains(adminEditBoth, "BERHASIL DIPERBARUI") || !strings.Contains(adminEditBoth, "Revisi Lapres 1") {
		t.Errorf("Expected admin edit both desc and deadline to succeed, got: %s", adminEditBoth)
	}

	// Edit tugas dengan ID yang tidak ada
	badIDEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 999 | besok", cfg, tSabtu)
	if !strings.Contains(badIDEdit, "tidak ditemukan") {
		t.Errorf("Expected non-existent task to report not found, got: %s", badIDEdit)
	}

	// 18. Test Filter Tugas per Mata Kuliah (!tugas sbd, !tugas aljabar, !tugas matkul sbd)
	// Filter tugas SBD (harus memunculkan tugas SBD dan tidak memunculkan Aljabar)
	filterSBD := tm.HandleCommand(groupJID, true, userJID, false, "!tugas sbd", cfg, tSabtu)
	if !strings.Contains(filterSBD, "SISTEM BASIS DATA") || strings.Contains(filterSBD, "ALJABAR LINEAR") {
		t.Errorf("Expected filter SBD to only show SBD tasks, got:\n%s", filterSBD)
	}

	// Filter tugas Aljabar
	filterAljabar := tm.HandleCommand(groupJID, true, userJID, false, "!tugas aljabar", cfg, tSabtu)
	if !strings.Contains(filterAljabar, "ALJABAR LINEAR") || strings.Contains(filterAljabar, "SISTEM BASIS DATA") {
		t.Errorf("Expected filter Aljabar to only show Aljabar tasks, got:\n%s", filterAljabar)
	}

	// Filter matkul yang belum ada tugas aktifnya (misal Sistem Operasi / SO)
	filterSO := tm.HandleCommand(groupJID, true, userJID, false, "!tugas so", cfg, tSabtu)
	if !strings.Contains(filterSO, "Tidak ada tugas aktif") || !strings.Contains(filterSO, "SISTEM OPERASI") {
		t.Errorf("Expected empty message for SO filter, got:\n%s", filterSO)
	}

	// Sub-perintah !tugas matkul sbd
	filterMatkulSBD := tm.HandleCommand(groupJID, true, userJID, false, "!tugas matkul sbd", cfg, tSabtu)
	if !strings.Contains(filterMatkulSBD, "SISTEM BASIS DATA") {
		t.Errorf("Expected !tugas matkul sbd to work, got:\n%s", filterMatkulSBD)
	}

	// 19. Test Riwayat Tugas Selesai (!tugas riwayat / !tugas arsip)
	// Tugas ID #1 sudah diselesaikan pada step 11 di atas, verifikasi muncul di !tugas riwayat
	riwayatResp := tm.HandleCommand(groupJID, true, userJID, false, "!tugas riwayat", cfg, tSabtu)
	if !strings.Contains(riwayatResp, "ARSIP & RIWAYAT TUGAS SELESAI") || !strings.Contains(riwayatResp, "#1") || !strings.Contains(riwayatResp, "✅") {
		t.Errorf("Expected task #1 in riwayat, got: %s", riwayatResp)
	}

	// Alias !tugas arsip
	arsipResp := tm.HandleCommand(groupJID, true, userJID, false, "!tugas arsip", cfg, tSabtu)
	if !strings.Contains(arsipResp, "ARSIP & RIWAYAT TUGAS SELESAI") || !strings.Contains(arsipResp, "#1") {
		t.Errorf("Expected task #1 in arsip, got: %s", arsipResp)
	}

	// Selesaikan tugas ID #2
	doneResp2 := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 2", cfg, tSabtu)
	if !strings.Contains(doneResp2, "TUGAS SELESAI") || !strings.Contains(doneResp2, "#2") {
		t.Errorf("Expected task #2 to be completed, got: %s", doneResp2)
	}

	// Cek daftar riwayat terbaru (ID #2 dan #1 harus ada)
	riwayatResp2 := tm.HandleCommand(groupJID, true, userJID, false, "!tugas riwayat", cfg, tSabtu)
	if !strings.Contains(riwayatResp2, "#2") || !strings.Contains(riwayatResp2, "#1") {
		t.Errorf("Expected both task #1 and #2 in riwayat, got: %s", riwayatResp2)
	}
}
