package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type portalClassContext struct {
	classID      int64
	classCode    string
	classSlug    string
	semesterID   int64
	semesterName string
	location     *time.Location
}

func portalNotFound(w http.ResponseWriter, s *Server) {
	s.writeJSON(w, http.StatusNotFound, map[string]string{
		"status": "error",
		"error":  "Portal kelas tidak ditemukan atau belum tersedia",
	})
}

func (s *Server) resolvePortalContext(r *http.Request) (*portalClassContext, int, string) {
	if s.academicRepo == nil {
		return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
	}

	slug := strings.ToLower(strings.TrimSpace(r.PathValue("slug")))
	if slug == "" {
		return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
	}

	ctx := r.Context()
	cls, err := s.academicRepo.GetClassBySlug(ctx, slug)
	if err != nil || cls == nil || cls.Status != "ACTIVE" {
		return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
	}

	var accessMode string
	var loc *time.Location
	loc, err = s.academicRepo.GetClassTimezone(ctx, cls.ID)
	if err != nil || loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	if err := s.academicRepo.DB().QueryRowContext(ctx, `SELECT portal_access_mode FROM class_settings WHERE class_id = ?`, cls.ID).Scan(&accessMode); err != nil {
		accessMode = "LINK"
	}
	if strings.ToUpper(strings.TrimSpace(accessMode)) == "CODE" {
		return nil, http.StatusForbidden, "Portal kelas ini memerlukan kode akses"
	}

	semesterID := int64(0)
	semesterName := ""
	if rawSemester := strings.TrimSpace(r.URL.Query().Get("semester_id")); rawSemester != "" {
		parsed, parseErr := strconv.ParseInt(rawSemester, 10, 64)
		if parseErr != nil || parsed <= 0 {
			return nil, http.StatusBadRequest, "Parameter semester_id tidak valid"
		}
		var status string
		var publishedAt sql.NullString
		var academicYear, term string
		lookupErr := s.academicRepo.DB().QueryRowContext(ctx, `SELECT status, published_at, academic_year, term FROM semesters WHERE id = ? AND class_id = ?`, parsed, cls.ID).Scan(&status, &publishedAt, &academicYear, &term)
		if lookupErr != nil {
			return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
		}
		if status == "DRAFT" || (status == "ARCHIVED" && !publishedAt.Valid) {
			return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
		}
		semesterID = parsed
		semesterName = academicYear + " " + term
	} else {
		var activeID int64
		var academicYear, term string
		activeErr := s.academicRepo.DB().QueryRowContext(ctx, `SELECT id, academic_year, term FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1`, cls.ID).Scan(&activeID, &academicYear, &term)
		if activeErr != nil {
			return nil, http.StatusNotFound, "Portal kelas tidak ditemukan atau belum tersedia"
		}
		semesterID = activeID
		semesterName = academicYear + " " + term
	}

	return &portalClassContext{
		classID:      cls.ID,
		classCode:    cls.Code,
		classSlug:    cls.Slug,
		semesterID:   semesterID,
		semesterName: semesterName,
		location:     loc,
	}, 0, ""
}

func writePortalError(s *Server, w http.ResponseWriter, code int, message string) {
	if code == http.StatusNotFound {
		portalNotFound(w, s)
		return
	}
	s.writeJSON(w, code, map[string]string{
		"status": "error",
		"error":  message,
	})
}

type portalScheduleItem struct {
	MataKuliah string   `json:"mata_kuliah"`
	Jenis      string   `json:"jenis"`
	JamMulai   string   `json:"jam_mulai"`
	JamSelesai string   `json:"jam_selesai"`
	Dosen      []string `json:"dosen"`
	Ruangan    string   `json:"ruangan"`
	Label      string   `json:"label"`
	Keterangan string   `json:"keterangan,omitempty"`
}

func parseStoredTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("timestamp kosong")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("format timestamp tidak didukung")
}

func formatClockIn(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("15.04")
}

