-- Align chat_class_contexts with ERD v3.0.3 and schema.sql.
-- Migration 003 created a divergent shape (surrogate id, no created_at,
-- ON DELETE CASCADE). Target shape uses chat_jid as primary key with
-- created_at tracking and RESTRICT deletes for historical data.
-- Rebuild preserves existing rows; on fresh databases the table is empty
-- at migration time so the copy is a no-op.
CREATE TABLE IF NOT EXISTS chat_class_contexts_new (
    chat_jid TEXT PRIMARY KEY,
    class_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

INSERT OR IGNORE INTO chat_class_contexts_new (chat_jid, class_id, created_at, updated_at)
SELECT chat_jid, class_id, updated_at, updated_at FROM chat_class_contexts;

DROP TABLE chat_class_contexts;

ALTER TABLE chat_class_contexts_new RENAME TO chat_class_contexts;

CREATE INDEX IF NOT EXISTS idx_chat_class_contexts_class_id ON chat_class_contexts(class_id);
