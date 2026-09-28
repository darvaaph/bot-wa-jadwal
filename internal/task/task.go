package task

import (
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/util"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type TaskItem struct {
	ID         int
	ScopeJID   string
	ClassID    string
	IsGroup    bool
	Matkul     string
	Deskripsi  string
	Deadline   string
	DeadlineAt time.Time
	CreatedBy  string
	IsDone     bool
	CreatedAt  time.Time
}

type TaskManager struct {
	db *sql.DB
}

// NewTaskManager menginisialisasi tabel tasks pada instance *sql.DB bersama
func NewTaskManager(db *sql.DB) (*TaskManager, error) {
	if db == nil {
		return nil, fmt.Errorf("koneksi database tidak boleh nil")
	}

	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scope_jid TEXT NOT NULL,
		class_id TEXT DEFAULT '',
		is_group BOOLEAN NOT NULL,
		matkul TEXT NOT NULL,
		deskripsi TEXT NOT NULL,
		deadline TEXT NOT NULL,
		deadline_at DATETIME,
		created_by TEXT NOT NULL,
		is_done BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_tasks_scope ON tasks(scope_jid, is_done);
	CREATE INDEX IF NOT EXISTS idx_tasks_class ON tasks(class_id, is_done);
	`
	_, err := db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel tasks: %w", err)
	}

	// Migrasi aman jika kolom deadline_at atau class_id belum ada pada database lama
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN deadline_at DATETIME;`)
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN class_id TEXT DEFAULT '';`)
	_, _ = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tasks_class ON tasks(class_id, is_done);`)

	return &TaskManager{db: db}, nil
}

// NewTaskManagerWithPath membuat koneksi baru dari path file dan menginisialisasi TaskManager
func NewTaskManagerWithPath(dbPath string) (*TaskManager, error) {
	db, err := database.InitDB(dbPath)
	if err != nil {
		return nil, err
	}
	return NewTaskManager(db)
}

func (tm *TaskManager) Close() error {
	if tm.db != nil {
		return tm.db.Close()
	}
	return nil
}

// parseDeadline mengonversi teks tenggat waktu menjadi time.Time dan label yang rapi
func parseDeadline(rawInput string, refNow time.Time) (time.Time, string) {
	clean := strings.TrimSpace(rawInput)
	lower := strings.ToLower(clean)

	jamStr := "23:59"
	hasExplicitTime := false
	if match := util.TimeRe.FindString(clean); match != "" {
		jamStr = strings.ReplaceAll(match, ".", ":")
		hasExplicitTime = true
	}

	var jam, menit int
	fmt.Sscanf(jamStr, "%d:%d", &jam, &menit)

	loc := refNow.Location()

	if strings.Contains(lower, "hari ini") || strings.Contains(lower, "hariini") || strings.Contains(lower, "today") {
		target := time.Date(refNow.Year(), refNow.Month(), refNow.Day(), jam, menit, 0, 0, loc)
		return target, fmt.Sprintf("Hari Ini, %02d:%02d WIB", jam, menit)
	}

	if strings.Contains(lower, "besok") || strings.Contains(lower, "tomorrow") {
		t := refNow.Add(24 * time.Hour)
		target := time.Date(t.Year(), t.Month(), t.Day(), jam, menit, 0, 0, loc)
		return target, fmt.Sprintf("Besok (%s), %02d:%02d WIB", util.GetHariIndonesia(target), jam, menit)
	}

	for dayName, weekday := range util.NamaHariMap {
		if strings.Contains(lower, dayName) {
			daysAhead := int(weekday - refNow.Weekday())
			if daysAhead < 0 {
				daysAhead += 7
			} else if daysAhead == 0 {
				// Jika hari ini sama dengan hari target, cek apakah jam sudah lewat
				targetToday := time.Date(refNow.Year(), refNow.Month(), refNow.Day(), jam, menit, 0, 0, loc)
				if targetToday.Before(refNow) {
					daysAhead = 7
				}
			}

			targetDate := refNow.AddDate(0, 0, daysAhead)
			target := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), jam, menit, 0, 0, loc)
			return target, fmt.Sprintf("%s, %d %s %02d:%02d WIB",
				util.GetHariIndonesia(target), target.Day(), util.GetBulanIndonesia(target), jam, menit)
		}
	}

	layouts := []string{
		"02-01-2006 15:04", "02/01/2006 15:04", "2006-01-02 15:04",
		"02-01-2006", "02/01/2006", "2006-01-02",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, clean, loc); err == nil {
			if !strings.Contains(layout, "15:04") {
				t = time.Date(t.Year(), t.Month(), t.Day(), jam, menit, 0, 0, loc)
			}
			return t, fmt.Sprintf("%s, %d %s %02d:%02d WIB",
				util.GetHariIndonesia(t), t.Day(), util.GetBulanIndonesia(t), t.Hour(), t.Minute())
		}
	}

	if dateWord, ok := util.ParseIndonesianDateWord(lower, refNow, loc); ok {
		target := time.Date(dateWord.Year(), dateWord.Month(), dateWord.Day(), jam, menit, 0, 0, loc)
		return target, fmt.Sprintf("%s, %d %s %02d:%02d WIB",
			util.GetHariIndonesia(target), target.Day(), util.GetBulanIndonesia(target), jam, menit)
	}

	// Jika pengguna hanya memasukkan jam, artikan sebagai tenggat hari ini
	if hasExplicitTime {
		rem := util.TimeRe.ReplaceAllString(lower, "")
		for _, w := range []string{"jam", "pukul", "wib", "wita", "wit", "pagi", "siang", "sore", "malam"} {
			rem = strings.ReplaceAll(rem, w, "")
		}
		rem = strings.Trim(rem, " \t\r\n.,:-/")
		if rem == "" {
			target := time.Date(refNow.Year(), refNow.Month(), refNow.Day(), jam, menit, 0, 0, loc)
			return target, fmt.Sprintf("Hari Ini (%s), %02d:%02d WIB", util.GetHariIndonesia(target), jam, menit)
		}
	}

	// Fallback jika tidak terdeteksi: default 5 hari dari sekarang
	defaultTarget := refNow.AddDate(0, 0, 5)
	defaultTarget = time.Date(defaultTarget.Year(), defaultTarget.Month(), defaultTarget.Day(), jam, menit, 0, 0, loc)
	return defaultTarget, clean
}

