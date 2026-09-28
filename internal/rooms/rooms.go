package rooms

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("ruangan tidak ditemukan")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrConflict     = errors.New("kode ruangan sudah digunakan")
)

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

// Actor carries audit identity for room operations.
type Actor struct {
	UserID           int64
	RoleAssignmentID int64
	CorrelationID    string
}

func insertRoomAudit(ctx context.Context, tx *sql.Tx, actor Actor, action string, roomID int64, after *string) error {
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = time.Now().UTC().Format(time.RFC3339Nano)
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
		actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'ROOM', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		actorUser, actorAssignment, actorType, action, roomID, after, corr)
	return err
}

type Room struct {
	ID              int64   `json:"id"`
	Code            string  `json:"code"`
	Name            string  `json:"name"`
	Building        *string `json:"building,omitempty"`
	RoomType        *string `json:"room_type,omitempty"`
	Capacity        *int    `json:"capacity,omitempty"`
	Status          string  `json:"status"`
	SourceUpdatedAt *string `json:"source_updated_at,omitempty"`
}

func scanRoom(row interface{ Scan(...any) error }) (*Room, error) {
	var r Room
	var building, roomType, source sql.NullString
	var capacity sql.NullInt64
	if err := row.Scan(&r.ID, &r.Code, &r.Name, &building, &roomType, &capacity, &r.Status, &source); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if building.Valid {
		r.Building = &building.String
	}
	if roomType.Valid {
		r.RoomType = &roomType.String
	}
	if capacity.Valid {
		v := int(capacity.Int64)
		r.Capacity = &v
	}
	if source.Valid {
		r.SourceUpdatedAt = &source.String
	}
	return &r, nil
}

