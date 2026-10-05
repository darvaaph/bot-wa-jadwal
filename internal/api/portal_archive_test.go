package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestV1Portal_ArchivedSemesterReadEndpoints(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at)
		VALUES (2, 1, '2023/2024', 'GANJIL', '2023-09-01', '2024-01-31', 'ARCHIVED', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       (3, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT', NULL, NULL, NULL),
		       (4, 1, '2022/2023', 'GANJIL', '2022-09-01', '2023-01-31', 'ARCHIVED', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

		INSERT INTO courses (id, code, name)
		VALUES (2, 'TI101', 'Algoritma Lama'), (3, 'TI301', 'Semester Draf');
		INSERT INTO course_offerings (id, semester_id, course_id, display_name, activity_type)
		VALUES (2, 2, 2, 'Algoritma Lama (Teori)', 'TEORI'),
		       (3, 3, 3, 'Semester Draf (Teori)', 'TEORI');

		INSERT INTO schedule_patterns (id, course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
		VALUES (2, 2, 1, 1, '10:00', '11:40', '2023-09-01', 'ACTIVE'),
		       (3, 3, 1, 1, '13:00', '14:40', '2025-09-01', 'ACTIVE');

		INSERT INTO teaching_events (id, event_kind, starts_at, ends_at, room_id, reason, lifecycle_status, published_at)
		VALUES (2, 'EXTRA', '2023-09-04T03:00:00Z', '2023-09-04T04:40:00Z', 1, 'Tambahan arsip', 'PUBLISHED', '2023-09-01T00:00:00Z'),
		       (3, 'EXTRA', '2023-09-04T05:00:00Z', '2023-09-04T06:40:00Z', 1, 'Belum terbit', 'DRAFT', NULL);
		INSERT INTO teaching_event_offerings (teaching_event_id, course_offering_id, participation_role, participation_status)
		VALUES (2, 2, 'OWNER', 'ACCEPTED'), (3, 2, 'OWNER', 'ACCEPTED');

		INSERT INTO tasks (id, course_offering_id, title, instructions, deadline_at, publication_status, published_at, created_by_user_id, submission_text)
		VALUES (2, 2, 'Tugas Arsip Terbit', 'Instruksi arsip', '2023-10-01T00:00:00Z', 'PUBLISHED', '2023-09-01T00:00:00Z', 1, 'Kumpulkan di kelas'),
		       (3, 2, 'Tugas Arsip Draf', 'Rahasia', '2023-10-02T00:00:00Z', 'DRAFT', NULL, 1, NULL);

		INSERT INTO materials (id, class_id, course_offering_id, task_id, title, material_type, url, visibility, status, created_by_user_id)
		VALUES (2, 1, 2, NULL, 'Materi Arsip', 'DOCUMENT', 'https://example.com/arsip', 'CLASS_ACCESS', 'ACTIVE', 1),
		       (3, 1, 2, NULL, 'Materi WhatsApp', 'DOCUMENT', 'https://example.com/wa', 'WHATSAPP_ONLY', 'ACTIVE', 1),
		       (4, 1, NULL, NULL, 'Materi Umum', 'DOCUMENT', 'https://example.com/umum', 'CLASS_ACCESS', 'ACTIVE', 1),
		       (5, 1, 2, 3, 'Materi Tugas Draf', 'DOCUMENT', 'https://example.com/draf', 'CLASS_ACCESS', 'ACTIVE', 1);
	`)
	if err != nil {
		t.Fatalf("gagal menyiapkan semester arsip: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		wantValue string
		wantKey   string
		notValues []string
	}{
		{
			name:    "schedule",
			path:    "/api/v1/portal/d4-ti-2024-a/schedule?semester_id=2&date=2023-09-04",
			wantKey: "offering", wantValue: "Algoritma Lama (Teori)",
			notValues: []string{"Struktur Data (Teori)", "Semester Draf (Teori)"},
		},
		{
			name: "tasks", path: "/api/v1/portal/d4-ti-2024-a/tasks?semester_id=2",
			wantKey: "title", wantValue: "Tugas Arsip Terbit", notValues: []string{"Tugas Arsip Draf", "Tugas Algoritma 1"},
		},
		{
			name: "changes", path: "/api/v1/portal/d4-ti-2024-a/changes?semester_id=2",
			wantKey: "reason", wantValue: "Tambahan arsip", notValues: []string{"Belum terbit"},
		},
		{
			name: "materials", path: "/api/v1/portal/d4-ti-2024-a/materials?semester_id=2",
			wantKey: "title", wantValue: "Materi Arsip", notValues: []string{"Slide Pertemuan 1", "Materi WhatsApp", "Materi Tugas Draf"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := portalResponseData(t, s, tt.path)
			if !containsPortalValue(data, tt.wantKey, tt.wantValue) {
				t.Fatalf("%s tidak memuat %q pada %s: %#v", tt.name, tt.wantValue, tt.wantKey, data)
			}
			for _, unwanted := range tt.notValues {
				if containsPortalValue(data, tt.wantKey, unwanted) {
					t.Errorf("%s membocorkan %q: %#v", tt.name, unwanted, data)
				}
			}
		})
	}

	materials := portalResponseData(t, s, "/api/v1/portal/d4-ti-2024-a/materials?semester_id=2")
	if !containsPortalValue(materials, "title", "Materi Umum") {
		t.Fatalf("materi umum kelas harus tersedia pada arsip: %#v", materials)
	}

	detail := portalResponseData(t, s, "/api/v1/portal/d4-ti-2024-a/tasks/2?semester_id=2")
	detailMap := detail.(map[string]any)
	task, ok := detailMap["task"].(map[string]any)
	if !ok || task["title"] != "Tugas Arsip Terbit" {
		t.Fatalf("detail tugas arsip tidak tepat: %#v", detail)
	}
	if containsPortalValue(detailMap["materials"], "title", "Materi Tugas Draf") ||
		containsPortalValue(detailMap["materials"], "title", "Materi WhatsApp") {
		t.Fatalf("detail tugas membocorkan materi tidak terbit: %#v", detailMap["materials"])
	}
}

func TestV1Portal_SemesterSelectionValidation(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO classes (id, code, slug, study_program, cohort_year, group_label, status)
		VALUES (2, 'D4-TI-2024-B', 'd4-ti-2024-b', 'D4 Teknik Informatika', 2024, 'B', 'ACTIVE');
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at)
		VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT', NULL, NULL, NULL),
		       (3, 1, '2023/2024', 'GANJIL', '2023-09-01', '2024-01-31', 'ARCHIVED', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       (4, 2, '2024/2025', 'GANJIL', '2024-09-01', '2025-01-31', 'ARCHIVED', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("gagal menyiapkan validasi semester: %v", err)
	}

	for _, tc := range []struct {
		semesterID string
		wantStatus int
	}{
		{semesterID: "abc", wantStatus: http.StatusUnprocessableEntity},
		{semesterID: "0", wantStatus: http.StatusUnprocessableEntity},
		{semesterID: "2", wantStatus: http.StatusNotFound},
		{semesterID: "3", wantStatus: http.StatusNotFound},
		{semesterID: "4", wantStatus: http.StatusNotFound},
		{semesterID: "999", wantStatus: http.StatusNotFound},
	} {
		t.Run(fmt.Sprintf("semester_%s", tc.semesterID), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/portal/d4-ti-2024-a/tasks?semester_id="+tc.semesterID, nil)
			w := httptest.NewRecorder()
			s.httpServer.Handler.ServeHTTP(w, req)
			if w.Code != tc.wantStatus {
				t.Fatalf("expected %d, got %d: %s", tc.wantStatus, w.Code, w.Body.String())
			}
		})
	}
}

func TestV1Portal_SemestersHideDraftAndUnpublished(t *testing.T) {
	db, s := setupV1TestEnv(t)
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO semesters (id, class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, archived_at)
		VALUES (2, 1, '2025/2026', 'GANJIL', '2025-09-01', '2026-01-31', 'DRAFT', NULL, NULL, NULL),
		       (3, 1, '2023/2024', 'GANJIL', '2023-09-01', '2024-01-31', 'ARCHIVED', NULL, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
		       (4, 1, '2022/2023', 'GANJIL', '2022-09-01', '2023-01-31', 'ARCHIVED', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`)
	if err != nil {
		t.Fatalf("gagal menyiapkan daftar semester: %v", err)
	}

	data := portalResponseData(t, s, "/api/v1/portal/d4-ti-2024-a/semesters")
	if containsPortalValue(data, "id", float64(2)) || containsPortalValue(data, "id", float64(3)) {
		t.Fatalf("semester draf/tidak terbit bocor ke portal: %#v", data)
	}
	if !containsPortalValue(data, "id", float64(1)) || !containsPortalValue(data, "id", float64(4)) {
		t.Fatalf("semester aktif/arsip terbit tidak lengkap: %#v", data)
	}
}

func portalResponseData(t *testing.T, s *Server, path string) any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	s.httpServer.Handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s expected 200, got %d: %s", path, w.Code, w.Body.String())
	}
	var body struct {
		Data any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons %s bukan JSON valid: %v", path, err)
	}
	return body.Data
}

func containsPortalValue(value any, key string, want any) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsPortalValue(item, key, want) {
				return true
			}
		}
	case map[string]any:
		if got, ok := typed[key]; ok && got == want {
			return true
		}
		for _, item := range typed {
			if containsPortalValue(item, key, want) {
				return true
			}
		}
	}
	return false
}
