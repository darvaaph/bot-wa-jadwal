package bot

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// V1ClassInfo menyimpan metadata kelas v1
type V1ClassInfo struct {
	ID           int64
	Code         string
	Slug         string
	StudyProgram string
	CohortYear   int
	GroupLabel   string
	Status       string
}

// V1BotService menangani kueri operasional bot WhatsApp ke database target v1 (bot_v1.db)
type V1BotService struct {
	db *sql.DB
}

// NewV1BotService membuat instance layanan bot v1
func NewV1BotService(db *sql.DB) *V1BotService {
	return &V1BotService{db: db}
}

// FindClass mencari kelas v1 berdasarkan kode, slug, atau alias (misal: "d4-ti-2024-a", "D4-TI-2024-A", "D4-TI-1A")
func (s *V1BotService) FindClass(ctx context.Context, identifier string) (*V1ClassInfo, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database v1 belum siap")
	}

	clean := strings.ToLower(strings.TrimSpace(identifier))
	if clean == "" {
		return nil, sql.ErrNoRows
	}

	// Coba cocokkan slug atau code langsung
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status
		FROM classes
		WHERE LOWER(slug) = ? OR LOWER(code) = ?
		LIMIT 1;
	`
	var c V1ClassInfo
	err := s.db.QueryRowContext(ctx, query, clean, clean).Scan(
		&c.ID, &c.Code, &c.Slug, &c.StudyProgram, &c.CohortYear, &c.GroupLabel, &c.Status,
	)
	if err == nil {
		return &c, nil
	}

	// Coba alias umum: format seperti "d4-ti-1a", "d4-ti-smt1-a", "smt 1 a" (Angkatan 2024 = Kelas 1)
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(clean, "-", ""), "_", ""), " ", "")
	if strings.Contains(normalized, "1a") {
		err = s.db.QueryRowContext(ctx, query, "d4-ti-2024-a", "d4-ti-2024-a").Scan(
			&c.ID, &c.Code, &c.Slug, &c.StudyProgram, &c.CohortYear, &c.GroupLabel, &c.Status,
		)
		if err == nil {
			return &c, nil
		}
	} else if strings.Contains(normalized, "1b") {
		err = s.db.QueryRowContext(ctx, query, "d4-ti-2024-b", "d4-ti-2024-b").Scan(
			&c.ID, &c.Code, &c.Slug, &c.StudyProgram, &c.CohortYear, &c.GroupLabel, &c.Status,
		)
		if err == nil {
			return &c, nil
		}
	}

	return nil, sql.ErrNoRows
}

// BindChannel mendaftarkan atau memperbarui kanal WhatsApp (JID) grup kelas ke tabel whatsapp_channels
func (s *V1BotService) BindChannel(ctx context.Context, classID int64, jid, groupName string) error {
	if s.db == nil {
		return fmt.Errorf("database v1 belum siap")
	}

	cleanJID := strings.TrimSpace(jid)
	if cleanJID == "" {
		return fmt.Errorf("JID kosong")
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status, verified_at)
		VALUES (?, ?, 'GROUP', ?, 'ACTIVE', CURRENT_TIMESTAMP)
		ON CONFLICT(jid) DO UPDATE SET
			class_id = excluded.class_id,
			display_name = excluded.display_name,
			status = 'ACTIVE',
			verified_at = CURRENT_TIMESTAMP;
	`, classID, cleanJID, groupName)

	return err
}

