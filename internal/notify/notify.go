package notify

import (
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/util"
	"context"
	"crypto/sha256"
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
	db               *sql.DB
	dailyTextBuilder func(context.Context, int64, time.Time) (string, error)
	messageFormatter func(string, string, string) string
}

func (s *Service) SetMessageFormatter(format func(string, string, string) string) {
	s.messageFormatter = format
}

type StableSender interface {
	SendTextWithID(context.Context, string, string, string) (string, error)
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

// SetDailyTextBuilder uses the same effective schedule resolver as bot commands.
func (s *Service) SetDailyTextBuilder(builder func(context.Context, int64, time.Time) (string, error)) {
	s.dailyTextBuilder = builder
}

func nowStr() string { return time.Now().UTC().Format(time.RFC3339Nano) }

type Payload struct {
	Text string `json:"text"`
	Link string `json:"link,omitempty"`
}

// Enqueue inserts a message idempotently. Returns (id, created).
// channelID <= 0 menyimpan pesan durable tanpa channel (PENDING, NULL, NULL)
// menunggu ReconcilePendingChannels; tidak lagi membuang intent.
func (s *Service) Enqueue(ctx context.Context, classID, channelID int64, eventType, entityType string, entityID int64, key, text, link string, scheduledAt time.Time, triggeredBy *int64) (int64, bool, error) {
	eventType = strings.TrimSpace(eventType)
	entityType = strings.TrimSpace(entityType)
	key = strings.TrimSpace(key)
	if classID <= 0 || eventType == "" || entityType == "" || entityID <= 0 || key == "" || strings.TrimSpace(text) == "" {
		return 0, false, ErrInvalidInput
	}
	payload, _ := json.Marshal(Payload{Text: text, Link: link})
	var chArg any
	var schedArg any
	if channelID > 0 {
		chArg = channelID
		if scheduledAt.IsZero() {
			scheduledAt = time.Now().UTC()
		}
		schedArg = scheduledAt.UTC().Format(time.RFC3339Nano)
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO notification_messages
		(class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at, triggered_by_user_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		ON CONFLICT(idempotency_key) DO NOTHING RETURNING id`,
		classID, chArg, eventType, entityType, entityID, key, string(payload), schedArg, triggeredBy,
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
	if err != nil {
		return 0, err
	}
	_, _ = s.ReconcilePendingChannels(ctx, &classID)
	return id, err
}

func dailyKey(classID int64, date string) string {
	return fmt.Sprintf("daily:%d:%s", classID, date)
}
func taskKey(classID int64, date string) string {
	return fmt.Sprintf("tasks:%d:%s", classID, date)
}
func eventRevokeKey(eventID int64) string { return fmt.Sprintf("event-revoke:%d", eventID) }
func replacementKey(eventID int64) string { return fmt.Sprintf("replacement-reminder:%d", eventID) }

// BuildDailyText composes the morning schedule summary.
func (s *Service) BuildDailyText(ctx context.Context, classID int64, date string, link string) (string, error) {
	loc, semID, classCode, classSlug, err := s.classContext(ctx, classID)
	if err != nil {
		return "", err
	}
	if link == "" && classSlug != "" {
		link = fmt.Sprintf("%s/c/%s", util.PublicBaseURL(), classSlug)
	}
	if s.dailyTextBuilder != nil {
		day, err := time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			return "", err
		}
		text, err := s.dailyTextBuilder(ctx, classID, day)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(text) == "" {
			return "", nil
		}
		if link != "" && !strings.Contains(text, link) {
			text += "\n\nDetail jadwal dan tugas:\n" + link
		}
		return text, nil
	}
	_ = semID
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return "", err
	}
	sessions, err := schedule.ResolveEffective(ctx, s.db, classID, semID, day)
	if err != nil {
		return "", err
	}
	hasActiveClasses := false
	for _, it := range sessions {
		if it.Kind != "HOLIDAY" && it.Kind != "SESSION_CANCELLED" {
			hasActiveClasses = true
			break
		}
	}
	if !hasActiveClasses {
		return "", nil
	}
	dayName := indonesianWeekday(date)
	var sb strings.Builder
	fmt.Fprintf(&sb, "*JADWAL KULIAH HARI INI*\n%s, %s | %s\n", dayName, date, classCode)
	for index, it := range sessions {
		name := it.Offering
		if name == "" {
			name = it.Title
		}
		fmt.Fprintf(&sb, "\n%d. *%s* (%s) [%s]\n   %s - %s\n   %s", index+1, name, it.ActivityType, it.Kind, it.StartTime, it.EndTime, it.Room)
	}
	_ = loc
	if link != "" {
		fmt.Fprintf(&sb, "\n\nDetail jadwal dan tugas:\n%s", link)
	}
	return sb.String(), nil
}

func formatTaskDeadline(deadline, now time.Time) string {
	return util.FormatTaskDeadline(deadline, now)
}

func taskUrgencyStatus(deadline time.Time, now time.Time) string {
	if deadline.IsZero() {
		return "Aktif"
	}
	loc := now.Location()
	if loc == nil {
		loc = time.Local
	}
	now = now.In(loc)
	deadline = deadline.In(loc)

	diff := deadline.Sub(now)
	if diff < 0 {
		return "Terlewat"
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	targetDay := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, loc)
	days := int(targetDay.Sub(today) / (24 * time.Hour))

	if days <= 0 {
		hours := int(diff.Hours())
		mins := int(diff.Minutes()) % 60
		if hours > 0 {
			return fmt.Sprintf("Hari ini (Sisa ~%d jam)", hours)
		}
		if mins > 0 {
			return fmt.Sprintf("Hari ini (Sisa ~%d menit)", mins)
		}
		return "Hari ini"
	}
	if days == 1 {
		return "Besok (H-1)"
	}
	return fmt.Sprintf("%d hari lagi (H-%d)", days, days)
}

// BuildTaskText composes the afternoon task reminder formatted for clear mobile readability without decorative emojis.
func (s *Service) BuildTaskText(ctx context.Context, classID int64, link string, optNow ...time.Time) (string, error) {
	loc, _, classCode, classSlug, err := s.classContext(ctx, classID)
	if err != nil {
		return "", err
	}

	refNow := time.Now()
	if len(optNow) > 0 && !optNow[0].IsZero() {
		refNow = optNow[0]
	}
	nowLocal := refNow.In(loc)

	rows, err := s.db.QueryContext(ctx, `SELECT c.name, COALESCE(co.activity_type, ''), t.title, COALESCE(t.task_type, ''), COALESCE(t.submission_text, ''), t.deadline_at 
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN courses c ON c.id = co.course_id 
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE sem.class_id = ? AND sem.status = 'ACTIVE' AND t.publication_status = 'PUBLISHED' AND t.deleted_at IS NULL
		AND t.archived_at IS NULL AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC LIMIT 15`, classID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type taskItem struct {
		courseName     string
		activityType   string
		title          string
		taskType       string
		submissionText string
		deadlineFmt    string
	}
	var activeTasks []taskItem

	for rows.Next() {
		var course, actType, title, taskType, subText, rawDeadline string
		if err := rows.Scan(&course, &actType, &title, &taskType, &subText, &rawDeadline); err != nil {
			return "", err
		}
		dl := util.ParseFlexibleTime(rawDeadline, loc)
		if dl.IsZero() {
			continue
		}
		dlLocal := dl.In(loc)
		if nowLocal.After(dlLocal) {
			// Hanya tugas mendatang, abaikan tugas yang sudah terlewat
			continue
		}

		item := taskItem{
			courseName:     course,
			activityType:   actType,
			title:          title,
			taskType:       taskType,
			submissionText: strings.TrimSpace(subText),
			deadlineFmt:    formatTaskDeadline(dlLocal, nowLocal),
		}

		activeTasks = append(activeTasks, item)
		if len(activeTasks) >= 15 {
			break
		}
	}

	rows.Close()
	exams, err := s.buildExamReminder(ctx, classID, nowLocal.AddDate(0, 0, 1))
	if err != nil {
		return "", err
	}
	if len(activeTasks) == 0 && exams == "" {
		return "", nil
	}

	var sb strings.Builder
	if len(activeTasks) > 0 {
		if classCode != "" {
			sb.WriteString(fmt.Sprintf("*PENGINGAT TUGAS • %s*\n", classCode))
		} else {
			sb.WriteString("*PENGINGAT TUGAS*\n")
		}
		sb.WriteString("─────────────\n")

		for i, t := range activeTasks {
			courseDisplay := util.FormatCourseWithActivity(t.courseName, t.activityType)
			titleDisplay := util.FormatTaskTitle(t.title, t.taskType)
			subLabel := util.TaskSubmissionLabel(t.taskType)
			sb.WriteString(fmt.Sprintf("\n%d. *%s*: %s\n", i+1, courseDisplay, titleDisplay))
			subText := strings.TrimSpace(t.submissionText)
			if subText != "" && subText != "-" {
				sb.WriteString(fmt.Sprintf("   ├ *%s*\n", t.deadlineFmt))
				sb.WriteString(fmt.Sprintf("   └ %s: %s\n", subLabel, subText))
			} else {
				sb.WriteString(fmt.Sprintf("   └ *%s*\n", t.deadlineFmt))
			}
		}
	}

	if exams != "" {
		if len(activeTasks) > 0 {
			sb.WriteString("\n" + exams)
		} else {
			sb.WriteString(exams)
		}
	}
	if link == "" && classSlug != "" {
		link = fmt.Sprintf("%s/c/%s", util.PublicBaseURL(), classSlug)
	}
	if link != "" {
		sb.WriteString("\n─────────────\n")
		sb.WriteString(fmt.Sprintf("Detail & pengumpulan:\n%s", link))
	}

	return sb.String(), nil
}

func (s *Service) buildExamReminder(ctx context.Context, classID int64, tomorrow time.Time) (string, error) {
	start := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
	rows, err := s.db.QueryContext(ctx, `SELECT co.display_name, te.event_kind, te.starts_at, te.ends_at,
		COALESCE(r.code, '-'), COALESCE(te.reason, '')
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id=te.id
		JOIN course_offerings co ON co.id=teo.course_offering_id
		JOIN semesters sem ON sem.id=co.semester_id
		LEFT JOIN rooms r ON r.id=te.room_id
		WHERE sem.class_id=? AND sem.status='ACTIVE' AND te.lifecycle_status='PUBLISHED'
		AND (teo.participation_role='OWNER' OR teo.participation_status='ACCEPTED')
		AND te.event_kind IN ('EXAM_UTS','EXAM_UAS')
		AND datetime(te.starts_at)>=datetime(?) AND datetime(te.starts_at)<datetime(?)
		ORDER BY te.starts_at, te.id`, classID, start.UTC().Format(time.RFC3339), start.AddDate(0, 0, 1).UTC().Format(time.RFC3339))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var b strings.Builder
	i := 0
	for rows.Next() {
		var course, kind, starts, ends, room, note string
		if err := rows.Scan(&course, &kind, &starts, &ends, &room, &note); err != nil {
			return "", err
		}
		st, err := time.Parse(time.RFC3339Nano, starts)
		if err != nil {
			return "", err
		}
		en, err := time.Parse(time.RFC3339Nano, ends)
		if err != nil {
			return "", err
		}
		if i == 0 {
			fmt.Fprintf(&b, "*PENGINGAT UJIAN BESOK (H-1)*\n%s\n", start.Format("02 Jan 2006"))
		}
		i++
		fmt.Fprintf(&b, "\n%d. *%s* [%s]\n   - Waktu: %s - %s WIB\n   - Ruang: %s\n", i, course, strings.TrimPrefix(kind, "EXAM_"), st.In(start.Location()).Format("15:04"), en.In(start.Location()).Format("15:04"), room)
		if note != "" {
			fmt.Fprintf(&b, "   - Keterangan: %s\n", note)
		}
	}
	return b.String(), rows.Err()
}

func (s *Service) classContext(ctx context.Context, classID int64) (*time.Location, int64, string, string, error) {
	var code, slug, tz string
	var semID int64
	err := s.db.QueryRowContext(ctx, `SELECT c.code, c.slug, COALESCE(cs.timezone,'Asia/Jakarta') FROM classes c
		LEFT JOIN class_settings cs ON cs.class_id = c.id WHERE c.id = ?`, classID).Scan(&code, &slug, &tz)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, "", "", ErrNotFound
	}
	if err != nil {
		return nil, 0, "", "", err
	}
	_ = s.db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ? AND status='ACTIVE' LIMIT 1`, classID).Scan(&semID)
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	return loc, semID, code, slug, nil
}

func reminderWindowOpen(local time.Time, hm string) bool {
	due, err := time.ParseInLocation("2006-01-02 15:04", local.Format("2006-01-02")+" "+hm, local.Location())
	return err == nil && !local.Before(due) && local.Before(due.Add(time.Hour))
}

// EnsureDailySummaries enqueues morning summaries for eligible classes.
func (s *Service) EnsureDailySummaries(ctx context.Context, now time.Time) (int, error) {
	return s.ensureClassReminders(ctx, now, true)
}

// EnsureTaskReminders enqueues afternoon reminders for eligible classes.
func (s *Service) EnsureTaskReminders(ctx context.Context, now time.Time) (int, error) {
	return s.ensureClassReminders(ctx, now, false)
}

func (s *Service) ensureClassReminders(ctx context.Context, now time.Time, daily bool) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT c.id, COALESCE(cs.morning_reminder_time,'06:00'),
		COALESCE(cs.afternoon_reminder_time,'17:00'), COALESCE(cs.timezone,'Asia/Jakarta')
		FROM classes c LEFT JOIN class_settings cs ON cs.class_id = c.id WHERE c.status='ACTIVE'
		AND EXISTS (SELECT 1 FROM semesters sem WHERE sem.class_id=c.id AND sem.status='ACTIVE')`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type dueClass struct {
		id   int64
		date string
	}
	var due []dueClass
	for rows.Next() {
		var classID int64
		var morning, afternoon, zone string
		if err := rows.Scan(&classID, &morning, &afternoon, &zone); err != nil {
			return 0, err
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			loc = time.FixedZone("WIB", 7*3600)
		}
		hm := afternoon
		if daily {
			hm = morning
		}
		local := now.In(loc)
		if reminderWindowOpen(local, hm) {
			due = append(due, dueClass{classID, local.Format("2006-01-02")})
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var failures error
	enqueued := 0
	for _, class := range due {
		event, key := "TASK_REMINDER", taskKey(class.id, class.date)
		if daily {
			event, key = "DAILY_SUMMARY", dailyKey(class.id, class.date)
		}
		exists, err := s.hasReminder(ctx, key)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		if exists {
			continue
		}
		var message string
		if daily {
			message, err = s.BuildDailyText(ctx, class.id, class.date, "")
		} else {
			message, err = s.BuildTaskText(ctx, class.id, "", now)
		}
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", key, err))
			continue
		}
		if message == "" {
			continue
		}
		channelID, _, err := s.ChannelForClass(ctx, class.id)
		if err != nil && !errors.Is(err, ErrNotFound) {
			failures = errors.Join(failures, err)
			continue
		}
		_, created, err := s.Enqueue(ctx, class.id, channelID, event, "CLASS", class.id, key, message, "", now, nil)
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", key, err))
		} else if created {
			enqueued++
		}
	}
	return enqueued, failures
}

func (s *Service) hasReminder(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM notification_messages WHERE idempotency_key=?)`, key).Scan(&exists)
	return exists, err
}

