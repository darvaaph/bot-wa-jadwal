package notify

import (
	"strings"
	"testing"
	"time"
)

func TestExamReminder_TomorrowWithoutTasks(t *testing.T) {
	s, ctx := newNotifyTestDB(t)
	_, err := s.db.Exec(`INSERT INTO users(id,identity_key,display_name,password_hash) VALUES(1,'qa','QA','disabled');
 INSERT INTO courses(id,code,name) VALUES(1,'QA','Aljabar');
 INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(1,1,1,'THEORY','Aljabar');
 INSERT INTO rooms(id,code,name) VALUES(1,'LAB','Lab Ujian');
 INSERT INTO teaching_events(id,event_kind,starts_at,ends_at,room_id,reason,lifecycle_status,published_by_user_id,published_at)
 VALUES(1,'EXAM_UAS','2026-11-09T18:00:00Z','2026-11-09T20:00:00Z',1,'Tutup buku. Pengawas: Dani','PUBLISHED',1,CURRENT_TIMESTAMP);
 INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(1,1,'OWNER','ACCEPTED');`)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	text, err := s.BuildTaskText(ctx, 1, "", now)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{"H-1", "[UAS]", "01:00 - 03:00", "LAB", "Pengawas: Dani", "/c/d4-ti-2024-a-n"} {
		if !strings.Contains(text, part) {
			t.Fatalf("missing %s: %s", part, text)
		}
	}
	for i := 0; i < 2; i++ {
		n, err := s.EnsureTaskReminders(ctx, now)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1-i {
			t.Fatalf("enqueued %d", n)
		}
	}
	if _, err := s.db.Exec(`UPDATE teaching_events SET lifecycle_status='REVOKED',revoked_by_user_id=1,revoked_at=CURRENT_TIMESTAMP,revocation_reason='Batal'`); err != nil {
		t.Fatal(err)
	}
	text, err = s.BuildTaskText(ctx, 1, "", now)
	if err != nil || text != "" {
		t.Fatal(text, err)
	}
}
