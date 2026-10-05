package academic

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ImportError merepresentasikan kesalahan atau peringatan pada proses impor kurikulum.
type ImportError struct {
	Location string `json:"location"`
	Field    string `json:"field"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// ImportJSON memvalidasi dan menerapkan berkas JSON jadwal lama ke dalam semester DRAFT.
func (s *SemesterService) ImportJSON(ctx context.Context, actor Actor, classID int64, semesterID *int64, academicYear, term, startsOn, endsOn string, raw json.RawMessage, userID int64) (int64, []ImportError, error) {
	type rawItem struct {
		Hari         string `json:"hari"`
		Jam          string `json:"jam"`
		KodeMatkul   string `json:"kode_matkul"`
		NamaMatkul   string `json:"nama_matkul"`
		InisialDosen string `json:"inisial_dosen"`
		Dosen        string `json:"dosen"`
		Ruang        string `json:"ruang"`
	}
	type rawDoc struct {
		Kampus     string            `json:"kampus"`
		Dosen      map[string]string `json:"dosen"`
		MataKuliah map[string]string `json:"mata_kuliah"`
		Jadwal     []rawItem         `json:"jadwal"`
	}
	var doc rawDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return 0, nil, ErrInvalidInput
	}
	errs := []ImportError{}
	addErr := func(loc, field, code, msg, severity string) {
		errs = append(errs, ImportError{Location: loc, Field: field, Code: code, Message: msg, Severity: severity})
	}
	if len(doc.MataKuliah) == 0 {
		addErr("mata_kuliah", "", "EMPTY", "daftar mata_kuliah kosong", "ERROR")
	}
	if len(doc.Jadwal) == 0 {
		addErr("jadwal", "", "EMPTY", "daftar jadwal kosong", "ERROR")
	}
	validDays := map[string]int{"senin": 1, "selasa": 2, "rabu": 3, "kamis": 4, "jumat": 5, "sabtu": 6, "minggu": 7}
	seen := map[string]bool{}
	for i, item := range doc.Jadwal {
		loc := fmt.Sprintf("jadwal[%d]", i)
		if strings.TrimSpace(item.KodeMatkul) == "" {
			addErr(loc, "kode_matkul", "REQUIRED", "kode_matkul wajib diisi", "ERROR")
		}
		if strings.TrimSpace(item.NamaMatkul) == "" {
			addErr(loc, "nama_matkul", "REQUIRED", "nama_matkul wajib diisi", "ERROR")
		}
		dow, ok := validDays[strings.ToLower(strings.TrimSpace(item.Hari))]
		if !ok {
			addErr(loc, "hari", "INVALID", "hari tidak dikenal", "ERROR")
		}
		parts := strings.Split(strings.TrimSpace(item.Jam), "-")
		if len(parts) != 2 {
			addErr(loc, "jam", "INVALID", "format jam harus HH:MM - HH:MM", "ERROR")
		} else {
			start := strings.TrimSpace(parts[0])
			end := strings.TrimSpace(parts[1])
			if len(start) != 5 || len(end) != 5 || start >= end {
				addErr(loc, "jam", "INVALID", "rentang jam tidak valid", "ERROR")
			}
			_ = dow
		}
		key := strings.ToLower(strings.TrimSpace(item.Hari)) + "|" + strings.TrimSpace(item.Jam) + "|" + strings.TrimSpace(item.KodeMatkul)
		if seen[key] {
			addErr(loc, "", "DUPLICATE", "baris duplikat", "ERROR")
		}
		seen[key] = true
		if _, ok := doc.MataKuliah[strings.TrimSpace(item.KodeMatkul)]; !ok && strings.TrimSpace(item.KodeMatkul) != "" {
			addErr(loc, "kode_matkul", "UNKNOWN_COURSE", "kode tidak ada di master mata_kuliah (akan dibuat otomatis)", "WARNING")
		}
	}
	blockers := 0
	for _, e := range errs {
		if e.Severity == "ERROR" {
			blockers++
		}
	}
	sumData, _ := json.Marshal(map[string]any{
		"total_rows": len(doc.Jadwal), "valid_rows": len(doc.Jadwal) - blockers,
		"warning_rows": len(errs) - blockers, "error_rows": blockers, "applied_rows": 0,
	})
	batchID, batchErr := s.createImportBatch(ctx, classID, semesterID, academicYear, term, startsOn, endsOn, checksum(raw), string(sumData), userID, blockers > 0)
	if batchErr != nil {
		return 0, errs, batchErr
	}
	for _, e := range errs {
		_ = s.recordImportError(ctx, batchID, e)
	}
	if blockers > 0 {
		return batchID, errs, ErrInvalidInput
	}
	targetSem, err := s.resolveOrCreateDraft(ctx, actor, classID, semesterID, academicYear, term, startsOn, endsOn)
	if err != nil {
		return batchID, errs, err
	}
	applied, err := s.applyImport(ctx, classID, targetSem, doc)
	if err != nil {
		return batchID, errs, err
	}
	sumData, _ = json.Marshal(map[string]any{
		"total_rows": len(doc.Jadwal), "valid_rows": len(doc.Jadwal),
		"warning_rows": len(errs), "error_rows": 0, "applied_rows": applied, "semester_id": targetSem,
	})
	_, _ = s.db.ExecContext(ctx, `UPDATE import_batches SET status='APPLIED', semester_id=?, summary_json=? WHERE id=?`, targetSem, string(sumData), batchID)

	after := fmt.Sprintf(`{"applied_rows":%d,"batch_id":%d}`, applied, batchID)
	_ = WriteAuditLog(ctx, nil, actor, &classID, &targetSem, "IMPORT", "SEMESTER", &targetSem, nil, &after, "")
	return targetSem, errs, nil
}

func (s *SemesterService) createImportBatch(ctx context.Context, classID int64, semesterID *int64, year, term, starts, ends, check, summary string, userID int64, invalid bool) (int64, error) {
	semVal := semesterID
	if semVal == nil {
		var latest sql.NullInt64
		_ = s.db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id = ? ORDER BY id DESC LIMIT 1`, classID).Scan(&latest)
		if latest.Valid {
			v := latest.Int64
			semVal = &v
		} else {
			tx, err := s.db.BeginTx(ctx, nil)
			if err != nil {
				return 0, err
			}
			defer tx.Rollback()
			y, t, st, en := year, term, starts, ends
			if y == "" {
				y = "2026/2027"
			}
			if t == "" {
				t = "GANJIL"
			}
			if st == "" {
				st = "2026-09-01"
			}
			if en == "" {
				en = "2027-01-31"
			}
			var nid int64
			if err := tx.QueryRowContext(ctx, `
				INSERT INTO semesters (class_id, academic_year, term, starts_on, ends_on, status)
				VALUES (?, ?, ?, ?, ?, 'DRAFT') RETURNING id
			`, classID, y, t, st, en).Scan(&nid); err != nil {
				return 0, err
			}
			if err := tx.Commit(); err != nil {
				return 0, err
			}
			semVal = &nid
		}
	}
	status := "READY"
	if invalid {
		status = "INVALID"
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO import_batches
		(class_id, semester_id, source_type, source_checksum, status, created_by_user_id, summary_json)
		VALUES (?, ?, 'JSON', ?, ?, ?, ?) RETURNING id
	`, classID, *semVal, check, status, userID, summary).Scan(&id)
	return id, err
}

func (s *SemesterService) recordImportError(ctx context.Context, batchID int64, e ImportError) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO import_errors (batch_id, row_number, field, error_code, message, severity)
		VALUES (?, 0, ?, ?, ?, ?)
	`, batchID, e.Field, e.Code, e.Message, e.Severity)
	return err
}

