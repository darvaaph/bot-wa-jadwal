package backup

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/maintenance"
)

var ErrStalePreview = errors.New("pratinjau pemulihan sudah berubah; muat ulang sebelum melanjutkan")
var ErrCrossClass = errors.New("pemulihan menyentuh teaching event lintas kelas")

type RestorePreview struct {
	BackupID           int64          `json:"backup_id"`
	ClassID            int64          `json:"class_id"`
	SemesterID         *int64         `json:"semester_id,omitempty"`
	Checksum           string         `json:"checksum"`
	Token              string         `json:"preview_token"`
	SourceCounts       map[string]int `json:"source_counts"`
	CurrentCounts      map[string]int `json:"current_counts"`
	CrossClassEventIDs []int64        `json:"cross_class_event_ids"`
	CanRestore         bool           `json:"can_restore"`
}

func (s *Service) readScoped(ctx context.Context, id int64) (*Record, *dump, error) {
	rec, err := s.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	base, err := filepath.Abs(s.dir)
	if err != nil {
		return nil, nil, err
	}
	target, err := filepath.Abs(rec.ArtifactRef)
	if err != nil {
		return nil, nil, err
	}
	base, err = filepath.EvalSymlinks(base)
	if err != nil {
		return nil, nil, err
	}
	target, err = filepath.EvalSymlinks(target)
	if err != nil {
		return nil, nil, err
	}
	if target == base || !strings.HasPrefix(target, base+string(os.PathSeparator)) {
		return nil, nil, ErrMismatch
	}
	if filepath.Ext(target) != ".json" {
		return nil, nil, ErrMismatch
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, nil, err
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != rec.Checksum {
		return nil, nil, ErrMismatch
	}
	var d dump
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, nil, err
	}
	if d.Version != 2 || d.ClassID != rec.ClassID || (rec.SemesterID == nil) != (d.SemesterID == nil) {
		return nil, nil, ErrMismatch
	}
	if rec.SemesterID != nil && *rec.SemesterID != *d.SemesterID {
		return nil, nil, ErrMismatch
	}
	return rec, &d, nil
}

func sortedDump(d *dump) {
	groups := [][]map[string]any{d.Semesters, d.Offerings, d.OffLect, d.Patterns, d.Events, d.Parts, d.Confirms, d.Tasks, d.Reviews, d.Materials, d.Courses, d.Lecturers}
	for _, group := range groups {
		sort.Slice(group, func(i, j int) bool { return int64Val(group[i]["id"]) < int64Val(group[j]["id"]) })
	}
}

func counts(d *dump) map[string]int {
	return map[string]int{"semesters": len(d.Semesters), "offerings": len(d.Offerings), "offering_lecturers": len(d.OffLect),
		"patterns": len(d.Patterns), "events": len(d.Events), "participations": len(d.Parts), "confirmations": len(d.Confirms),
		"tasks": len(d.Tasks), "reviews": len(d.Reviews), "materials": len(d.Materials)}
}

func publishedChanged(before, source *dump) bool {
	filter := func(rows []map[string]any, field, value string) []map[string]any {
		out := []map[string]any{}
		for _, row := range rows {
			if strVal(row[field]) == value {
				out = append(out, row)
			}
		}
		sort.Slice(out, func(i, j int) bool { return int64Val(out[i]["id"]) < int64Val(out[j]["id"]) })
		return out
	}
	left, _ := json.Marshal([]any{filter(before.Patterns, "status", "ACTIVE"), filter(before.Events, "lifecycle_status", "PUBLISHED"), filter(before.Tasks, "publication_status", "PUBLISHED"), filter(before.Materials, "status", "ACTIVE")})
	right, _ := json.Marshal([]any{filter(source.Patterns, "status", "ACTIVE"), filter(source.Events, "lifecycle_status", "PUBLISHED"), filter(source.Tasks, "publication_status", "PUBLISHED"), filter(source.Materials, "status", "ACTIVE")})
	return string(left) != string(right)
}

