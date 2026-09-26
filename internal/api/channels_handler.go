package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"bot-jadwal/internal/audit"
)

// chatCacheRefresher memuat ulang cache JID->kelas bot setelah admin
// menautkan/melepas kanal. Diimplementasikan oleh ChatSettingsManager.
type chatCacheRefresher interface {
	Refresh() error
}

type channelItem struct {
	ID          int64   `json:"id"`
	ClassID     int64   `json:"class_id"`
	ClassCode   string  `json:"class_code"`
	ClassSlug   string  `json:"class_slug"`
	JID         string  `json:"jid"`
	ChannelType string  `json:"channel_type"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
	VerifiedAt  *string `json:"verified_at,omitempty"`
}

func scanChannelItem(row interface{ Scan(...any) error }) (channelItem, error) {
	var it channelItem
	var verified sql.NullString
	err := row.Scan(&it.ID, &it.ClassID, &it.ClassCode, &it.ClassSlug, &it.JID,
		&it.ChannelType, &it.DisplayName, &it.Status, &verified)
	if err != nil {
		return it, err
	}
	if verified.Valid {
		it.VerifiedAt = &verified.String
	}
	return it, nil
}

func (s *Server) handleListChannels(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat mengelola kanal"})
		return
	}
	rows, err := s.academicRepo.DB().QueryContext(r.Context(), `
		SELECT wc.id, wc.class_id, c.code, c.slug, wc.jid, wc.channel_type, wc.display_name, wc.status, wc.verified_at
		FROM whatsapp_channels wc JOIN classes c ON c.id = wc.class_id
		ORDER BY wc.status, c.code, wc.id LIMIT 500`)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar kanal"})
		return
	}
	defer rows.Close()
	items := []channelItem{}
	for rows.Next() {
		it, err := scanChannelItem(rows)
		if err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal membaca kanal"})
			return
		}
		items = append(items, it)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

type unlinkedChatItem struct {
	ChatJID     string `json:"chat_jid"`
	DisplayName string `json:"display_name"`
	FirstSeenAt string `json:"first_seen_at"`
	LastSeenAt  string `json:"last_seen_at"`
}

func (s *Server) handleListUnlinkedChats(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat melihat chat"})
		return
	}
	rows, err := s.academicRepo.DB().QueryContext(r.Context(), `
		SELECT sc.chat_jid, sc.display_name, sc.first_seen_at, sc.last_seen_at
		FROM seen_chats sc
		WHERE sc.is_group = 1 AND sc.chat_jid NOT IN (SELECT jid FROM whatsapp_channels WHERE status = 'ACTIVE')
		ORDER BY sc.last_seen_at DESC LIMIT 100`)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mengambil daftar chat"})
		return
	}
	defer rows.Close()
	items := []unlinkedChatItem{}
	for rows.Next() {
		var it unlinkedChatItem
		if err := rows.Scan(&it.ChatJID, &it.DisplayName, &it.FirstSeenAt, &it.LastSeenAt); err != nil {
			s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal membaca chat"})
			return
		}
		items = append(items, it)
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"status": "success", "data": items})
}

func (s *Server) handleLinkChannel(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat menautkan kanal"})
		return
	}
	var payload struct {
		JID     string `json:"jid"`
		ClassID int64  `json:"class_id"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Format JSON tidak valid"})
		return
	}
	payload.JID = strings.TrimSpace(payload.JID)
	if payload.JID == "" || payload.ClassID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "Field jid dan class_id wajib diisi"})
		return
	}
	db := s.academicRepo.DB()
	var classCode string
	if err := db.QueryRowContext(r.Context(), `SELECT code FROM classes WHERE id = ?`, payload.ClassID).Scan(&classCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Kelas tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa kelas"})
		return
	}
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()
	var channelID int64
	err = tx.QueryRowContext(r.Context(), `INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status, verified_at)
		VALUES (?, ?, 'GROUP', ?, 'ACTIVE', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		ON CONFLICT(jid) DO UPDATE SET class_id = excluded.class_id, status = 'ACTIVE',
			verified_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'),
			updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
		RETURNING id`, payload.ClassID, payload.JID, classCode).Scan(&channelID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menautkan kanal"})
		return
	}
	var actorUser, actorAssignment any
	if ok {
		actorUser = principal.UserID
		actorAssignment = principal.RoleAssignmentID
	}
	after := `{"jid":"` + strings.ReplaceAll(payload.JID, `"`, ``) + `"}`
	corr := audit.NewCorrelationID()
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO audit_logs (
		class_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, 'USER', 'LINK', 'WHATSAPP_CHANNEL', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		payload.ClassID, actorUser, actorAssignment, channelID, after, corr); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mencatat audit"})
		return
	}
	if err := tx.Commit(); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal menautkan kanal"})
		return
	}
	if s.chatRefresher != nil {
		_ = s.chatRefresher.Refresh()
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"status": "success", "data": map[string]any{"id": channelID, "jid": payload.JID, "class_id": payload.ClassID}})
}

func (s *Server) handleRevokeChannel(w http.ResponseWriter, r *http.Request) {
	if s.academicRepo == nil {
		s.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "error": "Layanan akademik belum tersedia"})
		return
	}
	principal, ok := principalFromRequest(r)
	if !ok {
		if s.authService != nil {
			s.writeJSON(w, http.StatusUnauthorized, map[string]string{"status": "error", "error": "Sesi tidak valid atau telah berakhir"})
			return
		}
	} else if !principal.IsSystemAdmin() {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"status": "error", "error": "Hanya System Admin yang dapat melepas kanal"})
		return
	}
	channelID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || channelID <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"status": "error", "error": "ID kanal tidak valid"})
		return
	}
	var payload struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&payload)
	db := s.academicRepo.DB()
	var classID int64
	var status string
	if err := db.QueryRowContext(r.Context(), `SELECT class_id, status FROM whatsapp_channels WHERE id = ?`, channelID).Scan(&classID, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.writeJSON(w, http.StatusNotFound, map[string]string{"status": "error", "error": "Kanal tidak ditemukan"})
			return
		}
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memeriksa kanal"})
		return
	}
	if status != "ACTIVE" {
		s.writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": "Hanya kanal ACTIVE yang dapat dilepas"})
		return
	}
	tx, err := db.BeginTx(r.Context(), nil)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(), `UPDATE whatsapp_channels SET status = 'REVOKED',
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ? AND status = 'ACTIVE'`, channelID); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal melepas kanal"})
		return
	}
	var actorUser, actorAssignment any
	var reasonAny any
	if ok {
		actorUser = principal.UserID
		actorAssignment = principal.RoleAssignmentID
	}
	if strings.TrimSpace(payload.Reason) != "" {
		reasonAny = strings.TrimSpace(payload.Reason)
	}
	corr := audit.NewCorrelationID()
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO audit_logs (
		class_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, reason, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, 'USER', 'UNLINK', 'WHATSAPP_CHANNEL', ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'))`,
		classID, actorUser, actorAssignment, channelID, reasonAny, corr); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal mencatat audit"})
		return
	}
	if err := tx.Commit(); err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "error", "error": "Gagal melepas kanal"})
		return
	}
	if s.chatRefresher != nil {
		_ = s.chatRefresher.Refresh()
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "success", "message": "Tautan kanal dilepas (riwayat tersimpan)"})
}
