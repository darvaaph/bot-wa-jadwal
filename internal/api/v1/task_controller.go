package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
)

// CreateTaskV1Request adalah payload pembuatan tugas pengelola
type CreateTaskV1Request struct {
	OfferingID     int64   `json:"offering_id"`
	Title          string  `json:"title"`
	Instructions   string  `json:"instructions"`
	DeadlineAt     string  `json:"deadline_at"`
	TaskType       *string `json:"task_type,omitempty"`
	SubmissionText *string `json:"submission_text,omitempty"`
	SubmissionURL  *string `json:"submission_url,omitempty"`
	SaveAs         string  `json:"save_as"` // "draft" atau "published"
}

// PatchTaskRequest adalah payload pembaruan tugas dengan pemeriksaan versi (optimistic locking)
type PatchTaskRequest struct {
	Version        int     `json:"version"`
	Title          *string `json:"title,omitempty"`
	Instructions   *string `json:"instructions,omitempty"`
	DeadlineAt     *string `json:"deadline_at,omitempty"`
	TaskType       *string `json:"task_type,omitempty"`
	SubmissionText *string `json:"submission_text,omitempty"`
	SubmissionURL  *string `json:"submission_url,omitempty"`
	SaveAs         *string `json:"save_as,omitempty"` // "draft" atau "published"
}

// ReviewTaskRequest adalah payload keputusan review KM
type ReviewTaskRequest struct {
	Decision    string  `json:"decision"` // APPROVED, CHANGES_REQUESTED, REVOKED
	Note        *string `json:"note,omitempty"`
	TaskVersion int     `json:"task_version"`
}

// TaskStateRequest adalah payload untuk mengubah status tugas
type TaskStateRequest struct {
	Version int `json:"version"`
}

// TaskController mengelola endpoint tugas v1
type TaskController struct {
	db *sql.DB
}

// NewTaskController membuat instance controller baru
func NewTaskController(db *sql.DB) *TaskController {
	return &TaskController{db: db}
}

// RegisterRoutes mendaftarkan seluruh endpoint tugas v1 ke ServeMux
func (c *TaskController) RegisterRoutes(mux *http.ServeMux, auth *middleware.AuthManager) {
	mux.HandleFunc("GET /api/v1/tasks", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.GetTasks)))
	mux.HandleFunc("POST /api/v1/tasks", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.CreateTask)))
	mux.HandleFunc("GET /api/v1/tasks/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.GetTaskDetail)))
	mux.HandleFunc("PATCH /api/v1/tasks/{id}", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.PatchTask)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/reviews", auth.RequireAuth(middleware.RequireRole("KM", "SYSTEM_ADMIN")(c.ReviewTask)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/complete", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.CompleteTask)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/archive", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.ArchiveTask)))
	mux.HandleFunc("POST /api/v1/tasks/{id}/restore", auth.RequireAuth(middleware.RequireRole("KM", "PJ", "SYSTEM_ADMIN")(c.RestoreTask)))
}

