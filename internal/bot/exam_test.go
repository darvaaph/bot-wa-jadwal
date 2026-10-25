package bot

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExamSchedule_GranularRescheduleAndRevoke(t *testing.T) {
	for _, kind := range []string{"EXAM_UTS", "EXAM_UAS"} {
		t.Run(kind, func(t *testing.T) {
			s := setupTestV1DB(t)
			ctx := context.Background()
			loc := time.FixedZone("WIB", 7*3600)
			monday := time.Date(2025, 1, 6, 0, 0, 0, 0, loc)
			_, err := s.db.Exec(`INSERT INTO schedule_patterns(id,course_offering_id,day_of_week,start_time,end_time,effective_from) VALUES(2,2,1,'10:00','12:00','2024-09-01');
   INSERT INTO teaching_events(id,origin_schedule_pattern_id,origin_occurrence_date,event_kind,starts_at,ends_at,room_id,reason)
   VALUES(1,1,'2025-01-06',?,'2025-01-06T01:00:00Z','2025-01-06T03:00:00Z',1,'Buka Buku, Pengawas: Dani');
   INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(1,1,'OWNER','ACCEPTED');`, kind)
			if err != nil {
				t.Fatal(err)
			}
			check := func(day time.Time, want int, exam bool) {
				t.Helper()
				_, _, _, _, items, err := s.ResolveDailyItems(ctx, 1, day)
				if err != nil {
					t.Fatal(err)
				}
				if len(items) != want {
					t.Fatalf("%v: %+v", day, items)
				}
				found := false
				for _, it := range items {
					if it.Status == strings.TrimPrefix(kind, "EXAM_") {
						found = true
					}
					if it.Course == "Struktur Data (Praktik)" && it.Status != "REGULAR" {
						t.Fatal(it)
					}
				}
				if found != exam {
					t.Fatal(items)
				}
			}
			check(monday, 2, false)
			if _, err := s.db.Exec(`UPDATE teaching_events SET lifecycle_status='PUBLISHED',published_by_user_id=1,published_at=CURRENT_TIMESTAMP WHERE id=1`); err != nil {
				t.Fatal(err)
			}
			check(monday, 2, true)
			text, err := s.GetSchedule(ctx, 1, monday)
			if err != nil {
				t.Fatal(err)
			}
			for _, part := range []string{"[" + strings.TrimPrefix(kind, "EXAM_") + "]", "Ruang: LAB-1", "Buka Buku, Pengawas: Dani", "/c/d4-ti-2024-a"} {
				if !strings.Contains(text, part) {
					t.Fatalf("missing %q: %s", part, text)
				}
			}
			if _, err := s.db.Exec(`UPDATE teaching_events SET starts_at='2025-01-10T18:00:00Z',ends_at='2025-01-10T20:00:00Z' WHERE id=1`); err != nil {
				t.Fatal(err)
			}
			check(monday, 1, false)
			check(time.Date(2025, 1, 11, 0, 0, 0, 0, loc), 1, true)
			check(monday.AddDate(0, 0, 7), 2, false)
			if _, err := s.db.Exec(`UPDATE teaching_events SET lifecycle_status='REVOKED',revoked_by_user_id=1,revoked_at=CURRENT_TIMESTAMP,revocation_reason='Batal' WHERE id=1`); err != nil {
				t.Fatal(err)
			}
			check(monday, 2, false)
			check(time.Date(2025, 1, 11, 0, 0, 0, 0, loc), 0, false)
		})
	}
}
