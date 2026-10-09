package v1

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/schedule"
)

// CreatePatternRequest adalah payload pembuatan pola jadwal baru
type CreatePatternRequest struct {
	OfferingID  int64   `json:"offering_id"`
	DayOfWeek   int     `json:"day_of_week"` // 1-7
	StartTime   string  `json:"start_time"`  // "HH:MM"
	DurationMin int     `json:"duration_min"`
	RoomID      *int64  `json:"room_id,omitempty"`
	LecturerIDs []int64 `json:"lecturer_ids,omitempty"`
	MeetingLink *string `json:"meeting_link,omitempty"`
}

// CreateTeachingEventRequest adalah payload pembuatan kejadian perkuliahan
type CreateTeachingEventRequest struct {
	OwnerOfferingID        int64   `json:"owner_offering_id"`
	EventKind              string  `json:"event_kind"` // REPLACEMENT, EXTRA, HOLIDAY, SESSION_CANCELLED
	StartsAt               string  `json:"starts_at"`  // UTC RFC3339
	EndsAt                 string  `json:"ends_at"`    // UTC RFC3339
	OriginPatternID        *int64  `json:"origin_pattern_id,omitempty"`
	OriginDate             *string `json:"origin_date,omitempty"`
	ParticipantOfferingIDs []int64 `json:"participant_offering_ids,omitempty"`
	RoomID                 *int64  `json:"room_id,omitempty"`
	Reason                 *string `json:"reason,omitempty"`
	MeetingLink            *string `json:"meeting_link,omitempty"`
}

// PublishEventRequest adalah payload publikasi kejadian perkuliahan
type PublishEventRequest struct {
	Version                int     `json:"version"`
	ConflictOverrideReason *string `json:"conflict_override_reason,omitempty"`
}

// RevokeEventRequest adalah payload pencabutan jadwal oleh KM
type RevokeEventRequest struct {
	Reason  string `json:"reason"`
	Version int    `json:"version"`
}

// PatchPatternRequest adalah payload pembaruan pola jadwal
type PatchPatternRequest struct {
	OfferingID    *int64  `json:"offering_id,omitempty"`
	DayOfWeek     int     `json:"day_of_week"`
	StartTime     string  `json:"start_time"`
	DurationMin   int     `json:"duration_min"`
	RoomID        *int64  `json:"room_id,omitempty"`
	LecturerIDs   []int64 `json:"lecturer_ids,omitempty"`
	MeetingLink   *string `json:"meeting_link,omitempty"`
	Version       int     `json:"version"`
	EffectiveFrom string  `json:"effective_from,omitempty"`
	Reason        string  `json:"reason,omitempty"`
}

// ParticipationRequest merepresentasikan payload persetujuan keikutsertaan kelas gabungan
type ParticipationRequest struct {
	Action string `json:"action"` // accept, decline, leave
}

// ScheduleController mengelola endpoint jadwal v1 (patterns dan teaching events)
type ScheduleController struct {
	db *sql.DB
}

// NewScheduleController membuat instance controller jadwal baru
func NewScheduleController(db *sql.DB) *ScheduleController {
	return &ScheduleController{db: db}
}

// RegisterRoutes mendaftarkan rute jadwal v1 ke ServeMux
func (c *ScheduleController) RegisterRoutes(mux *http.ServeMux, auth *middleware.AuthManager) {
	mux.HandleFunc("GET /api/v1/schedule/patterns", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.GetPatterns)))
	mux.HandleFunc("POST /api/v1/schedule/patterns", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.CreatePattern)))
	mux.HandleFunc("PATCH /api/v1/schedule/patterns/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.PatchPattern)))
	mux.HandleFunc("DELETE /api/v1/schedule/patterns/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.DeletePattern)))
	mux.HandleFunc("POST /api/v1/teaching-events", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.CreateTeachingEvent)))
	mux.HandleFunc("GET /api/v1/teaching-events", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.GetTeachingEvents)))
	mux.HandleFunc("GET /api/v1/teaching-events/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.GetTeachingEventDetail)))
	mux.HandleFunc("DELETE /api/v1/teaching-events/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.DeleteTeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/preview", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.PreviewTeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/publish", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.PublishTeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/revoke", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.RevokeTeachingEvent)))
	mux.HandleFunc("POST /api/v1/teaching-events/{id}/participation", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.ParticipationTeachingEvent)))
}

// GetTeachingEventDetail returns one event together with its offering
// participations and append-only TU confirmation history.
func (c *ScheduleController) GetTeachingEventDetail(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	allowed, err := c.canReadTeachingEvent(r.Context(), u, eventID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi cakupan kejadian jadwal")
		return
	}
	if !allowed {
		if u.ActiveRole == "SYSTEM_ADMIN" {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian jadwal tidak ditemukan")
		} else {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses kejadian jadwal ditolak")
		}
		return
	}

	var roomID sql.NullInt64
	var id, offeringID int64
	var startsAt, endsAt common.DBTimestamp
	var kind, offering, room, reason, lifecycle, meetingLink string
	var version int
	err = c.db.QueryRowContext(r.Context(), `
		SELECT te.id, te.event_kind, co.id, co.display_name, te.starts_at, te.ends_at,
		       te.room_id, COALESCE(r.code, ''), COALESCE(te.reason, ''),
		       te.lifecycle_status, te.version, COALESCE(te.meeting_link, '')
		FROM teaching_events te
		JOIN teaching_event_offerings teo
		  ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		LEFT JOIN rooms r ON r.id = te.room_id
		WHERE te.id = ?`, eventID).Scan(
		&id, &kind, &offeringID, &offering, &startsAt, &endsAt, &roomID,
		&room, &reason, &lifecycle, &version, &meetingLink,
	)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat detail kejadian jadwal")
		return
	}

	var roomValue any
	if roomID.Valid {
		roomValue = roomID.Int64
	}
	event := map[string]any{
		"id":               id,
		"event_kind":       kind,
		"offering_id":      offeringID,
		"offering":         offering,
		"starts_at":        startsAt.RFC3339(),
		"ends_at":          endsAt.RFC3339(),
		"room_id":          roomValue,
		"room":             room,
		"reason":           reason,
		"lifecycle_status": lifecycle,
		"version":          version,
		"meeting_link":     meetingLink,
	}

	participations := []map[string]any{}
	rows, err := c.db.QueryContext(r.Context(), `
		SELECT co.id, co.display_name, teo.participation_role, teo.participation_status
		FROM teaching_event_offerings teo
		JOIN course_offerings co ON co.id = teo.course_offering_id
		WHERE teo.teaching_event_id = ?
		ORDER BY CASE teo.participation_role WHEN 'OWNER' THEN 0 ELSE 1 END, co.display_name`, eventID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat partisipasi kejadian jadwal")
		return
	}
	for rows.Next() {
		var pid int64
		var display, role, status string
		if err := rows.Scan(&pid, &display, &role, &status); err != nil {
			rows.Close()
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca partisipasi kejadian jadwal")
			return
		}
		participations = append(participations, map[string]any{
			"offering_id": pid, "offering": display,
			"participation_role": role, "participation_status": status,
		})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca partisipasi kejadian jadwal")
		return
	}
	rows.Close()

	confirmations := []map[string]any{}
	rows, err = c.db.QueryContext(r.Context(), `
		SELECT rc.id, rc.room_id, r.code, rc.confirmation_status,
		       COALESCE(rc.external_contact, ''), COALESCE(rc.note, ''),
		       u.id, u.display_name, rc.recorded_at, rc.confirmed_at
		FROM room_confirmations rc
		JOIN rooms r ON r.id = rc.room_id
		JOIN users u ON u.id = rc.recorded_by_user_id
		WHERE rc.teaching_event_id = ? AND rc.superseded_at IS NULL
		ORDER BY rc.recorded_at DESC, rc.id DESC`, eventID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat riwayat konfirmasi ruangan")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cid, rid, recordedByID int64
		var roomCode, status, contact, note, recordedByName string
		var recordedAt common.DBTimestamp
		var confirmedAt common.DBTimestamp
		if err := rows.Scan(&cid, &rid, &roomCode, &status, &contact, &note,
			&recordedByID, &recordedByName, &recordedAt, &confirmedAt); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca riwayat konfirmasi ruangan")
			return
		}
		confirmations = append(confirmations, map[string]any{
			"id": cid, "room_id": rid, "room": roomCode,
			"confirmation_status": status, "external_contact": contact, "note": note,
			"recorded_by": map[string]any{"id": recordedByID, "display_name": recordedByName},
			"recorded_at": recordedAt.RFC3339(), "confirmed_at": confirmedAt.RFC3339(),
		})
	}
	if err := rows.Err(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca riwayat konfirmasi ruangan")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"event": event, "participations": participations, "confirmations": confirmations,
	})
}

