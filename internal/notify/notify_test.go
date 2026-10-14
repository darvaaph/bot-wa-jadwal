package notify

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

type fakeSender struct {
	fail bool
	sent []string
}

func (f *fakeSender) SendText(ctx context.Context, jid, text string) (string, error) {
	if f.fail {
		return "", errors.New("WA offline")
	}
	f.sent = append(f.sent, jid+":"+text)
	return "provider-1", nil
}

func newNotifyTestDB(t *testing.T) (*Service, context.Context) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "notify.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-n','D4 TI',2024,'A')`); err != nil {
		t.Fatal(err)
	}
	now := "2026-09-24T10:00:00.000Z"
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode, morning_reminder_time, afternoon_reminder_time, replacement_reminder_minutes) VALUES (1,'Asia/Jakarta','LINK','06:00','17:00',60)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (1,'2026/2027','GANJIL','2026-09-01','2027-01-31','ACTIVE',?,?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status) VALUES (1,'120363@test@g.us','GROUP','Grup A','ACTIVE')`); err != nil {
		t.Fatal(err)
	}
	svc := NewService(db)
	_ = svc.db
	// Expose db via service for seeding.
	svcDB := db
	_ = svcDB
	return svc, ctx
}

func TestNotify_IdempotentEnqueueAndProcess(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	sender := &fakeSender{}

	id1, created1, err := svc.Enqueue(ctx, 1, 1, "DAILY_SUMMARY", "CLASS", 1, "daily:1:2026-10-06", "pagi", "", time.Date(2026, 10, 6, 6, 0, 0, 0, time.UTC), nil)
	if err != nil || !created1 {
		t.Fatalf("enqueue: err=%v created=%v", err, created1)
	}
	id2, created2, err := svc.Enqueue(ctx, 1, 1, "DAILY_SUMMARY", "CLASS", 1, "daily:1:2026-10-06", "pagi", "", time.Date(2026, 10, 6, 6, 0, 0, 0, time.UTC), nil)
	if err != nil || created2 || id1 != id2 {
		t.Fatalf("idempotency: id1=%d id2=%d created2=%v err=%v", id1, id2, created2, err)
	}
	sent, failed, err := svc.ProcessDue(ctx, sender, 10, time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC))
	if err != nil || sent != 1 || failed != 0 {
		t.Fatalf("process: sent=%d failed=%d err=%v", sent, failed, err)
	}
	if len(sender.sent) != 1 {
		t.Fatalf("sender dipanggil %d kali", len(sender.sent))
	}
	// Retry on SENT must fail.
	if err := svc.Retry(ctx, id1); err == nil {
		t.Fatalf("retry SENT seharusnya gagal")
	}
}

func TestNotify_FailureRetryNoDuplicate(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	bad := &fakeSender{fail: true}
	good := &fakeSender{}
	id, _, err := svc.Enqueue(ctx, 1, 1, "TASK_REMINDER", "CLASS", 1, "tasks:1:2026-10-06", "tugas", "", time.Date(2026, 10, 6, 17, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	sent, failed, _ := svc.ProcessDue(ctx, bad, 10, time.Date(2026, 10, 6, 18, 0, 0, 0, time.UTC))
	if sent != 0 || failed != 1 {
		t.Fatalf("fail path: sent=%d failed=%d", sent, failed)
	}
	items, _ := svc.List(ctx, 1, "FAILED", 10)
	if len(items) != 1 || items[0].Attempts != 1 || items[0].LastError == nil {
		t.Fatalf("list FAILED tidak tepat: %+v", items)
	}
	if err := svc.Retry(ctx, id); err != nil {
		t.Fatal(err)
	}
	sent, failed, _ = svc.ProcessDue(ctx, good, 10, time.Date(2026, 10, 6, 19, 0, 0, 0, time.UTC))
	if sent != 1 || failed != 0 {
		t.Fatalf("retry path: sent=%d failed=%d", sent, failed)
	}
	items, _ = svc.List(ctx, 1, "SENT", 10)
	if len(items) != 1 || items[0].Attempts != 2 {
		t.Fatalf("attempts diharapkan 2: %+v", items)
	}
}

func TestNotify_SupersedeOnRevoke(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	if _, _, err := svc.Enqueue(ctx, 1, 1, "SCHEDULE_CHANGE", "TEACHING_EVENT", 9, "event-publish:9", "perubahan", "", time.Now().UTC(), nil); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.EnqueueEventRevoked(ctx, 1, 9, "koreksi", nil); err != nil {
		t.Fatal(err)
	}
	items, _ := svc.List(ctx, 1, "SUPERSEDED", 10)
	if len(items) != 1 {
		t.Fatalf("superseded diharapkan 1, didapat %d", len(items))
	}
}

func TestNotify_Schedulers(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	// 06:00 WIB = 23:00 UTC previous day.
	morning := time.Date(2026, 10, 5, 23, 0, 0, 0, time.UTC)
	n, err := svc.EnsureDailySummaries(ctx, morning)
	if err != nil || n != 1 {
		t.Fatalf("daily: n=%d err=%v", n, err)
	}
	var scheduled string
	if err := svc.db.QueryRow(`SELECT scheduled_at FROM notification_messages WHERE idempotency_key='daily:1:2026-10-06'`).Scan(&scheduled); err != nil {
		t.Fatal(err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, scheduled)
	if err != nil || !parsed.Equal(morning) {
		t.Fatalf("waktu pengingat pagi=%q, ingin %s (UTC): %v", scheduled, morning, err)
	}
	n, _ = svc.EnsureDailySummaries(ctx, morning)
	if n != 0 {
		t.Fatalf("daily idempoten: n=%d", n)
	}
	// 17:00 WIB = 10:00 UTC.
	afternoon := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	// Tanpa tugas aktif, pengingat sore harus senyap (Opsi B: n == 0).
	n, err = svc.EnsureTaskReminders(ctx, afternoon)
	if err != nil || n != 0 {
		t.Fatalf("tasks tanpa tugas harus senyap: n=%d err=%v", n, err)
	}
	text, err := svc.BuildDailyText(ctx, 1, "2026-10-06", "")
	if err != nil || text == "" {
		t.Fatalf("build daily: %v", err)
	}

	// Masukkan 1 tugas aktif untuk menguji pengingat saat ada tugas.
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO users (id, identity_key, display_name, password_hash) VALUES (1, '+6281111111111', 'User 1', 'hash')`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO courses (id, code, name) VALUES (1, 'CS101', 'Pemrograman')`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name) VALUES (1, 1, 1, 'TEORI', 'Pemrograman (Teori)')`); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `INSERT INTO tasks (course_offering_id, title, instructions, submission_text, deadline_at, publication_status, published_at, created_by_user_id) VALUES (1, 'Tugas 1', 'Kerjakan soal 1-5', 'Kirim via email', '2026-10-10T23:59:00Z', 'PUBLISHED', '2026-10-06T08:00:00Z', 1)`); err != nil {
		t.Fatal(err)
	}

	// Dengan tugas aktif, pengingat sore mengantrekan 1 pesan.
	n, err = svc.EnsureTaskReminders(ctx, afternoon)
	if err != nil || n != 1 {
		t.Fatalf("tasks dengan tugas aktif: n=%d err=%v", n, err)
	}
	taskText, err := svc.BuildTaskText(ctx, 1, "")
	if err != nil || taskText == "" {
		t.Fatalf("build tasks: %v", err)
	}
}

