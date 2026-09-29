-- BE-005: Durable outbox. Pesan tanpa channel memakai
-- status PENDING, whatsapp_channel_id NULL, scheduled_at NULL.
-- scheduled_at NULL = menunggu konfigurasi, bukan siap diklaim worker.
-- Migrasi aman untuk database lama (NOT NULL) dan database baru (nullable).
PRAGMA foreign_keys=OFF;

CREATE TABLE IF NOT EXISTS notification_messages_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    whatsapp_channel_id INTEGER,
    event_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL CHECK (entity_id > 0),
    idempotency_key TEXT NOT NULL UNIQUE,
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'SENT', 'FAILED', 'CANCELLED', 'SUPERSEDED')),
    scheduled_at TEXT,
    sent_at TEXT,
    supersedes_message_id INTEGER,
    triggered_by_user_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (whatsapp_channel_id) REFERENCES whatsapp_channels(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (supersedes_message_id) REFERENCES notification_messages_new(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((status = 'SENT') = (sent_at IS NOT NULL)),
    CHECK (supersedes_message_id IS NULL OR supersedes_message_id <> id)
);

INSERT OR IGNORE INTO notification_messages_new
    (id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
     idempotency_key, payload_json, status, scheduled_at, sent_at,
     supersedes_message_id, triggered_by_user_id, created_at, updated_at)
SELECT id, class_id, whatsapp_channel_id, event_type, entity_type, entity_id,
     idempotency_key, payload_json, status, scheduled_at, sent_at,
     supersedes_message_id, triggered_by_user_id, created_at, updated_at
FROM notification_messages;

DROP TABLE IF EXISTS notification_messages;

ALTER TABLE notification_messages_new RENAME TO notification_messages;

CREATE INDEX IF NOT EXISTS idx_notification_messages_status_scheduled
    ON notification_messages(status, scheduled_at);
CREATE INDEX IF NOT EXISTS idx_notification_messages_class_created
    ON notification_messages(class_id, created_at);
CREATE INDEX IF NOT EXISTS idx_notification_messages_channel_status
    ON notification_messages(whatsapp_channel_id, status);
CREATE INDEX IF NOT EXISTS idx_notification_messages_entity ON notification_messages(entity_type, entity_id);

PRAGMA foreign_keys=ON;
