package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bot-jadwal/internal/ratelimit"
)

// SetRateLimiter memasang limiter terpusat pada server.
func (s *Server) SetRateLimiter(limiter *ratelimit.Service) {
	s.limiter = limiter
	if s.portalService != nil && limiter != nil {
		s.portalService.SetRateLimiter(portalLimiterAdapter{svc: limiter})
	}
}

// buildLimiter membangun limiter terpusat di atas database v1.
// Tanpa AuthHashKey valid, service memakai kunci efemeral (fingerprint tidak
// stabil lintas restart; production wajib key valid via Config.Validate).
func (s *Server) buildLimiter() {
	if s.v1DB == nil {
		return
	}
	if limiter, err := ratelimit.NewService(s.v1DB, s.authHashKey, nil); err == nil {
		s.SetRateLimiter(limiter)
	}
}

// portalLimiterAdapter menjembatani ratelimit.Service ke antarmuka
// CodeLimiter milik portal dengan policy kode portal.
type portalLimiterAdapter struct {
	svc *ratelimit.Service
}

func (a portalLimiterAdapter) Check(ctx context.Context, subject, source string) (bool, time.Duration, error) {
	res, err := a.svc.Check(ctx, ratelimit.PolicyPortalCode, subject, source)
	if err != nil {
		return false, 0, err
	}
	return res.Allowed, res.RetryAfter, nil
}

func (a portalLimiterAdapter) Record(ctx context.Context, subject, source, outcome string) error {
	return a.svc.Record(ctx, ratelimit.PolicyPortalCode, subject, source, outcome)
}

// clientSource menurunkan source identity request: IP peer ternormalisasi
// (tanpa port) atau client hop dari header proxy tepercaya (BE-012).
func (s *Server) clientSource(r *http.Request) string {
	return ratelimit.ClientSourceHTTP(r.RemoteAddr, r.Header.Get, s.trustedProxyCIDRs)
}

// limitExceeded menulis response 429 generik dengan Retry-After tanpa
// mengungkap counter internal.
func (s *Server) limitExceeded(w http.ResponseWriter, retryAfter time.Duration) {
	secs := int(retryAfter.Seconds())
	if secs < 1 {
		secs = 1
	}
	w.Header().Set("Retry-After", fmt.Sprintf("%d", secs))
	s.writeV1Error(w, http.StatusTooManyRequests, CodeTooManyRequests, "Terlalu banyak percobaan. Coba lagi nanti.")
}

// checkSensitiveLimit menegakkan policy registry rate limit terpusat (BE-012)
// untuk endpoint sensitif. False = response sudah ditulis (429/503).
// Nil limiter (mode tanpa DB) dilewati.
func (s *Server) checkSensitiveLimit(w http.ResponseWriter, r *http.Request, policy ratelimit.PolicyKey, subject string) bool {
	if s.limiter == nil {
		return true
	}
	res, err := s.limiter.Check(r.Context(), policy, subject, s.clientSource(r))
	if err != nil {
		s.writeV1Error(w, http.StatusServiceUnavailable, CodeServiceDown, "Layanan tidak tersedia. Coba lagi nanti.")
		return false
	}
	if !res.Allowed {
		s.limitExceeded(w, res.RetryAfter)
		return false
	}
	return true
}

// recordSensitiveLimit mencatat outcome limiter. Dipanggil setelah operasi
// domain commit; error operasional dicatat tanpa subject/source mentah.
func (s *Server) recordSensitiveLimit(policy ratelimit.PolicyKey, subject, source, outcome string) {
	if s.limiter == nil {
		return
	}
	if err := s.limiter.Record(context.Background(), policy, subject, source, outcome); err != nil {
		fmt.Printf("[RateLimit] gagal mencatat %s: %v\n", string(policy), err)
	}
}
