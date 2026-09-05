package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// JadwalItem merepresentasikan satu sesi perkuliahan dalam jadwal
type JadwalItem struct {
	Hari         string `json:"hari"`
	Jam          string `json:"jam"`
	KodeMatkul   string `json:"kode_matkul"`
	NamaMatkul   string `json:"nama_matkul"`
	InisialDosen string `json:"inisial_dosen"`
	Dosen        string `json:"dosen"`
	Ruang        string `json:"ruang"`
}

// JadwalConfig merepresentasikan struktur keseluruhan file jadwal.json
type JadwalConfig struct {
	mu              sync.RWMutex
	FilePath        string            `json:"-"`
	OverrideManager *OverrideManager  `json:"-"`
	Kampus          string            `json:"kampus"`
	Dosen           map[string]string `json:"dosen"`
	MataKuliah      map[string]string `json:"mata_kuliah"`
	Jadwal          []JadwalItem      `json:"jadwal"`
}

// SetOverrideManager menghubungkan pengelola jadwal pengganti ke JadwalConfig
func (j *JadwalConfig) SetOverrideManager(om *OverrideManager) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.OverrideManager = om
}

// LoadJadwal membaca file JSON dan mengubahnya menjadi struct JadwalConfig di memori
func LoadJadwal(filepath string) (*JadwalConfig, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file jadwal: %w", err)
	}

	var config JadwalConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("format JSON jadwal tidak valid: %w", err)
	}

	config.FilePath = filepath
	return &config, nil
}

// Reload membaca ulang file JSON dari disk tanpa perlu me-restart aplikasi
func (j *JadwalConfig) Reload() (string, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	filepath := j.FilePath
	if filepath == "" {
		filepath = "jadwal.json"
	}

	data, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("gagal membaca file jadwal: %w", err)
	}

	var temp JadwalConfig
	err = json.Unmarshal(data, &temp)
	if err != nil {
		return "", fmt.Errorf("format JSON jadwal tidak valid: %w", err)
	}

	j.Kampus = temp.Kampus
	j.Dosen = temp.Dosen
	j.MataKuliah = temp.MataKuliah
	j.Jadwal = temp.Jadwal

	return fmt.Sprintf("✅ *Data jadwal berhasil dimuat ulang!*\n• Kampus: %s\n• Total Sesi: %d jadwal mata kuliah", j.Kampus, len(j.Jadwal)), nil
}

func formatDurasi(menit int) string {
	if menit < 60 {
		return fmt.Sprintf("%d menit", menit)
	}
	jam := menit / 60
	sisaMenit := menit % 60
	if sisaMenit == 0 {
		return fmt.Sprintf("%d jam", jam)
	}
	return fmt.Sprintf("%d jam %d mnt", jam, sisaMenit)
}

