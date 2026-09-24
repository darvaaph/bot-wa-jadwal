package schedule

import (
	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/util"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type ScheduleOverride struct {
	ID           int
	ScopeJID     string
	Type         string // RESCHEDULE, CANCEL, EXTRA, HOLIDAY
	KodeMatkul   string
	NamaMatkul   string
	Dosen        string
	InisialDosen string
	OrigDate     string // YYYY-MM-DD
	OrigJam      string // contoh: "07:00 - 08:40"
	TargetDate   string // YYYY-MM-DD
	NewJam       string // contoh: "13:00 - 14:40"
	Ruang        string
	Alasan       string
	CreatedBy    string
	CreatedAt    time.Time
}

type OverrideManager struct {
	db           *sql.DB
	academicRepo *academic.Repository
}

// NewOverrideManager menginisialisasi OverrideManager berbasis target teaching_events dan teaching_event_offerings
func NewOverrideManager(db *sql.DB) (*OverrideManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	om := &OverrideManager{
		db:           db,
		academicRepo: academic.NewRepository(db),
	}

	// Backfill data legacy jika tabel lama schedule_overrides masih ada di database
	_ = om.backfillLegacyOverrides()

	return om, nil
}

// NewOverrideManagerWithPath membuat koneksi baru dari path file dan menginisialisasi OverrideManager
func NewOverrideManagerWithPath(dbPath string) (*OverrideManager, error) {
	db, err := database.InitDB(dbPath)
	if err != nil {
		return nil, err
	}
	return NewOverrideManager(db)
}

func (om *OverrideManager) Close() error {
	if om.db != nil {
		return om.db.Close()
	}
	return nil
}

func parseJam(jamStr string) (start, end string) {
	parts := strings.Split(jamStr, "-")
	if len(parts) >= 2 {
		start = strings.TrimSpace(parts[0])
		end = strings.TrimSpace(parts[1])
	} else if len(parts) == 1 {
		start = strings.TrimSpace(parts[0])
		end = "23:59"
	}
	if len(start) != 5 {
		start = "07:00"
	}
	if len(end) != 5 {
		end = "08:40"
	}
	if start >= end {
		end = "23:59"
	}
	return start, end
}

func parseTimeRange(dateStr, jamStr string) (startsAt, endsAt string) {
	start, end := parseJam(jamStr)
	startsAt = dateStr + "T" + start + ":00Z"
	endsAt = dateStr + "T" + end + ":00Z"
	return startsAt, endsAt
}

func parseJamFromTimes(startsAt, endsAt string) string {
	startTime := ""
	endTime := ""
	if len(startsAt) >= 16 {
		if strings.Contains(startsAt, "T") {
			parts := strings.Split(startsAt, "T")
			if len(parts) > 1 && len(parts[1]) >= 5 {
				startTime = parts[1][:5]
			}
		} else if strings.Contains(startsAt, " ") {
			parts := strings.Split(startsAt, " ")
			if len(parts) > 1 && len(parts[1]) >= 5 {
				startTime = parts[1][:5]
			}
		}
	}
	if len(endsAt) >= 16 {
		if strings.Contains(endsAt, "T") {
			parts := strings.Split(endsAt, "T")
			if len(parts) > 1 && len(parts[1]) >= 5 {
				endTime = parts[1][:5]
			}
		} else if strings.Contains(endsAt, " ") {
			parts := strings.Split(endsAt, " ")
			if len(parts) > 1 && len(parts[1]) >= 5 {
				endTime = parts[1][:5]
			}
		}
	}
	if startTime != "" && endTime != "" && (startTime != "00:00" || endTime != "23:59") {
		return startTime + " - " + endTime
	}
	return ""
}

// AddReschedule menambahkan perubahan jadwal kuliah ke tanggal/jam lain
func (om *OverrideManager) AddReschedule(
	scopeJID string, item JadwalItem, origDate, targetDate time.Time, newJam, newRuang, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ruang := newRuang
	if ruang == "" {
		ruang = item.Ruang
	}

	origDateStr := origDate.Format("2006-01-02")
	targetDateStr := targetDate.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err != nil {
			return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, err)
		}
		classID = cls.ID
	}

	userID, err := om.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan user %s: %w", createdBy, err)
	}

	var newRoomID *int64
	if ruang != "" && ruang != "-" {
		rid, err := om.academicRepo.EnsureRoom(ctx, ruang, ruang)
		if err == nil {
			newRoomID = &rid
		}
	}

	var origRoomID *int64
	if item.Ruang != "" && item.Ruang != "-" {
		rid, err := om.academicRepo.EnsureRoom(ctx, item.Ruang, item.Ruang)
		if err == nil {
			origRoomID = &rid
		}
	}

	offeringID, err := om.academicRepo.EnsureCourseOffering(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI")
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering: %w", err)
	}

	if item.Dosen != "" || item.InisialDosen != "" {
		lecID, err := om.academicRepo.EnsureLecturer(ctx, item.InisialDosen, item.Dosen)
		if err == nil && lecID > 0 {
			_ = om.academicRepo.EnsureOfferingLecturer(ctx, offeringID, lecID)
		}
	}

	origDay := int(origDate.Weekday())
	if origDay == 0 {
		origDay = 7
	}
	origStart, origEnd := parseJam(item.Jam)
	patternID, err := om.academicRepo.EnsureSchedulePattern(ctx, offeringID, origDay, origStart, origEnd, origRoomID)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan schedule_pattern asal: %w", err)
	}

	startsAt, endsAt := parseTimeRange(targetDateStr, newJam)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES (?, ?, 'REPLACEMENT', ?, ?, ?, '', 'PUBLISHED', ?, ?)
	`, patternID, origDateStr, startsAt, endsAt, newRoomID, userID, nowStr)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (REPLACEMENT): %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		) VALUES (?, ?, 'OWNER', 'ACCEPTED')
	`, id, offeringID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_event_offerings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "RESCHEDULE",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		OrigDate:     origDateStr,
		OrigJam:      item.Jam,
		TargetDate:   targetDateStr,
		NewJam:       newJam,
		Ruang:        ruang,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// AddCancel menandai kuliah ditiadakan pada tanggal tertentu
