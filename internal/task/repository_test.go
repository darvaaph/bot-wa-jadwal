package task

import (
	"context"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/database"
)

type taskSeedCtx struct {
	repo           *Repository
	pjUserID       int64
	kmAssignmentID int64
	classAID       int64
	classBID       int64
	offeringAID    int64
	offeringBID    int64
}

func newTaskSeedCtx(t *testing.T) *taskSeedCtx {
	t.Helper()

	db, err := database.InitDB(filepath.Join(t.TempDir(), "task_repo.db"))
	if err != nil {
		t.Fatalf("gagal membuka database uji: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()

	var pjID, kmID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('pj-test', 'PJ Test', 'hash') RETURNING id`).Scan(&pjID); err != nil {
		t.Fatalf("gagal membuat user PJ: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash) VALUES ('km-test', 'KM Test', 'hash') RETURNING id`).Scan(&kmID); err != nil {
		t.Fatalf("gagal membuat user KM: %v", err)
	}

	var classAID, classBID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A', 'd4-ti-2024-a', 'D4 TI', 2024, 'A') RETURNING id`).Scan(&classAID); err != nil {
		t.Fatalf("gagal membuat kelas A: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-B', 'd4-ti-2024-b', 'D4 TI', 2024, 'B') RETURNING id`).Scan(&classBID); err != nil {
		t.Fatalf("gagal membuat kelas B: %v", err)
	}

	now := "2026-09-24T10:00:00.000Z"
	var semAID, semBID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31', 'ACTIVE', ?, ?) RETURNING id`, classAID, now, now).Scan(&semAID); err != nil {
		t.Fatalf("gagal membuat semester A: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027', 'GANJIL', '2026-09-01', '2027-01-31', 'ACTIVE', ?, ?) RETURNING id`, classBID, now, now).Scan(&semBID); err != nil {
		t.Fatalf("gagal membuat semester B: %v", err)
	}

	var courseID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO courses (code, name) VALUES ('SBD', 'Sistem Basis Data') RETURNING id`).Scan(&courseID); err != nil {
		t.Fatalf("gagal membuat course: %v", err)
	}

	var offeringAID, offeringBID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?, ?, 'TEORI', 'Sistem Basis Data Teori') RETURNING id`, semAID, courseID).Scan(&offeringAID); err != nil {
		t.Fatalf("gagal membuat offering A: %v", err)
	}
	if err := db.QueryRowContext(ctx, `INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name) VALUES (?, ?, 'TEORI', 'Sistem Basis Data Teori') RETURNING id`, semBID, courseID).Scan(&offeringBID); err != nil {
		t.Fatalf("gagal membuat offering B: %v", err)
	}

	var kmAssignmentID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO role_assignments (user_id, role, scope_type, class_id, status) VALUES (?, 'KM', 'CLASS', ?, 'ACTIVE') RETURNING id`, kmID, classAID).Scan(&kmAssignmentID); err != nil {
		t.Fatalf("gagal membuat role assignment KM: %v", err)
	}

	return &taskSeedCtx{
		repo:           NewRepository(db),
		pjUserID:       pjID,
		kmAssignmentID: kmAssignmentID,
		classAID:       classAID,
		classBID:       classBID,
		offeringAID:    offeringAID,
		offeringBID:    offeringBID,
	}
}

func strValue(s string) *string {
	v := s
	return &v
}

func TestRepository_CreateTask(t *testing.T) {
	seed := newTaskSeedCtx(t)
	ctx := context.Background()

	submission := "Kumpulkan melalui LMS kelas"
	created, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Normalisasi Skema Perpustakaan",
		Instructions:     "Ubah tabel transaksi ke bentuk normal ketiga.",
		DeadlineAt:       "2026-09-30T16:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask gagal: %v", err)
	}
	if created.PublicationStatus != "DRAFT" {
		t.Errorf("publication_status = %q, diharapkan DRAFT", created.PublicationStatus)
	}
	if created.ReviewState != "NOT_REVIEWED" {
		t.Errorf("review_state = %q, diharapkan NOT_REVIEWED", created.ReviewState)
	}
	if created.Version != 1 {
		t.Errorf("version = %d, diharapkan 1", created.Version)
	}

	view, err := seed.repo.GetTaskByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTaskByID gagal: %v", err)
	}
	if view.CourseCode != "SBD" {
		t.Errorf("course_code = %q, diharapkan SBD", view.CourseCode)
	}
	if view.CourseName != "Sistem Basis Data" {
		t.Errorf("course_name = %q, diharapkan Sistem Basis Data", view.CourseName)
	}
	if view.ClassCode != "D4-TI-2024-A" {
		t.Errorf("class_code = %q, diharapkan D4-TI-2024-A", view.ClassCode)
	}
}

func TestRepository_SubmitReview_Approved(t *testing.T) {
	seed := newTaskSeedCtx(t)
	ctx := context.Background()

	submission := "Kumpulkan melalui LMS kelas"
	created, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Refaktor Modul Inventaris",
		Instructions:     "Rapikan struktur modul inventaris.",
		DeadlineAt:       "2026-09-28T13:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask gagal: %v", err)
	}

	if err := seed.repo.SubmitReview(ctx, ReviewTaskInput{
		TaskID:                   created.ID,
		Decision:                 "APPROVED",
		ReviewerRoleAssignmentID: seed.kmAssignmentID,
	}); err != nil {
		t.Fatalf("SubmitReview APPROVED gagal: %v", err)
	}

	view, err := seed.repo.GetTaskByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTaskByID gagal: %v", err)
	}
	if view.PublicationStatus != "PUBLISHED" {
		t.Errorf("publication_status = %q, diharapkan PUBLISHED", view.PublicationStatus)
	}
	if view.ReviewState != "APPROVED" {
		t.Errorf("review_state = %q, diharapkan APPROVED", view.ReviewState)
	}
	if view.PublishedAt == nil || *view.PublishedAt == "" {
		t.Errorf("published_at seharusnya terisi setelah APPROVED")
	}
}

func TestRepository_SubmitReview_ChangesRequested(t *testing.T) {
	seed := newTaskSeedCtx(t)
	ctx := context.Background()

	submission := "Kumpulkan melalui LMS kelas"
	created, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Latihan Ruang Vektor",
		Instructions:     "Kerjakan latihan ruang vektor.",
		DeadlineAt:       "2026-10-01T11:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask gagal: %v", err)
	}

	if err := seed.repo.SubmitReview(ctx, ReviewTaskInput{
		TaskID:                   created.ID,
		Decision:                 "CHANGES_REQUESTED",
		Note:                     strValue("Tambahkan contoh perhitungan."),
		ReviewerRoleAssignmentID: seed.kmAssignmentID,
	}); err != nil {
		t.Fatalf("SubmitReview CHANGES_REQUESTED gagal: %v", err)
	}

	view, err := seed.repo.GetTaskByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTaskByID gagal: %v", err)
	}
	if view.PublicationStatus != "DRAFT" {
		t.Errorf("publication_status = %q, diharapkan DRAFT", view.PublicationStatus)
	}
	if view.ReviewState != "CHANGES_REQUESTED" {
		t.Errorf("review_state = %q, diharapkan CHANGES_REQUESTED", view.ReviewState)
	}
}

func TestRepository_ListTasksByClass(t *testing.T) {
	seed := newTaskSeedCtx(t)
	ctx := context.Background()

	submission := "Kumpulkan melalui LMS kelas"
	first, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Tugas Tenggat Dekat",
		Instructions:     "Instruksi tugas pertama.",
		DeadlineAt:       "2026-09-25T10:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask pertama gagal: %v", err)
	}
	second, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Tugas Tenggat Jauh",
		Instructions:     "Instruksi tugas kedua.",
		DeadlineAt:       "2026-10-05T10:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask kedua gagal: %v", err)
	}
	_, err = seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringBID,
		Title:            "Tugas Kelas Lain",
		Instructions:     "Instruksi kelas B.",
		DeadlineAt:       "2026-09-26T10:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask kelas B gagal: %v", err)
	}

	if err := seed.repo.SubmitReview(ctx, ReviewTaskInput{
		TaskID:                   first.ID,
		Decision:                 "APPROVED",
		ReviewerRoleAssignmentID: seed.kmAssignmentID,
	}); err != nil {
		t.Fatalf("SubmitReview gagal: %v", err)
	}

	all, err := seed.repo.ListTasksByClass(ctx, seed.classAID, "")
	if err != nil {
		t.Fatalf("ListTasksByClass gagal: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("diharapkan 2 tugas kelas A, didapat %d", len(all))
	}
	if all[0].ID != first.ID || all[1].ID != second.ID {
		t.Errorf("urutan deadline tidak sesuai: didapat %d, %d", all[0].ID, all[1].ID)
	}

	published, err := seed.repo.ListTasksByClass(ctx, seed.classAID, "PUBLISHED")
	if err != nil {
		t.Fatalf("ListTasksByClass dengan filter gagal: %v", err)
	}
	if len(published) != 1 || published[0].ID != first.ID {
		t.Errorf("filter PUBLISHED seharusnya hanya mengembalikan tugas pertama")
	}
}

func TestRepository_DeleteTask(t *testing.T) {
	seed := newTaskSeedCtx(t)
	ctx := context.Background()

	submission := "Kumpulkan melalui LMS kelas"
	created, err := seed.repo.CreateTask(ctx, CreateTaskInput{
		CourseOfferingID: seed.offeringAID,
		Title:            "Tugas Dihapus",
		Instructions:     "Instruksi tugas hapus.",
		DeadlineAt:       "2026-09-27T10:00:00.000Z",
		TaskType:         "INDIVIDUAL",
		SubmissionText:   &submission,
		CreatedByUserID:  seed.pjUserID,
	})
	if err != nil {
		t.Fatalf("CreateTask gagal: %v", err)
	}

	if err := seed.repo.DeleteTask(ctx, created.ID, seed.pjUserID); err != nil {
		t.Fatalf("DeleteTask gagal: %v", err)
	}

	items, err := seed.repo.ListTasksByClass(ctx, seed.classAID, "")
	if err != nil {
		t.Fatalf("ListTasksByClass gagal: %v", err)
	}
	for _, item := range items {
		if item.ID == created.ID {
			t.Fatalf("tugas yang dihapus masih muncul pada daftar aktif")
		}
	}
}
