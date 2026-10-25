package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestExamMigration_PreservesReferencesAndIndexes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := OpenPool(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE);`); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.version >= 16 {
			break
		}
		if _, err := db.Exec(m.sql); err != nil {
			t.Fatalf("migration %d: %v", m.version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations VALUES (?,?)`, m.version, m.name); err != nil {
			t.Fatal(err)
		}
	}
	_, err = db.Exec(`
 INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(1,'QA','qa','TI',2026,'A');
 INSERT INTO users(id,identity_key,display_name,password_hash) VALUES(1,'qa','QA','disabled');
 INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(1,1,'2026/2027','GANJIL','2026-09-01','2027-01-31');
 INSERT INTO courses(id,code,name) VALUES(1,'QA','QA');
 INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(1,1,1,'THEORY','QA');
 INSERT INTO rooms(id,code,name) VALUES(1,'R1','Ruang QA');
 INSERT INTO schedule_patterns(id,course_offering_id,day_of_week,start_time,end_time) VALUES(1,1,1,'08:00','10:00');
 INSERT INTO teaching_events(id,event_kind,starts_at,ends_at,room_id,reason,meeting_link,version,result_schedule_pattern_id)
 VALUES(42,'EXTRA','2026-11-09T01:00:00Z','2026-11-09T03:00:00Z',1,'catatan lama','https://example.org/kelas',7,1);
 INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(42,1,'OWNER','ACCEPTED');
 INSERT INTO room_confirmations(teaching_event_id,room_id,confirmation_status,recorded_by_user_id) VALUES(42,1,'PENDING',1);
 CREATE INDEX exam_custom_index ON teaching_events(reason);
 CREATE TRIGGER exam_custom_trigger BEFORE DELETE ON teaching_events WHEN OLD.id=999 BEGIN SELECT RAISE(ABORT,'protected'); END;`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO teaching_events(event_kind,starts_at,ends_at) VALUES('EXAM_UTS','a','b')`); err == nil {
		t.Fatal("old constraint accepted exam")
	}
	if err := applyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var reason, link string
	var version, parts, rooms, objects, fk int
	if err := db.QueryRow(`SELECT reason,meeting_link,version FROM teaching_events WHERE id=42`).Scan(&reason, &link, &version); err != nil {
		t.Fatal(err)
	}
	if reason != "catatan lama" || link != "https://example.org/kelas" || version != 7 {
		t.Fatal(reason, link, version)
	}
	for query, dest := range map[string]*int{
		`SELECT COUNT(*) FROM teaching_event_offerings WHERE teaching_event_id=42`:                                                         &parts,
		`SELECT COUNT(*) FROM room_confirmations WHERE teaching_event_id=42`:                                                               &rooms,
		`SELECT COUNT(*) FROM sqlite_master WHERE name IN ('exam_custom_index','exam_custom_trigger','uq_teaching_events_result_pattern')`: &objects,
		`PRAGMA foreign_keys`: &fk,
	} {
		if err := db.QueryRow(query).Scan(dest); err != nil {
			t.Fatal(err)
		}
	}
	if parts != 1 || rooms != 1 || objects != 3 || fk != 1 {
		t.Fatal(parts, rooms, objects, fk)
	}
	for _, kind := range []string{"EXAM_UTS", "EXAM_UAS"} {
		if _, err := db.Exec(`INSERT INTO teaching_events(event_kind,starts_at,ends_at) VALUES(?,'a','b')`, kind); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO teaching_events(event_kind,starts_at,ends_at,room_id) VALUES('EXTRA','a','b',999)`); err == nil {
		t.Fatal("foreign key disabled")
	}
	if _, err := db.Exec(`INSERT INTO teaching_events(event_kind,starts_at,ends_at,result_schedule_pattern_id) VALUES('EXTRA','a','b',1)`); err == nil {
		t.Fatal("unique index lost")
	}
	if err := applyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
}
