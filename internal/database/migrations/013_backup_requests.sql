CREATE TABLE IF NOT EXISTS backup_requests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL REFERENCES classes(id),
    semester_id INTEGER REFERENCES semesters(id),
    requested_by_user_id INTEGER NOT NULL REFERENCES users(id),
    executed_by_user_id INTEGER REFERENCES users(id),
    backup_id INTEGER REFERENCES backup_records(id),
    reason TEXT NOT NULL CHECK (length(trim(reason)) >= 5),
    status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'PROCESSING', 'EXECUTED', 'FAILED', 'REJECTED')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    decided_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_backup_requests_class_status ON backup_requests(class_id, status);
