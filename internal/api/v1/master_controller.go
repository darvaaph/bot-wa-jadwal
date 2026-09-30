package v1

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/api/common"
)

// MasterController mengelola data master kampus: ruangan dan mata kuliah.
type MasterController struct {
	db *sql.DB
}

func NewMasterController(db *sql.DB) *MasterController {
	return &MasterController{db: db}
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
	res, err := c.db.Exec(`INSERT INTO rooms (code, name, building, room_type, capacity, status) VALUES (?, ?, ?, ?, ?, 'ACTIVE');`,
		code, strings.TrimSpace(req.Name), strings.TrimSpace(req.Building), strings.TrimSpace(req.RoomType), req.Capacity)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode ruangan sudah dipakai atau tidak valid")
		return
	}
	id, _ := res.LastInsertId()
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
	args = append(args, id)
	res, err := c.db.Exec(`UPDATE rooms SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah ruangan")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Ruangan tidak ditemukan")
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
	res, err := c.db.Exec(`INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE');`, code, name)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode mata kuliah sudah dipakai atau tidak valid")
		return
	}
	id, _ := res.LastInsertId()
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
	args = append(args, id)
	res, err := c.db.Exec(`UPDATE courses SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah mata kuliah")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Mata kuliah tidak ditemukan")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": id})
}
