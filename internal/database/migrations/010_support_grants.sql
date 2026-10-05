-- 010: Hibah dukungan break-glass System Admin (BE-004).
-- Idempoten: CREATE TABLE/INDEX IF NOT EXISTS agar aman untuk database lama,
-- database baru, maupun penerapan ulang penuh.
CREATE TABLE IF NOT EXISTS support_grants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id),
    class_id INTEGER NOT NULL REFERENCES classes(id),
    reason TEXT NOT NULL CHECK (length(trim(reason)) >= 10),
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'CLOSED', 'EXPIRED')),
    expires_at TEXT NOT NULL,
    closed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK ((status = 'ACTIVE' AND closed_at IS NULL) OR (status <> 'ACTIVE' AND closed_at IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_support_grants_user_status ON support_grants(user_id, status);
