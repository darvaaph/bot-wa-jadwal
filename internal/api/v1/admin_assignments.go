package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/audit"
	"bot-jadwal/internal/ratelimit"
)

// AssignmentItem merepresentasikan satu Penugasan Peran beserta identitas
// pengguna dan label cakupannya untuk layar Pengguna dan Penugasan.
type AssignmentItem struct {
	ID               int64   `json:"id"`
	UserID           int64   `json:"user_id"`
	IdentityKey      string  `json:"identity_key"`
	DisplayName      string  `json:"display_name"`
	UserStatus       string  `json:"user_status"`
	Role             string  `json:"role"`
	ScopeType        string  `json:"scope_type"`
	ClassID          *int64  `json:"class_id,omitempty"`
	ClassSlug        *string `json:"class_slug,omitempty"`
	ClassCode        *string `json:"class_code,omitempty"`
	SemesterID       *int64  `json:"semester_id,omitempty"`
	SemesterLabel    *string `json:"semester_label,omitempty"`
	CourseOfferingID *int64  `json:"course_offering_id,omitempty"`
	OfferingDisplay  *string `json:"offering_display,omitempty"`
	CourseCode       *string `json:"course_code,omitempty"`
	CourseName       *string `json:"course_name,omitempty"`
	Status           string  `json:"status"`
	ValidFrom        string  `json:"valid_from"`
	ValidUntil       *string `json:"valid_until,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

// InvitationItem merepresentasikan satu Undangan tanpa membocorkan token.
type InvitationItem struct {
	ID                 int64   `json:"id"`
	InvitedIdentityKey string  `json:"invited_identity_key"`
	Role               string  `json:"role"`
	ScopeType          string  `json:"scope_type"`
	ClassID            *int64  `json:"class_id,omitempty"`
	ClassSlug          *string `json:"class_slug,omitempty"`
	SemesterID         *int64  `json:"semester_id,omitempty"`
	CourseOfferingID   *int64  `json:"course_offering_id,omitempty"`
	Status             string  `json:"status"`
	ExpiresAt          string  `json:"expires_at"`
	IsExpired          bool    `json:"is_expired"`
	InvitedBy          *string `json:"invited_by,omitempty"`
	CreatedAt          string  `json:"created_at"`
}

// assignmentStatusBody adalah payload tangguhkan/cabut penugasan dan cabut undangan.
type assignmentStatusBody struct {
	Reason string `json:"reason"`
	Force  bool   `json:"force"`
}

// goStringTimeRe mencocokkan awalan format Go Time.String():
// "2006-01-02 15:04:05.999999999 -0700 ..." (data lama/demo menyimpan
// expires_at demikian, termasuk akhiran zona/monotonik seperti "+07 m=+...").
var goStringTimeRe = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? [+-]\d{4})`)

// normalizeTimestamp mengembalikan representasi RFC3339 dari nilai waktu
// database. Tulisan baru sudah RFC3339; data lama/demo bisa berformat Go
// String() yang gagal di-parse browser. Tak dapat di-parse -> apa adanya.
func normalizeTimestamp(v string) string {
	s := strings.TrimSpace(v)
	if s == "" {
		return s
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.Format(time.RFC3339)
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.Format(time.RFC3339)
	}
	if m := goStringTimeRe.FindStringSubmatch(s); m != nil {
		for _, f := range []string{"2006-01-02 15:04:05.999999999 -0700", "2006-01-02 15:04:05 -0700"} {
			if t, err := time.Parse(f, m[1]); err == nil {
				return t.Format(time.RFC3339)
			}
		}
	}
	return v
}

