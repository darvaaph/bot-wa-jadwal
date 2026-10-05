package v1

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/schedule"
)

// PreviewPattern menghitung dampak perubahan versi tanpa menyimpan data.
func (c *ScheduleController) PreviewPattern(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	patternID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || patternID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID jadwal tetap tidak valid")
		return
	}
	var req PatchPatternRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	var offID, classID, semesterID int64
	var day, version int
	var start, end, effectiveOld, semesterStart, semesterEnd, semesterStatus, offeringName string
	var room sql.NullInt64
	err = c.db.QueryRow(`SELECT sp.course_offering_id, s.class_id, s.id, sp.day_of_week, sp.version,
		sp.start_time, sp.end_time, sp.effective_from, s.starts_on, s.ends_on, s.status, co.display_name, sp.room_id
		FROM schedule_patterns sp JOIN course_offerings co ON co.id=sp.course_offering_id
		JOIN semesters s ON s.id=co.semester_id WHERE sp.id=? AND sp.status='ACTIVE' AND sp.effective_until IS NULL`, patternID).
		Scan(&offID, &classID, &semesterID, &day, &version, &start, &end, &effectiveOld,
			&semesterStart, &semesterEnd, &semesterStatus, &offeringName, &room)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Jadwal tetap tidak ditemukan")
		return
	}
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat jadwal tetap")
		return
	}
	effectiveOld = effectiveOld[:10]
	semesterStart = semesterStart[:10]
	semesterEnd = semesterEnd[:10]
	if (u.ActiveRole == "PJ" && (!u.ActiveCourseOfferingID.Valid || u.ActiveCourseOfferingID.Int64 != offID)) ||
		(u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID)) {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Jadwal tetap di luar cakupan Anda")
		return
	}
	if req.Version != version {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Versi jadwal berubah", map[string]any{"current_version": version})
		return
	}
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.Local
	}
	var date string
	var parsed time.Time
	if semesterStatus == "DRAFT" {
		date = semesterStart
		var err error
		parsed, err = time.ParseInLocation("2006-01-02", date, loc)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Tanggal semester tidak valid")
			return
		}
	} else {
		today := time.Now().In(loc).Format("2006-01-02")
		date = strings.TrimSpace(req.EffectiveFrom)
		var dateErr error
		parsed, dateErr = time.ParseInLocation("2006-01-02", date, loc)
		if dateErr != nil || parsed.Format("2006-01-02") != date || date < today || date < effectiveOld || date < semesterStart || date > semesterEnd || semesterStatus != "ACTIVE" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tanggal berlaku harus dari hari ini sampai akhir semester aktif")
			return
		}
		if strings.TrimSpace(req.Reason) == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan perubahan wajib diisi")
			return
		}
	}
	if req.OfferingID != nil && *req.OfferingID != offID {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Perubahan permanen harus tetap pada mata kuliah asal")
		return
	}
	newDay := day
	if req.DayOfWeek != 0 {
		newDay = req.DayOfWeek
	}
	if newDay < 1 || newDay > 7 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Hari tidak valid")
		return
	}
	newStart := start
	if req.StartTime != "" {
		newStart = req.StartTime
	}
	startAt, err := time.Parse("15:04", newStart)
	if err != nil || req.DurationMin < 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Jam atau durasi tidak valid")
		return
	}
	newEnd := end
	if req.DurationMin > 0 {
		newEnd = startAt.Add(time.Duration(req.DurationMin) * time.Minute).Format("15:04")
	}
	if newEnd <= newStart {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Jam selesai harus setelah jam mulai")
		return
	}
	if req.RoomID != nil {
		room = sql.NullInt64{Int64: *req.RoomID, Valid: true}
	}
	var roomID *int64
	if room.Valid {
		roomID = &room.Int64
	}
	lecturerIDs := lecturersForOfferingCtx(r.Context(), c.db, offID)
	conflicts, err := checkPatternRange(r.Context(), c.db, schedule.Candidate{
		OwnerClassID: classID, OwnerOfferingID: offID, RoomID: roomID,
		LecturerIDs: lecturerIDs,
		PatternDay:  newDay, PatternStart: newStart, PatternEnd: newEnd,
		ExcludePatternID: &patternID,
	}, date, semesterEnd)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa konflik")
		return
	}
	endDate, _ := time.ParseInLocation("2006-01-02", semesterEnd, loc)
	impacted := 0
	firstDates := []string{}
	for current := parsed; !current.After(endDate); current = current.AddDate(0, 0, 1) {
		weekday := int(current.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		if weekday != newDay {
			continue
		}
		impacted++
		if len(firstDates) < 5 {
			firstDates = append(firstDates, current.Format("2006-01-02"))
		}
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"pattern_id": patternID, "version": version, "semester_id": semesterID,
		"effective_from": date, "offering": offeringName,
		"old":                    map[string]any{"offering": offeringName, "day_of_week": day, "start_time": start, "end_time": end},
		"new":                    map[string]any{"offering": offeringName, "day_of_week": newDay, "start_time": newStart, "end_time": newEnd},
		"affected_session_count": impacted, "first_session_dates": firstDates,
		"conflicts": conflicts, "blocking": schedule.HasBlocking(conflicts),
	})
}