func (s *Service) Create(ctx context.Context, actor Actor, code, name, building, roomType string, capacity *int) (*Room, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	if code == "" || name == "" {
		return nil, ErrInvalidInput
	}
	if capacity != nil && *capacity <= 0 {
		return nil, ErrInvalidInput
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO rooms (code, name, building, room_type, capacity, status, source_updated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?) RETURNING id`,
		code, name, nullStr(building), nullStr(roomType), nullInt(capacity), now).Scan(&id)
	if err != nil {
		return nil, ErrConflict
	}
	after := `{"code":"` + code + `","name":"` + strings.ReplaceAll(name, `"`, ``) + `"}`
	if err := insertRoomAudit(ctx, tx, actor, "CREATE", id, &after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *Service) Get(ctx context.Context, id int64) (*Room, error) {
	return scanRoom(s.db.QueryRowContext(ctx, `SELECT id, code, name, building, room_type, capacity, status, source_updated_at FROM rooms WHERE id = ?`, id))
}

func (s *Service) List(ctx context.Context, status string) ([]Room, error) {
	query := `SELECT id, code, name, building, room_type, capacity, status, source_updated_at FROM rooms`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, strings.ToUpper(status))
	}
	query += ` ORDER BY code ASC LIMIT 500`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Room{}
	for rows.Next() {
		r, err := scanRoom(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *r)
	}
	return out, rows.Err()
}

type UpdateInput struct {
	Name            *string
	Building        *string
	RoomType        *string
	Capacity        *int
	ClearCapacity   bool
	Status          *string
	ExpectedVersion int // reserved; rooms has no version column, ignored
}

func (s *Service) Update(ctx context.Context, actor Actor, id int64, in UpdateInput) (*Room, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	name := current.Name
	if in.Name != nil && strings.TrimSpace(*in.Name) != "" {
		name = strings.TrimSpace(*in.Name)
	}
	building := current.Building
	if in.Building != nil {
		if strings.TrimSpace(*in.Building) == "" {
			building = nil
		} else {
			v := strings.TrimSpace(*in.Building)
			building = &v
		}
	}
	roomType := current.RoomType
	if in.RoomType != nil {
		if strings.TrimSpace(*in.RoomType) == "" {
			roomType = nil
		} else {
			v := strings.TrimSpace(*in.RoomType)
			roomType = &v
		}
	}
	capacity := current.Capacity
	if in.ClearCapacity {
		capacity = nil
	} else if in.Capacity != nil {
		if *in.Capacity <= 0 {
			return nil, ErrInvalidInput
		}
		capacity = in.Capacity
	}
	status := current.Status
	if in.Status != nil {
		v := strings.ToUpper(strings.TrimSpace(*in.Status))
		if v != "ACTIVE" && v != "INACTIVE" {
			return nil, ErrInvalidInput
		}
		status = v
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE rooms SET name=?, building=?, room_type=?, capacity=?, status=?, source_updated_at=?, updated_at=?
		WHERE id=?`, name, building, roomType, nullInt(capacity), status, now, now, id); err != nil {
		return nil, err
	}
	after := `{"name":"` + strings.ReplaceAll(name, `"`, ``) + `","status":"` + status + `"}`
	if err := insertRoomAudit(ctx, tx, actor, "UPDATE", id, &after); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

type Candidate struct {
	Room      Room     `json:"room"`
	Conflicts []string `json:"conflicts"`
}

// Availability searches ACTIVE rooms excluding known overlaps (patterns + published events).
func (s *Service) Availability(ctx context.Context, date, start, end string) ([]Candidate, string, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, "", ErrInvalidInput
	}
	if len(start) != 5 || len(end) != 5 || start >= end {
		return nil, "", ErrInvalidInput
	}
	wd := weekdayNumber(date)
	rooms, err := s.List(ctx, "ACTIVE")
	if err != nil {
		return nil, "", err
	}
	// Overlapping patterns on that weekday/time.
	patRows, err := s.db.QueryContext(ctx, `SELECT DISTINCT sp.room_id FROM schedule_patterns sp
		WHERE sp.status='ACTIVE' AND sp.room_id IS NOT NULL AND sp.day_of_week = ?
		AND sp.start_time < ? AND ? < sp.end_time
		AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)`, wd, end, start, date, date)
	if err != nil {
		return nil, "", err
	}
	busyPattern := map[int64]string{}
	for patRows.Next() {
		var rid int64
		if err := patRows.Scan(&rid); err == nil {
			busyPattern[rid] = "dipakai jadwal reguler"
		}
	}
	patRows.Close()
	// Overlapping published events that day.
	dayStart := date + "T00:00:00Z"
	dayEnd := date + "T23:59:59Z"
	evRows, err := s.db.QueryContext(ctx, `SELECT DISTINCT room_id FROM teaching_events
		WHERE lifecycle_status='PUBLISHED' AND room_id IS NOT NULL
		AND starts_at <= ? AND ends_at >= ? AND starts_at < ? AND ? < ends_at`,
		dayEnd, dayStart, dateTime(date, end), dateTime(date, start))
	if err != nil {
		return nil, "", err
	}
	busyEvent := map[int64]string{}
	for evRows.Next() {
		var rid int64
		if err := evRows.Scan(&rid); err == nil {
			busyEvent[rid] = "dipakai event terbit"
		}
	}
	evRows.Close()
	out := []Candidate{}
	for _, r := range rooms {
		conflicts := []string{}
		if reason, ok := busyPattern[r.ID]; ok {
			conflicts = append(conflicts, reason)
		}
		if reason, ok := busyEvent[r.ID]; ok {
			conflicts = append(conflicts, reason)
		}
		if len(conflicts) == 0 {
			out = append(out, Candidate{Room: r, Conflicts: []string{}})
		}
	}
	note := "Hasil berdasarkan data internal; wajib konfirmasi manual ke TU sebelum publikasi."
	if len(out) == 0 {
		note = "Tidak ada kandidat: semua ruangan bertabrakan atau data terbatas. " + note
	}
	return out, note, nil
}

func dateTime(date, hm string) string {
	return date + "T" + hm + ":00Z"
}

func weekdayNumber(date string) int {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return int(time.Now().Weekday())
	}
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

func nullStr(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

func nullInt(n *int) any {
	if n == nil {
		return nil
	}
	return *n
}
