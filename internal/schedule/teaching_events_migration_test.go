package schedule

import (
	"bot-jadwal/internal/database"
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestTeachingEventsMigration_ForeignKeysAndModelTarget(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database in-memory: %v", err)
	}
	defer db.Close()

	// Pastikan foreign keys aktif
	var fk int
	err = db.QueryRow("PRAGMA foreign_keys;").Scan(&fk)
	if err != nil || fk != 1 {
		t.Fatalf("PRAGMA foreign_keys tidak aktif: %v", err)
	}

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi OverrideManager: %v", err)
	}

	groupJID := "120363002@g.us"
	userJID := "628999999@s.whatsapp.net"
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	targetDate := now.Add(24 * time.Hour)

	item := JadwalItem{
		KodeMatkul:   "TI201",
		NamaMatkul:   "Pemrograman Web",
		Dosen:        "Budi Santoso",
		InisialDosen: "BS",
		Jam:          "08:00 - 09:40",
		Ruang:        "Lab 101",
	}

	// 1. Uji Tambah Reschedule (REPLACEMENT)
	resched, err := om.AddReschedule(groupJID, item, now, targetDate, "10:00 - 11:40", "Lab 102", userJID)
	if err != nil {
		t.Fatalf("AddReschedule gagal: %v", err)
	}
	if resched.Type != "RESCHEDULE" || resched.NewJam != "10:00 - 11:40" || resched.Ruang != "Lab 102" {
		t.Errorf("Hasil Reschedule tidak sesuai: %+v", resched)
	}

	// Verifikasi record pada target teaching_events & teaching_event_offerings
	var eventID int64
	var eventKind, lifecycleStatus string
	var originPatternID sql.NullInt64
	var roomID sql.NullInt64
	err = db.QueryRow(`
		SELECT id, event_kind, lifecycle_status, origin_schedule_pattern_id, room_id
		FROM teaching_events WHERE id = ?
	`, resched.ID).Scan(&eventID, &eventKind, &lifecycleStatus, &originPatternID, &roomID)
	if err != nil {
		t.Fatalf("teaching_events tidak ditemukan untuk event ID %d: %v", resched.ID, err)
	}
	if eventKind != "REPLACEMENT" || lifecycleStatus != "PUBLISHED" {
		t.Errorf("teaching_events kind/status tidak valid: %s / %s", eventKind, lifecycleStatus)
	}
	if !originPatternID.Valid || !roomID.Valid {
		t.Errorf("originPatternID atau roomID harus valid untuk REPLACEMENT")
	}

	// Verifikasi teaching_event_offerings
	var offeringID int64
	var role, pStatus string
	err = db.QueryRow(`
		SELECT course_offering_id, participation_role, participation_status
		FROM teaching_event_offerings WHERE teaching_event_id = ?
	`, eventID).Scan(&offeringID, &role, &pStatus)
	if err != nil {
		t.Fatalf("teaching_event_offerings tidak ditemukan: %v", err)
	}
	if role != "OWNER" || pStatus != "ACCEPTED" {
		t.Errorf("teaching_event_offerings role/status tidak valid: %s / %s", role, pStatus)
	}

	// 2. Uji Tambah Cancel (SESSION_CANCELLED)
	cancelOverride, err := om.AddCancel(groupJID, item, now, "Dosen rapat fakultas", userJID)
	if err != nil {
		t.Fatalf("AddCancel gagal: %v", err)
	}
	if cancelOverride.Type != "CANCEL" || cancelOverride.Alasan != "Dosen rapat fakultas" {
		t.Errorf("Hasil AddCancel tidak valid: %+v", cancelOverride)
	}

	// 3. Uji Tambah Extra (EXTRA)
	extraOverride, err := om.AddExtra(groupJID, item, targetDate, "13:00 - 15:30", "Lab 103", "Kuliah tambahan persiapan UTS", userJID)
	if err != nil {
		t.Fatalf("AddExtra gagal: %v", err)
	}
	if extraOverride.Type != "EXTRA" || extraOverride.NewJam != "13:00 - 15:30" {
		t.Errorf("Hasil AddExtra tidak valid: %+v", extraOverride)
	}

	// 4. Uji Tambah Libur (HOLIDAY)
	holidayDate := now.Add(48 * time.Hour)
	holidayOverride, err := om.AddHoliday(groupJID, holidayDate, "Hari Libur Nasional", userJID)
	if err != nil {
		t.Fatalf("AddHoliday gagal: %v", err)
	}
	if holidayOverride.Type != "HOLIDAY" {
		t.Errorf("Hasil AddHoliday tidak valid: %+v", holidayOverride)
	}

	// 5. Uji Pembacaan Overrides untuk Tanggal Tertentu
	overridesForNow, err := om.GetOverridesForDate(groupJID, now)
	if err != nil {
		t.Fatalf("GetOverridesForDate gagal: %v", err)
	}
	if len(overridesForNow) < 2 {
		t.Errorf("Diharapkan minimal 2 overrides pada tanggal now (RESCHEDULE asal & CANCEL), didapat %d", len(overridesForNow))
	}

	// 6. Uji GetHolidayOverride
	hol := om.GetHolidayOverride(groupJID, holidayDate)
	if hol == nil || hol.Alasan != "Hari Libur Nasional" {
		t.Errorf("GetHolidayOverride gagal mendeteksi libur: %+v", hol)
	}

	// 7. Uji Pembatalan Override (CancelOverride -> REVOKED)
	ok, err := om.CancelOverride(groupJID, resched.ID)
	if err != nil || !ok {
		t.Fatalf("CancelOverride gagal: %v (ok=%v)", err, ok)
	}

	var newLifecycle, revReason string
	var revUserID sql.NullInt64
	err = db.QueryRow(`
		SELECT lifecycle_status, COALESCE(revocation_reason, ''), revoked_by_user_id
		FROM teaching_events WHERE id = ?
	`, resched.ID).Scan(&newLifecycle, &revReason, &revUserID)
	if err != nil {
		t.Fatalf("Gagal membaca status REVOKED: %v", err)
	}
	if newLifecycle != "REVOKED" || !revUserID.Valid || revReason == "" {
		t.Errorf("Check constraint REVOKED tidak terpenuhi: lifecycle=%s, reason=%s", newLifecycle, revReason)
	}

	// Setelah dibatalkan, resched tidak boleh muncul lagi di GetOverridesForDate
	overridesAfterCancel, _ := om.GetOverridesForDate(groupJID, now)
	for _, o := range overridesAfterCancel {
		if o.ID == resched.ID {
			t.Errorf("Override yang telah di-REVOKED masih muncul pada query aktif!")
		}
	}
}