func isoWeekday(date time.Time) int {
	weekday := int(date.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}

func portalLecturers(db *sql.DB, r *http.Request, offeringID int64) []string {
	names := []string{}
	rows, err := db.QueryContext(r.Context(), `SELECT l.full_name FROM offering_lecturers ol
	JOIN lecturers l ON l.id = ol.lecturer_id
	WHERE ol.course_offering_id = ?
	ORDER BY CASE ol.responsibility WHEN 'PRIMARY' THEN 0 WHEN 'ASSISTANT' THEN 1 ELSE 2 END, l.full_name ASC`, offeringID)
	if err != nil {
		return names
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil && strings.TrimSpace(name) != "" {
			names = append(names, name)
		}
	}
	return names
}

func portalRoomName(db *sql.DB, r *http.Request, roomID sql.NullInt64) string {
	if !roomID.Valid {
		return ""
	}
	var name string
	if err := db.QueryRowContext(r.Context(), `SELECT name FROM rooms WHERE id = ?`, roomID.Int64).Scan(&name); err != nil {
		return ""
	}
	return name
}

type portalPatternRow struct {
	patternID  int64
	offeringID int64
	courseName string
	activity   string
	startTime  string
	endTime    string
	roomID     sql.NullInt64
}

type portalEventRow struct {
	eventID       int64
	kind          string
	lifecycle     string
	startsAt      string
	endsAt        string
	roomID        sql.NullInt64
	reason        sql.NullString
	originPattern sql.NullInt64
	originDate    sql.NullString
	courseName    string
	activity      string
	offeringID    int64
	publishedAt   sql.NullString
}

func eventLabel(kind, lifecycle string) string {
	if lifecycle == "REVOKED" {
		return "Publikasi Dicabut"
	}
	switch kind {
	case "REPLACEMENT":
		return "Kelas Pengganti"
	case "EXTRA":
		return "Kelas Tambahan"
	case "HOLIDAY":
		return "Hari Libur"
	case "SESSION_CANCELLED":
		return "Sesi Dibatalkan"
	default:
		return "Perubahan Jadwal"
	}
}

func (s *Server) buildEffectiveSchedule(pctx *portalClassContext, r *http.Request, localDate time.Time) ([]portalScheduleItem, bool, error) {
	db := s.academicRepo.DB()
	dayStart := time.Date(localDate.Year(), localDate.Month(), localDate.Day(), 0, 0, 0, 0, pctx.location)
	dayEnd := dayStart.Add(24 * time.Hour)
	windowStart := dayStart.UTC().Format(time.RFC3339Nano)
	windowEnd := dayEnd.UTC().Format(time.RFC3339Nano)
	dateStr := dayStart.Format("2006-01-02")

	patternRows, err := db.QueryContext(r.Context(), `SELECT sp.id, sp.course_offering_id, c.name, co.activity_type,
		sp.start_time, sp.end_time, sp.room_id
	FROM schedule_patterns sp
	JOIN course_offerings co ON co.id = sp.course_offering_id
	JOIN courses c ON c.id = co.course_id
	WHERE co.semester_id = ? AND sp.status = 'ACTIVE' AND sp.day_of_week = ?
	  AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
	ORDER BY sp.start_time ASC, c.name ASC`, pctx.semesterID, isoWeekday(dayStart), dateStr, dateStr)
	if err != nil {
		return nil, false, err
	}
	defer patternRows.Close()

	patterns := []portalPatternRow{}
	for patternRows.Next() {
		var row portalPatternRow
		if err := patternRows.Scan(&row.patternID, &row.offeringID, &row.courseName, &row.activity, &row.startTime, &row.endTime, &row.roomID); err != nil {
			return nil, false, err
		}
		patterns = append(patterns, row)
	}
	if err := patternRows.Err(); err != nil {
		return nil, false, err
	}

	eventRows, err := db.QueryContext(r.Context(), `SELECT te.id, te.event_kind, te.lifecycle_status, te.starts_at, te.ends_at,
		te.room_id, te.reason, te.origin_schedule_pattern_id, te.origin_occurrence_date,
		c.name, co.activity_type, co.id, te.published_at
	FROM teaching_events te
	JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
	JOIN course_offerings co ON co.id = teo.course_offering_id
	JOIN courses c ON c.id = co.course_id
	JOIN semesters sem ON sem.id = co.semester_id
	WHERE sem.class_id = ? AND sem.id = ? AND te.lifecycle_status IN ('PUBLISHED', 'REVOKED')
	  AND te.starts_at < ? AND te.ends_at > ?
	  AND ((teo.participation_role = 'OWNER') OR (teo.participation_role = 'PARTICIPANT' AND teo.participation_status = 'ACCEPTED'))
	ORDER BY te.starts_at ASC, te.id ASC`, pctx.classID, pctx.semesterID, windowEnd, windowStart)
	if err != nil {
		return nil, false, err
	}
	defer eventRows.Close()

	events := []portalEventRow{}
	for eventRows.Next() {
		var row portalEventRow
		if err := eventRows.Scan(&row.eventID, &row.kind, &row.lifecycle, &row.startsAt, &row.endsAt,
			&row.roomID, &row.reason, &row.originPattern, &row.originDate,
			&row.courseName, &row.activity, &row.offeringID, &row.publishedAt); err != nil {
			return nil, false, err
		}
		events = append(events, row)
	}
	if err := eventRows.Err(); err != nil {
		return nil, false, err
	}

	suppressed := map[int64]bool{}
	isHoliday := false
	for _, event := range events {
		if event.lifecycle != "PUBLISHED" {
			continue
		}
		if event.kind == "HOLIDAY" {
			isHoliday = true
		}
		if (event.kind == "REPLACEMENT" || event.kind == "SESSION_CANCELLED") && event.originPattern.Valid && event.originDate.Valid && event.originDate.String == dateStr {
			suppressed[event.originPattern.Int64] = true
		}
	}

	items := []portalScheduleItem{}
	if !isHoliday {
		for _, pattern := range patterns {
			if suppressed[pattern.patternID] {
				continue
			}
			items = append(items, portalScheduleItem{
				MataKuliah: pattern.courseName,
				Jenis:      pattern.activity,
				JamMulai:   strings.ReplaceAll(pattern.startTime, ":", "."),
				JamSelesai: strings.ReplaceAll(pattern.endTime, ":", "."),
				Dosen:      portalLecturers(db, r, pattern.offeringID),
				Ruangan:    portalRoomName(db, r, pattern.roomID),
				Label:      "Reguler",
			})
		}
	}

	for _, event := range events {
		if event.lifecycle != "PUBLISHED" || event.kind == "HOLIDAY" {
			continue
		}
		starts, err := parseStoredTime(event.startsAt)
		if err != nil {
			continue
		}
		ends, err := parseStoredTime(event.endsAt)
		if err != nil {
			continue
		}
		note := ""
		if event.reason.Valid {
			note = event.reason.String
		}
		items = append(items, portalScheduleItem{
			MataKuliah: event.courseName,
			Jenis:      event.activity,
			JamMulai:   formatClockIn(starts, pctx.location),
			JamSelesai: formatClockIn(ends, pctx.location),
			Dosen:      portalLecturers(db, r, event.offeringID),
			Ruangan:    portalRoomName(db, r, event.roomID),
			Label:      eventLabel(event.kind, event.lifecycle),
			Keterangan: note,
		})
	}

	if isHoliday {
		items = append(items, portalScheduleItem{
			MataKuliah: "Libur",
			Label:      "Hari Libur",
			Dosen:      []string{},
		})
	}

	return items, isHoliday, nil
}

func (s *Server) handlePortalSummary(w http.ResponseWriter, r *http.Request) {
	pctx, code, message := s.resolvePortalContext(r)
	if pctx == nil {
		writePortalError(s, w, code, message)
		return
	}

	now := time.Now().In(pctx.location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, pctx.location)
	schedule, _, err := s.buildEffectiveSchedule(pctx, r, today)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat ringkasan kelas"})
		return
	}

	db := s.academicRepo.DB()
	taskRows, err := db.QueryContext(r.Context(), `SELECT t.id, c.name, t.title, t.deadline_at
	FROM tasks t
	JOIN course_offerings co ON co.id = t.course_offering_id
	JOIN courses c ON c.id = co.course_id
	WHERE co.semester_id = ? AND t.publication_status = 'PUBLISHED'
	  AND t.deleted_at IS NULL AND t.archived_at IS NULL AND t.completed_at IS NULL
	ORDER BY t.deadline_at ASC LIMIT 5`, pctx.semesterID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat ringkasan kelas"})
		return
	}
	defer taskRows.Close()
	nearest := []map[string]any{}
	for taskRows.Next() {
		var id int64
		var course, title, deadline string
		if err := taskRows.Scan(&id, &course, &title, &deadline); err != nil {
			continue
		}
		nearest = append(nearest, map[string]any{"id": id, "mata_kuliah": course, "judul": title, "tenggat": deadline})
	}

	changeRows, err := db.QueryContext(r.Context(), `SELECT te.id, te.event_kind, te.lifecycle_status, te.starts_at, c.name
	FROM teaching_events te
	JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
	JOIN course_offerings co ON co.id = teo.course_offering_id
	JOIN courses c ON c.id = co.course_id
	JOIN semesters sem ON sem.id = co.semester_id
	WHERE sem.class_id = ? AND sem.id = ? AND te.lifecycle_status IN ('PUBLISHED', 'REVOKED')
	  AND ((teo.participation_role = 'OWNER') OR (teo.participation_role = 'PARTICIPANT' AND teo.participation_status = 'ACCEPTED'))
	GROUP BY te.id
	ORDER BY te.starts_at DESC LIMIT 5`, pctx.classID, pctx.semesterID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat ringkasan kelas"})
		return
	}
	defer changeRows.Close()
	changes := []map[string]any{}
	for changeRows.Next() {
		var id int64
		var kind, lifecycle, starts, course string
		if err := changeRows.Scan(&id, &kind, &lifecycle, &starts, &course); err != nil {
			continue
		}
		changes = append(changes, map[string]any{"id": id, "label": eventLabel(kind, lifecycle), "mata_kuliah": course, "waktu": starts})
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"kelas":             pctx.classCode,
			"semester":          pctx.semesterName,
			"tanggal":           today.Format("2006-01-02"),
			"jadwal_hari_ini":   schedule,
			"tugas_terdekat":    nearest,
			"perubahan_terbaru": changes,
		},
	})
}