func (c *ScheduleController) canReadTeachingEvent(ctx context.Context, u *common.UserContext, eventID int64) (bool, error) {
	var exists bool
	var err error
	switch u.ActiveRole {
	case "SYSTEM_ADMIN":
		err = c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM teaching_events WHERE id = ?)`, eventID).Scan(&exists)
	case "PJ":
		if !u.ActiveCourseOfferingID.Valid {
			return false, nil
		}
		err = c.db.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM teaching_event_offerings
			WHERE teaching_event_id = ? AND course_offering_id = ? AND participation_role = 'OWNER'
		)`, eventID, u.ActiveCourseOfferingID.Int64).Scan(&exists)
	case "KM":
		if !u.ActiveClassID.Valid {
			return false, nil
		}
		err = c.db.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM teaching_event_offerings teo
			JOIN course_offerings co ON co.id = teo.course_offering_id
			JOIN semesters sem ON sem.id = co.semester_id
			WHERE teo.teaching_event_id = ? AND sem.class_id = ?
		)`, eventID, u.ActiveClassID.Int64).Scan(&exists)
	default:
		return false, nil
	}
	return exists, err
}

// GetPatterns menangani GET /api/v1/schedule/patterns
func (c *ScheduleController) GetPatterns(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	offeringParam := r.URL.Query().Get("offering_id")
	dayParam := r.URL.Query().Get("day")

	query := `
		SELECT sp.id, sp.course_offering_id, co.display_name, sp.room_id, COALESCE(r.code, ''),
		       COALESCE(r.name, ''), COALESCE(r.building, ''), COALESCE(r.capacity, 0),
		       sp.day_of_week, sp.start_time, sp.end_time, sp.status, sp.version, COALESCE(sp.meeting_link, ''),
		       COALESCE(c.code, ''), COALESCE(co.activity_type, ''),
		       COALESCE((SELECT GROUP_CONCAT(l.full_name, ', ') FROM offering_lecturers ol JOIN lecturers l ON ol.lecturer_id = l.id WHERE ol.course_offering_id = co.id AND ol.superseded_at IS NULL), ''),
		       COALESCE((SELECT u_pj.display_name FROM role_assignments ra JOIN users u_pj ON ra.user_id = u_pj.id WHERE ra.course_offering_id = co.id AND ra.role = 'PJ' AND ra.status = 'ACTIVE' LIMIT 1), '')
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		WHERE sp.status = 'ACTIVE'
	`
	var args []any
	scopeParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if u.ActiveRole == "PJ" && scopeParam != "class" && u.ActiveCourseOfferingID.Valid {
		query += " AND sp.course_offering_id = ?"
		args = append(args, u.ActiveCourseOfferingID.Int64)
	} else if (u.ActiveRole == "KM" || (u.ActiveRole == "PJ" && scopeParam == "class")) && u.ActiveClassID.Valid {
		query += " AND sem.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	}

	if offeringParam != "" {
		offID, _ := strconv.ParseInt(offeringParam, 10, 64)
		if offID > 0 {
			query += " AND sp.course_offering_id = ?"
			args = append(args, offID)
		}
	}
	if dayParam != "" {
		day, _ := strconv.Atoi(dayParam)
		if day >= 1 && day <= 7 {
			query += " AND sp.day_of_week = ?"
			args = append(args, day)
		}
	}

	query += " ORDER BY sp.day_of_week ASC, sp.start_time ASC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat pola jadwal")
		return
	}
	defer rows.Close()

	var patterns []map[string]any
	for rows.Next() {
		var id, offID int64
		var offName, roomCode, roomName, roomBuilding string
		var roomCapacity int
		var dayOfWeek, version int
		var startTime, endTime, status, meetingLink string
		var courseCode, activityType, lecturerNames, pjName string
		var roomID sql.NullInt64

		if err := rows.Scan(
			&id, &offID, &offName, &roomID, &roomCode,
			&roomName, &roomBuilding, &roomCapacity,
			&dayOfWeek, &startTime, &endTime, &status, &version, &meetingLink,
			&courseCode, &activityType, &lecturerNames, &pjName,
		); err == nil {
			patterns = append(patterns, map[string]any{
				"id":                 id,
				"course_offering_id": offID,
				"offering":           offName,
				"display_name":       offName,
				"course_code":        courseCode,
				"activity_type":      activityType,
				"room_id": func() any {
					if roomID.Valid {
						return roomID.Int64
					}
					return nil
				}(),
				"room":          roomCode,
				"room_name":     roomName,
				"room_building": roomBuilding,
				"room_capacity": roomCapacity,
				"day_of_week":   dayOfWeek,
				"start_time":    startTime,
				"end_time":      endTime,
				"status":        status,
				"version":       version,
				"meeting_link":  meetingLink,
				"lecturer":      lecturerNames,
				"pj_name":       pjName,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, patterns)
}

// CreatePattern menangani POST /api/v1/schedule/patterns
func (c *ScheduleController) CreatePattern(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreatePatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	if req.DayOfWeek < 1 || req.DayOfWeek > 7 || req.DurationMin <= 0 || req.DurationMin >= 24*60 || strings.TrimSpace(req.StartTime) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "day_of_week (1-7), start_time (HH:MM), dan duration_min wajib valid")
		return
	}

	startT, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format start_time harus HH:MM")
		return
	}
	if startT.Hour()*60+startT.Minute()+req.DurationMin >= 24*60 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Jadwal harus selesai pada hari yang sama")
		return
	}
	endT := startT.Add(time.Duration(req.DurationMin) * time.Minute)
	endTime := endT.Format("15:04")
	var classID int64
	var semesterStart, semesterEnd, semesterStatus string
	if err := c.db.QueryRow(`
		SELECT sem.class_id, sem.starts_on, sem.ends_on, sem.status FROM course_offerings co
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE co.id = ? AND co.status = 'ACTIVE';
	`, req.OfferingID).Scan(&classID, &semesterStart, &semesterEnd, &semesterStatus); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Course offering tidak ditemukan atau tidak aktif")
		return
	}
	if u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != req.OfferingID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya diizinkan membuat pola untuk mata kuliah miliknya")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya diizinkan membuat pola untuk kelasnya")
		return
	}
	for _, lecturerID := range req.LecturerIDs {
		var assigned int
		if err := c.db.QueryRow(`SELECT COUNT(*) FROM offering_lecturers WHERE course_offering_id = ? AND lecturer_id = ? AND superseded_at IS NULL;`, req.OfferingID, lecturerID).Scan(&assigned); err != nil || assigned != 1 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "lecturer_ids harus terdaftar pada offering")
			return
		}
	}

	lectIDs := req.LecturerIDs
	if lectIDs == nil {
		rows, _ := c.db.Query(`SELECT lecturer_id FROM offering_lecturers WHERE course_offering_id=? AND superseded_at IS NULL`, req.OfferingID)
		if rows != nil {
			for rows.Next() {
				var lid int64
				if err := rows.Scan(&lid); err == nil {
					lectIDs = append(lectIDs, lid)
				}
			}
			rows.Close()
		}
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.Local
	}
	semesterStart, semesterEnd = semesterStart[:10], semesterEnd[:10]
	var effectiveFrom string
	switch semesterStatus {
	case "DRAFT":
		effectiveFrom = semesterStart
	case "ACTIVE":
		effectiveFrom = time.Now().In(loc).Format("2006-01-02")
		if effectiveFrom < semesterStart || effectiveFrom > semesterEnd {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Pola baru hanya dapat dibuat dalam semester aktif")
			return
		}
	default:
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Pola baru hanya dapat dibuat dalam semester draf atau aktif")
		return
	}
	conflicts, conflictErr := checkPatternRange(r.Context(), c.db, schedule.Candidate{
		OwnerClassID:    classID,
		OwnerOfferingID: req.OfferingID,
		RoomID:          req.RoomID,
		LecturerIDs:     lectIDs,
		PatternDay:      req.DayOfWeek,
		PatternStart:    req.StartTime,
		PatternEnd:      endTime,
	}, effectiveFrom, semesterEnd)
	if conflictErr != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa konflik jadwal")
		return
	}
	if schedule.HasBlocking(conflicts) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, conflictMessage(conflicts))
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var patternID int64
	var meetingLink any
	if req.MeetingLink != nil && strings.TrimSpace(*req.MeetingLink) != "" {
		meetingLink = strings.TrimSpace(*req.MeetingLink)
	}
	err = tx.QueryRow(`
		INSERT INTO schedule_patterns (
			course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, version, meeting_link
		)
		VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		RETURNING id;
	`, req.OfferingID, req.RoomID, req.DayOfWeek, req.StartTime, endTime, effectiveFrom, meetingLink).Scan(&patternID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan pola jadwal: %v", err))
		return
	}

	correlationID := fmt.Sprintf("create-pattern-%d-%d", patternID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"day":%d,"start":%q,"end":%q}`, req.DayOfWeek, req.StartTime, endTime)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "CREATE_PATTERN",
			EntityType:    "SCHEDULE_PATTERN",
			EntityID:      &patternID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit log")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi")
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":             patternID,
		"day_of_week":    req.DayOfWeek,
		"start_time":     req.StartTime,
		"end_time":       endTime,
		"version":        1,
		"effective_from": effectiveFrom,
	})
}

