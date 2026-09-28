package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	loginWindow      = 15 * time.Minute
	loginBlockPeriod = 15 * time.Minute
	maxLoginFailures = 5
	normalIdle       = 2 * time.Hour
	normalAbsolute   = 24 * time.Hour
	adminIdle        = 30 * time.Minute
	adminAbsolute    = 8 * time.Hour
	tokenBytes       = 32
)

var provisionMu sync.Mutex

type Config struct {
	HashKey []byte
	Clock   func() time.Time
	Random  io.Reader
}

type Service struct {
	db                *sql.DB
	hashKey           []byte
	clock             func() time.Time
	random            io.Reader
	dummyPasswordHash string
}

func NewService(db *sql.DB, config Config) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: database wajib diisi", ErrInvalidInput)
	}
	if len(config.HashKey) < 32 {
		return nil, fmt.Errorf("%w: hash key minimal 32 byte", ErrInvalidInput)
	}
	clock := config.Clock
	if clock == nil {
		clock = time.Now
	}
	random := config.Random
	if random == nil {
		random = rand.Reader
	}
	dummyPasswordHash, err := bcryptHashForDummyPassword()
	if err != nil {
		return nil, err
	}
	return &Service{
		db: db, hashKey: append([]byte(nil), config.HashKey...), clock: clock,
		random: random, dummyPasswordHash: dummyPasswordHash,
	}, nil
}