// GetUrgencyBadge menghasilkan label status hitung mundur berdasarkan selisih waktu nyata
func GetUrgencyBadge(deadlineAt time.Time, now time.Time) string {
	if deadlineAt.IsZero() {
		return "⏳ *TUGAS AKTIF*"
	}

	diff := deadlineAt.Sub(now)
	if diff < 0 {
		return "⌛ *LEWAT TENGGAT*"
	}

	if deadlineAt.Year() == now.Year() && deadlineAt.YearDay() == now.YearDay() {
		hours := int(diff.Hours())
		mins := int(diff.Minutes()) % 60
		if hours > 0 {
			return fmt.Sprintf("🚨 *DEADLINE HARI INI* (Sisa ~%d jam)", hours)
		}
		return fmt.Sprintf("🚨 *DEADLINE HARI INI* (Sisa ~%d menit)", mins)
	}

	tomorrow := now.Add(24 * time.Hour)
	if deadlineAt.Year() == tomorrow.Year() && deadlineAt.YearDay() == tomorrow.YearDay() {
		return "⚠️ *DEADLINE BESOK (H-1)*"
	}

	days := int(diff.Hours() / 24)
	if days <= 0 {
		days = 1
	}
	if days <= 3 {
		return fmt.Sprintf("⚠️ *H-%d* (%d hari lagi)", days, days)
	}
	return fmt.Sprintf("⏳ *H-%d* (%d hari lagi)", days, days)
}

func (tm *TaskManager) CheckDuplicate(scopeJID, matkul, deskripsi string, optClassID ...string) (bool, *TaskItem, error) {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	var rows *sql.Rows
	var err error
	if classID != "" {
		rows, err = tm.db.Query(`
			SELECT id, matkul, deskripsi, deadline, created_by 
			FROM tasks 
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND is_done = 0
		`, scopeJID, classID)
	} else {
		rows, err = tm.db.Query(`
			SELECT id, matkul, deskripsi, deadline, created_by 
			FROM tasks 
			WHERE scope_jid = ? AND is_done = 0
		`, scopeJID)
	}
	if err != nil {
		return false, nil, err
	}
	defer rows.Close()

	cleanMatkul := strings.ToLower(strings.TrimSpace(matkul))
	cleanDesc := strings.ToLower(strings.TrimSpace(deskripsi))

	for rows.Next() {
		var item TaskItem
		err := rows.Scan(&item.ID, &item.Matkul, &item.Deskripsi, &item.Deadline, &item.CreatedBy)
		if err != nil {
			continue
		}

		existingMatkul := strings.ToLower(item.Matkul)
		existingDesc := strings.ToLower(item.Deskripsi)

		if strings.Contains(existingMatkul, cleanMatkul) || strings.Contains(cleanMatkul, existingMatkul) {
			if strings.EqualFold(existingDesc, cleanDesc) ||
				(len(cleanDesc) > 3 && strings.Contains(existingDesc, cleanDesc)) ||
				(len(existingDesc) > 3 && strings.Contains(cleanDesc, existingDesc)) {
				return true, &item, nil
			}
		}
	}

	return false, nil, nil
}

// AddTask menambahkan tugas baru ke dalam database dengan parsing tenggat waktu dan asosiasi kelas opsional
func (tm *TaskManager) AddTask(scopeJID string, isGroup bool, matkul, deskripsi, rawDeadline, createdBy string, now time.Time, optClassID ...string) (int64, string, error) {
	targetTime, deadlineLabel := parseDeadline(rawDeadline, now)

	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	stmt, err := tm.db.Prepare(`
		INSERT INTO tasks (scope_jid, class_id, is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)
	`)
	if err != nil {
		return 0, "", err
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		scopeJID, classID, isGroup,
		strings.TrimSpace(matkul),
		strings.TrimSpace(deskripsi),
		deadlineLabel,
		targetTime.Format("2006-01-02 15:04:05"),
		createdBy,
	)
	if err != nil {
		return 0, "", err
	}

	id, err := res.LastInsertId()
	return id, deadlineLabel, err
}

