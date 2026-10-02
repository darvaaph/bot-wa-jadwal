package v1

import (
	"database/sql"
	"net/http"
	"sort"
	"time"

	"bot-jadwal/internal/api/common"
)

// Courses returns the published academic detail for each offering in one semester.
func (c *PortalController) Courses(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	var classID int64
	if err := c.db.QueryRowContext(r.Context(), `SELECT id FROM classes WHERE slug=?`, slug).Scan(&classID); err != nil {
		if err == sql.ErrNoRows {
			common.WriteV1Error(w, 404, common.CodeNotFound, "Kelas tidak ditemukan")
		} else {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat kelas")
		}
		return
	}
	if !c.portalAccessAllowed(w, r, classID) {
		return
	}
	semesterID, ok := c.resolvePortalSemester(w, r, classID)
	if !ok {
		return
	}
	location := time.UTC
	var timezone string
	if err := c.db.QueryRowContext(r.Context(), `SELECT timezone FROM class_settings WHERE class_id=?`, classID).Scan(&timezone); err == nil {
		if loc, err := time.LoadLocation(timezone); err == nil {
			location = loc
		}
	}
	today := time.Now().In(location).Format("2006-01-02")
	rows, err := c.db.QueryContext(r.Context(), `SELECT co.id, co.display_name, co.activity_type, cr.code, cr.name
		FROM course_offerings co JOIN courses cr ON cr.id=co.course_id
		WHERE co.semester_id=? AND co.status='ACTIVE' ORDER BY cr.name, co.id`, semesterID)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat mata kuliah")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var name, activity, code, course string
		if err := rows.Scan(&id, &name, &activity, &code, &course); err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca mata kuliah")
			return
		}
		item := map[string]any{"id": id, "display_name": name, "activity_type": activity, "course_code": code, "course_name": course,
			"lecturers": c.getOfferingLecturers(id), "schedule": []map[string]any{}, "tasks": []map[string]any{}, "materials": []map[string]any{}}
		patterns, err := c.db.QueryContext(r.Context(), `SELECT day_of_week,start_time,end_time,COALESCE(r.code,''),COALESCE(sp.meeting_link,'')
			FROM schedule_patterns sp LEFT JOIN rooms r ON r.id=sp.room_id WHERE sp.course_offering_id=? AND sp.status='ACTIVE'
			AND sp.effective_from<=? AND (sp.effective_until IS NULL OR sp.effective_until>=?) ORDER BY day_of_week,start_time`, id, today, today)
		if err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat jadwal mata kuliah")
			return
		}
		for patterns.Next() {
			var day int
			var start, end, room, link string
			if err := patterns.Scan(&day, &start, &end, &room, &link); err == nil {
				item["schedule"] = append(item["schedule"].([]map[string]any), map[string]any{"day_of_week": day, "start_time": start, "end_time": end, "room": room, "meeting_link": link})
			}
		}
		patterns.Close()
		tasks, err := c.db.QueryContext(r.Context(), `SELECT id,title,deadline_at FROM tasks WHERE course_offering_id=? AND publication_status='PUBLISHED' AND deleted_at IS NULL AND archived_at IS NULL ORDER BY deadline_at`, id)
		if err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat tugas mata kuliah")
			return
		}
		for tasks.Next() {
			var taskID int64
			var title, deadline string
			if err := tasks.Scan(&taskID, &title, &deadline); err == nil {
				item["tasks"] = append(item["tasks"].([]map[string]any), map[string]any{"id": taskID, "title": title, "deadline_at": deadline})
			}
		}
		tasks.Close()
		materials, err := c.db.QueryContext(r.Context(), `SELECT id,title,material_type,url FROM materials WHERE course_offering_id=? AND class_id=? AND status='ACTIVE' AND visibility='CLASS_ACCESS' AND deleted_at IS NULL ORDER BY id DESC`, id, classID)
		if err != nil {
			common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memuat materi mata kuliah")
			return
		}
		for materials.Next() {
			var matID int64
			var title, kind, url string
			if err := materials.Scan(&matID, &title, &kind, &url); err == nil {
				item["materials"] = append(item["materials"].([]map[string]any), map[string]any{"id": matID, "title": title, "material_type": kind, "url": url})
			}
		}
		materials.Close()
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal membaca mata kuliah")
		return
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["course_name"].(string) < out[j]["course_name"].(string) })
	common.WriteV1Success(w, 200, out)
}
