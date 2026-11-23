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

	// 2b. Verifikasi GetDailySummary (Pengingat otomatis harus senyap saat libur)
	summarySunday, err := svc.GetDailySummary(ctx, 1, sundayDate)
	if err != nil {
		t.Fatalf("GetDailySummary Minggu gagal: %v", err)
	}
	if summarySunday != "" {
		t.Errorf("Expected GetDailySummary Minggu kosong (senyap), got: %s", summarySunday)
	}

	summaryMonday, err := svc.GetDailySummary(ctx, 1, mondayDate)
	if err != nil {
		t.Fatalf("GetDailySummary Senin gagal: %v", err)
	}
	if !strings.Contains(summaryMonday, "Algoritma & Pemrograman") {
		t.Errorf("Expected GetDailySummary Senin memuat jadwal aktif, got: %s", summaryMonday)
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

func TestV1BotService_ResolveDailyItems_RespectsEffectiveDates(t *testing.T) {
	svc := setupTestV1DB(t)
	if _, err := svc.db.Exec(`UPDATE schedule_patterns SET effective_from='2025-01-06', effective_until='2025-01-13' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		date  string
		count int
	}{{"2024-12-30", 0}, {"2025-01-06", 1}, {"2025-01-13", 1}, {"2025-01-20", 0}} {
		day, _ := time.Parse("2006-01-02", tc.date)
		_, _, _, _, items, err := svc.ResolveDailyItems(context.Background(), 1, day)
		if err != nil || len(items) != tc.count {
			t.Errorf("date=%s items=%d want=%d err=%v", tc.date, len(items), tc.count, err)
		}
	}
}

func TestV1BotService_ResolveDailyItems_UsesClassDayForUTCEvent(t *testing.T) {
	svc := setupTestV1DB(t)
	_, err := svc.db.Exec(`INSERT INTO teaching_events (event_kind,starts_at,ends_at,lifecycle_status,published_by_user_id,published_at)
		VALUES ('EXTRA','2025-01-05T18:00:00Z','2025-01-05T19:00:00Z','PUBLISHED',1,CURRENT_TIMESTAMP);
		INSERT INTO teaching_event_offerings (teaching_event_id,course_offering_id,participation_role,participation_status)
		VALUES (1,1,'OWNER','ACCEPTED')`)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	day := time.Date(2025, 1, 6, 0, 0, 0, 0, loc)
	_, _, _, _, items, err := svc.ResolveDailyItems(context.Background(), 1, day)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		if item.Status == "EXTRA" && item.StartTime == "01:00" {
			found = true
		}
	}
	if !found {
		t.Fatalf("acara 01:00 WIB tidak ada: %+v", items)
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
	loc, _ := time.LoadLocation("Asia/Jakarta")

	// Kasus 1: Jam 08:30 -> Sedang berlangsung
	during := time.Date(2025, 1, 6, 8, 30, 0, 0, loc)
	resp, err := svc.GetNextClass(ctx, 1, during)
	if err != nil {
		t.Fatalf("GetNextClass during gagal: %v", err)
	}
	if !strings.Contains(resp, "SEDANG BERLANGSUNG") {
		t.Errorf("Expected SEDANG BERLANGSUNG, got: %s", resp)
	}

	// Kasus 2: Jam 07:00 -> Kuliah berikutnya
	before := time.Date(2025, 1, 6, 7, 0, 0, 0, loc)
	resp, err = svc.GetNextClass(ctx, 1, before)
	if err != nil {
		t.Fatalf("GetNextClass before gagal: %v", err)
	}
	if !strings.Contains(resp, "KULIAH BERIKUTNYA") {
		t.Errorf("Expected KULIAH BERIKUTNYA, got: %s", resp)
	}

	// Kasus 3: Jam 11:00 -> Sudah selesai
	after := time.Date(2025, 1, 6, 11, 0, 0, 0, loc)
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
	if !strings.Contains(respTasks, "Tugas Algoritma 1") {
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

func TestV1BotService_Tasks_ExcludesPastDeadline(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	// Tugas yang sudah lewat tenggat (2 jam yang lalu)
	dlPast := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	_, err := svc.db.ExecContext(ctx, `
		INSERT INTO tasks (
			course_offering_id, title, instructions, deadline_at,
			submission_text, publication_status, review_state, reviewed_version,
			created_by_user_id, published_at, version
		) VALUES (1, 'Tugas Kadaluarsa', 'Sudah lewat tenggat', ?,
		          'Kumpul di Kelas', 'PUBLISHED', 'APPROVED', 1,
		          1, CURRENT_TIMESTAMP, 1);
	`, dlPast)
	if err != nil {
		t.Fatalf("Insert past task gagal: %v", err)
	}

	// Tugas yang masih aktif (24 jam ke depan)
	dlFuture := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	_, err = svc.db.ExecContext(ctx, `
		INSERT INTO tasks (
			course_offering_id, title, instructions, deadline_at,
			submission_text, publication_status, review_state, reviewed_version,
			created_by_user_id, published_at, version
		) VALUES (1, 'Tugas Aktif Masa Depan', 'Masih berlaku', ?,
		          'Kumpul di Portal', 'PUBLISHED', 'APPROVED', 1,
		          1, CURRENT_TIMESTAMP, 1);
	`, dlFuture)
	if err != nil {
		t.Fatalf("Insert future task gagal: %v", err)
	}

	resp, err := svc.GetTasks(ctx, 1)
	if err != nil {
		t.Fatalf("GetTasks gagal: %v", err)
	}

	if strings.Contains(resp, "Tugas Kadaluarsa") {
		t.Errorf("Tugas yang sudah lewat tenggat seharusnya tidak dimunculkan di WhatsApp: %s", resp)
	}
	if !strings.Contains(resp, "Tugas Aktif Masa Depan") {
		t.Errorf("Tugas yang masih aktif harus dimunculkan di WhatsApp: %s", resp)
	}
}

func TestV1BotService_GetCoursePJs(t *testing.T) {
	t.Run("single, multiple, and unassigned PJ with deduplication", func(t *testing.T) {
		svc := setupTestV1DB(t)
		ctx := context.Background()
		_, err := svc.db.ExecContext(ctx, `
			INSERT INTO courses (id, code, name, status)
			VALUES (3, 'TI2103', 'Basis Data', 'ACTIVE');

			INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name, status)
			VALUES (3, 1, 3, 'THEORY', 'Basis Data (Teori)', 'ACTIVE');

			INSERT INTO users (id, identity_key, display_name, password_hash, status)
			VALUES
				(2, 'private:pj-2', 'Budi Santoso', 'disabled', 'ACTIVE'),
				(3, 'private:pj-3', 'Sinta Dewi', 'disabled', 'ACTIVE'),
				(4, 'private:pj-4', 'Andi Pratama', 'disabled', 'ACTIVE'),
				(5, 'private:pj-5', 'Sinta Dewi', 'disabled', 'ACTIVE');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id)
			VALUES
				(2, 'PJ', 'COURSE_OFFERING', 1, 1, 1),
				(3, 'PJ', 'COURSE_OFFERING', 1, 1, 2),
				(4, 'PJ', 'COURSE_OFFERING', 1, 1, 2),
				(5, 'PJ', 'COURSE_OFFERING', 1, 1, 2);
		`)
		if err != nil {
			t.Fatalf("seed PJ gagal: %v", err)
		}

		resp, err := svc.GetCoursePJs(ctx, 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs gagal: %v", err)
		}
		for _, want := range []string{
			"*DAFTAR PJ MATA KULIAH",
			"1. *Algoritma & Pemrograman (Teori)*\n   - PJ: Budi Santoso",
			"2. *Basis Data (Teori)*\n   - PJ: Belum ditetapkan",
			"3. *Struktur Data (Praktik)*\n   - PJ: Andi Pratama, Sinta Dewi",
			"\n─────────────\nDetail jadwal dan tugas:\n",
		} {
			if !strings.Contains(resp, want) {
				t.Errorf("GetCoursePJs tidak memuat %q:\n%s", want, resp)
			}
		}
		for _, privateValue := range []string{"private:pj-2", "private:pj-3", "private:pj-4", "private:pj-5"} {
			if strings.Contains(resp, privateValue) {
				t.Errorf("balasan membocorkan identity_key %q: %s", privateValue, resp)
			}
		}
	})

	t.Run("unassigned, inactive, revoked, future valid_from, and expired PJ handling", func(t *testing.T) {
		svc := setupTestV1DB(t)
		ctx := context.Background()
		_, err := svc.db.ExecContext(ctx, `
			INSERT INTO users (id, identity_key, display_name, password_hash, status)
			VALUES
				(2, 'private:suspended-user', 'PJ Nonaktif', 'disabled', 'SUSPENDED'),
				(3, 'private:expired-pj', 'PJ Kedaluwarsa', 'disabled', 'ACTIVE'),
				(4, 'private:revoked-user', 'PJ Dicabut', 'disabled', 'REVOKED'),
				(5, 'private:suspended-role', 'PJ Role Nonaktif', 'disabled', 'ACTIVE'),
				(6, 'private:future-role', 'PJ Masa Depan', 'disabled', 'ACTIVE'),
				(7, 'private:active-future-until', 'PJ Masih Berlaku', 'disabled', 'ACTIVE');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, valid_from)
			VALUES (2, 'PJ', 'COURSE_OFFERING', 1, 1, 1, '2020-01-01');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, valid_from, valid_until)
			VALUES (3, 'PJ', 'COURSE_OFFERING', 1, 1, 1, '2020-01-01', '2021-01-01');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, valid_from)
			VALUES (4, 'PJ', 'COURSE_OFFERING', 1, 1, 1, '2020-01-01');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, status, valid_from)
			VALUES (5, 'PJ', 'COURSE_OFFERING', 1, 1, 1, 'SUSPENDED', '2020-01-01');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, valid_from)
			VALUES (6, 'PJ', 'COURSE_OFFERING', 1, 1, 1, '2099-01-01');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, valid_from, valid_until)
			VALUES (7, 'PJ', 'COURSE_OFFERING', 1, 1, 2, '2020-01-01', '2099-01-01');
		`)
		if err != nil {
			t.Fatalf("seed PJ tidak aktif gagal: %v", err)
		}

		resp, err := svc.GetCoursePJs(ctx, 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs gagal: %v", err)
		}
		if strings.Count(resp, "Belum ditetapkan") != 1 {
			t.Errorf("hanya mata kuliah 1 yang seharusnya Belum ditetapkan: %s", resp)
		}
		if !strings.Contains(resp, "PJ: PJ Masih Berlaku") {
			t.Errorf("PJ dengan valid_until masa depan harus tetap tampil: %s", resp)
		}
		for _, name := range []string{"PJ Nonaktif", "PJ Kedaluwarsa", "PJ Dicabut", "PJ Role Nonaktif", "PJ Masa Depan"} {
			if strings.Contains(resp, name) {
				t.Errorf("PJ tidak aktif/belum berlaku %q tidak boleh ditampilkan: %s", name, resp)
			}
		}
	})

	t.Run("masks display_name fallback matching identity_key", func(t *testing.T) {
		svc := setupTestV1DB(t)
		ctx := context.Background()
		_, err := svc.db.ExecContext(ctx, `
			INSERT INTO users (id, identity_key, display_name, password_hash, status)
			VALUES (2, '+6281234567890', '+6281234567890', 'disabled', 'ACTIVE');

			INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id)
			VALUES (2, 'PJ', 'COURSE_OFFERING', 1, 1, 1);
		`)
		if err != nil {
			t.Fatalf("seed PJ fallback gagal: %v", err)
		}

		resp, err := svc.GetCoursePJs(ctx, 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs gagal: %v", err)
		}
		if strings.Contains(resp, "+6281234567890") {
			t.Errorf("nomor telepon/identity_key tidak boleh bocor meski tersimpan di display_name: %s", resp)
		}
		if !strings.Contains(resp, "PJ (nama belum diatur)") {
			t.Errorf("harus menampilkan placeholder nama aman saat display_name == identity_key: %s", resp)
		}
	})

	t.Run("keyword filters offering name course name code and fallback ordering", func(t *testing.T) {
		svc := setupTestV1DB(t)
		if _, err := svc.db.Exec(`
			INSERT INTO courses (id, code, name, status)
			VALUES
				(3, 'TI2103', 'Arsitektur Komputer', 'ACTIVE'),
				(4, 'TI2104', 'Matkul Nonaktif', 'ACTIVE');

			INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name, status)
			VALUES
				(3, 1, 3, 'THEORY', '', 'ACTIVE'),
				(4, 1, 4, 'THEORY', 'Matkul Arsip', 'ARCHIVED');

			UPDATE course_offerings SET display_name = 'Praktik Kelas' WHERE id = 2;
		`); err != nil {
			t.Fatalf("ubah/tambah offering gagal: %v", err)
		}

		fullResp, err := svc.GetCoursePJs(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs full gagal: %v", err)
		}
		if strings.Contains(fullResp, "Matkul Arsip") {
			t.Errorf("course_offering ARCHIVED tidak boleh tampil: %s", fullResp)
		}
		idxAlgo := strings.Index(fullResp, "Algoritma & Pemrograman (Teori)")
		idxArsi := strings.Index(fullResp, "Arsitektur Komputer")
		idxPrak := strings.Index(fullResp, "Praktik Kelas")
		if !(idxAlgo >= 0 && idxArsi > idxAlgo && idxPrak > idxArsi) {
			t.Errorf("urutan fallback c.name tidak urut abjad (Algo < Arsi < Prak):\n%s", fullResp)
		}

		for _, tc := range []struct {
			query string
			want  string
			omit  string
		}{
			{query: "Praktik", want: "Praktik Kelas", omit: "Algoritma"},
			{query: "Struktur Data", want: "Praktik Kelas", omit: "Algoritma"},
			{query: "TI2102", want: "Praktik Kelas", omit: "Algoritma"},
		} {
			resp, err := svc.GetCoursePJs(context.Background(), 1, tc.query)
			if err != nil {
				t.Fatalf("GetCoursePJs query %q gagal: %v", tc.query, err)
			}
			if !strings.Contains(resp, tc.want) || strings.Contains(resp, tc.omit) {
				t.Errorf("filter %q menghasilkan daftar tidak sesuai: %s", tc.query, resp)
			}
		}

		if _, err := svc.db.Exec(`UPDATE course_offerings SET display_name = 'Aljabar Linear (Teori)' WHERE id = 2;`); err != nil {
			t.Fatalf("ubah nama matkul gagal: %v", err)
		}
		resp, err := svc.GetCoursePJs(context.Background(), 1, "aljabar linear teori")
		if err != nil {
			t.Fatalf("GetCoursePJs dengan kata kunci terpisah gagal: %v", err)
		}
		if !strings.Contains(resp, "Aljabar Linear (Teori)") || strings.Contains(resp, "Algoritma") {
			t.Errorf("kata kunci terpisah harus cocok dengan nama matkul ber-tanda kurung: %s", resp)
		}

		if _, err := svc.db.Exec(`UPDATE course_offerings SET display_name = 'Komunikasi Data dan Jaringan' WHERE id = 2;`); err != nil {
			t.Fatalf("ubah nama matkul gagal: %v", err)
		}
		resp, err = svc.GetCoursePJs(context.Background(), 1, "komunikasi data dan jaringan")
		if err != nil {
			t.Fatalf("GetCoursePJs untuk komunikasi data dan jaringan gagal: %v", err)
		}
		if !strings.Contains(resp, "Komunikasi Data dan Jaringan") || strings.Contains(resp, "Algoritma") {
			t.Errorf("query komunikasi data dan jaringan seharusnya hanya cocok dengan matkul tersebut: %s", resp)
		}

		noMatchResp, err := svc.GetCoursePJs(context.Background(), 1, "TidakAda123")
		if err != nil {
			t.Fatalf("GetCoursePJs no match gagal: %v", err)
		}
		if !strings.Contains(noMatchResp, "Tidak ada mata kuliah yang cocok") || strings.Contains(noMatchResp, ".─") {
			t.Errorf("format pesan filter kosong tidak sesuai/menempel separator: %q", noMatchResp)
		}
	})

	t.Run("no active offerings in active semester", func(t *testing.T) {
		svc := setupTestV1DB(t)
		if _, err := svc.db.Exec(`UPDATE course_offerings SET status = 'ARCHIVED' WHERE semester_id = 1;`); err != nil {
			t.Fatalf("arsipkan offerings gagal: %v", err)
		}
		resp, err := svc.GetCoursePJs(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs gagal: %v", err)
		}
		if !strings.Contains(resp, "Belum ada mata kuliah yang terdaftar untuk semester aktif ini.") || strings.Contains(resp, ".─") {
			t.Errorf("pesan offerings kosong tidak sesuai/menempel separator: %q", resp)
		}
	})

	t.Run("no active semester", func(t *testing.T) {
		svc := setupTestV1DB(t)
		if _, err := svc.db.Exec(`UPDATE semesters SET status = 'ARCHIVED', archived_at = CURRENT_TIMESTAMP WHERE id = 1;`); err != nil {
			t.Fatalf("arsipkan semester gagal: %v", err)
		}
		resp, err := svc.GetCoursePJs(context.Background(), 1, "")
		if err != nil {
			t.Fatalf("GetCoursePJs gagal: %v", err)
		}
		if !strings.Contains(resp, "Belum ada semester aktif") {
			t.Errorf("harus menyatakan semester aktif belum tersedia: %s", resp)
		}
	})

	t.Run("menu and keywords include !pj", func(t *testing.T) {
		svc := setupTestV1DB(t)
		ctx := context.Background()
		menu, err := svc.GetMenu(ctx, 1)
		if err != nil || !strings.Contains(menu, "!pj") {
			t.Errorf("GetMenu harus memuat !pj (err=%v): %s", err, menu)
		}
		kw, err := svc.GetKeywords(ctx, 1)
		if err != nil || !strings.Contains(kw, "!pj") {
			t.Errorf("GetKeywords harus memuat !pj (err=%v): %s", err, kw)
		}
	})
}

func TestV1BotService_HandleScheduleCommand(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	loc, _ := time.LoadLocation("Asia/Jakarta")
	refNow := time.Date(2025, 1, 6, 9, 0, 0, 0, loc) // Senin

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

func TestFormatClassDisplayName(t *testing.T) {
	label := formatClassDisplayName("D4-TI-2025-A", "D4 Teknik Informatika", 2025, "A", "2026/2027")
	expected := "D4 TEKNIK INFORMATIKA 2A (Angkatan 2025)"
	if label != expected {
		t.Errorf("formatClassDisplayName gagal: got %q, want %q", label, expected)
	}

	labelD3 := formatClassDisplayName("D3-TI-2024-C", "D3 Teknik Informatika", 2024, "C", "2026/2027")
	expectedD3 := "D3 TEKNIK INFORMATIKA 3C (Angkatan 2024)"
	if labelD3 != expectedD3 {
		t.Errorf("formatClassDisplayName D3 gagal: got %q, want %q", labelD3, expectedD3)
	}
}

func TestAntiSlop_NoDecorativeEmojis(t *testing.T) {
	svc := setupTestV1DB(t)
	ctx := context.Background()

	mondayDate := time.Date(2025, 1, 6, 9, 0, 0, 0, time.UTC)
	resp, err := svc.GetSchedule(ctx, 1, mondayDate)
	if err != nil {
		t.Fatalf("GetSchedule gagal: %v", err)
	}

	slopEmojis := []string{"📅", "📌", "⏰", "🏢", "👨‍🏫", "🎉", "💬", "📋", "📚", "⏳", "📝", "🔗", "📍"}
	for _, emoji := range slopEmojis {
		if strings.Contains(resp, emoji) {
			t.Errorf("Output GetSchedule masih memuat emoji AI-slop %s: %s", emoji, resp)
		}
	}

	if !strings.Contains(resp, "- Waktu:") || !strings.Contains(resp, "- Ruang:") {
		t.Errorf("Output GetSchedule harus memuat format bullet strip (-): %s", resp)
	}

	// Verifikasi GetTasks bebas emoji dekoratif dan menggunakan strip
	dl := time.Now().Add(48 * time.Hour).Format(time.RFC3339)
	_, err = svc.db.ExecContext(ctx, `
		INSERT INTO tasks (
			course_offering_id, title, instructions, deadline_at,
			submission_text, publication_status, review_state, reviewed_version,
			created_by_user_id, published_at, version
		) VALUES (1, 'Tugas Algoritma AntiSlop', 'Buat laporan praktikum', ?,
		          'Kumpul di Portal', 'PUBLISHED', 'APPROVED', 1,
		          1, CURRENT_TIMESTAMP, 1);
	`, dl)
	if err != nil {
		t.Fatalf("Insert task gagal: %v", err)
	}

	respTasks, err := svc.GetTasks(ctx, 1)
	if err != nil {
		t.Fatalf("GetTasks gagal: %v", err)
	}

	for _, emoji := range slopEmojis {
		if strings.Contains(respTasks, emoji) {
			t.Errorf("Output GetTasks masih memuat emoji AI-slop %s: %s", emoji, respTasks)
		}
	}

	if !strings.Contains(respTasks, "Tugas Algoritma AntiSlop") || (!strings.Contains(respTasks, "├") && !strings.Contains(respTasks, "└")) {
		t.Errorf("Output GetTasks harus memuat judul dan konektor pohon (├/└): %s", respTasks)
	}
	if !strings.Contains(respTasks, "Kumpul di Portal") {
		t.Errorf("Output GetTasks harus memuat teks pengumpulan: %s", respTasks)
	}
	if !strings.Contains(respTasks, "(T)") {
		t.Errorf("Output GetTasks harus memuat badge singkat (T): %s", respTasks)
	}

	// Verifikasi tugas bertipe Kuis menampilkan [KUIS] dan Pelaksanaan:
	_, err = svc.db.ExecContext(ctx, `
		INSERT INTO tasks (
			course_offering_id, title, task_type, instructions, deadline_at,
			submission_text, publication_status, review_state, reviewed_version,
			created_by_user_id, published_at, version
		) VALUES (1, 'Materi Bab 1-3', 'Kuis', 'Closed book', ?,
		          'Di Kelas D105', 'PUBLISHED', 'APPROVED', 1,
		          1, CURRENT_TIMESTAMP, 1);
	`, dl)
	if err != nil {
		t.Fatalf("Insert quiz task gagal: %v", err)
	}

	respQuiz, err := svc.GetTasks(ctx, 1)
	if err != nil {
		t.Fatalf("GetTasks kuis gagal: %v", err)
	}
	if !strings.Contains(respQuiz, "[KUIS] Materi Bab 1-3") {
		t.Errorf("Output GetTasks untuk kuis harus memuat [KUIS] Materi Bab 1-3: %s", respQuiz)
	}
	if !strings.Contains(respQuiz, "Pelaksanaan: Di Kelas D105") {
		t.Errorf("Output GetTasks untuk kuis harus memuat Pelaksanaan: Di Kelas D105: %s", respQuiz)
	}
}
