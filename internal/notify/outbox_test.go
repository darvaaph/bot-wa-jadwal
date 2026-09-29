package notify

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

func newOutboxDB(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "outbox.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-B','d4-ti-2024-b','D4 TI',2024,'B')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (1,'Asia/Jakarta','LINK')`); err != nil {
		t.Fatal(err)
	}
	return NewService(db), ctx
}

func TestOutbox_PublishWithoutChannelDurable(t *testing.T) {
	svc, ctx := newOutboxDB(t)
	id, err := svc.EnqueueDurable(ctx, EnqueueRequest{
		ClassID: 1, EventType: "SCHEDULE_CHANGE", EntityType: "TEACHING_EVENT",
		EntityID: 7, IdempotencyKey: "evt-pub-7", Payload: map[string]any{"text": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var status string
	var ch, sched *string
	var c, s *string
	_ = c
	_ = s
	if err := svc.db.QueryRowContext(ctx, `SELECT status, whatsapp_channel_id, scheduled_at FROM notification_messages WHERE id=?`, id).Scan(&status, &ch, &sched); err != nil {
		t.Fatal(err)
	}
	if status != "PENDING" || ch != nil || sched != nil {
		t.Fatalf("tanpa channel harus PENDING/NULL/NULL, got %s/%v/%v", status, ch, sched)
	}
	// Worker tidak boleh mengklaim.
	sender := &fakeSender{}
	sent, failed, _ := svc.ProcessDue(ctx, sender, 10, time.Now().UTC().Add(time.Hour))
	if sent != 0 || failed != 0 {
		t.Fatalf("pesan tanpa channel tidak boleh diproses: sent=%d failed=%d", sent, failed)
	}
	// Idempotensi: key sama mengembalikan ID sama.
	id2, err := svc.EnqueueDurable(ctx, EnqueueRequest{
		ClassID: 1, EventType: "SCHEDULE_CHANGE", EntityType: "TEACHING_EVENT",
		EntityID: 7, IdempotencyKey: "evt-pub-7", Payload: map[string]any{"text": "x"},
	})
	if err != nil || id2 != id {
		t.Fatalf("idempotensi gagal: %d vs %d err=%v", id, id2, err)
	}
	var cnt int
	_ = svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_messages WHERE idempotency_key='evt-pub-7'`).Scan(&cnt)
	if cnt != 1 {
		t.Fatalf("duplikat outbox: %d", cnt)
	}
}

func TestOutbox_ReconcileOnChannelActivate(t *testing.T) {
	svc, ctx := newOutboxDB(t)
	if _, err := svc.EnqueueDurable(ctx, EnqueueRequest{
		ClassID: 1, EventType: "SCHEDULE_CHANGE", EntityType: "TEACHING_EVENT",
		EntityID: 8, IdempotencyKey: "evt-pub-8", Payload: map[string]any{"text": "y"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnsureChannel(ctx, 1, "120363@test2@g.us", "Grup B"); err != nil {
		t.Fatal(err)
	}
	var chID *int64
	var sched *string
	if err := svc.db.QueryRowContext(ctx, `SELECT whatsapp_channel_id, scheduled_at FROM notification_messages WHERE idempotency_key='evt-pub-8'`).Scan(&chID, &sched); err != nil {
		t.Fatal(err)
	}
	if chID == nil || sched == nil {
		t.Fatalf("reconciler harus mengisi channel dan jadwal: %v %v", chID, sched)
	}
	// Reconcile kedua kali tidak boleh ganda.
	n, _ := svc.ReconcilePendingChannels(ctx, nil)
	if n != 0 {
		t.Fatalf("reconcile idempoten diharapkan 0, got %d", n)
	}
}

func TestOutbox_ConcurrentClaimSingleWinner(t *testing.T) {
	svc, ctx := newOutboxDB(t)
	if _, err := svc.EnsureChannel(ctx, 1, "120363@test3@g.us", "Grup C"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Enqueue(ctx, 1, 1, "DAILY_SUMMARY", "CLASS", 1, "daily:1:2026-11-01", "pagi", "", time.Date(2026, 11, 1, 6, 0, 0, 0, time.UTC), nil); err != nil {
		t.Fatal(err)
	}
	// Dua worker klaim baris yang sama via update kondisional langsung.
	var wg sync.WaitGroup
	wins := make(chan int64, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := svc.db.ExecContext(ctx, `UPDATE notification_messages SET status='PROCESSING',
				updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
				WHERE idempotency_key='daily:1:2026-11-01' AND status='PENDING'`)
			if err == nil {
				if n, _ := res.RowsAffected(); n == 1 {
					wins <- 1
				}
			}
		}()
	}
	wg.Wait()
	close(wins)
	total := 0
	for range wins {
		total++
	}
	if total != 1 {
		t.Fatalf("hanya satu worker boleh menang klaim, got %d", total)
	}
}

func TestOutbox_EnqueueTxAtomic(t *testing.T) {
	svc, ctx := newOutboxDB(t)
	tx, err := svc.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnqueueTx(ctx, tx, EnqueueRequest{
		ClassID: 1, EventType: "SCHEDULE_CHANGE", EntityType: "TEACHING_EVENT",
		EntityID: 9, IdempotencyKey: "evt-tx-9", Payload: map[string]any{"text": "z"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = tx.Rollback()
	var cnt int
	_ = svc.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notification_messages WHERE idempotency_key='evt-tx-9'`).Scan(&cnt)
	if cnt != 0 {
		t.Fatalf("rollback harus membatalkan outbox, got %d", cnt)
	}
}
