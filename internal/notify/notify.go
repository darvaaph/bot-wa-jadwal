package notify

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNotFound     = errors.New("notifikasi tidak ditemukan")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrNoChannel    = errors.New("kelas belum memiliki kanal WhatsApp aktif")
)

// Sender abstracts WhatsApp delivery. Implementations must return provider message ID on success.
type Sender interface {
	SendText(ctx context.Context, jid, text string) (string, error)
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func nowStr() string { return time.Now().UTC().Format(time.RFC3339Nano) }

type Payload struct {
	Text string `json:"text"`
	Link string `json:"link,omitempty"`
}

// Enqueue inserts a message idempotently. Returns (id, created).
func (s *Service) Enqueue(ctx context.Context, classID, channelID int64, eventType, entityType string, entityID int64, key, text, link string, scheduledAt time.Time, triggeredBy *int64) (int64, bool, error) {
	eventType = strings.TrimSpace(eventType)
	entityType = strings.TrimSpace(entityType)
	key = strings.TrimSpace(key)
	if classID <= 0 || channelID <= 0 || eventType == "" || entityType == "" || entityID <= 0 || key == "" || strings.TrimSpace(text) == "" {
		return 0, false, ErrInvalidInput
	}
	payload, _ := json.Marshal(Payload{Text: text, Link: link})
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO notification_messages
		(class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
		classID, channelID, eventType, entityType, entityID, key, string(payload), scheduledAt.UTC().Format(time.RFC3339Nano), triggeredBy,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// Conflict: fetch existing id.
		if err := s.db.QueryRowContext(ctx, `SELECT id FROM notification_messages WHERE idempotency_key = ?`, key).Scan(&id); err != nil {
			return 0, false, err
		}
		return id, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// ChannelForClass returns the first ACTIVE channel for a class.
func (s *Service) ChannelForClass(ctx context.Context, classID int64) (int64, string, error) {
	var id int64
	var jid string
	err := s.db.QueryRowContext(ctx, `SELECT id, jid FROM whatsapp_channels WHERE class_id = ? AND status = 'ACTIVE' ORDER BY id LIMIT 1`, classID).Scan(&id, &jid)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	return id, jid, err
}

// EnsureChannel creates an ACTIVE channel if none exists (used by tests/ops; JID ownership stays explicit).
func (s *Service) EnsureChannel(ctx context.Context, classID int64, jid, displayName string) (int64, error) {
	jid = strings.TrimSpace(jid)
	if jid == "" {
		return 0, ErrInvalidInput
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status)
		VALUES (?, ?, 'GROUP', ?, 'ACTIVE') ON CONFLICT(jid) DO UPDATE SET class_id=excluded.class_id, status='ACTIVE', updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') RETURNING id`,
		classID, jid, displayName).Scan(&id)
	return id, err
}

func dailyKey(classID int64, date string) string {
	return fmt.Sprintf("daily:%d:%s", classID, date)
}
func taskKey(classID int64, date string) string {
	return fmt.Sprintf("tasks:%d:%s", classID, date)
}
func eventPublishKey(eventID int64) string { return fmt.Sprintf("event-publish:%d", eventID) }
func eventRevokeKey(eventID int64) string  { return fmt.Sprintf("event-revoke:%d", eventID) }
func replacementKey(eventID int64) string  { return fmt.Sprintf("replacement-reminder:%d", eventID) }

// BuildDailyText composes the morning summary per PRD template from effective schedule + urgent tasks.
func (s *Service) BuildDailyText(ctx context.Context, classID int64, date string, link string) (string, error) {
	loc, semID, classCode, err := s.classContext(ctx, classID)
	if err != nil {
		return "", err
	}
	_ = semID
	dayName := indonesianWeekday(date)
	var sb strings.Builder
	fmt.Fprintf(&sb, "*JADWAL KULIAH HARI INI*\n%s, %s | %s\n", dayName, date, classCode)
	rows, err := s.db.QueryContext(ctx, `SELECT c.name, co.activity_type, sp.start_time, sp.end_time,
		COALESCE(r.name, '-'), COALESCE(GROUP_CONCAT(l.full_name, ', '), '-')
		FROM schedule_patterns sp JOIN course_offerings co ON co.id = sp.course_offering_id
		JOIN courses c ON c.id = co.course_id JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN rooms r ON r.id = sp.room_id
		LEFT JOIN offering_lecturers ol ON ol.course_offering_id = co.id
		LEFT JOIN lecturers l ON l.id = ol.lecturer_id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE' AND sp.status = 'ACTIVE'
		AND sp.day_of_week = ? AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
		GROUP BY sp.id ORDER BY sp.start_time`, classID, weekdayNumber(date), date, date)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		var name, activity, start, end, room, lecturers string
		if err := rows.Scan(&name, &activity, &start, &end, &room, &lecturers); err != nil {
			return "", err
		}
		i++
		fmt.Fprintf(&sb, "\n%d. *%s* (%s)\n   %s - %s WIB\n   %s | %s", i, name, activity, start, end, room, lecturers)
	}
	if i == 0 {
		sb.WriteString("\nTidak ada jadwal hari ini.")
	}
	_ = loc
	if link != "" {
		fmt.Fprintf(&sb, "\n\nDetail jadwal dan tugas:\n%s", link)
	}
	return sb.String(), nil
}

// BuildTaskText composes the afternoon task reminder.
func (s *Service) BuildTaskText(ctx context.Context, classID int64, link string) (string, error) {
	_, _, classCode, err := s.classContext(ctx, classID)
	if err != nil {
		return "", err
	}
	_ = classCode
	rows, err := s.db.QueryContext(ctx, `SELECT c.name, t.title, t.deadline_at FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN courses c ON c.id = co.course_id JOIN semesters sem ON sem.id = co.semester_id
		WHERE sem.class_id = ? AND t.publication_status = 'PUBLISHED' AND t.deleted_at IS NULL
		AND t.archived_at IS NULL AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC LIMIT 10`, classID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var sb strings.Builder
	sb.WriteString("*PENGINGAT TUGAS*\n")
	now := time.Now().UTC()
	i := 0
	overdue := 0
	for rows.Next() {
		var course, title, deadline string
		if err := rows.Scan(&course, &title, &deadline); err != nil {
			return "", err
		}
		i++
		status := "Aktif"
		if dl, err := time.Parse(time.RFC3339Nano, deadline); err == nil && now.After(dl) {
			status = "Terlewat"
			overdue++
		} else if dl, err := time.Parse(time.RFC3339, deadline); err == nil && now.After(dl) {
			status = "Terlewat"
			overdue++
		}
		fmt.Fprintf(&sb, "\n%d. *%s* | %s\n   Deadline: %s (%s)", i, course, title, deadline, status)
	}
	if i == 0 {
		sb.WriteString("\nTidak ada tugas aktif.")
	}
	if link != "" {
		fmt.Fprintf(&sb, "\n\nLihat detail:\n%s", link)
	}
	return sb.String(), nil
}

func (s *Service) classContext(ctx context.Context, classID int64) (*time.Location, int64, string, error) {
	var code, tz string
	var semID int64
	err := s.db.QueryRowContext(ctx, `SELECT c.code, COALESCE(cs.timezone,'Asia/Jakarta') FROM classes c
		LEFT JOIN class_settings cs ON cs.class_id = c.id WHERE c.id = ?`, classID).Scan(&code, &tz)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, "", ErrNotFound
	}
	if err != nil {
		return nil, 0, "", err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ? AND status='ACTIVE' LIMIT 1`, classID).Scan(&semID)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return loc, semID, code, nil
}

// EnsureDailySummaries enqueues morning summaries for classes whose local time matches config.
func (s *Service) EnsureDailySummaries(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, COALESCE(cs.morning_reminder_time,'06:00'), COALESCE(cs.timezone,'Asia/Jakarta')
		FROM classes c LEFT JOIN class_settings cs ON cs.class_id = c.id WHERE c.status='ACTIVE'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var classID int64
		var hm, tz string
		if err := rows.Scan(&classID, &hm, &tz); err != nil {
			continue
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.FixedZone("WIB", 7*3600)
		}
		local := now.In(loc)
		if local.Format("15:04") != hm {
			continue
		}
		date := local.Format("2006-01-02")
		chID, _, err := s.ChannelForClass(ctx, classID)
		if err != nil {
			continue
		}
		text, err := s.BuildDailyText(ctx, classID, date, "")
		if err != nil {
			continue
		}
		sched := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, time.UTC)
		if _, created, err := s.Enqueue(ctx, classID, chID, "DAILY_SUMMARY", "CLASS", classID, dailyKey(classID, date), text, "", sched, nil); err == nil && created {
			enqueued++
		}
	}
	return enqueued, rows.Err()
}

// EnsureTaskReminders enqueues afternoon reminders similarly.
func (s *Service) EnsureTaskReminders(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, COALESCE(cs.afternoon_reminder_time,'17:00'), COALESCE(cs.timezone,'Asia/Jakarta')
		FROM classes c LEFT JOIN class_settings cs ON cs.class_id = c.id WHERE c.status='ACTIVE'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var classID int64
		var hm, tz string
		if err := rows.Scan(&classID, &hm, &tz); err != nil {
			continue
		}
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.FixedZone("WIB", 7*3600)
		}
		local := now.In(loc)
		if local.Format("15:04") != hm {
			continue
		}
		date := local.Format("2006-01-02")
		chID, _, err := s.ChannelForClass(ctx, classID)
		if err != nil {
			continue
		}
		text, err := s.BuildTaskText(ctx, classID, "")
		if err != nil {
			continue
		}
		sched := time.Date(local.Year(), local.Month(), local.Day(), local.Hour(), local.Minute(), 0, 0, time.UTC)
		if _, created, err := s.Enqueue(ctx, classID, chID, "TASK_REMINDER", "CLASS", classID, taskKey(classID, date), text, "", sched, nil); err == nil && created {
			enqueued++
		}
	}
	return enqueued, rows.Err()
}

// EnsureReplacementReminders enqueues pre-event reminders for PUBLISHED REPLACEMENT events.
func (s *Service) EnsureReplacementReminders(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT te.id, sem.class_id, COALESCE(cs.replacement_reminder_minutes,60),
		te.starts_at, ch.id FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role='OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN class_settings cs ON cs.class_id = sem.class_id
		JOIN whatsapp_channels ch ON ch.class_id = sem.class_id AND ch.status='ACTIVE'
		WHERE te.event_kind='REPLACEMENT' AND te.lifecycle_status='PUBLISHED'`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	enqueued := 0
	for rows.Next() {
		var eventID, classID, minutes, chID int64
		var startsAt string
		if err := rows.Scan(&eventID, &classID, &minutes, &startsAt, &chID); err != nil {
			continue
		}
		starts, err := time.Parse(time.RFC3339Nano, startsAt)
		if err != nil {
			starts, err = time.Parse(time.RFC3339, startsAt)
			if err != nil {
				continue
			}
		}
		remindAt := starts.Add(-time.Duration(minutes) * time.Minute)
		if now.Before(remindAt.Add(-time.Hour)) || now.After(starts) {
			continue
		}
		text := fmt.Sprintf("*PENGINGAT KELAS PENGGANTI*\nEvent #%d dimulai %s. Siapkan kehadiran tepat waktu.", eventID, starts.Format("02-01-2006 15:04"))
		if _, created, err := s.Enqueue(ctx, classID, chID, "REPLACEMENT_REMINDER", "TEACHING_EVENT", eventID, replacementKey(eventID), text, "", remindAt, nil); err == nil && created {
			enqueued++
		}
	}
	return enqueued, rows.Err()
}

// EnqueueEventPublished creates a change message after schedule publish (idempotent).
func (s *Service) EnqueueEventPublished(ctx context.Context, classID, eventID int64, text string, triggeredBy *int64) (int64, error) {
	chID, _, err := s.ChannelForClass(ctx, classID)
	if err != nil {
		return 0, ErrNoChannel
	}
	id, _, err := s.Enqueue(ctx, classID, chID, "SCHEDULE_CHANGE", "TEACHING_EVENT", eventID, eventPublishKey(eventID), text, "", time.Now().UTC(), triggeredBy)
	return id, err
}

// EnqueueEventRevoked creates a correction message and supersedes pending change messages.
// Both writes happen in one transaction so a crash cannot leave duplicates.
func (s *Service) EnqueueEventRevoked(ctx context.Context, classID, eventID int64, text string, triggeredBy *int64) (int64, error) {
	chID, _, err := s.ChannelForClass(ctx, classID)
	if err != nil {
		return 0, ErrNoChannel
	}
	payload, _ := json.Marshal(Payload{Text: text})
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO notification_messages
		(class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id)
		VALUES (?, ?, 'SCHEDULE_CORRECTION', 'TEACHING_EVENT', ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
		classID, chID, eventID, eventRevokeKey(eventID), string(payload), now, triggeredBy,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		if err := tx.QueryRowContext(ctx, `SELECT id FROM notification_messages WHERE idempotency_key = ?`, eventRevokeKey(eventID)).Scan(&id); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status='SUPERSEDED', supersedes_message_id=?,
		updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE entity_type='TEACHING_EVENT' AND entity_id=? AND id != ?
		AND event_type='SCHEDULE_CHANGE' AND status IN ('PENDING','PROCESSING','FAILED')`, id, eventID, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// staleProcessingTimeout bounds how long a message may sit in PROCESSING
// (e.g. worker crash between claim and attempt write) before requeue.
const staleProcessingTimeout = 10 * time.Minute

// ReapStaleProcessing returns stuck PROCESSING rows to PENDING.
func (s *Service) ReapStaleProcessing(ctx context.Context, now time.Time) (int64, error) {
	cutoff := now.Add(-staleProcessingTimeout).UTC().Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `UPDATE notification_messages SET status='PENDING',
		updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE status='PROCESSING' AND updated_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ProcessDue claims and sends due messages. Returns (sent, failed).
func (s *Service) ProcessDue(ctx context.Context, sender Sender, limit int, now time.Time) (int, int, error) {
	// Reap rows stuck in PROCESSING by a crashed worker before claiming.
	_, _ = s.ReapStaleProcessing(ctx, now)
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, whatsapp_channel_id, payload_json FROM notification_messages
		WHERE status IN ('PENDING','FAILED') AND scheduled_at <= ? ORDER BY scheduled_at ASC, id ASC LIMIT ?`,
		now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return 0, 0, err
	}
	type item struct {
		id, chID int64
		payload  string
	}
	var items []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.chID, &it.payload); err != nil {
			rows.Close()
			return 0, 0, err
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	sent, failed := 0, 0
	for _, it := range items {
		res, err := s.db.ExecContext(ctx, `UPDATE notification_messages SET status='PROCESSING',
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now')
			WHERE id=? AND status IN ('PENDING','FAILED')`, it.id)
		if err != nil {
			failed++
			continue
		}
		if n, _ := res.RowsAffected(); n != 1 {
			continue
		}
		var attempt int
		_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_number),0)+1 FROM notification_attempts WHERE notification_message_id=?`, it.id).Scan(&attempt)
		started := nowStr()
		var jid string
		_ = s.db.QueryRowContext(ctx, `SELECT jid FROM whatsapp_channels WHERE id=?`, it.chID).Scan(&jid)
		var p Payload
		_ = json.Unmarshal([]byte(it.payload), &p)
		if sender == nil || strings.TrimSpace(jid) == "" {
			_ = s.finishAttempt(ctx, it.id, attempt, started, false, "pengirim WhatsApp tidak tersedia (bot offline)", "")
			failed++
			continue
		}
		providerID, err := sender.SendText(ctx, jid, p.Text)
		if err != nil {
			_ = s.finishAttempt(ctx, it.id, attempt, started, false, err.Error(), "")
			failed++
			continue
		}
		_ = s.finishAttempt(ctx, it.id, attempt, started, true, "", providerID)
		sent++
	}
	return sent, failed, nil
}

func (s *Service) finishAttempt(ctx context.Context, messageID int64, attempt int, started string, ok bool, errMsg, providerID string) error {
	finished := nowStr()
	result := "FAILED"
	status := "FAILED"
	if ok {
		result = "SUCCESS"
		status = "SENT"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO notification_attempts
		(notification_message_id, attempt_number, started_at, finished_at, result, error_message, provider_message_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, messageID, attempt, started, finished, result, nullIfEmpty(errMsg), nullIfEmpty(providerID)); err != nil {
		return err
	}
	if ok {
		_, _ = tx.ExecContext(ctx, `UPDATE notification_messages SET status=?, sent_at=?, updated_at=? WHERE id=? AND status='PROCESSING'`, status, finished, finished, messageID)
	} else {
		_, _ = tx.ExecContext(ctx, `UPDATE notification_messages SET status=?, updated_at=? WHERE id=? AND status='PROCESSING'`, status, finished, messageID)
	}
	return tx.Commit()
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// GetMessageClass returns the owning class of a notification message.
func (s *Service) GetMessageClass(ctx context.Context, messageID int64) (int64, error) {
	var classID int64
	err := s.db.QueryRowContext(ctx, `SELECT class_id FROM notification_messages WHERE id = ?`, messageID).Scan(&classID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return classID, err
}

// Retry requeues a FAILED message.
func (s *Service) Retry(ctx context.Context, messageID int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE notification_messages SET status='PENDING',
		updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE id=? AND status='FAILED'`, messageID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return nil
}

type MessageItem struct {
	ID             int64   `json:"id"`
	ClassID        int64   `json:"class_id"`
	EventType      string  `json:"event_type"`
	EntityType     string  `json:"entity_type"`
	EntityID       int64   `json:"entity_id"`
	IdempotencyKey string  `json:"idempotency_key"`
	Status         string  `json:"status"`
	ScheduledAt    string  `json:"scheduled_at"`
	SentAt         *string `json:"sent_at,omitempty"`
	Text           string  `json:"text"`
	Attempts       int     `json:"attempts"`
	LastError      *string `json:"last_error,omitempty"`
}

// List returns messages for a class with optional status filter.
func (s *Service) List(ctx context.Context, classID int64, status string, limit int) ([]MessageItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT m.id, m.class_id, m.event_type, m.entity_type, m.entity_id, m.idempotency_key,
		m.status, m.scheduled_at, m.sent_at, m.payload_json,
		(SELECT COUNT(*) FROM notification_attempts a WHERE a.notification_message_id = m.id),
		(SELECT a.error_message FROM notification_attempts a WHERE a.notification_message_id = m.id ORDER BY a.attempt_number DESC LIMIT 1)
		FROM notification_messages m WHERE m.class_id = ?`
	args := []any{classID}
	if status != "" {
		query += ` AND m.status = ?`
		args = append(args, strings.ToUpper(status))
	}
	query += ` ORDER BY m.scheduled_at DESC, m.id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MessageItem{}
	for rows.Next() {
		var m MessageItem
		var sent, lastErr sql.NullString
		var payload string
		if err := rows.Scan(&m.ID, &m.ClassID, &m.EventType, &m.EntityType, &m.EntityID, &m.IdempotencyKey,
			&m.Status, &m.ScheduledAt, &sent, &payload, &m.Attempts, &lastErr); err != nil {
			return nil, err
		}
		if sent.Valid {
			m.SentAt = &sent.String
		}
		if lastErr.Valid {
			m.LastError = &lastErr.String
		}
		var p Payload
		if err := json.Unmarshal([]byte(payload), &p); err == nil {
			m.Text = p.Text
		} else {
			m.Text = payload
		}
		out = append(out, m)
	}
	return out, rows.Err()
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

func indonesianWeekday(date string) string {
	names := map[int]string{1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu"}
	return names[weekdayNumber(date)]
}