// EnsureReplacementReminders enqueues pre-event reminders for PUBLISHED REPLACEMENT events.
func (s *Service) EnsureReplacementReminders(ctx context.Context, now time.Time) (int, error) {
	current := now.UTC().Format(time.RFC3339Nano)
	rows, err := s.db.QueryContext(ctx, `SELECT te.id, sem.class_id, COALESCE(cs.replacement_reminder_minutes,60),
		te.starts_at, ch.id,teo.participation_role,COALESCE(cs.timezone,'Asia/Jakarta') FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND (teo.participation_role='OWNER' OR (teo.participation_role='PARTICIPANT' AND teo.participation_status='ACCEPTED'))
		JOIN course_offerings co ON co.id = teo.course_offering_id
		JOIN semesters sem ON sem.id = co.semester_id
		LEFT JOIN class_settings cs ON cs.class_id = sem.class_id
		JOIN whatsapp_channels ch ON ch.class_id = sem.class_id AND ch.status='ACTIVE'
		WHERE te.event_kind='REPLACEMENT' AND te.lifecycle_status='PUBLISHED' AND sem.status='ACTIVE'
		  AND julianday(te.starts_at) >= julianday(?)
		  AND julianday(te.starts_at) <= julianday(?, printf('+%d minutes', COALESCE(cs.replacement_reminder_minutes,60)+60))`, current, current)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type reminder struct {
		eventID, classID, channelID int64
		key, zone                   string
		starts, remindAt            time.Time
	}
	var due []reminder
	for rows.Next() {
		var eventID, classID, minutes, chID int64
		var startsAt, participationRole, zone string
		if err := rows.Scan(&eventID, &classID, &minutes, &startsAt, &chID, &participationRole, &zone); err != nil {
			return 0, err
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
		key := replacementKey(eventID)
		if participationRole == "PARTICIPANT" {
			key += fmt.Sprintf(":class:%d", classID)
		}
		due = append(due, reminder{eventID, classID, chID, key, zone, starts, remindAt})
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var failures error
	enqueued := 0
	for _, item := range due {
		exists, err := s.hasReminder(ctx, item.key)
		if err != nil {
			failures = errors.Join(failures, err)
			continue
		}
		if exists {
			continue
		}
		loc, err := time.LoadLocation(item.zone)
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", item.key, err))
			continue
		}
		text := fmt.Sprintf("*PENGINGAT KELAS PENGGANTI*\nEvent #%d dimulai %s. Siapkan kehadiran tepat waktu.", item.eventID, item.starts.In(loc).Format("02-01-2006 15:04 MST"))
		_, created, err := s.Enqueue(ctx, item.classID, item.channelID, "REPLACEMENT_REMINDER", "TEACHING_EVENT", item.eventID, item.key, text, "", item.remindAt, nil)
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("%s: %w", item.key, err))
		} else if created {
			enqueued++
		}
	}
	return enqueued, failures
}

