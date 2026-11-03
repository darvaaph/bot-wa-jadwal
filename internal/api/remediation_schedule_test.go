package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/notify"
)

func TestRemediation_ExamDraftUpdateAtomic(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; UPDATE teaching_events SET lifecycle_status='DRAFT',event_kind='EXAM_UTS',published_at=NULL,published_by_user_id=NULL;`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281298765432", "password123")
	payload := map[string]any{"version": 1, "owner_offering_id": 1, "event_kind": "EXAM_UTS", "starts_at": "2026-11-09T01:00:00Z", "ends_at": "2026-11-09T02:00:00Z", "reason": "Changed"}
	if _, err := db.Exec(`CREATE TRIGGER fail_draft_audit BEFORE INSERT ON audit_logs WHEN NEW.action='UPDATE_TEACHING_EVENT' BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	w := remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var reason string
	var version, count int
	db.QueryRow(`SELECT reason,version FROM teaching_events WHERE id=1`).Scan(&reason, &version)
	if reason != "Kuliah Tambahan" || version != 1 {
		t.Fatal("draft lost", reason, version)
	}
	db.Exec(`DROP TRIGGER fail_draft_audit`)
	w = remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 409 {
		t.Fatal("stale update accepted", w.Code, w.Body.String())
	}
	db.QueryRow(`SELECT COUNT(*) FROM teaching_events`).Scan(&count)
	if count != 1 {
		t.Fatal("draft duplicated", count)
	}
}