// GetTasks menangani GET /api/v1/tasks
func (c *TaskController) GetTasks(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	tab := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("tab")))
	if tab == "" {
		tab = "aktif"
	}
	offeringParam := r.URL.Query().Get("offering_id")

	query := `
		SELECT t.id, t.course_offering_id, co.display_name, t.title, t.instructions,
		       t.deadline_at, t.task_type, t.submission_text, t.submission_url,
		       t.publication_status, t.review_state, t.version, t.completed_at, t.archived_at,
		       t.created_at
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.deleted_at IS NULL
	`
	var args []any

	// Filter cakupan peran
	if u.ActiveRole == "PJ" && u.ActiveCourseOfferingID.Valid {
		query += " AND t.course_offering_id = ?"
		args = append(args, u.ActiveCourseOfferingID.Int64)
	} else if u.ActiveRole == "KM" && u.ActiveClassID.Valid {
		query += " AND sem.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	}

	if offeringParam != "" {
		offID, _ := strconv.ParseInt(offeringParam, 10, 64)
		if offID > 0 {
			query += " AND t.course_offering_id = ?"
			args = append(args, offID)
		}
	}

	// Filter berdasarkan Tab
	switch tab {
	case "draf":
		query += " AND t.publication_status = 'DRAFT' AND t.archived_at IS NULL"
	case "review":
		query += " AND t.publication_status = 'PUBLISHED' AND t.review_state = 'NOT_REVIEWED' AND t.archived_at IS NULL"
	case "selesai":
		query += " AND t.completed_at IS NOT NULL AND t.archived_at IS NULL"
	case "terlewat":
		query += " AND t.publication_status = 'PUBLISHED' AND t.deadline_at < datetime('now') AND t.completed_at IS NULL AND t.archived_at IS NULL"
	case "arsip":
		query += " AND t.archived_at IS NOT NULL"
	default: // "aktif"
		query += " AND t.publication_status = 'PUBLISHED' AND t.completed_at IS NULL AND t.archived_at IS NULL"
	}

	query += " ORDER BY t.deadline_at ASC;"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memuat tugas: %v", err))
		return
	}
	defer rows.Close()

	var tasks []map[string]any
	for rows.Next() {
		var id, offID int64
		var offName, title, instr, pubStatus, revState string
		var deadlineAt common.DBTimestamp
		var taskType, subText, subURL sql.NullString
		var version int
		var completedAt, archivedAt, createdAt common.DBTimestamp

		if err := rows.Scan(&id, &offID, &offName, &title, &instr, &deadlineAt, &taskType, &subText, &subURL, &pubStatus, &revState, &version, &completedAt, &archivedAt, &createdAt); err == nil {
			tasks = append(tasks, map[string]any{
				"id":                 id,
				"offering_id":        offID,
				"offering":           offName,
				"title":              title,
				"instructions":       instr,
				"deadline_at":        deadlineAt.RFC3339(),
				"task_type":          taskType.String,
				"submission_text":    subText.String,
				"submission_url":     subURL.String,
				"publication_status": pubStatus,
				"review_state":       revState,
				"version":            version,
				"completed_at": func() any {
					if completedAt.Valid {
						return completedAt.RFC3339()
					}
					return nil
				}(),
				"archived_at": func() any {
					if archivedAt.Valid {
						return archivedAt.RFC3339()
					}
					return nil
				}(),
				"created_at": func() any {
					if createdAt.Valid {
						return createdAt.RFC3339()
					}
					return nil
				}(),
			})
		}
	}

	common.WriteV1Success(w, http.StatusOK, tasks)
}

