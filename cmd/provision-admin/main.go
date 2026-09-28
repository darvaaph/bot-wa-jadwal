package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"bot-jadwal/internal/auth"
	"bot-jadwal/internal/config"
	"bot-jadwal/internal/database"
)

func main() {
	identity := flag.String("identity", "", "Identitas login System Admin pertama")
	displayName := flag.String("display-name", "", "Nama tampilan System Admin pertama")
	databasePath := flag.String("database", "", "Path database aplikasi (default: konfigurasi STORAGE_DIR)")
	flag.Parse()

	cfg := config.LoadConfig()
	if *databasePath != "" {
		cfg.AppDBPath = *databasePath
	}
	password := os.Getenv("BOT_JADWAL_ADMIN_PASSWORD")
	if *identity == "" || *displayName == "" || password == "" {
		log.Fatal("identity, display-name, dan environment BOT_JADWAL_ADMIN_PASSWORD wajib diisi")
	}
	if cfg.AuthHashKey == "" {
		log.Fatal("environment BOT_JADWAL_AUTH_HASH_KEY wajib diisi (minimal 32 byte)")
	}

	db, err := database.InitDB(cfg.AppDBPath)
	if err != nil {
		log.Fatalf("gagal membuka database: %v", err)
	}
	defer db.Close()

	service, err := auth.NewService(db, auth.Config{HashKey: []byte(cfg.AuthHashKey)})
	if err != nil {
		log.Fatalf("konfigurasi autentikasi tidak valid: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	user, _, err := service.ProvisionInitialSystemAdmin(ctx, auth.ProvisionInput{
		IdentityKey: *identity,
		DisplayName: *displayName,
		Password:    password,
	})
	if errors.Is(err, auth.ErrAlreadyProvisioned) {
		log.Fatal("System Admin awal sudah pernah dibuat; gunakan alur undangan untuk admin tambahan")
	}
	if err != nil {
		log.Fatalf("provisioning gagal: %v", err)
	}

	fmt.Printf("System Admin awal berhasil dibuat (user_id=%d).\n", user.ID)
}
