package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
)

type pjDashboardTask struct {
	ID                int64  `json:"id"`
	Title             string `json:"title"`
	Offering          string `json:"offering"`
	Lecturers         string `json:"lecturers"`
	DeadlineAt        string `json:"deadline_at"`
	PublicationStatus string `json:"publication_status"`
	ReviewState       string `json:"review_state"`
	CreatedByName     string `json:"created_by_name"`
	CreatedAt         string `json:"created_at"`
	Version           int    `json:"version"`
}

type pjDashboardSession struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Room       string `json:"room"`
	Lecturers  string `json:"lecturers"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	OfferingID int64  `json:"offering_id"`
	IsMyCourse bool   `json:"is_my_course"`
}

type pjDashboardCorrection struct {
	Needed       bool   `json:"needed"`
	Count        int    `json:"count"`
	TaskID       int64  `json:"task_id,omitempty"`
	TaskTitle    string `json:"task_title,omitempty"`
	ReviewerName string `json:"reviewer_name,omitempty"`
	Note         string `json:"note,omitempty"`
}

type pjDashboardOfferingItem struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	CourseCode  string `json:"course_code"`
	CourseName  string `json:"course_name"`
}

type pjDashboardData struct {
	Class struct {
		Code     string `json:"code"`
		Slug     string `json:"slug"`
		Label    string `json:"label"`
		Timezone string `json:"timezone"`
	} `json:"class"`
	Semester *struct {
		ID           int64  `json:"id"`
		AcademicYear string `json:"academic_year"`
		Term         string `json:"term"`
	} `json:"semester"`
	Offering struct {
		ID          int64  `json:"id"`
		DisplayName string `json:"display_name"`
		CourseCode  string `json:"course_code"`
		CourseName  string `json:"course_name"`
	} `json:"offering"`
	Offerings []pjDashboardOfferingItem `json:"offerings"`
	Date      string                    `json:"date"`
	Metrics   struct {
		PublishedCount     int    `json:"published_count"`
		NearCount          int    `json:"near_count"`
		PendingReviewCount int    `json:"pending_review_count"`
		PendingSubmittedAt string `json:"pending_submitted_at,omitempty"`
		TodayTotalSessions int    `json:"today_total_sessions"`
		TodayPJSessions    int    `json:"today_pj_sessions"`
	} `json:"metrics"`
	Correction pjDashboardCorrection `json:"correction"`
	Tasks      struct {
		List []pjDashboardTask `json:"list"`
	} `json:"tasks"`
	Schedule struct {
		Today []pjDashboardSession `json:"today"`
	} `json:"schedule"`
}

// handlePJDashboard mengembalikan ringkasan khusus untuk peran PJ Mata Kuliah.
func (s *Server) handlePJDashboard(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok || (u.ActiveRole != "PJ" && u.ActiveRole != "SYSTEM_ADMIN") {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Konteks PJ tidak aktif")
		return
	}
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusServiceUnavailable, CodeServiceDown, "Database belum siap")
		return
	}

	var classID int64
	if u.ActiveClassID.Valid {
		classID = u.ActiveClassID.Int64
	} else if classParam := r.URL.Query().Get("class_id"); classParam != "" {
		classID, _ = strconv.ParseInt(classParam, 10, 64)
	} else if classSlug := r.URL.Query().Get("class_slug"); classSlug != "" {
		_ = s.v1DB.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug = ?`, classSlug).Scan(&classID)
	}

	if classID == 0 && u.ActiveCourseOfferingID.Valid {
		_ = s.v1DB.QueryRowContext(r.Context(), `
			SELECT sem.class_id
			FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id
			WHERE co.id = ?`, u.ActiveCourseOfferingID.Int64).Scan(&classID)
	}

	if classID == 0 {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Konteks kelas PJ tidak aktif")
		return
	}

	var code, slug, program, group, zone string
	if err := s.v1DB.QueryRowContext(r.Context(), `
		SELECT c.code, c.slug, c.study_program, c.group_label, cs.timezone
		FROM classes c JOIN class_settings cs ON cs.class_id = c.id WHERE c.id = ?`, classID,
	).Scan(&code, &slug, &program, &group, &zone); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Kelas aktif gagal dimuat")
		return
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Zona waktu kelas tidak valid")
		return
	}
	now := time.Now().In(loc)

	data := pjDashboardData{}
	data.Class.Code, data.Class.Slug = code, slug
	data.Class.Label = strings.TrimSpace(program + " " + group)
	data.Class.Timezone = zone
	data.Date = now.Format("2006-01-02")
	data.Offerings = []pjDashboardOfferingItem{}
	data.Tasks.List = []pjDashboardTask{}
	data.Schedule.Today = []pjDashboardSession{}

	var semesterID int64
	var year, term string
	err = s.v1DB.QueryRowContext(r.Context(), `
		SELECT id, academic_year, term FROM semesters
		WHERE class_id = ? AND status = 'ACTIVE'`, classID,
	).Scan(&semesterID, &year, &term)
	if err != nil && err != sql.ErrNoRows {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Semester aktif gagal dimuat")
		return
	}
	if err == nil {
		data.Semester = &struct {
			ID           int64  `json:"id"`
			AcademicYear string `json:"academic_year"`
			Term         string `json:"term"`
		}{semesterID, year, term}

		offRows, err := s.v1DB.QueryContext(r.Context(), `
			SELECT co.id, co.display_name, c.code, c.name
			FROM course_offerings co
			JOIN courses c ON c.id = co.course_id
			WHERE co.semester_id = ? AND co.status = 'ACTIVE'
			ORDER BY co.display_name ASC`, semesterID)
		if err == nil {
			defer offRows.Close()
			for offRows.Next() {
				var item pjDashboardOfferingItem
				if err := offRows.Scan(&item.ID, &item.DisplayName, &item.CourseCode, &item.CourseName); err == nil {
					data.Offerings = append(data.Offerings, item)
				}
			}
		}

		var chosenOfferingID int64
		if offParam := r.URL.Query().Get("offering_id"); offParam != "" {
			if parsed, _ := strconv.ParseInt(offParam, 10, 64); parsed > 0 {
				chosenOfferingID = parsed
			}
		}
		if chosenOfferingID == 0 && u.ActiveCourseOfferingID.Valid {
			chosenOfferingID = u.ActiveCourseOfferingID.Int64
		}
		if chosenOfferingID == 0 && len(data.Offerings) > 0 {
			chosenOfferingID = data.Offerings[0].ID
		}

		for _, o := range data.Offerings {
			if o.ID == chosenOfferingID {
				data.Offering.ID = o.ID
				data.Offering.DisplayName = o.DisplayName
				data.Offering.CourseCode = o.CourseCode
				data.Offering.CourseName = o.CourseName
				break
			}
		}

		if chosenOfferingID > 0 {
			if err := s.fillPJDashboardTasks(r, &data, chosenOfferingID, now); err != nil {
				s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Ringkasan tugas gagal dimuat")
				return
			}
			if err := s.fillPJDashboardCorrection(r, &data, chosenOfferingID); err != nil {
				s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Data koreksi gagal dimuat")
				return
			}
		}

		if err := s.fillPJDashboardSchedule(r, &data, semesterID, chosenOfferingID, loc, now); err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Agenda kelas gagal dimuat")
			return
		}
	}

	s.writeV1Success(w, http.StatusOK, data)
}

