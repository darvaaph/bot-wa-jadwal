-- 011: Usulan koreksi master oleh KM (BE-007).
-- Idempoten: CREATE TABLE/INDEX IF NOT EXISTS agar aman untuk database lama,
-- database baru, maupun penerapan ulang penuh.
CREATE TABLE IF NOT EXISTS master_proposals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL CHECK (kind IN ('ROOM', 'COURSE')),
    target_id INTEGER,
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    note TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED')),
    proposed_by_user_id INTEGER NOT NULL REFERENCES users(id),
    class_id INTEGER NOT NULL REFERENCES classes(id),
    reviewed_by_user_id INTEGER REFERENCES users(id),
    review_note TEXT,
    decided_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK ((status = 'PENDING') = (decided_at IS NULL))
);

CREATE INDEX IF NOT EXISTS idx_master_proposals_status_kind ON master_proposals(status, kind);
CREATE INDEX IF NOT EXISTS idx_master_proposals_class ON master_proposals(class_id, status);
