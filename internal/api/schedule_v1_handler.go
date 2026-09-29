package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleGetV1Patterns menangani GET /api/v1/schedule/patterns
func (s *Server) handleGetV1Patterns(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	offeringParam := r.URL.Query().Get("offering_id")
	dayParam := r.URL.Query().Get("day")

	query := `
		SELECT sp.id, sp.course_offering_id, co.display_name, sp.room_id, COALESCE(r.code, ''),
		       sp.day_of_week, sp.start_time, sp.end_time, sp.status, sp.version
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		WHERE sp.status = 'ACTIVE'
	`
	var args []any

	if u.ActiveRole == "PJ" && u.ActiveCourseOfferingID.Valid {
		query += " AND sp.course_offering_id = ?"
		args = append(args, u.ActiveCourseOfferingID.Int64)
	} else if u.ActiveRole == "KM" && u.ActiveClassID.Valid {
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

	rows, err := s.v1DB.Query(query, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat pola jadwal")
		return
	}
	defer rows.Close()

	var patterns []map[string]any
	for rows.Next() {
		var id, offID int64
		var offName, roomCode, startTime, endTime, status string
		var roomID sql.NullInt64
		var dayOfWeek, version int

		if err := rows.Scan(&id, &offID, &offName, &roomID, &roomCode, &dayOfWeek, &startTime, &endTime, &status, &version); err == nil {
			patterns = append(patterns, map[string]any{
				"id":                 id,
				"course_offering_id": offID,
				"offering":           offName,
				"room_id": func() any {
					if roomID.Valid {
						return roomID.Int64
					}
					return nil
				}(),
				"room":        roomCode,
				"day_of_week": dayOfWeek,
				"start_time":  startTime,
				"end_time":    endTime,
				"status":      status,
				"version":     version,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, patterns)
}

// CreatePatternRequest adalah payload pembuatan pola jadwal baru
type CreatePatternRequest struct {
	OfferingID  int64  `json:"offering_id"`
	DayOfWeek   int    `json:"day_of_week"` // 1-7
	StartTime   string `json:"start_time"`  // "HH:MM"
	DurationMin int    `json:"duration_min"`
	RoomID      *int64 `json:"room_id,omitempty"`
}

// handleCreateV1Pattern menangani POST /api/v1/schedule/patterns
func (s *Server) handleCreateV1Pattern(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreatePatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	if req.DayOfWeek < 1 || req.DayOfWeek > 7 || req.DurationMin <= 0 || strings.TrimSpace(req.StartTime) == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "day_of_week (1-7), start_time (HH:MM), dan duration_min wajib valid")
		return
	}

	// Server menghitung end_time
	startT, err := time.Parse("15:04", req.StartTime)
	if err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Format start_time harus HH:MM")
		return
	}
	endT := startT.Add(time.Duration(req.DurationMin) * time.Minute)
	endTime := endT.Format("15:04")

	var patternID int64
	err = s.v1DB.QueryRow(`
		INSERT INTO schedule_patterns (
			course_offering_id, room_id, day_of_week, start_time, end_time, status, version
		)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', 1)
		RETURNING id;
	`, req.OfferingID, req.RoomID, req.DayOfWeek, req.StartTime, endTime).Scan(&patternID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan pola jadwal: %v", err))
		return
	}

	// Catat audit
	_, _ = s.v1DB.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id)
		VALUES (?, ?, ?, 'CREATE_PATTERN', 'SCHEDULE_PATTERN', ?);
	`, u.ActiveClassID, u.UserID, u.ActiveAssignmentID, patternID)

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"id":         patternID,
		"start_time": req.StartTime,
		"end_time":   endTime,
		"version":    1,
	})
}

// CreateTeachingEventRequest adalah payload kejadian perkuliahan
type CreateTeachingEventRequest struct {
	OwnerOfferingID int64   `json:"owner_offering_id"`
	EventKind       string  `json:"event_kind"` // REPLACEMENT, EXTRA, HOLIDAY, SESSION_CANCELLED
	StartsAt        string  `json:"starts_at"`  // UTC RFC3339
	EndsAt          string  `json:"ends_at"`    // UTC RFC3339
	OriginPatternID *int64  `json:"origin_pattern_id,omitempty"`
	OriginDate      *string `json:"origin_date,omitempty"`
	RoomID          *int64  `json:"room_id,omitempty"`
	Reason          *string `json:"reason,omitempty"`
}

// handleCreateV1TeachingEvent menangani POST /api/v1/teaching-events
func (s *Server) handleCreateV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreateTeachingEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	kind := strings.ToUpper(strings.TrimSpace(req.EventKind))
	if kind != "REPLACEMENT" && kind != "EXTRA" && kind != "HOLIDAY" && kind != "SESSION_CANCELLED" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "event_kind harus REPLACEMENT, EXTRA, HOLIDAY, atau SESSION_CANCELLED")
		return
	}

	// Aturan: REPLACEMENT dan SESSION_CANCELLED wajib memiliki origin_pattern_id dan origin_date
	if (kind == "REPLACEMENT" || kind == "SESSION_CANCELLED") && (req.OriginPatternID == nil || req.OriginDate == nil) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Kejadian pengganti atau pembatalan sesi wajib menyertakan origin_pattern_id dan origin_date")
		return
	}

	startsAt, errStart := time.Parse(time.RFC3339, req.StartsAt)
	endsAt, errEnd := time.Parse(time.RFC3339, req.EndsAt)
	if errStart != nil || errEnd != nil || !endsAt.After(startsAt) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Waktu starts_at dan ends_at harus berupa RFC3339 UTC dan ends_at harus setelah starts_at")
		return
	}

	// Validasi wewenang PJ: hanya boleh membuat teaching event untuk offering yang dipegang
	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != req.OwnerOfferingID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "PJ hanya diizinkan membuat jadwal untuk mata kuliah miliknya")
			return
		}
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var eventID int64
	err = tx.QueryRow(`
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status, version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'DRAFT', 1)
		RETURNING id;
	`, req.OriginPatternID, req.OriginDate, kind, startsAt, endsAt, req.RoomID, req.Reason).Scan(&eventID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan teaching event: %v", err))
		return
	}

	// Hubungkan Offering Pemilik (OWNER)
	_, err = tx.Exec(`
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (?, ?, 'OWNER', 'ACCEPTED');
	`, eventID, req.OwnerOfferingID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menghubungkan offering pemilik event")
		return
	}

	// Catat audit_logs
	_, _ = tx.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, role_assignment_id, action, entity_type, entity_id, after_state)
		VALUES (?, ?, ?, 'CREATE_TEACHING_EVENT', 'TEACHING_EVENT', ?, ?);
	`, u.ActiveClassID, u.UserID, u.ActiveAssignmentID, eventID, fmt.Sprintf(`{"kind":%q,"offering_id":%d}`, kind, req.OwnerOfferingID))

	_ = tx.Commit()

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"id":               eventID,
		"event_kind":       kind,
		"lifecycle_status": "DRAFT",
		"version":          1,
	})
}

