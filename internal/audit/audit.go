package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"strings"
)

// Entry describes one audit log row.
// ClassID/SemesterID may be nil for global actions.
type Entry struct {
	ClassID             *int64
	SemesterID          *int64
	ActorUserID         *int64
	ActorRoleAssignment *int64
	ActorContextJSON    *string
	ActorType           string // USER or SYSTEM
	Action              string
	EntityType          string
	EntityID            *int64
	BeforeJSON          *string
	AfterJSON           *string
	Reason              *string
	CorrelationID       string
}

// DBTX abstracts *sql.DB and *sql.Tx for audit writes inside transactions.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func NewCorrelationID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}

func strOrNil(s *string) any {
	if s == nil {
		return nil
	}
	if strings.TrimSpace(*s) == "" {
		return nil
	}
	return *s
}

// Write inserts one audit log row. Caller must ensure foreign keys exist.
func Write(ctx context.Context, db DBTX, e Entry) error {
	actorType := strings.ToUpper(strings.TrimSpace(e.ActorType))
	if actorType == "" {
		actorType = "USER"
	}
	corr := strings.TrimSpace(e.CorrelationID)
	if corr == "" {
		corr = NewCorrelationID()
	}
	_, err := db.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id,
		actor_context_json, actor_type, action, entity_type, entity_id,
		before_json, after_json, reason, correlation_id,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		e.ClassID, e.SemesterID, e.ActorUserID, e.ActorRoleAssignment,
		strOrNil(e.ActorContextJSON), actorType,
		strings.TrimSpace(e.Action), strings.TrimSpace(e.EntityType), e.EntityID,
		strOrNil(e.BeforeJSON), strOrNil(e.AfterJSON), strOrNil(e.Reason), corr,
	)
	return err
}
