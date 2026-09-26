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
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var ErrUnmappedScope = academic.ErrUnmappedScope

type BackfillReport struct {
	SourceTable string   `json:"source_table"`
	TotalRows   int      `json:"total_rows"`
	SuccessRows int      `json:"success_rows"`
	FailedRows  int      `json:"failed_rows"`
	Errors      []string `json:"errors,omitempty"`
}

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
	_, _ = om.BackfillLegacyOverridesContext(context.Background())

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

func parseTimeRangeUTC(dateStr, jamStr string, loc *time.Location) (startsAt, endsAt string, err error) {
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	start, end := parseJam(jamStr)
	startTime, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+start, loc)
	if err != nil {
		return "", "", fmt.Errorf("gagal parsing jam mulai %s: %w", start, err)
	}
	endTime, err := time.ParseInLocation("2006-01-02 15:04", dateStr+" "+end, loc)
	if err != nil {
		return "", "", fmt.Errorf("gagal parsing jam selesai %s: %w", end, err)
	}
	return startTime.UTC().Format(time.RFC3339), endTime.UTC().Format(time.RFC3339), nil
}

func parseTimeRange(dateStr, jamStr string) (startsAt, endsAt string) {
	startsAt, endsAt, _ = parseTimeRangeUTC(dateStr, jamStr, time.FixedZone("WIB", 7*3600))
	return startsAt, endsAt
}

func parseJamFromTimes(startsAt, endsAt string, loc *time.Location) string {
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	var startTimeStr, endTimeStr string
	if t, err := time.Parse(time.RFC3339, startsAt); err == nil {
		startTimeStr = t.In(loc).Format("15:04")
	} else if t, err := time.ParseInLocation("2006-01-02 15:04:05", startsAt, time.UTC); err == nil {
		startTimeStr = t.In(loc).Format("15:04")
	} else if len(startsAt) >= 16 && strings.Contains(startsAt, "T") {
		parts := strings.Split(startsAt, "T")
		if len(parts) > 1 && len(parts[1]) >= 5 {
			startTimeStr = parts[1][:5]
		}
	} else if len(startsAt) >= 16 && strings.Contains(startsAt, " ") {
		parts := strings.Split(startsAt, " ")
		if len(parts) > 1 && len(parts[1]) >= 5 {
			startTimeStr = parts[1][:5]
		}
	}

	if t, err := time.Parse(time.RFC3339, endsAt); err == nil {
		endTimeStr = t.In(loc).Format("15:04")
	} else if t, err := time.ParseInLocation("2006-01-02 15:04:05", endsAt, time.UTC); err == nil {
		endTimeStr = t.In(loc).Format("15:04")
	} else if len(endsAt) >= 16 && strings.Contains(endsAt, "T") {
		parts := strings.Split(endsAt, "T")
		if len(parts) > 1 && len(parts[1]) >= 5 {
			endTimeStr = parts[1][:5]
		}
	} else if len(endsAt) >= 16 && strings.Contains(endsAt, " ") {
		parts := strings.Split(endsAt, " ")
		if len(parts) > 1 && len(parts[1]) >= 5 {
			endTimeStr = parts[1][:5]
		}
	}

	if startTimeStr != "" && endTimeStr != "" {
		if startTimeStr == "00:00" && (endTimeStr == "23:59" || endTimeStr == "00:00") {
			return ""
		}
		return startTimeStr + " - " + endTimeStr
	}
	return ""
}

