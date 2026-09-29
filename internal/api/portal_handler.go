package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// portalAccessAllowed memeriksa mode portal dan memvalidasi token jika mode CODE.
func (s *Server) portalAccessAllowed(w http.ResponseWriter, r *http.Request, classID int64) bool {
	var mode string
	err := s.v1DB.QueryRow(`
		SELECT portal_access_mode
		FROM class_settings
		WHERE class_id = ?;
	`, classID).Scan(&mode)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat pengaturan kelas")
		return false
	}

	if mode == "LINK" {
		return true
	}

	token := r.Header.Get("X-Portal-Token")
	if token == "" {
		token = r.URL.Query().Get("portal_token")
	}

	if token == "" {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Kelas ini memerlukan kode akses portal")
		return false
	}

	if s.portalService == nil || s.portalService.ValidateSession(r.Context(), classID, token) != nil {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi portal tidak valid atau telah kedaluwarsa")
		return false
	}

	return true
}

// handlePortalSummary menangani GET /api/v1/portal/{slug}/summary
func (s *Server) handlePortalSummary(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := s.v1DB.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	dateParam := r.URL.Query().Get("date")
	var targetDate time.Time
	if dateParam != "" {
		t, err := time.ParseInLocation("2006-01-02", dateParam, loc)
		if err == nil {
			targetDate = t
		}
	}
	if targetDate.IsZero() {
		targetDate = time.Now().In(loc)
	}
	dateStr := targetDate.Format("2006-01-02")
	dayOfWeek := int(targetDate.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7 // Minggu = 7
	}

	// 1. Ambil jadwal hari ini (Patterns + Events PUBLISHED)
	scheduleItems, _ := s.getScheduleForDate(classID, targetDate, dayOfWeek)

	// 2. Cari now_event dan next_event berdasarkan waktu saat ini
	nowTimeStr := targetDate.Format("15:04")
	var nowEvent any
	var nextEvent any

	for _, item := range scheduleItems {
		start := item["starts_at"].(string)
		end := item["ends_at"].(string)

		if nowTimeStr >= start && nowTimeStr <= end {
			nowEvent = item
		} else if nowTimeStr < start && nextEvent == nil {
			nextEvent = item
		}
	}

	// 3. Ambil perubahan hari ini
	var changesToday []map[string]any
	for _, item := range scheduleItems {
		if item["kind"] != "REGULER" {
			changesToday = append(changesToday, item)
		}
	}

	// 4. Ambil tugas terdekat (hingga 3 tugas terdekat)
	var nearestTasks []map[string]any
	taskRows, err := s.v1DB.Query(`
		SELECT t.id, co.display_name, t.title, t.deadline_at, t.submission_url
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE'
		  AND t.publication_status = 'PUBLISHED'
		  AND t.deleted_at IS NULL
		  AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC
		LIMIT 3;
	`, classID)

	if err == nil {
		defer taskRows.Close()
		for taskRows.Next() {
			var tID int64
			var offering, title string
			var deadlineAt dbTimestamp
			var subURL sql.NullString
			if err := taskRows.Scan(&tID, &offering, &title, &deadlineAt, &subURL); err == nil {
				nearestTasks = append(nearestTasks, map[string]any{
					"id":             tID,
					"offering":       offering,
					"title":          title,
					"deadline_at":    deadlineAt.RFC3339(),
					"submission_url": subURL.String,
				})
			}
		}
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"class":         slug,
		"date":          dateStr,
		"now_event":     nowEvent,
		"next_event":    nextEvent,
		"today":         scheduleItems,
		"changes_today": changesToday,
		"nearest_tasks": nearestTasks,
	})
}

// handlePortalSchedule menangani GET /api/v1/portal/{slug}/schedule
func (s *Server) handlePortalSchedule(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := s.v1DB.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	dateParam := r.URL.Query().Get("date")
	var targetDate time.Time
	if dateParam != "" {
		t, err := time.ParseInLocation("2006-01-02", dateParam, loc)
		if err == nil {
			targetDate = t
		}
	}
	if targetDate.IsZero() {
		targetDate = time.Now().In(loc)
	}
	dateStr := targetDate.Format("2006-01-02")
	dayOfWeek := int(targetDate.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}

	items, err := s.getScheduleForDate(classID, targetDate, dayOfWeek)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat jadwal")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"class": slug,
		"date":  dateStr,
		"items": items,
	})
}

