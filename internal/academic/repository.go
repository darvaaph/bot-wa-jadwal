package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrUnmappedScope = errors.New("scope belum terpetakan ke kelas terdaftar")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func parseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("timestamp kosong")
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("format timestamp %q tidak didukung", raw)
}

func parseNullTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetClasses mengambil seluruh kelas akademik terdaftar yang terurut berdasarkan kode.
func (r *Repository) GetClasses(ctx context.Context) ([]Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		ORDER BY code ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data kelas: %w", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var createdAt, updatedAt string
		err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Slug,
			&c.StudyProgram,
			&c.CohortYear,
			&c.GroupLabel,
			&c.Status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris kelas: %w", err)
		}
		c.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp created_at kelas ID %d tidak valid: %w", c.ID, err)
		}
		c.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp updated_at kelas ID %d tidak valid: %w", c.ID, err)
		}
		classes = append(classes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris kelas: %w", err)
	}
	return classes, nil
}

// GetCoursesByClassID mengambil seluruh mata kuliah yang ditawarkan untuk kelas tertentu.
func (r *Repository) GetCoursesByClassID(ctx context.Context, classID int64) ([]Course, error) {
	query := `
		SELECT DISTINCT c.id, c.code, c.name, c.status, c.created_at, c.updated_at
		FROM courses c
		JOIN course_offerings co ON co.course_id = c.id
		JOIN semesters s ON s.id = co.semester_id
		WHERE s.class_id = ?
		ORDER BY c.code ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil mata kuliah untuk kelas ID %d: %w", classID, err)
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		var c Course
		var createdAt, updatedAt string
		err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Name,
			&c.Status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris mata kuliah: %w", err)
		}
		c.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp created_at mata kuliah ID %d tidak valid: %w", c.ID, err)
		}
		c.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp updated_at mata kuliah ID %d tidak valid: %w", c.ID, err)
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris mata kuliah: %w", err)
	}
	return courses, nil
}

func (r *Repository) CountClasses(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM classes;").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung jumlah kelas: %w", err)
	}
	return count, nil
}

func (r *Repository) GetClassByID(ctx context.Context, id int64) (*Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		WHERE id = ?;
	`
	var c Class
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Code,
		&c.Slug,
		&c.StudyProgram,
		&c.CohortYear,
		&c.GroupLabel,
		&c.Status,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil kelas ID %d: %w", id, err)
	}
	c.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp created_at kelas ID %d tidak valid: %w", id, err)
	}
	c.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp updated_at kelas ID %d tidak valid: %w", id, err)
	}
	return &c, nil
}

func (r *Repository) GetClassByCode(ctx context.Context, code string) (*Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		WHERE code = ?;
	`
	var c Class
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&c.ID,
		&c.Code,
		&c.Slug,
		&c.StudyProgram,
		&c.CohortYear,
		&c.GroupLabel,
		&c.Status,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil kelas dengan kode '%s': %w", code, err)
	}
	c.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp created_at kelas %q tidak valid: %w", code, err)
	}
	c.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp updated_at kelas %q tidak valid: %w", code, err)
	}
	return &c, nil
}

// EnsureUser memastikan user dengan identityKey terdaftar pada tabel users.
func (r *Repository) EnsureUser(ctx context.Context, identityKey, displayName string) (int64, error) {
	identityKey = strings.TrimSpace(identityKey)
	if identityKey == "" {
		return 0, fmt.Errorf("identityKey tidak boleh kosong")
	}
	if displayName == "" {
		displayName = identityKey
	}

	query := `
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES (?, ?, 'NOPASSWORD', 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET display_name = excluded.display_name, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, identityKey, displayName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("gagal memastikan user %s: %w", identityKey, err)
	}
	return id, nil
}