// GetSchedule menghasilkan teks balasan jadwal kuliah harian dari database target v1
func (s *V1BotService) GetSchedule(ctx context.Context, classID int64, targetDate time.Time) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	className, classSlug, semTerm, semYear, items, err := s.ResolveDailyItems(ctx, classID, targetDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Sprintf("⚠️ *JADWAL KULIAH %s*\n──────────\nBelum ada semester aktif yang dikonfigurasi untuk kelas ini.", className), nil
		}
		return "", err
	}

	weekday := int(targetDate.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	dayNames := map[int]string{
		1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu",
	}
	dayName := dayNames[weekday]
	dateStr := targetDate.Format("02 Jan 2006")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📅 *JADWAL KULIAH %s*\n", className))
	b.WriteString(fmt.Sprintf("📌 *Hari:* %s, %s (Semester %s %s)\n", dayName, dateStr, semTerm, semYear))
	b.WriteString("──────────\n")

	if len(items) == 0 {
		b.WriteString(fmt.Sprintf("🎉 *Tidak ada perkuliahan pada %s, %s! Libur/istirahat.*", dayName, dateStr))
		return b.String(), nil
	}

	for i, it := range items {
		statusLabel := ""
		switch it.Status {
		case "CANCELLED":
			statusLabel = " [🔴 DIBATALKAN]"
		case "REPLACEMENT":
			statusLabel = " [🔄 KULIAH PENGGANTI]"
		case "EXTRA":
			statusLabel = " [➕ KULIAH TAMBAHAN]"
		}

		b.WriteString(fmt.Sprintf("%d. *%s*%s\n", i+1, it.Course, statusLabel))
		b.WriteString(fmt.Sprintf("   ⏰ *Waktu:* %s - %s WIB\n", it.StartTime, it.EndTime))
		b.WriteString(fmt.Sprintf("   🏢 *Ruang:* %s\n", it.Room))
		if it.Lecturers != "" && it.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("   👨‍🏫 *Dosen:* %s\n", it.Lecturers))
		}
		if it.Note != "" {
			b.WriteString(fmt.Sprintf("   💬 *Keterangan:* %s\n", it.Note))
		}
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("_Akses Portal Kelas: /portal/%s/summary_", classSlug))

	return b.String(), nil
}

// ScheduleItem merepresentasikan satu sesi perkuliahan dari pola reguler maupun kejadian
type ScheduleItem struct {
	PatternID int64
	Course    string
	StartTime string
	EndTime   string
	Room      string
	Lecturers string
	Status    string // REGULAR, REPLACEMENT, CANCELLED, EXTRA
	Note      string
}

