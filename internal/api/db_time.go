package api

import (
	"fmt"
	"time"
)

// dbTimestamp accepts timestamps returned by both the legacy DATETIME schema
// and the canonical TEXT/RFC3339 schema.
type dbTimestamp struct {
	Time  time.Time
	Valid bool
}

func (t *dbTimestamp) Scan(src any) error {
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

func (t *dbTimestamp) parse(raw string) error {
	parsed, err := parseTime(raw)
	if err != nil {
		return err
	}
	t.Time = parsed
	t.Valid = true
	return nil
}

func (t dbTimestamp) RFC3339() any {
	if !t.Valid {
		return nil
	}
	return t.Time.Format(time.RFC3339)
}