// EnqueueEventRevoked creates a correction message and supersedes pending change messages.
// Both writes happen in one transaction so a crash cannot leave duplicates.
// Durable: tetap tersimpan tanpa channel ketika kelas belum punya kanal aktif.
func (s *Service) EnqueueEventRevoked(ctx context.Context, classID, eventID int64, text string, triggeredBy *int64) (int64, error) {
	chID, _, _ := s.ChannelForClass(ctx, classID)
	var chArg any
	var schedArg any
	if chID > 0 {
		chArg = chID
		schedArg = time.Now().UTC().Format(time.RFC3339Nano)
	}
	payload, _ := json.Marshal(Payload{Text: text})
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
		classID, chArg, eventID, eventRevokeKey(eventID), string(payload), schedArg, triggeredBy,
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

// ReapStaleProcessing only requeues claims that never started a send.
// An unfinished attempt may have reached the provider and needs explicit reconciliation.
func (s *Service) ReapStaleProcessing(ctx context.Context, now time.Time) (int64, error) {
	cutoff := now.Add(-staleProcessingTimeout).UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE notification_attempts SET finished_at=?,result='FAILED',error_message='DELIVERY_UNCERTAIN: worker stopped before finalization',updated_at=? WHERE result IS NULL AND notification_message_id IN (SELECT id FROM notification_messages WHERE status='PROCESSING' AND julianday(updated_at)<julianday(?))`, nowStr(), nowStr(), cutoff); err != nil {
		return 0, err
	}
	res, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status=CASE WHEN EXISTS(SELECT 1 FROM notification_attempts a WHERE a.notification_message_id=notification_messages.id AND a.error_message LIKE 'DELIVERY_UNCERTAIN:%') THEN 'FAILED' ELSE 'PENDING' END,updated_at=? WHERE status='PROCESSING' AND julianday(updated_at)<julianday(?)`, nowStr(), cutoff)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

// ProcessDue owns claiming, durable attempts, delivery and finalization for every caller.
func (s *Service) ProcessDue(ctx context.Context, sender Sender, limit int, now time.Time) (int, int, error) {
	if _, err := s.ReapStaleProcessing(ctx, now); err != nil {
		return 0, 0, err
	}
	if _, err := s.ReconcilePendingChannels(ctx, nil); err != nil {
		return 0, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT m.id FROM notification_messages m JOIN whatsapp_channels ch ON ch.id=m.whatsapp_channel_id AND ch.status='ACTIVE' WHERE m.status='PENDING' AND m.scheduled_at IS NOT NULL AND julianday(m.scheduled_at)<=julianday(?) ORDER BY m.scheduled_at,m.id LIMIT ?`, now.UTC().Format(time.RFC3339Nano), limit)
	if err != nil {
		return 0, 0, err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return 0, 0, err
	}
	sent, failed := 0, 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return sent, failed, err
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return sent, failed, err
		}
		res, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status='PROCESSING',updated_at=? WHERE id=? AND status='PENDING'`, nowStr(), id)
		if err != nil {
			tx.Rollback()
			return sent, failed, err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			tx.Rollback()
			continue
		}
		var jid, payload, eventType, slug, key, zone string
		var classID int64
		err = tx.QueryRowContext(ctx, `SELECT ch.jid,m.payload_json,m.event_type,c.slug,m.idempotency_key,m.class_id,COALESCE(cs.timezone,'Asia/Jakarta') FROM notification_messages m JOIN whatsapp_channels ch ON ch.id=m.whatsapp_channel_id AND ch.status='ACTIVE' JOIN classes c ON c.id=m.class_id LEFT JOIN class_settings cs ON cs.class_id=m.class_id WHERE m.id=?`, id).Scan(&jid, &payload, &eventType, &slug, &key, &classID, &zone)
		if err != nil {
			tx.Rollback()
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return sent, failed, err
		}
		loc, err := time.LoadLocation(zone)
		if err != nil {
			tx.Rollback()
			return sent, failed, err
		}
		if (eventType == "DAILY_SUMMARY" || eventType == "TASK_REMINDER") && (strings.HasPrefix(key, "daily:") || strings.HasPrefix(key, "tasks:")) && !strings.HasSuffix(key, now.In(loc).Format("2006-01-02")) {
			if _, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status='SUPERSEDED',updated_at=? WHERE id=?`, nowStr(), id); err != nil {
				tx.Rollback()
				return sent, failed, err
			}
			if err := tx.Commit(); err != nil {
				return sent, failed, err
			}
			continue
		}
		var attempt int
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(attempt_number),0)+1 FROM notification_attempts WHERE notification_message_id=?`, id).Scan(&attempt); err != nil {
			tx.Rollback()
			return sent, failed, err
		}
		hash := sha256.Sum256([]byte(key + "\x00" + jid))
		stableID := fmt.Sprintf("3EB0%X", hash[:14])
		if _, err := tx.ExecContext(ctx, `INSERT INTO notification_attempts(notification_message_id,attempt_number,started_at,provider_message_id) VALUES(?,?,?,?)`, id, attempt, nowStr(), stableID); err != nil {
			tx.Rollback()
			return sent, failed, err
		}
		if err := tx.Commit(); err != nil {
			return sent, failed, err
		}
		var active bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM notification_messages m JOIN whatsapp_channels ch ON ch.id=m.whatsapp_channel_id AND ch.status='ACTIVE' WHERE m.id=? AND m.status='PROCESSING' AND ch.jid=?)`, id, jid).Scan(&active); err != nil {
			return sent, failed, err
		}
		if !active {
			finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			err := s.finishAttempt(finalCtx, id, attempt, false, "intent superseded or channel revoked", "")
			cancel()
			if err != nil {
				return sent, failed, err
			}
			continue
		}
		var p Payload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			return sent, failed, err
		}
		text := p.Text
		if s.messageFormatter != nil {
			text = s.messageFormatter(eventType, payload, slug)
		}
		if eventType == "DAILY_SUMMARY" && strings.HasPrefix(key, "daily:") {
			freshText, freshErr := s.BuildDailyText(ctx, classID, now.In(loc).Format("2006-01-02"), "")
			if freshErr == nil && freshText != "" {
				text = freshText
			}
		}
		if eventType == "TASK_REMINDER" && strings.HasPrefix(key, "tasks:") {
			freshText, freshErr := s.BuildTaskText(ctx, classID, "", now)
			if freshErr == nil && freshText != "" {
				text = freshText
			}
		}
		if err != nil {
			finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			finishErr := s.finishAttempt(finalCtx, id, attempt, false, err.Error(), "")
			cancel()
			if finishErr != nil {
				return sent, failed, finishErr
			}
			return sent, failed, err
		}
		if strings.TrimSpace(text) == "" {
			finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			err := s.finishAttempt(finalCtx, id, attempt, true, "", "")
			cancel()
			if err != nil {
				return sent, failed, err
			}
			continue
		}
		var providerID string
		var sendErr error
		if sender == nil {
			sendErr = errors.New("pengirim WhatsApp tidak tersedia (bot offline)")
		} else if stable, ok := sender.(StableSender); ok {
			providerID, sendErr = stable.SendTextWithID(ctx, jid, text, stableID)
		} else {
			providerID, sendErr = sender.SendText(ctx, jid, text)
		}
		errMsg := ""
		if sendErr != nil {
			errMsg = sendErr.Error()
		}
		finalCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		err = s.finishAttempt(finalCtx, id, attempt, sendErr == nil, errMsg, providerID)
		cancel()
		if err != nil {
			return sent, failed, fmt.Errorf("delivery %d finalization failed (provider success=%t): %w", id, sendErr == nil, err)
		}
		if sendErr != nil {
			failed++
		} else {
			sent++
		}
	}
	return sent, failed, nil
}

func (s *Service) finishAttempt(ctx context.Context, messageID int64, attempt int, ok bool, errMsg, providerID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, status := "FAILED", "FAILED"
	if ok {
		result, status = "SUCCESS", "SENT"
	}
	finished := nowStr()
	res, err := tx.ExecContext(ctx, `UPDATE notification_attempts SET finished_at=?,result=?,error_message=?,provider_message_id=COALESCE(?,provider_message_id),updated_at=? WHERE notification_message_id=? AND attempt_number=? AND result IS NULL`, finished, result, nullIfEmpty(errMsg), nullIfEmpty(providerID), finished, messageID, attempt)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("attempt finalization conflict")
	}
	var sentAt any
	if ok {
		sentAt = finished
	}
	if _, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status=?,sent_at=?,updated_at=? WHERE id=? AND status='PROCESSING'`, status, sentAt, finished, messageID); err != nil {
		return err
	}
	return tx.Commit()
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
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
		var sched, sent, lastErr sql.NullString
		var payload string
		if err := rows.Scan(&m.ID, &m.ClassID, &m.EventType, &m.EntityType, &m.EntityID, &m.IdempotencyKey,
			&m.Status, &sched, &sent, &payload, &m.Attempts, &lastErr); err != nil {
			return nil, err
		}
		if sched.Valid {
			m.ScheduledAt = sched.String
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
