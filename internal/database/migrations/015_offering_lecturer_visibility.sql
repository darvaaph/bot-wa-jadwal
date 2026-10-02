ALTER TABLE offering_lecturers ADD COLUMN superseded_at TEXT;
CREATE INDEX IF NOT EXISTS idx_offering_lecturers_active ON offering_lecturers(course_offering_id, superseded_at);
