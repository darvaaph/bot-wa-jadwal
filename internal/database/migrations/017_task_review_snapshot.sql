-- version is the concurrency token; reviewed_version identifies reviewed content.
CREATE TABLE tasks_review_snapshot (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
    title TEXT,
    instructions TEXT,
    deadline_at TEXT,
    task_type TEXT,
    submission_text TEXT,
    submission_url TEXT,
    publication_status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (publication_status IN ('DRAFT','PUBLISHED','REVOKED')),
    review_state TEXT NOT NULL DEFAULT 'NOT_REVIEWED' CHECK (review_state IN ('NOT_REVIEWED','APPROVED','CHANGES_REQUESTED','REVOKED')),
    reviewed_version INTEGER,
    created_by_user_id INTEGER NOT NULL REFERENCES users(id),
    published_at TEXT,
    completed_at TEXT,
    archived_at TEXT,
    version INTEGER NOT NULL DEFAULT 1 CHECK(version>=1),
    deleted_at TEXT,
    deleted_by_user_id INTEGER REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CHECK(publication_status<>'PUBLISHED' OR (NULLIF(trim(title),'') IS NOT NULL AND NULLIF(trim(instructions),'') IS NOT NULL AND deadline_at IS NOT NULL AND (NULLIF(trim(submission_text),'') IS NOT NULL OR NULLIF(trim(submission_url),'') IS NOT NULL))),
    CHECK(publication_status='DRAFT' OR published_at IS NOT NULL),
    CHECK((review_state='NOT_REVIEWED' AND reviewed_version IS NULL) OR (review_state<>'NOT_REVIEWED' AND reviewed_version>=1 AND reviewed_version<=version)),
    CHECK(review_state<>'APPROVED' OR publication_status='PUBLISHED'),
    CHECK(review_state<>'CHANGES_REQUESTED' OR publication_status='DRAFT'),
    CHECK(review_state<>'REVOKED' OR publication_status='REVOKED'),
    CHECK((deleted_at IS NULL AND deleted_by_user_id IS NULL) OR (deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL))
);
INSERT INTO tasks_review_snapshot SELECT * FROM tasks;
UPDATE tasks_review_snapshot SET review_state='NOT_REVIEWED',reviewed_version=NULL
WHERE review_state<>'NOT_REVIEWED' AND NOT EXISTS (
    SELECT 1 FROM task_reviews r WHERE r.task_id=tasks_review_snapshot.id
    AND r.task_version=tasks_review_snapshot.reviewed_version AND r.decision=tasks_review_snapshot.review_state
);
DROP TABLE tasks;
ALTER TABLE tasks_review_snapshot RENAME TO tasks;