// handleGetV1TeachingEvents menangani GET /api/v1/teaching-events
func (s *Server) handleGetV1TeachingEvents(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))

	query := `
		SELECT te.id, te.event_kind, co.id, co.display_name, te.starts_at, te.ends_at,
		       COALESCE(r.code, ''), COALESCE(te.reason, ''), te.lifecycle_status, te.version
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE 1=1
	`
	var args []any

	if u.ActiveRole == "PJ" && u.ActiveCourseOfferingID.Valid {
		query += " AND co.id = ?"
		args = append(args, u.ActiveCourseOfferingID.Int64)
	} else if u.ActiveRole == "KM" && u.ActiveClassID.Valid {
		query += " AND sem.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	}

	if statusFilter != "" {
		query += " AND te.lifecycle_status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY te.starts_at DESC;"

	rows, err := s.v1DB.Query(query, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kejadian perkuliahan")
		return
	}
	defer rows.Close()

	var events []map[string]any
	for rows.Next() {
		var id, offID int64
		var kind, offDisplay, roomCode, reason, lifeStatus string
		var startsAt, endsAt time.Time
		var version int

		if err := rows.Scan(&id, &kind, &offID, &offDisplay, &startsAt, &endsAt, &roomCode, &reason, &lifeStatus, &version); err == nil {
			events = append(events, map[string]any{
				"id":               id,
				"event_kind":       kind,
				"offering_id":      offID,
				"offering":         offDisplay,
				"starts_at":        startsAt.Format(time.RFC3339),
				"ends_at":          endsAt.Format(time.RFC3339),
				"room":             roomCode,
				"reason":           reason,
				"lifecycle_status": lifeStatus,
				"version":          version,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, events)
}

type PublishEventRequest struct {
	Version                int     `json:"version"`
	ConflictOverrideReason *string `json:"conflict_override_reason,omitempty"`
}

// handlePublishV1TeachingEvent menangani POST /api/v1/teaching-events/{id}/publish
func (s *Server) handlePublishV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req PublishEventRequest
	if r.Body != nil && r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
			s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Payload JSON tidak valid")
			return
		}
	}

	var (
		curVersion int
		curStatus  string
		offeringID int64
		classID    int64
		roomID     sql.NullInt64
		startsAt   dbTimestamp
		endsAt     dbTimestamp
		offName    string
		reason     sql.NullString
		roomCode   sql.NullString
	)

	err = s.v1DB.QueryRow(`
		SELECT te.version, te.lifecycle_status, te.room_id, te.starts_at, te.ends_at, te.reason,
		       co.id, sem.class_id, co.display_name, r.code
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE te.id = ?;
	`, eventID).Scan(&curVersion, &curStatus, &roomID, &startsAt, &endsAt, &reason,
		&offeringID, &classID, &offName, &roomCode)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kejadian jadwal tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal membaca kejadian jadwal: %v", err))
		return
	}

	// Cek scope
	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offeringID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya dapat mempublikasikan untuk kelas Anda")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya dapat mempublikasikan untuk kelas Anda")
			return
		}
	} else {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Peran Anda tidak diizinkan")
		return
	}

	// Idempotency check: if already published, return success
	if curStatus == "PUBLISHED" {
		s.writeV1Success(w, http.StatusOK, map[string]any{
			"id":               eventID,
			"lifecycle_status": "PUBLISHED",
		})
		return
	}

	if curStatus != "DRAFT" {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Hanya DRAFT yang dapat dipublikasikan")
		return
	}

	if req.Version != 0 && req.Version != curVersion {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"lifecycle_status": curStatus,
			},
		})
		return
	}

	// Conflict check
	if roomID.Valid {
		var conflictCount int
		_ = s.v1DB.QueryRow(`
			SELECT COUNT(*)
			FROM teaching_events te
			WHERE te.id != ? AND te.room_id = ?
			  AND te.lifecycle_status = 'PUBLISHED'
			  AND te.starts_at < ? AND te.ends_at > ?;
		`, eventID, roomID.Int64, endsAt.Time.Format(time.RFC3339), startsAt.Time.Format(time.RFC3339)).Scan(&conflictCount)

		if conflictCount > 0 && req.ConflictOverrideReason == nil {
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Ruangan sudah digunakan oleh jadwal lain pada jam tersebut (blocking conflict)")
			return
		}
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE teaching_events
		SET lifecycle_status = 'PUBLISHED', published_by_user_id = ?, published_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = ? AND version = ?;
	`, u.UserID, eventID, curVersion)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menerbitkan jadwal: %v", err))
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Konflik versi saat menyimpan")
		return
	}

	// Audit log
	afterJSON := fmt.Sprintf(`{"lifecycle_status":"PUBLISHED","room_id":%v,"version":%d}`, func() any {
		if roomID.Valid {
			return roomID.Int64
		}
		return "null"
	}(), curVersion+1)

	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = fmt.Sprintf("pub-event-%d-%d", eventID, time.Now().UnixNano())
	}

	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			actor_type, actor_user_id, actor_role_assignment_id, class_id, semester_id,
			action, entity_type, entity_id, after_json, correlation_id
		) VALUES (
			'USER', ?, ?, ?, (SELECT semester_id FROM course_offerings WHERE id = ?),
			'PUBLISH_TEACHING_EVENT', 'TEACHING_EVENT', ?, ?, ?
		);
	`, u.UserID, u.ActiveAssignmentID, classID, offeringID, eventID, afterJSON, correlationID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mencatat audit log: %v", err))
		return
	}

	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi")
		return
	}

	// Queue notification
	rc := "-"
	if roomCode.Valid {
		rc = roomCode.String
	}
	rs := ""
	if reason.Valid {
		rs = reason.String
	}
	s.queueNotification(classID, "SCHEDULE_REPLACEMENT", "TEACHING_EVENT", eventID, map[string]any{
		"course":    offName,
		"starts_at": startsAt.Time.Format(time.RFC3339),
		"room":      rc,
		"reason":    rs,
	}, u.UserID)

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"id":               eventID,
		"lifecycle_status": "PUBLISHED",
	})
}

