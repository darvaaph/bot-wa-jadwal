package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/database"
	"bot-jadwal/internal/task"
)

func newV1TestServer(t *testing.T) (*Server, int64, int64, int64) {
	t.Helper()

	db, err := database.InitDB(filepath.Join(t.TempDir(), "api_v1.db"))
	if err != nil {
		t.Fatalf("gagal membuka database uji: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	var userID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('pj-v1', 'PJ V1', 'hash') RETURNING id`).Scan(&userID); err != nil {
		t.Fatalf("gagal membuat user: %v", err)
	}
	var classID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A', 'd4-ti-2024-a-v1', 'D4 TI', 2024, 'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatalf("gagal membuat kelas: %v", err)
	}
	now := "2026-09-24T10:00:00.000Z"
	var semID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31', 'ACTIVE', ?, ?) RETURNING id`, classID, now, now).Scan(&semID); err != nil {
		t.Fatalf("gagal membuat semester: %v", err)
	}
	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBDV1', 'Sistem Basis Data V1') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatalf("gagal membuat course: %v", err)
	}
	var offeringID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?, ?, 'TEORI', 'SBD V1 Teori') RETURNING id`, semID, courseID).Scan(&offeringID); err != nil {
		t.Fatalf("gagal membuat offering: %v", err)
	}
	var assignmentID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, status) VALUES (?, 'KM', 'CLASS', ?, 'ACTIVE') RETURNING id`, userID, classID).Scan(&assignmentID); err != nil {
		t.Fatalf("gagal membuat assignment: %v", err)
	}

	repo := task.NewRepository(db)
	srv := NewServer(":8080", nil, nil, nil)
	srv.SetTaskRepo(repo)
	return srv, classID, offeringID, assignmentID
}

func TestTasksV1_FullFlow(t *testing.T) {
	s, classID, offeringID, assignmentID := newV1TestServer(t)

	createBody, _ := json.Marshal(map[string]any{
		"course_offering_id": offeringID,
		"title":              "Tugas V1",
		"instructions":       "Kerjakan dengan benar.",
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
	if created.Data.ID <= 0 {
		t.Fatalf("create: ID tugas tidak valid")
	}
	taskID := created.Data.ID

	rr = performRequest(t, s, "GET", fmt.Sprintf("/api/v1/tasks?class_id=%d", classID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: diharapkan 200, didapat %d", rr.Code)
	}

	reviewBody, _ := json.Marshal(map[string]any{
		"decision":                    "APPROVED",
		"reviewer_role_assignment_id": assignmentID,
	})
	rr = performRequest(t, s, "POST", fmt.Sprintf("/api/v1/tasks/%d/reviews", taskID), reviewBody)
	if rr.Code != http.StatusOK {
		t.Fatalf("review: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "GET", fmt.Sprintf("/api/v1/tasks?class_id=%d&status=PUBLISHED", classID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list filtered: diharapkan 200, didapat %d", rr.Code)
	}
	var filtered struct {
		Status string              `json:"status"`
		Data   []task.TaskItemView `json:"data"`
	}
	decodeResponse(t, rr, &filtered)
	if len(filtered.Data) != 1 {
		t.Fatalf("filter PUBLISHED: diharapkan 1, didapat %d", len(filtered.Data))
	}

	rr = performRequest(t, s, "PATCH", fmt.Sprintf("/api/v1/tasks/%d/complete", taskID), nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("complete: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = performRequest(t, s, "GET", "/api/v1/tasks", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("list tanpa class_id: diharapkan 400, didapat %d", rr.Code)
	}

	badBody := bytes.NewBufferString(`{"title":"","deadline_at":""}`)
	reqBody := badBody.Bytes()
	rr = performRequest(t, s, "POST", "/api/v1/tasks", reqBody)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("create invalid: diharapkan 400, didapat %d", rr.Code)
	}
}
