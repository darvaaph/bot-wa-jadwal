package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Candidate adalah input yang diperiksa konfliknya.
// Waktu dibaca sebagai UTC; normalisasi zona dilakukan caller via OwnerTimezone.
type Candidate struct {
	OwnerClassID     int64
	OwnerOfferingID  int64
	ParticipantIDs   []int64
	RoomID           *int64
	LecturerIDs      []int64
	StartsAt         time.Time
	EndsAt           time.Time
	ExcludePatternID *int64
	ExcludeEventID   *int64
	// PatternTime dipakai untuk create/patch pola reguler (tanpa tanggal konkret):
	// DayOfWeek 1-7, StartTime/EndTime "HH:MM", Date untuk cek effective range.
	PatternDay   int
	PatternStart string
	PatternEnd   string
	PatternDate  string // YYYY-MM-DD, default hari ini
}

// Conflict adalah fakta benturan yang terdeteksi engine.
type Conflict struct {
	Code       string `json:"code"`
	Blocking   bool   `json:"blocking"`
	EntityType string `json:"entity_type"`
	EntityID   int64  `json:"entity_id"`
	StartsAt   string `json:"starts_at"`
	EndsAt     string `json:"ends_at"`
	Message    string `json:"message"`
}

const (
	CodeClassOverlap       = "CLASS_TIME_OVERLAP"
	CodeRoomOverlap        = "ROOM_TIME_OVERLAP"
	CodeLecturerOverlap    = "LECTURER_TIME_OVERLAP"
	CodeOfferingOverlap    = "OFFERING_TIME_OVERLAP"
	CodeParticipantOverlap = "PARTICIPANT_TIME_OVERLAP"
	CodeRoomConfirmNeeded  = "ROOM_CONFIRMATION_REQUIRED"
	CodeOutsideOwnerSem    = "OUTSIDE_OWNER_SEMESTER"
	CodeOutsidePartSem     = "OUTSIDE_PARTICIPANT_SEMESTER"
)

// overlaps memakai interval setengah terbuka [start, end).
// Sesi yang bersentuhan tepat di batas tidak bentrok.
func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

