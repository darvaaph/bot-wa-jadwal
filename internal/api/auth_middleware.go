package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const userContextKey = contextKey("v1_user_context")

// UserContext menyimpan data sesi dan konteks penugasan peran aktif dari token pengguna.
type UserContext struct {
	SessionID              int64
	UserID                 int64
	IdentityKey            string
	DisplayName            string
	UserStatus             string
	SessionVersion         int
	ActiveAssignmentID     int64
	ActiveRole             string
	ActiveScopeType        string
	ActiveClassID          sql.NullInt64
	ActiveClassSlug        string
	ActiveSemesterID       sql.NullInt64
	ActiveCourseOfferingID sql.NullInt64
}

// GetAuthContext mengambil konteks pengguna terautentikasi dari request context.
func GetAuthContext(r *http.Request) (*UserContext, bool) {
	ctxVal := r.Context().Value(userContextKey)
	if ctxVal == nil {
		return nil, false
	}
	u, ok := ctxVal.(*UserContext)
	return u, ok
}

// extractToken mengekstrak token dari header Authorization: Bearer <token> atau cookie bv1.
func extractToken(r *http.Request) string {
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
func (s *Server) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.v1DB == nil {
			s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum diinisialisasi")
			return
		}

		rawToken := extractToken(r)
		if rawToken == "" {
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Token autentikasi tidak ditemukan. Silakan login terlebih dahulu.")
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

		err := s.v1DB.QueryRow(query, tokenHash).Scan(
			&sessionID, &userID, &activeRoleAssignmentID, &sessionVersion,
			&lastSeenAtStr, &absoluteExpiresAtStr, &revokedAtStr,
			&identityKey, &displayName, &userStatus, &userSessionVersion,
			&role, &scopeType, &classID, &classSlug, &semesterID, &courseOfferingID,
			&raValidFrom, &raValidUntil,
		)

		if err == sql.ErrNoRows {
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi tidak valid atau telah berakhir")
			return
		} else if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi sesi")
			return
		}

		lastSeenAt, err := parseTime(lastSeenAtStr)
		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Format last_seen_at tidak valid")
			return
		}
		absoluteExpiresAt, err := parseTime(absoluteExpiresAtStr)
		if err != nil {
			s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Format absolute_expires_at tidak valid")
			return
		}
		var revokedAt sql.NullTime
		if revokedAtStr.Valid {
			t, err := parseTime(revokedAtStr.String)
			if err == nil {
				revokedAt = sql.NullTime{Time: t, Valid: true}
			}
		}

		// 1. Cek pencabutan sesi
		if revokedAt.Valid {
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi telah dicabut. Silakan login kembali.")
			return
		}

		// Cek validitas penugasan peran jika sesi memiliki role assignment aktif
		if activeRoleAssignmentID.Valid && !role.Valid {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Peran Anda telah dinonaktifkan, ditangguhkan, atau dicabut")
			return
		}

		// 2. Cek status akun pengguna
		if userStatus != "ACTIVE" {
			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Akun Anda saat ini dinonaktifkan")
			return
		}

		// 3. Cek versi sesi pengguna (jika direset / rotasi global)
		if sessionVersion != userSessionVersion {
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi telah direset oleh pembaruan akun. Silakan login kembali.")
			return
		}

		now := time.Now()

		if role.Valid {
			if raValidFrom.Valid {
				vf, err := parseTime(raValidFrom.String)
				if err == nil && now.Before(vf) {
					s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Penugasan peran Anda belum aktif")
					return
				}
			}
			if raValidUntil.Valid {
				vu, err := parseTime(raValidUntil.String)
				if err == nil && now.After(vu) {
					s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Penugasan peran Anda telah kedaluwarsa")
					return
				}
			}
		}

		// 4. Cek batas kedaluwarsa absolut
		if now.After(absoluteExpiresAt) {
			_, _ = s.v1DB.Exec(`UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'ABSOLUTE_TIMEOUT' WHERE id = ?;`, sessionID)
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi Anda telah kedaluwarsa (batas waktu absolut terlewati)")
			return
		}

		// 5. Cek batas kedaluwarsa idle (ADR-0003: Admin 30 menit, PJ/KM 2 jam)
		idleLimit := 2 * time.Hour
		if role.Valid && role.String == "SYSTEM_ADMIN" {
			idleLimit = 30 * time.Minute
		}

		if now.Sub(lastSeenAt) > idleLimit {
			_, _ = s.v1DB.Exec(`UPDATE user_sessions SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'IDLE_TIMEOUT' WHERE id = ?;`, sessionID)
			s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Sesi Anda telah kedaluwarsa karena tidak ada aktivitas")
			return
		}

		// 6. Perbarui waktu aktivitas terakhir (last_seen_at)
		_, _ = s.v1DB.Exec(`UPDATE user_sessions SET last_seen_at = CURRENT_TIMESTAMP WHERE id = ?;`, sessionID)

		// 7. Bentuk UserContext dan simpan ke request context
		userCtx := &UserContext{
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

		ctx := context.WithValue(r.Context(), userContextKey, userCtx)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// RequireRole memeriksa apakah peran pengguna saat ini termasuk salah satu dari peran yang diizinkan.
func (s *Server) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			u, ok := GetAuthContext(r)
			if !ok {
				s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
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

			s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Akses ditolak: peran Anda tidak memiliki wewenang untuk aksi ini")
		}
	}
}

func parseTime(raw string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, fmt.Errorf("timestamp kosong")
	}
	if idx := strings.Index(raw, " m="); idx != -1 {
		raw = raw[:idx]
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999 -0700 -0700",
		"2006-01-02 15:04:05.999999999 -0700 +07",
		"2006-01-02 15:04:05.999999999 -0700",
		"2006-01-02 15:04:05.999999 -0700 +07",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("format timestamp %q tidak didukung", raw)
}
