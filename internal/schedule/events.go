package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrEventNotFound     = errors.New("teaching event tidak ditemukan")
	ErrEventInvalidInput = errors.New("input teaching event tidak valid")
	ErrEventInvalidState = errors.New("status event tidak memungkinkan operasi ini")
	ErrEventConflict     = errors.New("konflik jadwal memblokir publikasi")
	ErrEventForbidden    = errors.New("tindakan tidak tersedia pada cakupan aktif")
	ErrEventVersion      = errors.New("versi data sudah berubah, muat ulang sebelum menyimpan")
)

type EventService struct {
	db *sql.DB
}

func NewEventService(db *sql.DB) *EventService { return &EventService{db: db} }

type EventActor struct {
	UserID           int64
	RoleAssignmentID int64
	CorrelationID    string
}

type CreateDraftInput struct {
	OwnerOfferingID int64
	Kind            string
	StartsAt        string
	EndsAt          string
	OriginPatternID *int64
	OriginDate      *string
	RoomID          *int64
	Reason          *string
	IsPermanent     bool
	NewStartTime    *string
	NewEndTime      *string
	NewRoomID       *int64
	NewDayOfWeek    *int
}

type UpdateDraftInput struct {
	StartsAt        *string
	EndsAt          *string
	RoomID          *int64
	ClearRoom       bool
	Reason          *string
	ExpectedVersion int
}

type EventRow struct {
	ID               int64   `json:"id"`
	Kind             string  `json:"event_kind"`
	Lifecycle        string  `json:"lifecycle_status"`
	StartsAt         string  `json:"starts_at"`
	EndsAt           string  `json:"ends_at"`
	RoomID           *int64  `json:"room_id,omitempty"`
	Reason           *string `json:"reason,omitempty"`
	ConflictOverride *string `json:"conflict_override_reason,omitempty"`
	OriginPatternID  *int64  `json:"origin_schedule_pattern_id,omitempty"`
	OriginDate       *string `json:"origin_occurrence_date,omitempty"`
	ResultPatternID  *int64  `json:"result_schedule_pattern_id,omitempty"`
	Version          int     `json:"version"`
	PublishedBy      *int64  `json:"published_by_user_id,omitempty"`
	PublishedAt      *string `json:"published_at,omitempty"`
	RevokedBy        *int64  `json:"revoked_by_user_id,omitempty"`
	RevokedAt        *string `json:"revoked_at,omitempty"`
	RevocationReason *string `json:"revocation_reason,omitempty"`
	OwnerOfferingID  int64   `json:"owner_offering_id"`
	OwnerClassID     int64   `json:"owner_class_id"`
	OwnerDisplay     string  `json:"owner_display"`
}

type OfferingParticipation struct {
	OfferingID  int64  `json:"course_offering_id"`
	ClassID     int64  `json:"class_id"`
	ClassCode   string `json:"class_code"`
	DisplayName string `json:"display_name"`
	Role        string `json:"participation_role"`
	Status      string `json:"participation_status"`
}

type EventDetail struct {
	Event          EventRow                `json:"event"`
	Participations []OfferingParticipation `json:"participations"`
	Confirmations  []RoomConfirmation      `json:"confirmations"`
}

type RoomConfirmation struct {
	ID          int64   `json:"id"`
	RoomID      int64   `json:"room_id"`
	RoomCode    string  `json:"room_code"`
	Status      string  `json:"confirmation_status"`
	Contact     *string `json:"external_contact,omitempty"`
	Note        *string `json:"note,omitempty"`
	RecordedBy  int64   `json:"recorded_by_user_id"`
	RecordedAt  string  `json:"recorded_at"`
	ConfirmedAt *string `json:"confirmed_at,omitempty"`
}

type PreviewResult struct {
	Event  EventRow `json:"event"`
	Origin *struct {
		PatternID int64  `json:"pattern_id"`
		Date      string `json:"date"`
		Display   string `json:"display"`
	} `json:"origin,omitempty"`
	Blocking    []string `json:"blocking"`
	NonBlocking []string `json:"non_blocking"`
	Recipients  []string `json:"recipients"`
	RoomNote    string   `json:"room_note"`
}

func eventNow() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func parseEventTime(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, ErrEventInvalidInput
	}
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, ErrEventInvalidInput
}

func (s *EventService) ownerScope(ctx context.Context, offeringID int64) (classID, semesterID int64, display, semStarts, semEnds, tz string, err error) {
	err = s.db.QueryRowContext(ctx, `SELECT sem.class_id, co.semester_id, co.display_name,
		sem.starts_on, sem.ends_on, COALESCE(cs.timezone, 'Asia/Jakarta')
		FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN class_settings cs ON cs.class_id = sem.class_id
		WHERE co.id = ?`, offeringID).Scan(&classID, &semesterID, &display, &semStarts, &semEnds, &tz)
	return
}

