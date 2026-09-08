package link

import (
	"bot-jadwal/internal/database"
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

	// 1. Non-admin di grup mencoba menambah tautan -> harus ditolak
	rejectReply := lm.HandleCommand(groupJID, true, userMember, false, "!link tambah Drive Kelas | https://s.id/drive-d4a")
	if !strings.Contains(rejectReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin addition to be rejected, got: %s", rejectReply)
	}

	// 2. Admin di grup menambah tautan Google Drive
	addDriveReply := lm.HandleCommand(groupJID, true, userAdmin, true, "!link tambah Drive Materi | https://s.id/drive-d4a | Folder slide & rekaman")
	if !strings.Contains(addDriveReply, "BERHASIL DISIMPAN") || !strings.Contains(addDriveReply, "Penyimpanan Materi") {
		t.Errorf("Expected drive link to be saved, got: %s", addDriveReply)
	}

	// 3. Admin di grup menambah tautan Zoom Meeting
	addZoomReply := lm.HandleCommand(groupJID, true, userAdmin, true, "!link tambah Zoom Aljabar Linear | https://meet.google.com/abc-xyz | Dosen: Bu Retno")
	if !strings.Contains(addZoomReply, "BERHASIL DISIMPAN") || !strings.Contains(addZoomReply, "Kuliah Daring") {
		t.Errorf("Expected zoom link to be saved, got: %s", addZoomReply)
	}

	// 4. Admin di grup menambah tautan GitHub Repo
	addRepoReply := lm.HandleCommand(groupJID, true, userAdmin, true, "!link tambah Repo Praktikum SBD | github.com/d4ti-sbd")
	if !strings.Contains(addRepoReply, "BERHASIL DISIMPAN") || !strings.Contains(addRepoReply, "https://github.com/d4ti-sbd") {
		t.Errorf("Expected repo link to be saved with normalized URL, got: %s", addRepoReply)
	}

	// 5. Test !link (Daftar semua tautan)
	listReply := lm.HandleCommand(groupJID, true, userMember, false, "!link")
	if !strings.Contains(listReply, "DAFTAR TAUTAN PENTING KELAS") ||
		!strings.Contains(listReply, "PENYIMPANAN & MATERI") ||
		!strings.Contains(listReply, "KULIAH DARING") ||
		!strings.Contains(listReply, "REPOSITORI & PROYEK") {
		t.Errorf("Expected categorized link list, got: %s", listReply)
	}

	// 6. Test Shortcut !drive
	driveReply := lm.HandleCommand(groupJID, true, userMember, false, "!drive")
	if !strings.Contains(driveReply, "GOOGLE DRIVE KELAS") || !strings.Contains(driveReply, "https://s.id/drive-d4a") {
		t.Errorf("Expected drive shortcut output, got: %s", driveReply)
	}

	// 7. Test Shortcut !zoom / !meet
	zoomReply := lm.HandleCommand(groupJID, true, userMember, false, "!zoom")
	if !strings.Contains(zoomReply, "RUANG KULIAH DARING") || !strings.Contains(zoomReply, "meet.google.com/abc-xyz") {
		t.Errorf("Expected meeting shortcut output, got: %s", zoomReply)
	}

	// 8. Test Filter / Cari tautan per Matkul (!link aljabar)
	searchReply := lm.HandleCommand(groupJID, true, userMember, false, "!link aljabar")
	if !strings.Contains(searchReply, "PENCARIAN TAUTAN") || !strings.Contains(searchReply, "Zoom Aljabar Linear") {
		t.Errorf("Expected search match for aljabar, got: %s", searchReply)
	}
	if strings.Contains(searchReply, "Repo Praktikum SBD") {
		t.Errorf("Search for aljabar should not include SBD repo")
	}

	// 9. Test Scope Isolation (Tautan grup TIDAK boleh bocor ke DM)
	dmListReply := lm.HandleCommand(dmJID, false, dmJID, true, "!link")
	if !strings.Contains(dmListReply, "Belum ada tautan penting yang dicatat") {
		t.Errorf("Expected empty links in fresh DM, got: %s", dmListReply)
	}

	// 10. User di DM pribadi bebas menambah link tanpa perlu hak admin
	dmAddReply := lm.HandleCommand(dmJID, false, dmJID, false, "!link tambah Drive Pribadi | https://s.id/pribadi")
	if !strings.Contains(dmAddReply, "BERHASIL DISIMPAN") {
		t.Errorf("Expected user to add link in DM without admin restriction, got: %s", dmAddReply)
	}

	// 11. Non-admin di grup mencoba menghapus tautan -> ditolak
	delReject := lm.HandleCommand(groupJID, true, userMember, false, "!link hapus 1")
	if !strings.Contains(delReject, "Akses Ditolak") {
		t.Errorf("Expected non-admin deletion to be rejected, got: %s", delReject)
	}

	// 12. Admin di grup menghapus tautan ID 1
	delSuccess := lm.HandleCommand(groupJID, true, userAdmin, true, "!link hapus 1")
	if !strings.Contains(delSuccess, "BERHASIL DIHAPUS") {
		t.Errorf("Expected admin deletion to succeed, got: %s", delSuccess)
	}

	// Verifikasi bahwa ID 1 sudah tidak ada di list
	afterDelList := lm.HandleCommand(groupJID, true, userMember, false, "!link")
	if strings.Contains(afterDelList, "Drive Materi") {
		t.Errorf("Expected deleted link to no longer appear in list, got: %s", afterDelList)
	}

	// 13. Test Bantuan !link bantuan
	helpReply := lm.HandleCommand(groupJID, true, userMember, false, "!link bantuan")
	if !strings.Contains(helpReply, "PANDUAN MODUL TAUTAN PENTING KELAS") {
		t.Errorf("Expected help guide, got: %s", helpReply)
	}
}
