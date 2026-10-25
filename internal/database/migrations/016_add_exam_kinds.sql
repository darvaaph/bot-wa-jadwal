CREATE TABLE teaching_events_exam (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    origin_schedule_pattern_id INTEGER REFERENCES schedule_patterns(id),
    origin_occurrence_date DATE,
    result_schedule_pattern_id INTEGER REFERENCES schedule_patterns(id),
    event_kind TEXT NOT NULL CHECK (event_kind IN ('REPLACEMENT','EXTRA','HOLIDAY','SESSION_CANCELLED','EXAM_UTS','EXAM_UAS')),
    starts_at DATETIME NOT NULL,
    ends_at DATETIME NOT NULL CHECK (ends_at > starts_at),
    room_id INTEGER REFERENCES rooms(id),
    reason TEXT,
    conflict_override_reason TEXT,
    lifecycle_status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (lifecycle_status IN ('DRAFT','PUBLISHED','REVOKED')),
    published_by_user_id INTEGER REFERENCES users(id),
    published_at DATETIME,
    revoked_by_user_id INTEGER REFERENCES users(id),
    revoked_at DATETIME,
    revocation_reason TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    meeting_link TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT INTO teaching_events_exam
    (id, origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id,
     event_kind, starts_at, ends_at, room_id, reason, conflict_override_reason,
     lifecycle_status, published_by_user_id, published_at, revoked_by_user_id,
     revoked_at, revocation_reason, version, meeting_link, created_at, updated_at)
SELECT id, origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id,
    event_kind, starts_at, ends_at, room_id, reason, conflict_override_reason,
    lifecycle_status, published_by_user_id, published_at, revoked_by_user_id,
    revoked_at, revocation_reason, version, meeting_link, created_at, updated_at
FROM teaching_events;

DROP TABLE teaching_events;
ALTER TABLE teaching_events_exam RENAME TO teaching_events;
