package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_MemuatJalurDatabaseV1(t *testing.T) {
	cfg := LoadConfig()
	if cfg.AppDBPath == "" || cfg.V1DBPath == "" {
		t.Fatalf("AppDBPath dan V1DBPath harus terisi")
	}
	if cfg.AppDBPath == cfg.V1DBPath {
		t.Errorf("Database v1 harus terpisah dari database lama agar bot tetap berjalan")
	}
	if filepath.Base(cfg.V1DBPath) != "bot_v1.db" {
		t.Errorf("Database v1 harus bernama bot_v1.db, didapat %s", cfg.V1DBPath)
	}
}

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

	// Pastikan folder terbuat
	info, err := os.Stat(tempStorage)
	if err != nil || !info.IsDir() {
		t.Fatalf("Folder %s harus terbuat", tempStorage)
	}

	// Uji migrasi file tiruan
	dummyRoot := "dummy_test_migrate.db"
	err = os.WriteFile(dummyRoot, []byte("test data"), 0644)
	if err != nil {
		t.Fatalf("Gagal membuat dummy file: %v", err)
	}
	defer os.Remove(dummyRoot)

	// Ubah nama migration files sementara di fungsi jika perlu atau tes manual
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
