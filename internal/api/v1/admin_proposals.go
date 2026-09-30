package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/audit"
)

// ProposalItem merepresentasikan satu usulan koreksi master oleh KM.
type ProposalItem struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind"`
	TargetID   *int64  `json:"target_id,omitempty"`
	TargetCode *string `json:"target_code,omitempty"`
	Payload    string  `json:"payload_json"`
	Note       string  `json:"note"`
	Status     string  `json:"status"`
	ProposedBy string  `json:"proposed_by"`
	ClassID    int64   `json:"class_id"`
	ClassSlug  string  `json:"class_slug"`
	ReviewedBy *string `json:"reviewed_by,omitempty"`
	ReviewNote *string `json:"review_note,omitempty"`
	DecidedAt  *string `json:"decided_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// CreateProposalRequest adalah payload usulan koreksi master (BE-007, khusus KM).
type CreateProposalRequest struct {
	Kind     string         `json:"kind"`
	TargetID *int64         `json:"target_id,omitempty"`
	Payload  map[string]any `json:"payload"`
	Note     string         `json:"note"`
}

// canonicalProposal memvalidasi + menormalkan payload usulan.
// newTarget benar bila usulan penambahan (target_id kosong).
func canonicalProposal(kind string, targetID *int64, payload map[string]any) (map[string]any, error) {
	isNew := targetID == nil || *targetID <= 0
	clean := map[string]any{}
	strField := func(key string) (string, bool) {
		v, ok := payload[key]
		if !ok || v == nil {
			return "", false
		}
		s, ok := v.(string)
		return strings.TrimSpace(s), ok
	}
	if kind == "ROOM" {
		for key := range payload {
			switch key {
			case "name", "building", "room_type", "capacity", "code":
			default:
				return nil, fmt.Errorf("field %q tidak dikenal untuk ruangan", key)
			}
		}
		if code, ok := strField("code"); ok && code != "" {
			if !isNew {
				return nil, fmt.Errorf("kode tidak dapat diubah")
			}
			clean["code"] = code
		} else if isNew {
			return nil, fmt.Errorf("kode wajib diisi untuk ruangan baru")
		}
		for _, key := range []string{"name", "building", "room_type"} {
			if s, ok := strField(key); ok && s != "" {
				clean[key] = s
			}
		}
		if v, ok := payload["capacity"]; ok && v != nil {
			n, ok := v.(float64)
			if !ok || n < 0 || n != float64(int(n)) {
				return nil, fmt.Errorf("kapasitas wajib angka bulat \u2265 0")
			}
			clean["capacity"] = int(n)
		}
		if len(clean) == 0 {
			return nil, fmt.Errorf("payload kosong: tidak ada field yang diusulkan")
		}
		return clean, nil
	}
	// COURSE
	for key := range payload {
		switch key {
		case "name", "code":
		default:
			return nil, fmt.Errorf("field %q tidak dikenal untuk mata kuliah", key)
		}
	}
	name, _ := strField("name")
	if name == "" {
		return nil, fmt.Errorf("nama mata kuliah wajib diisi")
	}
	clean["name"] = name
	if code, ok := strField("code"); ok && code != "" {
		if !isNew {
			return nil, fmt.Errorf("kode tidak dapat diubah")
		}
		clean["code"] = code
	} else if isNew {
		return nil, fmt.Errorf("kode wajib diisi untuk mata kuliah baru")
	}
	return clean, nil
}

// CreateProposal menangani POST /api/v1/master/proposals (khusus KM).
func (c *AdminController) CreateProposal(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "KM" || !u.ActiveClassID.Valid {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM yang berwenang mengusulkan koreksi master")
		return
	}

	var req CreateProposalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	kind := strings.ToUpper(strings.TrimSpace(req.Kind))
	if kind != "ROOM" && kind != "COURSE" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kind harus ROOM atau COURSE")
		return
	}
	if req.Payload == nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload usulan wajib diisi")
		return
	}
	clean, err := canonicalProposal(kind, req.TargetID, req.Payload)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, err.Error())
		return
	}
	var targetID sql.NullInt64
	if req.TargetID != nil && *req.TargetID > 0 {
		table := "rooms"
		if kind == "COURSE" {
			table = "courses"
		}
		var exists int
		if err := c.db.QueryRow(`SELECT 1 FROM `+table+` WHERE id = ?;`, *req.TargetID).Scan(&exists); err != nil {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Target usulan tidak ditemukan")
			return
		}
		targetID = sql.NullInt64{Int64: *req.TargetID, Valid: true}
	}
	payloadBytes, _ := json.Marshal(clean)

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi usulan")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		INSERT INTO master_proposals (kind, target_id, payload_json, note, status, proposed_by_user_id, class_id)
		VALUES (?, ?, ?, ?, 'PENDING', ?, ?);
	`, kind, targetID, string(payloadBytes), strings.TrimSpace(req.Note), u.UserID, u.ActiveClassID.Int64)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan usulan")
		return
	}
	id, _ := res.LastInsertId()

	uid := u.UserID
	var raid *int64
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	afterJSON := string(payloadBytes)
	if err := audit.Write(r.Context(), tx, audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		ClassID:       &u.ActiveClassID.Int64,
		Action:        "PROPOSE_MASTER_CORRECTION",
		EntityType:    "MASTER_PROPOSAL",
		EntityID:      &id,
		AfterJSON:     &afterJSON,
		Reason:        strings.TrimSpace(req.Note),
		CorrelationID: fmt.Sprintf("propose-%d-%d", id, time.Now().UnixNano()),
	}); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan usulan beserta auditnya")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan usulan")
		return
	}

	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "status": "PENDING"})
}