func (s *Server) handlePortalSchedule(w http.ResponseWriter, r *http.Request) {
	pctx, code, message := s.resolvePortalContext(r)
	if pctx == nil {
		writePortalError(s, w, code, message)
		return
	}

	now := time.Now().In(pctx.location)
	localDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, pctx.location)
	if rawDate := strings.TrimSpace(r.URL.Query().Get("date")); rawDate != "" {
		parsed, err := time.ParseInLocation("2006-01-02", rawDate, pctx.location)
		if err != nil {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter date tidak valid, gunakan format YYYY-MM-DD"})
			return
		}
		localDate = parsed
	}

	items, isHoliday, err := s.buildEffectiveSchedule(pctx, r, localDate)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat jadwal kelas"})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"kelas":      pctx.classCode,
			"semester":   pctx.semesterName,
			"tanggal":    localDate.Format("2006-01-02"),
			"hari_libur": isHoliday,
			"jadwal":     items,
		},
	})
}

func (s *Server) handlePortalTasks(w http.ResponseWriter, r *http.Request) {
	pctx, code, message := s.resolvePortalContext(r)
	if pctx == nil {
		writePortalError(s, w, code, message)
		return
	}

	group := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("group")))
	if group == "" {
		group = "all"
	}
	switch group {
	case "all", "today", "week", "upcoming", "overdue":
	default:
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter group tidak valid"})
		return
	}

	now := time.Now().In(pctx.location)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, pctx.location)
	weekday := isoWeekday(todayStart)
	weekStart := todayStart.AddDate(0, 0, -(weekday - 1))
	weekEnd := weekStart.AddDate(0, 0, 7)

	args := []any{pctx.semesterID}
	query := `SELECT t.id, c.code, c.name, co.activity_type, t.title, t.instructions, t.deadline_at,
		t.task_type, t.submission_text, t.submission_url
	FROM tasks t
	JOIN course_offerings co ON co.id = t.course_offering_id
	JOIN courses c ON c.id = co.course_id
	WHERE co.semester_id = ? AND t.publication_status = 'PUBLISHED'
	  AND t.deleted_at IS NULL AND t.archived_at IS NULL AND t.completed_at IS NULL`
	if rawOffering := strings.TrimSpace(r.URL.Query().Get("course_offering_id")); rawOffering != "" {
		offeringID, err := strconv.ParseInt(rawOffering, 10, 64)
		if err != nil || offeringID <= 0 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter course_offering_id tidak valid"})
			return
		}
		query += ` AND t.course_offering_id = ?`
		args = append(args, offeringID)
	}
	query += ` ORDER BY t.deadline_at ASC, t.id ASC`

	rows, err := s.academicRepo.DB().QueryContext(r.Context(), query, args...)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat tugas kelas"})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var courseCode, courseName, activity, title, instructions, deadline, taskType string
		var submissionText, submissionURL sql.NullString
		if err := rows.Scan(&id, &courseCode, &courseName, &activity, &title, &instructions, &deadline, &taskType, &submissionText, &submissionURL); err != nil {
			continue
		}
		deadlineTime, err := parseStoredTime(deadline)
		if err != nil {
			continue
		}
		local := deadlineTime.In(pctx.location)
		bucket := "upcoming"
		switch {
		case !local.Before(todayStart) && local.Before(todayStart.Add(24*time.Hour)):
			bucket = "today"
		case !local.Before(weekStart) && local.Before(weekEnd):
			bucket = "week"
		case local.Before(todayStart):
			bucket = "overdue"
		}
		if group != "all" && bucket != group {
			continue
		}
		submission := ""
		if submissionText.Valid && strings.TrimSpace(submissionText.String) != "" {
			submission = submissionText.String
		} else if submissionURL.Valid {
			submission = submissionURL.String
		}
		items = append(items, map[string]any{
			"id":                 id,
			"mata_kuliah":        courseName,
			"kode_mata_kuliah":   courseCode,
			"jenis_kelas":        activity,
			"judul":              title,
			"instruksi":          instructions,
			"tenggat":            local.Format("2006-01-02 15.04"),
			"zona_waktu":         pctx.location.String(),
			"jenis_tugas":        taskType,
			"tempat_pengumpulan": submission,
			"kelompok":           bucket,
		})
	}
	if err := rows.Err(); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat tugas kelas"})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"kelas":    pctx.classCode,
			"semester": pctx.semesterName,
			"kelompok": group,
			"tugas":    items,
		},
	})
}