// CreateTask menangani POST /api/v1/tasks
func (c *TaskController) CreateTask(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req CreateTaskV1Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	var offName string
	var classID int64
	err := c.db.QueryRow(`
		SELECT co.display_name, sem.class_id
		FROM course_offerings co
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE co.id = ? AND co.status = 'ACTIVE';
	`, req.OfferingID).Scan(&offName, &classID)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Course offering tidak ditemukan atau tidak aktif")
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi course offering")
		return
	}

	if u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != req.OfferingID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya berwenang membuat tugas untuk mata kuliah penugasannya")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang membuat tugas untuk kelas penugasannya")
		return
	}

	// 2. Periksa status publikasi
	saveAs := strings.ToLower(strings.TrimSpace(req.SaveAs))
	publicationStatus := "DRAFT"
	reviewState := "NOT_REVIEWED"

	if saveAs == "published" {
		if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Instructions) == "" || strings.TrimSpace(req.DeadlineAt) == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Publikasi tugas wajib menyertakan judul, petunjuk pengerjaan, dan batas waktu")
			return
		}
		hasSubmission := (req.SubmissionText != nil && *req.SubmissionText != "") || (req.SubmissionURL != nil && *req.SubmissionURL != "")
		if !hasSubmission {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Publikasi tugas wajib menyertakan salah satu tempat pengumpulan (submission_url atau submission_text)")
			return
		}
		publicationStatus = "PUBLISHED"
		if u.ActiveRole == "KM" || u.ActiveRole == "SYSTEM_ADMIN" {
			reviewState = "APPROVED"
		}
	} else {
		if strings.TrimSpace(req.Title) == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Judul tugas wajib diisi")
			return
		}
	}

	var deadlineTime time.Time
	if req.DeadlineAt != "" {
		t, err := time.Parse(time.RFC3339, req.DeadlineAt)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04", req.DeadlineAt)
			if err != nil {
				common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format deadline_at harus RFC3339 atau YYYY-MM-DD HH:MM")
				return
			}
		}
		deadlineTime = t
	} else {
		deadlineTime = time.Now().Add(24 * time.Hour)
	}

	// Mulai transaksi
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var taskID int64
	err = tx.QueryRow(`
		INSERT INTO tasks (
			course_offering_id, title, instructions, deadline_at, task_type,
			submission_text, submission_url, publication_status, review_state,
			reviewed_version, created_by_user_id, published_at, version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		RETURNING id;
	`, req.OfferingID, req.Title, req.Instructions, deadlineTime.UTC().Format(time.RFC3339), req.TaskType,
		req.SubmissionText, req.SubmissionURL, publicationStatus, reviewState,
		func() any {
			if reviewState == "APPROVED" {
				return 1
			}
			return nil
		}(),
		u.UserID,
		func() any {
			if publicationStatus == "PUBLISHED" {
				return time.Now().UTC().Format(time.RFC3339)
			}
			return nil
		}(),
	).Scan(&taskID)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan tugas: %v", err))
		return
	}

	if reviewState == "APPROVED" {
		_, err = tx.Exec(`
			INSERT INTO task_reviews (task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note)
			VALUES (?, ?, ?, 1, 'APPROVED', 'Disetujui otomatis saat dibuat oleh KM');
		`, taskID, u.UserID, u.ActiveAssignmentID)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat review otomatis")
			return
		}
	}

	// Audit log
	corrID := fmt.Sprintf("create-task-%d-%d", taskID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "CREATE_TASK",
			EntityType:    "TASK",
			EntityID:      &taskID,
			CorrelationID: corrID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit log")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal melakukan commit transaksi")
		return
	}

	if publicationStatus == "PUBLISHED" && reviewState == "APPROVED" {
		var offName string
		var classID int64
		_ = c.db.QueryRow(`
			SELECT co.display_name, sem.class_id
			FROM course_offerings co
			JOIN semesters sem ON co.semester_id = sem.id
			WHERE co.id = ?;
		`, req.OfferingID).Scan(&offName, &classID)

		sub := ""
		if req.SubmissionURL != nil && *req.SubmissionURL != "" {
			sub = *req.SubmissionURL
		} else if req.SubmissionText != nil && *req.SubmissionText != "" {
			sub = *req.SubmissionText
		}

		common.QueueNotification(c.db, classID, "TASK_PUBLISHED", "TASK", taskID, map[string]any{
			"course":         offName,
			"title":          req.Title,
			"deadline":       deadlineTime.Format("02 Jan 2006 15:04 WIB"),
			"instructions":   req.Instructions,
			"submission_url": sub,
		}, u.UserID)
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{
		"id":                 taskID,
		"publication_status": publicationStatus,
		"review_state":       reviewState,
		"version":            1,
	})
}