func (s *Service) ProvisionInitialSystemAdmin(ctx context.Context, in ProvisionInput) (*User, *RoleAssignment, error) {
	identity := NormalizeIdentity(in.IdentityKey)
	if identity == "" || strings.TrimSpace(in.DisplayName) == "" {
		return nil, nil, ErrInvalidInput
	}
	passwordHash, err := HashPassword(identity, in.Password)
	if err != nil {
		return nil, nil, err
	}

	provisionMu.Lock()
	defer provisionMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("memulai provisioning: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM role_assignments WHERE role = 'SYSTEM_ADMIN'
	)`).Scan(&exists); err != nil {
		return nil, nil, fmt.Errorf("memeriksa system admin awal: %w", err)
	}
	if exists {
		return nil, nil, ErrAlreadyProvisioned
	}

	now := s.clock().UTC()
	var userID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO users (
		identity_key, display_name, password_hash, status, session_version, created_at, updated_at
	) VALUES (?, ?, ?, 'ACTIVE', 1, ?, ?) RETURNING id`,
		identity, strings.TrimSpace(in.DisplayName), passwordHash, formatTime(now), formatTime(now),
	).Scan(&userID)
	if err != nil {
		return nil, nil, fmt.Errorf("membuat system admin awal: %w", err)
	}

	var assignmentID int64
	err = tx.QueryRowContext(ctx, `INSERT INTO role_assignments (
		user_id, role, scope_type, status, valid_from, version, created_at, updated_at
	) VALUES (?, 'SYSTEM_ADMIN', 'GLOBAL', 'ACTIVE', ?, 1, ?, ?) RETURNING id`,
		userID, formatTime(now), formatTime(now), formatTime(now),
	).Scan(&assignmentID)
	if err != nil {
		return nil, nil, fmt.Errorf("membuat assignment system admin awal: %w", err)
	}
	correlationID, err := s.newToken()
	if err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		actor_user_id, actor_role_assignment_id, actor_context_json, actor_type,
		action, entity_type, entity_id, after_json, reason, correlation_id,
		created_at, updated_at
	) VALUES (?, ?, ?, 'USER', 'PROVISION_SYSTEM_ADMIN', 'USER', ?, ?, ?, ?, ?, ?)`,
		userID, assignmentID,
		`{"role":"SYSTEM_ADMIN","scope_type":"GLOBAL","provisioning":true}`,
		userID, `{"status":"ACTIVE","session_version":1}`,
		"initial installation provisioning", correlationID, formatTime(now), formatTime(now),
	); err != nil {
		return nil, nil, fmt.Errorf("mencatat audit provisioning: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, fmt.Errorf("commit provisioning: %w", err)
	}
	user := &User{ID: userID, IdentityKey: identity, DisplayName: strings.TrimSpace(in.DisplayName), Status: "ACTIVE", SessionVersion: 1}
	assignment := &RoleAssignment{ID: assignmentID, UserID: userID, Role: RoleSystemAdmin, ScopeType: ScopeGlobal, Status: "ACTIVE", ValidFrom: now, Version: 1}
	return user, assignment, nil
}

func (s *Service) Authenticate(ctx context.Context, in LoginInput) (*AuthenticatedIdentity, error) {
	identity := NormalizeIdentity(in.IdentityKey)
	source := strings.TrimSpace(in.Source)
	identityHash := keyedHash(s.hashKey, "identity", identity)
	sourceHash := keyedHash(s.hashKey, "source", source)
	now := s.clock().UTC()

	blocked, err := s.loginBlocked(ctx, identityHash, sourceHash, now)
	if err != nil {
		return nil, fmt.Errorf("memeriksa pembatasan login: %w", err)
	}
	if identity == "" || source == "" {
		if err := s.recordLoginAttempt(ctx, nil, identityHash, sourceHash, "FAILURE", now); err != nil {
			return nil, fmt.Errorf("mencatat percobaan login: %w", err)
		}
		return nil, ErrAuthenticationFailed
	}
	if blocked {
		if err := s.recordLoginAttempt(ctx, nil, identityHash, sourceHash, "BLOCKED", now); err != nil {
			return nil, fmt.Errorf("mencatat percobaan login: %w", err)
		}
		return nil, ErrAuthenticationFailed
	}

	var user User
	var passwordHash string
	var lastLogin sql.NullString
	err = s.db.QueryRowContext(ctx, `SELECT id, identity_key, display_name, password_hash,
		status, session_version, last_login_at
		FROM users WHERE identity_key = ?`, identity).Scan(
		&user.ID, &user.IdentityKey, &user.DisplayName, &passwordHash,
		&user.Status, &user.SessionVersion, &lastLogin,
	)
	found := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("membaca akun: %w", err)
	}
	if !found {
		passwordHash = s.dummyPasswordHash
	}
	passwordMatches := VerifyPassword(passwordHash, in.Password)
	valid := found && user.Status == "ACTIVE" && passwordMatches
	if !valid {
		var userID *int64
		if err == nil {
			userID = &user.ID
		}
		if err := s.recordLoginAttempt(ctx, userID, identityHash, sourceHash, "FAILURE", now); err != nil {
			return nil, fmt.Errorf("mencatat percobaan login: %w", err)
		}
		return nil, ErrAuthenticationFailed
	}

	assignments, err := s.ListActiveAssignments(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if err := s.recordSuccessfulLogin(ctx, user.ID, identityHash, sourceHash, now); err != nil {
		return nil, err
	}
	user.LastLoginAt = &now
	return &AuthenticatedIdentity{User: user, Assignments: assignments}, nil
}

func (s *Service) loginBlocked(ctx context.Context, identityHash, sourceHash string, now time.Time) (bool, error) {
	var failures int
	windowStart := formatTime(now.Add(-loginWindow))
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM login_attempts
		WHERE identity_hash = ? AND source_hash = ? AND outcome = 'FAILURE'
		AND attempted_at >= ?
		AND id > COALESCE((
			SELECT MAX(id) FROM login_attempts
			WHERE identity_hash = ? AND source_hash = ? AND outcome = 'SUCCESS'
		), 0)`, identityHash, sourceHash, windowStart, identityHash, sourceHash).Scan(&failures)
	if err != nil {
		return false, err
	}
	if failures < maxLoginFailures {
		return false, nil
	}
	var lastFailure string
	err = s.db.QueryRowContext(ctx, `SELECT attempted_at FROM login_attempts
		WHERE identity_hash = ? AND source_hash = ? AND outcome = 'FAILURE'
		ORDER BY attempted_at DESC, id DESC LIMIT 1`, identityHash, sourceHash).Scan(&lastFailure)
	if err != nil {
		return false, err
	}
	parsed, err := parseTime(lastFailure)
	if err != nil {
		return false, err
	}
	return now.Before(parsed.Add(loginBlockPeriod)), nil
}

func (s *Service) recordLoginAttempt(ctx context.Context, userID *int64, identityHash, sourceHash, outcome string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO login_attempts (
		user_id, identity_hash, source_hash, outcome, attempted_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, identityHash, sourceHash, outcome, formatTime(now), formatTime(now), formatTime(now))
	return err
}

