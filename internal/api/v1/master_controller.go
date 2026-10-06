package v1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/util"
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
	var eid *int64
	if entityID > 0 {
		eid = &entityID
	}
	entry := audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		Action:        action,
		EntityType:    entity,
		EntityID:      eid,
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

// RoomInput mewakili format data kode, nama, gedung, tipe, dan kapasitas ruangan
type RoomInput struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Building string `json:"building"`
	RoomType string `json:"room_type"`
	Capacity *int   `json:"capacity"`
}

// POST /api/v1/master/rooms
func (c *MasterController) CreateRoom(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}

	// 1. Opsi A: bulk rooms object
	var bulkCheck struct {
		Rooms []RoomInput `json:"rooms"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Rooms) > 0 {
		c.bulkInsertRoomsInternal(w, r, bulkCheck.Rooms)
		return
	}

	// 2. Opsi B: bulk rooms array [...]
	var arrayCheck []RoomInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertRoomsInternal(w, r, arrayCheck)
		return
	}

	// 3. Default: single room
	var req RoomInput
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
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

// POST /api/v1/master/rooms/bulk
func (c *MasterController) BulkCreateRooms(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}
	var bulkCheck struct {
		Rooms []RoomInput `json:"rooms"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Rooms) > 0 {
		c.bulkInsertRoomsInternal(w, r, bulkCheck.Rooms)
		return
	}
	var arrayCheck []RoomInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertRoomsInternal(w, r, arrayCheck)
		return
	}
	common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar ruangan kosong atau tidak valid")
}

func (c *MasterController) bulkInsertRoomsInternal(w http.ResponseWriter, r *http.Request, items []RoomInput) {
	if len(items) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar ruangan kosong")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi impor ruangan")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO rooms (code, name, building, room_type, capacity, status)
		VALUES (?, ?, ?, ?, COALESCE(?, 32), 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET
			name = CASE WHEN excluded.name != '' THEN excluded.name ELSE rooms.name END,
			building = CASE WHEN excluded.building != '' THEN excluded.building ELSE rooms.building END,
			room_type = CASE WHEN excluded.room_type != '' THEN excluded.room_type ELSE rooms.room_type END,
			capacity = CASE WHEN excluded.capacity > 0 THEN excluded.capacity ELSE rooms.capacity END;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah impor ruangan")
		return
	}
	defer stmt.Close()

	inserted := 0
	skipped := 0
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		if code == "" {
			skipped++
			continue
		}
		name := strings.TrimSpace(item.Name)
		building := strings.TrimSpace(item.Building)
		roomType := strings.TrimSpace(item.RoomType)
		cap := 32
		if item.Capacity != nil && *item.Capacity > 0 {
			cap = *item.Capacity
		}
		if _, err := stmt.Exec(code, name, building, roomType, cap); err == nil {
			inserted++
		} else {
			skipped++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"BULK_IMPORT_ROOMS","total_received":%d,"total_imported":%d,"total_skipped":%d}`, len(items), inserted, skipped)
	if err := writeMasterAudit(r.Context(), tx, r, "BULK_CREATE_MASTER_ROOMS", "MASTER_ROOM", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit impor ruangan")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil impor ruangan")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_received": len(items),
		"total_imported": inserted,
		"total_skipped":  skipped,
		"message":        fmt.Sprintf("Berhasil mengimpor %d ruangan", inserted),
	})
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

// CourseInput mewakili format data kode dan nama mata kuliah
type CourseInput struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// POST /api/v1/master/courses
func (c *MasterController) CreateCourse(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}

	// 1. Opsi A: sync_jadwal flag
	var syncCheck struct {
		SyncJadwal bool `json:"sync_jadwal"`
	}
	if err := json.Unmarshal(bodyBytes, &syncCheck); err == nil && syncCheck.SyncJadwal {
		c.syncCoursesInternal(w, r)
		return
	}

	// 2. Opsi B: bulk courses object
	var bulkCheck struct {
		Courses []CourseInput `json:"courses"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Courses) > 0 {
		c.bulkInsertCoursesInternal(w, r, bulkCheck.Courses)
		return
	}

	// 3. Opsi B: bulk courses array [...]
	var arrayCheck []CourseInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertCoursesInternal(w, r, arrayCheck)
		return
	}

	// 4. Default: single course
	var req CourseInput
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
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