// GetAssignments menangani GET /api/v1/admin/assignments (khusus System Admin).
func (c *AdminController) GetAssignments(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya System Admin yang berwenang melihat penugasan peran")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	roleFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("role")))
	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	limit, offset := adminListPaging(r)

	query := `
		SELECT ra.id, ra.user_id, u.identity_key, u.display_name, u.status,
		       ra.role, ra.scope_type,
		       ra.class_id, c.slug, c.code,
		       ra.semester_id, (s.academic_year || ' ' || s.term),
		       ra.course_offering_id, co.display_name, crs.code, crs.name,
		       ra.status, ra.valid_from, ra.valid_until, ra.created_at
		FROM role_assignments ra
		JOIN users u ON u.id = ra.user_id
		LEFT JOIN classes c ON c.id = ra.class_id
		LEFT JOIN semesters s ON s.id = ra.semester_id
		LEFT JOIN course_offerings co ON co.id = ra.course_offering_id
		LEFT JOIN courses crs ON crs.id = co.course_id
		WHERE (1=1)
	`
	var args []any
	if statusFilter != "" {
		query += " AND ra.status = ?"
		args = append(args, statusFilter)
	}
	if roleFilter != "" {
		query += " AND ra.role = ?"
		args = append(args, roleFilter)
	}
	if classSlugFilter != "" {
		query += " AND c.slug = ?"
		args = append(args, classSlugFilter)
	}
	query += " ORDER BY ra.id ASC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat penugasan peran")
		return
	}
	defer rows.Close()

	items := []AssignmentItem{}
	for rows.Next() {
		var it AssignmentItem
		var classID, semesterID, offeringID sql.NullInt64
		var classSlug, classCode, semesterLabel, offeringDisplay, courseCode, courseName, validUntil sql.NullString
		if err := rows.Scan(
			&it.ID, &it.UserID, &it.IdentityKey, &it.DisplayName, &it.UserStatus,
			&it.Role, &it.ScopeType,
			&classID, &classSlug, &classCode,
			&semesterID, &semesterLabel,
			&offeringID, &offeringDisplay, &courseCode, &courseName,
			&it.Status, &it.ValidFrom, &validUntil, &it.CreatedAt,
		); err != nil {
			continue
		}
		if classID.Valid {
			it.ClassID = &classID.Int64
		}
		if classSlug.Valid {
			it.ClassSlug = &classSlug.String
		}
		if classCode.Valid {
			it.ClassCode = &classCode.String
		}
		if semesterID.Valid {
			it.SemesterID = &semesterID.Int64
		}
		if semesterLabel.Valid {
			it.SemesterLabel = &semesterLabel.String
		}
		if offeringID.Valid {
			it.CourseOfferingID = &offeringID.Int64
		}
		if offeringDisplay.Valid {
			it.OfferingDisplay = &offeringDisplay.String
		}
		if courseCode.Valid {
			it.CourseCode = &courseCode.String
		}
		if courseName.Valid {
			it.CourseName = &courseName.String
		}
		if validUntil.Valid {
			it.ValidUntil = &validUntil.String
		}
		items = append(items, it)
	}

	common.WriteV1Success(w, http.StatusOK, items)
}

// SuspendAssignment menangani POST /api/v1/admin/assignments/{id}/suspend.
func (c *AdminController) SuspendAssignment(w http.ResponseWriter, r *http.Request) {
	c.changeAssignmentStatus(w, r, "SUSPENDED")
}

// RevokeAssignment menangani POST /api/v1/admin/assignments/{id}/revoke.
func (c *AdminController) RevokeAssignment(w http.ResponseWriter, r *http.Request) {
	c.changeAssignmentStatus(w, r, "REVOKED")
}

