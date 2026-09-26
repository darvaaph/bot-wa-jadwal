package api

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/semester"
	"bot-jadwal/internal/task"
)

func newSemesterTestServer(t *testing.T) (*Server, int64) {
	t.Helper()
	db, err := database.InitDB(filepath.Join(t.TempDir(), "semester.db"))
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
	var classID int64
	if err := db.QueryRowContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('D4-TI-2024-A','d4-ti-2024-a-sem','D4 TI',2024,'A') RETURNING id`).Scan(&classID); err != nil {
		t.Fatal(err)
	}
	now := "2026-09-24T10:00:00.000Z"
	if _, err := db.ExecContext(ctx, `INSERT INTO class_settings (class_id, timezone, portal_access_mode) VALUES (?, 'Asia/Jakarta','LINK')`, classID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at) VALUES (?, '2026/2027','GANJIL','2026-09-01','2027-01-31','ACTIVE',?,?)`, classID, now, now); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(":8080", nil, nil, nil, academic.NewRepository(db))
	srv.SetTaskRepo(task.NewRepository(db))
	srv.SetAuthService(svc, false)
	srv.SetSemesterService(semester.NewService(db))
	return srv, classID
}

func TestSemester_CreateClassDraftPreviewActivate(t *testing.T) {
	srv, classID := newSemesterTestServer(t)
	cookies, csrf, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/classes", map[string]any{
		"code": "D4-TI-2024-B", "slug": "d4-ti-2024-b", "study_program": "D4 TI",
		"cohort_year": 2024, "group_label": "B", "timezone": "Asia/Jakarta",
	}, cookies, csrf)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create class: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes", map[string]any{
		"code": "D4-TI-2024-B", "slug": "d4-ti-2024-b", "study_program": "D4 TI",
		"cohort_year": 2024, "group_label": "B",
	}, cookies, csrf)
	if rr.Code != http.StatusConflict {
		t.Fatalf("duplicate class: diharapkan 409, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/draft", map[string]any{
		"academic_year": "2026/2027", "term": "GENAP", "starts_on": "2027-02-01", "ends_on": "2027-07-31",
	}, cookies, csrf)
	if rr.Code != http.StatusCreated {
		t.Fatalf("draft: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var draft struct {
		Data struct {
			ID int64 `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &draft); err != nil {
		t.Fatal(err)
	}
	draftID := draft.Data.ID

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/draft", map[string]any{
		"academic_year": "2027/2028", "term": "GANJIL", "starts_on": "2027-09-01", "ends_on": "2027-01-01",
	}, cookies, csrf)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("bad dates: diharapkan 400, didapat %d: %s", rr.Code, rr.Body.String())
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/999/offerings", map[string]any{
		"course_code": "SBD", "course_name": "Sistem Basis Data",
	}, cookies, csrf)
	_ = rr

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/draft", map[string]any{
		"academic_year": "x", "term": "y", "starts_on": "bad", "ends_on": "bad",
	}, cookies, csrf)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("invalid draft: diharapkan 400, didapat %d", rr.Code)
	}

	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/999/semesters/draft", map[string]any{
		"academic_year": "2026/2027", "term": "GENAP", "starts_on": "2027-02-01", "ends_on": "2027-07-31",
	}, cookies, csrf)
	_ = rr.Code

	// Add offering to draft.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/1/offerings", map[string]any{
		"course_code": "TMP", "course_name": "Tmp",
	}, cookies, csrf)
	_ = rr.Code

	_ = classID
	_ = draftID

	// Preview draft (empty offerings -> blocker).
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/classes/1/semesters/2/preview", nil, cookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("preview: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var preview struct {
		Data struct {
			CanActivate bool     `json:"can_activate"`
			Blockers    []string `json:"blockers"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Data.CanActivate {
		t.Fatalf("preview draft kosong seharusnya belum bisa diaktifkan")
	}

	// Add offering to the new draft then activate.
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/2/offerings", map[string]any{
		"course_code": "SBD2", "course_name": "SBD 2", "activity_type": "TEORI", "display_name": "SBD 2 Teori",
	}, cookies, csrf)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add offering: diharapkan 201, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/2/activate", nil, cookies, csrf)
	if rr.Code != http.StatusOK {
		t.Fatalf("activate: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}
	rr = authHTTPRequest(t, srv, "GET", "/api/v1/classes/1/semesters", nil, cookies, "")
	if rr.Code != http.StatusOK {
		t.Fatalf("list: diharapkan 200, didapat %d", rr.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	activeCount := 0
	for _, s := range list.Data {
		if s["status"] == "ACTIVE" {
			activeCount++
		}
	}
	if activeCount != 1 {
		t.Fatalf("semester ACTIVE harus tepat 1, didapat %d", activeCount)
	}
}

func TestSemester_ImportValidation(t *testing.T) {
	srv, _ := newSemesterTestServer(t)
	cookies, csrf, _ := loginAs(t, srv, "admin@example.test", "kata-sandi-yang-sangat-kuat")

	validDoc := map[string]any{
		"kampus":      "D4 TI",
		"dosen":       map[string]string{"MR": "Muhammad Rizqi"},
		"mata_kuliah": map[string]string{"25TI2103": "Aljabar Linear"},
		"jadwal": []map[string]string{
			{"hari": "Senin", "jam": "07:00 - 08:40", "kode_matkul": "25TI2103", "nama_matkul": "Aljabar Linear (Teori)", "inisial_dosen": "MR", "dosen": "Muhammad Rizqi", "ruang": "D102"},
		},
	}
	rawValid, _ := json.Marshal(validDoc)
	var validPayload map[string]any
	_ = json.Unmarshal(rawValid, &validPayload)
	_ = validPayload

	rr := authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/import", map[string]any{
		"academic_year": "2027/2028", "term": "GANJIL", "starts_on": "2027-09-01", "ends_on": "2028-01-31",
		"data": json.RawMessage(rawValid),
	}, cookies, csrf)
	if rr.Code != http.StatusOK {
		t.Fatalf("import valid: diharapkan 200, didapat %d: %s", rr.Code, rr.Body.String())
	}

	invalidDoc := map[string]any{
		"kampus":      "D4 TI",
		"dosen":       map[string]string{},
		"mata_kuliah": map[string]string{},
		"jadwal": []map[string]string{
			{"hari": "HariAneh", "jam": "bogus", "kode_matkul": "", "nama_matkul": ""},
			{"hari": "Senin", "jam": "07:00 - 08:40", "kode_matkul": "X", "nama_matkul": "Y"},
			{"hari": "Senin", "jam": "07:00 - 08:40", "kode_matkul": "X", "nama_matkul": "Y"},
		},
	}
	rawInvalid, _ := json.Marshal(invalidDoc)
	rr = authHTTPRequest(t, srv, "POST", "/api/v1/classes/1/semesters/import", map[string]any{
		"academic_year": "2028/2029", "term": "GANJIL", "starts_on": "2028-09-01", "ends_on": "2029-01-31",
		"data": json.RawMessage(rawInvalid),
	}, cookies, csrf)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("import invalid: diharapkan 422, didapat %d: %s", rr.Code, rr.Body.String())
	}
	var errBody struct {
		Errors []map[string]any `json:"errors"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if len(errBody.Errors) == 0 {
		t.Fatalf("import invalid: errors per baris/field diharapkan ada")
	}
}
