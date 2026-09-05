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

func main() {
	// 1. Muat data jadwal dari file JSON
	jadwalData, err := LoadJadwal("jadwal.json")
	if err != nil {
		fmt.Printf("Peringatan: %v\n", err)
		fmt.Println("Pastikan file jadwal.json tersedia di direktori yang sama.")
		return
	}
	fmt.Printf("Berhasil memuat data jadwal: %s (%d jadwal mata kuliah)\n", jadwalData.Kampus, len(jadwalData.Jadwal))

	// 2. Setup Pengingat Otomatis (Reminder Manager)
	reminderManager := LoadReminderManager("reminder_groups.json")

	// 3. Setup Pengelola Tugas (Task Manager - SQLite)
	taskManager, err := NewTaskManager("tugas.db")
	if err != nil {
		fmt.Printf("Peringatan inisialisasi database tugas: %v\n", err)
	} else {
		defer taskManager.Close()
		fmt.Println("Berhasil menghubungkan database tugas (tugas.db)")
	}

	// 4. Setup Pengelola Jadwal Pengganti (Override Manager - SQLite)
	overrideManager, err := NewOverrideManager("tugas.db")
	if err != nil {
		fmt.Printf("Peringatan inisialisasi database override: %v\n", err)
	} else {
		defer overrideManager.Close()
		jadwalData.SetOverrideManager(overrideManager)
		fmt.Println("Berhasil menghubungkan database jadwal pengganti")
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

	// 7. Daftarkan Event Handler untuk memproses pesan masuk
	client.AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
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

			// Handler Khusus Perintah Pengingat Otomatis (!reminder / !pengingat)
			if strings.HasPrefix(lowerMsg, "!reminder") || strings.HasPrefix(lowerMsg, "/reminder") ||
				strings.HasPrefix(lowerMsg, "#reminder") || strings.HasPrefix(lowerMsg, "!pengingat") ||
				strings.HasPrefix(lowerMsg, "/pengingat") || strings.HasPrefix(lowerMsg, "#pengingat") ||
				(!v.Info.IsGroup && (strings.HasPrefix(lowerMsg, "reminder") || strings.HasPrefix(lowerMsg, "pengingat"))) {

				parts := strings.Fields(lowerMsg)
				subCmd := ""
				if len(parts) > 1 {
					subCmd = parts[1]
				}

				var reminderReply string
				switch subCmd {
				case "on", "aktif", "start", "enable":
					chatJID := v.Info.Chat.String()
					groupName := "Grup Chat"
					if v.Info.IsGroup {
						info, err := client.GetGroupInfo(context.Background(), v.Info.Chat)
						if err == nil && info != nil && info.Name != "" {
							groupName = info.Name
						}
					}
					_, reminderReply = reminderManager.AddGroup(chatJID, groupName)

				case "off", "nonaktif", "stop", "disable", "matikan":
					chatJID := v.Info.Chat.String()
					_, reminderReply = reminderManager.RemoveGroup(chatJID)

				case "test", "tes", "try":
					reminderReply = fmt.Sprintf("🧪 *[SIMULASI PENGINGAT PAGI]*\n\n%s", BuildMorningReminder(v.Info.Chat.String(), jadwalData, taskManager, time.Now()))

				default:
					reminderReply = reminderManager.Status(v.Info.Chat.String())
				}

				// 1. Berikan reaksi emoji pada pesan pengingat
				reactionMsg := client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "⏰")
				_, _ = client.SendMessage(context.Background(), v.Info.Chat, reactionMsg)

				// 2. Simulasi "sedang mengetik..." selama 600ms
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
				time.Sleep(600 * time.Millisecond)
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

				// 3. Kirim balasan perintah reminder
				_, err := client.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
					Conversation: proto.String(reminderReply),
				})
				if err != nil {
					fmt.Printf("Gagal mengirim balasan reminder: %v\n", err)
				} else {
					fmt.Printf("Sukses membalas perintah reminder ke %s\n", v.Info.Chat.User)
				}
				return
			}

			// Handler Khusus Perintah Tugas (!tugas)
			if taskManager != nil && (strings.HasPrefix(lowerMsg, "!tugas") || strings.HasPrefix(lowerMsg, "/tugas") ||
				strings.HasPrefix(lowerMsg, "#tugas") || (!v.Info.IsGroup && strings.HasPrefix(lowerMsg, "tugas"))) {

				isAdmin := false
				if v.Info.IsGroup {
					isAdmin = isSenderGroupAdmin(context.Background(), client, v.Info.Chat, v.Info.Sender)
				} else {
					isAdmin = true // Di DM setiap orang adalah admin catatan miliknya sendiri
				}

				tugasReply := taskManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, jadwalData, time.Now())

				// 1. Berikan reaksi emoji pada pesan tugas
				reactionMsg := client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "📝")
				_, _ = client.SendMessage(context.Background(), v.Info.Chat, reactionMsg)

				// 2. Simulasi "sedang mengetik..." selama 600ms
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
				time.Sleep(600 * time.Millisecond)
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

				// 3. Kirim balasan tugas
				_, err := client.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
					Conversation: proto.String(tugasReply),
				})
				if err != nil {
					fmt.Printf("Gagal mengirim balasan tugas: %v\n", err)
				} else {
					fmt.Printf("Sukses membalas perintah tugas ke %s\n", v.Info.Chat.User)
				}
				return
			}

			// Handler Khusus Perintah Jadwal Pengganti / Override (!pindah, !kosong, !kuliahganti, !jadwalganti, !batalganti)
			isOverrideCmd := false
			for _, prefix := range []string{
				"!pindah", "/pindah", "#pindah", "!ganti", "/ganti", "#ganti",
				"!kosong", "/kosong", "#kosong", "!libur", "/libur", "#libur",
				"!kuliahganti", "/kuliahganti", "#kuliahganti", "!tambahkelas", "/tambahkelas",
				"!jadwalganti", "/jadwalganti", "#jadwalganti", "!overrides",
				"!batalganti", "/batalganti", "#batalganti",
			} {
				if strings.HasPrefix(lowerMsg, prefix) {
					isOverrideCmd = true
					break
				}
			}
			if !v.Info.IsGroup && !isOverrideCmd {
				for _, prefix := range []string{"pindah", "ganti", "kosong", "libur", "kuliahganti", "tambahkelas", "jadwalganti", "batalganti"} {
					if strings.HasPrefix(lowerMsg, prefix) {
						isOverrideCmd = true
						break
					}
				}
			}

			if overrideManager != nil && isOverrideCmd {
				isAdmin := false
				if v.Info.IsGroup {
					isAdmin = isSenderGroupAdmin(context.Background(), client, v.Info.Chat, v.Info.Sender)
				} else {
					isAdmin = true
				}

				overrideReply := overrideManager.HandleCommand(v.Info.Chat.String(), v.Info.IsGroup, v.Info.Sender.String(), isAdmin, msgText, jadwalData, time.Now())

				// 1. Berikan reaksi emoji pada pesan override
				reactionMsg := client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "🔄")
				_, _ = client.SendMessage(context.Background(), v.Info.Chat, reactionMsg)

				// 2. Simulasi "sedang mengetik..." selama 600ms
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
				time.Sleep(600 * time.Millisecond)
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

				// 3. Kirim balasan override
				_, err := client.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
					Conversation: proto.String(overrideReply),
				})
				if err != nil {
					fmt.Printf("Gagal mengirim balasan override: %v\n", err)
				} else {
					fmt.Printf("Sukses membalas perintah override ke %s\n", v.Info.Chat.User)
				}
				return
			}

			// Proses pesan masuk dengan parser perintah jadwal (menerapkan aturan Hybrid & Override)
			replyText := jadwalData.ProcessMessage(msgText, v.Info.IsGroup, v.Info.Chat.String())

			// Jika pesan cocok dengan salah satu perintah, kirim pesan balasan
			if replyText != "" {
				// 1. Berikan reaksi emoji pada pesan yang dikirim pengguna
				reactionMsg := client.BuildReaction(v.Info.Chat, v.Info.Sender, v.Info.ID, "📅")
				_, _ = client.SendMessage(context.Background(), v.Info.Chat, reactionMsg)

				// 2. Simulasi "sedang mengetik..." selama 800ms
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresenceComposing, types.ChatPresenceMediaText)
				time.Sleep(800 * time.Millisecond)
				_ = client.SendChatPresence(context.Background(), v.Info.Chat, types.ChatPresencePaused, types.ChatPresenceMediaText)

				// 3. Kirim pesan balasan jadwal
				_, err := client.SendMessage(context.Background(), v.Info.Chat, &waE2E.Message{
					Conversation: proto.String(replyText),
				})
				if err != nil {
					fmt.Printf("Gagal mengirim pesan balasan: %v\n", err)
				} else {
					fmt.Printf("Sukses membalas perintah '%s' ke %s\n", msgText, v.Info.Chat.User)
				}
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

	// 8. Jalankan background scheduler pengingat pagi otomatis (06:30 WIB)
	reminderManager.StartScheduler(client, jadwalData, taskManager)

	// 9. Tahan program agar terus berjalan sampai ditekan Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nMemutuskan koneksi bot...")
	client.Disconnect()
}