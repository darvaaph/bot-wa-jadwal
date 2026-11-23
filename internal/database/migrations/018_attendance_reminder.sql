ALTER TABLE class_settings ADD COLUMN attendance_reminder_minutes INTEGER NOT NULL DEFAULT 25 CHECK (attendance_reminder_minutes BETWEEN 0 AND 60);
