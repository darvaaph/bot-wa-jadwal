package task

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var ErrNotFound = errors.New("tugas tidak ditemukan")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func strPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func intPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func int64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func scanTaskRow(row interface{ Scan(dest ...any) error }) (*Task, error) {
	var t Task
	var title, instructions, deadlineAt, taskType sql.NullString
	var submissionText, submissionURL sql.NullString
	var reviewedVersion sql.NullInt64
	var publishedAt, completedAt, archivedAt sql.NullString
	var deletedAt sql.NullString
	var deletedBy sql.NullInt64
	var createdAt, updatedAt sql.NullString
	var err error

	err = row.Scan(
		&t.ID,
		&t.CourseOfferingID,
		&title,
		&instructions,
		&deadlineAt,
		&taskType,
		&submissionText,
		&submissionURL,
		&t.PublicationStatus,
		&t.ReviewState,
		&reviewedVersion,
		&t.CreatedByUserID,
		&publishedAt,
		&completedAt,
		&archivedAt,
		&t.Version,
		&deletedAt,
		&deletedBy,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	t.Title = title.String
	t.Instructions = instructions.String
	t.DeadlineAt = deadlineAt.String
	t.TaskType = taskType.String
	t.SubmissionText = strPtr(submissionText)
	t.SubmissionURL = strPtr(submissionURL)
	t.ReviewedVersion = intPtr(reviewedVersion)
	t.PublishedAt = strPtr(publishedAt)
	t.CompletedAt = strPtr(completedAt)
	t.ArchivedAt = strPtr(archivedAt)
	t.DeletedAt = strPtr(deletedAt)
	t.DeletedByUserID = int64Ptr(deletedBy)
	t.CreatedAt = createdAt.String
	t.UpdatedAt = updatedAt.String

	return &t, nil
}

const taskColumns = `t.id, t.course_offering_id, t.title, t.instructions, t.deadline_at,
	t.task_type, t.submission_text, t.submission_url,
	t.publication_status, t.review_state, t.reviewed_version,
	t.created_by_user_id, t.published_at, t.completed_at, t.archived_at,
	t.version, t.deleted_at, t.deleted_by_user_id, t.created_at, t.updated_at`

func (r *Repository) CreateTask(ctx context.Context, in CreateTaskInput) (*Task, error) {
	const query = `INSERT INTO tasks (
		course_offering_id, title, instructions, deadline_at, task_type,
		submission_text, submission_url,
		publication_status, review_state, created_by_user_id, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, 'DRAFT', 'NOT_REVIEWED', ?, 1)`

	res, err := r.db.ExecContext(ctx, query,
		in.CourseOfferingID,
		in.Title,
		in.Instructions,
		in.DeadlineAt,
		in.TaskType,
		in.SubmissionText,
		in.SubmissionURL,
		in.CreatedByUserID,
	)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	row := r.db.QueryRowContext(ctx, `SELECT `+taskColumns+` FROM tasks t WHERE t.id = ?`, id)
	return scanTaskRow(row)
}

func (r *Repository) GetTaskByID(ctx context.Context, id int64) (*TaskItemView, error) {
	const query = `SELECT ` + taskColumns + `, c.code, c.name, cl.code
	FROM tasks t
	JOIN course_offerings co ON co.id = t.course_offering_id
	JOIN courses c ON c.id = co.course_id
	JOIN semesters s ON s.id = co.semester_id
	JOIN classes cl ON cl.id = s.class_id
	WHERE t.id = ?`

	row := r.db.QueryRowContext(ctx, query, id)

	var view TaskItemView
	var title, instructions, deadlineAt, taskType sql.NullString
	var submissionText, submissionURL sql.NullString
	var reviewedVersion sql.NullInt64
	var publishedAt, completedAt, archivedAt sql.NullString
	var deletedAt sql.NullString
	var deletedBy sql.NullInt64
	var createdAt, updatedAt sql.NullString

	err := row.Scan(
		&view.ID,
		&view.CourseOfferingID,
		&title,
		&instructions,
		&deadlineAt,
		&taskType,
		&submissionText,
		&submissionURL,
		&view.PublicationStatus,
		&view.ReviewState,
		&reviewedVersion,
		&view.CreatedByUserID,
		&publishedAt,
		&completedAt,
		&archivedAt,
		&view.Version,
		&deletedAt,
		&deletedBy,
		&createdAt,
		&updatedAt,
		&view.CourseCode,
		&view.CourseName,
		&view.ClassCode,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	view.Title = title.String
	view.Instructions = instructions.String
	view.DeadlineAt = deadlineAt.String
	view.TaskType = taskType.String
	view.SubmissionText = strPtr(submissionText)
	view.SubmissionURL = strPtr(submissionURL)
	view.ReviewedVersion = intPtr(reviewedVersion)
	view.PublishedAt = strPtr(publishedAt)
	view.CompletedAt = strPtr(completedAt)
	view.ArchivedAt = strPtr(archivedAt)
	view.DeletedAt = strPtr(deletedAt)
	view.DeletedByUserID = int64Ptr(deletedBy)
	view.CreatedAt = createdAt.String
	view.UpdatedAt = updatedAt.String

	return &view, nil
}

func (r *Repository) ListTasksByClass(ctx context.Context, classID int64, statusFilter string) ([]TaskItemView, error) {
	query := `SELECT ` + taskColumns + `, c.code, c.name, cl.code
	FROM tasks t
	JOIN course_offerings co ON co.id = t.course_offering_id
	JOIN courses c ON c.id = co.course_id
	JOIN semesters s ON s.id = co.semester_id
	JOIN classes cl ON cl.id = s.class_id
	WHERE s.class_id = ? AND t.deleted_at IS NULL`

	args := []any{classID}
	if strings.TrimSpace(statusFilter) != "" {
		query += ` AND t.publication_status = ?`
		args = append(args, strings.TrimSpace(statusFilter))
	}
	query += ` ORDER BY t.deadline_at ASC, t.id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := []TaskItemView{}
	for rows.Next() {
		var view TaskItemView
		var title, instructions, deadlineAt, taskType sql.NullString
		var submissionText, submissionURL sql.NullString
		var reviewedVersion sql.NullInt64
		var publishedAt, completedAt, archivedAt sql.NullString
		var deletedAt sql.NullString
		var deletedBy sql.NullInt64
		var createdAt, updatedAt sql.NullString

		if err := rows.Scan(
			&view.ID,
			&view.CourseOfferingID,
			&title,
			&instructions,
			&deadlineAt,
			&taskType,
			&submissionText,
			&submissionURL,
			&view.PublicationStatus,
			&view.ReviewState,
			&reviewedVersion,
			&view.CreatedByUserID,
			&publishedAt,
			&completedAt,
			&archivedAt,
			&view.Version,
			&deletedAt,
			&deletedBy,
			&createdAt,
			&updatedAt,
			&view.CourseCode,
			&view.CourseName,
			&view.ClassCode,
		); err != nil {
			return nil, err
		}

		view.Title = title.String
		view.Instructions = instructions.String
		view.DeadlineAt = deadlineAt.String
		view.TaskType = taskType.String
		view.SubmissionText = strPtr(submissionText)
		view.SubmissionURL = strPtr(submissionURL)
		view.ReviewedVersion = intPtr(reviewedVersion)
		view.PublishedAt = strPtr(publishedAt)
		view.CompletedAt = strPtr(completedAt)
		view.ArchivedAt = strPtr(archivedAt)
		view.DeletedAt = strPtr(deletedAt)
		view.DeletedByUserID = int64Ptr(deletedBy)
		view.CreatedAt = createdAt.String
		view.UpdatedAt = updatedAt.String

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return views, nil
}

func validDecision(decision string) bool {
	switch decision {
	case "APPROVED", "CHANGES_REQUESTED", "REVOKED":
		return true
	default:
		return false
	}
}

func (r *Repository) SubmitReview(ctx context.Context, in ReviewTaskInput) error {
	if !validDecision(in.Decision) {
		return errors.New("decision tidak valid")
	}
	if in.Decision != "APPROVED" && (in.Note == nil || strings.TrimSpace(*in.Note) == "") {
		return errors.New("catatan wajib diisi untuk keputusan koreksi atau pembatalan")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var version int
	err = tx.QueryRowContext(ctx, `SELECT version FROM tasks WHERE id = ? AND deleted_at IS NULL`, in.TaskID).Scan(&version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	var reviewerUserID int64
	err = tx.QueryRowContext(ctx, `SELECT user_id FROM role_assignments WHERE id = ?`, in.ReviewerRoleAssignmentID).Scan(&reviewerUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("role assignment pereview tidak ditemukan")
		}
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO task_reviews (
		task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note
	) VALUES (?, ?, ?, ?, ?, ?)`,
		in.TaskID, reviewerUserID, in.ReviewerRoleAssignmentID, version, in.Decision, in.Note,
	)
	if err != nil {
		return err
	}

	switch in.Decision {
	case "APPROVED":
		_, err = tx.ExecContext(ctx, `UPDATE tasks
		SET review_state = 'APPROVED',
			publication_status = 'PUBLISHED',
			published_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
			reviewed_version = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, version, in.TaskID)
	case "CHANGES_REQUESTED":
		_, err = tx.ExecContext(ctx, `UPDATE tasks
		SET review_state = 'CHANGES_REQUESTED',
			publication_status = 'DRAFT',
			reviewed_version = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, version, in.TaskID)
	case "REVOKED":
		_, err = tx.ExecContext(ctx, `UPDATE tasks
		SET review_state = 'REVOKED',
			publication_status = 'REVOKED',
			reviewed_version = ?,
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?`, version, in.TaskID)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) CompleteTask(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE tasks
	SET completed_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
	WHERE id = ? AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteTask(ctx context.Context, id int64, userID int64) error {
	res, err := r.db.ExecContext(ctx, `UPDATE tasks
	SET deleted_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
		deleted_by_user_id = ?,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
	WHERE id = ? AND deleted_at IS NULL`, userID, id)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
