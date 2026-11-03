package schedule

import (
	"context"
	"database/sql"
	"time"

	"bot-jadwal/internal/audit"
)

// SyncDraftParticipants preserves responses unless an explicit participant list changes them.
func SyncDraftParticipants(ctx context.Context, tx *sql.Tx, actor EventActor, eventID, ownerOfferingID int64, ids []int64, start, end time.Time) error {
	var ownerClassID int64
	if err := tx.QueryRowContext(ctx, `SELECT sem.class_id FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=?`, ownerOfferingID).Scan(&ownerClassID); err != nil {
		return err
	}
	type participation struct {
		classID, semesterID    int64
		status, semesterStatus string
	}
	current := map[int64]participation{}
	rows, err := tx.QueryContext(ctx, `SELECT teo.course_offering_id,sem.class_id,sem.id,teo.participation_status,sem.status FROM teaching_event_offerings teo JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE teo.teaching_event_id=? AND teo.participation_role='PARTICIPANT'`, eventID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var p participation
		if err := rows.Scan(&id, &p.classID, &p.semesterID, &p.status, &p.semesterStatus); err != nil {
			rows.Close()
			return err
		}
		current[id] = p
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	explicit := ids != nil
	if !explicit {
		for id, p := range current {
			if p.status == "PENDING" || p.status == "ACCEPTED" {
				ids = append(ids, id)
			}
		}
	}
	selected := map[int64]bool{}
	writeAudit := func(id int64, p participation, from, to, action string) error {
		return audit.Write(ctx, tx, audit.Entry{Actor: audit.Actor{Type: "USER", UserID: &actor.UserID, RoleAssignmentID: &actor.RoleAssignmentID}, ClassID: &p.classID, SemesterID: &p.semesterID, Action: action, EntityType: "TEACHING_EVENT", EntityID: &eventID, Before: map[string]any{"offering_id": id, "status": from}, After: map[string]any{"offering_id": id, "status": to}})
	}
	for _, id := range ids {
		if id <= 0 || id == ownerOfferingID || selected[id] {
			return ErrEventInvalidInput
		}
		selected[id] = true
		var p participation
		var offeringStatus string
		if err := tx.QueryRowContext(ctx, `SELECT sem.class_id,sem.id,sem.status,co.status FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=?`, id).Scan(&p.classID, &p.semesterID, &p.semesterStatus, &offeringStatus); err != nil {
			if err == sql.ErrNoRows {
				return ErrEventInvalidInput
			}
			return err
		}
		if p.classID == ownerClassID || p.semesterStatus == "ARCHIVED" || offeringStatus != "ACTIVE" {
			return ErrEventInvalidInput
		}
		valid, err := OfferingPeriodContains(ctx, tx, id, start, end)
		if err != nil {
			return err
		}
		if !valid {
			return ErrEventInvalidInput
		}
		if !explicit {
			continue
		}
		old, exists := current[id]
		if exists && (old.status == "PENDING" || old.status == "ACCEPTED") {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(?,?,'PARTICIPANT','PENDING') ON CONFLICT(teaching_event_id,course_offering_id) DO UPDATE SET participation_status='PENDING',responded_by_user_id=NULL,responded_at=NULL,updated_at=CURRENT_TIMESTAMP`, eventID, id); err != nil {
			return err
		}
		if err := writeAudit(id, p, old.status, "PENDING", "INVITE_PARTICIPANT"); err != nil {
			return err
		}
	}
	if explicit {
		for id, p := range current {
			if selected[id] || p.status == "REMOVED" || p.status == "DECLINED" {
				continue
			}
			if p.semesterStatus == "ARCHIVED" {
				return ErrEventInvalidState
			}
			if _, err := tx.ExecContext(ctx, `UPDATE teaching_event_offerings SET participation_status='REMOVED',responded_by_user_id=?,responded_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP WHERE teaching_event_id=? AND course_offering_id=?`, actor.UserID, eventID, id); err != nil {
				return err
			}
			if err := writeAudit(id, p, p.status, "REMOVED", "REMOVE_PARTICIPANT"); err != nil {
				return err
			}
		}
	}
	return nil
}
