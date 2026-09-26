package task

import (
	"bot-jadwal/internal/database"
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
	if !strings.Contains(nonAdminReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected mutation to be redirected to dashboard, got: %s", nonAdminReply)
	}

	adminReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Jumat 23:59", cfg, refNow)
	if !strings.Contains(adminReply, "PENGELOLAAN DATA TERPUSAT") || !strings.Contains(adminReply, "app.html") {
		t.Errorf("Expected admin mutation to be redirected to dashboard, got: %s", adminReply)
	}

	dupReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah SBD | Laporan Praktikum Modul 1 | Sabtu 12:00", cfg, refNow)
	if !strings.Contains(dupReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected duplicate mutation to be redirected to dashboard, got: %s", dupReply)
	}

	badFormatReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah Tugas Tanpa Pipa", cfg, refNow)
	if !strings.Contains(badFormatReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected malformed mutation to be redirected to dashboard, got: %s", badFormatReply)
	}

	// Seed data baca via jalur non-perintah (dashboard/API), bukan via command.
	if _, _, err := tm.AddTask(groupJID, true, "Sistem Basis Data", "Laporan Praktikum Modul 1", "Jumat 23:59", userJID, refNow); err != nil {
		t.Fatalf("AddTask SBD failed: %v", err)
	}
	if _, _, err := tm.AddTask(groupJID, true, "Aljabar Linear", "Kuis Hari Ini", "hari ini 23:59", userJID, refNow); err != nil {
		t.Fatalf("AddTask Aljabar failed: %v", err)
	}
	if _, _, err := tm.AddTask(groupJID, true, "Matematika Diskrit Lanjut", "PR Logika", "besok 14:00", userJID, refNow); err != nil {
		t.Fatalf("AddTask Matdis failed: %v", err)
	}

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
	if !strings.Contains(userReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected DM mutation to be redirected to dashboard, got: %s", userReply)
	}

	// Tugas user 1 tidak boleh bocor ke user 2
	user2ListReply := tm.HandleCommand(otherUserJID, false, otherUserJID, true, "!tugas", cfg, refNow)
	if !strings.Contains(user2ListReply, "Tidak ada tugas aktif") {
		t.Errorf("Expected user2 tasks to be empty, got: %s", user2ListReply)
	}

	nonAdminDone := tm.HandleCommand(groupJID, true, userJID, false, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(nonAdminDone, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected non-admin completion to be redirected to dashboard, got: %s", nonAdminDone)
	}

	adminDone := tm.HandleCommand(groupJID, true, userJID, true, "!tugas selesai 1", cfg, refNow)
	if !strings.Contains(adminDone, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected admin completion to be redirected to dashboard, got: %s", adminDone)
	}

	// Penyelesaian via jalur non-perintah agar alur baca riwayat tetap teruji.
	if ok, err := tm.CompleteTask(groupJID, 1); err != nil || !ok {
		t.Fatalf("CompleteTask #1 failed: ok=%v err=%v", ok, err)
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
	if !strings.Contains(badMatkulReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected unlisted matkul mutation to be redirected to dashboard, got: %s", badMatkulReply)
	}

	mtkReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah mtk | Tugas Graph | 22.22", cfg, tSabtu)
	if !strings.Contains(mtkReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected 'mtk' mutation to be redirected to dashboard, got: %s", mtkReply)
	}

	matematikaReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah matematika | Tugas Tree | besok 10:00", cfg, tSabtu)
	if !strings.Contains(matematikaReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected 'matematika' mutation to be redirected to dashboard, got: %s", matematikaReply)
	}

	umumReply := tm.HandleCommand(groupJID, true, userJID, true, "!tugas tambah umum | Bawa Perlengkapan Lab | 22.22", cfg, tSabtu)
	if !strings.Contains(umumReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected 'umum' mutation to be redirected to dashboard, got: %s", umumReply)
	}

	nonAdminEdit := tm.HandleCommand(groupJID, true, userJID, false, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(nonAdminEdit, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected non-admin edit to be redirected to dashboard, got: %s", nonAdminEdit)
	}

	adminEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Minggu 23:59", cfg, tSabtu)
	if !strings.Contains(adminEdit, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected admin edit to be redirected to dashboard, got: %s", adminEdit)
	}

	adminEditBoth := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 2 | Revisi Lapres 1 | Senin 12:00", cfg, tSabtu)
	if !strings.Contains(adminEditBoth, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected admin edit both to be redirected to dashboard, got: %s", adminEditBoth)
	}

	badIDEdit := tm.HandleCommand(groupJID, true, userJID, true, "!tugas edit 999 | besok", cfg, tSabtu)
	if !strings.Contains(badIDEdit, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected edit of non-existent task to be redirected to dashboard, got: %s", badIDEdit)
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
	if !strings.Contains(doneResp2, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected completion via command to be redirected to dashboard, got: %s", doneResp2)
	}
	if ok, err := tm.CompleteTask(groupJID, 2); err != nil || !ok {
		t.Fatalf("CompleteTask #2 failed: ok=%v err=%v", ok, err)
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

	// Seluruh varian tambah via command dialihkan ke dashboard (read-only).
	// Normalisasi teori/praktikum kini menjadi tanggung jawab form dashboard.
	for _, cmd := range []string{
		"!tugas tambah alin | Pertemuan-1 | besok 8:40",
		"!tugas tambah alin praktikum | Pertemuan-1 | besok 8:40",
		"!tugas tambah alin teori | Pertemuan-1 | besok 8:40",
		"!tugas tambah alin | Laporan Praktikum Modul 2 | jumat 23:59",
		"!tugas tambah alin | Resume Bab 2 Transformasi Linier | jumat 23:59",
		"!tugas tambah aok | Tugas Pipeline | besok 10:00",
	} {
		resp := tm.HandleCommand(groupJID, true, userJID, true, cmd, cfg, refNow)
		if !strings.Contains(resp, "PENGELOLAAN DATA TERPUSAT") {
			t.Errorf("Expected %q to be redirected to dashboard, got:\n%s", cmd, resp)
		}
	}

	// Seed bacaan via jalur non-perintah dengan nama resmi sesi.
	if _, _, err := tm.AddTask(groupJID, true, "Aljabar Linear (Praktikum)", "Pertemuan-1", "besok 8:40", userJID, refNow); err != nil {
		t.Fatalf("AddTask praktikum failed: %v", err)
	}
	if _, _, err := tm.AddTask(groupJID, true, "Aljabar Linear (Teori)", "Pertemuan-1", "besok 8:40", userJID, refNow); err != nil {
		t.Fatalf("AddTask teori failed: %v", err)
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
	if !strings.Contains(doneReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected WA completion to be redirected to dashboard, got:\n%s", doneReply)
	}

	tasksAfterDone, err := tm.GetTasksByClassID(class3A, refNow)
	if err != nil || len(tasksAfterDone) != 1 {
		t.Errorf("Expected task to remain active after refused WA completion, got %d", len(tasksAfterDone))
	}

	// Perintah tambah pribadi via WA dialihkan; scope kelas tidak berubah.
	dmReply := tm.HandleCommand(userDM, false, userDM, false, "!tugas tambah Pribadi | Beli buku catatan | besok", cfg, refNow)
	if !strings.Contains(dmReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Fatalf("Expected personal mutation to be redirected to dashboard, got:\n%s", dmReply)
	}
	tasksClassCheck, err := tm.GetTasksByClassID(class3A, refNow)
	if err != nil || len(tasksClassCheck) != 1 {
		t.Errorf("Class tasks changed by refused WA mutation! Found %d tasks", len(tasksClassCheck))
	}
}

func TestHandleCommand_V3PublishedReads(t *testing.T) {
	db, err := database.InitDB(filepath.Join(t.TempDir(), "test_v3_tasks.db"))
	if err != nil {
		t.Fatalf("InitDB v3 failed: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	// NewTaskManager membuat tabel tasks warisan, sehingga manajer
	// dibangun langsung di atas database yang sudah berskema v3.
	tm := &TaskManager{db: db}

	var userID int64
	if err := db.QueryRow(`INSERT INTO users (identity_key, display_name, password_hash) VALUES ('pj-test', 'PJ Test', 'hash') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	var classID int64
	if err := db.QueryRow(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-3A', 'd4-ti-3a', 'D4 TI', 2024, 'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatalf("seed classes: %v", err)
	}
	now := "2026-09-24T10:00:00.000Z"
	var semID int64
	if err := db.QueryRow(`INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31', 'ACTIVE', ?, ?) RETURNING id`, classID, now, now).Scan(&semID); err != nil {
		t.Fatalf("seed semesters: %v", err)
	}
	var courseID int64
	if err := db.QueryRow(`INSERT INTO courses (code, name) VALUES ('SBDV3', 'Sistem Basis Data V3') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatalf("seed courses: %v", err)
	}
	var offeringID int64
	if err := db.QueryRow(`INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?, ?, 'TEORI', 'SBD V3 Teori') RETURNING id`, semID, courseID).Scan(&offeringID); err != nil {
		t.Fatalf("seed offerings: %v", err)
	}

	refNow := time.Date(2026, 9, 9, 10, 0, 0, 0, time.Local)
	todayDeadline := refNow.Format(time.RFC3339)
	seedTask := func(title, status, deadline, completedAt, deletedAt string) {
		t.Helper()
		var completed, deleted, deletedBy any
		if completedAt != "" {
			completed = completedAt
		}
		if deletedAt != "" {
			deleted = deletedAt
			deletedBy = userID
		}
		if _, err := db.Exec(`INSERT INTO tasks (course_offering_id, title, instructions, deadline_at, task_type, submission_text,
			publication_status, review_state, created_by_user_id, published_at, completed_at, deleted_at, deleted_by_user_id, version)
			VALUES (?, ?, 'Kerjakan dengan benar.', ?, 'INDIVIDUAL', 'Via LMS', ?, 'NOT_REVIEWED', ?, '2026-09-24T10:00:00.000Z', ?, ?, ?, 1)`,
			offeringID, title, deadline, status, userID, completed, deleted, deletedBy); err != nil {
			t.Fatalf("seed task %q: %v", title, err)
		}
	}
	seedTask("Tugas V3 Terbit", "PUBLISHED", "2026-09-30T16:00:00.000Z", "", "")
	seedTask("Tugas V3 Hari Ini", "PUBLISHED", todayDeadline, "", "")
	seedTask("Tugas V3 Draf", "DRAFT", "2026-09-30T16:00:00.000Z", "", "")
	seedTask("Tugas V3 Selesai", "PUBLISHED", "2026-09-01T16:00:00.000Z", "2026-09-05T10:00:00.000Z", "")
	seedTask("Tugas V3 Hapus", "PUBLISHED", "2026-09-30T16:00:00.000Z", "", "2026-09-06T10:00:00.000Z")

	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"

	countTasks := func() int {
		t.Helper()
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&n); err != nil {
			t.Fatalf("count tasks: %v", err)
		}
		return n
	}

	listReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas", nil, refNow, "D4-TI-3A")
	if !strings.Contains(listReply, "Tugas V3 Terbit") || !strings.Contains(listReply, "Tugas V3 Hari Ini") {
		t.Errorf("Expected published v3 tasks in list, got:\n%s", listReply)
	}
	for _, hidden := range []string{"Tugas V3 Draf", "Tugas V3 Selesai", "Tugas V3 Hapus"} {
		if strings.Contains(listReply, hidden) {
			t.Errorf("Expected %q hidden from active list, got:\n%s", hidden, listReply)
		}
	}

	before := countTasks()
	for _, cmd := range []string{
		"!tugas tambah SBD | Laporan 1 | Jumat 23:59",
		"!tugas selesai 1",
		"!tugas hapus 1",
		"!tugas edit 1 | minggu 23:59",
	} {
		resp := tm.HandleCommand(groupJID, true, userJID, true, cmd, nil, refNow, "D4-TI-3A")
		if !strings.Contains(resp, "PENGELOLAAN DATA TERPUSAT") {
			t.Errorf("Expected %q redirected, got:\n%s", cmd, resp)
		}
	}
	if got := countTasks(); got != before {
		t.Errorf("Mutation via command changed DB: before=%d after=%d", before, got)
	}

	todayReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas hari ini", nil, refNow, "D4-TI-3A")
	if !strings.Contains(todayReply, "Tugas V3 Hari Ini") || strings.Contains(todayReply, "Tugas V3 Terbit") {
		t.Errorf("Expected only today's v3 task, got:\n%s", todayReply)
	}

	historyReply := tm.HandleCommand(groupJID, true, userJID, false, "!tugas riwayat", nil, refNow, "D4-TI-3A")
	if !strings.Contains(historyReply, "Tugas V3 Selesai") {
		t.Errorf("Expected completed v3 task in history, got:\n%s", historyReply)
	}
}
