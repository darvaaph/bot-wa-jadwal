package semester

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("data tidak ditemukan")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrConflict     = errors.New("data bertabrakan")
	ErrInvalidState = errors.New("status tidak memungkinkan operasi ini")
	ErrForbidden    = errors.New("tindakan tidak tersedia pada cakupan aktif")
	ErrVersion      = errors.New("versi data sudah berubah, muat ulang sebelum menyimpan")
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Actor carries audit identity for semester operations.
type Actor struct {
	UserID           int64
	RoleAssignmentID int64
	CorrelationID    string
}

func auditIdentity(actor Actor) (any, any, string, string) {
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = nowStr()
	}
	var actorUser, actorAssignment any
	actorType := "USER"
	if actor.UserID > 0 {
		actorUser = actor.UserID
	} else {
		actorType = "SYSTEM"
	}
	if actor.RoleAssignmentID > 0 {
		actorAssignment = actor.RoleAssignmentID
	}
	return actorUser, actorAssignment, actorType, corr
}

func insertSemesterAudit(ctx context.Context, tx *sql.Tx, actor Actor, classID int64, semesterID *int64, action, entityType string, entityID int64, after, reason *string) error {
	actorUser, actorAssignment, actorType, corr := auditIdentity(actor)
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, reason, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, semesterID, actorUser, actorAssignment, actorType, action, entityType, entityID, after, reason, corr)
	return err
}

func nowStr() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// CreateClass creates a permanent class + default settings. Admin only (enforced in handler).
func (s *Service) CreateClass(ctx context.Context, actor Actor, code, slug, studyProgram string, cohortYear int, groupLabel, timezone string) (int64, error) {
	code = strings.TrimSpace(code)
	slug = strings.ToLower(strings.TrimSpace(slug))
	studyProgram = strings.TrimSpace(studyProgram)
	groupLabel = strings.TrimSpace(groupLabel)
	if code == "" || slug == "" || studyProgram == "" || groupLabel == "" {
		return 0, ErrInvalidInput
	}
	if cohortYear < 1900 || cohortYear > 9999 {
		return 0, ErrInvalidInput
	}
	if strings.Contains(code, "@") {
		return 0, ErrInvalidInput
	}
	tz := strings.TrimSpace(timezone)
	if tz == "" {
		tz = "Asia/Jakarta"
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return 0, ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var classID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE') RETURNING id`, code, slug, studyProgram, cohortYear, groupLabel).Scan(&classID)
	if err != nil {
		return 0, ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode, replacement_reminder_minutes)
		VALUES (?, ?, 'LINK', 60)`, classID, tz); err != nil {
		return 0, err
	}
	after := `{"code":"` + code + `"}`
	if err := insertSemesterAudit(ctx, tx, actor, classID, nil, "CREATE", "CLASS", classID, &after, nil); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return classID, nil
}

type DraftInput struct {
	AcademicYear     string
	Term             string
	StartsOn         string
	EndsOn           string
	SourceSemesterID *int64
}

// CreateDraft creates a DRAFT semester, optionally copying structure from a source semester.
func (s *Service) CreateDraft(ctx context.Context, actor Actor, classID int64, in DraftInput) (int64, error) {
	year := strings.TrimSpace(in.AcademicYear)
	term := strings.ToUpper(strings.TrimSpace(in.Term))
	starts := strings.TrimSpace(in.StartsOn)
	ends := strings.TrimSpace(in.EndsOn)
	if year == "" || term == "" || starts == "" || ends == "" {
		return 0, ErrInvalidInput
	}
	if starts >= ends {
		return 0, ErrInvalidInput
	}
	if _, err := time.Parse("2006-01-02", starts); err != nil {
		return 0, ErrInvalidInput
	}
	if _, err := time.Parse("2006-01-02", ends); err != nil {
		return 0, ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var newID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status)
		VALUES (?, ?, ?, ?, ?, 'DRAFT') RETURNING id`, classID, year, term, starts, ends).Scan(&newID)
	if err != nil {
		return 0, ErrConflict
	}
	if in.SourceSemesterID != nil {
		if err := copySemesterStructure(ctx, tx, classID, *in.SourceSemesterID, newID); err != nil {
			return 0, err
		}
	}
	after := `{"academic_year":"` + year + `","term":"` + term + `"}`
	if err := insertSemesterAudit(ctx, tx, actor, classID, &newID, "CREATE", "SEMESTER", newID, &after, nil); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newID, nil
}

