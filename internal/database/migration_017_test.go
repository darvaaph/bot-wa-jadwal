package database

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewSnapshotMigration_PreservesDataAndClearsUnprovenReviews(t *testing.T) {
	db, err := OpenPool(filepath.Join(t.TempDir(), "pre17.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	oldSchema := strings.Replace(schemaSQL, "reviewed_version >= 1 AND reviewed_version <= version", "reviewed_version = version", 1)
	if _, err := db.Exec(oldSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,name TEXT NOT NULL UNIQUE)`); err != nil {
		t.Fatal(err)
	}
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.version >= 17 {
			break
		}
		if m.version == 16 {
			if err := rebuildReferencedTable(context.Background(), db, m, "teaching_events"); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if _, err := db.Exec(m.sql); err != nil {
			t.Fatal(m.version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations VALUES(?,?)`, m.version, m.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(1,'QA','qa','TI',2026,'A'); INSERT INTO users(id,identity_key,display_name,password_hash) VALUES(1,'qa','QA','disabled'); INSERT INTO role_assignments(id,user_id,role,scope_type) VALUES(1,1,'SYSTEM_ADMIN','GLOBAL'); INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(1,1,'2026/2027','GANJIL','2026-09-01','2027-01-31'); INSERT INTO courses(id,code,name) VALUES(1,'QA','QA'); INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(1,1,1,'TEORI','QA');
	INSERT INTO tasks(id,course_offering_id,created_by_user_id,title,instructions,deadline_at,submission_url,publication_status,published_at,review_state,reviewed_version,version) VALUES(1,1,1,'Keep title','Keep instructions','2026-11-09T00:00:00Z','https://example.com','PUBLISHED',CURRENT_TIMESTAMP,'APPROVED',2,2),(2,1,1,'Reviewed','Instructions','2026-11-09T00:00:00Z','https://example.com','PUBLISHED',CURRENT_TIMESTAMP,'APPROVED',1,1);
	INSERT INTO task_reviews(task_id,reviewer_user_id,reviewer_role_assignment_id,task_version,decision) VALUES(1,1,1,1,'APPROVED'),(2,1,1,1,'APPROVED'); INSERT INTO materials(class_id,task_id,title,material_type,url,created_by_user_id) VALUES(1,1,'Keep material','DOCUMENT','https://example.com',1); CREATE INDEX custom_task_index ON tasks(title); CREATE TRIGGER custom_task_trigger BEFORE DELETE ON tasks WHEN OLD.id=999 BEGIN SELECT RAISE(ABORT,'protected'); END;`); err != nil {
		t.Fatal(err)
	}
	if err := applyMigrations(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	var title, state string
	var version, reviewed int
	if err := db.QueryRow(`SELECT title,version,review_state,COALESCE(reviewed_version,0) FROM tasks WHERE id=1`).Scan(&title, &version, &state, &reviewed); err != nil {
		t.Fatal(err)
	}
	if title != "Keep title" || version != 2 || state != "NOT_REVIEWED" || reviewed != 0 {
		t.Fatal(title, version, state, reviewed)
	}
	if _, err := db.Exec(`UPDATE tasks SET version=version+1,completed_at=CURRENT_TIMESTAMP WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT reviewed_version FROM tasks WHERE id=2`).Scan(&reviewed); err != nil || reviewed != 1 {
		t.Fatal(reviewed, err)
	}
	var references, objects int
	db.QueryRow(`SELECT COUNT(*) FROM materials WHERE task_id=1`).Scan(&references)
	db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name IN ('custom_task_index','custom_task_trigger')`).Scan(&objects)
	if references != 1 || objects != 2 {
		t.Fatal(references, objects)
	}
	rows, err := db.Query(`PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign key violation")
	}
}