// GetTaskDetail menangani GET /api/v1/tasks/{id}
func (c *TaskController) GetTaskDetail(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	taskIDStr := r.PathValue("id")
	taskID, _ := strconv.ParseInt(taskIDStr, 10, 64)

	var (
		id          int64
		offID       int64
		offName     string
		title       string
		instr       string
		deadlineAt  common.DBTimestamp
		taskType    sql.NullString
		subText     sql.NullString
		subURL      sql.NullString
		pubStatus   string
		revState    string
		version     int
		completedAt common.DBTimestamp
		archivedAt  common.DBTimestamp
		classID     int64
	)

	err := c.db.QueryRow(`
		SELECT t.id, t.course_offering_id, co.display_name, t.title, t.instructions,
		       t.deadline_at, t.task_type, t.submission_text, t.submission_url,
		       t.publication_status, t.review_state, t.version, t.completed_at, t.archived_at,
		       sem.class_id
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.id = ? AND t.deleted_at IS NULL;
	`, taskID).Scan(
		&id, &offID, &offName, &title, &instr, &deadlineAt, &taskType, &subText,
		&subURL, &pubStatus, &revState, &version, &completedAt, &archivedAt, &classID,
	)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tugas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat tugas")
		return
	}

	if u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses ditolak: hanya untuk PJ mata kuliah ini")
		return
	} else if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses ditolak: hanya untuk KM kelas ini")
		return
	}

	var reviews []map[string]any
	revRows, err := c.db.Query(`
		SELECT tr.id, u.display_name, tr.task_version, tr.decision, COALESCE(tr.note, ''), tr.created_at
		FROM task_reviews tr
		JOIN users u ON tr.reviewer_user_id = u.id
		WHERE tr.task_id = ?
		ORDER BY tr.created_at DESC;
	`, taskID)
	if err == nil {
		defer revRows.Close()
		for revRows.Next() {
			var rID int64
			var reviewer, decision, note string
			var taskVer int
			var createdAt common.DBTimestamp
			if err := revRows.Scan(&rID, &reviewer, &taskVer, &decision, &note, &createdAt); err == nil {
				reviews = append(reviews, map[string]any{
					"id":           rID,
					"reviewer":     reviewer,
					"task_version": taskVer,
					"decision":     decision,
					"note":         note,
					"created_at":   createdAt.RFC3339(),
				})
			}
		}
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"task": map[string]any{
			"id":                 id,
			"offering_id":        offID,
			"offering":           offName,
			"title":              title,
			"instructions":       instr,
			"deadline_at":        deadlineAt.RFC3339(),
			"task_type":          taskType.String,
			"submission_text":    subText.String,
			"submission_url":     subURL.String,
			"publication_status": pubStatus,
			"review_state":       revState,
			"version":            version,
			"is_completed":       completedAt.Valid,
			"is_archived":        archivedAt.Valid,
			"completed_at": func() any {
				if completedAt.Valid {
					return completedAt.RFC3339()
				}
				return nil
			}(),
			"archived_at": func() any {
				if archivedAt.Valid {
					return archivedAt.RFC3339()
				}
				return nil
			}(),
		},
		"reviews": reviews,
	})
}

