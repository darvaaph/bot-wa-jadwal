package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// WhatsAppSender mendefinisikan antarmuka pengiriman pesan WhatsApp (dapat diimplementasikan oleh *whatsmeow.Client atau mock untuk unit test)
type WhatsAppSender interface {
	SendMessage(ctx context.Context, to types.JID, message *waE2E.Message, extra ...whatsmeow.SendRequestExtra) (whatsmeow.SendResponse, error)
	IsConnected() bool
}

// NotificationWorker memantau antrean notification_messages dan menyiarkan pesan ke grup WhatsApp terkait
type NotificationWorker struct {
	db           *sql.DB
	sender       WhatsAppSender
	pollInterval time.Duration
	stopChan     chan struct{}
	wg           sync.WaitGroup
	running      bool
	mu           sync.Mutex
}

// NewNotificationWorker menginisialisasi worker pemroses antrean notifikasi WhatsApp
func NewNotificationWorker(db *sql.DB, sender WhatsAppSender, pollInterval ...time.Duration) *NotificationWorker {
	interval := 3 * time.Second
	if len(pollInterval) > 0 && pollInterval[0] > 0 {
		interval = pollInterval[0]
	}

	return &NotificationWorker{
		db:           db,
		sender:       sender,
		pollInterval: interval,
		stopChan:     make(chan struct{}),
	}
}

// Start menjalankan background goroutine untuk polling antrean notifikasi
func (w *NotificationWorker) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.stopChan = make(chan struct{})
	w.mu.Unlock()

	w.wg.Add(1)
	go w.loop()
	fmt.Printf("📬 [Notification Worker] Background worker siaran WhatsApp aktif (interval: %v)\n", w.pollInterval)
}

// Stop menghentikan background worker secara anggun (graceful stop)
func (w *NotificationWorker) Stop() {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return
	}
	w.running = false
	close(w.stopChan)
	w.mu.Unlock()

	w.wg.Wait()
	fmt.Println("🛑 [Notification Worker] Background worker siaran WhatsApp berhenti")
}

// loop menjalankan polling berkala terhadap antrean
func (w *NotificationWorker) loop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			_, _ = w.ProcessPending(ctx)
			cancel()
		}
	}
}

// PendingNotificationItem menampung data notifikasi yang siap diproses
type PendingNotificationItem struct {
	ID          int64
	ClassID     int64
	EventType   string
	PayloadJSON string
	ChannelJID  string
}