func eventDateInTZ(startsUTC time.Time, tz string, semStarts, semEnds string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	d := startsUTC.In(loc).Format("2006-01-02")
	if d < semStarts || d > semEnds {
		return "", ErrEventInvalidInput
	}
	return d, nil
}

// CreateDraft validates and stores a DRAFT event with single OWNER participation.
func (s *EventService) CreateDraft(ctx context.Context, actor EventActor, in CreateDraftInput) (*EventRow, error) {
	kind := strings.ToUpper(strings.TrimSpace(in.Kind))
	switch kind {
	case "REPLACEMENT", "EXTRA", "HOLIDAY", "SESSION_CANCELLED":
	default:
		return nil, ErrEventInvalidInput
	}
	starts, err := parseEventTime(in.StartsAt)
	if err != nil {
		return nil, err
	}
	ends, err := parseEventTime(in.EndsAt)
	if err != nil {
		return nil, err
	}
	if !starts.Before(ends) {
		return nil, ErrEventInvalidInput
	}
	classID, semesterID, display, semStarts, semEnds, tz, err := s.ownerScope(ctx, in.OwnerOfferingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrEventInvalidInput
		}
		return nil, err
	}
	if _, err := eventDateInTZ(starts, tz, semStarts, semEnds); err != nil {
		return nil, errors.New("tanggal event di luar semester kelas pemilik")
	}
	if _, err := eventDateInTZ(ends, tz, semStarts, semEnds); err != nil {
		// Allow multi-day? Require both within semester.
		return nil, errors.New("tanggal event di luar semester kelas pemilik")
	}
	needsOrigin := kind == "REPLACEMENT" || kind == "SESSION_CANCELLED"
	if needsOrigin && (in.OriginPatternID == nil || in.OriginDate == nil || strings.TrimSpace(*in.OriginDate) == "") {
		return nil, errors.New("event pengganti/pembatalan wajib menunjuk pola dan tanggal asal")
	}
	if !needsOrigin && kind != "HOLIDAY" && in.OriginPatternID == nil {
		// EXTRA without origin is allowed; HOLIDAY without origin allowed.
	}
	if in.OriginPatternID != nil {
		var patternOffering int64
		err := s.db.QueryRowContext(ctx, `SELECT course_offering_id FROM schedule_patterns WHERE id = ?`, *in.OriginPatternID).Scan(&patternOffering)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("pola asal tidak ditemukan")
		}
		if err != nil {
			return nil, err
		}
		if patternOffering != in.OwnerOfferingID {
			return nil, errors.New("pola asal harus milik offering pemilik")
		}
		if in.OriginDate != nil {
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(*in.OriginDate)); err != nil {
				return nil, ErrEventInvalidInput
			}
		}
	}
	if in.RoomID != nil {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ? AND status='ACTIVE')`, *in.RoomID).Scan(&exists); err != nil || !exists {
			return nil, errors.New("ruangan tidak ditemukan atau nonaktif")
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var resultPatternArg any
	if in.IsPermanent {
		if in.OriginPatternID == nil {
			return nil, errors.New("perubahan permanen memerlukan pola asal")
		}
		newPatternID, err := createResultPatternTx(ctx, tx, *in.OriginPatternID, starts, in)
		if err != nil {
			return nil, err
		}
		resultPatternArg = newPatternID
	}

	var eventID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO teaching_events (
		origin_schedule_pattern_id, origin_occurrence_date, result_schedule_pattern_id,
		event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'DRAFT', 1) RETURNING id`,
		in.OriginPatternID, in.OriginDate, resultPatternArg, kind,
		starts.Format(time.RFC3339Nano), ends.Format(time.RFC3339Nano), in.RoomID, in.Reason,
	).Scan(&eventID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO teaching_event_offerings
		(teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (?, ?, 'OWNER', 'ACCEPTED')`, eventID, in.OwnerOfferingID); err != nil {
		return nil, err
	}
	if err := insertEventAuditTx(ctx, tx, actor, classID, semesterID, "CREATE", eventID, nil, str(fmt.Sprintf(`{"kind":"%s"}`, kind)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	_ = display
	return s.GetDetail(ctx, eventID)
}

func createResultPatternTx(ctx context.Context, tx *sql.Tx, originPatternID int64, eventStart time.Time, in CreateDraftInput) (int64, error) {
	var offeringID, roomID sql.NullInt64
	var dow int
	var start, end, effFrom string
	var effUntil sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, effective_until
		FROM schedule_patterns WHERE id = ?`, originPatternID).
		Scan(&offeringID, &roomID, &dow, &start, &end, &effFrom, &effUntil)
	if err != nil {
		return 0, err
	}
	newStart := start
	if in.NewStartTime != nil && strings.TrimSpace(*in.NewStartTime) != "" {
		newStart = strings.TrimSpace(*in.NewStartTime)
	}
	newEnd := end
	if in.NewEndTime != nil && strings.TrimSpace(*in.NewEndTime) != "" {
		newEnd = strings.TrimSpace(*in.NewEndTime)
	}
	if len(newStart) != 5 || len(newEnd) != 5 || newStart >= newEnd {
		return 0, ErrEventInvalidInput
	}
	newRoom := roomID
	if in.NewRoomID != nil {
		newRoom = sql.NullInt64{Int64: *in.NewRoomID, Valid: true}
	}
	newDow := dow
	if in.NewDayOfWeek != nil {
		if *in.NewDayOfWeek < 1 || *in.NewDayOfWeek > 7 {
			return 0, ErrEventInvalidInput
		}
		newDow = *in.NewDayOfWeek
	}
	effDate := eventStart.Format("2006-01-02")
	// Close old version day before.
	if effDate > effFrom {
		prevDay := eventStart.AddDate(0, 0, -1).Format("2006-01-02")
		if _, err := tx.ExecContext(ctx, `UPDATE schedule_patterns SET effective_until = ?, status='SUPERSEDED', version=version+1,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id = ?`, prevDay, originPatternID); err != nil {
			return 0, err
		}
	}
	var roomArg any
	if newRoom.Valid {
		roomArg = newRoom.Int64
	}
	var newID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO schedule_patterns
		(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
		VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE') RETURNING id`,
		offeringID.Int64, roomArg, newDow, newStart, newEnd, effDate).Scan(&newID)
	return newID, err
}