func TestTeachingEventsMigration_LegacyBackfill(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	// Buat tabel legacy dan isi data lama
	_, err = db.Exec(`
		CREATE TABLE schedule_overrides (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_jid TEXT NOT NULL,
			override_type TEXT NOT NULL,
			kode_matkul TEXT NOT NULL,
			nama_matkul TEXT NOT NULL,
			dosen TEXT NOT NULL,
			inisial_dosen TEXT NOT NULL,
			orig_date TEXT NOT NULL,
			orig_jam TEXT NOT NULL,
			target_date TEXT NOT NULL,
			new_jam TEXT NOT NULL,
			ruang TEXT NOT NULL,
			alasan TEXT NOT NULL,
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO schedule_overrides (
			scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
			orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by
		) VALUES
		('legacy-group@g.us', 'RESCHEDULE', 'CS101', 'Algoritma', 'Dr. John', 'DJ', '2026-10-01', '08:00 - 09:40', '2026-10-02', '13:00 - 14:40', 'Lab 2', 'Pindah jadwal', 'admin@s.whatsapp.net'),
		('legacy-group@g.us', 'CANCEL', 'CS102', 'Struktur Data', 'Dr. Jane', 'JA', '2026-10-03', '10:00 - 11:40', '2026-10-03', '', 'Lab 1', 'Dosen sakit', 'admin@s.whatsapp.net'),
		('legacy-group@g.us', 'EXTRA', 'CS103', 'Basis Data', 'Dr. Bob', 'BO', '', '', '2026-10-04', '09:00 - 11:00', 'Lab 3', 'Tambahan materi', 'admin@s.whatsapp.net');
	`)
	if err != nil {
		t.Fatalf("Gagal membuat data legacy: %v", err)
	}

	// Inisialisasi OverrideManager baru yang akan memicu backfill otomatis
	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewOverrideManager: %v", err)
	}

	// Verifikasi apakah 3 baris legacy telah berhasil dimigrasikan ke teaching_events
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM teaching_events WHERE lifecycle_status = 'PUBLISHED'").Scan(&count)
	if err != nil {
		t.Fatalf("Gagal menghitung teaching_events: %v", err)
	}
	if count != 3 {
		t.Errorf("Diharapkan 3 teaching_events hasil backfill, didapat %d", count)
	}

	// Verifikasi apakah query GetOverridesForDate membaca data yang dimigrasikan
	dateResched, _ := time.Parse("2006-01-02", "2026-10-01")
	overrides, err := om.GetOverridesForDate("legacy-group@g.us", dateResched)
	if err != nil {
		t.Fatalf("GetOverridesForDate gagal: %v", err)
	}
	if len(overrides) == 0 {
		t.Fatalf("Overrides hasil migrasi tidak terbaca pada tanggal 2026-10-01")
	}
	if overrides[0].KodeMatkul != "CS101" || overrides[0].Type != "RESCHEDULE" {
		t.Errorf("Data override termigrasi tidak cocok: %+v", overrides[0])
	}
}

