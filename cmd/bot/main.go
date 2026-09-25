package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/api"
	"bot-jadwal/internal/auth"
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
	webOnly := flag.Bool("web-only", false, "Hanya jalankan Web Dashboard & REST API tanpa koneksi WhatsApp")
	sessionPath := flag.String("session", "", "Path file SQLite sesi WhatsApp (default: storage/sesi_bot.db)")
	port := flag.String("port", "", "Port HTTP server (contoh: 8080 atau :8080)")
	flag.Parse()

	fmt.Println("🚀 [Boot] Memulai Bot WhatsApp Jadwal Kuliah & Web API Server...")

	cfg := config.LoadConfig()
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

	appDB, err := database.InitDB(cfg.AppDBPath)
	if err != nil {
		fmt.Printf("❌ Gagal menginisialisasi database utama: %v\n", err)
		return
	} else {
		fmt.Printf("Berhasil menghubungkan database utama (%s) [WAL Mode]\n", cfg.AppDBPath)
	}

	var academicRepo *academic.Repository
	var authService *auth.Service
	if appDB != nil {
		academicRepo = academic.NewRepository(appDB)
		if cfg.AuthHashKey == "" {
			fmt.Println("❌ BOT_JADWAL_AUTH_HASH_KEY wajib diatur (minimal 32 byte); server tidak dijalankan")
			_ = appDB.Close()
			return
		} else {
			authService, err = auth.NewService(appDB, auth.Config{HashKey: []byte(cfg.AuthHashKey)})
			if err != nil {
				fmt.Printf("❌ Konfigurasi autentikasi pengurus tidak valid: %v\n", err)
				_ = appDB.Close()
				return
			} else {
				fmt.Println("🔐 Autentikasi pengurus siap")
			}
		}
		ctx, cancelAcademicStartup := context.WithTimeout(context.Background(), 30*time.Second)
		count, err := academicRepo.CountClasses(ctx)
		if err == nil && count == 0 {
			seedPath := cfg.DefaultJadwal
			if _, err := os.Stat(seedPath); os.IsNotExist(err) {
				seedPath = "jadwal.json"
			}
			if err := academic.SeedFromJSON(ctx, appDB, seedPath); err != nil {
				fmt.Printf("⚠️  Peringatan seeder otomatis: %v\n", err)
			} else {
				fmt.Printf("🌱 Berhasil menyemai data awal akademik dari %s\n", seedPath)
			}
		} else if count > 0 {
			fmt.Printf("🎓 Berhasil memuat domain akademik (%d kelas aktif di database)\n", count)
		}
		cancelAcademicStartup()
	}

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
	var taskRepo *task.Repository
	if appDB != nil {
		taskManager, err = task.NewTaskManager(appDB)
		if err != nil {
			fmt.Println("ℹ️  [Safe Purge] Modul tugas lama dinonaktifkan (menunggu migrasi ke skema akademik v3)")
		} else {
			fmt.Println("Berhasil menginisialisasi modul tugas")
		}
		taskRepo = task.NewRepository(appDB)
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
				)
			}
		})

		err = botClient.Connect(context.Background())
		if err != nil {
			panic(fmt.Sprintf("Gagal menyambungkan WhatsApp: %v", err))
		}

		reminderManager.StartScheduler(botClient.Client, classManager, chatSettingsManager, taskManager, linkManager)
	} else {
		fmt.Println("🌐 [Mode Web-Only] Berjalan tanpa WhatsApp. Server Linux Azure AMAN 100%.")
	}

	apiServer := api.NewServer(cfg.APIPort, botClient, classManager, taskManager, academicRepo)
	if taskRepo != nil {
		apiServer.SetTaskRepo(taskRepo)
	}
	if authService != nil {
		apiServer.SetAuthService(authService, cfg.SecureCookies)
	}
	_ = apiServer.Start()
	fmt.Printf("👉 Web Dashboard siap diakses: http://localhost%s\n", cfg.APIPort)

	stopSig := make(chan os.Signal, 1)
	signal.Notify(stopSig, os.Interrupt, syscall.SIGTERM)
	<-stopSig

	fmt.Println("\n🛑 [Graceful Shutdown] Sinyal penghentian diterima. Mematikan sistem dengan aman...")

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

	if botClient != nil {
		fmt.Println("⏳ Menutup koneksi database sesi (sesi_bot.db)...")
		if err := botClient.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup sesi_bot.db: %v\n", err)
		}
	}

	fmt.Println("✅ [Graceful Shutdown Selesai] Semua layanan dan database telah ditutup dengan bersih. Sampai jumpa!")
}
