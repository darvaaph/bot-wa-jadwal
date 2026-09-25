package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_EnsureStorageAndMigrate(t *testing.T) {
	tempStorage := "test_storage_temp"
	defer os.RemoveAll(tempStorage)

	cfg := &Config{
		StorageDir: tempStorage,
	}

	err := cfg.EnsureStorageAndMigrate()
	if err != nil {
		t.Fatalf("EnsureStorageAndMigrate failed: %v", err)
	}

	info, err := os.Stat(tempStorage)
	if err != nil || !info.IsDir() {
		t.Fatalf("Folder %s harus terbuat", tempStorage)
	}

	dummyRoot := "dummy_test_migrate.db"
	err = os.WriteFile(dummyRoot, []byte("test data"), 0644)
	if err != nil {
		t.Fatalf("Gagal membuat dummy file: %v", err)
	}
	defer os.Remove(dummyRoot)

	destPath := filepath.Join(tempStorage, dummyRoot)
	err = moveFile(dummyRoot, destPath)
	if err != nil {
		t.Fatalf("moveFile failed: %v", err)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("File tujuan harus ada di storage: %s", destPath)
	}
	if _, err := os.Stat(dummyRoot); !os.IsNotExist(err) {
		t.Errorf("File sumber harus terhapus setelah migrasi: %s", dummyRoot)
	}
}

func TestLoadConfig_AuthSettings(t *testing.T) {
	t.Setenv("BOT_JADWAL_AUTH_HASH_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("BOT_JADWAL_SECURE_COOKIES", "true")

	cfg := LoadConfig()
	if cfg.AuthHashKey != "0123456789abcdef0123456789abcdef" {
		t.Fatal("auth hash key tidak dimuat dari environment")
	}
	if !cfg.SecureCookies {
		t.Fatal("secure cookies seharusnya aktif")
	}
}