// getScheduleForDate mengumpulkan seluruh jadwal reguler dan kejadian perkuliahan terbit pada tanggal tertentu.
func (s *Server) getScheduleForDate(classID int64, targetDate time.Time, dayOfWeek int) ([]map[string]any, error) {
	dateStr := targetDate.Format("2006-01-02")

	// 1. Pola jadwal reguler aktif
	patternRows, err := s.v1DB.Query(`
		SELECT sp.id, co.id, co.display_name, c.name, co.activity_type,
		       sp.start_time, sp.end_time, COALESCE(r.code, ''), sp.effective_from, sp.effective_until
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE'
		  AND sp.status = 'ACTIVE'
		  AND sp.day_of_week = ?
		  AND (sp.effective_from IS NULL OR sp.effective_from <= ?)
		  AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
		ORDER BY sp.start_time ASC;
	`, classID, dayOfWeek, dateStr, dateStr)

	var items []map[string]any
	if err != nil {
		return nil, err
	}
	defer patternRows.Close()

	for patternRows.Next() {
		var patternID, offID int64
		var offDisplay, courseName, actType, startTime, endTime, roomCode string
		var effFrom, effUntil sql.NullString

		if err := patternRows.Scan(&patternID, &offID, &offDisplay, &courseName, &actType, &startTime, &endTime, &roomCode, &effFrom, &effUntil); err == nil {
			// Dosen pengampu
			lecturers := s.getOfferingLecturers(offID)

			items = append(items, map[string]any{
				"id":            fmt.Sprintf("pat_%d", patternID),
				"kind":          "REGULER",
				"offering":      offDisplay,
				"title":         courseName,
				"activity_type": actType,
				"starts_at":     startTime,
				"ends_at":       endTime,
				"room":          roomCode,
				"lecturers":     lecturers,
				"source": map[string]any{
					"pattern_id": patternID,
				},
			})
		}
	}

	// 2. Kejadian perkuliahan (teaching_events) berstatus PUBLISHED pada tanggal ini
	eventRows, err := s.v1DB.Query(`
		SELECT te.id, te.event_kind, co.id, co.display_name, c.name, co.activity_type,
		       strftime('%H:%M', te.starts_at) as start_time,
		       strftime('%H:%M', te.ends_at) as end_time,
		       COALESCE(r.code, ''), te.origin_schedule_pattern_id
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE'
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND date(te.starts_at) = ?;
	`, classID, dateStr)

	if err == nil {
		defer eventRows.Close()
		for eventRows.Next() {
			var eventID, offID int64
			var eventKind, offDisplay, courseName, actType, startTime, endTime, roomCode string
			var originPatID sql.NullInt64

			if err := eventRows.Scan(&eventID, &eventKind, &offID, &offDisplay, &courseName, &actType, &startTime, &endTime, &roomCode, &originPatID); err == nil {
				kindMap := map[string]string{
					"REPLACEMENT":       "PENGGANTI",
					"EXTRA":             "TAMBAHAN",
					"HOLIDAY":           "LIBUR",
					"SESSION_CANCELLED": "DIBATALKAN",
				}
				kindLabel, ok := kindMap[eventKind]
				if !ok {
					kindLabel = eventKind
				}

				lecturers := s.getOfferingLecturers(offID)

				// Jika pengganti/pembatalan memiliki origin pattern, tandai/gantikan
				items = append(items, map[string]any{
					"id":            fmt.Sprintf("ev_%d", eventID),
					"kind":          kindLabel,
					"offering":      offDisplay,
					"title":         courseName,
					"activity_type": actType,
					"starts_at":     startTime,
					"ends_at":       endTime,
					"room":          roomCode,
					"lecturers":     lecturers,
					"source": map[string]any{
						"event_id":   eventID,
						"pattern_id": originPatID.Int64,
					},
				})
			}
		}
	}

	return items, nil
}

// getOfferingLecturers mengembalikan daftar nama dosen untuk suatu offering.
func (s *Server) getOfferingLecturers(offeringID int64) []string {
	var lecturers []string
	rows, err := s.v1DB.Query(`
		SELECT l.full_name
		FROM offering_lecturers ol
		JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE ol.course_offering_id = ?;
	`, offeringID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				lecturers = append(lecturers, name)
			}
		}
	}
	return lecturers
}