// RevokeEventRequest adalah payload pencabutan jadwal oleh KM
type RevokeEventRequest struct {
	Reason  string `json:"reason"`
	Version int    `json:"version"`
}

// handleRevokeV1TeachingEvent menangani POST /api/v1/teaching-events/{id}/revoke
func (s *Server) handleRevokeV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang membatalkan/mencabut jadwal")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req RevokeEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Alasan pencabutan (reason) wajib diisi")
		return
	}

	var (
		curVersion int
		curStatus  string
		classID    int64
		offName    string
		offeringID int64
	)
	err = s.v1DB.QueryRow(`
		SELECT te.version, te.lifecycle_status, sem.class_id, co.display_name, co.id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE te.id = ?;
	`, eventID).Scan(&curVersion, &curStatus, &classID, &offName, &offeringID)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kejadian jadwal tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membaca kejadian jadwal")
		return
	}

	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya dapat mencabut jadwal untuk kelas Anda")
			return
		}
	}

	if req.Version != curVersion {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Versi data tidak cocok", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"lifecycle_status": curStatus,
			},
		})
		return
	}

	if curStatus != "PUBLISHED" {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Hanya jadwal yang sudah diterbitkan yang dapat dicabut")
		return
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
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
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencabut jadwal")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Konflik versi saat menyimpan")
		return
	}

	beforeJSON := fmt.Sprintf(`{"lifecycle_status":"%s"}`, curStatus)
	afterJSON := fmt.Sprintf(`{"lifecycle_status":"REVOKED","revocation_reason":%q,"version":%d}`, req.Reason, curVersion+1)

	_, err = tx.Exec(`
		INSERT INTO audit_logs (
			actor_type, actor_user_id, actor_role_assignment_id, class_id, semester_id,
			action, entity_type, entity_id, before_json, after_json, reason, correlation_id
		) VALUES (
			'USER', ?, ?, ?, (SELECT semester_id FROM course_offerings WHERE id = ?),
			'REVOKE_TEACHING_EVENT', 'TEACHING_EVENT', ?, ?, ?, ?, ?
		);
	`, u.UserID, u.ActiveAssignmentID, classID, offeringID, eventID, beforeJSON, afterJSON, req.Reason, r.Header.Get("X-Correlation-ID"))
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit log")
		return
	}

	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit transaksi")
		return
	}

	s.queueNotification(classID, "SCHEDULE_REVOKED", "TEACHING_EVENT", eventID, map[string]any{
		"course": offName,
		"reason": req.Reason,
	}, u.UserID)

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"id":               eventID,
		"lifecycle_status": "REVOKED",
	})
}

