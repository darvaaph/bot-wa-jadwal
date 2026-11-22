package util

import (
	"testing"
	"time"
)

func TestUtils(t *testing.T) {
	loc := time.Local
	ref := time.Date(2026, 9, 7, 10, 0, 0, 0, loc) // Senin, 7 September 2026

	if h := GetHariIndonesia(ref); h != "Senin" {
		t.Errorf("Expected 'Senin', got '%s'", h)
	}

	if b := GetBulanIndonesia(ref); b != "Sep" {
		t.Errorf("Expected 'Sep', got '%s'", b)
	}

	if !IsDayName("senin") || !IsDayName("Jum'at") || IsDayName("kalkulus") {
		t.Errorf("IsDayName failed validation")
	}

	if !IsDatePattern("12-09-2026") || !IsDatePattern("5 sep") || IsDatePattern("halo") {
		t.Errorf("IsDatePattern failed validation")
	}

	selasa := GetDateForDayName("selasa", ref)
	if selasa.Day() != 8 || selasa.Month() != 9 {
		t.Errorf("Expected Tuesday 8 Sep, got %v", selasa)
	}

	parsedDate, ok := ParseIndonesianDateWord("17 agustus", ref, loc)
	if !ok || parsedDate.Month() != time.August || parsedDate.Day() != 17 {
		t.Errorf("Expected 17 Aug, got %v (ok: %v)", parsedDate, ok)
	}

	sTime, eTime, err := ParseJamRange("07:00 - 08:40", ref)
	if err != nil || sTime.Hour() != 7 || eTime.Hour() != 8 || eTime.Minute() != 40 {
		t.Errorf("ParseJamRange error: %v, start: %v, end: %v", err, sTime, eTime)
	}

	dur := CalculateDurationInMinutes("07:00 - 08:40")
	if dur != 100 {
		t.Errorf("Expected 100 minutes, got %d", dur)
	}

	autoJam := AutoCompleteJamRange("13:00", 100)
	if autoJam != "13:00 - 14:40" {
		t.Errorf("Expected '13:00 - 14:40', got '%s'", autoJam)
	}

	if CleanCommandPrefix("!tugas") != "tugas" || CleanCommandPrefix("/pindah") != "pindah" || CleanCommandPrefix("#libur") != "libur" {
		t.Errorf("CleanCommandPrefix failed")
	}

	if !Contains([]string{"Senin", "Selasa"}, "senin") || Contains([]string{"Senin"}, "rabu") {
		t.Errorf("Contains failed")
	}

	ft := ParseFlexibleTime("2026-09-07 10:00:00", loc)
	if ft.Day() != 7 || ft.Hour() != 10 {
		t.Errorf("ParseFlexibleTime failed, got %v", ft)
	}

	// Di grup WA: wajib simbol prefix
	if !MatchCommandPrefix("!tugas", true, "tugas") {
		t.Errorf("MatchCommandPrefix(!tugas, group) should be true")
	}
	if !MatchCommandPrefix("/tugas 1", true, "tugas") {
		t.Errorf("MatchCommandPrefix(/tugas 1, group) should be true")
	}
	if !MatchCommandPrefix("#pindah aljabar", true, "pindah", "ganti") {
		t.Errorf("MatchCommandPrefix(#pindah, group) should be true")
	}
	if MatchCommandPrefix("tugas", true, "tugas") {
		t.Errorf("MatchCommandPrefix(tugas tanpa prefix, group) should be false")
	}
	if MatchCommandPrefix("!tugaskemarin", true, "tugas") {
		t.Errorf("MatchCommandPrefix(!tugaskemarin, group) should be false due to word boundary")
	}

	// Di DM: prefix opsional
	if !MatchCommandPrefix("tugas", false, "tugas") {
		t.Errorf("MatchCommandPrefix(tugas, DM) should be true")
	}
	if !MatchCommandPrefix("!tugas", false, "tugas") {
		t.Errorf("MatchCommandPrefix(!tugas, DM) should be true")
	}
	if !MatchCommandPrefix("setkelas 3B", false, "setkelas", "kelas") {
		t.Errorf("MatchCommandPrefix(setkelas 3B, DM) should be true")
	}

	if !IsMenuOrHelpCommand("!menu", true) {
		t.Errorf("IsMenuOrHelpCommand(!menu, group) should be true")
	}
	if !IsMenuOrHelpCommand("/help", true) {
		t.Errorf("IsMenuOrHelpCommand(/help, group) should be true")
	}
	if IsMenuOrHelpCommand("menu", true) {
		t.Errorf("IsMenuOrHelpCommand(menu tanpa prefix, group) should be false")
	}
	if !IsMenuOrHelpCommand("menu", false) {
		t.Errorf("IsMenuOrHelpCommand(menu, DM) should be true")
	}
	if !IsMenuOrHelpCommand("panduan", false) {
		t.Errorf("IsMenuOrHelpCommand(panduan, DM) should be true")
	}
	if IsMenuOrHelpCommand("jadwal", false) {
		t.Errorf("IsMenuOrHelpCommand(jadwal, DM) should be false")
	}

	foundData := FindDataDir("data/jadwal")
	if foundData == "" {
		t.Errorf("FindDataDir data/jadwal should not be empty")
	}
}

