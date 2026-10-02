package v1

import (
	"net/http"

	"bot-jadwal/internal/api/common"
)

func (c *AcademicController) GetRoomConfirmations(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	query := `SELECT rc.id,rc.teaching_event_id,rc.confirmation_status,COALESCE(rc.external_contact,''),COALESCE(rc.note,''),
		r.code,rc.recorded_at,rc.confirmed_at,rc.superseded_at,co.display_name
		FROM room_confirmations rc JOIN rooms r ON r.id=rc.room_id
		JOIN teaching_event_offerings teo ON teo.teaching_event_id=rc.teaching_event_id AND teo.participation_role='OWNER'
		JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id WHERE 1=1`
	args := []any{}
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, 403, common.CodeForbidden, "Konteks kelas diperlukan")
			return
		}
		query += ` AND sem.class_id=?`
		args = append(args, u.ActiveClassID.Int64)
	}
	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid {
			common.WriteV1Error(w, 403, common.CodeForbidden, "Konteks mata kuliah diperlukan")
			return
		}
		query += ` AND co.id=?`
		args = append(args, u.ActiveCourseOfferingID.Int64)
	}
	query += ` ORDER BY rc.id DESC LIMIT 100`
	rows, err := c.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat riwayat konfirmasi")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, event int64
		var status, contact, note, room, recorded, offering string
		var confirmed, superseded any
		if err := rows.Scan(&id, &event, &status, &contact, &note, &room, &recorded, &confirmed, &superseded, &offering); err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca konfirmasi")
			return
		}
		out = append(out, map[string]any{"id": id, "teaching_event_id": event, "confirmation_status": status, "external_contact": contact, "note": note, "room": room, "recorded_at": recorded, "confirmed_at": confirmed, "superseded_at": superseded, "offering": offering})
	}
	if err := rows.Err(); err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca konfirmasi")
		return
	}
	common.WriteV1Success(w, http.StatusOK, out)
}
