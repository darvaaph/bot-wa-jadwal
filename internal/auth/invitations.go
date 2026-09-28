package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const invitationTTL = 7 * 24 * time.Hour

type InvitationInput struct {
	Role             string
	ClassID          *int64
	SemesterID       *int64
	CourseOfferingID *int64
	IdentityKey      string
}

type Invitation struct {
	ID               int64
	Role             string
	ScopeType        string
	ClassID          *int64
	SemesterID       *int64
	CourseOfferingID *int64
	IdentityKey      string
	Status           string
	ExpiresAt        time.Time
	InvitedByUserID  int64
	CreatedAt        time.Time
}

func scopeForRole(role string) string {
	switch role {
	case RoleSystemAdmin:
		return ScopeGlobal
	case RoleKM:
		return ScopeClass
	case RolePJ:
		return ScopeCourseOffering
	}
	return ""
}

func (s *Service) checkInvitePermission(inviter Principal, in InvitationInput) error {
	if inviter.IsSystemAdmin() {
		return nil
	}
	if in.Role == "PJ" && inviter.Role == RoleKM && inviter.ScopeType == ScopeClass &&
		inviter.ClassID != nil && in.ClassID != nil && *inviter.ClassID == *in.ClassID {
		return nil
	}
	return ErrAccessDenied
}

func validateInvitationScope(in InvitationInput) (string, error) {
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	scope := scopeForRole(role)
	if scope == "" {
		return "", ErrInvalidInput
	}
	switch role {
	case RoleSystemAdmin:
		if in.ClassID != nil || in.SemesterID != nil || in.CourseOfferingID != nil {
			return "", ErrInvalidInput
		}
	case RoleKM:
		if in.ClassID == nil || *in.ClassID <= 0 || in.SemesterID != nil || in.CourseOfferingID != nil {
			return "", ErrInvalidInput
		}
	case RolePJ:
		if in.ClassID == nil || *in.ClassID <= 0 || in.SemesterID == nil || *in.SemesterID <= 0 ||
			in.CourseOfferingID == nil || *in.CourseOfferingID <= 0 {
			return "", ErrInvalidInput
		}
	}
	return scope, nil
}