// AddRescheduleContext menambahkan perubahan jadwal kuliah ke tanggal/jam lain dengan context
func (om *OverrideManager) AddRescheduleContext(
	ctx context.Context, scopeJID string, item JadwalItem, origDate, targetDate time.Time, newJam, newRuang, createdBy string,
) (*ScheduleOverride, error) {
	ruang := newRuang
	if ruang == "" {
		ruang = item.Ruang
	}

	origDateStr := origDate.Format("2006-01-02")
	targetDateStr := targetDate.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, ErrUnmappedScope)
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)

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

	offeringID, err := om.academicRepo.EnsureCourseOfferingForDate(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI", targetDate)
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

	startsAt, endsAt, err := parseTimeRangeUTC(targetDateStr, newJam, loc)
	if err != nil {
		return nil, err
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	queryEvent := `
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES (?, ?, 'REPLACEMENT', ?, ?, ?, '', 'PUBLISHED', ?, ?)
		RETURNING id;
	`
	var id int64
	err = tx.QueryRowContext(ctx, queryEvent, patternID, origDateStr, startsAt, endsAt, newRoomID, userID, nowStr).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (REPLACEMENT): %w", err)
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

// AddReschedule adapter dengan timeout bawaan
func (om *OverrideManager) AddReschedule(
	scopeJID string, item JadwalItem, origDate, targetDate time.Time, newJam, newRuang, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.AddRescheduleContext(ctx, scopeJID, item, origDate, targetDate, newJam, newRuang, createdBy)
}

// AddCancelContext menandai kuliah ditiadakan pada tanggal tertentu dengan context
func (om *OverrideManager) AddCancelContext(
	ctx context.Context, scopeJID string, item JadwalItem, targetDate time.Time, alasan, createdBy string,
) (*ScheduleOverride, error) {
	dateStr := targetDate.Format("2006-01-02")

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, ErrUnmappedScope)
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)

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

	offeringID, err := om.academicRepo.EnsureCourseOfferingForDate(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI", targetDate)
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

	startsAt, endsAt, err := parseTimeRangeUTC(dateStr, item.Jam, loc)
	if err != nil {
		return nil, err
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	queryEvent := `
		INSERT INTO teaching_events (
			origin_schedule_pattern_id, origin_occurrence_date, event_kind,
			starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES (?, ?, 'SESSION_CANCELLED', ?, ?, ?, ?, 'PUBLISHED', ?, ?)
		RETURNING id;
	`
	var id int64
	err = tx.QueryRowContext(ctx, queryEvent, patternID, dateStr, startsAt, endsAt, roomID, alasan, userID, nowStr).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (SESSION_CANCELLED): %w", err)
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

// AddCancel adapter dengan timeout bawaan
func (om *OverrideManager) AddCancel(
	scopeJID string, item JadwalItem, targetDate time.Time, alasan, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.AddCancelContext(ctx, scopeJID, item, targetDate, alasan, createdBy)
}

// AddExtraContext menambahkan jadwal kuliah pengganti/ekstra di hari tertentu dengan context
func (om *OverrideManager) AddExtraContext(
	ctx context.Context, scopeJID string, item JadwalItem, targetDate time.Time, jam, ruang, alasan, createdBy string,
) (*ScheduleOverride, error) {
	dateStr := targetDate.Format("2006-01-02")
	if ruang == "" {
		ruang = item.Ruang
	}

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, ErrUnmappedScope)
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)

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

	offeringID, err := om.academicRepo.EnsureCourseOfferingForDate(ctx, classID, item.KodeMatkul, item.NamaMatkul, "TEORI", targetDate)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering: %w", err)
	}

	if item.Dosen != "" || item.InisialDosen != "" {
		lecID, err := om.academicRepo.EnsureLecturer(ctx, item.InisialDosen, item.Dosen)
		if err == nil && lecID > 0 {
			_ = om.academicRepo.EnsureOfferingLecturer(ctx, offeringID, lecID)
		}
	}

	startsAt, endsAt, err := parseTimeRangeUTC(dateStr, jam, loc)
	if err != nil {
		return nil, err
	}
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	queryEvent := `
		INSERT INTO teaching_events (
			event_kind, starts_at, ends_at, room_id, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES ('EXTRA', ?, ?, ?, ?, 'PUBLISHED', ?, ?)
		RETURNING id;
	`
	var id int64
	err = tx.QueryRowContext(ctx, queryEvent, startsAt, endsAt, roomID, alasan, userID, nowStr).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (EXTRA): %w", err)
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

// AddExtra adapter dengan timeout bawaan
func (om *OverrideManager) AddExtra(
	scopeJID string, item JadwalItem, targetDate time.Time, jam, ruang, alasan, createdBy string,
) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.AddExtraContext(ctx, scopeJID, item, targetDate, jam, ruang, alasan, createdBy)
}

