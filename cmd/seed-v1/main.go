package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bot-jadwal/internal/database"
	"bot-jadwal/internal/seed"
)

func main() {
	manifestPath := flag.String("manifest", "migration_manifest.json", "Jalur berkas manifest migrasi JSON")
	targetDBPath := flag.String("db", filepath.Join("storage", "bot_v1.db"), "Jalur database target v1 SQLite")
	legacyDBPath := flag.String("legacy-db", filepath.Join("storage", "tugas.db"), "Jalur database lama SQLite (hanya-baca)")
	jsonOutput := flag.Bool("json", false, "Keluarkan laporan dalam format JSON murni")
	flag.Parse()

	// 1. Validasi keberadaan manifest
	if _, err := os.Stat(*manifestPath); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Berkas manifest tidak ditemukan: %s\n", *manifestPath)
		os.Exit(1)
	}

	manifest, err := seed.LoadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Gagal membaca manifest: %v\n", err)
		os.Exit(1)
	}

	manifestChecksum, err := seed.ComputeFileHash(*manifestPath)
	if err != nil {
		manifestChecksum = "unknown"
	}

	// 2. Validasi manifest sebelum menyentuh database
	if err := seed.ValidateManifest(manifest); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Validasi manifest ditolak:\n%v\n", err)
		os.Exit(1)
	}

	// 3. Inisialisasi koneksi database target
	targetDB, err := database.InitDB(*targetDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Gagal menginisialisasi database target v1 '%s': %v\n", *targetDBPath, err)
		os.Exit(1)
	}
	defer targetDB.Close()

	// 4. Jalankan proses seed pilot
	report, err := seed.SeedClasses(manifest, manifestChecksum, targetDB, *legacyDBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Proses seed pilot gagal: %v\n", err)
		os.Exit(1)
	}

	// 5. Tampilkan laporan
	if *jsonOutput {
		encoded, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(encoded))
		return
	}

	printHumanReport(report, *targetDBPath)
}

func printHumanReport(report *seed.SeedReport, dbPath string) {
	fmt.Println("================================================================================")
	fmt.Println("🌱 LAPORAN SEED KELAS PILOT v1 (Lapis L1)")
	fmt.Println("================================================================================")
	fmt.Printf("Database Target  : %s\n", dbPath)
	fmt.Printf("Checksum Manifest: %s\n", report.ManifestChecksum)
	fmt.Printf("Waktu Eksekusi   : %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Println("--------------------------------------------------------------------------------")

	for i, c := range report.Classes {
		fmt.Printf("\n🏫 [%d] KELAS PILOT: %s (Slug: %s)\n", i+1, c.ClassCode, c.Slug)
		fmt.Printf("   Sumber File    : %s\n", c.SourceFile)
		fmt.Printf("   Checksum Sumber: %s\n", c.SourceChecksum)
		fmt.Println("   📊 Rekonsiliasi Sumber vs Target:")
		fmt.Printf("      - Mata Kuliah        : Sumber = %-3d | Target Master = %-3d\n", c.SourceCounts.Courses, c.TargetCounts.Courses)
		fmt.Printf("      - Dosen Pengampu     : Sumber = %-3d | Target Master = %-3d\n", c.SourceCounts.Lecturers, c.TargetCounts.Lecturers)
		fmt.Printf("      - Course Offerings   : Target = %-3d\n", c.TargetCounts.CourseOfferings)
		fmt.Printf("      - Offering Lecturers : Target = %-3d\n", c.TargetCounts.OfferingLecturers)
		fmt.Printf("      - Pola Jadwal Reguler: Sumber = %-3d | Target Pola   = %-3d\n", c.SourceCounts.Schedules, c.TargetCounts.SchedulePatterns)
		fmt.Printf("      - Tugas Warisan      : Sumber = %-3d | Target Impor  = %-3d\n", c.SourceCounts.Tasks, c.TargetCounts.Tasks)
		fmt.Printf("      - Materi / Tautan    : Sumber = %-3d | Target Impor  = %-3d\n", c.SourceCounts.Links, c.TargetCounts.Materials)

		if len(c.FailedRows) > 0 {
			fmt.Printf("   ⚠️  Baris Gagal (%d baris):\n", len(c.FailedRows))
			for _, f := range c.FailedRows {
				fmt.Printf("      [Baris %d] Kolom '%s' (%s): %s\n", f.RowNumber, f.Field, f.ErrorCode, f.Message)
			}
		} else {
			fmt.Println("   ✅ Baris Gagal       : 0 (Semua jadwal valid)")
		}

		if len(c.ResolutionQueue) > 0 {
			fmt.Printf("   📋 Antrean Resolusi Pending (%d item):\n", len(c.ResolutionQueue))
			for _, q := range c.ResolutionQueue {
				fmt.Printf("      - [%s #%s] '%s' -> %s (Status: %s)\n", q.SourceType, q.SourceID, q.Subject, q.Reason, q.Status)
			}
		} else {
			fmt.Println("   ✅ Antrean Resolusi  : 0 (Tidak ada data ambigu)")
		}
	}

	fmt.Println("\n================================================================================")
	fmt.Println("🎉 Seed pilot berhasil diselesaikan dengan aman dan idempoten!")
	fmt.Println("================================================================================")
}