// CreateInvitation creates a single-use invitation token. Returns raw token (show once).
func (s *Service) CreateInvitation(ctx context.Context, inviter Principal, in InvitationInput) (*Invitation, string, error) {
	role := strings.ToUpper(strings.TrimSpace(in.Role))
	in.Role = role
	scope, err := validateInvitationScope(in)
	if err != nil {
		return nil, "", err
	}
	identity := NormalizeIdentity(in.IdentityKey)
	if identity == "" {
		return nil, "", ErrInvalidInput
	}
	if err := s.checkInvitePermission(inviter, in); err != nil {
		return nil, "", err
	}
	if role == "PJ" {
		var valid bool
		err := s.db.QueryRowContext(ctx, `SELECT EXISTS(
			SELECT 1 FROM course_offerings co JOIN semesters sem ON sem.id = co.semester_id
			WHERE co.id = ? AND co.semester_id = ? AND sem.class_id = ?
		)`, *in.CourseOfferingID, *in.SemesterID, *in.ClassID).Scan(&valid)
		if err != nil || !valid {
			return nil, "", ErrAccessDenied
		}
	}
	if role == "KM" {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM classes WHERE id = ?)`, *in.ClassID).Scan(&exists); err != nil || !exists {
			return nil, "", ErrAccessDenied
		}
	}

	now := s.clock().UTC()
	token, err := s.newToken()
	if err != nil {
		return nil, "", err
	}
	expires := now.Add(invitationTTL)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", err
	}
	defer tx.Rollback()

	// Revoke prior pending invitations for same identity+scope (resend semantics).
	revRows, err := tx.QueryContext(ctx, `SELECT id, class_id, semester_id FROM role_invitations
		WHERE invited_identity_key=? AND role=? AND status='PENDING'
		AND COALESCE(class_id,0)=COALESCE(?,0) AND COALESCE(semester_id,0)=COALESCE(?,0)
		AND COALESCE(course_offering_id,0)=COALESCE(?,0)`,
		identity, role, in.ClassID, in.SemesterID, in.CourseOfferingID)
	if err != nil {
		return nil, "", err
	}
	type revokedInvite struct {
		id         int64
		classID    sql.NullInt64
		semesterID sql.NullInt64
	}
	var revoked []revokedInvite
	for revRows.Next() {
		var ri revokedInvite
		if err := revRows.Scan(&ri.id, &ri.classID, &ri.semesterID); err != nil {
			revRows.Close()
			return nil, "", err
		}
		revoked = append(revoked, ri)
	}
	revRows.Close()
	if err := revRows.Err(); err != nil {
		return nil, "", err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE role_invitations SET status='REVOKED',
		revoked_at=?, updated_at=? WHERE invited_identity_key=? AND role=? AND status='PENDING'
		AND COALESCE(class_id,0)=COALESCE(?,0) AND COALESCE(semester_id,0)=COALESCE(?,0)
		AND COALESCE(course_offering_id,0)=COALESCE(?,0)`,
		formatTime(now), formatTime(now), identity, role, in.ClassID, in.SemesterID, in.CourseOfferingID); err != nil {
		return nil, "", err
	}

	var id int64
	err = tx.QueryRowContext(ctx, `INSERT INTO role_invitations (
		token_hash, invited_identity_key, role, scope_type, class_id, semester_id,
		course_offering_id, status, expires_at, invited_by_user_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, 'PENDING', ?, ?, ?, ?) RETURNING id`,
		tokenHash(token), identity, role, scope, in.ClassID, in.SemesterID, in.CourseOfferingID,
		formatTime(expires), inviter.UserID, formatTime(now), formatTime(now),
	).Scan(&id)
	if err != nil {
		return nil, "", err
	}
	corr, _ := s.newToken()
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'USER', 'INVITE', 'ROLE_INVITATION', ?, ?, ?, ?, ?)`,
		in.ClassID, in.SemesterID, inviter.UserID, inviter.RoleAssignmentID,
		id, `{"role":"`+role+`","identity":"`+identity+`"}`, corr, formatTime(now), formatTime(now),
	); err != nil {
		return nil, "", err
	}
	for _, ri := range revoked {
		var rc, rs any
		if ri.classID.Valid {
			rc = ri.classID.Int64
		}
		if ri.semesterID.Valid {
			rs = ri.semesterID.Int64
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
			class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
			action, entity_type, entity_id, reason, correlation_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, 'USER', 'REVOKE', 'ROLE_INVITATION', ?, ?, ?, ?, ?)`,
			rc, rs, inviter.UserID, inviter.RoleAssignmentID,
			ri.id, "superseded by resend", corr, formatTime(now), formatTime(now),
		); err != nil {
			return nil, "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "", err
	}
	return &Invitation{
		ID: id, Role: role, ScopeType: scope, ClassID: in.ClassID, SemesterID: in.SemesterID,
		CourseOfferingID: in.CourseOfferingID, IdentityKey: identity, Status: "PENDING",
		ExpiresAt: expires, InvitedByUserID: inviter.UserID, CreatedAt: now,
	}, token, nil
}

func scanInvitation(row interface{ Scan(...any) error }) (Invitation, error) {
	var inv Invitation
	var classID, semesterID, offeringID sql.NullInt64
	var expires, created string
	err := row.Scan(&inv.ID, &inv.Role, &inv.ScopeType, &classID, &semesterID, &offeringID,
		&inv.IdentityKey, &inv.Status, &expires, &inv.InvitedByUserID, &created)
	if err != nil {
		return inv, err
	}
	inv.ClassID = nullInt64Ptr(classID)
	inv.SemesterID = nullInt64Ptr(semesterID)
	inv.CourseOfferingID = nullInt64Ptr(offeringID)
	if inv.ExpiresAt, err = parseTime(expires); err != nil {
		return inv, err
	}
	if inv.CreatedAt, err = parseTime(created); err != nil {
		return inv, err
	}
	return inv, nil
}

