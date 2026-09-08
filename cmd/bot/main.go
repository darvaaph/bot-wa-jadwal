package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bot-jadwal/internal/api"
	"bot-jadwal/internal/bot"
	"bot-jadwal/internal/chat"
	"bot-jadwal/internal/config"
	"bot-jadwal/internal/database"
	"bot-jadwal/internal/link"
	"bot-jadwal/internal/reminder"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"

	"go.mau.fi/whatsmeow/types/events"
)

func main() {
	fmt.Println("🚀 [Boot] Memulai Bot WhatsApp Jadwal Kuliah & Web API Server...")

	// 1. Muat konfigurasi aplikasi & lakukan migrasi file runtime lama ke folder storage/
	cfg := config.LoadConfig()
	if err := cfg.EnsureStorageAndMigrate(); err != nil {
		fmt.Printf("⚠️ Peringatan direktori storage: %v\n", err)
	}

	// 2. Muat seluruh data jadwal kelas (Multi-Class Manager)
	classManager, err := schedule.NewClassManager(cfg.DataJadwalDir, cfg.DefaultJadwal)
	if err != nil {
		fmt.Printf("Peringatan: %v\n", err)
		fmt.Println("Pastikan direktori data/jadwal atau file jadwal.json tersedia.")
		return
	}
	fmt.Printf("Berhasil memuat %d kelas perkuliahan: %v (Kelas Default: %s)\n",
		len(classManager.ListClasses()), classManager.ListClasses(), classManager.GetDefaultClassID())

	// 3. Setup Pengingat Otomatis (Reminder Manager)
	reminderManager := reminder.LoadReminderManager(cfg.ReminderPath)

	// 4. Setup Database Tunggal Aplikasi (SQLite - storage/tugas.db dengan WAL & Busy Timeout)
	appDB, err := database.InitDB(cfg.AppDBPath)
	if err != nil {
		fmt.Printf("Peringatan inisialisasi database utama: %v\n", err)
	} else {
		fmt.Printf("Berhasil menghubungkan database utama (%s) [WAL Mode]\n", cfg.AppDBPath)
	}

	// 5. Setup Pengelola Setelan Chat / Pemilihan Kelas (Chat Settings Manager)
	var chatSettingsManager *chat.ChatSettingsManager
	if appDB != nil {
		chatSettingsManager, err = chat.NewChatSettingsManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul chat_settings: %v\n", err)
		} else {
			upgraded := chatSettingsManager.SyncWithClassManager(classManager)
			if upgraded > 0 {
				fmt.Printf("Berhasil menyelaraskan %d setelan kelas lama ke format kanonikal semester\n", upgraded)
			}
			fmt.Printf("Berhasil menginisialisasi modul setelan kelas (%d chat terhubung)\n", chatSettingsManager.CountSettings())
		}
	}

	// 6. Setup Pengelola Tugas (Task Manager)
	var taskManager *task.TaskManager
	if appDB != nil {
		taskManager, err = task.NewTaskManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul tugas: %v\n", err)
		} else {
			fmt.Println("Berhasil menginisialisasi modul tugas")
		}
	}

	// 7. Setup Pengelola Jadwal Pengganti (Override Manager)
	var overrideManager *schedule.OverrideManager
	if appDB != nil {
		overrideManager, err = schedule.NewOverrideManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul override: %v\n", err)
		} else {
			classManager.SetOverrideManager(overrideManager)
			fmt.Println("Berhasil menginisialisasi modul jadwal pengganti (terhubung ke seluruh kelas)")
		}
	}

	// 8. Setup Pengelola Tautan Penting Kelas (Link Manager)
	var linkManager *link.LinkManager
	if appDB != nil {
		linkManager, err = link.NewLinkManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul tautan: %v\n", err)
		} else {
			fmt.Println("Berhasil menginisialisasi modul tautan penting kelas")
		}
	}

	// 9. Setup Klien WhatsApp (Bot Client & Session Storage)
	botClient, err := bot.NewBotClient(cfg.SessionDBPath)
	if err != nil {
		panic(fmt.Sprintf("Gagal menginisialisasi BotClient: %v", err))
	}

	// 10. Daftarkan Event Handler WhatsApp
	botClient.Client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			fmt.Println("🟢 [Koneksi] Berhasil terhubung ke server WhatsApp!")
		case *events.Disconnected:
			fmt.Println("🟡 [Koneksi] Sambungan ke WhatsApp terputus. Sistem auto-reconnect aktif...")
		case *events.LoggedOut:
			fmt.Printf("🔴 [Koneksi] Sesi WhatsApp logout/unpaired: %s\n", v.PermanentDisconnectDescription())
		case *events.GroupInfo:
			bot.InvalidateGroupAdminCache(v.JID)
		case *events.Message:
			if v.Info.IsFromMe {
				return
			}
			go bot.HandleIncomingMessage(
				botClient.Client,
				v,
				classManager,
				chatSettingsManager,
				reminderManager,
				taskManager,
				overrideManager,
				linkManager,
			)
		}
	})

	// 11. Hubungkan Klien ke WhatsApp
	err = botClient.Connect(context.Background())
	if err != nil {
		panic(fmt.Sprintf("Gagal menyambungkan WhatsApp: %v", err))
	}

	// 12. Jalankan background scheduler pengingat pagi otomatis (06:00 WIB)
	reminderManager.StartScheduler(botClient.Client, classManager, chatSettingsManager, taskManager, linkManager)

	// 13. Jalankan HTTP REST API Server untuk Web Admin Dashboard (Fase A)
	apiServer := api.NewServer(cfg.APIPort, botClient, classManager)
	_ = apiServer.Start()

	// 14. Tangkap sinyal interupsi (Ctrl+C / SIGTERM) untuk Graceful Shutdown
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
	<-stopSig

	fmt.Println("\n🛑 [Graceful Shutdown] Sinyal penghentian diterima. Mematikan sistem dengan aman...")

	// Matikan HTTP REST API Server (toleransi timeout 5 detik)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("⚠️ Gagal mematikan API server: %v\n", err)
	}

	// Putuskan koneksi WhatsApp dan matikan watchdog
	fmt.Println("⏳ Memutuskan koneksi WhatsApp...")
	botClient.Disconnect()

	// Tutup database aplikasi (tugas.db) untuk checkpoint WAL
	if appDB != nil {
		fmt.Println("⏳ Menutup koneksi database aplikasi (tugas.db)...")
		if err := appDB.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup tugas.db: %v\n", err)
		}
	}

	// Tutup database sesi bot (sesi_bot.db)
	fmt.Println("⏳ Menutup koneksi database sesi (sesi_bot.db)...")
	if err := botClient.Close(); err != nil {
		fmt.Printf("⚠️ Gagal menutup sesi_bot.db: %v\n", err)
	}

	fmt.Println("✅ [Graceful Shutdown Selesai] Semua layanan dan database telah ditutup dengan bersih. Sampai jumpa!")
}
