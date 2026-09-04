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

// getBulanIndonesia mengonversi bulan ke bahasa Indonesia
func getBulanIndonesia(t time.Time) string {
	bulan := []string{
		"", "Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	return bulan[t.Month()]
}

// FormatList merapikan daftar item jadwal menjadi teks WhatsApp yang estetik dan mudah dibaca
func (j *JadwalConfig) FormatList(items []JadwalItem, judul string) string {
	if len(items) == 0 {
		return fmt.Sprintf("❌ *%s*\nTidak ada jadwal yang ditemukan.", judul)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *%s*\n", judul))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("*%d. %s* (%s)\n", i+1, item.NamaMatkul, item.KodeMatkul))
		sb.WriteString(fmt.Sprintf("   ⏰ Waktu  : %s, %s\n", item.Hari, item.Jam))
		sb.WriteString(fmt.Sprintf("   📍 Ruang  : %s\n", item.Ruang))
		sb.WriteString(fmt.Sprintf("   👨‍🏫 Dosen  : %s (%s)\n\n", item.Dosen, item.InisialDosen))
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("_Ketik *!menu* untuk melihat daftar perintah._")
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

	return j.FormatList(hasil, fmt.Sprintf("JADWAL HARI %s", strings.ToUpper(namaHariResmi)))
}

// SearchDosen mencari jadwal berdasarkan inisial atau nama dosen
func (j *JadwalConfig) SearchDosen(keyword string) string {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Mohon sertakan nama atau kode dosen.\nContoh: `!dosen MR` atau `!dosen Rizqi`"
	}

	var hasil []JadwalItem
	for _, item := range j.Jadwal {
		if strings.Contains(strings.ToLower(item.InisialDosen), keyword) ||
			strings.Contains(strings.ToLower(item.Dosen), keyword) {
			hasil = append(hasil, item)
		}
	}

	if len(hasil) > 0 {
		return j.FormatList(hasil, fmt.Sprintf("HASIL PENCARIAN DOSEN: \"%s\"", strings.ToUpper(keyword)))
	}

	// Cek apakah dosen terdaftar di master data jurusan meskipun tidak mengajar di kelas ini
	for id, nama := range j.Dosen {
		if strings.ToLower(id) == keyword || strings.Contains(strings.ToLower(nama), keyword) {
			var sb strings.Builder
			sb.WriteString("👨‍🏫 *INFORMASI DOSEN JTK*\n")
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
			sb.WriteString(fmt.Sprintf("• *Nama*: %s\n", nama))
			sb.WriteString(fmt.Sprintf("• *ID / Inisial*: %s\n\n", id))
			sb.WriteString("ℹ️ _Dosen terdaftar di JTK, namun tidak memiliki jadwal perkuliahan di kelas ini._\n")
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
			sb.WriteString("_Ketik *!menu* untuk melihat daftar perintah._")
			return sb.String()
		}
	}

	return fmt.Sprintf("❌ *HASIL PENCARIAN DOSEN: \"%s\"*\nData dosen tidak ditemukan di daftar dosen maupun jadwal.", keyword)
}

// SearchRuangan mencari jadwal di ruangan tertentu
func (j *JadwalConfig) SearchRuangan(keyword string) string {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Mohon sertakan nama atau kode ruangan.\nContoh: `!ruang Lab` atau `!ruang D105`"
	}

	var hasil []JadwalItem
	for _, item := range j.Jadwal {
		if strings.Contains(strings.ToLower(item.Ruang), keyword) {
			hasil = append(hasil, item)
		}
	}

	return j.FormatList(hasil, fmt.Sprintf("JADWAL DI RUANGAN: \"%s\"", keyword))
}

// SearchGlobal mencari jadwal berdasarkan kata kunci apa pun (matkul, dosen, ruangan, kode)
func (j *JadwalConfig) SearchGlobal(keyword string) string {
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return "⚠️ Mohon sertakan kata kunci pencarian.\nContoh: `!cari basis data`"
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

	return j.FormatList(hasil, fmt.Sprintf("HASIL PENCARIAN: \"%s\"", keyword))
}

