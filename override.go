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
	if strings.Contains(clean, "-") || strings.Contains(clean, "s/d") || strings.Contains(clean, "sampai") {
		clean = strings.ReplaceAll(clean, "s/d", "-")
		clean = strings.ReplaceAll(clean, "sampai", "-")
		p := strings.SplitN(clean, "-", 2)
		return fmt.Sprintf("%s - %s", strings.TrimSpace(p[0]), strings.TrimSpace(p[1]))
	}

	// Hanya jam mulai
	timeRe := regexp.MustCompile(`\b([01]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)
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
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("_Tips: Jadwal ini otomatis kedaluwarsa setelah tanggal lewat. Ketik `!batalganti %d` jika ingin membatalkan._", override.ID))
		return sb.String()

	case "kosong", "batal", "libur", "cancel":
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

	case "kuliahganti", "tambahkelas", "extraclass":
		if isGroup && !isAdmin {
			return "🔒 *Akses Ditolak*\nDi grup kelas, penambahan kuliah pengganti hanya dapat dilakukan oleh *Admin Grup* (Komti/Wakil)."
		}

		segments := strings.Split(payload, "|")
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
