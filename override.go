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

// ScheduleOverride merepresentasikan satu catatan perubahan/jadwal pengganti sementara
type ScheduleOverride struct {
	ID           int
	ScopeJID     string
	Type         string // RESCHEDULE, CANCEL, EXTRA
	KodeMatkul   string
	NamaMatkul   string
	Dosen        string
	InisialDosen string
	OrigDate     string // YYYY-MM-DD
	OrigJam      string // contoh: "07:00 - 08:40"
	TargetDate   string // YYYY-MM-DD
	NewJam       string // contoh: "13:00 - 14:40"
	Ruang        string
	Alasan       string
	CreatedBy    string
	CreatedAt    time.Time
}

// OverrideManager mengelola database jadwal pengganti sementara
type OverrideManager struct {
	db *sql.DB
}

// NewOverrideManager menginisialisasi tabel schedule_overrides pada SQLite
func NewOverrideManager(dbPath string) (*OverrideManager, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka database override: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS schedule_overrides (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		scope_jid TEXT NOT NULL,
		override_type TEXT NOT NULL,
		kode_matkul TEXT NOT NULL,
		nama_matkul TEXT NOT NULL,
		dosen TEXT NOT NULL,
		inisial_dosen TEXT NOT NULL,
		orig_date TEXT NOT NULL,
		orig_jam TEXT NOT NULL,
		target_date TEXT NOT NULL,
		new_jam TEXT NOT NULL,
		ruang TEXT NOT NULL,
		alasan TEXT NOT NULL,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_overrides_scope ON schedule_overrides(scope_jid, target_date);
	`
	_, err = db.Exec(query)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat tabel schedule_overrides: %w", err)
	}

	return &OverrideManager{db: db}, nil
}

// Close menutup koneksi database
func (om *OverrideManager) Close() error {
	if om.db != nil {
		return om.db.Close()
	}
	return nil
}

// AddReschedule menambahkan perubahan jadwal kuliah ke tanggal/jam lain
func (om *OverrideManager) AddReschedule(
	scopeJID string, item JadwalItem, origDate, targetDate time.Time, newJam, newRuang, createdBy string,
) (*ScheduleOverride, error) {
	ruang := newRuang
	if ruang == "" {
		ruang = item.Ruang
	}

	origDateStr := origDate.Format("2006-01-02")
	targetDateStr := targetDate.Format("2006-01-02")

	res, err := om.db.Exec(`
		INSERT INTO schedule_overrides (
			scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
			orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by
		) VALUES (?, 'RESCHEDULE', ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?)
	`, scopeJID, item.KodeMatkul, item.NamaMatkul, item.Dosen, item.InisialDosen,
		origDateStr, item.Jam, targetDateStr, newJam, ruang, createdBy)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "RESCHEDULE",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		OrigDate:     origDateStr,
		OrigJam:      item.Jam,
		TargetDate:   targetDateStr,
		NewJam:       newJam,
		Ruang:        ruang,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// AddCancel menandai kuliah ditiadakan pada tanggal tertentu
func (om *OverrideManager) AddCancel(
	scopeJID string, item JadwalItem, targetDate time.Time, alasan, createdBy string,
) (*ScheduleOverride, error) {
	dateStr := targetDate.Format("2006-01-02")

	res, err := om.db.Exec(`
		INSERT INTO schedule_overrides (
			scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
			orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by
		) VALUES (?, 'CANCEL', ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)
	`, scopeJID, item.KodeMatkul, item.NamaMatkul, item.Dosen, item.InisialDosen,
		dateStr, item.Jam, dateStr, item.Ruang, alasan, createdBy)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "CANCEL",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		OrigDate:     dateStr,
		OrigJam:      item.Jam,
		TargetDate:   dateStr,
		Ruang:        item.Ruang,
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// AddExtra menambahkan jadwal kuliah pengganti/ekstra di hari tertentu
func (om *OverrideManager) AddExtra(
	scopeJID string, item JadwalItem, targetDate time.Time, jam, ruang, alasan, createdBy string,
) (*ScheduleOverride, error) {
	dateStr := targetDate.Format("2006-01-02")
	if ruang == "" {
		ruang = item.Ruang
	}

	res, err := om.db.Exec(`
		INSERT INTO schedule_overrides (
			scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
			orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by
		) VALUES (?, 'EXTRA', ?, ?, ?, ?, '', '', ?, ?, ?, ?, ?)
	`, scopeJID, item.KodeMatkul, item.NamaMatkul, item.Dosen, item.InisialDosen,
		dateStr, jam, ruang, alasan, createdBy)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "EXTRA",
		KodeMatkul:   item.KodeMatkul,
		NamaMatkul:   item.NamaMatkul,
		Dosen:        item.Dosen,
		InisialDosen: item.InisialDosen,
		TargetDate:   dateStr,
		NewJam:       jam,
		Ruang:        ruang,
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// GetOverridesForDate mengambil seluruh catatan override yang mempengaruhi tanggal tertentu
func (om *OverrideManager) GetOverridesForDate(scopeJID string, date time.Time) ([]ScheduleOverride, error) {
	dateStr := date.Format("2006-01-02")
	rows, err := om.db.Query(`
		SELECT id, scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
		       orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by, created_at
		FROM schedule_overrides
		WHERE scope_jid = ? AND (orig_date = ? OR target_date = ?)
		ORDER BY id ASC
	`, scopeJID, dateStr, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleOverride
	for rows.Next() {
		var o ScheduleOverride
		var rawCreatedAt any
		err := rows.Scan(
			&o.ID, &o.ScopeJID, &o.Type, &o.KodeMatkul, &o.NamaMatkul, &o.Dosen, &o.InisialDosen,
			&o.OrigDate, &o.OrigJam, &o.TargetDate, &o.NewJam, &o.Ruang, &o.Alasan, &o.CreatedBy, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		o.CreatedAt = parseFlexibleTime(rawCreatedAt, date.Location())
		list = append(list, o)
	}

	return list, nil
}

// GetActiveOverrides mengambil semua override yang tanggal targetnya belum lewat
func (om *OverrideManager) GetActiveOverrides(scopeJID string, now time.Time) ([]ScheduleOverride, error) {
	todayStr := now.Format("2006-01-02")
	rows, err := om.db.Query(`
		SELECT id, scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
		       orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by, created_at
		FROM schedule_overrides
		WHERE scope_jid = ? AND (target_date >= ? OR orig_date >= ?)
		ORDER BY target_date ASC, id ASC
	`, scopeJID, todayStr, todayStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ScheduleOverride
	for rows.Next() {
		var o ScheduleOverride
		var rawCreatedAt any
		err := rows.Scan(
			&o.ID, &o.ScopeJID, &o.Type, &o.KodeMatkul, &o.NamaMatkul, &o.Dosen, &o.InisialDosen,
			&o.OrigDate, &o.OrigJam, &o.TargetDate, &o.NewJam, &o.Ruang, &o.Alasan, &o.CreatedBy, &rawCreatedAt,
		)
		if err != nil {
			continue
		}
		o.CreatedAt = parseFlexibleTime(rawCreatedAt, now.Location())
		list = append(list, o)
	}

	return list, nil
}

// CancelOverride menghapus perubahan jadwal sementara berdasarkan ID
func (om *OverrideManager) CancelOverride(scopeJID string, id int) (bool, error) {
	res, err := om.db.Exec(`DELETE FROM schedule_overrides WHERE scope_jid = ? AND id = ?`, scopeJID, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	return affected > 0, err
}

// AddHoliday menambahkan pengumuman libur harian (seluruh perkuliahan pada tanggal tersebut ditiadakan)
func (om *OverrideManager) AddHoliday(scopeJID string, targetDate time.Time, alasan string, createdBy string) (*ScheduleOverride, error) {
	tglStr := targetDate.Format("2006-01-02")
	if alasan == "" {
		alasan = "Libur Perkuliahan"
	}

	res, err := om.db.Exec(`
		INSERT INTO schedule_overrides (
			scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
			orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, scopeJID, "HOLIDAY", "LIBUR", "LIBUR SEHARIAN", "-", "-", tglStr, "-", tglStr, "-", "-", alasan, createdBy)
	if err != nil {
		return nil, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &ScheduleOverride{
		ID:           int(id),
		ScopeJID:     scopeJID,
		Type:         "HOLIDAY",
		KodeMatkul:   "LIBUR",
		NamaMatkul:   "LIBUR SEHARIAN",
		Dosen:        "-",
		InisialDosen: "-",
		OrigDate:     tglStr,
		OrigJam:      "-",
		TargetDate:   tglStr,
		NewJam:       "-",
		Ruang:        "-",
		Alasan:       alasan,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}, nil
}

// GetHolidayOverride mengecek apakah tanggal tertentu ditandai sebagai hari libur
func (om *OverrideManager) GetHolidayOverride(scopeJID string, date time.Time) *ScheduleOverride {
	tglStr := date.Format("2006-01-02")
	row := om.db.QueryRow(`
		SELECT id, scope_jid, override_type, kode_matkul, nama_matkul, dosen, inisial_dosen,
		       orig_date, orig_jam, target_date, new_jam, ruang, alasan, created_by, created_at
		FROM schedule_overrides
		WHERE scope_jid = ? AND target_date = ? AND override_type = 'HOLIDAY'
		ORDER BY id DESC LIMIT 1
	`, scopeJID, tglStr)

	var o ScheduleOverride
	var rawCreatedAt any
	err := row.Scan(
		&o.ID, &o.ScopeJID, &o.Type, &o.KodeMatkul, &o.NamaMatkul, &o.Dosen, &o.InisialDosen,
		&o.OrigDate, &o.OrigJam, &o.TargetDate, &o.NewJam, &o.Ruang, &o.Alasan, &o.CreatedBy, &rawCreatedAt,
	)
	if err != nil {
		return nil
	}
	o.CreatedAt = parseFlexibleTime(rawCreatedAt, date.Location())
	return &o
}

// CalculateDurationInMinutes menghitung durasi rentang jam dalam satuan menit (cth: "07:00 - 08:40" -> 100)
func CalculateDurationInMinutes(jamRange string) int {
	parts := strings.Split(jamRange, "-")
	if len(parts) < 2 {
		return 100 // default 2 SKS jika tidak bisa dihitung
	}

	parseMinute := func(s string) int {
		s = strings.TrimSpace(strings.ReplaceAll(s, ".", ":"))
		var h, m int
		fmt.Sscanf(s, "%d:%d", &h, &m)
		return h*60 + m
	}

	startMin := parseMinute(parts[0])
	endMin := parseMinute(parts[1])
	if endMin > startMin {
		return endMin - startMin
	}
	return 100
}

// AutoCompleteJamRange membuat rentang jam otomatis jika input hanya jam mulai (cth: "13:00" -> "13:00 - 14:40")
func AutoCompleteJamRange(inputJam string, durationMinutes int) string {
	clean := strings.TrimSpace(inputJam)
	timeRe := regexp.MustCompile(`\b([01]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)

	// Jika input sudah menyertakan jam mulai dan jam selesai (cth: "09:00 - 11:30" atau "sabtu 09:00 - 11:30")
	if strings.Contains(clean, "-") || strings.Contains(clean, "s/d") || strings.Contains(clean, "sampai") {
		matches := timeRe.FindAllString(clean, -1)
		if len(matches) >= 2 {
			start := strings.ReplaceAll(matches[0], ".", ":")
			end := strings.ReplaceAll(matches[1], ".", ":")
			return fmt.Sprintf("%s - %s", start, end)
		}
	}

	// Hanya jam mulai
	match := timeRe.FindString(clean)
	var startH, startM int
	if match != "" {
		match = strings.ReplaceAll(match, ".", ":")
		fmt.Sscanf(match, "%d:%d", &startH, &startM)
	} else {
		// Cek format angka bulat "13" atau "jam 13"
		numRe := regexp.MustCompile(`\b([01]?[0-9]|2[0-3])\b`)
		if m := numRe.FindString(clean); m != "" {
			fmt.Sscanf(m, "%d", &startH)
			startM = 0
		} else {
			startH = 13
			startM = 0
		}
	}

	if durationMinutes <= 0 {
		durationMinutes = 100
	}

	startTotal := startH*60 + startM
	endTotal := startTotal + durationMinutes
	endH := (endTotal / 60) % 24
	endM := endTotal % 60

	return fmt.Sprintf("%02d:%02d - %02d:%02d", startH, startM, endH, endM)
}

// ParseOverrideDate mengekstrak tanggal target dari input teks fleksibel
func ParseOverrideDate(rawInput string, refNow time.Time) time.Time {
	clean := strings.ToLower(strings.TrimSpace(rawInput))
	loc := refNow.Location()

	if strings.Contains(clean, "hari ini") || strings.Contains(clean, "today") {
		return refNow
	}
	if strings.Contains(clean, "besok") || strings.Contains(clean, "tomorrow") {
		return refNow.Add(24 * time.Hour)
	}

	namaHariMap := map[string]time.Weekday{
		"senin":  time.Monday,
		"selasa": time.Tuesday,
		"rabu":   time.Wednesday,
		"kamis":  time.Thursday,
		"jumat":  time.Friday,
		"sabtu":  time.Saturday,
		"minggu": time.Sunday,
	}

	for dayName, weekday := range namaHariMap {
		if strings.Contains(clean, dayName) {
			daysAhead := int(weekday - refNow.Weekday())
			if daysAhead <= 0 {
				daysAhead += 7
			}
			t := refNow.AddDate(0, 0, daysAhead)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		}
	}

	layouts := []string{"02-01-2006", "2006-01-02", "02/01/2006"}
	for _, layout := range layouts {
		re := regexp.MustCompile(`\b\d{2,4}[-/]\d{2}[-/]\d{2,4}\b`)
		if found := re.FindString(clean); found != "" {
			if t, err := time.ParseInLocation(layout, found, loc); err == nil {
				return t
			}
		}
	}

	bulanMap := map[string]time.Month{
		"jan": time.January, "januari": time.January,
		"feb": time.February, "februari": time.February,
		"mar": time.March, "maret": time.March,
		"apr": time.April, "april": time.April,
		"mei": time.May,
		"jun": time.June, "juni": time.June,
		"jul": time.July, "juli": time.July,
		"agu": time.August, "agustus": time.August, "ags": time.August,
		"sep": time.September, "september": time.September, "sept": time.September,
		"okt": time.October, "oktober": time.October,
		"nov": time.November, "november": time.November,
		"des": time.December, "desember": time.December,
	}

	dateWordRe := regexp.MustCompile(`\b(\d{1,2})[\s\-\/]+([a-zA-Z]+)(?:[\s\-\/]+(20\d{2}))?\b`)
	if matches := dateWordRe.FindStringSubmatch(clean); len(matches) >= 3 {
		day, _ := strconv.Atoi(matches[1])
		bStr := strings.ToLower(matches[2])
		year := refNow.Year()
		hasYear := false
		if len(matches) > 3 && matches[3] != "" {
			fmt.Sscanf(matches[3], "%d", &year)
			hasYear = true
		}

		if monthVal, ok := bulanMap[bStr]; ok && day >= 1 && day <= 31 {
			target := time.Date(year, monthVal, day, 0, 0, 0, 0, loc)
			if !hasYear && target.AddDate(0, 1, 0).Before(refNow) {
				target = target.AddDate(1, 0, 0)
			}
			return target
		}
	}

	return refNow.Add(24 * time.Hour)
}

// FormatActiveOverrides merapikan daftar jadwal pengganti aktif
func (om *OverrideManager) FormatActiveOverrides(overrides []ScheduleOverride) string {
	var sb strings.Builder
	sb.WriteString("🔄 *DAFTAR JADWAL PENGGANTI AKTIF*\n")
	sb.WriteString("──────────\n\n")

	if len(overrides) == 0 {
		sb.WriteString("ℹ️ Tidak ada perubahan jadwal atau kuliah pengganti sementara.\nSemua perkuliahan berjalan sesuai jadwal normal.\n")
		return sb.String()
	}

	for i, o := range overrides {
		sb.WriteString(fmt.Sprintf("*%d. [%s] %s*\n", i+1, strings.ToUpper(o.Type), o.NamaMatkul))
		sb.WriteString(fmt.Sprintf("   • ID Perubahan: #%d\n", o.ID))
		switch o.Type {
		case "HOLIDAY":
			sb.WriteString(fmt.Sprintf("   • Tanggal Libur : %s (Seharian)\n", o.TargetDate))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Keterangan    : %s\n", o.Alasan))
			}
		case "CANCEL":
			sb.WriteString(fmt.Sprintf("   • Tanggal Libur : %s (%s)\n", o.TargetDate, o.OrigJam))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Keterangan    : %s\n", o.Alasan))
			}
		case "RESCHEDULE":
			sb.WriteString(fmt.Sprintf("   • Semula : %s (%s)\n", o.OrigDate, o.OrigJam))
			sb.WriteString(fmt.Sprintf("   • Menjadi: %s (%s WIB)\n", o.TargetDate, o.NewJam))
			sb.WriteString(fmt.Sprintf("   • Ruang  : %s\n", o.Ruang))
		case "EXTRA":
			sb.WriteString(fmt.Sprintf("   • Tanggal: %s (%s WIB)\n", o.TargetDate, o.NewJam))
			sb.WriteString(fmt.Sprintf("   • Ruang  : %s\n", o.Ruang))
			if o.Alasan != "" {
				sb.WriteString(fmt.Sprintf("   • Ket    : %s\n", o.Alasan))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Admin dapat membatalkan perubahan dengan `!batalganti [ID]`._")
	return sb.String()
}

// GetDateForDayName mencari tanggal untuk nama hari (cth: "Senin" dari waktu refNow)
func GetDateForDayName(dayName string, refNow time.Time) time.Time {
	clean := strings.ToLower(strings.TrimSpace(dayName))
	loc := refNow.Location()

	namaHariMap := map[string]time.Weekday{
		"senin":  time.Monday,
		"selasa": time.Tuesday,
		"rabu":   time.Wednesday,
		"kamis":  time.Thursday,
		"jumat":  time.Friday,
		"jum'at": time.Friday,
		"sabtu":  time.Saturday,
		"minggu": time.Sunday,
	}

	targetWeekday, ok := namaHariMap[clean]
	if !ok {
		return refNow
	}

	daysAhead := int(targetWeekday - refNow.Weekday())
	if daysAhead < 0 {
		daysAhead += 7
	}
	t := refNow.AddDate(0, 0, daysAhead)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func isDayName(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	days := []string{"senin", "selasa", "rabu", "kamis", "jumat", "jum'at", "sabtu", "minggu"}
	for _, d := range days {
		if strings.Contains(s, d) {
			return true
		}
	}
	return false
}

func isDatePattern(s string) bool {
	re := regexp.MustCompile(`\b\d{1,4}[-/]\d{1,2}[-/]\d{1,4}\b`)
	return re.MatchString(s)
}

// ScheduleConflict menyimpan informasi mata kuliah yang bertabrakan waktu
type ScheduleConflict struct {
	Matkul string
	Jam    string
	Ruang  string
	Dosen  string
}

// CheckScheduleConflict memeriksa apakah jam baru di tanggal tertentu bertabrakan dengan jadwal aktif lainnya
func (om *OverrideManager) CheckScheduleConflict(
	scopeJID string, targetDate time.Time, newJam string, ignoreItem *JadwalItem, cfg *JadwalConfig,
) *ScheduleConflict {
	if cfg == nil {
		return nil
	}

	propStart, propEnd, err := parseJamRange(newJam, targetDate)
	if err != nil {
		return nil
	}

	targetDateStr := targetDate.Format("2006-01-02")
	hariTarget := getHariIndonesia(targetDate)

	overrides, _ := om.GetOverridesForDate(scopeJID, targetDate)

	// 1. Cek jadwal reguler pada hari tersebut
	cfg.mu.RLock()
	var normalItems []JadwalItem
	for _, it := range cfg.Jadwal {
		if strings.EqualFold(it.Hari, hariTarget) {
			normalItems = append(normalItems, it)
		}
	}
	cfg.mu.RUnlock()

	for _, it := range normalItems {
		// Abaikan jika ini adalah sesi matkul yang sama yang sedang dipindahkan
		if ignoreItem != nil && it.KodeMatkul == ignoreItem.KodeMatkul && it.Jam == ignoreItem.Jam {
			continue
		}

		// Periksa apakah jadwal reguler ini sudah ditiadakan (CANCEL) atau dipindahkan keluar (RESCHEDULE)
		isCancelledOrMoved := false
		for _, o := range overrides {
			if o.OrigDate == targetDateStr &&
				(o.KodeMatkul == it.KodeMatkul || strings.EqualFold(o.NamaMatkul, it.NamaMatkul)) &&
				(o.OrigJam == it.Jam || strings.Contains(it.Jam, strings.TrimSpace(strings.Split(o.OrigJam, "-")[0]))) {
				if o.Type == "CANCEL" || o.Type == "RESCHEDULE" {
					isCancelledOrMoved = true
					break
				}
			}
		}
		if isCancelledOrMoved {
			continue
		}

		itStart, itEnd, err := parseJamRange(it.Jam, targetDate)
		if err != nil {
			continue
		}

		// Cek irisan waktu: [propStart, propEnd) beririsan dengan [itStart, itEnd)
		if propStart.Before(itEnd) && propEnd.After(itStart) {
			return &ScheduleConflict{
				Matkul: it.NamaMatkul,
				Jam:    it.Jam,
				Ruang:  it.Ruang,
				Dosen:  it.Dosen,
			}
		}
	}

	// 2. Cek jadwal pengganti (inbound RESCHEDULE atau EXTRA) di tanggal tersebut
	for _, o := range overrides {
		if o.TargetDate == targetDateStr && (o.Type == "RESCHEDULE" || o.Type == "EXTRA") {
			if ignoreItem != nil && o.KodeMatkul == ignoreItem.KodeMatkul {
				continue
			}

			oStart, oEnd, err := parseJamRange(o.NewJam, targetDate)
			if err != nil {
				continue
			}

			if propStart.Before(oEnd) && propEnd.After(oStart) {
				return &ScheduleConflict{
					Matkul: o.NamaMatkul,
					Jam:    o.NewJam,
					Ruang:  o.Ruang,
					Dosen:  o.Dosen,
				}
			}
		}
	}

	return nil
}

// HandleCommand memproses perintah perubahan jadwal (!pindah, !kosong, !kuliahganti, !jadwalganti, !batalganti)
func (om *OverrideManager) HandleCommand(
	scopeJID string, isGroup bool, senderJID string, isAdmin bool, rawMsg string, cfg *JadwalConfig, now time.Time,
) string {
	clean := strings.TrimSpace(rawMsg)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		clean = strings.TrimSpace(clean[1:])
	}

	parts := strings.SplitN(clean, " ", 2)
	cmd := strings.ToLower(parts[0])
	payload := ""
	if len(parts) > 1 {
		payload = strings.TrimSpace(parts[1])
	}

	switch cmd {
	case "pindah", "ganti", "reschedule":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, perubahan jadwal hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		isForce := false
		var cleanSegments []string
		for _, seg := range segments {
			trimmed := strings.TrimSpace(seg)
			if strings.EqualFold(trimmed, "paksa") || strings.EqualFold(trimmed, "force") {
				isForce = true
			} else {
				cleanSegments = append(cleanSegments, trimmed)
			}
		}
		segments = cleanSegments

		if len(segments) < 2 {
			return "⚠️ *Format Perintah Pindah Kurang Tepat*\n\n" +
				"Gunakan format pemisah pipa `|`:\n" +
				"`!pindah [Matkul] | [Hari/Tanggal & Jam Baru] | [Ruang (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!pindah aljabar | besok 13:00`\n" +
				"• `!pindah sbd | jumat 15:00 - 16:40 | Lab 312`\n" +
				"• `!pindah matdis | 10-09-2026 09:00 | D105`"
		}

		matkulQuery := strings.TrimSpace(segments[0])
		timeRaw := strings.TrimSpace(segments[1])
		ruangBaru := ""
		if len(segments) > 2 {
			ruangBaru = strings.TrimSpace(segments[2])
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				var sb strings.Builder
				sb.WriteString("⚠️ *Ditemukan beberapa sesi mata kuliah yang cocok:*\n")
				for _, c := range candidates {
					sb.WriteString(fmt.Sprintf("• [%s] %s (%s, %s)\n", c.KodeMatkul, c.NamaMatkul, c.Hari, c.Jam))
				}
				sb.WriteString("\nSilakan perjelas nama sesi (contoh: `!pindah aljabar teori | ...` atau `!pindah aljabar senin | ...`)")
				return sb.String()
			}
			return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan. Ketik `!matkul` untuk melihat daftar mata kuliah.", matkulQuery)
		}

		origDate := GetDateForDayName(item.Hari, now)
		targetDate := ParseOverrideDate(timeRaw, now)
		baseDuration := CalculateDurationInMinutes(item.Jam)
		newJam := AutoCompleteJamRange(timeRaw, baseDuration)

		// Periksa bentrok jadwal pada tanggal & jam tujuan
		conflict := om.CheckScheduleConflict(scopeJID, targetDate, newJam, item, cfg)
		if conflict != nil && !isForce {
			hariTgt := getHariIndonesia(targetDate)
			tglTgt := targetDate.Format("02-01-2006")
			var sb strings.Builder
			sb.WriteString("⚠️ *PERINGATAN BENTROK JADWAL!*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("Waktu baru yang dipilih (*%s, %s, %s WIB*) bertabrakan dengan jadwal:\n\n", hariTgt, tglTgt, newJam))
			sb.WriteString(fmt.Sprintf("• *%s*\n", conflict.Matkul))
			sb.WriteString(fmt.Sprintf("  └ Jam   : %s WIB\n", conflict.Jam))
			sb.WriteString(fmt.Sprintf("  └ Ruang : %s\n", conflict.Ruang))
			if conflict.Dosen != "" {
				sb.WriteString(fmt.Sprintf("  └ Dosen : %s\n", conflict.Dosen))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("Jadwal tidak dipindahkan untuk mencegah jadwal kuliah ganda.\n\n")
			sb.WriteString("💡 *Apakah tetap ingin memindahkan?*\n")
			sb.WriteString(fmt.Sprintf("1. Pilih jam lain yang kosong (ketik `!%s` untuk cek jam kosong).\n", strings.ToLower(hariTgt)))
			sb.WriteString("2. Jika jam tersebut memang disepakati (misal kelas tersebut ditiadakan), tambahkan kata `paksa` di akhir:\n")
			sb.WriteString(fmt.Sprintf("   `!pindah %s | %s | paksa`", matkulQuery, timeRaw))
			return sb.String()
		}

		if ruangBaru == "" {
			ruangBaru = item.Ruang
		}

		override, err := om.AddReschedule(scopeJID, *item, origDate, targetDate, newJam, ruangBaru, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menyimpan perubahan jadwal: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *JADWAL BERHASIL DIPINDAHKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Semula      : %s (%s WIB)\n", override.OrigDate, override.OrigJam))
		sb.WriteString(fmt.Sprintf("• Menjadi     : %s (%s WIB)\n", override.TargetDate, override.NewJam))
		sb.WriteString(fmt.Sprintf("• Ruang       : %s\n", override.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen       : %s (%s)\n", item.Dosen, item.InisialDosen))
		if isForce {
			sb.WriteString("• Peringatan  : ⚠️ *Jadwal dipindahkan dengan konfirmasi bentrok (dipaksa oleh Admin).*\n")
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Tips: Jadwal ini otomatis kedaluwarsa setelah tanggal lewat. Ketik `!batalganti %d` jika ingin membatalkan._", override.ID))
		return sb.String()

	case "kosong", "batal", "cancel":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, peniadaan jadwal kuliah hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		matkulQuery := strings.TrimSpace(segments[0])
		if matkulQuery == "" {
			return "⚠️ *Format Perintah Kosong Kurang Tepat*\n\n" +
				"Gunakan format:\n" +
				"`!kosong [Matkul] | [Hari/Tanggal (Opsional)] | [Alasan (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!kosong aljabar` (meniadakan jadwal terdekat)\n" +
				"• `!kosong aljabar | besok | Dosen dinas luar`"
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				var sb strings.Builder
				sb.WriteString("⚠️ *Ditemukan beberapa sesi mata kuliah yang cocok:*\n")
				for _, c := range candidates {
					sb.WriteString(fmt.Sprintf("• [%s] %s (%s, %s)\n", c.KodeMatkul, c.NamaMatkul, c.Hari, c.Jam))
				}
				sb.WriteString("\nSilakan perjelas nama sesi (contoh: `!kosong aljabar teori`).")
				return sb.String()
			}
			return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan.", matkulQuery)
		}

		targetDate := now
		if !strings.EqualFold(item.Hari, getHariIndonesia(now)) {
			targetDate = GetDateForDayName(item.Hari, now)
		}
		alasan := ""

		if len(segments) > 1 {
			val := strings.TrimSpace(segments[1])
			if strings.Contains(strings.ToLower(val), "besok") || strings.Contains(strings.ToLower(val), "hari ini") || isDayName(val) || isDatePattern(val) {
				targetDate = ParseOverrideDate(val, now)
				if len(segments) > 2 {
					alasan = strings.TrimSpace(segments[2])
				}
			} else {
				alasan = val
			}
		}

		override, err := om.AddCancel(scopeJID, *item, targetDate, alasan, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal meniadakan jadwal: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *KULIAH DITANDAI DITIADAKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Tanggal     : %s (%s WIB)\n", override.TargetDate, override.OrigJam))
		if override.Alasan != "" {
			sb.WriteString(fmt.Sprintf("• Keterangan  : %s\n", override.Alasan))
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Jadwal pada tanggal tersebut akan dicoret. Ketik `!batalganti %d` untuk mengaktifkan kembali._", override.ID))
		return sb.String()

	case "libur", "holiday":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, pengumuman libur harian hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		dayOrDate := strings.TrimSpace(segments[0])
		if dayOrDate == "" {
			return "⚠️ *Format Perintah Libur Kurang Tepat*\n\n" +
				"Gunakan format pemisah pipa `|`:\n" +
				"`!libur [Hari/Tanggal] | [Keterangan/Nama Libur]`\n\n" +
				"*Contoh:*\n" +
				"• `!libur besok | Hari Kemerdekaan RI`\n" +
				"• `!libur senin | Libur Nasional Maulid Nabi`\n" +
				"• `!libur 17-08-2026 | HUT RI`"
		}

		alasan := "Libur Perkuliahan"
		if len(segments) > 1 {
			if trimmed := strings.TrimSpace(segments[1]); trimmed != "" {
				alasan = trimmed
			}
		}

		targetDate := ParseOverrideDate(dayOrDate, now)

		override, err := om.AddHoliday(scopeJID, targetDate, alasan, senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menetapkan hari libur: %v", err)
		}

		hariTgt := getHariIndonesia(targetDate)
		tglTgt := targetDate.Format("02-01-2006")

		var sb strings.Builder
		sb.WriteString("🌴 *PENGUMUMAN LIBUR BERHASIL DITETAPKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan : #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Tanggal Libur: %s, %s (Seharian)\n", hariTgt, tglTgt))
		sb.WriteString(fmt.Sprintf("• Keterangan   : %s\n", override.Alasan))
		sb.WriteString("──────────\n")
		sb.WriteString("✨ Seluruh perkuliahan pada hari tersebut otomatis ditiadakan.\n")
		sb.WriteString("⏰ Pengingat pagi pukul 06:30 WIB otomatis mengirimkan ucapan selamat libur.\n")
		sb.WriteString(fmt.Sprintf("_Ketik `!batalganti %d` jika ingin membatalkan status libur._", override.ID))
		return sb.String()

	case "kuliahganti", "tambahkelas", "extraclass":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penambahan kuliah pengganti hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
		isForce := false
		var cleanSegments []string
		for _, seg := range segments {
			trimmed := strings.TrimSpace(seg)
			if strings.EqualFold(trimmed, "paksa") || strings.EqualFold(trimmed, "force") {
				isForce = true
			} else {
				cleanSegments = append(cleanSegments, trimmed)
			}
		}
		segments = cleanSegments

		if len(segments) < 2 {
			return "⚠️ *Format Kuliah Pengganti Kurang Tepat*\n\n" +
				"Gunakan format:\n" +
				"`!kuliahganti [Matkul] | [Hari/Tanggal & Jam] | [Ruang (Opsional)]`\n\n" +
				"*Contoh:*\n" +
				"• `!kuliahganti matdis | sabtu 09:00 - 11:30 | D105`\n" +
				"• `!kuliahganti sbd | sabtu 13:00 | Lab 312`"
		}

		matkulQuery := strings.TrimSpace(segments[0])
		timeRaw := strings.TrimSpace(segments[1])
		ruangBaru := ""
		if len(segments) > 2 {
			ruangBaru = strings.TrimSpace(segments[2])
		}

		item, candidates := cfg.FindMataKuliah(matkulQuery, now)
		if item == nil {
			if len(candidates) > 1 {
				item = &candidates[0] // default ke kandidat pertama
			} else {
				return fmt.Sprintf("❌ Mata kuliah *\"%s\"* tidak ditemukan.", matkulQuery)
			}
		}

		targetDate := ParseOverrideDate(timeRaw, now)
		baseDuration := CalculateDurationInMinutes(item.Jam)
		newJam := AutoCompleteJamRange(timeRaw, baseDuration)

		// Periksa bentrok jadwal pada tanggal & jam kuliah pengganti
		conflict := om.CheckScheduleConflict(scopeJID, targetDate, newJam, nil, cfg)
		if conflict != nil && !isForce {
			hariTgt := getHariIndonesia(targetDate)
			tglTgt := targetDate.Format("02-01-2006")
			var sb strings.Builder
			sb.WriteString("⚠️ *PERINGATAN BENTROK JADWAL!*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("Waktu kuliah pengganti (*%s, %s, %s WIB*) bertabrakan dengan jadwal:\n\n", hariTgt, tglTgt, newJam))
			sb.WriteString(fmt.Sprintf("• *%s*\n", conflict.Matkul))
			sb.WriteString(fmt.Sprintf("  └ Jam   : %s WIB\n", conflict.Jam))
			sb.WriteString(fmt.Sprintf("  └ Ruang : %s\n", conflict.Ruang))
			if conflict.Dosen != "" {
				sb.WriteString(fmt.Sprintf("  └ Dosen : %s\n", conflict.Dosen))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("Kuliah pengganti tidak ditambahkan untuk mencegah jadwal kuliah ganda.\n\n")
			sb.WriteString("💡 *Apakah tetap ingin menambahkan?*\n")
			sb.WriteString(fmt.Sprintf("1. Pilih jam lain yang kosong (ketik `!%s` untuk cek jam kosong).\n", strings.ToLower(hariTgt)))
			sb.WriteString("2. Jika jam tersebut memang disepakati, tambahkan kata `paksa` di akhir:\n")
			sb.WriteString(fmt.Sprintf("   `!kuliahganti %s | %s | paksa`", matkulQuery, timeRaw))
			return sb.String()
		}

		if ruangBaru == "" {
			ruangBaru = item.Ruang
		}

		override, err := om.AddExtra(scopeJID, *item, targetDate, newJam, ruangBaru, "Kuliah Pengganti", senderJID)
		if err != nil {
			return fmt.Sprintf("❌ Gagal menambahkan kuliah pengganti: %v", err)
		}

		var sb strings.Builder
		sb.WriteString("✅ *KULIAH PENGGANTI DITAMBAHKAN*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("• ID Perubahan: #%d\n", override.ID))
		sb.WriteString(fmt.Sprintf("• Matkul      : %s\n", override.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Tanggal     : %s (%s WIB)\n", override.TargetDate, override.NewJam))
		sb.WriteString(fmt.Sprintf("• Ruang       : %s\n", override.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen       : %s (%s)\n", item.Dosen, item.InisialDosen))
		if isForce {
			sb.WriteString("• Peringatan  : ⚠️ *Kuliah pengganti ditambahkan dengan konfirmasi bentrok (dipaksa oleh Admin).*\n")
		}
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Bot akan otomatis menyertakan jadwal ini pada pengingat. Ketik `!batalganti %d` untuk menghapus._", override.ID))
		return sb.String()

	case "jadwalganti", "overrides", "listganti":
		list, err := om.GetActiveOverrides(scopeJID, now)
		if err != nil {
			return fmt.Sprintf("❌ Gagal memuat daftar jadwal pengganti: %v", err)
		}
		return om.FormatActiveOverrides(list)

	case "batalganti", "hapusganti", "rmganti":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, pembatalan jadwal pengganti hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		id, err := strconv.Atoi(payload)
		if err != nil || id <= 0 {
			return "⚠️ Sertakan ID perubahan yang ingin dibatalkan.\nContoh: `!batalganti 1`\n\nKetik `!jadwalganti` untuk melihat daftar ID perubahan aktif."
		}

		ok, err := om.CancelOverride(scopeJID, id)
		if err != nil {
			return fmt.Sprintf("❌ Gagal membatalkan jadwal pengganti: %v", err)
		}
		if !ok {
			return fmt.Sprintf("ℹ️ Jadwal pengganti dengan ID #%d tidak ditemukan.", id)
		}

		return fmt.Sprintf("🎉 *JADWAL PENGGANTI DIBATALKAN*\nPerubahan dengan ID #%d telah dihapus. Jadwal pada tanggal tersebut kembali normal.", id)

	default:
		var sb strings.Builder
		sb.WriteString("📖 *PANDUAN JADWAL PENGGANTI (OVERRIDE)*\n")
		sb.WriteString("──────────\n\n")
		sb.WriteString("• `!pindah [Matkul] | [Waktu Baru] | [Ruang]`\n")
		sb.WriteString("  ➔ Memindahkan jam/hari kuliah sementara\n")
		sb.WriteString("  _Cth: `!pindah aljabar | besok 13:00 | Lab 312`_\n\n")
		sb.WriteString("• `!kosong [Matkul] | [Hari/Tanggal] | [Alasan]`\n")
		sb.WriteString("  ➔ Menandai kuliah ditiadakan/kosong sementara\n")
		sb.WriteString("  _Cth: `!kosong sbd | besok | Dosen dinas luar`_\n\n")
		sb.WriteString("• `!kuliahganti [Matkul] | [Waktu] | [Ruang]`\n")
		sb.WriteString("  ➔ Menambah kuliah pengganti di hari lain\n")
		sb.WriteString("  _Cth: `!kuliahganti matdis | sabtu 09:00 | D105`_\n\n")
		sb.WriteString("• `!jadwalganti`\n")
		sb.WriteString("  ➔ Melihat daftar perubahan jadwal aktif\n\n")
		sb.WriteString("• `!batalganti [ID]`\n")
		sb.WriteString("  ➔ Membatalkan perubahan (kembali ke normal)\n")
		sb.WriteString("──────────\n")
		return sb.String()
	}
}