// GetProposals menangani GET /api/v1/master/proposals.
// System Admin lintas kelas; KM hanya kelasnya.
func (c *AdminController) GetProposals(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang melihat usulan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	kindFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("kind")))
	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	limit, offset := adminListPaging(r)

	query := `
		SELECT p.id, p.kind, p.target_id,
		       COALESCE(r.code, co.code),
		       p.payload_json, p.note, p.status,
		       pu.display_name, p.class_id, cl.slug,
		       ru.display_name, p.review_note, p.decided_at, p.created_at
		FROM master_proposals p
		JOIN users pu ON pu.id = p.proposed_by_user_id
		JOIN classes cl ON cl.id = p.class_id
		LEFT JOIN users ru ON ru.id = p.reviewed_by_user_id
		LEFT JOIN rooms r ON p.kind = 'ROOM' AND p.target_id = r.id
		LEFT JOIN courses co ON p.kind = 'COURSE' AND p.target_id = co.id
		WHERE (1=1)
	`
	var args []any
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Konteks kelas KM tidak valid")
			return
		}
		if classSlugFilter != "" {
			var ownSlug string
			_ = c.db.QueryRow(`SELECT slug FROM classes WHERE id = ?;`, u.ActiveClassID.Int64).Scan(&ownSlug)
			if classSlugFilter != ownSlug {
				common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Usulan tidak ditemukan")
				return
			}
		}
		query += " AND p.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND cl.slug = ?"
		args = append(args, classSlugFilter)
	}
	if statusFilter != "" {
		query += " AND p.status = ?"
		args = append(args, statusFilter)
	}
	if kindFilter != "" {
		query += " AND p.kind = ?"
		args = append(args, kindFilter)
	}
	query += " ORDER BY p.id DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat usulan")
		return
	}
	defer rows.Close()

	items := []ProposalItem{}
	for rows.Next() {
		var it ProposalItem
		var targetID sql.NullInt64
		var targetCode, reviewedBy, reviewNote, decidedAt sql.NullString
		if err := rows.Scan(
			&it.ID, &it.Kind, &targetID, &targetCode,
			&it.Payload, &it.Note, &it.Status,
			&it.ProposedBy, &it.ClassID, &it.ClassSlug,
			&reviewedBy, &reviewNote, &decidedAt, &it.CreatedAt,
		); err != nil {
			continue
		}
		if targetID.Valid {
			it.TargetID = &targetID.Int64
		}
		if targetCode.Valid {
			it.TargetCode = &targetCode.String
		}
		if reviewedBy.Valid {
			it.ReviewedBy = &reviewedBy.String
		}
		if reviewNote.Valid {
			it.ReviewNote = &reviewNote.String
		}
		if decidedAt.Valid {
			it.DecidedAt = &decidedAt.String
		}
		items = append(items, it)
	}

	common.WriteV1Success(w, http.StatusOK, items)
}