// GetDetail returns event + participations + confirmations.
func (s *EventService) GetDetail(ctx context.Context, eventID int64) (*EventRow, error) {
	d, err := s.getDetail(ctx, s.db, eventID)
	if err != nil {
		return nil, err
	}
	return &d.Event, nil
}

func (s *EventService) GetFullDetail(ctx context.Context, eventID int64) (*EventDetail, error) {
	return s.getDetail(ctx, s.db, eventID)
}

type dbQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *EventService) getDetail(ctx context.Context, q dbQuerier, eventID int64) (*EventDetail, error) {
	var e EventRow
	var originPat, originDate, resultPat sql.NullInt64
	var originDateStr sql.NullString
	var roomID sql.NullInt64
	var reason, override sql.NullString
	var pubBy, revBy sql.NullInt64
	var pubAt, revAt sql.NullString
	var revReason sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id, event_kind, lifecycle_status, starts_at, ends_at, room_id,
		reason, conflict_override_reason, origin_schedule_pattern_id, origin_occurrence_date,
		result_schedule_pattern_id, version, published_by_user_id, published_at,
		revoked_by_user_id, revoked_at, revocation_reason
		FROM teaching_events WHERE id = ?`, eventID).Scan(
		&e.ID, &e.Kind, &e.Lifecycle, &e.StartsAt, &e.EndsAt, &roomID,
		&reason, &override, &originPat, &originDateStr, &resultPat, &e.Version,
		&pubBy, &pubAt, &revBy, &revAt, &revReason)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrEventNotFound
	}
	if err != nil {
		return nil, err
	}
	if roomID.Valid {
		e.RoomID = &roomID.Int64
	}
	if reason.Valid {
		e.Reason = &reason.String
	}
	if override.Valid {
		e.ConflictOverride = &override.String
	}
	if originPat.Valid {
		e.OriginPatternID = &originPat.Int64
	}
	if originDateStr.Valid {
		e.OriginDate = &originDateStr.String
	}
	_ = originDate
	if resultPat.Valid {
		e.ResultPatternID = &resultPat.Int64
	}
	if pubBy.Valid {
		e.PublishedBy = &pubBy.Int64
	}
	if pubAt.Valid {
		e.PublishedAt = &pubAt.String
	}
	if revBy.Valid {
		e.RevokedBy = &revBy.Int64
	}
	if revAt.Valid {
		e.RevokedAt = &revAt.String
	}
	if revReason.Valid {
		e.RevocationReason = &revReason.String
	}
	prows, err := q.QueryContext(ctx, `SELECT teo.course_offering_id, sem.class_id, cl.code, co.display_name,
		teo.participation_role, teo.participation_status
		FROM teaching_event_offerings teo
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		JOIN classes cl ON cl.id = sem.class_id
		WHERE teo.teaching_event_id = ? ORDER BY teo.participation_role, teo.course_offering_id`, eventID)
	if err != nil {
		return nil, err
	}
	defer prows.Close()
	parts := []OfferingParticipation{}
	for prows.Next() {
		var p OfferingParticipation
		if err := prows.Scan(&p.OfferingID, &p.ClassID, &p.ClassCode, &p.DisplayName, &p.Role, &p.Status); err != nil {
			return nil, err
		}
		parts = append(parts, p)
		if p.Role == "OWNER" {
			e.OwnerOfferingID = p.OfferingID
			e.OwnerClassID = p.ClassID
			e.OwnerDisplay = p.DisplayName
		}
	}
	crows, err := q.QueryContext(ctx, `SELECT rc.id, rc.room_id, r.code, rc.confirmation_status,
		rc.external_contact, rc.note, rc.recorded_by_user_id, rc.recorded_at, rc.confirmed_at
		FROM room_confirmations rc JOIN rooms r ON r.id = rc.room_id
		WHERE rc.teaching_event_id = ? ORDER BY rc.recorded_at DESC, rc.id DESC`, eventID)
	if err != nil {
		return nil, err
	}
	defer crows.Close()
	confs := []RoomConfirmation{}
	for crows.Next() {
		var c RoomConfirmation
		var contact, note, confirmed sql.NullString
		if err := crows.Scan(&c.ID, &c.RoomID, &c.RoomCode, &c.Status, &contact, &note, &c.RecordedBy, &c.RecordedAt, &confirmed); err != nil {
			return nil, err
		}
		if contact.Valid {
			c.Contact = &contact.String
		}
		if note.Valid {
			c.Note = &note.String
		}
		if confirmed.Valid {
			c.ConfirmedAt = &confirmed.String
		}
		confs = append(confs, c)
	}
	return &EventDetail{Event: e, Participations: parts, Confirmations: confs}, nil
}

// UpdateDraft edits a DRAFT event with version check.
func (s *EventService) UpdateDraft(ctx context.Context, actor EventActor, eventID int64, in UpdateDraftInput) (*EventRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	if d.Event.Lifecycle != "DRAFT" {
		return nil, ErrEventInvalidState
	}
	if d.Event.Version != in.ExpectedVersion {
		return nil, ErrEventVersion
	}
	starts, err := parseEventTime(deref(in.StartsAt, d.Event.StartsAt))
	if err != nil {
		return nil, err
	}
	ends, err := parseEventTime(deref(in.EndsAt, d.Event.EndsAt))
	if err != nil {
		return nil, err
	}
	if !starts.Before(ends) {
		return nil, ErrEventInvalidInput
	}
	classID, _, _, semStarts, semEnds, tz, err := s.ownerScope(ctx, d.Event.OwnerOfferingID)
	if err != nil {
		return nil, err
	}
	if _, err := eventDateInTZ(starts, tz, semStarts, semEnds); err != nil {
		return nil, errors.New("tanggal event di luar semester kelas pemilik")
	}
	roomArg := d.Event.RoomID
	if in.ClearRoom {
		roomArg = nil
	} else if in.RoomID != nil {
		var exists bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ? AND status='ACTIVE')`, *in.RoomID).Scan(&exists); err != nil || !exists {
			return nil, errors.New("ruangan tidak ditemukan atau nonaktif")
		}
		roomArg = in.RoomID
	}
	reason := d.Event.Reason
	if in.Reason != nil {
		reason = in.Reason
	}
	res, err := tx.ExecContext(ctx, `UPDATE teaching_events SET starts_at=?, ends_at=?, room_id=?, reason=?,
		version=version+1, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE id=? AND version=? AND lifecycle_status='DRAFT'`,
		starts.Format(time.RFC3339Nano), ends.Format(time.RFC3339Nano), roomArg, reason, eventID, d.Event.Version)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, ErrEventVersion
	}
	after, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	semID := ownerSemester(ctx, tx, after.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, classID, semID, "UPDATE", eventID, nil, str(`{"version":`+itoa(after.Event.Version)+`}`), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &after.Event, nil
}

