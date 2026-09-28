package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RoomCandidateItem merepresentasikan ruangan yang tersedia untuk digunakan
type RoomCandidateItem struct {
	ID       int64  `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	RoomType string `json:"room_type"`
}

// handleGetRoomCandidates menangani GET /api/v1/rooms/candidates
// Mencari ruangan aktif yang tidak bentrok dengan jadwal aktif pada rentang waktu yang diminta.
func (s *Server) handleGetRoomCandidates(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	startsAtStr := strings.TrimSpace(r.URL.Query().Get("starts_at"))
	endsAtStr := strings.TrimSpace(r.URL.Query().Get("ends_at"))

	if startsAtStr == "" || endsAtStr == "" {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Parameter starts_at dan ends_at wajib disertakan (RFC3339)")
		return
	}

	startsAt, err1 := time.Parse(time.RFC3339, startsAtStr)
	endsAt, err2 := time.Parse(time.RFC3339, endsAtStr)
	if err1 != nil || err2 != nil || !endsAt.After(startsAt) {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Format waktu tidak valid atau ends_at mendahului starts_at")
		return
	}

	excludeEventIDStr := strings.TrimSpace(r.URL.Query().Get("exclude_event_id"))
	excludeEventID, _ := strconv.ParseInt(excludeEventIDStr, 10, 64)

	// Cari ruangan yang TIDAK sedang digunakan oleh event PUBLISHED pada rentang waktu tersebut
	query := `
		SELECT r.id, r.code, r.name, r.capacity, r.room_type
		FROM rooms r
		WHERE r.status = 'ACTIVE'
		  AND r.id NOT IN (
		      SELECT te.room_id
		      FROM teaching_events te
		      WHERE te.room_id IS NOT NULL
		        AND te.lifecycle_status = 'PUBLISHED'
		        AND te.id != ?
		        AND te.starts_at < ? AND te.ends_at > ?
		  )
		ORDER BY r.code;
	`

	rows, err := s.v1DB.Query(query, excludeEventID, endsAt.Format(time.RFC3339), startsAt.Format(time.RFC3339))
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal mencari kandidat ruangan: %v", err))
		return
	}
	defer rows.Close()

	candidates := []RoomCandidateItem{}
	for rows.Next() {
		var item RoomCandidateItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Capacity, &item.RoomType); err == nil {
			candidates = append(candidates, item)
		}
	}

	s.writeV1Success(w, http.StatusOK, candidates)
}

// RoomConfirmationRequest payload pengajuan/konfirmasi ruangan ke pihak TU
type RoomConfirmationRequest struct {
	RoomID             int64   `json:"room_id"`
	ConfirmationStatus string  `json:"confirmation_status"` // PENDING, CONFIRMED, REJECTED
	ExternalContact    *string `json:"external_contact,omitempty"`
	Note               *string `json:"note,omitempty"`
}

// handleCreateRoomConfirmation menangani POST /api/v1/teaching-events/{id}/room-confirmations
func (s *Server) handleCreateRoomConfirmation(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := strconv.ParseInt(eventIDStr, 10, 64)
	if err != nil || eventID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID kejadian jadwal tidak valid")
		return
	}

	var req RoomConfirmationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Payload JSON tidak valid")
		return
	}

	status := strings.ToUpper(strings.TrimSpace(req.ConfirmationStatus))
	if status != "PENDING" && status != "CONFIRMED" && status != "REJECTED" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Status konfirmasi harus PENDING, CONFIRMED, atau REJECTED")
		return
	}

	var roomExists int
	_ = s.v1DB.QueryRow(`SELECT COUNT(*) FROM rooms WHERE id = ?;`, req.RoomID).Scan(&roomExists)
	if roomExists == 0 {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Ruangan tidak ditemukan")
		return
	}

	var confirmedAt any
	if status == "CONFIRMED" {
		confirmedAt = time.Now().UTC().Format(time.RFC3339)
	}

	res, err := s.v1DB.Exec(`
		INSERT INTO room_confirmations (
			teaching_event_id, room_id, confirmation_status, external_contact, note, recorded_by_user_id, recorded_at, confirmed_at
		) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?);
	`, eventID, req.RoomID, status, req.ExternalContact, req.Note, u.UserID, confirmedAt)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan konfirmasi ruangan: %v", err))
		return
	}

	confID, _ := res.LastInsertId()

	// Update room_id pada teaching_event jika dikonfirmasi
	if status == "CONFIRMED" {
		_, _ = s.v1DB.Exec(`UPDATE teaching_events SET room_id = ? WHERE id = ?;`, req.RoomID, eventID)
	}

	// Catat audit_logs
	_, _ = s.v1DB.Exec(`
		INSERT INTO audit_logs (class_id, actor_user_id, actor_role_assignment_id, action, entity_type, entity_id, after_json)
		VALUES (?, ?, ?, 'CREATE_ROOM_CONFIRMATION', 'ROOM_CONFIRMATION', ?, ?);
	`, u.ActiveClassID, u.UserID, u.ActiveAssignmentID, confID,
		fmt.Sprintf(`{"event_id":%d,"room_id":%d,"status":%q}`, eventID, req.RoomID, status),
	)

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"id":                  confID,
		"teaching_event_id":   eventID,
		"room_id":             req.RoomID,
		"confirmation_status": status,
	})
}