// changeAssignmentStatus menangguhkan/mencabut satu penugasan, mencabut sesi yang
// memakai penugasan tersebut, dan mencatat audit. Cakupan: System Admin global;
// KM hanya untuk PJ pada kelas penugasannya.
func (c *AdminController) changeAssignmentStatus(w http.ResponseWriter, r *http.Request, target string) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang mengubah penugasan peran")
		return
	}

	targetID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || targetID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID penugasan tidak valid")
		return
	}

	var ra struct {
		id         int64
		userID     int64
		role       string
		classID    sql.NullInt64
		status     string
		identity   string
		display    string
		userStatus string
	}
	err = c.db.QueryRow(`
		SELECT ra.id, ra.user_id, ra.role, ra.class_id, ra.status, u.identity_key, u.display_name, u.status
		FROM role_assignments ra
		JOIN users u ON u.id = ra.user_id
		WHERE ra.id = ?;
	`, targetID).Scan(&ra.id, &ra.userID, &ra.role, &ra.classID, &ra.status, &ra.identity, &ra.display, &ra.userStatus)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Penugasan peran tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi penugasan peran")
		return
	}

	// Pemeriksaan cakupan: KM hanya mengelola PJ pada kelasnya. Di luar cakupan
	// kelas, keberadaan data tidak diungkap (404); di dalam cakupan tetapi aksi
	// tak diizinkan, tolak sebagai forbidden (403).
	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || !ra.classID.Valid || ra.classID.Int64 != u.ActiveClassID.Int64 {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Penugasan peran tidak ditemukan")
			return
		}
		if ra.role != "PJ" {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang mengelola penugasan PJ pada kelasnya")
			return
		}
	}

	if targetID == u.ActiveAssignmentID {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "Tidak dapat mengubah penugasan aktif Anda sendiri")
		return
	}

	if ra.status == target {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, fmt.Sprintf("Penugasan sudah berstatus %s", target))
		return
	}
	if ra.status == "REVOKED" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Penugasan yang dicabut tidak dapat diubah lagi")
		return
	}

	var body assignmentStatusBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan tindakan wajib diisi")
		return
	}

	// Guard KM terakhir: kelas tidak boleh kehilangan seluruh KM aktif kecuali
	// insiden keamanan eksplisit (force) — ACCESS_CONTROL §11.2.
	if ra.role == "KM" && ra.status == "ACTIVE" && ra.classID.Valid && !body.Force {
		var sisa int
		_ = c.db.QueryRow(`
			SELECT COUNT(*) FROM role_assignments
			WHERE role = 'KM' AND status = 'ACTIVE' AND class_id = ? AND id != ?;
		`, ra.classID.Int64, ra.id).Scan(&sisa)
		if sisa == 0 {
			common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Kelas akan kehilangan seluruh KM aktif. Undang pengganti dulu, atau ulangi dengan force untuk insiden keamanan")
			return
		}
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

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi penugasan")
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE role_assignments
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?;
	`, target, ra.id); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memperbarui penugasan peran")
		return
	}

	// Cabut sesi yang memakai penugasan ini agar pencabutan berlaku segera.
	res, err := tx.Exec(`
		UPDATE user_sessions
		SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = ?
		WHERE active_role_assignment_id = ? AND revoked_at IS NULL;
	`, target+"_"+reason, ra.id)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencabut sesi penugasan")
		return
	}
	revokedSessions, _ := res.RowsAffected()

	action := "SUSPEND_ASSIGNMENT"
	if target == "REVOKED" {
		action = "REVOKE_ASSIGNMENT"
	}
	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := fmt.Sprintf(`{"status":%q}`, ra.status)
		afterJSON := fmt.Sprintf(`{"status":%q,"revoked_sessions":%d}`, target, revokedSessions)
		correlationID := fmt.Sprintf("assignment-%d-%d-%d", ra.id, time.Now().UnixNano(), uid)
		entry := audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			Action:        action,
			EntityType:    "ROLE_ASSIGNMENT",
			EntityID:      &ra.id,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}
		if ra.classID.Valid {
			entry.ClassID = &ra.classID.Int64
		}
		if err := audit.Write(r.Context(), tx, entry); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit penugasan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit penugasan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyAdminMutation, subject, source, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"assignment_id":    ra.id,
		"status":           target,
		"revoked_sessions": revokedSessions,
	})
}

// GetInvitations menangani GET /api/v1/admin/invitations.
// System Admin melihat lintas kelas; KM hanya kelas penugasannya. Token tak pernah
// dikembalikan.
func (c *AdminController) GetInvitations(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang melihat undangan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	roleFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("role")))
	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	limit, offset := adminListPaging(r)

	query := `
		SELECT ri.id, ri.invited_identity_key, ri.role, ri.scope_type,
		       ri.class_id, c.slug, ri.semester_id, ri.course_offering_id,
		       ri.status, ri.expires_at,
		       CASE WHEN ri.status = 'PENDING' AND ri.expires_at <= CURRENT_TIMESTAMP THEN 1 ELSE 0 END,
		       u.display_name, ri.created_at
		FROM role_invitations ri
		LEFT JOIN classes c ON c.id = ri.class_id
		LEFT JOIN users u ON u.id = ri.invited_by_user_id
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
				common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Undangan tidak ditemukan")
				return
			}
		}
		query += " AND ri.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND c.slug = ?"
		args = append(args, classSlugFilter)
	}
	if statusFilter == "EXPIRED" {
		// Status EXPIRED tak tersimpan di kolom (hanya derivasi): petakan ke
		// PENDING yang lewat kedaluwarsa.
		query += " AND ri.status = 'PENDING' AND ri.expires_at <= CURRENT_TIMESTAMP"
	} else if statusFilter != "" {
		query += " AND ri.status = ?"
		args = append(args, statusFilter)
	}
	if roleFilter != "" {
		query += " AND ri.role = ?"
		args = append(args, roleFilter)
	}
	query += " ORDER BY ri.created_at DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat undangan")
		return
	}
	defer rows.Close()

	items := []InvitationItem{}
	for rows.Next() {
		var it InvitationItem
		var classID, semesterID, offeringID sql.NullInt64
		var classSlug, inviter sql.NullString
		var expired int
		if err := rows.Scan(
			&it.ID, &it.InvitedIdentityKey, &it.Role, &it.ScopeType,
			&classID, &classSlug, &semesterID, &offeringID,
			&it.Status, &it.ExpiresAt, &expired, &inviter, &it.CreatedAt,
		); err != nil {
			continue
		}
		if classID.Valid {
			it.ClassID = &classID.Int64
		}
		if classSlug.Valid {
			it.ClassSlug = &classSlug.String
		}
		if semesterID.Valid {
			it.SemesterID = &semesterID.Int64
		}
		if offeringID.Valid {
			it.CourseOfferingID = &offeringID.Int64
		}
		it.IsExpired = expired == 1
		if inviter.Valid {
			it.InvitedBy = &inviter.String
		}
		it.ExpiresAt = normalizeTimestamp(it.ExpiresAt)
		it.CreatedAt = normalizeTimestamp(it.CreatedAt)
		items = append(items, it)
	}

	common.WriteV1Success(w, http.StatusOK, items)
}