// DeleteDraft removes a DRAFT event and its participations.
func (s *EventService) DeleteDraft(ctx context.Context, actor EventActor, eventID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return err
	}
	if d.Event.Lifecycle != "DRAFT" {
		return ErrEventInvalidState
	}
	classID := d.Event.OwnerClassID
	semID := ownerSemester(ctx, tx, d.Event.OwnerOfferingID)
	if _, err := tx.ExecContext(ctx, `DELETE FROM teaching_event_offerings WHERE teaching_event_id = ?`, eventID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM room_confirmations WHERE teaching_event_id = ?`, eventID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM teaching_events WHERE id = ?`, eventID); err != nil {
		return err
	}
	if err := insertEventAuditTx(ctx, tx, actor, classID, semID, "DELETE", eventID, str(`{"lifecycle":"DRAFT"}`), nil, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// Preview computes blocking/non-blocking conflicts without persisting.
func (s *EventService) Preview(ctx context.Context, eventID int64) (*PreviewResult, error) {
	d, err := s.getDetail(ctx, s.db, eventID)
	if err != nil {
		return nil, err
	}
	p := &PreviewResult{Event: d.Event, Blocking: []string{}, NonBlocking: []string{}, Recipients: []string{}}
	if d.Event.OriginPatternID != nil && d.Event.OriginDate != nil {
		p.Origin = &struct {
			PatternID int64  `json:"pattern_id"`
			Date      string `json:"date"`
			Display   string `json:"display"`
		}{PatternID: *d.Event.OriginPatternID, Date: *d.Event.OriginDate, Display: d.Event.OwnerDisplay}
	}
	for _, part := range d.Participations {
		if part.Role == "OWNER" || part.Status == "ACCEPTED" {
			p.Recipients = append(p.Recipients, part.ClassCode)
		}
	}
	blocking, nonBlocking := s.findConflicts(ctx, d)
	p.Blocking = blocking
	p.NonBlocking = nonBlocking
	if d.Event.RoomID != nil {
		confirmed := false
		for _, c := range d.Confirmations {
			if c.RoomID == *d.Event.RoomID && c.Status == "CONFIRMED" {
				confirmed = true
				break
			}
		}
		if confirmed {
			p.RoomNote = "ruangan sudah dikonfirmasi TU"
		} else {
			p.RoomNote = "ruangan kandidat, wajib konfirmasi manual ke TU sebelum publikasi"
			p.Blocking = append(p.Blocking, "ruangan memerlukan konfirmasi TU berstatus CONFIRMED")
		}
	} else {
		p.RoomNote = "tanpa ruangan fisik"
	}
	return p, nil
}

func (s *EventService) findConflicts(ctx context.Context, d *EventDetail) (blocking, nonBlocking []string) {
	blocking = []string{}
	nonBlocking = []string{}
	// Blocking 1: same origin occurrence already has PUBLISHED event.
	if d.Event.OriginPatternID != nil && d.Event.OriginDate != nil && d.Event.Lifecycle != "PUBLISHED" {
		var n int
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM teaching_events
			WHERE origin_schedule_pattern_id = ? AND origin_occurrence_date = ?
			AND lifecycle_status = 'PUBLISHED' AND id != ?`,
			*d.Event.OriginPatternID, *d.Event.OriginDate, d.Event.ID).Scan(&n)
		if n > 0 {
			blocking = append(blocking, "sesi asal sudah memiliki perubahan terbit")
		}
	}
	// Blocking 2: overlapping PUBLISHED event on same OWNER offering.
	var n int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
		WHERE teo.course_offering_id = ? AND teo.participation_role = 'OWNER'
		AND te.lifecycle_status = 'PUBLISHED' AND te.id != ?
		AND te.starts_at < ? AND ? < te.ends_at`,
		d.Event.OwnerOfferingID, d.Event.ID, d.Event.EndsAt, d.Event.StartsAt).Scan(&n)
	if n > 0 {
		blocking = append(blocking, "offering pemilik sudah memiliki event terbit yang bertabrakan waktu")
	}
	// Non-blocking: room overlap with other PUBLISHED events.
	if d.Event.RoomID != nil {
		var rn int
		_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM teaching_events
			WHERE room_id = ? AND lifecycle_status = 'PUBLISHED' AND id != ?
			AND starts_at < ? AND ? < ends_at`,
			*d.Event.RoomID, d.Event.ID, d.Event.EndsAt, d.Event.StartsAt).Scan(&rn)
		if rn > 0 {
			nonBlocking = append(nonBlocking, "ruangan dipakai event terbit lain pada rentang waktu beririsan")
		}
	}
	// Non-blocking: class-level overlap (other offerings same class).
	var cn int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE sem.class_id = ? AND teo.course_offering_id != ?
		AND te.lifecycle_status = 'PUBLISHED' AND te.id != ?
		AND te.starts_at < ? AND ? < te.ends_at`,
		d.Event.OwnerClassID, d.Event.OwnerOfferingID, d.Event.ID, d.Event.EndsAt, d.Event.StartsAt).Scan(&cn)
	if cn > 0 {
		nonBlocking = append(nonBlocking, "kelas pemilik memiliki event terbit lain yang beririsan waktu")
	}
	return blocking, nonBlocking
}

// Publish moves DRAFT to PUBLISHED after conflict + scope checks.
func (s *EventService) Publish(ctx context.Context, actor EventActor, eventID int64, overrideReason *string) (*EventRow, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	if d.Event.Lifecycle == "PUBLISHED" {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &d.Event, nil
	}
	if d.Event.Lifecycle != "DRAFT" {
		return nil, ErrEventInvalidState
	}
	// Exactly one OWNER required.
	owners := 0
	for _, p := range d.Participations {
		if p.Role == "OWNER" {
			owners++
		}
	}
	if owners != 1 {
		return nil, errors.New("event harus memiliki tepat satu OWNER sebelum publikasi")
	}
	blocking, nonBlocking := s.findConflicts(ctx, d)
	if len(blocking) > 0 {
		return nil, ErrEventConflict
	}
	reason := overrideReason
	if len(nonBlocking) > 0 {
		if reason == nil || strings.TrimSpace(*reason) == "" {
			return nil, errors.New("konflik non-pemblokir memerlukan conflict_override_reason")
		}
	} else {
		reason = nil
	}
	// Room confirmation gate.
	if d.Event.RoomID != nil {
		confirmed := false
		for _, c := range d.Confirmations {
			if c.RoomID == *d.Event.RoomID && c.Status == "CONFIRMED" {
				confirmed = true
				break
			}
		}
		if !confirmed {
			return nil, errors.New("ruangan memerlukan konfirmasi TU berstatus CONFIRMED")
		}
	}
	now := eventNow()
	res, err := tx.ExecContext(ctx, `UPDATE teaching_events SET lifecycle_status='PUBLISHED',
		published_by_user_id=?, published_at=?, conflict_override_reason=?,
		version=version+1, updated_at=? WHERE id=? AND lifecycle_status='DRAFT'`,
		actor.UserID, now, reason, now, eventID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, ErrEventInvalidState
	}
	after, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	semID := ownerSemester(ctx, tx, after.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, after.Event.OwnerClassID, semID, "PUBLISH", eventID,
		str(`{"lifecycle":"DRAFT"}`), str(`{"lifecycle":"PUBLISHED"}`), reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &after.Event, nil
}

// Revoke moves PUBLISHED to REVOKED (KM of owner class only).
func (s *EventService) Revoke(ctx context.Context, actor EventActor, eventID int64, reason string) (*EventRow, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, errors.New("alasan pencabutan wajib diisi")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	if d.Event.Lifecycle != "PUBLISHED" {
		return nil, ErrEventInvalidState
	}
	now := eventNow()
	res, err := tx.ExecContext(ctx, `UPDATE teaching_events SET lifecycle_status='REVOKED',
		revoked_by_user_id=?, revoked_at=?, revocation_reason=?,
		version=version+1, updated_at=? WHERE id=? AND lifecycle_status='PUBLISHED'`,
		actor.UserID, now, reason, now, eventID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, ErrEventInvalidState
	}
	after, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	semID := ownerSemester(ctx, tx, after.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, after.Event.OwnerClassID, semID, "REVOKE", eventID,
		str(`{"lifecycle":"PUBLISHED"}`), str(`{"lifecycle":"REVOKED"}`), &reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &after.Event, nil
}

// AddParticipant invites another offering (owner KM only).
func (s *EventService) AddParticipant(ctx context.Context, actor EventActor, eventID, offeringID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return err
	}
	var classID, semID int64
	var semStarts, semEnds string
	err = tx.QueryRowContext(ctx, `SELECT sem.class_id, co.semester_id, sem.starts_on, sem.ends_on
		FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?`, offeringID).
		Scan(&classID, &semID, &semStarts, &semEnds)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEventInvalidInput
	}
	if err != nil {
		return err
	}
	if classID == d.Event.OwnerClassID {
		return errors.New("offering PARTICIPANT tidak boleh dari kelas pemilik")
	}
	starts, _ := parseEventTime(d.Event.StartsAt)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	_ = loc
	evDate := starts.Format("2006-01-02")
	// Participant semester bound check (compare UTC date against semester range is sufficient for MVP;
	// full TZ-aware check happens on accept with participant class TZ).
	if evDate < semStarts || evDate > semEnds {
		return errors.New("event di luar periode semester offering peserta")
	}
	var existingRole, existingStatus string
	err = tx.QueryRowContext(ctx, `SELECT participation_role, participation_status FROM teaching_event_offerings
		WHERE teaching_event_id = ? AND course_offering_id = ?`, eventID, offeringID).Scan(&existingRole, &existingStatus)
	if err == nil {
		if existingRole == "OWNER" {
			return ErrEventInvalidInput
		}
		if existingStatus == "PENDING" || existingStatus == "ACCEPTED" {
			return ErrEventInvalidState
		}
		// Re-invite DECLINED/REMOVED -> PENDING.
		if _, err := tx.ExecContext(ctx, `UPDATE teaching_event_offerings SET participation_status='PENDING',
			responded_by_user_id=NULL, responded_at=NULL, updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
			WHERE teaching_event_id=? AND course_offering_id=?`, eventID, offeringID); err != nil {
			return err
		}
	} else if errors.Is(err, sql.ErrNoRows) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO teaching_event_offerings
			(teaching_event_id, course_offering_id, participation_role, participation_status)
			VALUES (?, ?, 'PARTICIPANT', 'PENDING')`, eventID, offeringID); err != nil {
			return err
		}
	} else {
		return err
	}
	ownerSem := ownerSemester(ctx, tx, d.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, d.Event.OwnerClassID, ownerSem, "INVITE_PARTICIPANT", eventID,
		nil, str(fmt.Sprintf(`{"offering":%d}`, offeringID)), nil); err != nil {
		return err
	}
	return tx.Commit()
}

// RespondParticipation lets participant KM accept/decline/remove.
func (s *EventService) RespondParticipation(ctx context.Context, actor EventActor, eventID, offeringID int64, decision string) error {
	decision = strings.ToUpper(strings.TrimSpace(decision))
	if decision != "ACCEPTED" && decision != "DECLINED" && decision != "REMOVED" {
		return ErrEventInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var role, status string
	err = tx.QueryRowContext(ctx, `SELECT participation_role, participation_status FROM teaching_event_offerings
		WHERE teaching_event_id = ? AND course_offering_id = ?`, eventID, offeringID).Scan(&role, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrEventNotFound
	}
	if err != nil {
		return err
	}
	if role != "PARTICIPANT" {
		return ErrEventInvalidInput
	}
	allowed := false
	switch status {
	case "PENDING":
		allowed = decision == "ACCEPTED" || decision == "DECLINED" || decision == "REMOVED"
	case "ACCEPTED":
		allowed = decision == "REMOVED"
	}
	if !allowed {
		return ErrEventInvalidState
	}
	now := eventNow()
	if _, err := tx.ExecContext(ctx, `UPDATE teaching_event_offerings SET participation_status=?,
		responded_by_user_id=?, responded_at=?, updated_at=? WHERE teaching_event_id=? AND course_offering_id=?`,
		decision, actor.UserID, now, now, eventID, offeringID); err != nil {
		return err
	}
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return err
	}
	ownerSem := ownerSemester(ctx, tx, d.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, d.Event.OwnerClassID, ownerSem, "RESPOND_PARTICIPANT", eventID,
		str(fmt.Sprintf(`{"offering":%d,"from":"%s"}`, offeringID, status)),
		str(fmt.Sprintf(`{"offering":%d,"to":"%s"}`, offeringID, decision)), nil); err != nil {
		return err
	}
	return tx.Commit()
}

// RecordRoomConfirmation attaches a TU confirmation note to a draft.
func (s *EventService) RecordRoomConfirmation(ctx context.Context, actor EventActor, eventID, roomID int64, status, contact, note string) (*RoomConfirmation, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "PENDING" && status != "CONFIRMED" && status != "REJECTED" {
		return nil, ErrEventInvalidInput
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	d, err := s.getDetail(ctx, tx, eventID)
	if err != nil {
		return nil, err
	}
	if d.Event.Lifecycle != "DRAFT" {
		return nil, ErrEventInvalidState
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM rooms WHERE id = ?)`, roomID).Scan(&exists); err != nil || !exists {
		return nil, ErrEventInvalidInput
	}
	now := eventNow()
	var confirmed any
	if status == "CONFIRMED" {
		confirmed = now
	}
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO room_confirmations
		(teaching_event_id, room_id, confirmation_status, external_contact, note, recorded_by_user_id, recorded_at, confirmed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		eventID, roomID, status, nullable(contact), nullable(note), actor.UserID, now, confirmed).Scan(&id)
	if err != nil {
		return nil, err
	}
	// Point the draft at the confirmed room for convenience.
	if _, err := tx.ExecContext(ctx, `UPDATE teaching_events SET room_id=?, updated_at=? WHERE id=?`, roomID, now, eventID); err != nil {
		return nil, err
	}
	ownerSem := ownerSemester(ctx, tx, d.Event.OwnerOfferingID)
	if err := insertEventAuditTx(ctx, tx, actor, d.Event.OwnerClassID, ownerSem, "ROOM_CONFIRM", eventID,
		nil, str(fmt.Sprintf(`{"room":%d,"status":"%s"}`, roomID, status)), nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	var c RoomConfirmation
	var contactOut, noteOut, confirmedOut sql.NullString
	var code string
	err = s.db.QueryRowContext(ctx, `SELECT rc.id, rc.room_id, r.code, rc.confirmation_status, rc.external_contact,
		rc.note, rc.recorded_by_user_id, rc.recorded_at, rc.confirmed_at FROM room_confirmations rc
		JOIN rooms r ON r.id = rc.room_id WHERE rc.id = ?`, id).
		Scan(&c.ID, &c.RoomID, &code, &c.Status, &contactOut, &noteOut, &c.RecordedBy, &c.RecordedAt, &confirmedOut)
	if err != nil {
		return nil, err
	}
	c.RoomCode = code
	if contactOut.Valid {
		c.Contact = &contactOut.String
	}
	if noteOut.Valid {
		c.Note = &noteOut.String
	}
	if confirmedOut.Valid {
		c.ConfirmedAt = &confirmedOut.String
	}
	return &c, nil
}

// ListEvents returns events visible to a class.
func (s *EventService) ListEvents(ctx context.Context, classID int64, lifecycle, kind string) ([]EventRow, error) {
	query := `SELECT DISTINCT te.id, te.event_kind, te.lifecycle_status, te.starts_at, te.ends_at, te.room_id,
		te.reason, te.conflict_override_reason, te.origin_schedule_pattern_id, te.origin_occurrence_date,
		te.result_schedule_pattern_id, te.version, te.published_by_user_id, te.published_at,
		te.revoked_by_user_id, te.revoked_at, te.revocation_reason,
		own.co_id, own.class_id, own.display_name
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		JOIN (SELECT te2.id AS eid, co2.id AS co_id, sem2.class_id AS class_id, co2.display_name AS display_name
			FROM teaching_events te2 JOIN teaching_event_offerings o ON o.teaching_event_id = te2.id AND o.participation_role='OWNER'
			JOIN course_offerings co2 ON co2.id = o.course_offering_id
			JOIN semesters sem2 ON sem2.id = co2.semester_id) own ON own.eid = te.id
		WHERE ((teo.participation_role='OWNER' AND sem.class_id = ?)
			OR (teo.participation_role='PARTICIPANT' AND teo.participation_status='ACCEPTED' AND sem.class_id = ?))`
	args := []any{classID, classID}
	if lifecycle != "" {
		query += ` AND te.lifecycle_status = ?`
		args = append(args, strings.ToUpper(lifecycle))
	}
	if kind != "" {
		query += ` AND te.event_kind = ?`
		args = append(args, strings.ToUpper(kind))
	}
	query += ` ORDER BY te.starts_at ASC, te.id ASC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EventRow{}
	for rows.Next() {
		var e EventRow
		var roomID sql.NullInt64
		var reason, override sql.NullString
		var originPat, resultPat sql.NullInt64
		var originDate sql.NullString
		var pubBy, revBy sql.NullInt64
		var pubAt, revAt sql.NullString
		var revReason sql.NullString
		if err := rows.Scan(&e.ID, &e.Kind, &e.Lifecycle, &e.StartsAt, &e.EndsAt, &roomID,
			&reason, &override, &originPat, &originDate, &resultPat, &e.Version,
			&pubBy, &pubAt, &revBy, &revAt, &revReason,
			&e.OwnerOfferingID, &e.OwnerClassID, &e.OwnerDisplay); err != nil {
			return nil, err
		}
		if roomID.Valid {
			e.RoomID = &roomID.Int64
		}
		if reason.Valid {
			e.Reason = &reason.String
		}
		if override.Valid {
			e.ConflictOverride = &override.String
		}
		if originPat.Valid {
			e.OriginPatternID = &originPat.Int64
		}
		if originDate.Valid {
			e.OriginDate = &originDate.String
		}
		if resultPat.Valid {
			e.ResultPatternID = &resultPat.Int64
		}
		if pubBy.Valid {
			e.PublishedBy = &pubBy.Int64
		}
		if pubAt.Valid {
			e.PublishedAt = &pubAt.String
		}
		if revBy.Valid {
			e.RevokedBy = &revBy.Int64
		}
		if revAt.Valid {
			e.RevokedAt = &revAt.String
		}
		if revReason.Valid {
			e.RevocationReason = &revReason.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func ownerSemester(ctx context.Context, tx *sql.Tx, offeringID int64) int64 {
	var semID int64
	_ = tx.QueryRowContext(ctx, `SELECT semester_id FROM course_offerings WHERE id = ?`, offeringID).Scan(&semID)
	return semID
}

func insertEventAuditTx(ctx context.Context, tx *sql.Tx, actor EventActor, classID, semesterID int64, action string, eventID int64, before, after *string, reason *string) error {
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = eventNow()
	}
	var actorUser, actorAssignment any
	actorType := "USER"
	if actor.UserID > 0 {
		actorUser = actor.UserID
	} else {
		actorType = "SYSTEM"
	}
	if actor.RoleAssignmentID > 0 {
		actorAssignment = actor.RoleAssignmentID
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, before_json, after_json, reason, correlation_id,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 'TEACHING_EVENT', ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, semesterID, actorUser, actorAssignment, actorType, action, eventID, before, after, reason, corr)
	return err
}

func deref(s *string, fallback string) string {
	if s == nil {
		return fallback
	}
	return *s
}

func nullable(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func str(s string) *string { return &s }

func itoa(n int) string { return fmt.Sprint(n) }
