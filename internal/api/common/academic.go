package common

import (
	"database/sql"
	"net/http"

	"bot-jadwal/internal/audit"
)

// GuardAcademicMutation must run through the same transaction as the mutation.
func GuardAcademicMutation(w http.ResponseWriter, r *http.Request, tx *sql.Tx, classID int64, offeringIDs ...int64) bool {
	u, ok := GetAuthContext(r)
	if !ok {
		WriteV1Error(w, 401, CodeUnauthenticated, "Autentikasi diperlukan")
		return false
	}
	var active bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM classes WHERE id=? AND status='ACTIVE')`, classID).Scan(&active); err != nil {
		WriteV1Error(w, 500, "DB_ERROR", "Gagal memeriksa kelas")
		return false
	}
	if !active {
		WriteV1Error(w, 403, CodeForbidden, "Kelas hanya-baca")
		return false
	}
	for _, id := range offeringIDs {
		var status string
		if err := tx.QueryRowContext(r.Context(), `SELECT sem.status FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=? AND sem.class_id=?`, id, classID).Scan(&status); err != nil {
			WriteV1Error(w, 403, CodeForbidden, "Cakupan offering tidak valid")
			return false
		}
		if status == "ARCHIVED" {
			WriteV1Error(w, 403, CodeForbidden, "Semester arsip hanya-baca")
			return false
		}
	}
	if u.ActiveRole == "SYSTEM_ADMIN" {
		var id int64
		var reason string
		if err := tx.QueryRowContext(r.Context(), `SELECT id,reason FROM support_grants WHERE user_id=? AND class_id=? AND status='ACTIVE' AND julianday(expires_at)>julianday('now') ORDER BY id DESC LIMIT 1`, u.UserID, classID).Scan(&id, &reason); err != nil {
			WriteV1Error(w, 403, CodeForbidden, "Mode dukungan aktif untuk kelas ini diperlukan")
			return false
		}
		*r = *r.WithContext(audit.WithSupport(r.Context(), id, classID, reason))
	}
	return true
}

func GuardMaterialMutation(w http.ResponseWriter, r *http.Request, tx *sql.Tx, classID int64, offeringID, taskID sql.NullInt64) bool {
	var ids []int64
	if offeringID.Valid {
		ids = append(ids, offeringID.Int64)
	}
	if taskID.Valid {
		var id int64
		if err := tx.QueryRow(`SELECT course_offering_id FROM tasks WHERE id=?`, taskID).Scan(&id); err != nil {
			WriteV1Error(w, 403, CodeForbidden, "Tugas materi tidak valid")
			return false
		}
		ids = append(ids, id)
	}
	return GuardAcademicMutation(w, r, tx, classID, ids...)
}

func GuardTeachingEventMutation(w http.ResponseWriter, r *http.Request, tx *sql.Tx, eventID int64) bool {
	var classID, offeringID int64
	if err := tx.QueryRow(`SELECT sem.class_id,co.id FROM teaching_event_offerings teo JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE teo.teaching_event_id=? AND teo.participation_role='OWNER'`, eventID).Scan(&classID, &offeringID); err != nil {
		WriteV1Error(w, 403, CodeForbidden, "Cakupan kejadian tidak valid")
		return false
	}
	var archivedParticipant bool
	if err := tx.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM teaching_event_offerings teo JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE teo.teaching_event_id=? AND teo.participation_role='PARTICIPANT' AND teo.participation_status IN ('PENDING','ACCEPTED') AND sem.status='ARCHIVED')`, eventID).Scan(&archivedParticipant); err != nil {
		WriteV1Error(w, 500, "DB_ERROR", "Gagal memeriksa semester peserta")
		return false
	}
	if archivedParticipant {
		WriteV1Error(w, 403, CodeForbidden, "Semester peserta arsip hanya-baca")
		return false
	}
	return GuardAcademicMutation(w, r, tx, classID, offeringID)
}
