package v1

import (
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

// ChannelItem merepresentasikan satu kanal WhatsApp terdaftar beserta tautannya.
type ChannelItem struct {
	ID          int64   `json:"id"`
	JID         string  `json:"jid"`
	ChannelType string  `json:"channel_type"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
	ClassID     int64   `json:"class_id"`
	ClassSlug   string  `json:"class_slug"`
	ClassCode   string  `json:"class_code"`
	VerifiedAt  *string `json:"verified_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// LinkChannelRequest adalah payload menautkan grup ke kelas (BE-014).
type LinkChannelRequest struct {
	JID         string  `json:"jid"`
	ClassSlug   string  `json:"class_slug"`
	DisplayName *string `json:"display_name,omitempty"`
}

// validChannelJID memeriksa bentuk JID (user/group + domain bertitik).
func validChannelJID(jid string) bool {
	jid = strings.TrimSpace(jid)
	at := strings.LastIndex(jid, "@")
	if at <= 0 || len(jid) > 128 {
		return false
	}
	return strings.Contains(jid[at+1:], ".")
}

// channelScope menyelesaikan class_id dari slug + menegakkan cakupan.
// KM di luar kelasnya menerima 404 generik (FR-ACCESS-004).
func channelScope(c *AdminController, u *common.UserContext, slug string) (int64, error) {
	var classID int64
	if err := c.db.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, strings.TrimSpace(slug)).Scan(&classID); err != nil {
		return 0, sql.ErrNoRows
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		return 0, sql.ErrNoRows
	}
	return classID, nil
}

// GetChannels menangani GET /api/v1/whatsapp-channels.
// System Admin lintas kelas; KM hanya kelasnya.
func (c *AdminController) GetChannels(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang melihat kanal")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	limit, offset := adminListPaging(r)

	query := `
		SELECT wc.id, wc.jid, wc.channel_type, wc.display_name, wc.status,
		       wc.class_id, c.slug, c.code, wc.verified_at, wc.created_at
		FROM whatsapp_channels wc
		JOIN classes c ON c.id = wc.class_id
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
				common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kanal tidak ditemukan")
				return
			}
		}
		query += " AND wc.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND c.slug = ?"
		args = append(args, classSlugFilter)
	}
	if statusFilter != "" {
		query += " AND wc.status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY wc.id DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat kanal")
		return
	}
	defer rows.Close()

	items := []ChannelItem{}
	for rows.Next() {
		var it ChannelItem
		var verifiedAt sql.NullString
		if err := rows.Scan(
			&it.ID, &it.JID, &it.ChannelType, &it.DisplayName, &it.Status,
			&it.ClassID, &it.ClassSlug, &it.ClassCode, &verifiedAt, &it.CreatedAt,
		); err != nil {
			continue
		}
		if verifiedAt.Valid {
			it.VerifiedAt = &verifiedAt.String
		}
		items = append(items, it)
	}

	common.WriteV1Success(w, http.StatusOK, items)
}

// LinkChannel menangani POST /api/v1/whatsapp-channels (BE-014).
// JID yang sudah tertaut ke kelas lain DITOLAK (409) — pindah = lepas + taut
// ulang. Menautkan ulang JID yang sama ke kelas yang sama bersifat idempoten.
func (c *AdminController) LinkChannel(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang menautkan kanal")
		return
	}

	var req LinkChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	jid := strings.TrimSpace(req.JID)
	slug := strings.TrimSpace(req.ClassSlug)
	if !validChannelJID(jid) {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "JID kanal tidak valid (contoh: 12036...@g.us)")
		return
	}
	// KM boleh kosongkan kelas: otomatis kelas penugasannya.
	if slug == "" {
		if u.ActiveRole != "KM" || !u.ActiveClassID.Valid {
			common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "class_slug wajib diisi")
			return
		}
		_ = c.db.QueryRow(`SELECT slug FROM classes WHERE id = ?;`, u.ActiveClassID.Int64).Scan(&slug)
	}
	if slug == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "class_slug wajib diisi")
		return
	}
	classID, err := channelScope(c, u, slug)
	if err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	name := ""
	if req.DisplayName != nil {
		name = strings.TrimSpace(*req.DisplayName)
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi kanal")
		return
	}
	defer tx.Rollback()

	var existing struct {
		id      int64
		classID int64
		status  string
	}
	err = tx.QueryRow(`SELECT id, class_id, status FROM whatsapp_channels WHERE jid = ?;`, jid).Scan(&existing.id, &existing.classID, &existing.status)
	if err == nil {
		if existing.status == "ACTIVE" {
			if existing.classID != classID {
				common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "JID sudah tertaut ke kelas lain. Lepas dulu tautan lamanya, lalu tautkan ulang.")
				return
			}
			if _, err := tx.Exec(`
				UPDATE whatsapp_channels
				SET display_name = COALESCE(NULLIF(?, ''), display_name),
				    verified_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
				WHERE id = ?;
			`, name, existing.id); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui kanal")
				return
			}
			if err := channelAudit(r, tx, u, "LINK_CHANNEL", existing.id, classID, "", "Tautan ulang (idempoten)", "ACTIVE"); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit kanal")
				return
			}
			if err := tx.Commit(); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan kanal")
				return
			}
			common.WriteV1Success(w, http.StatusOK, map[string]any{"id": existing.id, "jid": jid, "status": "ACTIVE", "changed": false})
			return
		}
		// Baris non-ACTIVE (REVOKED/DISCONNECTED): taut ulang diizinkan,
		// termasuk pindah kelas — status lama sudah non-aktif.
		if _, err := tx.Exec(`
			UPDATE whatsapp_channels
			SET class_id = ?, status = 'ACTIVE',
			    display_name = COALESCE(NULLIF(?, ''), display_name),
			    verified_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?;
		`, classID, name, existing.id); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menautkan ulang kanal")
			return
		}
		if err := channelAudit(r, tx, u, "LINK_CHANNEL", existing.id, classID, "", "Taut ulang pasca-lepas", "ACTIVE"); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit kanal")
			return
		}
		if err := tx.Commit(); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan kanal")
			return
		}
		common.WriteV1Success(w, http.StatusOK, map[string]any{"id": existing.id, "jid": jid, "status": "ACTIVE", "changed": true})
		return
	} else if err != sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa kanal")
		return
	}

	if name == "" {
		name = "Grup WhatsApp"
	}
	res, err := tx.Exec(`
		INSERT INTO whatsapp_channels (class_id, jid, channel_type, display_name, status, verified_at)
		VALUES (?, ?, 'GROUP', ?, 'ACTIVE', CURRENT_TIMESTAMP);
	`, classID, jid, name)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menautkan kanal")
		return
	}
	id, _ := res.LastInsertId()
	if err := channelAudit(r, tx, u, "LINK_CHANNEL", id, classID, "", fmt.Sprintf("Tautkan %s", jid), "ACTIVE"); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit kanal")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan kanal")
		return
	}
	common.WriteV1Success(w, http.StatusCreated, map[string]any{"id": id, "jid": jid, "status": "ACTIVE", "changed": true})
}

// RevokeChannelRequest adalah payload melepas tautan kanal.
type RevokeChannelRequest struct {
	Reason string `json:"reason"`
}

// RevokeChannel menangani POST /api/v1/whatsapp-channels/{id}/revoke.
// Hanya ACTIVE yang dapat dilepas; baris dipertahankan (REVOKED) demi riwayat.
func (c *AdminController) RevokeChannel(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang melepas kanal")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID kanal tidak valid")
		return
	}

	var classID int64
	var status string
	if err := c.db.QueryRow(`SELECT class_id, status FROM whatsapp_channels WHERE id = ?;`, id).Scan(&classID, &status); err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kanal tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi kanal")
		return
	}
	if u.ActiveRole == "KM" && (!u.ActiveClassID.Valid || u.ActiveClassID.Int64 != classID) {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kanal tidak ditemukan")
		return
	}
	if status != "ACTIVE" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, fmt.Sprintf("Hanya kanal ACTIVE yang dapat dilepas (status: %s)", status))
		return
	}

	var body RevokeChannelRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan pelepasan wajib diisi")
		return
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi kanal")
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`
		UPDATE whatsapp_channels
		SET status = 'REVOKED', updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'ACTIVE';
	`, id)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal melepas kanal")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Status kanal telah berubah")
		return
	}
	if err := channelAudit(r, tx, u, "REVOKE_CHANNEL", id, classID, reason, "", "REVOKED"); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit kanal")
		return
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal melepas kanal")
		return
	}
	common.WriteV1Success(w, http.StatusOK, map[string]any{"id": id, "status": "REVOKED"})
}

// channelAudit mencatat mutasi kanal beraudit (reason = alasan pengguna).
func channelAudit(r *http.Request, tx *sql.Tx, u *common.UserContext, action string, entityID, classID int64, reason, note, status string) error {
	uid := u.UserID
	var raid *int64
	if u.ActiveAssignmentID != 0 {
		v := u.ActiveAssignmentID
		raid = &v
	}
	afterJSON := fmt.Sprintf(`{"status":%q,"note":%q}`, status, note)
	return audit.Write(r.Context(), tx, audit.Entry{
		Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
		ClassID:       &classID,
		Action:        action,
		EntityType:    "WHATSAPP_CHANNEL",
		EntityID:      &entityID,
		AfterJSON:     &afterJSON,
		Reason:        reason,
		CorrelationID: fmt.Sprintf("channel-%d-%d", entityID, time.Now().UnixNano()),
	})
}