func TestNotify_MorningReminderCatchesUpAfterRestart(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	late := time.Date(2026, 10, 5, 23, 20, 0, 0, time.UTC)
	n, err := svc.EnsureDailySummaries(ctx, late)
	if err != nil || n != 1 {
		t.Fatalf("20 menit terlambat: n=%d err=%v", n, err)
	}
	n, err = svc.EnsureDailySummaries(ctx, late.Add(10*time.Minute))
	if err != nil || n != 0 {
		t.Fatalf("harus idempoten: n=%d err=%v", n, err)
	}
}

func TestNotify_ReapsStaleProcessing(t *testing.T) {
	svc, ctx := newNotifyTestDB(t)
	sender := &fakeSender{}
	id, _, err := svc.Enqueue(ctx, 1, 1, "DAILY_SUMMARY", "CLASS", 1, "daily:1:2026-10-07", "pagi", "", time.Date(2026, 10, 7, 6, 0, 0, 0, time.UTC), nil)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate a worker crash: row stuck in PROCESSING with old heartbeat.
	old := time.Date(2026, 10, 7, 6, 0, 0, 0, time.UTC).Format(time.RFC3339Nano)
	if _, err := svc.db.ExecContext(ctx, `UPDATE notification_messages SET status='PROCESSING', updated_at=? WHERE id=?`, old, id); err != nil {
		t.Fatal(err)
	}
	sent, failed, err := svc.ProcessDue(ctx, sender, 10, time.Date(2026, 10, 7, 8, 0, 0, 0, time.UTC))
	if err != nil || sent != 1 || failed != 0 {
		t.Fatalf("reaped process: sent=%d failed=%d err=%v", sent, failed, err)
	}
	items, _ := svc.List(ctx, 1, "SENT", 10)
	if len(items) != 1 {
		t.Fatalf("pesan reaped harus SENT, didapat %+v", items)
	}
}