// POST /api/v1/master/courses/sync-jadwal
func (c *MasterController) SyncCoursesFromJadwal(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	c.syncCoursesInternal(w, r)
}

// POST /api/v1/master/courses/bulk
func (c *MasterController) BulkCreateCourses(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}
	var bulkCheck struct {
		Courses []CourseInput `json:"courses"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Courses) > 0 {
		c.bulkInsertCoursesInternal(w, r, bulkCheck.Courses)
		return
	}
	var arrayCheck []CourseInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertCoursesInternal(w, r, arrayCheck)
		return
	}
	common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar mata kuliah kosong atau tidak valid")
}

func (c *MasterController) syncCoursesInternal(w http.ResponseWriter, r *http.Request) {
	dir := util.FindDataDir("data/jadwal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "FS_ERROR", "Direktori data jadwal tidak dapat diakses")
		return
	}

	uniqueCourses := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var rf struct {
			MataKuliah map[string]string `json:"mata_kuliah"`
			Jadwal     []struct {
				Kode   string `json:"kode"`
				Matkul string `json:"matkul"`
			} `json:"jadwal"`
		}
		if err := json.Unmarshal(data, &rf); err != nil {
			continue
		}
		for k, v := range rf.MataKuliah {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" {
				uniqueCourses[k] = v
			}
		}
		for _, item := range rf.Jadwal {
			k := strings.TrimSpace(item.Kode)
			v := strings.TrimSpace(item.Matkul)
			if k != "" && v != "" {
				if _, exists := uniqueCourses[k]; !exists {
					uniqueCourses[k] = v
				}
			}
		}
	}

	if len(uniqueCourses) == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tidak ada data mata kuliah yang ditemukan pada berkas jadwal")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi sinkronisasi")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah sinkronisasi")
		return
	}
	defer stmt.Close()

	synced := 0
	for code, name := range uniqueCourses {
		if _, err := stmt.Exec(code, name); err == nil {
			synced++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"SYNC_JADWAL","total_found":%d,"total_synced":%d}`, len(uniqueCourses), synced)
	if err := writeMasterAudit(r.Context(), tx, r, "SYNC_MASTER_COURSES", "MASTER_COURSE", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit sinkronisasi")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil sinkronisasi")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_found":  len(uniqueCourses),
		"total_synced": synced,
		"message":      fmt.Sprintf("Berhasil menyinkronkan %d mata kuliah dari jadwal kurikulum", synced),
	})
}

func (c *MasterController) bulkInsertCoursesInternal(w http.ResponseWriter, r *http.Request, items []CourseInput) {
	if len(items) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar mata kuliah kosong")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi impor")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO courses (code, name, status) VALUES (?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, status = excluded.status;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah impor")
		return
	}
	defer stmt.Close()

	inserted := 0
	skipped := 0
	for _, item := range items {
		code := strings.TrimSpace(item.Code)
		name := strings.TrimSpace(item.Name)
		status := strings.ToUpper(strings.TrimSpace(item.Status))
		if status != "INACTIVE" {
			status = "ACTIVE"
		}
		if code == "" || name == "" {
			skipped++
			continue
		}
		if _, err := stmt.Exec(code, name, status); err == nil {
			inserted++
		} else {
			skipped++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"BULK_IMPORT","total_received":%d,"total_imported":%d,"total_skipped":%d}`, len(items), inserted, skipped)
	if err := writeMasterAudit(r.Context(), tx, r, "BULK_CREATE_MASTER_COURSES", "MASTER_COURSE", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit impor")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil impor")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_received": len(items),
		"total_imported": inserted,
		"total_skipped":  skipped,
		"message":        fmt.Sprintf("Berhasil mengimpor %d mata kuliah", inserted),
	})
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

// LecturerInput mewakili format data kode dan nama dosen
type LecturerInput struct {
	Code     string `json:"code"`
	FullName string `json:"full_name"`
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"`
}

