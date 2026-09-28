package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// generateSecureToken menghasilkan token acak 32-byte dengan awalan "bv1_"
func generateSecureToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "bv1_" + hex.EncodeToString(b), nil
}

// computeHash menghitung hash SHA-256 string
func computeHash(s string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(s)))
	return hex.EncodeToString(h[:])
}

// LoginRequest merepresentasikan payload login pengurus
type LoginRequest struct {
	IdentityKey string `json:"identity_key"`
	Password    string `json:"password"`
}

// RoleAssignmentItem merepresentasikan penugasan peran pengurus
type RoleAssignmentItem struct {
	ID         int64  `json:"id"`
	Role       string `json:"role"`
	ClassSlug  string `json:"class_slug,omitempty"`
	SemesterID *int64 `json:"semester_id,omitempty"`
	OfferingID *int64 `json:"offering_id,omitempty"`
}

// LoginResponse adalah payload data respons login sukses
type LoginResponse struct {
	Token             string               `json:"token"`
	TokenType         string               `json:"token_type"`
	ExpiresAt         time.Time            `json:"expires_at"`
	Assignments       []RoleAssignmentItem `json:"assignments"`
	NeedContextChoice bool                 `json:"need_context_choice"`
}

// handleLogin menangani POST /api/v1/auth/login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	cleanIdentity := strings.TrimSpace(req.IdentityKey)
	if cleanIdentity == "" || req.Password == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Nomor WhatsApp (identity_key) dan kata sandi wajib diisi")
		return
	}

	identityHash := computeHash(cleanIdentity)
	sourceHash := computeHash(r.RemoteAddr)

	// 1. Periksa batas percobaan gagal (Rate Limit: 5 gagal dalam 15 menit -> blokir 15 menit)
	var failedAttempts int
	_ = s.v1DB.QueryRow(`
		SELECT COUNT(*) FROM login_attempts
		WHERE identity_hash = ? AND attempted_at >= datetime('now', '-15 minutes') AND outcome = 'FAILED';
	`, identityHash).Scan(&failedAttempts)

	if failedAttempts >= 5 {
		_, _ = s.v1DB.Exec(`
			INSERT INTO login_attempts (identity_hash, source_hash, outcome)
			VALUES (?, ?, 'BLOCKED');
		`, identityHash, sourceHash)
		s.writeV1Error(w, http.StatusTooManyRequests, CodeTooManyRequests, "Terlalu banyak percobaan gagal. Akun diblokir sementara selama 15 menit demi keamanan.")
		return
	}

	// 2. Ambil data pengguna
	var (
		userID         int64
		identityKey    string
		displayName    string
		passwordHash   string
		status         string
		sessionVersion int
	)

	err := s.v1DB.QueryRow(`
		SELECT id, identity_key, display_name, password_hash, status, session_version
		FROM users
		WHERE identity_key = ?;
	`, cleanIdentity).Scan(&userID, &identityKey, &displayName, &passwordHash, &status, &sessionVersion)

	if err == sql.ErrNoRows {
		// Pesan generik agar tidak mengungkap keberadaan akun
		_, _ = s.v1DB.Exec(`INSERT INTO login_attempts (identity_hash, source_hash, outcome) VALUES (?, ?, 'FAILED');`, identityHash, sourceHash)
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Kredensial tidak valid")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memproses login")
		return
	}

	// 3. Verifikasi Password dengan bcrypt
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		_, _ = s.v1DB.Exec(`INSERT INTO login_attempts (user_id, identity_hash, source_hash, outcome) VALUES (?, ?, ?, 'FAILED');`, userID, identityHash, sourceHash)
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Kredensial tidak valid")
		return
	}

	// 4. Verifikasi status pengguna
	if status != "ACTIVE" {
		_, _ = s.v1DB.Exec(`INSERT INTO login_attempts (user_id, identity_hash, source_hash, outcome) VALUES (?, ?, ?, 'FAILED');`, userID, identityHash, sourceHash)
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Akun Anda saat ini dinonaktifkan")
		return
	}

	// 5. Catat login sukses
	_, _ = s.v1DB.Exec(`INSERT INTO login_attempts (user_id, identity_hash, source_hash, outcome) VALUES (?, ?, ?, 'SUCCESS');`, userID, identityHash, sourceHash)

	// 6. Ambil seluruh penugasan peran aktif pengguna
	rows, err := s.v1DB.Query(`
		SELECT ra.id, ra.role, c.slug, ra.semester_id, ra.course_offering_id
		FROM role_assignments ra
		LEFT JOIN classes c ON ra.class_id = c.id
		WHERE ra.user_id = ? AND ra.status = 'ACTIVE';
	`, userID)

	var assignments []RoleAssignmentItem
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var a RoleAssignmentItem
			var slug sql.NullString
			var semID, offID sql.NullInt64
			if err := rows.Scan(&a.ID, &a.Role, &slug, &semID, &offID); err == nil {
				if slug.Valid {
					a.ClassSlug = slug.String
				}
				if semID.Valid {
					a.SemesterID = &semID.Int64
				}
				if offID.Valid {
					a.OfferingID = &offID.Int64
				}
				assignments = append(assignments, a)
			}
		}
	}

	// Tentukan active_role_assignment_id awal
	var activeAssignmentID sql.NullInt64
	needContextChoice := len(assignments) > 1
	activeRole := "GUEST"

	if len(assignments) > 0 {
		activeAssignmentID = sql.NullInt64{Int64: assignments[0].ID, Valid: true}
		activeRole = assignments[0].Role
	}

	// Tentukan batas waktu absolut (Admin: 8 jam, PJ/KM: 24 jam)
	absTTL := 24 * time.Hour
	if activeRole == "SYSTEM_ADMIN" {
		absTTL = 8 * time.Hour
	}
	expiresAt := time.Now().Add(absTTL)

	// 7. Buat token sesi baru
	token, err := generateSecureToken()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token autentikasi")
		return
	}
	tokenHash := computeHash(token)

	_, err = s.v1DB.Exec(`
		INSERT INTO user_sessions (
			user_id, active_role_assignment_id, token_hash, session_version,
			created_at, last_seen_at, absolute_expires_at
		)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?);
	`, userID, activeAssignmentID, tokenHash, sessionVersion, expiresAt)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan sesi login")
		return
	}

	// Perbarui last_login_at
	_, _ = s.v1DB.Exec(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?;`, userID)

	// Set cookie bv1 (opsional sebagai fallback Alpine.js)
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.writeV1Success(w, http.StatusOK, LoginResponse{
		Token:             token,
		TokenType:         "Bearer",
		ExpiresAt:         expiresAt,
		Assignments:       assignments,
		NeedContextChoice: needContextChoice,
	})
}

// handleLogout menangani POST /api/v1/auth/logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	_, _ = s.v1DB.Exec(`
		UPDATE user_sessions
		SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'USER_LOGOUT'
		WHERE id = ?;
	`, u.SessionID)

	// Hapus cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.writeV1Success(w, http.StatusOK, map[string]bool{"revoked": true})
}

// handleGetMe menangani GET /api/v1/auth/me
func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	// Ambil daftar kelas yang dapat diakses pengguna
	var classes []map[string]any
	rows, err := s.v1DB.Query(`
		SELECT c.slug, c.code, c.study_program, c.cohort_year, c.group_label, c.status
		FROM classes c
		ORDER BY c.code;
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slug, code, prog, grp, st string
			var cohort int
			if err := rows.Scan(&slug, &code, &prog, &cohort, &grp, &st); err == nil {
				classes = append(classes, map[string]any{
					"slug":    slug,
					"code":    code,
					"program": prog,
					"cohort":  cohort,
					"group":   grp,
					"status":  st,
				})
			}
		}
	}

	userData := map[string]any{
		"id":           u.UserID,
		"identity_key": u.IdentityKey,
		"display_name": u.DisplayName,
		"status":       u.UserStatus,
	}

	var activeAssignmentData any
	if u.ActiveAssignmentID > 0 {
		activeAssignmentData = map[string]any{
			"id":         u.ActiveAssignmentID,
			"role":       u.ActiveRole,
			"scope_type": u.ActiveScopeType,
			"class_id":   u.ActiveClassID.Int64,
			"class_slug": u.ActiveClassSlug,
			"semester_id": func() any {
				if u.ActiveSemesterID.Valid {
					return u.ActiveSemesterID.Int64
				}
				return nil
			}(),
			"offering_id": func() any {
				if u.ActiveCourseOfferingID.Valid {
					return u.ActiveCourseOfferingID.Int64
				}
				return nil
			}(),
		}
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"user":              userData,
		"active_assignment": activeAssignmentData,
		"classes":           classes,
	})
}