// GetOverridesForDateContext mengambil seluruh catatan override yang mempengaruhi tanggal tertentu dengan context
func (om *OverrideManager) GetOverridesForDateContext(ctx context.Context, scopeJID string, date time.Time) ([]ScheduleOverride, error) {
	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, nil
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)
	dateStr := date.Format("2006-01-02")
	tLocal, _ := time.ParseInLocation("2006-01-02", dateStr, loc)
	startOfDayUTC := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
	endOfDayUTC := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 23, 59, 59, 999999999, loc).UTC().Format(time.RFC3339)

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
		  AND (te.origin_occurrence_date = ? OR (te.starts_at >= ? AND te.starts_at <= ?))
		GROUP BY te.id
		ORDER BY te.id ASC;
	`
	rows, err := om.db.QueryContext(ctx, query, classID, dateStr, startOfDayUTC, endOfDayUTC)
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
		if t, err := time.Parse(time.RFC3339, startsAt); err == nil {
			targetDate = t.In(loc).Format("2006-01-02")
		} else if len(startsAt) >= 10 {
			targetDate = startsAt[:10]
		}

		origDate := origOccDate
		if origDate == "" {
			origDate = targetDate
		}

		newJam := parseJamFromTimes(startsAt, endsAt, loc)
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

// GetOverridesForDate adapter dengan timeout bawaan
func (om *OverrideManager) GetOverridesForDate(scopeJID string, date time.Time) ([]ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.GetOverridesForDateContext(ctx, scopeJID, date)
}

// GetActiveOverridesContext mengambil semua override yang tanggal targetnya belum lewat dengan context
func (om *OverrideManager) GetActiveOverridesContext(ctx context.Context, scopeJID string, now time.Time) ([]ScheduleOverride, error) {
	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, nil
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)
	todayStr := now.Format("2006-01-02")
	tLocal, _ := time.ParseInLocation("2006-01-02", todayStr, loc)
	startOfDayUTC := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)

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
		  AND (te.starts_at >= ? OR te.origin_occurrence_date >= ?)
		GROUP BY te.id
		ORDER BY date(te.starts_at) ASC, te.id ASC;
	`
	rows, err := om.db.QueryContext(ctx, query, classID, startOfDayUTC, todayStr)
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
		if t, err := time.Parse(time.RFC3339, startsAt); err == nil {
			targetDate = t.In(loc).Format("2006-01-02")
		} else if len(startsAt) >= 10 {
			targetDate = startsAt[:10]
		}

		origDate := origOccDate
		if origDate == "" {
			origDate = targetDate
		}

		newJam := parseJamFromTimes(startsAt, endsAt, loc)
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

// GetActiveOverrides adapter dengan timeout bawaan
func (om *OverrideManager) GetActiveOverrides(scopeJID string, now time.Time) ([]ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.GetActiveOverridesContext(ctx, scopeJID, now)
}

// CancelOverrideContext membatalkan override aktif dengan context dan atribusi actor yang benar
func (om *OverrideManager) CancelOverrideContext(ctx context.Context, scopeJID string, id int, actorJID string) (bool, error) {
	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return false, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, ErrUnmappedScope)
	}

	actorKey := strings.TrimSpace(actorJID)
	if actorKey == "" {
		actorKey = "system"
	}

	actorUserID, err := om.academicRepo.EnsureUser(ctx, actorKey, actorKey)
	if err != nil {
		return false, fmt.Errorf("gagal memastikan user untuk actor %s: %w", actorKey, err)
	}

	nowStr := time.Now().UTC().Format(time.RFC3339)

	res, err := om.db.ExecContext(ctx, `
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
	`, actorUserID, nowStr, nowStr, id, classID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	return affected > 0, err
}