// GET /api/v1/master/lecturers
func (c *MasterController) GetLecturers(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	query := `SELECT id, code, full_name, status FROM lecturers WHERE (1=1)`
	var args []any
	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY code ASC;"
	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat master dosen")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id int64
		var code, fullName, status string
		if err := rows.Scan(&id, &code, &fullName, &status); err == nil {
			out = append(out, map[string]any{
				"id": id, "code": code, "full_name": fullName, "name": fullName, "status": status,
			})
		}
	}
	common.WriteV1Success(w, http.StatusOK, out)
}

// POST /api/v1/master/lecturers
func (c *MasterController) CreateLecturer(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}

	var syncCheck struct {
		SyncJadwal bool `json:"sync_jadwal"`
	}
	if err := json.Unmarshal(bodyBytes, &syncCheck); err == nil && syncCheck.SyncJadwal {
		c.syncLecturersInternal(w, r)
		return
	}

	var bulkCheck struct {
		Lecturers []LecturerInput `json:"lecturers"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Lecturers) > 0 {
		c.bulkInsertLecturersInternal(w, r, bulkCheck.Lecturers)
		return
	}

	var arrayCheck []LecturerInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertLecturersInternal(w, r, arrayCheck)
		return
	}

	var req LecturerInput
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	fullName := strings.TrimSpace(req.FullName)
	if code == "" || fullName == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode inisial dan nama dosen wajib diisi")
		return
	}
	status := "ACTIVE"
	if strings.ToUpper(strings.TrimSpace(req.Status)) == "INACTIVE" {
		status = "INACTIVE"
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi dosen")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, ?);`, code, fullName, status)
	if err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Kode dosen sudah dipakai atau tidak valid")
		return
	}
	id, _ := res.LastInsertId()
	afterJSON := fmt.Sprintf(`{"code":%q,"full_name":%q,"status":%q}`, code, fullName, status)
	if err := writeMasterAudit(r.Context(), tx, r, "CREATE_MASTER_LECTURER", "MASTER_LECTURER", id, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit dosen")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan dosen")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "code": code, "status": status})
}

// PATCH /api/v1/master/lecturers/{id}
func (c *MasterController) PatchLecturer(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID dosen tidak valid")
		return
	}
	var req struct {
		FullName *string `json:"full_name"`
		Name     *string `json:"name"`
		Status   *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	sets := []string{}
	args := []any{}
	fn := req.FullName
	if fn == nil {
		fn = req.Name
	}
	if fn != nil {
		val := strings.TrimSpace(*fn)
		if val == "" {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Nama dosen wajib diisi")
			return
		}
		sets = append(sets, "full_name = ?")
		args = append(args, val)
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
		fullName, status string
	}
	if err := c.db.QueryRow(`SELECT full_name, status FROM lecturers WHERE id = ?;`, id).
		Scan(&before.fullName, &before.status); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Dosen tidak ditemukan")
		return
	}
	args = append(args, id)
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi dosen")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE lecturers SET `+strings.Join(sets, ", ")+` WHERE id = ?;`, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengubah dosen")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Dosen tidak ditemukan")
		return
	}
	after := before
	if fn != nil {
		after.fullName = strings.TrimSpace(*fn)
	}
	if req.Status != nil {
		after.status = strings.ToUpper(strings.TrimSpace(*req.Status))
	}
	beforeJSON := fmt.Sprintf(`{"full_name":%q,"status":%q}`, before.fullName, before.status)
	afterJSON := fmt.Sprintf(`{"full_name":%q,"status":%q}`, after.fullName, after.status)
	if err := writeMasterAudit(r.Context(), tx, r, "UPDATE_MASTER_LECTURER", "MASTER_LECTURER", id, beforeJSON, afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit dosen")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan perubahan dosen")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": id})
}