// handlePortalTasks menangani GET /api/v1/portal/{slug}/tasks
func (s *Server) handlePortalTasks(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	var timezone string
	err := s.v1DB.QueryRow(`
		SELECT c.id, cs.timezone
		FROM classes c
		JOIN class_settings cs ON c.id = cs.class_id
		WHERE c.slug = ?;
	`, slug).Scan(&classID, &timezone)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}

	query := `
		SELECT t.id, co.display_name, t.title, t.instructions, t.deadline_at,
		       COALESCE(t.submission_text, ''), COALESCE(t.submission_url, ''), t.version
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE'
		  AND t.publication_status = 'PUBLISHED'
		  AND t.deleted_at IS NULL
	`
	args := []any{classID}

	group := r.URL.Query().Get("group")
	if group != "" && group != "hari_ini" && group != "minggu_ini" && group != "mendatang" && group != "terlewat" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "group harus hari_ini, minggu_ini, mendatang, atau terlewat")
		return
	}
	now := time.Now().In(loc)
	if group == "hari_ini" {
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).UTC().Format("2006-01-02T15:04:05Z")
		end := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc).UTC().Format("2006-01-02T15:04:05Z")
		query += " AND t.deadline_at BETWEEN ? AND ?"
		args = append(args, start, end)
	} else if group == "minggu_ini" {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		startWeek := now.AddDate(0, 0, -weekday+1)
		endWeek := startWeek.AddDate(0, 0, 6)
		start := time.Date(startWeek.Year(), startWeek.Month(), startWeek.Day(), 0, 0, 0, 0, loc).UTC().Format("2006-01-02T15:04:05Z")
		end := time.Date(endWeek.Year(), endWeek.Month(), endWeek.Day(), 23, 59, 59, 999999999, loc).UTC().Format("2006-01-02T15:04:05Z")
		query += " AND t.deadline_at BETWEEN ? AND ?"
		args = append(args, start, end)
	} else if group == "mendatang" {
		query += " AND t.deadline_at > ?"
		args = append(args, now.UTC().Format("2006-01-02T15:04:05Z"))
	} else if group == "terlewat" {
		query += " AND t.deadline_at < ? AND t.completed_at IS NULL"
		args = append(args, now.UTC().Format("2006-01-02T15:04:05Z"))
	}

	offering := r.URL.Query().Get("offering")
	if offering != "" {
		if offID, err := strconv.ParseInt(offering, 10, 64); err == nil {
			query += " AND co.id = ?"
			args = append(args, offID)
		} else {
			query += " AND co.display_name = ?"
			args = append(args, offering)
		}
	}

	q := r.URL.Query().Get("q")
	if q != "" {
		query += " AND (t.title LIKE '%' || ? || '%' OR t.instructions LIKE '%' || ? || '%')"
		args = append(args, q, q)
	}

	query += " ORDER BY t.deadline_at ASC;"

	rows, err := s.v1DB.Query(query, args...)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat tugas portal")
		return
	}
	defer rows.Close()

	var tasks []map[string]any
	for rows.Next() {
		var id int64
		var offering, title, instructions, subText, subURL string
		var deadlineAt dbTimestamp
		var version int

		if err := rows.Scan(&id, &offering, &title, &instructions, &deadlineAt, &subText, &subURL, &version); err == nil {
			tasks = append(tasks, map[string]any{
				"id":              id,
				"offering":        offering,
				"title":           title,
				"instructions":    instructions,
				"deadline_at":     deadlineAt.RFC3339(),
				"submission_text": subText,
				"submission_url":  subURL,
				"version":         version,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, tasks)
}

// handlePortalTaskDetail menangani GET /api/v1/portal/{slug}/tasks/{id}
func (s *Server) handlePortalTaskDetail(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	taskIDStr := r.PathValue("id")
	taskID, _ := strconv.ParseInt(taskIDStr, 10, 64)

	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	var (
		id           int64
		offeringID   int64
		offeringName string
		title        string
		instructions string
		deadlineAt   dbTimestamp
		taskType     sql.NullString
		subText      sql.NullString
		subURL       sql.NullString
		version      int
		completedAt  dbTimestamp
	)

	err = s.v1DB.QueryRow(`
		SELECT t.id, co.id, co.display_name, t.title, t.instructions, t.deadline_at,
		       t.task_type, t.submission_text, t.submission_url, t.version, t.completed_at
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.id = ? AND sem.class_id = ? AND t.publication_status = 'PUBLISHED' AND t.deleted_at IS NULL;
	`, taskID, classID).Scan(
		&id, &offeringID, &offeringName, &title, &instructions, &deadlineAt,
		&taskType, &subText, &subURL, &version, &completedAt,
	)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Tugas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat detail tugas")
		return
	}

	// Ambil materi terkait tugas ini
	var materials []map[string]any
	matRows, err := s.v1DB.Query(`
		SELECT id, title, material_type, url, description
		FROM materials
		WHERE (task_id = ? OR course_offering_id = ?) AND status = 'ACTIVE' AND deleted_at IS NULL;
	`, id, offeringID)
	if err == nil {
		defer matRows.Close()
		for matRows.Next() {
			var mID int64
			var mTitle, mType string
			var mURL, mDesc sql.NullString
			if err := matRows.Scan(&mID, &mTitle, &mType, &mURL, &mDesc); err == nil {
				materials = append(materials, map[string]any{
					"id":            mID,
					"title":         mTitle,
					"material_type": mType,
					"url":           mURL.String,
					"description":   mDesc.String,
				})
			}
		}
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"task": map[string]any{
			"id":              id,
			"offering_id":     offeringID,
			"offering":        offeringName,
			"title":           title,
			"instructions":    instructions,
			"deadline_at":     deadlineAt.RFC3339(),
			"task_type":       taskType.String,
			"submission_text": subText.String,
			"submission_url":  subURL.String,
			"version":         version,
			"is_completed":    completedAt.Valid,
		},
		"materials": materials,
	})
}