func (s *SemesterService) resolveOrCreateDraft(ctx context.Context, actor Actor, classID int64, semesterID *int64, year, term, starts, ends string) (int64, error) {
	if semesterID != nil && *semesterID > 0 {
		var status string
		var c int64
		if err := s.db.QueryRowContext(ctx, `SELECT status, class_id FROM semesters WHERE id = ?`, *semesterID).Scan(&status, &c); err != nil {
			return 0, ErrNotFound
		}
		if c != classID || status == "ARCHIVED" {
			return 0, ErrInvalidInput
		}
		return *semesterID, nil
	}
	if year == "" || term == "" || starts == "" || ends == "" || starts >= ends {
		return 0, ErrInvalidInput
	}
	return s.CreateDraft(ctx, actor, classID, DraftInput{AcademicYear: year, Term: strings.ToUpper(term), StartsOn: starts, EndsOn: ends})
}

func (s *SemesterService) applyImport(ctx context.Context, classID, semesterID int64, doc any) (int, error) {
	type rawItem struct {
		Hari         string `json:"hari"`
		Jam          string `json:"jam"`
		KodeMatkul   string `json:"kode_matkul"`
		NamaMatkul   string `json:"nama_matkul"`
		InisialDosen string `json:"inisial_dosen"`
		Dosen        string `json:"dosen"`
		Ruang        string `json:"ruang"`
	}
	type rawDoc struct {
		Kampus     string            `json:"kampus"`
		Dosen      map[string]string `json:"dosen"`
		MataKuliah map[string]string `json:"mata_kuliah"`
		Jadwal     []rawItem         `json:"jadwal"`
	}
	b, _ := json.Marshal(doc)
	var d rawDoc
	if err := json.Unmarshal(b, &d); err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	courseIDs := map[string]int64{}
	for code, name := range d.MataKuliah {
		var cid int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
			ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id
		`, code, name, now).Scan(&cid)
		if err != nil {
			return 0, err
		}
		courseIDs[code] = cid
	}
	dayMap := map[string]int{"senin": 1, "selasa": 2, "rabu": 3, "kamis": 4, "jumat": 5, "sabtu": 6, "minggu": 7}
	applied := 0
	for _, item := range d.Jadwal {
		code := strings.TrimSpace(item.KodeMatkul)
		cid, ok := courseIDs[code]
		if !ok {
			name := strings.TrimSpace(item.NamaMatkul)
			for _, suf := range []string{" (Praktikum)", " (Teori)", " (Praktik)"} {
				name = strings.ReplaceAll(name, suf, "")
			}
			if name == "" {
				name = code
			}
			err := tx.QueryRowContext(ctx, `
				INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id
			`, code, name, now).Scan(&cid)
			if err != nil {
				return 0, err
			}
			courseIDs[code] = cid
		}
		lower := strings.ToLower(item.NamaMatkul)
		activity := "KULIAH"
		if strings.Contains(lower, "praktikum") || strings.Contains(lower, "praktik") {
			activity = "PRAKTIKUM"
		} else if strings.Contains(lower, "teori") {
			activity = "TEORI"
		}
		var offeringID int64
		err := tx.QueryRowContext(ctx, `
			INSERT INTO course_offerings (semester_id, course_id, activity_type, display_name, status)
			VALUES (?, ?, ?, ?, 'ACTIVE')
			ON CONFLICT(semester_id, course_id, activity_type) DO UPDATE SET display_name=excluded.display_name, updated_at=?
			RETURNING id
		`, semesterID, cid, activity, strings.TrimSpace(item.NamaMatkul), now).Scan(&offeringID)
		if err != nil {
			return 0, err
		}
		initial := strings.TrimSpace(item.InisialDosen)
		full := strings.TrimSpace(item.Dosen)
		if initial != "" || full != "" {
			if initial == "" {
				initial = full
			}
			if full == "" {
				if alt, ok := d.Dosen[initial]; ok && strings.TrimSpace(alt) != "" {
					full = alt
				} else {
					full = initial
				}
			}
			var lid int64
			err := tx.QueryRowContext(ctx, `
				INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET full_name=excluded.full_name, updated_at=? RETURNING id
			`, initial, full, now).Scan(&lid)
			if err != nil {
				return 0, err
			}
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO offering_lecturers (course_offering_id, lecturer_id, responsibility)
				VALUES (?, ?, 'PRIMARY') ON CONFLICT(course_offering_id, lecturer_id) DO UPDATE SET responsibility='PRIMARY', superseded_at=NULL
			`, offeringID, lid)
		}
		var roomArg any
		if rc := strings.TrimSpace(item.Ruang); rc != "" {
			var rid int64
			err := tx.QueryRowContext(ctx, `
				INSERT INTO rooms (code, name, status) VALUES (?, ?, 'ACTIVE')
				ON CONFLICT(code) DO UPDATE SET name=excluded.name, updated_at=? RETURNING id
			`, rc, rc, now).Scan(&rid)
			if err != nil {
				return 0, err
			}
			roomArg = rid
		}
		dow := dayMap[strings.ToLower(strings.TrimSpace(item.Hari))]
		parts := strings.Split(item.Jam, "-")
		start, end := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		var semStarts string
		_ = tx.QueryRowContext(ctx, `SELECT starts_on FROM semesters WHERE id = ?`, semesterID).Scan(&semStarts)
		if semStarts == "" {
			semStarts = "2026-09-01"
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO schedule_patterns
			(course_offering_id, room_id, day_of_week, start_time, end_time, effective_from, status)
			VALUES (?, ?, ?, ?, ?, ?, 'ACTIVE')
		`, offeringID, roomArg, dow, start, end, semStarts); err != nil {
			return 0, err
		}
		applied++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return applied, nil
}
