package schedule

import (
	"context"
	"testing"
	"time"
)

func TestRemediation_SemesterDatesUseEachClassTimezone(t *testing.T) {
	db := seedConflictDB(t)
	if _, err := db.Exec(`INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(2,'C2','c2','TI',2024,'B'); INSERT INTO class_settings(class_id,timezone) VALUES(2,'Asia/Makassar'); INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(2,2,'2024/2025','GANJIL','2024-09-01','2025-01-31'); INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(3,2,1,'TEORI','Participant');`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		start, end         string
		owner, participant bool
	}{
		{"2024-09-01T00:30:00+07:00", "2024-09-01T01:30:00+07:00", true, true},
		{"2024-08-31T17:30:00Z", "2024-08-31T18:30:00Z", true, true},
		{"2025-01-31T16:30:00Z", "2025-01-31T17:00:00Z", true, false},
		{"2025-01-31T17:00:00Z", "2025-01-31T17:30:00Z", false, false},
	} {
		start, _ := time.Parse(time.RFC3339, tc.start)
		end, _ := time.Parse(time.RFC3339, tc.end)
		for off, want := range map[int64]bool{1: tc.owner, 3: tc.participant} {
			got, err := OfferingPeriodContains(context.Background(), db, off, start, end)
			if err != nil || got != want {
				t.Errorf("%d %s got %t want %t: %v", off, tc.start, got, want, err)
			}
		}
	}
}
