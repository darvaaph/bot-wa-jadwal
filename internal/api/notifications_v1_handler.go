package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/audit"
)

// NotificationResponseItem merepresentasikan pesan notifikasi dalam antrean siaran WhatsApp
type NotificationResponseItem struct {
	ID           int64   `json:"id"`
	ClassID      int64   `json:"class_id"`
	EventType    string  `json:"event_type"`
	EntityType   *string `json:"entity_type,omitempty"`
	EntityID     *int64  `json:"entity_id,omitempty"`
	Status       string  `json:"status"` // PENDING, PROCESSING, SENT, FAILED, CANCELLED
	PayloadJSON  string  `json:"payload_json"`
	ScheduledAt  *string `json:"scheduled_at,omitempty"`
	SentAt       *string `json:"sent_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
	AttemptCount int     `json:"attempt_count"`
}

// handleGetNotifications menangani GET /api/v1/notifications
func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
		limit = l
	}

	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
		offset = o
	}

	query := `
		SELECT nm.id, nm.class_id, nm.event_type, nm.entity_type, nm.entity_id,
		       nm.status, nm.payload_json, nm.scheduled_at, nm.sent_at, nm.created_at,
		       (SELECT COUNT(*) FROM notification_attempts na WHERE na.notification_message_id = nm.id) AS attempts
		FROM notification_messages nm
		WHERE (1=1)
	`
	var args []any

	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveClassID.Valid {
		query += " AND nm.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	}

	if statusFilter != "" {
		query += " AND nm.status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY nm.created_at DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := s.v1DB.Query(query, args...)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal memuat antrean notifikasi: %v", err))
		return
	}
	defer rows.Close()

	notifications := []NotificationResponseItem{}
	for rows.Next() {
		var (
			id          int64
			classID     int64
			eventType   string
			entityType  sql.NullString
			entityID    sql.NullInt64
			status      string
			payloadJSON string
			scheduledAt dbTimestamp
			sentAt      dbTimestamp
			createdAt   dbTimestamp
			attempts    int
		)

		if err := rows.Scan(&id, &classID, &eventType, &entityType, &entityID, &status, &payloadJSON, &scheduledAt, &sentAt, &createdAt, &attempts); err == nil {
			var schedStr *string
			if scheduledAt.Valid {
				formatted := scheduledAt.Time.Format(time.RFC3339)
				schedStr = &formatted
			}
			var sentStr *string
			if sentAt.Valid {
				formatted := sentAt.Time.Format(time.RFC3339)
				sentStr = &formatted
			}
			var eType *string
			if entityType.Valid {
				eType = &entityType.String
			}
			var eID *int64
			if entityID.Valid {
				eID = &entityID.Int64
			}

			notifications = append(notifications, NotificationResponseItem{
				ID:           id,
				ClassID:      classID,
				EventType:    eventType,
				EntityType:   eType,
				EntityID:     eID,
				Status:       status,
				PayloadJSON:  payloadJSON,
				ScheduledAt:  schedStr,
				SentAt:       sentStr,
				CreatedAt:    createdAt.Time.Format(time.RFC3339),
				AttemptCount: attempts,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, notifications)
}

// handleRetryNotification menangani POST /api/v1/notifications/{id}/retry
func (s *Server) handleRetryNotification(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	notifIDStr := r.PathValue("id")
	notifID, err := strconv.ParseInt(notifIDStr, 10, 64)
	if err != nil || notifID <= 0 {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "ID notifikasi tidak valid")
		return
	}

	var curStatus string
	var curClassID int64
	err = s.v1DB.QueryRow(`SELECT status, class_id FROM notification_messages WHERE id = ?;`, notifID).Scan(&curStatus, &curClassID)
	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Pesan notifikasi tidak ditemukan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi notifikasi")
		return
	}

	if u.ActiveRole != "SYSTEM_ADMIN" {
		if !u.ActiveClassID.Valid || u.ActiveClassID.Int64 != curClassID {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Hanya pengurus kelas terkait yang berwenang mencoba ulang pengiriman notifikasi")
			return
		}
	}

	// BE-010 state machine: hanya FAILED dan CANCELLED yang dapat diretry.
	// CANCELLED di sini bersifat reversibel (penjadwalan ulang oleh pengurus);
	// SUPERSEDED tidak pernah diretry (isi usang), SENT/PROCESSING/PENDING ditolak.
	// Retry hanya menjadwalkan ulang baris yang sama; notification_attempts
	// hanya dibuat worker saat delivery nyata dimulai.
	if curStatus != "FAILED" && curStatus != "CANCELLED" {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, fmt.Sprintf("Hanya notifikasi dengan status FAILED atau CANCELLED yang dapat dicoba ulang (status saat ini: %s)", curStatus))
		return
	}

	// Update status notifikasi menjadi PENDING
	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE notification_messages
		SET status = 'PENDING', scheduled_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN ('FAILED', 'CANCELLED');
	`, notifID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui status notifikasi")
		return
	}

	affected, err := res.RowsAffected()
	if err != nil || affected == 0 {
		s.writeV1Error(w, http.StatusConflict, CodeVersionConflict, "Status notifikasi telah berubah")
		return
	}

	// Catat audit_logs
	correlationID := fmt.Sprintf("retry-notif-%d", time.Now().UnixNano())
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, curStatus)
		afterJSON := `{"status":"PENDING"}`
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &curClassID,
			Action:        "RETRY_NOTIFICATION",
			EntityType:    "NOTIFICATION_MESSAGE",
			EntityID:      &notifID,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			CorrelationID: correlationID,
		}); err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencatat audit retry notifikasi")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memproses transaksi")
		return
	}

	var scheduledAt string
	_ = s.v1DB.QueryRow(`SELECT COALESCE(scheduled_at,'') FROM notification_messages WHERE id=?`, notifID).Scan(&scheduledAt)
	s.writeV1Success(w, http.StatusOK, map[string]any{
		"id":              notifID,
		"status":          "PENDING",
		"scheduled_at":    scheduledAt,
		"retry_scheduled": true,
	})
}
