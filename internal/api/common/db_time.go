package common

import (
	"fmt"
	"strings"
	"time"
)

// ParseTime mem-parsing berbagai format timestamp string ke time.Time.
func ParseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("timestamp kosong")
	}
	if idx := strings.Index(raw, " m="); idx != -1 {
		raw = raw[:idx]
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700 -0700",
		"2006-01-02 15:04:05.999999999 -0700 +07",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05.999999 -0700 +07",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05.999999999",
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

// DBTimestamp accepts timestamps returned by both the legacy DATETIME schema
// and the canonical TEXT/RFC3339 schema.
type DBTimestamp struct {
	Time  time.Time
	Valid bool
}

func (t *DBTimestamp) Scan(src any) error {
	switch value := src.(type) {
	case nil:
		t.Time = time.Time{}
		t.Valid = false
		return nil
	case time.Time:
		t.Time = value
		t.Valid = true
		return nil
	case string:
		return t.parse(value)
	case []byte:
		return t.parse(string(value))
	default:
		return fmt.Errorf("tipe timestamp database %T tidak didukung", src)
	}
}

func (t *DBTimestamp) parse(raw string) error {
	parsed, err := ParseTime(raw)
	if err != nil {
		return err
	}
	t.Time = parsed
	t.Valid = true
	return nil
}

func (t DBTimestamp) RFC3339() any {
	if !t.Valid {
		return nil
	}
	return t.Time.Format(time.RFC3339)
}
