package main

import (
	"context"
	"flag"
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
	"bot-jadwal/internal/notify"
	"bot-jadwal/internal/reminder"
	"bot-jadwal/internal/schedule"
	"bot-jadwal/internal/task"

	"go.mau.fi/whatsmeow/types/events"
)

func main() {
	webOnly := flag.Bool("web-only", false, "Hanya jalankan Web Dashboard & REST API tanpa koneksi WhatsApp")
	sessionPath := flag.String("session", "", "Path file SQLite sesi WhatsApp (default: storage/sesi_bot.db)")
	port := flag.String("port", "", "Port HTTP server (contoh: 8080 atau :8080)")
	flag.Parse()

	fmt.Println("🚀 [Boot] Memulai Bot WhatsApp Jadwal Kuliah & Web API Server...")

	cfg := config.LoadConfig()
	// BE-013: validasi security sebelum server menerima traffic.
	// Production gagal bila hash key, secure cookie, origin, atau proxy invalid.
	if err := cfg.Validate(); err != nil {
		fmt.Printf("❌ Konfigurasi security tidak valid: %v\n", err)
		return
	}
	if *sessionPath != "" {
		cfg.SessionDBPath = *sessionPath
	}
	if *port != "" {
		p := *port
		if p[0] != ':' {
			p = ":" + p
		}
		cfg.APIPort = p
	}

	if err := cfg.EnsureStorageAndMigrate(); err != nil {
		fmt.Printf("⚠️ Peringatan direktori storage: %v\n", err)
	}

	classManager, err := schedule.NewClassManager(cfg.DataJadwalDir, cfg.DefaultJadwal)
	if err != nil {
		fmt.Printf("Peringatan: %v\n", err)
		fmt.Println("Pastikan direktori data/jadwal atau file jadwal.json tersedia.")
		return
	}
	fmt.Printf("Berhasil memuat %d kelas perkuliahan: %v (Kelas Default: %s)\n",
		len(classManager.ListClasses()), classManager.ListClasses(), classManager.GetDefaultClassID())

	reminderManager := reminder.LoadReminderManager(cfg.ReminderPath)

	appDB, err := database.OpenPool(cfg.AppDBPath)
	if err != nil {
		fmt.Printf("❌ Gagal menginisialisasi database utama: %v\n", err)
		return
	} else {
		fmt.Printf("Berhasil menghubungkan database utama (%s) [WAL Mode]\n", cfg.AppDBPath)
	}

	// 4b. Setup Database v1 (SQLite - storage/bot_v1.db dengan WAL & Busy Timeout)
	v1DB, err := database.InitDB(cfg.V1DBPath)
	if err != nil {
		fmt.Printf("Peringatan inisialisasi database v1: %v\n", err)
	} else {
		fmt.Printf("Berhasil menghubungkan database v1 (%s) [WAL Mode]\n", cfg.V1DBPath)
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

	var taskManager *task.TaskManager
	if appDB != nil {
		taskManager, err = task.NewTaskManager(appDB)
		if err != nil {
			fmt.Println("ℹ️  [Safe Purge] Modul tugas lama dinonaktifkan (menunggu migrasi ke skema akademik v3)")
		} else {
			fmt.Println("Berhasil menginisialisasi modul tugas")
		}
	}

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

	var linkManager *link.LinkManager
	if appDB != nil {
		linkManager, err = link.NewLinkManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul tautan: %v\n", err)
		} else {
			fmt.Println("Berhasil menginisialisasi modul tautan penting kelas")
		}
	}

	var botClient *bot.BotClient
	var notifWorker *bot.NotificationWorker
	if !*webOnly {
		var err error
		botClient, err = bot.NewBotClient(cfg.SessionDBPath)
		if err != nil {
			panic(fmt.Sprintf("Gagal menginisialisasi BotClient: %v", err))
		}

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
					v1DB,
				)
			}
		})

		err = botClient.Connect(context.Background())
		if err != nil {
			panic(fmt.Sprintf("Gagal menyambungkan WhatsApp: %v", err))
		}

		reminderManager.StartScheduler(botClient.Client, classManager, chatSettingsManager, taskManager, linkManager)

		// 12b. Jalankan background worker siaran notifikasi WhatsApp (bot_v1.db)
		if v1DB != nil {
			notifWorker = bot.NewNotificationWorker(v1DB, botClient.Client, 5*time.Second)
			notifWorker.Start()
		}
	} else {
		fmt.Println("🌐 [Mode Web-Only] Berjalan tanpa WhatsApp. Server Linux Azure AMAN 100%.")
	}

	// 13. Jalankan HTTP REST API Server untuk Web Admin Dashboard dan API v1
	apiServer := api.NewServer(cfg.APIPort, botClient, classManager, taskManager, v1DB)
	apiServer.SetSecureCookies(cfg.SecureCookies)
	apiServer.SetSecurityOptions(api.SecurityOptions{
		Env:               cfg.Env,
		AuthHashKey:       cfg.AuthHashKey,
		AllowedOrigins:    cfg.AllowedOrigins,
		TrustedProxyCIDRs: cfg.TrustedProxyCIDRs,
		PublicBaseURL:     cfg.PublicBaseURL,
	})
	if err := apiServer.Start(); err != nil {
		fmt.Printf("❌ Gagal memulai server Web API: %v\n", err)
		return
	}
	fmt.Printf("👉 Web Dashboard siap diakses: http://localhost%s\n", cfg.APIPort)
	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
	<-stopSig

	fmt.Println("\n🛑 [Graceful Shutdown] Sinyal penghentian diterima. Mematikan sistem dengan aman...")

	// Hentikan background worker notifikasi WhatsApp
	if notifWorker != nil {
		fmt.Println("⏳ Menghentikan background worker notifikasi WhatsApp...")
		notifWorker.Stop()
	}

	// Matikan HTTP REST API Server (toleransi timeout 5 detik)
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("⚠️ Gagal mematikan API server: %v\n", err)
	}

	if botClient != nil {
		fmt.Println("⏳ Memutuskan koneksi WhatsApp...")
		botClient.Disconnect()
	}

	// Tutup database aplikasi (tugas.db) untuk checkpoint WAL
	if appDB != nil {
		fmt.Println("⏳ Menutup koneksi database aplikasi (tugas.db)...")
		if err := appDB.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup tugas.db: %v\n", err)
		}
	}

	// Tutup database v1 (bot_v1.db) untuk checkpoint WAL
	if v1DB != nil {
		fmt.Println("⏳ Menutup koneksi database v1 (bot_v1.db)...")
		if err := v1DB.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup bot_v1.db: %v\n", err)
		}
	}

	// Tutup database sesi bot (sesi_bot.db)
	if botClient != nil {
		fmt.Println("⏳ Menutup koneksi database sesi (sesi_bot.db)...")
		if err := botClient.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup sesi_bot.db: %v\n", err)
		}
	}

	fmt.Println("✅ [Graceful Shutdown Selesai] Semua layanan dan database telah ditutup dengan bersih. Sampai jumpa!")
}

// runNotifyScheduler enqueues due reminders every 30s and drains the outbox when a sender exists.
// Web publish never depends on WA: enqueue always runs, delivery is best-effort.
func runNotifyScheduler(svc *notify.Service, sender notify.Sender) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UTC()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		_, _ = svc.EnsureDailySummaries(ctx, now)
		_, _ = svc.EnsureTaskReminders(ctx, now)
		_, _ = svc.EnsureReplacementReminders(ctx, now)
		if sender != nil {
			_, _, _ = svc.ProcessDue(ctx, sender, 20, now)
		}
		cancel()
	}
}
