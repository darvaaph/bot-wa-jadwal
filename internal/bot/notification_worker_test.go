package bot

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/database"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
)

// mockWhatsAppSender untuk pengujian worker tanpa koneksi WhatsApp fisik
type mockWhatsAppSender struct {
	connected   bool
	sentTo      types.JID
	sentMessage *waE2E.Message
	shouldFail  bool
}

func (m *mockWhatsAppSender) SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error) {
	if m.shouldFail {
		return whatsmeow.SendResponse{}, fmt.Errorf("mock network error")
	}
	m.sentTo = to
	m.sentMessage = message
	return whatsmeow.SendResponse{
		ID:        types.MessageID("MOCK-WA-MSG-12345"),
		Timestamp: time.Now(),
	}, nil
}

func (m *mockWhatsAppSender) IsConnected() bool {
	return m.connected
}

func TestNotificationWorker_ProcessPending_Delivered(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "worker_test.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	if err := database.MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}

	// 1. Sisipkan kelas dan channel WhatsApp aktif
	_, err = db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (1, 'D4-TI-2024-A', 'd4-ti-2024-a', 'TI', 2024, 'A', 'ACTIVE');
		INSERT INTO whatsapp_channels (id, class_id, jid, channel_type, display_name, status)
		VALUES (1, 1, '120363001234567890@g.us', 'GROUP', 'Grup Kelas TI-24A', 'ACTIVE');
	`)
	if err != nil {
		t.Fatalf("Gagal insert kelas & channel: %v", err)
	}

	// 2. Sisipkan pesan notifikasi pending
	payload := `{"course":"Struktur Data","title":"Tugas Binary Tree","deadline":"2026-10-10 23:59 WIB"}`
	_, err = db.Exec(`
		INSERT INTO notification_messages (
			id, class_id, whatsapp_channel_id, event_type, idempotency_key, payload_json, status, created_at
		) VALUES (1, 1, 1, 'TASK_PUBLISHED', 'test-task-published-1', ?, 'PENDING', CURRENT_TIMESTAMP);
	`, payload)
	if err != nil {
		t.Fatalf("Gagal insert notifikasi: %v", err)
	}

	// 3. Jalankan worker dengan mock sender
	mock := &mockWhatsAppSender{connected: true}
	worker := NewNotificationWorker(db, mock, 50*time.Millisecond)

	count, err := worker.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 processed message, got %d", count)
	}

	// 4. Verifikasi status pesan berubah menjadi SENT
	var status string
	var sentAt string
	_ = db.QueryRow(`SELECT status, COALESCE(sent_at, '') FROM notification_messages WHERE id = 1;`).Scan(&status, &sentAt)
	if status != "SENT" {
		t.Errorf("Expected status SENT, got %s", status)
	}
	if sentAt == "" {
		t.Errorf("Expected non-empty sent_at")
	}

	// 5. Verifikasi rekam notification_attempts
	var attemptResult, providerMsgID string
	_ = db.QueryRow(`
		SELECT result, COALESCE(provider_message_id, '')
		FROM notification_attempts
		WHERE notification_message_id = 1;
	`).Scan(&attemptResult, &providerMsgID)

	if attemptResult != "DELIVERED" {
		t.Errorf("Expected attempt result DELIVERED, got %s", attemptResult)
	}
	if providerMsgID != "MOCK-WA-MSG-12345" {
		t.Errorf("Expected providerMsgID MOCK-WA-MSG-12345, got %s", providerMsgID)
	}

	// 6. Verifikasi isi pesan mock yang terkirim
	if mock.sentTo.String() != "120363001234567890@g.us" {
		t.Errorf("Sent to wrong JID: %s", mock.sentTo.String())
	}
	sentContent := mock.sentMessage.GetConversation()
	if !strings.Contains(sentContent, "Tugas Binary Tree") {
		t.Errorf("Expected sent message to contain task title, got: %s", sentContent)
	}
}

func TestNotificationWorker_ProcessPending_NoChannel(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "worker_test2.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	if err := database.MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}

	_, _ = db.Exec(`
		INSERT INTO classes (id, code, slug, status) VALUES (2, 'TI-B', 'ti-b', 'ACTIVE');
		INSERT INTO notification_messages (
			id, class_id, event_type, idempotency_key, payload_json, status, created_at
		) VALUES (2, 2, 'TASK_PUBLISHED', 'test-notif-no-chan', '{}', 'PENDING', CURRENT_TIMESTAMP);
	`)

	mock := &mockWhatsAppSender{connected: true}
	worker := NewNotificationWorker(db, mock)

	count, err := worker.ProcessPending(context.Background())
	if err != nil {
		t.Fatalf("ProcessPending error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 processed message, got %d", count)
	}

	var status string
	_ = db.QueryRow(`SELECT status FROM notification_messages WHERE id = 2;`).Scan(&status)
	if status != "FAILED" {
		t.Errorf("Expected status FAILED when channel missing, got %s", status)
	}

	var attemptResult string
	_ = db.QueryRow(`SELECT result FROM notification_attempts WHERE notification_message_id = 2;`).Scan(&attemptResult)
	if attemptResult != "NO_CHANNEL" {
		t.Errorf("Expected attempt result NO_CHANNEL, got %s", attemptResult)
	}
}

func TestNotificationWorker_ProcessPending_ClientOffline(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "worker_test3.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB gagal: %v", err)
	}
	defer db.Close()

	if err := database.MigrateV1(db); err != nil {
		t.Fatalf("MigrateV1 gagal: %v", err)
	}

	_, _ = db.Exec(`
		INSERT INTO classes (id, code, slug, status) VALUES (3, 'TI-C', 'ti-c', 'ACTIVE');
		INSERT INTO whatsapp_channels (id, class_id, jid, status) VALUES (3, 3, '120363001234567890@g.us', 'ACTIVE');
		INSERT INTO notification_messages (
			id, class_id, whatsapp_channel_id, event_type, idempotency_key, payload_json, status, created_at
		) VALUES (3, 3, 3, 'TASK_PUBLISHED', 'test-notif-offline', '{}', 'PENDING', CURRENT_TIMESTAMP);
	`)

	// Sender offline
	mock := &mockWhatsAppSender{connected: false}
	worker := NewNotificationWorker(db, mock)

	_, _ = worker.ProcessPending(context.Background())

	// Status harus tetap PENDING (dikembalikan agar dicoba lagi saat online)
	var status string
	_ = db.QueryRow(`SELECT status FROM notification_messages WHERE id = 3;`).Scan(&status)
	if status != "PENDING" {
		t.Errorf("Expected status to remain PENDING when client offline, got %s", status)
	}
}

func TestFormatNotificationText(t *testing.T) {
	tests := []struct {
		eventType   string
		payloadJSON string
		contains    string
	}{
		{"TASK_PUBLISHED", `{"course":"Basis Data","title":"Kuis 1"}`, "Kuis 1"},
		{"TASK_UPDATED", `{"course":"Basis Data","title":"Kuis 1","deadline":"Besok 10:00"}`, "Tenggat Baru"},
		{"SCHEDULE_REPLACEMENT", `{"course":"Jaringan Komputer","starts_at":"Senin 08:00","room":"LAB-1"}`, "PENGUMUMAN KULIAH PENGGANTI"},
		{"SCHEDULE_REVOKED", `{"course":"Jaringan Komputer","reason":"Dosen hadir jadwal normal"}`, "PEMBATALAN KULIAH PENGGANTI"},
	}

	for _, tc := range tests {
		text := FormatNotificationText(tc.eventType, tc.payloadJSON)
		if !strings.Contains(text, tc.contains) {
			t.Errorf("FormatNotificationText(%s) expected to contain %q, got: %s", tc.eventType, tc.contains, text)
		}
	}
}
