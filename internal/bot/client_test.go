package bot

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNewBotClient_PragmaAndWAL(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_session.db")

	client, err := NewBotClient(dbPath)
	if err != nil {
		t.Fatalf("NewBotClient gagal: %v", err)
	}
	defer client.Close()

	// Periksa apakah database sesi yang dibuat berjalan dalam mode WAL dan memiliki busy_timeout
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Gagal membuka db untuk verifikasi: %v", err)
	}
	defer db.Close()

	var journalMode string
	if err := db.QueryRowContext(context.Background(), "PRAGMA journal_mode;").Scan(&journalMode); err != nil {
		t.Fatalf("Gagal membaca journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Ekspektasi journal_mode=wal, didapat: %s", journalMode)
	}

	// Pastikan file database dibuat
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("File database sesi tidak ditemukan: %v", err)
	}
}

func TestNewBotClient_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_session_concurrent.db")

	client, err := NewBotClient(dbPath)
	if err != nil {
		t.Fatalf("NewBotClient gagal: %v", err)
	}
	defer client.Close()

	// Simulasi konkurensi goroutine whatsmeow saat membaca/menulis ke database sesi
	errChan := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := client.container.GetFirstDevice(context.Background())
			errChan <- err
		}()
	}

	for i := 0; i < 10; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Operasi konkuren gagal: %v", err)
		}
	}
}
