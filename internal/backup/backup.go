package backup

import (
	"bot-jadwal/internal/audit"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("backup tidak ditemukan")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrMismatch     = errors.New("paket backup tidak cocok dengan tujuan")
)

type Service struct {
	db  *sql.DB
	dir string
}

func NewService(db *sql.DB, dir string) *Service {
	if dir == "" {
		dir = "storage/backups"
	}
	return &Service{db: db, dir: dir}
}

type Record struct {
	ID          int64  `json:"id"`
	ClassID     int64  `json:"class_id"`
	SemesterID  *int64 `json:"semester_id,omitempty"`
	ArtifactRef string `json:"artifact_ref"`
	Checksum    string `json:"checksum"`
	Status      string `json:"status"`
	CreatedBy   int64  `json:"created_by_user_id"`
	Reason      string `json:"reason"`
	CreatedAt   string `json:"created_at"`
}

type dump struct {
	Version    int              `json:"version"`
	ClassID    int64            `json:"class_id"`
	SemesterID *int64           `json:"semester_id,omitempty"`
	Class      map[string]any   `json:"class"`
	Settings   map[string]any   `json:"settings,omitempty"`
	Semesters  []map[string]any `json:"semesters"`
	Courses    []map[string]any `json:"courses"`
	Offerings  []map[string]any `json:"offerings"`
	Lecturers  []map[string]any `json:"lecturers"`
	OffLect    []map[string]any `json:"offering_lecturers"`
	Patterns   []map[string]any `json:"patterns"`
	Events     []map[string]any `json:"events"`
	Parts      []map[string]any `json:"participants"`
	Confirms   []map[string]any `json:"confirmations"`
	Tasks      []map[string]any `json:"tasks"`
	Reviews    []map[string]any `json:"reviews"`
	Materials  []map[string]any `json:"materials"`
	Channels   []map[string]any `json:"channels"`
}

func (s *Service) Create(ctx context.Context, classID int64, semesterID *int64, userID int64, reason string) (*Record, error) {
	if classID <= 0 || userID <= 0 || strings.TrimSpace(reason) == "" {
		return nil, ErrInvalidInput
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM classes WHERE id = ?)`, classID).Scan(&exists); err != nil || !exists {
		return nil, ErrNotFound
	}
	if semesterID != nil {
		var c int64
		if err := s.db.QueryRowContext(ctx, `SELECT class_id FROM semesters WHERE id = ?`, *semesterID).Scan(&c); err != nil || c != classID {
			return nil, ErrInvalidInput
		}
	}
	d, err := s.buildDump(ctx, classID, semesterID)
	if err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	check := hex.EncodeToString(sum[:])
	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}
	name := fmt.Sprintf("backup-c%d", classID)
	if semesterID != nil {
		name += fmt.Sprintf("-s%d", *semesterID)
	}
	var randSuffix [4]byte
	_, _ = rand.Read(randSuffix[:])
	name += "-" + time.Now().UTC().Format("20060102T150405") + "-" + hex.EncodeToString(randSuffix[:]) + ".json"
	path := filepath.Join(s.dir, name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(path)
		}
	}()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO backup_records
		(class_id, semester_id, artifact_ref, checksum, status, created_by_user_id, reason, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'READY', ?, ?, ?, ?) RETURNING id`,
		classID, semesterID, path, check, userID, strings.TrimSpace(reason), now, now).Scan(&id)
	if err != nil {
		return nil, err
	}
	corr := check[:16]
	afterBytes, _ := json.Marshal(map[string]string{"checksum": check})
	afterStr := string(afterBytes)
	uid := userID
	eid := id
	if err := audit.Write(ctx, tx, audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid},
		ClassID:       &classID,
		SemesterID:    semesterID,
		Action:        "BACKUP",
		EntityType:    "BACKUP",
		EntityID:      &eid,
		AfterJSON:     &afterStr,
		Reason:        strings.TrimSpace(reason),
		CorrelationID: corr,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	var r Record
	var semID sql.NullInt64
	var createdAt string
	err := s.db.QueryRowContext(ctx, `SELECT id, class_id, semester_id, artifact_ref, checksum, status, created_by_user_id, reason, created_at
		FROM backup_records WHERE id = ?`, id).Scan(
		&r.ID, &r.ClassID, &semID, &r.ArtifactRef, &r.Checksum, &r.Status, &r.CreatedBy, &r.Reason, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if semID.Valid {
		r.SemesterID = &semID.Int64
	}
	r.CreatedAt = createdAt
	return &r, nil
}