func TestFormatTaskDeadline(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)
	now := time.Date(2026, 11, 7, 17, 0, 0, 0, loc)

	if got := FormatTaskDeadline(time.Date(2026, 11, 7, 23, 59, 0, 0, loc), now); got != "Hari ini (Sabtu, 23:59)" {
		t.Errorf("expected 'Hari ini (Sabtu, 23:59)', got '%s'", got)
	}
	if got := FormatTaskDeadline(time.Date(2026, 11, 8, 23, 59, 0, 0, loc), now); got != "Besok (Minggu, 23:59)" {
		t.Errorf("expected 'Besok (Minggu, 23:59)', got '%s'", got)
	}
	if got := FormatTaskDeadline(time.Date(2026, 11, 10, 13, 0, 0, 0, loc), now); got != "Selasa, 10 Nov (13:00)" {
		t.Errorf("expected 'Selasa, 10 Nov (13:00)', got '%s'", got)
	}
	if got := FormatTaskDeadline(time.Date(2027, 2, 1, 8, 0, 0, 0, loc), now); got != "Senin, 01 Feb 2027 (08:00)" {
		t.Errorf("expected 'Senin, 01 Feb 2027 (08:00)', got '%s'", got)
	}
}

func TestFormatCourseWithActivity(t *testing.T) {
	tests := []struct {
		name     string
		course   string
		activity string
		expected string
	}{
		{"Practice clean", "Sistem Basis Data", "PRACTICE", "Sistem Basis Data (P)"},
		{"Practice with suffix", "Sistem Basis Data (Praktikum)", "PRAKTIKUM", "Sistem Basis Data (P)"},
		{"Theory clean", "Aljabar Linear", "THEORY", "Aljabar Linear (T)"},
		{"Theory with suffix", "Aljabar Linear (Teori)", "TEORI", "Aljabar Linear (T)"},
		{"No activity", "Pancasila", "", "Pancasila"},
		{"Already has (P)", "Basis Data (P)", "PRACTICE", "Basis Data (P)"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatCourseWithActivity(tc.course, tc.activity)
			if got != tc.expected {
				t.Errorf("FormatCourseWithActivity(%q, %q) = %q, want %q", tc.course, tc.activity, got, tc.expected)
			}
		})
	}
}

func TestFormatTaskTitleAndSubmissionLabel(t *testing.T) {
	if got := FormatTaskTitle("Aljabar Relasional", "Kuis"); got != "[KUIS] Aljabar Relasional" {
		t.Errorf("expected '[KUIS] Aljabar Relasional', got %q", got)
	}
	if got := FormatTaskTitle("Kuis 1 - SQL", "Kuis"); got != "Kuis 1 - SQL" {
		t.Errorf("expected 'Kuis 1 - SQL' without duplicate prefix, got %q", got)
	}
	if got := FormatTaskTitle("Latihan Vektor", "Individu"); got != "Latihan Vektor" {
		t.Errorf("expected 'Latihan Vektor', got %q", got)
	}
	if got := TaskSubmissionLabel("Kuis"); got != "Pelaksanaan" {
		t.Errorf("expected 'Pelaksanaan', got %q", got)
	}
	if got := TaskSubmissionLabel("Kelompok"); got != "Pengumpulan" {
		t.Errorf("expected 'Pengumpulan', got %q", got)
	}
}
