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
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/ratelimit"
)

// supportTTL adalah masa berlaku satu hibah dukungan (BE-004).
const supportTTL = 60 * time.Minute

// SupportGrantItem merepresentasikan hibah Mode Dukungan aktif.
type SupportGrantItem struct {
	ID        int64  `json:"id"`
	ClassID   int64  `json:"class_id"`
	ClassSlug string `json:"class_slug"`
	ClassCode string `json:"class_code"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// supportNowUTC mengembalikan cap waktu UTC format T (selaras strftime
// penyimpanan) untuk perbandingan string yang benar.
func supportNowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05")
}

// EnterSupportRequest adalah payload masuk Mode Dukungan.
type EnterSupportRequest struct {
	ClassSlug string `json:"class_slug"`
	Reason    string `json:"reason"`
}

// EnterSupport menangani POST /api/v1/admin/support/enter (khusus System Admin).
// Membuat hibah dukungan 60 menit untuk satu kelas dengan alasan tercatat.
// Hibah aktif sebelumnya milik pengguna yang sama ditutup otomatis.
func (c *AdminController) EnterSupport(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang masuk Mode Dukungan")
		return
	}

	var req EnterSupportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Payload JSON tidak valid")
		return
	}
	slug := strings.TrimSpace(req.ClassSlug)
	reason := strings.TrimSpace(req.Reason)
	if slug == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "class_slug wajib diisi")
		return
	}
	if len([]rune(reason)) < 10 {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan dukungan minimal 10 karakter")
		return
	}
	var classID int64
	var classCode string
	if err := c.db.QueryRow(`SELECT id, code FROM classes WHERE slug = ?;`, slug).Scan(&classID, &classCode); err != nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	subject := fmt.Sprintf("admin:%d", u.UserID)
	var trustedProxies []string
	if c.secManager != nil {
		trustedProxies = c.secManager.TrustedProxyCIDRs()
	}
	source := middleware.ClientSource(r, trustedProxies)
	if c.rlManager != nil && !c.rlManager.CheckSensitiveLimit(w, r, trustedProxies, ratelimit.PolicyAdminMutation, subject) {
		return
	}

	now := supportNowUTC()
	expires := time.Now().UTC().Add(supportTTL).Format(time.RFC3339)

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi dukungan")
		return
	}
	defer tx.Rollback()

	// Satu hibah aktif per pengguna: tutup yang lama (termasuk yang kedaluwarsa)
	// dengan audit SUPPORT_EXIT masing-masing agar jejak lengkap.
	var superseded []int64
	supersededReasons := map[int64]string{}
	rows, err := tx.Query(`SELECT id, reason FROM support_grants WHERE user_id = ? AND status = 'ACTIVE';`, u.UserID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa dukungan aktif")
		return
	}
	for rows.Next() {
		var oldID int64
		var oldReason string
		if err := rows.Scan(&oldID, &oldReason); err == nil {
			superseded = append(superseded, oldID)
			supersededReasons[oldID] = oldReason
		}
	}
	rows.Close()
	if _, err := tx.Exec(`
		UPDATE support_grants
		SET status = 'CLOSED', closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = ? AND status = 'ACTIVE';
	`, u.UserID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menutup dukungan sebelumnya")
		return
	}

	res, err := tx.Exec(`
		INSERT INTO support_grants (user_id, class_id, reason, status, expires_at)
		VALUES (?, ?, ?, 'ACTIVE', ?);
	`, u.UserID, classID, reason, expires)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal membuat hibah dukungan")
		return
	}
	grantID, _ := res.LastInsertId()

	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		for _, oldID := range superseded {
			oldReason := supersededReasons[oldID]
			if oldReason == "" {
				oldReason = reason
			}
			afterJSON := `{"status":"CLOSED","superseded_by":` + strconv.FormatInt(grantID, 10) + `}`
			if err := audit.Write(r.Context(), tx, audit.Entry{
				Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
				ClassID:       &classID,
				Action:        "SUPPORT_EXIT",
				EntityType:    "SUPPORT_GRANT",
				EntityID:      &oldID,
				AfterJSON:     &afterJSON,
				Reason:        "Digantikan hibah dukungan baru; alasan awal: " + oldReason,
				CorrelationID: fmt.Sprintf("support-supersede-%d-%d", oldID, time.Now().UnixNano()),
			}); err != nil {
				common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit dukungan")
				return
			}
		}
		afterJSON := fmt.Sprintf(`{"grant_id":%d,"expires_at":%q,"superseded":%v}`, grantID, expires, superseded)
		correlationID := fmt.Sprintf("support-enter-%d-%d", grantID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &classID,
			Action:        "SUPPORT_ENTER",
			EntityType:    "SUPPORT_GRANT",
			EntityID:      &grantID,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit dukungan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit dukungan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyAdminMutation, subject, source, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusCreated, SupportGrantItem{
		ID:        grantID,
		ClassID:   classID,
		ClassSlug: slug,
		ClassCode: classCode,
		Reason:    reason,
		Status:    "ACTIVE",
		ExpiresAt: expires,
		CreatedAt: now,
	})
}

// ExitSupportRequest adalah payload keluar Mode Dukungan (alasan opsional).
type ExitSupportRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// ExitSupport menangani POST /api/v1/admin/support/exit (khusus System Admin).
func (c *AdminController) ExitSupport(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang keluar Mode Dukungan")
		return
	}

	grant, err := c.activeGrant(u.UserID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memeriksa dukungan aktif")
		return
	}
	if grant == nil {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Tidak ada Mode Dukungan yang aktif")
		return
	}

	var req ExitSupportRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	reason := grant.Reason
	if req.Reason != nil && strings.TrimSpace(*req.Reason) != "" {
		reason = strings.TrimSpace(*req.Reason)
	}

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi dukungan")
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`
		UPDATE support_grants
		SET status = 'CLOSED', closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'ACTIVE';
	`, grant.ID); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menutup dukungan")
		return
	}
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		afterJSON := `{"status":"CLOSED"}`
		correlationID := fmt.Sprintf("support-exit-%d-%d", grant.ID, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			ClassID:       &grant.ClassID,
			Action:        "SUPPORT_EXIT",
			EntityType:    "SUPPORT_GRANT",
			EntityID:      &grant.ID,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit dukungan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit dukungan")
		return
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"grant_id": grant.ID,
		"status":   "CLOSED",
	})
}

// GetActiveSupport menangani GET /api/v1/admin/support/active.
// Hibah yang lewat kedaluwarsa ditandai EXPIRED dan tidak dikembalikan.
func (c *AdminController) GetActiveSupport(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang melihat Mode Dukungan")
		return
	}

	// Tandai hibah pengguna yang kedaluwarsa. Transisi malas ini diaudit sebagai
	// SUPPORT_EXIT beralasan kedaluwarsa agar jejak lengkap; kedaluwarsa adalah
	// fakta waktu, dan endpoint ini memang endpoint baca-tulis status dukungan.
	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi dukungan")
		return
	}
	defer tx.Rollback()
	expiredRows, err := tx.Query(`SELECT id, class_id, reason FROM support_grants WHERE user_id = ? AND status = 'ACTIVE' AND expires_at <= ?;`, u.UserID, supportNowUTC())
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat dukungan aktif")
		return
	}
	type expiredGrant struct {
		id      int64
		classID int64
		reason  string
	}
	var expired []expiredGrant
	for expiredRows.Next() {
		var g expiredGrant
		if err := expiredRows.Scan(&g.id, &g.classID, &g.reason); err == nil {
			expired = append(expired, g)
		}
	}
	expiredRows.Close()
	for _, g := range expired {
		if _, err := tx.Exec(`
			UPDATE support_grants
			SET status = 'EXPIRED', closed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?;
		`, g.id); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menandai kedaluwarsa")
			return
		}
		afterJSON := `{"status":"EXPIRED"}`
		correlationID := fmt.Sprintf("support-expire-%d-%d", g.id, time.Now().UnixNano())
		if err := audit.Write(r.Context(), tx, audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &u.UserID},
			ClassID:       &g.classID,
			Action:        "SUPPORT_EXIT",
			EntityType:    "SUPPORT_GRANT",
			EntityID:      &g.id,
			AfterJSON:     &afterJSON,
			Reason:        "Kedaluwarsa otomatis; alasan awal: " + g.reason,
			CorrelationID: correlationID,
		}); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit dukungan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit dukungan")
		return
	}

	grant, err := c.activeGrant(u.UserID)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat dukungan aktif")
		return
	}
	if grant == nil {
		common.WriteV1Success(w, http.StatusOK, nil)
		return
	}
	common.WriteV1Success(w, http.StatusOK, grant)
}

// activeGrant mengembalikan hibah ACTIVE yang belum kedaluwarsa, atau nil.
func (c *AdminController) activeGrant(userID int64) (*SupportGrantItem, error) {
	var g SupportGrantItem
	err := c.db.QueryRow(`
		SELECT sg.id, sg.class_id, cl.slug, cl.code, sg.reason, sg.status, sg.expires_at, sg.created_at
		FROM support_grants sg
		JOIN classes cl ON cl.id = sg.class_id
		WHERE sg.user_id = ? AND sg.status = 'ACTIVE' AND sg.expires_at > ?
		ORDER BY sg.id DESC LIMIT 1;
	`, userID, supportNowUTC()).Scan(
		&g.ID, &g.ClassID, &g.ClassSlug, &g.ClassCode, &g.Reason, &g.Status, &g.ExpiresAt, &g.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}
