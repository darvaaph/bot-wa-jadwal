package ratelimit

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"time"
)

// PolicyKey mengidentifikasi kebijakan limiter pada registry terpusat.
type PolicyKey string

const (
	PolicyAuthLogin       PolicyKey = "AUTH_LOGIN"
	PolicyRecoveryRequest PolicyKey = "AUTH_RECOVERY_REQUEST"
	PolicyRecoveryConfirm PolicyKey = "AUTH_RECOVERY_CONFIRM"
	PolicyPortalCode      PolicyKey = "PORTAL_CODE_EXCHANGE"
	PolicyInviteAccept    PolicyKey = "INVITE_ACCEPT"
	PolicyBackupRestore   PolicyKey = "BACKUP_RESTORE"
	PolicyAdminMutation   PolicyKey = "ADMIN_MUTATION"
	PolicyPortalRotate    PolicyKey = "PORTAL_CODE_ROTATE"
)

// Policy menetapkan threshold limiter. Nilai production dibakukan pada
// docs/adr/0009-rate-limit-policies.md; jangan menyisipkan angka arbitrer
// langsung ke handler.
type Policy struct {
	MaxFailures int
	Window      time.Duration
	BlockPeriod time.Duration
}

// DefaultPolicies mengembalikan registry bawaan sesuai ADR-0009.
func DefaultPolicies() map[PolicyKey]Policy {
	return map[PolicyKey]Policy{
		PolicyAuthLogin:       {MaxFailures: 5, Window: 15 * time.Minute, BlockPeriod: 15 * time.Minute},
		PolicyRecoveryRequest: {MaxFailures: 5, Window: 15 * time.Minute, BlockPeriod: 30 * time.Minute},
		PolicyRecoveryConfirm: {MaxFailures: 10, Window: 15 * time.Minute, BlockPeriod: 30 * time.Minute},
		PolicyPortalCode:      {MaxFailures: 5, Window: 15 * time.Minute, BlockPeriod: 15 * time.Minute},
		PolicyInviteAccept:    {MaxFailures: 10, Window: 15 * time.Minute, BlockPeriod: 30 * time.Minute},
		PolicyBackupRestore:   {MaxFailures: 10, Window: 60 * time.Minute, BlockPeriod: 30 * time.Minute},
		PolicyAdminMutation:   {MaxFailures: 10, Window: 15 * time.Minute, BlockPeriod: 15 * time.Minute},
		PolicyPortalRotate:    {MaxFailures: 5, Window: 15 * time.Minute, BlockPeriod: 15 * time.Minute},
	}
}

// Service adalah rate limiter terpusat dan persisten. Seluruh instance yang
// berbagi database melihat state yang sama; tidak ada map in-memory sebagai
// sumber kebenaran. Hanya hash HMAC yang disimpan, tidak pernah nilai mentah.
type Service struct {
	db       *sql.DB
	hashKey  []byte
	policies map[PolicyKey]Policy
	clock    func() time.Time
}

// NewService membangun limiter. Key < 32 byte diganti kunci efemeral acak
// (fingerprint tidak stabil lintas restart; production wajib memakai key
// valid yang divalidasi Config.Validate).
func NewService(db *sql.DB, hashKey []byte, policies map[PolicyKey]Policy) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("database wajib diisi")
	}
	if len(hashKey) < 32 {
		ephemeral := make([]byte, 32)
		if _, err := rand.Read(ephemeral); err != nil {
			return nil, fmt.Errorf("membuat kunci limiter efemeral: %w", err)
		}
		hashKey = ephemeral
	}
	if policies == nil {
		policies = DefaultPolicies()
	}
	return &Service{db: db, hashKey: hashKey, policies: policies, clock: time.Now}, nil
}

// Fingerprint menghitung HMAC-SHA256 namespaced untuk subject dan source.
func (s *Service) Fingerprint(namespace, value string) string {
	mac := hmac.New(sha256.New, s.hashKey)
	_, _ = mac.Write([]byte(namespace))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

// CheckResult adalah hasil pemeriksaan limit.
type CheckResult struct {
	Allowed    bool
	RetryAfter time.Duration
}

// Check memeriksa apakah subject+source masih boleh mencoba. Operasi memakai
// transaksi IMMEDIATE agar request concurrent tidak melewati threshold.
// Error storage tidak pernah diabaikan: caller wajib gagal tertutup.
func (s *Service) Check(ctx context.Context, policy PolicyKey, subject, source string) (CheckResult, error) {
	pol, ok := s.policies[policy]
	if !ok {
		return CheckResult{}, fmt.Errorf("policy rate limit tidak dikenal")
	}
	now := s.clock().UTC()
	subHash := s.Fingerprint("subject:"+string(policy), subject)
	srcHash := s.Fingerprint("source", source)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return CheckResult{}, err
	}
	defer tx.Rollback()
	// Kunci tulis segera agar hitungan concurrent konsisten (SQLite).
	if _, err := tx.ExecContext(ctx, `SELECT COALESCE(MAX(id),0) FROM security_attempts`); err != nil {
		return CheckResult{}, err
	}
	var failures int
	windowStart := now.Add(-pol.Window).UTC().Format(time.RFC3339Nano)
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM security_attempts
		WHERE policy_key = ? AND subject_hash = ? AND source_hash = ? AND outcome = 'FAILURE'
		AND attempted_at >= ?
		AND id > COALESCE((
			SELECT MAX(id) FROM security_attempts
			WHERE policy_key = ? AND subject_hash = ? AND source_hash = ? AND outcome = 'SUCCESS'
		), 0)`, string(policy), subHash, srcHash, windowStart, string(policy), subHash, srcHash).Scan(&failures)
	if err != nil {
		return CheckResult{}, err
	}
	if failures < pol.MaxFailures {
		return CheckResult{Allowed: true}, nil
	}
	var lastFailure string
	err = tx.QueryRowContext(ctx, `SELECT attempted_at FROM security_attempts
		WHERE policy_key = ? AND subject_hash = ? AND source_hash = ? AND outcome = 'FAILURE'
		ORDER BY attempted_at DESC, id DESC LIMIT 1`, string(policy), subHash, srcHash).Scan(&lastFailure)
	if err != nil {
		return CheckResult{}, err
	}
	last, err := time.Parse(time.RFC3339Nano, lastFailure)
	if err != nil {
		return CheckResult{}, err
	}
	until := last.Add(pol.BlockPeriod)
	if now.Before(until) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO security_attempts
			(policy_key, subject_hash, source_hash, outcome, attempted_at, blocked_until)
			VALUES (?, ?, ?, 'BLOCKED', ?, ?)`,
			string(policy), subHash, srcHash, now.Format(time.RFC3339Nano), until.Format(time.RFC3339Nano)); err != nil {
			return CheckResult{}, err
		}
		if err := tx.Commit(); err != nil {
			return CheckResult{}, err
		}
		return CheckResult{Allowed: false, RetryAfter: until.Sub(now)}, nil
	}
	return CheckResult{Allowed: true}, nil
}

