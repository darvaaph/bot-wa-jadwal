package schedule

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

type EffectiveSession struct {
	PatternID, EventID, OfferingID                            int64
	Kind, Offering, Title, Code, ActivityType                 string
	StartTime, EndTime, Room, MeetingLink, OriginDate, Reason string
}

// ResolveEffective is the business schedule shared by portal, bot, and summaries.
func ResolveEffective(ctx context.Context, db *sql.DB, classID, semesterID int64, target time.Time) ([]EffectiveSession, error) {
	var zone string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE((SELECT timezone FROM class_settings WHERE class_id=?),'Asia/Jakarta')`, classID).Scan(&zone); err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return nil, err
	}
	target = target.In(loc)
	date := target.Format("2006-01-02")
	dayStart := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)
	day := int(target.Weekday())
	if day == 0 {
		day = 7
	}
	rows, err := db.QueryContext(ctx, `SELECT te.id,te.event_kind,co.id,co.display_name,c.name,c.code,co.activity_type,te.starts_at,te.ends_at,COALESCE(r.code,''),COALESCE(te.meeting_link,''),COALESCE(te.origin_schedule_pattern_id,0),substr(COALESCE(te.origin_occurrence_date,''),1,10),COALESCE(te.reason,'')
	FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id=te.id AND (teo.participation_role='OWNER' OR (teo.participation_role='PARTICIPANT' AND teo.participation_status='ACCEPTED'))
	JOIN course_offerings co ON co.id=teo.course_offering_id JOIN courses c ON c.id=co.course_id JOIN semesters sem ON sem.id=co.semester_id LEFT JOIN rooms r ON r.id=te.room_id
	WHERE sem.class_id=? AND ((?=0 AND sem.status='ACTIVE') OR sem.id=?) AND te.lifecycle_status='PUBLISHED'
	AND (substr(COALESCE(te.origin_occurrence_date,''),1,10)=? OR (julianday(te.starts_at)>=julianday(?) AND julianday(te.starts_at)<julianday(?)))`, classID, semesterID, semesterID, date, dayStart.UTC().Format(time.RFC3339), dayEnd.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	var events []EffectiveSession
	patternsSuppressed := map[int64]bool{}
	offeringsSuppressed := map[int64]bool{}
	for rows.Next() {
		var it EffectiveSession
		var startRaw, endRaw string
		if err := rows.Scan(&it.EventID, &it.Kind, &it.OfferingID, &it.Offering, &it.Title, &it.Code, &it.ActivityType, &startRaw, &endRaw, &it.Room, &it.MeetingLink, &it.PatternID, &it.OriginDate, &it.Reason); err != nil {
			rows.Close()
			return nil, err
		}
		start, err := parseEventTime(startRaw)
		if err != nil {
			rows.Close()
			return nil, err
		}
		end, err := parseEventTime(endRaw)
		if err != nil {
			rows.Close()
			return nil, err
		}
		if it.OriginDate == date && it.PatternID > 0 && (it.Kind == "REPLACEMENT" || it.Kind == "SESSION_CANCELLED" || it.Kind == "EXAM_UTS" || it.Kind == "EXAM_UAS") {
			patternsSuppressed[it.PatternID] = true
		}
		if !start.Before(dayStart) && start.Before(dayEnd) {
			if it.Kind == "HOLIDAY" || (it.Kind == "SESSION_CANCELLED" && it.OriginDate == "") {
				if it.PatternID > 0 {
					patternsSuppressed[it.PatternID] = true
				} else {
					offeringsSuppressed[it.OfferingID] = true
				}
			}
			it.StartTime = start.In(loc).Format("15:04")
			it.EndTime = end.In(loc).Format("15:04")
			events = append(events, it)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = db.QueryContext(ctx, `SELECT sp.id,co.id,co.display_name,c.name,c.code,co.activity_type,sp.start_time,sp.end_time,COALESCE(r.code,''),COALESCE(sp.meeting_link,'')
	FROM schedule_patterns sp JOIN course_offerings co ON co.id=sp.course_offering_id JOIN courses c ON c.id=co.course_id JOIN semesters sem ON sem.id=co.semester_id LEFT JOIN rooms r ON r.id=sp.room_id
	WHERE sem.class_id=? AND ((?=0 AND sem.status='ACTIVE') OR sem.id=?) AND sp.status='ACTIVE' AND sp.day_of_week=? AND (sp.effective_from IS NULL OR substr(sp.effective_from,1,10)<=?) AND (sp.effective_until IS NULL OR substr(sp.effective_until,1,10)>=?)`, classID, semesterID, semesterID, day, date, date)
	if err != nil {
		return nil, err
	}
	items := make([]EffectiveSession, 0)
	for rows.Next() {
		var it EffectiveSession
		if err := rows.Scan(&it.PatternID, &it.OfferingID, &it.Offering, &it.Title, &it.Code, &it.ActivityType, &it.StartTime, &it.EndTime, &it.Room, &it.MeetingLink); err != nil {
			rows.Close()
			return nil, err
		}
		if !patternsSuppressed[it.PatternID] && !offeringsSuppressed[it.OfferingID] {
			it.Kind = "REGULAR"
			items = append(items, it)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	items = append(items, events...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].StartTime < items[j].StartTime })
	return items, nil
}