// PatchTask menangani PATCH /api/v1/tasks/{id}
func (c *TaskController) PatchTask(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	taskIDStr := r.PathValue("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil || taskID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID tugas tidak valid")
		return
	}

	var req PatchTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	var (
		curID         int64
		curOfferingID int64
		curTitle      string
		curInstr      string
		curDeadline   common.DBTimestamp
		curTaskType   sql.NullString
		curSubText    sql.NullString
		curSubURL     sql.NullString
		curPubStatus  string
		curRevState   string
		curVersion    int
		curClassID    int64
	)

	err = c.db.QueryRow(`
		SELECT t.id, t.course_offering_id, t.title, t.instructions, t.deadline_at,
		       t.task_type, t.submission_text, t.submission_url, t.publication_status,
		       t.review_state, t.version, s.class_id
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters s ON co.semester_id = s.id
		WHERE t.id = ? AND t.deleted_at IS NULL;
	`, taskID).Scan(
		&curID, &curOfferingID, &curTitle, &curInstr, &curDeadline,
		&curTaskType, &curSubText, &curSubURL, &curPubStatus,
		&curRevState, &curVersion, &curClassID,
	)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tugas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi tugas")
		return
	}

	if u.ActiveRole == "PJ" {
		if !u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != curOfferingID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "PJ hanya diizinkan memperbarui tugas untuk mata kuliah yang ditugaskan")
			return
		}
	} else if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya diizinkan memperbarui tugas di kelasnya")
			return
		}
	}

	// Optimistic Locking: Periksa kesesuaian versi
	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict,
			"Data tugas telah diperbarui oleh pengguna lain. Silakan periksa perbedaan versi.",
			map[string]any{
				"current_version": curVersion,
				"current_data": map[string]any{
					"title":              curTitle,
					"instructions":       curInstr,
					"deadline_at":        curDeadline.RFC3339(),
					"task_type":          curTaskType.String,
					"submission_text":    curSubText.String,
					"submission_url":     curSubURL.String,
					"publication_status": curPubStatus,
					"review_state":       curRevState,
				},
			},
		)
		return
	}

	newTitle := curTitle
	if req.Title != nil && strings.TrimSpace(*req.Title) != "" {
		newTitle = strings.TrimSpace(*req.Title)
	}

	newInstr := curInstr
	if req.Instructions != nil {
		newInstr = strings.TrimSpace(*req.Instructions)
	}

	newDeadline := curDeadline.Time
	if req.DeadlineAt != nil && strings.TrimSpace(*req.DeadlineAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.DeadlineAt))
		if err != nil {
			parsed, err = time.Parse("2006-01-02 15:04:05", strings.TrimSpace(*req.DeadlineAt))
		}
		if err != nil {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Format deadline_at tidak valid (gunakan RFC3339)")
			return
		}
		newDeadline = parsed.UTC()
	}

	newTaskType := curTaskType
	if req.TaskType != nil {
		newTaskType = sql.NullString{String: strings.TrimSpace(*req.TaskType), Valid: strings.TrimSpace(*req.TaskType) != ""}
	}

	newSubText := curSubText
	if req.SubmissionText != nil {
		newSubText = sql.NullString{String: strings.TrimSpace(*req.SubmissionText), Valid: strings.TrimSpace(*req.SubmissionText) != ""}
	}

	newSubURL := curSubURL
	if req.SubmissionURL != nil {
		newSubURL = sql.NullString{String: strings.TrimSpace(*req.SubmissionURL), Valid: strings.TrimSpace(*req.SubmissionURL) != ""}
	}

	newPubStatus := curPubStatus
	if req.SaveAs != nil {
		saveAs := strings.ToUpper(strings.TrimSpace(*req.SaveAs))
		if saveAs == "PUBLISHED" {
			newPubStatus = "PUBLISHED"
		} else if saveAs == "DRAFT" {
			newPubStatus = "DRAFT"
		}
	}

	if newPubStatus == "PUBLISHED" {
		if strings.TrimSpace(newTitle) == "" || strings.TrimSpace(newInstr) == "" || newDeadline.IsZero() {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Publikasi tugas wajib menyertakan judul, petunjuk pengerjaan, dan batas waktu")
			return
		}
		hasSubmission := (newSubText.Valid && strings.TrimSpace(newSubText.String) != "") || (newSubURL.Valid && strings.TrimSpace(newSubURL.String) != "")
		if !hasSubmission {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Publikasi tugas wajib menyertakan salah satu tempat pengumpulan (submission_url atau submission_text)")
			return
		}
	}

	newVersion := curVersion + 1
	newRevState := "NOT_REVIEWED"
	var newReviewedVersion any
	if newPubStatus == "PUBLISHED" && (u.ActiveRole == "KM" || u.ActiveRole == "SYSTEM_ADMIN") {
		newRevState = "APPROVED"
		newReviewedVersion = newVersion
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE tasks
		SET title = ?, instructions = ?, deadline_at = ?, task_type = ?,
		    submission_text = ?, submission_url = ?, publication_status = ?,
		    review_state = ?, reviewed_version = ?,
		    published_at = CASE WHEN ? = 'PUBLISHED' THEN COALESCE(published_at, CURRENT_TIMESTAMP) ELSE published_at END,
		    version = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND version = ?;
	`, newTitle, newInstr, newDeadline.Format(time.RFC3339), newTaskType,
		newSubText, newSubURL, newPubStatus, newRevState, newReviewedVersion,
		newPubStatus, newVersion, taskID, curVersion)

	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui data tugas")
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi tugas berubah saat penyimpanan berlangsung", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"publication_status": curPubStatus,
			},
		})
		return
	}

	if (u.ActiveRole == "KM" || u.ActiveRole == "SYSTEM_ADMIN") && newPubStatus == "PUBLISHED" {
		_, err = tx.Exec(`
			INSERT INTO task_reviews (
				task_id, task_version, reviewer_user_id, reviewer_role_assignment_id,
				decision, note, created_at
			)
			VALUES (?, ?, ?, ?, 'APPROVED', 'Pembaruan tugas disetujui langsung oleh KM', CURRENT_TIMESTAMP);
		`, taskID, newVersion, u.UserID, u.ActiveAssignmentID)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat review otomatis")
			return
		}
	}

	corrID := fmt.Sprintf("patch-task-%d-%d", taskID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"version":%d,"title":%q}`, curVersion, curTitle)
		afterJSON := fmt.Sprintf(`{"version":%d,"title":%q,"review_state":%q}`, newVersion, newTitle, newRevState)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &curClassID,
			Action:        "UPDATE_TASK",
			EntityType:    "TASK",
			EntityID:      &taskID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			CorrelationID: corrID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit log")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal melakukan commit transaksi")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"id":                 taskID,
		"title":              newTitle,
		"deadline_at":        newDeadline.Format(time.RFC3339),
		"publication_status": newPubStatus,
		"review_state":       newRevState,
		"version":            newVersion,
	})
}