func (s *Service) crossClassEvents(ctx context.Context, rec *Record, d *dump) ([]int64, error) {
	ids := map[int64]bool{}
	for _, pt := range d.Parts {
		var classID int64
		if err := s.db.QueryRowContext(ctx, `SELECT sem.class_id FROM course_offerings co JOIN semesters sem ON sem.id=co.semester_id WHERE co.id=?`, int64Val(pt["course_offering_id"])).Scan(&classID); err != nil {
			return nil, ErrMismatch
		}
		if classID != rec.ClassID {
			ids[int64Val(pt["teaching_event_id"])] = true
		}
	}
	// Restoring an owned event would also overwrite participation added by another
	// class after the backup, so inspect the live database in both directions.
	for _, event := range d.Events {
		var foreign bool
		err := s.db.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM teaching_event_offerings teo
			JOIN course_offerings co ON co.id=teo.course_offering_id
			JOIN semesters sem ON sem.id=co.semester_id
			WHERE teo.teaching_event_id=? AND sem.class_id<>?)`, int64Val(event["id"]), rec.ClassID).Scan(&foreign)
		if err != nil {
			return nil, err
		}
		if foreign {
			ids[int64Val(event["id"])] = true
		}
	}
	query := `SELECT DISTINCT te.id FROM teaching_events te
		JOIN teaching_event_offerings target ON target.teaching_event_id=te.id AND target.participation_role='PARTICIPANT'
		JOIN course_offerings co ON co.id=target.course_offering_id
		JOIN semesters sem ON sem.id=co.semester_id
		JOIN teaching_event_offerings own ON own.teaching_event_id=te.id AND own.participation_role='OWNER'
		JOIN course_offerings oco ON oco.id=own.course_offering_id
		JOIN semesters osem ON osem.id=oco.semester_id
		WHERE sem.class_id=? AND osem.class_id<>?`
	args := []any{rec.ClassID, rec.ClassID}
	if rec.SemesterID != nil {
		query += ` AND sem.id=?`
		args = append(args, *rec.SemesterID)
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out, nil
}

func (s *Service) Preview(ctx context.Context, backupID int64) (*RestorePreview, error) {
	rec, d, err := s.readScoped(ctx, backupID)
	if err != nil {
		return nil, err
	}
	current, err := s.buildDump(ctx, rec.ClassID, rec.SemesterID)
	if err != nil {
		return nil, err
	}
	sortedDump(current)
	live, err := json.Marshal(current)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(append([]byte(rec.Checksum+":"), live...))
	blockers, err := s.crossClassEvents(ctx, rec, d)
	if err != nil {
		return nil, err
	}
	if rec.SemesterID != nil {
		for _, sem := range d.Semesters {
			if strVal(sem["status"]) != "ACTIVE" {
				continue
			}
			var otherActive int64
			err := s.db.QueryRowContext(ctx, `SELECT id FROM semesters WHERE class_id=? AND status='ACTIVE' AND id<>?`, rec.ClassID, *rec.SemesterID).Scan(&otherActive)
			if err == nil {
				return nil, fmt.Errorf("semester #%d sedang aktif; pemulihan semester lain tidak boleh menggantinya", otherActive)
			}
			if err != sql.ErrNoRows {
				return nil, err
			}
		}
	}
	return &RestorePreview{BackupID: backupID, ClassID: rec.ClassID, SemesterID: rec.SemesterID, Checksum: rec.Checksum,
		Token: hex.EncodeToString(hash[:]), SourceCounts: counts(d), CurrentCounts: counts(current),
		CrossClassEventIDs: blockers, CanRestore: len(blockers) == 0}, nil
}

func mapIDs(rows []map[string]any) map[int64]bool {
	out := make(map[int64]bool, len(rows))
	for _, row := range rows {
		out[int64Val(row["id"])] = true
	}
	return out
}

func normalizeValue(col string, value any) any {
	if value == nil {
		return nil
	}
	if col == "effective_from" || col == "effective_until" || col == "origin_occurrence_date" || col == "starts_on" || col == "ends_on" {
		v := strVal(value)
		if len(v) >= 10 {
			return v[:10]
		}
	}
	return value
}

func upsertRows(ctx context.Context, tx *sql.Tx, table string, rows []map[string]any, appendOnly bool) error {
	for _, row := range rows {
		cols := make([]string, 0, len(row))
		for key := range row {
			cols = append(cols, key)
		}
		sort.Strings(cols)
		values := make([]any, 0, len(cols))
		marks := make([]string, 0, len(cols))
		updates := make([]string, 0, len(cols))
		for _, col := range cols {
			values = append(values, normalizeValue(col, row[col]))
			marks = append(marks, "?")
			if col != "id" {
				updates = append(updates, col+"=excluded."+col)
			}
		}
		query := "INSERT INTO " + table + " (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(marks, ",") + ")"
		if appendOnly {
			query += " ON CONFLICT DO NOTHING"
		} else {
			query += " ON CONFLICT(id) DO UPDATE SET " + strings.Join(updates, ",")
		}
		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("restore %s id=%d: %w", table, int64Val(row["id"]), err)
		}
	}
	return nil
}

func verifySnapshotCounts(ctx context.Context, tx *sql.Tx, d *dump) error {
	for _, group := range []struct {
		table string
		rows  []map[string]any
	}{
		{"semesters", d.Semesters}, {"course_offerings", d.Offerings}, {"offering_lecturers", d.OffLect},
		{"schedule_patterns", d.Patterns}, {"teaching_events", d.Events}, {"teaching_event_offerings", d.Parts},
		{"room_confirmations", d.Confirms}, {"tasks", d.Tasks}, {"task_reviews", d.Reviews}, {"materials", d.Materials},
	} {
		if len(group.rows) == 0 {
			continue
		}
		marks := make([]string, len(group.rows))
		args := make([]any, len(group.rows))
		for i, row := range group.rows {
			marks[i] = "?"
			args[i] = int64Val(row["id"])
		}
		var count int
		query := "SELECT COUNT(*) FROM " + group.table + " WHERE id IN (" + strings.Join(marks, ",") + ")"
		if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
			return err
		}
		if count != len(group.rows) {
			return fmt.Errorf("jumlah %s setelah restore %d, paket %d", group.table, count, len(group.rows))
		}
	}
	return nil
}

func (s *Service) Execute(ctx context.Context, backupID, userID int64, reason, expectedToken string) (int64, error) {
	if userID <= 0 || strings.TrimSpace(reason) == "" || expectedToken == "" {
		return 0, ErrInvalidInput
	}
	release := maintenance.BeginRestore()
	defer release()
	preview, err := s.Preview(ctx, backupID)
	if err != nil {
		return 0, err
	}
	if !preview.CanRestore {
		return 0, ErrCrossClass
	}
	if preview.Token != expectedToken {
		return 0, ErrStalePreview
	}
	rec, d, err := s.readScoped(ctx, backupID)
	if err != nil {
		return 0, err
	}
	before, err := s.buildDump(ctx, rec.ClassID, rec.SemesterID)
	if err != nil {
		return 0, err
	}
	prePoint, err := s.Create(ctx, rec.ClassID, rec.SemesterID, userID, "Titik sebelum restore #"+fmt.Sprint(backupID))
	if err != nil {
		return 0, fmt.Errorf("gagal membuat titik pemulihan: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return prePoint.ID, err
	}
	defer tx.Rollback()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	// Hide rows that were created after the selected backup. Audit and reviews remain append-only.
	for _, pair := range []struct {
		table, sql      string
		current, source []map[string]any
	}{
		{"offering_lecturers", `UPDATE offering_lecturers SET superseded_at=? WHERE id=?`, before.OffLect, d.OffLect},
		{"materials", `UPDATE materials SET deleted_at=?, deleted_by_user_id=?, status='ARCHIVED' WHERE id=?`, before.Materials, d.Materials},
		{"room_confirmations", `UPDATE room_confirmations SET superseded_at=? WHERE id=?`, before.Confirms, d.Confirms},
		{"tasks", `UPDATE tasks SET deleted_at=?, deleted_by_user_id=? WHERE id=?`, before.Tasks, d.Tasks},
		{"schedule_patterns", `UPDATE schedule_patterns SET status='DELETED' WHERE id=?`, before.Patterns, d.Patterns},
		{"course_offerings", `UPDATE course_offerings SET status='ARCHIVED' WHERE id=?`, before.Offerings, d.Offerings},
		{"semesters", `UPDATE semesters SET status='ARCHIVED', archived_at=?, published_at=NULL WHERE id=?`, before.Semesters, d.Semesters},
	} {
		wanted := mapIDs(pair.source)
		for _, row := range pair.current {
			id := int64Val(row["id"])
			if wanted[id] {
				continue
			}
			var args []any
			switch pair.table {
			case "materials", "tasks":
				args = []any{now, userID, id}
			case "semesters":
				args = []any{now, id}
			case "room_confirmations", "offering_lecturers":
				args = []any{now, id}
			default:
				args = []any{id}
			}
			if _, err := tx.ExecContext(ctx, pair.sql, args...); err != nil {
				return prePoint.ID, err
			}
		}
	}
	for _, row := range before.Events {
		if mapIDs(d.Events)[int64Val(row["id"])] {
			continue
		}
		if strVal(row["lifecycle_status"]) == "PUBLISHED" {
			if _, err := tx.ExecContext(ctx, `UPDATE teaching_events SET lifecycle_status='REVOKED', revoked_by_user_id=?, revoked_at=?, revocation_reason='Pemulihan data akademik' WHERE id=?`, userID, now, int64Val(row["id"])); err != nil {
				return prePoint.ID, err
			}
		}
	}
	wantedParts := mapIDs(d.Parts)
	for _, row := range before.Parts {
		id := int64Val(row["id"])
		if wantedParts[id] || strVal(row["participation_role"]) != "PARTICIPANT" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE teaching_event_offerings SET participation_status='REMOVED', responded_by_user_id=?, responded_at=? WHERE id=?`, userID, now, id); err != nil {
			return prePoint.ID, err
		}
	}
	if rec.SemesterID == nil {
		if _, err := tx.ExecContext(ctx, `UPDATE semesters SET status='ARCHIVED', archived_at=COALESCE(archived_at,?) WHERE class_id=? AND status='ACTIVE'`, now, rec.ClassID); err != nil {
			return prePoint.ID, err
		}
	}
	for _, group := range []struct {
		table      string
		rows       []map[string]any
		appendOnly bool
	}{
		{"semesters", d.Semesters, false}, {"course_offerings", d.Offerings, false},
		{"offering_lecturers", d.OffLect, false}, {"schedule_patterns", d.Patterns, false},
		{"teaching_events", d.Events, false}, {"teaching_event_offerings", d.Parts, false},
		{"room_confirmations", d.Confirms, false}, {"tasks", d.Tasks, false},
		{"task_reviews", d.Reviews, true}, {"materials", d.Materials, false},
	} {
		if err := upsertRows(ctx, tx, group.table, group.rows, group.appendOnly); err != nil {
			return prePoint.ID, err
		}
	}
	// Prevent stale queued messages from describing data superseded by the restore.
	if rec.SemesterID == nil {
		if _, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status='CANCELLED', updated_at=? WHERE class_id=? AND status='PENDING'`, now, rec.ClassID); err != nil {
			return prePoint.ID, err
		}
	} else {
		for _, group := range []struct {
			kind string
			rows []map[string]any
		}{{"TASK", before.Tasks}, {"TEACHING_EVENT", before.Events}, {"SCHEDULE_PATTERN", before.Patterns}} {
			for _, row := range group.rows {
				if _, err := tx.ExecContext(ctx, `UPDATE notification_messages SET status='CANCELLED', updated_at=? WHERE class_id=? AND entity_type=? AND entity_id=? AND status='PENDING'`, now, rec.ClassID, group.kind, int64Val(row["id"])); err != nil {
					return prePoint.ID, err
				}
			}
		}
	}
	if publishedChanged(before, d) {
		var channelID sql.NullInt64
		err := tx.QueryRowContext(ctx, `SELECT id FROM whatsapp_channels WHERE class_id=? AND status='ACTIVE' ORDER BY id DESC LIMIT 1`, rec.ClassID).Scan(&channelID)
		if err != nil && err != sql.ErrNoRows {
			return prePoint.ID, err
		}
		payload, _ := json.Marshal(map[string]any{"backup_id": backupID, "semester_id": rec.SemesterID, "reason": strings.TrimSpace(reason)})
		_, err = tx.ExecContext(ctx, `INSERT INTO notification_messages(class_id,whatsapp_channel_id,event_type,entity_type,entity_id,idempotency_key,payload_json,status,scheduled_at,triggered_by_user_id)
			VALUES(?,?, 'ACADEMIC_RESTORE_CORRECTION','BACKUP_RECORD',?,?,?,'PENDING',?,?)`, rec.ClassID, channelID, backupID, fmt.Sprintf("restore-correction-%d-%d", backupID, prePoint.ID), string(payload), now, userID)
		if err != nil {
			return prePoint.ID, err
		}
	}
	if err := verifySnapshotCounts(ctx, tx, d); err != nil {
		return prePoint.ID, err
	}
	var fkTable string
	if err := tx.QueryRowContext(ctx, `PRAGMA foreign_key_check`).Scan(&fkTable); err != sql.ErrNoRows {
		if err == nil {
			return prePoint.ID, errors.New("relasi hasil pemulihan tidak valid")
		}
		// PRAGMA returns four columns for violations; a scan error also means a violation.
		return prePoint.ID, fmt.Errorf("validasi relasi hasil pemulihan: %w", err)
	}
	uid, eid, classID := userID, backupID, rec.ClassID
	countsJSON, _ := json.Marshal(map[string]any{"backup_id": backupID, "pre_restore_backup_id": prePoint.ID, "source_counts": counts(d)})
	after := string(countsJSON)
	if err := audit.Write(ctx, tx, audit.Entry{Actor: audit.Actor{Type: "USER", UserID: &uid}, ClassID: &classID, SemesterID: rec.SemesterID,
		Action: "RESTORE", EntityType: "BACKUP_RECORD", EntityID: &eid, AfterJSON: &after, Reason: strings.TrimSpace(reason),
		CorrelationID: fmt.Sprintf("restore-%d-%d", backupID, time.Now().UnixNano())}); err != nil {
		return prePoint.ID, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE backup_records SET status='VERIFIED', verified_at=?, updated_at=? WHERE id=?`, now, now, backupID); err != nil {
		return prePoint.ID, err
	}
	if err := tx.Commit(); err != nil {
		return prePoint.ID, err
	}
	return prePoint.ID, nil
}