// CancelOverride adapter dengan variadic actorJID dan timeout bawaan
func (om *OverrideManager) CancelOverride(scopeJID string, id int, actorJID ...string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	actor := "system"
	if len(actorJID) > 0 && actorJID[0] != "" {
		actor = actorJID[0]
	}
	return om.CancelOverrideContext(ctx, scopeJID, id, actor)
}

// AddHolidayContext menambahkan pengumuman libur harian dengan context
func (om *OverrideManager) AddHolidayContext(ctx context.Context, scopeJID string, targetDate time.Time, alasan, createdBy string) (*ScheduleOverride, error) {
	tglStr := targetDate.Format("2006-01-02")
	if alasan == "" {
		alasan = "Libur Perkuliahan"
	}

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil, fmt.Errorf("gagal memetakan kelas untuk scope %s: %w", scopeJID, ErrUnmappedScope)
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)

	userID, err := om.academicRepo.EnsureUser(ctx, createdBy, createdBy)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan user %s: %w", createdBy, err)
	}

	offeringID, err := om.academicRepo.EnsureCourseOfferingForDate(ctx, classID, "LIBUR", "LIBUR SEHARIAN", "TEORI", targetDate)
	if err != nil {
		return nil, fmt.Errorf("gagal memastikan course_offering libur: %w", err)
	}

	tLocal, _ := time.ParseInLocation("2006-01-02", tglStr, loc)
	startsAt := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
	endsAt := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 23, 59, 59, 0, loc).UTC().Format(time.RFC3339)
	nowStr := time.Now().UTC().Format(time.RFC3339)

	tx, err := om.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("gagal memulai transaksi: %w", err)
	}
	defer tx.Rollback()

	queryEvent := `
		INSERT INTO teaching_events (
			event_kind, starts_at, ends_at, reason, lifecycle_status,
			published_by_user_id, published_at
		) VALUES ('HOLIDAY', ?, ?, ?, 'PUBLISHED', ?, ?)
		RETURNING id;
	`
	var id int64
	err = tx.QueryRowContext(ctx, queryEvent, startsAt, endsAt, alasan, userID, nowStr).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("gagal menyisipkan teaching_events (HOLIDAY): %w", err)
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

// AddHoliday adapter dengan timeout bawaan
func (om *OverrideManager) AddHoliday(scopeJID string, targetDate time.Time, alasan string, createdBy string) (*ScheduleOverride, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return om.AddHolidayContext(ctx, scopeJID, targetDate, alasan, createdBy)
}