// ReviewTask menangani POST /api/v1/tasks/{id}/reviews
func (c *TaskController) ReviewTask(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "KM" && u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang meninjau tugas")
		return
	}

	taskIDStr := r.PathValue("id")
	taskID, _ := strconv.ParseInt(taskIDStr, 10, 64)

	var req ReviewTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}

	decision := strings.ToUpper(strings.TrimSpace(req.Decision))
	if decision != "APPROVED" && decision != "CHANGES_REQUESTED" && decision != "REVOKED" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Keputusan review harus APPROVED, CHANGES_REQUESTED, atau REVOKED")
		return
	}
	if (decision == "CHANGES_REQUESTED" || decision == "REVOKED") && (req.Note == nil || strings.TrimSpace(*req.Note) == "") {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "note wajib diisi untuk CHANGES_REQUESTED atau REVOKED")
		return
	}

	var currentVersion int
	var currentPubStatus string
	var classID int64
	err := c.db.QueryRow(`
		SELECT t.version, t.publication_status, sem.class_id
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.id = ? AND t.deleted_at IS NULL;
	`, taskID).Scan(&currentVersion, &currentPubStatus, &classID)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tugas tidak ditemukan")
		return
	}

	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya diizinkan mereview tugas di kelasnya")
		return
	}

	if req.TaskVersion != currentVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi tugas telah berubah. Muat ulang untuk melihat revisi terbaru.", map[string]any{
			"current_version": currentVersion,
			"current_data": map[string]any{
				"publication_status": currentPubStatus,
			},
		})
		return
	}

	newPubStatus := currentPubStatus
	if decision == "CHANGES_REQUESTED" {
		newPubStatus = "DRAFT"
	} else if decision == "REVOKED" {
		newPubStatus = "REVOKED"
	} else if decision == "APPROVED" {
		newPubStatus = "PUBLISHED"
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi review")
		return
	}
	defer tx.Rollback()

	newVersion := currentVersion + 1
	res, err := tx.Exec(`
		UPDATE tasks
		SET publication_status = ?, review_state = ?, reviewed_version = ?, version = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND version = ?;
	`, newPubStatus, decision, newVersion, newVersion, taskID, currentVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status tugas")
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi tugas berubah selama review", map[string]any{
			"current_version": currentVersion,
			"current_data": map[string]any{
				"publication_status": currentPubStatus,
			},
		})
		return
	}

	_, err = tx.Exec(`
		INSERT INTO task_reviews (task_id, reviewer_user_id, reviewer_role_assignment_id, task_version, decision, note)
		VALUES (?, ?, ?, ?, ?, ?);
	`, taskID, u.UserID, u.ActiveAssignmentID, newVersion, decision, req.Note)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan catatan review")
		return
	}

	correlationID := fmt.Sprintf("review-task-%d-%d", taskID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := fmt.Sprintf(`{"decision":%q}`, decision)
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "REVIEW_TASK",
			EntityType:    "TASK",
			EntityID:      &taskID,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit log")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal melakukan commit transaksi")
		return
	}

	if decision == "APPROVED" {
		var offName, title, instr, subURL, subText string
		var deadlineAt common.DBTimestamp
		var classID int64
		_ = c.db.QueryRow(`
			SELECT co.display_name, t.title, COALESCE(t.instructions, ''),
			       t.deadline_at, COALESCE(t.submission_url, ''), COALESCE(t.submission_text, ''),
			       sem.class_id
			FROM tasks t
			JOIN course_offerings co ON t.course_offering_id = co.id
			JOIN semesters sem ON co.semester_id = sem.id
			WHERE t.id = ?;
		`, taskID).Scan(&offName, &title, &instr, &deadlineAt, &subURL, &subText, &classID)

		sub := subURL
		if sub == "" {
			sub = subText
		}

		common.QueueNotification(c.db, classID, "TASK_PUBLISHED", "TASK", taskID, map[string]any{
			"course":         offName,
			"title":          title,
			"deadline":       deadlineAt.Time.Format("02 Jan 2006 15:04 WIB"),
			"instructions":   instr,
			"submission_url": sub,
		}, u.UserID)
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"task_id":            taskID,
		"publication_status": newPubStatus,
		"review_state":       decision,
		"version":            newVersion,
	})
}

