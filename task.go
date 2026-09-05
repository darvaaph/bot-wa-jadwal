package main

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// TaskItem merepresentasikan satu catatan tugas perkuliahan
type TaskItem struct {
	ID         int
	ScopeJID   string
	IsGroup    bool
	Matkul     string
	Deskripsi  string
	Deadline   string
	DeadlineAt time.Time
	CreatedBy  string
	IsDone     bool
	CreatedAt  time.Time
}

// TaskManager mengelola operasi CRUD tugas ke database SQLite
type TaskManager struct {
	db *sql.DB
}

// NewTaskManager menginisialisasi database SQLite dan membuat tabel jika belum ada
func NewTaskManager(dbPath string) (*TaskManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database tugas: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scope_jid TEXT NOT NULL,
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
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel tasks: %w", err)
	}

	// Migrasi aman jika kolom deadline_at belum ada pada database lama
	_, _ = db.Exec(`ALTER TABLE tasks ADD COLUMN deadline_at DATETIME;`)

	return &TaskManager{db: db}, nil
}

// Close menutup koneksi database
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

	// Cari jam (format HH:MM)
	jamStr := "23:59"
	timeRe := regexp.MustCompile(`\b([01]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)
	if match := timeRe.FindString(clean); match != "" {
		jamStr = strings.ReplaceAll(match, ".", ":")
	}

	var jam, menit int
	fmt.Sscanf(jamStr, "%d:%d", &jam, &menit)

	loc := refNow.Location()

	// 1. Hari ini / Today
	if strings.Contains(lower, "hari ini") || strings.Contains(lower, "hariini") || strings.Contains(lower, "today") {
		target := time.Date(refNow.Year(), refNow.Month(), refNow.Day(), jam, menit, 0, 0, loc)
		return target, fmt.Sprintf("Hari Ini, %02d:%02d WIB", jam, menit)
	}

	// 2. Besok / Tomorrow
	if strings.Contains(lower, "besok") || strings.Contains(lower, "tomorrow") {
		t := refNow.Add(24 * time.Hour)
		target := time.Date(t.Year(), t.Month(), t.Day(), jam, menit, 0, 0, loc)
		return target, fmt.Sprintf("Besok (%s), %02d:%02d WIB", getHariIndonesia(target), jam, menit)
	}

	// 3. Nama Hari (Senin, Selasa, Rabu, Kamis, Jumat, Sabtu, Minggu)
	namaHariMap := map[string]time.Weekday{
		"senin":     time.Monday,
		"monday":    time.Monday,
		"selasa":    time.Tuesday,
		"tuesday":   time.Tuesday,
		"rabu":      time.Wednesday,
		"wednesday": time.Wednesday,
		"kamis":     time.Thursday,
		"thursday":  time.Thursday,
		"jumat":     time.Friday,
		"jum'at":    time.Friday,
		"friday":    time.Friday,
		"sabtu":     time.Saturday,
		"saturday":  time.Saturday,
		"minggu":    time.Sunday,
		"sunday":    time.Sunday,
	}

	for dayName, weekday := range namaHariMap {
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
				getHariIndonesia(target), target.Day(), getBulanIndonesia(target), jam, menit)
		}
	}

	// 4. Format Tanggal Eksplisit (cth: 12-09-2026, 2026-09-12, 12/09/2026)
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
				getHariIndonesia(t), t.Day(), getBulanIndonesia(t), t.Hour(), t.Minute())
		}
	}

	// Fallback jika tidak terdeteksi: default 5 hari dari sekarang
	defaultTarget := refNow.AddDate(0, 0, 5)
	defaultTarget = time.Date(defaultTarget.Year(), defaultTarget.Month(), defaultTarget.Day(), jam, menit, 0, 0, loc)
	return defaultTarget, clean
}

// parseFlexibleTime membaca datetime SQLite baik yang bertipe string, []byte, maupun time.Time
func parseFlexibleTime(val any, loc *time.Location) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		if loc != nil {
			// Driver sqlite sering mengembalikan wall-clock time sebagai UTC ("2026-09-09 23:59:00 UTC").
			// Kita rekonsiliasi ke zona waktu lokal target tanpa pergeseran offset.
			return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), loc)
		}
		return v
	case string:
		return parseTimeString(v, loc)
	case []byte:
		return parseTimeString(string(v), loc)
	}
	return time.Time{}
}

func parseTimeString(s string, loc *time.Location) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, s, loc); err == nil {
			return t
		}
		if t, err := time.Parse(l, s); err == nil {
			if loc != nil {
				return t.In(loc)
			}
			return t
		}
	}
	return time.Time{}
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

	// Cek apakah jatuh tempo hari ini (tanggal & tahun sama)
	if deadlineAt.Year() == now.Year() && deadlineAt.YearDay() == now.YearDay() {
		hours := int(diff.Hours())
		mins := int(diff.Minutes()) % 60
		if hours > 0 {
			return fmt.Sprintf("🚨 *DEADLINE HARI INI* (Sisa ~%d jam)", hours)
		}
		return fmt.Sprintf("🚨 *DEADLINE HARI INI* (Sisa ~%d menit)", mins)
	}

	// Cek apakah jatuh tempo besok (H-1)
	tomorrow := now.Add(24 * time.Hour)
	if deadlineAt.Year() == tomorrow.Year() && deadlineAt.YearDay() == tomorrow.YearDay() {
		return "⚠️ *DEADLINE BESOK (H-1)*"
	}

	// Hitung hari tersisa
	days := int(diff.Hours() / 24)
	if days <= 0 {
		days = 1
	}
	if days <= 3 {
		return fmt.Sprintf("⚠️ *H-%d* (%d hari lagi)", days, days)
	}
	return fmt.Sprintf("⏳ *H-%d* (%d hari lagi)", days, days)
}

// CheckDuplicate memeriksa apakah tugas serupa sudah pernah dibuat dan masih aktif
func (tm *TaskManager) CheckDuplicate(scopeJID, matkul, deskripsi string) (bool, *TaskItem, error) {
	rows, err := tm.db.Query(`
		SELECT id, matkul, deskripsi, deadline, created_by 
		FROM tasks 
		WHERE scope_jid = ? AND is_done = 0
	`, scopeJID)
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

// AddTask menambahkan tugas baru ke dalam database dengan parsing tenggat waktu
func (tm *TaskManager) AddTask(scopeJID string, isGroup bool, matkul, deskripsi, rawDeadline, createdBy string, now time.Time) (int64, string, error) {
	targetTime, deadlineLabel := parseDeadline(rawDeadline, now)

	stmt, err := tm.db.Prepare(`
		INSERT INTO tasks (scope_jid, is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done)
		VALUES (?, ?, ?, ?, ?, ?, ?, 0)
	`)
	if err != nil {
		return 0, "", err
	}
	defer stmt.Close()

	res, err := stmt.Exec(
		scopeJID, isGroup,
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

// GetActiveTasks mengambil seluruh tugas yang belum selesai, diurutkan dari deadline terdekat
func (tm *TaskManager) GetActiveTasks(scopeJID string, now time.Time) ([]TaskItem, error) {
	// Otomatis bersihkan tugas grup yang sudah lewat tenggat lebih dari 2 hari
	_, _ = tm.db.Exec(`
		UPDATE tasks 
		SET is_done = 1 
		WHERE scope_jid = ? AND is_done = 0 AND deadline_at IS NOT NULL AND deadline_at < ?
	`, scopeJID, now.Add(-48*time.Hour).Format("2006-01-02 15:04:05"))

	rows, err := tm.db.Query(`
		SELECT id, scope_jid, is_group, matkul, deskripsi, deadline, deadline_at, created_by, is_done, created_at
		FROM tasks
		WHERE scope_jid = ? AND is_done = 0
		ORDER BY CASE WHEN deadline_at IS NULL THEN 1 ELSE 0 END, deadline_at ASC, id ASC
	`, scopeJID)
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
			&item.ID, &item.ScopeJID, &item.IsGroup, &item.Matkul,
			&item.Deskripsi, &item.Deadline, &rawDeadlineAt, &item.CreatedBy, &item.IsDone, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		item.DeadlineAt = parseFlexibleTime(rawDeadlineAt, now.Location())
		item.CreatedAt = parseFlexibleTime(rawCreatedAt, now.Location())
		items = append(items, item)
	}

	return items, nil
}

// GetDueTasks mengambil tugas yang mendekati deadline (misal: "hari_ini", "besok", atau "urgent" untuk pengingat pagi)
func (tm *TaskManager) GetDueTasks(scopeJID string, filter string, now time.Time) ([]TaskItem, error) {
	all, err := tm.GetActiveTasks(scopeJID, now)
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
		case "urgent": // Hari ini atau besok (untuk peringatan pagi jam 06:30)
			if isToday || isTomorrow {
				filtered = append(filtered, item)
			}
		}
	}

	return filtered, nil
}

// CompleteTask menandai tugas sebagai selesai berdasarkan ID
func (tm *TaskManager) CompleteTask(scopeJID string, taskID int) (bool, error) {
	res, err := tm.db.Exec(`
		UPDATE tasks 
		SET is_done = 1 
		WHERE scope_jid = ? AND id = ? AND is_done = 0
	`, scopeJID, taskID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// DeleteTask menghapus tugas secara permanen dari database
func (tm *TaskManager) DeleteTask(scopeJID string, taskID int) (bool, error) {
	res, err := tm.db.Exec(`
		DELETE FROM tasks 
		WHERE scope_jid = ? AND id = ?
	`, scopeJID, taskID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// FormatTaskList merapikan daftar tugas aktif menjadi pesan WhatsApp dengan badge urgensi otomatis
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
		sb.WriteString("🎉 *Tidak ada tugas aktif!*\nSemua tugas telah selesai atau belum ada tugas yang dicatat.\n\n")
		sb.WriteString("_Ketik `!tugas tambah` untuk menambah catatan tugas._")
		return sb.String()
	}

	for i, task := range tasks {
		badge := GetUrgencyBadge(task.DeadlineAt, now)
		sb.WriteString(fmt.Sprintf("*%d. [%s] %s*\n", i+1, strings.ToUpper(task.Matkul), task.Deskripsi))
		sb.WriteString(fmt.Sprintf("   • Status   : %s\n", badge))
		sb.WriteString(fmt.Sprintf("   • Tenggat  : %s\n", task.Deadline))
		sb.WriteString(fmt.Sprintf("   • ID Tugas : #%d\n", task.ID))
		if isGroup && task.CreatedBy != "" {
			creatorShort := strings.Split(task.CreatedBy, "@")[0]
			sb.WriteString(fmt.Sprintf("   • Oleh     : @%s\n", creatorShort))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Di grup, tugas tetap terpajang sampai tenggatnya selesai._")
	return sb.String()
}

// HandleCommand memproses seluruh sub-perintah tugas (!tugas, hari ini, besok, tambah, selesai, hapus, bantuan)
func (tm *TaskManager) HandleCommand(scopeJID string, isGroup bool, senderJID string, isAdmin bool, rawMsg string, now time.Time) string {
	clean := strings.TrimSpace(rawMsg)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		clean = strings.TrimSpace(clean[1:])
	}

	parts := strings.SplitN(clean, " ", 2)
	action := ""
	payload := ""
	if len(parts) > 1 {
		rest := strings.TrimSpace(parts[1])
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
		tasks, err := tm.GetActiveTasks(scopeJID, now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat daftar tugas: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now)

	case "hari ini", "hariini", "today":
		tasks, err := tm.GetDueTasks(scopeJID, "hari_ini", now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tugas hari ini: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now, "🚨 *TUGAS DEADLINE HARI INI*")

	case "besok", "tomorrow":
		tasks, err := tm.GetDueTasks(scopeJID, "besok", now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat tugas besok: %v", err)
		}
		return tm.FormatTaskList(tasks, isGroup, now, "⚠️ *TUGAS DEADLINE BESOK (H-1)*")

	case "tambah", "add":
		// Pengecekan Hak Akses: Di grup WAJIB Admin
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penambahan tugas hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil) agar daftar tugas tetap teratur."
		}

		// Validasi format pemisah pipa: Matkul | Deskripsi | Deadline
		segments := strings.Split(payload, "|")
		if len(segments) < 3 {
			return "⚠️ *Format Penambahan Tugas Kurang Tepat*\n\n" +
				"Gunakan tanda pemisah pipa `|`:\n" +
				"`!tugas tambah [Matkul] | [Deskripsi Tugas] | [Tenggat Waktu]`\n\n" +
				"*Contoh:*\n" +
				"• `!tugas tambah SBD | Lapres Modul 2 | Jumat 23:59`\n" +
				"• `!tugas tambah Aljabar | Latihan Bab 3 | Besok 12:00`"
		}

		matkul := strings.TrimSpace(segments[0])
		deskripsi := strings.TrimSpace(segments[1])
		rawDeadline := strings.TrimSpace(segments[2])

		if matkul == "" || deskripsi == "" || rawDeadline == "" {
			return "⚠️ Seluruh kolom (Matkul, Deskripsi, dan Tenggat Waktu) wajib diisi."
		}

		// Pengecekan Anti-Duplikasi
		isDup, existing, err := tm.CheckDuplicate(scopeJID, matkul, deskripsi)
		if err != nil {
			return fmt.Sprintf("❌ Terjadi kesalahan pengecekan data: %v", err)
		}
		if isDup && existing != nil {
			return fmt.Sprintf("⚠️ *Tugas Serupa Sudah Terdaftar!*\n\nTugas berikut sudah ada di daftar aktif:\n• *ID #%d: [%s] %s*\n• Tenggat: %s\n\nKetik `!tugas` untuk melihat daftar lengkap.",
				existing.ID, existing.Matkul, existing.Deskripsi, existing.Deadline)
		}

		id, label, err := tm.AddTask(scopeJID, isGroup, matkul, deskripsi, rawDeadline, senderJID, now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menyimpan tugas: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *TUGAS BERHASIL DITAMBAHKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Tugas : #%d\n", id))
		sb.WriteString(fmt.Sprintf("• Matkul   : %s\n", strings.ToUpper(matkul)))
		sb.WriteString(fmt.Sprintf("• Deskripsi: %s\n", deskripsi))
		sb.WriteString(fmt.Sprintf("• Tenggat  : %s\n", label))
		sb.WriteString("──────────\n")
		sb.WriteString("_Bot akan otomatis mengingatkan tugas ini saat mendekati tenggat._")
		return sb.String()

	case "selesai", "done":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penandaan tugas selesai hanya dapat dilakukan oleh *Admin Grup*."
		}

		taskID, err := strconv.Atoi(payload)
		if err != nil || taskID <= 0 {
			return "⚠️ Sertakan ID tugas yang ingin diselesaikan.\nContoh: `!tugas selesai 1`\n\nKetik `!tugas` untuk melihat nomor ID tugas."
		}

		ok, err := tm.CompleteTask(scopeJID, taskID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memperbarui status tugas: %v", err)
		}
		if !ok {
			return fmt.Sprintf("ℹ️ Tugas dengan ID #%d tidak ditemukan atau sudah diselesaikan sebelumnya.", taskID)
		}

		return fmt.Sprintf("🎉 *TUGAS SELESAI!*\nTugas dengan ID #%d telah ditandai selesai dan diarsipkan.", taskID)

	case "hapus", "delete", "rm":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penghapusan tugas hanya dapat dilakukan oleh *Admin Grup*."
		}

		taskID, err := strconv.Atoi(payload)
		if err != nil || taskID <= 0 {
			return "⚠️ Sertakan ID tugas yang ingin dihapus.\nContoh: `!tugas hapus 1`\n\nKetik `!tugas` untuk melihat nomor ID tugas."
		}

		ok, err := tm.DeleteTask(scopeJID, taskID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menghapus tugas: %v", err)
		}
		if !ok {
			return fmt.Sprintf("ℹ️ Tugas dengan ID #%d tidak ditemukan.", taskID)
		}

		return fmt.Sprintf("🗑️ *TUGAS DIHAPUS*\nTugas dengan ID #%d telah berhasil dihapus dari database.", taskID)

	case "bantuan", "help":
		fallthrough
	default:
		var sb strings.Builder
		sb.WriteString("📖 *PANDUAN DEADLINE TRACKER TUGAS*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!tugas`\n  ➔ Seluruh tugas aktif dengan hitung mundur\n\n")
		sb.WriteString("• `!tugas hari ini`\n  ➔ Tugas yang deadline-nya HARI INI\n\n")
		sb.WriteString("• `!tugas besok`\n  ➔ Tugas yang deadline-nya BESOK (H-1)\n\n")
		sb.WriteString("• `!tugas tambah [Matkul] | [Judul] | [Tenggat]`\n  ➔ Menambah tugas baru (Khusus Admin di grup)\n  Contoh: `!tugas tambah SBD | Lapres | Jumat 23:59`\n\n")
		sb.WriteString("• `!tugas selesai [ID]`\n  ➔ Menyelesaikan tugas\n\n")
		sb.WriteString("• `!tugas hapus [ID]`\n  ➔ Menghapus tugas dari sistem\n\n")
		sb.WriteString("──────────\n")
		sb.WriteString("_Tips: Bot otomatis memberi alert di jadwal pagi 06:30 jika ada tugas mendesak._")
		return sb.String()
	}
}
