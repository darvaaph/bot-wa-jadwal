package schedule

import (
	"context"
	"database/sql"
	"time"
)

type PeriodReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// OfferingPeriodContains uses a half-open event interval and local semester dates.
func OfferingPeriodContains(ctx context.Context, q PeriodReader, offeringID int64, start, end time.Time) (bool, error) {
	var first, last, zone string
	err := q.QueryRowContext(ctx, `SELECT substr(sem.starts_on,1,10),substr(sem.ends_on,1,10),COALESCE(cs.timezone,'Asia/Jakarta') FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id LEFT JOIN class_settings cs ON cs.class_id=sem.class_id WHERE co.id=?`, offeringID).Scan(&first, &last, &zone)
	if err != nil {
		return false, err
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return false, err
	}
	return end.After(start) && start.In(loc).Format("2006-01-02") >= first && end.Add(-time.Nanosecond).In(loc).Format("2006-01-02") <= last, nil
}
