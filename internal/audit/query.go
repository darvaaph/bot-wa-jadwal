package audit

import (
	"context"
	"database/sql"
	"strings"
)

type Filter struct {
	ClassID    *int64
	EntityType string
	EntityID   *int64
	ActorID    *int64
	Action     string
	From       string
	To         string
	Limit      int
}

type Item struct {
	ID                  int64   `json:"id"`
	ClassID             *int64  `json:"class_id,omitempty"`
	SemesterID          *int64  `json:"semester_id,omitempty"`
	ActorUserID         *int64  `json:"actor_user_id,omitempty"`
	ActorRoleAssignment *int64  `json:"actor_role_assignment_id,omitempty"`
	ActorType           string  `json:"actor_type"`
	Action              string  `json:"action"`
	EntityType          string  `json:"entity_type"`
	EntityID            *int64  `json:"entity_id,omitempty"`
	BeforeJSON          *string `json:"before_json,omitempty"`
	AfterJSON           *string `json:"after_json,omitempty"`
	Reason              *string `json:"reason,omitempty"`
	CorrelationID       string  `json:"correlation_id"`
	CreatedAt           string  `json:"created_at"`
}

// ListScoped returns audit rows visible to the caller.
// Admin (isAdmin) sees all with optional class filter.
// KM sees all rows of their class.
// PJ sees rows of their class limited to own offering entities + own actions.
func ListScoped(ctx context.Context, db *sql.DB, f Filter, isAdmin bool, kmClassID *int64, pjUserID *int64, pjOfferingID *int64) ([]Item, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := `SELECT id, class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, before_json, after_json, reason, correlation_id, created_at
		FROM audit_logs WHERE 1=1`
	args := []any{}
	if !isAdmin {
		if kmClassID != nil {
			query += ` AND (class_id = ? OR class_id IS NULL)`
			args = append(args, *kmClassID)
		} else if pjUserID != nil && pjOfferingID != nil {
			query += ` AND (actor_user_id = ? OR
				(entity_type = 'TASK' AND entity_id IN (SELECT id FROM tasks WHERE course_offering_id = ?)) OR
				(entity_type = 'TEACHING_EVENT' AND entity_id IN (SELECT teaching_event_id FROM teaching_event_offerings WHERE course_offering_id = ?)) OR
				(entity_type = 'MATERIAL' AND entity_id IN (SELECT id FROM materials WHERE course_offering_id = ?)))`
			args = append(args, *pjUserID, *pjOfferingID, *pjOfferingID, *pjOfferingID)
		} else {
			return []Item{}, nil
		}
	} else if f.ClassID != nil {
		query += ` AND class_id = ?`
		args = append(args, *f.ClassID)
	}
	if f.EntityType != "" {
		query += ` AND entity_type = ?`
		args = append(args, strings.ToUpper(f.EntityType))
	}
	if f.EntityID != nil {
		query += ` AND entity_id = ?`
		args = append(args, *f.EntityID)
	}
	if f.ActorID != nil {
		query += ` AND actor_user_id = ?`
		args = append(args, *f.ActorID)
	}
	if f.Action != "" {
		query += ` AND action = ?`
		args = append(args, strings.ToUpper(f.Action))
	}
	if f.From != "" {
		query += ` AND created_at >= ?`
		args = append(args, f.From)
	}
	if f.To != "" {
		query += ` AND created_at <= ?`
		args = append(args, f.To)
	}
	// PJ without explicit class filter is still implicitly class-scoped above; apply class filter if given.
	if f.ClassID != nil && !isAdmin && kmClassID == nil {
		query += ` AND class_id = ?`
		args = append(args, *f.ClassID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var classID, semID, actorUID, actorRA, entityID sql.NullInt64
		var before, after, reason sql.NullString
		if err := rows.Scan(&it.ID, &classID, &semID, &actorUID, &actorRA, &it.ActorType,
			&it.Action, &it.EntityType, &entityID, &before, &after, &reason, &it.CorrelationID, &it.CreatedAt); err != nil {
			return nil, err
		}
		if classID.Valid {
			it.ClassID = &classID.Int64
		}
		if semID.Valid {
			it.SemesterID = &semID.Int64
		}
		if actorUID.Valid {
			it.ActorUserID = &actorUID.Int64
		}
		if actorRA.Valid {
			it.ActorRoleAssignment = &actorRA.Int64
		}
		if entityID.Valid {
			it.EntityID = &entityID.Int64
		}
		if before.Valid {
			it.BeforeJSON = &before.String
		}
		if after.Valid {
			it.AfterJSON = &after.String
		}
		if reason.Valid {
			it.Reason = &reason.String
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
