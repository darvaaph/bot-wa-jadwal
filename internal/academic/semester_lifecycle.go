package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SemesterService mengelola siklus hidup semester, penawaran mata kuliah, dan impor data akademik.
type SemesterService struct {
	db *sql.DB
}

// NewSemesterService membuat instance baru SemesterService.
func NewSemesterService(db *sql.DB) *SemesterService {
	return &SemesterService{db: db}
}

// CreateClass membuat kelas permanen dan default settings.
func (s *SemesterService) CreateClass(ctx context.Context, actor Actor, code, slug, studyProgram string, cohortYear int, groupLabel, timezone string) (int64, error) {
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
	err = tx.QueryRowContext(ctx, `
		INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE') RETURNING id
	`, code, slug, studyProgram, cohortYear, groupLabel).Scan(&classID)
	if err != nil {
		return 0, ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO class_settings (class_id, timezone, portal_access_mode, replacement_reminder_minutes)
		VALUES (?, ?, 'LINK', 60)
	`, classID, tz); err != nil {
		return 0, err
	}
	after := fmt.Sprintf(`{"code":%q}`, code)
	if err := WriteAuditLog(ctx, tx, actor, &classID, nil, "CREATE", "CLASS", &classID, nil, &after, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return classID, nil
}

// DraftInput payload input pembuatan semester DRAFT.
type DraftInput struct {
	AcademicYear     string
	Term             string
	StartsOn         string
	EndsOn           string
	SourceSemesterID *int64
}

// CreateDraft membuat semester dengan status DRAFT, dengan opsi kloning struktur semester sebelumnya.
func (s *SemesterService) CreateDraft(ctx context.Context, actor Actor, classID int64, in DraftInput) (int64, error) {
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
	err = tx.QueryRowContext(ctx, `
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status)
		VALUES (?, ?, ?, ?, ?, 'DRAFT') RETURNING id
	`, classID, year, term, starts, ends).Scan(&newID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return 0, fmt.Errorf("%w: semester %s %s sudah ada di kelas ini (cek daftar semester, termasuk arsip)", ErrConflict, year, term)
		}
		return 0, err
	}
	if in.SourceSemesterID != nil {
		if err := copySemesterStructure(ctx, tx, classID, *in.SourceSemesterID, newID); err != nil {
			return 0, err
		}
	}
	after := fmt.Sprintf(`{"academic_year":%q,"term":%q}`, year, term)
	if err := WriteAuditLog(ctx, tx, actor, &classID, &newID, "CREATE", "SEMESTER", &newID, nil, &after, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newID, nil
}

// copySemesterStructure menyalin offerings, dosen, dan pola jadwal aktif tanpa tugas/review/PJ.
// Pola salinan memakai effective_from = starts_on semester tujuan agar konsisten dengan aturan DRAFT.
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
	var targetStart string
	if err := tx.QueryRowContext(ctx, `SELECT starts_on FROM semesters WHERE id = ?`, targetSemID).Scan(&targetStart); err != nil {
		return err
	}
	if len(targetStart) >= 10 {
		targetStart = targetStart[:10]
	}

	type offering struct {
		courseID, activity, display, status string
		oldID                               int64
	}
	var offerings []offering

	orows, err := tx.QueryContext(ctx, `
		SELECT id, course_id, activity_type, display_name, status
		FROM course_offerings WHERE semester_id = ?
	`, sourceSemID)
	if err != nil {
		return err
	}
	defer orows.Close()

	for orows.Next() {
		var o offering
		var cid int64
		if err := orows.Scan(&o.oldID, &cid, &o.activity, &o.display, &o.status); err != nil {
			return err
		}
		o.courseID = fmt.Sprint(cid)
		offerings = append(offerings, o)
	}

	for _, o := range offerings {
		var cid int64
		fmt.Sscanf(o.courseID, "%d", &cid)
		var newOfferingID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, 'ACTIVE') RETURNING id
		`, targetSemID, cid, o.activity, o.display).Scan(&newOfferingID)
		if err != nil {
			return err
		}

		lrows, err := tx.QueryContext(ctx, `
			SELECT lecturer_id, responsibility FROM offering_lecturers WHERE course_offering_id = ? AND superseded_at IS NULL
		`, o.oldID)
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
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
				VALUES (?, ?, ?) ON CONFLICT(course_offering_id, lecturer_id) DO UPDATE SET responsibility=excluded.responsibility, superseded_at=NULL
			`, newOfferingID, lid, resp); err != nil {
				lrows.Close()
				return err
			}
		}
		lrows.Close()

		prows, err := tx.QueryContext(ctx, `
			SELECT room_id, day_of_week, start_time, end_time FROM schedule_patterns
			WHERE course_offering_id = ? AND status = 'ACTIVE'
		`, o.oldID)
		if err != nil {
			return err
		}
		for prows.Next() {
			var roomID sql.NullInt64
			var dow int
			var start, end string
			if err := prows.Scan(&roomID, &dow, &start, &end); err != nil {
				prows.Close()
				return err
			}
			var roomArg any
			if roomID.Valid {
				roomArg = roomID.Int64
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO schedule_patterns (course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
				VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE')
			`, newOfferingID, roomArg, dow, start, end, targetStart); err != nil {
				prows.Close()
				return err
			}
		}
		prows.Close()
	}
	return nil
}

