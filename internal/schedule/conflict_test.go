package schedule

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"bot-jadwal/internal/database"
)

func seedConflictDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.MigrateV1(db); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	must := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatalf("seed: %v (%s)", err, q)
		}
	}
	must(`INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label) VALUES (1,'C1','c1','TI',2024,'A')`)
	must(`INSERT INTO class_settings (class_id, timezone) VALUES (1,'Asia/Jakarta')`)
	must(`INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (1,1,'2024/2025','GANJIL','2024-09-01','2025-01-31','ACTIVE',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`)
	must(`INSERT INTO courses (id, code, name) VALUES (1,'MK1','Matkul 1'),(2,'MK2','Matkul 2')`)
	must(`INSERT INTO course_offerings (id, semester_id, course_id, activity_type, display_name) VALUES (1,1,1,'TEORI','MK1'),(2,1,2,'TEORI','MK2')`)
	must(`INSERT INTO rooms (id, code, name) VALUES (1,'R1','Ruang 1'),(2,'R2','Ruang 2')`)
	must(`INSERT INTO lecturers (id, code, full_name) VALUES (1,'D1','Dosen 1')`)
	must(`INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility) VALUES (1,1,'PRIMARY'),(2,1,'PRIMARY')`)
	must(`INSERT INTO users (id, identity_key, display_name, password_hash) VALUES (1,'u1','U1','x')`)
	must(`INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status) VALUES (1,1,1,1,'08:00','09:40','2024-09-01','ACTIVE')`)
	must(`INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, lifecycle_status, published_by_user_id, published_at, version) VALUES (10,'EXTRA','2024-10-07T10:00:00Z','2024-10-07T11:00:00Z',2,'PUBLISHED',1,CURRENT_TIMESTAMP,1)`)
	must(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (10,2,'OWNER','ACCEPTED')`)
	must(`INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, lifecycle_status, version) VALUES (11,'EXTRA','2024-10-07T14:00:00Z','2024-10-07T15:00:00Z',NULL,'DRAFT',1)`)
	must(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (11,1,'OWNER','ACCEPTED')`)
	must(`INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, lifecycle_status, published_by_user_id, published_at, revoked_by_user_id, revoked_at, revocation_reason, version) VALUES (12,'EXTRA','2024-10-07T14:00:00Z','2024-10-07T15:00:00Z',NULL,'REVOKED',1,CURRENT_TIMESTAMP,1,CURRENT_TIMESTAMP,'batal',2)`)
	must(`INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status) VALUES (12,1,'OWNER','ACCEPTED')`)
	return db
}

func hasCode(cs []Conflict, code string) bool {
	for _, c := range cs {
		if c.Code == code {
			return true
		}
	}
	return false
}

func TestConflict_AdjacentNoOverlap(t *testing.T) {
	a0 := time.Date(2024, 10, 7, 8, 0, 0, 0, time.UTC)
	a1 := time.Date(2024, 10, 7, 10, 0, 0, 0, time.UTC)
	b0 := time.Date(2024, 10, 7, 10, 0, 0, 0, time.UTC)
	b1 := time.Date(2024, 10, 7, 11, 0, 0, 0, time.UTC)
	if overlaps(a0, a1, b0, b1) {
		t.Fatal("interval bersentuhan tidak boleh overlap")
	}
	if !overlaps(a0, a1.Add(time.Minute), b0, b1) {
		t.Fatal("overlap 1 menit harus terdeteksi")
	}
}

func TestConflict_RoomLecturerClassOffering(t *testing.T) {
	db := seedConflictDB(t)
	ctx := context.Background()
	room1 := int64(1)
	// Candidate Senin 2024-10-07 08:30-09:00 offering 2, room 1, dosen 1:
	// bentrok pola offering 1 (kelas sama, room sama, dosen sama).
	cs, err := CheckConflicts(ctx, db, Candidate{
		OwnerClassID: 1, OwnerOfferingID: 2, RoomID: &room1, LecturerIDs: []int64{1},
		StartsAt: time.Date(2024, 10, 7, 8, 30, 0, 0, time.UTC),
		EndsAt:   time.Date(2024, 10, 7, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{CodeClassOverlap, CodeRoomOverlap, CodeLecturerOverlap} {
		if !hasCode(cs, want) {
			t.Fatalf("diharapkan %s, got %+v", want, cs)
		}
	}
	if !HasBlocking(cs) {
		t.Fatal("harus blocking")
	}
}

func TestConflict_DraftRevokedIgnored(t *testing.T) {
	db := seedConflictDB(t)
	ctx := context.Background()
	// Candidate 14:15-14:20 offering 1 tanpa room/dosen: hanya overlap DRAFT 11
	// dan REVOKED 12 yang harus diabaikan sebagai sesi aktif.
	cs, err := CheckConflicts(ctx, db, Candidate{
		OwnerClassID: 1, OwnerOfferingID: 1,
		StartsAt: time.Date(2024, 10, 7, 14, 15, 0, 0, time.UTC),
		EndsAt:   time.Date(2024, 10, 7, 14, 20, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(cs, CodeOfferingOverlap) || hasCode(cs, CodeClassOverlap) {
		t.Fatalf("DRAFT/REVOKED tidak boleh dihitung; got %+v", cs)
	}
}

func TestConflict_OutsideSemesterRejected(t *testing.T) {
	db := seedConflictDB(t)
	ctx := context.Background()
	cs, err := CheckConflicts(ctx, db, Candidate{
		OwnerClassID: 1, OwnerOfferingID: 1,
		StartsAt: time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC),
		EndsAt:   time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(cs, CodeOutsideOwnerSem) {
		t.Fatalf("di luar semester harus ditolak: %+v", cs)
	}
}

func TestConflict_PatternModeSelfExclude(t *testing.T) {
	db := seedConflictDB(t)
	ctx := context.Background()
	room1 := int64(1)
	pid := int64(1)
	cs, err := CheckConflicts(ctx, db, Candidate{
		OwnerClassID: 1, OwnerOfferingID: 1, RoomID: &room1,
		PatternDay: 1, PatternStart: "08:00", PatternEnd: "09:40",
		PatternDate: "2024-10-07", ExcludePatternID: &pid,
	})
	if err != nil {
		t.Fatal(err)
	}
	if hasCode(cs, CodeRoomOverlap) || hasCode(cs, CodeOfferingOverlap) {
		t.Fatalf("exclude diri sendiri harus bersih: %+v", cs)
	}
}

func TestConflict_PreviewPublishIdentical(t *testing.T) {
	db := seedConflictDB(t)
	ctx := context.Background()
	room2 := int64(2)
	mk := func() []Conflict {
		cs, err := CheckConflicts(ctx, db, Candidate{
			OwnerClassID: 1, OwnerOfferingID: 1, RoomID: &room2, LecturerIDs: []int64{1},
			StartsAt: time.Date(2024, 10, 7, 10, 30, 0, 0, time.UTC),
			EndsAt:   time.Date(2024, 10, 7, 11, 30, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatal(err)
		}
		return cs
	}
	a, b := mk(), mk()
	if len(a) != len(b) {
		t.Fatalf("preview vs publish berbeda: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("konflik tidak deterministik: %+v vs %+v", a[i], b[i])
		}
	}
}