// copySemesterStructure copies offerings + lecturers + active patterns without tasks/publications/PJ assignments.
func copySemesterStructure(ctx context.Context, tx *sql.Tx, classID, sourceSemID, targetSemID int64) error {
	var sourceClass int64
	if err := tx.QueryRowContext(ctx, `SELECT class_id FROM semesters WHERE id = ?`, sourceSemID).Scan(&sourceClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if sourceClass != classID {
		return ErrInvalidInput
	}
	rows, err := tx.QueryContext(ctx, `SELECT course_id, activity_type, display_name, status FROM course_offerings WHERE semester_id = ?`, sourceSemID)
	if err != nil {
		return err
	}
	type offering struct {
		courseID, activity, display, status string
		oldID                               int64
	}
	var offerings []offering
	// Need old IDs for pattern copy; fetch separately.
	orows, err := tx.QueryContext(ctx, `SELECT id, course_id, activity_type, display_name, status FROM course_offerings WHERE semester_id = ?`, sourceSemID)
	if err != nil {
		rows.Close()
		return err
	}
	for orows.Next() {
		var o offering
		var cid int64
		if err := orows.Scan(&o.oldID, &cid, &o.activity, &o.display, &o.status); err != nil {
			orows.Close()
			rows.Close()
			return err
		}
		o.courseID = fmt.Sprint(cid)
		offerings = append(offerings, o)
	}
	orows.Close()
	rows.Close()
	_ = rows
	for _, o := range offerings {
		var cid int64
		fmt.Sscanf(o.courseID, "%d", &cid)
		var newOfferingID int64
		err := tx.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, 'ACTIVE') RETURNING id`, targetSemID, cid, o.activity, o.display).Scan(&newOfferingID)
		if err != nil {
			return err
		}
		lrows, err := tx.QueryContext(ctx, `SELECT lecturer_id, responsibility FROM offering_lecturers WHERE course_offering_id = ?`, o.oldID)
		if err != nil {
			return err
		}
		for lrows.Next() {
			var lid int64
			var resp string
			if err := lrows.Scan(&lid, &resp); err != nil {
				lrows.Close()
				return err
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
				VALUES (?, ?, ?) ON CONFLICT(course_offering_id, lecturer_id) DO NOTHING`, newOfferingID, lid, resp); err != nil {
				lrows.Close()
				return err
			}
		}
		lrows.Close()
		prows, err := tx.QueryContext(ctx, `SELECT room_id, day_of_week, start_time, end_time, effective_from FROM schedule_patterns
			WHERE course_offering_id = ? AND status = 'ACTIVE'`, o.oldID)
		if err != nil {
			return err
		}
		for prows.Next() {
			var roomID sql.NullInt64
			var dow int
			var start, end, effFrom string
			if err := prows.Scan(&roomID, &dow, &start, &end, &effFrom); err != nil {
				prows.Close()
				return err
			}
			var roomArg any
			if roomID.Valid {
				roomArg = roomID.Int64
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_patterns
				(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
				VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE')`, newOfferingID, roomArg, dow, start, end, effFrom); err != nil {
				prows.Close()
				return err
			}
		}
		prows.Close()
	}
	return nil
}

