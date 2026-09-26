package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/notify"
	"bot-jadwal/internal/task"
)

// TestAuthz_MaterialsCrossClass verifies K-1/K-2: pre-write scope checks on
// material update/delete, including the PJ offering rule.
func TestAuthz_MaterialsCrossClass(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")
	classB := ids["classB"]
	classA := ids["classA"]

	// Admin seeds: one material in class B, one general material in class A,
	// one material on PJ-A's own offering.
	seed := func(classID int64, offering *int64, title string) int64 {
		t.Helper()
		body := map[string]any{
			"class_id": classID, "title": title, "material_type": "DOCUMENT",
			"url": "https://example.com/m.pdf", "visibility": "CLASS_ACCESS",
		}
		if offering != nil {
			body["course_offering_id"] = *offering
		}
		payload, _ := json.Marshal(body)
		rr := authHTTPRequest(t, srv, "POST", "/api/v1/materials", json.RawMessage(payload), adminCookies, adminCSRF)
		if rr.Code != http.StatusCreated {
			t.Fatalf("seed material %q: diharapkan 201, didapat %d: %s", title, rr.Code, rr.Body.String())
		}
		var created struct {
			Data task.Material `json:"data"`
		}
		decodeResponse(t, rr, &created)
		return created.Data.ID
	}
	matB := seed(classB, nil, "Materi Kelas B")
	matGeneralA := seed(classA, nil, "Materi Umum A")
	offA := ids["offA"]
	matOwnA := seed(classA, &offA, "Materi Offering PJ")

	put := func(cookies []*http.Cookie, csrf string, id int64, title string) *http.Response {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"title": title, "version": 1})
		rr := authHTTPRequest(t, srv, "PUT", fmt.Sprintf("/api/v1/materials/%d", id), json.RawMessage(payload), cookies, csrf)
		return rr.Result()
	}

	// KM-A (class A) must not touch class B material.
	if rr := put(kmCookies, kmCSRF, matB, "hacked"); rr.StatusCode != http.StatusForbidden {
		t.Fatalf("KM-A update materi kelas B: diharapkan 403, didapat %d", rr.StatusCode)
	}
	rr := authHTTPRequest(t, srv, "DELETE", fmt.Sprintf("/api/v1/materials/%d", matB), nil, kmCookies, kmCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("KM-A delete materi kelas B: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// PJ-A must not touch general (non-offering) material even in own class.
	if rr := put(pjCookies, pjCSRF, matGeneralA, "hacked"); rr.StatusCode != http.StatusForbidden {
		t.Fatalf("PJ-A update materi umum: diharapkan 403, didapat %d", rr.StatusCode)
	}
	// Positive control: PJ-A can update own offering material.
	if rr := put(pjCookies, pjCSRF, matOwnA, "Materi Offering PJ v2"); rr.StatusCode != http.StatusOK {
		t.Fatalf("PJ-A update materi sendiri: diharapkan 200, didapat %d", rr.StatusCode)
	}
	// Class B material must be byte-identical (pre-write check, no mutation).
	rr = authHTTPRequest(t, srv, "GET", fmt.Sprintf("/api/v1/materials?class_id=%d", classB), nil, adminCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list materi B: diharapkan 200, didapat %d", rr.Code)
	}
	var list struct {
		Data []task.Material `json:"data"`
	}
	decodeResponse(t, rr, &list)
	if len(list.Data) != 1 || list.Data[0].Title != "Materi Kelas B" || list.Data[0].Version != 1 {
		t.Fatalf("materi kelas B berubah setelah 403: %+v", list.Data)
	}
}

// TestAuthz_EventRoomConfirmAndPreview verifies K-3/K-4: room confirmation
// requires owner scope; preview follows detail visibility.
func TestAuthz_EventRoomConfirmAndPreview(t *testing.T) {
	srv, ids := newEventsTestServer(t)
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")
	kmBCookies, kmBCSRF, _ := loginAs(t, srv, "km-b@example.test", "kata-sandi-km-b-kuat")

	draftBody, _ := json.Marshal(map[string]any{
		"owner_offering_id": ids["offA"], "event_kind": "EXTRA",
		"starts_at": "2026-11-06T02:00:00Z", "ends_at": "2026-11-06T03:40:00Z",
		"reason": "Tambahan",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/teaching-events/draft", json.RawMessage(draftBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &created)
	path := fmt.Sprintf("/api/v1/teaching-events/%d", created.Data.ID)

	// K-4: outsider class sees neither preview nor detail of a DRAFT.
	rr = authHTTPRequest(t, srv, "GET", path+"/preview", nil, kmBCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("preview lintas kelas: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", path, nil, kmBCookies, "")
	if rr.Code != http.StatusForbidden {
		t.Fatalf("detail lintas kelas: diharapkan 403, didapat %d", rr.Code)
	}
	rr = authHTTPRequest(t, srv, "GET", path+"/preview", nil, pjCookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("preview pemilik: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// K-3: outsider class cannot record confirmations.
	confirmBody, _ := json.Marshal(map[string]any{
		"room_id": ids["room"], "status": "CONFIRMED", "external_contact": "TU", "note": "ok",
	})
	rr = authHTTPRequest(t, srv, "POST", path+"/room-confirmation", json.RawMessage(confirmBody), kmBCookies, kmBCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("room-confirm lintas kelas: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// No confirmation row must exist after the rejected attempt.
	rr = authHTTPRequest(t, srv, "GET", path, nil, pjCookies, "")
	var detail struct {
		Data struct {
			Confirmations []map[string]any `json:"confirmations"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &detail)
	if len(detail.Data.Confirmations) != 0 {
		t.Fatalf("konfirmasi bocor setelah 403: %+v", detail.Data.Confirmations)
	}
	// Positive control: owner records successfully.
	rr = authHTTPRequest(t, srv, "POST", path+"/room-confirmation", json.RawMessage(confirmBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("room-confirm pemilik: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAuthz_RestoreCrossClass verifies K-5: restore resolves the true scope
// of the soft-deleted task instead of trusting the caller's class.
func TestAuthz_RestoreCrossClass(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	adminCookies, adminCSRF, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	// Invite + accept a KM for class B via API (no DB handle needed).
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/invitations", map[string]any{
		"role": "KM", "class_id": ids["classB"], "identity_key": "km-b@example.test",
	}, adminCookies, adminCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("invite KM-B: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var inv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	decodeResponse(t, rr, &inv)
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/invitations/"+inv.Data.Token+"/accept", map[string]any{
		"display_name": "KM B", "password": "kata-sandi-km-b-kuat",
	}, nil, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("accept KM-B: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	kmBCookies, kmBCSRF, _ := loginAs(t, srv, "km-b@example.test", "kata-sandi-km-b-kuat")

	// PJ-A creates and deletes a task in class A.
	createBody, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Tugas A", "instructions": "Kerjakan.",
		"deadline_at": "2026-09-30T16:00:00.000Z", "task_type": "INDIVIDUAL", "submission_text": "LMS",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(createBody), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Data task.Task `json:"data"`
	}
	decodeResponse(t, rr, &created)
	restorePath := fmt.Sprintf("/api/v1/tasks/%d/restore", created.Data.ID)
	restoreBody, _ := json.Marshal(map[string]any{"reason": "salah hapus"})
	rr = authHTTPRequest(t, srv, "DELETE", fmt.Sprintf("/api/v1/tasks/%d", created.Data.ID), nil, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// KM-B (other class) must be rejected and the task must stay deleted.
	rr = authHTTPRequest(t, srv, "POST", restorePath, json.RawMessage(restoreBody), kmBCookies, kmBCSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("restore lintas kelas: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	// Positive control: KM-A restores successfully, proving the row was untouched.
	rr = authHTTPRequest(t, srv, "POST", restorePath, json.RawMessage(restoreBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore sekelas: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAuthz_RetryCrossClass verifies S-1: KM can only retry their own class queue.
func TestAuthz_RetryCrossClass(t *testing.T) {
	srv, ids := newRetryScopeServer(t)
	kmACookies, kmACSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	kmBCookies, kmBCSRF, _ := loginAs(t, srv, "km-b@example.test", "kata-sandi-km-b-kuat")

	// KM-A retries class B's failed message: forbidden, message stays FAILED.
	rr := authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/notifications/%d/retry", ids["msgB"]), nil, kmACookies, kmACSRF)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("retry lintas kelas: diharapkan 403, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", fmt.Sprintf("/api/v1/notifications?class_id=%d&status=FAILED", ids["classB"]), nil, kmBCookies, "")
	var failed struct {
		Data []map[string]any `json:"data"`
	}
	decodeResponse(t, rr, &failed)
	if len(failed.Data) != 1 {
		t.Fatalf("pesan kelas B harus tetap FAILED setelah 403, didapat %+v", failed.Data)
	}
	// Positive control: KM-B retries own message.
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/notifications/%d/retry", ids["msgB"]), nil, kmBCookies, kmBCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("retry sekelas: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func newRetryScopeServer(t *testing.T) (*Server, map[string]int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "retry-scope.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	svc, err := auth.NewService(db, auth.Config{HashKey: []byte("0123456789abcdef0123456789abcdef")})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.ProvisionInitialSystemAdmin(ctx, auth.ProvisionInput{
		IdentityKey: "admin@example.test", DisplayName: "Admin", Password: "kata-sandi-yang-sangat-kuat",
	}); err != nil {
		t.Fatal(err)
	}
	ids := map[string]int64{}
	mkClass := func(code, slug, kmIdentity, kmName, kmPassword string) (classID, msgID int64) {
		t.Helper()
		if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES (?, ?, 'D4 TI', 2024, 'A') RETURNING id`, code, slug).Scan(&classID); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, classID); err != nil {
			t.Fatal(err)
		}
		var chID int64
		if err := db.QueryRowContext(ctx, `INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status) VALUES (?, ?, 'GROUP', 'Grup', 'ACTIVE') RETURNING id`, classID, "120363-"+slug+"@g.us").Scan(&chID); err != nil {
			t.Fatal(err)
		}
		hash, err := auth.HashPassword(kmIdentity, kmPassword)
		if err != nil {
			t.Fatal(err)
		}
		var uid int64
		if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES (?,?,?) RETURNING id`, kmIdentity, kmName, hash).Scan(&uid); err != nil {
			t.Fatal(err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, status, valid_from) VALUES (?, 'KM','CLASS',?,'ACTIVE','2026-01-01T00:00:00Z')`, uid, classID); err != nil {
			t.Fatal(err)
		}
		past := "2026-09-24T10:00:00.000Z"
		if err := db.QueryRowContext(ctx, `INSERT INTO notification_messages (class_id, whatsapp_channel_id, event_type, entity_type, entity_id, idempotency_key, payload_json, status, scheduled_at) VALUES (?, ?, 'DAILY_SUMMARY','CLASS',?, ?, '{"text":"x"}','FAILED', ?) RETURNING id`,
			classID, chID, classID, "k-"+slug, past).Scan(&msgID); err != nil {
			t.Fatal(err)
		}
		_ = chID
		return classID, msgID
	}
	classA, _ := mkClass("D4-TI-2024-A", "a-scope", "km-a@example.test", "KM A", "kata-sandi-km-a-kuat")
	classB, msgB := mkClass("D4-TI-2024-B", "b-scope", "km-b@example.test", "KM B", "kata-sandi-km-b-kuat")
	ids["classA"], ids["classB"], ids["msgB"] = classA, classB, msgB

	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetNotifyService(notify.NewService(db), &stubSender{})
	return srv, ids
}

// TestAuthz_ReviewStateGuards verifies S-8: APPROVED requires complete task
// data; REVOKED tasks cannot be re-reviewed without re-publish.
func TestAuthz_ReviewStateGuards(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")
	pjCookies, pjCSRF, _ := loginAs(t, srv, "pj-a@example.test", "kata-sandi-pj-a-kuat")

	// Incomplete DRAFT (no instructions, no submission).
	raw, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Draf Kosong",
		"deadline_at": "2026-09-30T16:00:00.000Z",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(raw), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create draft: %d %s", rr.Code, rr.Body.String())
	}
	var draft struct {
		Data task.Task `json:"data"`
	}
	decodeResponse(t, rr, &draft)

	approve, _ := json.Marshal(map[string]any{"decision": "APPROVED"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/reviews", draft.Data.ID), json.RawMessage(approve), kmCookies, kmCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("APPROVED atas draf tak lengkap: diharapkan 400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// Complete task -> publish -> KM REVOKED -> CHANGES must conflict.
	full, _ := json.Marshal(map[string]any{
		"course_offering_id": ids["offA"], "title": "Tugas Penuh", "instructions": "Kerjakan.",
		"deadline_at": "2026-09-30T16:00:00.000Z", "submission_text": "LMS",
	})
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/tasks", json.RawMessage(full), pjCookies, pjCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create full: %d", rr.Code)
	}
	var created struct {
		Data task.Task `json:"data"`
	}
	decodeResponse(t, rr, &created)
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/publish", created.Data.ID), nil, pjCookies, pjCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", rr.Code, rr.Body.String())
	}
	revoke, _ := json.Marshal(map[string]any{"decision": "REVOKED", "note": "batal"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/reviews", created.Data.ID), json.RawMessage(revoke), kmCookies, kmCSRF)
	if rr.Code != http.StatusOK {
		t.Fatalf("revoke review: %d %s", rr.Code, rr.Body.String())
	}
	changes, _ := json.Marshal(map[string]any{"decision": "CHANGES_REQUESTED", "note": "perbaiki"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/tasks/%d/reviews", created.Data.ID), json.RawMessage(changes), kmCookies, kmCSRF)
	if rr.Code != http.StatusConflict {
		t.Fatalf("CHANGES atas REVOKED: diharapkan 409, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

// TestAuthz_VersionGuards verifies S-4: materials update and semester
// activate reject stale/missing versions.
func TestAuthz_VersionGuards(t *testing.T) {
	srv, ids := newOpsTestServer(t)
	kmCookies, kmCSRF, _ := loginAs(t, srv, "km-a@example.test", "kata-sandi-km-a-kuat")

	// Materials update without version.
	matBody, _ := json.Marshal(map[string]any{
		"class_id": ids["classA"], "title": "M", "material_type": "DOCUMENT",
		"url": "https://example.com/m.pdf",
	})
	rr := authHTTPRequest(t, srv, "POST", "/api/v1/materials", json.RawMessage(matBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create material: %d %s", rr.Code, rr.Body.String())
	}
	var mat struct {
		Data task.Material `json:"data"`
	}
	decodeResponse(t, rr, &mat)
	noVer, _ := json.Marshal(map[string]any{"title": "M v2"})
	rr = authHTTPRequest(t, srv, "PUT", fmt.Sprintf("/api/v1/materials/%d", mat.Data.ID), json.RawMessage(noVer), kmCookies, kmCSRF)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("update tanpa version: diharapkan 400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	// Semester activate with bogus version.
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
	offBody, _ := json.Marshal(map[string]any{"course_code": "VG", "course_name": "VerGuard"})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/classes/%d/semesters/%d/offerings", ids["classA"], draft.Data.ID), json.RawMessage(offBody), kmCookies, kmCSRF)
	if rr.Code != http.StatusCreated {
		t.Fatalf("offering: %d %s", rr.Code, rr.Body.String())
	}
	badVer, _ := json.Marshal(map[string]any{"version": 999})
	rr = authHTTPRequest(t, srv, "POST", fmt.Sprintf("/api/v1/classes/%d/semesters/%d/activate", ids["classA"], draft.Data.ID), json.RawMessage(badVer), kmCookies, kmCSRF)
	if rr.Code != http.StatusConflict {
		t.Fatalf("activate versi basi: diharapkan 409, didapat %d: %s", rr.Code, rr.Body.String())
	}
}
