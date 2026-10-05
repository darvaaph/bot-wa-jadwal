package bot

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

func setupTestV1DB(t *testing.T) *V1BotService {
	dbPath := filepath.Join(t.TempDir(), "v1_service_test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}

	// Masukkan kelas pilot
	_, err = db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES 
			(1, 'D4-TI-2024-A', 'd4-ti-2024-a', 'Teknik Informatika', 2024, 'A', 'ACTIVE'),
			(2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'Teknik Informatika', 2024, 'B', 'ACTIVE');

		INSERT INTO users (id, identity_key, display_name, password_hash, status)
		VALUES (1, 'system:test', 'Pengguna Uji', 'disabled', 'ACTIVE');
		
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES 
			(1, 1, '2024/2025', 'GANJIL', '2024-09-01', '2025-02-28', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
			(2, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-02-28', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

		INSERT INTO rooms (id, code, name, building, room_type, capacity, status)
		VALUES (1, 'LAB-1', 'Laboratorium Komputer 1', 'Gedung TI Lt. 2', 'LAB', 30, 'ACTIVE');

		INSERT INTO courses (id, code, name, status)
		VALUES 
			(1, 'TI2101', 'Algoritma & Pemrograman', 'ACTIVE'),
			(2, 'TI2102', 'Struktur Data', 'ACTIVE');

		INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name, status)
		VALUES 
			(1, 1, 1, 'THEORY', 'Algoritma & Pemrograman (Teori)', 'ACTIVE'),
			(2, 1, 2, 'PRACTICE', 'Struktur Data (Praktik)', 'ACTIVE');

		INSERT INTO lecturers (id, code, full_name, status)
		VALUES (1, 'BDI', 'Budi Santoso, M.Kom.', 'ACTIVE');

		INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
		VALUES (1, 1, 'PRIMARY');

		-- Jadwal pola reguler: Senin (1) jam 08:00 - 10:00
		INSERT INTO schedule_patterns (
			id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status
		)
		VALUES (1, 1, 1, 1, '08:00', '10:00', '2024-09-01', 'ACTIVE');
	`)
	if err != nil {
		t.Fatalf("Seed test data gagal: %v", err)
	}

	return NewV1BotService(db)
}

func TestV1BotService_FindClass(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	// 1. Pencarian via slug
	c, err := svc.FindClass(ctx, "d4-ti-2024-a")
	if err != nil || c.ID != 1 {
		t.Errorf("FindClass slug gagal: %v, expected ID 1, got %+v", err, c)
	}

	// 2. Pencarian via code
	c, err = svc.FindClass(ctx, "D4-TI-2024-B")
	if err != nil || c.ID != 2 {
		t.Errorf("FindClass code gagal: %v, expected ID 2, got %+v", err, c)
	}

	// 3. Pencarian via alias santai mahasiswa ("1a", "smt1-a")
	c, err = svc.FindClass(ctx, "1a")
	if err != nil || c.ID != 1 {
		t.Errorf("FindClass alias '1a' gagal: %v, got %+v", err, c)
	}

	c, err = svc.FindClass(ctx, "d4-ti-smt1-a")
	if err != nil || c.ID != 1 {
		t.Errorf("FindClass alias 'd4-ti-smt1-a' gagal: %v, got %+v", err, c)
	}

	// 4. Kelas non-pilot tidak ditemukan
	_, err = svc.FindClass(ctx, "D4-TI-SMT3-A")
	if err == nil {
		t.Errorf("Expected D4-TI-SMT3-A to not be found in v1 pilot, but got nil error")
	}
}

func TestV1BotService_BindChannel(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	jid := "120363001234567890@g.us"
	err := svc.BindChannel(ctx, 1, jid, "Grup TI 2024 A")
	if err != nil {
		t.Fatalf("BindChannel gagal: %v", err)
	}

	// Cek idempotensi dan update display_name
	err = svc.BindChannel(ctx, 1, jid, "Grup TI 2024 A (Official)")
	if err != nil {
		t.Fatalf("BindChannel re-bind gagal: %v", err)
	}

	var dName string
	err = svc.db.QueryRowContext(ctx, `SELECT display_name FROM whatsapp_channels WHERE jid = ?;`, jid).Scan(&dName)
	if err != nil || dName != "Grup TI 2024 A (Official)" {
		t.Errorf("Verifikasi display_name gagal: %v, got %s", err, dName)
	}

	// JID tertaut kelas lain tidak boleh pindah diam-diam.
	if err = svc.BindChannel(ctx, 999, jid, "Grup Lain"); err == nil {
		t.Errorf("BindChannel lintas kelas harus ditolak")
	}
}

func TestV1BotService_GetSchedule(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	// 1. Senin reguler (misal: 06 Jan 2025 adalah Senin)
	mondayDate := time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)
	resp, err := svc.GetSchedule(ctx, 1, mondayDate)
	if err != nil {
		t.Fatalf("GetSchedule Senin gagal: %v", err)
	}
	if !strings.Contains(resp, "Algoritma & Pemrograman") || !strings.Contains(resp, "08:00 - 10:00 WIB") {
		t.Errorf("Output GetSchedule tidak sesuai: %s", resp)
	}

	// 2. Hari Minggu libur (12 Jan 2025 adalah Minggu)
	sundayDate := time.Date(2025, 1, 12, 10, 0, 0, 0, time.UTC)
	resp, err = svc.GetSchedule(ctx, 1, sundayDate)
	if err != nil {
		t.Fatalf("GetSchedule Minggu gagal: %v", err)
	}
	if !strings.Contains(resp, "Libur/istirahat") {
		t.Errorf("Expected output libur pada hari Minggu, got: %s", resp)
	}

	// 3. Tambahkan kejadian pengganti (teaching_events) pada hari Selasa (07 Jan 2025)
	tuesdayDate := time.Date(2025, 1, 7, 10, 0, 0, 0, time.UTC)
	tStart := tuesdayDate.Format(time.RFC3339)
	tEnd := tuesdayDate.Add(2 * time.Hour).Format(time.RFC3339)

	var evID int64
	err = svc.db.QueryRowContext(ctx, `
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES (1, '2025-01-06', 'REPLACEMENT', ?, ?, 1,
		          'Kuliah pengganti pertemuan 1', 'PUBLISHED', 1, CURRENT_TIMESTAMP)
		RETURNING id;
	`, tStart, tEnd).Scan(&evID)
	if err != nil {
		t.Fatalf("Insert teaching_events gagal: %v", err)
	}

	_, _ = svc.db.ExecContext(ctx, `
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		)
		VALUES (?, 2, 'OWNER', 'ACCEPTED');
	`, evID)

	resp, err = svc.GetSchedule(ctx, 1, tuesdayDate)
	if err != nil {
		t.Fatalf("GetSchedule Selasa pengganti gagal: %v", err)
	}
	if !strings.Contains(resp, "KULIAH PENGGANTI") || !strings.Contains(resp, "Struktur Data") {
		t.Errorf("Expected output kuliah pengganti, got: %s", resp)
	}
}

func TestV1BotService_GetWeeklySchedule(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	resp, err := svc.GetWeeklySchedule(ctx, 1)
	if err != nil {
		t.Fatalf("GetWeeklySchedule gagal: %v", err)
	}
	if !strings.Contains(resp, "JADWAL KULIAH LENGKAP SEMINGGU") || !strings.Contains(resp, "SENIN") {
		t.Errorf("Output GetWeeklySchedule tidak sesuai: %s", resp)
	}
}

func TestV1BotService_GetNextClass(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	// Senin 06 Jan 2025
	// Kuliah ada di 08:00 - 10:00

	// Kasus 1: Jam 08:30 -> Sedang berlangsung
	during := time.Date(2025, 1, 6, 8, 30, 0, 0, time.UTC)
	resp, err := svc.GetNextClass(ctx, 1, during)
	if err != nil {
		t.Fatalf("GetNextClass during gagal: %v", err)
	}
	if !strings.Contains(resp, "SEDANG BERLANGSUNG") {
		t.Errorf("Expected SEDANG BERLANGSUNG, got: %s", resp)
	}

	// Kasus 2: Jam 07:00 -> Kuliah berikutnya
	before := time.Date(2025, 1, 6, 7, 0, 0, 0, time.UTC)
	resp, err = svc.GetNextClass(ctx, 1, before)
	if err != nil {
		t.Fatalf("GetNextClass before gagal: %v", err)
	}
	if !strings.Contains(resp, "KULIAH BERIKUTNYA") {
		t.Errorf("Expected KULIAH BERIKUTNYA, got: %s", resp)
	}

	// Kasus 3: Jam 11:00 -> Sudah selesai
	after := time.Date(2025, 1, 6, 11, 0, 0, 0, time.UTC)
	resp, err = svc.GetNextClass(ctx, 1, after)
	if err != nil {
		t.Fatalf("GetNextClass after gagal: %v", err)
	}
	if !strings.Contains(resp, "telah selesai") {
		t.Errorf("Expected telah selesai, got: %s", resp)
	}
}

func TestV1BotService_Tasks_And_Links(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	// 1. Sisipkan tugas aktif
	dl := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	var taskID int64
	err := svc.db.QueryRowContext(ctx, `
		INSERT INTO tasks (
			course_offering_id, title, instructions, deadline_at,
			submission_text, publication_status, review_state, reviewed_version,
			created_by_user_id, published_at, version
		) VALUES (1, 'Tugas Algoritma 1', 'Buat flowchart kalkulator', ?,
		          'Kumpulkan melalui PJ', 'PUBLISHED', 'APPROVED', 1,
		          1, CURRENT_TIMESTAMP, 1)
		RETURNING id;
	`, dl).Scan(&taskID)
	if err != nil {
		t.Fatalf("Insert task gagal: %v", err)
	}

	// 2. Baca daftar tugas
	respTasks, err := svc.GetTasks(ctx, 1)
	if err != nil {
		t.Fatalf("GetTasks gagal: %v", err)
	}
	if !strings.Contains(respTasks, "Tugas Algoritma 1") || !strings.Contains(respTasks, "#1") {
		t.Errorf("Output GetTasks tidak sesuai: %s", respTasks)
	}

	// 3. Tandai tugas selesai
	completeResp, err := svc.CompleteTask(ctx, 1, taskID)
	if err != nil {
		t.Fatalf("CompleteTask gagal: %v", err)
	}
	if !strings.Contains(completeResp, "TUGAS SELESAI") {
		t.Errorf("Output CompleteTask tidak sesuai: %s", completeResp)
	}

	// 4. Verifikasi daftar tugas kosong setelah diselesaikan
	respTasksAfter, _ := svc.GetTasks(ctx, 1)
	if !strings.Contains(respTasksAfter, "Tidak ada tugas aktif") {
		t.Errorf("Expected tugas aktif kosong, got: %s", respTasksAfter)
	}

	// 5. Sisipkan materi / tautan penting
	_, _ = svc.db.ExecContext(ctx, `
		INSERT INTO materials (
			class_id, course_offering_id, title, material_type, url,
			description, status, created_by_user_id
		)
		VALUES (1, 1, 'Slide Pertemuan 1', 'DOCUMENT', 'https://drive.google.com/test',
		        'Materi pengantar algoritma', 'ACTIVE', 1);
	`)

	respLinks, err := svc.GetLinks(ctx, 1)
	if err != nil {
		t.Fatalf("GetLinks gagal: %v", err)
	}
	if !strings.Contains(respLinks, "Slide Pertemuan 1") || !strings.Contains(respLinks, "https://drive.google.com/test") {
		t.Errorf("Output GetLinks tidak sesuai: %s", respLinks)
	}
}

func TestV1BotService_HandleScheduleCommand(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	refNow := time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC) // Senin

	// Test !jadwal
	res, err := svc.HandleScheduleCommand(ctx, 1, "!jadwal", refNow)
	if err != nil || !strings.Contains(res, "Algoritma & Pemrograman") {
		t.Errorf("HandleScheduleCommand !jadwal gagal: %v, res: %s", err, res)
	}

	// Test !seminggu
	res, err = svc.HandleScheduleCommand(ctx, 1, "!seminggu", refNow)
	if err != nil || !strings.Contains(res, "JADWAL KULIAH LENGKAP SEMINGGU") {
		t.Errorf("HandleScheduleCommand !seminggu gagal: %v, res: %s", err, res)
	}

	// Test !next
	res, err = svc.HandleScheduleCommand(ctx, 1, "!next", refNow)
	if err != nil || !strings.Contains(res, "SEDANG BERLANGSUNG") {
		t.Errorf("HandleScheduleCommand !next gagal: %v, res: %s", err, res)
	}
}
