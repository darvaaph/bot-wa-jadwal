package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/rooms"
)

func (s *Server) handleListRooms(w http.ResponseWriter, r *http.Request) {
	if s.roomsService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan ruangan belum tersedia"})
		return
	}
	status := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = "ACTIVE"
	}
	if status != "ACTIVE" && status != "INACTIVE" && status != "ALL" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter status tidak valid"})
		return
	}
	filter := status
	if status == "ALL" {
		filter = ""
	}
	items, err := s.roomsService.List(r.Context(), filter)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar ruangan"})
		return
	}
	if items == nil {
		items = []rooms.Room{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func roomActor(r *http.Request) rooms.Actor {
	principal, ok := principalFromRequest(r)
	if !ok {
		return rooms.Actor{}
	}
	return rooms.Actor{UserID: principal.UserID, RoleAssignmentID: principal.RoleAssignmentID}
}

func (s *Server) handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if s.roomsService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan ruangan belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat mengelola master ruangan"})
		return
	}
	var payload struct {
		Code     string  `json:"code"`
		Name     string  `json:"name"`
		Building *string `json:"building"`
		RoomType *string `json:"room_type"`
		Capacity *int    `json:"capacity"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	building, roomType := "", ""
	if payload.Building != nil {
		building = *payload.Building
	}
	if payload.RoomType != nil {
		roomType = *payload.RoomType
	}
	created, err := s.roomsService.Create(r.Context(), roomActor(r), payload.Code, payload.Name, building, roomType, payload.Capacity)
	if err != nil {
		if strings.Contains(err.Error(), "sudah digunakan") {
			s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Kode ruangan sudah digunakan"})
			return
		}
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Data ruangan tidak valid"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": created})
}

func (s *Server) handleUpdateRoom(w http.ResponseWriter, r *http.Request) {
	if s.roomsService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan ruangan belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat mengelola master ruangan"})
		return
	}
	roomID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || roomID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID ruangan tidak valid"})
		return
	}
	var payload struct {
		Name          *string `json:"name"`
		Building      *string `json:"building"`
		RoomType      *string `json:"room_type"`
		Capacity      *int    `json:"capacity"`
		ClearCapacity bool    `json:"clear_capacity"`
		Status        *string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	updated, err := s.roomsService.Update(r.Context(), roomActor(r), roomID, rooms.UpdateInput{
		Name: payload.Name, Building: payload.Building, RoomType: payload.RoomType,
		Capacity: payload.Capacity, ClearCapacity: payload.ClearCapacity, Status: payload.Status,
	})
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": updated})
}

func (s *Server) handleRoomAvailability(w http.ResponseWriter, r *http.Request) {
	if s.roomsService == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan ruangan belum tersedia"})
		return
	}
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	start := strings.TrimSpace(r.URL.Query().Get("start"))
	end := strings.TrimSpace(r.URL.Query().Get("end"))
	if date == "" || start == "" || end == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter date, start, end wajib diisi (date YYYY-MM-DD, start/end HH:MM)"})
		return
	}
	candidates, note, err := s.roomsService.Availability(r.Context(), date, start, end)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Parameter tanggal/waktu tidak valid"})
		return
	}
	if candidates == nil {
		candidates = []rooms.Candidate{}
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": map[string]any{
		"candidates": candidates, "note": note,
	}})
}

func (s *Server) handleRoomProposal(w http.ResponseWriter, r *http.Request) {
	db := s.auditDB()
	if db == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan audit belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() && principal.Role != "KM" {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya KM yang dapat mengusulkan koreksi ruangan"})
		return
	}
	var payload struct {
		RoomID *int64  `json:"room_id"`
		Code   *string `json:"code"`
		Name   *string `json:"name"`
		Note   string  `json:"note"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil || strings.TrimSpace(payload.Note) == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field note wajib diisi"})
		return
	}
	var classID any
	var actorUser, actorRA any
	actorType := "SYSTEM"
	if ok {
		if principal.ClassID != nil {
			classID = *principal.ClassID
		}
		actorUser = principal.UserID
		actorRA = principal.RoleAssignmentID
		actorType = "USER"
	}
	var entityID any
	if payload.RoomID != nil {
		entityID = *payload.RoomID
	}
	after := `{"note":"` + strings.ReplaceAll(strings.TrimSpace(payload.Note), `"`, ``) + `"}`
	corr := audit.NewCorrelationID()
	if _, err := db.ExecContext(r.Context(), `INSERT INTO audit_logs (class_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, reason, correlation_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'ROOM_PROPOSAL', 'ROOM', ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, actorUser, actorRA, actorType, entityID, after, strings.TrimSpace(payload.Note), corr); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menyimpan usulan"})
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]string{"status": "success", "message": "Usulan koreksi ruangan tercatat untuk System Admin"})
}