// Activate mengaktifkan semester (DRAFT -> ACTIVE) dan mengarsipkan semester aktif lama secara atomik.
func (s *SemesterService) Activate(ctx context.Context, actor Actor, classID, semesterID int64, expectedVersion int) error {
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `
		UPDATE semesters SET status='ARCHIVED', archived_at=?, updated_at=?
		WHERE class_id = ? AND status = 'ACTIVE'
	`, now, now, classID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE semesters SET status='ACTIVE',
		published_at=COALESCE(published_at, ?), activated_at=?, version=version+1, updated_at=? WHERE id = ?
	`, now, now, now, semesterID); err != nil {
		return err
	}
	after := `{"status":"ACTIVE"}`
	if err := WriteAuditLog(ctx, tx, actor, &classID, &semesterID, "ACTIVATE", "SEMESTER", &semesterID, nil, &after, ""); err != nil {
		return err
	}
	return tx.Commit()
}

// SemesterListItem merepresentasikan baris semester ringkas.
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

// ListSemesters mengambil daftar seluruh semester untuk sebuah kelas.
func (s *SemesterService) ListSemesters(ctx context.Context, classID int64) ([]SemesterListItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.class_id, s.academic_year, s.term, s.starts_on, s.ends_on,
		s.status, s.published_at, s.activated_at, s.archived_at, s.version,
		(SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = s.id),
		(SELECT COUNT(*) FROM course_offerings co JOIN schedule_patterns sp ON sp.course_offering_id = co.id WHERE co.semester_id = s.id AND sp.status='ACTIVE')
		FROM semesters s WHERE s.class_id = ? ORDER BY s.starts_on DESC, s.id DESC
	`, classID)
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

// Preview agregat data, deteksi konflik, dan status kesiapan untuk peninjauan aktivasi semester.
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

