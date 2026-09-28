package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bot-jadwal/internal/database"
	"bot-jadwal/internal/seed"

	"golang.org/x/crypto/bcrypt"
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

	// 4b. Seed akun demo KM & PJ untuk pengujian API & Postman
	if err := seedDemoUsers(targetDB); err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Gagal membuat akun demo pengurus: %v\n", err)
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

func seedDemoUsers(db *sql.DB) error {
	pwdHash, err := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 1. Insert or update KM demo user (+6281234567890)
	var kmUserID int64
	err = db.QueryRow(`
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES ('+6281234567890', 'Ketua Murid (Demo)', ?, 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET password_hash = excluded.password_hash, status = 'ACTIVE'
		RETURNING id;
	`, string(pwdHash)).Scan(&kmUserID)
	if err != nil {
		return err
	}

	// 2. Insert or update PJ demo user (+6281298765432)
	var pjUserID int64
	err = db.QueryRow(`
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES ('+6281298765432', 'PJ Mata Kuliah (Demo)', ?, 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET password_hash = excluded.password_hash, status = 'ACTIVE'
		RETURNING id;
	`, string(pwdHash)).Scan(&pjUserID)
	if err != nil {
		return err
	}

	// 3. Pastikan role assignment KM dan PJ ada
	var classID, semesterID int64
	err = db.QueryRow(`SELECT id FROM classes WHERE slug = 'd4-ti-2024-a' LIMIT 1;`).Scan(&classID)
	if err != nil {
		return nil
	}
	_ = db.QueryRow(`SELECT id FROM semesters WHERE class_id = ? AND status = 'ACTIVE' LIMIT 1;`, classID).Scan(&semesterID)

	var offeringID int64
	_ = db.QueryRow(`SELECT id FROM course_offerings WHERE semester_id = ? LIMIT 1;`, semesterID).Scan(&offeringID)

	_, _ = db.Exec(`
		INSERT OR IGNORE INTO role_assignments (user_id, role, scope_type, class_id, semester_id, status)
		VALUES (?, 'KM', 'CLASS', ?, ?, 'ACTIVE');
	`, kmUserID, classID, semesterID)

	if offeringID > 0 {
		_, _ = db.Exec(`
			INSERT OR IGNORE INTO role_assignments (user_id, role, scope_type, class_id, semester_id, course_offering_id, status)
			VALUES (?, 'PJ', 'COURSE_OFFERING', ?, ?, ?, 'ACTIVE');
		`, pjUserID, classID, semesterID, offeringID)
	}

	return nil
}

