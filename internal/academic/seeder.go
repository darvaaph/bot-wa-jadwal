package academic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxSeedFileSize = 10 << 20

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

func readSeedFile(ctx context.Context, filePath string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(contextReader{ctx: ctx, reader: file}, maxSeedFileSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSeedFileSize {
		return nil, fmt.Errorf("ukuran file melebihi batas %d byte", maxSeedFileSize)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

type RawJadwalJSON struct {
	Kampus     string            `json:"kampus"`
	Dosen      map[string]string `json:"dosen"`
	MataKuliah map[string]string `json:"mata_kuliah"`
	Jadwal     []RawScheduleItem `json:"jadwal"`
}

type RawScheduleItem struct {
	Hari         string `json:"hari"`
	Jam          string `json:"jam"`
	KodeMatkul   string `json:"kode_matkul"`
	NamaMatkul   string `json:"nama_matkul"`
	InisialDosen string `json:"inisial_dosen"`
	Dosen        string `json:"dosen"`
	Ruang        string `json:"ruang"`
}

// SeedFromJSON membaca file JSON jadwal (seperti jadwal.json) dan menyemai data awal ke tabel
// classes, class_settings, semesters, courses, dan course_offerings.
func SeedFromJSON(ctx context.Context, db *sql.DB, filePath string) error {
	dataBytes, err := readSeedFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("gagal membaca file seeder %s: %w", filePath, err)
	}

	var rawData RawJadwalJSON
	if err := json.Unmarshal(dataBytes, &rawData); err != nil {
		return fmt.Errorf("gagal mengurai JSON seeder %s: %w", filePath, err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi seeder: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	classCode := "D4-TI-3A"
	classSlug := "d4-ti-3a"
	studyProgram := "D4 Teknik Informatika"
	groupLabel := "3A"
	cohortYear := 2024

	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	if strings.HasPrefix(baseName, "D") {
		// Contoh: D4-TI-SMT3-A -> group_label: 3A, code: D4-TI-3A
		parts := strings.Split(baseName, "-")
		if len(parts) >= 4 {
			smtNum := strings.TrimPrefix(parts[2], "SMT")
			groupLabel = smtNum + parts[3]
			classCode = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], groupLabel)
			classSlug = strings.ToLower(classCode)
		}
	}

	nowISO := time.Now().UTC().Format(time.RFC3339Nano)

	var classID int64
	classQuery := `
		INSERT INTO classes (code, slug, study_program, cohort_year, group_label, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			slug = excluded.slug,
			study_program = excluded.study_program,
			updated_at = excluded.updated_at
		RETURNING id;
	`
	err = tx.QueryRowContext(ctx, classQuery, classCode, classSlug, studyProgram, cohortYear, groupLabel, nowISO, nowISO).Scan(&classID)
	if err != nil {
		return fmt.Errorf("gagal menyemai tabel classes (%s): %w", classCode, err)
	}

	settingsQuery := `
		INSERT INTO class_settings (class_id, timezone, portal_access_mode, replacement_reminder_minutes, created_at, updated_at)
		VALUES (?, 'Asia/Jakarta', 'LINK', 60, ?, ?)
		ON CONFLICT(class_id) DO NOTHING;
	`
	if _, err := tx.ExecContext(ctx, settingsQuery, classID, nowISO, nowISO); err != nil {
		return fmt.Errorf("gagal menyemai class_settings: %w", err)
	}

	var semesterID int64
	academicYear := "2026/2027"
	term := "GANJIL"
	startsOn := "2026-09-01"
	endsOn := "2027-01-31"

	semesterQuery := `
		INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status, published_at, activated_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'ACTIVE', ?, ?, ?, ?)
		ON CONFLICT(class_id, academic_year, term) DO UPDATE SET
			status = 'ACTIVE',
			updated_at = excluded.updated_at
		RETURNING id;
	`
	err = tx.QueryRowContext(ctx, semesterQuery, classID, academicYear, term, startsOn, endsOn, nowISO, nowISO, nowISO, nowISO).Scan(&semesterID)
	if err != nil {
		return fmt.Errorf("gagal menyemai tabel semesters: %w", err)
	}

	courseIDMap := make(map[string]int64)
	courseStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO courses (code, name, status, created_at, updated_at)
		VALUES (?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(code) DO UPDATE SET
			name = excluded.name,
			updated_at = excluded.updated_at
		RETURNING id;
	`)
	if err != nil {
		return fmt.Errorf("gagal mempersiapkan query insert courses: %w", err)
	}
	defer courseStmt.Close()

	for cCode, cName := range rawData.MataKuliah {
		var cID int64
		err := courseStmt.QueryRowContext(ctx, cCode, cName, nowISO, nowISO).Scan(&cID)
		if err != nil {
			return fmt.Errorf("gagal menyemai course %s: %w", cCode, err)
		}
		courseIDMap[cCode] = cID
	}

	offeringStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'ACTIVE', ?, ?)
		ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET
			display_name = excluded.display_name,
			updated_at = excluded.updated_at;
	`)
	if err != nil {
		return fmt.Errorf("gagal mempersiapkan query insert course_offerings: %w", err)
	}
	defer offeringStmt.Close()

	for _, item := range rawData.Jadwal {
		cID, exists := courseIDMap[item.KodeMatkul]
		if !exists {
			// Jadwal boleh memuat mata kuliah yang belum ada di master.
			var newCID int64
			cleanName := item.NamaMatkul
			for _, suffix := range []string{" (Praktikum)", " (Teori)", " (Praktik)"} {
				cleanName = strings.ReplaceAll(cleanName, suffix, "")
			}
			err := courseStmt.QueryRowContext(ctx, item.KodeMatkul, cleanName, nowISO, nowISO).Scan(&newCID)
			if err != nil {
				return fmt.Errorf("gagal membuat master course on-the-fly untuk %s: %w", item.KodeMatkul, err)
			}
			cID = newCID
			courseIDMap[item.KodeMatkul] = cID
		}

		activityType := "KULIAH"
		lowerName := strings.ToLower(item.NamaMatkul)
		if strings.Contains(lowerName, "praktikum") || strings.Contains(lowerName, "praktik") {
			activityType = "PRAKTIKUM"
		} else if strings.Contains(lowerName, "teori") {
			activityType = "TEORI"
		}

		_, err = offeringStmt.ExecContext(ctx, semesterID, cID, activityType, item.NamaMatkul, nowISO, nowISO)
		if err != nil {
			return fmt.Errorf("gagal menyemai course_offering (%s - %s): %w", item.KodeMatkul, activityType, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit seeder: %w", err)
	}

	return nil
}