// GetActiveTasks mengambil seluruh tugas yang belum selesai, diurutkan dari deadline terdekat.
// Jika optClassID disertakan, kueri mencakup tugas dari scopeJID atau kelas terkait (Two-Way Sync).
func (tm *TaskManager) GetActiveTasks(scopeJID string, now time.Time, optClassID ...string) ([]TaskItem, error) {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	// Otomatis bersihkan tugas yang sudah lewat tenggat lebih dari 2 hari
	if classID != "" {
		_, _ = tm.db.Exec(`
			UPDATE tasks 
			SET is_done = 1 
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND is_done = 0 AND deadline_at IS NOT NULL AND deadline_at < ?
		`, scopeJID, classID, now.Add(-48*time.Hour).Format("2006-01-02 15:04:05"))
	} else {
		_, _ = tm.db.Exec(`
			UPDATE tasks 
			SET is_done = 1 
			WHERE scope_jid = ? AND is_done = 0 AND deadline_at IS NOT NULL AND deadline_at < ?
		`, scopeJID, now.Add(-48*time.Hour).Format("2006-01-02 15:04:05"))
	}

	var rows *sql.Rows
	var err error
	if classID != "" {
		rows, err = tm.db.Query(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND is_done = 0
			ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id ASC
		`, scopeJID, classID)
	} else {
		rows, err = tm.db.Query(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE scope_jid = ? AND is_done = 0
			ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id ASC
		`, scopeJID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var item TaskItem
		var rawDeadlineAt any
		var rawCreatedAt any
		err := rows.Scan(
			&item.ID, &item.ScopeJID, &item.ClassID, &item.IsGroup, &item.Matkul,
			&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		item.DeadlineAt = util.ParseFlexibleTime(rawDeadlineAt, now.Location())
		item.CreatedAt = util.ParseFlexibleTime(rawCreatedAt, now.Location())
		items = append(items, item)
	}

	return items, nil
}

// hasV3Schema melaporkan apakah database memakai skema akademik v3
// (tabel tasks relasional). Database lama pra-v3 memakai tabel tasks
// warisan berbasis scope_jid/is_done.
func (tm *TaskManager) hasV3Schema() bool {
	var ddl string
	err := tm.db.QueryRow(`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'tasks'`).Scan(&ddl)
	if err != nil {
		return false
	}
	return strings.Contains(ddl, "course_offering_id")
}

// resolveV3ClassID memetakan chat ke id kelas numerik v3 tanpa menebak:
// konteks chat pribadi/grup dulu, lalu pencocokan persis kode/slug kelas.
func (tm *TaskManager) resolveV3ClassID(scopeJID, classCode string) (int64, bool) {
	var classID int64
	if strings.TrimSpace(scopeJID) != "" {
		if err := tm.db.QueryRow(`SELECT class_id FROM chat_class_contexts WHERE chat_jid = ?`, scopeJID).Scan(&classID); err == nil {
			return classID, true
		}
		if err := tm.db.QueryRow(`SELECT class_id FROM whatsapp_channels WHERE jid = ? AND status = 'ACTIVE'`, scopeJID).Scan(&classID); err == nil {
			return classID, true
		}
	}
	if strings.TrimSpace(classCode) != "" {
		if err := tm.db.QueryRow(`SELECT id FROM classes WHERE code = ? OR slug = ? OR LOWER(code) = LOWER(?)`, classCode, classCode, classCode).Scan(&classID); err == nil {
			return classID, true
		}
	}
	return 0, false
}

// parseV3Time mengurai deadline_at v3 (UTC RFC3339) ke lokasi waktu acuan.
func parseV3Time(raw sql.NullString, loc *time.Location) (time.Time, bool) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if t, err := time.Parse(layout, strings.TrimSpace(raw.String)); err == nil {
			return t.In(loc), true
		}
	}
	return time.Time{}, false
}

// GetPublishedTasksV3 membaca tugas v3 yang terbit, belum dihapus, dan belum
// selesai: publication_status = 'PUBLISHED' AND deleted_at IS NULL AND
// completed_at IS NULL.
func (tm *TaskManager) GetPublishedTasksV3(scopeJID, classCode string, now time.Time) ([]TaskItem, error) {
	classID, ok := tm.resolveV3ClassID(scopeJID, classCode)
	if !ok {
		return nil, nil
	}
	rows, err := tm.db.Query(`
		SELECT t.id, c.code, c.name, t.title, t.instructions, t.deadline_at
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN semesters s ON s.id = co.semester_id
		WHERE s.class_id = ? AND t.publication_status = 'PUBLISHED'
			AND t.deleted_at IS NULL AND t.completed_at IS NULL
		ORDER BY t.deadline_at ASC, t.id ASC
	`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var id int64
		var courseCode, courseName, title string
		var instructions, deadlineAt sql.NullString
		if err := rows.Scan(&id, &courseCode, &courseName, &title, &instructions, &deadlineAt); err != nil {
			continue
		}
		desc := strings.TrimSpace(title)
		if instructions.Valid && strings.TrimSpace(instructions.String) != "" {
			flat := strings.Join(strings.Fields(instructions.String), " ")
			if len(flat) > 120 {
				flat = flat[:120] + "..."
			}
			desc = desc + " — " + flat
		}
		deadline, deadlineTime := "", time.Time{}
		if t, ok := parseV3Time(deadlineAt, now.Location()); ok {
			deadlineTime = t
			deadline = t.Format("02-01-2006, 15:04") + " WIB"
		}
		items = append(items, TaskItem{
			ID:         int(id),
			ScopeJID:   scopeJID,
			Matkul:     courseName,
			Deskripsi:  desc,
			Deadline:   deadline,
			DeadlineAt: deadlineTime,
			CreatedBy:  courseCode,
		})
	}
	return items, rows.Err()
}

