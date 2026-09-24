package chat

import (
	"bot-jadwal/internal/database"
	"testing"
)

func TestChatSettings_TargetTablesSeparationAndBackfill(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Gagal init DB: %v", err)
	}
	defer db.Close()

	mgr, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal init ChatSettingsManager: %v", err)
	}

	groupJID := "12036399999@g.us"
	dmJID := "62899999999@s.whatsapp.net"

	// 1. SetClass group -> harus masuk ke whatsapp_channels
	if err := mgr.SetClass(groupJID, "D4-TI-3A"); err != nil {
		t.Fatalf("Gagal SetClass group: %v", err)
	}

	var groupChannelCount int
	err = db.QueryRow("SELECT COUNT(*) FROM whatsapp_channels WHERE jid = ? AND channel_type = 'GROUP' AND status = 'ACTIVE'", groupJID).Scan(&groupChannelCount)
	if err != nil || groupChannelCount != 1 {
		t.Errorf("Group harus tercatat di whatsapp_channels, got count=%d, err=%v", groupChannelCount, err)
	}

	// 2. SetClass DM -> harus masuk ke chat_class_contexts
	if err := mgr.SetClass(dmJID, "D4-TI-3B"); err != nil {
		t.Fatalf("Gagal SetClass DM: %v", err)
	}

	var dmContextCount int
	err = db.QueryRow("SELECT COUNT(*) FROM chat_class_contexts WHERE chat_jid = ?", dmJID).Scan(&dmContextCount)
	if err != nil || dmContextCount != 1 {
		t.Errorf("DM harus tercatat di chat_class_contexts, got count=%d, err=%v", dmContextCount, err)
	}

	// Pastikan DM TIDAK masuk ke whatsapp_channels
	var dmInChannels int
	err = db.QueryRow("SELECT COUNT(*) FROM whatsapp_channels WHERE jid = ?", dmJID).Scan(&dmInChannels)
	if err != nil || dmInChannels != 0 {
		t.Errorf("DM tidak boleh masuk ke whatsapp_channels, got count=%d", dmInChannels)
	}

	// 3. Uji Backfill dari tabel legacy jika ada
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chat_settings (
			scope_jid TEXT PRIMARY KEY,
			class_id TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		INSERT INTO chat_settings (scope_jid, class_id) VALUES ('120363legacy@g.us', 'D4-TI-2A');
		INSERT INTO chat_settings (scope_jid, class_id) VALUES ('62811legacy@s.whatsapp.net', 'D4-TI-2B');
	`)
	if err != nil {
		t.Fatalf("Gagal membuat tabel legacy chat_settings: %v", err)
	}

	mgrReloaded, err := NewChatSettingsManager(db)
	if err != nil {
		t.Fatalf("Gagal NewChatSettingsManager setelah tabel legacy: %v", err)
	}

	if mgrReloaded.GetClass("120363legacy@g.us") != "D4-TI-2A" {
		t.Errorf("Kanal legacy group tidak terbackfill: %s", mgrReloaded.GetClass("120363legacy@g.us"))
	}
	if mgrReloaded.GetClass("62811legacy@s.whatsapp.net") != "D4-TI-2B" {
		t.Errorf("Konteks legacy DM tidak terbackfill: %s", mgrReloaded.GetClass("62811legacy@s.whatsapp.net"))
	}

	// 4. DeleteClass
	if err := mgr.DeleteClass(groupJID); err != nil {
		t.Fatalf("Gagal DeleteClass group: %v", err)
	}
	if mgr.GetClass(groupJID) != "" {
		t.Errorf("Group masih ada setelah delete")
	}
	if err := mgr.DeleteClass(dmJID); err != nil {
		t.Fatalf("Gagal DeleteClass DM: %v", err)
	}
	if mgr.GetClass(dmJID) != "" {
		t.Errorf("DM masih ada setelah delete")
	}
}
