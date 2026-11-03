package v1

import (
	"context"
	"database/sql"
	"time"

	"bot-jadwal/internal/api/common"
)

func queueTaskNotification(ctx context.Context, tx *sql.Tx, id int64, eventType string, userID int64) error {
	var classID int64
	var course, title, instructions, submission, slug, zone string
	var version int
	var reviewID int64
	var deadline common.DBTimestamp
	err := tx.QueryRowContext(ctx, `SELECT sem.class_id,co.display_name,t.title,t.instructions,t.deadline_at,COALESCE(NULLIF(t.submission_url,''),t.submission_text,''),cl.slug,COALESCE(cs.timezone,'Asia/Jakarta'),t.version FROM tasks t JOIN course_offerings co ON co.id=t.course_offering_id JOIN semesters sem ON sem.id=co.semester_id JOIN classes cl ON cl.id=sem.class_id LEFT JOIN class_settings cs ON cs.class_id=cl.id WHERE t.id=?`, id).Scan(&classID, &course, &title, &instructions, &deadline, &submission, &slug, &zone, &version)
	if err != nil {
		return err
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(id),0) FROM task_reviews WHERE task_id=?`, id).Scan(&reviewID); err != nil {
		return err
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return err
	}
	return common.QueueNotification(ctx, tx, classID, eventType, "TASK", id, map[string]any{"course": course, "title": title, "instructions": instructions, "deadline": deadline.Time.In(loc).Format("02 Jan 2006 15:04 MST"), "submission_url": submission, "class_slug": slug, "version": version, "review_id": reviewID}, userID)
}