// FormatList merapikan daftar item jadwal menjadi teks WhatsApp yang ringkas dan ramah mobile
func (j *JadwalConfig) FormatList(items []JadwalItem, judul string) string {
	if len(items) == 0 {
		return fmt.Sprintf("❌ *%s*\nTidak ada jadwal yang ditemukan.", judul)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *%s*\n", judul))
	sb.WriteString("──────────\n\n")

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, item.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Jam   : %s\n", item.Jam))
		sb.WriteString(fmt.Sprintf("• Ruang : %s\n", item.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen : %s (%s)\n\n", item.Dosen, item.InisialDosen))
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !menu untuk panduan._")
	return sb.String()
}

// GetJadwalSeminggu mengembalikan jadwal perkuliahan lengkap dari Senin sampai Jumat
func (j *JadwalConfig) GetJadwalSeminggu() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	hariList := []string{"Senin", "Selasa", "Rabu", "Kamis", "Jumat"}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *JADWAL SENIN - JUMAT*\n*(%s)*\n", j.Kampus))
	sb.WriteString("──────────\n\n")

	for _, hari := range hariList {
		var items []JadwalItem
		for _, item := range j.Jadwal {
			if strings.EqualFold(item.Hari, hari) {
				items = append(items, item)
			}
		}

		sb.WriteString(fmt.Sprintf("📌 *%s*\n", strings.ToUpper(hari)))
		if len(items) == 0 {
			sb.WriteString("_Tidak ada perkuliahan_\n\n")
			continue
		}

		for i, item := range items {
			sb.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, item.NamaMatkul))
			sb.WriteString(fmt.Sprintf("   • %s | %s (%s)\n", item.Jam, item.Ruang, item.InisialDosen))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !menu untuk panduan._")
	return sb.String()
}

// GetNextClass mengecek kuliah yang sedang berlangsung atau kuliah berikutnya hari ini
func (j *JadwalConfig) GetNextClass(currentTime time.Time) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	hariIndonesia := getHariIndonesia(currentTime)
	if hariIndonesia == "Sabtu" || hariIndonesia == "Minggu" {
		return fmt.Sprintf("🏖️ *HARI %s LIBUR*\nTidak ada jadwal perkuliahan hari ini.\n\nKetik `!senin` untuk melihat jadwal awal pekan.", strings.ToUpper(hariIndonesia))
	}

	var todayItems []JadwalItem
	for _, item := range j.Jadwal {
		if strings.EqualFold(item.Hari, hariIndonesia) {
			todayItems = append(todayItems, item)
		}
	}

	if len(todayItems) == 0 {
		return fmt.Sprintf("ℹ️ Tidak ada jadwal perkuliahan untuk hari %s.", hariIndonesia)
	}

	var ongoing *JadwalItem
	var ongoingEnd time.Time
	var next *JadwalItem
	var nextStart time.Time

	for _, item := range todayItems {
		start, end, err := parseJamRange(item.Jam, currentTime)
		if err != nil {
			continue
		}

		if (currentTime.Equal(start) || currentTime.After(start)) && currentTime.Before(end) {
			it := item
			ongoing = &it
			ongoingEnd = end
		} else if currentTime.Before(start) {
			if next == nil || start.Before(nextStart) {
				it := item
				next = &it
				nextStart = start
			}
		}
	}

	var sb strings.Builder
	if ongoing != nil {
		sisaMenit := int(ongoingEnd.Sub(currentTime).Minutes())
		sb.WriteString("📍 *KULIAH SEDANG BERLANGSUNG*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("*%s*\n", ongoing.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Jam   : %s (Selesai ~%d mnt lagi)\n", ongoing.Jam, sisaMenit))
		sb.WriteString(fmt.Sprintf("• Ruang : %s\n", ongoing.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen : %s (%s)\n\n", ongoing.Dosen, ongoing.InisialDosen))

		if next != nil {
			selisihMenit := int(nextStart.Sub(currentTime).Minutes())
			jamFormat := formatDurasi(selisihMenit)
			sb.WriteString("⏩ *KULIAH BERIKUTNYA:*\n")
			sb.WriteString(fmt.Sprintf("• %s\n", next.NamaMatkul))
			sb.WriteString(fmt.Sprintf("• %s (%s lagi) | %s\n", next.Jam, jamFormat, next.Ruang))
		} else {
			sb.WriteString("ℹ️ _Ini adalah mata kuliah terakhir hari ini._\n")
		}
		sb.WriteString("──────────")
		return sb.String()
	}

	if next != nil {
		selisihMenit := int(nextStart.Sub(currentTime).Minutes())
		jamFormat := formatDurasi(selisihMenit)
		sb.WriteString("📍 *KULIAH BERIKUTNYA HARI INI*\n")
		sb.WriteString("──────────\n")
		sb.WriteString(fmt.Sprintf("*%s*\n", next.NamaMatkul))
		sb.WriteString(fmt.Sprintf("• Jam   : %s (Mulai %s lagi)\n", next.Jam, jamFormat))
		sb.WriteString(fmt.Sprintf("• Ruang : %s\n", next.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen : %s (%s)\n", next.Dosen, next.InisialDosen))
		sb.WriteString("──────────\n")
		sb.WriteString("_Ketik !hari ini untuk jadwal lengkap hari ini._")
		return sb.String()
	}

	// Semua perkuliahan hari ini sudah lewat
	return fmt.Sprintf("✅ *KULIAH HARI INI SELESAI*\nSemua perkuliahan hari %s telah berakhir. Selamat istirahat!\n\nKetik `!besok` untuk melihat jadwal esok hari.", hariIndonesia)
}

// GetDaftarMatkul menampilkan seluruh mata kuliah semester ini dengan format ringkas
func (j *JadwalConfig) GetDaftarMatkul() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	type matkulInfo struct {
		Kode  string
		Nama  string
		Dosen []string
		Hari  []string
	}

	matkulMap := make(map[string]*matkulInfo)
	var listKode []string

	// Ambil dari master mata_kuliah jika ada
	for kode, nama := range j.MataKuliah {
		matkulMap[kode] = &matkulInfo{
			Kode: kode,
			Nama: nama,
		}
		listKode = append(listKode, kode)
	}

	// Petakan dosen dan hari dari daftar jadwal
	for _, item := range j.Jadwal {
		info, exists := matkulMap[item.KodeMatkul]
		if !exists {
			info = &matkulInfo{
				Kode: item.KodeMatkul,
				Nama: item.NamaMatkul,
			}
			matkulMap[item.KodeMatkul] = info
			listKode = append(listKode, item.KodeMatkul)
		}

		if item.InisialDosen != "" && !contains(info.Dosen, item.InisialDosen) {
			info.Dosen = append(info.Dosen, item.InisialDosen)
		}
		if item.Hari != "" && !contains(info.Hari, item.Hari) {
			info.Hari = append(info.Hari, item.Hari)
		}
	}

	sort.Strings(listKode)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 *DAFTAR MATA KULIAH*\n*(%s)*\n", j.Kampus))
	sb.WriteString("──────────\n\n")

	for i, kode := range listKode {
		info := matkulMap[kode]
		dosenStr := strings.Join(info.Dosen, ", ")
		if dosenStr == "" {
			dosenStr = "-"
		}
		hariStr := strings.Join(info.Hari, ", ")
		if hariStr == "" {
			hariStr = "-"
		}

		sb.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, info.Nama))
		sb.WriteString(fmt.Sprintf("• Kode  : %s\n", info.Kode))
		sb.WriteString(fmt.Sprintf("• Dosen : %s\n", dosenStr))
		sb.WriteString(fmt.Sprintf("• Hari  : %s\n\n", hariStr))
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !cari [matkul] untuk jadwal lengkap._")
	return sb.String()
}

// FormatAvailableCourses menyusun daftar ringkas mata kuliah kelas beserta kata kunci/alias inputnya
func (j *JadwalConfig) FormatAvailableCourses() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("📚 *Daftar Mata Kuliah Kelas:*\n")

	var codes []string
	for k := range j.MataKuliah {
		codes = append(codes, k)
	}
	sort.Strings(codes)

	aliasGuide := map[string]string{
		"25TI2101": "`aok` / `arsitektur`",
		"25TI2102": "`matdis` / `mtk` / `diskrit`",
		"25TI2103": "`aljabar` / `al`",
		"25TI2104": "`sbd` / `basis data`",
		"25TI2105": "`pp` / `pragmatics`",
		"25TI2106": "`so` / `os`",
		"25TI2107": "`komdat` / `jaringan`",
	}

	for i, code := range codes {
		name := j.MataKuliah[code]
		guide := aliasGuide[code]
		if guide != "" {
			sb.WriteString(fmt.Sprintf("%d. *%s* (ketik: %s)\n", i+1, name, guide))
		} else {
			sb.WriteString(fmt.Sprintf("%d. *%s*\n", i+1, name))
		}
	}
	sb.WriteString("• *Umum* (ketik: `umum` untuk tugas/kegiatan non-matkul)\n")
	return sb.String()
}


// GetByHari mencari jadwal berdasarkan hari tertentu, termasuk alias 'hari ini' dan 'besok'
func (j *JadwalConfig) GetByHari(hariInput string, refTime ...time.Time) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	hariInput = strings.ToLower(strings.TrimSpace(hariInput))
	waktuSekarang := time.Now()
	if len(refTime) > 0 && !refTime[0].IsZero() {
		waktuSekarang = refTime[0]
	}

	switch hariInput {
	case "hari ini", "hariini", "today", "now", "":
		hariInput = strings.ToLower(getHariIndonesia(waktuSekarang))
	case "besok", "tomorrow":
		hariInput = strings.ToLower(getHariIndonesia(waktuSekarang.Add(24 * time.Hour)))
	case "jum'at":
		hariInput = "jumat"
	}

	var hasil []JadwalItem
	var namaHariResmi string

	for _, item := range j.Jadwal {
		if strings.ToLower(item.Hari) == hariInput {
			hasil = append(hasil, item)
			namaHariResmi = item.Hari
		}
	}

	if namaHariResmi == "" {
		namaHariResmi = strings.Title(hariInput)
	}

	return j.FormatList(hasil, fmt.Sprintf("JADWAL %s (%s)", strings.ToUpper(namaHariResmi), j.Kampus))
}

