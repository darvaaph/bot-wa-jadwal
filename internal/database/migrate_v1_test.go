package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// Seam S1 (disepakati di docs/spec/be-v1-schema.md): perilaku migrasi
// diobservasi lewat database terisolasi, bukan detail SQL internal.

var v1Tables = []string{
	"users", "role_assignments", "role_invitations", "login_attempts",
	"user_sessions", "portal_sessions", "recovery_tokens",
	"classes", "class_settings", "semesters", "courses", "course_offerings",
	"lecturers", "offering_lecturers",
	"rooms", "schedule_patterns", "teaching_events",
	"teaching_event_offerings", "room_confirmations",
	"tasks", "task_reviews", "materials",
	"whatsapp_channels", "notification_messages", "notification_attempts",
	"audit_logs", "import_batches", "import_errors", "backup_records",
}

func openMigratedV1DB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := InitDB(filepath.Join(t.TempDir(), "v1.db"))
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fk); err != nil || fk != 1 {
		t.Fatalf("foreign_keys harus aktif, didapat fk=%d err=%v", fk, err)
	}
	if err := MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}
	return db
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var n string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", name).Scan(&n)
	return err == nil && n == name
}

func TestMigrateV1_MembangunSeluruhTabelTarget(t *testing.T) {
	db, err := InitDB(filepath.Join(t.TempDir(), "v1.db"))
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	if err := MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}
	for _, name := range v1Tables {
		if !tableExists(t, db, name) {
			t.Errorf("Tabel %s tidak dibuat", name)
		}
	}
}

func TestMigrateV1_AmanDijalankanUlang(t *testing.T) {
	db := openMigratedV1DB(t)
	if err := MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 ulang harus idempoten, didapat: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a','TI',2024,'A');`); err != nil {
		t.Fatalf("Tulis setelah migrasi ulang gagal: %v", err)
	}
}

func seedKelasAktif(t *testing.T, db *sql.DB, code, slug string) (classID, semID int64) {
	t.Helper()
	res, err := db.Exec(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES (?,?, 'TI', 2024, 'A');`, code, slug)
	if err != nil {
		t.Fatalf("Gagal membuat Kelas: %v", err)
	}
	classID, _ = res.LastInsertId()
	res, err = db.Exec(`
		INSERT INTO semesters (
			class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at
		) VALUES (?,?, 'Ganjil', '2026-09-01', '2027-01-31', 'ACTIVE', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`, classID, "2026/2027")
	if err != nil {
		t.Fatalf("Gagal membuat Semester aktif: %v", err)
	}
	semID, _ = res.LastInsertId()
	return classID, semID
}

func seedOffering(t *testing.T, db *sql.DB, semID int64, courseCode, courseName, activity string) int64 {
	t.Helper()
	if _, err := db.Exec(`INSERT OR IGNORE INTO courses (code, name) VALUES (?,?);`, courseCode, courseName); err != nil {
		t.Fatalf("Gagal membuat master mata kuliah: %v", err)
	}
	var courseID int64
	if err := db.QueryRow(`SELECT id FROM courses WHERE code = ?;`, courseCode).Scan(&courseID); err != nil {
		t.Fatalf("Gagal membaca master mata kuliah: %v", err)
	}
	res, err := db.Exec(`
		INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name)
		VALUES (?,?,?,?);
	`, semID, courseID, activity, courseName+" ("+activity+")")
	if err != nil {
		t.Fatalf("Gagal membuat offering: %v", err)
	}
	offID, _ := res.LastInsertId()
	return offID
}

func seedPengurus(t *testing.T, db *sql.DB, identity string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO users (identity_key, display_name, password_hash) VALUES (?, 'Pengurus', 'hash');`, identity)
	if err != nil {
		t.Fatalf("Gagal membuat pengurus: %v", err)
	}
	userID, _ := res.LastInsertId()
	return userID
}

func TestMigrateV1_MenolakSemesterAktifKedua(t *testing.T) {
	db := openMigratedV1DB(t)
	classID, _ := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	_, err := db.Exec(`INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status) VALUES (?, '2026/2027', 'Genap', '2027-02-01', '2027-06-30', 'ACTIVE');`, classID)
	if err == nil {
		t.Errorf("Semester ACTIVE kedua untuk satu Kelas harus ditolak")
	}
}

func TestMigrateV1_MenolakTugasYatim(t *testing.T) {
	db := openMigratedV1DB(t)
	_, err := db.Exec(`INSERT INTO tasks (course_offering_id, title, deadline_at, publication_status, review_state, version) VALUES (99999, 'X', '2026-10-10T16:59:00Z', 'DRAFT', 'NOT_REVIEWED', 1);`)
	if err == nil {
		t.Errorf("Tugas tanpa Course Offering harus ditolak foreign key")
	}
}

func TestMigrateV1_MenolakStatusLiar(t *testing.T) {
	db := openMigratedV1DB(t)
	_, err := db.Exec(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status) VALUES ('X','x','TI',2024,'A','SEMAUNYA');`)
	if err == nil {
		t.Errorf("Status Kelas liar harus ditolak batasan")
	}
}

func TestMigrateV1_MenolakOwnerGanda(t *testing.T) {
	db := openMigratedV1DB(t)
	_, semID := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	off1 := seedOffering(t, db, semID, "TI3105", "Pengembangan Web", "TEORI")
	off2 := seedOffering(t, db, semID, "TI3105", "Pengembangan Web", "PRAKTIK")
	res, err := db.Exec(`INSERT INTO teaching_events (event_kind, starts_at, ends_at, lifecycle_status, version) VALUES ('EXTRA','2026-10-05T02:00:00Z','2026-10-05T04:00:00Z','DRAFT',1);`)
	if err != nil {
		t.Fatalf("Gagal membuat kejadian: %v", err)
	}
	evID, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (?,?,'OWNER','ACCEPTED');`, evID, off1); err != nil {
		t.Fatalf("Gagal menautkan OWNER: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (?,?,'OWNER','ACCEPTED');`, evID, off2); err == nil {
		t.Errorf("OWNER kedua untuk satu kejadian harus ditolak")
	}
}

