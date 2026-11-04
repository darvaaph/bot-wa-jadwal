package v1

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/util"
)

func queueTaskNotification(ctx context.Context, tx *sql.Tx, id int64, eventType string, userID int64) error {
	var classID int64
	var course, title, instructions, subURL, subText, slug, zone, classCode, groupLabel, semYear string
	var cohortYear, version int
	var reviewID int64
	var deadline common.DBTimestamp
	query := `
		SELECT sem.class_id, co.display_name, t.title, COALESCE(t.instructions, ''),
		       t.deadline_at, COALESCE(t.submission_url, ''), COALESCE(t.submission_text, ''),
		       cl.slug, cl.code, cl.cohort_year, COALESCE(cl.group_label, ''),
		       COALESCE(sem.academic_year, ''), COALESCE(cs.timezone, 'Asia/Jakarta'), t.version
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		JOIN classes cl ON cl.id = sem.class_id
		LEFT JOIN class_settings cs ON cs.class_id = cl.id
		WHERE t.id = ?;
	`
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&classID, &course, &title, &instructions,
		&deadline, &subURL, &subText,
		&slug, &classCode, &cohortYear, &groupLabel,
		&semYear, &zone, &version,
	)
	if err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM task_reviews WHERE task_id = ?`, id).Scan(&reviewID); err != nil {
		return err
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	dlLocal := deadline.Time.In(loc)
	hari := util.GetHariIndonesia(dlLocal)
	deadlineFormatted := fmt.Sprintf("%s, %s", hari, dlLocal.Format("02 Jan 2006 15:04 MST"))

	shortName := util.FormatClassShort(classCode, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = classCode
	}

	payload := map[string]any{
		"course":          course,
		"title":           title,
		"instructions":    instructions,
		"deadline":        deadlineFormatted,
		"submission_text": subText,
		"submission_url":  subURL,
		"class_slug":      slug,
		"class_name":      shortName,
		"version":         version,
		"review_id":       reviewID,
	}

	return common.QueueNotification(ctx, tx, classID, eventType, "TASK", id, payload, userID)
}