func (s *Server) fillPJDashboardCorrection(r *http.Request, data *pjDashboardData, offeringID int64) error {
	var count int
	if err := s.v1DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM tasks
		WHERE course_offering_id = ? AND review_state = 'CHANGES_REQUESTED' AND deleted_at IS NULL`,
		offeringID,
	).Scan(&count); err != nil {
		return err
	}
	data.Correction.Count = count
	if count > 0 {
		data.Correction.Needed = true
		var taskID int64
		var title, reviewer, note string
		err := s.v1DB.QueryRowContext(r.Context(), `
			SELECT t.id, t.title, COALESCE(u.display_name, 'Ketua Murid'), COALESCE(tr.note, '')
			FROM tasks t
			LEFT JOIN task_reviews tr ON tr.task_id = t.id AND tr.decision = 'CHANGES_REQUESTED'
			LEFT JOIN users u ON u.id = tr.reviewer_user_id
			WHERE t.course_offering_id = ? AND t.review_state = 'CHANGES_REQUESTED' AND t.deleted_at IS NULL
			ORDER BY tr.created_at DESC, t.id DESC
			LIMIT 1`, offeringID,
		).Scan(&taskID, &title, &reviewer, &note)
		if err == nil {
			data.Correction.TaskID = taskID
			data.Correction.TaskTitle = title
			data.Correction.ReviewerName = reviewer
			data.Correction.Note = note
		}
	}
	return nil
}

func (s *Server) fillPJDashboardTasks(r *http.Request, data *pjDashboardData, offeringID int64, now time.Time) error {
	rows, err := s.v1DB.QueryContext(r.Context(), `
		SELECT t.id, t.title, co.display_name,
		       COALESCE((SELECT GROUP_CONCAT(l.full_name, ', ') FROM offering_lecturers ol
		                 JOIN lecturers l ON l.id = ol.lecturer_id
		                 WHERE ol.course_offering_id = co.id AND ol.superseded_at IS NULL), ''),
		       t.deadline_at, t.publication_status, t.review_state,
		       COALESCE(u.display_name, 'PJ Mata Kuliah'),
		       t.created_at, t.version, t.completed_at, t.archived_at
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		LEFT JOIN users u ON u.id = t.created_by_user_id
		WHERE t.course_offering_id = ? AND t.deleted_at IS NULL AND t.archived_at IS NULL
		ORDER BY
			CASE WHEN t.review_state = 'CHANGES_REQUESTED' THEN 0
			     WHEN t.publication_status = 'PUBLISHED' AND t.completed_at IS NULL THEN 1
			     ELSE 2 END,
			t.deadline_at ASC, t.id DESC`, offeringID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var task pjDashboardTask
		var deadline common.DBTimestamp
		var createdAt common.DBTimestamp
		var completedAt, archivedAt common.DBTimestamp
		if err := rows.Scan(
			&task.ID, &task.Title, &task.Offering, &task.Lecturers,
			&deadline, &task.PublicationStatus, &task.ReviewState,
			&task.CreatedByName, &createdAt, &task.Version, &completedAt, &archivedAt,
		); err != nil {
			return err
		}
		if deadline.Valid {
			task.DeadlineAt = deadline.Time.Format(time.RFC3339)
		}
		if createdAt.Valid {
			task.CreatedAt = createdAt.Time.Format(time.RFC3339)
		}

		if task.PublicationStatus == "PUBLISHED" && !completedAt.Valid && !archivedAt.Valid {
			data.Metrics.PublishedCount++
			if deadline.Valid && !deadline.Time.Before(now) && !deadline.Time.After(now.Add(72*time.Hour)) {
				data.Metrics.NearCount++
			}
		}

		if task.ReviewState == "NOT_REVIEWED" && !archivedAt.Valid {
			data.Metrics.PendingReviewCount++
			if data.Metrics.PendingSubmittedAt == "" && createdAt.Valid {
				data.Metrics.PendingSubmittedAt = createdAt.Time.Format(time.RFC3339)
			}
		}

		if len(data.Tasks.List) < 10 {
			data.Tasks.List = append(data.Tasks.List, task)
		}
	}
	return rows.Err()
}

func (s *Server) fillPJDashboardSchedule(r *http.Request, data *pjDashboardData, semesterID, myOfferingID int64, loc *time.Location, now time.Time) error {
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
	weekEnd := weekStart.AddDate(0, 0, 7)
	rows, err := s.v1DB.QueryContext(r.Context(), `
		SELECT sp.id, sp.course_offering_id, co.display_name, COALESCE(r.code, ''),
		       COALESCE((SELECT GROUP_CONCAT(l.full_name, ', ') FROM offering_lecturers ol
		                 JOIN lecturers l ON l.id = ol.lecturer_id
		                 WHERE ol.course_offering_id = co.id AND ol.superseded_at IS NULL), ''),
		       sp.day_of_week, sp.start_time, sp.end_time, sp.effective_from, sp.effective_until
		FROM schedule_patterns sp
		JOIN course_offerings co ON co.id = sp.course_offering_id
		LEFT JOIN rooms r ON r.id = sp.room_id
		WHERE co.semester_id = ? AND sp.status = 'ACTIVE'`, semesterID)
	if err != nil {
		return err
	}
	patterns := []kmPattern{}
	for rows.Next() {
		var p kmPattern
		if err := rows.Scan(&p.ID, &p.OfferingID, &p.Title, &p.Room, &p.Lecturers, &p.Day, &p.Start, &p.End, &p.From, &p.Until); err != nil {
			rows.Close()
			return err
		}
		patterns = append(patterns, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	byDate := map[string][]pjDashboardSession{}
	for d := weekStart; d.Before(weekEnd); d = d.AddDate(0, 0, 1) {
		date := d.Format("2006-01-02")
		weekday := (int(d.Weekday())+6)%7 + 1
		for _, p := range patterns {
			if p.Day != weekday || date < p.From || (p.Until.Valid && date > p.Until.String) {
				continue
			}
			start, e1 := time.ParseInLocation("2006-01-02 15:04", date+" "+p.Start, loc)
			end, e2 := time.ParseInLocation("2006-01-02 15:04", date+" "+p.End, loc)
			if e1 != nil || e2 != nil {
				continue
			}
			byDate[date] = append(byDate[date], pjDashboardSession{
				ID: "pattern-" + strconv.FormatInt(p.ID, 10), Kind: "REGULAR", Title: p.Title, Room: p.Room,
				Lecturers: p.Lecturers, StartsAt: start.Format(time.RFC3339), EndsAt: end.Format(time.RFC3339), OfferingID: p.OfferingID,
				IsMyCourse: p.OfferingID == myOfferingID,
			})
		}
	}

	rows, err = s.v1DB.QueryContext(r.Context(), `
		SELECT te.id, te.event_kind, te.starts_at, te.ends_at, te.origin_schedule_pattern_id,
		       te.origin_occurrence_date, co.id, co.display_name, COALESCE(r.code, ''),
		       COALESCE((SELECT GROUP_CONCAT(l.full_name, ', ') FROM offering_lecturers ol
		                 JOIN lecturers l ON l.id = ol.lecturer_id
		                 WHERE ol.course_offering_id = co.id AND ol.superseded_at IS NULL), '')
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON co.id = teo.course_offering_id
		LEFT JOIN rooms r ON r.id = te.room_id
		WHERE co.semester_id = ? AND te.lifecycle_status = 'PUBLISHED'
		  AND (te.origin_occurrence_date BETWEEN ? AND ? OR substr(te.starts_at,1,10) BETWEEN ? AND ?)`,
		semesterID, weekStart.Format("2006-01-02"), weekEnd.AddDate(0, 0, -1).Format("2006-01-02"),
		weekStart.AddDate(0, 0, -1).Format("2006-01-02"), weekEnd.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, offeringID int64
		var kind, startRaw, endRaw, title, room, lecturers string
		var originID sql.NullInt64
		var originDate sql.NullString
		if err := rows.Scan(&id, &kind, &startRaw, &endRaw, &originID, &originDate, &offeringID, &title, &room, &lecturers); err != nil {
			return err
		}
		start, e1 := common.ParseTime(startRaw)
		end, e2 := common.ParseTime(endRaw)
		if e1 != nil || e2 != nil {
			return fmt.Errorf("waktu kejadian %d tidak valid", id)
		}
		date := start.In(loc).Format("2006-01-02")
		if originDate.Valid && len(originDate.String) >= 10 && originID.Valid {
			origin := originDate.String[:10]
			list := byDate[origin][:0]
			for _, item := range byDate[origin] {
				if item.ID != "pattern-"+strconv.FormatInt(originID.Int64, 10) {
					list = append(list, item)
				}
			}
			byDate[origin] = list
		} else if kind == "HOLIDAY" || kind == "SESSION_CANCELLED" {
			list := byDate[date][:0]
			for _, item := range byDate[date] {
				if item.Kind != "REGULAR" || item.OfferingID != offeringID {
					list = append(list, item)
				}
			}
			byDate[date] = list
		}
		if kind != "REPLACEMENT" && kind != "EXTRA" {
			continue
		}
		if start.In(loc).Before(weekStart) || !start.In(loc).Before(weekEnd) {
			continue
		}
		byDate[date] = append(byDate[date], pjDashboardSession{
			ID: "event-" + strconv.FormatInt(id, 10), Kind: kind, Title: title, Room: room,
			Lecturers: lecturers, StartsAt: start.In(loc).Format(time.RFC3339), EndsAt: end.In(loc).Format(time.RFC3339), OfferingID: offeringID,
			IsMyCourse: offeringID == myOfferingID,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for date, items := range byDate {
		sort.Slice(items, func(i, j int) bool { return items[i].StartsAt < items[j].StartsAt })
		if date == data.Date {
			data.Schedule.Today = items
			data.Metrics.TodayTotalSessions = len(items)
			for _, it := range items {
				if it.IsMyCourse {
					data.Metrics.TodayPJSessions++
				}
			}
		}
	}
	return nil
}