// FindMataKuliah mencari mata kuliah dari query teks santai (alias, kode, substring)
func (j *JadwalConfig) FindMataKuliah(query string, refTime ...time.Time) (*JadwalItem, []JadwalItem) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	clean := strings.ToLower(strings.TrimSpace(query))
	if clean == "" {
		return nil, nil
	}

	aliasMap := map[string]string{
		"sbd":        "25TI2104",
		"basis data": "25TI2104",
		"basisdata":  "25TI2104",
		"matdis":     "25TI2102",
		"diskrit":    "25TI2102",
		"mtk":        "25TI2102",
		"matematika": "25TI2102",
		"aljabar":    "25TI2103",
		"al":         "25TI2103",
		"aok":        "25TI2101",
		"arsitektur": "25TI2101",
		"pp":         "25TI2105",
		"pragmatics": "25TI2105",
		"so":         "25TI2106",
		"os":         "25TI2106",
		"komdat":     "25TI2107",
		"jaringan":   "25TI2107",
	}

	targetKode := ""
	for alias, kode := range aliasMap {
		if clean == alias || strings.HasPrefix(clean, alias+" ") || strings.HasSuffix(clean, " "+alias) || strings.Contains(clean, " "+alias+" ") || (len(alias) >= 3 && strings.Contains(clean, alias)) {
			targetKode = kode
			break
		}
	}

	var candidates []JadwalItem
	for _, item := range j.Jadwal {
		if targetKode != "" && item.KodeMatkul == targetKode {
			candidates = append(candidates, item)
			continue
		}
		if strings.EqualFold(item.KodeMatkul, clean) ||
			strings.Contains(strings.ToLower(item.NamaMatkul), clean) {
			candidates = append(candidates, item)
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}
	if len(candidates) == 1 {
		return &candidates[0], candidates
	}

	// Jika kandidat > 1 (cth: Aljabar Praktikum vs Teori), cek apakah ada kata kunci penjelas
	for _, c := range candidates {
		cLower := strings.ToLower(c.NamaMatkul)
		cDay := strings.ToLower(c.Hari)
		if strings.Contains(clean, "teori") && strings.Contains(cLower, "teori") {
			return &c, candidates
		}
		if strings.Contains(clean, "praktikum") && strings.Contains(cLower, "praktikum") {
			return &c, candidates
		}
		if strings.Contains(clean, cDay) {
			return &c, candidates
		}
	}

	// Cek berdasarkan waktu acuan refTime jika diberikan
	if len(refTime) > 0 && !refTime[0].IsZero() {
		refDate := refTime[0]
		todayHari := strings.ToLower(getHariIndonesia(refDate))
		for _, c := range candidates {
			if strings.ToLower(c.Hari) == todayHari {
				return &c, candidates
			}
		}

		// Jika tidak ada di hari yang sama persis, pilih sesi terdekat yang akan datang
		type scoredCandidate struct {
			item      JadwalItem
			daysAhead int
		}
		var scored []scoredCandidate
		namaHariMap := map[string]time.Weekday{
			"senin":  time.Monday,
			"selasa": time.Tuesday,
			"rabu":   time.Wednesday,
			"kamis":  time.Thursday,
			"jumat":  time.Friday,
			"sabtu":  time.Saturday,
			"minggu": time.Sunday,
		}
		refWeekday := refDate.Weekday()
		for _, c := range candidates {
			cWeekday, ok := namaHariMap[strings.ToLower(c.Hari)]
			if !ok {
				continue
			}
			diff := int(cWeekday - refWeekday)
			if diff <= 0 {
				diff += 7
			}
			scored = append(scored, scoredCandidate{item: c, daysAhead: diff})
		}
		if len(scored) > 0 {
			sort.Slice(scored, func(i, j int) bool {
				return scored[i].daysAhead < scored[j].daysAhead
			})
			return &scored[0].item, candidates
		}
	}

	return &candidates[0], candidates
}

// RenderScheduleEntry merepresentasikan item jadwal yang diformat untuk tampilan akhir
type RenderScheduleEntry struct {
	StartTime    time.Time
	JamDisplay   string
	NamaMatkul   string
	Dosen        string
	InisialDosen string
	Ruang        string
	StatusType   string // NORMAL, CANCEL, MOVED_OUT, MOVED_IN, EXTRA
	Catatan      string
}

