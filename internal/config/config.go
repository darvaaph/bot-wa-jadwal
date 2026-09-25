package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	APIPort       string
	StorageDir    string
	AppDBPath     string
	SessionDBPath string
	ReminderPath  string
	DataJadwalDir string
	DefaultJadwal string
	AuthHashKey   string
	SecureCookies bool
}

// LoadConfig mengembalikan konfigurasi default atau berdasarkan environment variable
func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "storage"
	}

	secureCookies, _ := strconv.ParseBool(os.Getenv("BOT_JADWAL_SECURE_COOKIES"))

	return &Config{
		APIPort:       port,
		StorageDir:    storageDir,
		AppDBPath:     filepath.Join(storageDir, "tugas.db"),
		SessionDBPath: filepath.Join(storageDir, "sesi_bot.db"),
		ReminderPath:  filepath.Join(storageDir, "reminder_groups.json"),
		DataJadwalDir: "data/jadwal",
		DefaultJadwal: "jadwal.json",
		AuthHashKey:   os.Getenv("BOT_JADWAL_AUTH_HASH_KEY"),
		SecureCookies: secureCookies,
	}
}

// EnsureStorageAndMigrate memindahkan file runtime lama beserta sidecar SQLite ke storage
// agar migrasi tidak meninggalkan WAL atau SHM yang terkunci.
func (c *Config) EnsureStorageAndMigrate() error {
	if err := os.MkdirAll(c.StorageDir, 0755); err != nil {
		return fmt.Errorf("gagal membuat direktori storage '%s': %w", c.StorageDir, err)
	}

	migrationFiles := []string{
		"tugas.db",
		"tugas.db-wal",
		"tugas.db-shm",
		"sesi_bot.db",
		"sesi_bot.db-wal",
		"sesi_bot.db-shm",
		"reminder_groups.json",
	}

	for _, fileName := range migrationFiles {
		srcPath := fileName
		destPath := filepath.Join(c.StorageDir, fileName)

		if srcInfo, err := os.Stat(srcPath); err == nil && !srcInfo.IsDir() {
			if _, destErr := os.Stat(destPath); os.IsNotExist(destErr) {
				fmt.Printf("📦 [Migrasi Storage] Memindahkan %s -> %s...\n", srcPath, destPath)
				if err := moveFile(srcPath, destPath); err != nil {
					fmt.Printf("⚠️ Gagal memindahkan %s ke storage: %v\n", srcPath, err)
				}
			}
		}
	}

	return nil
}

// moveFile memindahkan file dengan aman lintas volume/drive
func moveFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	sourceFile.Close()
	return os.Remove(src)
}
