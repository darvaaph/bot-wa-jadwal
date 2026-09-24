package reminder

import (
	"bot-jadwal/internal/chat"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/link"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReminder_MultiClassResolution(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB in-memory: %v", err)
	}
	defer db.Close()

	settingsMgr, err := chat.NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi ChatSettingsManager: %v", err)
	}

	classMgr, err := schedule.NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	groupA := "120363001@g.us"
	groupB := "120363002@g.us"
	groupDefault := "120363003@g.us"

	if err := settingsMgr.SetClass(groupA, "3A"); err != nil {
		t.Fatalf("Gagal set class groupA: %v", err)
	}
	if err := settingsMgr.SetClass(groupB, "3B"); err != nil {
		t.Fatalf("Gagal set class groupB: %v", err)
	}

	loc, _ := time.LoadLocation("Asia/Jakarta")
	seninPagi := time.Date(2026, 9, 7, 6, 30, 0, 0, loc)

	cfgA := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupA))
	msgA := BuildMorningReminder(groupA, cfgA, nil, seninPagi)

	cfgB := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupB))
	msgB := BuildMorningReminder(groupB, cfgB, nil, seninPagi)

	cfgDef := classMgr.GetClassOrDefault(settingsMgr.GetClass(groupDefault))
	msgDef := BuildMorningReminder(groupDefault, cfgDef, nil, seninPagi)

	if !strings.Contains(msgA, "Aljabar Linear") {
		t.Errorf("Pengingat pagi Grup A harus berisi Aljabar Linear (Senin 3A): %s", msgA)
	}

	if !strings.Contains(msgB, "Sistem Operasi") {
		t.Errorf("Pengingat pagi Grup B harus berisi Sistem Operasi (Senin 3B): %s", msgB)
	}

	if !strings.Contains(msgDef, "Aljabar Linear") {
		t.Errorf("Pengingat pagi Grup Default harus menggunakan kelas default 3A: %s", msgDef)
	}
}

func TestBuildMorningReminder_WithMeetingLinks(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB in-memory: %v", err)
	}
	defer db.Close()

	classMgr, err := schedule.NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	linkMgr, err := link.NewLinkManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi LinkManager: %v", err)
	}

	groupJID := "120363009@g.us"
	cfg := classMgr.GetDefaultClass()

	loc, _ := time.LoadLocation("Asia/Jakarta")
	seninPagi := time.Date(2026, 9, 7, 6, 30, 0, 0, loc)

	_, err = linkMgr.AddLink(groupJID, true, "Zoom Kuliah Daring SBD", "https://zoom.us/j/999111222", "ID Passcode 1234", "admin@s.whatsapp.net")
	if err != nil {
		t.Fatalf("Gagal tambah zoom link: %v", err)
	}
	_, err = linkMgr.AddLink(groupJID, true, "Drive Modul Kuliah", "https://drive.google.com/drive/folders/xyz", "Modul lengkap", "admin@s.whatsapp.net")
	if err != nil {
		t.Fatalf("Gagal tambah drive link: %v", err)
	}

	msg := BuildMorningReminder(groupJID, cfg, nil, seninPagi, linkMgr)

	if !strings.Contains(msg, "TAUTAN KULIAH DARING HARI INI:") {
		t.Errorf("Harus menampilkan header tautan daring: %s", msg)
	}
	if !strings.Contains(msg, "Zoom Kuliah Daring SBD") || !strings.Contains(msg, "https://zoom.us/j/999111222") {
		t.Errorf("Harus memuat zoom link yang aktif: %s", msg)
	}
	if strings.Contains(msg, "Drive Modul Kuliah") {
		t.Errorf("Tautan drive tidak boleh masuk ke section kuliah daring: %s", msg)
	}
}

func TestBuildMorningReminder_WithTasks(t *testing.T) {
	// Reminder lama masih memakai adapter tugas pra-v3 yang terisolasi.
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy_tasks.db"))
	if err != nil {
		t.Fatalf("Gagal inisialisasi DB: %v", err)
	}
	defer db.Close()

	classMgr, err := schedule.NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		t.Fatalf("Gagal inisialisasi ClassManager: %v", err)
	}

	taskMgr, err := task.NewTaskManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi TaskManager: %v", err)
	}

	groupJID := "120363010@g.us"
	cfg := classMgr.GetDefaultClass()
	now := time.Now()

	_, _, err = taskMgr.AddTask(groupJID, true, "ALJABAR LINEAR", "Tugas Matriks", "hari ini 23:59", "admin@s.whatsapp.net", now)
	if err != nil {
		t.Fatalf("Gagal tambah tugas: %v", err)
	}

	reminderMsg := BuildMorningReminder(groupJID, cfg, taskMgr, now)
	if !strings.Contains(reminderMsg, "PERINGATAN DEADLINE TUGAS") || !strings.Contains(reminderMsg, "ALJABAR LINEAR") {
		t.Errorf("Expected morning reminder to include urgent task alert, got: %s", reminderMsg)
	}
}