func (om *OverrideManager) GetHolidayOverride(scopeJID string, date time.Time) *ScheduleOverride {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, scopeJID)
	if err != nil || classID == 0 {
		return nil
	}

	loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)
	tglStr := date.Format("2006-01-02")
	tLocal, _ := time.ParseInLocation("2006-01-02", tglStr, loc)
	startOfDayUTC := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
	endOfDayUTC := time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 23, 59, 59, 999999999, loc).UTC().Format(time.RFC3339)

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
		  AND (te.starts_at >= ? AND te.starts_at <= ?)
		ORDER BY te.id DESC
		LIMIT 1;
	`
	row := om.db.QueryRowContext(ctx, query, classID, startOfDayUTC, endOfDayUTC)

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

// BackfillLegacyOverridesContext menyalin data dari tabel legacy schedule_overrides ke teaching_events dengan manifest dan error tracking
func (om *OverrideManager) BackfillLegacyOverridesContext(ctx context.Context) (*BackfillReport, error) {
	report := &BackfillReport{
		SourceTable: "schedule_overrides",
	}

	var tblName string
	err := om.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name='schedule_overrides';").Scan(&tblName)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return report, nil
	}

	rows, err := om.db.QueryContext(ctx, `
		SELECT id, scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
		       orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by, created_at
		FROM schedule_overrides;
	`)
	if err != nil {
		return report, fmt.Errorf("gagal query schedule_overrides: %w", err)
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
		); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("Gagal scan baris: %v", err))
			report.FailedRows++
			continue
		}
		legacyRows = append(legacyRows, lr)
	}
	_ = rows.Close()

	report.TotalRows = len(legacyRows)
	if report.TotalRows == 0 {
		return report, nil
	}

	batchID, batchErr := om.academicRepo.EnsureMigrationBatch(ctx, "LEGACY_SCHEDULE_OVERRIDES")

	for _, lr := range legacyRows {
		classID, _, err := om.academicRepo.ResolveClassIDFromScope(ctx, lr.scopeJID)
		if err != nil || classID == 0 {
			errMsg := fmt.Sprintf("Baris ID %d: scope %s tidak terpetakan ke kelas terdaftar", lr.id, lr.scopeJID)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "scope_jid", "UNMAPPED_SCOPE", errMsg)
			}
			continue
		}

		loc, _ := om.academicRepo.GetClassTimezone(ctx, classID)

		userID, err := om.academicRepo.EnsureUser(ctx, lr.createdBy, lr.createdBy)
		if err != nil {
			errMsg := fmt.Sprintf("Baris ID %d: gagal memastikan user %s: %v", lr.id, lr.createdBy, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "created_by", "USER_ERROR", errMsg)
			}
			continue
		}

		var roomID *int64
		if lr.ruang != "" && lr.ruang != "-" {
			rid, err := om.academicRepo.EnsureRoom(ctx, lr.ruang, lr.ruang)
			if err == nil {
				roomID = &rid
			}
		}

		targetTime, _ := time.Parse("2006-01-02", lr.targetDate)
		if targetTime.IsZero() {
			targetTime = time.Now()
		}

		offeringID, err := om.academicRepo.EnsureCourseOfferingForDate(ctx, classID, lr.kodeMatkul, lr.namaMatkul, "TEORI", targetTime)
		if err != nil {
			errMsg := fmt.Sprintf("Baris ID %d: gagal memastikan course_offering: %v", lr.id, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "course_offering", "OFFERING_ERROR", errMsg)
			}
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
			startsAt, endsAt, err = parseTimeRangeUTC(lr.targetDate, lr.newJam, loc)
			if err != nil {
				startsAt, endsAt = parseTimeRange(lr.targetDate, lr.newJam)
			}

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
			startsAt, endsAt, err = parseTimeRangeUTC(lr.targetDate, lr.origJam, loc)
			if err != nil {
				startsAt, endsAt = parseTimeRange(lr.targetDate, lr.origJam)
			}

		case "EXTRA":
			eventKind = "EXTRA"
			var errExtra error
			startsAt, endsAt, errExtra = parseTimeRangeUTC(lr.targetDate, lr.newJam, loc)
			if errExtra != nil {
				startsAt, endsAt = parseTimeRange(lr.targetDate, lr.newJam)
			}

		case "HOLIDAY":
			eventKind = "HOLIDAY"
			tLocal, _ := time.ParseInLocation("2006-01-02", lr.targetDate, loc)
			startsAt = time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 0, 0, 0, 0, loc).UTC().Format(time.RFC3339)
			endsAt = time.Date(tLocal.Year(), tLocal.Month(), tLocal.Day(), 23, 59, 59, 0, loc).UTC().Format(time.RFC3339)

		default:
			errMsg := fmt.Sprintf("Baris ID %d: tipe override %s tidak valid", lr.id, lr.oType)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "override_type", "INVALID_TYPE", errMsg)
			}
			continue
		}

		var existingID int64
		err = om.db.QueryRowContext(ctx, `
			SELECT te.id FROM teaching_events te
			JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
			WHERE teo.course_offering_id = ? AND te.event_kind = ? AND te.starts_at = ?
		`, offeringID, eventKind, startsAt).Scan(&existingID)
		if err == nil && existingID > 0 {
			report.SuccessRows++
			continue
		}

		tx, err := om.db.BeginTx(ctx, nil)
		if err != nil {
			errMsg := fmt.Sprintf("Baris ID %d: gagal memulai tx: %v", lr.id, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "transaction", "TX_ERROR", errMsg)
			}
			continue
		}

		queryInsertEvent := `
			INSERT INTO teaching_events (
				origin_schedule_pattern_id, origin_occurrence_date, event_kind,
				starts_at, ends_at, room_id, reason, lifecycle_status,
				published_by_user_id, published_at, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, 'PUBLISHED', ?, ?, ?)
			RETURNING id;
		`
		var evID int64
		err = tx.QueryRowContext(ctx, queryInsertEvent, originPatternID, originOccDate, eventKind, startsAt, endsAt, roomID, lr.alasan, userID, lr.createdAt, lr.createdAt).Scan(&evID)
		if err != nil {
			_ = tx.Rollback()
			errMsg := fmt.Sprintf("Baris ID %d: gagal insert teaching_events: %v", lr.id, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "teaching_events", "INSERT_ERROR", errMsg)
			}
			continue
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO teaching_event_offerings (
				teaching_event_id, course_offering_id, participation_role, participation_status
			) VALUES (?, ?, 'OWNER', 'ACCEPTED')
		`, evID, offeringID)
		if err != nil {
			_ = tx.Rollback()
			errMsg := fmt.Sprintf("Baris ID %d: gagal insert teaching_event_offerings: %v", lr.id, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "teaching_event_offerings", "INSERT_ERROR", errMsg)
			}
			continue
		}

		if err := tx.Commit(); err != nil {
			errMsg := fmt.Sprintf("Baris ID %d: gagal commit tx: %v", lr.id, err)
			report.Errors = append(report.Errors, errMsg)
			report.FailedRows++
			if batchErr == nil {
				_ = om.academicRepo.RecordImportError(ctx, batchID, fmt.Sprintf("row:%d", lr.id), "transaction", "COMMIT_ERROR", errMsg)
			}
			continue
		}

		report.SuccessRows++
	}

	if batchErr == nil {
		summaryJSON := fmt.Sprintf(`{"total_rows":%d,"applied_rows":%d,"failed_rows":%d}`, report.TotalRows, report.SuccessRows, report.FailedRows)
		_ = om.academicRepo.UpdateImportBatchSummary(ctx, batchID, summaryJSON)
	}

	return report, nil
}

