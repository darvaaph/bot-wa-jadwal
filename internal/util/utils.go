package util

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NamaHariMap memetakan nama-nama hari (Indonesia dan Inggris) ke tipe time.Weekday
var NamaHariMap = map[string]time.Weekday{
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

// BulanMap memetakan nama dan singkatan bulan bahasa Indonesia ke time.Month
var BulanMap = map[string]time.Month{
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

// DateWordRe mencocokkan pola tanggal yang menggunakan nama bulan (cth: "5 sep", "8 september", "17 agustus 2026")
var DateWordRe = regexp.MustCompile(`\b(\d{1,2})[\s\-\/]+([a-zA-Z]+)(?:[\s\-\/]+(20\d{2}))?\b`)

// TimeRe mencocokkan format jam HH:MM atau HH.MM
var TimeRe = regexp.MustCompile(`\b([01]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)

// GetHariIndonesia mengonversi nama hari time.Weekday ke bahasa Indonesia
func GetHariIndonesia(t time.Time) string {
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

// GetBulanIndonesia mengonversi bulan ke singkatan bahasa Indonesia yang ringkas
func GetBulanIndonesia(t time.Time) string {
	bulan := []string{
		"", "Jan", "Feb", "Mar", "Apr", "Mei", "Jun",
		"Jul", "Agu", "Sep", "Okt", "Nov", "Des",
	}
	if int(t.Month()) >= 1 && int(t.Month()) <= 12 {
		return bulan[t.Month()]
	}
	return ""
}

// IsDayName mengecek apakah string mengandung nama hari kerja
func IsDayName(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	days := []string{"senin", "selasa", "rabu", "kamis", "jumat", "jum'at", "sabtu", "minggu"}
	for _, d := range days {
		if strings.Contains(s, d) {
			return true
		}
	}
	return false
}

// IsDatePattern memeriksa apakah teks mengandung format tanggal DD-MM-YYYY atau YYYY-MM-DD
func IsDatePattern(s string) bool {
	reDate := regexp.MustCompile(`\b\d{2,4}[-/]\d{2}[-/]\d{2,4}\b`)
	return reDate.MatchString(s) || DateWordRe.MatchString(s)
}

// GetDateForDayName mencari tanggal kalender untuk nama hari tertentu terhitung dari refNow
func GetDateForDayName(dayName string, refNow time.Time) time.Time {
	clean := strings.ToLower(strings.TrimSpace(dayName))
	loc := refNow.Location()

	targetWeekday, ok := NamaHariMap[clean]
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

// ParseIndonesianDateWord mengekstrak tanggal, nama bulan, dan tahun opsional dari teks
func ParseIndonesianDateWord(text string, refNow time.Time, loc *time.Location) (time.Time, bool) {
	if matches := DateWordRe.FindStringSubmatch(text); len(matches) >= 3 {
		day, _ := strconv.Atoi(matches[1])
		bStr := strings.ToLower(matches[2])
		year := refNow.Year()
		hasYear := false
		if len(matches) > 3 && matches[3] != "" {
			fmt.Sscanf(matches[3], "%d", &year)
			hasYear = true
		}

		if monthVal, ok := BulanMap[bStr]; ok && day >= 1 && day <= 31 {
			target := time.Date(year, monthVal, day, 0, 0, 0, 0, loc)
			if !hasYear && target.AddDate(0, 1, 0).Before(refNow) {
				target = target.AddDate(1, 0, 0)
			}
			return target, true
		}
	}
	return time.Time{}, false
}

// ParseJamRange membedah rentang jam "HH:MM - HH:MM" menjadi 2 objek time.Time
func ParseJamRange(jamStr string, refDate time.Time) (time.Time, time.Time, error) {
	parts := strings.Split(jamStr, "-")
	if len(parts) != 2 {
		return time.Time{}, time.Time{}, fmt.Errorf("format jam tidak valid: %s", jamStr)
	}

	cleanTime := func(s string) (int, int, error) {
		s = strings.TrimSpace(s)
		s = strings.ReplaceAll(s, ".", ":")
		var h, m int
		_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
		return h, m, err
	}

	h1, m1, err1 := cleanTime(parts[0])
	h2, m2, err2 := cleanTime(parts[1])
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("gagal parsing jam: %s", jamStr)
	}

	loc := refDate.Location()
	startTime := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), h1, m1, 0, 0, loc)
	endTime := time.Date(refDate.Year(), refDate.Month(), refDate.Day(), h2, m2, 0, 0, loc)

	return startTime, endTime, nil
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
		matches := TimeRe.FindAllString(clean, -1)
		if len(matches) >= 2 {
			start := strings.ReplaceAll(matches[0], ".", ":")
			end := strings.ReplaceAll(matches[1], ".", ":")
			return fmt.Sprintf("%s - %s", start, end)
		}
	}

	match := TimeRe.FindString(clean)
	if match == "" {
		return clean
	}

	cleanJam := strings.ReplaceAll(match, ".", ":")
	var startH, startM int
	fmt.Sscanf(cleanJam, "%d:%d", &startH, &startM)

	totalStartMinutes := startH*60 + startM
	totalEndMinutes := totalStartMinutes + durationMinutes

	endH := (totalEndMinutes / 60) % 24
	endM := totalEndMinutes % 60

	return fmt.Sprintf("%02d:%02d - %02d:%02d", startH, startM, endH, endM)
}

// ParseFlexibleTime membaca datetime SQLite baik yang bertipe string, []byte, maupun time.Time
func ParseFlexibleTime(val any, loc *time.Location) time.Time {
	if val == nil {
		return time.Time{}
	}
	switch v := val.(type) {
	case time.Time:
		if loc != nil {
			return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), v.Nanosecond(), loc)
		}
		return v
	case string:
		return ParseTimeString(v, loc)
	case []byte:
		return ParseTimeString(string(v), loc)
	}
	return time.Time{}
}

// ParseTimeString membaca string tanggal dengan berbagai layout umum
func ParseTimeString(s string, loc *time.Location) time.Time {
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

// Contains memeriksa apakah suatu string terdapat dalam slice (case-insensitive)
func Contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

// CleanCommandPrefix membersihkan awalan prefix perintah seperti !, /, atau #
func CleanCommandPrefix(msg string) string {
	clean := strings.TrimSpace(msg)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		return strings.TrimSpace(clean[1:])
	}
	return clean
}

// MatchCommandPrefix memeriksa apakah teks pesan diawali oleh salah satu kata kunci perintah.
// - Di grup WhatsApp: Pesan WAJIB diawali simbol prefix (!, /, atau #).
// - Di chat pribadi (DM): Simbol prefix bersifat opsional.
// Fungsi ini juga menjamin batas kata (word boundary) sehingga "!tugas" cocok, tetapi "!tugaskemarin" tidak.
func MatchCommandPrefix(msg string, isGroup bool, keywords ...string) bool {
	clean := strings.TrimSpace(msg)
	if clean == "" {
		return false
	}
	lower := strings.ToLower(clean)
	hasSymbol := strings.HasPrefix(lower, "!") || strings.HasPrefix(lower, "/") || strings.HasPrefix(lower, "#")
	if isGroup && !hasSymbol {
		return false
	}

	cmdName := lower
	if hasSymbol {
		cmdName = lower[1:]
	}

	for _, kw := range keywords {
		kwLower := strings.ToLower(kw)
		if strings.HasPrefix(cmdName, kwLower) {
			rest := cmdName[len(kwLower):]
			if rest == "" || strings.HasPrefix(rest, " ") || strings.HasPrefix(rest, "\n") || strings.HasPrefix(rest, "\t") {
				return true
			}
		}
	}
	return false
}

// IsMenuOrHelpCommand memeriksa apakah pesan merupakan perintah melihat menu atau panduan
func IsMenuOrHelpCommand(msg string, isGroup bool) bool {
	return MatchCommandPrefix(msg, isGroup, "menu", "help", "keyword", "keywords", "bantuan", "panduan")
}

// FindDataDir mencari path direktori atau file data (misal "data/jadwal" atau "jadwal.json")
// dengan mengecek direktori saat ini dan menaik hingga 4 tingkat parent directories.
// Ini menjamin test di subpackage maupun proses runtime di root selalu menemukan file data.
func FindDataDir(targetPath string) string {
	if targetPath == "" {
		return targetPath
	}
	if _, err := os.Stat(targetPath); err == nil {
		return targetPath
	}

	p := targetPath
	for i := 0; i < 4; i++ {
		p = filepath.Join("..", p)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return targetPath
}
