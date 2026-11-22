package notify

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

type remediationSender struct {
	send func(context.Context, string) (string, error)
	ids  []string
}

func (s *remediationSender) SendText(ctx context.Context, jid, text string) (string, error) {
	return s.SendTextWithID(ctx, jid, text, "")
}
func (s *remediationSender) SendTextWithID(ctx context.Context, jid, text, id string) (string, error) {
	s.ids = append(s.ids, id)
	return s.send(ctx, text)
}

func TestRemediation_EnqueueTxReadsOwnTransactionAndRollsBack(t *testing.T) {
	s, ctx := newOutboxDB(t)
	s.db.SetMaxOpenConns(1)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO whatsapp_channels(class_id,jid,channel_type,display_name) VALUES(1,'isolated@g.us','GROUP','Test')`); err != nil {
		t.Fatal(err)
	}
	req := EnqueueRequest{ClassID: 1, EventType: "TASK_PUBLISHED", EntityType: "TASK", EntityID: 1, IdempotencyKey: "atomic", Payload: Payload{Text: "atomic"}}
	id, err := s.EnqueueTx(ctx, tx, req)
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.EnqueueTx(ctx, tx, req)
	if err != nil || other != id {
		t.Fatal(id, other, err)
	}
	var channel int
	if err := tx.QueryRow(`SELECT whatsapp_channel_id FROM notification_messages WHERE id=?`, id).Scan(&channel); err != nil || channel == 0 {
		t.Fatal(channel, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM notification_messages`).Scan(&count)
	if count != 0 {
		t.Fatal("outbox survived rollback")
	}
}

