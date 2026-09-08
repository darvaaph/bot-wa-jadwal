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
