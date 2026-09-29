package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bot-jadwal/internal/api/common"
	"bot-jadwal/internal/ratelimit"
)

// RateLimitManager mengelola rate limiting terpusat untuk endpoint API v1.
type RateLimitManager struct {
	limiter *ratelimit.Service
}

// NewRateLimitManager membuat instance baru RateLimitManager.
func NewRateLimitManager(limiter *ratelimit.Service) *RateLimitManager {
	return &RateLimitManager{limiter: limiter}
}

// SetLimiter memperbarui service limiter aktif.
func (rm *RateLimitManager) SetLimiter(limiter *ratelimit.Service) {
	rm.limiter = limiter
}

// Limiter mengembalikan underlying ratelimit.Service.
func (rm *RateLimitManager) Limiter() *ratelimit.Service {
	return rm.limiter
}

// ClientSource menurunkan source identity request: IP peer ternormalisasi
// atau client hop dari header proxy tepercaya (BE-012).
func ClientSource(r *http.Request, trustedProxyCIDRs []string) string {
	return ratelimit.ClientSourceHTTP(r.RemoteAddr, r.Header.Get, trustedProxyCIDRs)
}

// LimitExceeded menulis response 429 generik dengan header Retry-After.
func LimitExceeded(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(retryAfter.Seconds())
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", secs))
	common.WriteV1Error(w, http.StatusTooManyRequests, common.CodeTooManyRequests, "Terlalu banyak percobaan. Coba lagi nanti.")
}

// CheckSensitiveLimit menegakkan policy registry rate limit terpusat (BE-012)
// untuk endpoint sensitif. Mengembalikan false bila batas terlampaui (response 429/503 ditulis).
func (rm *RateLimitManager) CheckSensitiveLimit(w http.ResponseWriter, r *http.Request, trustedProxyCIDRs []string, policy ratelimit.PolicyKey, subject string) bool {
	if rm == nil || rm.limiter == nil {
		return true
	}
	source := ClientSource(r, trustedProxyCIDRs)
	res, err := rm.limiter.Check(r.Context(), policy, subject, source)
	if err != nil {
		common.WriteV1Error(w, http.StatusServiceUnavailable, common.CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
		return false
	}
	if !res.Allowed {
		LimitExceeded(w, res.RetryAfter)
		return false
	}
	return true
}

// RecordSensitiveLimit mencatat outcome ke tabel security_attempts / rate limiter.
func (rm *RateLimitManager) RecordSensitiveLimit(policy ratelimit.PolicyKey, subject, source, outcome string) {
	if rm == nil || rm.limiter == nil {
		return
	}
	if err := rm.limiter.Record(context.Background(), policy, subject, source, outcome); err != nil {
		fmt.Printf("[RateLimit] gagal mencatat %s: %v\n", string(policy), err)
	}
}

// PortalLimiterAdapter menjembatani ratelimit.Service ke antarmuka CodeLimiter milik portal.
type PortalLimiterAdapter struct {
	Svc *ratelimit.Service
}

func (a PortalLimiterAdapter) Check(ctx context.Context, subject, source string) (bool, time.Duration, error) {
	res, err := a.Svc.Check(ctx, ratelimit.PolicyPortalCode, subject, source)
	if err != nil {
		return false, 0, err
	}
	return res.Allowed, res.RetryAfter, nil
}

func (a PortalLimiterAdapter) Record(ctx context.Context, subject, source, outcome string) error {
	return a.Svc.Record(ctx, ratelimit.PolicyPortalCode, subject, source, outcome)
}