// RevokeInvitation menangani POST /api/v1/admin/invitations/{id}/revoke.
// Hanya undangan PENDING yang dapat dicabut. KM hanya untuk PJ pada kelasnya.
func (c *AdminController) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang mencabut undangan")
		return
	}

	invID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || invID <= 0 {
		common.WriteV1Error(w, http.StatusBadRequest, common.CodeValidation, "ID undangan tidak valid")
		return
	}

	var inv struct {
		id       int64
		role     string
		classID  sql.NullInt64
		status   string
		identity string
	}
	err = c.db.QueryRow(`
		SELECT id, role, class_id, status, invited_identity_key
		FROM role_invitations WHERE id = ?;
	`, invID).Scan(&inv.id, &inv.role, &inv.classID, &inv.status, &inv.identity)
	if err == sql.ErrNoRows {
		common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Undangan tidak ditemukan")
		return
	} else if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi undangan")
		return
	}

	if u.ActiveRole == "KM" {
		if !u.ActiveClassID.Valid || !inv.classID.Valid || inv.classID.Int64 != u.ActiveClassID.Int64 {
			common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Undangan tidak ditemukan")
			return
		}
		if inv.role != "PJ" {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "KM hanya berwenang mencabut undangan PJ pada kelasnya")
			return
		}
	}

	if inv.status != "PENDING" {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, fmt.Sprintf("Hanya undangan PENDING yang dapat dicabut (status saat ini: %s)", inv.status))
		return
	}

	var body assignmentStatusBody
	_ = json.NewDecoder(r.Body).Decode(&body)
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		common.WriteV1Error(w, http.StatusUnprocessableEntity, common.CodeValidation, "Alasan tindakan wajib diisi")
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

	tx, err := c.db.Begin()
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi undangan")
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE role_invitations
		SET status = 'REVOKED', revoked_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'PENDING';
	`, inv.id)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mencabut undangan")
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		common.WriteV1Error(w, http.StatusConflict, common.CodeVersionConflict, "Status undangan telah berubah")
		return
	}

	{
		uid := u.UserID
		var raid *int64
		if u.ActiveAssignmentID != 0 {
			v := u.ActiveAssignmentID
			raid = &v
		}
		beforeJSON := `{"status":"PENDING"}`
		afterJSON := `{"status":"REVOKED"}`
		correlationID := fmt.Sprintf("revoke-inv-%d-%d-%d", inv.id, time.Now().UnixNano(), uid)
		entry := audit.Entry{
			Actor:         audit.Actor{Type: "USER", UserID: &uid, RoleAssignmentID: raid},
			Action:        "REVOKE_INVITATION",
			EntityType:    "ROLE_INVITATION",
			EntityID:      &inv.id,
			BeforeJSON:    &beforeJSON,
			AfterJSON:     &afterJSON,
			Reason:        reason,
			CorrelationID: correlationID,
		}
		if inv.classID.Valid {
			entry.ClassID = &inv.classID.Int64
		}
		if err := audit.Write(r.Context(), tx, entry); err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan audit undangan")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal commit undangan")
		return
	}
	if c.rlManager != nil {
		c.rlManager.RecordSensitiveLimit(ratelimit.PolicyAdminMutation, subject, source, "SUCCESS")
	}

	common.WriteV1Success(w, http.StatusOK, map[string]any{
		"invitation_id": inv.id,
		"status":        "REVOKED",
	})
}

// GetBackups menangani GET /api/v1/backups (BE-010).
// System Admin melihat lintas kelas; KM hanya kelas penugasannya.
// artifact_ref (jalur internal) tidak pernah dikembalikan.
func (c *AdminController) GetBackups(w http.ResponseWriter, r *http.Request) {
	u, ok := common.GetAuthContext(r)
	if !ok {
		common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}
	if u.ActiveRole != "SYSTEM_ADMIN" && u.ActiveRole != "KM" {
		common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Hanya KM atau System Admin yang berwenang melihat cadangan")
		return
	}

	statusFilter := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status")))
	classSlugFilter := strings.TrimSpace(r.URL.Query().Get("class_slug"))
	limit, offset := adminListPaging(r)

	query := `
		SELECT br.id, br.class_id, c.slug, br.semester_id, br.checksum, br.status,
		       br.reason, u.display_name, br.created_at, br.verified_at
		FROM backup_records br
		LEFT JOIN classes c ON c.id = br.class_id
		LEFT JOIN users u ON u.id = br.created_by_user_id
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
				common.WriteV1Error(w, http.StatusNotFound, common.CodeNotFound, "Cadangan tidak ditemukan")
				return
			}
		}
		query += " AND br.class_id = ?"
		args = append(args, u.ActiveClassID.Int64)
	} else if classSlugFilter != "" {
		query += " AND c.slug = ?"
		args = append(args, classSlugFilter)
	}
	if statusFilter != "" {
		query += " AND br.status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY br.id DESC LIMIT ? OFFSET ?;"
	args = append(args, limit, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memuat cadangan")
		return
	}
	defer rows.Close()

	items := []BackupResponseItem{}
	for rows.Next() {
		var it BackupResponseItem
		var classID sql.NullInt64
		var classSlug, reason, createdBy, verifiedAt sql.NullString
		var semesterID sql.NullInt64
		if err := rows.Scan(
			&it.ID, &classID, &classSlug, &semesterID, &it.Checksum, &it.Status,
			&reason, &createdBy, &it.CreatedAt, &verifiedAt,
		); err != nil {
			continue
		}
		if classID.Valid {
			it.ClassID = &classID.Int64
		}
		if classSlug.Valid {
			it.ClassSlug = &classSlug.String
		}
		if semesterID.Valid {
			it.SemesterID = &semesterID.Int64
		}
		if reason.Valid {
			it.Reason = &reason.String
		}
		if createdBy.Valid {
			it.CreatedBy = &createdBy.String
		}
		if verifiedAt.Valid {
			it.VerifiedAt = &verifiedAt.String
		}
		items = append(items, it)
	}

	common.WriteV1Success(w, http.StatusOK, items)
}

// adminListPaging membaca limit/offset dengan batas yang sama di seluruh daftar admin.
func adminListPaging(r *http.Request) (int, int) {
	limit := 50
	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 && l <= 100 {
		limit = l
	}
	offset := 0
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	return limit, offset
}