// GetCompletedTasksV3 membaca tugas v3 yang sudah selesai untuk arsip/riwayat.
func (tm *TaskManager) GetCompletedTasksV3(scopeJID, classCode string, limit int, now time.Time) ([]TaskItem, error) {
	classID, ok := tm.resolveV3ClassID(scopeJID, classCode)
	if !ok {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := tm.db.Query(`
		SELECT t.id, c.code, c.name, t.title, t.instructions, t.deadline_at
		FROM tasks t
		JOIN course_offerings co ON co.id = t.course_offering_id
		JOIN courses c ON c.id = co.course_id
		JOIN semesters s ON s.id = co.semester_id
		WHERE s.class_id = ? AND t.publication_status = 'PUBLISHED'
			AND t.deleted_at IS NULL AND t.completed_at IS NOT NULL
		ORDER BY t.completed_at DESC, t.id DESC
		LIMIT ?
	`, classID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var id int64
		var courseCode, courseName, title string
		var instructions, deadlineAt sql.NullString
		if err := rows.Scan(&id, &courseCode, &courseName, &title, &instructions, &deadlineAt); err != nil {
			continue
		}
		deadline, deadlineTime := "", time.Time{}
		if t, ok := parseV3Time(deadlineAt, now.Location()); ok {
			deadlineTime = t
			deadline = t.Format("02-01-2006, 15:04") + " WIB"
		}
		items = append(items, TaskItem{
			ID:         int(id),
			ScopeJID:   scopeJID,
			Matkul:     courseName,
			Deskripsi:  strings.TrimSpace(title),
			Deadline:   deadline,
			DeadlineAt: deadlineTime,
			IsDone:     true,
			CreatedBy:  courseCode,
		})
	}
	return items, rows.Err()
}

// GetDueTasks mengambil tugas yang mendekati deadline (misal: "hari_ini", "besok", atau "urgent" untuk pengingat pagi)
func (tm *TaskManager) GetDueTasks(scopeJID string, filter string, now time.Time, optClassID ...string) ([]TaskItem, error) {
	all, err := tm.GetActiveTasks(scopeJID, now, optClassID...)
	if err != nil {
		return nil, err
	}

	var filtered []TaskItem
	tomorrow := now.Add(24 * time.Hour)

	for _, item := range all {
		if item.DeadlineAt.IsZero() {
			continue
		}

		isToday := item.DeadlineAt.Year() == now.Year() && item.DeadlineAt.YearDay() == now.YearDay()
		isTomorrow := item.DeadlineAt.Year() == tomorrow.Year() && item.DeadlineAt.YearDay() == tomorrow.YearDay()

		switch filter {
		case "hari_ini", "today":
			if isToday {
				filtered = append(filtered, item)
			}
		case "besok", "tomorrow":
			if isTomorrow {
				filtered = append(filtered, item)
			}
		case "urgent": // Hari ini atau besok (untuk peringatan pagi jam 06:00)
			if isToday || isTomorrow {
				filtered = append(filtered, item)
			}
		}
	}

	return filtered, nil
}

func (tm *TaskManager) CompleteTask(scopeJID string, taskID int, optClassID ...string) (bool, error) {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	var res sql.Result
	var err error
	if classID != "" {
		res, err = tm.db.Exec(`
			UPDATE tasks 
			SET is_done = 1 
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND id = ? AND is_done = 0
		`, scopeJID, classID, taskID)
	} else {
		res, err = tm.db.Exec(`
			UPDATE tasks 
			SET is_done = 1 
			WHERE scope_jid = ? AND id = ? AND is_done = 0
		`, scopeJID, taskID)
	}
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (tm *TaskManager) GetCompletedTasks(scopeJID string, limit int, now time.Time, optClassID ...string) ([]TaskItem, error) {
	if limit <= 0 {
		limit = 50
	}
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	var rows *sql.Rows
	var err error
	if classID != "" {
		rows, err = tm.db.Query(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND is_done = 1
			ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at DESC, id DESC
			LIMIT ?
		`, scopeJID, classID, limit)
	} else {
		rows, err = tm.db.Query(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE scope_jid = ? AND is_done = 1
			ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at DESC, id DESC
			LIMIT ?
		`, scopeJID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var item TaskItem
		var rawDeadlineAt any
		var rawCreatedAt any
		err := rows.Scan(
			&item.ID, &item.ScopeJID, &item.ClassID, &item.IsGroup, &item.Matkul,
			&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		item.DeadlineAt = util.ParseFlexibleTime(rawDeadlineAt, now.Location())
		item.CreatedAt = util.ParseFlexibleTime(rawCreatedAt, now.Location())
		items = append(items, item)
	}

	return items, nil
}

func (tm *TaskManager) DeleteTask(scopeJID string, taskID int, optClassID ...string) (bool, error) {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	var res sql.Result
	var err error
	if classID != "" {
		res, err = tm.db.Exec(`
			DELETE FROM tasks 
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND id = ?
		`, scopeJID, classID, taskID)
	} else {
		res, err = tm.db.Exec(`
			DELETE FROM tasks 
			WHERE scope_jid = ? AND id = ?
		`, scopeJID, taskID)
	}
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (tm *TaskManager) GetAllActiveTasks(now time.Time) ([]TaskItem, error) {
	_, _ = tm.db.Exec(`
		UPDATE tasks
		SET is_done = 1
		WHERE is_done = 0 AND deadline_at IS NOT NULL AND deadline_at < ?
	`, now.Add(-48*time.Hour).Format("2006-01-02 15:04:05"))

	rows, err := tm.db.Query(`
		SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
		FROM tasks
		WHERE is_done = 0
		ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var item TaskItem
		var rawDeadlineAt any
		var rawCreatedAt any
		err := rows.Scan(
			&item.ID, &item.ScopeJID, &item.ClassID, &item.IsGroup, &item.Matkul,
			&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		item.DeadlineAt = util.ParseFlexibleTime(rawDeadlineAt, now.Location())
		item.CreatedAt = util.ParseFlexibleTime(rawCreatedAt, now.Location())
		items = append(items, item)
	}

	return items, nil
}

func (tm *TaskManager) GetTasksByClassID(classID string, now time.Time) ([]TaskItem, error) {
	classID = strings.TrimSpace(classID)
	if classID == "" {
		return tm.GetAllActiveTasks(now)
	}

	_, _ = tm.db.Exec(`
		UPDATE tasks
		SET is_done = 1
		WHERE class_id = ? AND is_done = 0 AND deadline_at IS NOT NULL AND deadline_at < ?
	`, classID, now.Add(-48*time.Hour).Format("2006-01-02 15:04:05"))

	rows, err := tm.db.Query(`
		SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
		FROM tasks
		WHERE class_id = ? AND is_done = 0
		ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id ASC
	`, classID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []TaskItem
	for rows.Next() {
		var item TaskItem
		var rawDeadlineAt any
		var rawCreatedAt any
		err := rows.Scan(
			&item.ID, &item.ScopeJID, &item.ClassID, &item.IsGroup, &item.Matkul,
			&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		item.DeadlineAt = util.ParseFlexibleTime(rawDeadlineAt, now.Location())
		item.CreatedAt = util.ParseFlexibleTime(rawCreatedAt, now.Location())
		items = append(items, item)
	}

	return items, nil
}

func (tm *TaskManager) AddWebTask(matkul, deskripsi, rawDeadline, createdBy string, now time.Time, optClassID ...string) (int64, string, error) {
	classID := ""
	scopeJID := "web-dashboard"
	if len(optClassID) > 0 && strings.TrimSpace(optClassID[0]) != "" {
		classID = strings.TrimSpace(optClassID[0])
		scopeJID = "web:" + classID
	}
	return tm.AddTask(scopeJID, false, matkul, deskripsi, rawDeadline, createdBy, now, classID)
}

func (tm *TaskManager) CompleteTaskByID(taskID int) (bool, error) {
	res, err := tm.db.Exec(`
		UPDATE tasks
		SET is_done = 1
		WHERE id = ? AND is_done = 0
	`, taskID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (tm *TaskManager) DeleteTaskByID(taskID int) (bool, error) {
	res, err := tm.db.Exec(`
		DELETE FROM tasks
		WHERE id = ?
	`, taskID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (tm *TaskManager) UpdateTask(scopeJID string, taskID int, newDesc string, newRawDeadline string, now time.Time, optClassID ...string) (*TaskItem, string, error) {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	var row *sql.Row
	if classID != "" {
		row = tm.db.QueryRow(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND id = ?
		`, scopeJID, classID, taskID)
	} else {
		row = tm.db.QueryRow(`
			SELECT id, scope_jid, COALESCE(class_id, ''), is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
			FROM tasks
			WHERE scope_jid = ? AND id = ?
		`, scopeJID, taskID)
	}

	var item TaskItem
	var rawDeadlineAt any
	var rawCreatedAt any
	err := row.Scan(
		&item.ID, &item.ScopeJID, &item.ClassID, &item.IsGroup, &item.Matkul,
		&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}

	oldDeadline := item.Deadline
	targetTime, deadlineLabel := parseDeadline(newRawDeadline, now)

	descToSet := item.Deskripsi
	if newDesc != "" {
		descToSet = strings.TrimSpace(newDesc)
	}

	if classID != "" {
		_, err = tm.db.Exec(`
			UPDATE tasks 
			SET deskripsi = ?, deadline = ?, deadline_at = ?
			WHERE (scope_jid = ? OR (class_id != '' AND class_id = ?)) AND id = ?
		`, descToSet, deadlineLabel, targetTime.Format("2006-01-02 15:04:05"), scopeJID, classID, taskID)
	} else {
		_, err = tm.db.Exec(`
			UPDATE tasks 
			SET deskripsi = ?, deadline = ?, deadline_at = ?
			WHERE scope_jid = ? AND id = ?
		`, descToSet, deadlineLabel, targetTime.Format("2006-01-02 15:04:05"), scopeJID, taskID)
	}
	if err != nil {
		return nil, "", err
	}

	item.Deskripsi = descToSet
	item.Deadline = deadlineLabel
	item.DeadlineAt = targetTime

	return &item, oldDeadline, nil
}

// matchesHint memeriksa apakah teks mengandung salah satu kata kunci hint.
// Untuk kata kunci pendek (<= 2 karakter, contoh: "pr"), pencocokan dilakukan per kata utuh.
func matchesHint(text string, keywords []string) bool {
	lower := strings.ToLower(text)
	words := strings.Fields(lower)
	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if len(kwLower) <= 2 {
			for _, w := range words {
				cleanW := strings.Trim(w, ".,:;()[]*~_\"'!-")
				if cleanW == kwLower {
					return true
				}
			}
		} else {
			if strings.Contains(lower, kwLower) {
				return true
			}
		}
	}
	return false
}

func (tm *TaskManager) FilterTasksByQuery(scopeJID string, query string, cfg *schedule.JadwalConfig, now time.Time, optClassID ...string) ([]TaskItem, string, error) {
	allTasks, err := tm.GetActiveTasks(scopeJID, now, optClassID...)
	if err != nil {
		return nil, "", err
	}

	cleanQuery := strings.TrimSpace(query)
	lowerQuery := strings.ToLower(cleanQuery)

	targetTitle := strings.ToUpper(cleanQuery)
	var matchedOfficialName string
	isQueryPrak := strings.Contains(lowerQuery, "praktikum") || strings.Contains(lowerQuery, "praktek") || strings.Contains(lowerQuery, "prak") || strings.Contains(lowerQuery, "lab")
	isQueryTeori := strings.Contains(lowerQuery, "teori") || strings.Contains(lowerQuery, "kelas")

	if cfg != nil {
		item, _ := cfg.FindMataKuliah(cleanQuery, now)
		if item != nil {
			if isQueryPrak || isQueryTeori {
				matchedOfficialName = item.NamaMatkul
				targetTitle = strings.ToUpper(item.NamaMatkul)
			} else if officialName, ok := cfg.MataKuliah[item.KodeMatkul]; ok && officialName != "" {
				matchedOfficialName = officialName
				targetTitle = strings.ToUpper(officialName)
			} else {
				matchedOfficialName = item.NamaMatkul
				targetTitle = strings.ToUpper(item.NamaMatkul)
			}
		}
	}

	var filtered []TaskItem
	for _, task := range allTasks {
		lowerMatkul := strings.ToLower(task.Matkul)
		lowerDesc := strings.ToLower(task.Deskripsi)

		if matchedOfficialName != "" {
			if isQueryPrak && !strings.Contains(lowerMatkul, "praktikum") {
				continue
			}
			if isQueryTeori && !strings.Contains(lowerMatkul, "teori") {
				continue
			}
			if strings.Contains(lowerMatkul, strings.ToLower(matchedOfficialName)) {
				filtered = append(filtered, task)
				continue
			}
		}

		if strings.Contains(lowerMatkul, lowerQuery) || strings.Contains(lowerDesc, lowerQuery) {
			filtered = append(filtered, task)
		}
	}

	return filtered, targetTitle, nil
}

func (tm *TaskManager) FormatTaskList(tasks []TaskItem, isGroup bool, now time.Time, judulCustom ...string) string {
	var sb strings.Builder
	judul := "📋 *DAFTAR TUGAS KELAS*"
	if !isGroup {
		judul = "📋 *CATATAN TUGAS PRIBADI*"
	}
	if len(judulCustom) > 0 && judulCustom[0] != "" {
		judul = judulCustom[0]
	}

	sb.WriteString(fmt.Sprintf("%s\n", judul))
	sb.WriteString("──────────\n\n")

	if len(tasks) == 0 {
		if len(judulCustom) > 0 && judulCustom[0] != "" {
			sb.WriteString("🎉 *Tidak ada tugas aktif untuk kriteria ini!*\nSemua tugas telah selesai atau belum ada tugas yang dicatat.\n\n")
			sb.WriteString("_Ketik `!tugas` untuk melihat seluruh tugas aktif._")
			return sb.String()
		}
		sb.WriteString("🎉 *Tidak ada tugas aktif!*\nSemua tugas telah selesai atau belum ada tugas yang dicatat.\n\n")
		sb.WriteString("_Kelola tugas melalui Web Dashboard Pengelola._")
		return sb.String()
	}

	for i, task := range tasks {
		badge := GetUrgencyBadge(task.DeadlineAt, now)
		sb.WriteString(fmt.Sprintf("*%d. [%s]*\n", i+1, strings.ToUpper(task.Matkul)))
		sb.WriteString(fmt.Sprintf("   • Tugas    : %s\n", task.Deskripsi))
		sb.WriteString(fmt.Sprintf("   • Status   : %s\n", badge))
		sb.WriteString(fmt.Sprintf("   • Tenggat  : %s\n", task.Deadline))
		sb.WriteString(fmt.Sprintf("   • ID Tugas : #%d\n", task.ID))
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Di grup, tugas tetap terpajang sampai tenggatnya selesai._")
	return sb.String()
}

func (tm *TaskManager) FormatCompletedTaskList(tasks []TaskItem, isGroup bool) string {
	var sb strings.Builder

	if isGroup {
		sb.WriteString("📜 *ARSIP & RIWAYAT TUGAS SELESAI*\n")
		sb.WriteString("_Daftar tugas kelas yang telah ditandai selesai_\n")
	} else {
		sb.WriteString("📜 *ARSIP TUGAS PRIBADI SELESAI*\n")
		sb.WriteString("_Daftar catatan tugas pribadi yang telah selesai_\n")
	}
	sb.WriteString("──────────\n\n")

	if len(tasks) == 0 {
		sb.WriteString("Belum ada riwayat tugas yang diselesaikan.\n\n")
		sb.WriteString("_Ketik `!tugas` untuk melihat daftar tugas aktif saat ini._")
		return sb.String()
	}

	for idx, t := range tasks {
		sb.WriteString(fmt.Sprintf("*%d. ✅ [%s]*\n", idx+1, strings.ToUpper(t.Matkul)))
		sb.WriteString(fmt.Sprintf("   • Tugas    : %s\n", t.Deskripsi))
		sb.WriteString(fmt.Sprintf("   • Tenggat  : %s\n", t.Deadline))
		sb.WriteString(fmt.Sprintf("   • ID Tugas : #%d\n", t.ID))
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString(fmt.Sprintf("_Total: %d tugas telah diselesaikan sepanjang semester._", len(tasks)))
	return sb.String()
}

// HandleCommand memproses seluruh sub-perintah tugas (!tugas, hari ini, besok, tambah, selesai, hapus, bantuan)
func (tm *TaskManager) HandleCommand(
	scopeJID string, isGroup bool, senderJID string, isAdmin bool, rawMsg string, cfg *schedule.JadwalConfig, now time.Time, optClassID ...string,
) string {
	classID := ""
	if len(optClassID) > 0 {
		classID = strings.TrimSpace(optClassID[0])
	}

	clean := util.CleanCommandPrefix(rawMsg)

	parts := strings.SplitN(clean, " ", 2)
	action := ""
	payload := ""
	rest := ""
	if len(parts) > 1 {
		rest = strings.TrimSpace(parts[1])
		lowerRest := strings.ToLower(rest)
		if strings.HasPrefix(lowerRest, "hari ini") || strings.HasPrefix(lowerRest, "hariini") || strings.HasPrefix(lowerRest, "today") {
			action = "hari ini"
		} else if strings.HasPrefix(lowerRest, "besok") || strings.HasPrefix(lowerRest, "tomorrow") {
			action = "besok"
		} else {
			subParts := strings.SplitN(rest, " ", 2)
			action = strings.ToLower(subParts[0])
			if len(subParts) > 1 {
				payload = strings.TrimSpace(subParts[1])
			}
		}
	}

	switch action {
	case "", "list", "daftar":
		tasks, err := tm.readActiveTasks(scopeJID, now, classID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat daftar tugas: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now)

	case "riwayat", "arsip", "history":
		tasks, err := tm.readCompletedTasks(scopeJID, now, classID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat riwayat tugas: %v", err)
		}
		return tm.FormatCompletedTaskList(tasks, isGroup)

	case "hari ini", "hariini", "today":
		tasks, err := tm.readDueTasks(scopeJID, "hari_ini", now, classID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tugas hari ini: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now, "🚨 *TUGAS DEADLINE HARI INI*")

	case "besok", "tomorrow":
		tasks, err := tm.readDueTasks(scopeJID, "besok", now, classID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tugas besok: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now, "⚠️ *TUGAS DEADLINE BESOK (H-1)*")

	case "matkul", "cari", "filter":
		query := payload
		if query == "" {
			query = action
		}
		tasks, title, err := tm.filterTasks(scopeJID, query, cfg, now, classID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memfilter tugas: %v", err)
		}
		header := fmt.Sprintf("📋 *DAFTAR TUGAS KELAS: %s*", title)
		if !isGroup {
			header = fmt.Sprintf("📋 *CATATAN TUGAS PRIBADI: %s*", title)
		}
		return tm.FormatTaskList(tasks, isGroup, now, header)

	case "tambah", "add":
		return util.DashboardRedirectNotice("tugas")

	case "selesai", "done":
		return util.DashboardRedirectNotice("tugas")

	case "hapus", "delete", "rm":
		return util.DashboardRedirectNotice("tugas")

	case "edit", "update", "mundur", "perpanjang", "ganti":
		return util.DashboardRedirectNotice("tugas")

	case "bantuan", "help":
		var sb strings.Builder
		sb.WriteString("📖 *PANDUAN DEADLINE TRACKER TUGAS*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!tugas`\n  ➔ Seluruh tugas aktif dengan hitung mundur\n\n")
		sb.WriteString("• `!tugas [matkul]`\n  ➔ Filter tugas per mata kuliah (Cth: `!tugas sbd`, `!tugas aljabar`)\n\n")
		sb.WriteString("• `!tugas hari ini`\n  ➔ Tugas yang deadline-nya HARI INI\n\n")
		sb.WriteString("• `!tugas besok`\n  ➔ Tugas yang deadline-nya BESOK (H-1)\n\n")
		sb.WriteString("• `!tugas riwayat / !tugas arsip`\n  ➔ Rekam jejak tugas yang sudah selesai (Arsip)\n\n")
		sb.WriteString("──────────\n")
		sb.WriteString("⚠️ *Penambahan, perubahan, dan penyelesaian tugas kini hanya melalui Web Dashboard Pengelola:*\n")
		sb.WriteString("👉 http://localhost:8080/app.html (atau domain portal Anda)\n\n")
		sb.WriteString("_Tips: Bot otomatis memberi alert di jadwal pagi 06:00 jika ada tugas mendesak._")
		return sb.String()

	default:
		query := rest
		if query != "" {
			tasks, title, err := tm.filterTasks(scopeJID, query, cfg, now, classID)
			if err == nil {
				isCourse := false
				if cfg != nil {
					item, _ := cfg.FindMataKuliah(query, now)
					if item != nil {
						isCourse = true
					}
				}
				if isCourse || len(tasks) > 0 {
					header := fmt.Sprintf("📋 *DAFTAR TUGAS KELAS: %s*", title)
					if !isGroup {
						header = fmt.Sprintf("📋 *CATATAN TUGAS PRIBADI: %s*", title)
					}
					return tm.FormatTaskList(tasks, isGroup, now, header)
				}
			}
		}

		var sb strings.Builder
		sb.WriteString("📖 *PANDUAN DEADLINE TRACKER TUGAS*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!tugas`\n  ➔ Seluruh tugas aktif dengan hitung mundur\n\n")
		sb.WriteString("• `!tugas [matkul]`\n  ➔ Filter tugas per mata kuliah (Cth: `!tugas sbd`, `!tugas aljabar`)\n\n")
		sb.WriteString("• `!tugas hari ini`\n  ➔ Tugas yang deadline-nya HARI INI\n\n")
		sb.WriteString("• `!tugas besok`\n  ➔ Tugas yang deadline-nya BESOK (H-1)\n\n")
		sb.WriteString("• `!tugas riwayat / !tugas arsip`\n  ➔ Rekam jejak tugas yang sudah selesai (Arsip)\n\n")
		sb.WriteString("──────────\n")
		sb.WriteString("⚠️ *Penambahan, perubahan, dan penyelesaian tugas kini hanya melalui Web Dashboard Pengelola:*\n")
		sb.WriteString("👉 http://localhost:8080/app.html (atau domain portal Anda)\n\n")
		sb.WriteString("──────────\n")
		sb.WriteString("_Tips: Bot otomatis memberi alert di jadwal pagi 06:00 jika ada tugas mendesak._")
		return sb.String()
	}
}

// readActiveTasks memilih sumber baca tugas: skema v3 bila tersedia,
// tabel warisan bila database belum dimigrasi.
func (tm *TaskManager) readActiveTasks(scopeJID string, now time.Time, classID string) ([]TaskItem, error) {
	if tm.hasV3Schema() {
		return tm.GetPublishedTasksV3(scopeJID, classID, now)
	}
	return tm.GetActiveTasks(scopeJID, now, classID)
}

// readCompletedTasks memilih sumber baca arsip tugas.
func (tm *TaskManager) readCompletedTasks(scopeJID string, now time.Time, classID string) ([]TaskItem, error) {
	if tm.hasV3Schema() {
		return tm.GetCompletedTasksV3(scopeJID, classID, 50, now)
	}
	return tm.GetCompletedTasks(scopeJID, 50, now, classID)
}

// readDueTasks memfilter tugas aktif berdasarkan kedekatan deadline.
func (tm *TaskManager) readDueTasks(scopeJID, filter string, now time.Time, classID string) ([]TaskItem, error) {
	all, err := tm.readActiveTasks(scopeJID, now, classID)
	if err != nil {
		return nil, err
	}
	tomorrow := now.Add(24 * time.Hour)
	var filtered []TaskItem
	for _, item := range all {
		if item.DeadlineAt.IsZero() {
			continue
		}
		isToday := item.DeadlineAt.Year() == now.Year() && item.DeadlineAt.YearDay() == now.YearDay()
		isTomorrow := item.DeadlineAt.Year() == tomorrow.Year() && item.DeadlineAt.YearDay() == tomorrow.YearDay()
		if filter == "hari_ini" && isToday {
			filtered = append(filtered, item)
		}
		if filter == "besok" && isTomorrow {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

// filterTasks memilih sumber filter tugas berdasarkan skema yang tersedia.
func (tm *TaskManager) filterTasks(scopeJID, query string, cfg *schedule.JadwalConfig, now time.Time, classID string) ([]TaskItem, string, error) {
	if tm.hasV3Schema() {
		all, err := tm.GetPublishedTasksV3(scopeJID, classID, now)
		if err != nil {
			return nil, "", err
		}
		lower := strings.ToLower(strings.TrimSpace(query))
		var matched []TaskItem
		for _, item := range all {
			if strings.Contains(strings.ToLower(item.Matkul), lower) ||
				strings.Contains(strings.ToLower(item.Deskripsi), lower) {
				matched = append(matched, item)
			}
		}
		return matched, strings.ToUpper(query), nil
	}
	return tm.FilterTasksByQuery(scopeJID, query, cfg, now, classID)
}
