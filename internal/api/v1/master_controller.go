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

// MasterController mengelola data master kampus: ruangan dan mata kuliah.
type MasterController struct {
	db *sql.DB
}

func NewMasterController(db *sql.DB) *MasterController {
	return &MasterController{db: db}
}

// masterActor mengambil identitas pelaku dari konteks autentikasi (route
// master tulis selalu di belakang RequireAuth + RequireRole).
func masterActor(r *http.Request) (uid int64, raid *int64, ok bool) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		return 0, nil, false
	}
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	return u.UserID, raid, true
}

// writeMasterAudit mencatat perubahan master (BE-006: FR-ROOM-003) pada database
// atau transaksi yang diberikan — pemanggil wajib commit agar keduanya atomik.
// before/after kosong berarti nil (khusus create: before kosong).
func writeMasterAudit(ctx context.Context, db audit.DBTX, r *http.Request, action, entity string, entityID int64, beforeJSON, afterJSON string) error {
	uid, raid, ok := masterActor(r)
	if !ok {
		return fmt.Errorf("konteks autentikasi hilang")
	}
	entry := audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		Action:        action,
		EntityType:    entity,
		EntityID:      &entityID,
		CorrelationID: fmt.Sprintf("master-%d-%d", entityID, time.Now().UnixNano()),
	}
	if strings.TrimSpace(beforeJSON) != "" {
		entry.BeforeJSON = &beforeJSON
	}
	if strings.TrimSpace(afterJSON) != "" {
		entry.AfterJSON = &afterJSON
	}
	return audit.Write(ctx, db, entry)
}

// GET /api/v1/master/rooms
func (c *MasterController) GetRooms(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	query := `SELECT id, code, COALESCE(name,''), COALESCE(building,''), COALESCE(room_type,''), COALESCE(capacity,0), status FROM rooms WHERE (1=1)`
	var args []any
	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY code ASC;"
	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat master ruangan")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var code, name, building, roomType, status string
		var capacity int
		if err := rows.Scan(&id, &code, &name, &building, &roomType, &capacity, &status); err == nil {
			out = append(out, map[string]any{
				"id": id, "code": code, "name": name, "building": building,
				"room_type": roomType, "capacity": capacity, "status": status,
			})
		}
	}
	common.WriteV1Success(w, http.StatusOK, out)
}

// POST /api/v1/master/rooms
func (c *MasterController) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	var req struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Building string `json:"building"`
		RoomType string `json:"room_type"`
		Capacity *int   `json:"capacity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode ruangan wajib diisi")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi ruangan")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO rooms (code, name, building, room_type, capacity, status) VALUES (?, ?, ?, ?, ?, 'ACTIVE');`,
		code, strings.TrimSpace(req.Name), strings.TrimSpace(req.Building), strings.TrimSpace(req.RoomType), req.Capacity)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode ruangan sudah dipakai atau tidak valid")
		return
	}
	id, _ := res.LastInsertId()
	afterJSON := fmt.Sprintf(`{"code":%q,"name":%q,"building":%q,"room_type":%q,"status":"ACTIVE"}`,
		code, strings.TrimSpace(req.Name), strings.TrimSpace(req.Building), strings.TrimSpace(req.RoomType))
	if err := writeMasterAudit(r.Context(), tx, r, "CREATE_MASTER_ROOM", "MASTER_ROOM", id, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan ruangan beserta auditnya")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan ruangan")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "code": code, "status": "ACTIVE"})
}

