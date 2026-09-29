package academic

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"bot-jadwal/internal/audit"
)

// Actor membawa identitas pelaksana aksi untuk pencatatan audit log.
type Actor struct {
	UserID           int64
	RoleAssignmentID int64
	CorrelationID    string
}

// AuditIdentity mengembalikan tuple data identitas audit.
func (a Actor) AuditIdentity() (any, any, string, string) {
	corr := strings.TrimSpace(a.CorrelationID)
	if corr == "" {
		corr = time.Now().UTC().Format(time.RFC3339Nano)
	}
	var actorUser, actorAssignment any
	actorType := "USER"
	if a.UserID > 0 {
		actorUser = a.UserID
	} else {
		actorType = "SYSTEM"
	}
	if a.RoleAssignmentID > 0 {
		actorAssignment = a.RoleAssignmentID
	}
	return actorUser, actorAssignment, actorType, corr
}

// WriteAuditLog mencatat jejak audit ke dalam transaksi database.
func WriteAuditLog(ctx context.Context, tx *sql.Tx, actor Actor, classID *int64, semesterID *int64, action, entityType string, entityID *int64, before, after *string, reason string) error {
	corr := strings.TrimSpace(actor.CorrelationID)
	if corr == "" {
		corr = time.Now().UTC().Format(time.RFC3339Nano)
	}
	var actorEntry audit.Actor
	if actor.UserID > 0 {
		uid := actor.UserID
		actorEntry = audit.Actor{Type: "USER", UserID: &uid}
		if actor.RoleAssignmentID > 0 {
			raid := actor.RoleAssignmentID
			actorEntry.RoleAssignmentID = &raid
		}
	} else {
		actorEntry = audit.Actor{Type: "SYSTEM"}
	}
	return audit.Write(ctx, tx, audit.Entry{
		Actor:         actorEntry,
		ClassID:       classID,
		SemesterID:    semesterID,
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		BeforeJSON:    before,
		AfterJSON:     after,
		Reason:        reason,
		CorrelationID: corr,
	})
}
