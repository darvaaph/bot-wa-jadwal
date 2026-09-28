package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const recoveryTTL = 1 * time.Hour

// UpdateRoleAssignmentStatus suspends, revokes, or reactivates an assignment.
// Allowed transitions: ACTIVE->SUSPENDED/REVOKED, SUSPENDED->ACTIVE/REVOKED.
func (s *Service) UpdateRoleAssignmentStatus(ctx context.Context, actor Principal, assignmentID int64, newStatus, reason string) error {
	newStatus = strings.ToUpper(strings.TrimSpace(newStatus))
	if newStatus != "ACTIVE" && newStatus != "SUSPENDED" && newStatus != "REVOKED" {
		return ErrInvalidInput
	}
	now := s.clock().UTC()
	target, err := s.getAssignment(ctx, assignmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	if target == nil || !s.canManageAssignment(actor, *target) {
		return ErrAccessDenied
	}
	if target.Status == newStatus {
		return nil
	}
	valid := (target.Status == "ACTIVE" && (newStatus == "SUSPENDED" || newStatus == "REVOKED")) ||
		(target.Status == "SUSPENDED" && (newStatus == "ACTIVE" || newStatus == "REVOKED"))
	if !valid {
		return ErrInvalidInput
	}
	// Guard: class must keep at least one active KM.
	if target.Role == RoleKM && target.ScopeType == ScopeClass && target.ClassID != nil &&
		target.Status == "ACTIVE" && newStatus != "ACTIVE" {
		var others int
		err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM role_assignments
			WHERE role='KM' AND scope_type='CLASS' AND class_id=? AND status='ACTIVE' AND id<>?
			AND valid_from<=? AND (valid_until IS NULL OR valid_until>?)`,
			*target.ClassID, target.ID, formatTime(now), formatTime(now)).Scan(&others)
		if err != nil {
			return err
		}
		if others == 0 {
			return errors.New("kelas harus memiliki minimal satu KM aktif")
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE role_assignments SET status=?, updated_at=? WHERE id=?`,
		newStatus, formatTime(now), assignmentID); err != nil {
		return err
	}
	// Revoke sessions bound to a non-active assignment.
	if newStatus != "ACTIVE" {
		if _, err := tx.ExecContext(ctx, `UPDATE user_sessions SET revoked_at=?, revocation_reason='ROLE_ASSIGNMENT_INACTIVE',
			updated_at=? WHERE active_role_assignment_id=? AND revoked_at IS NULL`,
			formatTime(now), formatTime(now), assignmentID); err != nil {
			return err
		}
	}
	corr, _ := s.newToken()
	var classVal, semVal any
	if target.ClassID != nil {
		classVal = *target.ClassID
	}
	if target.SemesterID != nil {
		semVal = *target.SemesterID
	}
	reasonVal := strings.TrimSpace(reason)
	var reasonAny any
	if reasonVal != "" {
		reasonAny = reasonVal
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, reason, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'USER', 'ASSIGN_ROLE', 'ROLE_ASSIGNMENT', ?, ?, ?, ?, ?, ?)`,
		classVal, semVal, actor.UserID, actor.RoleAssignmentID, assignmentID,
		`{"status":"`+newStatus+`"}`, reasonAny, corr, formatTime(now), formatTime(now),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) canManageAssignment(actor Principal, target RoleAssignment) bool {
	if actor.IsSystemAdmin() {
		return true
	}
	if actor.Role == RoleKM && actor.ScopeType == ScopeClass && actor.ClassID != nil {
		if target.Role == "PJ" && target.ClassID != nil && *target.ClassID == *actor.ClassID {
			return true
		}
	}
	return false
}

// ListRoleAssignments lists assignments visible to the actor.
func (s *Service) ListRoleAssignments(ctx context.Context, actor Principal, classID *int64) ([]RoleAssignment, error) {
	query := `SELECT id, user_id, role, scope_type, class_id, semester_id, course_offering_id,
		status, valid_from, valid_until, version FROM role_assignments WHERE 1=1`
	args := []any{}
	if !actor.IsSystemAdmin() {
		if actor.ClassID == nil {
			return []RoleAssignment{}, nil
		}
		query += ` AND class_id = ?`
		args = append(args, *actor.ClassID)
	} else if classID != nil {
		query += ` AND (class_id = ? OR class_id IS NULL)`
		args = append(args, *classID)
	}
	query += ` ORDER BY role, class_id, id LIMIT 500`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RoleAssignment{}
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// RequestRecovery creates a single-use recovery token. The raw token is returned
// to the caller, which must relay it over a verified channel (never in an
// unauthenticated API response). Unknown identities yield ErrAuthenticationFailed
// so public handlers can respond generically.
func (s *Service) RequestRecovery(ctx context.Context, identityKey, method, reason string) (string, error) {
	identity := NormalizeIdentity(identityKey)
	if identity == "" {
		return "", ErrInvalidInput
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = "ADMIN"
	}
	var userID int64
	err := s.db.QueryRowContext(ctx, `SELECT id FROM users WHERE identity_key=?`, identity).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		// Generic response to avoid enumeration; still return error for API to map to generic success.
		return "", ErrAuthenticationFailed
	}
	if err != nil {
		return "", err
	}
	now := s.clock().UTC()
	// Throttle: one active token per user; re-request within 5 minutes is rejected
	// to prevent token-spam invalidating legitimate tokens.
	var lastCreated string
	err = s.db.QueryRowContext(ctx, `SELECT created_at FROM recovery_tokens
		WHERE user_id = ? AND used_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`, userID, formatTime(now)).Scan(&lastCreated)
	if err == nil {
		if created, perr := parseTime(lastCreated); perr == nil && now.Sub(created) < 5*time.Minute {
			return "", ErrTooFrequent
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	token, err := s.newToken()
	if err != nil {
		return "", err
	}
	expires := now.Add(recoveryTTL)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE recovery_tokens SET used_at=? WHERE user_id=? AND used_at IS NULL`,
		formatTime(now), userID); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO recovery_tokens (user_id, token_hash, method, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, userID, tokenHash(token), method, formatTime(expires), formatTime(now), formatTime(now)); err != nil {
		return "", err
	}
	corr, _ := s.newToken()
	var reasonAny any
	if strings.TrimSpace(reason) != "" {
		reasonAny = strings.TrimSpace(reason)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		actor_user_id, actor_type, action, entity_type, entity_id, after_json, reason, correlation_id, created_at, updated_at
	) VALUES (?, 'USER', 'REQUEST_RECOVERY', 'USER', ?, ?, ?, ?, ?, ?)`,
		userID, userID, `{"method":"`+method+`"}`, reasonAny, corr, formatTime(now), formatTime(now)); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

// ConfirmRecovery resets password, bumps session_version (revoking all sessions), and consumes the token.
func (s *Service) ConfirmRecovery(ctx context.Context, token, newPassword string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidInput
	}
	now := s.clock().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id, userID int64
	var expires string
	var used sql.NullString
	var identity string
	err = tx.QueryRowContext(ctx, `SELECT rt.id, rt.user_id, rt.expires_at, rt.used_at, u.identity_key
		FROM recovery_tokens rt JOIN users u ON u.id = rt.user_id WHERE rt.token_hash=?`,
		tokenHash(token)).Scan(&id, &userID, &expires, &used, &identity)
	if errors.Is(err, sql.ErrNoRows) || used.Valid {
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	exp, err := parseTime(expires)
	if err != nil || !now.Before(exp) {
		return ErrInvalidInput
	}
	if err := ValidatePassword(identity, newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(identity, newPassword)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET password_hash=?, session_version=session_version+1, updated_at=? WHERE id=?`,
		hash, formatTime(now), userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE user_sessions SET revoked_at=?, revocation_reason='PASSWORD_RECOVERY', updated_at=?
		WHERE user_id=? AND revoked_at IS NULL`, formatTime(now), formatTime(now), userID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE recovery_tokens SET used_at=?, updated_at=? WHERE id=?`,
		formatTime(now), formatTime(now), id); err != nil {
		return err
	}
	corr, _ := s.newToken()
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		actor_user_id, actor_type, action, entity_type, entity_id, correlation_id, created_at, updated_at
	) VALUES (?, 'USER', 'CONFIRM_RECOVERY', 'USER', ?, ?, ?, ?)`,
		userID, userID, corr, formatTime(now), formatTime(now)); err != nil {
		return err
	}
	return tx.Commit()
}