// PATCH /api/v1/master/rooms/{id}
func (c *MasterController) PatchRoom(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID ruangan tidak valid")
		return
	}
	var req struct {
		Name     *string `json:"name"`
		Building *string `json:"building"`
		RoomType *string `json:"room_type"`
		Capacity *int    `json:"capacity"`
		Status   *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	sets := []string{}
	args := []any{}
	if req.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.Building != nil {
		sets = append(sets, "building = ?")
		args = append(args, strings.TrimSpace(*req.Building))
	}
	if req.RoomType != nil {
		sets = append(sets, "room_type = ?")
		args = append(args, strings.TrimSpace(*req.RoomType))
	}
	if req.Capacity != nil {
		if *req.Capacity < 0 {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kapasitas tidak boleh negatif")
			return
		}
		sets = append(sets, "capacity = ?")
		args = append(args, *req.Capacity)
	}
	if req.Status != nil {
		st := strings.ToUpper(strings.TrimSpace(*req.Status))
		if st != "ACTIVE" && st != "INACTIVE" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Status harus ACTIVE atau INACTIVE")
			return
		}
		sets = append(sets, "status = ?")
		args = append(args, st)
	}
	if len(sets) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tidak ada field yang diubah")
		return
	}
	var before struct {
		name, building, roomType, status string
		capacity                         int
	}
	if err := c.db.QueryRow(`SELECT name, COALESCE(building,''), COALESCE(room_type,''), COALESCE(capacity,0), status FROM rooms WHERE id = ?;`, id).
		Scan(&before.name, &before.building, &before.roomType, &before.capacity, &before.status); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Ruangan tidak ditemukan")
		return
	}
	args = append(args, id)
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi ruangan")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE rooms SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah ruangan")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Ruangan tidak ditemukan")
		return
	}
	after := before
	if req.Name != nil {
		after.name = strings.TrimSpace(*req.Name)
	}
	if req.Building != nil {
		after.building = strings.TrimSpace(*req.Building)
	}
	if req.RoomType != nil {
		after.roomType = strings.TrimSpace(*req.RoomType)
	}
	if req.Capacity != nil {
		after.capacity = *req.Capacity
	}
	if req.Status != nil {
		after.status = strings.ToUpper(strings.TrimSpace(*req.Status))
	}
	beforeJSON := fmt.Sprintf(`{"name":%q,"building":%q,"room_type":%q,"capacity":%d,"status":%q}`,
		before.name, before.building, before.roomType, before.capacity, before.status)
	afterJSON := fmt.Sprintf(`{"name":%q,"building":%q,"room_type":%q,"capacity":%d,"status":%q}`,
		after.name, after.building, after.roomType, after.capacity, after.status)
	if err := writeMasterAudit(r.Context(), tx, r, "UPDATE_MASTER_ROOM", "MASTER_ROOM", id, beforeJSON, afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan ruangan beserta auditnya")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan ruangan")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": id})
}

// GET /api/v1/master/courses
func (c *MasterController) GetCourses(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	query := `SELECT id, code, name, status FROM courses WHERE (1=1)`
	var args []any
	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY code ASC;"
	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat master mata kuliah")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var code, name, status string
		if err := rows.Scan(&id, &code, &name, &status); err == nil {
			out = append(out, map[string]any{"id": id, "code": code, "name": name, "status": status})
		}
	}
	common.WriteV1Success(w, http.StatusOK, out)
}

// POST /api/v1/master/courses
func (c *MasterController) CreateCourse(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode dan nama mata kuliah wajib diisi")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi mata kuliah")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE');`, code, name)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode mata kuliah sudah dipakai atau tidak valid")
		return
	}
	id, _ := res.LastInsertId()
	afterJSON := fmt.Sprintf(`{"code":%q,"name":%q,"status":"ACTIVE"}`, code, name)
	if err := writeMasterAudit(r.Context(), tx, r, "CREATE_MASTER_COURSE", "MASTER_COURSE", id, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan mata kuliah beserta auditnya")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan mata kuliah")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "code": code, "status": "ACTIVE"})
}

// PATCH /api/v1/master/courses/{id}
func (c *MasterController) PatchCourse(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID mata kuliah tidak valid")
		return
	}
	var req struct {
		Name   *string `json:"name"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	sets := []string{}
	args := []any{}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nama mata kuliah wajib diisi")
			return
		}
		sets = append(sets, "name = ?")
		args = append(args, strings.TrimSpace(*req.Name))
	}
	if req.Status != nil {
		st := strings.ToUpper(strings.TrimSpace(*req.Status))
		if st != "ACTIVE" && st != "INACTIVE" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Status harus ACTIVE atau INACTIVE")
			return
		}
		sets = append(sets, "status = ?")
		args = append(args, st)
	}
	if len(sets) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Tidak ada field yang diubah")
		return
	}
	var before struct {
		name, status string
	}
	if err := c.db.QueryRow(`SELECT name, status FROM courses WHERE id = ?;`, id).
		Scan(&before.name, &before.status); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Mata kuliah tidak ditemukan")
		return
	}
	args = append(args, id)
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi mata kuliah")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE courses SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah mata kuliah")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Mata kuliah tidak ditemukan")
		return
	}
	after := before
	if req.Name != nil {
		after.name = strings.TrimSpace(*req.Name)
	}
	if req.Status != nil {
		after.status = strings.ToUpper(strings.TrimSpace(*req.Status))
	}
	beforeJSON := fmt.Sprintf(`{"name":%q,"status":%q}`, before.name, before.status)
	afterJSON := fmt.Sprintf(`{"name":%q,"status":%q}`, after.name, after.status)
	if err := writeMasterAudit(r.Context(), tx, r, "UPDATE_MASTER_COURSE", "MASTER_COURSE", id, beforeJSON, afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan mata kuliah beserta auditnya")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan mata kuliah")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": id})
}