// Activate performs DRAFT->ACTIVE + archive old ACTIVE in one transaction.
func (s *Service) Activate(ctx context.Context, actor Actor, classID, semesterID int64, expectedVersion int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	var semClass int64
	var published sql.NullString
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT status, class_id, published_at, version FROM semesters WHERE id = ?`, semesterID).
		Scan(&status, &semClass, &published, &version); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if semClass != classID {
		return ErrInvalidInput
	}
	if status != "DRAFT" {
		return ErrInvalidState
	}
	if expectedVersion > 0 && version != expectedVersion {
		return ErrVersion
	}
	now := nowStr()
	if _, err := tx.ExecContext(ctx, `UPDATE semesters SET status='ARCHIVED', archived_at=?, updated_at=?
		WHERE class_id = ? AND status = 'ACTIVE'`, now, now, classID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE semesters SET status='ACTIVE',
		published_at=COALESCE(published_at, ?), activated_at=?, updated_at=? WHERE id = ?`,
		now, now, now, semesterID); err != nil {
		return err
	}
	after := `{"status":"ACTIVE"}`
	if err := insertSemesterAudit(ctx, tx, actor, classID, &semesterID, "ACTIVATE", "SEMESTER", semesterID, &after, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// SemesterListItem is a lightweight semester row.
type SemesterListItem struct {
	ID           int64   `json:"id"`
	ClassID      int64   `json:"class_id"`
	AcademicYear string  `json:"academic_year"`
	Term         string  `json:"term"`
	StartsOn     string  `json:"starts_on"`
	EndsOn       string  `json:"ends_on"`
	Status       string  `json:"status"`
	PublishedAt  *string `json:"published_at,omitempty"`
	ActivatedAt  *string `json:"activated_at,omitempty"`
	ArchivedAt   *string `json:"archived_at,omitempty"`
	Version      int     `json:"version"`
	Offerings    int     `json:"offerings"`
	Patterns     int     `json:"patterns"`
}

func (s *Service) ListSemesters(ctx context.Context, classID int64) ([]SemesterListItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT s.id, s.class_id, s.academic_year, s.term, s.starts_on, s.ends_on,
		s.status, s.published_at, s.activated_at, s.archived_at, s.version,
		(SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = s.id),
		(SELECT COUNT(*) FROM course_offerings co JOIN schedule_patterns sp ON sp.course_offering_id = co.id WHERE co.semester_id = s.id AND sp.status='ACTIVE')
		FROM semesters s WHERE s.class_id = ? ORDER BY s.starts_on DESC, s.id DESC`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SemesterListItem{}
	for rows.Next() {
		var it SemesterListItem
		var pub, act, arch sql.NullString
		if err := rows.Scan(&it.ID, &it.ClassID, &it.AcademicYear, &it.Term, &it.StartsOn, &it.EndsOn,
			&it.Status, &pub, &act, &arch, &it.Version, &it.Offerings, &it.Patterns); err != nil {
			return nil, err
		}
		if pub.Valid {
			it.PublishedAt = &pub.String
		}
		if act.Valid {
			it.ActivatedAt = &act.String
		}
		if arch.Valid {
			it.ArchivedAt = &arch.String
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Preview aggregates counts, conflicts, and completeness for activation review.
type Preview struct {
	Semester    SemesterListItem `json:"semester"`
	Offerings   int              `json:"offerings"`
	Patterns    int              `json:"patterns"`
	Lecturers   int              `json:"lecturers"`
	RoomsUsed   int              `json:"rooms_used"`
	Conflicts   []string         `json:"conflicts"`
	Warnings    []string         `json:"warnings"`
	CanActivate bool             `json:"can_activate"`
	Blockers    []string         `json:"blockers"`
}

func (s *Service) Preview(ctx context.Context, classID, semesterID int64) (*Preview, error) {
	var it SemesterListItem
	var pub, act, arch sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT s.id, s.class_id, s.academic_year, s.term, s.starts_on, s.ends_on,
		s.status, s.published_at, s.activated_at, s.archived_at, s.version,
		(SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = s.id),
		(SELECT COUNT(*) FROM course_offerings co JOIN schedule_patterns sp ON sp.course_offering_id = co.id WHERE co.semester_id = s.id AND sp.status='ACTIVE')
		FROM semesters s WHERE s.id = ? AND s.class_id = ?`, semesterID, classID).
		Scan(&it.ID, &it.ClassID, &it.AcademicYear, &it.Term, &it.StartsOn, &it.EndsOn,
			&it.Status, &pub, &act, &arch, &it.Version, &it.Offerings, &it.Patterns)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if pub.Valid {
		it.PublishedAt = &pub.String
	}
	if act.Valid {
		it.ActivatedAt = &act.String
	}
	if arch.Valid {
		it.ArchivedAt = &arch.String
	}
	p := &Preview{Semester: it, Offerings: it.Offerings, Patterns: it.Patterns, Conflicts: []string{}, Warnings: []string{}, Blockers: []string{}}
	var lecturers, rooms int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT ol.lecturer_id) FROM offering_lecturers ol
		JOIN course_offerings co ON co.id = ol.course_offering_id WHERE co.semester_id = ?`, semesterID).Scan(&lecturers)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT sp.room_id) FROM schedule_patterns sp
		JOIN course_offerings co ON co.id = sp.course_offering_id
		WHERE co.semester_id = ? AND sp.status='ACTIVE' AND sp.room_id IS NOT NULL`, semesterID).Scan(&rooms)
	p.Lecturers = lecturers
	p.RoomsUsed = rooms

	if it.Offerings == 0 {
		p.Blockers = append(p.Blockers, "semester belum memiliki mata kuliah")
	}
	if it.Patterns == 0 {
		p.Warnings = append(p.Warnings, "semester belum memiliki jadwal reguler")
	}
	// Overlap conflicts: same day overlapping times within semester.
	confRows, err := s.db.QueryContext(ctx, `SELECT a.day_of_week, a.start_time, a.end_time, b.start_time, b.end_time,
		c1.display_name, c2.display_name FROM schedule_patterns a
		JOIN course_offerings c1 ON c1.id = a.course_offering_id
		JOIN schedule_patterns b ON b.course_offering_id != a.course_offering_id
		JOIN course_offerings c2 ON c2.id = b.course_offering_id
		JOIN semesters s1 ON s1.id = c1.semester_id
		JOIN semesters s2 ON s2.id = c2.semester_id
		WHERE s1.id = ? AND s2.id = ? AND a.status='ACTIVE' AND b.status='ACTIVE'
		AND a.day_of_week = b.day_of_week AND a.id < b.id
		AND a.start_time < b.end_time AND b.start_time < a.end_time
		LIMIT 20`, semesterID, semesterID)
	if err == nil {
		defer confRows.Close()
		for confRows.Next() {
			var dow int
			var as, ae, bs, be, d1, d2 string
			if err := confRows.Scan(&dow, &as, &ae, &bs, &be, &d1, &d2); err == nil {
				p.Conflicts = append(p.Conflicts, fmt.Sprintf("hari %d: %s (%s-%s) bertabrakan dengan %s (%s-%s)", dow, d1, as, ae, d2, bs, be))
			}
		}
	}
	// Room conflicts.
	roomRows, err := s.db.QueryContext(ctx, `SELECT a.day_of_week, r.code, a.start_time, b.start_time
		FROM schedule_patterns a JOIN schedule_patterns b ON b.room_id = a.room_id
		JOIN rooms r ON r.id = a.room_id
		JOIN course_offerings c1 ON c1.id = a.course_offering_id
		JOIN course_offerings c2 ON c2.id = b.course_offering_id
		WHERE c1.semester_id = ? AND c2.semester_id = ? AND a.status='ACTIVE' AND b.status='ACTIVE'
		AND a.id < b.id AND a.day_of_week = b.day_of_week
		AND a.start_time < b.end_time AND b.start_time < a.end_time
		LIMIT 20`, semesterID, semesterID)
	if err == nil {
		defer roomRows.Close()
		for roomRows.Next() {
			var dow int
			var code, as, bs string
			if err := roomRows.Scan(&dow, &code, &as, &bs); err == nil {
				p.Conflicts = append(p.Conflicts, fmt.Sprintf("ruangan %s hari %d bentrok (%s vs %s)", code, dow, as, bs))
			}
		}
	}
	// Offerings without lecturer.
	var noLect int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = ?
		AND NOT EXISTS (SELECT 1 FROM offering_lecturers ol WHERE ol.course_offering_id = co.id)`, semesterID).Scan(&noLect)
	if noLect > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d mata kuliah belum memiliki dosen", noLect))
	}
	p.CanActivate = len(p.Blockers) == 0 && it.Status == "DRAFT"
	if it.Status != "DRAFT" {
		p.Blockers = append(p.Blockers, "hanya semester DRAFT yang dapat diaktifkan")
	}
	if p.Conflicts == nil {
		p.Conflicts = []string{}
	}
	if p.Warnings == nil {
		p.Warnings = []string{}
	}
	if p.Blockers == nil {
		p.Blockers = []string{}
	}
	return p, nil
}

// AddOffering creates a course offering manually in a DRAFT semester.
func (s *Service) AddOffering(ctx context.Context, actor Actor, classID, semesterID int64, courseCode, courseName, activityType, displayName string) (int64, error) {
	courseCode = strings.TrimSpace(courseCode)
	courseName = strings.TrimSpace(courseName)
	if courseName == "" {
		courseName = courseCode
	}
	if courseCode == "" {
		return 0, ErrInvalidInput
	}
	activityType = strings.ToUpper(strings.TrimSpace(activityType))
	if activityType == "" {
		activityType = "KULIAH"
	}
	if displayName == "" {
		displayName = courseName
	}
	var status string
	var semClass int64
	if err := s.db.QueryRowContext(ctx, `SELECT status, class_id FROM semesters WHERE id = ?`, semesterID).Scan(&status, &semClass); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if semClass != classID || status == "ARCHIVED" {
		return 0, ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var courseID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') RETURNING id`,
		courseCode, courseName).Scan(&courseID)
	if err != nil {
		return 0, err
	}
	var offeringID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
		VALUES (?, ?, ?, ?, 'ACTIVE') RETURNING id`, semesterID, courseID, activityType, displayName).Scan(&offeringID)
	if err != nil {
		return 0, ErrConflict
	}
	after := `{"display_name":"` + strings.ReplaceAll(displayName, `"`, ``) + `"}`
	if err := insertSemesterAudit(ctx, tx, actor, classID, &semesterID, "CREATE", "COURSE_OFFERING", offeringID, &after, nil); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return offeringID, nil
}

