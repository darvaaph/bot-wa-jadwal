package task

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

type Material struct {
	ID               int64   `json:"id"`
	ClassID          int64   `json:"class_id"`
	CourseOfferingID *int64  `json:"course_offering_id,omitempty"`
	TaskID           *int64  `json:"task_id,omitempty"`
	Title            string  `json:"title"`
	MaterialType     string  `json:"material_type"`
	URL              string  `json:"url"`
	Description      *string `json:"description,omitempty"`
	Visibility       string  `json:"visibility"`
	Status           string  `json:"status"`
	CreatedByUserID  int64   `json:"created_by_user_id"`
	Version          int     `json:"version"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

type CreateMaterialInput struct {
	ClassID          int64   `json:"class_id"`
	CourseOfferingID *int64  `json:"course_offering_id,omitempty"`
	TaskID           *int64  `json:"task_id,omitempty"`
	Title            string  `json:"title"`
	MaterialType     string  `json:"material_type"`
	URL              string  `json:"url"`
	Description      *string `json:"description,omitempty"`
	Visibility       string  `json:"visibility"`
	CreatedByUserID  int64   `json:"created_by_user_id"`
}

type UpdateMaterialInput struct {
	Title           string
	Description     *string
	URL             string
	Visibility      string
	Status          string
	ExpectedVersion int
}

func validMaterialType(s string) bool {
	switch s {
	case "DOCUMENT", "MEETING", "REPOSITORY", "PORTAL", "OTHER":
		return true
	}
	return false
}

func validVisibility(s string) bool { return s == "CLASS_ACCESS" || s == "WHATSAPP_ONLY" }

func normalizeURL(u string) (string, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", errors.New("url wajib diisi")
	}
	if !(strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://")) {
		return "", errors.New("url harus diawali http:// atau https://")
	}
	if len(u) > 2048 {
		return "", errors.New("url terlalu panjang")
	}
	return u, nil
}

func scanMaterialRow(row interface{ Scan(dest ...any) error }) (*Material, error) {
	var m Material
	var offeringID, taskID sql.NullInt64
	var desc sql.NullString
	var createdAt, updatedAt sql.NullString
	var deletedAt sql.NullString
	var deletedBy sql.NullInt64
	err := row.Scan(
		&m.ID, &m.ClassID, &offeringID, &taskID,
		&m.Title, &m.MaterialType, &m.URL, &desc,
		&m.Visibility, &m.Status, &m.CreatedByUserID, &m.Version,
		&deletedAt, &deletedBy, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	m.CourseOfferingID = int64Ptr(offeringID)
	m.TaskID = int64Ptr(taskID)
	m.Description = strPtr(desc)
	m.CreatedAt = createdAt.String
	m.UpdatedAt = updatedAt.String
	return &m, nil
}

const materialColumns = `id, class_id, course_offering_id, task_id, title, material_type, url, description, visibility, status, created_by_user_id, version, deleted_at, deleted_by_user_id, created_at, updated_at`

func (r *Repository) CreateMaterial(ctx context.Context, in CreateMaterialInput) (*Material, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, errors.New("judul materi wajib diisi")
	}
	mtype := strings.ToUpper(strings.TrimSpace(in.MaterialType))
	if mtype == "" {
		mtype = "OTHER"
	}
	if !validMaterialType(mtype) {
		return nil, errors.New("material_type tidak valid")
	}
	url, err := normalizeURL(in.URL)
	if err != nil {
		return nil, err
	}
	vis := strings.ToUpper(strings.TrimSpace(in.Visibility))
	if vis == "" {
		vis = "CLASS_ACCESS"
	}
	if !validVisibility(vis) {
		return nil, errors.New("visibility tidak valid")
	}
	if in.ClassID <= 0 {
		return nil, errors.New("class_id tidak valid")
	}
	if in.CreatedByUserID <= 0 {
		return nil, errors.New("created_by_user_id tidak valid")
	}
	if in.CourseOfferingID != nil {
		var classID int64
		err := r.db.QueryRowContext(ctx, `SELECT sem.class_id FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?`, *in.CourseOfferingID).Scan(&classID)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("course offering tidak ditemukan")
		}
		if err != nil {
			return nil, err
		}
		if classID != in.ClassID {
			return nil, errors.New("course offering bukan milik kelas ini")
		}
	}
	if in.TaskID != nil {
		scope, err := r.GetTaskScope(ctx, *in.TaskID)
		if err != nil {
			return nil, errors.New("tugas referensi tidak ditemukan")
		}
		if scope.ClassID != in.ClassID {
			return nil, errors.New("tugas referensi bukan milik kelas ini")
		}
		if in.CourseOfferingID != nil && scope.CourseOfferingID != *in.CourseOfferingID {
			return nil, errors.New("tugas referensi bukan milik offering ini")
		}
	}

	var id int64
	err = r.db.QueryRowContext(ctx, `INSERT INTO materials (
		class_id, course_offering_id, task_id, title, material_type, url, description,
		visibility, status, created_by_user_id, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, 1) RETURNING id`,
		in.ClassID, in.CourseOfferingID, in.TaskID, title, mtype, url, in.Description, vis, in.CreatedByUserID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id = ?`, id)
	return scanMaterialRow(row)
}

