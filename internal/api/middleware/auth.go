package middleware

import (
	"database/sql"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"bot-jadwal/internal/api/common"
)

// AuthManager mengelola verifikasi sesi dan otorisasi berbasis peran (RBAC).
type AuthManager struct {
	db *sql.DB
}

// NewAuthManager membuat instance AuthManager baru.
func NewAuthManager(db *sql.DB) *AuthManager {
	return &AuthManager{db: db}
}

// SetDB memperbarui database connection untuk AuthManager.
func (am *AuthManager) SetDB(db *sql.DB) {
	am.db = db
}

// ExtractToken mengekstrak token dari header Authorization: Bearer <token> atau cookie bv1.
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if cookie, err := r.Cookie("bv1"); err == nil && cookie.Value != "" {
		return strings.TrimSpace(cookie.Value)
	}

	return ""
}

// RequireAuth memverifikasi token Bearer pengguna, masa berlaku sesi (idle & absolute), serta session_version.
func (am *AuthManager) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if am.db == nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum diinisialisasi")
			return
		}

		rawToken := ExtractToken(r)
		if rawToken == "" {
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Token autentikasi tidak ditemukan. Silakan login terlebih dahulu.")
			return
		}

		// Hitung hash SHA-256 token untuk pencarian di database
		h := sha256.Sum256([]byte(rawToken))
		tokenHash := hex.EncodeToString(h[:])

		var (
			sessionID              int64
			userID                 int64
			activeRoleAssignmentID sql.NullInt64
			sessionVersion         int
			lastSeenAtStr          string
			absoluteExpiresAtStr   string
			revokedAtStr           sql.NullString
			identityKey            string
			displayName            string
			userStatus             string
			userSessionVersion     int
			role                   sql.NullString
			scopeType              sql.NullString
			classID                sql.NullInt64
			classSlug              sql.NullString
			semesterID             sql.NullInt64
			courseOfferingID       sql.NullInt64
			raValidFrom            sql.NullString
			raValidUntil           sql.NullString
		)

		query := `
			SELECT
				s.id, s.user_id, s.active_role_assignment_id, s.session_version,
				s.last_seen_at, s.absolute_expires_at, s.revoked_at,
				u.identity_key, u.display_name, u.status, u.session_version,
				ra.role, ra.scope_type, ra.class_id, c.slug, ra.semester_id, ra.course_offering_id,
				ra.valid_from, ra.valid_until
			FROM user_sessions s
			JOIN users u ON s.user_id = u.id
			LEFT JOIN role_assignments ra ON s.active_role_assignment_id = ra.id AND ra.status = 'ACTIVE'
			LEFT JOIN classes c ON ra.class_id = c.id
			WHERE s.token_hash = ?;
		`

		err := am.db.QueryRow(query, tokenHash).Scan(
			&sessionID, &userID, &activeRoleAssignmentID, &sessionVersion,
			&lastSeenAtStr, &absoluteExpiresAtStr, &revokedAtStr,
			&identityKey, &displayName, &userStatus, &userSessionVersion,
			&role, &scopeType, &classID, &classSlug, &semesterID, &courseOfferingID,
			&raValidFrom, &raValidUntil,
		)

		if err == sql.ErrNoRows {
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi tidak valid atau telah berakhir")
			return
		} else if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi sesi", err.Error())
			return
		}

		lastSeenAt, err := common.ParseTime(lastSeenAtStr)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Format last_seen_at tidak valid")
			return
		}
		absoluteExpiresAt, err := common.ParseTime(absoluteExpiresAtStr)
		if err != nil {
			common.WriteV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Format absolute_expires_at tidak valid")
			return
		}
		var revokedAt sql.NullTime
		if revokedAtStr.Valid {
			t, err := common.ParseTime(revokedAtStr.String)
			if err == nil {
				revokedAt = sql.NullTime{Time: t, Valid: true}
			}
		}

		// 1. Cek pencabutan sesi
		if revokedAt.Valid {
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi telah dicabut. Silakan login kembali.")
			return
		}

		// Cek validitas penugasan peran jika sesi memiliki role assignment aktif
		if activeRoleAssignmentID.Valid && !role.Valid {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Peran Anda telah dinonaktifkan, ditangguhkan, atau dicabut")
			return
		}

		// 2. Cek status akun pengguna
		if userStatus != "ACTIVE" {
			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akun Anda saat ini dinonaktifkan")
			return
		}

		// 3. Cek versi sesi pengguna (jika direset / rotasi global)
		if sessionVersion != userSessionVersion {
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi telah direset oleh pembaruan akun. Silakan login kembali.")
			return
		}

		now := time.Now()

		if role.Valid {
			if raValidFrom.Valid {
				vf, err := common.ParseTime(raValidFrom.String)
				if err != nil {
					common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Masa berlaku penugasan peran tidak valid")
					return
				}
				if now.Before(vf) {
					common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Penugasan peran Anda belum aktif")
					return
				}
			}
			if raValidUntil.Valid {
				vu, err := common.ParseTime(raValidUntil.String)
				if err != nil {
					common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Masa berlaku penugasan peran tidak valid")
					return
				}
				if now.After(vu) {
					common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Penugasan peran Anda telah kedaluwarsa")
					return
				}
			}
		}

		// 4. Cek batas kedaluwarsa absolut
		if now.After(absoluteExpiresAt) {
			_, _ = am.db.Exec(`UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'ABSOLUTE_TIMEOUT' WHERE id = ?;`, sessionID)
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi Anda telah kedaluwarsa (batas waktu absolut terlewati)")
			return
		}

		// 5. Cek batas kedaluwarsa idle (ADR-0003: Admin 30 menit, PJ/KM 2 jam)
		idleLimit := 2 * time.Hour
		if role.Valid && role.String == "SYSTEM_ADMIN" {
			idleLimit = 30 * time.Minute
		}

		if now.Sub(lastSeenAt) > idleLimit {
			_, _ = am.db.Exec(`UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'IDLE_TIMEOUT' WHERE id = ?;`, sessionID)
			common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Sesi Anda telah kedaluwarsa karena tidak ada aktivitas")
			return
		}

		// 6. Perbarui waktu aktivitas terakhir (last_seen_at)
		_, _ = am.db.Exec(`UPDATE user_sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?;`, sessionID)

		// 7. Bentuk UserContext dan simpan ke request context
		userCtx := &common.UserContext{
			SessionID:              sessionID,
			UserID:                 userID,
			IdentityKey:            identityKey,
			DisplayName:            displayName,
			UserStatus:             userStatus,
			SessionVersion:         sessionVersion,
			ActiveAssignmentID:     activeRoleAssignmentID.Int64,
			ActiveRole:             role.String,
			ActiveScopeType:        scopeType.String,
			ActiveClassID:          classID,
			ActiveClassSlug:        classSlug.String,
			ActiveSemesterID:       semesterID,
			ActiveCourseOfferingID: courseOfferingID,
		}

		rWithCtx := common.WithAuthContext(r, userCtx)
		next.ServeHTTP(w, rWithCtx)
	}
}

// RequireRole memeriksa apakah peran pengguna saat ini termasuk salah satu dari peran yang diizinkan.
func RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u, ok := common.GetAuthContext(r)
			if !ok {
				common.WriteV1Error(w, http.StatusUnauthorized, common.CodeUnauthenticated, "Autentikasi diperlukan")
				return
			}

			// SYSTEM_ADMIN selalu memiliki hak akses super
			if u.ActiveRole == "SYSTEM_ADMIN" {
				next.ServeHTTP(w, r)
				return
			}

			for _, requiredRole := range roles {
				if strings.EqualFold(u.ActiveRole, requiredRole) {
					next.ServeHTTP(w, r)
					return
				}
			}

			common.WriteV1Error(w, http.StatusForbidden, common.CodeForbidden, "Akses ditolak: peran Anda tidak memiliki wewenang untuk aksi ini")
		}
	}
}