// SwitchContextRequest adalah payload untuk ganti peran/konteks
type SwitchContextRequest struct {
	RoleAssignmentID int64 `json:"role_assignment_id"`
}

// handleSwitchContext menangani POST /api/v1/auth/switch-context
func (s *Server) handleSwitchContext(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req SwitchContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoleAssignmentID <= 0 {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "role_assignment_id wajib disertakan")
		return
	}

	// 1. Verifikasi bahwa role_assignment_id benar-benar milik pengguna ini dan aktif
	var role string
	err := s.v1DB.QueryRow(`
		SELECT role FROM role_assignments
		WHERE id = ? AND user_id = ? AND status = 'ACTIVE';
	`, req.RoleAssignmentID, u.UserID).Scan(&role)

	if err == sql.ErrNoRows {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "Penugasan peran tidak ditemukan atau Anda tidak memiliki akses")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi penugasan peran")
		return
	}

	// 2. Cabut sesi lama (rotasi token)
	_, _ = s.v1DB.Exec(`
		UPDATE user_sessions
		SET revoked_at = CURRENT_TIMESTAMP, revocation_reason = 'SWITCH_CONTEXT'
		WHERE id = ?;
	`, u.SessionID)

	// 3. Buat token baru dengan TTL sesuai peran baru
	absTTL := 24 * time.Hour
	if role == "SYSTEM_ADMIN" {
		absTTL = 8 * time.Hour
	}
	expiresAt := time.Now().Add(absTTL)

	newToken, err := generateSecureToken()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token baru")
		return
	}
	tokenHash := computeHash(newToken)

	_, err = s.v1DB.Exec(`
		INSERT INTO user_sessions (
			user_id, active_role_assignment_id, token_hash, session_version,
			created_at, last_seen_at, absolute_expires_at
		)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?);
	`, u.UserID, req.RoleAssignmentID, tokenHash, u.SessionVersion, expiresAt)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyimpan sesi baru")
		return
	}

	// Set cookie baru
	http.SetCookie(w, &http.Cookie{
		Name:     "bv1",
		Value:    newToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"token":      newToken,
		"token_type": "Bearer",
		"expires_at": expiresAt,
	})
}