// GetByHariWithOverrides mencari jadwal hari tertentu dengan memperhitungkan jadwal pengganti (override)
func (j *JadwalConfig) GetByHariWithOverrides(
	hariInput string, scopeJID string, om *OverrideManager, refTime ...time.Time,
) string {
	waktuSekarang := time.Now()
	if len(refTime) > 0 && !refTime[0].IsZero() {
		waktuSekarang = refTime[0]
	}

	hariInput = strings.ToLower(strings.TrimSpace(hariInput))
	var targetDate time.Time
	switch hariInput {
	case "hari ini", "hariini", "today", "now", "":
		targetDate = waktuSekarang
	case "besok", "tomorrow":
		targetDate = waktuSekarang.Add(24 * time.Hour)
	case "jum'at":
		targetDate = GetDateForDayName("jumat", waktuSekarang)
	default:
		targetDate = GetDateForDayName(hariInput, waktuSekarang)
	}

	if om == nil || scopeJID == "" {
		return j.GetByHari(hariInput, waktuSekarang)
	}

	overrides, err := om.GetOverridesForDate(scopeJID, targetDate)
	if err != nil || len(overrides) == 0 {
		return j.GetByHari(hariInput, waktuSekarang)
	}

	// Periksa apakah seluruh hari ini dinyatakan LIBUR (HOLIDAY)
	for _, o := range overrides {
		if o.Type == "HOLIDAY" {
			var sb strings.Builder
			tglIndo := targetDate.Format("02-01-2006")
			hariIndo := getHariIndonesia(targetDate)
			sb.WriteString("🌴 *PENGUMUMAN HARI LIBUR*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("📅 *%s, %s*\n", hariIndo, tglIndo))
			sb.WriteString(fmt.Sprintf("📢 *Keterangan:* %s\n\n", o.Alasan))
			sb.WriteString("Seluruh kegiatan perkuliahan pada hari ini ditiadakan.\n")
			sb.WriteString("Selamat menikmati hari libur dan beristirahat! ✨\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("_Status libur diset oleh Admin (ID: #%d). Ketik `!batalganti %d` untuk membatalkan._", o.ID, o.ID))
			return sb.String()
		}
	}

	targetDateStr := targetDate.Format("2006-01-02")
	hariTarget := getHariIndonesia(targetDate)

	j.mu.RLock()
	var normalItems []JadwalItem
	for _, it := range j.Jadwal {
		if strings.EqualFold(it.Hari, hariTarget) {
			normalItems = append(normalItems, it)
		}
	}
	j.mu.RUnlock()

	var entries []RenderScheduleEntry

	// 1. Proses jadwal normal
	for _, it := range normalItems {
		var matchedOverride *ScheduleOverride
		for _, o := range overrides {
			if o.OrigDate == targetDateStr &&
				(o.KodeMatkul == it.KodeMatkul || strings.EqualFold(o.NamaMatkul, it.NamaMatkul)) &&
				(o.OrigJam == it.Jam || strings.Contains(it.Jam, strings.TrimSpace(strings.Split(o.OrigJam, "-")[0]))) {
				matchedOverride = &o
				break
			}
		}

		startTime, _, _ := parseJamRange(it.Jam, targetDate)

		if matchedOverride != nil {
			if matchedOverride.Type == "CANCEL" {
				entries = append(entries, RenderScheduleEntry{
					StartTime:  startTime,
					JamDisplay: it.Jam,
					NamaMatkul: it.NamaMatkul,
					StatusType: "CANCEL",
					Catatan:    matchedOverride.Alasan,
				})
			} else if matchedOverride.Type == "RESCHEDULE" {
				catatan := fmt.Sprintf("Dipindah ke: %s (%s WIB) di %s", matchedOverride.TargetDate, matchedOverride.NewJam, matchedOverride.Ruang)
				entries = append(entries, RenderScheduleEntry{
					StartTime:  startTime,
					JamDisplay: it.Jam,
					NamaMatkul: it.NamaMatkul,
					StatusType: "MOVED_OUT",
					Catatan:    catatan,
				})
			}
		} else {
			entries = append(entries, RenderScheduleEntry{
				StartTime:    startTime,
				JamDisplay:   it.Jam,
				NamaMatkul:   it.NamaMatkul,
				Dosen:        it.Dosen,
				InisialDosen: it.InisialDosen,
				Ruang:        it.Ruang,
				StatusType:   "NORMAL",
			})
		}
	}

	// 2. Proses jadwal masuk (reschedule inbound atau extra)
	for _, o := range overrides {
		if o.TargetDate == targetDateStr {
			if o.Type == "RESCHEDULE" && (o.OrigDate != targetDateStr || o.OrigJam != o.NewJam) {
				startTime, _, _ := parseJamRange(o.NewJam, targetDate)
				entries = append(entries, RenderScheduleEntry{
					StartTime:    startTime,
					JamDisplay:   o.NewJam,
					NamaMatkul:   o.NamaMatkul,
					Dosen:        o.Dosen,
					InisialDosen: o.InisialDosen,
					Ruang:        o.Ruang,
					StatusType:   "MOVED_IN",
					Catatan:      fmt.Sprintf("Kuliah pengganti dari %s", o.OrigDate),
				})
			} else if o.Type == "EXTRA" {
				startTime, _, _ := parseJamRange(o.NewJam, targetDate)
				entries = append(entries, RenderScheduleEntry{
					StartTime:    startTime,
					JamDisplay:   o.NewJam,
					NamaMatkul:   o.NamaMatkul,
					Dosen:        o.Dosen,
					InisialDosen: o.InisialDosen,
					Ruang:        o.Ruang,
					StatusType:   "EXTRA",
					Catatan:      o.Alasan,
				})
			}
		}
	}

	if len(entries) == 0 {
		return fmt.Sprintf("❌ *JADWAL %s (%s)*\nTidak ada jadwal yang ditemukan.", strings.ToUpper(hariTarget), j.Kampus)
	}

	// Urutkan entri secara kronologis berdasarkan StartTime
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].StartTime.Before(entries[j].StartTime)
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *JADWAL %s, %02d %s (%s)*\n",
		strings.ToUpper(hariTarget), targetDate.Day(), getBulanIndonesia(targetDate), j.Kampus))
	sb.WriteString("──────────\n\n")

	for i, e := range entries {
		switch e.StatusType {
		case "CANCEL":
			sb.WriteString(fmt.Sprintf("*%d. ~~%s WIB~~*\n", i+1, e.JamDisplay))
			sb.WriteString(fmt.Sprintf("   ❌ *KULIAH DITIADAKAN*\n"))
			sb.WriteString(fmt.Sprintf("   %s\n", e.NamaMatkul))
			if e.Catatan != "" {
				sb.WriteString(fmt.Sprintf("   └ ℹ️ *Alasan: %s*\n", e.Catatan))
			}
		case "MOVED_OUT":
			sb.WriteString(fmt.Sprintf("*%d. ~~%s WIB~~*\n", i+1, e.JamDisplay))
			sb.WriteString(fmt.Sprintf("   ❌ *KULIAH DIPINDAHKAN*\n"))
			sb.WriteString(fmt.Sprintf("   %s\n", e.NamaMatkul))
			sb.WriteString(fmt.Sprintf("   └ ℹ️ *%s*\n", e.Catatan))
		case "MOVED_IN":
			sb.WriteString(fmt.Sprintf("*%d. %s WIB [KULIAH PENGGANTI]*\n", i+1, e.JamDisplay))
			sb.WriteString(fmt.Sprintf("   🔄 *%s*\n", e.NamaMatkul))
			sb.WriteString(fmt.Sprintf("   • Dosen : %s (%s)\n", e.Dosen, e.InisialDosen))
			sb.WriteString(fmt.Sprintf("   • Ruang : %s\n", e.Ruang))
			sb.WriteString(fmt.Sprintf("   └ ℹ️ *%s*\n", e.Catatan))
		case "EXTRA":
			sb.WriteString(fmt.Sprintf("*%d. %s WIB [KULIAH TAMBAHAN]*\n", i+1, e.JamDisplay))
			sb.WriteString(fmt.Sprintf("   🔄 *%s*\n", e.NamaMatkul))
			sb.WriteString(fmt.Sprintf("   • Dosen : %s (%s)\n", e.Dosen, e.InisialDosen))
			sb.WriteString(fmt.Sprintf("   • Ruang : %s\n", e.Ruang))
			if e.Catatan != "" {
				sb.WriteString(fmt.Sprintf("   └ ℹ️ *%s*\n", e.Catatan))
			}
		default: // NORMAL
			sb.WriteString(fmt.Sprintf("*%d. %s*\n", i+1, e.NamaMatkul))
			sb.WriteString(fmt.Sprintf("   • Jam   : %s WIB\n", e.JamDisplay))
			sb.WriteString(fmt.Sprintf("   • Ruang : %s\n", e.Ruang))
			sb.WriteString(fmt.Sprintf("   • Dosen : %s (%s)\n", e.Dosen, e.InisialDosen))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !menu untuk panduan._")
	return sb.String()
}

// GetNextClassWithOverrides mengecek kuliah aktif/berikutnya dengan memperhitungkan jadwal pengganti
func (j *JadwalConfig) GetNextClassWithOverrides(
	currentTime time.Time, scopeJID string, om *OverrideManager,
) string {
	if om == nil || scopeJID == "" {
		return j.GetNextClass(currentTime)
	}

	overrides, err := om.GetOverridesForDate(scopeJID, currentTime)
	if err != nil || len(overrides) == 0 {
		return j.GetNextClass(currentTime)
	}

	for _, o := range overrides {
		if o.Type == "HOLIDAY" {
			return fmt.Sprintf("🌴 *Hari Ini Libur Perkuliahan*\nKeterangan: *%s*\nTidak ada kuliah aktif maupun kelas berikutnya hari ini. Selamat berlibur! ✨", o.Alasan)
		}
	}

	hariIndonesia := getHariIndonesia(currentTime)
	todayStr := currentTime.Format("2006-01-02")

	j.mu.RLock()
	var todayItems []JadwalItem
	for _, item := range j.Jadwal {
		if strings.EqualFold(item.Hari, hariIndonesia) {
			todayItems = append(todayItems, item)
		}
	}
	j.mu.RUnlock()

	type ActiveCandidate struct {
		NamaMatkul   string
		Jam          string
		Ruang        string
		Dosen        string
		InisialDosen string
		IsOverride   bool
		StartTime    time.Time
		EndTime      time.Time
	}

	var candidates []ActiveCandidate

	// Masukkan jadwal normal yang tidak dicancel/dipindah
	for _, item := range todayItems {
		isCancelledOrMoved := false
		for _, o := range overrides {
			if o.OrigDate == todayStr &&
				(o.KodeMatkul == item.KodeMatkul || strings.EqualFold(o.NamaMatkul, item.NamaMatkul)) {
				isCancelledOrMoved = true
				break
			}
		}
		if isCancelledOrMoved {
			continue
		}
		start, end, err := parseJamRange(item.Jam, currentTime)
		if err != nil {
			continue
		}
		candidates = append(candidates, ActiveCandidate{
			NamaMatkul:   item.NamaMatkul,
			Jam:          item.Jam,
			Ruang:        item.Ruang,
			Dosen:        item.Dosen,
			InisialDosen: item.InisialDosen,
			IsOverride:   false,
			StartTime:    start,
			EndTime:      end,
		})
	}

	// Masukkan jadwal masuk (inbound reschedule / extra)
	for _, o := range overrides {
		if o.TargetDate == todayStr && (o.Type == "RESCHEDULE" || o.Type == "EXTRA") {
			start, end, err := parseJamRange(o.NewJam, currentTime)
			if err != nil {
				continue
			}
			candidates = append(candidates, ActiveCandidate{
				NamaMatkul:   o.NamaMatkul,
				Jam:          o.NewJam,
				Ruang:        o.Ruang,
				Dosen:        o.Dosen,
				InisialDosen: o.InisialDosen,
				IsOverride:   true,
				StartTime:    start,
				EndTime:      end,
			})
		}
	}

	if len(candidates) == 0 {
		return fmt.Sprintf("🏖️ *TIDAK ADA KULIAH AKTIF HARI INI*\nSeluruh perkuliahan hari %s ditiadakan atau libur.", strings.ToUpper(hariIndonesia))
	}

	var ongoing *ActiveCandidate
	var next *ActiveCandidate

	for _, c := range candidates {
		cand := c
		if (currentTime.Equal(cand.StartTime) || currentTime.After(cand.StartTime)) && currentTime.Before(cand.EndTime) {
			ongoing = &cand
		} else if currentTime.Before(cand.StartTime) {
			if next == nil || cand.StartTime.Before(next.StartTime) {
				next = &cand
			}
		}
	}

	var sb strings.Builder
	if ongoing != nil {
		sisaMenit := int(ongoing.EndTime.Sub(currentTime).Minutes())
		sb.WriteString("📍 *KULIAH SEDANG BERLANGSUNG*\n")
		sb.WriteString("──────────\n")
		if ongoing.IsOverride {
			sb.WriteString(fmt.Sprintf("*[KULIAH PENGGANTI] %s*\n", ongoing.NamaMatkul))
		} else {
			sb.WriteString(fmt.Sprintf("*%s*\n", ongoing.NamaMatkul))
		}
		sb.WriteString(fmt.Sprintf("• Jam   : %s (Selesai ~%d mnt lagi)\n", ongoing.Jam, sisaMenit))
		sb.WriteString(fmt.Sprintf("• Ruang : %s\n", ongoing.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen : %s (%s)\n\n", ongoing.Dosen, ongoing.InisialDosen))

		if next != nil {
			selisihMenit := int(next.StartTime.Sub(currentTime).Minutes())
			jamFormat := formatDurasi(selisihMenit)
			sb.WriteString("⏩ *KULIAH BERIKUTNYA:*\n")
			sb.WriteString(fmt.Sprintf("• %s\n", next.NamaMatkul))
			sb.WriteString(fmt.Sprintf("• %s (%s lagi) | %s\n", next.Jam, jamFormat, next.Ruang))
		} else {
			sb.WriteString("ℹ️ _Ini adalah mata kuliah terakhir hari ini._\n")
		}
		sb.WriteString("──────────")
		return sb.String()
	}

	if next != nil {
		selisihMenit := int(next.StartTime.Sub(currentTime).Minutes())
		jamFormat := formatDurasi(selisihMenit)
		sb.WriteString("📍 *KULIAH BERIKUTNYA HARI INI*\n")
		sb.WriteString("──────────\n")
		if next.IsOverride {
			sb.WriteString(fmt.Sprintf("*[KULIAH PENGGANTI] %s*\n", next.NamaMatkul))
		} else {
			sb.WriteString(fmt.Sprintf("*%s*\n", next.NamaMatkul))
		}
		sb.WriteString(fmt.Sprintf("• Jam   : %s (Mulai %s lagi)\n", next.Jam, jamFormat))
		sb.WriteString(fmt.Sprintf("• Ruang : %s\n", next.Ruang))
		sb.WriteString(fmt.Sprintf("• Dosen : %s (%s)\n", next.Dosen, next.InisialDosen))
		sb.WriteString("──────────\n")
		sb.WriteString("_Ketik !hari ini untuk jadwal lengkap hari ini._")
		return sb.String()
	}

	return fmt.Sprintf("✅ *KULIAH HARI INI SELESAI*\nSemua perkuliahan hari %s telah berakhir. Selamat istirahat!\n\nKetik `!besok` untuk melihat jadwal esok hari.", hariIndonesia)
}

// SearchDosen mencari jadwal berdasarkan inisial atau nama dosen
func (j *JadwalConfig) SearchDosen(keyword string) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Sertakan nama/kode dosen.\nContoh: `!dosen MR` atau `!dosen Rizqi`"
	}

	var hasil []JadwalItem
	for _, item := range j.Jadwal {
		if strings.Contains(strings.ToLower(item.InisialDosen), keyword) ||
			strings.Contains(strings.ToLower(item.Dosen), keyword) {
			hasil = append(hasil, item)
		}
	}

	if len(hasil) > 0 {
		return j.FormatList(hasil, fmt.Sprintf("DOSEN: \"%s\"", strings.ToUpper(keyword)))
	}

	// Cek apakah dosen terdaftar di master data jurusan meskipun tidak mengajar di kelas ini
	for id, nama := range j.Dosen {
		if strings.ToLower(id) == keyword || strings.Contains(strings.ToLower(nama), keyword) {
			var sb strings.Builder
			sb.WriteString("👨‍🏫 *INFO DOSEN JTK*\n")
			sb.WriteString("──────────\n")
			sb.WriteString(fmt.Sprintf("• Nama : %s\n", nama))
			sb.WriteString(fmt.Sprintf("• Kode : %s\n\n", id))
			sb.WriteString("ℹ️ _Tidak ada jadwal mengajar di kelas ini._\n")
			sb.WriteString("──────────")
			return sb.String()
		}
	}

	return fmt.Sprintf("❌ *DOSEN: \"%s\"*\nData tidak ditemukan di daftar dosen maupun jadwal.", keyword)
}

// SearchRuangan mencari jadwal di ruangan tertentu
func (j *JadwalConfig) SearchRuangan(keyword string) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Sertakan nama/kode ruangan.\nContoh: `!ruang Lab` atau `!ruang D105`"
	}

	var hasil []JadwalItem
	for _, item := range j.Jadwal {
		if strings.Contains(strings.ToLower(item.Ruang), keyword) {
			hasil = append(hasil, item)
		}
	}

	return j.FormatList(hasil, fmt.Sprintf("RUANGAN: \"%s\"", strings.ToUpper(keyword)))
}

