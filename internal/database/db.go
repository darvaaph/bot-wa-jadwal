package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

//go:embed migrations/*.sql
var migrationFiles embed.FS

const LatestSchemaVersion = 5

// SchemaSQL mengekspos string DDL SQL untuk keperluan inspeksi atau pengujian.
var SchemaSQL = schemaSQL

type migration struct {
	version int
	name    string
	sql     string
}

func loadMigrations() ([]migration, error) {
	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("gagal membaca migration ter-embed: %w", err)
	}

	migrations := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("nama migration %q tidak memiliki prefix versi", entry.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("versi migration %q tidak valid", entry.Name())
		}
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("gagal membaca migration %q: %w", entry.Name(), err)
		}
		migrations = append(migrations, migration{
			version: version,
			name:    strings.TrimSuffix(entry.Name(), ".sql"),
			sql:     string(body),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	for i := 1; i < len(migrations); i++ {
		if migrations[i-1].version == migrations[i].version {
			return nil, fmt.Errorf("versi migration %d dipakai lebih dari sekali", migrations[i].version)
		}
	}
	return migrations, nil
}

func applyMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		);
	`); err != nil {
		return fmt.Errorf("gagal membuat tabel schema_migrations: %w", err)
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, item := range migrations {
		var applied bool
		if err := db.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)",
			item.version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("gagal memeriksa migration %d: %w", item.version, err)
		}
		if applied {
			continue
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("gagal memulai migration %d: %w", item.version, err)
		}
		if _, err := tx.ExecContext(ctx, item.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d (%s) gagal: %w", item.version, item.name, err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			item.version,
			item.name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("gagal mencatat migration %d: %w", item.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("gagal commit migration %d: %w", item.version, err)
		}
	}
	return nil
}

// CurrentSchemaVersion mengembalikan migration tertinggi yang sudah diterapkan.
func CurrentSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM schema_migrations",
	).Scan(&version); err != nil {
		return 0, fmt.Errorf("gagal membaca versi schema: %w", err)
	}
	return version, nil
}

// InitDB memakai satu pool ber-WAL agar pembaca dan penulis tidak saling mengunci file.
func InitDB(dbPath string) (*sql.DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		dir := filepath.Dir(dbPath)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}
	}

	dsn := dbPath
	if dbPath != ":memory:" && !strings.HasPrefix(dbPath, "file:") {
		// busy_timeout menunggu penulis lain hingga 5 detik; WAL memungkinkan baca-tulis bersamaan.
		// foreign_keys menjaga referensi; synchronous NORMAL menyeimbangkan durabilitas dan throughput.
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

	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
	}
	db.SetConnMaxLifetime(time.Hour)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("gagal memverifikasi koneksi database: %w", err)
	}

	var isInitialized bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='users');").Scan(&isInitialized)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("gagal mengecek status inisialisasi skema database: %w", err)
	}

	if !isInitialized {
		if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("gagal mengeksekusi schema.sql: %w", err)
		}
	}
	if err := applyMigrations(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
