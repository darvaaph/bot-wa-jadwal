package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "modernc.org/sqlite"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func main() {
	// 1. Muat data jadwal dari file JSON
	jadwalData, err := LoadJadwal("jadwal.json")
	if err != nil {
		fmt.Printf("Peringatan: %v\n", err)
		fmt.Println("Pastikan file jadwal.json tersedia di direktori yang sama.")
		return
	}
	fmt.Printf("Berhasil memuat data jadwal: %s (%d jadwal mata kuliah)\n", jadwalData.Kampus, len(jadwalData.Jadwal))

	// 2. Setup Database Log & SQLite (Session Storage)
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	
	// Gunakan driver "sqlite" (pure Go tanpa CGO/GCC)
	container, err := sqlstore.New(context.Background(), "sqlite", "file:sesi_bot.db?_pragma=foreign_keys(1)", dbLog)
	if err != nil {
		panic(err)
	}

	// 3. Ambil sesi dari database
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		panic(err)
	}

	// 4. Setup Klien WhatsApp
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// 5. Daftarkan Event Handler untuk memproses pesan masuk
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

			lowerText := strings.ToLower(msgText)
			var replyText string

			// Routing perintah: mendukung awalan "!" maupun "/"
			if strings.HasPrefix(lowerText, "!menu") || strings.HasPrefix(lowerText, "/menu") ||
				strings.HasPrefix(lowerText, "!help") || strings.HasPrefix(lowerText, "/help") {
				replyText = jadwalData.GetMenu()
			} else if strings.HasPrefix(lowerText, "!jadwal") || strings.HasPrefix(lowerText, "/jadwal") {
				parts := strings.SplitN(msgText, " ", 2)
				arg := ""
				if len(parts) > 1 {
					arg = parts[1]
				}
				replyText = jadwalData.GetByHari(arg)
			} else if strings.HasPrefix(lowerText, "!dosen") || strings.HasPrefix(lowerText, "/dosen") {
				parts := strings.SplitN(msgText, " ", 2)
				arg := ""
				if len(parts) > 1 {
					arg = parts[1]
				}
				replyText = jadwalData.SearchDosen(arg)
			} else if strings.HasPrefix(lowerText, "!ruang") || strings.HasPrefix(lowerText, "/ruang") {
				parts := strings.SplitN(msgText, " ", 2)
				arg := ""
				if len(parts) > 1 {
					arg = parts[1]
				}
				replyText = jadwalData.SearchRuangan(arg)
			} else if strings.HasPrefix(lowerText, "!cari") || strings.HasPrefix(lowerText, "/cari") {
				parts := strings.SplitN(msgText, " ", 2)
				arg := ""
				if len(parts) > 1 {
					arg = parts[1]
				}
				replyText = jadwalData.SearchGlobal(arg)
			}

			// Jika pesan cocok dengan salah satu perintah, kirim pesan balasan
			if replyText != "" {
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

	// 6. Logika Login & QR Code
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

	// 7. Tahan program agar terus berjalan sampai ditekan Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("\nMemutuskan koneksi bot...")
	client.Disconnect()
}