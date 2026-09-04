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

// FormatList merapikan daftar item jadwal menjadi teks WhatsApp yang estetik dan mudah dibaca
func (j *JadwalConfig) FormatList(items []JadwalItem, judul string) string {
	if len(items) == 0 {
		return fmt.Sprintf("❌ *%s*\nTidak ada jadwal yang ditemukan.", judul)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📅 *%s*\n", judul))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")

	for i, item := range items {
		sb.WriteString(fmt.Sprintf("*%d. %s* (%s)\n", i+1, item.NamaMatkul, item.KodeMatkul))
		sb.WriteString(fmt.Sprintf("   ⏰ Waktu  : %s, %s\n", item.Hari, item.Jam))
		sb.WriteString(fmt.Sprintf("   📍 Ruang  : %s\n", item.Ruang))
		sb.WriteString(fmt.Sprintf("   👨‍🏫 Dosen  : %s (%s)\n\n", item.Dosen, item.InisialDosen))
	}

	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("_Ketik *!menu* untuk melihat daftar perintah._")
	return sb.String()
}

// GetByHari mencari jadwal berdasarkan hari tertentu, termasuk alias 'hari ini' dan 'besok'
func (j *JadwalConfig) GetByHari(hariInput string) string {
	hariInput = strings.ToLower(strings.TrimSpace(hariInput))
	waktuSekarang := time.Now()

	switch hariInput {
	case "hari ini", "today", "":
		hariInput = strings.ToLower(getHariIndonesia(waktuSekarang))
	case "besok", "tomorrow":
		hariInput = strings.ToLower(getHariIndonesia(waktuSekarang.Add(24 * time.Hour)))
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
		return "⚠️ Mohon sertakan nama atau kode dosen.\nContoh: `!dosen YD` atau `!dosen Yudi`"
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
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
			sb.WriteString(fmt.Sprintf("• *Nama*: %s\n", nama))
			sb.WriteString(fmt.Sprintf("• *ID / Inisial*: %s\n\n", id))
			sb.WriteString("ℹ️ _Dosen terdaftar di JTK, namun tidak memiliki jadwal perkuliahan di kelas ini._\n")
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
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
		return "⚠️ Mohon sertakan nama atau kode ruangan.\nContoh: `!ruang Lab 2` atau `!ruang 402`"
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
		return "⚠️ Mohon sertakan kata kunci pencarian.\nContoh: `!cari Algoritma`"
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

// GetMenu mengembalikan pesan bantuan dan daftar perintah bot
func (j *JadwalConfig) GetMenu() string {
	var sb strings.Builder
	sb.WriteString("🤖 *BOT JADWAL KULIAH OTOMATIS*\n")
	sb.WriteString(fmt.Sprintf("_%s_\n", j.Kampus))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n\n")
	sb.WriteString("Gunakan perintah berikut untuk mencari informasi:\n\n")
	sb.WriteString("📌 *Cek Jadwal Harian:*\n")
	sb.WriteString("• `!jadwal senin` (atau hari lainnya)\n")
	sb.WriteString("• `!jadwal hari ini`\n")
	sb.WriteString("• `!jadwal besok`\n\n")
	sb.WriteString("📌 *Pencarian Khusus:*\n")
	sb.WriteString("• `!dosen [inisial/nama]`\n  _Contoh: `!dosen BDS` atau `!dosen Bambang`_\n")
	sb.WriteString("• `!ruang [nama/nomor]`\n  _Contoh: `!ruang Lab` atau `!ruang 402`_\n")
	sb.WriteString("• `!cari [kata kunci]`\n  _Contoh: `!cari jaringan`_\n\n")
	sb.WriteString("📌 *Lainnya:*\n")
	sb.WriteString("• `!menu` atau `!help` (Tampilkan menu ini)\n\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("💡 *Catatan:* Perintah juga mendukung prefiks garis miring (contoh: `/jadwal senin`).")
	return sb.String()
}