// POST /api/v1/master/lecturers/sync-jadwal
func (c *MasterController) SyncLecturersFromJadwal(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	c.syncLecturersInternal(w, r)
}

// POST /api/v1/master/lecturers/bulk
func (c *MasterController) BulkCreateLecturers(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Gagal membaca payload request")
		return
	}
	var bulkCheck struct {
		Lecturers []LecturerInput `json:"lecturers"`
	}
	if err := json.Unmarshal(bodyBytes, &bulkCheck); err == nil && len(bulkCheck.Lecturers) > 0 {
		c.bulkInsertLecturersInternal(w, r, bulkCheck.Lecturers)
		return
	}
	var arrayCheck []LecturerInput
	if err := json.Unmarshal(bodyBytes, &arrayCheck); err == nil && len(arrayCheck) > 0 {
		c.bulkInsertLecturersInternal(w, r, arrayCheck)
		return
	}
	common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar dosen kosong atau tidak valid")
}

func (c *MasterController) syncLecturersInternal(w http.ResponseWriter, r *http.Request) {
	dir := util.FindDataDir("data/jadwal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "FS_ERROR", "Direktori data jadwal tidak dapat diakses")
		return
	}

	uniqueLecturers := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var rf struct {
			Dosen  map[string]string `json:"dosen"`
			Jadwal []struct {
				InisialDosen string `json:"inisial_dosen"`
				Dosen        string `json:"dosen"`
			} `json:"jadwal"`
		}
		if err := json.Unmarshal(data, &rf); err != nil {
			continue
		}
		for k, v := range rf.Dosen {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" && !strings.Contains(k, "+") {
				uniqueLecturers[k] = v
			}
		}
		for _, item := range rf.Jadwal {
			k := strings.TrimSpace(item.InisialDosen)
			v := strings.TrimSpace(item.Dosen)
			if k != "" && v != "" && !strings.Contains(k, "+") {
				if _, exists := uniqueLecturers[k]; !exists {
					uniqueLecturers[k] = v
				}
			}
		}
	}

	if len(uniqueLecturers) == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tidak ada data dosen yang ditemukan pada berkas jadwal")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi sinkronisasi")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah sinkronisasi dosen")
		return
	}
	defer stmt.Close()

	synced := 0
	for code, name := range uniqueLecturers {
		if _, err := stmt.Exec(code, name); err == nil {
			synced++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"SYNC_JADWAL_LECTURERS","total_found":%d,"total_synced":%d}`, len(uniqueLecturers), synced)
	if err := writeMasterAudit(r.Context(), tx, r, "SYNC_MASTER_LECTURERS", "MASTER_LECTURER", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit sinkronisasi dosen")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil sinkronisasi dosen")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_found":  len(uniqueLecturers),
		"total_synced": synced,
		"message":      fmt.Sprintf("Berhasil menyinkronkan %d dosen dari jadwal kurikulum", synced),
	})
}

func (c *MasterController) bulkInsertLecturersInternal(w http.ResponseWriter, r *http.Request, items []LecturerInput) {
	if len(items) == 0 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Daftar dosen kosong")
		return
	}
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi impor dosen")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name, status = excluded.status;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah impor dosen")
		return
	}
	defer stmt.Close()

	inserted := 0
	skipped := 0
	for _, item := range items {
		code := strings.ToUpper(strings.TrimSpace(item.Code))
		name := strings.TrimSpace(item.FullName)
		if name == "" && strings.TrimSpace(item.Name) != "" {
			name = strings.TrimSpace(item.Name)
		}
		if code == "" || name == "" {
			skipped++
			continue
		}
		status := "ACTIVE"
		if strings.ToUpper(strings.TrimSpace(item.Status)) == "INACTIVE" {
			status = "INACTIVE"
		}
		if _, err := stmt.Exec(code, name, status); err == nil {
			inserted++
		} else {
			skipped++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"BULK_IMPORT_LECTURERS","total_received":%d,"total_imported":%d,"total_skipped":%d}`, len(items), inserted, skipped)
	if err := writeMasterAudit(r.Context(), tx, r, "BULK_CREATE_MASTER_LECTURERS", "MASTER_LECTURER", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit impor dosen")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil impor dosen")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_received": len(items),
		"total_imported": inserted,
		"total_skipped":  skipped,
		"message":        fmt.Sprintf("Berhasil mengimpor %d dosen", inserted),
	})
}