func (s *Service) recordSuccessfulLogin(ctx context.Context, userID int64, identityHash, sourceHash string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO login_attempts (
		user_id, identity_hash, source_hash, outcome, attempted_at, created_at, updated_at
	) VALUES (?, ?, ?, 'SUCCESS', ?, ?, ?)`, userID, identityHash, sourceHash, formatTime(now), formatTime(now), formatTime(now)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET last_login_at = ?, updated_at = ? WHERE id = ?`, formatTime(now), formatTime(now), userID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ListActiveAssignments(ctx context.Context, userID int64) ([]RoleAssignment, error) {
	now := formatTime(s.clock().UTC())
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, role, scope_type, class_id,
		semester_id, course_offering_id, status, valid_from, valid_until, version
		FROM role_assignments
		WHERE user_id = ? AND status = 'ACTIVE' AND valid_from <= ?
		AND (valid_until IS NULL OR valid_until > ?)
		ORDER BY role, class_id, course_offering_id, id`, userID, now, now)
	if err != nil {
		return nil, fmt.Errorf("membaca assignment aktif: %w", err)
	}
	defer rows.Close()

	assignments := make([]RoleAssignment, 0)
	for rows.Next() {
		assignment, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterasi assignment aktif: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("menutup hasil assignment aktif: %w", err)
	}
	validAssignments := make([]RoleAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		if err := s.validateAssignmentScope(ctx, assignment); err == nil {
			validAssignments = append(validAssignments, assignment)
		} else if !errors.Is(err, ErrAccessDenied) {
			return nil, err
		}
	}
	return validAssignments, nil
}

func (s *Service) CreateSession(ctx context.Context, userID, assignmentID int64) (*SessionCredential, error) {
	now := s.clock().UTC()
	user, assignment, err := s.loadActiveContext(ctx, userID, assignmentID, now)
	if err != nil {
		return nil, err
	}
	token, err := s.newToken()
	if err != nil {
		return nil, err
	}
	_, absolute := sessionLimits(assignment.Role)
	expires := now.Add(absolute)
	var sessionID int64
	err = s.db.QueryRowContext(ctx, `INSERT INTO user_sessions (
		user_id, active_role_assignment_id, token_hash, session_version,
		last_seen_at, absolute_expires_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`,
		user.ID, assignment.ID, tokenHash(token), user.SessionVersion,
		formatTime(now), formatTime(expires), formatTime(now), formatTime(now),
	).Scan(&sessionID)
	if err != nil {
		return nil, fmt.Errorf("membuat sesi: %w", err)
	}
	return &SessionCredential{Token: token, Session: Session{ID: sessionID, UserID: user.ID, ActiveRoleAssignmentID: assignment.ID, LastSeenAt: now, AbsoluteExpiresAt: expires, CreatedAt: now}}, nil
}

func (s *Service) ValidateSession(ctx context.Context, token string) (*Principal, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidSession
	}
	now := s.clock().UTC()
	session, user, assignment, revokedAt, err := s.loadSessionContext(ctx, tokenHash(token))
	if err != nil {
		return nil, err
	}
	idle, _ := sessionLimits(assignment.Role)
	scopeErr := s.validateAssignmentScope(ctx, assignment)
	if scopeErr != nil && !errors.Is(scopeErr, ErrAccessDenied) {
		return nil, scopeErr
	}
	invalidReason := ""
	switch {
	case revokedAt != nil:
		invalidReason = "REVOKED"
	case user.Status != "ACTIVE":
		invalidReason = "USER_INACTIVE"
	case user.SessionVersion != session.sessionVersion:
		invalidReason = "SESSION_VERSION_CHANGED"
	case !now.Before(session.AbsoluteExpiresAt):
		invalidReason = "ABSOLUTE_EXPIRED"
	case !now.Before(session.LastSeenAt.Add(idle)):
		invalidReason = "IDLE_EXPIRED"
	case !assignmentEffective(assignment, now):
		invalidReason = "ASSIGNMENT_INACTIVE"
	case assignment.UserID != user.ID:
		invalidReason = "ASSIGNMENT_USER_MISMATCH"
	case scopeErr != nil:
		invalidReason = "ASSIGNMENT_SCOPE_INVALID"
	}
	if invalidReason != "" {
		if revokedAt == nil {
			_ = s.revokeSessionByID(ctx, session.ID, invalidReason, now)
		}
		return nil, ErrInvalidSession
	}
	result, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET last_seen_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`, formatTime(now), formatTime(now), session.ID)
	if err != nil {
		return nil, fmt.Errorf("memperbarui aktivitas sesi: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("memeriksa pembaruan aktivitas sesi: %w", err)
	}
	if affected != 1 {
		return nil, ErrInvalidSession
	}
	return principalFrom(user, assignment, session), nil
}

func (s *Service) SwitchContext(ctx context.Context, token string, assignmentID int64) (*SessionCredential, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrInvalidSession
	}
	principal, err := s.ValidateSession(ctx, token)
	if err != nil {
		return nil, err
	}
	now := s.clock().UTC()
	session, user, _, revokedAt, err := s.loadSessionContext(ctx, tokenHash(token))
	if err != nil {
		return nil, err
	}
	if revokedAt != nil || principal.SessionID != session.ID {
		return nil, ErrInvalidSession
	}
	_, target, err := s.loadActiveContext(ctx, user.ID, assignmentID, now)
	if err != nil {
		return nil, err
	}
	newToken, err := s.newToken()
	if err != nil {
		return nil, err
	}
	_, maximum := sessionLimits(target.Role)
	newAbsolute := session.CreatedAt.Add(maximum)
	if session.AbsoluteExpiresAt.Before(newAbsolute) {
		newAbsolute = session.AbsoluteExpiresAt
	}
	if !now.Before(newAbsolute) {
		return nil, ErrInvalidSession
	}
	result, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET
		active_role_assignment_id = ?, token_hash = ?, last_seen_at = ?,
		absolute_expires_at = ?, updated_at = ?
		WHERE id = ? AND token_hash = ? AND revoked_at IS NULL
		AND EXISTS (
			SELECT 1 FROM users u WHERE u.id = user_sessions.user_id
			AND u.status = 'ACTIVE' AND u.session_version = user_sessions.session_version
		)
		AND EXISTS (
			SELECT 1 FROM role_assignments ra WHERE ra.id = ? AND ra.user_id = user_sessions.user_id
			AND ra.status = 'ACTIVE' AND ra.valid_from <= ?
			AND (ra.valid_until IS NULL OR ra.valid_until > ?)
		)`,
		target.ID, tokenHash(newToken), formatTime(now), formatTime(newAbsolute), formatTime(now),
		session.ID, tokenHash(token), target.ID, formatTime(now), formatTime(now))
	if err != nil {
		return nil, fmt.Errorf("mengganti konteks sesi: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return nil, ErrInvalidSession
	}
	session.ActiveRoleAssignmentID = target.ID
	session.LastSeenAt = now
	session.AbsoluteExpiresAt = newAbsolute
	return &SessionCredential{Token: newToken, Session: session}, nil
}

func (s *Service) RevokeSession(ctx context.Context, token, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrInvalidInput
	}
	now := s.clock().UTC()
	result, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked_at = ?, revocation_reason = ?, updated_at = ?
		WHERE token_hash = ? AND revoked_at IS NULL`, formatTime(now), strings.TrimSpace(reason), formatTime(now), tokenHash(token))
	if err != nil {
		return fmt.Errorf("mencabut sesi: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil || affected != 1 {
		return ErrInvalidSession
	}
	return nil
}

// RequireOfferingMutation verifies both the active principal scope and the
// current academic ownership. Archived semesters remain readable but reject writes.
func (s *Service) RequireOfferingMutation(ctx context.Context, principal Principal, scope Scope) error {
	if err := principal.RequireOffering(scope); err != nil {
		return err
	}
	var allowed bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(
		SELECT 1 FROM course_offerings co
		JOIN semesters sem ON sem.id = co.semester_id
		WHERE co.id = ? AND co.semester_id = ? AND sem.class_id = ?
		AND sem.status <> 'ARCHIVED'
	)`, scope.CourseOfferingID, scope.SemesterID, scope.ClassID).Scan(&allowed)
	if err != nil {
		return fmt.Errorf("memeriksa cakupan mutasi offering: %w", err)
	}
	if !allowed {
		return ErrAccessDenied
	}
	return nil
}

func (s *Service) revokeSessionByID(ctx context.Context, sessionID int64, reason string, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE user_sessions SET revoked_at = ?, revocation_reason = ?, updated_at = ?
		WHERE id = ? AND revoked_at IS NULL`, formatTime(now), reason, formatTime(now), sessionID)
	return err
}