func (s *Service) List(ctx context.Context, classID *int64) ([]Record, error) {
	query := `SELECT id, class_id, semester_id, artifact_ref, checksum, status, created_by_user_id, reason, created_at FROM backup_records`
	args := []any{}
	if classID != nil {
		query += ` WHERE class_id = ?`
		args = append(args, *classID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 100`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		var r Record
		var semID sql.NullInt64
		if err := rows.Scan(&r.ID, &r.ClassID, &semID, &r.ArtifactRef, &r.Checksum, &r.Status, &r.CreatedBy, &r.Reason, &r.CreatedAt); err != nil {
			return nil, err
		}
		if semID.Valid {
			r.SemesterID = &semID.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Service) buildDump(ctx context.Context, classID int64, semesterID *int64) (*dump, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d := &dump{Version: 2, ClassID: classID, SemesterID: semesterID}
	var loadErr error
	load := func(query string, args []any) []map[string]any {
		if loadErr != nil {
			return nil
		}
		rows, err := rowsMap(ctx, tx, query, args)
		if err != nil {
			loadErr = err
		}
		return rows
	}
	semWhere := "sem.class_id = ?"
	semArgs := []any{classID}
	if semesterID != nil {
		semWhere += " AND sem.id = ?"
		semArgs = append(semArgs, *semesterID)
	}
	if cls := load(`SELECT id, code, slug, study_program, cohort_year, group_label, status FROM classes WHERE id = ?`, []any{classID}); len(cls) > 0 {
		d.Class = cls[0]
	} else {
		d.Class = map[string]any{}
	}
	d.Semesters = load(`SELECT id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at FROM semesters sem WHERE `+semWhere, semArgs)
	d.Offerings = load(`SELECT co.id, co.semester_id, co.course_id, co.activity_type, co.display_name, co.status FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.OffLect = load(`SELECT ol.id, ol.course_offering_id, ol.lecturer_id, ol.responsibility, ol.superseded_at FROM offering_lecturers ol JOIN course_offerings co ON co.id = ol.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND ol.superseded_at IS NULL`, semArgs)
	d.Patterns = load(`SELECT sp.id, sp.course_offering_id, sp.room_id, sp.day_of_week, sp.start_time, sp.end_time, sp.effective_from, sp.effective_until, sp.status, sp.version, sp.meeting_link FROM schedule_patterns sp JOIN course_offerings co ON co.id = sp.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Events = load(`SELECT DISTINCT te.id, te.origin_schedule_pattern_id, te.origin_occurrence_date, te.result_schedule_pattern_id, te.event_kind, te.starts_at, te.ends_at, te.room_id, te.reason, te.lifecycle_status, te.published_by_user_id, te.published_at, te.revoked_by_user_id, te.revoked_at, te.revocation_reason, te.version, te.meeting_link FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role='OWNER' JOIN course_offerings co ON co.id = teo.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Parts = load(`SELECT teo.id, teo.teaching_event_id, teo.course_offering_id, teo.participation_role, teo.participation_status FROM teaching_event_offerings teo WHERE teo.teaching_event_id IN (SELECT te.id FROM teaching_events te JOIN teaching_event_offerings own ON own.teaching_event_id=te.id AND own.participation_role='OWNER' JOIN course_offerings co ON co.id=own.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE `+semWhere+`)`, semArgs)
	d.Confirms = load(`SELECT rc.id, rc.teaching_event_id, rc.room_id, rc.confirmation_status, rc.external_contact, rc.note, rc.recorded_by_user_id, rc.recorded_at, rc.confirmed_at, rc.superseded_at FROM room_confirmations rc JOIN teaching_event_offerings teo ON teo.teaching_event_id = rc.teaching_event_id AND teo.participation_role='OWNER' JOIN course_offerings co ON co.id = teo.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND rc.superseded_at IS NULL`, semArgs)
	d.Tasks = load(`SELECT t.id, t.course_offering_id, t.title, t.instructions, t.deadline_at, t.task_type, t.submission_text, t.submission_url, t.publication_status, t.review_state, t.reviewed_version, t.created_by_user_id, t.published_at, t.completed_at, t.archived_at, t.version, t.deleted_at, t.deleted_by_user_id FROM tasks t JOIN course_offerings co ON co.id = t.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND t.deleted_at IS NULL`, semArgs)
	d.Reviews = load(`SELECT tr.id, tr.task_id, tr.reviewer_user_id, tr.reviewer_role_assignment_id, tr.task_version, tr.decision, tr.note FROM task_reviews tr JOIN tasks t ON t.id = tr.task_id JOIN course_offerings co ON co.id = t.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND t.deleted_at IS NULL`, semArgs)
	materialQuery := `SELECT m.id, m.class_id, m.course_offering_id, m.task_id, m.title, m.material_type, m.url, m.description, m.visibility, m.status, m.created_by_user_id, m.version, m.deleted_at, m.deleted_by_user_id FROM materials m WHERE m.class_id = ? AND m.deleted_at IS NULL`
	materialArgs := []any{classID}
	if semesterID != nil {
		materialQuery += ` AND m.course_offering_id IN (SELECT id FROM course_offerings WHERE semester_id=?)`
		materialArgs = append(materialArgs, *semesterID)
	}
	d.Materials = load(materialQuery, materialArgs)
	// Referenced masters.
	d.Courses = load(`SELECT DISTINCT c.id, c.code, c.name FROM courses c JOIN course_offerings co ON co.course_id = c.id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Lecturers = load(`SELECT DISTINCT l.id, l.code, l.full_name FROM lecturers l JOIN offering_lecturers ol ON ol.lecturer_id = l.id JOIN course_offerings co ON co.id = ol.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND ol.superseded_at IS NULL`, semArgs)
	if loadErr != nil {
		return nil, loadErr
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return d, nil
}

type rowQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func rowsMap(ctx context.Context, db rowQueryer, query string, args []any) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := map[string]any{}
		for i, c := range cols {
			m[c] = vals[i]
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func int64Val(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case []byte:
		var x int64
		fmt.Sscanf(string(n), "%d", &x)
		return x
	case string:
		var x int64
		fmt.Sscanf(n, "%d", &x)
		return x
	default:
		return 0
	}
}
func strVal(v any) string {
	switch s := v.(type) {
	case string:
		return s
	case []byte:
		return string(s)
	case nil:
		return ""
	default:
		return fmt.Sprint(s)
	}
}
func nullInt64(v any) any {
	if v == nil {
		return nil
	}
	switch n := v.(type) {
	case int64:
		if n == 0 {
			return nil
		}
		return n
	case []byte:
		if len(n) == 0 {
			return nil
		}
		return string(n)
	case string:
		if n == "" {
			return nil
		}
		return n
	default:
		return v
	}
}
