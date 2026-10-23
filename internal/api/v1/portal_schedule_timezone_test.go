package v1

import (
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

func TestPortalSchedule_UsesLocalClassDayForUTCEvent(t *testing.T) {
	db, err := database.InitDB(filepath.Join(t.TempDir(), "portal_schedule.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`INSERT INTO classes (id,code,slug,study_program,cohort_year,group_label) VALUES (1,'TI-A','ti-a','TI',2024,'A');
		INSERT INTO users (id,identity_key,display_name,password_hash) VALUES (1,'user:test','Test','disabled');
		INSERT INTO semesters (id,class_id,academic_year,term,starts_on,ends_on,status,published_at,activated_at)
		VALUES (1,1,'2024/2025','GANJIL','2024-09-01','2025-02-28','ACTIVE',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP);
		INSERT INTO courses (id,code,name) VALUES (1,'TI101','Algoritma');
		INSERT INTO course_offerings (id,semester_id,course_id,activity_type,display_name) VALUES (1,1,1,'THEORY','Algoritma');
		INSERT INTO teaching_events (id,event_kind,starts_at,ends_at,lifecycle_status,published_by_user_id,published_at)
		VALUES (1,'EXTRA','2025-01-05T18:00:00Z','2025-01-05T19:00:00Z','PUBLISHED',1,CURRENT_TIMESTAMP);
		INSERT INTO teaching_event_offerings (teaching_event_id,course_offering_id,participation_role,participation_status)
		VALUES (1,1,'OWNER','ACCEPTED')`)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	day := time.Date(2025, 1, 6, 0, 0, 0, 0, loc)
	items, err := (&PortalController{db: db}).getScheduleForDate(1, 1, day, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0]["starts_at"] != "01:00" {
		t.Fatalf("jadwal lokal tidak cocok: %+v", items)
	}
}