// decideProposal mengeksekusi approve/reject usulan (khusus System Admin).
func (c *AdminController) decideProposal(w http.ResponseWriter, r *http.Request, approve bool) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang memutuskan usulan")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID usulan tidak valid")
		return
	}

	var p struct {
		id       int64
		kind     string
		targetID sql.NullInt64
		payload  string
		status   string
		classID  int64
	}
	if err := c.db.QueryRow(`
		SELECT id, kind, target_id, payload_json, status, class_id
		FROM master_proposals WHERE id = ?;
	`, id).Scan(&p.id, &p.kind, &p.targetID, &p.payload, &p.status, &p.classID); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Usulan tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi usulan")
		return
	}
	if p.status != "PENDING" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, fmt.Sprintf("Usulan sudah %s", p.status))
		return
	}

	var body struct {
		ReviewNote *string `json:"review_note"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	reviewNote := ""
	if body.ReviewNote != nil {
		reviewNote = strings.TrimSpace(*body.ReviewNote)
	}
	if !approve && reviewNote == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Catatan penolakan wajib diisi")
		return
	}

	var fields map[string]any
	if err := json.Unmarshal([]byte(p.payload), &fields); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Payload usulan rusak")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi usulan")
		return
	}
	defer tx.Rollback()

	newStatus := "REJECTED"
	action := "REJECT_MASTER_PROPOSAL"
	masterAction := ""
	var masterEntityID int64
	var masterBefore, masterAfter string
	if approve {
		newStatus = "APPROVED"
		action = "APPROVE_MASTER_PROPOSAL"
		var applyErr error
		masterAction, masterEntityID, masterBefore, masterAfter, applyErr = applyProposal(tx, p.kind, p.targetID, fields)
		if applyErr != nil {
			if msg := applyErr.Error(); strings.Contains(msg, "UNIQUE") || strings.Contains(msg, "sudah dipakai") {
				common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, msg)
			} else {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menerapkan usulan")
			}
			return
		}
	}

	if _, err := tx.Exec(`
		UPDATE master_proposals
		SET status = ?, reviewed_by_user_id = ?, review_note = ?, decided_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'PENDING';
	`, newStatus, u.UserID, reviewNote, p.id); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memutuskan usulan")
		return
	}

	uid := u.UserID
	var raid *int64
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	// Satu ID korelasi untuk keputusan + penerapan agar jejak ganda tertaut.
	opusID := fmt.Sprintf("proposal-%d-%d", p.id, time.Now().UnixNano())
	beforeJSON := `{"status":"PENDING"}`
	afterJSON := fmt.Sprintf(`{"status":%q,"review_note":%q}`, newStatus, reviewNote)
	proposalEntry := audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		ClassID:       &p.classID,
		Action:        action,
		EntityType:    "MASTER_PROPOSAL",
		EntityID:      &p.id,
		BeforeJSON:    &beforeJSON,
		AfterJSON:     &afterJSON,
		Reason:        reviewNote,
		CorrelationID: opusID,
	}
	if err := audit.Write(r.Context(), tx, proposalEntry); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit usulan")
		return
	}
	if approve {
		entity := "MASTER_ROOM"
		if p.kind == "COURSE" {
			entity = "MASTER_COURSE"
		}
		if err := writeMasterAuditTx(r.Context(), tx, uid, raid, masterAction, entity, masterEntityID, masterBefore, masterAfter, opusID); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit master")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit usulan")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": p.id, "status": newStatus})
}

// ApproveProposal menangani POST /api/v1/master/proposals/{id}/approve.
func (c *AdminController) ApproveProposal(w http.ResponseWriter, r *http.Request) {
	c.decideProposal(w, r, true)
}

// RejectProposal menangani POST /api/v1/master/proposals/{id}/reject.
func (c *AdminController) RejectProposal(w http.ResponseWriter, r *http.Request) {
	c.decideProposal(w, r, false)
}

// applyProposal menerapkan payload usulan yang sudah dikanonikalisasi ke master.
// Mengembalikan aksi audit, ID entitas, dan JSON sebelum/sesudah.
func applyProposal(tx *sql.Tx, kind string, targetID sql.NullInt64, fields map[string]any) (string, int64, string, string, error) {
	strOf := func(key string) string {
		if v, ok := fields[key].(string); ok {
			return v
		}
		return ""
	}
	if kind == "ROOM" {
		if targetID.Valid {
			var before struct {
				name, building, roomType, status string
				capacity                         int
			}
			if err := tx.QueryRow(`SELECT name, COALESCE(building,''), COALESCE(room_type,''), COALESCE(capacity,0), status FROM rooms WHERE id = ?;`,
				targetID.Int64).Scan(&before.name, &before.building, &before.roomType, &before.capacity, &before.status); err != nil {
				return "", 0, "", "", fmt.Errorf("target ruangan tidak ditemukan")
			}
			sets := []string{}
			args := []any{}
			after := before
			if s := strOf("name"); s != "" {
				sets = append(sets, "name = ?")
				args = append(args, s)
				after.name = s
			}
			if s, ok := fields["building"].(string); ok {
				sets = append(sets, "building = ?")
				args = append(args, strings.TrimSpace(s))
				after.building = strings.TrimSpace(s)
			}
			if s, ok := fields["room_type"].(string); ok {
				sets = append(sets, "room_type = ?")
				args = append(args, strings.TrimSpace(s))
				after.roomType = strings.TrimSpace(s)
			}
			if n, ok := fields["capacity"].(float64); ok {
				sets = append(sets, "capacity = ?")
				args = append(args, int(n))
				after.capacity = int(n)
			}
			if len(sets) == 0 {
				return "", 0, "", "", fmt.Errorf("tidak ada field yang diterapkan")
			}
			args = append(args, targetID.Int64)
			if _, err := tx.Exec(`UPDATE rooms SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...); err != nil {
				return "", 0, "", "", err
			}
			beforeJSON := fmt.Sprintf(`{"name":%q,"building":%q,"room_type":%q,"capacity":%d,"status":%q}`,
				before.name, before.building, before.roomType, before.capacity, before.status)
			afterJSON := fmt.Sprintf(`{"name":%q,"building":%q,"room_type":%q,"capacity":%d,"status":%q}`,
				after.name, after.building, after.roomType, after.capacity, after.status)
			return "UPDATE_MASTER_ROOM", targetID.Int64, beforeJSON, afterJSON, nil
		}
		code := strOf("code")
		name := strOf("name")
		building := strOf("building")
		roomType := strOf("room_type")
		capacity := 0
		if n, ok := fields["capacity"].(float64); ok {
			capacity = int(n)
		}
		res, err := tx.Exec(`INSERT INTO rooms (code, name, building, room_type, capacity, status) VALUES (?, ?, ?, ?, ?, 'ACTIVE');`,
			code, name, building, roomType, capacity)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				return "", 0, "", "", fmt.Errorf("kode ruangan sudah dipakai")
			}
			return "", 0, "", "", err
		}
		id, _ := res.LastInsertId()
		afterJSON := fmt.Sprintf(`{"code":%q,"name":%q,"building":%q,"room_type":%q,"capacity":%d,"status":"ACTIVE"}`,
			code, name, building, roomType, capacity)
		return "CREATE_MASTER_ROOM", id, "", afterJSON, nil
	}
	// COURSE
	name := strOf("name")
	if targetID.Valid {
		var beforeName, beforeStatus string
		if err := tx.QueryRow(`SELECT name, status FROM courses WHERE id = ?;`, targetID.Int64).Scan(&beforeName, &beforeStatus); err != nil {
			return "", 0, "", "", fmt.Errorf("target mata kuliah tidak ditemukan")
		}
		if _, err := tx.Exec(`UPDATE courses SET name = ? WHERE id = ?;`, name, targetID.Int64); err != nil {
			return "", 0, "", "", err
		}
		return "UPDATE_MASTER_COURSE", targetID.Int64,
			fmt.Sprintf(`{"name":%q,"status":%q}`, beforeName, beforeStatus),
			fmt.Sprintf(`{"name":%q,"status":%q}`, name, beforeStatus), nil
	}
	code := strOf("code")
	res, err := tx.Exec(`INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE');`, code, name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return "", 0, "", "", fmt.Errorf("kode mata kuliah sudah dipakai")
		}
		return "", 0, "", "", err
	}
	id, _ := res.LastInsertId()
	return "CREATE_MASTER_COURSE", id, "",
		fmt.Sprintf(`{"code":%q,"name":%q,"status":"ACTIVE"}`, code, name), nil
}

// writeMasterAuditTx mencatat audit perubahan master di dalam transaksi.
func writeMasterAuditTx(ctx context.Context, tx *sql.Tx, uid int64, raid *int64, action, entity string, entityID int64, beforeJSON, afterJSON, correlationID string) error {
	entry := audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		Action:        action,
		EntityType:    entity,
		EntityID:      &entityID,
		CorrelationID: correlationID,
	}
	if strings.TrimSpace(beforeJSON) != "" {
		entry.BeforeJSON = &beforeJSON
	}
	if strings.TrimSpace(afterJSON) != "" {
		entry.AfterJSON = &afterJSON
	}
	return audit.Write(ctx, tx, entry)
}