// Record mencatat outcome percobaan. Error tidak boleh diabaikan caller.
func (s *Service) Record(ctx context.Context, policy PolicyKey, subject, source, outcome string) error {
	if _, ok := s.policies[policy]; !ok {
		return fmt.Errorf("policy rate limit tidak dikenal")
	}
	switch outcome {
	case "SUCCESS", "FAILURE", "BLOCKED":
	default:
		return fmt.Errorf("outcome tidak valid")
	}
	now := s.clock().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO security_attempts
		(policy_key, subject_hash, source_hash, outcome, attempted_at)
		VALUES (?, ?, ?, ?, ?)`,
		string(policy), s.Fingerprint("subject:"+string(policy), subject),
		s.Fingerprint("source", source), outcome, now.Format(time.RFC3339Nano))
	return err
}

// Purge menghapus event lama di luar retention. Retention didokumentasikan
// pada ADR-0009 (30 hari).
func (s *Service) Purge(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := s.clock().UTC().Add(-olderThan).Format(time.RFC3339Nano)
	res, err := s.db.ExecContext(ctx, `DELETE FROM security_attempts WHERE attempted_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PeerIP menormalisasi alamat peer: port dihapus, IPv4/IPv6 dikanonisasi.
// Port client sementara tidak boleh mengubah source identity.
func PeerIP(remoteAddr string) string {
	host := strings.TrimSpace(remoteAddr)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if ip := net.ParseIP(host); ip != nil {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
		return ip.To16().String()
	}
	return strings.ToLower(host)
}

// ClientSource menurunkan source identity request. Header proxy hanya
// dipakai bila peer langsung termasuk TRUSTED_PROXY_CIDRS; header dari peer
// yang tidak tepercaya diabaikan agar tidak dapat dipalsukan.
func ClientSource(remoteAddr string, header map[string]string, trustedCIDRs []string) string {
	peer := PeerIP(remoteAddr)
	if !PeerTrusted(peer, trustedCIDRs) {
		return peer
	}
	if fwd, ok := header["Forwarded"]; ok && strings.TrimSpace(fwd) != "" {
		if client := parseForwardedFor(fwd); client != "" {
			return client
		}
	}
	if xff, ok := header["X-Forwarded-For"]; ok && strings.TrimSpace(xff) != "" {
		// Elemen pertama adalah client asal menurut konvensi rantai proxy.
		first := strings.TrimSpace(strings.Split(xff, ",")[0])
		first = strings.Trim(first, "[]")
		if h, _, err := net.SplitHostPort(first); err == nil {
			first = h
		}
		if ip := net.ParseIP(strings.Trim(first, "[]")); ip != nil {
			if v4 := ip.To4(); v4 != nil {
				return v4.String()
			}
			return ip.To16().String()
		}
	}
	return peer
}

// ClientSourceHTTP adalah adaptor net/http untuk ClientSource.
func ClientSourceHTTP(remoteAddr string, get func(string) string, trustedCIDRs []string) string {
	return ClientSource(remoteAddr, map[string]string{
		"Forwarded":       get("Forwarded"),
		"X-Forwarded-For": get("X-Forwarded-For"),
		"X-FORWARDED-FOR": get("X-Forwarded-For"),
	}, trustedCIDRs)
}

// PeerTrusted memeriksa apakah IP peer termasuk salah satu CIDR tepercaya.
func PeerTrusted(peer string, cidrs []string) bool {
	ip := net.ParseIP(peer)
	if ip == nil || len(cidrs) == 0 {
		return false
	}
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// parseForwardedFor mengambil IP client pertama dari header Forwarded (for=...).
func parseForwardedFor(header string) string {
	for _, part := range strings.Split(header, ",") {
		for _, kv := range strings.Split(part, ";") {
			kv = strings.TrimSpace(kv)
			if len(kv) > 4 && strings.EqualFold(kv[:4], "for=") {
				val := strings.Trim(kv[4:], `"[] `)
				if h, _, err := net.SplitHostPort(val); err == nil {
					val = h
				}
				val = strings.Trim(val, "[]")
				if ip := net.ParseIP(val); ip != nil {
					if v4 := ip.To4(); v4 != nil {
						return v4.String()
					}
					return ip.To16().String()
				}
			}
		}
	}
	return ""
}