// SearchGlobal mencari jadwal berdasarkan kata kunci apa pun (matkul, dosen, ruangan, kode)
func (j *JadwalConfig) SearchGlobal(keyword string) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Sertakan kata kunci pencarian.\nContoh: `!cari basis data`"
	}

	var hasil []JadwalItem
	for _, item := range j.Jadwal {
		match := strings.Contains(strings.ToLower(item.NamaMatkul), keyword) ||
			strings.Contains(strings.ToLower(item.KodeMatkul), keyword) ||
			strings.Contains(strings.ToLower(item.Dosen), keyword) ||
			strings.Contains(strings.ToLower(item.InisialDosen), keyword) ||
			strings.Contains(strings.ToLower(item.Ruang), keyword) ||
			strings.Contains(strings.ToLower(item.Hari), keyword)

		if match {
			hasil = append(hasil, item)
		}
	}

	return j.FormatList(hasil, fmt.Sprintf("PENCARIAN: \"%s\"", strings.ToUpper(keyword)))
}

// ProcessMessage memproses pesan masuk dan mencocokkan dengan perintah jadwal yang tersedia.
// Menerapkan strategi Hybrid: di grup chat, pesan harus diawali prefix (! / / / #)
// agar tidak mengganggu obrolan umum. Di personal chat (DM), pengguna bebas mengetik tanpa prefix.
func (j *JadwalConfig) ProcessMessage(rawMsg string, opts ...any) string {
	trimmed := strings.TrimSpace(rawMsg)
	if trimmed == "" {
		return ""
	}

	inGroup := false
	scopeJID := ""
	now := time.Now()

	for _, opt := range opts {
		switch v := opt.(type) {
		case bool:
			inGroup = v
		case string:
			scopeJID = v
		case time.Time:
			now = v
		}
	}

	hasPrefix := false
	clean := trimmed

	// Deteksi prefix perintah (! / / / #)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		hasPrefix = true
		clean = strings.TrimSpace(clean[1:])
	}

	// Aturan Grup (Strategi Hybrid): Abaikan pesan di grup jika tidak diawali prefix perintah
	if inGroup && !hasPrefix {
		return ""
	}

	lower := strings.ToLower(clean)

	// 1. Menu Utama & Panduan Keyword
	if lower == "menu" {
		return j.GetMenu()
	}
	if lower == "keyword" || lower == "keywords" || lower == "help" || lower == "bantuan" || lower == "panduan" {
		return j.GetKeywords()
	}

	// 2. Kuliah Berikutnya / Sedang Berlangsung
	if lower == "next" || lower == "sekarang" || lower == "kuliah" || lower == "kuliah berikutnya" || lower == "ongoing" {
		if j.OverrideManager != nil && scopeJID != "" {
			return j.GetNextClassWithOverrides(now, scopeJID, j.OverrideManager)
		}
		return j.GetNextClass(now)
	}

	// 3. Daftar Mata Kuliah
	if lower == "matkul" || lower == "matakuliah" || lower == "mata kuliah" || lower == "daftar matkul" || lower == "daftarmatkul" {
		return j.GetDaftarMatkul()
	}

	// 4. Reload Data Jadwal dari Disk
	if lower == "reload" || lower == "refresh" || lower == "update" {
		res, err := j.Reload()
		if err != nil {
			return fmt.Sprintf("❌ Gagal reload data: %v", err)
		}
		return res
	}

	// 5. Jadwal Seminggu / Senin - Jumat
	switch lower {
	case "seminggu", "senin-jumat", "senin - jumat", "senin jumat", "senin_jumat",
		"sepekan", "pekan ini", "minggu ini", "all", "semua", "full",
		"jadwal semua", "jadwal seminggu", "jadwal full", "jadwal senin jumat", "jadwal senin-jumat":
		return j.GetJadwalSeminggu()
	}

	// 6. Pintasan Waktu Cepat (Hari Ini & Besok)
	if lower == "hari ini" || lower == "hariini" || lower == "today" || lower == "now" {
		if j.OverrideManager != nil && scopeJID != "" {
			return j.GetByHariWithOverrides("hari ini", scopeJID, j.OverrideManager, now)
		}
		return j.GetByHari("hari ini", now)
	}
	if lower == "besok" || lower == "tomorrow" {
		if j.OverrideManager != nil && scopeJID != "" {
			return j.GetByHariWithOverrides("besok", scopeJID, j.OverrideManager, now)
		}
		return j.GetByHari("besok", now)
	}

	// 7. Pintasan Nama Hari Langsung (misal: "!senin", "senin", "!jumat")
	namaHari := map[string]string{
		"senin":     "senin",
		"monday":    "senin",
		"selasa":    "selasa",
		"tuesday":   "selasa",
		"rabu":      "rabu",
		"wednesday": "rabu",
		"kamis":     "kamis",
		"thursday":  "kamis",
		"jumat":     "jumat",
		"jum'at":    "jumat",
		"friday":    "jumat",
		"sabtu":     "sabtu",
		"saturday":  "sabtu",
		"minggu":    "minggu",
		"sunday":    "minggu",
	}

	if targetHari, ok := namaHari[lower]; ok {
		if j.OverrideManager != nil && scopeJID != "" {
			return j.GetByHariWithOverrides(targetHari, scopeJID, j.OverrideManager, now)
		}
		return j.GetByHari(targetHari, now)
	}

	// 8. Perintah Jadwal Lengkap (misal: "!jadwal", "!jadwal senin", "!jadwal besok", "!jadwal seminggu")
	if strings.HasPrefix(lower, "jadwal") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = strings.TrimSpace(parts[1])
		}
		lowerArg := strings.ToLower(arg)
		if lowerArg == "semua" || lowerArg == "seminggu" || lowerArg == "full" || lowerArg == "all" ||
			lowerArg == "senin-jumat" || lowerArg == "senin jumat" || lowerArg == "senin - jumat" {
			return j.GetJadwalSeminggu()
		}
		if j.OverrideManager != nil && scopeJID != "" {
			return j.GetByHariWithOverrides(arg, scopeJID, j.OverrideManager, now)
		}
		return j.GetByHari(arg, now)
	}

	// 9. Perintah Dosen (misal: "!dosen MR", "dosen Rizqi")
	if strings.HasPrefix(lower, "dosen") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = strings.TrimSpace(parts[1])
		}
		return j.SearchDosen(arg)
	}

	// 10. Perintah Ruang / Ruangan / Lab (misal: "!ruang D105", "!lab")
	if strings.HasPrefix(lower, "ruang") || strings.HasPrefix(lower, "ruangan") || strings.HasPrefix(lower, "lab") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = strings.TrimSpace(parts[1])
		}
		if arg == "" && (lower == "lab" || lower == "ruang" || lower == "ruangan") {
			return j.SearchRuangan("lab")
		}
		return j.SearchRuangan(arg)
	}

	// 11. Perintah Cari Global (misal: "!cari sistem operasi")
	if strings.HasPrefix(lower, "cari") || strings.HasPrefix(lower, "search") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = strings.TrimSpace(parts[1])
		}
		return j.SearchGlobal(arg)
	}

	// 12. Fallback jika diawali prefix perintah tetapi tidak dikenali
	if hasPrefix {
		return fmt.Sprintf("⚠️ Perintah *\"%s\"* tidak dikenali.\n\nKetik *!menu* untuk melihat panduan perintah.", trimmed)
	}

	return ""
}

