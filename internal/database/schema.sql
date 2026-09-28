PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

BEGIN IMMEDIATE;

-- Root identity and academic entities.
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    identity_key TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'SUSPENDED', 'REVOKED')),
    session_version INTEGER NOT NULL DEFAULT 1 CHECK (session_version >= 1),
    last_login_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE classes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL UNIQUE,
    study_program TEXT NOT NULL,
    cohort_year INTEGER NOT NULL CHECK (cohort_year BETWEEN 1900 AND 9999),
    group_label TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE class_settings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL UNIQUE,
    timezone TEXT NOT NULL,
    portal_access_mode TEXT NOT NULL DEFAULT 'LINK'
        CHECK (portal_access_mode IN ('LINK', 'CODE')),
    portal_code_hash TEXT,
    portal_code_version INTEGER NOT NULL DEFAULT 1 CHECK (portal_code_version >= 1),
    meeting_link_visibility TEXT NOT NULL DEFAULT 'VALID_CLASS_ACCESS'
        CHECK (meeting_link_visibility IN ('VALID_CLASS_ACCESS', 'WHATSAPP_ONLY')),
    morning_reminder_time TEXT,
    afternoon_reminder_time TEXT,
    replacement_reminder_minutes INTEGER NOT NULL DEFAULT 60
        CHECK (replacement_reminder_minutes >= 0),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        (portal_access_mode = 'LINK' AND portal_code_hash IS NULL) OR
        (portal_access_mode = 'CODE' AND portal_code_hash IS NOT NULL)
    ),
    CHECK (morning_reminder_time IS NULL OR morning_reminder_time GLOB '[0-2][0-9]:[0-5][0-9]'),
    CHECK (afternoon_reminder_time IS NULL OR afternoon_reminder_time GLOB '[0-2][0-9]:[0-5][0-9]')
);

CREATE TABLE semesters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    academic_year TEXT NOT NULL,
    term TEXT NOT NULL,
    starts_on TEXT NOT NULL,
    ends_on TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'ACTIVE', 'ARCHIVED')),
    published_at TEXT,
    activated_at TEXT,
    archived_at TEXT,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (class_id, academic_year, term),
    CHECK (starts_on < ends_on),
    CHECK (status <> 'ACTIVE' OR (published_at IS NOT NULL AND activated_at IS NOT NULL)),
    CHECK (status <> 'ARCHIVED' OR archived_at IS NOT NULL)
);

CREATE UNIQUE INDEX uq_semesters_one_active_per_class
    ON semesters(class_id)
    WHERE status = 'ACTIVE';
CREATE INDEX idx_semesters_class_status ON semesters(class_id, status);

CREATE TABLE courses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE course_offerings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    semester_id INTEGER NOT NULL,
    course_id INTEGER NOT NULL,
    activity_type TEXT NOT NULL,
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (course_id) REFERENCES courses(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (semester_id, course_id, activity_type)
);

CREATE INDEX idx_course_offerings_semester_status
    ON course_offerings(semester_id, status);
CREATE INDEX idx_course_offerings_course_id ON course_offerings(course_id);

CREATE TABLE lecturers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE offering_lecturers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_offering_id INTEGER NOT NULL,
    lecturer_id INTEGER NOT NULL,
    responsibility TEXT NOT NULL
        CHECK (responsibility IN ('PRIMARY', 'ASSISTANT', 'OTHER')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (lecturer_id) REFERENCES lecturers(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (course_offering_id, lecturer_id)
);

CREATE INDEX idx_offering_lecturers_lecturer_id ON offering_lecturers(lecturer_id);

CREATE TABLE rooms (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    building TEXT,
    room_type TEXT,
    capacity INTEGER CHECK (capacity IS NULL OR capacity > 0),
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE')),
    source_updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_rooms_status_building ON rooms(status, building);

-- Identity and access entities that depend on academic scope.
CREATE TABLE role_invitations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    token_hash TEXT NOT NULL UNIQUE,
    invited_identity_key TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('SYSTEM_ADMIN', 'KM', 'PJ')),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('GLOBAL', 'CLASS', 'COURSE_OFFERING')),
    class_id INTEGER,
    semester_id INTEGER,
    course_offering_id INTEGER,
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'ACCEPTED', 'EXPIRED', 'REVOKED')),
    expires_at TEXT NOT NULL,
    invited_by_user_id INTEGER NOT NULL,
    accepted_at TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        (role = 'SYSTEM_ADMIN' AND scope_type = 'GLOBAL' AND class_id IS NULL AND semester_id IS NULL AND course_offering_id IS NULL) OR
        (role = 'KM' AND scope_type = 'CLASS' AND class_id IS NOT NULL AND semester_id IS NULL AND course_offering_id IS NULL) OR
        (role = 'PJ' AND scope_type = 'COURSE_OFFERING' AND class_id IS NOT NULL AND semester_id IS NOT NULL AND course_offering_id IS NOT NULL)
    ),
    CHECK ((status = 'ACCEPTED') = (accepted_at IS NOT NULL))
);

