package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// InitDB menginisialisasi koneksi SQLite tunggal dengan konfigurasi performa & kehandalan tinggi
// Menyatukan koneksi ke satu pool (*sql.DB) agar TaskManager dan OverrideManager tidak saling mengunci file database.
func InitDB(dbPath string) (*sql.DB, error) {
	dsn := dbPath
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		// Parameter SQLite via URI DSN:
		// - busy_timeout(5000): Menunggu hingga 5000ms jika sedang terjadi penulisan bersamaan
		// - journal_mode(WAL): Mengaktifkan Write-Ahead Logging untuk konkurensi baca-tulis tinggi
		// - foreign_keys(1): Mengaktifkan integritas referensial
		// - synchronous(NORMAL): Menghasilkan kecepatan tulis optimal tanpa risiko kerusakan data WAL
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)", dbPath)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database SQLite: %w", err)
	}

	// PRAGMA fallback jika DSN URI tidak diaktifkan oleh driver pada lingkungan tertentu
	_, _ = db.Exec(`
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA foreign_keys = ON;
		PRAGMA synchronous = NORMAL;
	`)

	// Konfigurasi pool koneksi Go database/sql
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("gagal memverifikasi koneksi database: %w", err)
	}

	return db, nil
}
