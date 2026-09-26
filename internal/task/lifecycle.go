package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrVersionConflict = errors.New("versi data sudah berubah, muat ulang sebelum menyimpan")
	ErrInvalidState    = errors.New("status tugas tidak memungkinkan operasi ini")
	ErrValidation      = errors.New("validasi tugas gagal")
)

// ActorInfo carries audit identity for lifecycle operations.
type ActorInfo struct {
	UserID           int64
	RoleAssignmentID int64
	ClassID          *int64
	SemesterID       *int64
	CorrelationID    string
}

type UpdateTaskInput struct {
	Title           string
	Instructions    string
	DeadlineAt      string
	TaskType        string
	SubmissionText  *string
	SubmissionURL   *string
	ExpectedVersion int
}

func taskToJSON(t *Task) string {
	b, _ := json.Marshal(t)
	return string(b)
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

// PublishTask moves DRAFT or REVOKED to PUBLISHED.
// isKM indicates the actor is KM (or System Admin support) so an APPROVED review is recorded.
// Idempotent: already PUBLISHED returns current row without changes.
func (r *Repository) PublishTask(ctx context.Context, taskID int64, actor ActorInfo, isKM bool) (*Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ? AND t.deleted_at IS NULL`, taskID)
	current, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if current.PublicationStatus == "PUBLISHED" {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return current, nil
	}
	if current.PublicationStatus != "DRAFT" && current.PublicationStatus != "REVOKED" {
		return nil, ErrInvalidState
	}

	title := strings.TrimSpace(current.Title)
	instructions := strings.TrimSpace(current.Instructions)
	deadline := strings.TrimSpace(current.DeadlineAt)
	hasSubmission := (current.SubmissionText != nil && strings.TrimSpace(*current.SubmissionText) != "") ||
		(current.SubmissionURL != nil && strings.TrimSpace(*current.SubmissionURL) != "")
	if title == "" || instructions == "" || deadline == "" || !hasSubmission {
		return nil, ErrValidation
	}
	if _, err := time.Parse(time.RFC3339Nano, deadline); err != nil {
		if _, err2 := time.Parse(time.RFC3339, deadline); err2 != nil {
			return nil, ErrValidation
		}
	}

	beforeJSON := taskToJSON(current)
	now := nowUTC()

	if isKM {
		var reviewerUserID int64
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM role_assignments WHERE id = ?`, actor.RoleAssignmentID).Scan(&reviewerUserID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("role assignment penerbit tidak ditemukan")
			}
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_reviews (
			task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note
		) VALUES (?, ?, ?, ?, 'APPROVED', NULL)`,
			taskID, reviewerUserID, actor.RoleAssignmentID, current.Version,
		); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET
			publication_status = 'PUBLISHED', review_state = 'APPROVED',
			reviewed_version = ?, published_at = COALESCE(published_at, ?),
			updated_at = ? WHERE id = ?`,
			current.Version, now, now, taskID); err != nil {
			return nil, err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE tasks SET
			publication_status = 'PUBLISHED', review_state = 'NOT_REVIEWED',
			reviewed_version = NULL, published_at = COALESCE(published_at, ?),
			updated_at = ? WHERE id = ?`,
			now, now, taskID); err != nil {
			return nil, err
		}
	}

	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := insertTaskAudit(ctx, tx, actor, "PUBLISH", taskID, &beforeJSON, str(taskToJSON(after)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

// UpdateTask edits visible fields with optimistic locking.
// PJ update sets new version to NOT_REVIEWED; KM update records APPROVED review for the new version.
func (r *Repository) UpdateTask(ctx context.Context, taskID int64, in UpdateTaskInput, actor ActorInfo, isKM bool) (*Task, error) {
	if in.ExpectedVersion < 1 {
		return nil, ErrInvalidState
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ? AND t.deleted_at IS NULL`, taskID)
	current, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if current.PublicationStatus == "REVOKED" {
		return nil, ErrInvalidState
	}
	if current.Version != in.ExpectedVersion {
		return nil, ErrVersionConflict
	}

	beforeJSON := taskToJSON(current)
	now := nowUTC()
	newVersion := current.Version + 1

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = current.Title
	}
	instructions := strings.TrimSpace(in.Instructions)
	if instructions == "" {
		instructions = current.Instructions
	}
	deadline := strings.TrimSpace(in.DeadlineAt)
	if deadline == "" {
		deadline = current.DeadlineAt
	} else {
		if _, err := time.Parse(time.RFC3339Nano, deadline); err != nil {
			if _, err2 := time.Parse(time.RFC3339, deadline); err2 != nil {
				return nil, ErrValidation
			}
		}
	}
	taskType := strings.TrimSpace(in.TaskType)
	if taskType == "" {
		taskType = current.TaskType
	}
	subText := in.SubmissionText
	if subText == nil {
		subText = current.SubmissionText
	}
	subURL := in.SubmissionURL
	if subURL == nil {
		subURL = current.SubmissionURL
	}

	if isKM {
		var reviewerUserID int64
		if err := tx.QueryRowContext(ctx, `SELECT user_id FROM role_assignments WHERE id = ?`, actor.RoleAssignmentID).Scan(&reviewerUserID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, errors.New("role assignment pengubah tidak ditemukan")
			}
			return nil, err
		}
		res, err := tx.ExecContext(ctx, `UPDATE tasks SET
			title = ?, instructions = ?, deadline_at = ?, task_type = ?,
			submission_text = ?, submission_url = ?,
			review_state = 'APPROVED', reviewed_version = ?,
			version = ?, updated_at = ?
			WHERE id = ? AND version = ?`,
			nullable(title), nullable(instructions), nullable(deadline), nullable(taskType),
			nullablePtr(subText), nullablePtr(subURL),
			newVersion, newVersion, now, taskID, current.Version)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return nil, ErrVersionConflict
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_reviews (
			task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note
		) VALUES (?, ?, ?, ?, 'APPROVED', NULL)`,
			taskID, reviewerUserID, actor.RoleAssignmentID, newVersion,
		); err != nil {
			return nil, err
		}
	} else {
		res, err := tx.ExecContext(ctx, `UPDATE tasks SET
			title = ?, instructions = ?, deadline_at = ?, task_type = ?,
			submission_text = ?, submission_url = ?,
			review_state = 'NOT_REVIEWED', reviewed_version = NULL,
			version = ?, updated_at = ?
			WHERE id = ? AND version = ?`,
			nullable(title), nullable(instructions), nullable(deadline), nullable(taskType),
			nullablePtr(subText), nullablePtr(subURL),
			newVersion, now, taskID, current.Version)
		if err != nil {
			return nil, err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return nil, ErrVersionConflict
		}
	}

	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := insertTaskAudit(ctx, tx, actor, "UPDATE", taskID, &beforeJSON, str(taskToJSON(after)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

// ArchiveTask sets archived_at without changing publication lifecycle.
func (r *Repository) ArchiveTask(ctx context.Context, taskID int64, actor ActorInfo) (*Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ? AND t.deleted_at IS NULL`, taskID)
	current, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	beforeJSON := taskToJSON(current)
	now := nowUTC()
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET archived_at = COALESCE(archived_at, ?), updated_at = ? WHERE id = ?`, now, now, taskID); err != nil {
		return nil, err
	}
	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := insertTaskAudit(ctx, tx, actor, "ARCHIVE", taskID, &beforeJSON, str(taskToJSON(after)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

// UnarchiveTask clears archived_at.
func (r *Repository) UnarchiveTask(ctx context.Context, taskID int64, actor ActorInfo) (*Task, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ? AND t.deleted_at IS NULL`, taskID)
	current, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	beforeJSON := taskToJSON(current)
	now := nowUTC()
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET archived_at = NULL, updated_at = ? WHERE id = ?`, now, taskID); err != nil {
		return nil, err
	}
	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := insertTaskAudit(ctx, tx, actor, "UNARCHIVE", taskID, &beforeJSON, str(taskToJSON(after)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

// RestoreTask clears soft-delete while keeping the deletion audit.
func (r *Repository) RestoreTask(ctx context.Context, taskID int64, actor ActorInfo, reason string) (*Task, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("alasan pemulihan wajib diisi")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tasks WHERE id = ? AND deleted_at IS NOT NULL)`, taskID).Scan(&exists); err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}
	beforeRow := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ?`, taskID)
	before, err := scanTaskRow(beforeRow)
	if err != nil {
		return nil, err
	}
	beforeJSON := taskToJSON(before)
	now := nowUTC()
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET deleted_at = NULL, deleted_by_user_id = NULL, updated_at = ? WHERE id = ?`, now, taskID); err != nil {
		return nil, err
	}
	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return nil, err
	}
	if err := insertTaskAudit(ctx, tx, actor, "RESTORE", taskID, &beforeJSON, str(taskToJSON(after)), &reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func getTaskInTx(ctx context.Context, tx *sql.Tx, taskID int64) (*Task, error) {
	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ?`, taskID)
	return scanTaskRow(row)
}

func insertTaskAudit(ctx context.Context, tx *sql.Tx, actor ActorInfo, action string, taskID int64, before, after *string, reason *string) error {
	scope, err := taskScopeInTx(ctx, tx, taskID)
	if err != nil {
		return err
	}
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = nowUTC()
	}
	var actorUser any
	var actorAssignment any
	actorType := "USER"
	if actor.UserID > 0 {
		actorUser = actor.UserID
	} else {
		actorType = "SYSTEM"
	}
	if actor.RoleAssignmentID > 0 {
		actorAssignment = actor.RoleAssignmentID
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id,
		actor_type, action, entity_type, entity_id, before_json, after_json, reason, correlation_id,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 'TASK', ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		scope.ClassID, scope.SemesterID, actorUser, actorAssignment,
		actorType, action, taskID, before, after, reason, corr,
	)
	return err
}

func taskScopeInTx(ctx context.Context, tx *sql.Tx, taskID int64) (AcademicScope, error) {
	var scope AcademicScope
	err := tx.QueryRowContext(ctx, `SELECT sem.class_id, co.semester_id, co.id
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE t.id = ?`, taskID).Scan(&scope.ClassID, &scope.SemesterID, &scope.CourseOfferingID)
	return scope, err
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func nullablePtr(s *string) any {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(*s) == "" {
		return nil
	}
	return *s
}

func str(s string) *string { return &s }

// SoftDeleteTask performs audited soft-delete.
func (r *Repository) SoftDeleteTask(ctx context.Context, taskID int64, actor ActorInfo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ? AND t.deleted_at IS NULL`, taskID)
	current, err := scanTaskRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	deletedBy := actor.UserID
	if deletedBy <= 0 {
		deletedBy = current.CreatedByUserID
	}
	beforeJSON := taskToJSON(current)
	now := nowUTC()
	res, err := tx.ExecContext(ctx, `UPDATE tasks SET deleted_at = ?, deleted_by_user_id = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`, now, deletedBy, now, taskID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	after, err := getTaskInTx(ctx, tx, taskID)
	if err != nil {
		return err
	}
	afterJSON := taskToJSON(after)
	if err := insertTaskAudit(ctx, tx, actor, "DELETE", taskID, &beforeJSON, &afterJSON, nil); err != nil {
		return err
	}
	return tx.Commit()
}
