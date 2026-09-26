-- 005: mencatat chat WhatsApp yang pernah terlihat bot agar admin dapat
-- menautkannya ke kelas dari dashboard (pengganti !setkelas grup).
-- Observasi pasif: bukan data akademik, tidak butuh audit tingkat bisnis.
CREATE TABLE IF NOT EXISTS seen_chats (
    chat_jid TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    is_group INTEGER NOT NULL DEFAULT 0 CHECK (is_group IN (0, 1)),
    first_seen_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    last_seen_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_seen_chats_last_seen
    ON seen_chats(is_group, last_seen_at);