// handleGetV1Classes menangani GET /api/v1/classes
func (s *Server) handleGetV1Classes(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	rows, err := s.v1DB.Query(`
		SELECT slug, code, study_program, cohort_year, group_label, status
		FROM classes
		ORDER BY code;
	`)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal mengambil daftar kelas")
		return
	}
	defer rows.Close()

	var classes []map[string]any
	for rows.Next() {
		var slug, code, prog, grp, st string
		var cohort int
		if err := rows.Scan(&slug, &code, &prog, &cohort, &grp, &st); err == nil {
			classes = append(classes, map[string]any{
				"slug":    slug,
				"code":    code,
				"program": prog,
				"cohort":  cohort,
				"group":   grp,
				"status":  st,
			})
		}
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"classes": classes,
	})
}

// InvitationRequest adalah payload pembuatan undangan
type InvitationRequest struct {
	Role               string `json:"role"` // KM atau PJ
	ClassSlug          string `json:"class_slug"`
	SemesterID         *int64 `json:"semester_id,omitempty"`
	OfferingID         *int64 `json:"offering_id,omitempty"`
	InvitedIdentityKey string `json:"invited_identity_key"`
}

// handleCreateInvitation menangani POST /api/v1/invitations
func (s *Server) handleCreateInvitation(w http.ResponseWriter, r *http.Request) {
	u, ok := GetAuthContext(r)
	if !ok {
		s.writeV1Error(w, http.StatusUnauthorized, CodeUnauthenticated, "Autentikasi diperlukan")
		return
	}

	var req InvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Payload JSON tidak valid")
		return
	}

	role := strings.ToUpper(strings.TrimSpace(req.Role))
	if role != "KM" && role != "PJ" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Peran undangan harus KM atau PJ")
		return
	}

	cleanIdentity := strings.TrimSpace(req.InvitedIdentityKey)
	if cleanIdentity == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Nomor WhatsApp (invited_identity_key) wajib diisi")
		return
	}

	// KM hanya boleh mengundang untuk kelasnya sendiri
	var classID int64
	err := s.v1DB.QueryRow(`SELECT id FROM classes WHERE slug = ?;`, req.ClassSlug).Scan(&classID)
	if err != nil {
		s.writeV1Error(w, http.StatusNotFound, CodeNotFound, "Kelas tidak ditemukan")
		return
	}

	if u.ActiveRole == "KM" && u.ActiveClassID.Valid && u.ActiveClassID.Int64 != classID {
		s.writeV1Error(w, http.StatusForbidden, CodeForbidden, "KM hanya berwenang membuat undangan untuk kelasnya sendiri")
		return
	}

	scopeType := "CLASS"
	if role == "PJ" {
		scopeType = "COURSE_OFFERING"
		if req.SemesterID == nil || req.OfferingID == nil {
			s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Undangan PJ wajib menyertakan semester_id dan offering_id")
			return
		}
	}

	// Revoke undangan PENDING sebelumnya untuk nomor identitas ini
	_, _ = s.v1DB.Exec(`
		UPDATE role_invitations
		SET status = 'REVOKED'
		WHERE invited_identity_key = ? AND status = 'PENDING';
	`, cleanIdentity)

	// Buat token undangan acak
	token, err := generateSecureToken()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "TOKEN_ERROR", "Gagal membuat token undangan")
		return
	}
	tokenHash := computeHash(token)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // Berlaku 7 hari

	var invID int64
	err = s.v1DB.QueryRow(`
		INSERT INTO role_invitations (
			token_hash, invited_identity_key, role, scope_type, class_id, semester_id,
			course_offering_id, status, expires_at, invited_by_user_id
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?)
		RETURNING id;
	`, tokenHash, cleanIdentity, role, scopeType, classID, req.SemesterID, req.OfferingID, expiresAt, u.UserID).Scan(&invID)

	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan undangan: %v", err))
		return
	}

	s.writeV1Success(w, http.StatusCreated, map[string]any{
		"invitation_id": invID,
		"token":         token,
		"expires_at":    expiresAt,
	})
}

