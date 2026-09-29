package audit

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"bot-jadwal/internal/database"
)

func TestAudit_Validation(t *testing.T) {
	uid := int64(1)
	cases := []struct {
		name  string
		entry Entry
		want  bool
	}{
		{"valid user", Entry{Actor: Actor{Type: "USER", UserID: &uid}, Action: "CREATE", EntityType: "TASK", CorrelationID: "c1"}, true},
		{"missing action", Entry{Actor: Actor{Type: "USER", UserID: &uid}, EntityType: "TASK", CorrelationID: "c1"}, false},
		{"missing entity", Entry{Actor: Actor{Type: "USER", UserID: &uid}, Action: "CREATE", CorrelationID: "c1"}, false},
		{"missing corr", Entry{Actor: Actor{Type: "USER", UserID: &uid}, Action: "CREATE", EntityType: "TASK"}, false},
		{"user tanpa id", Entry{Actor: Actor{Type: "USER"}, Action: "CREATE", EntityType: "TASK", CorrelationID: "c1"}, false},
		{"system dengan id", Entry{Actor: Actor{Type: "SYSTEM", UserID: &uid}, Action: "CREATE", EntityType: "TASK", CorrelationID: "c1"}, false},
		{"restore tanpa reason", Entry{Actor: Actor{Type: "SYSTEM"}, Action: "RESTORE", EntityType: "TASK", CorrelationID: "c1"}, false},
		{"restore whitespace", Entry{Actor: Actor{Type: "SYSTEM"}, Action: "RESTORE", EntityType: "TASK", CorrelationID: "c1", Reason: "   "}, false},
		{"system valid", Entry{Actor: Actor{Type: "SYSTEM"}, Action: "CREATE", EntityType: "TASK", CorrelationID: "c1"}, true},
	}
	for _, tc := range cases {
		if err := Validate(tc.entry); (err == nil) != tc.want {
			t.Errorf("%s: want valid=%v got err=%v", tc.name, tc.want, err)
		}
	}
}

func TestAudit_RedactionNested(t *testing.T) {
	raw := `{"title":"x","password":"rahasia","nested":{"portal_code":"123","ok":1},"list":[{"secret":"s"}]}`
	san, err := SanitizeJSON(&raw)
	if err != nil || san == nil {
		t.Fatalf("sanitize err=%v", err)
	}
	for _, bad := range []string{"rahasia", "123"} {
		if strings.Contains(*san, bad) {
			t.Fatalf("secret bocor: %s", *san)
		}
	}
	if !strings.Contains(*san, "[REDACTED]") {
		t.Fatalf("redaksi hilang: %s", *san)
	}
}

func TestAudit_AtomicRollback(t *testing.T) {
	db, err := database.InitDB(filepath.Join(t.TempDir(), "audit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('A','a','TI',2024,'A')`); err != nil {
		t.Fatal(err)
	}
	uid := int64(1)
	_ = uid
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Audit gagal validasi -> rollback domain.
	if _, err := tx.ExecContext(ctx, `INSERT INTO classes (code, slug, study_program, cohort_year, group_label) VALUES ('B','b','TI',2024,'B')`); err != nil {
		t.Fatal(err)
	}
	bad := Entry{Actor: Actor{Type: "USER"}, Action: "CREATE", EntityType: "CLASS", CorrelationID: "c1"}
	if err := writeEntry(ctx, tx, bad); err == nil {
		t.Fatal("audit invalid harus gagal")
	}
	_ = tx.Rollback()
	var n int
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM classes WHERE code='B'`).Scan(&n)
	if n != 0 {
		t.Fatalf("rollback harus membatalkan domain, got %d", n)
	}
}
