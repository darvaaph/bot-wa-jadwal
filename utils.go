package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// namaHariMap memetakan nama-nama hari (Indonesia dan Inggris) ke tipe time.Weekday
var namaHariMap = map[string]time.Weekday{
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

// bulanMap memetakan nama dan singkatan bulan bahasa Indonesia ke time.Month
var bulanMap = map[string]time.Month{
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

// dateWordRe mencocokkan pola tanggal yang menggunakan nama bulan (cth: "5 sep", "8 september", "17 agustus 2026")
var dateWordRe = regexp.MustCompile(`\b(\d{1,2})[\s\-\/]+([a-zA-Z]+)(?:[\s\-\/]+(20\d{2}))?\b`)

// timeRe mencocokkan format jam HH:MM atau HH.MM
var timeRe = regexp.MustCompile(`\b([01]?[0-9]|2[0-3])[:.]([0-5][0-9])\b`)

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

// isDayName mengecek apakah string mengandung nama hari kerja
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

// isDatePattern memeriksa apakah teks mengandung format tanggal DD-MM-YYYY atau YYYY-MM-DD
func isDatePattern(s string) bool {
	reDate := regexp.MustCompile(`\b\d{2,4}[-/]\d{2}[-/]\d{2,4}\b`)
	return reDate.MatchString(s) || dateWordRe.MatchString(s)
}

// GetDateForDayName mencari tanggal kalender untuk nama hari tertentu terhitung dari refNow
func GetDateForDayName(dayName string, refNow time.Time) time.Time {
	clean := strings.ToLower(strings.TrimSpace(dayName))
	loc := refNow.Location()

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

// parseIndonesianDateWord mengekstrak tanggal, nama bulan, dan tahun opsional dari teks
func parseIndonesianDateWord(text string, refNow time.Time, loc *time.Location) (time.Time, bool) {
	if matches := dateWordRe.FindStringSubmatch(text); len(matches) >= 3 {
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
			return target, true
		}
	}
	return time.Time{}, false
}

// parseJamRange membedah rentang jam "HH:MM - HH:MM" menjadi 2 objek time.Time
func parseJamRange(jamStr string, refDate time.Time) (time.Time, time.Time, error) {
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
		matches := timeRe.FindAllString(clean, -1)
		if len(matches) >= 2 {
			start := strings.ReplaceAll(matches[0], ".", ":")
			end := strings.ReplaceAll(matches[1], ".", ":")
			return fmt.Sprintf("%s - %s", start, end)
		}
	}

	match := timeRe.FindString(clean)
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

// parseFlexibleTime membaca datetime SQLite baik yang bertipe string, []byte, maupun time.Time
func parseFlexibleTime(val any, loc *time.Location) time.Time {
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

// contains memeriksa apakah suatu string terdapat dalam slice (case-insensitive)
func contains(slice []string, val string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, val) {
			return true
		}
	}
	return false
}

// cleanCommandPrefix membersihkan awalan prefix perintah seperti !, /, atau #
func cleanCommandPrefix(msg string) string {
	clean := strings.TrimSpace(msg)
	if strings.HasPrefix(clean, "!") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "#") {
		return strings.TrimSpace(clean[1:])
	}
	return clean
}

// matchCommandPrefix memeriksa apakah teks pesan diawali oleh salah satu kata kunci perintah.
// - Di grup WhatsApp: Pesan WAJIB diawali simbol prefix (!, /, atau #).
// - Di chat pribadi (DM): Simbol prefix bersifat opsional.
// Fungsi ini juga menjamin batas kata (word boundary) sehingga "!tugas" cocok, tetapi "!tugaskemarin" tidak.
func matchCommandPrefix(msg string, isGroup bool, keywords ...string) bool {
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

// isMenuOrHelpCommand memeriksa apakah pesan merupakan perintah melihat menu atau panduan
func isMenuOrHelpCommand(msg string, isGroup bool) bool {
	return matchCommandPrefix(msg, isGroup, "menu", "help", "keyword", "keywords", "bantuan", "panduan")
}

