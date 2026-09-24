package link

import (
	"bot-jadwal/internal/database"
	"context"
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

	ctx := context.Background()
	_, err = lm.academicRepo.EnsureClass(ctx, scope)
	if err != nil {
		t.Fatalf("Gagal memastikan kelas %s: %v", scope, err)
	}

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

	// 3. Uji Backfill dari tabel legacy class_links dengan manifest & import_errors
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
		INSERT INTO class_links (scope_jid, is_group, title, url, category, description, created_by)
		VALUES ('120363unmapped@g.us', 1, 'Unmapped Group Link', 'https://zoom.us/j/unmapped', 'meeting', 'Kelas gaib', '628999@s.whatsapp.net');
	`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel legacy class_links: %v", err)
	}

	lmReloaded, err := NewLinkManager(db)
	if err != nil {
		t.Fatalf("Gagal NewLinkManager untuk backfill: %v", err)
	}

	report, err := lmReloaded.BackfillLegacyLinksContext(ctx)
	if err != nil {
		t.Fatalf("BackfillLegacyLinksContext error: %v", err)
	}
	if report.TotalLegacy != 2 {
		t.Errorf("Expected 2 total legacy items, got %d", report.TotalLegacy)
	}
	if report.Migrated != 1 {
		t.Errorf("Expected 1 migrated item, got %d", report.Migrated)
	}
	if report.Skipped != 1 {
		t.Errorf("Expected 1 skipped item (unmapped), got %d", report.Skipped)
	}

	// Verifikasi pencatatan error ke import_errors
	var errCount int
	err = db.QueryRow("SELECT COUNT(*) FROM import_errors WHERE error_code = 'UNMAPPED_SCOPE'").Scan(&errCount)
	if err != nil || errCount < 1 {
		t.Errorf("Expected import_errors for UNMAPPED_SCOPE, got count=%d, err=%v", errCount, err)
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
