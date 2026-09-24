package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestInitDB(t *testing.T) {
	testDB := "test_db_init.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi InitDB: %v", err)
	}
	defer db.Close()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Gagal membaca journal_mode: %v", err)
	}
	if journalMode != "wal" && journalMode != "WAL" {
		t.Logf("Catatan: journal_mode adalah %s (di Windows / in-memory bisa bervariasi, pastikan operasional normal)", journalMode)
	}

	_, err = db.Exec(`CREATE TABLE test_concurrency (id INTEGER PRIMARY KEY AUTOINCREMENT, val TEXT);`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel test: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := db.Exec("INSERT INTO test_concurrency (val) VALUES (?);", fmt.Sprintf("val-%d", idx))
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Terjadi error konkurensi pada database pool: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_concurrency;").Scan(&count)
	if err != nil {
		t.Fatalf("Gagal menghitung jumlah baris: %v", err)
	}
	if count != 20 {
		t.Errorf("Diharapkan 20 baris, didapat %d", count)
	}
}

func TestSchemaVerification(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "schema_verify.db")

	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi InitDB: %v", err)
	}
	defer db.Close()

	expectedTables := []string{
		"users",
		"classes",
		"class_settings",
		"semesters",
		"courses",
		"course_offerings",
		"lecturers",
		"offering_lecturers",
		"rooms",
		"role_invitations",
		"role_assignments",
		"login_attempts",
		"user_sessions",
		"portal_sessions",
		"recovery_tokens",
		"schedule_patterns",
		"teaching_events",
		"teaching_event_offerings",
		"room_confirmations",
		"tasks",
		"task_reviews",
		"materials",
		"whatsapp_channels",
		"notification_messages",
		"notification_attempts",
		"audit_logs",
		"import_batches",
		"import_errors",
		"backup_records",
	}

	rows, err := db.Query(`
		SELECT name
		FROM sqlite_master
		WHERE type = 'table'
		  AND name NOT LIKE 'sqlite_%'
		  AND name <> 'schema_migrations'
		ORDER BY name;
	`)
	if err != nil {
		t.Fatalf("Gagal membaca sqlite_master: %v", err)
	}
	defer rows.Close()

	foundTables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Gagal membaca baris tabel: %v", err)
		}
		foundTables[name] = true
	}

	for _, tbl := range expectedTables {
		if !foundTables[tbl] {
			t.Errorf("Tabel yang diharapkan tidak ditemukan: %s", tbl)
		}
	}

	if len(foundTables) != len(expectedTables) {
		t.Errorf("Jumlah tabel domain %d, diharapkan tepat %d: %v", len(foundTables), len(expectedTables), foundTables)
	}
	t.Logf("Berhasil memverifikasi keberadaan %d tabel pada database", len(foundTables))

	fkRows, err := db.Query("PRAGMA foreign_key_check;")
	if err != nil {
		t.Fatalf("Gagal menjalankan PRAGMA foreign_key_check: %v", err)
	}
	defer fkRows.Close()

	var fkViolations []string
	for fkRows.Next() {
		var table, parent string
		var rowid, fkid int
		if err := fkRows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			t.Fatalf("Gagal membaca hasil foreign_key_check: %v", err)
		}
		fkViolations = append(fkViolations, fmt.Sprintf("tabel: %s, parent: %s, rowid: %d, fkid: %d", table, parent, rowid, fkid))
	}
	if len(fkViolations) > 0 {
		t.Errorf("Ditemukan pelanggaran foreign key: %v", fkViolations)
	} else {
		t.Log("PRAGMA foreign_key_check lulus tanpa error")
	}

	var fkEnabled int
	if err := db.QueryRow("PRAGMA foreign_keys;").Scan(&fkEnabled); err != nil {
		t.Fatalf("Gagal memeriksa status PRAGMA foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("PRAGMA foreign_keys harus bernilai 1, didapat: %d", fkEnabled)
	}

	var integrity string
	if err := db.QueryRow("PRAGMA integrity_check;").Scan(&integrity); err != nil {
		t.Fatalf("Gagal menjalankan integrity_check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity_check gagal: %s", integrity)
	}

	version, err := CurrentSchemaVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("Gagal membaca versi schema: %v", err)
	}
	if version != LatestSchemaVersion {
		t.Fatalf("Versi schema %d, diharapkan %d", version, LatestSchemaVersion)
	}

	var resultPatternIndexSQL string
	if err := db.QueryRow(`
		SELECT sql FROM sqlite_master
		WHERE type = 'index' AND name = 'uq_teaching_events_result_pattern';
	`).Scan(&resultPatternIndexSQL); err != nil {
		t.Fatalf("Unique index pola hasil tidak ditemukan: %v", err)
	}
	if !strings.Contains(strings.ToUpper(resultPatternIndexSQL), "UNIQUE INDEX") ||
		!strings.Contains(strings.ToUpper(resultPatternIndexSQL), "WHERE RESULT_SCHEDULE_PATTERN_ID IS NOT NULL") {
		t.Fatalf("Definisi unique index pola hasil tidak sesuai: %s", resultPatternIndexSQL)
	}

	// Menyisipkan semester dengan class_id yang tidak ada di tabel classes harus gagal
	_, err = db.Exec(`
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on)
		VALUES (99999, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31');
	`)
	if err == nil {
		t.Error("Diharapkan error foreign key constraint saat menyisipkan semester dengan class_id tidak valid, tetapi berhasil")
	} else {
		t.Logf("Foreign key constraint berhasil memblokir data tidak valid: %v", err)
	}

	// InitDB harus idempoten untuk file database yang sudah ada.
	db2, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("InitDB gagal saat dipanggil kedua kali (uji idempotensi): %v", err)
	}
	db2.Close()
}