func (r *Repository) ListMaterialsByClass(ctx context.Context, classID int64, offeringID *int64) ([]Material, error) {
	query := `SELECT ` + materialColumns + ` FROM materials WHERE class_id = ? AND deleted_at IS NULL`
	args := []any{classID}
	if offeringID != nil {
		query += ` AND (course_offering_id IS NULL OR course_offering_id = ?)`
		args = append(args, *offeringID)
	}
	query += ` ORDER BY updated_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Material{}
	for rows.Next() {
		m, err := scanMaterialRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateMaterial(ctx context.Context, id int64, in UpdateMaterialInput, actor ActorInfo) (*Material, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id = ? AND deleted_at IS NULL`, id)
	current, err := scanMaterialRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if in.ExpectedVersion > 0 && current.Version != in.ExpectedVersion {
		return nil, ErrVersionConflict
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = current.Title
	}
	url := strings.TrimSpace(in.URL)
	if url == "" {
		url = current.URL
	} else {
		var normErr error
		url, normErr = normalizeURL(url)
		if normErr != nil {
			return nil, normErr
		}
	}
	vis := strings.ToUpper(strings.TrimSpace(in.Visibility))
	if vis == "" {
		vis = current.Visibility
	}
	if !validVisibility(vis) {
		return nil, errors.New("visibility tidak valid")
	}
	status := strings.ToUpper(strings.TrimSpace(in.Status))
	if status == "" {
		status = current.Status
	}
	if status != "ACTIVE" && status != "INACTIVE" && status != "ARCHIVED" {
		return nil, errors.New("status materi tidak valid")
	}
	desc := in.Description
	if desc == nil {
		desc = current.Description
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `UPDATE materials SET title=?, url=?, description=?, visibility=?, status=?, version=version+1, updated_at=?
		WHERE id=? AND version=? AND deleted_at IS NULL`,
		title, url, desc, vis, status, now, id, current.Version)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, ErrVersionConflict
	}
	afterRow := tx.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id=?`, id)
	after, err := scanMaterialRow(afterRow)
	if err != nil {
		return nil, err
	}
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = now
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
	var semID any
	if actor.SemesterID != nil {
		semID = *actor.SemesterID
	}
	var classID any = current.ClassID
	if actor.ClassID != nil {
		classID = *actor.ClassID
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, before_json, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, 'UPDATE', 'MATERIAL', ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, semID, actorUser, actorAssignment, actorType, id,
		`{"title":"`+strings.ReplaceAll(current.Title, `"`, ``)+`"}`, `{"title":"`+strings.ReplaceAll(after.Title, `"`, ``)+`"}`, corr,
	); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (r *Repository) SoftDeleteMaterial(ctx context.Context, id int64, actor ActorInfo) error {
	deletedBy := actor.UserID
	if deletedBy <= 0 {
		if err := r.db.QueryRowContext(ctx, `SELECT created_by_user_id FROM materials WHERE id = ?`, id).Scan(&deletedBy); err != nil {
			return ErrNotFound
		}
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := r.db.ExecContext(ctx, `UPDATE materials SET deleted_at=?, deleted_by_user_id=?, updated_at=?
		WHERE id=? AND deleted_at IS NULL`, now, actor.UserID, now, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return nil
}