func TestTeachingEventsMigration_NoRuntimeLegacyUsage(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi: %v", err)
	}

	now := time.Now()
	item := JadwalItem{
		KodeMatkul: "TEST1",
		NamaMatkul: "Uji Regresi",
		Jam:        "07:00 - 08:40",
		Ruang:      "R101",
	}

	_, err = om.AddExtra("regression@g.us", item, now, "10:00 - 11:40", "R102", "Uji", "tester")
	if err != nil {
		t.Fatalf("AddExtra gagal: %v", err)
	}

	// Pastikan tabel legacy schedule_overrides TIDAK ADA sama sekali
	var tblName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schedule_overrides';").Scan(&tblName)
	if !strings.Contains(err.Error(), "no rows") && err != sql.ErrNoRows {
		t.Errorf("Tabel legacy schedule_overrides tidak boleh dibuat di runtime database baru, ditemukan: %s (err: %v)", tblName, err)
	}
}

func TestTeachingEventsMigration_ContextCancellationAndRollback(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Langsung batalkan context

	// Verifikasi QueryRowContext atau ExecContext dengan context yang dibatalkan
	var id int
	err = db.QueryRowContext(ctx, "SELECT id FROM teaching_events LIMIT 1").Scan(&id)
	if err == nil {
		t.Errorf("Diharapkan error context canceled, tapi berhasil")
	}

	// Uji Rollback Transaksi: jika salah satu insert gagal, tidak ada partial insert
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Gagal begin tx: %v", err)
	}

	// Insert teaching_event tanpa offering
	_, err = tx.Exec(`
		INSERT INTO teaching_events (
			event_kind, starts_at, ends_at, lifecycle_status
		) VALUES ('EXTRA', '2026-10-10T08:00:00Z', '2026-10-10T10:00:00Z', 'DRAFT')
	`)
	if err != nil {
		t.Fatalf("Exec insert event gagal: %v", err)
	}

	// Sengaja rollback transaksi
	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Rollback gagal: %v", err)
	}

	// Pastikan tidak ada data yang tersimpan setelah rollback
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM teaching_events WHERE starts_at = '2026-10-10T08:00:00Z'").Scan(&count)
	if err != nil {
		t.Fatalf("Query count gagal: %v", err)
	}
	if count != 0 {
		t.Errorf("Data masih ada setelah rollback! count = %d", count)
	}
}
