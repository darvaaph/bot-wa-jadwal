package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// AddOffering membuat penawaran mata kuliah (course offering) baru secara manual pada semester DRAFT.
func (s *SemesterService) AddOffering(ctx context.Context, actor Actor, classID, semesterID int64, courseCode, courseName, activityType, displayName string) (int64, error) {
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

	now := time.Now().UTC().Format(time.RFC3339Nano)
	var courseID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id
	`, courseCode, courseName, now).Scan(&courseID)
	if err != nil {
		return 0, err
	}
	var offeringID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
		VALUES (?, ?, ?, ?, 'ACTIVE') RETURNING id
	`, semesterID, courseID, activityType, displayName).Scan(&offeringID)
	if err != nil {
		return 0, ErrConflict
	}
	after := fmt.Sprintf(`{"display_name":%q}`, displayName)
	if err := WriteAuditLog(ctx, tx, actor, &classID, &semesterID, "CREATE", "COURSE_OFFERING", &offeringID, nil, &after, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return offeringID, nil
}

// AddPattern membuat pola jadwal reguler mingguan baru untuk sebuah offering.
func (s *SemesterService) AddPattern(ctx context.Context, actor Actor, offeringID int64, roomID *int64, dayOfWeek int, startTime, endTime, effectiveFrom string) (int64, error) {
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
	err = tx.QueryRowContext(ctx, `
		INSERT INTO schedule_patterns
		(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE') RETURNING id
	`, offeringID, roomID, dayOfWeek, startTime, endTime, effectiveFrom).Scan(&id)
	if err != nil {
		return 0, err
	}
	var classID, semesterID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT sem.class_id, co.semester_id FROM course_offerings co
		JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?
	`, offeringID).Scan(&classID, &semesterID); err != nil {
		return 0, err
	}
	after := fmt.Sprintf(`{"day_of_week":%d,"start_time":%q}`, dayOfWeek, startTime)
	if err := WriteAuditLog(ctx, tx, actor, &classID, &semesterID, "CREATE", "SCHEDULE_PATTERN", &id, nil, &after, ""); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}
