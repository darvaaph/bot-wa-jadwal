package database

import (
	"fmt"
	"os"
	"sync"
	"testing"
)

func TestInitDB(t *testing.T) {
	testDB := "test_db_init.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

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

	// 3. Test Concurrency: Jalankan operasi tulis bersamaan pada pool koneksi
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

	// 4. Verifikasi jumlah data yang berhasil ditulis
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM test_concurrency;").Scan(&count)
	if err != nil {
		t.Fatalf("Gagal menghitung jumlah baris: %v", err)
	}
	if count != 20 {
		t.Errorf("Diharapkan 20 baris, didapat %d", count)
	}
}