// Preview memeriksa kesiapan aktivasi semester, menghitung potensi konflik jadwal dan ruangan.
func (s *SemesterService) Preview(ctx context.Context, classID, semesterID int64) (*Preview, error) {
	var it SemesterListItem
	var pub, act, arch sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT s.id, s.class_id, s.academic_year, s.term, s.starts_on, s.ends_on,
		s.status, s.published_at, s.activated_at, s.archived_at, s.version,
		(SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = s.id),
		(SELECT COUNT(*) FROM course_offerings co JOIN schedule_patterns sp ON sp.course_offering_id = co.id WHERE co.semester_id = s.id AND sp.status='ACTIVE')
		FROM semesters s WHERE s.id = ? AND s.class_id = ?
	`, semesterID, classID).
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
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT ol.lecturer_id) FROM offering_lecturers ol
		JOIN course_offerings co ON co.id = ol.course_offering_id WHERE co.semester_id = ? AND ol.superseded_at IS NULL
	`, semesterID).Scan(&lecturers)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT sp.room_id) FROM schedule_patterns sp
		JOIN course_offerings co ON co.id = sp.course_offering_id
		WHERE co.semester_id = ? AND sp.status='ACTIVE' AND sp.room_id IS NOT NULL
	`, semesterID).Scan(&rooms)
	p.Lecturers = lecturers
	p.RoomsUsed = rooms

	if it.Offerings == 0 {
		p.Blockers = append(p.Blockers, "semester belum memiliki mata kuliah")
	}
	if it.Patterns == 0 {
		p.Warnings = append(p.Warnings, "semester belum memiliki jadwal reguler")
	}

	// Overlap conflicts
	confRows, err := s.db.QueryContext(ctx, `
		SELECT a.day_of_week, a.start_time, a.end_time, b.start_time, b.end_time,
		c1.display_name, c2.display_name FROM schedule_patterns a
		JOIN course_offerings c1 ON c1.id = a.course_offering_id
		JOIN schedule_patterns b ON b.course_offering_id != a.course_offering_id
		JOIN course_offerings c2 ON c2.id = b.course_offering_id
		JOIN semesters s1 ON s1.id = c1.semester_id
		JOIN semesters s2 ON s2.id = c2.semester_id
		WHERE s1.id = ? AND s2.id = ? AND a.status='ACTIVE' AND b.status='ACTIVE'
		AND a.day_of_week = b.day_of_week AND a.id < b.id
		AND a.start_time < b.end_time AND b.start_time < a.end_time
		LIMIT 20
	`, semesterID, semesterID)
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

	// Room conflicts
	roomRows, err := s.db.QueryContext(ctx, `
		SELECT a.day_of_week, r.code, a.start_time, b.start_time
		FROM schedule_patterns a JOIN schedule_patterns b ON b.room_id = a.room_id
		JOIN rooms r ON r.id = a.room_id
		JOIN course_offerings c1 ON c1.id = a.course_offering_id
		JOIN course_offerings c2 ON c2.id = b.course_offering_id
		WHERE c1.semester_id = ? AND c2.semester_id = ? AND a.status='ACTIVE' AND b.status='ACTIVE'
		AND a.id < b.id AND a.day_of_week = b.day_of_week
		AND a.start_time < b.end_time AND b.start_time < a.end_time
		LIMIT 20
	`, semesterID, semesterID)
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

	// Offerings without lecturer
	var noLect int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM course_offerings co WHERE co.semester_id = ?
		AND NOT EXISTS (SELECT 1 FROM offering_lecturers ol WHERE ol.course_offering_id = co.id AND ol.superseded_at IS NULL)
	`, semesterID).Scan(&noLect)
	if noLect > 0 {
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d mata kuliah belum memiliki dosen", noLect))
	}

	p.CanActivate = len(p.Blockers) == 0 && it.Status == "DRAFT"
	if it.Status != "DRAFT" {
		p.Blockers = append(p.Blockers, "hanya semester DRAFT yang dapat diaktifkan")
	}
	return p, nil
}

// DeleteDraft menghapus semester draf beserta offerings, lecturers, schedule patterns, dan batch import terkait jika belum aktif/arsip.
func (s *SemesterService) DeleteDraft(ctx context.Context, actor Actor, classID, semesterID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var curClassID int64
	var status, year, term string
	err = tx.QueryRowContext(ctx, `
		SELECT class_id, status, academic_year, term FROM semesters WHERE id = ?
	`, semesterID).Scan(&curClassID, &status, &year, &term)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	if curClassID != classID {
		return ErrInvalidInput
	}
	if status != "DRAFT" {
		return ErrInvalidState
	}

	// Hapus pola jadwal terkait offering semester ini
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM schedule_patterns WHERE course_offering_id IN (
			SELECT id FROM course_offerings WHERE semester_id = ?
		)
	`, semesterID); err != nil {
		return err
	}

	// Hapus dosen offering
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM offering_lecturers WHERE course_offering_id IN (
			SELECT id FROM course_offerings WHERE semester_id = ?
		)
	`, semesterID); err != nil {
		return err
	}

	// Hapus offerings
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM course_offerings WHERE semester_id = ?
	`, semesterID); err != nil {
		return err
	}

	// Hapus import errors dan import batches jika ada
	_, _ = tx.ExecContext(ctx, `
		DELETE FROM import_errors WHERE batch_id IN (
			SELECT id FROM curriculum_import_batches WHERE semester_id = ?
		)
	`, semesterID)
	_, _ = tx.ExecContext(ctx, `
		DELETE FROM curriculum_import_batches WHERE semester_id = ?
	`, semesterID)

	// Hapus semester
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM semesters WHERE id = ?
	`, semesterID); err != nil {
		return err
	}

	before := fmt.Sprintf(`{"academic_year":%q,"term":%q,"status":%q}`, year, term, status)
	if err := WriteAuditLog(ctx, tx, actor, &classID, &semesterID, "DELETE", "SEMESTER", &semesterID, &before, nil, "Hapus draf semester"); err != nil {
		return err
	}

	return tx.Commit()
}
