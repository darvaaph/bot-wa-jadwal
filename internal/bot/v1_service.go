package bot

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/maintenance"
	"bot-jadwal/internal/util"
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

	// Coba alias umum mahasiswa (cth: "1a", "2a", "3a", "4a", "smt1-a", "smt3-a", "d4-ti-2a", dll)
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(clean, "-", ""), "_", ""), " ", "")
	reAlias := regexp.MustCompile(`^(?:(d3|d4)?(?:ti)?)?(?:(smt|semester))?([1-8])([a-d])$`)
	if m := reAlias.FindStringSubmatch(normalized); len(m) == 5 {
		prodiPrefix := m[1]
		isExplicitSem := m[2] != ""
		rawDigit, _ := strconv.Atoi(m[3])
		targetGroup := strings.ToUpper(m[4])

		targetTingkat := rawDigit
		if isExplicitSem {
			targetTingkat = (rawDigit + 1) / 2
		}

		prodiPattern := "%"
		codePattern := "%"
		if prodiPrefix == "d3" {
			prodiPattern = "%D3%"
			codePattern = "D3%"
		} else if prodiPrefix == "d4" {
			prodiPattern = "%D4%"
			codePattern = "D4%"
		}

		aliasQuery := `
			SELECT c.id, c.code, c.slug, c.study_program, c.cohort_year, c.group_label, c.status
			FROM classes c
			JOIN semesters s ON c.id = s.class_id AND s.status = 'ACTIVE'
			WHERE c.status = 'ACTIVE'
			  AND UPPER(c.group_label) = ?
			  AND (CAST(SUBSTR(s.academic_year, 1, 4) AS INTEGER) - c.cohort_year + 1) = ?
			  AND (c.study_program LIKE ? OR c.code LIKE ?)
			LIMIT 1;
		`
		err = s.db.QueryRowContext(ctx, aliasQuery, targetGroup, targetTingkat, prodiPattern, codePattern).Scan(
			&c.ID, &c.Code, &c.Slug, &c.StudyProgram, &c.CohortYear, &c.GroupLabel, &c.Status,
		)
		if err == nil {
			return &c, nil
		}
	}

	return nil, sql.ErrNoRows
}

// formatClassDisplayName menyusun label nama kelas yang ramah dibaca mahasiswa.
// Contoh: D4-TI-2025-A -> "D4 TEKNIK INFORMATIKA 2A (Angkatan 2025)"
func formatClassDisplayName(code, studyProgram string, cohortYear int, groupLabel string, academicYear string) string {
	if code == "" {
		return ""
	}
	prodi := strings.ToUpper(strings.TrimSpace(studyProgram))
	if prodi == "" {
		if strings.HasPrefix(code, "D4") {
			prodi = "D4 TEKNIK INFORMATIKA"
		} else if strings.HasPrefix(code, "D3") {
			prodi = "D3 TEKNIK INFORMATIKA"
		} else {
			return code
		}
	} else {
		if !strings.HasPrefix(prodi, "D4") && !strings.HasPrefix(prodi, "D3") {
			if strings.HasPrefix(code, "D4") {
				prodi = "D4 " + prodi
			} else if strings.HasPrefix(code, "D3") {
				prodi = "D3 " + prodi
			}
		}
	}

	group := strings.ToUpper(strings.TrimSpace(groupLabel))
	if group == "" {
		parts := strings.Split(code, "-")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if len(last) == 1 {
				group = strings.ToUpper(last)
			}
		}
	}

	var startYear int
	if academicYear != "" {
		parts := strings.Split(academicYear, "/")
		if len(parts) > 0 {
			startYear, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		}
	}
	if startYear == 0 {
		startYear = time.Now().Year()
	}

	if cohortYear > 0 && group != "" {
		tingkat := startYear - cohortYear + 1
		if tingkat >= 1 && tingkat <= 5 {
			return fmt.Sprintf("%s %d%s (Angkatan %d)", prodi, tingkat, group, cohortYear)
		}
		return fmt.Sprintf("%s Kelas %s (Angkatan %d)", prodi, group, cohortYear)
	}

	return code
}