// EnsureRoom memastikan ruangan terdaftar pada tabel rooms.
func (r *Repository) EnsureRoom(ctx context.Context, code, name string) (int64, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "UMUM"
	}
	if name == "" {
		name = code
	}

	query := `
		INSERT INTO rooms (code, name, status)
		VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, code, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("gagal memastikan room %s: %w", code, err)
	}
	return id, nil
}

// EnsureClass memastikan kelas terdaftar pada tabel classes dan mengembalikannya.
func (r *Repository) EnsureClass(ctx context.Context, rawCode string) (*Class, error) {
	code := strings.TrimSpace(rawCode)
	if code == "" {
		return nil, fmt.Errorf("kode kelas tidak boleh kosong")
	}
	if strings.Contains(code, "@") {
		return nil, fmt.Errorf("kode kelas %q tidak valid: JID WhatsApp tidak boleh digunakan sebagai kode kelas", code)
	}

	cls, err := r.GetClassByCode(ctx, code)
	if err == nil && cls != nil {
		return cls, nil
	}

	var id int64
	var actualCode, slug, studyProgram, groupLabel, status, createdAt, updatedAt string
	var cohortYear int
	err = r.db.QueryRowContext(ctx, `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		WHERE LOWER(code) = LOWER(?) OR LOWER(slug) = LOWER(?)
		LIMIT 1;
	`, code, code).Scan(&id, &actualCode, &slug, &studyProgram, &cohortYear, &groupLabel, &status, &createdAt, &updatedAt)
	if err == nil {
		c := &Class{
			ID:           id,
			Code:         actualCode,
			Slug:         slug,
			StudyProgram: studyProgram,
			CohortYear:   cohortYear,
			GroupLabel:   groupLabel,
			Status:       status,
		}
		c.CreatedAt, _ = parseTime(createdAt)
		c.UpdatedAt, _ = parseTime(updatedAt)
		return c, nil
	}

	slugVal := strings.ToLower(code)
	program := "Teknik Informatika"
	cohort := 2024
	label := code
	parts := strings.Split(code, "-")
	if len(parts) >= 2 {
		label = parts[len(parts)-1]
	}

	query := `
		INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	err = r.db.QueryRowContext(ctx, query, code, slugVal, program, cohort, label).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan class %s: %w", code, err)
	}
	return r.GetClassByID(ctx, id)
}

// EnsureSemesterForDate memastikan kelas memiliki semester aktif yang mencakup tanggal tertentu.
func (r *Repository) EnsureSemesterForDate(ctx context.Context, classID int64, eventDate time.Time) (int64, error) {
	dateStr := eventDate.Format("2006-01-02")
	var semID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM semesters
		WHERE class_id = ? AND ? >= starts_on AND ? <= ends_on
		LIMIT 1
	`, classID, dateStr, dateStr).Scan(&semID)
	if err == nil {
		return semID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("gagal memeriksa semester untuk tanggal %s: %w", dateStr, err)
	}

	year := eventDate.Year()
	month := eventDate.Month()

	var academicYear, term, startsOn, endsOn string
	if month >= time.September {
		academicYear = fmt.Sprintf("%d/%d", year, year+1)
		term = "GANJIL"
		startsOn = fmt.Sprintf("%d-09-01", year)
		nextFebEnd := time.Date(year+1, time.March, 1, 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
		endsOn = nextFebEnd.Format("2006-01-02")
	} else if month <= time.February {
		academicYear = fmt.Sprintf("%d/%d", year-1, year)
		term = "GANJIL"
		startsOn = fmt.Sprintf("%d-09-01", year-1)
		febEnd := time.Date(year, time.March, 1, 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
		endsOn = febEnd.Format("2006-01-02")
	} else {
		academicYear = fmt.Sprintf("%d/%d", year-1, year)
		term = "GENAP"
		startsOn = fmt.Sprintf("%d-03-01", year)
		endsOn = fmt.Sprintf("%d-08-31", year)
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)
	query := `
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(class_id, academic_year, term) DO UPDATE SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	err = r.db.QueryRowContext(ctx, query, classID, academicYear, term, startsOn, endsOn, nowStr, nowStr).Scan(&semID)
	if err != nil {
		return 0, fmt.Errorf("gagal membuat semester %s %s: %w", academicYear, term, err)
	}
	return semID, nil
}

// EnsureSemester memastikan kelas memiliki semester aktif.
func (r *Repository) EnsureSemester(ctx context.Context, classID int64) (int64, error) {
	return r.EnsureSemesterForDate(ctx, classID, time.Now())
}

// EnsureCourseOfferingForDate memastikan mata kuliah dan offering-nya terdaftar untuk kelas dan tanggal tertentu.
func (r *Repository) EnsureCourseOfferingForDate(ctx context.Context, classID int64, courseCode, courseName, activityType string, refDate time.Time) (int64, error) {
	courseCode = strings.TrimSpace(courseCode)
	if courseCode == "" {
		courseCode = "MK-UMUM"
	}
	courseName = strings.TrimSpace(courseName)
	if courseName == "" {
		courseName = courseCode
	}
	activityType = strings.ToUpper(strings.TrimSpace(activityType))
	if activityType == "" {
		activityType = "TEORI"
	}

	var courseID int64
	queryCourse := `
		INSERT INTO courses (code, name, status)
		VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	err := r.db.QueryRowContext(ctx, queryCourse, courseCode, courseName).Scan(&courseID)
	if err != nil {
		return 0, fmt.Errorf("gagal memastikan course %s: %w", courseCode, err)
	}

	semID, err := r.EnsureSemesterForDate(ctx, classID, refDate)
	if err != nil {
		return 0, err
	}

	var offeringID int64
	queryOffering := `
		INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
		VALUES (?, ?, ?, ?, 'ACTIVE')
		ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET display_name = excluded.display_name, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	err = r.db.QueryRowContext(ctx, queryOffering, semID, courseID, activityType, courseName).Scan(&offeringID)
	if err != nil {
		return 0, fmt.Errorf("gagal memastikan course_offering: %w", err)
	}
	return offeringID, nil
}