// ProcessPending memproses hingga N pesan notifikasi dengan status PENDING
func (w *NotificationWorker) ProcessPending(ctx context.Context) (int, error) {
	if w.db == nil {
		return 0, fmt.Errorf("database nil")
	}

	query := `
		SELECT nm.id, nm.class_id, nm.event_type, nm.payload_json,
		       COALESCE(wc.jid, '') AS channel_jid
		FROM notification_messages nm
		JOIN whatsapp_channels wc ON nm.whatsapp_channel_id = wc.id AND wc.status = 'ACTIVE'
		WHERE nm.status = 'PENDING'
		  AND nm.whatsapp_channel_id IS NOT NULL
		  AND nm.scheduled_at IS NOT NULL
		  AND nm.scheduled_at <= CURRENT_TIMESTAMP
		ORDER BY nm.created_at ASC
		LIMIT 10;
	`

	rows, err := w.db.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var pending []PendingNotificationItem
	for rows.Next() {
		var item PendingNotificationItem
		if err := rows.Scan(&item.ID, &item.ClassID, &item.EventType, &item.PayloadJSON, &item.ChannelJID); err == nil {
			pending = append(pending, item)
		}
	}

	processedCount := 0
	for _, item := range pending {
		select {
		case <-ctx.Done():
			return processedCount, ctx.Err()
		default:
		}

		// 1. Transisi status atomik: PENDING -> PROCESSING
		res, err := w.db.ExecContext(ctx, `
			UPDATE notification_messages
			SET status = 'PROCESSING'
			WHERE id = ? AND status = 'PENDING';
		`, item.ID)
		if err != nil {
			continue
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			continue // Sudah diambil oleh goroutine lain
		}

		// 2. Tentukan nomor percobaan (attempt_number)
		var attemptNum int
		_ = w.db.QueryRowContext(ctx, `
			SELECT COALESCE(MAX(attempt_number), 0) + 1
			FROM notification_attempts
			WHERE notification_message_id = ?;
		`, item.ID).Scan(&attemptNum)

		// 3. Cari kanal WhatsApp jika belum terasosiasi di baris pesan
		channelJID := strings.TrimSpace(item.ChannelJID)
		if channelJID == "" {
			_ = w.db.QueryRowContext(ctx, `
				SELECT jid FROM whatsapp_channels
				WHERE class_id = ? AND status = 'ACTIVE'
				ORDER BY id DESC LIMIT 1;
			`, item.ClassID).Scan(&channelJID)
		}

		if channelJID == "" {
			// BE-005: tanpa kanal aktif pesan tetap PENDING menunggu rekonsiliasi.
			// Kembalikan ke PENDING tanpa attempt agar tidak dianggap gagal kirim.
			_, _ = w.db.ExecContext(ctx, `
				UPDATE notification_messages SET status = 'PENDING' WHERE id = ?;
			`, item.ID)
			continue
		}

		targetJID, err := types.ParseJID(channelJID)
		if err != nil {
			_, _ = w.db.ExecContext(ctx, `
				UPDATE notification_messages SET status = 'FAILED' WHERE id = ?;
			`, item.ID)
			_, _ = w.db.ExecContext(ctx, `
				INSERT INTO notification_attempts (
					notification_message_id, attempt_number, started_at, finished_at, result, error_message
				) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'FAILED', ?);
			`, item.ID, attemptNum, fmt.Sprintf("Format JID tidak valid: %v", err))
			processedCount++
			continue
		}

		// 4. Format pesan berdasarkan event_type dan payload_json
		msgText := FormatNotificationText(item.EventType, item.PayloadJSON)

		// 5. Cek kesiapan klien WhatsApp
		if w.sender == nil || !w.sender.IsConnected() {
			// Jika bot offline, kembalikan ke PENDING agar dicoba ulang otomatis saat online
			_, _ = w.db.ExecContext(ctx, `
				UPDATE notification_messages SET status = 'PENDING' WHERE id = ?;
			`, item.ID)
			continue
		}

		// 6. Kirim pesan ke WhatsApp
		msgToSend := &waE2E.Message{
			Conversation: proto.String(msgText),
		}

		resp, err := w.sender.SendMessage(ctx, targetJID, msgToSend)
		if err != nil {
			_, _ = w.db.ExecContext(ctx, `
				UPDATE notification_messages SET status = 'FAILED' WHERE id = ?;
			`, item.ID)
			_, _ = w.db.ExecContext(ctx, `
				INSERT INTO notification_attempts (
					notification_message_id, attempt_number, started_at, finished_at, result, error_message
				) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'FAILED', ?);
			`, item.ID, attemptNum, err.Error())
		} else {
			providerID := string(resp.ID)
			_, _ = w.db.ExecContext(ctx, `
				UPDATE notification_messages
				SET status = 'SENT', sent_at = CURRENT_TIMESTAMP
				WHERE id = ?;
			`, item.ID)
			_, _ = w.db.ExecContext(ctx, `
				INSERT INTO notification_attempts (
					notification_message_id, attempt_number, started_at, finished_at, result, provider_message_id
				) VALUES (?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'SUCCESS', ?);
			`, item.ID, attemptNum, providerID)
		}

		processedCount++
	}

	return processedCount, nil
}

