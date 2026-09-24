package schedule

import (
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/util"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOverrideManager(t *testing.T) {
	testDB := "test_overrides.db"
	defer os.Remove(testDB)

	db, err := database.InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database SQLite: %v", err)
	}
	defer db.Close()

	om, err := NewOverrideManager(db)
	if err != nil {
		t.Fatalf("Gagal inisialisasi NewOverrideManager: %v", err)
	}

	cfg, err := LoadJadwal("jadwal.json")
	if err != nil {
		t.Fatalf("Gagal load jadwal.json: %v", err)
	}
	cfg.SetOverrideManager(om)

	// Waktu acuan: Senin, 7 September 2026 pukul 10:00 WIB
	refNow := time.Date(2026, 9, 7, 10, 0, 0, 0, time.Local) // 7 Sep 2026 adalah Senin
	groupJID := "120363001@g.us"
	userJID := "628120001@s.whatsapp.net"

	ctx := context.Background()
	cls, err := om.academicRepo.EnsureClass(ctx, "2A")
	if err != nil {
		t.Fatalf("Gagal memastikan kelas 2A: %v", err)
	}
	_, err = db.Exec(`INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status) VALUES (?, ?, 'GROUP', 'Kelas 2A', 'ACTIVE')`, cls.ID, groupJID)
	if err != nil {
		t.Fatalf("Gagal memetakan whatsapp_channels: %v", err)
	}

	// Verifikasi penolakan JID yang belum dipetakan (P1)
	unmappedJID := "120363999999@g.us"
	dummyItem := JadwalItem{KodeMatkul: "TI101", NamaMatkul: "Aljabar", Jam: "07:00 - 08:40"}
	_, err = om.AddReschedule(unmappedJID, dummyItem, refNow, refNow.Add(24*time.Hour), "15:00 - 16:40", "Lab", "user")
	if err == nil || !strings.Contains(err.Error(), "scope belum terpetakan") {
		t.Errorf("Expected ErrUnmappedScope for unmapped scope, got: %v", err)
	}

	itemSBD, _ := cfg.FindMataKuliah("sbd", refNow)
	if itemSBD == nil || !strings.Contains(itemSBD.NamaMatkul, "Sistem Basis Data") {
		t.Errorf("Expected 'sbd' to find Sistem Basis Data, got: %v", itemSBD)
	}

	itemMatdis, _ := cfg.FindMataKuliah("matdis", refNow)
	if itemMatdis == nil || !strings.Contains(itemMatdis.NamaMatkul, "Matematika Diskrit Lanjut") {
		t.Errorf("Expected 'matdis' to find Matematika Diskrit Lanjut, got: %v", itemMatdis)
	}

	itemAljabarTeori, _ := cfg.FindMataKuliah("aljabar teori", refNow)
	if itemAljabarTeori == nil || !strings.Contains(itemAljabarTeori.NamaMatkul, "Teori") {
		t.Errorf("Expected 'aljabar teori' to match Teori session, got: %v", itemAljabarTeori)
	}

	dur := util.CalculateDurationInMinutes("07:00 - 08:40")
	if dur != 100 {
		t.Errorf("Expected duration 100 minutes, got %d", dur)
	}
	autoJam := util.AutoCompleteJamRange("13:00", 100)
	if autoJam != "13:00 - 14:40" {
		t.Errorf("Expected '13:00 - 14:40', got '%s'", autoJam)
	}

	nonAdminReply := om.HandleCommand(groupJID, true, userJID, false, "!pindah aljabar | besok 13:00", cfg, refNow)
	if !strings.Contains(nonAdminReply, "Akses Ditolak") {
		t.Errorf("Expected non-admin to be rejected, got: %s", nonAdminReply)
	}

	adminReply := om.HandleCommand(groupJID, true, userJID, true, "!pindah aljabar praktikum | besok 15:00 | Lab 312", cfg, refNow)
	if !strings.Contains(adminReply, "BERHASIL DIPINDAHKAN") || !strings.Contains(adminReply, "Lab 312") {
		t.Errorf("Expected reschedule success, got: %s", adminReply)
	}

	seninSchedule := cfg.GetByHariWithOverrides("hari ini", groupJID, om, refNow)
	if !strings.Contains(seninSchedule, "KULIAH DIPINDAHKAN") || !strings.Contains(seninSchedule, "Aljabar Linear") {
		t.Errorf("Expected origin schedule (Senin) to mark class as moved, got:\n%s", seninSchedule)
	}

	selasaDate := refNow.Add(24 * time.Hour) // Selasa, 8 Sep 2026
	selasaSchedule := cfg.GetByHariWithOverrides("besok", groupJID, om, refNow)
	if !strings.Contains(selasaSchedule, "KULIAH PENGGANTI") ||
		!strings.Contains(selasaSchedule, "15:00") ||
		!strings.Contains(selasaSchedule, "Lab 312") {
		t.Errorf("Expected destination schedule (Selasa) to contain make-up class, got:\n%s", selasaSchedule)
	}

	kosongReply := om.HandleCommand(groupJID, true, userJID, true, "!kosong sbd | besok | Dosen dinas luar", cfg, refNow)
	if !strings.Contains(kosongReply, "DITANDAI DITIADAKAN") {
		t.Errorf("Expected cancellation success, got: %s", kosongReply)
	}

	selasaWithKosong := cfg.GetByHariWithOverrides("besok", groupJID, om, refNow)
	if !strings.Contains(selasaWithKosong, "KULIAH DITIADAKAN") || !strings.Contains(selasaWithKosong, "Dosen dinas luar") {
		t.Errorf("Expected Selasa SBD to be marked cancelled, got:\n%s", selasaWithKosong)
	}

	extraReply := om.HandleCommand(groupJID, true, userJID, true, "!kuliahganti matdis | sabtu 09:00 - 11:30 | D105", cfg, refNow)
	if !strings.Contains(extraReply, "KULIAH PENGGANTI DITAMBAHKAN") {
		t.Errorf("Expected extra class success, got: %s", extraReply)
	}

	sabtuDate := time.Date(2026, 9, 12, 10, 0, 0, 0, time.Local) // Sabtu, 12 Sep 2026
	sabtuSchedule := cfg.GetByHariWithOverrides("sabtu", groupJID, om, refNow)
	if !strings.Contains(sabtuSchedule, "KULIAH TAMBAHAN") || !strings.Contains(sabtuSchedule, "09:00 - 11:30") {
		t.Errorf("Expected Saturday to show extra class, got:\n%s", sabtuSchedule)
	}

	tSeninPagi := time.Date(2026, 9, 7, 7, 15, 0, 0, time.Local)
	nextSenin := cfg.GetNextClassWithOverrides(tSeninPagi, groupJID, om)
	if strings.Contains(nextSenin, "SEDANG BERLANGSUNG") && strings.Contains(nextSenin, "Aljabar") {
		t.Errorf("Expected cancelled/moved class to NOT be ongoing, got:\n%s", nextSenin)
	}
	if !strings.Contains(nextSenin, "Programming Pragmatics") {
		t.Errorf("Expected next class to be Programming Pragmatics, got:\n%s", nextSenin)
	}

	tSelasaSore := time.Date(2026, 9, 8, 15, 10, 0, 0, time.Local)
	nextSelasa := cfg.GetNextClassWithOverrides(tSelasaSore, groupJID, om)
	if !strings.Contains(nextSelasa, "SEDANG BERLANGSUNG") || !strings.Contains(nextSelasa, "Aljabar Linear") {
		t.Errorf("Expected make-up class to be ongoing on Tuesday 15:10, got:\n%s", nextSelasa)
	}

	listReply := om.HandleCommand(groupJID, true, userJID, false, "!jadwalganti", cfg, refNow)
	if !strings.Contains(listReply, "DAFTAR JADWAL PENGGANTI AKTIF") ||
		!strings.Contains(listReply, "RESCHEDULE") ||
		!strings.Contains(listReply, "CANCEL") ||
		!strings.Contains(listReply, "EXTRA") {
		t.Errorf("Expected active override list, got:\n%s", listReply)
	}

	cancelReply := om.HandleCommand(groupJID, true, userJID, true, "!batalganti 1", cfg, refNow)
	if !strings.Contains(cancelReply, "DIBATALKAN") {
		t.Errorf("Expected cancellation of override, got: %s", cancelReply)
	}

	seninDepan := time.Date(2026, 9, 14, 10, 0, 0, 0, time.Local)
	seninDepanSchedule := cfg.GetByHariWithOverrides("hari ini", groupJID, om, seninDepan)
	if strings.Contains(seninDepanSchedule, "DIPINDAHKAN") {
		t.Errorf("Expected next week schedule to be normal without moved status, got:\n%s", seninDepanSchedule)
	}
	tSelasaPagi := time.Date(2026, 9, 8, 6, 30, 0, 0, time.Local)

	msgBesok := cfg.ProcessMessage("!besok", true, groupJID, refNow)
	if !strings.Contains(msgBesok, "KULIAH DITIADAKAN") || !strings.Contains(msgBesok, "Dosen dinas luar") {
		t.Errorf("Expected ProcessMessage('!besok') with group JID to render overrides, got:\n%s", msgBesok)
	}

	bentrokReply := om.HandleCommand(groupJID, true, userJID, true, "!pindah aljabar praktikum | rabu 07:30", cfg, refNow)
	if !strings.Contains(bentrokReply, "PERINGATAN BENTROK JADWAL") ||
		!strings.Contains(bentrokReply, "Arsitektur dan Organisasi Komputer") ||
		!strings.Contains(bentrokReply, "D111-Kelas") {
		t.Errorf("Expected conflict warning for AOK on Rabu 07:30, got:\n%s", bentrokReply)
	}

	paksaReply := om.HandleCommand(groupJID, true, userJID, true, "!pindah aljabar praktikum | rabu 07:30 | paksa", cfg, refNow)
	if !strings.Contains(paksaReply, "BERHASIL DIPINDAHKAN") || !strings.Contains(paksaReply, "dipaksa oleh Admin") {
		t.Errorf("Expected forced reschedule to succeed, got:\n%s", paksaReply)
	}

	bentrokExtra := om.HandleCommand(groupJID, true, userJID, true, "!kuliahganti sbd | rabu 07:00 - 09:00 | Lab 312", cfg, refNow)
	if !strings.Contains(bentrokExtra, "PERINGATAN BENTROK JADWAL") {
		t.Errorf("Expected extra class conflict warning, got:\n%s", bentrokExtra)
	}

	nonAdminLibur := om.HandleCommand(groupJID, true, userJID, false, "!libur besok | Hari Kemerdekaan RI", cfg, refNow)
	if !strings.Contains(nonAdminLibur, "Akses Ditolak") {
		t.Errorf("Expected non-admin holiday command to be rejected, got: %s", nonAdminLibur)
	}

	adminLibur := om.HandleCommand(groupJID, true, userJID, true, "!libur besok | Hari Kemerdekaan RI", cfg, refNow)
	if !strings.Contains(adminLibur, "PENGUMUMAN LIBUR BERHASIL DITETAPKAN") || !strings.Contains(adminLibur, "Hari Kemerdekaan RI") {
		t.Errorf("Expected holiday announcement to succeed, got:\n%s", adminLibur)
	}

	selasaLiburSchedule := cfg.GetByHariWithOverrides("besok", groupJID, om, refNow)
	if !strings.Contains(selasaLiburSchedule, "PENGUMUMAN HARI LIBUR") || !strings.Contains(selasaLiburSchedule, "Hari Kemerdekaan RI") {
		t.Errorf("Expected holiday card on Tuesday, got:\n%s", selasaLiburSchedule)
	}

	nextLibur := cfg.GetNextClassWithOverrides(tSelasaPagi, groupJID, om)
	if !strings.Contains(nextLibur, "Hari Ini Libur Perkuliahan") || !strings.Contains(nextLibur, "Hari Kemerdekaan RI") {
		t.Errorf("Expected next class on holiday to report holiday, got:\n%s", nextLibur)
	}

	_ = selasaDate
	_ = sabtuDate
}
