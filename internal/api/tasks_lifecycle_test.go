package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"bot-jadwal/internal/task"
)

func TestTasksLifecycle_PublishUpdateArchiveRestore(t *testing.T) {
	s, _, offeringID, _ := newV1TestServer(t)

	createBody, _ := json.Marshal(map[string]any{
		"course_offering_id": offeringID,
		"title":              "Tugas Lifecycle",
		"instructions":       "Kerjakan.",
		"deadline_at":        "2026-09-30T16:00:00.000Z",
		"task_type":          "INDIVIDUAL",
		"submission_text":    "Via LMS",
		"created_by_user_id": 1,
	})
	rr := performRequest(t, s, "POST", "/api/v1/tasks", createBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Status string    `json:"status"`
		Data   task.Task `json:"data"`
	}
	decodeResponse(t, rr, &created)
	taskID := created.Data.ID
	if created.Data.PublicationStatus != "DRAFT" {
		t.Fatalf("create: diharapkan DRAFT, didapat %s", created.Data.PublicationStatus)
	}

	rr = performRequest(t, s, "POST", fmt.Sprintf("/api/v1/tasks/%d/publish", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("publish: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var published struct {
		Status string    `json:"status"`
		Data   task.Task `json:"data"`
	}
	decodeResponse(t, rr, &published)
	if published.Data.PublicationStatus != "PUBLISHED" {
		t.Fatalf("publish: diharapkan PUBLISHED, didapat %s", published.Data.PublicationStatus)
	}

	rr = performRequest(t, s, "GET", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("detail: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	updateBody, _ := json.Marshal(map[string]any{
		"title":   "Tugas Lifecycle v2",
		"version": published.Data.Version,
	})
	rr = performRequest(t, s, "PUT", fmt.Sprintf("/api/v1/tasks/%d", taskID), updateBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("update: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var updated struct {
		Status string    `json:"status"`
		Data   task.Task `json:"data"`
	}
	decodeResponse(t, rr, &updated)
	if updated.Data.Version != published.Data.Version+1 {
		t.Fatalf("update: version diharapkan %d, didapat %d", published.Data.Version+1, updated.Data.Version)
	}

	staleBody, _ := json.Marshal(map[string]any{
		"title":   "Stale Update",
		"version": published.Data.Version,
	})
	rr = performRequest(t, s, "PUT", fmt.Sprintf("/api/v1/tasks/%d", taskID), staleBody)
	if rr.Code != http.StatusConflict {
		t.Fatalf("stale update: diharapkan 409, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "POST", fmt.Sprintf("/api/v1/tasks/%d/archive", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("archive: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "POST", fmt.Sprintf("/api/v1/tasks/%d/unarchive", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("unarchive: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "DELETE", fmt.Sprintf("/api/v1/tasks/%d", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	restoreBody, _ := json.Marshal(map[string]any{"reason": "salah hapus"})
	rr = performRequest(t, s, "POST", fmt.Sprintf("/api/v1/tasks/%d/restore", taskID), restoreBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("restore: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "GET", fmt.Sprintf("/api/v1/tasks/%d/reviews", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("reviews: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
}

func TestMaterials_CRUD(t *testing.T) {
	s, classID, offeringID, _ := newV1TestServer(t)

	createBody, _ := json.Marshal(map[string]any{
		"class_id":           classID,
		"course_offering_id": offeringID,
		"title":              "Slide SBD",
		"material_type":      "DOCUMENT",
		"url":                "https://example.com/slide.pdf",
		"visibility":         "CLASS_ACCESS",
	})
	rr := performRequest(t, s, "POST", "/api/v1/materials", createBody)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create material: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var created struct {
		Status string        `json:"status"`
		Data   task.Material `json:"data"`
	}
	decodeResponse(t, rr, &created)
	if created.Data.ID <= 0 {
		t.Fatalf("create material: ID tidak valid")
	}

	rr = performRequest(t, s, "GET", fmt.Sprintf("/api/v1/materials?class_id=%d", classID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list materials: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	badBody, _ := json.Marshal(map[string]any{
		"class_id":      classID,
		"title":         "Bad",
		"material_type": "DOCUMENT",
		"url":           "ftp://invalid",
	})
	rr = performRequest(t, s, "POST", "/api/v1/materials", badBody)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad url: diharapkan 400, didapat %d: %s", rr.Code, rr.Body.String())
	}
}
