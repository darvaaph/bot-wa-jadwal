package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/api/common"
)

func (s *Server) handlePublicationDelivery(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	kind := strings.ToUpper(r.PathValue("entityType"))
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, 422, common.CodeValidation, "ID publikasi tidak valid")
		return
	}
	var query string
	switch kind {
	case "TASK":
		query = `SELECT sem.class_id,co.id FROM tasks t JOIN course_offerings co ON co.id=t.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE t.id=?`
	case "SCHEDULE_PATTERN":
		query = `SELECT sem.class_id,co.id FROM schedule_patterns sp JOIN course_offerings co ON co.id=sp.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE sp.id=?`
	case "TEACHING_EVENT":
		query = `SELECT sem.class_id,co.id FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id=te.id AND teo.participation_role='OWNER' JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE te.id=?`
	default:
		common.WriteV1Error(w, 422, common.CodeValidation, "Jenis publikasi tidak valid")
		return
	}
	var classID, offeringID int64
	if err := s.v1DB.QueryRowContext(r.Context(), query, id).Scan(&classID, &offeringID); err != nil {
		if err == sql.ErrNoRows {
			common.WriteV1Error(w, 404, common.CodeNotFound, "Publikasi tidak ditemukan")
		} else {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat publikasi")
		}
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) || u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offeringID) {
		common.WriteV1Error(w, 404, common.CodeNotFound, "Publikasi tidak ditemukan")
		return
	}
	rows, err := s.v1DB.QueryContext(r.Context(), `SELECT id,event_type,status,created_at,sent_at FROM notification_messages WHERE class_id=? AND entity_type=? AND entity_id=? ORDER BY id DESC LIMIT 20`, classID, kind, id)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat status pengiriman")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var messageID int64
		var eventType, status, created string
		var sent sql.NullString
		if err := rows.Scan(&messageID, &eventType, &status, &created, &sent); err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca status pengiriman")
			return
		}
		item := map[string]any{"id": messageID, "event_type": eventType, "status": status, "created_at": created}
		if sent.Valid {
			item["sent_at"] = sent.String
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca status pengiriman")
		return
	}
	common.WriteV1Success(w, 200, out)
}
