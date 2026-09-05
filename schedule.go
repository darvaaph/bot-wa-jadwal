package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

	return &config, nil
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

// GetByHari mencari jadwal berdasarkan hari tertentu, termasuk alias 'hari ini' dan 'besok'
func (j *JadwalConfig) GetByHari(hariInput string) string {
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

// ProcessMessage memproses pesan masuk dan mengembalikan pesan balasan yang sesuai
func (j *JadwalConfig) ProcessMessage(rawMsg string) string {
	trimmed := strings.TrimSpace(rawMsg)
	if trimmed == "" {
		return ""
	}

	hasPrefix := false
	clean := trimmed

	// Deteksi prefix perintah (! / / / #)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		hasPrefix = true
		clean = strings.TrimSpace(clean[1:])
	}

	lower := strings.ToLower(clean)

	// 1. Menu & Bantuan
	if lower == "menu" || lower == "help" || lower == "info" || lower == "bantuan" {
		return j.GetMenu()
	}

	// 2. Jadwal Seminggu / Senin - Jumat
	switch lower {
	case "seminggu", "senin-jumat", "senin - jumat", "senin jumat", "senin_jumat",
		"sepekan", "pekan ini", "minggu ini", "all", "semua", "full",
		"jadwal semua", "jadwal seminggu", "jadwal full", "jadwal senin jumat", "jadwal senin-jumat":
		return j.GetJadwalSeminggu()
	}

	// 3. Pintasan Waktu Cepat (Hari Ini & Besok)
	if lower == "hari ini" || lower == "hariini" || lower == "today" || lower == "now" {
		return j.GetByHari("hari ini")
	}
	if lower == "besok" || lower == "tomorrow" {
		return j.GetByHari("besok")
	}

	// 4. Pintasan Nama Hari Langsung (misal: "!senin", "senin", "!jumat")
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

	// 5. Perintah Jadwal Lengkap (misal: "!jadwal", "!jadwal senin", "!jadwal besok", "!jadwal seminggu")
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

	// 6. Perintah Dosen (misal: "!dosen MR", "dosen Rizqi")
	if strings.HasPrefix(lower, "dosen") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.SearchDosen(arg)
	}

	// 7. Perintah Ruang / Ruangan / Lab (misal: "!ruang D105", "!lab")
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

	// 8. Perintah Cari Global (misal: "!cari sistem operasi")
	if strings.HasPrefix(lower, "cari") || strings.HasPrefix(lower, "search") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.SearchGlobal(arg)
	}

	// 9. Fallback jika diawali prefix perintah tetapi tidak dikenali
	if hasPrefix {
		return fmt.Sprintf("⚠️ Perintah *\"%s\"* tidak dikenali.\n\nKetik *!menu* untuk melihat panduan perintah.", trimmed)
	}

	return ""
}

// GetMenu mengembalikan pesan bantuan dan daftar perintah bot format Opsi 2 (Kompak & Bersih)
func (j *JadwalConfig) GetMenu() string {
	now := time.Now()
	hariStr := getHariIndonesia(now)
	tanggalStr := fmt.Sprintf("%s, %d %s %d", hariStr, now.Day(), getBulanIndonesia(now), now.Year())

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
	sb.WriteString(fmt.Sprintf("*Jadwal Kuliah (%s)*\n", j.Kampus))
	sb.WriteString(fmt.Sprintf("%s • _%s_\n", tanggalStr, statusHariIni))
	sb.WriteString("──────────\n\n")

	sb.WriteString("*Cek Hari:*\n")
	sb.WriteString("`!hari ini`  `!besok`  `!seminggu`\n")
	sb.WriteString("`!senin` `!selasa` `!rabu` `!kamis` `!jumat`\n\n")

	sb.WriteString("*Pencarian:*\n")
	sb.WriteString("• `!dosen [nama]`  (cth: `!dosen MR`)\n")
	sb.WriteString("• `!ruang [kode]`  (cth: `!ruang lab`)\n")
	sb.WriteString("• `!cari [kata]`   (cth: `!cari basis`)\n\n")

	sb.WriteString("──────────\n")
	sb.WriteString("_Tips: Bisa diketik tanpa tanda '!'_")
	return sb.String()
}
