package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// ReminderGroup menyimpan data grup WhatsApp yang terdaftar menerima pengingat otomatis
type ReminderGroup struct {
	JID     string    `json:"jid"`
	Name    string    `json:"name"`
	AddedAt time.Time `json:"added_at"`
}

// ReminderConfig menyimpan konfigurasi jadwal pengingat dan daftar grup
type ReminderConfig struct {
	Hour    int             `json:"hour"`   // Jam pengingat (default 6)
	Minute  int             `json:"minute"` // Menit pengingat (default 30)
	Groups  []ReminderGroup `json:"groups"`
}

// ReminderManager mengelola scheduler dan persistensi data grup pengingat
type ReminderManager struct {
	mu          sync.RWMutex
	filePath    string
	config      ReminderConfig
	lastRunDate string
}

// LoadReminderManager memuat daftar grup pengingat dari file JSON
func LoadReminderManager(filepath string) *ReminderManager {
	rm := &ReminderManager{
		filePath: filepath,
		config: ReminderConfig{
			Hour:   6,
			Minute: 30,
			Groups: []ReminderGroup{},
		},
	}

	data, err := os.ReadFile(filepath)
	if err == nil {
		var cfg ReminderConfig
		if err := json.Unmarshal(data, &cfg); err == nil {
			if cfg.Hour == 0 && cfg.Minute == 0 {
				cfg.Hour = 6
				cfg.Minute = 30
			}
			rm.config = cfg
		}
	}

	return rm
}

// Save menyimpan konfigurasi grup pengingat ke file JSON
func (rm *ReminderManager) Save() error {
	data, err := json.MarshalIndent(rm.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rm.filePath, data, 0644)
}

// AddGroup mendaftarkan grup baru ke daftar pengingat
func (rm *ReminderManager) AddGroup(jid string, name string) (bool, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, g := range rm.config.Groups {
		if g.JID == jid {
			return false, fmt.Sprintf("ℹ️ Grup *\"%s\"* sudah terdaftar dalam pengingat otomatis (Pukul %02d:%02d WIB).", g.Name, rm.config.Hour, rm.config.Minute)
		}
	}

	if name == "" {
		name = "Grup WhatsApp"
	}

	rm.config.Groups = append(rm.config.Groups, ReminderGroup{
		JID:     jid,
		Name:    name,
		AddedAt: time.Now(),
	})

	_ = rm.Save()
	return true, fmt.Sprintf("✅ *Pengingat Otomatis Diaktifkan!*\nJadwal kuliah akan otomatis dikirim ke grup ini setiap hari *Senin - Jumat* pukul *%02d:%02d WIB*.\n\nKetik `!reminder test` untuk mencoba simulasi pengiriman.", rm.config.Hour, rm.config.Minute)
}

