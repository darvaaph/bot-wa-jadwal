package maintenance

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Gagal membuka test db: %v", err)
	}

	schema := `
	CREATE TABLE classes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		slug TEXT NOT NULL UNIQUE
	);

	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		identity_key TEXT NOT NULL UNIQUE
	);

	CREATE TABLE security_attempts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		policy_key TEXT NOT NULL,
		subject_hash TEXT NOT NULL,
		source_hash TEXT NOT NULL,
		outcome TEXT NOT NULL,
		attempted_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	);

	CREATE TABLE user_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		absolute_expires_at TEXT NOT NULL,
		revoked_at TEXT
	);

	CREATE TABLE portal_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL,
		token_hash TEXT NOT NULL UNIQUE,
		access_code_version INTEGER NOT NULL DEFAULT 1,
		expires_at TEXT NOT NULL,
		revoked_at TEXT,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Gagal inisialisasi skema test: %v", err)
	}
	return db
}

func TestPurgeStaleRecords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	// 1. Setup security_attempts: 1 lama (> 30 hari), 1 baru
	oldTime := time.Now().UTC().Add(-35 * 24 * time.Hour).Format(time.RFC3339Nano)
	newTime := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano)

	_, err := db.Exec(`INSERT INTO security_attempts (policy_key, subject_hash, source_hash, outcome, attempted_at)
		VALUES ('login', 'sub1', 'src1', 'SUCCESS', ?),
		       ('login', 'sub2', 'src2', 'SUCCESS', ?);`, oldTime, newTime)
	if err != nil {
		t.Fatalf("Gagal insert security_attempts: %v", err)
	}

	// 2. Setup user_sessions: 1 revoked lama (> 30 hari), 1 aktif, 1 expired lama
	oldRevoked := time.Now().UTC().Add(-35 * 24 * time.Hour).Format(time.RFC3339)
	futureExp := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	oldExp := time.Now().UTC().Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	_, err = db.Exec(`INSERT INTO user_sessions (user_id, token_hash, absolute_expires_at, revoked_at)
		VALUES (1, 'token_revoked_old', ?, ?),
		       (1, 'token_active', ?, NULL),
		       (1, 'token_expired_old', ?, NULL);`, futureExp, oldRevoked, futureExp, oldExp)
	if err != nil {
		t.Fatalf("Gagal insert user_sessions: %v", err)
	}

	// 3. Setup portal_sessions: 1 revoked lama (> 7 hari), 1 aktif, 1 expired lama
	oldPortalRevoked := time.Now().UTC().Add(-10 * 24 * time.Hour).Format(time.RFC3339)
	oldPortalExp := time.Now().UTC().Add(-12 * 24 * time.Hour).Format(time.RFC3339)

	_, err = db.Exec(`INSERT INTO portal_sessions (class_id, token_hash, expires_at, revoked_at)
		VALUES (1, 'p_revoked_old', ?, ?),
		       (1, 'p_active', ?, NULL),
		       (1, 'p_expired_old', ?, NULL);`, futureExp, oldPortalRevoked, futureExp, oldPortalExp)
	if err != nil {
		t.Fatalf("Gagal insert portal_sessions: %v", err)
	}

	// Jalankan PurgeStaleRecords
	if err := PurgeStaleRecords(ctx, db); err != nil {
		t.Fatalf("PurgeStaleRecords error: %v", err)
	}

	// Verifikasi security_attempts
	var secCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM security_attempts;`).Scan(&secCount); err != nil || secCount != 1 {
		t.Fatalf("security_attempts diharapkan tersisa 1, didapat: %d (err: %v)", secCount, err)
	}

	// Verifikasi user_sessions
	var userCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_sessions;`).Scan(&userCount); err != nil || userCount != 1 {
		t.Fatalf("user_sessions diharapkan tersisa 1, didapat: %d (err: %v)", userCount, err)
	}

	// Verifikasi portal_sessions
	var portalCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM portal_sessions;`).Scan(&portalCount); err != nil || portalCount != 1 {
		t.Fatalf("portal_sessions diharapkan tersisa 1, didapat: %d (err: %v)", portalCount, err)
	}
}