// AddPattern creates a regular schedule pattern (basic validation; full conflict engine in BE-D).
func (s *Service) AddPattern(ctx context.Context, actor Actor, offeringID int64, roomID *int64, dayOfWeek int, startTime, endTime, effectiveFrom string) (int64, error) {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return 0, ErrInvalidInput
	}
	startTime = strings.TrimSpace(startTime)
	endTime = strings.TrimSpace(endTime)
	if len(startTime) != 5 || len(endTime) != 5 || startTime >= endTime {
		return 0, ErrInvalidInput
	}
	if effectiveFrom == "" {
		effectiveFrom = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", effectiveFrom); err != nil {
		return 0, ErrInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO schedule_patterns
		(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE') RETURNING id`,
		offeringID, roomID, dayOfWeek, startTime, endTime, effectiveFrom).Scan(&id)
	if err != nil {
		return 0, err
	}
	var classID, semesterID int64
	if err := tx.QueryRowContext(ctx, `SELECT sem.class_id, co.semester_id FROM course_offerings co
		JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?`, offeringID).Scan(&classID, &semesterID); err != nil {
		return 0, err
	}
	after := fmt.Sprintf(`{"day_of_week":%d,"start_time":%q}`, dayOfWeek, startTime)
	if err := insertSemesterAudit(ctx, tx, actor, classID, &semesterID, "CREATE", "SCHEDULE_PATTERN", id, &after, nil); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// ImportJSON validates and applies a legacy jadwal JSON into a DRAFT semester.
func (s *Service) ImportJSON(ctx context.Context, actor Actor, classID int64, semesterID *int64, academicYear, term, startsOn, endsOn string, raw json.RawMessage, userID int64) (int64, []ImportError, error) {
	type rawItem struct {
		Hari         string `json:"hari"`
		Jam          string `json:"jam"`
		KodeMatkul   string `json:"kode_matkul"`
		NamaMatkul   string `json:"nama_matkul"`
		InisialDosen string `json:"inisial_dosen"`
		Dosen        string `json:"dosen"`
		Ruang        string `json:"ruang"`
	}
	type rawDoc struct {
		Kampus     string            `json:"kampus"`
		Dosen      map[string]string `json:"dosen"`
		MataKuliah map[string]string `json:"mata_kuliah"`
		Jadwal     []rawItem         `json:"jadwal"`
	}
	var doc rawDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, nil, ErrInvalidInput
	}
	errs := []ImportError{}
	addErr := func(loc, field, code, msg, severity string) {
		errs = append(errs, ImportError{Location: loc, Field: field, Code: code, Message: msg, Severity: severity})
	}
	if len(doc.MataKuliah) == 0 {
		addErr("mata_kuliah", "", "EMPTY", "daftar mata_kuliah kosong", "ERROR")
	}
	if len(doc.Jadwal) == 0 {
		addErr("jadwal", "", "EMPTY", "daftar jadwal kosong", "ERROR")
	}
	validDays := map[string]int{"senin": 1, "selasa": 2, "rabu": 3, "kamis": 4, "jumat": 5, "sabtu": 6, "minggu": 7}
	seen := map[string]bool{}
	for i, item := range doc.Jadwal {
		loc := fmt.Sprintf("jadwal[%d]", i)
		if strings.TrimSpace(item.KodeMatkul) == "" {
			addErr(loc, "kode_matkul", "REQUIRED", "kode_matkul wajib diisi", "ERROR")
		}
		if strings.TrimSpace(item.NamaMatkul) == "" {
			addErr(loc, "nama_matkul", "REQUIRED", "nama_matkul wajib diisi", "ERROR")
		}
		dow, ok := validDays[strings.ToLower(strings.TrimSpace(item.Hari))]
		if !ok {
			addErr(loc, "hari", "INVALID", "hari tidak dikenal", "ERROR")
		}
		parts := strings.Split(strings.TrimSpace(item.Jam), "-")
		if len(parts) != 2 {
			addErr(loc, "jam", "INVALID", "format jam harus HH:MM - HH:MM", "ERROR")
		} else {
			start := strings.TrimSpace(parts[0])
			end := strings.TrimSpace(parts[1])
			if len(start) != 5 || len(end) != 5 || start >= end {
				addErr(loc, "jam", "INVALID", "rentang jam tidak valid", "ERROR")
			}
			_ = dow
		}
		key := strings.ToLower(strings.TrimSpace(item.Hari)) + "|" + strings.TrimSpace(item.Jam) + "|" + strings.TrimSpace(item.KodeMatkul)
		if seen[key] {
			addErr(loc, "", "DUPLICATE", "baris duplikat", "ERROR")
		}
		seen[key] = true
		if _, ok := doc.MataKuliah[strings.TrimSpace(item.KodeMatkul)]; !ok && strings.TrimSpace(item.KodeMatkul) != "" {
			addErr(loc, "kode_matkul", "UNKNOWN_COURSE", "kode tidak ada di master mata_kuliah (akan dibuat otomatis)", "WARNING")
		}
	}
	blockers := 0
	for _, e := range errs {
		if e.Severity == "ERROR" {
			blockers++
		}
	}
	sumData, _ := json.Marshal(map[string]any{
		"total_rows": len(doc.Jadwal), "valid_rows": len(doc.Jadwal) - blockers,
		"warning_rows": len(errs) - blockers, "error_rows": blockers, "applied_rows": 0,
	})
	// Record batch even on failure (status INVALID) for auditability.
	batchID, batchErr := s.createImportBatch(ctx, classID, semesterID, academicYear, term, startsOn, endsOn, checksum(raw), string(sumData), userID, blockers > 0)
	if batchErr != nil {
		return 0, errs, batchErr
	}
	for _, e := range errs {
		_ = s.recordImportError(ctx, batchID, e)
	}
	if blockers > 0 {
		return batchID, errs, ErrInvalidInput
	}
	// Apply: create/fetch DRAFT semester then seed courses/offerings/patterns/lecturers/rooms.
	targetSem, err := s.resolveOrCreateDraft(ctx, actor, classID, semesterID, academicYear, term, startsOn, endsOn)
	if err != nil {
		return batchID, errs, err
	}
	applied, err := s.applyImport(ctx, classID, targetSem, doc)
	if err != nil {
		return batchID, errs, err
	}
	sumData, _ = json.Marshal(map[string]any{
		"total_rows": len(doc.Jadwal), "valid_rows": len(doc.Jadwal),
		"warning_rows": len(errs), "error_rows": 0, "applied_rows": applied, "semester_id": targetSem,
	})
	_, _ = s.db.ExecContext(ctx, `UPDATE import_batches SET status='APPLIED', semester_id=?, summary_json=? WHERE id=?`, targetSem, string(sumData), batchID)
	actorUser, actorAssignment, actorType, corr := auditIdentity(actor)
	_, _ = s.db.ExecContext(ctx, `INSERT INTO audit_logs (class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'IMPORT', 'SEMESTER', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, targetSem, actorUser, actorAssignment, actorType, targetSem,
		fmt.Sprintf(`{"applied_rows":%d,"batch_id":%d}`, applied, batchID), corr)
	return targetSem, errs, nil
}

type ImportError struct {
	Location string `json:"location"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

func (s *Service) createImportBatch(ctx context.Context, classID int64, semesterID *int64, year, term, starts, ends, check, summary string, userID int64, invalid bool) (int64, error) {
	semVal := semesterID
	if semVal == nil {
		// Use a placeholder: latest DRAFT or any semester of the class for FK; fallback creates temp draft ref.
		var latest sql.NullInt64
		_ = s.db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ? ORDER BY id DESC LIMIT 1`, classID).Scan(&latest)
		if latest.Valid {
			v := latest.Int64
			semVal = &v
		} else {
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			defer tx.Rollback()
			y, t, st, en := year, term, starts, ends
			if y == "" {
				y = "2026/2027"
			}
			if t == "" {
				t = "GANJIL"
			}
			if st == "" {
				st = "2026-09-01"
			}
			if en == "" {
				en = "2027-01-31"
			}
			var nid int64
			if err := tx.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status)
				VALUES (?, ?, ?, ?, ?, 'DRAFT') RETURNING id`, classID, y, t, st, en).Scan(&nid); err != nil {
				return 0, err
			}
			if err := tx.Commit(); err != nil {
				return 0, err
			}
			semVal = &nid
		}
	}
	status := "READY"
	if invalid {
		status = "INVALID"
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO import_batches
		(class_id, semester_id, source_type, source_checksum, status, created_by_user_id, summary_json)
		VALUES (?, ?, 'JSON', ?, ?, ?, ?) RETURNING id`,
		classID, *semVal, check, status, userID, summary).Scan(&id)
	return id, err
}

func (s *Service) recordImportError(ctx context.Context, batchID int64, e ImportError) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO import_errors (batch_id, source_location, field_name, error_code, message, severity)
		VALUES (?, ?, ?, ?, ?, ?)`, batchID, e.Location, e.Field, e.Code, e.Message, e.Severity)
	return err
}