// POST /api/v1/master/rooms/sync-jadwal
func (c *MasterController) SyncRoomsFromJadwal(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	dir := util.FindDataDir("data/jadwal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "FS_ERROR", "Direktori data jadwal tidak dapat diakses")
		return
	}

	uniqueRooms := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var rf struct {
			Jadwal []struct {
				Ruang string `json:"ruang"`
			} `json:"jadwal"`
		}
		if err := json.Unmarshal(data, &rf); err != nil {
			continue
		}
		for _, item := range rf.Jadwal {
			rName := strings.TrimSpace(item.Ruang)
			if rName != "" {
				uniqueRooms[rName] = true
			}
		}
	}

	if len(uniqueRooms) == 0 {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tidak ada data ruangan yang ditemukan pada berkas jadwal")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi sinkronisasi ruangan")
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO rooms (code, name, building, room_type, capacity, status)
		VALUES (?, ?, ?, ?, 32, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, building = excluded.building, room_type = excluded.room_type;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah sinkronisasi ruangan")
		return
	}
	defer stmt.Close()

	synced := 0
	for rawRoom := range uniqueRooms {
		code := rawRoom
		building := "Kampus"
		roomType := "TEORI"
		if strings.HasPrefix(strings.ToUpper(code), "D") {
			building = "Gedung D"
		} else if strings.HasPrefix(strings.ToUpper(code), "H") {
			building = "Gedung H"
		}
		if strings.Contains(strings.ToLower(code), "lab") {
			roomType = "LAB"
		}
		name := code
		if parts := strings.Split(code, "-"); len(parts) >= 2 {
			name = strings.TrimSpace(parts[1])
		}

		if _, err := stmt.Exec(code, name, building, roomType); err == nil {
			synced++
		}
	}

	afterJSON := fmt.Sprintf(`{"action":"SYNC_JADWAL_ROOMS","total_found":%d,"total_synced":%d}`, len(uniqueRooms), synced)
	if err := writeMasterAudit(r.Context(), tx, r, "SYNC_MASTER_ROOMS", "MASTER_ROOM", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit sinkronisasi ruangan")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan hasil sinkronisasi ruangan")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"total_found":  len(uniqueRooms),
		"total_synced": synced,
		"message":      fmt.Sprintf("Berhasil menyinkronkan %d ruangan dari jadwal kurikulum", synced),
	})
}