CREATE INDEX idx_role_invitations_identity_status
    ON role_invitations(invited_identity_key, status);
CREATE INDEX idx_role_invitations_class_status
    ON role_invitations(class_id, status);
CREATE INDEX idx_role_invitations_expires_at
    ON role_invitations(expires_at)
    WHERE status = 'PENDING';

CREATE TABLE role_assignments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('SYSTEM_ADMIN', 'KM', 'PJ')),
    scope_type TEXT NOT NULL CHECK (scope_type IN ('GLOBAL', 'CLASS', 'COURSE_OFFERING')),
    class_id INTEGER,
    semester_id INTEGER,
    course_offering_id INTEGER,
    accepted_invitation_id INTEGER UNIQUE,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'SUSPENDED', 'REVOKED')),
    valid_from TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    valid_until TEXT,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (accepted_invitation_id) REFERENCES role_invitations(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (valid_until IS NULL OR valid_from < valid_until),
    CHECK (
        (role = 'SYSTEM_ADMIN' AND scope_type = 'GLOBAL' AND class_id IS NULL AND semester_id IS NULL AND course_offering_id IS NULL) OR
        (role = 'KM' AND scope_type = 'CLASS' AND class_id IS NOT NULL AND semester_id IS NULL AND course_offering_id IS NULL) OR
        (role = 'PJ' AND scope_type = 'COURSE_OFFERING' AND class_id IS NOT NULL AND semester_id IS NOT NULL AND course_offering_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_role_assignments_active_scope
    ON role_assignments(
        user_id,
        role,
        scope_type,
        COALESCE(class_id, 0),
        COALESCE(semester_id, 0),
        COALESCE(course_offering_id, 0)
    )
    WHERE status = 'ACTIVE';
CREATE INDEX idx_role_assignments_user_status ON role_assignments(user_id, status);
CREATE INDEX idx_role_assignments_class_role_status ON role_assignments(class_id, role, status);
CREATE INDEX idx_role_assignments_semester_id ON role_assignments(semester_id);
CREATE INDEX idx_role_assignments_course_offering_id ON role_assignments(course_offering_id);

CREATE TABLE login_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    identity_hash TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    outcome TEXT NOT NULL CHECK (outcome IN ('SUCCESS', 'FAILURE', 'BLOCKED')),
    attempted_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX idx_login_attempts_identity_source_time
    ON login_attempts(identity_hash, source_hash, attempted_at);
CREATE INDEX idx_login_attempts_user_time ON login_attempts(user_id, attempted_at);

CREATE TABLE user_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    active_role_assignment_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    session_version INTEGER NOT NULL CHECK (session_version >= 1),
    last_seen_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    absolute_expires_at TEXT NOT NULL,
    revoked_at TEXT,
    revocation_reason TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (active_role_assignment_id) REFERENCES role_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((revoked_at IS NULL AND revocation_reason IS NULL) OR
           (revoked_at IS NOT NULL AND revocation_reason IS NOT NULL)),
    CHECK (created_at < absolute_expires_at)
);

CREATE INDEX idx_user_sessions_user_revoked ON user_sessions(user_id, revoked_at);
CREATE INDEX idx_user_sessions_role_revoked
    ON user_sessions(active_role_assignment_id, revoked_at);
CREATE INDEX idx_user_sessions_active_expiry
    ON user_sessions(absolute_expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE portal_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    access_code_version INTEGER NOT NULL CHECK (access_code_version >= 1),
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (created_at < expires_at)
);

CREATE INDEX idx_portal_sessions_class_version_revoked
    ON portal_sessions(class_id, access_code_version, revoked_at);
CREATE INDEX idx_portal_sessions_active_expiry
    ON portal_sessions(expires_at)
    WHERE revoked_at IS NULL;

CREATE TABLE recovery_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    method TEXT NOT NULL CHECK (trim(method) <> ''),
    expires_at TEXT NOT NULL,
    used_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (created_at < expires_at),
    CHECK (used_at IS NULL OR used_at >= created_at)
);

CREATE INDEX idx_recovery_tokens_user_expiry ON recovery_tokens(user_id, expires_at);
CREATE INDEX idx_recovery_tokens_unused_expiry
    ON recovery_tokens(expires_at)
    WHERE used_at IS NULL;

-- Schedules and concrete teaching events.
CREATE TABLE schedule_patterns (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_offering_id INTEGER NOT NULL,
    room_id INTEGER,
    day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    effective_from TEXT NOT NULL,
    effective_until TEXT,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'SUPERSEDED', 'ARCHIVED', 'DELETED')),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (start_time GLOB '[0-2][0-9]:[0-5][0-9]'),
    CHECK (end_time GLOB '[0-2][0-9]:[0-5][0-9]'),
    CHECK (start_time < end_time),
    CHECK (effective_until IS NULL OR effective_from <= effective_until)
);

