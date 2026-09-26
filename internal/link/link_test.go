package link

import (
	"bot-jadwal/internal/database"
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func setupTestLinkDB(t *testing.T) (*sql.DB, *LinkManager) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory database: %v", err)
	}

	lm, err := NewLinkManager(db)
	if err != nil {
		t.Fatalf("Failed to create LinkManager: %v", err)
	}

	return db, lm
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"s.id/drive-d4a", "https://s.id/drive-d4a"},
		{"http://zoom.us/j/123", "http://zoom.us/j/123"},
		{"https://meet.google.com/abc", "https://meet.google.com/abc"},
		{"   github.com/d4ti1a   ", "https://github.com/d4ti1a"},
		{"", ""},
	}

	for _, tt := range tests {
		got := NormalizeURL(tt.input)
		if got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDetectLinkCategory(t *testing.T) {
	tests := []struct {
		title string
		url   string
		want  string
	}{
		{"Drive Materi Bersama", "https://s.id/drive-d4a", "drive"},
		{"Slide Kuliah", "https://drive.google.com/drive/folders/abc", "drive"},
		{"Zoom Aljabar Linear", "https://zoom.us/j/999888", "meeting"},
		{"Google Meet Praktikum", "https://meet.google.com/xyz-uvw", "meeting"},
		{"Repo Tugas Praktikum SBD", "https://github.com/kelas-sbd", "repo"},
		{"GitLab Proyek", "https://gitlab.com/d4ti", "repo"},
		{"Presensi SIAKAD", "https://siakad.polban.ac.id", "portal"},
		{"LMS Elearning", "https://elearning.polban.ac.id", "portal"},
		{"Spotify Playlist Belajar", "https://open.spotify.com/playlist/xyz", "umum"},
	}

	for _, tt := range tests {
		got := DetectLinkCategory(tt.title, tt.url)
		if got != tt.want {
			t.Errorf("DetectLinkCategory(%q, %q) = %q, want %q", tt.title, tt.url, got, tt.want)
		}
	}
}

func TestLinkManager_CRUD_And_Permissions(t *testing.T) {
	db, lm := setupTestLinkDB(t)
	defer db.Close()

	groupJID := "120363111111111111@g.us"
	dmJID := "628123456789@s.whatsapp.net"
	userAdmin := "628111111111@s.whatsapp.net"
	userMember := "628222222222@s.whatsapp.net"

	// Petakan scope chat ke kelas resmi sesuai DATA_MODEL
	ctx := context.Background()
	clsA, err := lm.academicRepo.EnsureClass(ctx, "2A")
	if err != nil {
		t.Fatalf("Gagal memastikan kelas 2A: %v", err)
	}
	_, err = db.Exec(`INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status) VALUES (?, ?, 'GROUP', 'Kelas 2A', 'ACTIVE')`, clsA.ID, groupJID)
	if err != nil {
		t.Fatalf("Gagal memetakan whatsapp_channels: %v", err)
	}

	clsB, err := lm.academicRepo.EnsureClass(ctx, "2B")
	if err != nil {
		t.Fatalf("Gagal memastikan kelas 2B: %v", err)
	}
	_, err = db.Exec(`INSERT INTO chat_class_contexts (chat_jid, class_id) VALUES (?, ?)`, dmJID, clsB.ID)
	if err != nil {
		t.Fatalf("Gagal memetakan chat_class_contexts: %v", err)
	}

	rejectReply := lm.HandleCommand(groupJID, true, userMember, false, "!link tambah Drive Kelas | https://s.id/drive-d4a")
	if !strings.Contains(rejectReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected mutation to be redirected to dashboard, got: %s", rejectReply)
	}

	// Seed bacaan via jalur non-perintah (dashboard/API), bukan via command.
	if _, err := lm.AddLink(groupJID, true, "Drive Materi", "https://s.id/drive-d4a", "Folder slide & rekaman", userAdmin); err != nil {
		t.Fatalf("AddLink drive failed: %v", err)
	}
	if _, err := lm.AddLink(groupJID, true, "Zoom Aljabar Linear", "https://meet.google.com/abc-xyz", "Dosen: Bu Retno", userAdmin); err != nil {
		t.Fatalf("AddLink zoom failed: %v", err)
	}
	if _, err := lm.AddLink(groupJID, true, "Repo Praktikum SBD", "github.com/d4ti-sbd", "", userAdmin); err != nil {
		t.Fatalf("AddLink repo failed: %v", err)
	}

	listReply := lm.HandleCommand(groupJID, true, userMember, false, "!link")
	if !strings.Contains(listReply, "DAFTAR TAUTAN PENTING KELAS") ||
		!strings.Contains(listReply, "PENYIMPANAN & MATERI") ||
		!strings.Contains(listReply, "KULIAH DARING") ||
		!strings.Contains(listReply, "REPOSITORI & PROYEK") {
		t.Errorf("Expected categorized link list, got: %s", listReply)
	}

	driveReply := lm.HandleCommand(groupJID, true, userMember, false, "!drive")
	if !strings.Contains(driveReply, "GOOGLE DRIVE KELAS") || !strings.Contains(driveReply, "https://s.id/drive-d4a") {
		t.Errorf("Expected drive shortcut output, got: %s", driveReply)
	}

	zoomReply := lm.HandleCommand(groupJID, true, userMember, false, "!zoom")
	if !strings.Contains(zoomReply, "RUANG KULIAH DARING") || !strings.Contains(zoomReply, "meet.google.com/abc-xyz") {
		t.Errorf("Expected meeting shortcut output, got: %s", zoomReply)
	}

	searchReply := lm.HandleCommand(groupJID, true, userMember, false, "!link aljabar")
	if !strings.Contains(searchReply, "PENCARIAN TAUTAN") || !strings.Contains(searchReply, "Zoom Aljabar Linear") {
		t.Errorf("Expected search match for aljabar, got: %s", searchReply)
	}
	if strings.Contains(searchReply, "Repo Praktikum SBD") {
		t.Errorf("Search for aljabar should not include SBD repo")
	}

	dmListReply := lm.HandleCommand(dmJID, false, dmJID, true, "!link")
	if !strings.Contains(dmListReply, "Belum ada tautan penting yang dicatat") {
		t.Errorf("Expected empty links in fresh DM, got: %s", dmListReply)
	}

	dmAddReply := lm.HandleCommand(dmJID, false, dmJID, false, "!link tambah Drive Pribadi | https://s.id/pribadi")
	if !strings.Contains(dmAddReply, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected DM mutation to be redirected to dashboard, got: %s", dmAddReply)
	}

	delReject := lm.HandleCommand(groupJID, true, userMember, false, "!link hapus 1")
	if !strings.Contains(delReject, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected deletion to be redirected to dashboard, got: %s", delReject)
	}

	delSuccess := lm.HandleCommand(groupJID, true, userAdmin, true, "!link hapus 1")
	if !strings.Contains(delSuccess, "PENGELOLAAN DATA TERPUSAT") {
		t.Errorf("Expected admin deletion to be redirected to dashboard, got: %s", delSuccess)
	}
	if ok, err := lm.DeleteLink(groupJID, 1); err != nil || !ok {
		t.Fatalf("DeleteLink #1 failed: ok=%v err=%v", ok, err)
	}

	afterDelList := lm.HandleCommand(groupJID, true, userMember, false, "!link")
	if strings.Contains(afterDelList, "Drive Materi") {
		t.Errorf("Expected deleted link to no longer appear in list, got: %s", afterDelList)
	}

	helpReply := lm.HandleCommand(groupJID, true, userMember, false, "!link bantuan")
	if !strings.Contains(helpReply, "PANDUAN MODUL TAUTAN PENTING KELAS") {
		t.Errorf("Expected help guide, got: %s", helpReply)
	}
}

func TestLinkManager_RejectUnmappedScope(t *testing.T) {
	db, lm := setupTestLinkDB(t)
	defer db.Close()

	unmappedJID := "120363999999999999@g.us"
	_, err := lm.AddLink(unmappedJID, true, "Drive Materi", "https://s.id/drive-unmapped", "", "admin@s.whatsapp.net")
	if err == nil {
		t.Fatalf("Expected AddLink to reject unmapped scope, but got nil error")
	}
	if !strings.Contains(err.Error(), "scope belum terpetakan") {
		t.Errorf("Expected ErrUnmappedScope, got: %v", err)
	}

	// Pastikan classes tidak bertambah dengan code berupa JID WhatsApp
	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM classes WHERE code = ?", unmappedJID).Scan(&count)
	if count != 0 {
		t.Errorf("DATA_MODEL violation: class must NOT be created with WhatsApp JID as code")
	}
}
