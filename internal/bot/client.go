package bot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

// BotClient membungkus klien whatsmeow dan database session storage
type BotClient struct {
	Client         *whatsmeow.Client
	container      *sqlstore.Container
	cancelWatchdog context.CancelFunc
}

// NewBotClient menginisialisasi store database sesi SQLite untuk WhatsApp
func NewBotClient(sessionDBPath string) (*BotClient, error) {
	if sessionDBPath != ":memory:" && !strings.HasPrefix(sessionDBPath, "file:") {
		dir := filepath.Dir(sessionDBPath)
		if dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0755)
		}
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", sessionDBPath)

	container, err := sqlstore.New(context.Background(), "sqlite", dsn, dbLog)
	if err != nil {
		return nil, fmt.Errorf("gagal menginisialisasi sqlstore sesi: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		_ = container.Close()
		return nil, fmt.Errorf("gagal mengambil perangkat dari database sesi: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	// Ketahanan Sambungan Internet (Auto-Reconnect Resilience)
	client.EnableAutoReconnect = true
	client.AutoReconnectHook = func(err error) bool {
		fmt.Printf("⚠️ [Auto-Reconnect] Sambungan putus (%v). Mencoba menyambung kembali...\n", err)
		return true
	}

	return &BotClient{
		Client:    client,
		container: container,
	}, nil
}

// Connect menghubungkan klien ke server WhatsApp atau meminta QR Code di terminal jika belum login
func (b *BotClient) Connect(ctx context.Context) error {
	if b.Client.Store.ID == nil {
		qrChan, _ := b.Client.GetQRChannel(ctx)
		err := b.Client.Connect()
		if err != nil {
			return fmt.Errorf("gagal connect saat meminta QR code: %w", err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				fmt.Println("Silakan scan QR Code di atas menggunakan WhatsApp!")
			} else {
				fmt.Println("Status Login:", evt.Event)
			}
		}
	} else {
		err := b.Client.Connect()
		if err != nil {
			return fmt.Errorf("gagal connect ke WhatsApp: %w", err)
		}
		fmt.Println("🟢 [Bot] Berhasil terhubung ke WhatsApp!")
	}

	// Jalankan Watchdog Supervisor di background goroutine
	watchdogCtx, cancel := context.WithCancel(context.Background())
	b.cancelWatchdog = cancel
	go b.runWatchdog(watchdogCtx)

	return nil
}

// runWatchdog memantau status koneksi WhatsApp secara berkala dan melakukan reconnect dengan exponential backoff
func (b *BotClient) runWatchdog(ctx context.Context) {
	backoff := 3 * time.Second
	maxBackoff := 30 * time.Second
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if b.Client.IsLoggedIn() && !b.Client.IsConnected() {
				fmt.Printf("⚠️ [Watchdog] Terdeteksi offline. Menjalankan auto-reconnect (jeda %v)...\n", backoff)
				time.Sleep(backoff)
				err := b.Client.Connect()
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
			} else if b.Client.IsConnected() {
				backoff = 3 * time.Second
			}
		}
	}
}

// Disconnect memutuskan koneksi WhatsApp dan menghentikan watchdog
func (b *BotClient) Disconnect() {
	if b.cancelWatchdog != nil {
		b.cancelWatchdog()
	}
	if b.Client != nil {
		b.Client.Disconnect()
	}
}

// Close menutup sqlstore database sesi
func (b *BotClient) Close() error {
	if b.container != nil {
		return b.container.Close()
	}
	return nil
}

// Status mengembalikan deskripsi status koneksi bot saat ini (untuk telemetri API)
func (b *BotClient) Status() string {
	if b.Client == nil {
		return "uninitialized"
	}
	if b.Client.IsConnected() {
		return "connected"
	}
	if b.Client.IsLoggedIn() {
		return "reconnecting"
	}
	return "waiting_qr"
}