// ProcessMessage memproses pesan masuk dan mengembalikan pesan balasan yang sesuai
func (j *JadwalConfig) ProcessMessage(rawMsg string) string {
	trimmed := strings.TrimSpace(rawMsg)
	if trimmed == "" {
		return ""
	}

	hasPrefix := false
	clean := trimmed

	// Deteksi prefix perintah (! / #)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		hasPrefix = true
		clean = strings.TrimSpace(clean[1:])
	}

	lower := strings.ToLower(clean)

	// 1. Menu & Bantuan
	if lower == "menu" || lower == "help" || lower == "info" || lower == "bantuan" {
		return j.GetMenu()
	}

	// 2. Pintasan Waktu Cepat (Hari Ini & Besok)
	if lower == "hari ini" || lower == "hariini" || lower == "today" || lower == "now" {
		return j.GetByHari("hari ini")
	}
	if lower == "besok" || lower == "tomorrow" {
		return j.GetByHari("besok")
	}

	// 3. Pintasan Nama Hari Langsung (misal: "!senin", "senin", "!jumat")
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

	// 4. Perintah Jadwal Lengkap (misal: "!jadwal", "!jadwal senin", "!jadwal besok")
	if strings.HasPrefix(lower, "jadwal") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.GetByHari(arg)
	}

	// 5. Perintah Dosen (misal: "!dosen MR", "dosen Rizqi")
	if strings.HasPrefix(lower, "dosen") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.SearchDosen(arg)
	}

	// 6. Perintah Ruang / Ruangan / Lab (misal: "!ruang D105", "!lab")
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

	// 7. Perintah Cari Global (misal: "!cari sistem operasi")
	if strings.HasPrefix(lower, "cari") || strings.HasPrefix(lower, "search") {
		parts := strings.SplitN(clean, " ", 2)
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}
		return j.SearchGlobal(arg)
	}

	// 8. Fallback jika diawali prefix perintah tetapi tidak dikenali
	if hasPrefix {
		return fmt.Sprintf("⚠️ Perintah *\"%s\"* tidak dikenali.\n\nKetik *!menu* untuk melihat panduan perintah yang tersedia.", trimmed)
	}

	return ""
}

// GetMenu mengembalikan pesan bantuan dan daftar perintah bot
func (j *JadwalConfig) GetMenu() string {
	now := time.Now()
	hariStr := getHariIndonesia(now)
	tanggalStr := fmt.Sprintf("%s, %02d %s %d", hariStr, now.Day(), getBulanIndonesia(now), now.Year())

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
		statusHariIni = fmt.Sprintf("%d Mata Kuliah Hari Ini", todayCount)
	} else {
		statusHariIni = "Libur / Tidak Ada Perkuliahan"
	}

	var sb strings.Builder
	sb.WriteString("🤖 *ASISTEN JADWAL KULIAH JTK*\n")
	sb.WriteString(fmt.Sprintf("🎓 *Kelas:* %s\n", j.Kampus))
	sb.WriteString(fmt.Sprintf("📅 *Waktu:* %s\n", tanggalStr))
	sb.WriteString(fmt.Sprintf("⚡ *Status:* _%s_\n", statusHariIni))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n\n")

	sb.WriteString("⚡ *JADWAL CEPAT (Sering Dipakai)*\n")
	sb.WriteString("  • `!hari ini`  ➜ Jadwal kuliah hari ini\n")
	sb.WriteString("  • `!besok`     ➜ Jadwal kuliah esok hari\n")
	sb.WriteString("  • `!senin` s/d `!jumat` ➜ Jadwal hari tertentu\n\n")

	sb.WriteString("🔍 *PENCARIAN SPESIFIK*\n")
	sb.WriteString("  • `!dosen MR`   ➜ Cari jadwal dosen (Pak Rizqi)\n")
	sb.WriteString("  • `!ruang Lab`  ➜ Cari jadwal di Lab tertentu\n")
	sb.WriteString("  • `!cari basis` ➜ Cari matkul / kata kunci apa pun\n\n")

	sb.WriteString("ℹ️ *TIPS PENGGUNAAN*\n")
	sb.WriteString("• Bisa tanpa tanda seru (contoh ketik: `hari ini` atau `senin`)\n")
	sb.WriteString("• Perintah bantuan: `!menu` atau `!help`\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("_Bot Asisten Kelas JTK Polban_")
	return sb.String()
}