// CreateTeachingEvent menangani POST /api/v1/teaching-events
func (c *ScheduleController) CreateTeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreateTeachingEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	kind := strings.ToUpper(strings.TrimSpace(req.EventKind))
	if kind != "REPLACEMENT" && kind != "EXTRA" && kind != "HOLIDAY" && kind != "SESSION_CANCELLED" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "event_kind harus REPLACEMENT, EXTRA, HOLIDAY, atau SESSION_CANCELLED")
		return
	}

	if (kind == "REPLACEMENT" || kind == "SESSION_CANCELLED") && (req.OriginPatternID == nil || req.OriginDate == nil) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kejadian pengganti atau pembatalan sesi wajib menyertakan origin_pattern_id dan origin_date")
		return
	}

	startsAt, errStart := time.Parse(time.RFC3339, req.StartsAt)
	endsAt, errEnd := time.Parse(time.RFC3339, req.EndsAt)
	if errStart != nil || errEnd != nil || !endsAt.After(startsAt) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Waktu starts_at dan ends_at harus berupa RFC3339 UTC dan ends_at harus setelah starts_at")
		return
	}
	var ownerClassID int64
	var semesterStarts, semesterEnds string
	if err := c.db.QueryRow(`
		SELECT sem.class_id, sem.starts_on, sem.ends_on
		FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id
		WHERE co.id = ? AND co.status = 'ACTIVE';
	`, req.OwnerOfferingID).Scan(&ownerClassID, &semesterStarts, &semesterEnds); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Owner offering tidak ditemukan atau tidak aktif")
		return
	}
	startDate := startsAt.Format("2006-01-02")
	endDate := endsAt.Format("2006-01-02")
	if startDate < semesterStarts || endDate > semesterEnds {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Waktu kejadian harus berada dalam rentang semester owner")
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != req.OwnerOfferingID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya diizinkan membuat jadwal untuk mata kuliah miliknya")
			return
		}
	} else if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != ownerClassID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya diizinkan membuat jadwal untuk kelasnya")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var eventID int64
	var eventLink any
	if req.MeetingLink != nil && strings.TrimSpace(*req.MeetingLink) != "" {
		eventLink = strings.TrimSpace(*req.MeetingLink)
	}
	err = tx.QueryRow(`
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status, version, meeting_link
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'DRAFT', 1, ?)
		RETURNING id;
	`, req.OriginPatternID, req.OriginDate, kind, startsAt, endsAt, req.RoomID, req.Reason, eventLink).Scan(&eventID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan teaching event: %v", err))
		return
	}

	_, err = tx.Exec(`
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (?, ?, 'OWNER', 'ACCEPTED');
	`, eventID, req.OwnerOfferingID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghubungkan offering pemilik event")
		return
	}
	for _, participantID := range req.ParticipantOfferingIDs {
		if participantID <= 0 || participantID == req.OwnerOfferingID {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "participant_offering_ids tidak valid")
			return
		}
		participantResult, insertErr := tx.Exec(`
			INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
			SELECT ?, co.id, 'PARTICIPANT', 'PENDING'
			FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id
			WHERE co.id = ? AND co.status = 'ACTIVE' AND ? >= sem.starts_on AND ? <= sem.ends_on;
		`, eventID, participantID, startDate, endDate)
		if insertErr != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Participant offering tidak valid")
			return
		}
		participantRows, _ := participantResult.RowsAffected()
		if participantRows != 1 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Participant offering tidak ditemukan atau tidak aktif")
			return
		}
	}
	correlationID := fmt.Sprintf("create-event-%d-%d", eventID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"kind":%q,"offering_id":%d}`, kind, req.OwnerOfferingID)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &ownerClassID,
			Action:        "CREATE_TEACHING_EVENT",
			EntityType:    "TEACHING_EVENT",
			EntityID:      &eventID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit teaching event")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit teaching event")
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":               eventID,
		"event_kind":       kind,
		"lifecycle_status": "DRAFT",
		"version":          1,
	})
}

// GetTeachingEvents menangani GET /api/v1/teaching-events
func (c *ScheduleController) GetTeachingEvents(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))

	query := `
		SELECT te.id, te.event_kind, co.id, co.display_name, te.starts_at, te.ends_at,
		       COALESCE(r.code, ''), COALESCE(te.reason, ''), te.lifecycle_status, te.version,
		       COALESCE(te.meeting_link, '')
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE 1=1
	`
	var args []any

	scopeParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("scope")))
	if u.ActiveRole == "PJ" && scopeParam != "class" {
		if !u.ActiveCourseOfferingID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks offering PJ tidak aktif")
			return
		}
		query += " AND co.id = ?"
		args = append(args, u.ActiveCourseOfferingID.Int64)
	} else if u.ActiveRole == "KM" || (u.ActiveRole == "PJ" && scopeParam == "class") {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks kelas tidak aktif")
			return
		}
		query += ` AND EXISTS (
			SELECT 1 FROM teaching_event_offerings visible_teo
			JOIN course_offerings visible_co ON visible_co.id = visible_teo.course_offering_id
			JOIN semesters visible_sem ON visible_sem.id = visible_co.semester_id
			WHERE visible_teo.teaching_event_id = te.id AND visible_sem.class_id = ?
		)`
		args = append(args, u.ActiveClassID.Int64)
	}

	if statusFilter != "" {
		query += " AND te.lifecycle_status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY te.starts_at DESC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kejadian perkuliahan")
		return
	}
	defer rows.Close()

	var events []map[string]any
	for rows.Next() {
		var id, offID int64
		var kind, offDisplay, roomCode, reason, lifeStatus, meetingLink string
		var startsAt, endsAt common.DBTimestamp
		var version int

		if err := rows.Scan(&id, &kind, &offID, &offDisplay, &startsAt, &endsAt, &roomCode, &reason, &lifeStatus, &version, &meetingLink); err == nil {
			events = append(events, map[string]any{
				"id":               id,
				"event_kind":       kind,
				"offering_id":      offID,
				"offering":         offDisplay,
				"starts_at":        startsAt.RFC3339(),
				"ends_at":          endsAt.RFC3339(),
				"room":             roomCode,
				"reason":           reason,
				"lifecycle_status": lifeStatus,
				"version":          version,
				"meeting_link":     meetingLink,
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, events)
}

// PublishTeachingEvent menangani POST /api/v1/teaching-events/{id}/publish
func (c *ScheduleController) PublishTeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req PublishEventRequest
	if r.Body == nil || r.Body == http.NoBody {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib diisi")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	if req.Version <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib lebih dari nol")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Header Idempotency-Key wajib diisi")
		return
	}
	keyHash := sha256.Sum256([]byte(idempotencyKey))
	correlationID := fmt.Sprintf("publish-event-%d-%s", eventID, hex.EncodeToString(keyHash[:]))

	var (
		curVersion  int
		curStatus   string
		offeringID  int64
		classID     int64
		roomID      sql.NullInt64
		startsAt    common.DBTimestamp
		endsAt      common.DBTimestamp
		offName     string
		reason      sql.NullString
		roomCode    sql.NullString
		originPatID sql.NullInt64
	)

	err = c.db.QueryRow(`
		SELECT te.version, te.lifecycle_status, te.room_id, te.starts_at, te.ends_at, te.reason,
		       co.id, sem.class_id, co.display_name, r.code, te.origin_schedule_pattern_id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE te.id = ?;
	`, eventID).Scan(&curVersion, &curStatus, &roomID, &startsAt, &endsAt, &reason,
		&offeringID, &classID, &offName, &roomCode, &originPatID)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian jadwal tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal membaca kejadian jadwal: %v", err))
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offeringID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat mempublikasikan untuk kelas Anda")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat mempublikasikan untuk kelas Anda")
			return
		}
	} else if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Peran Anda tidak diizinkan")
		return
	}

	if curStatus == "PUBLISHED" {
		var auditID int64
		err := c.db.QueryRow(`
			SELECT id FROM audit_logs
			WHERE action = 'PUBLISH_TEACHING_EVENT' AND entity_type = 'TEACHING_EVENT'
			  AND entity_id = ? AND correlation_id = ?
			LIMIT 1;
		`, eventID, correlationID).Scan(&auditID)
		if err == nil {
			common.WriteV1Success(w, http.StatusOK, map[string]any{"id": eventID, "lifecycle_status": "PUBLISHED"})
			return
		}
		if err != sql.ErrNoRows {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi idempotency key")
			return
		}
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Kejadian sudah dipublikasikan dengan idempotency key berbeda")
		return
	}

	if curStatus != "DRAFT" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Hanya DRAFT yang dapat dipublikasikan")
		return
	}

	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"lifecycle_status": curStatus,
			},
		})
		return
	}
	{
		var candRoom *int64
		if roomID.Valid {
			v := roomID.Int64
			candRoom = &v
		}
		lectIDs := lecturersForOfferingCtx(r.Context(), c.db, offeringID)
		partIDs := participantOfferingIDs(r.Context(), c.db, eventID, offeringID)
		var excludePat *int64
		if originPatID.Valid {
			v := originPatID.Int64
			excludePat = &v
		}
		conflicts, err := schedule.CheckConflicts(r.Context(), c.db, schedule.Candidate{
			OwnerClassID:     classID,
			OwnerOfferingID:  offeringID,
			ParticipantIDs:   partIDs,
			RoomID:           candRoom,
			LecturerIDs:      lectIDs,
			StartsAt:         startsAt.Time,
			EndsAt:           endsAt.Time,
			ExcludeEventID:   &eventID,
			ExcludePatternID: excludePat,
		})
		if err == nil {
			for _, conf := range conflicts {
				if conf.Code == schedule.CodeOutsideOwnerSem || conf.Code == schedule.CodeOutsidePartSem {
					common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, conflictMessage(conflicts))
					return
				}
			}
			if schedule.HasBlocking(conflicts) {
				common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, conflictMessage(conflicts))
				return
			}
			if schedule.NeedsOverride(conflicts) && (req.ConflictOverrideReason == nil || strings.TrimSpace(*req.ConflictOverrideReason) == "") {
				common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Konflik nonblocking memerlukan conflict_override_reason")
				return
			}
		}
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE teaching_events
		SET lifecycle_status = 'PUBLISHED', published_by_user_id = ?, published_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = ? AND version = ?;
	`, u.UserID, eventID, curVersion)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menerbitkan jadwal: %v", err))
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Konflik versi saat menyimpan")
		return
	}

	afterJSON := fmt.Sprintf(`{"lifecycle_status":"PUBLISHED","room_id":%v,"version":%d}`, func() any {
		if roomID.Valid {
			return roomID.Int64
		}
		return "null"
	}(), curVersion+1)

	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		var semIDPtr *int64
		if sid := func() *int64 {
			var sid int64
			if err := tx.QueryRow(`SELECT semester_id FROM course_offerings WHERE id = ?`, offeringID).Scan(&sid); err == nil {
				return &sid
			}
			return nil
		}(); sid != nil {
			semIDPtr = sid
		}
		publishReason := ""
		if req.ConflictOverrideReason != nil {
			publishReason = *req.ConflictOverrideReason
		}
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			SemesterID:    semIDPtr,
			Action:        "PUBLISH_TEACHING_EVENT",
			EntityType:    "TEACHING_EVENT",
			EntityID:      &eventID,
			AfterJSON:     &afterJSON,
			Reason:        publishReason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mencatat audit log: %v", err))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi")
		return
	}

	rc := "-"
	if roomCode.Valid {
		rc = roomCode.String
	}
	rs := ""
	if reason.Valid {
		rs = reason.String
	}
	common.QueueNotification(c.db, classID, "SCHEDULE_REPLACEMENT", "TEACHING_EVENT", eventID, map[string]any{
		"course":    offName,
		"starts_at": startsAt.Time.Format(time.RFC3339),
		"room":      rc,
		"reason":    rs,
	}, u.UserID)

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id":               eventID,
		"lifecycle_status": "PUBLISHED",
	})
}

