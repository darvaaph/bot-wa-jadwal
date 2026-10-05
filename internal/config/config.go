package config

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	APIPort       string
	StorageDir    string
	AppDBPath     string
	V1DBPath      string
	SessionDBPath string
	ReminderPath  string
	DataJadwalDir string
	DefaultJadwal string
	// BE-013: konfigurasi security eksplisit.
	Env               string
	AuthHashKey       string
	SecureCookies     bool
	AllowedOrigins    []string
	TrustedProxyCIDRs []string
	PublicBaseURL     string
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

	env := strings.ToLower(strings.TrimSpace(os.Getenv("BOT_JADWAL_ENV")))
	if env == "" {
		env = "development"
	}

	return &Config{
		APIPort:           port,
		StorageDir:        storageDir,
		AppDBPath:         filepath.Join(storageDir, "tugas.db"),
		V1DBPath:          filepath.Join(storageDir, "bot_v1.db"),
		SessionDBPath:     filepath.Join(storageDir, "sesi_bot.db"),
		ReminderPath:      filepath.Join(storageDir, "reminder_groups.json"),
		DataJadwalDir:     "data/jadwal",
		DefaultJadwal:     filepath.Join("data", "jadwal.json"),
		Env:               env,
		AuthHashKey:       os.Getenv("BOT_JADWAL_AUTH_HASH_KEY"),
		SecureCookies:     secureCookies,
		AllowedOrigins:    parseCSVList(os.Getenv("BOT_JADWAL_ALLOWED_ORIGINS")),
		TrustedProxyCIDRs: parseCSVList(os.Getenv("BOT_JADWAL_TRUSTED_PROXY_CIDRS")),
		PublicBaseURL:     strings.TrimSpace(os.Getenv("BOT_JADWAL_PUBLIC_BASE_URL")),
	}
}

// parseCSVList memecah daftar koma menjadi slice bersih tanpa nilai kosong.
func parseCSVList(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// IsProduction melaporkan apakah environment adalah production eksplisit.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// Validate memeriksa konfigurasi security sebelum server menerima traffic.
// Production gagal bila hash key kosong/pendek, secure cookie nonaktif,
// allowed origins kosong/malformed, atau proxy CIDR malformed.
// Error menyebut nama konfigurasi tanpa mencetak secret.
func (c *Config) Validate() error {
	switch c.Env {
	case "development", "test", "production":
	default:
		return fmt.Errorf("BOT_JADWAL_ENV tidak valid (development, test, atau production)")
	}
	for _, cidr := range c.TrustedProxyCIDRs {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(cidr)); err != nil {
			return fmt.Errorf("BOT_JADWAL_TRUSTED_PROXY_CIDRS memuat CIDR tidak valid")
		}
	}
	for _, origin := range c.AllowedOrigins {
		if err := ValidateOrigin(origin); err != nil {
			return fmt.Errorf("BOT_JADWAL_ALLOWED_ORIGINS memuat origin tidak valid: %w", err)
		}
	}
	if !c.IsProduction() {
		return nil
	}
	if len([]byte(c.AuthHashKey)) < 32 {
		return fmt.Errorf("BOT_JADWAL_AUTH_HASH_KEY wajib minimal 32 byte pada production")
	}
	if !c.SecureCookies {
		return fmt.Errorf("BOT_JADWAL_SECURE_COOKIES wajib true pada production")
	}
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("BOT_JADWAL_ALLOWED_ORIGINS wajib eksplisit pada production")
	}
	if c.PublicBaseURL != "" {
		u, err := url.Parse(c.PublicBaseURL)
		if err != nil || !strings.EqualFold(u.Scheme, "https") || u.Host == "" {
			return fmt.Errorf("BOT_JADWAL_PUBLIC_BASE_URL wajib HTTPS pada production")
		}
	}
	return nil
}

// ValidateOrigin memastikan origin berbentuk scheme://host[:port] eksplisit
// tanpa wildcard. Perbandingan CORS memakai exact match atas ketiganya.
func ValidateOrigin(origin string) error {
	trimmed := strings.TrimSpace(origin)
	if trimmed == "" || strings.Contains(trimmed, "*") {
		return fmt.Errorf("origin tidak boleh kosong atau memakai wildcard")
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return fmt.Errorf("format origin tidak valid")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme origin harus http atau https")
	}
	if u.Path != "" && u.Path != "/" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("origin tidak boleh memuat path, query, atau fragment")
	}
	return nil
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
