package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// EnqueueRequest mewakili satu intent notifikasi bisnis.
// Channel boleh kosong: pesan disimpan PENDING dengan channel dan jadwal NULL
// (menunggu konfigurasi), bukan dibuang.
type EnqueueRequest struct {
	ClassID           int64
	EventType         string
	EntityType        string
	EntityID          int64
	IdempotencyKey    string
	Payload           any
	TriggeredByUserID *int64
}

// DBTX melingkupi *sql.DB dan *sql.Tx agar enqueue dapat berjalan
// di dalam transaksi bisnis yang sama (atomic domain + outbox).
type DBTX interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func marshalPayload(v any) (string, error) {
	if s, ok := v.(string); ok {
		if json.Valid([]byte(s)) {
			return s, nil
		}
		b, err := json.Marshal(Payload{Text: s})
		return string(b), err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func validateEnqueue(req EnqueueRequest) error {
	if req.ClassID <= 0 || strings.TrimSpace(req.EventType) == "" ||
		strings.TrimSpace(req.EntityType) == "" || req.EntityID <= 0 ||
		strings.TrimSpace(req.IdempotencyKey) == "" {
		return ErrInvalidInput
	}
	return nil
}

// EnqueueTx menyimpan intent secara durable di dalam transaksi caller.
// Idempoten: key yang sama mengembalikan message ID yang sama tanpa
// membuat baris atau attempt baru. Channel diisi jika ada yang aktif,
// jika tidak maka NULL + scheduled_at NULL (menunggu rekonsiliasi).
func (s *Service) EnqueueTx(ctx context.Context, exec DBTX, req EnqueueRequest) (int64, error) {
	if err := validateEnqueue(req); err != nil {
		return 0, err
	}
	key := strings.TrimSpace(req.IdempotencyKey)
	payloadStr, err := marshalPayload(req.Payload)
	if err != nil {
		return 0, ErrInvalidInput
	}
	// Idempotency fast-path: kembalikan ID yang sudah ada.
	var existing int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM notification_messages WHERE idempotency_key = ?`, key).Scan(&existing); err == nil {
		return existing, nil
	}
	// Cari channel aktif untuk kelas (boleh tidak ada).
	var chID any
	var sched any
	var ch int64
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM whatsapp_channels WHERE class_id = ? AND status = 'ACTIVE' ORDER BY id LIMIT 1`, req.ClassID).Scan(&ch); err == nil {
		chID = ch
		sched = time.Now().UTC().Format(time.RFC3339Nano)
	}
	var id int64
	err = exec.QueryRowContext(ctx, `INSERT INTO notification_messages
		(class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
		req.ClassID, chID, strings.TrimSpace(req.EventType), strings.TrimSpace(req.EntityType),
		req.EntityID, key, payloadStr, sched, req.TriggeredByUserID,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM notification_messages WHERE idempotency_key = ?`, key).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	return id, err
}

// Enqueue menyimpan intent tanpa transaksi luar (membuka transaksi sendiri).
func (s *Service) EnqueueDurable(ctx context.Context, req EnqueueRequest) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id, err := s.EnqueueTx(ctx, tx, req)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// ReconcilePendingChannels memasangkan pesan PENDING tanpa channel ke
// channel aktif kelasnya secara atomik. Mengembalikan jumlah yang dijadwalkan.
func (s *Service) ReconcilePendingChannels(ctx context.Context, classID *int64) (int64, error) {
	query := `UPDATE notification_messages SET whatsapp_channel_id = (
			SELECT ch.id FROM whatsapp_channels ch
			WHERE ch.class_id = notification_messages.class_id AND ch.status = 'ACTIVE'
			ORDER BY ch.id LIMIT 1
		),
		scheduled_at = strftime('%Y-%m-%dT%H:%M:%fZ','now'),
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE status = 'PENDING' AND whatsapp_channel_id IS NULL AND scheduled_at IS NULL
		AND EXISTS (SELECT 1 FROM whatsapp_channels ch2 WHERE ch2.class_id = notification_messages.class_id AND ch2.status = 'ACTIVE')`
	var args []any
	if classID != nil {
		query += ` AND class_id = ?`
		args = append(args, *classID)
	}
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}
