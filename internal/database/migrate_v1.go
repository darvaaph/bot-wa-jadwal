package database

import (
	"database/sql"
	"fmt"
)

// MigrateV1 membangun seluruh tabel target v1 pada database kosong.
// Idempoten: aman dijalankan ulang. Tidak menyentuh database lain.
// Lihat docs/spec/be-v1-schema.md dan docs/product/DATA_MODEL.md.
func MigrateV1(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("koneksi database tidak boleh nil")
	}
	for _, stmt := range v1Schema {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrasi v1 gagal: %w", err)
		}
	}
	return nil
}

var v1Schema = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		identity_key TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL DEFAULT '',
		password_hash TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','REVOKED')),
		session_version INTEGER NOT NULL DEFAULT 1,
		last_login_at DATETIME,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`,
	`CREATE TABLE IF NOT EXISTS classes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		slug TEXT NOT NULL UNIQUE,
		study_program TEXT NOT NULL DEFAULT '',
		cohort_year INTEGER NOT NULL DEFAULT 0,
		group_label TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE','ARCHIVED')),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (study_program, cohort_year, group_label)
	);`,
	`CREATE TABLE IF NOT EXISTS class_settings (
		class_id INTEGER PRIMARY KEY REFERENCES classes(id),
		timezone TEXT NOT NULL DEFAULT 'Asia/Jakarta',
		portal_access_mode TEXT NOT NULL DEFAULT 'LINK' CHECK (portal_access_mode IN ('LINK','CODE')),
		portal_code_hash TEXT,
		portal_code_version INTEGER NOT NULL DEFAULT 1,
		meeting_link_visibility TEXT NOT NULL DEFAULT 'VALID_CLASS_ACCESS' CHECK (meeting_link_visibility IN ('VALID_CLASS_ACCESS','WHATSAPP_ONLY')),
		morning_reminder_time TEXT,
		afternoon_reminder_time TEXT,
		replacement_reminder_minutes INTEGER,
		version INTEGER NOT NULL DEFAULT 1
	);`,
	`CREATE TABLE IF NOT EXISTS semesters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		academic_year TEXT NOT NULL,
		term TEXT NOT NULL,
		starts_on DATE NOT NULL,
		ends_on DATE NOT NULL CHECK (ends_on > starts_on),
		status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','ARCHIVED')),
		published_at DATETIME,
		activated_at DATETIME,
		archived_at DATETIME,
		version INTEGER NOT NULL DEFAULT 1,
		UNIQUE (class_id, academic_year, term)
	);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_semesters_one_active ON semesters(class_id) WHERE status = 'ACTIVE';`,
	`CREATE INDEX IF NOT EXISTS idx_semesters_class_status ON semesters(class_id, status);`,
	`CREATE TABLE IF NOT EXISTS courses (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE'))
	);`,
	`CREATE TABLE IF NOT EXISTS course_offerings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		semester_id INTEGER NOT NULL REFERENCES semesters(id),
		course_id INTEGER NOT NULL REFERENCES courses(id),
		activity_type TEXT NOT NULL,
		display_name TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE','ARCHIVED')),
		version INTEGER NOT NULL DEFAULT 1,
		UNIQUE (semester_id, course_id, activity_type)
	);`,
	`CREATE TABLE IF NOT EXISTS lecturers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		full_name TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE'))
	);`,
	`CREATE TABLE IF NOT EXISTS offering_lecturers (
		course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
		lecturer_id INTEGER NOT NULL REFERENCES lecturers(id),
		responsibility TEXT NOT NULL DEFAULT 'PRIMARY' CHECK (responsibility IN ('PRIMARY','ASSISTANT','OTHER')),
		PRIMARY KEY (course_offering_id, lecturer_id)
	);`,
	`CREATE TABLE IF NOT EXISTS role_invitations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token_hash TEXT NOT NULL UNIQUE,
		invited_identity_key TEXT NOT NULL,
		role TEXT NOT NULL CHECK (role IN ('SYSTEM_ADMIN','KM','PJ')),
		scope_type TEXT NOT NULL CHECK (scope_type IN ('GLOBAL','CLASS','COURSE_OFFERING')),
		class_id INTEGER REFERENCES classes(id),
		semester_id INTEGER REFERENCES semesters(id),
		course_offering_id INTEGER REFERENCES course_offerings(id),
		status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','ACCEPTED','EXPIRED','REVOKED')),
		expires_at DATETIME NOT NULL,
		invited_by_user_id INTEGER REFERENCES users(id)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_invitations_identity ON role_invitations(invited_identity_key, status);`,
	`CREATE INDEX IF NOT EXISTS idx_invitations_class ON role_invitations(class_id, status);`,
	`CREATE TABLE IF NOT EXISTS role_assignments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		role TEXT NOT NULL CHECK (role IN ('SYSTEM_ADMIN','KM','PJ')),
		scope_type TEXT NOT NULL CHECK (scope_type IN ('GLOBAL','CLASS','COURSE_OFFERING')),
		class_id INTEGER REFERENCES classes(id),
		semester_id INTEGER REFERENCES semesters(id),
		course_offering_id INTEGER REFERENCES course_offerings(id),
		accepted_invitation_id INTEGER REFERENCES role_invitations(id),
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUSPENDED','REVOKED')),
		valid_from DATETIME,
		valid_until DATETIME,
		version INTEGER NOT NULL DEFAULT 1,
		CHECK (
			(role = 'SYSTEM_ADMIN' AND scope_type = 'GLOBAL' AND class_id IS NULL AND semester_id IS NULL AND course_offering_id IS NULL)
			OR (role = 'KM' AND scope_type = 'CLASS' AND class_id IS NOT NULL AND course_offering_id IS NULL)
			OR (role = 'PJ' AND scope_type = 'COURSE_OFFERING' AND class_id IS NOT NULL AND semester_id IS NOT NULL AND course_offering_id IS NOT NULL)
		)
	);`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_assignments_one_active ON role_assignments(user_id, role, scope_type, COALESCE(class_id,-1), COALESCE(semester_id,-1), COALESCE(course_offering_id,-1)) WHERE status = 'ACTIVE';`,
	`CREATE INDEX IF NOT EXISTS idx_assignments_user ON role_assignments(user_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_assignments_class ON role_assignments(class_id, role, status);`,
	`CREATE TABLE IF NOT EXISTS login_attempts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER REFERENCES users(id),
		identity_hash TEXT NOT NULL,
		source_hash TEXT NOT NULL,
		outcome TEXT NOT NULL CHECK (outcome IN ('SUCCESS','FAILED','BLOCKED')),
		attempted_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`,
	`CREATE INDEX IF NOT EXISTS idx_login_identity ON login_attempts(identity_hash, source_hash, attempted_at);`,
	`CREATE INDEX IF NOT EXISTS idx_login_user ON login_attempts(user_id, attempted_at);`,
	`CREATE TABLE IF NOT EXISTS user_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		active_role_assignment_id INTEGER REFERENCES role_assignments(id),
		token_hash TEXT NOT NULL UNIQUE,
		session_version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		last_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		absolute_expires_at DATETIME NOT NULL,
		revoked_at DATETIME,
		revocation_reason TEXT
	);`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_user ON user_sessions(user_id, revoked_at);`,
	`CREATE INDEX IF NOT EXISTS idx_sessions_assignment ON user_sessions(active_role_assignment_id, revoked_at);`,
	`CREATE TABLE IF NOT EXISTS portal_sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		token_hash TEXT NOT NULL UNIQUE,
		access_code_version INTEGER NOT NULL DEFAULT 1,
		expires_at DATETIME,
		revoked_at DATETIME
	);`,
	`CREATE INDEX IF NOT EXISTS idx_portal_class ON portal_sessions(class_id, access_code_version, revoked_at);`,
	`CREATE TABLE IF NOT EXISTS recovery_tokens (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL REFERENCES users(id),
		token_hash TEXT NOT NULL UNIQUE,
		method TEXT NOT NULL,
		expires_at DATETIME NOT NULL,
		used_at DATETIME
	);`,
	`CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL DEFAULT '',
		building TEXT,
		room_type TEXT,
		capacity INTEGER,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','INACTIVE')),
		source_updated_at DATETIME
	);`,
	`CREATE TABLE IF NOT EXISTS schedule_patterns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
		room_id INTEGER REFERENCES rooms(id),
		day_of_week INTEGER NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL CHECK (end_time > start_time),
		effective_from DATE,
		effective_until DATE,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','SUPERSEDED','ARCHIVED','DELETED')),
		version INTEGER NOT NULL DEFAULT 1
	);`,
	`CREATE INDEX IF NOT EXISTS idx_patterns_offering ON schedule_patterns(course_offering_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_patterns_day ON schedule_patterns(day_of_week, start_time);`,
	`CREATE INDEX IF NOT EXISTS idx_patterns_room ON schedule_patterns(room_id, day_of_week);`,
	`CREATE TABLE IF NOT EXISTS teaching_events (
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
		version INTEGER NOT NULL DEFAULT 1
	);`,
	`CREATE INDEX IF NOT EXISTS idx_events_lifecycle ON teaching_events(lifecycle_status, starts_at);`,
	`CREATE INDEX IF NOT EXISTS idx_events_origin ON teaching_events(origin_schedule_pattern_id, origin_occurrence_date);`,
	`CREATE TABLE IF NOT EXISTS teaching_event_offerings (
		teaching_event_id INTEGER NOT NULL REFERENCES teaching_events(id),
		course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
		participation_role TEXT NOT NULL CHECK (participation_role IN ('OWNER','PARTICIPANT')),
		participation_status TEXT NOT NULL DEFAULT 'PENDING' CHECK (participation_status IN ('PENDING','ACCEPTED','DECLINED','REMOVED')),
		responded_by_user_id INTEGER REFERENCES users(id),
		responded_at DATETIME,
		PRIMARY KEY (teaching_event_id, course_offering_id)
	);`,
	// Indeks parsial = paling-banyak-satu OWNER. Tepat-satu OWNER
	// ditolak pada transaksi publikasi (lapis endpoint), bukan di DDL.
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_event_one_owner ON teaching_event_offerings(teaching_event_id) WHERE participation_role = 'OWNER';`,
	`CREATE INDEX IF NOT EXISTS idx_event_offering ON teaching_event_offerings(course_offering_id, participation_status);`,
	`CREATE TABLE IF NOT EXISTS room_confirmations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		teaching_event_id INTEGER NOT NULL REFERENCES teaching_events(id),
		room_id INTEGER NOT NULL REFERENCES rooms(id),
		confirmation_status TEXT NOT NULL DEFAULT 'PENDING' CHECK (confirmation_status IN ('PENDING','CONFIRMED','REJECTED')),
		external_contact TEXT,
		note TEXT,
		recorded_by_user_id INTEGER NOT NULL REFERENCES users(id),
		recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		confirmed_at DATETIME
	);`,
	`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		course_offering_id INTEGER NOT NULL REFERENCES course_offerings(id),
		title TEXT NOT NULL,
		instructions TEXT NOT NULL DEFAULT '',
		deadline_at DATETIME NOT NULL,
		task_type TEXT,
		submission_text TEXT,
		submission_url TEXT,
		publication_status TEXT NOT NULL DEFAULT 'DRAFT' CHECK (publication_status IN ('DRAFT','PUBLISHED','REVOKED')),
		review_state TEXT NOT NULL DEFAULT 'NOT_REVIEWED' CHECK (review_state IN ('NOT_REVIEWED','APPROVED','CHANGES_REQUESTED','REVOKED')),
		reviewed_version INTEGER,
		created_by_user_id INTEGER REFERENCES users(id),
		published_at DATETIME,
		completed_at DATETIME,
		archived_at DATETIME,
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		deleted_by_user_id INTEGER REFERENCES users(id),
		CHECK (deleted_at IS NULL OR deleted_by_user_id IS NOT NULL)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_offering ON tasks(course_offering_id, publication_status, deadline_at);`,
	`CREATE INDEX IF NOT EXISTS idx_tasks_deadline ON tasks(deadline_at, completed_at);`,
	`CREATE TABLE IF NOT EXISTS task_reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL REFERENCES tasks(id),
		reviewer_user_id INTEGER NOT NULL REFERENCES users(id),
		reviewer_role_assignment_id INTEGER NOT NULL REFERENCES role_assignments(id),
		task_version INTEGER NOT NULL,
		decision TEXT NOT NULL CHECK (decision IN ('APPROVED','CHANGES_REQUESTED','REVOKED')),
		note TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (task_id, task_version, decision, reviewer_role_assignment_id)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_reviews_task ON task_reviews(task_id, created_at);`,
	`CREATE TABLE IF NOT EXISTS materials (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		course_offering_id INTEGER REFERENCES course_offerings(id),
		task_id INTEGER REFERENCES tasks(id),
		title TEXT NOT NULL,
		material_type TEXT NOT NULL DEFAULT 'OTHER' CHECK (material_type IN ('DOCUMENT','MEETING','REPOSITORY','PORTAL','OTHER')),
		url TEXT,
		description TEXT,
		visibility TEXT NOT NULL DEFAULT 'CLASS_ACCESS' CHECK (visibility IN ('CLASS_ACCESS','WHATSAPP_ONLY')),
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','ARCHIVED','DELETED')),
		created_by_user_id INTEGER REFERENCES users(id),
		version INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		deleted_at DATETIME,
		deleted_by_user_id INTEGER REFERENCES users(id),
		CHECK (deleted_at IS NULL OR deleted_by_user_id IS NOT NULL)
	);`,
	`CREATE INDEX IF NOT EXISTS idx_materials_class ON materials(class_id, status);`,
	`CREATE INDEX IF NOT EXISTS idx_materials_offering ON materials(course_offering_id, status);`,
	`CREATE TABLE IF NOT EXISTS whatsapp_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		jid TEXT NOT NULL UNIQUE,
		channel_type TEXT NOT NULL DEFAULT 'GROUP',
		display_name TEXT,
		status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','DISCONNECTED','REVOKED')),
		verified_at DATETIME
	);`,
	`CREATE TABLE IF NOT EXISTS notification_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		whatsapp_channel_id INTEGER REFERENCES whatsapp_channels(id),
		event_type TEXT NOT NULL,
		entity_type TEXT,
		entity_id INTEGER,
		idempotency_key TEXT NOT NULL UNIQUE,
		payload_json TEXT NOT NULL DEFAULT '{}',
		status TEXT NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING','PROCESSING','SENT','FAILED','CANCELLED','SUPERSEDED')),
		scheduled_at DATETIME,
		sent_at DATETIME,
		supersedes_message_id INTEGER REFERENCES notification_messages(id),
		triggered_by_user_id INTEGER REFERENCES users(id),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`,
	`CREATE INDEX IF NOT EXISTS idx_notif_status ON notification_messages(status, scheduled_at);`,
	`CREATE INDEX IF NOT EXISTS idx_notif_class ON notification_messages(class_id, created_at);`,
	`CREATE TABLE IF NOT EXISTS notification_attempts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		notification_message_id INTEGER NOT NULL REFERENCES notification_messages(id),
		attempt_number INTEGER NOT NULL,
		started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		finished_at DATETIME,
		result TEXT,
		error_message TEXT,
		provider_message_id TEXT,
		UNIQUE (notification_message_id, attempt_number)
	);`,
	`CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER REFERENCES classes(id),
		semester_id INTEGER REFERENCES semesters(id),
		actor_user_id INTEGER REFERENCES users(id),
		actor_role_assignment_id INTEGER REFERENCES role_assignments(id),
		actor_context_json TEXT,
		actor_type TEXT NOT NULL DEFAULT 'USER' CHECK (actor_type IN ('USER','SYSTEM')),
		action TEXT NOT NULL,
		entity_type TEXT,
		entity_id INTEGER,
		before_json TEXT,
		after_json TEXT,
		reason TEXT,
		correlation_id TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`,
	`CREATE INDEX IF NOT EXISTS idx_audit_class ON audit_logs(class_id, created_at);`,
	`CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_logs(entity_type, entity_id);`,
	`CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_user_id, created_at);`,
	`CREATE TABLE IF NOT EXISTS import_batches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		semester_id INTEGER REFERENCES semesters(id),
		source_type TEXT NOT NULL,
		checksum TEXT,
		status TEXT NOT NULL DEFAULT 'UPLOADED' CHECK (status IN ('UPLOADED','VALIDATING','INVALID','READY','APPLIED','FAILED')),
		created_by_user_id INTEGER REFERENCES users(id),
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		summary_json TEXT NOT NULL DEFAULT '{}'
	);`,
	`CREATE TABLE IF NOT EXISTS import_errors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		batch_id INTEGER NOT NULL REFERENCES import_batches(id),
		row_number INTEGER,
		field TEXT,
		error_code TEXT,
		message TEXT,
		severity TEXT NOT NULL CHECK (severity IN ('ERROR','WARNING'))
	);`,
	`CREATE INDEX IF NOT EXISTS idx_import_errors ON import_errors(batch_id, severity);`,
	`CREATE TABLE IF NOT EXISTS backup_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		class_id INTEGER NOT NULL REFERENCES classes(id),
		semester_id INTEGER REFERENCES semesters(id),
		artifact_ref TEXT,
		checksum TEXT,
		status TEXT NOT NULL DEFAULT 'CREATING' CHECK (status IN ('CREATING','READY','RESTORING','VERIFIED','FAILED')),
		created_by_user_id INTEGER REFERENCES users(id),
		reason TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		verified_at DATETIME
	);`,
}
