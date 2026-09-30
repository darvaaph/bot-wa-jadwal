-- 009: Tautan kelas daring per pola dan kejadian jadwal.
-- Idempoten (pola 007): rebuild tabel agar aman untuk database lama,
-- database baru, maupun penerapan ulang penuh.
PRAGMA foreign_keys=OFF;

CREATE TABLE IF NOT EXISTS schedule_patterns_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
	room_id INTEGER REFERENCES rooms(id),
	day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
	start_time TEXT NOT NULL,
	end_time TEXT NOT NULL CHECK (end_time > start_time),
	effective_from DATE,
	effective_until DATE,
	status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUPERSEDED','ARCHIVED','DELETED')),
	version INTEGER NOT NULL DEFAULT 1,
	meeting_link TEXT,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

INSERT OR IGNORE INTO schedule_patterns_new
	(id, course_offering_id, room_id, day_of_week, start_time, end_time,
	 effective_from, effective_until, status, version, created_at, updated_at)
SELECT id, course_offering_id, room_id, day_of_week, start_time, end_time,
	effective_from, effective_until, status, version, created_at, updated_at
FROM schedule_patterns;

DROP TABLE IF EXISTS schedule_patterns;

ALTER TABLE schedule_patterns_new RENAME TO schedule_patterns;

CREATE INDEX IF NOT EXISTS idx_patterns_offering ON schedule_patterns(course_offering_id, status);
CREATE INDEX IF NOT EXISTS idx_patterns_day ON schedule_patterns(day_of_week, start_time);
CREATE INDEX IF NOT EXISTS idx_patterns_room ON schedule_patterns(room_id, day_of_week);

CREATE TABLE IF NOT EXISTS teaching_events_new (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	origin_schedule_pattern_id INTEGER REFERENCES schedule_patterns(id),
	origin_occurrence_date DATE,
	result_schedule_pattern_id INTEGER REFERENCES schedule_patterns(id),
	event_kind TEXT NOT NULL CHECK (event_kind IN ('REPLACEMENT','EXTRA','HOLIDAY','SESSION_CANCELLED')),
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

INSERT OR IGNORE INTO teaching_events_new
	(id, origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id,
	 event_kind, starts_at, ends_at, room_id, reason, conflict_override_reason,
	 lifecycle_status, published_by_user_id, published_at, revoked_by_user_id,
	 revoked_at, revocation_reason, version, created_at, updated_at)
SELECT id, origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id,
	event_kind, starts_at, ends_at, room_id, reason, conflict_override_reason,
	lifecycle_status, published_by_user_id, published_at, revoked_by_user_id,
	revoked_at, revocation_reason, version, created_at, updated_at
FROM teaching_events;

DROP TABLE IF EXISTS teaching_events;

ALTER TABLE teaching_events_new RENAME TO teaching_events;

CREATE INDEX IF NOT EXISTS idx_events_lifecycle ON teaching_events(lifecycle_status, starts_at);
CREATE INDEX IF NOT EXISTS idx_events_origin ON teaching_events(origin_schedule_pattern_id, origin_occurrence_date);
CREATE UNIQUE INDEX IF NOT EXISTS uq_teaching_events_result_pattern
    ON teaching_events(result_schedule_pattern_id)
    WHERE result_schedule_pattern_id IS NOT NULL;

PRAGMA foreign_keys=ON;
