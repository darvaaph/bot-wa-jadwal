package database

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
)

func TestMigration004AlignsChatClassContexts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "chat_ctx_004.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("gagal membuka database uji: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE classes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE
	)`); err != nil {
		t.Fatalf("gagal membuat tabel classes: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE chat_class_contexts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_jid TEXT NOT NULL UNIQUE,
		class_id INTEGER NOT NULL,
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE CASCADE
	)`); err != nil {
		t.Fatalf("gagal membuat tabel format lama: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO classes (id, code) VALUES (7, 'D4-TI-2024-A')`); err != nil {
		t.Fatalf("gagal menyisipkan kelas: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chat_class_contexts (chat_jid, class_id, updated_at) VALUES
		('user-a@s.whatsapp.net', 7, '2026-09-20T10:00:00.000Z'),
		('user-b@s.whatsapp.net', 7, '2026-09-21T11:00:00.000Z')`); err != nil {
		t.Fatalf("gagal menyisipkan konteks lama: %v", err)
	}

	migrationSQL, err := migrationFiles.ReadFile("migrations/004_align_chat_class_contexts.sql")
	if err != nil {
		t.Fatalf("gagal membaca berkas migrasi 004: %v", err)
	}
	if _, err := db.Exec(string(migrationSQL)); err != nil {
		t.Fatalf("migrasi 004 gagal: %v", err)
	}

	cols, err := db.Query(`PRAGMA table_info(chat_class_contexts)`)
	if err != nil {
		t.Fatalf("gagal membaca PRAGMA: %v", err)
	}
	defer cols.Close()
	type colInfo struct {
		name string
		pk   int
	}
	var found []colInfo
	for cols.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := cols.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("gagal membaca kolom: %v", err)
		}
		found = append(found, colInfo{name: name, pk: pk})
	}
	if err := cols.Err(); err != nil {
		t.Fatalf("iterasi kolom gagal: %v", err)
	}

	expectPK := map[string]bool{"chat_jid": true}
	seen := map[string]bool{}
	var pkName string
	for _, c := range found {
		seen[c.name] = true
		if c.pk == 1 {
			pkName = c.name
		}
	}
	for _, want := range []string{"chat_jid", "class_id", "created_at", "updated_at"} {
		if !seen[want] {
			t.Errorf("kolom %q tidak ditemukan setelah migrasi: %v", want, seen)
		}
	}
	if seen["id"] {
		t.Errorf("kolom legacy id seharusnya hilang setelah migrasi")
	}
	if pkName != "chat_jid" {
		t.Errorf("primary key = %q, diharapkan chat_jid", pkName)
	}
	_ = expectPK

	var ddl string
	if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'chat_class_contexts'`).Scan(&ddl); err != nil {
		t.Fatalf("gagal membaca DDL: %v", err)
	}
	upper := strings.ToUpper(ddl)
	if !strings.Contains(upper, "ON DELETE RESTRICT") {
		t.Errorf("FK seharusnya RESTRICT, didapat: %s", ddl)
	}
	if strings.Contains(upper, "CASCADE") {
		t.Errorf("FK CASCADE seharusnya hilang, didapat: %s", ddl)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM chat_class_contexts`).Scan(&count); err != nil {
		t.Fatalf("gagal menghitung baris: %v", err)
	}
	if count != 2 {
		t.Fatalf("diharapkan 2 baris dipertahankan, didapat %d", count)
	}
	var created, updated string
	if err := db.QueryRow(`SELECT created_at, updated_at FROM chat_class_contexts WHERE chat_jid = 'user-a@s.whatsapp.net'`).Scan(&created, &updated); err != nil {
		t.Fatalf("gagal membaca baris: %v", err)
	}
	if created != "2026-09-20T10:00:00.000Z" || updated != "2026-09-20T10:00:00.000Z" {
		t.Errorf("timestamp tidak dipertahankan: created=%q updated=%q", created, updated)
	}

	var idxCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_chat_class_contexts_class_id'`).Scan(&idxCount); err != nil {
		t.Fatalf("gagal memeriksa index: %v", err)
	}
	if idxCount != 1 {
		t.Errorf("index idx_chat_class_contexts_class_id tidak ditemukan")
	}
	for _, stale := range []string{"idx_chat_class_contexts_jid", "idx_chat_class_contexts_class"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, stale).Scan(&n); err != nil {
			t.Fatalf("gagal memeriksa index %s: %v", stale, err)
		}
		if n != 0 {
			t.Errorf("index lama %s seharusnya hilang", stale)
		}
	}
}