// PatchPatternRequest adalah payload pembaruan pola jadwal
type PatchPatternRequest struct {
	OfferingID  *int64  `json:"offering_id,omitempty"`
	DayOfWeek   int     `json:"day_of_week"`
	StartTime   string  `json:"start_time"`
	DurationMin int     `json:"duration_min"`
	RoomID      *int64  `json:"room_id,omitempty"`
	LecturerIDs []int64 `json:"lecturer_ids,omitempty"`
	Version     int     `json:"version"`
}

// handlePatchV1Pattern menangani PATCH /api/v1/schedule/patterns/{id}
func (s *Server) handlePatchV1Pattern(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	patternIDStr := r.PathValue("id")
	patternID, err := strconv.ParseInt(patternIDStr, 10, 64)
	if err != nil || patternID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID pola jadwal tidak valid")
		return
	}

	var req PatchPatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Payload JSON tidak valid")
		return
	}

	var (
		curOfferingID int64
		curRoomID     sql.NullInt64
		curDayOfWeek  int
		curStartTime  string
		curEndTime    string
		curClassID    int64
	)

	err = s.v1DB.QueryRow(`
		SELECT sp.course_offering_id, sp.room_id, sp.day_of_week, sp.start_time, sp.end_time, s.class_id
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN semesters s ON co.semester_id = s.id
		WHERE sp.id = ? AND sp.effective_until IS NULL;
	`, patternID).Scan(&curOfferingID, &curRoomID, &curDayOfWeek, &curStartTime, &curEndTime, &curClassID)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Pola jadwal tidak ditemukan atau sudah tidak aktif")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi pola jadwal")
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != curOfferingID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "PJ hanya diizinkan memperbarui pola jadwal untuk mata kuliah miliknya")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya diizinkan memperbarui pola jadwal di kelasnya")
			return
		}
	}

	// Hitung jam selesai baru jika durasi dikirim
	newStartTime := curStartTime
	if strings.TrimSpace(req.StartTime) != "" {
		newStartTime = strings.TrimSpace(req.StartTime)
	}
	newEndTime := curEndTime
	if req.DurationMin > 0 {
		st, err := time.Parse("15:04", newStartTime)
		if err == nil {
			newEndTime = st.Add(time.Duration(req.DurationMin) * time.Minute).Format("15:04")
		}
	}
	newDayOfWeek := curDayOfWeek
	if req.DayOfWeek >= 1 && req.DayOfWeek <= 7 {
		newDayOfWeek = req.DayOfWeek
	}
	newRoomID := curRoomID
	if req.RoomID != nil {
		newRoomID = sql.NullInt64{Int64: *req.RoomID, Valid: true}
	}

	// Tutup pola lama dengan effective_until hari ini
	todayStr := time.Now().Format("2006-01-02")
	_, err = s.v1DB.Exec(`
		UPDATE schedule_patterns
		SET effective_until = ?
		WHERE id = ?;
	`, todayStr, patternID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui pola jadwal lama")
		return
	}

	// Insert pola baru
	var newPatternID int64
	err = s.v1DB.QueryRow(`
		INSERT INTO schedule_patterns (
			course_offering_id, room_id, day_of_week, start_time, end_time, effective_from
		) VALUES (?, ?, ?, ?, ?, ?) RETURNING id;
	`, curOfferingID, newRoomID, newDayOfWeek, newStartTime, newEndTime, todayStr).Scan(&newPatternID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan pola jadwal baru")
		return
	}

	// Audit log
	_, _ = s.v1DB.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, role_assignment_id, action, entity_type, entity_id, before_state, after_state)
		VALUES (?, ?, ?, 'UPDATE_PATTERN', 'SCHEDULE_PATTERN', ?, ?, ?);
	`, curClassID, u.UserID, u.ActiveAssignmentID, newPatternID,
		fmt.Sprintf(`{"old_id":%d,"day":%d,"start":%q}`, patternID, curDayOfWeek, curStartTime),
		fmt.Sprintf(`{"new_id":%d,"day":%d,"start":%q}`, newPatternID, newDayOfWeek, newStartTime),
	)

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"id":             newPatternID,
		"day_of_week":    newDayOfWeek,
		"start_time":     newStartTime,
		"end_time":       newEndTime,
		"effective_from": todayStr,
	})
}

// handlePreviewV1TeachingEvent menangani POST /api/v1/teaching-events/{id}/preview
func (s *Server) handlePreviewV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	_, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var (
		kind            string
		startsAt        dbTimestamp
		endsAt          dbTimestamp
		roomID          sql.NullInt64
		roomCode        sql.NullString
		offName         string
		originPatternID sql.NullInt64
	)

	err = s.v1DB.QueryRow(`
		SELECT te.event_kind, te.starts_at, te.ends_at, te.room_id, r.code, co.display_name, te.origin_schedule_pattern_id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE te.id = ?;
	`, eventID).Scan(&kind, &startsAt, &endsAt, &roomID, &roomCode, &offName, &originPatternID)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kejadian tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kejadian jadwal")
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
		err = s.v1DB.QueryRow(`
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

	conflicts := []map[string]any{}
	if roomID.Valid {
		var conflictCount int
		_ = s.v1DB.QueryRow(`
			SELECT COUNT(*)
			FROM teaching_events te
			WHERE te.id != ? AND te.room_id = ?
			  AND te.lifecycle_status = 'PUBLISHED'
			  AND te.starts_at < ? AND te.ends_at > ?;
		`, eventID, roomID.Int64, endsAt.Time.Format(time.RFC3339), startsAt.Time.Format(time.RFC3339)).Scan(&conflictCount)

		if conflictCount > 0 {
			conflicts = append(conflicts, map[string]any{
				"type":     "ROOM_OCCUPIED",
				"message":  fmt.Sprintf("Ruangan %s sudah digunakan oleh jadwal lain pada jam tersebut", roomCode.String),
				"blocking": true,
			})
		}
	}

	// Cek potensi konflik pattern (same time slot - for students)
	var patternConflictCount int
	_ = s.v1DB.QueryRow(`
		SELECT COUNT(*)
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id
		WHERE te.id != ? AND te.lifecycle_status = 'PUBLISHED'
		  AND te.starts_at < ? AND te.ends_at > ?
		  AND teo.course_offering_id IN (
			SELECT course_offering_id FROM teaching_event_offerings WHERE teaching_event_id = ?
		  );
	`, eventID, endsAt.Time.Format(time.RFC3339), startsAt.Time.Format(time.RFC3339), eventID).Scan(&patternConflictCount)
	if patternConflictCount > 0 {
		conflicts = append(conflicts, map[string]any{
			"type":     "PATTERN_CONFLICT",
			"message":  "Terdapat konflik jadwal dengan kelas lain untuk mahasiswa yang sama",
			"blocking": false,
		})
	}

	roomNote := ""
	if roomID.Valid {
		roomNote = "perlu konfirmasi TU"
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"old": oldData,
		"new": map[string]any{
			"offering":  offName,
			"starts_at": startsAt.Time.Format(time.RFC3339),
			"ends_at":   endsAt.Time.Format(time.RFC3339),
			"room":      roomCode.String,
		},
		"kind":      kind,
		"conflicts": conflicts,
		"room_note": roomNote,
	})
}

// ParticipationRequest merepresentasikan payload persetujuan keikutsertaan kelas gabungan
type ParticipationRequest struct {
	Action string `json:"action"` // accept, decline, leave
}

// handleParticipationV1TeachingEvent menangani POST /api/v1/teaching-events/{id}/participation
func (s *Server) handleParticipationV1TeachingEvent(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req ParticipationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Payload JSON tidak valid")
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
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Aksi partisipasi harus salah satu dari: accept, decline, leave")
		return
	}

	// Update partisipasi untuk offering yang terdaftar di event ini
	res, err := s.v1DB.Exec(`
		UPDATE teaching_event_offerings
		SET participation_status = ?
		WHERE teaching_event_id = ?
		  AND participation_role = 'PARTICIPANT'
		  AND course_offering_id IN (
		      SELECT co.id FROM course_offerings co
		      JOIN semesters s ON co.semester_id = s.id
		      WHERE s.class_id = ?
		  );
	`, newStatus, eventID, u.ActiveClassID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status partisipasi")
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Partisipasi kelas tidak ditemukan pada kejadian ini")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"event_id":             eventID,
		"participation_status": newStatus,
	})
}