// ResolveDailyItems mengambil dan menggabungkan sesi perkuliahan aktif untuk tanggal tertentu
func (s *V1BotService) ResolveDailyItems(ctx context.Context, classID int64, targetDate time.Time) (className, classSlug, semTerm, semYear string, items []ScheduleItem, err error) {
	if s.db == nil {
		return "", "", "", "", nil, fmt.Errorf("database v1 belum siap")
	}

	err = s.db.QueryRowContext(ctx, `SELECT code, slug FROM classes WHERE id = ?;`, classID).Scan(&className, &classSlug)
	if err != nil {
		return "", "", "", "", nil, err
	}

	var semID int64
	err = s.db.QueryRowContext(ctx, `
		SELECT id, academic_year, term
		FROM semesters
		WHERE class_id = ? AND status = 'ACTIVE'
		LIMIT 1;
	`, classID).Scan(&semID, &semYear, &semTerm)
	if err != nil {
		return className, classSlug, "", "", nil, err
	}

	weekday := int(targetDate.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	patternQuery := `
		SELECT sp.id, co.display_name, c.name, sp.start_time, sp.end_time,
		       COALESCE(r.code, '-') AS room_code,
		       COALESCE(GROUP_CONCAT(l.full_name, ', '), '-') AS lecturers
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = ?
		  AND sp.day_of_week = ?
		  AND sp.status = 'ACTIVE'
		GROUP BY sp.id
		ORDER BY sp.start_time ASC;
	`
	rows, err := s.db.QueryContext(ctx, patternQuery, semID, weekday)
	if err != nil {
		return className, classSlug, semTerm, semYear, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it ScheduleItem
		var offName, cName string
		if err := rows.Scan(&it.PatternID, &offName, &cName, &it.StartTime, &it.EndTime, &it.Room, &it.Lecturers); err == nil {
			it.Course = offName
			if it.Course == "" {
				it.Course = cName
			}
			it.Status = "REGULAR"
			items = append(items, it)
		}
	}

	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location()).Format(time.RFC3339)
	dayEnd := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 23, 59, 59, 0, targetDate.Location()).Format(time.RFC3339)

	eventQuery := `
		SELECT te.id, te.event_kind, te.starts_at, te.ends_at, COALESCE(te.reason, ''),
		       COALESCE(te.origin_schedule_pattern_id, 0),
		       co.display_name,
		       COALESCE(r.code, '-') AS room_code
		FROM teaching_events te
		JOIN teaching_event_offerings teo ON te.id = teo.teaching_event_id AND teo.participation_role = 'OWNER'
		JOIN course_offerings co ON teo.course_offering_id = co.id
		LEFT JOIN rooms r ON te.room_id = r.id
		WHERE co.semester_id = ?
		  AND te.lifecycle_status = 'PUBLISHED'
		  AND te.starts_at >= ? AND te.starts_at <= ?
		ORDER BY te.starts_at ASC;
	`
	evRows, err := s.db.QueryContext(ctx, eventQuery, semID, dayStart, dayEnd)
	if err == nil {
		defer evRows.Close()
		for evRows.Next() {
			var evID, originPatternID int64
			var evKind, startsAtStr, endsAtStr, reason, offName, roomCode string
			if err := evRows.Scan(&evID, &evKind, &startsAtStr, &endsAtStr, &reason, &originPatternID, &offName, &roomCode); err == nil {
				startTime := startsAtStr
				endTime := endsAtStr
				if tStart, errT := time.Parse(time.RFC3339, startsAtStr); errT == nil {
					startTime = tStart.Format("15:04")
				}
				if tEnd, errT := time.Parse(time.RFC3339, endsAtStr); errT == nil {
					endTime = tEnd.Format("15:04")
				}

				if evKind == "SESSION_CANCELLED" && originPatternID > 0 {
					for idx := range items {
						if items[idx].PatternID == originPatternID {
							items[idx].Status = "CANCELLED"
							items[idx].Note = reason
						}
					}
				} else {
					statusKind := "REPLACEMENT"
					if evKind == "EXTRA" {
						statusKind = "EXTRA"
					}
					items = append(items, ScheduleItem{
						Course:    offName,
						StartTime: startTime,
						EndTime:   endTime,
						Room:      roomCode,
						Status:    statusKind,
						Note:      reason,
					})
				}
			}
		}
	}

	return className, classSlug, semTerm, semYear, items, nil
}

// GetTasks menghasilkan daftar tugas aktif dari database target v1
func (s *V1BotService) GetTasks(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var className, classSlug string
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug FROM classes WHERE id = ?;`, classID).Scan(&className, &classSlug)

	query := `
		SELECT t.id, t.title, COALESCE(t.instructions, ''), t.deadline_at,
		       co.display_name, COALESCE(t.submission_url, '')
		FROM tasks t
		JOIN course_offerings co ON t.course_offering_id = co.id
		WHERE co.semester_id = (SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1)
		  AND t.publication_status = 'PUBLISHED'
		  AND t.deleted_at IS NULL
		  AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC;
	`
	rows, err := s.db.QueryContext(ctx, query, classID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type taskItem struct {
		id           int64
		title        string
		instructions string
		deadline     time.Time
		course       string
		url          string
	}

	var tasks []taskItem
	for rows.Next() {
		var it taskItem
		var dlStr string
		if err := rows.Scan(&it.id, &it.title, &it.instructions, &dlStr, &it.course, &it.url); err == nil {
			if tDl, errP := time.Parse(time.RFC3339, dlStr); errP == nil {
				it.deadline = tDl
			}
			tasks = append(tasks, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📝 *DAFTAR TUGAS KULIAH %s*\n", className))
	b.WriteString("──────────\n")

	if len(tasks) == 0 {
		b.WriteString("🎉 *Tidak ada tugas aktif saat ini. Semua tugas telah selesai!*")
		return b.String(), nil
	}

	now := time.Now()
	for i, t := range tasks {
		deadlineFormatted := t.deadline.Format("02 Jan 2006 15:04 WIB")
		timeStatus := ""
		if t.deadline.Before(now) {
			timeStatus = " ⚠️ [TERLEWAT]"
		} else if t.deadline.Sub(now) < 24*time.Hour {
			timeStatus = " 🚨 [SEGERA BERAKHIR]"
		}

		b.WriteString(fmt.Sprintf("%d. *[#%d - %s]* %s%s\n", i+1, t.id, t.course, t.title, timeStatus))
		b.WriteString(fmt.Sprintf("   ⏰ *Tenggat:* %s\n", deadlineFormatted))
		if t.instructions != "" {
			b.WriteString(fmt.Sprintf("   📋 *Instruksi:* %s\n", t.instructions))
		}
		if t.url != "" {
			b.WriteString(fmt.Sprintf("   🔗 *Pengumpulan:* %s\n", t.url))
		}
		if i < len(tasks)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("_Selesaikan tugas via portal atau ketik `!tugas selesai <id>`_"))

	return b.String(), nil
}

// CompleteTask menandai tugas selesai pada database v1
func (s *V1BotService) CompleteTask(ctx context.Context, classID, taskID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	res, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET completed_at = CURRENT_TIMESTAMP
		WHERE id = ? AND course_offering_id IN (
			SELECT co.id FROM course_offerings co
			JOIN semesters sem ON co.semester_id = sem.id
			WHERE sem.class_id = ?
		);
	`, taskID, classID)

	if err != nil {
		return "", err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return "⚠️ Tugas tidak ditemukan atau bukan milik kelas ini.", nil
	}

	return fmt.Sprintf("✅ *TUGAS SELESAI!*\nTugas ID #%d berhasil ditandai sebagai selesai.", taskID), nil
}