func (s *Server) handlePortalChanges(w http.ResponseWriter, r *http.Request) {
	pctx, code, message := s.resolvePortalContext(r)
	if pctx == nil {
		writePortalError(s, w, code, message)
		return
	}

	limit := 20
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 || parsed > 50 {
			s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter limit tidak valid"})
			return
		}
		limit = parsed
	}

	rows, err := s.academicRepo.DB().QueryContext(r.Context(), `SELECT te.id, te.event_kind, te.lifecycle_status,
		te.starts_at, te.ends_at, te.room_id, te.reason, te.published_at,
		c.name, co.activity_type, sp.day_of_week, sp.start_time, sp.end_time
	FROM teaching_events te
	JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
	JOIN course_offerings co ON co.id = teo.course_offering_id
	JOIN courses c ON c.id = co.course_id
	JOIN semesters sem ON sem.id = co.semester_id
	LEFT JOIN schedule_patterns sp ON sp.id = te.origin_schedule_pattern_id
	WHERE sem.class_id = ? AND sem.id = ? AND te.lifecycle_status IN ('PUBLISHED', 'REVOKED')
	  AND ((teo.participation_role = 'OWNER') OR (teo.participation_role = 'PARTICIPANT' AND teo.participation_status = 'ACCEPTED'))
	GROUP BY te.id
	ORDER BY te.starts_at DESC LIMIT ?`, pctx.classID, pctx.semesterID, limit)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat perubahan kelas"})
		return
	}
	defer rows.Close()

	db := s.academicRepo.DB()
	changes := []map[string]any{}
	for rows.Next() {
		var id int64
		var kind, lifecycle, starts, ends string
		var roomID sql.NullInt64
		var reason sql.NullString
		var publishedAt sql.NullString
		var course, activity string
		var originDay sql.NullInt64
		var originStart, originEnd sql.NullString
		if err := rows.Scan(&id, &kind, &lifecycle, &starts, &ends, &roomID, &reason,
			&publishedAt, &course, &activity, &originDay, &originStart, &originEnd); err != nil {
			continue
		}
		startsTime, err := parseStoredTime(starts)
		if err != nil {
			continue
		}
		endsTime, err := parseStoredTime(ends)
		if err != nil {
			continue
		}
		before := ""
		if originDay.Valid && originStart.Valid && originEnd.Valid {
			before = strings.ReplaceAll(originStart.String, ":", ".") + " - " + strings.ReplaceAll(originEnd.String, ":", ".")
		}
		after := formatClockIn(startsTime, pctx.location) + " - " + formatClockIn(endsTime, pctx.location)
		note := ""
		if reason.Valid {
			note = reason.String
		}
		changes = append(changes, map[string]any{
			"id":            id,
			"label":         eventLabel(kind, lifecycle),
			"mata_kuliah":   course,
			"jenis_kelas":   activity,
			"jadwal_semula": before,
			"jadwal_baru":   after,
			"waktu_mulai":   startsTime.In(pctx.location).Format("2006-01-02 15.04"),
			"ruangan":       portalRoomName(db, r, roomID),
			"keterangan":    note,
		})
	}
	if err := rows.Err(); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat perubahan kelas"})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"kelas":     pctx.classCode,
			"semester":  pctx.semesterName,
			"perubahan": changes,
		},
	})
}

func (s *Server) handlePortalSemesters(w http.ResponseWriter, r *http.Request) {
	pctx, code, message := s.resolvePortalContext(r)
	if pctx == nil {
		writePortalError(s, w, code, message)
		return
	}

	rows, err := s.academicRepo.DB().QueryContext(r.Context(), `SELECT id, academic_year, term, starts_on, ends_on, status
	FROM semesters
	WHERE class_id = ? AND published_at IS NOT NULL
	ORDER BY starts_on DESC`, pctx.classID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat arsip semester"})
		return
	}
	defer rows.Close()

	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var academicYear, term, startsOn, endsOn, status string
		if err := rows.Scan(&id, &academicYear, &term, &startsOn, &endsOn, &status); err != nil {
			continue
		}
		items = append(items, map[string]any{
			"id":             id,
			"semester":       academicYear + " " + term,
			"mulai":          startsOn,
			"selesai":        endsOn,
			"status":         status,
			"semester_aktif": id == pctx.semesterID,
		})
	}
	if err := rows.Err(); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memuat arsip semester"})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]any{
		"status": "success",
		"data": map[string]any{
			"kelas":    pctx.classCode,
			"semester": items,
		},
	})
}
