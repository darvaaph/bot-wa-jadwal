package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/schedule"
)

func (c *ScheduleController) PreviewCreatePattern(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, 401, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	var req CreatePatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OfferingID <= 0 || req.DayOfWeek < 1 || req.DayOfWeek > 7 || req.DurationMin <= 0 || req.DurationMin >= 24*60 {
		common.WriteV1Error(w, 422, common.CodeValidation, "Mata kuliah, hari, dan durasi harus valid")
		return
	}
	start, err := time.Parse("15:04", req.StartTime)
	if err != nil || start.Hour()*60+start.Minute()+req.DurationMin >= 24*60 {
		common.WriteV1Error(w, 422, common.CodeValidation, "Jam mulai dan durasi harus selesai pada hari yang sama")
		return
	}
	var classID int64
	var semesterStart, semesterEnd, semesterStatus string
	err = c.db.QueryRowContext(r.Context(), `SELECT sem.class_id, sem.starts_on, sem.ends_on, sem.status FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=? AND co.status='ACTIVE'`, req.OfferingID).Scan(&classID, &semesterStart, &semesterEnd, &semesterStatus)
	if err != nil {
		common.WriteV1Error(w, 404, common.CodeNotFound, "Mata kuliah tidak ditemukan")
		return
	}
	if u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != req.OfferingID) || u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, 403, common.CodeForbidden, "Mata kuliah di luar cakupan peran")
		return
	}
	assignedLecturers := lecturersForOfferingCtx(r.Context(), c.db, req.OfferingID)
	lecturers := assignedLecturers
	if req.LecturerIDs != nil {
		allowed := map[int64]bool{}
		for _, id := range assignedLecturers {
			allowed[id] = true
		}
		for _, id := range req.LecturerIDs {
			if !allowed[id] {
				common.WriteV1Error(w, 422, common.CodeValidation, "Dosen tidak terdaftar pada mata kuliah")
				return
			}
		}
		lecturers = req.LecturerIDs
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.Local
	}
	semesterStart, semesterEnd = semesterStart[:10], semesterEnd[:10]
	var effectiveFrom string
	switch semesterStatus {
	case "DRAFT":
		effectiveFrom = semesterStart
	case "ACTIVE":
		effectiveFrom = time.Now().In(loc).Format("2006-01-02")
		if effectiveFrom < semesterStart || effectiveFrom > semesterEnd {
			common.WriteV1Error(w, 422, common.CodeValidation, "Pola baru hanya dapat dibuat dalam semester aktif")
			return
		}
	default:
		common.WriteV1Error(w, 422, common.CodeValidation, "Pola baru hanya dapat dibuat dalam semester draf atau aktif")
		return
	}
	end := start.Add(time.Duration(req.DurationMin) * time.Minute).Format("15:04")
	conflicts, err := checkPatternRange(r.Context(), c.db, schedule.Candidate{OwnerClassID: classID, OwnerOfferingID: req.OfferingID, RoomID: req.RoomID, LecturerIDs: lecturers,
		PatternDay: req.DayOfWeek, PatternStart: req.StartTime, PatternEnd: end}, effectiveFrom, semesterEnd)
	if err != nil {
		common.WriteV1Error(w, 500, "DB_ERROR", "Gagal memeriksa konflik")
		return
	}
	common.WriteV1Success(w, 200, map[string]any{"offering_id": req.OfferingID, "day_of_week": req.DayOfWeek, "start_time": req.StartTime,
		"end_time": end, "duration_min": req.DurationMin, "room_id": req.RoomID, "effective_from": effectiveFrom, "effective_until": semesterEnd, "conflicts": conflictMaps(conflicts), "can_publish": !schedule.HasBlocking(conflicts)})
}