func (om *OverrideManager) AddCancel(
	scopeJID string, item JadwalItem, targetDate time.Time, alasan, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dateStr := targetDate.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err != nil {
			return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, err)
		}
		classID = cls.ID
	}

	userID, err := om.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan user %s: %w", createdBy, err)
	}

	var roomID *int64
	if item.Ruang != "" && item.Ruang != "-" {
		rid, err := om.academicRepo.EnsureRoom(ctx, item.Ruang, item.Ruang)
		if err == nil {
			roomID = &rid
		}
	}

	offeringID, err := om.academicRepo.EnsureCourseOffering(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI")
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering: %w", err)
	}

	if item.Dosen != "" || item.InisialDosen != "" {
		lecID, err := om.academicRepo.EnsureLecturer(ctx, item.InisialDosen, item.Dosen)
		if err == nil && lecID > 0 {
			_ = om.academicRepo.EnsureOfferingLecturer(ctx, offeringID, lecID)
		}
	}

	dayOfWeek := int(targetDate.Weekday())
	if dayOfWeek == 0 {
		dayOfWeek = 7
	}
	sTime, eTime := parseJam(item.Jam)
	patternID, err := om.academicRepo.EnsureSchedulePattern(ctx, offeringID, dayOfWeek, sTime, eTime, roomID)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan schedule_pattern asal: %w", err)
	}

	startsAt, endsAt := parseTimeRange(dateStr, item.Jam)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES (?, ?, 'SESSION_CANCELLED', ?, ?, ?, ?, 'PUBLISHED', ?, ?)
	`, patternID, dateStr, startsAt, endsAt, roomID, alasan, userID, nowStr)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (SESSION_CANCELLED): %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		) VALUES (?, ?, 'OWNER', 'ACCEPTED')
	`, id, offeringID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_event_offerings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "CANCEL",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		OrigDate:     dateStr,
		OrigJam:      item.Jam,
		TargetDate:   dateStr,
		Ruang:        item.Ruang,
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// AddExtra menambahkan jadwal kuliah pengganti/ekstra di hari tertentu
func (om *OverrideManager) AddExtra(
	scopeJID string, item JadwalItem, targetDate time.Time, jam, ruang, alasan, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dateStr := targetDate.Format("2006-01-02")
	if ruang == "" {
		ruang = item.Ruang
	}

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err != nil {
			return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, err)
		}
		classID = cls.ID
	}

	userID, err := om.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan user %s: %w", createdBy, err)
	}

	var roomID *int64
	if ruang != "" && ruang != "-" {
		rid, err := om.academicRepo.EnsureRoom(ctx, ruang, ruang)
		if err == nil {
			roomID = &rid
		}
	}

	offeringID, err := om.academicRepo.EnsureCourseOffering(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI")
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering: %w", err)
	}

	if item.Dosen != "" || item.InisialDosen != "" {
		lecID, err := om.academicRepo.EnsureLecturer(ctx, item.InisialDosen, item.Dosen)
		if err == nil && lecID > 0 {
			_ = om.academicRepo.EnsureOfferingLecturer(ctx, offeringID, lecID)
		}
	}

	startsAt, endsAt := parseTimeRange(dateStr, jam)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO teaching_events (
			event_kind, starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES ('EXTRA', ?, ?, ?, ?, 'PUBLISHED', ?, ?)
	`, startsAt, endsAt, roomID, alasan, userID, nowStr)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (EXTRA): %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		) VALUES (?, ?, 'OWNER', 'ACCEPTED')
	`, id, offeringID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_event_offerings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "EXTRA",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		TargetDate:   dateStr,
		NewJam:       jam,
		Ruang:        ruang,
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// GetOverridesForDate mengambil seluruh catatan override yang mempengaruhi tanggal tertentu
func (om *OverrideManager) GetOverridesForDate(scopeJID string, date time.Time) ([]ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dateStr := date.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err == nil && cls != nil {
			classID = cls.ID
		}
	}

	query := `
		SELECT
			te.id,
			te.event_kind,
			te.starts_at,
			te.ends_at,
			COALESCE(te.origin_occurrence_date, '') AS orig_occ_date,
			COALESCE(te.reason, '') AS reason,
			te.created_at,
			COALESCE(c.code, 'MK-UMUM') AS course_code,
			COALESCE(co.display_name, c.name, 'Mata Kuliah') AS course_name,
			COALESCE(r.code, '-') AS room_code,
			COALESCE(u.identity_key, 'system') AS creator_key,
			COALESCE(sp.start_time || ' - ' || sp.end_time, '') AS orig_jam,
			COALESCE(l.full_name, '-') AS lecturer_name,
			COALESCE(l.code, '-') AS lecturer_code
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters s ON s.id = co.semester_id
		LEFT JOIN courses c ON c.id = co.course_id
		LEFT JOIN rooms r ON r.id = te.room_id
		LEFT JOIN users u ON u.id = te.published_by_user_id
		LEFT JOIN schedule_patterns sp ON sp.id = te.origin_schedule_pattern_id
		LEFT JOIN offering_lecturers ol ON ol.course_offering_id = co.id
		LEFT JOIN lecturers l ON l.id = ol.lecturer_id
		WHERE s.class_id = ?
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND (te.origin_occurrence_date = ? OR date(te.starts_at) = ?)
		GROUP BY te.id
		ORDER BY te.id ASC;
	`
	rows, err := om.db.QueryContext(ctx, query, classID, dateStr, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleOverride
	for rows.Next() {
		var (
			id                                                             int
			eventKind, startsAt, endsAt, origOccDate, reason, rawCreatedAt string
			courseCode, courseName, roomCode, creatorKey, origJam          string
			lecturerName, lecturerCode                                     string
		)
		err := rows.Scan(
			&id, &eventKind, &startsAt, &endsAt, &origOccDate, &reason, &rawCreatedAt,
			&courseCode, &courseName, &roomCode, &creatorKey, &origJam, &lecturerName, &lecturerCode,
		)
		if err != nil {
			continue
		}

		var overrideType string
		switch eventKind {
		case "REPLACEMENT":
			overrideType = "RESCHEDULE"
		case "SESSION_CANCELLED":
			overrideType = "CANCEL"
		case "EXTRA":
			overrideType = "EXTRA"
		case "HOLIDAY":
			overrideType = "HOLIDAY"
		default:
			overrideType = eventKind
		}

		targetDate := ""
		if len(startsAt) >= 10 {
			targetDate = startsAt[:10]
		}

		origDate := origOccDate
		if origDate == "" {
			origDate = targetDate
		}

		newJam := parseJamFromTimes(startsAt, endsAt)
		if overrideType == "CANCEL" {
			newJam = ""
		}
		if overrideType == "HOLIDAY" {
			newJam = "-"
			origJam = "-"
			roomCode = "-"
		}

		createdAt := util.ParseFlexibleTime(rawCreatedAt, date.Location())

		list = append(list, ScheduleOverride{
			ID:           id,
			ScopeJID:     scopeJID,
			Type:         overrideType,
			KodeMatkul:   courseCode,
			NamaMatkul:   courseName,
			Dosen:        lecturerName,
			InisialDosen: lecturerCode,
			OrigDate:     origDate,
			OrigJam:      origJam,
			TargetDate:   targetDate,
			NewJam:       newJam,
			Ruang:        roomCode,
			Alasan:       reason,
			CreatedBy:    creatorKey,
			CreatedAt:    createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

// GetActiveOverrides mengambil semua override yang tanggal targetnya belum lewat
func (om *OverrideManager) GetActiveOverrides(scopeJID string, now time.Time) ([]ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	todayStr := now.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err == nil && cls != nil {
			classID = cls.ID
		}
	}

	query := `
		SELECT
			te.id,
			te.event_kind,
			te.starts_at,
			te.ends_at,
			COALESCE(te.origin_occurrence_date, '') AS orig_occ_date,
			COALESCE(te.reason, '') AS reason,
			te.created_at,
			COALESCE(c.code, 'MK-UMUM') AS course_code,
			COALESCE(co.display_name, c.name, 'Mata Kuliah') AS course_name,
			COALESCE(r.code, '-') AS room_code,
			COALESCE(u.identity_key, 'system') AS creator_key,
			COALESCE(sp.start_time || ' - ' || sp.end_time, '') AS orig_jam,
			COALESCE(l.full_name, '-') AS lecturer_name,
			COALESCE(l.code, '-') AS lecturer_code
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters s ON s.id = co.semester_id
		LEFT JOIN courses c ON c.id = co.course_id
		LEFT JOIN rooms r ON r.id = te.room_id
		LEFT JOIN users u ON u.id = te.published_by_user_id
		LEFT JOIN schedule_patterns sp ON sp.id = te.origin_schedule_pattern_id
		LEFT JOIN offering_lecturers ol ON ol.course_offering_id = co.id
		LEFT JOIN lecturers l ON l.id = ol.lecturer_id
		WHERE s.class_id = ?
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND (date(te.starts_at) >= ? OR te.origin_occurrence_date >= ?)
		GROUP BY te.id
		ORDER BY date(te.starts_at) ASC, te.id ASC;
	`
	rows, err := om.db.QueryContext(ctx, query, classID, todayStr, todayStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleOverride
	for rows.Next() {
		var (
			id                                                             int
			eventKind, startsAt, endsAt, origOccDate, reason, rawCreatedAt string
			courseCode, courseName, roomCode, creatorKey, origJam          string
			lecturerName, lecturerCode                                     string
		)
		err := rows.Scan(
			&id, &eventKind, &startsAt, &endsAt, &origOccDate, &reason, &rawCreatedAt,
			&courseCode, &courseName, &roomCode, &creatorKey, &origJam, &lecturerName, &lecturerCode,
		)
		if err != nil {
			continue
		}

		var overrideType string
		switch eventKind {
		case "REPLACEMENT":
			overrideType = "RESCHEDULE"
		case "SESSION_CANCELLED":
			overrideType = "CANCEL"
		case "EXTRA":
			overrideType = "EXTRA"
		case "HOLIDAY":
			overrideType = "HOLIDAY"
		default:
			overrideType = eventKind
		}

		targetDate := ""
		if len(startsAt) >= 10 {
			targetDate = startsAt[:10]
		}

		origDate := origOccDate
		if origDate == "" {
			origDate = targetDate
		}

		newJam := parseJamFromTimes(startsAt, endsAt)
		if overrideType == "CANCEL" {
			newJam = ""
		}
		if overrideType == "HOLIDAY" {
			newJam = "-"
			origJam = "-"
			roomCode = "-"
		}

		createdAt := util.ParseFlexibleTime(rawCreatedAt, now.Location())

		list = append(list, ScheduleOverride{
			ID:           id,
			ScopeJID:     scopeJID,
			Type:         overrideType,
			KodeMatkul:   courseCode,
			NamaMatkul:   courseName,
			Dosen:        lecturerName,
			InisialDosen: lecturerCode,
			OrigDate:     origDate,
			OrigJam:      origJam,
			TargetDate:   targetDate,
			NewJam:       newJam,
			Ruang:        roomCode,
			Alasan:       reason,
			CreatedBy:    creatorKey,
			CreatedAt:    createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return list, nil
}

func (om *OverrideManager) CancelOverride(scopeJID string, id int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err == nil && cls != nil {
			classID = cls.ID
		}
	}

	systemUserID, err := om.academicRepo.EnsureUser(ctx, "system", "System Administrator")
	if err != nil {
		systemUserID = 1
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)

	var res sql.Result
	if classID > 0 {
		res, err = om.db.ExecContext(ctx, `
			UPDATE teaching_events
			SET lifecycle_status = 'REVOKED',
				revoked_by_user_id = ?,
				revoked_at = ?,
				revocation_reason = 'Dibatalkan oleh pengguna',
				updated_at = ?
			WHERE id = ?
			  AND lifecycle_status = 'PUBLISHED'
			  AND id IN (
				  SELECT teo.teaching_event_id
				  FROM teaching_event_offerings teo
				  JOIN course_offerings co ON co.id = teo.course_offering_id
				  JOIN semesters s ON s.id = co.semester_id
				  WHERE s.class_id = ?
			  );
		`, systemUserID, nowStr, nowStr, id, classID)
	} else {
		res, err = om.db.ExecContext(ctx, `
			UPDATE teaching_events
			SET lifecycle_status = 'REVOKED',
				revoked_by_user_id = ?,
				revoked_at = ?,
				revocation_reason = 'Dibatalkan oleh pengguna',
				updated_at = ?
			WHERE id = ? AND lifecycle_status = 'PUBLISHED';
		`, systemUserID, nowStr, nowStr, id)
	}
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// AddHoliday menambahkan pengumuman libur harian (seluruh perkuliahan pada tanggal tersebut ditiadakan)
func (om *OverrideManager) AddHoliday(scopeJID string, targetDate time.Time, alasan string, createdBy string) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tglStr := targetDate.Format("2006-01-02")
	if alasan == "" {
		alasan = "Libur Perkuliahan"
	}

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err != nil {
			return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, err)
		}
		classID = cls.ID
	}

	userID, err := om.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan user %s: %w", createdBy, err)
	}

	offeringID, err := om.academicRepo.EnsureCourseOffering(ctx, classID, "LIBUR", "LIBUR SEHARIAN", "TEORI")
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering libur: %w", err)
	}

	startsAt := tglStr + "T00:00:00Z"
	endsAt := tglStr + "T23:59:59Z"
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO teaching_events (
			event_kind, starts_at, ends_at, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES ('HOLIDAY', ?, ?, ?, 'PUBLISHED', ?, ?)
	`, startsAt, endsAt, alasan, userID, nowStr)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (HOLIDAY): %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO teaching_event_offerings (
			teaching_event_id, course_offering_id, participation_role, participation_status
		) VALUES (?, ?, 'OWNER', 'ACCEPTED')
	`, id, offeringID)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_event_offerings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("gagal commit transaksi: %w", err)
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "HOLIDAY",
		KodeMatkul:   "LIBUR",
		NamaMatkul:   "LIBUR SEHARIAN",
		Dosen:        "-",
		InisialDosen: "-",
		OrigDate:     tglStr,
		OrigJam:      "-",
		TargetDate:   tglStr,
		NewJam:       "-",
		Ruang:        "-",
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

func (om *OverrideManager) GetHolidayOverride(scopeJID string, date time.Time) *ScheduleOverride {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tglStr := date.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		cls, err := om.academicRepo.EnsureClass(ctx, scopeJID)
		if err == nil && cls != nil {
			classID = cls.ID
		}
	}

	query := `
		SELECT
			te.id,
			te.event_kind,
			te.starts_at,
			te.ends_at,
			COALESCE(te.reason, '') AS reason,
			te.created_at,
			COALESCE(u.identity_key, 'system') AS creator_key
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters s ON s.id = co.semester_id
		LEFT JOIN users u ON u.id = te.published_by_user_id
		WHERE s.class_id = ?
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND te.event_kind = 'HOLIDAY'
		  AND date(te.starts_at) = ?
		ORDER BY te.id DESC
		LIMIT 1;
	`
	row := om.db.QueryRowContext(ctx, query, classID, tglStr)

	var (
		id                               int
		eventKind, startsAt, endsAt      string
		reason, rawCreatedAt, creatorKey string
	)
	err = row.Scan(&id, &eventKind, &startsAt, &endsAt, &reason, &rawCreatedAt, &creatorKey)
	if err != nil {
		return nil
	}

	return &ScheduleOverride{
		ID:           id,
		ScopeJID:     scopeJID,
		Type:         "HOLIDAY",
		KodeMatkul:   "LIBUR",
		NamaMatkul:   "LIBUR SEHARIAN",
		Dosen:        "-",
		InisialDosen: "-",
		OrigDate:     tglStr,
		OrigJam:      "-",
		TargetDate:   tglStr,
		NewJam:       "-",
		Ruang:        "-",
		Alasan:       reason,
		CreatedBy:    creatorKey,
		CreatedAt:    util.ParseFlexibleTime(rawCreatedAt, date.Location()),
	}
}

// backfillLegacyOverrides menyalin data dari tabel legacy schedule_overrides ke teaching_events & teaching_event_offerings
func (om *OverrideManager) backfillLegacyOverrides() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tblName string
	err := om.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='schedule_overrides';").Scan(&tblName)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return nil
	}

	rows, err := om.db.QueryContext(ctx, `
		SELECT id, scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
		       orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by, created_at
		FROM schedule_overrides;
	`)
	if err != nil {
		return nil
	}

	type legacyRow struct {
		id                                                                         int
		scopeJID, oType, kodeMatkul, namaMatkul, dosen, inisialDosen               string
		origDate, origJam, targetDate, newJam, ruang, alasan, createdBy, createdAt string
	}

	var legacyRows []legacyRow
	for rows.Next() {
		var lr legacyRow
		if err := rows.Scan(
			&lr.id, &lr.scopeJID, &lr.oType, &lr.kodeMatkul, &lr.namaMatkul, &lr.dosen, &lr.inisialDosen,
			&lr.origDate, &lr.origJam, &lr.targetDate, &lr.newJam, &lr.ruang, &lr.alasan, &lr.createdBy, &lr.createdAt,
		); err == nil {
			legacyRows = append(legacyRows, lr)
		}
	}
	_ = rows.Close()

	for _, lr := range legacyRows {
		classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, lr.scopeJID)
		if err != nil || classID == 0 {
			cls, err := om.academicRepo.EnsureClass(ctx, lr.scopeJID)
			if err != nil {
				continue
			}
			classID = cls.ID
		}

		userID, err := om.academicRepo.EnsureUser(ctx, lr.createdBy, lr.createdBy)
		if err != nil {
			continue
		}

		var roomID *int64
		if lr.ruang != "" && lr.ruang != "-" {
			rid, err := om.academicRepo.EnsureRoom(ctx, lr.ruang, lr.ruang)
			if err == nil {
				roomID = &rid
			}
		}

		offeringID, err := om.academicRepo.EnsureCourseOffering(ctx, classID, lr.kodeMatkul, lr.namaMatkul, "TEORI")
		if err != nil {
			continue
		}

		if lr.dosen != "" || lr.inisialDosen != "" {
			lecID, err := om.academicRepo.EnsureLecturer(ctx, lr.inisialDosen, lr.dosen)
			if err == nil && lecID > 0 {
				_ = om.academicRepo.EnsureOfferingLecturer(ctx, offeringID, lecID)
			}
		}

		var eventKind string
		var startsAt, endsAt string
		var originPatternID *int64
		var originOccDate *string

		switch lr.oType {
		case "RESCHEDULE":
			eventKind = "REPLACEMENT"
			tDate, err := time.Parse("2006-01-02", lr.origDate)
			dayOfWeek := 1
			if err == nil {
				dayOfWeek = int(tDate.Weekday())
				if dayOfWeek == 0 {
					dayOfWeek = 7
				}
			}
			sTime, eTime := parseJam(lr.origJam)
			patID, err := om.academicRepo.EnsureSchedulePattern(ctx, offeringID, dayOfWeek, sTime, eTime, roomID)
			if err == nil {
				originPatternID = &patID
			}
			originOccDate = &lr.origDate
			startsAt, endsAt = parseTimeRange(lr.targetDate, lr.newJam)

		case "CANCEL":
			eventKind = "SESSION_CANCELLED"
			tDate, err := time.Parse("2006-01-02", lr.targetDate)
			dayOfWeek := 1
			if err == nil {
				dayOfWeek = int(tDate.Weekday())
				if dayOfWeek == 0 {
					dayOfWeek = 7
				}
			}
			sTime, eTime := parseJam(lr.origJam)
			patID, err := om.academicRepo.EnsureSchedulePattern(ctx, offeringID, dayOfWeek, sTime, eTime, roomID)
			if err == nil {
				originPatternID = &patID
			}
			originOccDate = &lr.targetDate
			startsAt, endsAt = parseTimeRange(lr.targetDate, lr.origJam)

		case "EXTRA":
			eventKind = "EXTRA"
			startsAt, endsAt = parseTimeRange(lr.targetDate, lr.newJam)

		case "HOLIDAY":
			eventKind = "HOLIDAY"
			startsAt = lr.targetDate + "T00:00:00Z"
			endsAt = lr.targetDate + "T23:59:59Z"

		default:
			continue
		}

		var existingID int64
		err = om.db.QueryRowContext(ctx, `
			SELECT te.id FROM teaching_events te
			JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
			WHERE teo.course_offering_id = ? AND te.event_kind = ? AND te.starts_at = ?
		`, offeringID, eventKind, startsAt).Scan(&existingID)
		if err == nil && existingID > 0 {
			continue
		}

		tx, err := om.db.BeginTx(ctx, nil)
		if err != nil {
			continue
		}

		res, err := tx.ExecContext(ctx, `
			INSERT INTO teaching_events (
				origin_schedule_pattern_id, origin_occurrence_date, event_kind,
				starts_at, ends_at, room_id, reason, lifecycle_status,
				published_by_user_id, published_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, 'PUBLISHED', ?, ?, ?)
		`, originPatternID, originOccDate, eventKind, startsAt, endsAt, roomID, lr.alasan, userID, lr.createdAt, lr.createdAt)
		if err != nil {
			_ = tx.Rollback()
			continue
		}

		evID, err := res.LastInsertId()
		if err != nil {
			_ = tx.Rollback()
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO teaching_event_offerings (
				teaching_event_id, course_offering_id, participation_role, participation_status
			) VALUES (?, ?, 'OWNER', 'ACCEPTED')
		`, evID, offeringID)
		if err != nil {
			_ = tx.Rollback()
			continue
		}

		_ = tx.Commit()
	}

	return nil
}

// ParseOverrideDate mengekstrak tanggal target dari input teks fleksibel
func ParseOverrideDate(rawInput string, refNow time.Time) time.Time {
	clean := strings.ToLower(strings.TrimSpace(rawInput))
	loc := refNow.Location()

	if strings.Contains(clean, "hari ini") || strings.Contains(clean, "today") {
		return refNow
	}
	if strings.Contains(clean, "besok") || strings.Contains(clean, "tomorrow") {
		return refNow.Add(24 * time.Hour)
	}

	for dayName, weekday := range util.NamaHariMap {
		if strings.Contains(clean, dayName) {
			daysAhead := int(weekday - refNow.Weekday())
			if daysAhead <= 0 {
				daysAhead += 7
			}
			t := refNow.AddDate(0, 0, daysAhead)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
	}

	layouts := []string{"02-01-2006", "2006-01-02", "02/01/2006"}
	for _, layout := range layouts {
		re := regexp.MustCompile(`\b\d{2,4}[-/]\d{2}[-/]\d{2,4}\b`)
		if found := re.FindString(clean); found != "" {
			if t, err := time.ParseInLocation(layout, found, loc); err == nil {
				return t
			}
		}
	}

	if target, ok := util.ParseIndonesianDateWord(clean, refNow, loc); ok {
		return target
	}

	return refNow.Add(24 * time.Hour)
}

// FormatActiveOverrides merapikan daftar jadwal pengganti aktif
func (om *OverrideManager) FormatActiveOverrides(overrides []ScheduleOverride) string {
	var sb strings.Builder
	sb.WriteString("🔄 *DAFTAR JADWAL PENGGANTI AKTIF*\n")
	sb.WriteString("──────────\n\n")

	if len(overrides) == 0 {
		sb.WriteString("ℹ️ Tidak ada perubahan jadwal atau kuliah pengganti sementara.\nSemua perkuliahan berjalan sesuai jadwal normal.\n")
		return sb.String()
	}

	for i, o := range overrides {
		sb.WriteString(fmt.Sprintf("*%d. [%s] %s*\n", i+1, strings.ToUpper(o.Type), o.NamaMatkul))
		sb.WriteString(fmt.Sprintf("   • ID Perubahan: #%d\n", o.ID))
		switch o.Type {
		case "HOLIDAY":
			sb.WriteString(fmt.Sprintf("   • Tanggal Libur : %s (Seharian)\n", o.TargetDate))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Keterangan    : %s\n", o.Alasan))
			}
		case "CANCEL":
			sb.WriteString(fmt.Sprintf("   • Tanggal Libur : %s (%s)\n", o.TargetDate, o.OrigJam))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Keterangan    : %s\n", o.Alasan))
			}
		case "RESCHEDULE":
			sb.WriteString(fmt.Sprintf("   • Semula : %s (%s)\n", o.OrigDate, o.OrigJam))
			sb.WriteString(fmt.Sprintf("   • Menjadi: %s (%s WIB)\n", o.TargetDate, o.NewJam))
			sb.WriteString(fmt.Sprintf("   • Ruang  : %s\n", o.Ruang))
		case "EXTRA":
			sb.WriteString(fmt.Sprintf("   • Tanggal: %s (%s WIB)\n", o.TargetDate, o.NewJam))
			sb.WriteString(fmt.Sprintf("   • Ruang  : %s\n", o.Ruang))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Ket    : %s\n", o.Alasan))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Admin dapat membatalkan perubahan dengan `!batalganti [ID]`._")
	return sb.String()
}

type ScheduleConflict struct {
	Matkul string
	Jam    string
	Ruang  string
	Dosen  string
}

// CheckScheduleConflict memeriksa apakah jam baru di tanggal tertentu bertabrakan dengan jadwal aktif lainnya
func (om *OverrideManager) CheckScheduleConflict(
	scopeJID string, targetDate time.Time, newJam string, ignoreItem *JadwalItem, cfg *JadwalConfig,
) *ScheduleConflict {
	if cfg == nil {
		return nil
	}

	propStart, propEnd, err := util.ParseJamRange(newJam, targetDate)
	if err != nil {
		return nil
	}

	targetDateStr := targetDate.Format("2006-01-02")
	hariTarget := util.GetHariIndonesia(targetDate)

	overrides, _ := om.GetOverridesForDate(scopeJID, targetDate)

	cfg.mu.RLock()
	var normalItems []JadwalItem
	for _, it := range cfg.Jadwal {
		if strings.EqualFold(it.Hari, hariTarget) {
			normalItems = append(normalItems, it)
		}
	}
	cfg.mu.RUnlock()

	for _, it := range normalItems {
		// Abaikan jika ini adalah sesi matkul yang sama yang sedang dipindahkan
		if ignoreItem != nil && it.KodeMatkul == ignoreItem.KodeMatkul && it.Jam == ignoreItem.Jam {
			continue
		}

		// Periksa apakah jadwal reguler ini sudah ditiadakan (CANCEL) atau dipindahkan keluar (RESCHEDULE)
		isCancelledOrMoved := false
		for _, o := range overrides {
			if o.OrigDate == targetDateStr &&
				(o.KodeMatkul == it.KodeMatkul || strings.EqualFold(o.NamaMatkul, it.NamaMatkul)) &&
				(o.OrigJam == it.Jam || strings.Contains(it.Jam, strings.TrimSpace(strings.Split(o.OrigJam, "-")[0]))) {
				if o.Type == "CANCEL" || o.Type == "RESCHEDULE" {
					isCancelledOrMoved = true
					break
				}
			}
		}
		if isCancelledOrMoved {
			continue
		}

		itStart, itEnd, err := util.ParseJamRange(it.Jam, targetDate)
		if err != nil {
			continue
		}

		// Cek irisan waktu: [propStart, propEnd) beririsan dengan [itStart, itEnd)
		if propStart.Before(itEnd) && propEnd.After(itStart) {
			return &ScheduleConflict{
				Matkul: it.NamaMatkul,
				Jam:    it.Jam,
				Ruang:  it.Ruang,
				Dosen:  it.Dosen,
			}
		}
	}

	for _, o := range overrides {
		if o.TargetDate == targetDateStr && (o.Type == "RESCHEDULE" || o.Type == "EXTRA") {
			if ignoreItem != nil && o.KodeMatkul == ignoreItem.KodeMatkul {
				continue
			}

			oStart, oEnd, err := util.ParseJamRange(o.NewJam, targetDate)
			if err != nil {
				continue
			}

			if propStart.Before(oEnd) && propEnd.After(oStart) {
				return &ScheduleConflict{
					Matkul: o.NamaMatkul,
					Jam:    o.NewJam,
					Ruang:  o.Ruang,
					Dosen:  o.Dosen,
				}
			}
		}
	}

	return nil
}

// HandleCommand memproses perintah perubahan jadwal (!pindah, !kosong, !kuliahganti, !jadwalganti, !batalganti)
func (om *OverrideManager) HandleCommand(
	scopeJID string, isGroup bool, senderJID string, isAdmin bool, rawMsg string, cfg *JadwalConfig, now time.Time,
) string {
	clean := util.CleanCommandPrefix(rawMsg)

	parts := strings.SplitN(clean, " ", 2)
	cmd := strings.ToLower(parts[0])
	payload := ""
	if len(parts) > 1 {
		payload = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "pindah", "ganti", "reschedule":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, perubahan jadwal hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		isForce := false
		var cleanSegments []string
		for _, seg := range segments {
			trimmed := strings.TrimSpace(seg)
			if strings.EqualFold(trimmed, "paksa") || strings.EqualFold(trimmed, "force") {
				isForce = true
			} else {
				cleanSegments = append(cleanSegments, trimmed)
			}
		}
		segments = cleanSegments

		if len(segments) < 2 {
			return "⚠️ *Format Perintah Pindah Kurang Tepat*\n\n" +
				"Gunakan format pemisah pipa `|`:\n" +
				"`!pindah [Matkul] | [Hari/Tanggal & Jam Baru] | [Ruang (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!pindah aljabar | besok 13:00`\n" +
				"• `!pindah sbd | jumat 15:00 - 16:40 | Lab 312`\n" +
				"• `!pindah matdis | 10-09-2026 09:00 | D105`"
		}

		matkulQuery := strings.TrimSpace(segments[0])
		timeRaw := strings.TrimSpace(segments[1])
		ruangBaru := ""
		if len(segments) > 2 {
			ruangBaru = strings.TrimSpace(segments[2])
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				var sb strings.Builder
				sb.WriteString("⚠️ *Ditemukan beberapa sesi mata kuliah yang cocok:*\n")
				for _, c := range candidates {
					sb.WriteString(fmt.Sprintf("• [%s] %s (%s, %s)\n", c.KodeMatkul, c.NamaMatkul, c.Hari, c.Jam))
				}
				sb.WriteString("\nSilakan perjelas nama sesi (contoh: `!pindah aljabar teori | ...` atau `!pindah aljabar senin | ...`)")
				return sb.String()
			}
			return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan. Ketik `!matkul` untuk melihat daftar mata kuliah.", matkulQuery)
		}

		origDate := util.GetDateForDayName(item.Hari, now)
		targetDate := ParseOverrideDate(timeRaw, now)
		baseDuration := util.CalculateDurationInMinutes(item.Jam)
		newJam := util.AutoCompleteJamRange(timeRaw, baseDuration)

		conflict := om.CheckScheduleConflict(scopeJID, targetDate, newJam, item, cfg)
		if conflict != nil && !isForce {
			hariTgt := util.GetHariIndonesia(targetDate)
			tglTgt := targetDate.Format("02-01-2006")
			var sb strings.Builder
			sb.WriteString("⚠️ *PERINGATAN BENTROK JADWAL!*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("Waktu baru yang dipilih (*%s, %s, %s WIB*) bertabrakan dengan jadwal:\n\n", hariTgt, tglTgt, newJam))
			sb.WriteString(fmt.Sprintf("• *%s*\n", conflict.Matkul))
			sb.WriteString(fmt.Sprintf("  └ Jam   : %s WIB\n", conflict.Jam))
			sb.WriteString(fmt.Sprintf("  └ Ruang : %s\n", conflict.Ruang))
			if conflict.Dosen != "" {
				sb.WriteString(fmt.Sprintf("  └ Dosen : %s\n", conflict.Dosen))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("Jadwal tidak dipindahkan untuk mencegah jadwal kuliah ganda.\n\n")
			sb.WriteString("💡 *Apakah tetap ingin memindahkan?*\n")
			sb.WriteString(fmt.Sprintf("1. Pilih jam lain yang kosong (ketik `!%s` untuk cek jam kosong).\n", strings.ToLower(hariTgt)))
			sb.WriteString("2. Jika jam tersebut memang disepakati (misal kelas tersebut ditiadakan), tambahkan kata `paksa` di akhir:\n")
			sb.WriteString(fmt.Sprintf("   `!pindah %s | %s | paksa`", matkulQuery, timeRaw))
			return sb.String()
		}

		if ruangBaru == "" {
			ruangBaru = item.Ruang
		}

		override, err := om.AddReschedule(scopeJID, *item, origDate, targetDate, newJam, ruangBaru, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menyimpan perubahan jadwal: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *JADWAL BERHASIL DIPINDAHKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Semula      : %s (%s WIB)\n", override.OrigDate, override.OrigJam))
		sb.WriteString(fmt.Sprintf("• Menjadi     : %s (%s WIB)\n", override.TargetDate, override.NewJam))
		sb.WriteString(fmt.Sprintf("• Ruang       : %s\n", override.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen       : %s (%s)\n", item.Dosen, item.InisialDosen))
		if isForce {
			sb.WriteString("• Peringatan  : ⚠️ *Jadwal dipindahkan dengan konfirmasi bentrok (dipaksa oleh Admin).*\n")
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Tips: Jadwal ini otomatis kedaluwarsa setelah tanggal lewat. Ketik `!batalganti %d` jika ingin membatalkan._", override.ID))
		return sb.String()

	case "kosong", "batal", "cancel":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, peniadaan jadwal kuliah hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		matkulQuery := strings.TrimSpace(segments[0])
		if matkulQuery == "" {
			return "⚠️ *Format Perintah Kosong Kurang Tepat*\n\n" +
				"Gunakan format:\n" +
				"`!kosong [Matkul] | [Hari/Tanggal (Opsional)] | [Alasan (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!kosong aljabar` (meniadakan jadwal terdekat)\n" +
				"• `!kosong aljabar | besok | Dosen dinas luar`"
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				var sb strings.Builder
				sb.WriteString("⚠️ *Ditemukan beberapa sesi mata kuliah yang cocok:*\n")
				for _, c := range candidates {
					sb.WriteString(fmt.Sprintf("• [%s] %s (%s, %s)\n", c.KodeMatkul, c.NamaMatkul, c.Hari, c.Jam))
				}
				sb.WriteString("\nSilakan perjelas nama sesi (contoh: `!kosong aljabar teori`).")
				return sb.String()
			}
			return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan.", matkulQuery)
		}

		targetDate := now
		if !strings.EqualFold(item.Hari, util.GetHariIndonesia(now)) {
			targetDate = util.GetDateForDayName(item.Hari, now)
		}
		alasan := ""

		if len(segments) > 1 {
			val := strings.TrimSpace(segments[1])
			if strings.Contains(strings.ToLower(val), "besok") || strings.Contains(strings.ToLower(val), "hari ini") || util.IsDayName(val) || util.IsDatePattern(val) {
				targetDate = ParseOverrideDate(val, now)
				if len(segments) > 2 {
					alasan = strings.TrimSpace(segments[2])
				}
			} else {
				alasan = val
			}
		}

		override, err := om.AddCancel(scopeJID, *item, targetDate, alasan, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal meniadakan jadwal: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *KULIAH DITANDAI DITIADAKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Tanggal     : %s (%s WIB)\n", override.TargetDate, override.OrigJam))
		if override.Alasan != "" {
			sb.WriteString(fmt.Sprintf("• Keterangan  : %s\n", override.Alasan))
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Jadwal pada tanggal tersebut akan dicoret. Ketik `!batalganti %d` untuk mengaktifkan kembali._", override.ID))
		return sb.String()

	case "libur", "holiday":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, pengumuman libur harian hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		dayOrDate := strings.TrimSpace(segments[0])
		if dayOrDate == "" {
			return "⚠️ *Format Perintah Libur Kurang Tepat*\n\n" +
				"Gunakan format pemisah pipa `|`:\n" +
				"`!libur [Hari/Tanggal] | [Keterangan/Nama Libur]`\n\n" +
				"*Contoh:*\n" +
				"• `!libur besok | Hari Kemerdekaan RI`\n" +
				"• `!libur senin | Libur Nasional Maulid Nabi`\n" +
				"• `!libur 17-08-2026 | HUT RI`"
		}

		alasan := "Libur Perkuliahan"
		if len(segments) > 1 {
			if trimmed := strings.TrimSpace(segments[1]); trimmed != "" {
				alasan = trimmed
			}
		}

		targetDate := ParseOverrideDate(dayOrDate, now)

		override, err := om.AddHoliday(scopeJID, targetDate, alasan, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menetapkan hari libur: %v", err)
		}

		hariTgt := util.GetHariIndonesia(targetDate)
		tglTgt := targetDate.Format("02-01-2006")

		var sb strings.Builder
		sb.WriteString("🌴 *PENGUMUMAN LIBUR BERHASIL DITETAPKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan : #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Tanggal Libur: %s, %s (Seharian)\n", hariTgt, tglTgt))
		sb.WriteString(fmt.Sprintf("• Keterangan   : %s\n", override.Alasan))
		sb.WriteString("──────────\n")
		sb.WriteString("✨ Seluruh perkuliahan pada hari tersebut otomatis ditiadakan.\n")
		sb.WriteString("⏰ Pengingat pagi pukul 06:00 WIB otomatis mengirimkan ucapan selamat libur.\n")
		sb.WriteString(fmt.Sprintf("_Ketik `!batalganti %d` jika ingin membatalkan status libur._", override.ID))
		return sb.String()

	case "kuliahganti", "tambahkelas", "extraclass":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penambahan kuliah pengganti hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		isForce := false
		var cleanSegments []string
		for _, seg := range segments {
			trimmed := strings.TrimSpace(seg)
			if strings.EqualFold(trimmed, "paksa") || strings.EqualFold(trimmed, "force") {
				isForce = true
			} else {
				cleanSegments = append(cleanSegments, trimmed)
			}
		}
		segments = cleanSegments

		if len(segments) < 2 {
			return "⚠️ *Format Kuliah Pengganti Kurang Tepat*\n\n" +
				"Gunakan format:\n" +
				"`!kuliahganti [Matkul] | [Hari/Tanggal & Jam] | [Ruang (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!kuliahganti matdis | sabtu 09:00 - 11:30 | D105`\n" +
				"• `!kuliahganti sbd | sabtu 13:00 | Lab 312`"
		}

		matkulQuery := strings.TrimSpace(segments[0])
		timeRaw := strings.TrimSpace(segments[1])
		ruangBaru := ""
		if len(segments) > 2 {
			ruangBaru = strings.TrimSpace(segments[2])
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				item = &candidates[0]
			} else {
				return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan.", matkulQuery)
			}
		}

		targetDate := ParseOverrideDate(timeRaw, now)
		baseDuration := util.CalculateDurationInMinutes(item.Jam)
		newJam := util.AutoCompleteJamRange(timeRaw, baseDuration)

		conflict := om.CheckScheduleConflict(scopeJID, targetDate, newJam, nil, cfg)
		if conflict != nil && !isForce {
			hariTgt := util.GetHariIndonesia(targetDate)
			tglTgt := targetDate.Format("02-01-2006")
			var sb strings.Builder
			sb.WriteString("⚠️ *PERINGATAN BENTROK JADWAL!*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("Waktu kuliah pengganti (*%s, %s, %s WIB*) bertabrakan dengan jadwal:\n\n", hariTgt, tglTgt, newJam))
			sb.WriteString(fmt.Sprintf("• *%s*\n", conflict.Matkul))
			sb.WriteString(fmt.Sprintf("  └ Jam   : %s WIB\n", conflict.Jam))
			sb.WriteString(fmt.Sprintf("  └ Ruang : %s\n", conflict.Ruang))
			if conflict.Dosen != "" {
				sb.WriteString(fmt.Sprintf("  └ Dosen : %s\n", conflict.Dosen))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("Kuliah pengganti tidak ditambahkan untuk mencegah jadwal kuliah ganda.\n\n")
			sb.WriteString("💡 *Apakah tetap ingin menambahkan?*\n")
			sb.WriteString(fmt.Sprintf("1. Pilih jam lain yang kosong (ketik `!%s` untuk cek jam kosong).\n", strings.ToLower(hariTgt)))
			sb.WriteString("2. Jika jam tersebut memang disepakati, tambahkan kata `paksa` di akhir:\n")
			sb.WriteString(fmt.Sprintf("   `!kuliahganti %s | %s | paksa`", matkulQuery, timeRaw))
			return sb.String()
		}

		if ruangBaru == "" {
			ruangBaru = item.Ruang
		}

		override, err := om.AddExtra(scopeJID, *item, targetDate, newJam, ruangBaru, "Kuliah Pengganti", senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menambahkan kuliah pengganti: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *KULIAH PENGGANTI DITAMBAHKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Tanggal     : %s (%s WIB)\n", override.TargetDate, override.NewJam))
		sb.WriteString(fmt.Sprintf("• Ruang       : %s\n", override.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen       : %s (%s)\n", item.Dosen, item.InisialDosen))
		if isForce {
			sb.WriteString("• Peringatan  : ⚠️ *Kuliah pengganti ditambahkan dengan konfirmasi bentrok (dipaksa oleh Admin).*\n")
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Bot akan otomatis menyertakan jadwal ini pada pengingat. Ketik `!batalganti %d` untuk menghapus._", override.ID))
		return sb.String()

	case "jadwalganti", "overrides", "listganti":
		list, err := om.GetActiveOverrides(scopeJID, now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat daftar jadwal pengganti: %v", err)
		}
		return om.FormatActiveOverrides(list)

	case "batalganti", "hapusganti", "rmganti":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, pembatalan jadwal pengganti hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		id, err := strconv.Atoi(payload)
		if err != nil || id <= 0 {
			return "⚠️ Sertakan ID perubahan yang ingin dibatalkan.\nContoh: `!batalganti 1`\n\nKetik `!jadwalganti` untuk melihat daftar ID perubahan aktif."
		}

		ok, err := om.CancelOverride(scopeJID, id)
		if err != nil {
			return fmt.Sprintf("❌ Gagal membatalkan jadwal pengganti: %v", err)
		}
		if !ok {
			return fmt.Sprintf("ℹ️ Jadwal pengganti dengan ID #%d tidak ditemukan.", id)
		}

		return fmt.Sprintf("🎉 *JADWAL PENGGANTI DIBATALKAN*\nPerubahan dengan ID #%d telah dihapus. Jadwal pada tanggal tersebut kembali normal.", id)

	default:
		var sb strings.Builder
		sb.WriteString("📖 *PANDUAN JADWAL PENGGANTI (OVERRIDE)*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!pindah [Matkul] | [Waktu Baru] | [Ruang]`\n")
		sb.WriteString("  ➔ Memindahkan jam/hari kuliah sementara\n")
		sb.WriteString("  _Cth: `!pindah aljabar | besok 13:00 | Lab 312`_\n\n")
		sb.WriteString("• `!kosong [Matkul] | [Hari/Tanggal] | [Alasan]`\n")
		sb.WriteString("  ➔ Menandai kuliah ditiadakan/kosong sementara\n")
		sb.WriteString("  _Cth: `!kosong sbd | besok | Dosen dinas luar`_\n\n")
		sb.WriteString("• `!kuliahganti [Matkul] | [Waktu] | [Ruang]`\n")
		sb.WriteString("  ➔ Menambah kuliah pengganti di hari lain\n")
		sb.WriteString("  _Cth: `!kuliahganti matdis | sabtu 09:00 | D105`_\n\n")
		sb.WriteString("• `!jadwalganti`\n")
		sb.WriteString("  ➔ Melihat daftar perubahan jadwal aktif\n\n")
		sb.WriteString("• `!batalganti [ID]`\n")
		sb.WriteString("  ➔ Membatalkan perubahan (kembali ke normal)\n")
		sb.WriteString("──────────\n")
		return sb.String()
	}
}
