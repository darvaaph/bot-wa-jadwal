package common

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"bot-jadwal/internal/notify"
)

func QueueNotification(ctx context.Context, exec notify.DBTX, classID int64, eventType, entityType string, entityID int64, payload map[string]any, triggeredByUserID ...int64) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	h := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%d:%s", classID, eventType, entityType, entityID, b)))
	var userID *int64
	if len(triggeredByUserID) > 0 && triggeredByUserID[0] > 0 {
		userID = &triggeredByUserID[0]
	}
	_, err = notify.EnqueueIntent(ctx, exec, notify.EnqueueRequest{ClassID: classID, EventType: eventType, EntityType: entityType, EntityID: entityID, IdempotencyKey: hex.EncodeToString(h[:]), Payload: payload, TriggeredByUserID: userID})
	return err
}

func SupersedeNotifications(ctx context.Context, exec notify.DBTX, entityType string, entityID int64) error {
	_, err := exec.ExecContext(ctx, `UPDATE notification_messages SET status='SUPERSEDED',updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE entity_type=? AND entity_id=? AND status IN ('PENDING','FAILED','PROCESSING')`, entityType, entityID)
	return err
}