// FormatNotificationText menyusun teks siaran WhatsApp yang rapi dan profesional
func FormatNotificationText(eventType, payloadJSON string) string {
	var payload map[string]any
	_ = json.Unmarshal([]byte(payloadJSON), &payload)
	if payload == nil {
		payload = make(map[string]any)
	}

	getString := func(key string) string {
		if val, ok := payload[key]; ok && val != nil {
			return strings.TrimSpace(fmt.Sprintf("%v", val))
		}
		return ""
	}

	switch strings.ToUpper(eventType) {
	case "TASK_PUBLISHED":
		course := getString("course")
		if course == "" {
			course = getString("course_name")
		}
		title := getString("title")
		deadline := getString("deadline")
		instructions := getString("instructions")
		url := getString("submission_url")

		var b strings.Builder
		b.WriteString("📝 *TUGAS BARU DITERBITKAN*\n──────────\n")
		if course != "" {
			b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", course))
		}
		if title != "" {
			b.WriteString(fmt.Sprintf("📌 *Judul:* %s\n", title))
		}
		if deadline != "" {
			b.WriteString(fmt.Sprintf("⏰ *Tenggat Waktu:* %s\n", deadline))
		}
		if instructions != "" {
			b.WriteString(fmt.Sprintf("📋 *Instruksi:* %s\n", instructions))
		}
		if url != "" {
			b.WriteString(fmt.Sprintf("🔗 *Pengumpulan:* %s\n", url))
		}
		b.WriteString("\n_Lihat selengkapnya melalui Portal Mahasiswa._")
		return b.String()

	case "TASK_UPDATED":
		course := getString("course")
		title := getString("title")
		deadline := getString("deadline")

		var b strings.Builder
		b.WriteString("✏️ *PEMBARUAN INFORMASI TUGAS*\n──────────\n")
		if course != "" {
			b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", course))
		}
		if title != "" {
			b.WriteString(fmt.Sprintf("📌 *Judul:* %s\n", title))
		}
		if deadline != "" {
			b.WriteString(fmt.Sprintf("⏰ *Tenggat Baru:* %s\n", deadline))
		}
		b.WriteString("\n_Perubahan telah diperbarui di Portal Mahasiswa._")
		return b.String()

	case "SCHEDULE_REPLACEMENT", "TEACHING_EVENT_PUBLISHED":
		course := getString("course")
		startsAt := getString("starts_at")
		room := getString("room")
		reason := getString("reason")

		var b strings.Builder
		b.WriteString("📢 *PENGUMUMAN KULIAH PENGGANTI*\n──────────\n")
		if course != "" {
			b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", course))
		}
		if startsAt != "" {
			b.WriteString(fmt.Sprintf("📅 *Waktu Pelaksanaan:* %s\n", startsAt))
		}
		if room != "" {
			b.WriteString(fmt.Sprintf("🏢 *Ruangan:* %s\n", room))
		}
		if reason != "" {
			b.WriteString(fmt.Sprintf("💬 *Keterangan:* %s\n", reason))
		}
		b.WriteString("\n_Jadwal efektif perkuliahan telah diperbarui di Portal Mahasiswa._")
		return b.String()

	case "SCHEDULE_REVOKED", "TEACHING_EVENT_REVOKED":
		course := getString("course")
		reason := getString("reason")

		var b strings.Builder
		b.WriteString("⚠️ *PEMBATALAN KULIAH PENGGANTI*\n──────────\n")
		if course != "" {
			b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", course))
		}
		if reason != "" {
			b.WriteString(fmt.Sprintf("💬 *Alasan:* %s\n", reason))
		}
		b.WriteString("\n_Perkuliahan pengganti dibatalkan. Jadwal kembali mengikuti pola perkuliahan reguler._")
		return b.String()

	default:
		msg := getString("message")
		if msg == "" {
			msg = getString("text")
		}
		if msg != "" {
			return fmt.Sprintf("📢 *PENGUMUMAN KELAS*\n──────────\n%s", msg)
		}
		return fmt.Sprintf("📢 *PEMBERITAHUAN KELAS (%s)*\n──────────\nSilakan cek Portal Mahasiswa untuk informasi terbaru.", eventType)
	}
}