func (s *Service) resolveOrCreateDraft(ctx context.Context, actor Actor, classID int64, semesterID *int64, year, term, starts, ends string) (int64, error) {
	if semesterID != nil && *semesterID > 0 {
		var status string
		var c int64
		if err := s.db.QueryRowContext(ctx, `SELECT status, class_id FROM semesters WHERE id = ?`, *semesterID).Scan(&status, &c); err != nil {
			return 0, ErrNotFound
		}
		if c != classID || status == "ARCHIVED" {
			return 0, ErrInvalidInput
		}
		return *semesterID, nil
	}
	if year == "" || term == "" || starts == "" || ends == "" || starts >= ends {
		return 0, ErrInvalidInput
	}
	return s.CreateDraft(ctx, actor, classID, DraftInput{AcademicYear: year, Term: strings.ToUpper(term), StartsOn: starts, EndsOn: ends})
}

func (s *Service) applyImport(ctx context.Context, classID, semesterID int64, doc any) (int, error) {
	type rawItem struct {
		Hari         string `json:"hari"`
		Jam          string `json:"jam"`
		KodeMatkul   string `json:"kode_matkul"`
		NamaMatkul   string `json:"nama_matkul"`
		InisialDosen string `json:"inisial_dosen"`
		Dosen        string `json:"dosen"`
		Ruang        string `json:"ruang"`
	}
	type rawDoc struct {
		Kampus     string            `json:"kampus"`
		Dosen      map[string]string `json:"dosen"`
		MataKuliah map[string]string `json:"mata_kuliah"`
		Jadwal     []rawItem         `json:"jadwal"`
	}
	// Re-marshal to normalized struct.
	b, _ := json.Marshal(doc)
	var d rawDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := nowStr()
	courseIDs := map[string]int64{}
	for code, name := range d.MataKuliah {
		var cid int64
		err := tx.QueryRowContext(ctx, `INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id`, code, name, now).Scan(&cid)
		if err != nil {
			return 0, err
		}
		courseIDs[code] = cid
	}
	dayMap := map[string]int{"senin": 1, "selasa": 2, "rabu": 3, "kamis": 4, "jumat": 5, "sabtu": 6, "minggu": 7}
	applied := 0
	for _, item := range d.Jadwal {
		code := strings.TrimSpace(item.KodeMatkul)
		cid, ok := courseIDs[code]
		if !ok {
			name := strings.TrimSpace(item.NamaMatkul)
			for _, suf := range []string{" (Praktikum)", " (Teori)", " (Praktik)"} {
				name = strings.ReplaceAll(name, suf, "")
			}
			if name == "" {
				name = code
			}
			err := tx.QueryRowContext(ctx, `INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id`, code, name, now).Scan(&cid)
			if err != nil {
				return 0, err
			}
			courseIDs[code] = cid
		}
		lower := strings.ToLower(item.NamaMatkul)
		activity := "KULIAH"
		if strings.Contains(lower, "praktikum") || strings.Contains(lower, "praktik") {
			activity = "PRAKTIKUM"
		} else if strings.Contains(lower, "teori") {
			activity = "TEORI"
		}
		var offeringID int64
		err := tx.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, 'ACTIVE')
			ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET display_name=excluded.display_name, updated_at=?
			RETURNING id`, semesterID, cid, activity, strings.TrimSpace(item.NamaMatkul), now).Scan(&offeringID)
		if err != nil {
			return 0, err
		}
		initial := strings.TrimSpace(item.InisialDosen)
		full := strings.TrimSpace(item.Dosen)
		if initial != "" || full != "" {
			if initial == "" {
				initial = full
			}
			if full == "" {
				if alt, ok := d.Dosen[initial]; ok && strings.TrimSpace(alt) != "" {
					full = alt
				} else {
					full = initial
				}
			}
			var lid int64
			err := tx.QueryRowContext(ctx, `INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET full_name=excluded.full_name, updated_at=? RETURNING id`, initial, full, now).Scan(&lid)
			if err != nil {
				return 0, err
			}
			_, _ = tx.ExecContext(ctx, `INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
				VALUES (?, ?, 'PRIMARY') ON CONFLICT(course_offering_id, lecturer_id) DO NOTHING`, offeringID, lid)
		}
		var roomArg any
		if rc := strings.TrimSpace(item.Ruang); rc != "" {
			var rid int64
			err := tx.QueryRowContext(ctx, `INSERT INTO rooms (code, name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id`, rc, rc, now).Scan(&rid)
			if err != nil {
				return 0, err
			}
			roomArg = rid
		}
		dow := dayMap[strings.ToLower(strings.TrimSpace(item.Hari))]
		parts := strings.Split(item.Jam, "-")
		start, end := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		var semStarts string
		_ = tx.QueryRowContext(ctx, `SELECT starts_on FROM semesters WHERE id = ?`, semesterID).Scan(&semStarts)
		if semStarts == "" {
			semStarts = "2026-09-01"
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_patterns
			(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
			VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE')`, offeringID, roomArg, dow, start, end, semStarts); err != nil {
			return 0, err
		}
		applied++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return applied, nil
}
