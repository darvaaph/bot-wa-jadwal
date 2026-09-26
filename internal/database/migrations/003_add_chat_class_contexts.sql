CREATE TABLE IF NOT EXISTS chat_class_contexts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_jid TEXT NOT NULL UNIQUE,
    class_id INTEGER NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chat_class_contexts_jid ON chat_class_contexts(chat_jid);
CREATE INDEX IF NOT EXISTS idx_chat_class_contexts_class ON chat_class_contexts(class_id);
