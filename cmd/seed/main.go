package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bot-jadwal/internal/academic"
	"bot-jadwal/internal/config"
	"bot-jadwal/internal/database"
)

func main() {
	jsonPath := flag.String("file", "jadwal.json", "Path file JSON jadwal sumber seeder")
	dbPath := flag.String("db", "", "Path file SQLite database (default membaca dari config)")
	flag.Parse()

	cfg := config.LoadConfig()
	targetDB := cfg.AppDBPath
	if *dbPath != "" {
		targetDB = *dbPath
	}

	fmt.Println("🌱 [Seeder] Memulai penyemaian data akademik awal...")
	fmt.Printf("📁 Berkas sumber: %s\n", *jsonPath)
	fmt.Printf("🗄️  Target database: %s\n", targetDB)

	db, err := database.InitDB(targetDB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Gagal menginisialisasi database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Jika file tidak ditemukan di direktori kerja, coba cari di parent
	filePath := *jsonPath
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		altPath := filepath.Join("..", "..", filePath)
		if _, err := os.Stat(altPath); err == nil {
			filePath = altPath
		}
	}

	if err := academic.SeedFromJSON(ctx, db, filePath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Gagal menyemai data: %v\n", err)
		os.Exit(1)
	}

	repo := academic.NewRepository(db)
	classes, err := repo.GetClasses(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Gagal membaca data kelas terdaftar: %v\n", err)
	} else {
		fmt.Printf("✅ Berhasil menyemai %d kelas:\n", len(classes))
		for _, c := range classes {
			courses, _ := repo.GetCoursesByClassID(ctx, c.ID)
			fmt.Printf("   - %s (%s, Angkatan %d): %d mata kuliah\n", c.Code, c.StudyProgram, c.CohortYear, len(courses))
		}
	}

	fmt.Println("🎉 [Seeder Selesai] Data awal berhasil disemai ke skema baru!")
}