// GetMenu mengembalikan menu utama yang ringkas, bersih, dan nyaman dibaca di layar HP
func (j *JadwalConfig) GetMenu() string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	now := time.Now()
	hariStr := getHariIndonesia(now)
	tanggalStr := fmt.Sprintf("%s, %d %s", hariStr, now.Day(), getBulanIndonesia(now))

	// Hitung ringkasan jumlah matkul hari ini
	todayDayLower := strings.ToLower(hariStr)
	var todayCount int
	for _, item := range j.Jadwal {
		if strings.ToLower(item.Hari) == todayDayLower {
			todayCount++
		}
	}

	var statusHariIni string
	if todayCount > 0 {
		statusHariIni = fmt.Sprintf("%d Kuliah", todayCount)
	} else {
		statusHariIni = "Libur"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("*JADWAL KULIAH (%s)*\n", j.Kampus))
	sb.WriteString(fmt.Sprintf("%s • _%s_\n", tanggalStr, statusHariIni))
	sb.WriteString("──────────\n\n")

	sb.WriteString("📌 *Paling Sering Digunakan:*\n")
	sb.WriteString("• `!next` ➔ Kuliah sedang/berikutnya\n")
	sb.WriteString("• `!hari ini` ➔ Jadwal hari ini\n")
	sb.WriteString("• `!besok` ➔ Jadwal besok\n")
	sb.WriteString("• `!seminggu` ➔ Jadwal Senin - Jumat\n")
	sb.WriteString("• `!tugas` ➔ Catatan tugas & deadline\n\n")

	sb.WriteString("🔄 *Jadwal Pengganti (Admin):*\n")
	sb.WriteString("• `!pindah [matkul] | [waktu]` ➔ Geser jadwal\n")
	sb.WriteString("• `!kosong [matkul]` ➔ Kelas ditiadakan\n")
	sb.WriteString("• `!jadwalganti` ➔ Cek perubahan aktif\n\n")

	sb.WriteString("🔍 *Pencarian Cepat:*\n")
	sb.WriteString("• `!matkul` ➔ Daftar semua mata kuliah\n")
	sb.WriteString("• `!dosen [nama]` ➔ Cth: `!dosen MR`\n")
	sb.WriteString("• `!ruang [kode]` ➔ Cth: `!ruang lab`\n")
	sb.WriteString("• `!cari [kata]` ➔ Cth: `!cari basis`\n\n")

	sb.WriteString("⚙️ *Pengaturan & Kelas:*\n")
	sb.WriteString("• `!kelas` ➔ Daftar pilihan & kelas aktif\n")
	sb.WriteString("• `!setkelas [nama]` ➔ Pilih kelas (Admin)\n")
	sb.WriteString("• `!reminder on/off` ➔ Pengingat pagi 06:30\n")
	sb.WriteString("• `!keyword` ➔ Panduan semua kata kunci\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Di chat pribadi bisa tanpa tanda '!'_")
	return sb.String()
}