func parseHM(hm string) (int, int, bool) {
	var h, m int
	if _, err := fmt.Sscanf(hm, "%d:%d", &h, &m); err != nil {
		return 0, 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// CheckConflicts memeriksa candidate terhadap pola efektif + event PUBLISHED.
// Hasil stabil dan terurut; DRAFT/REVOKED diabaikan sebagai sesi aktif.
func CheckConflicts(ctx context.Context, db *sql.DB, c Candidate) ([]Conflict, error) {
	if !c.EndsAt.After(c.StartsAt) && c.PatternDay == 0 {
		return nil, fmt.Errorf("interval tidak valid")
	}
	var out []Conflict
	ownerLocation := time.UTC
	if c.OwnerClassID > 0 {
		var timezone string
		if err := db.QueryRowContext(ctx, `SELECT timezone FROM class_settings WHERE class_id=?`, c.OwnerClassID).Scan(&timezone); err == nil {
			if location, err := time.LoadLocation(timezone); err == nil {
				ownerLocation = location
			}
		}
	}

	// 1. Batas semester owner (hanya mode event bertanggal konkret).
	// Mode pola (PatternDay != 0) bersifat rekuren mingguan tanpa tanggal
	// tunggal; effective range pola sudah dibatasi query sesi di bawah.
	var semStart, semEnd string
	if c.OwnerOfferingID > 0 && c.PatternDay == 0 {
		err := db.QueryRowContext(ctx, `SELECT sem.starts_on, sem.ends_on FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?`, c.OwnerOfferingID).Scan(&semStart, &semEnd)
		if err == nil {
			d := c.StartsAt.Format("2006-01-02")
			if d < semStart || d > semEnd {
				out = append(out, Conflict{Code: CodeOutsideOwnerSem, Blocking: true, EntityType: "SEMESTER", Message: "kejadian di luar semester owner"})
			}
		}
	}
	// 2. Batas semester peserta.
	for _, pid := range c.ParticipantIDs {
		var ps, pe string
		err := db.QueryRowContext(ctx, `SELECT sem.starts_on, sem.ends_on FROM course_offerings co
			JOIN semesters sem ON sem.id = co.semester_id WHERE co.id = ?`, pid).Scan(&ps, &pe)
		if err == nil {
			d := c.StartsAt.Format("2006-01-02")
			if d < ps || d > pe {
				out = append(out, Conflict{Code: CodeOutsidePartSem, Blocking: true, EntityType: "COURSE_OFFERING", EntityID: pid, Message: "event di luar semester peserta"})
			}
		}
	}

	// Kumpulkan sesi aktif: pola efektif + event PUBLISHED.
	type session struct {
		entityType string
		entityID   int64
		offeringID int64
		classID    int64
		roomID     *int64
		starts     time.Time
		ends       time.Time
		lecturers  []int64
	}
	var sessions []session

	if c.PatternDay != 0 {
		// Mode pola: bandingkan jam HH:MM pada hari yang sama.
		date := c.PatternDate
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		rows, err := db.QueryContext(ctx, `SELECT sp.id, sp.course_offering_id, sem.class_id, sp.room_id,
			sp.start_time, sp.end_time FROM schedule_patterns sp
			JOIN course_offerings co ON co.id = sp.course_offering_id
			JOIN semesters sem ON sem.id = co.semester_id
			WHERE sp.status='ACTIVE' AND sp.day_of_week=? AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)`,
			c.PatternDay, date, date)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, off, classID int64
				var room sql.NullInt64
				var st, en string
				if err := rows.Scan(&id, &off, &classID, &room, &st, &en); err != nil {
					continue
				}
				if c.ExcludePatternID != nil && id == *c.ExcludePatternID {
					continue
				}
				// Overlap jam memakai string HH:MM dengan interval [start,end).
				if !(c.PatternStart < en && st < c.PatternEnd) {
					continue
				}
				var r *int64
				if room.Valid {
					v := room.Int64
					r = &v
				}
				lects := lecturersForOffering(ctx, db, off)
				sessions = append(sessions, session{"SCHEDULE_PATTERN", id, off, classID, r, time.Time{}, time.Time{}, lects})
			}
			rows.Close()
		}
		// Event PUBLISHED pada tanggal date yang jamnya overlap.
		rows2, err := db.QueryContext(ctx, `SELECT te.id, teo.course_offering_id, sem.class_id, te.room_id, te.starts_at, te.ends_at
			FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id=te.id AND teo.participation_role='OWNER'
			JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id
			WHERE te.lifecycle_status='PUBLISHED' AND date(te.starts_at) BETWEEN date(?, '-1 day') AND date(?, '+1 day')`, date, date)
		if err == nil {
			defer rows2.Close()
			for rows2.Next() {
				var id, off, classID int64
				var room sql.NullInt64
				var ss, es string
				if err := rows2.Scan(&id, &off, &classID, &room, &ss, &es); err != nil {
					continue
				}
				if c.ExcludeEventID != nil && id == *c.ExcludeEventID {
					continue
				}
				st, err1 := parseTime(ss)
				en, err2 := parseTime(es)
				if err1 != nil || err2 != nil {
					continue
				}
				// Konversi candidate pola ke time pada tanggal date untuk overlap.
				ch, cm, ok1 := parseHM(c.PatternStart)
				eh, em, ok2 := parseHM(c.PatternEnd)
				if !ok1 || !ok2 {
					continue
				}
				day, _ := time.Parse("2006-01-02", date)
				cs := time.Date(day.Year(), day.Month(), day.Day(), ch, cm, 0, 0, ownerLocation)
				ce := time.Date(day.Year(), day.Month(), day.Day(), eh, em, 0, 0, ownerLocation)
				if !overlaps(cs, ce, st, en) {
					continue
				}
				var r *int64
				if room.Valid {
					v := room.Int64
					r = &v
				}
				lects := lecturersForOffering(ctx, db, off)
				sessions = append(sessions, session{"TEACHING_EVENT", id, off, classID, r, st, en, lects})
			}
			rows2.Close()
		}
	} else {
		// Mode event: pola efektif pada tanggal candidate + event overlap.
		localStart := c.StartsAt.In(ownerLocation)
		date := localStart.Format("2006-01-02")
		dow := weekdayNum(localStart)
		rows, err := db.QueryContext(ctx, `SELECT sp.id, sp.course_offering_id, sem.class_id, sp.room_id,
			sp.start_time, sp.end_time FROM schedule_patterns sp
			JOIN course_offerings co ON co.id = sp.course_offering_id
			JOIN semesters sem ON sem.id = co.semester_id
			WHERE sp.status='ACTIVE' AND sp.day_of_week=? AND sp.effective_from <= ? AND (sp.effective_until IS NULL OR sp.effective_until >= ?)`,
			dow, date, date)
		if err == nil {
			for rows.Next() {
				var id, off, classID int64
				var room sql.NullInt64
				var st, en string
				if err := rows.Scan(&id, &off, &classID, &room, &st, &en); err != nil {
					continue
				}
				if c.ExcludePatternID != nil && id == *c.ExcludePatternID {
					continue
				}
				ch, cm, ok1 := parseHM(st)
				eh, em, ok2 := parseHM(en)
				if !ok1 || !ok2 {
					continue
				}
				ps := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), ch, cm, 0, 0, ownerLocation)
				pe := time.Date(localStart.Year(), localStart.Month(), localStart.Day(), eh, em, 0, 0, ownerLocation)
				if !overlaps(c.StartsAt, c.EndsAt, ps, pe) {
					continue
				}
				var r *int64
				if room.Valid {
					v := room.Int64
					r = &v
				}
				lects := lecturersForOffering(ctx, db, off)
				sessions = append(sessions, session{"SCHEDULE_PATTERN", id, off, classID, r, ps, pe, lects})
			}
			rows.Close()
		}
		rows2, err := db.QueryContext(ctx, `SELECT te.id, teo.course_offering_id, sem.class_id, te.room_id, te.starts_at, te.ends_at
			FROM teaching_events te JOIN teaching_event_offerings teo ON teo.teaching_event_id=te.id AND teo.participation_role='OWNER'
			JOIN course_offerings co ON co.id=teo.course_offering_id JOIN semesters sem ON sem.id=co.semester_id
			WHERE te.lifecycle_status='PUBLISHED' AND te.starts_at < ? AND te.ends_at > ?`,
			c.EndsAt.UTC().Format(time.RFC3339), c.StartsAt.UTC().Format(time.RFC3339))
		if err == nil {
			for rows2.Next() {
				var id, off, classID int64
				var room sql.NullInt64
				var ss, es string
				if err := rows2.Scan(&id, &off, &classID, &room, &ss, &es); err != nil {
					continue
				}
				if c.ExcludeEventID != nil && id == *c.ExcludeEventID {
					continue
				}
				st, err1 := parseTime(ss)
				en, err2 := parseTime(es)
				if err1 != nil || err2 != nil {
					continue
				}
				var r *int64
				if room.Valid {
					v := room.Int64
					r = &v
				}
				lects := lecturersForOffering(ctx, db, off)
				sessions = append(sessions, session{"TEACHING_EVENT", id, off, classID, r, st, en, lects})
			}
			rows2.Close()
		}
	}

	lectSet := map[int64]bool{}
	for _, l := range c.LecturerIDs {
		lectSet[l] = true
	}
	partSet := map[int64]bool{}
	for _, p := range c.ParticipantIDs {
		partSet[p] = true
	}
	for _, s := range sessions {
		// Offering sama.
		if c.OwnerOfferingID > 0 && s.offeringID == c.OwnerOfferingID {
			out = append(out, Conflict{Code: CodeOfferingOverlap, Blocking: true, EntityType: s.entityType, EntityID: s.entityID, StartsAt: s.starts.Format(time.RFC3339), EndsAt: s.ends.Format(time.RFC3339), Message: "offering yang sama memiliki sesi lain pada interval tersebut"})
		}
		// Kelas sama (offering berbeda).
		if c.OwnerClassID > 0 && s.classID == c.OwnerClassID && s.offeringID != c.OwnerOfferingID {
			out = append(out, Conflict{Code: CodeClassOverlap, Blocking: true, EntityType: s.entityType, EntityID: s.entityID, StartsAt: s.starts.Format(time.RFC3339), EndsAt: s.ends.Format(time.RFC3339), Message: "kelas memiliki sesi lain pada interval tersebut"})
		}
		// Ruangan sama.
		if c.RoomID != nil && s.roomID != nil && *c.RoomID == *s.roomID {
			out = append(out, Conflict{Code: CodeRoomOverlap, Blocking: true, EntityType: s.entityType, EntityID: s.entityID, StartsAt: s.starts.Format(time.RFC3339), EndsAt: s.ends.Format(time.RFC3339), Message: "ruangan dipakai sesi lain pada interval tersebut"})
		}
		// Dosen sama.
		for _, l := range s.lecturers {
			if lectSet[l] {
				out = append(out, Conflict{Code: CodeLecturerOverlap, Blocking: true, EntityType: s.entityType, EntityID: s.entityID, StartsAt: s.starts.Format(time.RFC3339), EndsAt: s.ends.Format(time.RFC3339), Message: "dosen mengajar sesi lain pada interval tersebut"})
				break
			}
		}
		// Peserta: sesi dari kelas peserta yang bentrok = nonblocking (perlu keputusan eksplisit).
		if partSet[s.offeringID] {
			out = append(out, Conflict{Code: CodeParticipantOverlap, Blocking: false, EntityType: s.entityType, EntityID: s.entityID, StartsAt: s.starts.Format(time.RFC3339), EndsAt: s.ends.Format(time.RFC3339), Message: "kelas peserta memiliki sesi lain (perlu konfirmasi)"})
		}
	}
	// Ruangan belum konfirmasi TU = nonblocking.
	if c.RoomID != nil {
		out = append(out, Conflict{Code: CodeRoomConfirmNeeded, Blocking: false, EntityType: "ROOM", EntityID: *c.RoomID, Message: "ruangan perlu konfirmasi TU sebelum publikasi"})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		if out[i].EntityType != out[j].EntityType {
			return out[i].EntityType < out[j].EntityType
		}
		return out[i].EntityID < out[j].EntityID
	})
	if out == nil {
		out = []Conflict{}
	}
	return out, nil
}

func lecturersForOffering(ctx context.Context, db *sql.DB, offeringID int64) []int64 {
	rows, err := db.QueryContext(ctx, `SELECT lecturer_id FROM offering_lecturers WHERE course_offering_id=? AND superseded_at IS NULL`, offeringID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

func parseTime(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("format waktu tidak dikenal")
}

func weekdayNum(t time.Time) int {
	wd := int(t.Weekday())
	if wd == 0 {
		return 7
	}
	return wd
}

// IsBlocking menandakan apakah daftar konflik menolak publish.
func HasBlocking(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if c.Blocking {
			return true
		}
	}
	return false
}

// NeedsOverride menandakan perlunya conflict_override_reason.
func NeedsOverride(conflicts []Conflict) bool {
	for _, c := range conflicts {
		if !c.Blocking && c.Code != CodeRoomConfirmNeeded {
			return true
		}
		if c.Code == CodeParticipantOverlap {
			return true
		}
	}
	return false
}

var _ = strings.TrimSpace