// BindChannel mendaftarkan atau memperbarui kanal WhatsApp (JID) grup kelas ke tabel whatsapp_channels.
// Liang pembatas: JID yang sudah tertaut ke kelas LAIN tidak dipindahkan
// (kembalikan galat; pemindahan = lepas + taut ulang via dashboard beraudit).
func (s *V1BotService) BindChannel(ctx context.Context, classID int64, jid, groupName string) error {
	release, allowed := maintenance.EnterMutation()
	if !allowed {
		return fmt.Errorf("pemulihan data sedang berlangsung; coba lagi setelah selesai")
	}
	defer release()
	if s.db == nil {
		return fmt.Errorf("database v1 belum siap")
	}

	cleanJID := strings.TrimSpace(jid)
	if cleanJID == "" {
		return fmt.Errorf("JID kosong")
	}

	var existingClass int64
	err := s.db.QueryRowContext(ctx, `SELECT class_id FROM whatsapp_channels WHERE jid = ?;`, cleanJID).Scan(&existingClass)
	if err == nil {
		if existingClass != classID {
			return fmt.Errorf("JID sudah tertaut ke kelas lain")
		}
		_, err = s.db.ExecContext(ctx, `
			UPDATE whatsapp_channels
			SET display_name = ?, status = 'ACTIVE', verified_at = CURRENT_TIMESTAMP
			WHERE jid = ?;
		`, groupName, cleanJID)
		return err
	} else if err != sql.ErrNoRows {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status, verified_at)
		VALUES (?, ?, 'GROUP', ?, 'ACTIVE', CURRENT_TIMESTAMP);
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
			return fmt.Sprintf("*JADWAL KULIAH %s*\n──────────\nBelum ada semester aktif yang dikonfigurasi untuk kelas ini.", className), nil
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
	b.WriteString(fmt.Sprintf("*JADWAL KULIAH %s*\n", className))
	b.WriteString(fmt.Sprintf("Hari: %s, %s (Semester %s %s)\n", dayName, dateStr, semTerm, semYear))
	b.WriteString("──────────\n")

	if len(items) == 0 {
		b.WriteString(fmt.Sprintf("Tidak ada perkuliahan pada %s, %s. (Libur/istirahat)", dayName, dateStr))
		return b.String(), nil
	}

	for i, it := range items {
		statusLabel := ""
		switch it.Status {
		case "CANCELLED":
			statusLabel = " [DIBATALKAN]"
		case "REPLACEMENT":
			statusLabel = " [KULIAH PENGGANTI]"
		case "EXTRA":
			statusLabel = " [KULIAH TAMBAHAN]"
		}

		b.WriteString(fmt.Sprintf("%d. *%s*%s\n", i+1, it.Course, statusLabel))
		b.WriteString(fmt.Sprintf("   - Waktu: %s - %s WIB\n", it.StartTime, it.EndTime))
		b.WriteString(fmt.Sprintf("   - Ruang: %s\n", it.Room))
		if it.Lecturers != "" && it.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("   - Dosen: %s\n", it.Lecturers))
		}
		if it.Note != "" {
			b.WriteString(fmt.Sprintf("   - Keterangan: %s\n", it.Note))
		}
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("Portal Kelas: %s/c/%s", util.PublicBaseURL(), classSlug))

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

	var code, slug, studyProgram, groupLabel string
	var cohortYear int
	err = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProgram, &cohortYear, &groupLabel)
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
		className = formatClassDisplayName(code, studyProgram, cohortYear, groupLabel, "")
		return className, slug, "", "", nil, err
	}

	className = formatClassDisplayName(code, studyProgram, cohortYear, groupLabel, semYear)
	classSlug = slug

	weekday := int(targetDate.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	date := targetDate.Format("2006-01-02")

	patternQuery := `
		SELECT sp.id, co.display_name, c.name, sp.start_time, sp.end_time,
		       COALESCE(r.code, '-') AS room_code,
		       COALESCE(GROUP_CONCAT(l.full_name, ', '), '-') AS lecturers
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id AND ol.superseded_at IS NULL
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = ?
		  AND co.status = 'ACTIVE'
		  AND sp.day_of_week = ?
		  AND sp.status = 'ACTIVE'
		  AND sp.effective_from <= ?
		  AND (sp.effective_until IS NULL OR sp.effective_until >= ?)
		GROUP BY sp.id
		ORDER BY sp.start_time ASC;
	`
	rows, err := s.db.QueryContext(ctx, patternQuery, semID, weekday, date, date)
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

	dayStart := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)

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
		  AND datetime(te.starts_at) >= datetime(?) AND datetime(te.starts_at) < datetime(?)
		ORDER BY te.starts_at ASC;
	`
	evRows, err := s.db.QueryContext(ctx, eventQuery, semID, dayStart.UTC().Format(time.RFC3339), dayEnd.UTC().Format(time.RFC3339))
	if err == nil {
		defer evRows.Close()
		for evRows.Next() {
			var evID, originPatternID int64
			var evKind, startsAtStr, endsAtStr, reason, offName, roomCode string
			if err := evRows.Scan(&evID, &evKind, &startsAtStr, &endsAtStr, &reason, &originPatternID, &offName, &roomCode); err == nil {
				startTime := startsAtStr
				endTime := endsAtStr
				if tStart, errT := time.Parse(time.RFC3339Nano, startsAtStr); errT == nil {
					startTime = tStart.In(targetDate.Location()).Format("15:04")
				}
				if tEnd, errT := time.Parse(time.RFC3339Nano, endsAtStr); errT == nil {
					endTime = tEnd.In(targetDate.Location()).Format("15:04")
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

// formatClassShort menyusun label nama kelas yang ringkas untuk pesan WhatsApp (contoh: "D4-TI 2A").
func formatClassShort(code string, cohortYear int, groupLabel string, academicYear string) string {
	if code == "" {
		return ""
	}
	prefix := "TI"
	if strings.HasPrefix(code, "D4") {
		prefix = "D4-TI"
	} else if strings.HasPrefix(code, "D3") {
		prefix = "D3-TI"
	}

	group := strings.ToUpper(strings.TrimSpace(groupLabel))
	if group == "" {
		parts := strings.Split(code, "-")
		if len(parts) > 0 {
			group = strings.ToUpper(parts[len(parts)-1])
		}
	}

	var startYear int
	if academicYear != "" {
		parts := strings.Split(academicYear, "/")
		if len(parts) > 0 {
			startYear, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
		}
	}
	if startYear == 0 {
		startYear = time.Now().Year()
	}

	if cohortYear > 0 && group != "" {
		tingkat := startYear - cohortYear + 1
		if tingkat >= 1 && tingkat <= 5 {
			return fmt.Sprintf("%s %d%s", prefix, tingkat, group)
		}
		return fmt.Sprintf("%s %s", prefix, group)
	}

	return code
}

// GetTasks menghasilkan daftar tugas aktif dari database target v1
func (s *V1BotService) GetTasks(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	// Lokasi zona waktu kelas (default Asia/Jakarta / WIB)
	var tzStr string
	_ = s.db.QueryRowContext(ctx, `SELECT timezone FROM class_settings WHERE class_id = ?;`, classID).Scan(&tzStr)
	loc := time.FixedZone("WIB", 7*3600)
	if tzStr != "" {
		if l, err := time.LoadLocation(tzStr); err == nil {
			loc = l
		}
	}

	query := `
		SELECT t.id, t.title, COALESCE(t.instructions, ''), t.deadline_at,
		       co.display_name, COALESCE(t.submission_url, ''), COALESCE(t.submission_text, '')
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
		id             int64
		title          string
		instructions   string
		deadline       time.Time
		course         string
		url            string
		submissionText string
	}

	var tasks []taskItem
	for rows.Next() {
		var it taskItem
		var dlStr string
		if err := rows.Scan(&it.id, &it.title, &it.instructions, &dlStr, &it.course, &it.url, &it.submissionText); err == nil {
			if tDl, errP := time.Parse(time.RFC3339, dlStr); errP == nil {
				it.deadline = tDl.In(loc)
			}
			tasks = append(tasks, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*DAFTAR TUGAS • %s*\n", shortName))
	b.WriteString("────────────────────\n")

	if len(tasks) == 0 {
		b.WriteString("Tidak ada tugas aktif saat ini. Semua tugas telah selesai.\n")
		b.WriteString("────────────────────\n")
		b.WriteString("_Ketik !jadwal untuk melihat jadwal kuliah hari ini._")
		return b.String(), nil
	}

	now := time.Now().In(loc)
	indoDays := [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	indoMonths := [...]string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

	for i, t := range tasks {
		dayName := indoDays[t.deadline.Weekday()]
		mName := indoMonths[t.deadline.Month()]
		deadlineFormatted := fmt.Sprintf("%s, %02d %s %d · %02d:%02d WIB",
			dayName, t.deadline.Day(), mName, t.deadline.Year(), t.deadline.Hour(), t.deadline.Minute())

		diff := t.deadline.Sub(now)
		var statusStr string
		if diff < 0 {
			statusStr = "Lewat Tenggat"
		} else if t.deadline.Year() == now.Year() && t.deadline.YearDay() == now.YearDay() {
			hours := int(diff.Hours())
			if hours > 0 {
				statusStr = fmt.Sprintf("Hari ini (Sisa ~%d jam)", hours)
			} else {
				mins := int(diff.Minutes())
				if mins <= 0 {
					mins = 1
				}
				statusStr = fmt.Sprintf("Hari ini (Sisa ~%d menit)", mins)
			}
		} else {
			tomorrow := now.AddDate(0, 0, 1)
			if t.deadline.Year() == tomorrow.Year() && t.deadline.YearDay() == tomorrow.YearDay() {
				statusStr = "Besok (H-1)"
			} else {
				days := int(diff.Hours() / 24)
				if days <= 0 {
					days = 1
				}
				statusStr = fmt.Sprintf("%d hari lagi", days)
			}
		}

		pengumpulan := ""
		if t.submissionText != "" && t.url != "" && t.submissionText != t.url {
			pengumpulan = fmt.Sprintf("%s (%s)", t.submissionText, t.url)
		} else if t.url != "" {
			pengumpulan = t.url
		} else if t.submissionText != "" {
			pengumpulan = t.submissionText
		}

		b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, t.title))
		b.WriteString(fmt.Sprintf("   - ID Tugas: #%d\n", t.id))
		b.WriteString(fmt.Sprintf("   - Matkul  : %s\n", t.course))
		b.WriteString(fmt.Sprintf("   - Tenggat : %s\n", deadlineFormatted))
		b.WriteString(fmt.Sprintf("   - Status  : %s\n", statusStr))
		if pengumpulan != "" {
			b.WriteString(fmt.Sprintf("   - Pengumpulan: %s\n", pengumpulan))
		}
		if i < len(tasks)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n────────────────────\n")
	portalURL := util.PublicBaseURL() + "/"
	if slug != "" {
		portalURL = fmt.Sprintf("%s/c/%s", util.PublicBaseURL(), slug)
	}
	b.WriteString(fmt.Sprintf("Detail instruksi & lampiran: %s\n", portalURL))
	b.WriteString("_Ketik !jadwal untuk melihat jadwal kuliah hari ini._")

	return b.String(), nil
}

// CompleteTask menandai tugas selesai pada database v1
func (s *V1BotService) CompleteTask(ctx context.Context, classID, taskID int64) (string, error) {
	release, allowed := maintenance.EnterMutation()
	if !allowed {
		return "", fmt.Errorf("pemulihan data sedang berlangsung; coba lagi setelah selesai")
	}
	defer release()
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
		return "Tugas tidak ditemukan atau bukan milik kelas ini.", nil
	}

	return fmt.Sprintf("*TUGAS SELESAI*\nTugas ID #%d berhasil ditandai sebagai selesai.", taskID), nil
}

// GetLinks menghasilkan daftar materi dan tautan penting dari database v1
func (s *V1BotService) GetLinks(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	className := formatClassDisplayName(code, studyProg, cohortYear, groupLabel, semYear)

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
	b.WriteString(fmt.Sprintf("*TAUTAN & MATERI PENTING %s*\n", className))
	b.WriteString("──────────\n")

	if len(links) == 0 {
		b.WriteString("Belum ada tautan atau materi yang dibagikan untuk kelas ini.")
		return b.String(), nil
	}

	for i, l := range links {
		b.WriteString(fmt.Sprintf("%d. *%s* (%s)\n", i+1, l.title, l.offering))
		b.WriteString(fmt.Sprintf("   - Tautan: %s\n", l.url))
		if l.description != "" {
			b.WriteString(fmt.Sprintf("   - Keterangan: %s\n", l.description))
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

	var code, slug, studyProg, groupLabel string
	var cohortYear int
	err := s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
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

	className := formatClassDisplayName(code, studyProg, cohortYear, groupLabel, semYear)
	classSlug := slug

	query := `
		SELECT sp.day_of_week, sp.start_time, sp.end_time,
		       co.display_name, c.name,
		       COALESCE(r.code, '-') AS room_code,
		       COALESCE(GROUP_CONCAT(l.full_name, ', '), '-') AS lecturers
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id AND ol.superseded_at IS NULL
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = ? AND co.status = 'ACTIVE' AND sp.status = 'ACTIVE'
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
	b.WriteString(fmt.Sprintf("*JADWAL KULIAH LENGKAP SEMINGGU %s*\n", className))
	b.WriteString(fmt.Sprintf("Semester: %s %s\n", semTerm, semYear))
	b.WriteString("──────────\n")

	if len(orderedDays) == 0 {
		b.WriteString("Belum ada jadwal perkuliahan reguler yang terdaftar.")
		return b.String(), nil
	}

	for _, d := range orderedDays {
		b.WriteString(fmt.Sprintf("\n*%s*\n", dayNames[d]))
		for i, it := range byDay[d] {
			b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, it.course))
			b.WriteString(fmt.Sprintf("   - Waktu: %s - %s WIB\n", it.startTime, it.endTime))
			b.WriteString(fmt.Sprintf("   - Ruang: %s\n", it.room))
			if it.lecturers != "" && it.lecturers != "-" {
				b.WriteString(fmt.Sprintf("   - Dosen: %s\n", it.lecturers))
			}
		}
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("Portal Kelas: %s/c/%s", util.PublicBaseURL(), classSlug))

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
		return fmt.Sprintf("Tidak ada jadwal kuliah hari ini (%s). (Libur/istirahat)", dayName), nil
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
		b.WriteString(fmt.Sprintf("*KULIAH SEDANG BERLANGSUNG (%s)*\n", className))
		b.WriteString("──────────\n")
		b.WriteString(fmt.Sprintf("- Mata Kuliah: %s\n", ongoing.Course))
		b.WriteString(fmt.Sprintf("- Waktu: %s - %s WIB (Sekarang: %s)\n", ongoing.StartTime, ongoing.EndTime, nowStr))
		b.WriteString(fmt.Sprintf("- Ruangan: %s\n", ongoing.Room))
		if ongoing.Lecturers != "" && ongoing.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("- Dosen: %s\n", ongoing.Lecturers))
		}
		if ongoing.Note != "" {
			b.WriteString(fmt.Sprintf("- Catatan: %s\n", ongoing.Note))
		}
	} else if nextUp != nil {
		b.WriteString(fmt.Sprintf("*KULIAH BERIKUTNYA HARI INI (%s)*\n", className))
		b.WriteString("──────────\n")
		b.WriteString(fmt.Sprintf("- Mata Kuliah: %s\n", nextUp.Course))
		b.WriteString(fmt.Sprintf("- Waktu: %s - %s WIB (Sekarang: %s)\n", nextUp.StartTime, nextUp.EndTime, nowStr))
		b.WriteString(fmt.Sprintf("- Ruangan: %s\n", nextUp.Room))
		if nextUp.Lecturers != "" && nextUp.Lecturers != "-" {
			b.WriteString(fmt.Sprintf("- Dosen: %s\n", nextUp.Lecturers))
		}
		if nextUp.Note != "" {
			b.WriteString(fmt.Sprintf("- Catatan: %s\n", nextUp.Note))
		}
	} else {
		b.WriteString(fmt.Sprintf("Seluruh perkuliahan kelas %s hari ini telah selesai.\n", className))
		b.WriteString("──────────\n")
		b.WriteString("Selamat beristirahat atau belajar mandiri untuk persiapan esok hari.")
	}

	b.WriteString("\n──────────\n")
	b.WriteString(fmt.Sprintf("Portal Kelas: %s/c/%s", util.PublicBaseURL(), classSlug))

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

// GetMenu menghasilkan teks menu panduan perintah lengkap untuk kelas v1 (tanpa emotikon dekoratif)
func (s *V1BotService) GetMenu(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	now := time.Now()
	loc := time.FixedZone("WIB", 7*3600)
	var tzStr string
	_ = s.db.QueryRowContext(ctx, `SELECT timezone FROM class_settings WHERE class_id = ?;`, classID).Scan(&tzStr)
	if tzStr != "" {
		if l, err := time.LoadLocation(tzStr); err == nil {
			loc = l
		}
	}
	nowInLoc := now.In(loc)

	indoDays := [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	indoMonths := [...]string{"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}
	dayName := indoDays[nowInLoc.Weekday()]
	mName := indoMonths[nowInLoc.Month()]

	_, _, _, _, items, _ := s.ResolveDailyItems(ctx, classID, nowInLoc)
	statusHariIni := "Libur / Tidak ada kuliah"
	if len(items) > 0 {
		statusHariIni = fmt.Sprintf("%d Sesi Kuliah", len(items))
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*PANDUAN PERINTAH • %s*\n", shortName))
	b.WriteString(fmt.Sprintf("%s, %02d %s · %s\n", dayName, nowInLoc.Day(), mName, statusHariIni))
	b.WriteString("────────────────────\n\n")

	b.WriteString("*Jadwal Kuliah:*\n")
	b.WriteString("- !jadwal        : Jadwal kuliah hari ini\n")
	b.WriteString("- !next          : Kuliah yang sedang berlangsung / berikutnya\n")
	b.WriteString("- !besok         : Jadwal kuliah esok hari\n")
	b.WriteString("- !seminggu      : Jadwal lengkap seminggu (Senin - Jumat)\n")
	b.WriteString("- ![hari]        : Jadwal hari tertentu (contoh: !senin, !jumat)\n\n")

	b.WriteString("*Tugas & Materi:*\n")
	b.WriteString("- !tugas           : Daftar tugas aktif & tenggat waktu terdekat\n")
	b.WriteString("- !link / !materi  : Tautan & materi penting kelas (Drive, Zoom, RPS)\n\n")

	b.WriteString("*Pencarian & Informasi:*\n")
	b.WriteString("- !matkul        : Daftar mata kuliah semester aktif\n")
	b.WriteString("- !dosen [nama]  : Cari info / jadwal dosen (contoh: !dosen Budi)\n")
	b.WriteString("- !ruang [kode]  : Cari info / jadwal ruangan (contoh: !ruang LAB-1)\n")
	b.WriteString("- !cari [kata]   : Cari jadwal mata kuliah / dosen / ruang\n\n")

	b.WriteString("*Pengaturan & Akses:*\n")
	b.WriteString("- !portal          : Akses Web Portal Mahasiswa & Dashboard\n")
	b.WriteString("- !kanal           : Cek ID WhatsApp (JID) chat/grup ini\n")
	b.WriteString("- !daftarkelas     : Daftar seluruh kelas yang terdaftar\n")
	b.WriteString("- !setkelas [nama] : Pilih kelas aktif pribadi (contoh: !setkelas 2a)\n\n")

	b.WriteString("────────────────────\n")
	b.WriteString("Catatan: Penambahan tugas, perubahan jadwal, dan materi dikelola oleh PJ/KM melalui Web Dashboard.")

	return b.String(), nil
}

// GetKeywords menghasilkan panduan kata kunci lengkap untuk kelas v1
func (s *V1BotService) GetKeywords(ctx context.Context, classID int64) (string, error) {
	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*DAFTAR KATA KUNCI PERINTAH • %s*\n", shortName))
	b.WriteString("────────────────────\n\n")

	b.WriteString("1. Jadwal Harian:\n")
	b.WriteString("   - !jadwal / !hari ini / !today : Jadwal hari ini\n")
	b.WriteString("   - !besok / !tomorrow          : Jadwal esok hari\n")
	b.WriteString("   - !senin, !selasa, !rabu, !kamis, !jumat, !sabtu\n\n")

	b.WriteString("2. Jadwal Pekanan:\n")
	b.WriteString("   - !seminggu / !semua / !all   : Jadwal perkuliahan Senin - Jumat\n\n")

	b.WriteString("3. Kuliah Berlangsung / Berikutnya:\n")
	b.WriteString("   - !next / !sekarang / !kuliah : Cek sesi kuliah jam ini / berikutnya\n\n")

	b.WriteString("4. Tugas Kuliah:\n")
	b.WriteString("   - !tugas                      : Daftar tugas aktif kelas\n\n")

	b.WriteString("5. Materi & Tautan:\n")
	b.WriteString("   - !link / !tautan / !drive    : Tautan Google Drive, Zoom, RPS kelas\n\n")

	b.WriteString("6. Pencarian:\n")
	b.WriteString("   - !matkul                     : Daftar seluruh mata kuliah semester ini\n")
	b.WriteString("   - !dosen [nama/kode]          : Cari jadwal mengajar dosen\n")
	b.WriteString("   - !ruang [kode/nama]          : Cari pemakaian ruangan kuliah\n")
	b.WriteString("   - !cari [kata kunci]          : Pencarian umum mata kuliah/dosen/ruang\n\n")

	b.WriteString("7. Konfigurasi:\n")
	b.WriteString("   - !portal                     : Link Portal Mahasiswa & Dashboard\n")
	b.WriteString("   - !kanal                      : Cek ID JID WhatsApp chat ini\n")
	b.WriteString("   - !daftarkelas                : Daftar seluruh kelas di sistem\n")
	b.WriteString("   - !setkelas [nama]            : Tautkan kelas (khusus DM pribadi)\n\n")

	b.WriteString("────────────────────\n")
	b.WriteString("Ketik !menu untuk menu ringkas.")

	return b.String(), nil
}

// GetCourses menghasilkan daftar mata kuliah aktif semester ini
func (s *V1BotService) GetCourses(ctx context.Context, classID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	query := `
		SELECT co.display_name, c.code, c.name,
		       COALESCE(GROUP_CONCAT(DISTINCT l.full_name), '-') AS lecturers
		FROM course_offerings co
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id AND ol.superseded_at IS NULL
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = (SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1)
		  AND co.status = 'ACTIVE'
		GROUP BY co.id
		ORDER BY co.display_name ASC;
	`
	rows, err := s.db.QueryContext(ctx, query, classID)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	type courseItem struct {
		displayName string
		code        string
		name        string
		lecturers   string
	}
	var list []courseItem
	for rows.Next() {
		var it courseItem
		if err := rows.Scan(&it.displayName, &it.code, &it.name, &it.lecturers); err == nil {
			list = append(list, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*DAFTAR MATA KULIAH • %s*\n", shortName))
	b.WriteString("────────────────────\n")

	if len(list) == 0 {
		b.WriteString("Belum ada mata kuliah yang terdaftar untuk semester aktif ini.")
		return b.String(), nil
	}

	for i, it := range list {
		title := it.displayName
		if title == "" {
			title = it.name
		}
		b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, title))
		if it.code != "" && it.code != "-" {
			b.WriteString(fmt.Sprintf("   - Kode : %s\n", it.code))
		}
		if it.lecturers != "" && it.lecturers != "-" {
			b.WriteString(fmt.Sprintf("   - Dosen: %s\n", it.lecturers))
		}
		if i < len(list)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n────────────────────\n")
	b.WriteString(fmt.Sprintf("Portal Kelas: %s/c/%s", util.PublicBaseURL(), slug))

	return b.String(), nil
}

// SearchDosen mencari jadwal mengajar dosen pada kelas ini
func (s *V1BotService) SearchDosen(ctx context.Context, classID int64, query string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return "Sertakan nama atau kode dosen yang dicari.\nContoh: `!dosen Budi` atau `!dosen BDI`", nil
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	sqlQuery := `
		SELECT co.display_name, sp.day_of_week, sp.start_time, sp.end_time,
		       COALESCE(r.code, '-'), l.code, l.full_name
		FROM lecturers l
		JOIN offering_lecturers ol ON l.id = ol.lecturer_id AND ol.superseded_at IS NULL
		JOIN course_offerings co ON ol.course_offering_id = co.id
		JOIN schedule_patterns sp ON co.id = sp.course_offering_id AND sp.status = 'ACTIVE'
		LEFT JOIN rooms r ON sp.room_id = r.id
		WHERE co.semester_id = (SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1)
		  AND co.status = 'ACTIVE'
		  AND (LOWER(l.code) LIKE '%' || ? || '%' OR LOWER(l.full_name) LIKE '%' || ? || '%')
		ORDER BY sp.day_of_week ASC, sp.start_time ASC;
	`
	qLower := strings.ToLower(cleanQuery)
	rows, err := s.db.QueryContext(ctx, sqlQuery, classID, qLower, qLower)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dayNames := map[int]string{
		1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu",
	}

	type matchItem struct {
		course    string
		day       int
		startTime string
		endTime   string
		room      string
		dosenCode string
		dosenName string
	}
	var matches []matchItem
	for rows.Next() {
		var it matchItem
		if err := rows.Scan(&it.course, &it.day, &it.startTime, &it.endTime, &it.room, &it.dosenCode, &it.dosenName); err == nil {
			matches = append(matches, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*PENCARIAN DOSEN: \"%s\" • %s*\n", strings.ToUpper(cleanQuery), shortName))
	b.WriteString("────────────────────\n")

	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("Tidak ditemukan jadwal dosen dengan kata kunci \"%s\" di kelas ini.", cleanQuery))
		return b.String(), nil
	}

	for i, m := range matches {
		b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, m.course))
		b.WriteString(fmt.Sprintf("   - Dosen: %s (%s)\n", m.dosenName, m.dosenCode))
		b.WriteString(fmt.Sprintf("   - Waktu: %s, %s - %s WIB\n", dayNames[m.day], m.startTime, m.endTime))
		b.WriteString(fmt.Sprintf("   - Ruang: %s\n", m.room))
		if i < len(matches)-1 {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}

// SearchRuang mencari jadwal pemakaian ruangan pada kelas ini
func (s *V1BotService) SearchRuang(ctx context.Context, classID int64, query string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return "Sertakan kode atau nama ruangan yang dicari.\nContoh: `!ruang LAB-1` atau `!ruang D105`", nil
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	sqlQuery := `
		SELECT r.code, r.name, co.display_name, sp.day_of_week, sp.start_time, sp.end_time
		FROM rooms r
		JOIN schedule_patterns sp ON r.id = sp.room_id AND sp.status = 'ACTIVE'
		JOIN course_offerings co ON sp.course_offering_id = co.id
		WHERE co.semester_id = (SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1)
		  AND co.status = 'ACTIVE'
		  AND (LOWER(r.code) LIKE '%' || ? || '%' OR LOWER(r.name) LIKE '%' || ? || '%')
		ORDER BY sp.day_of_week ASC, sp.start_time ASC;
	`
	qLower := strings.ToLower(cleanQuery)
	rows, err := s.db.QueryContext(ctx, sqlQuery, classID, qLower, qLower)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dayNames := map[int]string{
		1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu",
	}

	type matchRoom struct {
		roomCode  string
		roomName  string
		course    string
		day       int
		startTime string
		endTime   string
	}
	var matches []matchRoom
	for rows.Next() {
		var it matchRoom
		if err := rows.Scan(&it.roomCode, &it.roomName, &it.course, &it.day, &it.startTime, &it.endTime); err == nil {
			matches = append(matches, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*PENCARIAN RUANGAN: \"%s\" • %s*\n", strings.ToUpper(cleanQuery), shortName))
	b.WriteString("────────────────────\n")

	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("Tidak ditemukan jadwal perkuliahan di ruangan \"%s\" untuk kelas ini.", cleanQuery))
		return b.String(), nil
	}

	for i, m := range matches {
		b.WriteString(fmt.Sprintf("%d. *%s* (Ruang %s)\n", i+1, m.course, m.roomCode))
		b.WriteString(fmt.Sprintf("   - Waktu: %s, %s - %s WIB\n", dayNames[m.day], m.startTime, m.endTime))
		if i < len(matches)-1 {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}

// SearchGlobal mencari jadwal berdasarkan kata kunci umum (mata kuliah, dosen, ruang)
func (s *V1BotService) SearchGlobal(ctx context.Context, classID int64, query string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("database v1 belum siap")
	}

	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return "Sertakan kata kunci pencarian.\nContoh: `!cari basis data` atau `!cari lab`", nil
	}

	var code, slug, studyProg, groupLabel, semYear string
	var cohortYear int
	_ = s.db.QueryRowContext(ctx, `SELECT code, slug, study_program, cohort_year, group_label FROM classes WHERE id = ?;`, classID).Scan(&code, &slug, &studyProg, &cohortYear, &groupLabel)
	_ = s.db.QueryRowContext(ctx, `SELECT academic_year FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semYear)
	shortName := formatClassShort(code, cohortYear, groupLabel, semYear)
	if shortName == "" {
		shortName = code
	}

	sqlQuery := `
		SELECT co.display_name, sp.day_of_week, sp.start_time, sp.end_time,
		       COALESCE(r.code, '-'), COALESCE(GROUP_CONCAT(DISTINCT l.full_name), '-')
		FROM schedule_patterns sp
		JOIN course_offerings co ON sp.course_offering_id = co.id
		JOIN courses c ON co.course_id = c.id
		LEFT JOIN rooms r ON sp.room_id = r.id
		LEFT JOIN offering_lecturers ol ON co.id = ol.course_offering_id AND ol.superseded_at IS NULL
		LEFT JOIN lecturers l ON ol.lecturer_id = l.id
		WHERE co.semester_id = (SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1)
		  AND co.status = 'ACTIVE'
		  AND (LOWER(co.display_name) LIKE '%' || ? || '%' OR LOWER(c.name) LIKE '%' || ? || '%' OR LOWER(c.code) LIKE '%' || ? || '%' OR LOWER(COALESCE(r.code, '')) LIKE '%' || ? || '%' OR LOWER(COALESCE(l.full_name, '')) LIKE '%' || ? || '%')
		GROUP BY sp.id
		ORDER BY sp.day_of_week ASC, sp.start_time ASC;
	`
	qLower := strings.ToLower(cleanQuery)
	rows, err := s.db.QueryContext(ctx, sqlQuery, classID, qLower, qLower, qLower, qLower, qLower)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	dayNames := map[int]string{
		1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu", 7: "Minggu",
	}

	type searchMatch struct {
		course    string
		day       int
		startTime string
		endTime   string
		room      string
		lecturers string
	}
	var matches []searchMatch
	for rows.Next() {
		var it searchMatch
		if err := rows.Scan(&it.course, &it.day, &it.startTime, &it.endTime, &it.room, &it.lecturers); err == nil {
			matches = append(matches, it)
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("*HASIL PENCARIAN: \"%s\" • %s*\n", strings.ToUpper(cleanQuery), shortName))
	b.WriteString("────────────────────\n")

	if len(matches) == 0 {
		b.WriteString(fmt.Sprintf("Tidak ditemukan jadwal perkuliahan dengan kata kunci \"%s\" untuk kelas ini.", cleanQuery))
		return b.String(), nil
	}

	for i, m := range matches {
		b.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, m.course))
		b.WriteString(fmt.Sprintf("   - Waktu: %s, %s - %s WIB\n", dayNames[m.day], m.startTime, m.endTime))
		b.WriteString(fmt.Sprintf("   - Ruang: %s\n", m.room))
		if m.lecturers != "" && m.lecturers != "-" {
			b.WriteString(fmt.Sprintf("   - Dosen: %s\n", m.lecturers))
		}
		if i < len(matches)-1 {
			b.WriteString("\n")
		}
	}

	return b.String(), nil
}
