CREATE TRIGGER IF NOT EXISTS trg_audit_logs_prevent_update
BEFORE UPDATE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'audit_logs is append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_audit_logs_prevent_delete
BEFORE DELETE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'audit_logs is append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_task_reviews_prevent_update
BEFORE UPDATE ON task_reviews
BEGIN
    SELECT RAISE(ABORT, 'task_reviews is append-only');
END;

CREATE TRIGGER IF NOT EXISTS trg_task_reviews_prevent_delete
BEFORE DELETE ON task_reviews
BEGIN
    SELECT RAISE(ABORT, 'task_reviews is append-only');
END;
