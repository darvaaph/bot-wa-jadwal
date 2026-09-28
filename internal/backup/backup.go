package backup

import (
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
	if err := os.WriteFile(path, data, 0644); err != nil {
		return nil, err
	}
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
	if _, err := tx.ExecContext(ctx, `UPDATE backup_records SET status='VERIFIED', verified_at=?, updated_at=? WHERE id=?`, now, now, id); err != nil {
		return nil, err
	}
	corr := check[:16]
	afterJSON, _ := json.Marshal(map[string]string{"artifact": path})
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (class_id, semester_id, actor_user_id, actor_type, action,
		entity_type, entity_id, after_json, reason, correlation_id, created_at, updated_at)
		VALUES (?, ?, ?, 'USER', 'BACKUP', 'BACKUP', ?, ?, ?, ?, ?, ?)`,
		classID, semesterID, userID, id, string(afterJSON), strings.TrimSpace(reason), corr, now, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
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

// Restore verifies package scope, saves a pre-restore point, then replaces class-scope data.
func (s *Service) Restore(ctx context.Context, backupID, userID int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrInvalidInput
	}
	rec, err := s.Get(ctx, backupID)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(rec.ArtifactRef)
	if err != nil {
		return fmt.Errorf("artefak backup tidak terbaca: %w", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != rec.Checksum {
		return errors.New("checksum artefak tidak cocok")
	}
	var d dump
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}
	if d.ClassID != rec.ClassID {
		return ErrMismatch
	}
	if rec.SemesterID != nil && (d.SemesterID == nil || *d.SemesterID != *rec.SemesterID) {
		return ErrMismatch
	}
	// Pre-restore point.
	if _, err := s.Create(ctx, rec.ClassID, rec.SemesterID, userID, "pre-restore backup #"+fmt.Sprint(rec.ID)); err != nil {
		return fmt.Errorf("gagal membuat titik pemulihan: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.restoreDumpTx(ctx, tx, &d, rec); err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `UPDATE backup_records SET status='RESTORING', verified_at=NULL, updated_at=? WHERE id=?`, now, backupID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (class_id, semester_id, actor_user_id, actor_type, action,
		entity_type, entity_id, reason, correlation_id, created_at, updated_at)
		VALUES (?, ?, ?, 'USER', 'RESTORE', 'BACKUP', ?, ?, ?, ?, ?)`,
		rec.ClassID, rec.SemesterID, userID, backupID, strings.TrimSpace(reason), rec.Checksum[:16], now, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE backup_records SET status='VERIFIED', verified_at=?, updated_at=? WHERE id=?`, now, now, backupID); err != nil {
		return err
	}
	return tx.Commit()
}

func semesterScope(semID *int64) string {
	if semID != nil {
		return " AND sem.id = " + fmt.Sprint(*semID)
	}
	return ""
}

func (s *Service) buildDump(ctx context.Context, classID int64, semesterID *int64) (*dump, error) {
	d := &dump{Version: 1, ClassID: classID, SemesterID: semesterID}
	semWhere := "sem.class_id = ?"
	semArgs := []any{classID}
	if semesterID != nil {
		semWhere += " AND sem.id = ?"
		semArgs = append(semArgs, *semesterID)
	}
	if cls := rowsMap(ctx, s.db, `SELECT id, code, slug, study_program, cohort_year, group_label, status FROM classes WHERE id = ?`, []any{classID}); len(cls) > 0 {
		d.Class = cls[0]
	} else {
		d.Class = map[string]any{}
	}
	if st := rowsMap(ctx, s.db, `SELECT class_id, timezone, portal_access_mode, meeting_link_visibility, morning_reminder_time, afternoon_reminder_time, replacement_reminder_minutes FROM class_settings WHERE class_id = ?`, []any{classID}); len(st) > 0 {
		d.Settings = st[0]
	}
	d.Semesters = rowsMap(ctx, s.db, `SELECT id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at FROM semesters sem WHERE `+semWhere, semArgs)
	d.Offerings = rowsMap(ctx, s.db, `SELECT co.id, co.semester_id, co.course_id, co.activity_type, co.display_name, co.status FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.OffLect = rowsMap(ctx, s.db, `SELECT ol.course_offering_id, ol.lecturer_id, ol.responsibility FROM offering_lecturers ol JOIN course_offerings co ON co.id = ol.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Patterns = rowsMap(ctx, s.db, `SELECT sp.id, sp.course_offering_id, sp.room_id, sp.day_of_week, sp.start_time, sp.end_time, sp.effective_from, sp.effective_until, sp.status FROM schedule_patterns sp JOIN course_offerings co ON co.id = sp.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Events = rowsMap(ctx, s.db, `SELECT DISTINCT te.id, te.origin_schedule_pattern_id, te.origin_occurrence_date, te.result_schedule_pattern_id, te.event_kind, te.starts_at, te.ends_at, te.room_id, te.reason, te.lifecycle_status, te.published_by_user_id, te.published_at, te.revoked_by_user_id, te.revoked_at, te.revocation_reason FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id JOIN course_offerings co ON co.id = teo.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Parts = rowsMap(ctx, s.db, `SELECT teo.teaching_event_id, teo.course_offering_id, teo.participation_role, teo.participation_status FROM teaching_event_offerings teo JOIN course_offerings co ON co.id = teo.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Confirms = rowsMap(ctx, s.db, `SELECT rc.teaching_event_id, rc.room_id, rc.confirmation_status, rc.external_contact, rc.note, rc.recorded_by_user_id, rc.recorded_at, rc.confirmed_at FROM room_confirmations rc JOIN teaching_events te ON te.id = rc.teaching_event_id JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id JOIN course_offerings co ON co.id = teo.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` GROUP BY rc.teaching_event_id, rc.room_id, rc.confirmation_status, rc.external_contact, rc.note, rc.recorded_by_user_id, rc.recorded_at, rc.confirmed_at`, semArgs)
	d.Tasks = rowsMap(ctx, s.db, `SELECT t.id, t.course_offering_id, t.title, t.instructions, t.deadline_at, t.task_type, t.submission_text, t.submission_url, t.publication_status, t.review_state, t.reviewed_version, t.created_by_user_id, t.published_at, t.completed_at, t.archived_at, t.version FROM tasks t JOIN course_offerings co ON co.id = t.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND t.deleted_at IS NULL`, semArgs)
	d.Reviews = rowsMap(ctx, s.db, `SELECT tr.task_id, tr.reviewer_user_id, tr.reviewer_role_assignment_id, tr.task_version, tr.decision, tr.note FROM task_reviews tr JOIN tasks t ON t.id = tr.task_id JOIN course_offerings co ON co.id = t.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere+` AND t.deleted_at IS NULL`, semArgs)
	d.Materials = rowsMap(ctx, s.db, `SELECT m.id, m.class_id, m.course_offering_id, m.task_id, m.title, m.material_type, m.url, m.description, m.visibility, m.status, m.created_by_user_id FROM materials m WHERE m.class_id = ? AND m.deleted_at IS NULL`, []any{classID})
	d.Channels = rowsMap(ctx, s.db, `SELECT id, class_id, jid, channel_type, display_name, status FROM whatsapp_channels WHERE class_id = ?`, []any{classID})
	// Referenced masters.
	d.Courses = rowsMap(ctx, s.db, `SELECT DISTINCT c.id, c.code, c.name FROM courses c JOIN course_offerings co ON co.course_id = c.id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	d.Lecturers = rowsMap(ctx, s.db, `SELECT DISTINCT l.id, l.code, l.full_name FROM lecturers l JOIN offering_lecturers ol ON ol.lecturer_id = l.id JOIN course_offerings co ON co.id = ol.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE `+semWhere, semArgs)
	return d, nil
}

// restoreDumpTx replaces class-scope academic data from dump. Global masters upserted.
func (s *Service) restoreDumpTx(ctx context.Context, tx *sql.Tx, d *dump, rec *Record) error {
	var scopeArgs []any
	ownerEventScope := `SELECT te.id FROM teaching_events te
		JOIN teaching_event_offerings owner ON owner.teaching_event_id = te.id AND owner.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = owner.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE sem.class_id = ?`
	offeringScope := `SELECT co.id FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id WHERE sem.class_id = ?`
	semesterScope := `SELECT id FROM semesters WHERE class_id = ?`
	taskScope := `SELECT t.id FROM tasks t JOIN course_offerings co ON co.id = t.course_offering_id JOIN semesters sem ON sem.id = co.semester_id WHERE sem.class_id = ?`
	scopeArgs = []any{rec.ClassID}
	if rec.SemesterID != nil {
		ownerEventScope += ` AND sem.id = ?`
		offeringScope += ` AND sem.id = ?`
		semesterScope += ` AND id = ?`
		taskScope += ` AND sem.id = ?`
		scopeArgs = append(scopeArgs, *rec.SemesterID)
	}
	// Wipe class-scope mutable data (audits/imports/backups/notifications/channels preserved).
	// 1. Materials: dumps are class-scoped, so wipe all of the class's materials.
	if _, err := tx.ExecContext(ctx, `DELETE FROM materials WHERE class_id = ?`, rec.ClassID); err != nil {
		return fmt.Errorf("wipe materials: %w", err)
	}
	// 2-3. Reviews then tasks.
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_reviews WHERE task_id IN (`+taskScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe reviews: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id IN (`+taskScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe tasks: %w", err)
	}
	// 4-6. Confirmations, participations, then events (owner-scoped).
	if _, err := tx.ExecContext(ctx, `DELETE FROM room_confirmations WHERE teaching_event_id IN (`+ownerEventScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe confirmations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM teaching_event_offerings WHERE teaching_event_id IN (`+ownerEventScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe participations: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM teaching_events WHERE id IN (`+ownerEventScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe events: %w", err)
	}
	// 7-8. Patterns, offering lecturers (dependents, safe to wipe).
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_patterns WHERE course_offering_id IN (`+offeringScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe patterns: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM offering_lecturers WHERE course_offering_id IN (`+offeringScope+`)`, scopeArgs...); err != nil {
		return fmt.Errorf("wipe offering lecturers: %w", err)
	}
	// 9-10. Structural rows (semesters/offerings) are UPSERTED, not deleted:
	// role_assignments and invitations reference them and must survive restore.
	// Reinsert masters (upsert, keep IDs stable via explicit id insert).
	for _, c := range d.Courses {
		if _, err := tx.ExecContext(ctx, `INSERT INTO courses (id, code, name, status) VALUES (?, ?, ?, 'ACTIVE')
			ON CONFLICT(id) DO UPDATE SET code=excluded.code, name=excluded.name`, int64Val(c["id"]), strVal(c["code"]), strVal(c["name"])); err != nil {
			return err
		}
	}
	for _, l := range d.Lecturers {
		if _, err := tx.ExecContext(ctx, `INSERT INTO lecturers (id, code, full_name, status) VALUES (?, ?, ?, 'ACTIVE')
			ON CONFLICT(id) DO UPDATE SET code=excluded.code, full_name=excluded.full_name`, int64Val(l["id"]), strVal(l["code"]), strVal(l["full_name"])); err != nil {
			return err
		}
	}
	for _, sm := range d.Semesters {
		if _, err := tx.ExecContext(ctx, `INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET academic_year=excluded.academic_year, term=excluded.term,
			starts_on=excluded.starts_on, ends_on=excluded.ends_on, status=excluded.status,
			published_at=excluded.published_at, activated_at=excluded.activated_at, archived_at=excluded.archived_at`,
			int64Val(sm["id"]), rec.ClassID, strVal(sm["academic_year"]), strVal(sm["term"]), strVal(sm["starts_on"]), strVal(sm["ends_on"]), strVal(sm["status"]), sm["published_at"], sm["activated_at"], sm["archived_at"]); err != nil {
			return fmt.Errorf("restore semesters: %w", err)
		}
	}
	for _, o := range d.Offerings {
		if _, err := tx.ExecContext(ctx, `INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET semester_id=excluded.semester_id, course_id=excluded.course_id,
			activity_type=excluded.activity_type, display_name=excluded.display_name, status=excluded.status`,
			int64Val(o["id"]), int64Val(o["semester_id"]), int64Val(o["course_id"]), strVal(o["activity_type"]), strVal(o["display_name"]), strVal(o["status"])); err != nil {
			return fmt.Errorf("restore offerings: %w", err)
		}
	}
	for _, ol := range d.OffLect {
		if _, err := tx.ExecContext(ctx, `INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility) VALUES (?, ?, ?)
			ON CONFLICT(course_offering_id, lecturer_id) DO NOTHING`, int64Val(ol["course_offering_id"]), int64Val(ol["lecturer_id"]), strVal(ol["responsibility"])); err != nil {
			return err
		}
	}
	for _, p := range d.Patterns {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, effective_until, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, int64Val(p["id"]), int64Val(p["course_offering_id"]), nullInt64(p["room_id"]), int64Val(p["day_of_week"]), strVal(p["start_time"]), strVal(p["end_time"]), strVal(p["effective_from"]), p["effective_until"], strVal(p["status"])); err != nil {
			return err
		}
	}
	for _, e := range d.Events {
		if _, err := tx.ExecContext(ctx, `INSERT INTO teaching_events (id, origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id, event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, published_by_user_id, published_at, revoked_by_user_id, revoked_at, revocation_reason)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, int64Val(e["id"]), nullInt64(e["origin_schedule_pattern_id"]), e["origin_occurrence_date"], nullInt64(e["result_schedule_pattern_id"]), strVal(e["event_kind"]), strVal(e["starts_at"]), strVal(e["ends_at"]), nullInt64(e["room_id"]), e["reason"], strVal(e["lifecycle_status"]), nullInt64(e["published_by_user_id"]), e["published_at"], nullInt64(e["revoked_by_user_id"]), e["revoked_at"], e["revocation_reason"]); err != nil {
			return err
		}
	}
	for _, pt := range d.Parts {
		if _, err := tx.ExecContext(ctx, `INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
			VALUES (?, ?, ?, ?)`, int64Val(pt["teaching_event_id"]), int64Val(pt["course_offering_id"]), strVal(pt["participation_role"]), strVal(pt["participation_status"])); err != nil {
			return err
		}
	}
	for _, c := range d.Confirms {
		if _, err := tx.ExecContext(ctx, `INSERT INTO room_confirmations (teaching_event_id, room_id, confirmation_status, external_contact, note, recorded_by_user_id, recorded_at, confirmed_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, int64Val(c["teaching_event_id"]), int64Val(c["room_id"]), strVal(c["confirmation_status"]), c["external_contact"], c["note"], int64Val(c["recorded_by_user_id"]), strVal(c["recorded_at"]), c["confirmed_at"]); err != nil {
			return err
		}
	}
	for _, t := range d.Tasks {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tasks (id, course_offering_id, title, instructions, deadline_at, task_type, submission_text, submission_url, publication_status, review_state, reviewed_version, created_by_user_id, published_at, completed_at, archived_at, version)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, int64Val(t["id"]), int64Val(t["course_offering_id"]), t["title"], t["instructions"], t["deadline_at"], t["task_type"], t["submission_text"], t["submission_url"], strVal(t["publication_status"]), strVal(t["review_state"]), t["reviewed_version"], int64Val(t["created_by_user_id"]), t["published_at"], t["completed_at"], t["archived_at"], int64Val(t["version"])); err != nil {
			return err
		}
	}
	for _, r := range d.Reviews {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_reviews (task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note)
			VALUES (?, ?, ?, ?, ?, ?)`, int64Val(r["task_id"]), int64Val(r["reviewer_user_id"]), int64Val(r["reviewer_role_assignment_id"]), int64Val(r["task_version"]), strVal(r["decision"]), r["note"]); err != nil {
			return err
		}
	}
	for _, m := range d.Materials {
		if _, err := tx.ExecContext(ctx, `INSERT INTO materials (id, class_id, course_offering_id, task_id, title, material_type, url, description, visibility, status, created_by_user_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, int64Val(m["id"]), rec.ClassID, nullInt64(m["course_offering_id"]), nullInt64(m["task_id"]), strVal(m["title"]), strVal(m["material_type"]), strVal(m["url"]), m["description"], strVal(m["visibility"]), strVal(m["status"]), int64Val(m["created_by_user_id"])); err != nil {
			return err
		}
	}
	_ = scopeArgs
	return nil
}

func rowsMap(ctx context.Context, db *sql.DB, query string, args []any) []map[string]any {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return []map[string]any{}
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		m := map[string]any{}
		for i, c := range cols {
			m[c] = vals[i]
		}
		out = append(out, m)
	}
	return out
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
