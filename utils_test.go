package main

import (
	"testing"
	"time"
)

func TestUtils(t *testing.T) {
	loc := time.Local
	ref := time.Date(2026, 9, 7, 10, 0, 0, 0, loc) // Senin, 7 September 2026

	// 1. Test getHariIndonesia
	if h := getHariIndonesia(ref); h != "Senin" {
		t.Errorf("Expected 'Senin', got '%s'", h)
	}

	// 2. Test getBulanIndonesia
	if b := getBulanIndonesia(ref); b != "Sep" {
		t.Errorf("Expected 'Sep', got '%s'", b)
	}

	// 3. Test isDayName
	if !isDayName("senin") || !isDayName("Jum'at") || isDayName("kalkulus") {
		t.Errorf("isDayName failed validation")
	}

	// 4. Test isDatePattern
	if !isDatePattern("12-09-2026") || !isDatePattern("5 sep") || isDatePattern("halo") {
		t.Errorf("isDatePattern failed validation")
	}

	// 5. Test GetDateForDayName
	selasa := GetDateForDayName("selasa", ref)
	if selasa.Day() != 8 || selasa.Month() != 9 {
		t.Errorf("Expected Tuesday 8 Sep, got %v", selasa)
	}

	// 6. Test parseIndonesianDateWord
	parsedDate, ok := parseIndonesianDateWord("17 agustus", ref, loc)
	if !ok || parsedDate.Month() != time.August || parsedDate.Day() != 17 {
		t.Errorf("Expected 17 Aug, got %v (ok: %v)", parsedDate, ok)
	}

	// 7. Test parseJamRange
	sTime, eTime, err := parseJamRange("07:00 - 08:40", ref)
	if err != nil || sTime.Hour() != 7 || eTime.Hour() != 8 || eTime.Minute() != 40 {
		t.Errorf("parseJamRange error: %v, start: %v, end: %v", err, sTime, eTime)
	}

	// 8. Test CalculateDurationInMinutes
	dur := CalculateDurationInMinutes("07:00 - 08:40")
	if dur != 100 {
		t.Errorf("Expected 100 minutes, got %d", dur)
	}

	// 9. Test AutoCompleteJamRange
	autoJam := AutoCompleteJamRange("13:00", 100)
	if autoJam != "13:00 - 14:40" {
		t.Errorf("Expected '13:00 - 14:40', got '%s'", autoJam)
	}

	// 10. Test cleanCommandPrefix
	if cleanCommandPrefix("!tugas") != "tugas" || cleanCommandPrefix("/pindah") != "pindah" || cleanCommandPrefix("#libur") != "libur" {
		t.Errorf("cleanCommandPrefix failed")
	}

	// 11. Test contains
	if !contains([]string{"Senin", "Selasa"}, "senin") || contains([]string{"Senin"}, "rabu") {
		t.Errorf("contains failed")
	}

	// 12. Test parseFlexibleTime
	ft := parseFlexibleTime("2026-09-07 10:00:00", loc)
	if ft.Day() != 7 || ft.Hour() != 10 {
		t.Errorf("parseFlexibleTime failed, got %v", ft)
	}
}