func TestRemediation_SchedulePublishAndRevokeOutboxAtomic(t *testing.T) {
	db, s := setupV1TestEnv(t)
	db.SetMaxOpenConns(1)
	token := helperLogin(t, s, "+6281234567890", "password123")
	if _, err := db.Exec(`INSERT INTO notification_messages(class_id,whatsapp_channel_id,event_type,entity_type,entity_id,idempotency_key,payload_json,status,scheduled_at) VALUES(1,1,'SCHEDULE_REPLACEMENT','TEACHING_EVENT',1,'old','{}','PENDING',CURRENT_TIMESTAMP); CREATE TRIGGER fail_revoke_outbox BEFORE INSERT ON notification_messages WHEN NEW.event_type='SCHEDULE_REVOKED' BEGIN SELECT RAISE(ABORT,'injected'); END;`); err != nil {
		t.Fatal(err)
	}
	payload := map[string]any{"version": 1, "reason": "Dibatalkan untuk koreksi jadwal"}
	w := remediationRequest(s, "POST", "/api/v1/teaching-events/1/revoke", token, payload)
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var lifecycle, status string
	db.QueryRow(`SELECT lifecycle_status FROM teaching_events WHERE id=1`).Scan(&lifecycle)
	db.QueryRow(`SELECT status FROM notification_messages WHERE idempotency_key='old'`).Scan(&status)
	if lifecycle != "PUBLISHED" || status != "PENDING" {
		t.Fatal("partial revoke", lifecycle, status)
	}
	db.Exec(`DROP TRIGGER fail_revoke_outbox`)
	w = remediationRequest(s, "POST", "/api/v1/teaching-events/1/revoke", token, payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	db.QueryRow(`SELECT status FROM notification_messages WHERE idempotency_key='old'`).Scan(&status)
	if status != "SUPERSEDED" {
		t.Fatal(status)
	}
}

func TestRemediation_BotPortalEffectiveParity(t *testing.T) {
	for _, kind := range []string{"REPLACEMENT", "SESSION_CANCELLED", "HOLIDAY", "EXAM_UTS", "EXAM_UAS"} {
		t.Run(kind, func(t *testing.T) {
			db, s := setupV1TestEnv(t)
			seedSecondClass(t, db)
			if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; UPDATE schedule_patterns SET effective_from='2026-09-01'; UPDATE teaching_events SET event_kind=?,origin_schedule_pattern_id=1,origin_occurrence_date='2026-11-09',starts_at='2026-11-11T01:00:00Z',ends_at='2026-11-11T02:00:00Z' WHERE id=1; INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(1,2,'PARTICIPANT','ACCEPTED');`, kind); err != nil {
				t.Fatal(err)
			}
			service := bot.NewV1BotService(db)
			loc, _ := time.LoadLocation("Asia/Jakarta")
			for _, classID := range []int64{1, 2} {
				slug := "d4-ti-2024-a"
				if classID == 2 {
					slug = "d4-ti-2024-b"
				}
				for _, date := range []string{"2026-11-09", "2026-11-11", "2026-11-16"} {
					day, _ := time.ParseInLocation("2006-01-02", date, loc)
					_, _, _, _, items, err := service.ResolveDailyItems(context.Background(), classID, day)
					if err != nil {
						t.Fatal(err)
					}
					w := remediationRequest(s, "GET", "/api/v1/portal/"+slug+"/schedule?date="+date, "", nil)
					if w.Code != 200 {
						t.Fatal(w.Code, w.Body.String())
					}
					var res struct {
						Data struct {
							Items []map[string]any `json:"items"`
						} `json:"data"`
					}
					if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
						t.Fatal(err)
					}
					if len(items) != len(res.Data.Items) {
						t.Fatalf("class %d %s: bot=%+v portal=%s", classID, date, items, w.Body.String())
					}
					for i, it := range items {
						if it.StartTime != res.Data.Items[i]["starts_at"] || it.Course != res.Data.Items[i]["offering"] {
							t.Fatal("different business session", items, res.Data.Items)
						}
					}
				}
			}
		})
	}
}

func TestRemediation_ParticipationOneClassAndStateMachine(t *testing.T) {
	db, s := setupV1TestEnv(t)
	seedSecondClass(t, db)
	if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; INSERT INTO classes(id,code,slug,study_program,cohort_year,group_label) VALUES(3,'C','c','TI',2026,'C'); INSERT INTO semesters(id,class_id,academic_year,term,starts_on,ends_on) VALUES(3,3,'2026/2027','GANJIL','2026-09-01','2027-01-31'); INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(3,3,1,'TEORI','Third'); INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status) VALUES(1,2,'PARTICIPANT','PENDING'),(1,3,'PARTICIPANT','PENDING');`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281111111111", "password123")
	w := remediationRequest(s, "POST", "/api/v1/admin/support/enter", token, map[string]any{"class_slug": "d4-ti-2024-b", "reason": "Pemulihan partisipasi kelas"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = remediationRequest(s, "POST", "/api/v1/teaching-events/1/participation", token, map[string]any{"action": "accept"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var second, third string
	var actor int
	var responded string
	db.QueryRow(`SELECT participation_status,responded_by_user_id,responded_at FROM teaching_event_offerings WHERE course_offering_id=2`).Scan(&second, &actor, &responded)
	db.QueryRow(`SELECT participation_status FROM teaching_event_offerings WHERE course_offering_id=3`).Scan(&third)
	if second != "ACCEPTED" || third != "PENDING" || actor != 3 || responded == "" {
		t.Fatal(second, third, actor, responded)
	}
	w = remediationRequest(s, "POST", "/api/v1/teaching-events/1/participation", token, map[string]any{"action": "leave"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = remediationRequest(s, "POST", "/api/v1/teaching-events/1/participation", token, map[string]any{"action": "accept"})
	if w.Code != 409 {
		t.Fatal("REMOVED accepted without reinvite", w.Code, w.Body.String())
	}
}

func TestRemediation_RestoreCorrectionWithoutChannelReconciles(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE whatsapp_channels SET status='REVOKED'`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281111111111", "password123")
	w := remediationRequest(s, "POST", "/api/v1/backups", token, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "Backup untuk regresi restore"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)
	if _, err := db.Exec(`UPDATE tasks SET title='Changed after backup' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/v1/backups/%d", created.Data.ID)
	w = remediationRequest(s, "POST", base+"/restore-preview", token, map[string]any{})
	var preview struct {
		Data struct {
			Token string `json:"preview_token"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &preview)
	w = remediationRequest(s, "POST", base+"/restore-execute", token, map[string]any{"reason": "Koreksi dari backup uji", "preview_token": preview.Data.Token})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var channel, scheduled *string
	if err := db.QueryRow(`SELECT whatsapp_channel_id,scheduled_at FROM notification_messages WHERE event_type='ACADEMIC_RESTORE_CORRECTION'`).Scan(&channel, &scheduled); err != nil {
		t.Fatal(err)
	}
	if channel != nil || scheduled != nil {
		t.Fatal("waiting-channel contract", channel, scheduled)
	}
	if _, err := db.Exec(`INSERT INTO whatsapp_channels(class_id,jid,channel_type,display_name) VALUES(1,'restore-test@g.us','GROUP','Test')`); err != nil {
		t.Fatal(err)
	}
	n, err := notify.NewService(db).ReconcilePendingChannels(context.Background(), nil)
	if err != nil || n != 1 {
		t.Fatal(n, err)
	}
}

func TestRemediation_LegacyCannotReadCodeOrArchived(t *testing.T) {
	db, s := setupV1TestEnv(t)
	for _, sql := range []string{`UPDATE class_settings SET portal_access_mode='CODE',portal_code_hash='test-hash'`, `UPDATE class_settings SET portal_access_mode='LINK',portal_code_hash=NULL; UPDATE semesters SET status='ARCHIVED',archived_at=CURRENT_TIMESTAMP`} {
		if _, err := db.Exec(sql); err != nil {
			t.Fatal(err)
		}
		w := remediationRequest(s, "GET", "/api/tasks", "", nil)
		if w.Code != 200 || strings.Contains(w.Body.String(), "Tugas Algoritma") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestRemediation_SharedDraftPreservesResponsesAndArchives(t *testing.T) {
	db, s := setupV1TestEnv(t)
	seedSecondClass(t, db)
	if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; UPDATE teaching_events SET lifecycle_status='DRAFT',event_kind='EXAM_UTS',published_at=NULL,published_by_user_id=NULL; INSERT INTO teaching_event_offerings(teaching_event_id,course_offering_id,participation_role,participation_status,responded_by_user_id,responded_at) VALUES(1,2,'PARTICIPANT','ACCEPTED',1,'2026-11-01T00:00:00Z'); INSERT INTO course_offerings(id,semester_id,course_id,activity_type,display_name) VALUES(3,1,1,'PRAKTIKUM','Same class');`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	payload := map[string]any{"version": 1, "owner_offering_id": 1, "event_kind": "EXAM_UTS", "starts_at": "2026-11-09T01:00:00Z", "ends_at": "2026-11-09T02:00:00Z", "reason": "Changed"}
	for i := 1; i <= 2; i++ {
		payload["version"] = i
		if i == 2 {
			payload["participant_offering_ids"] = []int{2}
		}
		w := remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var status, responded string
		var actor int
		if err := db.QueryRow(`SELECT participation_status,responded_by_user_id,responded_at FROM teaching_event_offerings WHERE teaching_event_id=1 AND course_offering_id=2`).Scan(&status, &actor, &responded); err != nil {
			t.Fatal(err)
		}
		if status != "ACCEPTED" || actor != 1 || responded != "2026-11-01T00:00:00Z" {
			t.Fatal("response reset", status, actor, responded)
		}
	}
	payload["participant_offering_ids"] = []int{3}
	w := remediationRequest(s, "POST", "/api/v1/teaching-events", token, payload)
	if w.Code != 422 {
		t.Fatal("same-class participant accepted", w.Code, w.Body.String())
	}
	if _, err := db.Exec(`UPDATE semesters SET status='ARCHIVED',archived_at=CURRENT_TIMESTAMP WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	payload["version"] = 3
	delete(payload, "participant_offering_ids")
	w = remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 403 {
		t.Fatal("archived participant mutated", w.Code, w.Body.String())
	}
	w = remediationRequest(s, "DELETE", "/api/v1/teaching-events/1?version=3", token, nil)
	if w.Code != 403 {
		t.Fatal("archived participation deleted", w.Code, w.Body.String())
	}
}

func TestRemediation_SupportDraftDeletionAuditRollsBack(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE teaching_events SET lifecycle_status='DRAFT',published_at=NULL,published_by_user_id=NULL; CREATE TRIGGER fail_delete_event_audit BEFORE INSERT ON audit_logs WHEN NEW.action='DELETE_TEACHING_EVENT' BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281111111111", "password123")
	w := remediationRequest(s, "POST", "/api/v1/admin/support/enter", token, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "Pemulihan draf bermasalah"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = remediationRequest(s, "DELETE", "/api/v1/teaching-events/1?version=1", token, nil)
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var children int
	db.QueryRow(`SELECT COUNT(*) FROM teaching_event_offerings WHERE teaching_event_id=1`).Scan(&children)
	if children != 1 {
		t.Fatal("partial deletion", children)
	}
	db.Exec(`DROP TRIGGER fail_delete_event_audit`)
	w = remediationRequest(s, "DELETE", "/api/v1/teaching-events/1?version=1", token, nil)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var grantID int
	if err := db.QueryRow(`SELECT json_extract(actor_context_json,'$.support.support_grant_id') FROM audit_logs WHERE action='DELETE_TEACHING_EVENT'`).Scan(&grantID); err != nil || grantID == 0 {
		t.Fatal("missing support audit", grantID, err)
	}
}

func TestRemediation_DraftConfirmationsPreservedAndSuperseded(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; UPDATE teaching_events SET lifecycle_status='DRAFT',event_kind='EXAM_UTS',published_at=NULL,published_by_user_id=NULL,starts_at='2026-11-09T01:00:00Z',ends_at='2026-11-09T02:00:00Z'; INSERT INTO room_confirmations(teaching_event_id,room_id,confirmation_status,external_contact,note,recorded_by_user_id,confirmed_at) VALUES(1,1,'CONFIRMED','TU fixture','Keep original evidence',1,CURRENT_TIMESTAMP);`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	payload := map[string]any{"version": 1, "owner_offering_id": 1, "event_kind": "EXAM_UTS", "starts_at": "2026-11-09T01:00:00Z", "ends_at": "2026-11-09T02:00:00Z", "room_id": 1, "reason": "Reason only"}
	w := remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var active, total int
	var contact, note string
	db.QueryRow(`SELECT COUNT(*) FROM room_confirmations WHERE teaching_event_id=1 AND superseded_at IS NULL`).Scan(&active)
	if active != 1 {
		t.Fatal("reason-only edit invalidated confirmation")
	}
	payload["version"] = 2
	payload["ends_at"] = "2026-11-09T03:00:00Z"
	w = remediationRequest(s, "PATCH", "/api/v1/teaching-events/1", token, payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	db.QueryRow(`SELECT COUNT(*) FROM room_confirmations WHERE teaching_event_id=1 AND superseded_at IS NULL`).Scan(&active)
	db.QueryRow(`SELECT COUNT(*),external_contact,note FROM room_confirmations WHERE teaching_event_id=1`).Scan(&total, &contact, &note)
	if active != 0 || total != 1 || contact != "TU fixture" || note != "Keep original evidence" {
		t.Fatal(active, total, contact, note)
	}
}

func TestRemediation_DraftDeleteVersionFailureRollsBackChildren(t *testing.T) {
	db, s := setupV1TestEnv(t)
	if _, err := db.Exec(`UPDATE teaching_events SET lifecycle_status='DRAFT',published_at=NULL,published_by_user_id=NULL; CREATE TRIGGER change_draft_during_delete BEFORE DELETE ON teaching_event_offerings BEGIN UPDATE teaching_events SET version=version+1 WHERE id=OLD.teaching_event_id; END;`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	w := remediationRequest(s, "DELETE", "/api/v1/teaching-events/1?version=1", token, nil)
	if w.Code != 409 {
		t.Fatal("stale delete returned success", w.Code, w.Body.String())
	}
	var version, children int
	db.QueryRow(`SELECT version FROM teaching_events WHERE id=1`).Scan(&version)
	db.QueryRow(`SELECT COUNT(*) FROM teaching_event_offerings WHERE teaching_event_id=1`).Scan(&children)
	if version != 1 || children != 1 {
		t.Fatal("children lost", version, children)
	}
}

func TestRemediation_PublicationOutboxFailureAndIdempotency(t *testing.T) {
	db, s := setupV1TestEnv(t)
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`UPDATE semesters SET starts_on='2026-09-01',ends_on='2027-01-31'; UPDATE teaching_events SET lifecycle_status='DRAFT',published_at=NULL,published_by_user_id=NULL,room_id=NULL,starts_at='2026-11-07T01:00:00Z',ends_at='2026-11-07T02:00:00Z'; CREATE TRIGGER fail_publish_outbox BEFORE INSERT ON notification_messages BEGIN SELECT RAISE(ABORT,'injected'); END;`); err != nil {
		t.Fatal(err)
	}
	token := helperLogin(t, s, "+6281234567890", "password123")
	w := remediationRequest(s, "POST", "/api/v1/tasks", token, map[string]any{"offering_id": 1, "title": "Atomic publish", "instructions": "Instructions", "deadline_at": "2026-11-09T01:00:00Z", "submission_url": "https://example.com", "save_as": "published"})
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var tasks int
	db.QueryRow(`SELECT COUNT(*) FROM tasks`).Scan(&tasks)
	if tasks != 1 {
		t.Fatal("task publish partially committed")
	}
	publish := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "/api/v1/teaching-events/1/publish", strings.NewReader(`{"version":1}`))
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Idempotency-Key", "atomic-event")
		w := httptest.NewRecorder()
		s.httpServer.Handler.ServeHTTP(w, r)
		return w
	}
	w = publish()
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var state string
	var version, audits int
	db.QueryRow(`SELECT lifecycle_status,version FROM teaching_events WHERE id=1`).Scan(&state, &version)
	db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action='PUBLISH_TEACHING_EVENT'`).Scan(&audits)
	if state != "DRAFT" || version != 1 || audits != 0 {
		t.Fatal(state, version, audits)
	}
	db.Exec(`DROP TRIGGER fail_publish_outbox`)
	for i := 0; i < 2; i++ {
		w = publish()
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var intents int
	db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE entity_type='TEACHING_EVENT' AND entity_id=1`).Scan(&intents)
	if intents != 1 {
		t.Fatal("idempotent publication duplicated intent", intents)
	}
}