// AcceptInvitationRequest adalah payload menerima undangan
type AcceptInvitationRequest struct {
	Token       string `json:"token"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
}

// handleAcceptInvitation menangani POST /api/v1/invitations/accept
func (s *Server) handleAcceptInvitation(w http.ResponseWriter, r *http.Request) {
	if s.v1DB == nil {
		s.writeV1Error(w, http.StatusInternalServerError, "SERVER_ERROR", "Database v1 belum siap")
		return
	}

	var req AcceptInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Token) == "" {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Token undangan wajib disertakan")
		return
	}

	if len(req.Password) < 6 {
		s.writeV1Error(w, http.StatusUnprocessableEntity, CodeValidation, "Kata sandi minimal 6 karakter")
		return
	}

	tokenHash := computeHash(req.Token)

	var (
		invID            int64
		identityKey      string
		role             string
		scopeType        string
		classID          sql.NullInt64
		semesterID       sql.NullInt64
		courseOfferingID sql.NullInt64
		status           string
		expiresAt        time.Time
	)

	err := s.v1DB.QueryRow(`
		SELECT id, invited_identity_key, role, scope_type, class_id, semester_id, course_offering_id, status, expires_at
		FROM role_invitations
		WHERE token_hash = ?;
	`, tokenHash).Scan(&invID, &identityKey, &role, &scopeType, &classID, &semesterID, &courseOfferingID, &status, &expiresAt)

	if err == sql.ErrNoRows || status != "PENDING" {
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Undangan tidak valid atau telah digunakan")
		return
	} else if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memverifikasi undangan")
		return
	}

	if time.Now().After(expiresAt) {
		_, _ = s.v1DB.Exec(`UPDATE role_invitations SET status = 'EXPIRED' WHERE id = ?;`, invID)
		s.writeV1Error(w, http.StatusBadRequest, CodeValidation, "Undangan telah kedaluwarsa")
		return
	}

	// Enkripsi kata sandi dengan bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "CRYPTO_ERROR", "Gagal memproses kata sandi")
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = identityKey
	}

	tx, err := s.v1DB.Begin()
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal memulai transaksi")
		return
	}
	defer tx.Rollback()

	// 1. Buat atau perbarui akun pengguna
	var userID int64
	err = tx.QueryRow(`
		INSERT INTO users (identity_key, display_name, password_hash, status)
		VALUES (?, ?, ?, 'ACTIVE')
		ON CONFLICT(identity_key) DO UPDATE SET
			password_hash = excluded.password_hash,
			display_name = excluded.display_name,
			status = 'ACTIVE'
		RETURNING id;
	`, identityKey, displayName, string(hashedPassword)).Scan(&userID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menyimpan akun: %v", err))
		return
	}

	// 2. Berikan role assignment sesuai cakupan undangan
	var assignmentID int64
	err = tx.QueryRow(`
		INSERT INTO role_assignments (
			user_id, role, scope_type, class_id, semester_id, course_offering_id,
			accepted_invitation_id, status
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE')
		RETURNING id;
	`, userID, role, scopeType, classID, semesterID, courseOfferingID, invID).Scan(&assignmentID)
	if err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", fmt.Sprintf("Gagal menetapkan peran: %v", err))
		return
	}

	// 3. Tandai undangan ACCEPTED (token sekali pakai)
	_, _ = tx.Exec(`UPDATE role_invitations SET status = 'ACCEPTED' WHERE id = ?;`, invID)

	if err := tx.Commit(); err != nil {
		s.writeV1Error(w, http.StatusInternalServerError, "DB_ERROR", "Gagal menyelesaikan proses penerimaan undangan")
		return
	}

	s.writeV1Success(w, http.StatusOK, map[string]any{
		"user_id":       userID,
		"assignment_id": assignmentID,
	})
}