func TestMigrateV1_MenolakKunciIdempotensiGanda(t *testing.T) {
	db := openMigratedV1DB(t)
	classID, _ := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	channelResult, err := db.Exec(`
		INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status)
		VALUES (?, '120363000000000001@g.us', 'GROUP', 'Kelas Uji', 'ACTIVE');
	`, classID)
	if err != nil {
		t.Fatalf("Gagal membuat kanal notifikasi: %v", err)
	}
	channelID, _ := channelResult.LastInsertId()
	if _, err := db.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at
		) VALUES (?, ?, 'SCHEDULE_PUBLISHED', 'TEACHING_EVENT', 1, 'kunci-1', '{}', 'PENDING', CURRENT_TIMESTAMP);
	`, classID, channelID); err != nil {
		t.Fatalf("Gagal membuat notifikasi: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at
		) VALUES (?, ?, 'SCHEDULE_PUBLISHED', 'TEACHING_EVENT', 1, 'kunci-1', '{}', 'PENDING', CURRENT_TIMESTAMP);
	`, classID, channelID); err == nil {
		t.Errorf("Kunci idempotensi ganda harus ditolak")
	}
}

func TestMigrateV1_MenolakPolaWaktuTerbalik(t *testing.T) {
	db := openMigratedV1DB(t)
	_, semID := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	offID := seedOffering(t, db, semID, "TI3105", "Pengembangan Web", "TEORI")
	_, err := db.Exec(`INSERT INTO schedule_patterns (course_offering_id, day_of_week, start_time, end_time, version) VALUES (?, 1, '10:00', '08:00', 1);`, offID)
	if err == nil {
		t.Errorf("Pola dengan selesai <= mulai harus ditolak")
	}
}

func TestMigrateV1_MenolakKombinasiScopeSalah(t *testing.T) {
	db := openMigratedV1DB(t)
	classID, semID := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	offID := seedOffering(t, db, semID, "TI3105", "Pengembangan Web", "TEORI")
	userID := seedPengurus(t, db, "+628120001")

	cases := []struct {
		nama  string
		role  string
		scope string
		class any
		sem   any
		off   any
	}{
		{"PJ scope kelas", "PJ", "CLASS", classID, nil, nil},
		{"PJ tanpa offering", "PJ", "COURSE_OFFERING", classID, semID, nil},
		{"KM scope offering", "KM", "COURSE_OFFERING", classID, semID, offID},
		{"KM tanpa kelas", "KM", "CLASS", nil, nil, nil},
		{"Admin global berkela", "SYSTEM_ADMIN", "GLOBAL", classID, nil, nil},
	}
	for _, c := range cases {
		_, err := db.Exec(`INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id) VALUES (?,?,?,?,?,?);`,
			userID, c.role, c.scope, c.class, c.sem, c.off)
		if err == nil {
			t.Errorf("Kombinasi scope salah (%s) harus ditolak", c.nama)
		}
	}

	if _, err := db.Exec(`INSERT INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id) VALUES (?, 'PJ', 'COURSE_OFFERING', ?, ?, ?);`,
		userID, classID, semID, offID); err != nil {
		t.Errorf("Kombinasi scope PJ yang benar harus lolos: %v", err)
	}
}

func TestMigrateV1_MenolakPenugasanAktifGanda(t *testing.T) {
	db := openMigratedV1DB(t)
	classID, _ := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	userID := seedPengurus(t, db, "+628120001")
	if _, err := db.Exec(`INSERT INTO role_assignments (user_id, role, scope_type, class_id) VALUES (?, 'KM', 'CLASS', ?);`, userID, classID); err != nil {
		t.Fatalf("Gagal membuat penugasan KM: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO role_assignments (user_id, role, scope_type, class_id) VALUES (?, 'KM', 'CLASS', ?);`, userID, classID); err == nil {
		t.Errorf("Penugasan KM aktif ganda untuk kombinasi sama harus ditolak")
	}
}

func TestMigrateV1_MenolakHapusLunakTanpaPelaku(t *testing.T) {
	db := openMigratedV1DB(t)
	_, semID := seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	offID := seedOffering(t, db, semID, "TI3105", "Pengembangan Web", "TEORI")
	_, err := db.Exec(`INSERT INTO tasks (course_offering_id, title, deadline_at, deleted_at) VALUES (?, 'X', '2026-10-10T16:59:00Z', '2026-09-28T00:00:00Z');`, offID)
	if err == nil {
		t.Errorf("Hapus lunak Tugas tanpa pelaku harus ditolak")
	}
	_, err = db.Exec(`INSERT INTO materials (class_id, title, deleted_at) VALUES (1, 'Y', '2026-09-28T00:00:00Z');`)
	if err == nil {
		t.Errorf("Hapus lunak Materi tanpa pelaku harus ditolak")
	}
}

func TestMigrateV1_MenolakIdentitasKelasGanda(t *testing.T) {
	db := openMigratedV1DB(t)
	seedKelasAktif(t, db, "D4-TI-2024-A", "d4-ti-2024-a")
	_, err := db.Exec(`INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('LAIN','lain','TI',2024,'A');`)
	if err == nil {
		t.Errorf("Identitas Kelas ganda (prodi+angkatan+rombel) harus ditolak")
	}
}