// RemoveGroup menghapus grup dari daftar pengingat otomatis
func (rm *ReminderManager) RemoveGroup(jid string) (bool, string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	foundIndex := -1
	for i, g := range rm.config.Groups {
		if g.JID == jid {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return false, "ℹ️ Grup ini belum terdaftar dalam pengingat otomatis."
	}

	rm.config.Groups = append(rm.config.Groups[:foundIndex], rm.config.Groups[foundIndex+1:]...)
	_ = rm.Save()
	return true, "🔕 *Pengingat Otomatis Dinonaktifkan!*\nGrup ini tidak akan menerima jadwal otomatis harian lagi."
}

// Status menampilkan informasi status pengingat saat ini
func (rm *ReminderManager) Status(currentJID string) string {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	isRegistered := false
	for _, g := range rm.config.Groups {
		if g.JID == currentJID {
			isRegistered = true
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("⏰ *STATUS PENGINGAT OTOMATIS*\n")
	sb.WriteString("──────────\n")
	sb.WriteString(fmt.Sprintf("• Waktu Kirim : Setiap %02d:%02d WIB (Senin-Jumat)\n", rm.config.Hour, rm.config.Minute))
	sb.WriteString(fmt.Sprintf("• Total Grup  : %d grup aktif\n", len(rm.config.Groups)))
	if isRegistered {
		sb.WriteString("• Status Chat : 🟢 *Aktif di chat/grup ini*\n")
	} else {
		sb.WriteString("• Status Chat : ⚪ *Belum aktif di chat/grup ini*\n")
	}
	sb.WriteString("\n*Perintah Pengingat:*\n")
	sb.WriteString("• `!reminder on`   : Aktifkan di grup ini\n")
	sb.WriteString("• `!reminder off`  : Matikan di grup ini\n")
	sb.WriteString("• `!reminder test` : Uji kirim pengingat sekarang\n")
	sb.WriteString("──────────")
	return sb.String()
}

// BuildMorningReminder menyusun pesan pengingat pagi lengkap dengan alert tugas mendesak
func BuildMorningReminder(chatJID string, config *JadwalConfig, taskManager *TaskManager, now time.Time) string {
	var holiday *ScheduleOverride
	if config.OverrideManager != nil {
		holiday = config.OverrideManager.GetHolidayOverride(chatJID, now)
	}

	var pesan string
	if holiday != nil {
		hariIndo := getHariIndonesia(now)
		tglIndo := now.Format("02-01-2006")
		pesan = fmt.Sprintf("🌴 *SELAMAT PAGI & SELAMAT BERLIBUR!*\n──────────\nHari ini (*%s, %s*) perkuliahan diliburkan:\n📢 *%s*\n\nSeluruh jadwal perkuliahan ditiadakan. Selamat beristirahat dan menikmati hari libur! ✨",
			hariIndo, tglIndo, holiday.Alasan)
	} else {
		jadwalPagi := config.GetByHari("hari ini", now)
		if config.OverrideManager != nil {
			jadwalPagi = config.GetByHariWithOverrides("hari ini", chatJID, config.OverrideManager, now)
		}
		pesan = fmt.Sprintf("🌅 *SELAMAT PAGI!*\nBerikut jadwal perkuliahan hari ini:\n\n%s", jadwalPagi)
	}

	if taskManager != nil {
		urgentTasks, err := taskManager.GetDueTasks(chatJID, "urgent", now)
		if err == nil && len(urgentTasks) > 0 {
			var sb strings.Builder
			sb.WriteString("\n\n──────────\n")
			sb.WriteString("🚨 *PERINGATAN DEADLINE TUGAS:*\n")
			for _, ut := range urgentTasks {
				badge := GetUrgencyBadge(ut.DeadlineAt, now)
				sb.WriteString(fmt.Sprintf("• [%s] %s\n  └ %s\n", strings.ToUpper(ut.Matkul), ut.Deskripsi, badge))
			}
			sb.WriteString("──────────\n")
			sb.WriteString("_Ketik `!tugas` untuk detail instruksi tugas._")
			pesan += sb.String()
		}
	}

	return pesan
}

// StartScheduler menjalankan background goroutine untuk broadcast jadwal otomatis dengan dukungan multi-kelas
func (rm *ReminderManager) StartScheduler(
	client *whatsmeow.Client,
	classMgr *ClassManager,
	settingsMgr *ChatSettingsManager,
	taskManager *TaskManager,
) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now()
			// Hanya berjalan hari Senin s.d. Jumat
			if now.Weekday() < time.Monday || now.Weekday() > time.Friday {
				continue
			}

			rm.mu.RLock()
			targetHour := rm.config.Hour
			targetMinute := rm.config.Minute
			groups := make([]ReminderGroup, len(rm.config.Groups))
			copy(groups, rm.config.Groups)
			rm.mu.RUnlock()

			todayStr := now.Format("2006-01-02")
			// Cek apakah tepat jam pengingat dan belum pernah dikirim hari ini
			if now.Hour() == targetHour && now.Minute() == targetMinute && rm.lastRunDate != todayStr {
				rm.lastRunDate = todayStr

				if len(groups) == 0 {
					continue
				}

				fmt.Printf("[Scheduler] Mengirim pengingat jadwal pagi (%s) ke %d grup...\n", todayStr, len(groups))

				for _, g := range groups {
					var classConfig *JadwalConfig
					if settingsMgr != nil && classMgr != nil {
						classID := settingsMgr.GetClass(g.JID)
						if classID == "" {
							fmt.Printf("[Scheduler] Grup %s (%s) belum memilih kelas, mengirim peringatan onboarding...\n", g.Name, g.JID)
							pesanWarning := "⏰ *PENGINGAT PAGI OTOMATIS GAGAL DIKIRIM*\n──────────\nGrup ini belum menentukan kelas perkuliahan aktif.\nSilakan tentukan kelas terlebih dahulu dengan perintah:\n👉 `!setkelas [nama_kelas]` (Contoh: `!setkelas D4-TI-1A`)\n\nKetik `!daftarkelas` untuk melihat 19 pilihan kelas yang tersedia."
							targetJID, err := types.ParseJID(g.JID)
							if err == nil {
								_, _ = client.SendMessage(context.Background(), targetJID, &waE2E.Message{
									Conversation: proto.String(pesanWarning),
								})
							}
							continue
						}
						classConfig = classMgr.GetClassOrDefault(classID)
					} else if classMgr != nil {
						classConfig = classMgr.GetDefaultClass()
					}
					if classConfig == nil {
						continue
					}

					pesanGrup := BuildMorningReminder(g.JID, classConfig, taskManager, now)

					targetJID, err := types.ParseJID(g.JID)
					if err != nil {
						fmt.Printf("[Scheduler] Error parse JID %s: %v\n", g.JID, err)
						continue
					}

					_, err = client.SendMessage(context.Background(), targetJID, &waE2E.Message{
						Conversation: proto.String(pesanGrup),
					})
					if err != nil {
						fmt.Printf("[Scheduler] Gagal kirim ke grup %s (%s): %v\n", g.Name, g.JID, err)
					} else {
						fmt.Printf("[Scheduler] Sukses kirim pengingat ke grup %s\n", g.Name)
					}
					// Jeda singkat antar grup untuk menghindari rate limit WA
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()
}