// EnsureCourseOffering memastikan mata kuliah dan offering-nya terdaftar untuk kelas tertentu.
func (r *Repository) EnsureCourseOffering(ctx context.Context, classID int64, courseCode, courseName, activityType string) (int64, error) {
	return r.EnsureCourseOfferingForDate(ctx, classID, courseCode, courseName, activityType, time.Now())
}

// EnsureSchedulePattern memastikan schedule_pattern tersedia untuk offering tertentu.
func (r *Repository) EnsureSchedulePattern(ctx context.Context, courseOfferingID int64, dayOfWeek int, startTime, endTime string, roomID *int64) (int64, error) {
	if dayOfWeek < 1 || dayOfWeek > 7 {
		dayOfWeek = 1
	}
	startTime = strings.TrimSpace(startTime)
	if len(startTime) != 5 {
		startTime = "07:00"
	}
	endTime = strings.TrimSpace(endTime)
	if len(endTime) != 5 {
		endTime = "08:40"
	}
	if startTime >= endTime {
		endTime = "23:59"
	}

	var patternID int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id FROM schedule_patterns
		WHERE course_offering_id = ? AND day_of_week = ? AND start_time = ? AND status = 'ACTIVE'
		LIMIT 1
	`, courseOfferingID, dayOfWeek, startTime).Scan(&patternID)
	if err == nil {
		return patternID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("gagal memeriksa schedule_patterns: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
		INSERT INTO schedule_patterns (course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
		VALUES (?, ?, ?, ?, ?, '2024-01-01', 'ACTIVE')
		RETURNING id;
	`, courseOfferingID, roomID, dayOfWeek, startTime, endTime).Scan(&patternID)
	if err != nil {
		return 0, fmt.Errorf("gagal menyisipkan schedule_patterns: %w", err)
	}
	return patternID, nil
}

// ResolveClassIDFromScope memetakan scope JID WhatsApp ke ID kelas numerik dan kode kelasnya.
func (r *Repository) ResolveClassIDFromScope(ctx context.Context, scopeJID string) (int64, string, error) {
	scopeJID = strings.TrimSpace(scopeJID)
	if scopeJID == "" {
		return 0, "", fmt.Errorf("scopeJID tidak boleh kosong")
	}

	// 1. Jika grup WhatsApp (@g.us), cari di whatsapp_channels
	if strings.HasSuffix(scopeJID, "@g.us") {
		var classID int64
		var classCode string
		query := `
			SELECT c.id, c.code
			FROM whatsapp_channels wc
			JOIN classes c ON c.id = wc.class_id
			WHERE wc.jid = ? AND wc.status = 'ACTIVE'
			LIMIT 1;
		`
		err := r.db.QueryRowContext(ctx, query, scopeJID).Scan(&classID, &classCode)
		if err == nil {
			return classID, classCode, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, "", fmt.Errorf("gagal memeriksa whatsapp_channels untuk %s: %w", scopeJID, err)
		}
	}

	// 2. Cari di chat_class_contexts (baik personal maupun grup jika ada konteks)
	var classID int64
	var classCode string
	query := `
		SELECT c.id, c.code
		FROM chat_class_contexts ccc
		JOIN classes c ON c.id = ccc.class_id
		WHERE ccc.chat_jid = ?
		LIMIT 1;
	`
	err := r.db.QueryRowContext(ctx, query, scopeJID).Scan(&classID, &classCode)
	if err == nil {
		return classID, classCode, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, "", fmt.Errorf("gagal memeriksa chat_class_contexts untuk %s: %w", scopeJID, err)
	}

	// 3. Fallback: jika scopeJID adalah kode atau slug kelas langsung
	cls, err := r.GetClassByCode(ctx, scopeJID)
	if err == nil && cls != nil {
		return cls.ID, cls.Code, nil
	}

	query = `SELECT id, code FROM classes WHERE LOWER(code) = LOWER(?) OR LOWER(slug) = LOWER(?) LIMIT 1;`
	err = r.db.QueryRowContext(ctx, query, scopeJID, scopeJID).Scan(&classID, &classCode)
	if err == nil {
		return classID, classCode, nil
	}

	return 0, "", ErrUnmappedScope
}

