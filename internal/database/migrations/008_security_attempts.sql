-- BE-012: tabel generik percobaan rate limit terpusat dan persisten.
-- Hanya hash HMAC yang disimpan; tidak pernah identity, IP, token, atau code mentah.
CREATE TABLE IF NOT EXISTS security_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    policy_key TEXT NOT NULL,
    subject_hash TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('SUCCESS', 'FAILURE', 'BLOCKED')),
    attempted_at TEXT NOT NULL,
    blocked_until TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_security_attempts_policy_subject
    ON security_attempts(policy_key, subject_hash, source_hash, attempted_at);
CREATE INDEX IF NOT EXISTS idx_security_attempts_retention
    ON security_attempts(attempted_at);
