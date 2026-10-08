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

type kmDashboardTask struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Offering   string `json:"offering"`
	DeadlineAt string `json:"deadline_at"`
	Review     string `json:"review_state"`
}

type kmDashboardSession struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Room       string `json:"room"`
	Lecturers  string `json:"lecturers"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	OfferingID int64  `json:"offering_id"`
}

type kmDashboardData struct {
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
	Date  string `json:"date"`
	Tasks struct {
		ActiveCount int               `json:"active_count"`
		ReviewCount int               `json:"review_count"`
		NearCount   int               `json:"near_count"`
		Priority    []kmDashboardTask `json:"priority"`
	} `json:"tasks"`
	Schedule struct {
		WeekCount int                  `json:"week_count"`
		Today     []kmDashboardSession `json:"today"`
	} `json:"schedule"`
	Attention struct {
		FailedMessages     int `json:"failed_messages"`
		DraftEvents        int `json:"draft_events"`
		PendingInvitations int `json:"pending_invitations"`
	} `json:"attention"`
}

// handleKMDashboard returns one class-scoped, active-semester snapshot.
func (s *Server) handleKMDashboard(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok || u.ActiveRole != "KM" || !u.ActiveClassID.Valid {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Konteks KM kelas tidak aktif")
		return
	}
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusServiceUnavailable, CodeServiceDown, "Database belum siap")
		return
	}
	classID := u.ActiveClassID.Int64
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
	data := kmDashboardData{}
	data.Class.Code, data.Class.Slug = code, slug
	data.Class.Label = strings.TrimSpace(program + " " + group)
	data.Class.Timezone = zone
	data.Date = now.Format("2006-01-02")
	data.Tasks.Priority = []kmDashboardTask{}
	data.Schedule.Today = []kmDashboardSession{}

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
		if err := s.fillKMDashboardTasks(r, &data, semesterID, now); err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Ringkasan tugas gagal dimuat")
			return
		}
		if err := s.fillKMDashboardSchedule(r, &data, semesterID, loc, now); err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Agenda kelas gagal dimuat")
			return
		}
		if err := s.v1DB.QueryRowContext(r.Context(), `
			SELECT COUNT(*) FROM teaching_events te
			JOIN teaching_event_offerings teo ON teo.teaching_event_id = te.id AND teo.participation_role = 'OWNER'
			JOIN course_offerings co ON co.id = teo.course_offering_id
			WHERE co.semester_id = ? AND te.lifecycle_status = 'DRAFT'`, semesterID,
		).Scan(&data.Attention.DraftEvents); err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Draf jadwal gagal dihitung")
			return
		}
	}
	if err := s.v1DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM notification_messages WHERE class_id = ? AND status = 'FAILED'`, classID,
	).Scan(&data.Attention.FailedMessages); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Pesan gagal gagal dihitung")
		return
	}
	if err := s.v1DB.QueryRowContext(r.Context(), `
		SELECT COUNT(*) FROM role_invitations
		WHERE class_id = ? AND status = 'PENDING' AND julianday(expires_at) > julianday('now')`, classID,
	).Scan(&data.Attention.PendingInvitations); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Undangan gagal dihitung")
		return
	}
	s.writeV1Success(w, http.StatusOK, data)
}

func (s *Server) fillKMDashboardTasks(r *http.Request, data *kmDashboardData, semesterID int64, now time.Time) error {
	rows, err := s.v1DB.QueryContext(r.Context(), `
		SELECT t.id, t.title, co.display_name, t.deadline_at, t.review_state
		FROM tasks t JOIN course_offerings co ON co.id = t.course_offering_id
		WHERE co.semester_id = ? AND t.publication_status = 'PUBLISHED'
		  AND t.completed_at IS NULL AND t.archived_at IS NULL AND t.deleted_at IS NULL
		ORDER BY t.deadline_at ASC, t.id ASC`, semesterID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var task kmDashboardTask
		var deadline common.DBTimestamp
		if err := rows.Scan(&task.ID, &task.Title, &task.Offering, &deadline, &task.Review); err != nil {
			return err
		}
		data.Tasks.ActiveCount++
		if task.Review == "NOT_REVIEWED" {
			data.Tasks.ReviewCount++
		}
		if deadline.Valid {
			task.DeadlineAt = deadline.Time.Format(time.RFC3339)
			if !deadline.Time.Before(now) && !deadline.Time.After(now.Add(72*time.Hour)) {
				data.Tasks.NearCount++
			}
		}
		if len(data.Tasks.Priority) < 3 {
			data.Tasks.Priority = append(data.Tasks.Priority, task)
		}
	}
	return rows.Err()
}

type kmPattern struct {
	ID         int64
	OfferingID int64
	Title      string
	Room       string
	Lecturers  string
	Day        int
	Start      string
	End        string
	From       string
	Until      sql.NullString
}

func (s *Server) fillKMDashboardSchedule(r *http.Request, data *kmDashboardData, semesterID int64, loc *time.Location, now time.Time) error {
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

	byDate := map[string][]kmDashboardSession{}
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
			byDate[date] = append(byDate[date], kmDashboardSession{
				ID: "pattern-" + strconv.FormatInt(p.ID, 10), Kind: "REGULAR", Title: p.Title, Room: p.Room,
				Lecturers: p.Lecturers, StartsAt: start.Format(time.RFC3339), EndsAt: end.Format(time.RFC3339), OfferingID: p.OfferingID,
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
		byDate[date] = append(byDate[date], kmDashboardSession{
			ID: "event-" + strconv.FormatInt(id, 10), Kind: kind, Title: title, Room: room,
			Lecturers: lecturers, StartsAt: start.In(loc).Format(time.RFC3339), EndsAt: end.In(loc).Format(time.RFC3339), OfferingID: offeringID,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for date, items := range byDate {
		sort.Slice(items, func(i, j int) bool { return items[i].StartsAt < items[j].StartsAt })
		data.Schedule.WeekCount += len(items)
		if date == data.Date {
			data.Schedule.Today = items
		}
	}
	return nil
}
