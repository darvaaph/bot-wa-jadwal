package task

import (
	"bot-jadwal/internal/audit"
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
	ClassID          int64     `json:"class_id"`
	CourseOfferingID *int64    `json:"course_offering_id,omitempty"`
	TaskID           *int64    `json:"task_id,omitempty"`
	Title            string    `json:"title"`
	MaterialType     string    `json:"material_type"`
	URL              string    `json:"url"`
	Description      *string   `json:"description,omitempty"`
	Visibility       string    `json:"visibility"`
	CreatedByUserID  int64     `json:"created_by_user_id"`
	Actor            ActorInfo `json:"-"`
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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	err = tx.QueryRowContext(ctx, `INSERT INTO materials (
		class_id, course_offering_id, task_id, title, material_type, url, description,
		visibility, status, created_by_user_id, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, 1) RETURNING id`,
		in.ClassID, in.CourseOfferingID, in.TaskID, title, mtype, url, in.Description, vis, in.CreatedByUserID,
	).Scan(&id)
	if err != nil {
		return nil, err
	}
	row := tx.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id = ?`, id)
	created, err := scanMaterialRow(row)
	if err != nil {
		return nil, err
	}
	if err := insertMaterialAudit(ctx, tx, in.Actor, "CREATE", created, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return created, nil
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
	var actorEntry audit.Actor
	if actor.UserID > 0 {
		uid := actor.UserID
		actorEntry = audit.Actor{Type: "USER", UserID: &uid}
		if actor.RoleAssignmentID > 0 {
			raid := actor.RoleAssignmentID
			actorEntry.RoleAssignmentID = &raid
		}
	} else {
		actorEntry = audit.Actor{Type: "SYSTEM"}
	}
	var classPtr *int64
	if actor.ClassID != nil {
		classPtr = actor.ClassID
	} else {
		v := current.ClassID
		classPtr = &v
	}
	beforeStr := `{"title":"` + strings.ReplaceAll(current.Title, `"`, ``) + `"}`
	afterStr := `{"title":"` + strings.ReplaceAll(after.Title, `"`, ``) + `"}`
	eid := id
	if err := audit.Write(ctx, tx, audit.Entry{
		Actor:         actorEntry,
		ClassID:       classPtr,
		SemesterID:    actor.SemesterID,
		Action:        "UPDATE",
		EntityType:    "MATERIAL",
		EntityID:      &eid,
		BeforeJSON:    &beforeStr,
		AfterJSON:     &afterStr,
		CorrelationID: corr,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return after, nil
}

func (r *Repository) SoftDeleteMaterial(ctx context.Context, id int64, actor ActorInfo) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `SELECT `+materialColumns+` FROM materials WHERE id = ? AND deleted_at IS NULL`, id)
	current, err := scanMaterialRow(row)
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := tx.ExecContext(ctx, `UPDATE materials SET deleted_at=?, deleted_by_user_id=?, updated_at=?
		WHERE id=? AND deleted_at IS NULL`, now, deletedBy, now, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	if err := insertMaterialAudit(ctx, tx, actor, "DELETE", current, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// insertMaterialAudit writes one MATERIAL audit row inside the caller's transaction.
func insertMaterialAudit(ctx context.Context, tx *sql.Tx, actor ActorInfo, action string, m *Material, beforeTitle *string) error {
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = time.Now().UTC().Format(time.RFC3339Nano)
	}
	var actorEntry audit.Actor
	if actor.UserID > 0 {
		uid := actor.UserID
		actorEntry = audit.Actor{Type: "USER", UserID: &uid}
		if actor.RoleAssignmentID > 0 {
			raid := actor.RoleAssignmentID
			actorEntry.RoleAssignmentID = &raid
		}
	} else {
		actorEntry = audit.Actor{Type: "SYSTEM"}
	}
	var classPtr *int64
	if actor.ClassID != nil {
		classPtr = actor.ClassID
	} else {
		v := m.ClassID
		classPtr = &v
	}
	before := `{}`
	if beforeTitle != nil {
		before = `{"title":"` + strings.ReplaceAll(*beforeTitle, `"`, ``) + `"}`
	}
	after := `{"title":"` + strings.ReplaceAll(m.Title, `"`, ``) + `"}`
	eid := m.ID
	return audit.Write(ctx, tx, audit.Entry{
		Actor:         actorEntry,
		ClassID:       classPtr,
		SemesterID:    actor.SemesterID,
		Action:        action,
		EntityType:    "MATERIAL",
		EntityID:      &eid,
		BeforeJSON:    &before,
		AfterJSON:     &after,
		CorrelationID: corr,
	})
}

// MaterialScope carries the ownership of a material for pre-write authorization.
type MaterialScope struct {
	ClassID          int64
	CourseOfferingID *int64
}

// GetMaterialScope returns the class (and optional offering) of an active material.
func (r *Repository) GetMaterialScope(ctx context.Context, id int64) (MaterialScope, error) {
	var scope MaterialScope
	var offeringID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT class_id, course_offering_id FROM materials WHERE id = ? AND deleted_at IS NULL`,
		id).Scan(&scope.ClassID, &offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return scope, ErrNotFound
	}
	if err != nil {
		return scope, err
	}
	scope.CourseOfferingID = int64Ptr(offeringID)
	return scope, nil
}
