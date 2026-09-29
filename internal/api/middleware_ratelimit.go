package api

import (
	"net/http"
	"time"

	"bot-jadwal/internal/api/middleware"
	"bot-jadwal/internal/ratelimit"
)

// SetRateLimiter memasang limiter terpusat pada server.
func (s *Server) SetRateLimiter(limiter *ratelimit.Service) {
	s.limiter = limiter
	if s.rlManager != nil {
		s.rlManager.SetLimiter(limiter)
	}
	if s.portalService != nil && limiter != nil {
		s.portalService.SetRateLimiter(middleware.PortalLimiterAdapter{Svc: limiter})
	}
}

// buildLimiter membangun limiter terpusat di atas database v1.
func (s *Server) buildLimiter() {
	if s.v1DB == nil {
		return
	}
	if limiter, err := ratelimit.NewService(s.v1DB, s.authHashKey, nil); err == nil {
		s.SetRateLimiter(limiter)
	}
}

// clientSource menurunkan source identity request.
func (s *Server) clientSource(r *http.Request) string {
	return middleware.ClientSource(r, s.trustedProxyCIDRs)
}

// limitExceeded menulis response 429 generik dengan Retry-After.
func (s *Server) limitExceeded(w http.ResponseWriter, retryAfter time.Duration) {
	middleware.LimitExceeded(w, retryAfter)
}

// checkSensitiveLimit menegakkan policy registry rate limit terpusat (BE-012).
func (s *Server) checkSensitiveLimit(w http.ResponseWriter, r *http.Request, policy ratelimit.PolicyKey, subject string) bool {
	if s.rlManager != nil {
		return s.rlManager.CheckSensitiveLimit(w, r, s.trustedProxyCIDRs, policy, subject)
	}
	return true
}

// recordSensitiveLimit mencatat outcome limiter.
func (s *Server) recordSensitiveLimit(policy ratelimit.PolicyKey, subject, source, outcome string) {
	if s.rlManager != nil {
		s.rlManager.RecordSensitiveLimit(policy, subject, source, outcome)
	}
}
