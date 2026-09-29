package portal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("portal kelas tidak ditemukan")
	ErrInvalidCode  = errors.New("kode kelas tidak valid")
	ErrRateLimited  = errors.New("terlalu banyak percobaan, coba lagi nanti")
	ErrInvalidInput = errors.New("input tidak valid")
	ErrConflict     = errors.New("pengaturan portal berubah, muat ulang")
)

const (
	sessionTTL    = 30 * 24 * time.Hour
	attemptWindow = 15 * time.Minute
	maxAttempts   = 5
	blockPeriod   = 15 * time.Minute
)

type Service struct {
	db *sql.DB
	mu sync.Mutex
	// failures tracks recent failures per class+source for rate limiting.
	failures map[string][]time.Time
	blocks   map[string]time.Time
}

type Session struct {
	Token     string
	ExpiresAt time.Time
}

type RotationRequest struct {
	ClassID             int64
	Code                string
	ActorUserID         int64
	ActorRoleAssignment int64
}

type RotationResult struct {
	Code    string
	Version int
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, failures: map[string][]time.Time{}, blocks: map[string]time.Time{}}
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(code)))
	return hex.EncodeToString(sum[:])
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

func newAccessCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(100_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%08d", n.Int64()), nil
}

func limiterKey(classID int64, source string) string {
	return strings.TrimSpace(source) + "#" + itoa(classID)
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func (s *Service) checkRateLimit(classID int64, source string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := limiterKey(classID, source)
	if until, ok := s.blocks[key]; ok && now.Before(until) {
		return ErrRateLimited
	}
	cutoff := now.Add(-attemptWindow)
	kept := s.failures[key][:0]
	for _, t := range s.failures[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	s.failures[key] = kept
	if len(kept) >= maxAttempts {
		s.blocks[key] = now.Add(blockPeriod)
		return ErrRateLimited
	}
	return nil
}

func (s *Service) recordFailure(classID int64, source string, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := limiterKey(classID, source)
	s.failures[key] = append(s.failures[key], now)
}

func (s *Service) clearFailures(classID int64, source string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := limiterKey(classID, source)
	delete(s.failures, key)
	delete(s.blocks, key)
}

// VerifyCode checks the class code and creates a portal session on success.
// Returns raw session token for cookie/header use.
func (s *Service) VerifyCode(ctx context.Context, classID int64, code, source string) (Session, error) {
	if strings.TrimSpace(code) == "" {
		return Session{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	if err := s.checkRateLimit(classID, source, now); err != nil {
		return Session{}, err
	}
	var mode, codeHash sql.NullString
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT portal_access_mode, portal_code_hash, portal_code_version
		FROM class_settings WHERE class_id = ?`, classID).Scan(&mode, &codeHash, &version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrNotFound
		}
		return Session{}, err
	}
	if strings.ToUpper(strings.TrimSpace(mode.String)) != "CODE" || !codeHash.Valid {
		return Session{}, ErrNotFound
	}
	actualHash := hashCode(code)
	expectedHash := strings.TrimSpace(codeHash.String)
	if subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) != 1 {
		s.recordFailure(classID, source, now)
		return Session{}, ErrInvalidCode
	}
	s.clearFailures(classID, source)

	token, err := newToken()
	if err != nil {
		return Session{}, err
	}
	expires := now.Add(sessionTTL)
	_, err = s.db.ExecContext(ctx, `INSERT INTO portal_sessions (class_id, token_hash, access_code_version, expires_at)
		VALUES (?, ?, ?, ?)`, classID, hashToken(token), version, expires.Format(time.RFC3339Nano))
	if err != nil {
		return Session{}, err
	}
	return Session{Token: token, ExpiresAt: expires}, nil
}

// ValidateSession checks a portal token for a class (version-aware).
func (s *Service) ValidateSession(ctx context.Context, classID int64, token string) error {
	if strings.TrimSpace(token) == "" {
		return ErrInvalidCode
	}
	var version, currentVersion int
	var expires string
	var revoked sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT ps.access_code_version, ps.expires_at, ps.revoked_at,
		cs.portal_code_version FROM portal_sessions ps
		JOIN class_settings cs ON cs.class_id = ps.class_id
		WHERE ps.token_hash = ? AND ps.class_id = ?`, hashToken(token), classID).
		Scan(&version, &expires, &revoked, &currentVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidCode
	}
	if err != nil {
		return err
	}
	if revoked.Valid {
		return ErrInvalidCode
	}
	exp, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil {
		exp, err = time.Parse(time.RFC3339, expires)
		if err != nil {
			return ErrInvalidCode
		}
	}
	if !time.Now().UTC().Before(exp) {
		return ErrInvalidCode
	}
	if version != currentVersion {
		return ErrInvalidCode
	}
	return nil
}

func (s *Service) ResolveSession(ctx context.Context, token string) (int64, error) {
	if strings.TrimSpace(token) == "" {
		return 0, ErrInvalidCode
	}
	var classID int64
	var version, currentVersion int
	var expires, mode string
	var revoked sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT ps.class_id, ps.access_code_version, ps.expires_at,
		ps.revoked_at, cs.portal_code_version, cs.portal_access_mode
		FROM portal_sessions ps
		JOIN class_settings cs ON cs.class_id = ps.class_id
		WHERE ps.token_hash = ?`, hashToken(token)).
		Scan(&classID, &version, &expires, &revoked, &currentVersion, &mode)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrInvalidCode
	}
	if err != nil {
		return 0, err
	}
	if revoked.Valid || version != currentVersion || strings.ToUpper(strings.TrimSpace(mode)) != "CODE" {
		return 0, ErrInvalidCode
	}
	exp, err := time.Parse(time.RFC3339Nano, expires)
	if err != nil {
		exp, err = time.Parse(time.RFC3339, expires)
		if err != nil {
			return 0, ErrInvalidCode
		}
	}
	if !time.Now().UTC().Before(exp) {
		return 0, ErrInvalidCode
	}
	return classID, nil
}

func (s *Service) RotateCode(ctx context.Context, req RotationRequest) (RotationResult, error) {
	if req.ClassID <= 0 || req.ActorUserID <= 0 || req.ActorRoleAssignment <= 0 {
		return RotationResult{}, ErrInvalidInput
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		var err error
		code, err = newAccessCode()
		if err != nil {
			return RotationResult{}, err
		}
	}
	if len(code) < 6 || len(code) > 128 {
		return RotationResult{}, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RotationResult{}, err
	}
	defer tx.Rollback()

	var previousMode string
	var previousVersion int
	err = tx.QueryRowContext(ctx, `SELECT portal_access_mode, portal_code_version
		FROM class_settings WHERE class_id = ?`, req.ClassID).Scan(&previousMode, &previousVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return RotationResult{}, ErrNotFound
	}
	if err != nil {
		return RotationResult{}, err
	}

	newVersion := previousVersion + 1
	result, err := tx.ExecContext(ctx, `UPDATE class_settings SET
		portal_access_mode = 'CODE', portal_code_hash = ?, portal_code_version = ?,
		version = version + 1, updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE class_id = ? AND portal_code_version = ?`,
		hashCode(code), newVersion, req.ClassID, previousVersion)
	if err != nil {
		return RotationResult{}, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return RotationResult{}, err
	}
	if rowsAffected != 1 {
		return RotationResult{}, ErrConflict
	}

	if _, err := tx.ExecContext(ctx, `UPDATE portal_sessions
		SET revoked_at = strftime('%Y-%m-%dT%H:%M:%fZ','now')
		WHERE class_id = ? AND access_code_version < ? AND revoked_at IS NULL`, req.ClassID, newVersion); err != nil {
		return RotationResult{}, err
	}

	beforeJSON, err := json.Marshal(map[string]any{
		"portal_access_mode":  previousMode,
		"portal_code_version": previousVersion,
	})
	if err != nil {
		return RotationResult{}, err
	}
	afterJSON, err := json.Marshal(map[string]any{
		"portal_access_mode":  "CODE",
		"portal_code_version": newVersion,
	})
	if err != nil {
		return RotationResult{}, err
	}
	correlationID, err := newToken()
	if err != nil {
		return RotationResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO audit_logs (
		actor_type, actor_user_id, actor_role_assignment_id, class_id,
		action, entity_type, entity_id, before_json, after_json, correlation_id
	) VALUES ('USER', ?, ?, ?, 'ROTATE_PORTAL_CODE', 'CLASS_SETTINGS', ?, ?, ?, ?)`,
		req.ActorUserID, req.ActorRoleAssignment, req.ClassID, req.ClassID,
		string(beforeJSON), string(afterJSON), correlationID); err != nil {
		return RotationResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return RotationResult{}, err
	}
	return RotationResult{Code: code, Version: newVersion}, nil
}

// SetClassCode sets a new portal code (CODE mode) and bumps version, revoking old sessions logically.
func (s *Service) SetClassCode(ctx context.Context, classID int64, code string) error {
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 128 {
		return ErrInvalidInput
	}
	_, err := s.db.ExecContext(ctx, `UPDATE class_settings SET
		portal_access_mode = 'CODE', portal_code_hash = ?, portal_code_version = portal_code_version + 1, version = version + 1,
		updated_at = strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id = ?`, hashCode(code), classID)
	return err
}

// SetAccessMode switches LINK/CODE. Switching to LINK clears the code hash.
func (s *Service) SetAccessMode(ctx context.Context, classID int64, mode string) error {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	if mode != "LINK" && mode != "CODE" {
		return ErrInvalidInput
	}
	if mode == "LINK" {
		_, err := s.db.ExecContext(ctx, `UPDATE class_settings SET portal_access_mode='LINK',
			portal_code_hash=NULL, portal_code_version=portal_code_version+1, version = version + 1,
			updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id=?`, classID)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE class_settings SET portal_access_mode='CODE', version = version + 1,
		updated_at=strftime('%Y-%m-%dT%H:%M:%fZ','now') WHERE class_id=?`, classID)
	return err
}
