package academic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func parseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("timestamp kosong")
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("format timestamp %q tidak didukung", raw)
}

func parseNullTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := parseTime(*s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetClasses mengambil seluruh kelas akademik terdaftar yang terurut berdasarkan kode.
func (r *Repository) GetClasses(ctx context.Context) ([]Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		ORDER BY code ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data kelas: %w", err)
	}
	defer rows.Close()

	var classes []Class
	for rows.Next() {
		var c Class
		var createdAt, updatedAt string
		err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Slug,
			&c.StudyProgram,
			&c.CohortYear,
			&c.GroupLabel,
			&c.Status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris kelas: %w", err)
		}
		c.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp created_at kelas ID %d tidak valid: %w", c.ID, err)
		}
		c.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp updated_at kelas ID %d tidak valid: %w", c.ID, err)
		}
		classes = append(classes, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris kelas: %w", err)
	}
	return classes, nil
}

// GetCoursesByClassID mengambil seluruh mata kuliah yang ditawarkan untuk kelas tertentu.
func (r *Repository) GetCoursesByClassID(ctx context.Context, classID int64) ([]Course, error) {
	query := `
		SELECT DISTINCT c.id, c.code, c.name, c.status, c.created_at, c.updated_at
		FROM courses c
		JOIN course_offerings co ON co.course_id = c.id
		JOIN semesters s ON s.id = co.semester_id
		WHERE s.class_id = ?
		ORDER BY c.code ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, classID)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil mata kuliah untuk kelas ID %d: %w", classID, err)
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		var c Course
		var createdAt, updatedAt string
		err := rows.Scan(
			&c.ID,
			&c.Code,
			&c.Name,
			&c.Status,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("gagal membaca baris mata kuliah: %w", err)
		}
		c.CreatedAt, err = parseTime(createdAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp created_at mata kuliah ID %d tidak valid: %w", c.ID, err)
		}
		c.UpdatedAt, err = parseTime(updatedAt)
		if err != nil {
			return nil, fmt.Errorf("timestamp updated_at mata kuliah ID %d tidak valid: %w", c.ID, err)
		}
		courses = append(courses, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("kesalahan iterasi baris mata kuliah: %w", err)
	}
	return courses, nil
}

func (r *Repository) CountClasses(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM classes;").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung jumlah kelas: %w", err)
	}
	return count, nil
}

func (r *Repository) GetClassByID(ctx context.Context, id int64) (*Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		WHERE id = ?;
	`
	var c Class
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID,
		&c.Code,
		&c.Slug,
		&c.StudyProgram,
		&c.CohortYear,
		&c.GroupLabel,
		&c.Status,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil kelas ID %d: %w", id, err)
	}
	c.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp created_at kelas ID %d tidak valid: %w", id, err)
	}
	c.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp updated_at kelas ID %d tidak valid: %w", id, err)
	}
	return &c, nil
}

func (r *Repository) GetClassByCode(ctx context.Context, code string) (*Class, error) {
	query := `
		SELECT id, code, slug, study_program, cohort_year, group_label, status, created_at, updated_at
		FROM classes
		WHERE code = ?;
	`
	var c Class
	var createdAt, updatedAt string
	err := r.db.QueryRowContext(ctx, query, code).Scan(
		&c.ID,
		&c.Code,
		&c.Slug,
		&c.StudyProgram,
		&c.CohortYear,
		&c.GroupLabel,
		&c.Status,
		&createdAt,
		&updatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil kelas dengan kode '%s': %w", code, err)
	}
	c.CreatedAt, err = parseTime(createdAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp created_at kelas %q tidak valid: %w", code, err)
	}
	c.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return nil, fmt.Errorf("timestamp updated_at kelas %q tidak valid: %w", code, err)
	}
	return &c, nil
}
