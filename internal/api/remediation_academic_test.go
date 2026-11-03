package api

import (
	"fmt"
	"testing"
)

func TestRemediation_SupportGrantRequiredAndAudited(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281111111111", "password123")
	payload := map[string]any{"version": 1, "title": "Support edit"}
	w := remediationRequest(s, "PATCH", "/api/v1/tasks/1", token, payload)
	if w.Code != 403 {
		t.Fatal("without grant", w.Code, w.Body.String())
	}
	w = remediationRequest(s, "POST", "/api/v1/admin/support/enter", token, map[string]any{"class_slug": "d4-ti-2024-a", "reason": "Pemulihan data tugas kelas"})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = remediationRequest(s, "PATCH", "/api/v1/tasks/1", token, payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var context string
	if err := db.QueryRow(`SELECT actor_context_json FROM audit_logs WHERE action='UPDATE_TASK' ORDER BY id DESC LIMIT 1`).Scan(&context); err != nil || context == "" {
		t.Fatal("missing support context", err)
	}
	var grantID int
	if err := db.QueryRow(`SELECT json_extract(actor_context_json,'$.support.support_grant_id') FROM audit_logs WHERE action='UPDATE_TASK' ORDER BY id DESC LIMIT 1`).Scan(&grantID); err != nil || grantID == 0 {
		t.Fatal(context, err)
	}
	if _, err := db.Exec(`UPDATE support_grants SET expires_at=datetime('now','-1 minute')`); err != nil {
		t.Fatal(err)
	}
	payload["version"] = 2
	w = remediationRequest(s, "PATCH", "/api/v1/tasks/1", token, payload)
	if w.Code != 403 {
		t.Fatal("expired grant", w.Code, w.Body.String())
	}
	// Global administration is independent of academic support.
	if w = remediationRequest(s, "GET", "/api/v1/admin/status", token, nil); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestRemediation_ArchivedSemesterMutationsDenied(t *testing.T) {
	db, s := setupV1TestEnv(t)
	km := helperLogin(t, s, "+6281234567890", "password123")
	pj := helperLogin(t, s, "+6281298765432", "password123")
	if _, err := db.Exec(`UPDATE semesters SET status='ARCHIVED',archived_at=CURRENT_TIMESTAMP; UPDATE teaching_events SET lifecycle_status='DRAFT',published_at=NULL,published_by_user_id=NULL;`); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{km, pj} {
		for _, tc := range []struct {
			method, path string
			payload      any
		}{
			{"PATCH", "/api/v1/tasks/1", map[string]any{"version": 1, "title": "Changed"}},
			{"POST", "/api/v1/tasks", map[string]any{"offering_id": 1, "title": "Archived draft", "save_as": "draft"}},
			{"POST", "/api/v1/tasks/1/complete", map[string]any{"version": 1}},
			{"POST", "/api/v1/tasks/1/uncomplete", map[string]any{"version": 1}},
			{"POST", "/api/v1/tasks/1/archive", map[string]any{"version": 1}},
			{"POST", "/api/v1/tasks/1/restore", map[string]any{"version": 1}},
			{"POST", "/api/v1/tasks/1/reviews", map[string]any{"task_version": 1, "decision": "APPROVED"}},
			{"POST", "/api/v1/materials", map[string]any{"class_slug": "d4-ti-2024-a", "offering_id": 1, "title": "Archived", "url": "https://example.com"}},
			{"PATCH", "/api/v1/materials/1", map[string]any{"version": 1, "title": "Changed"}},
			{"DELETE", "/api/v1/materials/1?version=1", nil},
			{"DELETE", "/api/v1/schedule/patterns/1?version=1", nil},
			{"DELETE", "/api/v1/teaching-events/1?version=1", nil},
			{"POST", "/api/v1/teaching-events/1/room-confirmations", map[string]any{"room_id": 1, "confirmation_status": "CONFIRMED", "external_contact": "TU"}},
		} {
			w := remediationRequest(s, tc.method, tc.path, token, tc.payload)
			if w.Code != 403 {
				t.Errorf("%s %s = %d %s", tc.method, tc.path, w.Code, w.Body.String())
			}
		}
	}
}

func TestRemediation_TaskReviewProofSurvivesStateChanges(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	if _, err := db.Exec(`INSERT INTO task_reviews(task_id,reviewer_user_id,reviewer_role_assignment_id,task_version,decision) VALUES(1,1,1,1,'APPROVED')`); err != nil {
		t.Fatal(err)
	}
	for i, action := range []string{"complete", "uncomplete", "archive", "restore"} {
		w := remediationRequest(s, "POST", "/api/v1/tasks/1/"+action, token, map[string]any{"version": i + 1})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
		var reviewed, proofs int
		if err := db.QueryRow(`SELECT reviewed_version,(SELECT COUNT(*) FROM task_reviews r WHERE r.task_id=t.id AND r.task_version=t.reviewed_version AND r.decision=t.review_state) FROM tasks t WHERE id=1`).Scan(&reviewed, &proofs); err != nil {
			t.Fatal(err)
		}
		if reviewed != 1 || proofs != 1 {
			t.Fatal("invented review", action, reviewed, proofs)
		}
	}
}

func TestRemediation_TaskOutboxFailureRollsBackAndEditSupersedes(t *testing.T) {
	db, s := setupV1TestEnv(t)
	db.SetMaxOpenConns(1)
	token := helperLogin(t, s, "+6281234567890", "password123")
	if _, err := db.Exec(`CREATE TRIGGER fail_outbox BEFORE INSERT ON notification_messages BEGIN SELECT RAISE(ABORT,'injected outbox failure'); END;`); err != nil {
		t.Fatal(err)
	}
	w := remediationRequest(s, "PATCH", "/api/v1/tasks/1", token, map[string]any{"version": 1, "title": "Must rollback"})
	if w.Code != 500 {
		t.Fatal(w.Code, w.Body.String())
	}
	var version, audits int
	db.QueryRow(`SELECT version FROM tasks WHERE id=1`).Scan(&version)
	db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE action='UPDATE_TASK'`).Scan(&audits)
	if version != 1 || audits != 0 {
		t.Fatal("partial commit", version, audits)
	}
	db.Exec(`DROP TRIGGER fail_outbox`)
	for i := 1; i <= 2; i++ {
		w = remediationRequest(s, "PATCH", "/api/v1/tasks/1", token, map[string]any{"version": i, "title": fmt.Sprintf("Revision %d", i)})
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var pending, stale int
	db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE status='PENDING'`).Scan(&pending)
	db.QueryRow(`SELECT COUNT(*) FROM notification_messages WHERE status='SUPERSEDED'`).Scan(&stale)
	if pending != 1 || stale != 1 {
		t.Fatal("stale notification", pending, stale)
	}
}

func TestRemediation_MaterialAuditAtomic(t *testing.T) {
	db, s := setupV1TestEnv(t)
	token := helperLogin(t, s, "+6281234567890", "password123")
	if _, err := db.Exec(`CREATE TRIGGER fail_material_audit BEFORE INSERT ON audit_logs WHEN NEW.entity_type='MATERIAL' BEGIN SELECT RAISE(ABORT,'injected audit failure'); END;`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		payload      any
	}{
		{"POST", "/api/v1/materials", map[string]any{"class_slug": "d4-ti-2024-a", "offering_id": 1, "title": "New", "url": "https://example.com"}},
		{"PATCH", "/api/v1/materials/1", map[string]any{"version": 1, "url": "https://example.com/changed"}},
		{"DELETE", "/api/v1/materials/1?version=1", nil},
	} {
		w := remediationRequest(s, tc.method, tc.path, token, tc.payload)
		if w.Code != 500 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	var count, version int
	var status string
	db.QueryRow(`SELECT COUNT(*),version,status FROM materials`).Scan(&count, &version, &status)
	if count != 1 || version != 1 || status != "ACTIVE" {
		t.Fatal("mutation survived audit failure", count, version, status)
	}
	db.Exec(`DROP TRIGGER fail_material_audit`)
	w := remediationRequest(s, "PATCH", "/api/v1/materials/1", token, map[string]any{"version": 1, "url": "https://example.com/changed"})
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var before, after string
	var class, sem, actor int
	if err := db.QueryRow(`SELECT class_id,semester_id,actor_user_id,json_extract(before_json,'$.url'),json_extract(after_json,'$.url') FROM audit_logs WHERE entity_type='MATERIAL'`).Scan(&class, &sem, &actor, &before, &after); err != nil {
		t.Fatal(err)
	}
	if class != 1 || sem != 1 || actor != 1 || before == after {
		t.Fatal(class, sem, actor, before, after)
	}
}