CREATE INDEX idx_schedule_patterns_offering_status
    ON schedule_patterns(course_offering_id, status);
CREATE INDEX idx_schedule_patterns_day_start ON schedule_patterns(day_of_week, start_time);
CREATE INDEX idx_schedule_patterns_room_day ON schedule_patterns(room_id, day_of_week);

CREATE TABLE teaching_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    origin_schedule_pattern_id INTEGER,
    origin_occurrence_date TEXT,
    result_schedule_pattern_id INTEGER,
    event_kind TEXT NOT NULL
        CHECK (event_kind IN ('REPLACEMENT', 'EXTRA', 'HOLIDAY', 'SESSION_CANCELLED')),
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    room_id INTEGER,
    reason TEXT,
    conflict_override_reason TEXT,
    lifecycle_status TEXT NOT NULL DEFAULT 'DRAFT'
        CHECK (lifecycle_status IN ('DRAFT', 'PUBLISHED', 'REVOKED')),
    published_by_user_id INTEGER,
    published_at TEXT,
    revoked_by_user_id INTEGER,
    revoked_at TEXT,
    revocation_reason TEXT,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (origin_schedule_pattern_id) REFERENCES schedule_patterns(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (result_schedule_pattern_id) REFERENCES schedule_patterns(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (published_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (revoked_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (starts_at < ends_at),
    CHECK (
        event_kind NOT IN ('REPLACEMENT', 'SESSION_CANCELLED') OR
        (origin_schedule_pattern_id IS NOT NULL AND origin_occurrence_date IS NOT NULL)
    ),
    CHECK ((published_by_user_id IS NULL AND published_at IS NULL) OR
           (published_by_user_id IS NOT NULL AND published_at IS NOT NULL)),
    CHECK (lifecycle_status = 'DRAFT' OR published_at IS NOT NULL),
    CHECK (
        (lifecycle_status <> 'REVOKED' AND revoked_by_user_id IS NULL AND revoked_at IS NULL AND revocation_reason IS NULL) OR
        (lifecycle_status = 'REVOKED' AND revoked_by_user_id IS NOT NULL AND revoked_at IS NOT NULL AND revocation_reason IS NOT NULL)
    )
);

CREATE INDEX idx_teaching_events_lifecycle_start
    ON teaching_events(lifecycle_status, starts_at);
CREATE INDEX idx_teaching_events_origin_occurrence
    ON teaching_events(origin_schedule_pattern_id, origin_occurrence_date);
CREATE UNIQUE INDEX uq_teaching_events_result_pattern
    ON teaching_events(result_schedule_pattern_id)
    WHERE result_schedule_pattern_id IS NOT NULL;
CREATE INDEX idx_teaching_events_room_start ON teaching_events(room_id, starts_at);

CREATE TABLE teaching_event_offerings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    teaching_event_id INTEGER NOT NULL,
    course_offering_id INTEGER NOT NULL,
    participation_role TEXT NOT NULL
        CHECK (participation_role IN ('OWNER', 'PARTICIPANT')),
    participation_status TEXT NOT NULL
        CHECK (participation_status IN ('PENDING', 'ACCEPTED', 'DECLINED', 'REMOVED')),
    responded_by_user_id INTEGER,
    responded_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (teaching_event_id) REFERENCES teaching_events(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (responded_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (teaching_event_id, course_offering_id),
    CHECK (participation_role <> 'OWNER' OR participation_status = 'ACCEPTED'),
    CHECK ((responded_by_user_id IS NULL AND responded_at IS NULL) OR
           (responded_by_user_id IS NOT NULL AND responded_at IS NOT NULL))
);

CREATE UNIQUE INDEX uq_teaching_event_offerings_one_owner
    ON teaching_event_offerings(teaching_event_id)
    WHERE participation_role = 'OWNER';
CREATE INDEX idx_teaching_event_offerings_offering_status
    ON teaching_event_offerings(course_offering_id, participation_status);

CREATE TABLE room_confirmations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    teaching_event_id INTEGER NOT NULL,
    room_id INTEGER NOT NULL,
    confirmation_status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (confirmation_status IN ('PENDING', 'CONFIRMED', 'REJECTED')),
    external_contact TEXT,
    note TEXT,
    recorded_by_user_id INTEGER NOT NULL,
    recorded_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    confirmed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (teaching_event_id) REFERENCES teaching_events(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (recorded_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((confirmation_status = 'CONFIRMED') = (confirmed_at IS NOT NULL))
);

CREATE INDEX idx_room_confirmations_event_status
    ON room_confirmations(teaching_event_id, confirmation_status);
CREATE INDEX idx_room_confirmations_room_status
    ON room_confirmations(room_id, confirmation_status);

-- Tasks and learning materials.
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    course_offering_id INTEGER NOT NULL,
    title TEXT,
    instructions TEXT,
    deadline_at TEXT,
    task_type TEXT,
    submission_text TEXT,
    submission_url TEXT,
    publication_status TEXT NOT NULL DEFAULT 'DRAFT'
        CHECK (publication_status IN ('DRAFT', 'PUBLISHED', 'REVOKED')),
    review_state TEXT NOT NULL DEFAULT 'NOT_REVIEWED'
        CHECK (review_state IN ('NOT_REVIEWED', 'APPROVED', 'CHANGES_REQUESTED', 'REVOKED')),
    reviewed_version INTEGER,
    created_by_user_id INTEGER NOT NULL,
    published_at TEXT,
    completed_at TEXT,
    archived_at TEXT,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    deleted_at TEXT,
    deleted_by_user_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (deleted_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (
        publication_status <> 'PUBLISHED' OR
        (title IS NOT NULL AND trim(title) <> '' AND
         instructions IS NOT NULL AND trim(instructions) <> '' AND
         deadline_at IS NOT NULL AND
         (NULLIF(trim(submission_text), '') IS NOT NULL OR NULLIF(trim(submission_url), '') IS NOT NULL))
    ),
    CHECK (publication_status = 'DRAFT' OR published_at IS NOT NULL),
    CHECK (
        (review_state = 'NOT_REVIEWED' AND reviewed_version IS NULL) OR
        (review_state <> 'NOT_REVIEWED' AND reviewed_version = version)
    ),
    CHECK (review_state <> 'APPROVED' OR publication_status = 'PUBLISHED'),
    CHECK (review_state <> 'CHANGES_REQUESTED' OR publication_status = 'DRAFT'),
    CHECK (review_state <> 'REVOKED' OR publication_status = 'REVOKED'),
    CHECK ((deleted_at IS NULL AND deleted_by_user_id IS NULL) OR
           (deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL))
);

CREATE INDEX idx_tasks_offering_publication_deadline
    ON tasks(course_offering_id, publication_status, deadline_at);
CREATE INDEX idx_tasks_deadline_completed ON tasks(deadline_at, completed_at);
CREATE INDEX idx_tasks_active_publication
    ON tasks(publication_status, deadline_at)
    WHERE deleted_at IS NULL AND archived_at IS NULL;

CREATE TABLE task_reviews (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    reviewer_user_id INTEGER NOT NULL,
    reviewer_role_assignment_id INTEGER NOT NULL,
    task_version INTEGER NOT NULL CHECK (task_version >= 1),
    decision TEXT NOT NULL
        CHECK (decision IN ('APPROVED', 'CHANGES_REQUESTED', 'REVOKED')),
    note TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (reviewer_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (reviewer_role_assignment_id) REFERENCES role_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (task_id, task_version, decision, reviewer_role_assignment_id),
    CHECK (decision = 'APPROVED' OR NULLIF(trim(note), '') IS NOT NULL)
);

CREATE INDEX idx_task_reviews_task_created ON task_reviews(task_id, created_at);
CREATE INDEX idx_task_reviews_reviewer_user ON task_reviews(reviewer_user_id);

CREATE TABLE materials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    course_offering_id INTEGER,
    task_id INTEGER,
    title TEXT NOT NULL,
    material_type TEXT NOT NULL
        CHECK (material_type IN ('DOCUMENT', 'MEETING', 'REPOSITORY', 'PORTAL', 'OTHER')),
    url TEXT NOT NULL,
    description TEXT,
    visibility TEXT NOT NULL DEFAULT 'CLASS_ACCESS'
        CHECK (visibility IN ('CLASS_ACCESS', 'WHATSAPP_ONLY')),
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'INACTIVE', 'ARCHIVED')),
    created_by_user_id INTEGER NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1),
    deleted_at TEXT,
    deleted_by_user_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (course_offering_id) REFERENCES course_offerings(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (deleted_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((deleted_at IS NULL AND deleted_by_user_id IS NULL) OR
           (deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL))
);

CREATE INDEX idx_materials_class_status ON materials(class_id, status);
CREATE INDEX idx_materials_offering_status ON materials(course_offering_id, status);
CREATE INDEX idx_materials_task_id ON materials(task_id);

-- WhatsApp channels and notification outbox.
CREATE TABLE whatsapp_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    jid TEXT NOT NULL UNIQUE,
    channel_type TEXT NOT NULL CHECK (trim(channel_type) <> ''),
    display_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE'
        CHECK (status IN ('ACTIVE', 'DISCONNECTED', 'REVOKED')),
    verified_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX idx_whatsapp_channels_class_status ON whatsapp_channels(class_id, status);

CREATE TABLE chat_class_contexts (
    chat_jid TEXT PRIMARY KEY,
    class_id INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX idx_chat_class_contexts_class_id ON chat_class_contexts(class_id);

CREATE TABLE notification_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    whatsapp_channel_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id INTEGER NOT NULL CHECK (entity_id > 0),
    idempotency_key TEXT NOT NULL UNIQUE,
    payload_json TEXT NOT NULL CHECK (json_valid(payload_json)),
    status TEXT NOT NULL DEFAULT 'PENDING'
        CHECK (status IN ('PENDING', 'PROCESSING', 'SENT', 'FAILED', 'CANCELLED', 'SUPERSEDED')),
    scheduled_at TEXT NOT NULL,
    sent_at TEXT,
    supersedes_message_id INTEGER,
    triggered_by_user_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (whatsapp_channel_id) REFERENCES whatsapp_channels(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (supersedes_message_id) REFERENCES notification_messages(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (triggered_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((status = 'SENT') = (sent_at IS NOT NULL)),
    CHECK (supersedes_message_id IS NULL OR supersedes_message_id <> id)
);

CREATE INDEX idx_notification_messages_status_scheduled
    ON notification_messages(status, scheduled_at);
CREATE INDEX idx_notification_messages_class_created
    ON notification_messages(class_id, created_at);
CREATE INDEX idx_notification_messages_channel_status
    ON notification_messages(whatsapp_channel_id, status);
CREATE INDEX idx_notification_messages_entity ON notification_messages(entity_type, entity_id);

CREATE TABLE notification_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    notification_message_id INTEGER NOT NULL,
    attempt_number INTEGER NOT NULL CHECK (attempt_number >= 1),
    started_at TEXT NOT NULL,
    finished_at TEXT,
    result TEXT CHECK (result IS NULL OR result IN ('SUCCESS', 'FAILED')),
    error_message TEXT,
    provider_message_id TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (notification_message_id) REFERENCES notification_messages(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (notification_message_id, attempt_number),
    CHECK ((finished_at IS NULL AND result IS NULL) OR
           (finished_at IS NOT NULL AND result IS NOT NULL)),
    CHECK (result <> 'SUCCESS' OR error_message IS NULL),
    CHECK (result <> 'FAILED' OR NULLIF(trim(error_message), '') IS NOT NULL)
);

CREATE INDEX idx_notification_attempts_message_started
    ON notification_attempts(notification_message_id, started_at);

-- Audit, import, and backup operations.
CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER,
    semester_id INTEGER,
    actor_user_id INTEGER,
    actor_role_assignment_id INTEGER,
    actor_context_json TEXT CHECK (actor_context_json IS NULL OR json_valid(actor_context_json)),
    actor_type TEXT NOT NULL CHECK (actor_type IN ('USER', 'SYSTEM')),
    action TEXT NOT NULL CHECK (trim(action) <> ''),
    entity_type TEXT NOT NULL,
    entity_id INTEGER CHECK (entity_id IS NULL OR entity_id > 0),
    before_json TEXT CHECK (before_json IS NULL OR json_valid(before_json)),
    after_json TEXT CHECK (after_json IS NULL OR json_valid(after_json)),
    reason TEXT,
    correlation_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (actor_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (actor_role_assignment_id) REFERENCES role_assignments(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK (actor_type <> 'USER' OR actor_user_id IS NOT NULL),
    CHECK (actor_role_assignment_id IS NULL OR actor_user_id IS NOT NULL),
    CHECK (action NOT IN ('RESTORE', 'SESSION_CANCEL', 'OVERRIDE_CONFLICT') OR
           NULLIF(trim(reason), '') IS NOT NULL)
);

CREATE INDEX idx_audit_logs_class_created ON audit_logs(class_id, created_at);
CREATE INDEX idx_audit_logs_semester_created ON audit_logs(semester_id, created_at);
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_actor_created ON audit_logs(actor_user_id, created_at);
CREATE INDEX idx_audit_logs_correlation_id ON audit_logs(correlation_id);

CREATE TABLE import_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    semester_id INTEGER NOT NULL,
    source_type TEXT NOT NULL,
    source_checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'UPLOADED'
        CHECK (status IN ('UPLOADED', 'VALIDATING', 'INVALID', 'READY', 'APPLIED', 'FAILED')),
    created_by_user_id INTEGER NOT NULL,
    summary_json TEXT CHECK (summary_json IS NULL OR json_valid(summary_json)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    UNIQUE (class_id, semester_id, source_checksum)
);

CREATE INDEX idx_import_batches_class_status ON import_batches(class_id, status);
CREATE INDEX idx_import_batches_semester_status ON import_batches(semester_id, status);

CREATE TABLE import_errors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL,
    source_location TEXT NOT NULL,
    field_name TEXT,
    error_code TEXT NOT NULL,
    message TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('ERROR', 'WARNING')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (batch_id) REFERENCES import_batches(id) ON UPDATE RESTRICT ON DELETE RESTRICT
);

CREATE INDEX idx_import_errors_batch_severity ON import_errors(batch_id, severity);

CREATE TABLE backup_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    class_id INTEGER NOT NULL,
    semester_id INTEGER,
    artifact_ref TEXT NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'CREATING'
        CHECK (status IN ('CREATING', 'READY', 'RESTORING', 'VERIFIED', 'FAILED')),
    reason TEXT NOT NULL,
    created_by_user_id INTEGER NOT NULL,
    verified_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (class_id) REFERENCES classes(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (semester_id) REFERENCES semesters(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CHECK ((status = 'VERIFIED') = (verified_at IS NOT NULL))
);

CREATE INDEX idx_backup_records_class_status ON backup_records(class_id, status);
CREATE INDEX idx_backup_records_semester_status ON backup_records(semester_id, status);
CREATE INDEX idx_backup_records_created_by ON backup_records(created_by_user_id);

CREATE TRIGGER trg_audit_logs_prevent_update
BEFORE UPDATE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'audit_logs is append-only');
END;

CREATE TRIGGER trg_audit_logs_prevent_delete
BEFORE DELETE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'audit_logs is append-only');
END;

CREATE TRIGGER trg_task_reviews_prevent_update
BEFORE UPDATE ON task_reviews
BEGIN
    SELECT RAISE(ABORT, 'task_reviews is append-only');
END;

CREATE TRIGGER trg_task_reviews_prevent_delete
BEFORE DELETE ON task_reviews
BEGIN
    SELECT RAISE(ABORT, 'task_reviews is append-only');
END;

COMMIT;
