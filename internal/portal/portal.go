package portal

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound     = errors.New("portal kelas tidak ditemukan")
	ErrInvalidCode  = errors.New("kode kelas tidak valid")
	ErrRateLimited  = errors.New("terlalu banyak percobaan, coba lagi nanti")
	ErrInvalidInput = errors.New("input tidak valid")
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
func (s *Service) VerifyCode(ctx context.Context, classID int64, code, source string) (string, error) {
	if strings.TrimSpace(code) == "" {
		return "", ErrInvalidInput
	}
	now := time.Now().UTC()
	if err := s.checkRateLimit(classID, source, now); err != nil {
		return "", err
	}
	var mode, codeHash sql.NullString
	var version int
	err := s.db.QueryRowContext(ctx, `SELECT portal_access_mode, portal_code_hash, portal_code_version
		FROM class_settings WHERE class_id = ?`, classID).Scan(&mode, &codeHash, &version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	if strings.ToUpper(strings.TrimSpace(mode.String)) != "CODE" || !codeHash.Valid {
		return "", ErrNotFound
	}
	if hashCode(code) != strings.TrimSpace(codeHash.String) {
		s.recordFailure(classID, source, now)
		return "", ErrInvalidCode
	}
	s.clearFailures(classID, source)

	token, err := newToken()
	if err != nil {
		return "", err
	}
	expires := now.Add(sessionTTL)
	_, err = s.db.ExecContext(ctx, `INSERT INTO portal_sessions (class_id, token_hash, access_code_version, expires_at)
		VALUES (?, ?, ?, ?)`, classID, hashToken(token), version, expires.Format(time.RFC3339Nano))
	if err != nil {
		return "", err
	}
	return token, nil
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
