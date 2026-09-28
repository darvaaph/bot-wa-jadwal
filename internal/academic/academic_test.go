package academic

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"bot-jadwal/internal/database"
)

func TestAcademicRepositoryAndSeeder(t *testing.T) {
	testDB := "test_academic.db"
	defer os.Remove(testDB)
	defer os.Remove(testDB + "-wal")
	defer os.Remove(testDB + "-shm")

	db, err := database.InitDB(testDB)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	jadwalPath := filepath.Join("..", "..", "jadwal.json")
	if _, err := os.Stat(jadwalPath); os.IsNotExist(err) {
		jadwalPath = "jadwal.json"
	}

	err = SeedFromJSON(ctx, db, jadwalPath)
	if err != nil {
		t.Fatalf("Gagal menjalankan SeedFromJSON: %v", err)
	}

	repo := NewRepository(db)

	classes, err := repo.GetClasses(ctx)
	if err != nil {
		t.Fatalf("Gagal menjalankan GetClasses: %v", err)
	}
	if len(classes) == 0 {
		t.Fatal("Daftar kelas kosong setelah seeder dijalankan")
	}

	firstClass := classes[0]
	t.Logf("Kelas berhasil disemai: ID=%d, Code=%s, StudyProgram=%s, Cohort=%d",
		firstClass.ID, firstClass.Code, firstClass.StudyProgram, firstClass.CohortYear)

	if firstClass.Code == "" {
		t.Error("Kode kelas tidak boleh kosong")
	}
	if firstClass.Status != StatusActive {
		t.Errorf("Status kelas diharapkan %s, didapat %s", StatusActive, firstClass.Status)
	}

	courses, err := repo.GetCoursesByClassID(ctx, firstClass.ID)
	if err != nil {
		t.Fatalf("Gagal menjalankan GetCoursesByClassID: %v", err)
	}
	if len(courses) == 0 {
		t.Fatalf("Mata kuliah untuk kelas ID %d tidak boleh kosong", firstClass.ID)
	}
	t.Logf("Berhasil mengambil %d mata kuliah untuk kelas %s", len(courses), firstClass.Code)

	// Seeding pada database yang sama harus idempoten.
	err = SeedFromJSON(ctx, db, jadwalPath)
	if err != nil {
		t.Fatalf("SeedFromJSON kedua kali gagal (uji idempotensi): %v", err)
	}

	classesAfter, err := repo.GetClasses(ctx)
	if err != nil {
		t.Fatalf("Gagal GetClasses setelah re-seeding: %v", err)
	}
	if len(classesAfter) != len(classes) {
		t.Errorf("Jumlah kelas berubah setelah re-seeding: sebelumnya %d, sekarang %d", len(classes), len(classesAfter))
	}
}

func TestRepositoryRejectsMalformedTimestamp(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "malformed_timestamp.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	result, err := db.Exec(`
		INSERT INTO classes (
			code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "TEST-CLASS", "test-class", "Test", 2026, "A", StatusActive, "invalid-time", "2026-09-24T00:00:00Z")
	if err != nil {
		t.Fatalf("Gagal menyiapkan data uji: %v", err)
	}
	classID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Gagal membaca ID kelas: %v", err)
	}

	class, err := NewRepository(db).GetClassByID(context.Background(), classID)
	if err == nil {
		t.Fatalf("Timestamp rusak harus menghasilkan error, mendapat kelas: %+v", class)
	}
}

func TestRepositoryUsesBoundClassCode(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "parameter_binding.db")
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Gagal inisialisasi database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	class, err := NewRepository(db).GetClassByCode(context.Background(), `' OR 1=1 --`)
	if err != nil {
		t.Fatalf("Query dengan input adversarial gagal: %v", err)
	}
	if class != nil {
		t.Fatalf("Input adversarial tidak boleh cocok dengan kelas: %+v", class)
	}
}

func TestSeedFromJSONHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := SeedFromJSON(ctx, &sql.DB{}, filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("Context yang dibatalkan harus menghentikan seeder")
	}
}