// CompleteTask menangani POST /api/v1/tasks/{id}/complete
func (c *TaskController) CompleteTask(w http.ResponseWriter, r *http.Request) {
	c.handleTaskStateChange(w, r, "complete")
}

// ArchiveTask menangani POST /api/v1/tasks/{id}/archive
func (c *TaskController) ArchiveTask(w http.ResponseWriter, r *http.Request) {
	c.handleTaskStateChange(w, r, "archive")
}

// RestoreTask menangani POST /api/v1/tasks/{id}/restore
func (c *TaskController) RestoreTask(w http.ResponseWriter, r *http.Request) {
	c.handleTaskStateChange(w, r, "restore")
}

func (c *TaskController) handleTaskStateChange(w http.ResponseWriter, r *http.Request, action string) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	taskIDStr := r.PathValue("id")
	taskID, _ := strconv.ParseInt(taskIDStr, 10, 64)

	var req TaskStateRequest
	if r.Body == nil || r.Body == http.NoBody {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib diisi")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	if req.Version <= 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "version wajib lebih dari nol")
		return
	}

	var curVersion int
	var offID, classID int64
	var curPub, curRev string
	err := c.db.QueryRow(`
		SELECT t.version, t.course_offering_id, sem.class_id, t.publication_status, t.review_state
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		JOIN semesters sem ON co.semester_id = sem.id
		WHERE t.id = ?;
	`, taskID).Scan(&curVersion, &offID, &classID, &curPub, &curRev)

	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tugas tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memverifikasi tugas: %v", err))
		return
	}

	if u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses ditolak")
		return
	} else if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses ditolak")
		return
	}

	if req.Version != curVersion {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi tugas berubah", map[string]any{
			"current_version": curVersion,
			"current_data": map[string]any{
				"publication_status": curPub,
				"review_state":       curRev,
			},
		})
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	var updateQuery string
	var auditAction string
	if action == "complete" {
		updateQuery = "UPDATE tasks SET completed_at = CURRENT_TIMESTAMP, reviewed_version = CASE WHEN review_state = 'NOT_REVIEWED' THEN NULL ELSE version + 1 END, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND version = ?;"
		auditAction = "COMPLETE_TASK"
	} else if action == "archive" {
		updateQuery = "UPDATE tasks SET archived_at = CURRENT_TIMESTAMP, reviewed_version = CASE WHEN review_state = 'NOT_REVIEWED' THEN NULL ELSE version + 1 END, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND version = ?;"
		auditAction = "ARCHIVE_TASK"
	} else if action == "restore" {
		updateQuery = "UPDATE tasks SET archived_at = NULL, completed_at = NULL, deleted_at = NULL, deleted_by_user_id = NULL, reviewed_version = CASE WHEN review_state = 'NOT_REVIEWED' THEN NULL ELSE version + 1 END, version = version + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND version = ?;"
		auditAction = "RESTORE_TASK"
	}

	res, err := tx.Exec(updateQuery, taskID, curVersion)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memperbarui status tugas: %v", err))
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi tugas berubah", map[string]any{"current_version": curVersion})
		return
	}

	corrID := fmt.Sprintf("%s-task-%d-%d", action, taskID, time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        auditAction,
			EntityType:    "TASK",
			EntityID:      &taskID,
			CorrelationID: corrID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan audit log: %v", err))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal melakukan commit transaksi: %v", err))
		return
	}

	if action == "restore" {
		common.WriteV1Success(w, http.StatusOK, map[string]any{"restored": true, "version": curVersion + 1})
	} else {
		common.WriteV1Success(w, http.StatusOK, map[string]any{"updated": true, "version": curVersion + 1})
	}
}
