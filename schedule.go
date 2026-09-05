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
	mu         sync.RWMutex
	FilePath   string            `json:"-"`
	Kampus     string            `json:"kampus"`
	Dosen      map[string]string `json:"dosen"`
	MataKuliah map[string]string `json:"mata_kuliah"`
	Jadwal     []JadwalItem      `json:"jadwal"`
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

// getHariIndonesia mengonversi nama hari time.Weekday ke bahasa Indonesia
func getHariIndonesia(t time.Time) string {
	switch t.Weekday() {
	case time.Monday:
		return "Senin"
	case time.Tuesday:
		return "Selasa"
	case time.Wednesday:
		return "Rabu"
	case time.Thursday:
		return "Kamis"
	case time.Friday:
		return "Jumat"
	case time.Saturday:
		return "Sabtu"
	case time.Sunday:
		return "Minggu"
	default:
		return ""
	}
}

// getBulanIndonesia mengonversi bulan ke singkatan bahasa Indonesia yang ringkas
func getBulanIndonesia(t time.Time) string {
	bulan := []string{
		"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
		"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
	}
	return bulan[t.Month()]
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

func parseJamRange(jamStr string, refDate time.Time) (time.Time, time.Time, error) {
	parts := strings.Split(jamStr, "-")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("format jam tidak valid: %s", jamStr)
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])

	startTime, err := time.ParseInLocation("15:04", startStr, refDate.Location())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	endTime, err := time.ParseInLocation("15:04", endStr, refDate.Location())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	start := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), startTime.Hour(), startTime.Minute(), 0, 0, refDate.Location())
	end := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), endTime.Hour(), endTime.Minute(), 0, 0, refDate.Location())
	return start, end, nil
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

// GetByHari mencari jadwal berdasarkan hari tertentu, termasuk alias 'hari ini' dan 'besok'
func (j *JadwalConfig) GetByHari(hariInput string) string {
	j.mu.RLock()
	defer j.mu.RUnlock()

	hariInput = strings.ToLower(strings.TrimSpace(hariInput))
	waktuSekarang := time.Now()

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

// ProcessMessage memproses pesan masuk dan mengembalikan pesan balasan yang sesuai.
// Menggunakan strategi Hybrid: jika pesan berasal dari grup (isGroup = true), pesan WAJIB diawali prefix (!, /, #)
// agar tidak mengganggu obrolan umum. Di personal chat (DM), pengguna bebas mengetik tanpa prefix.
func (j *JadwalConfig) ProcessMessage(rawMsg string, isGroup ...bool) string {
	trimmed := strings.TrimSpace(rawMsg)
	if trimmed == "" {
		return ""
	}

	inGroup := false
	if len(isGroup) > 0 && isGroup[0] {
		inGroup = true
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
		return j.GetNextClass(time.Now())
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
		return j.GetByHari("hari ini")
	}
	if lower == "besok" || lower == "tomorrow" {
		return j.GetByHari("besok")
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
		return j.GetByHari(targetHari)
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
		return j.GetByHari(arg)
	}

	// 9. Perintah Dosen (misal: "!dosen MR", "dosen Rizqi")
	if strings.HasPrefix(lower, "dosen") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.SearchDosen(arg)
	}

	// 10. Perintah Ruang / Ruangan / Lab (misal: "!ruang D105", "!lab")
	if strings.HasPrefix(lower, "ruang") || strings.HasPrefix(lower, "ruangan") || strings.HasPrefix(lower, "lab") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
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
			arg = parts[1]
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
	sb.WriteString("• `!seminggu` ➔ Jadwal Senin - Jumat\n\n")

	sb.WriteString("🔍 *Pencarian Cepat:*\n")
	sb.WriteString("• `!matkul` ➔ Daftar semua mata kuliah\n")
	sb.WriteString("• `!dosen [nama]` ➔ Cth: `!dosen MR`\n")
	sb.WriteString("• `!ruang [kode]` ➔ Cth: `!ruang lab`\n")
	sb.WriteString("• `!cari [kata]` ➔ Cth: `!cari basis`\n\n")

	sb.WriteString("⚙️ *Lainnya:*\n")
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

	sb.WriteString("4️⃣ *Informasi & Pencarian:*\n")
	sb.WriteString("• `!matkul` ➔ Daftar semua mata kuliah\n")
	sb.WriteString("• `!dosen MR` ➔ Cari jadwal dosen inisial/nama\n")
	sb.WriteString("• `!ruang lab` ➔ Cari jadwal ruangan\n")
	sb.WriteString("• `!cari basis` ➔ Pencarian kata kunci global\n\n")

	sb.WriteString("5️⃣ *Pengaturan Admin:*\n")
	sb.WriteString("• `!reminder on` / `!reminder off` (grup)\n")
	sb.WriteString("• `!reminder test` ➔ Simulasi pengingat pagi\n")
	sb.WriteString("• `!reload` ➔ Segarkan data jadwal.json\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Ketik !menu untuk menu utama ringkas._")
	return sb.String()
}

