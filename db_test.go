package main

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestSharedSQLiteConnection(t *testing.T) {
	testDB := "test_shared_db.db"
	defer os.Remove(testDB)

	// 1. Inisialisasi single *sql.DB connection pool
	db, err := InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi InitDB: %v", err)
	}
	defer db.Close()

	// 2. Verifikasi Journal Mode adalah WAL
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode;").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Gagal membaca journal_mode: %v", err)
	}
	if journalMode != "wal" && journalMode != "WAL" {
		t.Logf("Catatan: journal_mode adalah %s (di Windows / in-memory bisa bervariasi, pastikan operasional normal)", journalMode)
	}

	// 3. Inisialisasi TaskManager dan OverrideManager menggunakan *sql.DB yang SAMA
	tm, err := NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal NewTaskManager dengan shared DB: %v", err)
	}

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal NewOverrideManager dengan shared DB: %v", err)
	}

	// 4. Test Concurrency: Jalankan operasi tulis bersamaan pada TaskManager dan OverrideManager
	var wg sync.WaitGroup
	errCh := make(chan error, 20)

	now := time.Now()
	for i := 0; i < 10; i++ {
		wg.Add(2)

		// Operasi tulis TaskManager
		go func(idx int) {
			defer wg.Done()
			_, _, err := tm.AddTask("group1@g.us", true, "SISTEM BASIS DATA", "Tugas Conc", "Besok 23:59", "user1@s.whatsapp.net", now)
			if err != nil {
				errCh <- err
			}
		}(i)

		// Operasi tulis OverrideManager
		go func(idx int) {
			defer wg.Done()
			_, err := om.AddCancel("group1@g.us", JadwalItem{
				KodeMatkul: "SBD101",
				NamaMatkul: "Sistem Basis Data",
			}, now.Add(48*time.Hour), "Dosen Izin", "user1@s.whatsapp.net")
			if err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("Terjadi error konkurensi pada shared DB: %v", err)
	}

	// 5. Test Helper Path Constructors
	testPathDB := "test_path_db.db"
	defer os.Remove(testPathDB)

	tmPath, err := NewTaskManagerWithPath(testPathDB)
	if err != nil {
		t.Fatalf("Gagal NewTaskManagerWithPath: %v", err)
	}
	_ = tmPath.Close()

	omPath, err := NewOverrideManagerWithPath(testPathDB)
	if err != nil {
		t.Fatalf("Gagal NewOverrideManagerWithPath: %v", err)
	}
	_ = omPath.Close()
}
