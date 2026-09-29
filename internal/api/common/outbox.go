package common

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// QueueNotification mendaftarkan pesan notifikasi siaran ke tabel notification_messages.
// BE-005 durable outbox: intent tetap disimpan PENDING dengan channel NULL
// ketika kelas belum punya kanal aktif; worker tidak mengklaimnya sampai
// ReconcilePendingChannels mengisi channel + scheduled_at.
func QueueNotification(db *sql.DB, classID int64, eventType, entityType string, entityID int64, payload map[string]any, triggeredByUserID ...int64) {
	if db == nil {
		return
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("[Notifikasi] payload %s/%s/%d tidak valid: %v\n", eventType, entityType, entityID, err)
		return
	}
	keySource := fmt.Sprintf("%s:%s:%d:%s", eventType, entityType, entityID, payloadBytes)
	keyHash := sha256.Sum256([]byte(keySource))
	idempotencyKey := hex.EncodeToString(keyHash[:])

	var userID any
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = triggeredByUserID[0]
	}

	// Cari kanal WhatsApp default yang aktif untuk kelas ini jika ada (boleh kosong).
	var channelID any
	var scheduledAt any
	var ch int64
	if err := db.QueryRow(`
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&ch); err == nil {
		channelID = ch
		scheduledAt = time.Now().UTC().Format(time.RFC3339Nano)
	}

	if _, err := db.Exec(`
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING;
	`, classID, channelID, eventType, entityType, entityID, idempotencyKey, string(payloadBytes), scheduledAt, userID); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "unique") {
			fmt.Printf("[Notifikasi] gagal menyimpan outbox %s/%s/%d: %v\n", eventType, entityType, entityID, err)
		}
	}
}

// QueueNotificationTx menyimpan intent notifikasi dalam transaksi bisnis yang sama.
func QueueNotificationTx(ctx context.Context, tx *sql.Tx, classID int64, eventType, entityType string, entityID int64, payload map[string]any, idempotencyKey string, triggeredByUserID ...int64) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(idempotencyKey) == "" {
		keySource := fmt.Sprintf("%s:%s:%d:%s", eventType, entityType, entityID, payloadBytes)
		keyHash := sha256.Sum256([]byte(keySource))
		idempotencyKey = hex.EncodeToString(keyHash[:])
	}
	var userID any
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = triggeredByUserID[0]
	}
	var channelID any
	var scheduledAt any
	var ch int64
	if err := tx.QueryRowContext(ctx, `
		SELECT id FROM whatsapp_channels
		WHERE class_id = ? AND status = 'ACTIVE'
		ORDER BY id DESC LIMIT 1;
	`, classID).Scan(&ch); err == nil {
		channelID = ch
		scheduledAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO notification_messages (
			class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
			idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING;
	`, classID, channelID, eventType, entityType, entityID, idempotencyKey, string(payloadBytes), scheduledAt, userID)
	return err
}