// EnsureLecturer memastikan dosen terdaftar pada tabel lecturers.
func (r *Repository) EnsureLecturer(ctx context.Context, code, fullName string) (int64, error) {
	code = strings.TrimSpace(code)
	fullName = strings.TrimSpace(fullName)
	if code == "" && fullName == "" {
		code = "-"
		fullName = "-"
	} else if code == "" {
		code = fullName
	} else if fullName == "" {
		fullName = code
	}

	query := `
		INSERT INTO lecturers (code, full_name, status)
		VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	var id int64
	err := r.db.QueryRowContext(ctx, query, code, fullName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("gagal memastikan lecturer %s: %w", code, err)
	}
	return id, nil
}

// EnsureOfferingLecturer menghubungkan dosen ke course_offering tertentu.
func (r *Repository) EnsureOfferingLecturer(ctx context.Context, offeringID, lecturerID int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
		VALUES (?, ?, 'PRIMARY')
		ON CONFLICT(course_offering_id, lecturer_id) DO NOTHING
	`, offeringID, lecturerID)
	if err != nil {
		return fmt.Errorf("gagal menghubungkan offering %d dengan lecturer %d: %w", offeringID, lecturerID, err)
	}
	return nil
}

// GetClassTimezone mengambil zona waktu kelas dari class_settings, default Asia/Jakarta
func (r *Repository) GetClassTimezone(ctx context.Context, classID int64) (*time.Location, error) {
	var tz string
	err := r.db.QueryRowContext(ctx, "SELECT timezone FROM class_settings WHERE class_id = ?", classID).Scan(&tz)
	if errors.Is(err, sql.ErrNoRows) || tz == "" {
		tz = "Asia/Jakarta"
	} else if err != nil {
		return nil, fmt.Errorf("gagal membaca class_settings timezone: %w", err)
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		// Fallback fixed zone WIB (UTC+7) jika tz database tidak terpasang di OS
		return time.FixedZone("WIB", 7*3600), nil
	}
	return loc, nil
}

// CreateImportBatch membuat catatan batch impor baru atau mengembalikan ID batch yang ada jika checksum sama.
func (r *Repository) CreateImportBatch(ctx context.Context, classID, semesterID int64, sourceType, checksum string, userID int64) (int64, error) {
	query := `
		INSERT INTO import_batches (class_id, semester_id, source_type, source_checksum, status, created_by_user_id)
		VALUES (?, ?, ?, ?, 'APPLIED', ?)
		ON CONFLICT(class_id, semester_id, source_checksum) DO UPDATE SET updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id;
	`
	var batchID int64
	err := r.db.QueryRowContext(ctx, query, classID, semesterID, sourceType, checksum, userID).Scan(&batchID)
	return batchID, err
}

// RecordImportError mencatat detail kegagalan baris ke tabel import_errors.
func (r *Repository) RecordImportError(ctx context.Context, batchID int64, sourceLoc, fieldName, errCode, msg string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO import_errors (batch_id, source_location, field_name, error_code, message, severity)
		VALUES (?, ?, ?, ?, ?, 'ERROR');
	`, batchID, sourceLoc, fieldName, errCode, msg)
	return err
}

// UpdateImportBatchSummary memperbarui summary_json pada batch impor.
func (r *Repository) UpdateImportBatchSummary(ctx context.Context, batchID int64, summaryJSON string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE import_batches
		SET summary_json = ?, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		WHERE id = ?;
	`, summaryJSON, batchID)
	return err
}

// UpdateImportBatchStats memperbarui summary_json pada batch impor dengan hitungan total, migrated, dan errors.
func (r *Repository) UpdateImportBatchStats(ctx context.Context, batchID int64, total, migrated, errors int) error {
	summary := fmt.Sprintf(`{"total":%d,"migrated":%d,"errors":%d}`, total, migrated, errors)
	return r.UpdateImportBatchSummary(ctx, batchID, summary)
}

// EnsureMigrationBatch memastikan import_batch tersedia untuk pencatatan migrasi atau backfill.
func (r *Repository) EnsureMigrationBatch(ctx context.Context, sourceType string) (int64, error) {
	var classID int64
	err := r.db.QueryRowContext(ctx, "SELECT id FROM classes LIMIT 1;").Scan(&classID)
	if err != nil {
		cls, errCls := r.EnsureClass(ctx, "GENERAL")
		if errCls != nil {
			return 0, errCls
		}
		classID = cls.ID
	}
	semID, err := r.EnsureSemester(ctx, classID)
	if err != nil {
		return 0, err
	}

	adminUserID, _ := r.EnsureUser(ctx, "system", "System Migration")
	checksum := fmt.Sprintf("migration-%s-%s", sourceType, time.Now().Format("2006-01-02"))
	return r.CreateImportBatch(ctx, classID, semID, sourceType, checksum, adminUserID)
}