// POST /api/v1/master/sync-all
func (c *MasterController) SyncAllMasterFromJadwal(w http.ResponseWriter, r *http.Request) {
	if c.db == nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	dir := util.FindDataDir("data/jadwal")
	entries, err := os.ReadDir(dir)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "FS_ERROR", "Direktori data jadwal tidak dapat diakses")
		return
	}

	uniqueCourses := make(map[string]string)
	uniqueLecturers := make(map[string]string)
	uniqueRooms := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			continue
		}
		filePath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		var rf struct {
			MataKuliah map[string]string `json:"mata_kuliah"`
			Dosen      map[string]string `json:"dosen"`
			Jadwal     []struct {
				Kode         string `json:"kode"`
				Matkul       string `json:"matkul"`
				InisialDosen string `json:"inisial_dosen"`
				Dosen        string `json:"dosen"`
				Ruang        string `json:"ruang"`
			} `json:"jadwal"`
		}
		if err := json.Unmarshal(data, &rf); err != nil {
			continue
		}

		for k, v := range rf.MataKuliah {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" {
				uniqueCourses[k] = v
			}
		}
		for k, v := range rf.Dosen {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k != "" && v != "" && !strings.Contains(k, "+") {
				uniqueLecturers[k] = v
			}
		}
		for _, item := range rf.Jadwal {
			mkCode := strings.TrimSpace(item.Kode)
			mkName := strings.TrimSpace(item.Matkul)
			if mkCode != "" && mkName != "" {
				if _, exists := uniqueCourses[mkCode]; !exists {
					uniqueCourses[mkCode] = mkName
				}
			}
			lecCode := strings.TrimSpace(item.InisialDosen)
			lecName := strings.TrimSpace(item.Dosen)
			if lecCode != "" && lecName != "" && !strings.Contains(lecCode, "+") {
				if _, exists := uniqueLecturers[lecCode]; !exists {
					uniqueLecturers[lecCode] = lecName
				}
			}
			rName := strings.TrimSpace(item.Ruang)
			if rName != "" {
				uniqueRooms[rName] = true
			}
		}
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi sinkronisasi terpadu")
		return
	}
	defer tx.Rollback()

	// 1. Upsert Courses
	stmtCourses, err := tx.Prepare(`INSERT INTO courses (code, name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah courses")
		return
	}
	defer stmtCourses.Close()
	coursesSynced := 0
	for code, name := range uniqueCourses {
		if _, err := stmtCourses.Exec(code, name); err == nil {
			coursesSynced++
		}
	}

	// 2. Upsert Lecturers
	stmtLecturers, err := tx.Prepare(`INSERT INTO lecturers (code, full_name, status) VALUES (?, ?, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET full_name = excluded.full_name;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah lecturers")
		return
	}
	defer stmtLecturers.Close()
	lecturersSynced := 0
	for code, name := range uniqueLecturers {
		if _, err := stmtLecturers.Exec(code, name); err == nil {
			lecturersSynced++
		}
	}

	// 3. Upsert Rooms
	stmtRooms, err := tx.Prepare(`INSERT INTO rooms (code, name, building, room_type, capacity, status)
		VALUES (?, ?, ?, ?, 32, 'ACTIVE')
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, building = excluded.building, room_type = excluded.room_type;`)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyiapkan perintah rooms")
		return
	}
	defer stmtRooms.Close()
	roomsSynced := 0
	for rawRoom := range uniqueRooms {
		code := rawRoom
		building := "Kampus"
		roomType := "TEORI"
		if strings.HasPrefix(strings.ToUpper(code), "D") {
			building = "Gedung D"
		} else if strings.HasPrefix(strings.ToUpper(code), "H") {
			building = "Gedung H"
		}
		if strings.Contains(strings.ToLower(code), "lab") {
			roomType = "LAB"
		}
		name := code
		if parts := strings.Split(code, "-"); len(parts) >= 2 {
			name = strings.TrimSpace(parts[1])
		}
		if _, err := stmtRooms.Exec(code, name, building, roomType); err == nil {
			roomsSynced++
		}
	}

	totalSynced := coursesSynced + roomsSynced + lecturersSynced
	afterJSON := fmt.Sprintf(`{"action":"SYNC_ALL_MASTER","courses":%d,"rooms":%d,"lecturers":%d,"total":%d}`,
		coursesSynced, roomsSynced, lecturersSynced, totalSynced)
	if err := writeMasterAudit(r.Context(), tx, r, "SYNC_ALL_MASTER", "MASTER_ALL", 0, "", afterJSON); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit sinkronisasi terpadu")
		return
	}

	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan sinkronisasi terpadu")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"courses_synced":   coursesSynced,
		"rooms_synced":     roomsSynced,
		"lecturers_synced": lecturersSynced,
		"total_synced":     totalSynced,
		"message":          fmt.Sprintf("Berhasil menyinkronkan seluruh master kampus: %d mata kuliah, %d ruangan, dan %d dosen", coursesSynced, roomsSynced, lecturersSynced),
	})
}
