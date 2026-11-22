package schedule

import (
	"bot-jadwal/internal/database"
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func examEventService(t *testing.T) *EventService {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "events.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(1,'QA','qa','TI',2026,'A');
 INSERT INTO users(id,identity_key,display_name,password_hash) VALUES(1,'qa','QA','disabled');
 INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(1,1,'2026/2027','GANJIL','2026-09-01','2027-01-31');
 INSERT INTO courses(id,code,name) VALUES(1,'QA','QA');
 INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(1,1,1,'THEORY','QA'),(2,1,1,'PRACTICE','QA Praktik');
 INSERT INTO schedule_patterns(id,course_offering_id,day_of_week,start_time,end_time) VALUES(1,1,1,'08:00','10:00'),(2,2,1,'10:00','12:00');`)
	if err != nil {
		t.Fatal(err)
	}
	return NewEventService(db)
}

func TestEventExam_DraftPublishRevoke(t *testing.T) {
	for _, kind := range []string{"EXAM_UTS", "EXAM_UAS"} {
		t.Run(kind, func(t *testing.T) {
			s := examEventService(t)
			ctx := context.Background()
			actor := EventActor{UserID: 1}
			pattern := int64(1)
			date := "2026-11-09"
			in := CreateDraftInput{OwnerOfferingID: 1, Kind: kind, StartsAt: "2026-11-09T01:00:00Z", EndsAt: "2026-11-09T03:00:00Z", OriginPatternID: &pattern, OriginDate: &date}
			draft, err := s.CreateDraft(ctx, actor, in)
			if err != nil {
				t.Fatal(err)
			}
			if draft.Kind != kind || draft.Lifecycle != "DRAFT" {
				t.Fatal(draft)
			}
			pub, err := s.Publish(ctx, actor, draft.ID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if pub.Lifecycle != "PUBLISHED" {
				t.Fatal(pub)
			}
			duplicate, err := s.CreateDraft(ctx, actor, in)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Publish(ctx, actor, duplicate.ID, nil); !errors.Is(err, ErrEventConflict) {
				t.Fatalf("duplicate: %v", err)
			}
			revoked, err := s.Revoke(ctx, actor, draft.ID, "Jadwal ujian dibatalkan")
			if err != nil {
				t.Fatal(err)
			}
			if revoked.Lifecycle != "REVOKED" {
				t.Fatal(revoked)
			}
			in.OriginPatternID = nil
			in.OriginDate = nil
			in.StartsAt = "2026-11-14T02:00:00Z"
			in.EndsAt = "2026-11-14T04:00:00Z"
			extra, err := s.CreateDraft(ctx, actor, in)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.Publish(ctx, actor, extra.ID, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestEventExam_RejectsInvalidOriginAndPermanent(t *testing.T) {
	s := examEventService(t)
	pattern := int64(1)
	other := int64(2)
	valid := "2026-11-09"
	invalid := "2026-02-30"
	for _, in := range []CreateDraftInput{
		{OriginPatternID: &pattern}, {OriginDate: &valid}, {OriginPatternID: &other, OriginDate: &valid},
		{OriginPatternID: &pattern, OriginDate: &invalid}, {OriginPatternID: &pattern, OriginDate: &valid, IsPermanent: true},
	} {
		in.OwnerOfferingID = 1
		in.Kind = "EXAM_UTS"
		in.StartsAt = "2026-11-09T01:00:00Z"
		in.EndsAt = "2026-11-09T03:00:00Z"
		if _, err := s.CreateDraft(context.Background(), EventActor{UserID: 1}, in); err == nil {
			t.Fatalf("accepted: %+v", in)
		}
	}
}
