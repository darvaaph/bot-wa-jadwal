package link

import (
	"bot-jadwal/internal/database"
	"database/sql"
	"testing"
)

func TestMaterialsMigration_TableVerificationAndBackfill(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal init DB in-memory: %v", err)
	}
	defer db.Close()

	lm, err := NewLinkManager(db)
	if err != nil {
		t.Fatalf("Gagal membuat LinkManager: %v", err)
	}

	scope := "D4-TI-1A"
	user := "6285551234@s.whatsapp.net"

	// 1. Tambah link
	id, err := lm.AddLink(scope, true, "Slide Algoritma", "https://drive.google.com/slide1", "Materi Pekan 1", user)
	if err != nil {
		t.Fatalf("Gagal AddLink: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Expected valid ID, got %d", id)
	}

	// 2. Verifikasi penyimpanan langsung pada tabel target materials
	var classID, createdByUserID int64
	var matType, visibility, status string
	err = db.QueryRow(`
		SELECT class_id, created_by_user_id, material_type, visibility, status
		FROM materials
		WHERE id = ?
	`, id).Scan(&classID, &createdByUserID, &matType, &visibility, &status)
	if err != nil {
		t.Fatalf("Gagal membaca dari tabel materials: %v", err)
	}

	if classID <= 0 {
		t.Errorf("classID pada materials harus valid FK, got %d", classID)
	}
	if createdByUserID <= 0 {
		t.Errorf("created_by_user_id harus valid FK users, got %d", createdByUserID)
	}
	if matType != "DOCUMENT" {
		t.Errorf("material_type untuk drive harus DOCUMENT, got %s", matType)
	}
	if visibility != "CLASS_ACCESS" {
		t.Errorf("visibility grup harus CLASS_ACCESS, got %s", visibility)
	}
	if status != "ACTIVE" {
		t.Errorf("status harus ACTIVE, got %s", status)
	}

	// Verifikasi foreign key check
	var fkCheck string
	_ = db.QueryRow("PRAGMA foreign_key_check;").Scan(&fkCheck)
	if fkCheck != "" {
		t.Errorf("Foreign key check gagal setelah insert materials: %s", fkCheck)
	}

	// 3. Uji Backfill dari tabel legacy class_links
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS class_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_jid TEXT NOT NULL,
			is_group BOOLEAN NOT NULL,
			title TEXT NOT NULL,
			url TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT 'umum',
			description TEXT DEFAULT '',
			created_by TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO class_links (scope_jid, is_group, title, url, category, description, created_by)
		VALUES ('D4-TI-1A', 1, 'Legacy Zoom Link', 'https://zoom.us/j/legacy123', 'meeting', 'Kelas pengganti', '628999@s.whatsapp.net');
	`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel legacy class_links: %v", err)
	}

	lmReloaded, err := NewLinkManager(db)
	if err != nil {
		t.Fatalf("Gagal NewLinkManager untuk backfill: %v", err)
	}

	links, err := lmReloaded.GetLinks("D4-TI-1A")
	if err != nil {
		t.Fatalf("Gagal GetLinks setelah backfill: %v", err)
	}

	foundLegacy := false
	for _, l := range links {
		if l.Title == "Legacy Zoom Link" && l.Category == "meeting" {
			foundLegacy = true
			break
		}
	}
	if !foundLegacy {
		t.Errorf("Tautan legacy tidak ditemukan setelah backfill: %v", links)
	}

	// 4. Soft delete test
	deleted, err := lmReloaded.DeleteLink("D4-TI-1A", id)
	if err != nil || !deleted {
		t.Fatalf("Gagal DeleteLink: %v", err)
	}

	var deletedAt sql.NullString
	_ = db.QueryRow("SELECT deleted_at FROM materials WHERE id = ?", id).Scan(&deletedAt)
	if !deletedAt.Valid || deletedAt.String == "" {
		t.Errorf("deleted_at harus terisi setelah soft delete")
	}
}