// GetLinks menghasilkan daftar materi dan tautan penting dari database v1
func (s *V1BotService) GetLinks(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var className string
	_ = s.db.QueryRowContext(ctx, `SELECT code FROM classes WHERE id = ?;`, classID).Scan(&className)

	query := `
		SELECT m.title, m.material_type, m.url, COALESCE(m.description, ''),
		       COALESCE(co.display_name, 'Umum')
		FROM materials m
		LEFT JOIN course_offerings co ON m.course_offering_id = co.id
		WHERE m.class_id = ? AND m.status = 'ACTIVE'
		ORDER BY m.created_at DESC;
	`
	rows, err := s.db.QueryContext(ctx, query, classID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type linkItem struct {
		title       string
		matType     string
		url         string
		description string
		offering    string
	}

	var links []linkItem
	for rows.Next() {
		var it linkItem
		if err := rows.Scan(&it.title, &it.matType, &it.url, &it.description, &it.offering); err == nil {
			links = append(links, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔗 *TAUTAN & MATERI PENTING %s*\n", className))
	b.WriteString("──────────\n")

	if len(links) == 0 {
		b.WriteString("Belum ada tautan atau materi yang dibagikan untuk kelas ini.")
		return b.String(), nil
	}

	for i, l := range links {
		icon := "🔗"
		switch l.matType {
		case "DOCUMENT":
			icon = "📄"
		case "MEETING":
			icon = "🎥"
		case "REPOSITORY":
			icon = "💻"
		case "PORTAL":
			icon = "🌐"
		}

		b.WriteString(fmt.Sprintf("%d. %s *%s* (%s)\n", i+1, icon, l.title, l.offering))
		b.WriteString(fmt.Sprintf("   👉 %s\n", l.url))
		if l.description != "" {
			b.WriteString(fmt.Sprintf("   💬 %s\n", l.description))
		}
		if i < len(links)-1 {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}

// GetWeeklySchedule menghasilkan jadwal lengkap seminggu (Senin - Jumat/Minggu) dari database target v1
func (s *V1BotService) GetWeeklySchedule(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var className, classSlug string
	err := s.db.QueryRowContext(ctx, `SELECT code, slug FROM classes WHERE id = ?;`, classID).Scan(&className, &classSlug)
	if err != nil {
		return "", err
	}

	var semID int64
	var semYear, semTerm string
	err = s.db.QueryRowContext(ctx, `
		SELECT id, academic_year, term
		FROM semesters
		WHERE class_id = ? AND status = 'ACTIVE'
		LIMIT 1;
	`, classID).Scan(&semID, &semYear, &semTerm)
	if err != nil {
		return "", err
	}

	query := `
		SELECT sp.day_of_week, sp.start_time, sp.end_time,
		       co.display_name, c.name,
		       COALESCE(r.code, '-') AS room_code,
		       COALESCE(GROUP_CONCAT(l.full_name, ', '), '-') AS lecturers
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = ? AND sp.status = 'ACTIVE'
		GROUP BY sp.id
		ORDER BY sp.day_of_week ASC, sp.start_time ASC;
	`
	rows, err := s.db.QueryContext(ctx, query, semID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dayNames := map[int]string{
		1: "SENIN", 2: "SELASA", 3: "RABU", 4: "KAMIS", 5: "JUMAT", 6: "SABTU", 7: "MINGGU",
	}

	type weeklyItem struct {
		day       int
		startTime string
		endTime   string
		course    string
		room      string
		lecturers string
	}
	byDay := make(map[int][]weeklyItem)
	var orderedDays []int

	for rows.Next() {
		var it weeklyItem
		var offName, cName string
		if err := rows.Scan(&it.day, &it.startTime, &it.endTime, &offName, &cName, &it.room, &it.lecturers); err == nil {
			it.course = offName
			if it.course == "" {
				it.course = cName
			}
			if len(byDay[it.day]) == 0 {
				orderedDays = append(orderedDays, it.day)
			}
			byDay[it.day] = append(byDay[it.day], it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📅 *JADWAL KULIAH LENGKAP SEMINGGU %s*\n", className))
	b.WriteString(fmt.Sprintf("📌 *Semester:* %s %s\n", semTerm, semYear))
	b.WriteString("──────────\n")

	if len(orderedDays) == 0 {
		b.WriteString("Belum ada jadwal perkuliahan reguler yang terdaftar.")
		return b.String(), nil
	}

	for _, d := range orderedDays {
		b.WriteString(fmt.Sprintf("\n📌 *%s*\n", dayNames[d]))
		for i, it := range byDay[d] {
			b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, it.course))
			b.WriteString(fmt.Sprintf("   ⏰ %s - %s WIB | 🏢 Ruang %s\n", it.startTime, it.endTime, it.room))
			if it.lecturers != "" && it.lecturers != "-" {
				b.WriteString(fmt.Sprintf("   👨‍🏫 %s\n", it.lecturers))
			}
		}
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("_Akses Portal Kelas: /portal/%s/summary_", classSlug))

	return b.String(), nil
}

// GetNextClass mencari mata kuliah yang sedang berlangsung atau perkuliahan berikutnya hari ini
func (s *V1BotService) GetNextClass(ctx context.Context, classID int64, now time.Time) (string, error) {
	className, classSlug, _, _, items, err := s.ResolveDailyItems(ctx, classID, now)
	if err != nil {
		return "", err
	}

	nowStr := now.Format("15:04")
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	dayNames := map[int]string{
		1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu",
	}
	dayName := dayNames[weekday]

	if len(items) == 0 {
		return fmt.Sprintf("🎉 *Tidak ada jadwal kuliah hari ini (%s)! Libur/istirahat.*", dayName), nil
	}

	var ongoing *ScheduleItem
	var nextUp *ScheduleItem

	for idx := range items {
		it := &items[idx]
		if it.Status == "CANCELLED" {
			continue
		}
		if it.StartTime <= nowStr && nowStr <= it.EndTime {
			ongoing = it
			break
		}
		if it.StartTime > nowStr {
			if nextUp == nil || it.StartTime < nextUp.StartTime {
				nextUp = it
			}
		}
	}

	var b strings.Builder
	if ongoing != nil {
		b.WriteString(fmt.Sprintf("🟢 *KULIAH SEDANG BERLANGSUNG (%s)*\n", className))
		b.WriteString("──────────\n")
		b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", ongoing.Course))
		b.WriteString(fmt.Sprintf("⏰ *Waktu:* %s - %s WIB (Sekarang: %s)\n", ongoing.StartTime, ongoing.EndTime, nowStr))
		b.WriteString(fmt.Sprintf("🏢 *Ruangan:* %s\n", ongoing.Room))
		if ongoing.Lecturers != "" && ongoing.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("👨‍🏫 *Dosen:* %s\n", ongoing.Lecturers))
		}
		if ongoing.Note != "" {
			b.WriteString(fmt.Sprintf("💬 *Catatan:* %s\n", ongoing.Note))
		}
	} else if nextUp != nil {
		b.WriteString(fmt.Sprintf("⏳ *KULIAH BERIKUTNYA HARI INI (%s)*\n", className))
		b.WriteString("──────────\n")
		b.WriteString(fmt.Sprintf("📚 *Mata Kuliah:* %s\n", nextUp.Course))
		b.WriteString(fmt.Sprintf("⏰ *Waktu:* %s - %s WIB (Sekarang: %s)\n", nextUp.StartTime, nextUp.EndTime, nowStr))
		b.WriteString(fmt.Sprintf("🏢 *Ruangan:* %s\n", nextUp.Room))
		if nextUp.Lecturers != "" && nextUp.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("👨‍🏫 *Dosen:* %s\n", nextUp.Lecturers))
		}
		if nextUp.Note != "" {
			b.WriteString(fmt.Sprintf("💬 *Catatan:* %s\n", nextUp.Note))
		}
	} else {
		b.WriteString(fmt.Sprintf("🎉 *Seluruh perkuliahan kelas %s hari ini telah selesai!*\n", className))
		b.WriteString("──────────\n")
		b.WriteString("Selamat beristirahat atau belajar mandiri untuk persiapan esok hari.")
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("_Akses Portal Kelas: /portal/%s/summary_", classSlug))

	return b.String(), nil
}

// HandleScheduleCommand memproses berbagai variasi query perintah jadwal
func (s *V1BotService) HandleScheduleCommand(ctx context.Context, classID int64, rawCmd string, now time.Time) (string, error) {
	clean := strings.TrimSpace(rawCmd)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		clean = strings.TrimSpace(clean[1:])
	}
	lower := strings.ToLower(clean)

	if lower == "next" || lower == "sekarang" || lower == "kuliah" || lower == "ongoing" || lower == "kuliah berikutnya" {
		return s.GetNextClass(ctx, classID, now)
	}

	if lower == "seminggu" || lower == "jadwal seminggu" || lower == "jadwal semua" || lower == "jadwal all" || lower == "semua" || lower == "all" || lower == "full" {
		return s.GetWeeklySchedule(ctx, classID)
	}

	targetDate := now
	arg := lower
	if strings.HasPrefix(lower, "jadwal") {
		parts := strings.SplitN(clean, " ", 2)
		if len(parts) > 1 {
			arg = strings.ToLower(strings.TrimSpace(parts[1]))
		} else {
			arg = ""
		}
	}

	dayNames := map[string]int{
		"senin": 1, "monday": 1,
		"selasa": 2, "tuesday": 2,
		"rabu": 3, "wednesday": 3,
		"kamis": 4, "thursday": 4,
		"jumat": 5, "jum'at": 5, "friday": 5,
		"sabtu": 6, "saturday": 6,
		"minggu": 7, "sunday": 7,
	}

	switch arg {
	case "", "hari ini", "hariini", "today", "now":
		targetDate = now
	case "besok", "tomorrow":
		targetDate = now.AddDate(0, 0, 1)
	case "kemarin", "yesterday":
		targetDate = now.AddDate(0, 0, -1)
	default:
		if targetDay, ok := dayNames[arg]; ok {
			curW := int(now.Weekday())
			if curW == 0 {
				curW = 7
			}
			diff := targetDay - curW
			if diff < 0 {
				diff += 7
			}
			targetDate = now.AddDate(0, 0, diff)
		}
	}

	return s.GetSchedule(ctx, classID, targetDate)
}