// handlePortalChanges menangani GET /api/v1/portal/{slug}/changes
func (s *Server) handlePortalChanges(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	query := `
		SELECT te.id, te.event_kind, co.display_name, te.starts_at, te.ends_at,
		       COALESCE(r.code, ''), COALESCE(te.reason, ''), te.published_at
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE sem.class_id = ? AND te.lifecycle_status = 'PUBLISHED'
	`
	args := []any{classID}

	since := r.URL.Query().Get("since")
	if since != "" {
		if _, err := time.Parse(time.RFC3339, since); err != nil {
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "since harus berformat RFC3339")
			return
		}
		query += " AND te.published_at >= ?"
		args = append(args, since)
	}

	query += " ORDER BY te.published_at DESC LIMIT 50;"

	rows, err := s.v1DB.Query(query, args...)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat riwayat perubahan")
		return
	}
	defer rows.Close()

	var changes []map[string]any
	for rows.Next() {
		var id int64
		var kind, offering, roomCode, reason string
		var startsAt, endsAt dbTimestamp
		var publishedAt dbTimestamp

		if err := rows.Scan(&id, &kind, &offering, &startsAt, &endsAt, &roomCode, &reason, &publishedAt); err == nil {
			changes = append(changes, map[string]any{
				"id":           id,
				"event_kind":   kind,
				"offering":     offering,
				"starts_at":    startsAt.RFC3339(),
				"ends_at":      endsAt.RFC3339(),
				"room":         roomCode,
				"reason":       reason,
				"published_at": publishedAt.RFC3339(),
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, changes)
}

// handlePortalMaterials menangani GET /api/v1/portal/{slug}/materials
func (s *Server) handlePortalMaterials(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	slug := r.PathValue("slug")
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, slug).Scan(&classID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kelas")
		return
	}

	if !s.portalAccessAllowed(w, r, classID) {
		return
	}

	query := `
		SELECT m.id, m.title, m.material_type, COALESCE(m.url, ''), COALESCE(m.description, '')
		FROM materials m
	`
	args := []any{}

	offering := r.URL.Query().Get("offering")
	if offering != "" {
		if offID, err := strconv.ParseInt(offering, 10, 64); err == nil {
			query += " WHERE m.class_id = ? AND m.course_offering_id = ? AND m.status = 'ACTIVE' AND m.deleted_at IS NULL"
			args = append(args, classID, offID)
		} else {
			query += `
				JOIN course_offerings co ON m.course_offering_id = co.id
				WHERE m.class_id = ? AND co.display_name = ? AND m.status = 'ACTIVE' AND m.deleted_at IS NULL
			`
			args = append(args, classID, offering)
		}
	} else {
		query += " WHERE m.class_id = ? AND m.status = 'ACTIVE' AND m.deleted_at IS NULL"
		args = append(args, classID)
	}

	query += " ORDER BY m.created_at DESC;"

	rows, err := s.v1DB.Query(query, args...)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat materi kelas")
		return
	}
	defer rows.Close()

	var materials []map[string]any
	for rows.Next() {
		var id int64
		var title, matType, urlStr, desc string
		if err := rows.Scan(&id, &title, &matType, &urlStr, &desc); err == nil {
			materials = append(materials, map[string]any{
				"id":            id,
				"title":         title,
				"material_type": matType,
				"url":           urlStr,
				"description":   desc,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, materials)
}
