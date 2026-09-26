package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"bot-jadwal/internal/task"
)

// TestAuditTrail_CoversMutations verifies Kritis-B: every covered mutation
// leaves an audit row readable via GET /api/v1/audit.
func TestAuditTrail_CoversMutations(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	auditCount := func(entityType string, entityID int64, action string) int {
		t.Helper()
		url := fmt.Sprintf("/api/v1/audit?entity_type=%s&entity_id=%d", entityType, entityID)
		if action != "" {
			url += "&action=" + action
		}
		rr := authHTTPRequest(t, srv, "GET", url, nil, adminCookies, "")
		if rr.Code != http.StatusOK {
			t.Fatalf("audit list %s/%d: diharapkan 200, didapat %d: %s", entityType, entityID, rr.Code, rr.Body.String())
		}
		var body struct {
			Data []map[string]any `json:"data"`
		}
		decodeResponse(t, rr, &body)
		return len(body.Data)
	}

	// 1. Task create -> CREATE.
	createBody, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Tugas Audit", "instructions": "Kerjakan.",
		"deadline_at": "2026-09-30T16:00:00.000Z", "task_type": "INDIVIDUAL", "submission_text": "LMS",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(createBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: %d %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data task.Task `json:"data"`
	}
	decodeResponse(t, rr, &created)
	taskID := created.Data.ID
	if n := auditCount("TASK", taskID, "CREATE"); n < 1 {
		t.Fatalf("audit CREATE/TASK hilang untuk task %d", taskID)
	}

	// 2. Publish (PJ) + review (KM APPROVED) -> PUBLISH + REVIEW_APPROVED.
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/publish", taskID), nil, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("TASK", taskID, "PUBLISH"); n < 1 {
		t.Fatalf("audit PUBLISH hilang untuk task %d", taskID)
	}
	reviewBody, _ := json.Marshal(map[string]any{"decision": "APPROVED"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/reviews", taskID), json.RawMessage(reviewBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("review: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("TASK", taskID, "REVIEW_APPROVED"); n < 1 {
		t.Fatalf("audit REVIEW_APPROVED hilang untuk task %d", taskID)
	}

	// 3. Complete -> COMPLETE.
	rr = authHTTPRequest(t, srv, "PATCH", fmt.Sprintf("/api/v1/tasks/%d/complete", taskID), nil, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("complete: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("TASK", taskID, "COMPLETE"); n < 1 {
		t.Fatalf("audit COMPLETE hilang untuk task %d", taskID)
	}

	// 4. Material create/update/delete -> CREATE/UPDATE/DELETE.
	matBody, _ := json.Marshal(map[string]any{
		"class_id": ids["classA"], "title": "Materi Audit", "material_type": "DOCUMENT",
		"url": "https://example.com/a.pdf", "visibility": "CLASS_ACCESS",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/materials", json.RawMessage(matBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create material: %d %s", rr.Code, rr.Body.String())
	}
	var matCreated struct {
		Data task.Material `json:"data"`
	}
	decodeResponse(t, rr, &matCreated)
	matID := matCreated.Data.ID
	if n := auditCount("MATERIAL", matID, "CREATE"); n < 1 {
		t.Fatalf("audit CREATE/MATERIAL hilang untuk %d", matID)
	}
	updBody, _ := json.Marshal(map[string]any{"title": "Materi Audit v2", "version": 1})
	rr = authHTTPRequest(t, srv, "PUT", fmt.Sprintf("/api/v1/materials/%d", matID), json.RawMessage(updBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("update material: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("MATERIAL", matID, "UPDATE"); n < 1 {
		t.Fatalf("audit UPDATE/MATERIAL hilang untuk %d", matID)
	}
	rr = authHTTPRequest(t, srv, "DELETE", fmt.Sprintf("/api/v1/materials/%d", matID), nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete material: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("MATERIAL", matID, "DELETE"); n < 1 {
		t.Fatalf("audit DELETE/MATERIAL hilang untuk %d", matID)
	}

	// 5. Semester draft + offering + activate.
	draftBody, _ := json.Marshal(map[string]any{
		"academic_year": "2027/2028", "term": "GANJIL", "starts_on": "2027-09-01", "ends_on": "2028-01-31",
	})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/classes/%d/semesters/draft", ids["classA"]), json.RawMessage(draftBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft: %d %s", rr.Code, rr.Body.String())
	}
	var draft struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &draft)
	if n := auditCount("SEMESTER", draft.Data.ID, "CREATE"); n < 1 {
		t.Fatalf("audit CREATE/SEMESTER hilang untuk %d", draft.Data.ID)
	}
	offBody, _ := json.Marshal(map[string]any{"course_code": "AUD", "course_name": "Audit", "activity_type": "TEORI"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/classes/%d/semesters/%d/offerings", ids["classA"], draft.Data.ID), json.RawMessage(offBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add offering: %d %s", rr.Code, rr.Body.String())
	}
	var offCreated struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &offCreated)
	if n := auditCount("COURSE_OFFERING", offCreated.Data.ID, "CREATE"); n < 1 {
		t.Fatalf("audit CREATE/COURSE_OFFERING hilang untuk %d", offCreated.Data.ID)
	}
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/classes/%d/semesters/%d/activate", ids["classA"], draft.Data.ID), nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("activate: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("SEMESTER", draft.Data.ID, "ACTIVATE"); n < 1 {
		t.Fatalf("audit ACTIVATE hilang untuk %d", draft.Data.ID)
	}

	// 6. Room create (admin) -> CREATE/ROOM.
	roomBody, _ := json.Marshal(map[string]any{"code": "RAUD", "name": "Ruang Audit"})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/admin/rooms", json.RawMessage(roomBody), adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create room: %d %s", rr.Code, rr.Body.String())
	}
	var roomCreated struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &roomCreated)
	if n := auditCount("ROOM", roomCreated.Data.ID, "CREATE"); n < 1 {
		t.Fatalf("audit CREATE/ROOM hilang untuk %d", roomCreated.Data.ID)
	}

	// 7. Invitation revoke -> REVOKE/ROLE_INVITATION.
	invBody, _ := json.Marshal(map[string]any{"role": "PJ", "class_id": ids["classA"], "semester_id": ids["semA"], "course_offering_id": ids["offA"], "identity_key": "pj-audit@example.test"})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations", json.RawMessage(invBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("invite: %d %s", rr.Code, rr.Body.String())
	}
	var invCreated struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &invCreated)
	if n := auditCount("ROLE_INVITATION", invCreated.Data.ID, "INVITE"); n < 1 {
		t.Fatalf("audit INVITE hilang untuk %d", invCreated.Data.ID)
	}
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/invitations/%d/revoke", invCreated.Data.ID), nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("ROLE_INVITATION", invCreated.Data.ID, "REVOKE"); n < 1 {
		t.Fatalf("audit REVOKE hilang untuk undangan %d", invCreated.Data.ID)
	}

	// 8. Class settings update -> UPDATE/CLASS_SETTING.
	rr = authHTTPRequest(t, srv, "GET", fmt.Sprintf("/api/v1/classes/%d/settings", ids["classA"]), nil, kmCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("get settings: %d %s", rr.Code, rr.Body.String())
	}
	var settings struct {
		Data struct {
			Version int `json:"version"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &settings)
	setBody, _ := json.Marshal(map[string]any{"afternoon_reminder_time": "16:30", "version": settings.Data.Version})
	rr = authHTTPRequest(t, srv, "PATCH", fmt.Sprintf("/api/v1/classes/%d/settings", ids["classA"]), json.RawMessage(setBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings: %d %s", rr.Code, rr.Body.String())
	}
	if n := auditCount("CLASS_SETTING", ids["classA"], "UPDATE"); n < 1 {
		t.Fatalf("audit UPDATE/CLASS_SETTING hilang untuk kelas %d", ids["classA"])
	}
}