// RevokeTeachingEvent menangani POST /api/v1/teaching-events/{id}/revoke
func (c *ScheduleController) RevokeTeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang membatalkan/mencabut jadwal")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req RevokeEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan pencabutan (reason) wajib diisi")
		return
	}

	var (
		curVersion int
		curStatus  string
		classID    int64
		offName    string
		offeringID int64
	)
	err = c.db.QueryRow(`
		SELECT te.version, te.lifecycle_status, sem.class_id, co.display_name, co.id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE te.id = ?;
	`, eventID).Scan(&curVersion, &curStatus, &classID, &offName, &offeringID)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian jadwal tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca kejadian jadwal")
		return
	}

	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat mencabut jadwal untuk kelas Anda")
			return
		}
	}

	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"lifecycle_status": curStatus,
			},
		})
		return
	}

	if curStatus != "PUBLISHED" {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Hanya jadwal yang sudah diterbitkan yang dapat dicabut")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE teaching_events
		SET lifecycle_status = 'REVOKED', revoked_by_user_id = ?, revoked_at = CURRENT_TIMESTAMP,
		    revocation_reason = ?, version = version + 1
		WHERE id = ? AND version = ?;
	`, u.UserID, req.Reason, eventID, curVersion)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencabut jadwal")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Konflik versi saat menyimpan")
		return
	}

	beforeJSON := fmt.Sprintf(`{"lifecycle_status":"%s"}`, curStatus)
	afterJSON := fmt.Sprintf(`{"lifecycle_status":"REVOKED","revocation_reason":%q,"version":%d}`, req.Reason, curVersion+1)

	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		var semIDPtr *int64
		if sid := func() *int64 {
			var sid int64
			if err := tx.QueryRow(`SELECT semester_id FROM course_offerings WHERE id = ?`, offeringID).Scan(&sid); err == nil {
				return &sid
			}
			return nil
		}(); sid != nil {
			semIDPtr = sid
		}
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			SemesterID:    semIDPtr,
			Action:        "REVOKE_TEACHING_EVENT",
			EntityType:    "TEACHING_EVENT",
			EntityID:      &eventID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        req.Reason,
			CorrelationID: r.Header.Get("X-Correlation-ID"),
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit log")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi")
		return
	}

	common.QueueNotification(c.db, classID, "SCHEDULE_REVOKED", "TEACHING_EVENT", eventID, map[string]any{
		"course": offName,
		"reason": req.Reason,
	}, u.UserID)

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id":               eventID,
		"lifecycle_status": "REVOKED",
	})
}

// PatchPattern menangani PATCH /api/v1/schedule/patterns/{id}
func (c *ScheduleController) PatchPattern(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	patternIDStr := r.PathValue("id")
	patternID, err := strconv.ParseInt(patternIDStr, 10, 64)
	if err != nil || patternID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID pola jadwal tidak valid")
		return
	}

	var req PatchPatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	var (
		curOfferingID    int64
		curRoomID        sql.NullInt64
		curDayOfWeek     int
		curStartTime     string
		curEndTime       string
		curClassID       int64
		curVersion       int
		curLink          sql.NullString
		curEffectiveFrom string
		semesterStart    string
		semesterEnd      string
		semesterStatus   string
	)

	err = c.db.QueryRow(`
		SELECT sp.course_offering_id, sp.room_id, sp.day_of_week, sp.start_time, sp.end_time, s.class_id, sp.version, sp.meeting_link,
		       sp.effective_from, s.starts_on, s.ends_on, s.status
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN semesters s ON co.semester_id = s.id
		WHERE sp.id = ? AND sp.status='ACTIVE' AND sp.effective_until IS NULL;
	`, patternID).Scan(&curOfferingID, &curRoomID, &curDayOfWeek, &curStartTime, &curEndTime, &curClassID, &curVersion, &curLink,
		&curEffectiveFrom, &semesterStart, &semesterEnd, &semesterStatus)

	if err == sql.ErrNoRows {
		var closedVersion int
		if cerr := c.db.QueryRow(`SELECT version FROM schedule_patterns WHERE id = ?`, patternID).Scan(&closedVersion); cerr == nil {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Pola jadwal sudah diperbarui ke versi baru", map[string]any{"current_version": closedVersion})
			return
		}
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pola jadwal tidak ditemukan atau sudah tidak aktif")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pola jadwal")
		return
	}
	if req.Version <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib lebih dari nol")
		return
	}
	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi pola jadwal berubah", map[string]any{"current_version": curVersion})
		return
	}
	curEffectiveFrom = curEffectiveFrom[:10]
	semesterStart = semesterStart[:10]
	semesterEnd = semesterEnd[:10]

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != curOfferingID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya diizinkan memperbarui pola jadwal untuk mata kuliah miliknya")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya diizinkan memperbarui pola jadwal di kelasnya")
			return
		}
	}

	newOfferingID := curOfferingID
	if req.OfferingID != nil && *req.OfferingID != curOfferingID {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Perubahan permanen harus tetap pada mata kuliah asal")
		return
	}

	newStartTime := curStartTime
	if strings.TrimSpace(req.StartTime) != "" {
		newStartTime = strings.TrimSpace(req.StartTime)
		if _, err := time.Parse("15:04", newStartTime); err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format start_time harus HH:MM")
			return
		}
	}
	newEndTime := curEndTime
	if req.DurationMin > 0 {
		st, err := time.Parse("15:04", newStartTime)
		if err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format start_time harus HH:MM")
			return
		}
		newEndTime = st.Add(time.Duration(req.DurationMin) * time.Minute).Format("15:04")
	} else if req.DurationMin < 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "duration_min tidak boleh negatif")
		return
	}
	newDayOfWeek := curDayOfWeek
	if req.DayOfWeek >= 1 && req.DayOfWeek <= 7 {
		newDayOfWeek = req.DayOfWeek
	} else if req.DayOfWeek != 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "day_of_week harus 1 sampai 7")
		return
	}
	newRoomID := curRoomID
	if req.RoomID != nil {
		newRoomID = sql.NullInt64{Int64: *req.RoomID, Valid: true}
	}
	for _, lecturerID := range req.LecturerIDs {
		var assigned int
		if err := c.db.QueryRow(`SELECT COUNT(*) FROM offering_lecturers WHERE course_offering_id = ? AND lecturer_id = ? AND superseded_at IS NULL;`, newOfferingID, lecturerID).Scan(&assigned); err != nil || assigned != 1 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "lecturer_ids harus terdaftar pada offering")
			return
		}
	}

	var patchRoom *int64
	if newRoomID.Valid {
		v := newRoomID.Int64
		patchRoom = &v
	}
	lectIDs := req.LecturerIDs
	if lectIDs == nil {
		rows, _ := c.db.Query(`SELECT lecturer_id FROM offering_lecturers WHERE course_offering_id=? AND superseded_at IS NULL`, newOfferingID)
		if rows != nil {
			for rows.Next() {
				var lid int64
				if err := rows.Scan(&lid); err == nil {
					lectIDs = append(lectIDs, lid)
				}
			}
			rows.Close()
		}
	}
	location, _ := time.LoadLocation("Asia/Jakarta")
	if location == nil {
		location = time.Local
	}
	if semesterStatus == "DRAFT" {
		effectiveDate := semesterStart
		conflicts, conflictErr := checkPatternRange(r.Context(), c.db, schedule.Candidate{
			OwnerClassID:     curClassID,
			OwnerOfferingID:  newOfferingID,
			RoomID:           patchRoom,
			LecturerIDs:      lectIDs,
			PatternDay:       newDayOfWeek,
			PatternStart:     newStartTime,
			PatternEnd:       newEndTime,
			ExcludePatternID: &patternID,
		}, effectiveDate, semesterEnd)
		if conflictErr != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa konflik jadwal")
			return
		}
		if schedule.HasBlocking(conflicts) {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, conflictMessage(conflicts))
			return
		}
		newLink := curLink.String
		if req.MeetingLink != nil {
			newLink = strings.TrimSpace(*req.MeetingLink)
		}
		var newLinkArg any
		if newLink != "" {
			newLinkArg = newLink
		}
		tx, err := c.db.Begin()
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
			return
		}
		defer tx.Rollback()
		res, err := tx.Exec(`
			UPDATE schedule_patterns
			SET day_of_week = ?, start_time = ?, end_time = ?, room_id = ?, meeting_link = ?,
			    effective_from = ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
			WHERE id = ? AND version = ? AND status='ACTIVE' AND effective_until IS NULL;
		`, newDayOfWeek, newStartTime, newEndTime, newRoomID, newLinkArg, effectiveDate, patternID, curVersion)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui pola jadwal draf")
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi pola jadwal berubah", map[string]any{"current_version": curVersion})
			return
		}
		correlationID := fmt.Sprintf("update-pattern-draft-%d-%d", patternID, time.Now().UnixNano())
		{
			uid := u.UserID
			var raid *int64
			if u.ActiveAssignmentID != 0 {
				v := u.ActiveAssignmentID
				raid = &v
			}
			beforeJSON := fmt.Sprintf(`{"old_id":%d,"day":%d,"start":%q}`, patternID, curDayOfWeek, curStartTime)
			afterJSON := fmt.Sprintf(`{"id":%d,"day":%d,"start":%q,"effective_from":%q}`, patternID, newDayOfWeek, newStartTime, effectiveDate)
			reason := strings.TrimSpace(req.Reason)
			var reasonPtr string
			if reason != "" {
				reasonPtr = reason
			}
			_ = reasonPtr
			if err := audit.Write(r.Context(), tx, audit.Entry{
				Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
				ClassID:       &curClassID,
				Action:        "UPDATE_PATTERN",
				EntityType:    "SCHEDULE_PATTERN",
				EntityID:      &patternID,
				BeforeJSON:    &beforeJSON,
				AfterJSON:     &afterJSON,
				Reason:        reason,
				CorrelationID: correlationID,
			}); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit pola jadwal")
				return
			}
		}
		if err := tx.Commit(); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit pola jadwal")
			return
		}
		common.WriteV1Success(w, http.StatusOK, map[string]any{
			"id":                  patternID,
			"replaces_pattern_id": patternID,
			"day_of_week":         newDayOfWeek,
			"start_time":          newStartTime,
			"end_time":            newEndTime,
			"effective_from":      effectiveDate,
			"effective_until":     nil,
			"version":             curVersion + 1,
		})
		return
	}
	today := time.Now().In(location)
	effectiveDate := today.AddDate(0, 0, 1).Format("2006-01-02")
	if req.EffectiveFrom != "" {
		effectiveDate = strings.TrimSpace(req.EffectiveFrom)
	}
	parsedDate, dateErr := time.ParseInLocation("2006-01-02", effectiveDate, location)
	if dateErr != nil || parsedDate.Format("2006-01-02") != effectiveDate || effectiveDate < today.Format("2006-01-02") || effectiveDate < semesterStart || effectiveDate > semesterEnd || semesterStatus != "ACTIVE" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tanggal berlaku harus dari hari ini sampai akhir semester aktif")
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan perubahan permanen wajib diisi")
		return
	}
	if effectiveDate < curEffectiveFrom {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tanggal berlaku mendahului pola jadwal asal")
		return
	}
	conflicts, conflictErr := checkPatternRange(r.Context(), c.db, schedule.Candidate{
		OwnerClassID:     curClassID,
		OwnerOfferingID:  newOfferingID,
		RoomID:           patchRoom,
		LecturerIDs:      lectIDs,
		PatternDay:       newDayOfWeek,
		PatternStart:     newStartTime,
		PatternEnd:       newEndTime,
		ExcludePatternID: &patternID,
	}, effectiveDate, semesterEnd)
	if conflictErr != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa konflik jadwal")
		return
	}
	if schedule.HasBlocking(conflicts) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, conflictMessage(conflicts))
		return
	}

	previousDate := parsedDate.AddDate(0, 0, -1).Format("2006-01-02")
	oldStatus := "ACTIVE"
	if effectiveDate == curEffectiveFrom {
		previousDate = curEffectiveFrom
		oldStatus = "SUPERSEDED"
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		UPDATE schedule_patterns
		SET effective_until = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND version = ? AND status='ACTIVE' AND effective_until IS NULL;
	`, previousDate, oldStatus, patternID, curVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui pola jadwal lama")
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi pola jadwal berubah")
		return
	}

	var newPatternID int64
	newLink := curLink.String
	if req.MeetingLink != nil {
		newLink = strings.TrimSpace(*req.MeetingLink)
	}
	err = tx.QueryRow(`
		INSERT INTO schedule_patterns (
			course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, version, meeting_link
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id;
	`, newOfferingID, newRoomID, newDayOfWeek, newStartTime, newEndTime, effectiveDate, curVersion+1, newLink).Scan(&newPatternID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan pola jadwal baru")
		return
	}

	correlationID := fmt.Sprintf("update-pattern-%d-%d", patternID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"old_id":%d,"day":%d,"start":%q}`, patternID, curDayOfWeek, curStartTime)
		afterJSON := fmt.Sprintf(`{"new_id":%d,"day":%d,"start":%q,"effective_from":%q}`, newPatternID, newDayOfWeek, newStartTime, effectiveDate)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &curClassID,
			Action:        "UPDATE_PATTERN",
			EntityType:    "SCHEDULE_PATTERN",
			EntityID:      &newPatternID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        strings.TrimSpace(req.Reason),
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit pola jadwal")
			return
		}
	}
	var channelID any
	var scheduledAt any
	var activeChannel int64
	if err := tx.QueryRow(`SELECT id FROM whatsapp_channels WHERE class_id = ? AND status = 'ACTIVE' ORDER BY id LIMIT 1`, curClassID).Scan(&activeChannel); err == nil {
		channelID = activeChannel
		scheduledAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	message, _ := json.Marshal(map[string]string{"text": fmt.Sprintf("Jadwal tetap berubah mulai %s. %s", effectiveDate, strings.TrimSpace(req.Reason))})
	if _, err := tx.Exec(`INSERT INTO notification_messages
		(class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id)
		VALUES (?, ?, 'SCHEDULE_CHANGE', 'SCHEDULE_PATTERN', ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING`, curClassID, channelID, newPatternID,
		fmt.Sprintf("pattern-version:%d:%d", patternID, curVersion+1), string(message), scheduledAt, u.UserID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengantrekan notifikasi perubahan pola")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit pola jadwal")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id":                  newPatternID,
		"replaces_pattern_id": patternID,
		"day_of_week":         newDayOfWeek,
		"start_time":          newStartTime,
		"end_time":            newEndTime,
		"effective_from":      effectiveDate,
		"effective_until":     nil,
		"version":             curVersion + 1,
	})
}

// PreviewTeachingEvent menangani POST /api/v1/teaching-events/{id}/preview
func (c *ScheduleController) PreviewTeachingEvent(w http.ResponseWriter, r *http.Request) {
	_, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var (
		kind              string
		startsAt          common.DBTimestamp
		endsAt            common.DBTimestamp
		roomID            sql.NullInt64
		roomCode          sql.NullString
		offName           string
		originPatternID   sql.NullInt64
		previewOfferingID int64
		previewClassID    int64
		meetingLink       sql.NullString
	)

	err = c.db.QueryRow(`
		SELECT te.event_kind, te.starts_at, te.ends_at, te.room_id, r.code, co.display_name, te.origin_schedule_pattern_id,
		       co.id, sem.class_id, te.meeting_link
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE te.id = ?;
	`, eventID).Scan(&kind, &startsAt, &endsAt, &roomID, &roomCode, &offName, &originPatternID, &previewOfferingID, &previewClassID, &meetingLink)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kejadian jadwal")
		return
	}

	var oldData map[string]any
	if originPatternID.Valid {
		var (
			pOffName   string
			pRoomCode  sql.NullString
			pDayOfWeek int
			pStartTime string
			pEndTime   string
		)
		err = c.db.QueryRow(`
			SELECT co.display_name, r.code, sp.day_of_week, sp.start_time, sp.end_time
			FROM schedule_patterns sp
			JOIN course_offerings co ON sp.course_offering_id = co.id
			LEFT JOIN rooms r ON sp.room_id = r.id
			WHERE sp.id = ?;
		`, originPatternID.Int64).Scan(&pOffName, &pRoomCode, &pDayOfWeek, &pStartTime, &pEndTime)
		if err == nil {
			oldData = map[string]any{
				"offering":    pOffName,
				"room":        pRoomCode.String,
				"day_of_week": pDayOfWeek,
				"start_time":  pStartTime,
				"end_time":    pEndTime,
			}
		}
	}

	var conflicts []map[string]any
	{
		var candRoom *int64
		if roomID.Valid {
			v := roomID.Int64
			candRoom = &v
		}
		lectIDs := lecturersForOfferingCtx(r.Context(), c.db, previewOfferingID)
		partIDs := participantOfferingIDs(r.Context(), c.db, eventID, previewOfferingID)
		var excludePat *int64
		if originPatternID.Valid {
			v := originPatternID.Int64
			excludePat = &v
		}
		if engineConflicts, err := schedule.CheckConflicts(r.Context(), c.db, schedule.Candidate{
			OwnerClassID:     previewClassID,
			OwnerOfferingID:  previewOfferingID,
			ParticipantIDs:   partIDs,
			RoomID:           candRoom,
			LecturerIDs:      lectIDs,
			StartsAt:         startsAt.Time,
			EndsAt:           endsAt.Time,
			ExcludeEventID:   &eventID,
			ExcludePatternID: excludePat,
		}); err == nil {
			conflicts = conflictMaps(engineConflicts)
		} else {
			conflicts = []map[string]any{}
		}
	}

	roomNote := ""
	if roomID.Valid {
		roomNote = "perlu konfirmasi TU"
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"old": oldData,
		"new": map[string]any{
			"offering":     offName,
			"starts_at":    startsAt.Time.Format(time.RFC3339),
			"ends_at":      endsAt.Time.Format(time.RFC3339),
			"room":         roomCode.String,
			"meeting_link": meetingLink.String,
		},
		"kind":      kind,
		"conflicts": conflicts,
		"room_note": roomNote,
	})
}

// ParticipationTeachingEvent menangani POST /api/v1/teaching-events/{id}/participation
func (c *ScheduleController) ParticipationTeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req ParticipationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	var newStatus string
	switch action {
	case "accept":
		newStatus = "ACCEPTED"
	case "decline":
		newStatus = "DECLINED"
	case "leave":
		newStatus = "REMOVED"
	default:
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Aksi partisipasi harus salah satu dari: accept, decline, leave")
		return
	}

	var res sql.Result
	if u.ActiveRole == "SYSTEM_ADMIN" {
		res, err = c.db.Exec(`
			UPDATE teaching_event_offerings
			SET participation_status = ?
			WHERE teaching_event_id = ?
			  AND participation_role = 'PARTICIPANT';
		`, newStatus, eventID)
	} else {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks kelas tidak valid untuk partisipasi")
			return
		}
		res, err = c.db.Exec(`
			UPDATE teaching_event_offerings
			SET participation_status = ?
			WHERE teaching_event_id = ?
			  AND participation_role = 'PARTICIPANT'
			  AND course_offering_id IN (
			      SELECT co.id FROM course_offerings co
			      JOIN semesters s ON co.semester_id = s.id
			      WHERE s.class_id = ?
			  );
		`, newStatus, eventID, u.ActiveClassID.Int64)
	}

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status partisipasi")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Partisipasi kelas tidak ditemukan pada kejadian ini")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"event_id":             eventID,
		"participation_status": newStatus,
	})
}

func conflictMessage(conflicts []schedule.Conflict) string {
	for _, c := range conflicts {
		if c.Blocking {
			return fmt.Sprintf("%s: %s", c.Code, c.Message)
		}
	}
	if len(conflicts) > 0 {
		return fmt.Sprintf("%s: %s", conflicts[0].Code, conflicts[0].Message)
	}
	return "Jadwal bentrok dengan sesi lain"
}

func conflictMaps(conflicts []schedule.Conflict) []map[string]any {
	out := []map[string]any{}
	for _, c := range conflicts {
		out = append(out, map[string]any{
			"code":        c.Code,
			"type":        c.Code,
			"message":     c.Message,
			"blocking":    c.Blocking,
			"entity_type": c.EntityType,
			"entity_id":   c.EntityID,
			"starts_at":   c.StartsAt,
			"ends_at":     c.EndsAt,
		})
	}
	return out
}

func lecturersForOfferingCtx(ctx context.Context, db *sql.DB, offeringID int64) []int64 {
	rows, err := db.QueryContext(ctx, `SELECT lecturer_id FROM offering_lecturers WHERE course_offering_id=? AND superseded_at IS NULL`, offeringID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func participantOfferingIDs(ctx context.Context, db *sql.DB, eventID, ownerOfferingID int64) []int64 {
	rows, err := db.QueryContext(ctx, `SELECT course_offering_id FROM teaching_event_offerings
		WHERE teaching_event_id=? AND participation_role='PARTICIPANT' AND course_offering_id != ?`, eventID, ownerOfferingID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// DeletePattern menangani DELETE /api/v1/schedule/patterns/{id}
// Hanya menghapus pola aktif (effective_until IS NULL). Versi lama (effective_until SET) tidak dihapus.
func (c *ScheduleController) DeletePattern(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "PJ" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM, PJ, atau System Admin yang berwenang menghapus pola jadwal")
		return
	}

	patternIDStr := r.PathValue("id")
	patternID, err := strconv.ParseInt(patternIDStr, 10, 64)
	if err != nil || patternID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID pola jadwal tidak valid")
		return
	}

	// Ambil version untuk optimistic locking
	var curVersion int
	var curOfferingID int64
	var curClassID int64
	var semesterStatus string
	err = c.db.QueryRow(`
		SELECT sp.version, sp.course_offering_id, s.class_id, s.status
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN semesters s ON co.semester_id = s.id
		WHERE sp.id = ? AND sp.effective_until IS NULL;
	`, patternID).Scan(&curVersion, &curOfferingID, &curClassID, &semesterStatus)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Pola jadwal aktif tidak ditemukan")
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca pola jadwal")
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != curOfferingID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat menghapus pola untuk offering penugasan Anda")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat menghapus pola untuk kelas penugasan Anda")
			return
		}
	}

	version := r.URL.Query().Get("version")
	if version == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib disertakan sebagai query parameter")
		return
	}
	ver, err := strconv.Atoi(version)
	if err != nil || ver <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version tidak valid")
		return
	}
	if ver != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
		})
		return
	}

	// DRAFT: hapus baris sungguhan. ACTIVE: soft delete agar riwayat portal tetap utuh.
	if semesterStatus == "DRAFT" {
		tx, err := c.db.Begin()
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
			return
		}
		defer tx.Rollback()
		res, err := tx.Exec(`DELETE FROM schedule_patterns WHERE id = ? AND version = ? AND effective_until IS NULL;`, patternID, curVersion)
		if err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "foreign key") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
				common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Pola dipakai perubahan jadwal draf. Hapus atau ubah event terkait dulu sebelum menghapus pola ini.")
				return
			}
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghapus pola jadwal")
			return
		}
		affected, _ := res.RowsAffected()
		if affected == 0 {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{"current_version": curVersion})
			return
		}
		correlationID := fmt.Sprintf("delete-pattern-draft-%d-%d", patternID, time.Now().UnixNano())
		{
			uid := u.UserID
			var raid *int64
			if u.ActiveAssignmentID != 0 {
				v := u.ActiveAssignmentID
				raid = &v
			}
			beforeJSON := fmt.Sprintf(`{"id":%d,"version":%d}`, patternID, curVersion)
			if err := audit.Write(r.Context(), tx, audit.Entry{
				Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
				ClassID:       &curClassID,
				Action:        "DELETE_PATTERN",
				EntityType:    "SCHEDULE_PATTERN",
				EntityID:      &patternID,
				BeforeJSON:    &beforeJSON,
				CorrelationID: correlationID,
			}); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit pola jadwal")
				return
			}
		}
		if err := tx.Commit(); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit hapus pola")
			return
		}
		common.WriteV1Success(w, http.StatusOK, map[string]any{
			"id": patternID, "deleted": true,
		})
		return
	}

	// Hapus (set effective_until = kemarin) — soft delete, riwayat tetap untuk audit
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err = c.db.Exec(`
		UPDATE schedule_patterns
		SET effective_until = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND effective_until IS NULL AND version = ?;
	`, yesterday, patternID, curVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghapus pola jadwal")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id": patternID, "deleted": true, "effective_until": yesterday,
	})
}

// DeleteTeachingEvent menangani DELETE /api/v1/teaching-events/{id}
// Hanya boleh menghapus event berstatus DRAFT.
func (c *ScheduleController) DeleteTeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" && u.ActiveRole != "PJ" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM, PJ, atau System Admin yang berwenang menghapus draf perubahan jadwal")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var curStatus string
	var curVersion int
	var curOfferingID int64
	var curClassID int64
	err = c.db.QueryRow(`
		SELECT te.lifecycle_status, te.version, co.id, s.class_id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters s ON co.semester_id = s.id
		WHERE te.id = ?;
	`, eventID).Scan(&curStatus, &curVersion, &curOfferingID, &curClassID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kejadian jadwal tidak ditemukan")
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca kejadian jadwal")
		return
	}

	if curStatus != "DRAFT" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Hanya DRAFT yang dapat dihapus. Event terbit gunakan Revoke.")
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != curOfferingID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat menghapus draf untuk offering penugasan Anda")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya dapat menghapus draf untuk kelas penugasan Anda")
			return
		}
	}

	version := r.URL.Query().Get("version")
	if version == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib disertakan sebagai query parameter")
		return
	}
	ver, err := strconv.Atoi(version)
	if err != nil || ver <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version tidak valid")
		return
	}
	if ver != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
		})
		return
	}

	// Hard delete DRAFT — hapus relasi anak dulu (FK tanpa CASCADE)
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM room_confirmations WHERE teaching_event_id = ?;`, eventID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghapus konfirmasi ruangan event")
		return
	}

	_, err = tx.Exec(`DELETE FROM teaching_event_offerings WHERE teaching_event_id = ?;`, eventID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghapus penawaran event")
		return
	}

	_, err = tx.Exec(`DELETE FROM teaching_events WHERE id = ? AND version = ? AND lifecycle_status = 'DRAFT';`, eventID, curVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghapus draf perubahan jadwal")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi hapus")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id": eventID, "deleted": true,
	})
}