func TestInitDBAppliesPendingMigration(t *testing.T) {
	testDB := filepath.Join(t.TempDir(), "pending_migration.db")
	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal membuat database awal: %v", err)
	}
	if _, err := db.Exec("DROP INDEX uq_teaching_events_result_pattern"); err != nil {
		t.Fatalf("Gagal menyiapkan index lama: %v", err)
	}
	if _, err := db.Exec("DELETE FROM schema_migrations"); err != nil {
		t.Fatalf("Gagal menyiapkan versi migration lama: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Gagal menutup database awal: %v", err)
	}

	db, err = InitDB(testDB)
	if err != nil {
		t.Fatalf("InitDB gagal menerapkan pending migration: %v", err)
	}
	defer db.Close()

	version, err := CurrentSchemaVersion(context.Background(), db)
	if err != nil {
		t.Fatalf("Gagal membaca versi hasil migration: %v", err)
	}
	if version != LatestSchemaVersion {
		t.Fatalf("Versi schema %d, diharapkan %d", version, LatestSchemaVersion)
	}

	var indexCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND name = 'uq_teaching_events_result_pattern';
	`).Scan(&indexCount); err != nil {
		t.Fatalf("Gagal memeriksa index hasil migration: %v", err)
	}
	if indexCount != 1 {
		t.Fatalf("Unique index hasil migration berjumlah %d, diharapkan 1", indexCount)
	}

	var triggerCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'trigger' AND name IN (
			'trg_audit_logs_prevent_update',
			'trg_audit_logs_prevent_delete',
			'trg_task_reviews_prevent_update',
			'trg_task_reviews_prevent_delete'
		);
	`).Scan(&triggerCount); err != nil {
		t.Fatalf("Gagal memeriksa trigger append-only: %v", err)
	}
	if triggerCount != 4 {
		t.Fatalf("Trigger append-only berjumlah %d, diharapkan 4", triggerCount)
	}
}

func TestAppendOnlyTablesRejectMutation(t *testing.T) {
	db, err := InitDB(filepath.Join(t.TempDir(), "append_only.db"))
	if err != nil {
		t.Fatalf("Gagal membuat database: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`
		INSERT INTO audit_logs (
			actor_type, action, entity_type, correlation_id
		) VALUES ('SYSTEM', 'CREATE', 'test', 'append-only-test');
	`); err != nil {
		t.Fatalf("Gagal menambahkan audit log: %v", err)
	}
	if _, err := db.Exec("UPDATE audit_logs SET action = 'UPDATE' WHERE correlation_id = ?", "append-only-test"); err == nil {
		t.Fatal("UPDATE audit_logs seharusnya ditolak")
	}
	if _, err := db.Exec("DELETE FROM audit_logs WHERE correlation_id = ?", "append-only-test"); err == nil {
		t.Fatal("DELETE audit_logs seharusnya ditolak")
	}
}