// GetInvitationByToken returns invitation metadata for the accept page (no secret).
func (s *Service) GetInvitationByToken(ctx context.Context, token string) (*Invitation, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidInput
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, role, scope_type, class_id, semester_id,
		course_offering_id, invited_identity_key, status, expires_at, invited_by_user_id, created_at
		FROM role_invitations WHERE token_hash = ?`, tokenHash(token))
	inv, err := scanInvitation(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrInvalidInput
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// AcceptInvitation creates user (or reuses matching identity) + role assignment atomically.
func (s *Service) AcceptInvitation(ctx context.Context, token, displayName, password string) (*User, *RoleAssignment, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil, ErrInvalidInput
	}
	now := s.clock().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()

	var invID int64
	var role, scope, identity, status, expires string
	var classID, semesterID, offeringID sql.NullInt64
	var invitedBy int64
	err = tx.QueryRowContext(ctx, `SELECT id, role, scope_type, invited_identity_key, status,
		expires_at, class_id, semester_id, course_offering_id, invited_by_user_id
		FROM role_invitations WHERE token_hash = ?`, tokenHash(token)).
		Scan(&invID, &role, &scope, &identity, &status, &expires, &classID, &semesterID, &offeringID, &invitedBy)
	if errors.Is(err, sql.ErrNoRows) || status != "PENDING" {
		return nil, nil, ErrInvalidInput
	}
	if err != nil {
		return nil, nil, err
	}
	exp, err := parseTime(expires)
	if err != nil || !now.Before(exp) {
		_, _ = tx.ExecContext(ctx, `UPDATE role_invitations SET status='EXPIRED', updated_at=? WHERE id=?`, formatTime(now), invID)
		_ = tx.Commit()
		return nil, nil, ErrInvalidInput
	}
	if err := ValidatePassword(identity, password); err != nil {
		return nil, nil, err
	}
	passwordHash, err := HashPassword(identity, password)
	if err != nil {
		return nil, nil, err
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		name = identity
	}

	var userID int64
	var sessionVersion int
	err = tx.QueryRowContext(ctx, `SELECT id, session_version FROM users WHERE identity_key = ?`, identity).Scan(&userID, &sessionVersion)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `INSERT INTO users (identity_key, display_name, password_hash, status, session_version, created_at, updated_at)
			VALUES (?, ?, ?, 'ACTIVE', 1, ?, ?) RETURNING id`, identity, name, passwordHash, formatTime(now), formatTime(now)).Scan(&userID)
		if err != nil {
			return nil, nil, err
		}
		sessionVersion = 1
	} else if err != nil {
		return nil, nil, err
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET display_name=?, password_hash=?, status='ACTIVE', updated_at=? WHERE id=?`,
			name, passwordHash, formatTime(now), userID); err != nil {
			return nil, nil, err
		}
	}

	var assignmentID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO role_assignments (
		user_id, role, scope_type, class_id, semester_id, course_offering_id,
		accepted_invitation_id, status, valid_from, version, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, 'ACTIVE', ?, 1, ?, ?) RETURNING id`,
		userID, role, scope, nullableInt(classID), nullableInt(semesterID), nullableInt(offeringID),
		invID, formatTime(now), formatTime(now), formatTime(now),
	).Scan(&assignmentID)
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE role_invitations SET status='ACCEPTED', accepted_at=?, updated_at=? WHERE id=?`,
		formatTime(now), formatTime(now), invID); err != nil {
		return nil, nil, err
	}
	corr, _ := s.newToken()
	var classVal, semVal any
	if classID.Valid {
		classVal = classID.Int64
	}
	if semesterID.Valid {
		semVal = semesterID.Int64
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, after_json, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'USER', 'ACCEPT_INVITE', 'ROLE_ASSIGNMENT', ?, ?, ?, ?, ?)`,
		classVal, semVal, userID, assignmentID, assignmentID,
		`{"role":"`+role+`"}`, corr, formatTime(now), formatTime(now),
	); err != nil {
		return nil, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	user := &User{ID: userID, IdentityKey: identity, DisplayName: name, Status: "ACTIVE", SessionVersion: sessionVersion}
	assignment := &RoleAssignment{ID: assignmentID, UserID: userID, Role: role, ScopeType: scope, Status: "ACTIVE", ValidFrom: now, Version: 1}
	assignment.ClassID = nullInt64Ptr(classID)
	assignment.SemesterID = nullInt64Ptr(semesterID)
	assignment.CourseOfferingID = nullInt64Ptr(offeringID)
	return user, assignment, nil
}

func nullableInt(n sql.NullInt64) any {
	if !n.Valid {
		return nil
	}
	return n.Int64
}

// RevokeInvitation cancels a pending invitation.
func (s *Service) RevokeInvitation(ctx context.Context, actor Principal, invitationID int64) error {
	now := s.clock().UTC()
	var role, status string
	var classID, semesterID sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT role, status, class_id, semester_id FROM role_invitations WHERE id=?`, invitationID).
		Scan(&role, &status, &classID, &semesterID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidInput
	}
	if err != nil {
		return err
	}
	if status != "PENDING" {
		return ErrInvalidInput
	}
	if !actor.IsSystemAdmin() {
		if !(actor.Role == RoleKM && role == "PJ" && actor.ClassID != nil && classID.Valid && *actor.ClassID == classID.Int64) {
			return ErrAccessDenied
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `UPDATE role_invitations SET status='REVOKED', revoked_at=?, updated_at=?
		WHERE id=? AND status='PENDING'`, formatTime(now), formatTime(now), invitationID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return ErrInvalidInput
	}
	corr, _ := s.newToken()
	var rc, rs any
	if classID.Valid {
		rc = classID.Int64
	}
	if semesterID.Valid {
		rs = semesterID.Int64
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		class_id, semester_id, actor_user_id, actor_role_assignment_id, actor_type,
		action, entity_type, entity_id, correlation_id, created_at, updated_at
	) VALUES (?, ?, ?, ?, 'USER', 'REVOKE', 'ROLE_INVITATION', ?, ?, ?, ?)`,
		rc, rs, actor.UserID, actor.RoleAssignmentID, invitationID, corr, formatTime(now), formatTime(now),
	); err != nil {
		return err
	}
	return tx.Commit()
}

// ResendInvitation revokes the old pending invite and issues a new token with same scope.
func (s *Service) ResendInvitation(ctx context.Context, actor Principal, invitationID int64) (string, error) {
	var role, identity string
	var classID, semesterID, offeringID sql.NullInt64
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT role, invited_identity_key, class_id, semester_id,
		course_offering_id, status FROM role_invitations WHERE id=?`, invitationID).
		Scan(&role, &identity, &classID, &semesterID, &offeringID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrInvalidInput
	}
	if err != nil {
		return "", err
	}
	if status != "PENDING" {
		return "", ErrInvalidInput
	}
	in := InvitationInput{Role: role, IdentityKey: identity}
	if classID.Valid {
		v := classID.Int64
		in.ClassID = &v
	}
	if semesterID.Valid {
		v := semesterID.Int64
		in.SemesterID = &v
	}
	if offeringID.Valid {
		v := offeringID.Int64
		in.CourseOfferingID = &v
	}
	_, token, err := s.CreateInvitation(ctx, actor, in)
	return token, err
}

// ListInvitations lists invites visible to the actor.
func (s *Service) ListInvitations(ctx context.Context, actor Principal, classID *int64) ([]Invitation, error) {
	query := `SELECT id, role, scope_type, class_id, semester_id, course_offering_id,
		invited_identity_key, status, expires_at, invited_by_user_id, created_at
		FROM role_invitations WHERE 1=1`
	args := []any{}
	if !actor.IsSystemAdmin() {
		if actor.ClassID == nil {
			return []Invitation{}, nil
		}
		query += ` AND class_id = ?`
		args = append(args, *actor.ClassID)
	} else if classID != nil {
		query += ` AND class_id = ?`
		args = append(args, *classID)
	}
	query += ` ORDER BY created_at DESC, id DESC LIMIT 200`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Invitation{}
	for rows.Next() {
		inv, err := scanInvitation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}
