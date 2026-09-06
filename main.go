package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// isSenderGroupAdmin memeriksa apakah pengirim pesan merupakan admin atau superadmin di grup
func isSenderGroupAdmin(ctx context.Context, client *whatsmeow.Client, groupJID, senderJID types.JID) bool {
	info, err := client.GetGroupInfo(ctx, groupJID)
	if err != nil || info == nil {
		return false
	}
	for _, p := range info.Participants {
		if p.JID.User == senderJID.User {
			return p.IsAdmin || p.IsSuperAdmin
		}
	}
	return false
}

// resolveSenderAdmin mengembalikan status hak akses admin (selalu true di DM pribadi, atau cek admin grup di grup WA)
func resolveSenderAdmin(ctx context.Context, client *whatsmeow.Client, isGroup bool, groupJID, senderJID types.JID) bool {
	if !isGroup {
		return true
	}
	return isSenderGroupAdmin(ctx, client, groupJID, senderJID)
}

// replyWithTyping mengirimkan reaksi emoji, simulasi status mengetik, dan pesan balasan ke pengguna
func replyWithTyping(
	ctx context.Context,
	client *whatsmeow.Client,
	chat types.JID,
	sender types.JID,
	msgID types.MessageID,
	replyText string,
	emoji string,
	typingDuration time.Duration,
	actionName string,
) {
	if replyText == "" {
		return
	}

	// 1. Berikan reaksi emoji pada pesan yang dikirim pengguna
	if emoji != "" {
		reactionMsg := client.BuildReaction(chat, sender, msgID, emoji)
		_, _ = client.SendMessage(ctx, chat, reactionMsg)
	}

	// 2. Simulasi status "sedang mengetik..."
	if typingDuration <= 0 {
		typingDuration = 600 * time.Millisecond
	}
	_ = client.SendChatPresence(ctx, chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
	time.Sleep(typingDuration)
	_ = client.SendChatPresence(ctx, chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

	// 3. Kirim pesan balasan
	_, err := client.SendMessage(ctx, chat, &waE2E.Message{
		Conversation: proto.String(replyText),
	})
	if err != nil {
		fmt.Printf("Gagal mengirim balasan %s ke %s: %v\n", actionName, chat.User, err)
	} else {
		fmt.Printf("Sukses membalas %s ke %s\n", actionName, chat.User)
	}
}

func main() {
	// 1. Muat seluruh data jadwal kelas (Multi-Class Manager)
	classManager, err := NewClassManager("data/jadwal", "jadwal.json")
	if err != nil {
		fmt.Printf("Peringatan: %v\n", err)
		fmt.Println("Pastikan direktori data/jadwal atau file jadwal.json tersedia.")
		return
	}
	fmt.Printf("Berhasil memuat %d kelas perkuliahan: %v (Kelas Default: %s)\n", len(classManager.ListClasses()), classManager.ListClasses(), classManager.GetDefaultClassID())

	// 2. Setup Pengingat Otomatis (Reminder Manager)
	reminderManager := LoadReminderManager("reminder_groups.json")

	// 3. Setup Database Tunggal Aplikasi (SQLite - tugas.db dengan WAL & Busy Timeout)
	appDB, err := InitDB("tugas.db")
	if err != nil {
		fmt.Printf("Peringatan inisialisasi database utama: %v\n", err)
	} else {
		fmt.Println("Berhasil menghubungkan database utama (tugas.db) [WAL Mode]")
	}

	// 4. Setup Pengelola Setelan Chat / Pemilihan Kelas (Chat Settings Manager - SQLite)
	var chatSettingsManager *ChatSettingsManager
	if appDB != nil {
		chatSettingsManager, err = NewChatSettingsManager(appDB)
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

	// 5. Setup Pengelola Tugas (Task Manager - SQLite)
	var taskManager *TaskManager
	if appDB != nil {
		taskManager, err = NewTaskManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul tugas: %v\n", err)
		} else {
			fmt.Println("Berhasil menginisialisasi modul tugas")
		}
	}

	// 6. Setup Pengelola Jadwal Pengganti (Override Manager - SQLite)
	var overrideManager *OverrideManager
	if appDB != nil {
		overrideManager, err = NewOverrideManager(appDB)
		if err != nil {
			fmt.Printf("Peringatan inisialisasi modul override: %v\n", err)
		} else {
			classManager.SetOverrideManager(overrideManager)
			fmt.Println("Berhasil menginisialisasi modul jadwal pengganti (terhubung ke seluruh kelas)")
		}
	}

	// 4. Setup Database Log & SQLite (Session Storage)
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	
	// Gunakan driver "sqlite" (pure Go tanpa CGO/GCC)
	container, err := sqlstore.New(context.Background(), "sqlite", "file:sesi_bot.db?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		panic(err)
	}

	// 5. Ambil sesi dari database
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		panic(err)
	}

	// 6. Setup Klien WhatsApp
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Ketahanan Sambungan Internet (Auto-Reconnect Resilience)
	client.EnableAutoReconnect = true
	client.AutoReconnectHook = func(err error) bool {
		fmt.Printf("⚠️ [Auto-Reconnect] Sambungan putus (%v). Mencoba menyambung kembali...\n", err)
		return true // Pantang menyerah, terus mencoba terhubung
	}

	// 7. Daftarkan Event Handler untuk memproses pesan masuk dan status koneksi
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Connected:
			fmt.Println("🟢 [Koneksi] Berhasil terhubung ke server WhatsApp!")
		case *events.Disconnected:
			fmt.Println("🟡 [Koneksi] Sambungan ke WhatsApp terputus. Sistem auto-reconnect aktif...")
		case *events.LoggedOut:
			fmt.Printf("🔴 [Koneksi] Sesi WhatsApp logout/unpaired: %s\n", v.PermanentDisconnectDescription())
		case *events.Message:
			// Abaikan pesan jika dikirim oleh bot sendiri
			if v.Info.IsFromMe {
				return
			}

			// Ekstraksi teks pesan dari tipe Conversation atau ExtendedTextMessage
			var msgText string
			if v.Message.GetConversation() != "" {
				msgText = v.Message.GetConversation()
			} else if v.Message.GetExtendedTextMessage() != nil && v.Message.GetExtendedTextMessage().GetText() != "" {
				msgText = v.Message.GetExtendedTextMessage().GetText()
			}

			msgText = strings.TrimSpace(msgText)
			if msgText == "" {
				return
			}

			// Log pesan yang diterima di konsol
			fmt.Printf("[Pesan Masuk dari %s]: %s\n", v.Info.Sender.User, msgText)

			lowerMsg := strings.ToLower(msgText)

			// Tentukan jadwal kelas aktif untuk chat/grup ini secara dinamis (Multi-Tenant)
			var activeClassID string
			if chatSettingsManager != nil {
				activeClassID = chatSettingsManager.GetClass(v.Info.Chat.String())
			}
			activeJadwal := classManager.GetClassOrDefault(activeClassID)

			// 1. Handler Khusus Perintah Pengaturan Kelas (!kelas / !daftarkelas / !setkelas / !pilihkelas / !resetkelas)
			if chatSettingsManager != nil && matchCommandPrefix(msgText, v.Info.IsGroup, "kelas", "daftarkelas", "setkelas", "pilihkelas", "resetkelas") {
				isAdmin := resolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender)
				classReply := chatSettingsManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, classManager)
				if classReply != "" {
					replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, classReply, "🏫", 600*time.Millisecond, "perintah kelas")
					return
				}
			}

			// 2. Handler Khusus Perintah Reload Jadwal Seluruh Kelas (!reload)
			if matchCommandPrefix(msgText, v.Info.IsGroup, "reload") {
				count, errs := classManager.ReloadAll()
				var reloadReply string
				if len(errs) > 0 {
					reloadReply = fmt.Sprintf("⚠️ Berhasil memuat ulang %d kelas, namun terdapat error: %v", count, errs)
				} else {
					reloadReply = fmt.Sprintf("🔄 *BERHASIL MEMUAT ULANG JADWAL!*\n──────────\nSeluruh konfigurasi jadwal (%d kelas) berhasil disegarkan dari disk ke memori.", count)
				}
				replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, reloadReply, "🔄", 600*time.Millisecond, "perintah reload")
				return
			}

			// 3. Handler Khusus Perintah Pengingat Otomatis (!reminder / !pengingat)
			if matchCommandPrefix(msgText, v.Info.IsGroup, "reminder", "pengingat") {
				parts := strings.Fields(lowerMsg)
				subCmd := ""
				if len(parts) > 1 {
					subCmd = parts[1]
				}

				var reminderReply string
				switch subCmd {
				case "on", "aktif", "start", "enable":
					if activeClassID == "" {
						reminderReply = "⚠️ *PENGINGAT TIDAK DAPAT DIAKTIFKAN*\n──────────\nChat ini belum menentukan kelas perkuliahan.\nSilakan atur kelas terlebih dahulu dengan perintah:\n👉 `!setkelas [nama_kelas]` (Contoh: `!setkelas D4-TI-1A`)\n\nKetik `!daftarkelas` untuk melihat 19 pilihan kelas yang tersedia."
					} else {
						chatJID := v.Info.Chat.String()
						groupName := "Grup Chat"
						if v.Info.IsGroup {
							info, err := client.GetGroupInfo(context.Background(), v.Info.Chat)
							if err == nil && info != nil && info.Name != "" {
								groupName = info.Name
							}
						}
						_, reminderReply = reminderManager.AddGroup(chatJID, groupName)
					}

				case "off", "nonaktif", "stop", "disable", "matikan":
					chatJID := v.Info.Chat.String()
					_, reminderReply = reminderManager.RemoveGroup(chatJID)

				case "test", "tes", "try":
					if activeClassID == "" && chatSettingsManager != nil {
						reminderReply = chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup)
					} else {
						reminderReply = fmt.Sprintf("🧪 *[SIMULASI PENGINGAT PAGI]*\n\n%s", BuildMorningReminder(v.Info.Chat.String(), activeJadwal, taskManager, time.Now()))
					}

				default:
					reminderReply = reminderManager.Status(v.Info.Chat.String())
				}

				replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, reminderReply, "⏰", 600*time.Millisecond, "perintah reminder")
				return
			}

			// 4. Handler Khusus Perintah Tugas (!tugas)
			if taskManager != nil && matchCommandPrefix(msgText, v.Info.IsGroup, "tugas") {
				if activeClassID == "" && chatSettingsManager != nil {
					replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding tugas")
					return
				}
				isAdmin := resolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender)
				tugasReply := taskManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now())
				replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, tugasReply, "📝", 600*time.Millisecond, "perintah tugas")
				return
			}

			// 5. Handler Khusus Perintah Jadwal Pengganti / Override (!pindah, !kosong, !kuliahganti, !jadwalganti, !batalganti)
			if overrideManager != nil && matchCommandPrefix(msgText, v.Info.IsGroup, "pindah", "ganti", "kosong", "libur", "kuliahganti", "tambahkelas", "jadwalganti", "overrides", "batalganti") {
				if activeClassID == "" && chatSettingsManager != nil {
					replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 600*time.Millisecond, "onboarding override")
					return
				}
				isAdmin := resolveSenderAdmin(context.Background(), client, v.Info.IsGroup, v.Info.Chat, v.Info.Sender)
				overrideReply := overrideManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, activeJadwal, time.Now())
				replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, overrideReply, "🔄", 600*time.Millisecond, "perintah override")
				return
			}

			// Proses pesan masuk dengan parser perintah jadwal (menerapkan aturan Hybrid & Override)
			replyText := activeJadwal.ProcessMessage(msgText, v.Info.IsGroup, v.Info.Chat.String())

			// Jika pesan cocok dengan salah satu perintah, kirim pesan balasan
			if replyText != "" {
				if activeClassID == "" && chatSettingsManager != nil {
					if isMenuOrHelpCommand(msgText, v.Info.IsGroup) {
						menuReply := chatSettingsManager.BuildUnconfiguredMenu(v.Info.IsGroup)
						replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, menuReply, "📅", 700*time.Millisecond, "menu unconfigured")
						return
					}
					if strings.Contains(replyText, "tidak dikenali") {
						replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, replyText, "⚠️", 700*time.Millisecond, fmt.Sprintf("perintah '%s'", msgText))
						return
					}
					// Jika chat belum memilih kelas, berikan panduan onboarding alih-alih menampilkan kelas default
					replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, chatSettingsManager.GetOnboardingPrompt(v.Info.IsGroup), "👋", 700*time.Millisecond, "onboarding jadwal")
					return
				}

				replyWithTyping(context.Background(), client, v.Info.Chat, v.Info.Sender, v.Info.ID, replyText, "📅", 700*time.Millisecond, fmt.Sprintf("perintah '%s'", msgText))
			}
		}
	})

	// 7. Logika Login & QR Code
	if client.Store.ID == nil {
		// Jika belum ada sesi login, minta QR Code
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render QR Code di terminal
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("Silakan scan QR Code di atas menggunakan WhatsApp!")
			} else {
				fmt.Println("Status Login:", evt.Event)
			}
		}
	} else {
		// Jika sudah login sebelumnya, langsung connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		fmt.Println("Bot berhasil terhubung ke WhatsApp!")
	}

	// 8. Jalankan background scheduler pengingat pagi otomatis (06:30 WIB) dengan multi-kelas
	reminderManager.StartScheduler(client, classManager, chatSettingsManager, taskManager)

	// 9. Jalankan Watchdog Supervisor untuk auto-reconnect berkala dengan exponential backoff
	watchdogCtx, cancelWatchdog := context.WithCancel(context.Background())
	go func() {
		backoff := 3 * time.Second
		maxBackoff := 30 * time.Second
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-watchdogCtx.Done():
				return
			case <-ticker.C:
				if client.IsLoggedIn() && !client.IsConnected() {
					fmt.Printf("⚠️ [Watchdog] Terdeteksi offline. Menjalankan auto-reconnect (jeda %v)...\n", backoff)
					time.Sleep(backoff)
					err := client.Connect()
					if err != nil {
						fmt.Printf("⚠️ [Watchdog] Gagal menyambung kembali: %v\n", err)
						backoff *= 2
						if backoff > maxBackoff {
							backoff = maxBackoff
						}
					} else {
						fmt.Println("🟢 [Watchdog] Koneksi WhatsApp berhasil dipulihkan!")
						backoff = 3 * time.Second
					}
				} else if client.IsConnected() {
					backoff = 3 * time.Second
				}
			}
		}
	}()

	// 10. Tahan program sampai menerima sinyal interupsi (Ctrl+C / SIGTERM)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\n🛑 [Graceful Shutdown] Sinyal penghentian diterima. Mematikan sistem dengan aman...")
	cancelWatchdog()

	// 1. Putuskan koneksi WhatsApp
	fmt.Println("⏳ Memutuskan koneksi WhatsApp...")
	client.Disconnect()

	// 2. Tutup database aplikasi (tugas.db) secara bersih untuk checkpoint WAL
	if appDB != nil {
		fmt.Println("⏳ Menutup koneksi database aplikasi (tugas.db)...")
		if err := appDB.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup tugas.db: %v\n", err)
		}
	}

	// 3. Tutup database sesi bot (sesi_bot.db)
	if container != nil {
		fmt.Println("⏳ Menutup koneksi database sesi (sesi_bot.db)...")
		if err := container.Close(); err != nil {
			fmt.Printf("⚠️ Gagal menutup sesi_bot.db: %v\n", err)
		}
	}

	fmt.Println("✅ [Graceful Shutdown Selesai] Semua database dan koneksi telah ditutup dengan bersih. Sampai jumpa!")
}