// BackfillLegacyOverrides adapter
func (om *OverrideManager) BackfillLegacyOverrides() (*BackfillReport, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return om.BackfillLegacyOverridesContext(ctx)
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

	switch cmd {
	case "pindah", "ganti", "reschedule":
		return util.DashboardRedirectNotice("jadwal kuliah")

	case "kosong", "batal", "cancel":
		return util.DashboardRedirectNotice("jadwal kuliah")

	case "libur", "holiday":
		return util.DashboardRedirectNotice("jadwal kuliah")

	case "kuliahganti", "tambahkelas", "extraclass":
		return util.DashboardRedirectNotice("jadwal kuliah")

	case "jadwalganti", "overrides", "listganti":
		list, err := om.GetActiveOverrides(scopeJID, now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat daftar jadwal pengganti: %v", err)
		}
		return om.FormatActiveOverrides(list)

	case "batalganti", "hapusganti", "rmganti":
		return util.DashboardRedirectNotice("jadwal kuliah")

	default:
		var sb strings.Builder
		sb.WriteString("📖 *JADWAL PENGGANTI (OVERRIDE)*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!jadwalganti`\n")
		sb.WriteString("  ➔ Melihat daftar perubahan jadwal aktif\n\n")
		sb.WriteString("──────────\n")
		sb.WriteString("⚠️ *Penambahan, perubahan, dan pembatalan jadwal kuliah kini hanya melalui Web Dashboard Pengelola:*\n")
		sb.WriteString("👉 http://localhost:8080/app.html (atau domain portal Anda)\n")
		sb.WriteString("──────────\n")
		return sb.String()
	}
}
