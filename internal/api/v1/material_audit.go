package v1

import (
	"database/sql"
	"net/http"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/audit"
)

type materialAuditSnapshot struct {
	ClassID     int64         `json:"class_id"`
	OfferingID  sql.NullInt64 `json:"offering_id"`
	TaskID      sql.NullInt64 `json:"task_id"`
	SemesterID  sql.NullInt64 `json:"semester_id"`
	Title       string        `json:"title"`
	URL         string        `json:"url"`
	Description string        `json:"description"`
	Type        string        `json:"material_type"`
	Status      string        `json:"status"`
	Version     int           `json:"version"`
}

func materialSnapshot(r *http.Request, tx *sql.Tx, id int64) (*materialAuditSnapshot, error) {
	var m materialAuditSnapshot
	err := tx.QueryRowContext(r.Context(), `SELECT m.class_id,m.course_offering_id,m.task_id,co.semester_id,m.title,m.url,COALESCE(m.description,''),m.material_type,m.status,m.version FROM materials m LEFT JOIN tasks t ON t.id=m.task_id LEFT JOIN course_offerings co ON co.id=COALESCE(m.course_offering_id,t.course_offering_id) WHERE m.id=?`, id).Scan(&m.ClassID, &m.OfferingID, &m.TaskID, &m.SemesterID, &m.Title, &m.URL, &m.Description, &m.Type, &m.Status, &m.Version)
	return &m, err
}

func writeMaterialAudit(r *http.Request, tx *sql.Tx, u *common.UserContext, id int64, action string, before *materialAuditSnapshot) error {
	after, err := materialSnapshot(r, tx, id)
	if err != nil {
		return err
	}
	var sem *int64
	if after.SemesterID.Valid {
		sem = &after.SemesterID.Int64
	}
	e := audit.Entry{Actor: audit.Actor{Type: "USER", UserID: &u.UserID, RoleAssignmentID: &u.ActiveAssignmentID}, ClassID: &after.ClassID, SemesterID: sem, Action: action, EntityType: "MATERIAL", EntityID: &id, After: after}
	if before != nil {
		e.Before = before
	}
	return audit.Write(r.Context(), tx, e)
}