// GetKeywords mengembalikan daftar lengkap seluruh kata kunci dan panduan perintah bot
func (j *JadwalConfig) GetKeywords() string {
	var sb strings.Builder
	sb.WriteString("📖 *DAFTAR LENGKAP KEYWORD*\n")
	sb.WriteString("──────────\n\n")

	sb.WriteString("1️⃣ *Jadwal Harian:*\n")
	sb.WriteString("• `!senin` `!selasa` `!rabu` `!kamis` `!jumat`\n")
	sb.WriteString("• Alias: `!today`, `!now`, `!tomorrow`, `!hari ini`, `!besok`\n\n")

	sb.WriteString("2️⃣ *Jadwal Pekanan:*\n")
	sb.WriteString("• `!seminggu` / `!senin-jumat` / `!semua`\n\n")

	sb.WriteString("3️⃣ *Kuliah Sedang/Berikutnya:*\n")
	sb.WriteString("• `!next` / `!sekarang`\n\n")

	sb.WriteString("4️⃣ *Tugas & Deadline:*\n")
	sb.WriteString("• `!tugas` ➔ Lihat daftar tugas aktif\n")
	sb.WriteString("• `!tugas sbd` ➔ Filter tugas per mata kuliah\n")
	sb.WriteString("• `!tugas riwayat` ➔ Rekam jejak tugas selesai\n")
	sb.WriteString("• `!tugas tambah SBD | Lapres | Jumat 23:59`\n")
	sb.WriteString("• `!tugas edit [ID] | Minggu 23:59` ➔ Ubah tenggat\n")
	sb.WriteString("• `!tugas selesai [ID]` ➔ Selesaikan tugas\n")
	sb.WriteString("• `!tugas hapus [ID]` ➔ Hapus tugas\n\n")

	sb.WriteString("5️⃣ *Jadwal Pengganti (Khusus Admin):*\n")
	sb.WriteString("• `!libur besok | Hari Kemerdekaan RI` ➔ Libur seharian\n")
	sb.WriteString("• `!pindah aljabar | besok 13:00 | Lab 312`\n")
	sb.WriteString("• `!kosong sbd | besok | Dosen dinas luar`\n")
	sb.WriteString("• `!kuliahganti matdis | sabtu 09:00 | D105`\n")
	sb.WriteString("• `!jadwalganti` ➔ Cek perubahan jadwal aktif\n")
	sb.WriteString("• `!batalganti [ID]` ➔ Hapus jadwal pengganti\n\n")

	sb.WriteString("6️⃣ *Pengaturan Kelas (Multi-Tenant):*\n")
	sb.WriteString("• `!daftarkelas` / `!kelas` ➔ Daftar semua kelas aktif\n")
	sb.WriteString("• `!setkelas 3A` ➔ Tautkan grup ke kelas tertentu (Admin)\n")
	sb.WriteString("• `!resetkelas` ➔ Kembalikan ke kelas bawaan default (Admin)\n\n")

	sb.WriteString("7️⃣ *Informasi & Pencarian:*\n")
	sb.WriteString("• `!matkul` ➔ Daftar semua mata kuliah\n")
	sb.WriteString("• `!dosen MR` ➔ Cari jadwal dosen inisial/nama\n")
	sb.WriteString("• `!ruang lab` ➔ Cari jadwal ruangan\n")
	sb.WriteString("• `!cari basis` ➔ Pencarian kata kunci global\n\n")

	sb.WriteString("8️⃣ *Pengaturan Admin Lainnya:*\n")
	sb.WriteString("• `!reminder on` / `!reminder off` (grup)\n")
	sb.WriteString("• `!reminder test` ➔ Simulasi pengingat pagi\n")
	sb.WriteString("• `!reload` ➔ Segarkan data jadwal.json\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !menu untuk menu utama ringkas._")
	return sb.String()
}