func TestRemediation_FinalizeAfterSenderCancelsContext(t *testing.T) {
	s, ctx := newOutboxDB(t)
	s.EnsureChannel(ctx, 1, "test@g.us", "Test")
	id, _, err := s.Enqueue(ctx, 1, 1, "TASK", "TASK", 1, "cancel-after-send", "hello", "", time.Now().Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sender := &remediationSender{send: func(context.Context, string) (string, error) { cancel(); return "provider-success", nil }}
	sent, _, err := s.ProcessDue(ctx, sender, 1, time.Now())
	if err != nil || sent != 1 {
		t.Fatal(sent, err)
	}
	var status, result string
	if err := s.db.QueryRow(`SELECT m.status,a.result FROM notification_messages m JOIN notification_attempts a ON a.notification_message_id=m.id WHERE m.id=?`, id).Scan(&status, &result); err != nil {
		t.Fatal(err)
	}
	if status != "SENT" || result != "SUCCESS" || sender.ids[0] == "" {
		t.Fatal(status, result, sender.ids)
	}
}

func TestRemediation_FinalizationFailureRequiresReconciliation(t *testing.T) {
	s, ctx := newOutboxDB(t)
	s.EnsureChannel(ctx, 1, "test@g.us", "Test")
	id, _, err := s.Enqueue(ctx, 1, 1, "TASK", "TASK", 1, "uncertain-send", "hello", "", time.Now().Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER fail_finalization BEFORE UPDATE ON notification_messages WHEN NEW.status='SENT' BEGIN SELECT RAISE(ABORT,'injected finalization failure'); END;`); err != nil {
		t.Fatal(err)
	}
	sender := &remediationSender{send: func(context.Context, string) (string, error) { return "provider-success", nil }}
	_, _, err = s.ProcessDue(ctx, sender, 1, time.Now())
	if err == nil {
		t.Fatal("finalization error swallowed")
	}
	var unfinished int
	s.db.QueryRow(`SELECT COUNT(*) FROM notification_attempts WHERE result IS NULL AND provider_message_id IS NOT NULL`).Scan(&unfinished)
	if unfinished != 1 {
		t.Fatal("send identity not durable", unfinished)
	}
	s.db.Exec(`DROP TRIGGER fail_finalization`)
	if _, err := s.ReapStaleProcessing(ctx, time.Now().Add(11*time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.ProcessDue(ctx, sender, 1, time.Now().Add(12*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.ids) != 1 {
		t.Fatal("uncertain delivery automatically repeated")
	}
	if err := s.Retry(ctx, id); err != nil {
		t.Fatal(err)
	}
	_, _, err = s.ProcessDue(ctx, sender, 1, time.Now().Add(13*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(sender.ids) != 2 || sender.ids[0] != sender.ids[1] {
		t.Fatal("explicit retry lost stable identity", sender.ids)
	}
}

func TestRemediation_InFlightSupersededIntentKeepsCorrection(t *testing.T) {
	s, ctx := newOutboxDB(t)
	s.EnsureChannel(ctx, 1, "test@g.us", "Test")
	id, _, err := s.Enqueue(ctx, 1, 1, "TASK", "TASK", 1, "old", "old content", "", time.Now().Add(-time.Hour), nil)
	if err != nil {
		t.Fatal(err)
	}
	sender := &remediationSender{send: func(ctx context.Context, text string) (string, error) {
		if text == "old content" {
			if _, err := s.db.Exec(`UPDATE notification_messages SET status='SUPERSEDED' WHERE id=?`, id); err != nil {
				return "", err
			}
			_, _, err := s.Enqueue(ctx, 1, 1, "TASK_UPDATED", "TASK", 1, "new", "new content", "", time.Now().Add(-time.Minute), nil)
			if err != nil {
				return "", err
			}
		} else if text != "new content" {
			return "", errors.New("unexpected payload")
		}
		return "provider-success", nil
	}}
	if _, _, err := s.ProcessDue(ctx, sender, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.ProcessDue(ctx, sender, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
	var status string
	s.db.QueryRow(`SELECT status FROM notification_messages WHERE id=?`, id).Scan(&status)
	if status != "SUPERSEDED" || len(sender.ids) != 2 {
		t.Fatal(status, sender.ids)
	}
}

func TestRemediation_FailedMessagesDoNotStarveNewMessages(t *testing.T) {
	s, ctx := newOutboxDB(t)
	s.EnsureChannel(ctx, 1, "test@g.us", "Test")
	for i := 1; i <= 11; i++ {
		text := "fail"
		if i == 11 {
			text = "deliver"
		}
		if _, _, err := s.Enqueue(ctx, 1, 1, "TEST", "TASK", int64(i), fmt.Sprintf("fair-%d", i), text, "", time.Now().Add(-time.Hour+time.Duration(i)*time.Second), nil); err != nil {
			t.Fatal(err)
		}
	}
	sender := &remediationSender{send: func(_ context.Context, text string) (string, error) {
		if text == "fail" {
			return "", errors.New("permanent provider rejection")
		}
		return "delivered", nil
	}}
	if _, failed, err := s.ProcessDue(ctx, sender, 10, time.Now()); err != nil || failed != 10 {
		t.Fatal(failed, err)
	}
	if sent, _, err := s.ProcessDue(ctx, sender, 10, time.Now()); err != nil || sent != 1 {
		t.Fatal("queue starved", sent, err)
	}
}

func TestRemediation_ReconcileLegacyOrphansPreservesFutureSchedule(t *testing.T) {
	s, ctx := newOutboxDB(t)
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if _, err := s.db.Exec(`INSERT INTO notification_messages(class_id,event_type,entity_type,entity_id,idempotency_key,payload_json,status,scheduled_at) VALUES(1,'TEST','TASK',1,'future-orphan','{}','PENDING',?)`, future); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnsureChannel(ctx, 1, "test@g.us", "Test"); err != nil {
		t.Fatal(err)
	}
	var scheduled string
	var channel int
	if err := s.db.QueryRow(`SELECT whatsapp_channel_id,scheduled_at FROM notification_messages WHERE idempotency_key='future-orphan'`).Scan(&channel, &scheduled); err != nil {
		t.Fatal(err)
	}
	if channel == 0 || scheduled != future {
		t.Fatal(channel, scheduled, future)
	}
}
