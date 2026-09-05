package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestOverrideManager(t *testing.T) {
	testDB := "test_overrides.db"
	defer os.Remove(testDB)

	om, err := NewOverrideManager(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewOverrideManager: %v", err)
	}
	defer om.Close()

	cfg, err := LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal load jadwal.json: %v", err)
	}
	cfg.SetOverrideManager(om)

	// Waktu acuan: Senin, 7 September 2026 pukul 10:00 WIB
	refNow := time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local) // 7 Sep 2026 adalah Senin
	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"

	// 1. Test Pencocokan Cerdas Mata Kuliah (FindMataKuliah)
	itemSBD, _ := cfg.FindMataKuliah("sbd", refNow)
	if itemSBD == nil || !strings.Contains(itemSBD.NamaMatkul, "Sistem Basis Data") {
		t.Errorf("Expected 'sbd' to find Sistem Basis Data, got: %v", itemSBD)
	}

	itemMatdis, _ := cfg.FindMataKuliah("matdis", refNow)
	if itemMatdis == nil || !strings.Contains(itemMatdis.NamaMatkul, "Matematika Diskrit Lanjut") {
		t.Errorf("Expected 'matdis' to find Matematika Diskrit Lanjut, got: %v", itemMatdis)
	}

	// Disambiguasi: Aljabar Teori vs Praktikum
	itemAljabarTeori, _ := cfg.FindMataKuliah("aljabar teori", refNow)
	if itemAljabarTeori == nil || !strings.Contains(itemAljabarTeori.NamaMatkul, "Teori") {
		t.Errorf("Expected 'aljabar teori' to match Teori session, got: %v", itemAljabarTeori)
	}

	// 2. Test Kalkulasi Durasi & Auto Complete Jam
	dur := CalculateDurationInMinutes("07:00 - 08:40")
	if dur != 100 {
		t.Errorf("Expected duration 100 minutes, got %d", dur)
	}
	autoJam := AutoCompleteJamRange("13:00", 100)
	if autoJam != "13:00 - 14:40" {
		t.Errorf("Expected '13:00 - 14:40', got '%s'", autoJam)
	}

	// 3. Test Non-Admin Ditolak di Grup
	nonAdminReply := om.HandleCommand(groupJID, true, userJID, false, "!pindah aljabar | besok 13:00", cfg, refNow)
	if !strings.Contains(nonAdminReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected, got: %s", nonAdminReply)
	}

	// 4. Test Admin Reschedule Aljabar (Senin 07:00) ke Besok (Selasa) 15:00 di Lab 312
	adminReply := om.HandleCommand(groupJID, true, userJID, true, "!pindah aljabar praktikum | besok 15:00 | Lab 312", cfg, refNow)
	if !strings.Contains(adminReply, "BERHASIL DIPINDAHKAN") || !strings.Contains(adminReply, "Lab 312") {
		t.Errorf("Expected reschedule success, got: %s", adminReply)
	}

	// 5. Verifikasi Jadwal Hari Asal (Senin, 7 Sep 2026): Aljabar Praktikum harus dicoret / DIPINDAHKAN
	seninSchedule := cfg.GetByHariWithOverrides("hari ini", groupJID, om, refNow)
	if !strings.Contains(seninSchedule, "KULIAH DIPINDAHKAN") || !strings.Contains(seninSchedule, "Aljabar Linear") {
		t.Errorf("Expected origin schedule (Senin) to mark class as moved, got:\n%s", seninSchedule)
	}

	// 6. Verifikasi Jadwal Hari Tujuan (Selasa, 8 Sep 2026): Aljabar Praktikum muncul sebagai KULIAH PENGGANTI di jam 15:00
	selasaDate := refNow.Add(24 * time.Hour) // Selasa, 8 Sep 2026
	selasaSchedule := cfg.GetByHariWithOverrides("besok", groupJID, om, refNow)
	if !strings.Contains(selasaSchedule, "KULIAH PENGGANTI") ||
		!strings.Contains(selasaSchedule, "15:00") ||
		!strings.Contains(selasaSchedule, "Lab 312") {
		t.Errorf("Expected destination schedule (Selasa) to contain make-up class, got:\n%s", selasaSchedule)
	}

	// 7. Test Kuliah Ditiadakan (!kosong SBD) pada hari Selasa
	kosongReply := om.HandleCommand(groupJID, true, userJID, true, "!kosong sbd | besok | Dosen dinas luar", cfg, refNow)
	if !strings.Contains(kosongReply, "DITANDAI DITIADAKAN") {
		t.Errorf("Expected cancellation success, got: %s", kosongReply)
	}

	// Cek jadwal Selasa lagi: SBD harus dicoret DITIADAKAN
	selasaWithKosong := cfg.GetByHariWithOverrides("besok", groupJID, om, refNow)
	if !strings.Contains(selasaWithKosong, "KULIAH DITIADAKAN") || !strings.Contains(selasaWithKosong, "Dosen dinas luar") {
		t.Errorf("Expected Selasa SBD to be marked cancelled, got:\n%s", selasaWithKosong)
	}

	// 8. Test Kuliah Pengganti di Hari Libur (!kuliahganti Sabtu)
	extraReply := om.HandleCommand(groupJID, true, userJID, true, "!kuliahganti matdis | sabtu 09:00 - 11:30 | D105", cfg, refNow)
	if !strings.Contains(extraReply, "KULIAH PENGGANTI DITAMBAHKAN") {
		t.Errorf("Expected extra class success, got: %s", extraReply)
	}

	sabtuDate := time.Date(2026, 9, 12, 10, 0, 0, 0, time.Local) // Sabtu, 12 Sep 2026
	sabtuSchedule := cfg.GetByHariWithOverrides("sabtu", groupJID, om, refNow)
	if !strings.Contains(sabtuSchedule, "KULIAH TAMBAHAN") || !strings.Contains(sabtuSchedule, "09:00 - 11:30") {
		t.Errorf("Expected Saturday to show extra class, got:\n%s", sabtuSchedule)
	}

	// 9. Test GetNextClassWithOverrides:
	// Di hari Senin jam 07:15: Aljabar (yang dipindahkan) harus DILEWATI, kuliah berikutnya adalah Programming Pragmatics (13:00)
	tSeninPagi := time.Date(2026, 9, 7, 7, 15, 0, 0, time.Local)
	nextSenin := cfg.GetNextClassWithOverrides(tSeninPagi, groupJID, om)
	if strings.Contains(nextSenin, "SEDANG BERLANGSUNG") && strings.Contains(nextSenin, "Aljabar") {
		t.Errorf("Expected cancelled/moved class to NOT be ongoing, got:\n%s", nextSenin)
	}
	if !strings.Contains(nextSenin, "Programming Pragmatics") {
		t.Errorf("Expected next class to be Programming Pragmatics, got:\n%s", nextSenin)
	}

	// Di hari Selasa jam 15:10: Kelas pindahan Aljabar di Lab 312 harus SEDANG BERLANGSUNG
	tSelasaSore := time.Date(2026, 9, 8, 15, 10, 0, 0, time.Local)
	nextSelasa := cfg.GetNextClassWithOverrides(tSelasaSore, groupJID, om)
	if !strings.Contains(nextSelasa, "SEDANG BERLANGSUNG") || !strings.Contains(nextSelasa, "Aljabar Linear") {
		t.Errorf("Expected make-up class to be ongoing on Tuesday 15:10, got:\n%s", nextSelasa)
	}

	// 10. Test Daftar Jadwal Pengganti Aktif (!jadwalganti)
	listReply := om.HandleCommand(groupJID, true, userJID, false, "!jadwalganti", cfg, refNow)
	if !strings.Contains(listReply, "DAFTAR JADWAL PENGGANTI AKTIF") ||
		!strings.Contains(listReply, "RESCHEDULE") ||
		!strings.Contains(listReply, "CANCEL") ||
		!strings.Contains(listReply, "EXTRA") {
		t.Errorf("Expected active override list, got:\n%s", listReply)
	}

	// 11. Test Pembatalan Override (!batalganti 1)
	cancelReply := om.HandleCommand(groupJID, true, userJID, true, "!batalganti 1", cfg, refNow)
	if !strings.Contains(cancelReply, "DIBATALKAN") {
		t.Errorf("Expected cancellation of override, got: %s", cancelReply)
	}

	// 12. Test Auto-Expiration di Minggu Depan (Senin, 14 Sep 2026)
	// Aljabar harus kembali normal tanpa status dipindahkan!
	seninDepan := time.Date(2026, 9, 14, 10, 0, 0, 0, time.Local)
	seninDepanSchedule := cfg.GetByHariWithOverrides("hari ini", groupJID, om, seninDepan)
	if strings.Contains(seninDepanSchedule, "DIPINDAHKAN") {
		t.Errorf("Expected next week schedule to be normal without moved status, got:\n%s", seninDepanSchedule)
	}
	// 13. Test Integrasi Pengingat Pagi dengan Override (BuildMorningReminder)
	// Pada hari Selasa (besok dari refNow), SBD ditiadakan (override #1 Aljabar sudah dibatalkan)
	tSelasaPagi := time.Date(2026, 9, 8, 6, 30, 0, 0, time.Local)
	reminderSelasa := BuildMorningReminder(groupJID, cfg, nil, tSelasaPagi)
	if !strings.Contains(reminderSelasa, "KULIAH DITIADAKAN") || !strings.Contains(reminderSelasa, "Dosen dinas luar") {
		t.Errorf("Expected morning reminder on Tuesday to contain cancelled SBD, got:\n%s", reminderSelasa)
	}

	// 14. Test ProcessMessage dengan Chat JID (Pintasan !besok mengenali override grup)
	msgBesok := cfg.ProcessMessage("!besok", true, groupJID, refNow)
	if !strings.Contains(msgBesok, "KULIAH DITIADAKAN") || !strings.Contains(msgBesok, "Dosen dinas luar") {
		t.Errorf("Expected ProcessMessage('!besok') with group JID to render overrides, got:\n%s", msgBesok)
	}

	_ = selasaDate
	_ = sabtuDate
}
