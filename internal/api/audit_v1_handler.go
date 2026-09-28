package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// AuditLogResponseItem merepresentasikan catatan riwayat audit sistem
type AuditLogResponseItem struct {
	ID          int64   `json:"id"`
	ClassID     *int64  `json:"class_id,omitempty"`
	ClassSlug   *string `json:"class_slug,omitempty"`
	ActorUserID *int64  `json:"actor_user_id,omitempty"`
	ActorName   *string `json:"actor_name,omitempty"`
	ActorRole   *string `json:"actor_role,omitempty"`
	Action      string  `json:"action"`
	EntityType  *string `json:"entity_type,omitempty"`
	EntityID    *int64  `json:"entity_id,omitempty"`
	BeforeJSON  *string `json:"before_json,omitempty"`
	AfterJSON   *string `json:"after_json,omitempty"`
	Reason      *string `json:"reason,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// handleGetAuditLogs menangani GET /api/v1/audit
func (s *Server) handleGetAuditLogs(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya KM atau System Admin yang berwenang menelaah log audit")
		return
	}

	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	actionFilter := strings.TrimSpace(r.URL.Query().Get("action"))
	entityTypeFilter := strings.TrimSpace(r.URL.Query().Get("entity_type"))

	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT al.id, al.class_id, cl.slug, al.actor_user_id, u.display_name,
		       ra.role, al.action, al.entity_type, al.entity_id,
		       al.before_json, al.after_json, al.reason, al.created_at
		FROM audit_logs al
		LEFT JOIN classes cl ON al.class_id = cl.id
		LEFT JOIN users u ON al.actor_user_id = u.id
		LEFT JOIN role_assignments ra ON al.actor_role_assignment_id = ra.id
		WHERE (1=1)
	`
	var args []any

	// Pembatasan cakupan: KM hanya dapat membaca audit kelas miliknya
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Konteks kelas KM tidak valid")
			return
		}
		query += " AND al.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND cl.slug = ?"
		args = append(args, classSlugFilter)
	}

	if actionFilter != "" {
		query += " AND al.action = ?"
		args = append(args, actionFilter)
	}

	if entityTypeFilter != "" {
		query += " AND al.entity_type = ?"
		args = append(args, entityTypeFilter)
	}

	query += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := s.v1DB.Query(query, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memuat log audit: %v", err))
		return
	}
	defer rows.Close()

	logs := []AuditLogResponseItem{}
	for rows.Next() {
		var (
			id         int64
			classID    sql.NullInt64
			classSlug  sql.NullString
			actorID    sql.NullInt64
			actorName  sql.NullString
			actorRole  sql.NullString
			action     string
			entityType sql.NullString
			entityID   sql.NullInt64
			beforeJSON sql.NullString
			afterJSON  sql.NullString
			reason     sql.NullString
			createdAt  time.Time
		)

		if err := rows.Scan(&id, &classID, &classSlug, &actorID, &actorName, &actorRole,
			&action, &entityType, &entityID, &beforeJSON, &afterJSON, &reason, &createdAt); err == nil {

			item := AuditLogResponseItem{
				ID:        id,
				Action:    action,
				CreatedAt: createdAt.Format(time.RFC3339),
			}
			if classID.Valid {
				item.ClassID = &classID.Int64
			}
			if classSlug.Valid {
				item.ClassSlug = &classSlug.String
			}
			if actorID.Valid {
				item.ActorUserID = &actorID.Int64
			}
			if actorName.Valid {
				item.ActorName = &actorName.String
			}
			if actorRole.Valid {
				item.ActorRole = &actorRole.String
			}
			if entityType.Valid {
				item.EntityType = &entityType.String
			}
			if entityID.Valid {
				item.EntityID = &entityID.Int64
			}
			if beforeJSON.Valid {
				item.BeforeJSON = &beforeJSON.String
			}
			if afterJSON.Valid {
				item.AfterJSON = &afterJSON.String
			}
			if reason.Valid {
				item.Reason = &reason.String
			}

			logs = append(logs, item)
		}
	}

	s.writeV1Success(w, http.StatusOK, logs)
}