func (s *Service) loadActiveContext(ctx context.Context, userID, assignmentID int64, now time.Time) (*User, *RoleAssignment, error) {
	var user User
	err := s.db.QueryRowContext(ctx, `SELECT id, identity_key, display_name, status, session_version
		FROM users WHERE id = ?`, userID).Scan(&user.ID, &user.IdentityKey, &user.DisplayName, &user.Status, &user.SessionVersion)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && user.Status != "ACTIVE") {
		return nil, nil, ErrAccessDenied
	}
	if err != nil {
		return nil, nil, fmt.Errorf("membaca pengguna sesi: %w", err)
	}
	assignment, err := s.getAssignment(ctx, assignmentID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil, ErrAccessDenied
	}
	if err != nil {
		return nil, nil, fmt.Errorf("membaca assignment sesi: %w", err)
	}
	if assignment.UserID != userID || !assignmentEffective(*assignment, now) {
		return nil, nil, ErrAccessDenied
	}
	if err := s.validateAssignmentScope(ctx, *assignment); err != nil {
		if errors.Is(err, ErrAccessDenied) {
			return nil, nil, ErrAccessDenied
		}
		return nil, nil, err
	}
	return &user, assignment, nil
}

func (s *Service) getAssignment(ctx context.Context, assignmentID int64) (*RoleAssignment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, user_id, role, scope_type, class_id,
		semester_id, course_offering_id, status, valid_from, valid_until, version
		FROM role_assignments WHERE id = ?`, assignmentID)
	assignment, err := scanAssignment(row)
	return &assignment, err
}

func (s *Service) validateAssignmentScope(ctx context.Context, assignment RoleAssignment) error {
	switch assignment.Role {
	case RoleSystemAdmin:
		if assignment.ScopeType == ScopeGlobal && assignment.ClassID == nil && assignment.SemesterID == nil && assignment.CourseOfferingID == nil {
			return nil
		}
	case RoleKM:
		if assignment.ScopeType == ScopeClass && assignment.ClassID != nil && assignment.SemesterID == nil && assignment.CourseOfferingID == nil {
			return nil
		}
	case RolePJ:
		if assignment.ScopeType == ScopeCourseOffering && assignment.ClassID != nil && assignment.SemesterID != nil && assignment.CourseOfferingID != nil {
			var valid bool
			err := s.db.QueryRowContext(ctx, `SELECT EXISTS(
				SELECT 1 FROM course_offerings co
				JOIN semesters sem ON sem.id = co.semester_id
				WHERE co.id = ? AND co.semester_id = ? AND sem.class_id = ?
			)`, *assignment.CourseOfferingID, *assignment.SemesterID, *assignment.ClassID).Scan(&valid)
			if err != nil {
				return fmt.Errorf("memvalidasi cakupan PJ: %w", err)
			}
			if valid {
				return nil
			}
		}
	}
	return ErrAccessDenied
}

func (s *Service) loadSessionContext(ctx context.Context, hash string) (Session, User, RoleAssignment, *time.Time, error) {
	var session Session
	var user User
	var assignment RoleAssignment
	var classID, semesterID, offeringID sql.NullInt64
	var lastSeen, absolute, created, validFrom string
	var validUntil, revokedAt sql.NullString
	var storedVersion int
	err := s.db.QueryRowContext(ctx, `SELECT
		s.id, s.user_id, s.active_role_assignment_id, s.session_version,
		s.last_seen_at, s.absolute_expires_at, s.created_at, s.revoked_at,
		u.identity_key, u.display_name, u.status, u.session_version,
		ra.id, ra.user_id, ra.role, ra.scope_type, ra.class_id, ra.semester_id,
		ra.course_offering_id, ra.status, ra.valid_from, ra.valid_until, ra.version
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		JOIN role_assignments ra ON ra.id = s.active_role_assignment_id
		WHERE s.token_hash = ?`, hash).Scan(
		&session.ID, &session.UserID, &session.ActiveRoleAssignmentID, &storedVersion,
		&lastSeen, &absolute, &created, &revokedAt,
		&user.IdentityKey, &user.DisplayName, &user.Status, &user.SessionVersion,
		&assignment.ID, &assignment.UserID, &assignment.Role, &assignment.ScopeType, &classID, &semesterID,
		&offeringID, &assignment.Status, &validFrom, &validUntil, &assignment.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return session, user, assignment, nil, ErrInvalidSession
	}
	if err != nil {
		return session, user, assignment, nil, fmt.Errorf("membaca sesi: %w", err)
	}
	user.ID = session.UserID
	assignment.ClassID = nullInt64Ptr(classID)
	assignment.SemesterID = nullInt64Ptr(semesterID)
	assignment.CourseOfferingID = nullInt64Ptr(offeringID)
	if session.LastSeenAt, err = parseTime(lastSeen); err != nil {
		return session, user, assignment, nil, ErrInvalidSession
	}
	if session.AbsoluteExpiresAt, err = parseTime(absolute); err != nil {
		return session, user, assignment, nil, ErrInvalidSession
	}
	if session.CreatedAt, err = parseTime(created); err != nil {
		return session, user, assignment, nil, ErrInvalidSession
	}
	if assignment.ValidFrom, err = parseTime(validFrom); err != nil {
		return session, user, assignment, nil, ErrInvalidSession
	}
	if validUntil.Valid {
		parsed, parseErr := parseTime(validUntil.String)
		if parseErr != nil {
			return session, user, assignment, nil, ErrInvalidSession
		}
		assignment.ValidUntil = &parsed
	}
	var revoked *time.Time
	if revokedAt.Valid {
		parsed, parseErr := parseTime(revokedAt.String)
		if parseErr != nil {
			return session, user, assignment, nil, ErrInvalidSession
		}
		revoked = &parsed
	}
	session.sessionVersion = storedVersion
	return session, user, assignment, revoked, nil
}

func scanAssignment(row interface{ Scan(...any) error }) (RoleAssignment, error) {
	var assignment RoleAssignment
	var classID, semesterID, offeringID sql.NullInt64
	var validFrom string
	var validUntil sql.NullString
	err := row.Scan(&assignment.ID, &assignment.UserID, &assignment.Role, &assignment.ScopeType,
		&classID, &semesterID, &offeringID, &assignment.Status, &validFrom, &validUntil, &assignment.Version)
	if err != nil {
		return assignment, err
	}
	assignment.ClassID = nullInt64Ptr(classID)
	assignment.SemesterID = nullInt64Ptr(semesterID)
	assignment.CourseOfferingID = nullInt64Ptr(offeringID)
	assignment.ValidFrom, err = parseTime(validFrom)
	if err != nil {
		return assignment, err
	}
	if validUntil.Valid {
		parsed, parseErr := parseTime(validUntil.String)
		if parseErr != nil {
			return assignment, parseErr
		}
		assignment.ValidUntil = &parsed
	}
	return assignment, nil
}

func assignmentEffective(assignment RoleAssignment, now time.Time) bool {
	return assignment.Status == "ACTIVE" && !now.Before(assignment.ValidFrom) &&
		(assignment.ValidUntil == nil || now.Before(*assignment.ValidUntil))
}

func principalFrom(user User, assignment RoleAssignment, session Session) *Principal {
	return &Principal{
		UserID: user.ID, IdentityKey: user.IdentityKey, DisplayName: user.DisplayName,
		RoleAssignmentID: assignment.ID, RoleAssignmentVersion: assignment.Version,
		Role: assignment.Role, ScopeType: assignment.ScopeType, ClassID: assignment.ClassID,
		SemesterID: assignment.SemesterID, CourseOfferingID: assignment.CourseOfferingID,
		SessionID: session.ID, SessionVersion: user.SessionVersion,
	}
}

func sessionLimits(role string) (time.Duration, time.Duration) {
	if role == RoleSystemAdmin {
		return adminIdle, adminAbsolute
	}
	return normalIdle, normalAbsolute
}

func (s *Service) newToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := io.ReadFull(s.random, buffer); err != nil {
		return "", fmt.Errorf("membuat token sesi: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	copy := value.Int64
	return &copy
}
